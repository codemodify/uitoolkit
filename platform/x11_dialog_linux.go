//go:build linux && cgo

package platform

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/Xatom.h>

static void ui_dlg_set_type(Display* d, Window w, Atom prop, Atom type) {
	XChangeProperty(d, w, prop, XA_ATOM, 32, PropModeReplace,
		(const unsigned char*)&type, 1);
	XFlush(d);
}

// ui_dlg_set_states writes the initial _NET_WM_STATE. A window manager
// reads it when it takes the window over, which is the only way to open
// *already* above the others, or maximized, rather than to be put there a
// frame later.
static void ui_dlg_set_states(Display* d, Window w, Atom prop, const Atom* states, int n) {
	XChangeProperty(d, w, prop, XA_ATOM, 32, PropModeReplace,
		(const unsigned char*)states, n);
	XFlush(d);
}

static void ui_dlg_set_transient(Display* d, Window w, Window parent) {
	XSetTransientForHint(d, w, parent);
	XFlush(d);
}

// ui_dlg_drop_transient removes WM_TRANSIENT_FOR, which is how a window
// stops belonging to another one. XSetTransientForHint with None would set
// the property to the None window, which is not the same as not having it.
static void ui_dlg_drop_transient(Display* d, Window w) {
	XDeleteProperty(d, w, XA_WM_TRANSIENT_FOR);
	XFlush(d);
}
*/
import "C"

// SetWindowRole tells the window manager what kind of window this is
// ([RoleSurface]).
//
// `_NET_WM_WINDOW_TYPE_DIALOG` is what makes a window manager centre the
// window on its parent rather than cascade it, give it a dialog's frame,
// and keep it above what it belongs to. The property is meant to be set
// before the window is mapped — which is where [WindowOptions.Role] sets
// it — and a manager that reads it only then is why changing it
// afterwards is documented as a hint rather than a command.
func (s *x11Surface) SetWindowRole(r WindowRole) bool {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return s.setWindowRoleLocked(r)
}

func (s *x11Surface) setWindowRoleLocked(r WindowRole) bool {
	c := s.conn
	if c.atomWinType == 0 {
		return false
	}
	t := c.atomWinTypeNormal
	switch r {
	case RoleDialog:
		t = c.atomWinTypeDialog
	case RoleUtility:
		t = c.atomWinTypeUtility
	}
	if t == 0 {
		return false
	}
	C.ui_dlg_set_type(c.dpy, s.win, c.atomWinType, t)
	was := s.role
	s.role = r
	if s.task.roleChanged(was, r) {
		s.applySkipTaskbarLocked()
	}
	return true
}

// SetOwner says which window this one belongs to ([OwnedSurface]).
//
// WM_TRANSIENT_FOR, which is how X11 has said it since ICCCM: the window
// manager keeps the two together, places the owned one over its owner rather
// than cascading it, and minimizes and raises them as one.
//
// A window manager reads it when it *maps* the window, which is why
// [WindowOptions.Owner] exists: an owner given afterwards is an owner the
// manager may already have placed the window without.
func (s *x11Surface) SetOwner(owner Surface) bool {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return s.setOwnerLocked(owner)
}

func (s *x11Surface) setOwnerLocked(owner Surface) bool {
	c := s.conn
	if owner == nil {
		s.owner = nil
		C.ui_dlg_drop_transient(c.dpy, s.win)
		return true
	}
	o, ok := owner.(*x11Surface)
	if !ok || o == s || o.win == 0 || o.conn != c {
		// Another backend's window, this window itself, or a window on
		// another display: none of them is an owner this one can have.
		return false
	}
	s.owner = o
	C.ui_dlg_set_transient(c.dpy, s.win, o.win)
	return true
}

// SetSkipTaskbar keeps the window out of the desktop's window list and its
// workspace switcher ([TaskbarSurface]).
//
// _NET_WM_STATE_SKIP_TASKBAR and _NET_WM_STATE_SKIP_PAGER. Before the window
// is mapped they are written as the initial state, which is the only way to
// open already out of the list rather than to appear in it for a frame;
// afterwards the state is the window manager's, and a client asks with a
// message. Writing the property directly on a mapped window is the mistake
// that makes a state change look like it worked and do nothing.
func (s *x11Surface) SetSkipTaskbar(skip bool) bool {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if c := s.conn; c.atomSkipTaskbar == 0 || c.atomSkipPager == 0 {
		return false
	}
	// Asked for in its own right, so a later change of role does not undo it.
	s.task.ask(skip)
	return s.applySkipTaskbarLocked()
}

