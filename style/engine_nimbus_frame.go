package style

import "github.com/codemodify/paintengine2d"

// The Nimbus window frame (see engine_nimbus.go), Java 6's own: a stacked
// frame, the shape Nimbus gives a frame it decorates itself. A rounded-top
// frame in the caption's grey-blue over a one-pixel edge, its title bar
// glossed from white, the bold title centred and dimmed while the window is
// in the backdrop, and round-cornered buttons at the right — Nimbus's grey
// gloss, pressed a shade darker — with the close button glowing red under
// the pointer and its cross turning white. Six pixels of frame run down the
// sides and along the bottom, with the body's gap line just inside.
//
// Nimbus floats its frames on a soft shadow, and that is the one the window
// gets: the same one engine_nimbus.go gives a dialog.
//
// Colours are engine_nimbus.go's, from the published Nimbus UIManager keys
// with the gradients measured off Java 6 screenshots.

func (nimbusEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := nbCaptionH(l)
	fw := snap(nbFrameW(l))
	side := snap(h - l.S(8))
	r := l.rx(6)
	dy, blur := nbShadow(l, PopupDialog)
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Right: fw, Bottom: fw, Left: fw},
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(2)),
		ButtonPad: Insets{
			Top:   snap((h - side) * 0.5),
			Right: snap(l.S(4)),
			Left:  snap(l.S(4)),
		},
		CenterTitle: true,
		Layout:      ":minimize,maximize,close",
		Radius:      [4]float32{r, r, 0, 0},
		Shadow:      ShadowReach(0, dy, blur, 0),
	}
}

func (nimbusEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := nbColors(l)
	u := c.u
	b := nbSnap(f.Window)
	capH := nbCaptionH(l)
	fw := snap(nbFrameW(l))
	if b.Dx() < 2*fw+capH || b.Dy() < capH+2*fw {
		return
	}
	border := Insets{Right: fw, Bottom: fw, Left: fw}
	if st.Maximized {
		border = Insets{}
	}
	r := nbRadius(b, l.rx(6))
	if st.Maximized {
		r = 0
	}
	grad, frame, edge, gap := c.capGrad, c.frameOn, c.frameEdgeOn, c.frameGapOn
	if !st.Active {
		grad, frame, edge, gap = c.capOffGrad, c.frameOff, c.frameEdgeOff, c.frameGapOff
	}
	in := b.Inset(u)
	ri := max(r-u, 0)
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), max(nbSnap(f.Caption).Max.Y-u-in.Min.Y, 0))
	body := paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+capH, b.Dx()-2*fw, b.Max.Y-capH-fw-b.Min.Y)
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		ctx.DrawPath(RoundRectPath(b, r, r, 0, 0), paintengine2d.Fill(edge))
		ctx.DrawPath(RoundRectPath(in, ri, ri, 0, 0), paintengine2d.Fill(frame))
		ctx.Save()
		ctx.ClipPath(RoundRectPath(in, ri, ri, 0, 0))
		ctx.DrawRect(bar, VGradient(bar, grad...))
		ctx.Restore()
		if !body.Empty() {
			ctx.DrawRect(body.Inset(-u), paintengine2d.Fill(gap))
		}
		ctx.Restore()
	}
}

func (nimbusEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := nbColors(l)
	fg := c.text
	if !st.Active {
		fg = c.dis
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	l.drawFittedText(ctx, f, title, b, fg, AlignCenter, 0)
}

func (nimbusEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := nbColors(l)
	u := c.u
	b = nbSnap(b)
	m := &c.grey
	switch {
	case cs.Pressed():
		m = &c.greyPress
	case cs.Hovered() && k == CaptionClose:
		m = &c.closeHotFace
	case cs.Hovered():
		m = &c.greyPress
	}
	// The gloss lays its shadow one pixel under the face, so the face keeps
	// that pixel of its box for it.
	gb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), max(b.Dy()-u, 0))
	c.gloss(ctx, gb, l.rx(3), m)
	glyph := c.arrowDark
	if k == CaptionClose && cs.Hovered() && !cs.Pressed() {
		glyph = c.white
	}
	if k == CaptionClose {
		DrawCross(ctx, gb.Inset(snap(gb.Dx()*0.3)), glyph, 1.8*u)
		return
	}
	DrawCaptionGlyph(ctx, gb, k, st.Maximized, glyph, snap(gb.Dx()*0.4), max(u, 1))
}
