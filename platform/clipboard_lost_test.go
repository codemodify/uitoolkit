package platform

import (
	"bytes"
	"testing"
	"time"
)

// Losing a selection ends the copy that was on it, not whatever is held
// when the word arrives.
//
// The window system reports a lost selection on its own schedule, and on
// Wayland that word reaches us on a goroutine. By then the program may
// have copied something else — and ending "whatever is held" would end
// that instead: a passphrase the user had just put on the clipboard
// going quietly missing, wiped and reported gone while it is still
// there.
func TestLosingASelectionEndsOnlyTheCopyThatWasOnIt(t *testing.T) {
	t.Cleanup(ClipboardClear)

	first := []byte("first-secret")
	c1 := ClipboardSetSecret(first, 30*time.Second)

	// The program copies something else before the loss is reported.
	second := []byte("second-secret")
	_ = ClipboardSetSecret(second, 30*time.Second)

	// Now the news about the *first* arrives.
	forgetHeldSecretIf(c1)

	if !ClipboardHoldsSecret() {
		t.Fatal("the second copy was ended by news about the first")
	}
	if bytes.Equal(second, make([]byte, len(second))) {
		t.Error("the second copy's bytes were wiped by news about the first")
	}
	// And news about the one actually held does end it.
	forgetHeldSecretIf(heldSecret())
	if ClipboardHoldsSecret() {
		t.Error("the held copy survived news about itself")
	}
	if !bytes.Equal(second, make([]byte, len(second))) {
		t.Error("the held copy was ended without being wiped")
	}
}
