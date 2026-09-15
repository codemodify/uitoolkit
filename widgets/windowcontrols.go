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
// — in the order the desktop's button layout gives. They are painted as
// generic glyphs over the look's tool-button face, hidden for actions the
// desktop cannot do (a window manager that cannot minimize gets no minimize
// button), kept out of the Tab order, and exposed to assistive technology
// as buttons named Minimize, Maximize or Restore, Close and Window Menu.
//
// A window that draws its own frame puts them into its title bar
// (Window.SetTitleBar and HeaderBar); apps rarely make them directly.
type WindowControls struct {
	widget.Base
	buttons []platform.CaptionButton
	hover   int
	press   int
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

// buttonW is one caption button's width; spacerW a spacer's.
func (c *WindowControls) buttonW() float32 {
	return float32(math.Round(float64(style.Dip(c.Look(), 40))))
}
func (c *WindowControls) spacerW() float32 {
	return float32(math.Round(float64(style.Dip(c.Look(), 10))))
}

// MinHeight is the least height the buttons want (the caption band grows
// to fit them).
func (c *WindowControls) MinHeight() float32 {
	return float32(math.Round(float64(style.Dip(c.Look(), 30))))
}

func (c *WindowControls) Measure(cons layout.Constraints) paintengine2d.Point {
	var w float32
	for _, b := range c.Shown() {
		if b == platform.CaptionSpacer {
			w += c.spacerW()
		} else {
			w += c.buttonW()
		}
	}
	return cons.Constrain(paintengine2d.Pt(w, c.MinHeight()))
}

func (c *WindowControls) Arrange(r paintengine2d.Rect) { c.SetBounds(r) }

// rects are the shown buttons' boxes, full height so a maximized window's
// buttons reach the screen edge.
func (c *WindowControls) rects() []paintengine2d.Rect {
	shown := c.Shown()
	h := c.LocalBounds().Dy()
	out := make([]paintengine2d.Rect, len(shown))
	var x float32
	for i, b := range shown {
		w := c.buttonW()
		if b == platform.CaptionSpacer {
			w = c.spacerW()
		}
		out[i] = paintengine2d.XYWH(x, 0, w, h)
		x += w
	}
	return out
}

// ButtonAt is the caption button at local point p (CaptionNone for none
// or a spacer).
func (c *WindowControls) ButtonAt(p paintengine2d.Point) platform.CaptionButton {
	if i := c.indexAt(p); i >= 0 {
		return c.Shown()[i]
	}
	return platform.CaptionNone
}

// ButtonRect is where button b is, in local coordinates (empty when it is
// not shown).
func (c *WindowControls) ButtonRect(b platform.CaptionButton) paintengine2d.Rect {
	for i, r := range c.rects() {
		if c.Shown()[i] == b {
			return r
		}
	}
	return paintengine2d.Rect{}
}

func (c *WindowControls) indexAt(p paintengine2d.Point) int {
	shown := c.Shown()
	for i, r := range c.rects() {
		if r.Contains(p) && shown[i] != platform.CaptionSpacer {
			return i
		}
	}
	return -1
}

// CaptionAt: the gaps of spacers are caption, the buttons are controls.
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
	h := c.frameHost()
	active, maximized := true, false
	if h != nil {
		active, maximized = h.Active(), h.WindowState().Maximized
	}
	for i, r := range c.rects() {
		b := shown[i]
		if b == platform.CaptionSpacer {
			continue
		}
		st := style.StateAutoRaise
		if !active {
			st |= style.StateBackdrop
		}
		if i == c.hover {
			st |= style.StateHovered
		}
		if i == c.press {
			st |= style.StatePressed
		}
		var fg paintengine2d.Color
		if b == platform.CaptionClose && (i == c.hover || i == c.press) {
			// Close turns red under the pointer (Windows, Breeze, Chromium).
			bg := lk.Palette().Danger
			if i == c.press {
				bg = bg.Lerp(paintengine2d.RGB(0, 0, 0), 0.2)
			}
			ctx.DrawRect(r, paintengine2d.Fill(bg))
			fg = paintengine2d.RGB(1, 1, 1)
		} else {
			fg = style.DrawFaceOf(lk, ctx, r, style.RoleTool, st)
		}
		if !active && i != c.hover {
			fg = fg.WithAlpha(fg.A * 0.55)
		}
		drawCaptionGlyph(ctx, lk, r, b, maximized, fg)
	}
}

