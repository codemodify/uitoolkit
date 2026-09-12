//go:build linux

package platform

import (
	"os"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestDbusMenuLayoutSignatureFinite(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("layout signature panicked: %v", r)
		}
	}()
	if err := assertFiniteMenuSignature(); err != nil {
		t.Fatal(err)
	}
	layout := buildMenuLayout([]StatusMenuItem{
		{Text: "Show Mail"},
		{Separator: true},
		{Text: "Quit"},
		{Text: "VIP", Checked: true, Disabled: true},
	})
	if len(layout.Children) != 4 {
		t.Fatalf("children %d", len(layout.Children))
	}
	// This is the exact SignatureOf call godbus handleCall does on the
	// GetLayout reply body. The v0.16.0 recursive []dbusMenuLayout panicked here.
	sig := dbus.SignatureOf(uint32(1), layout)
	if sig.String() != dbusMenuGetLayoutSig {
		t.Fatalf("reply signature %s", sig)
	}
	for i, child := range layout.Children {
		if child.Signature().String() != dbusMenuLayoutSig {
			t.Fatalf("child %d signature %s", i, child.Signature())
		}
	}
}

func TestRecursiveDbusMenuLayoutPanics(t *testing.T) {
	// Documents the v0.16.0 Plasma crash: a self-referential layout type
	// has no finite D-Bus signature.
	type recursiveMenuLayout struct {
		ID         int32
		Properties map[string]dbus.Variant
		Children   []recursiveMenuLayout
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected container nesting panic for recursive layout")
		}
	}()
	_ = dbus.SignatureOf(recursiveMenuLayout{})
}

func TestGetLayoutReplySignatureNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetLayout reply SignatureOf: %v", r)
		}
	}()
	s := &linuxStatusItem{
		menu: []StatusMenuItem{
			{Text: "Show Mail"},
			{Separator: true},
			{Text: "Quit"},
		},
		menuRev: 3,
	}
	rev, layout, derr := s.GetLayout(0, -1, nil)
	if derr != nil {
		t.Fatal(derr)
	}
	if rev != 3 || layout.ID != 0 || len(layout.Children) != 1 {
		t.Fatalf("stub layout rev=%d children=%d", rev, len(layout.Children))
	}
	_ = dbus.SignatureOf(rev, layout)
	_, leaf, derr := s.GetLayout(1, 0, nil)
	if derr != nil || leaf.ID != 1 || len(leaf.Children) != 0 {
		t.Fatalf("leaf %+v %v", leaf, derr)
	}
	_ = dbus.SignatureOf(rev, leaf)
}

func TestLinuxContextMenuInvokesOnMenu(t *testing.T) {
	n := 0
	var gx, gy int32
	s := &linuxStatusItem{
		opts: StatusItemOptions{
			OnMenu: func(x, y int32) {
				n++
				gx, gy = x, y
			},
		},
	}
	if err := s.ContextMenu(12, 34); err != nil {
		t.Fatal(err)
	}
	if n != 1 || gx != 12 || gy != 34 {
		t.Fatalf("ContextMenu n=%d %d,%d", n, gx, gy)
	}
	if err := s.SecondaryActivate(5, 6); err != nil {
		t.Fatal(err)
	}
	if n != 2 || gx != 5 || gy != 6 {
		t.Fatalf("SecondaryActivate n=%d %d,%d", n, gx, gy)
	}
	need, aerr := s.AboutToShow(0)
	if aerr != nil || need {
		t.Fatalf("AboutToShow %v %v", need, aerr)
	}
	if n != 3 {
		t.Fatalf("AboutToShow should open toolkit menu, n=%d", n)
	}
	if err := s.Event(1, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("Event stub click should open toolkit menu, n=%d", n)
	}
}

func TestDbusMenuEventInvokesOnClick(t *testing.T) {
	n := 0
	s := &linuxStatusItem{
		menu: []StatusMenuItem{
			{Text: "Show Mail", OnClick: func() { n++ }},
			{Separator: true},
			{Text: "Quit", OnClick: func() { n += 4 }},
		},
	}
	if err := s.Event(1, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("Event clicked %d", n)
	}
	ids, err := s.EventGroup([]dbusMenuEvent{
		{ID: 3, EventID: "clicked"},
	})
	if err != nil || ids != nil {
		t.Fatalf("EventGroup %v %v", ids, err)
	}
	if n != 5 {
		t.Fatalf("EventGroup clicked %d", n)
	}
	need, err := s.AboutToShow(0)
	if err != nil || need {
		t.Fatalf("AboutToShow %v %v", need, err)
	}
}

func TestNewStatusItemNeverPanics(t *testing.T) {
	os.Unsetenv("UITK_TRAY")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewStatusItem panicked: %v", r)
		}
	}()
	item, err := NewStatusItem(StatusItemOptions{
		ID:    "mailclientui",
		Title: "Mail",
		Menu: []StatusMenuItem{
			{Text: "Show Mail"},
			{Separator: true},
			{Text: "Quit"},
		},
		OnClick: func() {},
	})
	if err != nil || item == nil {
		t.Fatalf("item %v %v", item, err)
	}
	t.Cleanup(func() { _ = item.Close() })
	_ = item.SetMenu([]StatusMenuItem{{Text: "Show"}, {Separator: true}, {Text: "Quit"}})
	_ = item.Backend()
}

