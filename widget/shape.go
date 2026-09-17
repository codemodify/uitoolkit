package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
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
// A component that says nothing asks its *look* instead, and only when it
// names the face it paints ([ShapeRole]): a look whose round button is a
// picture of a disc knows where the disc is, and nothing else does. The look
// is asked once per size and its answer is remembered, so a skinned button
// costs one comparison a hit test and an ordinary look costs a type
// assertion that fails.
//
// The cost of all of this to a component that neither asks nor names a face
// — every widget in the toolkit but one, and every component an app writes —
// is two nil checks and one failed type assertion in HitTest.

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

// ShapeRole is what a component implements to let its look shape it: the
// face it paints, as an engine role. A look with a silhouette for that face
// ([style.ControlShapeEngine]) then decides where the control really is, and
// a press outside it falls through to whatever is behind — which is how a
// skin's round button stops taking clicks in its corners.
//
// It is only for a component whose box *is* one face. A check box is its
// indicator and its label, and shaping the pair by the indicator's art would
// make most of the control deaf.
//
// A component that says nothing here, and every look without the hook, keeps
// the whole box — the default, and what every widget in the toolkit did
// before any of this existed.
type ShapeRole interface {
	ShapeRole() style.Role
}

// lookShape is a component's silhouette as its look last gave it, with
// everything that could change the answer beside it. A component's own
// shape needs no such key — it is handed in — but a look's depends on the
// look, the face and the size, and building one rasterises art.
type lookShape struct {
	look  style.LookAndFeel
	role  style.Role
	w, h  int
	shape *platform.Shape // nil: the look shapes this face as its whole box
}

// hitsLookShape reports whether local point p is inside the silhouette the
// component's look gives the face it paints. True for a component that names
// no face, for a look with no silhouette for it, and whenever the answer
// cannot be worked out — a control nobody can click is a worse failure than
// a square one.
func (b *Base) hitsLookShape(p paintengine2d.Point) bool {
	r, ok := b.me().(ShapeRole)
	if !ok {
		return true
	}
	lb := b.LocalBounds()
	w, h := int(lb.Dx()), int(lb.Dy())
	if w < 1 || h < 1 {
		return true
	}
	lk, role := b.resolveLook(), r.ShapeRole()
	c := b.lookShape
	if c == nil || c.look != lk || c.role != role || c.w != w || c.h != h {
		c = &lookShape{look: lk, role: role, w: w, h: h}
		c.shape = platform.NewShapeSilhouette(style.ControlShapeOf(lk, lb, role))
		b.lookShape = c
	}
	if c.shape == nil {
		return true
	}
	ras := c.shape.Raster(w, h)
	return ras == nil || ras.Contains(int(p.X), int(p.Y))
}

// hitsSilhouette reports whether local point p is the component's at all:
// its own silhouette where it has one, else its look's for the face it
// paints, else — for everything that says nothing — its whole box.
func (b *Base) hitsSilhouette(p paintengine2d.Point) bool {
	if b.hitShape != nil || b.hitFn != nil {
		return b.hitsShape(p)
	}
	return b.hitsLookShape(p)
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
