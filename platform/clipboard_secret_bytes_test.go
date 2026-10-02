package platform

import "testing"

// The selection a process serves is held as bytes it can zero, not as a
// Go string it cannot.
//
// Owning a selection means this process goes on holding the value for as
// long as it owns it — that is how X11 and Wayland both work — so the
// question is not whether a copy exists but whether anything can wipe it.
// It used to be a string: dropped on a clear, a timeout or the loss of the
// selection, and then left in the heap until the collector felt like it.
func TestASelectionsCopyCanBeWiped(t *testing.T) {
	var c clipCache
	secret := []byte("correct horse battery staple")
	c.setBytes(secret)

	held, ok := c.getBytes()
	if !ok || string(held) != string(secret) {
		t.Fatalf("the cache did not hold what it was given: %q", held)
	}
	// The cache keeps its own copy, so the caller may wipe at once.
	if &held[0] == &secret[0] {
		t.Error("the cache kept the caller's slice instead of copying it")
	}

	c.invalidate()
	for i, b := range held[:len(secret)] {
		if b != 0 {
			t.Fatalf("losing the selection left byte %d as %q", i, b)
		}
	}
	if _, ok := c.getBytes(); ok {
		t.Error("the cache still answers after the selection was lost")
	}

	// Replacing one secret with another zeroes the first.
	c.setBytes(secret)
	first, _ := c.getBytes()
	c.setBytes([]byte("something else"))
	for i, b := range first[:len(secret)] {
		if b != 0 {
			t.Fatalf("the replaced value kept byte %d: %q", i, b)
		}
	}
}
