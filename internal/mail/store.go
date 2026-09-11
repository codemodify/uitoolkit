// Package mail is the Thunderbird-chrome Mail dogfood example:
// a pluggable Store plus the 3-pane desktop UI.
//
// v1 ships MemoryStore (in-memory, maildir-ish flags). A future IMAP
// backend can implement Store without rewriting the chrome — see the
// IMAP notes on Store.
package mail

import (
	"fmt"
	"strings"
	"time"
)

// FolderID names one mailbox (account-local, like a maildir directory).
type FolderID string

// MessageID is a stable id inside one Store.
type MessageID string

// FolderKind is a special-use mailbox (RFC 6154-ish) or a user folder.
type FolderKind int

const (
	FolderCustom FolderKind = iota
	FolderInbox
	FolderDrafts
	FolderSent
	FolderJunk
	FolderTrash
	FolderArchive
)

func (k FolderKind) String() string {
	switch k {
	case FolderInbox:
		return "Inbox"
	case FolderDrafts:
		return "Drafts"
	case FolderSent:
		return "Sent"
	case FolderJunk:
		return "Junk"
	case FolderTrash:
		return "Trash"
	case FolderArchive:
		return "Archives"
	default:
		return "Folder"
	}
}

// Account is one identity (From: address). IMAP would be one server login.
type Account struct {
	ID      string
	Name    string
	Address string
}

// Folder is one mailbox under an account. Parent is empty for top-level.
type Folder struct {
	ID        FolderID
	AccountID string
	Name      string
	Kind      FolderKind
	Parent    FolderID
}

// Message is a full RFC-822-ish record. Body is plain text (no MIME tree).
type Message struct {
	ID          MessageID
	Folder      FolderID
	AccountID   string
	From        string
	To          string
	Cc          string
	Bcc         string
	Subject     string
	Date        time.Time
	Size        int
	Read        bool
	Starred     bool
	HasAttach   bool
	Tags        []string
	Body        string
	Attachments []string
}

// Clone returns a shallow copy (tags / attachments copied).
func (m Message) Clone() Message {
	out := m
	if m.Tags != nil {
		out.Tags = append([]string(nil), m.Tags...)
	}
	if m.Attachments != nil {
		out.Attachments = append([]string(nil), m.Attachments...)
	}
	return out
}

// DisplayName returns the phrase before <email>, or the whole string.
func DisplayName(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.IndexByte(addr, '<'); i > 0 {
		return strings.TrimSpace(addr[:i])
	}
	return addr
}

// Correspondent is From in incoming folders and To in Sent/Drafts.
func (m Message) Correspondent(kind FolderKind) string {
	if kind == FolderSent || kind == FolderDrafts {
		return DisplayName(firstAddr(m.To))
	}
	return DisplayName(firstAddr(m.From))
}

func firstAddr(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// FlagPatch is a partial flag update (nil pointer = leave unchanged).
type FlagPatch struct {
	Read    *bool
	Starred *bool
	Tags    *[]string
}

// Store is the mail backend. MemoryStore is the v1 demo.
//
// Plugging in IMAP later (do not rewrite the UI):
//
//	type IMAPStore struct {
//	    // github.com/emersion/go-imap/v2 client, one selected mailbox, …
//	}
//
//	Accounts  — LOGIN + identity from the settings UI
//	Folders   — LIST / XLIST / SPECIAL-USE; map \Inbox \Sent \Trash \Junk \Drafts
//	List      — SELECT + FETCH 1:* (FLAGS ENVELOPE RFC822.SIZE)
//	Get       — FETCH BODY.PEEK[] or BODY[TEXT] (decode MIME)
//	SetFlags  — STORE +FLAGS.SILENT (\Seen \Flagged) / keywords for tags
//	Move      — MOVE (RFC 6851) or COPY + STORE \Deleted + EXPUNGE
//	Delete    — MOVE to \Trash, or EXPUNGE when already in Trash
//	Append    — APPEND to \Drafts / \Sent (maildir: new/ + flags file)
//	Update    — replace a draft (IMAP: APPEND + delete old UID)
//	Fetch     — NOOP or IDLE, then FETCH unseen; SMTP is separate
//	Send      — net/smtp or AUTH submission, then Append to Sent
//
// Maildir mapping (if you persist MemoryStore later): one directory per
// Folder, filename flags `:2,S` (Seen) and `:2,F` (Flagged), optional
// `cur/` / `new/` split. The UI only talks to this interface.
type Store interface {
	Accounts() []Account
	Folders(accountID string) []Folder
	Folder(id FolderID) (Folder, bool)

	List(folder FolderID) []Message
	Get(id MessageID) (Message, bool)

	SetFlags(id MessageID, patch FlagPatch) error
	Move(ids []MessageID, dest FolderID) error
	Delete(ids []MessageID) error
	Append(folder FolderID, msg Message) (MessageID, error)
	Update(id MessageID, msg Message) error

	// Fetch is “Get Messages”. MemoryStore injects a couple of demo
	// arrivals; IMAP would poll or IDLE.
	Fetch(accountID string) (newCount int, err error)

	Unread(folder FolderID) int
	UnreadTotal() int
}

func boolPtr(v bool) *bool { return &v }

func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func toggleTag(tags []string, tag string) []string {
	out := tags[:0:0]
	found := false
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			found = true
			continue
		}
		out = append(out, t)
	}
	if !found {
		return append(append([]string(nil), tags...), tag)
	}
	return out
}

func formatSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		kb := (n + 512) / 1024
		if kb < 1 {
			kb = 1
		}
		return fmt.Sprintf("%d KB", kb)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func formatDate(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	local := t.In(now.Location())
	y1, m1, d1 := now.Date()
	y2, m2, d2 := local.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return local.Format("3:04 PM")
	}
	yd := now.AddDate(0, 0, -1)
	yy, ym, ydN := yd.Date()
	if y2 == yy && m2 == ym && d2 == ydN {
		return "Yesterday"
	}
	if now.Sub(local) >= 0 && now.Sub(local) < 7*24*time.Hour {
		return local.Format("Monday")
	}
	return local.Format("2006-01-02")
}
