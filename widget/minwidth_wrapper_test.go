package widget

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// wideChild measures small when unbounded but knows it needs more.
type wideChild struct{ Base }

func (w *wideChild) MinWidth() float32 { return 848 }
func (w *wideChild) Measure(layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(320, 200)
}

// passThrough is the commonest thing an application writes: a wrapper that
// holds one thing and gives it everything it has. It implements no
// MinWidther, so it is probed.
type passThrough struct{ Base }

// A probe reads only the component's own measurements, so a wrapper round
// something that measures small answers small — whatever that thing says
// it needs. A splitter measures a fixed 320x200 unbounded, so a wrapper
// round one answered 320 against the 848 its panes needed, and a window
// sized from that let itself shrink to a third of what its content could
// live with.
func TestAWrapperIsNeverNarrowerThanWhatItHolds(t *testing.T) {
	child := &wideChild{}
	child.Init(child)
	w := &passThrough{}
	w.Init(w)
	w.Add(child)

	if got := MinWidthOf(child); got != 848 {
		t.Fatalf("the child answers %g for itself, want 848", got)
	}
	if got := MinWidthOf(w); got < 848 {
		t.Errorf("the wrapper answers %g, narrower than the %g it holds", got, float32(848))
	}
}
