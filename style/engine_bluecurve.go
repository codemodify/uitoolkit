package style

import "github.com/codemodify/paintengine2d"

// bluecurveEngine paints Red Hat's Bluecurve (2002; Red Hat Linux 8 and 9,
// Fedora Core 1–4), the look Clearlooks grew out of. It embeds the
// Clearlooks engine for GTK 2's layout and overrides the shapes that
// differ: square corners everywhere; flat buttons in a strong #5c5c5c
// outline, lit white inside at the top and left and grey at the bottom and
// right (swapped when pressed, a black outline on the default button);
// 13px check boxes whose tick is the blue spot colour; sphere-shaded blue
// radio dots; scroll bar sliders drawn as buttons with three diagonal grip
// lines; a progress bar that runs dark to light blue down its height, with
// no stripes; square bevelled notebook tabs with the others a shade
// darker; menu items in a blue gradient with white text; a flat blue list
// selection.
//
// Its greys are the background scaled in HLS by 1.065, 0.963, 0.896, 0.85,
// 0.768, 0.665, 0.4 and 0.205 (stretched by "contrast"); its spot colours
// are the selection × 1.62, 1.05 and 0.72.
//
// Pack data: params "contrast"; extra "prelight", "active" (bg[PRELIGHT],
// bg[ACTIVE]), "listActive" (base[ACTIVE]), "spot", "tooltip",
// "tooltipText", "caption".
type bluecurveEngine struct{ clearlooksEngine }

func init() {
	RegisterEngine(bluecurveEngine{})
	for _, p := range bluecurvePacks() {
		RegisterPack(p)
	}
}

func (bluecurveEngine) ID() string { return "bluecurve" }

// DefaultMetrics: Clearlooks' GTK 2 geometry with Bluecurve's square
// corners and 13px indicators.
func (bluecurveEngine) DefaultMetrics() ChromeMetrics {
	m := clearlooksEngine{}.DefaultMetrics()
	m.Square, m.Radius, m.RadiusSmall = true, 0, 0
	return m
}

// blFactors is Bluecurve's grey table.
var blFactors = [8]float32{1.065, 0.963, 0.896, 0.85, 0.768, 0.665, 0.4, 0.205}

// blSet is a look's resolved Bluecurve colours.
type blSet struct {
	bg, prelight, active, fg, disFg, base, text paintengine2d.Color
	spot, selFg, listActive                     paintengine2d.Color
	g                                           [8]paintengine2d.Color // gray0..7
	s1, s2, s3                                  paintengine2d.Color    // spot1..3
	menuGrad, progGrad                          []paintengine2d.GradientStop
	dotBody, dotHi, dotLo, white                paintengine2d.Color
	dot                                         []paintengine2d.GradientStop
	radio                                       [3][]paintengine2d.GradientStop // white, bg, active wells
}

type blKey struct{}

func blColors(l *Classic) *blSet {
	return l.Memo(blKey{}, func() any { return blBuild(l) }).(*blSet)
}

func blBuild(l *Classic) *blSet {
	p := l.palette
	c := &blSet{bg: p.Background, fg: p.Text, disFg: p.TextMuted, base: p.Field, white: Hex("#ffffff")}
	c.text = ReadableOn(c.base, 4.5, p.Text)
	c.prelight = l.X("prelight", clShade(c.bg, 1.065))
	c.active = l.X("active", clShade(c.bg, 0.89))
	c.spot = l.X("spot", p.Selection)
	c.spot.A = 1
	c.selFg = ReadableOn(c.spot, 3, p.TextOnAccent, p.Text)
	c.listActive = l.X("listActive", clShade(c.spot, 1.2))
	k := l.P("contrast", 1)
	for i, f := range blFactors {
		c.g[i] = clShade(c.bg, (f-0.7)*k+0.7)
	}
	c.s1, c.s2, c.s3 = clShade(c.spot, 1.62), clShade(c.spot, 1.05), clShade(c.spot, 0.72)
	c.menuGrad = []paintengine2d.GradientStop{Stop(0, clShade(c.spot, 0.9)), Stop(1, clShade(c.spot, 1.2))}
	c.progGrad = []paintengine2d.GradientStop{Stop(0, clShade(c.spot, 0.92)), Stop(1, clShade(c.spot, 1.66))}
	c.dotBody, c.dotHi, c.dotLo = clShade(c.spot, 0.86), clShade(c.spot, 1.3), clShade(c.spot, 0.6)
	c.dot = []paintengine2d.GradientStop{Stop(0, c.dotHi), Stop(0.45, c.dotBody), Stop(1, c.dotLo)}
	for i, fill := range [...]paintengine2d.Color{c.white, c.bg, c.active} {
		c.radio[i] = []paintengine2d.GradientStop{Stop(0, fill), Stop(0.7, fill), Stop(1, Mix(fill, c.g[6], 0.35))}
	}
	return c
}

