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
	// RoleUtility is a satellite panel that belongs to a primary window
	// rather than being a window in its own right: a player's equaliser and
	// playlist, a tool palette, a floating inspector. It keeps a close
	// control and loses its minimize and maximize ones, the desktop keeps it
	// with its owner rather than listing it beside one, and it gets no task
	// bar entry of its own.
	//
	// Per platform: `_NET_WM_WINDOW_TYPE_UTILITY` with
	// `_NET_WM_STATE_SKIP_TASKBAR` and `_NET_WM_STATE_SKIP_PAGER` on X11;
	// `WS_EX_TOOLWINDOW` on an owned window on Win32, which is what keeps it
	// off the task bar there; an `NSPanel` with
	// `NSWindowStyleMaskUtilityWindow` on macOS, where there is no per-window
	// task bar entry to skip in the first place.
	//
	// **Wayland has no utility toplevel.** xdg-shell describes a toplevel, a
	// dialog and a popup, and there is no protocol a client may use to ask to
	// be left out of a task bar — that is a privileged shell's business
	// there. A utility window is an ordinary toplevel with its parent set,
	// which is as much as the protocol allows, and [FrameSkipTaskbar] is
	// absent so an application can say so in its own interface rather than
	// believing it worked.
	RoleUtility
)

// SkipsTaskbar reports whether this role asks to be left out of the
// desktop's window list by its nature, before any explicit
// [WindowOptions.SkipTaskbar].
func (r WindowRole) SkipsTaskbar() bool { return r == RoleUtility }

func (r WindowRole) String() string {
	switch r {
	case RoleDialog:
		return "dialog"
	case RoleUtility:
		return "utility"
	}
	return "normal"
}

// RoleSurface is a backend that can tell the desktop what kind of window
// this is. A backend that cannot leaves the window ordinary.
type RoleSurface interface {
	SetWindowRole(r WindowRole) bool
}

// OwnedSurface is a backend that can say which window this one belongs to: a
// satellite panel's primary window, a dialog's parent.
//
// The desktop then keeps the two together — the owned window stays above its
// owner, is raised and minimized with it, and is placed over it rather than
// cascaded — which is the half of [RoleUtility] and [RoleDialog] that a role
// alone cannot say, because a role says what *kind* of window this is and not
// whose.
//
// A nil owner takes the relationship away. A window with none is a window of
// its own, which is what every window is until it says otherwise. A backend
// that cannot do it answers false and leaves [FrameOwner] out of its
// capabilities.
type OwnedSurface interface {
	SetOwner(owner Surface) bool
}

// TaskbarSurface is a backend that can keep a window out of the desktop's
// window list and its workspace switcher, whatever the window's role.
//
// [RoleUtility] asks for this by its nature; this is for the window that wants
// it on its own — a splash screen, a dock, a notification of the
// application's own — or that wants it back. A backend answers false where the
// desktop has no such notion, and leaves [FrameSkipTaskbar] out.
type TaskbarSurface interface {
	SetSkipTaskbar(skip bool) bool
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
