package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Controls of the web engine: buttons, toggles, ranges, fields, combos and
// spin boxes (see engine_web.go).

func (e webEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := webColors(l)
	f := c.face(l, b)
	r := min(c.rad(l, c.radius), f.Dy()*0.5)
	c.restShadowUnder(l, ctx, winSnap(b), f, r, st)
	fg := c.button(l, ctx, f, st)
	l.drawFittedText(ctx, c.btnFace, label, c.pressed(f, st), fg, AlignCenter, l.S(16))
	if st.Focused() && !st.Disabled() {
		c.focusRing(l, ctx, b, f, r)
		if st.Primary() && c.focusStyle == webFocusInside {
			// A light line inside the ring keeps it apart from the fill
			// (Primer's primary buttons).
			if _, w := c.ring(l); f.Dx() > 4*w && f.Dy() > 4*w {
				winRing(ctx, f.Inset(w), max(r-w, 0), c.px(l), paintengine2d.Fill(c.onPrimary))
			}
		}
	}
}

func (e webEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := webColors(l)
	f := winSnap(b)
	if !st.AutoRaise() && !st.Toggle() {
		// A free tool button is a push button: it keeps the focus margin.
		f = c.face(l, b)
	}
	fg := c.toolFace(l, ctx, f, st)
	winToolContent(l, ctx, b, label, icon, fg)
	if st.Focused() && !st.Disabled() {
		c.focusRing(l, ctx, b, f, min(c.rad(l, c.radius), f.Dy()*0.5))
	}
}

// ---- check boxes, radios, switches --------------------------------------------------------------

// toggleCell is a check box or radio's cell at the left of b (the Checkbox
// metric square, room for a focus ring round the indicator) and the
// indicator of side s centred in it.
func (c *webSet) toggleCell(l *Classic, b paintengine2d.Rect, side, s float32) (cell, g paintengine2d.Rect) {
	side = min(side, b.Dy(), b.Dx())
	cell = winSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	s = snap(min(l.S(s), cell.Dx(), cell.Dy()))
	return cell, flatCentered(cell, s, s)
}

// toggleLabel draws a check box, radio or switch label after the glyph g.
func (c *webSet) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, g paintengine2d.Rect, st ControlState, label string) {
	if label == "" {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	x := g.Max.X + l.S(8)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), fg, AlignStart, 0)
}

// toggleFocus rings a focused check box, radio or switch glyph g (corner r)
// inside the control's rect b: round it where the cell leaves room, else
// over its edge with a light line inside so it shows on a filled glyph.
func (c *webSet) toggleFocus(l *Classic, ctx *paintengine2d.Context, b, g paintengine2d.Rect, r float32) {
	col := webA(c.focus, c.focusA)
	if c.focusStyle == webFocusDotted {
		webDots(ctx, g.Inset(-winPx(l)).Intersect(winSnap(b)), c.text, winPx(l))
		return
	}
	gap, w := c.ring(l)
	if c.focusStyle == webFocusInside {
		gap = max(gap, c.px(l))
	}
	outer := g.Inset(-(gap + w))
	ctx.Save()
	ctx.ClipRect(b)
	if outer.Min.Y >= b.Min.Y-0.01 && outer.Max.Y <= b.Max.Y+0.01 && outer.Min.X >= b.Min.X-0.01 {
		webBand(ctx, outer, r+gap+w, g.Inset(-gap), r+gap, col)
	} else if g.Dx() > 4*w {
		winRing(ctx, g, r, w, paintengine2d.Fill(col))
		winRing(ctx, g.Inset(w), max(r-w, 0), c.px(l), paintengine2d.Fill(c.window))
	}
	ctx.Restore()
}

func (e webEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := webColors(l)
	_, g := c.toggleCell(l, b, l.metrics.Checkbox, c.checkSize)
	if g.Dx() >= 4 {
		c.checkBox(l, ctx, g, st, checked)
	}
	c.toggleLabel(l, ctx, b, g, st, label)
	if st.Focused() && !st.Disabled() && !c.checkFocus && g.Dx() >= 4 {
		c.toggleFocus(l, ctx, b, g, min(c.rad(l, c.checkR), g.Dx()*0.5))
	}
}

