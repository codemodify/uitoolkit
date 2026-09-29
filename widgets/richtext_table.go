package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/richtext"
)

// Laying a table row out.
//
// A table's column widths belong to the whole run of rows, not to any
// one of them, so a row cannot be laid out alone: they are computed for
// the run and cached on the widget, keyed by the run's first block. Every
// row of the run then reads the same answer, and the cache is dropped
// with the rest of the layout whenever the document or the look changes.
//
// Each cell becomes one fragment of a single line, placed at its column's
// x and given its column's width as its advance. That is all the
// positioning a table needs here, and it means the caret, the selection
// and the painter go on working on lines and fragments exactly as they do
// for a paragraph — the row's flattened text (cells joined by tabs) is
// what they count in.
//
// A cell is laid out the way a paragraph is — its own spans, its own
// faces, wrapped to its column — so bold, code and links survive in a
// <td>. It used to be drawn as its plain text in one face fitted to one
// line, which lost a newsletter's table of links and cut a long cell
// rather than folding it.
//
// The row is as tall as its tallest cell, and every cell's k-th line
// shares the row's k-th line, so the caret, the selection and the
// painter go on working on lines and fragments exactly as they do for a
// paragraph.
//
// What this still does not do: span columns, or nest a table in a cell.

// tableCols is the width of each column of a run, and the widths the
// cells were measured at.
type tableCols struct {
	sig   rtSig
	width float32
	cols  []float32
}

// layoutTableRow places b's cells at the run's shared column widths.
func (t *RichText) layoutTableRow(b *richtext.Block, width float32, lay *rtLayout) *rtLayout {
	run := t.runFor(b)
	cols := t.columnsFor(run, width)
	pad := t.cellPad()
	kind := richtext.TableRow
	if b.Header() {
		// A header cell's spans are drawn bold on top of whatever they
		// already ask for, which is what a <th> means.
		kind = richtext.Heading
	}

	// Each cell laid out on its own, at its column's width.
	type cell struct {
		lines []rtLine
		x     float32
	}
	cells := make([]cell, len(cols))
	pos := 0
	var x float32
	for j := range cols {
		var spans []richtext.Span
		if j < len(b.Cells) {
			spans = b.Cells[j]
		}
		room := max(cols[j]-pad*2, t.dip(8))
		n := cellRunes(spans)
		segs := t.segmentsOf(spans, kind, b.Level, pos, room)
		cells[j] = cell{lines: breakSegments(segs, room, pos+n), x: x + pad}
		pos += n + 1 // the tab that follows this cell
		x += cols[j]
	}
	// The last cell has no tab after it.
	if pos > 0 {
		pos--
	}

	rows := 0
	for _, c := range cells {
		rows = max(rows, len(c.lines))
	}
	if rows == 0 {
		rows = 1
	}

	before, after := t.spacing(b)
	// A paragraph's space after the last row of a table, and nothing
	// between the rows of one — they are one grid. Without it the block
	// that follows a table started right under its bottom rule, so a
	// quote's accent rule butted straight onto it.
	if !t.rowFollows(b) {
		after = t.dip(6)
	}
	lay.before = before
	lineGap := t.dip(2)
	base := t.face(richtext.Style{}, richtext.TableRow, 0)

	lines := make([]rtLine, rows)
	y := before
	for i := 0; i < rows; i++ {
		ln := &lines[i]
		asc, desc := base.Ascent, fontDescent(base)
		for _, c := range cells {
			if i >= len(c.lines) {
				continue
			}
			src := c.lines[i]
			var fx float32
			for _, f := range src.frags {
				f.x = c.x + fx
				if f.img != nil {
					f.w = f.imgW
					asc = max(asc, f.imgH)
				} else {
					f.w = f.font.Advance(f.text)
					asc = max(asc, f.font.Ascent)
					desc = max(desc, fontDescent(f.font))
				}
				fx += f.w
				ln.frags = append(ln.frags, f)
			}
		}
		ln.ascent, ln.descent = asc, desc
		ln.base = float32(math.Round(float64(lineGap*0.5 + asc)))
		ln.h = float32(math.Ceil(float64(asc + desc + lineGap)))
		ln.x, ln.w = 0, x
		ln.y = y
		y += ln.h
	}
	// The row is one block to everything that counts in offsets: the
	// first line starts at 0 and the last ends at the row's length, so a
	// caret dragged down a column still walks the flattened text.
	lines[0].start = 0
	for i := 0; i < rows-1; i++ {
		lines[i].soft = true
		lines[i].end = lines[i+1].start
	}
	lines[rows-1].end = b.Len()

	// Cell padding at the top, so text does not sit on the rule above it.
	pv := t.dip(3)
	for i := range lines {
		lines[i].y += pv
	}
	lay.lines = lines
	lay.h = before + (y - before) + pv*2 + after
	lay.cols = cols
	return lay
}

// cellRunes is how many runes a cell's spans hold, which is what the
// row's flattened text counts it as.
func cellRunes(spans []richtext.Span) int {
	n := 0
	for _, s := range spans {
		if s.Image != nil {
			n++
			continue
		}
		n += len([]rune(s.Text))
	}
	return n
}

// runFor is the run of table rows b belongs to.
func (t *RichText) runFor(b *richtext.Block) []*richtext.Block {
	bs := t.doc.Blocks()
	for i, x := range bs {
		if x == b {
			return t.doc.TableRun(i)
		}
	}
	return []*richtext.Block{b}
}

