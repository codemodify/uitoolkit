//go:build theme_engine_all || theme_engine_hangar

package style

import "github.com/codemodify/paintengine2d"

// The console's own window frame (see engine_hangar.go): a 30px caption
// band over a hairline, the title in capitals at the left with an ember
// tick before it, and square caption buttons at the right — the close one
// a solid ember block, which is the one button on the panel you are not
// supposed to press by accident and therefore the one that is marked.
//
// Square corners and a plain dark shadow: a console is a rectangle cut
// out of the dark, and a soft halo round it would be the only round thing
// in the whole look.

func (hangarEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	lw := hangarPx(l)
	h := snap(max(l.S(30), l.body.Height()+l.S(12)))
	side := snap(min(h-lw*2, l.S(26)))
	return DecorationSpec{
		Border:     Insets{Top: lw, Right: lw, Bottom: lw, Left: lw},
		Caption:    h,
		Button:     paintengine2d.Pt(side, side),
		ButtonGap:  lw,
		ButtonPad:  Insets{Top: snap((h - side) * 0.5), Right: snap((h-side)*0.5) - lw},
		CloseGap:   snap(l.S(6)),
		Layout:     ":minimize,maximize,close",
		ButtonSide: ButtonsRight,
		Shadow:     ShadowLayersReach(hangarWindowShadow(l, st)),
	}
}

// hangarWindowShadow is a plain dark drop, deeper for the active window:
// no spread of colour, because the ground is already dark and a soft halo
// would be the only curve in the look.
func hangarWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.35), DY: l.S(2), Blur: l.S(8)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.6), DY: l.S(4), Blur: l.S(16)},
		{Color: shadowBlack(0.3), Blur: l.S(2)},
	}
}

func (hangarEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, [4]float32{}, hangarWindowShadow(l, st))
}

func (hangarEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := hangarColorsOf(l)
	lw := hangarPx(l)
	bar := hangarSnap(f.Caption)
	band := c.band
	line := c.line
	if !st.Active {
		// The backdrop window loses its contrast rather than its colour:
		// a console with two windows open should still read as one
		// surface, with the live one brighter.
		band = Mix(c.band, c.ground, 0.55)
		line = Mix(c.line, c.ground, 0.45)
	}
	if !bar.Empty() {
		ctx.DrawRect(bar, paintengine2d.Fill(band))
		ctx.DrawRect(hangarSnap(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-lw, bar.Dx(), lw)), paintengine2d.Fill(line))
		if st.Active {
			hangarTick(l, ctx, paintengine2d.Pt(bar.Min.X, bar.Min.Y), l.S(20), c.accent)
		}
	}
	drawFrameBorder(ctx, f.Window, Insets{Top: lw, Right: lw, Bottom: lw, Left: lw}, line, st)
}

func (hangarEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := hangarColorsOf(l)
	col := c.text
	if !st.Active {
		col = c.dim
	}
	captionTitle(l, ctx, l.BoldFont(), b, hangarCaps(title), col, false, l.S(12))
}

// DrawCaptionButton: close is a filled ember block, the other two are
// hairline squares that fill with white on the way to being pressed.
func (hangarEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := hangarColorsOf(l)
	box := hangarSnap(b)
	lw := hangarPx(l)
	fg := c.text
	if !st.Active && !cs.Hovered() {
		fg = c.dim
	}
	if k == CaptionClose {
		fill := c.accent
		switch {
		case cs.Pressed():
			fill = c.press
		case cs.Hovered():
			fill = c.hot
		case !st.Active:
			fill = Mix(c.accent, c.ground, 0.45)
		}
		ctx.DrawRect(box, paintengine2d.Fill(fill))
		fg = c.onAccent
	} else if cs.Hovered() || cs.Pressed() {
		ctx.DrawRect(box, paintengine2d.Fill(hangarLift(c.band, cs)))
		hangarFrame(l, ctx, box, c.line)
	}
	u := min(box.Dx(), box.Dy()) / 26
	DrawCaptionGlyph(ctx, box, k, CaptionAlt(k, cs, st), fg, snap(9*u), max(snap(lw), 1))
}
