//go:build linux && cgo

package platform

/*
#cgo pkg-config: wayland-client
#include <wayland-client.h>
#include "xdg-shell-client-protocol.h"
#include "xdg-dialog-v1-client-protocol.h"

static struct xdg_dialog_v1 *uitk_wl_dialog(struct xdg_wm_dialog_v1 *m, struct xdg_toplevel *t) {
	if (!m || !t) return NULL;
	return xdg_wm_dialog_v1_get_xdg_dialog(m, t);
}

static void uitk_wl_dialog_destroy(struct xdg_dialog_v1 *d) { if (d) xdg_dialog_v1_destroy(d); }
*/
import "C"

// SetWindowRole tells the compositor this toplevel is a dialog
// ([RoleSurface]), through xdg_dialog_v1 where it is offered.
//
// Wayland has no window-type property and no way for a client to place a
// toplevel, so this *is* the centring: a compositor told a window is a
// dialog puts it in the middle of its parent, which is why
// [WindowOptions.Center] is documented as ignored here rather than as
// missing. A compositor that does not offer the protocol gets an
// ordinary toplevel, which is the fallback working — most of them
// already centre a dialog-shaped window on its parent.
//
// The dialog object is deliberately not made modal: modality is the
// application's business, the toolkit draws its own modal overlays, and
// a compositor-enforced modal that outlived a crashed dialog would lock
// the parent window with nothing to unlock it.
func (s *wlSurface) SetWindowRole(r WindowRole) bool {
	if s == nil || s.conn == nil {
		return false
	}
	wlMu.Lock()
	defer wlMu.Unlock()
	s.role = r
	if r != RoleDialog {
		if s.dialog != nil {
			C.uitk_wl_dialog_destroy(s.dialog)
			s.dialog = nil
		}
		return true
	}
	if s.dialog != nil {
		return true
	}
	if s.conn.dialogMan == nil || s.top == nil {
		return false
	}
	s.dialog = C.uitk_wl_dialog(s.conn.dialogMan, s.top)
	return s.dialog != nil
}

// Activate asks the compositor to bring this window to the front and
// give it the keyboard ([ActivateSurface]), through xdg_activation_v1.
//
// A compositor may refuse — that is the point of the protocol, and what
// stops a background window stealing the focus — so a true here means
// the request went out, not that the window is in front.
func (s *wlSurface) Activate() bool {
	if s == nil || s.conn == nil || s.conn.activation == nil {
		return false
	}
	s.requestActivate()
	return true
}
