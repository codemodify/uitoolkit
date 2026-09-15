package style

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// amigaEngine paints the Commodore Amiga's Workbench: 1.3 (1987), the blue
// and orange four-colour screen, and 2.0 to 3.1 (1990–1994), the grey 3D
// look Intuition called NewLook.
//
// Everything is whole pixels (engine_system7_pixel.go) on the line-doubled
// 640 × 512 screen the Workbench is usually seen on: one hires pixel is one
// cell across and two cells down, so Topaz 8 is 16 pixels tall — the
// toolkit's UI font size — and a one-pixel horizontal edge is as thick as
// the two-pixel vertical edges Intuition drew beside it. There are no
// gradients and no anti-aliased curves; each colour of a control fills in
// one path. Text is set in the look's monospace face, standing in for
// Topaz.
//
//   - Workbench 1.3: pen 0 blue screens and windows, white borders and
//     title bars with blue text, the close gadget at the left, the two
//     depth gadgets at the right and the drag bar's blue stripes, dotted
//     while the window is inactive. Gadgets are thin white outlines;
//     highlighting complements the pens — blue becomes orange and white
//     black — so a pressed button, a selected row and the string cursor
//     turn orange. Proportional gadgets have no arrows. Menus are white
//     with blue items; the hot item is the complement, a dark bar with
//     orange text.
//   - Workbench 2.0 / 3.1: grey, with Intuition's bevel — shine along the
//     top and left, shadow along the bottom and right, one-pixel
//     horizontal and two-pixel vertical edges meeting in 45° steps at the
//     top-right and bottom-left corners. Pressed gadgets recess and fill
//     with FILLPEN blue. String gadgets sit in the ridge frame, the cycle
//     gadget shows its looping arrow and a divider, check boxes are 26 × 11
//     raised boxes with a tick, and mutual-exclude buttons are raised
//     ovals. Window borders are FILLPEN blue while active; the title bar
//     holds the close gadget at the left and the zoom and depth gadgets at
//     the right. Menus are white with black text; the hot item is a black
//     bar with white text.
//   - Disabled gadgets are ghosted with Intuition's grid of dots (every
//     fourth pixel, the next row shifted by two) in pen 1; disabled menu
//     items are ghosted in the menu colour, which erases part of the text.
//   - The Amiga had no keyboard focus ring. This engine marks focus with a
//     dotted rectangle inside the gadget — orange on 1.3, black on 3.1, in
//     the complement on highlighted fills — and uses it everywhere.
//
// Pack data:
//
//	params: wb    13 = Workbench 1.3, 31 = Workbench 2.0 / 3.1
//	extra:  pen0 … pen3   the four Workbench colours (Palette prefs):
//	        1.3 blue, white, black, orange; 3.1 grey, black, white, blue
type amigaEngine struct{ BaseEngine }

func init() {
	RegisterEngine(amigaEngine{})
	for _, p := range amigaPacks() {
		RegisterPack(p)
	}
}

func (amigaEngine) ID() string { return "amiga" }

// DefaultMetrics map one hires pixel to one cell across and two down (the
// line-doubled Workbench): GadTools' 12-pixel buttons and 14-pixel string
// gadgets are 28 tall, the 26 × 11 check box is 26 × 22, the 17 × 9
// mutual-exclude button 17 × 18 and a title bar's eleven rows 22.
func (amigaEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Square:    true, BevelDepth: 2,
		ControlH: 28, FieldH: 28, ComboH: 28,
		Checkbox: 22, Radio: 18,
		MenuItemH: 22, MenuBarH: 22, TabH: 28, RowH: 22,
		TitleBar: 22, HeaderH: 24, ProgressH: 20, SliderH: 24, Thumb: 16,
		Scroll: 18, Pad: 8, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24, SpinnerW: 18, SwitchW: 40, SwitchH: 20,
	}
}

// amRow is the height of one hires pixel in cells: the Workbench is shown
// line-doubled, every Amiga row two cells tall and one cell wide.
const amRow = 2

// am13Side is how many cells wide the vertical lines of 1.3's outlines
// are: two hires pixels, as Intuition drew its 1.3 window borders.
const am13Side = 2

// amGhost is Intuition's ghosting grid: every fourth pixel of a row, the
// next row's dots shifted by two, each Amiga row two cells tall.
var amGhost = rpPat{0x88, 0x88, 0x22, 0x22, 0x88, 0x88, 0x22, 0x22}

// ---- colours ------------------------------------------------------------------

// am is the resolved colour set of a look.
type am struct {
	wb13 bool                   // Workbench 1.3 (else 2.0 / 3.1)
	pen  [4]paintengine2d.Color // the four Workbench colours

	win, text          paintengine2d.Color // window background and its text
	shine, shadow      paintengine2d.Color // the 3D edges
	line               paintengine2d.Color // 1.3 outlines
	fill, fillText     paintengine2d.Color // pressed and selected face, its text
	cur, curText       paintengine2d.Color // the string cursor, its character
	mark, markText     paintengine2d.Color // selected text in a string gadget
	hiText             paintengine2d.Color // headings
	bar, barText       paintengine2d.Color // screen title bar and menus
	menuHi, menuHiText paintengine2d.Color // the hot menu item
	menuLine           paintengine2d.Color // the menu box outline
	ghost              paintengine2d.Color // the ghosting dots on gadgets
	focus              paintengine2d.Color // the focus mark on the window
	muted              paintengine2d.Color // placeholders
	knob               paintengine2d.Color // proportional gadget knob
	dark               paintengine2d.Color // 1.3's pen 2: the depth gadgets' black
}

// The default Workbench colours: 1.3's blue, white, black and orange and
// 2.0's grey, black, white and blue.
var (
	amPens13 = [4]string{"#0055aa", "#ffffff", "#000022", "#ff8800"}
	amPens31 = [4]string{"#aaaaaa", "#000000", "#ffffff", "#6688bb"}
)

type amKey struct{}

func amColors(l *Classic) *am {
	return l.Memo(amKey{}, func() any { return amBuild(l) }).(*am)
}

func amBuild(l *Classic) *am {
	c := &am{wb13: l.P("wb", 31) < 20}
	def := amPens31
	if c.wb13 {
		def = amPens13
	}
	for i := range c.pen {
		c.pen[i] = l.X("pen"+itoa(i), hexColor(def[i]))
	}
	p := c.pen
	c.win, c.text = p[0], p[1]
	c.muted = l.palette.TextMuted
	if c.wb13 {
		// 1.3's screen: DetailPen 0, BlockPen 1; highlighting complements.
		c.shine, c.shadow, c.line = p[1], p[2], p[1]
		c.fill, c.fillText = p[3], p[2]
		c.cur, c.curText = p[3], p[2]
		c.mark, c.markText = p[1], p[0]
		c.hiText = p[3]
		c.bar, c.barText = p[1], p[0]
		c.menuHi, c.menuHiText = p[2], p[3]
		c.menuLine = p[0]
		c.ghost, c.focus, c.knob, c.dark = p[1], p[3], p[1], p[2]
		return c
	}
	// 2.0's DrawInfo pens: SHINE white, SHADOW black, FILL blue with
	// FILLTEXT black, HIGHLIGHTTEXT white, BARBLOCK white, BARDETAIL black.
	c.shine, c.shadow, c.line = p[2], p[1], p[1]
	c.fill, c.fillText = p[3], p[1]
	c.cur, c.curText = p[3], p[2]
	c.mark, c.markText = p[2], p[1]
	c.hiText = p[2]
	c.bar, c.barText = p[2], p[1]
	c.menuHi, c.menuHiText = p[1], p[2]
	c.menuLine = p[1]
	c.ghost, c.focus, c.knob, c.dark = p[1], p[1], p[3], p[1]
	return c
}

// comp is the complement of a pen colour on 1.3 — its pen number
// exclusive-ored with 3: blue ↔ orange, white ↔ black. Other colours are
// returned unchanged.
func (c *am) comp(col paintengine2d.Color) paintengine2d.Color {
	for i, p := range c.pen {
		if p == col {
			return c.pen[i^3]
		}
	}
	return col
}

// focusOn is the colour of the focus mark over a face of colour face: the
// look's focus colour, its complement where the face is that colour.
func (c *am) focusOn(face paintengine2d.Color) paintengine2d.Color {
	if face != c.focus {
		return c.focus
	}
	if c.wb13 {
		return c.comp(face)
	}
	return c.shine
}

// ---- drawing helpers ------------------------------------------------------------

// amBevel appends Intuition's 3D frame round the w × h cells at (x, y): hi
// along the top and left, lo along the bottom and right. Horizontal edges
// are one Amiga row tall; vertical edges are t cells wide (two for gadgets,
// one for the thin frame). The top row stops a pixel short, so the
// top-right corner is lo, and the left column runs to the bottom, so the
// bottom-left corner is hi: the colours meet in 45° steps. A recessed frame
// is the same drawing with the inks swapped.
func amBevel(hi, lo *rpInk, g rpGrid, x, y, w, h, t int) {
	if w < 2 || h < amRow+1 {
		return
	}
	hi.cells(g, x, y, w-1, amRow)
	hi.cells(g, x, y+amRow, 1, h-amRow)
	lo.cells(g, x+1, y+h-amRow, w-1, amRow)
	lo.cells(g, x+w-1, y, 1, h-amRow)
	if t > 1 && w > 3 && h > 2*amRow {
		hi.cells(g, x+1, y+amRow, 1, h-2*amRow)
		lo.cells(g, x+w-2, y+amRow, 1, h-2*amRow)
	}
}

// amBox appends a flat outline round the w × h cells at (x, y): its rows
// one Amiga row tall, its sides t cells wide.
func amBox(k *rpInk, g rpGrid, x, y, w, h, t int) {
	if w <= 0 || h <= 0 {
		return
	}
	if 2*t >= w || 2*amRow >= h {
		k.cells(g, x, y, w, h)
		return
	}
	k.cells(g, x, y, w, amRow)
	k.cells(g, x, y+h-amRow, w, amRow)
	k.cells(g, x, y+amRow, t, h-2*amRow)
	k.cells(g, x+w-t, y+amRow, t, h-2*amRow)
}

// amDots appends the focus mark, a dotted rectangle just inside the w × h
// cells at (x, y): every other hires pixel along the top and bottom rows,
// every other row down the sides.
func amDots(k *rpInk, g rpGrid, x, y, w, h int) {
	if w < 4 || h < 3*amRow {
		return
	}
	for i := 0; i < w; i += 2 {
		k.cells(g, x+i, y, 1, amRow)
		k.cells(g, x+i, y+h-amRow, 1, amRow)
	}
	for j := 2 * amRow; j+amRow < h-amRow; j += 2 * amRow {
		k.cells(g, x, y+j, 1, amRow)
		k.cells(g, x+w-1, y+j, 1, amRow)
	}
}

