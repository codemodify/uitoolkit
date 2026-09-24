//go:build linux

package platform

import (
	"fmt"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/codemodify/uitoolkit/style"
)

const (
	sniPath        = "/StatusNotifierItem"
	sniInterface   = "org.kde.StatusNotifierItem"
	sniWatcher     = "org.kde.StatusNotifierWatcher"
	sniWatcherPath = "/StatusNotifierWatcher"
	fdoNotify      = "org.freedesktop.Notifications"
	fdoNotifyPath  = "/org/freedesktop/Notifications"
	dbusMenuPath   = "/MenuBar"
	dbusMenuIface  = "com.canonical.dbusmenu"
	// sniMenuNone is the KStatusNotifierItem / Plasma sentinel that
	// disables dbusmenu import and makes the host call ContextMenu.
	// "/" is the spec “empty” path — Plasma does not treat it as this
	// sentinel. Any other non-"/" path that is not /NO_DBUSMENU is
	// imported as dbusmenu (see statusnotifieritemsource.cpp).
	sniMenuNone = dbus.ObjectPath("/NO_DBUSMENU")
)

type linuxStatusItem struct {
	mu       sync.Mutex
	opts     StatusItemOptions
	icon     StatusIcon
	tooltip  string
	title    string
	menu     []StatusMenuItem
	menuRev  uint32
	conn     *dbus.Conn
	props    *prop.Properties
	name     string
	sni      bool
	closed   bool
	notifyID uint32
	noteFn   map[uint32]func()
}

func newNativeStatusItem(opts StatusItemOptions) (item StatusItem, err error) {
	var s *linuxStatusItem
	defer func() {
		if recover() != nil {
			if s != nil {
				_ = s.Close()
			}
			item = newStubStatusItem(opts)
			err = nil
		}
	}()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return newStubStatusItem(opts), nil
	}
	s = &linuxStatusItem{
		opts:    opts,
		icon:    opts.Icon,
		tooltip: opts.Tooltip,
		title:   opts.Title,
		menu:    copyMenu(opts.Menu),
		menuRev: 1,
		conn:    conn,
		noteFn:  map[uint32]func(){},
	}
	if err := s.export(); err != nil {
		_ = conn.Close()
		return newStubStatusItem(opts), nil
	}
	s.registerSNI()
	s.listenBus()
	return s, nil
}

