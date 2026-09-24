package platform

// ScreenRectAt is the rectangle of the screen that holds the desktop
// point x, y, in logical pixels of the desktop — the unit
// [WindowOptions].X, Y and [ScreenPlacer.PlaceAtScreen] speak. It is what
// a window placed at an absolute point is constrained to
// ([SolveScreenMenu]).
//
// Where each backend takes it from:
//
//	X11        the RandR monitor holding the point, cut to
//	           _NET_WORKAREA — the screen less its panels.
//	Wayland    the output holding the point: zxdg_output_v1's
//	           logical_size where the compositor offers xdg-output (the
//	           only source that is right under fractional scaling), else
//	           wl_output's mode divided by its integer scale. A Wayland
//	           client is not told where the panels are, so this is the
//	           whole output; a layer surface is not pushed out of their
//	           way either, which is what makes that acceptable here.
//	elsewhere  nothing — ok is false.
//
// ok is false when the backend cannot say: no connection, no output has
// claimed the point, or the compositor has not yet sent the geometry.
// The caller then constrains against nothing rather than against a guess,
// which is what the toolkit did before it could ask at all.
func ScreenRectAt(x, y int) (FrameRect, bool) {
	if simScreenRect != nil {
		return *simScreenRect, simScreenRect.W > 0 && simScreenRect.H > 0
	}
	return screenRectAt(x, y)
}

// simScreenRect overrides [ScreenRectAt] for tests; nil means ask the
// backend.
var simScreenRect *FrameRect

// SimulateScreenRect makes [ScreenRectAt] answer r for every point,
// whatever the backend would say — the way [SimulateScreenPlacement] does
// for placement. It returns the function that puts the answer back. A
// rectangle with no area makes ScreenRectAt answer false. Tests only; it
// is process-wide.
func SimulateScreenRect(r FrameRect) func() {
	old := simScreenRect
	v := r
	simScreenRect = &v
	return func() { simScreenRect = old }
}
