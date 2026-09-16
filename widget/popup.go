package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
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
	inheritLook(from, popup, h)
	ph.SetPopup(popup)
	notifyPresented(from, popup)
	return true
}

// inheritLook gives a popup or overlay the look of the widget that opened
// it when that differs from the window's (a combo inside a themed preview
// opens a list in the preview's theme, not the window's).
func inheritLook(from, layer Component, h Host) {
	fl, ok := from.(interface{ Look() style.LookAndFeel })
	if !ok {
		return
	}
	sl, ok := layer.(interface{ SetLook(style.LookAndFeel) })
	if !ok {
		return
	}
	if lk := fl.Look(); lk != nil && lk != h.Look() {
		sl.SetLook(lk)
	}
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
	inheritLook(from, overlay, h)
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

// WindowRecter is implemented by app.Window: the visible window inside the
// surface, in surface coordinates. A frame the toolkit draws keeps an
// invisible margin around the window for its shadow, and a menu, a combo
// list or a tooltip must stay inside the window, not float in the margin
// where the compositor may clip it and clicks pass through.
type WindowRecter interface {
	WindowRect() paintengine2d.Rect
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
	if box, ok := popupWindowRect(popup); ok {
		if box.Dx() > 8 {
			maxW = box.Dx() - 8
		}
		if box.Dy() > 8 {
			maxH = box.Dy() - 8
		}
	}
	return maxW, maxH
}

// popupWindowRect is the box a popup must stay inside: the host's visible
// window, else its whole surface.
func popupWindowRect(c Component) (paintengine2d.Rect, bool) {
	if c == nil {
		return paintengine2d.Rect{}, false
	}
	h := c.Host()
	if h == nil {
		return paintengine2d.Rect{}, false
	}
	if wr, ok := h.(WindowRecter); ok {
		if box := wr.WindowRect(); box.Dx() > 8 && box.Dy() > 8 {
			return box, true
		}
	}
	sz, ok := h.(SurfaceSizer)
	if !ok {
		return paintengine2d.Rect{}, false
	}
	ww, hh := sz.SurfaceSize()
	if ww < 1 || hh < 1 {
		return paintengine2d.Rect{}, false
	}
	return paintengine2d.XYWH(0, 0, float32(ww), float32(hh)), true
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
	box := popupSurfaceBox(popup)
	surfH := box.Max.Y

	// Vertical: flip or scroll (v0.10.8). Height may shrink; width must not
	// be reduced to the leftover strip beside the anchor.
	below, above := float32(1e9), float32(1e9)
	if !box.Empty() {
		below = box.Max.Y - inset - (anchor.Max.Y + gap)
		above = anchor.Min.Y - gap - (box.Min.Y + inset)
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

	x, w := shiftPopupAxis(anchor.Min.X, w, box.Min.X, box.Max.X, inset)
	y := anchor.Max.Y + gap
	if !placeBelow {
		y = anchor.Min.Y - gap - h
	}
	y, h = shiftPopupAxis(y, h, box.Min.Y, box.Max.Y, inset)
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
	box := popupSurfaceBox(popup)
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
	x, w := shiftPopupAxis(origin.X, w, box.Min.X, box.Max.X, inset)
	y, h := shiftPopupAxis(origin.Y, h, box.Min.Y, box.Max.Y, inset)
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
	box := popupSurfaceBox(popup)
	surfW, surfH := box.Dx(), box.Dy()

	right := float32(1e9)
	left := float32(1e9)
	if surfW > 0 {
		right = box.Max.X - inset - anchor.Max.X
		left = anchor.Min.X - (box.Min.X + inset)
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
	x, w = shiftPopupAxis(x, w, box.Min.X, box.Max.X, inset)
	y, h = shiftPopupAxis(y, h, box.Min.Y, box.Max.Y, inset)
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
	box, ok := popupWindowRect(from)
	if !ok {
		box, ok = popupWindowRect(popup)
	}
	if !ok {
		return
	}
	inset := float32(4)
	b := popup.Bounds()
	w, h := b.Dx(), b.Dy()
	x, w := shiftPopupAxis(b.Min.X, w, box.Min.X, box.Max.X, inset)
	y, h := shiftPopupAxis(b.Min.Y, h, box.Min.Y, box.Max.Y, inset)
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
}

// popupSurfaceBox is the box a popup is placed in (an empty box when the
// host cannot say).
func popupSurfaceBox(c Component) paintengine2d.Rect {
	box, ok := popupWindowRect(c)
	if !ok || box.Dx() <= 8 || box.Dy() <= 8 {
		return paintengine2d.Rect{}
	}
	return box
}

// shiftPopupAxis translates so the full size stays inside [lo, hi] (the
// visible window, which a frame's margin keeps the popup out of). The size
// is only reduced when the popup is larger than that — never to squeeze it
// into the leftover strip beside an edge anchor.
func shiftPopupAxis(pos, size, lo, hi, inset float32) (float32, float32) {
	if hi <= lo {
		return pos, size
	}
	if inset < 0 {
		inset = 0
	}
	lo, hi = lo+inset, hi-inset
	if pos+size > hi {
		pos = hi - size
	}
	if pos < lo {
		pos = lo
	}
	if pos+size > hi {
		size = hi - pos
		if size < 1 {
			size = 1
		}
	}
	return pos, size
}