func nativeStatusItemAvailable() bool {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (s *linuxStatusItem) export() error {
	id := s.opts.ID
	if id == "" {
		id = "uitoolkit"
	}
	name := fmt.Sprintf("org.freedesktop.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := s.conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		name = fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
		reply, err = s.conn.RequestName(name, dbus.NameFlagDoNotQueue)
		if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
			return fmt.Errorf("status: request name: %v", err)
		}
	}
	s.name = name

	propsSpec := map[string]map[string]*prop.Prop{
		sniInterface: {
			"Category":            {Value: "ApplicationStatus", Writable: false, Emit: prop.EmitFalse},
			"Id":                  {Value: id, Writable: false, Emit: prop.EmitFalse},
			"Title":               {Value: s.title, Writable: false, Emit: prop.EmitTrue},
			"Status":              {Value: "Active", Writable: false, Emit: prop.EmitTrue},
			"WindowId":            {Value: int32(0), Writable: false, Emit: prop.EmitFalse},
			"IconName":            {Value: statusIconName(s.icon), Writable: false, Emit: prop.EmitTrue},
			"IconPixmap":          {Value: s.iconPixmaps(), Writable: false, Emit: prop.EmitTrue},
			"OverlayIconName":     {Value: "", Writable: false, Emit: prop.EmitFalse},
			"OverlayIconPixmap":   {Value: []sniPixmap{}, Writable: false, Emit: prop.EmitFalse},
			"AttentionIconName":   {Value: "", Writable: false, Emit: prop.EmitFalse},
			"AttentionIconPixmap": {Value: []sniPixmap{}, Writable: false, Emit: prop.EmitFalse},
			"AttentionMovieName":  {Value: "", Writable: false, Emit: prop.EmitFalse},
			"ToolTip":             {Value: s.toolTip(), Writable: false, Emit: prop.EmitTrue},
			"ItemIsMenu":          {Value: s.opts.ItemIsMenu, Writable: false, Emit: prop.EmitFalse},
			"Menu":                {Value: s.menuPath(), Writable: false, Emit: prop.EmitTrue},
		},
	}
	props, err := prop.Export(s.conn, sniPath, propsSpec)
	if err != nil {
		return err
	}
	s.props = props

	if err := assertFiniteMenuSignature(); err != nil {
		return err
	}
	// Export only the SNI / dbusmenu methods. Exporting the whole
	// linuxStatusItem on both paths would advertise GetLayout on the
	// StatusNotifierItem interface (and vice versa).
	if err := s.conn.ExportMethodTable(map[string]interface{}{
		"ContextMenu":       s.ContextMenu,
		"Activate":          s.Activate,
		"SecondaryActivate": s.SecondaryActivate,
		"Scroll":            s.Scroll,
	}, sniPath, sniInterface); err != nil {
		return err
	}
	if err := s.conn.Export(introspect.Introspectable(sniIntrospect), sniPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return err
	}
	// /MenuBar is the HostMenu dbusmenu. ToolkitMenu still exports the
	// methods so probing hosts do not error; GetLayout is empty and SNI
	// Menu is /NO_DBUSMENU so Plasma calls ContextMenu instead.
	if err := s.conn.ExportMethodTable(map[string]interface{}{
		"GetLayout":          s.GetLayout,
		"GetGroupProperties": s.GetGroupProperties,
		"GetProperty":        s.GetProperty,
		"Event":              s.Event,
		"EventGroup":         s.EventGroup,
		"AboutToShow":        s.AboutToShow,
		"AboutToShowGroup":   s.AboutToShowGroup,
	}, dbusMenuPath, dbusMenuIface); err != nil {
		return err
	}
	if _, err := prop.Export(s.conn, dbusMenuPath, map[string]map[string]*prop.Prop{
		dbusMenuIface: {
			"Version":       {Value: uint32(3), Writable: false, Emit: prop.EmitFalse},
			"TextDirection": {Value: "ltr", Writable: false, Emit: prop.EmitFalse},
			"Status":        {Value: "normal", Writable: false, Emit: prop.EmitFalse},
			"IconThemePath": {Value: []string{}, Writable: false, Emit: prop.EmitFalse},
		},
	}); err != nil {
		return err
	}
	_ = s.conn.Export(introspect.Introspectable(dbusMenuIntrospect), dbusMenuPath, "org.freedesktop.DBus.Introspectable")
	s.sni = true
	trayDebug("export name=%s Menu=%s ItemIsMenu=%v hostMenu=%v", s.name, s.menuPath(), s.opts.ItemIsMenu, s.hostMenu())
	return nil
}

func (s *linuxStatusItem) registerSNI() {
	defer func() { recover() }()
	s.mu.Lock()
	conn := s.conn
	closed := s.closed
	s.mu.Unlock()
	if closed || conn == nil {
		return
	}
	obj := conn.Object(sniWatcher, sniWatcherPath)
	call := obj.Call(sniWatcher+".RegisterStatusNotifierItem", 0, s.name)
	if call.Err != nil {
		_ = obj.Call(sniWatcher+".RegisterStatusNotifierItem", 0, sniPath)
	}
}

func (s *linuxStatusItem) hostMenu() bool {
	return s.opts.MenuChrome != ToolkitMenu
}

func (s *linuxStatusItem) menuPath() dbus.ObjectPath {
	if s.hostMenu() {
		return dbus.ObjectPath(dbusMenuPath)
	}
	return sniMenuNone
}

