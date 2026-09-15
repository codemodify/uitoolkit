package style

import (
	"math"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// os2Engine paints OS/2 Warp 4 (1996), the last redesign of IBM's
// Presentation Manager: pale #CCCCCC dialogs, chunky two-pixel bevels set in
// a one-pixel groove, pastel notebook tabs and a title well sunk into the
// window frame.
//
// Everything is drawn in whole pixels (see engine_system7_pixel.go). Warp 4
// used the 9.WarpSans bitmap font at 640 × 480; the toolkit's UI font is
// 16px, so control heights grow a little (the 26-pixel push button takes a
// 28px slot) while every bevel, groove and line stays a whole number of
// native pixels.
//
//   - Push buttons are a one-pixel groove (#808080 over white) round a
//     two-pixel raised bevel (white over #808080) on the #CCCCCC face, with
//     a black label. The default button trades the groove for a two-pixel
//     black border, and that border follows the keyboard focus: a focused
//     button wears it too, with no dotted rectangle. Pressed, the bevel
//     inverts and the label moves a pixel (inferred: not captured).
//   - Check boxes are small buttons: groove, raised bevel and a grey face;
//     the black two-pixel tick leaves the box through its top-right corner.
//     Radio buttons are round in the same treatment with a black dot
//     (inferred). Their keyboard focus is a one-on, one-off dotted rectangle
//     round the label.
//   - Disabled text and controls are halftoned: every other pixel erased to
//     the face by the 50% checkerboard (never embossed).
//   - Entry fields are white in a two-pixel sunken frame (#808080 over black,
//     white over the face); drop-down combos keep their raised button with
//     a 6 × 3 black triangle inside that frame. Selections are #808080 with
//     white text.
//   - Scroll bars: a one-pixel black separator beside a 14-pixel bar
//     (Warp 4 drew 12 + 1 at 640 × 480; 14 suits the 16px font), a #C0C0C0
//     shaft, raised arrow buttons longer than the bar is wide holding a tiny
//     black triangle, and a raised thumb gripped by six black-and-white
//     lines four pixels long.
//   - Notebooks carry Warp 4's major tabs across the top: pastel trapezoids,
//     white along the top and the left slant (one pixel per three rows),
//     shaded on the right; the selected tab has no line under it and its
//     colour runs four pixels into the page as a tapering lip.
//   - Windows: a four-pixel sizing border (#808080 then white on the top and
//     left, black then #808080 on the bottom and right), the program's mini
//     icon on the frame at the left, the left-aligned bold title in a sunken
//     #2B00AA well and the Close, Hide and Maximize glyphs to its right.
//   - Menus: #CCCCCC in a white and black one-pixel frame, a full-width
//     navy bar with white text on the hot item.
//   - Static text and group-box titles are navy (Palette.Text), the colour
//     the Label widget uses; the controls' own labels are black.
//   - Lists mark the current item with a dotted rectangle (cursored
//     emphasis). There are no drop shadows and no dimming behind dialogs.
//
// Colours and sizes are taken as facts from screenshots and documentation of
// the shipped system; every shape here is drawn procedurally, and the mini
// icon is an invention standing in for a program's own.
//
// Pack data:
//
//	extra:
//	  dark, hi                 the bevel's #808080 and white
//	  controlText              labels of buttons, check boxes, lists, menus
//	  selText                  text on the #808080 selection
//	  menuHi, menuHiText       the hot menu item's bar and text
//	  title, titleText         the active title well and its text
//	  titleOff, titleOffText   the inactive title well and its text
//	  shaft, shaftOff          scroll shaft with and without anything to scroll
//	  info                     fly-over help fill
//	  ribbon                   progress ribbon and a switch that is on
//	  tab0 … tab6              the notebook tab pastels
type os2Engine struct{ BaseEngine }

func init() {
	RegisterEngine(os2Engine{})
	for _, p := range os2Packs() {
		RegisterPack(p)
	}
}

func (os2Engine) ID() string { return "os2" }

// DefaultMetrics are Warp 4's proportions at the toolkit's 16px UI font:
// a 28px button slot (the 26-pixel button plus room for the default
// border), 13-pixel check boxes (in a 15px cell that leaves the tick room to
// leave the box) and radio buttons, 15px scroll bars (a 14-pixel bar and
// its separator).
func (os2Engine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Square:    true, BevelDepth: 2,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 15, Radio: 13,
		MenuItemH: 24, MenuBarH: 26, TabH: 30, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 18, SliderH: 26, Thumb: 11,
		Scroll: 15, Pad: 10, FieldPad: 5, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24, SpinnerW: 16, SwitchW: 36, SwitchH: 18,
	}
}

// ---- colours ------------------------------------------------------------------

// o2 is the resolved colour set of a look.
type o2 struct {
	face, dark, hi, black, text, static, field, dim paintengine2d.Color
	sel, selTxt, menuHi, menuTxt                    paintengine2d.Color
	title, titleTxt, titleOff, titleOffTxt          paintengine2d.Color
	shaft, shaftOff, info, ribbon                   paintengine2d.Color
	tabs                                            [7]paintengine2d.Color
}

type o2Key struct{}

func o2colors(l *Classic) *o2 {
	return l.Memo(o2Key{}, func() any { return o2build(l) }).(*o2)
}

// o2tabColors are the pastels Warp 4's notebook tabs wear.
var o2tabColors = [7]string{"#55dbff", "#82dbaa", "#8292ff", "#d7b6aa", "#ffffaa", "#aa92aa", "#ff9255"}

func o2build(l *Classic) *o2 {
	p := l.palette
	x := func(k, def string) paintengine2d.Color { return l.X(k, Hex(def)) }
	def := func(c paintengine2d.Color, d string) paintengine2d.Color {
		if colorUnset(c) {
			return Hex(d)
		}
		return c
	}
	c := &o2{
		face:        def(p.SurfaceAlt, "#cccccc"),
		dark:        x("dark", "#808080"),
		hi:          x("hi", "#ffffff"),
		black:       Hex("#000000"),
		text:        x("controlText", "#000000"),
		static:      def(p.Text, "#000080"),
		field:       def(p.Field, "#ffffff"),
		dim:         def(p.TextMuted, "#808080"),
		sel:         p.Selection,
		selTxt:      x("selText", "#ffffff"),
		menuHi:      x("menuHi", "#000080"),
		menuTxt:     x("menuHiText", "#ffffff"),
		title:       x("title", "#2b00aa"),
		titleTxt:    x("titleText", "#ffffff"),
		titleOff:    x("titleOff", "#cccccc"),
		titleOffTxt: x("titleOffText", "#555555"),
		shaft:       x("shaft", "#c0c0c0"),
		shaftOff:    x("shaftOff", "#cccccc"),
		info:        x("info", "#ffffc0"),
		ribbon:      x("ribbon", "#000080"),
	}
	if c.sel.A < 0.9 {
		c.sel = c.dark
	}
	for i, d := range o2tabColors {
		c.tabs[i] = x("tab"+itoa(i), d)
	}
	return c
}

// ---- drawing helpers -------------------------------------------------------------

// Frame geometry in native pixels (cells).
const (
	o2border = 4  // the sizing border
	o2above  = 5  // frame rows above the title well
	o2below  = 2  // frame rows between the title well and the client area
	o2glyph  = 14 // a title-bar glyph
	o2icon   = 16 // the program's mini icon
	o2lip    = 4  // rows the selected notebook tab runs into its page
)

