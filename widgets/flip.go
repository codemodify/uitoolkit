package widgets

import (
	"math"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// flipNow is the clock a turn runs by (tests replace it).
var flipNow = time.Now

const (
	// flipFrame is how often a running turn repaints, as fades do.
	flipFrame = 16 * time.Millisecond
	// flipDuration is how long a turn takes. Long enough to read as a
	// rotation rather than a cut, short enough that the face is a picture
	// for less time than a click takes.
	flipDuration = 420 * time.Millisecond
	// flipPerspectiveDefault is the camera distance in half-widths. At
	// 2.5 the near edge grows by about a third coming through 60°, which
	// is plainly a turn in depth without the fish-eye of a close camera.
	flipPerspectiveDefault = 2.5
	// flipStripsDefault is how many columns the face is cut into. The
	// error inside a column goes as the square of its width, so at a
	// 1200-pixel face these are under 10 pixels wide and the residual is
	// far inside a pixel; 128 axis-aligned blits is nothing next to
	// painting the face that made them.
	flipStripsDefault = 128
)

// FlipPanel shows one of two faces and turns about its vertical axis to
// show the other — the gesture Dashboard's widgets used to put their
// settings on the back of the thing they configured.
//
// **At rest the face that is up is an ordinary live subtree**: it lays
// out, paints, takes the pointer, holds focus and reaches assistive
// technology exactly as it would without the panel, because that is what
// it is. Only for the length of the turn is either face a picture. That
// is the whole point of the design — a settings page you cannot click is
// not a settings page — and it is why the two faces are measured
// together and given the same box: a card has one size, whichever way up
// it is.
//
// The face that is down is hidden, and [widget.Walk] stops at a hidden
// component, so one flag takes its whole subtree out of painting,
// hit-testing and the tab order together. Nothing on the back of the card
// can be tabbed into or read out while it is face down.
//
// Perspective and Strips are the projection's two knobs; both have
// defaults and neither needs setting. The turn is skipped outright when
// the look says not to animate ([style.Animations]) or when nothing of
// the panel is on screen ([widget.Exposed]) — the faces still swap, just
// at once.
type FlipPanel struct {
	widget.Base

	front, back widget.Component
	// showingBack is which face is up once the panel is at rest. It
	// changes at the *start* of a turn, so Showing answers for where the
	// panel is going rather than where it has been.
	showingBack bool

	// turn is the running animation, zero when at rest.
	turn flipTurn

	// OnFlip is called when a turn finishes, or at once for a turn that
	// was skipped, with the face now up.
	OnFlip func(back bool)

	// Perspective is the camera distance in half-widths; 0 takes the
	// default. Smaller is a nearer camera and a deeper foreshortening.
	Perspective float32
	// Strips is how many columns the face is cut into; 0 takes the
	// default.
	Strips int
}

// flipTurn is the state of a turn in progress.
type flipTurn struct {
	running bool
	start   time.Time
	// from and to are the faces, rasterised once at the first frame: the
	// one going away and the one coming up.
	from, to *paintengine2d.Image
	// box is the device size from and to were rasterised at, so a scale
	// change mid-turn can be noticed.
	box paintengine2d.Point
	// pending is a scheduled repaint.
	pending bool
	// hadFocus records that the keyboard was inside the face going away,
	// so it can be handed to the one coming up (see end).
	hadFocus bool
}

// NewFlipPanel builds a panel showing front, with back on its other side.
// Either may be nil; a nil face is a blank side.
func NewFlipPanel(front, back widget.Component) *FlipPanel {
	f := &FlipPanel{}
	f.Init(f)
	// The panel paints its faces itself: at rest through widget.PaintTree
	// so the live face is painted exactly as the walker would, and
	// mid-turn as strips of a picture.
	f.SetManagesChildren(true)
	f.SetFront(front)
	f.SetBack(back)
	return f
}

// SetFront replaces the front face.
func (f *FlipPanel) SetFront(c widget.Component) { f.setFace(&f.front, c) }

// SetBack replaces the back face.
func (f *FlipPanel) SetBack(c widget.Component) { f.setFace(&f.back, c) }

func (f *FlipPanel) setFace(slot *widget.Component, c widget.Component) {
	if *slot == c {
		return
	}
	if *slot != nil {
		f.Remove(*slot)
	}
	*slot = c
	if c != nil {
		f.Add(c)
	}
	f.syncFaces()
	f.Invalidate()
}

// Front is the front face (nil if there is none).
func (f *FlipPanel) Front() widget.Component { return f.front }

// Back is the back face (nil if there is none).
func (f *FlipPanel) Back() widget.Component { return f.back }

// Showing reports whether the back is the face that is up — during a turn,
// the face being turned to.
func (f *FlipPanel) Showing() bool { return f.showingBack }

// Turning reports whether a turn is running.
func (f *FlipPanel) Turning() bool { return f.turn.running }

// Flip turns to the other face.
func (f *FlipPanel) Flip() { f.show(!f.showingBack) }

// ShowFront turns to the front, and does nothing if it is already up.
func (f *FlipPanel) ShowFront() { f.show(false) }

// ShowBack turns to the back, and does nothing if it is already up.
func (f *FlipPanel) ShowBack() { f.show(true) }

// show turns to a face, or swaps to it where there is to be no animation.
func (f *FlipPanel) show(back bool) {
	if back == f.showingBack && !f.turn.running {
		return
	}
	f.showingBack = back
	if !f.animates() {
		f.turn = flipTurn{}
		f.syncFaces()
		f.Invalidate()
		f.flipped()
		return
	}
	// A turn asked for mid-turn starts over rather than reversing from
	// where the face had got to: reversing from a partial angle would
	// need the rasterised faces swapped and the clock run backwards, and
	// a card that snaps back to flat and turns again is honest about
	// having been asked twice.
	f.turn = flipTurn{running: true, start: flipNow(), hadFocus: f.focusInside()}
	if f.turn.hadFocus {
		// The face about to go face down is where the keyboard is, and it
		// is about to become a picture. Park the focus; end hands it to
		// the face that comes up.
		if h := f.Host(); h != nil {
			h.RequestFocus(nil)
		}
	}
	f.syncFaces()
	f.Invalidate()
}

// animates reports whether a turn should be drawn rather than swapped: the
// look allows motion, and enough of the panel is on screen to be worth
// asking for frames.
func (f *FlipPanel) animates() bool {
	return style.Animations() && !f.LocalBounds().Empty() && widget.Exposed(f)
}

// syncFaces puts the visibility where the panel's state says it goes: the
// face that is up while at rest, neither while turning (both are pictures
// then, and a visible face would also be painted live underneath).
func (f *FlipPanel) syncFaces() {
	up, down := f.front, f.back
	if f.showingBack {
		up, down = f.back, f.front
	}
	if up != nil {
		up.SetVisible(!f.turn.running)
	}
	if down != nil {
		down.SetVisible(false)
	}
}

// focusInside reports whether the window's focus is within this panel.
func (f *FlipPanel) focusInside() bool {
	h := f.Host()
	if h == nil {
		return false
	}
	cur := h.Focus()
	return cur != nil && widget.Contains(f, cur)
}

// Measure is the larger of the two faces on each axis, so the card is one
// size whichever way up it is and a flip never moves the layout.
func (f *FlipPanel) Measure(c layout.Constraints) paintengine2d.Point {
	var w, h float32
	for _, face := range []widget.Component{f.front, f.back} {
		if face == nil {
			continue
		}
		// A hidden face still has to answer: it is measured for the box it
		// will get when it comes up. TabPage asks the same way.
		vis := face.Visible()
		face.SetVisible(true)
		sz := face.Measure(c)
		face.SetVisible(vis)
		if sz.X > w {
			w = sz.X
		}
		if sz.Y > h {
			h = sz.Y
		}
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

// Arrange gives both faces the whole box. The face that is down is laid
// out too, so it is ready to paint the moment a turn starts.
func (f *FlipPanel) Arrange(r paintengine2d.Rect) {
	f.SetBounds(r)
	box := paintengine2d.XYWH(0, 0, r.Dx(), r.Dy())
	for _, face := range []widget.Component{f.front, f.back} {
		if face == nil {
			continue
		}
		vis := face.Visible()
		face.SetVisible(true)
		face.Arrange(box)
		face.SetVisible(vis)
	}
}

// MousePress swallows a press while turning: a card in the middle of a
// turn is a picture and not a target, and letting the press bubble would
// hand it to whatever is behind the panel.
func (f *FlipPanel) MousePress(e widget.MouseEvent) bool {
	if f.turn.running {
		return true
	}
	return f.Base.MousePress(e)
}

func (f *FlipPanel) Paint(ctx *paintengine2d.Context) {
	if !f.turn.running {
		if up := f.upFace(); up != nil {
			widget.PaintTree(up, ctx, nil)
		}
		return
	}
	t, done := f.progress()
	if done {
		// The clock ran out before this frame: finish on the face itself
		// rather than drawing a 180° picture of it.
		f.end()
		if up := f.upFace(); up != nil {
			widget.PaintTree(up, ctx, nil)
		}
		return
	}
	if !f.rasterise(ctx) {
		// Nothing to turn (no room, or a face too large to rasterise):
		// end the turn and show the face.
		f.end()
		if up := f.upFace(); up != nil {
			widget.PaintTree(up, ctx, nil)
		}
		return
	}
	f.paintTurn(ctx, t)
	f.schedule()
}

// upFace is the face that is up, or will be once the turn ends.
func (f *FlipPanel) upFace() widget.Component {
	if f.showingBack {
		return f.back
	}
	return f.front
}

// progress is how far through the turn the clock is, eased, and whether
// it is over. The easing is symmetric — a turn that starts and stops
// gently, rather than the ease-out a fade uses, because both ends of a
// rotation are visible.
func (f *FlipPanel) progress() (float32, bool) {
	e := float32(flipNow().Sub(f.turn.start)) / float32(flipDuration)
	if e >= 1 {
		return 1, true
	}
	if e < 0 {
		e = 0
	}
	return e * e * (3 - 2*e), false
}

// rasterise paints both faces into images once per turn, at the device
// resolution ctx is drawing at. It reports whether there are two pictures
// to turn.
//
// The faces are drawn through [widget.PaintTree] into a context of their
// own, which is how a face keeps its own look, clipping and text
// rasterisation; a face is painted *once* for the whole turn rather than
// once per frame, which is what makes the turn cost the strips and not the
// tree.
func (f *FlipPanel) rasterise(ctx *paintengine2d.Context) bool {
	box := f.LocalBounds()
	if box.Empty() {
		return false
	}
	dev := ctx.Matrix().TransformRect(box)
	w := int(math.Ceil(float64(dev.Dx())))
	h := int(math.Ceil(float64(dev.Dy())))
	if w < 1 || h < 1 {
		return false
	}
	want := paintengine2d.Pt(float32(w), float32(h))
	if f.turn.from != nil && f.turn.to != nil && f.turn.box == want {
		return true
	}
	from, to := f.back, f.front
	if f.showingBack {
		from, to = f.front, f.back
	}
	f.turn.from = f.faceImage(ctx, from, w, h, dev.Min)
	f.turn.to = f.faceImage(ctx, to, w, h, dev.Min)
	f.turn.box = want
	return f.turn.from != nil && f.turn.to != nil
}

// faceImage paints one face into a new image of w x h device pixels. A nil
// face gives a transparent image, so a one-sided panel turns to a blank
// back rather than refusing to turn.
func (f *FlipPanel) faceImage(ctx *paintengine2d.Context, face widget.Component, w, h int, at paintengine2d.Point) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	if face == nil {
		return img
	}
	lc := paintengine2d.NewContext(img)
	// The face's own device scale, with the panel's device origin taken
	// off: the same matrix the walker would have given it, moved to the
	// image's corner.
	lc.SetMatrix(paintengine2d.Translation(-at.X, -at.Y).Mul(ctx.Matrix()))
	vis := face.Visible()
	face.SetVisible(true)
	widget.PaintTree(face, lc, nil)
	face.SetVisible(vis)
	return img
}

// paintTurn draws the card at eased progress t.
func (f *FlipPanel) paintTurn(ctx *paintengine2d.Context, t float32) {
	box := f.LocalBounds()
	angle := t * float32(math.Pi)
	g := flipGeomAt(box.Dx(), box.Dy(), f.Perspective, angle)
	if g.edgeOn() {
		return
	}
	// Past a quarter turn the far side is what faces the viewer. The
	// geometry mirrors itself there — cosθ goes negative, so x(u) runs
	// backwards — and sampling the back face's columns in reverse undoes
	// that, which is what makes the back read the right way round at 180°
	// rather than as its own mirror image.
	img, mirror := f.turn.from, false
	if t > 0.5 {
		img, mirror = f.turn.to, true
	}
	if img == nil {
		return
	}
	n := f.Strips
	if n <= 0 {
		n = flipStripsDefault
	}
	// Never more columns than the picture has pixels: past that, columns
	// share a source pixel and cost blits that draw the same thing.
	if n > img.Width {
		n = img.Width
	}
	if n < 1 {
		return
	}
	imgW, imgH := float32(img.Width), float32(img.Height)
	midY := box.Dy() / 2
	for i := 0; i < n; i++ {
		// The column's edges in source space, -1..1.
		u0 := 2*float32(i)/float32(n) - 1
		u1 := 2*float32(i+1)/float32(n) - 1
		x0, x1 := g.xAt(u0), g.xAt(u1)
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		if x1-x0 <= 0 {
			continue
		}
		// One scale for the column, taken at its middle: exact at the
		// edges in x, and the residual in y is the square of the column's
		// width (flipGeom).
		half := g.halfHAt((u0 + u1) / 2)
		if half <= 0 {
			continue
		}
		s0, s1 := u0, u1
		if mirror {
			s0, s1 = -u1, -u0
		}
		src := paintengine2d.XYWH((s0+1)/2*imgW, 0, (s1-s0)/2*imgW, imgH)
		dst := paintengine2d.XYWH(x0, midY-half, x1-x0, 2*half)
		ctx.DrawImageRect(img, src, dst)
	}
}

// schedule asks for the next frame, once.
func (f *FlipPanel) schedule() {
	if f.turn.pending {
		return
	}
	f.turn.pending = true
	widget.After(f, flipFrame, func() {
		f.turn.pending = false
		if f.turn.running {
			f.Invalidate()
		}
	})
}

// end puts the panel at rest on the face it was turning to.
func (f *FlipPanel) end() {
	if !f.turn.running {
		return
	}
	had := f.turn.hadFocus
	f.turn = flipTurn{}
	f.syncFaces()
	if had {
		// The keyboard was in the face that went down; give it to the one
		// that came up rather than leaving the window with no focus.
		if h := f.Host(); h != nil {
			if cands := widget.Focusables(f.upFace()); len(cands) > 0 {
				h.RequestFocus(cands[0])
			}
		}
	}
	f.flipped()
}

// flipped announces the face now up.
func (f *FlipPanel) flipped() {
	if f.OnFlip != nil {
		f.OnFlip(f.showingBack)
	}
}
