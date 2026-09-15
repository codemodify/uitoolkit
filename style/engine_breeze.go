package style

import "github.com/codemodify/paintengine2d"

// breezeEngine paints in the manner of KDE Breeze, the Plasma 5 widget
// style (2014–2023, as of Plasma 5.27). It is written from Breeze's visible
// behaviour, its published colour schemes and its colour mixes (as
// numbers), checked against screenshots. Everything is flat and mixed from
// the colour scheme:
//
//	frame outline    mix(window, windowText, 0.25)
//	button outline   mix(button, buttonText, 0.3)
//	focus / hover    the scheme's focus / hover decoration (#3daee9)
//	frame background mix(window, base, 0.3) — group boxes, tab panes, menus
//
// Frames are 1px outlines with 3px corners (Plasma 6 moved to 5px; these
// are the Plasma 5 proportions). Push buttons are
// the button colour over a one-pixel drop shadow; hover and keyboard focus
// turn the outline to the highlight, a pressed button fills with a third of
// the highlight, the default button with a fifth. Check boxes and radios
// are 16px rounded squares / circles in text at 33%; when checked they fill
// with the highlight at 33% in a highlight outline and carry a text-coloured
// tick or dot, hover rings them in the focus colour and keyboard focus
// underlines their label. Sliders and progress bars are 6px rounded
// grooves filled with the highlight; the slider handle is a 20px circle in
// the button colour with an outline and a shadow. Scroll bars carry no
// arrows: a rounded handle in text at 50% that widens and shows its groove
// when the pointer is over the bar (Breeze's hover behaviour, as an
// overlay bar here). The selected tab is the window colour with a 3px
// highlight strip along its top edge; the others are the window darker
// 120%.
// Menus are rounded frame-background panels whose selected row is the
// focus colour at 30% in a focus outline; tool tips are rounded frames.
// Menu bars, tool bars and title strips share the scheme's "Header"
// colour (the tools area, from Plasma 5.21).
//
// Pack data (theme.json "extra"; defaults are Breeze Light):
//
//	button, buttonText, view, viewText, tip, tipText    colour-scheme sets
//	header, headerText                                  tools area
//	titleBar, titleText, titleBarOff, titleTextOff      KWin decoration (WM)
//	focus, hover, negative, disabledText
type breezeEngine struct{ BaseEngine }

func init() {
	RegisterEngine(breezeEngine{})
	for _, p := range breezePacks() {
		RegisterPack(p)
	}
}

func (breezeEngine) ID() string { return "breeze" }

// DefaultMetrics are Breeze's (Plasma 5) proportions at the toolkit's 16px
// UI font: 20px check boxes (16px drawn), 20px slider handles, 6px grooves,
// 3px corners.
func (breezeEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Radius: 3, RadiusSmall: 3,
		ControlH: 34, FieldH: 32, ComboH: 32,
		Checkbox: 20, Radio: 20,
		MenuItemH: 30, MenuBarH: 30, TabH: 32, RowH: 26,
		TitleBar: 30, HeaderH: 28, ProgressH: 16, SliderH: 24, Thumb: 20,
		Scroll: 12, Pad: 12, FieldPad: 8, FocusWidth: 1, Border: 1,
		ToolBarH: 40, StatusBarH: 26, SpinnerW: 22, SwitchW: 36, SwitchH: 20,
	}
}

// StyleHint: KDE's dialog order (the accepting button before the rejecting
// one: "OK  Cancel"), form labels right-aligned against their fields (as
// Breeze lays out Qt forms) and left-aligned tabs (Breeze's default).
func (breezeEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintHoverFadeMs {
		return 150 // Plasma's animated hover
	}
	switch h {
	case HintDialogPrimaryFirst, HintFormLabelsRight:
		return 1
	case HintMnemonics:
		return MnemonicsOnAlt // Plasma shows them while Alt is held
	}
	return 0
}

// ---- resolved colours ------------------------------------------------------------------

type breeze struct {
	win, text, base, viewText, btn, btnText, hl, hlText paintengine2d.Color
	tip, tipText, focus, hover, negative, dis           paintengine2d.Color
	header, headerText, title, titleText                paintengine2d.Color
	titleOff, titleTextOff                              paintengine2d.Color

	outline, btnOutline, frameBg, sep, focusOutline paintengine2d.Color
	arrow, arrowText, arrowBtn, shadow              paintengine2d.Color

	btnDown, btnChecked, btnDef, btnDefLine, btnDis paintengine2d.Color
	flatDown, flatChecked                           paintengine2d.Color

	chkLine, chkOn, chkOnDown, chkDown paintengine2d.Color

	groove, grooveFill, hlGrooveFill, sliderLine   paintengine2d.Color
	progFill, busyAlt                              paintengine2d.Color
	sbHandle, sbHandleIdle, sbGroove, sbGrooveFill paintengine2d.Color

	tabOff, tabHover, menuHot, tipLine, branch   paintengine2d.Color
	hlOff, hlOffText, hlDis                      paintengine2d.Color
	headerHot, headerDown, headerLine, headerSep paintengine2d.Color
	disField, disBtn                             paintengine2d.Color
}

type breezeKey struct{}

func breezeColors(l *Classic) *breeze {
	return l.Memo(breezeKey{}, func() any { return breezeBuild(l) }).(*breeze)
}

