package style

import "github.com/codemodify/paintengine2d"

// The OS/2 Warp 4 window frame (see engine_os2.go): a stacked frame, the
// Presentation Manager frame window. The four-pixel sizing border in the
// #CCCCCC face with its two bevel rings; above the client area the title
// row — the program's mini icon at the left, which opens the system menu,
// the sunken title well in #2B00AA with the bold title ten pixels in, and
// the glyphs at the right in Warp 4's order: Close (which Warp 4 added, left
// of the others), Hide and Maximize, fourteen pixels each and two apart,
// the well ending three pixels before the first of them.
//
// That order is the layout the pack asks for. PM's frames were square and
// cast no shadow.
//
// Sizes, the glyph shapes and the colours are engine_os2.go's, taken as
// facts from Warp 4 screenshots and documentation; the mini icon is that
// file's invention, drawn here at twelve pixels so it shares the glyphs' box.

func (os2Engine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	well := o2wellH(l)
	return DecorationSpec{
		Stacked:   true,
		Border:    Insets{Top: o2above * u, Right: o2border * u, Bottom: o2border * u, Left: o2border * u},
		Caption:   float32(well+o2below) * u,
		Button:    paintengine2d.Pt(o2glyph*u, o2glyph*u),
		ButtonGap: 2 * u,
		ButtonPad: Insets{
			Top:   float32((well-o2glyph)/2) * u,
			Right: (o2border + 1) * u,
			Left:  (o2border + 1) * u,
		},
		Layout: "icon:close,minimize,maximize",
	}
}

func (os2Engine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := o2colors(l)
	u := rpU(l)
	border := Insets{Top: o2above * u, Right: o2border * u, Bottom: o2border * u, Left: o2border * u}
	if st.Maximized {
		border = Insets{}
	}
	for _, part := range FrameParts(f, border) {
		ctx.DrawRect(part, paintengine2d.Fill(c.face))
	}
	if st.Maximized {
		return
	}
	g := rpGridAt(ctx, f.Window, u)
	if g.w < 8 || g.h < 8 {
		return
	}
	var white, dark, black rpInk
	rpEdge(&dark, &black, g, 0, 0, g.w, g.h, 1)
	rpEdge(&white, &dark, g, 1, 1, g.w-2, g.h-2, 1)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
}

// DrawCaptionTitle is the sunken title well with the bold title in it, the
// well ending three pixels short of the glyphs.
func (os2Engine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	wh := min(o2wellH(l), g.h)
	if g.w < 14 || wh < 4 {
		return
	}
	wg := g.sub(0, 0, g.w-3, wh)
	var white, dark, well rpInk
	well.cells(wg, 1, 1, wg.w-2, wg.h-2)
	rpEdge(&dark, &white, wg, 0, 0, wg.w, wg.h, 1)
	fill, tc := c.title, c.titleTxt
	if !st.Active {
		fill, tc = c.titleOff, c.titleOffTxt
	}
	well.fill(ctx, fill)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	l.drawFittedText(ctx, l.BoldFont(), title, wg.at(10, 1, wg.w-12, wg.h-2), tc, AlignStart, 0)
}

func (os2Engine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := o2colors(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < o2glyph || g.h < o2glyph {
		return
	}
	g = g.sub((g.w-o2glyph)/2, (g.h-o2glyph)/2, o2glyph, o2glyph)
	var white, dark, black, strip rpInk
	switch k {
	case CaptionClose:
		o2closeGlyph(&white, &dark, g, cs.Pressed())
	case CaptionMinimize:
		o2hideGlyph(&white, &dark, g)
	case CaptionMaximize:
		if st.Maximized {
			// Restore: the sunken square shrinks, as the window is about to.
			o2bevel(&white, &dark, g, 0, 0, o2glyph, o2glyph, 1, cs.Pressed())
			o2bevel(&white, &dark, g, 4, 4, o2glyph-8, o2glyph-8, 1, true)
			break
		}
		o2maxGlyph(&white, &dark, g)
	case CaptionMenu:
		o2frameMenuGlyph(&white, &dark, &black, &strip, g)
	}
	strip.fill(ctx, c.title)
	white.fill(ctx, c.hi)
	dark.fill(ctx, c.dark)
	black.fill(ctx, c.black)
}

// o2frameMenuGlyph appends the system-menu glyph in the 14-cell grid g: the
// mini icon of o2miniIcon at twelve pixels, so it shares the other glyphs'
// box — a small window with a title strip over a white client.
func o2frameMenuGlyph(white, dark, black, strip *rpInk, g rpGrid) {
	dark.cells(g, 11, 3, 1, 9)
	dark.cells(g, 3, 11, 8, 1)
	black.frame(g, 2, 2, 9, 9, 1)
	strip.cells(g, 3, 3, 7, 2)
	black.cells(g, 3, 5, 7, 1)
	white.cells(g, 3, 6, 7, 4)
	dark.cells(g, 4, 7, 5, 1)
}
