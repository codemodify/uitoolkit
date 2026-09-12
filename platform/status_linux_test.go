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
	if rev != 3 || layout.ID != 0 || len(layout.Children) != 3 {
		t.Fatalf("HostMenu layout rev=%d children=%d", rev, len(layout.Children))
	}
	_ = dbus.SignatureOf(rev, layout)
	_, leaf, derr := s.GetLayout(1, 0, nil)
	if derr != nil || leaf.ID != 1 || len(leaf.Children) != 0 {
		t.Fatalf("leaf %+v %v", leaf, derr)
	}
	if leaf.Properties["label"].Value() != "Show Mail" {
		t.Fatalf("leaf label %+v", leaf.Properties)
	}
	_ = dbus.SignatureOf(rev, leaf)
}

func TestHostMenuGetLayoutHasRealRows(t *testing.T) {
	s := &linuxStatusItem{
		opts: StatusItemOptions{MenuChrome: HostMenu},
		menu: []StatusMenuItem{
			{Text: "Show Mail", Icon: 0},
			{Separator: true},
			{Text: "Quit", Checked: true},
		},
		menuRev: 2,
	}
	rev, layout, err := s.GetLayout(0, -1, nil)
	if err != nil || rev != 2 || len(layout.Children) != 3 {
		t.Fatalf("rev=%d children=%d %v", rev, len(layout.Children), err)
	}
	labels := make([]string, 0, 3)
	for _, child := range layout.Children {
		leaf, ok := child.Value().(dbusMenuLeaf)
		if !ok {
			t.Fatalf("child type %T", child.Value())
		}
		if leaf.ID < 1 {
			t.Fatalf("id %d", leaf.ID)
		}
		if tpe, ok := leaf.Properties["type"]; ok && tpe.Value() == "separator" {
			labels = append(labels, "-")
			continue
		}
		lab, _ := leaf.Properties["label"].Value().(string)
		labels = append(labels, lab)
	}
	if labels[0] != "Show Mail" || labels[1] != "-" || labels[2] != "Quit" {
		t.Fatalf("labels %v", labels)
	}
}

func TestToolkitMenuGetLayoutEmptyAndPath(t *testing.T) {
	s := &linuxStatusItem{
		opts: StatusItemOptions{MenuChrome: ToolkitMenu},
		menu: []StatusMenuItem{{Text: "Show Mail"}, {Separator: true}, {Text: "Quit"}},
	}
	if s.menuPath() != sniMenuNone || sniMenuNone != dbus.ObjectPath("/NO_DBUSMENU") {
		t.Fatalf("ToolkitMenu path %q", s.menuPath())
	}
	_, layout, err := s.GetLayout(0, -1, nil)
	if err != nil || len(layout.Children) != 0 {
		t.Fatalf("ToolkitMenu must not export rows, children=%d %v", len(layout.Children), err)
	}
}

func TestLinuxContextMenuInvokesOnMenu(t *testing.T) {
	n := 0
	var gx, gy int32
	s := &linuxStatusItem{
		opts: StatusItemOptions{
			MenuChrome: ToolkitMenu,
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
	dispatched := 0
	s := &linuxStatusItem{
		opts: StatusItemOptions{
			MenuChrome: HostMenu,
			Dispatch: func(fn func()) {
				dispatched++
				fn()
			},
		},
		menu: []StatusMenuItem{
			{Text: "Show Mail", OnClick: func() { n++ }},
			{Separator: true},
			{Text: "Quit", OnClick: func() { n += 4 }},
		},
	}
	if err := s.Event(1, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if n != 1 || dispatched != 1 {
		t.Fatalf("Event clicked n=%d dispatched=%d", n, dispatched)
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
	opened := 0
	s.opts.OnMenu = func(x, y int32) { opened++ }
	need, err = s.AboutToShow(0)
	if err != nil || need || opened != 0 {
		t.Fatalf("HostMenu AboutToShow must not open toolkit menu, opened=%d", opened)
	}
}

func TestSetMenuDedupesLayoutUpdated(t *testing.T) {
	s := &linuxStatusItem{
		menu:    []StatusMenuItem{{Text: "Show"}, {Separator: true}, {Text: "Quit"}},
		menuRev: 4,
	}
	if err := s.SetMenu([]StatusMenuItem{{Text: "Show"}, {Separator: true}, {Text: "Quit"}}); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 4 {
		t.Fatalf("identical SetMenu must not bump rev, got %d", s.menuRev)
	}
	if err := s.SetMenu([]StatusMenuItem{{Text: "Show Mail"}, {Separator: true}, {Text: "Quit"}}); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 5 {
		t.Fatalf("changed SetMenu rev %d", s.menuRev)
	}
}

func TestWatcherOwnerChangedReregisters(t *testing.T) {
	s := &linuxStatusItem{}
	s.onWatcherOwnerChanged("org.freedesktop.Notifications", ":1.2")
	s.onWatcherOwnerChanged(sniWatcher, "")
	s.onWatcherOwnerChanged(sniWatcher, ":1.9")
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
	if len(layout.Children) != 3 {
		t.Fatalf("HostMenu GetLayout children %d want 3", len(layout.Children))
	}
	_ = dbus.SignatureOf(rev, layout)
	menuVar, derr := s.props.Get(sniInterface, "Menu")
	if derr != nil {
		t.Fatal(derr)
	}
	path, _ := menuVar.Value().(dbus.ObjectPath)
	if path != dbus.ObjectPath(dbusMenuPath) {
		t.Fatalf("Menu %q want %s", path, dbusMenuPath)
	}
}

func TestSNIMenuNoneIsKDESentinel(t *testing.T) {
	if sniMenuNone != dbus.ObjectPath("/NO_DBUSMENU") {
		t.Fatalf("sniMenuNone %q want /NO_DBUSMENU", sniMenuNone)
	}
}

func TestToolkitMenuExportsNoDbusmenuPath(t *testing.T) {
	_ = sessionBusOrSkip(t)
	os.Unsetenv("UITK_TRAY")
	item, err := newNativeStatusItem(StatusItemOptions{
		ID:         "uitoolkit-toolkitmenu-test",
		Title:      "Mail",
		MenuChrome: ToolkitMenu,
		Menu:       []StatusMenuItem{{Text: "Show Mail"}, {Separator: true}, {Text: "Quit"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = item.Close() })
	s, ok := item.(*linuxStatusItem)
	if !ok {
		t.Skip("native export fell back to stub")
	}
	menuVar, derr := s.props.Get(sniInterface, "Menu")
	if derr != nil {
		t.Fatal(derr)
	}
	path, _ := menuVar.Value().(dbus.ObjectPath)
	if path != sniMenuNone {
		t.Fatalf("ToolkitMenu Menu %q want %s", path, sniMenuNone)
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
		if len(layout.Children) != 3 {
			t.Fatalf("HostMenu layout children %d", len(layout.Children))
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
