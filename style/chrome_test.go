package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestLoadThemeEveryEmbeddedPack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	names := AllBuiltinThemeNames()
	if len(names) < 18 {
		t.Fatalf("expected era packs, got %v", names)
	}
	for _, name := range names {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("LoadTheme(%s) missing", name)
		}
		if pack.Tokens.Bevel == "" {
			t.Fatalf("%s empty bevel", name)
		}
		look := pack.Look()
		if look == nil {
			t.Fatalf("%s look nil", name)
		}
		if look.Pack() != name {
			t.Fatalf("%s pack %q", name, look.Pack())
		}
		// Paint must not panic.
		img := paintengine2d.NewImage(120, 48)
		ctx := paintengine2d.NewContext(img)
		look.DrawButton(ctx, paintengine2d.XYWH(4, 4, 80, 28), StateHovered, "OK")
		look.DrawMenuItem(ctx, paintengine2d.XYWH(4, 8, 110, 24), StateHovered, MenuRow{Label: "Open"})
	}
	if _, ok := LoadTheme("classic95"); !ok {
		t.Fatal("classic95 alias")
	}
	if _, ok := LoadTheme("luna-dark"); !ok {
		t.Fatal("luna-dark alias")
	}
}

func TestThemeChromeDistinct(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	samples := []string{"dark", "light", "motif", "cde", "next", "luna", "aqua", "fusion", "breeze", "fluent", "material", "flatlaf"}
	hot := map[string][]byte{}
	idle := map[string][]byte{}
	for _, name := range samples {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		look := pack.Look()
		idle[name] = rasterButton(look, StateNone)
		hot[name] = rasterButton(look, StateHovered)
		if bytesEqual(idle[name], hot[name]) {
			t.Fatalf("%s button idle and hot paint identically", name)
		}
		mh := rasterMenu(look, StateHovered)
		mi := rasterMenu(look, StateNone)
		if bytesEqual(mh, mi) {
			t.Fatalf("%s menu idle and hot paint identically", name)
		}
	}
	same := 0
	for i, a := range samples {
		for _, b := range samples[i+1:] {
			if bytesEqual(hot[a], hot[b]) {
				same++
				t.Errorf("hot button %s == %s", a, b)
			}
		}
	}
	if same > 0 {
		t.Fatalf("%d theme pairs collapsed to the same hot button", same)
	}
}

func TestLunaHotTrackLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pack, ok := LoadTheme("luna")
	if !ok {
		t.Fatal("luna")
	}
	if pack.Tokens.Bevel != BevelLunaHottrack {
		t.Fatalf("bevel %s", pack.Tokens.Bevel)
	}
	look := pack.Look()
	img := paintengine2d.NewImage(100, 36)
	ctx := paintengine2d.NewContext(img)
	look.DrawButton(ctx, paintengine2d.XYWH(8, 4, 84, 28), StateHovered, "OK")
	p := look.Palette()
	if countNear(img, p.MenuHover) < 20 {
		t.Fatalf("luna hot button lacks pale fill (near=%d)", countNear(img, p.MenuHover))
	}
	if countNear(img, p.MenuHoverBorder) < 8 {
		t.Fatalf("luna hot button lacks border (near=%d)", countNear(img, p.MenuHoverBorder))
	}
}

func TestClassic3DBevelLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"light", "motif", "cde"} {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if pack.Tokens.Bevel != BevelClassic3D {
			t.Fatalf("%s bevel %s", name, pack.Tokens.Bevel)
		}
		look := pack.Look()
		img := paintengine2d.NewImage(100, 36)
		ctx := paintengine2d.NewContext(img)
		b := paintengine2d.XYWH(8, 4, 84, 28)
		look.DrawButton(ctx, b, StateNone, "OK")
		p := look.Palette()
		top := countNearIn(img, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), 3), p.BevelLight)
		bot := countNearIn(img, paintengine2d.XYWH(b.Min.X, b.Max.Y-3, b.Dx(), 3), p.BevelDark)
		if top < 4 || bot < 4 {
			t.Fatalf("%s missing 3D bevel top=%d bot=%d", name, top, bot)
		}
	}
}

func TestNeXTChromeContrast(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pack, ok := LoadTheme("next")
	if !ok {
		t.Fatal("next")
	}
	p := pack.Look().Palette()
	chrome := p.Surface.R + p.Surface.G + p.Surface.B
	field := p.Field.R + p.Field.G + p.Field.B
	if field-chrome < 0.8 {
		t.Fatalf("NeXT should keep dark chrome / light content (chrome=%v field=%v)", chrome, field)
	}
}

func TestBevelStylesDifferAcrossEras(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := map[string]BevelStyle{
		"dark":     BevelClassic3D,
		"motif":    BevelClassic3D,
		"luna":     BevelLunaHottrack,
		"aqua":     BevelSoftShadow,
		"breeze":   BevelNone,
		"fluent":   BevelFluentAccent,
		"material": BevelSoftShadow,
		"flatlaf":  BevelNone,
	}
	seen := map[BevelStyle]int{}
	for name, bevel := range want {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if pack.Tokens.Bevel != bevel {
			t.Fatalf("%s bevel %s want %s", name, pack.Tokens.Bevel, bevel)
		}
		seen[bevel]++
	}
	if len(seen) < 4 {
		t.Fatalf("expected multiple bevel languages, got %v", seen)
	}
}

func rasterButton(look LookAndFeel, st ControlState) []byte {
	img := paintengine2d.NewImage(96, 36)
	ctx := paintengine2d.NewContext(img)
	look.DrawButton(ctx, paintengine2d.XYWH(4, 4, 88, 28), st, "OK")
	return imgBytes(img)
}

func rasterMenu(look LookAndFeel, st ControlState) []byte {
	img := paintengine2d.NewImage(140, 32)
	ctx := paintengine2d.NewContext(img)
	look.DrawMenuItem(ctx, paintengine2d.XYWH(2, 2, 136, 28), st, MenuRow{Label: "Open"})
	return imgBytes(img)
}

func imgBytes(img *paintengine2d.Image) []byte {
	out := make([]byte, img.Width*img.Height*4)
	i := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, a := img.PremulAt(x, y)
			out[i] = r
			out[i+1] = g
			out[i+2] = b
			out[i+3] = a
			i += 4
		}
	}
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countNear(img *paintengine2d.Image, c paintengine2d.Color) int {
	return countNearIn(img, paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height)), c)
}

func countNearIn(img *paintengine2d.Image, r paintengine2d.Rect, c paintengine2d.Color) int {
	n := 0
	tr, tg, tb := int(c.R*255+0.5), int(c.G*255+0.5), int(c.B*255+0.5)
	x0, y0 := int(r.Min.X), int(r.Min.Y)
	x1, y1 := int(r.Max.X), int(r.Max.Y)
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > img.Width {
		x1 = img.Width
	}
	if y1 > img.Height {
		y1 = img.Height
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			pr, pg, pb, pa := img.PremulAt(x, y)
			if pa < 20 {
				continue
			}
			dr, dg, db := int(pr)-tr, int(pg)-tg, int(pb)-tb
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if dr+dg+db < 80 {
				n++
			}
		}
	}
	return n
}
