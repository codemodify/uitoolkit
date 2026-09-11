package mail

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// imapClient is a production-minded IMAP4rev1 session (mailclientd only).
type imapClient struct {
	mu       sync.Mutex
	cfg      ServerConfig
	user     string
	conn     net.Conn
	r        *bufio.Reader
	tag      int
	caps     map[string]bool
	selected string
	uidval   uint32
	uidnext  uint32
	exists   int
	modseq   uint64
	idle         bool
	lastVanished []uint32
}

func newIMAPClient(cfg ServerConfig, address string) *imapClient {
	return &imapClient{
		cfg:  cfg,
		user: cfg.Username(address),
		caps: map[string]bool{},
	}
}

func (c *imapClient) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

func (c *imapClient) connectLocked() error {
	if c.conn != nil {
		return nil
	}
	host := c.cfg.Host
	if host == "" {
		return fmt.Errorf("imap: empty host")
	}
	if !strings.Contains(host, ":") {
		if c.cfg.implicitTLS(true) {
			host += ":993"
		} else {
			host += ":143"
		}
	}
	dialer := net.Dialer{Timeout: 20 * time.Second}
	var conn net.Conn
	var err error
	if c.cfg.implicitTLS(true) && !c.cfg.useStartTLS() {
		conn, err = tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName(host)})
	} else {
		conn, err = dialer.Dial("tcp", host)
	}
	if err != nil {
		return fmt.Errorf("imap: connect %s: %w", host, err)
	}
	c.conn = conn
	c.r = bufio.NewReaderSize(conn, 256*1024)
	if _, err := c.readRawLocked(); err != nil { // greeting
		c.closeLocked()
		return err
	}
	if err := c.capabilityLocked(); err != nil {
		c.closeLocked()
		return err
	}
	if c.cfg.useStartTLS() && !c.cfg.implicitTLS(false) {
		if _, err := c.cmdLocked("STARTTLS"); err != nil {
			c.closeLocked()
			return fmt.Errorf("imap: STARTTLS: %w", err)
		}
		tlsConn := tls.Client(c.conn, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName(host)})
		if err := tlsConn.Handshake(); err != nil {
			c.closeLocked()
			return fmt.Errorf("imap: TLS handshake: %w", err)
		}
		c.conn = tlsConn
		c.r = bufio.NewReaderSize(tlsConn, 256*1024)
		_ = c.capabilityLocked()
	}
	if err := c.loginLocked(); err != nil {
		c.closeLocked()
		return err
	}
	_ = c.capabilityLocked()
	if c.has("QRESYNC") || c.has("ENABLE") {
		_, _ = c.cmdLocked("ENABLE QRESYNC CONDSTORE")
		_ = c.capabilityLocked()
	}
	return nil
}

func serverName(hostport string) string {
	host, _, _ := strings.Cut(hostport, ":")
	return host
}

func (c *imapClient) loginLocked() error {
	pass := c.cfg.Password()
	auth := strings.ToLower(strings.TrimSpace(c.cfg.Auth))
	if auth == "" {
		auth = "plain"
	}
	if auth == "xoauth2" {
		token, err := resolveAccessToken(c.cfg, c.user)
		if err != nil {
			return err
		}
		raw := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.user, token)
		b64 := base64.StdEncoding.EncodeToString([]byte(raw))
		_, err = c.cmdLocked("AUTHENTICATE XOAUTH2 %s", b64)
		return err
	}
	if c.caps["AUTH=PLAIN"] || auth == "plain" {
		// LOGIN is widely deployed; AUTHENTICATE PLAIN when advertised.
		if c.caps["LOGINDISABLED"] && c.caps["AUTH=PLAIN"] {
			payload := base64.StdEncoding.EncodeToString([]byte("\x00" + c.user + "\x00" + pass))
			_, err := c.cmdLocked("AUTHENTICATE PLAIN %s", payload)
			return err
		}
	}
	if pass == "" {
		return fmt.Errorf("imap: empty password (set %s or passEnv)", EnvPass)
	}
	_, err := c.cmdLocked("LOGIN %s %s", imapQuote(c.user), imapQuote(pass))
	return err
}