// o2bevel appends an n-cell 3D edge: white on the top and left, #808080 on
// the bottom and right, or the reverse when sunk.
func o2bevel(white, dark *rpInk, g rpGrid, x, y, w, h, n int, sunk bool) {
	if sunk {
		rpEdge(dark, white, g, x, y, w, h, n)
		return
	}
	rpEdge(white, dark, g, x, y, w, h, n)
}

// box paints a raised (or sunk) face filling g: the #CCCCCC face in a
// two-pixel bevel (one pixel on very small boxes).
func (c *o2) box(ctx *paintengine2d.Context, g rpGrid, sunk bool) {
	if g.w < 2 || g.h < 2 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	n := 2
	if g.w < 6 || g.h < 6 {
		n = 1
	}
	var white, dark rpInk
	o2bevel(&white, &dark, g, 0, 0, g.w, g.h, n, sunk)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
}

// fieldBox paints an entry field's well filling g: white inside the
// two-pixel sunken frame, #808080 over black on the top and left, white
// over the face on the bottom and right.
func (c *o2) fieldBox(ctx *paintengine2d.Context, g rpGrid) {
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.field))
	if g.w < 4 || g.h < 4 {
		return
	}
	var white, dark, black, face rpInk
	rpEdge(&dark, &white, g, 0, 0, g.w, g.h, 1)
	rpEdge(&black, &face, g, 1, 1, g.w-2, g.h-2, 1)
	face.fill(ctx, c.face)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
}

// halftone overlays r with the 50% checkerboard in bg: how Presentation
// Manager greys a disabled item, every other pixel erased.
func (c *o2) halftone(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, bg paintengine2d.Color) {
	if !r.Empty() {
		rpPatFill(ctx, r, rpGray, bg, rpU(l))
	}
}

// label draws text fitted into b with rune idx underlined (the mnemonic;
// -1 for none); disabled text is halftoned into bg.
func (c *o2) label(l *Classic, ctx *paintengine2d.Context, f *Font, text string, idx int, b paintengine2d.Rect, col paintengine2d.Color, align Align, disabled bool, bg paintengine2d.Color) {
	if f == nil || text == "" || b.Empty() {
		return
	}
	l.drawFittedText(ctx, f, text, b, col, align, 0)
	box := rpTextBox(f, text, b, align)
	if idx >= 0 && f.Advance(text) <= b.Dx() {
		o2underline(ctx, f, text, idx, box.Min, col, rpU(l), b)
	}
	if disabled {
		c.halftone(l, ctx, box.Intersect(b), bg)
	}
}

// o2underline draws the mnemonic underline, one pixel below the baseline,
// under rune idx of text set at origin, kept inside clip.
func o2underline(ctx *paintengine2d.Context, f *Font, text string, idx int, origin paintengine2d.Point, col paintengine2d.Color, u float32, clip paintengine2d.Rect) {
	if idx < 0 || idx >= utf8.RuneCountInString(text) {
		return
	}
	x0 := snap(origin.X + f.CaretX(text, idx))
	x1 := snap(origin.X + f.CaretX(text, idx+1))
	if x1 <= x0 {
		return
	}
	r := paintengine2d.XYWH(x0, snap(origin.Y+f.Ascent)+u, x1-x0, u).Intersect(clip)
	if !r.Empty() {
		ctx.DrawRect(r, paintengine2d.Fill(col))
	}
}

// dotted paints the one-on, one-off dotted rectangle just inside r: Warp's
// cursored emphasis and keyboard focus.
func (c *o2) dotted(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, col paintengine2d.Color) {
	g := rpGridAt(ctx, r, rpU(l))
	var k rpInk
	k.dotted(g, 0, 0, g.w, g.h)
	k.fill(ctx, col)
}

// o2tri is a solid triangle of base w cells pointing dir: rows of w, w-2 …
// cells (the scroll arrow's 6 × 3 glyph, the combo's drop mark).
func o2tri(w int, dir Direction) rpMask {
	if w < 2 {
		w = 2
	}
	h := (w + 1) / 2
	m := rpNewMask(w, h)
	for i := 0; i < h; i++ {
		m.span(i, w-i, i)
	}
	switch dir { // drawn pointing down
	case DirUp:
		return m.turn(DirDown)
	case DirLeft, DirRight:
		return m.turn(DirDown).turn(dir)
	}
	return m
}

// o2tick is the check box mark in an n-cell cell whose bottom-left bs × bs
// cells, from row by, hold the box: a short stroke down to the right and a
// long one up through the box's top-right corner into the room above it,
// both two cells thick.
func o2tick(n, bs, by int) rpMask {
	m := rpNewMask(n, n)
	i0, i1 := 3, bs-3 // the face inside the groove and bevel
	vx := i0 + (i1-i0)*2/7
	vy := by + bs - 4
	ax := i0
	ay := vy - (vx - ax) - 1
	for d := 0; d < 2; d++ {
		m.line(ax+d, ay, vx+d, vy)
		m.line(vx+d, vy, bs-1+d, 0)
	}
	return m
}

// o2mark is the menu check mark in an n-cell box: the same two-cell tick
// without its box.
func o2mark(n int) rpMask {
	m := rpNewMask(n, n)
	vx, vy := n*3/8, n-2
	for d := 0; d < 2; d++ {
		m.line(d, vy-vx, vx+d, vy)
		m.line(vx+d, vy, n-2+d, 1)
	}
	return m
}

// o2split appends the set cells of m, a ring round an n-cell disc, to hi
// where they face the upper left and to lo elsewhere: the round version of
// a bevel (lo owns the cells on the diagonal).
func o2split(m rpMask, hi, lo *rpInk, g rpGrid) {
	a, b := rpNewMask(m.w, m.h), rpNewMask(m.w, m.h)
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if !m.get(x, y) {
				continue
			}
			if x+y < m.w-1 {
				a.set(x, y)
			} else {
				b.set(x, y)
			}
		}
	}
	a.emit(hi, g, 0, 0)
	b.emit(lo, g, 0, 0)
}

// o2tabStep is the narrowest tab a tab bar lays out, in pixels.
const o2tabStep = 56

// o2tabIndex picks the pastel of a tab whose left edge is at x. Warp 4
// cycled its pastels tab by tab, but the engine sees one tab at a time and
// not its index; so the pastel follows the tab's place along the strip in
// steps of the narrowest tab. Neighbours are at least one step apart and
// never share a colour (short of a tab seven steps wide), the first tab
// takes the first pastel, and a tab keeps its colour when it is selected.
func o2tabIndex(x float32) int {
	n := len(o2tabColors)
	i := int(math.Floor(float64(x/o2tabStep))) % n
	if i < 0 {
		i += n
	}
	return i
}

// pushButton paints Warp 4's push button filling g and returns the face
// inside its bevel and whether it is pressed. Every button keeps the
// default border's two cells: the groove sits in the inner one, the black
// border (default or focused) fills both.
func (c *o2) pushButton(ctx *paintengine2d.Context, g rpGrid, st ControlState) (rpGrid, bool) {
	pressed := st.Pressed() && !st.Disabled()
	emph := (st.Primary() || st.Focused()) && !st.Disabled()
	m := 2
	if g.w < 16 || g.h < 12 {
		m = 0 // too small for the border: the bevel alone
	}
	var white, dark, black rpInk
	if m > 0 {
		if emph {
			black.frame(g, 0, 0, g.w, g.h, 2)
		} else {
			rpEdge(&dark, &white, g, 1, 1, g.w-2, g.h-2, 1) // the groove
		}
	}
	body := g.inset(m)
	n := 2
	if body.w < 8 || body.h < 8 {
		n = 1
	}
	ctx.DrawRect(body.rect(), paintengine2d.Fill(c.face))
	o2bevel(&white, &dark, body, 0, 0, body.w, body.h, n, pressed)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	return body.inset(n), pressed
}

