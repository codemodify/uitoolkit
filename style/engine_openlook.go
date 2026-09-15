package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// openlookEngine paints OPEN LOOK, the look Sun and AT&T specified in 1988
// and Sun shipped as OpenWindows, in its 3D colour rendition: every part is
// the window colour BG1 with a one-pixel highlight on its upper left and a
// BG3 shadow on its lower right, recessed (edges swapped, BG2 fill) while
// pressed or selected.
//
//   - Buttons are obround — straight sides between rounded ends — and the
//     default button carries a second ring inside its border. Menu buttons
//     add the menu mark, a small triangle; the abbreviated menu button is a
//     small square holding only the mark (here: the combo box, with the
//     current value beside it).
//   - Check boxes are raised squares whose bold check mark leaves the box
//     at the top right. Exclusive settings (radio buttons, and the tab
//     strip) are rectangles; the chosen one is recessed.
//   - The scrollbar is an elevator — a box of three cells, two arrows round
//     a drag area — riding a thin cable that shows the visible proportion,
//     between two cable anchors.
//   - Text fields have no box: the text sits on an engraved underline and
//     the caret is a small triangle.
//   - Pop-up windows carry the pushpin at the left of their header and
//     L-shaped resize corners; menus highlight a recessed obround.
//   - Inactive controls are overlaid with a 50% checkerboard of the window
//     colour.
//
// Sizes follow OpenWindows 3 at its 12-point scale; colours derive from the
// window colour as OLGX did (BG2 90%, BG3 50% and the highlight 120% of its
// brightness).
//
// Pack data: the window colour is Palette.SurfaceAlt; extra "bg2", "bg3"
// and "hi" override the derived shades.
type openlookEngine struct{ BaseEngine }

func init() {
	RegisterEngine(openlookEngine{})
	for _, p := range openlookPacks() {
		RegisterPack(p)
	}
}

func (openlookEngine) ID() string { return "openlook" }

// DefaultMetrics: OpenWindows at 12pt scaled to the toolkit's 16px font —
// 20px buttons become 28px, 13px check boxes 16px; the 19px scrollbar keeps
// its native size.
func (openlookEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Square:    true, BevelDepth: 1,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 16, Radio: 16,
		MenuItemH: 26, MenuBarH: 32, TabH: 30, RowH: 22,
		TitleBar: 28, HeaderH: 24, ProgressH: 15, SliderH: 22, Thumb: 11,
		Scroll: 19, Pad: 10, FieldPad: 4, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 18, SwitchW: 36, SwitchH: 18,
	}
}

// ---- colours ------------------------------------------------------------------

type olc struct {
	bg1, bg2, bg3, hi              paintengine2d.Color
	black, white, text, dim, field paintengine2d.Color
	sel, selTxt                    paintengine2d.Color
}

type olKey struct{}

func olColors(l *Classic) *olc {
	return l.Memo(olKey{}, func() any { return olBuild(l) }).(*olc)
}

// olShade scales the HSV value of c by k the way OLGX derived its 3D
// colours: a value past 1 is capped and the saturation halved instead.
func olShade(c paintengine2d.Color, k float64) paintengine2d.Color {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	v := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	if v <= 0 {
		return paintengine2d.RGB(0, 0, 0)
	}
	s := (v - mn) / v
	nv := v * k
	if nv > 1 {
		nv = 1
		s *= 0.5
	}
	// Rebuild with the same hue: each channel keeps its place between the
	// new minimum and maximum.
	nmn := nv * (1 - s)
	f := func(ch float64) float32 {
		if v == mn {
			return float32(nv)
		}
		t := (ch - mn) / (v - mn)
		// Truncate to 8 bits as OLGX's integer arithmetic did.
		return float32(math.Floor((nmn+t*(nv-nmn))*255) / 255)
	}
	if v == mn {
		q := float32(math.Floor(nv*255) / 255)
		return paintengine2d.RGB(q, q, q)
	}
	return paintengine2d.RGB(f(r), f(g), f(b))
}

func olBuild(l *Classic) *olc {
	p := l.palette
	bg1 := p.SurfaceAlt
	c := &olc{
		bg1:   bg1,
		bg2:   l.X("bg2", olShade(bg1, 0.9)),
		bg3:   l.X("bg3", olShade(bg1, 0.5)),
		hi:    l.X("hi", olShade(bg1, 1.2)),
		black: paintengine2d.RGB(0, 0, 0),
		white: paintengine2d.RGB(1, 1, 1),
		text:  p.Text,
		dim:   p.TextMuted,
		field: p.Field,
		sel:   p.Selection,
	}
	if c.sel.A < 0.9 {
		c.sel = c.black
	}
	c.selTxt = ReadableOn(c.sel, 4.5, c.white, c.black)
	return c
}

// ---- drawing helpers -------------------------------------------------------------

// olBox paints a raised (or recessed) 3D rectangle of w × h cells at
// (x, y) of g: fill, then highlight on the upper left and BG3 on the lower
// right (swapped when recessed).
func (c *olc) box(ctx *paintengine2d.Context, g rpGrid, x, y, w, h int, recessed bool, fill paintengine2d.Color) {
	if w < 2 || h < 2 {
		return
	}
	ctx.DrawRect(g.at(x, y, w, h), paintengine2d.Fill(fill))
	var hi, lo rpInk
	if recessed {
		rpEdge(&lo, &hi, g, x, y, w, h, 1)
	} else {
		rpEdge(&hi, &lo, g, x, y, w, h, 1)
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.bg3)
}

// olObround is the OPEN LOOK button shape in a w × h cell box: straight
// sides between elliptical end caps — on the 19-pixel button they are 9
// pixels across and curve over 6 rows, leaving a 7-pixel straight run.
func olObround(w, h int) rpShape {
	r := float32(h) * 6 / 19
	if r < 2 {
		r = 2
	}
	return rpShape{w: w, h: h, r: r, rx: float32(h) * 9 / 19}
}

