package style

import "github.com/codemodify/paintengine2d"

// breeze6Engine paints KDE Plasma 6's Breeze (Plasma 6.0 to 6.7): the breeze
// engine's era for packs with param "plasma" 6, while the "breeze" packs stay
// Plasma 5.27 by design. It is written from the changes KDE published and
// their numbers, not from Breeze's source:
//
//   - frames round from 3 to 5px and check boxes from 2 to 4px (6.1);
//   - every outline is the frame contrast, 0.2, of the text colour mixed
//     into its background (0.25 and 0.3 before); a pressed button is the
//     highlight at 0.3 over the button colour, a checked box or radio the
//     same in a highlight outline, slider and progress values the
//     highlight at 0.7 over the window;
//   - keyboard focus adds a 2px band of the highlight at 0.3 round a
//     focused button, field or combo (5 and 7px corners), in a margin
//     each keeps round its face;
//   - scroll areas are frameless (6.0): lists, trees and tables sit on the
//     view colour with no outline, and the current row carries the focus;
//   - list, tree and table selections are 5px-rounded boxes in a 1px
//     outline, inset 2px from the row's sides and 1px from its top and
//     bottom (6.7); the hover is the highlight at 0.3;
//   - a pressed menu item is the solid focus colour (6.7), and the
//     highlight keeps 4px from the menu's sides;
//   - document tabs are 34px tall (6.3);
//   - Breeze Dark is darker (6.4): window #202326, view #141618, button
//     #292c30;
//   - the window decoration rounds its bottom corners too (6.5).
//
// The mixes live in the breeze colour set (breeze6Mixes), so the drawing
// this era keeps from Plasma 5 follows them.
type breeze6Engine struct{ breezeEngine }

func init() {
	for _, p := range breeze6Packs() {
		RegisterPack(p)
	}
}

// EngineFor paints packs with param "plasma" 6 with breeze6Engine.
func (breezeEngine) EngineFor(t ThemeTokens) Engine {
	if t.Params["plasma"] >= 6 {
		return breeze6Engine{}
	}
	return nil
}

// DefaultMetrics are Plasma 6's proportions at the toolkit's 16px UI font:
// the Plasma 5 sizes with 34px document tabs, rows a little taller for
// their rounded boxes, and buttons, fields and combos 2px larger all round
// for the focus band's margin.
func (breeze6Engine) DefaultMetrics() ChromeMetrics {
	m := breezeEngine{}.DefaultMetrics()
	m.Radius, m.RadiusSmall = 5, 5
	m.ControlH, m.FieldH, m.ComboH = 36, 34, 36
	m.TabH, m.RowH, m.MenuItemH = 34, 28, 30
	m.FieldPad, m.FocusWidth = 9, 2
	return m
}

// breeze6Mixes re-derives the colours Plasma 6 mixes differently: the 0.2
// frame contrast for every outline, the highlight at 0.3 for pressed
// buttons and checked boxes (over the button colour), at 0.7 for slider and
// progress values.
func breeze6Mixes(c *breeze) {
	const contrast = 0.2
	c.frameR = 4.5
	c.outline = Mix(c.win, c.text, contrast)
	c.sep = c.outline
	c.btnOutline = Mix(c.btn, c.btnText, contrast)
	c.btnDown = Mix(c.btn, c.hl, 0.3)
	c.btnDef = Mix(c.btn, c.hl, 0.2)
	c.btnDefLine = Mix(c.hl, c.btnOutline, 0.5)
	c.chkLine = c.outline
	c.chkOn = Mix(c.btn, c.hl, 0.3)
	c.chkOnDown = darkerPct(c.chkOn, 110)
	c.chkDown = darkerPct(c.btn, 110)
	c.hlGrooveFill = Mix(c.win, c.hl, 0.7)
	c.progFill = Mix(c.win, c.hl, 0.7)
}

// ---- shapes -------------------------------------------------------------------------------------

// breeze6Margin is the room buttons, fields and combos keep round their face
// for the focus band: 2px.
func breeze6Margin(l *Classic) float32 { return 2 * brU(l) }

