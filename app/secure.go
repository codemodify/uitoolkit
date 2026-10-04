package app

import "github.com/codemodify/uitoolkit/platform"

// Two things a passphrase prompt asks the desktop for, both optional and
// both meaning something a little different on each platform. Each
// reports what actually happened rather than what was asked, and each
// has an Available so an application can decide whether to make the
// promise at all.

// SetSecureInput asks the desktop to keep this window's keystrokes to
// itself, and reports whether it did.
//
// What that buys is not the same everywhere, and a prompt that tells the
// user "your keystrokes are protected" should know which it has:
//
//   - **macOS** takes the keyboard from every other process, including
//     the ones with accessibility permission, and lights the menu-bar
//     indicator.
//   - **X11** grabs the keyboard, so the server stops delivering the
//     keys to other clients. A client that has already opened the input
//     device still sees them, and a client that holds its own grab —
//     an open menu, a window manager mid-gesture — makes the attempt
//     fail; it is retried for a quarter of a second first, because the
//     usual cause is a menu on its way down.
//   - **Wayland** inhibits the compositor's own shortcuts, which is the
//     only thing there is to ask for: a client cannot see another
//     client's keys to begin with. The compositor may refuse, and
//     GNOME's asks the user the first time.
//   - **Windows** has nothing an ordinary application may use, and says
//     so through [platform.SecureInputAvailable].
//
// The X11 grab follows the focus: it is dropped when the window loses
// it and retaken when it comes back, because a grab held by an unfocused
// window would take the keyboard from the window being typed in. Turn it
// off when the prompt is done; closing the window does too.
func (w *Window) SetSecureInput(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.SecureInputSurface)
	if !ok {
		return false
	}
	w.secureWanted = on
	w.secureGot = s.SetSecureInput(on)
	w.noteSecureInput(on && w.SecureInputHeld())
	return w.secureGot
}

// SetExcludeFromCapture asks the desktop to leave this window out of
// screenshots, screen recordings and screen shares, and reports whether
// it did.
//
// Windows and macOS both enforce it at the compositor, so the capture
// gets the desktop without the window rather than a cooperating
// screenshot tool's idea of it. X11 and Wayland have no such thing — on
// X11 any client can read the root window, and on Wayland the screencast
// portal is the compositor's own business — so both answer false, and
// [platform.CaptureExclusionAvailable] says so in advance.
func (w *Window) SetExcludeFromCapture(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.CaptureExcludeSurface)
	if !ok {
		return false
	}
	return s.SetExcludeFromCapture(on)
}

// Activate brings the window to the front and gives it the keyboard.
//
// It is what a prompt does when it opens, and what an application does
// when a second copy of it is started and hands the request to the one
// already running. [Window.Raise] is the same thing on X11 and Wayland,
// named for the X11 request; this one is named for what the caller
// wants, and on Windows the two are genuinely different — raising is
// z-order and activating is focus.
//
// It reports whether the desktop was asked, not whether it agreed. Every
// modern desktop refuses focus to a window whose application is not
// already in front, which is the rule that stops a background window
// stealing the keyboard mid-sentence; what happens instead is the
// taskbar entry flashes, or the compositor marks the window urgent.
func (w *Window) Activate() bool {
	if w == nil || w.Closed() {
		return false
	}
	if a, ok := w.surf.(platform.ActivateSurface); ok {
		return a.Activate()
	}
	platform.RaiseSurface(w.surf)
	return true
}

// SetWindowRole tells the desktop what kind of window this is —
// ordinary, a dialog, or a satellite panel ([platform.WindowRole]).
//
// [platform.WindowOptions.Role] is the better place to say it, because a
// window manager reads the type when it takes the window over: a role
// set afterwards may not move a window it has already placed. This is
// for a window that becomes a dialog later.
func (w *Window) SetWindowRole(r platform.WindowRole) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.RoleSurface)
	if !ok {
		return false
	}
	return s.SetWindowRole(r)
}