// applySkipTaskbarLocked publishes whatever the policy now says.
func (s *x11Surface) applySkipTaskbarLocked() bool {
	c := s.conn
	if c.atomSkipTaskbar == 0 || c.atomSkipPager == 0 {
		return false
	}
	skip := s.task.skipping()
	if !s.mapped {
		// The initial state carries every state the window claims, this one
		// among them.
		s.setInitialStateLocked(s.state)
		return true
	}
	// Both in one message: the EWMH lets a client name two states at once,
	// and a window out of the task bar but still in the pager is in neither
	// of the two places the caller meant.
	s.ewmhStateLocked(skip, c.atomSkipTaskbar, c.atomSkipPager)
	return true
}

// Center puts the window in the middle of the work area of the monitor
// it is on ([CenterSurface]).
//
// The work area rather than the whole screen, so a centred dialog is
// centred in the space the panels leave rather than behind one. Before
// the window is mapped this is the position it opens at; afterwards it
// is a move, which a window manager may honour or ignore as it does any
// other.
func (s *x11Surface) Center() bool {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return false
	}
	x11Mu.Lock()
	sc := s.conn.displayScale()
	rx, ry := s.rootOriginLocked()
	lw, lh := s.geomLW, s.geomLH
	x11Mu.Unlock()
	// The monitor the window is already on, asked for by the point at
	// its middle: on a two-monitor desktop, centring must mean the
	// middle of *this* screen, not of the one the desktop happens to
	// call first.
	here := FrameRect{
		X: LogicalPixels(rx, sc) + lw/2,
		Y: LogicalPixels(ry, sc) + lh/2,
	}
	wa, ok := x11ScreenRectAt(here.X, here.Y)
	if !ok || wa.W <= 0 || wa.H <= 0 {
		return false
	}
	x := wa.X + (wa.W-lw)/2
	y := wa.Y + (wa.H-lh)/2
	if x < wa.X {
		x = wa.X
	}
	if y < wa.Y {
		y = wa.Y
	}
	return s.Move(x, y)
}

// Activate asks the window manager to bring the window to the front and
// give it the keyboard ([ActivateSurface]). On X11 that is
// _NET_ACTIVE_WINDOW, which is what Raise already sends.
func (s *x11Surface) Activate() bool { return s.Raise() }

// setInitialAboveLocked writes _NET_WM_STATE_ABOVE before the window is
// mapped, so it opens above the others rather than being raised a frame
// after — in which frame it can be covered by whatever launched it.
func (s *x11Surface) setInitialAboveLocked() {
	s.setInitialStateLocked(WindowState{KeepAbove: true})
}

// setInitialStateLocked writes the states of st a client may claim onto a
// window that is not mapped yet, for a window manager to read when it takes
// the window over.
//
// Only before the map. Afterwards _NET_WM_STATE is the window manager's to
// write and a client asks with a message (ui_ewmh_state), which is what
// SetKeepAbove and the maximize calls do.
//
// Its other caller is recreateOnVisual, and that is the bug it closes: the
// property lives on the *window*, and an X window's visual is fixed when it
// is created, so a frame that gains a shadow destroys the window and makes
// another one. Everything the window manager had been told about it went
// with it — a maximized window came back the size of its restored self, a
// full-screen one came back with decorations — and the application was never
// told, because from its own side nothing had changed.
//
// Minimized and Activated are left out on purpose: a client that writes
// _NET_WM_STATE_HIDDEN or _FOCUSED is claiming something only the window
// manager decides.
func (s *x11Surface) setInitialStateLocked(st WindowState) {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 || c.atomNetState == 0 {
		return
	}
	var set []C.Atom
	add := func(a C.Atom, on bool) {
		if on && a != 0 {
			set = append(set, a)
		}
	}
	add(c.atomAbove, st.KeepAbove)
	add(c.atomSkipTaskbar, s.task.skipping())
	add(c.atomSkipPager, s.task.skipping())
	add(c.atomMaxHorz, st.Maximized)
	add(c.atomMaxVert, st.Maximized)
	add(c.atomFullscr, st.Fullscreen)
	if len(set) == 0 {
		return
	}
	C.ui_dlg_set_states(c.dpy, s.win, c.atomNetState, &set[0], C.int(len(set)))
}
