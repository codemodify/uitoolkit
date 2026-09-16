package style

import "github.com/codemodify/paintengine2d"

// The Intuition window frame (see engine_amiga.go): a stacked frame, the
// shape Workbench gave every window, drawn in whole Amiga rows.
//
// Workbench 3.1 (and 2.0): a thin raised frame round the window and a
// recessed one round the body, the borders in FILLPEN blue while the window
// is active and grey when it is not; the title bar reads a shine row, the
// black title and a shadow row. The gadgets are raised boxes flush with the
// bar's ends — the 20-pixel close gadget at the left, the zoom gadget (a box
// in a box) and the depth gadget (two overlapping boxes), 24 pixels each, at
// the right.
//
// Workbench 1.3: white borders on the blue, two pixels at the sides and a
// row at the bottom; a white title bar with the blue title, the close gadget
// (a blue box round a black dot) at the left, the two depth gadgets at the
// right, each set off by a blue line, and the drag bar's two blue stripes
// filling the bar between them — dotted with white while the window is
// inactive, as Intuition ghosted it.
//
// The Amiga's own layout is the one the packs ask for: close at the left,
// zoom and depth at the right. Nothing was rounded and nothing cast a
// shadow. Where a desktop's layout asks for buttons Intuition had no gadget
// for, minimize takes the depth gadget's glyph (the Amiga's own "send this
// window behind") and the window menu a plain box.
//
// Geometry and glyphs are engine_amiga.go's, from Intuition's documented
// border widths and gadget sizes and from Workbench screenshots at 1:1.

// amFrameGeom is the frame's title bar in cells: its rows, the bar's own
// height (1.3 keeps the last row for the border's blue line) and the gadget
// widths (close, the others).
func amFrameGeom(l *Classic) (rows, bar, closeW, gadgetW int) {
	c := amColors(l)
	rows = amBarRows(l)
	bar = rows * amRow
	if c.wb13 {
		// 1.3's gadgets carry the blue line that sets them apart: half of it
		// on each side, so two neighbours stand two pixels apart.
		return rows, bar - amRow, 24, 24
	}
	return rows, bar, 20, 24
}

func (amigaEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := amColors(l)
	u := rpU(l)
	rows, bar, closeW, gadgetW := amFrameGeom(l)
	s := DecorationSpec{
		Stacked: true,
		Caption: float32(rows*amRow) * u,
		Button:  paintengine2d.Pt(float32(gadgetW)*u, float32(bar)*u),
		Layout:  "close:maximize",
	}
	if closeW != gadgetW {
		s.CloseButton = paintengine2d.Pt(float32(closeW)*u, float32(bar)*u)
	}
	if c.wb13 {
		s.Border = Insets{Right: am13Side * u, Bottom: amRow * u, Left: am13Side * u}
		return s
	}
	s.Border = Insets{Right: 4 * u, Bottom: 2 * amRow * u, Left: 4 * u}
	return s
}

func (e amigaEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := amColors(l)
	u := rpU(l)
	rows, bar, _, _ := amFrameGeom(l)
	border := DecorationOf(l, st).Border
	fill := c.fill
	if !st.Active {
		fill = c.win
	}
	if c.wb13 {
		fill = c.win
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(fill))
	}
	g := rpGridAt(ctx, f.Window, u)
	if g.w < 8 || g.h < rows*amRow {
		return
	}
	var hi, lo, white rpInk
	if c.wb13 {
		// The white bar over the blue, and the white borders under it.
		white.cells(g, 0, 0, g.w, bar)
		if !st.Maximized {
			white.cells(g, 0, bar+amRow, am13Side, g.h-bar-amRow)
			white.cells(g, g.w-am13Side, bar+amRow, am13Side, g.h-bar-amRow)
			white.cells(g, am13Side, g.h-amRow, g.w-2*am13Side, amRow)
		}
		white.fill(ctx, c.line)
		return
	}
	if st.Maximized {
		return
	}
	// 3.1: the raised frame round the window and the recessed one round the
	// body, whose top edge closes the title bar off.
	amBevel(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	amBevel(&lo, &hi, g, 3, rows*amRow-amRow, g.w-6, g.h-rows*amRow, 1)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
}

