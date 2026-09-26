package style

import "github.com/codemodify/paintengine2d"

// The KWin 2 "KDE Default" decoration (see engine_kde2.go), the one KDE 2
// set up on a fresh account — not "B II", which was one of the dozen
// alternatives it also shipped.
//
// It is a thin frame with three tells. The first is the **stipple**: the
// empty part of the title bar is woven with dots, a light one and a dark
// one a pixel down and right of it, repeating every three pixels across
// and every four down. The second is the flat fill: KDE 2's scheme set the
// title bar's blend colour to the title bar's own colour, so the bar has
// no gradient at all — only a lighter line along its very top. The third is
// where the buttons sit: the two at the left ride on the *title bar*
// colour and disappear into it, while the four at the right are drawn on
// the *button* colour and read as grey squares on the blue.
//
// Its lengths, scaled from KDE 2's 12px Helvetica to the toolkit's 16px: a
// 3px grab bar above a 16px title bar above a 1px line become 4, 21 and 1;
// the 4px side borders become 5; the 8px bottom grab bar becomes 10; and
// the 16px buttons become 21.
//
// KDE 2 ran on plain X with no compositor and nothing shaped: square
// corners, no shadow.
//
// Shapes and colours are engine_kde2.go's: documented metrics and colours
// measured on the KDE 2.0 and 2.2.2 screenshots named there.

// kw2Grab is the grab bar above the title bar, kw2Border the side borders
// and kw2Foot the grab bar along the bottom.
func kw2Grab(l *Classic) float32   { return max(snap(l.S(4)), 1) }
func kw2Border(l *Classic) float32 { return max(snap(l.S(5)), 2) }
func kw2Foot(l *Classic) float32   { return max(snap(l.S(10)), 3) }

// kw2CaptionH is the whole top decoration: the grab bar, the title bar and
// the line under it.
func kw2CaptionH(l *Classic) float32 {
	return kw2Grab(l) + kw2BarH(l) + kde3U(l)
}

// kw2BarH is the title bar itself.
func kw2BarH(l *Classic) float32 {
	return max(snap(l.body.Height()+l.S(5)), snap(l.S(21)))
}

// kw2Stipple weaves the dots over the bar: a light dot every three units
// across and four down, with a dark one a unit below and to the right.
func kw2Stipple(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, light, dark paintengine2d.Color) {
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	p := kde3Path()
	defer kde3Done(p)
	q := kde3Path()
	defer kde3Done(q)
	for y := b.Min.Y + 2*u; y+2*u <= b.Max.Y; y += 4 * u {
		for x := b.Min.X + u; x+2*u <= b.Max.X; x += 3 * u {
			p.AddRect(paintengine2d.XYWH(x, y, u, u))
			q.AddRect(paintengine2d.XYWH(x+u, y+u, u, u))
		}
	}
	ctx.DrawPath(p, paintengine2d.Fill(light))
	ctx.DrawPath(q, paintengine2d.Fill(dark))
}

// kw2Bar fills a title bar and returns the colour its label takes.
func kw2Bar(l *Classic, ctx *paintengine2d.Context, bar paintengine2d.Rect, active bool) paintengine2d.Color {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(bar)
	if r.Empty() {
		return c.capText
	}
	fill, top, fg := c.cap, c.capTop, c.capText
	dot, dark := c.capDot, c.capDotDark
	if !active {
		fill, top, fg = c.capOff, c.capOffTop, c.capOffText
		dot, dark = c.capOffDot, c.capOffDark
	}
	ctx.DrawRect(r, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), u), paintengine2d.Fill(top))
	if c.stipple {
		kw2Stipple(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y+u, r.Dx(), r.Dy()-u), u, dot, dark)
	}
	return fg
}

// kw2Button is a caption button: a square bevel over the given face with
// a black glyph, sinking when held.
func kw2Button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, maximized, hot, pressed bool, face paintengine2d.Color) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	if r.Dx() < 4*u || r.Dy() < 4*u {
		return
	}
	body := face
	if hot {
		body = Mix(face, paintengine2d.RGB(1, 1, 1), 0.2)
	}
	hi, lo := c.light, c.ring
	if pressed {
		hi, lo = lo, hi
	}
	ctx.DrawRect(r, paintengine2d.Fill(lo))
	ctx.DrawRect(paintengine2d.Rect{Min: r.Min, Max: paintengine2d.Pt(r.Max.X-u, r.Max.Y-u)}, paintengine2d.Fill(hi))
	in := r.Inset(u)
	if in.Dx() > 0 && in.Dy() > 0 {
		ctx.DrawRect(in, paintengine2d.Fill(body))
	}
	glyph := ReadableOn(body, 4.5, paintengine2d.RGB(0, 0, 0), paintengine2d.RGB(1, 1, 1))
	g := r
	if pressed {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	lw := max(l.S(1.5), u)
	if k == CaptionClose {
		DrawCross(ctx, g.Inset(g.Dx()*0.3), glyph, lw)
		return
	}
	DrawCaptionGlyph(ctx, g, k, maximized, glyph, g.Dx()*0.42, lw)
}

// ---- the desktop's frame ----------------------------------------------------------------

func (kde2Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := kw2CaptionH(l)
	bar := kw2BarH(l)
	side := snap(bar - kde3U(l)*2)
	return DecorationSpec{
		Stacked: true,
		Border: Insets{
			Top:    kw2Grab(l),
			Right:  kw2Border(l),
			Bottom: kw2Foot(l),
			Left:   kw2Border(l),
		},
		Caption:   h,
		Button:    paintengine2d.Pt(side, side),
		ButtonGap: max(snap(l.S(1)), 1),
		ButtonPad: Insets{Top: kw2Grab(l) + snap((bar-side)*0.5), Right: snap(l.S(2)), Left: snap(l.S(2))},
		Layout:    "icon:minimize,maximize,close",
	}
}

