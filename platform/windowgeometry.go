package platform

import "strings"

// The window-geometry seam: one interface for where a window is on the
// desktop, how big it may be, and whether it is on screen at all — and
// one value that says which of it the window system will do.
//
// # Why these belong together
//
// This was five interfaces — HostWindow, HostMover, HostPositioner,
// ScreenPlacer, SizingSurface — found by type assertion, and the free
// functions over them were the clearest case of silent failure in the
// package:
//
//	func RaiseSurface(s Surface)        { if h, ok := s.(HostWindow); ok { … } }
//	func HideSurface(s Surface)         { if h, ok := s.(HostWindow); ok { … } }
//	func MoveSurface(s Surface, x, y int) { if m, ok := s.(HostMover); ok { … } }
//
// No return value, no else. A window that could not be moved was not
// moved, and nothing anywhere found out. Two of the five were implemented
// by every backend, so their assertion never once answered no.
//
// They are one seam because they are one question asked three ways —
// where the desktop is keeping this window — and because an application
// that wants to put a window somewhere needs all three answers together:
// where it is now, whether it may be moved, and how big it is allowed to
// be when it gets there. Visibility is here rather than beside it because
// "not on the desktop" is the answer to the same question.
//
// **Every backend implements all of it.** A backend that cannot do
// something leaves its bit out of [WindowGeometry.GeometryCaps] and its
// method returns false.

// GeometryCaps is what the window system will do with one window's place
// on the desktop, now.
type GeometryCaps uint32

const (
	// GeometryMove: the client can ask for the window to be at a
	// position. X11 can (XMoveWindow), Win32 can (SetWindowPos), AppKit
	// can (setFrameOrigin:). **Wayland cannot at all**: an xdg_toplevel
	// has no position, a client is never told where its windows are and
	// cannot ask for one. It is the protocol, not an omission.
	GeometryMove GeometryCaps = 1 << iota
	// GeometryPosition: the backend is told where the desktop put the
	// window, so [WindowGeometry.Position] answers. It goes with
	// GeometryMove on every platform so far but is a separate question:
	// one is asking, the other is being told.
	GeometryPosition
	// GeometryScreenPlace: a window opened with [PlaceAtScreen] is really
	// put where it asks — which on Wayland needs zwlr_layer_shell_v1,
	// because a plain xdg_toplevel has nowhere to be put.
	GeometryScreenPlace
	// GeometryVisibility: the window can be shown, hidden and raised.
	GeometryVisibility
	// GeometrySizeLimits: the window system is told the window's minimum
	// and maximum size and honours them (WM_NORMAL_HINTS,
	// xdg_toplevel.set_min_size, WM_GETMINMAXINFO, contentMin/MaxSize).
	GeometrySizeLimits
)

// Has reports whether every capability in x is present.
func (c GeometryCaps) Has(x GeometryCaps) bool { return c&x == x }

var geometryCapNames = []string{
	"move", "position", "screen-place", "visibility", "size-limits",
}

// String lists the capabilities present, in bit order, space separated
// ("none" for none).
func (c GeometryCaps) String() string {
	if c == 0 {
		return "none"
	}
	out := make([]string, 0, len(geometryCapNames))
	for i, name := range geometryCapNames {
		if c&(1<<uint(i)) != 0 {
			out = append(out, name)
		}
	}
	return strings.Join(out, " ")
}

// WindowGeometry is a window's place on the desktop: where it is, how big
// it may be, and whether it is on screen.
//
// Every request reports whether it was made, not whether the desktop
// obliged — a window manager may put a window somewhere else, and
// [WindowGeometry.Position] is what it did, not what was asked.
type WindowGeometry interface {
	// GeometryCaps is what the window system will do with this window's
	// place on the desktop now. It is named for its seam rather than
	// being a bare Caps() because one surface implements several seams
	// and each has capabilities of its own — see [WindowFrame.FrameCaps].
	GeometryCaps() GeometryCaps

	// Move asks for the window's top-left corner to be at x, y in logical
	// pixels of the desktop (GeometryMove).
	Move(x, y int) bool
	// Position is where the desktop put the window, in logical pixels;
	// ok is false where the backend is not told (GeometryPosition).
	Position() (x, y int, ok bool)
	// PlaceAtScreen puts the window at x, y in logical pixels of the
	// desktop, by whatever means the backend has — on Wayland a layer
	// surface, elsewhere an ordinary move (GeometryScreenPlace).
	PlaceAtScreen(x, y int) bool

	// Show maps the window, Hide unmaps or minimizes it, Raise brings it
	// to the front, and Visible reports whether it is on the desktop
	// (GeometryVisibility).
	Show() bool
	Hide() bool
	Raise() bool
	Visible() bool

	// Sizing is the window's resize policy, SetSizing changes it and
	// re-states the limits, and SizeLimits is what the window system was
	// last told (GeometrySizeLimits).
	Sizing() Sizing
	SetSizing(s Sizing) bool
	SizeLimits() SizeLimits
}

// GeometryOf is s's geometry seam. **It is never nil.** A surface that has
// none answers with one that can do nothing: no capabilities, every
// request false, no position, not visible, and a resizable policy with no
// limits.
func GeometryOf(s Surface) WindowGeometry {
	if s == nil {
		return noGeometry{}
	}
	if g, ok := s.(WindowGeometry); ok {
		return g
	}
	return noGeometry{}
}

// GeometryCapsOf is what the window system will do with s's place on the
// desktop now (nothing, for a surface with no geometry seam).
func GeometryCapsOf(s Surface) GeometryCaps { return GeometryOf(s).GeometryCaps() }

// noGeometry is the geometry seam of a surface that has none.
type noGeometry struct{}

func (noGeometry) GeometryCaps() GeometryCaps  { return 0 }
func (noGeometry) Move(int, int) bool          { return false }
func (noGeometry) Position() (int, int, bool)  { return 0, 0, false }
func (noGeometry) PlaceAtScreen(int, int) bool { return false }
func (noGeometry) Show() bool                  { return false }
func (noGeometry) Hide() bool                  { return false }
func (noGeometry) Raise() bool                 { return false }
func (noGeometry) Visible() bool               { return false }
func (noGeometry) Sizing() Sizing              { return SizingResizable }
func (noGeometry) SetSizing(Sizing) bool       { return false }
func (noGeometry) SizeLimits() SizeLimits      { return SizeLimits{} }

// Every backend implements the whole seam. A capability the boundary
// grows cannot quietly stop being covered: the headless tests stop
// compiling rather than stop testing.
var _ WindowGeometry = (*Offscreen)(nil)
