package style

import "github.com/codemodify/paintengine2d"

// The NeXTSTEP / OPENSTEP and Window Maker window frame (see engine_next.go):
// a stacked frame, the shape the NeXT window server gave every window and
// Window Maker kept. A one-pixel black line round the window; the title bar
// inside it — the key window's dark bar, a light one in the backdrop, or the
// WM theme's FTitleBack / UTitleBack texture — with the bold title (centred,
// or as the theme's TitleJustify asks) and the buttons at its *ends*:
// miniaturize at the left, close at the right, which is the layout the packs
// ask for. NeXT's buttons are raised 15px squares three pixels in from the
// bar's ends, as NeXTSTEP 3.3 drew them at a 22px bar; Window Maker's
// new-style buttons are square tiles the bar's full height cut from its ends,
// each bevelled on its own. Along the bottom runs the notched resize bar, nine
// pixels tall, which is this frame's bottom border — the era's windows resized
// from it and from its two corner handles, not from a hairline.
//
// Nothing is rounded and nothing casts a shadow: the MegaPixel Display and
// X11 without a compositor drew a window's frame straight onto the screen.
// (Metrics from the NeXTSTEP User Interface Guidelines (Release 3) and from
// NeXTSTEP 3.3, OPENSTEP 4.2 and Window Maker screenshots read pixel by
// pixel, the same sources engine_next.go took its widgets from.)

// nxFrameBorder is the frame's border: the black line at the top and sides,
// the resize bar along the bottom.
func nxFrameBorder(l *Classic) Insets {
	u := nxU(l)
	return Insets{Top: u, Right: u, Bottom: nxResizeH(l), Left: u}
}

func (nextEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := nxColors(l)
	u := nxU(l)
	bar := nxTitleH(l)
	s := DecorationSpec{
		Stacked: true,
		Border:  nxFrameBorder(l),
		Caption: bar,
		// The title keeps to the bar's free space, aligned as the theme's
		// TitleJustify asks (DrawCaptionTitleSpan).
		Layout: "minimize:close",
	}
	if c.wm {
		// Window Maker: square tiles cut from the bar's ends.
		s.Button = paintengine2d.Pt(bar, bar)
		return s
	}
	// NeXT: 15px buttons at a 22px bar (the bar's height less seven pixels),
	// three pixels in from its ends and centred over its black under-line.
	side := snap(bar - 7*u)
	s.Button = paintengine2d.Pt(side, side)
	s.ButtonGap = 2 * u
	s.ButtonPad = Insets{Top: snap((bar - u - side) * 0.5), Right: 3 * u, Left: 3 * u}
	return s
}

func (e nextEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := nxColors(l)
	u := nxU(l)
	border := nxFrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	for _, part := range FrameParts(f, border) {
		nxFill(ctx, part, c.black)
	}
	tex := &c.ftitle
	if !st.Active {
		tex = &c.utitle
	}
	bar := nxSnap(f.Caption)
	tex.paint(ctx, bar)
	tex.raised(ctx, bar, u)
	if st.Maximized {
		return
	}
	// The resize bar fills the bottom border inside the frame's black line.
	w := nxSnap(f.Window)
	rh := nxResizeH(l)
	e.resizeBar(l, ctx, paintengine2d.XYWH(w.Min.X+u, w.Max.Y-rh, w.Dx()-2*u, rh-u))
}

func (e nextEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	e.DrawCaptionTitleSpan(l, ctx, b, b, title, st)
}

