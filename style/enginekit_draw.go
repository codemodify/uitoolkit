package style

// Small drawing helpers more than one engine needs: hairlines that land
// on the pixel grid, a tick stroked rather than filled, and the rest of
// the odds and ends that were written inside whichever engine wanted
// them first.
//
// No build tag, so everything here is in every build.

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// fuLine fills a horizontal or vertical 1u line.
func fuHLine(ctx *paintengine2d.Context, x0, x1, y, u float32, col paintengine2d.Color) {
	if x1 > x0 {
		ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, u), paintengine2d.Fill(col))
	}
}

// fuSnap rounds a rect to the pixel grid.
func fuSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

func fuVLine(ctx *paintengine2d.Context, x, y0, y1, u float32, col paintengine2d.Color) {
	if y1 > y0 {
		ctx.DrawRect(paintengine2d.XYWH(x, y0, u, y1-y0), paintengine2d.Fill(col))
	}
}

// strokeTick strokes the check mark a → v → b in col with butt caps, each
// end pushed out by half the pen along its arm so the ends read square.
// (The ends are extended by hand: square caps on an open polyline leave a
// stray band across the mark in paintengine2d's stroker.)
func strokeTick(ctx *paintengine2d.Context, a, v, b paintengine2d.Point, w float32, col paintengine2d.Color, join paintengine2d.Join) {
	if col.A <= 0 || w <= 0 {
		return
	}
	out := func(e paintengine2d.Point) paintengine2d.Point {
		dx, dy := e.X-v.X, e.Y-v.Y
		n := float32(math.Hypot(float64(dx), float64(dy)))
		if n <= 0 {
			return e
		}
		return paintengine2d.Pt(e.X+dx/n*w*0.5, e.Y+dy/n*w*0.5)
	}
	a, b = out(a), out(b)
	p := paintengine2d.NewPath()
	p.MoveTo(a.X, a.Y)
	p.LineTo(v.X, v.Y)
	p.LineTo(b.X, b.Y)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapButt, Join: join, MiterLimit: 4}})
}

// flatCentered is a w×h rect centred in b, on whole pixels.
func flatCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return flatSnap(paintengine2d.XYWH((b.Min.X+b.Max.X-w)*0.5, (b.Min.Y+b.Max.Y-h)*0.5, w, h))
}

// fluentTabPath is the selected tab's outline: 8px top corners and
// concave bottom corners (ear e) that meet the strip's bottom line. closed
// also runs along the bottom, for filling.
func fluentTabPath(b paintengine2d.Rect, top, r, e float32, closed bool) *paintengine2d.Path {
	const k = 0.5522847
	x0, x1 := b.Min.X+e, b.Max.X-e
	y1 := b.Max.Y
	r = min(r, (x1-x0)*0.5, (y1-top)*0.5)
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X, y1)
	// Concave ear up to the tab side.
	p.CubicTo(b.Min.X+e*k, y1, x0, y1-e+e*k, x0, y1-e)
	p.LineTo(x0, top+r)
	p.CubicTo(x0, top+r-r*k, x0+r-r*k, top, x0+r, top)
	p.LineTo(x1-r, top)
	p.CubicTo(x1-r+r*k, top, x1, top+r-r*k, x1, top+r)
	p.LineTo(x1, y1-e)
	p.CubicTo(x1, y1-e+e*k, b.Max.X-e*k, y1, b.Max.X, y1)
	if closed {
		p.Close()
	}
	return p
}

// md3xRoles gives a Material 3 colour set the roles of the 2023 spec on:
// the tone-based surfaces (surface at tone 98 / 6), the containers for
// elevation (cards and the drawer surface-container-low, menus and bars
// surface-container, dialogs surface-container-high), 10% focus and pressed
// layers and Expressive's secondary-container inactive track. A pack pins
// any role by name.
func md3xRoles(l *Classic, c *mdSet) {
	n := mdCorePalette(l.X("seed", Hex("#6750a4"))).neutral
	tone := func(k string, light, dark float64) paintengine2d.Color {
		if c.dark {
			return l.X(k, n.tone(dark))
		}
		return l.X(k, n.tone(light))
	}
	c.surface = tone("surface", 98, 6)
	c.bg = tone("background", 98, 6)
	c.card = tone("surfaceContainerLow", 96, 10)
	c.menu = tone("surfaceContainer", 94, 12)
	c.bar = c.menu
	c.dialog = tone("surfaceContainerHigh", 92, 17)
	c.textDis = mdOver(c.surface, c.onSurface, 0.38)
	c.aFocus, c.aPress = 0.10, 0.10
	c.trackOff = c.secondaryC
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

// adwCardR is the 12px corner of cards, boxed lists and frames, which GNOME
// 48 kept, for a shape of size b.
func adwCardR(l *Classic, b paintengine2d.Rect) float32 {
	return min(l.rx(12), adwPill(l, b))
}

// adwEraR is GNOME 42–47's design radius r as the look's GNOME draws it:
// GNOME 48 rounded the 6px controls, rows and menus to 9px and the 12px
// popovers, dialogs and windows to 15px.
func adwEraR(l *Classic, r float32) float32 {
	if l != nil && l.P("era", 0) >= 48 {
		switch r {
		case 6:
			return 9
		case 12:
			return 15
		}
	}
	return r
}

// flatSnap puts b on whole pixels; edges round half down, so the rect never
// covers a pixel whose centre lies outside b.
func flatSnap(b paintengine2d.Rect) paintengine2d.Rect {
	f := func(v float32) float32 { return float32(math.Ceil(float64(v) - 0.5 - 1e-3)) }
	x0, y0, x1, y1 := f(b.Min.X), f(b.Min.Y), f(b.Max.X), f(b.Max.Y)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(x1, y1)}
}

