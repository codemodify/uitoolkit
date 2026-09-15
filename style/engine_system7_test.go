package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// retroPack is the registration a retro engine promises for one pack.
type retroPack struct {
	name, label, lineage string
	year                 int
}

// retroCheckPacks checks that packs are registered with their engine,
// label, lineage and year, carry a one-line summary, and list in year order.
func retroCheckPacks(t *testing.T, engine string, packs []retroPack) {
	t.Helper()
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for i, w := range packs {
		if _, ok := pos[w.name]; !ok {
			t.Fatalf("pack %q not registered", w.name)
		}
		p, _ := LoadTheme(w.name)
		if p.Tokens.Engine != engine || p.Label != w.label || p.Lineage != w.lineage || p.Year != w.year {
			t.Fatalf("%s: engine %q label %q lineage %q year %d", w.name, p.Tokens.Engine, p.Label, p.Lineage, p.Year)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", w.name, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != engine {
			t.Fatalf("%s look paints with %q", w.name, lk.Engine().ID())
		}
		if i > 0 && packs[i-1].year < w.year && pos[packs[i-1].name] > pos[w.name] {
			t.Fatalf("packs not in year order: %s after %s", packs[i-1].name, w.name)
		}
	}
}

// retroLook is pack name at a display scale.
func retroLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("missing %s", name)
	}
	return WithScale(p.Look(), scale).(*Classic)
}

// retroNear reports whether pixel (x, y) is within 40 of want per channel.
func retroNear(img *paintengine2d.Image, x, y int, want paintengine2d.Color) bool {
	return nxNear(img, x, y, want)
}

// retroExercise paints every control of every pack at 1x and 2x and fails
// on paint outside the rects (aquaExercise's contract).
func retroExercise(t *testing.T, names ...string) {
	for _, n := range names {
		for _, scale := range []float32{1, 2} {
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), retroLook(t, n, scale))
		}
	}
}

// retroCountRects counts the rects of a path (one per closed contour).
func retroCountRects(p *paintengine2d.Path) int {
	n := 0
	for _, v := range p.Verbs() {
		if v == paintengine2d.VerbClose {
			n++
		}
	}
	return n
}

var system7PackNames = []string{"system1", "system7"}

func TestSystem7PacksRegisteredInYearOrder(t *testing.T) {
	retroCheckPacks(t, "system7", []retroPack{
		{"system1", "System 1", "Mac OS", 1984},
		{"system7", "System 7", "Mac OS", 1991},
	})
}

func TestSystem7PaintsEveryControlInsideItsRect(t *testing.T) {
	retroExercise(t, system7PackNames...)
}

// Mac dialogs put the default button last and right-align form labels.
func TestSystem7StyleHints(t *testing.T) {
	for _, n := range system7PackNames {
		lk := retroLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 0 || LookHint(lk, HintFormLabelsRight) != 1 || LookHint(lk, HintTabsCentered) != 0 {
			t.Fatalf("%s: want Cancel OK, right-aligned labels, left tabs", n)
		}
		if lk.eng().FieldFocusRing(lk) {
			t.Fatalf("%s: text fields show the insertion point only", n)
		}
		if s := ScrollBarStyleOf(lk); s.Arrows != ArrowsEnds || s.Overlay || s.Thickness != 16*lk.Scale() {
			t.Fatalf("%s: scroll bar %+v", n, s)
		}
	}
}