func (e webEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := webColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	_, g := c.toggleCell(l, b, side, c.radioSize)
	if g.Dx() >= 4 {
		c.radio(l, ctx, g, st, selected)
	}
	c.toggleLabel(l, ctx, b, g, st, label)
	if st.Focused() && !st.Disabled() && !c.checkFocus && g.Dx() >= 4 {
		c.toggleFocus(l, ctx, b, g, g.Dx()*0.5)
	}
}

// DrawSwitch is a toggle switch: a pill (or rounded) track, the accent when
// on, with a round knob; off it is the pack's off track (Avalonia's is an
// outline round a dark knob, the web systems' a grey fill round a white one).
func (e webEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := webColors(l)
	m := c.reach(l)
	th := min(l.metrics.SwitchH, b.Dy()-2*m)
	tw := min(l.metrics.SwitchW, b.Dx()-2*m)
	track := winSnap(paintengine2d.XYWH(b.Min.X+m, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	px := c.px(l)
	if track.Dx() < 3*track.Dy()*0.5 || track.Dy() < 6*px {
		c.toggleLabel(l, ctx, b, paintengine2d.XYWH(b.Min.X, b.Min.Y, max(track.Dx(), 0), b.Dy()), st, label)
		return
	}
	r := track.Dy() * 0.5
	if !c.pill {
		r = min(c.rad(l, c.switchR), r)
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	var fill, border, knob paintengine2d.Color
	if on {
		fill, border, knob = c.switchOn, paintengine2d.Color{}, c.knobOn
		switch {
		case st.Pressed() && !st.Disabled():
			fill = c.shade(c.switchOn, c.accentPress)
		case hot:
			fill = c.shade(c.switchOn, c.accentHover)
		}
	} else {
		fill, border, knob = c.switchOff, c.switchOffBorder, c.knobOff
		if hot {
			if border.A > 0 {
				border = webOver(c.window, Mix(c.flat(border), c.text, 0.25))
			} else {
				fill = webOver(c.flat(fill), webA(c.text, 0.06))
			}
		}
	}
	fill, border, knob = c.disabled(fill, st), c.disabled(border, st), c.disabled(knob, st)
	c.box(l, ctx, track, r, fill, border)
	d := track.Dy() - 2*snap(l.S(2))
	if c.knob > 0 {
		d = min(snap(l.S(c.knob)), track.Dy())
	}
	cy := (track.Min.Y + track.Max.Y) * 0.5
	cx := track.Min.X + track.Dy()*0.5
	if on {
		cx = track.Max.X - track.Dy()*0.5
	}
	kb := paintengine2d.XYWH(cx-d*0.5, cy-d*0.5, d, d)
	kr := d * 0.5
	if !c.pill {
		kr = max(min(r-(track.Dy()-d)*0.5, d*0.5), 0)
	}
	if knob.A > 0 {
		ctx.DrawRoundRect(kb, kr, kr, paintengine2d.Fill(knob))
		if c.knobBorder.A > 0 {
			kc := c.knobBorder
			if on {
				kc = c.switchOn
			}
			winRing(ctx, kb, kr, px, paintengine2d.Fill(c.disabled(kc, st)))
		}
	}
	c.toggleLabel(l, ctx, b, track, st, label)
	if st.Focused() && !st.Disabled() {
		c.toggleFocus(l, ctx, b, track, r)
	}
}

// shade is the pack's hover or pressed shade of a colour that is its accent,
// else the colour itself nudged towards the text.
func (c *webSet) shade(col, accentShade paintengine2d.Color) paintengine2d.Color {
	if nearColor(col, c.accent) {
		return accentShade
	}
	return Mix(col, c.text, 0.1)
}

// ---- sliders and progress -----------------------------------------------------------------------

// SliderTravel is where the thumb's centre runs: inside the focus margin.
func (webEngine) SliderTravel(l *Classic, b paintengine2d.Rect) (x0, x1 float32) {
	c := webColors(l)
	half := snap(l.metrics.Thumb)*0.5 + c.reach(l)
	return b.Min.X + half, b.Max.X - half
}

// DrawSlider is a thin rounded track, the accent up to the thumb: a filled
// disc (Avalonia's accent thumb) or a white disc in a border.
func (e webEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := webColors(l)
	b = winSnap(b)
	t = clamp1(t)
	m := c.reach(l)
	d := min(snap(l.metrics.Thumb), b.Dy()-2*m)
	if d < 4 || b.Dx() < d+2*m+4 {
		return
	}
	x0, x1 := e.SliderTravel(l, b)
	tx := x0 + (x1-x0)*t
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	th := min(max(snap(l.S(c.trackH)), 1), d)
	tr := paintengine2d.XYWH(b.Min.X+m, cy-snap(th*0.5), b.Dx()-2*m, th)
	ctx.DrawRoundRect(tr, th*0.5, th*0.5, paintengine2d.Fill(c.disabled(c.track, st)))
	if w := tx - tr.Min.X; w > 0 {
		ctx.DrawRoundRect(paintengine2d.XYWH(tr.Min.X, tr.Min.Y, w, th), th*0.5, th*0.5, paintengine2d.Fill(c.disabled(c.trackFill, st)))
	}
	ctr := paintengine2d.Pt(tx, cy)
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(b)
	if c.thumbStyle == 1 {
		if !st.Disabled() {
			DropShadow(ctx, paintengine2d.XYWH(tx-d*0.5, cy-d*0.5, d, d), d*0.5, paintengine2d.RGBA(0, 0, 0, 0.12), 0, l.S(0.5), l.S(1.5), 0)
		}
		ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(c.disabled(c.thumbBorder, st)))
		ctx.DrawCircle(ctr, d*0.5-c.px(l), paintengine2d.Fill(c.disabled(c.thumbFill, st)))
	} else {
		fill := c.thumbFill
		switch {
		case st.Pressed() && !st.Disabled():
			fill = c.shade(fill, c.accentPress)
		case hot:
			fill = c.shade(fill, c.accentHover)
		}
		ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(c.disabled(fill, st)))
	}
	if st.Focused() && !st.Disabled() {
		c.toggleFocus(l, ctx, b, paintengine2d.XYWH(tx-d*0.5, cy-d*0.5, d, d), d*0.5)
	}
}

// DrawProgressBar is a thin rounded bar over its track; the busy bar slides
// a segment.
func (webEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := webColors(l)
	b = winSnap(b)
	h := min(max(snap(l.S(c.progressH)), 1), b.Dy())
	if b.Dx() < 4 || h < 1 {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := h * 0.5
	ctx.DrawRoundRect(bar, r, r, paintengine2d.Fill(c.disabled(c.progressTrack, st)))
	fill := c.disabled(c.progress, st)
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.3
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		if seg := paintengine2d.XYWH(x, bar.Min.Y, span, h).Intersect(bar); seg.Dx() >= 1 {
			ctx.DrawRoundRect(seg, r, r, paintengine2d.Fill(fill))
		}
		return
	}
	if w := snap(bar.Dx() * clamp1(t)); w >= 1 {
		ctx.DrawRoundRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, r, paintengine2d.Fill(fill))
	}
}

