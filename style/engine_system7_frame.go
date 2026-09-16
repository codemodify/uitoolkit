package style

import "github.com/codemodify/paintengine2d"

// The Macintosh document-window frame (see engine_system7.go), System 1
// through System 7: a stacked frame drawn in whole cells like the rest of
// the engine. A one-pixel black line round the window; inside it the title
// bar — six horizontal stripes a pixel apart, the 11 × 11 close box nine
// pixels in from the left, the zoom box the same distance from the right,
// and the bold title centred in a margin cut out of the stripes — and a
// black line under it. System 7's colour chrome lightens the bar, greys the
// stripes and bevels bar and boxes in lavender and navy.
//
// The Mac's own layout is the one the packs ask for: close at the left, zoom
// at the right. Windows were square and cast no shadow (the desktop drew
// only the hard one-pixel shadow of engine_system7.go's PopupShadow, which
// windows on the toolkit's frame get from the desktop instead).
//
// Two departures, both because the toolkit's caption buttons stay live where
// the Mac's simply were not: a backdrop window keeps plain outlined boxes
// (the Mac drew none at all, though it does lose its stripes), and a
// desktop's layout may ask for buttons the Mac never had, which are drawn as
// boxes with the plainest glyph that fits — a bar for minimize, a small
// window for the window menu.
//
// Proportions are engine_system7.go's, which read them off System 6 and
// System 7 screenshots at 1:1: an 11-pixel box, nine pixels in, six stripes
// two pixels apart.

// s7FrameBox is the frame's caption boxes in cells: the box itself and the
// one-cell margin round it that breaks the stripes.
func s7FrameBox(l *Classic) (n, pad int) {
	_, _, n = s7closeBox(l)
	return n, 1
}

func (system7Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	bh := s7barH(l)
	n, pad := s7FrameBox(l)
	x, y, _ := s7closeBox(l)
	side := float32(n+2*pad) * u
	return DecorationSpec{
		Stacked: true,
		Border:  Insets{Top: u, Right: u, Bottom: u, Left: u},
		// The bar and the black line under it.
		Caption: float32(bh+1) * u,
		Button:  paintengine2d.Pt(side, side),
		// The boxes carry their own margin, so they need no gap of their own.
		ButtonPad: Insets{
			Top:   float32(y-1-pad) * u,
			Right: float32(x-1-pad) * u,
			Left:  float32(x-1-pad) * u,
		},
		CenterTitle: true,
		Layout:      "close:maximize",
	}
}

// s7FrameBar is the colour of the title bar's face and of the ink its stripes
// are drawn in, in state st.
func s7FrameBar(c *s7, st DecorationState) (face, stripe paintengine2d.Color) {
	if c.grey && st.Active {
		return c.bar, c.stripe
	}
	return c.win, c.black
}

func (system7Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := s7colors(l)
	u := rpU(l)
	bh := s7barH(l)
	border := Insets{Top: u, Right: u, Bottom: u, Left: u}
	if st.Maximized {
		border = Insets{}
	}
	face, stripe := s7FrameBar(c, st)
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.win))
	}
	g := rpGridAt(ctx, f.Caption, u)
	if g.w < 4 || g.h < 2 {
		return
	}
	bar := g.sub(0, 0, g.w, min(bh, g.h-1))
	line := c.black
	if c.grey && !st.Active {
		line = c.offLine
	}
	var frame, lav, lavDk, ink rpInk
	if c.grey && st.Active {
		ctx.DrawRect(bar.rect(), paintengine2d.Fill(face))
		rpEdge(&lav, &lavDk, bar, 0, 0, bar.w, bar.h, 1)
	}
	if st.Active {
		// Six stripes, one cell apart, level with the boxes; the title and
		// the boxes cut their own margins out of them as they paint.
		_, y, n := s7closeBox(l)
		for i := 0; i < 6; i++ {
			if row := y - 1 + 2*i; row < bar.h {
				ink.cells(bar, 1, row, bar.w-2, 1)
			}
		}
		_ = n
	}
	// The black line under the bar, and the window's own frame line.
	frame.cells(g, 0, bar.h, g.w, 1)
	if !st.Maximized {
		drawFrameBorder(ctx, f.Window, border, line)
	}
	lavDk.fill(ctx, c.lavDk)
	lav.fill(ctx, c.lav)
	ink.fill(ctx, stripe)
	frame.fill(ctx, line)
}

