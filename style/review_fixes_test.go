package style

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// --- glyph sheet concurrency -------------------------------------------------

// TestGlyphAtlasConcurrentMeasureAndPaint is the regression for the shared
// atlas data race: a paint loop blitting one face while another goroutine
// measures runes that are not baked yet. Run under -race.
func TestGlyphAtlasConcurrentMeasureAndPaint(t *testing.T) {
	paint := BakeFont(19, paintengine2d.White)
	measure := BakeFont(19, paintengine2d.Black) // same cached sheet
	if paint.GlyphAtlas() != measure.GlyphAtlas() {
		t.Fatal("same size should share one sheet")
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() { // paint loop
		defer wg.Done()
		img := paintengine2d.NewImage(320, 48)
		ctx := paintengine2d.NewContext(img)
		for {
			select {
			case <-stop:
				return
			default:
			}
			paint.Draw(ctx, "Inbox — 12 unread", paintengine2d.Pt(4, 18), paintengine2d.White)
			_ = paint.Advance("Inbox — 12 unread")
			_ = paint.InkWidth("Inbox")
		}
	}()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(seed rune) { // measure new runes
			defer wg.Done()
			for r := seed; r < seed+60; r++ {
				_ = measure.Advance(string(r))
				_ = measure.runeAdvance(r)
				_ = measure.Fit(string([]rune{r, r, r}), 12)
			}
		}(rune(0x100 + i*200))
	}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			img := paintengine2d.NewImage(200, 40)
			ctx := paintengine2d.NewContext(img)
			for j := 0; j < 50; j++ {
				measure.Draw(ctx, "Ωμλ ταβ", paintengine2d.Pt(2, 20), paintengine2d.White)
			}
		}()
	}
	close(stop)
	wg.Wait()
	if _, ok := paint.GlyphAtlas().Cell(paintengine2d.GlyphID('A')); !ok {
		t.Fatal("published sheet lost its preload")
	}
}

// --- glyph snapping ----------------------------------------------------------

// TestGlyphSnappingKeepsStemsCrisp pins finding 4: an unsnapped fractional
// origin used to resample the sheet (peak alpha ~191 over three columns).
func TestGlyphSnappingKeepsStemsCrisp(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	draw := func(ox float32) (peak uint8, cols int) {
		img := paintengine2d.NewImage(40, 40)
		ctx := paintengine2d.NewContext(img)
		f.Draw(ctx, "l", paintengine2d.Pt(ox, 10), paintengine2d.White)
		for x := 0; x < img.Width; x++ {
			ink := false
			for y := 0; y < img.Height; y++ {
				_, _, _, a := img.PremulAt(x, y)
				if a > peak {
					peak = a
				}
				if a > 10 {
					ink = true
				}
			}
			if ink {
				cols++
			}
		}
		return peak, cols
	}
	peak, cols := draw(10)
	if peak != 255 {
		t.Fatalf("integer origin peak alpha %d, want a 1:1 blit (255)", peak)
	}
	fpeak, fcols := draw(10.4)
	if fpeak != peak || fcols != cols {
		t.Fatalf("fractional origin should snap: peak %d/%d cols %d/%d", fpeak, peak, fcols, cols)
	}
	if fcols > 2 {
		t.Fatalf("a 16px stem smeared over %d columns", fcols)
	}
}

// TestSnapRunKeepsPositionsWithinHalfPixel guards accumulated drift.
func TestSnapRunKeepsPositionsWithinHalfPixel(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	const text = "The quick brown fox jumps over the lazy dog"
	atlas := f.ensure(text)
	raw := paintengine2d.NullShaper{}.Shape(text, atlas)
	snapped := snapRun(raw, atlas)
	if len(raw.Glyphs) != len(snapped.Glyphs) {
		t.Fatalf("glyph count %d vs %d", len(raw.Glyphs), len(snapped.Glyphs))
	}
	for i := range raw.Glyphs {
		cell, _ := atlas.Cell(raw.Glyphs[i].ID)
		d := snapped.Glyphs[i].X - raw.Glyphs[i].X
		if d > 0.5 || d < -0.5 {
			t.Fatalf("glyph %d moved %v", i, d)
		}
		dev := snapped.Glyphs[i].X + cell.Bearing.X
		if off := dev - float32(math.Round(float64(dev))); off > 1e-3 || off < -1e-3 {
			t.Fatalf("glyph %d lands at %v, not a pixel", i, dev)
		}
	}
}

