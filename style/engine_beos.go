package style

import "github.com/codemodify/paintengine2d"

// beosEngine paints BeOS, the look Be Inc. shipped from R3 (1998) to R5
// (2000): flat #D8D8D8 panels whose every bevel is a tint of that grey, the
// yellow window tab, and one blue that marks keyboard focus everywhere.
//
// Everything is drawn in whole pixels (engine_system7_pixel.go). Chrome keeps
// its native size — one-pixel bevels, 15-pixel scroll bars, 13-pixel check
// boxes, 14-pixel close and zoom boxes — while control heights follow the
// toolkit's 16px font (BeOS set its controls in 10-point Swis721, about two
// thirds of that):
//
//   - Windows: the tab is only as wide as its title and stands on the
//     frame's top-left corner, the close box at its left, the zoom box (two
//     overlapping squares) at its right, the bold title between; the frame is
//     five pixels (raised, flat, sunken, a grey line). The close and zoom
//     boxes are the only gradients BeOS drew.
//   - Push buttons: a #606060 outline with its corners cut, a white highlight
//     inside the top and left, grey on the bottom and right, an #E8E8E8 face;
//     the default button wears a second cut-corner outline; a pressed button
//     inverts its inside. Keyboard focus is a blue line under the label with
//     a white line under it — on buttons, check boxes, radios, tabs, pop-ups
//     and slider knobs; text controls and focused lists turn their frame blue.
//   - Check boxes are sunken squares crossed with a blue X; radio buttons are
//     sunken circles holding a blue dot with a white glint.
//   - Scroll bars: a grey outline round arrow boxes with embossed triangles at
//     both ends (R5 could double them; the toolkit places one pair), a
//     #C8C8C8 track and a proportional knob with three raised dots.
//   - Menus: #D8D8D8 in a dark outline; the item under the pointer turns
//     #999999 and keeps black text. Pop-up menu fields are raised boxes with
//     a drop shadow and a small triangle.
//   - Tab views: tabs with chamfered corners whose sides flare toward the
//     base; the selected tab opens into its page and overlaps its neighbours.
//   - Lists select in #B0B0B0 with black text, focused or not; outline lists
//     open with grey triangles. Status bars (progress) are #3296FF in a
//     sunken frame. Menus, windows and tooltips cast no shadow, and nothing
//     dims behind a dialog.
//
// Inventions, where BeOS had no such control or state: switches, spinners,
// splitters, accordion headers, table headers (after Tracker's column
// titles), tooltips (#FFFFD8 is a later Haiku colour; R5 had none), hover
// on tool buttons and menu titles, the pressed look of check boxes, radios,
// tabs, knobs and thumbs, the blue frame round the current row of a list
// (BListView drew none; later column list views did) and the busy bar.
//
// Pack data: Palette.Background is the panel colour and every bevel grey is
// one of its Interface Kit tints (lighten 1 and 2, darken 1 to 4);
// Palette.Selection is the list selection.
//
//	extra:
//	  tab, tabHi, tabLo          active window tab: fill, inner highlight, inner shadow
//	  tabOff, tabOffText         inactive tab fill and title
//	  boxDark, boxLight          close box rings; zoomDark the zoom box's dark ring
//	  boxOffDark, zoomOffDark    their dark rings on an inactive tab
//	  boxTop, boxEnd, boxOffEnd  the box faces' gradient ends (the middle is tab)
//	  nav, navRim                keyboard-navigation blue; the radio dot's rim
//	  status                     progress bar
//	  menuSel, menuHi, edge      menu selection, menu highlight, menu outline
//	  mid, arrowLo, track        frame grey, scroll arrow shadow, scroll track
//	  sliderLine, sliderBar      slider groove edge and bar
//	  knobEdge, knobFace, knobShade  slider knob
//	  expander, submenu          outline-list triangle fill, submenu mark fill
//	  info                       tooltip fill
type beosEngine struct{ BaseEngine }

func init() {
	RegisterEngine(beosEngine{})
	for _, p := range beosPacks() {
		RegisterPack(p)
	}
}

func (beosEngine) ID() string { return "beos" }

// DefaultMetrics are BeOS's proportions at the toolkit's 16px UI font: the
// 23-pixel BButton becomes a 30px slot whose 24px body leaves room for the
// default outline, the 20-pixel tab 26px, the 19-pixel text control 26px;
// check boxes (13px) and scroll bars (15px) keep their native size.
func (beosEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 3,
		Square:    true, BevelDepth: 2,
		ControlH: 30, FieldH: 26, ComboH: 26,
		Checkbox: 13, Radio: 13,
		MenuItemH: 24, MenuBarH: 26, TabH: 26, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 16, SliderH: 24, Thumb: 15,
		Scroll: 15, Pad: 10, FieldPad: 4, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24, SpinnerW: 15, SwitchW: 36, SwitchH: 17,
	}
}

// ---- colours ------------------------------------------------------------------

// The Interface Kit's tint factors (the Be Book's B_LIGHTEN_* and B_DARKEN_*).
const (
	beLighten2 = 0.385
	beLighten1 = 0.590
	beDarken1  = 1.147
	beDarken2  = 1.295
	beDarken3  = 1.407
	beDarken4  = 1.555
)

// beTint is tint_color's arithmetic: a factor under 1 moves c toward white
// (0 is white), one over 1 toward black (2 is black), in whole 8-bit steps.
// Of the panel grey 216 it gives 240, 232, 184, 152, 128 and 96.
func beTint(c paintengine2d.Color, t float32) paintengine2d.Color {
	ch := func(v float32) float32 {
		if t < 1 {
			v += (1 - v) * (1 - t)
		} else {
			v *= 2 - t
		}
		return float32(int(v*255+0.5)) / 255
	}
	return paintengine2d.RGBA(ch(c.R), ch(c.G), ch(c.B), c.A)
}

// beInvert is c with its colour channels inverted (a pressed BButton).
func beInvert(c paintengine2d.Color) paintengine2d.Color {
	return paintengine2d.RGBA(1-c.R, 1-c.G, 1-c.B, c.A)
}

// beColors is the resolved colour set of a look.
type beColors struct {
	panel, white, black, l1, l2, d1, d2, d3, d4 paintengine2d.Color
	text, dim, muted, field, fieldText          paintengine2d.Color
	rowSel, rowText, menuSel, menuHi, edge, mid paintengine2d.Color
	nav, navRim, navOff, navOffRim              paintengine2d.Color
	tab, tabHi, tabLo, tabOff, tabOffText       paintengine2d.Color
	boxDk, boxLt, zoomDk, boxOffDk, zoomOffDk   paintengine2d.Color
	arrowLo, track, slLine, slBar               paintengine2d.Color
	knobTL, knobFace, knobLo, status, statusOff paintengine2d.Color
	expFill, subFill, info                      paintengine2d.Color
	dnFace, dnLo, dnLo2                         paintengine2d.Color // a pressed button's inverted inside
	faceOn, faceDown, faceOff                   []paintengine2d.GradientStop
}

type beKey struct{}

// beColorsOf is the look's colour set, built once per look.
func beColorsOf(l *Classic) *beColors {
	return l.Memo(beKey{}, func() any { return beBuild(l) }).(*beColors)
}

