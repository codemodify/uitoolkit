package style

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func restoreGlassLight(t *testing.T) {
	t.Helper()
	p, on := GlassLight()
	t.Cleanup(func() {
		if on {
			SetGlassLight(p)
		} else {
			ClearGlassLight()
		}
	})
}

// gradientEnds pulls the two endpoints out of a linear-gradient paint.
func gradientEnds(t *testing.T, p paintengine2d.Paint) (start, end paintengine2d.Point) {
	t.Helper()
	g, ok := p.Shader.(paintengine2d.LinearGradient)
	if !ok {
		t.Fatalf("the rim paint is %T, not a linear gradient", p.Shader)
	}
	return g.Start, g.End
}

// **With no light set, the rim is exactly what it always was.** Every
// headless render — the screenshot suite, the theme sheets — has no
// pointer, so a look that adopts the tracking rim must render to the
// pixel as it did before. This is the test that keeps the feature from
// quietly regenerating every screenshot in the repo.
func TestAnUnlitRimIsTheTopLitOneItAlwaysWas(t *testing.T) {
	restoreGlassLight(t)
	ClearGlassLight()
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(64, 64))
	b := paintengine2d.XYWH(8, 12, 40, 24)
	lit, unlit := Hex("#ffffff"), Hex("#00000000")

	got := GlassRimPaint(ctx, b, lit, unlit)
	want := VGradient(b, Stop(0, lit), Stop(1, unlit))
	gs, ge := gradientEnds(t, got)
	ws, we := gradientEnds(t, want)
	if gs != ws || ge != we {
		t.Errorf("unlit rim runs %v→%v, the top-lit one runs %v→%v", gs, ge, ws, we)
	}
}

// The lit end of the rim faces the light: move the pointer round the
// glass and the bright side follows it.
func TestTheRimsLitSideFacesTheLight(t *testing.T) {
	restoreGlassLight(t)
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(200, 200))
	b := paintengine2d.XYWH(50, 50, 100, 100)
	c := b.Center()
	lit, unlit := Hex("#ffffff"), Hex("#00000000")

	for _, tc := range []struct {
		name  string
		light paintengine2d.Point
	}{
		{"above", paintengine2d.Pt(100, 0)},
		{"below", paintengine2d.Pt(100, 200)},
		{"left", paintengine2d.Pt(0, 100)},
		{"right", paintengine2d.Pt(200, 100)},
		{"above left", paintengine2d.Pt(0, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SetGlassLight(tc.light)
			start, end := gradientEnds(t, GlassRimPaint(ctx, b, lit, unlit))
			// Offset 0 is the lit stop, so start must be the end of the
			// axis nearer the light.
			dStart := math.Hypot(float64(start.X-tc.light.X), float64(start.Y-tc.light.Y))
			dEnd := math.Hypot(float64(end.X-tc.light.X), float64(end.Y-tc.light.Y))
			if dStart >= dEnd {
				t.Errorf("the lit end is %.1f from the light and the dark end %.1f: the rim is lit on the wrong side", dStart, dEnd)
			}
			// The axis runs through the middle of the glass.
			mx, my := (start.X+end.X)/2, (start.Y+end.Y)/2
			if math.Abs(float64(mx-c.X)) > 0.01 || math.Abs(float64(my-c.Y)) > 0.01 {
				t.Errorf("the gradient's middle is (%.2f,%.2f), the glass's is (%.2f,%.2f)", mx, my, c.X, c.Y)
			}
		})
	}
}

// The light is in device pixels, so a context that is scaled or shifted —
// which every component's is — still lights its glass from the right
// side. Getting this wrong lights every piece of glass in a window as
// though it were at the window's corner.
func TestTheLightIsBroughtBackThroughTheMatrix(t *testing.T) {
	restoreGlassLight(t)
	lit, unlit := Hex("#ffffff"), Hex("#00000000")
	// A component at (100,100) in a window drawn at 2x: its own 0,0 is
	// device (200,200).
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(800, 800))
	ctx.Scale(2, 2)
	ctx.Translate(100, 100)
	b := paintengine2d.XYWH(0, 0, 50, 50)

	// A light at device (200,200) is exactly this component's top left.
	SetGlassLight(paintengine2d.Pt(200, 200))
	start, _ := gradientEnds(t, GlassRimPaint(ctx, b, lit, unlit))
	if start.X >= b.Center().X || start.Y >= b.Center().Y {
		t.Errorf("a light at the component's own top left lit it from %v", start)
	}

	// The same device point with no transform is far away to the
	// bottom right of a component at the origin, so the lit side flips.
	plain := paintengine2d.NewContext(paintengine2d.NewImage(800, 800))
	start2, _ := gradientEnds(t, GlassRimPaint(plain, b, lit, unlit))
	if start2.X <= b.Center().X || start2.Y <= b.Center().Y {
		t.Errorf("without the transform the light should fall from the bottom right, lit from %v", start2)
	}
}

// A light exactly on the middle of the glass gives no direction, and a
// direction from a zero vector is noise: it falls back to top-lit.
func TestALightOnTheMiddleFallsBackToTopLit(t *testing.T) {
	restoreGlassLight(t)
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(200, 200))
	b := paintengine2d.XYWH(50, 50, 100, 100)
	lit, unlit := Hex("#ffffff"), Hex("#00000000")
	SetGlassLight(b.Center())
	gs, ge := gradientEnds(t, GlassRimPaint(ctx, b, lit, unlit))
	ws, we := gradientEnds(t, VGradient(b, Stop(0, lit), Stop(1, unlit)))
	if gs != ws || ge != we {
		t.Errorf("a light on the middle gave %v→%v, want the top-lit %v→%v", gs, ge, ws, we)
	}
}

