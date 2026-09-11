package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// DefaultTooltipDelay is how long a pointer must rest before a tip appears.
const DefaultTooltipDelayNanos = 450_000_000

// TipWrap attaches delayed hover text to a single child.
type TipWrap struct {
	widget.Base
	Text  string
	child widget.Component
}

// NewTip wraps child so hovering it shows text after the window delay.
func NewTip(text string, child widget.Component) *TipWrap {
	t := &TipWrap{Text: text, child: child}
	t.Init(t)
	if child != nil {
		t.Add(child)
	}
	return t
}

func (t *TipWrap) Tooltip() string { return t.Text }

func (t *TipWrap) Measure(c layout.Constraints) paintengine2d.Point {
	if t.child == nil {
		return c.Constrain(paintengine2d.Pt(0, 0))
	}
	return t.child.Measure(c)
}

func (t *TipWrap) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	if t.child != nil {
		t.child.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	}
}

func (t *TipWrap) Paint(*paintengine2d.Context) {}

// TooltipBubble is the floating hover card painted on the window tip layer.
type TooltipBubble struct {
	widget.Base
	Text string
}

// NewTooltipBubble sizes itself to text.
func NewTooltipBubble(text string) *TooltipBubble {
	b := &TooltipBubble{Text: text}
	b.Init(b)
	return b
}

func (b *TooltipBubble) Measure(c layout.Constraints) paintengine2d.Point {
	lk := b.Look()
	f := lk.Font()
	pad := lk.Metrics().TooltipPad
	if pad <= 0 {
		pad = 8
	}
	w := f.Advance(b.Text) + pad*2
	h := f.Height() + pad
	if h < 22 {
		h = 22
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *TooltipBubble) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *TooltipBubble) Paint(ctx *paintengine2d.Context) {
	b.Look().DrawTooltip(ctx, b.LocalBounds(), b.Text)
}

func (b *TooltipBubble) HitTest(paintengine2d.Point) widget.Component { return nil }
