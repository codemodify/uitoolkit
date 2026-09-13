package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Overlay is a dimmed full-window layer with a centered card (dialog pattern).
type Overlay struct {
	widget.Base
	Card      widget.Component
	OnClose   func()
	MinCardW  float32
	MinCardH  float32
	prevFocus widget.Component
}

func NewOverlay(card widget.Component) *Overlay {
	o := &Overlay{Card: card}
	o.Init(o)
	if card != nil {
		o.Add(card)
	}
	return o
}

// Presented traps focus inside the card (widget.Presenter). Without it the
// widget that was focused before the modal opened keeps receiving text and
// Return, so typing "leaks" under the dimmer.
func (o *Overlay) Presented(from widget.Component) {
	o.prevFocus = widget.FocusOwner(from)
	if o.prevFocus == nil {
		o.prevFocus = from
	}
	if o.Card != nil {
		widget.FocusFirstIn(o.Card)
	}
}

// FocusAnchor is the component focused before the overlay was shown.
func (o *Overlay) FocusAnchor() widget.Component { return o.prevFocus }

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
	minW, minH := o.MinCardW, o.MinCardH
	if minW < 280 {
		minW = 280
	}
	if minH < 140 {
		minH = 140
	}
	if cs.X < minW {
		cs.X = minW
	}
	if cs.Y < minH {
		cs.Y = minH
	}
	if cs.X > r.Dx()*0.92 {
		cs.X = r.Dx() * 0.92
	}
	if cs.Y > r.Dy()*0.88 {
		cs.Y = r.Dy() * 0.88
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
	widget.DismissOverlay(o)
	return true
}

func (o *Overlay) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		widget.DismissOverlay(o)
		return true
	}
	return false
}

func (o *Overlay) Dismissed() {
	// Only while the card still owns focus: a dismissal triggered by clicking
	// elsewhere must not steal focus back from what the user just clicked.
	widget.RestoreFocus(o, o.prevFocus, true)
	o.prevFocus = nil
	if o.OnClose != nil {
		o.OnClose()
	}
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
