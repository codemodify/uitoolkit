package dock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Saving and restoring an arrangement is every docking application's job,
// so it is the dock package's, not something each one re-implements: the
// three functions below are what examples/uitoolkit-sample-inspector used to carry itself.
//
// Each application's arrangement lives in a file of its own, named for
// the application, so two applications built on the toolkit never write
// over each other's — and neither ever touches look.json, which belongs
// to the Settings app.

// LayoutFile is where [Host.SaveLayoutFile] and [Host.LoadLayoutFile] keep
// the arrangement called name:
//
//	$XDG_CONFIG_HOME/uitoolkit/<name>-dock.json
//
// falling back to ~/.config/uitoolkit/<name>-dock.json, and to the
// temporary directory when there is no home either. name is an
// application's own name ("inspector"), not a path; see [ValidLayoutName].
func LayoutFile(name string) string {
	file := name + "-dock.json"
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit", file)
	}
	if home, _ := os.UserHomeDir(); home != "" {
		return filepath.Join(home, ".config", "uitoolkit", file)
	}
	return filepath.Join(os.TempDir(), "uitoolkit-"+file)
}

// ValidLayoutName reports whether name is usable as a layout name: a
// non-empty plain name with no path separator and no leading dot, so a
// layout can only ever be written beside the others.
func ValidLayoutName(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") {
		return false
	}
	return name == filepath.Base(name) && !strings.ContainsAny(name, `/\`)
}

// SaveLayoutFile writes the host's arrangement to [LayoutFile](name),
// creating the directory if it is not there yet.
//
// Call it when the window closes, or after a panel is moved. An
// application that does not mind losing the arrangement can ignore the
// error: a layout that cannot be saved is not worth interrupting anyone
// over, and the app comes up in its default arrangement next time.
func (h *Host) SaveLayoutFile(name string) error {
	if !ValidLayoutName(name) {
		return fmt.Errorf("dock: %q is not a layout name", name)
	}
	b, err := h.LayoutJSON()
	if err != nil {
		return err
	}
	path := LayoutFile(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadLayoutFile applies the arrangement saved under name, and reports
// whether one was applied.
//
// A first run has no file and a corrupt one cannot be read; both leave the
// host exactly as it was, so the usual call is
//
//	if ok, _ := host.LoadLayoutFile("inspector"); !ok {
//		host.SetDefaultLayout()
//	}
//
// The error says which of the two happened — use errors.Is(err,
// fs.ErrNotExist) for the first run — for an application that wants to
// tell the user its layout file is broken.
func (h *Host) LoadLayoutFile(name string) (bool, error) {
	if !ValidLayoutName(name) {
		return false, fmt.Errorf("dock: %q is not a layout name", name)
	}
	b, err := os.ReadFile(LayoutFile(name))
	if err != nil {
		return false, err
	}
	if err := h.ApplyLayoutJSON(b); err != nil {
		return false, err
	}
	return true, nil
}