// b6R is the 5px corner of Plasma 6's rows, tabs and window frames.
func b6R(l *Classic) float32 { return l.rx(5) }

// band paints Plasma 6's keyboard focus: a 2px band of the highlight at 0.3
// hugging the outside of face (the frame's 5px corner inside, 7px outside).
func (c *breeze) band(l *Classic, ctx *paintengine2d.Context, face paintengine2d.Rect) {
	w := breeze6Margin(l)
	r := float32(0)
	if !l.square() {
		r = brR(l) + brU(l)*0.5
	}
	webBand(ctx, face.Inset(-w), r+w, face, r, c.hl.WithAlpha(0.3))
}

// button6 is a Plasma 6 push button inside b: 2px in (the focus band's
// room), the button colour in its outline over a one-pixel shadow; hover and
// focus turn the outline to the highlight, a press fills it with the
// highlight at 0.3, the default button with a fifth; keyboard focus adds the
// band.
func (c *breeze) button6(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, def bool) {
	u := brU(l)
	sb := b.Inset(breeze6Margin(l))
	room := true
	if sb.Dx() < 3*u || sb.Dy() < 3*u {
		sb, room = b.Inset(u), false
		if sb.Dx() < 3*u || sb.Dy() < 3*u {
			return
		}
	}
	r := brR(l)
	enabled := !st.Disabled()
	down := enabled && st.Pressed()
	checked := st.Toggle() && st.Checked()
	bg, pen := c.btn, c.btnOutline
	switch {
	case down:
		bg = c.btnDown
	case checked:
		bg = c.btnChecked
	case def && enabled:
		bg, pen = c.btnDef, c.btnDefLine
	case !enabled:
		bg = c.btnDis
	}
	if enabled && (st.Hovered() || st.Focused() || down) {
		pen = c.hl
	}
	if enabled && !down && !checked {
		// The shadow: the outline ring half a pixel lower.
		ctx.DrawRoundRect(paintengine2d.XYWH(sb.Min.X+u*0.5, sb.Min.Y+u, sb.Dx()-u, sb.Dy()-u*0.5), r, r, paintengine2d.StrokePaint(c.shadow, u))
	}
	in := sb.Inset(u * 0.5)
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(bg))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(pen, u))
	if enabled && st.Focused() && room {
		c.band(l, ctx, sb)
	}
}

// field6 is a Plasma 6 text field inside b: the view colour in the frame
// outline 2px in; the hover colour's outline under the pointer, the focus
// colour's and the band while focused.
func (c *breeze) field6(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	u := brU(l)
	fill, line := c.base, c.outline
	switch {
	case st.Disabled():
		fill = c.disField
	case st.Focused():
		line = c.focus
	case st.Hovered():
		line = c.hover
	}
	f := b.Inset(u) // frame insets a pixel more for its outline
	if f.Dx() < 6*u || f.Dy() < 6*u {
		c.frame(l, ctx, b, fill, line)
		return
	}
	c.frame(l, ctx, f, fill, line)
	if st.Focused() && !st.Disabled() {
		c.band(l, ctx, b.Inset(breeze6Margin(l)))
	}
}

// breeze6Box fills box rounded (tl, tr, br, bl) and rings it with a u-wide
// outline just inside its edge.
func breeze6Box(ctx *paintengine2d.Context, box paintengine2d.Rect, rad [4]float32, u float32, fill, line paintengine2d.Color) {
	if box.Dx() < 2*u || box.Dy() < 2*u {
		return
	}
	lim := min(box.Dx(), box.Dy()) * 0.5
	for i := range rad {
		rad[i] = min(rad[i], lim)
	}
	if fill.A > 0 {
		ctx.DrawPath(RoundRectPath(box, rad[0], rad[1], rad[2], rad[3]), paintengine2d.Fill(fill))
	}
	if line.A > 0 {
		p := paintengine2d.NewPath()
		winAddRoundRect(p, box, rad[0], rad[1], rad[2], rad[3])
		winAddRoundRect(p, box.Inset(u), max(rad[0]-u, 0), max(rad[1]-u, 0), max(rad[2]-u, 0), max(rad[3]-u, 0))
		ctx.DrawPath(p, paintengine2d.Paint{Color: line, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
	}
}

// rowBox6 is where a Plasma 6 row's highlight paints inside the row b: 2px
// in from the sides, 1px from the top and bottom (the item margins).
func rowBox6(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	u := brU(l)
	box := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+2*u, b.Min.Y+u), Max: paintengine2d.Pt(b.Max.X-2*u, b.Max.Y-u)}
	if box.Dx() < 6*u || box.Dy() < 6*u {
		return b
	}
	return box
}

