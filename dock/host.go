package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Features is what a panel's title bar offers, as QDockWidget's features
// are. The zero value offers nothing; [DefaultFeatures] is the usual set.
type Features uint8

const (
	// FeatureClosable shows the close button.
	FeatureClosable Features = 1 << iota
	// FeatureFloatable shows the float button and lets the panel be
	// dragged out into a window of its own.
	FeatureFloatable
	// FeatureMovable lets the panel be dragged to another place in the
	// host.
	FeatureMovable
	// FeatureCollapsible shows the twisty that hides the panel's content.
	FeatureCollapsible
)

// DefaultFeatures is what a new panel offers: close, float, move, collapse.
const DefaultFeatures = FeatureClosable | FeatureFloatable | FeatureMovable | FeatureCollapsible

// Host is the dock host: a central widget with dock areas on its four
// sides (Qt's QMainWindow, the editor and panels of VS Code).
//
// Its skeleton is fixed — a vertical split of the top area, the middle and
// the bottom area, the middle a horizontal split of the left area, the
// centre and the right area — so the sash between an area and the centre
// is the same [Split] sash as the one between two panels, with the same
// minimum sizes and the same look.
type Host struct {
	widget.Base
	centre *centrePane
	root   *Split
	middle *Split
	areas  [4]*Split
	panels []*Panel
	opener WindowOpener
	ind    *dropIndicator
	drag   *dragState
	// def is the arrangement ResetLayout goes back to.
	def *Layout

	// OnLayoutChanged runs after anything that changes the arrangement:
	// a dock, a float, a close, a sash moved. An app saves its layout here.
	OnLayoutChanged func()
}

// NewHost is a dock host around a central widget. centre may be nil — then
// the areas share the whole box between them.
func NewHost(centre widget.Component) *Host {
	h := &Host{}
	h.Init(h)
	h.centre = newCentrePane(centre)
	for s := range h.areas {
		h.areas[s] = NewSplit(Side(s).vertical())
	}
	h.middle = NewSplit(false, h.areas[SideLeft], h.centre, h.areas[SideRight])
	h.middle.weights = []float32{1, 4, 1}
	h.root = NewSplit(true, h.areas[SideTop], h.middle, h.areas[SideBottom])
	h.root.weights = []float32{1, 4, 1}
	h.ind = newDropIndicator()
	h.Add(h.root)
	h.Add(h.ind)
	return h
}

// Centre is the central widget.
func (h *Host) Centre() widget.Component { return h.centre.child }

// SetCentre replaces the central widget.
func (h *Host) SetCentre(c widget.Component) {
	h.centre.setChild(c)
	h.relayout()
}

// SetWindowOpener gives the host a way to open windows for floating
// panels. Without one, panels cannot float; app.DockWindows supplies it.
func (h *Host) SetWindowOpener(o WindowOpener) { h.opener = o }

// Area is the split holding the panels on one side of the centre.
func (h *Host) Area(s Side) *Split { return h.areas[sideIndex(s)] }

// Panels are every panel the host knows: docked, floating and closed.
func (h *Host) Panels() []*Panel { return h.panels }

