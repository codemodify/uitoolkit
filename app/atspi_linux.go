//go:build linux

package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/godbus/dbus/v5"

	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// A11yEnv controls the AT-SPI2 bridge: "1" turns it on whatever the
// desktop says (tests, a screen reader started by hand), "0" keeps it off.
// Otherwise it follows org.a11y.Status, as Qt does: off until assistive
// technology is running, so apps pay nothing without it.
const A11yEnv = "UITK_A11Y"

const (
	atspiRootPath   = dbus.ObjectPath("/org/a11y/atspi/accessible/root")
	atspiPathPrefix = "/org/a11y/atspi/accessible/"
	atspiNullPath   = dbus.ObjectPath("/org/a11y/atspi/null")
	atspiRegistry   = "org.a11y.atspi.Registry"
	atspiIface      = "org.a11y.atspi."
	// atspiRefresh bounds how long a query waits for the UI goroutine.
	atspiRefresh = 2 * time.Second
)

// atspiRef is AT-SPI's object reference: a bus name and an object path.
type atspiRef struct {
	Name string
	Path dbus.ObjectPath
}

type atspiRect struct{ X, Y, W, H int32 }

type atspiRelation struct {
	Type    uint32
	Targets []atspiRef
}

type atspiActionInfo struct{ Name, Description, KeyBinding string }

// atspiObj is one exported object of a snapshot.
type atspiObj struct {
	node   *a11y.Node
	path   dbus.ObjectPath
	parent dbus.ObjectPath
	index  int
	kids   []dbus.ObjectPath
	win    *Window
}

// atspiBridge exports the app's accessibility trees on the AT-SPI bus.
type atspiBridge struct {
	app    *Application
	conn   *dbus.Conn
	name   string
	parent atspiRef
	appID  atomic.Int32

	mu    sync.Mutex
	objs  map[dbus.ObjectPath]*atspiObj
	stale atomic.Bool
	focus dbus.ObjectPath
	win   *Window
	// seen is the focused object as last described, to announce what
	// changes on it (a check box ticked by Space, a branch opened).
	seen *a11y.Node
}

// startA11y looks for assistive technology in the background and turns
// the bridge on when it runs (see A11yEnv).
func (a *Application) startA11y() {
	env := strings.TrimSpace(os.Getenv(A11yEnv))
	if env == "0" || (a.headless && env != "1") {
		return
	}
	go func() {
		// AT_SPI_BUS_ADDRESS names the accessibility bus directly, as
		// libatspi allows (a private test bus, a remote session).
		if addr := os.Getenv("AT_SPI_BUS_ADDRESS"); addr != "" && env == "1" {
			if b, err := newATSPIBridge(a, addr); err == nil {
				a.Post(func() { a.a11y = b; b.sync() })
				a.wakeUI()
			}
			return
		}
		sess, err := platform.SharedSessionBus()
		if err != nil {
			return
		}
		bus := sess.Object("org.a11y.Bus", "/org/a11y/bus")
		if env != "1" && !a11yEnabled(bus) {
			// Wait for a screen reader to start.
			match := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
				dbus.WithMatchMember("PropertiesChanged"), dbus.WithMatchObjectPath("/org/a11y/bus")}
			if sess.AddMatchSignal(match...) != nil {
				return
			}
			ch := make(chan *dbus.Signal, 4)
			sess.Signal(ch)
			for sig := range ch {
				if sig.Path == "/org/a11y/bus" && sig.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" && a11yEnabled(bus) {
					break
				}
			}
			sess.RemoveSignal(ch)
			_ = sess.RemoveMatchSignal(match...)
		}
		var addr string
		if err := bus.Call("org.a11y.Bus.GetAddress", 0).Store(&addr); err != nil || addr == "" {
			return
		}
		b, err := newATSPIBridge(a, addr)
		if err != nil {
			return
		}
		a.Post(func() { a.a11y = b; b.sync() })
		a.wakeUI()
	}()
}

