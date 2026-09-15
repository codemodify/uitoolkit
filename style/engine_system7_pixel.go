package style

import (
	"math"
	"sync"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// Whole-pixel drawing shared by the retro engines: system7, win31,
// openlook, amiga, beos and os2.
//
// The desktops of the 1980s and early 1990s were drawn a pixel at a time on
// 1-bit and 4-bit screens. QuickDraw's round rects and ovals are staircases
// rather than anti-aliased curves, a grey is a dither, and a 3D edge is a
// row of whole pixels. These helpers build such shapes out of whole-pixel
// rectangles laid on a grid of cells, one cell being one pixel of the
// original screen and u device pixels of ours, so a 2x display shows the
// same drawing pixel-doubled. Every cell of one colour goes into a single
// path that fills in one draw; rows that repeat merge into taller rects, so
// a staircase corner or a dither costs a few dozen rectangles.
//
// Shapes are computed from geometry (the circle through pixel centres, run
// lengths) and patterns from their logic; no bitmap of any original system
// is reproduced.

// rpU is one design pixel (cell) in whole device pixels: 1 at 1x, 2 at 1.5x
// and 2x, so lines stay crisp at every scale.
func rpU(l *Classic) float32 {
	u := float32(math.Floor(float64(l.S(1)) + 0.5))
	if u < 1 {
		return 1
	}
	return u
}

// rpGrid lays whole cells over a rect: the device origin of cell (0, 0),
// the cell size u, and how many whole cells fit across and down.
type rpGrid struct {
	x, y, u float32
	w, h    int
}

// rpGridOf snaps b to the device grid (keeping pixels whose centres lie
// inside b) and fits whole cells of u into it from its top-left corner.
func rpGridOf(b paintengine2d.Rect, u float32) rpGrid {
	if u < 1 {
		u = 1
	}
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	x1, y1 := snap(b.Max.X), snap(b.Max.Y)
	g := rpGrid{x: x0, y: y0, u: u}
	if x1 > x0 {
		g.w = int((x1 - x0) / u)
	}
	if y1 > y0 {
		g.h = int((y1 - y0) / u)
	}
	return g
}

// rpOrigin is the device position of the context's user origin when the
// context only translates (widgets paint in their own coordinates under a
// translation that layout may leave fractional); zero otherwise.
func rpOrigin(ctx *paintengine2d.Context) (float32, float32) {
	if ctx == nil {
		return 0, 0
	}
	m := ctx.Matrix()
	if m.A != 1 || m.D != 1 || m.B != 0 || m.C != 0 {
		return 0, 0
	}
	return m.E, m.F
}

// rpGridAt is rpGridOf snapped on the device's pixel grid rather than in
// user space, so a control whose origin falls between device pixels (a
// dialog centred on a half pixel) still draws in whole pixels.
func rpGridAt(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32) rpGrid {
	ox, oy := rpOrigin(ctx)
	if ox == 0 && oy == 0 {
		return rpGridOf(b, u)
	}
	g := rpGridOf(b.Translate(paintengine2d.Pt(ox, oy)), u)
	g.x -= ox
	g.y -= oy
	return g
}

// at is the device rect of cells [cx, cx+cw) × [cy, cy+ch).
func (g rpGrid) at(cx, cy, cw, ch int) paintengine2d.Rect {
	return paintengine2d.XYWH(g.x+float32(cx)*g.u, g.y+float32(cy)*g.u, float32(cw)*g.u, float32(ch)*g.u)
}

// rect is the device rect of the whole grid.
func (g rpGrid) rect() paintengine2d.Rect { return g.at(0, 0, g.w, g.h) }

// sub is the grid of cw × ch cells whose origin is cell (cx, cy) of g.
func (g rpGrid) sub(cx, cy, cw, ch int) rpGrid {
	if cw < 0 {
		cw = 0
	}
	if ch < 0 {
		ch = 0
	}
	return rpGrid{x: g.x + float32(cx)*g.u, y: g.y + float32(cy)*g.u, u: g.u, w: cw, h: ch}
}

// inset is g shrunk by n cells on every side.
func (g rpGrid) inset(n int) rpGrid { return g.sub(n, n, g.w-2*n, g.h-2*n) }

// centered is a cw × ch grid centred in g on whole cells.
func (g rpGrid) centered(cw, ch int) rpGrid {
	return g.sub((g.w-cw)/2, (g.h-ch)/2, cw, ch)
}

// ok reports whether g holds at least w × h cells.
func (g rpGrid) ok(w, h int) bool { return g.w >= w && g.h >= h }

// rpPaths recycles the paths inks gather into: a path keeps its capacity
// across paints, so a repaint builds its staircases and dithers without
// allocating. (Backends never keep the path itself: recorders snapshot
// it, the GPU caches its tessellation by content.)
var rpPaths = sync.Pool{New: func() any { return paintengine2d.NewPath() }}

func rpPath() *paintengine2d.Path { return rpPaths.Get().(*paintengine2d.Path) }

func rpRecycle(p *paintengine2d.Path) {
	if p != nil {
		p.Reset()
		rpPaths.Put(p)
	}
}

// rpInk gathers the whole-pixel rects of one colour into one path.
type rpInk struct{ p *paintengine2d.Path }

// add appends r (ignoring empty rects).
func (k *rpInk) add(r paintengine2d.Rect) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	if k.p == nil {
		k.p = rpPath()
	}
	k.p.AddRect(r)
}

