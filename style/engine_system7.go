package style

import "github.com/codemodify/paintengine2d"

// system7Engine paints the classic Macintosh desktop: System 1 (1984), the
// 1-bit Finder of the 512 × 342 screen, through System 7 (1991), which kept
// the black-and-white controls and gave colour screens its "Standard"
// window colours.
//
// Everything is drawn in whole pixels (see engine_system7_pixel.go), to the
// proportions of 12-point Chicago:
//
//   - Push buttons are QuickDraw round rects (the corner cuts three pixels,
//     then one, one, none on a 20-pixel button): a one-pixel black staircase
//     round white, inverted while pressed. The default button wears the
//     outline three pixels thick, one pixel off its edge; a disabled button
//     keeps its frame and greys its title (and its outline).
//   - Check boxes are 12-pixel squares crossed corner to corner; radio
//     buttons 12-pixel circles with a 6-pixel dot. Tracking thickens the
//     frame to two pixels.
//   - Scroll bars are 16 pixels wide: outlined arrows (a 13-pixel head over
//     a 7-pixel stem) in boxes at both ends, filled while pressed; the track
//     in QuickDraw's 25% "light gray"; the white scroll box, which never
//     changes size. A bar with nothing to scroll is white and empty.
//   - Document windows: a one-pixel frame and shadow, the title bar's six
//     stripes broken by the close box at the left, the zoom box at the
//     right and the centred bold title; inactive windows lose the stripes
//     and boxes. Menus hang from a white bar and drop the same hard shadow,
//     offset from their top and left corners.
//   - Selections invert; a list without keyboard focus outlines its
//     selection, and the list with it gets a two-pixel border one pixel out.
//   - Disabled text is grey the 1-bit way: every other pixel erased.
//
// System 7 (param "grey") keeps the controls and draws the chrome in its
// default "Standard" window colours: a light title bar with grey stripes and
// a lavender (#CCCCFF) and navy (#333366) bevel, the scroll track's 25%
// grey dots on light grey, arrow boxes with lavender-filled arrows, a grey
// scroll box with a lavender grip, grey text for disabled items and grey
// frames on inactive windows.
//
// Pack data:
//
//	params: grey   1 = System 7 colour chrome, 0 = 1-bit
//	extra (System 7 colours):
//	  bar, stripe            title bar face and stripes
//	  lav, lavDk, navy       the window colour's light, mid and dark
//	  boxFace                close and zoom box face
//	  arrowFace, arrowEdge   scroll arrow box face and its dark edge
//	  arrowFill              scroll arrow fill
//	  thumb, grip            scroll box face and the dark of its ridges
//	  trackBg, trackDot      scroll track and its dots
//	  empty                  a scroll bar with nothing to scroll
//	  progFill, progEmpty    progress bar
//	  offLine, offTitle      frame and title of inactive windows
//	  sep                    menu separators
//	  info                   Balloon Help fill
type system7Engine struct{ BaseEngine }

func init() {
	RegisterEngine(system7Engine{})
	for _, p := range system7Packs() {
		RegisterPack(p)
	}
}

func (system7Engine) ID() string { return "system7" }

// DefaultMetrics are the Mac's proportions at the toolkit's 16px UI font
// (the Mac used 12-point Chicago): push buttons in a 28px slot that also
// holds the default outline, 12px check boxes and radios, 16px scroll bars.
func (system7Engine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 4,
		Square:    true, BevelDepth: 1,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 12, Radio: 12,
		MenuItemH: 22, MenuBarH: 24, TabH: 26, RowH: 20,
		TitleBar: 26, HeaderH: 22, ProgressH: 16, SliderH: 24, Thumb: 11,
		Scroll: 16, Pad: 10, FieldPad: 4, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 22, SpinnerW: 13, SwitchW: 34, SwitchH: 16,
	}
}

// ---- colours ------------------------------------------------------------------

// s7 is the resolved colour set of a look.
type s7 struct {
	grey                                   bool // System 7 colour chrome
	black, white, text, dim, field, win    paintengine2d.Color
	sel, selTxt, info                      paintengine2d.Color
	bar, stripe, lav, lavDk, navy, boxFace paintengine2d.Color
	arrowFace, arrowEdge, arrowFill, grip  paintengine2d.Color
	thumb, trackBg, trackDot, empty        paintengine2d.Color
	progFill, progEmpty, offLine, offTitle paintengine2d.Color
	sep                                    paintengine2d.Color
}

type s7Key struct{}

func s7colors(l *Classic) *s7 {
	return l.Memo(s7Key{}, func() any { return s7build(l) }).(*s7)
}

func s7build(l *Classic) *s7 {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c := &s7{
		grey:  l.P("grey", 0) != 0,
		black: black, white: white,
		text:  p.Text,
		dim:   p.TextMuted,
		field: p.Field,
		win:   p.Background,
		sel:   p.Selection,
	}
	if c.sel.A < 0.9 {
		c.sel = black
	}
	c.selTxt = ReadableOn(c.sel, 4.5, white, black)
	x := func(k, def string) paintengine2d.Color { return l.X(k, Hex(def)) }
	c.info = x("info", "#ffffff")
	c.bar = x("bar", "#eeeeee")
	c.stripe = x("stripe", "#777777")
	c.lav = x("lav", "#ccccff")
	c.lavDk = x("lavDk", "#a3a3d7")
	c.navy = x("navy", "#333366")
	c.boxFace = x("boxFace", "#a4a4a4")
	c.arrowFace = x("arrowFace", "#dddddd")
	c.arrowEdge = x("arrowEdge", "#777777")
	c.arrowFill = x("arrowFill", "#a3a3d7")
	c.thumb = x("thumb", "#aaaaaa")
	c.grip = x("grip", "#666699")
	c.trackBg = x("trackBg", "#dddddd")
	c.trackDot = x("trackDot", "#777777")
	c.empty = x("empty", "#eeeeee")
	c.progFill = x("progFill", "#404040")
	c.progEmpty = x("progEmpty", "#ccccff")
	c.offLine = x("offLine", "#555555")
	c.offTitle = x("offTitle", "#808080")
	c.sep = x("sep", "#808080")
	return c
}

// ---- drawing helpers -------------------------------------------------------------

// label draws text fitted into b; disabled text is greyed the 1-bit way
// (every other pixel erased to bg) or, on colour screens, drawn in grey.
func (c *s7) label(l *Classic, ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect, col paintengine2d.Color, align Align, disabled bool, bg paintengine2d.Color) {
	if text == "" || b.Empty() {
		return
	}
	if disabled && c.grey {
		col = c.dim
	}
	l.drawFittedText(ctx, f, text, b, col, align, 0)
	if disabled && !c.grey {
		rpPatFill(ctx, rpTextBox(f, text, b, align).Intersect(b), rpGray, bg, rpU(l))
	}
}

// grey paints r in the disabled grey: the 50% dither erasing to bg on 1-bit
// screens, nothing on colour screens (their parts pick grey inks instead).
func (c *s7) greyOut(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, bg paintengine2d.Color) {
	if !c.grey {
		rpPatFill(ctx, r, rpGray, bg, rpU(l))
	}
}

