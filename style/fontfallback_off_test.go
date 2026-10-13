//go:build !font_fallback && !font_complex

package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// **The default build borrows nothing, and that is the contract.**
//
// One face draws everything and a rune it lacks is a box: `contracts.md`
// says so, every screenshot in the repository is taken that way, and an
// application that wants otherwise asks for it at build time with
// `-tags font_fallback`. This is the half of the pair that pins the
// default; `fontfallback_on_test.go` pins the other.
func TestTheDefaultBuildBorrowsNothing(t *testing.T) {
	// Even with the lending face named, and named as one that is
	// bundled so no machine's fonts come into it.
	SetFallbackFamilies(FamilyMono)
	t.Cleanup(func() { SetFallbackFamilies() })

	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if f.covers('Δ') {
		t.Skip("the bundled interface face has grown Greek; this test needs a rune it lacks")
	}
	if fb := fallbackFor(f, 'Δ'); fb != nil {
		t.Fatalf("the default build borrowed from %q", fb.Family)
	}

	ink := func(s string) *paintengine2d.Image {
		img := paintengine2d.NewImage(220, 40)
		ctx := paintengine2d.NewContext(img)
		f.Draw(ctx, s, paintengine2d.Pt(2, 2), paintengine2d.White)
		img.Touch()
		return img
	}
	one, two := ink("Song Δx"), ink("Song Σx")
	same := true
	for i := range one.Pix {
		if one.Pix[i] != two.Pix[i] {
			same = false
			break
		}
	}
	if !same {
		t.Error("two runes the face lacks drew differently; the default build has grown a fallback")
	}
	// One face, one sheet, one run — which is what keeps the layout
	// pass as cheap as it was.
	if shaped := f.shapeOf("AΔ"); len(shaped.runs) != 1 {
		t.Errorf("%d runs in a build that borrows nothing", len(shaped.runs))
	}
}
