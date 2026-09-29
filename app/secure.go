package app

import "github.com/codemodify/uitoolkit/platform"

// Two things a passphrase prompt asks the desktop for, both optional and
// both meaning something a little different on each platform. Each
// reports what actually happened rather than what was asked, and each
// has an Available so an application can decide whether to make the
// promise at all.

// SetSecureInput asks the desktop to keep this window's keystrokes to
// itself, and reports whether it did.
//
// What that buys is not the same everywhere, and a prompt that tells the
// user "your keystrokes are protected" should know which it has:
//
//   - **macOS** takes the keyboard from every other process, including
//     the ones with accessibility permission, and lights the menu-bar
//     indicator.
//   - **X11** grabs the keyboard, so the server stops delivering the
//     keys to other clients. A client that has already opened the input
//     device still sees them, and a client that holds its own grab —
//     an open menu, a window manager mid-gesture — makes the attempt
//     fail; it is retried for a quarter of a second first, because the
//     usual cause is a menu on its way down.
//   - **Wayland** inhibits the compositor's own shortcuts, which is the
//     only thing there is to ask for: a client cannot see another
//     client's keys to begin with. The compositor may refuse, and
//     GNOME's asks the user the first time.
//   - **Windows** has nothing an ordinary application may use, and says
//     so through [platform.SecureInputAvailable].
//
// The X11 grab follows the focus: it is dropped when the window loses
// it and retaken when it comes back, because a grab held by an unfocused
// window would take the keyboard from the window being typed in. Turn it
// off when the prompt is done; closing the window does too.
func (w *Window) SetSecureInput(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.SecureInputSurface)
	if !ok {
		return false
	}
	return s.SetSecureInput(on)
}

// SetExcludeFromCapture asks the desktop to leave this window out of
// screenshots, screen recordings and screen shares, and reports whether
// it did.
//
// Windows and macOS both enforce it at the compositor, so the capture
// gets the desktop without the window rather than a cooperating
// screenshot tool's idea of it. X11 and Wayland have no such thing — on
// X11 any client can read the root window, and on Wayland the screencast
// portal is the compositor's own business — so both answer false, and
// [platform.CaptureExclusionAvailable] says so in advance.
func (w *Window) SetExcludeFromCapture(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.CaptureExcludeSurface)
	if !ok {
		return false
	}
	return s.SetExcludeFromCapture(on)
}
