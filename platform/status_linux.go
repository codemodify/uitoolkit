//go:build linux

package platform

import (
	"fmt"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
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
	s.listenNotifyActions()
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
			"IconName":            {Value: s.icon.Name, Writable: false, Emit: prop.EmitTrue},
			"IconPixmap":          {Value: s.iconPixmaps(), Writable: false, Emit: prop.EmitTrue},
			"OverlayIconName":     {Value: "", Writable: false, Emit: prop.EmitFalse},
			"OverlayIconPixmap":   {Value: []sniPixmap{}, Writable: false, Emit: prop.EmitFalse},
			"AttentionIconName":   {Value: "", Writable: false, Emit: prop.EmitFalse},
			"AttentionIconPixmap": {Value: []sniPixmap{}, Writable: false, Emit: prop.EmitFalse},
			"AttentionMovieName":  {Value: "", Writable: false, Emit: prop.EmitFalse},
			"ToolTip":             {Value: s.toolTip(), Writable: false, Emit: prop.EmitTrue},
			"ItemIsMenu":          {Value: false, Writable: false, Emit: prop.EmitFalse},
			// "/" so Plasma calls ContextMenu instead of drawing dbusmenu.
			"Menu": {Value: dbus.ObjectPath("/"), Writable: false, Emit: prop.EmitFalse},
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
	_ = s.conn.Export(introspect.Introspectable(dbusMenuIntrospect), dbusMenuPath, "org.freedesktop.DBus.Introspectable")
	s.sni = true
	return nil
}

func (s *linuxStatusItem) registerSNI() {
	defer func() { recover() }()
	obj := s.conn.Object(sniWatcher, sniWatcherPath)
	call := obj.Call(sniWatcher+".RegisterStatusNotifierItem", 0, s.name)
	if call.Err != nil {
		_ = obj.Call(sniWatcher+".RegisterStatusNotifierItem", 0, sniPath)
	}
}

func (s *linuxStatusItem) listenNotifyActions() {
	if err := s.conn.AddMatchSignal(
		dbus.WithMatchInterface(fdoNotify),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		return
	}
	ch := make(chan *dbus.Signal, 8)
	s.conn.Signal(ch)
	go func() {
		defer func() { recover() }()
		for sig := range ch {
			if sig == nil || sig.Name != fdoNotify+".ActionInvoked" || len(sig.Body) < 1 {
				continue
			}
			id, _ := sig.Body[0].(uint32)
			s.mu.Lock()
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
		}
	}()
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
	img := resolveStatusImage(s.icon, 22)
	w, h, pix := sniARGB(img)
	if w == 0 {
		return []sniPixmap{}
	}
	return []sniPixmap{{W: int32(w), H: int32(h), Data: pix}}
}

func (s *linuxStatusItem) toolTip() sniToolTip {
	return sniToolTip{
		IconName: s.icon.Name,
		Pixmaps:  s.iconPixmaps(),
		Title:    s.title,
		Text:     s.tooltip,
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
		s.props.SetMust(sniInterface, "IconName", icon.Name)
		s.props.SetMust(sniInterface, "IconPixmap", s.iconPixmaps())
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
		icon = s.icon.Name
	}
	if icon == "" {
		icon = "mail-unread"
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
	_, _ = x, y
	s.mu.Lock()
	fn := s.opts.OnClick
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	invokeStatus(dispatch, fn)
	return nil
}

// SecondaryActivate implements StatusNotifierItem.SecondaryActivate.
func (s *linuxStatusItem) SecondaryActivate(x, y int32) *dbus.Error {
	s.mu.Lock()
	fn := s.opts.OnMenu
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	if fn != nil {
		invokeStatus(dispatch, func() { fn(x, y) })
		return nil
	}
	return s.Activate(x, y)
}

// ContextMenu implements StatusNotifierItem.ContextMenu (right-click).
// Plasma is pointed at Menu="/" so it calls this instead of dbusmenu.
func (s *linuxStatusItem) ContextMenu(x, y int32) *dbus.Error {
	s.mu.Lock()
	fn := s.opts.OnMenu
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	if fn != nil {
		invokeStatus(dispatch, func() { fn(x, y) })
	}
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
	_, _ = recursionDepth, properties
	s.mu.Lock()
	items := copyMenu(s.menu)
	rev = s.menuRev
	s.mu.Unlock()
	if rev == 0 {
		rev = 1
	}
	if parentID > 0 {
		i := int(parentID) - 1
		if i >= 0 && i < len(items) {
			return rev, dbusMenuLeafNode(int32(i+1), items[i]), nil
		}
	}
	return rev, buildMenuLayout(items), nil
}

// GetGroupProperties implements com.canonical.dbusmenu.
func (s *linuxStatusItem) GetGroupProperties(ids []int32, properties []string) ([]dbusMenuProps, *dbus.Error) {
	_, _ = ids, properties
	return nil, nil
}

// GetProperty implements com.canonical.dbusmenu.GetProperty.
func (s *linuxStatusItem) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	_, _ = id, name
	return dbus.MakeVariant(""), nil
}

