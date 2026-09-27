//go:build theme_engine_all || theme_engine_neumorphism || (!theme_engine_amiga && !theme_engine_beos && !theme_engine_bluecurve && !theme_engine_kde1 && !theme_engine_kde2 && !theme_engine_macos_tahoe && !theme_engine_openlook && !theme_engine_os2 && !theme_engine_oxygen && !theme_engine_plastik && !theme_engine_platinum && !theme_engine_win31)

package style

import "github.com/codemodify/paintengine2d"

// neuEngine paints Neumorphism, or "soft UI": the look that came out of a
// 2019 Dribbble post by Alexander Plyuto and spread through design work
// for a couple of years. Everything is one colour. A control is not a
// shape laid *on* the background, it is the background pushed out of the
// surface or pressed into it, and the only thing that says which is a
// pair of shadows — light from the top left, dark from the bottom right.
// Press it and the two swap, so the control sinks.
//
// That single rule decides everything else. There are almost no borders,
// because an edge would break the illusion that the surface is
// continuous. Contrast is low by construction: the face and the ground
// are the same colour, so the only contrast available is the shadows and
// the text. Corners are large and even.
//
// **It is the toolkit's default engine**, the one an application gets
// when it builds no others in (see docs/engines.md), so it is always
// built and never carries a build tag. Two things follow. It may not
// depend on another era engine — it embeds [BaseEngine], which is also
// always built, and overrides the parts that make it soft. And it has to
// be legible on its own, since it may be the only look an application
// ships: the accent stays saturated for focus and selection, and text
// keeps a real contrast ratio against the surface rather than the
// washed-out grey the style is usually criticised for.
//
// Pack data: no params. Two packs, a light "neumorphism" and a dark
// "neumorphism-night".
type neuEngine struct{ BaseEngine }

func init() {
	RegisterEngine(neuEngine{})
	for _, p := range neuPacks() {
		RegisterPack(p)
	}
}

func (neuEngine) ID() string { return "neumorphism" }

// DefaultMetrics: generous and round. A soft control needs room for its
// shadows to read, so everything is a little taller than a flat look and
// the radius is large enough that the highlight runs round the corner
// rather than stopping at it.
func (neuEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Radius:    14,
		Scroll:    14,
		FieldH:    34,
		Checkbox:  20,
		MenuItemH: 30,
		TitleBar:  38,
		Border:    0, // a border would break the continuous surface
		ViewFrame: 0, // a view is pressed in, not framed
		FieldPad:  10,
	}
}

// neuDepth is how far the shadows reach at 1x, and neuSpread how soft
// they are. Both scale with the look.
const (
	neuDepth  = 3.0
	neuSpread = 4.0
)

// neuSet is a look's resolved soft-UI colours: the one surface colour,
// the light that falls on it and the shadow it casts.
type neuSet struct {
	surface, light, shade paintengine2d.Color
	text, dim, accent     paintengine2d.Color
	onAccent              paintengine2d.Color
}

// neuColors derives the pair of shadows from the surface itself, which is
// what keeps the look coherent when a pack changes the background: the
// light is the surface lifted towards white, the shade the same surface
// pushed towards black, and neither is a fixed grey.
func neuColors(l *Classic) neuSet {
	p := l.Palette()
	s := p.Background
	lift, sink := float32(0.55), float32(0.34)
	if RelLuminance(s) < 0.4 {
		// On a dark surface the highlight has much less room than the
		// shadow, so it is lifted harder and the shadow is gentler;
		// equal amounts would make a dark control look flat.
		lift, sink = 0.22, 0.45
	}
	return neuSet{
		surface:  s,
		light:    Mix(s, paintengine2d.White, lift),
		shade:    Mix(s, paintengine2d.Black, sink),
		text:     p.Text,
		dim:      p.TextMuted,
		accent:   p.Accent,
		onAccent: p.TextOnAccent,
	}
}

// neuReach is how far the shadows need outside the face, so the face can
// be inset by it and the whole control still paint inside the rect it was
// given.
func neuReach(l *Classic) float32 { return l.S(neuDepth) + l.S(neuSpread)*0.5 }

