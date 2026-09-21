package platform

// HostWindow is an optional Surface capability: raise / show / hide the
// native window. Offscreen implements it as a visibility flag. X11 maps
// and sends _NET_ACTIVE_WINDOW. Wayland Hide drops the xdg_toplevel
// role (and may set_minimized first); Show remaps the role and requests
// xdg_activation_v1 when the compositor advertises it.
type HostWindow interface {
	Raise()
	Show()
	Hide()
	Visible() bool
}

// HostMover is an optional Surface capability: move the native window
// on the desktop, in logical pixels (X11 converts to the root window's
// device pixels; offscreen places it on its simulated desktop). Wayland
// does not implement it: a toplevel has no position.
type HostMover interface {
	Move(x, y int)
}

// HostPositioner is an optional Surface capability: say where the desktop
// put the window, in logical pixels of the desktop — the unit its size is
// stated in. X11 answers from the last ConfigureNotify, offscreen from
// where a test put the window.
//
// Wayland does not implement it, and that is the protocol and not an
// omission: a toplevel has no position. A client is never told where its
// windows are and cannot ask for one, so anything that would need two
// windows' positions — carrying a torn-off window under the pointer —
// goes through xdg-toplevel-drag-v1 instead ([ToplevelDragSurface]).
type HostPositioner interface {
	// Position is the window's top-left corner on the desktop, in logical
	// pixels. ok is false where the backend is not told.
	Position() (x, y int, ok bool)
}

// SurfacePosition is where the desktop put s, in logical pixels, and
// whether the backend knows (see [HostPositioner]).
func SurfacePosition(s Surface) (x, y int, ok bool) {
	if s == nil {
		return 0, 0, false
	}
	p, is := s.(HostPositioner)
	if !is {
		return 0, 0, false
	}
	return p.Position()
}

// RaiseSurface shows and raises s when it implements [HostWindow].
func RaiseSurface(s Surface) {
	if s == nil {
		return
	}
	if h, ok := s.(HostWindow); ok {
		h.Show()
		h.Raise()
	}
}

// HideSurface unmaps / minimizes s when supported.
func HideSurface(s Surface) {
	if s == nil {
		return
	}
	if h, ok := s.(HostWindow); ok {
		h.Hide()
	}
}

// SurfaceVisible reports true when s has no hide protocol or is mapped.
func SurfaceVisible(s Surface) bool {
	if s == nil {
		return false
	}
	if h, ok := s.(HostWindow); ok {
		return h.Visible()
	}
	return !s.Closed()
}

// MoveSurface places s at (x, y) on the desktop, in logical pixels, when
// the backend can.
func MoveSurface(s Surface, x, y int) {
	if s == nil {
		return
	}
	if m, ok := s.(HostMover); ok {
		m.Move(x, y)
	}
}

// SurfaceMoves reports whether s can be placed by the client at all (see
// [HostMover]). It is the other half of [SurfacePosition], and an app that
// keeps two windows stuck together has to ask both: X11 and the offscreen
// backend answer yes, Wayland answers no and that is the protocol rather
// than an omission.
func SurfaceMoves(s Surface) bool {
	if s == nil {
		return false
	}
	_, ok := s.(HostMover)
	return ok
}
