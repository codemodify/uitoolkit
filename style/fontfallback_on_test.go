//go:build font_fallback || font_complex || font_emoji

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

// **Changing the families changes what is drawn, at once.**
//
// Both the remembered decisions and the laid-out runs were kept across
// a change to the list, so a rune cached as "nobody has it" stayed a
// box after a face that has it was named, and one already borrowed went
// on being borrowed from a face no longer in the list.
// media-player-video and media-player-music found the two directions
// independently.
func TestChangingTheFamiliesTakesEffectAtOnce(t *testing.T) {
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if f.covers('Δ') {
		t.Skip("the bundled interface face has grown Greek")
	}
	t.Cleanup(func() { SetFallbackFamilies() })

	// Nobody lends: the rune is a box, and that answer is remembered.
	SetFallbackFamilies("Acme Nothing At All")
	if fb := fallbackFor(f, 'Δ'); fb != nil {
		t.Fatalf("something lent a glyph from %q", fb.Family)
	}
	boxed := f.shapeOf("Δ")
	if len(boxed.runs) != 1 {
		t.Fatalf("%d runs with nothing lending", len(boxed.runs))
	}

	// Now name one that has it: both the decision and the shaped run
	// have to be made again.
	SetFallbackFamilies(FamilyMono)
	if fb := fallbackFor(f, 'Δ'); fb == nil {
		t.Error("a newly named face was not asked, because the old answer was remembered")
	}
	if got := f.shapeOf("Δ"); len(got.runs) != 2 {
		t.Errorf("%d runs after naming a face that has the rune; the laid-out run was cached", len(got.runs))
	}

	// And the other way: take the face away and the glyph goes back to
	// being a box.
	SetFallbackFamilies("Acme Nothing At All")
	if fb := fallbackFor(f, 'Δ'); fb != nil {
		t.Errorf("%q still lends after it was taken off the list", fb.Family)
	}
	if got := f.shapeOf("Δ"); len(got.runs) != 1 {
		t.Errorf("%d runs after the lender was taken away; the borrowed glyph was kept", len(got.runs))
	}
}

// **A lender has to cover the rune, not the alphabet.**
//
// `bakeFaceAscent` refuses a face that cannot draw an 'A', which is
// right for the face that draws the interface — an unreadable window is
// worse than the wrong typeface — and wrong for one lending a single
// glyph. It kept every symbol and emoji face from ever lending
// (comms-mail item 58).
func TestALenderNeedNotDrawLatin(t *testing.T) {
	// A face that is installed and has no Latin at all is what the
	// check exists for, and what the fix is about. There may be none
	// on this machine.
	var found string
	for _, name := range []string{"Noto Color Emoji", "Noto Emoji", "Noto Sans Arabic", "Noto Naskh Arabic", "Noto Sans CJK SC", "Symbola"} {
		fam := ResolveFont([]string{name})
		if fam == "" {
			continue
		}
		f, err := bakeFaceAscent(fam, WeightRegular, 16, 18)
		if err != nil {
			t.Fatalf("%q was refused as a lender: %v — a lender has to cover a rune, not the alphabet", fam, err)
		}
		if !f.covers('A') {
			found = fam
			break
		}
	}
	if found == "" {
		t.Skip("no installed face without Latin to check this with")
	}
	// And the interface path keeps the check, which is the half that
	// stops an unreadable window.
	if _, err := bakeFaceAscent(found, WeightRegular, 16, 0); err == nil {
		t.Errorf("%q was accepted as the interface face, and it cannot draw Latin", found)
	}
}

// **A script that needs shaping is not borrowed for.**
//
// The toolkit has no shaping and no bidi, so a borrowed Arabic glyph
// would draw unjoined and in logical order — legibly wrong, where a box
// is honestly illegible. Those scripts stay boxes until shaping lands.
func TestAScriptThatNeedsShapingIsNotBorrowedFor(t *testing.T) {
	SetFallbackFamilies(FamilyMono, "Noto Sans", "DejaVu Sans", "Noto Naskh Arabic", "Noto Sans Arabic")
	t.Cleanup(func() { SetFallbackFamilies() })
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	for _, r := range []rune{'ا', 'ب', 'א', 'अ', 'ก'} {
		if fb := fallbackFor(f, r); fb != nil {
			t.Errorf("%c was borrowed from %q; it needs shaping the toolkit has not got", r, fb.Family)
		}
	}
	// And the ones a glyph at a time is the truth for are unaffected.
	for _, r := range []rune{'Δ', 'д', '✓'} {
		if fb := fallbackFor(f, r); fb == nil {
			t.Errorf("%c was not borrowed, and one glyph per rune is right for it", r)
		}
	}
}
