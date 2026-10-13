//go:build font_fallback

package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The two bundled faces differ in what they cover — Titillium Web is
// Latin only, JetBrains Mono has Greek, Cyrillic and the arrows — so a
// fallback test needs nothing installed and says the same thing on
// every machine, in CI, and under UITK_SYSTEM_FONTS=0.
func fallbackRig(t *testing.T) *Font {
	t.Helper()
	SetFallbackFamilies(FamilyMono)
	t.Cleanup(func() { SetFallbackFamilies() })
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if f.covers('Δ') {
		t.Skip("the bundled interface face has grown Greek; this test needs a rune it lacks")
	}
	return f
}

func fallbackInk(f *Font, s string) *paintengine2d.Image {
	img := paintengine2d.NewImage(220, 40)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, s, paintengine2d.Pt(2, 2), paintengine2d.White)
	img.Touch()
	return img
}

func fallbackPixelsDiffer(a, b *paintengine2d.Image) int {
	n := 0
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			n++
		}
	}
	return n
}

// **A rune the face has not got is borrowed from one that has.**
//
// media-player-music item 34: two different CJK song titles drew as the
// same row of .notdef boxes, pixel for pixel, so a playlist showed two
// identical rows for two different songs — with the accessible names
// correctly distinct, which is how it was noticed.
func TestARuneTheFaceLacksIsBorrowed(t *testing.T) {
	f := fallbackRig(t)

	one, two := fallbackInk(f, "Song Δx"), fallbackInk(f, "Song Σx")
	if n := fallbackPixelsDiffer(one, two); n < 20 {
		t.Errorf("two different titles differ in %d pixels: they are still the same row of boxes", n)
	}
	if a, b := f.Advance("Song Δx"), f.Advance("Song Σx"); a <= 0 || b <= 0 {
		t.Errorf("advances %v and %v", a, b)
	}

	// The borrowed glyph is drawn from the lending face's own sheet, so
	// the shaped text is two runs rather than one.
	shaped := f.shapeOf("AΔ")
	if len(shaped.runs) != 2 {
		t.Fatalf("%d runs for one borrowed glyph, want two (the face's own and the lender's)", len(shaped.runs))
	}
	if shaped.runs[0].Atlas == shaped.runs[1].Atlas {
		t.Error("both runs came from the same sheet")
	}
	if len(shaped.runs[0].Glyphs) != 1 || len(shaped.runs[1].Glyphs) != 1 {
		t.Errorf("the glyphs landed %d and %d", len(shaped.runs[0].Glyphs), len(shaped.runs[1].Glyphs))
	}
}

// **A borrowed glyph sits on the line it was borrowed for.**
//
// A glyph's vertical bearing is baked as the sheet's own ascent plus the
// glyph's top, so a lending face with a taller ascent hangs its glyphs
// below the text beside them. The lender is baked against the borrower's
// baseline instead.
func TestABorrowedGlyphSitsOnTheSameBaseline(t *testing.T) {
	f := fallbackRig(t)
	rows := func(s string) (top, bottom int) {
		img := fallbackInk(f, s)
		top, bottom = -1, -1
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				if img.NRGBAAt(x, y).A > 40 {
					if top < 0 {
						top = y
					}
					bottom = y
					break
				}
			}
		}
		return top, bottom
	}
	// The invariant, before the pixels: the lender is baked against the
	// borrower's ascent, not its own. At 16px the two bundled faces are
	// 18.125 and 16.312 apart, which is the 1.8px the glyph would ride
	// up by.
	fb := fallbackFor(f, 'Δ')
	if fb == nil {
		t.Fatal("nothing lent the glyph")
	}
	if fb.Ascent != f.Ascent {
		t.Errorf("the lender's ascent is %.3f and the borrower's %.3f: the glyph is baked to the wrong baseline",
			fb.Ascent, f.Ascent)
	}

	lt, lb := rows("H")
	gt, gb := rows("Δ")
	if lt < 0 || gt < 0 {
		t.Fatalf("nothing was drawn: latin rows %d..%d, borrowed %d..%d", lt, lb, gt, gb)
	}
	// Both are capitals: their tops and their baselines line up within a
	// pixel or two of hinting.
	if d := gb - lb; d > 1 || d < -1 {
		t.Errorf("the borrowed glyph's baseline is %d rows off the face's own (%d..%d against %d..%d)", d, gt, gb, lt, lb)
	}
}

// A rune no listed face has is still a box, and the list is still only
// consulted for runes the face lacks.
func TestWhatNobodyHasIsStillABox(t *testing.T) {
	f := fallbackRig(t)
	if fb := fallbackFor(f, '山'); fb != nil {
		t.Errorf("a Han rune was borrowed from %q, which has no CJK", fb.Family)
	}
	if fb := fallbackFor(f, 'A'); fb != nil {
		t.Error("a rune the face draws itself was borrowed anyway")
	}
}

// The application's list replaces the built-in one, and naming none
// puts the built-in list back.
func TestTheApplicationNamesTheLendingFaces(t *testing.T) {
	SetFallbackFamilies("Acme Nothing", FamilyMono)
	t.Cleanup(func() { SetFallbackFamilies() })
	if got := FallbackFamilies(); len(got) != 2 || got[0] != "Acme Nothing" || got[1] != FamilyMono {
		t.Fatalf("the list reads back %v", got)
	}
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if fb := fallbackFor(f, 'Δ'); fb == nil || fb.Family != FamilyMono {
		t.Error("a family that is not installed was not skipped for the one that is")
	}
	SetFallbackFamilies()
	if len(FallbackFamilies()) != len(defaultFallbackFamilies) {
		t.Error("naming none did not put the built-in list back")
	}
}
