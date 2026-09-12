package style

// MenuRow is the LookAndFeel paint input for one popup / context row.
type MenuRow struct {
	Label, Shortcut string
	Underline       int
	Separator       bool
	Checked         bool
	Radio           bool
	Icon            ToolIcon
}

// MenuChrome is the shared measure/paint box for a PopupMenu row.
// Lengths are host pixels (Look scale and density already applied).
type MenuChrome struct {
	PadL, PadR, PadT, PadB float32
	ItemPad                float32 // DrawMenuItem inner pad (check side + label trailing)
	CheckGutter            float32 // check/icon column after ItemPad (gray gutter)
	LabelGap               float32 // body space after the gray gutter before text
	AccelGap               float32
	Border                 float32 // frame/clip so the last glyph is not bitten
}

// CheckCol is the label origin relative to the item rect (pad + gutter).
func (c MenuChrome) CheckCol() float32 { return c.ItemPad + c.CheckGutter }

// GutterW is the gray icon column from the popup's left edge (not the label).
func (c MenuChrome) GutterW() float32 { return c.PadL + c.CheckCol() }

// LabelMinX is the painted label origin inside an arranged item row.
func (c MenuChrome) LabelMinX(rowMinX float32) float32 {
	return rowMinX + c.CheckCol() + c.LabelGap
}

// LabelMaxX is the reserved advance right edge (item pad stays free for ink).
func (c MenuChrome) LabelMaxX(rowMaxX float32) float32 { return rowMaxX - c.ItemPad }

// FrameWidth is the intrinsic menu width for the widest label/shortcut.
func (c MenuChrome) FrameWidth(maxLabel, maxAccel float32) float32 {
	w := c.PadL + c.CheckCol() + c.LabelGap + maxLabel + c.ItemPad + c.PadR + c.Border
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
		LabelGap:    8,
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
	c.LabelGap *= s
	c.AccelGap *= s
	c.Border *= s
	switch LookIconSize(lk) {
	case IconSizeSmall:
		c.CheckGutter = 8 * s
	case IconSizeLarge:
		c.CheckGutter = 14 * s
	}
	// Gutter must fit a ToolIcon (or check/radio) at the look's icon size.
	need := IconSizePixels(LookIconSize(lk))*s + 4*s
	if c.CheckCol() < need {
		c.CheckGutter = need - c.ItemPad
		if c.CheckGutter < 8*s {
			c.CheckGutter = 8 * s
		}
	}
	m := lk.Metrics()
	if m.RowPad > c.ItemPad {
		c.ItemPad = m.RowPad
	}
	return c
}
