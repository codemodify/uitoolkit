package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestKDE2PackRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "kde2", "KDE", 2000, map[string]string{"kde2": "KDE 2"})
	pos := kdePositions()
	if p, ok := pos["kde1"]; ok && pos["kde2"] < p {
		t.Errorf("KDE 2 (2000) sorts before KDE 1 (1998): %v %v", pos["kde2"], p)
	}
	for _, later := range []string{"keramik", "plastik", "oxygen", "breeze"} {
		if p, ok := pos[later]; ok && pos["kde2"] > p {
			t.Errorf("KDE 2 (2000) sorts after %s: %v %v", later, pos["kde2"], p)
		}
	}
}

// The pack carries the colour scheme KDE 2's Control Center wrote as its
// default: #dcdcdc windows over #e4e4e4 buttons, the #0a5f89 blue for both
// the selection and the title bar, white views.
func TestKDE2CarriesTheKDE2Scheme(t *testing.T) {
	lk := mustLook(t, "kde2")
	p := lk.Palette()
	for _, c := range []struct {
		name string
		got  paintengine2d.Color
		want string
	}{
		{"background", p.Background, "#dcdcdc"},
		{"button", p.SurfaceAlt, "#e4e4e4"},
		{"base", p.Field, "#ffffff"},
		{"selection", p.Selection, "#0a5f89"},
		{"selected text", p.TextOnAccent, "#ffffff"},
		{"alternate", lk.X("alternate", paintengine2d.Color{}), "#f0f0f0"},
		{"caption", lk.X("caption", paintengine2d.Color{}), "#0a5f89"},
		{"outline", lk.X("outline", paintengine2d.Color{}), "#4e4e4e"},
		{"ring", lk.X("ring", paintengine2d.Color{}), "#b7b7b7"},
	} {
		if got := colorHexPadded(c.got); got != c.want {
			t.Errorf("%s = %s, want %s", c.name, got, c.want)
		}
	}
	if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintFormLabelsRight) != 0 {
		t.Error("KDE 2: want OK before Cancel and left-aligned form labels")
	}
	// KDE 2 had no compositor, so nothing floats over a shadow.
	for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
		if !lk.PopupShadow(k).Zero() {
			t.Errorf("KDE 2 popup kind %d has a shadow %+v", k, lk.PopupShadow(k))
		}
	}
}

// KDE's own scroll bar layout: a step-back button at the top and both
// buttons together at the bottom.
func TestKDE2ScrollBarHasThreeButtons(t *testing.T) {
	lk := mustLook(t, "kde2")
	s := ScrollBarStyleOf(lk)
	if s.Arrows != ArrowsTripleEnd || s.Overlay || s.Thickness != 18 {
		t.Fatalf("KDE 2 scroll bar %+v: want an 18px bar with three step buttons", s)
	}
	parts := ScrollGeometry(lk, paintengine2d.XYWH(0, 0, 200, 300), true, 900, 300, 0, false)
	if parts.Dec.Min.Y != 0 || parts.Track.Min.Y != parts.Dec.Max.Y || parts.DecEnd.Min.Y < 200 || parts.Inc.Min.Y <= parts.DecEnd.Min.Y {
		t.Fatalf("KDE 2 vertical bar parts %+v: want back at the top, back and forward together at the bottom", parts)
	}
}

// The push button is HighColor's slab: the dark outline, a ring, two rows
// of light, then a face that falls from the button colour lightened to the
// same darkened — and the corner pixel is left as background.
func TestKDE2ButtonIsASlab(t *testing.T) {
	lk := mustLook(t, "kde2")
	img := paintengine2d.NewImage(140, 48)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 140, 48), paintengine2d.Fill(lk.Palette().Background))
	b := paintengine2d.XYWH(10, 8, 110, 30)
	lk.DrawButton(ctx, b, StateNone, "")
	at := func(x, y int) paintengine2d.Color {
		r, g, bl, a := img.PremulAt(x, y)
		return paintengine2d.RGBA(float32(r)/255, float32(g)/255, float32(bl)/255, float32(a)/255)
	}
	x := 60
	y0 := int(b.Min.Y)
	for _, c := range []struct {
		name string
		got  paintengine2d.Color
		want string
	}{
		{"outline", at(x, y0), "#4e4e4e"},
		{"ring", at(x, y0+1), "#b7b7b7"},
		{"light", at(x, y0+2), "#ffffff"},
		{"light", at(x, y0+3), "#ffffff"},
		{"the bottom of the face", at(x, int(b.Max.Y)-3), "#cfcfcf"},
		{"the ring below it", at(x, int(b.Max.Y)-2), "#b7b7b7"},
		{"the outline below that", at(x, int(b.Max.Y)-1), "#4e4e4e"},
	} {
		if !kdeClose(c.got, Hex(c.want), 6) {
			t.Errorf("%s = %s, want about %s", c.name, colorHexPadded(c.got), c.want)
		}
	}
	// The corner is cut: the pixel at the top left is still the window.
	if got := at(int(b.Min.X), y0); !kdeClose(got, lk.Palette().Background, 6) {
		t.Errorf("the corner pixel is %s, want the window colour — HighColor cut it away", colorHexPadded(got))
	}
}

