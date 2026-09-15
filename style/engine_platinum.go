package style

import "github.com/codemodify/paintengine2d"

// platinumEngine paints Mac OS 8 / 9 "Platinum" (1997): #dddddd faces,
// black one-pixel outlines with a white top-left highlight and a two-step
// shadow, softly rounded push buttons with a thick default ring, bevelled
// check boxes whose tick overshoots the box, sunken radio beads with a
// black dot, 16px scroll bars with an accent-coloured grip thumb, folder
// tabs with slanted sides, title bars of raised ridges around a bold
// centred title with the close box on the left, and the accent colour
// (Lavender by default) on menus, thumbs, progress bars and focus frames.
//
// Colours were sampled from Mac OS 8.0 / 9.0 screenshots at 1:1 and the
// figures of the Mac OS 8 Human Interface Guidelines: window #dddddd,
// title bar #cccccc, pane #eeeeee, text highlight #ccccff, menu highlight
// #333399 under a #6666cc top line, focus frame two pixels of #6666cc.
//
// Pack data ("extra" colours, derived from Palette.Accent when omitted):
//
//	accentDeep, accentDark, accentLight, accentPale, accentHi
//	    the seven-step accent ramp around Palette.Accent (thumbs, tubes,
//	    menu highlight, disclosure triangles)
//	pane        tab pane and selected tab face
//	title       title bar and window frame face
//	hi, lo, shadow, line   bevel highlight, inner and outer shadow, outline
//	press       pressed button face
//	track       scroll bar trough
//	menuHiTop   top line of the menu highlight
//	info        balloon help fill
type platinumEngine struct{ BaseEngine }

func init() {
	RegisterEngine(platinumEngine{})
	for _, p := range platinumPacks() {
		RegisterPack(p)
	}
}

func (platinumEngine) ID() string { return "platinum" }

// DefaultMetrics are Platinum's proportions at the toolkit's 16px UI font
// (Mac OS 8 used 12pt Charcoal): 20px buttons become 28px controls,
// 16px scroll bars stay 16px.
func (platinumEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Radius:    4, RadiusSmall: 3,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 13, Radio: 13,
		MenuItemH: 24, MenuBarH: 24, TabH: 28, RowH: 22,
		TitleBar: 26, HeaderH: 22, ProgressH: 16, SliderH: 26, Thumb: 13,
		Scroll: 16, Pad: 10, FieldPad: 6, FocusWidth: 2, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 15, SwitchW: 40, SwitchH: 20,
	}
}

// ---- colours ------------------------------------------------------------------

// plat is the resolved colour set of a look.
type plat struct {
	face, pane, title, hi, lo, shadow, black paintengine2d.Color
	text, dim, field, off                    paintengine2d.Color
	deep, dark, acc, light, pale, accHi      paintengine2d.Color
	sel, selTxt, hover                       paintengine2d.Color
	menuTxt, menuHiTxt, menuHiTop            paintengine2d.Color
	press, pressTxt, track, trackOff         paintengine2d.Color
	info, infoTxt, focus                     paintengine2d.Color
	tube, groove, greyTube                   []paintengine2d.GradientStop
	pole, poleShade, sphere, sunk, closeBox  []paintengine2d.GradientStop
	defRing                                  []paintengine2d.GradientStop
	ridge                                    paintengine2d.Color
}

type platKey struct{}

// platColors is the look's resolved colour set (built once per look).
func platColors(l *Classic) *plat {
	return l.Memo(platKey{}, func() any { return platBuild(l) }).(*plat)
}

func platBuild(l *Classic) *plat {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	acc := p.Accent
	c := &plat{
		face:   p.Background,
		pane:   l.X("pane", Shade(p.Background, 0.45)),
		title:  l.X("title", p.SurfaceAlt),
		hi:     l.X("hi", white),
		lo:     l.X("lo", Shade(p.Background, -0.23)),
		shadow: l.X("shadow", Shade(p.Background, -0.46)),
		black:  l.X("line", black),
		text:   p.Text,
		dim:    p.TextMuted,
		field:  p.Field,
		deep:   l.X("accentDeep", Shade(acc, -0.66)),
		dark:   l.X("accentDark", Shade(acc, -0.4)),
		acc:    acc,
		light:  l.X("accentLight", Shade(acc, 0.4)),
		pale:   l.X("accentPale", Shade(acc, 0.7)),
		accHi:  l.X("accentHi", Shade(acc, 0.86)),
		sel:    p.Selection,
		focus:  p.Focus,
		info:   l.X("info", white),
	}
	c.off = Mix(c.shadow, c.face, 0.3)
	c.selTxt = ReadableOn(c.sel, 4.5, c.text, white)
	c.hover = c.sel.WithAlpha(0.35)
	c.menuTxt = ReadableOn(c.face, 4.5, c.text)
	c.menuHiTop = l.X("menuHiTop", c.acc)
	c.menuHiTxt = ReadableOn(c.dark, 4.5, white, black)
	c.press = l.X("press", Shade(c.face, -0.58))
	c.pressTxt = ReadableOn(c.press, 4.5, white, black)
	c.track = l.X("track", Shade(c.face, -0.23))
	c.trackOff = Shade(c.face, 0.45)
	c.infoTxt = ReadableOn(c.info, 4.5, c.text, black)
	// The accent tube of progress bars and the grey groove next to it.
	c.tube = []paintengine2d.GradientStop{
		Stop(0, c.dark), Stop(0.14, c.acc), Stop(0.28, c.light), Stop(0.4, c.pale),
		Stop(0.48, c.accHi), Stop(0.58, c.pale), Stop(0.7, c.light), Stop(0.82, c.acc), Stop(0.92, c.dark), Stop(1, c.deep),
	}
	c.greyTube = []paintengine2d.GradientStop{
		Stop(0, Shade(c.face, -0.62)), Stop(0.14, c.shadow), Stop(0.3, c.lo), Stop(0.48, c.hi),
		Stop(0.62, c.face), Stop(0.8, c.lo), Stop(1, c.shadow),
	}
	c.groove = []paintengine2d.GradientStop{Stop(0, c.shadow), Stop(0.15, c.lo), Stop(0.85, c.lo), Stop(1, c.face)}
	c.pole = []paintengine2d.GradientStop{Stop(0, c.acc), Stop(0.5, c.acc), Stop(0.5, c.lo), Stop(1, c.lo)}
	c.poleShade = []paintengine2d.GradientStop{
		Stop(0, black.WithAlpha(0.5)), Stop(0.16, black.WithAlpha(0.12)), Stop(0.4, white.WithAlpha(0.55)),
		Stop(0.58, white.WithAlpha(0.08)), Stop(0.84, black.WithAlpha(0.18)), Stop(1, black.WithAlpha(0.55)),
	}
	c.sphere = []paintengine2d.GradientStop{Stop(0, c.hi), Stop(0.45, c.face), Stop(1, c.shadow)}
	c.sunk = []paintengine2d.GradientStop{Stop(0, Shade(c.face, -0.62)), Stop(0.5, c.shadow), Stop(1, c.face)}
	c.closeBox = []paintengine2d.GradientStop{Stop(0, Shade(c.title, -0.3)), Stop(1, c.hi)}
	c.defRing = []paintengine2d.GradientStop{Stop(0, c.face), Stop(1, c.shadow)}
	c.ridge = Shade(c.title, -0.42)
	return c
}

