package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Stack is one box of the dock tree: the panels dropped onto each other,
// shown one at a time under a shared title bar, with a tab strip when
// there is more than one (Qt's tabified docks, a VS Code view container).
//
// The stack is the leaf of an area's tree; a lone panel is a stack of one.
type Stack struct {
	widget.Base
	panels  []*Panel
	current int
	head    *stackHead
	strip   *stackStrip
	// wasActive is the last state the title bar was painted in, so a
	// focus move that does not cross the stack's edge repaints nothing.
	wasActive bool
}

// NewStack is a stack of panels, the first one current.
func NewStack(panels ...*Panel) *Stack {
	s := &Stack{}
	s.Init(s)
	s.head = newStackHead(s)
	s.strip = newStackStrip(s)
	s.Add(s.head)
	s.Add(s.strip)
	for _, p := range panels {
		s.addPanel(p, len(s.panels))
	}
	s.sync()
	return s
}

func (s *Stack) dockNode() {}

// Panels are the stack's panels, closed ones included, in tab order.
func (s *Stack) Panels() []*Panel { return s.panels }

// Open are the panels that are not closed, in tab order.
func (s *Stack) Open() []*Panel {
	out := make([]*Panel, 0, len(s.panels))
	for _, p := range s.panels {
		if !p.closed {
			out = append(out, p)
		}
	}
	return out
}

// Current is the panel the stack shows, or nil when every panel in it is
// closed.
func (s *Stack) Current() *Panel {
	if s.current < 0 || s.current >= len(s.panels) {
		return nil
	}
	if s.panels[s.current].closed {
		return nil
	}
	return s.panels[s.current]
}

// Select shows panel i of the stack (an index into Panels).
func (s *Stack) Select(i int) {
	if i < 0 || i >= len(s.panels) || s.panels[i].closed || i == s.current {
		return
	}
	s.current = i
	s.sync()
	s.RequestLayout()
	s.Invalidate()
	if h := hostOf(s); h != nil {
		h.relayout()
	}
}

// SelectPanel shows p, which must be in the stack.
func (s *Stack) SelectPanel(p *Panel) {
	for i, q := range s.panels {
		if q == p {
			s.Select(i)
			return
		}
	}
}

// IndexOf is p's position in the stack, or -1.
func (s *Stack) IndexOf(p *Panel) int {
	for i, q := range s.panels {
		if q == p {
			return i
		}
	}
	return -1
}

// addPanel puts p at index i without touching the layout.
func (s *Stack) addPanel(p *Panel, i int) {
	if p == nil {
		return
	}
	if i < 0 || i > len(s.panels) {
		i = len(s.panels)
	}
	s.panels = append(s.panels, nil)
	copy(s.panels[i+1:], s.panels[i:])
	s.panels[i] = p
	s.Add(p)
	if s.current >= i {
		s.current++
	}
	if s.current < 0 || s.current >= len(s.panels) {
		s.current = i
	}
}

// removePanel takes p out and reports whether it was there.
func (s *Stack) removePanel(p *Panel) bool {
	i := s.IndexOf(p)
	if i < 0 {
		return false
	}
	s.panels = append(s.panels[:i], s.panels[i+1:]...)
	s.Remove(p)
	if s.current >= len(s.panels) {
		s.current = len(s.panels) - 1
	}
	if s.current < 0 && len(s.panels) > 0 {
		s.current = 0
	}
	s.pickOpen()
	s.sync()
	return true
}

// pickOpen moves the current index onto an open panel when the one it is
// on has closed.
func (s *Stack) pickOpen() {
	if s.current >= 0 && s.current < len(s.panels) && !s.panels[s.current].closed {
		return
	}
	for i, p := range s.panels {
		if !p.closed {
			s.current = i
			return
		}
	}
}

// panelClosedChanged re-picks the current panel after one was closed or
// shown again.
func (s *Stack) panelClosedChanged(*Panel) {
	s.pickOpen()
	s.sync()
	s.Invalidate()
}