func (s *linuxStatusItem) listenBus() {
	_ = s.conn.AddMatchSignal(
		dbus.WithMatchInterface(fdoNotify),
		dbus.WithMatchMember("ActionInvoked"),
	)
	_ = s.conn.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, sniWatcher),
	)
	ch := make(chan *dbus.Signal, 16)
	s.conn.Signal(ch)
	go func() {
		defer func() { recover() }()
		for sig := range ch {
			if sig == nil {
				continue
			}
			switch sig.Name {
			case fdoNotify + ".ActionInvoked":
				if len(sig.Body) < 1 {
					continue
				}
				id, _ := sig.Body[0].(uint32)
				s.mu.Lock()
				if _, sent := s.noteFn[id]; !sent && id != s.notifyID {
					// Another notification's click: the server tells every
					// listener, and this item did not send that one (a
					// Notifier of the same app, or another app).
					s.mu.Unlock()
					continue
				}
				fn := s.noteFn[id]
				if fn == nil {
					fn = s.opts.OnNotifyClick
				}
				if fn == nil {
					fn = s.opts.OnClick
				}
				dispatch := s.opts.Dispatch
				s.mu.Unlock()
				invokeStatus(dispatch, fn)
			case "org.freedesktop.DBus.NameOwnerChanged":
				if len(sig.Body) < 3 {
					continue
				}
				name, _ := sig.Body[0].(string)
				newOwner, _ := sig.Body[2].(string)
				s.onWatcherOwnerChanged(name, newOwner)
			}
		}
	}()
}

func (s *linuxStatusItem) onWatcherOwnerChanged(name, newOwner string) {
	if name != sniWatcher || newOwner == "" {
		return
	}
	trayDebug("StatusNotifierWatcher owner %s; re-register", newOwner)
	s.registerSNI()
}

type sniPixmap struct {
	W    int32
	H    int32
	Data []byte
}

type sniToolTip struct {
	IconName string
	Pixmaps  []sniPixmap
	Title    string
	Text     string
}

func (s *linuxStatusItem) iconPixmaps() []sniPixmap {
	s.mu.Lock()
	icon := s.icon
	s.mu.Unlock()
	return iconPixmapsOf(icon)
}

// iconPixmapsOf renders an icon without touching shared state, so a
// caller that already holds the lock (or has a snapshot) cannot race
// with SetIcon.
func iconPixmapsOf(icon StatusIcon) []sniPixmap {
	img := resolveStatusImage(icon, 22)
	w, h, pix := sniARGB(img)
	if w == 0 {
		return []sniPixmap{}
	}
	return []sniPixmap{{W: int32(w), H: int32(h), Data: pix}}
}

func (s *linuxStatusItem) toolTip() sniToolTip {
	s.mu.Lock()
	icon, title, tip := s.icon, s.title, s.tooltip
	s.mu.Unlock()
	return sniToolTip{
		IconName: statusIconName(icon),
		Pixmaps:  iconPixmapsOf(icon),
		Title:    title,
		Text:     tip,
	}
}

func (s *linuxStatusItem) emit(member string) {
	if s.conn == nil {
		return
	}
	_ = s.conn.Emit(sniPath, sniInterface+"."+member)
}

func (s *linuxStatusItem) SetIcon(icon StatusIcon) error {
	s.mu.Lock()
	s.icon = icon
	s.mu.Unlock()
	if s.props != nil {
		s.props.SetMust(sniInterface, "IconName", statusIconName(icon))
		s.props.SetMust(sniInterface, "IconPixmap", iconPixmapsOf(icon))
	}
	s.emit("NewIcon")
	return nil
}

func (s *linuxStatusItem) SetTooltip(text string) error {
	s.mu.Lock()
	s.tooltip = text
	s.mu.Unlock()
	if s.props != nil {
		s.props.SetMust(sniInterface, "ToolTip", s.toolTip())
	}
	s.emit("NewToolTip")
	return nil
}

func (s *linuxStatusItem) SetTitle(text string) error {
	s.mu.Lock()
	s.title = text
	s.mu.Unlock()
	if s.props != nil {
		s.props.SetMust(sniInterface, "Title", text)
	}
	s.emit("NewTitle")
	return nil
}

func (s *linuxStatusItem) SetMenu(items []StatusMenuItem) error {
	s.mu.Lock()
	if menuRowsEqual(s.menu, items) {
		s.mu.Unlock()
		return nil
	}
	s.menu = copyMenu(items)
	s.menuRev++
	rev := s.menuRev
	s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Emit(dbusMenuPath, dbusMenuIface+".LayoutUpdated", rev, int32(0))
	}
	return nil
}