// The close box sits at the left of the title bar, where hit-testing looks
// for it: a black square frame around white (the pressed box bursts).
func TestSystem7CloseBoxAgreesWithPaint(t *testing.T) {
	for _, n := range system7PackNames {
		for _, scale := range []float32{1, 2} {
			lk := retroLook(t, n, scale)
			u := rpU(lk)
			b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not at the left of the title bar of %v (top %v)", n, scale, cr, b, in.Top)
			}
			img := paintengine2d.NewImage(int(b.Max.X+8*scale), int(b.Max.Y+8*scale))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			c := s7colors(lk)
			black, white := Hex("#000000"), Hex("#ffffff")
			edge, face, gap, burst := black, white, white, black
			if c.grey {
				// System 7: navy and lavender rings round the grey face.
				edge, face, gap, burst = c.navy, c.boxFace, c.bar, c.lav
			}
			ctr := cr.Center()
			if !retroNear(img, int(cr.Min.X), int(ctr.Y), edge) || !retroNear(img, int(cr.Min.X+u), int(cr.Min.Y), edge) {
				t.Fatalf("%s@%gx: close box edge missing at %v", n, scale, cr)
			}
			if !retroNear(img, int(ctr.X), int(ctr.Y), face) {
				t.Fatalf("%s@%gx: close box face missing", n, scale)
			}
			// A one-pixel gap breaks the stripes each side of the box.
			if !retroNear(img, int(cr.Min.X-u), int(cr.Min.Y), gap) {
				t.Fatalf("%s@%gx: no gap before the close box", n, scale)
			}
			img2 := paintengine2d.NewImage(img.Width, img.Height)
			ctx2 := paintengine2d.NewContext(img2)
			lk.DrawWindowFrame(ctx2, b, "Dialog", WindowState{Active: true, CanClose: true, ClosePress: true})
			if !retroNear(img2, int(ctr.X), int(ctr.Y), burst) {
				t.Fatalf("%s@%gx: a pressed close box does not burst", n, scale)
			}
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s@%gx: a frame too small for a title bar reports close rect %v", n, scale, r)
			}
		}
	}
}

// Menus and windows drop the hard one-pixel shadow down and to the right.
func TestSystem7HardShadow(t *testing.T) {
	lk := retroLook(t, "system1", 1)
	for _, kind := range []PopupKind{PopupMenu, PopupDialog, PopupTooltip} {
		if in := lk.PopupShadow(kind); in.Left != 0 || in.Top != 0 || in.Right != 1 || in.Bottom != 1 {
			t.Fatalf("kind %d reach %+v, want one pixel right and down", kind, in)
		}
	}
	img := paintengine2d.NewImage(60, 50)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(10, 10, 30, 20)
	lk.DrawPopupShadow(ctx, b, PopupMenu)
	if !retroNear(img, 40, 20, Hex("#000000")) || !retroNear(img, 20, 30, Hex("#000000")) {
		t.Fatal("menu shadow missing on the right and bottom")
	}
	if _, _, _, a := img.PremulAt(40, 10); a != 0 {
		t.Fatal("the shadow starts one pixel down, not at the top corner")
	}
}

// Selections invert to black; an unfocused list outlines its selection.
func TestSystem7SelectionInvertsAndOutlines(t *testing.T) {
	lk := retroLook(t, "system1", 1)
	img := paintengine2d.NewImage(120, 30)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(0, 0, 120, 24)
	lk.DrawListRow(ctx, b, StateChecked, "")
	if !retroNear(img, 60, 12, Hex("#000000")) {
		t.Fatal("an active selection is not inverted")
	}
	img = paintengine2d.NewImage(120, 30)
	ctx = paintengine2d.NewContext(img)
	lk.DrawListRow(ctx, b, StateChecked|StateInactive, "")
	if !retroNear(img, 60, 0, Hex("#000000")) {
		t.Fatal("an inactive selection has no outline")
	}
	if _, _, _, a := img.PremulAt(60, 12); a != 0 {
		t.Fatal("an inactive selection is filled")
	}
}

