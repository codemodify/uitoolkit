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

func TestHostMenuGetLayoutEmptyAndUnknownParent(t *testing.T) {
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menuRev: 1,
	}
	rev, layout, err := s.GetLayout(0, -1, nil)
	if err != nil || rev != 1 || layout.ID != 0 || len(layout.Children) != 0 {
		t.Fatalf("empty HostMenu layout rev=%d id=%d children=%d %v", rev, layout.ID, len(layout.Children), err)
	}
	if _, ok := layout.Properties["children-display"]; !ok {
		t.Fatal("empty root must keep children-display=submenu")
	}
	_, missing, err := s.GetLayout(4, 0, nil)
	if err != nil || missing.ID != 4 || len(missing.Children) != 0 {
		t.Fatalf("unknown parent %+v %v", missing, err)
	}
	if err := s.SetMenu(nil); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 1 {
		t.Fatalf("empty→empty SetMenu must not churn rev, got %d", s.menuRev)
	}
}

func TestItemIsMenuActivateDoesNotClick(t *testing.T) {
	n := 0
	opened := 0
	s := &linuxStatusItem{
		opts: StatusItemOptions{
			MenuChrome: HostMenu,
			ItemIsMenu: true,
			OnClick:    func() { n++ },
		},
	}
	if err := s.Activate(1, 2); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("HostMenu ItemIsMenu Activate must not fire OnClick, n=%d", n)
	}
	s.opts.OnMenu = func(x, y int32) { opened++ }
	s.opts.MenuChrome = ToolkitMenu
	if err := s.Activate(3, 4); err != nil {
		t.Fatal(err)
	}
	if opened != 1 || n != 0 {
		t.Fatalf("ToolkitMenu ItemIsMenu should OnMenu, opened=%d n=%d", opened, n)
	}
}

func TestSecondaryActivateHostMenuDoesNotRaise(t *testing.T) {
	n := 0
	s := &linuxStatusItem{
		opts: StatusItemOptions{
			MenuChrome: HostMenu,
			OnClick:    func() { n++ },
		},
	}
	if err := s.SecondaryActivate(8, 9); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("HostMenu SecondaryActivate must not Activate, n=%d", n)
	}
	if err := s.Activate(8, 9); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("plain Activate still raises, n=%d", n)
	}
}

