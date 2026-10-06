package platform

import (
	"testing"
	"time"
)

// A secret's timeout is the one part of this package with a clock in it, and
// the clear it runs gives selections up. On Wayland the callbacks that touch
// the same state — a source cancelled, a selection served down a pipe — are
// dispatched from the event loop without the backend's lock held, so a clear
// on a timer goroutine of its own raced them: two destroys of one data source,
// a cache zeroed while it was being written, or a selection another program
// had just made unset by a clear that had not yet seen the cancel in hand.
//
// So the timeout hands its work to the loop where there is one.

func TestASecretsTimeoutIsRunOnTheLoopAndNotItsTimer(t *testing.T) {
	t.Cleanup(ClipboardClear)
	noNativeClipboard(t)
	posted := make(chan func(), 1)
	SetLoopPoster(func(fn func()) { posted <- fn })
	t.Cleanup(func() { SetLoopPoster(nil) })

	c := ClipboardSetSecret([]byte("hunter2"), 20*time.Millisecond)
	var fn func()
	select {
	case fn = <-posted:
	case <-time.After(3 * time.Second):
		t.Fatal("the timeout never reached the loop")
	}
	// The timer did not do the work itself: that is the whole point.
	if c.Cleared() {
		t.Error("the copy was cleared on the timer's goroutine before the loop ran it")
	}
	if !ClipboardHoldsSecret() {
		t.Error("the copy stopped being held before the loop ran the clear")
	}
	fn()
	if !c.Cleared() {
		t.Error("running the posted work did not clear the copy")
	}
}

// And with no loop to hand it to — a headless tool, a test, a program that has
// not started one — the clear still happens, on whatever goroutine is there.
// Queued work nothing will ever drain would be a passphrase left on the
// clipboard, which is worse than the race it avoids.
func TestWithNoLoopASecretsTimeoutStillClearsItself(t *testing.T) {
	SetLoopPoster(nil)
	t.Cleanup(ClipboardClear)
	// Stubbed for the same reason as above, and here it is the difference
	// between a real pass and one a cancelled selection handed over.
	noNativeClipboard(t)

	c := ClipboardSetSecret([]byte("hunter2"), 20*time.Millisecond)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.Cleared() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("with no loop the timeout never cleared the copy")
}
