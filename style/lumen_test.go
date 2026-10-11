//go:build theme_engine_all || theme_engine_macos_tahoe

package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// **Lumen is a material; Tahoe is a reproduction.** Both draw the same
// shapes through the same engine, and what separates them is three
// numbers: a tint you can see, a blur that frosts without flattening,
// and the bevel they both already had. Rendered over content, chrome at
// Tahoe's eighth of an alpha is an outline you read straight through.
func TestLumenHasAMaterialAndTahoeDoesNot(t *testing.T) {
	tahoe, ok := LoadTheme("tahoe-night")
	if !ok {
		t.Skip("the macOS Tahoe engine is not in this build")
	}
	lumen, ok := LoadTheme("lumen-night")
	if !ok {
		t.Fatal("lumen-night is not registered, though the engine it rides with is")
	}
	tl, ll := tahoe.Look(), lumen.Look()
	tc, lc := tahoeColors(tl), tahoeColors(ll)

	if !(lc.glass.A > tc.glass.A*2) {
		t.Errorf("lumen's chrome glass is %.3f alpha against Tahoe's %.3f: not a material", lc.glass.A, tc.glass.A)
	}
	if !(lc.glassRim.A > tc.glassRim.A) {
		t.Errorf("lumen's rim is %.3f alpha against Tahoe's %.3f", lc.glassRim.A, tc.glassRim.A)
	}
	// The blur has to frost without flattening: some, and nowhere near
	// the fourteen that leaves a flat colour with nothing to bend.
	if b := ll.P("glassBlur", 0); !(b > 0 && b < 12) {
		t.Errorf("lumen's chrome blur is %v, want a few pixels", b)
	}
	if b := ll.P("menuBlur", 14); !(b > 0 && b < 12) {
		t.Errorf("lumen's menu blur is %v, want less than Tahoe's flattening 14", b)
	}
}

// **Tahoe is left exactly as it was: its chrome glass is clear.**
//
// This reads the pixels rather than the params, because the first
// version of it asked l.P("glassBlur", 0) and passed its *own* default —
// Tahoe's pack sets no such key, so the answer came back 0 whatever the
// engine did with it, and changing the engine's default to 7 left the
// test green. What the pack must preserve is that content behind
// Tahoe's chrome stays sharp, so that is what is measured.
func TestTahoeChromeDoesNotFrostAndLumensDoes(t *testing.T) {
	if _, ok := LoadTheme("tahoe-night"); !ok {
		t.Skip("the macOS Tahoe engine is not in this build")
	}
	const w, h = 240, 90

	// sharpness is the hardest edge left in a row through the capsule.
	// Fine stripes behind clear glass keep their edges; frosted, they
	// smear into each other.
	sharpness := func(theme string) int {
		pack, _ := LoadTheme(theme)
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#101830")))
		// Blocks the size of a list row's artwork, which is what chrome
		// actually floats over. Stripes a few pixels wide would be
		// flattened outright by any blur at all — that is the regime the
		// menus are in, and measuring there would say a frosted pane and
		// a flat one are the same thing.
		for x := 0; x < w; x += 48 {
			ctx.DrawRect(paintengine2d.XYWH(float32(x), 0, 24, h), paintengine2d.Fill(Hex("#e8eeff")))
		}
		DrawToolGroupOf(pack.Look(), ctx, paintengine2d.XYWH(24, 24, w-48, h-48), 3)
		img.Touch()
		// The measure is the steepest step as a fraction of the whole
		// range across the row, not the step on its own. A tint lowers
		// both in step and would otherwise look exactly like a blur
		// here; only a blur spreads an edge over several pixels and so
		// lowers the step while leaving the range.
		y := h / 2
		lo, hi, worst := 255, 0, 0
		for x := 60; x < w-60; x++ {
			a := int(img.Pix[(y*img.Width+x)*4])
			b := int(img.Pix[(y*img.Width+x+1)*4])
			if a < lo {
				lo = a
			}
			if a > hi {
				hi = a
			}
			if d := a - b; d > worst {
				worst = d
			} else if -d > worst {
				worst = -d
			}
		}
		if hi <= lo {
			t.Fatalf("%s: nothing behind the chrome varies at all", theme)
		}
		return worst * 100 / (hi - lo)
	}

	tahoe := sharpness("tahoe-night")
	lumen := sharpness("lumen-night")
	// Clear glass leaves the step as steep as the range: an edge behind
	// it is still an edge.
	if tahoe < 70 {
		t.Errorf("behind Tahoe's chrome the steepest step is %d%% of the range: it is frosting content, and its tool bar glass is clear", tahoe)
	}
	// Frosted glass spreads that edge over several pixels.
	if lumen > tahoe/2 {
		t.Errorf("behind Lumen's chrome the steepest step is %d%% of the range against Tahoe's %d%%: it is not frosting anything", lumen, tahoe)
	}
}