func (s *linuxStatusItem) Notify(n Notification) error {
	if s.conn == nil {
		return nil
	}
	icon := n.Icon.Name
	if icon == "" {
		icon = statusIconName(s.icon)
	}
	hints := map[string]dbus.Variant{
		"desktop-entry": dbus.MakeVariant(s.opts.ID),
		"urgency":       dbus.MakeVariant(byte(1)),
	}
	obj := s.conn.Object(fdoNotify, fdoNotifyPath)
	var id uint32
	err := obj.Call(fdoNotify+".Notify", 0,
		s.opts.Title, s.notifyID, icon, n.Title, n.Body,
		[]string{"default", "Open"}, hints, int32(8000),
	).Store(&id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.notifyID = id
	if n.OnClick != nil {
		s.noteFn[id] = n.OnClick
	}
	s.mu.Unlock()
	return nil
}

func (s *linuxStatusItem) Close() error {
	s.mu.Lock()
	s.closed = true
	conn := s.conn
	s.conn = nil
	s.sni = false
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	return nil
}

func (s *linuxStatusItem) Backend() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sni {
		return "sni"
	}
	if s.conn != nil {
		return "fdo-notify"
	}
	return "stub"
}

func (s *linuxStatusItem) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed && s.conn != nil
}

// Activate implements StatusNotifierItem.Activate (primary click).
func (s *linuxStatusItem) Activate(x, y int32) *dbus.Error {
	s.mu.Lock()
	menuOnly := s.opts.ItemIsMenu
	fn := s.opts.OnClick
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	if menuOnly {
		if s.invokeOnMenu(x, y, "Activate") {
			return nil
		}
		// HostMenu: the host draws dbusmenu. Do not also fire OnClick.
		trayDebug("Activate ignored (ItemIsMenu)")
		return nil
	}
	invokeStatus(dispatch, fn)
	return nil
}

func (s *linuxStatusItem) invokeOnMenu(x, y int32, via string) bool {
	s.mu.Lock()
	fn := s.opts.OnMenu
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	if fn == nil {
		return false
	}
	trayDebug("%s → OnMenu(%d,%d)", via, x, y)
	invokeStatus(dispatch, func() { fn(x, y) })
	return true
}

// SecondaryActivate implements StatusNotifierItem.SecondaryActivate
// (middle-click). ToolkitMenu opens the popup. HostMenu does not
// fall through to Activate — some hosts emit this on right-click
// while also showing dbusmenu.
func (s *linuxStatusItem) SecondaryActivate(x, y int32) *dbus.Error {
	if s.invokeOnMenu(x, y, "SecondaryActivate") {
		return nil
	}
	trayDebug("SecondaryActivate ignored (no OnMenu)")
	return nil
}

// ContextMenu implements StatusNotifierItem.ContextMenu (right-click).
// ToolkitMenu sets Menu=/NO_DBUSMENU so Plasma calls this. HostMenu
// advertises /MenuBar and the host draws dbusmenu instead.
func (s *linuxStatusItem) ContextMenu(x, y int32) *dbus.Error {
	if s.hostMenu() {
		return nil
	}
	_ = s.invokeOnMenu(x, y, "ContextMenu")
	return nil
}

// Scroll implements StatusNotifierItem.Scroll.
func (s *linuxStatusItem) Scroll(delta int32, orientation string) *dbus.Error {
	_, _ = delta, orientation
	return nil
}

