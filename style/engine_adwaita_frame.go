package style

import "github.com/codemodify/paintengine2d"

// The libadwaita window frame (see engine_adwaita.go): the window's header
// bar is its title bar — merged, 47px, in the header bar colour over its
// shade line — with a hairline edge round the window and the title centred
// in bold. Window controls are 24px round buttons at the ends (10px in, 6px
// apart), washed in the text colour at 10%, 15% under the pointer and 30%
// pressed — close too: GNOME's close is not red — with 16px symbolic
// glyphs, dimmed in a backdrop window. GTK 3's Adwaita (the gtk3 pack)
// keeps its flat square buttons that show a face under the pointer. The
// theme's own layout is GNOME's: close alone.

func (adwaitaEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := adwColors(l)
	lw := adwPx(l)
	h := snap(max(l.S(47), l.body.Height()+l.S(20)))
	side := snap(l.S(24))
	if c.gtk3 {
		h = snap(max(l.S(46), l.body.Height()+l.S(18)))
		side = snap(l.S(32))
	}
	r := adwR(l, 12, paintengine2d.XYWH(0, 0, 400, 300))
	s := DecorationSpec{
		Border:        Insets{Top: lw, Right: lw, Bottom: lw, Left: lw},
		Caption:       h,
		Button:        paintengine2d.Pt(side, side),
		ButtonGap:     snap(l.S(6)),
		ButtonPad:     Insets{Top: snap((h - side) * 0.5), Left: snap(l.S(10)) - lw, Right: snap(l.S(10)) - lw},
		CenterButtons: true,
		CenterTitle:   true,
		Layout:        ":close",
		Radius:        [4]float32{r, r, r, r},
		Shadow:        ShadowLayersReach(adwaitaWindowShadow(l, DecorationState{Active: true})),
	}
	if c.gtk3 {
		s.ButtonGap, s.ButtonPad.Left, s.ButtonPad.Right = 0, snap(l.S(6))-lw, snap(l.S(6))-lw
		s.Radius = [4]float32{r * 0.66, r * 0.66, 0, 0}
	}
	return s
}

// adwaitaWindowShadow is GNOME's window shadow, the one Adwaita's
// stylesheet gives a client-decorated window: 0 3px 9px 1px black at half
// alpha with a hairline layer under it, and a shallower pair in the
// backdrop.
func adwaitaWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{
			{Color: shadowBlack(0.2), DY: l.S(2), Blur: l.S(6), Spread: l.S(2)},
			{Color: shadowBlack(0.1), Blur: 0, Spread: l.S(1)},
		}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.5), DY: l.S(3), Blur: l.S(9), Spread: l.S(1)},
		{Color: shadowBlack(0.23), Blur: 0, Spread: l.S(1)},
	}
}

func (adwaitaEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, adwaitaWindowShadow(l, st))
}

func (adwaitaEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := adwColors(l)
	lw := adwPx(l)
	bar := adwSnap(f.Caption)
	bg := c.headerbar
	if !st.Active {
		bg = Mix(c.headerbar, c.win, 0.5)
	}
	ctx.DrawRect(bar, paintengine2d.Fill(bg))
	ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-lw, bar.Dx(), lw), paintengine2d.Fill(c.headerShade))
	if st.Maximized {
		return
	}
	edge := adwOver(c.win, adwInk(0.14))
	if c.dark {
		edge = adwOver(c.win, paintengine2d.RGBA(1, 1, 1, 0.1))
	}
	drawFrameBorder(ctx, f.Window, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, edge)
}

func (adwaitaEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := adwColors(l)
	col := c.text
	if !st.Active {
		col = c.dim
	}
	captionTitle(l, ctx, l.BoldFont(), b, title, col, true, l.S(8))
}

func (adwaitaEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := adwColors(l)
	b = adwSnap(b)
	fg := c.text
	if !st.Active && !cs.Hovered() {
		fg = c.dim
	}
	if c.gtk3 {
		fs := cs&(StateHovered|StatePressed) | StateAutoRaise
		if !st.Active {
			fs |= StateBackdrop
		}
		if cs.Hovered() || cs.Pressed() {
			fg = adwaitaEngine{}.Face(l, ctx, b, RoleTool, fs)
		}
	} else {
		a := float32(adwWashBtn)
		switch {
		case cs.Pressed():
			a = adwWashBtnActive
		case cs.Hovered():
			a = adwWashBtnHover
		}
		r := adwPill(l, b)
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(a)))
	}
	// The symbolic icons: 16px, drawn in 2px strokes on a 24px button.
	u := min(b.Dx(), b.Dy()) / 24
	if c.gtk3 {
		u = min(b.Dx(), b.Dy()) / 32
	}
	lw := max(snap(2*u), 1)
	g := adwCentered(b, 8*u, 8*u)
	switch k {
	case CaptionClose:
		p := paintengine2d.NewPath()
		p.MoveTo(g.Min.X, g.Min.Y)
		p.LineTo(g.Max.X, g.Max.Y)
		p.MoveTo(g.Max.X, g.Min.Y)
		p.LineTo(g.Min.X, g.Max.Y)
		ctx.DrawPath(p, adwStroke(fg, 2*u))
	default:
		DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, snap(10*u), lw)
	}
}
