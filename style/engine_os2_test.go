package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestOS2PackRegisteredInYearOrder(t *testing.T) {
	retroCheckPacks(t, "os2", []retroPack{{"os2warp", "OS/2 Warp 4", "IBM", 1996}})
	p, _ := LoadTheme("os2warp")
	if p.Tokens.Bevel != BevelClassic3D {
		t.Fatalf("bevel %q", p.Tokens.Bevel)
	}
}

func TestOS2PaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, "os2warp")
}

// Warp 4 is drawn in whole pixels, in its handful of system colours and
// the seven tab pastels.
func TestOS2IsWholePixel(t *testing.T) {
	retroWholePixels(t, "os2warp", 6+len(o2tabColors))
}

// Dialogs put the default button first (OK, Cancel, Help, bottom left)
// and left-align form labels; nothing floats on a shadow.
func TestOS2StyleHints(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "os2warp", scale)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintFormLabelsRight) != 0 || LookHint(lk, HintTabsCentered) != 0 {
			t.Fatalf("@%gx: want OK Cancel, left-aligned labels, left tabs", scale)
		}
		if lk.eng().FieldFocusRing(lk) {
			t.Fatal("entry fields show focus with the cursor alone")
		}
		if s := ScrollBarStyleOf(lk); s.Arrows != ArrowsEnds || s.Overlay || s.Thickness != 15*lk.Scale() {
			t.Fatalf("@%gx: scroll bar %+v", scale, s)
		}
		for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
			if !lk.PopupShadow(k).Zero() {
				t.Fatalf("@%gx: popup kind %d casts a shadow", scale, k)
			}
		}
		if !lk.TabOutset().Zero() {
			t.Fatal("Warp 4's tabs keep their size when selected")
		}
		if f := lk.ControlFont(RoleButton); f != lk.Font() {
			t.Fatal("controls are labelled in the body font")
		}
	}
}

// The Close glyph sits right of the title well, where hit-testing looks for
// it: a raised square with an engraved diagonal from its top right to its
// bottom left, sunk while pressed.
func TestOS2CloseGlyphAgreesWithPaint(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "os2warp", scale)
		c := o2colors(lk)
		u := rpU(lk)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Dx() != o2glyph*u || cr.Min.X < b.Min.X+b.Dx()*0.8 || cr.Max.X > b.Max.X-o2border*u || cr.Min.Y < b.Min.Y+o2above*u || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("@%gx: close rect %v not right of the title well of %v (top %v)", scale, cr, b, in.Top)
		}
		img := paintengine2d.NewImage(int(b.Max.X+8*scale), int(b.Max.Y+8*scale))
		lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "Dialog", WindowState{Active: true, CanClose: true})
		at := func(cx, cy int) (int, int) {
			return int(cr.Min.X + (float32(cx)+0.5)*u), int(cr.Min.Y + (float32(cy)+0.5)*u)
		}
		for _, p := range []struct {
			what   string
			cx, cy int
			want   paintengine2d.Color
		}{
			{"top edge", 5, 0, c.hi}, {"left edge", 0, 5, c.hi},
			{"right edge", o2glyph - 1, 5, c.dark}, {"bottom edge", 5, o2glyph - 1, c.dark},
			{"diagonal", 7, 6, c.dark}, {"diagonal", 11, 2, c.dark}, {"diagonal", 2, 11, c.dark},
			{"diagonal's light side", 7, 7, c.hi}, {"face", 4, 4, c.face}, {"face", 9, 9, c.face},
			{"gap before the glyph", -1, 5, c.face},
			{"the well's white edge", -4, 5, c.hi}, {"the well", -8, 5, c.title},
		} {
			x, y := at(p.cx, p.cy)
			if !retroNear(img, x, y, p.want) {
				t.Fatalf("@%gx: close glyph %s at cell (%d,%d) is not %v", scale, p.what, p.cx, p.cy, p.want)
			}
		}
		img2 := paintengine2d.NewImage(img.Width, img.Height)
		lk.DrawWindowFrame(paintengine2d.NewContext(img2), b, "Dialog", WindowState{Active: true, CanClose: true, ClosePress: true})
		if x, y := at(5, 0); !retroNear(img2, x, y, c.dark) {
			t.Fatalf("@%gx: a pressed close glyph does not sink", scale)
		}
		if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 40, 30)); !r.Empty() {
			t.Fatalf("@%gx: a frame too small for a title row reports close rect %v", scale, r)
		}
	}
}