// bevel paints a Bluecurve box: a 1px outline, then a 1px inner edge (hi
// top and left, lo bottom and right) around the fill. Square.
func (c *blSet) bevel(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, edge, hi, lo, fill paintengine2d.Color) {
	if b.Dx() < 2*lw || b.Dy() < 2*lw {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	in := b.Inset(lw)
	ctx.DrawRect(in, paintengine2d.Fill(fill))
	clLit(ctx, in, 0, lw, hi, lo)
}

// button paints a Bluecurve button face and returns the label colour:
// bg[state] flat in the grey-6 outline, white / grey-2 inside, swapped
// when pressed; the default button's outline is black.
func (c *blSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, flat bool) paintengine2d.Color {
	b = clSnap(b)
	lw := clPx(l)
	dis := st.Disabled()
	down := !dis && (st.Pressed() || (st.Toggle() && st.Checked()))
	hot := !dis && st.Hovered()
	fg := c.fg
	if dis {
		fg = c.disFg
	}
	if flat && !hot && !down {
		return fg
	}
	fill, hi, lo, edge := c.bg, c.white, c.g[2], c.g[6]
	switch {
	case dis:
		edge = c.g[5]
	case down:
		fill, hi, lo = c.active, c.g[2], c.white
	case hot:
		fill = c.prelight
	}
	if st.Primary() && !flat && !dis {
		edge = Hex("#000000")
	}
	c.bevel(ctx, b, lw, edge, hi, lo, fill)
	return fg
}

func (e bluecurveEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := blColors(l)
	lw := clPx(l)
	switch role {
	case RoleButton, RoleCombo:
		return c.button(l, ctx, b, st, false)
	case RoleTool:
		return c.button(l, ctx, b, st, true)
	case RoleField:
		// An entry: grey-5 border, a grey-1 line inside at the top and
		// left, the base colour.
		b = clSnap(b)
		fill := c.base
		if st.Disabled() {
			fill = c.bg
		}
		c.bevel(ctx, b, lw, c.g[5], c.g[1], paintengine2d.Color{}, fill)
		if st.Disabled() {
			return c.disFg
		}
		return c.text
	case RoleRow:
		if !st.Checked() {
			return c.text
		}
		if st.Backdrop() {
			ctx.DrawRect(b, paintengine2d.Fill(c.listActive))
		} else {
			ctx.DrawRect(b, paintengine2d.Fill(c.spot))
		}
		return c.selFg
	case RoleThumb:
		c.button(l, ctx, b, st, false)
		return c.fg
	case RoleTrack:
		c.trough(l, ctx, b)
		return c.fg
	}
	return e.clearlooksEngine.Face(l, ctx, b, role, st)
}

// trough is a scroll bar or progress trough: grey 3 in a grey-5 border.
func (c *blSet) trough(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = clSnap(b)
	lw := clPx(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.g[5]))
	if b.Dx() > 2*lw && b.Dy() > 2*lw {
		ctx.DrawRect(b.Inset(lw), paintengine2d.Fill(c.g[3]))
	}
}

// CheckIndicator is the 13px Bluecurve check box: square, a grey-6 frame
// around white (its first row and column greyed for depth), and the tick
// in the blue spot colour.
func (e bluecurveEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := blColors(l)
	lw := clPx(l)
	s := snap(min(box.Dx(), box.Dy()))
	if s < 5*lw {
		return
	}
	b := clSnap(adwCentered(box, s, s))
	dis := st.Disabled()
	fill := c.white
	switch {
	case dis:
		fill = c.bg
	case st.Pressed():
		fill = c.active
	}
	edge := c.g[6]
	if dis {
		edge = c.g[5]
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	in := b.Inset(lw)
	ctx.DrawRect(in, paintengine2d.Fill(fill))
	if !dis {
		sh := Mix(edge, c.white, 0.79)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(sh))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, lw, in.Dy()-lw), paintengine2d.Fill(sh))
	}
	if checked {
		col := c.spot
		if dis {
			col = c.disFg
		}
		clTick(ctx, b, col, max(s*0.2, 1.5*lw))
	}
}

