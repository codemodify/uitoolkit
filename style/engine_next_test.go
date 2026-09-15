package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var nextPackNames = []string{
	"next", "next-night", "openstep",
	"wmaker-default", "wmaker-openstep", "wmaker-night", "wmaker-steelbluesilk",
}

func TestNextPacksRegisteredInYearOrder(t *testing.T) {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	want := map[string]struct {
		label, lineage string
		year           int
	}{
		"next":                 {"NeXTSTEP", "NeXT", 1989},
		"next-night":           {"NeXTSTEP Night", "NeXT", 1989},
		"openstep":             {"OPENSTEP", "NeXT", 1994},
		"wmaker-default":       {"Window Maker", "Window Maker", 1997},
		"wmaker-openstep":      {"Window Maker OpenStep", "Window Maker", 1997},
		"wmaker-night":         {"Window Maker Night Sky", "Window Maker", 1998},
		"wmaker-steelbluesilk": {"Window Maker SteelBlueSilk", "Window Maker", 1999},
	}
	for _, n := range nextPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "next" || p.Label != w.label || p.Lineage != w.lineage || p.Year != w.year {
			t.Fatalf("%s: engine %q label %q lineage %q year %d", n, p.Tokens.Engine, p.Label, p.Lineage, p.Year)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "next" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	order := []string{"next", "openstep", "win95", "wmaker-default", "wmaker-night", "wmaker-steelbluesilk", "aqua"}
	for i := 1; i < len(order); i++ {
		if pos[order[i-1]] > pos[order[i]] {
			t.Fatalf("packs not ordered by year: %s after %s (%v)", order[i-1], order[i], pos)
		}
	}
	// The legacy era packs are replaced, not duplicated, and the dark one
	// says NeXT never had a dark mode.
	n := 0
	for _, name := range AllBuiltinThemeNames() {
		if name == "next" || name == "next-night" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("next / next-night listed %d times", n)
	}
	if p, _ := LoadTheme("next-night"); !strings.Contains(p.Summary, "Not historical") {
		t.Fatalf("next-night summary must say NeXTSTEP had no dark mode: %q", p.Summary)
	}
	if p, _ := LoadTheme("next-dark"); p.Name != "next-night" {
		t.Fatalf("next-dark alias resolves to %q", p.Name)
	}
}

// The Window Maker packs carry their theme files' textures verbatim.
func TestNextWindowMakerTexturesAreTheThemeFiles(t *testing.T) {
	cases := []struct {
		pack, key string
		kind      int
		cols      []string
	}{
		{"wmaker-default", "ftitle", nxHGrad, []string{"#505a5e", "#202a2e"}},
		{"wmaker-default", "mtext", nxHGrad, []string{"#c2c0c5", "#828085"}},
		{"wmaker-default", "icon", nxDGrad, []string{"#a6a6b6", "#515561"}},
		{"wmaker-openstep", "ftitle", nxDGrad, []string{"#000010", "#202070"}},
		{"wmaker-openstep", "mtext", nxHGrad, []string{"#d0d0d0", "#808080"}},
		{"wmaker-night", "ftitle", nxHGrad, []string{"#000000", "#3d637f", "#315c77", "#333f3e"}},
		{"wmaker-night", "resize", nxHGrad, []string{"#000000", "#494c63", "#444661", "#202841"}},
		{"wmaker-steelbluesilk", "ftitle", nxDGrad, []string{"#18191f", "#939abd", "#616185", "#616185", "#5f5f83", "#555575", "#59597a", "#555575", "#939abd"}},
		{"wmaker-steelbluesilk", "mtitle", nxVGrad, []string{"#18191f", "#474967", "#413b6d"}},
		{"next", "ftitle", nxSolid, []string{"#000000"}},
		{"next", "mtext", nxSolid, []string{"#aaaaaa"}},
	}
	for _, tc := range cases {
		p, _ := LoadTheme(tc.pack)
		if k := int(p.Tokens.Params[tc.key]); k != tc.kind {
			t.Errorf("%s %s: type %d want %d", tc.pack, tc.key, k, tc.kind)
		}
		for i, want := range tc.cols {
			got, ok := p.Tokens.Extra[tc.key+itoa(i)]
			if !ok || colorHexPadded(got) != want {
				t.Errorf("%s %s%d = %s want %s", tc.pack, tc.key, i, colorHexPadded(got), want)
			}
		}
		if _, extra := p.Tokens.Extra[tc.key+itoa(len(tc.cols))]; extra {
			t.Errorf("%s %s has more than %d colours", tc.pack, tc.key, len(tc.cols))
		}
	}
	if c := nxColors(mustLook(t, "wmaker-night")); colorHexPadded(c.hl) != "#ffe0ac" || colorHexPadded(c.hlTxt) != "#000000" {
		t.Fatalf("Night Sky highlight %s / %s", colorHexPadded(c.hl), colorHexPadded(c.hlTxt))
	}
}

func mustLook(t *testing.T, name string) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("missing %s", name)
	}
	return p.Look()
}

