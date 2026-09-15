package style

import "strings"

// FontPrefs lists the typefaces a pack reads in, most wanted first: the
// era's own face, then open look-alikes. The first one installed wins (see
// [ResolveFont]); none installed, the bundled Titillium Web / JetBrains Mono
// stand in. theme.json: "fonts": {"ui": [...], "mono": [...]}.
type FontPrefs struct {
	UI   []string `json:"ui,omitempty"`
	Mono []string `json:"mono,omitempty"`
}

// Empty reports that the prefs name no typeface.
func (f FontPrefs) Empty() bool { return len(f.UI) == 0 && len(f.Mono) == 0 }

// The typefaces of each era, as the platforms shipped them: a UI face, then
// look-alikes that are commonly installed (metric twins where they exist:
// Nimbus Sans and TeX Gyre Heros for Helvetica, Liberation for Arial and
// Courier New, Selawik for Segoe UI).
var (
	helvetica = []string{"Helvetica", "Nimbus Sans", "Nimbus Sans L", "TeX Gyre Heros", "Liberation Sans", "Arimo", "FreeSans"}
	courier   = []string{"Courier", "Nimbus Mono PS", "Nimbus Mono L", "Liberation Mono", "Cousine", "Courier New"}
	msSans    = []string{"MS Sans Serif", "Microsoft Sans Serif", "Tahoma", "Arial", "Liberation Sans", "Arimo"}
	tahoma    = []string{"Tahoma", "Verdana", "DejaVu Sans", "Liberation Sans"}
	courierNw = []string{"Courier New", "Liberation Mono", "Cousine"}
	segoe     = []string{"Segoe UI", "Selawik", "Open Sans", "Noto Sans", "Liberation Sans"}
	consolas  = []string{"Consolas", "Cascadia Mono", "Liberation Mono", "DejaVu Sans Mono"}
	lucida    = []string{"Lucida Grande", "Lucida Sans Unicode", "Lucida Sans", "DejaVu Sans", "Noto Sans"}
	monaco    = []string{"Monaco", "Menlo", "DejaVu Sans Mono"}
	vera      = []string{"DejaVu Sans", "Bitstream Vera Sans", "Noto Sans"}
	veraMono  = []string{"DejaVu Sans Mono", "Bitstream Vera Sans Mono", "Liberation Mono"}
	notoSans  = []string{"Noto Sans", "Oxygen", "DejaVu Sans"}
	hack      = []string{"Hack", "Noto Sans Mono", "DejaVu Sans Mono"}
	cantarell = []string{"Cantarell", "Adwaita Sans", "Noto Sans", "DejaVu Sans"}
)

