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

// IconSetName selects chrome ToolIcons, and it now names one of three
// kinds of thing:
//
//   - classic / sharp, drawn in process;
//   - a sanitized name, which is a folder of PNGs under
//     ~/.config/uitoolkit/icons/<name>/ (lucide, phosphor, tabler, …);
//   - "desktop:<Theme>", an installed freedesktop icon theme read from
//     /usr/share/icons or ~/.local/share/icons (see [SystemIconPrefix]).
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
//
// Name may be a pack this build cannot load, because theme engines are
// opt-in ([docs/engines.md]) and a look.json is shared with every other
// uitoolkit application on the machine. The name is kept so the
// preference survives a build that leaves the engine out;
// [Appearance.Missing] reports it, and the look is the build's default
// one.
type Appearance struct {
	Name     string // pack id: "dark", "light", or a user export
	Theme    ThemeName
	Corners  CornerStyle
	Icons    IconSetName
	IconSize IconSize
	// FontUI and FontMono are the typefaces the user chose for the two
	// font roles — the interface face and the monospaced one — or ""
	// for "whatever the pack asks for", which is the default.
	//
	// A choice here beats the pack. A pack's [FontPrefs] is a list of
	// wishes an era had, most wanted first, and the toolkit walks it
	// until it finds something installed; a name in this field is not a
	// wish, it is an instruction, so it goes in front of that list. Aqua
	// asks for Lucida Grande and this field says Cantarell: the window
	// reads in Cantarell. It is the same rule [CornerStyle] already
	// follows — the pack's own shape until the user says round or square
	// — and the way back is the same too: clear the field (the
	// chooser's "Theme font") and the pack has its era again.
	//
	// The pack's list is still the fallback, not a casualty: a family
	// that is not installed — a look.json carried to another machine,
	// a font uninstalled since — falls through to the era's own names
	// and then to the bundled faces, so a window is never left without
	// a typeface. See [withUserFonts].
	FontUI, FontMono string
	// ReduceMotion turns animations off (see [Animations]).
	ReduceMotion bool
	// FollowDesktop shows the pack's light or dark sibling to match the
	// desktop's preference (see [Appearance.Effective]); Name stays the
	// user's choice.
	FollowDesktop bool
	// NativeDialogs shows the desktop's own file dialogs (the XDG portal's)
	// instead of the toolkit's themed ones.
	NativeDialogs bool
	// ComboWheel lets the mouse wheel over a closed combo box step its
	// selection (see [ComboWheel]). It is off by default, and it is the
	// one preference in this struct that changes what an input device
	// does rather than what a window looks like.
	ComboWheel bool
	// Decorations is who draws the frame of a window: DecorationsAuto (the
	// toolkit for windows with their own title bar), DecorationsSystem (the
	// desktop's title bar and borders wherever it has them — Settings' "Use
	// system title bar and borders") or DecorationsToolkit (the toolkit for
	// every window).
	Decorations DecorationsPref
	// CaptionButtons is where a frame uitoolkit draws puts its caption
	// buttons: CaptionButtonsDesktop (the desktop's button layout, the
	// default) or CaptionButtonsTheme (the look's own: the Mac's traffic
	// lights on the left, GNOME's lone close, KDE's window menu).
	CaptionButtons CaptionButtonsPref
	// Renderer is the paint device a window's surface binds when it is
	// created: RendererAuto (the GPU where EGL starts, else the CPU),
	// RendererGPU or RendererCPU. UITK_PAINT overrides it wherever it is
	// set — the file is never rewritten to match — and a window already
	// open keeps the device it was created with. See [RendererPref].
	Renderer RendererPref
}

// CaptionButtonsPref is the look.json "captionButtons" preference.
type CaptionButtonsPref string

const (
	// CaptionButtonsDesktop is the default (the zero value; written
	// "desktop" or left out of look.json).
	CaptionButtonsDesktop CaptionButtonsPref = ""
	CaptionButtonsTheme   CaptionButtonsPref = "theme"
)

// ParseCaptionButtonsPref accepts desktop / theme (and look); anything else
// is the desktop's layout.
func ParseCaptionButtonsPref(s string) CaptionButtonsPref {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "theme", "look":
		return CaptionButtonsTheme
	}
	return CaptionButtonsDesktop
}

// DecorationsPref is the look.json "decorations" preference.
type DecorationsPref string

const (
	// DecorationsAuto is the default (the zero value; written "auto" or
	// left out of look.json).
	DecorationsAuto    DecorationsPref = ""
	DecorationsSystem  DecorationsPref = "system"
	DecorationsToolkit DecorationsPref = "toolkit"
)

// ParseDecorationsPref accepts auto / system / toolkit (and the platform's
// server / client); anything else is auto.
func ParseDecorationsPref(s string) DecorationsPref {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "system", "server", "native":
		return DecorationsSystem
	case "toolkit", "client", "custom":
		return DecorationsToolkit
	}
	return DecorationsAuto
}

