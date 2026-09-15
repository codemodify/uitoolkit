package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var motifPackNames = []string{
	"motif", "hp-vue", "cde", "cde-alpine", "cde-broica", "cde-charcoal", "cde-crimson", "cde-desert", "irix",
}

func TestMotifPacksRegisteredInYearOrder(t *testing.T) {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			t.Fatalf("pack %q listed twice", n)
		}
		pos[n] = i
	}
	years := map[string]int{"motif": 1990, "hp-vue": 1990, "irix": 1993}
	for _, n := range motifPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		want := 1993
		if y, ok := years[n]; ok {
			want = y
		}
		if p.Tokens.Engine != "motif" || p.Lineage != "Unix" || p.Year != want || p.Summary == "" || p.Label == "" {
			t.Fatalf("%s: engine %q lineage %q year %d summary %q", n, p.Tokens.Engine, p.Lineage, p.Year, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "motif" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	// The legacy era packs of the same names are replaced, not duplicated.
	if p, _ := LoadTheme("cde"); p.Label != "CDE Default" {
		t.Fatalf("cde pack is %q", p.Label)
	}
	// NeXT (1989) < Motif, HP VUE (1990) < CDE (1993) < Windows 95 (1995)
	// < Platinum (1997).
	for _, o := range [][2]string{{"next", "motif"}, {"motif", "cde"}, {"hp-vue", "cde-alpine"}, {"cde", "win95"}, {"irix", "win95"}, {"cde-desert", "platinum"}} {
		if pos[o[0]] > pos[o[1]] {
			t.Fatalf("%s sorts after %s: %v", o[0], o[1], pos)
		}
	}
}

// motifDerive must be XmGetColors: the values below were read off real
// screens — mwm's default frames (LightGrey, CadetBlue) and the CDE
// Default palette's window colour set as the CDE colour server drew it.
func TestMotifDeriveMatchesXmGetColors(t *testing.T) {
	cases := []struct {
		bg              string
		fg, ts, bs, sel string // "" = not observed
	}{
		{bg: "#d3d3d3", fg: "#000000", bs: "#767676"},
		{bg: "#5f9ea0", fg: "#ffffff", bs: "#2f4f50"},
		{bg: "#c600b2d2a87e", fg: "#000000", ts: "#e7deda", bs: "#6a605a", sel: "#a8988f"},
	}
	for _, c := range cases {
		fg, ts, bs, sel := motifDerive(motifSpec(c.bg))
		for _, v := range []struct {
			name, want string
			got        paintengine2d.Color
		}{{"fg", c.fg, fg}, {"ts", c.ts, ts}, {"bs", c.bs, bs}, {"sel", c.sel, sel}} {
			if v.want != "" && colorHexPadded(v.got) != v.want {
				t.Errorf("%s %s = %s, want %s", c.bg, v.name, colorHexPadded(v.got), v.want)
			}
		}
	}
}

// Whatever background a pack gives, the derived top shadow reads lighter
// and the bottom shadow darker; at the extremes (Motif's LITE and DARK
// models) the top shadow stays lighter than the bottom one.
func TestMotifShadowDerivationSanity(t *testing.T) {
	L := RelLuminance
	for _, hex := range []string{"#e0e0e0", "#d6d6d6", "#c4c4c4", "#aeb2c3", "#c6b2a8", "#8998aa", "#686f82", "#4992a7", "#404040", "#335577", "#7f2020"} {
		bg := Hex(hex)
		_, ts, bs, sel := motifDerive(bg)
		if !(L(ts) > L(bg) && L(bs) < L(bg) && L(sel) < L(bg)) {
			t.Errorf("%s: ts %s bs %s sel %s do not bracket the background", hex, colorHexPadded(ts), colorHexPadded(bs), colorHexPadded(sel))
		}
	}
	for _, hex := range []string{"#ffffff", "#f8f8f0", "#000000", "#101820"} {
		_, ts, bs, _ := motifDerive(Hex(hex))
		if L(ts) <= L(bs) {
			t.Errorf("%s: top shadow %s not lighter than bottom %s", hex, colorHexPadded(ts), colorHexPadded(bs))
		}
	}
	for _, n := range motifPackNames {
		p, _ := LoadTheme(n)
		c := motifColors(p.Look())
		for _, s := range []*mset{&c.win, &c.menu, &c.act, &c.inact} {
			if !(L(s.ts) > L(s.bg) && L(s.bs) < L(s.bg)) {
				t.Errorf("%s: set %s has ts %s bs %s", n, colorHexPadded(s.bg), colorHexPadded(s.ts), colorHexPadded(s.bs))
			}
		}
		// Body text reads on every set it is drawn on.
		for _, s := range []*mset{&c.win, &c.menu, &c.text} {
			if ContrastRatio(s.fg, s.bg) < 3 {
				t.Errorf("%s: foreground %s on %s", n, colorHexPadded(s.fg), colorHexPadded(s.bg))
			}
		}
	}
}

func TestMotifStyleHints(t *testing.T) {
	p, _ := LoadTheme("cde")
	lk := p.Look()
	if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 {
		t.Fatal("Motif: OK first (OK Cancel Help), tabs from the left")
	}
}

// motifAt reads an opaque pixel back as a colour.
func motifAt(img *paintengine2d.Image, x, y int) paintengine2d.Color {
	r, g, b, _ := img.PremulAt(x, y)
	return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
}

func motifExpect(t *testing.T, what string, img *paintengine2d.Image, x, y int, want paintengine2d.Color) {
	t.Helper()
	got := motifAt(img, x, y)
	d := func(a, b float32) float32 {
		if a > b {
			return a - b
		}
		return b - a
	}
	if d(got.R, want.R) > 2.5/255 || d(got.G, want.G) > 2.5/255 || d(got.B, want.B) > 2.5/255 {
		t.Errorf("%s: pixel (%d,%d) is %s, want %s", what, x, y, colorHexPadded(got), colorHexPadded(want))
	}
}

// Push buttons: a highlight band, a raised 2px shadow split on the
// diagonal; armed they invert and fill with the select colour; the default
// button gains a sunken ring; focus is the solid highlight band.
func TestMotifButtonShadowsAndStates(t *testing.T) {
	p, _ := LoadTheme("motif")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		c := motifColors(lk)
		th := int(c.px(lk))
		w, h := int(90*scale), int(32*scale)
		paint := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(w, h)
			lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, float32(w), float32(h)), st, "")
			return img
		}
		name := func(s string) string { return fmt.Sprintf("%s @%gx", s, scale) }
		img := paint(StateNone)
		motifExpect(t, name("highlight band"), img, 0, 0, c.win.bg)
		motifExpect(t, name("raised top-left"), img, th, th, c.win.ts)
		motifExpect(t, name("raised bottom-right"), img, w-th-1, h-th-1, c.win.bs)
		motifExpect(t, name("diagonal: top owns the corner"), img, w-th-1, th, c.win.ts)
		motifExpect(t, name("diagonal: right below it"), img, w-th-1, th+1, c.win.bs)
		motifExpect(t, name("face"), img, w/2, h/2, c.win.bg)
		img = paint(StateFocused)
		motifExpect(t, name("focus highlight"), img, 0, 0, c.hl)
		motifExpect(t, name("focus highlight inner"), img, th-1, h/2, c.hl)
		img = paint(StatePressed | StateHovered)
		motifExpect(t, name("armed top-left"), img, th, th, c.win.bs)
		motifExpect(t, name("armed bottom-right"), img, w-th-1, h-th-1, c.win.ts)
		motifExpect(t, name("armed face"), img, w/2, h/2, c.win.sel)
		img = paint(StatePrimary)
		motifExpect(t, name("default ring top-left"), img, th, th, c.win.bs)
		motifExpect(t, name("default ring bottom-right"), img, w-th-1, h-th-1, c.win.ts)
		// Hover is visible (the thin highlight on enter) but is not focus.
		hov, idle := paint(StateHovered), paint(StateNone)
		if motifAt(hov, 0, h/2) == motifAt(idle, 0, h/2) || motifAt(hov, 0, h/2) == c.hl {
			t.Errorf("%s: hover edge %s", name("hover"), colorHexPadded(motifAt(hov, 0, h/2)))
		}
	}
}

