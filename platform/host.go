package platform

// Where a window is on the desktop, whether it is on screen, and how big
// it may be, are one seam: [WindowGeometry], reached with [GeometryOf].
// These are the free functions over it, kept because they read better at
// the call site than GeometryOf(s).Thing() and because a surface may be
// nil there.
//
// They used to be the whole story, and it did not work:
//
//	func RaiseSurface(s Surface)          { if h, ok := s.(HostWindow); ok { … } }
//	func HideSurface(s Surface)           { if h, ok := s.(HostWindow); ok { … } }
//	func MoveSurface(s Surface, x, y int) { if m, ok := s.(HostMover); ok { … } }
//
// No return value and no else, so a window that could not be moved was
// not moved and nothing found out. Each of these now answers.

// SurfacePosition is where the desktop put s, in logical pixels, and
// whether the backend knows ([GeometryPosition]).
func SurfacePosition(s Surface) (x, y int, ok bool) { return GeometryOf(s).Position() }

// RaiseSurface shows and raises s, and reports whether there was anything
// to ask ([GeometryVisibility]).
func RaiseSurface(s Surface) bool {
	g := GeometryOf(s)
	if !g.Show() {
		return false
	}
	return g.Raise()
}

// HideSurface unmaps or minimizes s, and reports whether there was
// anything to ask ([GeometryVisibility]).
func HideSurface(s Surface) bool { return GeometryOf(s).Hide() }

// SurfaceVisible reports whether s is on the desktop. A surface with no
// geometry seam of its own is visible while it is open, which is what a
// backend with no hide protocol always answered.
func SurfaceVisible(s Surface) bool {
	if s == nil {
		return false
	}
	if !GeometryCapsOf(s).Has(GeometryVisibility) {
		return !s.Closed()
	}
	return GeometryOf(s).Visible()
}

// MoveSurface places s at (x, y) on the desktop, in logical pixels, and
// reports whether there was anything to ask ([GeometryMove]).
func MoveSurface(s Surface, x, y int) bool { return GeometryOf(s).Move(x, y) }

// SurfaceMoves reports whether s can be placed by the client at all. It is
// the other half of [SurfacePosition], and an app that keeps two windows
// stuck together has to ask both: X11, Windows, macOS and the offscreen
// backend answer yes; **Wayland answers no, and that is the protocol
// rather than an omission** — an xdg_toplevel has no position, a client is
// never told where its windows are and cannot ask for one. Anything that
// would need two windows' positions, such as carrying a torn-off window
// under the pointer, goes through xdg-toplevel-drag-v1 instead
// ([ToplevelDragSurface]).
func SurfaceMoves(s Surface) bool { return GeometryCapsOf(s).Has(GeometryMove) }