func (system7Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := s7colors(l)
	u := rpU(l)
	face, _ := s7FrameBar(c, st)
	g := rpGridAt(ctx, b, u)
	if g.w < 4 || g.h < 2 {
		return
	}
	// The title's own margin, cut out of the stripes.
	bar := g.sub(0, 0, g.w, g.h-1)
	ctx.DrawRect(bar.rect(), paintengine2d.Fill(face))
	col := c.text
	if c.grey && !st.Active {
		col = c.offTitle
	}
	l.drawFittedText(ctx, l.BoldFont(), title, bar.rect(), col, AlignCenter, 0)
}

func (system7Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := s7colors(l)
	u := rpU(l)
	n, pad := s7FrameBox(l)
	face, _ := s7FrameBar(c, st)
	g := rpGridAt(ctx, b, u)
	if g.w < n || g.h < n {
		return
	}
	// The box's margin breaks the stripes round it.
	ctx.DrawRect(g.rect(), paintengine2d.Fill(face))
	box := g.sub(pad, pad, n, n)
	pressed := cs.Pressed()
	var black, lav, navy rpInk
	if c.grey && st.Active {
		// Two rings, navy then lavender, round the box's face.
		bg := c.boxFace
		if pressed {
			bg = c.navy
		}
		ctx.DrawRect(box.rect(), paintengine2d.Fill(bg))
		rpEdge(&navy, &lav, box, 0, 0, n, n, 1)
		if !pressed {
			rpEdge(&lav, &navy, box, 1, 1, n-2, n-2, 1)
		}
	} else {
		bg := c.white
		if pressed {
			bg = c.black
		}
		ctx.DrawRect(box.rect(), paintengine2d.Fill(bg))
		black.frame(box, 0, 0, n, n, 1)
	}
	ink := &black
	if c.grey {
		ink = &navy
		if pressed {
			ink = &lav
		}
	} else if pressed {
		ink = &lav // the white burst on the inverted box
	}
	s7FrameGlyph(ink, box, k, n, st, pressed)
	navy.fill(ctx, c.navy)
	if c.grey {
		lav.fill(ctx, c.lav)
	} else {
		lav.fill(ctx, c.white)
	}
	black.fill(ctx, c.black)
}

// s7FrameGlyph inks the box's glyph: the zoom box's small square, the burst
// of a pressed close box, and for the buttons the Mac had no box for, a bar
// (minimize) and a small window (the window menu).
func s7FrameGlyph(ink *rpInk, box rpGrid, k CaptionButton, n int, st DecorationState, pressed bool) {
	switch k {
	case CaptionClose:
		if !pressed {
			return
		}
		// The pressed close box bursts: lines out from its centre.
		m := rpNewMask(n, n)
		mid := n / 2
		m.line(2, mid, n-3, mid)
		m.line(mid, 2, mid, n-3)
		m.line(2, 2, n-3, n-3)
		m.line(n-3, 2, 2, n-3)
		m.emit(ink, box, 0, 0)
	case CaptionMaximize:
		s := n * 7 / 11
		ink.frame(box, 0, 0, s, s, 1)
		if st.Maximized {
			// Already zoomed: the small square is filled in.
			ink.cells(box, 1, 1, s-2, s-2)
		}
	case CaptionMinimize:
		ink.cells(box, 2, n/2, n-4, 1)
	case CaptionMenu:
		ink.frame(box, 2, 2, n-4, n-4, 1)
		ink.cells(box, 2, 2, n-4, 2)
	}
}
