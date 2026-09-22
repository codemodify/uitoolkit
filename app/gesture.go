package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// gestureState is the touchpad gesture in progress over a window: the
// component that took its begin, or — for a swipe nobody took — how far
// it has gone, so its end can go back or forward (widget.GestureTarget
// states the contract).
type gestureState struct {
	to    widget.Component
	swipe paintengine2d.Point
	at    paintengine2d.Point
	open  bool
}

// swipeNavigateDIP is how far, in logical pixels, three fingers have to
// travel sideways for a swipe nobody took to go back or forward.
const swipeNavigateDIP = 96

func gestureEvent(c widget.Component, ev platform.Event) widget.GestureEvent {
	return widget.GestureEvent{
		Kind: ev.Gesture, Phase: ev.Phase, Pos: local(c, ev.Pos), Fingers: ev.Fingers,
		Delta: ev.Delta, Scale: ev.Scale, Rotation: ev.Rotation, Mods: ev.Mods,
	}
}

// gesture delivers an EventGesture.
func (w *Window) gesture(ev platform.Event) {
	g := &w.gest
	if ev.Phase == platform.GestureBegin {
		w.dismissTooltip()
		*g = gestureState{open: true, at: ev.Pos}
		for c := w.hit(ev.Pos); c != nil; c = c.Parent() {
			if !c.Enabled() || !c.Visible() {
				continue
			}
			if t, ok := c.(widget.GestureTarget); ok && t.Gesture(gestureEvent(c, ev)) {
				g.to = c
				return
			}
		}
		return
	}
	if !g.open {
		return
	}
	done := ev.Phase == platform.GestureEnd || ev.Phase == platform.GestureCancel
	if c := g.to; c != nil {
		if done {
			*g = gestureState{}
		}
		// A component taken out of the window while the fingers were
		// down hears no more of it.
		if t, ok := c.(widget.GestureTarget); ok && w.holds(c) {
			t.Gesture(gestureEvent(c, ev))
		}
		return
	}
	if ev.Gesture != platform.GestureSwipe {
		if done {
			*g = gestureState{}
		}
		return
	}
	g.swipe = g.swipe.Add(ev.Delta)
	if !done {
		return
	}
	dx, dy, at := g.swipe.X, g.swipe.Y, g.at
	*g = gestureState{}
	if ev.Phase != platform.GestureEnd {
		return
	}
	sc := w.scale
	if sc <= 0 {
		sc = 1
	}
	need := float32(swipeNavigateDIP) * sc
	if abs32(dx) < need || abs32(dx) < 2*abs32(dy) {
		return
	}
	// Fingers moving right bring the previous page back.
	w.navigateHistory(at, dx < 0)
}

// holds reports whether c is still in one of the window's trees.
func (w *Window) holds(c widget.Component) bool {
	top := c
	for p := c.Parent(); p != nil; p = p.Parent() {
		top = p
	}
	return top != nil && (top == w.root || top == w.overlay || top == w.popup || top == w.caption)
}

// navigateHistory offers back (forward false) or forward to the component
// under p and its ancestors (widget.HistoryNavigator).
func (w *Window) navigateHistory(p paintengine2d.Point, forward bool) bool {
	for c := w.hit(p); c != nil; c = c.Parent() {
		if !c.Enabled() || !c.Visible() {
			continue
		}
		if n, ok := c.(widget.HistoryNavigator); ok && n.NavigateHistory(forward) {
			return true
		}
	}
	return false
}

// thumbButton handles a press or release of the mouse's back or forward
// button: it is a command to the window rather than a click on what is
// under the pointer, so nothing is pressed, focused or dragged by it and
// an open menu stays open, as in Qt and GTK apps. The press navigates.
func (w *Window) thumbButton(ev platform.Event) bool {
	if ev.Button != platform.ButtonBack && ev.Button != platform.ButtonForward {
		return false
	}
	if ev.Kind == platform.EventMouseDown {
		w.navigateHistory(ev.Pos, ev.Button == platform.ButtonForward)
	}
	return true
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
