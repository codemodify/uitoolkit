package layout

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Child is the layout protocol a flex parent talks to.
type Child interface {
	Measure(c Constraints) paintengine2d.Point
	Arrange(r paintengine2d.Rect)
}

// Item is one flex child with an optional grow weight.
type Item struct {
	Node Child
	Flex float32
}

// Flex measures and arranges children along Axis.
type Flex struct {
	Axis    Axis
	Gap     float32
	Pad     paintengine2d.Rect // used as LTRB inset via Min/Max
	PadL    float32
	PadT    float32
	PadR    float32
	PadB    float32
	Align   Align
	Justify Justify
}

// Insets is shorthand padding.
func Insets(v float32) (l, t, r, b float32) { return v, v, v, v }

// Measure returns the box that fits children under c.
func (f Flex) Measure(c Constraints, items []Item) paintengine2d.Point {
	inner := c.Inset(f.PadL+f.PadR, f.PadT+f.PadB)
	cc := childConstraints(inner, f.Axis, false)
	kids, sizes := measurePass(items, cc, f.Axis)
	if len(kids) == 0 {
		sz := join(0, 0, f.Axis)
		sz.X += f.PadL + f.PadR
		sz.Y += f.PadT + f.PadB
		return c.Constrain(sz)
	}
	gaps := f.Gap * float32(len(kids)-1)
	avail, _ := split(paintengine2d.Pt(inner.MaxW, inner.MaxH), f.Axis)
	if !mainBounded(inner, f.Axis) {
		avail = -1
	}
	f.heightForWidth(kids, sizes, cc, avail-gaps)

	var main, cross float32
	for _, sz := range sizes {
		mw, cw := split(sz, f.Axis)
		main += mw
		if cw > cross {
			cross = cw
		}
	}
	main += gaps
	sz := join(main, cross, f.Axis)
	sz.X += f.PadL + f.PadR
	sz.Y += f.PadT + f.PadB
	return c.Constrain(sz)
}

// measurePass measures every non-nil child once, loose along the main
// axis, and returns the children and their sizes side by side.
func measurePass(items []Item, cc Constraints, axis Axis) ([]Item, []paintengine2d.Point) {
	kids := make([]Item, 0, len(items))
	sizes := make([]paintengine2d.Point, 0, len(items))
	for _, it := range items {
		if it.Node == nil {
			continue
		}
		kids = append(kids, it)
		sizes = append(sizes, wholePx(it.Node.Measure(cc)))
	}
	return kids, sizes
}

func mainBounded(c Constraints, axis Axis) bool {
	if axis == AxisHorizontal {
		return c.HasMaxW()
	}
	return c.HasMaxH()
}

// distribute gives each flex child its share of what the fixed ones left
// and returns every child's final main-axis size. A main axis with no
// bound (avail < 0) leaves them all at what they measured.
func (f Flex) distribute(kids []Item, sizes []paintengine2d.Point, avail float32) []float32 {
	out := make([]float32, len(kids))
	var fixed, flexSum float32
	for i, it := range kids {
		mw, _ := split(sizes[i], f.Axis)
		out[i] = mw
		if it.Flex > 0 {
			flexSum += it.Flex
		} else {
			fixed += mw
		}
	}
	if avail < 0 || flexSum <= 0 {
		return out
	}
	free := avail - fixed
	if free < 0 {
		free = 0
	}
	for i, it := range kids {
		if it.Flex > 0 {
			out[i] = free * (it.Flex / flexSum)
		}
	}
	return out
}

// heightForWidth is the second measure pass, and it is the reason a
// wrapping label in a row is not clipped.
//
// A flex child is measured before the row knows how wide it will be: a
// wrapping label reports the one line its whole text fits on, is then
// arranged narrower, wraps to three, and draws them centred in a
// one-line box with the first and last cut off. The only answer is to
// ask again once the width is known — Qt's heightForWidth, GTK's
// height-for-width, WPF's second measure pass — which is what this does
// for the children whose main-axis size the distribution changed.
//
// It only ever **grows** the recorded cross size. A child that wants
// less at its final width is left alone: there is no bug in a box that
// is too big for its contents, and shrinking here would move the
// geometry of every row in the toolkit for nothing.
//
// Horizontal only. Width-for-height is the same idea with the axes
// swapped and nothing in this toolkit needs it, so asking for it would
// be one more measure per child of every column, bought with nothing.
func (f Flex) heightForWidth(kids []Item, sizes []paintengine2d.Point, cc Constraints, avail float32) {
	if f.Axis != AxisHorizontal {
		return
	}
	finals := f.distribute(kids, sizes, avail)
	for i, it := range kids {
		mw, cw := split(sizes[i], f.Axis)
		if finals[i] == mw {
			continue
		}
		again := wholePx(it.Node.Measure(Constraints{
			MinH: cc.MinH, MaxH: cc.MaxH, MaxW: finals[i],
		}))
		if _, ncw := split(again, f.Axis); ncw > cw {
			sizes[i] = join(mw, ncw, f.Axis)
		}
	}
}