// DrawCaptionTitle is the title at the bar's left, with 1.3's drag-bar
// stripes running out from it to the gadgets.
func (e amigaEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := amColors(l)
	u := rpU(l)
	_, bar, _, _ := amFrameGeom(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < amRow {
		return
	}
	bar = min(bar, g.h)
	col, tx := c.text, 6
	if c.wb13 {
		col, tx = c.barText, 4
	}
	f := l.MonoFont()
	tw := min(int(f.Advance(title)/u+0.5)+1, g.w-tx-4)
	if tw <= 0 {
		return
	}
	if c.wb13 {
		// Rows 2–3 and 6–7 of the ten-row bar, kept central in a taller one.
		sx0, sx1 := tx+tw+6, g.w-4
		sy := max(0, bar/amRow-10) / 2 * amRow
		if sx1-sx0 >= 4 {
			var blue rpInk
			var stripes [2]paintengine2d.Rect
			for i := range stripes {
				stripes[i] = g.at(sx0, sy+(2+4*i)*amRow, sx1-sx0, 2*amRow)
				blue.add(stripes[i])
			}
			blue.fill(ctx, c.win)
			if !st.Active {
				for _, r := range stripes {
					rpPatFill(ctx, r, amGhost, c.line, u)
				}
			}
		}
	}
	tb := g.at(tx, 0, tw+2, bar)
	amText(l, ctx, title, tb, tb.Intersect(b), col, AlignStart)
}

func (e amigaEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := amColors(l)
	u := rpU(l)
	rows, _, _, _ := amFrameGeom(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 8 || g.h < 4 {
		return
	}
	if c.wb13 {
		e.frame13Gadget(l, ctx, g, k, cs, st)
		return
	}
	var hi, lo, grey, blue rpInk
	if cs.Pressed() {
		amBevel(&lo, &hi, g, 0, 0, g.w, g.h, 1)
	} else {
		amBevel(&hi, &lo, g, 0, 0, g.w, g.h, 1)
	}
	switch k {
	case CaptionClose:
		oy := (rows - 5) / 2 * amRow
		amG.closeFrame.emit(&lo, g, 7, oy)
		switch {
		case cs.Pressed():
			amG.closeMid.emit(&lo, g, 7, oy)
		case st.Active:
			amG.closeMid.emit(&hi, g, 7, oy)
		default:
			amG.closeMid.emit(&grey, g, 7, oy)
		}
	case CaptionMaximize:
		oy := (rows - 7) / 2 * amRow
		amG.zoomInk.emit(&lo, g, 5, oy)
		amG.zoomFill.emit(&blue, g, 5, oy)
		amG.zoomMid.emit(&hi, g, 5, oy)
	default:
		// The depth gadget stands in for minimize (its own job was to send
		// the window behind) and for the window menu.
		oy := (rows - 7) / 2 * amRow
		amG.depthInk.emit(&lo, g, 4, oy)
		amG.depthBack.emit(&grey, g, 4, oy)
		amG.depthFront.emit(&hi, g, 4, oy)
	}
	fill := c.fill
	if !st.Active {
		fill = c.win
	}
	grey.fill(ctx, c.win)
	blue.fill(ctx, fill)
	hi.fill(ctx, c.shine)
	lo.fill(ctx, c.shadow)
}

// frame13Gadget is a Workbench 1.3 title gadget: the blue line that sets it
// apart, a cell on each side, and the gadget's glyph on the white bar. A
// pressed close gadget is complemented, as Intuition drew it.
func (e amigaEngine) frame13Gadget(l *Classic, ctx *paintengine2d.Context, g rpGrid, k CaptionButton, cs ControlState, st DecorationState) {
	c := amColors(l)
	_, bar, _, _ := amFrameGeom(l)
	bar = min(bar, g.h)
	var blue, dark, orange, dot rpInk
	blue.cells(g, 0, 0, 1, bar)
	blue.cells(g, g.w-1, 0, 1, bar)
	box := g.sub(1, 0, g.w-2, bar)
	oy := (bar/amRow - 8) / 2 * amRow
	switch k {
	case CaptionClose:
		ox := (box.w - 16) / 2
		if cs.Pressed() {
			dark.cells(box, 0, 0, box.w, bar)
			amG.close13Box.emit(&orange, box, ox, oy)
			amG.close13Dot.emit(&dot, box, ox, oy)
		} else {
			amG.close13Box.emit(&blue, box, ox, oy)
			amG.close13Dot.emit(&dark, box, ox, oy)
		}
	case CaptionMaximize:
		// The front depth gadget: 1.3's "bring this window to the front".
		ox := (box.w - 20) / 2
		amG.frontDark.emit(&dark, box, ox, oy)
		amG.frontBlue.emit(&blue, box, ox, oy)
	default:
		ox := (box.w - 20) / 2
		amG.backBlue.emit(&blue, box, ox, oy)
		amG.backDark.emit(&dark, box, ox, oy)
	}
	dark.fill(ctx, c.dark)
	blue.fill(ctx, c.win)
	orange.fill(ctx, c.comp(c.win))
	dot.fill(ctx, c.line)
}
