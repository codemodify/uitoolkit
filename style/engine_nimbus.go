package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// nimbusEngine paints Nimbus, the vector look Java 6 Update 10 introduced
// in 2008: glossy rounded buttons over a cool grey canvas, the soft blue
// focus ring, blue glossy check boxes, radios, tabs and default buttons,
// capsule scroll thumbs, square inset text fields, an orange progress bar
// that stripes while indeterminate, striped table rows and dark blue
// selections with white text.
//
// Colours start from the published Nimbus UIManager keys (nimbusBase,
// nimbusBlueGrey, control, nimbusFocus, nimbusSelectionBackground,
// nimbusOrange …); the gradients are stops measured off screenshots of the
// shipped look. Every painter is written here from those observations.
//
// Controls keep a two-pixel margin inside their rect for the focus ring —
// nimbusFocus with a pale halo round it — and for the one-pixel shadow under
// buttons, as Nimbus does.
//
// Pack data:
//
//	extra   "nimbusBase", "nimbusBlueGrey", "control", "nimbusFocus",
//	        "nimbusSelectionBackground", "nimbusSelectedText", "text",
//	        "nimbusDisabledText", "nimbusLightBackground", "nimbusBorder",
//	        "nimbusOrange", "nimbusRed", "nimbusInfoBlue",
//	        "nimbusAlertYellow", "menu", "alternateRow",
//	        "scrollbar", "activeCaption", "inactiveCaption".
//	params  "stripes" 0 = plain table rows.
type nimbusEngine struct{ BaseEngine }

func init() {
	RegisterEngine(nimbusEngine{})
	for _, p := range nimbusPacks() {
		RegisterPack(p)
	}
}

func (nimbusEngine) ID() string { return "nimbus" }

// DefaultMetrics are Nimbus's sizes at its 12pt SansSerif (the toolkit's
// 16px font): 15px scroll bars, 18px check box and radio icons (a 14px box
// in room for the focus ring), 17px slider knob, 19px progress bar, rounded
// corners.
func (nimbusEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Radius: 6, RadiusSmall: 4, ViewFrame: 2,
		ControlH: 30, FieldH: 30, ComboH: 30,
		Checkbox: 18, Radio: 18,
		MenuItemH: 24, MenuBarH: 26, TabH: 30, RowH: 22,
		TitleBar: 28, HeaderH: 24, ProgressH: 19, SliderH: 26, Thumb: 17,
		Scroll: 15, Pad: 10, FieldPad: 6, FocusWidth: 2, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 20,
		SwitchW: 40, SwitchH: 22,
	}
}

// ---- resolved colours ---------------------------------------------------------

// nbFace is one glossy material: its face and border gradients (top to
// bottom) and the line under it (a soft shadow, or a highlight when
// pressed).
type nbFace struct {
	face, edge []paintengine2d.GradientStop
	under      paintengine2d.Color
}

// nb is a look's resolved Nimbus colours and materials, built once per
// look.
type nb struct {
	u       float32 // one Nimbus pixel in device pixels
	stripes bool

	base, blueGrey, control, focus, halo, sel, selText   paintengine2d.Color
	text, dis, light, border, orange, red, infoBlue      paintengine2d.Color
	yellow, menuBg, alt, scroll, capOn, capOff           paintengine2d.Color
	fieldTop, fieldSide, fieldBottom, fieldIn1, fieldIn2 paintengine2d.Color
	disFace, disEdge, sep, headerLine, tabLine, grip     paintengine2d.Color
	menuText, accel, rowText, tipText, arrowDark         paintengine2d.Color

	grey, greyHot, greyPress, greyDis nbFace // buttons, tabs, combo fields
	blue, blueHot, bluePress          nbFace // default button, selected tab, combo arrow, spinner
	tick, tickHot, tickPress          nbFace // unselected check boxes and radios
	tickOn, tickOnHot, tickOnPress    nbFace // selected ones

	thumb, thumbHot, thumbPress []paintengine2d.GradientStop // across a scroll thumb
	thumbEdge                   []paintengine2d.GradientStop
	arrowFace, arrowHot         []paintengine2d.GradientStop
	arrowPress                  []paintengine2d.GradientStop
	barGrad, headGrad, hotGrad  []paintengine2d.GradientStop
	trackGrad, progTrack, prog  []paintengine2d.GradientStop
	stripe, stripeDis           []paintengine2d.GradientStop
	capGrad, capOffGrad         []paintengine2d.GradientStop
	closeHot, iconInfo, iconErr []paintengine2d.GradientStop
	iconWarn                    []paintengine2d.GradientStop
	closeHotFace                nbFace
	progEdge, white, card       paintengine2d.Color
	frameOn, frameOff           paintengine2d.Color
	frameEdgeOn, frameEdgeOff   paintengine2d.Color
	frameGapOn, frameGapOff     paintengine2d.Color
	expander, arrowSep, heading paintengine2d.Color
	focusOnSel, tipTop          paintengine2d.Color
	iconEdge                    [3]paintengine2d.Color
}

type nbKey struct{}

// nbColors is the look's resolved Nimbus set (built once per look).
func nbColors(l *Classic) *nb {
	return l.Memo(nbKey{}, func() any { return nbBuild(l) }).(*nb)
}

// nbDefaults are Nimbus's published colour keys (plus the table stripe)
// the engine reads; a pack overrides any of them through "extra".
var nbDefaults = map[string]string{
	"nimbusBase": "#33628c", "nimbusBlueGrey": "#a9b0be", "control": "#d6d9df",
	"nimbusFocus": "#73a4d1", "nimbusSelectionBackground": "#39698a", "nimbusSelectedText": "#ffffff",
	"text": "#000000", "nimbusDisabledText": "#8e8f91", "nimbusLightBackground": "#ffffff",
	"nimbusBorder": "#9297a1", "nimbusOrange": "#bf6204", "nimbusRed": "#a92e22",
	"nimbusInfoBlue": "#2f5cb4", "nimbusAlertYellow": "#ffdc23", "menu": "#edeff2",
	"alternateRow": "#f2f2f2", "scrollbar": "#cdd0d5",
	"activeCaption": "#babec6", "inactiveCaption": "#bdc1c8",
}

// nbStops builds gradient stops from alternating offsets and colours.
func nbStops(pairs ...any) []paintengine2d.GradientStop {
	out := make([]paintengine2d.GradientStop, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		at, _ := pairs[i].(float64)
		var col paintengine2d.Color
		switch v := pairs[i+1].(type) {
		case string:
			col = Hex(v)
		case paintengine2d.Color:
			col = v
		}
		out = append(out, Stop(float32(at), col))
	}
	return out
}

// nbTint moves every stop of g toward to by t.
func nbTint(g []paintengine2d.GradientStop, to paintengine2d.Color, t float32) []paintengine2d.GradientStop {
	out := make([]paintengine2d.GradientStop, len(g))
	for i, s := range g {
		out[i] = Stop(s.Offset, Mix(s.Color, to, t))
	}
	return out
}