func a11yEnabled(bus dbus.BusObject) bool {
	for _, p := range []string{"IsEnabled", "ScreenReaderEnabled"} {
		v, err := bus.GetProperty("org.a11y.Status." + p)
		if err == nil {
			if on, ok := v.Value().(bool); ok && on {
				return true
			}
		}
	}
	return false
}

func newATSPIBridge(a *Application, addr string) (*atspiBridge, error) {
	conn, err := dbus.Connect(addr)
	if err != nil {
		return nil, err
	}
	b := &atspiBridge{app: a, conn: conn, name: conn.Names()[0]}
	b.stale.Store(true)
	sub := dbus.ObjectPath(strings.TrimSuffix(atspiPathPrefix, "/"))
	for iface, v := range map[string]any{
		atspiIface + "Accessible":             atspiAccessible{b},
		atspiIface + "Application":            atspiApplication{b},
		atspiIface + "Component":              atspiComponent{b},
		atspiIface + "Action":                 atspiAction{b},
		atspiIface + "Text":                   atspiText{b},
		"org.freedesktop.DBus.Properties":     atspiProps{b},
		"org.freedesktop.DBus.Introspectable": atspiIntrospect{},
	} {
		if err := conn.ExportSubtree(v, sub, iface); err != nil {
			conn.Close()
			return nil, err
		}
	}
	// No cache: clients ask object by object (an empty GetItems says so).
	if err := conn.Export(atspiCache{}, "/org/a11y/atspi/cache", atspiIface+"Cache"); err != nil {
		conn.Close()
		return nil, err
	}
	// The registry embeds the app's root under the desktop.
	reg := conn.Object(atspiRegistry, atspiRootPath)
	if err := reg.Call(atspiIface+"Socket.Embed", 0, atspiRef{b.name, atspiRootPath}).Store(&b.parent); err != nil {
		conn.Close()
		return nil, err
	}
	return b, nil
}

// sync runs on the UI goroutine after each frame: the snapshot is stale,
// and a moved focus or active window is announced.
func (b *atspiBridge) sync() {
	b.stale.Store(true)
	var win *Window
	for _, w := range b.app.Windows() {
		if w.Active() {
			win = w
		}
	}
	if win != b.win {
		if b.win != nil {
			b.emit(b.windowPath(b.win), "Window", "Deactivate", "", 0)
		}
		if win != nil {
			b.emit(b.windowPath(win), "Window", "Activate", "", 0)
		}
		b.win = win
	}
	if win == nil {
		return
	}
	f := win.focus
	if win.popup != nil {
		f = widget.CascadeLeaf(win.popup)
	}
	var path dbus.ObjectPath
	if f != nil {
		path = b.pathOf(widget.FocusID(f))
	}
	if path == "" {
		return
	}
	now := widget.FocusNode(f)
	if path == b.focus {
		b.announce(path, b.seen, now)
		b.seen = now
		return
	}
	if b.focus != "" {
		b.emit(b.focus, "Object", "StateChanged", "focused", 0)
	}
	b.emit(path, "Object", "StateChanged", "focused", 1)
	b.focus, b.seen = path, now
}

// atspiWatched are the states whose changes a screen reader speaks.
var atspiWatched = []struct {
	s      a11y.State
	detail string
}{
	{a11y.StateChecked, "checked"},
	{a11y.StateMixed, "indeterminate"},
	{a11y.StateSelected, "selected"},
	{a11y.StateExpanded, "expanded"},
	{a11y.StatePressed, "pressed"},
	{a11y.StateDisabled, "sensitive"},
}

// announce emits what changed on the focused object between two frames.
func (b *atspiBridge) announce(path dbus.ObjectPath, was, now *a11y.Node) {
	if was == nil || now == nil || was.ID != now.ID {
		return
	}
	for _, w := range atspiWatched {
		if was.State.Has(w.s) == now.State.Has(w.s) {
			continue
		}
		on := now.State.Has(w.s)
		if w.s == a11y.StateDisabled {
			on = !on // AT-SPI says "sensitive"
		}
		v := int32(0)
		if on {
			v = 1
		}
		b.emit(path, "Object", "StateChanged", w.detail, v)
	}
	if was.Name != now.Name {
		b.emitProp(path, "accessible-name", now.Name)
	}
	if was.Value != now.Value || was.Now != now.Now {
		b.emitProp(path, "accessible-value", now.Value)
	}
}

