package mail

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore is an in-memory, maildir-ish Store. Safe for the UI thread
// plus tests. Not a network client.
type MemoryStore struct {
	mu       sync.Mutex
	accounts []Account
	folders  []Folder
	messages []Message
	nextID   int
	fetches  map[string]int
	now      time.Time
}

// NewMemoryStore builds an empty store. now is used for Fetch timestamps
// and date formatting in tests; zero means time.Now.
func NewMemoryStore(now time.Time) *MemoryStore {
	if now.IsZero() {
		now = time.Now()
	}
	return &MemoryStore{nextID: 1, fetches: map[string]int{}, now: now}
}

// NewDemoStore returns a seeded two-account mailbox (50–200 messages).
func NewDemoStore() *MemoryStore {
	s := NewMemoryStore(DemoNow)
	seedDemo(s)
	return s
}

func (s *MemoryStore) Backend() string { return "memory" }

func (s *MemoryStore) Health() error { return nil }

func (s *MemoryStore) Accounts() []Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Account, len(s.accounts))
	copy(out, s.accounts)
	return out
}

func (s *MemoryStore) ListFolders(accountID string) []Folder {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Folder
	for _, f := range s.folders {
		if f.AccountID == accountID {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := folderRank(out[i].Kind), folderRank(out[j].Kind)
		if oi != oj {
			return oi < oj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *MemoryStore) GetFolder(id FolderID) (Folder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.folderLocked(id)
}

func (s *MemoryStore) folderLocked(id FolderID) (Folder, bool) {
	for _, f := range s.folders {
		if f.ID == id {
			return f, true
		}
	}
	return Folder{}, false
}

func (s *MemoryStore) ListMessages(folder FolderID) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Message
	for _, m := range s.messages {
		if m.Folder == folder {
			out = append(out, m.Clone())
		}
	}
	return out
}

func (s *MemoryStore) GetMessage(id MessageID) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.messages {
		if m.ID == id {
			return m.Clone(), true
		}
	}
	return Message{}, false
}

func (s *MemoryStore) SetFlags(id MessageID, patch FlagPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return fmt.Errorf("mail: no message %s", id)
	}
	if patch.Read != nil {
		s.messages[i].Read = *patch.Read
	}
	if patch.Starred != nil {
		s.messages[i].Starred = *patch.Starred
	}
	if patch.Tags != nil {
		s.messages[i].Tags = append([]string(nil), (*patch.Tags)...)
	}
	return nil
}

func (s *MemoryStore) Move(ids []MessageID, dest FolderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.folderLocked(dest); !ok {
		return fmt.Errorf("mail: no folder %s", dest)
	}
	for _, id := range ids {
		i, ok := s.indexLocked(id)
		if !ok {
			return fmt.Errorf("mail: no message %s", id)
		}
		s.messages[i].Folder = dest
	}
	return nil
}

func (s *MemoryStore) Delete(ids []MessageID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		i, ok := s.indexLocked(id)
		if !ok {
			return fmt.Errorf("mail: no message %s", id)
		}
		cur, ok := s.folderLocked(s.messages[i].Folder)
		if !ok {
			return fmt.Errorf("mail: no folder for %s", id)
		}
		if cur.Kind == FolderTrash {
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			continue
		}
		trash, ok := s.specialLocked(cur.AccountID, FolderTrash)
		if !ok {
			return fmt.Errorf("mail: no trash for %s", cur.AccountID)
		}
		s.messages[i].Folder = trash.ID
	}
	return nil
}

func (s *MemoryStore) Append(folder FolderID, msg Message) (MessageID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.folderLocked(folder)
	if !ok {
		return "", fmt.Errorf("mail: no folder %s", folder)
	}
	if msg.Date.IsZero() {
		msg.Date = s.now
	}
	if msg.Size <= 0 {
		msg.Size = len(msg.Subject) + len(msg.Body) + 80
	}
	s.nextID++
	msg.ID = MessageID(fmt.Sprintf("m-%04d", s.nextID))
	msg.Folder = folder
	msg.AccountID = f.AccountID
	s.messages = append(s.messages, msg)
	return msg.ID, nil
}

func (s *MemoryStore) Update(id MessageID, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return fmt.Errorf("mail: no message %s", id)
	}
	keep := s.messages[i]
	msg.ID = keep.ID
	if msg.Folder == "" {
		msg.Folder = keep.Folder
	}
	if msg.AccountID == "" {
		msg.AccountID = keep.AccountID
	}
	if msg.Date.IsZero() {
		msg.Date = keep.Date
	}
	if msg.Size <= 0 {
		msg.Size = len(msg.Subject) + len(msg.Body) + 80
	}
	s.messages[i] = msg
	return nil
}

