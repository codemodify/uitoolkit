package mail

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/codemodify/uitoolkit/style"
)

// ChromePrefs is UI-only (card vs table, density). Lives next to mail.json.
type ChromePrefs struct {
	CardView bool   `json:"cardView"`
	Density  string `json:"density"`
	Layout   string `json:"layout,omitempty"`
	Light    bool   `json:"light,omitempty"`
}

func chromePrefsPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", "mailui.json")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "uitoolkit", "mailui.json")
	}
	return filepath.Join(os.TempDir(), "uitoolkit-mailui.json")
}

func loadChromePrefs() ChromePrefs {
	var p ChromePrefs
	b, err := os.ReadFile(chromePrefsPath())
	if err != nil {
		return p
	}
	_ = json.Unmarshal(b, &p)
	return p
}

func saveChromePrefs(p ChromePrefs) {
	path := chromePrefsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}

func (p ChromePrefs) density() style.Density {
	return style.ParseDensity(p.Density)
}
