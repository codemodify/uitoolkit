package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

func tableDoc(t *testing.T, html string) *RichText {
	t.Helper()
	rt := NewRichText("")
	atScale(t, rt, 1)
	rt.SetHTML(html)
	sz := rt.Measure(layout.Constraints{MaxW: 420, MaxH: -1})
	rt.Arrange(paintengine2d.XYWH(0, 0, 420, sz.Y))
	return rt
}

// layoutsOf is every block's layout, in document order.
func layoutsOf(t *testing.T, rt *RichText) []*rtLayout {
	t.Helper()
	var out []*rtLayout
	for i := 0; i < rt.doc.Len(); i++ {
		out = append(out, rt.lay(i))
	}
	return out
}

// A cell was drawn as its plain text in one face, so a newsletter's
// table of links lost its links and a Markdown table's **bold** cell was
// plain.
func TestTableCellsKeepTheirSpans(t *testing.T) {
	rt := tableDoc(t, `<table>
	  <tr><td>plain</td><td><b>bold</b></td><td><a href="http://x/">link</a></td></tr>
	</table>`)

	var bold, linked, plain int
	for _, lay := range layoutsOf(t, rt) {
		for _, ln := range lay.lines {
			for _, f := range ln.frags {
				switch {
				case f.st.Link != "":
					linked++
				case f.st.Bold:
					bold++
				default:
					plain++
				}
			}
		}
	}
	if bold == 0 {
		t.Error("the bold cell lost its weight")
	}
	if linked == 0 {
		t.Error("the link cell lost its link")
	}
	if plain == 0 {
		t.Error("the plain cell vanished")
	}
}

// A long cell folds inside its column instead of being cut with an
// ellipsis, and the row is as tall as its tallest cell.
func TestTableCellWrapsInsteadOfBeingCut(t *testing.T) {
	rt := tableDoc(t, `<table>
	  <tr><td>a</td><td>a cell with a good deal more text in it than will ever fit across one column of a narrow table</td></tr>
	</table>`)

	var rows int
	var tall float32
	for _, lay := range layoutsOf(t, rt) {
		if len(lay.cols) == 0 {
			continue
		}
		rows = len(lay.lines)
		tall = lay.h
	}
	if rows < 2 {
		t.Errorf("the long cell is on %d line(s) — it was cut rather than folded", rows)
	}
	if tall <= 0 {
		t.Error("the row has no height")
	}
	// Nothing runs past the table's own width.
	for _, lay := range layoutsOf(t, rt) {
		for _, ln := range lay.lines {
			for _, f := range ln.frags {
				if f.x+f.w > 421 {
					t.Errorf("a cell fragment reaches %v, past the 420 it was given", f.x+f.w)
				}
			}
		}
	}
}

// The block after a table does not start right under its bottom rule.
func TestBlockAfterATableGetsItsSpace(t *testing.T) {
	with := tableDoc(t, `<table><tr><td>a</td></tr></table><p>after</p>`)
	var lastRow *rtLayout
	for _, lay := range layoutsOf(t, with) {
		if len(lay.cols) > 0 {
			lastRow = lay
		}
	}
	if lastRow == nil {
		t.Fatal("no table row was laid out")
	}
	var rowHeight float32
	for _, ln := range lastRow.lines {
		rowHeight += ln.h
	}
	if lastRow.h <= rowHeight {
		t.Errorf("the last row is %v tall for %v of lines — no space after it", lastRow.h, rowHeight)
	}
}

