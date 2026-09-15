//go:build linux

package platform

import (
	"bufio"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// fakeChooser answers the portal's OpenFile the way xdg-desktop-portal
// does: a Request path from the caller's name and handle_token, then a
// Response signal on it.
type fakeChooser struct {
	conn    *dbus.Conn
	mu      sync.Mutex
	options map[string]dbus.Variant
	title   string
}

func (f *fakeChooser) seen() (map[string]dbus.Variant, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.options, f.title
}

func (f *fakeChooser) OpenFile(sender dbus.Sender, parent, title string, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f.mu.Lock()
	f.options, f.title = options, title
	f.mu.Unlock()
	token, _ := options["handle_token"].Value().(string)
	path := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" +
		strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_") + "/" + token)
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = f.conn.Emit(path, portalRequest+".Response", uint32(0), map[string]dbus.Variant{
			"uris": dbus.MakeVariant([]string{"file:///tmp/a%20b.txt", "file:///home/ada/notes.md"}),
		})
	}()
	return path, nil
}

func privateBus(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command(bin, "--session", "--nofork", "--nopidfile", "--print-address=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skip("dbus-daemon:", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(line)
}

func TestFileChooserThroughPortal(t *testing.T) {
	addr := privateBus(t)
	portal, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer portal.Close()
	const name = "org.freedesktop.portal.Desktop.Test"
	if r, err := portal.RequestName(name, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Fatal("name", err)
	}
	fake := &fakeChooser{conn: portal}
	if err := portal.Export(fake, portalPath, portalFileChooser); err != nil {
		t.Fatal(err)
	}
	client, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	got := make(chan []string, 1)
	ok := openFileChooser(client, name, FileChooserOptions{
		Title: "Attach file", Multiple: true, Folder: "/home/ada",
		Filters: []FileFilter{{Name: "Text", Patterns: []string{"*.txt", "*.md"}}},
	}, func(paths []string) { got <- paths })
	if !ok {
		t.Fatal("the portal call failed")
	}
	select {
	case paths := <-got:
		if len(paths) != 2 || paths[0] != "/tmp/a b.txt" || paths[1] != "/home/ada/notes.md" {
			t.Fatalf("paths %q", paths)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no answer")
	}
	options, title := fake.seen()
	if title != "Attach file" {
		t.Fatalf("title %q", title)
	}
	if m, _ := options["multiple"].Value().(bool); !m {
		t.Fatal("multiple not asked for")
	}
	if b, _ := options["current_folder"].Value().([]byte); string(b) != "/home/ada\x00" {
		t.Fatalf("current_folder %q", b)
	}
	if _, ok := options["filters"]; !ok {
		t.Fatal("filters not passed")
	}
}
