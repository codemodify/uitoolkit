//go:build linux

package platform

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/godbus/dbus/v5"
)

// quietBus starts a private D-Bus daemon that can start nothing: no service
// directories, so a name nobody owns is simply absent — what a desktop
// without the portal looks like — and no test can ever start a real portal
// or notification server. The session bus of the desktop running the tests
// is never touched.
func quietBus(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon")
	}
	dir := t.TempDir()
	conf := filepath.Join(dir, "bus.conf")
	err = os.WriteFile(conf, []byte(`<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=`+dir+`</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "--config-file="+conf, "--nofork", "--nopidfile", "--print-address=1")
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

// busPair connects a service and a client to a quiet bus.
func busPair(t *testing.T) (service, client *dbus.Conn) {
	t.Helper()
	addr := quietBus(t)
	var err error
	if service, err = dbus.Connect(addr); err != nil {
		t.Fatal(err)
	}
	if client, err = dbus.Connect(addr); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close(); service.Close() })
	return service, client
}

func own(t *testing.T, c *dbus.Conn, name string) {
	t.Helper()
	if r, err := c.RequestName(name, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Fatal("name", name, err)
	}
}

// fakeNotifyPortal is org.freedesktop.portal.Notification.
type fakeNotifyPortal struct {
	mu      sync.Mutex
	added   []string
	last    map[string]dbus.Variant
	removed []string
}

func (f *fakeNotifyPortal) AddNotification(id string, n map[string]dbus.Variant) *dbus.Error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.added = append(f.added, id)
	f.last = n
	return nil
}

func (f *fakeNotifyPortal) RemoveNotification(id string) *dbus.Error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, id)
	return nil
}

// fakeNotifyServer is org.freedesktop.Notifications.
type fakeNotifyServer struct {
	mu       sync.Mutex
	next     uint32
	app      string
	replaces []uint32
	icon     string
	summary  string
	body     string
	actions  []string
	hints    map[string]dbus.Variant
	closed   []uint32
}

func (f *fakeNotifyServer) Notify(app string, replaces uint32, icon, summary, body string, actions []string, hints map[string]dbus.Variant, timeout int32) (uint32, *dbus.Error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.app, f.icon, f.summary, f.body, f.actions, f.hints = app, icon, summary, body, actions, hints
	f.replaces = append(f.replaces, replaces)
	if replaces != 0 {
		return replaces, nil
	}
	f.next++
	return f.next + 40, nil
}

func (f *fakeNotifyServer) CloseNotification(id uint32) *dbus.Error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, id)
	return nil
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(3 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		if ok() {
			return
		}
	}
	t.Fatal("timed out waiting for " + what)
}

func testNote(clicks chan string) DesktopNotification {
	return DesktopNotification{
		ID: "new-mail", Title: "Ada Lovelace", Body: "Notes on the engine",
		IconName: "mail-unread", Priority: PriorityHigh,
		Actions:    []NotificationAction{{ID: "open", Label: "Open Mail"}, {ID: "mark-read", Label: "Mark Read"}},
		OnActivate: func(a string) { clicks <- a },
	}
}

// Through the portal: AddNotification with the notification's id and the
// a{sv} the portal documents — title, body, a themed icon, priority, the
// default action and the buttons — and ActionInvoked back into the app,
// as the action the app named.
func TestNotifierThroughThePortal(t *testing.T) {
	t.Setenv(EnvNotify, "")
	svc, client := busPair(t)
	own(t, svc, portalDest)
	fake := &fakeNotifyPortal{}
	if err := svc.Export(fake, portalPath, portalNotification); err != nil {
		t.Fatal(err)
	}
	clicks := make(chan string, 4)
	n := newNotifier(NotifierOptions{AppName: "Mail"}, func() (*dbus.Conn, error) { return client, nil }, portalDest, fdoNotify)
	defer n.Close()
	id, err := n.Send(testNote(clicks))
	if err != nil || id != "new-mail" {
		t.Fatalf("send: %q %v", id, err)
	}
	if n.Backend() != "portal" {
		t.Fatalf("backend %q: no server is running, so the portal", n.Backend())
	}
	fake.mu.Lock()
	v := fake.last
	fake.mu.Unlock()
	if s, _ := v["title"].Value().(string); s != "Ada Lovelace" {
		t.Fatalf("title %v", v["title"])
	}
	if s, _ := v["body"].Value().(string); s != "Notes on the engine" {
		t.Fatalf("body %v", v["body"])
	}
	if s, _ := v["priority"].Value().(string); s != "high" {
		t.Fatalf("priority %v", v["priority"])
	}
	if s, _ := v["default-action"].Value().(string); s != "default" {
		t.Fatalf("default-action %v", v["default-action"])
	}
	// A struct decodes as its fields: (sv) is a string and a variant.
	// (godbus hands an exported method re-made variants, so their
	// signatures are the Go values' and not the wire's; the types are.)
	icon, _ := v["icon"].Value().([]any)
	if len(icon) != 2 || icon[0] != "themed" {
		t.Fatalf("icon %#v", v["icon"].Value())
	}
	if names, _ := icon[1].(dbus.Variant).Value().([]string); len(names) != 1 || names[0] != "mail-unread" {
		t.Fatalf("icon names %v", icon[1])
	}
	buttons, _ := v["buttons"].Value().([]map[string]dbus.Variant)
	if len(buttons) != 2 || buttons[0]["label"].Value() != "Open Mail" || buttons[1]["action"].Value() != "button-1" {
		t.Fatalf("buttons %v", buttons)
	}
	// Someone else's click is not ours; ours comes back as the app's action.
	_ = svc.Emit(portalPath, portalNotification+".ActionInvoked", "someone-else", "default", []dbus.Variant{})
	_ = svc.Emit(portalPath, portalNotification+".ActionInvoked", "new-mail", "button-1", []dbus.Variant{})
	select {
	case a := <-clicks:
		if a != "mark-read" {
			t.Fatalf("action %q, want mark-read", a)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the click never came back")
	}
	// Withdraw after a click is nothing (the portal took it down); a fresh
	// one is removed by id.
	if _, err := n.Send(DesktopNotification{ID: "second", Title: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := n.Withdraw("second"); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	removed := append([]string(nil), fake.removed...)
	fake.mu.Unlock()
	if len(removed) != 1 || removed[0] != "second" {
		t.Fatalf("removed %v", removed)
	}
}

// Through the notification server: Notify with the app's name, the icon
// name, the actions as key/label pairs with "default" first, the urgency
// and desktop-entry hints; the same ID again replaces it; ActionInvoked
// comes back as the app's action; pixels go as image-data.
func TestNotifierThroughTheServer(t *testing.T) {
	t.Setenv(EnvNotify, "")
	svc, client := busPair(t)
	own(t, svc, fdoNotify)
	own(t, svc, portalDest) // a portal too: outside a sandbox the server wins
	fake := &fakeNotifyServer{}
	if err := svc.Export(fake, fdoNotifyPath, fdoNotify); err != nil {
		t.Fatal(err)
	}
	portal := &fakeNotifyPortal{}
	_ = svc.Export(portal, portalPath, portalNotification)
	clicks := make(chan string, 4)
	n := newNotifier(NotifierOptions{AppName: "Mail", DesktopEntry: "mailclientui"}, func() (*dbus.Conn, error) { return client, nil }, portalDest, fdoNotify)
	defer n.Close()
	if _, err := n.Send(testNote(clicks)); err != nil {
		t.Fatal(err)
	}
	if n.Backend() != "fdo" || len(portal.added) != 0 {
		t.Fatalf("backend %q, portal got %v", n.Backend(), portal.added)
	}
	fake.mu.Lock()
	if fake.app != "Mail" || fake.icon != "mail-unread" || fake.summary != "Ada Lovelace" || fake.body != "Notes on the engine" {
		t.Fatalf("notify %q %q %q %q", fake.app, fake.icon, fake.summary, fake.body)
	}
	if strings.Join(fake.actions, "|") != "default|Open|button-0|Open Mail|button-1|Mark Read" {
		t.Fatalf("actions %q", fake.actions)
	}
	if u, _ := fake.hints["urgency"].Value().(byte); u != 1 {
		t.Fatalf("urgency %v", fake.hints["urgency"])
	}
	if e, _ := fake.hints["desktop-entry"].Value().(string); e != "mailclientui" {
		t.Fatalf("desktop-entry %v", fake.hints["desktop-entry"])
	}
	fake.mu.Unlock()
	// The same ID replaces it.
	note := testNote(clicks)
	note.Body = "Two new messages"
	if _, err := n.Send(note); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	if len(fake.replaces) != 2 || fake.replaces[0] != 0 || fake.replaces[1] != 41 {
		t.Fatalf("replaces_id %v, want [0 41]", fake.replaces)
	}
	fake.mu.Unlock()
	// Another app's notification (id 7), then ours.
	_ = svc.Emit(fdoNotifyPath, fdoNotify+".ActionInvoked", uint32(7), "default")
	_ = svc.Emit(fdoNotifyPath, fdoNotify+".ActionInvoked", uint32(41), "default")
	select {
	case a := <-clicks:
		if a != "" {
			t.Fatalf("a click on the body is action %q, want \"\"", a)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the click never came back")
	}
	_ = svc.Emit(fdoNotifyPath, fdoNotify+".ActionInvoked", uint32(41), "button-0")
	if a := <-clicks; a != "open" {
		t.Fatalf("button 0 is %q", a)
	}
	// Closed by the server: forgotten, so a click on the old id is nobody's.
	_ = svc.Emit(fdoNotifyPath, fdoNotify+".NotificationClosed", uint32(41), uint32(2))
	waitFor(t, "the close", func() bool { n.mu.Lock(); defer n.mu.Unlock(); return len(n.live) == 0 })
	// Pixels without a name: image-data, (iiibiiay) with straight RGBA.
	img := paintengine2d.NewImage(2, 2)
	img.SetColor(0, 0, paintengine2d.RGBA(1, 0, 0, 1))
	if _, err := n.Send(DesktopNotification{Title: "pixels", Icon: img}); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fields, _ := fake.hints["image-data"].Value().([]any)
	if len(fields) != 7 || fields[0] != int32(2) || fields[1] != int32(2) || fields[2] != int32(8) ||
		fields[3] != true || fields[4] != int32(8) || fields[5] != int32(4) {
		t.Fatalf("image-data %v", fields)
	}
	if px, _ := fields[6].([]byte); len(px) != 16 || px[0] != 255 || px[3] != 255 {
		t.Fatalf("pixels % x", px)
	}
}

// UITK_NOTIFY picks the service outright.
func TestNotifierEnvPicks(t *testing.T) {
	svc, client := busPair(t)
	own(t, svc, fdoNotify)
	own(t, svc, portalDest)
	_ = svc.Export(&fakeNotifyServer{}, fdoNotifyPath, fdoNotify)
	_ = svc.Export(&fakeNotifyPortal{}, portalPath, portalNotification)
	dial := func() (*dbus.Conn, error) { return client, nil }
	for want, env := range map[string]string{"portal": "portal", "fdo": "fdo"} {
		t.Setenv(EnvNotify, env)
		n := newNotifier(NotifierOptions{}, dial, portalDest, fdoNotify)
		if _, err := n.Send(DesktopNotification{Title: "x"}); err != nil || n.Backend() != want {
			t.Fatalf("UITK_NOTIFY=%s: %v via %q", env, err, n.Backend())
		}
		n.Close()
	}
	t.Setenv(EnvNotify, "off")
	n := newNotifier(NotifierOptions{}, dial, portalDest, fdoNotify)
	if id, err := n.Send(DesktopNotification{Title: "x"}); err != nil || id == "" || n.Backend() != "" {
		t.Fatalf("off: %q %v %q", id, err, n.Backend())
	}
}

// With neither service on the bus — and nothing D-Bus could start — Send
// fails at once, with ErrNoNotifications; with no bus at all, too. A stub
// sends nothing and says it did.
func TestNotifierWithoutServicesFailsQuickly(t *testing.T) {
	t.Setenv(EnvNotify, "")
	_, client := busPair(t)
	n := newNotifier(NotifierOptions{}, func() (*dbus.Conn, error) { return client, nil }, portalDest, fdoNotify)
	defer n.Close()
	start := time.Now()
	if _, err := n.Send(DesktopNotification{Title: "x"}); !errors.Is(err, ErrNoNotifications) {
		t.Fatalf("err %v", err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("took %v to find nothing", d)
	}
	none := newNotifier(NotifierOptions{}, func() (*dbus.Conn, error) { return nil, errNoSessionBus }, portalDest, fdoNotify)
	if _, err := none.Send(DesktopNotification{Title: "x"}); !errors.Is(err, ErrNoNotifications) {
		t.Fatalf("no bus: %v", err)
	}
	stub := NewNotifier(NotifierOptions{Stub: true})
	if id, err := stub.Send(DesktopNotification{Title: "x"}); err != nil || id == "" || stub.Backend() != "stub" {
		t.Fatalf("stub: %q %v %q", id, err, stub.Backend())
	}
}

// fakeOpenURI is org.freedesktop.portal.OpenURI; a file arrives as a
// descriptor, which it turns back into the path it names.
type fakeOpenURI struct {
	mu      sync.Mutex
	calls   []string
	options map[string]dbus.Variant
}

func (f *fakeOpenURI) record(call string, opts map[string]dbus.Variant) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
	f.options = opts
}

func fdPath(fd dbus.UnixFD) string {
	p, _ := os.Readlink("/proc/self/fd/" + strconv.Itoa(int(fd)))
	_ = syscall.Close(int(fd))
	return p
}

func (f *fakeOpenURI) OpenURI(parent, uri string, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f.record("OpenURI "+parent+" "+uri, opts)
	return "/org/freedesktop/portal/desktop/request/x/y", nil
}

func (f *fakeOpenURI) OpenFile(parent string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f.record("OpenFile "+parent+" "+fdPath(fd), opts)
	return "/org/freedesktop/portal/desktop/request/x/y", nil
}

func (f *fakeOpenURI) OpenDirectory(parent string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f.record("OpenDirectory "+parent+" "+fdPath(fd), opts)
	return "/org/freedesktop/portal/desktop/request/x/y", nil
}

// OpenURI through the portal: a link as OpenURI with the parent window and
// the options, a file as OpenFile and a folder as OpenDirectory by
// descriptor; xdg-open is not run.
func TestOpenURIThroughThePortal(t *testing.T) {
	svc, client := busPair(t)
	own(t, svc, portalDest)
	fake := &fakeOpenURI{}
	if err := svc.Export(fake, portalPath, portalOpenURI); err != nil {
		t.Fatal(err)
	}
	fell := 0
	defer swapFallback(func(string) error { fell++; return nil })()
	dir := t.TempDir()
	file := filepath.Join(dir, "notes on the engine.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := OpenURIOptions{ParentWindow: "wayland:handle-7", Ask: true, ActivationToken: "tok"}
	for _, uri := range []string{"https://example.org/a?b=c", "file://" + strings.ReplaceAll(file, " ", "%20"), dir} {
		if err := openURIWith(client, nil, portalDest, uri, opts); err != nil {
			t.Fatalf("%s: %v", uri, err)
		}
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	want := []string{
		"OpenURI wayland:handle-7 https://example.org/a?b=c",
		"OpenFile wayland:handle-7 " + file,
		"OpenDirectory wayland:handle-7 " + dir,
	}
	if strings.Join(fake.calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("calls:\n%s\nwant:\n%s", strings.Join(fake.calls, "\n"), strings.Join(want, "\n"))
	}
	if a, _ := fake.options["ask"].Value().(bool); !a {
		t.Fatal("ask not passed")
	}
	if tok, _ := fake.options["activation_token"].Value().(string); tok != "tok" {
		t.Fatal("activation_token not passed")
	}
	if h, _ := fake.options["handle_token"].Value().(string); h == "" {
		t.Fatal("no handle_token")
	}
	if fell != 0 {
		t.Fatalf("xdg-open ran %d times with a portal there", fell)
	}
}

// Without a portal — none running, none to start — OpenURI goes to
// xdg-open at once; with no bus at all, too. Nothing that looks like an
// option is ever handed to it.
func TestOpenURIFallsBackQuickly(t *testing.T) {
	_, client := busPair(t)
	var got []string
	defer swapFallback(func(u string) error { got = append(got, u); return nil })()
	start := time.Now()
	if err := openURIWith(client, nil, portalDest, "mailto:ada@example.org", OpenURIOptions{}); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("took %v to find no portal", d)
	}
	if err := openURIWith(nil, errNoSessionBus, portalDest, "https://example.org", OpenURIOptions{}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "mailto:ada@example.org https://example.org" {
		t.Fatalf("fallback got %q", got)
	}
	if err := OpenURI("--help", OpenURIOptions{}); !errors.Is(err, ErrNoOpener) {
		t.Fatalf("an option-looking URI: %v", err)
	}
	if err := OpenURI("  ", OpenURIOptions{}); !errors.Is(err, ErrNoOpener) {
		t.Fatalf("an empty URI: %v", err)
	}
	t.Setenv(EnvOpenURI, "xdg-open")
	if err := OpenURI("https://example.org/b", OpenURIOptions{}); err != nil || got[len(got)-1] != "https://example.org/b" {
		t.Fatalf("UITK_OPENURI=xdg-open: %v %q", err, got)
	}
}

// The file chooser, too, gives up at once without a portal, so the
// toolkit's own dialog shows instead of nothing; with one, the parent
// window goes to it.
func TestFileChooserParentAndNoPortal(t *testing.T) {
	svc, client := busPair(t)
	start := time.Now()
	if openFileChooser(client, portalDest, FileChooserOptions{Title: "x"}, func([]string) {}) {
		t.Fatal("a chooser opened with no portal")
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("took %v", d)
	}
	own(t, svc, portalDest)
	fake := &parentChooser{conn: svc}
	if err := svc.Export(fake, portalPath, portalFileChooser); err != nil {
		t.Fatal(err)
	}
	done := make(chan []string, 1)
	if !openFileChooser(client, portalDest, FileChooserOptions{Title: "Attach", ParentWindow: "x11:1a00003"}, func(p []string) { done <- p }) {
		t.Fatal("the portal call failed")
	}
	<-done
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.parent != "x11:1a00003" {
		t.Fatalf("parent %q", fake.parent)
	}
}

type parentChooser struct {
	conn   *dbus.Conn
	mu     sync.Mutex
	parent string
}

func (f *parentChooser) OpenFile(sender dbus.Sender, parent, title string, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f.mu.Lock()
	f.parent = parent
	f.mu.Unlock()
	token, _ := options["handle_token"].Value().(string)
	path := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" +
		strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_") + "/" + token)
	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = f.conn.Emit(path, portalRequest+".Response", uint32(1), map[string]dbus.Variant{})
	}()
	return path, nil
}

func swapFallback(fn func(string) error) func() {
	old := openURIFallback
	openURIFallback = fn
	return func() { openURIFallback = old }
}
