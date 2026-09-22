package style

import "github.com/codemodify/paintengine2d"

// The Windows 7 window frame (see engine_aero.go), as an opaque
// approximation of the glass (translucency is Phase 3 of
// docs/decorations.md): the frame colour's gradient with its sheen and two
// soft streaks all round the window — the caption merged with the app's
// title bar, as Windows 7 extended the glass under Explorer's address bar
// and Office's quick access tool bar — a dark outer edge, a line round the
// client area. The caption buttons hang from the top edge as one joined
// group, 19px tall at 96 DPI: minimize and maximize 26px wide in frosted
// glass that glows blue under the pointer, close 43px in red glass. The
// title is black at the left over a soft white glow.

// aeroMinMax are the glass faces of minimize and maximize: at rest, under
// the pointer, pressed and in a backdrop window.
var aeroMinMax = [4]aeroGlass{
	{border: paintengine2d.RGBA(0.18, 0.24, 0.33, 0.55), hi: paintengine2d.RGBA(1, 1, 1, 0.5),
		face: []paintengine2d.GradientStop{Stop(0, Hex("#eef4fb")), Stop(0.48, Hex("#dce7f4")), Stop(0.48, Hex("#c7d7ea")), Stop(1, Hex("#d7e3f1"))}},
	{border: paintengine2d.RGBA(0.1, 0.2, 0.36, 0.7), hi: paintengine2d.RGBA(1, 1, 1, 0.6),
		face: []paintengine2d.GradientStop{Stop(0, Hex("#c2e3fb")), Stop(0.48, Hex("#8ec9f2")), Stop(0.48, Hex("#5eaee8")), Stop(1, Hex("#86c6f0"))}},
	{border: paintengine2d.RGBA(0.06, 0.15, 0.3, 0.8),
		face: []paintengine2d.GradientStop{Stop(0, Hex("#8fb6d9")), Stop(0.48, Hex("#5d92c4")), Stop(0.48, Hex("#3a74ad")), Stop(1, Hex("#5a91c4"))}},
	{border: paintengine2d.RGBA(0.35, 0.42, 0.5, 0.45), hi: paintengine2d.RGBA(1, 1, 1, 0.5),
		face: []paintengine2d.GradientStop{Stop(0, Hex("#f3f6fa")), Stop(0.48, Hex("#e8edf4")), Stop(0.48, Hex("#dde4ee")), Stop(1, Hex("#e6ebf2"))}},
}

func (aeroEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	lw := winPx(l)
	fw := aeroFrameW(l)
	capH := aeroCaptionH(l)
	h := snap(capH * 0.62)
	r := l.rx(6)
	s := DecorationSpec{
		Border:      Insets{Top: lw, Right: fw, Bottom: fw, Left: fw},
		Caption:     capH - lw,
		Button:      paintengine2d.Pt(snap(h*26/19), h),
		CloseButton: paintengine2d.Pt(snap(h*43/19), h),
		ButtonGap:   -lw,
		Layout:      ":minimize,maximize,close",
		Radius:      [4]float32{r, r, r * 0.5, r * 0.5},
		Shadow:      ShadowLayersReach(aeroWindowShadow(l, DecorationState{Active: true})),
	}
	// Aero Glass is the frame's: the desktop blurred behind the caption and
	// the borders, the client area opaque. Windows 7 Basic ("glass" 0) has
	// none.
	s.Glass = aeroColors(l).glass
	s.GlassFrame = s.Glass
	return s
}

// aeroWindowShadow is the shadow Windows 7's glass windows cast: a soft
// one under the active window, a tighter one otherwise.
func aeroWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.18), DY: l.S(3), Blur: l.S(12)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.3), DY: l.S(6), Blur: l.S(22)},
		{Color: shadowBlack(0.12), DY: 0, Blur: l.S(4)},
	}
}

func (aeroEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, aeroWindowShadow(l, st))
}

// aeroGlassAlpha is how opaque Aero's glass colour is over the blurred
// desktop: most of what is behind shows, as on Windows 7.
const aeroGlassAlpha = 0.55