// rpTextBox is the box drawFittedText (pad 0) puts text in inside b.
func rpTextBox(f *Font, text string, b paintengine2d.Rect, align Align) paintengine2d.Rect {
	if f == nil || text == "" {
		return paintengine2d.Rect{}
	}
	tw := f.Advance(text)
	if tw > b.Dx() {
		tw = b.Dx()
	}
	th := f.Height()
	x := b.Min.X
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - 2
	}
	return paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-th)*0.5, tw, th)
}

// s7radius is the corner of a button body h cells tall: a quarter of its
// height (five on the Mac's 20-pixel buttons, which cut 3, 1, 1, 0).
func s7radius(h int) float32 {
	r := float32(h) / 4
	if r < 3 {
		r = 3
	}
	return r
}

// s7btnGeom is the layout of a push button in rect b: the grid, the margin
// every button keeps for the default outline (the outline's width plus a
// one-cell gap), the outline width and the body's corner radius.
func s7btnGeom(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) (g rpGrid, m, ring int, r float32) {
	g = rpGridAt(ctx, b, rpU(l))
	ring = 3
	if g.h < 30 {
		ring = 2 // a 28px slot keeps a 22px body: the outline gives a pixel
	}
	m = ring + 1
	if g.h-2*m < 14 || g.w-2*m < 20 {
		m, ring = 0, 0
	}
	return g, m, ring, s7radius(g.h - 2*m)
}

// s7arrowMask is the Mac scroll arrow pointing up, w cells wide: a head
// that widens by a cell each side per row from a one-cell tip to the full
// width, over a stem half as wide and four rows long; outlined, or filled
// while pressed.
func s7arrowMask(w int, filled bool) rpMask {
	if w%2 == 0 {
		w--
	}
	if w < 5 {
		w = 5
	}
	hh := (w + 1) / 2
	stemW := hh
	if stemW%2 == 0 {
		stemW--
	}
	stemH := max(2, (w+5)/4)
	m := rpNewMask(w, hh+stemH)
	cx := w / 2
	for i := 0; i < hh; i++ {
		m.span(cx-i, cx+i+1, i)
	}
	m.rect(cx-stemW/2, hh, stemW, stemH)
	if !filled {
		m = m.outline()
	}
	return m
}

// s7tri is a filled triangle of base w cells pointing dir: rows of w, w-2
// … 1 cells (the pop-up menu's 11 × 6 triangle, hierarchical marks).
func s7tri(w int, dir Direction) rpMask {
	if w%2 == 0 {
		w--
	}
	if w < 3 {
		w = 3
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

// s7check is the menu check mark in an n-cell box: a short stroke down to
// the right and a long one up to the top right, two cells thick.
func s7check(n int) rpMask {
	m := rpNewMask(n, n)
	bx, by := n*3/10, n*8/10
	m.line(1, by-bx+1, bx, by)
	m.line(2, by-bx+1, bx+1, by)
	m.line(bx, by, n-2, 1)
	m.line(bx+1, by, n-1, 1)
	return m
}

// ---- parts --------------------------------------------------------------------

func (e system7Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	fg := c.text
	if st.Disabled() && c.grey {
		fg = c.dim
	}
	if g.w < 3 || g.h < 3 {
		return fg
	}
	switch role {
	case RoleButton, RoleCombo:
		pressed := st.Pressed() && !st.Disabled()
		s := rpShape{w: g.w, h: g.h, r: s7radius(g.h)}
		var face, edge rpInk
		s.fill(&face, g, 0, 0)
		s.frame(&edge, g, 0, 0)
		if pressed {
			face.fill(ctx, c.black)
			return c.white
		}
		face.fill(ctx, c.white)
		edge.fill(ctx, c.black)
		return fg
	case RoleTool:
		switch {
		case st.Disabled():
		case st.Pressed() || (st.Toggle() && st.Checked()):
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.black))
			return c.white
		case st.Hovered():
			var k rpInk
			rpShape{w: g.w, h: g.h, r: 3}.frame(&k, g, 0, 0)
			k.fill(ctx, c.black)
		}
		return fg
	case RoleField, RoleCheck:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.field))
		var k rpInk
		k.frame(g, 0, 0, g.w, g.h, 1)
		k.fill(ctx, c.black)
		return fg
	case RoleRow:
		if !st.Checked() {
			return fg
		}
		if st.Inactive() || st.Backdrop() {
			// A list without keyboard focus outlines its selection.
			var k rpInk
			k.frame(g, 0, 0, g.w, g.h, 1)
			col := c.sel
			if c.grey {
				col = c.stripe
			}
			k.fill(ctx, col)
			return c.text
		}
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.sel))
		return c.selTxt
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.black))
			return c.white
		}
		return c.text
	case RoleThumb:
		c.thumbBox(ctx, g, g.h >= g.w, st.Pressed())
		return fg
	case RoleTrack:
		c.trackFill(l, ctx, g.rect(), true)
		return fg
	case RoleTab:
		return fg
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	return fg
}

// trackFill paints a scroll track: QuickDraw's 25% grey (black dots on
// white; System 7: grey dots on light grey), or the empty colour when there
// is nothing to scroll.
func (c *s7) trackFill(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, active bool) {
	if r.Empty() {
		return
	}
	switch {
	case !active && c.grey:
		ctx.DrawRect(r, paintengine2d.Fill(c.empty))
	case !active:
		ctx.DrawRect(r, paintengine2d.Fill(c.white))
	case c.grey:
		ctx.DrawRect(r, paintengine2d.Fill(c.trackBg))
		rpPatFill(ctx, r, rpLight, c.trackDot, rpU(l))
	default:
		ctx.DrawRect(r, paintengine2d.Fill(c.white))
		rpPatFill(ctx, r, rpLight, c.black, rpU(l))
	}
}

// thumbBox paints the scroll box filling g: a one-pixel black frame round
// white, or on System 7 round a grey face with a lavender and navy bevel
// and a grip of four ridges across its middle.
func (c *s7) thumbBox(ctx *paintengine2d.Context, g rpGrid, vertical, pressed bool) {
	if g.w < 3 || g.h < 3 {
		return
	}
	var black rpInk
	black.frame(g, 0, 0, g.w, g.h, 1)
	if !c.grey {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.white))
		black.fill(ctx, c.black)
		return
	}
	face := c.thumb
	if pressed {
		face = c.lavDk
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
	var hi, lo rpInk
	rpEdge(&hi, &lo, g, 1, 1, g.w-2, g.h-2, 1)
	// The grip: four ridges six cells long, lavender over the dark.
	for i := 0; i < 4; i++ {
		if vertical {
			y := g.h/2 - 4 + 2*i
			hi.cells(g, (g.w-6)/2, y, 6, 1)
			lo.cells(g, (g.w-6)/2, y+1, 6, 1)
		} else {
			x := g.w/2 - 4 + 2*i
			hi.cells(g, x, (g.h-6)/2, 1, 6)
			lo.cells(g, x+1, (g.h-6)/2, 1, 6)
		}
	}
	hi.fill(ctx, c.lav)
	lo.fill(ctx, c.navy)
	black.fill(ctx, c.black)
}