func (c *imapClient) capabilityLocked() error {
	lines, err := c.cmdLocked("CAPABILITY")
	if err != nil {
		return err
	}
	c.caps = map[string]bool{}
	for _, ln := range lines {
		u := strings.ToUpper(ln)
		if !strings.Contains(u, "CAPABILITY") {
			continue
		}
		for _, w := range strings.Fields(u) {
			if w == "*" || w == "CAPABILITY" {
				continue
			}
			c.caps[w] = true
		}
	}
	return nil
}

func (c *imapClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

func (c *imapClient) closeLocked() {
	if c.conn != nil {
		_, _ = c.cmdLocked("LOGOUT")
		_ = c.conn.Close()
	}
	c.conn = nil
	c.r = nil
	c.selected = ""
	c.idle = false
}

func (c *imapClient) has(cap string) bool {
	return c.caps[strings.ToUpper(cap)]
}

type imapListBox struct {
	Name, Delim string
	Attrs       []string
}

func (c *imapClient) createMailbox(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return err
	}
	_, err := c.cmdLocked("CREATE %s", imapQuote(name))
	return err
}

func (c *imapClient) list() ([]imapListBox, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return nil, err
	}
	lines, err := c.cmdLocked("LIST %s %s", imapQuote(""), imapQuote("*"))
	if err != nil {
		return nil, err
	}
	var out []imapListBox
	for _, ln := range lines {
		if b, ok := parseIMAPList(ln); ok {
			out = append(out, b)
		}
	}
	if c.has("LSUB") {
		if extra, err := c.cmdLocked("LSUB %s %s", imapQuote(""), imapQuote("*")); err == nil {
			seen := map[string]bool{}
			for _, b := range out {
				seen[b.Name] = true
			}
			for _, ln := range extra {
				if b, ok := parseIMAPList(ln); ok && !seen[b.Name] {
					out = append(out, b)
				}
			}
		}
	}
	return out, nil
}

func parseIMAPList(ln string) (imapListBox, bool) {
	u := strings.ToUpper(ln)
	if !strings.Contains(u, " LIST ") && !strings.Contains(u, " LSUB ") && !strings.HasPrefix(u, "* LIST") && !strings.HasPrefix(u, "* LSUB") {
		return imapListBox{}, false
	}
	attrs, rest, ok := cutParenList(ln)
	if !ok {
		name, _ := parseListLine(ln)
		return imapListBox{Name: name}, name != ""
	}
	delim, rest := nextQuoted(strings.TrimSpace(rest))
	name, _ := nextQuoted(strings.TrimSpace(rest))
	if name == "" {
		name, _ = parseListLine(ln)
	}
	return imapListBox{Name: name, Delim: delim, Attrs: attrs}, name != ""
}

func cutParenList(s string) (attrs []string, rest string, ok bool) {
	i := strings.IndexByte(s, '(')
	if i < 0 {
		return nil, s, false
	}
	j := strings.IndexByte(s[i:], ')')
	if j < 0 {
		return nil, s, false
	}
	j += i
	inner := strings.TrimSpace(s[i+1 : j])
	if inner != "" {
		attrs = strings.Fields(inner)
	}
	return attrs, s[j+1:], true
}

func nextQuoted(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if s[0] == '"' {
		var b strings.Builder
		esc := false
		for i := 1; i < len(s); i++ {
			c := s[i]
			if esc {
				b.WriteByte(c)
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				return b.String(), s[i+1:]
			}
			b.WriteByte(c)
		}
		return b.String(), ""
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", ""
	}
	return strings.Trim(fields[0], `"`), strings.TrimPrefix(s, fields[0])
}

type imapSelect struct {
	Exists      int
	UIDValidity uint32
	UIDNext     uint32
	HighestMod  uint64
	Flags       []string
}

func (c *imapClient) selectBox(name string, examine bool) (imapSelect, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return imapSelect{}, err
	}
	cmd := "SELECT"
	if examine {
		cmd = "EXAMINE"
	}
	lines, err := c.cmdLocked("%s %s", cmd, imapQuote(name))
	if err != nil {
		return imapSelect{}, err
	}
	c.selected = name
	st := parseSelect(lines)
	c.exists = st.Exists
	c.uidval = st.UIDValidity
	c.uidnext = st.UIDNext
	c.modseq = st.HighestMod
	c.lastVanished = parseVanished(lines)
	return st, nil
}

