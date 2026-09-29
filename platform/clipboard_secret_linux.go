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
// One honest limit: the X11 owner serves from a Go string, so the copy
// this process holds while it owns the selection cannot be zeroed. It is
// dropped on the timeout, on a clear and when the selection is lost —
// which is as much as a garbage-collected language allows without
// rewriting the selection service around a byte slice.
func clipboardNativeSetSecret(b []byte) {
	s := string(b)
	if waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe()) {
		wlClipSetSecret(s)
	}
	if x11Live() || os.Getenv("DISPLAY") != "" {
		if c, err := x11Get(); err == nil {
			c.setClipboardSel(s, true)
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
