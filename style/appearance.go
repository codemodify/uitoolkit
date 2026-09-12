package style

// ThemeName is a Classic palette preset. LookAndFeel.Name() matches these.
type ThemeName string

const (
	ThemeDark  ThemeName = "dark"
	ThemeLight ThemeName = "light"
)

// CornerStyle is the chrome radius policy (Metrics.Radius / RadiusSmall).
type CornerStyle string

const (
	CornersRound  CornerStyle = "round"
	CornersSquare CornerStyle = "square"
)

// IconSetName selects the vector ToolIcon glyph set Classic draws.
type IconSetName string

const (
	// IconSetClassic is the stock rounded-stroke set (CapRound).
	IconSetClassic IconSetName = "classic"
	// IconSetSharp is an angular, square-cap set (CapSquare / JoinMiter).
	IconSetSharp IconSetName = "sharp"
)

// Appearance is the first-class toolkit skin: theme, corner policy, icon set.
// Apps apply it with Look() / Application.SetLook — not a parallel theme.
type Appearance struct {
	Theme   ThemeName
	Corners CornerStyle
	Icons   IconSetName
}

// DefaultAppearance is Dark + rounded Classic chrome + classic icons.
func DefaultAppearance() Appearance {
	return Appearance{
		Theme:   ThemeDark,
		Corners: CornersRound,
		Icons:   IconSetClassic,
	}
}

// ParseTheme accepts dark / light (empty → dark).
func ParseTheme(s string) ThemeName {
	switch s {
	case "light", "Light", "paper":
		return ThemeLight
	default:
		return ThemeDark
	}
}

// ParseCorners accepts round / square (empty → round).
func ParseCorners(s string) CornerStyle {
	switch s {
	case "square", "Square", "rect", "sharp-corners":
		return CornersSquare
	default:
		return CornersRound
	}
}

// ParseIconSet accepts classic / sharp (empty → classic).
func ParseIconSet(s string) IconSetName {
	switch s {
	case "sharp", "Sharp", "geometric":
		return IconSetSharp
	default:
		return IconSetClassic
	}
}

// Normalize fills empty fields with defaults.
func (a Appearance) Normalize() Appearance {
	a.Theme = ParseTheme(string(a.Theme))
	a.Corners = ParseCorners(string(a.Corners))
	a.Icons = ParseIconSet(string(a.Icons))
	return a
}

func (a Appearance) String() string {
	a = a.Normalize()
	return string(a.Theme) + " / " + string(a.Corners) + " / " + string(a.Icons)
}

// ApplyCorners rewrites 1× (or already-scaled) radii for the corner policy.
// Square zeros Radius and RadiusSmall. Round restores default radii when both
// are zero, scaled by FontSize so HiDPI / density stays aligned.
func ApplyCorners(m Metrics, c CornerStyle) Metrics {
	switch ParseCorners(string(c)) {
	case CornersSquare:
		m.Radius = 0
		m.RadiusSmall = 0
	default:
		if m.Radius <= 0 && m.RadiusSmall <= 0 {
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

// Look builds a Classic LookAndFeel from the three settings (1× metrics).
// Application.SetLook still applies display scale afterward.
func (a Appearance) Look() *Classic {
	a = a.Normalize()
	name := string(a.Theme)
	p := Dark()
	if a.Theme == ThemeLight {
		p = Light()
	}
	m := ApplyCorners(DefaultMetrics(), a.Corners)
	return newClassic(name, p, m, a.Corners, a.Icons)
}

// PreferredLook loads XDG appearance prefs (or defaults) as a LookAndFeel.
func PreferredLook() LookAndFeel {
	return LoadAppearance().Look()
}

// LookAppearance reads theme / corners / icons from a live look.
func LookAppearance(l LookAndFeel) Appearance {
	a := DefaultAppearance()
	if l == nil {
		return a
	}
	if c, ok := l.(*Classic); ok {
		a.Theme = ParseTheme(c.Name())
		a.Corners = c.Corners()
		a.Icons = c.Icons()
		return a
	}
	a.Theme = ParseTheme(l.Name())
	m := l.Metrics()
	if m.Radius <= 0 && m.RadiusSmall <= 0 {
		a.Corners = CornersSquare
	}
	return a
}

// WithTheme rebuilds a Classic look with a new palette, keeping metrics,
// corners, and icon set (so density / scale survive a theme toggle).
func WithTheme(look LookAndFeel, theme ThemeName) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	theme = ParseTheme(string(theme))
	name := string(theme)
	p := Dark()
	if theme == ThemeLight {
		p = Light()
	}
	return newClassic(name, p, c.Metrics(), c.Corners(), c.Icons())
}

// WithCorners rebuilds a Classic look with a new radius policy.
func WithCorners(look LookAndFeel, corners CornerStyle) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	corners = ParseCorners(string(corners))
	m := ApplyCorners(c.Metrics(), corners)
	return newClassic(c.Name(), c.Palette(), m, corners, c.Icons())
}

// WithIcons rebuilds a Classic look with a new ToolIcon glyph set.
func WithIcons(look LookAndFeel, icons IconSetName) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	icons = ParseIconSet(string(icons))
	return newClassic(c.Name(), c.Palette(), c.Metrics(), c.Corners(), icons)
}

// WithAppearance applies all three settings to an existing Classic look,
// preserving current metrics (density / scale) except corner radii.
func WithAppearance(look LookAndFeel, a Appearance) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		if look == nil {
			return a.Look()
		}
		return look
	}
	a = a.Normalize()
	name := string(a.Theme)
	p := Dark()
	if a.Theme == ThemeLight {
		p = Light()
	}
	m := ApplyCorners(c.Metrics(), a.Corners)
	return newClassic(name, p, m, a.Corners, a.Icons)
}

func classicOf(look LookAndFeel) (*Classic, bool) {
	if look == nil {
		return nil, false
	}
	c, ok := look.(*Classic)
	return c, ok
}