// RadioIndicator is the Bluecurve radio: a grey-6 disc around a white one
// shaded at its upper left, and a 7px blue sphere for the dot.
func (e bluecurveEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := blColors(l)
	lw := clPx(l)
	s := min(box.Dx(), box.Dy())
	if s < 5*lw {
		return
	}
	ctr := box.Center()
	R := s * 0.5
	dis := st.Disabled()
	well := c.radio[0]
	switch {
	case dis:
		well = c.radio[1]
	case st.Pressed():
		well = c.radio[2]
	}
	edge := c.g[6]
	if dis {
		edge = c.g[5]
	}
	ctx.DrawCircle(ctr, R, paintengine2d.Fill(edge))
	inner := R - lw
	ctx.DrawCircle(ctr, inner, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(ctr.X+inner*0.3, ctr.Y+inner*0.3), Radius: inner * 1.3,
		Stops: well}))
	if !selected {
		return
	}
	d := R * 7 / 13
	if dis {
		ctx.DrawCircle(ctr, d, paintengine2d.Fill(c.disFg))
		return
	}
	ctx.DrawCircle(ctr, d, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(ctr.X-d*0.35, ctr.Y-d*0.35), Radius: d * 1.35,
		Stops: c.dot}))
}

// Arrow is Bluecurve's: a solid triangle, hollowed into a chevron with
// 3px arms once it is 7px or wider.
func (bluecurveEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	lw := clPx(l)
	w := min(b.Dx(), b.Dy())
	w = min(snap(w*0.6), snap(l.S(9)))
	if w < 7*lw {
		clTriangle(ctx, b, dir, col, w)
		return
	}
	h := w * 0.5
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-h, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
		p.LineTo(cx+h, cy+h*0.5)
	case DirDown:
		p.MoveTo(cx-h, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
		p.LineTo(cx+h, cy-h*0.5)
	case DirLeft:
		p.MoveTo(cx+h*0.5, cy-h)
		p.LineTo(cx-h*0.5, cy)
		p.LineTo(cx+h*0.5, cy+h)
	default:
		p.MoveTo(cx-h*0.5, cy-h)
		p.LineTo(cx+h*0.5, cy)
		p.LineTo(cx-h*0.5, cy+h)
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: 2 * lw, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// MenuHighlight is a Bluecurve menu item: the blue running lighter down,
// in a spot-3 outline with a spot-1 / spot-2 inner bevel.
func (bluecurveEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	in := b.Inset(lw)
	if attachBottom {
		in.Max.Y = b.Max.Y
	}
	ctx.DrawRect(in, VGradient(in, c.menuGrad...))
	clLit(ctx, in, 0, lw, c.s1, c.s2)
}

func (bluecurveEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := blColors(l)
	if hot {
		return ReadableOn(c.menuGrad[1].Color, 3, c.white, c.fg)
	}
	return c.fg
}

// DrawFocusRing is Bluecurve's 1px dotted rectangle in grey 6.
func (bluecurveEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	clDotted(ctx, b, clPx(l), blColors(l).g[6])
}

// DrawScrollBarParts: a grey-3 trough, steppers and a slider drawn as
// buttons, three 45° grip lines (grey 5 over white) on the slider.
func (e bluecurveEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := blColors(l)
	lw := clPx(l)
	if p.Bar.Empty() {
		return
	}
	ctx.DrawRect(p.Bar, paintengine2d.Fill(c.bg))
	tr := p.Track
	if vertical {
		tr = paintengine2d.XYWH(p.Bar.Min.X, p.Track.Min.Y-lw, p.Bar.Dx(), p.Track.Dy()+2*lw)
	} else {
		tr = paintengine2d.XYWH(p.Track.Min.X-lw, p.Bar.Min.Y, p.Track.Dx()+2*lw, p.Bar.Dy())
	}
	c.trough(l, ctx, tr.Intersect(p.Bar))
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		ps := st.Part(part)
		c.button(l, ctx, b, ps, false)
		col := c.g[7]
		if st.Disabled {
			col = c.g[5]
		}
		g := b
		if ps.Pressed() {
			g = g.Translate(paintengine2d.Pt(lw, lw))
		}
		l.Engine().Arrow(l, ctx, g, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if p.Thumb.Empty() {
		return
	}
	th := p.Thumb
	// At the ends of its travel the slider overlaps the stepper's outline.
	if vertical {
		if th.Min.Y <= p.Track.Min.Y+0.5 && !p.Dec.Empty() {
			th.Min.Y -= lw
		}
		if th.Max.Y >= p.Track.Max.Y-0.5 && !p.Inc.Empty() {
			th.Max.Y += lw
		}
	} else {
		if th.Min.X <= p.Track.Min.X+0.5 && !p.Dec.Empty() {
			th.Min.X -= lw
		}
		if th.Max.X >= p.Track.Max.X-0.5 && !p.Inc.Empty() {
			th.Max.X += lw
		}
	}
	th = th.Intersect(p.Bar)
	c.button(l, ctx, th, st.Part(ScrollThumbPart), false)
	if !st.Disabled {
		c.diagGrip(l, ctx, th.Inset(2*lw), vertical)
	}
}

// diagGrip is Bluecurve's grip: three 45° lines, 6px long and 5px apart,
// grey 5 with a white twin a pixel below, batched into two paths.
func (c *blSet) diagGrip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	lw := clPx(l)
	n := snap(l.S(6))
	gap := snap(l.S(5))
	long := b.Dy()
	if !vertical {
		long = b.Dx()
	}
	if long < 3*gap || min(b.Dx(), b.Dy()) < n*0.7 {
		return
	}
	ctr := b.Center()
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for i := -1; i <= 1; i++ {
		o := float32(i) * gap
		cx, cy := ctr.X, ctr.Y+o
		if !vertical {
			cx, cy = ctr.X+o, ctr.Y
		}
		h := n * 0.5
		for _, s := range [...]struct {
			p  *paintengine2d.Path
			dy float32
		}{{dk, 0}, {lt, lw}} {
			s.p.MoveTo(cx-h, cy+h+s.dy)
			s.p.LineTo(cx+h, cy-h+s.dy)
		}
	}
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawPath(lt, paintengine2d.StrokePaint(c.white, lw))
	ctx.DrawPath(dk, paintengine2d.StrokePaint(c.g[5], lw))
	ctx.Restore()
}

func (e bluecurveEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := blColors(l)
	c.trough(l, ctx, track)
	if !thumb.Empty() {
		c.button(l, ctx, thumb, st, false)
		c.diagGrip(l, ctx, clSnap(thumb).Inset(2*clPx(l)), thumb.Dy() >= thumb.Dx())
	}
}

// DrawProgressBar: the scroll trough, and a bar that runs from the spot
// colour darkened at the top to a light blue at the bottom, in a spot-2
// outline with a spot-1 / spot-3 bevel inside. No stripes.
func (e bluecurveEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 6*lw || b.Dy() < 5*lw {
		return
	}
	c.trough(l, ctx, b)
	tr := b.Inset(lw)
	var fill paintengine2d.Rect
	if indeterminate {
		fill = clPulse(tr, phase)
	} else {
		fill = paintengine2d.XYWH(tr.Min.X, tr.Min.Y, snap(tr.Dx()*clamp1(t)), tr.Dy())
	}
	fill = clSnap(fill)
	if fill.Dx() < 3*lw {
		return
	}
	if st.Disabled() {
		ctx.DrawRect(fill, paintengine2d.Fill(Mix(c.g[3], c.spot, 0.3)))
		return
	}
	ctx.DrawRect(fill, paintengine2d.Fill(c.s2))
	in := fill.Inset(lw)
	ctx.DrawRect(in, VGradient(in, c.progGrad...))
	clLit(ctx, in, 0, lw, c.s1, c.s3)
}

// DrawSlider is a Bluecurve scale: a 5px trough (grey 3, grey-5 border,
// grey 4 inside at the top) and a button knob with its corners cut and a
// grip of three diagonals.
func (e bluecurveEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := blColors(l)
	lw := clPx(l)
	t = clamp1(t)
	kl := min(snap(l.S(31)), snap(b.Dx()*0.4))
	kw := snap(min(l.S(15), b.Dy()-2*lw))
	if kw < 6*lw || kl < 6*lw {
		return
	}
	dis := st.Disabled()
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	th := snap(l.S(5))
	trough := clSnap(paintengine2d.XYWH(b.Min.X+lw, cy-th*0.5, b.Dx()-2*lw, th))
	c.bevel(ctx, trough, lw, c.g[5], c.g[4], paintengine2d.Color{}, c.g[3])
	// The knob's left edge, kept inside b.
	knob := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-kl)*t), snap(cy-kw*0.5), kl, kw)
	ks := StateNone
	switch {
	case dis:
		ks = StateDisabled
	case st.Pressed():
		ks = StatePressed
	case st.Hovered():
		ks = StateHovered
	}
	ctx.Save()
	r := min(2*lw, knob.Dy()*0.5)
	ctx.ClipRoundRect(knob, r, r)
	c.button(l, ctx, knob, ks, false)
	ctx.Restore()
	if !dis {
		c.diagGrip(l, ctx, knob.Inset(2*lw), false)
	}
	if st.Focused() && !dis {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

// DrawSpinner: two stacked Bluecurve buttons.
func (e bluecurveEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 5*lw || b.Dy() < 8*lw {
		return
	}
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(h paintengine2d.Rect, dir Direction, hover, press bool) {
		s := StateNone
		switch {
		case st.Disabled():
			s = StateDisabled
		case press:
			s = StatePressed
		case hover:
			s = StateHovered
		}
		fg := c.button(l, ctx, h, s, false)
		g := h
		if press {
			g = g.Translate(paintengine2d.Pt(lw, lw))
		}
		if !st.Disabled() {
			fg = c.g[7]
		}
		l.Engine().Arrow(l, ctx, g, dir, fg)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

// DrawComboBox is a Bluecurve option menu: a button with the label, a
// grey-3 / white divider and the chevron over a small grey-3 bar.
func (e bluecurveEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	bs := st &^ (StatePrimary | StateFocused)
	if open {
		bs |= StatePressed
	}
	fg := c.button(l, ctx, b, bs, false)
	aw := snap(l.S(22))
	ax := snap(b.Max.X - aw)
	if b.Dy() > 10*lw {
		sep := paintengine2d.XYWH(ax, b.Min.Y+snap(l.S(5)), lw, b.Dy()-2*snap(l.S(5)))
		ctx.DrawRect(sep, paintengine2d.Fill(c.g[3]))
		ctx.DrawRect(sep.Translate(paintengine2d.Pt(lw, 0)), paintengine2d.Fill(c.white))
	}
	ab := paintengine2d.XYWH(ax+lw, b.Min.Y, aw-lw, b.Dy())
	col := c.g[7]
	if st.Disabled() {
		col = c.g[5]
	}
	up := paintengine2d.XYWH(ab.Min.X, ab.Min.Y, ab.Dx(), ab.Dy()-snap(l.S(4)))
	l.Engine().Arrow(l, ctx, up, DirDown, col)
	bw, bh := snap(l.S(7)), max(snap(l.S(2)), lw)
	ctx.DrawRect(paintengine2d.XYWH(snap(ab.Center().X-bw*0.5), snap(up.Center().Y+l.S(5)), bw, bh), paintengine2d.Fill(c.g[3]))
	lb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, ax-b.Min.X-l.S(10), b.Dy())
	if open {
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	}
	clEmboss(l, ctx, l.body, text, lb, fg, AlignStart, 0, st.Disabled())
	if st.Focused() && !open && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ax-b.Min.X, b.Dy()).Inset(snap(l.S(4))))
	}
}

