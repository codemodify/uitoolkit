package style

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const appearanceFile = "look.json"

// appearanceFileJSON is the on-disk XDG document. Other apps (Mail, gallery)
// read the same file via LoadAppearance / PreferredLook.
type appearanceFileJSON struct {
	Theme   string `json:"theme"`
	Corners string `json:"corners"`
	Icons   string `json:"icons"`
}

// AppearancePath is $XDG_CONFIG_HOME/uitoolkit/look.json
// (or ~/.config/uitoolkit/look.json).
func AppearancePath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", appearanceFile)
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "uitoolkit", appearanceFile)
	}
	return filepath.Join(os.TempDir(), "uitoolkit-look.json")
}

// LoadAppearance reads XDG look.json. Missing or invalid files yield defaults.
func LoadAppearance() Appearance {
	a := DefaultAppearance()
	b, err := os.ReadFile(AppearancePath())
	if err != nil {
		return a
	}
	var raw appearanceFileJSON
	if json.Unmarshal(b, &raw) != nil {
		return a
	}
	a.Theme = ParseTheme(raw.Theme)
	a.Corners = ParseCorners(raw.Corners)
	a.Icons = ParseIconSet(raw.Icons)
	return a
}

// SaveAppearance writes look.json (mode 0600) under the XDG config dir.
func SaveAppearance(a Appearance) error {
	a = a.Normalize()
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(appearanceFileJSON{
		Theme:   string(a.Theme),
		Corners: string(a.Corners),
		Icons:   string(a.Icons),
	}, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}