// **The light reaches the pixels.** The tests above check the paint the
// rim is given; this one paints a real piece of a real pack's glass and
// reads the rim back, because a gradient computed correctly and then
// handed to the wrong ring would pass every one of them.
func TestTheRimIsBrighterOnTheLitSide(t *testing.T) {
	pack, ok := LoadTheme("tahoe-night")
	if !ok {
		t.Skipf("the %q pack is not in this build", "tahoe-night")
	}
	restoreGlassLight(t)
	was := GlassAvailable()
	SetGlassAvailable(true)
	t.Cleanup(func() { SetGlassAvailable(was) })
	lk := pack.Look()
	const w, h = 200, 160

	// brightest scans a short column at an inner edge: the rim is a
	// hairline and which row it lands on moves with the corner radius.
	brightest := func(img *paintengine2d.Image, y0, y1 int) int {
		m := 0
		for y := y0; y < y1; y++ {
			for x := w/2 - 3; x < w/2+3; x++ {
				i := (y*img.Width + x) * 4
				if v := int(img.Pix[i]) + int(img.Pix[i+1]) + int(img.Pix[i+2]); v > m {
					m = v
				}
			}
		}
		return m
	}
	rim := func() (top, bottom int) {
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#404a6a")))
		lk.DrawMenuFrame(ctx, paintengine2d.XYWH(20, 20, w-40, h-40))
		img.Touch()
		return brightest(img, 20, 28), brightest(img, h-28, h-20)
	}

	ClearGlassLight()
	unlitTop, unlitBottom := rim()
	if unlitTop <= unlitBottom {
		t.Fatalf("with no light the rim is %d at the top and %d at the bottom; it is not top-lit, so this pack cannot show the effect", unlitTop, unlitBottom)
	}

	SetGlassLight(paintengine2d.Pt(w/2, -200))
	if top, bottom := rim(); top <= bottom {
		t.Errorf("lit from above: the rim is %d at the top and %d at the bottom", top, bottom)
	}

	// The one that matters: the highlight moves to the other edge.
	SetGlassLight(paintengine2d.Pt(w/2, h+200))
	top, bottom := rim()
	if bottom <= top {
		t.Errorf("lit from below: the rim is %d at the top and %d at the bottom; the highlight did not move", top, bottom)
	}
	// And it moves by as much as it was lit by in the first place, rather
	// than by a pixel of rounding.
	if swing := bottom - top; swing < (unlitTop-unlitBottom)/2 {
		t.Errorf("the highlight swung by %d, against the %d it is lit by overhead", swing, unlitTop-unlitBottom)
	}
}

// **A pane of Tahoe's glass bends what is behind it.** Over striped
// content, the stripes inside the pane line up with the ones outside
// through its flat middle and stop lining up inside its bevel, which is
// what a lens does and what no amount of tint can imitate.
//
// It is on the tool bar groups and the sidebar rather than the menus. The
// menus were the first guess and were wrong: their backdrop is blurred at
// sigma 14, and a backdrop blurred that hard is a flat colour, so bending
// it gives the same flat colour back — measured, the pane's interior had
// a spread of zero. These panes are clear instead, an eighth of an alpha
// over the window's own content, so there is something sharp to bend.
func TestTahoesGlassBendsTheContentBehindIt(t *testing.T) {
	pack, ok := LoadTheme("tahoe-night")
	if !ok {
		t.Skipf("the %q pack is not in this build", "tahoe-night")
	}
	was := GlassAvailable()
	SetGlassAvailable(true)
	t.Cleanup(func() { SetGlassAvailable(was) })
	lk := pack.Look()
	const w, h = 260, 110
	const period, bar = 14, 7

	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(Hex("#13214a")))
	for x := 0; x < w; x += period {
		ctx.DrawRect(paintengine2d.XYWH(float32(x), 0, bar, h), paintengine2d.Fill(Hex("#b9c9f5")))
	}
	pane := paintengine2d.XYWH(30, 32, w-60, h-64)
	DrawToolGroupOf(lk, ctx, pane, 3)
	img.Touch()

	// A stripe is where the backdrop was painted light; "lit" is where
	// the pixel reads light now. Through the flat middle the two agree,
	// because a clear tint shifts every pixel the same way.
	y := h / 2
	lit := func(x int) bool {
		i := (y*img.Width + x) * 4
		return int(img.Pix[i])+int(img.Pix[i+1])+int(img.Pix[i+2]) > 420
	}
	striped := func(x int) bool { return x%period < bar }
	count := func(x0, x1 int) (agree, total int) {
		for x := x0; x < x1; x++ {
			total++
			if lit(x) == striped(x) {
				agree++
			}
		}
		return agree, total
	}

	// The middle of the pane: the stripes still line up.
	midA, midN := count(int(pane.Min.X)+40, int(pane.Max.X)-40)
	if midN < 20 || midA*10 < midN*8 {
		t.Fatalf("through the pane's middle only %d of %d pixels line up with the backdrop; the tint is not clear enough for this test to mean anything", midA, midN)
	}
	// Its bevel: they stop lining up, because the content is displaced.
	bevA, bevN := count(int(pane.Min.X)+2, int(pane.Min.X)+14)
	if bevN < 8 {
		t.Fatal("the bevel band is too narrow to measure")
	}
	if bevA*10 >= bevN*8 {
		t.Errorf("inside the bevel %d of %d pixels still line up with the backdrop (the middle: %d of %d) — the pane is not bending anything",
			bevA, bevN, midA, midN)
	}
}