// obround paints the 3D obround s at (x, y) of g: the highlight takes the
// top edge, the left end but its last three rows and the right end's top
// three rows (on the 19-pixel button; the split scales with the height);
// BG3 takes the rest (swapped when recessed).
func (c *olc) obround(ctx *paintengine2d.Context, g rpGrid, s rpShape, x, y int, recessed bool, fill paintengine2d.Color) {
	var face, hi, lo rpInk
	s.fill(&face, g, x, y)
	face.fill(ctx, fill)
	split := max(3, s.h*3/19)
	for row := 0; row < s.h; row++ {
		a1, b1, a2, b2, ok := s.frameRuns(row)
		if !ok {
			continue
		}
		top := row < s.h/2
		switch {
		case row == 0 || (a2 == 0 && b2 == 0 && top):
			hi.cells(g, x+a1, y+row, b1-a1, 1)
		case row == s.h-1 || (a2 == 0 && b2 == 0):
			lo.cells(g, x+a1, y+row, b1-a1, 1)
		default:
			if row < s.h-split {
				hi.cells(g, x+a1, y+row, b1-a1, 1)
			} else {
				lo.cells(g, x+a1, y+row, b1-a1, 1)
			}
			if row < split {
				hi.cells(g, x+a2, y+row, b2-a2, 1)
			} else {
				lo.cells(g, x+a2, y+row, b2-a2, 1)
			}
		}
	}
	if recessed {
		hi, lo = lo, hi
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.bg3)
}

// olMark is the menu mark: a small triangle of base n cells pointing dir.
func olMark(n int, dir Direction) rpMask {
	return s7tri(n, dir)
}

// mark paints the 3D engraved menu mark (BG3 on its upper left edges, the
// highlight on the lower right, BG2 inside) centred at cell (cx, cy).
func (c *olc) mark(ctx *paintengine2d.Context, g rpGrid, cx, cy, n int, dir Direction, recessed bool) {
	m := olMark(n, dir)
	ox, oy := cx-m.w/2, cy-m.h/2
	var fill rpInk
	m.emit(&fill, g, ox, oy)
	fill.fill(ctx, c.bg2)
	o := m.outline()
	// Split the outline: cells nearer the top-left are shadow, the rest light.
	var lo, hi rpInk
	for y := 0; y < o.h; y++ {
		for x := 0; x < o.w; x++ {
			if !o.get(x, y) {
				continue
			}
			var dark bool
			switch dir {
			case DirDown:
				dark = y == 0 || x < o.w/2
			case DirRight:
				dark = x == 0 || y < o.h/2
			case DirUp:
				dark = x < o.w/2 && y != o.h-1
			default:
				dark = y < o.h/2 && x != o.w-1
			}
			if recessed {
				dark = !dark
			}
			if dark {
				lo.cells(g, ox+x, oy+y, 1, 1)
			} else {
				hi.cells(g, ox+x, oy+y, 1, 1)
			}
		}
	}
	lo.fill(ctx, c.bg3)
	hi.fill(ctx, c.hi)
}

// olInactive overlays r with a 50% checkerboard of the window colour, how
// OPEN LOOK dims an inactive control.
func (c *olc) inactive(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect) {
	rpPatFill(ctx, r, rpGray, c.bg1, rpU(l))
}

// olButtonBody is the obround body of a button inside b (one cell of air
// above and below it, as in the 20px OPEN LOOK button cell).
func olButtonBody(g rpGrid) (x, y, w, h int) {
	h = g.h - 2
	if h > 30 {
		h = 30
	}
	if h < 6 {
		h = g.h
	}
	return 0, (g.h - h) / 2, g.w, h
}

// ---- parts --------------------------------------------------------------------

func (e openlookEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	fg := c.text
	if g.w < 3 || g.h < 3 {
		return fg
	}
	pressed := st.Pressed() && !st.Disabled()
	switch role {
	case RoleButton, RoleCombo:
		x, y, w, h := olButtonBody(g)
		fill := c.bg1
		if pressed {
			fill = c.bg2
		}
		c.obround(ctx, g, olObround(w, h), x, y, pressed, fill)
		return fg
	case RoleTool:
		switch {
		case st.Disabled():
		case pressed || (st.Toggle() && st.Checked()):
			c.box(ctx, g, 0, 0, g.w, g.h, true, c.bg2)
		case st.Hovered():
			c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
		}
		return fg
	case RoleField:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.field))
		c.etched(ctx, g)
		return l.fieldText()
	case RoleCheck:
		c.box(ctx, g, 0, 0, g.w, g.h, pressed, c.bg1)
		return fg
	case RoleRow:
		if st.Checked() {
			c.box(ctx, g, 0, 0, g.w, g.h, true, c.bg2)
		}
		return fg
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			c.obround(ctx, g, olObround(g.w, g.h), 0, 0, true, c.bg2)
		}
		return fg
	case RoleThumb:
		c.box(ctx, g, 0, 0, g.w, g.h, pressed, c.bg1)
		return fg
	case RoleTrack:
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
		rpPatFill(ctx, g.rect(), rpGray, c.bg3, rpU(l))
		return fg
	case RoleTab:
		return fg
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	return fg
}

// etched is the 2px border of scrolling lists and text windows: a
// recessed outer line and a raised inner one.
func (c *olc) etched(ctx *paintengine2d.Context, g rpGrid) {
	var hi, lo rpInk
	rpEdge(&lo, &hi, g, 0, 0, g.w, g.h, 1)
	rpEdge(&hi, &lo, g, 1, 1, g.w-2, g.h-2, 1)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.bg3)
}

