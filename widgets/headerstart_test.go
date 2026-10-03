package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// The caption is laid out before the content, so anything that measured
// a divider and then told the header about it would always be one frame
// behind — on a drag, a frame the user sees. The splitter reports on the
// drag instead, and the header takes a plain number.
func TestSplitterReportsOnTheDragNotTheLayout(t *testing.T) {
	side, body := NewLabel("side"), NewLabel("body")
	sp := NewSplitter(SplitColumns, side, body)
	sp.Ratio = 0.3
	atScale(t, sp, 1)

	var gotW, gotR float32
	var calls int
	sp.OnRatioChanged = func(paneA, ratio float32) {
		gotW, gotR, calls = paneA, ratio, calls+1
	}
	sp.Measure(layout.Tight(600, 300))
	sp.Arrange(paintengine2d.XYWH(0, 0, 600, 300))

	// A layout on its own says nothing: there is nothing new to say.
	if calls != 0 {
		t.Errorf("laying out fired the watcher %d times", calls)
	}

	div := sp.divider()
	sp.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(div.Min.X+1, 20), Button: platform.ButtonLeft})
	sp.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(420, 20)})
	if calls == 0 {
		t.Fatal("dragging the divider said nothing")
	}
	if gotW <= 0 || gotR <= 0 {
		t.Errorf("reported paneA=%v ratio=%v", gotW, gotR)
	}
	// The width reported is the one the pane actually got.
	if got := sp.PaneA().Dx(); got != gotW {
		t.Errorf("reported %v but the pane is %v", gotW, got)
	}
}

// SetRatio reports too, so restoring a saved layout is not a silent
// change the chrome misses.
func TestSplitterSetRatioReports(t *testing.T) {
	sp := NewSplitter(SplitColumns, NewLabel("a"), NewLabel("b"))
	atScale(t, sp, 1)
	sp.Measure(layout.Tight(600, 300))
	sp.Arrange(paintengine2d.XYWH(0, 0, 600, 300))

	n := 0
	sp.OnRatioChanged = func(float32, float32) { n++ }
	sp.SetRatio(0.6)
	if n != 1 {
		t.Errorf("SetRatio reported %d times", n)
	}
}

// The header's own items begin after StartWidth, so a bar can line up
// with a pane below it — and never under the window controls.
func TestHeaderBarStartWidth(t *testing.T) {
	item := NewButton("Fetch", nil)
	h := NewHeaderBar([]widget.Component{item}, nil, nil)
	atScale(t, h, 1)
	h.Arrange(paintengine2d.XYWH(0, 0, 800, 36))
	at0 := h.row.Bounds().Min.X + item.Bounds().Min.X

	h.StartWidth = 240
	h.Arrange(paintengine2d.XYWH(0, 0, 800, 36))
	at240 := h.row.Bounds().Min.X + item.Bounds().Min.X

	if at240 <= at0 {
		t.Errorf("StartWidth moved the item from %v to %v", at0, at240)
	}
	if at240 < 240 {
		t.Errorf("the item begins at %v, before the reserved %v", at240, 240)
	}
}
