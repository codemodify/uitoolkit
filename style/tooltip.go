package style

import "github.com/codemodify/paintengine2d"

// tipMeasureChars is how wide a tooltip may grow before its text wraps,
// counted in characters of the face the tip is painted in.
//
// 48 is the bottom of the 45–75 character measure typography calls
// comfortable for a paragraph and the top of what a hover card should be:
// a tip is read at a glance, over the control it explains, not settled
// into. It is a measure of the font rather than a pixel count, so an era
// face at 12px and the default at 16px both get a line of the same *words*,
// and a tip at 1.75x display scale is the same shape as at 1x. (The cap it
// replaces was 360 pixels, which is 48 characters of the default face at
// 1x to within three pixels — the same width, by accident, for exactly one
// font at one scale.)
const tipMeasureChars = 48

// tipAlphabet measures one character of a face: the average advance of the
// lowercase letters, the usual stand-in for "a character" in a line-length
// rule. A monospaced face answers the same number for any of them; a
// proportional face answers what its running text averages.
const tipAlphabet = "abcdefghijklmnopqrstuvwxyz"

// TooltipStyle is how a look lays a tooltip's text out. The engine names
// the face, the padding and the alignment — they differ by era, and a
// bubble measured in one face and painted in another is a bubble with its
// last word outside it — and the look adds the measure tips wrap at.
//
// It is what keeps [Engine.DrawTooltip]'s single string honest: the widget
// wraps with exactly the face and padding the engine will draw with, and
// hands the lines down joined by newlines.
type TooltipStyle struct {
	Face  *Font   // the face the tip's text is painted in
	Pad   float32 // the inset from the bubble's edge to its text
	Align Align   // how a line sits across the bubble
	MaxW  float32 // the widest the bubble may grow before the text wraps
}

// Wrap lays text out for a bubble no wider than maxW and returns the lines
// and the bubble that holds them. maxW of 0 — or more than the style's own
// measure — means the style's measure; anything smaller is the window
// having less room than the tip would like.
func (t TooltipStyle) Wrap(text string, maxW float32) ([]string, paintengine2d.Point) {
	if t.Face == nil || text == "" {
		return nil, paintengine2d.Point{}
	}
	if maxW <= 0 || maxW > t.MaxW {
		maxW = t.MaxW
	}
	budget := maxW - t.Pad*2
	// Never less than one wide glyph: a bubble squeezed past that wraps
	// every rune onto a line of its own, which is not readable at any
	// width, so it keeps the widest line it can and the engine elides.
	if w := t.Face.Advance("W"); budget < w {
		budget = w
	}
	lines := t.Face.Wrap(text, budget)
	var w float32
	for _, ln := range lines {
		w = max(w, t.Face.Advance(ln))
	}
	// Whole pixels, rounded up. The widest line has to be drawable in what
	// is left of the bubble after the padding, and an engine that insets by
	// exactly Pad on both sides would otherwise elide a line whose advance
	// landed a quarter of a pixel over the width it was measured at.
	return lines, paintengine2d.Pt(ceilPx(w)+t.Pad*2, ceilPx(t.Face.Height()*float32(len(lines)))+t.Pad)
}

// ceilPx rounds up to a whole pixel.
func ceilPx(v float32) float32 {
	if n := float32(int32(v)); n < v {
		return n + 1
	}
	return float32(int32(v))
}

// TooltipStyleLook is a look that lays its tooltips out itself.
type TooltipStyleLook interface {
	TooltipStyle() TooltipStyle
}

// TooltipStyleOf is how lk lays a tip's text out. A look with no opinion
// (one that is not a [Classic]) gets its body font, its Metrics.TooltipPad
// and the stock measure, so a tip still wraps.
func TooltipStyleOf(lk LookAndFeel) TooltipStyle {
	if lk == nil {
		return TooltipStyle{}
	}
	if t, ok := lk.(TooltipStyleLook); ok {
		return t.TooltipStyle()
	}
	f := lk.Font()
	pad := lk.Metrics().TooltipPad
	if pad <= 0 {
		pad = 8
	}
	return tipStyle(f, pad, AlignStart)
}

// TooltipStyle is how this look lays a tooltip's text out.
func (l *Classic) TooltipStyle() TooltipStyle { return l.eng().TooltipStyle(l) }

// tipStyle fills the measure in for a face, a padding and an alignment.
func (l *Classic) tipStyle(f *Font, pad float32, a Align) TooltipStyle {
	if f == nil {
		f = l.body
	}
	return tipStyle(f, pad, a)
}

func tipStyle(f *Font, pad float32, a Align) TooltipStyle {
	t := TooltipStyle{Face: f, Pad: pad, Align: a}
	if f != nil {
		t.MaxW = f.Advance(tipAlphabet)/float32(len([]rune(tipAlphabet)))*tipMeasureChars + pad*2
	}
	return t
}

// tipPad is the look's tooltip padding, or def where the pack states none.
func (l *Classic) tipPad(def float32) float32 {
	if l != nil && l.metrics.TooltipPad > 0 {
		return l.metrics.TooltipPad
	}
	return def
}

// drawTipText paints a tip's text into b — the bubble already inset by the
// engine's padding, as every DrawTooltip insets it — one line per newline.
//
// One line sits in b exactly as it always did — centred, and overflowing
// evenly where a tight era metric leaves it less than a line's height. A
// block the box cannot hold starts at the top instead and loses its last
// lines, because a tip the window cut short should keep its beginning.
// Each line is elided on its own if the bubble ended up narrower than the
// wrap it was measured for.
func (l *Classic) drawTipText(ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color) {
	t := l.TooltipStyle()
	l.drawFittedLines(ctx, t.Face, splitLines(text), b, col, t.Align, 0)
}
