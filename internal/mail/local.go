package mail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LocalStore is the production backend: on-disk cache + IMAP or POP3 sync + SMTP.
type LocalStore struct {
	mu         sync.Mutex
	dir        string
	cfg        MailConfig
	accounts   []Account
	identities []Identity
	folders    []Folder
	messages   []Message
	tags       []Tag
	rules      []FilterRule
	clients    map[string]*imapClient
	nextID     int
	health     error
	now        time.Time
	feat       *featureHost
	pushCancel func()
}

type folderMeta struct {
	UIDValidity uint32 `json:"uidValidity"`
	UIDNext     uint32 `json:"uidNext"`
	HighestMod  uint64 `json:"highestModseq,omitempty"`
	Remote      string `json:"remote"`
}

// NewLocalStore opens (or creates) the disk cache for cfg.
func NewLocalStore(cfg MailConfig) (*LocalStore, error) {
	return NewLocalStoreDir(cfg, DataDir())
}

func NewLocalStoreDir(cfg MailConfig, dir string) (*LocalStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &LocalStore{
		dir: dir, cfg: cfg, clients: map[string]*imapClient{},
		tags: DefaultTags(), nextID: 1, now: time.Now(), feat: newFeatureHost(),
	}
	s.loadLocked()
	for _, a := range cfg.Accounts {
		s.ensureAccount(a)
	}
	if len(s.accounts) == 0 {
		s.health = fmt.Errorf("mail: no accounts in %s", ConfigPath())
	}
	s.saveLocked()
	return s, nil
}

func (s *LocalStore) Backend() string { return "imap" }

func (s *LocalStore) Health() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

func (s *LocalStore) ensureAccount(a AccountConfig) {
	id := a.ID
	if id == "" {
		id = slug(a.Address)
	}
	found := false
	for i, x := range s.accounts {
		if x.ID == id {
			a.ID = id
			s.accounts[i] = accountFromConfig(a, "")
			found = true
			break
		}
	}
	if !found {
		a.ID = id
		s.accounts = append(s.accounts, accountFromConfig(a, ""))
	}
	if len(a.Identities) > 0 {
		for _, idn := range a.Identities {
			idn.AccountID = id
			s.identities = upsertIdentity(s.identities, idn)
		}
	} else if !hasIdentityFor(s.identities, id) {
		s.identities = append(s.identities, Identity{
			ID: id + "-default", AccountID: id, Name: a.Name, Address: a.Address, Default: true,
		})
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "@", "-")
	s = strings.ReplaceAll(s, ".", "-")
	if s == "" {
		return "acct"
	}
	return s
}

func hasIdentityFor(ids []Identity, accountID string) bool {
	for _, id := range ids {
		if id.AccountID == accountID {
			return true
		}
	}
	return false
}

func upsertIdentity(list []Identity, id Identity) []Identity {
	if id.ID == "" {
		id.ID = slug(id.Address)
	}
	for i, x := range list {
		if x.ID == id.ID {
			list[i] = id
			return list
		}
	}
	return append(list, id)
}

