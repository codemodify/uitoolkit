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
	// CornersTheme keeps each theme's native shape (Win95 square, Aqua
	// pill, Luna rounded). It is the default.
	CornersTheme  CornerStyle = "theme"
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

// IconSize is the chrome ToolIcon draw size (look.json "iconSize").
// Named values map to 1× pixels; HiDPI still picks name@2x.png when
// the destination is large enough — packs are not re-exported per size.
type IconSize string

const (
	IconSizeSmall  IconSize = "small"  // 16×16
	IconSizeMedium IconSize = "medium" // 24×24 (1× PNG / default)
	IconSizeLarge  IconSize = "large"  // 32×32
)

// Appearance is the resolved toolkit skin. Name is the color theme
// (look.json "theme": dark, light, or a user export). Corners, icons,
// and icon size are independent prefs in the same file.
type Appearance struct {
	Name     string // pack id: "dark", "light", or a user export
	Theme    ThemeName
	Corners  CornerStyle
	Icons    IconSetName
	IconSize IconSize
}

// DefaultAppearance is the embedded dark palette, round corners, classic icons.
func DefaultAppearance() Appearance {
	return Appearance{
		Name:     DefaultThemeName,
		Theme:    ThemeDark,
		Corners:  CornersTheme,
		Icons:    IconSetClassic,
		IconSize: IconSizeMedium,
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

// ParseCorners accepts theme / round / square (empty or unknown → theme).
func ParseCorners(s string) CornerStyle {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "square", "rect", "sharp-corners":
		return CornersSquare
	case "round", "rounded":
		return CornersRound
	default:
		return CornersTheme
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

// ParseIconSize accepts small / medium / large or 16 / 24 / 32
// (empty → medium).
func ParseIconSize(s string) IconSize {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "small", "16", "sm":
		return IconSizeSmall
	case "large", "32", "lg":
		return IconSizeLarge
	default:
		return IconSizeMedium
	}
}

// IconSizePixels is the 1× destination side for sz (16 / 24 / 32).
func IconSizePixels(sz IconSize) float32 {
	switch ParseIconSize(string(sz)) {
	case IconSizeSmall:
		return 16
	case IconSizeLarge:
		return 32
	default:
		return 24
	}
}

// Normalize fills empty fields with defaults. An empty Name becomes the
// matching embedded palette starter (dark / light). Corners, icons, and
// icon size do not change the theme name.
func (a Appearance) Normalize() Appearance {
	a.Theme = ParseTheme(string(a.Theme))
	a.Corners = ParseCorners(string(a.Corners))
	a.Icons = ParseIconSet(string(a.Icons))
	a.IconSize = ParseIconSize(string(a.IconSize))
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

// ApplyIconSize grows toolbar metrics so ToolIcons of sz fit. It never
// shrinks density/scale values already on m.
func ApplyIconSize(m Metrics, sz IconSize) Metrics {
	s := float32(1)
	def := DefaultMetrics()
	if def.FontSize > 0 && m.FontSize > 0 {
		s = m.FontSize / def.FontSize
	}
	px := IconSizePixels(sz) * s
	if m.ToolBtn < px+10*s {
		m.ToolBtn = px + 10*s
	}
	if m.ToolBarH < px+14*s {
		m.ToolBarH = px + 14*s
	}
	return m
}

// packMetrics rebuilds 1× chrome metrics from scratch: stock defaults, then
// density, then the theme pack, then corners and icon size.
//
// Rebuilding — instead of overlaying the new pack onto the live metrics —
// is what keeps a live theme switch from inheriting the previous pack's
// control heights, scrollbar width, or radii when the new pack leaves them
// unset.
func packMetrics(d Density, tok ThemeTokens, corners CornerStyle, sz IconSize) Metrics {
	// The pack (and its engine) set the default-density geometry; density
	// then shifts it by the same amounts it shifts the stock metrics, so a
	// compact Win95 still gets shorter rows than a default one.
	base := DefaultMetrics()
	m := ApplyChromeMetrics(base, tok.Metrics)
	m = shiftDensity(m, base, ApplyDensity(base, d))
	m = applyPackCorners(m, corners, tok.Metrics)
	return ApplyIconSize(m, sz)
}

// lookMetrics is packMetrics at a display scale.
func lookMetrics(d Density, scale float32, tok ThemeTokens, corners CornerStyle, sz IconSize) Metrics {
	return ScaleMetrics(packMetrics(d, tok, corners, sz), scale)
}

// Look builds a Classic LookAndFeel from the appearance prefs (1× metrics).
// Application.SetLook still applies display scale afterward.
func (a Appearance) Look() *Classic {
	a = a.Normalize()
	tok := tokensForAppearance(a)
	m := packMetrics(DensityDefault, tok, a.Corners, a.IconSize)
	return newClassic(string(tok.Family), tok.Palette, m, a.Corners, a.Icons, a.IconSize, tok).
		setPack(a.Name).setDensity(DensityDefault).setScale(1)
}

func tokensForAppearance(a Appearance) ThemeTokens {
	a = a.Normalize()
	if pack, ok := LoadTheme(a.Name); ok {
		tok := pack.Tokens
		if tok.Empty() {
			tok = ThemeTokens{Family: pack.Palette, Palette: paletteForFamily(pack.Palette)}
		}
		return withEraFonts(tok.Resolve(), pack.Name)
	}
	if pack, ok := LoadTheme(StarterName(a.Theme)); ok {
		return pack.Tokens.Resolve()
	}
	return ThemeTokens{Family: a.Theme, Palette: paletteForFamily(a.Theme)}.Resolve()
}

func paletteForFamily(t ThemeName) Palette {
	if ParseTheme(string(t)) == ThemeLight {
		return Light()
	}
	return Dark()
}

// PreferredLook loads XDG appearance prefs (or defaults) as a LookAndFeel.
func PreferredLook() LookAndFeel {
	return LoadAppearance().Look()
}

// LookAppearance reads pack name / palette / corners / icons / size from a live look.
func LookAppearance(l LookAndFeel) Appearance {
	a := DefaultAppearance()
	if l == nil {
		return a
	}
	if c, ok := l.(*Classic); ok {
		a.Theme = ParseTheme(c.Name())
		a.Corners = c.Corners()
		a.Icons = c.Icons()
		a.IconSize = c.IconSize()
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

// LookIconSize is the chrome icon size on a Classic look (medium otherwise).
func LookIconSize(l LookAndFeel) IconSize {
	if c, ok := l.(*Classic); ok {
		return c.IconSize()
	}
	return IconSizeMedium
}

// WithTheme rebuilds a Classic look with a new palette, keeping metrics,
// corners, and icon set (so density / scale survive a theme toggle).
func WithTheme(look LookAndFeel, theme ThemeName) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	theme = ParseTheme(string(theme))
	var tok ThemeTokens
	if pack, ok := LoadTheme(StarterName(theme)); ok {
		tok = pack.Tokens.Resolve()
	} else {
		tok = ThemeTokens{Family: theme, Palette: paletteForFamily(theme)}.Resolve()
	}
	m := lookMetrics(c.Density(), c.Scale(), tok, c.Corners(), c.IconSize())
	return newClassic(string(theme), tok.Palette, m, c.Corners(), c.Icons(), c.IconSize(), tok).
		setPack(StarterName(theme)).setDensity(c.Density()).setScale(c.Scale())
}

// WithCorners rebuilds a Classic look with a new radius policy.
func WithCorners(look LookAndFeel, corners CornerStyle) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	corners = ParseCorners(string(corners))
	tok := c.Tokens()
	m := lookMetrics(c.Density(), c.Scale(), tok, corners, c.IconSize())
	return newClassic(c.Name(), c.Palette(), m, corners, c.Icons(), c.IconSize(), tok).
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale())
}

// WithIcons rebuilds a Classic look with a new ToolIcon set (file or
// drawn). The theme pack name is kept; icons are independent of the pack.
func WithIcons(look LookAndFeel, icons IconSetName) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	icons = ParseIconSet(string(icons))
	return newClassic(c.Name(), c.Palette(), c.Metrics(), c.Corners(), icons, c.IconSize(), c.Tokens()).
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale())
}

