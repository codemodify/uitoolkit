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
