package style

import "github.com/codemodify/paintengine2d"

// The mwm window frame (see engine_motif.go), as the Motif window manager
// and CDE's dtwm drew it: a stacked frame whose 6px raised border is cut
// into resize handles a border-and-title-bar length from each corner, a
// sunken edge inside it, and a title bar of separately raised parts in the
// active or inactive colour set — the window menu button (a raised bar),
// the title box with the title centred (at the left in the packs that set
// it), minimize (a small raised square) and maximize (a large one). mwm had
// no close button (the window menu closes, or a double click on its
// button); where the button layout asks for one it is a raised part with a
// cross in the title's colour. A pressed part sinks.

func (motifEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	g := motifMwm(l, paintengine2d.XYWH(0, 0, 400, 300))
	return DecorationSpec{
		Stacked: true,
		Border:  Insets{Top: g.border, Right: g.border, Bottom: g.border, Left: g.border},
		Caption: g.bar,
		Button:  paintengine2d.Pt(g.bar, g.bar),
		Layout:  "icon:minimize,maximize",
	}
}

// motifFrameSet is the colour set a frame paints in.
func motifFrameSet(c *motifLook, st DecorationState) *mset {
	if st.Active {
		return &c.act
	}
	return &c.inact
}

func (motifEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := motifColors(l)
	s := motifFrameSet(c, st)
	g := motifMwm(l, paintengine2d.XYWH(0, 0, 400, 300))
	B, T := g.border, g.bar
	ib := mPx(l, 1)
	r := mSnap(f.Window)
	if st.Maximized {
		// The border is off the screen: the title bar's parts only.
		ctx.DrawRect(mSnap(f.Caption), paintengine2d.Fill(s.bg))
		return
	}
	for _, part := range FrameParts(f, Insets{Top: B, Right: B, Bottom: B, Left: B}) {
		ctx.DrawRect(part, paintengine2d.Fill(s.bg))
	}
	ext := c.px(l)
	if c.graded {
		mRings(ctx, r, s.hi1, s.lo1, s.hi2, s.lo2, ext)
	} else {
		mShadow(ctx, r, s.ts, s.bs, ext, false)
	}
	// Resize handle cuts, each handle bevelled on its own.
	cw := B + T
	lo, hi := paintengine2d.Fill(s.bs), paintengine2d.Fill(s.ts)
	if r.Dx() > 3*cw {
		for _, x := range [2]float32{r.Min.X + cw, r.Max.X - cw} {
			ctx.DrawRect(paintengine2d.XYWH(x-ib, r.Min.Y, ib, B), lo)
			ctx.DrawRect(paintengine2d.XYWH(x, r.Min.Y, ib, B), hi)
			ctx.DrawRect(paintengine2d.XYWH(x-ib, r.Max.Y-B, ib, B), lo)
			ctx.DrawRect(paintengine2d.XYWH(x, r.Max.Y-B, ib, B), hi)
		}
	}
	if r.Dy() > 3*cw {
		for _, y := range [2]float32{r.Min.Y + cw, r.Max.Y - cw} {
			ctx.DrawRect(paintengine2d.XYWH(r.Min.X, y-ib, B, ib), lo)
			ctx.DrawRect(paintengine2d.XYWH(r.Min.X, y, B, ib), hi)
			ctx.DrawRect(paintengine2d.XYWH(r.Max.X-B, y-ib, B, ib), lo)
			ctx.DrawRect(paintengine2d.XYWH(r.Max.X-B, y, B, ib), hi)
		}
	}
	// The sunken inner edge round the title bar and the client area.
	mShadow(ctx, r.Inset(B-ib), s.bs, s.ts, ib, false)
	ctx.DrawRect(mSnap(f.Caption), paintengine2d.Fill(s.bg))
	if c.outline {
		ctx.DrawRect(r.Inset(0.5), paintengine2d.StrokePaint(paintengine2d.RGB(0, 0, 0), 1))
	}
}

// DrawCaptionTitle is the raised title box between the buttons, the title
// in it.
func (motifEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := motifColors(l)
	s := motifFrameSet(c, st)
	ib := mPx(l, 1)
	box := mSnap(b)
	mShadow(ctx, box, s.ts, s.bs, ib, false)
	tb := box.Inset(ib + l.S(4))
	tb.Min.Y, tb.Max.Y = box.Min.Y, box.Max.Y
	align := AlignCenter
	if c.titleLeft {
		align = AlignStart
	}
	l.drawFittedText(ctx, l.body, title, tb, s.fg, align, 0)
}

func (motifEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := motifColors(l)
	s := motifFrameSet(c, st)
	ib := mPx(l, 1)
	p := mSnap(b)
	T := p.Dy()
	ctx.DrawRect(p, paintengine2d.Fill(s.bg))
	if cs.Pressed() {
		mShadow(ctx, p, s.bs, s.ts, ib, false)
	} else {
		mShadow(ctx, p, s.ts, s.bs, ib, false)
	}
	glyph := func(w, h float32) {
		w, h = snap(w), snap(h)
		gr := paintengine2d.XYWH(snap(p.Min.X+(p.Dx()-w)*0.5), snap(p.Min.Y+(p.Dy()-h)*0.5), w, h)
		mShadow(ctx, gr, s.ts, s.bs, ib, false)
	}
	switch k {
	case CaptionMenu:
		glyph(T*0.5, maxf(T*0.2, 3*ib))
	case CaptionMinimize:
		glyph(maxf(T*0.18, 3*ib), maxf(T*0.18, 3*ib))
	case CaptionMaximize:
		if st.Maximized {
			// Restore: the large square gives way to a middling one.
			glyph(T*0.36, T*0.36)
			return
		}
		glyph(T*0.52, T*0.52)
	case CaptionClose:
		DrawCross(ctx, p.Inset(T*0.3), s.fg, max(ib*1.5, l.S(1.5)))
	}
}
