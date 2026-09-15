package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TrackMode is how a grid row or column is sized.
type TrackMode uint8

const (
	// TrackAuto fits the track to its content (WPF "Auto").
	TrackAuto TrackMode = iota
	// TrackPx is a fixed length.
	TrackPx
	// TrackFlex shares the space left after the other tracks, by weight
	// (WPF "*"); it never shrinks below its content.
	TrackFlex
)

// Track sizes one grid row or column.
type Track struct {
	Mode TrackMode
	Size float32 // TrackPx: the length; TrackFlex: the weight (0 = 1)
	Min  float32 // never smaller than this
}

// Auto is a track that fits its content.
func Auto() Track { return Track{} }

// Px is a fixed-length track.
func Px(v float32) Track { return Track{Mode: TrackPx, Size: v} }

// Flex is a track that takes a weight's share of the leftover space.
func Flex(weight float32) Track { return Track{Mode: TrackFlex, Size: weight} }

// GridCell is where a grid child sits: its cell, its span and how it is
// aligned inside the cells it covers (by default it stretches across and
// centres vertically, so a field keeps its height in a taller row).
type GridCell struct {
	Row, Col         int
	RowSpan, ColSpan int
	HAlign, VAlign   layout.Align
}

// Grid lays children out in rows and columns — QGridLayout, GtkGrid, WPF's
// Grid, WinForms' TableLayoutPanel. Each child takes a cell and may span
// several; rows and columns fit their content, keep a fixed length or share
// the leftover space (Cols, Rows; missing tracks fit their content).
type Grid struct {
	widget.Base
	Cols, Rows     []Track
	ColGap, RowGap float32
	cells          []*GridCell // parallel to Children()
}

// NewGrid is an empty grid with 8px gaps.
func NewGrid() *Grid {
	g := &Grid{ColGap: 8, RowGap: 8}
	g.Init(g)
	return g
}

// Place puts c in the cell at row, col.
func (g *Grid) Place(c widget.Component, row, col int) *GridCell {
	return g.PlaceSpan(c, row, col, 1, 1)
}

// PlaceSpan puts c over rowSpan × colSpan cells starting at row, col.
func (g *Grid) PlaceSpan(c widget.Component, row, col, rowSpan, colSpan int) *GridCell {
	if rowSpan < 1 {
		rowSpan = 1
	}
	if colSpan < 1 {
		colSpan = 1
	}
	cell := &GridCell{Row: row, Col: col, RowSpan: rowSpan, ColSpan: colSpan, HAlign: layout.AlignStretch, VAlign: layout.AlignCenter}
	g.Base.Add(c)
	g.cells = append(g.cells, cell)
	g.RequestLayout()
	return cell
}

// CellOf is c's cell, or nil when c is not in the grid.
func (g *Grid) CellOf(c widget.Component) *GridCell {
	for i, ch := range g.Children() {
		if ch == c && i < len(g.cells) {
			return g.cells[i]
		}
	}
	return nil
}

// dims is the number of rows and columns in use.
func (g *Grid) dims() (rows, cols int) {
	rows, cols = len(g.Rows), len(g.Cols)
	for _, c := range g.cells {
		if n := c.Row + c.RowSpan; n > rows {
			rows = n
		}
		if n := c.Col + c.ColSpan; n > cols {
			cols = n
		}
	}
	return rows, cols
}

func trackAt(ts []Track, i int) Track {
	if i >= 0 && i < len(ts) {
		return ts[i]
	}
	return Track{}
}

// visible pairs each shown child with its cell.
func (g *Grid) visible(fn func(c widget.Component, cell *GridCell)) {
	for i, ch := range g.Children() {
		if i >= len(g.cells) || !ch.Visible() {
			continue
		}
		fn(ch, g.cells[i])
	}
}