// ---- fields -------------------------------------------------------------------------------------

func (e webEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	c := webColors(l)
	f := c.face(l, b)
	r := min(c.rad(l, c.fieldR), f.Dy()*0.5)
	c.restShadowUnder(l, ctx, winSnap(b), f, r, st)
	c.fieldFrame(l, ctx, f, st, c.fieldR)
	if st.Focused() && !st.Disabled() && c.focusStyle == webFocusOutside {
		c.fieldFocusRing(l, ctx, b, f, r)
	}
	l.baseDrawTextField(ctx, f, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

// DrawTextArea is the field's frame (through Face) with the text inside it.
func (e webEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := webColors(l)
	f := c.face(l, b)
	if st.Focused() && !st.Disabled() && c.focusStyle == webFocusOutside {
		c.fieldFocusRing(l, ctx, b, f, min(c.rad(l, c.fieldR), f.Dy()*0.5))
	}
	l.baseDrawTextArea(ctx, f, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face)
}

// comboArrowW is the chevron's column at a combo's right.
func (c *webSet) comboArrowW(l *Classic, f paintengine2d.Rect) float32 {
	return min(snap(l.S(24)), f.Dx()*0.4)
}

// DrawComboBox is a drop-down: the field's face (the control radius) with
// a chevron at the right; an editable one is a field (its text is the
// frameless editor's).
func (e webEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := webColors(l)
	f := c.face(l, b)
	r := min(c.rad(l, c.comboR), f.Dy()*0.5)
	fst := st
	if open {
		fst |= StatePressed
	}
	if c.focusStyle == webFocusDotted && !st.Editable() {
		// A drop-list shows focus with the adorner, not the field's border.
		fst &^= StateFocused
	}
	c.restShadowUnder(l, ctx, winSnap(b), f, r, st)
	c.fieldFrame(l, ctx, f, fst, c.comboR)
	if f.Dx() < 8 || f.Dy() < 6 {
		return
	}
	aw := c.comboArrowW(l, f)
	ab := paintengine2d.XYWH(f.Max.X-aw-l.S(2), f.Min.Y, aw, f.Dy())
	col := c.text2
	if st.Disabled() {
		col = c.textDis
	}
	e.Arrow(l, ctx, ab, DirDown, col)
	if !st.Editable() {
		fg := c.text
		if st.Disabled() {
			fg = c.textDis
		}
		pad := l.metrics.FieldPad
		l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(f.Min.X+pad, f.Min.Y, ab.Min.X-f.Min.X-pad, f.Dy()), fg, AlignStart, 0)
	}
	if (st.Focused() && !open || open && webIDE(l).openRing) && !st.Disabled() {
		if c.focusStyle == webFocusOutside || (c.focusStyle == webFocusDotted && !st.Editable()) {
			c.fieldFocusRing(l, ctx, b, f, r)
		}
	}
}

// ComboTextRect: an editable combo's editor fills the face up to the
// chevron; the editor keeps the field padding, so its text starts where a
// drop-list's does.
func (webEngine) ComboTextRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	c := webColors(l)
	f := c.face(l, b)
	px := c.px(l)
	aw := c.comboArrowW(l, f)
	return paintengine2d.XYWH(f.Min.X, f.Min.Y+px, max(0, f.Dx()-aw-l.S(2)), max(0, f.Dy()-2*px))
}

