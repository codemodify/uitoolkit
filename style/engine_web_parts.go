package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Parts, scroll bars, frames and behaviour of the web engine (see
// engine_web.go for the engine and its params).

// ---- faces --------------------------------------------------------------------------------------

func (e webEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := webColors(l)
	switch role {
	case RoleButton:
		return c.button(l, ctx, winSnap(b), st)
	case RoleTool:
		return c.toolFace(l, ctx, winSnap(b), st)
	case RoleField:
		c.fieldFrame(l, ctx, winSnap(b), st, c.fieldR)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleCombo:
		c.fieldFrame(l, ctx, winSnap(b), st, c.comboR)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		return c.row(l, ctx, winSnap(b), st, webList)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
			return c.menuText
		}
	case RoleThumb:
		col := c.scrollThumb
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			col = c.scrollThumbHot
		}
		b = winSnap(b)
		r := min(b.Dx(), b.Dy()) * 0.5
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
	case RoleTrack:
		if c.scrollTrack.A > 0 {
			ctx.DrawRect(b, paintengine2d.Fill(c.scrollTrack))
		}
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.window))
	}
	if st.Disabled() {
		return c.textDis
	}
	return c.text
}

// button paints a push button's face f (a primary button in the primary
// colour) and returns its label colour.
func (c *webSet) button(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState) paintengine2d.Color {
	f = c.pressed(f, st)
	if f.Dx() < 2 || f.Dy() < 2 {
		return c.btnText
	}
	r := c.rad(l, c.radius)
	var fill, border, fg paintengine2d.Color
	if st.Primary() {
		fill, border, fg = c.primary, c.primaryBorder, c.onPrimary
		switch {
		case st.Disabled():
			fill, border, fg = webA(fill, c.disA), webA(border, c.disA), webA(fg, max(c.disA, 0.65))
		case st.Pressed():
			fill = c.primaryPress
		case st.Hovered():
			fill = c.primaryHover
		}
	} else {
		fill, border, fg = c.btn, c.btnBorder, c.btnText
		switch {
		case st.Disabled():
			border, fg = webA(border, c.disA), c.textDis
		case st.Pressed():
			fill = c.btnPress
		case st.Hovered():
			fill = c.btnHover
		case st.Toggle() && st.Checked():
			fill = c.btnPress
		}
	}
	c.box(l, ctx, f, r, fill, border)
	return fg
}

// restShadow paints the faint resting shadow under a face f, where the
// control's focus margin leaves room for it inside b.
func (c *webSet) restShadowUnder(l *Classic, ctx *paintengine2d.Context, b, f paintengine2d.Rect, r float32, st ControlState) {
	if c.restShadow <= 0 || st.Disabled() || b.Max.Y-f.Max.Y < l.S(2) {
		return
	}
	ctx.Save()
	ctx.ClipRect(b)
	DropShadow(ctx, f, r, paintengine2d.RGBA(0, 0, 0, c.restShadow), 0, l.S(1), l.S(2), 0)
	ctx.Restore()
}

// fieldFrame paints a text field or combo face on f: the field fill in the
// input border, which takes the focus colour when focused (the combo's
// while its list is open) and the accent under the pointer where the pack
// says so; a focus band over it for the inside focus style.
func (c *webSet) fieldFrame(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState, radius float32) {
	if f.Dx() < 2 || f.Dy() < 2 {
		return
	}
	r := c.rad(l, radius)
	fill, border := c.field, c.border1
	focused := (st.Focused() || st.Pressed()) && !st.Disabled()
	switch {
	case st.Disabled():
		fill, border = c.fieldDis, webA(c.border1, c.disA)
	case focused:
		border = c.fieldFocus
	case st.Hovered() && c.hoverBorder:
		border = c.accent
	}
	c.box(l, ctx, f, r, fill, border)
	if focused && c.focusStyle == webFocusInside {
		if _, w := c.ring(l); w > c.px(l) && f.Dx() > 2*w && f.Dy() > 2*w {
			winRing(ctx, f, min(r, f.Dy()*0.5), w, paintengine2d.Fill(webA(c.fieldFocus, c.focusA)))
		}
	}
}

