package mail

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TokenBlob is a persisted OAuth token (never written to mail.json).
type TokenBlob struct {
	Provider     string    `json:"provider"`
	AccountKey   string    `json:"accountKey"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

// TokenStore is AES-GCM encrypted files under DataDir()/secrets.
// The 32-byte master key is:
//  1. secret-tool (libsecret) when `secret-tool` is on PATH, else
//  2. DataDir()/secrets/master.key (mode 0600).
//
// Documented in docs/mail.md — this is not a hardware-backed TPM vault.
type TokenStore struct {
	dir string
	mu  sync.Mutex
}

var defaultTokens sync.Once
var defaultTokenStore *TokenStore

// DefaultTokenStore is the process-wide encrypted token file store.
func DefaultTokenStore() *TokenStore {
	if defaultTokenStore != nil {
		return defaultTokenStore
	}
	defaultTokens.Do(func() {
		if defaultTokenStore == nil {
			defaultTokenStore = NewTokenStore(filepath.Join(DataDir(), "secrets"))
		}
	})
	return defaultTokenStore
}

func NewTokenStore(dir string) *TokenStore {
	return &TokenStore{dir: dir}
}

func (s *TokenStore) Put(key string, tok TokenBlob) error {
	if s == nil || key == "" {
		return fmt.Errorf("mail: token key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tok.AccountKey = key
	raw, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	keyb, err := s.masterKey()
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(keyb)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	sealed := gcm.Seal(nonce, nonce, raw, nil)
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(s.dir, slug(key)+".tok")
	return os.WriteFile(path, sealed, 0o600)
}

func (s *TokenStore) Get(key string) (TokenBlob, error) {
	if s == nil || key == "" {
		return TokenBlob{}, fmt.Errorf("mail: token key required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, slug(key)+".tok")
	sealed, err := os.ReadFile(path)
	if err != nil {
		return TokenBlob{}, err
	}
	keyb, err := s.masterKey()
	if err != nil {
		return TokenBlob{}, err
	}
	block, err := aes.NewCipher(keyb)
	if err != nil {
		return TokenBlob{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return TokenBlob{}, err
	}
	ns := gcm.NonceSize()
	if len(sealed) < ns {
		return TokenBlob{}, fmt.Errorf("mail: token blob too short")
	}
	plain, err := gcm.Open(nil, sealed[:ns], sealed[ns:], nil)
	if err != nil {
		return TokenBlob{}, fmt.Errorf("mail: decrypt token: %w", err)
	}
	var tok TokenBlob
	if err := json.Unmarshal(plain, &tok); err != nil {
		return TokenBlob{}, err
	}
	return tok, nil
}

func (s *TokenStore) Delete(key string) error {
	if s == nil || key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(filepath.Join(s.dir, slug(key)+".tok"))
}

func (s *TokenStore) masterKey() ([]byte, error) {
	if k, ok := secretToolLookup(); ok {
		sum := sha256.Sum256([]byte(k))
		return sum[:], nil
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, "master.key")
	b, err := os.ReadFile(path)
	if err == nil && len(b) == 32 {
		return b, nil
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, err
	}
	_ = secretToolStore(fmt.Sprintf("%x", key))
	return key, nil
}

func secretToolLookup() (string, bool) {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return "", false
	}
	cmd := exec.Command("secret-tool", "lookup", "service", "uitoolkit-mail", "attribute", "master")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(out))
	return s, s != ""
}

func secretToolStore(hexkey string) error {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return err
	}
	cmd := exec.Command("secret-tool", "store", "--label", "uitoolkit mail master",
		"service", "uitoolkit-mail", "attribute", "master")
	cmd.Stdin = strings.NewReader(hexkey)
	return cmd.Run()
}

// resolveAccessToken prefers UITK_MAIL_XOAUTH2, then the encrypted store
// (refreshing when expired).
func resolveAccessToken(cfg ServerConfig, address string) (string, error) {
	if t := strings.TrimSpace(os.Getenv(EnvXOAuth)); t != "" {
		return t, nil
	}
	key := tokenKey(cfg, address)
	tok, err := DefaultTokenStore().Get(key)
	if err != nil {
		return "", fmt.Errorf("imap: AUTH=XOAUTH2 needs %s or a stored refresh token (see docs/mail.md): %w", EnvXOAuth, err)
	}
	if tok.AccessToken != "" && (tok.Expiry.IsZero() || time.Now().Before(tok.Expiry.Add(-60*time.Second))) {
		return tok.AccessToken, nil
	}
	if tok.RefreshToken == "" {
		if tok.AccessToken != "" {
			return tok.AccessToken, nil
		}
		return "", fmt.Errorf("imap: stored OAuth token expired and has no refresh token")
	}
	fresh, err := refreshOAuthToken(tok)
	if err != nil {
		if tok.AccessToken != "" {
			return tok.AccessToken, nil
		}
		return "", err
	}
	_ = DefaultTokenStore().Put(key, fresh)
	return fresh.AccessToken, nil
}

func tokenKey(cfg ServerConfig, address string) string {
	if cfg.tokenKey != "" {
		return cfg.tokenKey
	}
	if u := cfg.Username(address); u != "" {
		return u
	}
	return address
}
