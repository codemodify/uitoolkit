package style

import "github.com/codemodify/paintengine2d"

// The Windows XP window frame (see engine_luna.go): a stacked frame. The
// caption is the scheme's vertical gradient (Luna's blue, olive or silver)
// across the whole window, darkened at its ends, the title in bold white
// (black on Silver) with its drop shadow at the left; the sizing frame's
// five lines run down the sides and along the bottom. Caption buttons are
// 21px glossy rounded squares at a 29px caption's scale, 5px from its top
// and 2px apart: minimize and maximize in a lighter shade of the caption
// inside a white ring with white glyphs, close in red. Hot buttons brighten,
// pressed ones darken; a backdrop window's buttons pale with the caption.
// Frames are square for now (XP's rounded top corners need translucency).

func (lunaEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := lunaCaptionH(l)
	fw := lunaFrameW(l)
	side := snap(h * 21 / 29)
	right := snap(h*4/29) + lunaPx(l) - fw
	r := l.rx(7)
	if st.Maximized || st.Tiled != 0 {
		r = 0
	}
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Right: fw, Bottom: fw, Left: fw},
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: snap(h * 2 / 29),
		ButtonPad: Insets{Top: snap(h * 5 / 29), Right: max(right, 0), Left: max(right, 0)},
		Layout:    "icon:minimize,maximize,close",
		Radius:    [4]float32{r, r, 0, 0},
	}
}

func (lunaEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := lunaColors(l)
	w := lunaSnap(f.Window)
	lw := lunaPx(l)
	side, bottom, capGrad := c.frame, c.bottom, c.caption
	if !st.Active {
		side, bottom, capGrad = c.frameOff, c.bottomOff, c.captionOff
	}
	bar := paintengine2d.Rect{Min: w.Min, Max: paintengine2d.Pt(w.Max.X, f.Caption.Max.Y)}
	ctx.DrawRect(bar, VGradient(bar, capGrad...))
	if st.Maximized {
		return
	}
	// The caption darkens toward its ends, like the bitmap.
	for i, a := range [2]float32{0.85, 0.35} {
		o := float32(i) * lw
		edge := paintengine2d.Fill(side[i].WithAlpha(a))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X+o, bar.Min.Y, lw, bar.Dy()), edge)
		ctx.DrawRect(paintengine2d.XYWH(bar.Max.X-o-lw, bar.Min.Y, lw, bar.Dy()), edge)
	}
	// The sizing frame: nested lines, outer to inner, below the caption.
	body := paintengine2d.Rect{Min: paintengine2d.Pt(w.Min.X, bar.Max.Y), Max: w.Max}
	for i := 0; i < 5; i++ {
		o := float32(i) * lw
		ctx.DrawRect(paintengine2d.XYWH(body.Min.X+o, body.Min.Y, lw, body.Dy()-o), paintengine2d.Fill(side[i]))
		ctx.DrawRect(paintengine2d.XYWH(body.Max.X-o-lw, body.Min.Y, lw, body.Dy()-o), paintengine2d.Fill(side[i]))
		ctx.DrawRect(paintengine2d.XYWH(body.Min.X+o, body.Max.Y-o-lw, body.Dx()-2*o, lw), paintengine2d.Fill(bottom[i]))
	}
}

func (lunaEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := lunaColors(l)
	text, shadow := c.capText, c.capShadow
	if !st.Active {
		text, shadow = c.capOff, c.capOffSh
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	lw := lunaPx(l)
	tb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y+lw, b.Dx()-l.S(6), b.Dy()-lw)
	if shadow.A > 0 {
		l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(lw, lw)), shadow, AlignStart, 0)
	}
	l.drawFittedText(ctx, f, title, tb, text, AlignStart, 0)
}