// CheckIndicator: a raised square (recessed while pressed) whose thick
// check mark leaves the box at the top right.
func (e openlookEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, box, u)
	n := min(g.w, g.h)
	if n < 7 {
		return
	}
	g = g.centered(n, n)
	bs := n - 2 // the box, with room above and right for the overshoot
	pressed := st.Pressed() && !st.Disabled()
	c.box(ctx, g, 0, 2, bs, bs, pressed, c.bg1)
	if checked {
		m := rpNewMask(n, n)
		px, py := bs*9/20, n-3 // the tick's point, low in the box
		sx, sy := 2, 2+bs*9/20
		for d := 0; d < 3; d++ {
			m.line(sx+d, sy, px+d, py)
		}
		for d := 0; d < 3; d++ {
			m.line(px+d-1, py, n-1-2+d, 0)
		}
		var k rpInk
		m.emit(&k, g, 0, 0)
		k.fill(ctx, c.black)
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// RadioIndicator is a small exclusive setting: a raised square, recessed
// and filled BG2 when chosen.
func (e openlookEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := olColors(l)
	g := rpGridAt(ctx, box, rpU(l))
	n := min(g.w, g.h)
	if n < 5 {
		return
	}
	g = g.centered(n, n)
	on := selected || (st.Pressed() && !st.Disabled())
	fill := c.bg1
	if on {
		fill = c.bg2
	}
	c.box(ctx, g, 0, 0, n, n, on, fill)
	if selected {
		var k rpInk
		k.frame(g, 1, 1, n-2, n-2, 1)
		k.fill(ctx, c.bg3)
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// Arrow is the OPEN LOOK arrow: a solid triangle with a flat tip.
func (e openlookEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	g := rpGridAt(ctx, b, rpU(l))
	n := min(g.w, g.h)
	if n < 4 {
		return
	}
	w := n * 2 / 3
	if w > 9 {
		w = 9
	}
	m := olArrowMask(w)
	m = m.turn(dir)
	var k rpInk
	m.emit(&k, g, (g.w-m.w)/2, (g.h-m.h)/2)
	k.fill(ctx, col)
}

// olArrowMask is an up-pointing triangle w cells wide with a flat tip two
// cells across (the elevator's arrows are 8 wide and 5 tall).
func olArrowMask(w int) rpMask {
	if w < 4 {
		w = 4
	}
	if w%2 == 1 {
		w++
	}
	h := w/2 + 1
	if h > w-3 {
		h = w - 3
	}
	m := rpNewMask(w, h)
	for i := 0; i < h; i++ {
		inset := (w/2 - 1) * (h - 1 - i) / max(h-1, 1)
		m.span(inset, w-inset, i)
	}
	return m
}

// Expander is the menu mark pointing right, down when open.
func (e openlookEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 7 || g.h < 7 {
		return
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	c.mark(ctx, g, g.w/2, g.h/2, 9, dir, false)
}

// MenuHighlight: the item under the pointer is a recessed obround.
func (e openlookEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 6 || g.h < 4 {
		return
	}
	c.obround(ctx, g, olObround(g.w, g.h), 0, 0, true, c.bg2)
}

func (e openlookEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return olColors(l).text
}

// FieldFocusRing: text fields show focus with the caret.
func (openlookEngine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing: a dotted rectangle just inside b.
func (e openlookEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	g := rpGridAt(ctx, b, rpU(l))
	var k rpInk
	k.dotted(g, 0, 0, g.w, g.h)
	k.fill(ctx, olColors(l).black)
}

// ---- scrollbars ---------------------------------------------------------------

// ScrollBarStyle: 19px bars. The cable anchors at the ends are the step
// buttons (6px boxes and a 2px gap); the thumb is at least as long as the
// 47px elevator, which is drawn at the thumb's place along the cable.
func (openlookEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 19, Arrows: ArrowsEnds, ArrowLen: 8, MinThumb: 47, FixedThumb: 47}
}

func (e openlookEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, p.Bar, u)
	if g.w < 5 || g.h < 5 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
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
	pos := func(r paintengine2d.Rect) (int, int) { // start and length along, in cells
		if vertical {
			return int((r.Min.Y-g.y)/u + 0.5), int(r.Dy()/u + 0.5)
		}
		return int((r.Min.X-g.x)/u + 0.5), int(r.Dx()/u + 0.5)
	}
	ew := across - 4 // the elevator's width: two cells of margin each side
	if ew < 5 {
		ew = across
	}
	mx := (across - ew) / 2
	active := !st.Disabled && !p.Thumb.Empty()
	// Cable anchors.
	lo, hi := 0, along
	anchor := func(r paintengine2d.Rect, part ScrollPart, first bool) {
		if r.Empty() {
			return
		}
		a0, n := pos(r)
		h := min(6, n)
		y := a0
		if first {
			lo = a0 + n
		} else {
			y = a0 + n - h
			hi = a0
		}
		ag := rpGridAt(ctx, cell(y, mx, h, ew), u)
		pressed := st.Pressed == part && active
		fill := c.bg1
		if pressed {
			fill = c.bg2
		}
		c.box(ctx, ag, 0, 0, ag.w, ag.h, pressed, fill)
	}
	anchor(p.Dec, ScrollDec, true)
	anchor(p.Inc, ScrollInc, false)
	// The cable: three cells wide, a 50% checkerboard of BG3, solid BG3
	// over the visible proportion.
	cw := 3
	cx := (across - cw) / 2
	if hi-lo > 0 {
		rpPatFill(ctx, cell(lo, cx, hi-lo, cw), rpGray, c.bg3, u)
	}
	if !active {
		return
	}
	t0, tn := pos(p.Proportion)
	ctx.DrawRect(cell(t0, cx, tn, cw), paintengine2d.Fill(c.bg3))
	// The elevator: three cells (arrow, drag area, arrow) of ew rows each,
	// one-cell lines between them, at the thumb's fraction of the track.
	el := 3*ew + 2
	tr0, trn := pos(p.Track)
	if el > trn {
		el = trn
	}
	if el < 7 {
		return
	}
	f := float32(0)
	if free := p.Track.Dy() - p.Thumb.Dy(); vertical && free > 0.5 {
		f = (p.Thumb.Min.Y - p.Track.Min.Y) / free
	} else if free := p.Track.Dx() - p.Thumb.Dx(); !vertical && free > 0.5 {
		f = (p.Thumb.Min.X - p.Track.Min.X) / free
	}
	a := tr0 + int(clamp1(f)*float32(trn-el)+0.5)
	eg := rpGridAt(ctx, cell(a, mx, el, ew), u)
	ctx.DrawRect(eg.rect(), paintengine2d.Fill(c.bg1))
	var hiK, loK rpInk
	rpEdge(&hiK, &loK, eg, 0, 0, eg.w, eg.h, 1)
	seg := (el - 2) / 3
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	// Lines round the drag cell: a highlight above it, BG3 below.
	line := func(at int, k *rpInk) {
		if vertical {
			k.cells(eg, 1, at, eg.w-2, 1)
		} else {
			k.cells(eg, at, 1, 1, eg.h-2)
		}
	}
	line(seg, &loK)
	line(seg+1, &hiK)
	line(2*seg+1, &loK)
	line(2*seg+2, &hiK)
	hiK.fill(ctx, c.hi)
	loK.fill(ctx, c.bg3)
	sub := func(i int) rpGrid { // cell i of the elevator
		start := [3]int{0, seg + 2, 2*seg + 3}[i]
		n := seg
		if i == 2 {
			n = el - start
		}
		if vertical {
			return eg.sub(0, start, eg.w, n)
		}
		return eg.sub(start, 0, n, eg.h)
	}
	atStart := t0 <= tr0
	atEnd := t0+tn >= tr0+trn
	for i, dir := range []Direction{dec, inc} {
		sg := sub(i * 2)
		m := olArrowMask(8).turn(dir)
		var k rpInk
		m.emit(&k, sg, (sg.w-m.w)/2, (sg.h-m.h)/2)
		col := c.bg3
		if (i == 0 && atStart) || (i == 1 && atEnd) {
			col = Mix(c.bg3, c.bg1, 0.6) // nothing more to scroll that way
		}
		k.fill(ctx, col)
	}
	if st.Pressed == ScrollThumbPart {
		// Dragging shows the dimple: a small recessed circle in the drag cell.
		mg := sub(1)
		d := 6
		if d > min(mg.w, mg.h)-2 {
			d = min(mg.w, mg.h) - 2
		}
		if d >= 3 {
			s := rpShape{w: d, h: d, r: float32(d) * 0.5}
			var face, edge rpInk
			ox, oy := (mg.w-d)/2, (mg.h-d)/2
			s.fill(&face, mg, ox, oy)
			face.fill(ctx, c.bg2)
			s.frame(&edge, mg, ox, oy)
			edge.fill(ctx, c.bg3)
		}
	}
}

// DrawScrollBar paints a bare bar: the cable and the elevator.
func (e openlookEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------

func (openlookEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox: an etched frame with the bold title set into its top line.
func (e openlookEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := olColors(l)
	u := rpU(l)
	f := l.BoldFont()
	ctx.DrawRect(b, paintengine2d.Fill(c.bg1))
	top := b.Min.Y
	if title != "" {
		top += f.Height() * 0.5
	}
	g := rpGridAt(ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, u)
	if raised {
		c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
	} else if g.w > 3 && g.h > 3 {
		var hi, lo rpInk
		rpEdge(&lo, &hi, g, 0, 0, g.w, g.h, 1)
		rpEdge(&hi, &lo, g, 1, 1, g.w-2, g.h-2, 1)
		hi.fill(ctx, c.hi)
		lo.fill(ctx, c.bg3)
	}
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
	ctx.DrawRect(paintengine2d.XYWH(snap(tx), b.Min.Y, snap(tw), f.Height()), paintengine2d.Fill(c.bg1))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// olHeaderH is the header band of a window frame in cells (15px at 12pt,
// grown with the bold font); olFrame the side and bottom border (5px).
func olHeaderH(l *Classic) int {
	u := rpU(l)
	h := int(l.BoldFont().Height()/u+0.5) + 2
	if h < 15 {
		h = 15
	}
	return h
}

const olFrame = 5

func (openlookEngine) WindowFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: float32(olFrame*2+olHeaderH(l)) * u, Right: olFrame * u, Bottom: olFrame * u, Left: olFrame * u}
}

// olPin is the pushpin box of a frame's header, in cells of g: 14 cells
// in, level with the header (its pulled-out width, 26 × 13 at 12pt).
func olPin(l *Classic, g rpGrid) (x, y, w, h int) {
	k := olPinScale(l)
	w, h = int(26*k+0.5), int(13*k+0.5)
	return 14, olFrame + (olHeaderH(l)-h)/2, w, h
}

// WindowCloseRect is the pushpin: pulling it out dismisses a pop-up.
func (e openlookEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	g := rpGridOf(b, rpU(l))
	if g.w < 60 || g.h < olHeaderH(l)+olFrame*2+4 {
		return paintengine2d.Rect{}
	}
	x, y, w, h := olPin(l, g)
	return g.at(x, y, w, h)
}

// PopupShadow: OpenWindows 3D menus, notices and pop-ups cast no shadow.
func (openlookEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (openlookEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {
}

// ItemFocus: a dotted rectangle round the current row.
func (e openlookEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	g := rpGridAt(ctx, b, rpU(l))
	var k rpInk
	k.dotted(g, 1, 1, g.w-2, g.h-2)
	k.fill(ctx, olColors(l).black)
}

// ViewFrameInsets: the scrolling list's 2px etched border.
func (openlookEngine) ViewFrameInsets(l *Classic) Insets {
	u := rpU(l)
	return Insets{Top: 2 * u, Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// DrawViewFrame: lists sit on the window colour inside the etched border.
func (e openlookEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	c.etched(ctx, g)
}

// StyleHint: notices put the default (affirmative) button first, and
// property windows right-align their labels against the controls.
func (openlookEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintDialogPrimaryFirst, HintFormLabelsRight:
		return 1
	}
	return 0
}

// DrawWindowBackground is the window colour.
func (openlookEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(olColors(l).bg1))
}

// DrawTabPane: the page under the exclusive settings that pick it.
func (e openlookEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
}

// olPinScale is how much the pushpin grows with the header: OPEN LOOK drew
// a 13-pixel pin in a 15-pixel header.
func olPinScale(l *Classic) float32 {
	return max(1, float32(olHeaderH(l)-2)/13)
}

// pushpin paints the pushpin at (x, y) of g, k times its 12-point size:
// stuck in, seen end on (a round head over a wide flat collar, 13 cells
// square), or pulled out and lying on its side (26 × 13: the needle
// pointing left, a thin collar, the barrel and a round knob, the hole it
// left as a small ring). Every part is lit from the upper left:
// highlight there, black on the lower right, the window colour inside.
func (c *olc) pushpin(ctx *paintengine2d.Context, g rpGrid, x, y int, in bool, k float32) {
	n := func(v float32) int { return max(1, int(v*k+0.5)) }
	var body, hi, lo rpInk
	flush := func() {
		body.fill(ctx, c.bg1)
		hi.fill(ctx, c.hi)
		lo.fill(ctx, c.black)
		body, hi, lo = rpInk{}, rpInk{}, rpInk{}
	}
	shade := func(m rpMask, ox, oy int) {
		o := m.outline()
		inner := m.minus(o)
		inner.emit(&body, g, x+ox, y+oy)
		for yy := 0; yy < o.h; yy++ {
			for xx := 0; xx < o.w; xx++ {
				if !o.get(xx, yy) {
					continue
				}
				if !m.get(xx, yy-1) || !m.get(xx-1, yy) {
					hi.cells(g, x+ox+xx, y+oy+yy, 1, 1)
				} else {
					lo.cells(g, x+ox+xx, y+oy+yy, 1, 1)
				}
			}
		}
	}
	if in {
		cw, ch, hd := n(13), n(5), n(9)
		collar := rpNewMask(cw, ch)
		collar.shape(rpShape{w: cw, h: ch, r: float32(ch) * 0.5, rx: float32(cw) * 0.5}, 0, 0)
		head := rpNewMask(hd, hd)
		head.shape(rpShape{w: hd, h: hd, r: float32(hd) * 0.5}, 0, 0)
		// The head sits on the collar, over its far half.
		shade(collar, 0, n(13)-ch)
		flush()
		shade(head, (cw-hd)/2, n(13)-ch/2-hd)
	} else {
		pw, ph := n(26), n(13)
		pin := rpNewMask(pw, ph)
		mid := ph / 2
		pin.rect(n(2), mid, n(9)-n(2), n(2))                                       // the needle
		pin.rect(0, mid, n(2), max(1, n(2)/2))                                     // its point
		pin.shape(rpShape{w: n(3), h: ph, r: float32(n(3)) * 0.5}, n(9), 0)        // the collar, edge on
		pin.rect(n(12), n(4), n(7), n(6))                                          // the barrel
		pin.shape(rpShape{w: pw - n(19), h: n(10), r: float32(n(3))}, n(19), n(2)) // the knob
		shade(pin, 0, 0)
		hs := n(4)
		hole := rpNewMask(hs, hs)
		hole.shape(rpShape{w: hs, h: hs, r: float32(hs) * 0.5}, 0, 0)
		ring := hole.outline()
		ring.emit(&lo, g, x, y+ph-hs)
	}
	flush()
}

// DrawWindowFrame is an olwm pop-up window: a two-pixel black outline
// round a 5px border of the window colour, the pushpin at the left of the
// header and the bold title centred in it; the header of the window with
// the input focus is recessed. L-shaped resize corners mark the corners.
func (e openlookEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	hh := olHeaderH(l)
	if g.w < 2*olFrame+8 || g.h < 2*olFrame+hh+2 {
		return
	}
	var black rpInk
	black.frame(g, 0, 0, g.w, g.h, 2)
	// Resize corners: L-shapes of 11 cells with 5-cell arms.
	if g.w > 40 && g.h > 40 {
		for _, cr := range [4][2]int{{0, 0}, {g.w - 11, 0}, {0, g.h - 11}, {g.w - 11, g.h - 11}} {
			m := rpNewMask(11, 11)
			fx, fy := cr[0] == 0, cr[1] == 0
			arm := func(x, y, w, h int) { m.rect(x, y, w, h) }
			if fy {
				arm(0, 0, 11, 5)
			} else {
				arm(0, 6, 11, 5)
			}
			if fx {
				arm(0, 0, 5, 11)
			} else {
				arm(6, 0, 5, 11)
			}
			var face rpInk
			m.emit(&face, g, cr[0], cr[1])
			face.fill(ctx, c.bg1)
			o := m.outline()
			o.emit(&black, g, cr[0], cr[1])
		}
	}
	header := g.sub(olFrame, olFrame, g.w-2*olFrame, hh)
	if st.Active {
		c.box(ctx, header, 0, 0, header.w, header.h, true, c.bg2)
	}
	black.fill(ctx, c.black)
	left := olFrame + 4
	if st.CanClose {
		if cr := e.WindowCloseRect(l, b); !cr.Empty() {
			x, y, w, _ := olPin(l, g)
			c.pushpin(ctx, g, x, y, !st.ClosePress, olPinScale(l))
			left = x + w + 4
		}
	}
	if title == "" {
		return
	}
	f := l.BoldFont()
	tb := header.rect()
	// Centred on the header, kept clear of the pin.
	tw := f.Advance(title)
	tx := tb.Min.X + (tb.Dx()-tw)*0.5
	if lx := g.x + float32(left)*u; tx < lx {
		tx = lx
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx, tb.Min.Y, tb.Max.X-tx-2*u, tb.Dy()), c.text, AlignStart, 0)
}

// ---- controls -----------------------------------------------------------------

// DrawPanel: raised panels are 3D; flat ones the plain window colour.
func (e openlookEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if raised {
		c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
}

// DrawButton is the obround OPEN LOOK button; the default button has a
// second ring inside its border.
func (e openlookEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 6 {
		return
	}
	x, y, w, h := olButtonBody(g)
	pressed := st.Pressed() && !st.Disabled()
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	s := olObround(w, h)
	c.obround(ctx, g, s, x, y, pressed, fill)
	if st.Primary() && w > 8 && h > 8 {
		var k rpInk
		rpShape{w: w - 4, h: h - 4, r: s.r - 2, rx: s.rx - 2}.frame(&k, g, x+2, y+2)
		k.fill(ctx, c.bg3)
	}
	lb := g.at(x+3, y, w-6, h)
	l.drawFittedText(ctx, l.body, label, lb, c.text, AlignCenter, l.S(6))
	if st.Focused() && !st.Disabled() && y > 0 {
		// Keyboard focus: a dotted obround in the cell of air round it.
		var d rpInk
		rpShape{w: w, h: h + 2, r: s.r + 1, rx: s.rx}.frameDots(&d, g, x, y-1)
		d.fill(ctx, c.black)
	} else if st.Focused() && !st.Disabled() {
		e.DrawFocusRing(l, ctx, rpTextBox(l.body, label, lb, AlignCenter).Inset(-l.S(2)).Intersect(lb))
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

func (e openlookEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := olColors(l)
	if label == "" {
		if st.Focused() {
			e.DrawFocusRing(l, ctx, box.Inset(-l.S(1)).Intersect(b))
		}
		return
	}
	gap := min(l.S(4), 8)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, c.text, AlignStart, 0)
	if st.Disabled() {
		c.inactive(l, ctx, rpTextBox(l.body, label, lb, AlignStart).Intersect(lb))
	}
	if st.Focused() {
		e.DrawFocusRing(l, ctx, labelFocusRect(l.body, label, lb, b))
	}
}

func (e openlookEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawRadio is an exclusive setting: the label inside a rectangle, raised,
// or recessed with the BG2 fill when chosen.
func (e openlookEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := olColors(l)
	u := rpU(l)
	if label == "" {
		side := l.metrics.Radio
		box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		e.RadioIndicator(l, ctx, box, st, selected)
		if st.Focused() {
			e.DrawFocusRing(l, ctx, box.Inset(-l.S(1)).Intersect(b))
		}
		return
	}
	g := rpGridAt(ctx, b, u)
	f := l.body
	w := int(f.Advance(label)/u+0.5) + 18
	if w > g.w {
		w = g.w
	}
	h := int(f.Height()/u+0.5) + 6
	if h > g.h {
		h = g.h
	}
	if w < 4 || h < 4 {
		return
	}
	y := (g.h - h) / 2
	on := selected || (st.Pressed() && !st.Disabled())
	fill := c.bg1
	if on {
		fill = c.bg2
	}
	c.box(ctx, g, 0, y, w, h, on, fill)
	lb := g.at(1, y, w-2, h)
	l.drawFittedText(ctx, f, label, lb, c.text, AlignCenter, 0)
	if st.Focused() {
		var k rpInk
		k.dotted(g, 2, y+2, w-4, h-4)
		k.fill(ctx, c.black)
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.at(0, y, w, h))
	}
}

// DrawSwitch: OPEN LOOK had none; a small slide in its 3D manner — a
// recessed slot, BG3 when on, and a raised knob.
func (e openlookEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := olColors(l)
	u := rpU(l)
	tw, th := l.metrics.SwitchW, l.metrics.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	g := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th), u)
	if g.w >= 10 && g.h >= 6 {
		slot := c.bg2
		if on {
			slot = c.bg3
		}
		c.box(ctx, g, 0, 0, g.w, g.h, true, slot)
		kw := g.w / 2
		kx := 1
		if on {
			kx = g.w - kw - 1
		}
		c.box(ctx, g, kx, 1, kw, g.h-2, false, c.bg1)
		if st.Disabled() {
			c.inactive(l, ctx, g.rect())
		}
	}
	e.toggleLabel(l, ctx, b, g.rect(), st, label)
}

// DrawComboBox is an abbreviated menu button — a small raised square with
// the menu mark and cut corners — with the current value beside it.
func (e openlookEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 8 {
		return
	}
	bh := g.h - 4
	if bh > 19 {
		bh = 19
	}
	bw := bh + 1
	by := (g.h - bh) / 2
	pressed := (open || st.Pressed()) && !st.Disabled()
	s := rpShape{w: bw, h: bh, cut: 1}
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	var face, hiK, loK rpInk
	s.fill(&face, g, 0, by)
	face.fill(ctx, fill)
	for row := 0; row < bh; row++ {
		a1, b1, a2, b2, ok := s.frameRuns(row)
		if !ok {
			continue
		}
		switch {
		case row == 0:
			hiK.cells(g, a1, by+row, b1-a1, 1)
		case row == bh-1:
			loK.cells(g, a1, by+row, b1-a1, 1)
		default:
			hiK.cells(g, a1, by+row, b1-a1, 1)
			loK.cells(g, a2, by+row, b2-a2, 1)
		}
	}
	if pressed {
		hiK, loK = loK, hiK
	}
	hiK.fill(ctx, c.hi)
	loK.fill(ctx, c.bg3)
	c.mark(ctx, g, bw/2, by+bh/2, 9, DirDown, pressed)
	lb := g.at(bw+4, 0, g.w-bw-4, g.h)
	l.drawFittedText(ctx, l.body, text, lb, c.text, AlignStart, 0)
	if st.Focused() && !open {
		e.DrawFocusRing(l, ctx, rpTextBox(l.body, text, lb, AlignStart).Inset(-l.S(2)).Intersect(lb))
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// olFieldOpts: reverse-video selections and the triangle caret.
func (c *olc) fieldOpts(l *Classic) rpFieldOpts {
	return rpFieldOpts{
		text: l.fieldText(), muted: c.dim, sel: c.sel, selTxt: c.selTxt,
		selOff: Mix(c.bg2, c.field, 0.3), caret: c.black, caretKind: rpCaretTriangle,
		ditherOff: true, bg: c.bg1,
	}
}

// DrawTextField: no box — the text sits on an engraved underline.
func (e openlookEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	var hi, lo rpInk
	hi.cells(g, 0, g.h-2, g.w, 1)
	lo.cells(g, 0, g.h-1, g.w, 1)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.bg3)
	pad := l.metrics.FieldPad
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-2*pad, b.Dy()-2*u)
	rpFieldText(l, ctx, inner, st, text, placeholder, caret, selA, selB, blink, scrollX, face, c.editOpts(l))
}

// DrawFramelessText: a spin box's or editable combo's text types as the
// text fields do.
func (e openlookEngine) DrawFramelessText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	rpFieldText(l, ctx, l.fieldTextBox(b), st, text, placeholder, caret, selA, selB, blink, scrollX, face, olColors(l).editOpts(l))
}

// editOpts is how a text field types: in the text colour.
func (c *olc) editOpts(l *Classic) rpFieldOpts {
	o := c.fieldOpts(l)
	o.text = c.text
	return o
}

// DrawTextArea: a text window, white inside the etched border.
func (e openlookEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 6 || g.h < 6 {
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.field))
	c.etched(ctx, g)
	pad := l.metrics.FieldPad + l.S(2)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-2*pad, b.Dy()-2*pad)
	o := c.fieldOpts(l)
	o.bg = c.field
	rpAreaText(l, ctx, inner, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face, o)
}

// DrawSpinner is a numeric field's pair of small raised buttons with
// arrows; the held one is recessed.
func (e openlookEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if g.w < 6 || g.h < 8 {
		return
	}
	mid := g.h / 2
	half := func(y, h int, dir Direction, pressed bool) {
		pressed = pressed && !st.Disabled()
		fill := c.bg1
		if pressed {
			fill = c.bg2
		}
		c.box(ctx, g, 0, y, g.w, h, pressed, fill)
		m := olArrowMask(min(8, g.w-4)).turn(dir)
		var k rpInk
		m.emit(&k, g, (g.w-m.w)/2, y+(h-m.h)/2)
		k.fill(ctx, c.bg3)
	}
	half(0, mid, DirUp, upPress)
	half(mid, g.h-mid, DirDown, downPress)
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// DrawTabBar: the tabs are a row of exclusive settings on the window colour.
func (e openlookEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(rpGridAt(ctx, b, rpU(l)).rect(), paintengine2d.Fill(olColors(l).bg1))
}

// DrawTab is one exclusive setting of the row: raised, or recessed with
// the BG2 fill when it is the chosen page.
func (e openlookEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 6 {
		return
	}
	h := g.h - 4
	if h < 4 {
		h = g.h
	}
	y := (g.h - h) / 2
	on := selected || (st.Pressed() && !st.Disabled())
	fill := c.bg1
	if on {
		fill = c.bg2
	}
	c.box(ctx, g, 0, y, g.w, h, on, fill)
	lb := g.at(2, y, g.w-4, h)
	l.drawFittedText(ctx, l.body, label, lb, c.text, AlignCenter, 0)
	if st.Focused() && selected {
		var k rpInk
		k.dotted(g, 2, y+2, g.w-4, h-4)
		k.fill(ctx, c.black)
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.at(0, y, g.w, h))
	}
}

// DrawMenuBar: OPEN LOOK has no menu bar; the control area at the top of a
// base window holds a row of menu buttons.
func (e openlookEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	var hi, lo rpInk
	lo.cells(g, 0, g.h-2, g.w, 1)
	hi.cells(g, 0, g.h-1, g.w, 1)
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.bg3)
}

// DrawMenuTitle is a menu button: an obround with the label and the menu
// mark, recessed while its menu is up.
func (e openlookEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 10 || g.h < 8 {
		return
	}
	h := g.h - 6
	if h > 26 {
		h = 26
	}
	y := (g.h - h) / 2
	pressed := (open || st.Pressed()) && !st.Disabled()
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	s := olObround(g.w-2, h)
	// The toolkit leaves little room round a title: narrow the end caps
	// when the label and the mark need it.
	tw := int(l.body.Advance(label)/u + 0.5)
	room := g.w - 2 - tw
	mark := 9
	if room < 2*int(s.rx)+mark+4 {
		s.rx = float32(max(3, (room-mark-4)/2))
		s.r = min(s.r, s.rx)
	}
	showMark := room >= 2*int(s.rx)+mark/2+4
	c.obround(ctx, g, s, 1, y, pressed, fill)
	lx := int(s.rx) + 1
	lb := g.at(lx, y, g.w-2-lx, h)
	if showMark {
		mx := g.w - 1 - int(s.rx) - mark/2 - 1
		c.mark(ctx, g, mx, y+h/2, mark, DirDown, pressed)
		lb = g.at(lx, y, mx-mark/2-1-lx, h)
	}
	l.drawFittedText(ctx, l.body, label, lb, c.text, AlignCenter, 0)
	if st.Focused() && !open {
		e.DrawFocusRing(l, ctx, rpTextBox(l.body, label, lb, AlignCenter).Inset(-l.S(2)).Intersect(g.rect()))
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// DrawMenuFrame: a menu is the window colour inside a raised bevel.
func (e openlookEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
}

// DrawMenuItem: the item under the pointer is a recessed obround the width
// of the menu; submenus end in the menu mark pointing right.
func (e openlookEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := olColors(l)
	u := rpU(l)
	ch := MenuChromeFor(l)
	full := paintengine2d.XYWH(b.Min.X-ch.PadL+2*u, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-4*u, b.Dy())
	fg := rpGridAt(ctx, full, u)
	if row.Separator {
		var hi, lo rpInk
		y := fg.h / 2
		lo.cells(fg, 2, y-1, fg.w-4, 1)
		hi.cells(fg, 2, y, fg.w-4, 1)
		lo.fill(ctx, c.bg3)
		hi.fill(ctx, c.hi)
		return
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() && fg.w > 6 && fg.h > 4 {
		c.obround(ctx, fg, olObround(fg.w, fg.h), 0, 0, true, c.bg2)
	}
	col := c.text
	f := l.body
	mark := rpGridAt(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ch.CheckCol(), b.Dy()), u)
	switch {
	case row.Checked && row.Radio:
		n := min(10, mark.h-6, mark.w-2)
		if n >= 4 {
			mg := mark.centered(n, n)
			c.box(ctx, mg, 0, 0, n, n, true, c.bg2)
		}
	case row.Checked:
		n := min(14, mark.w, mark.h-4)
		if n >= 7 {
			e.CheckIndicator(l, ctx, mark.centered(n, n).rect(), StateNone, true)
		}
	case row.Icon != IconNone:
		side := min(ch.CheckCol()-l.S(2), b.Dy()-l.S(4))
		ib := paintengine2d.XYWH(b.Min.X+(ch.CheckCol()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		l.drawToolIcon(ctx, ib, row.Icon, col)
	}
	labelRight := b.Max.X
	if row.Submenu {
		ab := rpGridAt(ctx, paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow+l.S(4), b.Dy()), u)
		c.mark(ctx, ab, 4, ab.h/2, 9, DirRight, false)
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
	lb := paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy())
	ctx.Save()
	ctx.ClipRect(lb)
	l.drawTextUnderline(ctx, f, row.Label, -1, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), col)
	ctx.Restore()
	if st.Disabled() {
		c.inactive(l, ctx, full)
	}
}

// DrawProgressBar is the OPEN LOOK gauge: a recessed pill with rounded
// ends and a solid bar for the value.
func (e openlookEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 5 {
		return
	}
	s := rpShape{w: g.w, h: g.h, r: min(5, float32(g.h)*0.5)}
	var face, hiK, loK rpInk
	s.fill(&face, g, 0, 0)
	face.fill(ctx, c.bg2)
	for row := 0; row < g.h; row++ {
		a1, b1, a2, b2, ok := s.frameRuns(row)
		if !ok {
			continue
		}
		if row < g.h/2 {
			loK.cells(g, a1, row, b1-a1, 1)
			if b2 > a2 {
				loK.cells(g, a2, row, b2-a2, 1)
			}
		} else {
			hiK.cells(g, a1, row, b1-a1, 1)
			if b2 > a2 {
				hiK.cells(g, a2, row, b2-a2, 1)
			}
		}
	}
	loK.fill(ctx, c.bg3)
	hiK.fill(ctx, c.hi)
	in := g.sub(2, 2, g.w-4, g.h-4)
	if in.w < 1 || in.h < 1 {
		return
	}
	bar := rpShape{w: in.w, h: in.h, r: min(3, float32(in.h)*0.5)}
	var k rpInk
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		span := in.w / 4
		x := int(float32(in.w+span)*phase) - span
		for y := 0; y < in.h; y++ {
			e := bar.edge(y)
			a, z := max(x, e), min(x+span, in.w-e)
			k.cells(in, a, y, z-a, 1)
		}
	} else {
		n := int(float32(in.w)*clamp1(t) + 0.5)
		for y := 0; y < in.h; y++ {
			e := bar.edge(y)
			z := min(n, in.w-e)
			k.cells(in, e, y, z-e, 1)
		}
	}
	k.fill(ctx, c.black)
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// DrawSlider: a groove with angled ends — solid black left of the knob,
// recessed BG2 to its right — and a raised knob, recessed while dragged.
func (e openlookEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 16 || g.h < 8 {
		return
	}
	t = clamp1(t)
	kw, kh := 11, min(17, g.h-2)
	gh := 5
	gy := (g.h - gh) / 2
	x0, x1 := kw/2, g.w-kw/2
	kx := x0 + int(float32(x1-x0-1)*t+0.5) - kw/2
	groove := rpShape{w: x1 - x0, h: gh, cut: 2}
	var dark, face, hiK, loK rpInk
	for y := 0; y < gh; y++ {
		e := groove.edge(y)
		a, z := x0+e, x1-e
		split := kx + kw/2
		if split > a {
			dark.cells(g, a, gy+y, min(split, z)-a, 1)
		}
		if z > split {
			face.cells(g, max(split, a), gy+y, z-max(split, a), 1)
		}
	}
	face.fill(ctx, c.bg2)
	dark.fill(ctx, c.black)
	loK.cells(g, max(kx+kw/2, x0), gy, x1-max(kx+kw/2, x0)-2, 1)
	hiK.cells(g, max(kx+kw/2, x0), gy+gh-1, x1-max(kx+kw/2, x0)-2, 1)
	loK.fill(ctx, c.bg3)
	hiK.fill(ctx, c.hi)
	pressed := st.Pressed() && !st.Disabled()
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	c.box(ctx, g, kx, (g.h-kh)/2, kw, kh, pressed, fill)
	if st.Focused() {
		e.DrawFocusRing(l, ctx, g.rect())
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

func (e openlookEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(9), b.Min.Y, b.Dx()-l.S(13), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e openlookEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := olColors(l)
	e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x+indent+l.S(3), b.Min.Y+(b.Dy()-f.Height())*0.5), c.text)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e openlookEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if st.Checked() && g.w > 1 && g.h > 1 {
		// One recessed bar across the row: a top and bottom edge per cell.
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg2))
		var hi, lo rpInk
		lo.cells(g, 0, 0, g.w, 1)
		hi.cells(g, 0, g.h-1, g.w, 1)
		lo.fill(ctx, c.bg3)
		hi.fill(ctx, c.hi)
	}
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.text, align, 0)
}

