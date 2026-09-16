package demo

import (
	"os"
	"path/filepath"

	"github.com/codemodify/uitoolkit/dock"
)

// dockLayoutPath is $XDG_CONFIG_HOME/uitoolkit/<app>-dock.json (or
// ~/.config/uitoolkit/<app>-dock.json). A demo's arrangement lives in a
// file of its own: look.json is Settings' file and must never be written
// from here, as the mail client's chrome prefs are kept apart for the same
// reason.
func dockLayoutPath(name string) string {
	file := name + "-dock.json"
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", file)
	}
	if home, _ := os.UserHomeDir(); home != "" {
		return filepath.Join(home, ".config", "uitoolkit", file)
	}
	return filepath.Join(os.TempDir(), "uitoolkit-"+file)
}

// loadDockLayout reads an app's saved arrangement, or nil when there is
// none to read. A missing or unreadable file is not worth troubling the
// user with: the app comes up in its default arrangement.
func loadDockLayout(name string) []byte {
	b, err := os.ReadFile(dockLayoutPath(name))
	if err != nil {
		return nil
	}
	return b
}

// saveDockLayout writes the host's arrangement out. Errors are swallowed
// for the same reason: an app whose layout cannot be saved should still
// run.
func saveDockLayout(name string, host *dock.Host) {
	b, err := host.LayoutJSON()
	if err != nil {
		return
	}
	path := dockLayoutPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}
