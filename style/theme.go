package style

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

//go:embed themes/*/theme.json
var starterFS embed.FS

// DefaultThemeName is the embedded dark palette starter.
const DefaultThemeName = "dark"

// ThemeSource says whether a pack is compiled in or loaded from disk.
type ThemeSource string

const (
	// ThemeSourceBuiltin is an embedded starter (not written to disk).
	ThemeSourceBuiltin ThemeSource = "builtin"
	// ThemeSourceUser is ~/.config/uitoolkit/themes/<name>/theme.json.
	ThemeSourceUser ThemeSource = "user"
)

// ThemePack is a named color theme (palette only). Corners and icons
// are independent look.json prefs, not part of the pack identity.
type ThemePack struct {
	Name    string
	Label   string
	Source  ThemeSource
	Palette ThemeName
}

type themeFileJSON struct {
	Label   string `json:"label,omitempty"`
	Palette string `json:"palette,omitempty"`
	Theme   string `json:"theme,omitempty"`   // alias for palette
	Corners string `json:"corners,omitempty"` // ignored; look.json owns corners
	Icons   string `json:"icons,omitempty"`   // ignored; look.json owns icons
}

var (
	themeNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

	embeddedOnce sync.Once
	embeddedPack map[string]ThemePack
)

// ThemesDir is $XDG_CONFIG_HOME/uitoolkit/themes
// (or ~/.config/uitoolkit/themes).
func ThemesDir() string {
	return filepath.Join(ConfigDir(), "themes")
}

// ThemeFile is themes/<name>/theme.json.
func ThemeFile(name string) string {
	return filepath.Join(ThemesDir(), name, "theme.json")
}

// StarterName is the embedded pack id for a palette (dark or light).
func StarterName(theme ThemeName) string {
	return string(ParseTheme(string(theme)))
}

// SplitLookThemeName reads a look.json "theme" value. Compound v0.11–v0.12.1
// ids (dark-round-classic, light-square-sharp, dark-round, …) become the
// palette name plus corners from the name. Other names are user packs.
func SplitLookThemeName(name string) (pack string, corners CornerStyle, hasCorners bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return DefaultThemeName, CornersRound, false
	}
	parts := strings.Split(name, "-")
	if len(parts) >= 1 && (parts[0] == "dark" || parts[0] == "light") {
		if len(parts) >= 2 && (parts[1] == "round" || parts[1] == "square") {
			return parts[0], ParseCorners(parts[1]), true
		}
		if len(parts) == 1 {
			return parts[0], CornersRound, false
		}
	}
	return name, CornersRound, false
}

// CanonicalStarterName maps a look.json / pack id onto the palette
// starter (dark / light). Trailing -round/-square/-classic/-sharp are
// dropped for the eight legacy compound names. Other names are unchanged.
func CanonicalStarterName(name string) string {
	pack, _, _ := SplitLookThemeName(name)
	if pack == "" {
		return DefaultThemeName
	}
	return pack
}

// SanitizeThemeName lowercases and accepts [a-z][a-z0-9_-]{0,63}.
func SanitizeThemeName(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	if s == "" || s == "." || s == ".." {
		return "", fmt.Errorf("theme name is required")
	}
	if strings.ContainsAny(s, `/\:`) {
		return "", fmt.Errorf("theme name cannot contain path separators")
	}
	if !themeNameRe.MatchString(s) {
		return "", fmt.Errorf("theme name must start with a letter and use only lowercase letters, digits, hyphen, or underscore")
	}
	return s, nil
}

// Display is the Settings picker label (short palette name).
func (p ThemePack) Display() string {
	if strings.TrimSpace(p.Label) != "" {
		return p.Label
	}
	return p.Name
}

// Appearance resolves the pack into palette knobs. Corners and icons
// stay at defaults; callers overlay look.json prefs.
func (p ThemePack) Appearance() Appearance {
	return Appearance{
		Name:    p.Name,
		Theme:   ParseTheme(string(p.Palette)),
		Corners: CornersRound,
		Icons:   IconSetClassic,
	}.Normalize()
}

// Look builds a Classic look from the pack palette (1× metrics, round).
func (p ThemePack) Look() *Classic {
	return p.Appearance().Look()
}

