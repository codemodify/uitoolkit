package mail

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/style"
)

func TestUnifiedInboxAggregates(t *testing.T) {
	s := NewDemoStore()
	ada := s.ListMessages(FolderAdaInbox)
	work := s.ListMessages(FolderWorkInbox)
	uni := s.ListMessages(FolderUnifiedInbox)
	if len(uni) != len(ada)+len(work) {
		t.Fatalf("unified %d want %d+%d", len(uni), len(ada), len(work))
	}
	unread := s.ListMessages(FolderUnifiedUnread)
	if len(unread) == 0 {
		t.Fatal("unified unread")
	}
	star := s.ListMessages(FolderUnifiedStarred)
	if len(star) == 0 {
		t.Fatal("unified starred")
	}
	imp := s.ListMessages(TagFolderID("Important"))
	if len(imp) == 0 {
		t.Fatal("tag Important")
	}
	f, ok := s.GetFolder(FolderUnifiedInbox)
	if !ok || !f.Virtual {
		t.Fatal("virtual get")
	}
	if s.Unread(FolderUnifiedInbox) < 1 {
		t.Fatal("unified unread count")
	}
}

func TestIdentitiesAndTags(t *testing.T) {
	s := NewDemoStore()
	ids := s.Identities("")
	if len(ids) < 3 {
		t.Fatalf("idents %d", len(ids))
	}
	ada := s.Identities(AcctAda)
	if len(ada) < 2 {
		t.Fatal("ada identities")
	}
	tags := s.ListTags()
	if len(tags) < 5 {
		t.Fatal(tags)
	}
	if _, err := s.PutTag(Tag{Name: "Later", Color: "#111111"}); err != nil {
		t.Fatal(err)
	}
	vfs := s.VirtualFolders()
	if len(vfs) < 4 {
		t.Fatalf("virtual %d", len(vfs))
	}
}

func TestFilterRulesApply(t *testing.T) {
	s := NewDemoStore()
	n, err := s.ApplyRules("")
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatal("expected some rule hits")
	}
	found := false
	for _, m := range s.ListMessages(FolderAdaInbox) {
		if strings.Contains(m.Subject, "Invoice") && hasTag(m.Tags, "Work") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("invoice not tagged Work")
	}
}

func TestRPCNewMethods(t *testing.T) {
	sock, stop, err := StartDemo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	cli, err := DialWait(sock, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	ids, err := cli.Identities("")
	if err != nil || len(ids) < 2 {
		t.Fatalf("idents %v %v", ids, err)
	}
	tags, err := cli.Tags()
	if err != nil || len(tags) < 3 {
		t.Fatalf("tags %v %v", tags, err)
	}
	vf, err := cli.VirtualFolders()
	if err != nil || len(vf) < 3 {
		t.Fatalf("virtual %v %v", vf, err)
	}
	uni, err := cli.ListMessages(FolderUnifiedInbox, Filter{})
	if err != nil || len(uni) < 10 {
		t.Fatalf("unified list %d %v", len(uni), err)
	}
	rules, err := cli.Rules()
	if err != nil || len(rules) < 1 {
		t.Fatalf("rules %v %v", rules, err)
	}
	n, err := cli.ApplyRules("")
	if err != nil {
		t.Fatal(err)
	}
	if n < 0 {
		t.Fatal(n)
	}
	part, err := cli.GetPart(uni[0].ID, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(part.Data) == 0 && part.Path == "" {
		t.Log("empty part ok for header-only")
	}
	res, err := cli.Sync(AcctAda)
	if err != nil {
		t.Fatal(err)
	}
	if res.New < 0 {
		t.Fatal(res)
	}
}

func TestMIMEParseAndSanitize(t *testing.T) {
	raw := []byte("From: Ada <ada@example.com>\r\n" +
		"To: Kai <kai@example.com>\r\n" +
		"Subject: =?UTF-8?Q?Caf=C3=A9?=\r\n" +
		"Date: Fri, 11 Sep 2026 17:00:00 +0000\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		"hello body\r\n")
	m, err := ParseRFC822(raw, FolderAdaInbox, AcctAda)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Subject, "Caf") {
		t.Fatalf("subject %q", m.Subject)
	}
	if !strings.Contains(m.Body, "hello") {
		t.Fatalf("body %q", m.Body)
	}
	html := `<p>Hi</p><script>alert(1)</script><a href="x" onclick="evil()">link</a>`
	if strings.Contains(SanitizeHTML(html), "script") || strings.Contains(SanitizeHTML(html), "onclick") {
		t.Fatalf("sanitize %q", SanitizeHTML(html))
	}
	if !strings.Contains(HTMLToText(html), "Hi") {
		t.Fatal(HTMLToText(html))
	}
	built := BuildRFC822(Message{To: "kai@example.com", Subject: "x", Body: "y"}, Identity{Name: "Ada", Address: "ada@example.com", Signature: "sig"}, nil)
	if !strings.Contains(string(built), "sig") {
		t.Fatal(string(built))
	}
}

func TestLocalStoreDiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := MailConfig{Accounts: []AccountConfig{{
		ID: "home", Name: "Ada", Address: "ada@example.com",
		Identities: []Identity{{ID: "home-1", Name: "Ada", Address: "ada@example.com", Default: true}},
	}}}
	s, err := NewLocalStoreDir(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts()) != 1 {
		t.Fatal(s.Accounts())
	}
	if len(s.Identities("")) != 1 {
		t.Fatal(s.Identities(""))
	}
	_, err = s.PutTag(Tag{Name: "Work", Color: "#d35400"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewLocalStoreDir(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.ListTags()) < 1 {
		t.Fatal("tags persist")
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv(EnvHost, "imap.example.com:993")
	t.Setenv(EnvUser, "you@example.com")
	t.Setenv(EnvName, "You")
	cfg, err := ConfigFromEnv()
	if err != nil || len(cfg.Accounts) != 1 {
		t.Fatalf("%+v %v", cfg, err)
	}
	if cfg.Accounts[0].IMAP.Host == "" || cfg.Accounts[0].SMTP.Host == "" {
		t.Fatal(cfg.Accounts[0])
	}
}

func TestChromePrefsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	saveChromePrefs(ChromePrefs{CardView: true, Density: "compact"})
	p := loadChromePrefs()
	if !p.CardView || p.density() != style.DensityCompact {
		t.Fatalf("%+v", p)
	}
	if _, err := os.Stat(filepath.Join(dir, "uitoolkit", "mailui.json")); err != nil {
		t.Fatal(err)
	}
}
