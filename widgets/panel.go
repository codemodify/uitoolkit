package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Panel is a framed surface with optional title and a single (or stacked)
// child body. The look decides the frame: a group box with an inline title
// (Win95 etched frame, Aqua rounded box) or, with Window set, an in-app
// window with a caption bar (dialogs, message boxes).
type Panel struct {
	widget.Base
	Title  string
	Raised bool
	// Window paints the panel as an in-app window: the look's frame and
	// caption with Title, and a close button when OnClose is set.
	Window  bool
	OnClose func()
	body    *FlexBox

	closeHot, closeDown bool
}

func NewPanel(title string, children ...widget.Component) *Panel {
	p := &Panel{Title: title, body: NewColumn(children...).WithGap(8)}
	p.Init(p)
	p.Base.Add(p.body)
	return p
}

func (p *Panel) Content() *FlexBox { return p.body }

func (p *Panel) Add(child widget.Component) {
	p.body.Add(child)
}

// windowLook is the look's in-app window chrome when Window is set.
func (p *Panel) windowLook() (style.WindowFrameLook, bool) {
	if !p.Window {
		return nil, false
	}
	w, ok := p.Look().(style.WindowFrameLook)
	return w, ok
}

// insets is the chrome around the body: window frame + padding, the
// look's group box, or the stock pad + title band.
func (p *Panel) insets() style.Insets {
	lk := p.Look()
	m := lk.Metrics()
	if w, ok := p.windowLook(); ok {
		in := w.WindowFrameInsets()
		return style.Insets{Top: in.Top + m.Pad, Right: in.Right + m.Pad, Bottom: in.Bottom + m.Pad, Left: in.Left + m.Pad}
	}
	if g, ok := lk.(style.GroupBoxLook); ok {
		return g.GroupBoxInsets(p.Title != "")
	}
	in := style.Insets{Top: m.Pad, Right: m.Pad, Bottom: m.Pad, Left: m.Pad}
	if p.Title != "" {
		in.Top += m.TitleBar
	}
	return in
}

func (p *Panel) Measure(c layout.Constraints) paintengine2d.Point {
	in := p.insets()
	inner := c.Inset(in.Left+in.Right, in.Top+in.Bottom)
	sz := p.body.Measure(inner)
	return c.Constrain(paintengine2d.Pt(sz.X+in.Left+in.Right, sz.Y+in.Top+in.Bottom))
}

func (p *Panel) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	p.body.Arrange(p.insets().Apply(p.LocalBounds()))
}

func (p *Panel) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	b := p.LocalBounds()
	if w, ok := p.windowLook(); ok {
		w.DrawWindowFrame(ctx, b, p.Title, style.WindowState{
			Active: true, CanClose: p.OnClose != nil,
			CloseHot: p.closeHot, ClosePress: p.closeDown && p.closeHot,
		})
		return
	}
	if g, ok := lk.(style.GroupBoxLook); ok {
		g.DrawGroupBox(ctx, b, p.Title, p.Raised)
		return
	}
	lk.DrawPanel(ctx, b, p.Raised)
	if p.Title != "" {
		bar := paintengine2d.XYWH(0, 0, b.Dx(), lk.Metrics().TitleBar)
		lk.DrawTitleBar(ctx, bar, p.Title, "")
	}
}

// closeRect is the caption close button in local coordinates.
func (p *Panel) closeRect() paintengine2d.Rect {
	if p.OnClose == nil {
		return paintengine2d.Rect{}
	}
	w, ok := p.windowLook()
	if !ok {
		return paintengine2d.Rect{}
	}
	return w.WindowCloseRect(p.LocalBounds())
}

func (p *Panel) MouseMove(e widget.MouseEvent) bool {
	cr := p.closeRect()
	hot := !cr.Empty() && cr.Contains(e.Pos)
	if hot != p.closeHot {
		p.closeHot = hot
		p.InvalidateRect(cr.Inset(-2))
	}
	return hot || p.closeDown
}

func (p *Panel) MouseExit() {
	if p.closeHot {
		p.closeHot = false
		p.InvalidateRect(p.closeRect().Inset(-2))
	}
	p.Base.MouseExit()
}

func (p *Panel) MousePress(e widget.MouseEvent) bool {
	cr := p.closeRect()
	if e.Button != platform.ButtonLeft || cr.Empty() || !cr.Contains(e.Pos) {
		return false
	}
	p.closeDown = true
	p.closeHot = true
	p.InvalidateRect(cr.Inset(-2))
	return true
}

func (p *Panel) MouseRelease(e widget.MouseEvent) bool {
	if !p.closeDown {
		return false
	}
	p.closeDown = false
	cr := p.closeRect()
	p.InvalidateRect(cr.Inset(-2))
	if !cr.Empty() && cr.Contains(e.Pos) && p.OnClose != nil {
		p.OnClose()
	}
	return true
}