func sessionBusOrSkip(t *testing.T) *dbus.Conn {
	t.Helper()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Skipf("no session bus: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestExportGetLayoutNoNestingPanic(t *testing.T) {
	_ = sessionBusOrSkip(t)
	os.Unsetenv("UITK_TRAY")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("export/GetLayout panicked: %v", r)
		}
	}()
	item, err := newNativeStatusItem(StatusItemOptions{
		ID:    "uitoolkit-dbusmenu-test",
		Title: "Mail",
		Menu: []StatusMenuItem{
			{Text: "Show Mail"},
			{Separator: true},
			{Text: "Quit"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Close() })
	s, ok := item.(*linuxStatusItem)
	if !ok {
		t.Skip("native export fell back to stub (no bus name)")
	}
	var rev uint32
	var layout dbusMenuNode
	call := s.conn.Object(s.name, dbusMenuPath).Call(
		dbusMenuIface+".GetLayout", 0, int32(0), int32(-1), []string{},
	)
	if err := call.Store(&rev, &layout); err != nil {
		t.Fatalf("GetLayout over bus: %v", err)
	}
	if len(layout.Children) > 1 {
		t.Fatalf("stub layout should not export real rows, children %d", len(layout.Children))
	}
	_ = dbus.SignatureOf(rev, layout)
	menuVar, derr := s.props.Get(sniInterface, "Menu")
	if derr != nil {
		t.Fatal(derr)
	}
	path, _ := menuVar.Value().(dbus.ObjectPath)
	if path != sniMenuNone {
		t.Fatalf("Menu %q want %s", path, sniMenuNone)
	}
}

func TestSurviveStatusNotifierWatcherGetLayout(t *testing.T) {
	conn := sessionBusOrSkip(t)
	os.Unsetenv("UITK_TRAY")
	hasWatcher := nameHasOwner(t, conn, sniWatcher)
	stopFake := func() {}
	if !hasWatcher {
		var err error
		stopFake, err = startFakeSNIWatcher(conn)
		if err != nil {
			t.Skipf("cannot own %s: %v", sniWatcher, err)
		}
	}
	defer stopFake()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("SNI watcher GetLayout panicked the item: %v", r)
		}
	}()
	item, err := NewStatusItem(StatusItemOptions{
		ID:      "mailclientui",
		Title:   "Mail",
		Tooltip: "Mail",
		Menu: []StatusMenuItem{
			{Text: "Show Mail"},
			{Separator: true},
			{Text: "Quit"},
		},
		OnClick: func() {},
	})
	if err != nil || item == nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Close() })
	// Plasma (and the fake watcher) call GetLayout right after register.
	time.Sleep(300 * time.Millisecond)
	if s, ok := item.(*linuxStatusItem); ok {
		var rev uint32
		var layout dbusMenuNode
		err := s.conn.Object(s.name, dbusMenuPath).Call(
			dbusMenuIface+".GetLayout", 0, int32(0), int32(-1), []string{},
		).Store(&rev, &layout)
		if err != nil {
			t.Fatalf("GetLayout after watcher: %v", err)
		}
		if len(layout.Children) > 1 {
			t.Fatalf("stub layout children %d", len(layout.Children))
		}
	}
	if !item.Alive() && item.Backend() == "stub" {
		t.Fatal("tray register fell back to stub; window would live but SNI export failed")
	}
}

func nameHasOwner(t *testing.T, conn *dbus.Conn, name string) bool {
	t.Helper()
	var owner bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, name).Store(&owner); err != nil {
		return false
	}
	return owner
}

// fakeSNIWatcher mimics Plasma: on RegisterStatusNotifierItem it immediately
// calls com.canonical.dbusmenu.GetLayout on the new item (the v0.16.0 panic).
type fakeSNIWatcher struct {
	conn *dbus.Conn
}

func (w *fakeSNIWatcher) RegisterStatusNotifierItem(service string) *dbus.Error {
	go func() {
		dest, path := service, dbus.ObjectPath(dbusMenuPath)
		if len(service) > 0 && service[0] == '/' {
			path = dbus.ObjectPath(service)
			if names := w.conn.Names(); len(names) > 0 {
				dest = names[0]
			}
		}
		var rev uint32
		var layout dbusMenuNode
		_ = w.conn.Object(dest, path).Call(
			dbusMenuIface+".GetLayout", 0, int32(0), int32(-1), []string{},
		).Store(&rev, &layout)
	}()
	return nil
}

func startFakeSNIWatcher(conn *dbus.Conn) (func(), error) {
	reply, err := conn.RequestName(sniWatcher, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return func() {}, fmtStartWatcher(err, reply)
	}
	w := &fakeSNIWatcher{conn: conn}
	if err := conn.ExportMethodTable(map[string]interface{}{
		"RegisterStatusNotifierItem": w.RegisterStatusNotifierItem,
	}, sniWatcherPath, sniWatcher); err != nil {
		_, _ = conn.ReleaseName(sniWatcher)
		return func() {}, err
	}
	return func() {
		_ = conn.Export(nil, sniWatcherPath, sniWatcher)
		_, _ = conn.ReleaseName(sniWatcher)
	}, nil
}

func fmtStartWatcher(err error, reply dbus.RequestNameReply) error {
	if err != nil {
		return err
	}
	return errWatcherOwned
}

var errWatcherOwned = errString("StatusNotifierWatcher already owned")

type errString string

func (e errString) Error() string { return string(e) }