// RendererPref is the look.json "renderer" preference: which paint
// device the toolkit binds to a window's surface when it is created.
//
// It is not what the toolkit looks like, and it is in this file anyway,
// beside "decorations" and "nativeDialogs" — the two other preferences
// that say *who does the work* rather than what the result looks like
// (the desktop's frame or the toolkit's, the desktop's file dialogs or
// the toolkit's). look.json is the toolkit's one preferences file: one
// atomic write from Settings, one read at start-up in every app, one
// watcher. A file of its own would have been a second of each for a
// single enum. See docs/settings.md, "Where the renderer preference
// lives".
type RendererPref string

const (
	// RendererAuto takes an EGL/GLES device when it initialises and the
	// CPU rasterizer when it does not. It is the default (the zero
	// value; written "auto" or left out of look.json).
	RendererAuto RendererPref = ""
	// RendererGPU asks for the EGL/GLES device. A window whose EGL
	// cannot start still opens, on the CPU: a preference is not a
	// promise the driver has to keep.
	RendererGPU RendererPref = "gpu"
	// RendererCPU is the portable scanline rasterizer, presented through
	// wl_shm or XPutImage.
	RendererCPU RendererPref = "cpu"
)

// ParseRendererPref accepts auto / gpu / cpu and the words the same three
// go by elsewhere (egl, gles; software, raster); anything else is auto.
func ParseRendererPref(s string) RendererPref {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "gpu", "egl", "gles":
		return RendererGPU
	case "cpu", "software", "raster":
		return RendererCPU
	}
	return RendererAuto
}

// Label is the word the renderer chooser shows for p.
func (p RendererPref) Label() string {
	switch ParseRendererPref(string(p)) {
	case RendererGPU:
		return "GPU"
	case RendererCPU:
		return "CPU"
	default:
		return "Auto"
	}
}

// DefaultAppearance is the default theme ([DefaultThemeName], a light
// pack) in its own corners, with the classic icons.
func DefaultAppearance() Appearance {
	// The pack's own family, not a fixed light: with engines opt-in the
	// default pack may be whatever this build carries, and a light
	// appearance over a dark pack is a look nobody asked for.
	name := DefaultTheme()
	family := ThemeLight
	if p, ok := builtinEraPack(name); ok {
		family = p.Palette
	}
	return Appearance{
		Name:     name,
		Theme:    family,
		Corners:  CornersTheme,
		Icons:    IconSetClassic,
		IconSize: IconSizeMedium,
		// A fresh appearance follows the desktop, because a program
		// nobody has configured should look like it belongs to the
		// desktop it opened on. Without this, a first run on a dark
		// desktop was a white window — and for a passphrase prompt that
		// flares up at the moment it matters, that is the worst time for
		// it.
		//
		// It is only the *default*. A look.json is a choice, and
		// resolveAppearance takes followDesktop from the file, so a user
		// who turned it off keeps it off.
		FollowDesktop: true,
	}
}

// FontEnv and MonoFontEnv override the saved typefaces for one process,
// the way [ThemeEnv] overrides the saved pack: UITK_FONT="Liberation
// Sans" ./app. The word "theme" (and the empty string) means "the pack's
// own", so a session can be run at a pack's era typography without
// editing look.json.
//
// Like UITK_THEME and unlike UITK_PAINT, an override read here is the
// loaded preference: it is not written back on its own, but a Settings
// started under it stages it and an Apply saves it. UITK_PAINT is the
// exception because it says which *device* paints rather than what is
// painted, and baking a debugging flag into everyone's preferences is a
// different kind of mistake from baking a typeface into them.
const (
	FontEnv     = "UITK_FONT"
	MonoFontEnv = "UITK_FONT_MONO"
)

// NormalizeFontChoice is a chosen family in the form the rest of the
// toolkit stores it: trimmed, with the words that mean "no choice"
// ("theme", "default", "pack") resolved to the empty string.
func NormalizeFontChoice(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "", "theme", "default", "pack", "theme font":
		return ""
	}
	return s
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
// heroicons / material-symbols, any sanitized file-set directory name,
// or "desktop:<Theme>" for an installed freedesktop icon theme
// (empty → classic).
func ParseIconSet(s string) IconSetName {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, SystemIconPrefix) {
		// A theme directory keeps its own spelling — Breeze_Light and
		// Papirus-Dark are the names on the disk — so it is checked
		// rather than sanitized. One that could escape the icon
		// directories is not a theme and reads as classic.
		if name := IconSetName(SystemIconPrefix + strings.TrimSpace(strings.TrimPrefix(s, SystemIconPrefix))); IsSystemIconSet(name) {
			return name
		}
		return IconSetClassic
	}
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
	a.FontUI = NormalizeFontChoice(a.FontUI)
	a.FontMono = NormalizeFontChoice(a.FontMono)
	a.Corners = ParseCorners(string(a.Corners))
	a.Icons = ParseIconSet(string(a.Icons))
	a.IconSize = ParseIconSize(string(a.IconSize))
	a.Decorations = ParseDecorationsPref(string(a.Decorations))
	a.CaptionButtons = ParseCaptionButtonsPref(string(a.CaptionButtons))
	a.Renderer = ParseRendererPref(string(a.Renderer))
	if strings.TrimSpace(a.Name) == "" {
		a.Name = StarterName(a.Theme)
	}
	return a
}