// cells appends cells [cx, cx+cw) × [cy, cy+ch) of g.
func (k *rpInk) cells(g rpGrid, cx, cy, cw, ch int) {
	if cw > 0 && ch > 0 {
		k.add(g.at(cx, cy, cw, ch))
	}
}

// fill paints everything gathered in c, in one draw, and empties the ink.
func (k *rpInk) fill(ctx *paintengine2d.Context, c paintengine2d.Color) {
	if k.p == nil {
		return
	}
	if c.A > 0 {
		ctx.DrawPath(k.p, paintengine2d.Fill(c))
	}
	rpRecycle(k.p)
	k.p = nil
}

// frame appends a rectangular frame t cells thick around the cw × ch cells
// at (cx, cy), as four rects that never overlap.
func (k *rpInk) frame(g rpGrid, cx, cy, cw, ch, t int) {
	if cw <= 0 || ch <= 0 || t <= 0 {
		return
	}
	if 2*t >= cw || 2*t >= ch {
		k.cells(g, cx, cy, cw, ch)
		return
	}
	k.cells(g, cx, cy, cw, t)
	k.cells(g, cx, cy+ch-t, cw, t)
	k.cells(g, cx, cy+t, t, ch-2*t)
	k.cells(g, cx+cw-t, cy+t, t, ch-2*t)
}

// rpEdge appends a bevel of n one-cell rings around the cw × ch cells at
// (cx, cy): hi along the top and left, lo along the bottom and right, lo
// owning the top-right and bottom-left corner cells (the Windows and Amiga
// convention). Either ink may be nil.
func rpEdge(hi, lo *rpInk, g rpGrid, cx, cy, cw, ch, n int) {
	for i := 0; i < n; i++ {
		x, y, w, h := cx+i, cy+i, cw-2*i, ch-2*i
		if w < 2 || h < 2 {
			return
		}
		if hi != nil {
			hi.cells(g, x, y, w-1, 1)
			hi.cells(g, x, y+1, 1, h-2)
		}
		if lo != nil {
			lo.cells(g, x, y+h-1, w, 1)
			lo.cells(g, x+w-1, y, 1, h-1)
		}
	}
}

// rpDotted appends the classic dotted focus rectangle just inside the
// cw × ch cells at (cx, cy): every other cell along each side.
func (k *rpInk) dotted(g rpGrid, cx, cy, cw, ch int) {
	if cw < 3 || ch < 3 {
		return
	}
	for x := 0; x < cw; x += 2 {
		k.cells(g, cx+x, cy, 1, 1)
		k.cells(g, cx+x, cy+ch-1, 1, 1)
	}
	for y := 2; y < ch-1; y += 2 {
		k.cells(g, cx, cy+y, 1, 1)
		k.cells(g, cx+cw-1, cy+y, 1, 1)
	}
}

// ---- shapes ------------------------------------------------------------------

