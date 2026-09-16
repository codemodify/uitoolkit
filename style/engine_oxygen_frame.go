package style

import "github.com/codemodify/paintengine2d"

// The Oxygen window decoration (see engine_oxygen.go), KDE 4's default: a
// stacked frame whose whole point is that there is no title bar. The
// window's own gradient — the vertical ramp over the top 300 pixels and the
// elliptical light on the top edge — runs on through the caption, so the
// title simply stands, centred and embossed, on the window itself; the
// float frame's light rim traces the edge, the corners are rounded five
// pixels and a four-pixel border runs down the sides and along the bottom.
// The buttons at the right are Oxygen's round slabs, 21 pixels; the close
// button's cross glows #bf0303 under the pointer.
//
// KDE 4 came with a compositor, so this is the first of the KDE frames with
// a real shadow: Oxygen's is wide and soft, and fainter behind a window in
// the backdrop. The buttons sit where kwin put them by default, which is the
// layout the pack asks for.
//
// Shapes and colours are engine_oxygen.go's, from Oxygen's published colour
// values measured against screenshots; the shadow's spread is measured off
// KDE 4 screenshots, which is all Oxygen's own shadow settings amount to.

func oxFrameBorder(l *Classic) Insets {
	k := oxK(l)
	return Insets{Top: 2 * k, Right: 4 * k, Bottom: 4 * k, Left: 4 * k}
}

func (oxygenEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := oxTitleH(l)
	side := min(snap(l.S(21)), h)
	fr := oxFrameBorder(l)
	r := l.rx(5)
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Top: snap(fr.Top), Right: snap(fr.Right), Bottom: snap(fr.Bottom), Left: snap(fr.Left)},
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(3)),
		ButtonPad: Insets{
			Top:   snap((h - side) * 0.5),
			Right: max(snap(l.S(6))-snap(fr.Right), 0),
			Left:  max(snap(l.S(6))-snap(fr.Left), 0),
		},
		CenterButtons: true,
		CenterTitle:   true,
		Layout:        "icon:minimize,maximize,close",
		Radius:        [4]float32{r, r, 0, 0},
		Shadow:        ShadowLayersReach(oxWindowShadow(l, DecorationState{Active: true})),
	}
}

// oxWindowShadow is Oxygen's drop shadow: wide and soft under the active
// window, a shallow one behind the others.
func oxWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.17), DY: l.S(3), Blur: l.S(14)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.32), DY: l.S(10), Blur: l.S(32)},
		{Color: shadowBlack(0.13), DY: l.S(1), Blur: l.S(4)},
	}
}

func (oxygenEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, oxWindowShadow(l, st))
}

// DrawDecoration paints the window's own gradient on through the caption and
// the border — the window background painted it already, so only the parts
// the frame owns are touched up — and traces the float frame's rim.
func (e oxygenEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := oxygenColors(l)
	k := oxK(l)
	b := fuSnap(f.Window)
	if b.Dx() < 12*k || b.Dy() < oxTitleH(l) {
		return
	}
	r := l.rx(5)
	if st.Maximized {
		r = 0
	}
	border := oxFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		ctx.ClipRoundRect(b, r, r)
		e.fillGradient(l, ctx, b, b.Dy())
		ctx.Restore()
	}
	if st.Maximized {
		return
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), f.Caption.Max.Y-b.Min.Y))
	c.popupFrame(l, ctx, b)
	ctx.Restore()
	// The rim down the sides and along the bottom, inside the border.
	for _, part := range FrameParts(f, border)[1:] {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		c.popupFrame(l, ctx, b)
		ctx.Restore()
	}
}

// DrawCaptionTitle is the embossed, centred title standing on the window
// itself: a light copy a pixel below it, then the title.
func (oxygenEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := oxygenColors(l)
	k := oxK(l)
	col := c.text
	if !st.Active {
		col = Mix(c.text, c.win, 0.45)
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	l.drawFittedText(ctx, f, title, b.Translate(paintengine2d.Pt(0, k)), c.winLight.WithAlpha(0.7), AlignCenter, 0)
	l.drawFittedText(ctx, f, title, b, col, AlignCenter, 0)
}

func (e oxygenEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := oxygenColors(l)
	u := oxK(l)
	bst := StateNone
	if cs.Pressed() {
		bst |= StatePressed
	}
	e.RadioIndicator(l, ctx, b, bst, false)
	glyph := c.btnText
	if k == CaptionClose {
		switch {
		case cs.Pressed():
			glyph = Mix(c.btnText, Hex("#bf0303"), 0.8)
		case cs.Hovered():
			glyph = Hex("#bf0303")
		}
	} else if cs.Hovered() || cs.Pressed() {
		glyph = Mix(c.btnText, c.text, 0.5)
	}
	if !st.Active {
		glyph = glyph.WithAlpha(glyph.A * 0.6)
	}
	g := b.Inset(b.Dx() * 0.34)
	if k == CaptionClose {
		// The engraved cross: a light copy a pixel below, then the glyph.
		DrawCross(ctx, g.Translate(paintengine2d.Pt(0, u)), c.winLight, 1.6*u)
		DrawCross(ctx, g, glyph, 1.6*u)
		return
	}
	DrawCaptionGlyph(ctx, b.Translate(paintengine2d.Pt(0, u)), k, st.Maximized, c.winLight, b.Dx()*0.34, max(l.S(1.4), 1))
	DrawCaptionGlyph(ctx, b, k, st.Maximized, glyph, b.Dx()*0.34, max(l.S(1.4), 1))
}
