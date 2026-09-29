package platform

// Holding the keyboard, and staying out of screenshots.
//
// Both are things a passphrase prompt asks the desktop for, and both are
// answered differently on every platform and not at all on some. So both
// are optional seams: a window that implements them says what it managed
// to do, and an [Available] function tells an application whether to
// offer the promise at all, rather than making one it cannot keep.

// SecureInputSurface is a window that can stop other clients seeing the
// keys typed into it.
//
// What that means is not the same everywhere, and the difference
// matters:
//
//   - **macOS** has the real thing. EnableSecureEventInput takes the
//     keyboard away from every other process, including key loggers with
//     accessibility permission, and lights the menu-bar indicator.
//   - **X11** has XGrabKeyboard, which stops other *clients* being sent
//     the keys. It does not stop a client that has already opened the
//     device, and it does not survive a client that holds its own grab —
//     so it is a real improvement on nothing and not a guarantee.
//   - **Wayland** has zwp_keyboard_shortcuts_inhibit_manager_v1, which
//     stops the compositor eating keys as its own shortcuts. It is not a
//     secrecy mechanism at all: on Wayland a client cannot see another
//     client's keys to begin with, which is what makes the inhibitor the
//     right and the only thing to ask for.
//   - **Windows** has nothing a normal application may use, so it says
//     so.
type SecureInputSurface interface {
	// SetSecureInput turns it on or off and reports whether the platform
	// did anything. A false from an available platform means the attempt
	// failed — somebody else holds the grab — and the caller may retry
	// or tell the user.
	SetSecureInput(on bool) bool
}

// CaptureExcludeSurface is a window that can be left out of screenshots
// and screen shares.
//
// Windows and macOS both have it and it works at the compositor, so a
// capture gets a blank rectangle or nothing at all. X11 and Wayland have
// no such thing: on X11 any client can read the root window, and on
// Wayland the screencast portal is the compositor's own business and
// takes no such hint. Linux answers false rather than pretending.
type CaptureExcludeSurface interface {
	SetExcludeFromCapture(on bool) bool
}

// SecureInputAvailable reports whether this build can hold the keyboard
// for a window ([SecureInputSurface]). It says nothing about whether a
// particular window will succeed.
func SecureInputAvailable() bool { return secureInputAvailable() }

// CaptureExclusionAvailable reports whether this build can keep a window
// out of screenshots ([CaptureExcludeSurface]).
func CaptureExclusionAvailable() bool { return captureExclusionAvailable() }

// LockKeysSurface is a backend that can be asked for the lock keys'
// state without waiting for an event.
//
// It is what lets a prompt warn about Caps Lock *before* the first
// keystroke. X11 and Windows can both answer at any moment; Wayland
// cannot — a client learns the modifiers from the compositor and has
// nothing to query — and macOS answers from the current event, which is
// close enough that the window's own tracking covers it.
type LockKeysSurface interface {
	LockKeys() (caps, num bool)
}
