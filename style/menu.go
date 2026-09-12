package style

// MenuChrome is the shared measure/paint box for a PopupMenu row.
// Lengths are host pixels (Look scale and density already applied).
type MenuChrome struct {
	PadL, PadR, PadT, PadB float32
	ItemPad                float32 // DrawMenuItem inner pad (check side + label trailing)
	CheckGutter            float32 // space between check/icon and label
	AccelGap               float32
	Border                 float32 // frame/clip so the last glyph is not bitten
}

// CheckCol is the label origin relative to the item rect (pad + gutter).
func (c MenuChrome) CheckCol() float32 { return c.ItemPad + c.CheckGutter }

// LabelMinX is the painted label origin inside an arranged item row.
func (c MenuChrome) LabelMinX(rowMinX float32) float32 { return rowMinX + c.CheckCol() }

// LabelMaxX is the reserved advance right edge (item pad stays free for ink).
func (c MenuChrome) LabelMaxX(rowMaxX float32) float32 { return rowMaxX - c.ItemPad }

// FrameWidth is the intrinsic menu width for the widest label/shortcut.
func (c MenuChrome) FrameWidth(maxLabel, maxAccel float32) float32 {
	w := c.PadL + c.CheckCol() + maxLabel + c.ItemPad + c.PadR + c.Border
	if maxAccel > 0 {
		w += c.AccelGap + maxAccel
	}
	return w
}

// MenuChromeFor returns DrawMenuItem / PopupMenu metrics from the host look.
// Scale and density come from Look metrics so measure and paint stay aligned.
func MenuChromeFor(lk LookAndFeel) MenuChrome {
	c := MenuChrome{
		PadL:        8,
		PadR:        10,
		PadT:        6,
		PadB:        8,
		ItemPad:     10,
		CheckGutter: 10,
		AccelGap:    20,
		Border:      2,
	}
	if lk == nil {
		return c
	}
	s := LookScale(lk)
	if s <= 0 {
		s = 1
	}
	c.PadL *= s
	c.PadR *= s
	c.PadT *= s
	c.PadB *= s
	c.ItemPad *= s
	c.CheckGutter *= s
	c.AccelGap *= s
	c.Border *= s
	m := lk.Metrics()
	if m.RowPad > c.ItemPad {
		c.ItemPad = m.RowPad
	}
	return c
}
