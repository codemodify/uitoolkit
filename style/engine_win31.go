package style

import "github.com/codemodify/paintengine2d"

// win31Engine paints Windows 3.1 (1992) on a 16-colour VGA screen, whose
// 16-pixel System font cell matches the toolkit's UI font, so most sizes
// are the original pixel counts:
//
//   - Push buttons: a black outline with its four corner pixels left out,
//     a two-pixel white bevel on the top and left and a two-pixel dark grey
//     one on the bottom and right round the light grey face. The default
//     button has a second black rectangle inside the outline. Pressed, the
//     bevel becomes one dark line on the top and left and the label moves a
//     pixel down and right; disabled labels are dithered.
//   - Check boxes are 13-pixel squares crossed corner to corner; radio
//     buttons 13-pixel circles with a 7-pixel dot; both thicken their
//     outline to two pixels while pressed.
//   - Scroll bars are 17 pixels with black lines: arrow buttons and a
//     square thumb that never changes size on the plain light grey shaft.
//     They, the drop-down button and the caption's arrows have the small
//     system buttons' bevel: one white pixel on the top and left, two dark
//     grey on the bottom and right.
//   - A drop-down list's button touches its field; an editable combo's
//     stands eight pixels off its edit box, in a frame of its own.
//   - Windows and dialogs: the navy caption with its bold white title, the
//     control-menu box at the left (double-click closed a window — there
//     was no close button), minimize and maximize arrows at the right, and
//     the modal dialog's thick frame in the caption colour.
//   - Menus are white with a navy highlight and no shadow; dialogs are
//     white; edit fields and lists have a one-pixel black border; group
//     boxes are black rectangles.
//   - Focus is the dotted rectangle, drawn with XOR on the 16-colour
//     palette: dark grey dots on a button, black on white, yellow on navy.
//
// The Windows Default and Hot Dog Stand schemes are packs; Hot Dog Stand
// moved here from the win95 engine, which had drawn the 3.1 scheme with
// Windows 95 shapes and a yellow button face.
//
// Pack data: Palette.Background is the window colour (dialogs, fields,
// lists), SurfaceAlt the button face, Selection the highlight. Extra:
//
//	hi, shadow, frame          button highlight and shadow, window frame
//	caption, captionText       active title bar
//	captionOff, captionOffText inactive title bar
//	border                     the active border (sizing frames)
//	menu, menuText             menu bar and menus
//	scroll                     scroll bar shaft
//	gray                       disabled ("gray") text
//	info                       tooltip fill (tooltips postdate 3.1)
type win31Engine struct{ BaseEngine }

func init() {
	RegisterEngine(win31Engine{})
	for _, p := range win31Packs() {
		RegisterPack(p)
	}
}

func (win31Engine) ID() string { return "win31" }

// DefaultMetrics: VGA Windows 3.1 — 17px scroll bars, 13px check boxes and
// radios, 20px captions and 18px menu bars, grown where the toolkit's
// font needs the room.
func (win31Engine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Square:    true, BevelDepth: 2,
		ControlH: 28, FieldH: 24, ComboH: 24,
		Checkbox: 13, Radio: 13,
		MenuItemH: 22, MenuBarH: 22, TabH: 26, RowH: 20,
		TitleBar: 24, HeaderH: 22, ProgressH: 18, SliderH: 24, Thumb: 11,
		Scroll: 17, Pad: 10, FieldPad: 3, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 23, SpinnerW: 17, SwitchW: 36, SwitchH: 18,
	}
}

// ---- colours ------------------------------------------------------------------

// w31 is the resolved system-colour set of a look.
type w31 struct {
	win, text, face, hi, shadow, frame, btnText    paintengine2d.Color
	sel, selText, gray, muted                      paintengine2d.Color
	caption, captionText, captionOff, captionOffT  paintengine2d.Color
	border, menu, menuText, scroll, info, infoText paintengine2d.Color
}

type w31Key struct{}

func w31colors(l *Classic) *w31 {
	return l.Memo(w31Key{}, func() any { return w31build(l) }).(*w31)
}

func w31build(l *Classic) *w31 {
	p := l.palette
	c := &w31{
		win:     p.Background,
		text:    p.Text,
		face:    p.SurfaceAlt,
		sel:     p.Selection,
		selText: p.TextOnAccent,
		muted:   p.TextMuted,
	}
	x := func(k, def string) paintengine2d.Color { return l.X(k, Hex(def)) }
	c.hi = x("hi", "#ffffff")
	c.shadow = x("shadow", "#808080")
	c.frame = x("frame", "#000000")
	c.btnText = x("btnText", "#000000")
	c.caption = x("caption", "#000080")
	c.captionText = x("captionText", "#ffffff")
	c.captionOff = x("captionOff", "#ffffff")
	c.captionOffT = x("captionOffText", "#000000")
	c.border = x("border", "#c0c0c0")
	c.menu = x("menu", "#ffffff")
	c.menuText = x("menuText", "#000000")
	c.scroll = x("scroll", "#c0c0c0")
	c.gray = x("gray", "#c0c0c0")
	c.info = x("info", "#ffffc0")
	c.infoText = ReadableOn(c.info, 4.5, c.text, paintengine2d.RGB(0, 0, 0))
	return c
}

// w31vga is the 16-colour palette of the Windows 3.1 VGA driver, in its
// index order: XOR-drawing turns index i into 15-i.
var w31vga = [16]paintengine2d.Color{
	Hex("#000000"), Hex("#800000"), Hex("#008000"), Hex("#808000"),
	Hex("#000080"), Hex("#800080"), Hex("#008080"), Hex("#c0c0c0"),
	Hex("#808080"), Hex("#ff0000"), Hex("#00ff00"), Hex("#ffff00"),
	Hex("#0000ff"), Hex("#ff00ff"), Hex("#00ffff"), Hex("#ffffff"),
}

// w31xor is the colour XOR-drawn focus dots take over c: the nearest VGA
// colour's opposite index (black on white, dark grey on light grey,
// yellow on navy).
func w31xor(c paintengine2d.Color) paintengine2d.Color {
	best, bd := 0, float32(9)
	for i, v := range w31vga {
		dr, dg, db := v.R-c.R, v.G-c.G, v.B-c.B
		if d := dr*dr + dg*dg + db*db; d < bd {
			best, bd = i, d
		}
	}
	return w31vga[15-best]
}

