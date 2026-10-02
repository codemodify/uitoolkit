package widgets

import (
	"math"
	"strings"

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
	Cols, Rows []Track
	// ColGap and RowGap are 1x design lengths, scaled by the look like
	// every other length in the toolkit ([style.Dip]); RawSpacing takes
	// them as device pixels instead. A form whose gaps did not follow
	// the display was half as airy on a 2x screen as on a 1x one, beside
	// text and controls that had doubled.
	ColGap, RowGap float32
	RawSpacing     bool
	// cells is keyed by the component, not by its position among the
	// children. It used to be a slice parallel to Children(), which the
	// inherited Remove and ClearChildren know nothing about: removing
	// the first child left every later child wearing its neighbour's
	// cell, so a grid silently rearranged itself. Reparenting had the
	// same effect.
	cells map[widget.Component]*GridCell
	// order keeps the placement order, so dims and visible walk the
	// cells in the order they were placed rather than in map order.
	order []widget.Component
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
	if g.cells == nil {
		g.cells = map[widget.Component]*GridCell{}
	}
	if _, seen := g.cells[c]; !seen {
		g.order = append(g.order, c)
	}
	g.cells[c] = cell
	g.RequestLayout()
	return cell
}

// CellOf is c's cell, or nil when c is not in the grid.
func (g *Grid) CellOf(c widget.Component) *GridCell { return g.cells[c] }

