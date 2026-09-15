package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

var amigaPackNames = []string{"amiga13", "amiga31"}

func TestAmigaPacksRegisteredInYearOrder(t *testing.T) {
	retroCheckPacks(t, "amiga", []retroPack{
		{"amiga13", "Workbench 1.3", "Amiga", 1987},
		{"amiga31", "Workbench 3.1", "Amiga", 1994},
	})
	for _, n := range amigaPackNames {
		p, _ := LoadTheme(n)
		if p.Tokens.Bevel != BevelClassic3D {
			t.Fatalf("%s: bevel %q", n, p.Tokens.Bevel)
		}
		for i := 0; i < 4; i++ {
			if _, ok := p.Tokens.Extra["pen"+itoa(i)]; !ok {
				t.Fatalf("%s: pack has no pen%d", n, i)
			}
		}
	}
	if !amColors(retroLook(t, "amiga13", 1)).wb13 || amColors(retroLook(t, "amiga31", 1)).wb13 {
		t.Fatal("the wb param does not select the look")
	}
}

func TestAmigaPaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, amigaPackNames...)
}

// Every control survives the tiny and odd sizes layout can hand it — the
// cell arithmetic must neither panic nor paint outside the rect.
func TestAmigaSurvivesTinyRects(t *testing.T) {
	type paint func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState)
	controls := []struct {
		name string
		draw paint
	}{
		{"button", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawButton(ctx, b, st, "OK")
		}},
		{"checkbox", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawCheckbox(ctx, b, st, true, "C")
		}},
		{"radio", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawRadio(ctx, b, st, true, "R")
		}},
		{"switch", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSwitch(ctx, b, st, true, "S")
		}},
		{"cycle", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawComboBox(ctx, b, st, "Choice", true)
		}},
		{"string", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTextField(ctx, b, st, "text", "", 2, 0, 2, true, 0, nil)
		}},
		{"area", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTextArea(ctx, b, st, []TextLine{{Text: "a", Start: 0, End: 1}}, 1, 0, 1, true, 0, 0, "", nil)
		}},
		{"spinner", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSpinner(ctx, b, st, true, false, true, false)
		}},
		{"slider", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSlider(ctx, b, st, 0.5)
		}},
		{"progress", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawProgressBar(ctx, b, st, 0.5, false, 0)
			lk.DrawProgressBar(ctx, b, st, 0, true, 0.3)
		}},
		{"tabs", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTabBar(ctx, b)
			lk.DrawTab(ctx, b, st, "Tab", true)
			lk.DrawTab(ctx, b, st, "Tab", false)
			lk.DrawTabPane(ctx, b)
		}},
		{"menus", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawMenuBar(ctx, b)
			lk.DrawMenuTitle(ctx, b, st, "File", 0, true)
			lk.DrawMenuFrame(ctx, b)
		}},
		{"rows", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawListRow(ctx, b, st, "Row")
			lk.DrawTreeRow(ctx, b, st, true, false, 3, "Node", false)
			lk.DrawTableCell(ctx, b, st, "Cell", AlignEnd, nil)
			lk.DrawTableHeader(ctx, b, st, "Name", true, true)
			lk.DrawItemFocus(ctx, b, st)
		}},
		{"tool", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawToolBar(ctx, b)
			lk.DrawToolButton(ctx, b, st, "Tool", IconSave)
		}},
		{"accordion", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawAccordionHeader(ctx, b, st, "Section", true)
		}},
		{"lines", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSplitter(ctx, b, true, st)
			lk.DrawSplitter(ctx, b, false, st)
			lk.DrawSeparator(ctx, b, true)
			lk.DrawSeparator(ctx, b, false)
		}},
		{"frames", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawPanel(ctx, b, true)
			lk.DrawGroupBox(ctx, b, "Group", false)
			lk.DrawGroupBox(ctx, b, "Group", true)
			lk.DrawViewFrame(ctx, b, st)
			lk.DrawFocusRing(ctx, b)
		}},
		{"window", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawWindowFrame(ctx, b, "Window", WindowState{Active: st == StateNone, CanClose: true, Maximizable: true, ClosePress: st.Pressed()})
		}},
		{"bars", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawStatusBar(ctx, b, []string{"a", "b", "c"})
			lk.DrawTitleBar(ctx, b, "Title", "sub")
			lk.DrawTooltip(ctx, b, "Tip")
		}},
		{"scroller", func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawArrow(ctx, b, DirLeft, Hex("#000000"))
			lk.DrawScrollBar(ctx, b, paintengine2d.XYWH(b.Min.X, b.Min.Y+b.Dy()*0.3, b.Dx(), b.Dy()*0.3), st)
		}},
	}
	sizes := []float32{0, 1, 3, 7, 12, 19, 27, 33, 45, 61}
	states := []ControlState{StateNone, StatePressed | StateHovered | StateFocused | StateChecked, StateDisabled}
	for _, n := range amigaPackNames {
		for _, scale := range []float32{1, 2} {
			lk := retroLook(t, n, scale)
			for _, c := range controls {
				for _, w := range sizes {
					for _, h := range sizes {
						w, h := w*scale, h*scale
						for _, st := range states {
							img := paintengine2d.NewImage(int(w)+20, int(h)+20)
							ctx := paintengine2d.NewContext(img)
							b := paintengine2d.XYWH(10, 10, w, h)
							c.draw(lk, ctx, b, st)
							if ctx.SaveCount() != 0 {
								t.Fatalf("%s@%gx %s %vx%v: %d saved states left", n, scale, c.name, w, h, ctx.SaveCount())
							}
							if x, y, ok := aquaOutside(img, b); ok {
								t.Fatalf("%s@%gx %s %vx%v state %#x: painted outside at (%d,%d)", n, scale, c.name, w, h, uint32(st), x, y)
							}
						}
					}
				}
			}
		}
	}
}

