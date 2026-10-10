package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The whole window turns over, as against a panel inside it turning over.
//
// There are two of these on purpose. A [widgets.FlipPanel] turns one part
// of a window — a card in a list, a pane beside others — and the window
// around it does not move. This turns the application: everything the
// window holds goes on the front, the settings go on the back, and what
// rotates is the window.
//
// **A toplevel window cannot be rotated by the program that owns it.**
// Neither X11 nor Wayland has a way to hand the display server a
// transform for a window, and on Wayland the compositor owns the geometry
// outright — KWin can turn a window over as an effect of its own, but not
// because a client asked. So what turns here is everything the toolkit
// draws, and the window is made to follow it: the panel is the window's
// whole content, and while it turns the window's shape is set to the
// card's silhouette every frame, so the desktop shows through wherever
// the card is not. What the eye is given is a window rotating in place;
// what is actually happening is a rectangle whose visible part is a
// turning trapezoid.
//
// That has a consequence worth knowing before reaching for it: the
// desktop's own frame cannot turn, because the toolkit does not draw it.
// A window that keeps the desktop's title bar would turn its contents
// inside a frame that stayed still, which looks like a mistake. Use it
// with the toolkit's own decorations or none at all — [SetFlipFaces] says
// so if the window is wearing the desktop's.

// SetFlipFaces gives the window a front and a back and turns the whole
// window over between them. It replaces the window's content with the
// panel it returns, which the caller may go on to configure (its
// Perspective, its Strips, its OnFlip).
//
// The window follows the card with its own shape while it turns, so that
// what the eye sees rotate is the window. Pass false to
// [Window.SetFlipShapesWindow] to leave the window a plain rectangle and
// turn only what is drawn inside it.
func (w *Window) SetFlipFaces(front, back widget.Component) *widgets.FlipPanel {
	if w == nil {
		return nil
	}
	p := widgets.NewFlipPanel(front, back)
	w.flip = p
	w.flipShapes = true
	p.OnFrame = w.followFlip
	w.SetContent(p)
	return p
}

// FlipPanel is the window's two-sided content, or nil where it has none.
func (w *Window) FlipPanel() *widgets.FlipPanel {
	if w == nil {
		return nil
	}
	return w.flip
}

// Flip turns the window over to its other face.
func (w *Window) Flip() {
	if w != nil && w.flip != nil {
		w.flip.Flip()
	}
}

// ShowingBack reports whether the window's back is the face up — during a
// turn, the face being turned to.
func (w *Window) ShowingBack() bool {
	return w != nil && w.flip != nil && w.flip.Showing()
}

// SetFlipShapesWindow decides whether the window takes the turning card's
// silhouette as its shape.
//
// On, which is the default, the desktop shows through around the card and
// the window itself appears to rotate. Off, the window stays the
// rectangle it was and the card turns inside it, against whatever the
// window paints behind — which is the honest choice for a window wearing
// the desktop's own frame, since that frame cannot turn with it.
func (w *Window) SetFlipShapesWindow(on bool) {
	if w == nil || w.flipShapes == on {
		return
	}
	w.flipShapes = on
	if !on {
		w.clearFlipShape()
		return
	}
	w.followFlip()
}

// FlipShapesWindow reports whether the window takes the card's silhouette.
func (w *Window) FlipShapesWindow() bool { return w != nil && w.flipShapes }

// followFlip puts the window's shape on the card as it is drawn now. It
// runs at the end of every frame of a turn, and once when the turn ends.
func (w *Window) followFlip() {
	if w == nil || w.flip == nil || !w.flipShapes {
		return
	}
	if !w.flip.Turning() {
		w.clearFlipShape()
		return
	}
	outline := w.flip.CardOutline()
	if outline == nil {
		// Edge on: nothing is drawn, so the window shows nothing. An
		// empty shape rather than no shape, or the frame the card
		// vanished for would flash the whole rectangle back.
		w.SetShape(platform.NewShape(paintengine2d.NewPath()))
		w.flipShaped = true
		return
	}
	// The panel is the window's whole content, but it sits wherever the
	// content was placed: the outline is panel-local and the shape is in
	// the window's own space.
	if o := widget.DeviceOrigin(w.flip); o.X != 0 || o.Y != 0 {
		outline.Transform(paintengine2d.Translation(o.X, o.Y))
	}
	w.SetShape(platform.NewShape(outline))
	w.flipShaped = true
}

// clearFlipShape puts a rectangular window back, and only one this flip
// shaped: a window with a silhouette of its own keeps it.
func (w *Window) clearFlipShape() {
	if w == nil || !w.flipShaped {
		return
	}
	w.flipShaped = false
	w.SetShape(nil)
}
