package platform

import "time"

// _NET_WM_SYNC_REQUEST (EWMH 1.5, "_NET_WM_SYNC_REQUEST"): the handshake an
// X11 window manager uses to resize a window no faster than the client can
// paint it. The client lists _NET_WM_SYNC_REQUEST in WM_PROTOCOLS and names
// an XSync counter in _NET_WM_SYNC_REQUEST_COUNTER; before each resize step
// the manager sends a WM_PROTOCOLS client message carrying a 64-bit value,
// then configures the window; the client sets the counter to that value
// once it has drawn the window at its new size, and the manager, which
// waits for the counter, only then shows the new size and takes the next
// step. Without it a drag-resize shows the window half drawn — the old
// frame stretched, or black where nothing has been painted yet — and runs
// ahead of the client. The pure state below is x11_sync_linux.go's, kept
// free of cgo so it is tested headless.

// EnvX11Sync set to 0 makes the X11 backend leave _NET_WM_SYNC_REQUEST out:
// a window the manager resizes as fast as it likes, which is how the
// difference is measured.
const EnvX11Sync = "UITK_X11_SYNC"

// syncOverdue is how long a request may wait for the frame that answers it
// before it is answered anyway: an app that does not repaint on a resize (or
// a request no configure followed) must not leave the window manager's
// resize stalled. KWin waits far longer; a frame is 16 ms.
const syncOverdue = 250 * time.Millisecond

// syncRequest is one window's side of the handshake.
type syncRequest struct {
	value uint64
	// waiting: a request came and has not been answered; armed: the
	// configure after it changed the window's size, so a frame at the new
	// size is on its way; since is when the wait (or the arming) began.
	waiting, armed bool
	since          time.Time
}

// request takes the manager's client message: the value to answer with.
// A request that replaces one still waiting takes its place — the manager
// only ever waits for the latest value.
func (r *syncRequest) request(value uint64, now time.Time) {
	r.value, r.waiting, r.armed, r.since = value, true, false, now
}

// configured follows the ConfigureNotify after a request. A size that did
// not change leaves nothing to draw, so the request is answered at once;
// a new size is answered by the frame that shows it (presented).
func (r *syncRequest) configured(resized bool, now time.Time) (answer bool, value uint64) {
	if !r.waiting {
		return false, 0
	}
	if !resized {
		r.waiting = false
		return true, r.value
	}
	r.armed, r.since = true, now
	return false, 0
}

// presented follows a frame put on the window: the one that answers an
// armed request. A frame before the configure (a caret blink while the
// manager is still deciding) answers nothing.
func (r *syncRequest) presented() (answer bool, value uint64) {
	if !r.waiting || !r.armed {
		return false, 0
	}
	r.waiting, r.armed = false, false
	return true, r.value
}

// overdue answers a request that has waited longer than limit for its frame.
func (r *syncRequest) overdue(now time.Time, limit time.Duration) (answer bool, value uint64) {
	if !r.waiting || now.Sub(r.since) < limit {
		return false, 0
	}
	r.waiting, r.armed = false, false
	return true, r.value
}

// deadline is when a waiting request becomes overdue (zero when none
// waits), so an idle loop wakes for it.
func (r *syncRequest) deadline(limit time.Duration) time.Time {
	if !r.waiting {
		return time.Time{}
	}
	return r.since.Add(limit)
}

// syncValue is the 64-bit value of a _NET_WM_SYNC_REQUEST message, whose
// data.l[2] holds its low 32 bits and data.l[3] its high 32.
func syncValue(lo, hi uint32) uint64 { return uint64(hi)<<32 | uint64(lo) }
