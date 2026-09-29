package platform

// WindowRole is what kind of window this is, as far as the desktop is
// concerned. It is a hint the window manager reads, not a behaviour the
// toolkit implements: what changes is where the window opens, whether it
// gets a taskbar entry, and what its frame looks like.
type WindowRole uint8

const (
	// RoleNormal — the zero value — is an ordinary application window.
	RoleNormal WindowRole = iota
	// RoleDialog is a dialog: a passphrase prompt, an alert, a
	// preferences sheet. Desktops treat one differently in ways a client
	// cannot fake — it is centred on its parent or its monitor rather
	// than cascaded, it may be kept off the taskbar and above its
	// parent, and its frame is a dialog's.
	//
	// Per platform: `_NET_WM_WINDOW_TYPE_DIALOG` on X11, `xdg_dialog_v1`
	// on Wayland where the compositor offers it (and an ordinary
	// toplevel where it does not — every compositor already centres what
	// it thinks is a dialog), an owned window without minimize or
	// maximize on Win32, and an `NSPanel` on macOS.
	RoleDialog
)

// RoleSurface is a backend that can tell the desktop what kind of window
// this is. A backend that cannot leaves the window ordinary.
type RoleSurface interface {
	SetWindowRole(r WindowRole) bool
}

// ActivateSurface is a backend that can ask the desktop to bring this
// window to the front and give it the keyboard.
//
// It is what [Raise] has always done on X11 and Wayland, named for what
// a caller wants rather than for the X11 request, and separated because
// "raise" and "activate" are different things on Windows: raising is
// z-order, activating is focus, and a prompt needs the second.
type ActivateSurface interface {
	Activate() bool
}

// CenterSurface is a backend that can put a window in the middle of the
// monitor it is opening on.
//
// Wayland deliberately has no implementation: a client cannot place a
// toplevel there, and every compositor already centres a window it has
// been told is a dialog — which is the whole reason [RoleDialog] carries
// its weight there and [WindowOptions.Center] does not.
type CenterSurface interface {
	Center() bool
}
