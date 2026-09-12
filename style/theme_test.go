package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedStartersCoverMatrix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got := ListThemes()
	if len(got) != 8 {
		t.Fatalf("starters %d: %+v", len(got), namesOf(got))
	}
	for _, pal := range []ThemeName{ThemeDark, ThemeLight} {
		for _, cor := range []CornerStyle{CornersRound, CornersSquare} {
			for _, ic := range []IconSetName{IconSetClassic, IconSetSharp} {
				name := StarterName(pal, cor, ic)
				pack, ok := LoadTheme(name)
				if !ok {
					t.Fatalf("missing %s", name)
				}
				if pack.Source != ThemeSourceBuiltin {
					t.Fatalf("%s source %s", name, pack.Source)
				}
				if pack.Palette != pal || pack.Corners != cor || pack.Icons != ic {
					t.Fatalf("%s %+v", name, pack)
				}
				look := pack.Look()
				if look.Name() != string(pal) || look.Corners() != cor || look.Icons() != ic {
					t.Fatalf("look %s %+v", name, LookAppearance(look))
				}
				if look.Pack() != name {
					t.Fatalf("pack %s got %s", name, look.Pack())
				}
			}
		}
	}
	if _, err := os.Stat(ThemesDir()); !os.IsNotExist(err) {
		t.Fatalf("listing builtins must not write %s", ThemesDir())
	}
}

func TestSaveAppearanceWritesThemeNameOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}
	if err := SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["theme"] != "light-square-sharp" {
		t.Fatalf("theme %v", doc["theme"])
	}
	if _, ok := doc["corners"]; ok {
		t.Fatalf("corners leaked: %s", raw)
	}
	if _, ok := doc["icons"]; ok {
		t.Fatalf("icons leaked: %s", raw)
	}
	got := LoadAppearance()
	if got != want.Normalize() {
		t.Fatalf("got %+v want %+v", got, want.Normalize())
	}
}

func TestLoadAppearanceMigratesLegacyTriad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := []byte("{\n  \"theme\": \"light\",\n  \"corners\": \"square\",\n  \"icons\": \"sharp\"\n}\n")
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	want := Appearance{Name: "light-square-sharp", Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	look := PreferredLook().(*Classic)
	if look.Name() != "light" || look.Corners() != CornersSquare || look.Icons() != IconSetSharp {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceBareDarkIsDefaultStarter(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if LoadAppearance() != DefaultAppearance() {
		t.Fatalf("%+v", LoadAppearance())
	}
}

func TestUserThemeOverridesBuiltin(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pack, err := ExportAppearance("dark-round-classic", Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp})
	if err != nil {
		t.Fatal(err)
	}
	if pack.Source != ThemeSourceUser || pack.Palette != ThemeLight {
		t.Fatalf("%+v", pack)
	}
	got, ok := LoadTheme("dark-round-classic")
	if !ok || got.Source != ThemeSourceUser || got.Palette != ThemeLight || got.Corners != CornersSquare {
		t.Fatalf("load %+v ok=%v", got, ok)
	}
	listed := ListThemes()
	users, builtins := 0, 0
	for _, p := range listed {
		if p.Name == "dark-round-classic" {
			if p.Source != ThemeSourceUser {
				t.Fatal("shadowed builtin still listed")
			}
			users++
		}
		if p.Source == ThemeSourceBuiltin {
			builtins++
		}
	}
	if users != 1 || builtins != 7 || len(listed) != 8 {
		t.Fatalf("list users=%d builtins=%d n=%d", users, builtins, len(listed))
	}
}

func TestExportThemeWritesUserPack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	look := Appearance{Theme: ThemeDark, Corners: CornersSquare, Icons: IconSetClassic}.Look()
	pack, err := ExportTheme("ocean", look)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Name != "ocean" || pack.Source != ThemeSourceUser {
		t.Fatalf("%+v", pack)
	}
	path := ThemeFile("ocean")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc themeFileJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Palette != "dark" || doc.Corners != "square" || doc.Icons != "classic" {
		t.Fatalf("%+v", doc)
	}
	listed := ListThemes()
	if listed[0].Name != "ocean" || listed[0].Source != ThemeSourceUser {
		t.Fatalf("user pack should be listed first: %+v", namesOf(listed))
	}
	if err := SaveAppearance(pack.Appearance()); err != nil {
		t.Fatal(err)
	}
	pref := LoadAppearance()
	if pref.Name != "ocean" || pref.Theme != ThemeDark || pref.Corners != CornersSquare {
		t.Fatalf("preferred %+v", pref)
	}
}

func TestSanitizeThemeName(t *testing.T) {
	got, err := SanitizeThemeName(" Ocean Breeze ")
	if err != nil || got != "ocean-breeze" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := SanitizeThemeName("../etc"); err == nil {
		t.Fatal("path escape")
	}
	if _, err := SanitizeThemeName(""); err == nil {
		t.Fatal("empty")
	}
}

func namesOf(packs []ThemePack) []string {
	out := make([]string, len(packs))
	for i, p := range packs {
		out[i] = p.Name
	}
	return out
}
