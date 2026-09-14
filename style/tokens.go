package style

import (
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// BevelStyle is the control-chrome language a theme pack paints with.
type BevelStyle string

const (
	// BevelNone is flat fill + 1px border (Breeze / FlatLaf).
	BevelNone BevelStyle = "none"
	// BevelClassic3D is Motif / Win95 inset and outset edges.
	BevelClassic3D BevelStyle = "classic-3d"
	// BevelLunaHottrack is Office XP pale fill + thin border on hot/press/focus.
	BevelLunaHottrack BevelStyle = "luna-hottrack"
	// BevelSoftShadow is Aqua / Material raised faces with drop shadow.
	BevelSoftShadow BevelStyle = "soft-shadow"
	// BevelFluentAccent is Fluent-lite: wash + accent focus bar.
	BevelFluentAccent BevelStyle = "fluent-accent"
)

// ParseBevel accepts the five chrome languages (empty → none).
func ParseBevel(s string) BevelStyle {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(BevelClassic3D), "classic", "win95", "motif":
		return BevelClassic3D
	case string(BevelLunaHottrack), "luna", "hottrack", "xp":
		return BevelLunaHottrack
	case string(BevelSoftShadow), "soft", "aqua", "material":
		return BevelSoftShadow
	case string(BevelFluentAccent), "fluent":
		return BevelFluentAccent
	default:
		return BevelNone
	}
}

// ChromeState is fill + border for one interactive state.
type ChromeState struct {
	Fill   paintengine2d.Color
	Border paintengine2d.Color
}

// ChromeMetrics are theme-specific geometry knobs. Zero means “use DefaultMetrics
// (or corners / density overlays)”. Corners still override radii.
type ChromeMetrics struct {
	Radius      float32
	RadiusSmall float32
	BevelDepth  float32
	GutterWidth float32
	Scroll      float32
	ControlH    float32
	FieldH      float32
	ComboH      float32
	Elevation   int

	// Square marks a natively square-cornered look (Win95, Motif, NeXT):
	// with the "theme" Corners pref its radii stay 0.
	Square bool

	// Engine-level geometry (all 1x design pixels; 0 = keep the default).
	Checkbox   float32 // checkbox indicator side
	Radio      float32 // radio indicator side
	MenuItemH  float32
	MenuBarH   float32
	TabH       float32
	RowH       float32
	TitleBar   float32 // panel / dialog caption band
	HeaderH    float32 // table header
	ProgressH  float32
	SliderH    float32
	Thumb      float32 // slider thumb
	Pad        float32
	FieldPad   float32
	ToolBarH   float32
	StatusBarH float32
	SpinnerW   float32
	Border     float32
	FocusWidth float32
	SwitchW    float32
	SwitchH    float32
}

// MergeChromeMetrics fills every zero field of over from base (over wins).
func MergeChromeMetrics(base, over ChromeMetrics) ChromeMetrics {
	pick := func(a, b float32) float32 {
		if b != 0 {
			return b
		}
		return a
	}
	out := over
	out.Square = over.Square || (base.Square && over.Radius == 0 && over.RadiusSmall == 0)
	out.Radius = pick(base.Radius, over.Radius)
	out.RadiusSmall = pick(base.RadiusSmall, over.RadiusSmall)
	out.BevelDepth = pick(base.BevelDepth, over.BevelDepth)
	out.GutterWidth = pick(base.GutterWidth, over.GutterWidth)
	out.Scroll = pick(base.Scroll, over.Scroll)
	out.ControlH = pick(base.ControlH, over.ControlH)
	out.FieldH = pick(base.FieldH, over.FieldH)
	out.ComboH = pick(base.ComboH, over.ComboH)
	if over.Elevation == 0 {
		out.Elevation = base.Elevation
	}
	out.Checkbox = pick(base.Checkbox, over.Checkbox)
	out.Radio = pick(base.Radio, over.Radio)
	out.MenuItemH = pick(base.MenuItemH, over.MenuItemH)
	out.MenuBarH = pick(base.MenuBarH, over.MenuBarH)
	out.TabH = pick(base.TabH, over.TabH)
	out.RowH = pick(base.RowH, over.RowH)
	out.TitleBar = pick(base.TitleBar, over.TitleBar)
	out.HeaderH = pick(base.HeaderH, over.HeaderH)
	out.ProgressH = pick(base.ProgressH, over.ProgressH)
	out.SliderH = pick(base.SliderH, over.SliderH)
	out.Thumb = pick(base.Thumb, over.Thumb)
	out.Pad = pick(base.Pad, over.Pad)
	out.FieldPad = pick(base.FieldPad, over.FieldPad)
	out.ToolBarH = pick(base.ToolBarH, over.ToolBarH)
	out.StatusBarH = pick(base.StatusBarH, over.StatusBarH)
	out.SpinnerW = pick(base.SpinnerW, over.SpinnerW)
	out.Border = pick(base.Border, over.Border)
	out.FocusWidth = pick(base.FocusWidth, over.FocusWidth)
	out.SwitchW = pick(base.SwitchW, over.SwitchW)
	out.SwitchH = pick(base.SwitchH, over.SwitchH)
	return out
}