// amFocus paints the focus mark inside the w × h cells at (x, y) in col.
func amFocus(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, col paintengine2d.Color) {
	var k rpInk
	amDots(&k, g, x, y, w, h)
	k.fill(ctx, col)
}

// amFocusIn paints the focus mark just inside r.
func amFocusIn(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, col paintengine2d.Color) {
	g := rpGridAt(ctx, r, rpU(l))
	amFocus(ctx, g, 0, 0, g.w, g.h, col)
}

// amText draws text in the look's monospace face (Topaz's stand-in) fitted
// into r, never outside bound.
func amText(l *Classic, ctx *paintengine2d.Context, text string, r, bound paintengine2d.Rect, col paintengine2d.Color, align Align) {
	if text == "" || r.Empty() || bound.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(bound)
	l.drawFittedText(ctx, l.MonoFont(), text, r, col, align, 0)
	ctx.Restore()
}

// amGhostOver ghosts r the Intuition way: the dot grid in ink.
func amGhostOver(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, ink paintengine2d.Color) {
	rpPatFill(ctx, r, amGhost, ink, rpU(l))
}

// gadget paints a gadget body over the w × h cells at (x, y) and returns
// its label colour: on 3.1 the raised frame round the grey face, recessed
// and filled with FILLPEN blue while pressed; on 1.3 a white outline round
// the blue, complemented (orange face, black outline) while pressed.
func (c *am) gadget(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, pressed bool) paintengine2d.Color {
	if w <= 0 || h <= 0 {
		return c.text
	}
	if c.wb13 {
		face, ink, fg := c.win, c.line, c.text
		if pressed {
			face, ink, fg = c.comp(face), c.comp(ink), c.comp(fg)
		}
		ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(face))
		var k rpInk
		amBox(&k, g, x, y, w, h, am13Side)
		k.fill(ctx, ink)
		return fg
	}
	face, fg := c.win, c.text
	if pressed {
		face, fg = c.fill, c.fillText
	}
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(face))
	var hi, lo rpInk
	if pressed {
		amBevel(&lo, &hi, g, x, y, w, h, 2)
	} else {
		amBevel(&hi, &lo, g, x, y, w, h, 2)
	}
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
	return fg
}

// faceOf is the face colour gadget paints, pressed or not.
func (c *am) faceOf(pressed bool) paintengine2d.Color {
	switch {
	case !pressed:
		return c.win
	case c.wb13:
		return c.comp(c.win)
	}
	return c.fill
}

// stringFrame paints a string gadget's frame and face over the w × h cells
// at (x, y): Intuition's ridge on 3.1 — a raised frame round a recessed
// one, two rows tall and four pixels wide — and a white outline on 1.3.
// It returns how many cells the frame takes on every side.
func (c *am) stringFrame(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int) int {
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(c.win))
	if c.wb13 {
		var k rpInk
		amBox(&k, g, x, y, w, h, am13Side)
		k.fill(ctx, c.line)
		return 2
	}
	var hi, lo rpInk
	amBevel(&hi, &lo, g, x, y, w, h, 2)
	amBevel(&lo, &hi, g, x+2, y+amRow, w-4, h-2*amRow, 2)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
	return 4
}

// container paints a proportional gadget's container over the w × h cells
// at (x, y): a raised frame round grey on 3.1, a white outline round blue
// on 1.3. The knob keeps a gap of two cells (two pixels across, a row
// down) inside the frame.
func (c *am) container(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int) {
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(c.win))
	var hi, lo rpInk
	if c.wb13 {
		amBox(&hi, g, x, y, w, h, am13Side)
		hi.fill(ctx, c.line)
		return
	}
	amBevel(&hi, &lo, g, x, y, w, h, 2)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
}

// knobAt paints the proportional gadget's knob over the w × h cells at
// (x, y): 3.x's raised FILLPEN block, recessed while it is dragged, or
// 1.3's solid white block, complemented while it is dragged.
func (c *am) knobAt(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, pressed bool) {
	if w <= 0 || h <= 0 {
		return
	}
	if c.wb13 {
		col := c.knob
		if pressed {
			col = c.comp(col)
		}
		ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(col))
		return
	}
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(c.knob))
	var hi, lo rpInk
	if pressed {
		amBevel(&lo, &hi, g, x, y, w, h, 2)
	} else {
		amBevel(&hi, &lo, g, x, y, w, h, 2)
	}
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
}

// amEtched appends a 3.1 engraved line (shadow, then shine) of n cells along
// the axis at (x, y): two Amiga rows tall across a horizontal line, two
// pixels wide down a vertical one.
func amEtched(sh, hi *rpInk, g rpGrid, x, y, n int, vertical bool) {
	if vertical {
		sh.cells(g, x, y, 1, n)
		hi.cells(g, x+1, y, 1, n)
		return
	}
	sh.cells(g, x, y, n, amRow)
	hi.cells(g, x, y+amRow, n, amRow)
}

// ---- glyphs --------------------------------------------------------------------
//
// Glyphs are composed in hires pixels from rects and runs and doubled into
// cells by amTall; they are drawn from their geometry, not copied from any
// system bitmap.

// amTall doubles the rows of a mask drawn in hires pixels (at most half as
// many rows as a mask holds) so it can be emitted in cells.
func amTall(m rpMask) rpMask {
	o := rpNewMask(m.w, 2*m.h)
	for y := 0; y < m.h && 2*y+1 < o.h; y++ {
		o.rows[2*y] = m.rows[y]
		o.rows[2*y+1] = m.rows[y]
	}
	return o
}

// amChevron is the open arrowhead of Intuition's arrow gadgets pointing
// dir, in cells, its strokes two pixels wide. Pointing up or down it is n
// rows tall and 2n pixels wide, stepping a pixel a row; pointing left or
// right it is 2n-1 rows tall and 2n pixels wide, stepping two pixels a row.
func amChevron(dir Direction, n int) rpMask {
	if n < 2 {
		n = 2
	}
	if dir == DirLeft || dir == DirRight {
		if n > 8 {
			n = 8
		}
		rows, w := 2*n-1, 2*n
		m := rpNewMask(w, rows)
		for r := 0; r < rows; r++ {
			d := n - 1 - r
			if d < 0 {
				d = -d
			}
			x := 2 * d
			if dir == DirRight {
				x = w - 2 - x
			}
			m.span(x, x+2, r)
		}
		return amTall(m)
	}
	if n > 16 {
		n = 16
	}
	m := rpNewMask(2*n, n)
	for r := 0; r < n; r++ {
		y := r
		if dir == DirDown {
			y = n - 1 - r
		}
		m.span(n-1-r, n+1-r, y)
		m.span(n-1+r, n+1+r, y)
	}
	return amTall(m)
}

// amArrowIn emits into k a chevron pointing dir as large as fits the w × h
// cells at (x, y) of g (at most most rows for up and down), centred.
func amArrowIn(k *rpInk, g rpGrid, x, y, w, h int, dir Direction, most int) {
	var n int
	if dir == DirLeft || dir == DirRight {
		n = min((h/amRow+1)/2, w/2, most)
	} else {
		n = min(h/amRow, w/2, most)
	}
	if n < 2 {
		return
	}
	m := amChevron(dir, n)
	m.emit(k, g, x+(w-m.w)/2, y+(h-m.h)/2)
}

// amTick is the check mark in hires pixels, rows tall: a long stroke rising
// a pixel a row to the right and a short one falling into its foot, both sw
// pixels wide.
func amTick(rows, sw int) rpMask {
	if rows < 3 {
		rows = 3
	}
	if rows > 16 {
		rows = 16
	}
	j := (rows - 1) / 2
	m := rpNewMask(rows-1+j+sw, rows)
	for r := 0; r < rows; r++ {
		x := rows - 1 - r + j
		m.span(x, x+sw, r)
		if r >= rows-1-j {
			x2 := r - (rows - 1 - j)
			m.span(x2, x2+sw, r)
		}
	}
	return m
}

// amOvalEdge is how many cells row r of an oval w cells wide and rows Amiga
// rows tall is cut in from each side: the ellipse through the pixel
// centres.
func amOvalEdge(w, rows, r int) int {
	a := float64(w) * 0.5
	b := float64(rows) * 0.5
	dy := (float64(r) + 0.5 - b) / b
	v := 1 - dy*dy
	if v < 0 {
		v = 0
	}
	e := int(math.Floor(a - a*math.Sqrt(v) + 0.5))
	if 2*e >= w {
		e = (w - 1) / 2
	}
	return e
}

// amGlyphSet holds the fixed glyphs, built once.
type amGlyphSet struct {
	closeFrame, closeMid            rpMask // 3.1 close: 5 × 5 box, 3 × 3 middle
	depthInk, depthBack, depthFront rpMask // 3.1 depth: two 11 × 5 boxes
	zoomInk, zoomFill, zoomMid      rpMask // 3.1 zoom: 13 × 7 box, 7 × 4 box
	sizeInk, sizeFill               rpMask // 3.1 size: 11 × 6 triangle
	close13Box, close13Dot          rpMask // 1.3 close: 16 × 8 box, 4 × 2 dot
	backBlue, backDark              rpMask // 1.3 depth, to back
	frontBlue, frontDark            rpMask // 1.3 depth, to front
	size13                          rpMask // 1.3 size: two overlapping boxes
	key                             rpMask // the Amiga key, A knocked out
	cycle                           rpMask // the cycle gadget's loop
}

var amG = amBuildGlyphs()

func amRect(w, h, x, y, rw, rh int) rpMask {
	m := rpNewMask(w, h)
	m.rect(x, y, rw, rh)
	return m
}

