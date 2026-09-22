package platform

import (
	"math"
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// Popups — menus, combo lists, tooltips — as surfaces of their own.
//
// A popup drawn inside its window can never be larger than the window: a
// menu opened from a 275-pixel strip is cut off at the strip's edge, and a
// skin's silhouette stops where the window does. Qt and GTK open every
// popup as a real surface instead, and so does this: an xdg_popup on
// Wayland, an override-redirect window on X11.
//
// The popup stays part of its window in every way the app can see. Its
// widget tree is still the window's popup layer, it is still in the
// window's accessibility tree, and the keyboard goes where it always went.
// Every input event on a popup surface arrives on its *root* window — the
// top-level it was opened from — with the position already translated into
// that window's device pixels, exactly as if the popup were drawn inside
// it. That is the whole trick: hit-testing, hover, capture, the click
// outside that dismisses, the Retains rule a menu bar relies on — none of
// it knows the popup moved out.
//
// The window system places the popup. It is handed what the popup hangs
// from and which way it would like to go ([PopupPlacement], xdg_positioner's
// vocabulary) and flips, slides or shrinks it against the edges of the
// screen: the compositor does it on Wayland, where a client never learns
// where its window is, and [SolvePopup] does it on X11 and in the
// offscreen backend, within the work area of the monitor the window is on.
//
// A backend that cannot — no display (headless and offscreen, unless a
// test simulates it), or a compositor that refuses a popup — leaves the app
// drawing popups inside the window, as it always did. UITK_POPUPS=layer
// forces that everywhere, for comparison.

// EnvPopups chooses where popups go: "surface" (the default) opens them as
// surfaces of their own wherever the backend can, "layer" draws them inside
// their window everywhere.
const EnvPopups = "UITK_POPUPS"

// PopupSurfacesAllowed reports whether UITK_POPUPS leaves popups free to be
// surfaces of their own (anything but layer, inwindow, window, off, 0).
func PopupSurfacesAllowed() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvPopups))) {
	case "layer", "inwindow", "in-window", "window", "off", "0", "false", "no":
		return false
	}
	return true
}

// PopupAdjust is how the window system may move a popup that would run off
// the screen (xdg_positioner.constraint_adjustment, the same bits in the
// same order).
type PopupAdjust uint8

const (
	// AdjustSlideX, AdjustSlideY: move it along the axis until it fits.
	AdjustSlideX PopupAdjust = 1 << iota
	AdjustSlideY
	// AdjustFlipX, AdjustFlipY: put it on the other side of its anchor.
	AdjustFlipX
	AdjustFlipY
	// AdjustResizeX, AdjustResizeY: shrink it to what is left (a list
	// then scrolls).
	AdjustResizeX
	AdjustResizeY
)

// PopupPlacement says where a popup goes, in xdg_positioner's terms: a
// point of an anchor rectangle, the direction the popup extends from it,
// and what the window system may do when that runs off the screen.
//
// Everything is in **logical pixels relative to the root window's visible
// box** — its window geometry, the margin a frame keeps for its shadow left
// out — whatever the popup's parent. A backend whose protocol wants
// positions relative to a nested popup's parent converts.
type PopupPlacement struct {
	// Anchor is what the popup hangs from: a menu title, a combo box, a
	// submenu's row, the pointer (a 1x1 box).
	Anchor FrameRect
	// AnchorEdge is the point of Anchor the popup starts from: a side or a
	// corner (EdgeBottom|EdgeLeft is the bottom-left corner), zero for the
	// centre.
	AnchorEdge Edges
	// Gravity is the direction the popup extends from that point:
	// EdgeBottom|EdgeRight hangs it down and to the right.
	Gravity Edges
	// W, H are the popup's size (its visible box, the shadow's margin left
	// out).
	W, H int
	// Adjust is what may be done when it does not fit.
	Adjust PopupAdjust
}

// PopupRole is what a popup is, which decides how it takes input.
type PopupRole uint8

const (
	// PopupRoleMenu: a menu, a combo list, a date picker — it takes the
	// pointer and the keyboard while it is up (an xdg_popup grab, X11
	// pointer and keyboard grabs), and a click anywhere outside the
	// application takes it down.
	PopupRoleMenu PopupRole = iota
	// PopupRoleTooltip: it takes nothing and every press goes through it.
	PopupRoleTooltip
)