// DrawTableHeader: raised header cells; the sort column carries the mark.
func (e openlookEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 4 {
		return
	}
	pressed := st.Pressed() && !st.Disabled()
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	c.box(ctx, g, 0, 0, g.w, g.h, pressed, fill)
	aw := 0
	if sorted && g.w > 24 {
		aw = 12
		dir := DirDown
		if asc {
			dir = DirUp
		}
		c.mark(ctx, g, g.w-8, g.h/2, 9, dir, false)
	}
	l.drawFittedText(ctx, l.body, label, g.at(4, 0, g.w-8-aw, g.h), c.text, AlignStart, 0)
}

// DrawToolBar: a control area over an engraved line.
func (e openlookEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	e.DrawMenuBar(l, ctx, b)
}

func (e openlookEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := olColors(l)
	l.baseDrawToolButton(ctx, b, st, label, icon)
	if st.Disabled() {
		c.inactive(l, ctx, b)
	}
}

// DrawStatusBar is the base window's footer: messages on the window colour
// under an engraved line.
func (e openlookEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	var hi, lo rpInk
	lo.cells(g, 0, 0, g.w, 1)
	hi.cells(g, 0, 1, g.w, 1)
	lo.fill(ctx, c.bg3)
	hi.fill(ctx, c.hi)
	if len(parts) == 0 {
		return
	}
	slot := float32(g.w) / float32(len(parts))
	for i, s := range parts {
		x := int(slot * float32(i))
		al := AlignStart
		if i == len(parts)-1 && len(parts) > 1 {
			al = AlignEnd // the right footer
		}
		l.drawFittedText(ctx, l.body, s, g.at(x+6, 2, int(slot)-12, g.h-2), c.text, al, 0)
	}
}