func amBuildGlyphs() amGlyphSet {
	var s amGlyphSet
	// 3.1 close: a 5 × 5 box outlined in one pixel round its middle.
	box, mid := amRect(5, 5, 0, 0, 5, 5), amRect(5, 5, 1, 1, 3, 3)
	s.closeFrame, s.closeMid = amTall(box.minus(mid)), amTall(mid)
	// 3.1 depth: two 11 × 5 boxes, the front one four right and two down.
	back, backIn := amRect(15, 7, 0, 0, 11, 5), amRect(15, 7, 1, 1, 9, 3)
	front, frontIn := amRect(15, 7, 4, 2, 11, 5), amRect(15, 7, 5, 3, 9, 3)
	s.depthInk = amTall(back.minus(backIn).minus(front).or(front.minus(frontIn)))
	s.depthBack, s.depthFront = amTall(backIn.minus(front)), amTall(frontIn)
	// 3.1 zoom: a 13 × 7 box with a 7 × 4 box, two-pixel sides, in its
	// top-left corner.
	big, bigIn := amRect(13, 7, 0, 0, 13, 7), amRect(13, 7, 1, 1, 11, 5)
	small, smallIn := amRect(13, 7, 0, 0, 7, 4), amRect(13, 7, 2, 1, 3, 2)
	s.zoomInk = amTall(big.minus(bigIn).or(small.minus(smallIn)))
	s.zoomFill, s.zoomMid = amTall(bigIn.minus(small)), amTall(smallIn)
	// 3.1 size: an 11 × 6 right triangle, its right angle at the bottom
	// right, outlined in one pixel.
	tri := rpNewMask(11, 6)
	for r := 0; r < 6; r++ {
		tri.span(11-((r+1)*11+3)/6, 11, r)
	}
	edge := tri.outline()
	s.sizeInk, s.sizeFill = amTall(edge), amTall(tri.minus(edge))
	// 1.3 close: a 16 × 8 box with two-pixel sides round a 4 × 2 dot.
	s.close13Box = amTall(amRect(16, 8, 0, 0, 16, 8).minus(amRect(16, 8, 2, 1, 12, 6)))
	s.close13Dot = amTall(amRect(16, 8, 6, 3, 4, 2))
	// 1.3 depth: two 14 × 6 boxes, the front one four right and two down;
	// one outlined round white, one solid black.
	bk, bkOut := amRect(18, 8, 0, 0, 14, 6), amRect(18, 8, 0, 0, 14, 6).minus(amRect(18, 8, 2, 1, 10, 4))
	fr, frOut := amRect(18, 8, 4, 2, 14, 6), amRect(18, 8, 4, 2, 14, 6).minus(amRect(18, 8, 6, 3, 10, 4))
	s.backBlue, s.backDark = amTall(bkOut.minus(fr)), amTall(fr)
	s.frontBlue, s.frontDark = amTall(frOut), amTall(bk.minus(fr))
	// 1.3 size: two outlined boxes, the second overlapping the first.
	aOut := amRect(12, 6, 0, 0, 8, 4).minus(amRect(12, 6, 2, 1, 4, 2))
	b, bOut := amRect(12, 6, 4, 2, 8, 4), amRect(12, 6, 4, 2, 8, 4).minus(amRect(12, 6, 6, 3, 4, 2))
	s.size13 = amTall(aOut.minus(b).or(bOut))
	// The Amiga key: a keycap eleven pixels by seven rows with an A
	// knocked out of it.
	k := amRect(11, 7, 0, 0, 11, 7)
	letter := rpNewMask(11, 7)
	letter.span(5, 6, 1)
	letter.span(4, 5, 2)
	letter.span(6, 7, 2)
	letter.span(3, 4, 3)
	letter.span(7, 8, 3)
	letter.span(3, 8, 4)
	letter.span(2, 3, 5)
	letter.span(8, 9, 5)
	s.key = amTall(k.minus(letter))
	// The cycle gadget's loop: an oval ring 13 pixels by 8 rows, two pixels
	// thick, cut open at its upper left, where the top of the loop turns
	// down into an arrowhead seven pixels wide pointing at the rest of the
	// ring.
	const cw, cr, co = 13, 8, 2
	cyc := rpNewMask(cw+co, cr)
	for r := 0; r < cr; r++ {
		e := amOvalEdge(cw, cr, r)
		if r == 0 || r == cr-1 {
			a := e
			if r == 0 {
				a = e + 2
			}
			cyc.span(co+a, co+cw-e, r)
			continue
		}
		cyc.span(co+cw-e-2, co+cw-e, r)
		if r > cr/2 {
			cyc.span(co+e, co+e+2, r)
		}
	}
	cyc.span(0, 7, 1)
	cyc.span(1, 6, 2)
	cyc.span(2, 5, 3)
	s.cycle = amTall(cyc)
	return s
}

// ---- parts --------------------------------------------------------------------

func (e amigaEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	pressed := st.Pressed() && !st.Disabled()
	switch role {
	case RoleButton, RoleCombo, RoleCheck:
		return c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	case RoleTool:
		switch {
		case st.Disabled():
		case pressed || (st.Toggle() && st.Checked()):
			return c.gadget(ctx, g, 0, 0, g.w, g.h, true)
		case st.Hovered():
			return c.gadget(ctx, g, 0, 0, g.w, g.h, false)
		}
		return c.text
	case RoleField:
		c.stringFrame(ctx, g, 0, 0, g.w, g.h)
		return c.text
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.fill))
			return c.fillText
		}
		return c.text
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.menuHi))
			return c.menuHiText
		}
		return c.barText
	case RoleThumb:
		c.knobAt(ctx, g, 0, 0, g.w, g.h, pressed)
		return c.text
	case RoleTrack:
		c.container(ctx, g, 0, 0, g.w, g.h)
		return c.text
	case RoleTab:
		return c.text
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	return c.text
}