func (b *atspiBridge) emitProp(path dbus.ObjectPath, prop, value string) {
	_ = b.conn.Emit(path, atspiIface+"Event.Object.PropertyChange", prop, int32(0), int32(0),
		dbus.MakeVariant(value), map[string]dbus.Variant{})
}

func (b *atspiBridge) emit(path dbus.ObjectPath, class, member, detail string, d1 int32) {
	_ = b.conn.Emit(path, atspiIface+"Event."+class+"."+member, detail, d1, int32(0),
		dbus.MakeVariant(int32(0)), map[string]dbus.Variant{})
}

func (b *atspiBridge) windowPath(w *Window) dbus.ObjectPath {
	if w.root == nil {
		return atspiRootPath
	}
	return b.pathOf(1<<63 | w.root.ID())
}

func (b *atspiBridge) pathOf(id uint64) dbus.ObjectPath {
	return dbus.ObjectPath(atspiPathPrefix + strconv.FormatUint(id, 10))
}

// refresh rebuilds the snapshot on the UI goroutine when it is stale.
func (b *atspiBridge) refresh() {
	if !b.stale.Load() {
		return
	}
	done := make(chan map[dbus.ObjectPath]*atspiObj, 1)
	b.app.Post(func() { done <- b.build() })
	select {
	case objs := <-done:
		b.mu.Lock()
		b.objs = objs
		b.mu.Unlock()
		b.stale.Store(false)
	case <-time.After(atspiRefresh):
	}
}

func (b *atspiBridge) build() map[dbus.ObjectPath]*atspiObj {
	objs := map[dbus.ObjectPath]*atspiObj{}
	name := filepath.Base(os.Args[0])
	root := &atspiObj{node: &a11y.Node{Role: a11y.RoleUnknown, Name: name}, path: atspiRootPath}
	objs[atspiRootPath] = root
	var add func(n *a11y.Node, parent *atspiObj, index int, w *Window)
	add = func(n *a11y.Node, parent *atspiObj, index int, w *Window) {
		o := &atspiObj{node: n, path: b.pathOf(n.ID), parent: parent.path, index: index, win: w}
		if _, dup := objs[o.path]; dup {
			return
		}
		objs[o.path] = o
		parent.kids = append(parent.kids, o.path)
		for i, c := range n.Children {
			add(c, o, i, w)
		}
	}
	for i, w := range b.app.Windows() {
		add(w.AccessibleTree(), root, i, w)
	}
	return objs
}

func (b *atspiBridge) obj(msg dbus.Message) (*atspiObj, *dbus.Error) {
	path, _ := msg.Headers[dbus.FieldPath].Value().(dbus.ObjectPath)
	b.refresh()
	b.mu.Lock()
	o := b.objs[path]
	b.mu.Unlock()
	if o == nil {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownObject", []any{string(path)})
	}
	return o, nil
}

func (b *atspiBridge) ref(p dbus.ObjectPath) atspiRef {
	if p == "" {
		return atspiRef{b.name, atspiNullPath}
	}
	return atspiRef{b.name, p}
}

// do performs a on o on the UI goroutine.
func (b *atspiBridge) do(o *atspiObj, a a11y.Action) bool {
	if o.win == nil {
		return false
	}
	done := make(chan bool, 1)
	b.app.Post(func() { done <- o.win.AccessibleAction(o.node.ID, a) })
	select {
	case ok := <-done:
		return ok
	case <-time.After(atspiRefresh):
		return false
	}
}

