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
	// The buttons are the caption: full height, flush to the top and to
	// the right corner, no gap between them and none around them. A
	// console's controls are cut out of the band rather than resting on
	// it, and a close button that is the whole corner is the one a hand
	// can hit without looking.
	return DecorationSpec{
		Border:     Insets{Top: lw, Right: lw, Bottom: lw, Left: lw},
		Caption:    h,
		Button:     paintengine2d.Pt(snap(h*1.15), h),
		ButtonGap:  0,
		ButtonPad:  Insets{},
		CloseGap:   0,
		Layout:     ":minimize,maximize,close",
		ButtonSide: ButtonsRight,
		Shadow:     ShadowLayersReach(hangarWindowShadow(l, st)),
	}
}

// WindowShape cuts three corners off the window: the top right, the
// bottom right and the bottom left.
//
// It is a *silhouette*, not a painted corner — the desktop shows through
// the cut, which is the whole point of it. The top left stays square
// because that is where the title and its ember mark are, and a window
// whose name started in a diagonal would read as a mistake.
//
// A path rather than a union of rectangles, so the diagonal is exact at
// 1.75 as at 1 rather than a staircase of whole pixels.
func (hangarEngine) WindowShape(l *Classic, f DecorationFrame, st DecorationState) *Silhouette {
	if f.Window.Empty() {
		return nil
	}
	// The frame's chamfer, not the box's: the close button cuts the same
	// corner and the two have to be the same diagonal.
	cut := hangarChamfer(l)
	if cut <= 0 || min(f.Window.Dx(), f.Window.Dy()) < cut*3 {
		return nil
	}
	return &Silhouette{Path: hangarCutRect(f.Window, cut, hangarPanelSides)}
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
	win := hangarSnap(f.Window)
	cut := hangarChamfer(l)
	if !bar.Empty() {
		// The band is clipped to the window's outline so it cannot paint
		// into the corner the silhouette cuts away.
		ctx.Save()
		ctx.ClipPath(hangarCutRect(win, cut, hangarPanelSides))
		ctx.DrawRect(bar, paintengine2d.Fill(band))
		ctx.DrawRect(hangarSnap(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-lw, bar.Dx(), lw)), paintengine2d.Fill(line))
		ctx.Restore()
		if st.Active {
			hangarMark(l, ctx, bar, c.accent)
		}
	}
	if st.NoBorder {
		return
	}
	// The border follows the cut rather than the rectangle, and the
	// bracket hairline sits inside its bottom left corner: the frame's
	// own version of the mark every panel in this look wears.
	//
	// Drawn here *and* over the content (DrawDecorationOver): a cut
	// crosses the content's box, so content that fills the corner — a
	// rail down the window's left edge, which is what reported this —
	// covers whatever was drawn there first.
	hangarEdge(l, ctx, win, cut, line, c.bracket, st)
}

// hangarEdge is the window's outline and its bracket: the two things
// this look puts at the very edge of a window.
func hangarEdge(l *Classic, ctx *paintengine2d.Context, win paintengine2d.Rect, cut float32, line, bracket paintengine2d.Color, st DecorationState) {
	lw := hangarPx(l)
	ctx.DrawPath(hangarCutRect(win.Inset(lw*0.5), cut, hangarPanelSides),
		paintengine2d.StrokePaint(line, lw))
	if st.Active {
		hangarBrackets(l, ctx, win, cut, bracket)
	}
}

// DrawDecorationOver puts the window's outline and its bracket back over
// the content.
//
// A rectangular border lies outside the content's box and nothing can
// cover it. A cut corner's does not: the chamfer crosses the box, so the
// hairline along it and the bracket inside it were painted first and
// then covered by whatever the content drew in that corner. secretvault
// reported it with a rail running down the window's left edge to the
// bottom, where this look cuts the corner and puts its mark.
func (hangarEngine) DrawDecorationOver(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	if st.NoBorder || f.Window.Empty() {
		return
	}
	c := hangarColorsOf(l)
	line := c.line
	if !st.Active {
		line = Mix(c.line, c.ground, 0.45)
	}
	hangarEdge(l, ctx, hangarSnap(f.Window), hangarChamfer(l), line, c.bracket, st)
}

func (hangarEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := hangarColorsOf(l)
	col := c.text
	if !st.Active {
		col = c.dim
	}
	// The title clears the ember mark at the head of the band.
	captionTitle(l, ctx, l.BoldFont(), b, hangarCaps(title), col, false, l.S(18))
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
		// The close button is the window's top right corner, so it wears
		// the same cut: the ember block ends in the diagonal the frame
		// does, and the two read as one shape rather than as a square
		// sitting in a cut corner.
		ctx.DrawPath(hangarCutRect(box, hangarChamfer(l), hangarSides{TR: true}),
			paintengine2d.Fill(fill))
		fg = c.onAccent
	} else if cs.Hovered() || cs.Pressed() {
		ctx.DrawRect(box, paintengine2d.Fill(hangarLift(c.band, cs)))
		hangarFrame(l, ctx, box, c.line)
	}
	u := min(box.Dx(), box.Dy()) / 26
	DrawCaptionGlyph(ctx, box, k, CaptionAlt(k, cs, st), fg, snap(9*u), max(snap(lw), 1))
}
