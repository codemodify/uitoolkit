package style

import "github.com/codemodify/paintengine2d"

// The Keramik window decoration (see engine_keramik.go): a stacked frame,
// KDE 3.1's default. The window is a rounded slab — a dark outline round the
// caption colour, its top corners rounded seven pixels — whose title bar is
// a gel band in the caption colour; the bold title rides in a lighter
// "bubble", a pill from the left of the bar as wide as the title; the
// buttons at the right are small rounded gels, the close one lighting up
// under the pointer. A three-pixel border runs down the sides and along the
// bottom, with the client edge drawn dark above the body.
//
// KDE 3 had no compositor of its own, so the corners were shaped with the X
// Shape extension and nothing cast a shadow: rounded top, square bottom, no
// shadow. The buttons sit where kwin put them by default, which is the
// layout the pack asks for: the window menu at the left, minimize, maximize
// and close at the right.
//
// One departure: Keramik's bubble rose a few pixels *above* the title bar,
// into the desktop. A uitoolkit window is a rectangle whose every pixel is
// the window's, so the bubble sits inside the bar instead.
//
// Shapes and colours are engine_keramik.go's, written from the look's
// visible behaviour and from colours measured on KDE 3.1 screenshots.

// kmFrameBorder is the decoration's border: three pixels at the sides and,
// at the bottom, the client edge as well.
func kmFrameBorder(l *Classic) Insets {
	fr := snap(l.S(3))
	return Insets{Right: fr, Bottom: fr + kde3U(l), Left: fr}
}

// kmFrameButton is a caption button's side.
func kmFrameButton(l *Classic) float32 {
	h := kmCaptionH(l)
	side := snap(h - l.S(10))
	if side < snap(l.S(12)) {
		side = snap(h * 0.6)
	}
	return side
}

func (keramikEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := kmCaptionH(l)
	side := kmFrameButton(l)
	fr := kmFrameBorder(l)
	r := l.rx(7)
	return DecorationSpec{
		Stacked:   true,
		Border:    fr,
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(l.S(4)),
		// The buttons keep seven pixels from the window's edge, the border
		// included.
		ButtonPad: Insets{Top: snap((h - side) * 0.5), Right: max(snap(l.S(7))-fr.Right, 0), Left: max(snap(l.S(7))-fr.Left, 0)},
		Layout:    "icon:minimize,maximize,close",
		Radius:    [4]float32{r, r, 0, 0},
	}
}

func (keramikEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := keramikColors(l)
	u := kde3U(l)
	border := kmFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	b := kde3Snap(f.Window)
	// The bar runs the window's whole width: the side border starts under it.
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), kde3Snap(f.Caption).Max.Y-b.Min.Y)
	base, stops, line := c.capBase, c.cap, c.capLine
	if !st.Active {
		base, stops, line = c.capOffBase, c.capOff, c.capOffLine
	}
	r := l.rx(7)
	if st.Maximized {
		r = 0
	}
	ri := max(r-u, 0)
	body := border.Apply(paintengine2d.XYWH(b.Min.X, bar.Max.Y, b.Dx(), b.Max.Y-bar.Max.Y))
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		kde3Rounded(ctx, b, r, true, true, false, true, paintengine2d.Fill(line))
		kde3Rounded(ctx, b.Inset(u), ri, true, true, false, true, paintengine2d.Fill(base))
		kde3Rounded(ctx, paintengine2d.XYWH(bar.Min.X+u, bar.Min.Y+u, bar.Dx()-2*u, bar.Dy()-u), ri, true, true, false, true,
			VGradient(bar, stops...))
		// The client edge: a dark line above and left of the body.
		ctx.DrawRect(paintengine2d.XYWH(body.Min.X-u, bar.Max.Y, b.Max.X-border.Right-body.Min.X+u, u), paintengine2d.Fill(line))
		ctx.Restore()
	}
}

// DrawCaptionTitle is the title bubble: a pill in the bar, a touch lighter
// than it, with the bold title inside.
func (keramikEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := keramikColors(l)
	u := kde3U(l)
	base, bubble := c.capBase, c.bubble
	fg, shadow := c.capText, c.capShadow
	if !st.Active {
		base, bubble = c.capOffBase, c.bubbleOff
		fg, shadow = c.capOffText, c.capOffShadow
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	bw := min(f.Advance(title)+l.S(28), b.Dx()-l.S(2))
	if bw < l.S(20) {
		return
	}
	bb := kde3Snap(paintengine2d.XYWH(b.Min.X+l.S(5), b.Min.Y+u, bw, b.Dy()-l.S(3)-u))
	br := bb.Dy() * 0.5
	if l.square() {
		br = 0
	}
	ctx.DrawRoundRect(bb, br, br, paintengine2d.Fill(Mix(base, paintengine2d.RGB(1, 1, 1), 0.7)))
	bi := bb.Inset(u)
	ctx.DrawRoundRect(bi, max(br-u, 0), max(br-u, 0), VGradient(bi, bubble...))
	tb := paintengine2d.XYWH(bi.Min.X+l.S(12), bi.Min.Y, bi.Dx()-l.S(16), bi.Dy())
	if shadow.A > 0 {
		l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(u, u)), shadow, AlignStart, 0)
	}
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
}

func (keramikEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	r := min(l.rx(3), b.Dx()*0.3)
	face := c.closeIdle
	switch {
	case cs.Pressed():
		face = c.closePress
	case cs.Hovered():
		face = c.closeHot
	}
	line := c.capLine
	if !st.Active {
		line = c.capOffLine
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(line.WithAlpha(0.9)))
	in := b.Inset(u)
	ctx.DrawRoundRect(in, max(r-u, 0), max(r-u, 0), VGradient(in, face...))
	g := in
	if cs.Pressed() {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	DrawCaptionGlyph(ctx, g, k, st.Maximized, c.btnGlyph, in.Dx()*0.46, max(l.S(1.8), u))
}
