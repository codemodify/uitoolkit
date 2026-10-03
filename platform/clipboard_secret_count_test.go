//go:build windows || (darwin && cgo)

package platform

// Windows and macOS are never told that another program has copied
// something. They count changes instead — GetClipboardSequenceNumber,
// NSPasteboard.changeCount — so the only way to notice is to read the count
// again, which is what clipboardStillHoldsOurSecret does. X11 and Wayland are
// told, and have their own test under the display rig.
//
// clipboardNativeSet is how a test copies *without* the toolkit's
// bookkeeping: the clipboard changes, secretHeld is untouched, and this
// process is in exactly the state another program's copy leaves it in.

import (
	"bytes"
	"testing"
)

// After another program copies, a secret paste reads that copy.
//
// It used to read the held secret for the rest of its time — thirty seconds
// in a password manager. Copy a passphrase from another program into a new
// vault's "Passphrase" and "Repeat" and the vault was made with the item
// copied before: a passphrase nobody meant to set, and no sign that anything
// had happened.
func TestAnotherProgramsCopyEndsTheHeldSecret(t *testing.T) {
	t.Cleanup(ClipboardClear)

	secret := []byte("the-held-passphrase")
	c := ClipboardSetSecret(secret, -1)
	if !ClipboardHoldsSecret() {
		t.Skip("no clipboard here: the copy did not take")
	}

	const theirs = "what the other program copied"
	clipboardNativeSet(theirs)

	if ClipboardHoldsSecret() {
		t.Error("the clipboard still claims to hold this process's secret")
	}
	if !c.Cleared() {
		t.Error("the held copy was not ended")
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Errorf("the held bytes were not wiped: %q", secret)
	}
	got := ClipboardGetSecret()
	defer wipeBytes(got)
	if string(got) != theirs {
		t.Errorf("a secret paste read %q, want the other program's copy %q", got, theirs)
	}
}

// And a secret nobody has taken is still served, which is the whole point of
// holding it: a test that ended it unconditionally would pass the one above
// and break every paste.
func TestAnUntouchedSecretIsStillTheClipboards(t *testing.T) {
	t.Cleanup(ClipboardClear)

	secret := []byte("still-ours")
	want := string(secret)
	ClipboardSetSecret(secret, -1)
	if !ClipboardHoldsSecret() {
		t.Skip("no clipboard here: the copy did not take")
	}
	got := ClipboardGetSecret()
	defer wipeBytes(got)
	if string(got) != want {
		t.Errorf("a secret paste read %q, want the held secret", got)
	}
	if !ClipboardHoldsSecret() {
		t.Error("reading the secret ended it")
	}
}

func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