// ThemeTokens is the first-class skin a theme pack can fully specify.
// Palette + bevel + state colors + metrics drive every LookAndFeel Draw*.
type ThemeTokens struct {
	Bevel  BevelStyle
	Family ThemeName
	Era    string
	// Engine is the registry id of the painter ("" → base: the Bevel modes).
	Engine string
	// Extra holds engine-specific named colours (theme.json "extra"), read
	// by engines through Classic.X.
	Extra map[string]paintengine2d.Color
	// Params holds engine-specific numbers (theme.json "params"), read
	// through Classic.P.
	Params   map[string]float32
	Metrics  ChromeMetrics
	Palette  Palette
	Hot      ChromeState
	Pressed  ChromeState
	Disabled ChromeState
	Selected ChromeState
	Focus    ChromeState
}

// Empty reports a zero-value token set (legacy palette-only packs).
func (t ThemeTokens) Empty() bool {
	return t.Bevel == "" && t.Engine == "" && len(t.Extra) == 0 && len(t.Params) == 0 &&
		t.Metrics == (ChromeMetrics{}) &&
		colorUnset(t.Palette.Background) && colorUnset(t.Palette.Surface) &&
		colorUnset(t.Hot.Fill)
}

// Resolve fills derived palette and state colors from Family / Dark / Light.
func (t ThemeTokens) Resolve() ThemeTokens {
	if t.Family != ThemeLight {
		t.Family = ParseTheme(string(t.Family))
	}
	base := Dark()
	if t.Family == ThemeLight {
		base = Light()
	}
	t.Palette = overlayPalette(base, t.Palette)
	t.Palette = ResolveMenuChrome(t.Palette)
	t.Palette = ResolveBevelChromeFor(t.Palette, t.Family)
	if e, ok := EngineByID(t.Engine); ok {
		t.Metrics = MergeChromeMetrics(e.DefaultMetrics(), t.Metrics)
	}
	t.Metrics = clampChromeMetrics(t.Metrics)
	if t.Bevel == "" {
		t.Bevel = BevelNone
	}
	if colorUnset(t.Hot.Fill) {
		t.Hot.Fill = t.Palette.MenuHover
		if colorUnset(t.Hot.Fill) {
			t.Hot.Fill = t.Palette.Highlight
		}
	}
	if colorUnset(t.Hot.Border) {
		t.Hot.Border = t.Palette.MenuHoverBorder
		if colorUnset(t.Hot.Border) {
			t.Hot.Border = t.Palette.Accent
		}
	}
	if colorUnset(t.Pressed.Fill) {
		t.Pressed.Fill = t.Hot.Fill.Lerp(t.Palette.AccentPress, 0.28)
		t.Pressed.Fill.A = 1
	}
	if colorUnset(t.Pressed.Border) {
		t.Pressed.Border = t.Hot.Border
	}
	if colorUnset(t.Selected.Fill) {
		t.Selected.Fill = t.Palette.Selection
		if t.Selected.Fill.A < 0.08 {
			t.Selected.Fill = t.Palette.Accent.WithAlpha(0.28)
		}
	}
	if colorUnset(t.Selected.Border) {
		t.Selected.Border = t.Palette.Accent
	}
	if colorUnset(t.Disabled.Fill) {
		t.Disabled.Fill = t.Palette.Surface
	}
	if colorUnset(t.Disabled.Border) {
		t.Disabled.Border = t.Palette.Divider
	}
	if colorUnset(t.Focus.Fill) {
		t.Focus.Fill = t.Palette.Focus.WithAlpha(0.16)
	}
	if colorUnset(t.Focus.Border) {
		t.Focus.Border = t.Palette.Focus
	}
	if t.Metrics.BevelDepth <= 0 && t.Bevel == BevelClassic3D {
		t.Metrics.BevelDepth = 1
	}
	return t
}