// platU is one design pixel at the look's scale (never under a device pixel).
func platU(l *Classic) float32 {
	u := l.S(1)
	if u < 1 {
		u = 1
	}
	return u
}

// ---- painting helpers -----------------------------------------------------------

// bevel is the Platinum raised edge inside b: hi along the top and left,
// the two-step shadow (lo inside, shadow outside) along the bottom and
// right, keeping clear of rounded corners of radius r.
func (c *plat) bevel(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32, hi, lo, shadow paintengine2d.Color) {
	if b.Dx() < u*4 || b.Dy() < u*4 {
		return
	}
	k := r * 0.6
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+k, b.Min.Y, b.Dx()-k*2-u, u), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+k, u, b.Dy()-k*2-u), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+k, b.Max.Y-u, b.Dx()-k*2, u), paintengine2d.Fill(shadow))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+k, u, b.Dy()-k*2), paintengine2d.Fill(shadow))
	if !colorUnset(lo) {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+k, b.Max.Y-u*2, b.Dx()-k*2-u, u), paintengine2d.Fill(lo))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u*2, b.Min.Y+k, u, b.Dy()-k*2-u), paintengine2d.Fill(lo))
	}
}

// button paints a Platinum push-button body into b: black rounded outline,
// face and bevel; pressed buttons go dark grey.
func (c *plat) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, pressed, disabled bool) {
	u := platU(l)
	r := l.S(4)
	edge := c.black
	if disabled {
		edge = c.off
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(edge))
	in := b.Inset(u)
	ri := r - u
	face := c.face
	if pressed && !disabled {
		face = c.press
	}
	ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(face))
	switch {
	case disabled:
	case pressed:
		c.bevel(ctx, in, ri, u, Shade(c.press, -0.3), paintengine2d.Color{}, Shade(c.press, 0.25))
	default:
		c.bevel(ctx, in, ri, u, c.hi, c.lo, c.shadow)
	}
}

// ring is the focus frame: two pixels of the accent, just inside b.
func (c *plat) ring(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32) {
	if b.Dx() < u*5 || b.Dy() < u*5 {
		return
	}
	ctx.DrawRoundRect(b.Inset(u), r, r, paintengine2d.StrokePaint(c.focus, u*2))
}

// ridges are the raised lines of an active Platinum title bar: n white
// lines, each over a dark line shifted one pixel down and right.
func (c *plat) ridges(ctx *paintengine2d.Context, x0, x1, cy, u float32, n int) {
	if x1-x0 < u*4 {
		return
	}
	y := snap(cy - float32(n)*u)
	for i := 0; i < n; i++ {
		yy := y + float32(i)*u*2
		ctx.DrawRect(paintengine2d.XYWH(x0, yy, x1-x0-u, u), paintengine2d.Fill(c.hi))
		ctx.DrawRect(paintengine2d.XYWH(x0+u, yy+u, x1-x0-u, u), paintengine2d.Fill(c.ridge))
	}
}

// thumb is the accent-coloured scroll thumb with its grip lines.
func (c *plat) thumb(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, vertical, pressed bool) {
	ctx.DrawRect(b, paintengine2d.Fill(c.black))
	in := b.Inset(u)
	if in.Empty() {
		return
	}
	face := c.light
	if pressed {
		face = c.acc
	}
	ctx.DrawRect(in, paintengine2d.Fill(face))
	Edge(ctx, in, c.pale, c.acc)
	if pressed {
		Edge(ctx, in, c.acc, c.light)
	}
	// Grip: four ridges across the thumb, centred.
	n := 4
	if vertical {
		if in.Dy() < u*12 {
			return
		}
		w := in.Dx() - u*6
		y := snap(in.Min.Y + in.Dy()*0.5 - u*4)
		for i := 0; i < n; i++ {
			yy := y + float32(i)*u*2
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u*3, yy, w, u), paintengine2d.Fill(c.pale))
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u*3, yy+u, w, u), paintengine2d.Fill(c.dark))
		}
		return
	}
	if in.Dx() < u*12 {
		return
	}
	h := in.Dy() - u*6
	x := snap(in.Min.X + in.Dx()*0.5 - u*4)
	for i := 0; i < n; i++ {
		xx := x + float32(i)*u*2
		ctx.DrawRect(paintengine2d.XYWH(xx, in.Min.Y+u*3, u, h), paintengine2d.Fill(c.pale))
		ctx.DrawRect(paintengine2d.XYWH(xx+u, in.Min.Y+u*3, u, h), paintengine2d.Fill(c.dark))
	}
}

// platTri fills an isosceles triangle of base w and height h centred on
// (cx, cy), pointing dir; outline strokes it too when set.
func platTri(ctx *paintengine2d.Context, cx, cy, w, h float32, dir Direction, fill, outline paintengine2d.Color, u float32) {
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-w*0.5, cy+h*0.5)
		p.LineTo(cx+w*0.5, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
	case DirDown:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
	case DirLeft:
		p.MoveTo(cx+h*0.5, cy-w*0.5)
		p.LineTo(cx+h*0.5, cy+w*0.5)
		p.LineTo(cx-h*0.5, cy)
	default:
		p.MoveTo(cx-h*0.5, cy-w*0.5)
		p.LineTo(cx-h*0.5, cy+w*0.5)
		p.LineTo(cx+h*0.5, cy)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(fill))
	if !colorUnset(outline) {
		ctx.DrawPath(p, paintengine2d.Paint{Color: outline, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: u, Join: paintengine2d.JoinMiter, MiterLimit: 8}})
	}
}