// Check boxes are squares (raised off, sunken and select-filled on);
// radios are diamonds (the box corners stay clear).
func TestMotifToggleIndicators(t *testing.T) {
	p, _ := LoadTheme("motif")
	lk := p.Look()
	c := motifColors(lk)
	e := lk.Engine()
	img := paintengine2d.NewImage(17, 17)
	ctx := paintengine2d.NewContext(img)
	e.CheckIndicator(lk, ctx, paintengine2d.XYWH(0, 0, 14, 14), StateNone, false)
	motifExpect(t, "check off top-left", img, 0, 0, c.win.ts)
	motifExpect(t, "check off bottom-right", img, 13, 13, c.win.bs)
	img = paintengine2d.NewImage(17, 17)
	e.CheckIndicator(lk, paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 14, 14), StateChecked, true)
	motifExpect(t, "check on top-left", img, 0, 0, c.win.bs)
	motifExpect(t, "check on fill", img, 7, 7, c.win.sel)
	for _, sel := range []bool{false, true} {
		img = paintengine2d.NewImage(17, 17)
		e.RadioIndicator(lk, paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 17, 17), StateNone, sel)
		if _, _, _, a := img.PremulAt(1, 1); a != 0 {
			t.Fatalf("radio %v paints its box corner: not a diamond", sel)
		}
		fill, tip := c.win.bg, c.win.ts
		if sel {
			fill, tip = c.win.sel, c.win.bs
		}
		motifExpect(t, fmt.Sprintf("radio %v centre", sel), img, 8, 8, fill)
		motifExpect(t, fmt.Sprintf("radio %v top vertex", sel), img, 8, 0, tip)
	}
}