// PopupOptions open a popup.
type PopupOptions struct {
	// Parent is the surface the popup hangs from: the root window, or a
	// popup already open from it (a submenu's parent menu).
	Parent Surface
	// Placement is where it goes; Role what it is.
	Placement PopupPlacement
	Role      PopupRole
	// Frame is the popup's frame from its first frame on: the margin its
	// shadow is painted in, its silhouette, the glass behind it.
	Frame Frame
}

// PopupOpener is a surface that can open popups as surfaces of their own.
// The Wayland and X11 top-level surfaces and their popups implement it;
// Offscreen does once a test calls [Offscreen.SimulatePopups].
type PopupOpener interface {
	// PopupsSupported reports whether OpenPopup can work here now.
	PopupsSupported() bool
	// OpenPopup opens a popup. The error means the app draws this one
	// inside its window instead.
	OpenPopup(opts PopupOptions) (PopupSurface, error)
}

// PopupSurface is a popup opened by [PopupOpener.OpenPopup]. It is a
// Surface like any other — the app paints its buffer and presents it — with
// three differences: it is placed by the window system, it hands every
// input event to its root window (see the file comment), and its size and
// place are the window system's answer rather than the app's request.
type PopupSurface interface {
	Surface
	// SetFrame and Frame are the popup's frame (see [Frame]): the margin
	// its shadow is painted in, its silhouette, glass.
	SetFrame(f Frame)
	Frame() Frame
	// Reposition moves the popup to p (xdg_popup.reposition): a combo list
	// that grew, a menu whose anchor moved. false means the backend
	// cannot, and the app reopens it instead.
	Reposition(p PopupPlacement) bool
	// Placed is where the window system put the popup's visible box:
	// logical pixels relative to the root window's visible box, the unit
	// PopupPlacement speaks. Its size is the popup's, which the window
	// system may have shrunk.
	Placed() FrameRect
	// Origin is where the popup's buffer (0, 0) is in the root window's
	// buffer, in device pixels: the popup's visible box at Placed, less
	// the popup's own margin. Events from the popup arrive translated by
	// exactly this.
	Origin() paintengine2d.Point
	// Root is the top-level window the popup belongs to.
	Root() Surface
}

// PopupWorkArea is an optional capability: the work area of the monitor a
// window is on — the screen less its panels — in logical pixels relative
// to the window's visible box, which is where its popups may go. X11 and
// Offscreen know it; a Wayland client never learns where its window is, so
// there the compositor places popups on its own.
type PopupWorkArea interface {
	PopupWorkArea() (FrameRect, bool)
}

// SurfacePopupWorkArea is s's work area, where it can say.
func SurfacePopupWorkArea(s Surface) (FrameRect, bool) {
	if a, ok := s.(PopupWorkArea); ok {
		return a.PopupWorkArea()
	}
	return FrameRect{}, false
}

// SolvePopup places a popup the way xdg_positioner does: at the anchor
// point, extending in the direction of gravity, and then — axis by axis,
// wherever the result runs out of area — flipped to the other side of the
// anchor if that fits, slid back inside, and finally shrunk, as far as
// p.Adjust allows each. area is the monitor's work area in the same
// coordinates as p; an empty area constrains nothing.
//
// It is the X11 and offscreen backends' placement, and the reference the
// headless tests hold the Wayland compositor's answer to.
func SolvePopup(p PopupPlacement, area FrameRect) FrameRect {
	r := popupRect(p)
	if area.W < 1 || area.H < 1 {
		return r
	}
	fitsX := func(r FrameRect) bool { return r.X >= area.X && r.X+r.W <= area.X+area.W }
	fitsY := func(r FrameRect) bool { return r.Y >= area.Y && r.Y+r.H <= area.Y+area.H }
	if !fitsX(r) && p.Adjust&AdjustFlipX != 0 {
		q := p
		q.AnchorEdge, q.Gravity = flipEdgesX(q.AnchorEdge), flipEdgesX(q.Gravity)
		if f := popupRect(q); fitsX(f) {
			r.X = f.X
		}
	}
	if !fitsX(r) && p.Adjust&AdjustSlideX != 0 {
		if r.X+r.W > area.X+area.W {
			r.X = area.X + area.W - r.W
		}
		if r.X < area.X {
			r.X = area.X
		}
	}
	if !fitsX(r) && p.Adjust&AdjustResizeX != 0 {
		x0, x1 := max(r.X, area.X), min(r.X+r.W, area.X+area.W)
		if x1 > x0 {
			r.X, r.W = x0, x1-x0
		}
	}
	if !fitsY(r) && p.Adjust&AdjustFlipY != 0 {
		q := p
		q.AnchorEdge, q.Gravity = flipEdgesY(q.AnchorEdge), flipEdgesY(q.Gravity)
		if f := popupRect(q); fitsY(f) {
			r.Y = f.Y
		}
	}
	if !fitsY(r) && p.Adjust&AdjustSlideY != 0 {
		if r.Y+r.H > area.Y+area.H {
			r.Y = area.Y + area.H - r.H
		}
		if r.Y < area.Y {
			r.Y = area.Y
		}
	}
	if !fitsY(r) && p.Adjust&AdjustResizeY != 0 {
		y0, y1 := max(r.Y, area.Y), min(r.Y+r.H, area.Y+area.H)
		if y1 > y0 {
			r.Y, r.H = y0, y1-y0
		}
	}
	return r
}