// selectSync uses QRESYNC when the server advertised it and we have a prior modseq.
func (c *imapClient) selectSync(name string, examine bool, meta folderMeta) (imapSelect, []uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return imapSelect{}, nil, err
	}
	cmd := "SELECT"
	if examine {
		cmd = "EXAMINE"
	}
	var lines []string
	var err error
	if (c.has("QRESYNC") || c.has("ENABLE")) && meta.UIDValidity != 0 && meta.HighestMod != 0 {
		lines, err = c.cmdLocked("%s %s (QRESYNC (%d %d))", cmd, imapQuote(name), meta.UIDValidity, meta.HighestMod)
		if err != nil {
			lines, err = c.cmdLocked("%s %s", cmd, imapQuote(name))
		}
	} else {
		lines, err = c.cmdLocked("%s %s", cmd, imapQuote(name))
	}
	if err != nil {
		return imapSelect{}, nil, err
	}
	c.selected = name
	st := parseSelect(lines)
	c.exists = st.Exists
	c.uidval = st.UIDValidity
	c.uidnext = st.UIDNext
	c.modseq = st.HighestMod
	c.lastVanished = parseVanished(lines)
	return st, c.lastVanished, nil
}

func (c *imapClient) uidVanished(meta folderMeta) ([]uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]uint32(nil), c.lastVanished...), nil
}

func parseVanished(lines []string) []uint32 {
	var out []uint32
	for _, ln := range lines {
		u := strings.ToUpper(ln)
		i := strings.Index(u, "VANISHED")
		if i < 0 {
			continue
		}
		rest := strings.TrimSpace(ln[i+len("VANISHED"):])
		if strings.HasPrefix(strings.ToUpper(rest), "(EARLIER)") {
			rest = strings.TrimSpace(rest[len("(EARLIER)"):])
		}
		for _, part := range strings.Split(rest, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if a, b, ok := strings.Cut(part, ":"); ok {
				lo, hi := atoi(a), atoi(b)
				if hi-lo > 100000 {
					continue
				}
				for n := lo; n <= hi; n++ {
					out = append(out, uint32(n))
				}
				continue
			}
			if n := atoi(part); n > 0 {
				out = append(out, uint32(n))
			}
		}
	}
	return out
}

func parseSelect(lines []string) imapSelect {
	var st imapSelect
	for _, ln := range lines {
		u := strings.ToUpper(ln)
		fields := strings.Fields(ln)
		if len(fields) >= 3 && fields[0] == "*" {
			if n, err := strconv.Atoi(fields[1]); err == nil && strings.EqualFold(fields[2], "EXISTS") {
				st.Exists = n
			}
		}
		if i := strings.Index(u, "[UIDVALIDITY "); i >= 0 {
			st.UIDValidity = uint32(atoi(u[i+13:]))
		}
		if i := strings.Index(u, "[UIDNEXT "); i >= 0 {
			st.UIDNext = uint32(atoi(u[i+9:]))
		}
		if i := strings.Index(u, "[HIGHESTMODSEQ "); i >= 0 {
			st.HighestMod = uint64(atoi(u[i+15:]))
		}
	}
	return st
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

type imapMeta struct {
	UID          uint32
	Flags        []string
	Size         int
	From         string
	To           string
	Cc           string
	Subject      string
	Date         time.Time
	Parts        []Part
	RFCMessageID string
	InReplyTo    string
}

func (c *imapClient) uidFetchMeta(fromUID uint32) ([]imapMeta, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.selected == "" {
		return nil, fmt.Errorf("imap: no mailbox selected")
	}
	set := "1:*"
	if fromUID > 1 {
		set = fmt.Sprintf("%d:*", fromUID)
	}
	items := "(UID FLAGS RFC822.SIZE ENVELOPE BODYSTRUCTURE)"
	lines, err := c.cmdLocked("UID FETCH %s %s", set, items)
	if err != nil {
		return nil, err
	}
	return parseUIDFetchMeta(lines), nil
}

func (c *imapClient) uidFetchFlags(fromUID uint32, changedSince uint64) ([]imapMeta, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	set := "1:*"
	if fromUID > 1 {
		set = fmt.Sprintf("%d:*", fromUID)
	}
	cmd := fmt.Sprintf("UID FETCH %s (UID FLAGS)", set)
	if changedSince > 0 && c.has("CONDSTORE") {
		cmd = fmt.Sprintf("UID FETCH %s (UID FLAGS) (CHANGEDSINCE %d)", set, changedSince)
	}
	lines, err := c.cmdLocked("%s", cmd)
	if err != nil {
		return nil, err
	}
	return parseUIDFetchMeta(lines), nil
}

func (c *imapClient) uidFetchRFC822(uid uint32) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	lines, err := c.cmdLocked("UID FETCH %d (UID BODY.PEEK[])", uid)
	if err != nil {
		return nil, err
	}
	return extractLiteralBody(lines), nil
}