func nbBuild(l *Classic) *nb {
	col := func(k string) paintengine2d.Color { return l.X(k, Hex(nbDefaults[k])) }
	white := Hex("#ffffff")
	c := &nb{
		u:        nbPx(l),
		stripes:  l.P("stripes", 1) != 0,
		base:     col("nimbusBase"),
		blueGrey: col("nimbusBlueGrey"),
		control:  col("control"),
		focus:    col("nimbusFocus"),
		sel:      col("nimbusSelectionBackground"),
		selText:  col("nimbusSelectedText"),
		text:     col("text"),
		dis:      col("nimbusDisabledText"),
		light:    col("nimbusLightBackground"),
		border:   col("nimbusBorder"),
		orange:   col("nimbusOrange"),
		red:      col("nimbusRed"),
		infoBlue: col("nimbusInfoBlue"),
		yellow:   col("nimbusAlertYellow"),
		menuBg:   col("menu"),
		alt:      col("alternateRow"),
		scroll:   col("scrollbar"),
		capOn:    col("activeCaption"),
		capOff:   col("inactiveCaption"),
	}
	c.halo = Mix(c.focus, c.control, 0.55)
	c.selText = ReadableOn(c.sel, 4.5, c.selText, white, c.text)
	c.rowText = ReadableOn(c.light, 4.5, Hex("#232324"), c.text)
	c.tipText = c.text
	c.menuText = c.text
	c.accel = ReadableOn(c.menuBg, 4.5, Mix(c.text, c.menuBg, 0.35), c.text)
	c.arrowDark = Mix(c.text, c.base, 0.25)

	// Text fields: dark along the top, lighter down the sides, lightest at
	// the bottom, with a two-line shadow inside the top edge.
	c.fieldTop, c.fieldSide, c.fieldBottom = Hex("#8b8c8f"), Hex("#a4a7ac"), Hex("#c0c0c1")
	c.fieldIn1, c.fieldIn2 = Hex("#cbcbcc"), Hex("#e5e5e5")
	c.disFace, c.disEdge = Mix(c.control, white, 0.45), Mix(c.border, c.control, 0.5)
	c.sep = Mix(c.border, c.control, 0.25)
	c.headerLine, c.tabLine = Hex("#747980"), Mix(c.base, Hex("#000000"), 0.55)
	c.grip = Mix(c.border, c.control, 0.1)

	// The grey material (buttons, tabs, combo fields), measured off Nimbus
	// screenshots: bright at the top, down to the canvas grey by 60%, up
	// again to a bluish white at the bottom, in a border that darkens
	// downward, over a one-pixel soft shadow.
	c.grey = nbFace{
		face:  nbStops(0.0, "#fbfbfc", 0.6, c.control, 0.7, c.control, 1.0, "#f5f7fd"),
		edge:  nbStops(0.0, "#959b9e", 0.5, "#7d8286", 1.0, "#55585e"),
		under: Hex("#b6b9be"),
	}
	c.greyHot = nbFace{face: nbTint(c.grey.face, white, 0.45), edge: nbTint(c.grey.edge, c.base, 0.12), under: c.grey.under}
	c.greyPress = nbFace{
		face:  nbStops(0.0, "#a4a6ac", 0.12, "#babdc3", 0.2, "#bec1c7", 1.0, "#bdc0c6"),
		edge:  nbStops(0.0, "#7d7f83", 1.0, "#a2a4a9"),
		under: Hex("#f1f2f4"),
	}
	c.greyDis = nbFace{
		face:  nbStops(0.0, Mix(Hex("#fbfbfc"), c.control, 0.3), 1.0, Mix(c.control, white, 0.35)),
		edge:  nbStops(0.0, c.disEdge, 1.0, c.disEdge),
		under: paintengine2d.Color{},
	}
	// The blue material (default buttons, selected tabs, selected check
	// boxes, the combo arrow and spinner): white-blue to blue-grey, a dark
	// navy border.
	c.blue = nbFace{
		face:  nbStops(0.0, "#f6f8fa", 0.35, "#b8c9d7", 0.55, "#a2b6c9", 1.0, "#bbd0e3"),
		edge:  nbStops(0.0, "#62778a", 1.0, "#253a4d"),
		under: Hex("#b8bcc3"),
	}
	c.blueHot = nbFace{face: nbTint(c.blue.face, white, 0.35), edge: c.blue.edge, under: c.blue.under}
	c.bluePress = nbFace{face: nbTint(c.blue.face, c.base, 0.3), edge: nbTint(c.blue.edge, Hex("#000000"), 0.2), under: Hex("#f1f2f4")}
	// Check boxes and radios: grey glass, the blue glass when selected.
	c.tick = nbFace{
		face:  nbStops(0.0, "#f9f9fa", 0.5, "#d9dce1", 0.6, c.control, 1.0, "#e1e4ea"),
		edge:  nbStops(0.0, "#94979d", 1.0, "#4a4d52"),
		under: Hex("#c2c6d0"),
	}
	c.tickHot = nbFace{face: nbTint(c.tick.face, white, 0.5), edge: c.tick.edge, under: c.tick.under}
	c.tickPress = nbFace{face: nbTint(c.tick.face, c.blueGrey, 0.45), edge: c.tick.edge, under: c.tick.under}
	c.tickOn = nbFace{
		face:  nbStops(0.0, "#f6f8fa", 0.3, "#d7e1e9", 0.6, "#a7bccf", 1.0, "#b7ccdf"),
		edge:  nbStops(0.0, "#677b8c", 1.0, "#273c4f"),
		under: Hex("#b9bec9"),
	}
	c.tickOnHot = nbFace{face: nbTint(c.tickOn.face, white, 0.3), edge: c.tickOn.edge, under: c.tickOn.under}
	c.tickOnPress = nbFace{face: nbTint(c.tickOn.face, c.base, 0.25), edge: c.tickOn.edge, under: c.tickOn.under}

	// The scroll thumb: lit from the left edge, darkest a third across,
	// light again at the right, inside a border dark on the unlit side.
	c.thumb = nbStops(0.0, "#f9fbfc", 0.33, "#abc0d3", 0.7, "#cce1f0", 1.0, "#e5f8fb")
	c.thumbHot = nbTint(c.thumb, white, 0.35)
	c.thumbPress = nbTint(c.thumb, c.base, 0.25)
	c.thumbEdge = nbStops(0.0, "#8499ac", 1.0, "#33485b")
	c.arrowFace = nbStops(0.0, "#fafafa", 0.5, "#dde0e3", 1.0, "#d2d5d9")
	c.arrowHot = nbTint(c.arrowFace, white, 0.6)
	c.arrowPress = nbTint(c.arrowFace, c.blueGrey, 0.5)

	c.barGrad = nbStops(0.0, "#fbfbfc", 0.7, c.control, 1.0, c.control)
	c.headGrad = nbStops(0.0, "#f1f2f5", 0.6, "#d8dbe0", 0.72, "#d7dae0", 1.0, "#edf0f6")
	c.hotGrad = nbStops(0.0, Mix(c.sel, white, 0.22), 0.5, c.sel, 1.0, Mix(c.sel, Hex("#000000"), 0.12))
	c.trackGrad = nbStops(0.0, Mix(c.blueGrey, Hex("#000000"), 0.1), 1.0, Mix(c.control, white, 0.4))
	c.progTrack = nbStops(0.0, Mix(c.control, white, 0.2), 1.0, Mix(c.control, white, 0.75))
	c.prog = nbStops(0.0, Mix(c.orange, white, 0.62), 0.45, Mix(c.orange, white, 0.18), 0.55, c.orange, 1.0, Mix(c.orange, white, 0.3))
	c.progEdge = Mix(c.orange, Hex("#000000"), 0.3)
	c.stripe = nbStops(0.0, Mix(c.orange, white, 0.42), 0.5, Mix(c.orange, white, 0.42), 0.5, c.orange, 1.0, c.orange)
	c.stripeDis = nbStops(0.0, Mix(c.control, white, 0.5), 0.5, Mix(c.control, white, 0.5), 0.5, c.control, 1.0, c.control)
	c.capGrad = nbStops(0.0, Mix(c.capOn, white, 0.7), 1.0, c.capOn)
	c.capOffGrad = nbStops(0.0, Mix(c.capOff, white, 0.8), 1.0, Mix(c.capOff, white, 0.35))
	c.closeHot = nbStops(0.0, Mix(c.red, white, 0.55), 0.55, c.red, 1.0, Mix(c.red, white, 0.2))
	c.iconInfo = nbStops(0.0, Mix(c.infoBlue, white, 0.55), 1.0, Mix(c.infoBlue, Hex("#000000"), 0.15))
	c.iconErr = nbStops(0.0, Mix(c.red, white, 0.5), 1.0, Mix(c.red, Hex("#000000"), 0.15))
	c.iconWarn = nbStops(0.0, Mix(c.yellow, white, 0.55), 1.0, Mix(c.yellow, c.orange, 0.35))
	black := Hex("#000000")
	c.closeHotFace = nbFace{face: c.closeHot, edge: nbStops(0.0, Mix(c.red, black, 0.2), 1.0, Mix(c.red, black, 0.45))}
	c.white = white
	c.card = Mix(c.control, c.light, 0.35)
	c.frameOn, c.frameOff = c.capOn, Mix(c.capOff, white, 0.35)
	c.frameEdgeOn, c.frameEdgeOff = Mix(c.frameOn, black, 0.35), Mix(c.frameOff, black, 0.35)
	c.frameGapOn, c.frameGapOff = Mix(c.frameEdgeOn, c.frameOn, 0.4), Mix(c.frameEdgeOff, c.frameOff, 0.4)
	c.expander = Mix(c.text, c.control, 0.3)
	c.arrowSep = Mix(c.blueGrey, c.light, 0.3)
	c.heading = Mix(c.text, c.base, 0.35)
	c.focusOnSel = Mix(c.focus, c.light, 0.35)
	c.tipTop = Mix(c.control, c.light, 0.75)
	c.iconEdge = [3]paintengine2d.Color{Mix(c.infoBlue, black, 0.35), Mix(c.yellow, black, 0.5), Mix(c.red, black, 0.4)}
	return c
}

// ---- drawing helpers ---------------------------------------------------------------

