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
	_ = GlyphTint()
}
