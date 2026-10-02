package platform

import "testing"

// A transfer that grows leaves nothing behind.
//
// append reallocates when it runs out of capacity and drops the old array
// for the collector with whatever was in it. A passphrase read in pieces
// — a private key pasted into a secret field, over a Wayland pipe or an
// X11 INCR transfer — therefore left a copy of every prefix it outgrew,
// each holding the start of the secret and none of them reachable.
func TestAGrowingTransferWipesWhatItOutgrew(t *testing.T) {
	var out []byte
	var seen [][]byte // the arrays it grew through
	for i := 0; i < 40; i++ {
		before := out
		out = appendWiping(out, []byte("0123456789abcdef"))
		if cap(before) != 0 && cap(before) != cap(out) {
			seen = append(seen, before[:cap(before)])
		}
	}
	if len(seen) == 0 {
		t.Fatal("it never grew, so there is nothing to check")
	}
	for i, old := range seen {
		for j, b := range old {
			if b != 0 {
				t.Fatalf("array %d it outgrew still holds byte %d: %q", i, j, b)
			}
		}
	}
	// And it still assembles the right thing.
	if len(out) != 40*16 {
		t.Errorf("assembled %d bytes, want %d", len(out), 40*16)
	}
	for i := 0; i < len(out); i += 16 {
		if string(out[i:i+16]) != "0123456789abcdef" {
			t.Fatalf("the payload is wrong at %d", i)
		}
	}
}
