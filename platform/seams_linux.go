//go:build linux && cgo

package platform

// The seams each Linux backend implements, asserted.
//
// They were not, and the gap was not harmless: the feature matrix in
// docs/platform.md was maintained by hand, and "does this backend still
// satisfy DropReceiver" was a question nothing but a careful reading
// answered. A method whose signature drifted would have failed at the
// call site in the app package, or — for a seam found by type assertion
// — not failed at all, and the capability would have quietly switched
// itself off.
//
// One line each, and the compiler is the matrix.
var (
	_ WindowFrame    = (*wlSurface)(nil)
	_ WindowGeometry = (*wlSurface)(nil)
	_ PopupSurface   = (*wlSurface)(nil)
	_ PopupOpener    = (*wlSurface)(nil)
	_ DragSurface    = (*wlSurface)(nil)
	_ DropNegotiator = (*wlSurface)(nil)
	_ DropReceiver   = (*wlSurface)(nil)
	_ IMESurface     = (*wlSurface)(nil)
	// Wayland inhibits the compositor's shortcuts; it cannot keep a
	// window out of a capture, and does not claim to.
	_ SecureInputSurface = (*wlSurface)(nil)
	_ RoleSurface        = (*wlSurface)(nil)
	_ ActivateSurface    = (*wlSurface)(nil)
	// No CenterSurface: a client cannot place a toplevel, and a
	// compositor told the window is a dialog centres it itself.

	_ WindowFrame    = (*x11Surface)(nil)
	_ WindowGeometry = (*x11Surface)(nil)
	_ PopupSurface   = (*x11Surface)(nil)
	_ PopupOpener    = (*x11Surface)(nil)
	_ DragSurface    = (*x11Surface)(nil)
	_ DropNegotiator = (*x11Surface)(nil)
	_ DropReceiver   = (*x11Surface)(nil)
	_ IMESurface     = (*x11Surface)(nil)
	// X11 grabs the keyboard; the root window is readable by any
	// client, so there is no capture exclusion to implement.
	_ SecureInputSurface = (*x11Surface)(nil)
	_ RoleSurface        = (*x11Surface)(nil)
	_ ActivateSurface    = (*x11Surface)(nil)
	_ CenterSurface      = (*x11Surface)(nil)
	// X11 can be asked for the lock keys at any moment; Wayland learns
	// them from the compositor and has nothing to query.
	_ LockKeysSurface = (*x11Surface)(nil)
)