func breezeBuild(l *Classic) *breeze {
	p := l.palette
	c := &breeze{
		win:  p.Background,
		text: p.Text,
		base: p.Field,
		hl:   p.Selection,
	}
	if c.hl.A < 0.9 {
		c.hl = p.Accent
	}
	// (Not l.fieldText(): Memo is not re-entrant.)
	c.viewText = l.X("viewText", ReadableOn(c.base, 4.5, c.text))
	c.btn = l.X("button", p.SurfaceAlt)
	c.btnText = l.X("buttonText", c.text)
	c.hlText = p.TextOnAccent
	c.tip = l.X("tip", c.btn)
	c.tipText = l.X("tipText", c.btnText)
	c.focus = l.X("focus", c.hl)
	c.hover = l.X("hover", c.focus)
	c.negative = l.X("negative", p.Danger)
	// Disabled text: faded most of the way into a slightly darkened
	// window colour.
	c.dis = l.X("disabledText", Mix(c.text, Shade(c.win, -0.04), 0.62))
	c.header = l.X("header", c.win)
	c.headerText = l.X("headerText", c.text)
	c.title = l.X("titleBar", c.header)
	c.titleText = l.X("titleText", c.headerText)
	c.titleOff = l.X("titleBarOff", c.win)
	c.titleTextOff = l.X("titleTextOff", p.TextMuted)

	// Outlines, frames and marks.
	c.outline = Mix(c.win, c.text, 0.25)
	c.btnOutline = Mix(c.btn, c.btnText, 0.3)
	c.frameBg = Mix(c.win, c.base, 0.3)
	c.sep = c.outline
	c.focusOutline = Mix(c.focus, c.text, 0.15)
	c.arrow = Mix(c.text, c.win, 0.15)
	c.arrowText = Mix(c.viewText, c.base, 0.15)
	c.arrowBtn = Mix(c.btnText, c.btn, 0.15)
	c.shadow = paintengine2d.RGBA(0, 0, 0, 0.125)

	c.btnDown = Mix(c.btn, c.hl, 0.333)
	c.btnChecked = Mix(c.btn, c.btnText, 0.125)
	c.btnDef = Mix(c.btn, c.hl, 0.2)
	c.btnDefLine = Mix(c.hl, Mix(c.btn, c.btnText, 0.333), 0.5)
	c.btnDis = Shade(c.btn, -0.04)
	c.flatDown = c.hl.WithAlpha(0.33)
	c.flatChecked = c.btnText.WithAlpha(0.125)

	c.chkLine = c.viewText.WithAlpha(0.33)
	c.chkOn = c.hl.WithAlpha(0.33)
	c.chkOnDown = darkerPct(Mix(c.base, c.hl, 0.33), 110)
	c.chkDown = darkerPct(c.base, 110)

	c.groove = c.text.WithAlpha(0.2)
	c.grooveFill = c.text.WithAlpha(0.1)
	c.hlGrooveFill = c.hl.WithAlpha(0.5)
	c.sliderLine = Mix(c.win, c.text, 0.4)
	// Progress: the highlight at half strength over the window.
	c.progFill = Mix(c.win, c.hl, 0.5)
	c.busyAlt = Mix(c.hl, c.win, 0.7)
	c.sbHandle = c.text.WithAlpha(0.5)
	c.sbHandleIdle = c.text.WithAlpha(0.35)
	c.sbGroove = c.text.WithAlpha(0.2)
	c.sbGrooveFill = c.text.WithAlpha(0.1)

	c.tabOff = darkerPct(c.win, 120)
	c.tabHover = Mix(c.win, c.hover, 0.2)
	c.menuHot = c.focus.WithAlpha(0.3)
	c.tipLine = Mix(c.tip, c.tipText, 0.25)
	c.branch = Mix(c.base, c.viewText, 0.25)
	c.headerHot = Mix(c.btn, c.hover, 0.2)
	c.headerDown = Mix(c.btn, c.focus, 0.2)
	c.headerLine = c.text.WithAlpha(0.1)
	c.headerSep = c.text.WithAlpha(0.2)
	c.disField = Shade(c.base, -0.04)
	c.hlOff = l.X("selectionInactive", Mix(c.hl, c.base, 0.5))
	c.hlOffText = ReadableOn(c.hlOff, 4.5, c.hlText, c.viewText)
	c.hlDis = Mix(c.hl, c.win, 0.5)
	return c
}

// ---- painting helpers ------------------------------------------------------------------

// brU is one design pixel at the look's scale (never under a device pixel).
func brU(l *Classic) float32 {
	u := snap(l.S(1))
	if u < 1 {
		u = 1
	}
	return u
}

// brR is the 3px frame radius for a 1px pen stroked half a pixel in.
func brR(l *Classic) float32 { return l.rx(2.5) }

// frame is Breeze's frame: a 1px margin, then the outline (stroked half a
// pixel in) around the fill.
func (c *breeze) frame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill, line paintengine2d.Color) {
	u := brU(l)
	f := b.Inset(u)
	if f.Dx() < 2*u || f.Dy() < 2*u {
		return
	}
	r := brR(l)
	in := f.Inset(u * 0.5)
	if fill.A > 0 {
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(fill))
	}
	if line.A > 0 {
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(line, u))
	}
}

// button is a raised push button: the button colour in its outline over a
// one-pixel drop shadow; hover / focus / press draw the outline in the
// highlight.
func (c *breeze) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, def bool) {
	u := brU(l)
	sb := b.Inset(u) // the button, a pixel in for its shadow
	if sb.Dx() < 3*u || sb.Dy() < 3*u {
		return
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
}

// flat is a flat (tool) button: nothing at rest, the highlight outline
// when hot or focused, a highlight wash when held.
func (c *breeze) flat(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) bool {
	u := brU(l)
	sb := b.Inset(u)
	if sb.Dx() < 3*u || sb.Dy() < 3*u {
		return false
	}
	enabled := !st.Disabled()
	down := enabled && st.Pressed()
	checked := st.Toggle() && st.Checked()
	var bg, pen paintengine2d.Color
	switch {
	case down:
		bg = c.flatDown
	case checked:
		bg, pen = c.flatChecked, c.btnOutline
	}
	if enabled && (st.Hovered() || st.Focused() || down) {
		pen = c.hl
	}
	if bg.A == 0 && pen.A == 0 {
		return false
	}
	r := brR(l)
	in := sb.Inset(u * 0.5)
	if bg.A > 0 {
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(bg))
	}
	if pen.A > 0 {
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(pen, u))
	}
	return true
}