func beBuild(l *Classic) *beColors {
	p := l.palette
	x := func(k, def string) paintengine2d.Color { return l.X(k, Hex(def)) }
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	panel := p.Background
	c := &beColors{
		panel: panel, white: white, black: black,
		l1: beTint(panel, beLighten1), l2: beTint(panel, beLighten2),
		d1: beTint(panel, beDarken1), d2: beTint(panel, beDarken2),
		d3: beTint(panel, beDarken3), d4: beTint(panel, beDarken4),
		text: p.Text, muted: p.TextMuted, field: p.Field,
	}
	// Disabled labels are the panel darkened three steps.
	c.dim = c.d3
	// Not l.fieldText(): it memoises too, and Memo does not nest.
	c.fieldText = p.Text
	if !colorUnset(p.Field) {
		c.fieldText = ReadableOn(p.Field, 4.5, p.Text)
	}
	c.rowSel = p.Selection
	if c.rowSel.A < 0.9 {
		c.rowSel = Hex("#b0b0b0")
	}
	c.rowText = ReadableOn(c.rowSel, 4.5, c.fieldText, black, white)
	c.menuSel = x("menuSel", "#999999")
	c.menuHi = x("menuHi", "#efefef")
	c.edge = x("edge", "#646464")
	c.mid = x("mid", "#888888")
	c.nav = x("nav", "#0000e5")
	c.navRim = x("navRim", "#00009a")
	c.navOff = beTint(c.nav, beLighten2)
	c.navOffRim = beTint(c.navRim, beLighten2)
	c.tab = x("tab", "#ffcb00")
	c.tabHi = x("tabHi", "#ffff50")
	c.tabLo = x("tabLo", "#af7b00")
	c.tabOff = x("tabOff", "#e8e8e8")
	c.tabOffText = x("tabOffText", "#505050")
	c.boxDk = x("boxDark", "#b78200")
	c.boxLt = x("boxLight", "#ffff3f")
	c.zoomDk = x("zoomDark", "#d29d00")
	c.boxOffDk = x("boxOffDark", "#a0a0a0")
	c.zoomOffDk = x("zoomOffDark", "#bbbbbb")
	top, end := x("boxTop", "#ffec21"), x("boxEnd", "#eab500")
	c.faceOn = []paintengine2d.GradientStop{Stop(0, top), Stop(0.5, c.tab), Stop(1, end)}
	c.faceDown = []paintengine2d.GradientStop{Stop(0, end), Stop(0.5, c.tab), Stop(1, top)}
	c.faceOff = []paintengine2d.GradientStop{Stop(0, white), Stop(1, x("boxOffEnd", "#d3d3d3"))}
	c.arrowLo = x("arrowLo", "#c0c0c0")
	c.track = x("track", "#c8c8c8")
	c.slLine = x("sliderLine", "#909090")
	c.slBar = x("sliderBar", "#505050")
	c.knobTL = x("knobEdge", "#787878")
	c.knobFace = x("knobFace", "#f0f0f0")
	c.knobLo = x("knobShade", "#c8c8c8")
	c.status = x("status", "#3296ff")
	c.statusOff = beTint(c.status, beLighten1)
	c.expFill = x("expander", "#909090")
	c.subFill = x("submenu", "#e7e7e7")
	c.info = x("info", "#ffffd8")
	c.dnFace, c.dnLo, c.dnLo2 = beInvert(c.l1), beInvert(c.d2), beInvert(panel)
	return c
}

// ---- drawing helpers -------------------------------------------------------------

// beCut appends the one-cell outline of the w × h cells at (x, y) with its
// four corner cells left out: the corners of BeOS buttons.
func beCut(k *rpInk, g rpGrid, x, y, w, h int) {
	if w < 3 || h < 3 {
		k.cells(g, x, y, w, h)
		return
	}
	k.cells(g, x+1, y, w-2, 1)
	k.cells(g, x+1, y+h-1, w-2, 1)
	k.cells(g, x, y+1, 1, h-2)
	k.cells(g, x+w-1, y+1, 1, h-2)
}

// beAnd is the cells set in both a and b.
func beAnd(a, b rpMask) rpMask { return a.minus(a.minus(b)) }

// beTri is a solid triangle pointing up, w cells across its base (odd) and h
// rows tall, its sides straight from the apex to the base corners: 9 × 8 is
// the scroll arrow, 9 × 5 the outline-list triangle, 5 × 3 the pop-up mark.
func beTri(w, h int) rpMask {
	if w%2 == 0 {
		w--
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m := rpNewMask(w, h)
	c := w / 2
	for y := 0; y < h; y++ {
		hw := c
		if h > 1 {
			hw = (2*c*y + h - 1) / (2 * (h - 1))
		}
		m.span(c-hw, c+hw+1, y)
	}
	return m
}

// beCross is the check box's X in s × s cells: two diagonals two cells wide
// that meet in a single cell when s is odd (one cell wide in a box too small
// for two).
func beCross(s int) rpMask {
	m := rpNewMask(s, s)
	if s < 5 {
		m.line(0, 0, s-1, s-1)
		m.line(s-1, 0, 0, s-1)
		return m
	}
	for y := 0; y < s; y++ {
		yy := min(y, s-1-y)
		if s%2 == 1 && yy == s/2 {
			m.set(yy, y)
			continue
		}
		m.span(yy, yy+2, y)
		m.span(s-2-yy, s-yy, y)
	}
	return m
}

// beTick is a menu check mark n cells wide: a short stroke down to a valley a
// third of the way across and a long one up to the top-right corner, both
// two cells wide.
func beTick(n int) rpMask {
	vx := n / 3
	vy := n - 1 - vx
	m := rpNewMask(n, vy+1)
	for d := 0; d < 2; d++ {
		m.line(d, vy-vx, vx+d, vy)
		m.line(vx+d, vy, n-1, vy-(n-1-vx-d))
	}
	return m
}

// well paints the BTextControl frame round the w × h cells at (x, y) of g
// and fills its inside: a #B8B8B8 and white outer ring round a #606060 and
// #D8D8D8 inner ring, which turns keyboard-navigation blue on all four sides
// when ring is set. A disabled well softens its inner ring.
func (c *beColors) well(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, fill paintengine2d.Color, disabled, ring bool) {
	if w < 5 || h < 5 {
		ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(fill))
		return
	}
	ctx.DrawRect(g.at(x+2, y+2, w-4, h-4), paintengine2d.Fill(fill))
	var oHi, oLo, iHi, iLo rpInk
	rpEdge(&oHi, &oLo, g, x, y, w, h, 1)
	if ring {
		iHi.frame(g, x+1, y+1, w-2, h-2, 1)
		iHi.fill(ctx, c.nav)
	} else {
		rpEdge(&iHi, &iLo, g, x+1, y+1, w-2, h-2, 1)
		hi, lo := c.d4, c.panel
		if disabled {
			hi, lo = c.d2, c.l2
		}
		iHi.fill(ctx, hi)
		iLo.fill(ctx, lo)
	}
	oHi.fill(ctx, c.d1)
	oLo.fill(ctx, c.white)
}

// navLine is BeOS's keyboard focus mark: a line of the navigation blue one
// row under the baseline of text laid out in box tb (rpTextBox's box for
// face f), and a white line under it, kept inside g.
func (c *beColors) navLine(ctx *paintengine2d.Context, g rpGrid, f *Font, tb paintengine2d.Rect) {
	if f == nil || tb.Dx() <= 0 || g.w < 1 || g.h < 2 {
		return
	}
	row := int((tb.Min.Y+f.Ascent-g.y)/g.u+0.5) + 1
	row = max(0, min(row, g.h-2))
	x0 := max(int((tb.Min.X-g.x)/g.u+0.5), 0)
	x1 := min(int((tb.Max.X-g.x)/g.u+0.5), g.w)
	if x1 <= x0 {
		return
	}
	ctx.DrawRect(g.at(x0, row, x1-x0, 1), paintengine2d.Fill(c.nav))
	ctx.DrawRect(g.at(x0, row+1, x1-x0, 1), paintengine2d.Fill(c.white))
}

// label draws text fitted into lb and, when focus is set, underlines it with
// the navigation mark inside g.
func (c *beColors) label(l *Classic, ctx *paintengine2d.Context, g rpGrid, f *Font, text string, lb paintengine2d.Rect, col paintengine2d.Color, align Align, focus bool) {
	if text == "" || lb.Empty() {
		return
	}
	l.drawFittedText(ctx, f, text, lb, col, align, 0)
	if focus {
		tb := rpTextBox(f, text, lb, align)
		tb.Min.X, tb.Max.X = max(tb.Min.X, lb.Min.X), min(tb.Max.X, lb.Max.X)
		c.navLine(ctx, g, f, tb)
	}
}

// button paints a BButton body filling g and returns its label colour: a
// #606060 outline with its corners cut; inside it a row of the face, then a
// white highlight two cells thick along the top and left; #989898 and then
// #D8D8D8 along the bottom and right; the #E8E8E8 face. Pressed, the inside
// inverts (R5's clicked button); disabled, outline and shadow fade.
func (c *beColors) button(ctx *paintengine2d.Context, g rpGrid, st ControlState) paintengine2d.Color {
	if g.w < 8 || g.h < 8 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.l1))
		return c.text
	}
	face, lo1, hi, lo2, line, fg := c.l1, c.d2, c.white, c.panel, c.d4, c.text
	switch {
	case st.Disabled():
		lo1, lo2, line, fg = c.panel, c.l1, c.d2, c.dim
	case st.Pressed():
		face, lo1, hi, lo2, fg = c.dnFace, c.dnLo, c.black, c.dnLo2, c.white
	}
	ctx.DrawRect(g.at(1, 1, g.w-2, g.h-2), paintengine2d.Fill(face))
	var kLo1, kHi, kLo2, kLine rpInk
	rpEdge(nil, &kLo1, g, 1, 1, g.w-2, g.h-2, 1)
	rpEdge(&kHi, &kLo2, g, 2, 2, g.w-4, g.h-4, 1)
	rpEdge(&kHi, nil, g, 3, 3, g.w-6, g.h-6, 1)
	beCut(&kLine, g, 0, 0, g.w, g.h)
	kLo1.fill(ctx, lo1)
	kLo2.fill(ctx, lo2)
	kHi.fill(ctx, hi)
	kLine.fill(ctx, line)
	return fg
}