func (s *LocalStore) PutAccount(in AccountConfig) (Account, error) {
	a, err := SanitizeAccountConfig(in)
	if err != nil {
		return Account{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, _ := LoadConfig()
	a = keepExistingSecrets(a, file.Accounts)
	file.Accounts = upsertAccountConfig(file.Accounts, a)
	if err := SaveConfig(file); err != nil {
		return Account{}, err
	}
	s.cfg.Accounts = upsertAccountConfig(s.cfg.Accounts, a)
	s.ensureAccount(a)
	s.health = nil
	s.saveLocked()
	for _, x := range s.accounts {
		if x.ID == a.ID {
			return x, nil
		}
	}
	return accountFromConfig(a, ""), nil
}

func (s *LocalStore) DeleteAccount(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("mail: account id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	addr := ""
	found := false
	for _, a := range s.accounts {
		if a.ID == id {
			found, addr = true, a.Address
			break
		}
	}
	if !found {
		for _, a := range s.cfg.Accounts {
			aid := a.ID
			if aid == "" {
				aid = slug(a.Address)
			}
			if aid == id || slug(a.Address) == id {
				found, id, addr = true, aid, a.Address
				break
			}
		}
	}
	if !found {
		return fmt.Errorf("mail: no account %s", id)
	}
	if c := s.clients[id]; c != nil {
		c.close()
		delete(s.clients, id)
	}
	file, _ := LoadConfig()
	file.Accounts = dropAccountConfig(file.Accounts, id)
	if err := SaveConfig(file); err != nil {
		return err
	}
	s.cfg.Accounts = dropAccountConfig(s.cfg.Accounts, id)
	s.dropAccountLocked(id)
	_ = DefaultTokenStore().Delete(id)
	if addr != "" {
		_ = DefaultTokenStore().Delete(addr)
	}
	if len(s.accounts) == 0 {
		s.health = fmt.Errorf("mail: no accounts in %s", ConfigPath())
	}
	s.saveLocked()
	return nil
}

func (s *LocalStore) dropAccountLocked(id string) {
	accts := s.accounts[:0]
	for _, a := range s.accounts {
		if a.ID != id {
			accts = append(accts, a)
		}
	}
	s.accounts = accts
	idents := s.identities[:0]
	for _, idn := range s.identities {
		if idn.AccountID != id {
			idents = append(idents, idn)
		}
	}
	s.identities = idents
	var drop []FolderID
	folders := s.folders[:0]
	for _, f := range s.folders {
		if f.AccountID == id {
			drop = append(drop, f.ID)
			continue
		}
		folders = append(folders, f)
	}
	s.folders = folders
	msgs := s.messages[:0]
	for _, m := range s.messages {
		if m.AccountID == id {
			if s.feat != nil && s.feat.index != nil {
				s.feat.index.remove(m.ID)
			}
			continue
		}
		msgs = append(msgs, m)
	}
	s.messages = msgs
	if s.feat != nil {
		ops := s.feat.outbox[:0]
		for _, op := range s.feat.outbox {
			if op.AccountID == id {
				continue
			}
			ops = append(ops, op)
		}
		s.feat.outbox = ops
	}
	_ = os.RemoveAll(filepath.Join(s.dir, "raw", id))
	for _, fid := range drop {
		_ = os.Remove(filepath.Join(s.dir, "meta", slug(string(fid))+".json"))
	}
}

func (s *LocalStore) Accounts() []Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Account, len(s.accounts))
	copy(out, s.accounts)
	return out
}

func (s *LocalStore) ListFolders(accountID string) []Folder {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Folder
	for _, f := range s.folders {
		if f.AccountID == accountID && !f.Virtual {
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

func (s *LocalStore) GetFolder(id FolderID) (Folder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sid := SmartFolderID(id); sid != "" && s.feat != nil {
		for _, sf := range s.feat.ListSmartFolders() {
			if sf.ID == sid {
				return Folder{ID: id, AccountID: AccountSmart, Name: sf.Name, Kind: FolderCustom, Virtual: true}, true
			}
		}
	}
	if f, ok := virtualFolderByID(id); ok {
		return f, true
	}
	return s.folderLocked(id)
}

func (s *LocalStore) folderLocked(id FolderID) (Folder, bool) {
	for _, f := range s.folders {
		if f.ID == id {
			return f, true
		}
	}
	return Folder{}, false
}

func (s *LocalStore) CreateFolder(accountID, name string, parent FolderID) (Folder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if accountID == "" || name == "" {
		return Folder{}, fmt.Errorf("mail: account and folder name required")
	}
	remote := name
	if parent != "" {
		if p, ok := s.folderLocked(parent); ok && p.Remote != "" {
			remote = p.Remote + "/" + name
		}
	}
	id := FolderID(accountID + "/" + slug(name))
	f := Folder{ID: id, AccountID: accountID, Name: name, Kind: FolderCustom, Parent: parent, Remote: remote}
	if cli, err := s.clientLocked(accountID); err == nil {
		if err := cli.createMailbox(remote); err != nil {
			return Folder{}, err
		}
	}
	s.folders = append(s.folders, f)
	s.saveLocked()
	return f, nil
}

func (s *LocalStore) ListMessages(folder FolderID) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listLocked(folder)
}

func (s *LocalStore) listLocked(folder FolderID) []Message {
	var out []Message
	for _, m := range s.messages {
		f, ok := s.folderLocked(m.Folder)
		kind := FolderCustom
		if ok {
			kind = f.Kind
		}
		if IsVirtual(folder) {
			if matchVirtual(folder, m, f, kind, s.feat.snap()) {
				out = append(out, m.Clone())
			}
			continue
		}
		if m.Folder == folder {
			out = append(out, m.Clone())
		}
	}
	return out
}

func (s *LocalStore) GetMessage(id MessageID) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return Message{}, false
	}
	m := s.messages[i].Clone()
	if m.Body == "" && m.HTML == "" {
		if raw := s.readRawLocked(m); len(raw) > 0 {
			if parsed, err := ParseRFC822(raw, m.Folder, m.AccountID); err == nil {
				parsed.ID = m.ID
				parsed.UID = m.UID
				parsed.Read = m.Read
				parsed.Starred = m.Starred
				parsed.Tags = m.Tags
				s.messages[i] = parsed
				m = parsed.Clone()
			}
		} else {
			s.fetchBodyLocked(&s.messages[i])
			m = s.messages[i].Clone()
		}
	}
	return m, true
}

func (s *LocalStore) GetRaw(id MessageID) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return nil, fmt.Errorf("mail: no message %s", id)
	}
	m := s.messages[i]
	if raw := s.readRawLocked(m); len(raw) > 0 {
		return append([]byte(nil), raw...), nil
	}
	s.fetchBodyLocked(&s.messages[i])
	if raw := s.readRawLocked(s.messages[i]); len(raw) > 0 {
		return append([]byte(nil), raw...), nil
	}
	return nil, fmt.Errorf("mail: no raw source for %s", id)
}

func (s *LocalStore) fetchBodyLocked(m *Message) {
	if m == nil {
		return
	}
	if raw := s.readRawLocked(*m); len(raw) > 0 {
		if parsed, err := ParseRFC822(raw, m.Folder, m.AccountID); err == nil {
			parsed.ID = m.ID
			parsed.UID = m.UID
			parsed.Read = m.Read
			parsed.Starred = m.Starred
			parsed.Tags = m.Tags
			*m = parsed
		}
		return
	}
	cli, err := s.clientLocked(m.AccountID)
	if err != nil {
		return
	}
	f, ok := s.folderLocked(m.Folder)
	if !ok || m.UID == 0 {
		return
	}
	remote := f.Remote
	if remote == "" {
		remote = f.Name
	}
	if _, err := cli.selectBox(remote, true); err != nil {
		return
	}
	raw, err := cli.uidFetchRFC822(m.UID)
	if err != nil || len(raw) == 0 {
		return
	}
	s.writeRawLocked(*m, raw)
	if parsed, err := ParseRFC822(raw, m.Folder, m.AccountID); err == nil {
		parsed.ID = m.ID
		parsed.UID = m.UID
		parsed.Read = m.Read
		parsed.Starred = m.Starred
		parsed.Tags = m.Tags
		*m = parsed
	}
}

func (s *LocalStore) Search(q SearchQuery) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.feat.snap()
	var idx *searchIndex
	if s.feat != nil {
		idx = s.feat.index
	}
	hits := searchMessages(s.messages, SearchQuery{AccountID: q.AccountID, Filter: q.Filter}, idx)
	if q.Folder == "" {
		return hits
	}
	var out []Message
	for _, m := range hits {
		f, ok := s.folderLocked(m.Folder)
		kind := FolderCustom
		if ok {
			kind = f.Kind
		}
		if IsVirtual(q.Folder) {
			if matchVirtual(q.Folder, m, f, kind, snap) {
				out = append(out, m)
			}
			continue
		}
		if m.Folder == q.Folder {
			out = append(out, m)
		}
	}
	return out
}

