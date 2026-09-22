//go:build linux && cgo

package platform

/*
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/extensions/sync.h>

static int ui_sync_init(Display *d) {
	int ev, err, maj, min;
	if (!XSyncQueryExtension(d, &ev, &err)) return 0;
	return XSyncInitialize(d, &maj, &min) ? 1 : 0;
}
static XSyncCounter ui_sync_counter(Display *d) {
	XSyncValue v;
	XSyncIntToValue(&v, 0);
	return XSyncCreateCounter(d, v);
}
static void ui_sync_set(Display *d, XSyncCounter c, unsigned int lo, unsigned int hi) {
	XSyncValue v;
	XSyncIntsToValue(&v, lo, (int)hi);
	XSyncSetCounter(d, c, v);
	XFlush(d);
}
static void ui_sync_destroy(Display *d, XSyncCounter c) { if (d && c) XSyncDestroyCounter(d, c); }

// ui_sync_announce lists _NET_WM_SYNC_REQUEST in WM_PROTOCOLS, beside
// WM_DELETE_WINDOW, and names the counter in _NET_WM_SYNC_REQUEST_COUNTER.
static void ui_sync_announce(Display *d, Window w, XSyncCounter c, Atom del, Atom req, Atom prop) {
	Atom protos[2] = { del, req };
	XSetWMProtocols(d, w, protos, 2);
	long id = (long)c;
	XChangeProperty(d, w, prop, XA_CARDINAL, 32, PropModeReplace, (unsigned char *)&id, 1);
}

// ui_sync_message reads a WM_PROTOCOLS client message: 1 with the value's
// halves when it is _NET_WM_SYNC_REQUEST.
static int ui_sync_message(XEvent *e, Atom protocols, Atom req, unsigned int *lo, unsigned int *hi) {
	if (e->type != ClientMessage || e->xclient.message_type != protocols || e->xclient.format != 32) return 0;
	if ((Atom)e->xclient.data.l[0] != req) return 0;
	*lo = (unsigned int)e->xclient.data.l[2];
	*hi = (unsigned int)e->xclient.data.l[3];
	return 1;
}
*/
import "C"

import (
	"os"
	"time"
)

// x11Sync is the window's XSync counter for _NET_WM_SYNC_REQUEST (xsync.go)
// and the handshake's state.
type x11Sync struct {
	counter C.XSyncCounter
	req     syncRequest
}

// syncReadyLocked reports whether the connection speaks the handshake: the
// server has XSync and UITK_X11_SYNC is not 0. The atoms are interned the
// first time.
func (c *x11Conn) syncReadyLocked() bool {
	if c.syncState == 0 {
		c.syncState = -1
		if os.Getenv(EnvX11Sync) != "0" && C.ui_sync_init(c.dpy) != 0 {
			c.syncState = 1
			c.atomProtocols = internAtom(c.dpy, "WM_PROTOCOLS")
			c.atomDelete = internAtom(c.dpy, "WM_DELETE_WINDOW")
			c.atomSyncReq = internAtom(c.dpy, "_NET_WM_SYNC_REQUEST")
			c.atomSyncCounter = internAtom(c.dpy, "_NET_WM_SYNC_REQUEST_COUNTER")
		}
	}
	return c.syncState > 0
}

// setupSyncLocked gives a toplevel its counter (once) and announces it on
// its X window — again after the window is made afresh on another visual,
// since the announcement is the window's while the counter is the
// client's.
func (s *x11Surface) setupSyncLocked() {
	c := s.conn
	if s.popup || c == nil || c.dpy == nil || s.win == 0 || !c.syncReadyLocked() {
		return
	}
	if s.sync.counter == 0 {
		s.sync.counter = C.ui_sync_counter(c.dpy)
	}
	if s.sync.counter != 0 {
		C.ui_sync_announce(c.dpy, s.win, s.sync.counter, c.atomDelete, c.atomSyncReq, c.atomSyncCounter)
	}
}

// syncMessageLocked takes a _NET_WM_SYNC_REQUEST client message; false for
// any other event.
func (s *x11Surface) syncMessageLocked(xe *C.XEvent) bool {
	c := s.conn
	if s.sync.counter == 0 || c.syncState <= 0 {
		return false
	}
	var lo, hi C.uint
	if C.ui_sync_message(xe, c.atomProtocols, c.atomSyncReq, &lo, &hi) == 0 {
		return false
	}
	s.sync.req.request(syncValue(uint32(lo), uint32(hi)), time.Now())
	return true
}

// syncConfiguredLocked follows a ConfigureNotify: one that left the size
// alone is answered at once.
func (s *x11Surface) syncConfiguredLocked(resized bool) {
	if ok, v := s.sync.req.configured(resized, time.Now()); ok {
		s.syncAnswerLocked(v)
	}
}

// syncPresentedLocked follows a frame put on the window.
func (s *x11Surface) syncPresentedLocked() {
	if ok, v := s.sync.req.presented(); ok {
		s.syncAnswerLocked(v)
	}
}

// syncOverdueLocked answers every request that has waited too long.
func (c *x11Conn) syncOverdueLocked() {
	now := time.Now()
	for _, s := range c.surfaces {
		if s == nil || s.sync.counter == 0 {
			continue
		}
		if ok, v := s.sync.req.overdue(now, syncOverdue); ok {
			s.syncAnswerLocked(v)
		}
	}
}

// syncDeadlineLocked is when the window's waiting request becomes overdue.
func (s *x11Surface) syncDeadlineLocked() time.Time {
	return s.sync.req.deadline(syncOverdue)
}

func (s *x11Surface) syncAnswerLocked(v uint64) {
	if s.conn == nil || s.conn.dpy == nil || s.sync.counter == 0 {
		return
	}
	C.ui_sync_set(s.conn.dpy, s.sync.counter, C.uint(uint32(v)), C.uint(uint32(v>>32)))
}

// closeSyncLocked destroys the counter with the window.
func (s *x11Surface) closeSyncLocked() {
	if s.conn != nil && s.conn.dpy != nil {
		C.ui_sync_destroy(s.conn.dpy, s.sync.counter)
	}
	s.sync = x11Sync{}
}