// sunken paints a pushed-in tool button filling g: the cut-corner outline,
// #989898 inside its top and left, white inside its bottom and right.
func (c *beColors) sunken(ctx *paintengine2d.Context, g rpGrid, face paintengine2d.Color) {
	if g.w < 5 || g.h < 5 {
		return
	}
	ctx.DrawRect(g.at(1, 1, g.w-2, g.h-2), paintengine2d.Fill(face))
	var kLine, kHi, kLo rpInk
	beCut(&kLine, g, 0, 0, g.w, g.h)
	rpEdge(&kLo, &kHi, g, 1, 1, g.w-2, g.h-2, 1)
	kLo.fill(ctx, c.d2)
	kHi.fill(ctx, c.white)
	kLine.fill(ctx, c.d4)
}

// popup paints a BMenuField's box filling g and returns its label colour: a
// #989898 top and left edge with an #EFEFEF highlight inside, #B8B8B8 inside
// a #606060 bottom and right edge, and a #989898 drop shadow a cell below and
// right of that. It takes the menu selection while its menu is up.
func (c *beColors) popup(ctx *paintengine2d.Context, g rpGrid, st ControlState, open bool) paintengine2d.Color {
	w, h := g.w, g.h
	if w < 8 || h < 8 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
		return c.text
	}
	face, fg, edge := c.panel, c.text, c.d4
	switch {
	case st.Disabled():
		fg, edge = c.dim, c.d2
	case open || st.Pressed():
		face = c.menuSel
	}
	ctx.DrawRect(g.at(2, 2, w-5, h-5), paintengine2d.Fill(face))
	var kD2, kHi, kD1, kEdge rpInk
	kD2.cells(g, 0, 0, w-2, 1)
	kD2.cells(g, 0, 1, 1, h-3)
	kHi.cells(g, 1, 1, w-4, 1)
	kHi.cells(g, 1, 2, 1, h-5)
	kD1.cells(g, w-3, 1, 1, h-3)
	kD1.cells(g, 1, h-3, w-4, 1)
	kEdge.cells(g, w-2, 0, 1, h-1)
	kEdge.cells(g, 0, h-2, w-2, 1)
	kD2.cells(g, w-1, 1, 1, h-1)
	kD2.cells(g, 1, h-1, w-2, 1)
	kHi.fill(ctx, c.menuHi)
	kD1.fill(ctx, c.d1)
	kD2.fill(ctx, c.d2)
	kEdge.fill(ctx, edge)
	return fg
}

// knob paints BSlider's block thumb filling g: a #787878 edge along the top
// and left, the #F0F0F0 face, and a shadow of #C8C8C8, #909090 and black
// along the bottom and right. Pressed, the face greys (an invention).
func (c *beColors) knob(ctx *paintengine2d.Context, g rpGrid, st ControlState) {
	if g.w < 7 || g.h < 7 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.knobFace))
		return
	}
	face, e0, s1, s2, s3 := c.knobFace, c.knobTL, c.knobLo, c.slLine, c.black
	switch {
	case st.Disabled():
		face, e0, s1, s2, s3 = c.l2, c.d1, c.l1, c.d1, c.d2
	case st.Pressed():
		face = c.panel
	}
	ctx.DrawRect(g.at(1, 1, g.w-3, g.h-3), paintengine2d.Fill(face))
	var k0, k1, k2, k3 rpInk
	rpEdge(&k0, &k3, g, 0, 0, g.w, g.h, 1)
	rpEdge(nil, &k2, g, 1, 1, g.w-2, g.h-2, 1)
	rpEdge(nil, &k1, g, 2, 2, g.w-4, g.h-4, 1)
	k0.fill(ctx, e0)
	k1.fill(ctx, s1)
	k2.fill(ctx, s2)
	k3.fill(ctx, s3)
}

// ---- parts --------------------------------------------------------------------

func (e beosEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	if g.w < 3 || g.h < 3 {
		return fg
	}
	switch role {
	case RoleButton:
		return c.button(ctx, g, st)
	case RoleCombo:
		return c.popup(ctx, g, st, false)
	case RoleTool:
		switch {
		case st.Disabled():
		case st.Pressed():
			c.sunken(ctx, g, c.d1)
		case st.Toggle() && st.Checked():
			face := c.track
			if st.Hovered() {
				face = c.panel
			}
			c.sunken(ctx, g, face)
		case st.Hovered():
			// Tool bars had no standard look; under the pointer a tool
			// raises into a BButton.
			c.button(ctx, g, StateNone)
		}
		return fg
	case RoleField, RoleCheck:
		fill := c.field
		switch {
		case st.Disabled():
			fill = c.l2
		case role == RoleCheck && st.Pressed():
			fill = c.panel
		}
		c.well(ctx, g, 0, 0, g.w, g.h, fill, st.Disabled(), role == RoleField && st.Focused() && !st.Disabled())
		if role == RoleField && !st.Disabled() {
			return c.fieldText
		}
		return fg
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.rowSel))
			return c.rowText
		}
		return c.fieldText
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.menuSel))
		}
		return fg
	case RoleThumb:
		face := c.panel
		if st.Pressed() {
			face = c.track
		}
		ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
		var hi, lo, tr rpInk
		rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
		if g.h >= g.w {
			tr.cells(g, 0, g.h-1, g.w, 1)
		} else {
			tr.cells(g, g.w-1, 0, 1, g.h)
		}
		hi.fill(ctx, c.white)
		lo.fill(ctx, c.d1)
		tr.fill(ctx, c.d4)
		return fg
	case RoleTrack:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.track))
		return fg
	case RoleBar:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
		var hi, lo rpInk
		rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
		hi.fill(ctx, c.menuHi)
		lo.fill(ctx, c.d2)
		return fg
	case RoleSplitter:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
		if st.Hovered() || st.Pressed() {
			var hi, lo rpInk
			rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
			hi.fill(ctx, c.white)
			lo.fill(ctx, c.d2)
		}
		return fg
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	return fg
}

// CheckIndicator is BCheckBox's box: the text-control frame round a white
// square, crossed with a blue X two pixels wide when checked; disabled, a
// soft frame on #F0F0F0 and a pale X.
func (e beosEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5), 32)
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	fill := c.field
	switch {
	case st.Disabled():
		fill = c.l2
	case st.Pressed():
		fill = c.panel
	}
	c.well(ctx, g, 0, 0, n, n, fill, st.Disabled(), false)
	if !checked || n < 9 {
		return
	}
	m := beCross(n - 6)
	var k rpInk
	m.emit(&k, g, 3, 3)
	col := c.nav
	if st.Disabled() {
		col = c.navOff
	}
	k.fill(ctx, col)
}

// RadioIndicator is BRadioButton's circle: the check box's two rings drawn
// round (#B8B8B8 and white outside, #606060 and #D8D8D8 inside, split along
// the diagonal), white within, and when on a blue dot with a darker rim and
// a white glint at its upper left.
func (e beosEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5), 32)
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	outer, mid, core, br := rpNewMask(n, n), rpNewMask(n, n), rpNewMask(n, n), rpNewMask(n, n)
	outer.shape(rpShape{w: n, h: n, r: float32(n) * 0.5}, 0, 0)
	mid.shape(rpShape{w: n - 2, h: n - 2, r: float32(n-2) * 0.5}, 1, 1)
	core.shape(rpShape{w: n - 4, h: n - 4, r: float32(n-4) * 0.5}, 2, 2)
	for y := 0; y < n; y++ {
		br.span(n-1-y, n, y) // the lower right half, the anti-diagonal with it
	}
	ringO, ringI := outer.minus(mid), mid.minus(core)
	oLo, iLo := beAnd(ringO, br), beAnd(ringI, br)
	oHi, iHi := ringO.minus(br), ringI.minus(br)
	var kCore, kOHi, kOLo, kIHi, kILo rpInk
	core.emit(&kCore, g, 0, 0)
	oHi.emit(&kOHi, g, 0, 0)
	oLo.emit(&kOLo, g, 0, 0)
	iHi.emit(&kIHi, g, 0, 0)
	iLo.emit(&kILo, g, 0, 0)
	fill, ih, il := c.field, c.d4, c.panel
	switch {
	case st.Disabled():
		fill, ih, il = c.l2, c.d2, c.l2
	case st.Pressed():
		fill = c.panel
	}
	kCore.fill(ctx, fill)
	kOHi.fill(ctx, c.d1)
	kOLo.fill(ctx, c.white)
	kIHi.fill(ctx, ih)
	kILo.fill(ctx, il)
	if !selected {
		return
	}
	d := n - 6
	if d%2 == 0 {
		d--
	}
	if d < 3 {
		return
	}
	o := (n - d) / 2
	dot := rpNewMask(n, n)
	dot.shape(rpShape{w: d, h: d, r: float32(d) * 0.5}, o, o)
	rim := dot.outline()
	in := dot.minus(rim)
	var kIn, kRim, kGlint rpInk
	in.emit(&kIn, g, 0, 0)
	rim.emit(&kRim, g, 0, 0)
	blue, rimCol := c.nav, c.navRim
	if st.Disabled() {
		blue, rimCol = c.navOff, c.navOffRim
	}
	kIn.fill(ctx, blue)
	kRim.fill(ctx, rimCol)
	if d >= 5 {
		kGlint.cells(g, o+2, o+2, 2, 1)
		kGlint.fill(ctx, c.white)
	}
}