// etched is the Platinum separator groove: a grey line over a white one.
func (c *plat) etched(ctx *paintengine2d.Context, x, y, n, u float32, vertical bool) {
	if vertical {
		ctx.DrawRect(paintengine2d.XYWH(x, y, u, n), paintengine2d.Fill(c.shadow))
		ctx.DrawRect(paintengine2d.XYWH(x+u, y, u, n), paintengine2d.Fill(c.hi))
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(x, y, n, u), paintengine2d.Fill(c.shadow))
	ctx.DrawRect(paintengine2d.XYWH(x, y+u, n, u), paintengine2d.Fill(c.hi))
}

// ---- parts --------------------------------------------------------------------

func (e platinumEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := platColors(l)
	u := platU(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	switch role {
	case RoleButton, RoleCombo:
		pressed := st.Pressed() && !st.Disabled()
		c.button(l, ctx, b, pressed, st.Disabled())
		if pressed {
			return c.pressTxt
		}
		return fg
	case RoleTool:
		// Bevel buttons: flat until the pointer arrives, sunken when held.
		switch {
		case st.Disabled():
		case st.Pressed():
			ctx.DrawRect(b, paintengine2d.Fill(c.lo))
			Edge(ctx, b, c.shadow, c.hi)
		case st.Checked():
			ctx.DrawRect(b, paintengine2d.Fill(Mix(c.lo, c.face, 0.4)))
			Edge(ctx, b, c.shadow, c.hi)
		case st.Hovered():
			Edge(ctx, b, c.hi, c.shadow)
		}
		return fg
	case RoleField:
		fill := c.field
		if st.Disabled() {
			fill = c.face
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		Edge(ctx, b, c.shadow, c.hi)
		ctx.DrawRect(b.Inset(u*1.5), paintengine2d.StrokePaint(c.black, u))
		return fg
	case RoleCheck:
		face := c.face
		if st.Pressed() && !st.Disabled() {
			face = c.press
		}
		edge := c.black
		if st.Disabled() {
			edge = c.off
		}
		ctx.DrawRect(b, paintengine2d.Fill(edge))
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(face))
		if !st.Disabled() && !st.Pressed() {
			Edge(ctx, b.Inset(u), c.hi, c.lo)
		}
		return fg
	case RoleRow:
		if st.Checked() && st.Inactive() {
			// Classic Mac OS draws the selection of an inactive list as a
			// hollow frame in the highlight colour.
			u := l.S(1)
			ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.sel, u))
			return fg
		}
		if st.Checked() {
			ctx.DrawRect(b, paintengine2d.Fill(c.sel))
			return c.selTxt
		}
		if st.Hovered() && !st.Disabled() {
			ctx.DrawRect(b, paintengine2d.Fill(c.hover))
		}
		return fg
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			e.MenuHighlight(l, ctx, b, false)
			return c.menuHiTxt
		}
		return c.menuTxt
	case RoleThumb:
		c.thumb(ctx, b, u, b.Dy() > b.Dx(), st.Pressed())
		return fg
	case RoleTrack:
		ctx.DrawRect(b, paintengine2d.Fill(c.track))
		Edge(ctx, b, c.shadow, Shade(c.track, 0.3))
		return fg
	case RoleTab:
		return fg // DrawTab paints the tab
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return fg
	}
	return fg
}

func (e platinumEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := platColors(l)
	u := platU(l)
	s := snap(box.Dx())
	box = paintengine2d.XYWH(snap(box.Min.X), snap(box.Min.Y), s, s)
	e.Face(l, ctx, box, RoleCheck, st)
	if !checked {
		return
	}
	// The Platinum tick: bold, with a grey drop shadow, overshooting the
	// top right corner of the box.
	tick := func(dx, dy float32, col paintengine2d.Color) {
		p := paintengine2d.NewPath()
		p.MoveTo(box.Min.X+s*0.22+dx, box.Min.Y+s*0.48+dy)
		p.LineTo(box.Min.X+s*0.45+dx, box.Min.Y+s*0.76+dy)
		p.LineTo(box.Min.X+s*1.05+dx, box.Min.Y-s*0.1+dy)
		ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: s * 0.16, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 6}})
	}
	col := c.text
	switch {
	case st.Disabled():
		col = c.off
	case st.Pressed():
		col = c.pressTxt
	default:
		tick(u, u, c.shadow)
	}
	tick(0, 0, col)
}

func (e platinumEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := platColors(l)
	u := platU(l)
	d := box.Dx()
	if box.Dy() < d {
		d = box.Dy()
	}
	d = snap(d)
	cx, cy := snap(box.Min.X+(box.Dx()-d)*0.5)+d*0.5, snap(box.Min.Y+(box.Dy()-d)*0.5)+d*0.5
	ctr := paintengine2d.Pt(cx, cy)
	r := d * 0.5
	edge := c.black
	if st.Disabled() {
		edge = c.off
	}
	ctx.DrawCircle(ctr, r, paintengine2d.Fill(edge))
	in := r - u
	switch {
	case st.Disabled():
		ctx.DrawCircle(ctr, in, paintengine2d.Fill(c.face))
	case st.Pressed():
		ctx.DrawCircle(ctr, in, paintengine2d.Fill(c.press))
	case selected:
		// Sunken bead: dark at the top, light at the bottom.
		ctx.DrawCircle(ctr, in, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(cx, cy-in), End: paintengine2d.Pt(cx, cy+in), Stops: c.sunk}))
	default:
		// Raised bead: a sphere lit from the top left.
		ctx.DrawCircle(ctr, in, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: paintengine2d.Pt(cx-in*0.35, cy-in*0.35), Radius: in * 1.45, Stops: c.sphere}))
	}
	if selected {
		dot := c.text
		if st.Disabled() {
			dot = c.off
		} else if st.Pressed() {
			dot = c.pressTxt
		}
		ctx.DrawCircle(ctr, r*0.42, paintengine2d.Fill(dot))
	}
}

func (platinumEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	// Platinum arrows are solid black, 9×5 in a 16px button.
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	w := s * 0.56
	if w > l.S(9) {
		w = l.S(9)
	}
	if w < l.S(4) {
		w = l.S(4)
	}
	platTri(ctx, (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5, w, w*0.55, dir, col, paintengine2d.Color{}, 0)
}

// Expander is the Platinum disclosure triangle: accent filled, outlined.
func (platinumEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := platColors(l)
	u := platU(l)
	s := l.S(9)
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	platTri(ctx, cx, cy, s, s*0.6, dir, c.light, c.deep, u)
}

// MenuHighlight is the Platinum menu selection: the dark accent with a
// lighter top line.
func (platinumEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := platColors(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.dark))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), platU(l)), paintengine2d.Fill(c.menuHiTop))
}

