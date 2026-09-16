package style

import "github.com/codemodify/paintengine2d"

// The Windows 8 and 10 window frames (see engine_metro.go).
//
// Windows 10: a merged white (dark: #2b2b2b) title bar over a hairline
// border, caption buttons 46px wide filling its height, flat, a 10% wash of
// the text colour under the pointer and 20% pressed, the close button red
// (#e81123, #f1707a pressed) behind a white glyph, 10px glyphs; the title at
// the left.
//
// Windows 8: a stacked frame, the thick flat window colour all round (8px,
// grey in the backdrop) with the title centred in the caption; minimize and
// maximize 26px flat on the frame with a lighter wash under the pointer,
// close a 46px red button hanging from the top edge, all 20px tall.

func (metroEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := metroColors(l)
	capH := metroCaptionH(l)
	lw := winPx(l)
	s := DecorationSpec{Layout: ":minimize,maximize,close"}
	if c.win10 {
		s.Border = Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}
		s.Caption = capH
		s.Button = paintengine2d.Pt(snap(l.S(46)), 0)
		s.Shadow = ShadowLayersReach(metroWindowShadow(l, DecorationState{Active: true}))
		return s
	}
	// Windows 8 windows were flat on the desktop: a thick coloured frame
	// and no shadow at all.
	fw := metroFrameW(l)
	h := snap(capH * 0.62)
	s.Stacked = true
	s.Border = Insets{Top: lw, Right: fw, Bottom: fw, Left: fw}
	s.Caption = capH - lw
	s.Button = paintengine2d.Pt(snap(h*26/20), h)
	s.CloseButton = paintengine2d.Pt(snap(h*46/20), h)
	return s
}

func (metroEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := metroColors(l)
	lw := winPx(l)
	frame, border := c.frame, c.frameBorder
	if !st.Active {
		frame, border = c.frameOff, c.frameBorderOff
	}
	ctx.DrawRect(winSnap(f.Caption), paintengine2d.Fill(frame))
	if st.Maximized {
		return
	}
	w := f.Window
	if c.win10 {
		drawFrameBorder(ctx, w, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, border)
		return
	}
	// Windows 8: the coloured frame round the client, in a line of its
	// border colour outside and in.
	fw := metroFrameW(l)
	drawFrameBorder(ctx, w, Insets{Top: f.Caption.Max.Y - w.Min.Y, Right: fw, Bottom: fw, Left: fw}, frame)
	drawFrameBorder(ctx, w, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, border)
	client := paintengine2d.Rect{Min: paintengine2d.Pt(w.Min.X+fw, f.Caption.Max.Y), Max: paintengine2d.Pt(w.Max.X-fw, w.Max.Y-fw)}
	drawFrameBorder(ctx, client.Inset(-lw), Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, border)
}

func (metroEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := metroColors(l)
	col := c.capText
	if !st.Active {
		col = c.capTextOff
	}
	captionTitle(l, ctx, l.body, b, title, col, !c.win10, l.S(8))
}

func (metroEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := metroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	fg, fgOff := c.capText, c.capTextOff
	if c.win10 {
		g := snap(l.S(10))
		flatCaption{
			fg: fg, fgOff: fgOff,
			hover: fg.WithAlpha(0.1), press: fg.WithAlpha(0.2),
			close: c.closeHot, closePress: c.closePress, onClose: c.closeGlyphHot,
			min: g, max: g, cls: g, lw: lw,
		}.draw(ctx, b, k, cs, st)
		return
	}
	g := snap(min(b.Dy()*0.45, l.S(10)))
	w := max(lw*1.6, b.Dy()*0.1)
	if k == CaptionClose {
		fill, glyph := c.close, c.closeGlyph
		switch {
		case !st.Active:
			fill, glyph = c.closeOff, c.closeGlyphOff
		case cs.Pressed():
			fill, glyph = c.closePress, c.closeGlyphHot
		case cs.Hovered():
			fill, glyph = c.closeHot, c.closeGlyphHot
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		DrawCaptionGlyph(ctx, b, k, st.Maximized, glyph, g, w)
		return
	}
	switch {
	case cs.Pressed():
		ctx.DrawRect(b, paintengine2d.Fill(fg.WithAlpha(0.25)))
	case cs.Hovered():
		ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(1, 1, 1).WithAlpha(0.35)))
	}
	col := fg
	if !st.Active {
		col = fgOff
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, col, g, w)
}

// metroWindowShadow is the tight shadow a Windows 10 window casts (Windows
// 8's flat windows had none).
func metroWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.14), DY: l.S(1), Blur: l.S(8)}}
	}
	return []WindowShadow{{Color: shadowBlack(0.25), DY: l.S(3), Blur: l.S(14)}}
}

func (metroEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	if !metroColors(l).win10 {
		return
	}
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, metroWindowShadow(l, st))
}
