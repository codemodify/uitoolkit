package platform

import "github.com/codemodify/paintengine2d"

// DragAction is what a drag's target does with the data. It is a bit set
// so a source can offer several at once and a target answer with the one
// it picked; a single action is a one-bit value. Wayland's
// wl_data_device_manager.dnd_action is the same idea, and XDND names the
// same three in XdndActionCopy / Move / Link.
type DragAction uint32

const (
	// DragNone is "nothing will be done with it": a drag no target takes,
	// or one that was cancelled.
	DragNone DragAction = 0
	DragCopy DragAction = 1 << 0
	DragMove DragAction = 1 << 1
	DragLink DragAction = 1 << 2
)

// dragActionOrder is the preference between actions both sides allow:
// copying is the safe default, a move destroys the original, a link is
// the most specialised.
var dragActionOrder = [...]DragAction{DragCopy, DragMove, DragLink}

// Has reports whether a is (or contains) b.
func (a DragAction) Has(b DragAction) bool { return b != 0 && a&b == b }

// One is the single action a carries: the first of copy, move and link in
// it. DragNone when it is empty.
func (a DragAction) One() DragAction {
	for _, one := range dragActionOrder {
		if a&one != 0 {
			return one
		}
	}
	return DragNone
}

func (a DragAction) String() string {
	switch a.One() {
	case DragCopy:
		return "copy"
	case DragMove:
		return "move"
	case DragLink:
		return "link"
	}
	return "none"
}

// NegotiateDragAction is the one action a drop performs: the source's
// preferred action when the target allows it, else the first of copy,
// move and link that both the source's offered set and the target's
// allowed set carry. DragNone when they have none in common — the drag
// is refused.
func NegotiateDragAction(offered, preferred, allowed DragAction) DragAction {
	both := offered & allowed
	if both == 0 {
		return DragNone
	}
	if p := preferred.One(); p != DragNone && both.Has(p) {
		return p
	}
	return both.One()
}

// DragPayload is a drag this application starts: what it offers, what it
// looks like under the pointer, and what may be done with it. Data is
// called on the event-loop goroutine when a target asks for a type, so it
// must not block.
type DragPayload struct {
	// Types are the types offered, best first.
	Types []string
	// Data reads one of Types. Reporting false refuses that type.
	Data func(mime string) ([]byte, bool)
	// Icon follows the pointer; Hotspot is the point in it that sits
	// under the pointer. A nil Icon drags without a picture.
	Icon    *paintengine2d.Image
	Hotspot paintengine2d.Point
	// Actions are what the source allows (DragCopy when zero); Preferred
	// is the one it would rather the target took.
	Actions   DragAction
	Preferred DragAction
}

// DragSurface is implemented by surfaces that can start a drag of their
// own — the source half of drag and drop. StartDrag begins one from the
// last button press (the press serial on Wayland, the press grab on X11)
// and reports whether the drag started; the drag ends with an
// [EventDragEnd] carrying the action the target performed. CancelDrag
// stops one early (Escape).
type DragSurface interface {
	StartDrag(p DragPayload) bool
	CancelDrag()
	// Dragging reports whether a drag started here is still running.
	Dragging() bool
}

// DropNegotiator is implemented by surfaces whose drag protocol asks the
// target, while a drag is over it, whether it would take a drop and what
// it would do with it: XDND answers every XdndPosition with an
// XdndStatus, Wayland calls wl_data_offer.accept and set_actions. The app
// package calls this for every [EventDragMotion]; mime "" (or DragNone
// for action) refuses, which is what makes the source show a "no drop"
// cursor.
//
// allowed is every action the target would take and action the one it
// would take now. Both are needed: Wayland's compositor picks the action
// from the user's modifiers out of what the target allows, so a target
// that named only the one action it had settled on would pin the drag to
// it and Shift for a move could never do anything.
type DropNegotiator interface {
	AcceptDrag(mime string, allowed, action DragAction)
}

// ModifierDragAction is the action the user is asking for with the keys
// they hold, narrowed to what the drag allows: Shift moves, Ctrl copies,
// the two together link — the convention every X11 desktop shares. A
// Wayland compositor does this itself and reports the result; on X11 it
// is the drag's source that has to.
//
// fallback is what the source would rather have when no modifier says
// otherwise; with none of it allowed, the first action the drag allows.
func ModifierDragAction(mods Modifiers, allows, fallback DragAction) DragAction {
	var want DragAction
	switch {
	case mods.Ctrl() && mods.Shift():
		want = DragLink
	case mods.Shift():
		want = DragMove
	case mods.Ctrl():
		want = DragCopy
	}
	if want != DragNone && allows.Has(want) {
		return want
	}
	if one := fallback.One(); one != DragNone && allows.Has(one) {
		return one
	}
	return allows.One()
}