// sync shows only the current panel and hides the tab strip when the stack
// holds a single open panel.
func (s *Stack) sync() {
	cur := s.Current()
	for _, p := range s.panels {
		p.SetVisible(p == cur)
	}
	s.strip.SetVisible(len(s.Open()) > 1)
	if cur != nil {
		s.head.SetVisible(true)
	}
}

// Empty reports whether every panel in the stack is closed.
func (s *Stack) Empty() bool { return s.Current() == nil }

// collapsed reports whether the stack shows only its chrome.
func (s *Stack) collapsed() bool {
	cur := s.Current()
	return cur != nil && cur.collapsed
}

// headH is the title bar's height.
func (s *Stack) headH() float32 {
	lk := s.Look()
	h := lk.Metrics().AccordionH
	if min := style.Dip(lk, 22); h < min {
		h = min
	}
	return h
}

// stripH is the tab strip's height, zero when it is hidden.
func (s *Stack) stripH() float32 {
	if !s.strip.Visible() {
		return 0
	}
	lk := s.Look()
	h := lk.Metrics().TabH
	if min := style.Dip(lk, 20); h < min {
		h = min
	}
	return h
}

// chromeH is the room the title bar and the tab strip take together.
func (s *Stack) chromeH() float32 { return s.headH() + s.stripH() }

// MinSize is the chrome plus the current panel's own minimum, or just the
// chrome while the stack is collapsed.
func (s *Stack) MinSize() paintengine2d.Point {
	h := s.chromeH()
	if s.collapsed() {
		return paintengine2d.Pt(style.Dip(s.Look(), 80), h)
	}
	cur := s.Current()
	if cur == nil {
		return paintengine2d.Pt(0, h)
	}
	m := cur.MinContent()
	return paintengine2d.Pt(m.X, m.Y+h)
}

// Fixed keeps a collapsed stack at its chrome height, so neighbours take
// the space it gave up instead of sharing it out again.
func (s *Stack) Fixed(vertical bool) (float32, bool) {
	if vertical && s.collapsed() {
		return s.chromeH(), true
	}
	return 0, false
}

func (s *Stack) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(s.MinSize())
}

func (s *Stack) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	b := s.LocalBounds()
	y := b.Min.Y
	hh := s.headH()
	if hh > b.Dy() {
		hh = b.Dy()
	}
	s.head.Arrange(paintengine2d.XYWH(b.Min.X, y, b.Dx(), hh))
	y += hh
	if sh := s.stripH(); sh > 0 {
		if y+sh > b.Max.Y {
			sh = b.Max.Y - y
		}
		s.strip.Arrange(paintengine2d.XYWH(b.Min.X, y, b.Dx(), sh))
		y += sh
	}
	cur := s.Current()
	if cur == nil {
		return
	}
	if s.collapsed() {
		cur.Arrange(paintengine2d.XYWH(b.Min.X, y, b.Dx(), 0))
		return
	}
	cur.Arrange(paintengine2d.XYWH(b.Min.X, y, b.Dx(), maxF(0, b.Max.Y-y)))
}

// Paint draws the box the panels sit in; the title bar, tab strip and the
// current panel paint themselves as children.
func (s *Stack) Paint(ctx *paintengine2d.Context) {
	b := s.LocalBounds()
	if b.Empty() {
		return
	}
	lk := s.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Surface))
	if bw := lk.Metrics().Border; bw > 0 && lk.Palette().Border.A > 0 {
		ctx.DrawRect(b.Inset(bw*0.5), paintengine2d.StrokePaint(lk.Palette().Border, bw))
	}
}

// active reports whether the keyboard focus is inside the stack, so its
// title bar marks the pane the keyboard is in (Qt Creator and VS Code both
// mark theirs).
func (s *Stack) active() bool { return widget.FocusWithin(s) }