func TestNextStyleHintsAndScrollers(t *testing.T) {
	lk := mustLook(t, "next")
	if LookHint(lk, HintDialogPrimaryFirst) != 0 || LookHint(lk, HintTabsCentered) != 0 {
		t.Fatal("NeXT: default button last, tabs from the left")
	}
	s := ScrollBarStyleOf(lk)
	if s.Arrows != ArrowsTogetherEnd || s.Overlay || s.Thickness != 20 {
		t.Fatalf("NeXT scroller %+v: want 20px gutter with the arrows together at the end", s)
	}
	if lk.eng().FieldFocusRing(lk) {
		t.Fatal("NeXT fields show the caret only")
	}
}

// The close button is at the right of the title bar and the miniaturize
// button at the left; hit-testing (WindowCloseRect) agrees with the paint.
func TestNextCloseButtonAgreesWithPaint(t *testing.T) {
	for _, n := range []string{"next", "openstep", "wmaker-default", "wmaker-night"} {
		for _, scale := range []float32{1, 2} {
			lk := WithScale(mustLook(t, n), scale).(*Classic)
			b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not at the right of the title bar of %v (top inset %v)", n, scale, cr, b, in.Top)
			}
			img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			c := nxColors(lk)
			// The × crosses at the button's centre in the glyph colour.
			glyph := c.ink
			if c.wm {
				glyph = c.ftitleTxt
			}
			ctr := cr.Center()
			if !nxNear(img, int(ctr.X), int(ctr.Y), glyph) {
				r, g, bl, _ := img.PremulAt(int(ctr.X), int(ctr.Y))
				t.Fatalf("%s@%gx: close centre rgb(%d,%d,%d), want the glyph %s", n, scale, r, g, bl, colorHexPadded(glyph))
			}
			// NeXT buttons are light-grey squares inset in the black bar
			// (the face starts inside the 1px highlight; at NeXT's own 15px
			// size the × reaches 3px in).
			if !c.wm {
				u := nxU(lk)
				if !nxNear(img, int(cr.Min.X+2*u), int(cr.Min.Y+2*u), c.face) {
					t.Fatalf("%s@%gx: close button face missing at its inner top-left", n, scale)
				}
				if !nxNear(img, int(cr.Min.X-2*u), int(ctr.Y), Hex("#000000")) {
					t.Fatalf("%s@%gx: the key title bar left of the close button is not black", n, scale)
				}
			}
			mini := nxTitleButton(lk, b, false)
			if mini.Min.X < b.Min.X || mini.Max.X > b.Min.X+b.Dx()*0.2 {
				t.Fatalf("%s@%gx: miniaturize button %v not at the left", n, scale, mini)
			}
			// No close button: no hit rect beyond an empty frame.
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s@%gx: a frame too small for a title bar reports close rect %v", n, scale, r)
			}
		}
	}
}

func nxNear(img *paintengine2d.Image, x, y int, want paintengine2d.Color) bool {
	r, g, b, _ := img.PremulAt(x, y)
	wr, wg, wb, _ := want.Premul8()
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(r, wr) < 40 && d(g, wg) < 40 && d(b, wb) < 40
}