func TestDbusMenuEventIgnoresDisabledAndSeparator(t *testing.T) {
	n := 0
	s := &linuxStatusItem{
		opts: StatusItemOptions{MenuChrome: HostMenu},
		menu: []StatusMenuItem{
			{Text: "Show", OnClick: func() { n++ }},
			{Separator: true, OnClick: func() { n += 10 }},
			{Text: "Later", Disabled: true, OnClick: func() { n += 100 }},
		},
	}
	if err := s.Event(2, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(3, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("separator/disabled Event must not click, n=%d", n)
	}
	if err := s.Event(1, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("enabled Event n=%d", n)
	}
	ids, err := s.EventGroup([]dbusMenuEvent{
		{ID: 1, EventID: "clicked"},
		{ID: 99, EventID: "clicked"},
		{ID: 0, EventID: "clicked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 99 || ids[1] != 0 {
		t.Fatalf("unknown EventGroup ids %v", ids)
	}
	if n != 2 {
		t.Fatalf("EventGroup valid click n=%d", n)
	}
}

func TestStatusIconNameFallback(t *testing.T) {
	if got := statusIconName(StatusIcon{Name: "mail-unread"}); got != "mail-unread" {
		t.Fatalf("named %q", got)
	}
	if got := statusIconName(StatusIcon{}); got != "application-default-icon" {
		t.Fatalf("empty fallback %q", got)
	}
	s := &linuxStatusItem{icon: StatusIcon{}}
	if s.toolTip().IconName != "application-default-icon" {
		t.Fatalf("tooltip IconName %q", s.toolTip().IconName)
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
	if err := s.SetMenu(nil); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 6 {
		t.Fatalf("clearing the menu must bump rev, got %d", s.menuRev)
	}
	if err := s.SetMenu(nil); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 6 {
		t.Fatalf("second empty SetMenu must not churn, got %d", s.menuRev)
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
	itemIsMenu, derr := s.props.Get(sniInterface, "ItemIsMenu")
	if derr != nil {
		t.Fatal(derr)
	}
	if on, _ := itemIsMenu.Value().(bool); on {
		t.Fatal("default ItemIsMenu must be false")
	}
	iconName, derr := s.props.Get(sniInterface, "IconName")
	if derr != nil {
		t.Fatal(derr)
	}
	if name, _ := iconName.Value().(string); name != "application-default-icon" {
		t.Fatalf("empty IconName fallback %q", name)
	}
	var ver dbus.Variant
	if err := s.conn.Object(s.name, dbusMenuPath).Call(
		"org.freedesktop.DBus.Properties.Get", 0, dbusMenuIface, "Version",
	).Store(&ver); err != nil {
		t.Fatalf("dbusmenu Version: %v", err)
	}
	if u, _ := ver.Value().(uint32); u != 3 {
		t.Fatalf("dbusmenu Version %v want 3", ver.Value())
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

// nestedStatusMenu is the tree the submenu tests share. Pre-order ids:
// 1 Show Mail, 2 separator, 3 Folders, 4 Inbox, 5 separator, 6 Archive,
// 7 2025, 8 Quit.
func nestedStatusMenu(clicks map[string]int) []StatusMenuItem {
	hit := func(name string) func() { return func() { clicks[name]++ } }
	return []StatusMenuItem{
		{Text: "Show Mail", OnClick: hit("show")},
		{Separator: true},
		{Text: "Folders", OnClick: hit("folders"), Submenu: []StatusMenuItem{
			{Text: "Inbox", OnClick: hit("inbox")},
			{Separator: true},
			{Text: "Archive", Submenu: []StatusMenuItem{
				{Text: "2025", OnClick: hit("2025")},
			}},
		}},
		{Text: "Quit", OnClick: hit("quit")},
	}
}

func childLeaf(t *testing.T, children []dbus.Variant, i int) dbusMenuLeaf {
	t.Helper()
	if i < 0 || i >= len(children) {
		t.Fatalf("child %d of %d", i, len(children))
	}
	leaf, ok := children[i].Value().(dbusMenuLeaf)
	if !ok {
		t.Fatalf("child %d is %T, want dbusMenuLeaf", i, children[i].Value())
	}
	return leaf
}

func TestNestedGetLayoutNumbersTreeInPreOrder(t *testing.T) {
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menu:    nestedStatusMenu(map[string]int{}),
		menuRev: 7,
	}
	rev, layout, err := s.GetLayout(0, -1, nil)
	if err != nil || rev != 7 || len(layout.Children) != 4 {
		t.Fatalf("rev=%d children=%d %v", rev, len(layout.Children), err)
	}
	// The whole reply still has to have a finite signature: this is the
	// call godbus makes on the way out, now with two levels of nesting.
	if sig := dbus.SignatureOf(rev, layout); sig.String() != dbusMenuGetLayoutSig {
		t.Fatalf("nested reply signature %s", sig)
	}
	folders := childLeaf(t, layout.Children, 2)
	if folders.ID != 3 {
		t.Fatalf("Folders id %d want 3 (pre-order)", folders.ID)
	}
	if folders.Properties["children-display"].Value() != "submenu" {
		t.Fatalf("Folders props %+v want children-display=submenu", folders.Properties)
	}
	if len(folders.Children) != 3 {
		t.Fatalf("Folders children %d want 3", len(folders.Children))
	}
	inbox := childLeaf(t, folders.Children, 0)
	if inbox.ID != 4 || inbox.Properties["label"].Value() != "Inbox" {
		t.Fatalf("Inbox %d %+v", inbox.ID, inbox.Properties)
	}
	if _, nested := inbox.Properties["children-display"]; nested {
		t.Fatal("a command row must not claim children-display")
	}
	sep := childLeaf(t, folders.Children, 1)
	if sep.Properties["type"].Value() != "separator" {
		t.Fatalf("separator inside the submenu lost: %+v", sep.Properties)
	}
	archive := childLeaf(t, folders.Children, 2)
	if archive.ID != 6 || len(archive.Children) != 1 {
		t.Fatalf("Archive id=%d children=%d", archive.ID, len(archive.Children))
	}
	if y := childLeaf(t, archive.Children, 0); y.ID != 7 || y.Properties["label"].Value() != "2025" {
		t.Fatalf("grandchild %d %+v", y.ID, y.Properties)
	}
	if quit := childLeaf(t, layout.Children, 3); quit.ID != 8 {
		t.Fatalf("Quit id %d want 8 (after the subtree, not 4)", quit.ID)
	}
	// Every id in the tree is unique — that is the whole point of the
	// pre-order walk, and the old id == index+1 scheme could not do it.
	seen := map[int32]bool{}
	var walk func(children []dbus.Variant)
	walk = func(children []dbus.Variant) {
		for i := range children {
			leaf := childLeaf(t, children, i)
			if seen[leaf.ID] {
				t.Fatalf("duplicate id %d", leaf.ID)
			}
			seen[leaf.ID] = true
			walk(leaf.Children)
		}
	}
	walk(layout.Children)
	if len(seen) != 8 {
		t.Fatalf("ids %v want 8 rows", seen)
	}
}

func TestGetLayoutHonoursRecursionDepth(t *testing.T) {
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menu:    nestedStatusMenu(map[string]int{}),
		menuRev: 1,
	}
	// 0: the node alone. A host asking "what is this row?" is not asking
	// for the menu under it.
	_, none, err := s.GetLayout(0, 0, nil)
	if err != nil || len(none.Children) != 0 {
		t.Fatalf("depth 0 children %d %v", len(none.Children), err)
	}
	// 1: the immediate children, and no grandchildren.
	_, one, err := s.GetLayout(0, 1, nil)
	if err != nil || len(one.Children) != 4 {
		t.Fatalf("depth 1 children %d %v", len(one.Children), err)
	}
	folders := childLeaf(t, one.Children, 2)
	if len(folders.Children) != 0 {
		t.Fatalf("depth 1 delivered grandchildren: %d", len(folders.Children))
	}
	if folders.Properties["children-display"].Value() != "submenu" {
		t.Fatal("a truncated parent must still say it has a submenu")
	}
	// A subtree root with its own level.
	_, sub, err := s.GetLayout(3, 1, nil)
	if err != nil || sub.ID != 3 || len(sub.Children) != 3 {
		t.Fatalf("subtree %+v %v", sub, err)
	}
	if archive := childLeaf(t, sub.Children, 2); len(archive.Children) != 0 {
		t.Fatalf("depth 1 from id 3 delivered grandchildren: %d", len(archive.Children))
	}
	// 2 from the root reaches the grandchild.
	_, two, err := s.GetLayout(0, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(childLeaf(t, two.Children, 2).Children); got != 3 {
		t.Fatalf("depth 2 children of Folders %d", got)
	}
	if got := len(childLeaf(t, childLeaf(t, two.Children, 2).Children, 2).Children); got != 0 {
		t.Fatalf("depth 2 reached great-grandchildren: %d", got)
	}
	// -1: everything.
	_, all, err := s.GetLayout(0, -1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(childLeaf(t, childLeaf(t, all.Children, 2).Children, 2).Children); got != 1 {
		t.Fatalf("depth -1 stopped early: %d", got)
	}
	// An id that is in no tree is still an answer, not an error.
	_, unknown, err := s.GetLayout(99, -1, nil)
	if err != nil || unknown.ID != 99 || len(unknown.Children) != 0 {
		t.Fatalf("unknown id %+v %v", unknown, err)
	}
}

func TestDbusMenuEventOnChildClicksChildNotParent(t *testing.T) {
	clicks := map[string]int{}
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menu:    nestedStatusMenu(clicks),
		menuRev: 1,
	}
	if err := s.Event(4, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if clicks["inbox"] != 1 {
		t.Fatalf("child id 4 must fire Inbox, got %v", clicks)
	}
	if err := s.Event(7, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if clicks["2025"] != 1 {
		t.Fatalf("grandchild id 7 must fire, got %v", clicks)
	}
	// A parent is not a command, even though this row was given OnClick.
	if err := s.Event(3, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(6, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	if clicks["folders"] != 0 {
		t.Fatalf("a cascade parent must not fire OnClick, got %v", clicks)
	}
	// The separator inside the submenu is inert too.
	if err := s.Event(5, "clicked", dbus.MakeVariant(""), 0); err != nil {
		t.Fatal(err)
	}
	ids, err := s.EventGroup([]dbusMenuEvent{
		{ID: 8, EventID: "clicked"},
		{ID: 42, EventID: "clicked"},
	})
	if err != nil || len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("EventGroup idErrors %v %v", ids, err)
	}
	if clicks["quit"] != 1 || clicks["show"] != 0 {
		t.Fatalf("clicks %v", clicks)
	}
}

func TestDbusMenuPropertiesResolveChildIDs(t *testing.T) {
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menu:    nestedStatusMenu(map[string]int{}),
		menuRev: 1,
	}
	v, err := s.GetProperty(7, "label")
	if err != nil {
		t.Fatal(err)
	}
	if v.Value() != "2025" {
		t.Fatalf("GetProperty(7, label) = %v", v.Value())
	}
	if v, err := s.GetProperty(3, "children-display"); err != nil || v.Value() != "submenu" {
		t.Fatalf("parent children-display %v %v", v.Value(), err)
	}
	props, err := s.GetGroupProperties([]int32{4, 6, 99}, []string{"label", "children-display"})
	if err != nil {
		t.Fatal(err)
	}
	if len(props) != 2 {
		t.Fatalf("group properties %+v want the two known ids", props)
	}
	if props[0].ID != 4 || props[0].Properties["label"].Value() != "Inbox" {
		t.Fatalf("child props %+v", props[0])
	}
	if _, ok := props[0].Properties["children-display"]; ok {
		t.Fatal("Inbox has no submenu")
	}
	if props[1].ID != 6 || props[1].Properties["children-display"].Value() != "submenu" {
		t.Fatalf("nested parent props %+v", props[1])
	}
}

func TestAboutToShowNeedsNoUpdateForParents(t *testing.T) {
	s := &linuxStatusItem{
		opts:    StatusItemOptions{MenuChrome: HostMenu},
		menu:    nestedStatusMenu(map[string]int{}),
		menuRev: 1,
	}
	// The layout was delivered in full by GetLayout, so no id needs a
	// re-fetch before its menu opens — including a submenu parent.
	for _, id := range []int32{0, 3, 6} {
		need, err := s.AboutToShow(id)
		if err != nil || need {
			t.Fatalf("AboutToShow(%d) = %v %v", id, need, err)
		}
	}
	updates, errIDs, err := s.AboutToShowGroup([]int32{0, 3, 77})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 0 {
		t.Fatalf("updatesNeeded %v", updates)
	}
	if len(errIDs) != 1 || errIDs[0] != 77 {
		t.Fatalf("idErrors %v want [77]", errIDs)
	}
}

func TestSetMenuNoticesASubmenuChange(t *testing.T) {
	s := &linuxStatusItem{menu: nestedStatusMenu(map[string]int{}), menuRev: 2}
	if err := s.SetMenu(nestedStatusMenu(map[string]int{})); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 2 {
		t.Fatalf("same tree must not bump rev, got %d", s.menuRev)
	}
	changed := nestedStatusMenu(map[string]int{})
	changed[2].Submenu[0].Text = "Unread"
	if err := s.SetMenu(changed); err != nil {
		t.Fatal(err)
	}
	if s.menuRev != 3 {
		t.Fatalf("a changed child row must bump rev, got %d", s.menuRev)
	}
}
