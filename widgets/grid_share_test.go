package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// wrapper measures one long line when it is offered no width, and wraps
// to whatever it is given — the manager's secret text, which draws a long
// key or a recovery share from its bytes and so cannot be a Label.
type wrapper struct {
	widget.Base
	n float32 // characters
}

func (w *wrapper) MinWidth() float32 { return 40 }
func (w *wrapper) Measure(c layout.Constraints) paintengine2d.Point {
	line := w.n * 8
	if !c.HasMaxW() || c.MaxW <= 0 {
		return paintengine2d.Pt(line, 16)
	}
	rows := 1 + int(line/c.MaxW)
	return c.Constrain(paintengine2d.Pt(c.MaxW, float32(rows)*16))
}

// A Share column is measured at the share it is given, not unbounded, so
// a cell that wraps does not drag the grid out to the width of its text
// on one line.
//
// It cannot be done by bounding the ordinary measuring pass: Measure ends
// in Constrain, so a bounded answer cannot be told from a clamped one,
// and a control that genuinely cannot shrink would report the width it
// was squeezed to and lose its floor. That is what this mode is for.
func TestAShareColumnIsMeasuredAtItsShare(t *testing.T) {
	look := style.DarkLook()
	mk := func(col Track) *Grid {
		g := NewGrid()
		g.Cols = []Track{Auto(), col}
		g.ColGap, g.RowGap = 8, 8
		lab := NewLabel("Key:")
		w := &wrapper{n: 200} // 1600 device pixels on one line
		w.Init(w)
		g.Place(lab, 0, 0)
		g.Place(w, 0, 1)
		g.SetLook(look)
		g.SetHost(&host{})
		lab.SetLook(look)
		w.SetLook(look)
		return g
	}

	// Unbounded, a Flex column takes the whole line as what it wants.
	flex := mk(Flex(1)).Measure(layout.Unbounded())
	share := mk(Share(1)).Measure(layout.Unbounded())
	if share.X >= flex.X {
		t.Errorf("a share column asks for %g, no less than the flex column's %g", share.X, flex.X)
	}

	// Given a real width, the cell is laid out inside the column rather
	// than running off its edge.
	g := mk(Share(1))
	const W = 400
	g.Arrange(paintengine2d.XYWH(0, 0, W, 200))
	var cell widget.Component
	widget.Walk(g, func(c widget.Component) {
		if _, ok := c.(*wrapper); ok {
			cell = c
		}
	})
	if cell == nil {
		t.Fatal("no cell")
	}
	b := cell.Bounds()
	if b.Max.X > W+0.5 {
		t.Errorf("the cell ends at %g, past the %d-pixel grid", b.Max.X, W)
	}
	if b.Dx() <= 0 {
		t.Error("the cell got no width")
	}
}