func (s *LocalStore) SetFlags(id MessageID, patch FlagPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return fmt.Errorf("mail: no message %s", id)
	}
	m := &s.messages[i]
	var add, rem []string
	if patch.Read != nil {
		m.Read = *patch.Read
		if *patch.Read {
			add = append(add, `\Seen`)
		} else {
			rem = append(rem, `\Seen`)
		}
	}
	if patch.Starred != nil {
		m.Starred = *patch.Starred
		if *patch.Starred {
			add = append(add, `\Flagged`)
		} else {
			rem = append(rem, `\Flagged`)
		}
	}
	if patch.Tags != nil {
		m.Tags = append([]string(nil), (*patch.Tags)...)
		for _, t := range m.Tags {
			add = append(add, imapSafeKeyword(t))
		}
	}
	if s.feat != nil && !s.feat.Online() {
		s.feat.mu.Lock()
		s.feat.enqueueLocked(OutboxOp{Kind: "flag", MessageID: id, Patch: patch, AccountID: m.AccountID, UID: m.UID})
		s.feat.mu.Unlock()
		s.saveLocked()
		return nil
	}
	s.pushFlagsLocked(*m, add, rem)
	s.saveLocked()
	return nil
}

func imapSafeKeyword(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func (s *LocalStore) pushFlagsLocked(m Message, add, rem []string) {
	if m.UID == 0 {
		return
	}
	cli, err := s.clientLocked(m.AccountID)
	if err != nil {
		return
	}
	f, ok := s.folderLocked(m.Folder)
	if !ok {
		return
	}
	remote := f.Remote
	if remote == "" {
		remote = f.Name
	}
	if _, err := cli.selectBox(remote, false); err != nil {
		return
	}
	_ = cli.uidStore(m.UID, add, rem)
}

func (s *LocalStore) Move(ids []MessageID, dest FolderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	df, ok := s.folderLocked(dest)
	if !ok {
		return fmt.Errorf("mail: no folder %s", dest)
	}
	for _, id := range ids {
		i, ok := s.indexLocked(id)
		if !ok {
			return fmt.Errorf("mail: no message %s", id)
		}
		m := s.messages[i]
		if s.feat != nil && !s.feat.Online() {
			s.feat.mu.Lock()
			s.feat.enqueueLocked(OutboxOp{Kind: "move", MessageID: id, Dest: dest, AccountID: m.AccountID, UID: m.UID})
			s.feat.mu.Unlock()
			s.messages[i].Folder = dest
			continue
		}
		if m.UID != 0 {
			if cli, err := s.clientLocked(m.AccountID); err == nil {
				sf, _ := s.folderLocked(m.Folder)
				remote := sf.Remote
				if remote == "" {
					remote = sf.Name
				}
				dremote := df.Remote
				if dremote == "" {
					dremote = df.Name
				}
				if _, err := cli.selectBox(remote, false); err == nil {
					if err := cli.uidMove(m.UID, dremote); err != nil {
						s.feat.mu.Lock()
						s.feat.enqueueLocked(OutboxOp{Kind: "move", MessageID: id, Dest: dest, AccountID: m.AccountID, UID: m.UID, Error: err.Error()})
						s.feat.mu.Unlock()
					}
				}
			}
		}
		s.messages[i].Folder = dest
	}
	s.saveLocked()
	return nil
}

func (s *LocalStore) Delete(ids []MessageID) error {
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
		if s.feat != nil && !s.feat.Online() {
			s.feat.mu.Lock()
			s.feat.enqueueLocked(OutboxOp{Kind: "delete", MessageID: id, AccountID: s.messages[i].AccountID, UID: s.messages[i].UID})
			s.feat.mu.Unlock()
		}
		if cur.Kind == FolderTrash {
			if s.feat == nil || s.feat.Online() {
				s.expungeLocked(s.messages[i])
			}
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			continue
		}
		trash, ok := s.specialLocked(cur.AccountID, FolderTrash)
		if !ok {
			s.expungeLocked(s.messages[i])
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			continue
		}
		s.messages[i].Folder = trash.ID
		if s.messages[i].UID != 0 {
			if cli, err := s.clientLocked(cur.AccountID); err == nil {
				remote := cur.Remote
				if remote == "" {
					remote = cur.Name
				}
				dremote := trash.Remote
				if dremote == "" {
					dremote = trash.Name
				}
				if _, err := cli.selectBox(remote, false); err == nil {
					_ = cli.uidMove(s.messages[i].UID, dremote)
				}
			}
		}
	}
	s.saveLocked()
	return nil
}

func (s *LocalStore) expungeLocked(m Message) {
	if m.UID == 0 {
		return
	}
	cli, err := s.clientLocked(m.AccountID)
	if err != nil {
		return
	}
	f, ok := s.folderLocked(m.Folder)
	if !ok {
		return
	}
	remote := f.Remote
	if remote == "" {
		remote = f.Name
	}
	if _, err := cli.selectBox(remote, false); err != nil {
		return
	}
	_ = cli.uidStore(m.UID, []string{`\Deleted`}, nil)
	_ = cli.expunge()
}

func (s *LocalStore) Append(folder FolderID, msg Message) (MessageID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.folderLocked(folder)
	if !ok {
		return "", fmt.Errorf("mail: no folder %s", folder)
	}
	if msg.Date.IsZero() {
		msg.Date = s.now
	}
	s.nextID++
	msg.ID = MessageID(fmt.Sprintf("%s-m-%04d", f.AccountID, s.nextID))
	msg.Folder = folder
	msg.AccountID = f.AccountID
	if msg.ThreadID == "" {
		msg.ThreadID = ThreadIDOf(msg)
	}
	if msg.Category == "" && s.feat != nil {
		msg.Category = messageCategory(msg, s.feat.snap())
	}
	if msg.Size <= 0 {
		msg.Size = len(msg.Subject) + len(msg.Body) + 80
	}
	ident := s.defaultIdentLocked(f.AccountID)
	raw := BuildRFC822(msg, ident, nil)
	s.writeRawLocked(msg, raw)
	if cli, err := s.clientLocked(f.AccountID); err == nil {
		remote := f.Remote
		if remote == "" {
			remote = f.Name
		}
		_ = cli.appendRaw(remote, raw, `\Seen`)
	}
	s.messages = append(s.messages, msg)
	if s.feat != nil && s.feat.index != nil {
		s.feat.index.add(msg)
	}
	s.saveLocked()
	return msg.ID, nil
}