// Both Workbenches are four-colour screens drawn in whole pixels.
func TestAmigaIsWholePixel(t *testing.T) {
	retroWholePixels(t, "amiga13", 4)
	retroWholePixels(t, "amiga31", 4)
}

// Requesters put the positive choice first (lower left), GadTools
// right-aligns labels left of their gadgets, string gadgets show only their
// cursor, nothing casts a shadow and every gadget is labelled in Topaz's
// stand-in.
func TestAmigaStyleHints(t *testing.T) {
	for _, n := range amigaPackNames {
		lk := retroLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintFormLabelsRight) != 1 || LookHint(lk, HintTabsCentered) != 0 {
			t.Fatalf("%s: want OK first, right-aligned labels, left tabs", n)
		}
		if lk.eng().FieldFocusRing(lk) {
			t.Fatalf("%s: string gadgets show their cursor only", n)
		}
		for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
			if !lk.PopupShadow(k).Zero() {
				t.Fatalf("%s: kind %d casts a shadow", n, k)
			}
		}
		for _, r := range []Role{RoleButton, RoleCheck, RoleMenu, RoleTab, RoleCombo, RoleTool, RoleField} {
			if lk.ControlFont(r) != lk.MonoFont() {
				t.Fatalf("%s: role %v is not labelled in the monospace face", n, r)
			}
		}
		s := ScrollBarStyleOf(lk)
		if s.Overlay {
			t.Fatalf("%s: scrollers reserve their gutter", n)
		}
		if amColors(lk).wb13 {
			if s.Arrows != ArrowsNone {
				t.Fatalf("%s: 1.3's proportional gadgets have no arrows: %+v", n, s)
			}
		} else if s.Arrows != ArrowsTogetherEnd || s.Thickness != 18 {
			t.Fatalf("%s: 3.1's scrollers keep their arrows together at the end: %+v", n, s)
		}
	}
}

