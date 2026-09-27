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

// PlaceSurfaceAtScreen puts s at x, y in logical pixels of the desktop and
// reports whether it went there ([WindowGeometry.PlaceAtScreen]).
//
// It is a separate request from [WindowGeometry.Move] on purpose, and
// Wayland is why: a layer surface can be placed while a toplevel cannot
// be moved at all, so [GeometryScreenPlace] and [GeometryMove] are
// different answers there. Everywhere else they are the same answer and
// the backend implements one in terms of the other.
func PlaceSurfaceAtScreen(s Surface, x, y int) bool {
	return GeometryOf(s).PlaceAtScreen(x, y)
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
//
// It asks the backend rather than testing its name. The old test was
// `Name() != "wayland"`, which answered yes for every backend that was
// not the one known to have trouble — so a new backend was assumed able
// before anyone had written the code, and the one place that knew the
// truth was the last to be asked.
func ScreenPlacementAvailable() bool {
	if simScreenPlace != nil {
		return *simScreenPlace
	}
	return BackendCapsOf(Default(false)).Has(BackendScreenPlace)
}
