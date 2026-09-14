package style

import "github.com/codemodify/paintengine2d"

// win95Engine paints Windows 95 / 98 / 2000 "classic" chrome: square
// corners, the four-colour 3D bevel (highlight, light, shadow, dark
// shadow), navy selection with white text, dotted focus rectangles,
// arrow-button scrollbars and chamfered tabs that join their page.
//
// Colours come from the pack: Palette.SurfaceAlt is COLOR_3DFACE and the
// four bevel colours are the extras "hi", "light", "shadow", "dk" (derived
// from the palette when a pack omits them). Recipes follow the shipped
// system metrics as documented by the 98.css reconstruction.
//
// This is the reference engine: read it before writing another one.
type win95Engine struct{ BaseEngine }

func init() {
	RegisterEngine(win95Engine{})
	for _, p := range win95Packs() {
		RegisterPack(p)
	}
}

func (win95Engine) ID() string { return "win95" }

// DefaultMetrics are Win95's proportions scaled to the toolkit's 16px UI
// font (Windows used 11px MS Sans Serif): short controls, 16px scrollbars.
func (win95Engine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Square: true, BevelDepth: 2,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 13, Radio: 12,
		MenuItemH: 24, MenuBarH: 24, TabH: 26, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 20,
		Scroll: 16, Pad: 10, FieldPad: 5, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24,
	}
}

// w95 is the resolved system-colour set of a look.
type w95 struct {
	face, hi, light, shadow, dk    paintengine2d.Color
	text, gray, field, sel, selTxt paintengine2d.Color
	info, infoTxt                  paintengine2d.Color
	cap1, cap2, capTxt             paintengine2d.Color
	capOff1, capOff2, capOffTxt    paintengine2d.Color
}

type w95Key struct{}

// w95colors is the look's resolved colour set (built once per look).
func w95colors(l *Classic) w95 {
	return l.Memo(w95Key{}, func() any { return w95build(l) }).(w95)
}

func w95build(l *Classic) w95 {
	p := l.palette
	face := p.SurfaceAlt
	c := w95{
		face:   face,
		hi:     l.X("hi", Shade(face, 0.75)),
		light:  l.X("light", Shade(face, 0.35)),
		shadow: l.X("shadow", Shade(face, -0.33)),
		dk:     l.X("dk", Shade(face, -0.9)),
		text:   p.Text,
		gray:   p.TextMuted,
		field:  p.Field,
		sel:    p.Selection,
		selTxt: p.TextOnAccent,
		info:   l.X("info", Hex("#ffffe1")),
	}
	if c.sel.A < 0.9 {
		c.sel = p.Accent
	}
	c.infoTxt = l.X("infoText", Hex("#000000"))
	c.cap1 = l.X("caption", c.sel)
	c.cap2 = l.X("caption2", c.cap1)
	c.capTxt = l.X("captionText", Hex("#ffffff"))
	c.capOff1 = l.X("captionOff", c.shadow)
	c.capOff2 = l.X("captionOff2", c.capOff1)
	c.capOffTxt = l.X("captionOffText", c.face)
	return c
}

// raised is a push-button bevel: white / black outer, light / shadow inner.
func (c w95) raised(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	Bevel4(ctx, b, c.hi, c.dk, c.light, c.shadow)
}

// raisedWindow is a window / menu frame bevel: light / black outer,
// white / shadow inner.
func (c w95) raisedWindow(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	Bevel4(ctx, b, c.light, c.dk, c.hi, c.shadow)
}

// sunken is a field well: shadow / white outer, black / light inner.
func (c w95) sunken(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	Bevel4(ctx, b, c.shadow, c.hi, c.dk, c.light)
}

// pushed is a pressed button: black / white outer, shadow / light inner.
func (c w95) pushed(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	Bevel4(ctx, b, c.dk, c.hi, c.shadow, c.light)
}

// thin is a 1px bevel (tool buttons, status panels, menu titles on 98).
func (c w95) thin(ctx *paintengine2d.Context, b paintengine2d.Rect, up bool) {
	if up {
		Edge(ctx, b, c.hi, c.shadow)
		return
	}
	Edge(ctx, b, c.shadow, c.hi)
}

// trackFill approximates the dithered white/face scrollbar trough.
func (c w95) trackFill() paintengine2d.Color { return Mix(c.face, c.hi, 0.5) }

// ---- parts ------------------------------------------------------------------