func (s *LocalStore) Update(id MessageID, msg Message) error {
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
	msg.UID = keep.UID
	s.messages[i] = msg
	s.saveLocked()
	return nil
}

func (s *LocalStore) Fetch(accountID string) (int, error) {
	res, err := s.Sync(accountID)
	return res.New, err
}

func (s *LocalStore) Sync(accountID string) (SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := SyncResult{AccountID: accountID}
	accts := s.accounts
	if accountID != "" {
		accts = nil
		for _, a := range s.accounts {
			if a.ID == accountID {
				accts = []Account{a}
			}
		}
	}
	for _, a := range accts {
		n, err := s.syncAccountLocked(a.ID)
		res.New += n
		nf := 0
		for _, f := range s.folders {
			if f.AccountID == a.ID && !f.Virtual {
				nf++
			}
		}
		res.Folders += nf
		if err != nil {
			s.health = err
			res.Error = err.Error()
		}
	}
	s.saveLocked()
	return res, s.health
}

func (s *LocalStore) syncAccountLocked(accountID string) (int, error) {
	if cfg, ok := s.accountCfg(accountID); ok && cfg.IsPOP3() {
		return s.syncPOP3Locked(accountID)
	}
	cli, err := s.clientLocked(accountID)
	if err != nil {
		return 0, err
	}
	boxes, err := cli.list()
	if err != nil {
		return 0, err
	}
	added := 0
	for _, b := range boxes {
		if b.Name == "" {
			continue
		}
		kind := folderKindFromIMAP(b.Name, b.Attrs)
		id := FolderID(accountID + "/" + slug(b.Name))
		if kind == FolderInbox {
			id = FolderID(accountID + "/inbox")
		}
		f, ok := s.folderLocked(id)
		if !ok {
			f = Folder{ID: id, AccountID: accountID, Name: displayIMAPName(b.Name, b.Delim), Kind: kind, Remote: b.Name}
			s.folders = append(s.folders, f)
		} else {
			f.Remote = b.Name
			f.Kind = kind
			s.replaceFolderLocked(f)
		}
		n, err := s.syncFolderLocked(cli, f)
		if err != nil {
			s.health = err
			continue
		}
		added += n
	}
	return added, nil
}

func (s *LocalStore) syncPOP3Locked(accountID string) (int, error) {
	cfg, ok := s.accountCfg(accountID)
	if !ok {
		return 0, fmt.Errorf("mail: no account %s", accountID)
	}
	in := cfg.Incoming()
	if in.Host == "" {
		return 0, fmt.Errorf("mail: no POP3 host for %s", accountID)
	}
	in.tokenKey = accountID
	cli := newPOP3Client(in, cfg.Address)
	if err := cli.connect(); err != nil {
		return 0, err
	}
	defer cli.close()

	s.ensureLocalSpecialsLocked(accountID)
	inbox, ok := s.specialLocked(accountID, FolderInbox)
	if !ok {
		return 0, fmt.Errorf("mail: no inbox for %s", accountID)
	}

	uidls, err := cli.uidl()
	if err != nil {
		n, sterr := cli.stat()
		if sterr != nil {
			return 0, err
		}
		for i := 1; i <= n; i++ {
			uidls = append(uidls, popUIDL{N: i, UIDL: fmt.Sprintf("n%d", i)})
		}
	}

	haveRFC := map[string]bool{}
	for _, m := range s.messages {
		if m.AccountID == accountID && m.RFCMessageID != "" {
			haveRFC[strings.TrimSpace(m.RFCMessageID)] = true
		}
	}

	added := 0
	for _, u := range uidls {
		id := popMessageID(accountID, u.UIDL)
		if _, exists := s.indexLocked(id); exists {
			continue
		}
		raw, err := cli.retr(u.N)
		if err != nil {
			s.health = err
			continue
		}
		msg, err := ParseRFC822(raw, inbox.ID, accountID)
		if err != nil {
			msg = Message{Folder: inbox.ID, AccountID: accountID, Body: string(raw), Size: len(raw)}
		}
		if mid := strings.TrimSpace(msg.RFCMessageID); mid != "" && haveRFC[mid] {
			continue
		}
		msg.ID = id
		msg.Folder = inbox.ID
		msg.AccountID = accountID
		if msg.Date.IsZero() {
			msg.Date = s.now
		}
		if msg.ThreadID == "" {
			msg.ThreadID = ThreadIDOf(msg)
		}
		if msg.Category == "" && s.feat != nil {
			msg.Category = messageCategory(msg, s.feat.snap())
		}
		s.writeRawLocked(msg, raw)
		s.messages = append(s.messages, msg)
		if s.feat != nil && s.feat.index != nil {
			s.feat.index.add(msg)
		}
		if msg.RFCMessageID != "" {
			haveRFC[strings.TrimSpace(msg.RFCMessageID)] = true
		}
		added++
	}
	return added, nil
}

func (s *LocalStore) ensureLocalSpecialsLocked(accountID string) {
	for _, spec := range defaultSpecials() {
		id := FolderID(accountID + "/" + strings.ToLower(spec.Name))
		if spec.Kind == FolderInbox {
			id = FolderID(accountID + "/inbox")
		}
		if _, ok := s.folderLocked(id); ok {
			continue
		}
		s.folders = append(s.folders, Folder{
			ID: id, AccountID: accountID, Name: spec.Name, Kind: spec.Kind, Remote: spec.Remote,
		})
	}
}

