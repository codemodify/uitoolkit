package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestBakeFontDrawsKnownGlyphs(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	for _, r := range []rune{'A', 'a', '0', ',', ' '} {
		if _, ok := f.Atlas.Cell(paintengine2d.GlyphID(r)); !ok {
			t.Fatalf("missing %q", string(r))
		}
	}
	if f.Advance("OK") < 8 {
		t.Fatalf("advance %v", f.Advance("OK"))
	}
	img := paintengine2d.NewImage(80, 24)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, "OK", paintengine2d.Pt(2, 4), paintengine2d.White)
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a > 20 {
				n++
			}
		}
	}
	if n < 10 {
		t.Fatalf("expected glyph pixels, n=%d", n)
	}
	if f.Height() <= f.Ascent {
		t.Fatalf("height %v ascent %v", f.Height(), f.Ascent)
	}
	if !GlyphTint() {
		t.Fatal("paintengine2d v0.7.2+ blit RGB tint is required")
	}
	if !f.Outline || f.Family != FamilyUI {
		t.Fatalf("BakeFont must be Titillium outline, got family=%q outline=%v", f.Family, f.Outline)
	}
}

func TestBakeFontTintAppliesThemeColor(t *testing.T) {
	red := BakeFont(16, paintengine2d.RGB(1, 0, 0))
	blue := BakeFont(16, paintengine2d.RGB(0.2, 0.5, 1))
	if red.Atlas != blue.Atlas {
		t.Fatal("same size should share the white atlas")
	}
	img := paintengine2d.NewImage(80, 24)
	ctx := paintengine2d.NewContext(img)
	red.Draw(ctx, "OK", paintengine2d.Pt(2, 4), paintengine2d.RGB(1, 0, 0))
	found := false
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a > 20 && int(r) > int(g)+24 && int(r) > int(b)+24 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected red-tinted glyph pixels from white atlas")
	}
}
