package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// WindowControls are a window's caption buttons on one side of its title
// bar — minimize, maximize (restore when maximized), close, the window menu
// — in the order the desktop's button layout gives. The look paints them
// (style.DrawCaptionButtonOf: Win95's bevelled boxes, XP's glossy ones,
// Aqua's traffic lights, generic glyphs over the tool face elsewhere) at the
// size and spacing of its frame (style.DecorationOf). They are hidden for
// actions the desktop cannot do (a window manager that cannot minimize gets
// no minimize button), kept out of the Tab order, and exposed to assistive
// technology as buttons named Minimize, Maximize or Restore, Close and
// Window Menu.
//
// A window that draws its own frame puts them into its title bar
// (Window.SetTitleBar and HeaderBar); apps rarely make them directly.
// CaptionAction is a button of the application's own in the window's
// caption, beside the window controls: a browser's profile or extensions
// button, an editor's layout switches, anything a title bar carries that
// is not close, minimize or maximize.
//
// It is drawn on the era's **tool** face rather than its window-control
// face, and that is not a shortcut. A look draws its window controls as
// one piece — the shape and the glyph together, keyed on which control
// it is — so there is no way to borrow a close button's shape and put a
// different mark in it: 67 of the packs draw a glyph of their own for
// any value they do not recognise. The tool face is the era's own button
// for a mark, it is separable, and every pack has one.
type CaptionAction struct {
	// Icon is the mark. An icon alone cannot be read aloud, so Name is
	// not optional in practice.
	Icon style.ToolIcon
	// Name is what it does, in words: the tooltip and the accessible
	// name.
	Name string
	// Lead puts it at the caption's start rather than beside the
	// close button.
	Lead bool
	// Checked draws it held down, for a button that toggles something.
	Checked bool
	// OnClick runs when it is pressed. A menu is opened from here.
	OnClick func()
}

// captionActionBase is where an action's pseudo-button value starts, above
// every real [platform.CaptionButton]. The controls carry actions in the
// same list as the buttons so that placement, hit testing, hovering and
// the accessibility tree all work on one thing rather than two.
const captionActionBase platform.CaptionButton = 128

type WindowControls struct {
	widget.Base
	// actions are the application's own buttons on this side.
	actions []CaptionAction
	buttons []platform.CaptionButton
	hover   int
	press   int
	// lead: the buttons sit at the caption's left end, their outer side
	// the left.
	lead bool
}

// NewWindowControls makes caption buttons, left to right.
func NewWindowControls(buttons ...platform.CaptionButton) *WindowControls {
	c := &WindowControls{hover: -1, press: -1}
	c.Init(c)
	c.SetButtons(buttons)
	return c
}

// SetButtons replaces the buttons (left to right).
func (c *WindowControls) SetButtons(buttons []platform.CaptionButton) {
	c.buttons = append(c.buttons[:0:0], buttons...)
	c.hover, c.press = -1, -1
	c.RequestLayout()
	c.Invalidate()
}

// SetLeading tells the buttons they sit at the caption's left end (their
// outer padding goes on the left).
func (c *WindowControls) SetLeading(lead bool) {
	if c.lead != lead {
		c.lead = lead
		c.RequestLayout()
	}
}

// Buttons are the configured buttons, including ones the desktop cannot do.
func (c *WindowControls) Buttons() []platform.CaptionButton { return c.buttons }

// FocusOnClick is false: caption buttons never take the keyboard focus.
func (c *WindowControls) FocusOnClick() bool { return false }

// frameAbove is the host where its window can be kept above the others.
func (c *WindowControls) frameAbove() widget.FrameAbove {
	a, ok := c.Host().(widget.FrameAbove)
	if !ok {
		return nil
	}
	return a
}

// canKeepAbove reports whether the window can be kept above the others: a
// desktop that will do it ([platform.FrameKeepAbove]) and a host that
// carries the state. A keep-above button is not drawn otherwise.
func (c *WindowControls) canKeepAbove() bool {
	return c.caps().Has(platform.FrameKeepAbove) && c.frameAbove() != nil
}

