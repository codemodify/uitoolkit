package platform

import (
	"math"
	"os"
	"strings"
)

// This file holds the window-frame vocabulary shared by every backend: who
// draws a window's frame (Decorations), what the desktop says about the
// window (WindowState) and what a frame the toolkit draws hands back to
// the desktop — a move, a resize, the window menu. [WindowFrame] is the
// seam those travel through; this is the vocabulary they are said in. It
// has no cgo and no build tag, so the decoders below are tested headless.

// Decorations says who draws a top-level window's frame: its title bar,
// caption buttons and resize borders.
type Decorations uint8

const (
	// DecorationsAuto leaves the choice to the toolkit: the app package picks
	// the toolkit's frame for a window with a custom title bar
	// (Window.SetTitleBar) and the desktop's frame for every other window.
	// A backend handed Auto treats it as DecorationsServer.
	DecorationsAuto Decorations = iota
	// DecorationsServer: the compositor or window manager draws the frame
	// (KWin's Breeze frame, an X11 window manager's). Where nothing else can
	// (GNOME's Mutter and Weston have no server-side decorations) the
	// toolkit draws it after all.
	DecorationsServer
	// DecorationsClient: uitoolkit draws the frame (a client-side frame).
	DecorationsClient
	// DecorationsNone: no frame at all; the app draws whatever it wants
	// (splash screens, kiosks, shaped windows).
	DecorationsNone
)

// EnvDecorations overrides every window's decoration mode for the process:
// UITK_DECORATIONS=server|client|none (also system / toolkit, and auto), the
// testing and escape hatch that UITK_PAINT is for painting.
const EnvDecorations = "UITK_DECORATIONS"

// EnvXdgDecoration set to 0 makes the Wayland backend ignore the
// compositor's zxdg_decoration_manager_v1, as if it had no server-side
// decorations (GNOME, Weston): a testing switch for that path on KWin.
const EnvXdgDecoration = "UITK_XDG_DECORATION"

func (d Decorations) String() string {
	switch d {
	case DecorationsServer:
		return "server"
	case DecorationsClient:
		return "client"
	case DecorationsNone:
		return "none"
	}
	return "auto"
}

// ParseDecorations reads a decoration mode: auto, server (system, ssd),
// client (toolkit, csd) or none (frameless). ok is false for anything else.
func ParseDecorations(s string) (Decorations, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "":
		return DecorationsAuto, s != ""
	case "server", "system", "ssd", "native":
		return DecorationsServer, true
	case "client", "toolkit", "csd", "custom":
		return DecorationsClient, true
	case "none", "frameless", "undecorated":
		return DecorationsNone, true
	}
	return DecorationsAuto, false
}

// DecorationsFromEnv is the UITK_DECORATIONS override; ok is false when it is
// unset, "auto" or not a mode.
func DecorationsFromEnv() (Decorations, bool) {
	d, ok := ParseDecorations(os.Getenv(EnvDecorations))
	if !ok || d == DecorationsAuto {
		return DecorationsAuto, false
	}
	return d, true
}

// FrameInsets is a band around a window's edges, in whole device pixels.
type FrameInsets struct {
	Top, Right, Bottom, Left int
}

// Zero reports whether every side is zero.
func (in FrameInsets) Zero() bool {
	return in.Top == 0 && in.Right == 0 && in.Bottom == 0 && in.Left == 0
}

// Width and Height are how much the band adds to a window's size.
func (in FrameInsets) Width() int  { return in.Left + in.Right }
func (in FrameInsets) Height() int { return in.Top + in.Bottom }

