//go:build linux

package platform

import (
	"testing"
	"time"
)

// A Wayland transfer is a pipe the other side writes and closes, so only
// EOF says the payload is whole. A timeout, a read error and the size cap
// all leave a prefix — and a prefix handed back as success is a truncated
// paste, or, for a drag negotiated as Move, an acknowledgement of data
// the application never received, after which the source is entitled to
// delete the original.
func TestTransferOnlyEOFCounts(t *testing.T) {
	chunks := func(parts ...string) func([]byte, int) int {
		i := 0
		return func(buf []byte, ms int) int {
			if i >= len(parts) {
				return 0 // EOF
			}
			n := copy(buf, parts[i])
			i++
			return n
		}
	}
	far := func() time.Time { return time.Now().Add(time.Minute) }

	t.Run("EOF is complete", func(t *testing.T) {
		out, how := wlReadLoop(chunks("hello ", "world"), far())
		if !how.ok() || string(out) != "hello world" {
			t.Fatalf("got %q %v", out, how)
		}
	})

	t.Run("a stalled sender is not complete", func(t *testing.T) {
		// "partial" arrives, then the source keeps the pipe open and
		// the reader reports a timeout it cannot tell from an error.
		i := 0
		read := func(buf []byte, ms int) int {
			if i == 0 {
				i++
				return copy(buf, "partial")
			}
			return -1
		}
		out, how := wlReadLoop(read, far())
		if how.ok() {
			t.Fatalf("a stalled transfer reported success with %q", out)
		}
		if string(out) != "partial" {
			t.Fatalf("the prefix read so far is %q", out)
		}
	})

	t.Run("an expired deadline is not complete", func(t *testing.T) {
		out, how := wlReadLoop(chunks("anything"), time.Now().Add(-time.Second))
		if how.ok() {
			t.Fatalf("an expired deadline reported success with %q", out)
		}
		if how != wlTransferTimedOut {
			t.Fatalf("ended as %v, want timed out", how)
		}
	})

	t.Run("past the cap is not complete", func(t *testing.T) {
		big := make([]byte, 1<<20)
		read := func(buf []byte, ms int) int { return copy(buf, big) }
		out, how := wlReadLoop(read, far())
		if how.ok() {
			t.Fatalf("an unbounded sender reported success")
		}
		if how != wlTransferTooBig {
			t.Fatalf("ended as %v, want too big", how)
		}
		if len(out) <= wlTransferMax {
			t.Fatalf("stopped at %d bytes, before the cap", len(out))
		}
	})
}
