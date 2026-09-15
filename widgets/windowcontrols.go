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
type WindowControls struct {
	widget.Base
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

func (c *WindowControls) frameHost() widget.FrameHost {
	if h, ok := c.Host().(widget.FrameHost); ok {
		return h
	}
	return nil
}

// Shown are the buttons painted: those the desktop can do.
func (c *WindowControls) Shown() []platform.CaptionButton {
	caps := platform.WMCaps(0)
	if h := c.frameHost(); h != nil {
		caps = h.FrameCaps()
	}
	out := make([]platform.CaptionButton, 0, len(c.buttons))
	for _, b := range c.buttons {
		switch b {
		case platform.CaptionMinimize:
			if !caps.Can(platform.CapMinimize) {
				continue
			}
		case platform.CaptionMaximize:
			if !caps.Can(platform.CapMaximize) {
				continue
			}
		case platform.CaptionNone:
			continue
		}
		out = append(out, b)
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
	if s.Button.Y > 0 {
		return max(s.ButtonPad.Top+s.Button.Y, s.Caption)
	}
	return max(s.Caption, s.ButtonPad.Top+float32(math.Round(float64(style.Dip(c.Look(), 24)))))
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
		if b == platform.CaptionSpacer {
			w += c.spacerW()
		} else {
			w += s.Button.X
		}
		if i > 0 {
			w += s.ButtonGap
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
	bh := s.Button.Y
	if bh <= 0 {
		bh = max(h-top, 0)
	} else if s.CenterButtons && h > s.Caption {
		top += float32(math.Round(float64(h-s.Caption) * 0.5))
	}
	out := make([]paintengine2d.Rect, len(shown))
	x := float32(0)
	if c.lead {
		x = s.ButtonPad.Left
	}
	for i, b := range shown {
		w := s.Button.X
		if b == platform.CaptionSpacer {
			w = c.spacerW()
		}
		out[i] = paintengine2d.XYWH(x, top, w, bh)
		x += w + s.ButtonGap
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
		style.DrawCaptionButtonOf(lk, ctx, r, style.CaptionButton(b), cs, ds)
	}
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
func (c *WindowControls) activate(i int) bool {
	shown := c.Shown()
	h := c.frameHost()
	if i < 0 || i >= len(shown) || h == nil {
		return false
	}
	switch shown[i] {
	case platform.CaptionClose:
		h.RequestClose()
	case platform.CaptionMinimize:
		h.Minimize()
	case platform.CaptionMaximize:
		h.ToggleMaximize()
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
