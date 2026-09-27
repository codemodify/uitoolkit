//go:build linux

package platform

import (
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
)

// fdWaker is a pipe the UI loop can poll (Wayland wl_display + extra fd)
// and Application.Post can write to.
//
// The read end is owned by an *os.File so the Go runtime poller supplies
// the timeout. The previous syscall.Select version indexed a 1024-bit
// fd_set with the raw descriptor, which panics (Go) or smashes the stack
// (C) as soon as the process has more than 1024 descriptors open — a
// perfectly ordinary state for an app with many fonts, sockets and
// buffers. The raw descriptor is still available for C pollers via FD.
type fdWaker struct {
	r, w *os.File
	fd   int
}

func newFDWaker() *fdWaker {
	var p [2]int
	if err := syscall.Pipe2(p[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		return &fdWaker{fd: -1}
	}
	return &fdWaker{
		r:  os.NewFile(uintptr(p[0]), "uitk-wake-r"),
		w:  os.NewFile(uintptr(p[1]), "uitk-wake-w"),
		fd: p[0],
	}
}

// FD is the raw read descriptor, for a C poll loop (wl_display + waker).
// It stays registered with the Go poller: reading it from C is only safe
// because every Go-side read goes through Wait / Drain.
func (w *fdWaker) FD() int {
	if w == nil || w.r == nil {
		return -1
	}
	return w.fd
}

// Signal makes the next Wait return at once. It never blocks.
//
// It writes the raw descriptor rather than calling os.File.Write. The
// pipe is O_NONBLOCK, but its write end is registered with the Go
// runtime poller, and os.File.Write on a registered descriptor does not
// answer EAGAIN — it parks the goroutine until the pipe drains. Nothing
// guarantees a drain: WakeSurface signals this pipe for every post while
// the X11 and offscreen waits never read it, so a run of posts fills the
// buffer and the *next* Signal blocks before it can wake the surface.
// Posting from the UI thread then deadlocks the loop against itself.
//
// A full pipe already means what the byte would mean, so EAGAIN is the
// answer rather than a problem: a wake is pending and unread, and one
// more byte would say nothing new. That is what makes a wake pipe
// coalescing, and it holds only if the write refuses to wait.
func (w *fdWaker) Signal() {
	if w == nil || w.w == nil {
		return
	}
	rc, err := w.w.SyscallConn()
	if err != nil {
		return
	}
	_ = rc.Write(func(fd uintptr) bool {
		var b [1]byte
		_, _ = syscall.Write(int(fd), b[:])
		// Always done. Returning false asks the poller to wait for the
		// pipe to become writable, which is the block this avoids.
		return true
	})
}

// Drain empties the pipe. It reads the descriptor directly until it
// would block: a Read under a deadline already past fails before it
// reads anything, which left the pipe readable for ever and a poller
// waiting on it spinning.
func (w *fdWaker) Drain() {
	if w == nil || w.r == nil {
		return
	}
	rc, err := w.r.SyscallConn()
	if err != nil {
		return
	}
	_ = rc.Read(func(fd uintptr) bool {
		var buf [64]byte
		for {
			if n, err := syscall.Read(int(fd), buf[:]); n <= 0 || err != nil {
				return true
			}
		}
	})
}

func (w *fdWaker) Wait(timeout time.Duration) bool {
	if w == nil || w.r == nil {
		if timeout > 0 {
			time.Sleep(timeout)
		}
		return false
	}
	if timeout >= 0 {
		_ = w.r.SetReadDeadline(time.Now().Add(timeout))
	} else {
		_ = w.r.SetReadDeadline(time.Time{})
	}
	var buf [64]byte
	_, err := w.r.Read(buf[:])
	_ = w.r.SetReadDeadline(time.Time{})
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return false
		}
		return false
	}
	w.Drain()
	return true
}

func (w *fdWaker) Close() {
	if w == nil {
		return
	}
	if w.r != nil {
		_ = w.r.Close()
		w.r = nil
	}
	if w.w != nil {
		_ = w.w.Close()
		w.w = nil
	}
	w.fd = -1
}

var (
	loopWaker   = newFDWaker()
	loopWakerMu sync.Mutex
)

func signalLoopWake() {
	loopWakerMu.Lock()
	w := loopWaker
	loopWakerMu.Unlock()
	w.Signal()
}

func waitLoopWake(timeout time.Duration) bool {
	loopWakerMu.Lock()
	w := loopWaker
	loopWakerMu.Unlock()
	return w.Wait(timeout)
}

// drainLoopWake empties the loop waker after a wait saw it signalled.
func drainLoopWake() {
	loopWakerMu.Lock()
	w := loopWaker
	loopWakerMu.Unlock()
	w.Drain()
}

func loopWakeFD() int {
	loopWakerMu.Lock()
	defer loopWakerMu.Unlock()
	return loopWaker.FD()
}
