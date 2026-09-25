package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every pack has to be able to lay a tip out: a face to measure it in, a
// padding, and a measure wide enough to be worth wrapping at. A pack whose
// TooltipStyle came back empty would measure its bubble at nothing and show
// an empty box.
func TestEveryPackLaysATipOut(t *testing.T) {
	packs := append(ListBuiltinThemes(), ListSkins()...)
	if len(packs) < 100 {
		t.Fatalf("only %d packs — the registry did not load", len(packs))
	}
	for _, p := range packs {
		lk := p.Look()
		ts := TooltipStyleOf(lk)
		if ts.Face == nil {
			t.Errorf("%s: no tip face", p.Name)
			continue
		}
		if ts.Pad <= 0 {
			t.Errorf("%s: tip padding %v", p.Name, ts.Pad)
		}
		// The measure is 48 characters of the tip's own face plus the
		// padding, so it tracks the era's type rather than a constant.
		want := ts.Face.Advance(tipAlphabet)/float32(len(tipAlphabet))*tipMeasureChars + ts.Pad*2
		if d := ts.MaxW - want; d > 0.01 || d < -0.01 {
			t.Errorf("%s: measure %v, want %v", p.Name, ts.MaxW, want)
		}
		lines, sz := ts.Wrap(longSentence, 0)
		if len(lines) < 2 {
			t.Errorf("%s: a 95-character tip did not wrap: %q", p.Name, lines)
		}
		if got := strings.Join(lines, " "); got != longSentence {
			t.Errorf("%s: the wrap changed the text: %q", p.Name, got)
		}
		for _, ln := range lines {
			if w := ts.Face.Advance(ln); w > sz.X-ts.Pad*2 {
				t.Errorf("%s: line %q is %.1f wide in a %.1f bubble", p.Name, ln, w, sz.X-ts.Pad*2)
			}
		}
		if want := ts.Face.Height() * float32(len(lines)); sz.Y < want {
			t.Errorf("%s: bubble %.1f tall for %d lines of %.1f", p.Name, sz.Y, len(lines), ts.Face.Height())
		}
	}
}

const longSentence = "KDE's and GNOME's own Open and Save dialogs, through the XDG portal, instead of the themed ones."

// A tip narrower than the look wants wraps to what it was given, and a
// bubble squeezed past a single glyph still returns lines rather than
// looping.
func TestTooltipStyleWrapsToTheRoomItIsGiven(t *testing.T) {
	ts := TooltipStyleOf(LightLook())
	wide, _ := ts.Wrap(longSentence, 0)
	narrow, sz := ts.Wrap(longSentence, ts.MaxW*0.5)
	if len(narrow) <= len(wide) {
		t.Fatalf("half the width should take more lines: %d then %d", len(wide), len(narrow))
	}
	if sz.X > ts.MaxW*0.5 {
		t.Fatalf("bubble %v wider than the %v it was given", sz.X, ts.MaxW*0.5)
	}
	if lines, _ := ts.Wrap(longSentence, 1); len(lines) == 0 {
		t.Fatal("a bubble with no room still has to say something")
	}
	if lines, sz := ts.Wrap("", 0); lines != nil || sz != (paintengine2d.Point{}) {
		t.Fatal("no text, no bubble")
	}
}

// The engines that set their tips in another face say so through
// TooltipStyle, which is what the widget measures with: a bubble measured
// in the body face and painted in a wider one loses its last word, and one
// measured in a wider face than it is painted in stands half empty.
func TestTooltipFacesThatAreNotTheBodyFace(t *testing.T) {
	look := func(name string) LookAndFeel {
		t.Helper()
		p, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("no pack %q", name)
		}
		return p.Look()
	}
	// Material sets a plain tooltip in body-small, four points under the
	// body face. It used to be measured in body and drawn in small, with a
	// fallback to body for whatever did not fit.
	md := look("material")
	if f := TooltipStyleOf(md).Face; f.Size >= md.Font().Size {
		t.Errorf("material: tip face %v, body %v — body-small is smaller", f.Size, md.Font().Size)
	}
	// Workbench had one face, and a tip is in it like everything else.
	am := look("amiga31")
	if f := TooltipStyleOf(am).Face; f != am.MonoFont() {
		t.Error("amiga: a tip is Topaz, the look's mono face")
	}
}
