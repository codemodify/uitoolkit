package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// **A wheel notch ends a click gesture even when nothing scrolls.**
//
// Turning the wheel between two presses means the user did not mean them
// as a double click, and that is as true of a list short enough to fit
// as of one already at the end the wheel is pushing towards. Both were
// missed: the views reset the history only where the offset actually
// moved, so the two early returns — exactly the cases with nothing to
// scroll — kept it alive and the second press completed a double click
// across the notch. Reported against media-player-music as #29,
// "partial", with a probe of its own.
func TestAWheelEndsTheGestureEvenWithNothingToScroll(t *testing.T) {
	for _, tc := range []struct {
		name string
		dy   float32
	}{{"wheel up", 1}, {"wheel down", -1}} {
		t.Run(tc.name, func(t *testing.T) {
			// Three rows in a 300-high box: nothing to scroll either way.
			l, _, act := listRig(t, 3)
			if l.MaxOffset() > 0 {
				t.Fatalf("the list scrolls (max offset %v); this is about one that cannot", l.MaxOffset())
			}
			at := rowPoint(l, 1)
			l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
			if l.MouseWheel(widget.MouseEvent{Pos: at, Scroll: paintengine2d.Pt(0, tc.dy)}) {
				t.Fatal("a list with nothing to scroll took the notch")
			}
			l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
			if len(*act) != 0 {
				t.Errorf("two presses across a wheel notch activated %v", *act)
			}
		})
	}
}

// At the boundary of a list that *can* scroll: the notch moves nothing,
// and still ends the gesture.
func TestAWheelAtTheScrollBoundaryEndsTheGesture(t *testing.T) {
	l, _, act := listRig(t, 60)
	if l.MaxOffset() <= 0 {
		t.Fatal("this list does not scroll, so there is no boundary to sit at")
	}
	if l.OffsetY != 0 {
		t.Fatalf("a fresh list is scrolled to %v", l.OffsetY)
	}
	at := rowPoint(l, 1)
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	// Up, at the top: clamped to the same offset. A positive Scroll.Y
	// moves the view down, so up is the negative one.
	l.MouseWheel(widget.MouseEvent{Pos: at, Scroll: paintengine2d.Pt(0, -1)})
	if l.OffsetY != 0 {
		t.Fatalf("the wheel scrolled to %v at the top", l.OffsetY)
	}
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 0 {
		t.Errorf("two presses across a boundary notch activated %v", *act)
	}
}

// The control: two presses with no wheel between them still commit, or
// the tests above would pass on a list that never activates at all.
func TestTwoPressesWithNoWheelStillActivate(t *testing.T) {
	l, _, act := listRig(t, 3)
	at := rowPoint(l, 1)
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 1 {
		t.Errorf("a plain double click activated %v, want one row", *act)
	}
}