// ---- drawing helpers -------------------------------------------------------------

// raised adds the Windows 3.1 button bevel inside the w × h cells at
// (x, y): two cells of highlight on the top and left, two of shadow on the
// bottom and right, split on the diagonal at the corners.
func w31raised(hi, lo *rpInk, g rpGrid, x, y, w, h int) {
	n := 2
	if w < 8 || h < 8 {
		n = 1
	}
	rpEdge(hi, lo, g, x, y, w, h, n)
}

// w31small adds the bevel of the small system buttons (scroll arrows and
// thumb, the drop-down button, the caption's arrows) inside the w × h
// cells at (x, y): one cell of highlight on the top and left, two of
// shadow on the bottom and right.
func w31small(hi, lo *rpInk, g rpGrid, x, y, w, h int) {
	rpEdge(hi, lo, g, x, y, w, h, 1)
	if w < 6 || h < 6 {
		return
	}
	lo.cells(g, x+1, y+h-2, w-2, 1)
	lo.cells(g, x+w-2, y+1, 1, h-3)
}

// w31comboGap is the air between an editable combo's edit box and its
// button (CBS_DROPDOWN); a drop-down list's button touches its field.
const w31comboGap = 8

// w31combo splits a combo box's cells: the button, as wide as a scroll
// bar with its lines, starts at bx; the field (the edit box, when split)
// is fw cells wide.
func w31combo(g rpGrid, editable bool) (bx, fw int, split bool) {
	bx = g.w - min(g.h, 17)
	split = editable && bx-w31comboGap >= 12
	fw = bx
	if split {
		fw -= w31comboGap
	}
	return bx, fw, split
}

// button paints a push button's body — outline without corner pixels
// (and the default button's second rectangle), face and bevel — in g, and
// returns the face rect cells (inside outline and bevel) for the label.
func (c *w31) button(ctx *paintengine2d.Context, g rpGrid, pressed, def bool) {
	if g.w < 4 || g.h < 4 {
		return
	}
	var black, hi, lo rpInk
	rpShape{w: g.w, h: g.h, cut: 1}.frame(&black, g, 0, 0)
	in := 1
	if def {
		black.frame(g, 1, 1, g.w-2, g.h-2, 1)
		in = 2
	}
	ctx.DrawRect(g.at(in, in, g.w-2*in, g.h-2*in), paintengine2d.Fill(c.face))
	if pressed {
		lo.cells(g, in, in, g.w-2*in, 1)
		lo.cells(g, in, in+1, 1, g.h-2*in-1)
	} else {
		w31raised(&hi, &lo, g, in, in, g.w-2*in, g.h-2*in)
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	black.fill(ctx, c.frame)
}

// label draws a control label; disabled labels on the button face are
// dithered (Windows did so when the gray text colour was the face's own),
// elsewhere drawn in the gray text colour.
func (c *w31) label(l *Classic, ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect, col paintengine2d.Color, align Align, disabled bool, bg paintengine2d.Color) {
	if text == "" || b.Empty() {
		return
	}
	dither := disabled && nearColor(c.gray, bg)
	if disabled && !dither {
		col = c.gray
	}
	l.drawFittedText(ctx, f, text, b, col, align, 0)
	if dither {
		rpPatFill(ctx, rpTextBox(f, text, b, align).Intersect(b), rpGray, bg, rpU(l))
	}
}

// focus paints the dotted focus rectangle over the w × h cells at (x, y),
// in the XOR colour of what lies under it.
func w31focus(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, under paintengine2d.Color) {
	var k rpInk
	k.dotted(g, x, y, w, h)
	k.fill(ctx, w31xor(under))
}

// w31arrow is the scroll arrow pointing up in 7 × 7 cells: a triangle of
// 1, 3, 5 and 7 cells over a 3-cell stem three rows long.
func w31arrow() rpMask {
	m := rpNewMask(7, 7)
	for i := 0; i < 4; i++ {
		m.span(3-i, 4+i, i)
	}
	m.rect(2, 4, 3, 3)
	return m
}

// w31tri is a solid triangle w cells wide pointing dir (w-2 … 1 rows).
func w31tri(w int, dir Direction) rpMask { return s7tri(w, dir) }

// ---- parts --------------------------------------------------------------------

func (e win31Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.btnText
	if g.w < 3 || g.h < 3 {
		return fg
	}
	pressed := st.Pressed() && !st.Disabled()
	switch role {
	case RoleButton, RoleCombo, RoleThumb:
		c.button(ctx, g, pressed, st.Primary() && role == RoleButton)
		return fg
	case RoleTool:
		switch {
		case st.Disabled():
		case pressed || (st.Toggle() && st.Checked()) || st.Hovered():
			c.button(ctx, g, pressed || st.Checked(), false)
		}
		return fg
	case RoleField, RoleCheck:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		var k rpInk
		k.frame(g, 0, 0, g.w, g.h, 1)
		k.fill(ctx, c.frame)
		return c.text
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.sel))
			return c.selText
		}
		return c.text
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.sel))
			return c.selText
		}
		return c.menuText
	case RoleTrack:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.scroll))
		return fg
	case RoleTab:
		return fg
	case RolePanel, RoleBar, RoleSplitter:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
		return fg
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	return c.text
}

// CheckIndicator is the 3.1 check box: a black square (two pixels thick
// while pressed) round the window colour, crossed corner to corner.
func (e win31Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5))
	if n < 6 {
		return
	}
	g = g.centered(n, n)
	t := 1
	if st.Pressed() && !st.Disabled() {
		t = 2
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k, x rpInk
	k.frame(g, 0, 0, n, n, t)
	if checked {
		m := rpNewMask(n, n)
		m.line(1, 1, n-2, n-2)
		m.line(n-2, 1, 1, n-2)
		m.emit(&x, g, 0, 0)
	}
	// Disabled, only the label greys.
	x.fill(ctx, c.text)
	k.fill(ctx, c.frame)
}