// skinEngineID is the engine registry id and the value of a skin pack's
// "engine" token. It is deliberately neutral: this is the toolkit's skin
// format, not an imitation of any one player's.
const skinEngineID = "skin"

// skinHitKey identifies one derived mask: the sprite, the box it was fitted
// to, where in that box it was drawn, and the display scale that fitted it.
type skinHitKey struct {
	sprite *SkinSprite
	w, h   int
	at     paintengine2d.Rect
	scale  float32
	// panel is whether the sprite is painted as a panel's piece
	// (panelDraw) rather than as a control's face.
	panel bool
}

// skinRolePart maps an engine Role onto the part name a skin binds it with.
var skinRolePart = map[Role]string{
	RoleButton:   "button",
	RoleTool:     "tool",
	RoleField:    "field",
	RoleCheck:    "check",
	RoleRow:      "row",
	RoleTab:      "tab",
	RoleThumb:    "thumb",
	RoleTrack:    "track",
	RoleMenu:     "menu",
	RoleCombo:    "combo",
	RoleSplitter: "splitter",
	RoleBar:      "bar",
	RolePanel:    "panel",
}

// skinStateName is the state a control is in, as the manifest names it.
//
// The order is the order a person would read the control: disabled first
// because it outranks everything, then the on/off axis, then the pointer,
// then focus, then the default button, then a backdrop window.
func skinStateName(st ControlState) string {
	switch {
	case st.Disabled():
		return "disabled"
	case st.Checked() && st.Pressed():
		return "checkedPressed"
	case st.Checked() && st.Hovered():
		return "checkedHover"
	case st.Checked():
		return "checked"
	// The default button keeps its own face under the pointer. Its art is
	// the accent, and swapping it for the ordinary hover face would make
	// the one button the dialog is steering you to look like the others.
	case st.Primary() && st.Pressed():
		return "defaultPressed"
	case st.Primary() && st.Hovered():
		return "defaultHover"
	case st.Primary():
		return "default"
	case st.Pressed():
		return "pressed"
	case st.Hovered():
		return "hover"
	case st.Focused():
		return "focus"
	case st.Inactive() || st.Backdrop():
		return "inactive"
	}
	return "normal"
}

func flat8(v float64) float32 { return float32(math.Floor(math.Min(1, math.Max(0, v))*255+0.5)) / 255 }

// adwPill is the radius that rounds b into a stadium (0 for square looks).
func adwPill(l *Classic, b paintengine2d.Rect) float32 {
	if l.square() {
		return 0
	}
	return max(min(b.Dx(), b.Dy())*0.5, 0)
}

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

	// frameR is the frame radius for a 1px pen: 2.5 (3px frames), 4.5 on
	// Plasma 6 (5px).
	frameR float32
}

// lunaCross fills the caption × (CloseGlyph.bmp): two bars w thick from
// corner to corner of b, as quads so the ends stay flat and whole.
func lunaCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if b.Empty() || w <= 0 {
		return
	}
	fill := paintengine2d.Fill(col)
	for _, d := range [2][4]float32{{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}, {b.Max.X, b.Min.Y, b.Min.X, b.Max.Y}} {
		x0, y0, x1, y1 := d[0], d[1], d[2], d[3]
		dx, dy := x1-x0, y1-y0
		n := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		nx, ny := -dy/n*w*0.5, dx/n*w*0.5
		p := paintengine2d.NewPath()
		p.MoveTo(x0+nx, y0+ny)
		p.LineTo(x1+nx, y1+ny)
		p.LineTo(x1-nx, y1-ny)
		p.LineTo(x0-nx, y0-ny)
		p.Close()
		ctx.DrawPath(p, fill)
	}
}

// adwCentered is a w×h rect centred in b.
func adwCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return paintengine2d.XYWH(b.Min.X+(b.Dx()-w)*0.5, b.Min.Y+(b.Dy()-h)*0.5, w, h)
}

// adwRing strokes a w-wide ring just inside b, following radius r.
func adwRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, w float32, col paintengine2d.Color) {
	if b.Dx() < 2*w || b.Dy() < 2*w || col.A <= 0 {
		return
	}
	ri := max(r-w*0.5, 0)
	ctx.DrawRoundRect(b.Inset(w*0.5), ri, ri, paintengine2d.StrokePaint(col, w))
}

// grayLevel is Qt's documented qGray weighting, (11r + 16g + 5b) / 32, on
// 0..255.
func grayLevel(c paintengine2d.Color) float32 {
	return (clamp1(c.R)*11 + clamp1(c.G)*16 + clamp1(c.B)*5) / 32 * 255
}

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

// b6R is the 5px corner of Plasma 6's rows, tabs and window frames.
func b6R(l *Classic) float32 { return l.rx(5) }

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

// breeze6Margin is the room buttons, fields and combos keep round their face
// for the focus band: 2px.
func breeze6Margin(l *Classic) float32 { return 2 * brU(l) }

// brR is the frame radius (3px, Plasma 6's 5px) for a 1px pen stroked half
// a pixel in.
func brR(l *Classic) float32 { return l.rx(breezeColors(l).frameR) }

// brU is one design pixel at the look's scale (never under a device pixel).
func brU(l *Classic) float32 {
	u := snap(l.S(1))
	if u < 1 {
		u = 1
	}
	return u
}

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
	c.frameR = 2.5
	if l.P("plasma", 5) >= 6 {
		breeze6Mixes(c)
	}
	return c
}

type breezeKey struct{}