// CheckIndicator is the Mac check box: a one-pixel square, two pixels
// while the mouse tracks it, crossed corner to corner when checked. A
// disabled box stays black on 1-bit screens (only its title dims).
func (e system7Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := s7colors(l)
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
	var k rpInk
	k.frame(g, 0, 0, n, n, t)
	if checked {
		m := rpNewMask(n, n)
		m.line(t, t, n-1-t, n-1-t)
		m.line(n-1-t, t, t, n-1-t)
		m.emit(&k, g, 0, 0)
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.field))
	ink := c.black
	if st.Disabled() && c.grey {
		ink = c.dim
	}
	k.fill(ctx, ink)
}

// RadioIndicator is the Mac radio button: a one-pixel circle (two while
// tracking) with a black dot three pixels in when selected.
func (e system7Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h, int(box.Dx()/u+0.5))
	if n < 6 {
		return
	}
	g = g.centered(n, n)
	disc := rpShape{w: n, h: n, r: float32(n) * 0.5}
	var face, ink rpInk
	disc.fill(&face, g, 0, 0)
	face.fill(ctx, c.field)
	if st.Pressed() && !st.Disabled() {
		disc.ring(&ink, g, 0, 0, 2)
	} else {
		disc.frame(&ink, g, 0, 0)
	}
	if selected {
		d := n / 2
		if (n-d)%2 != 0 {
			d--
		}
		rpShape{w: d, h: d, r: float32(d) * 0.5}.fill(&ink, g, (n-d)/2, (n-d)/2)
	}
	col := c.black
	if st.Disabled() && c.grey {
		col = c.dim
	}
	ink.fill(ctx, col)
}

// Arrow is a solid whole-pixel triangle (pop-up marks, sort indicators).
func (e system7Engine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 3 {
		return
	}
	m := s7tri(min(n*2/3, 11), dir)
	var k rpInk
	m.emit(&k, g, (g.w-m.w)/2, (g.h-m.h)/2)
	k.fill(ctx, col)
}

// Expander is the Finder's disclosure triangle: outlined, pointing right,
// down when the folder is open.
func (e system7Engine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h, 13) - 2
	if n < 5 {
		return
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	m := s7tri(n, dir)
	ox, oy := (g.w-m.w)/2, (g.h-m.h)/2
	var face, edge rpInk
	m.emit(&face, g, ox, oy)
	o := m.outline()
	o.emit(&edge, g, ox, oy)
	face.fill(ctx, c.field)
	edge.fill(ctx, col)
}

// MenuHighlight inverts the item: black with white text.
func (e system7Engine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(s7colors(l).black))
}

func (e system7Engine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := s7colors(l)
	if hot {
		return c.white
	}
	return c.text
}

// FieldFocusRing: text fields show the insertion point only.
func (system7Engine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is a dotted (50% grey) rectangle just inside b.
func (e system7Engine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	g := rpGridAt(ctx, b, rpU(l))
	var k rpInk
	k.dotted(g, 0, 0, g.w, g.h)
	k.fill(ctx, s7colors(l).text)
}

// ---- scrollbars -------------------------------------------------------------

// ScrollBarStyle: 16px bars with an arrow box at each end; the thumb is at
// least as long as the scroll box drawn at its place.
func (system7Engine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsEnds, MinThumb: 16, FixedThumb: 16}
}

// rpBoxAt places a fixed-length box of n cells along a track at the
// thumb's fraction: its first cell, the track starting at cell t0 and
// running tn cells (overlap lets the box sit on the lines at the ends).
// The Mac's scroll box, Windows 3.1's thumb and OPEN LOOK's elevator never
// changed size; the thumb the toolkit hit-tests is the proportional one,
// which always contains the box.
func rpBoxAt(track, thumb paintengine2d.Rect, vertical bool, g rpGrid, n, overlap int) (int, bool) {
	if thumb.Empty() {
		return 0, false
	}
	var t0, tl, th0, thl, o float32
	if vertical {
		t0, tl, th0, thl, o = track.Min.Y, track.Dy(), thumb.Min.Y, thumb.Dy(), g.y
	} else {
		t0, tl, th0, thl, o = track.Min.X, track.Dx(), thumb.Min.X, thumb.Dx(), g.x
	}
	f := float32(0)
	if free := tl - thl; free > 0.5 {
		f = clamp1((th0 - t0) / free)
	}
	a := int((t0-o)/g.u+0.5) - overlap
	span := int(tl/g.u+0.5) + 2*overlap - n
	if span < 0 {
		return 0, false
	}
	return a + int(f*float32(span)+0.5), true
}

func (e system7Engine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := s7colors(l)
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
	// cell maps (along, across) cells to a device rect of g.
	cell := func(a, x, n, m int) paintengine2d.Rect {
		if vertical {
			return g.at(x, a, m, n)
		}
		return g.at(a, x, n, m)
	}
	pos := func(r paintengine2d.Rect) (int, int) {
		if vertical {
			return int((r.Min.Y-g.y)/u + 0.5), int(r.Dy()/u + 0.5)
		}
		return int((r.Min.X-g.x)/u + 0.5), int(r.Dx()/u + 0.5)
	}
	var black rpInk
	black.frame(g, 0, 0, g.w, g.h, 1)
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
		if i == 0 {
			lo = a0 + n
			black.add(cell(a0+n-1, 1, 1, across-2))
		} else {
			hi = a0
			black.add(cell(a0, 1, 1, across-2))
		}
		box := rpGridAt(ctx, cell(a0+1, 1, n-2, across-2), u)
		pressed := st.Pressed == a.part && active
		face := c.white
		switch {
		case c.grey && !active:
			face = c.empty
		case c.grey && pressed:
			face = c.thumb
		case c.grey:
			face = c.arrowFace
		}
		ctx.DrawRect(box.rect(), paintengine2d.Fill(face))
		if c.grey && active {
			var hk, lk rpInk
			if pressed {
				rpEdge(&lk, &hk, box, 0, 0, box.w, box.h, 1)
			} else {
				rpEdge(&hk, &lk, box, 0, 0, box.w, box.h, 1)
			}
			hk.fill(ctx, c.white)
			lk.fill(ctx, c.arrowEdge)
		}
		w := min(box.w, box.h) - 1
		if c.grey {
			w = min(box.w, box.h) - 3
		}
		m := s7arrowMask(w, pressed && !c.grey).turn(a.dir)
		ox, oy := (box.w-m.w)/2, (box.h-m.h)/2
		switch {
		case !c.grey:
			var k rpInk
			m.emit(&k, box, ox, oy)
			k.fill(ctx, c.black)
		case !active:
			var k rpInk
			m.emit(&k, box, ox, oy)
			k.fill(ctx, c.arrowEdge)
		default:
			// Outlined in navy round the lavender fill; solid navy pressed.
			fm := s7arrowMask(w, true).turn(a.dir)
			var fk, ok rpInk
			fm.emit(&fk, box, ox, oy)
			if pressed {
				fk.fill(ctx, c.navy)
				break
			}
			fk.fill(ctx, c.arrowFill)
			m.emit(&ok, box, ox, oy)
			ok.fill(ctx, c.navy)
		}
	}
	if hi > lo {
		c.trackFill(l, ctx, cell(lo, 1, hi-lo, across-2), active)
	}
	if active {
		// The scroll box: 16 cells along, inside the bar's side lines.
		if a, ok := rpBoxAt(p.Track, p.Thumb, vertical, g, across, 1); ok {
			c.thumbBox(ctx, rpGridAt(ctx, cell(a, 1, across, across-2), u), vertical, st.Pressed == ScrollThumbPart)
		}
	}
	black.fill(ctx, c.black)
}