func (s *MemoryStore) Fetch(accountID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fetches == nil {
		s.fetches = map[string]int{}
	}
	n := s.fetches[accountID]
	if n >= 3 {
		return 0, nil
	}
	inbox, ok := s.specialLocked(accountID, FolderInbox)
	if !ok {
		return 0, fmt.Errorf("mail: no inbox for %s", accountID)
	}
	s.fetches[accountID] = n + 1
	s.nextID++
	id := MessageID(fmt.Sprintf("m-%04d", s.nextID))
	acct := accountID
	for _, a := range s.accounts {
		if a.ID == accountID {
			acct = a.Address
			break
		}
	}
	msg := Message{
		ID:        id,
		Folder:    inbox.ID,
		AccountID: accountID,
		From:      "Fetch Robot <fetch@demo.invalid>",
		To:        acct,
		Subject:   fmt.Sprintf("Get Messages demo #%d", n+1),
		Date:      s.now.Add(time.Duration(n) * time.Minute),
		Read:      false,
		Body:      "This arrival was injected by MemoryStore.Fetch.\nIMAP would FETCH unseen here.\n",
	}
	msg.Size = len(msg.Subject) + len(msg.Body) + 80
	s.messages = append(s.messages, msg)
	return 1, nil
}

func (s *MemoryStore) Unread(folder FolderID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.messages {
		if m.Folder == folder && !m.Read {
			n++
		}
	}
	return n
}

func (s *MemoryStore) UnreadTotal() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.messages {
		if !m.Read {
			n++
		}
	}
	return n
}

func (s *MemoryStore) MessageCount(folder FolderID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.messages {
		if m.Folder == folder {
			n++
		}
	}
	return n
}

func (s *MemoryStore) CreateFolder(accountID, name string, parent FolderID) (Folder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if accountID == "" || name == "" {
		return Folder{}, fmt.Errorf("mail: account and folder name required")
	}
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id := FolderID(accountID + "/" + slug)
	if parent != "" {
		id = parent + "/" + FolderID(slug)
	}
	for _, f := range s.folders {
		if f.ID == id {
			return Folder{}, fmt.Errorf("mail: folder %s exists", id)
		}
	}
	f := Folder{ID: id, AccountID: accountID, Name: name, Kind: FolderCustom, Parent: parent}
	s.folders = append(s.folders, f)
	return f, nil
}

func (s *MemoryStore) Search(q SearchQuery) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Message
	for _, m := range s.messages {
		if q.AccountID != "" && m.AccountID != q.AccountID {
			continue
		}
		if q.Folder != "" && m.Folder != q.Folder {
			continue
		}
		if q.Filter.Match(m) {
			out = append(out, m.Clone())
		}
	}
	return out
}

// Folders / Folder / List / Get are aliases for older call sites.
func (s *MemoryStore) Folders(accountID string) []Folder { return s.ListFolders(accountID) }
func (s *MemoryStore) Folder(id FolderID) (Folder, bool) { return s.GetFolder(id) }
func (s *MemoryStore) List(folder FolderID) []Message    { return s.ListMessages(folder) }
func (s *MemoryStore) Get(id MessageID) (Message, bool)  { return s.GetMessage(id) }

func (s *MemoryStore) indexLocked(id MessageID) (int, bool) {
	for i, m := range s.messages {
		if m.ID == id {
			return i, true
		}
	}
	return -1, false
}

func (s *MemoryStore) specialLocked(accountID string, kind FolderKind) (Folder, bool) {
	for _, f := range s.folders {
		if f.AccountID == accountID && f.Kind == kind {
			return f, true
		}
	}
	return Folder{}, false
}

func (s *MemoryStore) addAccount(a Account) { s.accounts = append(s.accounts, a) }

func (s *MemoryStore) addFolder(f Folder) { s.folders = append(s.folders, f) }

func (s *MemoryStore) addMessage(m Message) {
	s.nextID++
	if m.ID == "" {
		m.ID = MessageID(fmt.Sprintf("m-%04d", s.nextID))
	}
	if m.Size <= 0 {
		m.Size = len(m.Subject) + len(m.Body) + 80
		if m.HasAttach {
			m.Size += 24 * 1024
		}
	}
	s.messages = append(s.messages, m)
}

func folderRank(k FolderKind) int {
	switch k {
	case FolderInbox:
		return 0
	case FolderDrafts:
		return 1
	case FolderSent:
		return 2
	case FolderJunk:
		return 3
	case FolderTrash:
		return 4
	case FolderArchive:
		return 5
	default:
		return 10
	}
}

func (s *MemoryStore) Special(accountID string, kind FolderKind) (Folder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.specialLocked(accountID, kind)
}

func (s *MemoryStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}
