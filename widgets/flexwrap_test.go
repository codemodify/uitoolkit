package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A wrapping label that flexes in a row is measured before the row knows
// how wide it will be: it reports the one line its text fits on, is then
// arranged narrower, and wraps. Without a second measure pass the row is
// one line tall and the label draws its other lines outside its box.
func TestWrappingLabelInARowGetsItsLines(t *testing.T) {
	text := "Ada Lovelace invited you to a meeting about the analytical engine on Tuesday"
	lbl := NewLabel(text)
	lbl.Wrap = true
	row := NewRow(lbl, NewButton("Accept", nil), NewButton("Decline", nil)).WithGap(8)
	row.AddFlex(lbl, 1)
	row.SetLook(style.DarkLook())
	row.SetHost(&host{})

	const w = 380
	got := row.Measure(layout.Loose(w, 600))
	row.Arrange(paintengine2d.XYWH(0, 0, w, got.Y))

	// The label folded: its box is at least two lines.
	line := lbl.Look().Font().Height()
	lines := int((lbl.Bounds().Dy() - 2) / line)
	if lines < 2 {
		t.Fatalf("the label got %v px, about %d line(s) of %v", lbl.Bounds().Dy(), lines, line)
	}
	// And the row is tall enough to hold it: the label is inside.
	if lbl.Bounds().Max.Y > got.Y {
		t.Fatalf("the label ends at %v, past the row's %v", lbl.Bounds().Max.Y, got.Y)
	}
	// The height it reports is the height it needs at that width, not
	// one line's: measuring the label alone at its final width agrees.
	alone := lbl.Measure(layout.Constraints{MaxW: lbl.Bounds().Dx(), MaxH: -1})
	if lbl.Bounds().Dy() < alone.Y {
		t.Fatalf("the label was given %v, wants %v at %v wide", lbl.Bounds().Dy(), alone.Y, lbl.Bounds().Dx())
	}
}

// A row of things that do not care about their width is unchanged: the
// second pass only ever grows a child, so nothing that measured happily
// the first time moves.
func TestSecondPassLeavesOrdinaryRowsAlone(t *testing.T) {
	a, b := newSized(60, 20), newSized(40, 30)
	row := NewRow(a, b).WithGap(10)
	row.SetLook(style.DarkLook())
	row.SetHost(&host{})
	row.AddFlex(a, 1)
	got := row.Measure(layout.Loose(300, 200))
	if got.Y != 30 {
		t.Fatalf("row height %v, want the tallest child's 30", got.Y)
	}
	row.Arrange(paintengine2d.XYWH(0, 0, 300, got.Y))
	if w := a.Bounds().Dx(); w != 300-10-40 {
		t.Fatalf("the flex child got %v", w)
	}
	// A row stretches its children across by default, so the short one
	// is the row's height — that is the existing contract, and the
	// second pass does not change it.
	if h := a.Bounds().Dy(); h != 30 {
		t.Fatalf("the flex child is %v tall in a %v row", h, got.Y)
	}
	// With AlignStart it keeps its own height, and still does.
	row.Spec.Align = layout.AlignStart
	row.Arrange(paintengine2d.XYWH(0, 0, 300, got.Y))
	if h := a.Bounds().Dy(); h != 20 {
		t.Fatalf("aligned to the start, the flex child is %v tall", h)
	}
}
