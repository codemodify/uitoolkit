package style

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestAppearanceLookThemeAndCorners(t *testing.T) {
	dark := Appearance{Theme: ThemeDark, Corners: CornersRound, Icons: IconSetClassic}.Look()
	if dark.Name() != "dark" || dark.Corners() != CornersRound || dark.Pack() != "dark" {
		t.Fatalf("dark %+v", LookAppearance(dark))
	}
	if dark.Metrics().Radius < 4 || dark.Metrics().RadiusSmall < 2 {
		t.Fatalf("round radii %+v", dark.Metrics())
	}
	sq := Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp}.Look()
	if sq.Name() != "light" || sq.Corners() != CornersSquare || sq.Icons() != IconSetSharp || sq.Pack() != "light" {
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

func TestWithAppearanceAppliesIconSize(t *testing.T) {
	hi := WithScale(DarkLook(), 2)
	next := WithAppearance(hi, Appearance{Theme: ThemeLight, IconSize: IconSizeLarge})
	c := next.(*Classic)
	if c.IconSize() != IconSizeLarge {
		t.Fatal(c.IconSize())
	}
	if next.Metrics().FontSize != hi.Metrics().FontSize {
		t.Fatal("scale")
	}
}

func TestWithDensityKeepsCornersAndIcons(t *testing.T) {
	look := Appearance{Theme: ThemeDark, Corners: CornersSquare, Icons: IconSetSharp, IconSize: IconSizeLarge}.Look()
	d := WithDensity(look, DensityCompact)
	c := d.(*Classic)
	if c.Corners() != CornersSquare || c.Icons() != IconSetSharp || c.IconSize() != IconSizeLarge {
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

func TestIconDownloadAndPenPaint(t *testing.T) {
	if ToolIconName(IconDownload) != "download" {
		t.Fatalf("download stem %q", ToolIconName(IconDownload))
	}
	if ToolIconName(IconPen) != "pen" {
		t.Fatalf("pen stem %q", ToolIconName(IconPen))
	}
	down := paintIcon(IconSetClassic, IconDownload)
	pen := paintIcon(IconSetClassic, IconPen)
	if down < 8 || pen < 8 {
		t.Fatalf("ink download=%d pen=%d", down, pen)
	}
	if maskDiff(rasterIcon(IconSetClassic, IconDownload), rasterIcon(IconSetClassic, IconPen)) < 8 {
		t.Fatal("download and pen should differ")
	}
	if maskDiff(rasterIcon(IconSetClassic, IconDownload), rasterIcon(IconSetClassic, IconOpen)) < 8 {
		t.Fatal("download should differ from open")
	}
	if maskDiff(rasterIcon(IconSetClassic, IconPen), rasterIcon(IconSetClassic, IconNew)) < 8 {
		t.Fatal("pen should differ from new")
	}
	for _, name := range toolIconFileCandidates(IconPen, 24) {
		if name == "new.png" || name == "new@2x.png" || name == "edit.png" {
			t.Fatalf("pen must not fall back to %s: %v", name, toolIconFileCandidates(IconPen, 24))
		}
	}
}

func TestIconMailPaintsAndThemeName(t *testing.T) {
	if ToolIconName(IconMail) != "mail" {
		t.Fatalf("file stem %q", ToolIconName(IconMail))
	}
	if ToolIconThemeName(IconMail) != "mail-unread" {
		t.Fatalf("theme %q", ToolIconThemeName(IconMail))
	}
	mail := paintIcon(IconSetClassic, IconMail)
	info := paintIcon(IconSetClassic, IconInfo)
	if mail < 8 || info < 8 {
		t.Fatalf("ink mail=%d info=%d", mail, info)
	}
	if maskDiff(rasterIcon(IconSetClassic, IconMail), rasterIcon(IconSetClassic, IconInfo)) < 8 {
		t.Fatal("mail glyph should differ from info")
	}
	cands := toolIconFileCandidates(IconMail, 24)
	if len(cands) < 4 || cands[0] != "mail.png" {
		t.Fatalf("candidates %v", cands)
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

func TestParseIconSize(t *testing.T) {
	if ParseIconSize("") != IconSizeMedium || ParseIconSize("24") != IconSizeMedium {
		t.Fatal("default")
	}
	if ParseIconSize("small") != IconSizeSmall || ParseIconSize("16") != IconSizeSmall {
		t.Fatal("small")
	}
	if ParseIconSize("LARGE") != IconSizeLarge || ParseIconSize("32") != IconSizeLarge {
		t.Fatal("large")
	}
	if IconSizePixels(IconSizeSmall) != 16 || IconSizePixels(IconSizeMedium) != 24 || IconSizePixels(IconSizeLarge) != 32 {
		t.Fatal("pixels")
	}
}

func TestAppearanceNormalizesIconSize(t *testing.T) {
	got := Appearance{}.Normalize()
	if got.IconSize != IconSizeMedium {
		t.Fatalf("%+v", got)
	}
	if (Appearance{IconSize: "16"}).Normalize().IconSize != IconSizeSmall {
		t.Fatal("16")
	}
}

func TestWithIconSizeGrowsToolbar(t *testing.T) {
	med := DarkLook()
	lg := WithIconSize(med, IconSizeLarge).(*Classic)
	if lg.IconSize() != IconSizeLarge {
		t.Fatal(lg.IconSize())
	}
	if lg.Metrics().ToolBarH <= med.Metrics().ToolBarH {
		t.Fatalf("large bar %v vs medium %v", lg.Metrics().ToolBarH, med.Metrics().ToolBarH)
	}
	_, sideM, _ := ToolButtonChromeFor(med, med.Metrics().ToolBtn)
	_, sideL, _ := ToolButtonChromeFor(lg, lg.Metrics().ToolBtn)
	if sideL <= sideM {
		t.Fatalf("icon side large %v medium %v", sideL, sideM)
	}
	sm := WithIconSize(med, IconSizeSmall).(*Classic)
	_, sideS, _ := ToolButtonChromeFor(sm, sm.Metrics().ToolBtn)
	if sideS >= sideM {
		t.Fatalf("small %v >= medium %v", sideS, sideM)
	}
}

func TestAppearancePrefsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := Appearance{Theme: ThemeLight, Corners: CornersSquare, Icons: IconSetSharp, IconSize: IconSizeLarge}
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
	if look.Name() != "light" || look.Corners() != CornersSquare || look.Icons() != IconSetSharp || look.IconSize() != IconSizeLarge {
		t.Fatalf("preferred %+v", LookAppearance(look))
	}
}

func TestLoadAppearanceMissingIsDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if LoadAppearance() != DefaultAppearance() {
		t.Fatal(LoadAppearance())
	}
}

// The default theme is a built-in pack, and the default appearance names
// the pack's own family.
func TestDefaultThemeIsABuiltinPack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme(DefaultThemeName)
	if !ok || p.Source != ThemeSourceBuiltin {
		t.Fatalf("default theme %q is not built in", DefaultThemeName)
	}
	if a := DefaultAppearance(); a.Theme != p.Palette {
		t.Fatalf("default appearance family %s, the pack's %s", a.Theme, p.Palette)
	}
	if got := PreferredLook().(*Classic).Pack(); got != DefaultThemeName {
		t.Fatalf("with no look.json the look is %q", got)
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

func TestDecorationsPrefRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(ThemeEnv, "")
	for in, want := range map[string]DecorationsPref{
		"system": DecorationsSystem, "toolkit": DecorationsToolkit, "auto": DecorationsAuto,
		"": DecorationsAuto, "server": DecorationsSystem, "client": DecorationsToolkit, "bogus": DecorationsAuto,
	} {
		if got := ParseDecorationsPref(in); got != want {
			t.Errorf("ParseDecorationsPref(%q) = %q want %q", in, got, want)
		}
	}
	ap := DefaultAppearance()
	if ap.Decorations != DecorationsAuto {
		t.Fatalf("default %q", ap.Decorations)
	}
	ap.Decorations = DecorationsSystem
	if err := SaveAppearance(ap); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"decorations": "system"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	if got := LoadAppearance(); got.Decorations != DecorationsSystem {
		t.Fatalf("loaded %q", got.Decorations)
	}
	ap.Decorations = DecorationsAuto
	if err := SaveAppearance(ap); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(AppearancePath())
	if strings.Contains(string(raw), "decorations") {
		t.Fatalf("auto is left out of look.json: %s", raw)
	}
}