// brArrow strokes Breeze's 1px chevron centred in b: 9px across and 4px
// deep in a 10px box (measured from Breeze screenshots), its arms on pixel
// centres.
func brArrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	span := min(l.S(10), b.Dx(), b.Dy())
	if span < 3 || col.A <= 0 {
		return
	}
	u := brU(l)
	half, depth := span*0.45, snap(span*0.4)
	// The chevron's width runs along one axis, centred on a pixel edge;
	// its depth along the other, centred on a pixel centre.
	ctr := b.Center()
	vertical := dir == DirUp || dir == DirDown
	cx, cy := snap(ctr.X), snap(ctr.Y)+u*0.5
	if !vertical {
		cx, cy = snap(ctr.X)+u*0.5, snap(ctr.Y)
	}
	sign := float32(1)
	if dir == DirUp || dir == DirLeft {
		sign = -1
	}
	// pt maps (along the width, along the depth) to the page.
	pt := func(w, d float32) (float32, float32) {
		if vertical {
			return cx + w, cy + sign*d
		}
		return cx + sign*d, cy + w
	}
	p := paintengine2d.NewPath()
	p.MoveTo(pt(-half, -depth*0.5))
	p.LineTo(pt(0, depth*0.5))
	p.LineTo(pt(half, -depth*0.5))
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// focusLine is Breeze's label focus: a 1px focus-coloured underline two
// pixels below the text.
func (c *breeze) focusLine(l *Classic, ctx *paintengine2d.Context, f *Font, label string, lb, clip paintengine2d.Rect, align Align) {
	u := brU(l)
	tw := f.Advance(label)
	if tw > lb.Dx() {
		tw = lb.Dx()
	}
	x := lb.Min.X
	if align == AlignCenter {
		x = lb.Min.X + (lb.Dx()-tw)*0.5
	}
	y := snap(lb.Min.Y + (lb.Dy()+f.Height())*0.5 + l.S(1))
	if y+u > clip.Max.Y {
		y = clip.Max.Y - u
	}
	ctx.DrawRect(paintengine2d.XYWH(x, y, tw, u), paintengine2d.Fill(c.focus))
}

// ---- parts ---------------------------------------------------------------------------------

func (e breezeEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
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
		c.button(l, ctx, b, st, st.Primary())
	case RoleTool:
		c.flat(l, ctx, b, st)
	case RoleField:
		fill, line := c.base, c.outline
		switch {
		case st.Disabled():
			fill = c.disField
		case st.Focused():
			line = c.focus
		case st.Hovered():
			line = c.hover
		}
		c.frame(l, ctx, b, fill, line)
		fg = c.viewText
		if st.Disabled() {
			fg = c.dis
		}
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, st.Checked())
	case RoleRow:
		return c.row(ctx, b, st)
	case RoleMenu:
		if st.Hovered() || st.Pressed() || st.Checked() {
			e.MenuHighlight(l, ctx, b, false)
		}
		return c.text
	case RoleThumb:
		c.handle(l, ctx, b, b.Dy() >= b.Dx(), st.Hovered(), st.Hovered() || st.Pressed(), st.Pressed())
	case RoleTrack:
		c.sbTrack(l, ctx, b, b.Dy() >= b.Dx())
	case RoleBar:
		ctx.DrawRect(b, paintengine2d.Fill(c.header))
		return c.headerText
	case RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.win))
		return c.text
	}
	return fg
}

// CheckIndicator is Breeze's check box (Plasma 5.21 and later): a 16px
// square with 2px corners — the view colour in text at 33%, or the
// highlight at 33% in a highlight outline when checked — with a 2px
// text-coloured tick; hover rings it in the focus colour.
func (breezeEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
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
	r := l.rx(2)
	in := f.Inset(u * 0.5)
	enabled := !st.Disabled()
	down := enabled && st.Pressed()
	bg, pen := c.base, c.chkLine
	switch {
	case !enabled && checked:
		bg, pen = c.disField, c.chkLine
	case !enabled:
		bg = c.disField
	case checked && down:
		bg, pen = c.chkOnDown, c.hl
	case checked:
		bg, pen = c.base, c.hl
	case down:
		bg = c.chkDown
	}
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(bg))
	if checked && enabled && !down {
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.chkOn))
	}
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(pen, u))
	if enabled && st.Hovered() {
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.focus, u))
	}
	if !checked {
		return
	}
	mark := c.viewText
	if !enabled {
		mark = c.dis
	}
	// The tick, measured from Breeze screenshots of the 16px frame: a short
	// arm from (24.5%, 49.5%) down to (44%, 69%), the long arm up to
	// (80.5%, 32%).
	side := f.Dx()
	at := func(fx, fy float32) paintengine2d.Point { return paintengine2d.Pt(f.Min.X+fx*side, f.Min.Y+fy*side) }
	strokeTick(ctx, at(0.245, 0.495), at(0.44, 0.69), at(0.805, 0.32), 2*u*1.001, mark, paintengine2d.JoinMiter)
}

// RadioIndicator is the circular version of the check box with a 6px
// text-coloured dot.
func (breezeEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := breezeColors(l)
	u := brU(l)
	side := box.Dx()
	if box.Dy() < side {
		side = box.Dy()
	}
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
	bg, pen := c.base, c.chkLine
	switch {
	case !enabled:
		bg = c.disField
	case selected && down:
		bg, pen = c.chkOnDown, c.hl
	case selected:
		pen = c.hl
	case down:
		bg = c.chkDown
	}
	rr := r - u*0.5
	ctx.DrawCircle(ctr, rr, paintengine2d.Fill(bg))
	if selected && enabled && !down {
		ctx.DrawCircle(ctr, rr, paintengine2d.Fill(c.chkOn))
	}
	ctx.DrawCircle(ctr, rr, paintengine2d.StrokePaint(pen, u))
	if enabled && st.Hovered() {
		ctx.DrawCircle(ctr, rr, paintengine2d.StrokePaint(c.focus, u))
	}
	if selected {
		dot := c.viewText
		if !enabled {
			dot = c.dis
		}
		// The dot is 3/8 of the circle's radius (measured).
		ctx.DrawCircle(ctr, r*0.375, paintengine2d.Fill(dot))
	}
}

// Arrow is Breeze's 1px chevron.
func (breezeEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	brArrow(l, ctx, b, dir, col)
}

// Expander is the tree branch chevron.
func (breezeEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	brArrow(l, ctx, b, dir, col)
}

// MenuHighlight is the selected menu item: the focus colour at 30% in the
// focus outline, rounded. An open menu-bar title is
// the solid focus colour (strong focus).
func (breezeEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := breezeColors(l)
	u := brU(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	r := brR(l)
	in := b.Inset(u * 0.5)
	if attachBottom {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.focus))
		return
	}
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.menuHot))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.focusOutline, u))
}

// Menu labels keep the window text on the translucent highlight; an open
// menu-bar title uses the highlighted text.
func (breezeEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return breezeColors(l).text
}