// row6 paints a Plasma 6 row's highlight into box and returns its label
// colour: the selection in an outline of its colour mixed a little towards
// the text (the scheme's inactive selection in an inactive window), lighter
// under the pointer; the highlight at 0.3 in a half-strength outline under
// the pointer. KDE keeps a selection when only the view loses focus.
func (c *breeze) row6(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState) paintengine2d.Color {
	u := brU(l)
	r := b6R(l)
	rad := [4]float32{r, r, r, r}
	hot := st.Hovered() && !st.Disabled()
	switch {
	case st.Checked():
		fill, fg := c.hl, c.hlText
		switch {
		case st.Disabled():
			fill = c.hlDis
		case st.Backdrop():
			fill, fg = c.hlOff, c.hlOffText
		case hot:
			fill = lighterPct(c.hl, 110)
		}
		breeze6Box(ctx, box, rad, u, fill, Mix(fill, c.viewText, 0.15))
		return fg
	case hot:
		breeze6Box(ctx, box, rad, u, c.hl.WithAlpha(0.3), c.hl.WithAlpha(0.5))
	}
	if st.Disabled() {
		return c.dis
	}
	return c.viewText
}

// ---- parts --------------------------------------------------------------------------------------

func (e breeze6Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := breezeColors(l)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	if b.Empty() {
		return fg
	}
	switch role {
	case RoleButton, RoleCombo:
		c.button6(l, ctx, b, st, st.Primary())
		return fg
	case RoleField:
		c.field6(l, ctx, b, st)
		if st.Disabled() {
			return c.dis
		}
		return c.viewText
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, st.Checked())
		return fg
	case RoleRow:
		return c.row6(l, ctx, rowBox6(l, fuSnap(b)), st)
	case RoleMenu:
		if st.Pressed() && !st.Disabled() {
			ctx.DrawRoundRect(b, brR(l), brR(l), paintengine2d.Fill(c.focus))
			return c.hlText
		}
	}
	return e.breezeEngine.Face(l, ctx, b, role, st)
}

// CheckIndicator is Plasma 6's check box: a 16px square with 4px corners in
// the button colour and the frame outline, or the highlight at 0.3 over the
// button colour in a highlight outline once checked, with a 2px tick in the
// text colour; hover rings it in the focus colour.
func (breeze6Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := breezeColors(l)
	u := brU(l)
	b := box
	if d := b.Dx() - b.Dy(); d > 0 {
		b = paintengine2d.XYWH(b.Min.X+d*0.5, b.Min.Y, b.Dy(), b.Dy())
	} else if d < 0 {
		b = paintengine2d.XYWH(b.Min.X, b.Min.Y-d*0.5, b.Dx(), b.Dx())
	}
	f := b.Inset(2 * u * b.Dx() / l.S(20))
	if f.Dx() < 6*u {
		f = b
	}
	if f.Dx() < 4*u {
		return
	}
	r := min(l.rx(4), f.Dx()*0.5)
	in := f.Inset(u * 0.5)
	enabled := !st.Disabled()
	down := enabled && st.Pressed()
	bg, pen := c.btn, c.chkLine
	switch {
	case !enabled:
		bg = c.btnDis
	case checked && down:
		bg, pen = c.chkOnDown, c.hl
	case checked:
		bg, pen = c.chkOn, c.hl
	case down:
		bg = c.chkDown
	}
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(bg))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(pen, u))
	if enabled && st.Hovered() {
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.focus, u))
	}
	if !checked {
		return
	}
	mark := c.btnText
	if !enabled {
		mark = c.dis
	}
	side := f.Dx()
	at := func(fx, fy float32) paintengine2d.Point { return paintengine2d.Pt(f.Min.X+fx*side, f.Min.Y+fy*side) }
	strokeTick(ctx, at(0.245, 0.495), at(0.44, 0.69), at(0.805, 0.32), 2*u*1.001, mark, paintengine2d.JoinMiter)
}

