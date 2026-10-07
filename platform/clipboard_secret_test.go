package platform

import (
	"bytes"
	"testing"
	"time"
)

// A secret copy is readable, is not left in the ordinary in-memory
// clipboard as a string, and comes back as bytes.
func TestSecretClipboardRoundTrip(t *testing.T) {
	t.Cleanup(ClipboardClear)
	// About SecretClip's own bookkeeping, not about any backend, so the
	// native seam stands in: under a live session (tools/test-display.sh)
	// the compositor cancels the data source of a client with no window and
	// no focus within milliseconds, which legitimately ends the held copy —
	// and this test would then be measuring the compositor.
	noNativeClipboard(t)
	ClipboardSet("something ordinary")
	c := ClipboardSetSecret([]byte("hunter2"), -1)
	if !ClipboardHoldsSecret() {
		t.Fatal("the clipboard does not hold a secret")
	}
	got := ClipboardGetSecret()
	if string(got) != "hunter2" {
		t.Fatalf("read back %q", got)
	}
	// The in-process string buffer was not given the secret.
	clipMu.Lock()
	inMemory := clip
	clipMu.Unlock()
	if inMemory != "" {
		t.Fatalf("the in-memory clipboard holds %q", inMemory)
	}
	c.Clear()
	if ClipboardHoldsSecret() {
		t.Fatal("still held after Clear")
	}
}

// ClipboardSetSecret takes ownership: after the clear, the caller's
// slice is zero.
func TestSecretClipboardWipesTheCallersBuffer(t *testing.T) {
	t.Cleanup(ClipboardClear)
	b := []byte("correct horse battery staple")
	c := ClipboardSetSecret(b, -1)
	c.Clear()
	if !bytes.Equal(b, make([]byte, len(b))) {
		t.Fatalf("not wiped: %q", b)
	}
	if !c.Cleared() {
		t.Fatal("Cleared() is false")
	}
	// Clearing twice is not an error and does not touch anything else.
	c.Clear()
}