func parseThemeFile(name string, raw []byte, src ThemeSource) (ThemePack, error) {
	var doc themeFileJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ThemePack{}, err
	}
	pal := doc.Palette
	if pal == "" {
		pal = doc.Theme
	}
	label := strings.TrimSpace(doc.Label)
	if label == "" {
		label = name
	}
	return ThemePack{
		Name:    name,
		Label:   label,
		Source:  src,
		Palette: ParseTheme(pal),
	}, nil
}

func loadEmbedded() map[string]ThemePack {
	embeddedOnce.Do(func() {
		embeddedPack = map[string]ThemePack{}
		_ = fs.WalkDir(starterFS, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || path.Base(p) != "theme.json" {
				return err
			}
			name := path.Base(path.Dir(p))
			b, err := starterFS.ReadFile(p)
			if err != nil {
				return nil
			}
			pack, err := parseThemeFile(name, b, ThemeSourceBuiltin)
			if err != nil {
				return nil
			}
			embeddedPack[name] = pack
			return nil
		})
	})
	return embeddedPack
}

func starterOrder() []string {
	return []string{StarterName(ThemeDark), StarterName(ThemeLight)}
}

func readUserTheme(name string) (ThemePack, bool) {
	b, err := os.ReadFile(ThemeFile(name))
	if err != nil {
		return ThemePack{}, false
	}
	pack, err := parseThemeFile(name, b, ThemeSourceUser)
	if err != nil {
		return ThemePack{}, false
	}
	return pack, true
}

func listUserThemeMap() map[string]ThemePack {
	out := map[string]ThemePack{}
	entries, err := os.ReadDir(ThemesDir())
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, err := SanitizeThemeName(name); err != nil {
			continue
		}
		if pack, ok := readUserTheme(name); ok {
			out[name] = pack
		}
	}
	return out
}

// ListUserThemes returns exported color themes, sorted by name.
func ListUserThemes() []ThemePack {
	users := listUserThemeMap()
	var names []string
	for n := range users {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]ThemePack, 0, len(names))
	for _, n := range names {
		out = append(out, users[n])
	}
	return out
}

// ListBuiltinThemes returns embedded dark / light when not shadowed.
func ListBuiltinThemes() []ThemePack {
	users := listUserThemeMap()
	embedded := loadEmbedded()
	out := make([]ThemePack, 0, 2)
	for _, n := range starterOrder() {
		if _, shadowed := users[n]; shadowed {
			continue
		}
		if p, ok := embedded[n]; ok {
			out = append(out, p)
		}
	}
	return out
}

// ListThemes returns embedded starters (not shadowed) then user packs.
func ListThemes() []ThemePack {
	builtins := ListBuiltinThemes()
	users := ListUserThemes()
	out := make([]ThemePack, 0, len(builtins)+len(users))
	out = append(out, builtins...)
	out = append(out, users...)
	return out
}

// LoadTheme prefers a user pack at themes/<name>/theme.json, then an
// embedded starter. Legacy compound ids map to dark / light.
func LoadTheme(name string) (ThemePack, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ThemePack{}, false
	}
	if pack, ok := readUserTheme(name); ok {
		return pack, true
	}
	if pack, ok := loadEmbedded()[name]; ok {
		return pack, true
	}
	packName, _, _ := SplitLookThemeName(name)
	if packName != name {
		if pack, ok := readUserTheme(packName); ok {
			return pack, true
		}
		if pack, ok := loadEmbedded()[packName]; ok {
			return pack, true
		}
	}
	return ThemePack{}, false
}

// ExportTheme writes the current look's palette as themes/<name>/theme.json.
// Corners and icons stay look.json prefs; they are not stored in the pack.
func ExportTheme(name string, look LookAndFeel) (ThemePack, error) {
	return ExportAppearance(name, LookAppearance(look))
}

// ExportAppearance writes a user color theme (palette only).
func ExportAppearance(name string, a Appearance) (ThemePack, error) {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return ThemePack{}, err
	}
	a = a.Normalize()
	pack := ThemePack{
		Name:    clean,
		Label:   clean,
		Source:  ThemeSourceUser,
		Palette: a.Theme,
	}
	doc := themeFileJSON{
		Label:   pack.Label,
		Palette: string(pack.Palette),
	}
	if err := writeJSONFile(ThemeFile(clean), doc); err != nil {
		return ThemePack{}, err
	}
	return pack, nil
}