// o2thumbInks appends the scroll thumb filling g: a raised two-pixel box and,
// when it is long enough, the grip across its middle — six one-cell lines,
// black and white in turn, four cells long.
func o2thumbInks(face, white, dark, black *rpInk, g rpGrid, vertical bool) {
	if g.w < 4 || g.h < 4 {
		return
	}
	face.add(g.rect())
	o2bevel(white, dark, g, 0, 0, g.w, g.h, 2, false)
	along, across := g.h, g.w
	if !vertical {
		along, across = g.w, g.h
	}
	if along < 12 || across < 8 {
		return
	}
	a0, x0 := (along-6)/2, (across-4)/2
	for i := 0; i < 6; i++ {
		k := black
		if i%2 == 1 {
			k = white
		}
		if vertical {
			k.cells(g, x0, a0+i, 4, 1)
		} else {
			k.cells(g, a0+i, x0, 1, 4)
		}
	}
}

// ---- parts --------------------------------------------------------------------

func (e os2Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if g.w < 3 || g.h < 3 {
		return fg
	}
	pressed := st.Pressed() && !st.Disabled()
	switch role {
	case RoleButton:
		c.pushButton(ctx, g, st)
		return fg
	case RoleTool:
		switch {
		case st.Disabled():
		case pressed || (st.Toggle() && st.Checked()):
			c.box(ctx, g, true)
		case st.Hovered():
			c.box(ctx, g, false)
		}
		return fg
	case RoleField, RoleCombo:
		c.fieldBox(ctx, g)
		return fg
	case RoleCheck:
		var white, dark rpInk
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
		rpEdge(&dark, &white, g, 0, 0, g.w, g.h, 1)
		o2bevel(&white, &dark, g, 1, 1, g.w-2, g.h-2, 2, pressed)
		white.fill(ctx, c.hi)
		dark.fill(ctx, c.dark)
		return fg
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.sel))
			return c.selTxt
		}
		return fg
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.menuHi))
			return c.menuTxt
		}
		return fg
	case RoleThumb:
		var face, white, dark, black rpInk
		o2thumbInks(&face, &white, &dark, &black, g, g.h >= g.w)
		face.fill(ctx, c.face)
		white.fill(ctx, c.hi)
		dark.fill(ctx, c.dark)
		black.fill(ctx, c.black)
		return fg
	case RoleTrack:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.shaft))
		return fg
	case RoleTab:
		return fg
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	return fg
}

// CheckIndicator is Warp 4's check box: a small button — groove, raised
// two-pixel bevel, grey face — whose black tick leaves it through the
// top-right corner (the cell keeps two rows and columns for it). Pressed,
// the bevel sinks.
func (e os2Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5), 32)
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	over := 2
	if n < 12 {
		over = 0
	}
	bs := n - over
	pressed := st.Pressed() && !st.Disabled()
	var white, dark, black rpInk
	rpEdge(&dark, &white, g, 0, over, bs, bs, 1)
	o2bevel(&white, &dark, g, 1, over+1, bs-2, bs-2, 2, pressed)
	ctx.DrawRect(g.at(3, over+3, bs-6, bs-6), paintengine2d.Fill(c.face))
	if checked {
		m := o2tick(n, bs, over)
		m.emit(&black, g, 0, 0)
	}
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	if st.Disabled() {
		c.halftone(l, ctx, g.rect(), c.face)
	}
}

// RadioIndicator is a round check box (inferred: not captured): a groove
// ring, a raised ring (one cell at the 13-pixel size, two from 17) and the
// grey face, with a black dot when selected. Pressed, the raised ring sinks.
func (e os2Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5), 32)
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	disc := rpShape{w: n, h: n, r: float32(n) * 0.5}
	inner := rpNewMask(n, n)
	inner.shape(disc, 0, 0)
	groove := inner.outline()
	inner = inner.minus(groove)
	raised := inner.outline()
	inner = inner.minus(raised)
	if n >= 17 {
		r2 := inner.outline()
		raised = raised.or(r2)
		inner = inner.minus(r2)
	}
	var face, white, dark, black rpInk
	o2split(groove, &dark, &white, g)
	if st.Pressed() && !st.Disabled() {
		o2split(raised, &dark, &white, g)
	} else {
		o2split(raised, &white, &dark, g)
	}
	inner.emit(&face, g, 0, 0)
	if selected {
		d := n - 8
		if (n-d)%2 != 0 {
			d--
		}
		if d >= 3 {
			rpShape{w: d, h: d, r: float32(d) * 0.5}.fill(&black, g, (n-d)/2, (n-d)/2)
		}
	}
	face.fill(ctx, c.face)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	if st.Disabled() {
		c.halftone(l, ctx, g.rect(), c.face)
	}
}

// Arrow is a solid whole-pixel triangle (sort marks, submenus, the
// toolkit's own arrow buttons).
func (e os2Engine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 3 {
		return
	}
	w := min(max(n*2/3, 3), 9)
	m := o2tri(w, dir)
	var k rpInk
	m.emit(&k, g, (g.w-m.w)/2, (g.h-m.h)/2)
	k.fill(ctx, col)
}

// Expander is the tree view's plus / minus button: a small raised square
// in a dark frame with the sign in col.
func (e os2Engine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h, 11)
	if n%2 == 0 {
		n--
	}
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	var white, dark, sign rpInk
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	dark.frame(g, 0, 0, n, n, 1)
	o2bevel(&white, &dark, g, 1, 1, n-2, n-2, 1, false)
	s := n - 6
	sign.cells(g, 3, n/2, s, 1)
	if !expanded {
		sign.cells(g, n/2, 3, 1, s)
	}
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	sign.fill(ctx, col)
}

// MenuHighlight is the full-width navy bar of the hot item (and an open
// title).
func (e os2Engine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(o2colors(l).menuHi))
}

func (e os2Engine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := o2colors(l)
	if hot {
		return c.menuTxt
	}
	return c.text
}

// FieldFocusRing: entry fields show focus with the cursor alone.
func (os2Engine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is the one-on, one-off dotted rectangle just inside b.
func (e os2Engine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := o2colors(l)
	c.dotted(l, ctx, b, c.black)
}

// ---- scrollbars -------------------------------------------------------------

// ScrollBarStyle: a 14-pixel bar and its one-pixel separator, arrow
// buttons 18 pixels long at each end (Warp 4's are 12 × 16).
func (os2Engine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 15, Arrows: ArrowsEnds, ArrowLen: 18, MinThumb: 18}
}

