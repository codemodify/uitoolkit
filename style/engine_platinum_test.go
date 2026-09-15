package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var platinumPackNames = []string{"platinum", "platinum-lime"}

func TestPlatinumPacksRegisteredInYearOrder(t *testing.T) {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for _, n := range platinumPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != "platinum" || p.Lineage != "Mac OS" || p.Year != 1997 {
			t.Fatalf("%s: engine %q lineage %q year %d", n, p.Tokens.Engine, p.Lineage, p.Year)
		}
		if lk := p.Look(); lk.Engine().ID() != "platinum" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	// Mac OS 8 (1997) sorts after Windows 95 and before Aqua (2001).
	if pos["win95"] > pos["platinum"] || pos["platinum"] > pos["aqua"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
}

func TestPlatinumStyleHints(t *testing.T) {
	p, _ := LoadTheme("platinum")
	lk := p.Look()
	if LookHint(lk, HintDialogPrimaryFirst) != 0 || LookHint(lk, HintTabsCentered) != 0 {
		t.Fatal("Platinum: default button last, tabs from the left")
	}
}

// The close box sits at the left of the title bar, where it is painted.
func TestPlatinumCloseBoxOnTheLeft(t *testing.T) {
	p, _ := LoadTheme("platinum")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("scale %v: close box %v not at the left of frame %v", scale, cr, b)
		}
		img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
		ctx := paintengine2d.NewContext(img)
		lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
		// Its black frame is on the rect's edge.
		r, g, bl, _ := img.PremulAt(int(cr.Min.X), int(cr.Min.Y+cr.Dy()*0.5))
		if r > 40 || g > 40 || bl > 40 {
			t.Fatalf("scale %v: close box edge rgb(%d,%d,%d), want black", scale, r, g, bl)
		}
	}
}

func TestPlatinumPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range platinumPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}
