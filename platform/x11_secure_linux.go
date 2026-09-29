//go:build linux && cgo

package platform

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>

// ui_grab_kbd takes the keyboard for w, owner_events True so the
// application's own windows still hear their own keys. It reports 1 only
// on GrabSuccess; the other four results are all "somebody else has it",
// which the caller retries.
static int ui_grab_kbd(Display* d, Window w) {
	int r = XGrabKeyboard(d, w, True, GrabModeAsync, GrabModeAsync, CurrentTime);
	XFlush(d);
	return r == GrabSuccess;
}

static void ui_ungrab_kbd(Display* d) {
	XUngrabKeyboard(d, CurrentTime);
	XFlush(d);
}
*/
import "C"

import "time"

// SetSecureInput grabs the keyboard for this window ([SecureInputSurface]).
//
// On X11 a grab stops the server delivering key events to any other
// client — which is what `xinput test` from another terminal stops
// seeing — and that is the whole of what X11 offers. It does not stop a
// client that has opened the evdev device directly, and it cannot take
// the keyboard from a client that already holds a grab of its own: a
// menu is open somewhere, a window manager is mid-gesture. So a failure
// is ordinary rather than exceptional, and it is retried for a moment
// before being reported, because the usual cause is a menu that is about
// to close.
//
// The grab is released when the window loses focus and retaken when it
// comes back, because a grab held by an unfocused window would take the
// keyboard away from the window the user is actually typing in.
func (s *x11Surface) SetSecureInput(on bool) bool {
	if s == nil || s.conn == nil || s.Closed() {
		return false
	}
	if !on {
		x11Mu.Lock()
		defer x11Mu.Unlock()
		s.secureWanted = false
		s.ungrabLocked()
		return true
	}
	// Somebody else holding the keyboard is nearly always a menu on its
	// way down, and a prompt that gave up on the first attempt would
	// fail exactly when it was opened from one. The lock is dropped
	// between tries so the event loop can run — which is what lets that
	// menu finish closing.
	deadline := time.Now().Add(250 * time.Millisecond)
	for {
		x11Mu.Lock()
		s.secureWanted = true
		ok := s.grabLocked()
		x11Mu.Unlock()
		if ok || time.Now().After(deadline) {
			return ok
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *x11Surface) grabLocked() bool {
	if s.secureHeld {
		return true
	}
	if s.conn == nil || s.conn.dpy == nil {
		return false
	}
	s.secureHeld = C.ui_grab_kbd(s.conn.dpy, s.win) != 0
	return s.secureHeld
}

func (s *x11Surface) ungrabLocked() {
	if !s.secureHeld {
		return
	}
	if s.conn != nil && s.conn.dpy != nil {
		C.ui_ungrab_kbd(s.conn.dpy)
	}
	s.secureHeld = false
}

// secureFocusChangedLocked re-takes or drops the grab as the window's
// focus changes, so a window that asked for secure input holds the
// keyboard only while it is the one being typed into — a grab held by an
// unfocused window would take the keys from the window the user is
// actually in.
//
// One attempt, no retry: this runs inside the event loop with the
// display lock held, and a loop that slept here would stop the
// compositor's events being read, which is the thing that would let the
// grab succeed.
func (s *x11Surface) secureFocusChangedLocked(focused bool) {
	if s == nil || !s.secureWanted {
		return
	}
	if focused {
		s.grabLocked()
		return
	}
	s.ungrabLocked()
}