func displayIMAPName(name, delim string) string {
	if strings.EqualFold(name, "INBOX") {
		return "Inbox"
	}
	if delim != "" && strings.Contains(name, delim) {
		parts := strings.Split(name, delim)
		return parts[len(parts)-1]
	}
	return name
}

func (s *LocalStore) replaceFolderLocked(f Folder) {
	for i, x := range s.folders {
		if x.ID == f.ID {
			s.folders[i] = f
			return
		}
	}
	s.folders = append(s.folders, f)
}

func (s *LocalStore) syncFolderLocked(cli *imapClient, f Folder) (int, error) {
	remote := f.Remote
	if remote == "" {
		remote = f.Name
	}
	st, vanished, err := cli.selectSync(remote, true, s.loadFolderMeta(f.ID))
	if err != nil {
		return 0, err
	}
	_ = vanished
	meta := s.loadFolderMeta(f.ID)
	if meta.UIDValidity != 0 && st.UIDValidity != 0 && meta.UIDValidity != st.UIDValidity {
		s.dropFolderMessagesLocked(f.ID)
		meta = folderMeta{}
	}
	from := uint32(1)
	if meta.UIDNext > 1 {
		from = meta.UIDNext
	}
	list, err := cli.uidFetchMeta(from)
	if err != nil {
		return 0, err
	}
	added := 0
	maxUID := meta.UIDNext
	for _, im := range list {
		if im.UID == 0 {
			continue
		}
		if im.UID >= maxUID {
			maxUID = im.UID + 1
		}
		id := MessageID(fmt.Sprintf("%s:%d", f.ID, im.UID))
		if _, ok := s.indexLocked(id); ok {
			continue
		}
		m := Message{
			ID: id, Folder: f.ID, AccountID: f.AccountID,
			From: im.From, To: im.To, Cc: im.Cc, Subject: im.Subject,
			Date: im.Date, Size: im.Size, UID: im.UID,
			Read: imapFlagSeen(im.Flags), Starred: imapFlagStar(im.Flags),
			Tags: imapKeywords(im.Flags), Parts: im.Parts,
			RFCMessageID: im.RFCMessageID, InReplyTo: im.InReplyTo,
		}
		m.ThreadID = ThreadIDOf(m)
		if s.feat != nil {
			m.Category = messageCategory(m, s.feat.snap())
		}
		for _, p := range im.Parts {
			if p.Filename != "" {
				m.HasAttach = true
				m.Attachments = append(m.Attachments, p.Filename)
			}
		}
		s.messages = append(s.messages, m)
		s.applyRulesOnLocked(&s.messages[len(s.messages)-1])
		added++
	}
	if vanished, err := cli.uidVanished(meta); err == nil {
		for _, uid := range vanished {
			id := MessageID(fmt.Sprintf("%s:%d", f.ID, uid))
			if s.feat != nil && s.feat.pendingFor(id) {
				continue
			}
			if i, ok := s.indexLocked(id); ok {
				s.messages = append(s.messages[:i], s.messages[i+1:]...)
			}
		}
	}
	if flags, err := cli.uidFetchFlags(1, meta.HighestMod); err == nil {
		for _, im := range flags {
			id := MessageID(fmt.Sprintf("%s:%d", f.ID, im.UID))
			if s.feat != nil && s.feat.pendingFor(id) {
				continue
			}
			if i, ok := s.indexLocked(id); ok {
				s.messages[i].Read = imapFlagSeen(im.Flags)
				s.messages[i].Starred = imapFlagStar(im.Flags)
			}
		}
	}
	s.saveFolderMeta(f.ID, folderMeta{
		UIDValidity: st.UIDValidity, UIDNext: maxUID, HighestMod: st.HighestMod, Remote: remote,
	})
	return added, nil
}

func (s *LocalStore) dropFolderMessagesLocked(id FolderID) {
	out := s.messages[:0]
	for _, m := range s.messages {
		if m.Folder != id {
			out = append(out, m)
		}
	}
	s.messages = out
}

func (s *LocalStore) Unread(folder FolderID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.messages {
		if m.Read {
			continue
		}
		f, ok := s.folderLocked(m.Folder)
		kind := FolderCustom
		if ok {
			kind = f.Kind
		}
		if IsVirtual(folder) {
			if matchVirtual(folder, m, f, kind, s.feat.snap()) {
				n++
			}
			continue
		}
		if m.Folder == folder {
			n++
		}
	}
	return n
}

func (s *LocalStore) UnreadTotal() int {
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

func (s *LocalStore) MessageCount(folder FolderID) int {
	return len(s.ListMessages(folder))
}

func (s *LocalStore) Identities(accountID string) []Identity {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Identity
	for _, id := range s.identities {
		if accountID == "" || id.AccountID == accountID {
			out = append(out, id)
		}
	}
	return out
}

func (s *LocalStore) PutIdentity(id Identity) (Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id.ID == "" {
		id.ID = slug(id.Address)
	}
	s.identities = upsertIdentity(s.identities, id)
	s.saveLocked()
	return id, nil
}

func (s *LocalStore) DeleteIdentity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.identities[:0]
	for _, x := range s.identities {
		if x.ID != id {
			out = append(out, x)
		}
	}
	s.identities = out
	s.saveLocked()
	return nil
}

func (s *LocalStore) ListTags() []Tag {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneTags(s.tags)
}

func (s *LocalStore) PutTag(t Tag) (Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags = upsertTag(s.tags, t)
	s.saveLocked()
	return t, nil
}

func (s *LocalStore) VirtualFolders() []Folder {
	s.mu.Lock()
	defer s.mu.Unlock()
	base := []Folder{
		{ID: FolderUnifiedInbox, AccountID: AccountUnified, Name: "Unified Inbox", Kind: FolderInbox, Virtual: true, MatchKind: FolderInbox},
		{ID: FolderUnifiedUnread, AccountID: AccountUnified, Name: "Unread", Kind: FolderCustom, Virtual: true},
		{ID: FolderUnifiedStarred, AccountID: AccountUnified, Name: "Starred", Kind: FolderCustom, Virtual: true},
	}
	out := append(base, extraVirtualFolders(s.feat.snap())...)
	for _, t := range s.tags {
		out = append(out, Folder{
			ID: TagFolderID(t.Name), AccountID: AccountTags, Name: t.Name,
			Kind: FolderCustom, Virtual: true, Tag: t.Name,
		})
	}
	return out
}