// DrawScrollBar paints a bare track and scroll box (widgets that lay out
// their own bar).
func (e system7Engine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------

func (system7Engine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is the dotted rectangle the 1992 guidelines suggest round
// a group, its bold title breaking the top edge at the left; a raised box
// is a shadowed panel.
func (e system7Engine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := s7colors(l)
	u := rpU(l)
	f := l.BoldFont()
	if raised {
		e.DrawPanel(l, ctx, b, true)
		if title != "" {
			l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+l.metrics.Pad, b.Min.Y+u*2, b.Dx()-l.metrics.Pad*2, f.Height()), c.text, AlignStart, 0)
		}
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	top := b.Min.Y
	if title != "" {
		top += f.Height() * 0.5
	}
	g := rpGridAt(ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, u)
	var k rpInk
	k.dotted(g, 0, 0, g.w, g.h)
	k.fill(ctx, c.text)
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
	ctx.DrawRect(paintengine2d.XYWH(snap(tx), b.Min.Y, snap(tw), f.Height()), paintengine2d.Fill(c.win))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// s7barH is the title bar's height in cells: 18 rows between the frame's
// top line and the separator at 12pt, grown with the toolkit's bold font.
func s7barH(l *Classic) int {
	h := int(l.BoldFont().Height()/rpU(l)+0.5) + 2
	if h < 18 {
		h = 18
	}
	return h
}

func (system7Engine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: float32(s7barH(l)+2) * u, Right: u, Bottom: u, Left: u}
}

// s7closeBox is the close box of a frame's title bar in cells: 11 × 11,
// nine cells from the frame's left edge, level with the stripes.
func s7closeBox(l *Classic) (x, y, n int) {
	bh := s7barH(l)
	n = 11
	return 9, 1 + (bh-n)/2, n
}

// WindowCloseRect is the close box at the left of the title bar.
func (e system7Engine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	if g.w < 40 || g.h < s7barH(l)+4 {
		return paintengine2d.Rect{}
	}
	x, y, n := s7closeBox(l)
	return g.at(x, y, n, n)
}

// PopupShadow: menus, balloons and windows cast the Mac's hard one-pixel
// shadow down and to the right.
func (system7Engine) PopupShadow(l *Classic, kind PopupKind) Insets {
	u := rpU(l)
	return Insets{Right: u, Bottom: u}
}

// DrawPopupShadow: a window's shadow runs its whole right and bottom edge
// one pixel out; a menu's starts three pixels below its top and two in from
// its left; a balloon's follows its round corners.
func (system7Engine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	var k rpInk
	switch kind {
	case PopupTooltip:
		s := rpShape{w: g.w, h: g.h, r: s7balloonR(g)}
		s.fill(&k, g.sub(1, 1, g.w, g.h), 0, 0)
		ctx.Save()
		// Only what shows past the balloon: its right and bottom edge.
		ctx.ClipRect(paintengine2d.Rect{Min: paintengine2d.Pt(g.x+u, g.y+u), Max: paintengine2d.Pt(g.x+float32(g.w+1)*u, g.y+float32(g.h+1)*u)})
		k.fill(ctx, c.black)
		ctx.Restore()
		return
	case PopupMenu:
		k.cells(g, g.w, 3, 1, g.h-2)
		k.cells(g, 2, g.h, g.w-2, 1)
	default:
		k.cells(g, g.w, 1, 1, g.h)
		k.cells(g, 1, g.h, g.w-1, 1)
	}
	k.fill(ctx, c.black)
}

func s7balloonR(g rpGrid) float32 {
	return min(float32(g.h)*0.35, 8)
}

// ItemFocus: none — a list with keyboard focus is framed as a whole (its
// view frame grows the two-pixel border), the System 7 way.
func (system7Engine) ItemFocus(*Classic, *paintengine2d.Context, paintengine2d.Rect, ControlState) {}

// ViewFrameInsets: the list's one-pixel frame inside the room for the
// two-pixel focus border and its one-pixel gap.
func (system7Engine) ViewFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: 4 * u, Right: 4 * u, Bottom: 4 * u, Left: 4 * u}
}

// DrawViewFrame: white inside a one-pixel frame; the list with keyboard
// focus gets a two-pixel border one pixel outside it.
func (e system7Engine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 8 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	ctx.DrawRect(g.at(3, 3, g.w-6, g.h-6), paintengine2d.Fill(c.field))
	var k rpInk
	k.frame(g, 3, 3, g.w-6, g.h-6, 1)
	if st.Focused() && !st.Disabled() {
		k.frame(g, 0, 0, g.w, g.h, 2)
	}
	k.fill(ctx, c.black)
}

// StyleHint: the default button goes last (bottom right), and form labels
// are right-aligned against their fields.
func (system7Engine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintFormLabelsRight:
		return 1
	case HintMnemonics:
		return MnemonicsNever // the Mac has no mnemonics
	}
	return 0
}

// ControlFont: the Mac labels buttons, check boxes, radio buttons, menus,
// pop-ups and tabs in the bold system font (Chicago).
func (system7Engine) ControlFont(l *Classic, role Role) *Font {
	switch role {
	case RoleButton, RoleCheck, RoleMenu, RoleTab, RoleCombo:
		return l.BoldFont()
	}
	return l.body
}

// DrawWindowBackground is the white of a Mac window.
func (system7Engine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(s7colors(l).win))
}

// DrawTabPane is the page under the tabs: white in a one-pixel frame.
func (e system7Engine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.black)
}