func (win95Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := w95colors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.gray
	}
	switch role {
	case RoleButton, RoleThumb, RoleCombo:
		if role == RoleCombo {
			break // the combo field is painted by DrawComboBox
		}
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		switch {
		case st.Pressed() && !st.Disabled():
			c.pushed(ctx, b)
		case st.Primary() && !st.Disabled():
			// Default button: a black frame around the raised bevel.
			ctx.DrawRect(b.Inset(0.5), paintengine2d.StrokePaint(c.dk, 1))
			c.raised(ctx, b.Inset(1))
		default:
			c.raised(ctx, b)
		}
		return fg
	case RoleTool:
		switch {
		case st.Disabled():
		case st.Pressed():
			c.thin(ctx, b, false)
		case st.Checked():
			// Latched tool buttons sit in a dithered well.
			ctx.DrawRect(b.Inset(1), paintengine2d.Fill(c.trackFill()))
			c.thin(ctx, b, false)
		case st.Hovered():
			c.thin(ctx, b, true)
		}
		return fg
	case RoleField, RoleCheck:
		fill := c.field
		if st.Disabled() {
			fill = c.face
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		c.sunken(ctx, b)
		return fg
	case RoleRow, RoleMenu:
		if st.Checked() || (role == RoleMenu && (st.Hovered() || st.Pressed())) {
			ctx.DrawRect(b, paintengine2d.Fill(c.sel))
			return c.selTxt
		}
		return fg // no hot-tracking in Win95 lists
	case RoleTrack:
		ctx.DrawRect(b, paintengine2d.Fill(c.trackFill()))
		return fg
	case RoleTab:
		return fg // DrawTab paints the tab shape
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		return fg
	}
	return fg
}

func (win95Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := w95colors(l)
	fill := c.field
	if st.Disabled() || (st.Pressed() && !st.Disabled()) {
		fill = c.face // a pressed box greys like Windows did
	}
	ctx.DrawRect(box, paintengine2d.Fill(fill))
	c.sunken(ctx, box)
	if checked {
		tick := c.text
		if st.Disabled() {
			tick = c.gray
		}
		PixelTick(ctx, box.Inset(l.S(3)), tick)
	}
}

func (win95Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := w95colors(l)
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	r := box.Dx() * 0.5
	fill := c.field
	if st.Disabled() || st.Pressed() {
		fill = c.face
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(fill))
	// Two rings split on the diagonal: shadow / white outside, black /
	// light inside — the round version of the field bevel.
	const pi = 3.14159265
	arc := func(rad float32, col paintengine2d.Color, upperLeft bool) {
		start := float32(-pi / 4)
		if upperLeft {
			start = 3 * pi / 4
		}
		ctx.DrawArc(paintengine2d.Pt(cx, cy), rad, rad, start, pi, paintengine2d.StrokePaint(col, 1))
	}
	arc(r-0.5, c.shadow, true)
	arc(r-0.5, c.hi, false)
	arc(r-1.5, c.dk, true)
	arc(r-1.5, c.light, false)
	if selected {
		dot := c.text
		if st.Disabled() {
			dot = c.gray
		}
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r*0.34, paintengine2d.Fill(dot))
	}
}

func (win95Engine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	// Win95 arrows are small and solid (the Marlett glyphs): 7x4 in a
	// 16px button.
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	g := s * 0.5
	if g > l.S(8) {
		g = l.S(8)
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	FillArrow(ctx, paintengine2d.XYWH(cx-g*0.5, cy-g*0.5, g, g), dir, col)
}

func (win95Engine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	// The TreeView +/- box: 9px, grey frame, black sign.
	c := w95colors(l)
	s := l.S(9)
	x := b.Min.X + (b.Dx()-s)*0.5
	y := b.Min.Y + (b.Dy()-s)*0.5
	box := paintengine2d.XYWH(snap(x), snap(y), s, s)
	ctx.DrawRect(box, paintengine2d.Fill(c.field))
	ctx.DrawRect(box.Inset(0.5), paintengine2d.StrokePaint(c.shadow, 1))
	mid := box.Min.Y + s*0.5 - 0.5
	ctx.DrawRect(paintengine2d.XYWH(box.Min.X+2, snap(mid), s-4, 1), paintengine2d.Fill(c.text))
	if !expanded {
		ctx.DrawRect(paintengine2d.XYWH(snap(box.Min.X+s*0.5-0.5), box.Min.Y+2, 1, s-4), paintengine2d.Fill(c.text))
	}
}

func (win95Engine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := w95colors(l)
	if l.P("flatMenuBar", 0) != 0 && attachBottom {
		// Windows 98 / 2000 menu bar: an open title is a sunken 1px well.
		c.thin(ctx, b, false)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.sel))
}

