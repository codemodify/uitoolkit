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

// ThemePack is a named era skin (tokens + family). Corners and icons
// are independent look.json prefs, not part of the pack identity.
type ThemePack struct {
	Name    string
	Label   string
	Source  ThemeSource
	Palette ThemeName // dark / light family (Mail View + legacy)
	Era     string
	Tokens  ThemeTokens
}

type themeFileJSON struct {
	Label     string             `json:"label,omitempty"`
	Palette   string             `json:"palette,omitempty"` // family: dark / light
	Theme     string             `json:"theme,omitempty"`   // alias for palette
	Family    string             `json:"family,omitempty"`
	Era       string             `json:"era,omitempty"`
	Bevel     string             `json:"bevel,omitempty"`
	Elevation int                `json:"elevation,omitempty"`
	Metrics   *chromeMetricsJSON `json:"metrics,omitempty"`
	Colors    map[string]string  `json:"colors,omitempty"`
	Corners   string             `json:"corners,omitempty"` // ignored; look.json owns corners
	Icons     string             `json:"icons,omitempty"`   // ignored; look.json owns icons
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
	fam := p.Palette
	if p.Tokens.Family != "" {
		fam = p.Tokens.Family
	}
	return Appearance{
		Name:    p.Name,
		Theme:   ParseTheme(string(fam)),
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
		pal = doc.Family
	}
	if pal == "" {
		pal = doc.Theme
	}
	label := strings.TrimSpace(doc.Label)
	if label == "" {
		label = name
	}
	era := strings.TrimSpace(doc.Era)
	tok := tokensFromJSON(doc)
	if base, ok := builtinEraPack(name); ok {
		if era == "" {
			era = base.Era
		}
		if label == name && base.Label != "" {
			label = base.Label
		}
		if tok.Empty() || (len(doc.Colors) == 0 && doc.Bevel == "") {
			tok = base.Tokens
		} else {
			tok = mergeTokens(base.Tokens, tok)
		}
		if pal == "" {
			pal = string(base.Palette)
		}
	}
	tok.Family = ParseTheme(pal)
	if tok.Era == "" {
		tok.Era = era
	}
	tok = tok.Resolve()
	return ThemePack{
		Name:    name,
		Label:   label,
		Source:  src,
		Palette: tok.Family,
		Era:     era,
		Tokens:  tok,
	}, nil
}

// themeDoc is the on-disk document for a resolved token set. Reading it back
// through parseThemeFile reproduces the same tokens at 8-bit colour
// precision, which is what keeps themes/*/theme.json in sync with the Go
// era packs (see TestShippedThemeJSONMatchesPacks).
func themeDoc(label, era string, tok ThemeTokens) themeFileJSON {
	fam := string(ParseTheme(string(tok.Family)))
	return themeFileJSON{
		Label:     label,
		Palette:   fam,
		Family:    fam,
		Era:       era,
		Bevel:     string(tok.Bevel),
		Elevation: tok.Metrics.Elevation,
		Metrics:   tok.Metrics.json(),
		Colors:    tokensToColorMap(tok),
	}
}

func mergeTokens(base, over ThemeTokens) ThemeTokens {
	out := base
	if over.Bevel != "" {
		out.Bevel = over.Bevel
	}
	if over.Family != "" {
		out.Family = over.Family
	}
	if over.Era != "" {
		out.Era = over.Era
	}
	out.Palette = overlayPalette(base.Palette, over.Palette)
	out.Metrics = mergeChromeMetrics(base.Metrics, over.Metrics)
	if !colorUnset(over.Hot.Fill) {
		out.Hot.Fill = over.Hot.Fill
	}
	if !colorUnset(over.Hot.Border) {
		out.Hot.Border = over.Hot.Border
	}
	if !colorUnset(over.Pressed.Fill) {
		out.Pressed.Fill = over.Pressed.Fill
	}
	if !colorUnset(over.Pressed.Border) {
		out.Pressed.Border = over.Pressed.Border
	}
	if !colorUnset(over.Selected.Fill) {
		out.Selected.Fill = over.Selected.Fill
	}
	if !colorUnset(over.Selected.Border) {
		out.Selected.Border = over.Selected.Border
	}
	if !colorUnset(over.Disabled.Fill) {
		out.Disabled.Fill = over.Disabled.Fill
	}
	if !colorUnset(over.Disabled.Border) {
		out.Disabled.Border = over.Disabled.Border
	}
	if !colorUnset(over.Focus.Fill) {
		out.Focus.Fill = over.Focus.Fill
	}
	if !colorUnset(over.Focus.Border) {
		out.Focus.Border = over.Focus.Border
	}
	return out
}

