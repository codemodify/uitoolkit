package style

import "github.com/codemodify/paintengine2d"

// The kwm window frame (see engine_kde1.go), as KDE 1's window manager drew
// it and as Windows did not: a stacked frame whose four-pixel border is
// raised grey all the way round, with no caption bar across it. What holds
// the title is a *strip* sunk into that grey — a dark line above it and a
// light one below — which fills only the space the buttons leave, and whose
// colour runs across its width: #000080 into black on the focused window,
// #808080 into the window grey on the rest, with the bold title at the left
// over the deep end. The buttons stand outside the strip on the bare grey,
// flat line-art with no box under them, and raise a bevel only under the
// pointer so a uitoolkit window still says what is clickable.
//
// kwm's own metrics, scaled from KDE 1's 12px Helvetica to the toolkit's
// 16px: a 4px border becomes 5, a 20px title row 26, and the 2px
// separation that makes an 18px strip becomes 3.
//
// KDE 1 ran on plain X with no compositor and nothing shaped: square
// corners, no shadow.

// kw1Border is the raised grey border round the window.
func kw1Border(l *Classic) float32 { return max(snap(l.S(5)), 2) }

// kw1CaptionH is the title row: the strip plus kwm's separation above and
// below it.
func kw1CaptionH(l *Classic) float32 {
	return max(snap(l.body.Height()+l.S(8)), snap(l.S(26)))
}

// kw1Sep is the gap kwm leaves between the strip and the frame.
func kw1Sep(l *Classic) float32 { return max(snap(l.S(3)), 1) }

// kw1Strip sinks the title strip into the grey of b and fills it with the
// blend, returning the box the title is written in and its colour. An
// empty box means there was no room for a strip.
func kw1Strip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, active bool) (paintengine2d.Rect, paintengine2d.Color) {
	c := w95colors(l)
	sep := kw1Sep(l)
	from, to, fg := c.cap1, c.cap2, c.capTxt
	if !active {
		from, to, fg = c.capOff1, c.capOff2, c.capOffTxt
	}
	r := paintengine2d.XYWH(snap(b.Min.X), snap(b.Min.Y+sep), snap(b.Dx()), snap(b.Dy()-2*sep))
	if r.Dx() < 4 || r.Dy() < 4 {
		return paintengine2d.Rect{}, fg
	}
	// Sunk: the dark line above and left, the light one below and right.
	c.sunken(ctx, r)
	in := r.Inset(max(snap(l.S(1)), 1))
	if in.Dx() <= 0 || in.Dy() <= 0 {
		return paintengine2d.Rect{}, fg
	}
	if from == to {
		ctx.DrawRect(in, paintengine2d.Fill(from))
	} else {
		ctx.DrawRect(in, HGradient(in, Stop(0, from), Stop(1, to)))
	}
	return in, fg
}

// kw1Title writes the bold title at the left of the strip.
func kw1Title(l *Classic, ctx *paintengine2d.Context, in paintengine2d.Rect, title string, fg paintengine2d.Color) {
	if in.Empty() || title == "" {
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	pad := snap(l.S(5))
	tb := paintengine2d.XYWH(in.Min.X+pad, in.Min.Y, in.Dx()-pad*2, in.Dy())
	if tb.Dx() <= 0 {
		return
	}
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
}

// kw1Glyph paints one caption glyph on the bare frame: flat at rest, on a
// raised bevel under the pointer and a pushed one while held.
func kw1Glyph(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, maximized, hot, pressed bool) {
	c := w95colors(l)
	r := paintengine2d.XYWH(snap(b.Min.X), snap(b.Min.Y), snap(b.Dx()), snap(b.Dy()))
	if hot || pressed {
		ctx.DrawRect(r, paintengine2d.Fill(c.face))
		if pressed {
			c.pushed(ctx, r)
		} else {
			c.raised(ctx, r)
		}
	}
	g := r
	if pressed {
		g = g.Translate(paintengine2d.Pt(1, 1))
	}
	// kwm's glyphs: a dot for iconify, an outlined square for maximize, a
	// thin cross for close, the application's box for the window menu.
	lw := max(snap(l.S(1.5)), 1)
	switch k {
	case CaptionMinimize:
		d := max(snap(g.Dx()*0.2), 2)
		ctx.DrawRect(paintengine2d.XYWH(snap(g.Min.X+(g.Dx()-d)*0.5), snap(g.Min.Y+(g.Dy()-d)*0.5), d, d), paintengine2d.Fill(c.text))
	case CaptionClose:
		DrawCross(ctx, g.Inset(g.Dx()*0.28), c.text, lw)
	default:
		DrawCaptionGlyph(ctx, g, k, maximized, c.text, g.Dx()*0.44, lw)
	}
}

// ---- the desktop's frame ----------------------------------------------------------------

func (kde1Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	fr := kw1Border(l)
	h := kw1CaptionH(l)
	side := snap(h - kw1Sep(l)*2)
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Top: fr, Right: fr, Bottom: fr, Left: fr},
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: max(snap(l.S(1)), 1),
		CloseGap:  snap(l.S(3)),
		ButtonPad: Insets{Top: snap((h - side) * 0.5), Right: snap(l.S(2)), Left: snap(l.S(2))},
		Layout:    "icon:minimize,maximize,close",
	}
}

