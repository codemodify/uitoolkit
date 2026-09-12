package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedStartersArePalettes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got := ListThemes()
	if len(got) != 2 {
		t.Fatalf("starters %d: %+v", len(got), namesOf(got))
	}
	for _, pal := range []ThemeName{ThemeDark, ThemeLight} {
		name := StarterName(pal)
		pack, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if pack.Source != ThemeSourceBuiltin {
			t.Fatalf("%s source %s", name, pack.Source)
		}
		if pack.Palette != pal {
			t.Fatalf("%s %+v", name, pack)
		}
		look := pack.Look()
		if look.Name() != string(pal) || look.Pack() != name {
			t.Fatalf("look %s %+v", name, LookAppearance(look))
		}
	}
	if _, err := os.Stat(ThemesDir()); !os.IsNotExist(err) {
		t.Fatalf("listing builtins must not write %s", ThemesDir())
	}
}

func TestSplitLookThemeName(t *testing.T) {
	cases := []struct {
		in         string
		pack       string
		corners    CornerStyle
		hasCorners bool
	}{
		{"dark", "dark", CornersRound, false},
		{"light", "light", CornersRound, false},
		{"dark-round", "dark", CornersRound, true},
		{"dark-round-classic", "dark", CornersRound, true},
		{"dark-round-sharp", "dark", CornersRound, true},
		{"light-square-sharp", "light", CornersSquare, true},
		{"light-square-classic", "light", CornersSquare, true},
		{"dark-square", "dark", CornersSquare, true},
		{"  Light-Round-Classic  ", "light", CornersRound, true},
		{"ocean", "ocean", CornersRound, false},
	}
	for _, tc := range cases {
		pack, cor, has := SplitLookThemeName(tc.in)
		if pack != tc.pack || cor != tc.corners || has != tc.hasCorners {
			t.Fatalf("%q → %s %s has=%v want %s %s has=%v", tc.in, pack, cor, has, tc.pack, tc.corners, tc.hasCorners)
		}
		if CanonicalStarterName(tc.in) != tc.pack {
			t.Fatalf("canonical %q → %q want %q", tc.in, CanonicalStarterName(tc.in), tc.pack)
		}
	}
}

func TestLoadThemeMapsLegacyStarterNames(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, old := range []string{"dark-round-classic", "dark-round-sharp", "dark-round", "light-square-sharp"} {
		pack, ok := LoadTheme(old)
		if !ok {
			t.Fatalf("missing map for %s", old)
		}
		want := CanonicalStarterName(old)
		if pack.Name != want || pack.Source != ThemeSourceBuiltin {
			t.Fatalf("%s → %+v want %s builtin", old, pack, want)
		}
	}
}

func TestSaveAppearanceWritesTriad(t *testing.T) {
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
	if doc["theme"] != "light" {
		t.Fatalf("theme %v", doc["theme"])
	}
	if doc["corners"] != "square" {
		t.Fatalf("corners %v", doc["corners"])
	}
	if doc["icons"] != "sharp" {
		t.Fatalf("icons %v", doc["icons"])
	}
	got := LoadAppearance()
	if got != want.Normalize() {
		t.Fatalf("got %+v want %+v", got, want.Normalize())
	}
}