// CheckIndicator is the GadTools check box, drawn over box: a raised box
// with a bold tick when checked, recessed while the mouse holds it. 1.3
// had no check box gadget; there it is drawn the way 1.3 applications drew
// their toggles, a white outline with a white tick, complemented while held.
func (e amigaEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	if g.w < 8 || g.h < 3*amRow {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	if checked {
		rows := min((g.h-4-2)/amRow, (g.w-6)*2/3, 7)
		if rows >= 3 {
			sw := 2
			if rows >= 6 {
				sw = 3
			}
			m := amTall(amTick(rows, sw))
			var k rpInk
			m.emit(&k, g, (g.w-m.w+1)/2, (g.h-m.h)/2)
			k.fill(ctx, fg)
		}
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// RadioIndicator is the GadTools mutual-exclude button, drawn over box: a
// raised oval that recesses with its middle filled FILLPEN blue when
// selected. On 1.3, which had no such gadget, it is a white oval outline
// round a white dot, complemented while the mouse holds it.
func (e amigaEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	w, rows := g.w, g.h/amRow
	if w < 7 || rows < 4 {
		return
	}
	g = g.sub(0, (g.h-rows*amRow)/2, w, rows*amRow)
	pressed := st.Pressed() && !st.Disabled()
	var hi, lo, face rpInk
	for r := 0; r < rows; r++ {
		ed := amOvalEdge(w, rows, r)
		y := r * amRow
		if r == 0 || r == rows-1 {
			k := &hi
			if r == rows-1 {
				k = &lo
			}
			k.cells(g, ed, y, w-2*ed, amRow)
			continue
		}
		sw := max(amOvalEdge(w, rows, r-1), amOvalEdge(w, rows, r+1)) - ed
		if sw < 2 {
			sw = 2
		}
		if 2*(ed+sw) >= w {
			sw = (w - 2*ed) / 2
		}
		hi.cells(g, ed, y, sw, amRow)
		lo.cells(g, w-ed-sw, y, sw, amRow)
		face.cells(g, ed+sw, y, w-2*(ed+sw), amRow)
	}
	if c.wb13 {
		faceCol, ink := c.win, c.line
		if pressed {
			faceCol, ink = c.comp(faceCol), c.comp(ink)
		}
		face.fill(ctx, faceCol)
		hi.fill(ctx, ink)
		lo.fill(ctx, ink)
		if selected {
			// The dot: the oval two pixels and a row inside the ring.
			dw, dr := w-8, rows-4
			if dw >= 3 && dr >= 1 {
				var dot rpInk
				for r := 0; r < dr; r++ {
					ed := amOvalEdge(dw, dr, r)
					dot.cells(g, 4+ed, (2+r)*amRow, dw-2*ed, amRow)
				}
				dot.fill(ctx, ink)
			}
		}
	} else {
		faceCol := c.win
		if selected {
			faceCol = c.fill
		}
		face.fill(ctx, faceCol)
		if selected || pressed {
			hi.fill(ctx, c.shadow)
			lo.fill(ctx, c.shine)
		} else {
			hi.fill(ctx, c.shine)
			lo.fill(ctx, c.shadow)
		}
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// Arrow is Intuition's open chevron, as large as fits b.
func (e amigaEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	var k rpInk
	most := 5
	if dir == DirLeft || dir == DirRight {
		most = 4
	}
	amArrowIn(&k, g, 0, 0, g.w, g.h, dir, most)
	k.fill(ctx, col)
}

// Expander: the Amiga had no tree gadget; the disclosure mark is the arrow
// gadgets' chevron, pointing right when closed and down when open.
func (e amigaEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	var k rpInk
	amArrowIn(&k, g, 0, 0, g.w, g.h, dir, 3)
	k.fill(ctx, col)
}

// MenuHighlight complements the item: a black bar on 3.1 and on 1.3 the
// complement of white, pen 2.
func (e amigaEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(amColors(l).menuHi))
}

// MenuTextColor: blue (1.3) or black (3.1) items, orange or white when hot.
func (e amigaEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := amColors(l)
	if hot {
		return c.menuHiText
	}
	return c.barText
}

// FieldFocusRing: an active string gadget shows only its cursor.
func (amigaEngine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is the focus mark, a dotted rectangle just inside b.
func (e amigaEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	amFocusIn(l, ctx, b, amColors(l).focus)
}

// ---- scrollbars -----------------------------------------------------------------

// ScrollBarStyle: 3.1's scrollers are 18 pixels across with their two
// arrow gadgets together at the bottom (right) end, as in window borders;
// 1.3's proportional gadgets are 16 across and have no arrows.
func (amigaEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	if amColors(l).wb13 {
		return ScrollBarStyle{Thickness: 16, Arrows: ArrowsNone, MinThumb: 16}
	}
	return ScrollBarStyle{Thickness: 18, Arrows: ArrowsTogetherEnd, ArrowLen: 20, MinThumb: 20}
}

// DrawScrollBarParts is a proportional gadget: the knob in its container,
// four cells clear of the container's edge, and on 3.1 the arrow gadgets —
// raised buttons with open chevrons. With nothing to scroll the knob fills
// the container, as Intuition drew a full body.
func (e amigaEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := amColors(l)
	u := rpU(l)
	if p.Bar.Empty() {
		return
	}
	gb := rpGridAt(ctx, p.Bar, u)
	ctx.DrawRect(gb.rect(), paintengine2d.Fill(c.win))
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	e.scrollArrow(l, ctx, p.Dec, dec, st.Pressed == ScrollDec && !st.Disabled)
	e.scrollArrow(l, ctx, p.Inc, inc, st.Pressed == ScrollInc && !st.Disabled)
	if p.Track.Empty() {
		return
	}
	tg := rpGridAt(ctx, p.Track, u)
	if tg.w < 4 || tg.h < 4 {
		return
	}
	c.container(ctx, tg, 0, 0, tg.w, tg.h)
	along, across := tg.h, tg.w
	if !vertical {
		along, across = tg.w, tg.h
	}
	// The knob keeps the frame and a gap clear; a narrow bar (1.5x, where
	// cells round up to two pixels) narrows the gap to one cell.
	in := 4
	if across < 16 {
		in = 3
	}
	a0, a1 := in, along-in // the knob's span along the axis, in cells
	if !p.Thumb.Empty() && !st.Disabled && along > 2*in+3*amRow {
		t0, t1 := p.Thumb.Min.Y-tg.y, p.Thumb.Max.Y-tg.y
		if !vertical {
			t0, t1 = p.Thumb.Min.X-tg.x, p.Thumb.Max.X-tg.x
		}
		// The thumb's span over the whole track maps onto the knob's
		// travel inside the container.
		k := float32(along-2*in) / float32(along)
		a0 = max(in, in+int(t0/u*k+0.5))
		a1 = min(along-in, in+int(t1/u*k+0.5))
		if a1-a0 < 3*amRow {
			a1 = min(along-in, a0+3*amRow)
			a0 = a1 - 3*amRow
		}
	}
	pressed := st.Pressed == ScrollThumbPart && !st.Disabled
	if vertical {
		c.knobAt(ctx, tg, in, a0, tg.w-2*in, a1-a0, pressed)
	} else {
		c.knobAt(ctx, tg, a0, in, a1-a0, tg.h-2*in, pressed)
	}
}

// scrollArrow is one of 3.1's arrow gadgets.
func (e amigaEngine) scrollArrow(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, dir Direction, pressed bool) {
	if r.Empty() {
		return
	}
	c := amColors(l)
	g := rpGridAt(ctx, r, rpU(l))
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	var k rpInk
	amArrowIn(&k, g, 2, amRow+1, g.w-4, g.h-2*amRow-2, dir, 5)
	k.fill(ctx, fg)
}

// DrawScrollBar paints a bare container and knob (widgets that lay out
// their own bar).
func (e amigaEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered(), Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -----------------------------------------------------------------

// GroupBoxInsets: the frame, its title set into the top line.
func (amigaEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	u := rpU(l)
	pad := l.metrics.Pad
	top := pad + 2*amRow*u
	if hasTitle {
		top = l.MonoFont().Height() + pad*0.5
	}
	side := pad + 2*u
	return Insets{Top: top, Right: side, Bottom: pad + 2*amRow*u, Left: side}
}

// DrawGroupBox: GadTools had no titled frames; this is the bevel-box
// grouping of 3.x preferences editors — a thin ridge (a raised frame round a
// recessed one) with the title in HIGHLIGHTTEXTPEN white set into its top
// line. 1.3 draws a white outline and an orange title. A raised box is a
// raised panel with its title inside.
func (e amigaEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := amColors(l)
	u := rpU(l)
	f := l.MonoFont()
	pad := l.metrics.Pad
	if raised {
		e.DrawPanel(l, ctx, b, true)
		if title != "" {
			amText(l, ctx, title, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+2*amRow*u, b.Dx()-2*pad, f.Height()), b, c.hiText, AlignStart)
		}
		return
	}
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	top := 0
	if title != "" {
		top = int(f.Height()*0.5/u) - amRow
		if top < 0 {
			top = 0
		}
	}
	fg := g.sub(0, top, g.w, g.h-top)
	if fg.w < 6 || fg.h < 3*amRow {
		return
	}
	var hi, lo rpInk
	if c.wb13 {
		amBox(&hi, fg, 0, 0, fg.w, fg.h, am13Side)
		hi.fill(ctx, c.line)
	} else {
		amBevel(&hi, &lo, fg, 0, 0, fg.w, fg.h, 1)
		amBevel(&lo, &hi, fg, 1, amRow, fg.w-2, fg.h-2*amRow, 1)
		hi.fill(ctx, c.shine)
		lo.fill(ctx, c.shadow)
	}
	if title == "" {
		return
	}
	tx := 8
	tw := int(f.Advance(title)/u+0.5) + 8
	if tw > g.w-2*tx {
		tw = g.w - 2*tx
	}
	if tw <= 8 {
		return
	}
	th := min(int(f.Height()/u+0.5), g.h)
	ctx.DrawRect(g.at(tx, 0, tw, th), paintengine2d.Fill(c.win))
	amText(l, ctx, title, g.at(tx+4, 0, tw-4, th), b, c.hiText, AlignStart)
}

// amBarRows is a window title bar's height in Amiga rows: eleven for
// Topaz 8 — a shine row, nine rows of title, a shadow row — and more when
// the display scale makes the font taller than the cells.
func amBarRows(l *Classic) int {
	u := rpU(l)
	n := int(math.Ceil(float64(l.MonoFont().Size/u)/amRow)) + 1
	if n < 9 {
		n = 9
	}
	return n + 2
}

// amFrameFits reports whether a frame of g has room for its title bar and
// gadgets.
func amFrameFits(g rpGrid, bh int) bool { return g.w >= 48 && g.h >= bh+3*amRow }

// WindowFrameInsets: Intuition's borders, four pixels left and right, two
// rows at the bottom and the title bar at the top.
func (amigaEngine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: float32(amBarRows(l)*amRow) * u, Right: 4 * u, Bottom: 2 * amRow * u, Left: 4 * u}
}

// WindowCloseRect is the close gadget at the left end of the title bar.
func (e amigaEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	bh := amBarRows(l) * amRow
	if !amFrameFits(g, bh) {
		return paintengine2d.Rect{}
	}
	if amColors(l).wb13 {
		return g.at(0, 0, 20, bh-amRow)
	}
	return g.at(0, 0, 20, bh)
}

// PopupShadow: nothing on the Amiga cast a shadow.
func (amigaEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (amigaEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// ItemFocus: GadTools listviews had no keyboard focus; the current row
// carries the engine's focus mark.
func (e amigaEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := amColors(l)
	col := c.focus
	if st.Checked() {
		col = c.focusOn(c.fill)
	}
	g := rpGridAt(ctx, b, rpU(l))
	amFocus(ctx, g, 1, 0, g.w-2, g.h, col)
}

// ViewFrameInsets: a listview's frame, two cells on every side.
func (amigaEngine) ViewFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: 2 * u, Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// DrawViewFrame is a GadTools listview: raised (the recessed box was for
// read-only lists) on 3.1, a white outline on 1.3.
func (e amigaEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	c.container(ctx, g, 0, 0, g.w, g.h)
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// StyleHint: requesters put the positive choice at the lower left and
// Cancel at the right; GadTools right-aligns a label placed left of its
// gadget.
func (amigaEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintDialogPrimaryFirst, HintFormLabelsRight:
		return 1
	case HintMnemonics:
		// Intuition menus show Amiga-key shortcuts, no underlines.
		return MnemonicsNever
	}
	return 0
}

// ControlFont: every gadget is labelled in Topaz, the look's monospace
// stand-in.
func (amigaEngine) ControlFont(l *Classic, role Role) *Font { return l.MonoFont() }

// ToolBarInsets: four pixels either side, no grip.
func (amigaEngine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(4), Right: l.S(4)} }

// TabOutset: tabs do not overlap.
func (amigaEngine) TabOutset(*Classic) Insets { return Insets{} }

// DrawWindowBackground is the window's pen 0: blue on 1.3, grey on 3.1.
func (amigaEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(amColors(l).win))
}

// DrawTabPane: the page under ReAction-style tabs, a raised box (1.3: a
// white outline) whose top edge is the tab bar's bottom line.
func (e amigaEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	c.container(ctx, g, 0, 0, g.w, g.h)
}

// DrawWindowFrame is an Intuition window: its borders, the title bar with
// the close gadget at the left and the depth gadget (3.1: zoom and depth)
// at the right, and the title. See frame31 and frame13.
func (e amigaEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	bh := amBarRows(l) * amRow
	if !amFrameFits(g, bh) {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		return
	}
	if c.wb13 {
		e.frame13(l, ctx, b, g, bh, title, st)
		return
	}
	e.frame31(l, ctx, b, g, bh, title, st)
}

// frame31 is a 3.1 window. The outer edge is a thin raised frame and the
// inner edge round the grey body a thin recessed one; between them the
// borders are FILLPEN blue while the window is active and grey when it is
// not. The title bar reads a shine row, nine rows of fill with the black
// title, and a shadow row. Each gadget is a thin raised box: the close
// gadget (a 5 × 5 box, white in the middle, grey when inactive) at the
// left, the zoom gadget (a box in a box) and the depth gadget (two
// overlapping boxes) at the right.
func (e amigaEngine) frame31(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, g rpGrid, bh int, title string, st WindowState) {
	c := amColors(l)
	W, H := g.w, g.h
	fill := c.fill
	if !st.Active {
		fill = c.win
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(fill))
	ctx.DrawRect(g.at(4, bh, W-8, H-bh-2*amRow), paintengine2d.Fill(c.win))
	var hi, lo, grey, blue rpInk
	amBevel(&hi, &lo, g, 0, 0, W, H, 1)
	amBevel(&lo, &hi, g, 3, bh-amRow, W-6, H-bh, 1)
	rows := bh / amRow
	x0, x1 := 0, W
	if st.CanClose {
		pressed := st.ClosePress
		if pressed {
			amBevel(&lo, &hi, g, 0, 0, 20, bh, 1)
		} else {
			amBevel(&hi, &lo, g, 0, 0, 20, bh, 1)
		}
		oy := (rows - 5) / 2 * amRow
		amG.closeFrame.emit(&lo, g, 7, oy)
		switch {
		case pressed:
			amG.closeMid.emit(&lo, g, 7, oy)
		case st.Active:
			amG.closeMid.emit(&hi, g, 7, oy)
		default:
			amG.closeMid.emit(&grey, g, 7, oy)
		}
		x0 = 20
	}
	if W >= x0+24+24 {
		x1 = W - 24
		amBevel(&hi, &lo, g, x1, 0, 24, bh, 1)
		oy := (rows - 7) / 2 * amRow
		amG.depthInk.emit(&lo, g, x1+4, oy)
		amG.depthBack.emit(&grey, g, x1+4, oy)
		amG.depthFront.emit(&hi, g, x1+4, oy)
		if st.Maximizable && W >= x0+48+24 {
			x1 -= 24
			amBevel(&hi, &lo, g, x1, 0, 24, bh, 1)
			amG.zoomInk.emit(&lo, g, x1+5, oy)
			amG.zoomFill.emit(&blue, g, x1+5, oy)
			amG.zoomMid.emit(&hi, g, x1+5, oy)
		}
	}
	grey.fill(ctx, c.win)
	blue.fill(ctx, fill)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
	if title != "" && x1-x0 > 16 {
		tb := g.at(x0+6, amRow, x1-x0-12, bh-2*amRow)
		amText(l, ctx, title, tb, tb.Intersect(b), c.text, AlignStart)
	}
}

// frame13 is a 1.3 window: white borders (two pixels at the sides, a row
// at the bottom) round the blue, a white title bar with blue text, the
// close gadget — a blue box round a black dot — at the left and the two
// depth gadgets at the right, each set off by a blue line. Between the
// title and the depth gadgets run the drag bar's two blue stripes, dotted
// with white while the window is inactive. A pressed close gadget is
// complemented.
func (e amigaEngine) frame13(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, g rpGrid, bh int, title string, st WindowState) {
	c := amColors(l)
	u := rpU(l)
	W, H := g.w, g.h
	bw := bh - amRow // the white bar; the next row is the border's blue line
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var white, dark, blue, orange, dot rpInk
	white.cells(g, 0, 0, W, bw)
	white.cells(g, 0, bw, 2, H-bw)
	white.cells(g, W-2, bw, 2, H-bw)
	white.cells(g, 2, H-amRow, W-4, amRow)
	oy := (bw/amRow - 8) / 2 * amRow
	x0, x1 := 0, W
	if st.CanClose {
		blue.cells(g, 20, 0, 2, bw)
		if st.ClosePress {
			dark.cells(g, 0, 0, 20, bw)
			amG.close13Box.emit(&orange, g, 2, oy)
			amG.close13Dot.emit(&dot, g, 2, oy)
		} else {
			amG.close13Box.emit(&blue, g, 2, oy)
			amG.close13Dot.emit(&dark, g, 2, oy)
		}
		x0 = 22
	}
	if W >= x0+48+24 {
		x1 = W - 48
		blue.cells(g, W-48, 0, 2, bw)
		blue.cells(g, W-24, 0, 2, bw)
		amG.backBlue.emit(&blue, g, W-46+2, oy)
		amG.backDark.emit(&dark, g, W-46+2, oy)
		amG.frontDark.emit(&dark, g, W-22+2, oy)
		amG.frontBlue.emit(&blue, g, W-22+2, oy)
	}
	// The drag bar's stripes: rows 2–3 and 6–7 of the ten-row bar (kept
	// central when the scale makes the bar taller).
	f := l.MonoFont()
	tx := x0 + 4
	tw := 0
	if title != "" {
		tw = min(int(f.Advance(title)/u+0.5)+1, x1-tx-8)
	}
	sx0, sx1 := tx, x1-4
	if tw > 0 {
		sx0 = tx + tw + 6
	}
	sy := max(0, bw/amRow-10) / 2 * amRow
	var stripes [2]paintengine2d.Rect
	if sx1-sx0 >= 4 {
		for i := range stripes {
			stripes[i] = g.at(sx0, sy+(2+4*i)*amRow, sx1-sx0, 2*amRow)
			blue.add(stripes[i])
		}
	}
	white.fill(ctx, c.line)
	dark.fill(ctx, c.dark)
	blue.fill(ctx, c.win)
	orange.fill(ctx, c.comp(c.win))
	dot.fill(ctx, c.line)
	if !st.Active {
		for _, r := range stripes {
			rpPatFill(ctx, r, amGhost, c.line, u)
		}
	}
	if tw > 0 {
		tb := g.at(tx, 0, tw+2, bw)
		amText(l, ctx, title, tb, tb.Intersect(b), c.barText, AlignStart)
	}
}

// ---- controls -------------------------------------------------------------------

// DrawPanel: a raised panel is a raised box (a DrawBevelBox; 1.3: a white
// outline); a flat one is the window colour.
func (e amigaEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if !raised || g.w < 6 || g.h < 3*amRow {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		return
	}
	c.container(ctx, g, 0, 0, g.w, g.h)
}

// DrawButton is a GadTools button: the raised frame round grey with the
// label centred, recessed and FILLPEN blue while pressed. 1.3 had no
// system button; its applications drew a white outline and complemented it
// when hit. The Amiga marked no default button. Focus is the dotted mark
// inside the frame; a disabled button is ghosted.
func (e amigaEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 3*amRow {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	amText(l, ctx, label, g.at(3, amRow, g.w-6, g.h-2*amRow), b, fg, AlignCenter)
	if st.Focused() && !st.Disabled() {
		amFocus(ctx, g, 3, amRow+1, g.w-6, g.h-2*amRow-2, c.focusOn(c.faceOf(pressed)))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawLabel draws text in Topaz's stand-in, nudged to read on the window.
func (e amigaEngine) DrawLabel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	amText(l, ctx, text, b, b, l.onWindow(col), align)
}

// toggleLabel draws the caption of a check box, radio or switch at lx and
// the focus mark round it (round the gadget gad when there is none).
func (e amigaEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, gad paintengine2d.Rect, lx float32, st ControlState, label string) {
	c := amColors(l)
	u := rpU(l)
	if label == "" {
		if st.Focused() {
			amFocusIn(l, ctx, gad.Inset(-2*u).Intersect(b), c.focus)
		}
		return
	}
	f := l.MonoFont()
	lb := paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx, b.Dy())
	amText(l, ctx, label, lb, b, c.text, AlignStart)
	if st.Disabled() {
		amGhostOver(l, ctx, rpTextBox(f, label, lb, AlignStart).Intersect(b), c.ghost)
	}
	if st.Focused() {
		amFocusIn(l, ctx, labelFocusRect(f, label, lb, b), c.focus)
	}
}

// DrawCheckbox: the 26 × 11 check box with its label to the right.
func (e amigaEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	u := rpU(l)
	side := l.metrics.Checkbox
	if side > b.Dy() {
		side = b.Dy()
	}
	hc := int(side/u + 0.5)
	wc := min((hc*13+5)/11, int(b.Dx()/u))
	box := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-float32(hc)*u)*0.5), float32(wc)*u, float32(hc)*u)
	e.CheckIndicator(l, ctx, box, st, checked)
	lx := min(b.Min.X+l.metrics.Checkbox+l.S(8), box.Max.X+6*u)
	e.toggleLabel(l, ctx, b, box, lx, st, label)
}

// DrawRadio: the 17 × 9 mutual-exclude button with its label to the right.
func (e amigaEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	u := rpU(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	if side > b.Dy() {
		side = b.Dy()
	}
	hc := int(side/u+0.5) &^ 1
	wc := min(hc-1, int(b.Dx()/u))
	box := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-float32(hc)*u)*0.5), float32(wc)*u, float32(hc)*u)
	e.RadioIndicator(l, ctx, box, st, selected)
	lx := min(b.Min.X+side+l.S(8), box.Max.X+6*u)
	e.toggleLabel(l, ctx, b, box, lx, st, label)
}