// RadioIndicator is the 3.1 radio button: a 13-pixel black circle (two
// pixels while pressed) with a 7-pixel dot.
func (e win31Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5))
	if n < 6 {
		return
	}
	g = g.centered(n, n)
	disc := rpShape{w: n, h: n, r: float32(n) * 0.5}
	var face, ring, dot rpInk
	disc.fill(&face, g, 0, 0)
	if st.Pressed() && !st.Disabled() {
		disc.ring(&ring, g, 0, 0, 2)
	} else {
		disc.frame(&ring, g, 0, 0)
	}
	if selected {
		d := n/2 + 1
		if (n-d)%2 != 0 {
			d--
		}
		rpShape{w: d, h: d, r: float32(d) * 0.5}.fill(&dot, g, (n-d)/2, (n-d)/2)
	}
	face.fill(ctx, c.win)
	dot.fill(ctx, c.text)
	ring.fill(ctx, c.frame)
}

// Arrow is a solid whole-pixel triangle.
func (e win31Engine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 3 {
		return
	}
	m := w31tri(min(n*2/3, 9), dir)
	var k rpInk
	m.emit(&k, g, (g.w-m.w)/2, (g.h-m.h)/2)
	k.fill(ctx, col)
}

// Expander is a +/- box in a one-pixel black frame.
func (e win31Engine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h, 11)
	if n%2 == 0 {
		n--
	}
	if n < 5 {
		return
	}
	bg := g.centered(n, n)
	ctx.DrawRect(bg.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.frame(bg, 0, 0, n, n, 1)
	k.cells(bg, 2, n/2, n-4, 1)
	if !expanded {
		k.cells(bg, n/2, 2, 1, n-4)
	}
	k.fill(ctx, c.frame)
}

// MenuHighlight is the navy highlight bar.
func (e win31Engine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(w31colors(l).sel))
}

func (e win31Engine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := w31colors(l)
	if hot {
		return c.selText
	}
	return c.menuText
}

// FieldFocusRing: edit fields show the caret only.
func (win31Engine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is the XOR dotted rectangle just inside b (over the window
// colour).
func (e win31Engine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	g := rpGridAt(ctx, b, rpU(l))
	w31focus(ctx, g, 0, 0, g.w, g.h, w31colors(l).win)
}

// ---- scrollbars -------------------------------------------------------------

// ScrollBarStyle: 17px bars with an arrow button at each end and a square
// thumb as long as the bar is thick.
func (win31Engine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsEnds, MinThumb: 17, FixedThumb: 17}
}

func (e win31Engine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, p.Bar, u)
	if g.w < 6 || g.h < 6 {
		return
	}
	active := !st.Disabled && !p.Thumb.Empty()
	along, across := g.h, g.w
	if !vertical {
		along, across = g.w, g.h
	}
	cell := func(a, x, n, m int) paintengine2d.Rect {
		if vertical {
			return g.at(x, a, m, n)
		}
		return g.at(a, x, n, m)
	}
	sub := func(a, x, n, m int) rpGrid { return rpGridAt(ctx, cell(a, x, n, m), u) }
	pos := func(r paintengine2d.Rect) (int, int) {
		if vertical {
			return int((r.Min.Y-g.y)/u + 0.5), int(r.Dy()/u + 0.5)
		}
		return int((r.Min.X-g.x)/u + 0.5), int(r.Dx()/u + 0.5)
	}
	shaft := c.scroll
	if !active {
		shaft = c.win
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(shaft))
	var black, hi, lo rpInk
	black.frame(g, 0, 0, g.w, g.h, 1)
	lo0, hi0 := 1, along-1
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow := w31arrow()
	type end struct {
		r    paintengine2d.Rect
		dir  Direction
		part ScrollPart
	}
	var glyph, etch rpInk
	for i, a := range [2]end{{p.Dec, dec, ScrollDec}, {p.Inc, inc, ScrollInc}} {
		if a.r.Empty() {
			continue
		}
		a0, n := pos(a.r)
		if i == 0 {
			lo0 = a0 + n
			black.add(cell(a0+n-1, 1, 1, across-2))
		} else {
			hi0 = a0
			black.add(cell(a0, 1, 1, across-2))
		}
		fg := sub(a0+1, 1, n-2, across-2)
		ctx.DrawRect(fg.rect(), paintengine2d.Fill(c.face))
		pressed := st.Pressed == a.part && active
		off := 0
		if pressed {
			lo.cells(fg, 0, 0, fg.w, 1)
			lo.cells(fg, 0, 1, 1, fg.h-1)
			off = 1
		} else {
			w31small(&hi, &lo, fg, 0, 0, fg.w, fg.h)
		}
		m := arrow.turn(a.dir)
		ox, oy := (fg.w-m.w)/2+off, (fg.h-m.h)/2+off
		if active {
			m.emit(&glyph, fg, ox, oy)
		} else {
			// Nothing to scroll: etched glyphs, grey over a white copy.
			m.emit(&etch, fg, ox+1, oy+1)
			m.emit(&lo, fg, ox, oy)
		}
	}
	a, boxed := rpBoxAt(p.Track, p.Thumb, vertical, g, across, 1)
	// A held track shows the part being paged inverted, up to the thumb.
	if active && boxed && (st.Pressed == ScrollPageDec || st.Pressed == ScrollPageInc) {
		from, n := lo0, a+1-lo0
		if st.Pressed == ScrollPageInc {
			from, n = a+across-1, hi0-a-across+1
		}
		if n > 0 {
			ctx.DrawRect(cell(from, 1, n, across-2), paintengine2d.Fill(w31xor(c.scroll)))
		}
	}
	if active {
		if boxed {
			tg := sub(a, 0, across, across)
			black.frame(tg, 0, 0, tg.w, tg.h, 1)
			face := tg.inset(1)
			ctx.DrawRect(face.rect(), paintengine2d.Fill(c.face))
			w31small(&hi, &lo, face, 0, 0, face.w, face.h)
		}
	}
	etch.fill(ctx, c.hi)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	glyph.fill(ctx, c.btnText)
	black.fill(ctx, c.frame)
}

func (e win31Engine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------

func (win31Engine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a one-pixel black rectangle; the title breaks its top
// line after a 7-pixel stub, with 3 pixels of air each side.
func (e win31Engine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := w31colors(l)
	u := rpU(l)
	f := l.BoldFont()
	bg := c.win
	if raised {
		bg = c.face
	}
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	top := b.Min.Y
	if title != "" {
		top += f.Height()*0.5 - u
	}
	g := rpGridAt(ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, u)
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.frame)
	if title == "" {
		return
	}
	tx := b.Min.X + 7*u
	tw := min(f.Advance(title)+6*u, b.Dx()-14*u)
	if tw <= 0 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(snap(tx), b.Min.Y, snap(tw), f.Height()), paintengine2d.Fill(bg))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+3*u, b.Min.Y, tw-3*u, f.Height()), c.text, AlignStart, 0)
}