// nbPx is one Nimbus pixel: a device pixel at 1x, two at 2x.
func nbPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
}

// nbSnap puts b on whole device pixels.
func nbSnap(b paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(snap(b.Min.X), snap(b.Min.Y)),
		Max: paintengine2d.Pt(snap(b.Max.X), snap(b.Max.Y)),
	}
}

// body is the part of a control's rect Nimbus paints the control in: two
// pixels in on every side, room for the focus ring and the shadow.
func (c *nb) body(b paintengine2d.Rect) paintengine2d.Rect {
	return nbSnap(b).Inset(2 * c.u)
}

// nbRects batches rects of one colour into a single path, filled in one
// operation.
type nbRects struct {
	p *paintengine2d.Path
}

func (r *nbRects) add(x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if r.p == nil {
		r.p = paintengine2d.NewPath()
	}
	r.p.AddRect(paintengine2d.XYWH(x, y, w, h))
}

func (r *nbRects) fill(ctx *paintengine2d.Context, col paintengine2d.Color) {
	if r.p != nil && col.A > 0 {
		ctx.DrawPath(r.p, paintengine2d.Fill(col))
	}
}

// nbIf returns s when cond holds.
func nbIf(cond bool, s ControlState) ControlState {
	if cond {
		return s
	}
	return 0
}

// radius clamps r to what a body of that size can round.
func nbRadius(b paintengine2d.Rect, r float32) float32 {
	m := b.Dx()
	if b.Dy() < m {
		m = b.Dy()
	}
	if r > m*0.5 {
		r = m * 0.5
	}
	if r < 0 {
		r = 0
	}
	return r
}

// gloss paints a material over body with corner radius r: the line under
// it, the border, then the face one pixel in.
func (c *nb) gloss(ctx *paintengine2d.Context, body paintengine2d.Rect, r float32, m *nbFace) {
	u := c.u
	if body.Dx() < 3*u || body.Dy() < 3*u {
		return
	}
	r = nbRadius(body, r)
	if m.under.A > 0 {
		ctx.DrawRoundRect(body.Translate(paintengine2d.Pt(0, u)), r, r, paintengine2d.Fill(m.under))
	}
	ctx.DrawRoundRect(body, r, r, VGradient(body, m.edge...))
	in := body.Inset(u)
	ri := nbRadius(in, r-u)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, m.face...))
}

// glossCorners is gloss with per-corner radii (tabs, spinner halves).
func (c *nb) glossCorners(ctx *paintengine2d.Context, body paintengine2d.Rect, tl, tr, br, bl float32, m *nbFace, under bool) {
	u := c.u
	if body.Dx() < 3*u || body.Dy() < 3*u {
		return
	}
	if under && m.under.A > 0 {
		ctx.DrawPath(RoundRectPath(body.Translate(paintengine2d.Pt(0, u)), tl, tr, br, bl), paintengine2d.Fill(m.under))
	}
	ctx.DrawPath(RoundRectPath(body, tl, tr, br, bl), VGradient(body, m.edge...))
	in := body.Inset(u)
	sub := func(v float32) float32 { return max(v-u, 0) }
	ctx.DrawPath(RoundRectPath(in, sub(tl), sub(tr), sub(br), sub(bl)), VGradient(in, m.face...))
}

// ring is Nimbus's focus: nimbusFocus one pixel outside body and a pale
// halo outside that — both inside the control's rect.
func (c *nb) ring(ctx *paintengine2d.Context, body paintengine2d.Rect, r float32) {
	u := c.u
	r = nbRadius(body, r)
	ctx.DrawRoundRect(body.Inset(-1.5*u), r+1.5*u, r+1.5*u, paintengine2d.StrokePaint(c.halo, u))
	ctx.DrawRoundRect(body.Inset(-0.5*u), r+0.5*u, r+0.5*u, paintengine2d.StrokePaint(c.focus, u))
}

// arrow fills a small solid triangle pointing dir, centred on ctr, w wide.
func (c *nb) arrow(ctx *paintengine2d.Context, ctr paintengine2d.Point, w float32, dir Direction, col paintengine2d.Color) {
	if col.A <= 0 || w <= 0 {
		return
	}
	h := w * 0.55
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(ctr.X-w*0.5, ctr.Y+h*0.5)
		p.LineTo(ctr.X+w*0.5, ctr.Y+h*0.5)
		p.LineTo(ctr.X, ctr.Y-h*0.5)
	case DirDown:
		p.MoveTo(ctr.X-w*0.5, ctr.Y-h*0.5)
		p.LineTo(ctr.X+w*0.5, ctr.Y-h*0.5)
		p.LineTo(ctr.X, ctr.Y+h*0.5)
	case DirLeft:
		p.MoveTo(ctr.X+h*0.5, ctr.Y-w*0.5)
		p.LineTo(ctr.X+h*0.5, ctr.Y+w*0.5)
		p.LineTo(ctr.X-h*0.5, ctr.Y)
	default:
		p.MoveTo(ctr.X-h*0.5, ctr.Y-w*0.5)
		p.LineTo(ctr.X-h*0.5, ctr.Y+w*0.5)
		p.LineTo(ctr.X+h*0.5, ctr.Y)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// buttonFace picks the grey (or, for a default button, blue) material for
// a control state.
func (c *nb) buttonFace(st ControlState, blue bool) *nbFace {
	switch {
	case st.Disabled():
		return &c.greyDis
	case st.Pressed() || (st.Checked() && st.Toggle()):
		if blue {
			return &c.bluePress
		}
		return &c.greyPress
	case st.Hovered():
		if blue {
			return &c.blueHot
		}
		return &c.greyHot
	}
	if blue {
		return &c.blue
	}
	return &c.grey
}

// field paints a text field or view well in body: white with a border dark
// along the top and light at the bottom and a soft shadow inside the top.
func (c *nb) fieldWell(ctx *paintengine2d.Context, body paintengine2d.Rect, disabled bool) {
	u := c.u
	if body.Dx() < 3*u || body.Dy() < 3*u {
		return
	}
	if disabled {
		ctx.DrawRect(body, paintengine2d.Fill(c.disFace))
		ctx.DrawRect(body.Inset(0.5*u), paintengine2d.StrokePaint(c.disEdge, u))
		return
	}
	ctx.DrawRect(body, VGradient(body, Stop(0, c.fieldTop), Stop(0.5, c.fieldSide), Stop(1, c.fieldBottom)))
	in := body.Inset(u)
	ctx.DrawRect(in, paintengine2d.Fill(c.light))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.fieldIn1))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+u, in.Dx(), u), paintengine2d.Fill(c.fieldIn2))
}

// ---- parts ------------------------------------------------------------------------

func (e nimbusEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := nbColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	switch role {
	case RoleButton:
		c.gloss(ctx, c.body(b), l.rx(5), c.buttonFace(st, st.Primary()))
	case RoleTool:
		if st.Hovered() || st.Pressed() || st.Checked() {
			c.gloss(ctx, nbSnap(b).Inset(c.u), l.rx(4), c.buttonFace(st|nbIf(st.Checked(), StateToggle), false))
		}
	case RoleField, RoleCombo:
		c.fieldWell(ctx, c.body(b), st.Disabled())
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(nbSnap(b), paintengine2d.Fill(c.sel))
			return c.selText
		}
		return c.rowText
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
		return c.selText
	case RoleThumb:
		c.thumbPaint(ctx, b, b.Dy() >= b.Dx(), st)
	case RoleTrack:
		ctx.DrawRect(nbSnap(b), paintengine2d.Fill(c.scroll))
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.control))
	}
	return fg
}

// indicator picks a check box or radio material.
func (c *nb) indicator(st ControlState, on bool) *nbFace {
	switch {
	case st.Disabled():
		return &c.greyDis
	case on && st.Pressed():
		return &c.tickOnPress
	case on && st.Hovered():
		return &c.tickOnHot
	case on:
		return &c.tickOn
	case st.Pressed():
		return &c.tickPress
	case st.Hovered():
		return &c.tickHot
	}
	return &c.tick
}

// CheckIndicator is a 14px rounded glass box in the 18px icon — grey, or
// the blue glass with a black check when selected; focus rings the box.
func (nimbusEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := nbColors(l)
	u := c.u
	body := c.body(box)
	if body.Dx() < 6*u || body.Dy() < 6*u {
		return
	}
	on := checked || st.Checked()
	c.gloss(ctx, body, 2.5*u, c.indicator(st, on))
	if on {
		mark := c.text
		if st.Disabled() {
			mark = c.dis
		}
		s := body.Dx()
		x, y := body.Min.X, body.Min.Y
		p := paintengine2d.NewPath()
		p.MoveTo(x+s*0.25, y+s*0.46)
		p.LineTo(x+s*0.44, y+s*0.72)
		p.LineTo(x+s*0.78, y+s*0.2)
		ctx.DrawPath(p, paintengine2d.Paint{Color: mark, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 2.2 * u, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	}
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, 2.5*u)
	}
}