// Panel is the panel with this name, or nil.
func (h *Host) Panel(name string) *Panel {
	for _, p := range h.panels {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// sideIndex keeps a bad Side from panicking the whole app.
func sideIndex(s Side) int {
	if s < 0 || int(s) >= 4 {
		return int(SideLeft)
	}
	return int(s)
}

// ---- docking ------------------------------------------------------------

// Dock puts p in a stack of its own at the end of the area on side. It is
// how an app builds its first arrangement.
func (h *Host) Dock(p *Panel, side Side) {
	if p == nil {
		return
	}
	h.adopt(p)
	h.detach(p)
	area := h.areas[sideIndex(side)]
	area.insert(len(area.kids), NewStack(p), 1)
	p.homeSide = side
	h.after()
}

// DockInto tabs p into target's stack, beside it.
func (h *Host) DockInto(p, target *Panel) {
	if p == nil || target == nil || p == target {
		return
	}
	st := target.Stack()
	if st == nil {
		return
	}
	h.adopt(p)
	h.detach(p)
	st.addPanel(p, st.IndexOf(target)+1)
	st.SelectPanel(p)
	st.sync()
	p.homeSide = h.sideOf(st)
	h.after()
}

// DockBeside splits target's stack and puts p on the given side of it:
// SideTop and SideBottom stack them, SideLeft and SideRight set them side
// by side.
func (h *Host) DockBeside(p, target *Panel, side Side) {
	if p == nil || target == nil || p == target {
		return
	}
	st := target.Stack()
	if st == nil {
		return
	}
	h.adopt(p)
	h.detach(p)
	h.splitAt(st, NewStack(p), side)
	p.homeSide = h.sideOf(st)
	h.after()
}

// splitAt puts add next to at, on the given side of it, splitting the
// parent when its axis does not already run that way.
func (h *Host) splitAt(at Node, add Node, side Side) {
	parent, i := h.parentOf(at)
	if parent == nil {
		return
	}
	wantVertical := side == SideTop || side == SideBottom
	before := side == SideTop || side == SideLeft
	if parent.vertical == wantVertical {
		j := i
		if !before {
			j = i + 1
		}
		parent.insert(j, add, 1)
		return
	}
	// The parent runs the other way: replace the pane with a split of the
	// two, keeping the share the pane had.
	w := parent.weights[i]
	parent.drop(at)
	inner := NewSplit(wantVertical)
	if before {
		inner.insert(0, add, 1)
		inner.insert(1, at, 1)
	} else {
		inner.insert(0, at, 1)
		inner.insert(1, add, 1)
	}
	parent.insert(i, inner, w)
}

// Undock takes p out of the tree, leaving it parentless. The stack it came
// from is dropped when it is left empty.
func (h *Host) Undock(p *Panel) {
	h.detach(p)
	h.after()
}

// detach takes p out of its stack and prunes whatever that leaves empty.
func (h *Host) detach(p *Panel) {
	st := p.Stack()
	if st == nil {
		return
	}
	p.homeSide = h.sideOf(st)
	p.homeStack = st
	st.removePanel(p)
	if len(st.panels) == 0 {
		p.homeStack = nil
		h.prune(st)
	}
}

// prune drops an empty node and every split it leaves with nothing in it,
// so the tree never keeps hollow branches. Area roots stay: they are the
// skeleton.
func (h *Host) prune(n Node) {
	for n != nil {
		if h.isSkeleton(n) {
			return
		}
		parent, _ := h.parentOf(n)
		if parent == nil {
			return
		}
		parent.drop(n)
		if len(parent.kids) == 1 && !h.isSkeleton(parent) {
			// A split with one pane left is no split at all: lift the pane
			// into its place.
			h.collapseSplit(parent)
			return
		}
		if len(parent.kids) > 0 {
			return
		}
		n = parent
	}
}

// collapseSplit lifts a split's one remaining pane into the split's place.
func (h *Host) collapseSplit(sp *Split) {
	if len(sp.kids) != 1 {
		return
	}
	only := sp.kids[0]
	parent, i := h.parentOf(sp)
	if parent == nil {
		return
	}
	w := parent.weights[i]
	sp.drop(only)
	parent.drop(sp)
	parent.insert(i, only, w)
}

// isSkeleton reports whether n is one of the fixed splits the host is
// built from, which are never pruned.
func (h *Host) isSkeleton(n Node) bool {
	if n == h.root || n == h.middle || n == Node(h.centre) {
		return true
	}
	for _, a := range h.areas {
		if n == Node(a) {
			return true
		}
	}
	return false
}

// parentOf is the split n hangs from, and n's index in it.
func (h *Host) parentOf(n Node) (*Split, int) {
	var found *Split
	idx := -1
	var walk func(*Split)
	walk = func(sp *Split) {
		if found != nil || sp == nil {
			return
		}
		for i, k := range sp.kids {
			if k == n {
				found, idx = sp, i
				return
			}
			if inner, ok := k.(*Split); ok {
				walk(inner)
			}
		}
	}
	walk(h.root)
	return found, idx
}

// sideOf is the area n sits in, and SideLeft when it is not in one.
func (h *Host) sideOf(n Node) Side {
	for s, a := range h.areas {
		if a == n || nodeContains(a, n) {
			return Side(s)
		}
	}
	return SideLeft
}

// nodeContains reports whether n is root or somewhere under it.
func nodeContains(root *Split, n Node) bool {
	for _, k := range root.kids {
		if k == n {
			return true
		}
		if inner, ok := k.(*Split); ok && nodeContains(inner, n) {
			return true
		}
	}
	return false
}

// adopt records a panel the host has not seen before.
func (h *Host) adopt(p *Panel) {
	for _, q := range h.panels {
		if q == p {
			return
		}
	}
	if p.features == 0 {
		p.features = DefaultFeatures
	}
	p.owner = h
	h.panels = append(h.panels, p)
}

// stacks are every stack in the tree, in tree order.
func (h *Host) stacks() []*Stack {
	var out []*Stack
	var walk func(Node)
	walk = func(n Node) {
		switch v := n.(type) {
		case *Stack:
			out = append(out, v)
		case *Split:
			for _, k := range v.kids {
				walk(k)
			}
		}
	}
	walk(h.root)
	return out
}

// after re-lays out the host and tells the app the arrangement changed.
func (h *Host) after() {
	h.relayout()
	h.layoutChanged()
}

// relayout re-measures and repaints the host.
func (h *Host) relayout() {
	h.RequestLayout()
	h.Invalidate()
	if b := h.Bounds(); !b.Empty() {
		h.Arrange(b)
	}
}

// layoutChanged tells the app the arrangement is worth saving again.
func (h *Host) layoutChanged() {
	if h.OnLayoutChanged != nil {
		h.OnLayoutChanged()
	}
}

// ---- layout -------------------------------------------------------------

func (h *Host) Measure(c layout.Constraints) paintengine2d.Point {
	m := h.root.MinSize()
	w, hh := m.X, m.Y
	if c.HasMaxW() {
		w = c.MaxW
	}
	if c.HasMaxH() {
		hh = c.MaxH
	}
	return c.Constrain(paintengine2d.Pt(w, hh))
}

func (h *Host) Arrange(r paintengine2d.Rect) {
	h.SetBounds(r)
	b := h.LocalBounds()
	h.root.Arrange(b)
	h.ind.Arrange(b)
}

// Paint fills the host with the look's window background, so the gaps
// between areas are the pack's own colour.
func (h *Host) Paint(ctx *paintengine2d.Context) {
	b := h.LocalBounds()
	if b.Empty() {
		return
	}
	lk := h.Look()
	if bg, ok := lk.(style.WindowBackgroundLook); ok {
		bg.DrawWindowBackground(ctx, b)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Background))
}

// Describe implements widget.Accessible: the host is the pane every dock
// area hangs from.
func (h *Host) Describe(n *a11y.Node) {
	n.Role = a11y.RolePane
	if n.Name == "" {
		n.Name = "Dock host"
	}
}

// ---- keyboard -----------------------------------------------------------

// KeyPress takes the shortcuts that work anywhere in the host: F6 moves
// the focus from one dock area to the next (Shift+F6 back, as Windows and
// Qt Creator do), Ctrl+W closes the panel the focus is in, and Escape
// gives up a drag.
func (h *Host) KeyPress(e widget.KeyEvent) bool {
	switch {
	case e.Key == platform.KeyEscape && h.dragging():
		h.cancelDrag()
		return true
	case e.Key == platform.KeyF6 && !e.Mods.Ctrl():
		return h.cyclePanel(!e.Mods.Shift())
	case e.Key == platform.KeyW && e.Mods.Ctrl():
		if p := h.FocusedPanel(); p != nil && p.features&FeatureClosable != 0 {
			return p.Close()
		}
	}
	return false
}

// FocusedPanel is the panel the keyboard is in, or nil. The focus counts as
// the panel's when it is on the panel's own chrome too — its title bar or
// its tab — so Ctrl+W closes the panel whose close button is focused.
func (h *Host) FocusedPanel() *Panel {
	f := widget.FocusOwner(h)
	if f == nil {
		return nil
	}
	for c := f; c != nil; c = c.Parent() {
		switch v := c.(type) {
		case *Panel:
			return v
		case *Stack:
			return v.Current()
		}
	}
	return nil
}

// cyclePanel moves the focus to the next open panel in tree order, or the
// one before it. It reports whether the focus moved.
func (h *Host) cyclePanel(forward bool) bool {
	var open []*Panel
	for _, st := range h.stacks() {
		if cur := st.Current(); cur != nil {
			open = append(open, cur)
		}
	}
	if len(open) == 0 {
		return false
	}
	at := -1
	if cur := h.FocusedPanel(); cur != nil {
		for i, p := range open {
			if p == cur {
				at = i
			}
		}
	}
	step := 1
	if !forward {
		step = -1
	}
	next := (at + step + len(open)) % len(open)
	target := open[next]
	if st := target.Stack(); st != nil && !widget.FocusFirstIn(target) {
		st.head.RequestFocus()
		widget.MarkKeyboardFocus(st.head)
	}
	return true
}

// ---- the centre ---------------------------------------------------------

// centrePane is the host's central widget as a node of the tree, so the
// sash between it and an area is an ordinary split sash.
type centrePane struct {
	widget.Base
	child widget.Component
}

func newCentrePane(child widget.Component) *centrePane {
	c := &centrePane{}
	c.Init(c)
	c.setChild(child)
	return c
}

func (c *centrePane) dockNode() {}

func (c *centrePane) setChild(child widget.Component) {
	if c.child != nil {
		c.Remove(c.child)
	}
	c.child = child
	if child != nil {
		c.Add(child)
	}
}

// Empty is true without a central widget, so a host that is all panels
// gives the whole box to its areas.
func (c *centrePane) Empty() bool { return c.child == nil }

// Fixed is never set: the centre takes whatever the areas leave.
func (c *centrePane) Fixed(bool) (float32, bool) { return 0, false }

// MinSize keeps room for the central widget however far the panels are
// dragged in.
func (c *centrePane) MinSize() paintengine2d.Point {
	if c.child == nil {
		return paintengine2d.Point{}
	}
	lk := c.Look()
	return paintengine2d.Pt(style.Dip(lk, 120), style.Dip(lk, 80))
}

func (c *centrePane) Measure(con layout.Constraints) paintengine2d.Point {
	if c.child == nil {
		return con.Constrain(paintengine2d.Point{})
	}
	return c.child.Measure(con)
}

func (c *centrePane) Arrange(r paintengine2d.Rect) {
	c.SetBounds(r)
	if c.child != nil {
		c.child.Arrange(c.LocalBounds())
	}
}
