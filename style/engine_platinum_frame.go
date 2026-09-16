package style

import "github.com/codemodify/paintengine2d"

// The Mac OS 8 / 9 "Platinum" window frame (see engine_platinum.go): a
// stacked frame. A black one-pixel outline round the window with the
// Platinum bevel just inside it — white along the top and left, shadow along
// the bottom and right — and a second black line round the content; the
// title bar in #cccccc with the bold title centred, the close box eleven
// pixels square seven pixels in from the left, the collapse and zoom boxes
// the same distance in from the right (zoom outermost, five pixels apart),
// and the raised ridges filling the bar each side of the title.
//
// That is the Mac OS 8 Human Interface Guidelines' document window, and the
// layout the packs ask for: close at the left, collapse and zoom at the
// right — the collapse box, which rolled the window up into its title bar,
// standing in for minimize. Square and shadowless, as the Appearance
// Manager drew it.
//
// A backdrop window keeps flat boxes here, where Mac OS 8 drew none at all
// and only the dimmed title: the toolkit's caption buttons stay live and
// hoverable, so they have to be visible. The buttons the Mac had no box for
// (the window menu) get the plainest glyph that fits.
//
// Metrics are engine_platinum.go's, from the Guidelines' figures and Mac
// OS 8.0 / 9.0 screenshots at 1:1.

// platFrameU is Platinum's one-pixel line on whole device pixels, so the
// frame's outline, bevel and boxes stay crisp at any scale.
func platFrameU(l *Classic) float32 { return max(snap(platU(l)), 1) }

// platFrameBorder is the frame's border: the black outline, the bevel and
// the black line round the content.
func platFrameBorder(l *Classic) Insets {
	u := platFrameU(l)
	return Insets{Top: 2 * u, Right: 3 * u, Bottom: 3 * u, Left: 3 * u}
}

func (platinumEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := platFrameU(l)
	bar := platBarH(l)
	s := snap(l.S(11))
	return DecorationSpec{
		Stacked: true,
		Border:  platFrameBorder(l),
		// The bar, the gap under it and the content's black line.
		Caption: bar + 3*u,
		// A box's widget holds the etch Platinum rings it with.
		Button:    paintengine2d.Pt(s+2*u, s+2*u),
		ButtonGap: max(snap(l.S(5))-2*u, u),
		ButtonPad: Insets{
			Top:   max(snap((bar-s)*0.5)-u, 0),
			Right: max(snap(l.S(7))-u, 0),
			Left:  max(snap(l.S(7))-u, 0),
		},
		Layout: "close:minimize,maximize",
	}
}

func (platinumEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := platColors(l)
	u := platFrameU(l)
	border := platFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.title))
	}
	w := f.Window
	if !st.Maximized {
		ctx.DrawRect(w.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
		Edge(ctx, w.Inset(u), c.hi, Shade(c.title, -0.25))
		// The content's own black line, down the sides and along the bottom;
		// the caption closes it off at the top.
		in := w.Inset(2 * u)
		y := f.Caption.Max.Y - u
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, y, u, in.Max.Y-y), paintengine2d.Fill(c.black))
		ctx.DrawRect(paintengine2d.XYWH(in.Max.X-u, y, u, in.Max.Y-y), paintengine2d.Fill(c.black))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-u, in.Dx(), u), paintengine2d.Fill(c.black))
	}
	ctx.DrawRect(paintengine2d.XYWH(f.Caption.Min.X, f.Caption.Max.Y-u, f.Caption.Dx(), u), paintengine2d.Fill(c.black))
}

// DrawCaptionTitle is the bold title with the bar's ridges running out from
// it to the buttons on either side, as Platinum filled the free bar.
func (platinumEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := platColors(l)
	u := platFrameU(l)
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), max(b.Dy()-3*u, 0))
	f := l.BoldFont()
	tw := min(f.Advance(title), bar.Dx())
	tx := snap(bar.Min.X + (bar.Dx()-tw)*0.5)
	col := c.text
	if !st.Active {
		col = c.dim
	}
	if st.Active {
		// Ridges only on the active window, as on the Mac.
		gap := l.S(8)
		cy := bar.Min.Y + bar.Dy()*0.5
		c.ridges(ctx, bar.Min.X+l.S(4), tx-gap, cy, u, 6)
		c.ridges(ctx, tx+tw+gap, bar.Max.X-l.S(4), cy, u, 6)
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx, bar.Min.Y, tw, bar.Dy()), col, AlignStart, 0)
}

func (platinumEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := platColors(l)
	u := platFrameU(l)
	box := b.Inset(u)
	if box.Dx() < 3*u || box.Dy() < 3*u {
		return
	}
	if !st.Active && !cs.Hovered() && !cs.Pressed() {
		// A backdrop window's boxes are flat: the outline and the bar's face.
		ctx.DrawRect(box, paintengine2d.Fill(c.black))
		ctx.DrawRect(box.Inset(u), paintengine2d.Fill(c.title))
	} else {
		c.winBox(ctx, box, u, cs.Pressed())
	}
	ink := c.black
	if cs.Pressed() {
		ink = Shade(c.title, -0.1)
	}
	in := box.Inset(u)
	switch k {
	case CaptionClose:
		// The close box carries no glyph at all.
	case CaptionMaximize:
		g := paintengine2d.XYWH(in.Min.X, in.Min.Y, snap(in.Dx()*0.55), snap(in.Dy()*0.55))
		ctx.DrawRect(g.Inset(u*0.5), paintengine2d.StrokePaint(ink, u))
		if st.Maximized {
			// Already zoomed: the small square is filled in.
			ctx.DrawRect(g.Inset(u), paintengine2d.Fill(ink))
		}
	case CaptionMinimize:
		// The collapse box: the two lines that rolled a window up.
		mid := snap(in.Min.Y + in.Dy()*0.5)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, mid-u, in.Dx(), u), paintengine2d.Fill(ink))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, mid+u, in.Dx(), u), paintengine2d.Fill(ink))
	case CaptionMenu:
		ctx.DrawRect(in.Inset(u*0.5), paintengine2d.StrokePaint(ink, u))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), 2*u), paintengine2d.Fill(ink))
	}
}