func overlayPalette(base, over Palette) Palette {
	out := base
	overlayColor := func(dst *paintengine2d.Color, src paintengine2d.Color) {
		if !colorUnset(src) {
			*dst = src
		}
	}
	overlayColor(&out.Background, over.Background)
	overlayColor(&out.Surface, over.Surface)
	overlayColor(&out.SurfaceAlt, over.SurfaceAlt)
	overlayColor(&out.Overlay, over.Overlay)
	overlayColor(&out.Border, over.Border)
	overlayColor(&out.Divider, over.Divider)
	overlayColor(&out.Text, over.Text)
	overlayColor(&out.TextMuted, over.TextMuted)
	overlayColor(&out.TextOnAccent, over.TextOnAccent)
	overlayColor(&out.Accent, over.Accent)
	overlayColor(&out.AccentHover, over.AccentHover)
	overlayColor(&out.AccentPress, over.AccentPress)
	overlayColor(&out.Danger, over.Danger)
	overlayColor(&out.Success, over.Success)
	overlayColor(&out.Warning, over.Warning)
	overlayColor(&out.Track, over.Track)
	overlayColor(&out.Thumb, over.Thumb)
	overlayColor(&out.Field, over.Field)
	overlayColor(&out.FieldBorder, over.FieldBorder)
	overlayColor(&out.Focus, over.Focus)
	overlayColor(&out.Selection, over.Selection)
	overlayColor(&out.Shadow, over.Shadow)
	overlayColor(&out.Highlight, over.Highlight)
	overlayColor(&out.MenuHover, over.MenuHover)
	overlayColor(&out.MenuHoverBorder, over.MenuHoverBorder)
	overlayColor(&out.MenuGutter, over.MenuGutter)
	overlayColor(&out.BevelLight, over.BevelLight)
	overlayColor(&out.BevelDark, over.BevelDark)
	return out
}

// ResolveBevelChrome fills BevelLight / BevelDark when a pack omitted them.
// The palette family is inferred; [ResolveBevelChromeFor] takes it directly.
func ResolveBevelChrome(p Palette) Palette {
	return ResolveBevelChromeFor(p, paletteFamily(p))
}

// ResolveBevelChromeFor fills BevelLight / BevelDark for a known family.
func ResolveBevelChromeFor(p Palette, fam ThemeName) Palette {
	if colorUnset(p.BevelLight) {
		if p.Highlight.A > 0.2 && (p.Highlight.R+p.Highlight.G+p.Highlight.B) > 1.6 {
			p.BevelLight = paintengine2d.RGB(p.Highlight.R, p.Highlight.G, p.Highlight.B)
		} else if ParseTheme(string(fam)) == ThemeDark && p.Text.R > 0.6 {
			p.BevelLight = p.SurfaceAlt.Lerp(paintengine2d.RGB(1, 1, 1), 0.45)
			p.BevelLight.A = 1
		} else {
			p.BevelLight = paintengine2d.RGB(1, 1, 1)
		}
	}
	if colorUnset(p.BevelDark) {
		if !colorUnset(p.Shadow) && p.Shadow.A > 0.4 {
			p.BevelDark = paintengine2d.RGB(p.Shadow.R, p.Shadow.G, p.Shadow.B)
		} else {
			p.BevelDark = p.Border
			if colorUnset(p.BevelDark) {
				p.BevelDark = paintengine2d.RGB(0.25, 0.25, 0.25)
			}
		}
	}
	return p
}

