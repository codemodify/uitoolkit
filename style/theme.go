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

// DefaultThemeName is the embedded dark / round / classic starter.
const DefaultThemeName = "dark-round-classic"

// ThemeSource says whether a pack is compiled in or loaded from disk.
type ThemeSource string

const (
	// ThemeSourceBuiltin is an embedded starter (not written to disk).
	ThemeSourceBuiltin ThemeSource = "builtin"
	// ThemeSourceUser is ~/.config/uitoolkit/themes/<name>/theme.json.
	ThemeSourceUser ThemeSource = "user"
)

// ThemePack is a named installable look (GTK/KDE-like). Corners and icons
// live inside the pack; look.json stores only the pack name.
type ThemePack struct {
	Name    string
	Label   string
	Source  ThemeSource
	Palette ThemeName
	Corners CornerStyle
	Icons   IconSetName
}

type themeFileJSON struct {
	Label   string `json:"label,omitempty"`
	Palette string `json:"palette,omitempty"`
	Theme   string `json:"theme,omitempty"` // alias for palette
	Corners string `json:"corners,omitempty"`
	Icons   string `json:"icons,omitempty"`
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

// StarterName is the embedded pack id for a palette × corners × icons combo.
func StarterName(theme ThemeName, corners CornerStyle, icons IconSetName) string {
	return string(ParseTheme(string(theme))) + "-" + string(ParseCorners(string(corners))) + "-" + string(ParseIconSet(string(icons)))
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

// Display is the Settings picker label (exported packs first by convention).
func (p ThemePack) Display() string {
	label := p.Label
	if label == "" {
		label = p.Name
	}
	switch p.Source {
	case ThemeSourceUser:
		return label + "  · exported"
	default:
		return label + "  · builtin"
	}
}

// Appearance resolves the pack into the live LookAndFeel knobs.
func (p ThemePack) Appearance() Appearance {
	return Appearance{
		Name:    p.Name,
		Theme:   ParseTheme(string(p.Palette)),
		Corners: ParseCorners(string(p.Corners)),
		Icons:   ParseIconSet(string(p.Icons)),
	}.Normalize()
}

// Look builds a Classic look from the pack (1× metrics).
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
		Corners: ParseCorners(doc.Corners),
		Icons:   ParseIconSet(doc.Icons),
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
	var names []string
	for _, pal := range []ThemeName{ThemeDark, ThemeLight} {
		for _, cor := range []CornerStyle{CornersRound, CornersSquare} {
			for _, ic := range []IconSetName{IconSetClassic, IconSetSharp} {
				names = append(names, StarterName(pal, cor, ic))
			}
		}
	}
	return names
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

func listUserThemes() map[string]ThemePack {
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

// ListThemes returns user packs (sorted by name) then embedded starters
// whose names are not shadowed. A user pack with the same name as a
// builtin wins in LoadTheme and is listed once (as exported).
func ListThemes() []ThemePack {
	users := listUserThemes()
	var userNames []string
	for n := range users {
		userNames = append(userNames, n)
	}
	sort.Strings(userNames)
	out := make([]ThemePack, 0, len(users)+8)
	for _, n := range userNames {
		out = append(out, users[n])
	}
	embedded := loadEmbedded()
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

// LoadTheme prefers a user pack at themes/<name>/theme.json, then an
// embedded starter of the same name.
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
	return ThemePack{}, false
}

// ExportTheme writes the current look as themes/<name>/theme.json so the
// user can tweak the pack. The directory name is the pack id.
func ExportTheme(name string, look LookAndFeel) (ThemePack, error) {
	return ExportAppearance(name, LookAppearance(look))
}

// ExportAppearance writes a user theme pack for the resolved appearance.
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
		Corners: a.Corners,
		Icons:   a.Icons,
	}
	doc := themeFileJSON{
		Label:   pack.Label,
		Palette: string(pack.Palette),
		Corners: string(pack.Corners),
		Icons:   string(pack.Icons),
	}
	if err := writeJSONFile(ThemeFile(clean), doc); err != nil {
		return ThemePack{}, err
	}
	return pack, nil
}