// toolFace paints a tool button's face on f and returns its glyph colour. In
// a tool bar (StateAutoRaise) the button is flat until the pointer is over
// it; a latched one is the accent glyph (SourceGit) or a wash. A free toggle
// (a segment) wears the pack's segment look when on; a free tool button is
// a push button.
func (c *webSet) toolFace(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if f.Dx() < 2 || f.Dy() < 2 {
		return c.text
	}
	r := min(c.rad(l, c.radius), f.Dy()*0.5)
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	latched := st.Toggle() && st.Checked()
	switch {
	case latched && !st.AutoRaise():
		switch c.seg {
		case 0:
			c.box(l, ctx, f, f.Dy()*0.5, c.disabled(c.accent, st), paintengine2d.Color{})
			return c.disabled(c.onAccent, st)
		case 2:
			c.box(l, ctx, f, r, c.disabled(c.segTrack, st), paintengine2d.Color{})
			return c.disabled(c.segOnText, st)
		}
		if !st.Disabled() {
			ctx.Save()
			ctx.ClipRect(f)
			DropShadow(ctx, f.Inset(c.px(l)), r, paintengine2d.RGBA(0, 0, 0, 0.08), 0, c.px(l)*0.5, l.S(1.5), 0)
			ctx.Restore()
		}
		c.box(l, ctx, f, r, c.disabled(c.segOn, st), c.disabled(c.border2, st))
		return c.disabled(c.segOnText, st)
	case latched:
		if c.toolWash {
			c.box(l, ctx, f, r, webA(c.wash, 1.6), paintengine2d.Color{})
			return c.disabled(c.text, st)
		}
		return c.disabled(c.accent, st)
	case st.AutoRaise() || st.Toggle():
		if st.Disabled() {
			return c.textDis
		}
		if hot && c.toolWash {
			a := float32(1)
			if st.Pressed() {
				a = 1.8
			}
			c.box(l, ctx, f, r, webA(c.wash, a), paintengine2d.Color{})
		}
		if st.Toggle() && !st.AutoRaise() {
			if hot {
				return c.text
			}
			return c.text2
		}
		if hot {
			return c.text
		}
		return webA(c.text, c.toolAlpha)
	}
	return c.button(l, ctx, f, st&^StatePrimary)
}

// ---- toggles ------------------------------------------------------------------------------------

func (e webEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := webColors(l)
	s := snap(min(l.S(c.checkSize), box.Dx(), box.Dy()))
	if s < 4 {
		return
	}
	c.checkBox(l, ctx, flatCentered(box, s, s), st, checked)
}

// checkBox paints a check box of face g: SourceGit's unfilled box with an
// accent tick, or a box filled with the check colour and a white tick.
func (c *webSet) checkBox(l *Classic, ctx *paintengine2d.Context, g paintengine2d.Rect, st ControlState, on bool) {
	px := c.px(l)
	s := g.Dx()
	r := min(c.rad(l, c.checkR), s*0.5)
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	keyFocus := st.Focused() && !st.Disabled() && c.checkFocus
	border := c.checkBorder
	if hot && c.hoverBorder {
		border = c.accent
	}
	if c.checkStyle == 0 {
		fill, mark, bw := c.checkOff, c.checkMark, px
		if keyFocus {
			border, bw = c.accent, 2*px
			if on {
				fill, mark = c.accent, c.onAccent
			}
		}
		fill, border, mark = c.disabled(fill, st), c.disabled(border, st), c.disabled(mark, st)
		if fill.A > 0 {
			ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(fill))
		}
		winRing(ctx, g, r, bw, paintengine2d.Fill(border))
		if on {
			// A 12px tick in the 16px box, a line below the centre.
			t := flatCentered(g, s*0.75, s*0.75).Translate(paintengine2d.Pt(0, snap(s/16)))
			webTick(ctx, t, mark, max(l.S(1.75), 1.2))
		}
		return
	}
	if on {
		fill := c.checkOn
		if hot {
			fill = webOver(fill, webA(c.onAccent, 0.08))
		}
		c.box(l, ctx, g, r, c.disabled(fill, st), paintengine2d.Color{})
		webTick(ctx, g.Inset(s*0.14), c.disabled(c.checkMark, st), max(l.S(1.6), 1.2))
	} else {
		c.box(l, ctx, g, r, c.disabled(c.checkOff, st), c.disabled(border, st))
	}
	if keyFocus {
		winRing(ctx, g, r, 2*px, paintengine2d.Fill(c.accent))
	}
}