// Menus arm by raising: a shadow box, never a fill.
func TestMotifMenusArmByRaising(t *testing.T) {
	p, _ := LoadTheme("cde")
	lk := p.Look()
	c := motifColors(lk)
	ch := MenuChromeFor(lk)
	w, h := int(ch.PadL+200+ch.PadR), 26
	img := paintengine2d.NewImage(w, h)
	lk.DrawMenuItem(paintengine2d.NewContext(img), paintengine2d.XYWH(ch.PadL, 0, 200, float32(h)), StateHovered, MenuRow{Label: "Open"})
	th := int(c.px(lk))
	x0 := th + 1
	motifExpect(t, "armed item top-left", img, x0, 0, c.menu.ts)
	motifExpect(t, "armed item bottom-right", img, w-x0-1, h-1, c.menu.bs)
	if _, _, _, a := img.PremulAt(w-x0-8, h/2); a != 0 {
		t.Fatal("armed menu item is filled; Motif raises it")
	}
	img = paintengine2d.NewImage(60, 30)
	lk.DrawMenuTitle(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 60, 30), StateNone, "", -1, true)
	if _, _, _, a := img.PremulAt(30, 15); a != 0 {
		t.Fatal("open menu title is filled; Motif raises it")
	}
}

// Scroll bar arrows are bevelled triangles in the trough (no button box),
// sunken while held; with nothing to scroll the slider fills the trough.
func TestMotifScrollBarArrows(t *testing.T) {
	p, _ := LoadTheme("motif")
	lk := p.Look()
	c := motifColors(lk)
	th := c.px(lk)
	view := paintengine2d.XYWH(0, 0, 16, 120)
	parts := ScrollGeometry(lk, view, true, 400, 120, 40, false)
	if parts.Dec.Empty() || parts.Inc.Empty() || parts.Thumb.Empty() {
		t.Fatalf("parts %+v", parts)
	}
	paint := func(st ScrollState) *paintengine2d.Image {
		img := paintengine2d.NewImage(16, 120)
		DrawScrollBarParts(lk, paintengine2d.NewContext(img), parts, true, st)
		return img
	}
	// The up arrow's cell is (th, th, 12, 12) at 1x; XmeDrawArrow puts the
	// lit left edge of its second band at (+2, +7).
	edge := func(img *paintengine2d.Image) paintengine2d.Color { return motifAt(img, int(th)+2, int(th)+7) }
	img := paint(ScrollState{})
	motifExpect(t, "trough beside the arrow", img, int(th), int(th), c.trough)
	motifExpect(t, "sunken trough frame", img, 0, 60, c.win.bs)
	if edge(img) != c.win.ts {
		t.Fatalf("up arrow's left edge is %s, want the top shadow", colorHexPadded(edge(img)))
	}
	if held := paint(ScrollState{Pressed: ScrollDec, Hot: ScrollDec}); edge(held) != c.win.bs {
		t.Fatalf("held up arrow's left edge is %s; it should sink", colorHexPadded(edge(held)))
	}
	full := ScrollGeometry(lk, view, true, 100, 120, 0, false)
	img = paintengine2d.NewImage(16, 120)
	DrawScrollBarParts(lk, paintengine2d.NewContext(img), full, true, ScrollState{Disabled: true})
	if motifAt(img, 8, 60) == c.trough {
		t.Fatal("with nothing to scroll the slider should fill the trough")
	}
}