func (win95Engine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := w95colors(l)
	if hot && l.P("flatMenuBar", 0) == 0 {
		return c.selTxt
	}
	return c.text
}

// Fields show only the caret when focused.
func (win95Engine) FieldFocusRing(l *Classic) bool { return false }

// ---- scrollbars -------------------------------------------------------------

func (win95Engine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsEnds, MinThumb: 8}
}

func (e win95Engine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := w95colors(l)
	ctx.DrawRect(p.Bar, paintengine2d.Fill(c.trackFill()))
	// A held track darkens the part being paged (black dither on 95).
	if st.Pressed == ScrollPageDec || st.Pressed == ScrollPageInc {
		pg := p.Track
		if !p.Thumb.Empty() {
			if vertical {
				if st.Pressed == ScrollPageDec {
					pg.Max.Y = p.Thumb.Min.Y
				} else {
					pg.Min.Y = p.Thumb.Max.Y
				}
			} else if st.Pressed == ScrollPageDec {
				pg.Max.X = p.Thumb.Min.X
			} else {
				pg.Min.X = p.Thumb.Max.X
			}
		}
		ctx.DrawRect(pg, paintengine2d.Fill(c.dk))
	}
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		ctx.DrawRect(b, paintengine2d.Fill(c.face))
		pressed := st.Pressed == part && !st.Disabled
		if pressed {
			// Pressed arrows go flat with a single shadow line.
			ctx.DrawRect(b.Inset(0.5), paintengine2d.StrokePaint(c.shadow, 1))
		} else {
			c.raised(ctx, b)
		}
		g := b.Inset(b.Dx() * 0.25)
		if pressed {
			g = g.Translate(paintengine2d.Pt(1, 1))
		}
		col := c.text
		if st.Disabled {
			// Disabled glyphs are embossed: white under grey.
			e.Arrow(l, ctx, g.Translate(paintengine2d.Pt(1, 1)), dir, c.hi)
			col = c.shadow
		}
		e.Arrow(l, ctx, g, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if !p.Thumb.Empty() {
		ctx.DrawRect(p.Thumb, paintengine2d.Fill(c.face))
		c.raised(ctx, p.Thumb)
	}
}

// ---- frames -----------------------------------------------------------------

func (win95Engine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

func (win95Engine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	top := b.Min.Y
	if title != "" {
		top = b.Min.Y + l.body.Height()*0.5
	}
	frame := paintengine2d.XYWH(b.Min.X, snap(top), b.Dx(), b.Max.Y-snap(top))
	Etched(ctx, frame, c.shadow, c.hi)
	if title == "" {
		return
	}
	f := l.body
	tx := b.Min.X + l.S(8)
	tw := f.Advance(title) + l.S(4)
	ctx.DrawRect(paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), paintengine2d.Fill(c.face))
	f.Draw(ctx, title, paintengine2d.Pt(tx+l.S(2), b.Min.Y), c.text)
}

// captionH is the in-app window caption height.
func w95CaptionH(l *Classic) float32 {
	h := l.body.Height() + l.S(6)
	if h < l.S(18) {
		h = l.S(18)
	}
	return h
}

func (win95Engine) WindowFrameInsets(l *Classic) Insets {
	fr := l.S(4)
	return Insets{Top: fr + w95CaptionH(l) + l.S(1), Right: fr, Bottom: fr, Left: fr}
}

func (e win95Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	c.raisedWindow(ctx, b)
	in := l.S(3)
	bar := paintengine2d.XYWH(b.Min.X+in, b.Min.Y+in, b.Dx()-in*2, w95CaptionH(l))
	c1, c2, tc := c.cap1, c.cap2, c.capTxt
	if !st.Active {
		c1, c2, tc = c.capOff1, c.capOff2, c.capOffTxt
	}
	if c1 == c2 {
		ctx.DrawRect(bar, paintengine2d.Fill(c1))
	} else {
		ctx.DrawRect(bar, HGradient(bar, Stop(0, c1), Stop(1, c2)))
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	right := bar.Max.X
	if st.CanClose {
		bh := bar.Dy() - l.S(4)
		bw := bh + l.S(2)
		cb := paintengine2d.XYWH(bar.Max.X-bw-l.S(2), bar.Min.Y+l.S(2), bw, bh)
		ctx.DrawRect(cb, paintengine2d.Fill(c.face))
		if st.ClosePress {
			c.pushed(ctx, cb)
		} else {
			c.raised(ctx, cb)
		}
		g := cb.Inset(cb.Dy() * 0.28)
		if st.ClosePress {
			g = g.Translate(paintengine2d.Pt(1, 1))
		}
		DrawCross(ctx, g, c.text, l.S(1.6))
		right = cb.Min.X - l.S(2)
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(bar.Min.X+l.S(4), bar.Min.Y, right-bar.Min.X-l.S(4), bar.Dy()), tc, AlignStart, 0)
	}
}

// ---- controls ---------------------------------------------------------------

func (e win95Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	fg := e.Face(l, ctx, b, RoleButton, st)
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(1, 1))
	}
	e.label(l, ctx, lb, label, fg, st.Disabled())
	if st.Focused() && !st.Disabled() {
		in := l.S(4)
		if st.Primary() {
			in = l.S(5)
		}
		DottedRect(ctx, b.Inset(in), w95colors(l).text)
	}
}

