package platform

import (
	"testing"
	"time"
)

// refuseNativeCopy runs fn with the platform's secret copy forced to fail, and
// returns what fn returned.
//
// A refusal cannot be arranged from outside: it is another program holding the
// Windows clipboard open for a quarter of a second, or a pasteboard write the
// window server declines. The seam is swapped for the length of the call
// instead, which is enough to pin what the *portable* half does with a
// refusal — the part that decides whether the program is told it holds a
// secret.
func refuseNativeCopy(t *testing.T, fn func() *SecretClip) *SecretClip {
	t.Helper()
	was := nativeSetSecret
	nativeSetSecret = func([]byte) bool { return false }
	t.Cleanup(func() { nativeSetSecret = was })
	return fn()
}

// noNativeClipboard stands in for the platform's secret copy with one that
// always succeeds, so no selection is actually taken.
//
// For the tests about the *portable* half. A live compositor gives the
// selection only to a client with a window and the focus, and a `go test`
// process has neither: the source it has just made is cancelled a few
// milliseconds later, which ends the held copy and stops its timer. A test
// about what the timer does then measures the compositor instead, and would
// pass or fail by which pass of tools/test-display.sh it ran in.
func noNativeClipboard(t *testing.T) {
	t.Helper()
	was := nativeSetSecret
	nativeSetSecret = func([]byte) bool { return true }
	t.Cleanup(func() { nativeSetSecret = was })
}

// **A refused copy does not leave the one before it on the clipboard.**
//
// The copy being replaced was let go before the new one was tried, so
// where another program held the clipboard open for every try — and
// nothing on it changed — the previous secret stayed there with its
// timer stopped and nobody holding it: not cleared at its time, nor at
// the vault's lock, nor at the quit. secretvault reported it as item
// 86, after its own copies on Windows started going through the
// toolkit.
func TestARefusedCopyClearsTheOneItWasReplacing(t *testing.T) {
	noNativeClipboard(t)
	first := ClipboardSetSecret([]byte("passphrase"), time.Minute)
	if first.Cleared() {
		t.Fatal("the first copy was never made")
	}

	cleared := make(chan struct{}, 1)
	first.OnCleared(func() { cleared <- struct{}{} })

	second := refuseNativeCopy(t, func() *SecretClip {
		return ClipboardSetSecret([]byte("another"), time.Minute)
	})
	if !second.Cleared() {
		t.Error("a refused copy was not ended")
	}
	if !first.Cleared() {
		t.Error("the copy the refused one was replacing is still held: its passphrase is on the clipboard with nothing left to clear it")
	}
	select {
	case <-cleared:
	case <-time.After(2 * time.Second):
		t.Error("the application was never told the first copy had gone")
	}
	if ClipboardHoldsSecret() {
		t.Error("a secret is still held after a refused copy")
	}
}

// And the ordinary way round: a copy that is made takes the one before
// it off the clipboard, so nothing is cleared out from under it.
func TestACopyThatIsMadeOnlyForgetsTheOneBefore(t *testing.T) {
	noNativeClipboard(t)
	first := ClipboardSetSecret([]byte("passphrase"), time.Minute)
	second := ClipboardSetSecret([]byte("another"), time.Minute)
	if !first.Cleared() {
		t.Error("the copy that was replaced is still live")
	}
	if second.Cleared() {
		t.Error("the copy that was made was ended with the one it replaced")
	}
	if !ClipboardHoldsSecret() {
		t.Error("nothing is held after a copy that was made")
	}
}