// eraEngineFonts are the typefaces of each engine's era.
var eraEngineFonts = map[string]FontPrefs{
	// The first Mac through System 7: Chicago 12.
	"system7": {UI: []string{"Chicago", "ChicagoFLF", "Charcoal", "Geneva"}, Mono: monaco},
	// Windows 3.1 dialogs: MS Sans Serif (the bold System font elsewhere).
	"win31": {UI: msSans, Mono: append([]string{"Fixedsys", "Fixedsys Excelsior"}, courierNw...)},
	// Sun's OpenWindows set OPEN LOOK in Lucida.
	"openlook": {UI: []string{"Lucida Sans", "Lucida Sans Unicode", "Lucida Grande", "DejaVu Sans"}, Mono: []string{"Lucida Sans Typewriter", "Lucida Console", "DejaVu Sans Mono"}},
	// Workbench: Topaz 8, a monospaced bitmap face; monospaced faces stand in.
	"amiga": {UI: []string{"Topaz", "Topaz New", "Amiga Topaz", "DejaVu Sans Mono", "Liberation Mono", "Noto Sans Mono"}, Mono: []string{"Topaz", "Topaz New", "DejaVu Sans Mono", "Liberation Mono"}},
	// BeOS: Swiss 721, a Helvetica design.
	"beos": {UI: append([]string{"Swis721 BT", "Swiss 721"}, helvetica...), Mono: courier},
	// OS/2 Warp 4: WarpSans (Workplace Sans is an open look-alike).
	"os2": {UI: append([]string{"WarpSans", "Workplace Sans"}, helvetica...), Mono: courier},
	// NeXTSTEP, OPENSTEP and Window Maker set their UI in Helvetica.
	"next": {UI: helvetica, Mono: courier},
	// Motif, HP VUE, CDE and IRIX read in X11's Helvetica.
	"motif": {UI: helvetica, Mono: courier},
	// Windows 95 and 98: MS Sans Serif 8pt.
	"win95": {UI: msSans, Mono: courierNw},
	// Mac OS 8 and 9: Charcoal (Chicago before it).
	"platinum": {UI: []string{"Charcoal", "Chicago", "ChicagoFLF", "Geneva"}, Mono: monaco},
	// Mac OS X through 10.9: Lucida Grande 13pt.
	"aqua": {UI: lucida, Mono: monaco},
	// Windows XP: Tahoma 8pt.
	"luna": {UI: tahoma, Mono: courierNw},
	// Red Hat Linux 8 and 9: Luxi Sans.
	"bluecurve": {UI: []string{"Luxi Sans", "Bitstream Vera Sans", "DejaVu Sans", "Nimbus Sans"}, Mono: []string{"Luxi Mono", "Bitstream Vera Sans Mono", "DejaVu Sans Mono"}},
	// GNOME 2 and the Qt look after it: Bitstream Vera, then DejaVu.
	"clearlooks": {UI: vera, Mono: veraMono},
	// KDE 3 and Qt 4's Plastique: "Sans Serif", Bitstream Vera then DejaVu.
	"keramik": {UI: vera, Mono: veraMono},
	"plastik": {UI: vera, Mono: veraMono},
	// Windows Vista and 7: Segoe UI 9pt.
	"aero": {UI: segoe, Mono: consolas},
	// KDE 4: DejaVu Sans, later the Oxygen font.
	"oxygen": {UI: []string{"DejaVu Sans", "Oxygen", "Noto Sans"}, Mono: []string{"Oxygen Mono", "DejaVu Sans Mono", "Noto Sans Mono"}},
	// Qt 5's Fusion and KDE Plasma 5: Noto Sans (the Oxygen font before 5.5).
	"fusion": {UI: notoSans, Mono: hack},
	"breeze": {UI: notoSans, Mono: hack},
	// Windows 8 and 10: Segoe UI.
	"metro": {UI: segoe, Mono: consolas},
	// GNOME 3 and libadwaita: Cantarell.
	"adwaita": {UI: cantarell, Mono: []string{"Source Code Pro", "Adwaita Mono", "DejaVu Sans Mono", "Noto Sans Mono"}},
	// Windows 11: Segoe UI Variable.
	"fluent": {UI: append([]string{"Segoe UI Variable Text", "Segoe UI Variable"}, segoe...), Mono: append([]string{"Cascadia Mono"}, consolas...)},
	// macOS 10.11 onwards: San Francisco (Inter is the closest open face).
	"macos": {UI: []string{"SF Pro Text", "SF Pro", "SF NS", ".SF NS Text", "Inter", "Noto Sans"}, Mono: []string{"SF Mono", "Menlo", "DejaVu Sans Mono", "Noto Sans Mono"}},
	// Material Design: Roboto.
	"material": {UI: []string{"Roboto", "Noto Sans", "Open Sans"}, Mono: []string{"Roboto Mono", "Noto Sans Mono", "DejaVu Sans Mono"}},
	// FlatLaf takes the platform's UI font: Segoe UI in its reference
	// screenshots, Inter when an app bundles FlatLaf's; IntelliJ's mono.
	"flatlaf": {UI: append([]string{"Segoe UI", "Inter"}, segoe[1:]...), Mono: append([]string{"JetBrains Mono"}, consolas...)},
}

// eraPackFonts override the engine's typefaces for one pack.
var eraPackFonts = map[string]FontPrefs{
	// Windows 2000 moved dialogs to Tahoma.
	"win2000": {UI: tahoma, Mono: courierNw},
	// GNOME 3.14 kept DejaVu Sans Mono.
	"adwaita-gtk3": {UI: cantarell, Mono: veraMono},
	// OS X 10.10 Yosemite: Helvetica Neue, the year before San Francisco.
	"yosemite": {UI: append([]string{"Helvetica Neue"}, helvetica...), Mono: []string{"Menlo", "DejaVu Sans Mono", "Noto Sans Mono"}},
}

// withEraFonts fills the typefaces tok leaves unset from its pack's era:
// the pack's own entry, then its engine's.
func withEraFonts(tok ThemeTokens, pack string) ThemeTokens {
	era, ok := eraPackFonts[strings.ToLower(strings.TrimSpace(pack))]
	if !ok {
		era = eraEngineFonts[tok.Engine]
	}
	if len(tok.Fonts.UI) == 0 {
		tok.Fonts.UI = era.UI
	}
	if len(tok.Fonts.Mono) == 0 {
		tok.Fonts.Mono = era.Mono
	}
	return tok
}

// lookFamilies resolves tok's typefaces to installed or bundled families.
func lookFamilies(tok ThemeTokens) (ui, mono string) {
	if ui = ResolveFont(tok.Fonts.UI); ui == "" {
		ui = FamilyUI
	}
	if mono = ResolveFont(tok.Fonts.Mono); mono == "" {
		mono = FamilyMono
	}
	return ui, mono
}
