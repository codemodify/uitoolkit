package layout

import "github.com/codemodify/paintengine2d"

// Constraints is a Measure box: min/max width and height. Max < 0 means unbounded.
type Constraints struct {
	MinW, MinH float32
	MaxW, MaxH float32
}

// Tight forces an exact size.
func Tight(w, h float32) Constraints {
	return Constraints{MinW: w, MinH: h, MaxW: w, MaxH: h}
}

// Loose is 0..size.
func Loose(w, h float32) Constraints {
	return Constraints{MaxW: w, MaxH: h}
}

// Unbounded has no max on either axis.
func Unbounded() Constraints {
	return Constraints{MaxW: -1, MaxH: -1}
}

func (c Constraints) HasMaxW() bool { return c.MaxW >= 0 }
func (c Constraints) HasMaxH() bool { return c.MaxH >= 0 }

// Inset shrinks max (and min if needed) by padding on both axes.
func (c Constraints) Inset(dx, dy float32) Constraints {
	out := c
	out.MinW = max0(c.MinW - dx)
	out.MinH = max0(c.MinH - dy)
	if c.HasMaxW() {
		out.MaxW = max0(c.MaxW - dx)
	}
	if c.HasMaxH() {
		out.MaxH = max0(c.MaxH - dy)
	}
	return out
}

// Constrain clamps size into the box.
func (c Constraints) Constrain(sz paintengine2d.Point) paintengine2d.Point {
	w, h := sz.X, sz.Y
	if w < c.MinW {
		w = c.MinW
	}
	if h < c.MinH {
		h = c.MinH
	}
	if c.HasMaxW() && w > c.MaxW {
		w = c.MaxW
	}
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	return paintengine2d.Pt(w, h)
}

// Axis is the main axis of a flex layout.
type Axis int

const (
	AxisHorizontal Axis = iota
	AxisVertical
)

// Align is cross-axis alignment.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// Justify is main-axis packing.
type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

func max0(v float32) float32 {
	if v < 0 {
		return 0
	}
	return v
}