func (o *atspiObj) interfaces() []string {
	out := []string{atspiIface + "Accessible"}
	if o.path == atspiRootPath {
		return append(out, atspiIface+"Application")
	}
	out = append(out, atspiIface+"Component")
	if o.node.Actions != 0 {
		out = append(out, atspiIface+"Action")
	}
	if o.node.HasRange {
		out = append(out, atspiIface+"Value")
	}
	switch o.node.Role {
	case a11y.RoleTextField, a11y.RolePasswordField, a11y.RoleTextArea, a11y.RoleLabel, a11y.RoleHeading:
		out = append(out, atspiIface+"Text")
	}
	return out
}

func (o *atspiObj) actions() []a11y.Action {
	var out []a11y.Action
	for a := a11y.ActionDefault; a <= a11y.ActionScrollIntoView; a++ {
		if o.node.Actions.Has(a) {
			out = append(out, a)
		}
	}
	return out
}

// text is what the Text interface reads: a field's value, a label's name.
func (o *atspiObj) text() string {
	switch o.node.Role {
	case a11y.RoleLabel, a11y.RoleHeading:
		return o.node.Name
	case a11y.RolePasswordField:
		return strings.Repeat("•", utf8.RuneCountInString(o.node.Value))
	}
	return o.node.Value
}

// ---- org.a11y.atspi.Accessible ----------------------------------------------------

type atspiAccessible struct{ b *atspiBridge }

func (a atspiAccessible) GetChildAtIndex(msg dbus.Message, i int32) (atspiRef, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return atspiRef{}, err
	}
	if i < 0 || int(i) >= len(o.kids) {
		return a.b.ref(""), nil
	}
	return a.b.ref(o.kids[i]), nil
}

func (a atspiAccessible) GetChildren(msg dbus.Message) ([]atspiRef, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, err
	}
	out := make([]atspiRef, len(o.kids))
	for i, k := range o.kids {
		out[i] = a.b.ref(k)
	}
	return out, nil
}

func (a atspiAccessible) GetIndexInParent(msg dbus.Message) (int32, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return -1, err
	}
	if o.path == atspiRootPath {
		return -1, nil
	}
	return int32(o.index), nil
}

func (a atspiAccessible) GetRelationSet(msg dbus.Message) ([]atspiRelation, *dbus.Error) {
	if _, err := a.b.obj(msg); err != nil {
		return nil, err
	}
	return []atspiRelation{}, nil
}

func (a atspiAccessible) GetRole(msg dbus.Message) (uint32, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return 0, err
	}
	if o.path == atspiRootPath {
		return atspiRoleApplication, nil
	}
	return atspiRole(o.node.Role), nil
}

func (a atspiAccessible) GetRoleName(msg dbus.Message) (string, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return "", err
	}
	if o.path == atspiRootPath {
		return "application", nil
	}
	return atspiRoleNames[atspiRole(o.node.Role)], nil
}

func (a atspiAccessible) GetLocalizedRoleName(msg dbus.Message) (string, *dbus.Error) {
	return a.GetRoleName(msg)
}

func (a atspiAccessible) GetState(msg dbus.Message) ([]uint32, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, err
	}
	w := atspiStates(o.node)
	return w[:], nil
}

func (a atspiAccessible) GetAttributes(msg dbus.Message) (map[string]string, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, err
	}
	attrs := map[string]string{"toolkit": "uitoolkit"}
	if o.node.Level > 0 {
		attrs["level"] = strconv.Itoa(o.node.Level)
	}
	if o.node.Index > 0 {
		attrs["posinset"] = strconv.Itoa(o.node.Index)
		attrs["setsize"] = strconv.Itoa(o.node.Count)
	}
	return attrs, nil
}

func (a atspiAccessible) GetApplication(msg dbus.Message) (atspiRef, *dbus.Error) {
	return a.b.ref(atspiRootPath), nil
}

func (a atspiAccessible) GetInterfaces(msg dbus.Message) ([]string, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, err
	}
	return o.interfaces(), nil
}

// ---- org.a11y.atspi.Application ---------------------------------------------------