// Labels read on every fill the engine paints them on.
func TestSystem7LabelsReadable(t *testing.T) {
	for _, n := range system7PackNames {
		lk := retroLook(t, n, 1)
		c := s7colors(lk)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.win, 7},
			{"text on field", lk.fieldText(), c.field, 7},
			{"selected text", c.selTxt, c.sel, 4.5},
			{"inverted text", c.white, c.black, 7},
			{"balloon text", ReadableOn(c.info, 4.5, c.text), c.info, 4.5},
			{"title text", c.text, c.bar, 4.5},
			{"inactive title", c.offTitle, c.win, 3},
			{"disabled text", c.dim, c.win, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// ---- the whole-pixel helpers --------------------------------------------------

// The 50% pattern is a one-pixel checkerboard anchored to the device grid,
// whether it is listed cell by cell or filled as even-odd stripes.
func TestRetroGrayPatternIsACheckerboard(t *testing.T) {
	for _, tc := range []struct {
		w, h int
		u    float32
	}{{40, 30, 1}, {300, 200, 1}, {64, 48, 2}, {400, 300, 2}} {
		img := paintengine2d.NewImage(tc.w, tc.h)
		ctx := paintengine2d.NewContext(img)
		// Two abutting pieces keep one phase.
		rpPatFill(ctx, paintengine2d.XYWH(3, 5, float32(tc.w/2)-3, float32(tc.h-5)), rpGray, Hex("#000000"), tc.u)
		rpPatFill(ctx, paintengine2d.XYWH(float32(tc.w/2), 5, float32(tc.w-tc.w/2), float32(tc.h-5)), rpGray, Hex("#000000"), tc.u)
		for y := 5; y < tc.h; y++ {
			for x := 3; x < tc.w; x++ {
				cx, cy := int(float32(x)/tc.u), int(float32(y)/tc.u)
				_, _, _, a := img.PremulAt(x, y)
				if want := (cx+cy)%2 == 0; (a == 255) != want || (a != 0 && a != 255) {
					t.Fatalf("%dx%d@%v: cell (%d,%d) alpha %d, want set=%v", tc.w, tc.h, tc.u, x, y, a, want)
				}
			}
		}
	}
}

// Round rects are symmetric staircases: every row of a filled shape is a
// single run centred in the box, and the one-cell frame is the border of
// that fill.
func TestRetroShapesAreSymmetricStaircases(t *testing.T) {
	for _, s := range []rpShape{{w: 60, h: 20, r: 8}, {w: 12, h: 12, r: 6}, {w: 13, h: 13, r: 6.5}, {w: 30, h: 17, cut: 1}, {w: 40, h: 24, r: 5, top: true}} {
		fill := rpNewMask(32, 32)
		frame := rpNewMask(32, 32)
		if s.w > 32 {
			s.w = 32
		}
		fill.shape(s, 0, 0)
		for y := 0; y < s.h; y++ {
			e := s.edge(y)
			if e < 0 || 2*e >= s.w {
				t.Fatalf("%+v: row %d edge %d", s, y, e)
			}
			if !s.top && s.edge(s.h-1-y) != e {
				t.Fatalf("%+v: rows %d and %d differ", s, y, s.h-1-y)
			}
			if y > 0 && y < s.h/2 && s.edge(y) > s.edge(y-1) {
				t.Fatalf("%+v: the corner widens back at row %d", s, y)
			}
			a1, b1, a2, b2, ok := s.frameRuns(y)
			if !ok {
				t.Fatalf("%+v: row %d has no outline", s, y)
			}
			frame.span(a1, b1, y)
			frame.span(a2, b2, y)
		}
		if o := fill.outline(); o.rows != frame.rows {
			t.Fatalf("%+v: frame runs differ from the outline of the fill", s)
		}
	}
	// A QuickDraw-like 12-pixel circle: rows 4, 8, 10, 10, 12 … wide.
	want := []int{4, 2, 1, 1, 0, 0}
	s := rpShape{w: 12, h: 12, r: 6}
	for y, e := range want {
		if s.edge(y) != e {
			t.Fatalf("12px circle row %d edge %d want %d", y, s.edge(y), e)
		}
	}
}

// A filled staircase costs a handful of rects, not one per row.
func TestRetroShapesBatchRows(t *testing.T) {
	var k rpInk
	g := rpGrid{u: 1, w: 200, h: 40}
	rpShape{w: 200, h: 40, r: 8}.fill(&k, g, 0, 0)
	if n := retroCountRects(k.p); n > 2*8+1 {
		t.Fatalf("a 40-row round rect fills with %d rects", n)
	}
}

// retroWholePixels paints the look's shapes without any text at 1x and 2x
// and fails on partly covered pixels or on more distinct colours than the
// look's palette: no anti-aliased curve may slip into a whole-pixel look.
func retroWholePixels(t *testing.T, name string, maxColours int) {
	t.Helper()
	for _, tc := range []struct{ scale, dx, dy float32 }{{1, 0, 0}, {2, 0, 0}, {1, 0.5, 0.25}, {2, 0.5, 0.75}} {
		scale := tc.scale
		lk := retroLook(t, name, scale)
		sc := scale
		img := paintengine2d.NewImage(int(640*sc)+2, int(260*sc)+2)
		ctx := paintengine2d.NewContext(img)
		// Layout may leave a control between device pixels; it must still
		// draw in whole pixels.
		ctx.Translate(tc.dx, tc.dy)
		r := func(x, y, w, h float32) paintengine2d.Rect { return paintengine2d.XYWH(x*sc, y*sc, w*sc, h*sc) }
		for i, st := range []ControlState{StateNone, StatePressed | StateHovered, StatePrimary, StatePrimary | StateFocused, StateFocused} {
			x := float32(4 + i*84)
			lk.DrawButton(ctx, r(x, 4, 76, 28), st, "")
			lk.DrawCheckbox(ctx, r(x, 36, 24, 22), st, true, "")
			lk.DrawRadio(ctx, r(x+30, 36, 24, 22), st, true, "")
			lk.DrawComboBox(ctx, r(x, 62, 76, 26), st, "", false)
			lk.DrawTab(ctx, r(x, 92, 76, 30), st, "", i%2 == 0)
		}
		view := r(430, 4, 60, 200)
		DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, view, true, 900*sc, view.Dy(), 200*sc, false), true, ScrollState{Pressed: ScrollDec})
		DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, r(4, 210, 300, 40), false, 900*sc, 300*sc, 100*sc, false), false, ScrollState{})
		lk.DrawWindowFrame(ctx, r(500, 4, 136, 120), "", WindowState{Active: true, CanClose: true, Maximizable: true})
		lk.DrawMenuFrame(ctx, r(500, 130, 136, 60))
		colours := map[[4]uint8]bool{}
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				rr, g, b, a := img.PremulAt(x, y)
				if a != 0 && a != 255 {
					t.Fatalf("%s@%gx: pixel (%d,%d) is partly covered (alpha %d)", name, scale, x, y, a)
				}
				if a == 255 {
					colours[[4]uint8{rr, g, b, a}] = true
				}
			}
		}
		if len(colours) > maxColours {
			t.Fatalf("%s@%gx: %d distinct colours, want at most %d (anti-aliasing?)", name, scale, len(colours), maxColours)
		}
	}
}

