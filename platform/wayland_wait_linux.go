//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <wayland-client.h>

// ui_wl_wait_begin gets the display ready to read: 1 when events were
// already queued (dispatched now, nothing to wait for), 0 when it is
// prepared and flushed and the caller waits, -1 with no display.
static int ui_wl_wait_begin(struct wl_display *d) {
	if (!d) return -1;
	if (wl_display_prepare_read(d) != 0) {
		wl_display_dispatch_pending(d);
		return 1;
	}
	wl_display_flush(d);
	return 0;
}

// ui_wl_wait_end reads what arrived (or gives the prepared read back) and
// dispatches it.
static int ui_wl_wait_end(struct wl_display *d, int readable) {
	if (readable) {
		wl_display_read_events(d);
	} else {
		wl_display_cancel_read(d);
	}
	return wl_display_dispatch_pending(d);
}

static int ui_wl_display_fd(struct wl_display *d) { return d ? wl_display_get_fd(d) : -1; }
*/
import "C"

import (
	"os"
	"syscall"
	"time"
)

// The run loop's wait on Wayland.
//
// It used to be a poll(2) inside cgo. A goroutine in a cgo call keeps its
// P, and the Go runtime's sysmon thread will not take an idle P back from
// a call for 10 ms, polling every 20 µs and backing off meanwhile: some
// sixty wake-ups after every frame, which is 130 a second for a blinking
// caret that paints twice. Here the goroutine parks in the Go poller
// instead, its P goes idle at once, and sysmon sleeps.
//
// The poller waits on an epoll set of the connection's own that holds the
// display's fd and the loop waker's: an epoll fd is readable while any fd
// in it is, so one pollable file stands for both, with the timeout as its
// read deadline. libwayland's read protocol is kept around the wait:
// prepare_read and flush before it, then read_events or cancel_read.

// wlPoller is a connection's epoll set, in the Go poller.
type wlPoller struct {
	f       *os.File
	rc      syscall.RawConn
	epfd    int
	display int
	wake    int // the loop waker's fd in the set, -1 none yet
}

func newWlPoller(display int) *wlPoller {
	if display < 0 {
		return nil
	}
	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil
	}
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(display)}
	if syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, display, &ev) != nil || syscall.SetNonblock(epfd, true) != nil {
		syscall.Close(epfd)
		return nil
	}
	f := os.NewFile(uintptr(epfd), "uitk-wl-poll")
	rc, err := f.SyscallConn()
	if err != nil {
		f.Close()
		return nil
	}
	return &wlPoller{f: f, rc: rc, epfd: epfd, display: display, wake: -1}
}

// watchWake puts the loop waker's fd in the set (it is made once, but
// follow it should it change).
func (p *wlPoller) watchWake(fd int) {
	if fd == p.wake {
		return
	}
	if p.wake >= 0 {
		_ = syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_DEL, p.wake, nil)
		p.wake = -1
	}
	if fd < 0 {
		return
	}
	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(fd)}
	if syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_ADD, fd, &ev) == nil {
		p.wake = fd
	}
}

// wait blocks until the display has something to read or the waker was
// signalled, or timeout passes (< 0: no limit; 0: just look).
func (p *wlPoller) wait(timeout time.Duration) (display, woken bool) {
	var evs [2]syscall.EpollEvent
	ready := func() bool {
		n, _ := syscall.EpollWait(p.epfd, evs[:], 0)
		for _, ev := range evs[:max(n, 0)] {
			if int(ev.Fd) == p.display {
				display = true
			} else {
				woken = true
			}
		}
		return n > 0
	}
	if timeout == 0 {
		ready()
		return
	}
	if timeout > 0 {
		_ = p.f.SetReadDeadline(time.Now().Add(timeout))
	} else {
		_ = p.f.SetReadDeadline(time.Time{})
	}
	// Read calls ready again each time the poller sees the set readable;
	// false parks the goroutine until then or the deadline.
	_ = p.rc.Read(func(uintptr) bool { return ready() })
	return
}

func (p *wlPoller) close() {
	if p != nil {
		_ = p.f.Close()
	}
}

// waitDisplay is wlSurface.Wait's body: false when there is no poller to
// wait with, and the caller falls back on the poll in C.
func (c *wlConn) waitDisplay(timeout time.Duration) (dispatched, ok bool) {
	if c.poll == nil {
		if c.pollFailed {
			return false, false
		}
		if c.poll = newWlPoller(int(C.ui_wl_display_fd(c.dpy))); c.poll == nil {
			c.pollFailed = true
			return false, false
		}
	}
	c.poll.watchWake(loopWakeFD())
	switch C.ui_wl_wait_begin(c.dpy) {
	case -1:
		return false, true
	case 1:
		return true, true
	}
	display, woken := c.poll.wait(timeout)
	readable := C.int(0)
	if display {
		readable = 1
	}
	n := C.ui_wl_wait_end(c.dpy, readable)
	if woken {
		drainLoopWake()
	}
	return n != 0 || woken, true
}