// w31barH is the caption bar's height in cells (18 at VGA, grown with the
// bold font).
func w31barH(l *Classic) int {
	return max(18, int(l.BoldFont().Height()/rpU(l)+0.5))
}

// The modal frame: one black line, a band of the caption colour (four
// cells, three on top), a white line; then the caption between black lines.
const (
	w31band    = 4
	w31bandTop = 3
)

func (win31Engine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	top := 1 + w31bandTop + 1 + 1 + w31barH(l) + 1
	side := float32(1+w31band+1) * u
	return Insets{Top: float32(top) * u, Right: side, Bottom: side, Left: side}
}

// w31caption is the caption bar of a frame in cells of g: x, y, w, h.
func w31caption(l *Classic, g rpGrid) (x, y, w, h int) {
	return 1 + w31band + 1, 1 + w31bandTop + 1 + 1, g.w - 2*(1+w31band+1), w31barH(l)
}

// WindowCloseRect is the control-menu box at the caption's left end:
// double-clicking it closed a Windows 3.1 window.
func (e win31Engine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	x, y, w, h := w31caption(l, g)
	if w < 3*h || g.h < y+h+4 {
		return paintengine2d.Rect{}
	}
	return g.at(x, y, h, h)
}

// PopupShadow: Windows 3.1 menus and dialogs cast no shadow.
func (win31Engine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (win31Engine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// ItemFocus is the XOR dotted rectangle round the current item.
func (e win31Engine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	under := c.win
	if st.Checked() {
		under = c.sel
	}
	w31focus(ctx, g, 0, 0, g.w, g.h, under)
}

// DrawViewFrame: list boxes are the window colour in a one-pixel black
// border.
func (e win31Engine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	e.Face(l, ctx, b, RoleField, st)
}

// StyleHint: Windows puts OK first.
func (win31Engine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// ControlFont: 3.1 drew its dialog font (MS Sans Serif 8) bold, and the
// System font of captions and menus is bold too.
func (win31Engine) ControlFont(l *Classic, role Role) *Font {
	switch role {
	case RoleButton, RoleCheck, RoleMenu, RoleTab, RoleCombo:
		return l.BoldFont()
	}
	return l.body
}

// DrawWindowBackground is the window colour: stock 3.1 dialogs were white.
func (win31Engine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(w31colors(l).win))
}

// DrawTabPane: 3.1 had no tab control; the tabbed dialogs of Word 6 and
// Excel 5 put a raised grey page under the tabs.
func (e win31Engine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var black, hi, lo rpInk
	black.frame(g, 0, 0, g.w, g.h, 1)
	rpEdge(&hi, &lo, g, 1, 1, g.w-2, g.h-2, 1)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	black.fill(ctx, c.frame)
}

// DrawWindowFrame is a Windows 3.1 modal dialog: a black line, a thick band
// in the caption colour, a white line, then the caption — the control-menu
// box, the bold centred title, a maximize button when the window has one.
func (e win31Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	x, y, w, h := w31caption(l, g)
	if w < 8 || g.h < y+h+3 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		return
	}
	band, bar, barText := c.caption, c.caption, c.captionText
	if !st.Active {
		band, bar, barText = c.captionOff, c.captionOff, c.captionOffT
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var bandK, white, black, hi, lo, glyph rpInk
	// The band: four cells at the sides and bottom, three on top.
	bandK.cells(g, 1, 1, g.w-2, w31bandTop)
	bandK.cells(g, 1, g.h-1-w31band, g.w-2, w31band)
	bandK.cells(g, 1, 1+w31bandTop, w31band, g.h-2-w31bandTop-w31band)
	bandK.cells(g, g.w-1-w31band, 1+w31bandTop, w31band, g.h-2-w31bandTop-w31band)
	white.frame(g, w31band+1, w31bandTop+1, g.w-2*w31band-2, g.h-w31bandTop-w31band-2, 1)
	black.frame(g, 0, 0, g.w, g.h, 1)
	// The caption between black lines.
	black.cells(g, x, y-1, w, 1)
	black.cells(g, x, y+h, w, 1)
	bandK.fill(ctx, band)
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(bar))
	left, right := x, x+w
	if st.CanClose {
		if cr := e.WindowCloseRect(l, b); !cr.Empty() {
			// The control-menu box: a grey square with a divider on its
			// right and the "space bar" glyph, a white bar outlined in black
			// with a dark grey shadow.
			ctx.DrawRect(g.at(x, y, h-1, h), paintengine2d.Fill(c.face))
			black.cells(g, x+h-1, y, 1, h)
			bw := min(13, h-4)
			bx, by := x+(h-1-bw)/2, y+(h-3)/2
			black.frame(g, bx, by, bw, 3, 1)
			white.cells(g, bx+1, by+1, bw-2, 1)
			lo.cells(g, bx+1, by+3, bw, 1)
			lo.cells(g, bx+bw, by+1, 1, 2)
			if st.ClosePress {
				// Held: the box inverts, as its menu would open.
				ctx.DrawRect(g.at(x, y, h-1, h), paintengine2d.Fill(w31xor(c.face)))
			}
			left = x + h
		}
	}
	if st.Maximizable && w > 4*h {
		// The maximize button: bevelled grey with an up triangle, a black
		// line on its left.
		mx := x + w - h
		black.cells(g, mx-1, y, 1, h)
		ctx.DrawRect(g.at(mx, y, h, h), paintengine2d.Fill(c.face))
		w31small(&hi, &lo, g, mx, y, h, h)
		t := w31tri(7, DirUp)
		t.emit(&glyph, g, mx+(h-t.w)/2, y+(h-t.h)/2)
		right = mx - 1
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	white.fill(ctx, c.hi)
	glyph.fill(ctx, c.btnText)
	black.fill(ctx, c.frame)
	if title != "" && right-left > 6 {
		f := l.BoldFont()
		l.drawFittedText(ctx, f, title, g.at(left+2, y, right-left-4, h), barText, AlignCenter, 0)
	}
}

// ---- controls -------------------------------------------------------------------

// DrawPanel: flat panels are the window colour; raised ones a grey face in
// a black line with a one-pixel bevel (the CTL3D look of 3.1 apps).
func (e win31Engine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if !raised || g.w < 4 || g.h < 4 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		return
	}
	e.DrawTabPane(l, ctx, b)
}

// DrawButton is the 3.1 push button.
func (e win31Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 6 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	c.button(ctx, g, pressed, st.Primary())
	lb := g.inset(3).rect()
	if pressed {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	f := l.BoldFont()
	c.label(l, ctx, f, label, lb, c.btnText, AlignCenter, st.Disabled(), c.face)
	if st.Focused() && !st.Disabled() {
		// The dotted rectangle round the label, a couple of pixels clear.
		tb := rpTextBox(f, label, lb, AlignCenter).Inset(-2 * u).Intersect(g.inset(3).rect().Translate(paintengine2d.Pt(0, 0)))
		if label == "" {
			tb = g.inset(4).rect()
		}
		fg := rpGridAt(ctx, tb, u)
		w31focus(ctx, fg, 0, 0, fg.w, fg.h, c.face)
	}
}

func (e win31Engine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := w31colors(l)
	u := rpU(l)
	if label == "" {
		if st.Focused() {
			e.DrawFocusRing(l, ctx, box.Inset(-u).Intersect(b))
		}
		return
	}
	f := l.BoldFont()
	gap := min(l.S(5), 8)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	c.label(l, ctx, f, label, lb, c.text, AlignStart, st.Disabled(), c.win)
	if st.Focused() {
		e.DrawFocusRing(l, ctx, labelFocusRect(f, label, lb, b))
	}
}

func (e win31Engine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e win31Engine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch: 3.1 had none; a slot in a black line, the highlight colour
// when on, with a small push-button knob.
func (e win31Engine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := w31colors(l)
	tw, th := min(l.metrics.SwitchW, b.Dx()), min(l.metrics.SwitchH, b.Dy())
	g := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th), rpU(l))
	if g.w >= 10 && g.h >= 8 {
		fill := c.win
		if on && !st.Disabled() {
			fill = c.sel
		}
		ctx.DrawRect(g.rect(), paintengine2d.Fill(fill))
		var k rpInk
		k.frame(g, 0, 0, g.w, g.h, 1)
		k.fill(ctx, c.frame)
		kw := g.h + 2
		kx := 0
		if on {
			kx = g.w - kw
		}
		c.button(ctx, g.sub(kx, 0, kw, g.h), false, false)
	}
	e.toggleLabel(l, ctx, b, g.rect(), st, label)
}

// DrawComboBox is the drop-down list: a black frame round the field and,
// across a black divider, the bevelled button with its arrow over a bar.
// A focused list highlights its text in navy with the XOR dots round it.
// An editable combo (the drop-down combo box) is an edit box with the
// button standing apart, eight pixels on, in its own frame.
func (e win31Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 12 || g.h < 8 {
		return
	}
	bx, fw, split := w31combo(g, st.Editable())
	bw := g.w - bx
	var black, hi, lo, glyph rpInk
	if split {
		ctx.DrawRect(g.at(0, 0, fw, g.h), paintengine2d.Fill(c.win))
		black.frame(g, 0, 0, fw, g.h, 1)
		black.frame(g, bx, 0, bw, g.h, 1)
	} else {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		black.frame(g, 0, 0, g.w, g.h, 1)
		black.cells(g, bx, 1, 1, g.h-2)
	}
	btn := g.sub(bx+1, 1, bw-2, g.h-2)
	ctx.DrawRect(btn.rect(), paintengine2d.Fill(c.face))
	pressed := (open || st.Pressed()) && !st.Disabled()
	off := 0
	if pressed {
		lo.cells(btn, 0, 0, btn.w, 1)
		lo.cells(btn, 0, 1, 1, btn.h-1)
		off = 1
	} else {
		w31small(&hi, &lo, btn, 0, 0, btn.w, btn.h)
	}
	// The glyph: a stem, the 7-5-3-1 triangle, a blank row, a bar.
	m := rpNewMask(7, 9)
	m.rect(2, 0, 3, 3)
	for i := 0; i < 4; i++ {
		m.span(i, 7-i, 3+i)
	}
	m.rect(0, 8, 7, 1)
	m.emit(&glyph, btn, (btn.w-7)/2+off, (btn.h-9)/2+off)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	gcol := c.btnText
	if st.Disabled() {
		gcol = c.shadow
	}
	glyph.fill(ctx, gcol)
	black.fill(ctx, c.frame)
	if split {
		// The frameless field inside draws the text.
		return
	}
	// The field, with a white margin.
	fg := g.sub(2, 2, fw-3, g.h-4)
	tc := c.text
	if st.Focused() && !open && !st.Disabled() && !st.Editable() {
		ctx.DrawRect(fg.rect(), paintengine2d.Fill(c.sel))
		tc = c.selText
		w31focus(ctx, fg, 0, 0, fg.w, fg.h, c.sel)
	}
	c.label(l, ctx, l.BoldFont(), text, fg.at(2, 0, fg.w-3, fg.h), tc, AlignStart, st.Disabled(), c.win)
}

// ComboTextRect: an editable combo's text fills its edit box, left of the
// gap and the button; the frameless field keeps its text FieldPad from
// the box's edge, two pixels clear of the black line.
func (win31Engine) ComboTextRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	if g.w < 12 || g.h < 8 {
		return paintengine2d.Rect{}
	}
	_, fw, _ := w31combo(g, true)
	return g.at(0, 1, fw, g.h-2)
}

