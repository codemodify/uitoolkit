//go:build linux

package platform

import "time"

// How a selection or drop transfer ended.
//
// The distinction is the whole point: a Wayland transfer is a pipe the
// other side writes and closes, so **only EOF says the payload is
// whole**. A timeout, a read error and the size cap all leave a prefix
// of the data, and a prefix is not a short answer — it is a wrong one.
// Handing it back as success gives a truncated paste, and for a drag
// negotiated as Move it lets the application acknowledge a payload it
// did not receive, after which the source is entitled to delete the
// original.
type wlTransfer int

const (
	wlTransferComplete wlTransfer = iota // EOF: the source finished writing
	wlTransferTimedOut                   // the deadline passed with the pipe still open
	wlTransferFailed                     // the read failed
	wlTransferTooBig                     // past wlTransferMax
)

func (t wlTransfer) ok() bool { return t == wlTransferComplete }

func (t wlTransfer) String() string {
	switch t {
	case wlTransferComplete:
		return "complete"
	case wlTransferTimedOut:
		return "timed out"
	case wlTransferFailed:
		return "failed"
	case wlTransferTooBig:
		return "too big"
	}
	return "unknown"
}

// wlTransferMax bounds what one selection or drop may carry, so a source
// that writes without end costs memory rather than all of it.
const wlTransferMax = 64 << 20

// wlReadLoop reads chunks until the transfer ends, and says how it
// ended. read fills buf, waiting at most ms milliseconds, and answers
// the way read(2) does: a positive count, 0 for EOF, negative for an
// error or a timeout it could not distinguish.
//
// It is separate from the cgo caller so the completion rule can be
// tested without a compositor.
func wlReadLoop(read func(buf []byte, ms int) int, deadline time.Time) ([]byte, wlTransfer) {
	var out []byte
	buf := make([]byte, 4096)
	// The staging buffer holds whatever came through it, which for a
	// passphrase pasted from another program is the passphrase. Zeroed on
	// the way out so the only copy left is the one the caller took.
	defer func() {
		for i := range buf {
			buf[i] = 0
		}
	}()
	for {
		remain := time.Until(deadline)
		if remain <= 0 {
			return out, wlTransferTimedOut
		}
		n := read(buf, int(remain/time.Millisecond)+1)
		switch {
		case n == 0:
			return out, wlTransferComplete
		case n < 0:
			// The reader cannot tell a poll timeout from a read
			// error, and neither is EOF, so neither is complete.
			if time.Now().After(deadline) {
				return out, wlTransferTimedOut
			}
			return out, wlTransferFailed
		}
		out = append(out, buf[:n]...)
		if len(out) > wlTransferMax {
			return out, wlTransferTooBig
		}
	}
}