// caps is what the window system will do for the window's frame now.
func (c *WindowControls) caps() platform.FrameCaps {
	if h := c.frameHost(); h != nil {
		return h.FrameCaps()
	}
	return 0
}

// keptAbove reports whether the window is being kept above the others: the
// button paints as a toggle that is on.
func (c *WindowControls) keptAbove() bool {
	a := c.frameAbove()
	return a != nil && a.KeepAbove()
}

// hiddenByWindow reports whether the window has turned this button off.
//
// A look and a desktop decide which buttons *can* be there; this is the
// window saying which of them it wants — a tool window with no maximize,
// a dialog with only a close. It is a per-window override and nothing
// else: it cannot add a button the desktop cannot do.
func (c *WindowControls) hiddenByWindow(b platform.CaptionButton) bool {
	h, ok := c.Host().(interface {
		CaptionButtonHidden(platform.CaptionButton) bool
	})
	return ok && h.CaptionButtonHidden(b)
}

func (c *WindowControls) frameHost() widget.FrameHost {
	if h, ok := c.Host().(widget.FrameHost); ok {
		return h
	}
	return nil
}

// SetActions replaces the application's own caption buttons on this side.
func (c *WindowControls) SetActions(a []CaptionAction) {
	c.actions = append(c.actions[:0], a...)
	c.Invalidate()
}

// Actions are the application's own caption buttons on this side.
func (c *WindowControls) Actions() []CaptionAction { return c.actions }

// actionAt is the action a pseudo-button value names, or nil.
func (c *WindowControls) actionAt(b platform.CaptionButton) *CaptionAction {
	if b < captionActionBase {
		return nil
	}
	if i := int(b - captionActionBase); i >= 0 && i < len(c.actions) {
		return &c.actions[i]
	}
	return nil
}