type atspiApplication struct{ b *atspiBridge }

func (a atspiApplication) GetLocale(msg dbus.Message, lctype uint32) (string, *dbus.Error) {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}
	}
	return "C", nil
}

// ---- org.a11y.atspi.Component -----------------------------------------------------

type atspiComponent struct{ b *atspiBridge }

// extents is o's box in logical pixels. Screen coordinates (0) are the
// window's too: a Wayland client cannot know where its window is.
func (o *atspiObj) extents() atspiRect {
	r := o.node.Bounds
	s := float32(1)
	if o.win != nil && o.win.scale > 0 {
		s = o.win.scale
	}
	return atspiRect{int32(r.Min.X / s), int32(r.Min.Y / s), int32(r.Dx() / s), int32(r.Dy() / s)}
}

func (c atspiComponent) GetExtents(msg dbus.Message, coord uint32) (atspiRect, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return atspiRect{}, err
	}
	return o.extents(), nil
}

func (c atspiComponent) GetPosition(msg dbus.Message, coord uint32) (int32, int32, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return 0, 0, err
	}
	e := o.extents()
	return e.X, e.Y, nil
}

func (c atspiComponent) GetSize(msg dbus.Message) (int32, int32, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return 0, 0, err
	}
	e := o.extents()
	return e.W, e.H, nil
}

func (c atspiComponent) Contains(msg dbus.Message, x, y int32, coord uint32) (bool, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return false, err
	}
	e := o.extents()
	return x >= e.X && y >= e.Y && x < e.X+e.W && y < e.Y+e.H, nil
}

func (c atspiComponent) GetAccessibleAtPoint(msg dbus.Message, x, y int32, coord uint32) (atspiRef, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return atspiRef{}, err
	}
	c.b.mu.Lock()
	defer c.b.mu.Unlock()
	// The deepest descendant under the point.
	hit := dbus.ObjectPath("")
	var walk func(p dbus.ObjectPath)
	walk = func(p dbus.ObjectPath) {
		for _, k := range c.b.objs[p].kids {
			ko := c.b.objs[k]
			e := ko.extents()
			if x >= e.X && y >= e.Y && x < e.X+e.W && y < e.Y+e.H {
				hit = k
				walk(k)
				return
			}
		}
	}
	walk(o.path)
	return c.b.ref(hit), nil
}

func (c atspiComponent) GetLayer(msg dbus.Message) (uint32, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return 0, err
	}
	if o.node.Role == a11y.RoleWindow || o.node.Role == a11y.RoleDialog {
		return 7, nil // ATSPI_LAYER_WINDOW
	}
	return 3, nil // ATSPI_LAYER_WIDGET
}

func (c atspiComponent) GetMDIZOrder(msg dbus.Message) (int16, *dbus.Error) { return 0, nil }

func (c atspiComponent) GetAlpha(msg dbus.Message) (float64, *dbus.Error) { return 1, nil }

func (c atspiComponent) GrabFocus(msg dbus.Message) (bool, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return false, err
	}
	return c.b.do(o, a11y.ActionFocus), nil
}

func (c atspiComponent) ScrollTo(msg dbus.Message, how uint32) (bool, *dbus.Error) {
	o, err := c.b.obj(msg)
	if err != nil {
		return false, err
	}
	return c.b.do(o, a11y.ActionScrollIntoView), nil
}

// ---- org.a11y.atspi.Action --------------------------------------------------------

type atspiAction struct{ b *atspiBridge }

func (a atspiAction) nth(msg dbus.Message, i int32) (*atspiObj, a11y.Action, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, 0, err
	}
	acts := o.actions()
	if i < 0 || int(i) >= len(acts) {
		return o, 0, dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"no such action"})
	}
	return o, acts[i], nil
}

func (a atspiAction) GetName(msg dbus.Message, i int32) (string, *dbus.Error) {
	o, act, err := a.nth(msg, i)
	if err != nil {
		return "", err
	}
	return atspiActionName(o.node, act), nil
}

