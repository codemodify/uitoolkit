package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// runeInk is how much ink one rune leaves, so a rune the face draws is
// told from one it renders as the notdef box.
func runeInk(t *testing.T, f *Font, r rune) int {
	t.Helper()
	img := paintengine2d.NewImage(28, 28)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, string(r), paintengine2d.Pt(4, 4), paintengine2d.White)
	img.Touch()
	n := 0
	for y := 0; y < 28; y++ {
		for x := 0; x < 28; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				n++
			}
		}
	}
	return n
}

// What the bundled faces cover, as docs/contracts.md states it.
//
// The table on that page is what an application plans around: whether a
// rune is safe to type decides whether it needs an icon. A face swapped
// or revised without the table following would make the documentation
// promise coverage that is not there, and the application that believed
// it would ship boxes.
func TestBundledFaceCoverageIsAsDocumented(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "0")
	ui := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	mono := BakeFamily(FamilyMono, WeightRegular, 16, paintengine2d.White)
	// A private-use rune no face has: the notdef box, to compare against.
	uiBox, monoBox := runeInk(t, ui, ''), runeInk(t, mono, '')

	has := func(f *Font, box int, r rune) bool {
		ink := runeInk(t, f, r)
		return ink != box && ink > 0
	}
	for _, c := range []struct {
		r       rune
		inUI    bool
		inMono  bool
		whatFor string
	}{
		// Latin, both.
		{'A', true, true, "Latin"},
		{'é', true, true, "accented Latin"},
		// The marks an interface reaches for: the mono face has them and
		// the interface face does not, which is the whole reason the
		// icons exist.
		{'✓', false, true, "check mark"},
		{'✗', false, true, "cross"},
		{'→', false, true, "arrow"},
		{'└', false, true, "box drawing"},
		// Greek and Cyrillic: mono only.
		{'Ω', false, true, "Greek"},
		{'Д', false, true, "Cyrillic"},
		// CJK: neither.
		{'中', false, false, "CJK"},
		// The four drawn from paths, both faces (style/symbols.go).
		{'★', true, true, "star"},
		{'📎', true, true, "paperclip"},
		{'●', true, true, "bullet"},
		{'🔇', true, true, "muted"},
	} {
		if got := has(ui, uiBox, c.r); got != c.inUI {
			t.Errorf("%s %q: the interface face has it = %v, docs/contracts.md says %v",
				c.whatFor, c.r, got, c.inUI)
		}
		if got := has(mono, monoBox, c.r); got != c.inMono {
			t.Errorf("%s %q: the mono face has it = %v, docs/contracts.md says %v",
				c.whatFor, c.r, got, c.inMono)
		}
	}
}

// Pair kerning is applied and no GSUB feature is, which is the whole of
// what "shapes rune by rune with GPOS kerning" means.
func TestKerningYesLigaturesNo(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "0")
	f := BakeFamily(FamilyUI, WeightRegular, 14, paintengine2d.White)
	if pair, apart := f.Advance("AV"), f.Advance("A")+f.Advance("V"); pair >= apart {
		t.Errorf("AV measures %v and A+V %v: the face is not kerned", pair, apart)
	}
	// No GSUB, so "fi" is two glyphs and measures as two.
	if lig, apart := f.Advance("fi"), f.Advance("f")+f.Advance("i"); lig != apart {
		t.Errorf("fi measures %v and f+i %v: something substituted a ligature", lig, apart)
	}
}
