//go:build font_fallback || font_complex || font_emoji

package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The case that was reported, against the machine's own fonts: two
// different CJK song titles must not draw as the same row of boxes.
// Skipped where no CJK family is installed, which is most containers.
func TestTwoCJKTitlesDrawDifferently(t *testing.T) {
	if ResolveFont(defaultFallbackFamilies) == "" {
		t.Skip("no fallback family installed")
	}
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if fallbackFor(f, '山') == nil {
		t.Skip("no installed face covers Han")
	}
	ink := func(s string) *paintengine2d.Image {
		img := paintengine2d.NewImage(220, 40)
		ctx := paintengine2d.NewContext(img)
		f.Draw(ctx, s, paintengine2d.Pt(2, 2), paintengine2d.White)
		img.Touch()
		return img
	}
	a, b := ink("Song 山川"), ink("Song 海河")
	n := 0
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			n++
		}
	}
	if n < 50 {
		t.Errorf("the two titles differ in %d pixels", n)
	}
	t.Logf("the two titles differ in %d pixels", n)
}