// RadioIndicator is the round twin with a text-coloured dot.
func (breeze6Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := breezeColors(l)
	u := brU(l)
	side := min(box.Dx(), box.Dy())
	ctr := box.Center()
	r := side*0.5 - 2*u*side/l.S(20)
	if r < 3*u {
		r = side * 0.5
	}
	if r < 2*u {
		return
	}
	enabled := !st.Disabled()
	down := enabled && st.Pressed()
	bg, pen := c.btn, c.chkLine
	switch {
	case !enabled:
		bg = c.btnDis
	case selected && down:
		bg, pen = c.chkOnDown, c.hl
	case selected:
		bg, pen = c.chkOn, c.hl
	case down:
		bg = c.chkDown
	}
	rr := r - u*0.5
	ctx.DrawCircle(ctr, rr, paintengine2d.Fill(bg))
	ctx.DrawCircle(ctr, rr, paintengine2d.StrokePaint(pen, u))
	if enabled && st.Hovered() {
		ctx.DrawCircle(ctr, rr, paintengine2d.StrokePaint(c.focus, u))
	}
	if selected {
		dot := c.btnText
		if !enabled {
			dot = c.dis
		}
		ctx.DrawCircle(ctr, r*0.375, paintengine2d.Fill(dot))
	}
}

// ItemFocus marks the current row: the focus colour's outline on its
// rounded box, or, on a selected row, a line of the selection's text colour
// inside the selection (frameless views have no frame to ring).
func (breeze6Engine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := breezeColors(l)
	u := brU(l)
	box := rowBox6(l, fuSnap(b))
	if st.Disabled() || box.Dx() < 8*u || box.Dy() < 8*u {
		return
	}
	r := b6R(l)
	if st.Checked() {
		in := box.Inset(2 * u)
		winRing(ctx, in, max(r-2*u, 0), u, paintengine2d.Fill(c.hlText.WithAlpha(0.6)))
		return
	}
	winRing(ctx, box, r, u, paintengine2d.Fill(c.focus))
}

// ViewFrameInsets: Plasma 6's scroll areas are frameless.
func (breeze6Engine) ViewFrameInsets(l *Classic) Insets { return Insets{} }

// DrawViewFrame is a frameless view: the view colour edge to edge.
func (breeze6Engine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := breezeColors(l)
	fill := c.base
	if st.Disabled() {
		fill = c.disField
	}
	if !b.Empty() {
		ctx.DrawRect(b, paintengine2d.Fill(fill))
	}
}

// ---- controls -----------------------------------------------------------------------------------

func (e breeze6Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := breezeColors(l)
	b = fuSnap(b)
	c.button6(l, ctx, b, st, st.Primary())
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, l.S(14))
}

func (e breeze6Engine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := min(l.metrics.Checkbox, b.Dy())
	box := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e breeze6Engine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	side = min(side, b.Dy())
	box := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawComboBox is a non-editable combo: the Plasma 6 push button, the text
// and the chevron in a 20px column at the right.
func (e breeze6Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := breezeColors(l)
	b = fuSnap(b)
	bst := st
	if open {
		bst |= StatePressed
	}
	c.button6(l, ctx, b, bst, false)
	aw := l.S(20)
	ab := paintengine2d.XYWH(b.Max.X-aw-l.S(7), b.Min.Y, aw, b.Dy())
	col := c.arrowBtn
	switch {
	case st.Disabled():
		col = c.dis
	case st.Focused() && !st.Hovered():
		col = c.btnText
	}
	brArrow(l, ctx, ab, DirDown, col)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(11), b.Min.Y, ab.Min.X-b.Min.X-l.S(13), b.Dy())
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
}