func (a atspiAction) GetLocalizedName(msg dbus.Message, i int32) (string, *dbus.Error) {
	return a.GetName(msg, i)
}

func (a atspiAction) GetDescription(msg dbus.Message, i int32) (string, *dbus.Error) {
	return a.GetName(msg, i)
}

func (a atspiAction) GetKeyBinding(msg dbus.Message, i int32) (string, *dbus.Error) {
	o, _, err := a.nth(msg, i)
	if err != nil {
		return "", err
	}
	return o.node.Shortcut, nil
}

func (a atspiAction) GetActions(msg dbus.Message) ([]atspiActionInfo, *dbus.Error) {
	o, err := a.b.obj(msg)
	if err != nil {
		return nil, err
	}
	var out []atspiActionInfo
	for _, act := range o.actions() {
		n := atspiActionName(o.node, act)
		out = append(out, atspiActionInfo{n, n, o.node.Shortcut})
	}
	return out, nil
}

func (a atspiAction) DoAction(msg dbus.Message, i int32) (bool, *dbus.Error) {
	o, act, err := a.nth(msg, i)
	if err != nil {
		return false, err
	}
	return a.b.do(o, act), nil
}

// ---- org.a11y.atspi.Text ----------------------------------------------------------

type atspiText struct{ b *atspiBridge }

func runeSlice(s string, start, end int32) string {
	r := []rune(s)
	if end < 0 || int(end) > len(r) {
		end = int32(len(r))
	}
	if start < 0 {
		start = 0
	}
	if start > end {
		return ""
	}
	return string(r[start:end])
}

func (t atspiText) GetText(msg dbus.Message, start, end int32) (string, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return "", err
	}
	return runeSlice(o.text(), start, end), nil
}

func (t atspiText) GetCharacterAtOffset(msg dbus.Message, off int32) (int32, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return 0, err
	}
	r := []rune(o.text())
	if off < 0 || int(off) >= len(r) {
		return 0, nil
	}
	return r[off], nil
}

// boundary finds the unit (0 char, 1 word, 2 sentence, 3 line, 4 paragraph:
// AtspiTextGranularity) around off.
func boundary(s string, off int32, unit uint32) (string, int32, int32) {
	r := []rune(s)
	n := int32(len(r))
	if off < 0 {
		off = 0
	}
	if off > n {
		off = n
	}
	switch unit {
	case 0:
		if off >= n {
			return "", n, n
		}
		return string(r[off]), off, off + 1
	case 1:
		a, b := off, off
		for a > 0 && r[a-1] != ' ' && r[a-1] != '\n' {
			a--
		}
		for b < n && r[b] != ' ' && r[b] != '\n' {
			b++
		}
		return string(r[a:b]), a, b
	}
	a, b := off, off
	for a > 0 && r[a-1] != '\n' {
		a--
	}
	for b < n && r[b] != '\n' {
		b++
	}
	return string(r[a:b]), a, b
}

func (t atspiText) GetStringAtOffset(msg dbus.Message, off int32, unit uint32) (string, int32, int32, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return "", 0, 0, err
	}
	s, a, b := boundary(o.text(), off, unit)
	return s, a, b, nil
}

func (t atspiText) GetTextAtOffset(msg dbus.Message, off int32, kind uint32) (string, int32, int32, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return "", 0, 0, err
	}
	// AtspiTextBoundaryType: 0 char, 1-2 word, 3-4 sentence, 5-6 line.
	unit := uint32(0)
	switch {
	case kind == 1 || kind == 2:
		unit = 1
	case kind >= 3:
		unit = 3
	}
	s, a, b := boundary(o.text(), off, unit)
	return s, a, b, nil
}

func (t atspiText) GetNSelections(msg dbus.Message) (int32, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return 0, err
	}
	if o.node.SelEnd > o.node.SelStart {
		return 1, nil
	}
	return 0, nil
}