func (e os2Engine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, p.Bar, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	active := !st.Disabled && !p.Thumb.Empty()
	along, across := g.h, g.w
	if !vertical {
		along, across = g.w, g.h
	}
	// sub is the grid of n cells along from a and m across from x.
	sub := func(a, x, n, m int) rpGrid {
		if vertical {
			return g.sub(x, a, m, n)
		}
		return g.sub(a, x, n, m)
	}
	pos := func(r paintengine2d.Rect) (int, int) {
		if vertical {
			return int((r.Min.Y-g.y)/u + 0.5), int(r.Dy()/u + 0.5)
		}
		return int((r.Min.X-g.x)/u + 0.5), int(r.Dx()/u + 0.5)
	}
	sep := 1 // the separator on the content side
	if across < 8 {
		sep = 0
	}
	shaft := c.shaft
	if !active {
		shaft = c.shaftOff
	}
	ctx.DrawRect(sub(0, sep, along, across-sep).rect(), paintengine2d.Fill(shaft))
	var face, white, dark, black rpInk
	if sep > 0 {
		black.add(sub(0, 0, along, 1).rect())
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	type end struct {
		r    paintengine2d.Rect
		dir  Direction
		part ScrollPart
	}
	var ghost [2]paintengine2d.Rect
	for i, a := range [2]end{{p.Dec, dec, ScrollDec}, {p.Inc, inc, ScrollInc}} {
		if a.r.Empty() {
			continue
		}
		a0, n := pos(a.r)
		bg := sub(a0, sep, n, across-sep)
		if bg.w < 4 || bg.h < 4 {
			continue
		}
		pressed := active && st.Pressed == a.part
		face.add(bg.rect())
		o2bevel(&white, &dark, bg, 0, 0, bg.w, bg.h, 2, pressed)
		tri := o2tri(6, a.dir)
		ox, oy := (bg.w-tri.w)/2, (bg.h-tri.h)/2
		if pressed {
			ox, oy = ox+1, oy+1
		}
		tri.emit(&black, bg, ox, oy)
		if !active {
			ghost[i] = bg.at(ox, oy, tri.w, tri.h)
		}
	}
	if active {
		t0, tn := pos(p.Thumb)
		o2thumbInks(&face, &white, &dark, &black, sub(t0, sep, tn, across-sep), vertical)
	}
	face.fill(ctx, c.face)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	for _, r := range ghost {
		c.halftone(l, ctx, r, c.face)
	}
}

// DrawScrollBar paints a bare shaft and thumb (widgets that lay out their
// own bar).
func (e os2Engine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered(), Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------

func (os2Engine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is an etched frame (#808080 over white) with its navy title
// at the top left, after a six-pixel stub of the top line; a raised box is
// a bevelled panel with the title inside.
func (e os2Engine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := o2colors(l)
	u := rpU(l)
	f := l.body
	if raised {
		g := rpGridAt(ctx, b, u)
		c.box(ctx, g, false)
		if title != "" && g.w > 16 {
			l.drawFittedText(ctx, f, title, paintengine2d.XYWH(g.x+8*u, g.y+3*u, float32(g.w-16)*u, f.Height()), c.static, AlignStart, 0)
		}
		return
	}
	ctx.DrawRect(rpGridAt(ctx, b, u).rect(), paintengine2d.Fill(c.face))
	top := b.Min.Y
	if title != "" {
		top += f.Height() * 0.5
	}
	g := rpGridAt(ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	var white, dark rpInk
	rpEdge(&dark, &white, g, 0, 0, g.w, g.h, 1)
	rpEdge(&white, &dark, g, 1, 1, g.w-2, g.h-2, 1)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	if title == "" || g.w < 24 {
		return
	}
	tw := min(int(math.Ceil(float64(f.Advance(title)/u))), g.w-18)
	if tw <= 0 {
		return
	}
	// The gap: from the stub's end, two cells of air each side of the text.
	ctx.DrawRect(paintengine2d.XYWH(g.x+6*u, b.Min.Y, float32(tw+4)*u, f.Height()), paintengine2d.Fill(c.face))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(g.x+8*u, b.Min.Y, float32(tw)*u, f.Height()), c.static, AlignStart, 0)
}

// o2wellH is the title well's height in cells: 18 pixels at 9.WarpSans
// Bold, grown with the toolkit's bold font.
func o2wellH(l *Classic) int {
	h := int(l.BoldFont().Height()/rpU(l)+0.5) + 3
	if h < 18 {
		h = 18
	}
	return h
}

// o2titleRow is the title row of a frame, in cells of the frame's grid.
type o2titleRow struct {
	ok               bool
	wy, wh, wx0, wx1 int // the well
	ix, iy           int // the mini icon
	gy               int // the glyphs' top row
	close, hide, max int // glyph columns, -1 when absent
}

// o2row lays out a frame's title row: the mini icon on the frame at the
// left, the well, and the glyphs at the right in Warp 4's order — Close
// (new in Warp 4, set left of the others), Hide, Maximize.
func o2row(l *Classic, g rpGrid, canClose, maximizable bool) o2titleRow {
	r := o2titleRow{close: -1, hide: -1, max: -1, wy: o2above, wh: o2wellH(l)}
	if g.w < 2*o2border+o2icon+60 || g.h < r.wy+r.wh+o2below+o2border+2 {
		return r
	}
	r.ok = true
	r.ix, r.iy = o2border+1, r.wy+(r.wh-o2icon)/2
	r.wx0 = r.ix + o2icon + 2
	r.gy = r.wy + (r.wh-o2glyph)/2
	x := g.w - o2border - 1 // the left edge of the leftmost glyph so far
	if maximizable {
		r.max = x - o2glyph
		r.hide = r.max - 2 - o2glyph
		x = r.hide - 2
	}
	if canClose {
		r.close = x - o2glyph
		x = r.close
	}
	r.wx1 = x
	if x < g.w-o2border-1 {
		r.wx1 = x - 3
	}
	if r.wx1-r.wx0 < 24 {
		r.ok = false
	}
	return r
}

func (os2Engine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: float32(o2above+o2wellH(l)+o2below) * u, Right: o2border * u, Bottom: o2border * u, Left: o2border * u}
}

// WindowCloseRect is the Close glyph right of the title well. The toolkit's
// in-app windows are never maximizable, so Close is the rightmost glyph;
// WindowCloseRect has no WindowState to follow a frame that also shows
// Hide and Maximize (Close then sits two glyphs further left).
func (e os2Engine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	r := o2row(l, g, true, false)
	if !r.ok || r.close < 0 {
		return paintengine2d.Rect{}
	}
	return g.at(r.close, r.gy, o2glyph, o2glyph)
}

// PopupShadow: Warp 4's menus, lists and dialogs cast no shadow.
func (os2Engine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (os2Engine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// ItemFocus is cursored emphasis: a dotted rectangle round the current
// item, white on the #808080 selection.
func (e os2Engine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := o2colors(l)
	col := c.text
	if st.Checked() {
		col = c.selTxt
	}
	c.dotted(l, ctx, b, col)
}

// ViewFrameInsets: lists sit in the entry field's two-pixel well.
func (os2Engine) ViewFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: 2 * u, Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// DrawViewFrame: white inside the sunken frame; focus shows as the dotted
// rectangle of the current item.
func (e os2Engine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	o2colors(l).fieldBox(ctx, rpGridAt(ctx, b, rpU(l)))
}

// StyleHint: push buttons line up bottom left, the default (OK) first,
// then Cancel and Help; form labels are left-aligned.
func (os2Engine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// ControlFont: 9.WarpSans labels every control; only window titles are
// bold.
func (os2Engine) ControlFont(l *Classic, role Role) *Font { return l.body }

// ToolBarInsets: no grip, six pixels of air at each end.
func (os2Engine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(6), Right: l.S(6)} }

// DrawWindowBackground is the #CCCCCC dialog background.
func (os2Engine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(o2colors(l).face))
}

// TabOutset: Warp 4's tabs keep their size when selected; the lip into the
// page marks the selection.
func (os2Engine) TabOutset(*Classic) Insets { return Insets{} }

// DrawTabPane is the notebook page in its raised two-pixel bevel; the tab
// bar paints over its top edge and draws that edge itself.
func (e os2Engine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	o2colors(l).box(ctx, rpGridAt(ctx, b, rpU(l)), false)
}

// o2miniIcon appends the program's mini icon with its top-left cell at (x, y)
// of g — an invention: a small window with a title strip, a white client
// holding two lines, a black outline and a dark shadow.
func o2miniIcon(white, dark, black, strip *rpInk, g rpGrid, x, y int) {
	dark.cells(g, x+14, y+3, 1, 11)
	dark.cells(g, x+2, y+13, 12, 1)
	black.frame(g, x+1, y+2, 13, 11, 1)
	strip.cells(g, x+2, y+3, 11, 2)
	black.cells(g, x+2, y+5, 11, 1)
	white.cells(g, x+2, y+6, 11, 6)
	dark.cells(g, x+4, y+8, 7, 1)
	dark.cells(g, x+4, y+10, 5, 1)
}

// o2closeGlyph appends Warp 4's Close glyph in the 14-cell grid g: a raised
// square with an engraved diagonal from its top right to its bottom left.
func o2closeGlyph(white, dark *rpInk, g rpGrid, pressed bool) {
	o2bevel(white, dark, g, 0, 0, o2glyph, o2glyph, 1, pressed)
	d, w := rpNewMask(o2glyph, o2glyph), rpNewMask(o2glyph, o2glyph)
	d.line(o2glyph-2, 1, 1, o2glyph-2)
	w.line(o2glyph-2, 2, 2, o2glyph-2)
	d.emit(dark, g, 0, 0)
	w.emit(white, g, 0, 0)
}

// o2hideGlyph appends the Hide glyph: a 10-cell square outline dashed two
// on, two off, embossed (white one cell down and right, #808080 on top).
func o2hideGlyph(white, dark *rpInk, g rpGrid) {
	m := rpNewMask(o2glyph, o2glyph)
	for i := 0; i < 10; i++ {
		if i%4 < 2 {
			m.set(2+i, 2)
			m.set(2+i, 11)
			m.set(2, 2+i)
			m.set(11, 2+i)
		}
	}
	m.emit(white, g, 1, 1)
	m.emit(dark, g, 0, 0)
}

// o2maxGlyph appends the Maximize glyph: a raised square holding a sunken
// square two cells in.
func o2maxGlyph(white, dark *rpInk, g rpGrid) {
	o2bevel(white, dark, g, 0, 0, o2glyph, o2glyph, 1, false)
	o2bevel(white, dark, g, 2, 2, o2glyph-4, o2glyph-4, 1, true)
}

// DrawWindowFrame is a Warp 4 frame window: the four-pixel sizing border,
// the program's mini icon on the frame, the left-aligned bold title in the
// sunken well (#2B00AA when active; #CCCCCC with dark grey text when not —
// inferred) and the glyphs right of the well. The glyphs show no hover.
func (e os2Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	if g.w < 8 || g.h < 8 {
		return
	}
	var white, dark, black, strip, well rpInk
	rpEdge(&dark, &black, g, 0, 0, g.w, g.h, 1)
	rpEdge(&white, &dark, g, 1, 1, g.w-2, g.h-2, 1)
	r := o2row(l, g, st.CanClose, st.Maximizable)
	var wg rpGrid
	if r.ok {
		wg = g.sub(r.wx0, r.wy, r.wx1-r.wx0, r.wh)
		well.cells(wg, 1, 1, wg.w-2, wg.h-2)
		rpEdge(&dark, &white, wg, 0, 0, wg.w, wg.h, 1)
		o2miniIcon(&white, &dark, &black, &strip, g, r.ix, r.iy)
		if r.close >= 0 {
			o2closeGlyph(&white, &dark, g.sub(r.close, r.gy, o2glyph, o2glyph), st.ClosePress)
		}
		if r.hide >= 0 {
			o2hideGlyph(&white, &dark, g.sub(r.hide, r.gy, o2glyph, o2glyph))
		}
		if r.max >= 0 {
			o2maxGlyph(&white, &dark, g.sub(r.max, r.gy, o2glyph, o2glyph))
		}
	}
	fill, tc := c.title, c.titleTxt
	if !st.Active {
		fill, tc = c.titleOff, c.titleOffTxt
	}
	well.fill(ctx, fill)
	strip.fill(ctx, c.title)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	if r.ok && title != "" && wg.w > 14 {
		// Bold, left-aligned, ten pixels from the well's edge.
		l.drawFittedText(ctx, l.BoldFont(), title, wg.at(10, 1, wg.w-12, wg.h-2), tc, AlignStart, 0)
	}
}

// ---- controls -------------------------------------------------------------------

// DrawPanel: a raised panel is the face in the two-pixel bevel; a flat one
// is the plain face.
func (e os2Engine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if raised {
		c.box(ctx, g, false)
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
}

// DrawButton is Warp 4's push button (see pushButton); the pressed label
// moves a pixel, a disabled one is halftoned.
func (e os2Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 6 {
		return
	}
	in, pressed := c.pushButton(ctx, g, st)
	lb := in.rect()
	if pressed {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	c.label(l, ctx, l.body, label, -1, lb, c.text, AlignCenter, st.Disabled(), c.face)
}

// toggleLabel draws a check box or radio caption (black; halftoned when
// disabled) and, with keyboard focus, the dotted rectangle round it.
func (e os2Engine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := o2colors(l)
	if label == "" {
		if st.Focused() {
			e.DrawFocusRing(l, ctx, box.Inset(-l.S(2)).Intersect(b))
		}
		return
	}
	f := l.body
	gap := min(l.S(6), 8)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	c.label(l, ctx, f, label, -1, lb, c.text, AlignStart, st.Disabled(), c.face)
	if st.Focused() {
		e.DrawFocusRing(l, ctx, labelFocusRect(f, label, lb, b))
	}
}

func (e os2Engine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	u := rpU(l)
	y := b.Min.Y + (b.Dy()-side)*0.5
	// Centre the box, not its cell: the tick's room sits above it.
	if y-u >= b.Min.Y && side >= 12*u {
		y -= u
	}
	box := paintengine2d.XYWH(b.Min.X, y, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e os2Engine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch: Warp 4 had none; this one is drawn in its manner — an entry
// field's sunken slot, the ribbon colour when on, and a raised knob that
// sinks while pressed.
func (e os2Engine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := o2colors(l)
	tw, th := min(l.metrics.SwitchW, b.Dx()), min(l.metrics.SwitchH, b.Dy())
	g := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th), rpU(l))
	if g.w >= 14 && g.h >= 10 {
		c.fieldBox(ctx, g)
		in := g.inset(2)
		if on {
			ctx.DrawRect(in.rect(), paintengine2d.Fill(c.ribbon))
		}
		kw := in.w / 2
		kx := 0
		if on {
			kx = in.w - kw
		}
		c.box(ctx, in.sub(kx, 0, kw, in.h), st.Pressed() && !st.Disabled())
		if st.Disabled() {
			c.halftone(l, ctx, g.rect(), c.face)
		}
	}
	e.toggleLabel(l, ctx, b, g.rect(), st, label)
}

// DrawComboBox is a drop-down list: the entry field's well with its raised
// button inside at the right, a 6 × 3 black triangle on it, sunk while the
// list is down. With keyboard focus the text is selected.
func (e os2Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 16 || g.h < 10 {
		return
	}
	c.fieldBox(ctx, g)
	in := g.inset(2)
	bw := min(16, in.w/3)
	btn := in.sub(in.w-bw, 0, bw, in.h)
	pressed := (open || st.Pressed()) && !st.Disabled()
	c.box(ctx, btn, pressed)
	tri := o2tri(6, DirDown)
	ox, oy := (btn.w-tri.w)/2, (btn.h-tri.h)/2
	if pressed {
		ox, oy = ox+1, oy+1
	}
	var k rpInk
	tri.emit(&k, btn, ox, oy)
	k.fill(ctx, c.black)
	f := l.body
	lb := in.at(3, 0, in.w-bw-5, in.h)
	fg, bg := c.text, c.field
	if st.Focused() && !open && !st.Disabled() && text != "" {
		hb := rpGridAt(ctx, rpTextBox(f, text, lb, AlignStart).Inset(-2*u).Intersect(in.at(1, 1, in.w-bw-2, in.h-2)), u)
		ctx.DrawRect(hb.rect(), paintengine2d.Fill(c.sel))
		fg, bg = c.selTxt, c.sel
	}
	c.label(l, ctx, f, text, -1, lb, fg, AlignStart, st.Disabled(), bg)
	if st.Disabled() {
		c.halftone(l, ctx, btn.at(ox, oy, tri.w, tri.h), c.face)
	}
}

// fieldOpts: #808080 selections with white text (kept without focus), a
// one-pixel cursor, disabled text halftoned into bg.
func (c *o2) fieldOpts(bg paintengine2d.Color) rpFieldOpts {
	return rpFieldOpts{
		text: c.text, muted: c.dim, sel: c.sel, selTxt: c.selTxt,
		caret: c.text, caretKind: rpCaretBar, ditherOff: true, bg: bg,
	}
}

// DrawTextField is an entry field: white in the two-pixel sunken frame.
func (e os2Engine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := o2colors(l)
	u := rpU(l)
	c.fieldBox(ctx, rpGridAt(ctx, b, u))
	pad := max(l.metrics.FieldPad, 4*u)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+2*u, b.Dx()-2*pad, b.Dy()-4*u)
	rpFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.fieldOpts(c.field))
}

// DrawTextArea is a multi-line entry field in the same frame.
func (e os2Engine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := o2colors(l)
	u := rpU(l)
	c.fieldBox(ctx, rpGridAt(ctx, b, u))
	pad := max(l.metrics.FieldPad, 4*u)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-2*pad, b.Dy()-2*pad)
	rpAreaText(l, ctx, inner, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, c.fieldOpts(c.field))
}

// DrawSpinner is the spin button's pair of raised arrow buttons, stacked;
// the held one sinks.
func (e os2Engine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 6 || g.h < 8 {
		return
	}
	mid := g.h / 2
	var face, white, dark, black rpInk
	var ghost [2]paintengine2d.Rect
	for i, dir := range [2]Direction{DirUp, DirDown} {
		y, h, pressed := 0, mid, upPress
		if i == 1 {
			y, h, pressed = mid, g.h-mid, downPress
		}
		pressed = pressed && !st.Disabled()
		bg := g.sub(0, y, g.w, h)
		n := 2
		if bg.w < 10 || bg.h < 9 {
			n = 1
		}
		face.add(bg.rect())
		o2bevel(&white, &dark, bg, 0, 0, bg.w, bg.h, n, pressed)
		tri := o2tri(min(6, bg.w-2*n-2), dir)
		ox, oy := (bg.w-tri.w)/2, (bg.h-tri.h)/2
		if pressed {
			ox, oy = ox+1, oy+1
		}
		tri.emit(&black, bg, ox, oy)
		ghost[i] = bg.at(ox, oy, tri.w, tri.h)
	}
	face.fill(ctx, c.face)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	if st.Disabled() {
		for _, r := range ghost {
			c.halftone(l, ctx, r, c.face)
		}
	}
}

// DrawTabBar: the window face above Warp 4's notebook page, whose top edge
// — the two white rows of its raised bevel — runs along the bar the lip's
// depth above its bottom; the tabs stand on that edge. (The page's sides
// are DrawTabPane's: pages often paint over them.)
func (e os2Engine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	if g.w < 8 || g.h < 3*o2lip {
		return
	}
	top := g.h - o2lip
	var white, dark rpInk
	white.cells(g, 0, top, g.w-1, 1)
	white.cells(g, 0, top+1, g.w-2, 1)
	dark.cells(g, g.w-1, top, 1, 2)
	dark.cells(g, g.w-2, top+1, 1, 1)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
}

// DrawTab is one of Warp 4's major tabs: a pastel trapezoid (its colour
// picked by its place in the strip, see o2tabIndex) whose top and left
// slant, one cell in per three rows, are white and whose right slant
// carries a two-pixel shadow. The selected tab has no line under it: its
// colour runs four rows into the page as a lip that narrows a cell each
// side per row. Pressed, the edges invert; with keyboard focus the label
// is ringed with dots.
func (e os2Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	lip := o2lip
	if g.h < 3*o2lip {
		lip = 0
	}
	t := g.h - lip // the tab's own rows
	if g.w < 12 || t < 6 {
		return
	}
	col := c.tabs[o2tabIndex(b.Min.X)]
	// Warp's slant is a cell per three rows; a tab too narrow for that
	// around its label (the tab bar pads labels by 28 pixels at any scale)
	// steepens just enough for the label to fit at the top of its line.
	f := l.body
	ty := max((t-int(f.Height()/u+0.5))/2, 0)
	k := 3
	if label != "" {
		tw := int(math.Ceil(float64(f.Advance(label)/u))) + 2
		if room := (g.w - 3 - tw) / 2; room <= 0 {
			k = t
		} else if (t-1-ty)/k > room {
			k = (t - 1 - ty + room - 1) / room
		}
	}
	e0 := min((t-1)/k, (g.w-8)/2)
	slant := func(y int) int { return min((t-1-y)/k, e0) }
	var face, white, dark rpInk
	white.cells(g, e0, 0, g.w-2*e0-2, 1)
	dark.cells(g, g.w-e0-2, 0, 2, 1)
	for y := 1; y < t; {
		s := slant(y)
		y2 := y + 1
		for y2 < t && slant(y2) == s {
			y2++
		}
		n := y2 - y
		white.cells(g, s, y, 1, n)
		face.cells(g, s+1, y, g.w-2*s-3, n)
		dark.cells(g, g.w-s-2, y, 2, n)
		y = y2
	}
	if selected {
		for i := 0; i < lip && 2*i < g.w; i++ {
			face.cells(g, i, t+i, g.w-2*i, 1)
		}
	}
	face.fill(ctx, col)
	if st.Pressed() && !st.Disabled() {
		white.fill(ctx, c.dark)
		dark.fill(ctx, c.hi)
	} else {
		white.fill(ctx, c.hi)
		dark.fill(ctx, c.dark)
	}
	// The label's room: the tab's width at the top of the text line.
	within := func(y int) paintengine2d.Rect {
		s := slant(y)
		return g.at(s+1, 1, g.w-2*s-3, t-1)
	}
	lb := within(ty)
	c.label(l, ctx, f, label, -1, lb, c.text, AlignCenter, st.Disabled(), col)
	if st.Focused() && selected {
		fr := lb.Inset(2 * u)
		if tb := rpTextBox(f, label, lb, AlignCenter); !tb.Empty() {
			fr = tb.Inset(-2 * u).Intersect(within(max(ty-2, 0)))
		}
		c.dotted(l, ctx, fr, c.black)
	}
}

// DrawMenuBar is the plain #CCCCCC menu bar.
func (e os2Engine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(o2colors(l).face))
}

// DrawMenuTitle: an open (or pressed) title is the navy bar with white
// text, flush with the menu below it; the mnemonic is underlined.
func (e os2Engine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg, bg := c.text, c.face
	if (open || st.Pressed()) && !st.Disabled() && g.h > 2 {
		ctx.DrawRect(g.at(0, 1, g.w, g.h-1), paintengine2d.Fill(c.menuHi))
		fg, bg = c.menuTxt, c.menuHi
	}
	c.label(l, ctx, l.body, label, underline, b, fg, AlignCenter, st.Disabled(), bg)
	if st.Focused() && !open && g.w > 4 && g.h > 4 {
		c.dotted(l, ctx, g.at(1, 2, g.w-2, g.h-3), c.black)
	}
}

// DrawMenuFrame: #CCCCCC in a one-pixel frame, white on the top and left,
// black on the bottom and right.
func (e os2Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var white, black rpInk
	rpEdge(&white, &black, g, 0, 0, g.w, g.h, 1)
	white.fill(ctx, c.hi)
	black.fill(ctx, c.black)
}

// DrawMenuItem: the hot item is a navy bar across the whole menu with white
// text; check marks and the submenu triangle are solid, disabled items
// halftoned, separators etched.
func (e os2Engine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := o2colors(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*u, b.Dy()), u)
	if row.Separator {
		if full.w > 6 {
			var white, dark rpInk
			y := full.h/2 - 1
			dark.cells(full, 2, y, full.w-4, 1)
			white.cells(full, 2, y+1, full.w-4, 1)
			white.fill(ctx, c.hi)
			dark.fill(ctx, c.dark)
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	col, bg := c.text, c.face
	if hot {
		ctx.DrawRect(full.rect(), paintengine2d.Fill(c.menuHi))
		col, bg = c.menuTxt, c.menuHi
	}
	f := l.body
	mark := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked && row.Radio:
		if d := min(6, mark.w-2, mark.h-4); d >= 3 {
			var k rpInk
			rpShape{w: d, h: d, r: float32(d) * 0.5}.fill(&k, mark, (mark.w-d)/2, (mark.h-d)/2)
			k.fill(ctx, col)
		}
	case row.Checked:
		if n := min(11, mark.w-2, mark.h-4); n >= 6 {
			m := o2mark(n)
			var k rpInk
			m.emit(&k, mark, (mark.w-n)/2, (mark.h-n)/2)
			k.fill(ctx, col)
		}
	case row.Icon != IconNone:
		side := min(ch.CheckCol()-l.S(2), b.Dy()-l.S(4))
		ib := paintengine2d.XYWH(b.Min.X+(ch.CheckCol()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		l.drawToolIcon(ctx, ib, row.Icon, col)
		if st.Disabled() {
			c.halftone(l, ctx, ib, bg)
		}
	}
	labelRight := b.Max.X
	if row.Submenu {
		ab := rpGridAt(ctx, paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow+l.S(4), b.Dy()), u)
		tri := o2tri(7, DirRight)
		var k rpInk
		tri.emit(&k, ab, 0, (ab.h-tri.h)/2)
		k.fill(ctx, col)
		labelRight = ch.ArrowMinX(b.Max.X)
	}
	if row.Shortcut != "" {
		tw := f.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		c.label(l, ctx, f, row.Shortcut, -1, paintengine2d.XYWH(sx, b.Min.Y, tw+l.S(2), b.Dy()), col, AlignStart, st.Disabled(), bg)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	c.label(l, ctx, f, row.Label, row.Underline, paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()), col, AlignStart, st.Disabled(), bg)
}

// DrawProgressBar: Warp 4 had no progress control (programs used a
// read-only slider with a ribbon strip); this is the entry field's well
// with the navy ribbon filling from the left, a moving block when busy.
func (e os2Engine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 10 || g.h < 8 {
		return
	}
	c.fieldBox(ctx, g)
	in := g.inset(3)
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		span := max(in.w/4, 4)
		x := int(float32(in.w+span)*phase) - span
		a, z := max(x, 0), min(x+span, in.w)
		if z > a {
			ctx.DrawRect(in.at(a, 0, z-a, in.h), paintengine2d.Fill(c.ribbon))
		}
	} else if n := int(float32(in.w)*clamp1(t) + 0.5); n > 0 {
		ctx.DrawRect(in.at(0, 0, n, in.h), paintengine2d.Fill(c.ribbon))
	}
	if st.Disabled() {
		c.halftone(l, ctx, in.rect(), c.field)
	}
}

// DrawSlider is a Warp 4 slider: a sunken shaft and the raised arm with an
// engraved line down its middle (an addition of this drawing); the arm
// sinks while dragged, and keyboard focus rings the control with dots.
func (e os2Engine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 20 || g.h < 10 {
		return
	}
	kw, kh := 11, min(g.h-2, 19)
	sh := 6
	c.fieldBox(ctx, g.sub(kw/2-2, (g.h-sh)/2, g.w-2*(kw/2)+4, sh))
	kx := int(float32(g.w-kw)*clamp1(t) + 0.5)
	arm := g.sub(kx, (g.h-kh)/2, kw, kh)
	c.box(ctx, arm, st.Pressed() && !st.Disabled())
	if kh > 8 {
		var white, dark rpInk
		dark.cells(arm, kw/2, 3, 1, kh-6)
		white.cells(arm, kw/2+1, 3, 1, kh-6)
		white.fill(ctx, c.hi)
		dark.fill(ctx, c.dark)
	}
	if st.Disabled() {
		c.halftone(l, ctx, g.rect(), c.face)
	}
	if st.Focused() {
		c.dotted(l, ctx, g.rect(), c.black)
	}
}

// DrawListRow: the selected item is #808080 with white text, in and out of
// focus alike; the current item of a focused list gets cursored emphasis.
func (e os2Engine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a tree view item: the plus / minus button and the name,
// which alone carries the selection and the cursored emphasis.
func (e os2Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := o2colors(l)
	u := rpU(l)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(3)
	hb := rpGridAt(ctx, paintengine2d.XYWH(lx-l.S(3), b.Min.Y+u, f.Advance(label)+l.S(6), b.Dy()-2*u).Intersect(b), u).rect()
	fg := c.text
	if st.Checked() && !hb.Empty() {
		ctx.DrawRect(hb, paintengine2d.Fill(c.sel))
		fg = c.selTxt
	}
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text)
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() && !hb.Empty() {
		e.ItemFocus(l, ctx, hb, st)
	}
}

// DrawTableCell: the selected row is #808080 with white text; the table
// rings the current row with ItemFocus after its cells.
func (e os2Engine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := o2colors(l)
	fg := c.text
	if st.Checked() {
		ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(c.sel))
		fg = c.selTxt
	}
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, align, 0)
}