// HighColor's arrows have their tips cut flat: the point is a quarter of
// the base, not a single pixel.
func TestKDE2ArrowsAreFlatTopped(t *testing.T) {
	lk := mustLook(t, "kde2")
	img := paintengine2d.NewImage(40, 40)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 40, 40), paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	DrawArrowOf(lk, ctx, paintengine2d.XYWH(8, 8, 24, 24), DirUp, paintengine2d.RGB(0, 0, 0))
	ink := func(y int) int {
		n := 0
		for x := 0; x < 40; x++ {
			if r, _, _, _ := img.PremulAt(x, y); r < 128 {
				n++
			}
		}
		return n
	}
	top, base := 0, 0
	for y := 0; y < 40; y++ {
		if n := ink(y); n > 0 {
			if top == 0 {
				top = n
			}
			base = n
		}
	}
	if top < 2 {
		t.Fatalf("the arrow's first row has %d pixels: it comes to a point, not a flat top", top)
	}
	if base < top*2 {
		t.Fatalf("the arrow's base (%d) is not wider than its top (%d)", base, top)
	}
}

// The title bar is flat — KDE 2's scheme blended it into itself — and
// woven with the stipple: a lighter dot and a darker one every three
// pixels across and four down.
func TestKDE2TitleBarIsFlatAndStippled(t *testing.T) {
	lk := mustLook(t, "kde2")
	b := paintengine2d.XYWH(0, 0, 360, 200)
	img := paintengine2d.NewImage(int(b.Max.X), int(b.Max.Y))
	lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "", WindowState{Active: true})
	cap := lk.X("caption", paintengine2d.Color{})
	y0 := int(kw2Grab(lk) + kde3U(lk))
	y1 := int(kw2Grab(lk) + kw2BarH(lk))
	band := paintengine2d.XYWH(120, float32(y0), 120, float32(y1-y0))
	var flat, light, dark int
	for y := y0; y < y1; y++ {
		for x := 120; x < 240; x++ {
			r, g, bl, _ := img.PremulAt(x, y)
			c := paintengine2d.RGBA(float32(r)/255, float32(g)/255, float32(bl)/255, 1)
			switch {
			case kdeClose(c, cap, 4):
				flat++
			case Luma(c) > Luma(cap):
				light++
			default:
				dark++
			}
		}
	}
	total := int(band.Dx() * band.Dy())
	if flat < total/2 {
		t.Errorf("only %d of %d title pixels are the caption colour: the bar is not flat", flat, total)
	}
	// One light and one dark dot per 3×4 cell is a twelfth each; allow the
	// bar's top highlight line to skew it.
	if light < total/24 || dark < total/24 {
		t.Errorf("the stipple is missing: %d light and %d dark dots in %d pixels", light, dark, total)
	}
	if light > total/6 || dark > total/6 {
		t.Errorf("the stipple is too dense: %d light and %d dark dots in %d pixels", light, dark, total)
	}
}

// The armed menu row is a ramp under a dark line with a white label, which
// is what KDE 2.0 measured — not the navy a list row selects with.
func TestKDE2MenuArmsWithARampNotTheSelection(t *testing.T) {
	lk := mustLook(t, "kde2")
	img := paintengine2d.NewImage(220, 30)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(0, 0, 220, 26)
	lk.DrawMenuItem(ctx, b, StateHovered, MenuRow{Label: "Open"})
	sel := lk.Palette().Selection
	if n := countNearIn(img, b, sel); n > 4 {
		t.Errorf("the armed row is filled with the selection colour (%d pixels)", n)
	}
	top := func(y int) int {
		r, g, bl, _ := img.PremulAt(110, y)
		return int(r) + int(g) + int(bl)
	}
	if top(1) <= top(int(b.Max.Y)-3) {
		t.Errorf("the armed row does not darken downwards: %d then %d", top(1), top(int(b.Max.Y)-3))
	}
	if ContrastRatio(lk.Engine().MenuTextColor(lk, true), Hex("#999999")) < 1.9 {
		t.Error("the armed row's label does not stand out on the ramp")
	}
}

func TestKDE2CloseButtonAgreesWithPaint(t *testing.T) {
	kdeCheckClose(t, "kde2")
}

func TestKDE2PaintsEveryControlInsideItsRect(t *testing.T) {
	p, _ := LoadTheme("kde2")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		aquaExercise(t, fmt.Sprintf("kde2@%gx", scale), lk)
		kdeExtras(t, fmt.Sprintf("kde2@%gx", scale), lk)
	}
}
