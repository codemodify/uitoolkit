package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Wrap lays its children out left to right and starts a new line when the
// next one would not fit (Qt's flow layout, GTK's FlowBox, WPF's and
// Avalonia's WrapPanel): a row of buttons folds on a narrow window instead
// of running off it. Its height follows the width it is given.
type Wrap struct {
	widget.Base
	Gap     float32 // between items on a line, 1x (0 = 8)
	LineGap float32 // between lines, 1x (0 = Gap)
	// TrailRight pushes the last item of every line to the right edge of
	// the box, where the line has room and more than one item on it. A
	// line of one is left where it is: pushing the only thing on a line
	// to the far side reads as a different block, not as an aligned one.
	//
	// It is for a row whose last item is of a different kind from the
	// rest — Settings' renderer among its typefaces — and it keeps the
	// folding, which pinning the item to a fixed column would not.
	TrailRight bool
}

// NewWrap flows children into lines.
func NewWrap(children ...widget.Component) *Wrap {
	w := &Wrap{}
	w.Init(w)
	for _, c := range children {
		if c != nil {
			w.Add(c)
		}
	}
	return w
}

func (w *Wrap) gaps() (gap, line float32) {
	lk := w.Look()
	g := w.Gap
	if g <= 0 {
		g = 8
	}
	lg := w.LineGap
	if lg <= 0 {
		lg = g
	}
	return style.Dip(lk, g), style.Dip(lk, lg)
}

// flow places the visible children in lines no wider than maxW (< 0: one
// line) and reports each child's rect and the extent they cover.
func (w *Wrap) flow(maxW float32) ([]paintengine2d.Rect, paintengine2d.Point) {
	// Flow at a whole-pixel width, because that is what Arrange will
	// have: SetBounds snaps a component's bounds to the pixel grid, so a
	// measure taken at 300.4 and a layout done at 300 disagree — one
	// fits a child on the line that the other folds, and the difference
	// shows as a line's height of empty space under the last row, or a
	// row running past the edge. Rounding both to the same grid makes
	// the two passes answer the same question.
	if maxW > 0 {
		maxW = float32(math.Round(float64(maxW)))
	}
	gap, lineGap := w.gaps()
	kids := w.Children()
	rects := make([]paintengine2d.Rect, len(kids))
	var x, y, lineH, width float32
	started := false
	for i, c := range kids {
		if !c.Visible() {
			continue
		}
		sz := c.Measure(layout.Constraints{MaxW: maxW, MaxH: -1})
		cw, ch := float32(math.Ceil(float64(sz.X)-1e-3)), float32(math.Ceil(float64(sz.Y)-1e-3))
		if started && maxW >= 0 && x+gap+cw > maxW {
			// A new line.
			y += lineH + lineGap
			x, lineH, started = 0, 0, false
		}
		if started {
			x += gap
		}
		rects[i] = paintengine2d.XYWH(x, y, cw, ch)
		x += cw
		lineH = max(lineH, ch)
		width = max(width, x)
		started = true
	}
	// Centre each item in its line's height.
	if w.TrailRight && maxW > 0 {
		w.trailRight(rects, maxW)
	}
	return w.centreLines(rects), paintengine2d.Pt(width, y+lineH)
}

// trailRight slides the last item of each line out to the right edge.
// The slack is whatever the line did not use; a line of one item keeps
// it, and the item never moves left, so a line that is already full is
// untouched.
func (w *Wrap) trailRight(rects []paintengine2d.Rect, maxW float32) {
	kids := w.Children()
	last, count := -1, 0
	flush := func() {
		if last >= 0 && count > 1 {
			if slack := maxW - rects[last].Max.X; slack > 0 {
				rects[last] = rects[last].Translate(paintengine2d.Pt(slack, 0))
			}
		}
		last, count = -1, 0
	}
	var line float32 = -1
	for i, c := range kids {
		if !c.Visible() {
			continue
		}
		if last >= 0 && rects[i].Min.Y != line {
			flush()
		}
		line, last, count = rects[i].Min.Y, i, count+1
	}
	flush()
}

// centreLines centres every item vertically within its line.
func (w *Wrap) centreLines(rects []paintengine2d.Rect) []paintengine2d.Rect {
	lineH := map[float32]float32{}
	for i, r := range rects {
		if w.Children()[i].Visible() {
			lineH[r.Min.Y] = max(lineH[r.Min.Y], r.Dy())
		}
	}
	for i, r := range rects {
		if h := lineH[r.Min.Y]; h > r.Dy() {
			rects[i] = r.Translate(paintengine2d.Pt(0, (h-r.Dy())*0.5))
		}
	}
	return rects
}

func (w *Wrap) Measure(c layout.Constraints) paintengine2d.Point {
	maxW := float32(-1)
	if c.HasMaxW() {
		maxW = c.MaxW
	}
	_, size := w.flow(maxW)
	return c.Constrain(size)
}

func (w *Wrap) Arrange(r paintengine2d.Rect) {
	w.SetBounds(r)
	rects, _ := w.flow(w.LocalBounds().Dx())
	for i, c := range w.Children() {
		if c.Visible() {
			c.Arrange(rects[i])
		} else {
			c.Arrange(paintengine2d.Rect{})
		}
	}
}