// Frame is what a window whose frame the toolkit draws tells the window
// system. All of it is in device pixels, in surface coordinates.
//
// The surface is the visible window grown by Margin: an invisible band
// holding the drop shadow, with the resize handles Input deep inside it and
// the rest clicking through to whatever is behind. The compositor is told
// the visible window (Wayland's xdg_surface.set_window_geometry, X11's
// _GTK_FRAME_EXTENTS), so snapping, tiling and the task bar ignore the
// shadow.
type Frame struct {
	// Margin is the invisible band: zero for a frame with no shadow, and
	// zero on an edge that is maximized or tiled against another window.
	Margin FrameInsets
	// Input is how far into Margin a press still reaches the window (the
	// resize band in the shadow); the rest of the margin is not ours.
	Input FrameInsets
	// Radius are the visible window's corner radii — top-left, top-right,
	// bottom-right, bottom-left — which the opaque region leaves out.
	Radius [4]float32
	// Alpha: the buffer needs an alpha channel (there is a shadow or a
	// rounded corner). Without it the backend keeps its opaque buffers and
	// the full opaque region, as for a window with no frame of ours.
	Alpha bool

	// Shape, when non-nil, is the window's silhouette (see [Shape]):
	// exactly where the window is, and so exactly where a press reaches it.
	// Everything else — the margin, and a hole in the middle as much as the
	// space around a disc — is not the window at all: the desktop shows
	// through it and a press lands on whatever is behind.
	//
	// It replaces Margin and Input entirely: a shaped window that wants a
	// resize band puts the band in the shape itself. An empty but non-nil
	// Shape is a window that takes no input anywhere (a click-through
	// overlay); a nil Shape is an ordinary rectangular window, and takes
	// exactly the path it took before shapes existed.
	Shape []FrameRect
	// Opaque, when non-nil, replaces the rounded-rect opaque region: the
	// pixels a compositor may treat as solid and skip drawing behind.
	// Claiming a translucent pixel here is visible corruption, so it holds
	// only pixels the window really does paint solid.
	Opaque []FrameRect
	// Blur is where the compositor should blur what is *behind* the window
	// — real glass, not the window's own pixels (ext_background_effect_v1
	// on Wayland, _KDE_NET_WM_BLUR_BEHIND_REGION on X11). Nil asks for
	// none. A desktop that cannot blur ignores it, so a look that asks
	// keeps whatever it paints for itself.
	Blur []FrameRect
}

// Zero reports whether f asks for nothing: no margin, no rounded corner, no
// alpha, no shape and no blur — an opaque frame that fills its surface.
func (f Frame) Zero() bool {
	return f.Margin.Zero() && !f.Alpha && f.Radius == [4]float32{} &&
		f.Shape == nil && f.Opaque == nil && f.Blur == nil
}

// Same reports whether f and g ask for the same thing. Frame stopped being
// comparable with == when it grew regions, and every backend compares the
// frame it is handed against the one it already applied.
func (f Frame) Same(g Frame) bool {
	if f.Margin != g.Margin || f.Input != g.Input || f.Radius != g.Radius || f.Alpha != g.Alpha {
		return false
	}
	return sameRects(f.Shape, g.Shape) && sameRects(f.Opaque, g.Opaque) && sameRects(f.Blur, g.Blur)
}

// sameRects compares two region lists, telling "no region" (nil) apart from
// "the empty region" (non-nil, no rectangles): the first leaves the
// surface's own default in place, the second states that nothing is there.
func sameRects(a, b []FrameRect) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FrameMargin is a margin of at least want device pixels that is also a
// whole number of logical pixels at scale: Wayland states a window's
// geometry in logical pixels, so a margin that is not one would leave the
// visible window a fraction of a pixel off the compositor's idea of it. It
// looks a few logical pixels ahead and then gives up and rounds — at 1.75
// a 10px margin becomes 12 logical (21 device), at 1.5 it stays 10 (15).
func FrameMargin(want, scale float32) int {
	if want <= 0 {
		return 0
	}
	if scale <= 0 {
		scale = 1
	}
	l0 := int(math.Ceil(float64(want/scale) - 1e-4))
	if l0 < 1 {
		l0 = 1
	}
	for l := l0; l <= l0+8; l++ {
		d := float64(l) * float64(scale)
		if math.Abs(d-math.Round(d)) < 1e-3 {
			return int(math.Round(d))
		}
	}
	return int(math.Round(float64(l0) * float64(scale)))
}

// FrameRect is a box of a window's surface, in whatever unit the caller
// works in (logical pixels on Wayland, device pixels on X11).
type FrameRect struct{ X, Y, W, H int }

// InputRect is where a press reaches a window whose frame the toolkit
// draws: the visible window win grown by the resize band in the shadow.
// The rest of the margin is not the window's — it clicks through to
// whatever is behind it.
func InputRect(win FrameRect, margin, input FrameInsets) FrameRect {
	l := min(input.Left, margin.Left)
	t := min(input.Top, margin.Top)
	r := min(input.Right, margin.Right)
	b := min(input.Bottom, margin.Bottom)
	return FrameRect{X: win.X - l, Y: win.Y - t, W: win.W + l + r, H: win.H + t + b}
}