func (aeroEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := aeroColors(l)
	lw := winPx(l)
	w := winSnap(f.Window)
	border := DecorationOf(l, st).Border
	grad := c.glassGrad
	if !st.Active {
		grad = c.glassOff
	}
	// real: the desktop is really blurred behind this frame now.
	real := c.glass && st.Active && GlassBehind(l)
	if real {
		tint := make([]paintengine2d.GradientStop, len(grad))
		for i, g := range grad {
			tint[i] = paintengine2d.GradientStop{Offset: g.Offset, Color: g.Color.WithAlpha(g.Color.A * aeroGlassAlpha)}
		}
		grad = tint
	}
	glassBox := w
	if st.Maximized {
		// The glass runs on under the screen's edge: its gradient keeps
		// the proportions of the caption.
		glassBox = paintengine2d.Rect{Min: w.Min, Max: paintengine2d.Pt(w.Max.X, f.Caption.Max.Y)}
	}
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		if real {
			// The compositor blurs the desktop behind the frame: take the
			// window's own background out from under it and lay the glass
			// down translucent, so the blur shows through.
			ctx.DrawRect(part, paintengine2d.Paint{Color: paintengine2d.White, Blend: paintengine2d.BlendDestOut})
		}
		ctx.DrawRect(part, VGradient(glassBox, grad...))
		if c.glass {
			sh := paintengine2d.XYWH(w.Min.X, w.Min.Y, w.Dx(), f.Caption.Max.Y-w.Min.Y)
			sh.Max.Y = sh.Min.Y + sh.Dy()*0.55
			ctx.DrawRect(sh, VGradient(sh, c.sheen...))
			capH := f.Caption.Max.Y - w.Min.Y
			streak := paintengine2d.NewPath()
			for _, fx := range [2]float32{0.18, 0.62} {
				x := w.Min.X + w.Dx()*fx
				sw := capH * 0.9
				streak.MoveTo(x, w.Min.Y)
				streak.LineTo(x+sw, w.Min.Y)
				streak.LineTo(x+sw-w.Dy()*0.35, w.Max.Y)
				streak.LineTo(x-w.Dy()*0.35, w.Max.Y)
				streak.Close()
			}
			ctx.DrawPath(streak, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.1)))
		}
		ctx.Restore()
	}
	if st.Maximized {
		return
	}
	drawFrameBorder(ctx, w, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, c.frameEdge)
	drawFrameBorder(ctx, w.Inset(lw), Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, paintengine2d.RGBA(1, 1, 1, 0.35))
	// The line round the client area, just outside it: the caption's bottom
	// row and the borders' inner edges.
	client := paintengine2d.Rect{Min: paintengine2d.Pt(w.Min.X+border.Left, f.Caption.Max.Y), Max: paintengine2d.Pt(w.Max.X-border.Right, w.Max.Y-border.Bottom)}
	if !f.Bar.Empty() {
		client.Min.Y = f.Bar.Max.Y
	}
	drawFrameBorder(ctx, client.Inset(-lw), Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, c.clientEdge)
}

func (aeroEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := aeroColors(l)
	f := l.body
	tb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(6), b.Dy())
	if tb.Dx() < l.S(8) {
		return
	}
	if c.glass {
		// The caption text sits on a blurred white blob so it reads on any
		// glass colour.
		tw := min(f.Advance(title), tb.Dx())
		h := f.Height()
		gr := paintengine2d.XYWH(tb.Min.X-l.S(2), tb.Min.Y+(tb.Dy()-h)*0.5+l.S(2), tw+l.S(4), h-l.S(4))
		ctx.Save()
		ctx.ClipRect(b)
		DropShadow(ctx, gr, gr.Dy()*0.5, c.capGlow, 0, 0, l.S(12), 0)
		ctx.Restore()
	}
	col := c.capText
	if !st.Active {
		col = c.capTextOff
	}
	l.drawFittedText(ctx, f, title, tb, col, AlignStart, 0)
}

func (aeroEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := aeroColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if k == CaptionClose {
		c.closeButton(l, ctx, b, WindowState{Active: st.Active, CloseHot: cs.Hovered(), ClosePress: cs.Pressed()})
		return
	}
	g := &aeroMinMax[0]
	switch {
	case !st.Active && !cs.Hovered():
		g = &aeroMinMax[3]
	case cs.Pressed():
		g = &aeroMinMax[2]
	case cs.Hovered():
		g = &aeroMinMax[1]
	}
	in := c.paintGlass(l, ctx, b, g, l.rx(3), false)
	if in.Empty() {
		return
	}
	fg := paintengine2d.RGB(1, 1, 1)
	edge := paintengine2d.RGBA(0.1, 0.15, 0.25, 0.8)
	if !st.Active && !cs.Hovered() {
		fg, edge = c.closeGlyphOff, paintengine2d.Color{}
	}
	side := snap(b.Dy() * 0.42)
	w := max(snap(b.Dy()*0.12), lw)
	if edge.A > 0 {
		// A dark rim round the white glyph, as Windows 7 drew them.
		for _, d := range []paintengine2d.Point{{X: -lw}, {X: lw}, {Y: -lw}, {Y: lw}} {
			DrawCaptionGlyph(ctx, b.Translate(d), k, st.Maximized, edge, side, w)
		}
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, fg, side, w)
}
