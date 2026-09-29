package widgets

import (
	"bytes"
	"testing"
)

// SetBytes is how a generated passphrase gets into the field. It takes
// the buffer over rather than copying it: a copy would leave the caller
// holding the original, and two buffers to wipe instead of one.
func TestSecretFieldSetBytesTakesOwnership(t *testing.T) {
	f := NewSecretField("")
	f.TextInput('o')
	f.TextInput('l')
	f.TextInput('d')
	old := f.Bytes()
	if string(old) != "old" {
		t.Fatalf("setup: %q", old)
	}

	gen := []byte("correct horse battery staple")
	f.SetBytes(gen)

	if got := f.Bytes(); !bytes.Equal(got, []byte("correct horse battery staple")) {
		t.Errorf("Bytes() = %q", got)
	}
	if f.Len() != len("correct horse battery staple") {
		t.Errorf("Len() = %d", f.Len())
	}
	if f.Caret() != len(gen) {
		t.Errorf("caret at %d, want the end (%d)", f.Caret(), len(gen))
	}
	if a, b := f.Selection(); a != b {
		t.Errorf("selection %d..%d, want none", a, b)
	}
}

// The buffer it replaces is wiped, so the old passphrase does not
// survive in freed memory.
func TestSecretFieldSetBytesWipesWhatWasThere(t *testing.T) {
	f := NewSecretField("")
	for _, r := range "secret" {
		f.TextInput(r)
	}
	// Reach into the field's own buffer so the wipe can be observed: a
	// copy would not show it.
	was := f.buf[:len(f.buf):len(f.buf)]
	f.SetBytes([]byte("new"))

	if !bytes.Equal(was, make([]byte, len(was))) {
		t.Errorf("the old buffer still holds %q", was)
	}
}

// Handed its own buffer back, the field must not wipe the value being
// set.
func TestSecretFieldSetBytesWithItsOwnBuffer(t *testing.T) {
	f := NewSecretField("")
	for _, r := range "same" {
		f.TextInput(r)
	}
	f.SetBytes(f.buf)
	if got := f.Bytes(); string(got) != "same" {
		t.Errorf("Bytes() = %q, want %q", got, "same")
	}
}

// An empty slice is a legal value and must not panic on the aliasing
// check.
func TestSecretFieldSetBytesEmpty(t *testing.T) {
	f := NewSecretField("")
	f.SetBytes(nil)
	if !f.Empty() || f.Len() != 0 {
		t.Error("SetBytes(nil) should empty the field")
	}
	f.SetBytes([]byte{})
	if !f.Empty() {
		t.Error("SetBytes of an empty slice should empty the field")
	}
	f.SetBytes([]byte("x"))
	f.SetBytes(nil)
	if !f.Empty() {
		t.Error("SetBytes(nil) over a value should empty the field")
	}
}

// Wipe still works on a buffer the field was handed.
func TestSecretFieldWipeAfterSetBytes(t *testing.T) {
	f := NewSecretField("")
	gen := []byte("passphrase")
	f.SetBytes(gen)
	f.Wipe()
	if !f.Empty() {
		t.Error("the field is not empty after Wipe")
	}
	if !bytes.Equal(gen, make([]byte, len(gen))) {
		t.Errorf("the handed-over buffer still holds %q", gen)
	}
}

// OnChange fires, so a strength meter set up on the field follows a
// generated value in.
func TestSecretFieldSetBytesNotifies(t *testing.T) {
	f := NewSecretField("")
	n := 0
	f.OnChange = func(*SecretField) { n++ }
	f.SetBytes([]byte("abc"))
	if n != 1 {
		t.Errorf("OnChange fired %d times, want 1", n)
	}
}
