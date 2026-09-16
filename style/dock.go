package style

import "github.com/codemodify/paintengine2d"

// DockLook is implemented by looks that paint the mark a dragged dock
// panel's drop target gets themselves. Looks that do not get the default
// below, which is built from their own palette and metrics, so every pack
// marks a drop target in its own colours without an engine of its own.
type DockLook interface {
	// DrawDropIndicator fills b to show where a dragged panel would land.
	// tab is set when the drop would put the panel into an existing stack
	// as another tab rather than beside it.
	DrawDropIndicator(ctx *paintengine2d.Context, b paintengine2d.Rect, tab bool)
}

// DrawDropIndicatorOf marks b as where a dragged dock panel would land: a
// wash of the pack's accent under its outline, and a tab stub along the top
// when the drop would tab the panel into the stack it is over.
//
// Qt Creator, Visual Studio and VS Code all mark the target this way; the
// alternative — a compass of buttons floating over the window — needs
// art of its own in every one of the packs and reads worse at a glance.
func DrawDropIndicatorOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, tab bool) {
	if lk == nil || ctx == nil || b.Empty() {
		return
	}
	if d, ok := lk.(DockLook); ok {
		d.DrawDropIndicator(ctx, b, tab)
		return
	}
	drawPlainDropIndicator(lk, ctx, b, tab)
}

// dropIndicatorInk is the colour a pack marks a drop target in: its accent,
// falling back to the focus and selection colours for packs whose accent is
// transparent (the plainest monochrome looks).
func dropIndicatorInk(p Palette) paintengine2d.Color {
	for _, c := range []paintengine2d.Color{p.Accent, p.Focus, p.Selection, p.Text} {
		if c.A > 0 {
			return c
		}
	}
	return paintengine2d.RGB(0, 0, 0)
}

// drawPlainDropIndicator is the default mark, built from the pack's own
// palette and metrics.
func drawPlainDropIndicator(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, tab bool) {
	p := lk.Palette()
	ink := dropIndicatorInk(p)
	r := lk.Metrics().Radius
	w := Dip(lk, 2)
	if w < 1 {
		w = 1
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(ink.WithAlpha(0.18)))
	ctx.DrawRoundRect(b.Inset(w*0.5), r, r, paintengine2d.StrokePaint(ink, w))
	if !tab {
		return
	}
	// A stub of a tab along the top says the panel joins this stack rather
	// than splitting it.
	th := Dip(lk, 4)
	tw := b.Dx() * 0.4
	if maxTab := Dip(lk, 96); tw > maxTab {
		tw = maxTab
	}
	if tw < 1 || th < 1 || th > b.Dy() {
		return
	}
	stub := paintengine2d.XYWH(b.Min.X+w, b.Min.Y+w, tw, th)
	ctx.DrawRect(stub, paintengine2d.Fill(ink))
}
