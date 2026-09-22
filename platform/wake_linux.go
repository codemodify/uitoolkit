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

func (w *fdWaker) Signal() {
	if w == nil || w.w == nil {
		return
	}
	var b [1]byte
	_, _ = w.w.Write(b[:])
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
