package style

// BaseFontSize is the unscaled Classic body size (1× design pixels).
const BaseFontSize = 16

// LookScale is the display scale implied by look metrics (1 at default 16px).
func LookScale(lk LookAndFeel) float32 {
	if lk == nil {
		return 1
	}
	s := lk.Metrics().FontSize / BaseFontSize
	if s <= 0 {
		return 1
	}
	return s
}

// Dip scales a 1× design-pixel length by the look's display scale.
func Dip(lk LookAndFeel, v float32) float32 {
	return v * LookScale(lk)
}

// FittedRowHeight is the paint/hit height for a virtualized row.
// explicit is a 1× design value (the widget default). At 1× the value is
// kept so existing layouts stay dense; on HiDPI it grows with the baked
// font so glyphs cannot overflow row boxes.
func FittedRowHeight(lk LookAndFeel, explicit float32) float32 {
	h := explicit
	if h <= 0 {
		if lk != nil && lk.Metrics().RowH > 0 {
			h = lk.Metrics().RowH
		} else {
			h = 24
		}
	}
	if lk == nil || LookScale(lk) <= 1.01 {
		return h
	}
	if scaled := Dip(lk, h); scaled > h {
		h = scaled
	}
	pad := lk.Metrics().RowPad
	if pad <= 0 {
		pad = 6
	}
	if f := lk.Font(); f != nil {
		if min := f.Height() + pad; min > h {
			h = min
		}
	}
	return h
}
