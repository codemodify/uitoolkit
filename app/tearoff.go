package app

import (
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Tear-off: a drag that carries a window of its own. A tab dragged out of
// its strip, a dock panel dragged out of its host — the drag offers the
// thing itself (a private type, so another window of this application
// knows its own), and a window rides along under the pointer so the user
// sees what they are carrying and can put it down anywhere.
//
// Three ways it can end, and the window's fate differs in each:
//
//	merged    a target took the content away — another window's tab
//	          strip, the dock host — so the window it was carried in has
//	          nothing left in it
//	kept      the pointer was let go over nothing: the window stays
//	          where the desktop left it
//	cancelled Escape, or the drag never started: nothing happened and the
//	          source takes its content back
//
// What becomes of the window is the source's ([widget.TearOff.Done]): the
// one case where it must live on is a window the drag only moved, and
// only the source knows that it did.
//
// Who moves the window under the pointer is the desktop's: a Wayland
// compositor with xdg-toplevel-drag-v1 carries it, and on X11 the toolkit
// moves it itself, since X11 clients place their own windows. Without
// either the drag still runs — the window is simply made at the drop
// instead of at the start, wherever the compositor cares to put it, which
// is the fallback Chromium ships for the same reason.

// StartTearOff begins a drag that carries a window of its own. It
// implements [widget.TearOffHost], so a component starts one with
// widget.StartTearOff(c, d, t).
//
// d is the drag: what it offers, its picture, its payload — everything an
// ordinary drag carries, which is what lets an ordinary drop target take
// it. t is the window half (see [widget.TearOff]): the window is opened
// now where the desktop can carry it and at the drop where it cannot.
func (w *Window) StartTearOff(d *widget.Drag, t *widget.TearOff) bool {
	if t == nil || t.Open == nil {
		return w.StartDrag(d)
	}
	return w.startDrag(d, t)
}

// DragsWindows reports whether a drag from this window can carry a window
// under the pointer. It implements the capability [widget.DragsWindows]
// asks for, so a tear-off source knows whether its content leaves now or
// at the drop.
func (w *Window) DragsWindows() bool {
	return w != nil && !w.Closed() && platform.DragsToplevels(w.surf)
}

// Position is where the desktop put this window, in the same pixels its
// size is in, and whether it is known at all. A Wayland client is never
// told (and cannot ask), so it answers false there; X11 answers from the
// last ConfigureNotify and the offscreen backend from where a test put
// the window.
func (w *Window) Position() (x, y int, ok bool) {
	if w == nil || w.Closed() {
		return 0, 0, false
	}
	return platform.SurfacePosition(w.surf)
}

// Move asks the desktop to put this window at x, y, in the same root pixels
// [Window.Position] answers in, and reports whether the desktop let it.
//
// It is [Window.Position]'s other half and it has the same shape: X11
// clients place their own windows, so it works there and in tests; a
// Wayland toplevel has no position and cannot be given one, so it answers
// false and does nothing. An app that keeps two windows stuck together —
// a player with its playlist under it — asks [Window.CanMove] first and
// says so in its own interface when the answer is no, rather than
// pretending the windows moved.
func (w *Window) Move(x, y int) bool {
	if !w.CanMove() {
		return false
	}
	platform.MoveSurface(w.surf, x, y)
	return true
}

// CanMove reports whether this desktop lets the client place the window.
func (w *Window) CanMove() bool {
	return w != nil && !w.Closed() && platform.SurfaceMoves(w.surf)
}

// carry opens the window a tear-off goes into and hands it to the drag,
// where the desktop can carry one. Where it cannot, nothing happens here:
// the window is made at the drop instead, and the source keeps showing
// what is being dragged until then.
func (r *dragRun) carry() {
	if r == nil || r.tear == nil || r.torn != nil || r.ended {
		return
	}
	if r.local || r.win == nil || !r.win.DragsWindows() {
		return
	}
	if !r.openTorn(r.tear) {
		return
	}
	surf := r.torn.Surface()
	if surf == nil {
		return
	}
	// The offset is where in the new window the pointer sits. A source
	// guesses it from the window the drag started in, which may be the
	// larger of the two, so it is kept inside the window it belongs to.
	w, h := surf.Size()
	off := r.tear.Offset
	platform.AttachToplevel(r.win.surf, surf, clampOffset(off.X, w), clampOffset(off.Y, h))
}

// clampOffset keeps a pointer offset inside a window of extent size.
func clampOffset(v float32, size int) int {
	if v < 0 || size < 1 {
		return 0
	}
	if int(v) > size-1 {
		return size - 1
	}
	return int(v)
}

// openTorn asks the tear-off for its window. It reports whether there is
// one: a source that changed its mind returns nil, and the drag runs on
// as a plain one.
func (r *dragRun) openTorn(t *widget.TearOff) bool {
	win := t.Open()
	if win == nil {
		return false
	}
	r.torn = win
	return true
}

// finishTear is what became of a tear-off once the drag is over. A target
// took the content away, so the window that carried it has nothing left
// in it; or the pointer was let go over nothing, so the window stays —
// and is opened now if the desktop could not carry it; or the drag was
// called off and nothing happened at all.
//
// Only a move takes the content: a target that copied it — a file manager
// taking the folder behind a tab — leaves the torn-off window standing,
// which is what a copy means.
func (r *dragRun) finishTear(action platform.DragAction) {
	if r == nil || r.tear == nil {
		return
	}
	t := r.tear
	r.tear = nil
	res := widget.TearCancelled
	switch {
	case action.Has(platform.DragMove):
		res = widget.TearMerged
	case r.dropped:
		res = widget.TearKept
		if r.torn == nil && !r.openTorn(t) {
			// Nothing took it and no window would open: the content has
			// nowhere to go, so it goes back where it came from.
			res = widget.TearCancelled
		}
	}
	win := r.torn
	r.torn = nil
	if t.Done != nil {
		t.Done(res, win)
	}
}