func (kde1Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := w95colors(l)
	fr := kw1Border(l)
	if st.Maximized {
		fr = 0
	}
	border := Insets{Top: fr, Right: fr, Bottom: fr, Left: fr}
	for _, part := range FrameParts(f, border) {
		if !part.Empty() {
			ctx.DrawRect(part, paintengine2d.Fill(c.face))
		}
	}
	ctx.DrawRect(f.Caption, paintengine2d.Fill(c.face))
	if fr > 0 {
		c.raisedWindow(ctx, f.Window)
	}
}

// DrawCaptionTitle is the sunken strip and the bold title on it: the strip
// fills whatever the buttons leave, which is where kwm put it.
func (kde1Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	in, fg := kw1Strip(l, ctx, b, st.Active)
	kw1Title(l, ctx, in, title, fg)
}

func (kde1Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	kw1Glyph(l, ctx, b, k, st.Maximized, cs.Hovered() && !cs.Pressed(), cs.Pressed())
}

// ---- the in-app window ------------------------------------------------------------------

func (kde1Engine) WindowFrameInsets(l *Classic) Insets {
	fr := kw1Border(l)
	return Insets{Top: fr + kw1CaptionH(l), Right: fr, Bottom: fr, Left: fr}
}

// kw1CloseRect is the close glyph at the right of the in-app title row.
func kw1CloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	fr, h := kw1Border(l), kw1CaptionH(l)
	if b.Dy() < fr*2+h || b.Dx() < fr*2+h*3 {
		return paintengine2d.Rect{}
	}
	side := snap(h - kw1Sep(l)*2)
	return paintengine2d.XYWH(snap(b.Max.X-fr-l.S(2)-side), snap(b.Min.Y+fr+(h-side)*0.5), side, side)
}

func (kde1Engine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return kw1CloseRect(l, b)
}

// DrawWindowFrame is the same kwm frame round an in-app window or dialog.
func (kde1Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := w95colors(l)
	fr, h := kw1Border(l), kw1CaptionH(l)
	if b.Dx() < fr*2+h*2 || b.Dy() < fr*2+h {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	c.raisedWindow(ctx, b)
	// The strip keeps the close button's slot whether or not the window
	// can be closed, so the two frames differ only inside that box.
	cb := kw1CloseRect(l, b)
	right := b.Max.X - fr
	if !cb.Empty() {
		right = cb.Min.X - snap(l.S(2))
	}
	bar := paintengine2d.XYWH(b.Min.X+fr, b.Min.Y+fr, right-(b.Min.X+fr), h)
	if bar.Dx() > 0 {
		in, fg := kw1Strip(l, ctx, bar, st.Active)
		kw1Title(l, ctx, in, title, fg)
	}
	if st.CanClose && !cb.Empty() {
		kw1Glyph(l, ctx, cb, CaptionClose, false, st.CloseHot && !st.ClosePress, st.ClosePress)
	}
}