// rpInset is how many cells row y (0 = the outermost row) of a corner of
// radius r cuts from the straight edge: the circle through pixel centres,
// rounded to whole cells, as QuickDraw rasterised its round rects.
func rpInset(r float32, y int) int {
	if r <= 0 || y < 0 {
		return 0
	}
	d := r - float32(y) - 0.5
	if d <= 0 {
		return 0
	}
	hw := math.Sqrt(float64(r*r - d*d))
	v := int(math.Floor(float64(r) - hw + 0.5))
	if v < 0 {
		return 0
	}
	return v
}

// rpInsetE is rpInset for an elliptical corner rx cells across and ry down.
func rpInsetE(rx, ry float32, y int) int {
	if rx <= 0 || ry <= 0 || y < 0 {
		return 0
	}
	d := ry - float32(y) - 0.5
	if d <= 0 {
		return 0
	}
	q := 1 - float64(d*d)/float64(ry*ry)
	if q < 0 {
		q = 0
	}
	hw := float64(rx) * math.Sqrt(q)
	v := int(math.Floor(float64(rx) - hw + 0.5))
	if v < 0 {
		return 0
	}
	return v
}

// rpShape is a whole-pixel shape in a w × h cell box, symmetric about both
// axes: a rectangle whose corners are cut by a circle of radius r (a round
// rect; an oval or a stadium when r is half the short side) or, when r is 0,
// by a 45° chamfer of cut cells.
type rpShape struct {
	w, h int
	r    float32
	cut  int
	// top keeps the bottom corners square (tabs).
	top bool
	// rx, when set, makes the corners elliptical: rx across, r down
	// (OPEN LOOK's buttons end in caps wider than they are tall).
	rx float32
}

// edge is the first filled cell of row y; the row spans [edge, w-edge).
// Rows outside the box report w (empty).
func (s rpShape) edge(y int) int {
	if y < 0 || y >= s.h || s.w <= 0 {
		return s.w
	}
	if yb := s.h - 1 - y; yb < y {
		if s.top {
			return 0
		}
		y = yb
	}
	if s.r > 0 {
		r := s.r
		if m := float32(s.h) * 0.5; r > m {
			r = m
		}
		rx := s.rx
		if rx <= 0 {
			rx = r
		}
		if m := float32(s.w) * 0.5; rx > m {
			rx = m
			if s.rx <= 0 && r > m {
				r = m
			}
		}
		if rx == r {
			return rpInset(r, y)
		}
		return rpInsetE(rx, r, y)
	}
	if y < s.cut {
		return s.cut - y
	}
	return 0
}

// rpRows gathers the one or two runs of each row of a shape and merges
// rows that repeat into taller rects.
type rpRows struct {
	k              *rpInk
	g              rpGrid
	ox, oy         int
	y0             int
	a1, b1, a2, b2 int
	open           bool
}

// row adds row y (rows come in order; an empty row flushes first).
func (r *rpRows) row(y, a1, b1, a2, b2 int) {
	if r.open && a1 == r.a1 && b1 == r.b1 && a2 == r.a2 && b2 == r.b2 {
		return
	}
	r.flush(y)
	r.y0, r.a1, r.b1, r.a2, r.b2, r.open = y, a1, b1, a2, b2, true
}

func (r *rpRows) flush(y int) {
	if !r.open {
		return
	}
	n := y - r.y0
	if n > 0 {
		r.k.cells(r.g, r.ox+r.a1, r.oy+r.y0, r.b1-r.a1, n)
		r.k.cells(r.g, r.ox+r.a2, r.oy+r.y0, r.b2-r.a2, n)
	}
	r.open = false
}

// fill appends the whole shape with its top-left cell at (ox, oy) of g.
func (s rpShape) fill(k *rpInk, g rpGrid, ox, oy int) {
	rr := rpRows{k: k, g: g, ox: ox, oy: oy}
	for y := 0; y < s.h; y++ {
		e := s.edge(y)
		if 2*e >= s.w {
			rr.flush(y)
			continue
		}
		rr.row(y, e, s.w-e, 0, 0)
	}
	rr.flush(s.h)
}