// --- .notdef ----------------------------------------------------------------

// TestMissingRuneGetsNotdef pins finding 5: unsupported runes used to
// measure 0 and paint nothing, and were re-looked-up on every call.
func TestMissingRuneGetsNotdef(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	const cjk = "日本語"
	adv := f.Advance(cjk)
	if adv <= 0 {
		t.Fatalf("unsupported runes must still measure, got %v", adv)
	}
	if per := adv / 3; per < f.Size*0.2 {
		t.Fatalf("notdef advance %v is too small for size %v", per, f.Size)
	}
	cell, ok := f.GlyphAtlas().Cell(paintengine2d.GlyphID('日'))
	if !ok {
		t.Fatal("notdef cell must be cached, not re-resolved on every measure")
	}
	if cell.Src.Empty() {
		t.Fatal("notdef should have pixels (a box), not an empty cell")
	}
	img := paintengine2d.NewImage(80, 32)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, cjk, paintengine2d.Pt(4, 6), paintengine2d.White)
	ink := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 20 {
				ink++
			}
		}
	}
	if ink < 12 {
		t.Fatalf("notdef boxes should paint, ink=%d", ink)
	}
	if got := f.Fit(cjk, adv/2); got == cjk {
		t.Fatalf("Fit must be able to truncate unsupported text, got %q", got)
	}
}

// --- font cache / atlas budget ----------------------------------------------

// TestFontCacheEvictsPastBudget pins finding 6: the size cache never evicted.
func TestFontCacheEvictsPastBudget(t *testing.T) {
	old := fontCacheBudget
	// Sheets are one byte a pixel and grow to what they hold (about
	// 32–128 KB at these sizes), so 256 KB keeps only a few.
	fontCacheBudget = 256 << 10
	defer func() { fontCacheBudget = old }()
	for i := 0; i < 24; i++ {
		BakeFont(12+float32(i)*0.5, paintengine2d.White)
	}
	entries, bytes := fontCacheStats()
	if bytes > fontCacheBudget+(2<<20) {
		t.Fatalf("cache holds %d bytes, budget %d", bytes, fontCacheBudget)
	}
	if entries > 8 {
		t.Fatalf("expected eviction, %d entries still cached", entries)
	}
	// An evicted face keeps working for whoever still holds it.
	f := BakeFont(12, paintengine2d.White)
	if f.Advance("OK") <= 0 {
		t.Fatal("evicted-then-rebaked face must still measure")
	}
}

// TestAtlasGrowthIsCappedAndStillMeasures pins the 151 MB growth path.
func TestAtlasGrowthIsCappedAndStillMeasures(t *testing.T) {
	face, err := faceFor(FamilyUI, WeightRegular)
	if err != nil {
		t.Fatal(err)
	}
	a := newOTAtlas(face, 16)
	if _, err := a.bake(otPreload); err != nil {
		t.Fatal(err)
	}
	d := a.draft(a.atlas())
	for d.grow() {
		if d.img.Width > atlasMaxSide || d.img.Height > atlasMaxSide {
			t.Fatalf("grew past the cap: %dx%d", d.img.Width, d.img.Height)
		}
	}
	if d.img.Width != atlasMaxSide || d.img.Height != atlasMaxSide {
		t.Fatalf("growth should stop at the cap, got %dx%d", d.img.Width, d.img.Height)
	}
	// A full sheet keeps advances so layout is still correct.
	d.packer.x, d.packer.y, d.packer.rowH = atlasMaxSide, atlasMaxSide, 0
	cell, err := d.bakeCell(notdefBoxPath(16), 9)
	if err != nil {
		t.Fatal(err)
	}
	if cell.Advance != 9 {
		t.Fatalf("full sheet must keep the advance, got %v", cell.Advance)
	}
	if !d.full {
		t.Fatal("draft should be marked full")
	}
	if a.atlasBytes() <= 0 {
		t.Fatal("atlas byte accounting")
	}
}

func notdefBoxPath(size float32) *paintengine2d.Path {
	p, _ := notdefBox(size)
	return p
}

