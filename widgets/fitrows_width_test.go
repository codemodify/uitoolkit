package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// **A view that sizes itself to its rows must count them at the width it
// will be laid out at, not the one it was offered.** Layout rounds each
// edge of a box to the nearest pixel, so a view offered 282.5 is handed
// 282 or 283 — and a path measured as two rows at 282.5 wraps to three at
// 282, with the third row behind the scroll bar. At a fractional display
// scale that is the common case, because a column's inner width is
// fractional.
//
// The rule lives in TextArea.rowsFor, which floors the box it is asked
// about; this is the test that says so from the outside, across the
// scales and the widths where the two answers differ.
func TestAFitRowsViewCountsRowsAtTheWidthItWillGet(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75} {
		lk := style.WithScale(style.DarkLook(), scale)
		a := NewTextArea("/home/person/Mail/Local Folders/Archive/2026/October/invitations and other long names", "", nil)
		a.FitRows, a.MinRows, a.MaxRows = true, 1, 12
		a.SetLook(lk)
		a.SetHost(&host{})
		pad := a.fieldPad()*2 + a.lineH()*0 // the chrome the rows sit in
		for w := float32(120); w <= 420; w += 0.5 {
			h := a.Measure(layout.Loose(w, 4000)).Y
			// The box may be laid out a fraction narrower than it was
			// offered, and that is where the rows have to fit.
			want := a.rowsFor(floorPx(w))
			if want > a.MaxRows {
				want = a.MaxRows
			}
			if got := (h - pad) / a.lineH(); got < float32(want)-0.01 {
				t.Fatalf("scale %g, offered %g: %g rows of room for the %d the text needs at %g",
					scale, w, got, want, floorPx(w))
			}
		}
	}
}
