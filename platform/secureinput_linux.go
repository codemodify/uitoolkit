//go:build linux

package platform

import "os"

// On Linux "secure input" is two different things under one name, and
// the toolkit offers whichever the session has: an X11 keyboard grab, or
// the Wayland shortcuts inhibitor. Neither hides keys from a process
// that has already opened the input device, and the Wayland one is not
// about secrecy at all — see [SecureInputSurface].
func secureInputAvailable() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// No: X11 lets any client read the root window, and Wayland's screencast
// is the compositor's own and takes no hint from a client.
func captureExclusionAvailable() bool { return false }
