package style

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var shippedIconSets = []string{"lucide", "phosphor", "tabler", "heroicons", "material-symbols"}

func TestRepoIconSetsCoverAllToolIcons(t *testing.T) {
	root := filepath.Join("..", "icons")
	for _, set := range shippedIconSets {
		for _, icon := range AllToolIcons() {
			lo := filepath.Join(root, set, ToolIconFileName(icon))
			hi := filepath.Join(root, set, ToolIconHiDPIFileName(icon))
			checkPNG(t, lo, 24)
			checkPNG(t, hi, 48)
		}
		if _, err := os.Stat(filepath.Join(root, set, "LICENSE")); err != nil {
			t.Errorf("missing %s/LICENSE", set)
		}
	}
}

func checkPNG(t *testing.T, path string, size int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("missing %s", path)
		return
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Errorf("decode %s: %v", path, err)
		return
	}
	if cfg.Width != size || cfg.Height != size {
		t.Errorf("%s size %d×%d want %d×%d", path, cfg.Width, cfg.Height, size, size)
	}
}

func TestListIconSetsListsInstalled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetIconCache()
	listed := ListIconSets()
	if len(listed) != 2 || listed[0].Name != IconSetClassic || listed[1].Name != IconSetSharp {
		t.Fatalf("drawn builtins only: %+v", listed)
	}
	if listed[0].Source != ThemeSourceBuiltin || listed[1].Source != ThemeSourceBuiltin {
		t.Fatalf("drawn source %+v", listed)
	}
	installRepoIconSet(t, "lucide")
	listed = ListIconSets()
	if len(listed) != 3 || listed[2].Name != IconSetLucide || listed[2].Source != ThemeSourceBuiltin {
		t.Fatalf("premiere should be builtin: %+v", listed)
	}
	if listed[2].Label != "Lucide" {
		t.Fatalf("label %q", listed[2].Label)
	}
	if len(ListBuiltinIconSets()) != 3 || len(ListUserIconSets()) != 0 {
		t.Fatalf("split builtin=%d user=%d", len(ListBuiltinIconSets()), len(ListUserIconSets()))
	}

	dst := filepath.Join(IconsDir(), "my-set")
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "icons", "lucide", "search.png")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "search.png"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	listed = ListIconSets()
	users := ListUserIconSets()
	if len(users) != 1 || users[0].Name != IconSetName("my-set") || users[0].Source != ThemeSourceUser {
		t.Fatalf("custom user set %+v listed %+v", users, listed)
	}
	if users[0].Label != "my-set" {
		t.Fatalf("user label %q", users[0].Label)
	}
	if err := DeleteUserIconSet(IconSetLucide); err == nil {
		t.Fatal("must not delete premiere")
	}
	if err := DeleteUserIconSet(IconSetClassic); err == nil {
		t.Fatal("must not delete drawn builtin")
	}
	if err := DeleteUserIconSet(IconSetName("my-set")); err != nil {
		t.Fatal(err)
	}
	if len(ListUserIconSets()) != 0 {
		t.Fatalf("user icons after delete %+v", ListUserIconSets())
	}
}

func TestFileIconLoadTintFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetIconCache()
	installRepoIconSet(t, "lucide")

	red := paintFileIcon(IconSetLucide, IconSearch, paintengine2d.RGB(1, 0, 0))
	blue := paintFileIcon(IconSetLucide, IconSearch, paintengine2d.RGB(0, 0, 1))
	if inkCount(red) < 8 || inkCount(blue) < 8 {
		t.Fatalf("no ink red=%d blue=%d", inkCount(red), inkCount(blue))
	}
	if !tinted(red, 1, 0, 0) {
		t.Fatal("search should tint red")
	}
	if !tinted(blue, 0, 0, 1) {
		t.Fatal("search should tint blue")
	}

	// Missing glyph → drawn classic (same ink mask as classic).
	classic := rasterIcon(IconSetClassic, IconCut)
	missing := rasterIcon(IconSetLucide, IconCut) // cut.png is installed; delete both sizes
	_ = os.Remove(filepath.Join(IconSetDir(IconSetLucide), "cut.png"))
	_ = os.Remove(filepath.Join(IconSetDir(IconSetLucide), "cut@2x.png"))
	resetIconCache()
	fallback := rasterIcon(IconSetLucide, IconCut)
	if maskDiff(classic, fallback) > 4 {
		t.Fatalf("missing file should match classic (diff=%d)", maskDiff(classic, fallback))
	}
	if maskDiff(classic, missing) < 8 {
		t.Fatal("installed lucide cut should differ from classic before delete")
	}

	// Unknown set name with no directory → classic.
	resetIconCache()
	unknown := rasterIcon(IconSetName("not-installed"), IconSearch)
	if maskDiff(classic, unknown) > 4 && maskDiff(rasterIcon(IconSetClassic, IconSearch), unknown) > 4 {
		t.Fatal("unknown file set should fall back")
	}
}