func mergeChromeMetrics(base, over ChromeMetrics) ChromeMetrics {
	out := base
	if over.Radius != 0 {
		out.Radius = over.Radius
	}
	if over.RadiusSmall != 0 {
		out.RadiusSmall = over.RadiusSmall
	}
	if over.BevelDepth != 0 {
		out.BevelDepth = over.BevelDepth
	}
	if over.GutterWidth != 0 {
		out.GutterWidth = over.GutterWidth
	}
	if over.Scroll != 0 {
		out.Scroll = over.Scroll
	}
	if over.ControlH != 0 {
		out.ControlH = over.ControlH
	}
	if over.FieldH != 0 {
		out.FieldH = over.FieldH
	}
	if over.ComboH != 0 {
		out.ComboH = over.ComboH
	}
	if over.Elevation != 0 {
		out.Elevation = over.Elevation
	}
	return out
}

func loadEmbedded() map[string]ThemePack {
	embeddedOnce.Do(func() {
		embeddedPack = map[string]ThemePack{}
		for name, p := range eraPackIndex() {
			p.Source = ThemeSourceBuiltin
			embeddedPack[name] = p
		}
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
			if _, exists := embeddedPack[name]; !exists {
				embeddedPack[name] = pack
			}
			return nil
		})
	})
	return embeddedPack
}

func starterOrder() []string {
	if names := builtinEraOrder(); len(names) > 0 {
		return names
	}
	return []string{StarterName(ThemeDark), StarterName(ThemeLight)}
}

// userThemeDirs maps a canonical pack id onto the directory that holds it.
//
// Listing and loading must agree: a folder named "MyTheme" lists (and is
// exported / deleted) as "mytheme", so the loader has to find it under the
// canonical id too. Only sanitized names are accepted, which is also what
// keeps "../../etc" out of [ThemeFile] — and os.ReadDir does not follow
// symlinks, so a symlinked folder is not a directory entry here.
func userThemeDirs() map[string]string {
	out := map[string]string{}
	entries, err := os.ReadDir(ThemesDir())
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name, err := SanitizeThemeName(e.Name())
		if err != nil {
			continue
		}
		if _, dup := out[name]; dup {
			continue // first (sorted) entry wins, deterministically
		}
		out[name] = e.Name()
	}
	return out
}

// ThemeSourceFile is the theme.json backing a user pack, or "" when name is
// a builtin (or not installed). Watchers stamp this file so editing a pack
// applies without a restart.
func ThemeSourceFile(name string) string {
	clean, err := SanitizeThemeName(aliasThemeName(strings.ToLower(strings.TrimSpace(name))))
	if err != nil {
		return ""
	}
	dir, ok := userThemeDirs()[clean]
	if !ok {
		return ""
	}
	return filepath.Join(ThemesDir(), dir, "theme.json")
}

func readUserTheme(name string) (ThemePack, bool) {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return ThemePack{}, false
	}
	dir, ok := userThemeDirs()[clean]
	if !ok {
		return ThemePack{}, false
	}
	b, err := os.ReadFile(filepath.Join(ThemesDir(), dir, "theme.json"))
	if err != nil {
		return ThemePack{}, false
	}
	pack, err := parseThemeFile(clean, b, ThemeSourceUser)
	if err != nil {
		return ThemePack{}, false
	}
	return pack, true
}

