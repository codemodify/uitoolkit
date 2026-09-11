package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// Panel is a framed surface with optional title and a single (or stacked) child body.
type Panel struct {
	widget.Base
	Title  string
	Raised bool
	body   *FlexBox
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

func (p *Panel) chrome() (pad, titleH float32) {
	lk := p.Look()
	m := lk.Metrics()
	pad = m.Pad
	if p.Title != "" {
		titleH = m.TitleBar
	}
	return
}

func (p *Panel) Measure(c layout.Constraints) paintengine2d.Point {
	pad, titleH := p.chrome()
	inner := c.Inset(pad*2, pad*2+titleH)
	sz := p.body.Measure(inner)
	return c.Constrain(paintengine2d.Pt(sz.X+pad*2, sz.Y+pad*2+titleH))
}

func (p *Panel) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	pad, titleH := p.chrome()
	p.body.Arrange(paintengine2d.XYWH(pad, pad+titleH, r.Dx()-pad*2, r.Dy()-pad*2-titleH))
}

func (p *Panel) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	lk.DrawPanel(ctx, p.LocalBounds(), p.Raised)
	if p.Title != "" {
		m := lk.Metrics()
		bar := paintengine2d.XYWH(0, 0, p.LocalBounds().Dx(), m.TitleBar)
		lk.TitleFont().Draw(ctx, p.Title, paintengine2d.Pt(m.Pad, (m.TitleBar-lk.TitleFont().Height())*0.5), lk.Palette().Text)
		_ = bar
	}
}