func (s *LocalStore) ListRules() []FilterRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneRules(s.rules)
}

func (s *LocalStore) PutRule(r FilterRule) (FilterRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		r.ID = nextRuleID(s.rules)
	}
	found := false
	for i, x := range s.rules {
		if x.ID == r.ID {
			s.rules[i] = r
			found = true
			break
		}
	}
	if !found {
		s.rules = append(s.rules, r)
	}
	s.saveLocked()
	return r, nil
}

func (s *LocalStore) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.rules[:0]
	for _, r := range s.rules {
		if r.ID != id {
			out = append(out, r)
		}
	}
	s.rules = out
	s.saveLocked()
	return nil
}

func (s *LocalStore) ApplyRules(folder FolderID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for i := range s.messages {
		if folder != "" && s.messages[i].Folder != folder && !IsVirtual(folder) {
			continue
		}
		if s.applyRulesOnLocked(&s.messages[i]) {
			n++
		}
	}
	s.saveLocked()
	return n, nil
}

func (s *LocalStore) applyRulesOnLocked(m *Message) bool {
	changed := false
	for _, r := range s.rules {
		if !r.match(*m) {
			continue
		}
		stop, err := applyRuleActions(s, m, r.Actions)
		if err == nil {
			changed = true
		}
		if stop || r.Stop {
			break
		}
	}
	return changed
}

func (s *LocalStore) deleteOne(id MessageID) error {
	i, ok := s.indexLocked(id)
	if !ok {
		return fmt.Errorf("mail: no message %s", id)
	}
	s.expungeLocked(s.messages[i])
	s.messages = append(s.messages[:i], s.messages[i+1:]...)
	return nil
}

func (s *LocalStore) moveOne(id MessageID, dest FolderID) error {
	i, ok := s.indexLocked(id)
	if !ok {
		return fmt.Errorf("mail: no message %s", id)
	}
	s.messages[i].Folder = dest
	return nil
}

func (s *LocalStore) indexOf(id MessageID) (int, bool) { return s.indexLocked(id) }
func (s *LocalStore) messageAt(i int) *Message         { return &s.messages[i] }

func (s *LocalStore) GetPart(id MessageID, partID string) (PartData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.indexLocked(id)
	if !ok {
		return PartData{}, fmt.Errorf("mail: no message %s", id)
	}
	m := s.messages[i]
	raw := s.readRawLocked(m)
	if len(raw) == 0 {
		s.fetchBodyLocked(&s.messages[i])
		m = s.messages[i]
		raw = s.readRawLocked(m)
	}
	if len(raw) == 0 && m.UID != 0 {
		if cli, err := s.clientLocked(m.AccountID); err == nil {
			if f, ok := s.folderLocked(m.Folder); ok {
				remote := f.Remote
				if remote == "" {
					remote = f.Name
				}
				if _, err := cli.selectBox(remote, true); err == nil {
					if b, err := cli.uidFetchSection(m.UID, partID); err == nil {
						p := partByID(m.Parts, partID)
						p.Data = b
						return p, nil
					}
				}
			}
		}
	}
	if len(raw) == 0 {
		return PartData{}, fmt.Errorf("mail: no body for %s", id)
	}
	parsed, err := ParseRFC822(raw, m.Folder, m.AccountID)
	if err != nil {
		return PartData{}, err
	}
	if partID == "" || partID == "1" && len(parsed.Parts) == 0 {
		return PartData{Part: Part{ID: "1", MIMEType: "text/plain", Size: len(parsed.Body)}, Data: []byte(parsed.Body)}, nil
	}
	// Prefer cached raw section via IMAP; otherwise return text body for text parts.
	for _, p := range parsed.Parts {
		if p.ID == partID {
			if strings.HasPrefix(strings.ToLower(p.MIMEType), "text/html") {
				return PartData{Part: p, Data: []byte(parsed.HTML)}, nil
			}
			if strings.HasPrefix(strings.ToLower(p.MIMEType), "text/") {
				return PartData{Part: p, Data: []byte(parsed.Body)}, nil
			}
			return PartData{Part: p}, nil
		}
	}
	return PartData{}, fmt.Errorf("mail: no part %s", partID)
}

func partByID(parts []Part, id string) PartData {
	for _, p := range parts {
		if p.ID == id {
			return PartData{Part: p}
		}
	}
	return PartData{Part: Part{ID: id}}
}

func (s *LocalStore) OpenPart(id MessageID, partID string) (PartData, error) {
	p, err := s.GetPart(id, partID)
	if err != nil {
		return p, err
	}
	name := p.Filename
	if name == "" {
		name = string(id) + "-" + partID
	}
	name = filepath.Base(name)
	dir := filepath.Join(s.dir, "open")
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, name)
	if len(p.Data) > 0 {
		if err := os.WriteFile(path, p.Data, 0o600); err != nil {
			return p, err
		}
	}
	p.Path = path
	openCachedFile(path)
	return p, nil
}

func (s *LocalStore) UnreadAll() int { return s.UnreadTotal() }

func (s *LocalStore) indexLocked(id MessageID) (int, bool) {
	for i, m := range s.messages {
		if m.ID == id {
			return i, true
		}
	}
	return -1, false
}

func (s *LocalStore) specialLocked(accountID string, kind FolderKind) (Folder, bool) {
	for _, f := range s.folders {
		if f.AccountID == accountID && f.Kind == kind && !f.Virtual {
			return f, true
		}
	}
	return Folder{}, false
}

func (s *LocalStore) defaultIdentLocked(accountID string) Identity {
	var first Identity
	for _, id := range s.identities {
		if id.AccountID != accountID {
			continue
		}
		if first.ID == "" {
			first = id
		}
		if id.Default {
			return id
		}
	}
	return first
}

