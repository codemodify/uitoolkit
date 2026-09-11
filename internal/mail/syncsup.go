package mail

import (
	"context"
	"strings"
	"time"
)

// StartPush runs the multi-folder IDLE supervisor (INBOX + Sent) and a
// CONDSTORE/QRESYNC poll of other mailboxes. Safe to call once per LocalStore.
func (s *LocalStore) StartPush(ctx context.Context) {
	s.mu.Lock()
	if s.pushCancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	s.pushCancel = cancel
	s.mu.Unlock()
	go s.pushLoop(ctx)
}

func (s *LocalStore) pushLoop(ctx context.Context) {
	s.watchAccounts(ctx)
	tick := time.NewTicker(2 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if s.feat != nil && !s.feat.Online() {
				continue
			}
			_, _ = s.Sync("")
		}
	}
}

func (s *LocalStore) watchAccounts(ctx context.Context) {
	s.mu.Lock()
	accts := append([]Account(nil), s.accounts...)
	s.mu.Unlock()
	for _, a := range accts {
		go s.idleAccount(ctx, a.ID)
	}
}

func (s *LocalStore) idleAccount(ctx context.Context, accountID string) {
	if s.feat != nil && !s.feat.Online() {
		return
	}
	s.mu.Lock()
	cfg, ok := s.accountCfg(accountID)
	var watch []Folder
	for _, f := range s.folders {
		if f.AccountID == accountID && (f.Kind == FolderInbox || f.Kind == FolderSent) {
			watch = append(watch, f)
		}
	}
	s.mu.Unlock()
	if !ok || cfg.IsPOP3() || cfg.IMAP.Host == "" {
		return
	}
	cfg.IMAP.tokenKey = accountID
	for _, f := range watch {
		f := f
		go s.idleFolder(ctx, cfg, f)
	}
}

func (s *LocalStore) idleFolder(ctx context.Context, cfg AccountConfig, f Folder) {
	cli := newIMAPClient(cfg.IMAP, cfg.Address)
	if err := cli.connect(); err != nil {
		return
	}
	defer cli.close()
	remote := f.Remote
	if remote == "" {
		remote = f.Name
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if s.feat != nil && !s.feat.Online() {
			time.Sleep(5 * time.Second)
			continue
		}
		if _, err := cli.selectBox(remote, true); err != nil {
			time.Sleep(10 * time.Second)
			_ = cli.connect()
			continue
		}
		if !cli.has("IDLE") {
			time.Sleep(2 * time.Minute)
			s.mu.Lock()
			_, _ = s.syncFolderLocked(cli, f)
			s.mu.Unlock()
			continue
		}
		woke, err := cli.idleOnce(25 * time.Minute)
		if err != nil {
			time.Sleep(5 * time.Second)
			cli.close()
			_ = cli.connect()
			continue
		}
		if woke {
			s.mu.Lock()
			n, _ := s.syncFolderLocked(cli, f)
			s.saveLocked()
			s.mu.Unlock()
			if n > 0 {
				s.notifyNew(f.AccountID, n)
			}
		}
	}
}

func (s *LocalStore) notifyNew(accountID string, n int) {
	if s.feat == nil || n <= 0 {
		return
	}
	p := s.feat.NotifyPrefs()
	if !p.Enabled {
		return
	}
	title := "New mail"
	body := itoa(n) + " new message(s)"
	if p.VIPOnly {
		title = "VIP mail"
	}
	if p.Desktop {
		notifyDesktop(title, body)
	}
}

func syncMode(cli *imapClient) string {
	if cli == nil {
		return "poll"
	}
	var parts []string
	if cli.has("QRESYNC") {
		parts = append(parts, "QRESYNC")
	}
	if cli.has("CONDSTORE") {
		parts = append(parts, "CONDSTORE")
	}
	if cli.has("IDLE") {
		parts = append(parts, "IDLE")
	}
	if len(parts) == 0 {
		return "FLAGS FETCH"
	}
	return strings.Join(parts, "+")
}
