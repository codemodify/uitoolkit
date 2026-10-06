package platform

import "testing"

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
