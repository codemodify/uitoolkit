package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
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
// The toolkit's own frame turns with it. It is drawn outside the
// content's box and every component is clipped to its own box, so the
// panel cannot reach it: a framed window turns as a *picture* of itself
// instead, taken with each face up and turned over everything the window
// draws (turnWholeWindow, below). That is what keeps the resize edges,
// the minimise and the maximise — an application used to have to give
// all three up to get the turn, and draw a caption of its own on each
// face.
//
// The **desktop's** frame cannot turn, because the toolkit does not draw
// it: a window that keeps the desktop's title bar turns its contents
// inside a frame that stays still, which looks like a mistake. Use it
// with the toolkit's own decorations or none at all — [SetFlipFaces]
// says so if the window is wearing the desktop's.

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
//
// A window wearing the toolkit's own frame turns as one picture, frame
// and all; one without turns its content, which is all there is of it.
// Either way the card is the whole window and the window's shape follows
// it, so what the eye sees rotate is the window.
func (w *Window) Flip() {
	if w == nil || w.flip == nil || w.Turning() {
		return
	}
	if !w.turnsWhole() {
		w.flip.Flip()
		return
	}
	w.turnWholeWindow()
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

// ---- turning a framed window ------------------------------------------------

// A window wearing the toolkit's own frame turns over *with* it.
//
// The panel alone cannot do that. It is the window's content, every
// component is clipped to its own box, and the frame is drawn outside
// that box — so a turn drawn by the panel left the title bar, the
// buttons and the border standing still while the inside rotated, which
// reads as a mistake rather than a turn. The way round it was to give up
// the toolkit's frame and have each face draw a caption of its own, with
// no minimise, maximise or resizing, which is the trade an application
// reported rather than accepted.
//
// So a framed window turns as one picture: the window is photographed
// with each face up, and the two pictures are turned over the whole
// window — frame, caption, border and all — while the window's shape
// follows the card, exactly as it does for a frameless one. The content
// underneath has already swapped to the face being turned to, so when
// the picture goes there is nothing to put back.
//
// Two full repaints at the start of a turn is what that costs. It is
// paid once per flip, by a gesture that takes four hundred milliseconds.

// flipShot is one of those photographs: a component that draws an image
// over its box and nothing else.
type flipShot struct {
	widget.Base
	img *paintengine2d.Image
}

func newFlipShot(img *paintengine2d.Image) *flipShot {
	s := &flipShot{img: img}
	s.Init(s)
	return s
}

// wipe zeroes the photograph. A picture of the window is a copy of
// whatever the window showed, which for a password manager turning over
// with an item open is a copy of the password (secretvault item 78).
func (s *flipShot) wipe() {
	if s != nil && s.img != nil {
		clear(s.img.Pix)
		s.img = nil
	}
}

func (s *flipShot) Paint(ctx *paintengine2d.Context) {
	b := s.LocalBounds()
	if s.img == nil || b.Empty() {
		return
	}
	src := paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height))
	ctx.DrawImageRect(s.img, src, b)
}

// Turning reports whether the window is in the middle of a turn, whether
// that is the whole window (a framed one) or the card inside it.
func (w *Window) Turning() bool {
	if w == nil {
		return false
	}
	if w.turn != nil && w.turn.Turning() {
		return true
	}
	return w.flip != nil && w.flip.Turning()
}

// turnsWhole reports whether a flip should turn the window as one
// picture: there is a frame of the toolkit's to turn, and motion is
// wanted at all (with it off the faces swap and there is nothing to
// photograph).
func (w *Window) turnsWhole() bool {
	return w.geom.framed && widget.Animates(w.flip) && !w.windowBox().Empty()
}

// turnWholeWindow photographs the window with each face up and turns the
// two pictures over everything the window draws.
func (w *Window) turnWholeWindow() {
	front := w.windowShot()
	if front == nil {
		w.flip.Flip()
		return
	}
	// The content swaps underneath without a turn of its own: what the
	// eye is given is the picture over it, and the live face is ready
	// the moment that picture goes. ShowNow rather than a motion
	// setting, because UITK_ANIMATIONS=1 beats a setting — and then the
	// content would turn inside the picture of itself turning.
	w.flip.ShowNow(!w.flip.Showing())
	back := w.windowShot()
	if back == nil {
		return
	}

	p := widgets.NewFlipPanel(newFlipShot(front), newFlipShot(back))
	p.Perspective, p.Strips = w.flip.Perspective, w.flip.Strips
	// The decision was taken above, by the panel this one stands in for.
	p.SetMotion(style.MotionAlways)
	p.SetHost(w)
	p.OnFrame = w.followWindowTurn
	w.turn = p
	w.arrangeTurn()
	p.Flip()
	w.fullInvalidate()
}

// windowShot photographs the window as it looks now: a frame is painted
// and the window's own box is copied out of the surface, without the
// shadow margin around it.
func (w *Window) windowShot() *paintengine2d.Image {
	w.laid = false
	w.fullInvalidate()
	w.frame()
	buf := w.surf.Buffer()
	box := w.windowBox()
	if buf == nil || box.Empty() {
		return nil
	}
	x0, y0, x1, y1 := box.IntBounds()
	iw, ih := x1-x0, y1-y0
	if iw < 1 || ih < 1 {
		return nil
	}
	img := paintengine2d.NewImage(iw, ih)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawImageRect(buf,
		paintengine2d.XYWH(float32(x0), float32(y0), float32(iw), float32(ih)),
		paintengine2d.XYWH(0, 0, float32(iw), float32(ih)))
	img.Touch()
	return img
}

// arrangeTurn gives the turning picture the whole window: the frame is
// in it, so its box is the window and not the content.
func (w *Window) arrangeTurn() {
	if w.turn == nil {
		return
	}
	b := w.windowBox()
	_ = w.turn.Measure(layout.Tight(b.Dx(), b.Dy()))
	w.turn.Arrange(b)
}

// followWindowTurn puts the window's shape on the turning picture, and
// takes the picture down when the turn is over.
func (w *Window) followWindowTurn() {
	if w == nil || w.turn == nil {
		return
	}
	if !w.turn.Turning() {
		// The panel wipes its own rasterised copies; the photographs
		// under them are this window's to wipe (secretvault item 78).
		w.turn.Wipe()
		for _, c := range []widget.Component{w.turn.Front(), w.turn.Back()} {
			if shot, ok := c.(*flipShot); ok {
				shot.wipe()
			}
		}
		w.turn = nil
		w.clearFlipShape()
		w.laid = false
		w.fullInvalidate()
		return
	}
	if !w.flipShapes {
		return
	}
	outline := w.turn.CardOutline()
	if outline == nil {
		w.SetShape(platform.NewShape(paintengine2d.NewPath()))
		w.flipShaped = true
		return
	}
	// No translation: the picture is arranged at the window's box, so
	// its own coordinates start at the window's corner — which is where
	// a shape is stated from. Moving it by the shadow margin, as the
	// content panel has to, left a strip of the still window showing
	// down the near edge of the turning one.
	w.SetShape(platform.NewShape(outline))
	w.flipShaped = true
}