func TestFileIconPicksHiDPI(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetIconCache()
	dst := filepath.Join(IconSetDir(IconSetLucide))
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "icons", "lucide")
	// Only @2x for search — 1× dest should still paint via fallback candidate.
	hi, err := os.ReadFile(filepath.Join(src, "search@2x.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "search@2x.png"), hi, 0o600); err != nil {
		t.Fatal(err)
	}
	small := paintFileIcon(IconSetLucide, IconSearch, paintengine2d.RGB(1, 1, 1))
	if inkCount(small) < 8 {
		t.Fatalf("1× dest should use @2x when 24 is missing (ink=%d)", inkCount(small))
	}

	lo, err := os.ReadFile(filepath.Join(src, "search.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "search.png"), lo, 0o600); err != nil {
		t.Fatal(err)
	}
	resetIconCache()
	cands := toolIconFileCandidates(IconSearch, 24)
	if len(cands) != 2 || cands[0] != "search.png" {
		t.Fatalf("1× candidates %v", cands)
	}
	cands = toolIconFileCandidates(IconSearch, 48)
	if len(cands) != 2 || cands[0] != "search@2x.png" {
		t.Fatalf("2× candidates %v", cands)
	}
	large := paintengine2d.NewImage(64, 64)
	ctx := paintengine2d.NewContext(large)
	DrawToolIcon(ctx, paintengine2d.XYWH(8, 8, 48, 48), IconSearch, paintengine2d.RGB(0, 1, 0), IconSetLucide)
	if inkCount(large) < 16 {
		t.Fatalf("2× dest no ink %d", inkCount(large))
	}
	if !tinted(large, 0, 1, 0) {
		t.Fatal("2× dest should tint green")
	}
}

func TestLookJSONThemeAndIcons(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := Appearance{Name: "dark", Theme: ThemeDark, Corners: CornersRound, Icons: IconSetLucide}
	if err := SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(raw), `"theme": "dark"`, `"corners": "round"`, `"icons": "lucide"`) {
		t.Fatalf("look.json: %s", raw)
	}
	got := LoadAppearance()
	if got != want.Normalize() {
		t.Fatalf("got %+v want %+v", got, want.Normalize())
	}
	look := PreferredLook().(*Classic)
	if look.Pack() != "dark" || look.Icons() != IconSetLucide {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceIconsOverlayPackDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"light-square-sharp","icons":"phosphor"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if got.Name != "light" || got.Theme != ThemeLight || got.Corners != CornersSquare || got.Icons != IconSetPhosphor {
		t.Fatalf("%+v", got)
	}
}

func TestParseIconSetFileNames(t *testing.T) {
	if ParseIconSet("lucide") != IconSetLucide || ParseIconSet("phosphor") != IconSetPhosphor {
		t.Fatal("shipped names")
	}
	if ParseIconSet("tabler") != IconSetTabler || ParseIconSet("heroicons") != IconSetHeroicons {
		t.Fatal("tabler/heroicons")
	}
	if ParseIconSet("material-symbols") != IconSetMaterialSymbols || ParseIconSet("Material") != IconSetMaterialSymbols {
		t.Fatal("material-symbols")
	}
	if ParseIconSet("My Set") != IconSetName("my-set") {
		t.Fatalf("%q", ParseIconSet("My Set"))
	}
	if !IsFileIconSet(IconSetLucide) || IsFileIconSet(IconSetClassic) {
		t.Fatal("file vs builtin")
	}
	if FallbackIcons(IconSetLucide) != IconSetClassic || FallbackIcons(IconSetSharp) != IconSetSharp {
		t.Fatal("fallback")
	}
}

func installRepoIconSet(t *testing.T, name string) {
	t.Helper()
	src := filepath.Join("..", "icons", name)
	dst := filepath.Join(IconsDir(), name)
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func paintFileIcon(set IconSetName, icon ToolIcon, col paintengine2d.Color) *paintengine2d.Image {
	img := paintengine2d.NewImage(32, 32)
	ctx := paintengine2d.NewContext(img)
	DrawToolIcon(ctx, paintengine2d.XYWH(2, 2, 28, 28), icon, col, set)
	return img
}

func inkCount(img *paintengine2d.Image) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a > 20 {
				n++
			}
		}
	}
	return n
}

func tinted(img *paintengine2d.Image, r, g, b float32) bool {
	hits := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			pr, pg, pb, a := img.PremulAt(x, y)
			if a < 40 {
				continue
			}
			// Premul: channel ≈ tint * alpha. Require the dominant channel.
			if r > 0.5 && pr > pg && pr > pb {
				hits++
			}
			if b > 0.5 && pb > pr && pb > pg {
				hits++
			}
			if g > 0.5 && pg > pr && pg > pb {
				hits++
			}
		}
	}
	return hits > 4
}

func maskDiff(a, b *paintengine2d.Image) int {
	diff := 0
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			_, _, _, aa := a.PremulAt(x, y)
			_, _, _, ba := b.PremulAt(x, y)
			if (aa > 20) != (ba > 20) {
				diff++
			}
		}
	}
	return diff
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsStr(s, p) {
			return false
		}
	}
	return true
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