func (t atspiText) GetSelection(msg dbus.Message, i int32) (int32, int32, *dbus.Error) {
	o, err := t.b.obj(msg)
	if err != nil {
		return 0, 0, err
	}
	return int32(o.node.SelStart), int32(o.node.SelEnd), nil
}

// ---- properties -------------------------------------------------------------------

type atspiProps struct{ b *atspiBridge }

func (p atspiProps) props(o *atspiObj, iface string) map[string]dbus.Variant {
	n := o.node
	switch strings.TrimPrefix(iface, atspiIface) {
	case "Accessible":
		parent := p.b.ref(o.parent)
		if o.path == atspiRootPath {
			parent = p.b.parent
		}
		return map[string]dbus.Variant{
			"Name":         dbus.MakeVariant(n.Name),
			"Description":  dbus.MakeVariant(n.Description),
			"Parent":       dbus.MakeVariant(parent),
			"ChildCount":   dbus.MakeVariant(int32(len(o.kids))),
			"Locale":       dbus.MakeVariant(""),
			"AccessibleId": dbus.MakeVariant(strconv.FormatUint(n.ID, 10)),
			"HelpText":     dbus.MakeVariant(""),
		}
	case "Application":
		return map[string]dbus.Variant{
			"ToolkitName":  dbus.MakeVariant("uitoolkit"),
			"Version":      dbus.MakeVariant("1"),
			"AtspiVersion": dbus.MakeVariant("2.1"),
			"Id":           dbus.MakeVariant(p.b.appID.Load()),
		}
	case "Action":
		return map[string]dbus.Variant{"NActions": dbus.MakeVariant(int32(len(o.actions())))}
	case "Value":
		return map[string]dbus.Variant{
			"MinimumValue":     dbus.MakeVariant(n.Min),
			"MaximumValue":     dbus.MakeVariant(n.Max),
			"MinimumIncrement": dbus.MakeVariant(n.Step),
			"CurrentValue":     dbus.MakeVariant(n.Now),
			"Text":             dbus.MakeVariant(n.Value),
		}
	case "Text":
		return map[string]dbus.Variant{
			"CharacterCount": dbus.MakeVariant(int32(utf8.RuneCountInString(o.text()))),
			"CaretOffset":    dbus.MakeVariant(int32(n.Caret)),
		}
	}
	return map[string]dbus.Variant{}
}

func (p atspiProps) Get(msg dbus.Message, iface, name string) (dbus.Variant, *dbus.Error) {
	o, err := p.b.obj(msg)
	if err != nil {
		return dbus.Variant{}, err
	}
	if v, ok := p.props(o, iface)[name]; ok {
		return v, nil
	}
	return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{iface + "." + name})
}

func (p atspiProps) GetAll(msg dbus.Message, iface string) (map[string]dbus.Variant, *dbus.Error) {
	o, err := p.b.obj(msg)
	if err != nil {
		return nil, err
	}
	return p.props(o, iface), nil
}

func (p atspiProps) Set(msg dbus.Message, iface, name string, v dbus.Variant) *dbus.Error {
	// The registry numbers the application.
	if iface == atspiIface+"Application" && name == "Id" {
		if id, ok := v.Value().(int32); ok {
			p.b.appID.Store(id)
			return nil
		}
	}
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{name})
}

// atspiCacheItem is one entry of AT-SPI's cache (GetItems' signature
// a((so)(so)(so)iiassusau)).
type atspiCacheItem struct {
	Obj, App, Parent atspiRef
	Index, Children  int32
	Ifaces           []string
	Name             string
	Role             uint32
	Description      string
	States           []uint32
}

type atspiCache struct{}

func (atspiCache) GetItems() ([]atspiCacheItem, *dbus.Error) { return []atspiCacheItem{}, nil }

type atspiIntrospect struct{}

func (atspiIntrospect) Introspect(msg dbus.Message) (string, *dbus.Error) {
	return `<node><interface name="org.a11y.atspi.Accessible"/><interface name="org.a11y.atspi.Component"/></node>`, nil
}
