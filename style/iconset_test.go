package style

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestRepoIconSetsCoverAllToolIcons(t *testing.T) {
	root := filepath.Join("..", "icons")
	for _, set := range []string{"filled", "outline", "duotone"} {
		for _, icon := range AllToolIcons() {
			path := filepath.Join(root, set, ToolIconFileName(icon))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("missing %s", path)
				continue
			}
			doc, err := parseSVG(raw)
			if err != nil || doc.empty() {
				t.Errorf("parse %s: %v empty=%v", path, err, doc.empty())
			}
			if doc.vbW != 24 || doc.vbH != 24 {
				t.Errorf("%s viewBox %v×%v", path, doc.vbW, doc.vbH)
			}
		}
	}
}

func TestListIconSetsListsInstalled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetIconCache()
	listed := ListIconSets()
	if len(listed) != 2 || listed[0].Name != IconSetClassic || listed[1].Name != IconSetSharp {
		t.Fatalf("builtins only: %+v", listed)
	}
	installRepoIconSet(t, "outline")
	listed = ListIconSets()
	if len(listed) != 3 || listed[2].Name != IconSetOutline || listed[2].Source != "user" {
		t.Fatalf("installed %+v", listed)
	}
}

func TestFileIconLoadTintFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetIconCache()
	installRepoIconSet(t, "outline")

	red := paintFileIcon(IconSetOutline, IconSearch, paintengine2d.RGB(1, 0, 0))
	blue := paintFileIcon(IconSetOutline, IconSearch, paintengine2d.RGB(0, 0, 1))
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
	missing := rasterIcon(IconSetOutline, IconCut) // cut.svg is installed; delete it
	_ = os.Remove(filepath.Join(IconSetDir(IconSetOutline), "cut.svg"))
	resetIconCache()
	fallback := rasterIcon(IconSetOutline, IconCut)
	if maskDiff(classic, fallback) > 4 {
		t.Fatalf("missing file should match classic (diff=%d)", maskDiff(classic, fallback))
	}
	if maskDiff(classic, missing) < 8 {
		t.Fatal("installed outline cut should differ from classic before delete")
	}

	// Unknown set name with no directory → classic.
	resetIconCache()
	unknown := rasterIcon(IconSetName("not-installed"), IconSearch)
	if maskDiff(classic, unknown) > 4 && maskDiff(rasterIcon(IconSetClassic, IconSearch), unknown) > 4 {
		t.Fatal("unknown file set should fall back")
	}
}

func TestLookJSONThemeAndIcons(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := Appearance{Name: "dark-round-classic", Theme: ThemeDark, Corners: CornersRound, Icons: IconSetOutline}
	if err := SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(raw), `"theme": "dark-round-classic"`, `"icons": "outline"`) {
		t.Fatalf("look.json: %s", raw)
	}
	got := LoadAppearance()
	if got != want.Normalize() {
		t.Fatalf("got %+v want %+v", got, want.Normalize())
	}
	look := PreferredLook().(*Classic)
	if look.Pack() != "dark-round-classic" || look.Icons() != IconSetOutline {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceIconsOverlayPackDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := AppearancePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"light-square-sharp","icons":"duotone"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if got.Name != "light-square-sharp" || got.Theme != ThemeLight || got.Corners != CornersSquare || got.Icons != IconSetDuotone {
		t.Fatalf("%+v", got)
	}
}

func TestParseIconSetFileNames(t *testing.T) {
	if ParseIconSet("outline") != IconSetOutline || ParseIconSet("filled") != IconSetFilled {
		t.Fatal("shipped names")
	}
	if ParseIconSet("My Set") != IconSetName("my-set") {
		t.Fatalf("%q", ParseIconSet("My Set"))
	}
	if !IsFileIconSet(IconSetOutline) || IsFileIconSet(IconSetClassic) {
		t.Fatal("file vs builtin")
	}
	if FallbackIcons(IconSetOutline) != IconSetClassic || FallbackIcons(IconSetSharp) != IconSetSharp {
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