// DrawTableHeader: column headings are raised one-pixel buttons, sunk
// while pressed; the sort column carries a solid triangle.
func (e os2Engine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var white, dark, black rpInk
	o2bevel(&white, &dark, g, 0, 0, g.w, g.h, 1, pressed)
	lb := b
	if pressed {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	aw := float32(0)
	if sorted && g.w > 24 {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		tri := o2tri(7, dir)
		ag := rpGridAt(ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(2), lb.Min.Y, aw, lb.Dy()), u)
		tri.emit(&black, ag, (ag.w-tri.w)/2, (ag.h-tri.h)/2)
	}
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
	c.label(l, ctx, l.body, label, -1, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()), c.text, AlignStart, st.Disabled(), c.face)
}

// DrawToolBar is the face over an etched line.
func (e os2Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	if g.h < 4 {
		return
	}
	var white, dark rpInk
	dark.cells(g, 0, g.h-2, g.w, 1)
	white.cells(g, 0, g.h-1, g.w, 1)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
}

// DrawToolButton: flat until the pointer arrives, then a raised two-pixel
// button; pressed and latched tools sink and their icon moves a pixel.
// Disabled tools are halftoned.
func (e os2Engine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b.Inset(u), u)
	sunk := (st.Pressed() || (st.Toggle() && st.Checked())) && !st.Disabled()
	switch {
	case st.Disabled():
	case sunk:
		c.box(ctx, g, true)
	case st.Hovered():
		c.box(ctx, g, false)
	}
	cb := b
	if sunk {
		cb = cb.Translate(paintengine2d.Pt(u, u))
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := cb.Min.X + pad
	ctx.Save()
	ctx.ClipRect(b)
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, cb.Min.Y+(cb.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(cb.Min.X+(cb.Dx()-iconSide)*0.5, cb.Min.Y+(cb.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, c.text)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, cb.Min.Y, cb.Max.X-pad-x, cb.Dy()), c.text, AlignStart, 0)
	}
	ctx.Restore()
	if st.Disabled() {
		c.halftone(l, ctx, g.rect(), c.face)
	}
	if st.Focused() && g.w > 6 && g.h > 6 {
		c.dotted(l, ctx, g.inset(3).rect(), c.black)
	}
}