// RadioIndicator is the same glass as a 14px disc with a black dot.
func (nimbusEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := nbColors(l)
	u := c.u
	body := c.body(box)
	d := body.Dx()
	if body.Dy() < d {
		d = body.Dy()
	}
	if d < 6*u {
		return
	}
	body = paintengine2d.XYWH(body.Min.X, body.Min.Y+(body.Dy()-d)*0.5, d, d)
	on := selected || st.Checked()
	c.gloss(ctx, body, d*0.5, c.indicator(st, on))
	if on {
		dot := c.text
		if st.Disabled() {
			dot = c.dis
		}
		ctx.DrawCircle(body.Center(), d*0.21, paintengine2d.Fill(dot))
	}
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, d*0.5)
	}
}

// Arrow is a small solid triangle.
func (nimbusEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	c := nbColors(l)
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	w := s * 0.55
	if m := 9 * c.u; w > m {
		w = m
	}
	c.arrow(ctx, b.Center(), w, dir, col)
}

// Expander is the tree's solid triangle: right when collapsed, down when
// expanded.
func (nimbusEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := nbColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	c.arrow(ctx, b.Center(), 9*c.u, dir, col)
}

// MenuHighlight is the selected menu item: the selection blue with a
// little gloss; an open menu bar title rounds its top corners.
func (nimbusEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := nbColors(l)
	b = nbSnap(b)
	if b.Dx() < 3*c.u || b.Dy() < 3*c.u {
		return
	}
	if attachBottom {
		r := nbRadius(b, l.rx(4))
		ctx.DrawPath(RoundRectPath(b, r, r, 0, 0), VGradient(b, c.hotGrad...))
		return
	}
	ctx.DrawRect(b, VGradient(b, c.hotGrad...))
}

func (nimbusEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := nbColors(l)
	if hot {
		return c.selText
	}
	return c.menuText
}

// Focused fields, combos and views take the focus ring.
func (nimbusEngine) FieldFocusRing(l *Classic) bool { return true }

// DrawFocusRing rings the inside of b in nimbusFocus with its halo.
func (nimbusEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	c.ring(ctx, nbSnap(b).Inset(2*c.u), 0)
}

// ---- scroll bars --------------------------------------------------------------------

func (nimbusEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 15, Arrows: ArrowsEnds, ArrowLen: 17, MinThumb: 29}
}

// thumbPaint is the capsule scroll thumb: the lit glass across its short
// axis inside a border dark on the unlit side.
func (c *nb) thumbPaint(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	b = nbSnap(b)
	u := c.u
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	face := c.thumb
	switch {
	case st.Pressed():
		face = c.thumbPress
	case st.Hovered():
		face = c.thumbHot
	}
	r := nbRadius(b, min(b.Dx(), b.Dy())*0.5)
	grad := func(rr paintengine2d.Rect, stops []paintengine2d.GradientStop) paintengine2d.Paint {
		if vertical {
			return HGradient(rr, stops...)
		}
		return VGradient(rr, stops...)
	}
	ctx.DrawRoundRect(b, r, r, grad(b, c.thumbEdge))
	in := b.Inset(u)
	ri := nbRadius(in, r-u)
	ctx.DrawRoundRect(in, ri, ri, grad(in, face))
}

func (e nimbusEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := nbColors(l)
	u := c.u
	if p.Bar.Empty() {
		return
	}
	bar := nbSnap(p.Bar)
	ctx.DrawRect(bar, paintengine2d.Fill(c.scroll))
	// A hairline against the content along the bar's inner edge.
	if vertical {
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, u, bar.Dy()), paintengine2d.Fill(c.sep))
	} else {
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), u), paintengine2d.Fill(c.sep))
	}
	button := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		b = nbSnap(b)
		face := c.arrowFace
		switch {
		case st.Disabled:
		case st.Pressed == part:
			face = c.arrowPress
		case st.Hot == part:
			face = c.arrowHot
		}
		if vertical {
			ctx.DrawRect(b, HGradient(b, face...))
		} else {
			ctx.DrawRect(b, VGradient(b, face...))
		}
		// The button's end towards the track curves into it: a soft dark
		// arc across the bar.
		var arc paintengine2d.Rect
		w := b.Dx()
		if !vertical {
			w = b.Dy()
		}
		switch dir {
		case DirUp:
			arc = paintengine2d.XYWH(b.Min.X, b.Max.Y-w*0.35, b.Dx(), w*0.7)
		case DirDown:
			arc = paintengine2d.XYWH(b.Min.X, b.Min.Y-w*0.35, b.Dx(), w*0.7)
		case DirLeft:
			arc = paintengine2d.XYWH(b.Max.X-w*0.35, b.Min.Y, w*0.7, b.Dy())
		default:
			arc = paintengine2d.XYWH(b.Min.X-w*0.35, b.Min.Y, w*0.7, b.Dy())
		}
		ctx.Save()
		ctx.ClipRect(b)
		ctx.DrawOval(arc.Inset(0.5*u), paintengine2d.StrokePaint(c.border, u))
		ctx.Restore()
		col := c.arrowDark
		if st.Disabled {
			col = c.dis
		}
		ctr := b.Center()
		off := w * 0.12
		switch dir {
		case DirUp:
			ctr.Y -= off
		case DirDown:
			ctr.Y += off
		case DirLeft:
			ctr.X -= off
		default:
			ctr.X += off
		}
		c.arrow(ctx, ctr, 7*u, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	button(p.Dec, dec, ScrollDec)
	button(p.Inc, inc, ScrollInc)
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	t := nbSnap(p.Thumb)
	if vertical {
		t = paintengine2d.XYWH(t.Min.X+u, t.Min.Y, t.Dx()-u, t.Dy())
	} else {
		t = paintengine2d.XYWH(t.Min.X, t.Min.Y+u, t.Dx(), t.Dy()-u)
	}
	ts := StateNone
	switch {
	case st.Pressed == ScrollThumbPart:
		ts = StatePressed
	case st.Hot == ScrollThumbPart:
		ts = StateHovered
	}
	c.thumbPaint(ctx, t, vertical, ts)
}

// DrawScrollBar is the thumb-in-track fallback for callers without scroll
// bar geometry.
func (e nimbusEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := nbColors(l)
	ctx.DrawRect(nbSnap(track), paintengine2d.Fill(c.scroll))
	if !thumb.Empty() {
		c.thumbPaint(ctx, thumb, track.Dy() >= track.Dx(), st)
	}
}

// ---- frames -------------------------------------------------------------------------

func (nimbusEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a titled border: a rounded line in nimbusBorder with the
// title over it; a raised box is a lighter card inside the line.
func (nimbusEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	f := l.body
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	r := nbRadius(frame, l.rx(5))
	if raised {
		ctx.DrawRoundRect(frame, r, r, paintengine2d.Fill(c.card))
	}
	var tx, tw float32
	if title != "" {
		tx = b.Min.X + l.S(10)
		tw = f.Advance(title) + l.S(8)
		if tw > b.Dx()-l.S(20) {
			tw = b.Dx() - l.S(20)
		}
	}
	stroke := paintengine2d.StrokePaint(c.border, u)
	fr := frame.Inset(0.5 * u)
	if title == "" || tw <= 0 {
		ctx.DrawRoundRect(fr, r, r, stroke)
		return
	}
	gap := paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height())
	for _, clip := range [3]paintengine2d.Rect{
		paintengine2d.XYWH(b.Min.X, b.Min.Y, gap.Min.X-b.Min.X, b.Dy()),
		paintengine2d.XYWH(gap.Max.X, b.Min.Y, b.Max.X-gap.Max.X, b.Dy()),
		paintengine2d.XYWH(gap.Min.X, gap.Max.Y, gap.Dx(), b.Max.Y-gap.Max.Y),
	} {
		if clip.Empty() {
			continue
		}
		ctx.Save()
		ctx.ClipRect(clip)
		ctx.DrawRoundRect(fr, r, r, stroke)
		ctx.Restore()
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(4), b.Min.Y, tw-l.S(4), f.Height()), c.text, AlignStart, 0)
}

