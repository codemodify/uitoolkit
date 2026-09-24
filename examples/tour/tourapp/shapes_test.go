package tourapp

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// The tour's silhouette stamp zooms and turns with a pinch, within bounds;
// a cancelled pinch puts the zoom back.
func TestShapeStampPinches(t *testing.T) {
	s := newShapePreview(&shapesPage{})
	ev := func(ph platform.GesturePhase, scale, rot float32) widget.GestureEvent {
		return widget.GestureEvent{Kind: platform.GesturePinch, Phase: ph, Fingers: 2, Scale: scale, Rotation: rot}
	}
	if !s.Gesture(ev(platform.GestureBegin, 1, 0)) {
		t.Fatal("the stamp refused a pinch")
	}
	s.Gesture(ev(platform.GestureUpdate, 2, 10))
	s.Gesture(ev(platform.GestureUpdate, 2.5, 5))
	if s.zoom != 2.5 || s.turn != 15 {
		t.Fatalf("zoom %v turn %v", s.zoom, s.turn)
	}
	s.Gesture(ev(platform.GestureUpdate, 50, 0))
	if s.zoom != 3 {
		t.Fatalf("zoom %v, want the cap", s.zoom)
	}
	s.Gesture(ev(platform.GestureCancel, 50, 0))
	if s.zoom != 1 {
		t.Fatalf("a cancelled pinch left zoom %v", s.zoom)
	}
	swipe := ev(platform.GestureBegin, 1, 0)
	swipe.Kind = platform.GestureSwipe
	if s.Gesture(swipe) {
		t.Fatal("the stamp took a swipe")
	}
}