func (s *LocalStore) clientLocked(accountID string) (*imapClient, error) {
	if c := s.clients[accountID]; c != nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
		return c, nil
	}
	var cfg AccountConfig
	for _, a := range s.cfg.Accounts {
		id := a.ID
		if id == "" {
			id = slug(a.Address)
		}
		if id == accountID {
			cfg = a
			break
		}
	}
	if cfg.IsPOP3() {
		return nil, fmt.Errorf("mail: account %s is POP3 (no IMAP session)", accountID)
	}
	if cfg.IMAP.Host == "" {
		return nil, fmt.Errorf("mail: no IMAP host for %s", accountID)
	}
	cfg.IMAP.tokenKey = accountID
	c := newIMAPClient(cfg.IMAP, cfg.Address)
	if err := c.connect(); err != nil {
		s.health = err
		return nil, err
	}
	s.clients[accountID] = c
	return c, nil
}

func (s *LocalStore) accountCfg(accountID string) (AccountConfig, bool) {
	for _, a := range s.cfg.Accounts {
		id := a.ID
		if id == "" {
			id = slug(a.Address)
		}
		if id == accountID {
			return a, true
		}
	}
	return AccountConfig{}, false
}

func (s *LocalStore) loadLocked() {
	_ = readJSONFile(filepath.Join(s.dir, "accounts.json"), &s.accounts)
	_ = readJSONFile(filepath.Join(s.dir, "identities.json"), &s.identities)
	_ = readJSONFile(filepath.Join(s.dir, "folders.json"), &s.folders)
	_ = readJSONFile(filepath.Join(s.dir, "messages.json"), &s.messages)
	_ = readJSONFile(filepath.Join(s.dir, "tags.json"), &s.tags)
	_ = readJSONFile(filepath.Join(s.dir, "rules.json"), &s.rules)
	if s.feat == nil {
		s.feat = newFeatureHost()
	}
	_ = readJSONFile(filepath.Join(s.dir, "smart.json"), &s.feat.smart)
	_ = readJSONFile(filepath.Join(s.dir, "vip.json"), &s.feat.vips)
	_ = readJSONFile(filepath.Join(s.dir, "muted.json"), &s.feat.muted)
	_ = readJSONFile(filepath.Join(s.dir, "categories.json"), &s.feat.cats)
	_ = readJSONFile(filepath.Join(s.dir, "notify.json"), &s.feat.notify)
	_ = readJSONFile(filepath.Join(s.dir, "outbox.json"), &s.feat.outbox)
	if len(s.tags) == 0 {
		s.tags = DefaultTags()
	}
	assignThreadIDs(s.messages)
	if s.feat.index != nil {
		s.feat.index.rebuild(s.messages)
	}
	for _, m := range s.messages {
		if n := idSeq(m.ID); n >= s.nextID {
			s.nextID = n + 1
		}
	}
}

func (s *LocalStore) saveLocked() {
	_ = writeJSONFile(filepath.Join(s.dir, "accounts.json"), s.accounts)
	_ = writeJSONFile(filepath.Join(s.dir, "identities.json"), s.identities)
	_ = writeJSONFile(filepath.Join(s.dir, "folders.json"), s.folders)
	_ = writeJSONFile(filepath.Join(s.dir, "messages.json"), s.messages)
	_ = writeJSONFile(filepath.Join(s.dir, "tags.json"), s.tags)
	_ = writeJSONFile(filepath.Join(s.dir, "rules.json"), s.rules)
	if s.feat != nil {
		_ = writeJSONFile(filepath.Join(s.dir, "smart.json"), s.feat.smart)
		_ = writeJSONFile(filepath.Join(s.dir, "vip.json"), s.feat.vips)
		_ = writeJSONFile(filepath.Join(s.dir, "muted.json"), s.feat.muted)
		_ = writeJSONFile(filepath.Join(s.dir, "categories.json"), s.feat.cats)
		_ = writeJSONFile(filepath.Join(s.dir, "notify.json"), s.feat.notify)
		_ = writeJSONFile(filepath.Join(s.dir, "outbox.json"), s.feat.outbox)
	}
}

func (s *LocalStore) rawPath(m Message) string {
	return filepath.Join(s.dir, "raw", string(m.AccountID), fmt.Sprintf("%s.eml", m.ID))
}

func (s *LocalStore) writeRawLocked(m Message, raw []byte) {
	p := s.rawPath(m)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	_ = os.WriteFile(p, raw, 0o600)
}

func (s *LocalStore) readRawLocked(m Message) []byte {
	b, err := os.ReadFile(s.rawPath(m))
	if err != nil {
		return nil
	}
	return b
}

func (s *LocalStore) loadFolderMeta(id FolderID) folderMeta {
	var m folderMeta
	_ = readJSONFile(filepath.Join(s.dir, "meta", slug(string(id))+".json"), &m)
	return m
}

func (s *LocalStore) saveFolderMeta(id FolderID, m folderMeta) {
	dir := filepath.Join(s.dir, "meta")
	_ = os.MkdirAll(dir, 0o700)
	_ = writeJSONFile(filepath.Join(dir, slug(string(id))+".json"), m)
}