func (e webEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := webColors(l)
	d := snap(min(l.S(c.radioSize), box.Dx(), box.Dy()))
	if d < 4 {
		return
	}
	c.radio(l, ctx, flatCentered(box, d, d), st, selected)
}

// radio paints a radio of face g: a ring with the accent dot (SourceGit,
// shadcn) or a disc of the check colour round a white centre (Primer).
func (c *webSet) radio(l *Classic, ctx *paintengine2d.Context, g paintengine2d.Rect, st ControlState, on bool) {
	px := c.px(l)
	ctr := paintengine2d.Pt((g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5)
	R := g.Dx() * 0.5
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	keyFocus := st.Focused() && !st.Disabled() && c.checkFocus
	border := c.checkBorder
	if hot && c.hoverBorder {
		border = c.accent
	}
	dot := min(l.S(c.radioDot), g.Dx()-4*px) * 0.5
	ring := func(col paintengine2d.Color, w float32) {
		p := paintengine2d.NewPath()
		p.AddCircle(ctr, R)
		p.AddCircle(ctr, max(R-w, 0))
		ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
	}
	if c.radioStyle == 0 {
		if keyFocus {
			// SourceGit: the ring fills with the accent round a white dot (an
			// accent one, ringed in white, when selected).
			ctx.DrawCircle(ctr, R, paintengine2d.Fill(c.accent))
			if on {
				ctx.DrawCircle(ctr, dot, paintengine2d.Fill(c.onAccent))
				ctx.DrawCircle(ctr, max(dot-px, 0), paintengine2d.Fill(c.checkOn))
			} else {
				ctx.DrawCircle(ctr, dot, paintengine2d.Fill(c.onAccent))
			}
			return
		}
		if f := c.disabled(c.checkOff, st); f.A > 0 {
			ctx.DrawCircle(ctr, R, paintengine2d.Fill(f))
		}
		ring(c.disabled(border, st), px)
		if on {
			ctx.DrawCircle(ctr, dot, paintengine2d.Fill(c.disabled(c.checkOn, st)))
		}
		return
	}
	if on {
		ctx.DrawCircle(ctr, R, paintengine2d.Fill(c.disabled(c.checkOn, st)))
		ctx.DrawCircle(ctr, dot, paintengine2d.Fill(c.disabled(c.checkMark, st)))
	} else {
		ctx.DrawCircle(ctr, R, paintengine2d.Fill(c.disabled(c.checkOff, st)))
		ring(c.disabled(border, st), px)
	}
	if keyFocus {
		ring(c.accent, 2*px)
	}
}

// ---- glyphs -------------------------------------------------------------------------------------

// Arrow is the thin chevron of combos, spinners and submenus.
func (webEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	webChevron(ctx, b, dir, l.S(8), l.S(4), max(l.S(1.25), 1), col)
}

// Expander is SourceGit's solid triangle or a chevron.
func (webEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := webColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	if c.triangles {
		webTriangle(ctx, b, dir, l.S(8), col)
		return
	}
	webChevron(ctx, b, dir, l.S(8), l.S(4), max(l.S(1.3), 1), col)
}

// MenuHighlight is a menu's hot row: a rounded wash (or the accent).
func (webEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := webColors(l)
	b = winSnap(b)
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	r := min(c.rad(l, c.menuRowR), b.Dy()*0.5)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.menuHover))
}

func (webEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := webColors(l)
	if hot {
		return c.menuText
	}
	return c.text
}