// DrawCaptionTitleSpan lays the bar out between the buttons as
// DrawWindowFrame does: Window Maker's middle section is a piece of its own,
// bevelled between the tiles — its highlight down the seam beside the
// miniaturize tile, its shadow beside the close tile — and the title is
// aligned in it (NeXT's in the room two pixels clear of its buttons) as the
// theme's TitleJustify asks.
func (nextEngine) DrawCaptionTitleSpan(l *Classic, ctx *paintengine2d.Context, free, bar paintengine2d.Rect, title string, st DecorationState) {
	c := nxColors(l)
	u := nxU(l)
	tex, txt := &c.ftitle, c.ftitleTxt
	if !st.Active {
		tex, txt = &c.utitle, c.utitleTxt
	}
	lead, trail := free.Min.X > bar.Min.X+0.5, free.Max.X < bar.Max.X-0.5
	left, right := free.Min.X, free.Max.X
	if c.wm {
		if lead || trail {
			tex.raised(ctx, nxSnap(paintengine2d.XYWH(free.Min.X, bar.Min.Y, free.Dx(), bar.Dy())), u)
		}
	} else {
		if lead {
			left += 2 * u
		}
		if trail {
			right -= 2 * u
		}
	}
	if title == "" {
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	tb := paintengine2d.XYWH(left+l.S(6), bar.Min.Y+u, right-left-l.S(12), bar.Dy()-3*u)
	l.drawFittedText(ctx, f, title, tb, txt, nxAlign(c.justify), 0)
}

func (e nextEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	if c.wm {
		e.wmFrameButton(l, ctx, b, k, cs, st)
		return
	}
	cst := StateNone
	if cs.Hovered() {
		cst |= StateHovered
	}
	if cs.Pressed() {
		cst |= StatePressed
	}
	ink := c.button(ctx, b, u, cst)
	g := b.Inset(snap(b.Dx() * 0.2))
	if cs.Pressed() {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	nxFrameGlyph(ctx, g, k, st.Maximized, u, c.cross*u, ink)
}

// wmFrameButton is a Window Maker title tile: the bar's texture (already
// painted under it) bevelled on its own, the glyph in the title colour, a
// light wash under the pointer; pushed, the tile turns white with a black
// outline and a black glyph.
func (e nextEngine) wmFrameButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := nxColors(l)
	u := nxU(l)
	glyph := c.ftitleTxt
	tex := &c.ftitle
	if !st.Active {
		glyph, tex = c.utitleTxt, &c.utitle
	}
	if cs.Pressed() {
		nxFill(ctx, b, paintengine2d.RGB(1, 1, 1))
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
		glyph = c.black
	} else {
		if cs.Hovered() {
			nxFill(ctx, b, c.hi.WithAlpha(0.18))
		}
		tex.raised(ctx, b, u)
	}
	side := snap(b.Dy() * 0.46)
	g := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-side)*0.5), snap(b.Min.Y+(b.Dy()-side)*0.5), side, side)
	if cs.Pressed() {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	nxFrameGlyph(ctx, g, k, st.Maximized, u, 1.6*u, glyph)
}

// nxFrameGlyph draws a caption glyph in g: the era's own miniaturize window
// and close cross (cw wide), and for the buttons NeXT's title bars never had
// — a desktop's layout asks for them — plainly in the same flat ink: zoom an
// outlined square (two of them while maximized), the window menu a stack of
// three bars.
func nxFrameGlyph(ctx *paintengine2d.Context, g paintengine2d.Rect, k CaptionButton, maximized bool, u, cw float32, col paintengine2d.Color) {
	switch k {
	case CaptionClose:
		nxCross(ctx, g.Inset(cw*0.35), col, cw)
	case CaptionMinimize:
		nxMini(ctx, g.Inset(u*0.5), u, col)
	case CaptionMaximize:
		nxZoom(ctx, g, u, col, maximized)
	case CaptionMenu:
		b := nxSnap(g)
		if b.Dy() < 5*u {
			return
		}
		for i := float32(0); i < 3; i++ {
			y := snap(b.Min.Y + i*(b.Dy()-u)*0.5)
			nxFill(ctx, paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), col)
		}
	}
}

// nxZoom is the zoom glyph: an outlined square, and while the window is
// maximized a second one behind it.
func nxZoom(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, col paintengine2d.Color, maximized bool) {
	b = nxSnap(b)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	box := func(r paintengine2d.Rect) {
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), u), col)
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Max.Y-u, r.Dx(), u), col)
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, u, r.Dy()), col)
		nxFill(ctx, paintengine2d.XYWH(r.Max.X-u, r.Min.Y, u, r.Dy()), col)
	}
	if !maximized {
		box(b)
		return
	}
	// Restore: the back square shows its top and right edges past the front
	// one.
	d := max(snap(b.Dx()*0.25), u)
	back := paintengine2d.XYWH(b.Min.X+d, b.Min.Y, b.Dx()-d, b.Dy()-d)
	front := paintengine2d.XYWH(b.Min.X, b.Min.Y+d, b.Dx()-d, b.Dy()-d)
	nxFill(ctx, paintengine2d.XYWH(back.Min.X, back.Min.Y, back.Dx(), u), col)
	nxFill(ctx, paintengine2d.XYWH(back.Max.X-u, back.Min.Y, u, back.Dy()), col)
	nxFill(ctx, paintengine2d.XYWH(back.Min.X, back.Min.Y, u, d), col)
	box(front)
}
