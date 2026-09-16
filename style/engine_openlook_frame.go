package style

import "github.com/codemodify/paintengine2d"

// The OPEN LOOK window frame (see engine_openlook.go), as olwm drew a base
// window: a stacked frame. A two-pixel black outline round a five-pixel
// border of the window colour, with the L-shaped resize corners cut into it
// at all four corners; inside it the header, whose bold title is centred on
// the window and which is recessed while the window has the input focus.
//
// OPEN LOOK put no close, iconify or zoom button on the frame at all: the
// header carries one control, the window menu button at its left — the
// abbreviated menu button, a small raised box with cut corners and the
// engraved menu mark — and every window operation is on the menu it opens.
// So "icon:" is the layout the pack asks for. Where a desktop's layout asks
// for more, the extra buttons are the same box with the plainest glyph that
// fits.
//
// Square and shadowless: olwm drew the frame straight onto the root window.
// Sizes are engine_openlook.go's, from the OPEN LOOK Graphical User
// Interface Application Style Guidelines' figures and OpenWindows 3
// screenshots.

// olFrameButton is a header button's box in cells: OPEN LOOK's abbreviated
// menu button, 19 cells at most, centred in the header.
func olFrameButton(l *Classic) (w, h int) {
	h = min(olHeaderH(l)-4, 19)
	if h < 6 {
		h = 6
	}
	return h + 1, h
}

func (openlookEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	hh := olHeaderH(l)
	bw, bh := olFrameButton(l)
	return DecorationSpec{
		Stacked: true,
		Border:  Insets{Top: olFrame * u, Right: olFrame * u, Bottom: olFrame * u, Left: olFrame * u},
		// The header and the border's row under it.
		Caption:   float32(hh+olFrame) * u,
		Button:    paintengine2d.Pt(float32(bw)*u, float32(bh)*u),
		ButtonGap: 4 * u,
		ButtonPad: Insets{
			Top:   float32((hh-bh)/2) * u,
			Right: 9 * u,
			Left:  9 * u,
		},
		CenterTitle: true,
		Layout:      "icon:",
	}
}

func (openlookEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := olColors(l)
	u := rpU(l)
	hh := olHeaderH(l)
	border := Insets{Top: olFrame * u, Right: olFrame * u, Bottom: olFrame * u, Left: olFrame * u}
	if st.Maximized {
		border = Insets{}
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.bg1))
	}
	g := rpGridAt(ctx, f.Window, u)
	var black rpInk
	if !st.Maximized && g.w >= 2*olFrame+8 && g.h >= 2*olFrame+hh+2 {
		black.frame(g, 0, 0, g.w, g.h, 2)
		// The resize corners: L-shapes of 11 cells with 5-cell arms.
		if g.w > 40 && g.h > 40 {
			for _, cr := range [4][2]int{{0, 0}, {g.w - 11, 0}, {0, g.h - 11}, {g.w - 11, g.h - 11}} {
				m := rpNewMask(11, 11)
				if cr[1] == 0 {
					m.rect(0, 0, 11, 5)
				} else {
					m.rect(0, 6, 11, 5)
				}
				if cr[0] == 0 {
					m.rect(0, 0, 5, 11)
				} else {
					m.rect(6, 0, 5, 11)
				}
				var face rpInk
				m.emit(&face, g, cr[0], cr[1])
				face.fill(ctx, c.bg1)
				o := m.outline()
				o.emit(&black, g, cr[0], cr[1])
			}
		}
	}
	if st.Active {
		// The header of the window with the input focus is recessed.
		hg := rpGridAt(ctx, f.Caption, u)
		c.box(ctx, hg, 0, 0, hg.w, min(hh, hg.h), true, c.bg2)
	}
	black.fill(ctx, c.black)
}

func (openlookEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := olColors(l)
	u := rpU(l)
	col := c.text
	if !st.Active {
		col = c.dim
	}
	g := rpGridAt(ctx, b, u)
	if g.w < 4 {
		return
	}
	l.drawFittedText(ctx, l.BoldFont(), title, g.at(0, 0, g.w, min(olHeaderH(l), g.h)), col, AlignCenter, 0)
}

func (openlookEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := olColors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	bw, bh := min(g.w, g.h+1), min(g.h, g.w-1)
	if bw < 6 || bh < 6 {
		return
	}
	pressed := cs.Pressed()
	// The abbreviated menu button: a raised box with cut corners.
	s := rpShape{w: bw, h: bh, cut: 1}
	fill := c.bg1
	if pressed {
		fill = c.bg2
	}
	var face, hiK, loK, ink rpInk
	s.fill(&face, g, 0, 0)
	face.fill(ctx, fill)
	for row := 0; row < bh; row++ {
		a1, b1, a2, b2, ok := s.frameRuns(row)
		if !ok {
			continue
		}
		switch {
		case row == 0:
			hiK.cells(g, a1, row, b1-a1, 1)
		case row == bh-1:
			loK.cells(g, a1, row, b1-a1, 1)
		default:
			hiK.cells(g, a1, row, b1-a1, 1)
			loK.cells(g, a2, row, b2-a2, 1)
		}
	}
	if pressed {
		hiK, loK = loK, hiK
	}
	hiK.fill(ctx, c.hi)
	loK.fill(ctx, c.bg3)
	if k == CaptionMenu {
		c.mark(ctx, g, bw/2, bh/2, min(9, bh-2), DirDown, pressed)
		return
	}
	// The buttons OPEN LOOK kept on the window menu: the plainest glyph in
	// the header's ink.
	n := max(min(bw, bh)-6, 3)
	x0, y0 := (bw-n)/2, (bh-n)/2
	switch k {
	case CaptionMinimize:
		ink.cells(g, x0, bh/2, n, 1)
	case CaptionMaximize:
		ink.frame(g, x0, y0, n, n, 1)
		if st.Maximized && n > 4 {
			ink.frame(g, x0+2, y0+2, n-4, n-4, 1)
		}
	case CaptionClose:
		m := rpNewMask(bw, bh)
		m.line(x0, y0, x0+n-1, y0+n-1)
		m.line(x0+n-1, y0, x0, y0+n-1)
		m.emit(&ink, g, 0, 0)
	}
	ink.fill(ctx, c.black)
}
