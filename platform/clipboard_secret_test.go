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
	ClipboardSetSecret([]byte("hunter2"), 0)
	if !ClipboardHoldsSecret() {
		t.Fatal("a zero timeout cleared it at once")
	}
}