func (platinumEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := platColors(l)
	if hot {
		return c.menuHiTxt
	}
	return c.menuTxt
}

// Fields and lists show the accent focus frame.
func (platinumEngine) FieldFocusRing(l *Classic) bool { return true }

// ---- scrollbars -------------------------------------------------------------

func (platinumEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsEnds, MinThumb: 16}
}

func (e platinumEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := platColors(l)
	u := platU(l)
	if p.Bar.Empty() {
		return
	}
	active := !st.Disabled && !p.Thumb.Empty()
	ctx.DrawRect(p.Bar, paintengine2d.Fill(c.black))
	body := p.Bar.Inset(u)
	if !p.Track.Empty() {
		t := p.Track
		if vertical {
			t = paintengine2d.XYWH(body.Min.X, t.Min.Y, body.Dx(), t.Dy())
		} else {
			t = paintengine2d.XYWH(t.Min.X, body.Min.Y, t.Dx(), body.Dy())
		}
		if active {
			e.Face(l, ctx, t, RoleTrack, StateNone)
			if st.Pressed == ScrollPageDec || st.Pressed == ScrollPageInc {
				ctx.DrawRect(t, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.12)))
			}
		} else {
			ctx.DrawRect(t, paintengine2d.Fill(c.trackOff))
		}
	}
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		ab := b.Inset(u)
		pressed := st.Pressed == part && active
		face := c.face
		if pressed {
			face = c.press
		}
		ctx.DrawRect(ab, paintengine2d.Fill(face))
		if pressed {
			Edge(ctx, ab, Shade(c.press, -0.3), Shade(c.press, 0.25))
		} else {
			Edge(ctx, ab, c.hi, c.lo)
		}
		col := c.text
		switch {
		case !active:
			col = c.off
		case pressed:
			col = c.pressTxt
		}
		e.Arrow(l, ctx, ab, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if !active {
		return
	}
	th := p.Thumb
	if vertical {
		th = paintengine2d.XYWH(p.Bar.Min.X, th.Min.Y, p.Bar.Dx(), th.Dy())
	} else {
		th = paintengine2d.XYWH(th.Min.X, p.Bar.Min.Y, th.Dx(), p.Bar.Dy())
	}
	c.thumb(ctx, th, u, vertical, st.Pressed == ScrollThumbPart)
}

// DrawScrollBar paints a bare track + thumb (widgets that lay out their own).
func (e platinumEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -----------------------------------------------------------------

func (platinumEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is the Platinum primary group box: an etched frame with the
// bold title on its top line. A raised box is a bevelled placard.
func (e platinumEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := platColors(l)
	u := platU(l)
	f := l.BoldFont()
	if raised {
		e.DrawPanel(l, ctx, b, true)
		if title != "" {
			l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+l.metrics.Pad, b.Min.Y+u*2, b.Dx()-l.metrics.Pad*2, f.Height()), c.text, AlignStart, 0)
		}
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	top := b.Min.Y
	if title != "" {
		top = b.Min.Y + f.Height()*0.5
	}
	frame := paintengine2d.XYWH(b.Min.X, snap(top), b.Dx(), b.Max.Y-snap(top))
	// Etched frame, scaled: grey outer line with white inside and below.
	c.etched(ctx, frame.Min.X, frame.Min.Y, frame.Dx()-u, u, false)
	c.etched(ctx, frame.Min.X, frame.Min.Y, frame.Dy()-u, u, true)
	c.etched(ctx, frame.Min.X, frame.Max.Y-u*2, frame.Dx(), u, false)
	c.etched(ctx, frame.Max.X-u*2, frame.Min.Y, frame.Dy(), u, true)
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(8)
	tw := f.Advance(title) + l.S(6)
	if tw > b.Dx()-l.S(16) {
		tw = b.Dx() - l.S(16)
	}
	ctx.DrawRect(paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), paintengine2d.Fill(c.face))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// platFrame is the window frame thickness (black, white, 2× face, shadow,
// black) and platBarH the title bar height.
func platFrame(l *Classic) float32 { return platU(l) * 6 }

func platBarH(l *Classic) float32 {
	h := l.BoldFont().Height() + l.S(4)
	if h < l.S(19) {
		h = l.S(19)
	}
	return snap(h)
}

func (platinumEngine) WindowFrameInsets(l *Classic) Insets {
	fr := platFrame(l)
	return Insets{Top: platU(l)*2 + platBarH(l) + platU(l)*2, Right: fr, Bottom: fr, Left: fr}
}

// platBox is a title-bar widget box (close, zoom, collapse) of bar.
func platBox(l *Classic, bar paintengine2d.Rect, left bool, i int) paintengine2d.Rect {
	s := snap(l.S(11))
	if s > bar.Dy()-platU(l)*4 {
		s = bar.Dy() - platU(l)*4
	}
	y := snap(bar.Min.Y + (bar.Dy()-s)*0.5)
	if left {
		return paintengine2d.XYWH(snap(bar.Min.X+l.S(7)), y, s, s)
	}
	return paintengine2d.XYWH(snap(bar.Max.X-l.S(7)-s-float32(i)*(s+l.S(5))), y, s, s)
}

// platBar is the title bar strip of a frame of bounds b.
func platBar(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	u := platU(l)
	return paintengine2d.XYWH(b.Min.X+u*2, b.Min.Y+u*2, b.Dx()-u*4, platBarH(l))
}

// WindowCloseRect is the square close box at the title bar's left end.
// PopupShadow: classic Mac menus and balloons cast a hard one-pixel shadow
// to the lower right; windows cast none.
func (platinumEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	if kind == PopupDialog {
		return Insets{}
	}
	return ShadowReach(l.S(1), l.S(1), 0, 0)
}

func (platinumEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if kind == PopupDialog {
		return
	}
	DropShadow(ctx, b, 0, paintengine2d.RGBA(0, 0, 0, 0.55), l.S(1), l.S(1), 0, 0)
}

// ItemFocus: none — a focused list is ringed as a whole (its view frame
// draws the look's focus ring), the Mac way.
func (platinumEngine) ItemFocus(*Classic, *paintengine2d.Context, paintengine2d.Rect, ControlState) {}

func (platinumEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return platBox(l, platBar(l, b), true, 0)
}

// winBox paints one title-bar box: etched into the bar, black frame and a
// diagonal light gradient; pressed boxes darken.
func (c *plat) winBox(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, pressed bool) {
	Edge(ctx, b.Inset(-u), Shade(c.title, -0.3), c.hi)
	ctx.DrawRect(b, paintengine2d.Fill(c.black))
	in := b.Inset(u)
	if pressed {
		ctx.DrawRect(in, paintengine2d.Fill(Shade(c.title, -0.45)))
		Edge(ctx, in, Shade(c.title, -0.65), Shade(c.title, -0.2))
		return
	}
	ctx.DrawRect(in, DGradient(in, c.closeBox...))
	Edge(ctx, in, c.hi, Shade(c.title, -0.3))
}

// DrawWindowFrame is a Platinum window: a grey frame, a title bar of raised
// ridges around the bold centred title, the close box on the left and the
// zoom and collapse boxes on the right. Inactive windows go flat.
func (e platinumEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := platColors(l)
	u := platU(l)
	fr := platFrame(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.title))
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
	Edge(ctx, b.Inset(u), c.hi, Shade(c.title, -0.25))
	bar := platBar(l, b)
	content := paintengine2d.XYWH(b.Min.X+fr, bar.Max.Y+u*2, b.Dx()-fr*2, b.Max.Y-fr-bar.Max.Y-u*2)
	if !content.Empty() {
		ctx.DrawRect(content.Inset(-u), paintengine2d.Fill(c.black))
		ctx.DrawRect(content, paintengine2d.Fill(c.face))
		ctx.DrawRect(paintengine2d.XYWH(content.Min.X, content.Min.Y, content.Dx(), u), paintengine2d.Fill(c.hi))
		ctx.DrawRect(paintengine2d.XYWH(content.Min.X, content.Min.Y, u, content.Dy()), paintengine2d.Fill(c.hi))
	}
	f := l.BoldFont()
	tw := float32(0)
	if title != "" {
		tw = f.Advance(title)
	}
	left, right := bar.Min.X+u*2, bar.Max.X-u*2
	gap := l.S(8)
	var tx float32
	place := func() {
		// Centred over the whole bar, kept clear of the boxes.
		tx = bar.Min.X + (bar.Dx()-tw)*0.5
		if tx < left+gap {
			tx = left + gap
		}
		if tx+tw > right-gap {
			tx = right - gap - tw
		}
	}
	place()
	if st.Active {
		if st.CanClose {
			cb := e.WindowCloseRect(l, b)
			c.winBox(ctx, cb, u, st.ClosePress)
			left = cb.Max.X + l.S(4)
		}
		zoom := platBox(l, bar, false, 0)
		c.winBox(ctx, zoom, u, false)
		ctx.DrawRect(paintengine2d.XYWH(zoom.Min.X+u, zoom.Min.Y+u, zoom.Dx()*0.55, zoom.Dy()*0.55), paintengine2d.StrokePaint(c.black, u))
		shade := platBox(l, bar, false, 1)
		c.winBox(ctx, shade, u, false)
		mid := snap(shade.Min.Y + shade.Dy()*0.5)
		ctx.DrawRect(paintengine2d.XYWH(shade.Min.X+u, mid-u, shade.Dx()-u*2, u), paintengine2d.Fill(c.black))
		ctx.DrawRect(paintengine2d.XYWH(shade.Min.X+u, mid+u, shade.Dx()-u*2, u), paintengine2d.Fill(c.black))
		right = shade.Min.X - l.S(4)
		place()
		cy := bar.Min.Y + bar.Dy()*0.5
		if title == "" {
			c.ridges(ctx, left, right, cy, u, 6)
		} else {
			c.ridges(ctx, left, tx-gap, cy, u, 6)
			c.ridges(ctx, tx+tw+gap, right, cy, u, 6)
		}
	}
	if title == "" {
		return
	}
	col := c.text
	if !st.Active {
		col = c.dim
	}
	if tx < left {
		tx = left
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx, bar.Min.Y, right-tx, bar.Dy()), col, AlignStart, 0)
}

