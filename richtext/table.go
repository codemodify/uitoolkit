package richtext

import "strings"

// Tables, quotes and rules.
//
// A table is a run of consecutive [TableRow] blocks and nothing else:
// there is no table object, no column spanning and no nesting. That is
// enough to show the tables that arrive in mail and in Markdown as
// tables rather than as their aligned text, and it keeps a document a
// flat list of blocks — which is what makes the caret, the selection and
// every offset in this package work without any of them knowing what a
// table is.
//
// A row's text lives twice: in Cells, which the layout reads, and
// flattened into Spans with a tab between cells, which everything else
// reads. They are kept in step by [NewTableRow] and by the parser;
// [Block.SyncCells] is how a caller that edits Cells puts Spans back.

// CellSeparator is what joins a row's cells in its flattened Spans. A
// tab, because that is what a table pasted as plain text uses and what
// every spreadsheet reads back.
const CellSeparator = "\t"

// NewTableRow builds a row from its cells. header marks the row as the
// table's heading, which the layout draws in bold with a rule under it.
func NewTableRow(header bool, cells ...[]Span) *Block {
	b := &Block{Kind: TableRow}
	if header {
		b.Level = 1
	}
	b.Cells = make([][]Span, len(cells))
	for i, c := range cells {
		b.Cells[i] = normalize(c)
	}
	b.SyncCells()
	return b
}

// NewTableRowText is [NewTableRow] from plain strings.
func NewTableRowText(header bool, cells ...string) *Block {
	out := make([][]Span, len(cells))
	for i, c := range cells {
		out[i] = []Span{{Text: c}}
	}
	return NewTableRow(header, out...)
}

// Header reports whether this row is the table's heading row.
func (b *Block) Header() bool { return b != nil && b.Kind == TableRow && b.Level == 1 }

// SyncCells rebuilds a row's Spans from its Cells, with a tab between
// them. A caller that has edited Cells calls it; the parser and
// [NewTableRow] already do.
func (b *Block) SyncCells() {
	if b == nil || b.Kind != TableRow {
		return
	}
	var spans []Span
	for i, cell := range b.Cells {
		if i > 0 {
			spans = append(spans, Span{Text: CellSeparator})
		}
		spans = append(spans, cell...)
	}
	b.Spans = normalize(spans)
	b.n = 0
}

// NewRule is a horizontal rule.
func NewRule() *Block { return &Block{Kind: Rule} }

// NewQuote is a quoted paragraph at the given nesting, 0 the outermost.
func NewQuote(level int, spans ...Span) *Block {
	return NewBlock(Quote, min(max(level, 0), MaxLevel), spans...)
}

// TableRun is the run of table rows this block starts, or nil when the
// block at i is not one.
//
// It is what a layout asks for: the column widths of a table are a
// property of the whole run, so a row cannot be laid out alone.
func (d *Doc) TableRun(i int) []*Block {
	bs := d.Blocks()
	if i < 0 || i >= len(bs) || bs[i].Kind != TableRow {
		return nil
	}
	// Back to the first row of the run, so any row answers for the run
	// it is in rather than for the one that starts at it.
	start := i
	for start > 0 && bs[start-1].Kind == TableRow {
		start--
	}
	end := start
	for end < len(bs) && bs[end].Kind == TableRow {
		end++
	}
	return bs[start:end]
}

// TableColumns is how many columns the widest row of a run has.
func TableColumns(run []*Block) int {
	n := 0
	for _, b := range run {
		if len(b.Cells) > n {
			n = len(b.Cells)
		}
	}
	return n
}

// CellText is cell j's text, or "" when the row is shorter than that.
func (b *Block) CellText(j int) string {
	if b == nil || j < 0 || j >= len(b.Cells) {
		return ""
	}
	var sb strings.Builder
	for _, s := range b.Cells[j] {
		sb.WriteString(s.Text)
	}
	return sb.String()
}
