package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The tracker keeps the gesture contract whatever the window system
// sends: begin, updates, one end; Scale cumulative from 1; nothing without
// a begin; a new begin cancels an open gesture; an end of another kind is
// not this gesture's end.
func TestGestureTrackerContract(t *testing.T) {
	var g gestureTracker
	at := paintengine2d.Pt(10, 20)
	if evs := g.update(GesturePinch, at, paintengine2d.Pt(1, 1), 1.5, 3, 0); len(evs) != 0 {
		t.Fatalf("an update with no begin was delivered: %+v", evs)
	}
	if evs := g.finish(GesturePinch, false, at, 0); len(evs) != 0 {
		t.Fatalf("an end with no begin was delivered: %+v", evs)
	}

	evs := g.begin(GesturePinch, 2, at, ModCtrl)
	if len(evs) != 1 || evs[0].Kind != EventGesture || evs[0].Gesture != GesturePinch || evs[0].Phase != GestureBegin ||
		evs[0].Fingers != 2 || evs[0].Scale != 1 || evs[0].Pos != at || evs[0].Mods != ModCtrl {
		t.Fatalf("begin: %+v", evs)
	}
	evs = g.update(GesturePinch, at, paintengine2d.Pt(2, -1), 1.25, 5, 0)
	if len(evs) != 1 || evs[0].Phase != GestureUpdate || evs[0].Scale != 1.25 || evs[0].Rotation != 5 ||
		evs[0].Delta != paintengine2d.Pt(2, -1) || evs[0].Fingers != 2 {
		t.Fatalf("update: %+v", evs)
	}
	// A swipe's update is not the pinch's.
	if evs := g.update(GestureSwipe, at, paintengine2d.Pt(9, 9), 0, 0, 0); len(evs) != 0 {
		t.Fatalf("another kind's update went to the pinch: %+v", evs)
	}
	// Scale 0 (none given) keeps the last.
	evs = g.update(GesturePinch, at, paintengine2d.Pt(0, 0), 0, 0, 0)
	if evs[0].Scale != 1.25 {
		t.Fatalf("scale %v, want the last 1.25", evs[0].Scale)
	}
	if evs := g.finish(GestureSwipe, false, at, 0); len(evs) != 0 {
		t.Fatalf("a swipe's end ended the pinch: %+v", evs)
	}
	// Wayland's end carries no scale: the last one is kept.
	evs = g.finish(GesturePinch, false, at, 0)
	if len(evs) != 1 || evs[0].Phase != GestureEnd || evs[0].Scale != 1.25 {
		t.Fatalf("end: %+v", evs)
	}
	if g.open() {
		t.Fatal("still open after its end")
	}

	// A begin while a gesture is open cancels that one first.
	g.begin(GestureSwipe, 3, at, 0)
	evs = g.begin(GesturePinch, 2, at, 0)
	if len(evs) != 2 || evs[0].Gesture != GestureSwipe || evs[0].Phase != GestureCancel ||
		evs[1].Gesture != GesturePinch || evs[1].Phase != GestureBegin {
		t.Fatalf("begin over an open gesture: %+v", evs)
	}
	// Swipes carry no rotation and keep no scale.
	g.begin(GestureSwipe, 3, at, 0)
	evs = g.update(GestureSwipe, at, paintengine2d.Pt(4, 0), 2, 30, 0)
	if evs[0].Rotation != 0 || evs[0].Scale != 1 {
		t.Fatalf("a swipe took a pinch's values: %+v", evs)
	}
	evs = g.cancel(at, 0)
	if len(evs) != 1 || evs[0].Phase != GestureCancel || evs[0].Gesture != GestureSwipe {
		t.Fatalf("cancel: %+v", evs)
	}
	if evs := g.cancel(at, 0); len(evs) != 0 {
		t.Fatalf("cancelling nothing said something: %+v", evs)
	}
	// Hold: begin and end, cancelled when the fingers move.
	g.begin(GestureHold, 2, at, 0)
	evs = g.finish(GestureHold, true, at, 0)
	if len(evs) != 1 || evs[0].Gesture != GestureHold || evs[0].Phase != GestureCancel {
		t.Fatalf("hold: %+v", evs)
	}
}

// A popup's gesture reaches its root window like its other pointer input,
// moved into the root's coordinates.
func TestGestureFromPopupGoesToRoot(t *testing.T) {
	if !popupInput(EventGesture) {
		t.Fatal("a gesture over a popup is not input for its window")
	}
	ev := popupEvent(Event{Kind: EventGesture, Pos: paintengine2d.Pt(5, 6)}, paintengine2d.Pt(100, 200))
	if ev.Pos != paintengine2d.Pt(105, 206) {
		t.Fatalf("pos %v", ev.Pos)
	}
}