func TestLoadAppearanceReadsTriad(t *testing.T) {
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
	want := Appearance{Name: "light", Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
	look := PreferredLook().(*Classic)
	if look.Name() != "light" || look.Corners() != CornersSquare || look.Icons() != IconSetSharp || look.Pack() != "light" {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceMigratesCompoundNames(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		raw  string
		want Appearance
	}{
		{`{"theme":"light-square-sharp","icons":"phosphor"}`, Appearance{Name: "light", Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetPhosphor}},
		{`{"theme":"dark-round-classic","icons":"lucide"}`, Appearance{Name: "dark", Theme: ThemeDark, Corners: CornersRound, Icons: IconSetLucide}},
		{`{"theme":"dark-round"}`, Appearance{Name: "dark", Theme: ThemeDark, Corners: CornersRound, Icons: IconSetClassic}},
		{`{"theme":"light-square","corners":"round","icons":"sharp"}`, Appearance{Name: "light", Theme: ThemeLight, Corners: CornersRound, Icons: IconSetSharp}},
	}
	for _, tc := range cases {
		if err := os.WriteFile(path, []byte(tc.raw+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got := LoadAppearance()
		if got != tc.want {
			t.Fatalf("%s → %+v want %+v", tc.raw, got, tc.want)
		}
	}
}

func TestLoadAppearanceBareDarkIsDefault(t *testing.T) {
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
	pack, err := ExportAppearance("dark", Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp})
	if err != nil {
		t.Fatal(err)
	}
	if pack.Source != ThemeSourceUser || pack.Palette != ThemeLight {
		t.Fatalf("%+v", pack)
	}
	got, ok := LoadTheme("dark")
	if !ok || got.Source != ThemeSourceUser || got.Palette != ThemeLight {
		t.Fatalf("load %+v ok=%v", got, ok)
	}
	listed := ListThemes()
	users, builtins := 0, 0
	for _, p := range listed {
		if p.Name == "dark" {
			if p.Source != ThemeSourceUser {
				t.Fatal("shadowed builtin still listed")
			}
			users++
		}
		if p.Source == ThemeSourceBuiltin {
			builtins++
		}
	}
	if users != 1 || builtins != 1 || len(listed) != 2 {
		t.Fatalf("list users=%d builtins=%d n=%d %+v", users, builtins, len(listed), namesOf(listed))
	}
	if len(ListBuiltinThemes()) != 1 || len(ListUserThemes()) != 1 {
		t.Fatalf("split builtin=%d user=%d", len(ListBuiltinThemes()), len(ListUserThemes()))
	}
}

func TestExportThemeWritesPaletteOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	look := Appearance{Theme: ThemeDark, Corners: CornersSquare, Icons: IconSetClassic}.Look()
	pack, err := ExportTheme("ocean", look)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Name != "ocean" || pack.Source != ThemeSourceUser || pack.Palette != ThemeDark {
		t.Fatalf("%+v", pack)
	}
	path := ThemeFile("ocean")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc themeFileJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Palette != "dark" {
		t.Fatalf("%+v", doc)
	}
	if doc.Corners != "" || doc.Icons != "" {
		t.Fatalf("pack must be palette only: %+v", doc)
	}
	listed := ListThemes()
	if listed[len(listed)-1].Name != "ocean" || listed[len(listed)-1].Source != ThemeSourceUser {
		t.Fatalf("user pack should be listed after builtins: %+v", namesOf(listed))
	}
	if err := SaveAppearance(Appearance{Name: pack.Name, Theme: pack.Palette, Corners: CornersSquare, Icons: IconSetClassic}); err != nil {
		t.Fatal(err)
	}
	pref := LoadAppearance()
	if pref.Name != "ocean" || pref.Theme != ThemeDark || pref.Corners != CornersSquare {
		t.Fatalf("preferred %+v", pref)
	}
}

func TestDeleteUserTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := ExportAppearance("ocean", Appearance{Theme: ThemeLight}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteUserTheme("dark"); err == nil {
		t.Fatal("must not delete builtin")
	}
	if err := DeleteUserTheme("../etc"); err == nil {
		t.Fatal("path escape")
	}
	if err := DeleteUserTheme("ocean"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ThemeFile("ocean")); !os.IsNotExist(err) {
		t.Fatalf("theme dir still present: %v", err)
	}
	if _, ok := LoadTheme("ocean"); ok {
		t.Fatal("deleted pack still loads")
	}
	if len(ListUserThemes()) != 0 {
		t.Fatalf("user list %+v", ListUserThemes())
	}
}

func TestDeleteUserThemeUnshadowsBuiltin(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := ExportAppearance("dark", Appearance{Theme: ThemeLight}); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadTheme("dark")
	if !ok || got.Source != ThemeSourceUser || got.Palette != ThemeLight {
		t.Fatalf("shadow %+v ok=%v", got, ok)
	}
	if err := DeleteUserTheme("dark"); err != nil {
		t.Fatal(err)
	}
	got, ok = LoadTheme("dark")
	if !ok || got.Source != ThemeSourceBuiltin || got.Palette != ThemeDark {
		t.Fatalf("unshadow %+v ok=%v", got, ok)
	}
	a := AfterUserThemeDeleted(Appearance{Name: "dark", Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}, "dark")
	if a.Name != "dark" || a.Theme != ThemeDark || a.Corners != CornersSquare || a.Icons != IconSetSharp {
		t.Fatalf("after delete shadow %+v", a)
	}
	a = AfterUserThemeDeleted(Appearance{Name: "ocean", Theme: ThemeLight, Corners: CornersSquare}, "ocean")
	if a.Name != "light" || a.Theme != ThemeLight || a.Corners != CornersSquare {
		t.Fatalf("after delete user %+v", a)
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
