//go:build linux && cgo

package platform

/*
#cgo pkg-config: wayland-client
#include <wayland-client.h>
#include "keyboard-shortcuts-inhibit-unstable-v1-client-protocol.h"

static struct zwp_keyboard_shortcuts_inhibitor_v1 *uitk_wl_inhibit(
		struct zwp_keyboard_shortcuts_inhibit_manager_v1 *m,
		struct wl_surface *surf, struct wl_seat *seat) {
	if (!m || !surf || !seat) return NULL;
	return zwp_keyboard_shortcuts_inhibit_manager_v1_inhibit_shortcuts(m, surf, seat);
}

static void uitk_wl_inhibitor_destroy(struct zwp_keyboard_shortcuts_inhibitor_v1 *i) {
	if (i) zwp_keyboard_shortcuts_inhibitor_v1_destroy(i);
}

static void uitk_wl_flush(struct wl_display *d) { if (d) wl_display_flush(d); }
*/
import "C"

// SetSecureInput inhibits the compositor's keyboard shortcuts for this
// surface ([SecureInputSurface]).
//
// This is a different thing from the X11 grab it shares a name with, and
// the difference is worth being clear about. On Wayland a client cannot
// see another client's keys at all — the compositor routes them to the
// focused surface and nowhere else — so there is nothing here for a
// grab to protect against. What is left is the compositor's own
// shortcuts, which do take keys before the surface sees them, and
// zwp_keyboard_shortcuts_inhibit_manager_v1 is the protocol for asking
// it to stop while a passphrase is being typed.
//
// The compositor may refuse, and GNOME's asks the user the first time.
// A false here means the protocol is not offered at all; a granted
// inhibitor that the compositor later deactivates is its business, and
// the keys still reach the surface either way.
func (s *wlSurface) SetSecureInput(on bool) bool {
	if s == nil || s.conn == nil {
		return false
	}
	wlMu.Lock()
	defer wlMu.Unlock()
	if !on {
		if s.inhibitor != nil {
			C.uitk_wl_inhibitor_destroy(s.inhibitor)
			s.inhibitor = nil
			C.uitk_wl_flush(s.conn.dpy)
		}
		return true
	}
	if s.conn.shortcutMan == nil || s.surf == nil || s.conn.seat == nil {
		return false
	}
	if s.inhibitor != nil {
		// Asking twice for the same surface and seat is a protocol
		// error ("already_inhibited"), not a no-op.
		return true
	}
	s.inhibitor = C.uitk_wl_inhibit(s.conn.shortcutMan, s.surf, s.conn.seat)
	if s.inhibitor == nil {
		return false
	}
	C.uitk_wl_flush(s.conn.dpy)
	return true
}
