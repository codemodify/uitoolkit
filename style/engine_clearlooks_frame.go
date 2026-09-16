package style

import "github.com/codemodify/paintengine2d"

// The Metacity window frame of the GNOME 2 era (see engine_clearlooks.go):
// a stacked frame, and the one Clearlooks, Cleanlooks, Ubuntu's Human and
// Red Hat's Bluecurve all wore, since Bluecurve is the engine Clearlooks
// grew out of and inherits this frame (engine_bluecurve.go).
//
// A dark outline round the window with a light inner edge; the title bar in
// three bands of the theme's caption colour, lit along its top and closed
// off by a darker line, with the bold title centred over its own shadow; the
// buttons at the right are small rounded buttons of the same colour that
// lighten under the pointer and darken when pressed, the close one carrying
// a white ×. The border is four pixels down the sides and along the bottom.
//
// The top corners are the theme's: Clearlooks, Cleanlooks and Human round
// them five pixels, Bluecurve — square everywhere, as Red Hat drew it —
// leaves them square. Metacity shipped without a compositor, so no window
// casts a shadow. The buttons sit where GNOME 2 put them, which is the
// layout the packs ask for: the window menu at the left, minimize, maximize
// and close at the right.
//
// Shapes and colours are engine_clearlooks.go's, from the GTK 2 engine's
// documented shading factors and from screenshots of the stock themes.

// clFrameBorder is the frame's border: four pixels down the sides and along
// the bottom, the title bar taking the top.
func clFrameBorder(l *Classic) Insets {
	fr := snap(l.S(4))
	return Insets{Right: fr, Bottom: fr, Left: fr}
}

// clFrameBar is the title bar's own height inside the frame's outline.
func clFrameBar(l *Classic) float32 {
	return clCaptionH(l) + snap(l.S(4)) - 2*clPx(l)
}

func (clearlooksEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	lw := clPx(l)
	h := clFrameBar(l)
	side := snap(h - l.S(8))
	r := l.rx(5)
	return DecorationSpec{
		Stacked:   true,
		Border:    clFrameBorder(l),
		Caption:   clCaptionH(l) + snap(l.S(4)),
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(2)),
		// The buttons keep the outline's width from the caption's ends: the
		// border under the bar is already four pixels wide.
		ButtonPad:   Insets{Top: lw + snap((h-side)*0.5), Right: lw, Left: lw},
		CenterTitle: true,
		Layout:      "icon:minimize,maximize,close",
		Radius:      [4]float32{r, r, 0, 0},
	}
}

func (clearlooksEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := clColors(l)
	lw := clPx(l)
	b := clSnap(f.Window)
	if b.Dx() < 12*lw || b.Dy() < 12*lw {
		return
	}
	border := clFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	base := c.caption
	stops := c.capGrad
	if !st.Active {
		base, stops = c.bg, c.capOff
	}
	r := clR(l, 5, b)
	if st.Maximized {
		r = 0
	}
	ri := max(r-lw, 0)
	in := b.Inset(lw)
	// The bar runs the window's whole width inside the outline; the border
	// under it is what the sides are inset by.
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), max(clSnap(f.Caption).Max.Y-lw-in.Min.Y, 0))
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		ctx.DrawRoundRectCorners(b, r, r, 0, 0, paintengine2d.Fill(clShade(base, 0.45)))
		ctx.DrawRoundRectCorners(in, ri, ri, 0, 0, paintengine2d.Fill(c.bg))
		ctx.DrawRoundRectCorners(bar, ri, ri, 0, 0, VGradient(bar, stops...))
		ctx.Save()
		ctx.ClipPath(RoundRectPath(bar, ri, ri, 0, 0))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), lw), paintengine2d.Fill(clShade(base, 1.2)))
		ctx.Restore()
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y, bar.Dx(), lw), paintengine2d.Fill(clShade(base, 0.5)))
		ctx.Restore()
	}
}

func (clearlooksEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := clColors(l)
	lw := clPx(l)
	f := l.BoldFont()
	if !st.Active {
		l.drawFittedText(ctx, f, title, b, c.captionOff, AlignCenter, 8)
		return
	}
	l.drawFittedText(ctx, f, title, b.Translate(paintengine2d.Pt(lw, lw)), clShade(c.caption, 0.5), AlignCenter, 8)
	l.drawFittedText(ctx, f, title, b, c.captionText, AlignCenter, 8)
}

func (clearlooksEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := clColors(l)
	lw := clPx(l)
	base, glyph := c.caption, c.captionText
	if !st.Active {
		base, glyph = c.bg, Mix(c.bg, c.fg, 0.55)
	}
	r := clR(l, 3, b)
	fill := paintengine2d.Paint(VGradient(b, c.capBtn...))
	switch {
	case !st.Active:
		fill = VGradient(b, c.capOffBtn...)
	case cs.Pressed():
		fill = VGradient(b, c.capDown...)
	case cs.Hovered():
		fill = VGradient(b, c.capHot...)
	}
	clFrame(ctx, b, r, lw, paintengine2d.Fill(clShade(base, 0.6)), fill)
	if st.Active {
		clLit(ctx, b.Inset(lw), max(r-lw, 0), lw, clShade(base, 1.18), paintengine2d.Color{})
	}
	if k == CaptionClose {
		g := b.Inset(snap(b.Dx() * 0.3))
		p := paintengine2d.NewPath()
		p.MoveTo(g.Min.X, g.Min.Y)
		p.LineTo(g.Max.X, g.Max.Y)
		p.MoveTo(g.Max.X, g.Min.Y)
		p.LineTo(g.Min.X, g.Max.Y)
		ctx.DrawPath(p, paintengine2d.Paint{Color: glyph, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 2 * lw, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
		return
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, glyph, snap(b.Dx()*0.42), max(lw, 1))
}
