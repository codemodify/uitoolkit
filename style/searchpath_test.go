package style

import (
	"os"
	"path/filepath"
	"testing"
)

// writePNG puts a 1x1 PNG at path, so a stem counts as present.
func writePNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, embeddedNoIcon24, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ~/.config/uitoolkit belongs to the person using the machine — it is
// this toolkit's ~/.icons. An application that ships art of its own must
// not write there: two that do overwrite each other, last one wins, and
// a program built against a newer toolkit installs stems an older
// program's copy then removes.
//
// So an application keeps its art wherever it likes and registers the
// directory. This is the whole contract.
func TestAppSearchPathFillsGapsWithoutTouchingTheUsers(t *testing.T) {
	user := t.TempDir()
	app := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	InvalidateIconCache()

	// The user installed lucide a year ago: it has "save" and not "print".
	writePNG(t, filepath.Join(user, "uitoolkit", "icons", "lucide", "save.png"))
	// The application ships the same set with both.
	writePNG(t, filepath.Join(app, "icons", "lucide", "save.png"))
	writePNG(t, filepath.Join(app, "icons", "lucide", "print.png"))

	AddSearchPath(app)

	dirs := IconSetDirs(IconSetLucide)
	if len(dirs) != 2 {
		t.Fatalf("%d directories for the set, want the user's and the app's: %v", len(dirs), dirs)
	}
	if got, want := dirs[0], filepath.Join(user, "uitoolkit", "icons", "lucide"); got != want {
		t.Errorf("the user's directory is not first: %v", dirs)
	}

	// The stem the user's copy predates comes from the application.
	if _, ok := loadFileIcon(IconSetLucide, IconPrint, 24); !ok {
		t.Error("print did not resolve from the application's copy")
	}
	// And nothing was written into the user's directory.
	if _, err := os.Stat(filepath.Join(user, "uitoolkit", "icons", "lucide", "print.png")); err == nil {
		t.Error("the toolkit wrote into the user's icon directory")
	}
}

// Where the user has the file, the user's copy is the one used: their
// theming choice is not overridden by whatever an application ships.
func TestUsersCopyWinsOverTheApps(t *testing.T) {
	user := t.TempDir()
	app := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	InvalidateIconCache()

	userFile := filepath.Join(user, "uitoolkit", "icons", "lucide", "save.png")
	writePNG(t, userFile)
	writePNG(t, filepath.Join(app, "icons", "lucide", "save.png"))
	AddSearchPath(app)

	dirs := IconSetDirs(IconSetLucide)
	if filepath.Dir(userFile) != dirs[0] {
		t.Fatalf("the user's directory is not searched first: %v", dirs)
	}
}

// A set only an application ships is listed and selectable, so a program
// can use its own art without asking anyone to install anything.
func TestAppOnlyIconSetIsListed(t *testing.T) {
	user := t.TempDir()
	app := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	InvalidateIconCache()

	writePNG(t, filepath.Join(app, "icons", "housestyle", "save.png"))
	AddSearchPath(app)
	InvalidateIconCache()

	found := false
	for _, s := range ListIconSets() {
		if string(s.Name) == "housestyle" {
			found = true
		}
	}
	if !found {
		t.Error("a set the application ships is not listed")
	}
}

// Two applications registering their own directories do not collide:
// each keeps its own, and neither has written anywhere shared.
func TestTwoAppsDoNotOverrideEachOther(t *testing.T) {
	user := t.TempDir()
	first, second := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	InvalidateIconCache()

	writePNG(t, filepath.Join(first, "icons", "one", "save.png"))
	writePNG(t, filepath.Join(second, "icons", "two", "save.png"))
	AddSearchPath(first)
	AddSearchPath(second)
	InvalidateIconCache()

	names := map[string]bool{}
	for _, s := range ListIconSets() {
		names[string(s.Name)] = true
	}
	if !names["one"] || !names["two"] {
		t.Errorf("both applications' sets should be listed, got %v", names)
	}
}

// Registering the same directory twice is ignored, so it is safe from an
// init or from several packages.
func TestAddSearchPathIsIdempotent(t *testing.T) {
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	dir := t.TempDir()
	AddSearchPath(dir)
	AddSearchPath(dir)
	if got := len(SearchPaths()); got != 1 {
		t.Errorf("%d paths registered, want 1", got)
	}
	AddSearchPath("")
	if got := len(SearchPaths()); got != 1 {
		t.Errorf("an empty path was registered")
	}
}

// Themes and skins take the same route, so an application shipping one
// does not have to write into the user's directory either.
func TestThemesAndSkinsSearchTheSamePaths(t *testing.T) {
	user := t.TempDir()
	app := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)
	AddSearchPath(app)

	for _, tc := range []struct {
		kind string
		dirs []string
	}{
		{"icons", IconSearchDirs()},
		{"themes", ThemeSearchDirs()},
		{"skins", SkinSearchDirs()},
	} {
		if len(tc.dirs) != 2 {
			t.Errorf("%s: %d directories, want the user's and the app's", tc.kind, len(tc.dirs))
			continue
		}
		if want := filepath.Join(user, "uitoolkit", tc.kind); tc.dirs[0] != want {
			t.Errorf("%s: first is %q, want the user's %q", tc.kind, tc.dirs[0], want)
		}
		if want := filepath.Join(app, tc.kind); tc.dirs[1] != want {
			t.Errorf("%s: second is %q, want the app's %q", tc.kind, tc.dirs[1], want)
		}
	}
}

// An application's theme is not Settings' to delete: removing it would
// be one program reaching into another's files.
func TestDeletingAnAppThemeIsRefused(t *testing.T) {
	user := t.TempDir()
	app := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", user)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)

	dir := filepath.Join(app, "themes", "housestyle")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "theme.json"),
		[]byte(`{"name":"housestyle","palette":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	AddSearchPath(app)

	if err := DeleteUserTheme("housestyle"); err == nil {
		t.Error("Settings deleted a theme belonging to an application")
	}
	if _, err := os.Stat(filepath.Join(dir, "theme.json")); err != nil {
		t.Error("the application's theme file was removed")
	}
}