// DrawTabBar: the background with the page frame's top edge (grey 6 over
// its white inner line).
func (e bluecurveEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*lw, b.Dx(), lw), paintengine2d.Fill(c.g[6]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.white))
}

// DrawTab is a square bevelled tab: the current one the background colour
// and open into the page, the others 2px lower in bg[ACTIVE].
func (e bluecurveEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	t := b
	fill := c.bg
	if !selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+snap(l.S(2)), b.Dx(), b.Dy()-snap(l.S(2))-2*lw)
		fill = c.active
		if st.Hovered() && !st.Disabled() {
			fill = Mix(c.active, c.bg, 0.5)
		}
	}
	if t.Dx() < 6*lw || t.Dy() < 6*lw {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx(), t.Dy()), paintengine2d.Fill(c.g[6]))
	in := paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+lw, t.Dx()-2*lw, t.Dy()-lw)
	ctx.DrawRect(in, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.white))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.white))
	ctx.DrawRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y+lw, lw, in.Dy()-lw), paintengine2d.Fill(c.g[2]))
	if selected {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X+lw, b.Max.Y-2*lw, in.Dx()-2*lw, 2*lw), paintengine2d.Fill(fill))
	}
	fg := c.fg
	if st.Disabled() {
		fg = c.disFg
	}
	lb := paintengine2d.XYWH(t.Min.X, t.Min.Y+lw, t.Dx(), t.Dy()-lw)
	clEmboss(l, ctx, l.body, label, lb, fg, AlignCenter, l.S(10), st.Disabled())
	if st.Focused() && selected {
		f := l.body
		w := min(f.Advance(label)+l.S(6), lb.Dx()-l.S(6))
		h := f.Height() + l.S(2)
		l.Engine().DrawFocusRing(l, ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h))
	}
}