// DrawWindowFrame is a Mac document window: a one-pixel frame, the title
// bar — six stripes broken by the close box at the left, the zoom box at
// the right and the bold centred title in its white margin, bare when the
// window is inactive — and a black line under it. System 7 lightens the
// bar, greys the stripes, bevels bar and boxes in lavender and navy, and
// greys the frame and title of an inactive window.
func (e system7Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 8 {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
		return
	}
	bh := s7barH(l)
	if bh+3 > g.h {
		bh = g.h - 3
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	line := c.black
	if c.grey && !st.Active {
		line = c.offLine
	}
	var frame, black, stripe, lav, lavDk, navy rpInk
	frame.frame(g, 0, 0, g.w, g.h, 1)
	frame.cells(g, 1, bh+1, g.w-2, 1)
	if c.grey && st.Active {
		ctx.DrawRect(g.at(1, 1, g.w-2, bh), paintengine2d.Fill(c.bar))
		rpEdge(&lav, &lavDk, g, 1, 1, g.w-2, bh, 1)
	}
	cx, cy, cn := s7closeBox(l)
	closeBox := st.Active && st.CanClose && cx+cn+14 <= g.w && cy+cn <= bh+1
	zoomX := -1
	if st.Active && st.Maximizable && g.w > 80 && cy+cn <= bh+1 {
		zoomX = g.w - 9 - cn
	}
	// The title, centred on the bar and clear of the boxes.
	f := l.BoldFont()
	a, z := 4, g.w-4
	if closeBox {
		a = cx + cn + 4
	}
	if zoomX >= 0 {
		z = zoomX - 4
	}
	tw, tx := 0, 0
	if title != "" && z-a > 12 {
		tw = min(int(f.Advance(title)/u+0.5)+1, z-a-8)
		tx = max(a+4, min((g.w-tw)/2, z-4-tw))
	}
	if st.Active {
		// Six stripes one pixel apart, level with the boxes, broken by the
		// boxes (a pixel each side) and the title's margin.
		var cuts [3][2]int
		nc := 0
		if closeBox {
			cuts[nc] = [2]int{cx - 1, cx + cn + 1}
			nc++
		}
		if tw > 0 {
			cuts[nc] = [2]int{tx - 6, tx + tw + 6}
			nc++
		}
		if zoomX >= 0 {
			cuts[nc] = [2]int{zoomX - 1, zoomX + cn + 1}
			nc++
		}
		ink := &black
		if c.grey {
			ink = &stripe
		}
		for i := 0; i < 6; i++ {
			y, x := cy+2*i, 2
			for j := 0; j <= nc; j++ {
				stop, next := g.w-2, g.w-2
				if j < nc {
					stop, next = cuts[j][0], cuts[j][1]
				}
				if stop > x {
					ink.cells(g, x, y, stop-x, 1)
				}
				x = next
			}
		}
		box := func(x int, pressed, zoom bool) {
			bg := g.sub(x, cy, cn, cn)
			if c.grey {
				// Two rings: navy and lavender, then the reverse, round grey.
				face := c.boxFace
				if pressed {
					face = c.navy
				}
				ctx.DrawRect(bg.rect(), paintengine2d.Fill(face))
				rpEdge(&navy, &lav, bg, 0, 0, cn, cn, 1)
				if !pressed {
					rpEdge(&lav, &navy, bg, 1, 1, cn-2, cn-2, 1)
				}
			} else {
				ctx.DrawRect(bg.rect(), paintengine2d.Fill(c.white))
				black.frame(bg, 0, 0, cn, cn, 1)
			}
			if zoom {
				k := &black
				if c.grey {
					k = &navy
				}
				k.frame(bg, 0, 0, cn*7/11, cn*7/11, 1)
				return
			}
			if pressed {
				// The pressed close box bursts: lines out from its centre.
				m := rpNewMask(cn, cn)
				mid := cn / 2
				m.line(2, mid, cn-3, mid)
				m.line(mid, 2, mid, cn-3)
				m.line(2, 2, cn-3, cn-3)
				m.line(cn-3, 2, 2, cn-3)
				k := &black
				if c.grey {
					k = &lav
				}
				m.emit(k, bg, 0, 0)
			}
		}
		if closeBox {
			box(cx, st.ClosePress, false)
		}
		if zoomX >= 0 {
			box(zoomX, false, true)
		}
	}
	stripe.fill(ctx, c.stripe)
	lavDk.fill(ctx, c.lavDk)
	lav.fill(ctx, c.lav)
	navy.fill(ctx, c.navy)
	black.fill(ctx, c.black)
	frame.fill(ctx, line)
	if tw <= 0 {
		return
	}
	col := c.text
	if c.grey && !st.Active {
		col = c.offTitle
	}
	l.drawFittedText(ctx, f, title, g.at(tx, 1, tw, bh), col, AlignCenter, 0)
}

// ---- controls -------------------------------------------------------------------

// DrawPanel: a raised panel is a white card in a one-pixel frame with the
// Mac's shadow along its right and bottom; a flat one is plain white.
func (e system7Engine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	if !raised || g.w < 4 || g.h < 4 {
		return
	}
	var k rpInk
	k.frame(g, 0, 0, g.w-1, g.h-1, 1)
	k.cells(g, g.w-1, 1, 1, g.h-1)
	k.cells(g, 1, g.h-1, g.w-2, 1)
	k.fill(ctx, c.black)
}

// DrawButton is the Mac push button: black and white on every screen.
func (e system7Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := s7colors(l)
	u := rpU(l)
	g, m, ring, r := s7btnGeom(l, ctx, b)
	if g.w < 6 || g.h < 6 {
		return
	}
	body := g.inset(m)
	s := rpShape{w: body.w, h: body.h, r: r}
	pressed := st.Pressed() && !st.Disabled()
	var face, ink, grey rpInk
	s.fill(&face, body, 0, 0)
	s.frame(&ink, body, 0, 0)
	if st.Primary() && ring > 0 {
		// The default button's outline: three pixels thick, one pixel off,
		// grey when the button is disabled.
		o := rpShape{w: g.w, h: g.h, r: r + float32(m) - 1}
		switch {
		case !st.Disabled():
			o.ring(&ink, g, 0, 0, ring)
		case c.grey:
			o.ring(&grey, g, 0, 0, ring)
		default:
			o.ringDots(&ink, g, 0, 0, ring)
		}
	}
	faceCol, fg := c.white, c.text
	if pressed {
		faceCol, fg = c.black, c.white
	}
	face.fill(ctx, faceCol)
	ink.fill(ctx, c.black)
	grey.fill(ctx, c.dim)
	if st.Focused() && !st.Disabled() {
		// Keyboard focus: a dotted halo one pixel out from the body.
		var dots rpInk
		if m > 0 {
			rpShape{w: body.w + 2, h: body.h + 2, r: r + 1}.frameDots(&dots, g, m-1, m-1)
		} else if body.w > 6 && body.h > 6 {
			rpShape{w: body.w - 4, h: body.h - 4, r: max(r-2, 1)}.frameDots(&dots, body, 2, 2)
		}
		dots.fill(ctx, c.black)
	}
	c.label(l, ctx, l.BoldFont(), label, body.rect().Inset(u*3), fg, AlignCenter, st.Disabled(), faceCol)
}

// toggleLabel draws a check box or radio caption (bold) with the dotted
// focus frame round it; the Mac set the label four pixels after the box.
func (e system7Engine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := s7colors(l)
	if label == "" {
		if st.Focused() {
			e.DrawFocusRing(l, ctx, box.Inset(-l.S(2)).Intersect(b))
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

func (e system7Engine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e system7Engine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch: the Mac had none; this is a small slide switch in the same
// drawing — a stadium track, black when on, and a white knob.
func (e system7Engine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := s7colors(l)
	tw, th := min(l.metrics.SwitchW, b.Dx()), min(l.metrics.SwitchH, b.Dy())
	g := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th), rpU(l))
	if g.w >= 8 && g.h >= 6 {
		track := rpShape{w: g.w, h: g.h, r: float32(g.h) * 0.5}
		knob := rpShape{w: g.h, h: g.h, r: float32(g.h) * 0.5}
		kx := 0
		if on {
			kx = g.w - g.h
		}
		var face, ink, kf rpInk
		track.fill(&face, g, 0, 0)
		track.frame(&ink, g, 0, 0)
		knob.fill(&kf, g, kx, 0)
		knob.frame(&ink, g, kx, 0)
		fill := c.white
		if on {
			fill = c.black
		}
		face.fill(ctx, fill)
		kf.fill(ctx, c.white)
		ink.fill(ctx, c.black)
		if st.Disabled() {
			c.greyOut(l, ctx, g.rect(), c.win)
		}
	}
	e.toggleLabel(l, ctx, b, g.rect(), st, label)
}

// DrawComboBox is the pop-up menu: a one-pixel box with the Mac shadow on
// its right and bottom (starting a few pixels from the corners), the
// current item in bold and the 11 × 6 black triangle at the right. It
// inverts while its menu is open.
func (e system7Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 6 {
		return
	}
	body := g.sub(0, 0, g.w-1, g.h-1)
	inv := (open || st.Pressed()) && !st.Disabled()
	faceCol, fg := c.field, c.text
	if inv {
		faceCol, fg = c.black, c.white
	}
	ctx.DrawRect(body.rect(), paintengine2d.Fill(faceCol))
	var k rpInk
	k.frame(body, 0, 0, body.w, body.h, 1)
	k.cells(g, g.w-1, 3, 1, g.h-3)
	k.cells(g, 2, g.h-1, g.w-3, 1)
	k.fill(ctx, c.black)
	tri := s7tri(min(11, body.h-4), DirDown)
	tx := body.w - tri.w - 4
	var tk rpInk
	tri.emit(&tk, body, tx, (body.h-tri.h)/2)
	tk.fill(ctx, fg)
	lb := body.at(6, 0, tx-10, body.h)
	c.label(l, ctx, l.BoldFont(), text, lb, fg, AlignStart, st.Disabled(), faceCol)
	if st.Disabled() {
		c.greyOut(l, ctx, body.at(tx, 1, tri.w, body.h-2), faceCol)
	}
	if st.Focused() && !open && !st.Disabled() {
		var d rpInk
		d.dotted(body, 2, 2, body.w-4, body.h-4)
		d.fill(ctx, fg)
	}
}

// fieldOpts: inverted selections, outlined when the field lacks focus,
// and a one-pixel insertion point.
func (c *s7) fieldOpts(l *Classic) rpFieldOpts {
	return rpFieldOpts{
		text: l.fieldText(), muted: c.dim, sel: c.sel, selTxt: c.selTxt,
		caret: c.text, caretKind: rpCaretBar, selOffFrame: true,
		ditherOff: !c.grey, ditherMuted: !c.grey, bg: c.field,
	}
}

// DrawTextField: a one-pixel frame round white, the text two pixels inside
// it; focus shows the insertion point.
func (e system7Engine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := s7colors(l)
	u := rpU(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := max(l.metrics.FieldPad, 3*u)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+u, b.Dx()-2*pad, b.Dy()-2*u)
	rpFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.fieldOpts(l))
}

func (e system7Engine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := s7colors(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := max(l.metrics.FieldPad, 3*rpU(l))
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-2*pad, b.Dy()-2*pad)
	rpAreaText(l, ctx, inner, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, c.fieldOpts(l))
}

// DrawSpinner is System 7's "little arrows": a small box split in two with
// a triangle in each half; the held half inverts.
func (e system7Engine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 5 || g.h < 8 {
		return
	}
	w := min(g.w, 13)
	g = g.sub((g.w-w)/2, 0, w, g.h)
	s := rpShape{w: g.w, h: g.h, r: 2}
	var face, ink, inv rpInk
	s.fill(&face, g, 0, 0)
	s.frame(&ink, g, 0, 0)
	mid := g.h / 2
	ink.cells(g, 1, mid, g.w-2, 1)
	pressUp, pressDn := upPress && !st.Disabled(), downPress && !st.Disabled()
	for y := 1; y < g.h-1; y++ {
		if (y < mid && pressUp) || (y > mid && pressDn) {
			e := s.edge(y) + 1
			inv.cells(g, e, y, g.w-2*e, 1)
		}
	}
	face.fill(ctx, c.white)
	inv.fill(ctx, c.black)
	ink.fill(ctx, c.black)
	for i, dir := range [2]Direction{DirUp, DirDown} {
		y0, y1, pressed := 1, mid, pressUp
		if i == 1 {
			y0, y1, pressed = mid+1, g.h-1, pressDn
		}
		tri := s7tri(min(7, g.w-4), dir)
		var tk rpInk
		tri.emit(&tk, g, (g.w-tri.w)/2, y0+(y1-y0-tri.h)/2)
		fg := c.black
		switch {
		case pressed:
			fg = c.white
		case st.Disabled() && c.grey:
			fg = c.dim
		}
		tk.fill(ctx, fg)
	}
	if st.Disabled() {
		c.greyOut(l, ctx, g.inset(1).rect(), c.white)
	}
}

// DrawTabBar: the Mac had no tab control until Mac OS 8; these are
// folder tabs in the same black-and-white drawing, standing on the page's
// top line.
func (e system7Engine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.cells(g, 0, g.h-1, g.w, 1)
	k.fill(ctx, c.black)
}

// DrawTab is a folder tab with round top corners; the selected tab stands
// taller and opens into the page below it.
func (e system7Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 8 || g.h < 8 {
		return
	}
	top := 0
	if !selected {
		top = 3
	}
	tg := g.sub(0, top, g.w, g.h-top)
	s := rpShape{w: tg.w, h: tg.h, r: 5, top: true}
	var face, ink rpInk
	s.fill(&face, tg, 0, 0)
	s.frame(&ink, tg, 0, 0)
	faceCol, fg := c.win, c.text
	if st.Pressed() && !st.Disabled() {
		faceCol, fg = c.black, c.white
	}
	face.fill(ctx, faceCol)
	ink.fill(ctx, c.black)
	if selected {
		ctx.DrawRect(tg.at(1, tg.h-1, tg.w-2, 1), paintengine2d.Fill(faceCol))
	}
	f := l.BoldFont()
	lb := tg.at(2, 1, tg.w-4, tg.h-2)
	c.label(l, ctx, f, label, lb, fg, AlignCenter, st.Disabled(), faceCol)
	if st.Focused() && selected {
		e.DrawFocusRing(l, ctx, rpTextBox(f, label, lb, AlignCenter).Inset(-l.S(3)).Intersect(lb))
	}
}

// DrawMenuBar is the white menu bar with its black bottom line.
func (e system7Engine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.cells(g, 0, g.h-1, g.w, 1)
	k.fill(ctx, c.black)
}

// DrawMenuTitle inverts an open (or pressed) title between the bar's top
// row and its bottom line; titles are bold.
func (e system7Engine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if !st.Disabled() && (open || st.Pressed()) && g.h > 3 {
		ctx.DrawRect(g.at(0, 1, g.w, g.h-2), paintengine2d.Fill(c.black))
		fg = c.white
	}
	c.label(l, ctx, l.BoldFont(), label, b, fg, AlignCenter, st.Disabled(), c.win)
	if st.Focused() && !open {
		e.DrawFocusRing(l, ctx, g.at(1, 1, g.w-2, g.h-2))
	}
}

// DrawMenuFrame is a white menu in a one-pixel black frame (its shadow is
// the PopupShadow).
func (e system7Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.black)
}

// DrawMenuItem: bold items, inverted when hot, the check mark at the left,
// the black hierarchical triangle at the right; separators are a dotted
// line (solid grey on colour screens).
func (e system7Engine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := s7colors(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*u, b.Dy()), u)
	if row.Separator {
		r := full.at(0, full.h/2, full.w, 1)
		if c.grey {
			ctx.DrawRect(r, paintengine2d.Fill(c.sep))
			return
		}
		rpPatFill(ctx, r, rpGray, c.black, u)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	col, bg := c.text, c.win
	if hot {
		ctx.DrawRect(full.rect(), paintengine2d.Fill(c.black))
		col, bg = c.white, c.black
	}
	f := l.BoldFont()
	mark := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked && row.Radio:
		var k rpInk
		rpShape{w: 6, h: 6, r: 3}.fill(&k, mark, (mark.w-6)/2, (mark.h-6)/2)
		k.fill(ctx, col)
	case row.Checked:
		if n := min(11, mark.w-2, mark.h-4); n >= 5 {
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
		tri := s7tri(9, DirRight)
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
		c.label(l, ctx, f, row.Shortcut, paintengine2d.XYWH(sx, b.Min.Y, tw+l.S(2), b.Dy()), col, AlignStart, st.Disabled(), bg)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	c.label(l, ctx, f, row.Label, paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()), col, AlignStart, st.Disabled(), bg)
}

// DrawProgressBar is the Finder's copy bar: a one-pixel frame filled from
// the left (black on white; dark grey on lavender in colour); the
// indeterminate bar is a barber pole of diagonal stripes.
func (e system7Engine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	in := g.inset(1)
	empty, fill := c.white, c.black
	if c.grey {
		empty, fill = c.progEmpty, c.progFill
		if st.Disabled() {
			empty, fill = c.empty, c.dim
		}
	}
	ctx.DrawRect(in.rect(), paintengine2d.Fill(empty))
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.black)
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		rpStripes(ctx, in, fill, int(phase*8))
	} else if n := int(float32(in.w)*clamp1(t) + 0.5); n > 0 {
		ctx.DrawRect(in.at(0, 0, n, in.h), paintengine2d.Fill(fill))
	}
	if st.Disabled() {
		c.greyOut(l, ctx, in.rect(), c.white)
	}
}