func (c *w31) fieldOpts(l *Classic) rpFieldOpts {
	return rpFieldOpts{
		text: c.text, muted: c.gray, sel: c.sel, selTxt: c.selText,
		selOff: paintengine2d.Color{}, caret: w31xor(c.win), caretKind: rpCaretBar,
		bg: c.win,
	}
}

// DrawTextField: a one-pixel black border round the window colour, the
// text two pixels in; focus shows only the caret.
func (e win31Engine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := w31colors(l)
	u := rpU(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := max(l.metrics.FieldPad, 3*u)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+u, b.Dx()-2*pad, b.Dy()-2*u)
	rpFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.editOpts(l, st))
}

// DrawFramelessText: a spin box's or editable combo's text types as the
// edit fields do, with the XOR caret bar and the highlight.
func (e win31Engine) DrawFramelessText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	rpFieldText(l, ctx, l.fieldTextBox(b), st, text, placeholder, caret, selA, selB, blink, scrollX, face, w31colors(l).editOpts(l, st))
}

// editOpts is how an edit field types; its selection shows only while it
// has the focus.
func (c *w31) editOpts(l *Classic, st ControlState) rpFieldOpts {
	o := c.fieldOpts(l)
	if !st.Focused() {
		o.sel = paintengine2d.Color{A: 0}
	}
	return o
}

