package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// Tearing off: a drag that takes something out of its window and into a
// window of its own — a tab dragged out of a strip, a dock panel dragged
// out of its host. It is an ordinary [Drag], carrying an ordinary offer,
// with a window riding along; that is what lets the same gesture drop the
// thing back into another window's strip, or on the desktop, or nowhere.
//
// Who moves the window is the desktop's business: a Wayland compositor
// with xdg-toplevel-drag-v1 carries it, an X11 client moves its own. On a
// compositor with neither, the drag still runs and the window is made at
// the drop instead, wherever the compositor puts it — which is why
// [DragsWindows] is worth asking before the source takes anything out of
// itself.

// TearOffWindow is the window a tear-off goes into: app.Window implements
// it as it stands. A source never makes one itself — [TearOff.Open] is
// asked for it at the moment the window is due.
type TearOffWindow interface {
	Host
	// Surface is the window's native surface: the drag attaches it, so
	// the desktop carries the window under the pointer.
	Surface() platform.Surface
	// Close destroys the window, which is what a source does with one
	// whose content has gone somewhere else ([TearOff.Done]).
	Close()
}

// TearResult is how a tear-off ended.
type TearResult uint8

const (
	// TearCancelled: the drag was called off (Escape, or it never
	// started). Nothing happened and the source takes its content back.
	TearCancelled TearResult = iota
	// TearKept: the pointer was let go where nothing took the drop. The
	// window stays where the desktop put it, holding the content.
	TearKept
	// TearMerged: a target took the content away (it performed a move) —
	// another window's tab strip, the dock host the panel came from.
	// Whoever took it holds it now, so the window that carried it has
	// nothing left in it.
	TearMerged
)

// TearOff is the window half of a drag (see the note at the top of this
// file). It goes to [StartTearOff] beside the [Drag] that carries the
// content itself.
type TearOff struct {
	// Open opens the window the content goes into, fills it and shows it.
	// It is called as the drag starts where the desktop carries a window
	// under the pointer, and at the drop where it does not — so a source
	// asks [DragsWindows] to know whether its content leaves now or then.
	// Returning nil gives the tear-off up: the drag runs on without a
	// window, and a drop that nothing takes ends in TearCancelled.
	Open func() TearOffWindow
	// Offset is where in that window the pointer sits, in the window's
	// own coordinates, so the window does not jump as it is picked up.
	Offset paintengine2d.Point
	// Done is how it ended, with the window it ended with (nil when none
	// was ever opened). Closing that window is the source's: a merged
	// tear-off's window has nothing left in it and a cancelled one should
	// never have existed, while a window the drag only moved — a floating
	// panel dragged back to its host — is the panel's own and stays.
	Done func(res TearResult, win TearOffWindow)
}

// TearOffHost is implemented by app.Window: start a drag that carries a
// window of its own.
type TearOffHost interface {
	StartTearOff(d *Drag, t *TearOff) bool
}

// StartTearOff begins a tear-off drag from c's window: d is what the drag
// offers — put a private type first, so another window of this
// application knows its own — and t the window it carries. It reports
// false when no drag could start (nothing to offer, or one is already
// running).
func StartTearOff(c Component, d *Drag, t *TearOff) bool {
	if c == nil || d == nil || t == nil {
		return false
	}
	if d.Source == nil {
		d.Source = c
	}
	h, ok := c.Host().(TearOffHost)
	if !ok {
		return false
	}
	return h.StartTearOff(d, t)
}

// DragsWindows reports whether a drag from c's window can carry a window
// under the pointer. Where it cannot — a Wayland compositor without
// xdg-toplevel-drag-v1 — a tear-off runs as a plain drag and its window
// appears at the drop, so the source keeps showing what it tore off until
// then (Chromium's fallback, for the same reason).
func DragsWindows(c Component) bool {
	if c == nil {
		return false
	}
	h, ok := c.Host().(interface{ DragsWindows() bool })
	return ok && h.DragsWindows()
}
