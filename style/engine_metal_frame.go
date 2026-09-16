package style

import "github.com/codemodify/paintengine2d"

// The Swing Metal window frame (see engine_metal.go): a stacked frame. Metal
// is the one Java look and feel that draws its own window decorations
// (JFrame.setDefaultLookAndFeelDecorated), and this is what MetalTitlePane
// and its root-pane border put on screen: a five-line border in primary 1 —
// secondary while the window is in the backdrop — grooved down the middle of
// each side but left solid for sixteen pixels at every corner; the title bar
// in primary 3, or Ocean's blue wash on the active window, with the bold
// title at its left; the bumps filling the bar from the title to the
// buttons; and the buttons flush squares in the bar's own colour, edged
// black, whose glyphs are Metal's icons. The frame icon at the left opens
// the window menu.
//
// Square and shadowless: Swing painted the whole frame into its own window,
// and nothing behind it showed through.
//
// Lengths and colours are engine_metal.go's, from the Java Look and Feel
// Design Guidelines' tables and from Java 5 and Ocean screenshots.

func mtlFrameBorder(l *Classic) Insets {
	fw := snap(mtlFrameW(l))
	return Insets{Top: fw, Right: fw, Bottom: fw, Left: fw}
}

func (metalEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := mtlColors(l)
	h := mtlCaptionH(l)
	side := snap(h - l.S(6))
	return DecorationSpec{
		Stacked: true,
		Border:  mtlFrameBorder(l),
		// The title bar and the frame line under it.
		Caption:   h + c.u,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(2)),
		ButtonPad: Insets{Top: snap((h - side) * 0.5), Right: snap(l.S(3)), Left: snap(l.S(3))},
		Layout:    "icon:minimize,maximize,close",
	}
}

func (metalEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := mtlColors(l)
	u := c.u
	b := mtlSnap(f.Window)
	border := mtlFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	frame, groove, grooveHi, bar := c.frameOn, c.grooveOn, c.grooveOnHi, c.titleOn
	if !st.Active {
		frame, groove, grooveHi, bar = c.frameOff, c.grooveOff, c.grooveOffHi, c.titleOff
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(frame))
	}
	if !st.Maximized {
		// The groove: a dark line two pixels in and a light one inside it,
		// broken for the solid corners.
		corner := snap(l.S(16))
		var gd, gl mtlRects
		x0, y0, x1, y1 := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
		if b.Dx() > 2*corner && b.Dy() > 2*corner {
			gd.add(x0+corner, y0+2*u, b.Dx()-2*corner, u)
			gl.add(x0+corner, y0+3*u, b.Dx()-2*corner, u)
			gd.add(x0+corner, y1-3*u, b.Dx()-2*corner, u)
			gl.add(x0+corner, y1-2*u, b.Dx()-2*corner, u)
			gd.add(x0+2*u, y0+corner, u, b.Dy()-2*corner)
			gl.add(x0+3*u, y0+corner, u, b.Dy()-2*corner)
			gd.add(x1-3*u, y0+corner, u, b.Dy()-2*corner)
			gl.add(x1-2*u, y0+corner, u, b.Dy()-2*corner)
		}
		gl.fill(ctx, grooveHi)
		gd.fill(ctx, groove)
	}
	tb := paintengine2d.XYWH(f.Caption.Min.X, f.Caption.Min.Y, f.Caption.Dx(), max(f.Caption.Dy()-u, 0))
	if c.ocean && st.Active {
		c.wash(ctx, tb, false)
	} else {
		ctx.DrawRect(tb, paintengine2d.Fill(bar))
	}
	// The frame's line under the title bar.
	ctx.DrawRect(paintengine2d.XYWH(tb.Min.X, tb.Max.Y, tb.Dx(), u), paintengine2d.Fill(frame))
}

// DrawCaptionTitle is the bold title at the bar's left with the bumps
// filling what is left of the bar.
func (metalEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := mtlColors(l)
	u := c.u
	tile := c.titleBumps
	if !st.Active {
		tile = c.titleOffBumps
	}
	f := c.bold2
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), max(b.Dy()-u, 0))
	tx := bar.Min.X + l.S(6)
	tw := min(f.Advance(title), max(bar.Max.X-tx, 0))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx, bar.Min.Y, bar.Max.X-tx, bar.Dy()), c.text, AlignStart, 0)
	bx := snap(tx + tw + l.S(6))
	if bar.Max.X-bx > 4*u && bar.Dy() > 6*u {
		c.bumps(ctx, paintengine2d.XYWH(bx, bar.Min.Y+3*u, bar.Max.X-bx, bar.Dy()-6*u), tile)
	}
}

func (metalEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := mtlColors(l)
	u := c.u
	b = mtlSnap(b)
	fill := c.titleOn
	if !st.Active {
		fill = c.titleOff
	}
	if cs.Pressed() {
		fill = c.pressed
	}
	if c.ocean && st.Active && !cs.Pressed() {
		c.wash(ctx, b, false)
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(fill))
	}
	edge := c.black
	if cs.Hovered() && !cs.Pressed() {
		edge = c.p1
	}
	c.flush(ctx, b, 1, edge, c.white, !cs.Pressed())
	g := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-u, b.Dy()-u)
	if k == CaptionClose {
		DrawCross(ctx, g.Inset(snap(b.Dx()*0.28)), c.black, 2*u)
		return
	}
	DrawCaptionGlyph(ctx, g, k, st.Maximized, c.black, snap(g.Dx()*0.45), max(u, 1))
}
