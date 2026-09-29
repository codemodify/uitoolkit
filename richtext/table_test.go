package richtext

import (
	"strings"
	"testing"
)

func docHTML(t *testing.T, html string) *Doc {
	t.Helper()
	d := New()
	if err := d.SetHTML(html); err != nil {
		t.Fatal(err)
	}
	return d
}

// A table comes in as rows of cells rather than as its text run
// together, which is what it used to be: richtext read <table> for the
// characters inside it and nothing else.
func TestTableRoundTrip(t *testing.T) {
	const html = `<table><tr><th>Name</th><th>Size</th></tr>` +
		`<tr><td>report.pdf</td><td>1.2 MB</td></tr>` +
		`<tr><td>notes.txt</td><td>4 kB</td></tr></table>`
	d := docHTML(t, html)
	bs := d.Blocks()
	if len(bs) != 3 {
		t.Fatalf("%d blocks, want 3 rows: %v", len(bs), blockKinds(bs))
	}
	for i, b := range bs {
		if b.Kind != TableRow {
			t.Fatalf("block %d is %v", i, b.Kind)
		}
		if len(b.Cells) != 2 {
			t.Fatalf("row %d has %d cells", i, len(b.Cells))
		}
	}
	if !bs[0].Header() || bs[1].Header() {
		t.Fatalf("header rows: %v %v", bs[0].Header(), bs[1].Header())
	}
	if got := bs[1].CellText(0); got != "report.pdf" {
		t.Fatalf("cell %q", got)
	}
	// The row's text is also flat, so the caret and the selection work
	// without knowing what a table is.
	if got := bs[1].Text(); got != "report.pdf\t1.2 MB" {
		t.Fatalf("row text %q", got)
	}
	// And it comes back out as a table.
	out := d.HTML()
	for _, want := range []string{"<table>", "<th>Name</th>", "<td>report.pdf</td>", "</table>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("HTML has no %q:\n%s", want, out)
		}
	}
	// Round-trips to the same shape.
	again := docHTML(t, out)
	if len(again.Blocks()) != 3 {
		t.Fatalf("second pass: %v", blockKinds(again.Blocks()))
	}
	if again.Blocks()[0].CellText(1) != "Size" {
		t.Fatalf("second pass lost a cell: %q", again.Blocks()[0].CellText(1))
	}
}

// A blockquote is quoted text at a level rather than a plain paragraph,
// so a mail thread's nesting survives.
func TestQuoteRoundTrip(t *testing.T) {
	const html = `<blockquote><p>Ada wrote:</p>` +
		`<blockquote><p>Grace wrote this first.</p></blockquote>` +
		`<p>and then this.</p></blockquote><p>After.</p>`
	d := docHTML(t, html)
	bs := d.Blocks()
	if len(bs) != 4 {
		t.Fatalf("%d blocks: %v", len(bs), blockKinds(bs))
	}
	want := []struct {
		kind  Kind
		level int
		text  string
	}{
		{Quote, 0, "Ada wrote:"},
		{Quote, 1, "Grace wrote this first."},
		{Quote, 0, "and then this."},
		{Paragraph, 0, "After."},
	}
	for i, w := range want {
		if bs[i].Kind != w.kind || bs[i].Level != w.level || bs[i].Text() != w.text {
			t.Fatalf("block %d: %v level %d %q, want %v level %d %q",
				i, bs[i].Kind, bs[i].Level, bs[i].Text(), w.kind, w.level, w.text)
		}
	}
	out := d.HTML()
	if strings.Count(out, "<blockquote>") != 2 || strings.Count(out, "</blockquote>") != 2 {
		t.Fatalf("quotes not bracketed:\n%s", out)
	}
	again := docHTML(t, out)
	for i, w := range want {
		b := again.Blocks()[i]
		if b.Kind != w.kind || b.Level != w.level {
			t.Fatalf("second pass block %d: %v level %d", i, b.Kind, b.Level)
		}
	}
}

// An <hr> is a block of its own, which it was not: it used to be
// dropped entirely.
func TestRuleRoundTrip(t *testing.T) {
	d := docHTML(t, `<p>Above</p><hr><p>Below</p>`)
	bs := d.Blocks()
	if len(bs) != 3 || bs[1].Kind != Rule {
		t.Fatalf("blocks %v", blockKinds(bs))
	}
	if bs[1].Text() != "" {
		t.Fatalf("a rule has text: %q", bs[1].Text())
	}
	if !strings.Contains(d.HTML(), "<hr>") {
		t.Fatalf("no rule in:\n%s", d.HTML())
	}
}

// Malformed tables are the common kind: a cell outside a row, a row left
// open. Losing the text would be worse than an odd shape.
func TestMalformedTableKeepsItsText(t *testing.T) {
	d := docHTML(t, `<table><td>loose</td><tr><td>a</td><td>b</td></table>`)
	var text []string
	for _, b := range d.Blocks() {
		if b.Kind == TableRow {
			for j := range b.Cells {
				text = append(text, b.CellText(j))
			}
		}
	}
	for _, want := range []string{"loose", "a", "b"} {
		if !contains(text, want) {
			t.Fatalf("%q was lost, got %v", want, text)
		}
	}
}

// Built by hand, a row keeps its cells and its flat text in step.
func TestNewTableRow(t *testing.T) {
	r := NewTableRowText(true, "One", "Two", "Three")
	if !r.Header() || len(r.Cells) != 3 {
		t.Fatalf("header=%v cells=%d", r.Header(), len(r.Cells))
	}
	if got := r.Text(); got != "One\tTwo\tThree" {
		t.Fatalf("text %q", got)
	}
	r.Cells[1] = []Span{{Text: "Deux"}}
	r.SyncCells()
	if got := r.Text(); got != "One\tDeux\tThree" {
		t.Fatalf("after SyncCells: %q", got)
	}
}

// TableRun is the run of rows a row belongs to, asked from any of them.
func TestTableRun(t *testing.T) {
	d := docHTML(t, `<p>a</p><table><tr><td>1</td></tr><tr><td>2</td></tr></table><p>b</p>`)
	if run := d.TableRun(0); run != nil {
		t.Fatalf("a paragraph is in a run of %d", len(run))
	}
	for _, i := range []int{1, 2} {
		run := d.TableRun(i)
		if len(run) != 2 {
			t.Fatalf("from block %d the run is %d rows", i, len(run))
		}
		if TableColumns(run) != 1 {
			t.Fatalf("columns %d", TableColumns(run))
		}
	}
}

func blockKinds(bs []*Block) []Kind {
	out := make([]Kind, len(bs))
	for i, b := range bs {
		out[i] = b.Kind
	}
	return out
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
