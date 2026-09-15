package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// Scrolling views (ListView, TreeView, TableView) sit in the look's view
// frame: Win95's sunken well, Luna's hairline, nothing on flat looks. A view
// works in view space — the viewport inside the frame, origin at its top-left
// — so its scrolling and row maths are unchanged; Paint draws the frame over
// the whole view and translates into view space, and mouse positions convert
// on the way in.

// viewFrame is the frame a view shows: the look's, unless it opted out.
func viewFrame(lk style.LookAndFeel, frameless bool) style.Insets {
	if frameless || lk == nil {
		return style.Insets{}
	}
	return style.ViewFrameInsetsOf(lk)
}

// viewInner is the viewport in view space: the bounds minus the frame.
func viewInner(b paintengine2d.Rect, in style.Insets) paintengine2d.Rect {
	w := b.Dx() - in.Left - in.Right
	h := b.Dy() - in.Top - in.Bottom
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return paintengine2d.XYWH(0, 0, w, h)
}

// toView maps a point in the view's local space into view space.
func toView(p paintengine2d.Point, in style.Insets) paintengine2d.Point {
	return paintengine2d.Pt(p.X-in.Left, p.Y-in.Top)
}

// fromView maps a view-space rect back into the view's local space.
func fromView(r paintengine2d.Rect, in style.Insets) paintengine2d.Rect {
	if r.Empty() {
		return r
	}
	return r.Translate(paintengine2d.Pt(in.Left, in.Top))
}

// beginViewFrame paints the frame over b (the whole view) and moves ctx
// into view space. It reports whether it did; the caller restores ctx then.
func beginViewFrame(ctx *paintengine2d.Context, lk style.LookAndFeel, b paintengine2d.Rect, in style.Insets, st style.ControlState) bool {
	if in.Zero() {
		return false
	}
	style.DrawViewFrameOf(lk, ctx, b, st)
	ctx.Save()
	ctx.Translate(in.Left, in.Top)
	return true
}
