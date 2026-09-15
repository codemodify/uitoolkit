package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

var win31PackNames = []string{"win31", "win-hotdog"}

func TestWin31PacksRegisteredInYearOrder(t *testing.T) {
	retroCheckPacks(t, "win31", []retroPack{
		{"win31", "Windows 3.1", "Windows", 1992},
		{"win-hotdog", "Hot Dog Stand", "Windows", 1992},
	})
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	if pos["win31"] > pos["win95"] || pos["win-hotdog"] > pos["win95"] {
		t.Fatal("the 3.1 packs list before Windows 95")
	}
}

func TestWin31PaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, win31PackNames...)
}

// Windows puts OK first; fields show the caret only; 17px bars.
func TestWin31StyleHints(t *testing.T) {
	for _, n := range win31PackNames {
		lk := retroLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintFormLabelsRight) != 0 {
			t.Fatalf("%s: want OK Cancel and left-aligned labels", n)
		}
		if lk.eng().FieldFocusRing(lk) {
			t.Fatalf("%s: edit fields show the caret only", n)
		}
		if s := ScrollBarStyleOf(lk); s.Arrows != ArrowsEnds || s.Thickness != 17 || s.MinThumb != 17 {
			t.Fatalf("%s: scroll bar %+v", n, s)
		}
	}
}

// XOR focus dots on the 16-colour palette: dark grey on the button face,
// black on white, yellow on the navy highlight.
func TestWin31FocusIsXORed(t *testing.T) {
	for _, tc := range [][2]string{{"#c0c0c0", "#808080"}, {"#ffffff", "#000000"}, {"#000080", "#ffff00"}, {"#ff0000", "#008080"}} {
		if got := colorHexPadded(w31xor(Hex(tc[0]))); got != tc[1] {
			t.Fatalf("xor of %s = %s, want %s", tc[0], got, tc[1])
		}
	}
}

// The push button: an outline without its corner pixels, two pixels of
// bevel, and a second black rectangle on the default button.
func TestWin31ButtonShape(t *testing.T) {
	lk := retroLook(t, "win31", 1)
	black, white, grey := Hex("#000000"), Hex("#ffffff"), Hex("#808080")
	paint := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(100, 40)
		lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(4, 4, 80, 28), st, "")
		return img
	}
	img := paint(StateNone)
	if _, _, _, a := img.PremulAt(4, 4); a != 0 {
		t.Fatal("the outline's corner pixel is painted")
	}
	if !retroNear(img, 40, 4, black) || !retroNear(img, 40, 5, white) || !retroNear(img, 40, 6, white) {
		t.Fatal("top: black outline over two white rows")
	}
	if !retroNear(img, 40, 31, black) || !retroNear(img, 40, 30, grey) || !retroNear(img, 40, 29, grey) {
		t.Fatal("bottom: two grey rows over the black outline")
	}
	img = paint(StatePrimary)
	if !retroNear(img, 4, 18, black) || !retroNear(img, 5, 18, black) || !retroNear(img, 6, 18, white) {
		t.Fatal("the default button has two pixels of black")
	}
	img = paint(StatePressed)
	if !retroNear(img, 40, 5, grey) || !retroNear(img, 40, 6, Hex("#c0c0c0")) {
		t.Fatal("pressed: one grey line on top, then the face")
	}
}

// The scroll thumb is a square as long as the bar is thick, whatever the
// proportion of the view.
func TestWin31ThumbIsSquare(t *testing.T) {
	lk := retroLook(t, "win31", 1)
	for _, content := range []float32{220, 2000} {
		img := paintengine2d.NewImage(40, 220)
		ctx := paintengine2d.NewContext(img)
		view := paintengine2d.XYWH(0, 0, 17, 200)
		p := ScrollGeometry(lk, view, true, content, 200, 10, false)
		DrawScrollBarParts(lk, ctx, p, true, ScrollState{})
		// The thumb's left highlight runs down its face, inside the bar's
		// black line.
		white := 0
		for y := 18; y < 182; y++ {
			if retroNear(img, 1, y, Hex("#ffffff")) {
				white++
			}
		}
		if white < 13 || white > 16 {
			t.Fatalf("content %v: thumb face %d rows, want the 15-pixel square", content, white)
		}
	}
}

// The control-menu box sits at the left of the caption, where
// WindowCloseRect reports it: grey, with the bar glyph, navy beyond.
func TestWin31ControlMenuBoxAgreesWithPaint(t *testing.T) {
	for _, n := range win31PackNames {
		for _, scale := range []float32{1, 2} {
			lk := retroLook(t, n, scale)
			c := w31colors(lk)
			b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: control-menu box %v not at the left of the caption (top %v)", n, scale, cr, in.Top)
			}
			img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			u := rpU(lk)
			if !retroNear(img, int(cr.Min.X+2*u), int(cr.Min.Y+2*u), c.face) {
				t.Fatalf("%s@%gx: control-menu box face missing", n, scale)
			}
			if !retroNear(img, int(cr.Max.X+3*u), int(cr.Min.Y+2*u), c.caption) {
				t.Fatalf("%s@%gx: caption colour missing beside the box", n, scale)
			}
			ctr := cr.Center()
			found := false
			for dy := -3; dy <= 3; dy++ {
				if retroNear(img, int(ctr.X), int(ctr.Y)+dy*int(u), c.frame) {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s@%gx: no bar glyph in the control-menu box", n, scale)
			}
		}
	}
}

// Labels read on what the engine paints them on (Hot Dog Stand's white on
// red is its famous 4:1).
func TestWin31LabelsReadable(t *testing.T) {
	for _, n := range win31PackNames {
		lk := retroLook(t, n, 1)
		c := w31colors(lk)
		body := 4.5
		if n == "win-hotdog" {
			body = 3.9
		}
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"window text", c.text, c.win, body},
			{"button text", c.btnText, c.face, 7},
			{"selected text", c.selText, c.sel, 7},
			{"caption text", c.captionText, c.caption, 7},
			{"inactive caption", c.captionOffT, c.captionOff, body},
			{"menu text", c.menuText, c.menu, 7},
			{"tooltip", c.infoText, c.info, 7},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Windows 3.1 is drawn in whole pixels of its 16-colour palette.
func TestWin31IsWholePixel(t *testing.T) {
	retroWholePixels(t, "win31", 8)
}
