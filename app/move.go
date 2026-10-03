package app

import "github.com/codemodify/uitoolkit/platform"

// OnMove is called when the desktop moves the window, with its new
// position in logical pixels.
//
// It is how an application that keeps other windows beside this one — a
// player with its equaliser and playlist snapped to it, anything built on
// [rack.Desk] — hears about a drag. Without it the only way to notice was
// to read [Window.Position] on a timer, which is a trade between a timer
// fast enough that the satellites look attached and one slow enough not
// to cost anything: at 70 ms the panels visibly trail the window.
//
// **Not every desktop sends it.** A Wayland toplevel has no position in
// that protocol — a client is never told where its windows are — so no
// callback arrives there, and [Window.SendsMoveEvents] says so in advance
// rather than leaving an application waiting for an event that cannot
// come. X11, Windows and macOS report it.
//
// It is one callback per window, like OnLockKeys: the window's own code.
func (w *Window) OnMove(fn func(x, y int)) {
	if w == nil {
		return
	}
	w.onMove = fn
}

// SendsMoveEvents reports whether this window's backend tells it when the
// desktop moves it (see [Window.OnMove]). False on Wayland.
func (w *Window) SendsMoveEvents() bool {
	return w != nil && w.surf != nil && platform.SendsMoveEvents(w.surf)
}

// noteMoved reports a move the backend saw.
func (w *Window) noteMoved(x, y int) {
	if w == nil || w.onMove == nil {
		return
	}
	w.onMove(x, y)
}