// FocusMoved implements widget.FocusWatcher: the title bar is repainted
// when the focus crosses the stack's edge, since nothing else would ask
// it to — the two components at either end of the move repaint
// themselves, and the bar is neither.
func (s *Stack) FocusMoved(now widget.Component) {
	in := now != nil && widget.Contains(s, now)
	if in == s.wasActive {
		return
	}
	s.wasActive = in
	s.head.Invalidate()
}

// raise makes the stack the one the keyboard is in, moving the focus into
// the current panel's content when it is not already inside.
func (s *Stack) raise() {
	cur := s.Current()
	if cur == nil || widget.FocusWithin(s) {
		return
	}
	if !widget.FocusFirstIn(cur) {
		s.head.RequestFocus()
	}
	s.Invalidate()
}

// Describe implements widget.Accessible: a stack of one panel is a plain
// pane, a tabbed stack the panel behind its tabs.
func (s *Stack) Describe(n *a11y.Node) {
	n.Role = a11y.RolePane
	if len(s.Open()) > 1 {
		n.Role = a11y.RoleTabPanel
	}
	if n.Name == "" {
		if cur := s.Current(); cur != nil {
			n.Name = widget.PlainText(cur.Title())
		}
	}
}

// ---- the title bar ------------------------------------------------------

// headPart is one hit target of a stack's title bar.
type headPart uint8

const (
	headNone headPart = iota
	headGrip          // the title itself: press and drag docks the panel elsewhere
	headCollapse
	headFloat
	headClose
)

// headButtons are the title bar's buttons, right to left as they are laid
// out.
var headButtons = [...]headPart{headClose, headFloat, headCollapse}

// stackHead is a stack's title bar: the current panel's name, the drag
// handle, and the collapse, float and close buttons (QDockWidget's title
// bar, a VS Code view header).
type stackHead struct {
	widget.Base
	stack *Stack
	hover headPart
	press headPart
	// focus is the button the keyboard is on, an index into the buttons
	// this title bar shows; -1 when none is.
	focus int
}

func newStackHead(s *Stack) *stackHead {
	h := &stackHead{stack: s, hover: headNone, press: headNone, focus: -1}
	h.Init(h)
	h.SetWantsFocus(true)
	h.SetFocusVisibleOnly(true)
	return h
}

// FocusOnClick is false: clicking the title bar starts a drag and must not
// move the focus ring onto it.
func (h *stackHead) FocusOnClick() bool { return false }

