package style

import "github.com/codemodify/paintengine2d"

// The web engine's window frame (see engine_web.go): SourceGit's custom
// frame on Linux and Windows, which the other web packs share in their own
// colours. A merged title bar in the pack's title-bar colour with a 1px
// structure line along its bottom (the one the selected browser tab
// opens), 38px tall (30 maximized); a 1px window border; caption buttons
// 48×30 at the top right, flat, with a black-25% wash under the pointer
// and a close button that turns red (SourceGit's pure red, Windows 11's
// elsewhere); glyphs in the text colour — a 11px bar, a 10px square, a 9px
// cross. The window title is centred and bold.

func (webEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := webColors(l)
	px := c.px(l)
	capH := snap(l.S(38))
	if st.Maximized {
		capH = snap(l.S(30))
	}
	r := c.rad(l, 8)
	s := DecorationSpec{
		Border:  Insets{Top: px, Right: px, Bottom: px, Left: px},
		Caption: capH,
		Button:  paintengine2d.Pt(snap(l.S(48)), snap(l.S(30))),
		Layout:  ":minimize,maximize,close",
		Radius:  [4]float32{r, r, r, r},
		Shadow:  ShadowLayersReach(webWindowShadow(l, DecorationState{Active: true})),
	}
	return s
}

func (webEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := webColors(l)
	px := c.px(l)
	cb := winSnap(f.Caption)
	ctx.DrawRect(cb, paintengine2d.Fill(c.titleBar))
	ctx.DrawRect(paintengine2d.XYWH(cb.Min.X, cb.Max.Y-px, cb.Dx(), px), paintengine2d.Fill(c.border0))
	if !st.Maximized {
		border := c.windowBorder
		if !st.Active {
			border = Mix(c.windowBorder, c.titleBar, 0.35)
		}
		drawFrameBorder(ctx, f.Window, Insets{Top: px, Right: px, Bottom: px, Left: px}, border)
	}
}

func (webEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := webColors(l)
	col := c.text
	if !st.Active {
		col = c.text2
	}
	captionTitle(l, ctx, c.bold, b, title, col, true, l.S(16))
}

func (webEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := webColors(l)
	px := c.px(l)
	hover := l.X("captionHover", webA(c.wash, 1.6))
	flatCaption{
		fg: c.text, fgOff: c.text2,
		hover: hover, press: webA(hover, 1.5),
		close: c.close, closePress: Mix(c.close, paintengine2d.RGB(0, 0, 0), 0.15), onClose: c.onClose,
		min: snap(l.S(11)), max: snap(l.S(10)), cls: snap(l.S(9)), lw: px,
	}.draw(ctx, winSnap(b), k, cs, st)
}

// DrawBrowserTabBar is the strip under browser tabs: the title bar's colour
// with the structure line the selected tab opens along its bottom.
func (webEngine) DrawBrowserTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	b = winSnap(b)
	if b.Empty() {
		return
	}
	px := c.px(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.titleBar))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border0))
}

// webWindowShadow is SourceGit's own frame shadow — a 12px blur of black
// at 38% around the window — dimmed in the backdrop.
func webWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	a := float32(0.38)
	if !st.Active {
		a = 0.2
	}
	return []WindowShadow{{Color: shadowBlack(a), Blur: l.S(12)}}
}

func (webEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, webWindowShadow(l, st))
}