func (c *imapClient) uidFetchSection(uid uint32, section string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if section == "" {
		section = "1"
	}
	lines, err := c.cmdLocked("UID FETCH %d (BODY.PEEK[%s])", uid, section)
	if err != nil {
		return nil, err
	}
	return extractLiteralBody(lines), nil
}

func (c *imapClient) uidStore(uid uint32, add, rem []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(add) > 0 {
		if _, err := c.cmdLocked("UID STORE %d +FLAGS.SILENT (%s)", uid, strings.Join(add, " ")); err != nil {
			return err
		}
	}
	if len(rem) > 0 {
		if _, err := c.cmdLocked("UID STORE %d -FLAGS.SILENT (%s)", uid, strings.Join(rem, " ")); err != nil {
			return err
		}
	}
	return nil
}

func (c *imapClient) uidCopy(uid uint32, dest string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.cmdLocked("UID COPY %d %s", uid, imapQuote(dest))
	return err
}

func (c *imapClient) uidMove(uid uint32, dest string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.has("MOVE") {
		_, err := c.cmdLocked("UID MOVE %d %s", uid, imapQuote(dest))
		return err
	}
	if _, err := c.cmdLocked("UID COPY %d %s", uid, imapQuote(dest)); err != nil {
		return err
	}
	if _, err := c.cmdLocked("UID STORE %d +FLAGS.SILENT (\\Deleted)", uid); err != nil {
		return err
	}
	_, err := c.cmdLocked("EXPUNGE")
	return err
}

func (c *imapClient) expunge() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.cmdLocked("EXPUNGE")
	return err
}

func (c *imapClient) appendRaw(mbox string, raw []byte, flags string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return err
	}
	head := fmt.Sprintf("APPEND %s {%d}", imapQuote(mbox), len(raw))
	if flags != "" {
		head = fmt.Sprintf("APPEND %s (%s) {%d}", imapQuote(mbox), flags, len(raw))
	}
	_, err := c.cmdLiteralLocked(head, string(raw))
	return err
}

func (c *imapClient) search(args string) ([]uint32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	lines, err := c.cmdLocked("UID SEARCH %s", args)
	if err != nil {
		return nil, err
	}
	var out []uint32
	for _, ln := range lines {
		u := strings.ToUpper(ln)
		if !strings.Contains(u, " SEARCH") {
			continue
		}
		for _, w := range strings.Fields(ln) {
			if n, err := strconv.Atoi(w); err == nil && n > 0 {
				out = append(out, uint32(n))
			}
		}
	}
	return out, nil
}

func (c *imapClient) noop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return err
	}
	_, err := c.cmdLocked("NOOP")
	return err
}

// idleOnce sends IDLE, waits until wait or an untagged EXISTS/FETCH/EXPUNGE, then DONE.
func (c *imapClient) idleOnce(wait time.Duration) (woke bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.has("IDLE") {
		return false, fmt.Errorf("imap: no IDLE")
	}
	if err := c.connectLocked(); err != nil {
		return false, err
	}
	c.tag++
	tag := fmt.Sprintf("A%03d", c.tag)
	if _, err := io.WriteString(c.conn, tag+" IDLE\r\n"); err != nil {
		return false, err
	}
	c.idle = true
	deadline := time.Now().Add(wait)
	_ = c.conn.SetReadDeadline(deadline)
	defer func() {
		_ = c.conn.SetReadDeadline(time.Time{})
		c.idle = false
	}()
	for {
		ln, err := c.readRawLocked()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				_, _ = io.WriteString(c.conn, "DONE\r\n")
				_, _ = c.readUntilTaggedLocked(tag)
				return false, nil
			}
			return false, err
		}
		u := strings.ToUpper(ln)
		if strings.HasPrefix(strings.TrimSpace(ln), "+") {
			continue
		}
		if strings.Contains(u, " EXISTS") || strings.Contains(u, " EXPUNGE") || strings.Contains(u, " FETCH") || strings.Contains(u, " RECENT") {
			_, _ = io.WriteString(c.conn, "DONE\r\n")
			_, _ = c.readUntilTaggedLocked(tag)
			return true, nil
		}
		if strings.HasPrefix(ln, tag+" ") {
			return false, nil
		}
	}
}

