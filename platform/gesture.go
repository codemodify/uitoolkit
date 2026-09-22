package platform

import "github.com/codemodify/paintengine2d"

// GestureKind is the touchpad gesture an [EventGesture] belongs to.
//
// Only a touchpad makes these: the fingers are counted and tracked by the
// input stack (libinput), and the window hears the gesture it recognised,
// not the fingers. Two fingers moving together are not a gesture at all —
// they scroll, and arrive as [EventScroll] with ScrollPrecise.
type GestureKind uint8

const (
	GestureNone GestureKind = iota
	// GesturePinch: two or more fingers moving apart or together, and
	// turning. Scale and Rotation say by how much; Delta is how far the
	// point between the fingers moved.
	GesturePinch
	// GestureSwipe: three or more fingers moving together. Delta is how
	// far they moved.
	GestureSwipe
	// GestureHold: fingers put down on the pad and kept still. It begins
	// when they touch and ends when they lift (GestureEnd) or start to
	// move (GestureCancel — the movement then turns into a scroll, a
	// swipe or a pinch). Wayland only; X11 has no hold.
	GestureHold
)

func (k GestureKind) String() string {
	switch k {
	case GesturePinch:
		return "pinch"
	case GestureSwipe:
		return "swipe"
	case GestureHold:
		return "hold"
	}
	return "none"
}

// GesturePhase is where in its life a gesture is: every gesture is one
// GestureBegin, any number of GestureUpdate, then exactly one GestureEnd
// or GestureCancel. Gestures do not overlap: a second one begins only
// after the first has ended.
type GesturePhase uint8

const (
	GestureBegin GesturePhase = iota
	GestureUpdate
	// GestureEnd: the fingers lifted and the gesture is complete; act on it.
	GestureEnd
	// GestureCancel: the gesture was called off — the desktop took it
	// over, a finger was added or lifted, the window lost the pointer. Undo
	// whatever the updates did, or leave it, but do not act as on an end.
	GestureCancel
)

func (p GesturePhase) String() string {
	switch p {
	case GestureBegin:
		return "begin"
	case GestureUpdate:
		return "update"
	case GestureEnd:
		return "end"
	case GestureCancel:
		return "cancel"
	}
	return "?"
}

// gestureTracker turns a backend's gesture callbacks into [EventGesture]
// events that keep the contract above whatever the window system sends: an
// update or end with no begin before it is dropped, a begin while another
// gesture is open cancels that one first, Scale is cumulative from 1 at the
// begin (Wayland and XInput both say so, but an end carries none on
// Wayland), and an end that belongs to another kind of gesture than the
// open one is not the open one's end.
//
// Positions and deltas are the backend's to convert: it hands the tracker
// device pixels.
type gestureTracker struct {
	kind    GestureKind
	fingers int
	scale   float32
}

// open reports whether a gesture is in progress.
func (g *gestureTracker) open() bool { return g.kind != GestureNone }

// begin starts a gesture; a gesture still open is cancelled first.
func (g *gestureTracker) begin(kind GestureKind, fingers int, pos paintengine2d.Point, mods Modifiers) []Event {
	var out []Event
	if g.open() {
		out = append(out, g.finish(g.kind, true, pos, mods)...)
	}
	if kind == GestureNone {
		return out
	}
	g.kind, g.fingers, g.scale = kind, fingers, 1
	return append(out, Event{
		Kind: EventGesture, Gesture: kind, Phase: GestureBegin, Fingers: fingers,
		Pos: pos, Scale: 1, Mods: mods,
	})
}

// update moves the open gesture on: delta is how far it moved since the
// last event, scale the pinch's spread relative to its begin (0 keeps the
// last), rotation the degrees it turned since the last event.
func (g *gestureTracker) update(kind GestureKind, pos, delta paintengine2d.Point, scale, rotation float32, mods Modifiers) []Event {
	if !g.open() || kind != g.kind {
		return nil
	}
	if kind == GesturePinch && scale > 0 {
		g.scale = scale
	}
	ev := Event{
		Kind: EventGesture, Gesture: kind, Phase: GestureUpdate, Fingers: g.fingers,
		Pos: pos, Delta: delta, Scale: g.scale, Mods: mods,
	}
	if kind == GesturePinch {
		ev.Rotation = rotation
	}
	return []Event{ev}
}

// finish ends the open gesture of this kind, or cancels it.
func (g *gestureTracker) finish(kind GestureKind, cancelled bool, pos paintengine2d.Point, mods Modifiers) []Event {
	if !g.open() || kind != g.kind {
		return nil
	}
	phase := GestureEnd
	if cancelled {
		phase = GestureCancel
	}
	ev := Event{
		Kind: EventGesture, Gesture: kind, Phase: phase, Fingers: g.fingers,
		Pos: pos, Scale: g.scale, Mods: mods,
	}
	g.kind, g.fingers, g.scale = GestureNone, 0, 0
	return []Event{ev}
}

// cancel calls off whatever gesture is open (the pointer left the window).
func (g *gestureTracker) cancel(pos paintengine2d.Point, mods Modifiers) []Event {
	return g.finish(g.kind, true, pos, mods)
}