// DrawSwitch: the Amiga had no switch. This one is drawn in the gadget
// language: a recessed slot (1.3: an outline) that fills with FILLPEN blue
// (1.3: orange) when on, and a raised knob (1.3: the white prop knob) at
// the end the switch is set to.
func (e amigaEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := amColors(l)
	u := rpU(l)
	tw, th := l.metrics.SwitchW, l.metrics.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-th)*0.5), tw, th)
	g := rpGridAt(ctx, track, u)
	if g.w >= 12 && g.h >= 4*amRow {
		face := c.win
		if on {
			face = c.fill
		}
		ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
		var hi, lo rpInk
		if c.wb13 {
			amBox(&hi, g, 0, 0, g.w, g.h, am13Side)
			hi.fill(ctx, c.line)
		} else {
			amBevel(&lo, &hi, g, 0, 0, g.w, g.h, 2)
			hi.fill(ctx, c.shine)
			lo.fill(ctx, c.shadow)
		}
		kw := g.w/2 - 2
		kx := 2
		if on {
			kx = g.w - 2 - kw
		}
		pressed := st.Pressed() && !st.Disabled()
		if c.wb13 {
			c.knobAt(ctx, g, kx+2, 2*amRow, kw-4, g.h-4*amRow, pressed)
		} else {
			c.gadget(ctx, g, kx, amRow, kw, g.h-2*amRow, pressed)
		}
		if st.Disabled() {
			amGhostOver(l, ctx, g.rect(), c.ghost)
		}
	}
	e.toggleLabel(l, ctx, b, track, track.Max.X+l.S(8), st, label)
}

