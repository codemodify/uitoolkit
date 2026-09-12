package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const appearanceFile = "look.json"

// appearanceFileJSON is the on-disk XDG document. Other apps (Mail, gallery)
// read the same file via LoadAppearance / PreferredLook.
//
// Current format stores the theme pack name only. A legacy v0.11.0/0.11.1
// file with theme/corners/icons is migrated to the matching starter name.
type appearanceFileJSON struct {
	Theme   string `json:"theme"`
	Corners string `json:"corners,omitempty"`
	Icons   string `json:"icons,omitempty"`
}

// ConfigDir is $XDG_CONFIG_HOME/uitoolkit (or ~/.config/uitoolkit).
func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "uitoolkit")
	}
	return filepath.Join(os.TempDir(), "uitoolkit")
}

// AppearancePath is $XDG_CONFIG_HOME/uitoolkit/look.json
// (or ~/.config/uitoolkit/look.json).
func AppearancePath() string {
	return filepath.Join(ConfigDir(), appearanceFile)
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func resolveLookThemeName(raw appearanceFileJSON) string {
	if raw.Corners != "" || raw.Icons != "" {
		return StarterName(ParseTheme(raw.Theme), ParseCorners(raw.Corners), ParseIconSet(raw.Icons))
	}
	name := strings.TrimSpace(raw.Theme)
	if name == "" {
		return DefaultThemeName
	}
	if _, ok := LoadTheme(name); ok {
		return name
	}
	switch name {
	case "dark", "Dark":
		return DefaultThemeName
	case "light", "Light", "paper":
		return StarterName(ThemeLight, CornersRound, IconSetClassic)
	default:
		return name
	}
}

// LoadAppearance reads XDG look.json. Missing or invalid files yield defaults.
// A legacy triad (theme + corners + icons) maps to the matching starter pack.
func LoadAppearance() Appearance {
	b, err := os.ReadFile(AppearancePath())
	if err != nil {
		return DefaultAppearance()
	}
	var raw appearanceFileJSON
	if json.Unmarshal(b, &raw) != nil {
		return DefaultAppearance()
	}
	name := resolveLookThemeName(raw)
	if pack, ok := LoadTheme(name); ok {
		return pack.Appearance()
	}
	return DefaultAppearance()
}

// SaveAppearance writes look.json with the theme pack name only (mode 0600).
func SaveAppearance(a Appearance) error {
	a = a.Normalize()
	return writeJSONFile(AppearancePath(), appearanceFileJSON{Theme: a.Name})
}