func (e win31Engine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := w31colors(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := max(l.metrics.FieldPad, 3*rpU(l))
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-2*pad, b.Dy()-2*pad)
	rpAreaText(l, ctx, inner, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, c.fieldOpts(l))
}

// DrawSpinner is a pair of stacked arrow buttons, like the scroll arrows.
func (e win31Engine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 7 || g.h < 10 {
		return
	}
	var black, hi, lo, glyph rpInk
	black.frame(g, 0, 0, g.w, g.h, 1)
	mid := g.h / 2
	black.cells(g, 1, mid, g.w-2, 1)
	for i, dir := range [2]Direction{DirUp, DirDown} {
		y0, y1, pressed := 1, mid, upPress
		if i == 1 {
			y0, y1, pressed = mid+1, g.h-1, downPress
		}
		fg := g.sub(1, y0, g.w-2, y1-y0)
		ctx.DrawRect(fg.rect(), paintengine2d.Fill(c.face))
		off := 0
		if pressed && !st.Disabled() {
			lo.cells(fg, 0, 0, fg.w, 1)
			lo.cells(fg, 0, 1, 1, fg.h-1)
			off = 1
		} else {
			w31small(&hi, &lo, fg, 0, 0, fg.w, fg.h)
		}
		t := w31tri(min(7, fg.w-4), dir)
		t.emit(&glyph, fg, (fg.w-t.w)/2+off, (fg.h-t.h)/2+off)
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	gc := c.btnText
	if st.Disabled() {
		gc = c.shadow
	}
	glyph.fill(ctx, gc)
	black.fill(ctx, c.frame)
}

// DrawTabBar: the strip the tabs stand on (grey, as the page under them).
func (e win31Engine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var black, hi rpInk
	black.cells(g, 0, g.h-2, g.w, 1)
	hi.cells(g, 0, g.h-1, g.w, 1)
	black.fill(ctx, c.frame)
	hi.fill(ctx, c.hi)
}

// DrawTab: Word 6-style tabs — a black outline with clipped top corners, a
// white highlight on the top and left, a dark right edge; the selected tab
// is taller and runs into the page.
func (e win31Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 8 {
		return
	}
	top := 0
	if !selected {
		top = 2
	}
	tg := g.sub(0, top, g.w, g.h-top)
	s := rpShape{w: tg.w, h: tg.h + 2, cut: 2, top: true}
	var face, black, hi, lo rpInk
	for y := 0; y < tg.h; y++ {
		e := s.edge(y)
		face.cells(tg, e, y, tg.w-2*e, 1)
	}
	face.fill(ctx, c.face)
	for y := 0; y < tg.h; y++ {
		a1, b1, a2, b2, ok := s.frameRuns(y)
		if !ok {
			continue
		}
		if y == 0 {
			black.cells(tg, a1, y, b1-a1, 1)
			continue
		}
		black.cells(tg, a1, y, b1-a1, 1)
		black.cells(tg, a2, y, b2-a2, 1)
		// Inner edges: white on the left and top, grey on the right.
		hi.cells(tg, b1, y, 1, 1)
		lo.cells(tg, a2-1, y, 1, 1)
		if y == 1 {
			hi.cells(tg, b1, y, a2-b1, 1)
		}
	}
	if selected {
		// Open the page's edge under the selected tab.
		ctx.DrawRect(tg.at(1, tg.h-2, tg.w-2, 2), paintengine2d.Fill(c.face))
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	black.fill(ctx, c.frame)
	f := l.BoldFont()
	lb := tg.at(3, 2, tg.w-6, tg.h-3)
	c.label(l, ctx, f, label, lb, c.btnText, AlignCenter, st.Disabled(), c.face)
	if st.Focused() && selected {
		fr := rpGridAt(ctx, rpTextBox(f, label, lb, AlignCenter).Inset(-2*u).Intersect(lb), u)
		w31focus(ctx, fr, 0, 0, fr.w, fr.h, c.face)
	}
}

// DrawMenuBar is the white menu bar over a black line.
func (e win31Engine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.menu))
	var k rpInk
	k.cells(g, 0, g.h-1, g.w, 1)
	k.fill(ctx, c.frame)
}

// DrawMenuTitle: an open title is highlighted navy with white text.
func (e win31Engine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.menuText
	if !st.Disabled() && (open || st.Pressed()) && g.h > 2 {
		ctx.DrawRect(g.at(0, 0, g.w, g.h-1), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.gray
	}
	l.drawLabeled(ctx, l.BoldFont(), label, underline, g.at(0, 0, g.w, g.h-1), fg)
	if st.Focused() && !open {
		w31focus(ctx, g, 1, 1, g.w-2, g.h-3, c.menu)
	}
}

// DrawMenuFrame: a white menu in a one-pixel black border, no shadow.
func (e win31Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.menu))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.frame)
}