// popupRect is p unconstrained: the anchor point, and the popup extending
// from it in the direction of gravity.
func popupRect(p PopupPlacement) FrameRect {
	a := p.Anchor
	x, y := a.X+a.W/2, a.Y+a.H/2
	switch {
	case p.AnchorEdge&EdgeLeft != 0:
		x = a.X
	case p.AnchorEdge&EdgeRight != 0:
		x = a.X + a.W
	}
	switch {
	case p.AnchorEdge&EdgeTop != 0:
		y = a.Y
	case p.AnchorEdge&EdgeBottom != 0:
		y = a.Y + a.H
	}
	switch {
	case p.Gravity&EdgeLeft != 0:
		x -= p.W
	case p.Gravity&EdgeRight == 0:
		x -= p.W / 2
	}
	switch {
	case p.Gravity&EdgeTop != 0:
		y -= p.H
	case p.Gravity&EdgeBottom == 0:
		y -= p.H / 2
	}
	return FrameRect{X: x, Y: y, W: max(p.W, 1), H: max(p.H, 1)}
}

func flipEdgesX(e Edges) Edges {
	switch {
	case e&EdgeLeft != 0:
		return e&^EdgeLeft | EdgeRight
	case e&EdgeRight != 0:
		return e&^EdgeRight | EdgeLeft
	}
	return e
}

func flipEdgesY(e Edges) Edges {
	switch {
	case e&EdgeTop != 0:
		return e&^EdgeTop | EdgeBottom
	case e&EdgeBottom != 0:
		return e&^EdgeBottom | EdgeTop
	}
	return e
}

// PopupOrigin is where a popup's buffer goes in its root window's buffer,
// device pixels: the root's visible box starts at winMin, the popup's
// visible box at placed (logical, relative to that), and the popup's own
// margin is left of and above it. Rounded, so the popup's pixels land on
// the root's grid and a menu is as crisp in its own surface as it was
// inside the window.
func PopupOrigin(winMin paintengine2d.Point, placed FrameRect, margin FrameInsets, scale float32) paintengine2d.Point {
	if scale <= 0 {
		scale = 1
	}
	x := winMin.X + float32(math.Round(float64(float32(placed.X)*scale))) - float32(margin.Left)
	y := winMin.Y + float32(math.Round(float64(float32(placed.Y)*scale))) - float32(margin.Top)
	return paintengine2d.Pt(x, y)
}

// popupInput reports whether a popup hands ev to its root window: input —
// the pointer, the keyboard, text — and nothing about the popup's own
// surface (its size, its configure, its close), which is the backend's
// business and would read as the root's own if it were passed on.
func popupInput(k EventKind) bool {
	switch k {
	case EventMouseDown, EventMouseUp, EventMouseMove, EventScroll, EventPointerLeave,
		EventKeyDown, EventKeyUp, EventText, EventFocusIn, EventFocusOut,
		EventIMEPreedit, EventIMECommit, EventIMECancel:
		return true
	}
	return false
}

// popupEvent is ev from a popup, as its root window hears it: every
// position moved by the popup's origin.
func popupEvent(ev Event, origin paintengine2d.Point) Event {
	switch ev.Kind {
	case EventMouseDown, EventMouseUp, EventMouseMove, EventScroll:
		ev.Pos = paintengine2d.Pt(ev.Pos.X+origin.X, ev.Pos.Y+origin.Y)
	}
	return ev
}