// Fields paint their own focus (the focus-coloured outline) in Face.
func (breezeEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the focus-coloured frame outline Breeze uses for
// keyboard focus on flat items.
func (breezeEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := breezeColors(l)
	u := brU(l)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := brR(l)
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.StrokePaint(c.focus, u))
}

// ItemFocus marks the current item with the focus colour's rounded
// outline; a selected item carries no extra mark (as in Breeze).
func (breezeEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := breezeColors(l)
	u := brU(l)
	if st.Checked() || st.Disabled() || b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := brR(l)
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.StrokePaint(c.focus, u))
}

// ViewFrameInsets: views sit in the frame's 1px margin and outline.
func (breezeEngine) ViewFrameInsets(l *Classic) Insets {
	u := brU(l)
	return Insets{Top: 2 * u, Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// DrawViewFrame frames item views: the view colour in the frame outline, turning the hover colour under the pointer and the focus
// colour when the view has keyboard focus.
func (breezeEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := breezeColors(l)
	fill, line := c.base, c.outline
	switch {
	case st.Disabled():
		fill = c.disField
	case st.Focused():
		line = c.focus
	case st.Hovered():
		line = c.hover
	}
	c.frame(l, ctx, b, fill, line)
}

// ---- scroll bars ---------------------------------------------------------------------------------

// ScrollBarStyle: no arrow buttons (Plasma 5's default); a thin overlay bar
// whose 8px handle and groove appear at full size when the pointer is over
// it.
func (breezeEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 12, Overlay: true, Inset: 2, Arrows: ArrowsNone, MinThumb: 20, EndPad: 3}
}

// sbTrack is the scroll bar groove: an 8px rounded groove in text at 20%.
func (c *breeze) sbTrack(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	u := brU(l)
	w := l.S(8)
	var g paintengine2d.Rect
	if vertical {
		if w > b.Dx() {
			w = b.Dx()
		}
		g = paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-w)*0.5), b.Min.Y, snap(w), b.Dy())
	} else {
		if w > b.Dy() {
			w = b.Dy()
		}
		g = paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-w)*0.5), b.Dx(), snap(w))
	}
	if g.Dx() < 2*u || g.Dy() < 2*u {
		return
	}
	r := g.Dx() * 0.5
	if g.Dy() < g.Dx() {
		r = g.Dy() * 0.5
	}
	in := g.Inset(u * 0.5)
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.sbGrooveFill))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.sbGroove, u))
}

// handle is the scroll bar handle: a rounded bar outlined in its colour
// and filled with the colour at 50% over the window; thin at rest, 8px when
// the bar is hovered, the hover colour when the handle itself is hot.
func (c *breeze) handle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical, hot, wide, down bool) {
	u := brU(l)
	w := l.S(6)
	if wide {
		w = l.S(8)
	}
	var h paintengine2d.Rect
	if vertical {
		if w > b.Dx() {
			w = b.Dx()
		}
		h = paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-w)*0.5), b.Min.Y, snap(w), b.Dy())
	} else {
		if w > b.Dy() {
			w = b.Dy()
		}
		h = paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-w)*0.5), b.Dx(), snap(w))
	}
	if h.Dx() < 2*u || h.Dy() < 2*u {
		return
	}
	col := c.sbHandleIdle
	switch {
	case down || hot:
		col = c.hover
	case wide:
		col = c.sbHandle
	}
	r := h.Dx() * 0.5
	if h.Dy() < h.Dx() {
		r = h.Dy() * 0.5
	}
	in := h.Inset(u * 0.5)
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(Mix(c.win, col, col.A*0.5)))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(col, u))
}

func (e breezeEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := breezeColors(l)
	if p.Bar.Empty() {
		return
	}
	hover := st.Hovered || st.Hot != ScrollNone || st.Pressed != ScrollNone
	if hover && !st.Disabled && !p.Track.Empty() {
		c.sbTrack(l, ctx, e.across(p.Bar, p.Track, vertical), vertical)
	}
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	c.handle(l, ctx, e.across(p.Bar, p.Thumb, vertical), vertical,
		st.Hot == ScrollThumbPart, hover, st.Pressed == ScrollThumbPart)
}

// across stretches r over the bar's full thickness.
func (breezeEngine) across(bar, r paintengine2d.Rect, vertical bool) paintengine2d.Rect {
	if vertical {
		return paintengine2d.XYWH(bar.Min.X, r.Min.Y, bar.Dx(), r.Dy())
	}
	return paintengine2d.XYWH(r.Min.X, bar.Min.Y, r.Dx(), bar.Dy())
}

func (e breezeEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	vertical := track.Dy() >= track.Dx()
	ss := ScrollState{Disabled: st.Disabled(), Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed, ss.Hot = ScrollThumbPart, ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, vertical, ss)
}

// ---- frames -------------------------------------------------------------------------------------------

func (breezeEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top += l.body.Height() + l.S(4)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a frame-background panel in the frame outline with the
// title centred inside its top edge.
func (e breezeEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := breezeColors(l)
	c.frame(l, ctx, b, c.frameBg, c.outline)
	if title == "" {
		return
	}
	f := l.body
	tb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y+l.S(4), b.Dx()-l.S(16), f.Height()+l.S(2))
	l.drawFittedText(ctx, f, title, tb, c.text, AlignCenter, 0)
}

// brTitleH is the KWin Breeze decoration's title bar height at the UI font.
func brTitleH(l *Classic) float32 {
	h := snap(l.body.Height() + l.S(10))
	if h < snap(l.S(24)) {
		h = snap(l.S(24))
	}
	return h
}

func (breezeEngine) WindowFrameInsets(l *Classic) Insets {
	u := brU(l)
	return Insets{Top: brTitleH(l), Right: u, Bottom: u, Left: u}
}

// WindowCloseRect is the decoration's close button, the circle at the
// right end of the title bar.
func (breezeEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	h := brTitleH(l)
	if b.Dy() < h || b.Dx() < h*2 {
		return paintengine2d.Rect{}
	}
	side := snap(l.S(18))
	if side > h-l.S(4) {
		side = snap(h - l.S(4))
	}
	return paintengine2d.XYWH(snap(b.Max.X-side-l.S(6)), snap(b.Min.Y+(h-side)*0.5), side, side)
}

