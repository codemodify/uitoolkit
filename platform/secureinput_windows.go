//go:build windows

package platform

// Windows has no secure input a normal application may ask for. There is
// no counterpart to EnableSecureEventInput, and the low-level hook API
// that would be needed to take the keyboard is the thing a key logger
// uses rather than the thing that stops one.
func secureInputAvailable() bool { return false }

// SetWindowDisplayAffinity with WDA_EXCLUDEFROMCAPTURE, which the
// compositor honours: the window is composited to the screen and left
// out of every capture of it.
func captureExclusionAvailable() bool { return true }
