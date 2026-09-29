package app

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// Sizing a window to what is in it.
//
// WindowOptions wants a size before the window exists, and a dialog
// whose height depends on its text — a description of one line or of
// five — does not know it yet. The workaround was to build the content,
// give it the application's look, measure it, throw the look away, and
// then add the caption's height by hand wherever the toolkit draws the
// frame. Two of those steps are guesses: the application's scale is not
// the window's on a multi-monitor desktop, and the caption's height is
// the look's business rather than the caller's.
//
// FitToContent measures at the window's own scale, through the window's
// own frame, which is the only place both are known.

// FitToContent makes the window as tall as its content, keeping its
// width, and reports whether it asked.
//
// The height it asks for is the content's measured height plus whatever
// chrome this window wears: none where the desktop draws the frame, the
// caption and borders where the toolkit draws them. A fixed-size dialog
// sized this way has no clipped last row and no empty band under it, at
// any display scale.
//
// It is a request, like every other size: a desktop may refuse it, and a
// window whose content is taller than the screen gets what the desktop
// allows.
func (w *Window) FitToContent() bool {
	if w == nil || w.Closed() || w.root == nil {
		return false
	}
	lw, h, ok := w.contentFit()
	if !ok {
		return false
	}
	w.SetSize(lw, h)
	w.recenterIfCentred()
	w.laid = false
	return true
}

// recenterIfCentred puts the window back in the middle after its size
// has changed, if the middle is where it was put.
//
// A resize keeps the top-left corner, so a dialog that was centred at
// the size it was made with is off-centre at the size it is fitted to —
// by half the difference, which for a dialog that grows from a guessed
// height to a measured one is most of the change. Centring and fitting
// are the two halves of the same request ("a dialog, in front, the right
// size"), and a caller has no way to sequence them itself: the fit is
// what changes the size, and it happens inside the toolkit.
//
// A window the user has since moved is not re-centred, because the
// desktop owns its position once the user has expressed one — but the
// toolkit cannot see a move on every backend, so this is best-effort and
// only ever runs on a window the toolkit centred itself.
func (w *Window) recenterIfCentred() {
	if w == nil || !w.centred {
		return
	}
	w.Center()
}

// contentFit is the logical width and height a window fitted to its
// content would ask for.
func (w *Window) contentFit() (width, height int, ok bool) {
	// The frame first, so the content box below is the one this window
	// actually has rather than the one it had before its decorations
	// were decided.
	w.applyFrame()
	sw, sh := w.surf.Size()
	g := w.layoutFrame(paintengine2d.XYWH(0, 0, float32(sw), float32(sh)))
	if g.content.Dx() < 1 {
		return 0, 0, false
	}
	// The chrome is whatever the visible window has that the content box
	// does not: zero under a desktop frame, the caption and borders
	// under the toolkit's.
	chrome := w.windowBox().Dy() - g.content.Dy()
	want := w.root.Measure(layout.Constraints{
		MinW: g.content.Dx(), MaxW: g.content.Dx(), MaxH: -1,
	})
	sc := w.Scale()
	if sc <= 0 {
		sc = 1
	}
	lw, _ := w.Size()
	lh := int(math.Ceil(float64((want.Y + chrome) / sc)))
	if lh < 1 {
		lh = 1
	}
	return lw, lh, true
}

// fitOnFirstLayout honours [platform.WindowOptions.FitContent] once,
// the first time the window is laid out with content in it.
//
// The option cannot be applied when the window is made, because the
// content is set afterwards; this is the first moment both the content
// and the window's own scale and frame exist, which is exactly what the
// hand-written version of this could not have at the same time.
func (w *Window) fitOnFirstLayout() {
	if w == nil || !w.wantFit || w.root == nil {
		return
	}
	w.wantFit = false
	lw, lh, ok := w.contentFit()
	if !ok {
		return
	}
	w.SetSize(lw, lh)
	w.recenterIfCentred()
	// The window just changed size under a layout that has finished, so
	// the tree is arranged for the box it used to have. Lay it out again
	// here rather than waiting for the backend's resize event, so the
	// very first frame the window paints is the right one — a dialog
	// that showed its old size for a frame is a flash the user sees.
	//
	// It recurses exactly once: wantFit is already false, so the second
	// pass returns at the guard above.
	w.laid = false
	w.fullInvalidate()
	w.layout()
}
