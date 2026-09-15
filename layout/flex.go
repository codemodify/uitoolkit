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
	var main, cross float32
	n := 0
	for _, it := range items {
		if it.Node == nil {
			continue
		}
		n++
		sz := wholePx(it.Node.Measure(childConstraints(inner, f.Axis, false)))
		mw, cw := split(sz, f.Axis)
		main += mw
		if cw > cross {
			cross = cw
		}
	}
	if n > 1 {
		main += f.Gap * float32(n-1)
	}
	sz := join(main, cross, f.Axis)
	sz.X += f.PadL + f.PadR
	sz.Y += f.PadT + f.PadB
	return c.Constrain(sz)
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
	type measured struct {
		it Item
		sz paintengine2d.Point
	}
	var ms []measured
	var mainFixed, flexSum float32
	n := 0
	cc := childConstraints(Loose(inner.Dx(), inner.Dy()), f.Axis, false)
	for _, it := range items {
		if it.Node == nil {
			continue
		}
		n++
		sz := wholePx(it.Node.Measure(cc))
		ms = append(ms, measured{it, sz})
		mw, _ := split(sz, f.Axis)
		if it.Flex > 0 {
			flexSum += it.Flex
		} else {
			mainFixed += mw
		}
	}
	gaps := float32(0)
	if n > 1 {
		gaps = f.Gap * float32(n-1)
	}
	avail, crossMax := split(inner.Size(), f.Axis)
	free := avail - mainFixed - gaps
	if free < 0 {
		free = 0
	}

	// First pass sizes.
	sizes := make([]paintengine2d.Point, len(ms))
	var used float32
	for i, m := range ms {
		mw, cw := split(m.sz, f.Axis)
		if m.it.Flex > 0 && flexSum > 0 {
			mw = free * (m.it.Flex / flexSum)
		}
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

	for i, m := range ms {
		sz := sizes[i]
		mw, cw := split(sz, f.Axis)
		var crossOff float32
		switch f.Align {
		case AlignCenter:
			crossOff = (crossMax - cw) * 0.5
		case AlignEnd:
			crossOff = crossMax - cw
		}
		r := place(inner.Min, f.Axis, cursor, crossOff, mw, cw)
		m.it.Node.Arrange(r)
		cursor += mw + space
		_ = i
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