// WithIconSize rebuilds a Classic look with a new ToolIcon draw size.
func WithIconSize(look LookAndFeel, sz IconSize) LookAndFeel {
	c, ok := classicOf(look)
	if !ok {
		return look
	}
	sz = ParseIconSize(string(sz))
	m := lookMetrics(c.Density(), c.Scale(), c.Tokens(), c.Corners(), sz)
	return newClassic(c.Name(), c.Palette(), m, c.Corners(), c.Icons(), sz, c.Tokens()).
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale())
}

// WithAppearance applies the appearance prefs to an existing Classic look,
// keeping its density and display scale.
//
// Metrics are rebuilt (defaults → density → pack → corners → icon size →
// scale) rather than overlaid on the live look: a pack that leaves ControlH
// or ComboH unset must fall back to the density default, not to whatever the
// previously selected pack asked for.
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
	tok := tokensForAppearance(a)
	if _, ok := LoadTheme(pack); !ok {
		pack = StarterName(a.Theme)
	}
	m := lookMetrics(c.Density(), c.Scale(), tok, a.Corners, a.IconSize)
	return newClassic(string(tok.Family), tok.Palette, m, a.Corners, a.Icons, a.IconSize, tok).
		setPack(pack).setDensity(c.Density()).setScale(c.Scale())
}

func classicOf(look LookAndFeel) (*Classic, bool) {
	if look == nil {
		return nil, false
	}
	c, ok := look.(*Classic)
	return c, ok
}
