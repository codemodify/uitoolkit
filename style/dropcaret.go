package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// The insertion caret: the line every desktop toolkit draws in the gap a
// dragged item would land in — Qt's drop indicator between two rows, GTK's
// tree-view drop line, the Finder's insertion bar, Explorer's. It is what
// tells a drop *between* two items from a drop *onto* one, which keeps the
// row highlight [DrawDropIndicatorOf] paints.

// DropCaretLook is implemented by looks that paint that caret themselves.
// Looks that do not get the default below, which is built from their own
// palette and metrics, so every pack marks the gap in its own colours
// without an engine of its own.
type DropCaretLook interface {
	// DrawDropCaret marks b as the gap a dragged item would be inserted
	// into. b is the caret's own rectangle — a thin bar lying along the
	// gap, [DropCaretRect] — and vertical says it runs down the gap
	// between two items side by side (a tab strip) rather than across the
	// gap between two stacked ones (a list, a tree, a table).
	DrawDropCaret(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool)
}

// DrawDropCaretOf marks b as the gap a dragged item would be inserted
// into: a bar in the pack's accent with a wedge at its leading end, so the
// mark reads as an insertion point and not as the edge of an item.
func DrawDropCaretOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	if lk == nil || ctx == nil || b.Empty() {
		return
	}
	if d, ok := lk.(DropCaretLook); ok {
		d.DrawDropCaret(ctx, b, vertical)
		return
	}
	drawPlainDropCaret(lk, ctx, b, vertical)
}

// DropCaretThickness is how thick a pack draws the caret: two
// device-independent pixels, and never less than one whole device pixel —
// a caret thinner than that is antialiased into a grey smear and stops
// reading as a line at all.
func DropCaretThickness(lk LookAndFeel) float32 {
	w := Dip(lk, 2)
	if w < 1 {
		w = 1
	}
	return w
}

// DropCaretRect is the bar a caret is drawn as: a line of the pack's caret
// thickness centred on the gap at at, running from lo to hi along the
// other axis. vertical is as in [DropCaretLook.DrawDropCaret] — at is an x
// for a vertical caret and a y for a horizontal one.
//
// The line is snapped to whole pixels: a 2 px caret centred on a half
// pixel is drawn as two 1 px half-lit rows, which looks like a seam
// between rows rather than a mark laid over them.
func DropCaretRect(lk LookAndFeel, at, lo, hi float32, vertical bool) paintengine2d.Rect {
	if hi <= lo {
		return paintengine2d.Rect{}
	}
	w := DropCaretThickness(lk)
	at = float32(math.Round(float64(at - w*0.5)))
	if vertical {
		return paintengine2d.XYWH(at, lo, w, hi-lo)
	}
	return paintengine2d.XYWH(lo, at, hi-lo, w)
}

// drawPlainDropCaret is the default mark, built from the pack's own
// palette and metrics.
func drawPlainDropCaret(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	ink := dropIndicatorInk(lk.Palette())
	ctx.DrawRect(b, paintengine2d.Fill(ink))
	// A wedge at the leading end — the left of a line between rows, the
	// top of one between tabs — so the caret reads as an insertion point
	// and not as the edge of an item.
	p := paintengine2d.NewPath()
	if vertical {
		w := b.Dx() * 2
		p.MoveTo(b.Min.X-w, b.Min.Y)
		p.LineTo(b.Max.X+w, b.Min.Y)
		p.LineTo((b.Min.X+b.Max.X)*0.5, b.Min.Y+w*1.5)
	} else {
		h := b.Dy() * 2
		p.MoveTo(b.Min.X, b.Min.Y-h)
		p.LineTo(b.Min.X, b.Max.Y+h)
		p.LineTo(b.Min.X+h*1.5, (b.Min.Y+b.Max.Y)*0.5)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(ink))
}
