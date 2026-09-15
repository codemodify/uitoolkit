package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestBeOSPackRegisteredInYearOrder(t *testing.T) {
	retroCheckPacks(t, "beos", []retroPack{{"beos", "BeOS", "Be", 1998}})
}

func TestBeOSPaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, "beos")
}

// The chrome is whole pixels, also at fractional origins. The close and
// zoom boxes are the only gradients, so the colour budget allows their
// shades on top of the flat greys.
func TestBeOSIsWholePixel(t *testing.T) {
	retroWholePixels(t, "beos", 80)
}

// BAlert puts the default button last (rightmost); labels sit left of
// their fields; text controls ring their focus; the scroll bar has an arrow
// box at each end; nothing floats on a shadow.
func TestBeOSStyleHints(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "beos", scale)
		if LookHint(lk, HintDialogPrimaryFirst) != 0 || LookHint(lk, HintFormLabelsRight) != 0 || LookHint(lk, HintTabsCentered) != 0 {
			t.Fatalf("@%gx: want Cancel OK, left labels, left tabs", scale)
		}
		if !lk.eng().FieldFocusRing(lk) {
			t.Fatalf("@%gx: a focused text control turns its frame blue", scale)
		}
		if s := ScrollBarStyleOf(lk); s.Arrows != ArrowsEnds || s.Overlay || s.Thickness != 15*scale || s.MinThumb != 15*scale {
			t.Fatalf("@%gx: scroll bar %+v", scale, s)
		}
		for _, kind := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
			if !lk.PopupShadow(kind).Zero() {
				t.Fatalf("@%gx: kind %d casts a shadow", scale, kind)
			}
		}
		if in := lk.TabOutset(); in.Left != 2*rpU(lk) || in.Right != 2*rpU(lk) || in.Top != 0 {
			t.Fatalf("@%gx: tab outset %+v", scale, in)
		}
		for _, r := range []Role{RoleButton, RoleCheck, RoleMenu, RoleTab, RoleCombo} {
			if lk.ControlFont(r) != lk.Font() {
				t.Fatalf("@%gx: role %d is not labelled in the plain font", scale, r)
			}
		}
	}
}

// The close box sits on the yellow tab at the frame's top-left, where hit
// testing looks for it: dark and light rings round a shaded yellow face. The
// tab is only as wide as its title; beside it the frame's top edge is bare.
func TestBeOSCloseBoxOnTheYellowTab(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "beos", scale)
		c := beColorsOf(lk)
		u := rpU(lk)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("@%gx: close rect %v not at the left of the tab of %v (top %v)", scale, cr, b, in.Top)
		}
		if cr.Dx() != 14*u || cr.Dy() != 14*u {
			t.Fatalf("@%gx: close box %v is not 14 pixels", scale, cr)
		}
		paint := func(ws WindowState) *paintengine2d.Image {
			img := paintengine2d.NewImage(int(b.Max.X+8*scale), int(b.Max.Y+8*scale))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", ws)
			return img
		}
		at := func(img *paintengine2d.Image, x, y float32, want paintengine2d.Color, what string) {
			t.Helper()
			if !retroNear(img, int(x), int(y), want) {
				r, g, bb, _ := img.PremulAt(int(x), int(y))
				t.Fatalf("@%gx: %s at (%v,%v) is %d,%d,%d", scale, what, x, y, r, g, bb)
			}
		}
		ctr := cr.Center()
		img := paint(WindowState{Active: true, CanClose: true})
		at(img, cr.Min.X, ctr.Y, c.boxDk, "close box outer ring (left)")
		at(img, cr.Min.X+u, cr.Min.Y, c.boxDk, "close box outer ring (top)")
		at(img, cr.Max.X-u, ctr.Y, c.boxLt, "close box outer ring (right)")
		at(img, cr.Min.X+u, cr.Min.Y+3*u, c.boxLt, "close box inner ring (left)")
		at(img, cr.Min.X+2*u, cr.Min.Y+2*u, Hex("#ffec21"), "close box face (top left)")
		at(img, ctr.X, ctr.Y, c.tab, "close box face (middle)")
		at(img, cr.Max.X-3*u, cr.Max.Y-3*u, Hex("#eab500"), "close box face (bottom right)")
		at(img, cr.Max.X+4*u, ctr.Y, Hex("#ffcb00"), "active tab")
		at(img, b.Max.X-20*u, ctr.Y, c.panel, "beside the tab")
		// The tab's bottom row is the frame's top line.
		at(img, cr.Max.X+4*u, b.Min.Y+float32(beTabH(lk)-1)*u, c.d2, "frame top line under the tab")
		// Pressed, the face shades the other way.
		img = paint(WindowState{Active: true, CanClose: true, ClosePress: true})
		at(img, cr.Min.X+2*u, cr.Min.Y+2*u, Hex("#eab500"), "pressed close box face")
		// Inactive: a grey tab, grey and white rings, a white-to-grey face.
		img = paint(WindowState{Active: false, CanClose: true})
		at(img, cr.Max.X+4*u, ctr.Y, Hex("#e8e8e8"), "inactive tab")
		at(img, cr.Min.X, ctr.Y, Hex("#a0a0a0"), "inactive close box ring")
		at(img, cr.Min.X+2*u, cr.Min.Y+2*u, Hex("#ffffff"), "inactive close box face")
		if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
			t.Fatalf("@%gx: a frame too small for a tab reports close rect %v", scale, r)
		}
	}
}

