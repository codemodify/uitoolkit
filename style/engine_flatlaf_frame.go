package style

import "github.com/codemodify/paintengine2d"

// The FlatLaf window frame (see engine_flatlaf.go), after FlatLaf's own
// decorated title pane: a merged title bar in the window colour — the
// menu bar and the app's tool bar sit in it — a 1px border, the title at
// the left, and 44×30 flat caption buttons at the top right that darken
// (lighten on the dark themes) by a tenth under the pointer and a fifth
// pressed; close turns red (a lighter red pressed) behind a white glyph.
// Glyphs are 10px lines of one pixel.

func (flatlafEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	px := flatPx(l)
	h := flatTitleH(l)
	return DecorationSpec{
		Border:  Insets{Top: px, Right: px, Bottom: px, Left: px},
		Caption: h,
		Button:  paintengine2d.Pt(snap(l.S(44)), snap(l.S(30))),
		Layout:  ":minimize,maximize,close",
		Shadow:  flatlafEngine{}.PopupShadow(l, PopupDialog),
	}
}

func (flatlafEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := flatColors(l)
	ctx.DrawRect(flatSnap(f.Caption), paintengine2d.Fill(c.bg))
	if !st.Maximized {
		px := flatPx(l)
		drawFrameBorder(ctx, f.Window, Insets{Top: px, Right: px, Bottom: px, Left: px}, c.border)
	}
}

func (flatlafEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := flatColors(l)
	col := c.titleText
	if !st.Active {
		col = c.titleOff
	}
	captionTitle(l, ctx, l.body, b, title, col, false, l.S(10))
}

func (flatlafEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := flatColors(l)
	shade := paintengine2d.RGB(0, 0, 0)
	if c.dark {
		shade = paintengine2d.RGB(1, 1, 1)
	}
	g := snap(min(l.S(10), b.Dy()*0.4))
	flatCaption{
		fg: c.titleText, fgOff: c.titleOff,
		hover: shade.WithAlpha(0.1), press: shade.WithAlpha(0.2),
		close: c.close, closePress: Mix(c.close, paintengine2d.RGB(1, 1, 1), 0.3), onClose: paintengine2d.RGB(1, 1, 1),
		min: g, max: g, cls: g, lw: flatPx(l),
	}.draw(ctx, flatSnap(b), k, cs, st)
}