func (c *imapClient) cmdLocked(format string, args ...any) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("imap: not connected")
	}
	c.tag++
	tag := fmt.Sprintf("A%03d", c.tag)
	line := tag + " " + fmt.Sprintf(format, args...) + "\r\n"
	if _, err := io.WriteString(c.conn, line); err != nil {
		return nil, err
	}
	return c.readUntilTaggedLocked(tag)
}

func (c *imapClient) cmdLiteralLocked(head, literal string) ([]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("imap: not connected")
	}
	c.tag++
	tag := fmt.Sprintf("A%03d", c.tag)
	if _, err := io.WriteString(c.conn, tag+" "+head+"\r\n"); err != nil {
		return nil, err
	}
	cont, err := c.readRawLocked()
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.TrimSpace(cont), "+") {
		if strings.HasPrefix(cont, tag+" ") {
			return nil, fmt.Errorf("imap: %s", strings.TrimSpace(cont[len(tag):]))
		}
		return nil, fmt.Errorf("imap: expected + continuation, got %s", strings.TrimSpace(cont))
	}
	if _, err := io.WriteString(c.conn, literal+"\r\n"); err != nil {
		return nil, err
	}
	return c.readUntilTaggedLocked(tag)
}

func (c *imapClient) readUntilTaggedLocked(tag string) ([]string, error) {
	var lines []string
	for {
		ln, err := c.readRawLocked()
		if err != nil {
			return lines, err
		}
		if strings.HasPrefix(ln, tag+" ") {
			rest := strings.TrimSpace(ln[len(tag)+1:])
			if strings.HasPrefix(rest, "OK") {
				return lines, nil
			}
			return lines, fmt.Errorf("imap: %s", rest)
		}
		lines = append(lines, ln)
	}
}

func (c *imapClient) readRawLocked() (string, error) {
	if c.r == nil {
		return "", fmt.Errorf("imap: not connected")
	}
	var buf strings.Builder
	for {
		ln, err := c.r.ReadString('\n')
		if err != nil {
			return buf.String(), err
		}
		buf.WriteString(ln)
		s := strings.TrimRight(ln, "\r\n")
		if n, ok := trailingLiteralSize(s); ok {
			lit := make([]byte, n)
			if _, err := io.ReadFull(c.r, lit); err != nil {
				return buf.String(), err
			}
			buf.Write(lit)
			continue
		}
		return buf.String(), nil
	}
}

func trailingLiteralSize(s string) (int, bool) {
	i := strings.LastIndexByte(s, '{')
	if i < 0 || !strings.HasSuffix(s, "}") {
		return 0, false
	}
	inner := s[i+1 : len(s)-1]
	if inner == "" {
		return 0, false
	}
	for _, r := range inner {
		if !unicode.IsDigit(r) {
			return 0, false
		}
	}
	n, err := strconv.Atoi(inner)
	if err != nil || n < 0 || n > 64*1024*1024 {
		return 0, false
	}
	return n, true
}

func extractLiteralBody(lines []string) []byte {
	joined := strings.Join(lines, "")
	i := strings.IndexByte(joined, '{')
	if i < 0 {
		// Some servers inline small bodies as quoted strings after BODY[]
		if q := extractQuotedAfter(joined, "BODY"); q != "" {
			return []byte(q)
		}
		return nil
	}
	j := strings.IndexByte(joined[i:], '}')
	if j < 0 {
		return nil
	}
	n, err := strconv.Atoi(joined[i+1 : i+j])
	if err != nil || n < 0 {
		return nil
	}
	start := i + j + 1
	if start < len(joined) && (joined[start] == '\r' || joined[start] == '\n') {
		if joined[start] == '\r' && start+1 < len(joined) && joined[start+1] == '\n' {
			start += 2
		} else {
			start++
		}
	}
	if start+n > len(joined) {
		n = len(joined) - start
	}
	if n < 0 {
		return nil
	}
	return []byte(joined[start : start+n])
}

