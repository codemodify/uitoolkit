package style

import (
	"github.com/codemodify/paintengine2d"
)

// A look's silhouette: the shape of the window it frames, and the shape of
// the controls it paints.
//
// Everything about cutting a window to a silhouette is in platform and app
// (docs/shapes.md). This file is the half a *look* owns: two optional engine
// hooks and the two accessors that apply the toolkit's rules to them, so an
// engine states its era's outline and nothing else — exactly as
// [DecorationEngine] states its era's frame and [DecorationOf] drops the
// corners and the shadow where they make no sense.
//
// The type is a description rather than a platform.Shape because platform
// imports style, not the other way round: a look says what its outline *is*
// and the layer that owns the window system turns it into a region
// (platform.NewShapeSilhouette). That is also why nothing here rasterises —
// a look is asked for its silhouette on every layout, and building a shape
// is the expensive half.

// Silhouette is an outline a look declares, in device pixels, with its
// origin at the top-left of whatever it describes: the visible window for
// [WindowShapeEngine], the control's own box for [ControlShapeEngine].
//
// It is a path or a mask, never both, and the two are not interchangeable:
//
//   - A path is resolution-independent. It is rasterised at the size the
//     window or control actually is, so it is exact at 1.75 as at 1. This is
//     what a look that can describe its outline wants, which is nearly all
//     of them — BeOS's tab, a skin's union of rounded rectangles.
//   - A Mask is artwork whose alpha channel *is* the outline: a skin whose
//     window is a picture, or a control face cut from a sprite. It is
//     resampled to the size it is asked for, as any bitmap is.
type Silhouette struct {
	// Path is the outline, or nil for a mask-backed silhouette.
	Path *paintengine2d.Path
	// EvenOdd fills Path even-odd, so a contour inside another one becomes
	// a hole that can be seen and clicked through. A simple outline wants
	// the default (nonzero winding), which is also what makes a union of
	// overlapping rectangles one silhouette rather than a set of holes.
	EvenOdd bool
	// Mask is an image whose alpha channel is the coverage (nil for a
	// path). Both of paintengine2d's formats work: a coverage mask as it
	// is, a premultiplied RGBA image by its alpha.
	Mask *paintengine2d.Image
}

// Empty reports whether the silhouette describes nothing at all — which is
// not the same as a nil silhouette, meaning "no silhouette, the plain
// rectangle every window and every control has by default".
func (s *Silhouette) Empty() bool {
	if s == nil {
		return true
	}
	if s.Mask != nil {
		return s.Mask.Width < 1 || s.Mask.Height < 1
	}
	return s.Path == nil || s.Path.Empty()
}

// SilhouetteRect is one rounded rectangle of an outline, its corner radii
// top-left clockwise.
type SilhouetteRect struct {
	Rect   paintengine2d.Rect
	Radius [4]float32
}

// SilhouetteOfRects is the outline a union of rounded rectangles encloses: a
// tab above a body, a head above a narrower panel, a plain rounded window.
// It is the form a look's outline nearly always takes, and the form both
// consumers want anyway — a compositor region is a rectangle list and
// hit-testing one is a loop.
//
// The union is filled by the nonzero winding rule, so two rectangles that
// overlap are one silhouette. Even-odd would turn the overlap into a hole,
// which is why a look that wants a hole says so itself.
func SilhouetteOfRects(rects ...SilhouetteRect) *Silhouette {
	p := paintengine2d.NewPath()
	n := 0
	for _, r := range rects {
		b := r.Rect.Canon()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			continue
		}
		p.AddRoundRectCorners(b, r.Radius[0], r.Radius[1], r.Radius[2], r.Radius[3])
		n++
	}
	if n == 0 {
		return nil
	}
	return &Silhouette{Path: p}
}

// ---- the window ------------------------------------------------------------

