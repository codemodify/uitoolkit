package widget

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/diag"
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
//
// The look reaches the popup *here* — a popup is a root, so it has no
// look of its own until it is given the window's ([PreparePopup]) — and
// a caller that measured it before calling this measured it in the
// toolkit's default look at scale 1. On a screen at twice that the popup
// was then drawn at twice the size inside a box of the first: its words
// cut short, its last rows off the bottom. So when the look changes
// under a popup that was already arranged, the popup is measured again
// and grown where it stands. Measuring after [PreparePopup], which the
// Place* helpers all do, leaves nothing to grow.
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
	was := lookOf(popup)
	popup.SetHost(h)
	inheritLook(from, popup, h)
	if now := lookOf(popup); measuresDifferently(was, now) {
		refitPopup(popup, was, now)
	}
	ph.SetPopup(popup)
	notifyPresented(from, popup)
	return true
}

// lookOf is the look a popup resolves to, or nil for one that has none to
// resolve (a component that is not a widget).
func lookOf(c Component) style.LookAndFeel {
	l, ok := c.(interface{ Look() style.LookAndFeel })
	if !ok {
		return nil
	}
	return l.Look()
}

// measuresDifferently reports whether a size measured in look a could
// have come out other than in look b: the display scale and every metric
// a widget measures itself from. It is a comparison of *values*, not of
// looks: the same look is re-derived behind a theme scope often enough —
// every open of a combo box in a themed pane hands its menu a fresh
// object — and re-arranging a popup because a pointer moved would undo a
// placement that was already right.
func measuresDifferently(a, b style.LookAndFeel) bool {
	if a == nil || b == nil {
		return a != b
	}
	return style.LookScale(a) != style.LookScale(b) || a.Metrics() != b.Metrics()
}

// refitPopup grows a popup that was arranged in a look other than the one
// it is about to be drawn in, which is what a caller who measured before
// [ShowPopup] could only have done.
//
// It grows and never shrinks: a popup deliberately arranged smaller than
// its content — a panel holding a list that scrolls — is a size somebody
// chose, and the look changing is no reason to overrule it. What was
// chosen in the *wrong* look is not a size anybody chose, and a box that
// is too small for the content is the one case this can tell apart.
//
// Growing stays inside the window, by the same clamp the Place* helpers
// use: a popup that would now hang off the bottom is moved up, and one
// taller than the window keeps the room there is.
func refitPopup(popup Component, was, now style.LookAndFeel) {
	b := popup.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		// Never arranged: there is no size to be wrong, and whoever shows
		// a popup like this is placing it after.
		return
	}
	want := popup.Measure(layout.Unbounded())
	box := popupSurfaceBox(popup)
	maxW, maxH := want.X, want.Y
	if !box.Empty() {
		maxW, maxH = box.Dx()-8, box.Dy()-8
	}
	if want.Y > maxH {
		// Taller than the window: measure against the room there is,
		// which is how a long list becomes one that scrolls.
		if sz := popup.Measure(layout.Constraints{MaxW: -1, MaxH: maxH}); sz.Y > 0 {
			want = paintengine2d.Pt(max(sz.X, want.X), sz.Y)
		}
	}
	w, h := b.Dx(), b.Dy()
	if want.X > w {
		w = min(want.X, maxW)
	}
	if want.Y > h {
		h = min(want.Y, maxH)
	}
	if w <= b.Dx() && h <= b.Dy() {
		return
	}
	x, w := shiftPopupAxis(b.Min.X, w, box.Min.X, box.Max.X, 4)
	y, h := shiftPopupAxis(b.Min.Y, h, box.Min.Y, box.Max.Y, 4)
	popup.Arrange(paintengine2d.XYWH(x, y, w, h))
	diag.Report(diag.Finding{
		Level: diag.Note,
		Area:  "popup",
		Asked: fmt.Sprintf("a popup shown at the %gx%g it was arranged at",
			b.Dx(), b.Dy()),
		Got: fmt.Sprintf("%gx%g, its size in the window's look (it was measured in %s at scale %g, and is drawn in %s at scale %g)",
			w, h, lookName(was), style.LookScale(was), lookName(now), style.LookScale(now)),
		Fix: "widget.PreparePopup(from, popup) before measuring it, or place it with widget.PlacePopupForAnchor / PlacePopup",
	})
}

// lookName is a look's pack name for a diagnostic, and "the default look"
// for one that has none.
func lookName(l style.LookAndFeel) string {
	if l == nil {
		return "no look"
	}
	if p, ok := l.(interface{ Pack() string }); ok {
		if n := p.Pack(); n != "" {
			return n
		}
	}
	return "the default look"
}

