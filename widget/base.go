package widget

import (
	"sync/atomic"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

var nextID uint64

// Base is the embeddable Component implementation.
type Base struct {
	self     Component
	id       uint64
	name     string
	parent   Component
	children []Component
	bounds   paintengine2d.Rect
	pref     paintengine2d.Point
	visible  bool
	enabled  bool
	focus    bool
	hovered  bool
	manages  bool
	look     style.LookAndFeel
	host     Host
}

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
func (b *Base) SetBounds(r paintengine2d.Rect) { b.bounds = r.Canon() }
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

func (b *Base) SetLook(l style.LookAndFeel) {
	b.look = l
	b.Invalidate()
}

func (b *Base) SetHost(h Host) {
	b.host = h
	for _, ch := range b.children {
		ch.SetHost(h)
	}
}

func (b *Base) SetPreferred(w, h float32) { b.pref = paintengine2d.Pt(w, h) }

func (b *Base) Preferred() paintengine2d.Point { return b.pref }

func (b *Base) resolveLook() style.LookAndFeel {
	if b.look != nil {
		return b.look
	}
	if b.host != nil {
		return b.host.Look()
	}
	if b.parent != nil {
		return b.parent.Look()
	}
	return style.DarkLook()
}

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
	b.Invalidate()
}

func (b *Base) ClearChildren() {
	for _, ch := range b.children {
		ch.setParent(nil)
	}
	b.children = nil
	b.Invalidate()
}

func (b *Base) Measure(c layout.Constraints) paintengine2d.Point {
	if b.pref.X > 0 || b.pref.Y > 0 {
		return c.Constrain(b.pref)
	}
	return c.Constrain(paintengine2d.Pt(0, 0))
}

func (b *Base) Arrange(r paintengine2d.Rect) { b.bounds = r.Canon() }

func (b *Base) Paint(ctx *paintengine2d.Context) { _ = ctx }

func (b *Base) HitTest(local paintengine2d.Point) Component {
	if !b.visible {
		return nil
	}
	lb := b.LocalBounds()
	if lb.Empty() || !lb.Contains(local) {
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
func (b *Base) FocusLost()                 { b.Invalidate() }

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
	if b.Focused() {
		s |= style.StateFocused
	}
	if b.hovered {
		s |= style.StateHovered
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