func (a Appearance) String() string {
	a = a.Normalize()
	return a.Name
}

// Missing reports that [Appearance.Name] names a pack this build cannot
// load, so the window wears the build's default look instead of the one
// the preference asks for.
//
// The usual cause is that theme engines are chosen at build time and
// this build left that one out: a look.json is shared with every other
// uitoolkit application on the machine, so a pack somebody else's build
// paints is a name this one has never heard of. The other cause is a
// name that was always wrong, and the two cannot be told apart from the
// name alone — what is missing is the engine's `init`, and an engine
// that is not compiled in leaves nothing behind to be asked about. See
// [EngineIDs] for what this build does have.
//
// It is answered rather than stored, so it cannot go stale: an
// application that sets Name and asks again gets the truth about the new
// name. [MissingThemeNote] is the line to log.
func (a Appearance) Missing() bool {
	name := strings.TrimSpace(a.Name)
	return name != "" && !ThemeAvailable(name)
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
	a = a.Effective()
	tok := tokensForAppearance(a)
	m := packMetrics(DensityDefault, tok, a.Corners, a.IconSize)
	return newClassic(string(tok.Family), tok.Palette, m, a.Corners, a.Icons, a.IconSize, tok).
		setPack(a.Name).setDensity(DensityDefault).setScale(1).
		setAppearanceBits(a.FontUI, a.FontMono, a.FollowDesktop)
}

func tokensForAppearance(a Appearance) ThemeTokens {
	a = a.Normalize()
	if pack, ok := LoadTheme(a.Name); ok {
		tok := pack.Tokens
		if tok.Empty() {
			tok = ThemeTokens{Family: pack.Palette, Palette: paletteForFamily(pack.Palette)}
		}
		tok = tok.Resolve()
		if a.FollowDesktop {
			tok = withDesktopAccent(tok)
		}
		return withUserFonts(withEraFonts(tok, pack.Name), a.FontUI, a.FontMono)
	}
	// The pack is not in this build ([Appearance.Missing]). Show the one
	// this build would have shown if look.json had said nothing, not the
	// bare light or dark starter: a build whose default is Plastik
	// should look like Plastik when somebody else's theme name reaches
	// it, and the starter is a palette with no era at all.
	//
	// Only where the families agree, which after resolveAppearance they
	// do — it no longer guesses one from the name. An application that
	// set Theme itself means it, and gets that family's starter.
	if def, ok := LoadTheme(DefaultTheme()); ok && def.Palette == a.Theme {
		return withUserFonts(withEraFonts(def.Tokens.Resolve(), def.Name), a.FontUI, a.FontMono)
	}
	if pack, ok := LoadTheme(StarterName(a.Theme)); ok {
		return withUserFonts(pack.Tokens.Resolve(), a.FontUI, a.FontMono)
	}
	return withUserFonts(ThemeTokens{Family: a.Theme, Palette: paletteForFamily(a.Theme)}.Resolve(), a.FontUI, a.FontMono)
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

// LookAppearance is the appearance a look was built from, as far as the
// look itself can say: the pack, the palette, the corners, the icons and
// their size, the two typefaces the user pinned (empty where the pack's
// own era was used) and whether it follows the desktop's light / dark
// mode and accent.
//
// The last three cannot be read out of a finished look — it carries the
// family it *resolved* to, not whether that family was an instruction —
// so a look built through [Appearance.Look] or [WithAppearance] carries
// them explicitly and every derivation passes them on. A look built by
// hand ([NewClassic]) never pinned anything and answers empty, which is
// the truth about it. This is what makes a derived look
// ([Themed], the theme cascade) keep a pinned typeface instead of
// silently falling back to the pack's era.
//
// The preferences that belong to an application rather than to a look —
// who draws the frame, where the caption buttons go, reduced motion, the
// desktop's file dialogs — still come back at their defaults, because a
// look has nothing to do with them. app.Application.Appearance is the
// whole of it, and what an app that switches packs should start from.
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
		a.FontUI, a.FontMono = c.userUI, c.userMono
		a.FollowDesktop = c.follow
		return a.Normalize()
	}
	a.FontUI, a.FontMono = "", ""
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
		setPack(StarterName(theme)).setDensity(c.Density()).setScale(c.Scale()).carry(c)
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
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale()).carry(c)
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
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale()).carry(c)
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
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale()).carry(c)
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
	a = a.Effective()
	pack := a.Name
	tok := tokensForAppearance(a)
	if _, ok := LoadTheme(pack); !ok {
		pack = StarterName(a.Theme)
	}
	m := lookMetrics(c.Density(), c.Scale(), tok, a.Corners, a.IconSize)
	return newClassic(string(tok.Family), tok.Palette, m, a.Corners, a.Icons, a.IconSize, tok).
		setPack(pack).setDensity(c.Density()).setScale(c.Scale()).
		setAppearanceBits(a.FontUI, a.FontMono, a.FollowDesktop)
}

func classicOf(look LookAndFeel) (*Classic, bool) {
	if look == nil {
		return nil, false
	}
	c, ok := look.(*Classic)
	return c, ok
}