// rpStripes fills g with diagonal stripes four cells wide (a barber pole),
// shifted by off cells, in one path.
func rpStripes(ctx *paintengine2d.Context, g rpGrid, col paintengine2d.Color, off int) {
	var k rpInk
	for y := 0; y < g.h; y++ {
		for x := -8; x < g.w+8; x += 8 {
			a := x + (y+off)%8
			s, e := max(a, 0), min(a+4, g.w)
			if e > s {
				k.cells(g, s, y, e-s, 1)
			}
		}
	}
	k.fill(ctx, col)
}

// DrawSlider: the Mac had no slider before Mac OS 8; this one is drawn in
// the 1-bit manner — a thin framed rail and a white pointer that inverts
// while dragged.
func (e system7Engine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 16 || g.h < 10 {
		return
	}
	kw, kh := 11, min(15, g.h-2)
	var ink rpInk
	rail := g.sub(kw/2, g.h/2-2, g.w-kw/2*2, 4)
	ctx.DrawRect(rail.rect(), paintengine2d.Fill(c.white))
	ink.frame(rail, 0, 0, rail.w, rail.h, 1)
	kx := int(float32(g.w-kw)*clamp1(t) + 0.5)
	ky := (g.h - kh) / 2
	// The pointer: a box ending in a point at the bottom.
	m := rpNewMask(kw, kh)
	body := kh - kw/2
	m.rect(0, 0, kw, body)
	for i := 0; i < kw/2+1; i++ {
		m.span(i, kw-i, body+i)
	}
	var face rpInk
	m.emit(&face, g, kx, ky)
	o := m.outline()
	faceCol := c.white
	if st.Pressed() && !st.Disabled() {
		faceCol = c.black
	}
	ink.fill(ctx, c.black)
	face.fill(ctx, faceCol)
	var edge rpInk
	o.emit(&edge, g, kx, ky)
	edge.fill(ctx, c.black)
	if st.Disabled() {
		c.greyOut(l, ctx, g.rect(), c.win)
	}
	if st.Focused() {
		e.DrawFocusRing(l, ctx, g.rect())
	}
}

