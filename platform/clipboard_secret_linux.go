//go:build linux && cgo

package platform

import "os"

// The Linux secret clipboard.
//
// Both protocols work the same way here and differently from Windows and
// macOS: the *owner* serves a selection on request, so this process goes
// on holding the value for as long as it owns it. What the secret path
// changes is which selections are taken, what the offer says about
// itself, and that the value is dropped when the timeout fires rather
// than at the next copy.
//
//   - **CLIPBOARD only.** PRIMARY is pasted by a middle click anywhere
//     on the desktop, with no Ctrl+V and no intent, and nothing undoes
//     it. A secret copy takes CLIPBOARD and gives PRIMARY up.
//   - **x-kde-passwordManagerHint**, as an X11 target whose value is
//     "secret" and as a Wayland mime type. Klipper reads it, and so do
//     the managers that follow KDE's lead; the ones that do not see an
//     ordinary text offer, which is why the timeout is the thing
//     actually relied on rather than the hint.
//
// The value is carried as bytes the whole way down. Both owners keep a
// copy of their own while they serve the selection — that is what owning
// one means — and both zero it when the selection is given up, cleared or
// replaced. Nothing on this path makes a Go string of a secret, which
// could not be zeroed and would sit in the heap until the collector got
// to it.
func clipboardNativeSetSecret(b []byte) {
	if waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe()) {
		wlClipSetSecret(b)
	}
	if x11Live() || os.Getenv("DISPLAY") != "" {
		if c, err := x11Get(); err == nil {
			c.setClipboardSelBytes(b, true)
		}
	}
}

func clipboardNativeClear() {
	if waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe()) {
		wlClipSet("")
	}
	if x11Live() || os.Getenv("DISPLAY") != "" {
		if c, err := x11Get(); err == nil {
			c.clearClipboard()
		}
	}
}

// clipboardNativeGetSecret reads the clipboard as bytes, so a secret put
// there by another program — a passphrase copied out of a terminal or
// another password manager — never becomes a Go string in this process.
//
// It is the read half of what clipboardNativeSetSecret does for the write
// half. Both owners hand the bytes over rather than copying them, and
// zero what they held.
func clipboardNativeGetSecret() ([]byte, bool) {
	if waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe()) {
		if b, ok := wlClipGetBytes(false); ok {
			return b, true
		}
	}
	if x11Live() || os.Getenv("DISPLAY") != "" {
		if b, ok := x11ClipGetBytes(false); ok {
			return b, true
		}
	}
	return nil, false
}