// FieldFocusRing: fields show focus in their own face (the focus border and
// ring), so the base painters add none.
func (webEngine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is focus on an arbitrary rect b, inside it: the dotted
// adorner or a band of the focus colour.
func (webEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	b = winSnap(b)
	if b.Dx() < 6 || b.Dy() < 6 {
		return
	}
	if c.focusStyle == webFocusDotted {
		webDots(ctx, b, c.text, winPx(l))
		return
	}
	_, w := c.ring(l)
	winRing(ctx, b, min(c.rad(l, c.radius), b.Dy()*0.5), w, paintengine2d.Fill(webA(c.focus, c.focusA)))
}

// ---- scroll bars --------------------------------------------------------------------------------

// ScrollBarStyle: transient overlay bars as wide as the scroll metric, the
// thin idle thumb growing to the bar under the pointer; SourceGit's show
// arrow cells while expanded.
func (webEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	c := webColors(l)
	s := ScrollBarStyle{Overlay: true, Transient: true, MinThumb: 24, EndPad: 2}
	if c.scrollArrows {
		s.Arrows, s.ArrowLen, s.EndPad = ArrowsEnds, 12, 0.01
	}
	return s
}

func (e webEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := webColors(l)
	bar := winSnap(p.Bar)
	if bar.Empty() || st.Disabled {
		return
	}
	wide := st.Hovered || st.Pressed != ScrollNone || st.Hot != ScrollNone
	thick := bar.Dx()
	if !vertical {
		thick = bar.Dy()
	}
	across := func(r paintengine2d.Rect, w float32) paintengine2d.Rect {
		if vertical {
			return paintengine2d.XYWH(snap((bar.Min.X+bar.Max.X-w)*0.5), r.Min.Y, w, r.Dy())
		}
		return paintengine2d.XYWH(r.Min.X, snap((bar.Min.Y+bar.Max.Y-w)*0.5), r.Dx(), w)
	}
	if wide && c.scrollTrack.A > 0 {
		r := thick * 0.5
		ctx.DrawRoundRect(bar, r, r, paintengine2d.Fill(c.scrollTrack))
	}
	if wide {
		dec, inc := DirUp, DirDown
		if !vertical {
			dec, inc = DirLeft, DirRight
		}
		arrow := func(r paintengine2d.Rect, dir Direction, part ScrollPart) {
			if r.Empty() {
				return
			}
			col := c.scrollThumb
			if st.Hot == part || st.Pressed == part {
				col = c.scrollThumbHot
			}
			webTriangle(ctx, winSnap(r), dir, min(l.S(8), thick), col)
		}
		arrow(p.Dec, dec, ScrollDec)
		arrow(p.Inc, inc, ScrollInc)
	}
	if p.Thumb.Empty() {
		return
	}
	// The thumb keeps the pack's gap from the view's edge inside the bar.
	w := min(max(snap(l.S(c.scrollIdle)), 1), thick)
	if wide {
		w = max(thick-2*snap(l.S(c.scrollInset)), w)
	}
	t := across(winSnap(p.Thumb), w).Intersect(bar)
	if t.Empty() {
		return
	}
	col := c.scrollThumb
	if st.Hot == ScrollThumbPart || st.Pressed == ScrollThumbPart {
		col = c.scrollThumbHot
	}
	r := min(t.Dx(), t.Dy()) * 0.5
	ctx.DrawRoundRect(t, r, r, paintengine2d.Fill(col))
}

func (e webEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), winScrollState(st))
}

// ---- frames -------------------------------------------------------------------------------------

// webHeadH is the band a group's heading takes above its card.
func webHeadH(l *Classic) float32 { return snap(l.BoldFont().Height() + l.S(8)) }

// GroupBoxInsets: a titled group is a heading above a card.
func (webEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top += webHeadH(l)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

func (webEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := webColors(l)
	card := winSnap(b)
	if title != "" {
		hh := webHeadH(l)
		l.drawFittedText(ctx, c.bold, title, paintengine2d.XYWH(b.Min.X+l.S(2), b.Min.Y, b.Dx()-l.S(4), hh-l.S(4)), c.text, AlignStart, 0)
		card = paintengine2d.XYWH(card.Min.X, card.Min.Y+hh, card.Dx(), card.Dy()-hh)
	}
	c.box(l, ctx, card, c.rad(l, c.cardR), c.card, c.border0)
}

// DrawPanel: a raised panel is a card; a flat one the window.
func (webEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := webColors(l)
	if raised {
		c.box(l, ctx, winSnap(b), c.rad(l, c.cardR), c.card, c.border0)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.window))
}