func (e system7Engine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := l.fieldText()
	if st.Checked() {
		fg = e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
}

// DrawTreeRow is a Finder list row: the disclosure triangle and the name,
// which alone is highlighted when selected.
func (e system7Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := s7colors(l)
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
	fg := l.fieldText()
	if st.Checked() {
		hb := paintengine2d.XYWH(lx-l.S(3), b.Min.Y+l.S(1), f.Advance(label)+l.S(6), b.Dy()-l.S(2)).Intersect(b)
		fg = e.Face(l, ctx, hb, RoleRow, st&^StateFocused)
	}
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.black)
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawTableCell: selected rows invert; in a view without keyboard focus
// the selection is outlined along its top and bottom.
func (e system7Engine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := l.fieldText()
	if st.Checked() {
		if st.Inactive() || st.Backdrop() {
			var k rpInk
			k.cells(g, 0, 0, g.w, 1)
			k.cells(g, 0, g.h-1, g.w, 1)
			col := c.sel
			if c.grey {
				col = c.stripe
			}
			k.fill(ctx, col)
		} else {
			ctx.DrawRect(g.rect(), paintengine2d.Fill(c.sel))
			fg = c.selTxt
		}
	}
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, align, 0)
}

// DrawTableHeader: Finder list headings — plain labels over a black line,
// the sort column's title underlined.
func (e system7Engine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 2 || g.h < 4 {
		return
	}
	faceCol, fg := c.win, c.text
	if st.Pressed() && !st.Disabled() {
		faceCol, fg = c.black, c.white
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(faceCol))
	var k rpInk
	k.cells(g, 0, g.h-1, g.w, 1)
	k.fill(ctx, c.black)
	rpPatFill(ctx, g.at(g.w-1, 2, 1, g.h-4), rpGray, c.black, u)
	aw := float32(0)
	if sorted {
		aw = l.S(10)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()-u), dir, fg)
	}
	f := l.body
	lb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()-u)
	c.label(l, ctx, f, label, lb, fg, AlignStart, st.Disabled(), faceCol)
	if sorted {
		if tb := rpTextBox(f, label, lb, AlignStart).Intersect(lb); !tb.Empty() {
			ctx.DrawRect(rpGridAt(ctx, paintengine2d.XYWH(tb.Min.X, tb.Max.Y-l.S(3), tb.Dx(), u), u).rect(), paintengine2d.Fill(fg))
		}
	}
}