// label draws centred text; disabled text is embossed (white under grey).
func (win95Engine) label(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, fg paintengine2d.Color, disabled bool) {
	if text == "" {
		return
	}
	if disabled {
		c := w95colors(l)
		l.drawFittedText(ctx, l.body, text, b.Translate(paintengine2d.Pt(1, 1)), c.hi, AlignCenter, 8)
		fg = c.shadow
	}
	l.drawFittedText(ctx, l.body, text, b, fg, AlignCenter, 8)
}

func (e win95Engine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	if st.Pressed() || (st.Checked() && st.Toggle()) {
		// The icon sinks with the button.
		e.Face(l, ctx, b, RoleTool, st)
		l.baseDrawToolButton(ctx, b.Translate(paintengine2d.Pt(1, 1)), st&^(StateHovered|StatePressed|StateToggle|StateFocused), label, icon)
		return
	}
	l.baseDrawToolButton(ctx, b, st, label, icon)
}

func (e win95Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := w95colors(l)
	fill := c.field
	if st.Disabled() {
		fill = c.face
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	c.sunken(ctx, b)
	in := b.Inset(2)
	bw := l.S(16)
	if bw > in.Dy() {
		bw = in.Dy()
	}
	btn := paintengine2d.XYWH(in.Max.X-bw, in.Min.Y, bw, in.Dy())
	ctx.DrawRect(btn, paintengine2d.Fill(c.face))
	if open {
		ctx.DrawRect(btn.Inset(0.5), paintengine2d.StrokePaint(c.shadow, 1))
	} else {
		c.raised(ctx, btn)
	}
	g := btn
	if open {
		g = g.Translate(paintengine2d.Pt(1, 1))
	}
	col := c.text
	if st.Disabled() {
		col = c.shadow
	}
	e.Arrow(l, ctx, g, DirDown, col)
	// The selected text of a focused drop-list is highlighted navy.
	tb := paintengine2d.XYWH(in.Min.X+l.S(2), in.Min.Y+l.S(2), btn.Min.X-in.Min.X-l.S(4), in.Dy()-l.S(4))
	tc := c.text
	if st.Disabled() {
		tc = c.gray
	} else if st.Focused() && !open {
		ctx.DrawRect(tb, paintengine2d.Fill(c.sel))
		tc = c.selTxt
		DottedRect(ctx, tb, c.hi)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(tb.Min.X+l.S(3), tb.Min.Y, tb.Dx()-l.S(4), tb.Dy()), tc, AlignStart, 0)
}

func (e win95Engine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := w95colors(l)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	up := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y)
	dn := paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid)
	half := func(r paintengine2d.Rect, dir Direction, pressed bool) {
		ctx.DrawRect(r, paintengine2d.Fill(c.face))
		if pressed && !st.Disabled() {
			c.pushed(ctx, r)
			r = r.Translate(paintengine2d.Pt(1, 1))
		} else {
			c.raised(ctx, r)
		}
		col := c.text
		if st.Disabled() {
			col = c.shadow
		}
		e.Arrow(l, ctx, r, dir, col)
	}
	half(up, DirUp, upPress)
	half(dn, DirDown, downPress)
}

func (e win95Engine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	// The page's top edge: tabs sit on it and the selected one opens it.
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(c.hi))
}

