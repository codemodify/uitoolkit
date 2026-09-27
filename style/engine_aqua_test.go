//go:build theme_engine_all || theme_engine_aqua

package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var aquaPackNames = []string{"aqua", "aqua-graphite", "brushed-metal", "aqua-night"}

func TestAquaPacksRegisteredInYearOrder(t *testing.T) {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for _, n := range aquaPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != "aqua" || p.Lineage != "Mac OS" {
			t.Fatalf("%s: engine %q lineage %q", n, p.Tokens.Engine, p.Lineage)
		}
		if lk := p.Look(); lk.Engine().ID() != "aqua" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	if pos["aqua"] > pos["brushed-metal"] || pos["aqua-graphite"] > pos["brushed-metal"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
	// The two legacy era packs are replaced, not duplicated, and the dark
	// one says it is an invention.
	if p, _ := LoadTheme("aqua"); p.Year != 2001 || p.Label != "Aqua" {
		t.Fatalf("aqua pack %q %d", p.Label, p.Year)
	}
	if p, _ := LoadTheme("aqua-night"); !strings.Contains(p.Summary, "Not historical") {
		t.Fatalf("aqua-night summary must say Aqua had no dark mode: %q", p.Summary)
	}
}

func TestAquaStyleHints(t *testing.T) {
	p, _ := LoadTheme("aqua")
	lk := p.Look()
	if LookHint(lk, HintTabsCentered) != 1 {
		t.Fatal("Aqua centres its tabs")
	}
	if LookHint(lk, HintDialogPrimaryFirst) != 0 {
		t.Fatal("Aqua puts the default button last")
	}
}

// The close button is the red traffic light at the left of the title bar,
// and hit-testing (WindowCloseRect) agrees with what is painted.
func TestAquaCloseIsTheRedLightOnTheLeft(t *testing.T) {
	p, _ := LoadTheme("aqua")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("scale %v: close rect %v not the left light of frame %v (top inset %v)", scale, cr, b, in.Top)
		}
		img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
		ctx := paintengine2d.NewContext(img)
		lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
		c := cr.Center()
		r, g, bl, _ := img.PremulAt(int(c.X), int(c.Y))
		if int(r) < int(g)+40 || int(r) < int(bl)+40 {
			t.Fatalf("scale %v: close light centre is rgb(%d,%d,%d), want red", scale, r, g, bl)
		}
	}
}

func TestAquaPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range aquaPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			exerciseEngine(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}