// DrawToolBar: white with a black line under it.
func (e system7Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	e.DrawMenuBar(l, ctx, b)
}

// DrawToolButton: flat until the pointer arrives (a rounded outline);
// pressed and latched tools invert, like MacPaint's tool palette.
func (e system7Engine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := s7colors(l)
	u := rpU(l)
	fg := e.Face(l, ctx, b.Inset(u), RoleTool, st)
	bg := c.win
	if fg == c.white {
		bg = c.black
	}
	if st.Focused() {
		var k rpInk
		g := rpGridAt(ctx, b, u)
		k.dotted(g, 2, 2, g.w-4, g.h-4)
		k.fill(ctx, ReadableOn(bg, 3, c.black, c.white))
	}
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
	if label != "" {
		c.label(l, ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-pad-x, b.Dy()), fg, AlignStart, false, bg)
	}
	if st.Disabled() {
		c.greyOut(l, ctx, b.Inset(u), c.win)
	}
}

// DrawStatusBar is a white strip under a black line, parts divided by
// lines, and the size box at the right end: a small square in front of a
// larger one offset down and right.
func (e system7Engine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	var k rpInk
	k.cells(g, 0, 0, g.w, 1)
	box := min(g.h-1, 16)
	bx := g.w - box
	if len(parts) > 0 {
		slot := float32(bx) / float32(len(parts))
		for i, s := range parts {
			x := int(slot * float32(i))
			if i > 0 {
				k.cells(g, x, 1, 1, g.h-1)
			}
			l.drawFittedText(ctx, l.body, s, g.at(x+6, 1, int(slot)-10, g.h-1), c.text, AlignStart, 0)
		}
	}
	if box >= 13 {
		k.cells(g, bx, 1, 1, g.h-1)
		sg := g.sub(bx+1, 1+(g.h-1-box)/2, box-1, box)
		k.frame(sg, 4, 4, 9, 9, 1)
		ctx.DrawRect(sg.at(2, 2, 7, 7), paintengine2d.Fill(c.win))
		k.frame(sg, 2, 2, 7, 7, 1)
	}
	k.fill(ctx, c.black)
}

// DrawTitleBar is a panel heading: the bold title over a black line.
func (e system7Engine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := s7colors(l)
	u := rpU(l)
	e.DrawMenuBar(l, ctx, b)
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		// Secondary text is grey: a dither on 1-bit screens.
		c.label(l, ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.text, AlignStart, true, c.win)
	}
}

// DrawAccordionHeader: a disclosure triangle and a bold title over a
// dotted rule; pressed inverts.
func (e system7Engine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	faceCol, fg := c.win, c.text
	if st.Pressed() && !st.Disabled() {
		faceCol, fg = c.black, c.white
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(faceCol))
	rpPatFill(ctx, g.at(0, g.h-1, g.w, 1), rpGray, c.black, u)
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, fg)
	c.label(l, ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()-u), fg, AlignStart, st.Disabled(), faceCol)
	if st.Focused() {
		e.DrawFocusRing(l, ctx, g.at(0, 0, g.w, g.h-1))
	}
}

// DrawSeparator is a grey line: the 50% pattern on 1-bit screens.
func (e system7Engine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := s7colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	r := g.at(0, g.h/2, g.w, 1)
	if vertical {
		r = g.at(g.w/2, 2, 1, g.h-4)
	}
	if c.grey {
		ctx.DrawRect(r, paintengine2d.Fill(c.sep))
		return
	}
	rpPatFill(ctx, r, rpGray, c.black, u)
}

// DrawSplitter: a black line, doubled under the pointer.
func (e system7Engine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.win))
	n := 1
	if st.Hovered() || st.Pressed() {
		n = 2
	}
	var k rpInk
	if vertical {
		k.cells(g, (g.w-n)/2, 0, n, g.h)
	} else {
		k.cells(g, 0, (g.h-n)/2, g.w, n)
	}
	k.fill(ctx, c.black)
}

// DrawTooltip is Balloon Help (System 7): a white rounded balloon in a
// one-pixel black line.
func (e system7Engine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := s7colors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 4 || g.h < 4 {
		return
	}
	s := rpShape{w: g.w, h: g.h, r: s7balloonR(g)}
	var face, ink rpInk
	s.fill(&face, g, 0, 0)
	s.frame(&ink, g, 0, 0)
	face.fill(ctx, c.info)
	ink.fill(ctx, c.black)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), ReadableOn(c.info, 4.5, c.text), AlignStart, 0)
}

// DrawOverlay: the Mac never dimmed the screen behind a modal dialog.
func (system7Engine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- packs --------------------------------------------------------------------

func system7Pack(name, label string, year int, summary string, pal Palette, extra map[string]string, params map[string]float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "system7",
		Bevel:   BevelNone,
		Family:  ThemeLight,
		Palette: pal,
		Params:  params,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.15), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Mac OS", Summary: summary,
		Era: "Classic Macintosh", Palette: ThemeLight, Tokens: tok,
	}
}

// s7palette is a white Mac palette; face is the chrome grey, dim the grey
// of disabled text.
func s7palette(face, dim, danger, success, warning string) Palette {
	white, black := hexColor("#ffffff"), hexColor("#000000")
	return Palette{
		Background: white, Surface: white, SurfaceAlt: hexColor(face),
		Border: black, Divider: black,
		Text: black, TextMuted: hexColor(dim), TextOnAccent: white,
		Accent: black, AccentHover: black, AccentPress: black,
		Field: white, FieldBorder: black,
		Focus: black, Selection: black,
		Track: hexColor("#bbbbbb"), Thumb: white,
		Highlight: white, Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: black, MenuHoverBorder: black, MenuGutter: white,
		Danger: hexColor(danger), Success: hexColor(success), Warning: hexColor(warning),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: white, BevelDark: black,
	}
}

func system7Packs() []ThemePack {
	return []ThemePack{
		system7Pack("system1", "System 1", 1984,
			"The 1984 Finder: 1-bit black on white, round-rect buttons, striped title bars and dithered greys.",
			s7palette("#ffffff", "#666666", "#000000", "#000000", "#000000"), nil, nil),
		system7Pack("system7", "System 7", 1991,
			"System 7 in colour: the black-and-white controls in the Standard window colours, lavender, navy and grey.",
			s7palette("#eeeeee", "#808080", "#cc0000", "#007700", "#aa6600"),
			map[string]string{
				"bar": "#eeeeee", "stripe": "#777777", "lav": "#ccccff", "lavDk": "#a3a3d7", "navy": "#333366",
				"boxFace": "#a4a4a4", "arrowFace": "#dddddd", "arrowEdge": "#777777", "arrowFill": "#a3a3d7",
				"thumb": "#aaaaaa", "grip": "#666699", "trackBg": "#dddddd", "trackDot": "#777777", "empty": "#eeeeee",
				"progFill": "#404040", "progEmpty": "#ccccff", "offLine": "#555555", "offTitle": "#808080",
				"sep": "#808080", "info": "#ffffcc",
			},
			map[string]float32{"grey": 1}),
	}
}