// DrawSpinner is the spin box's arrow column. Inside a field it drew
// (StateFrameless) it is the chevrons alone; standing on its own it
// continues a field's frame, 2px in, round its right end.
func (e breeze6Engine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	m := breeze6Margin(l)
	f := paintengine2d.XYWH(b.Min.X, b.Min.Y+m, b.Dx()-m, b.Dy()-2*m)
	if f.Dx() < 4*u || f.Dy() < 6*u {
		return
	}
	if !st.Frameless() {
		fill, line := c.base, c.outline
		switch {
		case st.Disabled():
			fill = c.disField
		case st.Focused():
			line = c.focus
		case st.Hovered() || upHover || downHover:
			line = c.hover
		}
		r := brR(l)
		in := f.Inset(u * 0.5)
		path := RoundRectPath(paintengine2d.XYWH(in.Min.X-r, in.Min.Y, in.Dx()+r, in.Dy()), 0, r, r, 0)
		ctx.Save()
		ctx.ClipRect(f)
		ctx.DrawPath(path, paintengine2d.Fill(fill))
		ctx.DrawPath(path, paintengine2d.StrokePaint(line, u))
		ctx.Restore()
	}
	mid := snap((f.Min.Y + f.Max.Y) * 0.5)
	up := paintengine2d.XYWH(f.Min.X, f.Min.Y, f.Dx(), mid-f.Min.Y)
	dn := paintengine2d.XYWH(f.Min.X, mid, f.Dx(), f.Max.Y-mid)
	arrow := func(r paintengine2d.Rect, dir Direction, hot, press bool) {
		col := c.arrowText
		switch {
		case st.Disabled():
			col = c.dis
		case hot || press:
			col = c.hover
		}
		brArrow(l, ctx, r, dir, col)
	}
	arrow(up.Translate(paintengine2d.Pt(0, l.S(1))), DirUp, upHover, upPress)
	arrow(dn.Translate(paintengine2d.Pt(0, -l.S(1))), DirDown, downHover, downPress)
}

// DrawTab: the selected tab is the window colour in the frame outline with
// 5px top corners and a 3px highlight strip along its top, opening into the
// pane; the others are the window darker 120%, rounded only at the strip's
// ends and washed with the hover colour at 20% under the pointer.
func (e breeze6Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	r := b6R(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if selected {
		rr := brR(l)
		body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()+2*u)
		ctx.Save()
		ctx.ClipRect(b)
		in := body.Inset(u * 0.5)
		ctx.DrawRoundRectCorners(in, rr, rr, 0, 0, paintengine2d.Fill(c.win))
		ctx.DrawRoundRectCorners(in, rr, rr, 0, 0, paintengine2d.StrokePaint(c.outline, u))
		strip := snap(l.S(3))
		ctx.DrawRoundRectCorners(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), strip), r, r, 0, 0, paintengine2d.Fill(c.hl))
		ctx.Restore()
	} else {
		bg := c.tabOff
		if st.Hovered() && !st.Disabled() {
			bg = c.tabHover
		}
		tl, tr := float32(0), float32(0)
		if st.First() {
			tl = r
		}
		if st.Last() {
			tr = r
		}
		ctx.DrawRoundRectCorners(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u), tl, tr, 0, 0, paintengine2d.Fill(bg))
	}
	lb := b
	if selected {
		lb.Min.Y += snap(l.S(2))
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		c.focusLine(l, ctx, l.body, label, lb.Inset(l.S(4)), b, AlignCenter)
	}
}

