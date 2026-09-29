//go:build darwin

package platform

// EnableSecureEventInput, which is the real thing: the keyboard is taken
// away from every other process while it is on, and the menu bar shows
// the indicator that says so.
func secureInputAvailable() bool { return true }

// NSWindow.sharingType = .none, which the window server honours for
// screenshots, screen recordings and screen sharing.
func captureExclusionAvailable() bool { return true }