// The zoom box stands at the tab's right end, two pixels in from its edge
// lines, when the window can zoom: the back square's ring shows below and
// right of the front one.
func TestBeOSZoomBoxAtTheTabsRightEnd(t *testing.T) {
	lk := retroLook(t, "beos", 1)
	c := beColorsOf(lk)
	b := paintengine2d.XYWH(0, 0, 300, 180)
	plain := paintengine2d.NewImage(300, 180)
	lk.DrawWindowFrame(paintengine2d.NewContext(plain), b, "Dialog", WindowState{Active: true, CanClose: true})
	zoom := paintengine2d.NewImage(300, 180)
	lk.DrawWindowFrame(paintengine2d.NewContext(zoom), b, "Dialog", WindowState{Active: true, CanClose: true, Maximizable: true})
	// Find the tab's right edge on the zoomed frame: the #606060 line.
	y := beBoxY(lk) + 7
	edge := -1
	for x := 20; x < 300; x++ {
		if retroNear(zoom, x, 1, c.d4) {
			edge = x
			break
		}
	}
	if edge < 0 {
		t.Fatal("no tab edge")
	}
	zx := edge + 1 - 4 - 14
	if !retroNear(zoom, zx, beBoxY(lk)+3, c.zoomDk) || !retroNear(zoom, zx+13, y+6, c.boxLt) {
		t.Fatalf("zoom box rings missing at x=%d", zx)
	}
	if retroNear(plain, zx, beBoxY(lk)+3, c.zoomDk) {
		t.Fatal("a window that cannot zoom shows a zoom box")
	}
}

// A focused button, check box, radio and tab underline their label in the
// keyboard-navigation blue; a focused text control rings its inside in it.
func TestBeOSFocusIsBlue(t *testing.T) {
	lk := retroLook(t, "beos", 1)
	c := beColorsOf(lk)
	blue := func(draw func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState)) (int, int) {
		n := [2]int{}
		for i, st := range []ControlState{StateNone, StateFocused} {
			img := paintengine2d.NewImage(140, 32)
			draw(paintengine2d.NewContext(img), paintengine2d.XYWH(2, 2, 136, 28), st)
			for y := 0; y < img.Height; y++ {
				for x := 0; x < img.Width; x++ {
					if r, g, bb, _ := img.PremulAt(x, y); r == 0 && g == 0 && bb == 0xe5 {
						n[i]++
					}
				}
			}
		}
		return n[0], n[1]
	}
	for _, tc := range []struct {
		name string
		draw func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState)
		min  int
	}{
		{"button", func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawButton(ctx, b, st, "Button")
		}, 20},
		{"check box", func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawCheckbox(ctx, b, st, false, "Check")
		}, 20},
		{"radio", func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawRadio(ctx, b, st, false, "Radio")
		}, 20},
		{"tab", func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTab(ctx, b, st, "Tab", true)
		}, 12},
		{"text control", func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTextField(ctx, b, st, "Text", "", 0, 0, 0, false, 0, nil)
		}, 200},
	} {
		plain, focused := blue(tc.draw)
		if plain != 0 || focused < tc.min {
			t.Errorf("%s: %d blue pixels unfocused, %d focused (want 0 and ≥ %d)", tc.name, plain, focused, tc.min)
		}
	}
	// A focused list is framed in blue outside its border.
	img := paintengine2d.NewImage(120, 80)
	lk.DrawViewFrame(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 120, 80), StateFocused)
	if !retroNear(img, 60, 0, c.nav) || !retroNear(img, 0, 40, c.nav) || !retroNear(img, 60, 40, c.field) {
		t.Fatal("a focused list is not framed in blue round its white view")
	}
}