// ApplyChromeMetrics overlays non-zero theme metrics onto m.
func ApplyChromeMetrics(m Metrics, cm ChromeMetrics) Metrics {
	if cm.Radius > 0 {
		m.Radius = cm.Radius
	}
	if cm.RadiusSmall > 0 {
		m.RadiusSmall = cm.RadiusSmall
	}
	if cm.Scroll > 0 {
		m.Scroll = cm.Scroll
	}
	if cm.ControlH > 0 {
		m.ControlH = cm.ControlH
	}
	if cm.ComboH > 0 {
		m.ComboH = cm.ComboH
	}
	if cm.FieldH > 0 && cm.ComboH <= 0 {
		m.ComboH = cm.FieldH
	}
	set := func(dst *float32, v float32) {
		if v > 0 {
			*dst = v
		}
	}
	set(&m.Checkbox, cm.Checkbox)
	set(&m.Radio, cm.Radio)
	set(&m.MenuItemH, cm.MenuItemH)
	set(&m.MenuBarH, cm.MenuBarH)
	set(&m.TabH, cm.TabH)
	set(&m.RowH, cm.RowH)
	set(&m.TitleBar, cm.TitleBar)
	set(&m.HeaderH, cm.HeaderH)
	set(&m.ProgressH, cm.ProgressH)
	set(&m.SliderH, cm.SliderH)
	set(&m.Thumb, cm.Thumb)
	set(&m.Pad, cm.Pad)
	set(&m.FieldPad, cm.FieldPad)
	set(&m.ToolBarH, cm.ToolBarH)
	set(&m.StatusBarH, cm.StatusBarH)
	set(&m.SpinnerW, cm.SpinnerW)
	set(&m.Border, cm.Border)
	set(&m.FocusWidth, cm.FocusWidth)
	set(&m.SwitchW, cm.SwitchW)
	set(&m.SwitchH, cm.SwitchH)
	return m
}

// applyPackCorners applies the Corners pref against a pack's metrics:
// theme keeps the pack's native shape (square packs stay square), round
// forces rounded radii, square zeros them.
func applyPackCorners(m Metrics, c CornerStyle, cm ChromeMetrics) Metrics {
	switch ParseCorners(string(c)) {
	case CornersTheme:
		if cm.Square {
			m.Radius, m.RadiusSmall = 0, 0
			return m
		}
		return ApplyThemeCorners(m, CornersRound, cm.Radius, cm.RadiusSmall)
	case CornersRound:
		return ApplyThemeCorners(m, CornersRound, cm.Radius, cm.RadiusSmall)
	default:
		return ApplyThemeCorners(m, CornersSquare, 0, 0)
	}
}

// ApplyThemeCorners applies the orthogonal Corners pref. Square zeros radii.
// Round uses theme radii when set, otherwise stock DefaultMetrics radii
// (scaled to the current font size).
func ApplyThemeCorners(m Metrics, c CornerStyle, themeRadius, themeRadiusSmall float32) Metrics {
	switch ParseCorners(string(c)) {
	case CornersSquare:
		m.Radius = 0
		m.RadiusSmall = 0
	default:
		if themeRadius > 0 || themeRadiusSmall > 0 {
			if themeRadius > 0 {
				m.Radius = themeRadius
			}
			if themeRadiusSmall > 0 {
				m.RadiusSmall = themeRadiusSmall
			}
		} else if m.Radius <= 0 && m.RadiusSmall <= 0 {
			def := DefaultMetrics()
			s := float32(1)
			if def.FontSize > 0 && m.FontSize > 0 {
				s = m.FontSize / def.FontSize
			}
			m.Radius = def.Radius * s
			m.RadiusSmall = def.RadiusSmall * s
		}
	}
	return m
}

// ParseHexColor accepts #rgb, #rrggbb, #rrggbbaa (optional #).
func ParseHexColor(s string) (paintengine2d.Color, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "#")
	if s == "" {
		return paintengine2d.Color{}, false
	}
	expand := func(c byte) byte {
		v, err := strconv.ParseUint(string([]byte{c}), 16, 8)
		if err != nil {
			return 0
		}
		return byte(v*16 + v)
	}
	var r, g, b, a uint8
	a = 255
	switch len(s) {
	case 3:
		r, g, b = expand(s[0]), expand(s[1]), expand(s[2])
	case 4:
		r, g, b, a = expand(s[0]), expand(s[1]), expand(s[2]), expand(s[3])
	case 6, 8:
		n, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return paintengine2d.Color{}, false
		}
		if len(s) == 6 {
			r = uint8(n >> 16)
			g = uint8(n >> 8)
			b = uint8(n)
		} else {
			r = uint8(n >> 24)
			g = uint8(n >> 16)
			b = uint8(n >> 8)
			a = uint8(n)
		}
	default:
		return paintengine2d.Color{}, false
	}
	return paintengine2d.RGBA(float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255), true
}

