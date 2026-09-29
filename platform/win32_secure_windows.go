//go:build windows

package platform

// Keeping a window out of screenshots on Windows.
//
// SetWindowDisplayAffinity with WDA_EXCLUDEFROMCAPTURE tells the
// compositor to composite the window to the screen and leave it out of
// every capture of that screen: PrintScreen, the Snipping Tool, a
// meeting's screen share, a remote desktop. It is what a password
// manager's window uses, and it is enforced by the window server rather
// than asked of the capturing application.
//
// WDA_MONITOR (1) is the older value, which blanks the window in a
// capture; EXCLUDEFROMCAPTURE (0x11) leaves it out altogether and is
// Windows 10 2004 and later. Setting it fails on older builds, which is
// reported rather than hidden.
const (
	wdaNone              = 0x00
	wdaExcludeFromCature = 0x11
)

var procSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")

// SetExcludeFromCapture keeps this window out of screenshots and screen
// shares ([CaptureExcludeSurface]).
func (s *winSurface) SetExcludeFromCapture(on bool) bool {
	if s == nil || s.hwnd == 0 || s.closed {
		return false
	}
	affinity := uintptr(wdaNone)
	if on {
		affinity = wdaExcludeFromCature
	}
	r, _, _ := procSetWindowDisplayAffinity.Call(s.hwnd, affinity)
	return r != 0
}

// Windows has no SecureInputSurface: there is no counterpart to macOS's
// EnableSecureEventInput that an ordinary application may use, and the
// low-level hook API that would be needed to take the keyboard is the
// thing a key logger uses rather than the thing that stops one.
var _ CaptureExcludeSurface = (*winSurface)(nil)
