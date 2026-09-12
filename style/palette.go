package style

import "github.com/codemodify/paintengine2d"

// Palette is the semantic color set a LookAndFeel draws from.
type Palette struct {
	Background   paintengine2d.Color
	Surface      paintengine2d.Color
	SurfaceAlt   paintengine2d.Color
	Overlay      paintengine2d.Color
	Border       paintengine2d.Color
	Divider      paintengine2d.Color
	Text         paintengine2d.Color
	TextMuted    paintengine2d.Color
	TextOnAccent paintengine2d.Color
	Accent       paintengine2d.Color
	AccentHover  paintengine2d.Color
	AccentPress  paintengine2d.Color
	Danger       paintengine2d.Color
	Success      paintengine2d.Color
	Warning      paintengine2d.Color
	Track        paintengine2d.Color
	Thumb        paintengine2d.Color
	Field        paintengine2d.Color
	FieldBorder  paintengine2d.Color
	Focus        paintengine2d.Color
	Selection    paintengine2d.Color
	Shadow       paintengine2d.Color
	Highlight    paintengine2d.Color
	// Menu hover chrome (Office XP hot-track). Themes tint these;
	// zero channels fall back to Accent / SurfaceAlt via ResolveMenuChrome.
	MenuHover       paintengine2d.Color
	MenuHoverBorder paintengine2d.Color
	MenuGutter      paintengine2d.Color
}

// Dark is the default night theme (cool graphite + blue accent).
func Dark() Palette {
	return Palette{
		Background:      paintengine2d.RGB(0.10, 0.11, 0.14),
		Surface:         paintengine2d.RGB(0.14, 0.16, 0.20),
		SurfaceAlt:      paintengine2d.RGB(0.18, 0.20, 0.25),
		Overlay:         paintengine2d.RGBA(0.04, 0.05, 0.07, 0.55),
		Border:          paintengine2d.RGB(0.28, 0.31, 0.38),
		Divider:         paintengine2d.RGB(0.22, 0.24, 0.30),
		Text:            paintengine2d.RGB(0.91, 0.93, 0.96),
		TextMuted:       paintengine2d.RGB(0.62, 0.66, 0.73),
		TextOnAccent:    paintengine2d.RGB(0.98, 0.99, 1.00),
		Accent:          paintengine2d.RGB(0.29, 0.56, 0.95),
		AccentHover:     paintengine2d.RGB(0.38, 0.64, 0.98),
		AccentPress:     paintengine2d.RGB(0.22, 0.44, 0.82),
		Danger:          paintengine2d.RGB(0.90, 0.32, 0.36),
		Success:         paintengine2d.RGB(0.30, 0.72, 0.50),
		Warning:         paintengine2d.RGB(0.93, 0.72, 0.28),
		Track:           paintengine2d.RGB(0.22, 0.24, 0.30),
		Thumb:           paintengine2d.RGB(0.42, 0.46, 0.54),
		Field:           paintengine2d.RGB(0.11, 0.12, 0.16),
		FieldBorder:     paintengine2d.RGB(0.32, 0.35, 0.42),
		Focus:           paintengine2d.RGB(0.45, 0.70, 1.00),
		Selection:       paintengine2d.RGBA(0.29, 0.56, 0.95, 0.38),
		Shadow:          paintengine2d.RGBA(0, 0, 0, 0.35),
		Highlight:       paintengine2d.RGBA(1, 1, 1, 0.08),
		MenuHover:       paintengine2d.RGB(0.22, 0.34, 0.52),
		MenuHoverBorder: paintengine2d.RGB(0.38, 0.58, 0.88),
		MenuGutter:      paintengine2d.RGB(0.14, 0.16, 0.20),
	}
}

// Light is a paper / ink theme with the same accent family.
func Light() Palette {
	return Palette{
		Background:      paintengine2d.RGB(0.93, 0.94, 0.96),
		Surface:         paintengine2d.RGB(0.99, 0.99, 1.00),
		SurfaceAlt:      paintengine2d.RGB(0.90, 0.92, 0.95),
		Overlay:         paintengine2d.RGBA(0.12, 0.14, 0.18, 0.35),
		Border:          paintengine2d.RGB(0.72, 0.75, 0.80),
		Divider:         paintengine2d.RGB(0.82, 0.84, 0.88),
		Text:            paintengine2d.RGB(0.12, 0.14, 0.18),
		TextMuted:       paintengine2d.RGB(0.38, 0.42, 0.48),
		TextOnAccent:    paintengine2d.RGB(1, 1, 1),
		Accent:          paintengine2d.RGB(0.18, 0.42, 0.86),
		AccentHover:     paintengine2d.RGB(0.24, 0.50, 0.92),
		AccentPress:     paintengine2d.RGB(0.14, 0.32, 0.70),
		Danger:          paintengine2d.RGB(0.78, 0.18, 0.22),
		Success:         paintengine2d.RGB(0.16, 0.58, 0.38),
		Warning:         paintengine2d.RGB(0.78, 0.54, 0.10),
		Track:           paintengine2d.RGB(0.80, 0.82, 0.86),
		Thumb:           paintengine2d.RGB(0.48, 0.52, 0.58),
		Field:           paintengine2d.RGB(1, 1, 1),
		FieldBorder:     paintengine2d.RGB(0.68, 0.72, 0.78),
		Focus:           paintengine2d.RGB(0.18, 0.42, 0.86),
		Selection:       paintengine2d.RGBA(0.18, 0.42, 0.86, 0.28),
		Shadow:          paintengine2d.RGBA(0.10, 0.12, 0.16, 0.18),
		Highlight:       paintengine2d.RGBA(1, 1, 1, 0.55),
		MenuHover:       paintengine2d.RGB(0.68, 0.80, 0.95),
		MenuHoverBorder: paintengine2d.RGB(0.19, 0.42, 0.77),
		MenuGutter:      paintengine2d.RGB(0.86, 0.86, 0.88),
	}
}

