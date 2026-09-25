package widgets

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
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
//
// The bubble wraps its text rather than growing one line long: a tip is a
// sentence about a control, and a sentence is wider than any window at one
// line. It wraps at the measure its look gives a tip ([style.TooltipStyle],
// 48 characters of the tip's own face), or at the room the window has,
// whichever is less, in exactly the face and padding the engine will paint
// it in — so what was measured is what fits.
//
// The lines reach the engine as one string with a newline between them:
// [style.LookAndFeel.DrawTooltip] takes a string, every engine in the
// toolkit paints it through the look's line-block painter, and the tip is
// as tall as its lines need.
type TooltipBubble struct {
	widget.Base
	Text string

	// lines is the last wrap, the text it was made from and the bubble it
	// asked for: Paint hands the lines down, and wraps again only for a
	// bubble nobody measured or one the window has since squeezed
	// narrower than the wrap was measured at.
	lines   []string
	lineFor string
	lineBox float32
}

// NewTooltipBubble sizes itself to text.
func NewTooltipBubble(text string) *TooltipBubble {
	b := &TooltipBubble{Text: text}
	b.Init(b)
	return b
}

// Lines is the text as the bubble wrapped it (empty before it is measured).
func (b *TooltipBubble) Lines() []string { return b.lines }

// wrap lays the text out for a bubble no wider than maxW and returns the
// size that holds it. maxW <= 0 asks for the look's own measure.
func (b *TooltipBubble) wrap(maxW float32) paintengine2d.Point {
	ts := style.TooltipStyleOf(b.Look())
	lines, sz := ts.Wrap(b.Text, maxW)
	b.lines, b.lineFor, b.lineBox = lines, b.Text, sz.X
	return sz
}

func (b *TooltipBubble) Measure(c layout.Constraints) paintengine2d.Point {
	maxW := float32(0)
	if c.HasMaxW() {
		maxW = c.MaxW
	}
	sz := b.wrap(maxW)
	if sz.Y < 22 {
		sz.Y = 22
	}
	return c.Constrain(sz)
}

func (b *TooltipBubble) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *TooltipBubble) Paint(ctx *paintengine2d.Context) {
	bounds := b.LocalBounds()
	if b.lines == nil || b.lineFor != b.Text || bounds.Dx() < b.lineBox-0.5 {
		b.wrap(bounds.Dx())
	}
	b.Look().DrawTooltip(ctx, bounds, strings.Join(b.lines, "\n"))
}

func (b *TooltipBubble) HitTest(paintengine2d.Point) widget.Component { return nil }
