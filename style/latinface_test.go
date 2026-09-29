package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A face that cannot draw Latin cannot draw the interface, and asking
// for one must not kill the process.
//
// This used to panic. Noto Sans Arabic, Noto Sans CJK and Noto Color
// Emoji have no Latin 'A', thirty such families are in the chooser's
// list on an ordinary desktop, and the choice is saved in look.json — so
// picking one made every uitoolkit application on the machine crash on
// start and keep crashing, because the setting outlived the process.
func TestScriptOnlyFaceFallsBackInsteadOfPanicking(t *testing.T) {
	// Every family the chooser offers, because the invariant is about
	// all of them: asking for any installed face must give back a face
	// that can draw the interface, and must not panic. That used to be
	// false for about thirty of them on an ordinary desktop — the
	// script-only ones with no Latin 'A' — and the choice is saved in
	// look.json, so it crashed on every start until the file was edited.
	fams := ListFontFamilies()
	if len(fams) <= 2 {
		t.Skip("no installed fonts here (UITK_SYSTEM_FONTS=0)")
	}
	checked := 0
	for _, fam := range fams {
		f := BakeFamily(fam, WeightRegular, 14, paintengine2d.White)
		if f == nil {
			t.Fatalf("%s: no face at all", fam)
		}
		if f.Advance("Hello") <= 0 {
			t.Errorf("%s: the face it gave back cannot draw Latin", fam)
		}
		// Where it fell back, it says which face it really is rather
		// than claiming to be the one that was asked for.
		if f.Family != fam && f.Family != "" && f.Family != FamilyUI {
			t.Errorf("%s: fell back to %q, which is neither the bundled face nor the request", fam, f.Family)
		}
		checked++
	}
	t.Logf("%d installed families, all usable", checked)
}

// A face with no Latin at all is the case that used to panic, named so
// the reason this exists is not lost if the desktop stops having one.
func TestAFaceWithNoLatinFallsBack(t *testing.T) {
	var noLatin string
	for _, fam := range []string{"Noto Sans Arabic", "Noto Kufi Arabic", "Noto Sans Hebrew", "Noto Color Emoji"} {
		if installed(fam) {
			noLatin = fam
			break
		}
	}
	if noLatin == "" {
		t.Skip("no Latin-less face installed here")
	}
	f := BakeFamily(noLatin, WeightRegular, 14, paintengine2d.White)
	if f == nil || f.Advance("Hello") <= 0 {
		t.Fatalf("%s: no usable face came back", noLatin)
	}
	if f.Family == noLatin {
		t.Fatalf("%s: reports itself as a face that cannot draw Latin", noLatin)
	}
}

// A family that is simply not installed already fell back, and still
// does: the two cases have the same answer.
func TestUnknownFamilyFallsBack(t *testing.T) {
	f := BakeFamily("No Such Family At All", WeightRegular, 14, paintengine2d.White)
	if f == nil || f.Advance("Hello") <= 0 {
		t.Fatal("an unknown family did not fall back")
	}
}

// The bundled faces are never swapped out from under a caller, whatever
// happens: they are the fallback, so a failure there has nowhere to go
// and must be reported rather than papered over.
func TestBundledFacesStillBake(t *testing.T) {
	for _, fam := range []string{FamilyUI, FamilyMono, ""} {
		f := BakeFamily(fam, WeightRegular, 14, paintengine2d.White)
		if f == nil || f.Advance("Hello") <= 0 {
			t.Fatalf("%q did not bake", fam)
		}
	}
}

// installed reports whether the desktop running the test has this
// family, so the test says nothing on a machine without it.
func installed(family string) bool {
	for _, f := range ListFontFamilies() {
		if strings.EqualFold(f, family) {
			return true
		}
	}
	return false
}