// Arrow is a solid whole-pixel triangle (sort marks, pop-up marks).
func (e beosEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 4 {
		return
	}
	w := min(n-2, 9)
	m := beTri(w, w/2+1).turn(dir)
	var k rpInk
	m.emit(&k, g, (g.w-m.w)/2, (g.h-m.h)/2)
	k.fill(ctx, col)
}

// Expander is BOutlineListView's latch: a triangle outlined in col (black)
// and filled #909090, 9 × 5 pointing down when open, 5 × 9 pointing right
// when closed.
func (e beosEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 7 {
		return
	}
	w, h := 9, 5
	if n < 11 {
		w, h = 7, 4
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	m := beTri(w, h).turn(dir)
	o := m.outline()
	in := m.minus(o)
	ox, oy := (g.w-m.w)/2, (g.h-m.h)/2
	var kIn, kOut rpInk
	in.emit(&kIn, g, ox, oy)
	o.emit(&kOut, g, ox, oy)
	kIn.fill(ctx, c.expFill)
	kOut.fill(ctx, col)
}

// MenuHighlight is the menu selection: flat #999999, the text stays black.
func (e beosEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(beColorsOf(l).menuSel))
}

func (e beosEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return beColorsOf(l).text
}

// FieldFocusRing: a focused BTextControl turns its inner ring blue.
func (beosEngine) FieldFocusRing(*Classic) bool { return true }

// DrawFocusRing is a one-pixel frame of the navigation blue just inside b.
func (e beosEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 3 || g.h < 3 {
		return
	}
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, beColorsOf(l).nav)
}

// ---- scroll bars -------------------------------------------------------------

// ScrollBarStyle: B_V_SCROLL_BAR_WIDTH, 15 pixels with the outline, an
// arrow box at each end, a knob no shorter than the bar is wide.
func (beosEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 15, Arrows: ArrowsEnds, MinThumb: 15}
}

// DrawScrollBarParts is BScrollBar: a #989898 outline; arrow boxes (white
// top and left, #C0C0C0 bottom and right) with an embossed 9 × 8 triangle; a
// #C8C8C8 track shaded #B8B8B8 two pixels deep along its leading side; the
// knob with a white edge, #B8B8B8 shadow, a #606060 line across its trailing
// end and three raised dots. A bar with nothing to scroll has no knob, pale
// faces and hollow arrows.
func (e beosEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, p.Bar, u)
	if g.w < 5 || g.h < 5 {
		return
	}
	active := !st.Disabled && !p.Thumb.Empty()
	along, across := g.h, g.w
	if !vertical {
		along, across = g.w, g.h
	}
	// sub maps (along, across) cells to a sub-grid of g.
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
	var fTrack, fPanel, fOff, fDown rpInk // faces
	var kD1, kL1, kLo, kWhite, kD4, kD2 rpInk
	kD2.frame(g, 0, 0, g.w, g.h, 1)
	lo, hi := 1, along-1
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	type end struct {
		r    paintengine2d.Rect
		dir  Direction
		part ScrollPart
	}
	for i, a := range [2]end{{p.Dec, dec, ScrollDec}, {p.Inc, inc, ScrollInc}} {
		if a.r.Empty() {
			continue
		}
		a0, n := pos(a.r)
		s, cn := 1, a0+n-1
		if i == 0 {
			lo = a0 + n
		} else {
			s, cn = a0, along-1-a0
			hi = a0
		}
		if cn < 5 || across < 7 {
			continue
		}
		box := sub(s, 1, cn, across-2)
		pressed := active && st.Pressed == a.part
		switch {
		case !active:
			fOff.cells(box, 1, 1, box.w-2, box.h-2)
			rpEdge(&kWhite, &kL1, box, 0, 0, box.w, box.h, 1)
		case pressed:
			fDown.cells(box, 1, 1, box.w-2, box.h-2)
			rpEdge(&kD2, &kWhite, box, 0, 0, box.w, box.h, 1)
		default:
			fPanel.cells(box, 1, 1, box.w-2, box.h-2)
			rpEdge(&kWhite, &kLo, box, 0, 0, box.w, box.h, 1)
		}
		tw := min(9, across-6, cn-4)
		if tw < 3 {
			continue
		}
		if tw%2 == 0 {
			tw--
		}
		m := beTri(tw, max(tw-1, 2)).turn(a.dir)
		ox, oy := (box.w-m.w)/2, (box.h-m.h)/2
		if !active {
			o := m.outline()
			o.emit(&kD1, box, ox, oy)
			continue
		}
		m.emit(&kWhite, box, ox+1, oy+1)
		m.emit(&kD4, box, ox, oy)
	}
	// The knob's span along the track (empty when there is none).
	t0, t1 := hi, hi
	if active {
		a0, n := pos(p.Thumb)
		t0, t1 = max(a0, lo), min(a0+n, hi)
		if t1-t0 < 4 || across < 6 {
			t0, t1 = hi, hi
		}
	}
	if hi > lo {
		if active {
			for _, seg := range [2][2]int{{lo, t0}, {t1, hi}} {
				if n := seg[1] - seg[0]; n > 0 {
					fTrack.add(sub(seg[0], 3, n, across-4).rect())
					kD1.add(sub(seg[0], 1, n, 2).rect())
				}
			}
		} else {
			fOff.add(sub(lo, 1, hi-lo, across-2).rect())
		}
	}
	if tn := t1 - t0; tn >= 4 {
		w := across - 2
		face := &fPanel
		if st.Pressed == ScrollThumbPart {
			face = &fTrack
		}
		face.add(sub(t0+1, 2, tn-3, w-2).rect())
		kWhite.add(sub(t0, 1, 1, w-1).rect())
		kWhite.add(sub(t0+1, 1, tn-3, 1).rect())
		kD1.add(sub(t0, w, tn-1, 1).rect())
		kD1.add(sub(t0+tn-2, 1, 1, w-1).rect())
		kD4.add(sub(t0+tn-1, 1, 1, w).rect())
		if tn >= 26 && w >= 9 {
			// The grip: three raised 5 × 5 squares, two cells apart.
			x0 := 1 + (w-5)/2
			a := t0 + 1 + (tn-3-19)/2
			for i := 0; i < 3; i++ {
				rpEdge(&kWhite, &kD2, sub(a+7*i, x0, 5, 5), 0, 0, 5, 5, 1)
			}
		}
	}
	fTrack.fill(ctx, c.track)
	fPanel.fill(ctx, c.panel)
	fOff.fill(ctx, c.l2)
	fDown.fill(ctx, c.d1)
	kD1.fill(ctx, c.d1)
	kL1.fill(ctx, c.l1)
	kLo.fill(ctx, c.arrowLo)
	kWhite.fill(ctx, c.white)
	kD4.fill(ctx, c.d4)
	kD2.fill(ctx, c.d2)
}