// DrawComboBox is the cycle gadget: a button with the looping arrow at its
// left, a divider, and the current choice. Pressed or open it recesses
// (1.3, which had no cycle gadget, complements it).
func (e amigaEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 30 || g.h < 3*amRow+2 {
		return
	}
	pressed := (open || st.Pressed()) && !st.Disabled()
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	gw := 20
	var ink, hi rpInk
	m := amG.cycle
	if g.h >= m.h+2*amRow {
		m.emit(&ink, g, (gw-m.w)/2+1, (g.h-m.h)/2)
	}
	dy, dh := 2*amRow, g.h-4*amRow
	if c.wb13 {
		ink.cells(g, gw, dy, 1, dh)
	} else {
		amEtched(&ink, &hi, g, gw, dy, dh, true)
	}
	hi.fill(ctx, c.shine)
	ink.fill(ctx, fg)
	tb := g.at(gw+4, amRow, g.w-gw-8, g.h-2*amRow)
	amText(l, ctx, text, tb, b, fg, AlignCenter)
	if st.Focused() && !open && !st.Disabled() {
		amFocus(ctx, g, gw+3, amRow+1, g.w-gw-6, g.h-2*amRow-2, c.focusOn(c.faceOf(pressed)))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// fieldOpts styles the text of a string gadget: pen 1 text and the string
// cursor, a block over the character in the complement of the gadget's
// pens — blue with a white character on 3.1, orange with black on 1.3.
// Intuition's string gadgets had no selection; a selection here is marked
// in the bar colours (black on white, 1.3 blue on white) so the cursor
// stays apart from it.
func (c *am) fieldOpts() amTextOpts {
	return amTextOpts{text: c.text, muted: c.muted, sel: c.mark, selText: c.markText, cur: c.cur, curText: c.curText}
}

// DrawTextField is a string gadget: the ridge frame (1.3: a white outline)
// round the window colour, the text, and the block cursor while active.
func (e amigaEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 3*amRow {
		return
	}
	fr := c.stringFrame(ctx, g, 0, 0, g.w, g.h)
	pad := l.metrics.FieldPad
	if pad < float32(fr+1)*u {
		pad = float32(fr+1) * u
	}
	inner := paintengine2d.Rect{
		Min: paintengine2d.Pt(b.Min.X+pad, g.y+float32(fr)*u),
		Max: paintengine2d.Pt(b.Max.X-pad, g.y+float32(g.h-fr)*u),
	}
	amFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.fieldOpts())
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawTextArea: a multi-line string gadget in the same frame.
func (e amigaEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 3*amRow {
		return
	}
	fr := c.stringFrame(ctx, g, 0, 0, g.w, g.h)
	pad := l.metrics.FieldPad
	if pad < float32(fr+1)*u {
		pad = float32(fr+1) * u
	}
	amAreaText(l, ctx, b.Inset(pad), st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, c.fieldOpts())
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawSpinner: Intuition's integer gadget had no arrows; these are two
// stacked arrow gadgets like a scroller's, the held one recessed (1.3:
// complemented).
func (e amigaEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 6*amRow {
		return
	}
	mid := g.h / 2
	for _, part := range [2]struct {
		y, h    int
		dir     Direction
		pressed bool
	}{{0, mid, DirUp, upPress}, {mid, g.h - mid, DirDown, downPress}} {
		fg := c.gadget(ctx, g, 0, part.y, g.w, part.h, part.pressed && !st.Disabled())
		var k rpInk
		amArrowIn(&k, g, 2, part.y+amRow+1, g.w-4, part.h-2*amRow-2, part.dir, 4)
		k.fill(ctx, fg)
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawSlider is a GadTools slider: the scroller's container, 10 rows tall,
// and its knob.
func (e amigaEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 24 || g.h < 5*amRow {
		return
	}
	t = clamp1(t)
	ch := min(g.h, 10*amRow)
	y0 := (g.h - ch) / 2
	c.container(ctx, g, 0, y0, g.w, ch)
	const in = 4
	kw := min(16, (g.w-2*in)/3)
	kx := in + int(float32(g.w-2*in-kw)*t+0.5)
	c.knobAt(ctx, g, kx, y0+in, kw, ch-2*in, st.Pressed() && !st.Disabled())
	if st.Focused() && !st.Disabled() {
		amFocus(ctx, g, 0, 0, g.w, g.h, c.focus)
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawProgressBar: the Workbench's fuel gauge — a thin recessed frame (1.3:
// a white outline) filling with FILLPEN blue (1.3: orange) from the left.
// A busy bar runs diagonal stripes, two pixels to a row.
func (e amigaEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 3*amRow {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var hi, lo rpInk
	side := 1
	if c.wb13 {
		side = am13Side
		amBox(&hi, g, 0, 0, g.w, g.h, side)
		hi.fill(ctx, c.line)
	} else {
		amBevel(&lo, &hi, g, 0, 0, g.w, g.h, 1)
		hi.fill(ctx, c.shine)
		lo.fill(ctx, c.shadow)
	}
	in := g.sub(side+1, amRow, g.w-2*side-2, g.h-2*amRow)
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		amStripes(ctx, in, c.fill, int(phase*8))
	} else if n := int(float32(in.w)*clamp1(t) + 0.5); n > 0 {
		ctx.DrawRect(in.at(0, 0, n, in.h), paintengine2d.Fill(c.fill))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// amStripes fills g with diagonal stripes four cells wide, eight apart,
// stepping two cells a row (a hires pixel pair per Amiga row), shifted by
// off cells.
func amStripes(ctx *paintengine2d.Context, g rpGrid, col paintengine2d.Color, off int) {
	var k rpInk
	for y := 0; y < g.h; y += amRow {
		h := min(amRow, g.h-y)
		s0 := (y + off) % 8
		for x := -8 + s0; x < g.w; x += 8 {
			a, z := max(x, 0), min(x+4, g.w)
			if z > a {
				k.cells(g, a, y, z-a, h)
			}
		}
	}
	k.fill(ctx, col)
}

// DrawTabBar: ReAction-style tabs stand on the page's top edge, a shine
// line (1.3: white).
func (e amigaEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	if g.h < amRow {
		return
	}
	var k rpInk
	k.cells(g, 0, g.h-amRow, g.w, amRow)
	if c.wb13 {
		k.fill(ctx, c.line)
		return
	}
	k.fill(ctx, c.shine)
}

// DrawTab: the Amiga had no tabs until ReAction's clicktab (3.5); these are
// in its manner — raised tabs whose top corners step in a pixel a row, the
// unselected ones two rows lower and standing on the page's edge, the
// selected one open into the page. Pressed tabs recess to FILLPEN (1.3:
// complement).
func (e amigaEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 5*amRow {
		return
	}
	top, bot := 0, g.h
	if !selected {
		top, bot = 2*amRow, g.h-amRow
	}
	pressed := st.Pressed() && !st.Disabled()
	face, fg := c.win, c.text
	if pressed {
		face, fg = c.faceOf(true), c.fillText
		if c.wb13 {
			fg = c.comp(c.text)
		}
	}
	var hi, lo, fk, edge rpInk
	w, h := g.w, bot-top
	hi.cells(g, 2, top, w-4, amRow)
	hi.cells(g, 1, top+amRow, 1, amRow)
	lo.cells(g, w-2, top+amRow, 1, amRow)
	fk.cells(g, 2, top+amRow, w-4, amRow)
	t := 2
	hi.cells(g, 0, top+2*amRow, t, h-2*amRow)
	lo.cells(g, w-t, top+2*amRow, t, h-2*amRow)
	fk.cells(g, t, top+2*amRow, w-2*t, h-2*amRow)
	if !selected {
		// The page's top edge runs on under an unselected tab.
		edge.cells(g, 0, g.h-amRow, w, amRow)
	}
	fk.fill(ctx, face)
	switch {
	case c.wb13:
		ink := c.line
		if pressed {
			ink = c.comp(ink)
		}
		hi.fill(ctx, ink)
		lo.fill(ctx, ink)
		edge.fill(ctx, c.line)
	case pressed:
		hi.fill(ctx, c.shadow)
		lo.fill(ctx, c.shine)
		edge.fill(ctx, c.shine)
	default:
		hi.fill(ctx, c.shine)
		lo.fill(ctx, c.shadow)
		edge.fill(ctx, c.shine)
	}
	lb := g.at(t+2, top+amRow, w-2*t-4, h-amRow)
	f := l.MonoFont()
	amText(l, ctx, label, lb, b, fg, AlignCenter)
	if st.Focused() && selected {
		fr := lb
		if label != "" {
			fr = rpTextBox(f, label, lb, AlignCenter).Inset(-3 * u).Intersect(lb)
		}
		amFocusIn(l, ctx, fr, c.focusOn(face))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.at(0, top, w, h), c.ghost)
	}
}

// DrawMenuBar is the screen title bar doubling as the menu strip: white,
// with 3.1's black trim line along its bottom.
func (e amigaEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bar))
	if !c.wb13 && g.h > 2*amRow {
		var k rpInk
		k.cells(g, 0, g.h-amRow, g.w, amRow)
		k.fill(ctx, c.shadow)
	}
}

// DrawMenuTitle: blue (1.3) or black (3.1) titles; the title under the
// pointer while menus are up is complemented — a dark box with orange or
// white text.
func (e amigaEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	h := g.h
	if !c.wb13 {
		h -= amRow
	}
	if h <= 0 {
		return
	}
	hot := !st.Disabled() && (open || st.Pressed() || st.Hovered())
	col, bg := c.barText, c.bar
	if hot {
		ctx.DrawRect(g.at(0, 0, g.w, h), paintengine2d.Fill(c.menuHi))
		col, bg = c.menuHiText, c.menuHi
	}
	amText(l, ctx, label, g.at(0, 0, g.w, h), b, col, AlignCenter)
	if st.Focused() && !open {
		amFocus(ctx, g, 1, 0, g.w-2, h, col)
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.at(0, 0, g.w, h), bg)
	}
}

// DrawMenuFrame is a menu box: white, outlined in blue (1.3) or black
// (3.1).
func (e amigaEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bar))
	var k rpInk
	amBox(&k, g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.menuLine)
}

// amShortcut is the key an Amiga menu shows after its command glyph for a
// toolkit shortcut such as "Ctrl+O" (one key with one command modifier).
func amShortcut(s string) (string, bool) {
	i := strings.LastIndexAny(s, "+-")
	if i <= 0 || i >= len(s)-1 {
		return "", false
	}
	mod, key := s[:i], s[i+1:]
	ok := false
	for _, m := range [...]string{"ctrl", "control", "cmd", "command", "meta", "super", "alt"} {
		if strings.EqualFold(mod, m) {
			ok = true
			break
		}
	}
	if !ok || utf8.RuneCountInString(key) != 1 {
		return "", false
	}
	if k := key[0]; k >= 'a' && k <= 'z' {
		const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		key = upper[k-'a' : k-'a'+1]
	}
	return key, true
}

// DrawMenuItem: the check mark at the left, the label, and at the right the
// Amiga key and the command letter, or "»" for a submenu. The hot item is
// complemented; separators are 3.x's bars (1.3: a dotted line); disabled
// items are ghosted in the menu colour.
func (e amigaEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := amColors(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := paintengine2d.XYWH(b.Min.X-ch.PadL+2*u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-4*u, b.Dy())
	fg := rpGridAt(ctx, full, u)
	if fg.w < 4 || fg.h < amRow {
		return
	}
	if row.Separator {
		y := (fg.h - amRow) / 2
		var k rpInk
		if c.wb13 {
			for x := 0; x < fg.w; x += 2 {
				k.cells(fg, x, y, 1, amRow)
			}
			k.fill(ctx, c.menuLine)
			return
		}
		k.cells(fg, 2, y, fg.w-4, amRow)
		k.fill(ctx, c.shadow)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	col, bg := c.barText, c.bar
	if hot {
		ctx.DrawRect(fg.rect(), paintengine2d.Fill(c.menuHi))
		col, bg = c.menuHiText, c.menuHi
	}
	f := l.MonoFont()
	cc := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked:
		rows := min((cc.h-2)/amRow, (cc.w-2)*2/3, 5)
		if rows >= 3 {
			m := amTall(amTick(rows, 2))
			var k rpInk
			m.emit(&k, cc, (cc.w-m.w)/2, (cc.h-m.h)/2)
			k.fill(ctx, col)
		}
	case row.Icon != IconNone:
		side := min(ch.CheckCol()-l.S(4), b.Dy()-l.S(4))
		if side > 4 {
			ib := paintengine2d.XYWH(b.Min.X+(ch.CheckCol()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
			l.drawToolIcon(ctx, ib, row.Icon, col)
		}
	}
	right := ch.LabelMaxX(b.Max.X)
	labelRight := right
	if row.Submenu {
		// NewLook menus mark a submenu with Topaz's "»", flush right.
		sw := f.Advance("»")
		amText(l, ctx, "»", paintengine2d.XYWH(right-sw, b.Min.Y, sw+u, b.Dy()), full, col, AlignStart)
		right -= sw + 4*u
		labelRight = right
	}
	if row.Shortcut != "" {
		if key, ok := amShortcut(row.Shortcut); ok {
			kw := f.Advance(key)
			gw := float32(amG.key.w) * u
			x := snap(right - kw - gw - 3*u)
			kg := rpGridAt(ctx, paintengine2d.XYWH(x, b.Min.Y, gw, b.Dy()), u)
			var k rpInk
			amG.key.emit(&k, kg, 0, (kg.h-amG.key.h)/2)
			k.fill(ctx, col)
			amText(l, ctx, key, paintengine2d.XYWH(x+gw+3*u, b.Min.Y, kw+u, b.Dy()), full, col, AlignStart)
			labelRight = x - ch.AccelGap
		} else {
			tw := f.Advance(row.Shortcut)
			amText(l, ctx, row.Shortcut, paintengine2d.XYWH(right-tw, b.Min.Y, tw+u, b.Dy()), full, col, AlignStart)
			labelRight = right - tw - ch.AccelGap
		}
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	amText(l, ctx, row.Label, paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()), full, col, AlignStart)
	if st.Disabled() {
		amGhostOver(l, ctx, fg.rect(), bg)
	}
}

// DrawListRow is a listview entry: pen 1 text; the selected entry is
// highlighted in FILLPEN blue with FILLTEXT black (1.3: complemented to
// orange with black). The Amiga had no view focus, so the selection keeps
// its colour in unfocused views and inactive windows.
func (e amigaEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if st.Checked() {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.fill))
		fg = c.fillText
	}
	amText(l, ctx, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), b, fg, AlignStart)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawTreeRow: the Amiga had no tree gadget; this is a listview row with
// the chevron disclosure mark, indented by depth.
func (e amigaEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if st.Checked() {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.fill))
		fg = c.fillText
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()).Intersect(b), expanded, fg)
	}
	lx := x + indent + l.S(4)
	amText(l, ctx, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(2), b.Dy()), b, fg, AlignStart)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawTableCell: a listview column entry, highlighted like a row.