func (e win95Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := w95colors(l)
	t := b
	if !selected {
		// Unselected tabs sit 2px lower and narrower than the selected one.
		t = paintengine2d.XYWH(b.Min.X+l.S(2), b.Min.Y+l.S(2), b.Dx()-l.S(4), b.Dy()-l.S(2))
	}
	ch := l.S(2) // chamfer
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X+1, t.Min.Y+1, t.Dx()-2, t.Dy()-1), paintengine2d.Fill(c.face))
	// Left + top highlight with the chamfered corner, right shadow + black.
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y+ch, 1, t.Dy()-ch), paintengine2d.Fill(c.hi))
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X+ch, t.Min.Y, t.Dx()-ch*2, 1), paintengine2d.Fill(c.hi))
	ctx.DrawRect(paintengine2d.XYWH(t.Min.X+1, t.Min.Y+1, 1, 1), paintengine2d.Fill(c.hi))
	ctx.DrawRect(paintengine2d.XYWH(t.Max.X-1, t.Min.Y+ch, 1, t.Dy()-ch), paintengine2d.Fill(c.dk))
	ctx.DrawRect(paintengine2d.XYWH(t.Max.X-2, t.Min.Y+1, 1, 1), paintengine2d.Fill(c.dk))
	ctx.DrawRect(paintengine2d.XYWH(t.Max.X-2, t.Min.Y+ch, 1, t.Dy()-ch), paintengine2d.Fill(c.shadow))
	if selected {
		// Open the page edge under the selected tab.
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X+1, b.Max.Y-1, t.Dx()-3, 1), paintengine2d.Fill(c.face))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.gray
	}
	lb := t
	if selected {
		lb.Max.Y -= l.S(2)
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, 8)
	if st.Focused() && selected {
		f := l.body
		w := f.Advance(label) + l.S(6)
		if w > lb.Dx()-l.S(6) {
			w = lb.Dx() - l.S(6)
		}
		h := f.Height() + l.S(2)
		DottedRect(ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h), c.text)
	}
}

func (win95Engine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(w95colors(l).face))
}

func (e win95Engine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := w95colors(l)
	hot := !st.Disabled() && (open || st.Pressed() || st.Hovered())
	fg := c.text
	if l.P("flatMenuBar", 0) != 0 {
		// 98 / 2000: hover raises a thin bevel, an open title sinks.
		switch {
		case open || st.Pressed():
			c.thin(ctx, b.Inset(1), false)
		case st.Hovered() && !st.Disabled():
			c.thin(ctx, b.Inset(1), true)
		}
	} else if hot && open {
		ctx.DrawRect(b.Inset(1), paintengine2d.Fill(c.sel))
		fg = c.selTxt
	}
	if st.Disabled() {
		fg = c.gray
	}
	lb := b
	if open && l.P("flatMenuBar", 0) != 0 {
		lb = lb.Translate(paintengine2d.Pt(1, 1))
	}
	l.drawLabeled(ctx, l.body, label, underline, lb, fg)
	if st.Focused() && !open {
		DottedRect(ctx, b.Inset(2), c.text)
	}
}

func (win95Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	c.raisedWindow(ctx, b)
}

func (e win95Engine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	if row.Separator {
		c := w95colors(l)
		ch := MenuChromeFor(l)
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := b.Min.X - ch.PadL + l.S(3)
		EtchedLine(ctx, x0, y-1, b.Max.X+ch.PadR-l.S(3)-x0, false, c.shadow, c.hi)
		return
	}
	l.baseDrawMenuItem(ctx, b, st, row)
}