// ---- controls ---------------------------------------------------------------

// DrawPanel: a flat panel is the Platinum tab pane (light face, black
// outline, raised bevel); a raised one is a bevelled placard.
func (e platinumEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := platColors(l)
	u := platU(l)
	if raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
		Edge(ctx, b.Inset(u), c.hi, c.shadow)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.pane))
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
	Edge(ctx, b.Inset(u), c.hi, c.lo)
}

func (e platinumEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := platColors(l)
	u := platU(l)
	body := b
	if st.Primary() {
		// The default ring: a black outline and a bevelled band, three
		// pixels wide, hugging the button.
		r := l.S(6)
		edge := c.black
		if st.Disabled() {
			edge = c.off
		}
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(edge))
		band := b.Inset(u)
		ctx.DrawRoundRect(band, r-u, r-u, VGradient(band, c.defRing...))
		if !st.Disabled() {
			c.bevel(ctx, band, r-u, u, c.hi, paintengine2d.Color{}, c.shadow)
		}
		body = b.Inset(u * 3)
	}
	pressed := st.Pressed() && !st.Disabled()
	c.button(l, ctx, body, pressed, st.Disabled())
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.dim
	case pressed:
		fg = c.pressTxt
	}
	l.drawFittedText(ctx, l.body, label, body, fg, AlignCenter, l.S(6))
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body.Inset(u), l.S(3), u)
	}
}

// DrawComboBox is the Platinum pop-up button: a bevelled rounded button
// with the double arrow behind an etched divider on the right.
func (e platinumEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := platColors(l)
	u := platU(l)
	pressed := (open || st.Pressed()) && !st.Disabled()
	c.button(l, ctx, b, pressed, st.Disabled())
	aw := snap(l.S(18))
	ax := b.Max.X - aw - u
	fg, glyph := c.text, c.text
	switch {
	case st.Disabled():
		fg, glyph = c.dim, c.off
	case pressed:
		fg, glyph = c.pressTxt, c.pressTxt
	}
	if !pressed {
		c.etched(ctx, snap(ax), b.Min.Y+u*3, b.Dy()-u*6, u, true)
	}
	cx, cy := ax+aw*0.5+u, b.Min.Y+b.Dy()*0.5
	w, h := l.S(7), l.S(4)
	platTri(ctx, cx, cy-h*0.5-l.S(1.5), w, h, DirUp, glyph, paintengine2d.Color{}, 0)
	platTri(ctx, cx, cy+h*0.5+l.S(1.5), w, h, DirDown, glyph, paintengine2d.Color{}, 0)
	if text == "" {
		fg = c.dim
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, ax-b.Min.X-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.ring(ctx, b.Inset(u), l.S(3), u)
	}
}

