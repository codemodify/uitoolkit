package style

import "github.com/codemodify/paintengine2d"

// The Qt Fusion window frame (see engine_fusion.go): a stacked frame. Fusion
// is a widget style, not a window manager, so the frame it draws for its own
// windows is the one QStyle draws for an MDI sub-window — the only window
// frame Qt itself has ever painted: the title bar a vertical gradient in the
// highlight colour (the window colour while the window is in the backdrop)
// with chamfered top corners, a light line just inside its top edge, the
// bold title centred over a soft shadow, and small bevelled boxes at the
// right that wash lighter under the pointer and darken when pressed. A
// two-pixel border in the frame line, lit down its left and shaded down its
// right, runs round the body.
//
// Fusion arrived with Qt 5 in 2012, on desktops that composite, so the frame
// keeps the chamfer and drops a modest shadow.
//
// Shapes and colours are engine_fusion.go's, each following from a Qt-style
// colour-role calculation checked against screenshots.

func fuFrameBorder(l *Classic) Insets {
	u := max(snap(fuU(l)), 1)
	return Insets{Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

func (fusionEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := fuTitleH(l)
	side := snap(h - l.S(7))
	fr := fuFrameBorder(l)
	ch := l.rx(4)
	return DecorationSpec{
		Stacked:   true,
		Border:    fr,
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(2)),
		ButtonPad: Insets{
			Top:   snap((h - side) * 0.5),
			Right: max(snap(l.S(5))-fr.Right, 0),
			Left:  max(snap(l.S(5))-fr.Left, 0),
		},
		CenterTitle: true,
		Layout:      ":minimize,maximize,close",
		Radius:      [4]float32{ch, ch, 0, 0},
		Shadow:      ShadowLayersReach(fuWindowShadow(l, DecorationState{Active: true})),
	}
}

// fuWindowShadow is the shadow a composited desktop drops under a Fusion
// window: Qt states none of its own, so this is the modest one the looks of
// its era shared.
func fuWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.16), DY: l.S(2), Blur: l.S(10)}}
	}
	return []WindowShadow{{Color: shadowBlack(0.3), DY: l.S(6), Blur: l.S(24)}}
}

func (fusionEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, fuWindowShadow(l, st))
}

func (fusionEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := fusionColors(l)
	u := max(snap(fuU(l)), 1)
	b := fuSnap(f.Window)
	if b.Dx() < 8*u || b.Dy() < fuTitleH(l) {
		return
	}
	border := fuFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), fuSnap(f.Caption).Max.Y-b.Min.Y)
	body := paintengine2d.XYWH(b.Min.X, bar.Max.Y, b.Dx(), b.Max.Y-bar.Max.Y)
	stops, line, hi := c.title, c.titleLine, c.titleHi
	if !st.Active {
		stops, line, hi = c.titleOff, c.titleLineOff, c.titleHiOff
	}
	ch := l.rx(4)
	if st.Maximized {
		ch = 0
	}
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		if !body.Empty() {
			ctx.DrawRect(body, paintengine2d.Fill(c.win))
			ctx.DrawRect(body.Inset(u*0.5), paintengine2d.StrokePaint(c.frameLine, u))
			fuVLine(ctx, body.Min.X+u, body.Min.Y, body.Max.Y-u, u, c.light)
			fuHLine(ctx, body.Min.X+u, body.Max.X-u, body.Max.Y-2*u, u, c.menuShadow)
			fuVLine(ctx, body.Max.X-2*u, body.Min.Y, body.Max.Y-u, u, c.menuShadow)
		}
		ctx.DrawPath(RoundRectPath(bar, ch, ch, 0, 0), VGradient(bar, stops...))
		ctx.Save()
		ctx.ClipRect(bar)
		ctx.DrawRoundRectCorners(bar.Inset(u*0.5), ch, ch, 0, 0, paintengine2d.StrokePaint(line, u))
		ctx.Restore()
		fuHLine(ctx, bar.Min.X+6*u, bar.Max.X-6*u, bar.Min.Y+u, u, hi)
		ctx.Restore()
	}
}

func (fusionEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := fusionColors(l)
	u := max(snap(fuU(l)), 1)
	f := l.bold
	if f == nil {
		f = l.body
	}
	if !st.Active {
		l.drawFittedText(ctx, f, title, b, c.titleTextOff, AlignCenter, 0)
		return
	}
	l.drawFittedText(ctx, f, title, b.Translate(paintengine2d.Pt(u, u)), c.titleShadowText.WithAlpha(0.45), AlignCenter, 0)
	l.drawFittedText(ctx, f, title, b, c.titleText, AlignCenter, 0)
}

func (fusionEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := fusionColors(l)
	u := max(snap(fuU(l)), 1)
	b = fuSnap(b)
	switch {
	case cs.Pressed():
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(darkerPct(c.hl, 120)))
	case cs.Hovered():
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 20.0/255)))
	}
	border := c.mdiLine
	if !st.Active {
		border = c.mdiLineOff
	}
	r := fuR(l, 2)
	fuStroke(ctx, b, r, u, border)
	hi := paintengine2d.RGBA(1, 1, 1, 60.0/255)
	if cs.Pressed() {
		hi = darkerPct(c.hl, 130)
	}
	fuHLine(ctx, b.Min.X+2*u, b.Max.X-2*u, b.Min.Y+u, u, hi)
	fuVLine(ctx, b.Min.X+u, b.Min.Y+2*u, b.Max.Y-2*u, u, hi)
	glyph := c.titleText
	if !st.Active {
		glyph = c.text
	}
	if k == CaptionClose {
		DrawCross(ctx, b.Inset(b.Dx()*0.3), glyph, l.S(1.5))
		return
	}
	DrawCaptionGlyph(ctx, b, k, st.Maximized, glyph, snap(b.Dx()*0.4), max(u, 1))
}
