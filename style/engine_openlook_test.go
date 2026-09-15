package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestOpenLookPackRegistered(t *testing.T) {
	retroCheckPacks(t, "openlook", []retroPack{{"openlook", "OPEN LOOK", "Sun", 1988}})
}

func TestOpenLookPaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, "openlook")
}

// Notices put the default button first; property windows right-align
// their labels; the 19px elevator bar.
func TestOpenLookStyleHints(t *testing.T) {
	lk := retroLook(t, "openlook", 1)
	if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintFormLabelsRight) != 1 {
		t.Fatal("want the default first and right-aligned labels")
	}
	if s := ScrollBarStyleOf(lk); s.Thickness != 19 || s.Arrows != ArrowsEnds || s.MinThumb != 47 {
		t.Fatalf("scrollbar %+v", s)
	}
	if !lk.PopupShadow(PopupMenu).Zero() {
		t.Fatal("3D OPEN LOOK menus cast no shadow")
	}
}

// OLGX derives the 3D colours from the window colour: BG2 90%, BG3 50%,
// the highlight 120% of its brightness.
func TestOpenLookColoursDeriveFromTheWindowColour(t *testing.T) {
	c := olColors(retroLook(t, "openlook", 1))
	for _, tc := range []struct {
		got  paintengine2d.Color
		want string
	}{{c.bg1, "#cccccc"}, {c.bg2, "#b7b7b7"}, {c.bg3, "#666666"}, {c.hi, "#f4f4f4"}} {
		if colorHexPadded(tc.got) != tc.want {
			t.Fatalf("got %s want %s", colorHexPadded(tc.got), tc.want)
		}
	}
	// A saturated colour past full brightness keeps its hue, halving its
	// saturation instead.
	h := olShade(Hex("#4080e0"), 1.2)
	if !(h.B >= h.G && h.G >= h.R) || h.B < 0.99 {
		t.Fatalf("highlight of a blue %s", colorHexPadded(h))
	}
}

// The check mark leaves the box at its top right.
func TestOpenLookCheckLeavesTheBox(t *testing.T) {
	lk := retroLook(t, "openlook", 1)
	img := paintengine2d.NewImage(30, 30)
	ctx := paintengine2d.NewContext(img)
	box := paintengine2d.XYWH(5, 5, 16, 16)
	lk.eng().CheckIndicator(lk, ctx, box, StateNone, true)
	// The box is 14 cells from row 7; the tick's long arm ends at the top
	// right corner of the indicator, above and right of the box.
	if !retroNear(img, 20, 5, Hex("#000000")) && !retroNear(img, 19, 5, Hex("#000000")) {
		t.Fatal("the check mark does not leave the box")
	}
	img2 := paintengine2d.NewImage(30, 30)
	lk.eng().CheckIndicator(lk, paintengine2d.NewContext(img2), box, StateNone, false)
	if _, _, _, a := img2.PremulAt(20, 5); a != 0 {
		t.Fatal("an unchecked box paints outside its square")
	}
}

// The elevator keeps its size whatever the proportion; the cable shows it.
func TestOpenLookElevatorIsFixed(t *testing.T) {
	lk := retroLook(t, "openlook", 1)
	c := olColors(lk)
	measure := func(content float32) (elev, dark int) {
		img := paintengine2d.NewImage(30, 320)
		ctx := paintengine2d.NewContext(img)
		view := paintengine2d.XYWH(0, 0, 19, 300)
		DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, view, true, content, 300, 0, false), true, ScrollState{})
		for y := 0; y < 300; y++ {
			// The elevator's highlight edge runs down its left side.
			if retroNear(img, 2, y, c.hi) {
				elev++
			}
			if retroNear(img, 9, y, c.bg3) && retroNear(img, 9, y+1, c.bg3) {
				dark++
			}
		}
		return
	}
	e1, d1 := measure(400)
	e2, d2 := measure(3000)
	if e1 != e2 || e1 < 40 {
		t.Fatalf("elevator %d vs %d rows: it must not change size", e1, e2)
	}
	if d1 <= d2 {
		t.Fatalf("the cable's proportion indicator (%d vs %d) must shrink with the proportion", d1, d2)
	}
}

// The pushpin is where WindowCloseRect says, in the header at the left.
func TestOpenLookPushpinAgreesWithPaint(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "openlook", scale)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.25 || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("@%gx pin %v not at the left of the header (top %v)", scale, cr, in.Top)
		}
		img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
		ctx := paintengine2d.NewContext(img)
		lk.DrawWindowFrame(ctx, b, "Pop-up", WindowState{Active: true, CanClose: true})
		// The pin's shaded edges (BG3 and the highlight) stand out of the
		// header band.
		c := olColors(lk)
		dark := 0
		for y := int(cr.Min.Y); y < int(cr.Max.Y); y++ {
			for x := int(cr.Min.X); x < int(cr.Max.X); x++ {
				if retroNear(img, x, y, c.bg3) || retroNear(img, x, y, c.hi) {
					dark++
				}
			}
		}
		if dark < int(8*scale) {
			t.Fatalf("@%gx no pushpin drawn in %v", scale, cr)
		}
	}
}

func TestOpenLookLabelsReadable(t *testing.T) {
	lk := retroLook(t, "openlook", 1)
	c := olColors(lk)
	for _, chk := range []struct {
		what   string
		fg, bg paintengine2d.Color
		min    float64
	}{
		{"text on window", c.text, c.bg1, 7},
		{"text on pressed", c.text, c.bg2, 7},
		{"text on field", lk.fieldText(), c.field, 7},
		{"reverse video", c.selTxt, c.sel, 7},
		{"muted text", c.dim, c.bg1, 4.5},
	} {
		if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
			t.Errorf("%s contrast %.2f < %.1f", chk.what, r, chk.min)
		}
	}
}

// OPEN LOOK is drawn in whole pixels of its four 3D shades, black and white.
func TestOpenLookIsWholePixel(t *testing.T) {
	retroWholePixels(t, "openlook", 8)
}