// Focus is the solid highlight band inside the rect.
func TestMotifFocusRingInside(t *testing.T) {
	p, _ := LoadTheme("cde")
	lk := p.Look()
	c := motifColors(lk)
	img := paintengine2d.NewImage(40, 20)
	lk.DrawFocusRing(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 40, 20))
	motifExpect(t, "highlight corner", img, 0, 0, c.hl)
	motifExpect(t, "highlight inner edge", img, 1, 10, c.hl)
	if _, _, _, a := img.PremulAt(20, 10); a != 0 {
		t.Fatal("focus highlight fills the control")
	}
	if c.hl != c.act.bg {
		t.Fatalf("CDE highlights focus in the active colour: %s", colorHexPadded(c.hl))
	}
}

// The close control is the window menu button at the left of the title
// bar; hit-testing (WindowCloseRect) agrees with what is painted.
func TestMotifCloseIsTheWindowMenuButton(t *testing.T) {
	p, _ := LoadTheme("cde")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		c := motifColors(lk)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("scale %v: close rect %v not the menu button of frame %v (top inset %v)", scale, cr, b, in.Top)
		}
		for _, press := range []bool{false, true} {
			img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
			lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "Dialog", WindowState{Active: true, CanClose: true, CloseHot: press, ClosePress: press})
			tl, br := c.act.ts, c.act.bs
			if press {
				tl, br = br, tl
			}
			w := fmt.Sprintf("scale %v pressed %v", scale, press)
			motifExpect(t, w+": menu button top-left", img, int(cr.Min.X), int(cr.Min.Y), tl)
			motifExpect(t, w+": menu button bottom-right", img, int(cr.Max.X)-1, int(cr.Max.Y)-1, br)
			// The glyph is the raised bar in the middle of the button.
			cx, cy := int(cr.Min.X+cr.Dx()*0.5), int(cr.Min.Y+cr.Dy()*0.5)
			found := false
			for y := cy - int(3*scale); y <= cy+int(3*scale); y++ {
				if motifAt(img, cx, y) == c.act.ts {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: no bar glyph in the menu button", w)
			}
		}
	}
}

func TestMotifPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range motifPackNames {
		p, _ := LoadTheme(n)
		scales := []float32{1, 2}
		if n == "cde" || n == "irix" {
			scales = append(scales, 1.5)
		}
		for _, scale := range scales {
			lk := WithScale(p.Look(), scale).(*Classic)
			name := fmt.Sprintf("%s@%gx", n, scale)
			aquaExercise(t, name, lk)
			// The page under a tab bar and the window background too.
			for _, f := range []func(*paintengine2d.Context, paintengine2d.Rect){
				func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabPane(ctx, b) },
				func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawWindowBackground(ctx, b) },
			} {
				m := 6 * scale
				img := paintengine2d.NewImage(int(200*scale+2*m), int(90*scale+2*m))
				b := paintengine2d.XYWH(m, m, 200*scale, 90*scale)
				f(paintengine2d.NewContext(img), b)
				if x, y, ok := aquaOutside(img, b); ok {
					t.Errorf("%s: pane painted outside its rect at (%d,%d)", name, x, y)
				}
			}
		}
	}
}