// The close gadget sits at the left end of the title bar, where hit-testing
// looks for it. 3.1: a black-outlined box with a white middle in a blue
// gadget, the middle black while pressed. 1.3: a blue box round a black dot
// on the white bar, the whole gadget complemented while pressed.
func TestAmigaCloseGadgetAgreesWithPaint(t *testing.T) {
	for _, n := range amigaPackNames {
		for _, scale := range []float32{1, 2} {
			lk := retroLook(t, n, scale)
			c := amColors(lk)
			u := rpU(lk)
			b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X != b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Min.Y != b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not at the left end of the title bar of %v (top %v)", n, scale, cr, b, in.Top)
			}
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s@%gx: a frame too small for a title bar reports close rect %v", n, scale, r)
			}
			paint := func(st WindowState) *paintengine2d.Image {
				img := paintengine2d.NewImage(int(b.Max.X+8*scale), int(b.Max.Y+8*scale))
				lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "Dialog", st)
				return img
			}
			img := paint(WindowState{Active: true, CanClose: true})
			pressed := paint(WindowState{Active: true, CanClose: true, ClosePress: true})
			ctr := cr.Center()
			cx, cy := int(ctr.X), int(ctr.Y)
			x0 := int(cr.Min.X)
			if c.wb13 {
				if !retroNear(img, cx, cy, c.dark) {
					t.Fatalf("%s@%gx: no black dot in the close gadget at %v", n, scale, cr)
				}
				if !retroNear(img, x0+int(2*u), cy, c.win) || !retroNear(img, x0, cy, c.line) {
					t.Fatalf("%s@%gx: the close gadget's blue box is missing", n, scale)
				}
				if !retroNear(pressed, cx, cy, c.line) || !retroNear(pressed, x0, cy, c.dark) {
					t.Fatalf("%s@%gx: a pressed close gadget is not complemented", n, scale)
				}
				continue
			}
			if !retroNear(img, cx, cy, c.shine) {
				t.Fatalf("%s@%gx: the close box's middle is not white at %v", n, scale, cr)
			}
			if !retroNear(img, x0+int(7*u), cy, c.shadow) || !retroNear(img, x0+int(4*u), cy, c.fill) {
				t.Fatalf("%s@%gx: the close box's outline or the gadget's blue is missing", n, scale)
			}
			if !retroNear(img, x0, cy, c.shine) || !retroNear(img, int(cr.Max.X)-1, cy, c.shadow) {
				t.Fatalf("%s@%gx: the close gadget is not a raised box", n, scale)
			}
			if !retroNear(pressed, cx, cy, c.shadow) || !retroNear(pressed, x0, cy, c.shadow) {
				t.Fatalf("%s@%gx: a pressed close gadget does not recess", n, scale)
			}
		}
	}
}

// Intuition's bevel: shine along the top and left, shadow along the bottom
// and right, the top-right corner pixel shadow and the bottom-left shine,
// horizontal edges one row (two cells) and vertical edges two pixels.
func TestAmigaBevelCorners(t *testing.T) {
	lk := retroLook(t, "amiga31", 1)
	c := amColors(lk)
	img := paintengine2d.NewImage(100, 40)
	b := paintengine2d.XYWH(0, 0, 80, 28)
	lk.DrawButton(paintengine2d.NewContext(img), b, StateNone, "")
	for _, p := range []struct {
		x, y int
		want paintengine2d.Color
		what string
	}{
		{0, 0, c.shine, "top-left"}, {78, 0, c.shine, "top row"}, {79, 0, c.shadow, "top-right corner"},
		{78, 2, c.shadow, "inner right column"}, {1, 2, c.shine, "inner left column"},
		{0, 27, c.shine, "bottom-left corner"}, {1, 27, c.shadow, "bottom row"}, {1, 26, c.shadow, "bottom row"},
		{40, 1, c.shine, "top edge, second cell"}, {40, 2, c.win, "face under the top edge"},
		{2, 14, c.win, "face inside the left edge"},
	} {
		if !retroNear(img, p.x, p.y, p.want) {
			t.Errorf("%s at (%d,%d) is not %v", p.what, p.x, p.y, p.want)
		}
	}
	// Pressed: recessed and FILLPEN blue.
	img = paintengine2d.NewImage(100, 40)
	lk.DrawButton(paintengine2d.NewContext(img), b, StatePressed|StateHovered, "")
	if !retroNear(img, 0, 0, c.shadow) || !retroNear(img, 79, 14, c.shine) || !retroNear(img, 40, 14, c.fill) {
		t.Error("a pressed button does not recess into FILLPEN")
	}
}