func (e win95Engine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	Edge(ctx, b, c.shadow, c.hi)
	in := b.Inset(l.S(2))
	if in.Empty() {
		return
	}
	col := c.sel
	if st.Disabled() {
		col = c.shadow
	}
	cw := in.Dy() * 0.62 // chunk width
	gap := l.S(2)
	chunk := func(x0, x1 float32) {
		for x := x0; x < x1; x += cw + gap {
			w := cw
			if x+w > x1 {
				w = x1 - x
			}
			if w > 0.5 {
				ctx.DrawRect(paintengine2d.XYWH(snap(x), in.Min.Y, w, in.Dy()), paintengine2d.Fill(col))
			}
		}
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		span := (cw + gap) * 3
		x := in.Min.X + (in.Dx()+span)*phase - span
		lo, hi := x, x+span
		if lo < in.Min.X {
			lo = in.Min.X
		}
		if hi > in.Max.X {
			hi = in.Max.X
		}
		chunk(lo, hi)
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	chunk(in.Min.X, in.Min.X+in.Dx()*t)
}

func (e win95Engine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := w95colors(l)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	tw := l.S(11)
	th := b.Dy() - l.S(2)
	if th > l.S(21) {
		th = l.S(21)
	}
	x0 := b.Min.X + tw*0.5
	x1 := b.Max.X - tw*0.5
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	groove := paintengine2d.XYWH(b.Min.X+l.S(2), cy-l.S(2), b.Dx()-l.S(4), l.S(4))
	c.sunken(ctx, groove)
	ctx.DrawRect(groove.Inset(2), paintengine2d.Fill(c.face))
	tx := snap(x0 + (x1-x0)*t - tw*0.5)
	ty := snap(b.Min.Y + (b.Dy()-th)*0.5)
	// The trackbar thumb: a rectangle ending in a point at the bottom.
	pt := tw * 0.5
	path := paintengine2d.NewPath()
	path.MoveTo(tx, ty)
	path.LineTo(tx+tw, ty)
	path.LineTo(tx+tw, ty+th-pt)
	path.LineTo(tx+tw*0.5, ty+th)
	path.LineTo(tx, ty+th-pt)
	path.Close()
	fill := c.face
	if st.Pressed() {
		fill = c.trackFill()
	}
	ctx.DrawPath(path, paintengine2d.Fill(fill))
	line := func(ax, ay, bx, by float32, col paintengine2d.Color) {
		ctx.DrawLine(paintengine2d.Pt(ax, ay), paintengine2d.Pt(bx, by), paintengine2d.StrokePaint(col, 1))
	}
	line(tx+0.5, ty+th-pt, tx+0.5, ty+0.5, c.hi)
	line(tx+0.5, ty+0.5, tx+tw-0.5, ty+0.5, c.hi)
	line(tx+tw-0.5, ty+0.5, tx+tw-0.5, ty+th-pt, c.dk)
	line(tx+tw-0.5, ty+th-pt, tx+tw*0.5, ty+th-0.5, c.dk)
	line(tx+0.5, ty+th-pt, tx+tw*0.5, ty+th-0.5, c.hi)
	line(tx+tw-1.5, ty+1.5, tx+tw-1.5, ty+th-pt, c.shadow)
	if st.Focused() {
		DottedRect(ctx, b.Inset(1), c.text)
	}
}

func (e win95Engine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	// Windows had no switch; draw it as a latching slot with a raised knob.
	c := w95colors(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	track := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-th)*0.5), tw, th)
	fill := c.field
	if on && !st.Disabled() {
		fill = c.sel
	}
	if st.Disabled() {
		fill = c.face
	}
	ctx.DrawRect(track, paintengine2d.Fill(fill))
	c.sunken(ctx, track)
	kw := th - l.S(4)
	kx := track.Min.X + l.S(2)
	if on {
		kx = track.Max.X - l.S(2) - kw
	}
	knob := paintengine2d.XYWH(kx, track.Min.Y+l.S(2), kw, th-l.S(4))
	ctx.DrawRect(knob, paintengine2d.Fill(c.face))
	c.raised(ctx, knob)
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.gray
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
		if st.Focused() {
			DottedRect(ctx, labelFocusRect(l.body, label, lb, b), c.text)
		}
	}
}

func (e win95Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	c := w95colors(l)
	m := l.metrics
	indent := m.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(6)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	// Dotted connector lines, one per ancestor level plus the elbow.
	for d := 0; d <= depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
		y1 := b.Max.Y
		if d == depth {
			y1 = cy
		}
		for y := snap(b.Min.Y); y < y1; y += 2 {
			ctx.DrawRect(paintengine2d.XYWH(gx, y, 1, 1), paintengine2d.Fill(c.shadow))
		}
		if d == depth {
			for xx := gx; xx < x+indent+l.S(2); xx += 2 {
				ctx.DrawRect(paintengine2d.XYWH(xx, cy, 1, 1), paintengine2d.Fill(c.shadow))
			}
		}
	}
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + indent + l.S(4)
	tw := f.Advance(label) + l.S(4)
	lb := paintengine2d.XYWH(lx-l.S(2), b.Min.Y+l.S(1), tw, b.Dy()-l.S(2))
	fg := c.text
	if selected {
		ctx.DrawRect(lb, paintengine2d.Fill(c.sel))
		fg = c.selTxt
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

func (e win95Engine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	lb := b
	if st.Pressed() && !st.Disabled() {
		ctx.DrawRect(b.Inset(0.5), paintengine2d.StrokePaint(c.shadow, 1))
		lb = lb.Translate(paintengine2d.Pt(1, 1))
	} else {
		c.raised(ctx, b)
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(2), lb.Min.Y, aw, lb.Dy()), dir, c.shadow)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()), c.text, AlignStart, 0)
}