// colorUnset is the zero Color (user palettes that omit a field).
func colorUnset(c paintengine2d.Color) bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// ResolveMenuChrome fills MenuHover / MenuHoverBorder / MenuGutter when
// unset so a custom Accent still tints Office XP menu chrome.
func ResolveMenuChrome(p Palette) Palette {
	if colorUnset(p.MenuHover) {
		p.MenuHover = p.SurfaceAlt.Lerp(p.Accent, 0.42)
		p.MenuHover.A = 1
	}
	if colorUnset(p.MenuHoverBorder) {
		p.MenuHoverBorder = p.Accent.Lerp(p.Border, 0.22)
		p.MenuHoverBorder.A = 1
	}
	if colorUnset(p.MenuGutter) {
		p.MenuGutter = p.SurfaceAlt.Lerp(p.Background, 0.38)
		p.MenuGutter.A = 1
	}
	return p
}

// Metrics is the geometric skin (radii, paddings). Apps swap LookAndFeel,
// not a global singleton. Square Appearance sets Radius and RadiusSmall to 0.
type Metrics struct {
	Radius      float32
	RadiusSmall float32
	Pad         float32
	Gap         float32
	Border      float32
	FocusWidth  float32
	ControlH    float32
	Checkbox    float32
	SliderH     float32
	Thumb       float32
	Scroll      float32
	Splitter    float32
	TitleBar    float32
	FontSize    float32
	TitleSize   float32
	FontFamily  string
	MonoFamily  string
	Stroke      float32
	FieldPad    float32
	MenuBarH    float32
	MenuItemH   float32
	TabH        float32
	TreeIndent  float32
	StatusBarH  float32
	ToolBarH    float32
	ToolBtn     float32
	ProgressH   float32
	Radio       float32
	HeaderH     float32
	SpinnerW    float32
	TooltipPad  float32
	SwitchW     float32
	SwitchH     float32
	AccordionH  float32
	RowH        float32 // list / table / tree row
	RowPad      float32 // ink padding inside a row
}

// ScaleMetrics multiplies spatial LookAndFeel metrics by display scale.
// Font sizes, padding, and control heights grow; hairline strokes stay ≥ 1.
func ScaleMetrics(m Metrics, scale float32) Metrics {
	if scale <= 0 || scale == 1 {
		return m
	}
	s := func(v float32) float32 { return v * scale }
	m.Radius = s(m.Radius)
	m.RadiusSmall = s(m.RadiusSmall)
	m.Pad = s(m.Pad)
	m.Gap = s(m.Gap)
	m.Border = s(m.Border)
	if m.Border < 1 {
		m.Border = 1
	}
	m.FocusWidth = s(m.FocusWidth)
	m.ControlH = s(m.ControlH)
	m.Checkbox = s(m.Checkbox)
	m.SliderH = s(m.SliderH)
	m.Thumb = s(m.Thumb)
	m.Scroll = s(m.Scroll)
	m.Splitter = s(m.Splitter)
	m.TitleBar = s(m.TitleBar)
	m.FontSize = s(m.FontSize)
	m.TitleSize = s(m.TitleSize)
	m.Stroke = s(m.Stroke)
	if m.Stroke < 1 {
		m.Stroke = 1
	}
	m.FieldPad = s(m.FieldPad)
	m.MenuBarH = s(m.MenuBarH)
	m.MenuItemH = s(m.MenuItemH)
	m.TabH = s(m.TabH)
	m.TreeIndent = s(m.TreeIndent)
	m.StatusBarH = s(m.StatusBarH)
	m.ToolBarH = s(m.ToolBarH)
	m.ToolBtn = s(m.ToolBtn)
	m.ProgressH = s(m.ProgressH)
	m.Radio = s(m.Radio)
	m.HeaderH = s(m.HeaderH)
	m.SpinnerW = s(m.SpinnerW)
	m.TooltipPad = s(m.TooltipPad)
	m.SwitchW = s(m.SwitchW)
	m.SwitchH = s(m.SwitchH)
	m.AccordionH = s(m.AccordionH)
	m.RowH = s(m.RowH)
	m.RowPad = s(m.RowPad)
	return m
}

// DefaultMetrics is dense but finger-friendly at 1× scale.
func DefaultMetrics() Metrics {
	return Metrics{
		Radius:      8,
		RadiusSmall: 5,
		Pad:         12,
		Gap:         10,
		Border:      1,
		FocusWidth:  2,
		ControlH:    34,
		Checkbox:    18,
		SliderH:     28,
		Thumb:       16,
		Scroll:      10,
		Splitter:    6,
		TitleBar:    36,
		FontSize:    16,
		TitleSize:   22,
		FontFamily:  DefaultFontFamily,
		MonoFamily:  DefaultMonoFamily,
		Stroke:      1.7,
		FieldPad:    10,
		MenuBarH:    30,
		MenuItemH:   28,
		TabH:        32,
		TreeIndent:  16,
		StatusBarH:  28,
		ToolBarH:    36,
		ToolBtn:     30,
		ProgressH:   14,
		Radio:       18,
		HeaderH:     28,
		SpinnerW:    22,
		TooltipPad:  8,
		SwitchW:     42,
		SwitchH:     22,
		AccordionH:  30,
		RowH:        24,
		RowPad:      6,
	}
}

// Align is a 2D alignment hint for labels and flex cross-axis.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)
