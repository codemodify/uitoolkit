//go:build linux

package platform

import (
	"sync"
	"syscall"
	"time"
)

// fdWaker is a pipe the UI loop can poll (Wayland wl_display + extra fd)
// and Application.Post can write to. Pipe2 is in Go 1.22 syscall; Eventfd
// is not.
type fdWaker struct {
	r, w int
}

func newFDWaker() *fdWaker {
	var p [2]int
	if err := syscall.Pipe2(p[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		return &fdWaker{r: -1, w: -1}
	}
	return &fdWaker{r: p[0], w: p[1]}
}

func (w *fdWaker) FD() int {
	if w == nil {
		return -1
	}
	return w.r
}

func (w *fdWaker) Signal() {
	if w == nil || w.w < 0 {
		return
	}
	var b [1]byte
	_, _ = syscall.Write(w.w, b[:])
}

func (w *fdWaker) Drain() {
	if w == nil || w.r < 0 {
		return
	}
	var buf [64]byte
	for {
		if _, err := syscall.Read(w.r, buf[:]); err != nil {
			break
		}
	}
}

func (w *fdWaker) Wait(timeout time.Duration) bool {
	if w == nil || w.r < 0 {
		if timeout > 0 {
			time.Sleep(timeout)
		}
		return false
	}
	var set syscall.FdSet
	set.Bits[w.r/64] |= 1 << (uint(w.r) % 64)
	var tv *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(int64(timeout))
		tv = &t
	}
	n, err := syscall.Select(w.r+1, &set, nil, nil, tv)
	if err != nil || n <= 0 {
		return false
	}
	w.Drain()
	return true
}

func (w *fdWaker) Close() {
	if w == nil {
		return
	}
	if w.r >= 0 {
		_ = syscall.Close(w.r)
		w.r = -1
	}
	if w.w >= 0 {
		_ = syscall.Close(w.w)
		w.w = -1
	}
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

func loopWakeFD() int {
	loopWakerMu.Lock()
	defer loopWakerMu.Unlock()
	return loopWaker.FD()
}
