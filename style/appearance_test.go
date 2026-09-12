package style

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestAppearanceLookThemeAndCorners(t *testing.T) {
	dark := Appearance{Theme: ThemeDark, Corners: CornersRound, Icons: IconSetClassic}.Look()
	if dark.Name() != "dark" || dark.Corners() != CornersRound || dark.Pack() != DefaultThemeName {
		t.Fatalf("dark %+v", LookAppearance(dark))
	}
	if dark.Metrics().Radius < 4 || dark.Metrics().RadiusSmall < 2 {
		t.Fatalf("round radii %+v", dark.Metrics())
	}
	sq := Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}.Look()
	if sq.Name() != "light" || sq.Corners() != CornersSquare || sq.Icons() != IconSetSharp || sq.Pack() != "light-square-sharp" {
		t.Fatalf("square %+v", LookAppearance(sq))
	}
	if sq.Metrics().Radius != 0 || sq.Metrics().RadiusSmall != 0 {
		t.Fatalf("square radii %+v", sq.Metrics())
	}
}

func TestWithAppearancePreservesScale(t *testing.T) {
	hi := WithScale(DarkLook(), 2)
	next := WithAppearance(hi, Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp})
	if next.Name() != "light" {
		t.Fatal("theme")
	}
	if next.Metrics().FontSize != hi.Metrics().FontSize {
		t.Fatalf("font %v vs %v", next.Metrics().FontSize, hi.Metrics().FontSize)
	}
	if next.Metrics().Radius != 0 {
		t.Fatalf("square should zero radius, got %v", next.Metrics().Radius)
	}
	c := next.(*Classic)
	if c.Icons() != IconSetSharp {
		t.Fatal("icons")
	}
	back := WithCorners(next, CornersRound)
	if back.Metrics().Radius <= 0 {
		t.Fatal("round restore")
	}
}

func TestWithDensityKeepsCornersAndIcons(t *testing.T) {
	look := Appearance{Theme: ThemeDark, Corners: CornersSquare, Icons: IconSetSharp}.Look()
	d := WithDensity(look, DensityCompact)
	c := d.(*Classic)
	if c.Corners() != CornersSquare || c.Icons() != IconSetSharp {
		t.Fatalf("lost appearance %+v", LookAppearance(d))
	}
	if d.Metrics().Radius != 0 {
		t.Fatal("compact square radius")
	}
	if d.Metrics().ControlH >= DefaultMetrics().ControlH {
		t.Fatal("compact height")
	}
}

func TestWithScaleKeepsIcons(t *testing.T) {
	look := Appearance{Icons: IconSetSharp, Corners: CornersSquare}.Look()
	hi := WithScale(look, 2).(*Classic)
	if hi.Icons() != IconSetSharp || hi.Corners() != CornersSquare {
		t.Fatalf("%+v", LookAppearance(hi))
	}
}

func TestSquareButtonRadiusZero(t *testing.T) {
	round := DarkLook()
	square := Appearance{Corners: CornersSquare}.Look()
	if round.Metrics().RadiusSmall <= 0 {
		t.Fatal("round")
	}
	if square.Metrics().RadiusSmall != 0 || square.Metrics().Radius != 0 {
		t.Fatal("square metrics")
	}
}

func TestIconSetsPaintDistinctInk(t *testing.T) {
	classic := paintIcon(IconSetClassic, IconSearch)
	sharp := paintIcon(IconSetSharp, IconSearch)
	if classic == 0 || sharp == 0 {
		t.Fatalf("ink classic=%d sharp=%d", classic, sharp)
	}
	same := 0
	imgC := rasterIcon(IconSetClassic, IconSearch)
	imgS := rasterIcon(IconSetSharp, IconSearch)
	diff := 0
	for y := 0; y < imgC.Height; y++ {
		for x := 0; x < imgC.Width; x++ {
			_, _, _, ac := imgC.PremulAt(x, y)
			_, _, _, as := imgS.PremulAt(x, y)
			if ac > 20 {
				same++
			}
			if (ac > 20) != (as > 20) {
				diff++
			}
		}
	}
	if diff < 20 {
		t.Fatalf("classic and sharp search glyphs look the same (diff=%d)", diff)
	}
	_ = same
}

func TestAppearancePrefsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}
	if err := SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	path := AppearancePath()
	if filepath.Dir(path) != filepath.Join(dir, "uitoolkit") {
		t.Fatalf("path %s", path)
	}
	got := LoadAppearance()
	if got != want.Normalize() {
		t.Fatalf("got %+v want %+v", got, want)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
	look := PreferredLook().(*Classic)
	if look.Name() != "light" || look.Corners() != CornersSquare || look.Icons() != IconSetSharp {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceMissingIsDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if LoadAppearance() != DefaultAppearance() {
		t.Fatal(LoadAppearance())
	}
}

func rasterIcon(set IconSetName, icon ToolIcon) *paintengine2d.Image {
	img := paintengine2d.NewImage(32, 32)
	ctx := paintengine2d.NewContext(img)
	DrawToolIcon(ctx, paintengine2d.XYWH(2, 2, 28, 28), icon, paintengine2d.RGB(1, 1, 1), set)
	return img
}

func paintIcon(set IconSetName, icon ToolIcon) int {
	img := rasterIcon(set, icon)
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