// 1.3 highlights by complementing the pens: a pressed button turns orange
// with a black outline, a selected row orange with black text.
func TestAmiga13HighlightsByComplement(t *testing.T) {
	lk := retroLook(t, "amiga13", 1)
	c := amColors(lk)
	blue, white, black, orange := c.pen[0], c.pen[1], c.pen[2], c.pen[3]
	if c.comp(blue) != orange || c.comp(white) != black || c.comp(orange) != blue || c.comp(black) != white {
		t.Fatal("the complement does not exclusive-or the pen numbers with 3")
	}
	img := paintengine2d.NewImage(100, 40)
	lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 80, 28), StateNone, "")
	if !retroNear(img, 0, 14, white) || !retroNear(img, 40, 14, blue) {
		t.Fatal("a 1.3 button is not a white outline round blue")
	}
	img = paintengine2d.NewImage(100, 40)
	lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 80, 28), StatePressed|StateHovered, "")
	if !retroNear(img, 0, 14, black) || !retroNear(img, 40, 14, orange) {
		t.Fatal("a pressed 1.3 button is not complemented")
	}
	img = paintengine2d.NewImage(120, 30)
	lk.DrawListRow(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 120, 24), StateChecked, "")
	if !retroNear(img, 60, 12, orange) {
		t.Fatal("a selected 1.3 row is not orange")
	}
}

// Disabled gadgets are ghosted with Intuition's dot grid: every fourth
// pixel of a row, alternate rows shifted by two, each Amiga row two cells
// tall.
func TestAmigaGhostsDisabledGadgets(t *testing.T) {
	lk := retroLook(t, "amiga31", 1)
	c := amColors(lk)
	img := paintengine2d.NewImage(100, 40)
	lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 80, 28), StateDisabled, "")
	dots := 0
	for y := 4; y < 24; y++ {
		for x := 4; x < 76; x++ {
			if retroNear(img, x, y, c.shadow) {
				dots++
				// A dot is a hires pixel: one cell wide, two tall.
				if retroNear(img, x+1, y, c.shadow) || retroNear(img, x-1, y, c.shadow) {
					t.Fatalf("ghost dot at (%d,%d) is wider than a pixel", x, y)
				}
				want := (y/2)&1 == 0 && x%4 == 0 || (y/2)&1 == 1 && x%4 == 2
				if !want {
					t.Fatalf("ghost dot at (%d,%d) is off the grid", x, y)
				}
			}
		}
	}
	if dots < 72*20/4-40 || dots > 72*20/4+40 {
		t.Fatalf("%d ghost dots over the face, want about a quarter of it", dots)
	}
}

// Labels read on every fill the engine paints them on.
func TestAmigaLabelsReadable(t *testing.T) {
	for _, n := range amigaPackNames {
		lk := retroLook(t, n, 1)
		c := amColors(lk)
		titleBar := c.fill
		if c.wb13 {
			titleBar = c.bar
		}
		pressed := c.fillText
		if c.wb13 {
			pressed = c.comp(c.text)
		}
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.win, 4.5},
			{"text on field", lk.fieldText(), lk.Palette().Field, 4.5},
			{"palette text", lk.Palette().Text, lk.Palette().Background, 4.5},
			{"selected row", c.fillText, c.fill, 4.5},
			{"pressed gadget", pressed, c.faceOf(true), 4.5},
			{"menu item", c.barText, c.bar, 4.5},
			{"hot menu item", c.menuHiText, c.menuHi, 4.5},
			{"selected text", c.markText, c.mark, 4.5},
			{"window title", c.barText, titleBar, 4.5},
			{"inactive window title", c.text, c.win, 4.5},
			{"placeholder", c.muted, c.win, 3},
			{"focus mark", c.focus, c.win, 3},
			{"focus mark on a selection", c.focusOn(c.fill), c.fill, 3},
			{"string cursor", c.curText, c.cur, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Menus show the Amiga key and the command letter for a one-key shortcut.
func TestAmigaMenuShortcuts(t *testing.T) {
	for _, tc := range []struct {
		in, key string
		ok      bool
	}{{"Ctrl+O", "O", true}, {"Ctrl+q", "Q", true}, {"Cmd-S", "S", true}, {"Ctrl+Shift+S", "", false}, {"F5", "", false}, {"Ctrl+F5", "", false}} {
		if key, ok := amShortcut(tc.in); key != tc.key || ok != tc.ok {
			t.Errorf("amShortcut(%q) = %q %v, want %q %v", tc.in, key, ok, tc.key, tc.ok)
		}
	}
}