// DrawScrollBar paints a bare track and knob (widgets that lay out their
// own bar).
func (e beosEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered(), Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------

func (beosEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is BBox: the fancy border, a groove of #808080 over white,
// with the bold label breaking its top edge; a raised box is the plain
// border, white above #808080.
func (e beosEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := beColorsOf(l)
	u := rpU(l)
	f := l.BoldFont()
	ctx.DrawRect(rpGridAt(ctx, b, u).rect(), paintengine2d.Fill(c.panel))
	top := b.Min.Y
	if title != "" {
		top += f.Height() * 0.5
	}
	g := rpGridAt(ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	var hi, lo rpInk
	if raised {
		rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	} else {
		hi.frame(g, 1, 1, g.w-1, g.h-1, 1)
		lo.frame(g, 0, 0, g.w-1, g.h-1, 1)
	}
	hi.fill(ctx, c.white)
	lo.fill(ctx, c.d3)
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(8)
	tw := f.Advance(title) + l.S(6)
	if tw > b.Dx()-l.S(16) {
		tw = b.Dx() - l.S(16)
	}
	if tw <= 0 {
		return
	}
	ctx.DrawRect(rpGridAt(ctx, paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), u).rect(), paintengine2d.Fill(c.panel))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// beFrame is the window border's thickness in cells.
const beFrame = 5

// beTabH is the window tab's height in cells: 20 for 12-point bold, grown
// to the toolkit's bold font. Its last row is the frame's top line.
func beTabH(l *Classic) int {
	h := int(l.BoldFont().Height()/rpU(l) + 0.5)
	if h < 20 {
		h = 20
	}
	return h
}

// beBoxY is the first row of the close and zoom boxes: four under the
// tab's top on a 20-row tab, centred as the tab grows.
func beBoxY(l *Classic) int { return 4 + (beTabH(l)-20)/2 }

// WindowFrameInsets: the tab stands above the frame, so the top inset holds
// it as well as the five-pixel border.
func (beosEngine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: float32(beTabH(l)-1+beFrame) * u, Right: beFrame * u, Bottom: beFrame * u, Left: beFrame * u}
}

// beWinOK reports whether a frame of g.w × g.h cells is big enough to draw,
// and whether it has room for the close box.
func beWinOK(l *Classic, g rpGrid) (frame, close bool) {
	frame = g.w >= 2*beFrame+8 && g.h >= beTabH(l)+2*beFrame+2
	return frame, frame && g.w >= 40
}

// WindowCloseRect is the close box, four pixels in from the tab's corner.
func (e beosEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	if _, ok := beWinOK(l, g); !ok {
		return paintengine2d.Rect{}
	}
	return g.at(4, beBoxY(l), 14, 14)
}

// PopupShadow: BeOS menus, tooltips and windows cast no shadow.
func (beosEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (beosEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {
}

// ItemFocus: a one-pixel navigation-blue frame round the current row. An
// invention: BListView marked only the focused view (its frame turns blue);
// later column list views framed the focus row like this.
func (e beosEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	e.DrawFocusRing(l, ctx, b)
}

// ViewFrameInsets: BScrollView's two-pixel fancy border and, outside it, a
// pixel for the blue frame of the focused view.
func (beosEngine) ViewFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: 3 * u, Right: 3 * u, Bottom: 3 * u, Left: 3 * u}
}

// DrawViewFrame: white inside the text-control frame; the view with keyboard
// focus is ringed in the navigation blue.
func (e beosEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 8 {
		return
	}
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	col := c.panel
	if st.Focused() && !st.Disabled() {
		col = c.nav
	}
	k.fill(ctx, col)
	c.well(ctx, g, 1, 1, g.w-2, g.h-2, c.field, st.Disabled(), false)
}

// StyleHint: BAlert's buttons line the bottom with the default rightmost,
// labels sit left of their text controls, tabs start at the left.
func (beosEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintMnemonics {
		return MnemonicsNever // BeOS menus show shortcuts, no underlines
	}
	return 0
}

// ControlFont: BeOS labels every control in the plain system font; only
// window titles and box labels are bold.
func (beosEngine) ControlFont(l *Classic, role Role) *Font { return l.body }

// DrawWindowBackground is the panel grey.
func (beosEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(beColorsOf(l).panel))
}

// TabOutset: the selected tab is drawn two pixels wider on each side, over
// its neighbours.
func (beosEngine) TabOutset(l *Classic) Insets {
	u := rpU(l)
	return Insets{Left: 2 * u, Right: 2 * u}
}

// DrawTabPane is BTabView's page: the panel grey, white along the top and
// left, #646464 along the bottom and right.
func (e beosEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	if g.w < 3 || g.h < 3 {
		return
	}
	var hi, lo rpInk
	rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	hi.fill(ctx, c.white)
	lo.fill(ctx, c.edge)
}

// closeBox paints the close box at cell (x, y) of g: 14 cells, two rings —
// dark on the top and left and light on the bottom and right, then the
// reverse — round a 10 × 10 face shaded diagonally from light to dark
// yellow. Pressed, both rings sink and the shading reverses; on an inactive
// tab it is grey and white.
func (c *beColors) closeBox(ctx *paintengine2d.Context, g rpGrid, x, y int, active, pressed bool) {
	bx := g.sub(x, y, 14, 14)
	dk, lt, stops := c.boxDk, c.boxLt, c.faceOn
	if !active {
		dk, lt, stops = c.boxOffDk, c.white, c.faceOff
	} else if pressed {
		stops = c.faceDown
	}
	face := bx.at(2, 2, 10, 10)
	ctx.DrawRect(face, DGradient(face, stops...))
	var kd, kl rpInk
	rpEdge(&kd, &kl, bx, 0, 0, 14, 14, 1)
	if pressed {
		rpEdge(&kd, &kl, bx, 1, 1, 12, 12, 1)
	} else {
		rpEdge(&kl, &kd, bx, 1, 1, 12, 12, 1)
	}
	kd.fill(ctx, dk)
	kl.fill(ctx, lt)
}

// zoomBox paints the zoom box at cell (x, y) of g: an 11-cell square at the
// bottom right and a 9-cell square at the top left in front of it, each a
// sunken ring, a raised ring and a shaded face.
func (c *beColors) zoomBox(ctx *paintengine2d.Context, g rpGrid, x, y int, active bool) {
	dk, lt, stops := c.zoomDk, c.boxLt, c.faceOn
	if !active {
		dk, lt, stops = c.zoomOffDk, c.white, c.faceOff
	}
	for _, sq := range [2][3]int{{x + 3, y + 3, 11}, {x, y, 9}} {
		n := sq[2]
		s := g.sub(sq[0], sq[1], n, n)
		face := s.at(2, 2, n-4, n-4)
		ctx.DrawRect(face, DGradient(face, stops...))
		var kd, kl rpInk
		rpEdge(&kd, &kl, s, 0, 0, n, n, 1)
		rpEdge(&kl, &kd, s, 1, 1, n-2, n-2, 1)
		kd.fill(ctx, dk)
		kl.fill(ctx, lt)
	}
}

// DrawWindowFrame is a BeOS window. The yellow tab — grey when the window is
// inactive — is only as wide as its contents and stands on the frame's
// top-left corner, its last row the frame's top line: #989898 and #FFFF50 on
// its top and left, #AF7B00 and #606060 on its right; the close box four
// pixels in, the bold title eighteen pixels after it, the zoom box two
// pixels in from the right edge. Beside the tab the window colour stands in
// for whatever is behind. The frame is five pixels: #989898, white, the
// panel, #888888 and #989898 from the outside on the top and left, #606060,
// #888888, the panel, white and #989898 on the bottom and right.
func (e beosEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	frameOK, room := beWinOK(l, g)
	if !frameOK {
		return
	}
	closeOn := room && st.CanClose
	th := beTabH(l)
	fy, fh := th-1, g.h-th+1
	// The tab's layout in cells: close box, title, zoom box.
	f := l.BoldFont()
	tx := 10
	if closeOn {
		tx = 4 + 14 + 19
	}
	zoomOn := st.Maximizable
	right := 12
	if zoomOn {
		right = 17 + 14 + 4
		if tx+right > g.w {
			zoomOn, right = false, 12
		}
	}
	tw := 0
	if title != "" {
		tw = min(int(f.Advance(title)/u+0.99), g.w-tx-right)
		if tw < 8 {
			tw = 0
		}
	}
	tabW := min(tx+tw+right, g.w)
	var kTab, kHi, kLo, kD2, kD4, kWhite, kMid rpInk
	kTab.cells(g, 2, 2, tabW-4, th-3)
	kD2.cells(g, 0, 0, tabW-1, 1)
	kD2.cells(g, 0, 1, 1, th-2)
	kD4.cells(g, tabW-1, 0, 1, th-1)
	kHi.cells(g, 1, 1, tabW-3, 1)
	kHi.cells(g, 1, 2, 1, th-3)
	kLo.cells(g, tabW-2, 1, 1, th-2)
	tl := [beFrame]*rpInk{&kD2, &kWhite, nil, &kMid, &kD2}
	br := [beFrame]*rpInk{&kD4, &kMid, nil, &kWhite, &kD2}
	for i := 0; i < beFrame; i++ {
		rpEdge(tl[i], br[i], g, i, fy+i, g.w-2*i, fh-2*i, 1)
	}
	fill, hi, lo, tc := c.tab, c.tabHi, c.tabLo, c.text
	if !st.Active {
		fill, hi, lo, tc = c.tabOff, c.white, c.d2, c.tabOffText
	}
	kTab.fill(ctx, fill)
	kHi.fill(ctx, hi)
	kLo.fill(ctx, lo)
	kWhite.fill(ctx, c.white)
	kMid.fill(ctx, c.mid)
	kD2.fill(ctx, c.d2)
	kD4.fill(ctx, c.d4)
	by := beBoxY(l)
	if closeOn {
		c.closeBox(ctx, g, 4, by, st.Active, st.ClosePress)
	}
	if zoomOn {
		c.zoomBox(ctx, g, tabW-4-14, by, st.Active)
	}
	if tw > 0 {
		l.drawFittedText(ctx, f, title, g.at(tx, 1, tw, th-1), tc, AlignStart, 0)
	}
}

// ---- controls -----------------------------------------------------------------

// DrawPanel: a raised panel is white along its top and left and #989898
// along its bottom and right; a flat one is the panel grey.
func (e beosEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	if !raised || g.w < 3 || g.h < 3 {
		return
	}
	var hi, lo rpInk
	rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	hi.fill(ctx, c.white)
	lo.fill(ctx, c.d2)
}

// beBtnMargin is the room every push button keeps round its body for the
// default button's outline: four cells (#606060, #B8B8B8 and two of the
// face), three in a slot under 32 cells (the outline gives up a row of the
// face), none when the button is too small.
func beBtnMargin(g rpGrid) int {
	m := 4
	if g.h < 32 {
		m = 3
	}
	if g.h-2*m < 14 || g.w-2*m < 20 {
		return 0
	}
	return m
}

// DrawButton is BButton. The default button's outline sits in the margin
// every button keeps, so a row of buttons lines up whichever is default.
func (e beosEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 8 {
		return
	}
	m := beBtnMargin(g)
	if st.Primary() && m > 0 {
		ctx.DrawRect(g.at(2, 2, g.w-4, g.h-4), paintengine2d.Fill(c.l1))
		var kOut, kIn rpInk
		beCut(&kOut, g, 0, 0, g.w, g.h)
		kIn.frame(g, 1, 1, g.w-2, g.h-2, 1)
		oc, ic := c.d4, c.d1
		if st.Disabled() {
			oc, ic = c.d1, c.l1
		}
		kOut.fill(ctx, oc)
		kIn.fill(ctx, ic)
	}
	body := g.inset(m)
	fg := c.button(ctx, body, st)
	c.label(l, ctx, body.inset(1), l.body, label, body.at(3, 0, body.w-6, body.h), fg, AlignCenter, st.Focused() && !st.Disabled())
}

// toggleLabel draws a check box, radio or switch caption six pixels after
// its indicator, underlined in blue while focused (the indicator is framed
// in blue when there is no caption).
func (e beosEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := beColorsOf(l)
	u := rpU(l)
	focus := st.Focused() && !st.Disabled()
	if label == "" {
		if focus {
			e.DrawFocusRing(l, ctx, box.Inset(-2*u).Intersect(b))
		}
		return
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	col := c.text
	if st.Disabled() {
		col = c.dim
	}
	c.label(l, ctx, rpGridAt(ctx, b, u), l.body, label, lb, col, AlignStart, focus)
}

func (e beosEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e beosEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch: BeOS had none. This one is built from its parts: a
// text-control well, white when off and the status blue when on, with the
// slider's block knob sliding in it.
func (e beosEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := beColorsOf(l)
	tw, th := min(l.metrics.SwitchW, b.Dx()), min(l.metrics.SwitchH, b.Dy())
	g := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th), rpU(l))
	if g.w >= 14 && g.h >= 8 {
		fill := c.field
		switch {
		case st.Disabled() && on:
			fill = c.statusOff
		case st.Disabled():
			fill = c.l2
		case on:
			fill = c.status
		}
		c.well(ctx, g, 0, 0, g.w, g.h, fill, st.Disabled(), false)
		kw := g.w / 2
		kx := 0
		if on {
			kx = g.w - kw
		}
		c.knob(ctx, g.sub(kx, 0, kw, g.h), st)
	}
	e.toggleLabel(l, ctx, b, g.rect(), st, label)
}

// DrawComboBox is BMenuField's pop-up: the raised, shadowed box with the
// current item and a 5 × 3 triangle at its right.
func (e beosEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 20 || g.h < 10 {
		return
	}
	fg := c.popup(ctx, g, st, open)
	m := beTri(5, 3).turn(DirDown)
	tx := g.w - 3 - 5 - m.w
	var k rpInk
	m.emit(&k, g, tx, (g.h-2-m.h)/2)
	col := c.d4
	if st.Disabled() {
		col = c.d1
	}
	k.fill(ctx, col)
	lb := g.at(6, 0, tx-10, g.h-2)
	c.label(l, ctx, g.sub(2, 2, g.w-5, g.h-5), l.body, text, lb, fg, AlignStart, st.Focused() && !open && !st.Disabled())
}

// fieldOpts: a grey selection with black text (paler when the field lacks
// focus) and a one-pixel black insertion point.
func (c *beColors) fieldOpts() rpFieldOpts {
	return rpFieldOpts{
		text: c.fieldText, muted: c.dim, sel: c.rowSel, selTxt: c.rowText,
		selOff: c.panel, caret: c.text, caretKind: rpCaretBar, bg: c.field,
	}
}

// DrawTextField is BTextControl's text box: the two-ring frame, white
// inside, its inner ring blue while it has focus.
func (e beosEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 6 {
		return
	}
	e.Face(l, ctx, b, RoleField, st)
	inner := g.at(2, 2, g.w-4, g.h-4)
	inner.Min.X += l.S(3)
	inner.Max.X -= l.S(3)
	rpFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.fieldOpts())
}

