package mail

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// pop3Client is a small RFC 1939 session (mailclientd only).
// Enough for inbox retrieve (USER/PASS, UIDL, RETR). Gaps: no TOP-only
// preview, no server-side folders/flags, leave-on-server (no DELE).
type pop3Client struct {
	cfg     ServerConfig
	user    string
	timeout time.Duration
	conn    net.Conn
	r       *bufio.Reader
}

type popUIDL struct {
	N    int
	UIDL string
}

func newPOP3Client(cfg ServerConfig, address string) *pop3Client {
	return &pop3Client{
		cfg:     cfg,
		user:    cfg.Username(address),
		timeout: 20 * time.Second,
	}
}

func (c *pop3Client) connect() error {
	host := strings.TrimSpace(c.cfg.Host)
	if host == "" {
		return fmt.Errorf("pop3: empty host")
	}
	if !strings.Contains(host, ":") {
		if c.cfg.implicitTLS(true) {
			host += ":995"
		} else {
			host += ":110"
		}
	}
	to := c.timeout
	if to <= 0 {
		to = 20 * time.Second
	}
	dialer := net.Dialer{Timeout: to}
	var conn net.Conn
	var err error
	if c.cfg.implicitTLS(true) && !c.cfg.useStartTLS() {
		conn, err = tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{
			MinVersion: tls.VersionTLS12, ServerName: serverName(host),
		})
	} else {
		conn, err = dialer.Dial("tcp", host)
	}
	if err != nil {
		return fmt.Errorf("pop3: connect %s: %w", host, err)
	}
	c.conn = conn
	c.r = bufio.NewReaderSize(conn, 256*1024)
	_ = conn.SetDeadline(time.Now().Add(to))
	greet, err := c.readLine()
	if err != nil {
		c.close()
		return fmt.Errorf("pop3: greeting: %w", err)
	}
	if !strings.HasPrefix(greet, "+OK") {
		c.close()
		return fmt.Errorf("pop3: greeting %s", greet)
	}
	if c.cfg.useStartTLS() && !c.cfg.implicitTLS(false) {
		if _, err := c.cmd("STLS"); err != nil {
			c.close()
			return fmt.Errorf("pop3: STLS: %w", err)
		}
		tlsConn := tls.Client(c.conn, &tls.Config{
			MinVersion: tls.VersionTLS12, ServerName: serverName(host),
		})
		if err := tlsConn.Handshake(); err != nil {
			c.close()
			return fmt.Errorf("pop3: TLS handshake: %w", err)
		}
		c.conn = tlsConn
		c.r = bufio.NewReaderSize(tlsConn, 256*1024)
		_ = c.conn.SetDeadline(time.Now().Add(to))
	}
	if err := c.login(); err != nil {
		c.close()
		return err
	}
	return nil
}

func (c *pop3Client) login() error {
	pass := c.cfg.Password()
	if pass == "" && c.cfg.Pass == "" && c.cfg.PassEnv == "" {
		// Greeting-only probe (Test connection before a password is typed).
		return nil
	}
	if pass == "" {
		return fmt.Errorf("pop3: empty password (set pop.password in mail.json, or %s / passEnv)", EnvPass)
	}
	if _, err := c.cmd("USER " + c.user); err != nil {
		return fmt.Errorf("pop3 USER: %w", err)
	}
	if _, err := c.cmd("PASS " + pass); err != nil {
		return fmt.Errorf("pop3 PASS: %w", err)
	}
	return nil
}

func (c *pop3Client) uidl() ([]popUIDL, error) {
	if _, err := c.cmd("UIDL"); err != nil {
		return nil, err
	}
	lines, err := c.readDot()
	if err != nil {
		return nil, err
	}
	var out []popUIDL
	for _, ln := range lines {
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		out = append(out, popUIDL{N: n, UIDL: fields[1]})
	}
	return out, nil
}

func (c *pop3Client) stat() (int, error) {
	ln, err := c.cmd("STAT")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(strings.TrimPrefix(ln, "+OK"))
	if len(fields) < 1 {
		return 0, fmt.Errorf("pop3 STAT: %s", ln)
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fmt.Errorf("pop3 STAT: %s", ln)
	}
	return n, nil
}

func (c *pop3Client) retr(n int) ([]byte, error) {
	if _, err := c.cmd(fmt.Sprintf("RETR %d", n)); err != nil {
		return nil, err
	}
	return c.readDotRaw()
}

func (c *pop3Client) cmd(s string) (string, error) {
	if err := writeCRLF(c.conn, s); err != nil {
		return "", err
	}
	ln, err := c.readLine()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(ln, "+OK") {
		return ln, nil
	}
	return ln, fmt.Errorf("%s", ln)
}

func (c *pop3Client) readLine() (string, error) {
	ln, err := c.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(ln, "\r\n"), nil
}

func (c *pop3Client) readDot() ([]string, error) {
	var lines []string
	for {
		ln, err := c.readLine()
		if err != nil {
			return nil, err
		}
		if ln == "." {
			return lines, nil
		}
		if strings.HasPrefix(ln, "..") {
			ln = ln[1:]
		}
		lines = append(lines, ln)
	}
}

func (c *pop3Client) readDotRaw() ([]byte, error) {
	var b strings.Builder
	first := true
	for {
		ln, err := c.readLine()
		if err != nil {
			if err == io.EOF && b.Len() > 0 {
				break
			}
			return nil, err
		}
		if ln == "." {
			break
		}
		if strings.HasPrefix(ln, "..") {
			ln = ln[1:]
		}
		if !first {
			b.WriteString("\r\n")
		}
		first = false
		b.WriteString(ln)
	}
	return []byte(b.String()), nil
}

func (c *pop3Client) close() {
	if c.conn == nil {
		return
	}
	_, _ = c.cmd("QUIT")
	_ = c.conn.Close()
	c.conn = nil
	c.r = nil
}

func popMessageID(accountID, uidl string) MessageID {
	safe := slug(uidl)
	safe = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '-'
		}
		return r
	}, safe)
	if safe == "" {
		safe = "msg"
	}
	return MessageID(accountID + "-pop-" + safe)
}