func hexColor(s string) paintengine2d.Color {
	c, ok := ParseHexColor(s)
	if !ok {
		return paintengine2d.Color{}
	}
	return c
}

func hexAlpha(s string, a float32) paintengine2d.Color {
	c := hexColor(s)
	c.A = a
	return c
}

func colorHexPadded(c paintengine2d.Color) string {
	if colorUnset(c) {
		return ""
	}
	r := int(clamp1(c.R)*255 + 0.5)
	g := int(clamp1(c.G)*255 + 0.5)
	b := int(clamp1(c.B)*255 + 0.5)
	a := int(clamp1(c.A)*255 + 0.5)
	if c.A == 0 && (c.R != 0 || c.G != 0 || c.B != 0) {
		a = 0
	}
	if a >= 254 {
		return "#" + pad2(r) + pad2(g) + pad2(b)
	}
	return "#" + pad2(r) + pad2(g) + pad2(b) + pad2(a)
}

func pad2(v int) string {
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	s := strconv.FormatInt(int64(v), 16)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

type chromeMetricsJSON struct {
	Radius      *float32 `json:"radius,omitempty"`
	RadiusSmall *float32 `json:"radiusSmall,omitempty"`
	BevelDepth  *float32 `json:"bevelDepth,omitempty"`
	GutterWidth *float32 `json:"gutterWidth,omitempty"`
	Scroll      *float32 `json:"scroll,omitempty"`
	ControlH    *float32 `json:"controlH,omitempty"`
	FieldH      *float32 `json:"fieldH,omitempty"`
	ComboH      *float32 `json:"comboH,omitempty"`
	Elevation   *int     `json:"elevation,omitempty"`
	Square      *bool    `json:"square,omitempty"`
	Checkbox    *float32 `json:"checkbox,omitempty"`
	Radio       *float32 `json:"radio,omitempty"`
	MenuItemH   *float32 `json:"menuItemH,omitempty"`
	MenuBarH    *float32 `json:"menuBarH,omitempty"`
	TabH        *float32 `json:"tabH,omitempty"`
	RowH        *float32 `json:"rowH,omitempty"`
	TitleBar    *float32 `json:"titleBar,omitempty"`
	HeaderH     *float32 `json:"headerH,omitempty"`
	ProgressH   *float32 `json:"progressH,omitempty"`
	SliderH     *float32 `json:"sliderH,omitempty"`
	Thumb       *float32 `json:"thumb,omitempty"`
	Pad         *float32 `json:"pad,omitempty"`
	FieldPad    *float32 `json:"fieldPad,omitempty"`
	ToolBarH    *float32 `json:"toolBarH,omitempty"`
	StatusBarH  *float32 `json:"statusBarH,omitempty"`
	SpinnerW    *float32 `json:"spinnerW,omitempty"`
	Border      *float32 `json:"border,omitempty"`
	FocusWidth  *float32 `json:"focusWidth,omitempty"`
	SwitchW     *float32 `json:"switchW,omitempty"`
	SwitchH     *float32 `json:"switchH,omitempty"`
}

// geometry pairs the engine-level JSON fields with their ChromeMetrics
// fields so encode / decode / clamp stay in one table.
func (j *chromeMetricsJSON) geometry(cm *ChromeMetrics) []struct {
	name string
	js   **float32
	v    *float32
} {
	return []struct {
		name string
		js   **float32
		v    *float32
	}{
		{"checkbox", &j.Checkbox, &cm.Checkbox},
		{"radio", &j.Radio, &cm.Radio},
		{"menuItemH", &j.MenuItemH, &cm.MenuItemH},
		{"menuBarH", &j.MenuBarH, &cm.MenuBarH},
		{"tabH", &j.TabH, &cm.TabH},
		{"rowH", &j.RowH, &cm.RowH},
		{"titleBar", &j.TitleBar, &cm.TitleBar},
		{"headerH", &j.HeaderH, &cm.HeaderH},
		{"progressH", &j.ProgressH, &cm.ProgressH},
		{"sliderH", &j.SliderH, &cm.SliderH},
		{"thumb", &j.Thumb, &cm.Thumb},
		{"pad", &j.Pad, &cm.Pad},
		{"fieldPad", &j.FieldPad, &cm.FieldPad},
		{"toolBarH", &j.ToolBarH, &cm.ToolBarH},
		{"statusBarH", &j.StatusBarH, &cm.StatusBarH},
		{"spinnerW", &j.SpinnerW, &cm.SpinnerW},
		{"border", &j.Border, &cm.Border},
		{"focusWidth", &j.FocusWidth, &cm.FocusWidth},
		{"switchW", &j.SwitchW, &cm.SwitchW},
		{"switchH", &j.SwitchH, &cm.SwitchH},
	}
}

func (cm ChromeMetrics) json() *chromeMetricsJSON {
	if cm == (ChromeMetrics{}) {
		return nil
	}
	out := &chromeMetricsJSON{}
	setF := func(dst **float32, v float32) {
		if v != 0 {
			x := v
			*dst = &x
		}
	}
	setF(&out.Radius, cm.Radius)
	setF(&out.RadiusSmall, cm.RadiusSmall)
	setF(&out.BevelDepth, cm.BevelDepth)
	setF(&out.GutterWidth, cm.GutterWidth)
	setF(&out.Scroll, cm.Scroll)
	setF(&out.ControlH, cm.ControlH)
	setF(&out.FieldH, cm.FieldH)
	setF(&out.ComboH, cm.ComboH)
	if cm.Elevation != 0 {
		e := cm.Elevation
		out.Elevation = &e
	}
	for _, g := range out.geometry(&cm) {
		setF(g.js, *g.v)
	}
	if cm.Square {
		sq := true
		out.Square = &sq
	}
	return out
}

// Chrome metric bounds. Zero always means "unset"; any other value is
// clamped into a range a control can actually paint, so a typo
// ("scroll": 160) or a hostile theme.json cannot produce screen-sized
// chrome or a 2e9-pixel shadow offset.
const (
	maxThemeRadius = 64
	maxBevelDepth  = 4
	maxGutterWidth = 256
	minThemeScroll = 2
	maxThemeScroll = 64
	minControlSide = 8
	maxControlSide = 256
	maxElevation   = 8
)

// clampMetric keeps v inside [min, max]. Zero stays zero (unset); negative
// and non-finite values fall back to unset. Every change is logged once at
// the point a theme is parsed or resolved.
func clampMetric(name string, v, min, max float32) float32 {
	if v == 0 {
		return 0
	}
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		log.Printf("uitk theme: %s is not a finite number, ignoring", name)
		return 0
	}
	if v < 0 {
		log.Printf("uitk theme: %s %v is negative, ignoring", name, v)
		return 0
	}
	if v < min {
		log.Printf("uitk theme: %s %v below %v, clamped", name, v, min)
		return min
	}
	if v > max {
		log.Printf("uitk theme: %s %v above %v, clamped", name, v, max)
		return max
	}
	return v
}

