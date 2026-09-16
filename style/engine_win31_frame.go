package style

import "github.com/codemodify/paintengine2d"

// The Windows 3.1 window frame (see engine_win31.go): a stacked frame. The
// sizing frame — a black line, four pixels of the active border colour and a
// white line — round the window, the caption between black lines in navy
// with the bold white title centred, the control-menu box at the caption's
// left end with a black divider down its right, and at its right end the
// minimize and maximize buttons, small bevelled squares with a down and an
// up triangle and a black line down their left. Restore, while the window
// is maximized, shows both triangles at once, as Program Manager's did.
//
// The control-menu box is the window menu (double-clicking it closed the
// window; 3.1 had no close button at all), which is what the pack's layout
// asks for: "icon:minimize,maximize". Where a desktop's layout asks for
// close as well it gets the same small bevelled square with a cross drawn
// the way 3.1 drew its 1-bit glyphs.
//
// Square, shadowless and unrounded, and the caption colour follows the
// scheme (Hot Dog Stand's is its own). Sizes are engine_win31.go's: the
// original VGA pixel counts.

// w31FrameBorder is the sizing frame: the black line, the border band and
// the white line.
func w31FrameBorder(l *Classic) Insets {
	u := rpU(l)
	side := float32(1+w31band+1) * u
	// The caption sits under a black line of its own.
	return Insets{Top: float32(1+w31bandTop+1+1) * u, Right: side, Bottom: side, Left: side}
}

func (win31Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	h := w31barH(l)
	return DecorationSpec{
		Stacked: true,
		Border:  w31FrameBorder(l),
		// The caption and the black line under it.
		Caption: float32(h+1) * u,
		Button:  paintengine2d.Pt(float32(h)*u, float32(h)*u),
		Layout:  "icon:minimize,maximize",
	}
}

func (win31Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := w31colors(l)
	u := rpU(l)
	border := w31FrameBorder(l)
	if st.Maximized {
		border = Insets{}
	}
	bar := c.caption
	if !st.Active {
		bar = c.captionOff
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.border))
	}
	ctx.DrawRect(f.Caption, paintengine2d.Fill(bar))
	g := rpGridAt(ctx, f.Window, u)
	if g.w < 8 || g.h < 8 {
		return
	}
	var white, black rpInk
	// The black line under the caption, which a maximized window keeps.
	cg := rpGridAt(ctx, f.Caption, u)
	black.cells(cg, 0, max(cg.h-1, 0), cg.w, 1)
	if !st.Maximized {
		black.frame(g, 0, 0, g.w, g.h, 1)
		white.frame(g, w31band+1, w31bandTop+1, g.w-2*w31band-2, g.h-w31bandTop-w31band-2, 1)
		// The black line above the caption closes the white one off.
		black.cells(g, w31band+1, w31bandTop+1+1, g.w-2*w31band-2, 1)
	}
	white.fill(ctx, c.hi)
	black.fill(ctx, c.frame)
}

func (win31Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := w31colors(l)
	u := rpU(l)
	col := c.captionText
	if !st.Active {
		col = c.captionOffT
	}
	g := rpGridAt(ctx, b, u)
	if g.w < 6 {
		return
	}
	l.drawFittedText(ctx, l.BoldFont(), title, g.at(2, 0, g.w-4, max(g.h-1, 1)), col, AlignCenter, 0)
}

func (e win31Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := w31colors(l)
	u := rpU(l)
	h := w31barH(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 6 || g.h < 6 {
		return
	}
	g = g.sub(0, 0, min(g.w, h), min(g.h, h))
	var white, black, hi, lo, glyph rpInk
	if k == CaptionMenu {
		// The control-menu box: a grey square with a divider down its right
		// and the "space bar" glyph — a white bar outlined in black with a
		// dark grey shadow.
		face := c.face
		if cs.Pressed() {
			face = w31xor(c.face)
		}
		ctx.DrawRect(g.at(0, 0, g.w-1, g.h), paintengine2d.Fill(face))
		black.cells(g, g.w-1, 0, 1, g.h)
		bw := min(13, g.h-4)
		bx, by := (g.w-1-bw)/2, (g.h-3)/2
		black.frame(g, bx, by, bw, 3, 1)
		white.cells(g, bx+1, by+1, bw-2, 1)
		lo.cells(g, bx+1, by+3, bw, 1)
		lo.cells(g, bx+bw, by+1, 1, 2)
		white.fill(ctx, c.hi)
		lo.fill(ctx, c.shadow)
		black.fill(ctx, c.frame)
		return
	}
	// The small bevelled buttons, each with a black line down its left.
	black.cells(g, 0, 0, 1, g.h)
	box := g.sub(1, 0, g.w-1, g.h)
	ctx.DrawRect(box.rect(), paintengine2d.Fill(c.face))
	if cs.Pressed() {
		// Pressed: the bevel turns over, as 3.1's small buttons did.
		w31small(&lo, &hi, box, 0, 0, box.w, box.h)
	} else {
		w31small(&hi, &lo, box, 0, 0, box.w, box.h)
	}
	switch {
	case k == CaptionMinimize:
		t := w31tri(7, DirDown)
		t.emit(&glyph, box, (box.w-t.w)/2, (box.h-t.h)/2)
	case k == CaptionMaximize && st.Maximized:
		// Restore: the up and down triangles together.
		up, dn := w31tri(7, DirUp), w31tri(7, DirDown)
		up.emit(&glyph, box, (box.w-up.w)/2, (box.h-up.h)/2-up.h/2)
		dn.emit(&glyph, box, (box.w-dn.w)/2, (box.h-dn.h)/2+dn.h/2+1)
	case k == CaptionMaximize:
		t := w31tri(7, DirUp)
		t.emit(&glyph, box, (box.w-t.w)/2, (box.h-t.h)/2)
	default:
		// Close, which 3.1 had no button for: a 1-bit cross.
		n := min(box.w, box.h) - 6
		m := rpNewMask(box.w, box.h)
		x0, y0 := (box.w-n)/2, (box.h-n)/2
		m.line(x0, y0, x0+n-1, y0+n-1)
		m.line(x0+n-1, y0, x0, y0+n-1)
		m.line(x0+1, y0, x0+n-1, y0+n-2)
		m.line(x0+n-2, y0, x0, y0+n-2)
		m.emit(&glyph, box, 0, 0)
	}
	hi.fill(ctx, c.hi)
	lo.fill(ctx, c.shadow)
	glyph.fill(ctx, c.btnText)
	black.fill(ctx, c.frame)
}
