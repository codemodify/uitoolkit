package widget

import (
	"math"
	"sync/atomic"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

var nextID uint64

// Base is the embeddable Component implementation.
type Base struct {
	self             Component
	id               uint64
	name             string
	parent           Component
	children         []Component
	bounds           paintengine2d.Rect
	pref             paintengine2d.Point
	visible          bool
	enabled          bool
	focus            bool
	hovered          bool
	manages          bool
	keyNav           bool
	focusVisibleOnly bool
	look             style.LookAndFeel
	// theme is this component's own level of the theme cascade, and nil
	// for every component that was never given one, which is almost all
	// of them: a [style.ThemeOverride] is six strings, and inline it
	// would make every widget in every window carry ninety-six bytes for
	// something hardly any of them use (the same reason acc and anchor
	// below are pointers). lookMemo and lookGen are the look this
	// component last resolved to and the generation that answer was good
	// for. See widget/theme.go.
	theme    *style.ThemeOverride
	lookMemo style.LookAndFeel
	lookGen  uint64
	host     Host
	// acc holds an accessible name and description, when set (most
	// components take theirs from their text).
	acc *accLabel
	// hitShape is the component's own silhouette (SetHitShape) and hitFn
	// the callback that rebuilds it at every size (SetHitShapeFunc); only
	// one is ever set, and hitCur remembers what the callback last built
	// for hitCurW by hitCurH. transparent is whether the component paints
	// nothing solid over its box. All of this is nil / false for every
	// component that does not ask, which is every component by default:
	// HitTest pays one nil check for it.
	hitShape         *platform.Shape
	hitFn            func(paintengine2d.Point) *platform.Shape
	hitCur           *platform.Shape
	hitCurW, hitCurH int
	transparent      bool
	// lookShape is the silhouette the look gave the face this component
	// paints, remembered for as long as the look, the face and the size
	// are the same. It stays nil for every component that does not name a
	// face ([ShapeRole]), which is all but one of them.
	lookShape *lookShape
	// anchor is what the component hangs from when it is a popup: set by
	// the Place* functions (widget/popup.go), nil for everything else.
	anchor *PopupAnchor
	// keysPass makes a popup non-capturing: keys it does not take reach
	// the widget it hangs from (widget.SetPopupKeysPass).
	keysPass bool
}

type accLabel struct{ name, desc string }

// Init binds the outer Component so HitTest / Invalidate return the concrete type.
func (b *Base) Init(self Component) {
	if b.id == 0 {
		b.id = atomic.AddUint64(&nextID, 1)
	}
	b.self = self
	b.visible = true
	b.enabled = true
}

func (b *Base) me() Component {
	if b.self != nil {
		return b.self
	}
	return b
}

func (b *Base) ID() uint64                     { return b.id }
func (b *Base) Name() string                   { return b.name }
func (b *Base) SetName(n string)               { b.name = n }
func (b *Base) Parent() Component              { return b.parent }
func (b *Base) setParent(p Component)          { b.parent = p }
func (b *Base) Children() []Component          { return b.children }
func (b *Base) Bounds() paintengine2d.Rect     { return b.bounds }
func (b *Base) SetBounds(r paintengine2d.Rect) { b.bounds = PixelRect(r.Canon()) }
func (b *Base) Visible() bool                  { return b.visible }
func (b *Base) Enabled() bool                  { return b.enabled }
func (b *Base) WantsFocus() bool               { return b.focus }
func (b *Base) SetWantsFocus(v bool)           { b.focus = v }
func (b *Base) ManagesChildren() bool          { return b.manages }
func (b *Base) SetManagesChildren(v bool)      { b.manages = v }
func (b *Base) Host() Host                     { return b.host }
func (b *Base) Look() style.LookAndFeel        { return b.resolveLook() }

func (b *Base) LocalBounds() paintengine2d.Rect {
	return paintengine2d.XYWH(0, 0, b.bounds.Dx(), b.bounds.Dy())
}

func (b *Base) SetVisible(v bool) {
	if b.visible == v {
		return
	}
	b.visible = v
	b.Invalidate()
}

func (b *Base) SetEnabled(v bool) {
	if b.enabled == v {
		return
	}
	b.enabled = v
	b.Invalidate()
}

// SetLook puts an exact look on this component and its subtree, past the
// cascade: whatever the window or a scope above says, this subtree draws
// in l. It is the low-level form — it takes a look already built at a
// display scale, and nothing rescales it afterwards — and it is what
// widgets.ThemeScope and an inherited popup are built on.
//
// An application that wants a pane in another pack wants
// [Base.SetTheme], which states a level of the cascade instead and
// inherits the window's scale, density and everything it does not name.
func (b *Base) SetLook(l style.LookAndFeel) {
	if b.look != l {
		b.look = l
		// Only when it moved: a popup handed the same look it already
		// had — a combo box reopened in the same pane — must not throw
		// every memo in the process away.
		LooksChanged()
	}
	b.Invalidate()
}

func (b *Base) SetHost(h Host) {
	if b.host != h {
		// A different window is a different look at the root of the
		// cascade (a tab torn into a window of its own, a popup mounted
		// on a layer).
		LooksChanged()
	}
	b.host = h
	for _, ch := range b.children {
		ch.SetHost(h)
	}
}

func (b *Base) SetPreferred(w, h float32) { b.pref = paintengine2d.Pt(w, h) }

func (b *Base) Preferred() paintengine2d.Point { return b.pref }

func (b *Base) Add(child Component) {
	if child == nil {
		return
	}
	if child.Parent() != nil {
		child.Parent().Remove(child)
	}
	b.children = append(b.children, child)
	child.setParent(b.me())
	child.SetHost(b.host)
	// The subtree hangs somewhere else now, so the levels above it are
	// different ones.
	LooksChanged()
	b.Invalidate()
}

func (b *Base) Remove(child Component) {
	if child == nil {
		return
	}
	out := b.children[:0]
	for _, ch := range b.children {
		if ch != child {
			out = append(out, ch)
		}
	}
	b.children = out
	child.setParent(nil)
	LooksChanged()
	b.Invalidate()
}

func (b *Base) ClearChildren() {
	for _, ch := range b.children {
		ch.setParent(nil)
	}
	b.children = nil
	LooksChanged()
	b.Invalidate()
}

func (b *Base) Measure(c layout.Constraints) paintengine2d.Point {
	if b.pref.X > 0 || b.pref.Y > 0 {
		return c.Constrain(b.pref)
	}
	return c.Constrain(paintengine2d.Pt(0, 0))
}

func (b *Base) Arrange(r paintengine2d.Rect) { b.bounds = PixelRect(r.Canon()) }

// PixelRect rounds r's edges to whole pixels. Layout works in device pixels
// and every component's bounds go through it (layout rounding, as in WPF,
// Avalonia and WinUI): each origin then sits on the pixel grid, so a look's
// pixel snapping and its 1px lines land on real pixels instead of smearing
// over two, and neighbours that share an edge still share it.
func PixelRect(r paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(roundPx(r.Min.X), roundPx(r.Min.Y)),
		Max: paintengine2d.Pt(roundPx(r.Max.X), roundPx(r.Max.Y)),
	}
}

