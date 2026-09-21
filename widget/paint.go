package widget

import "github.com/codemodify/paintengine2d"

// PaintTree paints c and descendants. dirty is in device (root) pixels.
// If dirty is nil or empty, everything visible is painted.
func PaintTree(c Component, ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	if c == nil || !c.Visible() {
		return
	}
	paintNode(c, ctx, dirty)
}

func paintNode(c Component, ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	if c == nil || !c.Visible() {
		return
	}
	b := c.Bounds()
	local := c.LocalBounds()
	if local.Empty() && !b.Empty() {
		local = paintengine2d.XYWH(0, 0, b.Dx(), b.Dy())
	}
	if local.Empty() {
		return
	}
	ctx.Save()
	ctx.Translate(b.Min.X, b.Min.Y)
	ctx.ClipRect(local)
	if ctx.ClipEmpty() || ctx.QuickReject(local) {
		ctx.Restore()
		return
	}
	if dirty != nil && !dirty.Empty() {
		dev := ctx.Matrix().TransformRect(local)
		if !dirty.Overlaps(dev) {
			ctx.Restore()
			return
		}
	}
	if pc, ok := c.(PaintClipper); ok {
		pc.PaintClip(ctx)
	}
	c.Paint(ctx)
	if !c.ManagesChildren() {
		for _, ch := range c.Children() {
			paintNode(ch, ctx, dirty)
		}
	}
	ctx.Restore()
}

// PaintClipper is what a component implements to cut itself and everything
// painted under it to a shape of its own, beyond the box every component is
// already clipped to: an in-app window cut to its look's silhouette. It is
// called with the component's own box already clipped and translated, and
// narrows the clip further; it paints nothing and never widens anything.
type PaintClipper interface {
	PaintClip(ctx *paintengine2d.Context)
}

// HitRoot hit-tests p in root-local (window) coordinates.
func HitRoot(root Component, p paintengine2d.Point) Component {
	if root == nil {
		return nil
	}
	b := root.Bounds()
	lp := paintengine2d.Pt(p.X-b.Min.X, p.Y-b.Min.Y)
	return root.HitTest(lp)
}
