package style

import "github.com/codemodify/paintengine2d"

// The Windows 95 to 2000 window frame (see engine_win95.go): a stacked
// frame, as every Windows of the era drew it. The 4px raised sizing border
// in the face colour; the caption bar inset 3px from the window's edge, in
// the active caption colour (Windows 98's and 2000's two-colour gradient
// where the scheme has one, grey in the backdrop) with the title in bold at
// its left; a face-coloured line under it, then the app's title-bar row.
// Caption buttons are bevelled boxes (16×14 at 96 DPI: the bar's height
// less 4, two wider), 2px below the bar's top, minimize and maximize side
// by side and close 2px apart, its right edge 2px in from the bar's end.
// Their glyphs are drawn the way the era's Marlett font shaped them: a thick
// bar at the bottom, a square with a thick top edge, two of them for
// restore, a heavy cross; a pressed button sinks and its glyph moves one
// pixel down and right. Maximized, the borders are off the screen and the
// bar runs from edge to edge.

// w95FrameGeom are the frame's lengths: the sizing border, the caption
// bar's inset from the window's edge and its height.
func w95FrameGeom(l *Classic) (frame, inset, bar float32) {
	return max(snap(l.S(4)), 2), max(snap(l.S(3)), 2), snap(w95CaptionH(l))
}

func (win95Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	fr, in, bar := w95FrameGeom(l)
	bh := bar - snap(l.S(4))
	bw := bh + snap(l.S(2))
	gap := max(snap(l.S(1)), 1)
	// The bar reaches 1px into the side borders; the buttons keep 2px from
	// its ends.
	pad := snap(l.S(2))
	if !st.Maximized {
		pad -= fr - in
	}
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Top: in, Right: fr, Bottom: fr, Left: fr},
		Caption:   bar + gap,
		Button:    paintengine2d.Pt(bw, bh),
		CloseGap:  snap(l.S(2)),
		ButtonPad: Insets{Top: snap(l.S(2)), Right: pad, Left: pad},
		Layout:    "icon:minimize,maximize,close",
	}
}

func (win95Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := w95colors(l)
	fr, in, barH := w95FrameGeom(l)
	w := f.Window
	border := Insets{Top: in, Right: fr, Bottom: fr, Left: fr}
	ext := fr - in
	if st.Maximized {
		border, ext = Insets{}, 0
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.face))
	}
	if !st.Maximized {
		c.raisedWindow(ctx, w)
	}
	bar := paintengine2d.XYWH(f.Caption.Min.X-ext, f.Caption.Min.Y, f.Caption.Dx()+2*ext, barH)
	c1, c2 := c.cap1, c.cap2
	if !st.Active {
		c1, c2 = c.capOff1, c.capOff2
	}
	if c1 == c2 {
		ctx.DrawRect(bar, paintengine2d.Fill(c1))
	} else {
		ctx.DrawRect(bar, HGradient(bar, Stop(0, c1), Stop(1, c2)))
	}
}

func (win95Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := w95colors(l)
	_, _, barH := w95FrameGeom(l)
	col := c.capTxt
	if !st.Active {
		col = c.capOffTxt
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	pad := snap(l.S(4))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad, barH), col, AlignStart, 0)
}

func (win95Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	if cs.Pressed() {
		c.pushed(ctx, b)
	} else {
		c.raised(ctx, b)
	}
	g := b
	if cs.Pressed() {
		g = g.Translate(paintengine2d.Pt(1, 1))
	}
	w95Glyph(ctx, g, k, st.Maximized, c.text)
}

// w95Glyph draws a caption glyph in the proportions of the era's 16×14
// buttons, scaled to b and kept on whole pixels.
func w95Glyph(ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, maximized bool, col paintengine2d.Color) {
	u := max(b.Dy()/14, 1)
	px := func(v float32) float32 { return snap(v * u) }
	t := max(px(2), 2) // the thick strokes
	lw := max(px(1), 1)
	// The glyph cell: 16×14 units centred in the button.
	x0 := snap(b.Min.X + (b.Dx()-16*u)*0.5)
	y0 := snap(b.Min.Y + (b.Dy()-14*u)*0.5)
	fill := paintengine2d.Fill(col)
	box := func(x, y, w, h float32) {
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(x, y, w, t))
		p.AddRect(paintengine2d.XYWH(x, y+h-lw, w, lw))
		p.AddRect(paintengine2d.XYWH(x, y+t, lw, h-t-lw))
		p.AddRect(paintengine2d.XYWH(x+w-lw, y+t, lw, h-t-lw))
		ctx.DrawPath(p, fill)
	}
	switch k {
	case CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(x0+px(4), y0+px(9), px(6), t), fill)
	case CaptionMaximize:
		if !maximized {
			box(x0+px(3), y0+px(2), px(9), px(9))
			return
		}
		// Restore: the back window's top and right edges, the front one
		// whole.
		back := paintengine2d.XYWH(x0+px(5), y0+px(1), px(6), px(6))
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(back.Min.X, back.Min.Y, back.Dx(), t))
		p.AddRect(paintengine2d.XYWH(back.Max.X-lw, back.Min.Y+t, lw, back.Dy()-t))
		p.AddRect(paintengine2d.XYWH(back.Min.X, back.Min.Y+t, lw, px(2)))
		ctx.DrawPath(p, fill)
		box(x0+px(3), y0+px(4), px(6), px(6))
	case CaptionClose:
		// A heavy cross: two bars as wide as the thick strokes, their ends
		// cut flat.
		lunaCross(ctx, paintengine2d.XYWH(x0+px(4)+t*0.5, y0+px(3)+t*0.3, px(8)-t, px(7)-t*0.6), col, t*0.85)
	case CaptionMenu:
		// The control-menu box: a window with a thick title bar.
		box(x0+px(4), y0+px(3), px(8), px(7))
	}
}
