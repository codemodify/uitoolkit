package platform

import (
	"testing"
	"time"
)

// An application shows "copied — clears in 45s". Without a callback it
// has to poll Cleared, which is a timer of its own and a window in which
// the interface says the passphrase is still on the clipboard when it is
// not.
func TestSecretClipOnClearedFiresOnTimeout(t *testing.T) {
	done := make(chan struct{})
	c := ClipboardSetSecret([]byte("hunter2"), 20*time.Millisecond)
	c.OnCleared(func() { close(done) })

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the timeout cleared the copy without saying so")
	}
	if !c.Cleared() {
		t.Error("Cleared is false after OnCleared fired")
	}
}

// A second copy replaces the first, which is a clearing of the first.
func TestSecretClipOnClearedFiresWhenReplaced(t *testing.T) {
	done := make(chan struct{})
	first := ClipboardSetSecret([]byte("one"), time.Hour)
	first.OnCleared(func() { close(done) })

	second := ClipboardSetSecret([]byte("two"), time.Hour)
	defer second.Clear()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the replaced copy never reported being cleared")
	}
}

// Installing on a copy that is already gone must not lose the event to a
// timeout that beat the caller to it.
func TestSecretClipOnClearedAfterTheFact(t *testing.T) {
	c := ClipboardSetSecret([]byte("gone"), time.Hour)
	c.Clear()

	called := false
	c.OnCleared(func() { called = true })
	if !called {
		t.Error("OnCleared on an already-cleared copy said nothing")
	}
}

// It fires once, not once per route to the same clearing.
func TestSecretClipOnClearedFiresOnce(t *testing.T) {
	n := 0
	c := ClipboardSetSecret([]byte("once"), time.Hour)
	c.OnCleared(func() { n++ })
	c.Clear()
	c.Clear()
	ClipboardClear()
	if n != 1 {
		t.Errorf("OnCleared fired %d times, want 1", n)
	}
}

// A nil clip is already cleared, and says so rather than panicking.
func TestSecretClipOnClearedNil(t *testing.T) {
	var c *SecretClip
	called := false
	c.OnCleared(func() { called = true })
	if !called {
		t.Error("a nil clip should report itself cleared")
	}
}