// softOut paints b as the surface pushed out of the page: the dark shadow
// down and right, the light one up and left, then the face over both.
//
// **The shadows are drawn inside b, not around it.** A control is given a
// rect and everything it draws belongs in that rect — the same rule the
// window frame follows for its own drop shadow. So the face is inset by
// [neuReach] and the shadows live in the margin that inset leaves. It
// costs the look nothing: the face wanted breathing room anyway, and it
// means a soft control can be laid out beside anything else without
// bleeding over it.
//
// Each shadow is a few translucent rounded rects stepping outwards rather
// than a real blur — at this radius the difference is invisible and the
// cost is a handful of fills instead of a convolution.
func (g neuSet) softOut(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32) {
	face, fr, reach := neuFace(l, b, r)
	if face.Dx() <= 0 || face.Dy() <= 0 {
		return
	}
	// Constant alpha per ring, offsets spanning the whole reach and
	// drawn outermost first. The falloff then comes from *coverage*
	// rather than from the alpha: near the face every ring overlaps, at
	// the edge of the reach only the outermost does. Weighting the
	// alpha instead makes the outer rings invisible and the shadow
	// never leaves the face.
	const steps, ringA = 5, 0.17
	for i := steps; i >= 1; i-- {
		off := reach * float32(i) / float32(steps)
		ctx.DrawPath(RoundRectPath(face.Translate(paintengine2d.Pt(off, off)), fr, fr, fr, fr),
			paintengine2d.Fill(withA(g.shade, ringA)))
		ctx.DrawPath(RoundRectPath(face.Translate(paintengine2d.Pt(-off, -off)), fr, fr, fr, fr),
			paintengine2d.Fill(withA(g.light, ringA)))
	}
	ctx.DrawPath(RoundRectPath(face, fr, fr, fr, fr), paintengine2d.Fill(g.surface))
}

