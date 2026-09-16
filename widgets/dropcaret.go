package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// Dropping between rows: the insertion caret a list, a tree and a table
// draw in the gap a dragged row would land in, and the rule that decides
// whether a drop lands in a gap at all or on the row under the pointer.
//
// A view that only takes drops *onto* its rows (files onto a folder) keeps
// the row highlight it always had; one that can be reordered splits every
// row into three bands — a gap above, the row itself, a gap below — so
// that "into this folder" and "between these two" are different gestures
// and look different while the drag is still in the air.

// dropBand is how much of a row, at its top and its bottom, counts as the
// gap either side of it rather than as the row: a quarter each, leaving
// the middle half for a drop onto the row. That is the split GTK's tree
// view and Qt's item views both make, and it is the widest the gaps can
// be without a drop aimed at a folder landing beside it.
const dropBand = 0.25

// rowDropSpot is where a drop at view-space y lands in a column of count
// rows of rowH each, scrolled by offsetY: onto row row, or in the gap
// before row at. Exactly one of them is set; the other is -1.
//
// onto says the view takes a drop on a row at all. When it does not, every
// position falls in a gap — the row's own midpoint is then the boundary,
// so the caret follows the pointer without a dead band in between. Past
// the last row the gap is count, which is how a drop lands at the end.
func rowDropSpot(y, offsetY, rowH float32, count int, onto bool) (row, at int) {
	if rowH <= 0 {
		return -1, -1
	}
	if count <= 0 {
		// An empty view is one gap: a drop in it is the first row.
		return -1, 0
	}
	pos := float64((y + offsetY) / rowH)
	i := int(math.Floor(pos))
	if i < 0 {
		return -1, 0
	}
	if i >= count {
		return -1, count
	}
	frac := float32(pos - float64(i))
	switch {
	case !onto:
		if frac < 0.5 {
			return -1, i
		}
		return -1, i + 1
	case frac < dropBand:
		return -1, i
	case frac >= 1-dropBand:
		return -1, i + 1
	}
	return i, -1
}

// rowCaretRect is the bar the caret is drawn as for a gap whose view-space
// y is y, running from x0 to x1. It is pulled inside view at the two ends,
// so the caret above the first row and the one below the last are drawn
// whole rather than sliced in half by the viewport's edge.
func rowCaretRect(lk style.LookAndFeel, y, x0, x1 float32, view paintengine2d.Rect) paintengine2d.Rect {
	if view.Empty() || x1 <= x0 {
		return paintengine2d.Rect{}
	}
	if h := style.DropCaretThickness(lk) * 0.5; view.Dy() > 2*h {
		if y < view.Min.Y+h {
			y = view.Min.Y + h
		}
		if y > view.Max.Y-h {
			y = view.Max.Y - h
		}
	}
	return style.DropCaretRect(lk, y, x0, x1, false).Intersect(view)
}

// paintDropCaret draws the insertion caret for the gap before row at, in
// the view-space band from x0 to x1. rows is how many rows there are, so
// that the gap past the last one lands under it rather than off the view.
func paintDropCaret(ctx *paintengine2d.Context, lk style.LookAndFeel, at, rows int, x0, x1, top, rowH, offsetY float32, view paintengine2d.Rect) {
	if at < 0 || at > rows {
		return
	}
	r := rowCaretRect(lk, top+float32(at)*rowH-offsetY, x0, x1, view)
	if r.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(view)
	style.DrawDropCaretOf(lk, ctx, r, false)
	ctx.Restore()
}