// The retro looks are drawn in whole pixels: 1-bit System 1 in black and
// white, System 7 in its handful of window colours.
func TestSystem7IsWholePixel(t *testing.T) {
	retroWholePixels(t, "system1", 2)
	retroWholePixels(t, "system7", 16)
}

// Elliptical caps (OPEN LOOK's obround) cut more across than down, and a
// circular corner is the special case rx == r.
func TestRetroEllipticalCorners(t *testing.T) {
	round := rpShape{w: 40, h: 19, r: 6}
	oval := rpShape{w: 40, h: 19, r: 6, rx: 9}
	if round.edge(0) >= oval.edge(0) || oval.edge(0) > 9 {
		t.Fatalf("top row cut: round %d, elliptical %d", round.edge(0), oval.edge(0))
	}
	for y := 0; y < 19; y++ {
		if e := (rpShape{w: 40, h: 19, r: 6, rx: 6}).edge(y); e != round.edge(y) {
			t.Fatalf("row %d: rx == r gives %d, the circle %d", y, e, round.edge(y))
		}
	}
	// The caps curve over their six rows (the last ones round to the
	// straight edge), leaving a straight run of at least 19 - 2*6 rows.
	straight := 0
	for y := 0; y < 19; y++ {
		if oval.edge(y) == 0 {
			straight++
		}
	}
	if straight < 7 || straight > 11 {
		t.Fatalf("straight run %d rows, want 7 to 11", straight)
	}
}

// Glyph masks turn to every direction without losing cells.
func TestRetroMaskTurns(t *testing.T) {
	up := s7arrowMask(13, true)
	count := func(m rpMask) int {
		n := 0
		for y := 0; y < m.h; y++ {
			for x := 0; x < m.w; x++ {
				if m.get(x, y) {
					n++
				}
			}
		}
		return n
	}
	for _, dir := range []Direction{DirDown, DirLeft, DirRight} {
		m := up.turn(dir)
		if count(m) != count(up) {
			t.Fatalf("turning %d loses cells", dir)
		}
	}
	if !up.get(6, 0) || up.get(0, 0) {
		t.Fatal("the up arrow's tip is the top centre cell")
	}
	if r := up.turn(DirRight); !r.get(r.w-1, 6) || r.get(r.w-1, 0) {
		t.Fatal("the right arrow's tip is the right middle cell")
	}
	if l := up.turn(DirLeft); !l.get(0, 6) {
		t.Fatal("the left arrow's tip is the left middle cell")
	}
}
