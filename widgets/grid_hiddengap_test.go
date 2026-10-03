package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A hidden track costs no gap anywhere: not in placement, and not in the
// space the flexible tracks have to share.
//
// v0.23.2 skipped a zero-width track's gap when placing the tracks but
// left the solver reserving it, so a flexible column came up one gap
// short and the grid did not reach its own right edge.
func TestAHiddenTrackCostsNoGapInTheSolverEither(t *testing.T) {
	look := style.DarkLook()
	g := NewGrid()
	g.Cols = []Track{Auto(), Auto(), Share(1)}
	g.ColGap, g.RowGap = 8, 8
	a, b, c := NewLabel("A"), NewLabel("B"), NewLabel("C")
	g.Place(a, 0, 0)
	g.Place(b, 0, 1)
	g.Place(c, 0, 2)
	g.SetLook(look)
	g.SetHost(&host{})
	for _, w := range []widget.Component{a, b, c} {
		w.SetLook(look)
	}

	const W = 400
	arrange := func() float32 {
		g.Arrange(paintengine2d.XYWH(0, 0, W, 40))
		return c.Bounds().Max.X
	}

	b.SetVisible(false)
	if got := arrange(); got != W {
		t.Errorf("with one of three columns hidden the last reaches %g, want %d", got, W)
	}
	b.SetVisible(true)
	if got := arrange(); got != W {
		t.Errorf("with all three showing the last reaches %g, want %d", got, W)
	}
}
