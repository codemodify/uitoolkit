//go:build theme_engine_all || theme_engine_neumorphism || (!theme_engine_amiga && !theme_engine_beos && !theme_engine_bluecurve && !theme_engine_kde1 && !theme_engine_kde2 && !theme_engine_macos_tahoe && !theme_engine_openlook && !theme_engine_os2 && !theme_engine_oxygen && !theme_engine_plastik && !theme_engine_platinum && !theme_engine_win31)

package style

import "github.com/codemodify/paintengine2d"

// The soft-UI window frame. It follows the same rule as everything else
// in this engine: the frame is not a band laid on the window, it is the
// window's own surface, and the only thing separating it from the content
// below is the light falling on it. So there is no border, no caption
// line, and no filled title bar — the caption is the surface, and the
// title sits on it.
//
// The caption buttons are the one place the look uses a shape: each is a
// small raised disc, because a bare glyph on a borderless surface has
// nothing to say it can be clicked. They sink when pressed, which is the
// same inversion every other control uses.
//
// An engine without a frame of its own gets the generic adapter, which
// would give this look a bevelled caption from another century.

// neuCaptionH is the title row: tall enough for the buttons to be discs
// with room around them.
func neuCaptionH(l *Classic) float32 {
	return max(snap(l.body.Height()+l.S(18)), snap(l.S(40)))
}

// neuCapButton is the diameter of a caption button's disc.
func neuCapButton(l *Classic) float32 {
	return max(snap(min(l.S(22), neuCaptionH(l)-l.S(12))), 8)
}

func (neuEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	h := neuCaptionH(l)
	side := neuCapButton(l)
	r := l.S(l.Metrics().Radius)
	if r <= 0 {
		r = l.S(14)
	}
	return DecorationSpec{
		// No border: a line round the window would cut the surface.
		Border:        Insets{},
		Caption:       h,
		Button:        paintengine2d.Pt(side, side),
		ButtonGap:     snap(l.S(8)),
		ButtonPad:     Insets{Top: snap((h - side) * 0.5), Left: snap(l.S(10)), Right: snap(l.S(10))},
		CenterButtons: true,
		CenterTitle:   true,
		Layout:        "icon:minimize,maximize,close",
		Radius:        [4]float32{r, r, 0, 0},
		Shadow:        ShadowLayersReach(neuWindowShadow(l, st)),
	}
}

// neuWindowShadow is the look's own shadow language at window scale: a
// wide, soft, low-contrast fall, and a shallower one behind an inactive
// window.
func neuWindowShadow(l *Classic, st DecorationState) []WindowShadow {
	if !st.Active {
		return []WindowShadow{{Color: shadowBlack(0.14), DY: l.S(4), Blur: l.S(18)}}
	}
	return []WindowShadow{
		{Color: shadowBlack(0.20), DY: l.S(10), Blur: l.S(30)},
		{Color: shadowBlack(0.10), DY: l.S(2), Blur: l.S(8)},
	}
}

// DrawDecoration paints the caption as more surface. Nothing is drawn
// across the window's body: the frame has no border, and the caption is
// the same colour as what is under it.
func (neuEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	g := neuColors(l)
	if f.Caption.Dx() <= 0 || f.Caption.Dy() <= 0 {
		return
	}
	ctx.DrawRect(f.Caption, paintengine2d.Fill(g.surface))
}

// DrawCaptionTitle writes the title straight onto the surface, dimmed on
// an inactive window because there is no caption colour to change.
func (e neuEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	if title == "" || b.Dx() <= 0 {
		return
	}
	g := neuColors(l)
	col := g.text
	if !st.Active {
		col = g.dim
	}
	l.drawFittedText(ctx, l.BoldFont(), title, b, col, AlignCenter, 0)
}

// DrawCaptionButton is a small raised disc with the glyph over it, sunk
// when pressed. Close takes the accent on hover rather than the red every
// other look uses: a red disc on this surface reads as an error, not a
// button.
func (e neuEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	g := neuColors(l)
	d := min(b.Dx(), b.Dy())
	box := paintengine2d.XYWH(b.Min.X+(b.Dx()-d)/2, b.Min.Y+(b.Dy()-d)/2, d, d)
	r := d / 2
	if cs&StatePressed != 0 {
		g.softIn(l, ctx, box, r)
	} else {
		g.softOut(l, ctx, box, r)
	}
	col := g.text
	if !st.Active {
		col = g.dim
	}
	if cs&StateHovered != 0 {
		col = g.accent
	}
	face, _, _ := neuFace(l, box, r)
	DrawCaptionGlyph(ctx, face, k, CaptionAlt(k, cs, st), col,
		snap(face.Dx()*0.52), max(l.S(1.5), 1))
}
