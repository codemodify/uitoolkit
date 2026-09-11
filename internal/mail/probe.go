package mail

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// ProbeRequest is accounts.test / hosts.probe.
type ProbeRequest struct {
	Address   string `json:"address"`
	User      string `json:"user,omitempty"`
	Password  string `json:"password,omitempty"`
	Protocol  string `json:"protocol,omitempty"` // imap, pop3, or empty (try both)
	Host      string `json:"host,omitempty"`
	IMAP      string `json:"imap,omitempty"`
	POP       string `json:"pop,omitempty"`
	TimeoutMS int    `json:"timeoutMs,omitempty"`
	Auto      bool   `json:"auto,omitempty"` // try the other protocol if the preferred fails
}

// ProbeResult is what Test connection reports.
type ProbeResult struct {
	OK       bool   `json:"ok"`
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	TLSMode  string `json:"tlsMode,omitempty"` // ssl, starttls, plain
	Error    string `json:"error,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type probeCand struct {
	Protocol string
	Host     string
	Implicit bool
	StartTLS bool
}

func (c probeCand) tlsMode() string {
	if c.Implicit {
		return "ssl"
	}
	if c.StartTLS {
		return "starttls"
	}
	return "plain"
}

// ProbeAccount dials IMAP and/or POP3 with the typed user/password.
// Overall time is capped (default 8s) so the Add Account UI stays usable.
func ProbeAccount(req ProbeRequest) ProbeResult {
	user := strings.TrimSpace(req.User)
	if user == "" {
		user = strings.TrimSpace(req.Address)
	}
	pass := req.Password
	cands := probeCandidates(req)
	if len(cands) == 0 {
		return ProbeResult{Error: "no host to probe — enter an email or incoming server"}
	}
	overall := 8 * time.Second
	if req.TimeoutMS > 0 {
		overall = time.Duration(req.TimeoutMS) * time.Millisecond
	}
	deadline := time.Now().Add(overall)
	var last error
	tried := 0
	for _, c := range cands {
		if time.Now().After(deadline) {
			break
		}
		to := 2 * time.Second
		if rem := time.Until(deadline); rem < to {
			to = rem
		}
		if to < 200*time.Millisecond {
			break
		}
		tried++
		err := probeOne(c, user, pass, to)
		if err == nil {
			detail := c.Protocol + " " + c.Host + " " + c.tlsMode()
			if pass == "" {
				detail += " (greeting only — no password)"
			}
			return ProbeResult{
				OK: true, Protocol: c.Protocol, Host: c.Host, TLSMode: c.tlsMode(),
				Detail: detail,
			}
		}
		last = err
	}
	msg := "connection failed"
	if last != nil {
		msg = last.Error()
	}
	return ProbeResult{Error: msg, Detail: fmt.Sprintf("tried %d candidate(s)", tried)}
}

func probeCandidates(req ProbeRequest) []probeCand {
	prefer := strings.ToLower(strings.TrimSpace(req.Protocol))
	if prefer == "pop" {
		prefer = ProtoPOP3
	}
	auto := req.Auto || prefer == "" || prefer == "auto"
	if prefer == "auto" {
		prefer = ""
	}

	var out []probeCand
	host := strings.TrimSpace(req.Host)
	imapHost := strings.TrimSpace(req.IMAP)
	popHost := strings.TrimSpace(req.POP)
	if host != "" {
		if prefer == ProtoPOP3 {
			out = append(out, candsForHost(ProtoPOP3, host)...)
		} else if prefer == ProtoIMAP {
			out = append(out, candsForHost(ProtoIMAP, host)...)
		} else {
			out = append(out, candsForHost(ProtoIMAP, host)...)
			out = append(out, candsForHost(ProtoPOP3, host)...)
		}
		if !auto {
			return dedupeProbeCands(out)
		}
	}
	if imapHost != "" && (auto || prefer != ProtoPOP3) {
		out = append(out, candsForHost(ProtoIMAP, imapHost)...)
	}
	if popHost != "" && (auto || prefer == ProtoPOP3 || prefer == "") {
		out = append(out, candsForHost(ProtoPOP3, popHost)...)
	}

	domain := domainOfAddress(req.Address)
	if domain != "" {
		g := GuessMailHosts(req.Address)
		if auto || prefer != ProtoPOP3 {
			out = append(out, candsForHost(ProtoIMAP, g.IMAP)...)
			out = append(out, imapCommonHosts(domain)...)
		}
		if auto || prefer == ProtoPOP3 || prefer == "" {
			out = append(out, candsForHost(ProtoPOP3, g.POP)...)
			out = append(out, popCommonHosts(domain)...)
		}
	}
	out = dedupeProbeCands(out)
	if prefer == ProtoPOP3 || prefer == ProtoIMAP {
		out = sortProbeCands(out, prefer)
	}
	return out
}

func candsForHost(proto, host string) []probeCand {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}
	proto = NormalizeProtocol(proto)
	if !strings.Contains(host, ":") {
		if proto == ProtoPOP3 {
			return []probeCand{
				{Protocol: ProtoPOP3, Host: host + ":995", Implicit: true},
				{Protocol: ProtoPOP3, Host: host + ":110", StartTLS: true},
			}
		}
		return []probeCand{
			{Protocol: ProtoIMAP, Host: host + ":993", Implicit: true},
			{Protocol: ProtoIMAP, Host: host + ":143", StartTLS: true},
		}
	}
	impl, stls := tlsModeForHost(host)
	return []probeCand{{Protocol: proto, Host: host, Implicit: impl, StartTLS: stls}}
}

func imapCommonHosts(domain string) []probeCand {
	return []probeCand{
		{Protocol: ProtoIMAP, Host: "imap." + domain + ":993", Implicit: true},
		{Protocol: ProtoIMAP, Host: "mail." + domain + ":993", Implicit: true},
		{Protocol: ProtoIMAP, Host: "imap." + domain + ":143", StartTLS: true},
		{Protocol: ProtoIMAP, Host: "mail." + domain + ":143", StartTLS: true},
	}
}

func popCommonHosts(domain string) []probeCand {
	return []probeCand{
		{Protocol: ProtoPOP3, Host: "pop." + domain + ":995", Implicit: true},
		{Protocol: ProtoPOP3, Host: "mail." + domain + ":995", Implicit: true},
		{Protocol: ProtoPOP3, Host: "pop." + domain + ":110", StartTLS: true},
		{Protocol: ProtoPOP3, Host: "mail." + domain + ":110", StartTLS: true},
	}
}

func tlsModeForHost(host string) (implicit, starttls bool) {
	switch {
	case strings.HasSuffix(host, ":993"), strings.HasSuffix(host, ":995"), strings.HasSuffix(host, ":465"):
		return true, false
	case strings.HasSuffix(host, ":143"), strings.HasSuffix(host, ":110"), strings.HasSuffix(host, ":587"):
		return false, true
	default:
		// Custom / test ports: plain TCP (caller can still set tls flags).
		return false, false
	}
}

func dedupeProbeCands(in []probeCand) []probeCand {
	seen := map[string]bool{}
	var out []probeCand
	for _, c := range in {
		key := c.Protocol + "|" + c.Host + "|" + c.tlsMode()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func sortProbeCands(in []probeCand, prefer string) []probeCand {
	if prefer == "" {
		return in
	}
	var first, rest []probeCand
	for _, c := range in {
		if c.Protocol == prefer {
			first = append(first, c)
		} else {
			rest = append(rest, c)
		}
	}
	return append(first, rest...)
}

func probeOne(c probeCand, user, pass string, timeout time.Duration) error {
	if c.Protocol == ProtoPOP3 {
		return probePOP3(c, user, pass, timeout)
	}
	return probeIMAP(c, user, pass, timeout)
}

func dialMail(host string, implicit bool, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	if implicit {
		return tls.DialWithDialer(&d, "tcp", host, &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: serverName(host),
		})
	}
	return d.Dial("tcp", host)
}

func upgradeTLS(conn net.Conn, host string, timeout time.Duration) (net.Conn, error) {
	_ = conn.SetDeadline(time.Now().Add(timeout))
	t := tls.Client(conn, &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName(host),
	})
	if err := t.Handshake(); err != nil {
		return nil, err
	}
	return t, nil
}

func probeIMAP(c probeCand, user, pass string, timeout time.Duration) error {
	conn, err := dialMail(c.Host, c.Implicit, timeout)
	if err != nil {
		return fmt.Errorf("imap %s: %w", c.Host, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	r := bufio.NewReaderSize(conn, 64*1024)
	greet, err := readCRLF(r)
	if err != nil {
		return fmt.Errorf("imap %s greeting: %w", c.Host, err)
	}
	if !strings.Contains(strings.ToUpper(greet), "OK") && !strings.Contains(strings.ToUpper(greet), "PREAUTH") {
		return fmt.Errorf("imap %s: unexpected greeting %q", c.Host, greet)
	}
	if c.StartTLS && !c.Implicit {
		if err := writeCRLF(conn, "a1 STARTTLS"); err != nil {
			return err
		}
		ln, err := readCRLF(r)
		if err != nil {
			return fmt.Errorf("imap STARTTLS: %w", err)
		}
		if !strings.Contains(strings.ToUpper(ln), "OK") {
			return fmt.Errorf("imap STARTTLS: %s", ln)
		}
		up, err := upgradeTLS(conn, c.Host, timeout)
		if err != nil {
			return fmt.Errorf("imap STARTTLS handshake: %w", err)
		}
		conn = up
		_ = conn.SetDeadline(time.Now().Add(timeout))
		r = bufio.NewReaderSize(conn, 64*1024)
	}
	if strings.TrimSpace(pass) == "" {
		return nil
	}
	if err := writeCRLF(conn, fmt.Sprintf("a2 LOGIN %s %s", imapQuote(user), imapQuote(pass))); err != nil {
		return err
	}
	for {
		ln, err := readCRLF(r)
		if err != nil {
			return fmt.Errorf("imap LOGIN: %w", err)
		}
		up := strings.ToUpper(ln)
		if strings.HasPrefix(up, "A2 ") {
			if strings.Contains(up, " OK") {
				_ = writeCRLF(conn, "a3 LOGOUT")
				return nil
			}
			return fmt.Errorf("imap LOGIN: %s", ln)
		}
	}
}

func probePOP3(c probeCand, user, pass string, timeout time.Duration) error {
	cli := &pop3Client{cfg: ServerConfig{
		Host: c.Host, User: user, Pass: pass,
		TLS: boolPtrVal(c.Implicit), StartTLS: boolPtrVal(c.StartTLS),
	}, user: user, timeout: timeout}
	if err := cli.connect(); err != nil {
		return err
	}
	cli.close()
	return nil
}

func readCRLF(r *bufio.Reader) (string, error) {
	ln, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(ln, "\r\n"), nil
}

func writeCRLF(c net.Conn, s string) error {
	_, err := c.Write([]byte(s + "\r\n"))
	return err
}