// solve sizes the tracks of one axis. content[i] is the natural length of
// single-span children in track i; spans are (first, count, need) of the
// children covering several tracks. avail < 0 means "no size to fill":
// flexible tracks then take their content.
func solve(ts []Track, n int, content []float32, spans [][3]float32, gap, avail float32) []float32 {
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		t := trackAt(ts, i)
		switch t.Mode {
		case TrackPx:
			out[i] = t.Size
		default:
			out[i] = content[i]
		}
		if out[i] < t.Min {
			out[i] = t.Min
		}
	}
	// Spanning children grow their tracks when they do not fit: flexible
	// tracks first, then auto ones, evenly.
	for _, sp := range spans {
		first, count, need := int(sp[0]), int(sp[1]), sp[2]
		have := gap * float32(count-1)
		for i := first; i < first+count && i < n; i++ {
			have += out[i]
		}
		if need <= have {
			continue
		}
		grow := func(mode TrackMode) bool {
			var idx []int
			for i := first; i < first+count && i < n; i++ {
				if trackAt(ts, i).Mode == mode {
					idx = append(idx, i)
				}
			}
			if len(idx) == 0 {
				return false
			}
			each := (need - have) / float32(len(idx))
			for _, i := range idx {
				out[i] += each
			}
			return true
		}
		if !grow(TrackFlex) {
			grow(TrackAuto)
		}
	}
	if avail < 0 {
		return out
	}
	// Flexible tracks share what the others leave, by weight, but never
	// below their content.
	used := gap * float32(max(n-1, 0))
	var weight float32
	for i := 0; i < n; i++ {
		t := trackAt(ts, i)
		if t.Mode == TrackFlex {
			w := t.Size
			if w <= 0 {
				w = 1
			}
			weight += w
			continue
		}
		used += out[i]
	}
	if weight == 0 {
		return out
	}
	left := avail - used
	for i := 0; i < n; i++ {
		t := trackAt(ts, i)
		if t.Mode != TrackFlex {
			continue
		}
		w := t.Size
		if w <= 0 {
			w = 1
		}
		if share := left * w / weight; share > out[i] {
			out[i] = share
		}
	}
	return out
}

// columns sizes the columns to fill avail (or naturally when avail < 0).
func (g *Grid) columns(avail float32) []float32 {
	_, nc := g.dims()
	content := make([]float32, nc)
	var spans [][3]float32
	g.visible(func(c widget.Component, cell *GridCell) {
		w := c.Measure(layout.Unbounded()).X
		if cell.ColSpan == 1 {
			if cell.Col < nc && w > content[cell.Col] {
				content[cell.Col] = w
			}
			return
		}
		spans = append(spans, [3]float32{float32(cell.Col), float32(cell.ColSpan), w})
	})
	return solve(g.Cols, nc, content, spans, g.ColGap, avail)
}

// spanLen is the length of count tracks from first, gaps included.
func spanLen(sizes []float32, first, count int, gap float32) float32 {
	var l float32
	for i := first; i < first+count && i < len(sizes); i++ {
		l += sizes[i]
	}
	if count > 1 {
		l += gap * float32(count-1)
	}
	return l
}

// rows sizes the rows for the given column widths, filling avail.
func (g *Grid) rows(cols []float32, avail float32) []float32 {
	nr, _ := g.dims()
	content := make([]float32, nr)
	var spans [][3]float32
	g.visible(func(c widget.Component, cell *GridCell) {
		cw := spanLen(cols, cell.Col, cell.ColSpan, g.ColGap)
		h := c.Measure(layout.Constraints{MaxW: cw, MaxH: -1}).Y
		if cell.RowSpan == 1 {
			if cell.Row < nr && h > content[cell.Row] {
				content[cell.Row] = h
			}
			return
		}
		spans = append(spans, [3]float32{float32(cell.Row), float32(cell.RowSpan), h})
	})
	return solve(g.Rows, nr, content, spans, g.RowGap, avail)
}