// nbCaptionH is an internal frame's title bar: the bold title between
// three-pixel margins, tall enough for its buttons.
func nbCaptionH(l *Classic) float32 {
	f := l.bold
	if f == nil {
		f = l.body
	}
	h := f.Height() + l.S(8)
	if m := l.S(24); h < m {
		h = m
	}
	return snap(h)
}

// nbFrameW is the frame round an internal frame's body (its published
// content margins).
func nbFrameW(l *Classic) float32 { return 6 * nbColors(l).u }

func (nimbusEngine) WindowFrameInsets(l *Classic) Insets {
	fw := nbFrameW(l)
	return Insets{Top: nbCaptionH(l), Right: fw, Bottom: fw, Left: fw}
}

// WindowCloseRect is the round-cornered close button at the right of the
// title bar.
func (nimbusEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = nbSnap(b)
	h := nbCaptionH(l)
	fw := nbFrameW(l)
	if b.Dx() < 2*h+2*fw || b.Dy() < h+2*fw {
		return paintengine2d.Rect{}
	}
	side := snap(h - l.S(8))
	return paintengine2d.XYWH(b.Max.X-fw-l.S(4)-side, b.Min.Y+snap((h-side)*0.5), side, side)
}

// PopupShadow: Nimbus floats menus, tool tips and frames on soft shadows.
func (nimbusEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	dy, blur := nbShadow(l, kind)
	return ShadowReach(0, dy, blur, 0)
}

func (nimbusEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	dy, blur := nbShadow(l, kind)
	a := float32(0.24)
	switch kind {
	case PopupTooltip:
		a = 0.18
	case PopupDialog:
		a = 0.3
	}
	DropShadow(ctx, b, l.rx(4), paintengine2d.RGBA(0, 0, 0, a), 0, dy, blur, 0)
}

func nbShadow(l *Classic, kind PopupKind) (dy, blur float32) {
	switch kind {
	case PopupTooltip:
		return l.S(1), l.S(4)
	case PopupDialog:
		return l.S(4), l.S(16)
	}
	return l.S(2), l.S(8)
}

// Nimbus's option panes put the default button last: "Cancel  OK"
// (OptionPane.isYesLast).
func (nimbusEngine) StyleHint(l *Classic, h StyleHint) int { return 0 }

// DrawWindowFrame is a Nimbus internal frame: a rounded-top frame in the
// caption's grey-blue, its title bar glossed from white, the bold title
// centred (dimmed when inactive) and a round-cornered close button that
// glows red under the pointer.
func (e nimbusEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	capH := nbCaptionH(l)
	fw := nbFrameW(l)
	if b.Dx() < 2*fw+capH || b.Dy() < capH+2*fw {
		ctx.DrawRect(b, paintengine2d.Fill(c.control))
		return
	}
	r := nbRadius(b, l.rx(6))
	grad, frame, edge, gap := c.capGrad, c.frameOn, c.frameEdgeOn, c.frameGapOn
	if !st.Active {
		grad, frame, edge, gap = c.capOffGrad, c.frameOff, c.frameEdgeOff, c.frameGapOff
	}
	ctx.DrawPath(RoundRectPath(b, r, r, 0, 0), paintengine2d.Fill(edge))
	in := b.Inset(u)
	ctx.DrawPath(RoundRectPath(in, max(r-u, 0), max(r-u, 0), 0, 0), paintengine2d.Fill(frame))
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), capH-u)
	ctx.Save()
	ctx.ClipPath(RoundRectPath(in, max(r-u, 0), max(r-u, 0), 0, 0))
	ctx.DrawRect(bar, VGradient(bar, grad...))
	ctx.Restore()
	body := paintengine2d.XYWH(b.Min.X+fw, b.Min.Y+capH, b.Dx()-2*fw, b.Dy()-capH-fw)
	ctx.DrawRect(body.Inset(-u), paintengine2d.Fill(gap))
	ctx.DrawRect(body, paintengine2d.Fill(c.control))
	right := bar.Max.X - l.S(4)
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			m := &c.grey
			switch {
			case st.ClosePress:
				m = &c.greyPress
			case st.CloseHot:
				m = &c.closeHotFace
			}
			c.gloss(ctx, cb, l.rx(3), m)
			glyph := c.arrowDark
			if st.CloseHot && !st.ClosePress {
				glyph = c.white
			}
			DrawCross(ctx, cb.Inset(snap(cb.Dx()*0.3)), glyph, 1.8*u)
			right = cb.Min.X - l.S(4)
		}
	}
	if title != "" {
		fg := c.text
		if !st.Active {
			fg = c.dis
		}
		f := l.bold
		if f == nil {
			f = l.body
		}
		// Centred on the bar, kept clear of the buttons on both sides.
		m := bar.Max.X - right
		if m < l.S(8) {
			m = l.S(8)
		}
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(bar.Min.X+m, bar.Min.Y, bar.Dx()-2*m, bar.Dy()), fg, AlignCenter, 0)
	}
}

// TabOutset: tabs keep their slots.
func (nimbusEngine) TabOutset(l *Classic) Insets { return Insets{} }

// TabOverlap: neighbouring tabs share one border line.
func (nimbusEngine) TabOverlap(l *Classic) float32 { return nbColors(l).u }

// SpinBoxStyle: the stepper sits inside the field's frame, stacked.
func (nimbusEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{Inside: true} }

// ToolBarInsets keeps the dotted handle clear of the first button.
func (nimbusEngine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(16), Right: l.S(4)} }

// ControlFont: Nimbus labels everything in its plain default font.
func (nimbusEngine) ControlFont(l *Classic, role Role) *Font { return l.body }

// ItemFocus rings the current row in nimbusFocus.
func (nimbusEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := nbColors(l)
	b = nbSnap(b)
	if b.Dx() < 3*c.u || b.Dy() < 3*c.u {
		return
	}
	col := c.focus
	if st.Checked() {
		col = c.focusOnSel
	}
	ctx.DrawRect(b.Inset(0.5*c.u), paintengine2d.StrokePaint(col, c.u))
}

// ViewFrameInsets: a scroll pane's border and the pixel inside it that
// takes the focus halo.
func (nimbusEngine) ViewFrameInsets(l *Classic) Insets {
	u := nbColors(l).u
	return Insets{Top: 2 * u, Left: 2 * u, Right: 2 * u, Bottom: 2 * u}
}

// DrawViewFrame is a scroll pane: white inside a nimbusBorder line; focused,
// the line turns nimbusFocus with the halo inside it.
func (nimbusEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	bg := c.light
	if st.Disabled() {
		bg = c.disFace
	}
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	edge := c.border
	if st.Focused() && !st.Disabled() {
		edge = c.focus
		ctx.DrawRect(b.Inset(1.5*u), paintengine2d.StrokePaint(c.halo, u))
	}
	ctx.DrawRect(b.Inset(0.5*u), paintengine2d.StrokePaint(edge, u))
}

// DrawTabPane is the page: the canvas under the selected tab's blue band,
// edged in the tab strip's dark line.
func (nimbusEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.control))
	band := 3 * u
	if b.Dy() < band+2*u {
		return
	}
	br := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), band)
	ctx.DrawRect(br, paintengine2d.Fill(c.blue.face[len(c.blue.face)-1].Color))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, br.Max.Y, b.Dx(), u), paintengine2d.Fill(c.tabLine))
}

// ---- controls -----------------------------------------------------------------

func (e nimbusEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := nbColors(l)
	body := c.body(b)
	r := l.rx(5)
	c.gloss(ctx, body, r, c.buttonFace(st, st.Primary()))
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, body, fg, AlignCenter, l.S(12))
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, r)
	}
}

// toggleLabel draws a check box, radio or switch caption.
func (c *nb) toggleLabel(l *Classic, ctx *paintengine2d.Context, lb paintengine2d.Rect, st ControlState, label string) {
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

func (e nimbusEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := nbColors(l)
	side := l.metrics.Checkbox
	box := nbSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	if label != "" {
		gap := l.S(4)
		c.toggleLabel(l, ctx, paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy()), st, label)
	}
}

func (e nimbusEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := nbColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := nbSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	if label != "" {
		gap := l.S(4)
		c.toggleLabel(l, ctx, paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy()), st, label)
	}
}