// DrawMenuItem is a menu row: the rounded focus highlight 4px from the
// menu's sides (the solid focus colour and the highlighted text while
// pressed), Plasma 6 check boxes and radios in the check column, the
// shortcut at 70% and a chevron for submenus.
func (e breeze6Engine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := breezeColors(l)
	u := brU(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		fuHLine(ctx, b.Min.X-ch.PadL+l.S(5), b.Max.X+ch.PadR-l.S(5), y, u, c.sep)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	fg := c.text
	if hot {
		k := min(snap(l.S(4)), ch.PadL)
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y+u, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy()-2*u)
		if st.Pressed() {
			ctx.DrawRoundRect(hb, brR(l), brR(l), paintengine2d.Fill(c.focus))
			fg = c.hlText
		} else {
			e.MenuHighlight(l, ctx, hb, false)
		}
	}
	if st.Disabled() {
		fg = c.dis
	}
	gw := ch.CheckCol()
	side := snap(min(l.S(20), b.Dy()))
	ib := fuSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	ist := StateNone
	if st.Disabled() {
		ist = StateDisabled
	}
	switch {
	case row.Radio:
		e.RadioIndicator(l, ctx, ib, ist, row.Checked)
	case row.Checked:
		e.CheckIndicator(l, ctx, ib, ist, true)
	case row.Icon != IconNone:
		l.drawToolIcon(ctx, ib.Inset(l.S(1)), row.Icon, fg)
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow, b.Dy())
		col := c.arrow
		switch {
		case st.Disabled():
			col = c.dis
		case hot && st.Pressed():
			col = fg
		}
		brArrow(l, ctx, ab, DirRight, col)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := l.body.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		l.body.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg.WithAlpha(fg.A*0.7))
		right = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if right < lx {
		right = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	l.drawTextUnderline(ctx, l.body, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

func (e breeze6Engine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := breezeColors(l)
	b = fuSnap(b)
	fg := c.row6(l, ctx, rowBox6(l, b), st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(10), b.Min.Y, b.Dx()-l.S(14), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow: the rounded row selection, branch lines in text at 20% over
// the view and the chevron expander.
func (e breeze6Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	fg := c.row6(l, ctx, rowBox6(l, b), st)
	selected, hovered := st.Checked(), st.Hovered() && !st.Disabled()
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(6)
	aw := l.S(14)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	line := Mix(c.base, c.viewText, 0.2)
	if selected {
		line = Mix(c.hl, c.hlText, 0.35)
	}
	for d := 0; d < depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + aw*0.5)
		switch {
		case d == depth-1 && !st.HasNextSibling(depth):
			fuVLine(ctx, gx, b.Min.Y, cy, u, line)
		case d == depth-1 || st.HasNextSibling(d+1):
			fuVLine(ctx, gx, b.Min.Y, b.Max.Y, u, line)
		}
	}
	if depth > 0 {
		gx := snap(b.Min.X + pad + float32(depth-1)*indent + aw*0.5)
		fuHLine(ctx, gx, x+aw*0.2, cy, u, line)
	}
	if !leaf {
		col := c.arrowText
		switch {
		case selected:
			col = fg
		case hovered:
			col = c.hover
		}
		dir := DirRight
		if expanded {
			dir = DirDown
		}
		brArrow(l, ctx, paintengine2d.XYWH(x, b.Min.Y, aw, b.Dy()), dir, col)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	ctx.Save()
	ctx.ClipRect(rowBox6(l, b))
	f.Draw(ctx, label, paintengine2d.Pt(x+aw+l.S(4), b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableCell: a table row's cells share one rounded selection box, no
// grid lines.
func (e breeze6Engine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := breezeColors(l)
	cell := fuSnap(b)
	ctx.Save()
	ctx.ClipRect(cell)
	fg := c.row6(l, ctx, rowBox6(l, CellSpan(cell, st, l.S(24))), st)
	ctx.Restore()
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	if st.First() {
		pad = max(pad, l.S(8))
	}
	label = f.Fit(label, max(b.Dx()-pad*2, 4))
	tw := f.Advance(label)
	x := b.Min.X + pad
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - pad
	}
	ctx.Save()
	ctx.ClipRect(cell)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawWindowFrame is KWin's Breeze decoration since Plasma 6.5: 5px corners
// all round, a flat title bar in the Header colours with the title centred
// and the close circle (red under the pointer, darker held), a separator
// above the window, the outline mixed from the text at 0.2.
func (e breeze6Engine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	h := brTitleH(l)
	if b.Dx() < 6*u || b.Dy() < h {
		return
	}
	r := b6R(l)
	bar, fg := c.title, c.titleText
	if !st.Active {
		bar, fg = c.titleOff, c.titleTextOff
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.win))
	ctx.DrawRoundRectCorners(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), h), r, r, 0, 0, paintengine2d.Fill(bar))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y+h-u, u, c.sep)
	ctx.DrawRoundRect(b.Inset(u*0.5), max(r-u*0.5, 0), max(r-u*0.5, 0), paintengine2d.StrokePaint(Mix(c.win, c.text, 0.2), u))
	right := b.Max.X - l.S(8)
	if st.CanClose {
		if cb := e.WindowCloseRect(l, b); !cb.Empty() {
			glyph := fg
			switch {
			case st.ClosePress:
				ctx.DrawCircle(cb.Center(), cb.Dx()*0.5, paintengine2d.Fill(darkerPct(c.negative, 200)))
				glyph = bar
			case st.CloseHot && st.Active:
				ctx.DrawCircle(cb.Center(), cb.Dx()*0.5, paintengine2d.Fill(lighterPct(c.negative, 150)))
				glyph = bar
			case st.CloseHot:
				ctx.DrawCircle(cb.Center(), cb.Dx()*0.5, paintengine2d.Fill(c.negative))
				glyph = bar
			}
			k := cb.Dx() / 18
			DrawCross(ctx, paintengine2d.XYWH(cb.Min.X+5*k, cb.Min.Y+5*k, 8*k, 8*k), glyph, u*1.001)
			right = cb.Min.X - l.S(6)
		}
	}
	if title != "" {
		left := b.Min.X + (b.Max.X - right)
		l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(left, b.Min.Y, right-left, h-u), fg, AlignCenter, 0)
	}
}

// ---- packs --------------------------------------------------------------------------------------

// breeze6Light is Breeze Light as Plasma 6 ships it: the 5.27 colour scheme,
// unchanged since, with the title bar in the Header colours as KWin's Breeze
// decoration draws it.
var breeze6Light = func() breezeScheme {
	s := breezeLightScheme
	s.wmActive, s.wmInactive = "#dee0e2", "#eff0f1"
	return s
}()

// breeze6Dark is Breeze Dark as Plasma 6.4 darkened it.
var breeze6Dark = breezeScheme{
	window: "#202326", windowText: "#fcfcfc", view: "#141618", viewText: "#fcfcfc",
	button: "#292c30", buttonText: "#fcfcfc", selection: "#3daee9", selectionText: "#fcfcfc",
	tooltip: "#292c30", tooltipText: "#fcfcfc", header: "#292c30", headerText: "#fcfcfc",
	inactiveText: "#a1a9b1", negative: "#da4453", neutral: "#f67400", positive: "#27ae60",
	link: "#1d99f3", focus: "#3daee9", hover: "#3daee9",
	wmActive: "#292c30", wmActiveText: "#fcfcfc", wmInactive: "#202326", wmInactiveText: "#a1a9b1",
	selectionAlt: "#1e5774",
}

// breeze6Pack is a Plasma 6 pack: the breeze pack of scheme s with Plasma
// 6's 0.2 outlines and 0.3 pressed mix in its palette.
func breeze6Pack(name, label, summary string, fam ThemeName, s breezeScheme) ThemePack {
	p := breezePack(name, label, summary, fam, s)
	p.Year = 2024
	tok := &p.Tokens
	win, text := hexColor(s.window), hexColor(s.windowText)
	outline := Mix(win, text, 0.2)
	pal := &tok.Palette
	pal.Border, pal.Divider, pal.FieldBorder, pal.BevelDark = outline, outline, outline, outline
	pal.AccentPress = Mix(hexColor(s.button), hexColor(s.selection), 0.3)
	pal.Highlight = hexColor(s.selection).WithAlpha(0.3)
	tok.Params = map[string]float32{"plasma": 6}
	breezeChrome(tok)
	return p
}

func breeze6Packs() []ThemePack {
	return []ThemePack{
		breeze6Pack("breeze6", "Breeze (Plasma 6)",
			"KDE Plasma 6's Breeze: 5px frames in a 20% outline, frameless views, rounded outlined selections, a soft focus band around the focused control.",
			ThemeLight, breeze6Light),
		breeze6Pack("breeze6-night", "Breeze Dark (Plasma 6)",
			"Plasma 6.4's darker Breeze Dark: #202326 windows, #141618 views and #292c30 buttons in the Plasma 6 shapes.",
			ThemeDark, breeze6Dark),
	}
}