func (e win95Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	EtchedLine(ctx, b.Min.X, b.Min.Y, b.Dx(), false, c.shadow, c.hi)
}

func (e win95Engine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	if len(parts) == 0 {
		return
	}
	grip := l.S(14)
	slot := (b.Dx() - grip) / float32(len(parts))
	gap := l.S(2)
	for i, s := range parts {
		r := paintengine2d.XYWH(b.Min.X+slot*float32(i)+gap, b.Min.Y+gap, slot-gap*2, b.Dy()-gap*2)
		c.thin(ctx, r, false)
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(r.Min.X+l.S(4), r.Min.Y, r.Dx()-l.S(8), r.Dy()), c.text, AlignStart, 0)
	}
	// Size grip: three diagonal white/shadow line pairs.
	gx, gy := b.Max.X, b.Max.Y
	for i := 0; i < 3; i++ {
		o := l.S(4) * float32(i+1)
		ctx.DrawLine(paintengine2d.Pt(gx-o, gy-1), paintengine2d.Pt(gx-1, gy-o), paintengine2d.StrokePaint(c.shadow, 1))
		ctx.DrawLine(paintengine2d.Pt(gx-o+1, gy-1), paintengine2d.Pt(gx-1, gy-o+1), paintengine2d.StrokePaint(c.hi, 1))
	}
}

func (e win95Engine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	// Panel headings and the app title strip: face with an etched rule.
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	EtchedLine(ctx, b.Min.X, b.Max.Y-2, b.Dx(), false, c.shadow, c.hi)
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2), c.gray, AlignStart, 0)
	}
}

func (e win95Engine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.face))
	if raised {
		c.thin(ctx, b, true)
	}
}

func (e win95Engine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := w95colors(l)
	e.Face(l, ctx, b, RoleButton, st&^StateFocused)
	lb := b
	if st.Pressed() {
		lb = lb.Translate(paintengine2d.Pt(1, 1))
	}
	e.Expander(l, ctx, paintengine2d.XYWH(lb.Min.X+l.S(4), lb.Min.Y, l.S(14), lb.Dy()), expanded, c.text)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(lb.Min.X+l.S(24), lb.Min.Y, lb.Dx()-l.S(28), lb.Dy()), c.text, AlignStart, 0)
	if st.Focused() {
		DottedRect(ctx, b.Inset(l.S(3)), c.text)
	}
}

func (win95Engine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := w95colors(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X) * 0.5)
		EtchedLine(ctx, x-1, b.Min.Y+l.S(2), b.Dy()-l.S(4), true, c.shadow, c.hi)
		return
	}
	y := snap((b.Min.Y + b.Max.Y) * 0.5)
	EtchedLine(ctx, b.Min.X, y-1, b.Dx(), false, c.shadow, c.hi)
}

func (win95Engine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	ctx.DrawRect(b, paintengine2d.Fill(w95colors(l).face))
}

func (win95Engine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := w95colors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.info))
	ctx.DrawRect(b.Inset(0.5), paintengine2d.StrokePaint(c.infoTxt, 1))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.infoTxt, AlignStart, 0)
}

// snap rounds to the pixel grid so 1px lines stay crisp.
func snap(v float32) float32 {
	if v < 0 {
		return float32(int(v - 0.5))
	}
	return float32(int(v + 0.5))
}

// ---- packs --------------------------------------------------------------------

func win95Pack(name, label string, year int, summary string, fam ThemeName, pal Palette, extra map[string]string, params map[string]float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "win95",
		Bevel:   BevelClassic3D,
		Family:  fam,
		Palette: pal,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Params = params
	tok.Hot = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Windows", Summary: summary,
		Era: "Windows classic", Palette: fam, Tokens: tok,
	}
}