// DrawWindowFrame is the KWin Breeze decoration: a flat title bar in the WM
// colour with a centred title and the close button (its × on the bar; a
// red circle when hot, darker when held), a separator above the window.
func (e breezeEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	h := brTitleH(l)
	if b.Dx() < 6*u || b.Dy() < h {
		return
	}
	r := l.rx(3)
	bar, fg := c.title, c.titleText
	if !st.Active {
		bar, fg = c.titleOff, c.titleTextOff
	}
	ctx.DrawRoundRectCorners(b, r, r, 0, 0, paintengine2d.Fill(c.win))
	ctx.DrawRoundRectCorners(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), h), r, r, 0, 0, paintengine2d.Fill(bar))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y+h-u, u, c.sep)
	ctx.DrawRoundRectCorners(b.Inset(u*0.5), r, r, 0, 0, paintengine2d.StrokePaint(Mix(c.win, c.text, 0.35).WithAlpha(0.6), u))
	right := b.Max.X - l.S(8)
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
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

// brPaneTop is where a tab pane's top edge sits: on the last row of the
// tab bar (the selected tab covers it).
func brPaneTop(l *Classic, b paintengine2d.Rect) float32 {
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		return b.Min.Y
	}
	return snap(b.Min.Y + th - brU(l))
}

// DrawTabPane is the frame background in the frame outline, square at the
// top-left where the tab bar starts.
func (breezeEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	top := brPaneTop(l, b)
	pane := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if pane.Dx() < 4*u || pane.Dy() < 4*u {
		return
	}
	r := brR(l)
	tl := r
	if top > b.Min.Y {
		tl = 0
	}
	in := pane.Inset(u * 0.5)
	ctx.DrawRoundRectCorners(in, tl, r, r, r, paintengine2d.Fill(c.frameBg))
	ctx.DrawRoundRectCorners(in, tl, r, r, r, paintengine2d.StrokePaint(c.outline, u))
}

// ---- controls ----------------------------------------------------------------------------------------------

func (e breezeEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := breezeColors(l)
	if raised {
		c.frame(l, ctx, b, c.frameBg, c.outline)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	c.frame(l, ctx, b, c.base, c.outline)
}

func (e breezeEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := breezeColors(l)
	b = fuSnap(b)
	c.button(l, ctx, b, st, st.Primary())
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, l.S(12))
}

func (e breezeEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := breezeColors(l)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			e.DrawFocusRing(l, ctx, box.Intersect(b))
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gap := l.S(4)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusLine(l, ctx, l.body, label, lb, b, AlignStart)
	}
}

func (e breezeEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	if side > b.Dy() {
		side = b.Dy()
	}
	box := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e breezeEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	if side > b.Dy() {
		side = b.Dy()
	}
	box := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch is the Plasma (QQC2 Breeze) switch: a pill in the frame
// colours — the highlight at 33% in a highlight outline when on — with a
// slider-style round handle that slides to the checked end.
func (e breezeEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := breezeColors(l)
	u := brU(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if tw <= 0 {
		tw = l.S(36)
	}
	if th <= 0 {
		th = l.S(20)
	}
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 6*u || track.Dy() < 6*u {
		return
	}
	pill := paintengine2d.XYWH(track.Min.X+u, track.Min.Y+track.Dy()*0.2, track.Dx()-2*u, track.Dy()*0.6)
	pill = fuSnap(pill)
	r := pill.Dy() * 0.5
	if l.square() {
		r = 0
	}
	in := pill.Inset(u * 0.5)
	enabled := !st.Disabled()
	switch {
	case on && enabled:
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(Mix(c.win, c.hl, 0.33)))
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.hl, u))
	default:
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(Mix(c.win, c.text, 0.1)))
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.groove, u))
	}
	hs := track.Dy()
	hx := track.Min.X
	if on {
		hx = track.Max.X - hs
	}
	c.sliderHandle(l, ctx, paintengine2d.XYWH(hx, track.Min.Y, hs, hs), st)
	if label != "" {
		fg := c.text
		if !enabled {
			fg = c.dis
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
		if st.Focused() && enabled {
			c.focusLine(l, ctx, l.body, label, lb, b, AlignStart)
		}
	}
}

// sliderHandle is the slider handle: a circle one pixel inside hb in the
// button colour, outlined (highlight when hot or focused) over an
// ellipse shadow unless held.
func (c *breeze) sliderHandle(l *Classic, ctx *paintengine2d.Context, hb paintengine2d.Rect, st ControlState) {
	u := brU(l)
	side := hb.Dx()
	if hb.Dy() < side {
		side = hb.Dy()
	}
	r := side*0.5 - u
	if r < 2*u {
		return
	}
	ctr := hb.Center()
	enabled := !st.Disabled()
	if enabled && !st.Pressed() {
		ctx.DrawCircle(paintengine2d.Pt(ctr.X+u*0.35, ctr.Y+u*0.35), r, paintengine2d.StrokePaint(c.shadow, u))
	}
	line := c.sliderLine
	switch {
	case !enabled:
		line = Mix(c.win, c.text, 0.25)
	case st.Focused() || st.Hovered() || st.Pressed():
		line = c.hl
	}
	fill := c.btn
	if !enabled {
		fill = c.btnDis
	}
	ctx.DrawCircle(ctr, r-u*0.5, paintengine2d.Fill(fill))
	ctx.DrawCircle(ctr, r-u*0.5, paintengine2d.StrokePaint(line, u))
}

// brGroove is the slider / progress groove: a 6px rounded groove outlined
// in its colour, filled with it at half strength.
func brGroove(l *Classic, ctx *paintengine2d.Context, g paintengine2d.Rect, line, fill paintengine2d.Color) {
	u := brU(l)
	if g.Dx() < 2*u || g.Dy() < 2*u {
		return
	}
	r := g.Dy() * 0.5
	if g.Dx() < g.Dy() {
		r = g.Dx() * 0.5
	}
	in := g.Inset(u * 0.5)
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(line, u))
}

