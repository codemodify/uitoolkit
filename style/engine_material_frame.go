package style

import "github.com/codemodify/paintengine2d"

// The Material window frame (see engine_material.go). Material Design has
// no desktop title bar of its own; this one follows its components and
// Chrome OS's frames: a merged title bar in the top app bar's surface (at
// 3dp elevation in Material 2, the surface container in Material 3) with a
// hairline border, the title in bold at the left, and the caption buttons
// as 32px icon buttons: 20px icons in the medium-emphasis colour on a
// circular state layer under the pointer (8%, 12% pressed, of the colour on
// the surface) — close included, as in Material and Chrome OS; 8px apart.

func (materialEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	px := mdPx(l)
	h := snap(max(l.BoldFont().Height()+l.S(20), l.S(40)))
	side := snap(min(l.S(32), h-l.S(8)))
	return DecorationSpec{
		Border:        Insets{Top: px, Right: px, Bottom: px, Left: px},
		Caption:       h,
		Button:        paintengine2d.Pt(side, side),
		ButtonGap:     snap(l.S(4)),
		ButtonPad:     Insets{Top: snap((h - side) * 0.5), Left: snap(l.S(8)) - px, Right: snap(l.S(8)) - px},
		CenterButtons: true,
		Layout:        ":minimize,maximize,close",
		Shadow:        ShadowLayersReach(materialWindowShadow(l, DecorationState{Active: true})),
	}
}

func (materialEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := mdColors(l)
	bar := c.bar
	if !st.Active {
		bar = Mix(c.bar, c.bg, 0.5)
	}
	ctx.DrawRect(mdSnap(f.Caption), paintengine2d.Fill(bar))
	if !st.Maximized {
		px := mdPx(l)
		drawFrameBorder(ctx, f.Window, Insets{Top: px, Right: px, Bottom: px, Left: px}, c.divider)
	}
}

func (materialEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := mdColors(l)
	col := c.text
	if !st.Active {
		col = c.text2
	}
	captionTitle(l, ctx, l.BoldFont(), b, title, col, false, l.S(16))
}

func (materialEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := mdColors(l)
	b = mdSnap(b)
	fg := c.text2
	if !st.Active && !cs.Hovered() {
		fg = c.textDis
	}
	a := float32(0)
	switch {
	case cs.Pressed():
		a = c.aPress
	case cs.Hovered():
		a = c.aHover
	}
	if a > 0 {
		ctx.DrawCircle(b.Center(), min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	s := snap(min(b.Dx(), b.Dy()) * 20 / 32 * 0.6)
	DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, s, max(snap(l.S(1.5)), 1))
}

// materialWindowShadow is Material's elevation under a window: the key and
// ambient pair of a raised surface.
func materialWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.2), DY: l.S(2), Blur: l.S(6)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.2), DY: l.S(8), Blur: l.S(10)},
		{Color: shadowBlack(0.14), DY: l.S(3), Blur: l.S(14), Spread: l.S(2)},
	}
}

func (materialEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, materialWindowShadow(l, st))
}