func sum(v []float32, gap float32) float32 {
	var s float32
	for _, x := range v {
		s += x
	}
	if len(v) > 1 {
		s += gap * float32(len(v)-1)
	}
	return s
}

func (g *Grid) Measure(c layout.Constraints) paintengine2d.Point {
	cols := g.columns(-1)
	w := sum(cols, g.ColGap)
	if c.HasMaxW() && w > c.MaxW {
		cols = g.columns(c.MaxW)
		w = c.MaxW
	}
	h := sum(g.rows(cols, -1), g.RowGap)
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (g *Grid) Arrange(r paintengine2d.Rect) {
	g.SetBounds(r)
	cols := g.columns(r.Dx())
	rows := g.rows(cols, r.Dy())
	xs := make([]float32, len(cols)+1)
	for i, w := range cols {
		xs[i+1] = xs[i] + w + g.ColGap
	}
	ys := make([]float32, len(rows)+1)
	for i, h := range rows {
		ys[i+1] = ys[i] + h + g.RowGap
	}
	g.visible(func(c widget.Component, cell *GridCell) {
		if cell.Col >= len(cols) || cell.Row >= len(rows) {
			return
		}
		box := paintengine2d.XYWH(xs[cell.Col], ys[cell.Row],
			spanLen(cols, cell.Col, cell.ColSpan, g.ColGap), spanLen(rows, cell.Row, cell.RowSpan, g.RowGap))
		c.Arrange(alignIn(c, box, cell.HAlign, cell.VAlign))
	})
}

// alignIn places c inside box: stretched, or at its natural size aligned.
func alignIn(c widget.Component, box paintengine2d.Rect, h, v layout.Align) paintengine2d.Rect {
	sz := c.Measure(layout.Constraints{MaxW: box.Dx(), MaxH: box.Dy()})
	x, w := box.Min.X, box.Dx()
	if h != layout.AlignStretch && sz.X < w {
		w = sz.X
		switch h {
		case layout.AlignCenter:
			x += (box.Dx() - w) * 0.5
		case layout.AlignEnd:
			x = box.Max.X - w
		}
	}
	y, ht := box.Min.Y, box.Dy()
	if v != layout.AlignStretch && sz.Y < ht {
		ht = sz.Y
		switch v {
		case layout.AlignCenter:
			y += (box.Dy() - ht) * 0.5
		case layout.AlignEnd:
			y = box.Max.Y - ht
		}
	}
	return paintengine2d.XYWH(x, y, w, ht)
}

// Form is the dialog form layout (QFormLayout, GtkGrid forms): a label
// column and a field column that takes the rest of the width. Labels are
// right-aligned against their fields where the look's platform did it
// (Mac OS, NeXT) and left-aligned elsewhere.
type Form struct {
	Grid
	labels []*Label
	n      int
}

// NewForm is an empty form.
func NewForm() *Form {
	f := &Form{}
	f.ColGap, f.RowGap = 10, 8
	f.Cols = []Track{Auto(), Flex(1)}
	f.Init(f)
	return f
}

// AddRow adds a labelled field and returns the label.
func (f *Form) AddRow(label string, field widget.Component) *Label {
	l := NewLabel(label)
	f.labels = append(f.labels, l)
	f.Place(l, f.n, 0)
	if field != nil {
		f.Place(field, f.n, 1)
	}
	f.n++
	return l
}

// AddWide adds a row that spans both columns (a heading, a check box).
func (f *Form) AddWide(c widget.Component) {
	f.PlaceSpan(c, f.n, 0, 1, 2)
	f.n++
}

func (f *Form) Arrange(r paintengine2d.Rect) {
	align := style.AlignStart
	if style.LookHint(f.Look(), style.HintFormLabelsRight) == 1 {
		align = style.AlignEnd
	}
	for _, l := range f.labels {
		l.Align = align
	}
	f.Grid.Arrange(r)
}
