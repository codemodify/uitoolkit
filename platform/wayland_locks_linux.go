//go:build linux && cgo

package platform

// The lock keys on Wayland.
//
// A Wayland client cannot query the keyboard the way X11's
// XkbGetIndicatorState does — but it does not have to. The compositor
// sends wl_keyboard.modifiers immediately after wl_keyboard.enter, before
// any key: wayland.xml asks it of the compositor, and Mutter, KWin and
// wlroots all do it. The backend already folds that into its xkb state
// and c.mods (uitkWlKeyMods); until now it stopped there and told nobody,
// so a passphrase prompt opened with Caps Lock on said nothing of it
// until the first key — by which time the passphrase it was there to
// warn about had been typed.
//
// This was reported as a limit of the protocol in v0.23.2's notes. It is
// not; the state was in hand the whole time.

// LockKeys is the lock state as of the last modifiers event the
// compositor sent, which arrives with the focus rather than with the
// first key ([LockKeysSurface]).
func (s *wlSurface) LockKeys() (caps, num bool) {
	if s == nil || s.conn == nil {
		return false, false
	}
	wlMu.Lock()
	m := s.conn.mods
	wlMu.Unlock()
	return m.CapsLock(), m.NumLock()
}