// clampChromeMetrics bounds every theme-supplied geometry knob.
func clampChromeMetrics(cm ChromeMetrics) ChromeMetrics {
	cm.Radius = clampMetric("metrics.radius", cm.Radius, 0, maxThemeRadius)
	cm.RadiusSmall = clampMetric("metrics.radiusSmall", cm.RadiusSmall, 0, maxThemeRadius)
	cm.BevelDepth = clampMetric("metrics.bevelDepth", cm.BevelDepth, 0, maxBevelDepth)
	cm.GutterWidth = clampMetric("metrics.gutterWidth", cm.GutterWidth, 0, maxGutterWidth)
	cm.Scroll = clampMetric("metrics.scroll", cm.Scroll, minThemeScroll, maxThemeScroll)
	cm.ControlH = clampMetric("metrics.controlH", cm.ControlH, minControlSide, maxControlSide)
	cm.FieldH = clampMetric("metrics.fieldH", cm.FieldH, minControlSide, maxControlSide)
	cm.ComboH = clampMetric("metrics.comboH", cm.ComboH, minControlSide, maxControlSide)
	if cm.Elevation < 0 {
		log.Printf("uitk theme: metrics.elevation %d is negative, ignoring", cm.Elevation)
		cm.Elevation = 0
	}
	if cm.Elevation > maxElevation {
		log.Printf("uitk theme: metrics.elevation %d above %d, clamped", cm.Elevation, maxElevation)
		cm.Elevation = maxElevation
	}
	var j chromeMetricsJSON
	for _, g := range j.geometry(&cm) {
		max := float32(maxControlSide)
		switch g.name {
		case "pad", "fieldPad", "border", "focusWidth":
			max = 64
		}
		*g.v = clampMetric("metrics."+g.name, *g.v, 1, max)
	}
	return cm
}

