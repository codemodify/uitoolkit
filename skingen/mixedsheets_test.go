package skingen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/skingen"
	"github.com/codemodify/uitoolkit/style"
)

// A plan that mixes pixelated and smooth sheets must load as it was written.
//
// One pixelated sheet used to set design.pixelated for the *whole skin*, and
// only that sheet wrote the key. The loader rightly inherits the design
// default where a sheet omits it, so the deliberately smooth sheets came back
// pixelated: the plan said one thing and the manifest meant another, before
// anything was rendered. A player had to post-process the generator's output
// and add explicit `pixelated: false` keys itself.

// mixedPlan is two ordinary sheets, one of each kind. order flips which comes
// first, because a bug that depends on the order is still a bug.
func mixedPlan(pixelatedFirst bool) *skingen.Plan {
	fill := func(hex string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(skingen.Hex(hex)))
		}
	}
	pix := &skingen.Sheet{
		Name: "pixels", W: 32, H: 32, Pixelated: true,
		Cells: []skingen.Cell{{Name: "button.normal", X: 0, Y: 0, W: 32, H: 32, Draw: fill("#40444c")}},
	}
	smooth := &skingen.Sheet{
		Name: "smooth", W: 32, H: 32,
		Cells: []skingen.Cell{{Name: "button.hover", X: 0, Y: 0, W: 32, H: 32, Draw: fill("#4c515b")}},
	}
	sheets := []*skingen.Sheet{pix, smooth}
	if !pixelatedFirst {
		sheets = []*skingen.Sheet{smooth, pix}
	}
	return &skingen.Plan{
		Name: "mixed", Label: "Mixed", Base: "breeze-night",
		Sheets: sheets,
		Parts: []skingen.PartBinding{{
			Part:   "button",
			States: [][2]string{{"normal", "button.normal"}, {"hover", "button.hover"}},
		}},
	}
}

func TestAMixedPlanKeepsEachSheetsSamplingPolicy(t *testing.T) {
	for _, pixelatedFirst := range []bool{true, false} {
		dir := t.TempDir()
		p := mixedPlan(pixelatedFirst)
		if err := skingen.Write(dir, p); err != nil {
			t.Fatalf("Write: %v", err)
		}
		sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
		if err != nil {
			t.Fatalf("the mixed plan does not load: %v", err)
		}
		for _, want := range p.Sheets {
			got := sk.Sheets[want.Name]
			if got == nil {
				t.Fatalf("pixelated-first=%v: sheet %q did not load", pixelatedFirst, want.Name)
			}
			if got.Pixelated != want.Pixelated {
				t.Errorf("pixelated-first=%v: sheet %q loaded Pixelated=%v, the plan said %v",
					pixelatedFirst, want.Name, got.Pixelated, want.Pixelated)
			}
		}
	}
}

// A plan whose sheets agree is unchanged: design says it once, which is what
// the shipped manifests look like.
func TestAnAllPixelatedPlanStillSaysSoOnce(t *testing.T) {
	dir := t.TempDir()
	p := mixedPlan(true)
	for _, sh := range p.Sheets {
		sh.Pixelated = true
	}
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !sk.Design.Pixelated {
		t.Error("a wholly pixelated plan did not set the design default")
	}
	for name, sh := range sk.Sheets {
		if !sh.Pixelated {
			t.Errorf("sheet %q loaded smooth", name)
		}
	}
}

// And one with no pixelated sheet says nothing at all.
func TestAnAllSmoothPlanClaimsNoPixelation(t *testing.T) {
	dir := t.TempDir()
	p := mixedPlan(true)
	for _, sh := range p.Sheets {
		sh.Pixelated = false
	}
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if sk.Design.Pixelated {
		t.Error("a wholly smooth plan set the design default")
	}
	for name, sh := range sk.Sheets {
		if sh.Pixelated {
			t.Errorf("sheet %q loaded pixelated", name)
		}
	}
}
