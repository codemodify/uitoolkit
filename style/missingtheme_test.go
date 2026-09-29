package style

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLook puts a look.json naming theme in a fresh config home.
func writeLook(t *testing.T, theme string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "uitoolkit"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"version":2,"theme":"` + theme + `"}`
	if err := os.WriteFile(filepath.Join(dir, "uitoolkit", "look.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A saved pack this build cannot load must not be read as dark.
//
// Theme engines are opt-in and a look.json is shared with every other
// uitoolkit application on the machine, so a name from somebody else's
// build reaches this one routinely. It used to be run through
// ParseTheme, which answers the two words "light" and "dark" and reads
// everything else as dark: the window then painted the dark palette,
// and a user whose light theme had turned black had nothing to go on.
func TestMissingPackDoesNotTurnTheLookDark(t *testing.T) {
	writeLook(t, "no-such-pack-in-any-build")
	a := LoadAppearance()
	if a.Name != "no-such-pack-in-any-build" {
		t.Fatalf("the name did not survive: %q", a.Name)
	}
	if !a.Missing() {
		t.Fatal("Missing() is false for a pack that cannot be loaded")
	}
	if a.Theme != DefaultAppearance().Theme {
		t.Fatalf("family %q, want the default appearance's %q", a.Theme, DefaultAppearance().Theme)
	}
}

// And the look it builds is the one this build would have shown if
// look.json had said nothing — the default pack, not a bare starter with
// no era at all.
func TestMissingPackShowsTheBuildsDefault(t *testing.T) {
	writeLook(t, "no-such-pack-in-any-build")
	got := LoadAppearance().Look()
	def, ok := LoadTheme(DefaultTheme())
	if !ok {
		t.Skip("this build has no default pack")
	}
	want := Appearance{Name: def.Name, Theme: def.Palette, Corners: CornersTheme,
		Icons: IconSetClassic, IconSize: IconSizeMedium}.Look()
	if got.Palette() != want.Palette() {
		t.Fatalf("painted %s's palette, want %s's", got.Pack(), def.Name)
	}
	// The name is still the user's, so the preference survives and comes
	// back the moment the engine is in the build.
	if got.Pack() != "no-such-pack-in-any-build" {
		t.Fatalf("the look forgot the saved pack: %q", got.Pack())
	}
}

// Missing is answered, not stored, so it cannot go stale.
func TestMissingIsAnswered(t *testing.T) {
	writeLook(t, "no-such-pack-in-any-build")
	a := LoadAppearance()
	if !a.Missing() {
		t.Fatal("missing")
	}
	a.Name = DefaultTheme()
	if a.Missing() {
		t.Fatalf("%q reads as missing", a.Name)
	}
	a.Name = ""
	if a.Missing() {
		t.Fatal("an empty name is not a missing pack")
	}
	if !ThemeAvailable(DefaultTheme()) || ThemeAvailable("no-such-pack-in-any-build") {
		t.Fatal("ThemeAvailable disagrees")
	}
}

// The note names both packs and can be logged unconditionally.
func TestMissingThemeNote(t *testing.T) {
	if n := MissingThemeNote(DefaultTheme()); n != "" {
		t.Fatalf("a pack that is here got a note: %q", n)
	}
	if n := MissingThemeNote(""); n != "" {
		t.Fatalf("an empty name got a note: %q", n)
	}
	n := MissingThemeNote("no-such-pack-in-any-build")
	for _, want := range []string{"no-such-pack-in-any-build", DefaultTheme(), "engines.md"} {
		if !strings.Contains(n, want) {
			t.Fatalf("the note does not mention %q: %q", want, n)
		}
	}
}

// A user pack on disk is not missing, whatever engines this build has:
// it carries its own palette and needs nobody's init.
func TestUserPackIsNeverMissing(t *testing.T) {
	writeLook(t, "light")
	if _, err := ExportAppearance("my-export", Appearance{Name: "light", Theme: ThemeLight}.Normalize()); err != nil {
		t.Fatal(err)
	}
	a := Appearance{Name: "my-export", Theme: ThemeLight}
	if a.Missing() {
		t.Fatal("a user export reads as missing")
	}
}

// The case that was reported, in the build where it happens: a narrowed
// build carrying a look.json that names a light pack from a wider one.
// metal-steel is light, and it used to come up dark.
//
// It runs only where metal-steel is genuinely absent — tools/test.sh
// builds every engine, so this is the run docs/engines.md calls the
// default product build (`tools/testenv.sh go test ./style/`).
func TestLightPackFromAWiderBuildStaysLight(t *testing.T) {
	const pack = "metal-steel"
	if packBuilt(pack) {
		t.Skipf("%q is in this build; the bug needs one without it", pack)
	}
	writeLook(t, pack)
	a := LoadAppearance()
	if !a.Missing() {
		t.Fatalf("%q loaded after all", pack)
	}
	if a.Theme == ThemeDark && DefaultAppearance().Theme == ThemeLight {
		t.Fatalf("%q came up dark in a build whose default is light", pack)
	}
	if got, want := a.Theme, DefaultAppearance().Theme; got != want {
		t.Fatalf("family %q, want %q", got, want)
	}
}