func extractQuotedAfter(s, key string) string {
	u := strings.ToUpper(s)
	i := strings.Index(u, key)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	q := strings.IndexByte(rest, '"')
	if q < 0 {
		return ""
	}
	out, _ := nextQuoted(rest[q:])
	return out
}

func parseUIDFetchMeta(lines []string) []imapMeta {
	var out []imapMeta
	for _, ln := range lines {
		u := strings.ToUpper(ln)
		if !strings.Contains(u, " FETCH ") {
			continue
		}
		m := imapMeta{}
		if i := strings.Index(u, "UID "); i >= 0 {
			m.UID = uint32(atoi(ln[i+4:]))
		}
		if i := strings.Index(u, "RFC822.SIZE "); i >= 0 {
			m.Size = atoi(ln[i+12:])
		}
		if i := strings.Index(u, "FLAGS "); i >= 0 {
			if attrs, _, ok := cutParenList(ln[i+6:]); ok {
				m.Flags = attrs
			}
		}
		if i := strings.Index(u, "ENVELOPE "); i >= 0 {
			env, _, err := parseSexp(ln, i+len("ENVELOPE "))
			if err == nil {
				applyEnvelope(&m, env)
			}
		}
		if i := strings.Index(u, "BODYSTRUCTURE "); i >= 0 {
			bs, _, err := parseSexp(ln, i+len("BODYSTRUCTURE "))
			if err == nil {
				m.Parts = walkBodyStructure(bs, "")
			}
		}
		if m.UID != 0 || m.Subject != "" || len(m.Flags) > 0 {
			out = append(out, m)
		}
	}
	return out
}

type sexp struct {
	Nil  bool
	Num  int64
	Str  string
	List []sexp
	Atom bool
}

func parseSexp(s string, i int) (sexp, int, error) {
	for i < len(s) && (s[i] == ' ' || s[i] == '\r' || s[i] == '\n' || s[i] == '\t') {
		i++
	}
	if i >= len(s) {
		return sexp{Nil: true}, i, io.EOF
	}
	switch s[i] {
	case '(':
		var list []sexp
		i++
		for {
			for i < len(s) && (s[i] == ' ' || s[i] == '\r' || s[i] == '\n' || s[i] == '\t') {
				i++
			}
			if i >= len(s) {
				return sexp{List: list}, i, nil
			}
			if s[i] == ')' {
				return sexp{List: list}, i + 1, nil
			}
			el, ni, err := parseSexp(s, i)
			if err != nil {
				return sexp{List: list}, ni, err
			}
			list = append(list, el)
			i = ni
		}
	case '"':
		var b strings.Builder
		i++
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
			if s[i] == '"' {
				return sexp{Str: b.String()}, i + 1, nil
			}
			b.WriteByte(s[i])
			i++
		}
		return sexp{Str: b.String()}, i, nil
	case '{':
		j := strings.IndexByte(s[i:], '}')
		if j < 0 {
			return sexp{}, i, fmt.Errorf("imap: bad literal")
		}
		n := atoi(s[i+1 : i+j])
		i = i + j + 1
		if i < len(s) && s[i] == '\r' {
			i++
		}
		if i < len(s) && s[i] == '\n' {
			i++
		}
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		return sexp{Str: s[i:end]}, end, nil
	default:
		if strings.HasPrefix(strings.ToUpper(s[i:]), "NIL") && (i+3 == len(s) || !isAtomChar(s[i+3])) {
			return sexp{Nil: true}, i + 3, nil
		}
		j := i
		for j < len(s) && isAtomChar(s[j]) {
			j++
		}
		tok := s[i:j]
		if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
			return sexp{Num: n, Str: tok}, j, nil
		}
		return sexp{Str: tok, Atom: true}, j, nil
	}
}

func isAtomChar(c byte) bool {
	if c <= 32 || c >= 127 {
		return false
	}
	switch c {
	case '(', ')', '{', '}', '"', '\\', '%', '*':
		return false
	}
	return true
}

