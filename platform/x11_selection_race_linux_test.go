//go:build linux && cgo

package platform

import (
	"os"
	"testing"
	"time"
)

// Two races on X11's selection, both of which read as "the clipboard is
// empty" and neither of which the suite could see until it ran against a
// server. Each is rare on its own — about one try in ten — so each of these
// repeats the sequence until a race would have to be very lucky indeed.

// A copy, a clear and another copy leave the clipboard holding the second
// copy.
//
// Giving a selection up and taking it again makes the server send a
// SelectionClear for the ownership that ended, and it arrives *after* the new
// one is in hand. Acting on it turned ownClip off while the selection really
// was this process's, so the next paste asked the server to convert a
// selection this process then refused to serve and read nothing at all; the
// ownership timestamp was too old to tell the stale clear from a real one,
// because it came from whenever the last event happened to arrive.
func TestACopyAfterAClearIsWhatIsPasted(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Cleanup(ClipboardClear)
	const want = "the user's own copy"
	for i := 0; i < 20; i++ {
		ClipboardSet("something else")
		ClipboardClear()
		ClipboardSet(want)
		// The clear for the ownership that ended is in flight: read it now,
		// which is what the selection service does on its own tick a few
		// milliseconds later. Without this the paste below happens to drain
		// it itself, *after* deciding what to do, and the race never shows.
		pumpX11(t)
		if got := ClipboardGet(); got != want {
			t.Fatalf("round %d: the clipboard reads %q, want %q", i, got, want)
		}
	}
}

// And the same for the secret path, which reads bytes rather than a string
// and has its own way in.
func TestASecretCopyAfterAClearIsWhatIsPasted(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Cleanup(ClipboardClear)
	for i := 0; i < 20; i++ {
		ClipboardSet("something else")
		ClipboardClear()
		// Not through ClipboardSetSecret, whose held copy would answer the
		// read without the X server being asked at all: this is about what
		// the *selection* holds.
		c, err := x11Retain()
		if err != nil {
			t.Skip("no X11 connection")
		}
		c.setClipboardSelBytes([]byte("the-passphrase"), true)
		c.release()
		pumpX11(t)
		got, ok := x11ClipGetBytes(false)
		if !ok || string(got) != "the-passphrase" {
			t.Fatalf("round %d: the selection reads %q (ok=%v)", i, got, ok)
		}
	}
}

// A read while the connection is being let go reads what is there.
//
// This process keeps its X connection open only while something is using it
// or while it owns a selection, so the moment another client takes the
// clipboard the selection service lets the connection go — and a paste that
// had just started had the display closed under it. It came back empty after
// a single wait, which reads exactly like an empty clipboard, and it is a
// use-after-free besides.
func TestAPasteSurvivesTheConnectionBeingLetGo(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Cleanup(ClipboardClear)
	const theirs = "what the other program copied"
	for i := 0; i < 10; i++ {
		// This process owns the clipboard...
		ClipboardSet("ours")
		// ...and another program takes it, which is what makes this process
		// stop keeping its connection open.
		clipOwner(t, theirs)
		for j := 0; j < 50; j++ {
			pumpX11(t)
		}
		if got := ClipboardGet(); got != theirs {
			t.Fatalf("round %d: the paste read %q, want the other program's copy", i, got)
		}
	}
}

// A read holds the connection open for as long as it takes.
//
// Without the reference the selection service could close the display between
// the conversion request and its answer — which is not only an empty paste
// but a use-after-free, since the wait holds the Display across the unlock.
// The owner here is another program, so this process owns no selection and
// the service is free to let the connection go, which is exactly the state
// the bug needed.
func TestAReadHoldsTheConnectionOpen(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Cleanup(ClipboardClear)
	clipOwner(t, "theirs")
	c, err := x11Retain()
	if err != nil {
		t.Skip("no X11 connection")
	}
	defer c.release()

	// Sampled while the read is in flight: the read is a round trip to
	// another process, so there are milliseconds to look in.
	seen := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		best := 0
		for {
			select {
			case <-done:
				seen <- best
				return
			default:
			}
			x11Mu.Lock()
			if c.refs > best {
				best = c.refs
			}
			x11Mu.Unlock()
			time.Sleep(100 * time.Microsecond)
		}
	}()
	// One reference is this test's own, so a read that holds one of its own
	// takes it to two.
	before := refsOf(c)
	if got, ok := x11ClipGetBytes(false); !ok || string(got) != "theirs" {
		t.Fatalf("the read came back %q (ok=%v)", got, ok)
	}
	close(done)
	if peak := <-seen; peak <= before {
		t.Errorf("the connection was held at %d references during the read, no more than the %d before it", peak, before)
	}
	if after := refsOf(c); after != before {
		t.Errorf("the read left the reference count at %d, not the %d it found", after, before)
	}
}

func refsOf(c *x11Conn) int {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return c.refs
}