// Event implements com.canonical.dbusmenu.Event.
func (s *linuxStatusItem) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	_, _ = data, timestamp
	if eventID != "clicked" {
		return nil
	}
	s.mu.Lock()
	var fn func()
	i := int(id) - 1
	if i >= 0 && i < len(s.menu) {
		fn = s.menu[i].OnClick
	}
	dispatch := s.opts.Dispatch
	s.mu.Unlock()
	invokeStatus(dispatch, fn)
	return nil
}

// EventGroup implements com.canonical.dbusmenu.EventGroup.
func (s *linuxStatusItem) EventGroup(events []dbusMenuEvent) ([]int32, *dbus.Error) {
	for _, ev := range events {
		_ = s.Event(ev.ID, ev.EventID, ev.Data, ev.Timestamp)
	}
	return nil, nil
}

// AboutToShow implements com.canonical.dbusmenu.AboutToShow.
func (s *linuxStatusItem) AboutToShow(id int32) (bool, *dbus.Error) {
	_ = id
	return false, nil
}

// AboutToShowGroup implements com.canonical.dbusmenu.AboutToShowGroup.
func (s *linuxStatusItem) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	_ = ids
	return nil, nil, nil
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

func menuItemProps(it StatusMenuItem) map[string]dbus.Variant {
	props := map[string]dbus.Variant{}
	if it.Separator {
		props["type"] = dbus.MakeVariant("separator")
		return props
	}
	props["label"] = dbus.MakeVariant(it.Text)
	if it.Disabled {
		props["enabled"] = dbus.MakeVariant(false)
	}
	if it.Checked {
		props["toggle-type"] = dbus.MakeVariant("checkmark")
		props["toggle-state"] = dbus.MakeVariant(int32(1))
	}
	return props
}

func dbusMenuLeafNode(id int32, it StatusMenuItem) dbusMenuNode {
	return dbusMenuNode{
		ID:         id,
		Properties: menuItemProps(it),
		Children:   []dbus.Variant{},
	}
}

func buildMenuLayout(items []StatusMenuItem) dbusMenuNode {
	root := emptyMenuLayout()
	if len(items) == 0 {
		return root
	}
	children := make([]dbus.Variant, 0, len(items))
	for i, it := range items {
		children = append(children, dbus.MakeVariant(dbusMenuLeaf{
			ID:         int32(i + 1),
			Properties: menuItemProps(it),
			Children:   []dbus.Variant{},
		}))
	}
	root.Children = children
	return root
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
	layout := buildMenuLayout([]StatusMenuItem{
		{Text: "Show"},
		{Separator: true},
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
    <signal name="LayoutUpdated">
      <arg name="revision" type="u"/>
      <arg name="parent" type="i"/>
    </signal>
  </interface>
</node>`