func listUserThemeMap() map[string]ThemePack {
	out := map[string]ThemePack{}
	for name := range userThemeDirs() {
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

// ThemeEraGroup is a Settings section of built-in packs from one decade.
type ThemeEraGroup struct {
	Era   string
	Packs []ThemePack
}

// ListBuiltinThemesByEra returns unshadowed embedded packs grouped oldest → newest.
func ListBuiltinThemesByEra() []ThemeEraGroup {
	var groups []ThemeEraGroup
	var cur ThemeEraGroup
	for _, p := range ListBuiltinThemes() {
		era := p.Era
		if era == "" {
			era = "Built-in"
		}
		if cur.Era != "" && cur.Era != era {
			groups = append(groups, cur)
			cur = ThemeEraGroup{}
		}
		cur.Era = era
		cur.Packs = append(cur.Packs, p)
	}
	if cur.Era != "" {
		groups = append(groups, cur)
	}
	return groups
}

// ListBuiltinThemes returns embedded era packs when not shadowed.
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
// embedded starter. Legacy compound ids map to dark / light. Era aliases
// (classic95, luna-dark, …) resolve to the shipped pack id.
func LoadTheme(name string) (ThemePack, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ThemePack{}, false
	}
	name = aliasThemeName(strings.ToLower(name))
	// Sanitize before any path is built from name: look.json is shared with
	// other apps and a value like "../../../evil" must not reach ReadFile.
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return ThemePack{}, false
	}
	if pack, ok := readUserTheme(clean); ok {
		return pack, true
	}
	if pack, ok := loadEmbedded()[clean]; ok {
		return pack, true
	}
	packName, _, _ := SplitLookThemeName(clean)
	if packName != clean {
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
	tok := ThemeTokens{Family: a.Theme}
	if live, ok := LoadTheme(a.Name); ok {
		tok = live.Tokens
		if tok.Family == "" {
			tok.Family = a.Theme
		}
	} else {
		if base, ok := builtinEraPack(StarterName(a.Theme)); ok {
			tok = base.Tokens
		}
	}
	tok.Family = ParseTheme(string(tok.Family))
	tok = tok.Resolve()
	pack := ThemePack{
		Name:    clean,
		Label:   clean,
		Source:  ThemeSourceUser,
		Palette: tok.Family,
		Era:     tok.Era,
		Tokens:  tok,
	}
	doc := themeDoc(pack.Label, pack.Era, tok)
	if err := writeJSONFile(ThemeFile(clean), doc); err != nil {
		return ThemePack{}, err
	}
	return pack, nil
}

// DeleteUserTheme removes themes/<name>/ from disk. Built-in embedded
// starters cannot be deleted. The name must resolve to a user pack.
func DeleteUserTheme(name string) error {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return err
	}
	if _, ok := readUserTheme(clean); !ok {
		return fmt.Errorf("not a user theme: %s", clean)
	}
	dir, ok := userThemeDirs()[clean]
	if !ok {
		return fmt.Errorf("not a user theme: %s", clean)
	}
	return os.RemoveAll(filepath.Join(ThemesDir(), dir))
}

// AfterUserThemeDeleted rewrites a if it named the deleted pack.
// A shadowed builtin of the same name is kept; otherwise the matching
// embedded palette starter (dark / light) is selected.
func AfterUserThemeDeleted(a Appearance, deleted string) Appearance {
	a = a.Normalize()
	deleted = strings.ToLower(strings.TrimSpace(deleted))
	if a.Name != deleted {
		return a
	}
	if pack, ok := LoadTheme(deleted); ok {
		a.Name = pack.Name
		a.Theme = pack.Palette
		return a
	}
	return FallbackBuiltinTheme(a)
}

// FallbackBuiltinTheme keeps corners/icons and selects the embedded
// starter for a.Theme (or the other palette if that name is still a
// user pack).
func FallbackBuiltinTheme(a Appearance) Appearance {
	a = a.Normalize()
	name := StarterName(a.Theme)
	if pack, ok := LoadTheme(name); ok && pack.Source == ThemeSourceBuiltin {
		a.Name = pack.Name
		a.Theme = pack.Palette
		return a
	}
	other := ThemeLight
	if a.Theme == ThemeLight {
		other = ThemeDark
	}
	a.Name = StarterName(other)
	a.Theme = other
	return a
}