// DrawSpinner is the Platinum "little arrows": a small bevelled box split
// in two; the held half goes dark.
func (e platinumEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := platColors(l)
	u := platU(l)
	w := b.Dx()
	if w > l.S(15) {
		w = l.S(15)
	}
	box := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-w)*0.5), b.Min.Y, snap(w), b.Dy())
	r := l.S(3)
	edge := c.black
	if st.Disabled() {
		edge = c.off
	}
	ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(edge))
	mid := snap(box.Min.Y + box.Dy()*0.5)
	half := func(hb paintengine2d.Rect, dir Direction, pressed bool) {
		face := c.face
		if pressed && !st.Disabled() {
			face = c.press
		}
		ctx.DrawRoundRect(hb, r-u, r-u, paintengine2d.Fill(face))
		col := c.text
		switch {
		case st.Disabled():
			col = c.off
		case pressed:
			col = c.pressTxt
		default:
			Edge(ctx, hb, c.hi, c.lo)
		}
		e.Arrow(l, ctx, hb, dir, col)
	}
	half(paintengine2d.XYWH(box.Min.X+u, box.Min.Y+u, box.Dx()-u*2, mid-box.Min.Y-u), DirUp, upPress)
	half(paintengine2d.XYWH(box.Min.X+u, mid+u, box.Dx()-u*2, box.Max.Y-mid-u*2), DirDown, downPress)
}

// DrawTabBar is the strip above a tab pane: window face with the pane's
// black top edge along its bottom.
func (e platinumEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.black))
}

// DrawTab is a Platinum folder tab: slanted sides, rounded top corners, a
// black outline; the selected tab is lighter and opens into its pane.
func (e platinumEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := platColors(l)
	u := platU(l)
	top := b.Min.Y + l.S(4)
	if selected {
		top = b.Min.Y + l.S(1)
	}
	top = snap(top)
	slant := l.S(6)
	r := l.S(4)
	x0, x1, y1 := b.Min.X, b.Max.X, b.Max.Y
	shape := func(close bool) *paintengine2d.Path {
		p := paintengine2d.NewPath()
		p.MoveTo(x0, y1)
		p.LineTo(x0+slant-r*0.35, top+r)
		p.QuadTo(x0+slant, top, x0+slant+r, top)
		p.LineTo(x1-slant-r, top)
		p.QuadTo(x1-slant, top, x1-slant+r*0.35, top+r)
		p.LineTo(x1, y1)
		if close {
			p.Close()
		}
		return p
	}
	face := c.title
	switch {
	case selected:
		face = c.pane
	case st.Pressed() && !st.Disabled():
		face = Shade(c.title, -0.12)
	case st.Hovered() && !st.Disabled():
		face = Mix(c.title, c.pane, 0.35)
	}
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawPath(shape(true), paintengine2d.Fill(face))
	// Highlight inside the left slant and along the top, then the outline.
	hl := paintengine2d.NewPath()
	hl.MoveTo(x0+u, y1-u)
	hl.LineTo(x0+slant-r*0.35+u, top+r)
	hl.QuadTo(x0+slant+u*0.5, top+u, x0+slant+r, top+u)
	hl.LineTo(x1-slant-r, top+u)
	ctx.DrawPath(hl, paintengine2d.StrokePaint(c.hi, u))
	ctx.DrawPath(shape(false), paintengine2d.Paint{Color: c.black, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	ctx.Restore()
	if selected {
		// Open the pane edge under the selected tab.
		ctx.DrawRect(paintengine2d.XYWH(x0+u, y1-u, x1-x0-u*2, u), paintengine2d.Fill(c.pane))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	lb := paintengine2d.XYWH(x0+slant, top, x1-x0-slant*2, y1-top)
	f := l.BoldFont()
	if f.Advance(label) > lb.Dx()-l.S(4) {
		f = l.body
	}
	l.drawFittedText(ctx, f, label, lb, fg, AlignCenter, l.S(4))
	if st.Focused() && selected {
		tw := f.Advance(label) + l.S(8)
		if tw > lb.Dx() {
			tw = lb.Dx()
		}
		th := f.Height() + l.S(2)
		c.ring(ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-tw)*0.5, lb.Min.Y+(lb.Dy()-th)*0.5, tw, th), 0, u)
	}
}

func (e platinumEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.black))
}

func (e platinumEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := platColors(l)
	u := platU(l)
	hl := paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), b.Dy()-u*2)
	fg := c.menuTxt
	switch {
	case st.Disabled():
		fg = c.dim
	case open || st.Pressed():
		ctx.DrawRect(hl, paintengine2d.Fill(c.dark))
		fg = c.menuHiTxt
	case st.Hovered():
		// Platinum had no hover here; a pale accent keeps the feedback.
		ctx.DrawRect(hl, paintengine2d.Fill(c.pale.WithAlpha(0.6)))
	}
	// Mac menus never underline mnemonics; the keys still work.
	l.drawLabeled(ctx, l.body, label, -1, b, fg)
	if st.Focused() && !open {
		c.ring(ctx, hl, 0, u)
	}
}

// DrawMenuFrame is the Platinum menu: face, black outline, raised bevel.
func (e platinumEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
	Edge(ctx, b.Inset(u), c.hi, c.shadow)
}