// columnsFor is the run's column widths at this width, measured once and
// cached: every row of a table asks the same question and must get the
// same answer, or the columns would not line up.
func (t *RichText) columnsFor(run []*richtext.Block, width float32) []float32 {
	if len(run) == 0 {
		return nil
	}
	key := run[0]
	sig := t.sig()
	if t.tcols == nil {
		t.tcols = map[*richtext.Block]*tableCols{}
	}
	if got, ok := t.tcols[key]; ok && got.sig == sig && got.width == width {
		return got.cols
	}
	cols := t.measureColumns(run, width)
	t.tcols[key] = &tableCols{sig: sig, width: width, cols: cols}
	return cols
}

// measureColumns gives each column the width of its widest cell, then
// shares out what is left or takes back what is over.
//
// Over-wide is the case that matters: a mail table is routinely wider
// than the pane it is read in, and the choice is between a column nobody
// can see and a column that is elided. Columns are scaled down in
// proportion, with a floor, so the widest column gives up the most —
// which is nearly always the one with room to spare.
func (t *RichText) measureColumns(run []*richtext.Block, width float32) []float32 {
	n := richtext.TableColumns(run)
	if n == 0 {
		return nil
	}
	pad := t.cellPad()
	want := make([]float32, n)
	for _, b := range run {
		kind := richtext.TableRow
		if b.Header() {
			kind = richtext.Heading
		}
		for j := 0; j < n && j < len(b.Cells); j++ {
			// Measured span by span in the face each one asks for, so a
			// bold cell is not measured in the regular face and then
			// drawn wider than the column it was given.
			var w float32
			for _, sp := range b.Cells[j] {
				if sp.Image != nil {
					iw, _ := t.imageBox(sp.Image, math.MaxFloat32)
					w += iw
					continue
				}
				w += t.face(sp.Style, kind, b.Level).Advance(drawText(sp.Text))
			}
			if w += pad * 2; w > want[j] {
				want[j] = w
			}
		}
	}
	floor := t.dip(28)
	var total float32
	for j := range want {
		want[j] = max(want[j], floor)
		total += want[j]
	}
	if total <= width || total <= 0 {
		return want
	}
	// Too wide: take the excess out of the columns in proportion to what
	// each has above the floor, so a column that is already at the floor
	// gives up nothing.
	var spare float32
	for _, w := range want {
		spare += w - floor
	}
	if spare <= 0 {
		return want
	}
	k := min((total-width)/spare, 1)
	for j := range want {
		want[j] = float32(math.Round(float64(want[j] - (want[j]-floor)*k)))
	}
	return want
}

// paintBlockChrome draws what a block is made of besides its text: the
// rule down a quote's side, the line a horizontal rule is, and a table's
// header rule and column separators.
//
// It is one function rather than three because all three are the same
// shape — a few lines in the divider colour, positioned from the block's
// own layout — and because they all have to happen before the text, so a
// selection's highlight covers them rather than the other way round.
func (t *RichText) paintBlockChrome(ctx *paintengine2d.Context, b *richtext.Block,
	lay *rtLayout, i int, in paintengine2d.Rect, oy float32, text paintengine2d.Color) {
	if b == nil || lay == nil || len(lay.lines) == 0 {
		return
	}
	lk := t.Look()
	rule := lk.Palette().Divider
	if rule.A == 0 {
		rule = text.WithAlpha(0.35)
	}
	switch b.Kind {
	case richtext.Quote:
		// One rule per level, at each level's indent, so a quote inside
		// a quote shows as two — which is how a mail thread's depth is
		// read at a glance.
		w := t.quoteRuleWidth()
		top, bottom := oy+lay.lines[0].y, oy+lay.h-lay.before
		for lv := 0; lv <= min(max(b.Level, 0), richtext.MaxLevel); lv++ {
			x := in.Min.X + t.quoteIndent()*float32(lv) + t.quoteIndent()*0.4
			ctx.DrawRect(paintengine2d.XYWH(x, top, w, max(bottom-top, 0)),
				paintengine2d.Fill(lk.Palette().Accent.WithAlpha(0.55)))
		}
	case richtext.Rule:
		ln := lay.lines[0]
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, oy+ln.y, t.laidW, t.ruleHeight()),
			paintengine2d.Fill(rule))
	case richtext.TableRow:
		ln := lay.lines[0]
		y, h := oy+ln.y, ln.h
		hair := max(t.dip(1), 1)
		// A line between the columns, and one under the row. The run's
		// last row gets no line under it — the table ends there, and a
		// trailing rule reads as an empty row.
		var x float32
		for j, w := range lay.cols {
			if j > 0 {
				ctx.DrawRect(paintengine2d.XYWH(in.Min.X+x, y, hair, h),
					paintengine2d.Fill(rule.WithAlpha(rule.A*0.6)))
			}
			x += w
		}
		under := rule
		if b.Header() {
			// Heavier under the header, which is what says the first row
			// is a heading rather than data.
			under = text.WithAlpha(0.55)
			hair = max(t.dip(1.5), 1)
		}
		if b.Header() || t.hasRowAfter(i) {
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, y+h-hair, x, hair), paintengine2d.Fill(under))
		}
	}
}

// hasRowAfter reports whether block i is followed by another table row,
// so the last row of a table gets no rule under it.
func (t *RichText) hasRowAfter(i int) bool {
	bs := t.doc.Blocks()
	return i+1 < len(bs) && bs[i+1].Kind == richtext.TableRow
}

// rowFollows reports whether another row of the same table comes after
// b, which is how the last row of a run knows to leave space after it.
func (t *RichText) rowFollows(b *richtext.Block) bool {
	bs := t.doc.Blocks()
	for i, x := range bs {
		if x == b {
			return i+1 < len(bs) && bs[i+1].Kind == richtext.TableRow
		}
	}
	return false
}
