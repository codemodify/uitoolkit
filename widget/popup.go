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

// Presenter is a popup or overlay that wants to know when it is mounted on a
// layer, and by whom. Menus and modal dialogs use it to remember the focus
// owner so dismissal can hand focus back to the anchor instead of leaving a
// detached node focused.
type Presenter interface {
	Presented(from Component)
}

func notifyPresented(from, c Component) {
	if pr, ok := c.(Presenter); ok {
		pr.Presented(from)
	}
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
	notifyPresented(from, popup)
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
	notifyPresented(from, overlay)
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
	surfW, surfH := popupSurfaceWH(popup)

	// Vertical: flip or scroll (v0.10.8). Height may shrink; width must not
	// be reduced to the leftover strip beside the anchor.
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
		}
		if sz.Y > 0 && sz.Y < h {
			h = sz.Y
		}
		if w < minW {
			w = minW
		}
		if w < intrinsic.X {
			w = intrinsic.X
		}
	}

	x, w := shiftPopupX(anchor.Min.X, w, surfW, inset)
	y := anchor.Max.Y + gap
	if !placeBelow {
		y = anchor.Min.Y - gap - h
	}
	y, h = shiftPopupY(y, h, surfH, inset)
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
	_, capH := popupCap(popup, maxW, maxH)
	surfW, surfH := popupSurfaceWH(popup)
	if h > capH {
		sz := popup.Measure(layout.Constraints{MaxW: -1, MaxH: capH})
		w, h = sz.X, sz.Y
		if w < intrinsic.X {
			w = intrinsic.X
		}
		if h > capH {
			h = capH
		}
	}
	inset := float32(4)
	x, w := shiftPopupX(origin.X, w, surfW, inset)
	y, h := shiftPopupY(origin.Y, h, surfH, inset)
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}

// PlacePopupBeside sizes popup and places it to the right of anchor (a
// parent menu row in window space), flipping to the left when that side
// has more room. Top-aligned to the row, then clamped on-screen.
func PlacePopupBeside(from, popup Component, anchor paintengine2d.Rect) {
	if popup == nil {
		return
	}
	PreparePopup(from, popup)
	intrinsic := popup.Measure(layout.Unbounded())
	w, h := intrinsic.X, intrinsic.Y
	inset := float32(4)
	surfW, surfH := popupSurfaceWH(popup)

	right := float32(1e9)
	left := float32(1e9)
	if surfW > 0 {
		right = surfW - inset - anchor.Max.X
		left = anchor.Min.X - inset
		if right < 0 {
			right = 0
		}
		if left < 0 {
			left = 0
		}
	}
	placeRight := true
	if w > right && left > right {
		placeRight = false
	}
	availW := right
	if !placeRight {
		availW = left
	}
	if surfW > 0 && w > availW && availW >= 1 {
		// Prefer shifting the full intrinsic width; only shrink when the
		// surface itself is narrower than the menu.
		if availW >= intrinsic.X || surfW-2*inset < intrinsic.X {
			w = availW
			sz := popup.Measure(layout.Constraints{MaxW: w, MaxH: -1})
			if sz.Y > h {
				h = sz.Y
			}
			if sz.X > 0 && sz.X < w {
				w = sz.X
			}
			if w < 1 {
				w = 1
			}
		}
	}

	x := anchor.Max.X
	if !placeRight {
		x = anchor.Min.X - w
	}
	y := anchor.Min.Y
	if surfH > 0 && h > surfH-2*inset {
		sz := popup.Measure(layout.Constraints{MaxW: -1, MaxH: surfH - 2*inset})
		w, h = sz.X, sz.Y
		if w < intrinsic.X && surfW-2*inset >= intrinsic.X {
			w = intrinsic.X
		}
	}
	x, w = shiftPopupX(x, w, surfW, inset)
	y, h = shiftPopupY(y, h, surfH, inset)
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}

// CascadeHost is a popup that may show a sibling cascade (submenu) menu.
type CascadeHost interface {
	Cascade() Component
}

// CascadeOf is the open child cascade of c, if any.
func CascadeOf(c Component) Component {
	if c == nil {
		return nil
	}
	if h, ok := c.(CascadeHost); ok {
		return h.Cascade()
	}
	return nil
}

// WalkCascade visits root then each nested cascade (parent first).
func WalkCascade(root Component, fn func(Component)) {
	for c := root; c != nil; c = CascadeOf(c) {
		fn(c)
	}
}

// CascadeLeaf is the deepest open cascade, or root when none is open.
func CascadeLeaf(root Component) Component {
	c := root
	for {
		next := CascadeOf(c)
		if next == nil {
			return c
		}
		c = next
	}
}

// HitCascade hit-tests p against root and its cascade chain (leaf first).
func HitCascade(root Component, p paintengine2d.Point) Component {
	var chain []Component
	WalkCascade(root, func(c Component) { chain = append(chain, c) })
	for i := len(chain) - 1; i >= 0; i-- {
		if h := HitRoot(chain[i], p); h != nil {
			return h
		}
	}
	return nil
}

// PaintCascade paints root then each cascade sibling (child on top).
func PaintCascade(root Component, ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	WalkCascade(root, func(c Component) {
		PaintTree(c, ctx, dirty)
	})
}

// RecordCascade records root then each cascade sibling into rec.
func RecordCascade(root Component, rec *paintengine2d.Recorder, ctx *paintengine2d.Context, dirty *paintengine2d.Damage, cache *SceneCache, fullContent bool) {
	WalkCascade(root, func(c Component) {
		RecordTree(c, rec, ctx, dirty, cache, fullContent)
	})
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
	b := popup.Bounds()
	w, h := b.Dx(), b.Dy()
	x, w := shiftPopupX(b.Min.X, w, float32(ww), inset)
	y, h := shiftPopupY(b.Min.Y, h, float32(hh), inset)
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}

func popupSurfaceWH(c Component) (w, h float32) {
	sz, ok := popupSurface(c)
	if !ok {
		return 0, 0
	}
	ww, hh := sz.SurfaceSize()
	if ww > 8 {
		w = float32(ww)
	}
	if hh > 8 {
		h = float32(hh)
	}
	return w, h
}

// shiftPopupX translates so the full width stays on-screen. Width is only
// reduced when the popup is wider than the surface itself — never to squeeze
// into the leftover strip beside a right-edge anchor.
func shiftPopupX(x, w, surfW, inset float32) (float32, float32) {
	return shiftPopupAxis(x, w, surfW, inset)
}

func shiftPopupY(y, h, surfH, inset float32) (float32, float32) {
	return shiftPopupAxis(y, h, surfH, inset)
}

func shiftPopupAxis(pos, size, surf, inset float32) (float32, float32) {
	if surf <= 0 {
		return pos, size
	}
	if inset < 0 {
		inset = 0
	}
	max := surf - inset
	if pos+size > max {
		pos = max - size
	}
	if pos < inset {
		pos = inset
	}
	if pos+size > max {
		size = max - pos
		if size < 1 {
			size = 1
		}
	}
	return pos, size
}