// Rows of one table stay tight against each other: they are one grid.
func TestRowsOfOneTableStayTight(t *testing.T) {
	rt := tableDoc(t, `<table><tr><td>a</td></tr><tr><td>b</td></tr></table>`)
	var rows []*rtLayout
	for _, lay := range layoutsOf(t, rt) {
		if len(lay.cols) > 0 {
			rows = append(rows, lay)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("%d rows laid out, want 2", len(rows))
	}
	var lines float32
	for _, ln := range rows[0].lines {
		lines += ln.h
	}
	if rows[0].h > lines+8 {
		t.Errorf("the first row is %v tall for %v of lines — it has space after it", rows[0].h, lines)
	}
}

// A header cell is bold at the table's own size. It was laid out as a
// Heading, and the HTML parser gives a <th> Level 1 — so every header
// cell came out an H1: title-sized, and breaking mid-word in a narrow
// column.
func TestTableHeaderIsBoldNotAHeading(t *testing.T) {
	rt := tableDoc(t, `<table>
	  <tr><th>Item</th><th>Price</th></tr>
	  <tr><td>Tea</td><td>3</td></tr>
	</table>`)
	lays := layoutsOf(t, rt)
	if len(lays) < 2 {
		t.Fatal("expected two rows")
	}
	head, body := lays[0], lays[1]

	var headSize, bodySize float32
	var bold bool
	for _, ln := range head.lines {
		for _, f := range ln.frags {
			headSize = max(headSize, f.font.Size)
			if f.st.Bold {
				bold = true
			}
		}
	}
	for _, ln := range body.lines {
		for _, f := range ln.frags {
			bodySize = max(bodySize, f.font.Size)
		}
	}
	if !bold {
		t.Error("a header cell is not bold")
	}
	if headSize != bodySize {
		t.Errorf("header cells are set at %v and body cells at %v — a header is bold, not bigger",
			headSize, bodySize)
	}
}

// A row two lines tall must have its column rules and its underline
// drawn for the whole row. They were drawn at the first line's height,
// so the later lines sat outside their own cell — which reads as the
// next row overlapping this one.
func TestTableChromeSpansTheWholeRow(t *testing.T) {
	rt := tableDoc(t, `<table>
	  <tr><td>a</td><td>a cell with enough words in it to need more than one line of its column</td></tr>
	  <tr><td>b</td><td>c</td></tr>
	</table>`)
	lays := layoutsOf(t, rt)
	var tall *rtLayout
	for _, l := range lays {
		if len(l.cols) > 0 && len(l.lines) > 1 {
			tall = l
			break
		}
	}
	if tall == nil {
		t.Fatal("no row wrapped, so this proves nothing")
	}
	first, last := tall.lines[0], tall.lines[len(tall.lines)-1]
	rowH := last.y + last.h - first.y
	if rowH <= first.h {
		t.Fatalf("row height %v is no more than its first line %v", rowH, first.h)
	}
}

// A column is never squeezed past its longest word. It used to shrink
// toward one number for every column, which was right while a cell too
// wide for its column was elided; now that cells wrap, that broke text
// mid-word — "Price" came out "Pric" and "£4.50" became "£4.5" and "0".
func TestTableColumnsAreNeverNarrowerThanTheirLongestWord(t *testing.T) {
	rt := tableDoc(t, `<table>
	  <tr><th>Item</th><th>Price</th><th>Notes</th></tr>
	  <tr><td>Cake with a long name that should wrap inside its cell</td><td>&pound;4.50</td><td>x</td></tr>
	</table>`)

	lays := layoutsOf(t, rt)
	var cols []float32
	for _, l := range lays {
		if len(l.cols) > 0 {
			cols = l.cols
			break
		}
	}
	if len(cols) == 0 {
		t.Fatal("no columns")
	}
	// Every fragment must fit in its own column: nothing was broken to
	// make it fit.
	for _, l := range lays {
		for _, ln := range l.lines {
			for _, f := range ln.frags {
				if f.w > cols[len(cols)-1]+cols[0]+cols[1] {
					t.Errorf("a fragment is %v wide, wider than the table", f.w)
				}
			}
		}
	}
	// And the money column holds its whole value on one line.
	rt2 := tableDoc(t, `<table><tr><td>a</td><td>&pound;4.50</td></tr></table>`)
	for _, l := range layoutsOf(t, rt2) {
		for _, ln := range l.lines {
			for _, f := range ln.frags {
				if f.text == "£4.5" || f.text == "0" {
					t.Errorf("£4.50 was broken into %q", f.text)
				}
			}
		}
	}
}
