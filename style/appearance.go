package style

import "strings"

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

// IconSetName selects chrome ToolIcons. classic / sharp are drawn in
// process. Any other sanitized name is a file set under
// ~/.config/uitoolkit/icons/<name>/ (lucide, phosphor, tabler, …).
type IconSetName string

const (
	// IconSetClassic is the stock rounded-stroke set (CapRound).
	IconSetClassic IconSetName = "classic"
	// IconSetSharp is an angular, square-cap set (CapSquare / JoinMiter).
	IconSetSharp IconSetName = "sharp"
	// IconSetLucide is the shipped Lucide PNG set (install from repo icons/).
	IconSetLucide IconSetName = "lucide"
	// IconSetPhosphor is the shipped Phosphor regular PNG set.
	IconSetPhosphor IconSetName = "phosphor"
	// IconSetTabler is the shipped Tabler outline PNG set.
	IconSetTabler IconSetName = "tabler"
	// IconSetHeroicons is the shipped Heroicons outline PNG set.
	IconSetHeroicons IconSetName = "heroicons"
	// IconSetMaterialSymbols is the shipped Material Symbols outlined PNG set.
	IconSetMaterialSymbols IconSetName = "material-symbols"
)

// Appearance is the resolved toolkit skin. Name is the color theme
// (look.json "theme": dark, light, or a user export). Corners and
// icons are independent prefs in the same file.
type Appearance struct {
	Name    string // pack id: "dark", "light", or a user export
	Theme   ThemeName
	Corners CornerStyle
	Icons   IconSetName
}

// DefaultAppearance is the embedded dark palette, round corners, classic icons.
func DefaultAppearance() Appearance {
	return Appearance{
		Name:    DefaultThemeName,
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

// ParseIconSet accepts classic / sharp / lucide / phosphor / tabler /
// heroicons / material-symbols or any sanitized file-set directory
// name (empty → classic).
func ParseIconSet(s string) IconSetName {
	s = strings.TrimSpace(s)
	switch s {
	case "sharp", "Sharp", "geometric":
		return IconSetSharp
	case "lucide", "Lucide":
		return IconSetLucide
	case "phosphor", "Phosphor":
		return IconSetPhosphor
	case "tabler", "Tabler":
		return IconSetTabler
	case "heroicons", "Heroicons":
		return IconSetHeroicons
	case "material-symbols", "Material Symbols", "material", "Material":
		return IconSetMaterialSymbols
	case "", "classic", "Classic":
		return IconSetClassic
	default:
		if name, err := SanitizeIconSetName(s); err == nil {
			return IconSetName(name)
		}
		return IconSetClassic
	}
}

// Normalize fills empty fields with defaults. An empty Name becomes the
// matching embedded palette starter (dark / light). Corners and icons
// do not change the theme name.
func (a Appearance) Normalize() Appearance {
	a.Theme = ParseTheme(string(a.Theme))
	a.Corners = ParseCorners(string(a.Corners))
	a.Icons = ParseIconSet(string(a.Icons))
	if strings.TrimSpace(a.Name) == "" {
		a.Name = StarterName(a.Theme)
	}
	return a
}

func (a Appearance) String() string {
	a = a.Normalize()
	return a.Name
}

// WithPalette switches to the embedded dark / light starter and keeps
// corners and icons (Mail View → Dark / Light).
func (a Appearance) WithPalette(theme ThemeName) Appearance {
	a = a.Normalize()
	a.Theme = ParseTheme(string(theme))
	a.Name = StarterName(a.Theme)
	return a
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
	return newClassic(name, p, m, a.Corners, a.Icons).setPack(a.Name)
}

// PreferredLook loads XDG appearance prefs (or defaults) as a LookAndFeel.
func PreferredLook() LookAndFeel {
	return LoadAppearance().Look()
}

// LookAppearance reads pack name / palette / corners / icons from a live look.
func LookAppearance(l LookAndFeel) Appearance {
	a := DefaultAppearance()
	if l == nil {
		return a
	}
	if c, ok := l.(*Classic); ok {
		a.Theme = ParseTheme(c.Name())
		a.Corners = c.Corners()
		a.Icons = c.Icons()
		a.Name = c.Pack()
		return a.Normalize()
	}
	a.Theme = ParseTheme(l.Name())
	m := l.Metrics()
	if m.Radius <= 0 && m.RadiusSmall <= 0 {
		a.Corners = CornersSquare
	}
	a.Name = ""
	return a.Normalize()
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
	return newClassic(name, p, c.Metrics(), c.Corners(), c.Icons()).
		setPack(StarterName(theme))
}

// WithCorners rebuilds a Classic look with a new radius policy.
func WithCorners(look LookAndFeel, corners CornerStyle) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	corners = ParseCorners(string(corners))
	m := ApplyCorners(c.Metrics(), corners)
	return newClassic(c.Name(), c.Palette(), m, corners, c.Icons()).
		setPack(c.Pack())
}

// WithIcons rebuilds a Classic look with a new ToolIcon set (file or
// drawn). The theme pack name is kept; icons are independent of the pack.
func WithIcons(look LookAndFeel, icons IconSetName) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	icons = ParseIconSet(string(icons))
	return newClassic(c.Name(), c.Palette(), c.Metrics(), c.Corners(), icons).
		setPack(c.Pack())
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
	pack := a.Name
	if _, ok := LoadTheme(pack); !ok {
		pack = StarterName(a.Theme)
	}
	name := string(a.Theme)
	p := Dark()
	if a.Theme == ThemeLight {
		p = Light()
	}
	m := ApplyCorners(c.Metrics(), a.Corners)
	return newClassic(name, p, m, a.Corners, a.Icons).setPack(pack)
}

func classicOf(look LookAndFeel) (*Classic, bool) {
	if look == nil {
		return nil, false
	}
	c, ok := look.(*Classic)
	return c, ok
}
