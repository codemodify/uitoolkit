package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// PopupHost is implemented by app.Window: floating menus without a dimmer.
type PopupHost interface {
	Host
	SetPopup(c Component)
	Popup() Component
	DismissPopup()
}

// PointerRetain is chrome that should receive the click that would otherwise
// only dismiss a popup (MenuBar titles).
type PointerRetain interface {
	RetainsPointer() bool
}

// Dismisser is notified when a popup is taken down.
type Dismisser interface {
	Dismissed()
}

// ShowPopup places c on the window popup layer. c should already be Arranged
// in window coordinates.
func ShowPopup(from Component, popup Component) bool {
	if from == nil || popup == nil {
		return false
	}
	h := from.Host()
	if h == nil {
		return false
	}
	ph, ok := h.(PopupHost)
	if !ok {
		return false
	}
	popup.SetHost(h)
	ph.SetPopup(popup)
	return true
}

// DismissPopup closes the window popup layer, if any.
func DismissPopup(from Component) {
	if from == nil {
		return
	}
	h := from.Host()
	if h == nil {
		return
	}
	if ph, ok := h.(PopupHost); ok {
		ph.DismissPopup()
	}
}

// Retains reports whether c or an ancestor wants the dismiss-click delivered.
func Retains(c Component) bool {
	for n := c; n != nil; n = n.Parent() {
		if r, ok := n.(PointerRetain); ok && r.RetainsPointer() {
			return true
		}
	}
	return false
}

// SurfaceSizer is implemented by app.Window so popups can stay on-screen.
type SurfaceSizer interface {
	SurfaceSize() (int, int)
}

// PlacePopup sizes popup and positions its top-left at origin (window space).
func PlacePopup(popup Component, origin paintengine2d.Point, maxW, maxH float32) {
	if popup == nil {
		return
	}
	if maxW <= 0 {
		maxW = 320
	}
	if maxH <= 0 {
		maxH = 480
	}
	sz := popup.Measure(layout.Loose(maxW, maxH))
	popup.Arrange(paintengine2d.XYWH(origin.X, origin.Y, sz.X, sz.Y))
}

// ClampToSurface shifts popup so it stays inside the host surface.
func ClampToSurface(from, popup Component) {
	if from == nil || popup == nil {
		return
	}
	h := from.Host()
	sz, ok := h.(SurfaceSizer)
	if !ok {
		return
	}
	ww, hh := sz.SurfaceSize()
	if ww < 1 || hh < 1 {
		return
	}
	b := popup.Bounds()
	dx, dy := float32(0), float32(0)
	if b.Max.X > float32(ww)-4 {
		dx = float32(ww) - 4 - b.Max.X
	}
	if b.Max.Y > float32(hh)-4 {
		dy = float32(hh) - 4 - b.Max.Y
	}
	if b.Min.X+dx < 4 {
		dx = 4 - b.Min.X
	}
	if b.Min.Y+dy < 4 {
		dy = 4 - b.Min.Y
	}
	if dx != 0 || dy != 0 {
		popup.Arrange(b.Translate(paintengine2d.Pt(dx, dy)))
	}
}