// Push buttons sit in a groove; the default border — two pixels of black —
// replaces it on the default button and follows the keyboard focus.
func TestOS2DefaultBorderFollowsFocus(t *testing.T) {
	lk := retroLook(t, "os2warp", 1)
	c := o2colors(lk)
	b := paintengine2d.XYWH(2, 2, 76, 28)
	paint := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(80, 32)
		lk.DrawButton(paintengine2d.NewContext(img), b, st, "")
		return img
	}
	normal := paint(StateNone)
	if _, _, _, a := normal.PremulAt(30, 2); a != 0 {
		t.Fatal("a normal button paints the default border's outer ring")
	}
	if !retroNear(normal, 30, 3, c.dark) || !retroNear(normal, 30, 4, c.hi) || !retroNear(normal, 30, 28, c.hi) || !retroNear(normal, 30, 27, c.dark) {
		t.Fatal("a normal button is not a groove round a raised bevel")
	}
	for _, st := range []ControlState{StatePrimary, StateFocused, StatePrimary | StateFocused} {
		img := paint(st)
		if !retroNear(img, 30, 2, c.black) || !retroNear(img, 30, 3, c.black) || !retroNear(img, 30, 29, c.black) || !retroNear(img, 2, 15, c.black) {
			t.Fatalf("state %#x: no two-pixel black border", st)
		}
	}
	if img := paint(StatePressed | StateHovered); !retroNear(img, 30, 4, c.dark) || !retroNear(img, 30, 27, c.hi) {
		t.Fatal("a pressed button's bevel does not invert")
	}
}

// Disabled controls are halftoned, not embossed: every other pixel of the
// tick goes back to the face and nothing white is added.
func TestOS2DisabledIsHalftoned(t *testing.T) {
	lk := retroLook(t, "os2warp", 1)
	c := o2colors(lk)
	count := func(st ControlState, col paintengine2d.Color) int {
		img := paintengine2d.NewImage(24, 24)
		lk.eng().CheckIndicator(lk, paintengine2d.NewContext(img), paintengine2d.XYWH(2, 2, 15, 15), st, true)
		return countNear(img, col)
	}
	on, off := count(StateNone, c.black), count(StateDisabled, c.black)
	if on < 20 || off*10 < on*3 || off*10 > on*7 {
		t.Fatalf("disabled tick keeps %d of %d black pixels, want about half", off, on)
	}
	if count(StateDisabled, c.hi) > count(StateNone, c.hi) {
		t.Fatal("a disabled check box is embossed")
	}
}

// The selected notebook tab has no line under it: its colour runs into the
// page below its rows; an unselected tab leaves the page's top edge to the
// tab bar.
func TestOS2SelectedTabRunsIntoThePage(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "os2warp", scale)
		c := o2colors(lk)
		u := rpU(lk)
		b := paintengine2d.XYWH(0, 0, 100*scale, 30*scale)
		col := c.tabs[o2tabIndex(b.Min.X)]
		for _, sel := range []bool{false, true} {
			img := paintengine2d.NewImage(int(b.Dx()), int(b.Dy()))
			ctx := paintengine2d.NewContext(img)
			lk.DrawTabBar(ctx, b)
			lk.DrawTab(ctx, b, StateNone, "General", sel)
			y := int(b.Max.Y - float32(o2lip)*u + u*0.5) // the page's top row
			x := int(b.Dx() * 0.5)
			if got := retroNear(img, x, y, col); got != sel {
				t.Fatalf("@%gx selected=%v: tab colour under the tab = %v", scale, sel, got)
			}
			if !sel && !retroNear(img, x, y, c.hi) {
				t.Fatalf("@%gx: the page's top edge is missing under an unselected tab", scale)
			}
			// The white top edge, and the pastel face left of and below the label.
			if !retroNear(img, x, int(u*0.5), c.hi) || !retroNear(img, int(3.5*u), y-int(2*u), col) {
				t.Fatalf("@%gx: a tab is not a white-topped pastel trapezoid", scale)
			}
			if _, _, _, a := img.PremulAt(1, int(u*1.5)); a == 0 {
				t.Fatalf("@%gx: the tab bar leaves its corner unpainted", scale)
			}
		}
	}
}

// Labels read on every fill the engine paints them on; static text and
// group titles are navy, the Label widget's colour.
func TestOS2LabelsReadable(t *testing.T) {
	lk := retroLook(t, "os2warp", 1)
	c := o2colors(lk)
	if colorHexPadded(lk.Palette().Text) != "#000080" || colorHexPadded(c.static) != "#000080" {
		t.Fatalf("static text %s, want navy", colorHexPadded(lk.Palette().Text))
	}
	type check struct {
		what   string
		fg, bg paintengine2d.Color
		min    float64
	}
	checks := []check{
		{"static text on the face", c.static, c.face, 7},
		{"control text on the face", c.text, c.face, 7},
		{"entry field text", c.text, c.field, 7},
		{"hot menu item", c.menuTxt, c.menuHi, 7},
		{"active title", c.titleTxt, c.title, 7},
		{"inactive title", c.titleOffTxt, c.titleOff, 4.5},
		// OS/2's own white on #808080 selection, 3.95:1.
		{"selected item", c.selTxt, c.sel, 3.9},
		{"placeholder", c.dim, c.field, 3.5},
		{"fly-over help", c.black, c.info, 7},
	}
	for i, tab := range c.tabs {
		checks = append(checks, check{"tab " + itoa(i) + " label", c.text, tab, 7})
	}
	for _, chk := range checks {
		if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
			t.Errorf("%s: contrast %.2f < %.1f", chk.what, r, chk.min)
		}
	}
	// The group box title is painted navy.
	img := paintengine2d.NewImage(200, 80)
	lk.DrawGroupBox(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 200, 80), "Options", false)
	if countNear(img, c.static) < 20 {
		t.Fatal("the group box title is not navy")
	}
}