// DrawSpinner: the step buttons stacked at the right inside the field's
// border (a hairline against the text), a wash under the pointer.
func (e webEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	if b.Dx() < 4*px || b.Dy() < 6*px {
		return
	}
	if !st.Frameless() {
		f := c.face(l, b)
		c.fieldFrame(l, ctx, f, st&^StateFocused, c.fieldR)
		b = f.Inset(px)
	} else if m := c.reach(l); m > 0 {
		// Inside a field whose face keeps a focus margin: stay in the face.
		fi := webEngine{}.ViewFrameInsets(l)
		if k := m + px - fi.Top; k > 0 && b.Dy() > 2*k+4*px && b.Dx() > k+4*px {
			b = paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+k), Max: paintengine2d.Pt(b.Max.X-k, b.Max.Y-k)}
		}
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+snap(l.S(3)), px, max(b.Dy()-2*snap(l.S(3)), 0)), paintengine2d.Fill(c.border2))
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	r := min(c.rad(l, c.radius)*0.5, l.S(3))
	half := func(h paintengine2d.Rect, dir Direction, hover, press bool) {
		in := h.Inset(snap(l.S(2)))
		col := c.text2
		switch {
		case st.Disabled():
			col = c.textDis
		case press:
			if !in.Empty() {
				ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(webA(c.wash, 1.8)))
			}
			col = c.text
		case hover:
			if !in.Empty() {
				ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.wash))
			}
			col = c.text
		}
		webChevron(ctx, h, dir, l.S(7), l.S(3.5), max(l.S(1.2), 1), col)
	}
	half(paintengine2d.XYWH(b.Min.X+px, b.Min.Y, b.Dx()-px, mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X+px, mid, b.Dx()-px, b.Max.Y-mid), DirDown, downHover, downPress)
}