// DrawTextArea is a BTextView in the same frame.
func (e beosEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 8 {
		return
	}
	e.Face(l, ctx, b, RoleField, st)
	inner := g.at(2, 2, g.w-4, g.h-4).Inset(l.S(3))
	rpAreaText(l, ctx, inner, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, c.fieldOpts())
}

// DrawSpinner: BeOS had no spinner; two of the scroll bar's arrow boxes
// stacked in its outline stand in.
func (e beosEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 7 || g.h < 12 {
		return
	}
	w := min(g.w, 15)
	g = g.sub((g.w-w)/2, 0, w, g.h)
	mid := g.h / 2
	var fPanel, fDown, fOff, kWhite, kLo, kL1, kD1, kD4, kD2 rpInk
	kD2.frame(g, 0, 0, g.w, g.h, 1)
	kD2.cells(g, 1, mid, g.w-2, 1)
	for _, half := range [2]struct {
		y, n    int
		dir     Direction
		pressed bool
	}{{1, mid - 1, DirUp, upPress}, {mid + 1, g.h - mid - 2, DirDown, downPress}} {
		cg := g.sub(1, half.y, g.w-2, half.n)
		if cg.w < 3 || cg.h < 3 {
			continue
		}
		switch {
		case st.Disabled():
			fOff.cells(cg, 1, 1, cg.w-2, cg.h-2)
			rpEdge(&kWhite, &kL1, cg, 0, 0, cg.w, cg.h, 1)
		case half.pressed:
			fDown.cells(cg, 1, 1, cg.w-2, cg.h-2)
			rpEdge(&kD2, &kWhite, cg, 0, 0, cg.w, cg.h, 1)
		default:
			fPanel.cells(cg, 1, 1, cg.w-2, cg.h-2)
			rpEdge(&kWhite, &kLo, cg, 0, 0, cg.w, cg.h, 1)
		}
		tw := min(7, cg.w-4, 2*cg.h-5)
		if tw < 3 {
			continue
		}
		if tw%2 == 0 {
			tw--
		}
		m := beTri(tw, tw/2+1).turn(half.dir)
		ox, oy := (cg.w-m.w)/2, (cg.h-m.h)/2
		if st.Disabled() {
			o := m.outline()
			o.emit(&kD1, cg, ox, oy)
			continue
		}
		m.emit(&kWhite, cg, ox+1, oy+1)
		m.emit(&kD4, cg, ox, oy)
	}
	fPanel.fill(ctx, c.panel)
	fDown.fill(ctx, c.d1)
	fOff.fill(ctx, c.l2)
	kL1.fill(ctx, c.l1)
	kLo.fill(ctx, c.arrowLo)
	kD1.fill(ctx, c.d1)
	kWhite.fill(ctx, c.white)
	kD4.fill(ctx, c.d4)
	kD2.fill(ctx, c.d2)
}

// DrawTabBar: the panel grey, and along its bottom the page's white top
// edge that the tabs stand on.
func (e beosEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	if g.h >= 2 {
		ctx.DrawRect(g.at(0, g.h-1, g.w, 1), paintengine2d.Fill(c.white))
	}
}

// beTabEdge is the first cell of row y of a tab h rows tall: its sides
// flare out by two cells from the top to the base, and its top corners are
// chamfered two cells.
func beTabEdge(y, h int) int {
	e := 0
	if h > 1 {
		e = (2*(h-1-y) + (h-1)/2) / (h - 1)
	}
	if y < 2 {
		e += 2 - y
	}
	return e
}