func (lunaEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := lunaColors(l)
	b = lunaSnap(b)
	if k == CaptionClose {
		c.closeButton(l, ctx, b, WindowState{Active: st.Active, CloseHot: cs.Hovered(), ClosePress: cs.Pressed()})
		return
	}
	lw := lunaPx(l)
	grad := c.caption
	if !st.Active {
		grad = c.captionOff
	}
	mid := lunaStopAt(grad, 0.5)
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	lift := float32(0)
	switch {
	case cs.Pressed():
		lift = -0.22
	case cs.Hovered():
		lift = 0.18
	}
	shade := func(t float32) paintengine2d.Color {
		if t < 0 {
			return Mix(mid, black, -t)
		}
		return Mix(mid, white, t)
	}
	face := []paintengine2d.GradientStop{Stop(0, shade(0.42+lift)), Stop(0.45, shade(0.2+lift)), Stop(1, shade(0.05+lift))}
	ring := white.WithAlpha(0.9)
	if Luma(mid) > 0.5 {
		ring = Mix(mid, black, 0.45)
	}
	if !st.Active {
		ring = ring.WithAlpha(0.6)
	}
	r := l.rx(3)
	in, ri := lunaFrame(ctx, b, r, lw, ring)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, face...))
	if st.Active && !cs.Pressed() {
		// The gloss in the upper left.
		gl := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx()*0.7, in.Dy()*0.6)
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRoundRect(gl, gl.Dy()*0.5, gl.Dy()*0.5, paintengine2d.Radial(paintengine2d.RadialGradient{Center: gl.Min, Radius: gl.Dx(), Stops: lunaGloss}))
		ctx.Restore()
	}
	fg := ReadableOn(shade(0.2+lift), 3, white, black)
	lunaGlyph(ctx, b, k, st.Maximized, fg, lw)
}

// lunaStopAt is the colour of gradient stops at offset t.
func lunaStopAt(stops []paintengine2d.GradientStop, t float32) paintengine2d.Color {
	if len(stops) == 0 {
		return paintengine2d.Color{}
	}
	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		if t <= b.Offset {
			if b.Offset <= a.Offset {
				return b.Color
			}
			return Mix(a.Color, b.Color, (t-a.Offset)/(b.Offset-a.Offset))
		}
	}
	return stops[len(stops)-1].Color
}

// lunaGlyph draws XP's caption glyphs, 21px buttons' proportions scaled to
// b: a thick bar low at the left (minimize), a square with a heavy top
// (maximize), two overlapping ones (restore), a window (the window menu).
func lunaGlyph(ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, maximized bool, col paintengine2d.Color, lw float32) {
	u := b.Dx() / 21
	px := func(v float32) float32 { return snap(v * u) }
	t := max(px(3), 2*lw)
	thin := max(px(2), lw)
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	fill := paintengine2d.Fill(col)
	box := func(x, y, w, h float32) {
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(x, y, w, t))
		p.AddRect(paintengine2d.XYWH(x, y+h-thin, w, thin))
		p.AddRect(paintengine2d.XYWH(x, y+t, thin, h-t-thin))
		p.AddRect(paintengine2d.XYWH(x+w-thin, y+t, thin, h-t-thin))
		ctx.DrawPath(p, fill)
	}
	switch k {
	case CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(x0+px(5), y0+px(13), px(7), t), fill)
	case CaptionMaximize:
		if !maximized {
			box(x0+px(5), y0+px(5), px(11), px(11))
			return
		}
		p := paintengine2d.NewPath()
		back := paintengine2d.XYWH(x0+px(8), y0+px(4), px(9), px(9))
		p.AddRect(paintengine2d.XYWH(back.Min.X, back.Min.Y, back.Dx(), t))
		p.AddRect(paintengine2d.XYWH(back.Max.X-thin, back.Min.Y+t, thin, back.Dy()-t))
		ctx.DrawPath(p, fill)
		box(x0+px(4), y0+px(8), px(9), px(9))
	case CaptionMenu:
		box(x0+px(5), y0+px(6), px(11), px(9))
	case CaptionClose:
		lunaCross(ctx, b.Inset(snap(b.Dx()*0.27)), col, max(b.Dx()*0.13, lw*1.5))
	}
}