func roundPx(v float32) float32 { return float32(math.Floor(float64(v) + 0.5)) }

func (b *Base) Paint(ctx *paintengine2d.Context) { _ = ctx }

func (b *Base) HitTest(local paintengine2d.Point) Component {
	if !b.visible {
		return nil
	}
	lb := b.LocalBounds()
	if lb.Empty() || !lb.Contains(local) {
		return nil
	}
	// A component with a silhouette — its own, or the one its look gives
	// the face it paints — takes input only inside it, and neither do its
	// children: a press outside goes to whatever is behind, exactly as it
	// does outside a shaped window. Every component that asks for neither —
	// the default, and every widget in the toolkit but one — pays two nil
	// checks and a type assertion here.
	if !b.hitsSilhouette(local) {
		return nil
	}
	for i := len(b.children) - 1; i >= 0; i-- {
		ch := b.children[i]
		if !ch.Visible() {
			continue
		}
		cb := ch.Bounds()
		lp := paintengine2d.Pt(local.X-cb.Min.X, local.Y-cb.Min.Y)
		if hit := ch.HitTest(lp); hit != nil {
			return hit
		}
	}
	return b.me()
}

func (b *Base) MousePress(MouseEvent) bool   { return false }
func (b *Base) MouseRelease(MouseEvent) bool { return false }
func (b *Base) MouseMove(MouseEvent) bool    { return false }
func (b *Base) MouseEnter() {
	if !b.hovered {
		b.hovered = true
		b.Invalidate()
	}
}
func (b *Base) MouseExit() {
	if b.hovered {
		b.hovered = false
		b.Invalidate()
	}
}
func (b *Base) MouseWheel(MouseEvent) bool { return false }
func (b *Base) Hovered() bool              { return b.hovered }
func (b *Base) KeyPress(KeyEvent) bool     { return false }
func (b *Base) KeyRelease(KeyEvent) bool   { return false }
func (b *Base) TextInput(rune) bool        { return false }
func (b *Base) FocusGained()               { b.Invalidate() }