// DrawTab is a BTabView tab: the page colour, white along the top and the
// left side, #646464 down the right side. Unselected tabs stand on the
// page's top line; the selected one, drawn last and wider, opens into it.
func (e beosEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 12 || g.h < 8 {
		return
	}
	w, h := g.w, g.h
	last := h - 2 // the last row of the tab's body
	if selected {
		last = h - 1
	}
	var kFace, kWhite, kDark rpInk
	face := rpRows{k: &kFace, g: g}
	left := rpRows{k: &kWhite, g: g}
	rightSide := rpRows{k: &kDark, g: g}
	for y := 0; y <= last; y++ {
		ed := beTabEdge(y, h)
		if y == 0 {
			kWhite.cells(g, ed, 0, w-2*ed, 1)
			continue
		}
		in := max(ed+1, beTabEdge(y-1, h))
		if 2*in >= w {
			continue
		}
		left.row(y, ed, in, 0, 0)
		rightSide.row(y, w-in, w-ed, 0, 0)
		face.row(y, in, w-in, 0, 0)
	}
	face.flush(last + 1)
	left.flush(last + 1)
	rightSide.flush(last + 1)
	if !selected {
		kWhite.cells(g, 0, h-1, w, 1)
	}
	fc := c.panel
	if st.Pressed() && !st.Disabled() {
		fc = c.track // an invention: BeOS switched pages on the press
	}
	kFace.fill(ctx, fc)
	kWhite.fill(ctx, c.white)
	kDark.fill(ctx, c.edge)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	lb := g.at(4, 1, w-8, h-2)
	c.label(l, ctx, g.sub(3, 1, w-6, h-2), l.body, label, lb, fg, AlignCenter, selected && st.Focused() && !st.Disabled())
}

// DrawMenuBar is BMenuBar: the panel grey, #EFEFEF along the top and left,
// #989898 along the bottom and right.
func (e beosEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	e.Face(l, ctx, b, RoleBar, StateNone)
}

// DrawMenuTitle: an open title takes the menu selection between the bar's
// edges; a hovered one raises a one-pixel bevel (an invention: BeOS tracked
// menus only while the button was down).
func (e beosEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.dim
	case open || st.Pressed():
		e.MenuHighlight(l, ctx, g.at(0, 1, g.w, g.h-2), true)
	case st.Hovered():
		var hi, lo rpInk
		rpEdge(&hi, &lo, g, 0, 1, g.w, g.h-2, 1)
		hi.fill(ctx, c.white)
		lo.fill(ctx, c.d2)
	}
	c.label(l, ctx, g.sub(0, 1, g.w, g.h-2), l.body, label, b, fg, AlignCenter, st.Focused() && !open && !st.Disabled())
}

// DrawMenuFrame is a BMenu window: the panel grey in a #646464 outline, with
// #EFEFEF inside its top and left and #989898 inside its bottom and right.
func (e beosEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	if g.w < 5 || g.h < 5 {
		return
	}
	var kOut, hi, lo rpInk
	kOut.frame(g, 0, 0, g.w, g.h, 1)
	rpEdge(&hi, &lo, g, 1, 1, g.w-2, g.h-2, 1)
	hi.fill(ctx, c.menuHi)
	lo.fill(ctx, c.d2)
	kOut.fill(ctx, c.edge)
}

// subArrow is the submenu mark: a triangle pointing right, 8 × 9, outlined
// in #808080 with a white highlight inside its upper left and #E7E7E7 inside.
func (c *beColors) subArrow(ctx *paintengine2d.Context, g rpGrid, disabled bool) {
	h := min(9, g.h-2)
	if h%2 == 0 {
		h--
	}
	if h < 5 || g.w < h {
		return
	}
	m := beTri(h, h-1).turn(DirRight)
	o := m.outline()
	in := m.minus(o)
	hl := rpNewMask(m.w, m.h)
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if in.get(x, y) && (o.get(x-1, y) || o.get(x, y-1)) {
				hl.set(x, y)
			}
		}
	}
	rest := in.minus(hl)
	oy := (g.h - m.h) / 2
	var kOut, kHi, kIn rpInk
	o.emit(&kOut, g, 0, oy)
	hl.emit(&kHi, g, 0, oy)
	rest.emit(&kIn, g, 0, oy)
	line := c.d3
	if disabled {
		line = c.d1
	}
	kIn.fill(ctx, c.subFill)
	kHi.fill(ctx, c.white)
	kOut.fill(ctx, line)
}

// DrawMenuItem: the item under the pointer takes the menu selection across
// the menu, its text stays black; a check mark at the left, the submenu
// triangle at the right; separators are #B8B8B8 over #EFEFEF, three pixels
// in from the frame.
func (e beosEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := beColorsOf(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+2*u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-4*u, b.Dy()), u)
	if row.Separator {
		if full.w > 8 && full.h >= 2 {
			y := full.h/2 - 1
			ctx.DrawRect(full.at(3, y, full.w-6, 1), paintengine2d.Fill(c.d1))
			ctx.DrawRect(full.at(3, y+1, full.w-6, 1), paintengine2d.Fill(c.menuHi))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, full.rect(), false)
	}
	col := e.MenuTextColor(l, hot)
	if st.Disabled() {
		col = c.dim
	}
	f := l.body
	mark := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked:
		// Radio-mode menus mark their choice with the same check.
		if n := min(9, mark.w-2, mark.h-4); n >= 6 {
			m := beTick(n)
			var k rpInk
			m.emit(&k, mark, (mark.w-m.w)/2, (mark.h-m.h)/2)
			k.fill(ctx, col)
		}
	case row.Icon != IconNone:
		side := min(ch.CheckCol()-l.S(2), b.Dy()-l.S(4))
		ib := paintengine2d.XYWH(b.Min.X+(ch.CheckCol()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		l.drawToolIcon(ctx, ib, row.Icon, col)
	}
	labelRight := b.Max.X
	if row.Submenu {
		ax := ch.ArrowMinX(b.Max.X)
		c.subArrow(ctx, rpGridAt(ctx, paintengine2d.XYWH(ax, b.Min.Y, b.Max.X-ax, b.Dy()), u), st.Disabled())
		labelRight = ax
	}
	if row.Shortcut != "" {
		tw := f.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		l.drawFittedText(ctx, f, row.Shortcut, paintengine2d.XYWH(sx, b.Min.Y, tw+l.S(2), b.Dy()), col, AlignStart, 0)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	l.drawFittedText(ctx, f, row.Label, paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()), col, AlignStart, 0)
}

// DrawProgressBar is BStatusBar's bar: #3296FF filling from the left in the
// text-control frame, white where it has not reached. The busy bar (an
// invention: BStatusBar had none) is a block of the bar colour that slides.
func (e beosEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 6 {
		return
	}
	empty, bar := c.white, c.status
	if st.Disabled() {
		empty, bar = c.l2, c.statusOff
	}
	c.well(ctx, g, 0, 0, g.w, g.h, empty, st.Disabled(), false)
	in := g.inset(2)
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		seg := max(in.w/4, 4)
		x := int(float32(in.w+seg)*phase) - seg
		if a, z := max(x, 0), min(x+seg, in.w); z > a {
			ctx.DrawRect(in.at(a, 0, z-a, in.h), paintengine2d.Fill(bar))
		}
		return
	}
	if n := int(float32(in.w)*clamp1(t) + 0.5); n > 0 {
		ctx.DrawRect(in.at(0, 0, n, in.h), paintengine2d.Fill(bar))
	}
}

// DrawSlider is BSlider with its block thumb: a seven-pixel groove (#909090
// and black above, white below, the #505050 bar between) and the knob,
// underlined in blue while the slider has focus.
func (e beosEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 24 || g.h < 9 {
		return
	}
	kw := min(max(int(l.metrics.Thumb/u+0.5), 9), g.w/2)
	kh := min(13, g.h)
	gh := 7
	gy := (g.h - gh) / 2
	ky := (g.h - kh) / 2
	x0, x1 := 2, g.w-2
	bar, line := c.slBar, c.slLine
	if st.Disabled() {
		bar, line = c.d1, c.d1
	}
	ctx.DrawRect(g.at(x0+2, gy+2, x1-x0-3, gh-3), paintengine2d.Fill(bar))
	var kLine, kWhite, kBlack rpInk
	rpEdge(&kLine, &kWhite, g, x0, gy, x1-x0, gh, 1)
	rpEdge(&kBlack, nil, g, x0+1, gy+1, x1-x0-2, gh-2, 1)
	kLine.fill(ctx, line)
	kWhite.fill(ctx, c.white)
	blk := c.black
	if st.Disabled() {
		blk = c.d2
	}
	kBlack.fill(ctx, blk)
	kx := int(float32(g.w-kw)*clamp1(t) + 0.5)
	c.knob(ctx, g.sub(kx, ky, kw, kh), st)
	if !st.Focused() || st.Disabled() {
		return
	}
	if row := ky + kh; row+2 <= g.h {
		ctx.DrawRect(g.at(kx, row, kw, 1), paintengine2d.Fill(c.nav))
		ctx.DrawRect(g.at(kx, row+1, kw, 1), paintengine2d.Fill(c.white))
		return
	}
	var k rpInk
	k.frame(g, kx+1, ky+1, kw-3, kh-3, 1)
	k.fill(ctx, c.nav)
}