// DrawMenuItem: shortcuts follow the label colour, no underlines, etched
// separators.
func (e platinumEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := platColors(l)
	ch := MenuChromeFor(l)
	u := platU(l)
	if row.Separator {
		y := snap((b.Min.Y+b.Max.Y)*0.5) - u
		x0 := b.Min.X - ch.PadL + u*2
		c.etched(ctx, x0, y, b.Max.X+ch.PadR-u*2-x0, u, false)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := c.menuTxt
	switch {
	case st.Disabled():
		fg = c.dim
	case hot:
		fg = c.menuHiTxt
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	l.drawMenuGutter(ctx, b, ch, row, fg)
	right := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = l.S(10)
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		platTri(ctx, ab.Min.X+ab.Dx()*0.5, ab.Min.Y+ab.Dy()*0.5, l.S(9), l.S(5), DirRight, fg, paintengine2d.Color{}, 0)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := f.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg)
		right = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if right < lx {
		right = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	f.Draw(ctx, row.Label, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// DrawProgressBar: the determinate bar is an accent tube beside a grey
// groove; the busy bar is the striped cylinder, turning with phase.
func (e platinumEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := platColors(l)
	u := platU(l)
	h := b.Dy()
	if h > l.S(14) {
		h = l.S(14)
	}
	bar := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-h)*0.5), b.Dx(), snap(h))
	ctx.DrawRect(bar, paintengine2d.Fill(c.black))
	in := bar.Inset(u)
	if in.Empty() {
		return
	}
	tube := c.tube
	if st.Disabled() {
		tube = c.greyTube
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		per := l.S(14)
		sx := in.Min.X + phase*per
		pole := c.pole
		ctx.DrawRect(in, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(sx, in.Min.Y), End: paintengine2d.Pt(sx+per*0.5, in.Min.Y+per*0.5),
			Stops: pole, Tile: paintengine2d.TileRepeat}))
		ctx.DrawRect(in, VGradient(in, c.poleShade...))
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	ctx.DrawRect(in, VGradient(in, c.groove...))
	w := snap(in.Dx() * t)
	if w <= 0 {
		return
	}
	fill := paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy())
	ctx.DrawRect(fill, VGradient(fill, tube...))
	if fill.Max.X < in.Max.X {
		ctx.DrawRect(paintengine2d.XYWH(fill.Max.X, in.Min.Y, u, in.Dy()), paintengine2d.Fill(c.black))
	}
}

// DrawSlider is the Platinum slider: a black-outlined groove and the
// accent thumb that points down, with grip lines.
func (e platinumEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := platColors(l)
	u := platU(l)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	tw := snap(l.metrics.Thumb)
	if tw <= 0 {
		tw = snap(l.S(13))
	}
	th := snap(tw * 1.3)
	if th > b.Dy()-u*2 {
		th = b.Dy() - u*2
	}
	cy := snap(b.Min.Y + b.Dy()*0.5)
	gh := snap(l.S(6))
	groove := paintengine2d.XYWH(b.Min.X+l.S(2), cy-gh*0.5, b.Dx()-l.S(4), gh)
	ctx.DrawRoundRect(groove, gh*0.5, gh*0.5, paintengine2d.Fill(c.black))
	gi := groove.Inset(u)
	ctx.DrawRoundRect(gi, gi.Dy()*0.5, gi.Dy()*0.5, VGradient(gi, c.groove...))
	x0, x1 := b.Min.X+tw*0.5+l.S(2), b.Max.X-tw*0.5-l.S(2)
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	ty := snap(cy - th*0.45)
	pt := tw * 0.45
	knob := paintengine2d.NewPath()
	knob.MoveTo(tx+u*2, ty)
	knob.LineTo(tx+tw-u*2, ty)
	knob.QuadTo(tx+tw, ty, tx+tw, ty+u*2)
	knob.LineTo(tx+tw, ty+th-pt)
	knob.LineTo(tx+tw*0.5, ty+th)
	knob.LineTo(tx, ty+th-pt)
	knob.LineTo(tx, ty+u*2)
	knob.QuadTo(tx, ty, tx+u*2, ty)
	knob.Close()
	face, edge, hi, grip := c.light, c.black, c.pale, c.dark
	switch {
	case st.Disabled():
		face, edge, hi, grip = c.face, c.off, c.hi, c.off
	case st.Pressed():
		face = c.acc
	}
	ctx.DrawPath(knob, paintengine2d.Fill(face))
	ctx.DrawRect(paintengine2d.XYWH(tx+u, ty+u, tw-u*2, u), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(tx+u, ty+u, u, th-pt-u), paintengine2d.Fill(hi))
	ctx.DrawPath(knob, paintengine2d.Paint{Color: edge, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Join: paintengine2d.JoinMiter, MiterLimit: 6}})
	gx := snap(tx + tw*0.5 - u*2.5)
	for i := 0; i < 3; i++ {
		ctx.DrawRect(paintengine2d.XYWH(gx+float32(i)*u*2, ty+u*3, u, th-pt-u*3), paintengine2d.Fill(grip))
	}
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, paintengine2d.XYWH(tx-u*3, ty-u*3, tw+u*6, th+u*6).Intersect(b), 0, u)
	}
}

// DrawSwitch: Platinum had no switch; it is a sunken slot (accent when on)
// with a small bevelled knob.
func (e platinumEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := platColors(l)
	u := platU(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	track := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-th)*0.5), snap(tw), snap(th))
	r := l.S(3)
	ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.black))
	in := track.Inset(u)
	fill := c.track
	if on && !st.Disabled() {
		fill = c.light
	}
	if st.Disabled() {
		fill = c.face
	}
	ctx.DrawRoundRect(in, r-u, r-u, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X+r, in.Min.Y, in.Dx()-r*2, u), paintengine2d.Fill(c.shadow))
	kw := in.Dy()
	kx := in.Min.X
	if on {
		kx = in.Max.X - kw
	}
	c.button(l, ctx, paintengine2d.XYWH(kx, in.Min.Y, kw, in.Dy()), false, st.Disabled())
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, track, r, u)
	}
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.dim
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
}

func (e platinumEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	selected, hovered := st.Checked(), st.Hovered()
	fg := l.fieldText()
	if selected || hovered {
		fg = e.Face(l, ctx, b, RoleRow, st&^StateFocused)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy()), fg, AlignStart, 0)
}

func (e platinumEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	selected, hovered := st.Checked(), st.Hovered()
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(3)
	fg := l.fieldText()
	if selected || hovered {
		// The Finder highlights the name, not the whole row.
		hb := paintengine2d.XYWH(lx-l.S(3), b.Min.Y+l.S(1), f.Advance(label)+l.S(6), b.Dy()-l.S(2)).Intersect(b)
		fg = e.Face(l, ctx, hb, RoleRow, st&^StateFocused)
	}
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, fg)
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawTableHeader: Platinum list headers are bevelled buttons; the sorted
// column is sunk a shade darker.
func (e platinumEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := platColors(l)
	u := platU(l)
	face := c.face
	switch {
	case st.Pressed() && !st.Disabled():
		face = c.lo
	case sorted:
		face = Mix(c.face, c.lo, 0.5)
	}
	ctx.DrawRect(b, paintengine2d.Fill(face))
	if st.Pressed() && !st.Disabled() {
		Edge(ctx, b, c.shadow, c.hi)
	} else {
		Edge(ctx, b, c.hi, c.shadow)
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.black))
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()-u), fg, AlignStart, 0)
}

func (e platinumEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	c.etched(ctx, b.Min.X, b.Max.Y-u*2, b.Dx(), u, false)
}

