package style

import "github.com/codemodify/paintengine2d"

// The Plastik window decoration (see engine_plastik.go): a stacked frame,
// the default of KDE 3.4 and 3.5. The window colour runs round the body in a
// four-pixel border read from the outside in — a dark contour, a light line,
// the caption colour, then a darker line round the client — and its top
// corners are rounded three pixels. The title bar is a gentle vertical
// gradient in the caption colour, lit along its top, with the bold title at
// the left over its one-pixel shadow; the buttons at the right are small
// square surfaces, a light frame round a lighter face that brightens under
// the pointer and sinks when pressed.
//
// KDE 3 shaped its corners with the X Shape extension and had no compositor,
// so: rounded top, square bottom, no shadow. The buttons sit where kwin put
// them by default, which is the layout the pack asks for.
//
// Shapes and colours are engine_plastik.go's, measured on stock KDE 3.5 and
// Qt 4.8 Plastique screenshots.

func plFrameBorder(l *Classic) Insets {
	fr := 4 * kde3U(l)
	return Insets{Right: fr, Bottom: fr, Left: fr}
}

func (plastikEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := kde3U(l)
	h := plCaptionH(l)
	side := snap(h - l.S(8))
	fr := plFrameBorder(l)
	r := l.rx(3)
	return DecorationSpec{
		Stacked:   true,
		Border:    fr,
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(3)),
		ButtonPad: Insets{
			Top:   snap((h-side)*0.5 + u*0.5),
			Right: max(snap(l.S(6))-fr.Right, 0),
			Left:  max(snap(l.S(6))-fr.Left, 0),
		},
		Layout: "icon:minimize,maximize,close",
		Radius: [4]float32{r, r, 0, 0},
	}
}

func (plastikEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := plastikColors(l)
	u := kde3U(l)
	border := plFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	b := kde3Snap(f.Window)
	capBottom := kde3Snap(f.Caption).Max.Y
	base, stops, out, lit, line := c.capBase, c.cap, c.capOut, c.capLit, c.capLine
	if !st.Active {
		base, stops, out, lit, line = c.capOffBase, c.capOff, c.capOffOut, c.capOffLit, c.capOffLine
	}
	r := l.rx(3)
	if st.Maximized {
		r = 0
	}
	body := paintengine2d.Rect{
		Min: paintengine2d.Pt(b.Min.X+border.Left, capBottom),
		Max: paintengine2d.Pt(b.Max.X-border.Right, b.Max.Y-border.Bottom),
	}
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		kde3Rounded(ctx, b, r, true, true, false, true, paintengine2d.Fill(out))
		b1, r1 := b.Inset(u), max(r-u, 0)
		kde3Rounded(ctx, b1, r1, true, true, false, true, paintengine2d.Fill(lit))
		b2, r2 := b1.Inset(u), max(r1-u, 0)
		kde3Rounded(ctx, b2, r2, true, true, false, true, paintengine2d.Fill(base))
		bar := paintengine2d.XYWH(b2.Min.X, b2.Min.Y, b2.Dx(), capBottom-b2.Min.Y-u)
		kde3Rounded(ctx, bar, r2, true, true, false, true, VGradient(bar, stops...))
		// The darker line round the client.
		ctx.DrawRect(body.Inset(-u), paintengine2d.Fill(line))
		ctx.Restore()
	}
}

func (plastikEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := plastikColors(l)
	u := kde3U(l)
	fg, shadow := c.capText, c.capShadow
	if !st.Active {
		fg, shadow = c.capOffText, c.capOffShadow
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(6), b.Dy())
	if shadow.A > 0 {
		l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(u, u)), shadow, AlignStart, 0)
	}
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
}

func (plastikEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := plastikColors(l)
	u := kde3U(l)
	white := paintengine2d.RGB(1, 1, 1)
	base, glyph := c.capBase, c.capText
	if !st.Active {
		base, glyph = c.capOffBase, c.capOffText
	}
	b = kde3Snap(b)
	r := l.rx(2)
	face := Mix(base, white, 0.12)
	switch {
	case cs.Pressed():
		face = darkerPct(base, 118)
	case cs.Hovered():
		face = Mix(base, white, 0.32)
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(Mix(base, white, 0.45)))
	ctx.DrawRoundRect(b.Inset(u), max(r-u, 0), max(r-u, 0), paintengine2d.Fill(face))
	if k == CaptionClose {
		DrawCross(ctx, b.Inset(b.Dx()*0.3), glyph, max(l.S(2), u))
		return
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, glyph, b.Dx()*0.42, max(l.S(1.5), u))
}