// DrawTabPane is the notebook frame: grey 6, white / grey-2 inside.
func (e bluecurveEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := blColors(l)
	c.bevel(ctx, clSnap(b), clPx(l), c.g[6], c.white, c.g[2], c.bg)
}

// DrawTableHeader is a column header drawn as a Bluecurve button.
func (e bluecurveEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := blColors(l)
	lw := clPx(l)
	fg := c.button(l, ctx, b, st&^(StatePrimary|StateFocused), false)
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		l.Engine().Arrow(l, ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(4), lb.Min.Y, aw, lb.Dy()), dir, c.g[7])
	}
	clEmboss(l, ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()), fg, AlignStart, 0, st.Disabled())
}

// DrawMenuFrame is a Bluecurve menu: grey-5 outline, white / grey-2 bevel.
func (e bluecurveEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := blColors(l)
	c.bevel(ctx, clSnap(b), clPx(l), c.g[5], c.white, c.g[2], c.bg)
}

// DrawMenuBar is flat, with one grey-3 line along the bottom.
func (e bluecurveEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.g[3]))
}

// DrawToolBar is flat between a grey-0 top line and a grey-3 bottom line.
func (e bluecurveEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := blColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.g[0]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.g[3]))
}

// ItemFocus: the dotted rectangle, in the selected text colour on a
// selection.
func (e bluecurveEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := blColors(l)
	col := c.g[6]
	if st.Checked() {
		col = c.selFg.WithAlpha(0.75)
	}
	clDotted(ctx, b, clPx(l), col)
}

