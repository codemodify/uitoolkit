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

// OverlayHost is implemented by app.Window: modal dimmed dialogs.
type OverlayHost interface {
	Host
	SetOverlay(c Component)
	Overlay() Component
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

// ShowOverlay places c on the window overlay layer.
func ShowOverlay(from Component, overlay Component) bool {
	if from == nil || overlay == nil {
		return false
	}
	h := from.Host()
	if h == nil {
		return false
	}
	oh, ok := h.(OverlayHost)
	if !ok {
		return false
	}
	overlay.SetHost(h)
	oh.SetOverlay(overlay)
	return true
}

// DismissOverlay closes the window overlay layer, if any.
func DismissOverlay(from Component) {
	if from == nil {
		return
	}
	h := from.Host()
	if h == nil {
		return
	}
	if oh, ok := h.(OverlayHost); ok {
		oh.SetOverlay(nil)
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

// PreparePopup attaches from's host (look, scale, popup layer) before Measure.
func PreparePopup(from, popup Component) {
	if from == nil || popup == nil {
		return
	}
	if h := from.Host(); h != nil {
		popup.SetHost(h)
	}
}

func popupSurface(c Component) (SurfaceSizer, bool) {
	if c == nil {
		return nil, false
	}
	h := c.Host()
	if h == nil {
		return nil, false
	}
	sz, ok := h.(SurfaceSizer)
	return sz, ok
}

func popupCap(popup Component, maxW, maxH float32) (float32, float32) {
	if maxW <= 0 {
		maxW = 320
	}
	if maxH <= 0 {
		maxH = 480
	}
	if sz, ok := popupSurface(popup); ok {
		ww, hh := sz.SurfaceSize()
		if ww > 8 {
			maxW = float32(ww) - 8
		}
		if hh > 8 {
			maxH = float32(hh) - 8
		}
	}
	return maxW, maxH
}

// PlacePopupForAnchor sizes popup (host look already applied) and places it
// below anchor, or above when the surface has more room that way. minW is
// typically the closed ComboBox width. gap is space between anchor and
// popup (0 = flush, never overlapping).
func PlacePopupForAnchor(from, popup Component, anchor paintengine2d.Rect, minW, gap float32) {
	if popup == nil {
		return
	}
	PreparePopup(from, popup)
	if gap < 0 {
		gap = 0
	}
	intrinsic := popup.Measure(layout.Unbounded())
	w, h := intrinsic.X, intrinsic.Y
	if w < minW {
		w = minW
	}
	inset := float32(4)
	surfW, surfH := float32(0), float32(0)
	if sz, ok := popupSurface(popup); ok {
		ww, hh := sz.SurfaceSize()
		if ww > 8 {
			surfW = float32(ww)
		}
		if hh > 8 {
			surfH = float32(hh)
		}
	}

	maxX := surfW - inset
	if surfW <= 0 {
		maxX = w + inset
	}
	availW := maxX - inset
	if surfW > 0 && w > availW && availW > 1 {
		w = availW
	}
	x := anchor.Min.X
	if x+w > maxX {
		x = maxX - w
	}
	if x < inset {
		x = inset
	}

	below, above := float32(1e9), float32(1e9)
	if surfH > 0 {
		below = surfH - inset - (anchor.Max.Y + gap)
		above = anchor.Min.Y - gap - inset
		if below < 0 {
			below = 0
		}
		if above < 0 {
			above = 0
		}
	}
	placeBelow := true
	if h > below && above > below {
		placeBelow = false
	}
	availH := below
	if !placeBelow {
		availH = above
	}
	if surfH > 0 && h > availH && availH >= 1 {
		h = availH
		sz := popup.Measure(layout.Constraints{MaxW: -1, MaxH: h})
		if sz.X > w {
			w = sz.X
			if surfW > 0 && w > availW {
				w = availW
			}
		}
		if sz.Y > 0 && sz.Y < h {
			h = sz.Y
		}
	}

	y := anchor.Max.Y + gap
	if !placeBelow {
		y = anchor.Min.Y - gap - h
	}
	if y < inset {
		y = inset
	}
	if surfH > 0 && y+h > surfH-inset {
		y = surfH - inset - h
		if y < inset {
			y = inset
		}
	}
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}

// PlacePopup sizes popup to its intrinsic Measure and positions its top-left
// at origin (window space). maxW/maxH are fallbacks when the host surface
// size is unknown; a known surface wins so HiDPI menus are not capped at
// 1× design pixels.
func PlacePopup(popup Component, origin paintengine2d.Point, maxW, maxH float32) {
	if popup == nil {
		return
	}
	intrinsic := popup.Measure(layout.Unbounded())
	w, h := intrinsic.X, intrinsic.Y
	capW, capH := popupCap(popup, maxW, maxH)
	shrink := false
	if w > capW {
		w = capW
		shrink = true
	}
	if h > capH {
		h = capH
		shrink = true
	}
	if shrink {
		cons := layout.Loose(w, h)
		if intrinsic.X <= capW {
			cons.MaxW = -1
		}
		sz := popup.Measure(cons)
		w, h = sz.X, sz.Y
		if w > capW {
			w = capW
		}
		if h > capH {
			h = capH
		}
	}
	popup.Arrange(paintengine2d.XYWH(origin.X, origin.Y, w, h))
}

// ClampToSurface keeps popup inside the host surface. It repositions first
// (prefer full intrinsic size). Only if the popup is larger than the
// surface does it shrink the arranged box — the popup should then scroll
// rather than crop rows.
func ClampToSurface(from, popup Component) {
	if from == nil || popup == nil {
		return
	}
	sz, ok := popupSurface(from)
	if !ok {
		sz, ok = popupSurface(popup)
	}
	if !ok {
		return
	}
	ww, hh := sz.SurfaceSize()
	if ww < 1 || hh < 1 {
		return
	}
	inset := float32(4)
	maxX := float32(ww) - inset
	maxY := float32(hh) - inset
	availW := maxX - inset
	availH := maxY - inset
	if availW < 1 {
		availW = 1
	}
	if availH < 1 {
		availH = 1
	}

	b := popup.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > availW {
		w = availW
	}
	if h > availH {
		h = availH
	}
	x, y := b.Min.X, b.Min.Y
	if x+w > maxX {
		x = maxX - w
	}
	if y+h > maxY {
		y = maxY - h
	}
	if x < inset {
		x = inset
	}
	if y < inset {
		y = inset
	}
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}