// GetLayout implements com.canonical.dbusmenu.
//
// The layout type is D-Bus (ia{sv}av). Children are []dbus.Variant wrapping
// leaf nodes — never a recursive Go struct. A []T of the same T makes
// godbus getSignature panic ("container nesting too deep") when Plasma's
// StatusNotifierWatcher calls GetLayout after RegisterStatusNotifierItem.
func (s *linuxStatusItem) GetLayout(parentID int32, recursionDepth int32, properties []string) (rev uint32, layout dbusMenuNode, derr *dbus.Error) {
	defer func() {
		if recover() != nil {
			rev = 1
			layout = emptyMenuLayout()
			derr = nil
		}
	}()
	_ = properties
	tree, rev, host := s.menuTree()
	if !host {
		trayDebug("dbusmenu.GetLayout parent=%d (ToolkitMenu empty; Menu=/NO_DBUSMENU)", parentID)
		return rev, emptyMenuLayout(), nil
	}
	trayDebug("dbusmenu.GetLayout parent=%d depth=%d rows=%d (HostMenu)", parentID, recursionDepth, len(tree.nodes))
	if parentID != 0 {
		if _, ok := tree.nodes[parentID]; !ok {
			return rev, unknownMenuNode(parentID), nil
		}
	}
	return rev, tree.layoutNode(parentID, recursionDepth), nil
}

// GetGroupProperties implements com.canonical.dbusmenu. Any id in the
// tree resolves, not only a top-level row.
func (s *linuxStatusItem) GetGroupProperties(ids []int32, properties []string) ([]dbusMenuProps, *dbus.Error) {
	tree, _, _ := s.menuTree()
	out := make([]dbusMenuProps, 0, len(ids))
	for _, id := range ids {
		node, ok := tree.nodes[id]
		if !ok {
			continue
		}
		props := menuItemProps(node.item)
		if len(properties) > 0 {
			filtered := make(map[string]dbus.Variant, len(properties))
			for _, name := range properties {
				if v, ok := props[name]; ok {
					filtered[name] = v
				}
			}
			props = filtered
		}
		out = append(out, dbusMenuProps{ID: id, Properties: props})
	}
	return out, nil
}

// GetProperty implements com.canonical.dbusmenu.GetProperty, for any id
// in the tree.
func (s *linuxStatusItem) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	tree, _, _ := s.menuTree()
	if node, ok := tree.nodes[id]; ok {
		if v, ok := menuItemProps(node.item)[name]; ok {
			return v, nil
		}
	}
	return dbus.MakeVariant(""), nil
}

// Event implements com.canonical.dbusmenu.Event.
func (s *linuxStatusItem) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	_, _ = data, timestamp
	if eventID != "clicked" {
		return nil
	}
	if !s.hostMenu() {
		_ = s.invokeOnMenu(0, 0, "dbusmenu.Event")
		return nil
	}
	tree, _, _ := s.menuTree()
	s.mu.Lock()
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	var fn func()
	// A cascade parent is not a command: menuItemClickable says no to it,
	// so a host that does send "clicked" for the row it opens a submenu
	// from (some do, alongside AboutToShow) fires nothing.
	if node, ok := tree.nodes[id]; ok && menuItemClickable(node.item) {
		fn = node.item.OnClick
	}
	if fn != nil {
		trayDebug("dbusmenu.Event clicked id=%d", id)
	}
	invokeStatus(dispatch, fn)
	return nil
}

// EventGroup implements com.canonical.dbusmenu.EventGroup. An id that is
// nowhere in the tree comes back in idErrors.
func (s *linuxStatusItem) EventGroup(events []dbusMenuEvent) ([]int32, *dbus.Error) {
	tree, _, _ := s.menuTree()
	var unknown []int32
	for _, ev := range events {
		if _, ok := tree.nodes[ev.ID]; !ok {
			unknown = append(unknown, ev.ID)
			continue
		}
		_ = s.Event(ev.ID, ev.EventID, ev.Data, ev.Timestamp)
	}
	return unknown, nil
}

// AboutToShow implements com.canonical.dbusmenu.AboutToShow.
//
// The answer is "no update needed" for every id, parents included. Hosts
// call this before opening a menu or a submenu, and the bool means "throw
// away the layout you have and fetch it again" — not "may I open?". This
// exporter hands out the whole tree in GetLayout and re-announces it with
// LayoutUpdated whenever SetMenu changes it, so a child menu is already
// on the host's side by the time it asks. Answering true would make every
// submenu hover cost a round of GetLayout for a layout that did not move.
// A menu built lazily per-open would need the opposite answer.
func (s *linuxStatusItem) AboutToShow(id int32) (bool, *dbus.Error) {
	_ = id
	if !s.hostMenu() {
		_ = s.invokeOnMenu(0, 0, "dbusmenu.AboutToShow")
	}
	return false, nil
}