func (webEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(webColors(l).window))
}

// captionH is an in-app window's caption band.
func (c *webSet) captionH(l *Classic) float32 {
	return snap(max(l.metrics.TitleBar, l.body.Height()+l.S(8)))
}

func (webEngine) WindowFrameInsets(l *Classic) Insets {
	c := webColors(l)
	px := c.px(l)
	return Insets{Top: c.captionH(l), Right: px, Bottom: px, Left: px}
}

// WindowCloseRect: SourceGit's square caption cell filling the strip at
// the right, or the web dialog's icon button.
func (webEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	h := c.captionH(l)
	if b.Dy() < h+2*px || b.Dx() < 4*h {
		return paintengine2d.Rect{}
	}
	if c.captionStyle == 0 {
		w := min(snap(l.S(c.captionBtn)), snap(b.Dx()*0.3))
		line := float32(0)
		if c.captionLine {
			line = px
		}
		return paintengine2d.XYWH(b.Max.X-px-w, b.Min.Y+px, w, h-px-line)
	}
	side := snap(min(l.S(28), h-l.S(8)))
	return paintengine2d.XYWH(b.Max.X-px-snap(l.S(10))-side, b.Min.Y+snap((h-side)*0.5), side, side)
}

// DrawWindowFrame is an in-app window: SourceGit's dialog (a caption strip
// in the title bar colour, the title centred in bold, the close cell that
// turns red, a 1px window border round 8px corners) or a web dialog (the
// popover's colour and border, a title at the left, an icon close button).
func (e webEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	h := c.captionH(l)
	if b.Dx() < 4*px || b.Dy() < 4*px {
		return
	}
	r := min(c.rad(l, c.windowR), b.Dx()*0.25, b.Dy()*0.25)
	fg := c.text
	if !st.Active {
		fg = c.text2
	}
	cb := paintengine2d.Rect{}
	if st.CanClose {
		cb = e.WindowCloseRect(l, b)
	}
	if c.captionStyle == 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.window))
		ctx.Save()
		ctx.ClipRoundRect(b.Inset(px), max(r-px, 0), max(r-px, 0))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), min(h, b.Dy())), paintengine2d.Fill(c.titleBar))
		if c.captionLine && b.Dy() > h+px {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+h-px, b.Dx(), px), paintengine2d.Fill(c.border0))
		}
		glyph := fg
		if !cb.Empty() && (st.CloseHot || st.ClosePress) {
			ctx.DrawRect(cb, paintengine2d.Fill(c.close))
			glyph = c.onClose
		}
		ctx.Restore()
		winRing(ctx, b, r, px, paintengine2d.Fill(c.windowBorder))
		if !cb.Empty() {
			s := snap(min(l.S(9), cb.Dy()*0.5))
			DrawCross(ctx, flatCentered(cb, s, s), glyph, max(l.S(1), 1))
		}
	} else {
		c.box(l, ctx, b, r, c.dialog, c.popupBorder)
		if c.captionLine && b.Dy() > h+px {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X+px, b.Min.Y+h-px, b.Dx()-2*px, px), paintengine2d.Fill(c.border2))
		}
		if !cb.Empty() {
			glyph := c.text2
			if st.CloseHot || st.ClosePress {
				a := float32(1)
				if st.ClosePress {
					a = 1.8
				}
				cr := min(c.rad(l, c.radius), cb.Dy()*0.5)
				ctx.DrawRoundRect(cb, cr, cr, paintengine2d.Fill(webA(c.wash, a)))
				glyph = c.text
			}
			if !st.Active {
				glyph = c.textDis
			}
			s := snap(min(l.S(10), cb.Dy()*0.45))
			DrawCross(ctx, flatCentered(cb, s, s), glyph, max(l.S(1.4), 1))
		}
	}
	if title == "" {
		return
	}
	right := b.Max.X - px - l.S(8)
	if !cb.Empty() {
		right = cb.Min.X - l.S(6)
	}
	x := b.Min.X + l.S(16)
	band := paintengine2d.XYWH(x, b.Min.Y+px, right-x, h-2*px)
	if c.captionCenter {
		// Centred on the window, kept clear of the close button.
		side := b.Max.X - right
		band = paintengine2d.XYWH(b.Min.X+side, b.Min.Y+px, b.Dx()-2*side, h-2*px)
		l.drawFittedText(ctx, c.bold, title, band, fg, AlignCenter, l.S(8))
		return
	}
	l.drawFittedText(ctx, c.bold, title, band, fg, AlignStart, 0)
}

