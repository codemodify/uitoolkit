package platform

import (
	"bytes"
	"testing"
	"time"
)

// A copied secret ends when the clipboard moves on.
//
// A SecretClip was held from ClipboardSetSecret until it was cleared, and
// nothing ended it when the clipboard went elsewhere. For the rest of its
// timeout — thirty seconds in a password manager — a paste into a
// SecretField read the *old* secret instead of what was actually on the
// clipboard, ClipboardHoldsSecret went on saying yes, and the clear at
// the timeout emptied whatever the clipboard held by then.
func TestAnOrdinaryCopyEndsTheHeldSecret(t *testing.T) {
	t.Cleanup(ClipboardClear)
	b := []byte("hunter2")
	ClipboardSetSecret(b, 30*time.Second)
	if !ClipboardHoldsSecret() {
		t.Fatal("the secret was not held")
	}

	// Anything else copied — here the program's own ordinary copy — and
	// the secret is no longer what the clipboard has.
	ClipboardSet("an ordinary thing")

	if ClipboardHoldsSecret() {
		t.Error("a secret is still held after an ordinary copy")
	}
	if !bytes.Equal(b, make([]byte, len(b))) {
		t.Errorf("the held copy was not wiped when it was superseded: %q", b)
	}
	// A secret paste now reads the clipboard rather than serving the old
	// secret from memory.
	if got := string(ClipboardGetSecret()); got != "an ordinary thing" {
		t.Errorf("a secret paste reads %q, want what was actually copied", got)
	}
}