// inheritLook gives a popup, a submenu, an overlay or a tooltip the look
// of the widget that opened it, whenever that is not the window's: a
// combo box inside a Metal-themed pane opens a Metal list, and the menu
// and every submenu that cascade out of a Metal-themed tab are Metal
// too.
//
// A popup is a root — it hangs from no parent, and on Wayland and X11 it
// is a surface of its own — so the cascade cannot reach it by walking
// up. This is where it is carried across instead, and it is carried as
// an exact look rather than as a level of the cascade, because the
// opener's look is already at the window's display scale and re-deriving
// it would be one rebuild for nothing.
//
// The look is set (or cleared) on every open, never only when there is
// one to impose: a popup reused by its widget — a combo box keeps its
// list — must not keep the theme of a scope it has since left.
func inheritLook(from, layer Component, h Host) {
	fl, ok := from.(interface{ Look() style.LookAndFeel })
	if !ok {
		return
	}
	sl, ok := layer.(interface{ SetLook(style.LookAndFeel) })
	if !ok {
		return
	}
	lk := fl.Look()
	if lk == nil || (h != nil && lk == h.Look()) {
		// Nothing of its own: the popup resolves the window's look the
		// ordinary way, and a look left over from a previous open goes.
		lk = nil
	}
	sl.SetLook(lk)
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
	// A host that stacks puts this one on top of whatever is already
	// there; one that does not replaces it, as it always did. A
	// confirmation raised from inside a dialog used to close that dialog
	// and run its OnClose — which for a dialog that wipes its secret
	// fields on close wiped them.
	if st, ok := h.(OverlayStackHost); ok {
		st.PushOverlay(overlay)
	} else {
		oh.SetOverlay(overlay)
	}
	notifyPresented(from, overlay)
	return true
}

// OverlayStackHost is a host whose overlays stack: a dialog raised from
// inside another comes back to it. A host without it has one overlay,
// and showing a second replaces the first.
type OverlayStackHost interface {
	PushOverlay(Component)
	PopOverlay(Component) bool
	Overlay() Component
	// Overlays is the stack, oldest first.
	Overlays() []Component
}

// DismissOverlay closes the overlay from is in and leaves whatever was
// under it up. Called from outside every overlay — a window's content
// taking down whatever is on screen — it closes the top one, which is
// what it did before overlays stacked.
//
// The one case that closes nothing is an overlay that *was* on the stack
// and is no longer, which is reached on the way out of every dismissal:
// an overlay's OnClose commonly calls back into the code that closes it,
// and popping "the top" there would take the dialog underneath down too.
func DismissOverlay(from Component) {
	if from == nil {
		return
	}
	h := from.Host()
	if h == nil {
		return
	}
	if st, ok := h.(OverlayStackHost); ok {
		root := overlayRootOf(from)
		for _, o := range st.Overlays() {
			if o == from || o == root {
				st.PopOverlay(o)
				return
			}
		}
		// from is not in any overlay on the stack, which is two cases
		// that must not be treated alike.
		//
		// A component in the window's own content asking to take down
		// whatever is up is what this call meant before overlays
		// stacked, and closing nothing silently broke it: a program
		// quitting with a recovery key on screen stopped wiping it, and
		// a cancelled file chooser stayed up. That one closes the top.
		//
		// An overlay that *was* on the stack and is no longer closes
		// nothing. That case is reached on the way out of every
		// dismissal — an overlay's OnClose calls back into the code that
		// closes it — and popping the top there would take the dialog
		// underneath down as well.
		if inHostTree(from, st) {
			if top := st.Overlay(); top != nil {
				st.PopOverlay(top)
			}
		}
		return
	}
	if oh, ok := h.(OverlayHost); ok {
		oh.SetOverlay(nil)
	}
}

// overlayRootOf is the top of from's own tree, which is the overlay it
// sits in when it sits in one.
func overlayRootOf(from Component) Component {
	root := from
	for n := from; n != nil; n = n.Parent() {
		root = n
	}
	return root
}

// ContentRootHost is a host that can say which of its trees are its own
// content rather than an overlay.
type ContentRootHost interface {
	// ContentRoots are the host's non-overlay trees: its content, and
	// the caption where the toolkit draws one.
	ContentRoots() []Component
}

