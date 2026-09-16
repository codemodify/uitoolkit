package style

import "github.com/codemodify/paintengine2d"

// The Mac OS X 10.0–10.4 window frame (see engine_aqua.go): a stacked
// frame. The 22px title bar in its light gradient with pinstripes (brushed
// metal in the metal packs), a line under it and a hairline round the
// window; the title centred on the window; the three traffic lights, 13px
// gel drops 20px apart from 8px in, where the theme's layout puts them on
// the left (close, minimize, zoom), grey in a backdrop window. The pointer
// over a light shows its glyph (×, −, +); a pressed light darkens. The
// app's title bar sits under the title bar on the window's pinstripes, as
// the tool bars of the era's windows did.

// aquaFrameU is the frame's hairline on whole pixels.
func aquaFrameU(l *Classic) float32 { return max(snap(aquaU(l)), 1) }

func (aquaEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := aquaFrameU(l)
	bar := aquaTitleH(l)
	// A light's box holds its drop shadow: a hairline more at the sides,
	// two below.
	d := aquaLight(l, paintengine2d.XYWH(0, 0, l.S(400), l.S(300)), 0).Dx()
	top := snap((bar - d) * 0.5)
	pad := snap(l.S(8)) - 2*u
	r := l.S(5)
	return DecorationSpec{
		Stacked:     true,
		Border:      Insets{Top: u, Right: u, Bottom: u, Left: u},
		Caption:     bar,
		Button:      paintengine2d.Pt(d+2*u, d+2*u),
		ButtonGap:   snap(l.S(20)) - d - 2*u,
		ButtonPad:   Insets{Top: top, Left: pad, Right: pad},
		CenterTitle: true,
		Layout:      "close,minimize,maximize:",
		Radius:      [4]float32{r, r, 0, 0},
		Shadow:      ShadowLayersReach(aquaWindowShadow(l, DecorationState{Active: true})),
	}
}

// aquaWindowShadow is Aqua's famous drop shadow: deep and soft under the
// active window, faint under the others (measured off Mac OS X 10.4
// screenshots — Apple published no numbers).
func aquaWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.2), DY: l.S(3), Blur: l.S(12)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.38), DY: l.S(14), Blur: l.S(40)},
		{Color: shadowBlack(0.16), DY: l.S(2), Blur: l.S(6)},
	}
}

func (aquaEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, aquaWindowShadow(l, st))
}

func (aquaEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := aquaColors(l)
	u := aquaFrameU(l)
	bar := f.Caption
	if st.Maximized {
		bar = paintengine2d.Rect{Min: paintengine2d.Pt(f.Window.Min.X, bar.Min.Y), Max: paintengine2d.Pt(f.Window.Max.X, bar.Max.Y)}
	}
	if c.metal {
		c.texture(l, ctx, bar)
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), u), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.55)))
	} else {
		stops := c.titleStops
		if !st.Active {
			// A backdrop window's title bar pales toward the window.
			stops = make([]paintengine2d.GradientStop, len(c.titleStops))
			for i, s := range c.titleStops {
				stops[i] = Stop(s.Offset, Mix(s.Color, c.win, 0.45))
			}
		}
		ctx.DrawRect(bar, VGradient(bar, stops...))
		aquaStripes(l, ctx, bar, paintengine2d.RGBA(1, 1, 1, 0), c.titleLo, c.titleHi, c.period)
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-u, bar.Dx(), u), paintengine2d.Fill(c.titleEdge))
	}
	if !st.Maximized {
		drawFrameBorder(ctx, f.Window, Insets{Top: u, Right: u, Bottom: u, Left: u}, Mix(c.titleEdge, c.win, 0.2))
	}
}

func (aquaEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := aquaColors(l)
	col := c.titleTxt
	if !st.Active {
		col = c.titleOff
	}
	u := aquaFrameU(l)
	captionTitle(l, ctx, l.body, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u), title, col, true, l.S(8))
}

func (aquaEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := aquaColors(l)
	u := aquaU(l)
	fu := aquaFrameU(l)
	b = paintengine2d.XYWH(b.Min.X+fu, b.Min.Y, b.Dx()-2*fu, b.Dy()-2*fu)
	i := 0
	switch k {
	case CaptionMinimize:
		i = 1
	case CaptionMaximize:
		i = 2
	case CaptionMenu:
		i = -1
	}
	g := &c.lightOff
	if i >= 0 && (st.Active || cs.Hovered()) {
		g = &c.lights[i]
	}
	if cs.Pressed() {
		p := g.toward(paintengine2d.RGB(0, 0, 0), 0.25, 0.9)
		p.fg = g.fg
		g = &p
	}
	aquaDrop(ctx, b, b.Dx()*0.5, u, 0.18)
	aquaGelPaint(ctx, b, b.Dx()*0.5, u, g, false)
	if !cs.Hovered() && !cs.Pressed() && i >= 0 {
		return
	}
	fg := g.fg
	s := b.Inset(b.Dx() * 0.3)
	lw := max(l.S(1.5), 1)
	switch k {
	case CaptionClose:
		DrawCross(ctx, s, fg, lw)
	case CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(s.Min.X, snap((s.Min.Y+s.Max.Y-lw)*0.5), s.Dx(), max(snap(lw), 1)), paintengine2d.Fill(fg))
	default:
		// Zoom's plus (and the window menu's).
		t := max(snap(lw), 1)
		ctx.DrawRect(paintengine2d.XYWH(s.Min.X, snap((s.Min.Y+s.Max.Y-t)*0.5), s.Dx(), t), paintengine2d.Fill(fg))
		ctx.DrawRect(paintengine2d.XYWH(snap((s.Min.X+s.Max.X-t)*0.5), s.Min.Y, t, s.Dy()), paintengine2d.Fill(fg))
	}
}