// softIn is the same surface pressed into the page: the shadows swap
// sides and move inside the shape, so the control reads as a well. It is
// clipped to the face, so it is inside b for the same reason.
func (g neuSet) softIn(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32) {
	face, fr, reach := neuFace(l, b, r)
	if face.Dx() <= 0 || face.Dy() <= 0 {
		return
	}
	ctx.DrawPath(RoundRectPath(face, fr, fr, fr, fr), paintengine2d.Fill(g.surface))
	ctx.Save()
	ctx.ClipPath(RoundRectPath(face, fr, fr, fr, fr))
	const steps, ringA = 5, 0.19
	w := max(l.S(2), 1)
	for i := steps; i >= 1; i-- {
		off := reach * float32(i) / float32(steps)
		a := float32(ringA)
		in := face.Inset(w * 0.5)
		ctx.DrawRoundRect(in.Translate(paintengine2d.Pt(off, off)), fr, fr,
			paintengine2d.Paint{Color: withA(g.shade, a), Style: paintengine2d.StyleStroke,
				Stroke: paintengine2d.Stroke{Width: w, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		ctx.DrawRoundRect(in.Translate(paintengine2d.Pt(-off, -off)), fr, fr,
			paintengine2d.Paint{Color: withA(g.light, a), Style: paintengine2d.StyleStroke,
				Stroke: paintengine2d.Stroke{Width: w, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	}
	ctx.Restore()
}

// neuFace is the face inside b, its corner radius, and how far the
// shadows may reach. A control too small to hold a full inset keeps as
// much of one as it can rather than vanishing.
func neuFace(l *Classic, b paintengine2d.Rect, r float32) (paintengine2d.Rect, float32, float32) {
	reach := neuReach(l)
	if m := min(b.Dx(), b.Dy()) / 4; reach > m {
		reach = max(m, 0)
	}
	face := b.Inset(reach)
	return face, min(r, min(face.Dx(), face.Dy())/2), reach
}

// withA is c at alpha a.
func withA(c paintengine2d.Color, a float32) paintengine2d.Color {
	c.A = a
	return c
}

// neuRadius is the look's corner radius for a box of height h, never more
// than half of it, so a short control is a capsule rather than a
// mis-drawn rectangle.
func neuRadius(l *Classic, h float32) float32 {
	r := l.S(l.Metrics().Radius)
	if r <= 0 {
		r = l.S(14)
	}
	return min(r, h/2)
}

// Face: pressed and checked controls are pressed *into* the surface,
// everything else stands out of it. A disabled control keeps its shape
// and loses its shadows, which is the softest way this look can say
// "not now" without a colour it does not have.
func (e neuEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	g := neuColors(l)
	r := neuRadius(l, b.Dy())
	fg := g.text
	switch {
	case st&StateDisabled != 0:
		ctx.DrawPath(RoundRectPath(b, r, r, r, r), paintengine2d.Fill(g.surface))
		return g.dim
	case st&(StatePressed|StateChecked) != 0:
		g.softIn(l, ctx, b, r)
		if st&StateChecked != 0 {
			fg = g.accent
		}
	default:
		g.softOut(l, ctx, b, r)
	}
	if st&StateHovered != 0 && st&StatePressed == 0 {
		// The only hover this look can afford: a breath of the accent
		// over the face, since a border would break the surface.
		ctx.DrawPath(RoundRectPath(b, r, r, r, r), paintengine2d.Fill(withA(g.accent, 0.07)))
	}
	if st&StateFocused != 0 {
		in := b.Inset(l.S(2))
		ri := max(r-l.S(2), 0)
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Paint{Color: withA(g.accent, 0.9),
			Style:  paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: l.S(2), Join: paintengine2d.JoinRound, MiterLimit: 4}})
	}
	return fg
}

// CheckIndicator is a pressed well; the tick is the accent, because a
// soft surface has no colour of its own to mark with.
func (e neuEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	g := neuColors(l)
	r := neuRadius(l, box.Dy()) * 0.6
	g.softIn(l, ctx, box, r)
	if !checked {
		return
	}
	col := g.accent
	if st&StateDisabled != 0 {
		col = g.dim
	}
	s := box.Dx()
	p := paintengine2d.NewPath()
	p.MoveTo(box.Min.X+s*0.24, box.Min.Y+s*0.52)
	p.LineTo(box.Min.X+s*0.43, box.Min.Y+s*0.72)
	p.LineTo(box.Max.X-s*0.20, box.Min.Y+s*0.29)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: max(s*0.14, l.S(2)),
			Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// RadioIndicator is the same well, round, with a raised dot in it: the
// one place the look puts something *out* of a surface that is already in.
func (e neuEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	g := neuColors(l)
	g.softIn(l, ctx, box, box.Dy()/2)
	if !selected {
		return
	}
	col := g.accent
	if st&StateDisabled != 0 {
		col = g.dim
	}
	d := box.Inset(box.Dx() * 0.3)
	ctx.DrawPath(RoundRectPath(d, d.Dy()/2, d.Dy()/2, d.Dy()/2, d.Dy()/2), paintengine2d.Fill(col))
}

// ViewBackground: a list is part of the surface, not a white well cut
// into it, so it is the window colour at every state.
func (e neuEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	return neuColors(l).surface
}

// ViewFrameInsets / DrawViewFrame: a view is pressed into the surface,
// so its frame is the depth of that press rather than a border.
func (e neuEngine) ViewFrameInsets(l *Classic) Insets {
	d := snap(l.S(neuDepth + 1))
	return Insets{Top: d, Right: d, Bottom: d, Left: d}
}

func (e neuEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	g := neuColors(l)
	g.softIn(l, ctx, b, neuRadius(l, b.Dy()))
}

// GroupBox is a card standing out of the surface.
func (e neuEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	d := snap(l.S(neuDepth + 3))
	top := d
	if hasTitle {
		top = snap(l.S(neuDepth+3) + float32(l.Font().Height()))
	}
	return Insets{Top: top, Right: d, Bottom: d, Left: d}
}

func (e neuEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	g := neuColors(l)
	r := neuRadius(l, b.Dy())
	if raised {
		g.softOut(l, ctx, b, r)
	} else {
		g.softIn(l, ctx, b, r)
	}
	if title == "" {
		return
	}
	f := l.Font()
	l.drawFittedText(ctx, f, title, b.Inset(l.S(neuDepth+8)), g.text, AlignStart, 0)
}

// FieldFocusRing: yes — with no borders anywhere, the ring is the only
// thing that says which field has the keyboard.
func (e neuEngine) FieldFocusRing(l *Classic) bool { return true }

// ScrollBarStyle: a thin overlay with no arrows. Buttons would need
// borders to read, and this look has none.
func (e neuEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Overlay: true, Inset: 3}
}

// PopupShadow is the look's own shadow language, one size larger: a menu
// floats further off the surface than a button stands out of it.
func (e neuEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	d := snap(l.S(neuDepth * 3))
	return Insets{Top: d, Right: d, Bottom: d, Left: d}
}

func (e neuEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	g := neuColors(l)
	r := neuRadius(l, b.Dy())
	// The reach declared by PopupShadow is a promise: nothing may be
	// painted outside b grown by it. The spread and the downward bias
	// therefore share one budget rather than adding up — 70% of the
	// reach goes on spread and 30% on the fall, so the lowest edge of
	// the widest ring lands exactly on the reach and not past it.
	reach := l.S(neuDepth * 3)
	const steps, ringA = 6, 0.09
	for i := steps; i >= 1; i-- {
		t := float32(i) / float32(steps)
		grow := reach * t * 0.7
		box := b.Inset(-grow).Translate(paintengine2d.Pt(0, reach*t*0.3))
		ctx.DrawPath(RoundRectPath(box, r, r, r, r), paintengine2d.Fill(withA(g.shade, ringA)))
	}
}

// MenuHighlight is a pressed well behind the item, not a filled bar.
func (e neuEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	g := neuColors(l)
	ctx.DrawPath(RoundRectPath(b, neuRadius(l, b.Dy())*0.7, neuRadius(l, b.Dy())*0.7,
		neuRadius(l, b.Dy())*0.7, neuRadius(l, b.Dy())*0.7), paintengine2d.Fill(withA(g.accent, 0.14)))
}

func (e neuEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return neuColors(l).text
}

// neuPacks are the two soft-UI packs: the pale grey the style is known
// for, and a dark one. Neither is a recolour of the other — a dark
// neumorphic surface needs a *lighter* highlight and a deeper shadow to
// read as raised at all, which neuColors handles, and it needs its own
// accent because the pale one disappears on it.
func neuPacks() []ThemePack {
	light := Light()
	// #e0e5ec is the surface the original Dribbble work used and the one
	// every tutorial since has copied; the look is tuned for it.
	light.Background = hexColor("#e0e5ec")
	light.Surface = hexColor("#e0e5ec")
	light.SurfaceAlt = hexColor("#e0e5ec")
	light.Field = hexColor("#e0e5ec")
	light.Text = hexColor("#31344b")
	light.TextMuted = hexColor("#9aa0b5")
	light.Accent = hexColor("#5b7cfa")
	light.AccentHover = hexColor("#7490fb")
	light.TextOnAccent = hexColor("#ffffff")
	light.Selection = hexColor("#5b7cfa")
	light.Border = hexColor("#e0e5ec")
	light.FieldBorder = hexColor("#e0e5ec")
	light.MenuHover = hexColor("#d6dbe3")

	dark := Dark()
	dark.Background = hexColor("#2e3239")
	dark.Surface = hexColor("#2e3239")
	dark.SurfaceAlt = hexColor("#2e3239")
	dark.Field = hexColor("#2e3239")
	dark.Text = hexColor("#e6e9f0")
	dark.TextMuted = hexColor("#8a90a0")
	dark.Accent = hexColor("#7d9bff")
	dark.AccentHover = hexColor("#95adff")
	dark.TextOnAccent = hexColor("#16181d")
	dark.Selection = hexColor("#7d9bff")
	dark.Border = hexColor("#2e3239")
	dark.FieldBorder = hexColor("#2e3239")
	dark.MenuHover = hexColor("#373c45")

	mk := func(name, label string, fam ThemeName, pal Palette, summary string) ThemePack {
		p := eraPack(name, label, EraNeumorph, fam, BevelNone, ThemeTokens{
			Palette: pal,
			Metrics: neuEngine{}.DefaultMetrics(),
		})
		p.Tokens.Engine = "neumorphism"
		p.Summary = summary
		// 2019: Alexander Plyuto's "Skeuomorph Mobile Banking" post, the
		// one the name and the whole style came out of.
		p.Year = 2019
		p.Lineage = "Soft UI"
		return p
	}
	return []ThemePack{
		mk("neumorphism", "Neumorphism UI", ThemeLight, light,
			"Soft UI: one surface colour, controls pushed out of it or pressed into it, and nothing but a pair of shadows to say which."),
		mk("neumorphism-night", "Neumorphism UI Night", ThemeDark, dark,
			"The same surface after dark, with a harder highlight and a deeper shadow so a raised control still reads."),
	}
}
