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

// ui_dlg_set_state writes the initial _NET_WM_STATE. A window manager
// reads it when it takes the window over, which is the only way to open
// *already* above the others rather than to be raised a frame later.
static void ui_dlg_set_state(Display* d, Window w, Atom prop, Atom state) {
	XChangeProperty(d, w, prop, XA_ATOM, 32, PropModeReplace,
		(const unsigned char*)&state, 1);
	XFlush(d);
}

static void ui_dlg_set_transient(Display* d, Window w, Window parent) {
	XSetTransientForHint(d, w, parent);
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
	if r == RoleDialog {
		t = c.atomWinTypeDialog
	}
	if t == 0 {
		return false
	}
	C.ui_dlg_set_type(c.dpy, s.win, c.atomWinType, t)
	s.role = r
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
	c := s.conn
	if c.atomNetState == 0 || c.atomAbove == 0 {
		return
	}
	C.ui_dlg_set_state(c.dpy, s.win, c.atomNetState, c.atomAbove)
}