// DrawSlider is the groove in text at 20%, the highlight from
// the start to the handle's centre, the round handle.
func (e breezeEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := breezeColors(l)
	b = fuSnap(b)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	hs := l.metrics.Thumb
	if hs <= 0 {
		hs = l.S(20)
	}
	if hs > b.Dy() {
		hs = b.Dy()
	}
	hs = snap(hs)
	if b.Dx() < hs || hs < 6 {
		return
	}
	cy := b.Min.Y + b.Dy()*0.5
	gt := snap(l.S(6))
	g := paintengine2d.XYWH(b.Min.X, snap(cy-gt*0.5), b.Dx(), gt)
	hx := snap(b.Min.X + (b.Dx()-hs)*t)
	mid := hx + hs*0.5
	if st.Disabled() {
		brGroove(l, ctx, g, c.groove, c.grooveFill)
	} else {
		brGroove(l, ctx, paintengine2d.XYWH(mid, g.Min.Y, g.Max.X-mid, g.Dy()), c.groove, c.grooveFill)
		brGroove(l, ctx, paintengine2d.XYWH(g.Min.X, g.Min.Y, mid-g.Min.X, g.Dy()), c.hl, c.hlGrooveFill)
	}
	c.sliderHandle(l, ctx, paintengine2d.XYWH(hx, snap(cy-hs*0.5), hs, hs), st)
}

// DrawProgressBar is a 6px groove in text at 20% and the highlight bar
// over it; busy bars alternate 14px highlight /
// pale segments that scroll.
func (e breezeEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := breezeColors(l)
	b = fuSnap(b)
	gt := snap(l.S(6))
	if gt > b.Dy() {
		gt = b.Dy()
	}
	g := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-gt)*0.5), b.Dx(), gt)
	if g.Dx() < 4 || g.Dy() < 2 {
		return
	}
	brGroove(l, ctx, g, c.groove, Mix(c.win, c.text, 0.1))
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	line, fill := c.hl, c.progFill
	if st.Disabled() {
		line, fill = Mix(c.win, c.text, 0.35), Mix(c.win, c.text, 0.2)
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		seg := l.S(14)
		r := g.Dy() * 0.5
		ctx.Save()
		ctx.ClipRoundRect(g, r, r)
		ctx.DrawRect(g, paintengine2d.Fill(c.busyAlt))
		for x := g.Min.X - 2*seg + phase*2*seg; x < g.Max.X; x += 2 * seg {
			ctx.DrawRect(paintengine2d.XYWH(x, g.Min.Y, seg, g.Dy()), paintengine2d.Fill(line))
		}
		ctx.Restore()
		return
	}
	w := snap(g.Dx() * t)
	if w <= 0 {
		return
	}
	bar := paintengine2d.XYWH(g.Min.X, g.Min.Y, w, g.Dy())
	if w < g.Dy() {
		// A bar shorter than its thickness keeps its round start (the
		// contents rect grows to the thickness and is clipped).
		ctx.Save()
		ctx.ClipRect(bar)
		brGroove(l, ctx, paintengine2d.XYWH(g.Min.X, g.Min.Y, g.Dy(), g.Dy()), line, fill)
		ctx.Restore()
		return
	}
	brGroove(l, ctx, bar, line, fill)
}

// DrawComboBox is a non-editable combo: the push-button frame, the text
// and the chevron in a 20px column at the right.
func (e breezeEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := breezeColors(l)
	b = fuSnap(b)
	bst := st
	if open {
		bst |= StatePressed
	}
	c.button(l, ctx, b, bst, false)
	aw := l.S(20)
	ab := paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy())
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
	tb := paintengine2d.XYWH(b.Min.X+l.S(10), b.Min.Y, ab.Min.X-b.Min.X-l.S(12), b.Dy())
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
}

// DrawSpinner is the spin box arrow column: it continues the field's
// frame (view colour, frame outline; focus / hover colour) and stacks the
// up and down chevrons, each turning the hover colour when hot.
func (e breezeEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	f := paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx()-u, b.Dy()-2*u)
	if f.Dx() < 4*u || f.Dy() < 6*u {
		return
	}
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

func (e breezeEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	// The frame outline along the bottom (the bar itself is transparent).
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	if b.Dy() < 2*u {
		return
	}
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.outline)
}

// DrawTab (tabs on top): the selected tab is
// the window colour in the frame outline with rounded top corners and a
// 3px highlight strip along its top, opening into the pane; the others are
// the window darker 120%, rounded only at the ends of the strip, washed with
// the hover colour at 20% when hot.
func (e breezeEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	r := l.rx(3)
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
		ctx.DrawRoundRectCorners(in, rr, rr, 0, 0, paintengine2d.StrokePaint(Mix(c.win, c.text, 0.25), u))
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

// DrawMenuBar: the tools area is the Header colour (Plasma 5.21+).
func (e breezeEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(breezeColors(l).header))
}

// DrawMenuTitle (strong menu focus): the hovered title is
// the hover colour, the open one the focus colour with highlighted text.
func (e breezeEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := breezeColors(l)
	b = fuSnap(b)
	fg := c.headerText
	switch {
	case st.Disabled():
		fg = c.dis
	case open || st.Pressed():
		ctx.DrawRoundRect(b, brR(l), brR(l), paintengine2d.Fill(c.focus))
		fg = c.hlText
	case st.Hovered():
		ctx.DrawRoundRect(b, brR(l), brR(l), paintengine2d.Fill(Mix(c.header, c.hover, 0.4)))
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		c.focusLine(l, ctx, l.body, label, b, b, AlignCenter)
	}
}

// DrawMenuFrame is a rounded frame-background panel in the frame outline.
func (e breezeEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := breezeColors(l)
	u := brU(l)
	r := brR(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	in := b.Inset(u * 0.5)
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.frameBg))
	ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.outline, u))
}