// The trough stipple is a one-pixel checkerboard of the trough colour over
// the face, on both sides of every pixel.
func TestNextTroughStippleIsACheckerboard(t *testing.T) {
	const w, h = 150, 90
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	a, b := Hex("#aaaaaa"), Hex("#555555")
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(a))
	tile := nxStipple(b)
	// Two abutting pieces (a trough split by its knob) keep one phase.
	nxDither(ctx, paintengine2d.XYWH(3, 5, 70, h-5), tile)
	nxDither(ctx, paintengine2d.XYWH(73, 5, w-73, h-5), tile)
	for y := 5; y < h; y++ {
		for x := 3; x < w; x++ {
			want := a
			if (x+y)%2 == 0 {
				want = b
			}
			if !nxNear(img, x, y, want) {
				r, g, bl, _ := img.PremulAt(x, y)
				t.Fatalf("stipple at (%d,%d) rgb(%d,%d,%d), want %s", x, y, r, g, bl, colorHexPadded(want))
			}
		}
	}
}

// wrlib's dgradient: t = (x/w + y/h) / 2 — corners at 0 and 1, the other
// two corners half way.
func TestNextDiagonalGradientIsWrlibs(t *testing.T) {
	r := paintengine2d.XYWH(10, 20, 300, 20)
	p := nxDiag(r, []paintengine2d.GradientStop{Stop(0, Hex("#000000")), Stop(1, Hex("#ffffff"))})
	g := p.Shader.(paintengine2d.LinearGradient)
	tAt := func(x, y float32) float32 {
		d := g.End.Sub(g.Start)
		return paintengine2d.Pt(x, y).Sub(g.Start).Dot(d) / d.LenSq()
	}
	for _, c := range []struct{ x, y, want float32 }{
		{10, 20, 0}, {310, 40, 1}, {310, 20, 0.5}, {10, 40, 0.5}, {160, 30, 0.5},
	} {
		if got := tAt(c.x, c.y); got < c.want-0.001 || got > c.want+0.001 {
			t.Fatalf("t(%v,%v) = %v want %v", c.x, c.y, got, c.want)
		}
	}
}

// Solid textures take wTextureMakeSolid's bevel: value ×1.6 and ÷2 (NeXT's
// #aaaaaa gives white and #555555).
func TestNextSolidBevelIsWmakers(t *testing.T) {
	l, d := nxSolidBevel(Hex("#aaaaaa"))
	if colorHexPadded(l) != "#ffffff" || colorHexPadded(d) != "#555555" {
		t.Fatalf("#aaaaaa bevel %s / %s", colorHexPadded(l), colorHexPadded(d))
	}
	l, d = nxSolidBevel(Hex("#000000"))
	if colorHexPadded(l) != "#b6b6b6" || colorHexPadded(d) != "#616161" {
		t.Fatalf("black bevel %s / %s", colorHexPadded(l), colorHexPadded(d))
	}
}

// Hot and pressed buttons and menu cells paint differently from idle ones,
// and the NeXT highlight is the white lit face with black text.
func TestNextLitStates(t *testing.T) {
	lk := mustLook(t, "next")
	c := nxColors(lk)
	if colorHexPadded(c.lit) != "#ffffff" || colorHexPadded(c.litTxt) != "#000000" || colorHexPadded(c.hl) != "#ffffff" {
		t.Fatalf("NeXT lit %s/%s highlight %s", colorHexPadded(c.lit), colorHexPadded(c.litTxt), colorHexPadded(c.hl))
	}
	img := paintengine2d.NewImage(100, 40)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(4, 4, 88, 28)
	lk.DrawButton(ctx, b, StatePressed|StateHovered, "")
	if !nxNear(img, 40, 18, c.lit) || !nxNear(img, 4, 18, Hex("#000000")) {
		t.Fatal("pressed NeXT button: white face, black top-left edge")
	}
	for _, n := range nextPackNames {
		lk := mustLook(t, n)
		for _, st := range []ControlState{StateHovered, StatePressed | StateHovered} {
			if bytesEqual(rasterButton(lk, StateNone), rasterButton(lk, st)) {
				t.Fatalf("%s: button %#x paints like idle", n, uint32(st))
			}
			if bytesEqual(rasterMenu(lk, StateNone), rasterMenu(lk, st)) {
				t.Fatalf("%s: menu item %#x paints like idle", n, uint32(st))
			}
		}
	}
}

func TestNextPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range nextPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}
