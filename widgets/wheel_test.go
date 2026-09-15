package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A touchpad's pixels scroll as they are; a wheel notch scrolls three
// lines. Small touchpad deltas used to count as notches and jump.
func TestWheelAndTouchpad(t *testing.T) {
	if got := wheelDelta(3, 20, true); got != 3 {
		t.Fatalf("3 touchpad pixels scroll %v", got)
	}
	if got := wheelDelta(1, 20, false); got != 60 {
		t.Fatalf("a notch scrolls %v, want three 20px lines", got)
	}
	if got := wheelDelta(0.25, 20, false); got != 15 {
		t.Fatalf("a quarter notch (hi-res wheel) scrolls %v", got)
	}
	l := NewListView(100, func(i int) string { return "row" }, nil)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	l.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 2), Precise: true})
	if l.OffsetY != 2 {
		t.Fatalf("two touchpad pixels moved the list %v", l.OffsetY)
	}
}