// DrawMenuItem is the rounded focus highlight, Breeze check
// boxes and radios in the check column, the shortcut at 70% and a chevron
// for submenus; labels stay in the window text.
func (e breezeEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := breezeColors(l)
	u := brU(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := b.Min.X - ch.PadL + l.S(5)
		x1 := b.Max.X + ch.PadR - l.S(5)
		fuHLine(ctx, x0, x1, y, u, c.sep)
		return
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gw := ch.CheckCol()
	side := snap(l.S(20))
	if side > b.Dy() {
		side = snap(b.Dy())
	}
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
		if st.Disabled() {
			col = c.dis
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

// row paints an item view row's background for state st and returns its
// label colour: the highlight (lighter when also hovered), the highlight
// at 20% for hover. KDE keeps a selection when only the view loses focus;
// an inactive window (Backdrop) shows the scheme's inactive selection
// colour.
func (c *breeze) row(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	bg, fg := c.hl, c.hlText
	switch {
	case st.Disabled():
		bg, fg = c.hlDis, c.hlText
	case st.Backdrop():
		bg, fg = c.hlOff, c.hlOffText
	}
	hot := st.Hovered() && !st.Disabled()
	switch {
	case st.Checked() && hot:
		ctx.DrawRect(b, paintengine2d.Fill(lighterPct(bg, 110)))
		return fg
	case st.Checked():
		ctx.DrawRect(b, paintengine2d.Fill(bg))
		return fg
	case hot:
		ctx.DrawRect(b, paintengine2d.Fill(c.hl.WithAlpha(0.2)))
	}
	if st.Disabled() {
		return c.dis
	}
	return c.viewText
}

func (e breezeEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := breezeColors(l)
	fg := c.row(ctx, b, st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow: the full-row selection, branch lines in text at 25% over
// the view and the chevron expander.
func (e breezeEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := breezeColors(l)
	u := brU(l)
	fg := c.row(ctx, b, st)
	selected, hovered := st.Checked(), st.Hovered() && !st.Disabled()
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	aw := l.S(14)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	line := c.branch
	if selected {
		line = Mix(c.hl, c.hlText, 0.35)
	}
	// ViewDrawTreeBranchLines: one vertical per ancestor, the elbow for the
	// item.
	for d := 0; d < depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + aw*0.5)
		switch {
		case d == depth-1 && !st.HasNextSibling(depth):
			fuVLine(ctx, gx, b.Min.Y, cy, u, line) // down to the last child's elbow
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
			col = c.hlText
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
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x+aw+l.S(4), b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e breezeEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := breezeColors(l)
	u := brU(l)
	fg := c.row(ctx, b, st)
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
	avail := b.Dx() - pad*2
	if avail < 4 {
		avail = 4
	}
	label = f.Fit(label, avail)
	tw := f.Advance(label)
	x := b.Min.X + pad
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - pad
	}
	ctx.Save()
	ctx.ClipRect(b.Inset(u))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	// Grid lines in the separator colour at low strength.
	grid := Mix(c.base, c.viewText, 0.1)
	fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y, u, grid)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, grid)
}

// DrawTableHeader is the button colour (hover / focus
// tinted at 20%), a bottom line in text at 10%, a separator in text at 20%
// and the chevron sort indicator.
func (e breezeEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	bg := c.btn
	switch {
	case st.Disabled():
	case st.Pressed():
		bg = c.headerDown
	case st.Hovered():
		bg = c.headerHot
	}
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.headerLine)
	if !st.Last() {
		fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y-u, u, c.headerSep)
	}
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		brArrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy()), dir, c.arrowBtn)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(12)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar: the Header colour of the tools area with its separator
// along the bottom.
func (e breezeEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.header))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, Mix(c.header, c.headerText, 0.25))
}

// DrawStatusBar: the window colour and plain texts.
func (e breezeEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := breezeColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	if n := len(parts); n > 0 {
		slot := b.Dx() / float32(n)
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(10), b.Min.Y, slot-l.S(16), b.Dy()), c.text, AlignStart, 0)
		}
	}
}

// DrawTitleBar (panel headings): the Header colour with a separator below.
func (e breezeEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.header))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, Mix(c.header, c.headerText, 0.25))
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.headerText, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), l.palette.TextMuted, AlignStart, 0)
	}
}

// DrawAccordionHeader is a tool box tab: flat, the chevron and the label;
// hover and focus outline it in the highlight, a press washes it.
func (e breezeEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := breezeColors(l)
	u := brU(l)
	b = fuSnap(b)
	if !c.flat(l, ctx, b, st) {
		fuHLine(ctx, b.Min.X+u, b.Max.X-u, b.Max.Y-u, u, c.sep)
	}
	fg := c.text
	col := c.arrow
	if st.Disabled() {
		fg, col = c.dis, c.dis
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	brArrow(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, l.S(12), b.Dy()), dir, col)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(28), b.Min.Y, b.Dx()-l.S(32), b.Dy()), fg, AlignStart, 0)
}

// DrawSeparator is one line in the separator colour.
func (e breezeEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := breezeColors(l)
	u := brU(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		m := l.S(2)
		fuVLine(ctx, x, b.Min.Y+m, b.Max.Y-m, u, c.sep)
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	fuHLine(ctx, b.Min.X, b.Max.X, y, u, c.sep)
}

// DrawSplitter: Breeze splitters are a one-pixel separator
// (Splitter_SplitterWidth 1) that turns the hover colour when hot.
func (e breezeEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := breezeColors(l)
	u := brU(l)
	col := c.sep
	if st.Hovered() && !st.Disabled() {
		col = c.hover
	}
	if vertical {
		fuVLine(ctx, snap((b.Min.X+b.Max.X)*0.5)-u, b.Min.Y, b.Max.Y, u, col)
		return
	}
	fuHLine(ctx, b.Min.X, b.Max.X, snap((b.Min.Y+b.Max.Y)*0.5)-u, u, col)
}

// DrawTooltip is a rounded tool-tip-coloured frame in mix(tip, tipText,
// 0.25).
func (e breezeEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := breezeColors(l)
	u := brU(l)
	r := brR(l)
	if b.Dx() > 3*u && b.Dy() > 3*u {
		in := b.Inset(u * 0.5)
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.tip))
		ctx.DrawRoundRect(in, r, r, paintengine2d.StrokePaint(c.tipLine, u))
	}
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs -----------------------------------------------------------------------------------------------------

// breezeScheme is one Plasma colour scheme's published colours: the
// background and text of each colour set (window, view, button, selection,
// tooltip, header), the window set's inactive / negative / neutral /
// positive text, the title bar colours and the focus / hover decoration.
type breezeScheme struct {
	window, windowText, view, viewText, button, buttonText string
	selection, selectionText, tooltip, tooltipText         string
	header, headerText, inactiveText                       string
	negative, neutral, positive, link, focus, hover        string
	wmActive, wmActiveText, wmInactive, wmInactiveText     string
	selectionAlt                                           string
}

