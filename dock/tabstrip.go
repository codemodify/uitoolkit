package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// stackStrip is the tab strip a stack grows when more than one panel is
// open in it: one tab per open panel, the current one selected. Dragging a
// tab docks that one panel elsewhere, leaving the rest of the stack where
// it is — as Qt Creator and VS Code do.
type stackStrip struct {
	widget.Base
	stack *Stack
	hover int
	press int
}

func newStackStrip(s *Stack) *stackStrip {
	t := &stackStrip{stack: s, hover: -1, press: -1}
	t.Init(t)
	t.SetWantsFocus(true)
	t.SetFocusVisibleOnly(true)
	return t
}

// tabs are the open panels the strip shows.
func (t *stackStrip) tabs() []*Panel { return t.stack.Open() }

// current is the index into tabs() of the panel the stack shows, or -1.
func (t *stackStrip) current() int {
	cur := t.stack.Current()
	for i, p := range t.tabs() {
		if p == cur {
			return i
		}
	}
	return -1
}

// rects are the tabs' boxes, an equal share of the strip clamped to what a
// label needs, and never thinner than the floor.
func (t *stackStrip) rects() []paintengine2d.Rect {
	tabs := t.tabs()
	if len(tabs) == 0 {
		return nil
	}
	b := t.LocalBounds()
	lk := t.Look()
	floor := style.Dip(lk, 44)
	w := b.Dx() / float32(len(tabs))
	if w < floor {
		w = floor
	}
	out := make([]paintengine2d.Rect, len(tabs))
	x := b.Min.X
	for i := range tabs {
		out[i] = paintengine2d.XYWH(x, b.Min.Y, w, b.Dy())
		x += w
	}
	return out
}

// indexAt is the tab at local point p, or -1. Later tabs win the border
// they share with the one before.
func (t *stackStrip) indexAt(p paintengine2d.Point) int {
	rects := t.rects()
	for i := len(rects) - 1; i >= 0; i-- {
		if rects[i].Contains(p) {
			return i
		}
	}
	return -1
}

func (t *stackStrip) Measure(c layout.Constraints) paintengine2d.Point {
	w := float32(120)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, t.stack.stripH()))
}

func (t *stackStrip) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *stackStrip) Paint(ctx *paintengine2d.Context) {
	b := t.LocalBounds()
	if b.Empty() {
		return
	}
	lk := t.Look()
	lk.DrawTabBar(ctx, b)
	tabs := t.tabs()
	rects := t.rects()
	cur := t.current()
	face := style.ControlFontOf(lk, style.RoleTab)
	draw := func(i int) {
		if i < 0 || i >= len(rects) {
			return
		}
		st := t.State() &^ (style.StateHovered | style.StatePressed | style.StateFocused)
		if t.hover == i {
			st |= style.StateHovered
		}
		if t.press == i {
			st |= style.StatePressed
		}
		if i == 0 {
			st |= style.StateFirst
		}
		if i == len(rects)-1 {
			st |= style.StateLast
		}
		if i == cur && t.Focused() && t.KeyNav() {
			st |= style.StateFocused
		}
		label := tabs[i].Title()
		if face != nil {
			label = face.Fit(label, rects[i].Dx()-style.Dip(lk, 10))
		}
		lk.DrawTab(ctx, rects[i], st, label, i == cur)
	}
	for i := range rects {
		if i != cur {
			draw(i)
		}
	}
	// The selected tab paints last so it may overlap its neighbours, as
	// every notebook look expects.
	draw(cur)
}

func (t *stackStrip) MouseMove(e widget.MouseEvent) bool {
	if host := hostOf(t); host != nil && host.dragArmed() {
		host.dragMove(widget.LocalToWindow(t, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
		return true
	}
	if i := t.indexAt(e.Pos); i != t.hover {
		t.hover = i
		t.Invalidate()
	}
	return t.hover >= 0
}

func (t *stackStrip) MouseExit() {
	if t.hover >= 0 {
		t.hover = -1
		t.Invalidate()
	}
	t.Base.MouseExit()
}

func (t *stackStrip) MousePress(e widget.MouseEvent) bool {
	i := t.indexAt(e.Pos)
	if i < 0 {
		return false
	}
	tabs := t.tabs()
	t.press = i
	t.stack.SelectPanel(tabs[i])
	t.stack.raise()
	t.MarkPointerFocus()
	t.Invalidate()
	// A press on a tab may become a drag of that one panel out of the
	// stack; the host decides once the pointer has moved far enough.
	if tabs[i].features&FeatureMovable != 0 {
		if host := hostOf(t); host != nil {
			host.beginDrag(t.stack, tabs[i], widget.LocalToWindow(t, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
		}
	}
	return true
}

func (t *stackStrip) MouseRelease(e widget.MouseEvent) bool {
	if host := hostOf(t); host != nil && host.dragArmed() {
		host.endDrag(widget.LocalToWindow(t, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
	}
	if t.press < 0 {
		return false
	}
	t.press = -1
	t.Invalidate()
	return true
}

// MouseWheel steps through the tabs, as a notebook does.
func (t *stackStrip) MouseWheel(e widget.MouseEvent) bool {
	if e.Scroll.Y == 0 || len(t.tabs()) < 2 {
		return false
	}
	step := 1
	if e.Scroll.Y < 0 {
		step = -1
	}
	t.selectTab(t.current() + step)
	return true
}

func (t *stackStrip) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() || len(t.tabs()) == 0 {
		return false
	}
	t.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyLeft:
		t.selectTab(t.current() - 1)
		return true
	case platform.KeyRight:
		t.selectTab(t.current() + 1)
		return true
	case platform.KeyHome:
		t.selectTab(0)
		return true
	case platform.KeyEnd:
		t.selectTab(len(t.tabs()) - 1)
		return true
	}
	return false
}

// selectTab shows tab i of the strip, clamped to the ones there are.
func (t *stackStrip) selectTab(i int) {
	tabs := t.tabs()
	if len(tabs) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= len(tabs) {
		i = len(tabs) - 1
	}
	t.stack.SelectPanel(tabs[i])
}

// Describe implements widget.Accessible.
func (t *stackStrip) Describe(n *a11y.Node) { n.Role = a11y.RoleTabList }

// AccessibleItems implements widget.AccessibleItems: one node per tab.
func (t *stackStrip) AccessibleItems() []*a11y.Node {
	tabs := t.tabs()
	rects := t.rects()
	cur := t.current()
	out := make([]*a11y.Node, 0, len(tabs))
	for i, p := range tabs {
		var r paintengine2d.Rect
		if i < len(rects) {
			r = rects[i]
		}
		n := &a11y.Node{
			ID:     widget.ItemID(t, i),
			Role:   a11y.RoleTab,
			Name:   widget.PlainText(p.Title()),
			Bounds: widget.LocalToWindow(t, r),
		}
		n.Index, n.Count = i+1, len(tabs)
		n.State |= a11y.StateSelectable
		if i == cur {
			n.State |= a11y.StateSelected
		}
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

// AccessibleAction implements widget.AccessibleActor: select a tab.
func (t *stackStrip) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault || i < 0 || i >= len(t.tabs()) || !t.Enabled() {
		return false
	}
	t.selectTab(i)
	return true
}

// AccessibleFocusItem implements widget.AccessibleFocusItem: the keyboard
// sits on the selected tab.
func (t *stackStrip) AccessibleFocusItem() int {
	if !t.Focused() {
		return -1
	}
	return t.current()
}
