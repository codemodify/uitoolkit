package style

import "github.com/codemodify/paintengine2d"

// The Windows 11 window frame (see engine_fluent.go), from the Windows App
// SDK title bar guidance and WinUI's caption-button resources: a merged
// title bar on Mica, 32px (the app's content taller), a 1px window stroke,
// 8px corners (square when maximized or snapped), caption buttons 46px wide
// filling the title bar's height at the right, flat, with the subtle fills
// under the pointer and pressed and #c42b1c behind a white close glyph;
// glyphs 10px in the primary text colour, the disabled text colour in a
// backdrop window. The title sits at the left in the body face.

func (fluentEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	lw := winPx(l)
	r := l.rx(8)
	s := DecorationSpec{
		Border:  Insets{Top: lw, Right: lw, Bottom: lw, Left: lw},
		Caption: fluentCaptionH(l),
		Button:  paintengine2d.Pt(snap(l.S(46)), 0),
		Layout:  ":minimize,maximize,close",
		Radius:  [4]float32{r, r, r, r},
	}
	if st.Maximized || st.Tiled != 0 {
		s.Radius = [4]float32{}
	}
	s.Shadow = fluentEngine{}.PopupShadow(l, PopupDialog)
	return s
}

func (fluentEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := fluentColors(l)
	ctx.DrawRect(winSnap(f.Caption), paintengine2d.Fill(c.bg))
	if !st.Maximized {
		lw := winPx(l)
		// The window stroke is translucent: flatten it over the Mica.
		drawFrameBorder(ctx, f.Window, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, Mix(c.bg, c.windowStroke.WithAlpha(1), c.windowStroke.A))
	}
}

func (fluentEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := fluentColors(l)
	col := c.text
	if !st.Active {
		col = c.text3
	}
	captionTitle(l, ctx, l.body, b, title, col, false, l.S(12))
}

func (fluentEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := fluentColors(l)
	g := snap(l.S(10))
	flatCaption{
		fg: c.text, fgOff: c.text3,
		hover: c.subtle, press: c.subtle2,
		close: c.closeHot, closePress: c.closePress, onClose: c.onClose,
		min: g, max: g, cls: g, lw: winPx(l),
	}.draw(ctx, winSnap(b), k, cs, st)
}
