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
// Current format stores theme, corners, icons, and icon size independently:
//
//	{ "theme": "dark", "corners": "square", "icons": "lucide", "iconSize": "medium" }
//
// Compound v0.11–v0.12.1 theme ids (dark-round-classic, light-square-sharp,
// dark-round, …) migrate to palette + corners. Pack-level corners/icons
// are ignored.
type appearanceFileJSON struct {
	// Version 2 introduced corners "theme" (the theme's native shape) as
	// the default; older files carry "round" as the written-out default.
	Version  int    `json:"version,omitempty"`
	Theme    string `json:"theme"`
	Corners  string `json:"corners,omitempty"`
	Icons    string `json:"icons,omitempty"`
	IconSize string `json:"iconSize,omitempty"`
	// ReduceMotion turns animations off.
	ReduceMotion bool `json:"reduceMotion,omitempty"`
	// FollowDesktop swaps the theme for its light or dark sibling to match
	// the desktop.
	FollowDesktop bool `json:"followDesktop,omitempty"`
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

func resolveAppearance(raw appearanceFileJSON) Appearance {
	a := DefaultAppearance()
	packName, cornersFromName, hasCorners := SplitLookThemeName(raw.Theme)
	if pack, ok := LoadTheme(strings.TrimSpace(raw.Theme)); ok {
		a.Name = pack.Name
		a.Theme = pack.Palette
	} else if pack, ok := LoadTheme(packName); ok {
		a.Name = pack.Name
		a.Theme = pack.Palette
	} else if packName == "light" {
		a.Name = StarterName(ThemeLight)
		a.Theme = ThemeLight
	} else if clean, err := SanitizeThemeName(packName); err == nil && clean != "" && clean != "dark" {
		// A pack that is not installed (yet) keeps its name so the pref
		// survives, but only in canonical form: look.json is shared with
		// other apps and the name ends up in file paths.
		a.Name = clean
		a.Theme = ParseTheme(clean)
	}
	if strings.TrimSpace(raw.Corners) != "" {
		a.Corners = ParseCorners(raw.Corners)
		if raw.Version < 2 && a.Corners == CornersRound {
			// Pre-v2 files wrote "round" as the default, before themes had
			// shapes of their own: read it as "keep the theme's shape".
			a.Corners = CornersTheme
		}
	} else if hasCorners {
		a.Corners = cornersFromName
		if a.Corners == CornersRound {
			a.Corners = CornersTheme
		}
	}
	if strings.TrimSpace(raw.Icons) != "" {
		a.Icons = ParseIconSet(raw.Icons)
	}
	if strings.TrimSpace(raw.IconSize) != "" {
		a.IconSize = ParseIconSize(raw.IconSize)
	}
	a.ReduceMotion = raw.ReduceMotion
	a.FollowDesktop = raw.FollowDesktop
	return a.Normalize()
}

// LoadAppearance reads XDG look.json. Missing or invalid files yield defaults.
// Compound theme ids and a legacy triad both resolve to independent
// theme / corners / icons fields.
//
// UITK_THEME=<pack> overrides the saved theme for this process, like
// GTK_THEME or QT_STYLE_OVERRIDE (UITK_THEME=win95 ./app); corners, icons
// and icon size still come from look.json. The override is never written
// back unless the user saves from Settings.
func LoadAppearance() Appearance {
	var raw appearanceFileJSON
	ok := false
	if b, err := os.ReadFile(AppearancePath()); err == nil && json.Unmarshal(b, &raw) == nil {
		ok = true
	} else {
		raw = appearanceFileJSON{}
	}
	if env := strings.TrimSpace(os.Getenv(ThemeEnv)); env != "" {
		// The pack asked for is the pack shown: no swapping it for its
		// light or dark sibling.
		raw.Theme = env
		raw.FollowDesktop = false
		ok = true
	}
	if !ok {
		return DefaultAppearance()
	}
	return resolveAppearance(raw)
}

// ThemeEnv names the environment variable that overrides the saved theme.
const ThemeEnv = "UITK_THEME"

// SaveAppearance writes look.json with theme, corners, icons, and iconSize (mode 0600).
func SaveAppearance(a Appearance) error {
	a = a.Normalize()
	return writeJSONFile(AppearancePath(), appearanceFileJSON{
		Version:  2,
		Theme:    a.Name,
		Corners:  string(a.Corners),
		Icons:    string(a.Icons),
		IconSize: string(a.IconSize),

		ReduceMotion:  a.ReduceMotion,
		FollowDesktop: a.FollowDesktop,
	})
}