// AboutToShowGroup implements com.canonical.dbusmenu.AboutToShowGroup:
// no id needs an update (see [linuxStatusItem.AboutToShow]), and an id
// that is not in the tree is reported in idErrors rather than silently
// counted as fine.
func (s *linuxStatusItem) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	if !s.hostMenu() {
		_ = s.invokeOnMenu(0, 0, "dbusmenu.AboutToShowGroup")
		return nil, nil, nil
	}
	tree, _, _ := s.menuTree()
	var unknown []int32
	for _, id := range ids {
		if id == 0 {
			continue // the root is always there
		}
		if _, ok := tree.nodes[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	return nil, unknown, nil
}

// dbusMenuNode is the dbusmenu layout struct: D-Bus type (ia{sv}av).
// Children must be []dbus.Variant (typically dbusMenuLeaf values), not
// []dbusMenuNode — that recursive Go type has no finite signature.
type dbusMenuNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

// dbusMenuLeaf is one child row. Same wire type (ia{sv}av) as the root,
// but a distinct Go type so SignatureOf cannot recurse even if someone
// later puts leaves in a typed slice.
type dbusMenuLeaf struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

const dbusMenuLayoutSig = "(ia{sv}av)"
const dbusMenuGetLayoutSig = "u(ia{sv}av)"

func emptyMenuLayout() dbusMenuNode {
	return dbusMenuNode{
		ID:         0,
		Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")},
		Children:   []dbus.Variant{},
	}
}

func unknownMenuNode(id int32) dbusMenuNode {
	return dbusMenuNode{
		ID:         id,
		Properties: map[string]dbus.Variant{},
		Children:   []dbus.Variant{},
	}
}

func menuItemProps(it StatusMenuItem) map[string]dbus.Variant {
	props := map[string]dbus.Variant{}
	if it.Separator {
		props["type"] = dbus.MakeVariant("separator")
		return props
	}
	props["label"] = dbus.MakeVariant(it.Text)
	if len(menuItemChildren(it)) > 0 {
		// The dbusmenu way of saying "this row opens a child menu"; hosts
		// draw the cascade arrow off this property and never send a
		// "clicked" for the row itself.
		props["children-display"] = dbus.MakeVariant("submenu")
	}
	if it.Disabled {
		props["enabled"] = dbus.MakeVariant(false)
	}
	if it.Checked {
		props["toggle-type"] = dbus.MakeVariant("checkmark")
		props["toggle-state"] = dbus.MakeVariant(int32(1))
	}
	if it.Icon != 0 {
		if name := style.ToolIconThemeName(it.Icon); name != "" {
			props["icon-name"] = dbus.MakeVariant(name)
		}
	}
	return props
}

// menuNode is one row of a flattened menu: the row itself and the ids of
// the rows it opens (empty for a command row).
type menuNode struct {
	item     StatusMenuItem
	children []int32
}

// menuTreeIndex is a whole menu addressed the way dbusmenu addresses it:
// by id. Id 0 is the menu root, which is not a row — its children are
// the top-level ids in roots.
type menuTreeIndex struct {
	nodes map[int32]menuNode
	roots []int32
}

// flattenMenu numbers a menu tree in pre-order — 1, 2, 3, …, a parent
// before the rows under it — and records each row's children.
//
// dbusmenu ids are flat: every row in the tree, at any depth, needs one
// id that is unique across the whole menu, and GetLayout, GetProperty,
// GetGroupProperties and Event all have to agree about which row an id
// means. The old rule (id == index + 1 in one flat slice) had no ids left
// for children and so could not describe a tree at all.
//
// Pre-order numbering is stable because it is a pure function of the menu:
// the same rows always produce the same ids, so the four methods agree
// without sharing state, and an id a host remembered from GetLayout still
// names that row when the click comes back. That is also why the index is
// rebuilt from the menu snapshot on each call instead of being cached
// beside s.menu: a cache would have to be invalidated everywhere s.menu
// is touched, and menus are a handful of rows.
//
// When the menu does change, SetMenu bumps menuRev and emits
// LayoutUpdated, which is how a host learns its remembered ids are stale.
func flattenMenu(items []StatusMenuItem) menuTreeIndex {
	tree := menuTreeIndex{nodes: map[int32]menuNode{}}
	next := int32(1)
	var walk func(rows []StatusMenuItem) []int32
	walk = func(rows []StatusMenuItem) []int32 {
		ids := make([]int32, 0, len(rows))
		for _, it := range rows {
			id := next
			next++
			ids = append(ids, id)
			node := menuNode{item: it}
			// The parent is numbered before its subtree, then updated:
			// walk allocates the children's ids.
			tree.nodes[id] = node
			if kids := menuItemChildren(it); len(kids) > 0 {
				node.children = walk(kids)
				tree.nodes[id] = node
			}
		}
		return ids
	}
	tree.roots = walk(items)
	return tree
}

// menuTree snapshots the menu as an id index, with the layout revision
// and whether this item draws its own menu (HostMenu).
func (s *linuxStatusItem) menuTree() (menuTreeIndex, uint32, bool) {
	s.mu.Lock()
	items := copyMenu(s.menu)
	rev := s.menuRev
	host := s.hostMenu()
	s.mu.Unlock()
	if rev == 0 {
		rev = 1
	}
	return flattenMenu(items), rev, host
}

// childIDs is the ids under id, with 0 meaning the menu root.
func (t menuTreeIndex) childIDs(id int32) []int32 {
	if id == 0 {
		return t.roots
	}
	return t.nodes[id].children
}

// layoutNode builds the layout subtree rooted at id (0 is the menu root).
//
// depth is the dbusmenu recursionDepth: -1 delivers every level, 0 this
// node alone with an empty child array, and n this node plus n levels
// below it. A host that asks for one level gets one level — it uses the
// answer to decide whether it still has to call GetLayout when a submenu
// opens.
func (t menuTreeIndex) layoutNode(id, depth int32) dbusMenuNode {
	node := dbusMenuNode{ID: id, Children: []dbus.Variant{}}
	if id == 0 {
		node.Properties = map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")}
	} else {
		node.Properties = menuItemProps(t.nodes[id].item)
	}
	if depth != 0 {
		node.Children = t.layoutChildren(t.childIDs(id), depth-1)
	}
	return node
}

// layoutChildren wraps each child as a dbusMenuLeaf variant. Leaves carry
// their own children the same way, which is how the tree is expressed
// without a recursive Go type (see [dbusMenuNode]): the nesting lives in
// the dbus.Variant values, whose signature is fixed at "v".
func (t menuTreeIndex) layoutChildren(ids []int32, depth int32) []dbus.Variant {
	out := make([]dbus.Variant, 0, len(ids))
	for _, id := range ids {
		leaf := dbusMenuLeaf{
			ID:         id,
			Properties: menuItemProps(t.nodes[id].item),
			Children:   []dbus.Variant{},
		}
		if depth != 0 {
			leaf.Children = t.layoutChildren(t.nodes[id].children, depth-1)
		}
		out = append(out, dbus.MakeVariant(leaf))
	}
	return out
}

// buildMenuLayout is the whole menu as one layout root (recursionDepth -1).
func buildMenuLayout(items []StatusMenuItem) dbusMenuNode {
	return flattenMenu(items).layoutNode(0, -1)
}

// assertFiniteMenuSignature is called before Export so a recursive layout
// type fails export (stub fallback) instead of panicking later in
// godbus (*Conn).handleCall when Plasma invokes GetLayout.
func assertFiniteMenuSignature() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("dbusmenu layout signature: %v", r)
		}
	}()
	node := dbus.SignatureOf(dbusMenuNode{})
	leaf := dbus.SignatureOf(dbusMenuLeaf{})
	if node.String() != dbusMenuLayoutSig {
		return fmt.Errorf("dbusmenu node signature %s, want %s", node, dbusMenuLayoutSig)
	}
	if leaf.String() != dbusMenuLayoutSig {
		return fmt.Errorf("dbusmenu leaf signature %s, want %s", leaf, dbusMenuLayoutSig)
	}
	// Nested on purpose: the depth that could reintroduce the recursive
	// type is a child of a child, not the top level.
	layout := buildMenuLayout([]StatusMenuItem{
		{Text: "Show"},
		{Separator: true},
		{Text: "Folders", Submenu: []StatusMenuItem{
			{Text: "Inbox"},
			{Text: "More", Submenu: []StatusMenuItem{{Text: "Archive"}}},
		}},
		{Text: "Quit", Checked: true},
	})
	got := dbus.SignatureOf(uint32(1), layout)
	if got.String() != dbusMenuGetLayoutSig {
		return fmt.Errorf("GetLayout signature %s, want %s", got, dbusMenuGetLayoutSig)
	}
	return nil
}