// OpaqueRects is the window less the squares its rounded corners live in —
// what a compositor may take as solid. radius is in device pixels
// (top-left clockwise) and scale converts it to win's unit, rounded up so
// nothing translucent is ever claimed opaque. An empty result means
// "nothing is certainly opaque".
func OpaqueRects(win FrameRect, radius [4]float32, scale float32) []FrameRect {
	if win.W < 1 || win.H < 1 {
		return nil
	}
	if scale <= 0 {
		scale = 1
	}
	up := func(v float32) int {
		if v <= 0 {
			return 0
		}
		return int(math.Ceil(float64(v / scale)))
	}
	tl, tr, br, bl := up(radius[0]), up(radius[1]), up(radius[2]), up(radius[3])
	top, bottom := max(tl, tr), max(br, bl)
	if top+bottom >= win.H {
		return nil
	}
	out := []FrameRect{{X: win.X, Y: win.Y + top, W: win.W, H: win.H - top - bottom}}
	if top > 0 && win.W-tl-tr > 0 {
		out = append(out, FrameRect{X: win.X + tl, Y: win.Y, W: win.W - tl - tr, H: top})
	}
	if bottom > 0 && win.W-bl-br > 0 {
		out = append(out, FrameRect{X: win.X + bl, Y: win.Y + win.H - bottom, W: win.W - bl - br, H: bottom})
	}
	return out
}

// Edges is a set of window edges. A resize edge is one side or a corner
// (two adjacent sides).
type Edges uint8

const (
	EdgeTop Edges = 1 << iota
	EdgeBottom
	EdgeLeft
	EdgeRight
)

// Valid reports whether e is one side or a corner: not empty and not two
// opposite sides.
func (e Edges) Valid() bool {
	if e == 0 || e&^(EdgeTop|EdgeBottom|EdgeLeft|EdgeRight) != 0 {
		return false
	}
	return e&(EdgeTop|EdgeBottom) != EdgeTop|EdgeBottom && e&(EdgeLeft|EdgeRight) != EdgeLeft|EdgeRight
}

func (e Edges) String() string {
	if e == 0 {
		return "none"
	}
	var parts []string
	for _, x := range []struct {
		e    Edges
		name string
	}{{EdgeTop, "top"}, {EdgeBottom, "bottom"}, {EdgeLeft, "left"}, {EdgeRight, "right"}} {
		if e&x.e != 0 {
			parts = append(parts, x.name)
		}
	}
	return strings.Join(parts, "-")
}

// ResizeCursor is the pointer shape for resizing from e (the default arrow
// for anything that is not a side or a corner).
func (e Edges) ResizeCursor() Cursor {
	switch e {
	case EdgeTop:
		return CursorResizeN
	case EdgeBottom:
		return CursorResizeS
	case EdgeLeft:
		return CursorResizeW
	case EdgeRight:
		return CursorResizeE
	case EdgeTop | EdgeLeft:
		return CursorResizeNW
	case EdgeTop | EdgeRight:
		return CursorResizeNE
	case EdgeBottom | EdgeLeft:
		return CursorResizeSW
	case EdgeBottom | EdgeRight:
		return CursorResizeSE
	}
	return CursorDefault
}

// xdgResizeEdge maps e to xdg_toplevel.resize_edge (top 1, bottom 2, left 4,
// right 8, corners their sums); 0 (none) when e is not a side or a corner.
func xdgResizeEdge(e Edges) uint32 {
	if !e.Valid() {
		return 0
	}
	var v uint32
	if e&EdgeTop != 0 {
		v |= 1
	}
	if e&EdgeBottom != 0 {
		v |= 2
	}
	if e&EdgeLeft != 0 {
		v |= 4
	}
	if e&EdgeRight != 0 {
		v |= 8
	}
	return v
}

// _NET_WM_MOVERESIZE directions (EWMH 1.5 §4.3).
const (
	netMoveResizeSizeTopLeft     = 0
	netMoveResizeSizeTop         = 1
	netMoveResizeSizeTopRight    = 2
	netMoveResizeSizeRight       = 3
	netMoveResizeSizeBottomRight = 4
	netMoveResizeSizeBottom      = 5
	netMoveResizeSizeBottomLeft  = 6
	netMoveResizeSizeLeft        = 7
	netMoveResizeMove            = 8
	netMoveResizeCancel          = 11
)

// netMoveResizeDirection maps e to a _NET_WM_MOVERESIZE size direction.
func netMoveResizeDirection(e Edges) (uint32, bool) {
	switch e {
	case EdgeTop | EdgeLeft:
		return netMoveResizeSizeTopLeft, true
	case EdgeTop:
		return netMoveResizeSizeTop, true
	case EdgeTop | EdgeRight:
		return netMoveResizeSizeTopRight, true
	case EdgeRight:
		return netMoveResizeSizeRight, true
	case EdgeBottom | EdgeRight:
		return netMoveResizeSizeBottomRight, true
	case EdgeBottom:
		return netMoveResizeSizeBottom, true
	case EdgeBottom | EdgeLeft:
		return netMoveResizeSizeBottomLeft, true
	case EdgeLeft:
		return netMoveResizeSizeLeft, true
	}
	return 0, false
}