// frameRuns is row y of the shape's one-cell outline (every filled cell
// with an empty 4-neighbour): one or two runs [a1, b1) and [a2, b2).
func (s rpShape) frameRuns(y int) (a1, b1, a2, b2 int, ok bool) {
	e := s.edge(y)
	if 2*e >= s.w {
		return 0, 0, 0, 0, false
	}
	if y == 0 || y == s.h-1 {
		return e, s.w - e, 0, 0, true
	}
	m := s.edge(y - 1)
	if n := s.edge(y + 1); n > m {
		m = n
	}
	m--
	if m < e {
		m = e
	}
	if 2*m+2 >= s.w {
		return e, s.w - e, 0, 0, true
	}
	return e, m + 1, s.w - 1 - m, s.w - e, true
}

// frame appends the shape's one-cell outline, the thin staircase QuickDraw
// framed round rects with.
func (s rpShape) frame(k *rpInk, g rpGrid, ox, oy int) {
	rr := rpRows{k: k, g: g, ox: ox, oy: oy}
	for y := 0; y < s.h; y++ {
		a1, b1, a2, b2, ok := s.frameRuns(y)
		if !ok {
			rr.flush(y)
			continue
		}
		rr.row(y, a1, b1, a2, b2)
	}
	rr.flush(s.h)
}

// frameDots appends every other cell of the outline — a dotted outline,
// the cells whose x+y is even in the grid's own coordinates.
func (s rpShape) frameDots(k *rpInk, g rpGrid, ox, oy int) {
	for y := 0; y < s.h; y++ {
		a1, b1, a2, b2, ok := s.frameRuns(y)
		if !ok {
			continue
		}
		for _, r := range [2][2]int{{a1, b1}, {a2, b2}} {
			for x := r[0]; x < r[1]; x++ {
				if (ox+x+oy+y)&1 == 0 {
					k.cells(g, ox+x, oy+y, 1, 1)
				}
			}
		}
	}
}

// ringRuns is row y of the band t cells thick inside the shape's edge: the
// shape minus the concentric shape t cells smaller (QuickDraw's thick pen).
func (s rpShape) ringRuns(y, t int) (a1, b1, a2, b2 int, ok bool) {
	e := s.edge(y)
	if 2*e >= s.w {
		return 0, 0, 0, 0, false
	}
	in := rpShape{w: s.w - 2*t, h: s.h - 2*t, r: max(s.r-float32(t), 0), cut: max(s.cut-t, 0), top: s.top}
	if s.rx > 0 {
		in.rx = max(s.rx-float32(t), 0.01)
	}
	if y < t || y >= s.h-t || in.w <= 0 {
		return e, s.w - e, 0, 0, true
	}
	ie := in.edge(y - t)
	if 2*ie >= in.w {
		return e, s.w - e, 0, 0, true
	}
	return e, t + ie, s.w - t - ie, s.w - e, true
}

// ring appends the band t cells thick inside the shape's edge.
func (s rpShape) ring(k *rpInk, g rpGrid, ox, oy, t int) {
	if t <= 0 {
		return
	}
	if t == 1 {
		s.frame(k, g, ox, oy)
		return
	}
	rr := rpRows{k: k, g: g, ox: ox, oy: oy}
	for y := 0; y < s.h; y++ {
		a1, b1, a2, b2, ok := s.ringRuns(y, t)
		if !ok {
			rr.flush(y)
			continue
		}
		rr.row(y, a1, b1, a2, b2)
	}
	rr.flush(s.h)
}

// ringDots appends every other cell of the band (x+y even): a band in
// the 50% grey of 1-bit screens.
func (s rpShape) ringDots(k *rpInk, g rpGrid, ox, oy, t int) {
	for y := 0; y < s.h; y++ {
		a1, b1, a2, b2, ok := s.ringRuns(y, t)
		if !ok {
			continue
		}
		for _, r := range [2][2]int{{a1, b1}, {a2, b2}} {
			for x := r[0]; x < r[1]; x++ {
				if (ox+x+oy+y)&1 == 0 {
					k.cells(g, ox+x, oy+y, 1, 1)
				}
			}
		}
	}
}

// ---- masks ---------------------------------------------------------------------

