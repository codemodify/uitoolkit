package style

import (
	"testing"
)

// Medium and semibold: the installed face of that weight, or the nearest
// drawn heavier; the bundled face draws its regular heavier by degrees.
func TestFontWeights(t *testing.T) {
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	for w, fc := range map[Weight]int{WeightRegular: fcRegular, WeightMedium: fcMedium, WeightSemibold: fcSemibold, WeightBold: fcBold} {
		if fcWeight(w) != fc || cssWeight(fc) != w {
			t.Errorf("weight %d ↔ fontconfig %d", w, fc)
		}
	}
	reg, _ := faceFor(FamilyUI, WeightRegular)
	med, _ := faceFor(FamilyUI, WeightMedium)
	semi, _ := faceFor(FamilyUI, WeightSemibold)
	bold, _ := faceFor(FamilyUI, WeightBold)
	if reg != uiReg || bold != uiBold {
		t.Fatal("regular and bold are the bundled faces")
	}
	if !(med.embolden > 0 && semi.embolden > med.embolden && semi.embolden < synthBold) {
		t.Fatalf("embolden: medium %v, semibold %v", med.embolden, semi.embolden)
	}
	if again, _ := faceFor(FamilyUI, WeightMedium); again != med {
		t.Fatal("the emboldened face is cached")
	}
	// A look's weight fonts: regular < medium < semibold in advance.
	t.Setenv(SystemFontsEnv, "0")
	lk := DarkLook()
	text := "Settings and more"
	r := lk.Font().Advance(text)
	m := WeightFontOf(lk, WeightMedium).Advance(text)
	s := WeightFontOf(lk, WeightSemibold).Advance(text)
	if !(r < m && m < s) {
		t.Fatalf("advances regular %v, medium %v, semibold %v", r, m, s)
	}
	if WeightFontOf(lk, WeightBold) != lk.BoldFont() || WeightFontOf(lk, WeightRegular) != lk.Font() {
		t.Fatal("bold and regular are the look's own faces")
	}
}
