package style

import "github.com/codemodify/paintengine2d"

// The macOS window frame (see engine_macos.go), Yosemite to today: a merged
// title bar — the app's tool bar is the title bar, as in a unified tool-bar
// window — 28px for a plain title (22px before Big Sur) and 52px for a tool
// bar of the app's own (38px before), in the title bar's colour with a line
// under it and a hairline round the window. The traffic lights are 12px
// circles 20px apart from 12px in (8px before Big Sur), centred in the
// title bar, where the theme's layout puts them on the left; grey in a
// backdrop window, their glyphs (×, −, +) showing under the pointer. The
// title is centred on the window.

func (macosEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	c := macColors(l)
	px := macPx(l)
	capH, pad := l.S(22), l.S(8)
	if c.bigSur {
		capH, pad = l.S(28), l.S(12)
	}
	if st.Custom {
		capH = l.S(38)
		if c.bigSur {
			capH = l.S(52)
		}
	}
	d := snap(l.S(12))
	capH = snap(max(capH, l.body.Height()+l.S(6)))
	r := l.rx(4)
	if c.bigSur {
		r = l.rx(10)
	}
	s := DecorationSpec{
		Border:        Insets{Top: px, Right: px, Bottom: px, Left: px},
		Caption:       capH,
		Button:        paintengine2d.Pt(d, d),
		ButtonGap:     snap(l.S(20)) - d,
		ButtonPad:     Insets{Top: snap((capH - d) * 0.5), Left: snap(pad) - px, Right: snap(pad) - px},
		CenterButtons: true,
		CenterTitle:   true,
		Layout:        "close,minimize,maximize:",
		Radius:        [4]float32{r, r, r, r},
		Shadow:        ShadowLayersReach(macosWindowShadow(l, DecorationState{Active: true})),
	}
	return s
}

// macosWindowShadow is the Mac's window shadow: a large soft one under the
// key window and a much smaller one under the others. Apple publishes no
// numbers; these are measured off screenshots of Big Sur and Yosemite.
func macosWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.18), DY: l.S(4), Blur: l.S(16)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.34), DY: l.S(18), Blur: l.S(48)},
		{Color: shadowBlack(0.14), DY: l.S(2), Blur: l.S(8)},
	}
}

func (macosEngine) DrawDecorationShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st DecorationState) {
	DrawShadowLayers(ctx, b, DecorationOf(l, st).Radius, macosWindowShadow(l, st))
}

func (macosEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := macColors(l)
	px := macPx(l)
	bar := macSnap(f.Caption)
	if st.Active {
		ctx.DrawRect(bar, VGradient(bar, c.titleStops...))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-px, bar.Dx(), px), paintengine2d.Fill(c.titleEdge))
		if !c.bigSur && !c.dark {
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), px), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.6)))
		}
	} else {
		ctx.DrawRect(bar, paintengine2d.Fill(c.titleOffBar))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-px, bar.Dx(), px), paintengine2d.Fill(c.titleOffEdge))
	}
	if !st.Maximized {
		drawFrameBorder(ctx, f.Window, Insets{Top: px, Right: px, Bottom: px, Left: px}, c.winEdge)
	}
}

func (macosEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := macColors(l)
	col := c.titleText
	if !st.Active {
		col = c.titleOff
	}
	f := l.body
	if c.bigSur {
		f = l.WeightFont(WeightSemibold)
	}
	captionTitle(l, ctx, f, b, title, col, true, l.S(8))
}

func (macosEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := macColors(l)
	px := macPx(l)
	i := 0
	switch k {
	case CaptionMinimize:
		i = 1
	case CaptionMaximize:
		i = 2
	}
	fill, rim := c.lightOff, c.lightOffEdge
	if k != CaptionMenu && (st.Active || cs.Hovered()) {
		fill, rim = c.lights[i], c.lightEdges[i]
	}
	if cs.Pressed() {
		fill = mdOver(fill, paintengine2d.RGB(0, 0, 0), 0.2)
	}
	ctr := paintengine2d.Pt((b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5)
	rad := min(b.Dx(), b.Dy()) * 0.5
	ctx.DrawCircle(ctr, rad, paintengine2d.Fill(rim))
	ctx.DrawCircle(ctr, rad-px*0.75, paintengine2d.Fill(fill))
	if !cs.Hovered() && !cs.Pressed() {
		return
	}
	glyph := Mix(rim, paintengine2d.RGB(0, 0, 0), 0.55)
	if i == 0 {
		glyph = c.glyph
	}
	s := b.Inset(b.Dx() * 0.28)
	lw := max(l.S(1.2), 1)
	switch k {
	case CaptionClose:
		DrawCross(ctx, s, glyph, lw)
	case CaptionMinimize:
		ctx.DrawRect(paintengine2d.XYWH(s.Min.X, snap((s.Min.Y+s.Max.Y-lw)*0.5), s.Dx(), max(snap(lw), 1)), paintengine2d.Fill(glyph))
	default:
		t := max(snap(lw), 1)
		ctx.DrawRect(paintengine2d.XYWH(s.Min.X, snap((s.Min.Y+s.Max.Y-t)*0.5), s.Dx(), t), paintengine2d.Fill(glyph))
		ctx.DrawRect(paintengine2d.XYWH(snap((s.Min.X+s.Max.X-t)*0.5), s.Min.Y, t, s.Dy()), paintengine2d.Fill(glyph))
	}
}
