package mail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Env extras for the real client.
const (
	EnvConfig   = "UITK_MAIL_CONFIG"
	EnvData     = "UITK_MAIL_DATA"
	EnvSMTPHost = "UITK_MAIL_SMTP"
	EnvPassEnv  = "UITK_MAIL_PASS_ENV" // name of env var that holds the password
	EnvXOAuth   = "UITK_MAIL_XOAUTH2"
)

// MailConfig is ~/.config/uitoolkit/mail.json (never store raw secrets).
type MailConfig struct {
	Accounts []AccountConfig `json:"accounts"`
}

// AccountConfig is one IMAP + SMTP login.
type AccountConfig struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Address    string           `json:"address"`
	IMAP       ServerConfig     `json:"imap"`
	SMTP       ServerConfig     `json:"smtp"`
	Identities []Identity       `json:"identities,omitempty"`
}

// ServerConfig is a host + how to get the password.
//
//	"passEnv": "UITK_MAIL_PASS"  reads os.Getenv
//	"user" may differ from the From address
//
// tls: implicit TLS (993 / 465). starttls: upgrade after connect (587 / 143).
type ServerConfig struct {
	Host     string `json:"host"`
	User     string `json:"user,omitempty"`
	PassEnv  string `json:"passEnv,omitempty"`
	TLS      *bool  `json:"tls,omitempty"`
	StartTLS *bool  `json:"starttls,omitempty"`
	Auth     string `json:"auth,omitempty"` // plain (default), login, xoauth2
}

func (s ServerConfig) Username(fallback string) string {
	if strings.TrimSpace(s.User) != "" {
		return s.User
	}
	return fallback
}

func (s ServerConfig) Password() string {
	name := strings.TrimSpace(s.PassEnv)
	if name == "" {
		name = EnvPass
	}
	return os.Getenv(name)
}

func (s ServerConfig) implicitTLS(defaultTLS bool) bool {
	if s.TLS != nil {
		return *s.TLS
	}
	host := s.Host
	if strings.HasSuffix(host, ":993") || strings.HasSuffix(host, ":465") {
		return true
	}
	if strings.HasSuffix(host, ":587") || strings.HasSuffix(host, ":143") || strings.HasSuffix(host, ":25") {
		return false
	}
	return defaultTLS
}

func (s ServerConfig) useStartTLS() bool {
	if s.StartTLS != nil {
		return *s.StartTLS
	}
	return strings.HasSuffix(s.Host, ":587") || strings.HasSuffix(s.Host, ":143")
}

// ConfigPath is the account file.
func ConfigPath() string {
	if p := os.Getenv(EnvConfig); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", "mail.json")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "uitoolkit", "mail.json")
	}
	return filepath.Join(os.TempDir(), "uitoolkit-mail.json")
}

// DataDir is the on-disk cache root.
func DataDir() string {
	if p := os.Getenv(EnvData); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", "mail")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".local", "share", "uitoolkit", "mail")
	}
	return filepath.Join(os.TempDir(), "uitoolkit-mail")
}

// LoadConfig reads the JSON file. Missing file is not an error (empty config).
func LoadConfig() (MailConfig, error) {
	path := ConfigPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return MailConfig{}, nil
		}
		return MailConfig{}, err
	}
	var cfg MailConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return MailConfig{}, fmt.Errorf("mail config %s: %w", path, err)
	}
	return cfg, nil
}

// ConfigFromEnv builds a single-account config from UITK_MAIL_* (no file).
func ConfigFromEnv() (MailConfig, error) {
	host := strings.TrimSpace(os.Getenv(EnvHost))
	user := strings.TrimSpace(os.Getenv(EnvUser))
	if host == "" || user == "" {
		return MailConfig{}, fmt.Errorf("imap: set UITK_MAIL_HOST and UITK_MAIL_USER (or a config file)")
	}
	name := strings.TrimSpace(os.Getenv(EnvName))
	if name == "" {
		name = DisplayName(user)
	}
	smtpHost := strings.TrimSpace(os.Getenv(EnvSMTPHost))
	if smtpHost == "" {
		smtpHost = guessSMTP(host)
	}
	passEnv := strings.TrimSpace(os.Getenv(EnvPassEnv))
	if passEnv == "" {
		passEnv = EnvPass
	}
	id := "imap"
	acc := AccountConfig{
		ID: id, Name: name, Address: user,
		IMAP: ServerConfig{Host: host, User: user, PassEnv: passEnv},
		SMTP: ServerConfig{Host: smtpHost, User: user, PassEnv: passEnv},
		Identities: []Identity{{
			ID: id + "-default", AccountID: id, Name: name, Address: user, Default: true,
		}},
	}
	return MailConfig{Accounts: []AccountConfig{acc}}, nil
}

func guessSMTP(imapHost string) string {
	host, _, _ := strings.Cut(imapHost, ":")
	host = strings.TrimPrefix(host, "imap.")
	if host == imapHost || host == "" {
		return "smtp." + strings.TrimPrefix(imapHost, "imap.")
	}
	return "smtp." + host + ":587"
}

func boolPtrVal(v bool) *bool { return &v }