// inHostTree reports whether c hangs under one of the host's own trees
// rather than under an overlay. It is how "close whatever is up" is told
// from "close the overlay I am in, which has already gone".
//
// A host that cannot say answers no, which keeps the safe behaviour: a
// dismissal that has already happened does not take a second dialog with
// it.
func inHostTree(c Component, st OverlayStackHost) bool {
	cr, ok := st.(ContentRootHost)
	if !ok {
		return false
	}
	root := overlayRootOf(c)
	for _, r := range cr.ContentRoots() {
		if r != nil && r == root {
			return true
		}
	}
	return false
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

// PopupAreaHost is implemented by app.Window when its popups are surfaces
// of their own (a Wayland xdg_popup, an X11 override-redirect window)
// rather than layers drawn inside it. A popup is then placed in the box
// PopupArea answers — the work area of the monitor the window is on, in the
// window's device pixels, so it may run past the window's edges and does
// wherever it has to — instead of being squeezed into [WindowRecter]'s box.
// ok is false while popups are drawn inside the window.
type PopupAreaHost interface {
	PopupArea() (box paintengine2d.Rect, ok bool)
}

// PopupSide is where a popup went relative to what it hangs from.
type PopupSide uint8

const (
	// PopupBelow and PopupAbove: under or over the anchor, left edges
	// aligned (a menu bar's menu, a combo box's list).
	PopupBelow PopupSide = iota
	PopupAbove
	// PopupRight and PopupLeft: beside the anchor, top edges aligned (a
	// submenu beside its row).
	PopupRight
	PopupLeft
	// PopupAt: from a point, down and to the right (a context menu at the
	// pointer, a tooltip below it).
	PopupAt
)

// PopupAnchor is what a popup hangs from, as the Place* functions leave it:
// the anchor's box in window device pixels and the side of it the popup
// went to. A popup that is a surface of its own hands both to the window
// system (xdg_positioner), which flips and slides it against the screen's
// edges exactly as the placement here does against the window's.
type PopupAnchor struct {
	Rect paintengine2d.Rect
	Side PopupSide
}

type anchored interface{ anchorSlot() **PopupAnchor }

func (b *Base) anchorSlot() **PopupAnchor { return &b.anchor }

// SetPopupAnchor records what popup hangs from. The Place* functions call
// it; a popup placed by hand (a tooltip) states its own.
func SetPopupAnchor(popup Component, a PopupAnchor) {
	if s, ok := popup.(anchored); ok {
		cp := a
		*s.anchorSlot() = &cp
	}
}

// PopupAnchorOf is what popup hangs from, when it was placed by one of the
// Place* functions or SetPopupAnchor.
func PopupAnchorOf(popup Component) (PopupAnchor, bool) {
	if s, ok := popup.(anchored); ok {
		if a := *s.anchorSlot(); a != nil {
			return *a, true
		}
	}
	return PopupAnchor{}, false
}

type keyPasser interface{ keysPassSlot() *bool }

func (b *Base) keysPassSlot() *bool { return &b.keysPass }

// SetPopupKeysPass makes a popup **non-capturing**: a key it does not
// take reaches the widget that has the focus, which is normally the one
// the popup hangs from.
//
// A popup owns the keyboard by default, and that is right for a menu: a
// menu is where the keyboard is while it is open, and a key leaking past
// it to the page behind would act on something the user cannot see.
//
// It is wrong for the other kind of popup — a completion list, a
// type-ahead drop-down, a search-suggest panel — where the user is still
// typing into the field underneath. There the popup wants the keys that
// are about the list (Up, Down, Return, Escape, which it takes by
// returning true from KeyPress) and nothing else; every letter, every
// Backspace and every caret key belongs to the field. Without this the
// only way to build one was to lay the list out inline and reflow the
// form around it.
//
// The popup is still offered every key first, so it decides what is
// about the list. It does not take the focus either, so the field keeps
// its caret: it is the app that opens and updates the popup as the text
// changes, and dismisses it when there is nothing to suggest.
func SetPopupKeysPass(popup Component, v bool) {
	if s, ok := popup.(keyPasser); ok {
		*s.keysPassSlot() = v
	}
}

// PopupKeysPass reports whether popup lets the keys it did not take
// through to the focused widget ([SetPopupKeysPass]).
func PopupKeysPass(popup Component) bool {
	if s, ok := popup.(keyPasser); ok {
		return *s.keysPassSlot()
	}
	return false
}

// PreparePopup attaches from's host and from's look to popup before it is
// measured.
//
// The look has to be on before Measure and not only before paint: packs
// differ in their metrics, and a list measured in the window's look and
// then painted in a scope's would be the wrong size. Every Place*
// function starts here, so a submenu — which is opened by its parent
// menu rather than through [ShowPopup] — inherits the cascade too.
func PreparePopup(from, popup Component) {
	if from == nil || popup == nil {
		return
	}
	h := from.Host()
	if h == nil {
		return
	}
	popup.SetHost(h)
	inheritLook(from, popup, h)
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

// popupWindowRect is the box a popup must stay inside: the screen's work
// area where popups are surfaces of their own, else the host's visible
// window, else its whole surface.
func popupWindowRect(c Component) (paintengine2d.Rect, bool) {
	if c == nil {
		return paintengine2d.Rect{}, false
	}
	h := c.Host()
	if h == nil {
		return paintengine2d.Rect{}, false
	}
	if pa, ok := h.(PopupAreaHost); ok {
		if box, ok := pa.PopupArea(); ok && box.Dx() > 8 && box.Dy() > 8 {
			return box, true
		}
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
	side := PopupBelow
	if !placeBelow {
		side = PopupAbove
	}
	// The gap belongs to the anchor: a window system that flips the popup
	// keeps it the same distance off on the other side.
	SetPopupAnchor(popup, PopupAnchor{Rect: paintengine2d.Rect{
		Min: paintengine2d.Pt(anchor.Min.X, anchor.Min.Y-gap),
		Max: paintengine2d.Pt(anchor.Max.X, anchor.Max.Y+gap),
	}, Side: side})
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
	SetPopupAnchor(popup, PopupAnchor{Rect: paintengine2d.XYWH(origin.X, origin.Y, 1, 1), Side: PopupAt})
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
	side := PopupRight
	if !placeRight {
		side = PopupLeft
	}
	SetPopupAnchor(popup, PopupAnchor{Rect: anchor, Side: side})
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
