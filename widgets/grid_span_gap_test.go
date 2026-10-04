package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A child spanning a hidden track stops at the grid's edge like any other.
//
// A track nothing visible is in takes no width and no gap: that is what lets a
// form of eight rows showing three not carry five gaps it has no use for, and
// what lets a flexible column reach the grid's own right edge. The span length
// did not follow the rule — one gap per boundary, hidden or not — so a label
// spanning three columns with the middle one hidden ran a gap past the edge,
// 408 in a 400-pixel grid, while the child beside it ended correctly at 400.
func TestASpanOverAHiddenTrackStopsAtTheEdge(t *testing.T) {
	g := NewGrid()
	g.RawSpacing = true
	g.Cols = []Track{Auto(), Auto(), Share(1)}
	g.ColGap = 8
	left, hidden, right := NewLabel("A"), NewLabel("B"), NewLabel("C")
	g.Place(left, 0, 0)
	g.Place(hidden, 0, 1)
	g.Place(right, 0, 2)
	hidden.SetVisible(false)
	span := NewLabel("Spanning row")
	g.PlaceSpan(span, 1, 0, 1, 3)
	g.SetLook(style.DarkLook())
	g.SetHost(&host{})
	const W = 400
	g.Arrange(paintengine2d.XYWH(0, 0, W, 80))

	re := widget.DeviceBounds(right).Max.X
	se := widget.DeviceBounds(span).Max.X
	if re != W {
		t.Errorf("the flexible column ends at %g, not the grid's %d", re, W)
	}
	if se != re {
		t.Errorf("the spanning child ends at %g and the column at %g: they should agree", se, re)
	}
}

// And with nothing hidden the span still carries its gaps, which is what makes
// it line up with the columns it spans: a fix that dropped them would pass the
// test above and shrink every spanning cell in every grid.
func TestASpanOverVisibleTracksKeepsItsGaps(t *testing.T) {
	g := NewGrid()
	g.RawSpacing = true
	g.Cols = []Track{Auto(), Auto(), Share(1)}
	g.ColGap = 8
	left, mid, right := NewLabel("A"), NewLabel("B"), NewLabel("C")
	g.Place(left, 0, 0)
	g.Place(mid, 0, 1)
	g.Place(right, 0, 2)
	span := NewLabel("Spanning row")
	g.PlaceSpan(span, 1, 0, 1, 3)
	g.SetLook(style.DarkLook())
	g.SetHost(&host{})
	const W = 400
	g.Arrange(paintengine2d.XYWH(0, 0, W, 80))

	re := widget.DeviceBounds(right).Max.X
	se := widget.DeviceBounds(span).Max.X
	if re != W {
		t.Errorf("the flexible column ends at %g, not the grid's %d", re, W)
	}
	if se != re {
		t.Errorf("the spanning child ends at %g and the column at %g: they should agree", se, re)
	}
}

// Two of three hidden: the span is one track wide and carries no gap at all.
func TestASpanOverOneRemainingTrackCarriesNoGap(t *testing.T) {
	g := NewGrid()
	g.RawSpacing = true
	g.Cols = []Track{Auto(), Auto(), Share(1)}
	g.ColGap = 8
	a, b, c := NewLabel("A"), NewLabel("B"), NewLabel("C")
	g.Place(a, 0, 0)
	g.Place(b, 0, 1)
	g.Place(c, 0, 2)
	a.SetVisible(false)
	b.SetVisible(false)
	span := NewLabel("Spanning row")
	g.PlaceSpan(span, 1, 0, 1, 3)
	g.SetLook(style.DarkLook())
	g.SetHost(&host{})
	const W = 400
	g.Arrange(paintengine2d.XYWH(0, 0, W, 80))

	if se := widget.DeviceBounds(span).Max.X; se != W {
		t.Errorf("the spanning child ends at %g, not the grid's %d", se, W)
	}
}