// DrawStatusBar: Warp 4 had no status bar control; the parts sit in
// one-pixel sunken panels on the face.
func (e os2Engine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	if len(parts) == 0 || g.w < 8 || g.h < 6 {
		return
	}
	var white, dark rpInk
	slot := g.w / len(parts)
	for i := range parts {
		x := i * slot
		w := slot
		if i == len(parts)-1 {
			w = g.w - x
		}
		if w > 4 {
			rpEdge(&dark, &white, g, x+1, 1, w-2, g.h-2, 1)
		}
	}
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	for i, s := range parts {
		x := i * slot
		w := slot
		if i == len(parts)-1 {
			w = g.w - x
		}
		if w > 10 {
			l.drawFittedText(ctx, l.body, s, g.at(x+5, 2, w-10, g.h-4), c.text, AlignStart, 0)
		}
	}
}

// DrawTitleBar is a panel heading: the bold navy title (static text) over
// an etched line.
func (e os2Engine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	if g.h >= 4 {
		var white, dark rpInk
		dark.cells(g, 0, g.h-2, g.w, 1)
		white.cells(g, 0, g.h-1, g.w, 1)
		white.fill(ctx, c.hi)
		dark.fill(ctx, c.dark)
	}
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.static, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.text, AlignStart, 0)
	}
}