// DrawSwitch has no Nimbus original: a rounded field slot that fills with
// the blue glass when on, and a round glass knob.
func (e nimbusEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := nbColors(l)
	u := c.u
	tw, th := l.metrics.SwitchW, l.metrics.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := nbSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	body := track.Inset(2 * u)
	if body.Dx() < 8*u || body.Dy() < 6*u {
		return
	}
	r := body.Dy() * 0.5
	m := &c.grey
	switch {
	case st.Disabled():
		m = &c.greyDis
	case on:
		m = &c.blue
	}
	if on || st.Disabled() {
		c.gloss(ctx, body, r, m)
	} else {
		ctx.DrawRoundRect(body, r, r, VGradient(body, Stop(0, c.fieldTop), Stop(1, c.fieldBottom)))
		in := body.Inset(u)
		ctx.DrawRoundRect(in, r-u, r-u, VGradient(in, Stop(0, c.fieldIn1), Stop(0.3, c.light)))
	}
	kd := body.Dy()
	kx := body.Min.X
	if on {
		kx = body.Max.X - kd
	}
	knob := paintengine2d.XYWH(kx, body.Min.Y, kd, kd)
	c.gloss(ctx, knob, kd*0.5, c.buttonFace(st&^(StatePrimary|StateChecked|StateToggle), false))
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, r)
	}
	if label != "" {
		gap := l.S(6)
		c.toggleLabel(l, ctx, paintengine2d.XYWH(track.Max.X+gap, b.Min.Y, b.Max.X-track.Max.X-gap, b.Dy()), st, label)
	}
}

// DrawComboBox is a non-editable combo: a rounded grey glass field with the
// blue glass arrow segment at its right, split by a pale line.
func (e nimbusEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := nbColors(l)
	u := c.u
	body := c.body(b)
	if body.Dx() < 12*u || body.Dy() < 8*u {
		return
	}
	r := nbRadius(body, l.rx(4))
	down := (open || st.Pressed()) && !st.Disabled()
	grey := c.buttonFace(st&^StatePrimary|nbIf(down, StatePressed), false)
	c.gloss(ctx, body, r, grey)
	aw := snap(19 * u)
	if aw > body.Dx()*0.5 {
		aw = snap(body.Dx() * 0.5)
	}
	ab := paintengine2d.XYWH(body.Max.X-aw, body.Min.Y, aw, body.Dy())
	if !st.Disabled() {
		blue := &c.blue
		switch {
		case down:
			blue = &c.bluePress
		case st.Hovered():
			blue = &c.blueHot
		}
		ctx.Save()
		ctx.ClipRect(ab)
		c.gloss(ctx, body, r, blue)
		ctx.Restore()
		ctx.DrawRect(paintengine2d.XYWH(ab.Min.X, ab.Min.Y+u, u, ab.Dy()-2*u), paintengine2d.Fill(c.arrowSep))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	c.arrow(ctx, paintengine2d.Pt(ab.Center().X+u*0.5, ab.Center().Y), 8*u, DirDown, fg)
	tb := paintengine2d.XYWH(body.Min.X+l.metrics.FieldPad, body.Min.Y, ab.Min.X-body.Min.X-l.metrics.FieldPad-2*u, body.Dy())
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() && !open {
		c.ring(ctx, body, r)
	}
}

// DrawTextField: the square inset well with the focus ring round it.
func (e nimbusEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := nbColors(l)
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	body := c.body(b)
	c.fieldWell(ctx, body, st.Disabled())
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, 0)
	}
	l.baseDrawTextField(ctx, b, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

// DrawSpinner is the stepper: two blue glass buttons stacked, rounded at
// their outer corners, inside the field's frame or on their own.
func (e nimbusEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := nbColors(l)
	u := c.u
	box := nbSnap(b)
	if !st.Frameless() {
		box = c.body(b)
	}
	if box.Dx() < 6*u || box.Dy() < 8*u {
		return
	}
	r := nbRadius(box, l.rx(4))
	mid := snap((box.Min.Y + box.Max.Y) * 0.5)
	half := func(hb paintengine2d.Rect, top bool, dir Direction, hover, press bool) {
		m := &c.blue
		switch {
		case st.Disabled():
			m = &c.greyDis
		case press:
			m = &c.bluePress
		case hover:
			m = &c.blueHot
		}
		tr, br := float32(0), float32(0)
		if top {
			tr = r
		} else {
			br = r
		}
		c.glossCorners(ctx, hb, 0, tr, br, 0, m, !top)
		col := c.text
		if st.Disabled() {
			col = c.dis
		}
		c.arrow(ctx, hb.Center(), 7*u, dir, col)
	}
	half(paintengine2d.XYWH(box.Min.X, box.Min.Y, box.Dx(), mid-box.Min.Y+u), true, DirUp, upHover, upPress)
	half(paintengine2d.XYWH(box.Min.X, mid, box.Dx(), box.Max.Y-mid), false, DirDown, downHover, downPress)
}

// DrawTabBar is the strip: the canvas with the dark line the tabs stand on.
func (nimbusEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	b = nbSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.control))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-c.u, b.Dx(), c.u), paintengine2d.Fill(c.tabLine))
}

// DrawTab is a Nimbus tab: rounded top corners, grey glass, the blue glass
// when selected (opening into the page's blue band), focus ringed inside.
func (e nimbusEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	t := paintengine2d.XYWH(b.Min.X, b.Min.Y+2*u, b.Dx(), b.Dy()-3*u)
	if selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+2*u, b.Dx(), b.Dy()-2*u)
	}
	if t.Dx() < 8*u || t.Dy() < 8*u {
		return
	}
	r := nbRadius(t, l.rx(5))
	var m *nbFace
	switch {
	case st.Disabled():
		m = &c.greyDis
	case selected && (st.Pressed() || st.Hovered()):
		m = &c.blueHot
	case selected:
		m = &c.blue
	case st.Pressed():
		m = &c.greyPress
	case st.Hovered():
		m = &c.greyHot
	default:
		m = &c.grey
	}
	ctx.Save()
	ctx.ClipRect(t)
	shape := paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx(), t.Dy()+u)
	c.glossCorners(ctx, shape, r, r, 0, 0, m, false)
	ctx.Restore()
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(t.Min.X, t.Min.Y+u, t.Dx(), t.Dy()-u), fg, AlignCenter, l.S(10))
	if st.Focused() && selected && !st.Disabled() {
		fr := t.Inset(1.5 * u)
		fr.Max.Y = t.Max.Y - u
		ctx.Save()
		ctx.ClipRect(t)
		ctx.DrawPath(RoundRectPath(fr, r-u, r-u, 0, 0), paintengine2d.StrokePaint(c.focus, u))
		ctx.Restore()
	}
}

// DrawPanel: the canvas; raised is a lighter card in a hairline.
func (nimbusEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := nbColors(l)
	b = nbSnap(b)
	if !raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.control))
		return
	}
	r := nbRadius(b, l.rx(5))
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.card))
	ctx.DrawRoundRect(b.Inset(0.5*c.u), r, r, paintengine2d.StrokePaint(c.border, c.u))
}

// DrawMenuBar is the white-to-canvas wash over a nimbusBorder line.
func (nimbusEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	b = nbSnap(b)
	ctx.DrawRect(b, VGradient(b, c.barGrad...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-c.u, b.Dx(), c.u), paintengine2d.Fill(c.border))
}

// DrawMenuTitle: an open or pointed-at title is the rounded selection blue
// with white text.
func (e nimbusEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	fg := c.text
	hl := paintengine2d.XYWH(b.Min.X+u, b.Min.Y+2*u, b.Dx()-2*u, b.Dy()-3*u)
	switch {
	case st.Disabled():
		fg = c.dis
	case open || st.Pressed() || st.Hovered():
		r := nbRadius(hl, l.rx(4))
		ctx.DrawRoundRect(hl, r, r, VGradient(hl, c.hotGrad...))
		fg = c.selText
	case st.Focused():
		r := nbRadius(hl, l.rx(4))
		ctx.DrawRoundRect(hl.Inset(0.5*u), r, r, paintengine2d.StrokePaint(c.focus, u))
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame is a popup: the menu colour in a nimbusBorder line.
func (nimbusEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	b = nbSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.menuBg))
	ctx.DrawRect(b.Inset(0.5*c.u), paintengine2d.StrokePaint(c.border, c.u))
}