// ---- shadows ------------------------------------------------------------------------------------

func (webEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	s, ok := webColors(l).shadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(0, s.dy, s.blur, s.spread)
}

func (webEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	c := webColors(l)
	s, ok := c.shadow(l, kind)
	if !ok {
		return
	}
	r := c.rad(l, c.overlayR)
	switch kind {
	case PopupTooltip:
		r = c.rad(l, c.tipR)
	case PopupDialog:
		r = c.rad(l, c.windowR)
	}
	// A blur wider than the (spread) layer would leave no core to fade
	// from: soften it to what the layer holds, within the declared reach.
	if lim := min(b.Dx(), b.Dy()) + 2*s.spread - 2; s.blur > lim {
		s.blur = max(lim, 0)
	}
	DropShadow(ctx, b, r, paintengine2d.RGBA(0, 0, 0, s.a), 0, s.dy, s.blur, s.spread)
}

// ---- views --------------------------------------------------------------------------------------

// ViewFrameInsets: the view's hairline, and room for its corner so the
// rows inside never cover it.
func (webEngine) ViewFrameInsets(l *Classic) Insets {
	c := webColors(l)
	h := c.px(l)
	v := h
	if r := c.rad(l, c.viewR); r > h {
		v = max(h, float32(math.Ceil(float64(r-(r-h)/math.Sqrt2))))
	}
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is a list, tree or table in the field colour inside the
// separator hairline; a sidebar is its pane, edge to edge.
func (webEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := webColors(l)
	if st.Sidebar() {
		if !b.Empty() {
			ctx.DrawRect(b, paintengine2d.Fill(c.sidebar))
		}
		return
	}
	b = winSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	fill := c.field
	if st.Disabled() {
		fill = c.fieldDis
	}
	c.box(l, ctx, b, c.rad(l, c.viewR), fill, c.border2)
}

// ViewBackground: a sidebar's rows sit on its pane, other views on the
// field colour.
func (webEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	c := webColors(l)
	if st.Sidebar() {
		return c.sidebar
	}
	return c.field
}

// ItemFocus marks the current row of a table (lists and trees mark theirs
// as they paint): the dotted adorner or a ring inside the row.
func (webEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	webColors(l).itemFocus(l, ctx, winSnap(b), st, webTable)
}

// ---- behaviour ----------------------------------------------------------------------------------

// StyleHint: the dialog button order is the design system's — SourceGit's
// popups put OK first, the web systems put the primary action last; menus
// underline mnemonics while Alt is held.
func (webEngine) StyleHint(l *Classic, h StyleHint) int {
	c := webColors(l)
	switch h {
	case HintDialogPrimaryFirst:
		if c.primaryFirst {
			return 1
		}
	case HintMnemonics:
		return MnemonicsOnAlt
	case HintHoverFadeMs:
		return int(l.P("hoverFade", 0))
	}
	return 0
}

// SpinBoxStyle: the step buttons sit inside the field's border, stacked.
func (webEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{Inside: true} }

// TabOutset: a selected browser tab's flares reach past its slot.
func (e webEngine) TabOutset(l *Classic) Insets {
	if webColors(l).tabStyle == webTabBrowser {
		return e.BrowserTabOutset(l)
	}
	return Insets{}
}

// ControlFont: bold or medium buttons and tabs, as the pack says.
func (webEngine) ControlFont(l *Classic, role Role) *Font {
	c := webColors(l)
	switch role {
	case RoleButton:
		return c.btnFace
	case RoleTab:
		return c.tabFace
	}
	return l.body
}
