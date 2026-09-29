package widgets

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
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
// What this does not do: wrap a cell over several lines, span columns, or
// nest a table in a cell. A cell whose text is wider than its column is
// elided. Mail tables are small and wide-ish; a cell that needs a
// paragraph of its own is a document this widget is not for.

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
	face := lay.base
	if b.Header() {
		face = t.boldFace(lay.base)
	}

	var frags []rtFrag
	var x float32
	// Offsets into the row's flattened text, so a caret in a cell is a
	// caret in the block: cells are joined by one tab each.
	pos := 0
	for j := range cols {
		text := ""
		if j < len(b.Cells) {
			text = drawText(b.CellText(j))
		}
		room := max(cols[j]-pad*2, 0)
		shown := text
		if face.Advance(shown) > room {
			shown = face.Fit(shown, room)
		}
		n := len([]rune(text))
		frags = append(frags, rtFrag{
			start: pos, end: pos + n, text: shown, font: face,
			x: x + pad, w: max(cols[j]-pad, 0),
		})
		pos += n + 1 // the tab that follows this cell
		x += cols[j]
	}
	// The last cell has no tab after it.
	if pos > 0 {
		pos--
	}

	h := face.Height() + t.dip(6)
	before, after := t.spacing(b)
	lay.before = before
	lay.lines = []rtLine{{
		start: 0, end: b.Len(), y: before, h: h,
		base: face.Ascent + t.dip(3), x: 0, w: x,
		frags: frags, ascent: face.Ascent, descent: fontDescent(face),
	}}
	lay.h = before + h + after
	lay.cols = cols
	return lay
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
	base := t.face(richtext.Style{}, richtext.TableRow, 0)
	bold := t.boldFace(base)
	want := make([]float32, n)
	for _, b := range run {
		f := base
		if b.Header() {
			f = bold
		}
		for j := 0; j < n && j < len(b.Cells); j++ {
			w := f.Advance(drawText(b.CellText(j))) + pad*2
			if w > want[j] {
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

// boldFace is f at bold weight, for a header row.
func (t *RichText) boldFace(f *style.Font) *style.Font {
	if f == nil {
		return nil
	}
	return style.BakeFamily(f.Family, style.WeightBold, f.Size, f.Color)
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