// SetOwner says which window this one belongs to: a satellite panel's primary
// window, a dialog's parent ([platform.OwnedSurface]). A nil owner takes the
// relationship away. It reports whether the desktop will keep the two
// together.
//
// A role says what *kind* of window this is and an owner says whose, and a
// satellite panel needs both: the role is what keeps it off the task bar, the
// owner is what keeps it above the window it belongs to, raised and minimized
// with it, and placed over it rather than cascaded.
//
// [platform.WindowOptions.Owner] is the better place to say it where it can
// be: X11's WM_TRANSIENT_FOR and a window manager's placement are both read
// when the window is mapped, so an owner given afterwards is an owner the
// manager may already have placed the window without.
//
// False means the desktop has no such notion —
// [platform.FrameSkipTaskbar] and [platform.FrameOwner] are what to ask
// beforehand — and the window is simply a window of its own.
func (w *Window) SetOwner(owner *Window) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.OwnedSurface)
	if !ok {
		return false
	}
	if owner == nil {
		return s.SetOwner(nil)
	}
	if owner == w || owner.Closed() {
		return false
	}
	return s.SetOwner(owner.surf)
}

// SetSkipTaskbar keeps the window out of the desktop's window list and its
// workspace switcher, and reports whether the desktop can
// ([platform.TaskbarSurface]).
//
// [platform.WindowRole] RoleUtility asks for this by its nature; this is for
// the window that wants it on its own — a splash screen, a dock, a
// notification of the application's own — and for the one that wants it back.
//
// **It cannot be done on Wayland**, where no protocol lets an ordinary client
// ask: false there, and an application that minds should say so in its own
// interface rather than leave the user wondering. Ask
// [platform.FrameSkipTaskbar] of [Window.FrameCaps] to find out without
// changing anything.
func (w *Window) SetSkipTaskbar(skip bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.TaskbarSurface)
	if !ok {
		return false
	}
	return s.SetSkipTaskbar(skip)
}

// Center puts the window in the middle of the work area of the monitor
// it is on, and reports whether it could.
//
// Wayland cannot and answers false: a client may not place a toplevel
// there. A window opened with [platform.WindowRole] RoleDialog is
// centred by the compositor instead, which is the Wayland way of asking
// for the same thing.
func (w *Window) Center() bool {
	if w == nil || w.Closed() {
		return false
	}
	s, ok := w.surf.(platform.CenterSurface)
	if !ok {
		return false
	}
	if !s.Center() {
		return false
	}
	// Remembered so a later fit can put the window back in the middle
	// rather than growing it off-centre from its top-left corner.
	w.centred = true
	return true
}

// SecureInputHeld reports whether secure input is in effect now, which is
// not the same as what [Window.SetSecureInput] answered when it was asked.
//
// On X11 the keyboard grab follows the focus — dropped when the window
// loses it, retaken when it comes back — and a retake can fail, so a
// window that held the keyboard a moment ago may not hold it now. A
// prompt that tells a person their keystrokes are protected should say it
// from this, and say nothing while it is false.
//
// A backend that cannot be asked answers from what it said when it was
// asked, which is the best it has — an offscreen surface records the
// request and reports success, so it reads as held.
func (w *Window) SecureInputHeld() bool {
	if w == nil || w.Closed() || w.surf == nil {
		return false
	}
	if h, ok := w.surf.(platform.SecureInputHeldSurface); ok {
		return h.SecureInputHeld()
	}
	return w.secureWanted && w.secureGot
}

// OnSecureInput is called whenever secure input comes into effect or
// falls out of it — the X11 grab being dropped at a focus-out and retaken
// at the focus-in, or a retake that failed.
//
// It is one callback per window, the way OnLockKeys is: the window's own
// code, rather than a widget's.
func (w *Window) OnSecureInput(fn func(held bool)) {
	if w == nil {
		return
	}
	w.onSecureInput = fn
}

// noteSecureInput reports a change in what is actually in effect.
func (w *Window) noteSecureInput(held bool) {
	if w == nil || held == w.secureHeld {
		return
	}
	w.secureHeld = held
	if w.onSecureInput != nil {
		w.onSecureInput(held)
	}
}