func (kde2Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := kde2Colors(l)
	u := kde3U(l)
	border := Insets{Top: kw2Grab(l), Right: kw2Border(l), Bottom: kw2Foot(l), Left: kw2Border(l)}
	if st.Maximized {
		border = Insets{}
	}
	b := kde3Snap(f.Window)
	for _, part := range FrameParts(f, border) {
		if part.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(part)
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		if !st.Maximized {
			kde3Frame(ctx, b, u, paintengine2d.Color{}, c.light, c.outline)
		}
		ctx.Restore()
	}
	cap := kde3Snap(f.Caption)
	ctx.DrawRect(cap, paintengine2d.Fill(c.bg))
	bar := paintengine2d.XYWH(cap.Min.X, cap.Min.Y+kw2Grab(l), cap.Dx(), kw2BarH(l))
	kw2Bar(l, ctx, bar, st.Active)
	// The line that closes the top decoration.
	ctx.DrawRect(paintengine2d.XYWH(cap.Min.X, bar.Max.Y, cap.Dx(), u), paintengine2d.Fill(c.outline))
}

// DrawCaptionTitle is the title at the left of the bar, over the stipple.
func (kde2Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := kde2Colors(l)
	fg := c.capText
	if !st.Active {
		fg = c.capOffText
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y+kw2Grab(l), b.Dx(), kw2BarH(l))
	pad := snap(l.S(4))
	tb := paintengine2d.XYWH(bar.Min.X+pad, bar.Min.Y, max(bar.Dx()-2*pad, 0), bar.Dy())
	if tb.Dx() <= 0 || title == "" {
		return
	}
	// The title stands on the bar, so it needs the stipple cleared behind
	// it — kwin drew the caption over a plain run of the bar's colour.
	fill := c.cap
	if !st.Active {
		fill = c.capOff
	}
	w := min(f.Advance(title)+pad, tb.Dx())
	ctx.DrawRect(kde3Snap(paintengine2d.XYWH(tb.Min.X-pad*0.5, bar.Min.Y+kde3U(l), w+pad, bar.Dy()-kde3U(l))), paintengine2d.Fill(fill))
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
}

func (kde2Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := kde2Colors(l)
	// The window-menu button rides on the title bar's colour; the rest are
	// grey squares in the button colour.
	face := c.btn
	if k == CaptionMenu {
		face = c.cap
		if !st.Active {
			face = c.capOff
		}
	}
	kw2Button(l, ctx, b, k, st.Maximized, cs.Hovered() && !cs.Pressed(), cs.Pressed(), face)
}

// ---- the in-app window ------------------------------------------------------------------

func (kde2Engine) WindowFrameInsets(l *Classic) Insets {
	return Insets{
		Top:    kw2CaptionH(l),
		Right:  kw2Border(l),
		Bottom: kw2Foot(l),
		Left:   kw2Border(l),
	}
}

// kw2CloseRect is the close button at the right of the in-app title bar.
func kw2CloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = kde3Snap(b)
	bar := kw2BarH(l)
	if b.Dy() < kw2CaptionH(l)+kw2Foot(l) || b.Dx() < kw2Border(l)*2+bar*3 {
		return paintengine2d.Rect{}
	}
	side := snap(bar - kde3U(l)*2)
	return paintengine2d.XYWH(
		snap(b.Max.X-kw2Border(l)-l.S(2)-side),
		snap(b.Min.Y+kw2Grab(l)+(bar-side)*0.5), side, side)
}

func (kde2Engine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return kw2CloseRect(l, b)
}

// DrawWindowFrame is the same decoration round an in-app window or dialog.
func (kde2Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := kde2Colors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	grab, side, bar := kw2Grab(l), kw2Border(l), kw2BarH(l)
	if b.Dx() < side*2+bar*2 || b.Dy() < kw2CaptionH(l)+kw2Foot(l) {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	kde3Frame(ctx, b, u, paintengine2d.Color{}, c.light, c.outline)
	barR := paintengine2d.XYWH(b.Min.X+side, b.Min.Y+grab, b.Dx()-2*side, bar)
	fg := kw2Bar(l, ctx, barR, st.Active)
	ctx.DrawRect(paintengine2d.XYWH(barR.Min.X, barR.Max.Y, barR.Dx(), u), paintengine2d.Fill(c.outline))
	cb := kw2CloseRect(l, b)
	f := l.bold
	if f == nil {
		f = l.body
	}
	// The caption keeps the close button's slot whether or not the window
	// can be closed, so the two frames differ only inside that box.
	right := barR.Max.X
	if !cb.Empty() {
		right = cb.Min.X - snap(l.S(2))
	}
	pad := snap(l.S(4))
	if tb := paintengine2d.XYWH(barR.Min.X+pad, barR.Min.Y, right-barR.Min.X-2*pad, barR.Dy()); tb.Dx() > 0 && title != "" {
		w := min(f.Advance(title)+pad, tb.Dx())
		fill := c.cap
		if !st.Active {
			fill = c.capOff
		}
		ctx.DrawRect(kde3Snap(paintengine2d.XYWH(tb.Min.X-pad*0.5, barR.Min.Y+u, w+pad, barR.Dy()-u)), paintengine2d.Fill(fill))
		l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
	}
	if st.CanClose && !cb.Empty() {
		kw2Button(l, ctx, cb, CaptionClose, false, st.CloseHot && !st.ClosePress, st.ClosePress, c.btn)
	}
}
