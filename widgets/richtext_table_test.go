package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
)

func richTextWith(t *testing.T, html string, w, h float32) (*RichText, *paintengine2d.Image) {
	t.Helper()
	rt := NewRichText("")
	rt.SetLook(style.DarkLook())
	rt.SetHost(&host{})
	rt.SetHTML(html)
	rt.Measure(layout.Loose(w, h))
	rt.Arrange(paintengine2d.XYWH(0, 0, w, h))
	img := paintengine2d.NewImage(int(w), int(h))
	rt.Paint(paintengine2d.NewContext(img))
	return rt, img
}

// A table's columns line up: every row of a run is laid out at the same
// widths, because the widths belong to the run rather than to any row.
func TestTableColumnsLineUp(t *testing.T) {
	rt, _ := richTextWith(t, `<table>`+
		`<tr><th>Name</th><th>Size</th></tr>`+
		`<tr><td>a-very-long-file-name.pdf</td><td>1.2 MB</td></tr>`+
		`<tr><td>x</td><td>4 kB</td></tr></table>`, 400, 200)

	var cols [][]float32
	for i := 0; i < rt.doc.Len(); i++ {
		if rt.doc.Block(i).Kind == richtext.TableRow {
			cols = append(cols, rt.lay(i).cols)
		}
	}
	if len(cols) != 3 {
		t.Fatalf("%d rows laid out", len(cols))
	}
	for i := 1; i < len(cols); i++ {
		if len(cols[i]) != len(cols[0]) {
			t.Fatalf("row %d has %d columns, row 0 has %d", i, len(cols[i]), len(cols[0]))
		}
		for j := range cols[0] {
			if cols[i][j] != cols[0][j] {
				t.Fatalf("column %d is %v on row %d and %v on row 0", j, cols[i][j], i, cols[0][j])
			}
		}
	}
	// The first column is the wider one: it holds the long file name.
	if cols[0][0] <= cols[0][1] {
		t.Fatalf("columns %v — the long one is not wider", cols[0])
	}
	// And the cells of one row start at the column offsets.
	ln := rt.lay(1).lines[0]
	if len(ln.frags) != 2 {
		t.Fatalf("%d cells in the row's line", len(ln.frags))
	}
	if ln.frags[1].x <= ln.frags[0].x {
		t.Fatalf("cell xs %v %v", ln.frags[0].x, ln.frags[1].x)
	}
}

// A table wider than the pane is squeezed rather than run off the edge:
// a column nobody can see is worse than one that is elided.
func TestWideTableFitsThePane(t *testing.T) {
	const html = `<table><tr>` +
		`<td>a very long first column indeed</td>` +
		`<td>a very long second column indeed</td>` +
		`<td>a very long third column indeed</td></tr></table>`
	rt, _ := richTextWith(t, html, 200, 120)
	cols := rt.lay(0).cols
	var total float32
	for _, w := range cols {
		total += w
	}
	if total > 200 {
		t.Fatalf("the table is %v wide in a pane of 200: %v", total, cols)
	}
	if len(cols) != 3 {
		t.Fatalf("%d columns", len(cols))
	}
}

// A quote, a rule and a table each paint something a paragraph does not.
func TestBlockChromePaints(t *testing.T) {
	plain := func(html string) *paintengine2d.Image {
		_, img := richTextWith(t, html, 320, 160)
		return img
	}
	for _, c := range []struct{ name, quiet, loud string }{
		{"quote", `<p>Ada wrote this.</p>`, `<blockquote><p>Ada wrote this.</p></blockquote>`},
		{"rule", `<p>a</p><p>b</p>`, `<p>a</p><hr><p>b</p>`},
		{"table", `<p>Name Size</p>`, `<table><tr><th>Name</th><th>Size</th></tr><tr><td>a</td><td>b</td></tr></table>`},
	} {
		if n := pixelsDiffering(plain(c.quiet), plain(c.loud)); n < 20 {
			t.Errorf("%s: only %d pixels differ from the plain version", c.name, n)
		}
	}
}

// A nested quote shows its depth: two rules rather than one.
func TestNestedQuoteDrawsTwoRules(t *testing.T) {
	one := func(html string) *paintengine2d.Image {
		_, img := richTextWith(t, html, 320, 160)
		return img
	}
	shallow := one(`<blockquote><p>Ada wrote this.</p></blockquote>`)
	deep := one(`<blockquote><blockquote><p>Ada wrote this.</p></blockquote></blockquote>`)
	if n := pixelsDiffering(shallow, deep); n < 20 {
		t.Fatalf("a quote inside a quote looks the same: %d pixels differ", n)
	}
}

// The row's flattened text is what the caret counts in, so a click in a
// table lands somewhere sensible rather than panicking.
func TestCaretInATable(t *testing.T) {
	rt, _ := richTextWith(t, `<table><tr><td>one</td><td>two</td></tr></table>`, 320, 120)
	blk := rt.doc.Block(0)
	if got := blk.Text(); got != "one\ttwo" {
		t.Fatalf("row text %q", got)
	}
	for x := float32(0); x < 320; x += 7 {
		pos, ok := rt.posAt(paintengine2d.Pt(x, 12))
		if !ok {
			continue
		}
		if pos.Block < 0 || pos.Block >= rt.doc.Len() {
			t.Fatalf("x=%v gave block %d", x, pos.Block)
		}
		if pos.Off < 0 || pos.Off > blk.Len() {
			t.Fatalf("x=%v gave offset %d of %d", x, pos.Off, blk.Len())
		}
	}
}
