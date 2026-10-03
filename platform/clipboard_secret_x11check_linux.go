//go:build linux && cgo

package platform

// x11StillServesTheSecret reports whether this process's X11 owner still
// holds CLIPBOARD with a secret on it.
//
// It exists for one case, and it is the case most Linux desktops are in.
// A secret copy takes X11's CLIPBOARD *and* the Wayland selection,
// because XWayland is running and DISPLAY is set — GNOME and KDE both do
// this by default. A compositor that carries an X11 owner's copy over to
// Wayland then sets the Wayland selection to it, which cancels this
// program's own Wayland source moments after the copy.
//
// That cancel does not mean the secret has left this program. The X11
// owner is still serving it, to every paste, through the compositor's
// bridge as well. Ending the held copy there — wiping it, stopping its
// timer and saying it is gone — leaves the passphrase being served with
// nothing left to clear it: not the timeout, which is stopped, nor the
// quit, which finds nothing held. It stays until something else is
// copied or the program ends, which for a password manager in the tray
// can be days.
func x11StillServesTheSecret() bool {
	if !x11Live() {
		return false
	}
	c, err := x11Get()
	if err != nil || c == nil {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return c.ownClip && c.clipSecret
}

// wlStillServesTheSecret is the other direction: whether this process's
// Wayland data source still holds the selection with a secret on it.
//
// Its caller is X11's SelectionClear. On a Wayland desktop a secret copy
// takes both selections, and the compositor's own bridge is what takes X11's
// CLIPBOARD away from this process moments later — the same handover seen
// from the other side. Ending the held copy on that word would stop the
// timer, leaving the Wayland selection serving the passphrase with nothing
// left to clear it.
func wlStillServesTheSecret() bool {
	wlMu.Lock()
	defer wlMu.Unlock()
	return wlc != nil && wlc.dpy != nil && wlc.dataSrc != nil && wlc.clipSecret
}