// buttons are the parts this title bar shows, in layout order from the
// right edge.
func (h *stackHead) buttons() []headPart {
	cur := h.stack.Current()
	if cur == nil {
		return nil
	}
	out := make([]headPart, 0, len(headButtons))
	for _, p := range headButtons {
		switch p {
		case headClose:
			if cur.features&FeatureClosable == 0 {
				continue
			}
		case headFloat:
			if cur.features&FeatureFloatable == 0 || !h.canFloat() {
				continue
			}
		case headCollapse:
			if cur.features&FeatureCollapsible == 0 {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// canFloat reports whether the host can open a window for a panel, or —
// for a panel that already floats — dock it back.
func (h *stackHead) canFloat() bool {
	if cur := h.stack.Current(); cur != nil && cur.Floating() {
		return true
	}
	host := hostOf(h)
	return host != nil && host.opener != nil
}

// btnSize is the width of a title bar button's box. It is narrower than
// the bar is tall — three square buttons at the bar's height would eat a
// narrow sidebar's title, and Qt's and Visual Studio's dock buttons are
// slim for the same reason.
func (h *stackHead) btnSize() float32 {
	s := h.LocalBounds().Dy()
	if s <= 0 {
		s = h.stack.headH()
	}
	if wide := style.Dip(h.Look(), 18); s > wide {
		s = wide
	}
	return s
}

// buttonRect is the box of the i'th button of buttons(), counted from the
// right edge.
func (h *stackHead) buttonRect(i int) paintengine2d.Rect {
	b := h.LocalBounds()
	sz := h.btnSize()
	pad := style.Dip(h.Look(), 2)
	x := b.Max.X - pad - float32(i+1)*sz
	return paintengine2d.XYWH(x, b.Min.Y, sz, b.Dy())
}

// titleRect is the room left for the panel's name.
func (h *stackHead) titleRect() paintengine2d.Rect {
	b := h.LocalBounds()
	n := float32(len(h.buttons()))
	pad := style.Dip(h.Look(), 6)
	right := b.Max.X - pad - n*h.btnSize()
	return paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, maxF(0, right-b.Min.X-pad), b.Dy())
}

// partAt is the title bar part at local point p.
func (h *stackHead) partAt(p paintengine2d.Point) headPart {
	if !h.LocalBounds().Contains(p) {
		return headNone
	}
	for i, part := range h.buttons() {
		if h.buttonRect(i).Contains(p) {
			return part
		}
	}
	return headGrip
}

func (h *stackHead) Measure(c layout.Constraints) paintengine2d.Point {
	w := float32(120)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h.stack.headH()))
}

func (h *stackHead) Arrange(r paintengine2d.Rect) { h.SetBounds(r) }

func (h *stackHead) Paint(ctx *paintengine2d.Context) {
	b := h.LocalBounds()
	if b.Empty() {
		return
	}
	lk := h.Look()
	cur := h.stack.Current()
	active := h.stack.active()
	st := h.State() &^ (style.StateHovered | style.StatePressed)
	if active {
		st |= style.StateChecked
	}
	fg := style.DrawFaceOf(lk, ctx, b, style.RoleBar, st)
	if active {
		// A rule along the foot of the title bar marks the panel the
		// keyboard is in, as Qt Creator and VS Code mark theirs. The face
		// alone is not enough: plenty of packs paint a checked bar exactly
		// like an unchecked one.
		h.drawActiveRule(ctx, b)
	}
	if cur == nil {
		return
	}
	// The title, elided to whatever the buttons left.
	tr := h.titleRect()
	if !tr.Empty() {
		face := style.ControlFontOf(lk, style.RoleBar)
		title := cur.Title()
		if face != nil {
			title = face.Fit(title, tr.Dx())
		}
		lk.DrawLabel(ctx, tr, title, fg, style.AlignStart)
	}
	// The buttons.
	for i, part := range h.buttons() {
		r := h.buttonRect(i)
		bst := h.State() &^ (style.StateHovered | style.StatePressed | style.StateFocused)
		if h.hover == part {
			bst |= style.StateHovered
		}
		if h.press == part {
			bst |= style.StatePressed
		}
		bfg := fg
		if bst&(style.StateHovered|style.StatePressed) != 0 {
			bfg = style.DrawFaceOf(lk, ctx, r, style.RoleTool, bst)
		}
		if h.Focused() && h.KeyNav() && h.focus == i {
			lk.DrawFocusRing(ctx, r.Inset(style.Dip(lk, 2)))
		}
		h.drawGlyph(ctx, r, part, bfg)
	}
}

// drawActiveRule underlines the title bar of the panel the keyboard is in.
// It is the pack's accent where that shows against the bar, and its focus
// or selection colour otherwise.
func (h *stackHead) drawActiveRule(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := h.Look().Palette()
	ink := p.Accent
	for _, c := range []paintengine2d.Color{p.Accent, p.Focus, p.Selection, p.Text} {
		if c.A > 0 {
			ink = c
			break
		}
	}
	w := maxF(1, style.Dip(h.Look(), 2))
	if w > b.Dy() {
		w = b.Dy()
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-w, b.Dx(), w), paintengine2d.Fill(ink))
}