// Both sides of the pack are registered and named for what they are.
func TestLumenShipsLightAndDark(t *testing.T) {
	if _, ok := LoadTheme("tahoe"); !ok {
		t.Skip("the macOS Tahoe engine is not in this build")
	}
	for _, n := range []string{"lumen", "lumen-night"} {
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("%s is not registered", n)
		}
		if p.Summary == "" || p.Label == "" {
			t.Errorf("%s has no label or summary", n)
		}
	}
}

// **Two shapes of one material.** Round is Apple's capsule; square is
// the same glass with the corner only taken off. They differ in
// ThemePack.Corners and nothing else, so the material must be identical
// and only the corner may move.
func TestLumenShipsBothShapes(t *testing.T) {
	if _, ok := LoadTheme("lumen-night"); !ok {
		t.Skip("the macOS Tahoe engine is not in this build")
	}
	const w, h = 200, 60
	// filledCorner counts how much of the capsule's top-left 12x12 is
	// drawn. A pill cuts most of it away; a hinted corner keeps nearly
	// all of it.
	filledCorner := func(theme string) int {
		pack, ok := LoadTheme(theme)
		if !ok {
			t.Fatalf("%s is not registered", theme)
		}
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#000000")))
		box := paintengine2d.XYWH(10, 10, w-20, h-20)
		DrawToolGroupOf(pack.Look(), ctx, box, 3)
		img.Touch()
		// Brightness, not alpha: the ground is painted opaque black, so
		// every pixel in the image is opaque and an alpha test would
		// count the whole square whatever shape was drawn on it.
		n := 0
		for y := 10; y < 22; y++ {
			for x := 10; x < 22; x++ {
				if img.Pix[(y*img.Width+x)*4] > 24 {
					n++
				}
			}
		}
		return n
	}
	round := filledCorner("lumen-night")
	square := filledCorner("lumen-square-night")
	if square <= round {
		t.Errorf("the square pack's corner is %d/144 filled and the round one's %d: the shapes do not differ", square, round)
	}
	if round >= 100 {
		t.Errorf("the round pack's corner is %d/144 filled; it is not a capsule", round)
	}
	if square < 100 {
		t.Errorf("the square pack's corner is %d/144 filled; that is a curve, not a hint", square)
	}

	// Same material, both shapes: only the corner was meant to move.
	rc := tahoeColors(mustTheme(t, "lumen-night").Look())
	sc := tahoeColors(mustTheme(t, "lumen-square-night").Look())
	if rc.glass != sc.glass || rc.glassRim != sc.glassRim {
		t.Errorf("the two shapes have different glass: %v/%v against %v/%v", rc.glass, rc.glassRim, sc.glass, sc.glassRim)
	}
}

func mustTheme(t *testing.T, name string) ThemePack {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("%s is not registered", name)
	}
	return p
}

// CornersSoft takes the corner off rather than squaring it: a clamp, so
// a pack already tighter than the hint keeps its own shape, and a pack
// that is square on purpose stays square.
func TestSoftCornersAreAHintNotACurve(t *testing.T) {
	def := DefaultMetrics()
	for _, tc := range []struct {
		name string
		in   float32
		want float32
	}{
		{"a capsule comes down to the hint", 24, SoftRadius},
		{"so does a middling radius", 8, SoftRadius},
		{"one already tighter is left alone", 2, 2},
		{"square stays square", 0, 0},
	} {
		if got := softened(tc.in, def.FontSize); got != tc.want {
			t.Errorf("%s: softened(%v) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
	// It is a 1x design length, so it follows the font size like every
	// other length here.
	if got := softened(24, def.FontSize*2); got != SoftRadius*2 {
		t.Errorf("at twice the font size the hint is %v, want %v", got, float32(SoftRadius*2))
	}
}