// WindowShapeEngine is an optional engine hook: the silhouette a top-level
// window in this look really has, so the desktop shows through everywhere it
// is not and a press there lands on whatever is behind. An engine without it
// frames a rectangle, which is every look that does not say otherwise.
//
// f is the frame's geometry with the visible window at the *origin* — f.Window
// is (0, 0, width, height) in device pixels and f.Caption is the caption band
// inside it — so an engine states its outline in the same coordinates it is
// handed, and the window moves it by the shadow's margin on the way to the
// compositor. The look's scale is l.Scale().
//
// The caption box is there because it is the one part of the frame whose
// extent a look cannot work out for itself: BeOS's tab is as wide as its
// title, and the title is the window's. Stating the outline against the box
// the caption was actually given is what keeps the tab that is painted and
// the tab that takes clicks the same tab.
type WindowShapeEngine interface {
	WindowShape(l *Classic, f DecorationFrame, st DecorationState) *Silhouette
}

// WindowShapeOf is lk's window silhouette in state st, or nil for the plain
// rectangle every window has by default.
//
// The state's rules are applied here, for every look alike, and they are the
// rules [Window.ShapeActive] already follows for an app's own shape: a
// maximized or tiled window fills a box the desktop chose and shares its
// edges with the screen or a neighbour, so a silhouette there would leave
// gaps the user cannot click and the desktop did not expect. (A full-screen
// window has no toolkit frame at all, so it never gets this far.)
//
// An uncomposited screen keeps its silhouette, unlike the corners and the
// shadow beside it: X11's bounding shape still cuts the window there, which
// is the one case where cutting is the *only* thing that works — per-pixel
// alpha counts for nothing without a compositing manager. The result is a
// hard-edged version of the same outline, exactly as docs/shapes.md promises
// for an app's own shape.
func WindowShapeOf(lk LookAndFeel, f DecorationFrame, st DecorationState) *Silhouette {
	if lk == nil || f.Window.Empty() || st.Maximized || st.Tiled != 0 {
		return nil
	}
	c, e := decorationFor(lk)
	sh, ok := e.(WindowShapeEngine)
	if !ok {
		return nil
	}
	s := sh.WindowShape(c, f, st)
	if s.Empty() {
		return nil
	}
	return s
}

// WindowShaped reports whether lk's frame declares a silhouette of its own
// at all, whatever the state — what an app asks to know whether a window in
// this look will ever be cut to a shape.
func WindowShaped(lk LookAndFeel) bool {
	if lk == nil {
		return false
	}
	_, e := decorationFor(lk)
	_, ok := e.(WindowShapeEngine)
	return ok
}

// ---- controls ---------------------------------------------------------------

// ControlShapeEngine is an optional engine hook: the silhouette of the face
// this look paints for a role in box b, so a control that looks round takes
// the pointer where it looks round. An engine without it paints faces that
// fill their box, which is every look that does not say otherwise — and the
// box is what a component takes input across, as it always has.
//
// b is the control's box with its origin at (0, 0), in device pixels.
//
// There is deliberately no ControlState here. A hit area that changed with
// the pointer would decide its own input: a hover face one pixel smaller
// than the resting one leaves a ring of pixels where the control hovers,
// stops covering the pointer, unhovers, and hovers again. The silhouette is
// the control's resting face, and the states are free to be any size.
type ControlShapeEngine interface {
	ControlShape(l *Classic, b paintengine2d.Rect, role Role) *Silhouette
}

// ControlShapeOf is lk's silhouette for the face of role in box b, or nil
// for the whole box — the default, and the answer for all 121 packs.
func ControlShapeOf(lk LookAndFeel, b paintengine2d.Rect, role Role) *Silhouette {
	if lk == nil || b.Empty() {
		return nil
	}
	c, ok := lk.(*Classic)
	if !ok || c == nil {
		return nil
	}
	e, ok := c.eng().(ControlShapeEngine)
	if !ok {
		return nil
	}
	s := e.ControlShape(c, b, role)
	if s.Empty() {
		return nil
	}
	return s
}

// ControlShapes reports whether lk's engine shapes its controls at all: the
// one question a component asks before doing any work for a hit test.
func ControlShapes(lk LookAndFeel) bool {
	c, ok := lk.(*Classic)
	if !ok || c == nil {
		return false
	}
	_, ok = c.eng().(ControlShapeEngine)
	return ok
}