// The timeout clears it without anyone asking.
func TestSecretClipboardClearsItself(t *testing.T) {
	t.Cleanup(ClipboardClear)
	b := []byte("hunter2")
	ClipboardSetSecret(b, 30*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for ClipboardHoldsSecret() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if ClipboardHoldsSecret() {
		t.Fatal("still there after the timeout")
	}
	if !bytes.Equal(b, make([]byte, len(b))) {
		t.Fatalf("the timeout did not wipe it: %q", b)
	}
}

// A second secret supersedes the first, and the first is wiped rather
// than left for a timer that would clear a clipboard it no longer owns.
func TestSecondSecretSupersedesTheFirst(t *testing.T) {
	t.Cleanup(ClipboardClear)
	noNativeClipboard(t) // see TestSecretClipboardRoundTrip
	first := []byte("first")
	c1 := ClipboardSetSecret(first, -1)
	second := []byte("second")
	c2 := ClipboardSetSecret(second, -1)
	if !bytes.Equal(first, make([]byte, len(first))) {
		t.Fatalf("the superseded copy was not wiped: %q", first)
	}
	if !c1.Cleared() || c2.Cleared() {
		t.Fatalf("cleared: first=%v second=%v", c1.Cleared(), c2.Cleared())
	}
	if string(ClipboardGetSecret()) != "second" {
		t.Fatal("the clipboard is not the second secret")
	}
	// The stale handle must not clear the live copy.
	c1.Clear()
	if !ClipboardHoldsSecret() {
		t.Fatal("a stale handle cleared the live secret")
	}
}

// ClipboardClearSecret takes a secret back and leaves an ordinary copy
// alone. That distinction is what lets the toolkit clear on quit without
// throwing away whatever the user copied.
func TestClearSecretLeavesAnOrdinaryCopy(t *testing.T) {
	t.Cleanup(ClipboardClear)
	ClipboardSet("the user's own copy")
	ClipboardClearSecret()
	if got := ClipboardGet(); got != "the user's own copy" {
		t.Fatalf("an ordinary copy was cleared: %q", got)
	}
	ClipboardSetSecret([]byte("hunter2"), -1)
	ClipboardClearSecret()
	if ClipboardHoldsSecret() {
		t.Fatal("the secret survived")
	}
}

// Zero means the default timeout rather than "clear immediately".
func TestZeroTimeoutIsTheDefault(t *testing.T) {
	t.Cleanup(ClipboardClear)
	noNativeClipboard(t) // see TestSecretClipboardRoundTrip
	ClipboardSetSecret([]byte("hunter2"), 0)
	if !ClipboardHoldsSecret() {
		t.Fatal("a zero timeout cleared it at once")
	}
}

// The copy is wiped before the API stops saying it is held, not after.
//
// Clear dropped secretHeld first — so ClipboardHoldsSecret answered false
// at once — then told the window system, and only then zeroed the bytes.
// For the whole of that round trip to a compositor, an X server or the
// Windows clipboard, the passphrase was still in the heap while every way
// of asking said nothing was held. It read as a flaky test on a quiet
// machine and failed every time on a loaded VM.
func TestASecretIsWipedBeforeItStopsBeingHeld(t *testing.T) {
	t.Cleanup(ClipboardClear)
	for i := 0; i < 50; i++ {
		b := []byte("hunter2")
		c := ClipboardSetSecret(b, 0)
		c.Clear()
		// Clear has returned, so there is no race left to lose: by the
		// time anything can observe "not held", the bytes are zero.
		if ClipboardHoldsSecret() {
			t.Fatalf("round %d: still held after Clear", i)
		}
		if !bytes.Equal(b, make([]byte, len(b))) {
			t.Fatalf("round %d: not held, but the buffer still reads %q", i, b)
		}
	}
}

// A copy the platform refuses comes back already cleared, and is not held.
//
// Windows notes the clipboard's sequence number whether or not anything was
// copied, and macOS the pasteboard's changeCount, so a copy that failed — the
// clipboard held open by another program for every try, or SetClipboardData
// refusing the block after EmptyClipboard had already emptied the clipboard —
// was taken for one that was made. The program said it had copied, a paste was
// served the held secret while the clipboard held something else, and the clear
// at its time found the number unchanged and emptied a clipboard holding the
// person's own work.
//
// The seam reports now, and this is the rule the report feeds: a refusal ends
// the copy at once. What a given backend *does* with a refusal is its own test,
// and on Windows and macOS needs those machines.
func TestARefusedCopyComesBackCleared(t *testing.T) {
	t.Cleanup(ClipboardClear)
	secret := []byte("the-refused-passphrase")
	c := refuseNativeCopy(t, func() *SecretClip { return ClipboardSetSecret(secret, -1) })

	if !c.Cleared() {
		t.Error("a refused copy is not cleared")
	}
	if ClipboardHoldsSecret() {
		t.Error("a refused copy is held: a paste would be served it, and the clear would empty somebody else's clipboard")
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Errorf("a refused copy left the passphrase in the heap: %q", secret)
	}
	// And the application hears about it without polling.
	told := false
	c.OnCleared(func() { told = true })
	if !told {
		t.Error("OnCleared on a refused copy did not call back")
	}
}

// A copy with no window system at all is still a copy: there is nothing to
// refuse it, and the in-memory copy is the copy. This is the half of the rule
// that every displayless test depends on, and the one a careless reading of
// "report whether it took" would break.
func TestACopyWithNoWindowSystemIsStillHeld(t *testing.T) {
	t.Cleanup(ClipboardClear)
	c := ClipboardSetSecret([]byte("hunter2"), -1)
	if c.Cleared() {
		t.Fatal("a copy with no window system came back cleared")
	}
	if !ClipboardHoldsSecret() {
		t.Error("a copy with no window system is not held")
	}
	if got := ClipboardGetSecret(); string(got) != "hunter2" {
		t.Errorf("read back %q, want the copy", got)
	}
}
