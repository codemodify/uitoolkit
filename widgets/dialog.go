package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// Overlay is a dimmed full-window layer with a centered card (dialog pattern).
type Overlay struct {
	widget.Base
	Card    widget.Component
	OnClose func()
}

func NewOverlay(card widget.Component) *Overlay {
	o := &Overlay{Card: card}
	o.Init(o)
	if card != nil {
		o.Add(card)
	}
	return o
}

func (o *Overlay) Measure(c layout.Constraints) paintengine2d.Point {
	if c.HasMaxW() && c.HasMaxH() {
		return paintengine2d.Pt(c.MaxW, c.MaxH)
	}
	return paintengine2d.Pt(400, 300)
}

func (o *Overlay) Arrange(r paintengine2d.Rect) {
	o.SetBounds(r)
	if o.Card == nil {
		return
	}
	cs := o.Card.Measure(layout.Loose(r.Dx()*0.8, r.Dy()*0.8))
	if cs.X < 280 {
		cs.X = 280
	}
	if cs.Y < 140 {
		cs.Y = 140
	}
	x := (r.Dx() - cs.X) * 0.5
	y := (r.Dy() - cs.Y) * 0.5
	o.Card.Arrange(paintengine2d.XYWH(x, y, cs.X, cs.Y))
}

func (o *Overlay) Paint(ctx *paintengine2d.Context) {
	o.Look().DrawOverlay(ctx, o.LocalBounds())
}

func (o *Overlay) MousePress(e widget.MouseEvent) bool {
	if o.Card != nil && o.Card.Bounds().Contains(e.Pos) {
		return false
	}
	if o.OnClose != nil {
		o.OnClose()
	}
	return true
}

// DialogCard is a titled panel with a message and action row.
func DialogCard(title, body string, actions ...widget.Component) *Panel {
	col := NewColumn(
		NewTitle(title),
		NewLabel(body),
		NewRow(actions...).WithGap(8).WithJustify(layout.JustifyEnd),
	).WithGap(12).WithPad(4)
	p := NewPanel("", col)
	p.Raised = true
	return p
}
