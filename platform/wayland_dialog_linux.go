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

// uitk_wl_set_parent says which toplevel this one belongs to. A NULL parent
// takes the relationship away, which the protocol spells out.
static void uitk_wl_set_parent(struct xdg_toplevel *t, struct xdg_toplevel *parent) {
	if (t) xdg_toplevel_set_parent(t, parent);
}
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
	// A utility window is an ordinary toplevel here. xdg-shell describes a
	// toplevel, a dialog and a popup and nothing else, and there is no
	// protocol an ordinary client may use to ask to be left out of a task
	// bar — that is a privileged shell's business on Wayland. What a
	// satellite panel *can* have is its parent (SetOwner), which keeps it
	// with the window it belongs to; FrameSkipTaskbar is absent so an
	// application can say the rest is not possible rather than believe it
	// worked. See [RoleUtility].
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

// SetOwner says which toplevel this one belongs to ([OwnedSurface]), with
// xdg_toplevel.set_parent.
//
// It is the one half of a satellite panel's policy Wayland does offer: the
// compositor stacks the child above its parent and keeps the two together.
// The other half — staying out of the task bar — has no protocol for an
// ordinary client, and [FrameSkipTaskbar] is absent here because of it.
func (s *wlSurface) SetOwner(owner Surface) bool {
	if s == nil || s.conn == nil {
		return false
	}
	wlMu.Lock()
	defer wlMu.Unlock()
	if s.top == nil {
		return false
	}
	if owner == nil {
		s.owner = nil
		C.uitk_wl_set_parent(s.top, nil)
		return true
	}
	o, ok := owner.(*wlSurface)
	if !ok || o == s || o.top == nil || o.conn != s.conn {
		// Another backend's window, this window itself, or one on another
		// connection: none of them is a parent this toplevel can have.
		return false
	}
	s.owner = o
	C.uitk_wl_set_parent(s.top, o.top)
	return true
}