// DrawMenuItem: bold items, the navy highlight across the menu, a black
// tick for checked items, a black triangle for submenus, a black line for
// separators, disabled items in the gray text colour.
func (e win31Engine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := w31colors(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*u, b.Dy()), u)
	if row.Separator {
		var k rpInk
		k.cells(full, 0, full.h/2, full.w, 1)
		k.fill(ctx, c.frame)
		return
	}
	col := c.menuText
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		ctx.DrawRect(full.rect(), paintengine2d.Fill(c.sel))
		col = c.selText
	}
	if st.Disabled() {
		col = c.gray
	}
	f := l.BoldFont()
	mark := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked && row.Radio:
		var k rpInk
		rpShape{w: 7, h: 7, r: 3.5}.fill(&k, mark, (mark.w-7)/2, (mark.h-7)/2)
		k.fill(ctx, col)
	case row.Checked:
		if n := min(9, mark.w-2, mark.h-4); n >= 5 {
			m := s7check(n)
			var k rpInk
			m.emit(&k, mark, (mark.w-n)/2, (mark.h-n)/2)
			k.fill(ctx, col)
		}
	case row.Icon != IconNone:
		side := min(ch.CheckCol()-l.S(2), b.Dy()-l.S(4))
		ib := paintengine2d.XYWH(b.Min.X+(ch.CheckCol()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		l.drawToolIcon(ctx, ib, row.Icon, col)
	}
	labelRight := b.Max.X
	if row.Submenu {
		ab := rpGridAt(ctx, paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow+l.S(4), b.Dy()), u)
		t := w31tri(9, DirRight)
		var k rpInk
		t.emit(&k, ab, 0, (ab.h-t.h)/2)
		k.fill(ctx, col)
		labelRight = ch.ArrowMinX(b.Max.X)
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
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()))
	l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), col)
	ctx.Restore()
}

// DrawProgressBar: 3.1 had no progress control; Setup's gauge was a black
// frame round white, filling with the highlight colour.
func (e win31Engine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	in := g.inset(1)
	ctx.DrawRect(in.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.frame)
	fill := c.sel
	if st.Disabled() {
		fill = c.shadow
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		span := in.w / 4
		x := int(float32(in.w+span)*phase) - span
		a, z := max(x, 0), min(x+span, in.w)
		if z > a {
			ctx.DrawRect(in.at(a, 0, z-a, in.h), paintengine2d.Fill(fill))
		}
		return
	}
	if n := int(float32(in.w)*clamp1(t) + 0.5); n > 0 {
		ctx.DrawRect(in.at(0, 0, n, in.h), paintengine2d.Fill(fill))
	}
}

// DrawSlider: 3.1 had none; a sunken groove and a push-button thumb.
func (e win31Engine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 16 || g.h < 10 {
		return
	}
	kw, kh := 11, min(g.h-2, 21)
	cy := g.h / 2
	var hi, lo, black rpInk
	gr := g.sub(kw/2, cy-2, g.w-kw/2*2, 4)
	ctx.DrawRect(gr.rect(), paintengine2d.Fill(c.win))
	rpEdge(&lo, &hi, gr, 0, 0, gr.w, gr.h, 1)
	black.frame(gr, 1, 1, gr.w-2, gr.h-2, 1)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	black.fill(ctx, c.frame)
	kx := int(float32(g.w-kw)*clamp1(t) + 0.5)
	c.button(ctx, g.sub(kx, (g.h-kh)/2, kw, kh), st.Pressed() && !st.Disabled(), false)
	if st.Focused() {
		w31focus(ctx, g, 0, 0, g.w, g.h, c.win)
	}
}

func (e win31Engine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a File Manager directory row: solid lines to the parent,
// a +/- box on branches, the name highlighted when selected.
func (e win31Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	ic := int(indent/u + 0.5)
	pad := 4
	x := pad + depth*ic
	cy := g.h / 2
	var lines rpInk
	for d := 0; d <= depth; d++ {
		gx := pad + d*ic + ic/2
		if d == depth {
			lines.cells(g, gx, 0, 1, cy+1)
			lines.cells(g, gx, cy, ic/2+2, 1)
		} else {
			lines.cells(g, gx, 0, 1, g.h)
		}
	}
	lines.fill(ctx, c.shadow)
	if !leaf {
		e.Expander(l, ctx, g.at(x, 0, ic, g.h), expanded, c.frame)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := g.x + float32(x+ic+4)*u
	lb := paintengine2d.XYWH(lx-2*u, b.Min.Y+u, f.Advance(label)+4*u, b.Dy()-2*u).Intersect(b)
	fg := c.text
	if st.Checked() {
		ctx.DrawRect(rpGridAt(ctx, lb, u).rect(), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, lb, st)
	}
}

func (e win31Engine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	fg := e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, align, 0)
}

// DrawTableHeader: raised grey heading buttons edged in black.
func (e win31Engine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var black, hi, lo rpInk
	black.cells(g, 0, g.h-1, g.w, 1)
	black.cells(g, g.w-1, 0, 1, g.h-1)
	pressed := st.Pressed() && !st.Disabled()
	off := float32(0)
	if pressed {
		lo.cells(g, 0, 0, g.w-1, 1)
		lo.cells(g, 0, 1, 1, g.h-2)
		off = u
	} else {
		rpEdge(&hi, &lo, g, 0, 0, g.w-1, g.h-1, 1)
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	black.fill(ctx, c.frame)
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(3)+off, b.Min.Y+off, aw, b.Dy()), dir, c.btnText)
	}
	l.drawFittedText(ctx, l.BoldFont(), label, paintengine2d.XYWH(b.Min.X+l.S(5)+off, b.Min.Y+off, b.Dx()-l.S(9)-aw, b.Dy()-u), c.btnText, AlignStart, 0)
}

// DrawToolBar: a grey bar over a black line (Office 4's toolbars).
func (e win31Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var hi, k rpInk
	hi.cells(g, 0, 0, g.w, 1)
	k.cells(g, 0, g.h-1, g.w, 1)
	hi.fill(ctx, c.hi)
	k.fill(ctx, c.frame)
}

// DrawToolButton: flat on the bar until the pointer arrives; then, pressed
// and latched, the push-button body.
func (e win31Engine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := w31colors(l)
	u := rpU(l)
	e.Face(l, ctx, b.Inset(u), RoleTool, st)
	off := float32(0)
	if (st.Pressed() || (st.Toggle() && st.Checked())) && !st.Disabled() {
		off = u
	}
	fg := c.btnText
	if st.Disabled() {
		fg = c.shadow
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad + off
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5+off, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5+off, b.Min.Y+(b.Dy()-iconSide)*0.5+off, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y+off, b.Max.X-pad-x, b.Dy()), fg, AlignStart, 0)
	}
	if st.Focused() {
		g := rpGridAt(ctx, b, u)
		w31focus(ctx, g, 3, 3, g.w-6, g.h-6, c.face)
	}
}