func applyEnvelope(m *imapMeta, env sexp) {
	// (date subject from sender reply-to to cc bcc in-reply-to message-id)
	if len(env.List) < 6 {
		return
	}
	m.Date = parseIMAPDate(env.List[0].Str)
	m.Subject = decodeRFC2047(env.List[1].Str)
	m.From = formatAddrList(env.List[2])
	m.To = formatAddrList(env.List[5])
	if len(env.List) > 6 {
		m.Cc = formatAddrList(env.List[6])
	}
	if len(env.List) > 8 {
		m.InReplyTo = env.List[8].Str
	}
	if len(env.List) > 9 {
		m.RFCMessageID = env.List[9].Str
	}
}

func formatAddrList(s sexp) string {
	if s.Nil || len(s.List) == 0 {
		return ""
	}
	var parts []string
	for _, a := range s.List {
		if len(a.List) < 4 {
			continue
		}
		name := decodeRFC2047(a.List[0].Str)
		mbox := a.List[2].Str
		host := a.List[3].Str
		addr := mbox
		if host != "" {
			addr = mbox + "@" + host
		}
		if name != "" {
			parts = append(parts, name+" <"+addr+">")
		} else {
			parts = append(parts, addr)
		}
	}
	return strings.Join(parts, ", ")
}

func parseIMAPDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC1123Z, time.RFC1123, "Mon, 2 Jan 2006 15:04:05 -0700",
		"02-Jan-2006 15:04:05 -0700", time.RFC822Z,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if t, err := mailParseDate(s); err == nil {
		return t
	}
	return time.Time{}
}

func mailParseDate(s string) (time.Time, error) {
	return mail.ParseDate(s)
}

func walkBodyStructure(s sexp, prefix string) []Part {
	if len(s.List) == 0 {
		return nil
	}
	// multipart: (part part ... "subtype")
	if len(s.List) > 0 && len(s.List[0].List) > 0 {
		var out []Part
		n := 0
		for _, el := range s.List {
			if len(el.List) == 0 {
				break
			}
			n++
			id := strconv.Itoa(n)
			if prefix != "" {
				id = prefix + "." + id
			}
			out = append(out, walkBodyStructure(el, id)...)
		}
		return out
	}
	// single: ("text" "plain" (k v...) NIL NIL enc size ...)
	if len(s.List) < 7 {
		return nil
	}
	typ := strings.ToLower(s.List[0].Str + "/" + s.List[1].Str)
	params := map[string]string{}
	for i := 0; i+1 < len(s.List[2].List); i += 2 {
		params[strings.ToLower(s.List[2].List[i].Str)] = s.List[2].List[i+1].Str
	}
	id := prefix
	if id == "" {
		id = "1"
	}
	p := Part{
		ID: id, MIMEType: typ, Size: int(s.List[6].Num),
		Charset: params["charset"], Filename: decodeRFC2047(params["name"]),
		Inline: !strings.EqualFold(typ, "application/octet-stream") && params["name"] == "",
	}
	return []Part{p}
}

func folderKindFromIMAP(name string, attrs []string) FolderKind {
	joined := strings.ToUpper(strings.Join(attrs, " ") + " " + name)
	switch {
	case strings.EqualFold(name, "INBOX"):
		return FolderInbox
	case strings.Contains(joined, "TRASH") || strings.Contains(joined, "\\TRASH"):
		return FolderTrash
	case strings.Contains(joined, "SENT") || strings.Contains(joined, "\\SENT"):
		return FolderSent
	case strings.Contains(joined, "DRAFT"):
		return FolderDrafts
	case strings.Contains(joined, "JUNK") || strings.Contains(joined, "SPAM"):
		return FolderJunk
	case strings.Contains(joined, "ARCHIVE"):
		return FolderArchive
	}
	return FolderCustom
}

func imapFlagSeen(flags []string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, `\Seen`) {
			return true
		}
	}
	return false
}

func imapFlagStar(flags []string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, `\Flagged`) {
			return true
		}
	}
	return false
}

func imapKeywords(flags []string) []string {
	var out []string
	for _, f := range flags {
		if strings.HasPrefix(f, `\`) || f == "" {
			continue
		}
		out = append(out, f)
	}
	return out
}