// DrawMenuItem is a Nimbus menu row: the selection blue across the popup
// with white text, a plain tick or dot for check and radio items, a thin
// separator.
func (e nimbusEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := nbColors(l)
	u := c.u
	ch := MenuChromeFor(l)
	left := snap(b.Min.X - ch.PadL + u)
	right := snap(b.Max.X + ch.PadR - u)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		if x0 := snap(ch.LabelMinX(b.Min.X)) - l.S(4); right > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, right-x0-l.S(4), u), paintengine2d.Fill(c.sep))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot && right > left {
		e.MenuHighlight(l, ctx, paintengine2d.XYWH(left, snap(b.Min.Y), right-left, snap(b.Max.Y)-snap(b.Min.Y)), false)
	}
	fg := c.menuText
	switch {
	case st.Disabled():
		fg = c.dis
	case hot:
		fg = c.selText
	}
	gw := ch.CheckCol()
	ib := paintengine2d.XYWH(b.Min.X, b.Min.Y, gw, b.Dy())
	switch {
	case row.Radio && row.Checked:
		ctx.DrawCircle(ib.Center(), 3.5*u, paintengine2d.Fill(fg))
	case row.Checked:
		s := snap(l.S(10))
		x, y := ib.Center().X-s*0.5, ib.Center().Y-s*0.5
		p := paintengine2d.NewPath()
		p.MoveTo(x+s*0.1, y+s*0.52)
		p.LineTo(x+s*0.4, y+s*0.82)
		p.LineTo(x+s*0.92, y+s*0.18)
		ctx.DrawPath(p, paintengine2d.Paint{Color: fg, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 2 * u, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	case row.Icon != IconNone && !row.Radio:
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		c.arrow(ctx, ab.Center(), 8*u, DirRight, fg)
		labelRight = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := f.InkWidth(row.Shortcut)
		if tw <= 0 {
			tw = f.Advance(row.Shortcut)
		}
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		acc := c.accel
		switch {
		case st.Disabled():
			acc = c.dis
		case hot:
			acc = c.selText
		}
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), acc)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()))
	l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// DrawProgressBar is a rounded well filled with glossy nimbusOrange;
// indeterminate, the fill runs the whole well in moving diagonal stripes.
func (nimbusEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	if b.Dx() < 8*u || b.Dy() < 6*u {
		return
	}
	r := nbRadius(b, l.rx(4))
	edge := c.border
	if st.Disabled() {
		edge = c.disEdge
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(edge))
	in := b.Inset(u)
	ri := nbRadius(in, r-u)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.progTrack...))
	fill := in.Inset(u)
	if fill.Dx() < 2*u || fill.Dy() < 2*u {
		return
	}
	rf := nbRadius(fill, ri-u)
	edgeCol, face := c.progEdge, c.prog
	if st.Disabled() {
		edgeCol, face = c.disEdge, c.greyDis.face
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		period := snap(27 * u)
		off := fill.Min.X + phase*period
		stripe := c.stripe
		if st.Disabled() {
			stripe = c.stripeDis
		}
		paint := paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(off, fill.Min.Y),
			End:   paintengine2d.Pt(off+period*0.5, fill.Min.Y-period*0.5),
			Stops: stripe,
			Tile:  paintengine2d.TileRepeat,
		})
		ctx.DrawRoundRect(fill, rf, rf, paintengine2d.Fill(edgeCol))
		stripes := fill.Inset(u)
		ctx.DrawRoundRect(stripes, max(rf-u, 0), max(rf-u, 0), paint)
		ctx.DrawRoundRect(stripes, max(rf-u, 0), max(rf-u, 0), VGradient(stripes, Stop(0, paintengine2d.RGBA(1, 1, 1, 0.35)), Stop(0.5, paintengine2d.RGBA(1, 1, 1, 0.05)), Stop(1, paintengine2d.RGBA(1, 1, 1, 0.15))))
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	w := snap(fill.Dx() * t)
	if w < 2*u {
		return
	}
	fr := paintengine2d.XYWH(fill.Min.X, fill.Min.Y, w, fill.Dy())
	rr := nbRadius(fr, rf)
	paint := VGradient(fr, face...)
	ctx.DrawRoundRect(fr, rr, rr, paintengine2d.Fill(edgeCol))
	fi := fr.Inset(u)
	if fi.Dx() > 0 && fi.Dy() > 0 {
		ctx.DrawRoundRect(fi, max(rr-u, 0), max(rr-u, 0), paint)
	}
}

// DrawSlider is a thin rounded groove and the round grey glass knob;
// focus rings the knob.
func (e nimbusEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	kd := snap(17 * u)
	if kd > b.Dy()-4*u {
		kd = snap(b.Dy() - 4*u)
	}
	if kd < 8*u || b.Dx() < kd+4*u {
		return
	}
	cy := snap((b.Min.Y+b.Max.Y)*0.5) - kd*0.5
	x0 := b.Min.X + 2*u
	span := b.Dx() - 4*u - kd
	kx := snap(x0 + span*t)
	groove := paintengine2d.XYWH(b.Min.X+2*u, snap(cy+kd*0.5-3*u), b.Dx()-4*u, 6*u)
	gr := groove.Dy() * 0.5
	edge := c.border
	if st.Disabled() {
		edge = c.disEdge
	}
	ctx.DrawRoundRect(groove, gr, gr, paintengine2d.Fill(edge))
	gi := groove.Inset(u)
	ctx.DrawRoundRect(gi, gr-u, gr-u, VGradient(gi, c.trackGrad...))
	knob := paintengine2d.XYWH(kx, cy, kd, kd)
	c.gloss(ctx, knob, kd*0.5, c.buttonFace(st&^(StatePrimary|StateChecked|StateToggle), false))
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, knob, kd*0.5)
	}
}

// DrawListRow: the selection blue with white text across the row.
func (e nimbusEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := nbColors(l)
	fg := c.rowText
	if st.Checked() {
		ctx.DrawRect(nbSnap(b), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(5), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a JTree row: solid triangles, no lines, the selection
// across the row.
func (e nimbusEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := nbColors(l)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	fg := c.rowText
	if st.Checked() {
		ctx.DrawRect(nbSnap(b), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.dis
	}
	x := b.Min.X + pad + float32(depth)*indent
	if !leaf {
		col := c.expander
		if st.Checked() {
			col = fg
		}
		if st.ExpanderHot() {
			col = fg
			if !st.Checked() {
				col = c.base
			}
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, col)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x+indent+l.S(4), b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableHeader is the glossy header cell with a dark line under it and
// a short divider at its right.
func (e nimbusEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	grad := c.headGrad
	switch {
	case st.Pressed() && !st.Disabled():
		grad = c.greyPress.face
	case st.Hovered() && !st.Disabled():
		grad = c.greyHot.face
	}
	ctx.DrawRect(b, VGradient(b, grad...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.headerLine))
	if b.Dy() > 8*u {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+3*u, u, b.Dy()-6*u), paintengine2d.Fill(c.border))
	}
	fg := c.text
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
		c.arrow(ctx, paintengine2d.Pt(b.Max.X-l.S(6)-aw*0.5, (b.Min.Y+b.Max.Y)*0.5), 7*u, dir, c.arrowDark)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(5), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()-u), fg, AlignStart, 0)
}

// DrawTableCell: white and alternateRow stripes (odd rows), the selection
// blue with white text, no grid.
func (e nimbusEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := nbColors(l)
	sb := nbSnap(b)
	fg := c.rowText
	switch {
	case st.Checked():
		ctx.DrawRect(sb, paintengine2d.Fill(c.sel))
		fg = c.selText
	case st.Alternate() && c.stripes:
		ctx.DrawRect(sb, paintengine2d.Fill(c.alt))
	}
	if st.Disabled() {
		fg = c.dis
	}
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
	ctx.ClipRect(sb)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawToolBar is the canvas with the dotted handle at the left.
func (nimbusEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.control))
	if b.Dx() < 16*u || b.Dy() < 10*u {
		return
	}
	// The handle: two columns of embossed dots.
	var dk, lt nbRects
	x0 := b.Min.X + 4*u
	for y := b.Min.Y + 5*u; y+2*u <= b.Max.Y-5*u; y += 4 * u {
		for k := float32(0); k < 2; k++ {
			x := x0 + k*4*u
			lt.add(x+u, y+u, 2*u, 2*u)
			dk.add(x, y, 2*u, 2*u)
		}
	}
	lt.fill(ctx, c.light)
	dk.fill(ctx, c.grip)
}

func (e nimbusEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := nbColors(l)
	fg := c.text
	if !st.Disabled() && (st.Hovered() || st.Pressed() || st.Checked()) {
		c.gloss(ctx, nbSnap(b).Inset(c.u), l.rx(4), c.buttonFace(st|nbIf(st.Checked(), StateToggle), false))
	}
	if st.Disabled() {
		fg = c.dis
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		if icon == IconNone {
			l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, l.S(4))
		} else {
			right := b.Max.X - pad
			if right < x {
				right = x
			}
			ctx.Save()
			ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
			l.body.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-l.body.Height())*0.5), fg)
			ctx.Restore()
		}
	}
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, nbSnap(b).Inset(2*c.u), l.rx(4))
	}
}