// DrawStatusBar: File Manager's grey bar under a black line with sunken
// panes, eight pixels apart.
func (e win31Engine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 20 || g.h < 6 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var black, hi, lo rpInk
	black.cells(g, 0, 0, g.w, 1)
	n := len(parts)
	if n > 0 {
		avail := g.w - 16 - 8*(n-1)
		w := avail / n
		for i, s := range parts {
			x := 8 + i*(w+8)
			pg := g.sub(x, 3, w, g.h-5)
			rpEdge(&lo, &hi, pg, 0, 0, pg.w, pg.h, 1)
			l.drawFittedText(ctx, l.body, s, pg.at(4, 1, pg.w-8, pg.h-2), c.btnText, AlignStart, 0)
		}
	}
	black.fill(ctx, c.frame)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
}

// DrawTitleBar is a panel heading: the bold title on the button face over
// a black line.
func (e win31Engine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.face))
	var hi, k rpInk
	hi.cells(g, 0, 0, g.w, 1)
	k.cells(g, 0, g.h-1, g.w, 1)
	hi.fill(ctx, c.hi)
	k.fill(ctx, c.frame)
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.btnText, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.shadow, AlignStart, 0)
	}
}

// DrawAccordionHeader: a push-button bar with a +/- box and a bold title.
func (e win31Engine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := w31colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 8 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	c.button(ctx, g, pressed, false)
	off := float32(0)
	if pressed {
		off = u
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(6)+off, b.Min.Y+off, l.S(12), b.Dy()), expanded, c.frame)
	f := l.BoldFont()
	lb := paintengine2d.XYWH(b.Min.X+l.S(24)+off, b.Min.Y+off, b.Dx()-l.S(28), b.Dy())
	c.label(l, ctx, f, title, lb, c.btnText, AlignStart, st.Disabled(), c.face)
	if st.Focused() {
		fr := rpGridAt(ctx, rpTextBox(f, title, lb, AlignStart).Inset(-2*u).Intersect(g.inset(3).rect()), u)
		w31focus(ctx, fr, 0, 0, fr.w, fr.h, c.face)
	}
}

// DrawSeparator is a one-pixel black line.
func (e win31Engine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	var k rpInk
	if vertical {
		k.cells(g, g.w/2, 2, 1, g.h-4)
	} else {
		k.cells(g, 0, g.h/2, g.w, 1)
	}
	k.fill(ctx, c.frame)
}

// DrawSplitter: File Manager's split bar, grey between black lines.
func (e win31Engine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fill := c.face
	if st.Pressed() {
		fill = w31xor(c.face)
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(fill))
	var k rpInk
	if vertical {
		k.cells(g, 0, 0, 1, g.h)
		k.cells(g, g.w-1, 0, 1, g.h)
	} else {
		k.cells(g, 0, 0, g.w, 1)
		k.cells(g, 0, g.h-1, g.w, 1)
	}
	k.fill(ctx, c.frame)
}

// TooltipStyle: Windows 3.1 pads a tip by 4.
func (e win31Engine) TooltipStyle(l *Classic) TooltipStyle {
	return l.tipStyle(l.body, l.tipPad(l.S(4)), AlignStart)
}

// DrawTooltip: tooltips came after 3.1 (Word 6 drew its own): a pale
// yellow box in a black line.
func (e win31Engine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := w31colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.info))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.frame)
	pad := l.TooltipStyle().Pad
	l.drawTipText(ctx, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), text, c.infoText)
}

// DrawOverlay: Windows 3.1 did not dim the window behind a dialog.
func (win31Engine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- packs --------------------------------------------------------------------

func win31Pack(name, label, summary string, pal Palette, extra map[string]string) ThemePack {
	tok := ThemeTokens{
		Engine:  "win31",
		Bevel:   BevelClassic3D,
		Family:  ThemeLight,
		Palette: pal,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	return ThemePack{
		Name: name, Label: label, Year: 1992, Lineage: "Windows", Summary: summary,
		Era: "Windows 3.x", Palette: ThemeLight, Tokens: tok,
	}
}

// w31palette builds a palette from a scheme: window and its text, button
// face, highlight and its text.
func w31palette(win, text, face, sel, selText string) Palette {
	f := hexColor(face)
	return Palette{
		Background: hexColor(win), Surface: hexColor(win), SurfaceAlt: f,
		Border: hexColor("#000000"), Divider: hexColor("#808080"),
		Text: hexColor(text), TextMuted: hexColor("#808080"), TextOnAccent: hexColor(selText),
		Accent: hexColor(sel), AccentHover: hexColor(sel), AccentPress: hexColor(sel),
		Field: hexColor(win), FieldBorder: hexColor("#000000"),
		Focus: hexColor(text), Selection: hexColor(sel),
		Track: f, Thumb: f,
		Highlight: hexColor("#ffffff"), Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: hexColor(sel), MenuHoverBorder: hexColor(sel), MenuGutter: hexColor("#ffffff"),
		Danger: hexColor("#800000"), Success: hexColor("#008000"), Warning: hexColor("#808000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: hexColor("#ffffff"), BevelDark: hexColor("#808080"),
	}
}

func win31Packs() []ThemePack {
	hot := w31palette("#ff0000", "#ffffff", "#c0c0c0", "#000000", "#ffffff")
	hot.TextMuted = hexColor("#ffff00")
	hot.Danger, hot.Success, hot.Warning = hexColor("#ffff00"), hexColor("#ffffff"), hexColor("#ffff00")
	hot.Accent, hot.AccentHover, hot.AccentPress = hexColor("#ffff00"), hexColor("#ffff00"), hexColor("#ffff00")
	return []ThemePack{
		win31Pack("win31", "Windows 3.1",
			"Windows 3.1 on VGA: navy captions, white dialogs, black-edged bevelled buttons and the dotted focus.",
			w31palette("#ffffff", "#000000", "#c0c0c0", "#000080", "#ffffff"), nil),
		win31Pack("win-hotdog", "Hot Dog Stand",
			"The infamous Windows 3.1 scheme: red windows, black captions, yellow desktop and no mercy.",
			hot, map[string]string{
				"caption": "#000000", "captionText": "#ffffff", "captionOff": "#ff0000", "captionOffText": "#ffffff",
				"border": "#ff0000", "gray": "#808080", "menu": "#ffffff", "menuText": "#000000",
			}),
	}
}