// w95Palette builds a classic palette from the scheme's system colours.
func w95Palette(face, text, field, fieldText, sel, selText, gray, accent string) Palette {
	f := hexColor(face)
	return Palette{
		Background: f, Surface: f, SurfaceAlt: f,
		Border: hexColor("#000000"), Divider: Shade(f, -0.33),
		Text: hexColor(text), TextMuted: hexColor(gray), TextOnAccent: hexColor(selText),
		Accent: hexColor(accent), AccentHover: hexColor(accent), AccentPress: hexColor(accent),
		Field: hexColor(field), FieldBorder: Shade(f, -0.33),
		Focus: hexColor(text), Selection: hexColor(sel),
		Track: Mix(f, hexColor("#ffffff"), 0.5), Thumb: f,
		Highlight: hexColor("#ffffff"), Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: hexColor(sel), MenuHoverBorder: hexColor(sel), MenuGutter: f,
		Danger: hexColor("#c00000"), Success: hexColor("#008000"), Warning: hexColor("#c08000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.30),
		BevelLight: hexColor("#ffffff"), BevelDark: Shade(f, -0.33),
	}
}

func win95Packs() []ThemePack {
	std := w95Palette("#c0c0c0", "#000000", "#ffffff", "#000000", "#000080", "#ffffff", "#808080", "#000080")
	w2k := w95Palette("#d4d0c8", "#000000", "#ffffff", "#000000", "#0a246a", "#ffffff", "#808080", "#0a246a")
	hot := w95Palette("#ffff00", "#000000", "#ffffff", "#000000", "#000000", "#ffffff", "#808080", "#ff0000")
	hc := w95Palette("#000000", "#ffffff", "#000000", "#ffffff", "#800080", "#ffffff", "#00ff00", "#00ffff")
	dark := w95Palette("#3c3c3c", "#e6e6e6", "#1f1f1f", "#e6e6e6", "#1c3f94", "#ffffff", "#9a9a9a", "#6f9bff")
	dark.Danger, dark.Success, dark.Warning = hexColor("#ff6b6b"), hexColor("#5fd35f"), hexColor("#f0c040")
	hc.Danger, hc.Success, hc.Warning = hexColor("#ff4040"), hexColor("#00ff00"), hexColor("#ffff00")
	return []ThemePack{
		win95Pack("win95", "Windows 95", 1995, "The four-colour bevel, navy captions, dotted focus.", ThemeLight, std,
			map[string]string{"hi": "#ffffff", "light": "#dfdfdf", "shadow": "#808080", "dk": "#000000",
				"caption": "#000080", "captionOff": "#808080", "captionOffText": "#c0c0c0"}, nil),
		win95Pack("win98", "Windows 98", 1998, "Windows 95 with gradient captions and flat hot-tracked menus.", ThemeLight, std,
			map[string]string{"hi": "#ffffff", "light": "#dfdfdf", "shadow": "#808080", "dk": "#000000",
				"caption": "#000080", "caption2": "#1084d0", "captionOff": "#808080", "captionOff2": "#b5b5b5", "captionOffText": "#c0c0c0"},
			map[string]float32{"flatMenuBar": 1}),
		win95Pack("win2000", "Windows 2000", 2000, "Warm grey #d4d0c8, blue gradient captions.", ThemeLight, w2k,
			map[string]string{"hi": "#ffffff", "light": "#d4d0c8", "shadow": "#808080", "dk": "#404040",
				"caption": "#0a246a", "caption2": "#a6caf0", "captionOff": "#808080", "captionOff2": "#c0c0c0", "captionOffText": "#d4d0c8"},
			map[string]float32{"flatMenuBar": 1}),
		win95Pack("win-hotdog", "Hot Dog Stand", 1992, "The infamous Windows 3.1 scheme: red, yellow and no mercy.", ThemeLight, hot,
			map[string]string{"hi": "#ffffff", "light": "#ffff80", "shadow": "#808000", "dk": "#000000",
				"caption": "#ff0000", "captionText": "#ffffff", "captionOff": "#ffffff", "captionOffText": "#000000"}, nil),
		win95Pack("win-highcontrast", "High Contrast Black", 1995, "Accessibility scheme: black, white, cyan and green.", ThemeDark, hc,
			map[string]string{"hi": "#ffffff", "light": "#c0c0c0", "shadow": "#808080", "dk": "#ffffff",
				"caption": "#800080", "captionOff": "#008000", "info": "#000000", "infoText": "#ffffff"}, nil),
		win95Pack("win95-dark", "Windows 95 Dark", 1995, "Classic 95 shapes in a dark grey scheme.", ThemeDark, dark,
			map[string]string{"hi": "#707070", "light": "#545454", "shadow": "#222222", "dk": "#0a0a0a",
				"caption": "#1c3f94", "caption2": "#3f6fd0", "captionOff": "#2c2c2c", "captionOff2": "#444444", "captionOffText": "#9a9a9a",
				"info": "#2b2b2b", "infoText": "#e6e6e6"},
			map[string]float32{"flatMenuBar": 1}),
	}
}