// DrawStatusBar has no Nimbus original: the bar wash under a nimbusBorder
// line, panes split by short dividers.
func (nimbusEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	ctx.DrawRect(b, VGradient(b, c.barGrad...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.border))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(snap(x), b.Min.Y+l.S(5), u, b.Dy()-l.S(9)), paintengine2d.Fill(c.sep))
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y+u, slot-l.S(12), b.Dy()-u), c.text, AlignStart, 0)
	}
}

// DrawTitleBar is a panel heading: the bar wash, a nimbusBorder line under
// it, the title bold.
func (nimbusEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	ctx.DrawRect(b, VGradient(b, c.headGrad...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.border))
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.heading, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.dis, AlignStart, 0)
	}
}

// DrawAccordionHeader has no Nimbus original: a grey glass header with a
// triangle and the title.
func (e nimbusEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := nbColors(l)
	body := c.body(b)
	r := l.rx(4)
	c.gloss(ctx, body, r, c.buttonFace(st&^(StatePrimary|StateChecked), false))
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	e.Expander(l, ctx, paintengine2d.XYWH(body.Min.X+l.S(4), body.Min.Y, l.S(14), body.Dy()), expanded, c.expander)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(body.Min.X+l.S(22), body.Min.Y, body.Dx()-l.S(26), body.Dy()), fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.ring(ctx, body, r)
	}
}

// DrawSeparator is a hairline with a white line under it.
func (nimbusEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := nbColors(l)
	u := c.u
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		y0, h := snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4))
		ctx.DrawRect(paintengine2d.XYWH(x, y0, u, h), paintengine2d.Fill(c.sep))
		ctx.DrawRect(paintengine2d.XYWH(x+u, y0, u, h), paintengine2d.Fill(c.light))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), paintengine2d.Fill(c.sep))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+u, b.Dx(), u), paintengine2d.Fill(c.light))
}

// DrawSplitter is the canvas with a short column of embossed dots at its
// middle, blue under the pointer or while dragged.
func (nimbusEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := nbColors(l)
	b = nbSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.control))
	dot := c.grip
	if (st.Hovered() || st.Pressed() || st.Focused()) && !st.Disabled() {
		dot = c.focus
	}
	ctr := b.Center()
	var dk, lt nbRects
	for k := -2; k <= 2; k++ {
		o := float32(k) * 4 * u
		x, y := snap(ctr.X-u), snap(ctr.Y-u)
		if vertical {
			y += o
		} else {
			x += o
		}
		lt.add(x+u, y+u, 2*u, 2*u)
		dk.add(x, y, 2*u, 2*u)
	}
	ctx.Save()
	ctx.ClipRect(b)
	lt.fill(ctx, c.light)
	dk.fill(ctx, dot)
	ctx.Restore()
}

// DrawTooltip is a pale rounded box in a nimbusBorder line.
func (nimbusEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := nbColors(l)
	b = nbSnap(b)
	r := nbRadius(b, l.rx(4))
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.border))
	in := b.Inset(c.u)
	ctx.DrawRoundRect(in, max(r-c.u, 0), max(r-c.u, 0), VGradient(in, Stop(0, c.tipTop), Stop(1, c.control)))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(5)
	}
	if text == "" {
		return
	}
	// The bubble is sized to the text: draw it unfitted, clipped.
	ctx.Save()
	ctx.ClipRect(b)
	l.body.Draw(ctx, text, paintengine2d.Pt(b.Min.X+pad, b.Min.Y+(b.Dy()-l.body.Height())*0.5), c.tipText)
	ctx.Restore()
}

// DrawMessageIcon: glossy symbols — a nimbusInfoBlue disc with "i" (or
// "?"), a nimbusAlertYellow triangle with "!", a nimbusRed disc with ×.
func (nimbusEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := nbColors(l)
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	if s < 8 || icon == IconNone {
		return
	}
	ctr := b.Center()
	r := s * 0.45
	u := s / 32
	disc := paintengine2d.XYWH(ctr.X-r, ctr.Y-r, 2*r, 2*r)
	white := c.white
	f := l.bold
	if f == nil {
		f = l.body
	}
	glyph := func(g string, col paintengine2d.Color, dy float32) {
		w := f.Advance(g)
		f.Draw(ctx, g, paintengine2d.Pt(ctr.X-w*0.5, ctr.Y-f.Height()*0.5+dy), col)
	}
	shine := func(shape *paintengine2d.Path, rr paintengine2d.Rect) {
		ctx.Save()
		ctx.ClipPath(shape)
		gl := paintengine2d.XYWH(rr.Min.X, rr.Min.Y, rr.Dx(), rr.Dy()*0.5)
		ctx.DrawRect(gl, VGradient(gl, Stop(0, paintengine2d.RGBA(1, 1, 1, 0.5)), Stop(1, paintengine2d.RGBA(1, 1, 1, 0.08))))
		ctx.Restore()
	}
	switch icon {
	case IconInfo, IconQuestion:
		p := paintengine2d.NewPath()
		p.AddCircle(ctr, r)
		ctx.DrawPath(p, VGradient(disc, c.iconInfo...))
		shine(p, disc)
		ctx.DrawPath(p, nbRoundStroke(c.iconEdge[0], 1.2*u))
		g := "i"
		if icon == IconQuestion {
			g = "?"
		}
		glyph(g, white, 0)
	case IconWarning:
		p := paintengine2d.NewPath()
		p.MoveTo(ctr.X, ctr.Y-r)
		p.LineTo(ctr.X+r*1.05, ctr.Y+r*0.85)
		p.LineTo(ctr.X-r*1.05, ctr.Y+r*0.85)
		p.Close()
		ctx.DrawPath(p, VGradient(disc, c.iconWarn...))
		shine(p, disc)
		ctx.DrawPath(p, nbRoundStroke(c.iconEdge[1], 1.2*u))
		glyph("!", c.text, r*0.2)
	case IconError:
		p := paintengine2d.NewPath()
		p.AddCircle(ctr, r)
		ctx.DrawPath(p, VGradient(disc, c.iconErr...))
		shine(p, disc)
		ctx.DrawPath(p, nbRoundStroke(c.iconEdge[2], 1.2*u))
		DrawCross(ctx, paintengine2d.XYWH(ctr.X-r*0.4, ctr.Y-r*0.4, r*0.8, r*0.8), white, 3.5*u)
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// nbRoundStroke is an outline stroke with round joins, so sharp corners
// (a warning triangle's) stay close to the shape.
func nbRoundStroke(col paintengine2d.Color, w float32) paintengine2d.Paint {
	return paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
}

// ---- pack ---------------------------------------------------------------------------

func nimbusPacks() []ThemePack {
	x := nbDefaults
	h := func(k string) paintengine2d.Color { return hexColor(x[k]) }
	ctl := h("control")
	pal := Palette{
		Background: ctl, Surface: ctl, SurfaceAlt: ctl,
		Border: h("nimbusBorder"), Divider: Mix(h("nimbusBorder"), ctl, 0.25),
		Text: h("text"), TextMuted: h("nimbusDisabledText"), TextOnAccent: h("nimbusSelectedText"),
		Accent: h("nimbusBase"), AccentHover: Mix(h("nimbusBase"), hexColor("#ffffff"), 0.2), AccentPress: Shade(h("nimbusBase"), -0.2),
		Field: h("nimbusLightBackground"), FieldBorder: hexColor("#8b8c8f"),
		Focus: h("nimbusFocus"), Selection: h("nimbusSelectionBackground"),
		Track: h("scrollbar"), Thumb: h("nimbusBlueGrey"),
		Highlight: paintengine2d.RGBA(1, 1, 1, 0.6), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: h("nimbusSelectionBackground"), MenuHoverBorder: h("nimbusSelectionBackground"), MenuGutter: h("menu"),
		Danger: h("nimbusRed"), Success: hexColor("#3f7f1f"), Warning: hexColor("#a85600"),
		Overlay:    paintengine2d.RGBA(0.1, 0.12, 0.16, 0.35),
		BevelLight: hexColor("#ffffff"), BevelDark: h("nimbusBorder"),
	}
	tok := ThemeTokens{
		Engine:  "nimbus",
		Bevel:   BevelSoftShadow,
		Family:  ThemeLight,
		Era:     "Nimbus",
		Palette: pal,
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: hexColor("#bec1c7"), Border: hexColor("#7d7f83")}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.2), Border: pal.Focus}
	return []ThemePack{{
		Name: "nimbus", Label: "Nimbus", Year: 2008, Lineage: "Java",
		Summary: "Java 6's vector look: glossy rounded controls, blue focus glow, orange progress.",
		Era:     "Nimbus", Palette: ThemeLight, Tokens: tok,
	}}
}
