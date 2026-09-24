package platform

// Absolute placement: a window that has to be at one point of the screen
// and nowhere else.
//
// Most windows do not care. The desktop puts them where it likes and
// [WindowOptions].X, Y are at most a hint — that is why a Wayland toplevel
// ignores them, and why saying so in the doc comment was for years the
// whole of the story.
//
// A tray menu does care. The host hands the toolkit a point in root
// coordinates (SNI ContextMenu) and the menu belongs *there*, next to the
// icon that was clicked; a menu in the middle of the screen is not a menu
// that was placed loosely, it is a bug. Such a window asks for
// [PlaceAtScreen], and the backend either honours it or says it cannot —
// on Wayland through zwlr_layer_shell_v1, which is the only way a client
// puts a surface at an absolute position (see wayland_layer_linux.go).
//
// The point of [ScreenPlacementAvailable] is that the layer above can ask
// *before* it commits to a chrome that needs placement: app.NewStatusItem
// falls back to the desktop-drawn menu where the answer is no.

// Placement says what a window's X, Y mean.
type Placement uint8

const (
	// PlaceDesktop — the zero value — leaves the window to the desktop.
	// X, Y are a hint that X11 honours and a Wayland toplevel ignores.
	PlaceDesktop Placement = iota
	// PlaceAtScreen asks for the window's top-left corner to be exactly
	// at X, Y on the screen, in logical pixels of the desktop. A backend
	// that cannot do it opens an ordinary window rather than failing, so
	// the caller that *needs* the position has to ask
	// [ScreenPlacementAvailable] first.
	PlaceAtScreen
)

// ScreenPlacer is an optional Surface capability: put the window at an
// absolute point of the screen and report whether it went there. It is
// distinct from [HostMover] on purpose — a Wayland surface can answer this
// one (a layer surface can move; a toplevel cannot), while [HostMover]
// keeps meaning "this backend places windows at all", which Wayland does
// not.
type ScreenPlacer interface {
	// PlaceAtScreen moves the window to x, y in logical pixels of the
	// desktop. false means the window did not move.
	PlaceAtScreen(x, y int) bool
}

// PlaceSurfaceAtScreen puts s at x, y in logical pixels of the desktop and
// reports whether it went there: through [ScreenPlacer] where the surface
// has an opinion, else through [HostMover], else not at all.
func PlaceSurfaceAtScreen(s Surface, x, y int) bool {
	if s == nil {
		return false
	}
	if p, ok := s.(ScreenPlacer); ok {
		return p.PlaceAtScreen(x, y)
	}
	if m, ok := s.(HostMover); ok {
		m.Move(x, y)
		return true
	}
	return false
}

// simScreenPlace overrides [ScreenPlacementAvailable] for tests; nil means
// ask the backend.
var simScreenPlace *bool

// SimulateScreenPlacement makes [ScreenPlacementAvailable] answer want,
// whatever the backend would say — the way Offscreen.SimulatePopups makes
// a headless desktop open popups. It returns the function that puts the
// answer back. Tests only; it is process-wide.
func SimulateScreenPlacement(want bool) func() {
	old := simScreenPlace
	v := want
	simScreenPlace = &v
	return func() { simScreenPlace = old }
}

// ScreenPlacementAvailable reports whether a window opened with
// Place: [PlaceAtScreen] will really be put where it asks.
//
// X11, Windows and the offscreen desktop place windows and answer yes.
// Wayland answers yes only where the compositor offers
// zwlr_layer_shell_v1 (KDE, sway, Hyprland, wayfire do; GNOME/Mutter has
// declined it), because a plain xdg_toplevel has no position at all.
func ScreenPlacementAvailable() bool {
	if simScreenPlace != nil {
		return *simScreenPlace
	}
	if Default(false).Name() != "wayland" {
		return true
	}
	return LayerSurfacesAvailable()
}