// DrawTitleBar: a bold heading over an engraved line.
func (e openlookEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	var hi, lo rpInk
	lo.cells(g, 0, g.h-2, g.w, 1)
	hi.cells(g, 0, g.h-1, g.w, 1)
	lo.fill(ctx, c.bg3)
	hi.fill(ctx, c.hi)
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.dim, AlignStart, 0)
	}
}

// DrawAccordionHeader: the menu mark and a bold label; pressed recesses.
func (e openlookEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	pressed := st.Pressed() && !st.Disabled()
	if pressed {
		c.box(ctx, g, 0, 0, g.w, g.h, true, c.bg2)
	} else {
		ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, c.text)
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()), c.text, AlignStart, 0)
	if st.Focused() {
		e.DrawFocusRing(l, ctx, b)
	}
	if st.Disabled() {
		c.inactive(l, ctx, g.rect())
	}
}

// DrawSeparator: an engraved line.
func (e openlookEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	var hi, lo rpInk
	if vertical {
		x := g.w/2 - 1
		lo.cells(g, x, 2, 1, g.h-4)
		hi.cells(g, x+1, 2, 1, g.h-4)
	} else {
		y := g.h/2 - 1
		lo.cells(g, 0, y, g.w, 1)
		hi.cells(g, 0, y+1, g.w, 1)
	}
	lo.fill(ctx, c.bg3)
	hi.fill(ctx, c.hi)
}