// Check boxes cross in blue; radio buttons hold a blue dot.
func TestBeOSMarksAreBlue(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := retroLook(t, "beos", scale)
		c := beColorsOf(lk)
		u := rpU(lk)
		side := lk.metrics.Checkbox
		img := paintengine2d.NewImage(int(40*scale), int(40*scale))
		ctx := paintengine2d.NewContext(img)
		box := paintengine2d.XYWH(4*scale, 4*scale, side, side)
		lk.eng().CheckIndicator(lk, ctx, box, StateNone, true)
		// The X's first stroke starts three cells in from the corner.
		if !retroNear(img, int(box.Min.X+3*u), int(box.Min.Y+3*u), c.nav) || !retroNear(img, int(box.Min.X+2*u), int(box.Min.Y+2*u), c.field) {
			t.Fatalf("@%gx: the check box's X is not blue on white", scale)
		}
		img = paintengine2d.NewImage(int(40*scale), int(40*scale))
		ctx = paintengine2d.NewContext(img)
		lk.eng().RadioIndicator(lk, ctx, box, StateNone, true)
		if !retroNear(img, int(box.Min.X+6*u), int(box.Min.Y+7*u), c.nav) {
			t.Fatalf("@%gx: the radio's dot is not blue", scale)
		}
	}
}

// Lists select in #B0B0B0 with black text, focused or not.
func TestBeOSSelectionKeepsItsGrey(t *testing.T) {
	lk := retroLook(t, "beos", 1)
	for _, st := range []ControlState{StateChecked, StateChecked | StateInactive, StateChecked | StateBackdrop} {
		img := paintengine2d.NewImage(120, 30)
		lk.DrawListRow(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 120, 24), st, "")
		if !retroNear(img, 60, 12, Hex("#b0b0b0")) {
			t.Fatalf("state %#x: selection is not #B0B0B0", uint32(st))
		}
	}
}

// Labels read on every fill the engine paints them on.
func TestBeOSLabelsReadable(t *testing.T) {
	lk := retroLook(t, "beos", 1)
	c := beColorsOf(lk)
	for _, chk := range []struct {
		what   string
		fg, bg paintengine2d.Color
		min    float64
	}{
		{"text on panel", c.text, c.panel, 7},
		{"button label", c.text, c.l1, 7},
		{"pressed button label", c.white, c.dnFace, 7},
		{"text in a field", c.fieldText, c.field, 7},
		{"list selection", c.rowText, c.rowSel, 4.5},
		{"menu selection", c.text, c.menuSel, 4.5},
		{"window title", c.text, c.tab, 7},
		{"inactive window title", c.tabOffText, c.tabOff, 4.5},
		{"tooltip", c.text, c.info, 7},
		{"secondary text", c.muted, c.panel, 4.5},
		{"accent label", lk.Palette().Accent, c.panel, 4.5},
		{"danger label", lk.Palette().Danger, c.panel, 4.5},
		{"placeholder", c.dim, c.field, 3},
		// BeOS greyed disabled labels to darken-3; they are meant to recede.
		{"disabled label", c.dim, c.panel, 2.5},
	} {
		if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
			t.Errorf("%s: contrast %.2f < %.1f", chk.what, r, chk.min)
		}
	}
}

// The bevel greys are the Interface Kit's tints of the panel colour.
func TestBeOSTintsOfThePanel(t *testing.T) {
	c := beColorsOf(retroLook(t, "beos", 1))
	for _, tc := range []struct {
		got  paintengine2d.Color
		want string
	}{{c.l2, "#f0f0f0"}, {c.l1, "#e8e8e8"}, {c.d1, "#b8b8b8"}, {c.d2, "#989898"}, {c.d3, "#808080"}, {c.d4, "#606060"}} {
		if w := Hex(tc.want); int(tc.got.R*255+0.5) != int(w.R*255+0.5) {
			t.Fatalf("tint %v, want %s", tc.got, tc.want)
		}
	}
}
