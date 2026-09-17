package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// A component's own silhouette.
//
// Every component takes input across its whole box, and always has: that is
// the default, it is what all 121 packs and every widget in the toolkit
// rely on, and nothing here changes it. A component that wants otherwise
// says so with [Base.SetHitShape], and then a press outside its silhouette
// goes to whatever is behind it in the tree, exactly as a press outside a
// shaped window's silhouette goes to the window behind it.
//
// The cost of this to a component that does not ask is one nil check in
// HitTest. The shape is consulted only for components that have one.

// SetHitShape gives the component a silhouette to take input inside,
// stated in its own local device pixels. A press outside it is not the
// component's — nor any of its children's — and falls through to whatever
// is behind. nil, the default, is the whole box.
//
// A shape that has to follow the component's size wants
// [Base.SetHitShapeFunc] instead.
func (b *Base) SetHitShape(s *platform.Shape) {
	b.hitFn = nil
	b.hitShape = s
	b.hitCur, b.hitCurW, b.hitCurH = nil, 0, 0
}

// SetHitShapeFunc gives the component a silhouette that follows its size:
// fn is called with the component's size in local device pixels whenever it
// changes, and returns the shape for that size (nil for none). nil drops
// the callback.
func (b *Base) SetHitShapeFunc(fn func(size paintengine2d.Point) *platform.Shape) {
	b.hitFn = fn
	b.hitShape = nil
	b.hitCur, b.hitCurW, b.hitCurH = nil, 0, 0
}

// HitShape is the silhouette in effect for the component's current size
// (nil: its whole box).
func (b *Base) HitShape() *platform.Shape {
	if b == nil {
		return nil
	}
	if b.hitFn == nil {
		return b.hitShape
	}
	lb := b.LocalBounds()
	w, h := int(lb.Dx()), int(lb.Dy())
	if w < 1 || h < 1 {
		return nil
	}
	// Remembered until the size changes: a callback hands back a fresh
	// Shape each time, and a fresh Shape rasterises itself afresh.
	if b.hitCur != nil && b.hitCurW == w && b.hitCurH == h {
		return b.hitCur
	}
	b.hitCur = b.hitFn(paintengine2d.Pt(float32(w), float32(h)))
	b.hitCurW, b.hitCurH = w, h
	return b.hitCur
}

// hitsShape reports whether local point p is inside the component's
// silhouette. A component with no silhouette is its whole box, so this is
// only ever asked of one that has one.
func (b *Base) hitsShape(p paintengine2d.Point) bool {
	s := b.HitShape()
	if s == nil {
		return true
	}
	lb := b.LocalBounds()
	r := s.Raster(max(int(lb.Dx()), 1), max(int(lb.Dy()), 1))
	if r == nil {
		return true
	}
	return r.Contains(int(p.X), int(p.Y))
}

// SetTransparent marks the component as painting nothing solid over its
// box: the window does not claim those pixels opaque, so a compositor
// blends whatever is behind them through. It is what a glass panel or a
// see-through overlay declares.
//
// It changes nothing on its own — a window states an opaque region only
// when it is shaped or glassy — and it is never consulted for an ordinary
// opaque window, which is why declaring it costs nothing.
func (b *Base) SetTransparent(on bool) {
	if b == nil || b.transparent == on {
		return
	}
	b.transparent = on
	b.Invalidate()
}

// Transparent reports whether the component declared itself see-through.
func (b *Base) Transparent() bool { return b != nil && b.transparent }

// TransparentRects collects the device-space boxes of every component in
// root's tree that declared itself see-through, appending to out. A window
// subtracts them from the region it tells the compositor is solid.
//
// It walks the tree, so it is called when a window lays itself out and only
// when that window is shaped or glassy — never for an ordinary window.
func TransparentRects(root Component, out []paintengine2d.Rect) []paintengine2d.Rect {
	if root == nil || !root.Visible() {
		return out
	}
	if t, ok := root.(interface{ Transparent() bool }); ok && t.Transparent() {
		out = append(out, DeviceBounds(root))
	}
	for _, c := range root.Children() {
		out = TransparentRects(c, out)
	}
	return out
}

// HitShaper is what a component implements to be asked about its
// silhouette. Every component embedding [Base] does, so this exists for the
// rare component that implements Component from scratch.
type HitShaper interface {
	// HitShape is the component's silhouette in its own local device
	// pixels, or nil for its whole box.
	HitShape() *platform.Shape
}

// ShapeHit reports whether local point p is inside c's silhouette — true
// for a component that has none, which is every component by default.
func ShapeHit(c Component, p paintengine2d.Point) bool {
	h, ok := c.(HitShaper)
	if !ok {
		return true
	}
	s := h.HitShape()
	if s == nil {
		return true
	}
	lb := c.LocalBounds()
	r := s.Raster(max(int(lb.Dx()), 1), max(int(lb.Dy()), 1))
	return r == nil || r.Contains(int(p.X), int(p.Y))
}