// DrawSplitter: the window colour with an engraved line; raised grip
// under the pointer.
func (e openlookEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	if st.Hovered() || st.Pressed() {
		c.box(ctx, g, 0, 0, g.w, g.h, st.Pressed(), c.bg1)
		return
	}
	ctx.DrawRect(g.rect(), paintengine2d.Fill(c.bg1))
	e.DrawSeparator(l, ctx, b, vertical)
}

// DrawTooltip: OPEN LOOK had no tooltips (help came with the Help key);
// a small raised panel stands in.
func (e openlookEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := olColors(l)
	g := rpGridAt(ctx, b, rpU(l))
	c.box(ctx, g, 0, 0, g.w, g.h, false, c.bg1)
	var k rpInk
	k.frame(g, 0, 0, g.w, g.h, 1)
	k.fill(ctx, c.black)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.text, AlignStart, 0)
}

// DrawOverlay: OpenWindows did not dim what lay behind a notice.
func (openlookEngine) DrawOverlay(*Classic, *paintengine2d.Context, paintengine2d.Rect) {}

// ---- packs --------------------------------------------------------------------

func openlookPacks() []ThemePack {
	bg := hexColor("#cccccc")
	black, white := hexColor("#000000"), hexColor("#ffffff")
	pal := Palette{
		Background: bg, Surface: bg, SurfaceAlt: bg,
		Border: black, Divider: hexColor("#666666"),
		Text: black, TextMuted: hexColor("#4d4d4d"), TextOnAccent: white,
		Accent: black, AccentHover: black, AccentPress: black,
		Field: white, FieldBorder: hexColor("#666666"),
		Focus: black, Selection: black,
		Track: hexColor("#b7b7b7"), Thumb: bg,
		Highlight: hexColor("#f4f4f4"), Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: hexColor("#b7b7b7"), MenuHoverBorder: hexColor("#666666"), MenuGutter: bg,
		Danger: hexColor("#b00000"), Success: hexColor("#006600"), Warning: hexColor("#805000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0),
		BevelLight: hexColor("#f4f4f4"), BevelDark: hexColor("#666666"),
	}
	tok := ThemeTokens{
		Engine:   "openlook",
		Bevel:    BevelClassic3D,
		Family:   ThemeLight,
		Palette:  pal,
		Hot:      ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder},
		Selected: ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder},
		Focus:    ChromeState{Fill: black.WithAlpha(0.15), Border: black},
	}
	return []ThemePack{{
		Name: "openlook", Label: "OPEN LOOK", Year: 1988, Lineage: "Sun",
		Summary: "Sun OpenWindows in 3D: obround buttons, menu marks, the elevator scrollbar and the pushpin.",
		Era:     "OPEN LOOK", Palette: ThemeLight, Tokens: tok,
	}}
}
