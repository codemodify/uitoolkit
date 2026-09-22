package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// GestureEvent is a touchpad gesture in the receiver's local coordinates.
// Kind and Phase are the platform's ([platform.GestureKind],
// [platform.GesturePhase]); Pos is where the pointer is.
type GestureEvent struct {
	Kind  platform.GestureKind
	Phase platform.GesturePhase
	Pos   paintengine2d.Point
	// Fingers is how many fingers make the gesture.
	Fingers int
	// Delta is how far the fingers (a pinch: the point between them)
	// moved since the last event, in device pixels.
	Delta paintengine2d.Point
	// Scale is a pinch's spread relative to its begin: 1 at the begin,
	// 2 when the fingers are twice as far apart. A zoom that started at z
	// is z·Scale now.
	Scale float32
	// Rotation is the degrees a pinch turned since the last event,
	// clockwise.
	Rotation float32
	Mods     platform.Modifiers
}

// GestureTarget is a component that takes touchpad gestures — an image
// view zooming on a pinch, a pager turning on a swipe.
//
// The contract, which is Qt's for QGestureEvent and GTK's for its
// gesture controllers: the begin goes to the component under the pointer,
// then to its ancestors, until one returns true. That one is the gesture's
// for the rest of it — every update, and the end or the cancel, go to it
// alone and wherever the pointer is by then, and what it returns for them
// is ignored. A gesture nobody takes at its begin is not offered again:
// its updates go nowhere. Disabled and hidden components are skipped.
//
// A horizontal swipe nobody takes still means something: when it ends
// the window hands it to the nearest [HistoryNavigator].
type GestureTarget interface {
	Gesture(e GestureEvent) bool
}

// HistoryNavigator is a component that goes back and forward through a
// history — a file manager's folders, a browser's pages, a help viewer's
// topics. The window calls NavigateHistory on the one under the pointer or
// its nearest ancestor that is one, for the mouse's back and forward
// buttons and for a horizontal touchpad swipe no [GestureTarget] took
// (fingers moving right go back, as a page slides back into view). It
// reports whether it moved; the walk up stops at the first that did.
//
// Keys are the component's own business: Alt+Left and Alt+Right are the
// convention on both desktops.
type HistoryNavigator interface {
	NavigateHistory(forward bool) bool
}