// DrawAccordionHeader: Warp 4 had none; a raised bar with the tree view's
// plus / minus button, sunk while pressed.
func (e os2Engine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	pressed := st.Pressed() && !st.Disabled()
	c.box(ctx, g, pressed)
	lb := b
	if pressed {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	e.Expander(l, ctx, paintengine2d.XYWH(lb.Min.X+l.S(5), lb.Min.Y, l.S(14), lb.Dy()), expanded, c.text)
	c.label(l, ctx, l.body, title, -1, paintengine2d.XYWH(lb.Min.X+l.S(26), lb.Min.Y, lb.Dx()-l.S(30), lb.Dy()), c.text, AlignStart, st.Disabled(), c.face)
	if st.Focused() && g.w > 8 && g.h > 8 {
		c.dotted(l, ctx, g.inset(3).rect(), c.black)
	}
}

// DrawSeparator is an etched line, #808080 over white.
func (e os2Engine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	var white, dark rpInk
	if vertical {
		x := g.w/2 - 1
		dark.cells(g, x, 2, 1, g.h-4)
		white.cells(g, x+1, 2, 1, g.h-4)
	} else {
		y := g.h/2 - 1
		dark.cells(g, 0, y, g.w, 1)
		white.cells(g, 0, y+1, g.w, 1)
	}
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
}

// DrawSplitter: the face with an etched line down its middle; a raised bar
// under the pointer, sunk while dragged.
func (e os2Engine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	switch {
	case st.Pressed():
		c.box(ctx, g, true)
	case st.Hovered():
		c.box(ctx, g, false)
	default:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
		e.DrawSeparator(l, ctx, b, vertical)
	}
}

// DrawTooltip is Warp 4's fly-over help: black text on pale yellow in a
// one-pixel black line (the yellow is inferred).
func (e os2Engine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := o2colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.info))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.black)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), ReadableOn(c.info, 4.5, c.black), AlignStart, 0)
}