func (e amigaEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if st.Checked() {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.fill))
		fg = c.fillText
	}
	f := face
	if f == nil {
		f = l.MonoFont()
	}
	pad := tableCellPad(b.Dx(), f.Advance(label))
	lb := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	ctx.Save()
	ctx.ClipRect(b)
	l.drawFittedText(ctx, f, label, lb, fg, align, 0)
	ctx.Restore()
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawTableHeader: column titles are buttons (as in ReAction's
// listbrowser), recessed while pressed; the sort column shows a chevron.
func (e amigaEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 3*amRow {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	aw := 0
	if sorted && g.w >= 24 {
		dir := DirDown
		if asc {
			dir = DirUp
		}
		aw = 8
		var k rpInk
		amArrowIn(&k, g, g.w-4-aw, amRow, aw, g.h-2*amRow, dir, 3)
		k.fill(ctx, fg)
	}
	amText(l, ctx, label, g.at(5, amRow, g.w-10-aw, g.h-2*amRow), b, fg, AlignStart)
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawToolBar: the Amiga had no tool bars (applications drew button
// strips); this is a raised strip, a shine row over it and a shadow row
// under it (1.3: a white line under the blue).
func (e amigaEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	if g.h < 3*amRow {
		return
	}
	var hi, lo rpInk
	if c.wb13 {
		hi.cells(g, 0, g.h-amRow, g.w, amRow)
		hi.fill(ctx, c.line)
		return
	}
	hi.cells(g, 0, 0, g.w, amRow)
	lo.cells(g, 0, g.h-amRow, g.w, amRow)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
}

// DrawToolButton: flat until the pointer arrives, when it rises into a
// gadget (1.3: its outline shows); pressed and latched tools recess into
// FILLPEN blue (1.3: complement).
func (e amigaEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := amColors(l)
	u := rpU(l)
	fg := e.Face(l, ctx, b, RoleTool, st)
	pressed := !st.Disabled() && (st.Pressed() || (st.Toggle() && st.Checked()))
	g := rpGridAt(ctx, b, u)
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		ctx.Save()
		ctx.ClipRect(b)
		l.drawToolIcon(ctx, ib, icon, fg)
		ctx.Restore()
		x = ib.Max.X + iconGap
	}
	if label != "" {
		amText(l, ctx, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-pad-x, b.Dy()), b, fg, AlignStart)
	}
	if st.Focused() && !st.Disabled() && g.w > 8 {
		amFocus(ctx, g, 3, amRow+1, g.w-6, g.h-2*amRow-2, c.focusOn(c.faceOf(pressed)))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawStatusBar: the Amiga had none; this is a strip of GadTools text
// displays — each part in a thin recessed box (1.3: parts divided by white
// lines under a white rule) — ending in the window's size gadget.
func (e amigaEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 24 || g.h < 4*amRow {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var hi, lo, fillk rpInk
	sw := 18
	if c.wb13 {
		sw = 16
	}
	sx := g.w - sw
	if c.wb13 {
		hi.cells(g, 0, 0, g.w, amRow)
		hi.cells(g, sx, amRow, am13Side, g.h-amRow)
	}
	slot := 0
	if len(parts) > 0 {
		slot = sx / len(parts)
	}
	for i := range parts {
		x := i * slot
		w := slot - 2
		if c.wb13 {
			if i > 0 {
				hi.cells(g, x, amRow, am13Side, g.h-amRow)
			}
		} else if w > 4 {
			amBevel(&lo, &hi, g, x, amRow, w, g.h-2*amRow, 1)
		}
	}
	// The size gadget.
	rows := (g.h - amRow) / amRow
	if c.wb13 {
		m := amG.size13
		oy := amRow + (rows*amRow-m.h)/2
		if m.h <= g.h-amRow {
			m.emit(&hi, g, sx+(sw-m.w)/2+1, oy)
		}
	} else {
		amBevel(&hi, &lo, g, sx, 0, sw, g.h, 1)
		m := amG.sizeInk
		oy := (g.h - m.h) / 2
		if m.h <= g.h-2*amRow {
			amG.sizeInk.emit(&lo, g, sx+(sw-m.w)/2, oy)
			amG.sizeFill.emit(&fillk, g, sx+(sw-m.w)/2, oy)
		}
	}
	if c.wb13 {
		hi.fill(ctx, c.line)
	} else {
		fillk.fill(ctx, c.shine)
		hi.fill(ctx, c.shine)
		lo.fill(ctx, c.shadow)
	}
	for i, s := range parts {
		x := i * slot
		tb := g.at(x+5, amRow, slot-10, g.h-2*amRow)
		if c.wb13 {
			tb = g.at(x+6, amRow, slot-10, g.h-amRow)
		}
		amText(l, ctx, s, tb, b, c.text, AlignStart)
	}
}

// DrawTitleBar is the screen title bar — the strip the Workbench shows
// its title and free memory in: white with blue (1.3) or black (3.1) text,
// 3.1's trim line under it, and the screen's depth gadgets at the right.
func (e amigaEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 3*amRow {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bar))
	h := g.h
	var hi, lo, grey, dark rpInk
	if !c.wb13 {
		h -= amRow
		lo.cells(g, 0, h, g.w, amRow)
	}
	f := l.MonoFont()
	x := l.metrics.Pad
	used := x
	if title != "" {
		used += f.Advance(title) + l.S(16)
	}
	if subtitle != "" {
		used += f.Advance(subtitle)
	}
	gw := 24
	if c.wb13 {
		gw = 48
	}
	right := b.Max.X
	if float32(g.w-gw)*u > used+l.S(8) && h >= 8*amRow {
		oy := (h/amRow - 8) / 2 * amRow
		if c.wb13 {
			for i := 0; i < 2; i++ {
				gx := g.w - 48 + 24*i
				hi.cells(g, gx, 0, 2, h)
				if i == 0 {
					amG.backBlue.emit(&hi, g, gx+4, oy)
					amG.backDark.emit(&dark, g, gx+4, oy)
				} else {
					amG.frontDark.emit(&dark, g, gx+4, oy)
					amG.frontBlue.emit(&hi, g, gx+4, oy)
				}
			}
		} else {
			gx := g.w - gw
			grey.cells(g, gx+1, amRow, gw-2, h-2*amRow)
			amBevel(&hi, &lo, g, gx, 0, gw, h, 1)
			oy = (h/amRow - 7) / 2 * amRow
			amG.depthInk.emit(&lo, g, gx+4, oy)
			amG.depthBack.emit(&grey, g, gx+4, oy)
			amG.depthFront.emit(&hi, g, gx+4, oy)
		}
		right = g.x + float32(g.w-gw)*u
	}
	grey.fill(ctx, c.win)
	dark.fill(ctx, c.dark)
	if c.wb13 {
		hi.fill(ctx, c.barText)
	} else {
		hi.fill(ctx, c.shine)
		lo.fill(ctx, c.shadow)
	}
	band := g.at(0, 0, g.w, h)
	if title != "" {
		amText(l, ctx, title, paintengine2d.XYWH(b.Min.X+x, band.Min.Y, right-b.Min.X-x, band.Dy()), band, c.barText, AlignStart)
		x += f.Advance(title) + l.S(16)
	}
	if subtitle != "" && b.Min.X+x < right {
		amText(l, ctx, subtitle, paintengine2d.XYWH(b.Min.X+x, band.Min.Y, right-b.Min.X-x, band.Dy()), band, c.barText, AlignStart)
	}
}

// DrawAccordionHeader: the Amiga had no accordion; this is a full-width
// button with the chevron disclosure mark and the title.
func (e amigaEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := amColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 3*amRow {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	fg := c.gadget(ctx, g, 0, 0, g.w, g.h, pressed)
	if ew := min(12, g.w-8); ew > 0 {
		e.Expander(l, ctx, g.at(4, amRow, ew, g.h-2*amRow), expanded, fg)
	}
	amText(l, ctx, title, g.at(20, amRow, g.w-24, g.h-2*amRow), b, fg, AlignStart)
	if st.Focused() && !st.Disabled() {
		amFocus(ctx, g, 3, amRow+1, g.w-6, g.h-2*amRow-2, c.focusOn(c.faceOf(pressed)))
	}
	if st.Disabled() {
		amGhostOver(l, ctx, g.rect(), c.ghost)
	}
}

// DrawSeparator: GadTools had no separator gadget; this is the engraved
// line of 3.x's bevel boxes (shadow over shine), and on 1.3 a white line.
func (e amigaEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	var sh, hi rpInk
	if vertical {
		if g.w < 2 || g.h < 3*amRow {
			return
		}
		if c.wb13 {
			hi.cells(g, (g.w-am13Side)/2, amRow, am13Side, g.h-2*amRow)
		} else {
			amEtched(&sh, &hi, g, g.w/2-1, amRow, g.h-2*amRow, true)
		}
	} else {
		if g.h < 2*amRow {
			return
		}
		if c.wb13 {
			hi.cells(g, 0, (g.h-amRow)/2, g.w, amRow)
		} else {
			amEtched(&sh, &hi, g, 0, g.h/2-amRow, g.w, false)
		}
	}
	if c.wb13 {
		hi.fill(ctx, c.line)
		return
	}
	sh.fill(ctx, c.shadow)
	hi.fill(ctx, c.shine)
}

// DrawSplitter: Intuition had no split panes; this is an engraved line in
// the window colour that becomes a raised FILLPEN bar under the pointer or
// while held (1.3: a white line that turns orange).
func (e amigaEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	if (vertical && g.w < 2) || (!vertical && g.h < 2*amRow) {
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	var sh, hi rpInk
	switch {
	case c.wb13:
		col := c.line
		if hot {
			col = c.comp(c.win)
		}
		if vertical {
			hi.cells(g, (g.w-am13Side)/2, 0, am13Side, g.h)
		} else {
			hi.cells(g, 0, (g.h-amRow)/2, g.w, amRow)
		}
		hi.fill(ctx, col)
		return
	case hot && vertical && g.w >= 4:
		x := (g.w - 4) / 2
		ctx.DrawRect(g.at(x, 0, 4, g.h), paintengine2d.Fill(c.fill))
		hi.cells(g, x, 0, 1, g.h)
		sh.cells(g, x+3, 0, 1, g.h)
	case hot && !vertical && g.h >= 3*amRow:
		y := (g.h - 3*amRow) / 2
		ctx.DrawRect(g.at(0, y, g.w, 3*amRow), paintengine2d.Fill(c.fill))
		hi.cells(g, 0, y, g.w, amRow)
		sh.cells(g, 0, y+2*amRow, g.w, amRow)
	case vertical:
		amEtched(&sh, &hi, g, g.w/2-1, 0, g.h, true)
	default:
		amEtched(&sh, &hi, g, 0, g.h/2-amRow, g.w, false)
	}
	sh.fill(ctx, c.shadow)
	hi.fill(ctx, c.shine)
}

// DrawTooltip: the Amiga had no tooltips (3.x applications showed help in
// the screen title bar); this is a small box in the menus' colours.
func (e amigaEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := amColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bar))
	var k rpInk
	amBox(&k, g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.menuLine)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	amText(l, ctx, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), b, c.barText, AlignStart)
}

// DrawOverlay: Intuition never dimmed the screen behind a requester.
func (amigaEngine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- text in string gadgets ------------------------------------------------------

// amTextOpts colours the text of a string gadget.
type amTextOpts struct {
	text, muted  paintengine2d.Color // text; placeholder
	sel, selText paintengine2d.Color // selection and its text
	cur, curText paintengine2d.Color // the block cursor and its character
}

// amFace is the face a string gadget sets its text in: the one the widget
// lays its caret out with, else Topaz's stand-in.
func amFace(l *Classic, face *Font) *Font {
	if face != nil {
		return face
	}
	return l.MonoFont()
}

// amSnap rounds v, in a context whose origin sits at o on the device, to
// the device pixel grid: a control laid out between pixels still gets
// whole-pixel boxes.
func amSnap(v, o float32) float32 { return snap(v+o) - o }

// amCursor paints Intuition's string cursor for position i of text drawn
// at (ox, y): the character cell filled in cur with its character redrawn
// in curText, or a space-wide block past the end of the text.
func amCursor(ctx *paintengine2d.Context, f *Font, text string, i int, ox, y, u float32, o amTextOpts) {
	n := utf8.RuneCountInString(text)
	if i < 0 {
		i = 0
	}
	if i > n {
		i = n
	}
	dx, _ := rpOrigin(ctx)
	x0 := amSnap(ox+f.CaretX(text, i), dx)
	x1 := x0 + snap(f.Advance(" "))
	if i < n {
		x1 = amSnap(ox+f.CaretX(text, i+1), dx)
	}
	if x1-x0 < 2*u {
		x1 = x0 + 2*u
	}
	box := paintengine2d.XYWH(x0, y, x1-x0, snap(f.Height()))
	ctx.DrawRect(box, paintengine2d.Fill(o.cur))
	if i < n {
		ctx.Save()
		ctx.ClipRect(box)
		f.Draw(ctx, text, paintengine2d.Pt(ox, y), o.curText)
		ctx.Restore()
	}
}

// amFieldText paints the text of a string gadget inside inner: the
// selection, the text and the block cursor, on whole pixels.
func amFieldText(l *Classic, ctx *paintengine2d.Context, inner paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font, o amTextOpts) {
	if inner.Empty() {
		return
	}
	u := rpU(l)
	f := amFace(l, face)
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(inner)
	show, col := text, o.text
	if text == "" && placeholder != "" && !st.Focused() {
		show, col = placeholder, o.muted
	}
	dx, dy := rpOrigin(ctx)
	ty := amSnap(inner.Min.Y+(inner.Dy()-f.Height())*0.5, dy)
	ox := inner.Min.X - scrollX
	var selBox paintengine2d.Rect
	if selA != selB && text != "" && show == text {
		if selA > selB {
			selA, selB = selB, selA
		}
		x0, x1 := amSnap(ox+f.CaretX(text, selA), dx), amSnap(ox+f.CaretX(text, selB), dx)
		selBox = paintengine2d.XYWH(x0, ty, x1-x0, snap(f.Height()))
		ctx.DrawRect(selBox, paintengine2d.Fill(o.sel))
	}
	f.Draw(ctx, show, paintengine2d.Pt(ox, ty), col)
	if !selBox.Empty() {
		ctx.Save()
		ctx.ClipRect(selBox)
		f.Draw(ctx, show, paintengine2d.Pt(ox, ty), o.selText)
		ctx.Restore()
	}
	if st.Focused() && blink && show == text {
		amCursor(ctx, f, text, caret, ox, ty, u, o)
	}
}

// amAreaText paints the lines of a multi-line string gadget inside inner:
// selection per line, text, the block cursor.
func amAreaText(l *Classic, ctx *paintengine2d.Context, inner paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font, o amTextOpts) {
	if inner.Empty() {
		return
	}
	u := rpU(l)
	f := amFace(l, face)
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(inner)
	lh := f.Height() + 2
	empty := len(lines) == 0 || (len(lines) == 1 && lines[0].Text == "" && lines[0].End <= lines[0].Start)
	if empty && placeholder != "" && !st.Focused() {
		f.Draw(ctx, placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), o.muted)
		return
	}
	if selA > selB {
		selA, selB = selB, selA
	}
	dx, dy := rpOrigin(ctx)
	for i, line := range lines {
		y := amSnap(inner.Min.Y+float32(i)*lh-scrollY, dy)
		if y+lh < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		ox := inner.Min.X - scrollX
		if line.Text != "" {
			f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), o.text)
		}
		if selA != selB && selB > line.Start && selA < line.End {
			n := utf8.RuneCountInString(line.Text)
			a, z := max(selA, line.Start)-line.Start, min(selB, line.End)-line.Start
			x0, x1 := amSnap(ox+f.CaretX(line.Text, a), dx), amSnap(ox+f.CaretX(line.Text, z), dx)
			if z > n || selB > line.Start+n {
				x1 = inner.Max.X
			}
			if x1 > x0 {
				box := paintengine2d.XYWH(x0, y, x1-x0, snap(f.Height()))
				ctx.DrawRect(box, paintengine2d.Fill(o.sel))
				if line.Text != "" {
					ctx.Save()
					ctx.ClipRect(box)
					f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), o.selText)
					ctx.Restore()
				}
			}
		}
		if st.Focused() && blink && caret >= line.Start && caret <= line.End {
			onThis := caret < line.End || i == len(lines)-1
			if caret == line.End && i < len(lines)-1 && lines[i+1].Start == line.End {
				onThis = false
			}
			if onThis {
				amCursor(ctx, f, line.Text, max(caret-line.Start, 0), ox, y, u, o)
			}
		}
	}
}