func idSeq(id MessageID) int {
	s := string(id)
	i := strings.LastIndex(s, "-")
	if i < 0 {
		return 0
	}
	n := 0
	for _, c := range s[i+1:] {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func readJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// SendViaSMTP builds RFC822 and submits, then APPENDs to Sent.
func (s *LocalStore) SendViaSMTP(accountID, identityID string, msg Message, files []AttachedFile) (MessageID, error) {
	s.mu.Lock()
	cfg, ok := s.accountCfg(accountID)
	ident := s.defaultIdentLocked(accountID)
	for _, id := range s.identities {
		if id.ID == identityID {
			ident = id
			if id.AccountID != "" {
				accountID = id.AccountID
				if c, ok2 := s.accountCfg(accountID); ok2 {
					cfg = c
					ok = true
				}
			}
		}
	}
	s.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("mail: no SMTP account %s", accountID)
	}
	if msg.From == "" {
		msg.From = ident.DisplayFrom()
	}
	raw := BuildRFC822(msg, ident, files)
	rcpts := append(splitAddrs(msg.To), splitAddrs(msg.Cc)...)
	rcpts = append(rcpts, splitAddrs(msg.Bcc)...)
	cfg.SMTP.tokenKey = accountID
	if s.feat != nil && !s.feat.Online() {
		id, err := s.Append(FolderOutbox, msg)
		if err != nil {
			// no physical outbox folder — keep the draft in memory list via a queued send
			s.mu.Lock()
			s.nextID++
			id = MessageID(fmt.Sprintf("%s-out-%04d", accountID, s.nextID))
			msg.ID = id
			msg.AccountID = accountID
			s.messages = append(s.messages, msg)
			s.mu.Unlock()
		}
		s.feat.mu.Lock()
		cp := msg.Clone()
		s.feat.enqueueLocked(OutboxOp{Kind: "send", AccountID: accountID, MessageID: id, Message: &cp})
		s.feat.mu.Unlock()
		s.mu.Lock()
		s.saveLocked()
		s.mu.Unlock()
		return id, nil
	}
	if err := SendSMTP(cfg.SMTP, extractAddr(msg.From), rcpts, raw); err != nil {
		s.feat.mu.Lock()
		cp := msg.Clone()
		s.feat.enqueueLocked(OutboxOp{Kind: "send", AccountID: accountID, Message: &cp, Error: err.Error()})
		s.feat.mu.Unlock()
		return "", err
	}
	sent, ok := specialFolder(s, accountID, FolderSent)
	if !ok {
		return "", nil
	}
	return s.Append(sent.ID, msg)
}

func (s *LocalStore) extras() *featureHost {
	if s.feat == nil {
		s.feat = newFeatureHost()
	}
	return s.feat
}

func (s *LocalStore) SetOnline(v bool)       { s.extras().SetOnline(v) }
func (s *LocalStore) Online() bool           { return s.extras().Online() }
func (s *LocalStore) ListOutbox() []OutboxOp { return s.extras().ListOutbox() }
func (s *LocalStore) ListSmartFolders() []SmartFolder {
	return s.extras().ListSmartFolders()
}
func (s *LocalStore) PutSmartFolder(sf SmartFolder) (SmartFolder, error) {
	out, err := s.extras().PutSmartFolder(sf)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return out, err
}
func (s *LocalStore) DeleteSmartFolder(id string) error {
	err := s.extras().DeleteSmartFolder(id)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return err
}
func (s *LocalStore) MuteThread(id string, muted bool) error {
	err := s.extras().MuteThread(id, muted)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return err
}
func (s *LocalStore) MutedThreads() []string { return s.extras().MutedThreads() }
func (s *LocalStore) ListVIPs() []VIP        { return s.extras().ListVIPs() }
func (s *LocalStore) PutVIP(v VIP) (VIP, error) {
	out, err := s.extras().PutVIP(v)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return out, err
}
func (s *LocalStore) DeleteVIP(address string) error {
	err := s.extras().DeleteVIP(address)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return err
}
func (s *LocalStore) NotifyPrefs() NotifyPrefs { return s.extras().NotifyPrefs() }
func (s *LocalStore) PutNotifyPrefs(p NotifyPrefs) NotifyPrefs {
	out := s.extras().PutNotifyPrefs(p)
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return out
}
func (s *LocalStore) SetSenderCategory(address, category string) error {
	err := s.extras().SetSenderCategory(address, category)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages {
		if canonAddr(s.messages[i].From) == canonAddr(address) {
			s.messages[i].Category = category
		}
	}
	s.saveLocked()
	return nil
}
func (s *LocalStore) ListSenderCategories() []SenderCat {
	return s.extras().ListSenderCategories()
}

func (s *LocalStore) FlushOutbox() (int, error) {
	if s.feat == nil {
		return 0, nil
	}
	s.feat.mu.Lock()
	ops := append([]OutboxOp(nil), s.feat.outbox...)
	s.feat.mu.Unlock()
	flushed := 0
	var last error
	for _, op := range ops {
		if err := s.flushOne(op); err != nil {
			last = err
			s.feat.mu.Lock()
			for i := range s.feat.outbox {
				if s.feat.outbox[i].ID == op.ID {
					s.feat.outbox[i].Error = err.Error()
					s.feat.outbox[i].Tries++
				}
			}
			s.feat.mu.Unlock()
			continue
		}
		s.feat.mu.Lock()
		s.feat.dropOutboxLocked(op.ID)
		s.feat.mu.Unlock()
		flushed++
	}
	s.mu.Lock()
	s.saveLocked()
	s.mu.Unlock()
	return flushed, last
}

func (s *LocalStore) flushOne(op OutboxOp) error {
	switch op.Kind {
	case "flag":
		s.mu.Lock()
		defer s.mu.Unlock()
		i, ok := s.indexLocked(op.MessageID)
		if !ok {
			return nil // vanished; drop
		}
		s.pushFlagsLocked(s.messages[i], nil, nil)
		var add, rem []string
		if op.Patch.Read != nil {
			if *op.Patch.Read {
				add = append(add, `\Seen`)
			} else {
				rem = append(rem, `\Seen`)
			}
		}
		if op.Patch.Starred != nil {
			if *op.Patch.Starred {
				add = append(add, `\Flagged`)
			} else {
				rem = append(rem, `\Flagged`)
			}
		}
		s.pushFlagsLocked(s.messages[i], add, rem)
		return nil
	case "move":
		return s.Move([]MessageID{op.MessageID}, op.Dest)
	case "delete":
		return s.Delete([]MessageID{op.MessageID})
	case "send":
		if op.Message == nil {
			return nil
		}
		_, err := s.SendViaSMTP(op.AccountID, "", *op.Message, nil)
		return err
	default:
		return nil
	}
}