func (j *chromeMetricsJSON) metrics() ChromeMetrics {
	if j == nil {
		return ChromeMetrics{}
	}
	var cm ChromeMetrics
	if j.Radius != nil {
		cm.Radius = *j.Radius
	}
	if j.RadiusSmall != nil {
		cm.RadiusSmall = *j.RadiusSmall
	}
	if j.BevelDepth != nil {
		cm.BevelDepth = *j.BevelDepth
	}
	if j.GutterWidth != nil {
		cm.GutterWidth = *j.GutterWidth
	}
	if j.Scroll != nil {
		cm.Scroll = *j.Scroll
	}
	if j.ControlH != nil {
		cm.ControlH = *j.ControlH
	}
	if j.FieldH != nil {
		cm.FieldH = *j.FieldH
	}
	if j.ComboH != nil {
		cm.ComboH = *j.ComboH
	}
	if j.Elevation != nil {
		cm.Elevation = *j.Elevation
	}
	for _, g := range j.geometry(&cm) {
		if *g.js != nil {
			*g.v = **g.js
		}
	}
	if j.Square != nil {
		cm.Square = *j.Square
	}
	return clampChromeMetrics(cm)
}

func tokensToColorMap(t ThemeTokens) map[string]string {
	p := t.Palette
	m := map[string]string{}
	put := func(k string, c paintengine2d.Color) {
		if s := colorHexPadded(c); s != "" {
			m[k] = s
		}
	}
	put("background", p.Background)
	put("surface", p.Surface)
	put("surfaceAlt", p.SurfaceAlt)
	put("overlay", p.Overlay)
	put("border", p.Border)
	put("divider", p.Divider)
	put("text", p.Text)
	put("textMuted", p.TextMuted)
	put("textOnAccent", p.TextOnAccent)
	put("accent", p.Accent)
	put("accentHover", p.AccentHover)
	put("accentPress", p.AccentPress)
	put("danger", p.Danger)
	put("success", p.Success)
	put("warning", p.Warning)
	put("track", p.Track)
	put("thumb", p.Thumb)
	put("field", p.Field)
	put("fieldBorder", p.FieldBorder)
	put("focus", p.Focus)
	put("selection", p.Selection)
	put("shadow", p.Shadow)
	put("highlight", p.Highlight)
	put("menuHover", p.MenuHover)
	put("menuHoverBorder", p.MenuHoverBorder)
	put("menuGutter", p.MenuGutter)
	put("bevelLight", p.BevelLight)
	put("bevelDark", p.BevelDark)
	put("hotFill", t.Hot.Fill)
	put("hotBorder", t.Hot.Border)
	put("pressedFill", t.Pressed.Fill)
	put("pressedBorder", t.Pressed.Border)
	put("selectedFill", t.Selected.Fill)
	put("selectedBorder", t.Selected.Border)
	put("disabledFill", t.Disabled.Fill)
	put("disabledBorder", t.Disabled.Border)
	put("focusFill", t.Focus.Fill)
	put("focusBorder", t.Focus.Border)
	if len(m) == 0 {
		return nil
	}
	return m
}

