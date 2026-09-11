package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SendSMTP submits RFC822 on the account's SMTP server.
func SendSMTP(cfg ServerConfig, from string, to []string, raw []byte) error {
	host := cfg.Host
	if host == "" {
		return fmt.Errorf("smtp: empty host")
	}
	if !strings.Contains(host, ":") {
		if cfg.implicitTLS(false) {
			host += ":465"
		} else {
			host += ":587"
		}
	}
	user := cfg.Username(from)
	pass := cfg.Password()
	authName := strings.ToLower(strings.TrimSpace(cfg.Auth))
	serverName := serverName(host)
	dialer := net.Dialer{Timeout: 20 * time.Second}

	var client *smtp.Client
	if cfg.implicitTLS(false) && strings.HasSuffix(host, ":465") {
		conn, err := tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName})
		if err != nil {
			return fmt.Errorf("smtp: connect %s: %w", host, err)
		}
		c, err := smtp.NewClient(conn, serverName)
		if err != nil {
			_ = conn.Close()
			return err
		}
		client = c
	} else {
		conn, err := dialer.Dial("tcp", host)
		if err != nil {
			return fmt.Errorf("smtp: connect %s: %w", host, err)
		}
		c, err := smtp.NewClient(conn, serverName)
		if err != nil {
			_ = conn.Close()
			return err
		}
		client = c
		if cfg.useStartTLS() || !cfg.implicitTLS(false) {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}); err != nil {
					_ = client.Close()
					return fmt.Errorf("smtp: STARTTLS: %w", err)
				}
			}
		}
	}
	defer client.Close()

	if authName == "xoauth2" {
		return fmt.Errorf("smtp: AUTH=XOAUTH2 is a documented stub — use PLAIN for now")
	}
	if user != "" && pass != "" {
		if err := client.Auth(smtp.PlainAuth("", user, pass, serverName)); err != nil {
			// Some servers want LOGIN; retry via AUTH PLAIN raw is enough for most.
			if err2 := client.Auth(smtpLogin{user, pass}); err2 != nil {
				return fmt.Errorf("smtp: AUTH: %v / %v", err, err2)
			}
		}
	}
	if err := client.Mail(extractAddr(from)); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		rcpt = extractAddr(rcpt)
		if rcpt == "" {
			continue
		}
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: RCPT %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

type smtpLogin struct{ user, pass string }

func (a smtpLogin) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.user), nil
}

func (a smtpLogin) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	s := strings.ToLower(string(fromServer))
	if strings.Contains(s, "user") {
		return []byte(a.user), nil
	}
	return []byte(a.pass), nil
}

func extractAddr(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '<'); i >= 0 {
		if j := strings.IndexByte(s[i:], '>'); j > 0 {
			return strings.TrimSpace(s[i+1 : i+j])
		}
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func splitAddrs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if a := extractAddr(p); a != "" {
			out = append(out, a)
		}
	}
	return out
}