// rpMask is a small one-bit cell bitmap (at most 64 × 64) that the engines
// compose their glyphs in — arrows, check marks, boxes, dots — from lines,
// spans and shapes, then draw as whole pixels. Bit x of rows[y] is cell
// (x, y).
type rpMask struct {
	w, h int
	rows [64]uint64
}

func rpNewMask(w, h int) rpMask {
	if w > 64 {
		w = 64
	}
	if h > 64 {
		h = 64
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return rpMask{w: w, h: h}
}

// set turns cell (x, y) on (cells outside the mask are ignored).
func (m *rpMask) set(x, y int) {
	if x >= 0 && y >= 0 && x < m.w && y < m.h {
		m.rows[y] |= 1 << uint(x)
	}
}

// get reports cell (x, y).
func (m *rpMask) get(x, y int) bool {
	return x >= 0 && y >= 0 && x < m.w && y < m.h && m.rows[y]&(1<<uint(x)) != 0
}

// span turns on cells [x0, x1) of row y.
func (m *rpMask) span(x0, x1, y int) {
	if y < 0 || y >= m.h {
		return
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 > m.w {
		x1 = m.w
	}
	for x := x0; x < x1; x++ {
		m.rows[y] |= 1 << uint(x)
	}
}

// rect turns on the w × h cells at (x, y).
func (m *rpMask) rect(x, y, w, h int) {
	for yy := y; yy < y+h; yy++ {
		m.span(x, x+w, yy)
	}
}

// line turns on the cells of the Bresenham line from (x0, y0) to (x1, y1).
func (m *rpMask) line(x0, y0, x1, y1 int) {
	dx, dy := x1-x0, y1-y0
	sx, sy := 1, 1
	if dx < 0 {
		dx, sx = -dx, -1
	}
	if dy < 0 {
		dy, sy = -dy, -1
	}
	err := dx - dy
	for {
		m.set(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// shape turns on the cells of s with its top-left cell at (x, y).
func (m *rpMask) shape(s rpShape, x, y int) {
	for yy := 0; yy < s.h; yy++ {
		e := s.edge(yy)
		if 2*e < s.w {
			m.span(x+e, x+s.w-e, y+yy)
		}
	}
}

// outline keeps the set cells that have an empty 4-neighbour.
func (m rpMask) outline() rpMask {
	o := m
	for y := 0; y < m.h; y++ {
		r := m.rows[y]
		var up, dn uint64
		if y > 0 {
			up = m.rows[y-1]
		}
		if y < m.h-1 {
			dn = m.rows[y+1]
		}
		o.rows[y] = r &^ (r & (r << 1) & (r >> 1) & up & dn)
	}
	return o
}

// minus clears the cells set in o.
func (m rpMask) minus(o rpMask) rpMask {
	for y := 0; y < m.h; y++ {
		m.rows[y] &^= o.rows[y]
	}
	return m
}

// or sets the cells set in o.
func (m rpMask) or(o rpMask) rpMask {
	for y := 0; y < m.h; y++ {
		m.rows[y] |= o.rows[y]
	}
	return m
}

// turn rotates a glyph drawn pointing up so it points dir.
func (m rpMask) turn(dir Direction) rpMask {
	switch dir {
	case DirDown:
		o := rpMask{w: m.w, h: m.h}
		for y := 0; y < m.h; y++ {
			o.rows[y] = m.rows[m.h-1-y]
		}
		return o
	case DirLeft, DirRight:
		o := rpMask{w: m.h, h: m.w}
		for y := 0; y < m.h; y++ {
			for x := 0; x < m.w; x++ {
				if m.rows[y]&(1<<uint(x)) == 0 {
					continue
				}
				nx, ny := y, x
				if dir == DirRight {
					nx = m.h - 1 - y
				}
				o.rows[ny] |= 1 << uint(nx)
			}
		}
		return o
	}
	return m
}

// emit appends the set cells with the mask's top-left cell at (ox, oy) of
// g: one rect per run, repeated rows merged.
func (m *rpMask) emit(k *rpInk, g rpGrid, ox, oy int) {
	y := 0
	for y < m.h {
		row := m.rows[y]
		if row == 0 {
			y++
			continue
		}
		y2 := y + 1
		for y2 < m.h && m.rows[y2] == row {
			y2++
		}
		x := 0
		for x < m.w {
			if row&(1<<uint(x)) == 0 {
				x++
				continue
			}
			s := x
			for x < m.w && row&(1<<uint(x)) != 0 {
				x++
			}
			k.cells(g, ox+s, oy+y, x-s, y2-y)
		}
		y = y2
	}
}

// ---- patterns ------------------------------------------------------------------

// rpPat is an 8 × 8 one-bit pattern, one byte per row with the most
// significant bit leftmost — how QuickDraw, Windows and Intuition all
// described their fill patterns.
type rpPat [8]uint8

var (
	// rpGray is the 50% grey of 1-bit screens: a one-cell checkerboard.
	rpGray = rpPat{0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55}
	// rpLight is a 25% grey: every fourth cell of each row, alternate rows
	// shifted by two cells.
	rpLight = rpPat{0x88, 0x22, 0x88, 0x22, 0x88, 0x22, 0x88, 0x22}
	// rpDots is a sparse 12.5% dot grid (every fourth cell of every other
	// row, alternate dotted rows shifted by two), used to ghost gadgets.
	rpDots = rpPat{0x88, 0x00, 0x22, 0x00, 0x88, 0x00, 0x22, 0x00}
)

// rpMaxCells is where rpPatFill stops listing every cell of a checkerboard
// and fills it as the even-odd overlap of stripes instead (the CPU fills up
// to 4096 separate rects of one path analytically).
const rpMaxCells = 4096

// rpPatFill paints the set cells of pat over r in col, in one path. The
// pattern is anchored to the device grid in cells of u, so pieces of one
// surface painted separately (a trough split by its thumb) keep one phase.
func rpPatFill(ctx *paintengine2d.Context, r paintengine2d.Rect, pat rpPat, col paintengine2d.Color, u float32) {
	if ctx == nil || col.A <= 0 {
		return
	}
	if u < 1 {
		u = 1
	}
	// Work on the device grid: the pattern's phase and every cell edge are
	// whole device pixels wherever the control's origin lies.
	ox, oy := rpOrigin(ctx)
	x0, y0, x1, y1 := snap(r.Min.X+ox), snap(r.Min.Y+oy), snap(r.Max.X+ox), snap(r.Max.Y+oy)
	if x1 <= x0 || y1 <= y0 {
		return
	}
	dev := func(a, b, c, d float32) paintengine2d.Rect {
		return paintengine2d.Rect{Min: paintengine2d.Pt(a-ox, b-oy), Max: paintengine2d.Pt(c-ox, d-oy)}
	}
	cx0 := int(math.Floor(float64(x0 / u)))
	cx1 := int(math.Ceil(float64(x1 / u)))
	cy0 := int(math.Floor(float64(y0 / u)))
	cy1 := int(math.Ceil(float64(y1 / u)))
	clampX := func(v float32) float32 { return max(x0, min(x1, v)) }
	clampY := func(v float32) float32 { return max(y0, min(y1, v)) }
	cells := (cx1 - cx0) * (cy1 - cy0)
	if cells > 2*rpMaxCells && (pat == rpGray || pat == rpPat{0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa, 0x55, 0xaa}) {
		// A checkerboard is the exclusive-or of every other column with
		// every other row: a few hundred stripes filled even-odd instead of
		// one rect per cell.
		phase := 0 // rpGray: cell (x, y) is set when x+y is even
		if pat != rpGray {
			phase = 1
		}
		p := rpPath()
		defer rpRecycle(p)
		for cx := cx0; cx < cx1; cx++ {
			if (cx+phase)&1 == 0 {
				p.AddRect(dev(clampX(float32(cx)*u), y0, clampX(float32(cx+1)*u), y1))
			}
		}
		for cy := cy0; cy < cy1; cy++ {
			if cy&1 == 1 {
				p.AddRect(dev(x0, clampY(float32(cy)*u), x1, clampY(float32(cy+1)*u)))
			}
		}
		ctx.DrawPath(p, paintengine2d.Paint{Color: col, FillRule: paintengine2d.FillEvenOdd})
		return
	}
	p := rpPath()
	defer rpRecycle(p)
	for cy := cy0; cy < cy1; cy++ {
		bits := pat[cy&7]
		if bits == 0 {
			continue
		}
		ry0, ry1 := clampY(float32(cy)*u), clampY(float32(cy+1)*u)
		if ry1 <= ry0 {
			continue
		}
		if bits == 0xff {
			p.AddRect(dev(x0, ry0, x1, ry1))
			continue
		}
		on := func(cx int) bool { return bits&(0x80>>uint(cx&7)) != 0 }
		for cx := cx0; cx < cx1; {
			if !on(cx) {
				cx++
				continue
			}
			s := cx
			for cx < cx1 && on(cx) {
				cx++
			}
			rx0, rx1 := clampX(float32(s)*u), clampX(float32(cx)*u)
			if rx1 > rx0 {
				p.AddRect(dev(rx0, ry0, rx1, ry1))
			}
		}
	}
	if !p.Empty() {
		ctx.DrawPath(p, paintengine2d.Fill(col))
	}
}

// ---- text in fields -------------------------------------------------------------

// Caret shapes of rpFieldOpts.
const (
	rpCaretBar      = iota // a one-cell vertical bar
	rpCaretTriangle        // OPEN LOOK's small triangle under the baseline
)

// rpFieldOpts styles the text of a whole-pixel text field or text area.
type rpFieldOpts struct {
	text, muted paintengine2d.Color // text; placeholder and disabled text
	sel, selTxt paintengine2d.Color // selection fill and its text
	selOff      paintengine2d.Color // selection fill without focus (unset: sel)
	caret       paintengine2d.Color
	caretKind   int
	ditherOff   bool                // grey disabled text the 1-bit way
	ditherMuted bool                // and placeholders too (1-bit looks have no grey)
	bg          paintengine2d.Color // what the dither erases to
	selOffFrame bool                // an unfocused selection is outlined, not filled
}

// rpCaret paints the insertion point at x for a line of face f at y.
func rpCaret(ctx *paintengine2d.Context, x, y float32, f *Font, u float32, o rpFieldOpts) {
	x = snap(x)
	switch o.caretKind {
	case rpCaretTriangle:
		// A solid triangle five cells wide and six tall, its tip two cells
		// above the baseline.
		base := snap(y + f.Ascent + 2*u)
		var k rpInk
		for i := 0; i < 3; i++ {
			k.add(paintengine2d.XYWH(x-float32(i)*u, base-float32(3-i)*2*u, float32(2*i+1)*u, 2*u))
		}
		k.fill(ctx, o.caret)
	default:
		ctx.DrawRect(paintengine2d.XYWH(x, snap(y+2*u), u, snap(f.Height()-4*u)), paintengine2d.Fill(o.caret))
	}
}

// rpFieldText paints the text of a single-line field inside inner: the
// selection, the text, the caret on whole pixels.
func rpFieldText(l *Classic, ctx *paintengine2d.Context, inner paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font, o rpFieldOpts) {
	if inner.Empty() {
		return
	}
	u := rpU(l)
	f := l.faceOrBody(face)
	ctx.Save()
	ctx.ClipRect(inner)
	show, col := text, o.text
	dim := st.Disabled() && o.ditherOff
	if text == "" && placeholder != "" && !st.Focused() {
		show, col = placeholder, o.muted
		if o.ditherMuted {
			col, dim = o.text, true
		}
	}
	if st.Disabled() && !o.ditherOff {
		col = o.muted
	}
	ty := snap(inner.Min.Y + (inner.Dy()-f.Height())*0.5)
	ox := inner.Min.X - scrollX
	var selBox paintengine2d.Rect
	if selA != selB && text != "" && show == text {
		if selA > selB {
			selA, selB = selB, selA
		}
		x0, x1 := snap(ox+f.CaretX(text, selA)), snap(ox+f.CaretX(text, selB))
		selBox = paintengine2d.XYWH(x0, ty, x1-x0, snap(f.Height()))
		switch {
		case st.Focused():
			ctx.DrawRect(selBox, paintengine2d.Fill(o.sel))
		case o.selOffFrame:
			var k rpInk
			g := rpGridAt(ctx, selBox, u)
			k.frame(g, 0, 0, g.w, g.h, 1)
			k.fill(ctx, o.sel)
			selBox = paintengine2d.Rect{}
		default:
			c := o.selOff
			if colorUnset(c) {
				c = o.sel
			}
			ctx.DrawRect(selBox, paintengine2d.Fill(c))
			if c != o.sel {
				selBox = paintengine2d.Rect{}
			}
		}
	}
	f.Draw(ctx, show, paintengine2d.Pt(ox, ty), col)
	if !selBox.Empty() {
		ctx.Save()
		ctx.ClipRect(selBox)
		f.Draw(ctx, show, paintengine2d.Pt(ox, ty), o.selTxt)
		ctx.Restore()
	}
	if dim && show != "" {
		rpPatFill(ctx, paintengine2d.XYWH(ox, ty, f.Advance(show), f.Height()).Intersect(inner), rpGray, o.bg, u)
	}
	if st.Focused() && blink && show == text {
		rpCaret(ctx, ox+f.CaretX(text, caret), ty, f, u, o)
	}
	ctx.Restore()
}

// rpAreaText paints the lines of a text area inside inner (selection per
// line, text, caret), on whole pixels.
func rpAreaText(l *Classic, ctx *paintengine2d.Context, inner paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font, o rpFieldOpts) {
	if inner.Empty() {
		return
	}
	u := rpU(l)
	f := l.faceOrBody(face)
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(inner)
	lh := f.Height() + 2
	empty := len(lines) == 0 || (len(lines) == 1 && lines[0].Text == "" && lines[0].End <= lines[0].Start)
	if empty && placeholder != "" && !st.Focused() {
		if o.ditherMuted {
			f.Draw(ctx, placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), o.text)
			rpPatFill(ctx, paintengine2d.XYWH(inner.Min.X, inner.Min.Y, f.Advance(placeholder), f.Height()).Intersect(inner), rpGray, o.bg, u)
		} else {
			f.Draw(ctx, placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), o.muted)
		}
		return
	}
	col := o.text
	if st.Disabled() && !o.ditherOff {
		col = o.muted
	}
	if selA > selB {
		selA, selB = selB, selA
	}
	for i, line := range lines {
		y := snap(inner.Min.Y + float32(i)*lh - scrollY)
		if y+lh < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		ox := inner.Min.X - scrollX
		if selA != selB && selB > line.Start && selA < line.End {
			n := utf8.RuneCountInString(line.Text)
			a, b := max(selA, line.Start)-line.Start, min(selB, line.End)-line.Start
			x0, x1 := snap(ox+f.CaretX(line.Text, a)), snap(ox+f.CaretX(line.Text, b))
			if b > n || selB > line.Start+n {
				x1 = inner.Max.X
			}
			if x1 < x0 {
				x1 = x0
			}
			box := paintengine2d.XYWH(x0, y, x1-x0, snap(f.Height()))
			fill := o.sel
			if !st.Focused() && !colorUnset(o.selOff) {
				fill = o.selOff
			}
			ctx.DrawRect(box, paintengine2d.Fill(fill))
			if line.Text != "" {
				f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), col)
				if fill == o.sel {
					ctx.Save()
					ctx.ClipRect(box)
					f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), o.selTxt)
					ctx.Restore()
				}
			}
		} else if line.Text != "" {
			f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), col)
		}
		if st.Disabled() && o.ditherOff && line.Text != "" {
			rpPatFill(ctx, paintengine2d.XYWH(ox, y, f.Advance(line.Text), f.Height()).Intersect(inner), rpGray, o.bg, u)
		}
		if st.Focused() && blink && caret >= line.Start && caret <= line.End {
			onThis := caret < line.End || i == len(lines)-1
			if caret == line.End && i < len(lines)-1 && lines[i+1].Start == line.End {
				onThis = false
			}
			if onThis {
				rpCaret(ctx, ox+f.CaretX(line.Text, max(caret-line.Start, 0)), y, f, u, o)
			}
		}
	}
}
