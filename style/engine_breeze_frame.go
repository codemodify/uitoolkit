package style

import "github.com/codemodify/paintengine2d"

// The Plasma Breeze window frame (see engine_breeze.go), after the look of
// KWin's Breeze decoration: a merged title bar in the window manager's
// title-bar colour (the Header colour Plasma 6 continues into the tool
// bar), a hairline outline round the window, the title centred, and the
// buttons as glyphs on the bar — minimize a chevron down, maximize a
// chevron up, restore a diamond, close a cross, the window menu a small
// window — 18px, 4px apart, centred in the bar, 6px from its ends. Under
// the pointer a button fills a circle behind its glyph (red for close, the
// glyph turning the bar's colour); pressed, the circle deepens. A backdrop
// window's glyphs dim with its title.

func (breezeEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := brU(l)
	h := brTitleH(l)
	side := snap(min(l.S(18), h-l.S(4)))
	r := l.rx(3)
	s := DecorationSpec{
		Border:        Insets{Top: u, Right: u, Bottom: u, Left: u},
		Caption:       h,
		Button:        paintengine2d.Pt(side, side),
		ButtonGap:     snap(l.S(4)),
		ButtonPad:     Insets{Top: snap((h - side) * 0.5), Left: snap(l.S(6)) - u, Right: snap(l.S(6)) - u},
		CenterButtons: true,
		CenterTitle:   true,
		Layout:        "icon:minimize,maximize,close",
		Radius:        [4]float32{r, r, 0, 0},
		Shadow:        breezeEngine{}.PopupShadow(l, PopupDialog),
	}
	return s
}

func (breezeEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := breezeColors(l)
	u := brU(l)
	bar := c.title
	if !st.Active {
		bar = c.titleOff
	}
	ctx.DrawRect(fuSnap(f.Caption), paintengine2d.Fill(bar))
	if !st.Maximized {
		drawFrameBorder(ctx, f.Window, Insets{Top: u, Right: u, Bottom: u, Left: u}, Mix(bar, c.text, 0.35).WithAlpha(1))
	}
}

func (breezeEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := breezeColors(l)
	col := c.titleText
	if !st.Active {
		col = c.titleTextOff
	}
	captionTitle(l, ctx, l.body, b, title, col, true, l.S(8))
}

func (breezeEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := breezeColors(l)
	u := brU(l)
	bar, fg := c.title, c.titleText
	if !st.Active {
		bar, fg = c.titleOff, c.titleTextOff
	}
	b = fuSnap(b)
	ctr := b.Center()
	rad := min(b.Dx(), b.Dy()) * 0.5
	glyph := fg
	switch {
	case k == CaptionClose && cs.Pressed():
		ctx.DrawCircle(ctr, rad, paintengine2d.Fill(darkerPct(c.negative, 200)))
		glyph = bar
	case k == CaptionClose && cs.Hovered():
		ctx.DrawCircle(ctr, rad, paintengine2d.Fill(lighterPct(c.negative, 150)))
		glyph = bar
	case cs.Pressed():
		ctx.DrawCircle(ctr, rad, paintengine2d.Fill(fg.WithAlpha(0.45)))
	case cs.Hovered():
		ctx.DrawCircle(ctr, rad, paintengine2d.Fill(fg.WithAlpha(0.2)))
	}
	// Glyphs in the 18px box's proportions.
	k18 := b.Dx() / 18
	g := paintengine2d.XYWH(b.Min.X+5*k18, b.Min.Y+5*k18, 8*k18, 8*k18)
	stroke := paintengine2d.Paint{Color: glyph, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: max(u*1.001, 1), Cap: paintengine2d.CapRound, Join: paintengine2d.JoinMiter, MiterLimit: 4}}
	p := paintengine2d.NewPath()
	switch k {
	case CaptionClose:
		DrawCross(ctx, g, glyph, u*1.001)
		return
	case CaptionMinimize:
		p.MoveTo(g.Min.X, g.Min.Y+2*k18)
		p.LineTo((g.Min.X+g.Max.X)*0.5, g.Max.Y-2*k18)
		p.LineTo(g.Max.X, g.Min.Y+2*k18)
	case CaptionMaximize:
		if st.Maximized {
			cx, cy := (g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5
			p.MoveTo(cx, g.Min.Y)
			p.LineTo(g.Max.X, cy)
			p.LineTo(cx, g.Max.Y)
			p.LineTo(g.Min.X, cy)
			p.Close()
		} else {
			p.MoveTo(g.Min.X, g.Max.Y-2*k18)
			p.LineTo((g.Min.X+g.Max.X)*0.5, g.Min.Y+2*k18)
			p.LineTo(g.Max.X, g.Max.Y-2*k18)
		}
	case CaptionMenu:
		p.AddRect(g.Inset(k18 * 0.5))
		p.MoveTo(g.Min.X+k18*0.5, g.Min.Y+2.5*k18)
		p.LineTo(g.Max.X-k18*0.5, g.Min.Y+2.5*k18)
	}
	ctx.DrawPath(p, stroke)
}