// Shown are the buttons painted: those the desktop can do, those the
// window has not hidden, and the application's own.
func (c *WindowControls) Shown() []platform.CaptionButton {
	caps := c.caps()
	out := make([]platform.CaptionButton, 0, len(c.buttons))
	for _, b := range c.buttons {
		switch b {
		case platform.CaptionMinimize:
			if !caps.Has(platform.FrameMinimize) {
				continue
			}
		case platform.CaptionMaximize:
			if !caps.Has(platform.FrameMaximize) {
				continue
			}
		case platform.CaptionKeepAbove:
			if !c.canKeepAbove() {
				continue
			}
		case platform.CaptionNone:
			continue
		}
		if c.hiddenByWindow(b) {
			continue
		}
		out = append(out, b)
	}
	// The application's own buttons, beside the window's: at the start
	// of a lead group and the end of a trailing one, so they sit next to
	// the edge's controls rather than between them and the corner.
	for i := range c.actions {
		v := captionActionBase + platform.CaptionButton(i)
		if c.lead {
			out = append(out, v)
		} else {
			out = append([]platform.CaptionButton{v}, out...)
		}
	}
	// A spacer at either end of a side is no gap between buttons.
	for len(out) > 0 && out[0] == platform.CaptionSpacer {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == platform.CaptionSpacer {
		out = out[:len(out)-1]
	}
	return out
}

// DecorationState is the frame state the buttons paint in: the header bar's
// (the window's active and maximized states), else an active window's.
func (c *WindowControls) DecorationState() style.DecorationState {
	for p := c.Parent(); p != nil; p = p.Parent() {
		if hb, ok := p.(*HeaderBar); ok {
			return hb.DecorationState()
		}
	}
	return frameState(c.frameHost(), false)
}

// frameState is the frame state of the window h (an active, restored
// window without one).
func frameState(h widget.FrameHost, custom bool) style.DecorationState {
	st := style.DecorationState{Active: true, Custom: custom}
	if h != nil {
		ws := h.WindowState()
		st.Active = h.Active()
		st.Maximized = ws.Maximized
		st.Tiled = style.Edges(ws.Tiled)
		st.Solid = ws.Solid
		if r, ok := h.(widget.FrameRequests); ok {
			st.Role = r.FrameRole()
			st.Caption = r.CaptionHeight()
		}
	}
	return st
}

// spec is the look's frame in the window's current state.
func (c *WindowControls) spec() style.DecorationSpec {
	return style.DecorationOf(c.Look(), c.DecorationState())
}

func (c *WindowControls) spacerW() float32 {
	return float32(math.Round(float64(style.Dip(c.Look(), 10))))
}

// MinHeight is the least height the buttons want (the caption grows to fit
// them).
func (c *WindowControls) MinHeight() float32 {
	s := c.spec()
	if h := max(s.Button.Y, s.CloseButton.Y); h > 0 {
		return max(s.ButtonPad.Top+h, s.Caption)
	}
	return max(s.Caption, s.ButtonPad.Top+float32(math.Round(float64(style.Dip(c.Look(), 24)))))
}

// size is shown button b's box in spec s (a spacer's width and no height).
func (c *WindowControls) size(s style.DecorationSpec, b platform.CaptionButton) paintengine2d.Point {
	if b == platform.CaptionSpacer {
		return paintengine2d.Pt(c.spacerW(), 0)
	}
	return s.ButtonBox(style.CaptionButton(b))
}

// gap is the room between shown buttons a and b in spec s.
func gap(s style.DecorationSpec, a, b platform.CaptionButton) float32 {
	g := s.ButtonGap
	if a == platform.CaptionClose || b == platform.CaptionClose {
		g += s.CloseGap
	}
	return g
}

// width is the room the shown buttons take in spec s, the outer padding
// included.
func (c *WindowControls) width(s style.DecorationSpec) float32 {
	shown := c.Shown()
	if len(shown) == 0 {
		return 0
	}
	var w float32
	for i, b := range shown {
		w += c.size(s, b).X
		if i > 0 {
			w += gap(s, shown[i-1], b)
		}
	}
	if c.lead {
		return w + s.ButtonPad.Left
	}
	return w + s.ButtonPad.Right
}

func (c *WindowControls) Measure(cons layout.Constraints) paintengine2d.Point {
	return cons.Constrain(paintengine2d.Pt(c.width(c.spec()), c.MinHeight()))
}

func (c *WindowControls) Arrange(r paintengine2d.Rect) { c.SetBounds(r) }

// rects are the shown buttons' boxes, as the look places them.
func (c *WindowControls) rects() []paintengine2d.Rect {
	s := c.spec()
	shown := c.Shown()
	h := c.LocalBounds().Dy()
	top := s.ButtonPad.Top
	if s.CenterButtons && h > s.Caption {
		top += float32(math.Round(float64(h-s.Caption) * 0.5))
	}
	out := make([]paintengine2d.Rect, len(shown))
	x := float32(0)
	if c.lead {
		x = s.ButtonPad.Left
	}
	for i, b := range shown {
		if i > 0 {
			x += gap(s, shown[i-1], b)
		}
		sz := c.size(s, b)
		bh := sz.Y
		if bh <= 0 {
			bh = max(h-s.ButtonPad.Top, 0)
		}
		out[i] = paintengine2d.XYWH(x, top, sz.X, bh)
		x += sz.X
	}
	return out
}

// hitRects are the boxes that take the pointer: each button's box run up to
// the caption's top edge, the outermost one out to the window's side too, so
// a flick into a maximized window's corner hits it (Fitts's law, as Windows
// does). Gaps between buttons stay caption.
func (c *WindowControls) hitRects() []paintengine2d.Rect {
	rs := c.rects()
	w := c.LocalBounds().Dx()
	for i := range rs {
		rs[i].Min.Y = 0
		if c.lead && i == 0 {
			rs[i].Min.X = 0
		}
		if !c.lead && i == len(rs)-1 {
			rs[i].Max.X = w
		}
	}
	return rs
}

// ButtonAt is the caption button at local point p (CaptionNone for none
// or a spacer).
func (c *WindowControls) ButtonAt(p paintengine2d.Point) platform.CaptionButton {
	if i := c.indexAt(p); i >= 0 {
		return c.Shown()[i]
	}
	return platform.CaptionNone
}

// ButtonRect is where button b is painted, in local coordinates (empty when
// it is not shown).
func (c *WindowControls) ButtonRect(b platform.CaptionButton) paintengine2d.Rect {
	shown := c.Shown()
	for i, r := range c.rects() {
		if shown[i] == b {
			return r
		}
	}
	return paintengine2d.Rect{}
}

func (c *WindowControls) indexAt(p paintengine2d.Point) int {
	shown := c.Shown()
	for i, r := range c.hitRects() {
		if r.Contains(p) && shown[i] != platform.CaptionSpacer {
			return i
		}
	}
	return -1
}

// CaptionAt: the gaps and spacers are caption, the buttons are controls.
func (c *WindowControls) CaptionAt(p paintengine2d.Point) bool { return c.indexAt(p) < 0 }

// Tooltip names the hovered button.
func (c *WindowControls) Tooltip() string {
	shown := c.Shown()
	if c.hover >= 0 && c.hover < len(shown) {
		return c.buttonName(shown[c.hover])
	}
	return ""
}

func (c *WindowControls) buttonName(b platform.CaptionButton) string {
	switch b {
	case platform.CaptionClose:
		return "Close"
	case platform.CaptionMinimize:
		return "Minimize"
	case platform.CaptionMaximize:
		if h := c.frameHost(); h != nil && h.WindowState().Maximized {
			return "Restore"
		}
		return "Maximize"
	case platform.CaptionMenu:
		return "Window Menu"
	case platform.CaptionKeepAbove:
		if c.keptAbove() {
			return "Stop Keeping Above Others"
		}
		return "Keep Above Others"
	}
	return ""
}

func (c *WindowControls) Paint(ctx *paintengine2d.Context) {
	lk := c.Look()
	shown := c.Shown()
	ds := c.DecorationState()
	for i, r := range c.rects() {
		b := shown[i]
		if b == platform.CaptionSpacer {
			continue
		}
		var cs style.ControlState
		if i == c.hover {
			cs |= style.StateHovered
		}
		if i == c.press {
			cs |= style.StatePressed | style.StateHovered
		}
		if b == platform.CaptionKeepAbove {
			// A toggle, not a command. While it is on it is a button held
			// down — which is how a set toggle has looked in every era
			// from Motif to Breeze, and how KWin draws its own keep-above
			// button — so every engine shows the state through the pressed
			// art it already has, and the glyph fills in as well.
			cs |= style.StateToggle
			if c.keptAbove() {
				cs |= style.StateChecked | style.StatePressed
			}
		}
		if a := c.actionAt(b); a != nil {
			c.paintAction(ctx, lk, r, *a, cs)
			continue
		}
		style.DrawCaptionButtonOf(lk, ctx, r, style.CaptionButton(b), cs, ds)
	}
}

// paintAction draws one of the application's own caption buttons: the
// era's tool face, and the app's mark on it.
func (c *WindowControls) paintAction(ctx *paintengine2d.Context, lk style.LookAndFeel,
	r paintengine2d.Rect, a CaptionAction, cs style.ControlState) {
	if a.Checked {
		cs |= style.StateChecked | style.StateToggle
	}
	lk.DrawToolButton(ctx, r, cs, "", style.IconNone)
	if a.Icon == style.IconNone {
		return
	}
	// The tool face's own ink, since the tool face is what is under it.
	col := lk.Palette().Text
	if cs.Disabled() {
		col = lk.Palette().TextMuted
	}
	side := min(r.Dx(), r.Dy()) * 0.55
	box := paintengine2d.XYWH(
		r.Min.X+(r.Dx()-side)*0.5,
		r.Min.Y+(r.Dy()-side)*0.5, side, side)
	style.DrawToolIcon(ctx, box, a.Icon, col, style.IconSetOf(lk))
}

func (c *WindowControls) invalidateButton(i int) {
	if rs := c.hitRects(); i >= 0 && i < len(rs) {
		c.InvalidateRect(rs[i].Union(c.rects()[i]).Inset(-1))
	}
}

func (c *WindowControls) MouseMove(e widget.MouseEvent) bool {
	if i := c.indexAt(e.Pos); i != c.hover {
		old := c.hover
		c.hover = i
		c.invalidateButton(old)
		c.invalidateButton(i)
	}
	return true
}

func (c *WindowControls) MouseExit() {
	c.invalidateButton(c.hover)
	c.invalidateButton(c.press)
	c.hover, c.press = -1, -1
	c.Base.MouseExit()
}

func (c *WindowControls) MousePress(e widget.MouseEvent) bool {
	i := c.indexAt(e.Pos)
	if i < 0 {
		return false
	}
	if e.Button == platform.ButtonRight {
		// Right-clicking a caption button asks for the window menu, as
		// the desktop's own frame does.
		if h := c.frameHost(); h != nil {
			h.ShowWindowMenu(widget.DeviceOrigin(c).Add(e.Pos))
		}
		return true
	}
	if e.Button != platform.ButtonLeft {
		return false
	}
	c.press = i
	c.invalidateButton(i)
	return true
}

func (c *WindowControls) MouseRelease(e widget.MouseEvent) bool {
	was := c.press
	c.press = -1
	c.invalidateButton(was)
	if was >= 0 && was == c.indexAt(e.Pos) && e.Button == platform.ButtonLeft {
		c.activate(was)
	}
	return true
}

// activate runs shown button i.
// ActivateForTest presses the i-th shown button, for a test driving the
// caption without a pointer.
func (c *WindowControls) ActivateForTest(i int) bool { return c.activate(i) }

func (c *WindowControls) activate(i int) bool {
	shown := c.Shown()
	h := c.frameHost()
	if i < 0 || i >= len(shown) || h == nil {
		return false
	}
	if a := c.actionAt(shown[i]); a != nil {
		if a.OnClick != nil {
			a.OnClick()
		}
		return true
	}
	switch shown[i] {
	case platform.CaptionClose:
		h.RequestClose()
	case platform.CaptionMinimize:
		h.Minimize()
	case platform.CaptionMaximize:
		h.ToggleMaximize()
	case platform.CaptionKeepAbove:
		a := c.frameAbove()
		if a == nil {
			return false
		}
		a.ToggleKeepAbove()
	case platform.CaptionMenu:
		r := c.rects()[i]
		h.ShowWindowMenu(widget.DeviceOrigin(c).Add(paintengine2d.Pt(r.Min.X, r.Max.Y)))
	default:
		return false
	}
	return true
}

// Describe implements widget.Accessible: a pane holding the buttons.
func (c *WindowControls) Describe(n *a11y.Node) { n.Role = a11y.RolePane }

// AccessibleItems are the caption buttons.
func (c *WindowControls) AccessibleItems() []*a11y.Node {
	shown := c.Shown()
	rects := c.rects()
	var out []*a11y.Node
	for i, b := range shown {
		if b == platform.CaptionSpacer {
			continue
		}
		n := item(c, i, a11y.RoleButton, c.buttonName(b), rects[i])
		n.Actions = n.Actions.With(a11y.ActionDefault)
		if b == platform.CaptionKeepAbove {
			// A screen reader is told it is a toggle and which way it is.
			n.Role = a11y.RoleToggleButton
			n.State |= a11y.StateCheckable
			if c.keptAbove() {
				n.State |= a11y.StateChecked | a11y.StatePressed
			}
		}
		out = append(out, n)
	}
	return out
}

// AccessibleAction presses caption button i.
func (c *WindowControls) AccessibleAction(i int, a a11y.Action) bool {
	if a != a11y.ActionDefault {
		return false
	}
	return c.activate(i)
}