// ---- packs --------------------------------------------------------------------

func amigaPack(name, label string, year int, summary string, fam ThemeName, pal Palette, pens [4]string, wb float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "amiga",
		Bevel:   BevelClassic3D,
		Family:  fam,
		Palette: pal,
		Extra:   map[string]paintengine2d.Color{},
		Params:  map[string]float32{"wb": wb},
	}
	for i, v := range pens {
		tok.Extra["pen"+itoa(i)] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.15), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Amiga", Summary: summary,
		Era: "Commodore Amiga", Palette: fam, Tokens: tok,
	}
}

// amPalette is a Workbench's shared palette, from its four pens: win (pen
// 0) under text (pen 1), line the outlines and dividers, sel the highlight
// with selText on it, bar the white of bars and menus, dark the black of
// the menu complement and the bevel shadow, alert the colour of warnings
// and the focus mark. muted, the placeholder colour widgets use, is the
// one tone that is not a pen: pen 1 half dithered into pen 0.
func amPalette(win, text, line, sel, selText, bar, dark, muted, alert string) Palette {
	w, t, s, d := hexColor(win), hexColor(text), hexColor(sel), hexColor(dark)
	return Palette{
		Background: w, Surface: w, SurfaceAlt: w,
		Border: hexColor(line), Divider: hexColor(line),
		Text: t, TextMuted: hexColor(muted), TextOnAccent: hexColor(selText),
		Accent: s, AccentHover: s, AccentPress: s,
		Field: w, FieldBorder: hexColor(line),
		Focus: hexColor(alert), Selection: s,
		Track: w, Thumb: s,
		Highlight: hexColor(bar), Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: d, MenuHoverBorder: d, MenuGutter: hexColor(bar),
		Danger: hexColor(alert), Success: t, Warning: hexColor(alert),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: hexColor(bar), BevelDark: d,
	}
}

func amigaPacks() []ThemePack {
	return []ThemePack{
		amigaPack("amiga13", "Workbench 1.3", 1987,
			"Workbench 1.3: blue and orange on four colours, white title bars and borders, highlights by complement.",
			ThemeDark,
			amPalette("#0055aa", "#ffffff", "#ffffff", "#ff8800", "#000022", "#ffffff", "#000022", "#aaccee", "#ff8800"),
			amPens13, 13),
		amigaPack("amiga31", "Workbench 3.1", 1994,
			"Workbench 2.0 to 3.1: Intuition's grey 3D look, blue active borders, ridged string and cycle gadgets.",
			ThemeLight,
			amPalette("#aaaaaa", "#000000", "#000000", "#6688bb", "#000000", "#ffffff", "#000000", "#555555", "#000000"),
			amPens31, 31),
	}
}
