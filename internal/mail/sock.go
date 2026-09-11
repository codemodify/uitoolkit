package mail

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Env vars for mailclientd / mailclientui.
const (
	EnvMail   = "UITK_MAIL"      // memory (default) | imap
	EnvSock   = "UITK_MAIL_SOCK" // Unix socket path
	EnvHost   = "UITK_MAIL_HOST"
	EnvUser   = "UITK_MAIL_USER"
	EnvPass   = "UITK_MAIL_PASS"
	EnvTLS    = "UITK_MAIL_TLS"
	EnvName   = "UITK_MAIL_NAME"
)

// DefaultSocket is the mailclientd listen path.
// UITK_MAIL_SOCK overrides; else $XDG_RUNTIME_DIR/mailclientd.sock;
// else /tmp/mailclientd-<uid>.sock.
func DefaultSocket() string {
	if p := os.Getenv(EnvSock); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "mailclientd.sock")
	}
	return filepath.Join(os.TempDir(), "mailclientd-"+strconv.Itoa(os.Getuid())+".sock")
}

// OpenStore selects MemoryStore (default) or the IMAP skeleton.
func OpenStore() (Store, error) {
	switch os.Getenv(EnvMail) {
	case "", "memory", "demo", "mem":
		return NewDemoStore(), nil
	case "imap":
		s := NewIMAPStoreFromEnv()
		if err := s.Health(); err != nil {
			return s, err
		}
		if err := s.Connect(); err != nil {
			return s, err
		}
		return s, nil
	default:
		return nil, fmt.Errorf("UITK_MAIL=%s: want memory or imap", os.Getenv(EnvMail))
	}
}

// StartDemo runs mailclientd in-process on a temp socket with MemoryStore.
// The UI must still Dial — this is not an in-memory Store shortcut.
func StartDemo(ctx context.Context) (socket string, stop func(), err error) {
	dir, err := os.MkdirTemp("", "mailclientd-")
	if err != nil {
		return "", nil, err
	}
	socket = filepath.Join(dir, "mailclientd.sock")
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, socket, NewDemoStore())
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			stop = func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				_ = os.RemoveAll(dir)
			}
			return socket, stop, nil
		}
		select {
		case err := <-done:
			_ = os.RemoveAll(dir)
			if err == nil {
				err = fmt.Errorf("mailclientd exited before listen")
			}
			cancel()
			return "", nil, err
		case <-time.After(15 * time.Millisecond):
		}
	}
	cancel()
	_ = os.RemoveAll(dir)
	return "", nil, fmt.Errorf("mailclientd: socket not ready")
}