// drawGlyph paints a title bar button's mark: the caption cross for close,
// the look's arrow for collapse, and a small offset frame for float —
// Qt Creator's and Visual Studio's float glyph.
func (h *stackHead) drawGlyph(ctx *paintengine2d.Context, r paintengine2d.Rect, part headPart, col paintengine2d.Color) {
	lk := h.Look()
	s := style.Dip(lk, 9)
	lw := maxF(1, style.Dip(lk, 1))
	switch part {
	case headClose:
		style.DrawCaptionGlyph(ctx, r, style.CaptionClose, false, col, s, lw)
	case headCollapse:
		dir := style.DirDown
		if h.stack.collapsed() {
			dir = style.DirRight
		}
		box := centreBox(r, style.Dip(lk, 8))
		style.DrawArrowOf(lk, ctx, box, dir, col)
	case headFloat:
		// Two overlapping frames: the panel lifting out of the window.
		box := centreBox(r, s)
		back := paintengine2d.XYWH(box.Min.X, box.Min.Y+box.Dy()*0.3, box.Dx()*0.7, box.Dy()*0.7)
		front := paintengine2d.XYWH(box.Min.X+box.Dx()*0.3, box.Min.Y, box.Dx()*0.7, box.Dy()*0.7)
		if h.stack.Current() != nil && h.stack.Current().Floating() {
			// Docking back: the frame drops into the window.
			back, front = front, back
		}
		ctx.DrawRect(back.Inset(lw*0.5), paintengine2d.StrokePaint(col.WithAlpha(col.A*0.6), lw))
		ctx.DrawRect(front.Inset(lw*0.5), paintengine2d.StrokePaint(col, lw))
	}
}

// centreBox is a square of side s in the middle of r.
func centreBox(r paintengine2d.Rect, s float32) paintengine2d.Rect {
	return paintengine2d.XYWH(r.Min.X+(r.Dx()-s)*0.5, r.Min.Y+(r.Dy()-s)*0.5, s, s)
}

