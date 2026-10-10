package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// glassLightStep is how far the pointer travels before the light on glass
// is moved, in logical pixels.
//
// Moving it repaints every piece of glass in the window, so it is not
// worth doing for a pixel. Eight is under the width of the specular rim
// itself, so the highlight slides rather than steps, and it cuts the
// repaints a slow drag across a window causes by an order of magnitude.
const glassLightStep = 8

// trackGlassLight moves the light on glass to the pointer.
//
// Only for a look that asks for glass: on the other hundred and thirty
// packs there is no rim for it to light and the repaint would buy
// nothing. The position handed to style is in device pixels, because
// that is the only space an engine painting some component's local rect
// can get back to (style.GlassRimPaint).
func (w *Window) trackGlassLight(pos paintengine2d.Point) {
	if w == nil || !style.WantsGlass(w.Look()) {
		return
	}
	if w.litAt != (paintengine2d.Point{}) {
		dx, dy := pos.X-w.litAt.X, pos.Y-w.litAt.Y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx < glassLightStep && dy < glassLightStep {
			return
		}
	}
	w.litAt = pos
	s := w.Scale()
	style.SetGlassLight(paintengine2d.Pt(pos.X*s, pos.Y*s))
	w.fullInvalidate()
}

// glassLightLeft puts the light back overhead when the pointer leaves a
// window whose look wants glass: a highlight frozen wherever the pointer
// happened to go out is worse than none.
func (w *Window) glassLightLeft() {
	if w == nil || w.litAt == (paintengine2d.Point{}) {
		return
	}
	w.litAt = paintengine2d.Point{}
	style.ClearGlassLight()
	if style.WantsGlass(w.Look()) {
		w.fullInvalidate()
	}
}