func (b *Base) FocusLost() {
	b.keyNav = false
	b.Invalidate()
}

// SetFocusVisibleOnly paints StateFocused only after keyboard navigation
// (GTK :focus-visible / Qt WA_KeyboardFocusChange). Editors leave this off.
func (b *Base) SetFocusVisibleOnly(v bool) { b.focusVisibleOnly = v }

// FocusVisibleOnly reports the keyboard-only focus-ring policy.
func (b *Base) FocusVisibleOnly() bool { return b.focusVisibleOnly }

// KeyNav is true after Tab / arrow / mnemonic until pointer activation.
func (b *Base) KeyNav() bool { return b.keyNav }

// MarkKeyboardFocus records that focus arrived from the keyboard.
func (b *Base) MarkKeyboardFocus() {
	if !b.keyNav {
		b.keyNav = true
		b.Invalidate()
	}
}

// MarkPointerFocus records that focus arrived from the pointer.
func (b *Base) MarkPointerFocus() {
	if b.keyNav {
		b.keyNav = false
		b.Invalidate()
	}
}

func (b *Base) Invalidate() {
	b.InvalidateRect(b.LocalBounds())
}

func (b *Base) InvalidateRect(r paintengine2d.Rect) {
	h := b.host
	if h == nil && b.parent != nil {
		h = b.parent.Host()
	}
	if h != nil {
		h.Invalidate(b.me(), r)
	}
}

func (b *Base) RequestFocus() {
	h := b.host
	if h == nil && b.parent != nil {
		h = b.parent.Host()
	}
	if h != nil {
		h.RequestFocus(b.me())
	}
}

// RequestLayout asks the host to Measure/Arrange before the next paint.
func (b *Base) RequestLayout() {
	h := b.host
	if h == nil && b.parent != nil {
		h = b.parent.Host()
	}
	if h != nil {
		h.RequestLayout()
	}
}

func (b *Base) Focused() bool {
	h := b.host
	if h == nil && b.parent != nil {
		h = b.parent.Host()
	}
	return h != nil && h.Focus() == b.me()
}

func (b *Base) State() style.ControlState {
	var s style.ControlState
	if !b.enabled {
		s |= style.StateDisabled
	}
	if b.Focused() && (!b.focusVisibleOnly || b.keyNav) {
		s |= style.StateFocused
	}
	if b.hovered {
		s |= style.StateHovered
	}
	if !WindowActive(b.me()) {
		// The window is in the backdrop; looks may subdue what their
		// platform did (Aqua's default button loses its blue).
		s |= style.StateBackdrop
	}
	return s
}

// Find returns the first pre-order node of type T, if any.
func Find[T Component](root Component) (T, bool) {
	var found T
	ok := false
	Walk(root, func(c Component) {
		if ok {
			return
		}
		if v, yes := c.(T); yes {
			found, ok = v, true
		}
	})
	return found, ok
}

// Walk pre-order visits c and visible descendants.
func Walk(c Component, fn func(Component)) {
	if c == nil || !c.Visible() {
		return
	}
	fn(c)
	for _, ch := range c.Children() {
		Walk(ch, fn)
	}
}

// Contains reports whether target is root or a descendant (ignores visibility).
func Contains(root, target Component) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, ch := range root.Children() {
		if Contains(ch, target) {
			return true
		}
	}
	return false
}

// Focusables returns tab-order candidates (visible, enabled, wants focus).
func Focusables(root Component) []Component {
	var out []Component
	Walk(root, func(c Component) {
		if c.Visible() && c.Enabled() && c.WantsFocus() {
			out = append(out, c)
		}
	})
	return out
}

// DeviceOrigin of c in the root's local space.
func DeviceOrigin(c Component) paintengine2d.Point {
	var p paintengine2d.Point
	for n := c; n != nil; n = n.Parent() {
		b := n.Bounds()
		p.X += b.Min.X
		p.Y += b.Min.Y
	}
	return p
}

// DeviceBounds is the axis-aligned box of c in root space.
func DeviceBounds(c Component) paintengine2d.Rect {
	o := DeviceOrigin(c)
	b := c.Bounds()
	return paintengine2d.XYWH(o.X, o.Y, b.Dx(), b.Dy())
}