// --- theme name safety / canonicalization ------------------------------------

// TestLoadThemeRejectsPathTraversal pins finding 2.
func TestLoadThemeRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "cfg")
	t.Setenv("XDG_CONFIG_HOME", cfg)
	outside := filepath.Join(root, "outside", "evil")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"label":"PWNED","palette":"light","colors":{"accent":"#ff0000"}}`)
	if err := os.WriteFile(filepath.Join(outside, "theme.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cfg, "uitoolkit"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"../../../outside/evil",
		"..",
		"../outside/evil",
		filepath.Join(root, "outside", "evil"),
		"themes/../../outside/evil",
	} {
		if pack, ok := LoadTheme(name); ok {
			t.Fatalf("LoadTheme(%q) escaped the themes dir: %+v", name, pack)
		}
	}
	look := []byte(`{"theme":"../../../outside/evil"}`)
	if err := os.WriteFile(filepath.Join(cfg, "uitoolkit", "look.json"), look, 0o644); err != nil {
		t.Fatal(err)
	}
	a := LoadAppearance()
	if strings.Contains(a.Name, "..") || strings.Contains(a.Name, "/") {
		t.Fatalf("appearance kept a traversal name: %q", a.Name)
	}
	if a.Name != DefaultThemeName {
		t.Fatalf("unusable theme name should fall back to %q, got %q", DefaultThemeName, a.Name)
	}
}

// TestUserThemeDirCanonicalization pins finding 7 for themes: a folder whose
// name is not canonical listed but never loaded.
func TestUserThemeDirCanonicalization(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "uitoolkit", "themes", "MyTheme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"palette":"light","colors":{"accent":"#ff0000"}}`)
	if err := os.WriteFile(filepath.Join(dir, "theme.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	listed := ListUserThemes()
	if len(listed) != 1 || listed[0].Name != "mytheme" {
		t.Fatalf("listing should canonicalize: %+v", listed)
	}
	for _, name := range []string{"MyTheme", "mytheme", " MyTheme "} {
		if _, ok := LoadTheme(name); !ok {
			t.Fatalf("LoadTheme(%q) must find the folder the lister showed", name)
		}
	}
	if err := SaveAppearance(Appearance{Name: "MyTheme", Theme: ThemeLight}); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if got.Name != "mytheme" {
		t.Fatalf("round trip lost the pack: %q", got.Name)
	}
	if ThemeSourceFile("MyTheme") != filepath.Join(ThemesDir(), "MyTheme", "theme.json") {
		t.Fatalf("ThemeSourceFile %q", ThemeSourceFile("MyTheme"))
	}
	if ThemeSourceFile("dark") != "" {
		t.Fatal("builtin packs have no source file")
	}
	if err := DeleteUserTheme("MyTheme"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(ListUserThemes()) != 0 {
		t.Fatal("delete should remove the folder")
	}
}

// --- metric rebuild ----------------------------------------------------------

// TestWithAppearanceRebuildsMetrics pins finding 3: pack metrics leaked into
// the next pack on a live theme switch.
func TestWithAppearanceRebuildsMetrics(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "uitoolkit", "themes", "wide")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"palette":"dark","metrics":{"comboH":44,"controlH":50,"scroll":24}}`)
	if err := os.WriteFile(filepath.Join(dir, "theme.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	var lk LookAndFeel = DarkLook()
	lk = WithAppearance(lk, Appearance{Name: "wide"})
	if lk.Metrics().ComboH != 44 || lk.Metrics().ControlH != 50 || lk.Metrics().Scroll != 24 {
		t.Fatalf("pack metrics not applied: %+v", lk.Metrics())
	}
	for _, next := range []string{"breeze", "dark", "fusion", "light"} {
		got := WithAppearance(lk, Appearance{Name: next}).Metrics()
		want := Appearance{Name: next}.Look().Metrics()
		if got.ComboH != want.ComboH || got.ControlH != want.ControlH || got.Scroll != want.Scroll || got.Radius != want.Radius {
			t.Fatalf("%s inherited previous pack: got combo=%v control=%v scroll=%v radius=%v want %v/%v/%v/%v",
				next, got.ComboH, got.ControlH, got.Scroll, got.Radius,
				want.ComboH, want.ControlH, want.Scroll, want.Radius)
		}
	}
}

// TestWithAppearanceKeepsDensityAndScale pins the second half of finding 3:
// the scale factor used to be re-derived from FontSize.
func TestWithAppearanceKeepsDensityAndScale(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	base := WithScale(WithDensity(DarkLook(), DensityCompact), 2)
	c, ok := base.(*Classic)
	if !ok || c.Density() != DensityCompact || c.Scale() != 2 {
		t.Fatalf("base density=%v scale=%v", c.Density(), c.Scale())
	}
	got := WithAppearance(base, Appearance{Name: "motif"})
	want := WithScale(WithDensity(Appearance{Name: "motif"}.Look(), DensityCompact), 2)
	if got.Metrics() != want.Metrics() {
		t.Fatalf("compact@2x reload:\n got %+v\nwant %+v", got.Metrics(), want.Metrics())
	}
	gc := got.(*Classic)
	if gc.Density() != DensityCompact || gc.Scale() != 2 {
		t.Fatalf("lost density/scale: %v %v", gc.Density(), gc.Scale())
	}
}

// --- theme value hygiene -----------------------------------------------------

// TestThemeMetricsAreClamped pins finding 8.
func TestThemeMetricsAreClamped(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "uitoolkit", "themes", "big")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"palette":"dark","metrics":{"scroll":1e9,"controlH":1e9,"radius":1e9,"comboH":-5,"bevelDepth":99},"elevation":2147483647}`)
	if err := os.WriteFile(filepath.Join(dir, "theme.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	pack, ok := LoadTheme("big")
	if !ok {
		t.Fatal("pack should still load")
	}
	cm := pack.Tokens.Metrics
	if cm.Scroll != maxThemeScroll || cm.ControlH != maxControlSide || cm.Radius != maxThemeRadius {
		t.Fatalf("not clamped: %+v", cm)
	}
	if cm.ComboH != 0 {
		t.Fatalf("negative comboH should be unset, got %v", cm.ComboH)
	}
	if cm.BevelDepth != maxBevelDepth || cm.Elevation != maxElevation {
		t.Fatalf("bevel/elevation: %+v", cm)
	}
	m := pack.Appearance().Look().Metrics()
	if m.Scroll > maxThemeScroll || m.ControlH > maxControlSide || m.Radius > maxThemeRadius {
		t.Fatalf("look metrics unbounded: %+v", m)
	}
	// Programmatic tokens are clamped by Resolve too.
	tok := ThemeTokens{Metrics: ChromeMetrics{Radius: 5000, Elevation: -3}}.Resolve()
	if tok.Metrics.Radius != maxThemeRadius || tok.Metrics.Elevation != 0 {
		t.Fatalf("Resolve clamp: %+v", tok.Metrics)
	}
}

// TestExplicitTransparentColour pins finding 9.
func TestExplicitTransparentColour(t *testing.T) {
	raw := []byte(`{"palette":"dark","colors":{"shadow":"#00000000","selection":"#0000","menuGutter":"#00000000"}}`)
	pack, err := parseThemeFile("clear", raw, ThemeSourceUser)
	if err != nil {
		t.Fatal(err)
	}
	p := pack.Tokens.Palette
	for name, c := range map[string]paintengine2d.Color{
		"shadow":     p.Shadow,
		"selection":  p.Selection,
		"menuGutter": p.MenuGutter,
	} {
		if c.A > 0.01 {
			t.Fatalf("%s should stay transparent, got %+v", name, c)
		}
		if r, g, b, a := c.Premul8(); r|g|b|a != 0 {
			t.Fatalf("%s should quantise to nothing, got %d %d %d %d", name, r, g, b, a)
		}
	}
	// It must survive an export round trip.
	if hex := colorHexPadded(p.Shadow); hex != "#00000000" {
		t.Fatalf("export %q", hex)
	}
	back, ok := ParseHexColor("#00000000")
	if !ok || !colorUnset(back) {
		t.Fatal("parse of transparent black")
	}
	if colorUnset(markExplicitColor(back)) {
		t.Fatal("explicit transparency must not read as unset")
	}
}

// TestResolveBevelChromeUsesFamily pins finding 13 (ParseTheme("") was
// constant-true, so a light pack was treated as dark).
func TestResolveBevelChromeUsesFamily(t *testing.T) {
	p := Light()
	p.Text = paintengine2d.RGB(0.9, 0.9, 0.9) // pale text on a light pack
	p.Highlight = paintengine2d.Color{}
	p.BevelLight = paintengine2d.Color{}
	p.SurfaceAlt = paintengine2d.RGB(0.2, 0.2, 0.2)
	dark := ResolveBevelChromeFor(p, ThemeDark)
	light := ResolveBevelChromeFor(p, ThemeLight)
	if dark.BevelLight == light.BevelLight {
		t.Fatal("family must change the derived bevel highlight")
	}
	if light.BevelLight != paintengine2d.RGB(1, 1, 1) {
		t.Fatalf("light family should use a white highlight, got %+v", light.BevelLight)
	}
}

// --- icons -------------------------------------------------------------------

// TestIconSetDirCanonicalization pins finding 7 for icon sets.
func TestIconSetDirCanonicalization(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	resetIconCache()
	dir := filepath.Join(cfg, "uitoolkit", "icons", "My Icons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMaskPNG(t, filepath.Join(dir, "open.png"), 24)
	resetIconCache()
	if got := IconSetDir("my-icons"); got != dir {
		t.Fatalf("IconSetDir %q want %q", got, dir)
	}
	if !fileIconSetInstalled("my-icons") {
		t.Fatal("set listed from that folder must count as installed")
	}
	sets := ListIconSets()
	found := false
	for _, s := range sets {
		if s.Name == "my-icons" {
			found = true
		}
	}
	if !found {
		t.Fatalf("not listed: %+v", sets)
	}
	if _, ok := loadPNGIcon(filepath.Join(IconSetDir("my-icons"), "open.png")); !ok {
		t.Fatal("the PNG in the folder must load")
	}
}

// TestIconHiDPIPicksNativeAsset pins finding 14.
func TestIconHiDPIPicksNativeAsset(t *testing.T) {
	if got := stemFileCandidates("open", 24); got[0] != "open.png" {
		t.Fatalf("24px should use the 1× asset: %v", got)
	}
	if got := stemFileCandidates("open", 32); got[0] != "open@2x.png" {
		t.Fatalf("32px (IconSizeLarge) must not upscale the 24px asset: %v", got)
	}
	if got := stemFileCandidates("open", 48); got[0] != "open@2x.png" {
		t.Fatalf("48px: %v", got)
	}
	if got := toolIconFileCandidates(IconOpen, 32); got[0] != "open@2x.png" {
		t.Fatalf("tool icon candidates: %v", got)
	}
}

// TestIconCacheGenerationInvalidates pins finding 11: misses are cached (no
// stat per draw) and a generation bump re-reads.
func TestIconCacheGenerationInvalidates(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	resetIconCache()
	dir := filepath.Join(cfg, "uitoolkit", "icons", "lucide")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "open.png")
	if _, ok := loadPNGIcon(path); ok {
		t.Fatal("missing file should not load")
	}
	gen := IconGeneration()
	// A negative entry is cached: creating the file is not seen until the
	// cache is invalidated (or the TTL lapses).
	writeMaskPNG(t, path, 24)
	if _, ok := loadPNGIcon(path); ok {
		t.Fatal("negative entry should be reused within the generation")
	}
	InvalidateIconCache()
	if IconGeneration() == gen {
		t.Fatal("generation must advance")
	}
	if _, ok := loadPNGIcon(path); !ok {
		t.Fatal("invalidated cache must re-read the file")
	}
	// A decoded icon is cached, so deleting it keeps painting this frame.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadPNGIcon(path); !ok {
		t.Fatal("cached image should survive until invalidation")
	}
	InvalidateIconCache()
	if _, ok := loadPNGIcon(path); ok {
		t.Fatal("after invalidation the deleted file must miss")
	}
}

func writeMaskPNG(t *testing.T, path string, side int) {
	t.Helper()
	img := paintengine2d.NewImage(side, side)
	for y := 2; y < side-2; y++ {
		for x := 2; x < side-2; x++ {
			img.SetColor(x, y, paintengine2d.White)
		}
	}
	if err := img.WritePNGFile(path); err != nil {
		t.Fatal(err)
	}
}