// dims is the number of rows and columns in use.
func (g *Grid) dims() (rows, cols int) {
	rows, cols = len(g.Rows), len(g.Cols)
	for _, c := range g.placed() {
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
	for _, ch := range g.Children() {
		cell := g.cells[ch]
		if cell == nil || !ch.Visible() {
			continue
		}
		fn(ch, cell)
	}
}

// placed is every cell still holding a child of this grid, in placement
// order. A child taken out by Remove or ClearChildren drops out here
// rather than leaving its cell behind to enlarge the grid.
func (g *Grid) placed() []*GridCell {
	out := make([]*GridCell, 0, len(g.order))
	live := map[widget.Component]bool{}
	for _, ch := range g.Children() {
		live[ch] = true
	}
	for _, c := range g.order {
		if live[c] {
			if cell := g.cells[c]; cell != nil {
				out = append(out, cell)
			}
		}
	}
	return out
}

// solve sizes the tracks of one axis. content[i] is the natural length of
// single-span children in track i; spans are (first, count, need) of the
// children covering several tracks. avail < 0 means "no size to fill":
// flexible tracks then take their content.
// solve sizes n tracks. content is each track's natural size — what its
// children want when nothing constrains them — and minc is the least
// each can be squeezed to; pass nil for minc where shrinking is not on
// offer, and every track is then treated as unshrinkable.
func solve(ts []Track, n int, content, minc []float32, spans [][3]float32, gap, avail float32) []float32 {
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
	//
	// "Never below their content" is why this iterates. Taking
	// max(content, share) for each track independently lets a track that
	// needs more than its share keep its content *while the others still
	// take a share computed as though it had not* — so the row adds up
	// to more than there is. Two equal Flex columns with natural widths
	// 90 and 10 in 100 pixels came out 90 and 50, and the second child
	// ended 40 pixels past the right edge.
	//
	// Instead: a track whose content exceeds its share is frozen at its
	// content and its width comes out of the pot, then the rest share
	// what is left. Freezing one can push another over its new share, so
	// it repeats until nothing else freezes.
	weightOf := func(i int) float32 {
		w := trackAt(ts, i).Size
		if w <= 0 {
			w = 1
		}
		return w
	}
	fixed := gap * float32(max(n-1, 0))
	var flex []int
	for i := 0; i < n; i++ {
		if trackAt(ts, i).Mode == TrackFlex {
			flex = append(flex, i)
			continue
		}
		fixed += out[i]
	}
	if len(flex) == 0 {
		return out
	}
	for {
		var weight, left float32
		left = avail - fixed
		for _, i := range flex {
			weight += weightOf(i)
		}
		if weight <= 0 {
			break
		}
		froze := false
		for k := 0; k < len(flex); k++ {
			i := flex[k]
			share := left * weightOf(i) / weight
			// The floor is what this track can actually be squeezed to,
			// not what it would like.
			//
			// Freezing at the natural width is right for a child that
			// cannot shrink — a button is as wide as its label — and
			// wrong for one that can. A wrapping label measured with
			// nothing to constrain it reports the width of its whole text
			// on one line: 1453 pixels for a sentence. Treating that as a
			// floor gave its column 1453 of the 400 there were, so the
			// form ran off its own right edge and the height came back as
			// the single line nobody would ever see. Its real floor is
			// its longest word.
			floor := out[i]
			if minc != nil && minc[i] > 0 && minc[i] < floor {
				floor = minc[i]
			}
			if floor > share {
				// Even squeezed it needs more than its share: give it
				// that, and take it out of what the others divide.
				out[i] = floor
				fixed += floor
				flex = append(flex[:k], flex[k+1:]...)
				k--
				froze = true
			}
		}
		if froze {
			continue
		}
		left = avail - fixed
		weight = 0
		for _, i := range flex {
			weight += weightOf(i)
		}
		for _, i := range flex {
			out[i] = left * weightOf(i) / weight
		}
		break
	}
	return out
}

// columns sizes the columns to fill avail (or naturally when avail < 0).
func (g *Grid) columns(avail float32) []float32 {
	_, nc := g.dims()
	content := make([]float32, nc)
	var minc []float32
	if avail >= 0 {
		// The least each column can be squeezed to, which only matters
		// when there is a width to fit into. It costs one more measure
		// per cell, so it is not taken for a natural-size pass.
		minc = make([]float32, nc)
	}
	var spans [][3]float32
	g.visible(func(c widget.Component, cell *GridCell) {
		natural := c.Measure(layout.Unbounded())
		w := ceilPx(natural.X)
		if cell.ColSpan == 1 {
			if cell.Col < nc && w > content[cell.Col] {
				content[cell.Col] = w
			}
			// Only a flexible track can be squeezed, so only those pay
			// for the extra measure.
			if minc != nil && cell.Col < nc && trackAt(g.Cols, cell.Col).Mode == TrackFlex {
				if m := minWidthOf(c, natural); m > minc[cell.Col] {
					minc[cell.Col] = m
				}
			}
			return
		}
		spans = append(spans, [3]float32{float32(cell.Col), float32(cell.ColSpan), w})
	})
	return solve(g.Cols, nc, content, minc, spans, g.colGap(), avail)
}

// minWidthOf is how narrow a column may be squeezed for this child, or 0
// for a child that cannot be squeezed at all.
//
// Whether a child folds is told by its *height*, not its width. Asking
// it to fit in a narrow box cannot answer it on its own, because a child
// that cannot fold is clipped to the constraint just the same as one
// that can, and both come back narrow — so a button asked to fit in a
// pixel says 65 and is then squeezed to 65, which is not a button any
// more. A child that folds gets taller when it is narrowed; one that
// cannot keeps its height.
//
// The probe is a quarter of the child's natural width, with a small
// floor, which makes the answer an approximation of the true min-content
// width — a wrapping label's longest word — and deliberately so: the
// exact value needs a search, and squeezing a label to its longest word
// is not a layout anyone wants anyway. What matters is telling "this can
// give way" from "this cannot", and a quarter is far enough in to settle
// it. One pixel is not: a label with no room for a single rune stops
// wrapping and reports one line, which reads as a child that does not
// fold at all.
func minWidthOf(c widget.Component, natural paintengine2d.Point) float32 {
	if natural.X <= 0 {
		return 0
	}
	probe := natural.X * 0.25
	if probe < 24 {
		probe = 24
	}
	if probe >= natural.X {
		return 0
	}
	got := c.Measure(layout.Constraints{MaxW: probe, MaxH: -1})
	if got.Y <= natural.Y {
		return 0
	}
	if w := ceilPx(min(got.X, probe)); w >= 1 {
		return w
	}
	return 1
}

// MinWidth implements [widget.MinWidther]: the sum of the columns'
// floors, which is the width below which this grid's children start
// leaving its box.
//
// A grid cannot answer this by measuring itself — asked to fit in one
// pixel it reports the one pixel it was constrained to — so it is
// computed from the same floors the column solver uses.
func (g *Grid) MinWidth() float32 {
	_, nc := g.dims()
	if nc == 0 {
		return 0
	}
	floors := make([]float32, nc)
	g.visible(func(c widget.Component, cell *GridCell) {
		if cell.Col >= nc || cell.ColSpan != 1 {
			return
		}
		natural := c.Measure(layout.Unbounded())
		m := ceilPx(natural.X)
		if trackAt(g.Cols, cell.Col).Mode == TrackFlex {
			if got := minWidthOf(c, natural); got > 0 {
				m = got
			}
		}
		if m > floors[cell.Col] {
			floors[cell.Col] = m
		}
	})
	var w float32
	for _, f := range floors {
		w += f
	}
	if nc > 1 {
		w += g.colGap() * float32(nc-1)
	}
	return w
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
		cw := spanLen(cols, cell.Col, cell.ColSpan, g.colGap())
		h := ceilPx(c.Measure(layout.Constraints{MaxW: cw, MaxH: -1}).Y)
		if cell.RowSpan == 1 {
			if cell.Row < nr && h > content[cell.Row] {
				content[cell.Row] = h
			}
			return
		}
		spans = append(spans, [3]float32{float32(cell.Row), float32(cell.RowSpan), h})
	})
	// No minimum for rows: height-for-width is the direction that
	// matters, and nothing here folds sideways to get shorter.
	return solve(g.Rows, nr, content, nil, spans, g.rowGap(), avail)
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
	w := sum(cols, g.colGap())
	if c.HasMaxW() {
		if w > c.MaxW {
			w = c.MaxW
		}
		// Row heights are measured at the widths Arrange will hand out,
		// not at the columns' natural ones.
		//
		// Arrange always calls columns(r.Dx()), which shares the whole
		// width out; Measure only did that when the natural widths did
		// not fit, so a grid with room to spare measured its rows narrow
		// and laid them out wide. A child whose height depends on its
		// width — a wrapping label, a chip field — was therefore measured
		// tall and arranged short, and the form kept the difference as
		// blank space under it.
		//
		// The reported width stays the natural one where it fits, so a
		// grid is not greedy in a row that would otherwise leave it
		// alone.
		cols = g.columns(c.MaxW)
	}
	h := sum(g.rows(cols, -1), g.rowGap())
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (g *Grid) Arrange(r paintengine2d.Rect) {
	g.SetBounds(r)
	cols := g.columns(r.Dx())
	rows := g.rows(cols, r.Dy())
	xs := make([]float32, len(cols)+1)
	for i, w := range cols {
		xs[i+1] = xs[i] + w + g.colGap()
	}
	ys := make([]float32, len(rows)+1)
	for i, h := range rows {
		ys[i+1] = ys[i] + h + g.rowGap()
	}
	g.visible(func(c widget.Component, cell *GridCell) {
		if cell.Col >= len(cols) || cell.Row >= len(rows) {
			return
		}
		box := paintengine2d.XYWH(xs[cell.Col], ys[cell.Row],
			spanLen(cols, cell.Col, cell.ColSpan, g.colGap()), spanLen(rows, cell.Row, cell.RowSpan, g.rowGap()))
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
	rows   []*FormRow
	n      int
}

// FormRow is one caption-and-field row, for showing and hiding the two
// together.
//
// Hiding the field alone leaves its caption behind, and a form whose
// fields depend on a choice — what a keyslot of each kind takes, what a
// protocol needs — otherwise shows every field it might ever need, or is
// torn down and built again. [Form.Row] returns one.
type FormRow struct {
	Label *Label
	Field widget.Component
	form  *Form
}

// SetVisible shows or hides the caption and the field together. The grid
// closes the row's gap, so nothing is left where the row was.
func (r *FormRow) SetVisible(v bool) {
	if r == nil {
		return
	}
	if r.Label != nil {
		r.Label.SetVisible(v)
	}
	if r.Field != nil {
		r.Field.SetVisible(v)
	}
	if r.form != nil {
		r.form.Invalidate()
	}
}

// Visible reports whether the row is showing.
func (r *FormRow) Visible() bool {
	return r != nil && r.Label != nil && r.Label.Visible()
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
	return f.Row(label, field).Label
}

// Row adds a caption-and-field row and returns a handle to both, for
// showing and hiding them together ([FormRow]).
func (f *Form) Row(label string, field widget.Component) *FormRow {
	l := f.addRow(label, field)
	r := &FormRow{Label: l, Field: field, form: f}
	f.rows = append(f.rows, r)
	return r
}

func (f *Form) addRow(label string, field widget.Component) *Label {
	l := NewLabel(label)
	f.labels = append(f.labels, l)
	f.Place(l, f.n, 0)
	if field != nil {
		f.Place(field, f.n, 1)
		// The label names its field for assistive technology (Qt's buddy)
		// — but not where the field is itself a piece of text. A label's
		// accessible name is read *instead of* its text, so a row of
		// caption and value ("Program:", "/usr/bin/mail") came out as
		// "Program:", "Program", and the value, which is the whole reason
		// a consent prompt is on the screen, was never read at all. Qt's
		// buddy is for something that takes input; a value is read as
		// what it says.
		if _, isText := field.(*Label); !isText {
			if nm, ok := field.(interface {
				AccessibleName() string
				SetAccessibleName(string)
			}); ok && nm.AccessibleName() == "" {
				nm.SetAccessibleName(strings.TrimSuffix(strings.TrimSpace(widget.PlainText(label)), ":"))
			}
		}
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

// ceilPx rounds a measured length up to whole pixels, so a child sized to
// its content keeps that width after layout rounding.
func ceilPx(v float32) float32 { return float32(math.Ceil(float64(v) - 1e-3)) }

// colGap and rowGap are the gaps at the look's scale, or as given when
// RawSpacing says the caller has already scaled them.
func (g *Grid) colGap() float32 { return g.gapOf(g.ColGap) }
func (g *Grid) rowGap() float32 { return g.gapOf(g.RowGap) }

func (g *Grid) gapOf(v float32) float32 {
	if g.RawSpacing {
		return v
	}
	return style.Dip(g.Look(), v)
}