func (h *stackHead) MouseMove(e widget.MouseEvent) bool {
	if host := hostOf(h); host != nil && host.dragArmed() {
		host.dragMove(widget.LocalToWindow(h, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
		return true
	}
	if p := h.partAt(e.Pos); p != h.hover {
		h.hover = p
		h.Invalidate()
	}
	return h.hover != headNone
}

func (h *stackHead) MouseExit() {
	if h.hover != headNone {
		h.hover = headNone
		h.Invalidate()
	}
	h.Base.MouseExit()
}

func (h *stackHead) MousePress(e widget.MouseEvent) bool {
	cur := h.stack.Current()
	if cur == nil {
		return false
	}
	part := h.partAt(e.Pos)
	switch part {
	case headNone:
		return false
	case headGrip:
		h.stack.raise()
		if cur.features&FeatureMovable == 0 {
			return true
		}
		if host := hostOf(h); host != nil {
			host.beginDrag(h.stack, cur, widget.LocalToWindow(h, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
		}
		return true
	default:
		h.press = part
		h.Invalidate()
		return true
	}
}

func (h *stackHead) MouseRelease(e widget.MouseEvent) bool {
	if host := hostOf(h); host != nil && host.dragArmed() {
		host.endDrag(widget.LocalToWindow(h, paintengine2d.XYWH(e.Pos.X, e.Pos.Y, 0, 0)).Min)
		return true
	}
	if h.press == headNone {
		return false
	}
	part := h.press
	h.press = headNone
	h.Invalidate()
	if h.partAt(e.Pos) == part {
		h.activate(part)
	}
	return true
}

// activate runs a title bar button.
func (h *stackHead) activate(part headPart) {
	cur := h.stack.Current()
	if cur == nil {
		return
	}
	switch part {
	case headCollapse:
		cur.ToggleCollapsed()
	case headFloat:
		if cur.Floating() {
			cur.Dock()
		} else {
			cur.Float()
		}
	case headClose:
		cur.Close()
	}
}

func (h *stackHead) KeyPress(e widget.KeyEvent) bool {
	btns := h.buttons()
	if len(btns) == 0 || !h.Enabled() {
		return false
	}
	h.MarkKeyboardFocus()
	switch e.Key {
	case platform.KeyLeft:
		h.moveFocus(1, len(btns))
		return true
	case platform.KeyRight:
		h.moveFocus(-1, len(btns))
		return true
	case platform.KeyHome:
		h.focus = len(btns) - 1
		h.Invalidate()
		return true
	case platform.KeyEnd:
		h.focus = 0
		h.Invalidate()
		return true
	case platform.KeyReturn, platform.KeySpace:
		if h.focus >= 0 && h.focus < len(btns) {
			h.activate(btns[h.focus])
			return true
		}
	}
	return false
}

// moveFocus steps the keyboard through the buttons. They are laid out from
// the right edge, so a step of +1 walks leftwards on screen.
func (h *stackHead) moveFocus(step, n int) {
	if h.focus < 0 {
		h.focus = 0
	} else {
		h.focus += step
	}
	if h.focus < 0 {
		h.focus = n - 1
	}
	if h.focus >= n {
		h.focus = 0
	}
	h.Invalidate()
}

func (h *stackHead) FocusGained() {
	if h.focus < 0 {
		h.focus = 0
	}
	h.Base.FocusGained()
}

func (h *stackHead) FocusLost() {
	h.focus = -1
	h.Base.FocusLost()
}

// Tooltip names the button under the pointer.
func (h *stackHead) Tooltip() string {
	switch h.hover {
	case headCollapse:
		if h.stack.collapsed() {
			return "Expand"
		}
		return "Collapse"
	case headFloat:
		if cur := h.stack.Current(); cur != nil && cur.Floating() {
			return "Dock"
		}
		return "Float"
	case headClose:
		return "Close"
	}
	return ""
}

// Describe implements widget.Accessible: the title bar is a named pane
// holding its buttons.
func (h *stackHead) Describe(n *a11y.Node) {
	n.Role = a11y.RolePane
	if n.Name == "" {
		if cur := h.stack.Current(); cur != nil {
			n.Name = widget.PlainText(cur.Title())
		}
	}
}

// AccessibleItems implements widget.AccessibleItems: the title bar's
// buttons, which are drawn rather than made of components.
func (h *stackHead) AccessibleItems() []*a11y.Node {
	btns := h.buttons()
	out := make([]*a11y.Node, 0, len(btns))
	for i, part := range btns {
		n := &a11y.Node{
			ID:     widget.ItemID(h, i),
			Role:   a11y.RoleButton,
			Name:   headPartName(part, h.stack),
			Bounds: widget.LocalToWindow(h, h.buttonRect(i)),
		}
		n.State |= a11y.StateFocusable
		n.Actions = n.Actions.With(a11y.ActionDefault)
		out = append(out, n)
	}
	return out
}

// AccessibleAction implements widget.AccessibleActor: assistive technology
// presses a title bar button.
func (h *stackHead) AccessibleAction(i int, a a11y.Action) bool {
	btns := h.buttons()
	if a != a11y.ActionDefault || i < 0 || i >= len(btns) || !h.Enabled() {
		return false
	}
	h.activate(btns[i])
	return true
}

// AccessibleFocusItem implements widget.AccessibleFocusItem: the button the
// keyboard is on.
func (h *stackHead) AccessibleFocusItem() int {
	if !h.Focused() {
		return -1
	}
	return h.focus
}

// headPartName is what assistive technology calls a title bar button.
func headPartName(part headPart, s *Stack) string {
	switch part {
	case headCollapse:
		if s.collapsed() {
			return "Expand"
		}
		return "Collapse"
	case headFloat:
		if cur := s.Current(); cur != nil && cur.Floating() {
			return "Dock"
		}
		return "Float"
	case headClose:
		return "Close"
	}
	return ""
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
