package platform

import "sync"

// Work a background goroutine must not do itself.
//
// A backend's state belongs to the goroutine running the event loop. On
// Wayland that is not a convention but a fact of the protocol: the listener
// callbacks — a selection asked for, a data source cancelled — are dispatched
// from inside wl_display_dispatch_pending, which wlSurface.Poll and Wait call
// *before* they take wlMu, and which the clipboard service calls with wlMu
// released. Anything touching the same fields from another goroutine races
// them: two destroys of one data source, which can bring the program down; a
// cached value zeroed while it is being written to a reader's pipe; a clear
// that has not yet seen a cancel already in hand, unsetting a selection
// another program has just made.
//
// The secret clipboard is where that bites, because it is the one part of this
// package with a clock in it: the timeout fires on a timer goroutine thirty
// seconds after the copy, and the clear it runs gives selections up.
//
// This package cannot reach the application layer's own Post — the dependency
// runs the other way — so the application lends it one.
var (
	loopPostMu sync.RWMutex
	loopPost   func(func())
)

// SetLoopPoster installs the function that hands work to the UI goroutine, for
// the length of a run loop. nil removes it.
//
// It is installed while a loop is *running* and removed when it stops, which is
// the whole of the contract: posted work that nothing will ever drain is worse
// than work on the wrong goroutine, and a secret whose clear sat in a queue
// forever would be a passphrase left on the clipboard.
func SetLoopPoster(fn func(func())) {
	loopPostMu.Lock()
	loopPost = fn
	loopPostMu.Unlock()
}

// onLoop runs fn on the UI goroutine where one is running, and on the calling
// goroutine where none is — a headless tool, a test, a program that has not
// started its loop — because there is nothing to race there and the work still
// has to happen.
func onLoop(fn func()) {
	if fn == nil {
		return
	}
	loopPostMu.RLock()
	post := loopPost
	loopPostMu.RUnlock()
	if post == nil {
		fn()
		return
	}
	post(fn)
}
