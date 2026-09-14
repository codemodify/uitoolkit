package style

// Density is chrome spacing: Compact / Default / Relaxed (v0.9).
// It is independent of display scale (HiDPI); apply density first, then scale.
type Density int

const (
	DensityDefault Density = iota
	DensityCompact
	DensityRelaxed
)

func (d Density) String() string {
	switch d {
	case DensityCompact:
		return "compact"
	case DensityRelaxed:
		return "relaxed"
	default:
		return "default"
	}
}

// ParseDensity accepts compact / default / relaxed (empty → default).
func ParseDensity(s string) Density {
	switch s {
	case "compact", "Compact":
		return DensityCompact
	case "relaxed", "Relaxed", "comfortable":
		return DensityRelaxed
	default:
		return DensityDefault
	}
}

// ApplyDensity rewrites 1× metrics for the chosen chrome density.
// Compact keeps a 14px body; Default and Relaxed stay at 16px.
// shiftDensity adds the density adjustment (dens - base, per metric) to m.
func shiftDensity(m, base, dens Metrics) Metrics {
	add := func(dst *float32, b, d float32) {
		if v := *dst + (d - b); v > 0 {
			*dst = v
		}
	}
	add(&m.Pad, base.Pad, dens.Pad)
	add(&m.Gap, base.Gap, dens.Gap)
	add(&m.ControlH, base.ControlH, dens.ControlH)
	add(&m.TitleBar, base.TitleBar, dens.TitleBar)
	add(&m.FontSize, base.FontSize, dens.FontSize)
	add(&m.TitleSize, base.TitleSize, dens.TitleSize)
	add(&m.FieldPad, base.FieldPad, dens.FieldPad)
	add(&m.MenuBarH, base.MenuBarH, dens.MenuBarH)
	add(&m.MenuItemH, base.MenuItemH, dens.MenuItemH)
	add(&m.TabH, base.TabH, dens.TabH)
	add(&m.StatusBarH, base.StatusBarH, dens.StatusBarH)
	add(&m.ToolBarH, base.ToolBarH, dens.ToolBarH)
	add(&m.ToolBtn, base.ToolBtn, dens.ToolBtn)
	add(&m.ComboH, base.ComboH, dens.ComboH)
	add(&m.HeaderH, base.HeaderH, dens.HeaderH)
	add(&m.AccordionH, base.AccordionH, dens.AccordionH)
	add(&m.RowH, base.RowH, dens.RowH)
	add(&m.RowPad, base.RowPad, dens.RowPad)
	return m
}

func ApplyDensity(m Metrics, d Density) Metrics {
	switch d {
	case DensityCompact:
		m.Pad = 8
		m.Gap = 6
		m.ControlH = 28
		m.TitleBar = 30
		m.FontSize = 14
		m.TitleSize = 18
		m.FieldPad = 6
		m.MenuBarH = 26
		m.MenuItemH = 24
		m.TabH = 28
		m.StatusBarH = 24
		m.ToolBarH = 30
		m.ToolBtn = 26
		m.ComboH = 26
		m.HeaderH = 24
		m.AccordionH = 26
		m.RowH = 20
		m.RowPad = 4
	case DensityRelaxed:
		m.Pad = 16
		m.Gap = 14
		m.ControlH = 40
		m.TitleBar = 42
		m.FontSize = 16
		m.TitleSize = 24
		m.FieldPad = 12
		m.MenuBarH = 36
		m.MenuItemH = 34
		m.TabH = 38
		m.StatusBarH = 32
		m.ToolBarH = 44
		m.ToolBtn = 36
		m.ComboH = 36
		m.HeaderH = 34
		m.AccordionH = 36
		m.RowH = 32
		m.RowPad = 10
	default:
		// DefaultMetrics values for density-sensitive fields.
		m.Pad = 12
		m.Gap = 10
		m.ControlH = 34
		m.TitleBar = 36
		m.FontSize = 16
		m.TitleSize = 22
		m.FieldPad = 10
		m.MenuBarH = 30
		m.MenuItemH = 28
		m.TabH = 32
		m.StatusBarH = 28
		m.ToolBarH = 36
		m.ToolBtn = 30
		m.ComboH = 30
		m.HeaderH = 28
		m.AccordionH = 30
		m.RowH = 24
		m.RowPad = 6
	}
	return m
}

// WithDensity rebuilds a Classic look at density d, keeping the look's
// display scale (Application.SetLook does not re-scale an already scaled
// look). Metrics are rebuilt from defaults + density + pack, so switching
// density cannot inherit stale values.
func WithDensity(look LookAndFeel, d Density) LookAndFeel {
	if look == nil {
		return look
	}
	c, ok := look.(*Classic)
	if !ok {
		return look
	}
	tok := c.Tokens()
	m := lookMetrics(d, c.Scale(), tok, c.Corners(), c.IconSize())
	return newClassic(c.Name(), c.Palette(), m, c.Corners(), c.Icons(), c.IconSize(), tok).
		setPack(c.Pack()).setDensity(d).setScale(c.Scale())
}

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