// DrawListRow is a BStringItem: #B0B0B0 across the row when selected,
// black text either way.
func (e beosEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := e.Face(l, ctx, b, RoleRow, st)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a BOutlineListView row: the latch triangle, then the item.
func (e beosEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := beColorsOf(l)
	fg := e.Face(l, ctx, b, RoleRow, st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()).Intersect(b), expanded, c.black)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	if lx := x + indent + l.S(3); lx < b.Max.X-l.S(2) {
		l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(2), b.Dy()), fg, AlignStart, 0)
	}
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableCell: #B0B0B0 across a selected row, black text.
func (e beosEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := beColorsOf(l)
	fg := c.fieldText
	if st.Checked() {
		ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(c.rowSel))
		fg = c.rowText
	}
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, align, 0)
}

// DrawTableHeader: after Tracker's column titles — raised panel-grey cells
// (white top and left, #989898 bottom and right) with a small triangle on
// the sort column; pressed cells sink.
func (e beosEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	face, tl, br := c.panel, c.white, c.d2
	if pressed {
		face, tl, br = c.d1, c.d2, c.white
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
	var hi, lo rpInk
	rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	hi.fill(ctx, tl)
	lo.fill(ctx, br)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, c.d4)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar: BeOS had no standard tool bar; a strip with the menu bar's
// edges stands in.
func (e beosEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	e.Face(l, ctx, b, RoleBar, StateNone)
}

// DrawToolButton: flat on the bar, a BButton under the pointer, sunken while
// pressed or latched; focus is the blue frame.
func (e beosEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := beColorsOf(l)
	fg := e.Face(l, ctx, b, RoleTool, st)
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" && x < b.Max.X-pad {
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-pad-x, b.Dy()), fg, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		g := rpGridAt(ctx, b, rpU(l))
		var k rpInk
		k.frame(g, 2, 2, g.w-4, g.h-4, 1)
		k.fill(ctx, c.nav)
	}
}

// DrawStatusBar is a window's bottom strip: a #989898 line over a white
// one, parts divided by grooves, and at the right end the resize knob of a
// document window — ten dots in a triangle, each #888888 with a white pixel
// below and right of it, three pixels apart.
func (e beosEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 6 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	var kD2, kWhite, kDot rpInk
	kD2.cells(g, 0, 0, g.w, 1)
	kWhite.cells(g, 0, 1, g.w, 1)
	k := min(g.h-2, 14)
	kx := g.w - k
	if len(parts) > 0 {
		slot := float32(kx) / float32(len(parts))
		for i, s := range parts {
			x := int(slot * float32(i))
			if i > 0 && g.h > 8 {
				kD2.cells(g, x, 4, 1, g.h-7)
				kWhite.cells(g, x+1, 4, 1, g.h-7)
			}
			l.drawFittedText(ctx, l.body, s, g.at(x+6, 2, int(slot)-10, g.h-2), c.text, AlignStart, 0)
		}
	}
	if k >= 13 {
		kg := g.sub(kx, g.h-k, k, k)
		kWhite.cells(kg, 0, 0, k, 1)
		kWhite.cells(kg, 0, 1, 1, k-1)
		for r := 1; r <= 4; r++ {
			y := k - 3 - 3*(4-r)
			for j := 0; j < r; j++ {
				x := k - 3 - 3*j
				kDot.cells(kg, x, y, 1, 1)
				kWhite.cells(kg, x+1, y+1, 1, 1)
			}
		}
	}
	kD2.fill(ctx, c.d2)
	kWhite.fill(ctx, c.white)
	kDot.fill(ctx, c.mid)
}

// DrawTitleBar is a panel heading: the bold title over a groove.
func (e beosEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	if g.h >= 4 {
		ctx.DrawRect(g.at(0, g.h-2, g.w, 1), paintengine2d.Fill(c.d2))
		ctx.DrawRect(g.at(0, g.h-1, g.w, 1), paintengine2d.Fill(c.white))
	}
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.muted, AlignStart, 0)
	}
}

// DrawAccordionHeader: BeOS had none; a raised panel-grey strip with the
// outline list's triangle and a bold title stands in, sunk while pressed.
func (e beosEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 6 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	face, tl, br := c.panel, c.white, c.d2
	if pressed {
		face, tl, br = c.d1, c.d2, c.white
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
	var hi, lo rpInk
	rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	hi.fill(ctx, tl)
	lo.fill(ctx, br)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, c.black)
	c.label(l, ctx, g.inset(1), l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()), fg, AlignStart, st.Focused() && !st.Disabled())
}

// DrawSeparator is a groove: #989898 beside white.
func (e beosEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if vertical {
		if g.w < 2 || g.h < 5 {
			return
		}
		x := g.w/2 - 1
		ctx.DrawRect(g.at(x, 2, 1, g.h-4), paintengine2d.Fill(c.d2))
		ctx.DrawRect(g.at(x+1, 2, 1, g.h-4), paintengine2d.Fill(c.white))
		return
	}
	if g.w < 2 || g.h < 2 {
		return
	}
	y := g.h/2 - 1
	ctx.DrawRect(g.at(0, y, g.w, 1), paintengine2d.Fill(c.d2))
	ctx.DrawRect(g.at(0, y+1, g.w, 1), paintengine2d.Fill(c.white))
}

// DrawSplitter: BeOS had no split view; a groove on the panel grey, raised
// into a bar under the pointer and sunk while dragged.
func (e beosEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 2 || g.h < 2 {
		return
	}
	if st.Hovered() || st.Pressed() {
		face, tl, br := c.panel, c.white, c.d2
		if st.Pressed() {
			face, tl, br = c.d1, c.d2, c.white
		}
		ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
		var hi, lo rpInk
		rpEdge(&hi, &lo, g, 0, 0, g.w, g.h, 1)
		hi.fill(ctx, tl)
		lo.fill(ctx, br)
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.panel))
	e.DrawSeparator(l, ctx, b, vertical)
}

// DrawTooltip: R5 had no tooltips; this is Haiku's later pale yellow
// (#FFFFD8) in a one-pixel #606060 frame.
func (e beosEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := beColorsOf(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.info))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.d4)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.text, AlignStart, 0)
}

// DrawOverlay: BeOS did not dim the screen behind a modal window.
func (beosEngine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- packs --------------------------------------------------------------------

func beosPacks() []ThemePack {
	panel := hexColor("#d8d8d8")
	black, white := hexColor("#000000"), hexColor("#ffffff")
	nav := hexColor("#0000e5")
	pal := Palette{
		Background: panel, Surface: panel, SurfaceAlt: panel,
		Border: hexColor("#606060"), Divider: hexColor("#989898"),
		Text: black, TextMuted: hexColor("#505050"), TextOnAccent: white,
		Accent: nav, AccentHover: nav, AccentPress: nav,
		Field: white, FieldBorder: hexColor("#606060"),
		Focus: nav, Selection: hexColor("#b0b0b0"),
		Track: hexColor("#c8c8c8"), Thumb: panel,
		Highlight: white, Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: hexColor("#999999"), MenuHoverBorder: hexColor("#999999"), MenuGutter: panel,
		Danger: hexColor("#c00000"), Success: hexColor("#007000"), Warning: hexColor("#9a5c00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: white, BevelDark: hexColor("#989898"),
	}
	extra := map[string]paintengine2d.Color{}
	for k, v := range map[string]string{
		"tab": "#ffcb00", "tabHi": "#ffff50", "tabLo": "#af7b00", "tabOff": "#e8e8e8", "tabOffText": "#505050",
		"boxDark": "#b78200", "boxLight": "#ffff3f", "zoomDark": "#d29d00",
		"nav": "#0000e5", "status": "#3296ff", "menuSel": "#999999", "info": "#ffffd8",
	} {
		extra[k] = hexColor(v)
	}
	tok := ThemeTokens{
		Engine:   "beos",
		Bevel:    BevelClassic3D,
		Family:   ThemeLight,
		Palette:  pal,
		Extra:    extra,
		Hot:      ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover},
		Selected: ChromeState{Fill: pal.Selection, Border: pal.Selection},
		Focus:    ChromeState{Fill: nav.WithAlpha(0.15), Border: nav},
	}
	return []ThemePack{{
		Name: "beos", Label: "BeOS", Year: 1998, Lineage: "Be",
		Summary: "BeOS R5: the yellow window tab, soft grey bevels on #D8D8D8 and blue keyboard-focus underlines.",
		Era:     "BeOS", Palette: ThemeLight, Tokens: tok,
	}}
}