func applyColorMap(t ThemeTokens, m map[string]string) ThemeTokens {
	if len(m) == 0 {
		return t
	}
	get := func(k string) (paintengine2d.Color, bool) {
		s, ok := m[k]
		if !ok {
			return paintengine2d.Color{}, false
		}
		return ParseHexColor(s)
	}
	// A colour present in the file wins even when it is fully transparent
	// black: markExplicitColor keeps it out of the "unset" bucket so a theme
	// can switch off a shadow or a selection wash.
	setP := func(dst *paintengine2d.Color, k string) {
		if c, ok := get(k); ok {
			*dst = markExplicitColor(c)
		}
	}
	setP(&t.Palette.Background, "background")
	setP(&t.Palette.Surface, "surface")
	setP(&t.Palette.SurfaceAlt, "surfaceAlt")
	setP(&t.Palette.Overlay, "overlay")
	setP(&t.Palette.Border, "border")
	setP(&t.Palette.Divider, "divider")
	setP(&t.Palette.Text, "text")
	setP(&t.Palette.TextMuted, "textMuted")
	setP(&t.Palette.TextOnAccent, "textOnAccent")
	setP(&t.Palette.Accent, "accent")
	setP(&t.Palette.AccentHover, "accentHover")
	setP(&t.Palette.AccentPress, "accentPress")
	setP(&t.Palette.Danger, "danger")
	setP(&t.Palette.Success, "success")
	setP(&t.Palette.Warning, "warning")
	setP(&t.Palette.Track, "track")
	setP(&t.Palette.Thumb, "thumb")
	setP(&t.Palette.Field, "field")
	setP(&t.Palette.FieldBorder, "fieldBorder")
	setP(&t.Palette.Focus, "focus")
	setP(&t.Palette.Selection, "selection")
	setP(&t.Palette.Shadow, "shadow")
	setP(&t.Palette.Highlight, "highlight")
	setP(&t.Palette.MenuHover, "menuHover")
	setP(&t.Palette.MenuHoverBorder, "menuHoverBorder")
	setP(&t.Palette.MenuGutter, "menuGutter")
	setP(&t.Palette.BevelLight, "bevelLight")
	setP(&t.Palette.BevelDark, "bevelDark")
	setP(&t.Hot.Fill, "hotFill")
	setP(&t.Hot.Border, "hotBorder")
	setP(&t.Pressed.Fill, "pressedFill")
	setP(&t.Pressed.Border, "pressedBorder")
	setP(&t.Selected.Fill, "selectedFill")
	setP(&t.Selected.Border, "selectedBorder")
	setP(&t.Disabled.Fill, "disabledFill")
	setP(&t.Disabled.Border, "disabledBorder")
	setP(&t.Focus.Fill, "focusFill")
	setP(&t.Focus.Border, "focusBorder")
	return t
}

func tokensFromJSON(doc themeFileJSON) ThemeTokens {
	fam := doc.Family
	if fam == "" {
		fam = doc.Palette
	}
	if fam == "" {
		fam = doc.Theme
	}
	t := ThemeTokens{
		Bevel:   ParseBevel(doc.Bevel),
		Family:  ParseTheme(fam),
		Era:     strings.TrimSpace(doc.Era),
		Engine:  strings.ToLower(strings.TrimSpace(doc.Engine)),
		Metrics: doc.Metrics.metrics(),
	}
	for k, v := range doc.Extra {
		if c, ok := ParseHexColor(v); ok {
			if t.Extra == nil {
				t.Extra = map[string]paintengine2d.Color{}
			}
			t.Extra[k] = markExplicitColor(c)
		}
	}
	for k, v := range doc.Params {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			continue
		}
		if t.Params == nil {
			t.Params = map[string]float32{}
		}
		t.Params[k] = v
	}
	if doc.Elevation != 0 && t.Metrics.Elevation == 0 {
		t.Metrics.Elevation = doc.Elevation
	}
	t = applyColorMap(t, doc.Colors)
	// Legacy { "palette": "dark" } with no colors keeps family only;
	// Resolve + builtin merge restore the era pack.
	if doc.Palette == "dark" || doc.Palette == "light" {
		if t.Family == "" {
			t.Family = ParseTheme(doc.Palette)
		}
	}
	return t
}