// WindowState is what the compositor or window manager says about a
// top-level window.
type WindowState struct {
	// Maximized: maximized both ways. Fullscreen covers the screen.
	Maximized, Fullscreen bool
	// Activated: paint the window's frame as active. It follows the
	// desktop's idea of the active window (xdg_toplevel's activated,
	// X11's _NET_WM_STATE_FOCUSED), not keyboard focus.
	Activated bool
	// Resizing: an interactive resize is running.
	Resizing bool
	// Suspended: the window is not visible at all (minimized, on another
	// desktop, behind a full-screen window): animations and the caret
	// blink may pause (xdg-shell v6).
	Suspended bool
	// Minimized: iconified (X11 _NET_WM_STATE_HIDDEN; Wayland never says).
	Minimized bool
	// Tiled are the edges that touch a screen edge or a tiled neighbour:
	// no shadow and square corners there, and no resizing from them.
	Tiled Edges
	// Constrained are the edges that cannot be resized (xdg-shell v7).
	Constrained Edges
	// Solid: the desktop does not composite the window (X11 without a
	// compositing manager), so a frame the toolkit draws must be solid —
	// no shadow, no rounded corners, nothing translucent, since alpha is
	// simply ignored there. Wayland always composites.
	Solid bool
	// KeepAbove: the desktop keeps the window above the others
	// (_NET_WM_STATE_ABOVE). It is the desktop's answer, not the
	// application's request — a window manager that refused says so by
	// never setting it. Wayland's core protocol has no such state and
	// never sets it; see [FrameKeepAbove].
	KeepAbove bool
}

// xdgStateFromMask decodes the backend's xdg_toplevel.configure state set:
// bit n is set when the array held state value n (maximized 1, fullscreen 2,
// resizing 3, activated 4, tiled_left..tiled_bottom 5..8, suspended 9,
// constrained_left..constrained_bottom 10..13).
func xdgStateFromMask(mask uint32) WindowState {
	has := func(v uint) bool { return mask&(1<<v) != 0 }
	st := WindowState{
		Maximized:  has(1),
		Fullscreen: has(2),
		Resizing:   has(3),
		Activated:  has(4),
		Suspended:  has(9),
	}
	for v, e := range map[uint]Edges{5: EdgeLeft, 6: EdgeRight, 7: EdgeTop, 8: EdgeBottom} {
		if has(v) {
			st.Tiled |= e
		}
	}
	for v, e := range map[uint]Edges{10: EdgeLeft, 11: EdgeRight, 12: EdgeTop, 13: EdgeBottom} {
		if has(v) {
			st.Constrained |= e
		}
	}
	return st
}

// xdgFrameCapsFromMask decodes xdg_toplevel.wm_capabilities: bit n is set
// when the array held capability value n (window_menu 1, maximize 2,
// fullscreen 3, minimize 4). What is not in the array the compositor will
// not do for this window, so nothing else is assumed here — the backend
// decides what to answer *before* the compositor has said at all (see
// frameDesktopCaps).
func xdgFrameCapsFromMask(mask uint32) FrameCaps {
	var c FrameCaps
	for v, cap := range map[uint]FrameCaps{1: FrameMenu, 2: FrameMaximize, 3: FrameFullscreen, 4: FrameMinimize} {
		if mask&(1<<v) != 0 {
			c |= cap
		}
	}
	return c
}

// effectiveDecorations is what a backend reports after a negotiation:
// want is what the app asked for, answer the compositor's mode
// (DecorationsServer / DecorationsClient; DecorationsAuto when there is
// no one to ask, as on GNOME, where only a client frame exists).
func effectiveDecorations(want, answer Decorations) Decorations {
	if answer == DecorationsServer {
		return DecorationsServer
	}
	if want == DecorationsNone {
		return DecorationsNone
	}
	return DecorationsClient
}

// requestedDecorations normalises what an app asks a backend for: Auto is
// the system frame.
func requestedDecorations(d Decorations) Decorations {
	if d == DecorationsAuto {
		return DecorationsServer
	}
	return d
}

// shadePin applies a rolled-up window's height pin to its stated limits: a
// shaded window is exactly that tall and not resizable vertically. h of 0
// leaves the limits alone.
func shadePin(l SizeLimits, h int) SizeLimits {
	if h > 0 {
		l.MinHeight, l.MaxHeight = h, h
	}
	return l
}