// DrawOverlay: Presentation Manager never dimmed what lay behind a dialog.
func (os2Engine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- packs --------------------------------------------------------------------

func os2Packs() []ThemePack {
	face := hexColor("#cccccc")
	black, white, navy, dark := hexColor("#000000"), hexColor("#ffffff"), hexColor("#000080"), hexColor("#808080")
	pal := Palette{
		Background: face, Surface: face, SurfaceAlt: face,
		Border: black, Divider: dark,
		Text: navy, TextMuted: dark, TextOnAccent: white,
		Accent: hexColor("#2b00aa"), AccentHover: hexColor("#2b00aa"), AccentPress: navy,
		Field: white, FieldBorder: dark,
		Focus: black, Selection: dark,
		Track: hexColor("#c0c0c0"), Thumb: face,
		Highlight: white, Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: navy, MenuHoverBorder: navy, MenuGutter: face,
		Danger: hexColor("#c00000"), Success: hexColor("#007000"), Warning: hexColor("#9a5a00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: white, BevelDark: dark,
	}
	extra := map[string]string{
		"dark": "#808080", "hi": "#ffffff", "controlText": "#000000", "selText": "#ffffff",
		"menuHi": "#000080", "menuHiText": "#ffffff",
		"title": "#2b00aa", "titleText": "#ffffff", "titleOff": "#cccccc", "titleOffText": "#555555",
		"shaft": "#c0c0c0", "shaftOff": "#cccccc", "info": "#ffffc0", "ribbon": "#000080",
	}
	for i, d := range o2tabColors {
		extra["tab"+itoa(i)] = d
	}
	tok := ThemeTokens{
		Engine:   "os2",
		Bevel:    BevelClassic3D,
		Family:   ThemeLight,
		Palette:  pal,
		Extra:    map[string]paintengine2d.Color{},
		Hot:      ChromeState{Fill: navy, Border: navy},
		Selected: ChromeState{Fill: dark, Border: dark},
		Focus:    ChromeState{Fill: black.WithAlpha(0.15), Border: black},
	}
	for k, v := range extra {
		tok.Extra[k] = hexColor(v)
	}
	return []ThemePack{{
		Name: "os2warp", Label: "OS/2 Warp 4", Year: 1996, Lineage: "IBM",
		Summary: "IBM's Warp 4 desktop: #CCCCCC dialogs, grooved two-pixel bevels, pastel notebook tabs, a sunken title well and halftoned disabled items.",
		Era:     "OS/2", Palette: ThemeLight, Tokens: tok,
	}}
}
