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
	// BevelLight / BevelDark are the 3D highlight and shadow edges
	// (Win95 / Motif). Unset values are filled by ResolveBevelChrome.
	BevelLight paintengine2d.Color
	BevelDark  paintengine2d.Color
}

// Dark is the Classic 95 night twin (charcoal 3D + navy select).
func Dark() Palette {
	return Palette{
		Background:      paintengine2d.RGB(0.16, 0.16, 0.16),
		Surface:         paintengine2d.RGB(0.22, 0.22, 0.22),
		SurfaceAlt:      paintengine2d.RGB(0.26, 0.26, 0.26),
		Overlay:         paintengine2d.RGBA(0.04, 0.04, 0.04, 0.55),
		Border:          paintengine2d.RGB(0.12, 0.12, 0.12),
		Divider:         paintengine2d.RGB(0.18, 0.18, 0.18),
		Text:            paintengine2d.RGB(0.92, 0.92, 0.92),
		TextMuted:       paintengine2d.RGB(0.62, 0.62, 0.62),
		TextOnAccent:    paintengine2d.RGB(1, 1, 1),
		Accent:          paintengine2d.RGB(0.00, 0.00, 0.50),
		AccentHover:     paintengine2d.RGB(0.12, 0.22, 0.70),
		AccentPress:     paintengine2d.RGB(0.00, 0.00, 0.35),
		Danger:          paintengine2d.RGB(0.80, 0.22, 0.22),
		Success:         paintengine2d.RGB(0.22, 0.58, 0.32),
		Warning:         paintengine2d.RGB(0.80, 0.62, 0.16),
		Track:           paintengine2d.RGB(0.22, 0.22, 0.22),
		Thumb:           paintengine2d.RGB(0.32, 0.32, 0.32),
		Field:           paintengine2d.RGB(0.12, 0.12, 0.12),
		FieldBorder:     paintengine2d.RGB(0.08, 0.08, 0.08),
		Focus:           paintengine2d.RGB(0.00, 0.00, 0.00),
		Selection:       paintengine2d.RGBA(0.00, 0.00, 0.50, 0.85),
		Shadow:          paintengine2d.RGBA(0, 0, 0, 0.45),
		Highlight:       paintengine2d.RGBA(1, 1, 1, 0.10),
		MenuHover:       paintengine2d.RGB(0.00, 0.00, 0.50),
		MenuHoverBorder: paintengine2d.RGB(0.00, 0.00, 0.25),
		MenuGutter:      paintengine2d.RGB(0.14, 0.14, 0.14),
		BevelLight:      paintengine2d.RGB(0.55, 0.55, 0.55),
		BevelDark:       paintengine2d.RGB(0.08, 0.08, 0.08),
	}
}

// Light is Classic 95 / Win95 silver with navy selection.
func Light() Palette {
	return Palette{
		Background:      paintengine2d.RGB(0.75, 0.75, 0.75),
		Surface:         paintengine2d.RGB(0.75, 0.75, 0.75),
		SurfaceAlt:      paintengine2d.RGB(0.75, 0.75, 0.75),
		Overlay:         paintengine2d.RGBA(0.12, 0.12, 0.12, 0.35),
		Border:          paintengine2d.RGB(0.50, 0.50, 0.50),
		Divider:         paintengine2d.RGB(0.50, 0.50, 0.50),
		Text:            paintengine2d.RGB(0, 0, 0),
		TextMuted:       paintengine2d.RGB(0.50, 0.50, 0.50),
		TextOnAccent:    paintengine2d.RGB(1, 1, 1),
		Accent:          paintengine2d.RGB(0.00, 0.00, 0.50),
		AccentHover:     paintengine2d.RGB(0.00, 0.00, 0.65),
		AccentPress:     paintengine2d.RGB(0.00, 0.00, 0.35),
		Danger:          paintengine2d.RGB(0.75, 0.15, 0.15),
		Success:         paintengine2d.RGB(0.12, 0.50, 0.22),
		Warning:         paintengine2d.RGB(0.75, 0.55, 0.10),
		Track:           paintengine2d.RGB(0.75, 0.75, 0.75),
		Thumb:           paintengine2d.RGB(0.75, 0.75, 0.75),
		Field:           paintengine2d.RGB(1, 1, 1),
		FieldBorder:     paintengine2d.RGB(0.50, 0.50, 0.50),
		Focus:           paintengine2d.RGB(0, 0, 0),
		Selection:       paintengine2d.RGBA(0.00, 0.00, 0.50, 0.90),
		Shadow:          paintengine2d.RGBA(0, 0, 0, 0.35),
		Highlight:       paintengine2d.RGBA(1, 1, 1, 0.85),
		MenuHover:       paintengine2d.RGB(0.00, 0.00, 0.50),
		MenuHoverBorder: paintengine2d.RGB(0.00, 0.00, 0.25),
		MenuGutter:      paintengine2d.RGB(0.62, 0.62, 0.62),
		BevelLight:      paintengine2d.RGB(1, 1, 1),
		BevelDark:       paintengine2d.RGB(0.50, 0.50, 0.50),
	}
}

// colorUnset is the zero Color (user palettes that omit a field).
//
// A theme that asks for fully transparent black would collide with "absent",
// so the JSON reader marks such a value explicit with [markExplicitColor].
func colorUnset(c paintengine2d.Color) bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// clearAlpha is the presence marker for a colour a theme set to fully
// transparent black ("#00000000" / "#0000"). It still quantises to alpha 0
// in 8-bit output, so the colour paints nothing — it just is not "unset".
const clearAlpha = 1.0 / 512.0

// markExplicitColor tags a parsed colour as present. Only transparent black
// needs it; every other value is already distinguishable from the zero Color.
func markExplicitColor(c paintengine2d.Color) paintengine2d.Color {
	if colorUnset(c) {
		c.A = clearAlpha
	}
	return c
}

// paletteFamily guesses dark / light from a palette, for callers of
// [ResolveBevelChrome] that do not carry a family.
func paletteFamily(p Palette) ThemeName {
	lum := func(c paintengine2d.Color) float32 { return 0.299*c.R + 0.587*c.G + 0.114*c.B }
	if lum(p.Text) > lum(p.Background) {
		return ThemeDark
	}
	return ThemeLight
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
	ComboH      float32 // closed ComboBox; toolbar-friendly, not grown with icon size
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
	ViewFrame   float32 // frame around list / tree / table views (0: flat)
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
	m.ViewFrame = s(m.ViewFrame)
	m.TreeIndent = s(m.TreeIndent)
	m.StatusBarH = s(m.StatusBarH)
	m.ToolBarH = s(m.ToolBarH)
	m.ToolBtn = s(m.ToolBtn)
	m.ComboH = s(m.ComboH)
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
		ComboH:      30,
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

// ComboHeight is the closed ComboBox row. ComboH (else ToolBtn) stays
// flush with toolbar chrome and is not grown by icon-size ToolBtn bumps;
// ControlH stays the taller standalone button.
func ComboHeight(m Metrics) float32 {
	if m.ComboH > 0 {
		return m.ComboH
	}
	if m.ToolBtn > 0 {
		return m.ToolBtn
	}
	return m.ControlH
}

// FieldHeight is the closed TextField / NumberField row. Same metric as
// ComboHeight — ControlH was oversized next to ToolBtn chrome.
func FieldHeight(m Metrics) float32 { return ComboHeight(m) }

// Align is a 2D alignment hint for labels and flex cross-axis.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)