// DrawStatusBar is a Platinum window header: a bevelled strip with a black
// edge, the parts, and the ridged size box in the corner.
func (e platinumEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.black))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), u), paintengine2d.Fill(c.hi))
	grip := l.S(16)
	if len(parts) > 0 {
		slot := (b.Dx() - grip) / float32(len(parts))
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			if i > 0 {
				c.etched(ctx, snap(x), b.Min.Y+l.S(5), b.Dy()-l.S(9), u, true)
			}
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y+u*2, slot-l.S(14), b.Dy()-u*2), c.text, AlignStart, 0)
		}
	}
	// Size box: diagonal ridges.
	gx, gy := b.Max.X-l.S(3), b.Max.Y-l.S(3)
	for i := 1; i <= 3; i++ {
		o := l.S(3.5) * float32(i)
		ctx.DrawLine(paintengine2d.Pt(gx-o, gy), paintengine2d.Pt(gx, gy-o), paintengine2d.StrokePaint(c.shadow, u))
		ctx.DrawLine(paintengine2d.Pt(gx-o+u, gy), paintengine2d.Pt(gx, gy-o+u), paintengine2d.StrokePaint(c.hi, u))
	}
}

// DrawTitleBar is a Platinum window header strip with a bold title.
func (e platinumEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u*2, b.Dx(), u), paintengine2d.Fill(c.shadow))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.black))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u*2), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u*2), c.dim, AlignStart, 0)
	}
}

// DrawAccordionHeader is a Platinum disclosure row: triangle, bold label
// and an etched rule.
func (e platinumEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := platColors(l)
	u := platU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.pale.WithAlpha(0.35)))
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, c.text)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()-u*2), fg, AlignStart, 0)
	c.etched(ctx, b.Min.X, b.Max.Y-u*2, b.Dx(), u, false)
	if st.Focused() {
		c.ring(ctx, b, 0, u)
	}
}

func (platinumEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := platColors(l)
	u := platU(l)
	if vertical {
		c.etched(ctx, snap((b.Min.X+b.Max.X)*0.5)-u, b.Min.Y+l.S(2), b.Dy()-l.S(4), u, true)
		return
	}
	c.etched(ctx, b.Min.X, snap((b.Min.Y+b.Max.Y)*0.5)-u, b.Dx(), u, false)
}

// DrawSplitter: face with a short ridged grip, darker while dragged.
func (platinumEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := platColors(l)
	face := c.face
	if st.Hovered() || st.Pressed() {
		face = Mix(c.face, c.lo, 0.5)
	}
	ctx.DrawRect(b, paintengine2d.Fill(face))
	g := l.S(14)
	if vertical {
		GripLines(ctx, paintengine2d.XYWH(b.Min.X, (b.Min.Y+b.Max.Y)*0.5-g*0.5, b.Dx(), g), true, 2, l.S(3), c.hi, c.shadow)
	} else {
		GripLines(ctx, paintengine2d.XYWH((b.Min.X+b.Max.X)*0.5-g*0.5, b.Min.Y, g, b.Dy()), false, 2, l.S(3), c.hi, c.shadow)
	}
}

// DrawTooltip is Balloon Help: a white rounded balloon with a black rim.
func (platinumEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := platColors(l)
	u := platU(l)
	r := l.S(6)
	if r > b.Dy()*0.5 {
		r = b.Dy() * 0.5
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.black))
	ctx.DrawRoundRect(b.Inset(u), r-u, r-u, paintengine2d.Fill(c.info))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.infoTxt, AlignStart, 0)
}

// DrawFocusRing is the Platinum focus frame: two pixels of the accent.
func (platinumEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	platColors(l).ring(ctx, b, 0, platU(l))
}

// ---- packs --------------------------------------------------------------------

func platinumPack(name, label string, year int, summary string, pal Palette, extra map[string]string) ThemePack {
	tok := ThemeTokens{
		Engine:  "platinum",
		Bevel:   BevelClassic3D,
		Family:  ThemeLight,
		Palette: pal,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.2), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Mac OS", Summary: summary,
		Era: "Platinum", Palette: ThemeLight, Tokens: tok,
	}
}

// platinumPalette is the Platinum grey scheme around one accent colour
// (menus, thumbs, focus) and the text highlight colour.
func platinumPalette(accent, accentDark, highlight string) Palette {
	face := hexColor("#dddddd")
	return Palette{
		Background: face, Surface: face, SurfaceAlt: hexColor("#cccccc"),
		Border: hexColor("#000000"), Divider: hexColor("#999999"),
		Text: hexColor("#000000"), TextMuted: hexColor("#777777"), TextOnAccent: hexColor("#ffffff"),
		Accent: hexColor(accent), AccentHover: Shade(hexColor(accent), 0.3), AccentPress: hexColor(accentDark),
		Field: hexColor("#ffffff"), FieldBorder: hexColor("#000000"),
		Focus: hexColor(accent), Selection: hexColor(highlight),
		Track: hexColor("#aaaaaa"), Thumb: Shade(hexColor(accent), 0.4),
		Highlight: paintengine2d.RGBA(1, 1, 1, 0.85), Shadow: paintengine2d.RGBA(0, 0, 0, 0.3),
		MenuHover: hexColor(accentDark), MenuHoverBorder: hexColor(accentDark), MenuGutter: face,
		Danger: hexColor("#cc0000"), Success: hexColor("#007a00"), Warning: hexColor("#b86e00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: hexColor("#ffffff"), BevelDark: hexColor("#999999"),
	}
}

func platinumPacks() []ThemePack {
	return []ThemePack{
		platinumPack("platinum", "Platinum", 1997, "Mac OS 8 and 9: grey bevels, ridged title bars, the Lavender accent.",
			platinumPalette("#6666cc", "#333399", "#ccccff"),
			map[string]string{
				"accentDeep": "#000055", "accentDark": "#333399", "accentLight": "#9999ff",
				"accentPale": "#ccccff", "accentHi": "#eeeeff", "pane": "#eeeeee", "title": "#cccccc",
			}),
		platinumPack("platinum-lime", "Platinum Lime", 1997, "Mac OS 8 with its Lime accent colour on menus, thumbs and progress bars.",
			platinumPalette("#669933", "#336600", "#ccff99"),
			map[string]string{
				"accentDeep": "#113300", "accentDark": "#336600", "accentLight": "#99cc66",
				"accentPale": "#ccff66", "accentHi": "#ffffcc", "pane": "#eeeeee", "title": "#cccccc",
			}),
	}
}
