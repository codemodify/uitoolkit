//go:build linux && cgo

package platform

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/XKBlib.h>

// ui_lock_state is XkbGetIndicatorState, which answers at any moment
// rather than waiting for a key: bit 0 is Caps Lock and bit 1 Num Lock,
// which is the order every XKB keyboard maps them in.
static unsigned int ui_lock_state(Display* d) {
	unsigned int st = 0;
	if (!d) return 0;
	if (XkbGetIndicatorState(d, XkbUseCoreKbd, &st) != Success) return 0;
	return st;
}
*/
import "C"

// LockKeys is the lock keys' state now ([LockKeysSurface]).
//
// X11 can be asked at any moment, so a prompt can warn about Caps Lock
// before the first keystroke rather than after the first refusal.
func (s *x11Surface) LockKeys() (caps, num bool) {
	if s == nil || s.conn == nil || s.conn.dpy == nil {
		return false, false
	}
	x11Mu.Lock()
	st := uint32(C.ui_lock_state(s.conn.dpy))
	x11Mu.Unlock()
	return st&1 != 0, st&2 != 0
}