type dbusMenuProps struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type dbusMenuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

const sniIntrospect = `
<node>
  <interface name="org.kde.StatusNotifierItem">
    <method name="ContextMenu"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
    <method name="Activate"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
    <method name="SecondaryActivate"><arg name="x" type="i" direction="in"/><arg name="y" type="i" direction="in"/></method>
    <method name="Scroll"><arg name="delta" type="i" direction="in"/><arg name="orientation" type="s" direction="in"/></method>
    <property name="Category" type="s" access="read"/>
    <property name="Id" type="s" access="read"/>
    <property name="Title" type="s" access="read"/>
    <property name="Status" type="s" access="read"/>
    <property name="WindowId" type="i" access="read"/>
    <property name="IconName" type="s" access="read"/>
    <property name="IconPixmap" type="a(iiay)" access="read"/>
    <property name="ToolTip" type="(sa(iiay)ss)" access="read"/>
    <property name="ItemIsMenu" type="b" access="read"/>
    <property name="Menu" type="o" access="read"/>
    <signal name="NewTitle"/><signal name="NewIcon"/><signal name="NewToolTip"/><signal name="NewStatus"><arg name="status" type="s"/></signal>
  </interface>
  ` + introspect.IntrospectDataString + `
</node>`

const dbusMenuIntrospect = `
<node>
  <interface name="com.canonical.dbusmenu">
    <method name="GetLayout">
      <arg name="parentId" type="i" direction="in"/>
      <arg name="recursionDepth" type="i" direction="in"/>
      <arg name="propertyNames" type="as" direction="in"/>
      <arg name="revision" type="u" direction="out"/>
      <arg name="layout" type="(ia{sv}av)" direction="out"/>
    </method>
    <method name="GetGroupProperties">
      <arg name="ids" type="ai" direction="in"/>
      <arg name="propertyNames" type="as" direction="in"/>
      <arg name="properties" type="a(ia{sv})" direction="out"/>
    </method>
    <method name="GetProperty">
      <arg name="id" type="i" direction="in"/>
      <arg name="name" type="s" direction="in"/>
      <arg name="value" type="v" direction="out"/>
    </method>
    <method name="Event">
      <arg name="id" type="i" direction="in"/>
      <arg name="eventId" type="s" direction="in"/>
      <arg name="data" type="v" direction="in"/>
      <arg name="timestamp" type="u" direction="in"/>
    </method>
    <method name="AboutToShow">
      <arg name="id" type="i" direction="in"/>
      <arg name="needUpdate" type="b" direction="out"/>
    </method>
    <method name="EventGroup">
      <arg name="events" type="a(isvu)" direction="in"/>
      <arg name="idErrors" type="ai" direction="out"/>
    </method>
    <method name="AboutToShowGroup">
      <arg name="ids" type="ai" direction="in"/>
      <arg name="updatesNeeded" type="ai" direction="out"/>
      <arg name="idErrors" type="ai" direction="out"/>
    </method>
    <property name="Version" type="u" access="read"/>
    <property name="TextDirection" type="s" access="read"/>
    <property name="Status" type="s" access="read"/>
    <property name="IconThemePath" type="as" access="read"/>
    <signal name="LayoutUpdated">
      <arg name="revision" type="u"/>
      <arg name="parent" type="i"/>
    </signal>
  </interface>
</node>`