// Arrange places children into bounds.
func (f Flex) Arrange(bounds paintengine2d.Rect, items []Item) {
	// Children are arranged in the parent's local space (0,0 is the
	// parent's top-left). PaintTree translates by Bounds().Min.
	inner := paintengine2d.Rect{
		Min: paintengine2d.Pt(f.PadL, f.PadT),
		Max: paintengine2d.Pt(bounds.Dx()-f.PadR, bounds.Dy()-f.PadB),
	}
	if inner.Empty() {
		for _, it := range items {
			if it.Node != nil {
				it.Node.Arrange(paintengine2d.Rect{})
			}
		}
		return
	}
	cc := childConstraints(Loose(inner.Dx(), inner.Dy()), f.Axis, false)
	kids, measured := measurePass(items, cc, f.Axis)
	n := len(kids)
	gaps := float32(0)
	if n > 1 {
		gaps = f.Gap * float32(n-1)
	}
	avail, crossMax := split(inner.Size(), f.Axis)
	// The height-for-width pass, before the sizes are fixed: a wrapping
	// label that folds to three lines at its final width has to be
	// arranged three lines tall, or it draws outside its box.
	f.heightForWidth(kids, measured, cc, avail-gaps)
	finals := f.distribute(kids, measured, avail-gaps)

	sizes := make([]paintengine2d.Point, n)
	var used float32
	for i := range kids {
		mw := finals[i]
		_, cw := split(measured[i], f.Axis)
		if f.Align == AlignStretch {
			cw = crossMax
		}
		sizes[i] = join(mw, cw, f.Axis)
		used += mw
	}
	used += gaps

	var cursor float32
	switch f.Justify {
	case JustifyCenter:
		cursor = (avail - used) * 0.5
	case JustifyEnd:
		cursor = avail - used
	default:
		cursor = 0
	}
	space := float32(0)
	if f.Justify == JustifySpaceBetween && n > 1 {
		cursor = 0
		space = (avail - (used - gaps)) / float32(n-1)
		if space < f.Gap {
			space = f.Gap
		}
	} else {
		space = f.Gap
	}

	for i, it := range kids {
		mw, cw := split(sizes[i], f.Axis)
		var crossOff float32
		switch f.Align {
		case AlignCenter:
			crossOff = (crossMax - cw) * 0.5
		case AlignEnd:
			crossOff = crossMax - cw
		}
		it.Node.Arrange(place(inner.Min, f.Axis, cursor, crossOff, mw, cw))
		cursor += mw + space
	}
}

func split(sz paintengine2d.Point, axis Axis) (main, cross float32) {
	if axis == AxisHorizontal {
		return sz.X, sz.Y
	}
	return sz.Y, sz.X
}

func join(main, cross float32, axis Axis) paintengine2d.Point {
	if axis == AxisHorizontal {
		return paintengine2d.Pt(main, cross)
	}
	return paintengine2d.Pt(cross, main)
}

func place(origin paintengine2d.Point, axis Axis, main, cross, mw, cw float32) paintengine2d.Rect {
	if axis == AxisHorizontal {
		return paintengine2d.XYWH(origin.X+main, origin.Y+cross, mw, cw)
	}
	return paintengine2d.XYWH(origin.X+cross, origin.Y+main, cw, mw)
}

func childConstraints(parent Constraints, axis Axis, stretch bool) Constraints {
	_ = stretch
	if axis == AxisHorizontal {
		return Constraints{MinH: parent.MinH, MaxH: parent.MaxH, MaxW: parent.MaxW}
	}
	return Constraints{MinW: parent.MinW, MaxW: parent.MaxW, MaxH: parent.MaxH}
}

// wholePx rounds a measured size up to whole pixels. Components land on
// whole pixels (layout rounding rounds each edge); a whole-pixel size keeps
// its exact width wherever it lands, so a label measured to its text is
// never shaved a fraction short and elided.
func wholePx(p paintengine2d.Point) paintengine2d.Point {
	return paintengine2d.Pt(float32(math.Ceil(float64(p.X)-1e-3)), float32(math.Ceil(float64(p.Y)-1e-3)))
}