// DrawViewFrame: a scrolled window's 1px grey-5 frame.
func (e bluecurveEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := blColors(l)
	in := e.ViewFrameInsets(l)
	if in.Zero() || b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.g[5]))
	ctx.DrawRect(in.Apply(b), paintengine2d.Fill(c.base))
}

func bluecurvePacks() []ThemePack {
	// redhat-artwork's Bluecurve gtkrc: #e6e6e6, the #4464ac selection.
	pal := clPalette("#e6e6e6", "#000000", "#ffffff", "#000000", "#4464ac", "#ffffff", "#777777", "#4464ac")
	pal.Border, pal.Divider = Hex("#5c5c5c"), Hex("#c4c4c4")
	pal.MenuHover, pal.MenuHoverBorder = Hex("#4464ac"), Hex("#3b4c71")
	p := clPack("bluecurve", "Bluecurve", 2002, "Red Hat",
		"Red Hat Linux 8 and Fedora Core: square bevelled greys, blue ticks and menus, diagonal grips.", pal, clClassic,
		map[string]string{
			"prelight": "#f5f5f5", "active": "#cccccc", "listActive": "#5e7ab7",
			"menu": "#e6e6e6", "notebook": "#e6e6e6", "check": "#4464ac",
			"tooltip": "#ffffbf", "tooltipText": "#000000",
		}, ChromeMetrics{})
	p.Tokens.Engine = "bluecurve"
	p.Era = "Bluecurve"
	return []ThemePack{p}
}
