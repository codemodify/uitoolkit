package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Two programs built with this toolkit must not claim one identity.
//
// Both Linux backends hardcoded "uitoolkit": Wayland in
// xdg_toplevel.set_app_id, X11 in WM_CLASS. A desktop did exactly what it
// was told, so a mail client and a password vault — two unrelated
// programs — shared one task-bar entry and one icon, and nothing the
// applications could do would separate them.
func TestAppIDSeparatesTwoProgramsByDefault(t *testing.T) {
	SetAppID("")
	t.Cleanup(func() { SetAppID("") })

	// The default is the executable's own name, which is what makes two
	// programs distinct without either of them asking.
	exe, err := os.Executable()
	if err != nil {
		t.Skip("no executable path on this system")
	}
	want := sanitizeAppID(filepath.Base(exe))
	if got := AppID(); got != want {
		t.Errorf("default app id is %q, want the executable's name %q", got, want)
	}
	if AppID() == "uitoolkit" && want != "uitoolkit" {
		t.Error("the default is still the constant every program shared")
	}

	// X11 carries the id and its capitalized form, and WM_CLASS wants
	// both.
	if c := AppClass(); c == "" || c[0] < 'A' || c[0] > 'Z' {
		t.Errorf("class %q is not capitalized", c)
	}
	if !strings.EqualFold(AppClass(), AppID()) {
		t.Errorf("class %q and id %q should differ only in case", AppClass(), AppID())
	}

	// An application that ships a .desktop file states its id.
	SetAppID("com.example.Notes")
	if got := AppID(); got != "com.example.Notes" {
		t.Errorf("app id is %q after being set", got)
	}

	// What a protocol will not carry is dropped rather than passed on.
	SetAppID("my app/../../etc/passwd\x00; rm -rf")
	got := AppID()
	for _, bad := range []string{"/", "\x00", " ", ";"} {
		if strings.Contains(got, bad) {
			t.Errorf("sanitized id %q still has %q", got, bad)
		}
	}
	if got == "" {
		t.Error("sanitizing left nothing at all")
	}

	// And an empty one goes back to the default rather than to nothing:
	// a window with no id at all is worse than a shared one.
	SetAppID("")
	if AppID() == "" {
		t.Error("clearing the id left a window with no identity")
	}
}