// drawCaptionGlyph draws a generic caption-button glyph centred in r:
// an ✕ for close, a bar for minimize, a square for maximize, two for
// restore, a small window for the window menu. Lines are whole device
// pixels so they stay crisp at any scale.
func drawCaptionGlyph(ctx *paintengine2d.Context, lk style.LookAndFeel, r paintengine2d.Rect, b platform.CaptionButton, maximized bool, col paintengine2d.Color) {
	round := func(v float32) float32 { return float32(math.Round(float64(v))) }
	s := round(style.Dip(lk, 10))
	lw := max(1, round(style.Dip(lk, 1)))
	x0 := round(r.Min.X + (r.Dx()-s)*0.5)
	y0 := round(r.Min.Y + (r.Dy()-s)*0.5)
	g := paintengine2d.XYWH(x0, y0, s, s)
	fill := paintengine2d.Fill(col)
	frame := func(q paintengine2d.Rect) {
		ctx.DrawRect(paintengine2d.XYWH(q.Min.X, q.Min.Y, q.Dx(), lw), fill)
		ctx.DrawRect(paintengine2d.XYWH(q.Min.X, q.Max.Y-lw, q.Dx(), lw), fill)
		ctx.DrawRect(paintengine2d.XYWH(q.Min.X, q.Min.Y+lw, lw, q.Dy()-2*lw), fill)
		ctx.DrawRect(paintengine2d.XYWH(q.Max.X-lw, q.Min.Y+lw, lw, q.Dy()-2*lw), fill)
	}
	switch b {
	case platform.CaptionClose:
		var p paintengine2d.Path
		p.MoveTo(g.Min.X, g.Min.Y)
		p.LineTo(g.Max.X, g.Max.Y)
		p.MoveTo(g.Max.X, g.Min.Y)
		p.LineTo(g.Min.X, g.Max.Y)
		ctx.DrawPath(&p, paintengine2d.StrokePaint(col, lw*1.15))
	case platform.CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(g.Min.X, round(g.Min.Y+s*0.5), s, lw), fill)
	case platform.CaptionMaximize:
		if !maximized {
			frame(g)
			return
		}
		// Restore: a square in front of another, the back one showing
		// its top and right edges.
		d := max(lw*2, round(style.Dip(lk, 2)))
		back := paintengine2d.XYWH(g.Min.X+d, g.Min.Y, s-d, s-d)
		front := paintengine2d.XYWH(g.Min.X, g.Min.Y+d, s-d, s-d)
		ctx.DrawRect(paintengine2d.XYWH(back.Min.X, back.Min.Y, back.Dx(), lw), fill)
		ctx.DrawRect(paintengine2d.XYWH(back.Max.X-lw, back.Min.Y, lw, back.Dy()), fill)
		ctx.DrawRect(paintengine2d.XYWH(back.Min.X, back.Min.Y, lw, front.Min.Y-back.Min.Y), fill)
		ctx.DrawRect(paintengine2d.XYWH(front.Max.X, back.Max.Y-lw, back.Max.X-front.Max.X, lw), fill)
		frame(front)
	case platform.CaptionMenu:
		w := paintengine2d.XYWH(g.Min.X, round(g.Min.Y+s*0.1), s, round(s*0.8))
		frame(w)
		ctx.DrawRect(paintengine2d.XYWH(w.Min.X, w.Min.Y, w.Dx(), max(lw*2, round(s*0.25))), fill)
	}
}

func (c *WindowControls) invalidateButton(i int) {
	if rs := c.rects(); i >= 0 && i < len(rs) {
		c.InvalidateRect(rs[i].Inset(-1))
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