// Breeze Light (Plasma 5.27).
var breezeLightScheme = breezeScheme{
	window: "#eff0f1", windowText: "#232629", view: "#ffffff", viewText: "#232629",
	button: "#fcfcfc", buttonText: "#232629", selection: "#3daee9", selectionText: "#ffffff",
	tooltip: "#f7f7f7", tooltipText: "#232629", header: "#dee0e2", headerText: "#232629",
	inactiveText: "#707d8a", negative: "#da4453", neutral: "#f67400", positive: "#27ae60",
	link: "#2980b9", focus: "#3daee9", hover: "#3daee9",
	wmActive: "#e3e5e7", wmActiveText: "#232629", wmInactive: "#eff0f1", wmInactiveText: "#707d8a",
	selectionAlt: "#a3d4fa",
}

// Breeze Dark (Plasma 5.27).
var breezeDarkScheme = breezeScheme{
	window: "#2a2e32", windowText: "#fcfcfc", view: "#1b1e20", viewText: "#fcfcfc",
	button: "#31363b", buttonText: "#fcfcfc", selection: "#3daee9", selectionText: "#fcfcfc",
	tooltip: "#31363b", tooltipText: "#fcfcfc", header: "#31363b", headerText: "#fcfcfc",
	inactiveText: "#a1a9b1", negative: "#da4453", neutral: "#f67400", positive: "#27ae60",
	link: "#1d99f3", focus: "#3daee9", hover: "#3daee9",
	wmActive: "#31363b", wmActiveText: "#fcfcfc", wmInactive: "#2a2e32", wmInactiveText: "#a1a9b1",
	selectionAlt: "#1e5774",
}

func (s breezeScheme) palette() Palette {
	win, text, view := hexColor(s.window), hexColor(s.windowText), hexColor(s.view)
	btn, sel := hexColor(s.button), hexColor(s.selection)
	outline := Mix(win, text, 0.25)
	frameBg := Mix(win, view, 0.3)
	return Palette{
		Background: win, Surface: win, SurfaceAlt: btn,
		Border: outline, Divider: outline,
		Text: text, TextMuted: hexColor(s.inactiveText), TextOnAccent: hexColor(s.selectionText),
		Accent: sel, AccentHover: hexColor(s.hover), AccentPress: Mix(btn, sel, 0.333),
		Field: view, FieldBorder: outline,
		Focus: hexColor(s.focus), Selection: sel,
		Track: Mix(win, text, 0.1), Thumb: Mix(win, text, 0.25),
		Highlight: sel.WithAlpha(0.2), Shadow: paintengine2d.RGBA(0, 0, 0, 0.125),
		MenuHover: Mix(frameBg, hexColor(s.focus), 0.3), MenuHoverBorder: Mix(hexColor(s.focus), text, 0.15), MenuGutter: frameBg,
		Danger: hexColor(s.negative), Success: hexColor(s.positive), Warning: hexColor(s.neutral),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: Shade(win, 0.5), BevelDark: outline,
	}
}

func (s breezeScheme) extra() map[string]paintengine2d.Color {
	return map[string]paintengine2d.Color{
		"button": hexColor(s.button), "buttonText": hexColor(s.buttonText),
		"view": hexColor(s.view), "viewText": hexColor(s.viewText),
		"tip": hexColor(s.tooltip), "tipText": hexColor(s.tooltipText),
		"header": hexColor(s.header), "headerText": hexColor(s.headerText),
		"titleBar": hexColor(s.wmActive), "titleText": hexColor(s.wmActiveText),
		"titleBarOff": hexColor(s.wmInactive), "titleTextOff": hexColor(s.wmInactiveText),
		"focus": hexColor(s.focus), "hover": hexColor(s.hover), "negative": hexColor(s.negative),
		"selectionInactive": hexColor(s.selectionAlt),
	}
}

func breezePack(name, label, summary string, fam ThemeName, s breezeScheme) ThemePack {
	pal := s.palette()
	tok := ThemeTokens{
		Engine:  "breeze",
		Bevel:   BevelNone,
		Family:  fam,
		Palette: pal,
		Era:     EraBreeze,
		Extra:   s.extra(),
	}
	breezeChrome(&tok)
	return ThemePack{
		Name: name, Label: label, Year: 2014, Lineage: "KDE", Summary: summary,
		Era: EraBreeze, Palette: fam, Tokens: tok,
	}
}

// breezeChrome sets the chrome states Breeze derives from its palette.
func breezeChrome(tok *ThemeTokens) {
	pal := tok.Palette
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: pal.AccentPress, Border: pal.Accent}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
}

// Accented is Plasma's accent colour (5.25 onwards): the selection and the
// focus and hover decorations take it, and everything Breeze mixes from
// them follows (pressed buttons, checked marks, progress, menus).
func (breezeEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	tok = CloneTokenMaps(tok)
	p := &tok.Palette
	frameBg := Mix(p.Background, p.Field, 0.3)
	p.Accent, p.AccentHover, p.Selection, p.Focus = accent, accent, accent, accent
	p.AccentPress = Mix(p.SurfaceAlt, accent, 0.333)
	p.Highlight = accent.WithAlpha(0.2)
	p.MenuHover = Mix(frameBg, accent, 0.3)
	p.MenuHoverBorder = Mix(accent, p.Text, 0.15)
	// Breeze keeps white on its own #3daee9 (2.5:1); a pale accent gets
	// the text colour instead.
	p.TextOnAccent = ReadableOn(accent, 2.2, p.TextOnAccent, p.Text)
	tok.Extra["focus"], tok.Extra["hover"] = accent, accent
	tok.Extra["selectionInactive"] = Mix(accent, p.Field, 0.5)
	breezeChrome(&tok)
	return tok
}

func breezePacks() []ThemePack {
	return []ThemePack{
		breezePack("breeze", "Breeze", "KDE Plasma 5's flat style: 3px frames mixed from the text colour, #3daee9 focus and hover, round slider handles.", ThemeLight, breezeLightScheme),
		breezePack("breeze-night", "Breeze Dark", "Plasma 5's Breeze in the Breeze Dark colour scheme (Plasma 5.27 values).", ThemeDark, breezeDarkScheme),
	}
}
