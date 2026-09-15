package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// fusionEngine paints in the manner of Qt's Fusion style (Qt 5.0, 2012), the
// cross-platform look every Qt application falls back to. It is written
// from Fusion's visible behaviour and its palette relations (as numbers),
// checked against screenshots. Every colour follows from a Qt-style
// palette; the key shades, as percentages of HSV value (lighterPct /
// darkerPct):
//
//	outline         window darker 140% — every 1px frame
//	focus outline   highlight darker 125%, HSL lightness capped at 160/255
//	                — keyboard focus and the default button
//	button tone     button lighter 100 + (180 − gray)/6 %, saturation ×0.75
//	tab frame       button tone lighter 104% — panes and the selected tab
//	contrast line   white at 30/255 — the light line inside frames
//
// Push buttons are 2px-rounded vertical gradients (base lighter 124% →
// base lighter 102%) in the outline with the inner contrast line; the
// default button is tinted towards the highlight and outlined in it, and
// focus is the highlighted outline, never a dotted rectangle. Check boxes
// are square gradient wells with Fusion's stroked tick; radios are circles
// with a dot. Scroll bars are 14px gradient grooves with a square gradient
// slider and an arrow button at each end; sliders are a 7px groove filled
// with the highlight up to a 2px-rounded gradient handle. Tabs have rounded
// top corners, unselected ones sit 2px lower and darker, the selected one
// is lighter and opens into the pane. Progress bars fill with a highlight
// gradient; the busy bar adds Fusion's moving diagonal stripes. Group boxes
// put the title above a faint rounded frame; tree branches are small
// triangles.
//
// Pack data (theme.json "extra", all optional — derived from the palette):
//
//	window, button, base, highlight, highlightText, disabledText
//	tip, tipText, tipBorder     tooltip (Qt's #ffffdc / black)
//
// The shared palette carries Qt's palette roles: Background = Window,
// SurfaceAlt = Button, Field = Base, Selection = Highlight,
// TextOnAccent = HighlightedText.
type fusionEngine struct{ BaseEngine }

func init() {
	RegisterEngine(fusionEngine{})
	for _, p := range fusionPacks() {
		RegisterPack(p)
	}
}

func (fusionEngine) ID() string { return "fusion" }

// DefaultMetrics are Fusion's proportions at the toolkit's 16px UI font (Qt
// used 9–10pt): 14px indicators and scroll bars, 15px slider handles, 2px
// corners.
func (fusionEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Radius: 2, RadiusSmall: 2,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 14, Radio: 14,
		MenuItemH: 26, MenuBarH: 26, TabH: 30, RowH: 24,
		TitleBar: 26, HeaderH: 26, ProgressH: 20, SliderH: 24, Thumb: 15,
		Scroll: 14, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 16, SwitchW: 40, SwitchH: 20,
	}
}

// Dialogs: Qt on Linux uses its KDE button-box layout, where the accepting
// button comes before the rejecting one: "OK  Cancel" (Windows' layout
// agrees). Form labels are left-aligned and tabs start at the left.
func (fusionEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// ---- colour arithmetic ---------------------------------------------------------------
//
// Fusion's shades are percentages of HSV value, as Qt documents for
// QColor::lighter / darker: lighterPct(c, 150) is 50% more value, and
// value pushed past full drains saturation instead, so light colours whiten.
// Breeze and Oxygen use these too.

// hsvOf splits c into hue (degrees), saturation and value (0..1).
func hsvOf(c paintengine2d.Color) (h, s, v float32) {
	r, g, b := clamp1(c.R), clamp1(c.G), clamp1(c.B)
	hi := max(r, g, b)
	lo := min(r, g, b)
	v = hi
	span := hi - lo
	if hi <= 0 || span <= 0 {
		return 0, 0, v
	}
	s = span / hi
	var sector float32
	switch hi {
	case r:
		sector = (g - b) / span
	case g:
		sector = 2 + (b-r)/span
	default:
		sector = 4 + (r-g)/span
	}
	h = sector * 60
	if h < 0 {
		h += 360
	}
	return h, s, v
}

// hsvColor builds a colour from hue (degrees), saturation, value and alpha.
func hsvColor(h, s, v, a float32) paintengine2d.Color {
	if s <= 0 {
		return paintengine2d.RGBA(v, v, v, a)
	}
	// Each channel is v less a share of the chroma that depends on how far
	// the hue is from the channel's own primary.
	chroma := v * s
	ch := func(n float32) float32 {
		k := float32(math.Mod(float64(n+h/60), 6))
		d := min(k, 4-k, 1)
		if d < 0 {
			d = 0
		}
		return v - chroma*d
	}
	return paintengine2d.RGBA(ch(5), ch(3), ch(1), a)
}

// lighterPct returns c with f% of its HSV value (f > 100 lightens).
func lighterPct(c paintengine2d.Color, f float32) paintengine2d.Color {
	h, s, v := hsvOf(c)
	v *= f / 100
	if over := v - 1; over > 0 {
		s = max(0, s-over)
		v = 1
	}
	return hsvColor(h, s, v, c.A)
}

// darkerPct returns c with its HSV value divided by f/100.
func darkerPct(c paintengine2d.Color, f float32) paintengine2d.Color {
	h, s, v := hsvOf(c)
	return hsvColor(h, s, v*100/f, c.A)
}

// mixPct is pct% of a and the rest of b, keeping a's alpha.
func mixPct(a, b paintengine2d.Color, pct float32) paintengine2d.Color {
	c := Mix(b, a, pct/100)
	c.A = a.A
	return c
}

// withLightness keeps c's hue and HSL saturation at HSL lightness l (0..1).
func withLightness(c paintengine2d.Color, l float32) paintengine2d.Color {
	r, g, b := clamp1(c.R), clamp1(c.G), clamp1(c.B)
	hi, lo := max(r, g, b), min(r, g, b)
	h, _, _ := hsvOf(c)
	var sat float32
	if span := hi - lo; span > 0 {
		if mid := (hi + lo) / 2; mid < 0.5 {
			sat = span / (hi + lo)
		} else {
			sat = span / (2 - hi - lo)
		}
	}
	// HSL back to HSV: same hue, value and saturation from l and sat.
	v := l + sat*min(l, 1-l)
	var sv float32
	if v > 0 {
		sv = 2 * (1 - l/v)
	}
	return hsvColor(h, sv, v, c.A)
}

// grayLevel is Qt's documented qGray weighting, (11r + 16g + 5b) / 32, on
// 0..255.
func grayLevel(c paintengine2d.Color) float32 {
	return (clamp1(c.R)*11 + clamp1(c.G)*16 + clamp1(c.B)*5) / 32 * 255
}

// ---- resolved colours ------------------------------------------------------------------

type fusion struct {
	win, text, base, btn, hl, hlText, dis paintengine2d.Color
	tip, tipText, tipBorder, disHl        paintengine2d.Color
	light                                 paintengine2d.Color

	outline, outlineLt, hlOutline, btnColor, tabFrame paintengine2d.Color
	contrast, topShadow, lightShade, darkShade        paintengine2d.Color

	// Push buttons: the gradient, hot gradient, down fill and the default
	// button's tinted versions.
	btnStops, btnHotStops, defStops, defHotStops []paintengine2d.GradientStop
	btnDown, defDown, disOutline                 paintengine2d.Color

	// Check boxes and radios.
	chkStops                    []paintengine2d.GradientStop
	chkDown, chkPen, radioPen   paintengine2d.Color
	mark, markDot, markDotStrok paintengine2d.Color

	arrow paintengine2d.Color // windowText at 160/255

	// Scroll bars.
	sbGroove, sbSlider, sbSliderHot []paintengine2d.GradientStop
	sbSliderDown, sbLine, sbEdge    paintengine2d.Color

	// Sliders and progress.
	groove, grooveHl         []paintengine2d.GradientStop
	grooveHlLine, handleDrop paintengine2d.Color
	handleStops              []paintengine2d.GradientStop
	progStops, progDisStops  []paintengine2d.GradientStop
	progLine, stripe         paintengine2d.Color
	focusStops               []paintengine2d.GradientStop
	focusLine                paintengine2d.Color

	// Tabs and headers.
	tabSel, tabStops []paintengine2d.GradientStop
	headerStops      []paintengine2d.GradientStop
	headerDown       paintengine2d.Color

	// Menus, bars, frames.
	menuBg, menuBorder, menuLight, menuShadow, barLine paintengine2d.Color
	toolStops                                          []paintengine2d.GradientStop
	groupFill, groupLine, sepDark, sepLight            paintengine2d.Color

	// MDI title bar (in-app windows): active and inactive.
	title, titleOff                      []paintengine2d.GradientStop
	titleLine, titleLineOff, titleHi     paintengine2d.Color
	titleHiOff, titleText, titleTextOff  paintengine2d.Color
	titleShadowText, mdiLine, mdiLineOff paintengine2d.Color
	frameLine                            paintengine2d.Color
}

type fusionKey struct{}

func fusionColors(l *Classic) *fusion {
	return l.Memo(fusionKey{}, func() any { return fusionBuild(l) }).(*fusion)
}

func fusionBuild(l *Classic) *fusion {
	p := l.palette
	c := &fusion{
		win:    l.X("window", p.Background),
		text:   p.Text,
		base:   l.X("base", p.Field),
		btn:    l.X("button", p.SurfaceAlt),
		hl:     l.X("highlight", p.Selection),
		hlText: l.X("highlightText", p.TextOnAccent),
		dis:    l.X("disabledText", Mix(p.Background, p.Text, 0.25)),
	}
	if c.hl.A < 0.9 {
		c.hl = p.Accent
	}
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c.disHl = l.X("disabledHighlight", Hex("#919191"))
	c.tip = l.X("tip", Hex("#ffffdc"))
	c.tipText = l.X("tipText", black)
	c.tipBorder = l.X("tipBorder", c.tipText)
	c.light = lighterPct(c.win, 150)

	// The key shades.
	c.outline = darkerPct(c.win, 140)
	c.outlineLt = lighterPct(c.outline, 110)
	c.hlOutline = darkerPct(c.hl, 125)
	if _, _, v := hsvOf(c.hlOutline); v > 160.0/255 {
		c.hlOutline = withLightness(c.hlOutline, 160.0/255)
	}
	gray := grayLevel(c.btn)
	step := (180 - gray) / 6
	if step < 1 {
		step = 1
	}
	bc := lighterPct(c.btn, 100+float32(int(step)))
	h, s, v := hsvOf(bc)
	c.btnColor = hsvColor(h, s*0.75, v, 1)
	c.tabFrame = lighterPct(c.btnColor, 104)
	c.contrast = white.WithAlpha(30.0 / 255)
	c.topShadow = black.WithAlpha(18.0 / 255)
	c.lightShade = white.WithAlpha(90.0 / 255)
	c.darkShade = black.WithAlpha(60.0 / 255)

	// Button gradient: base lighter 124% → base lighter 102%.
	grad := func(base paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, lighterPct(base, 124)), Stop(1, lighterPct(base, 102))}
	}
	c.btnStops = grad(darkerPct(c.btnColor, 104))
	c.btnHotStops = grad(c.btnColor)
	c.btnDown = darkerPct(c.btnColor, 110)
	def := mixPct(c.btnColor, lighterPct(c.hlOutline, 130), 90)
	c.defStops = grad(darkerPct(def, 104))
	c.defHotStops = grad(def)
	c.defDown = darkerPct(def, 110)
	c.disOutline = lighterPct(c.outline, 115)

	c.chkDown = mixPct(c.base, c.text, 85)
	c.chkStops = []paintengine2d.GradientStop{Stop(0, darkerPct(c.base, 115)), Stop(0.15, c.base), Stop(1, c.base)}
	c.chkPen = c.outlineLt
	c.radioPen = darkerPct(c.win, 150)
	c.mark = darkerPct(c.text, 120)
	c.markDot = c.mark.WithAlpha(180.0 / 255)
	c.markDotStrok = c.mark.WithAlpha(200.0 / 255)
	c.arrow = c.text.WithAlpha(160.0 / 255)

	// Scroll bars.
	c.sbGroove = []paintengine2d.GradientStop{
		Stop(0, darkerPct(c.btnColor, 107)), Stop(0.1, darkerPct(c.btnColor, 105)),
		Stop(0.9, darkerPct(c.btnColor, 105)), Stop(1, darkerPct(c.btnColor, 107)),
	}
	start, stop := lighterPct(c.btnColor, 118), c.btnColor
	c.sbSlider = []paintengine2d.GradientStop{Stop(0, lighterPct(c.btnColor, 108)), Stop(1, c.btnColor)}
	c.sbSliderHot = []paintengine2d.GradientStop{Stop(0, darkerPct(start, 102)), Stop(1, lighterPct(stop, 102))}
	c.sbSliderDown = mixPct(start, stop, 40)
	c.sbLine = c.outline.WithAlpha(180.0 / 255)
	c.sbEdge = c.outline.WithAlpha(40.0 / 255)

	// Sliders.
	gh, gs, gv := hsvOf(c.btnColor)
	grooveColor := hsvColor(gh, gs, gv*0.9, 1)
	c.groove = []paintengine2d.GradientStop{Stop(0, darkerPct(grooveColor, 110)), Stop(1, lighterPct(grooveColor, 110))}
	c.grooveHl = []paintengine2d.GradientStop{Stop(0, c.hl), Stop(1, lighterPct(c.hl, 130))}
	c.grooveHlLine = c.outline
	if hlo := darkerPct(c.hl, 140); grayLevel(c.outline) > grayLevel(hlo) {
		c.grooveHlLine = hlo
	}
	c.handleDrop = black.WithAlpha(40.0 / 255)
	c.handleStops = grad(c.btnColor)

	// Progress bars.
	c.progStops = []paintengine2d.GradientStop{Stop(0, lighterPct(c.hl, 120)), Stop(1, c.hl)}
	c.progLine = c.outline
	if hlo := darkerPct(c.hl, 140); grayLevel(c.outline) > grayLevel(hlo) {
		c.progLine = hlo
	}
	c.stripe = lighterPct(c.hl, 120).WithAlpha(120.0 / 255)
	pd := darkerPct(c.win, 115)
	c.progDisStops = []paintengine2d.GradientStop{Stop(0, lighterPct(pd, 110)), Stop(1, pd)}
	// Focus box: the focus outline at 30/255, lighter at the top, in its
	// 80/255 edge darkened by 120%.
	ff := c.hlOutline.WithAlpha(30.0 / 255)
	c.focusStops = []paintengine2d.GradientStop{Stop(0, lighterPct(ff, 160)), Stop(1, ff)}
	c.focusLine = darkerPct(c.hlOutline.WithAlpha(80.0/255), 120)

	// Tabs.
	c.tabSel = []paintengine2d.GradientStop{Stop(0, lighterPct(c.tabFrame, 104)), Stop(1, c.tabFrame)}
	c.tabStops = []paintengine2d.GradientStop{
		Stop(0, darkerPct(c.tabFrame, 108)), Stop(0.85, darkerPct(c.tabFrame, 108)), Stop(1, darkerPct(c.tabFrame, 116)),
	}

	// Header sections.
	hs, he := lighterPct(c.btnColor, 104), darkerPct(c.btnColor, 102)
	c.headerStops = []paintengine2d.GradientStop{
		Stop(0, hs), Stop(0.5, mixPct(hs, he, 60)), Stop(0.501, mixPct(hs, he, 40)),
		Stop(0.92, he), Stop(1, darkerPct(he, 104)),
	}
	c.headerDown = darkerPct(c.btnColor, 110)

	// Menus, menu bar, tool bar.
	c.menuBg = lighterPct(c.base, 108)
	c.menuBorder = darkerPct(c.win, 160)
	c.menuLight = lighterPct(c.win, 160)
	c.menuShadow = darkerPct(c.win, 110)
	c.barLine = mixPct(darkerPct(c.win, 120), lighterPct(c.outline, 140), 60)
	c.toolStops = []paintengine2d.GradientStop{Stop(0, lighterPct(c.win, 104)), Stop(1, c.win)}
	// Group boxes: a translucent black frame over the window.
	c.groupFill = Mix(c.win, black, 0.03)
	c.groupLine = Mix(c.win, black, 0.10)
	c.sepDark = darkerPct(c.win, 110)
	c.sepLight = lighterPct(c.win, 110)

	// In-app window title bars.
	tb := func(base paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{
			Stop(0, lighterPct(base, 114)), Stop(0.5, lighterPct(base, 102)),
			Stop(0.51, darkerPct(base, 104)), Stop(1, base),
		}
	}
	c.title, c.titleOff = tb(c.hl), tb(c.win)
	c.titleLine, c.titleLineOff = darkerPct(c.hl, 180), darkerPct(c.outline, 110)
	c.titleHi, c.titleHiOff = lighterPct(c.hl, 120), lighterPct(c.win, 120)
	c.titleText = ReadableOn(c.hl, 3, white, c.hlText)
	c.titleTextOff = c.text
	c.titleShadowText = lighterPct(c.text, 120)
	dh, ds, dv := hsvOf(c.btn)
	c.mdiLine = darkerPct(c.hl, 180)
	c.mdiLineOff = darkerPct(hsvColor(dh, ds, dv*0.7, 1), 110)
	c.frameLine = darkerPct(c.outline, 150)
	return c
}

// ---- painting helpers ------------------------------------------------------------------

// fuU is one design pixel at the look's scale, never under a device pixel.
func fuU(l *Classic) float32 {
	u := snap(l.S(1))
	if u < 1 {
		u = 1
	}
	return u
}

// fuR is the Fusion corner radius (2px), 0 when the Corners pref is square.
func fuR(l *Classic, v float32) float32 { return l.rx(v) }

// fuStroke strokes a crisp 1u rounded rectangle just inside b.
func fuStroke(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32, col paintengine2d.Color) {
	if b.Dx() < u*2 || b.Dy() < u*2 {
		return
	}
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.StrokePaint(col, u))
}

// fuLine fills a horizontal or vertical 1u line.
func fuHLine(ctx *paintengine2d.Context, x0, x1, y, u float32, col paintengine2d.Color) {
	if x1 > x0 {
		ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, u), paintengine2d.Fill(col))
	}
}

func fuVLine(ctx *paintengine2d.Context, x, y0, y1, u float32, col paintengine2d.Color) {
	if y1 > y0 {
		ctx.DrawRect(paintengine2d.XYWH(x, y0, u, y1-y0), paintengine2d.Fill(col))
	}
}

// fuSnap rounds a rect to the pixel grid.
func fuSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

// button paints a push button face into b: the 2px-rounded gradient in its
// outline with the inner contrast line.
func (c *fusion) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, def bool) {
	u := fuU(l)
	r := fuR(l, 2)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	enabled := !st.Disabled()
	down := enabled && (st.Pressed() || (st.Toggle() && st.Checked()))
	fill := VGradient(b, c.btnStops...)
	switch {
	case def && down:
		fill = paintengine2d.Fill(c.defDown)
	case down:
		fill = paintengine2d.Fill(c.btnDown)
	case def && st.Hovered():
		fill = VGradient(b, c.defHotStops...)
	case def:
		fill = VGradient(b, c.defStops...)
	case enabled && st.Hovered():
		fill = VGradient(b, c.btnHotStops...)
	}
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, fill)
	line := c.outline
	switch {
	case !enabled:
		line = c.disOutline
	case def || st.Focused():
		line = c.hlOutline
	}
	fuStroke(ctx, b, r, u, line)
	fuStroke(ctx, b.Inset(u), r, u, c.contrast)
}

// focusRect is Fusion's focus box: a translucent highlight box with a
// darker translucent edge (tool buttons, check / radio labels, rows).
func (c *fusion) focusRect(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	u := fuU(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	r := fuR(l, 1)
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, VGradient(b, c.focusStops...))
	fuStroke(ctx, b, r, u, c.focusLine)
}

// frame is a text field: the base fill in the outline (the focus outline
// with a soft second ring when focused) and a 1px inner top shadow.
func (c *fusion) frame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	u := fuU(l)
	r := fuR(l, 2)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	fill := c.base
	if st.Disabled() {
		fill = c.win
	}
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.Fill(fill))
	line := c.outline
	focus := st.Focused() && !st.Disabled()
	if focus {
		line = c.hlOutline
	}
	fuStroke(ctx, b, r, u, line)
	if focus {
		fuStroke(ctx, b.Inset(u), fuR(l, 1.7), u, c.hlOutline.WithAlpha(40.0/255))
	}
	fuHLine(ctx, b.Min.X+2*u, b.Max.X-2*u, b.Min.Y+u, u, c.topShadow)
}

// fuArrow fills Fusion's small solid triangle centred in b: 14×8
// proportions, at most 8px across (scaled).
func fuArrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if b.Dx() <= 1 || b.Dy() <= 1 || col.A <= 0 {
		return
	}
	size := l.S(8)
	if m := b.Dx(); m < size {
		size = m
	}
	if m := b.Dy(); m < size {
		size = m
	}
	w, h := size, size*8/14
	if dir == DirLeft || dir == DirRight {
		w, h = h, w
	}
	x := b.Min.X + (b.Dx()-w)*0.5
	y := b.Min.Y + (b.Dy()-h)*0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirDown:
		p.MoveTo(x, y)
		p.LineTo(x+w, y)
		p.LineTo(x+w*0.5, y+h)
	case DirRight:
		p.MoveTo(x, y)
		p.LineTo(x, y+h)
		p.LineTo(x+w, y+h*0.5)
	case DirLeft:
		p.MoveTo(x+w, y)
		p.LineTo(x+w, y+h)
		p.LineTo(x, y+h*0.5)
	default:
		p.MoveTo(x, y+h)
		p.LineTo(x+w, y+h)
		p.LineTo(x+w*0.5, y)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// ---- parts ---------------------------------------------------------------------------------

func (e fusionEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := fusionColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if b.Empty() {
		return fg
	}
	switch role {
	case RoleButton, RoleCombo:
		c.button(l, ctx, b, st, st.Primary() && !st.Disabled())
	case RoleTool:
		// Auto-raise: the panel shows when hot, held or latched.
		if st.Disabled() && !(st.Toggle() && st.Checked()) {
			return fg
		}
		if st.Hovered() || st.Pressed() || (st.Toggle() && st.Checked()) {
			c.button(l, ctx, b, st&^StateFocused, false)
		}
	case RoleField:
		c.frame(l, ctx, b, st)
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, st.Checked())
	case RoleRow:
		return c.row(l, ctx, b, st)
	case RoleMenu:
		if st.Hovered() || st.Pressed() || st.Checked() {
			e.MenuHighlight(l, ctx, b, false)
			return c.hlText
		}
	case RoleThumb:
		c.slider(l, ctx, b, b.Dy() >= b.Dx(), st.Hovered(), st.Pressed())
	case RoleTrack:
		c.sbTrack(l, ctx, b, b.Dy() >= b.Dx())
	case RoleBar, RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.win))
	}
	return fg
}

// CheckIndicator is a square well with a base gradient (darker at the very
// top), the lightened outline (the focus outline with keyboard focus) and
// Fusion's two-segment tick.
func (fusionEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := fusionColors(l)
	u := fuU(l)
	b := fuSnap(box)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	in := b.Inset(u * 0.5)
	if st.Pressed() && !st.Disabled() {
		ctx.DrawRect(in, paintengine2d.Fill(c.chkDown))
	} else if st.Disabled() {
		ctx.DrawRect(in, paintengine2d.Fill(c.win))
	} else {
		ctx.DrawRect(in, VGradient(b, c.chkStops...))
	}
	pen := c.chkPen
	if st.Focused() && !st.Disabled() {
		pen = c.hlOutline
	}
	ctx.DrawRect(in, paintengine2d.StrokePaint(pen, u))
	if !checked {
		return
	}
	mark := c.mark
	if st.Disabled() {
		mark = c.dis
	}
	// The tick, measured from Fusion screenshots of a 14px box: a short arm
	// from (26%, 49%) down to (43%, 77%), the long arm up to (74%, 22%).
	side := b.Dx()
	pw := max(l.S(1.5), 0.11*side)
	at := func(fx, fy float32) paintengine2d.Point { return paintengine2d.Pt(b.Min.X+fx*side, b.Min.Y+fy*side) }
	ctx.Save()
	ctx.ClipRect(b)
	strokeTick(ctx, at(0.26, 0.49), at(0.43, 0.77), at(0.74, 0.22), pw, mark, paintengine2d.JoinBevel)
	ctx.Restore()
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

// RadioIndicator is a base circle in the window darker 150%, a
// translucent text-coloured dot when selected.
func (fusionEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := fusionColors(l)
	u := fuU(l)
	side := box.Dx()
	if box.Dy() < side {
		side = box.Dy()
	}
	if side < 4*u {
		return
	}
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	r := side*0.5 - u*0.5
	fill := c.base
	switch {
	case st.Disabled():
		fill = c.win
	case st.Pressed():
		fill = c.chkDown
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(fill))
	pen := c.radioPen
	if st.Focused() && !st.Disabled() {
		pen = c.hlOutline
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.StrokePaint(pen, u))
	if selected {
		dot, ring := c.markDot, c.markDotStrok
		if st.Disabled() {
			dot, ring = c.dis, c.dis
		}
		dr := r / 2.32
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), dr, paintengine2d.Fill(dot))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), dr, paintengine2d.StrokePaint(ring, u))
	}
}

// Arrow is Fusion's small solid triangle.
func (fusionEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	fuArrow(l, ctx, b, dir, col)
}

// Expander is the tree branch mark: the down / right arrow at 160/255.
func (fusionEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := fusionColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	fuArrow(l, ctx, b, dir, c.arrow)
}

// MenuHighlight is the selected menu item (and the open menu-bar title):
// the highlight in its darker outline, square.
func (fusionEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := fusionColors(l)
	u := fuU(l)
	if b.Dx() < 2*u || b.Dy() < 2*u {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.hl))
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.hlOutline, u))
}

func (fusionEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := fusionColors(l)
	if hot {
		return c.hlText
	}
	return c.text
}

// Fields paint their own focus (the highlighted outline) in Face.
func (fusionEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the focus box inside b.
func (fusionEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	fusionColors(l).focusRect(l, ctx, fuSnap(b))
}

// ItemFocus is the focus box over the current item: the translucent
// highlight box in its 1px darker edge (on a selection the box is
// invisible and the edge marks the row).
func (fusionEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if st.Disabled() {
		return
	}
	fusionColors(l).focusRect(l, ctx, fuSnap(b))
}

// ViewFrameInsets: item views sit in a 1px outline.
func (fusionEngine) ViewFrameInsets(l *Classic) Insets {
	u := fuU(l)
	return Insets{Top: u, Right: u, Bottom: u, Left: u}
}

// DrawViewFrame is a scroll area's panel: the base colour in a square,
// lightened outline (no focus ring: the current item carries the focus
// mark).
func (fusionEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	fill := c.base
	if st.Disabled() {
		fill = c.win
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	if b.Dx() > 2*u && b.Dy() > 2*u {
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(lighterPct(c.outline, 108), u))
	}
}

// ---- scroll bars ---------------------------------------------------------------------------------

// ScrollBarStyle: 14px bars, a button at each end, a 26px minimum slider.
func (fusionEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 14, Arrows: ArrowsEnds, MinThumb: 26}
}

// sbTrack is the scroll bar groove: a gradient across the bar with a
// translucent outline along its leading edge and a faint inner edge.
func (c *fusion) sbTrack(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	u := fuU(l)
	if b.Empty() {
		return
	}
	if vertical {
		ctx.DrawRect(b, HGradient(b, c.sbGroove...))
		fuVLine(ctx, b.Min.X, b.Min.Y, b.Max.Y, u, c.sbLine)
		fuVLine(ctx, b.Min.X+u, b.Min.Y, b.Max.Y, u, c.sbEdge)
		fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y, u, c.sbEdge)
		return
	}
	ctx.DrawRect(b, VGradient(b, c.sbGroove...))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y, u, c.sbLine)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y+u, u, c.sbEdge)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.sbEdge)
}

// slider is the scroll bar slider: a square gradient box in the
// translucent outline with the inner contrast line.
func (c *fusion) slider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical, hot, down bool) {
	u := fuU(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	across := func(stops []paintengine2d.GradientStop) paintengine2d.Paint {
		if vertical {
			return HGradient(b, stops...)
		}
		return VGradient(b, stops...)
	}
	fill := across(c.sbSlider)
	switch {
	case down:
		fill = paintengine2d.Fill(c.sbSliderDown)
	case hot:
		fill = across(c.sbSliderHot)
	}
	ctx.DrawRect(b, fill)
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.sbLine, u))
	ctx.DrawRect(b.Inset(u*1.5), paintengine2d.StrokePaint(c.contrast, u))
}

func (e fusionEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := fusionColors(l)
	u := fuU(l)
	if p.Bar.Empty() {
		return
	}
	c.sbTrack(l, ctx, fuSnap(p.Bar), vertical)
	button := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		b = fuSnap(b)
		across := func(stops []paintengine2d.GradientStop) paintengine2d.Paint {
			if vertical {
				return HGradient(b, stops...)
			}
			return VGradient(b, stops...)
		}
		fill := across(c.sbSlider)
		switch {
		case st.Disabled:
		case st.Pressed == part:
			fill = paintengine2d.Fill(c.btnColor)
		case st.Hot == part:
			fill = across(c.sbSliderHot)
		}
		ctx.DrawRect(b, fill)
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.sbLine, u))
		ctx.DrawRect(b.Inset(u*1.5), paintengine2d.StrokePaint(c.contrast, u))
		col := c.arrow
		if st.Disabled {
			col = c.dis
		}
		fuArrow(l, ctx, b.Inset(u*2), dir, col)
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
	c.slider(l, ctx, fuSnap(p.Thumb), vertical,
		st.Hot == ScrollThumbPart || (st.Hovered && st.Pressed == ScrollNone && st.Hot == ScrollThumbPart),
		st.Pressed == ScrollThumbPart)
}

// DrawScrollBar paints a bare track + thumb (widgets that lay out their own).
func (e fusionEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	vertical := track.Dy() >= track.Dx()
	ss := ScrollState{Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	if st.Hovered() {
		ss.Hot, ss.Hovered = ScrollThumbPart, true
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, vertical, ss)
}

// ---- frames -------------------------------------------------------------------------------------------

func (fusionEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top += l.body.Height() + l.S(3)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox: the title at the top left over a faint rounded frame a
// shade darker than the window.
func (fusionEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	top := b.Min.Y
	f := l.body
	if title != "" {
		th := f.Height() + l.S(3)
		fg := c.text
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+u, b.Min.Y, b.Dx()-2*u, f.Height()), fg, AlignStart, 0)
		top = snap(b.Min.Y + th)
	}
	fr := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if fr.Dx() < 4*u || fr.Dy() < 4*u {
		return
	}
	r := fuR(l, 2)
	fill := c.groupFill
	if raised {
		fill = c.tabFrame
	}
	ctx.DrawRoundRect(fr.Inset(u*0.5), r, r, paintengine2d.Fill(fill))
	fuStroke(ctx, fr, r, u, c.groupLine)
}

// fuTitleH is the in-app window title bar height at the UI font.
func fuTitleH(l *Classic) float32 {
	h := snap(l.body.Height() + l.S(8))
	if h < snap(l.S(22)) {
		h = snap(l.S(22))
	}
	return h
}

func (fusionEngine) WindowFrameInsets(l *Classic) Insets {
	u := fuU(l)
	return Insets{Top: fuTitleH(l), Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// WindowCloseRect is the MDI close button at the right of the title bar.
func (fusionEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	h := fuTitleH(l)
	if b.Dy() < h || b.Dx() < h*2 {
		return paintengine2d.Rect{}
	}
	side := snap(h - l.S(7))
	return paintengine2d.XYWH(snap(b.Max.X-side-l.S(5)), snap(b.Min.Y+(h-side)*0.5), side, side)
}

// DrawWindowFrame is an in-app (MDI-style) window: the title bar gradient
// in the highlight (window colour when inactive) with chamfered top
// corners, a centred title and the close button, over a window body in a
// dark border.
func (e fusionEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	h := fuTitleH(l)
	if b.Dx() < 8*u || b.Dy() < h {
		return
	}
	// Body.
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y+h, b.Dx(), b.Dy()-h)
	ctx.DrawRect(body, paintengine2d.Fill(c.win))
	ctx.DrawRect(body.Inset(u*0.5), paintengine2d.StrokePaint(c.frameLine, u))
	fuVLine(ctx, body.Min.X+u, body.Min.Y, body.Max.Y-u, u, c.light)
	fuHLine(ctx, body.Min.X+u, body.Max.X-u, body.Max.Y-2*u, u, c.menuShadow)
	fuVLine(ctx, body.Max.X-2*u, body.Min.Y, body.Max.Y-u, u, c.menuShadow)
	// Title bar.
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), h)
	stops, line, hi := c.title, c.titleLine, c.titleHi
	if !st.Active {
		stops, line, hi = c.titleOff, c.titleLineOff, c.titleHiOff
	}
	ch := l.S(4)
	path := RoundRectPath(bar, ch, ch, 0, 0)
	ctx.DrawPath(path, VGradient(bar, stops...))
	ctx.Save()
	ctx.ClipRect(bar)
	ctx.DrawRoundRectCorners(bar.Inset(u*0.5), ch, ch, 0, 0, paintengine2d.StrokePaint(line, u))
	ctx.Restore()
	fuHLine(ctx, bar.Min.X+6*u, bar.Max.X-6*u, bar.Min.Y+u, u, hi)
	right := bar.Max.X - l.S(6)
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			c.closeBox(l, ctx, cb, st)
			right = cb.Min.X - l.S(4)
		}
	}
	if title == "" {
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	tr := paintengine2d.XYWH(bar.Min.X+l.S(8), bar.Min.Y, right-bar.Min.X-l.S(8), bar.Dy())
	if st.Active {
		l.drawFittedText(ctx, f, title, tr.Translate(paintengine2d.Pt(u, u)), c.titleShadowText.WithAlpha(0.45), AlignCenter, 0)
		l.drawFittedText(ctx, f, title, tr, c.titleText, AlignCenter, 0)
		return
	}
	l.drawFittedText(ctx, f, title, tr, c.titleTextOff, AlignCenter, 0)
}

// closeBox is the title bar's close button: a small bevelled box with a cross.
func (c *fusion) closeBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st WindowState) {
	u := fuU(l)
	switch {
	case st.ClosePress:
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(darkerPct(c.hl, 120)))
	case st.CloseHot:
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 20.0/255)))
	}
	border := c.mdiLine
	if !st.Active {
		border = c.mdiLineOff
	}
	r := fuR(l, 2)
	fuStroke(ctx, b, r, u, border)
	hi := paintengine2d.RGBA(1, 1, 1, 60.0/255)
	if st.ClosePress {
		hi = darkerPct(c.hl, 130)
	}
	fuHLine(ctx, b.Min.X+2*u, b.Max.X-2*u, b.Min.Y+u, u, hi)
	fuVLine(ctx, b.Min.X+u, b.Min.Y+2*u, b.Max.Y-2*u, u, hi)
	glyph := c.titleText
	if !st.Active {
		glyph = c.text
	}
	g := b.Inset(b.Dx() * 0.3)
	DrawCross(ctx, g, glyph, l.S(1.5))
}

// TabOutset: Fusion tabs overlap by a pixel; the selected tab reaches over
// its right neighbour's border and is painted last.
func (fusionEngine) TabOutset(l *Classic) Insets {
	return Insets{Right: fuU(l)}
}

// fuPaneTop is where a tab pane's top edge sits: 2px up into the tab bar.
func fuPaneTop(l *Classic, b paintengine2d.Rect) float32 {
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		return b.Min.Y
	}
	return snap(b.Min.Y + th - 2*fuU(l))
}

// DrawTabPane is the tab frame colour in the outline with the inner
// contrast line and a faint shadow under the bottom edge.
func (fusionEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	top := fuPaneTop(l, b)
	pane := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if pane.Dx() < 3*u || pane.Dy() < 3*u {
		return
	}
	ctx.DrawRect(pane, paintengine2d.Fill(c.tabFrame))
	fuHLine(ctx, pane.Min.X, pane.Max.X, pane.Max.Y-u, u, paintengine2d.RGBA(0, 0, 0, 15.0/255))
	fr := paintengine2d.XYWH(pane.Min.X, pane.Min.Y, pane.Dx(), pane.Dy()-u)
	ctx.DrawRect(fr.Inset(u*0.5), paintengine2d.StrokePaint(c.outline, u))
	ctx.DrawRect(fr.Inset(u*1.5), paintengine2d.StrokePaint(c.contrast, u))
}

// ---- controls ----------------------------------------------------------------------------------------------

func (e fusionEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.tabFrame))
		if b.Dx() > 3*u && b.Dy() > 3*u {
			ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.outline, u))
			ctx.DrawRect(b.Inset(u*1.5), paintengine2d.StrokePaint(c.contrast, u))
		}
		return
	}
	// The lightened outline around the window colour.
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	if b.Dx() > 2*u && b.Dy() > 2*u {
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(lighterPct(c.outline, 108), u))
	}
}

func (e fusionEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := fusionColors(l)
	b = fuSnap(b)
	c.button(l, ctx, b, st, st.Primary() && !st.Disabled())
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	// Fusion never shifts the label of a pressed button.
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, l.S(8))
}

// fuToggleLabel draws a check box / radio caption and, with keyboard
// focus, the focus box around it.
func (fusionEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := fusionColors(l)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(l, ctx, fuSnap(box.Inset(-fuU(l)).Intersect(b)))
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	if st.Focused() && !st.Disabled() {
		fr := labelFocusRect(l.body, label, lb, b)
		c.focusRect(l, ctx, fuSnap(fr))
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

func (e fusionEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	if side > b.Dy() {
		side = b.Dy()
	}
	box := fuSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e fusionEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
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

// DrawSwitch: Qt has no switch; Fusion's is a slider groove filled with
// the highlight when on, with the slider handle at its end.
func (e fusionEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := fusionColors(l)
	u := fuU(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if tw <= 0 {
		tw = l.S(40)
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
	g := fuSnap(paintengine2d.XYWH(track.Min.X+u, track.Min.Y+track.Dy()*0.25, track.Dx()-2*u, track.Dy()*0.5))
	r := g.Dy() * 0.5
	if l.square() {
		r = 0
	}
	stops, line := c.groove, c.outline
	if on && !st.Disabled() {
		stops, line = c.grooveHl, c.grooveHlLine
	}
	ctx.DrawRoundRect(g.Inset(u*0.5), r, r, VGradient(g, stops...))
	ctx.DrawRoundRect(g.Inset(u*0.5), r, r, paintengine2d.StrokePaint(line, u))
	hs := track.Dy()
	hx := track.Min.X
	if on {
		hx = track.Max.X - hs
	}
	c.handle(l, ctx, paintengine2d.XYWH(hx, track.Min.Y, hs, hs), st)
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.dis
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		if st.Focused() && !st.Disabled() {
			c.focusRect(l, ctx, fuSnap(labelFocusRect(l.body, label, lb, b)))
		}
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
}

// handle is the slider handle in hb: a 2px-rounded gradient square over a
// translucent black drop, outlined (the focus outline with focus).
func (c *fusion) handle(l *Classic, ctx *paintengine2d.Context, hb paintengine2d.Rect, st ControlState) {
	u := fuU(l)
	hb = fuSnap(hb)
	if hb.Dx() < 5*u || hb.Dy() < 5*u {
		return
	}
	r := paintengine2d.XYWH(hb.Min.X+u, hb.Min.Y+u, hb.Dx()-3*u, hb.Dy()-3*u)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X-u, r.Min.Y+2*u, r.Dx()+2*u, r.Dy()-4*u), paintengine2d.Fill(c.handleDrop))
	rad := fuR(l, 2)
	fill := VGradient(hb.Inset(2*u), c.handleStops...)
	if st.Pressed() && !st.Disabled() {
		fill = paintengine2d.Fill(c.btnDown)
	}
	ctx.DrawRoundRect(r.Inset(u*0.5), rad, rad, fill)
	line := c.outline
	switch {
	case st.Disabled():
		line = c.disOutline
	case st.Focused():
		line = c.hlOutline
	}
	fuStroke(ctx, r, rad, u, line)
	fuStroke(ctx, r.Inset(u), rad, u, c.contrast)
	// The 10/255 shadow along the bottom and right.
	sh := paintengine2d.RGBA(0, 0, 0, 10.0/255)
	fuHLine(ctx, r.Min.X+2*u, r.Max.X-2*u, r.Max.Y, u, sh)
	fuVLine(ctx, r.Max.X, r.Min.Y+4*u, r.Max.Y-3*u, u, sh)
}

// DrawSlider is a 7px groove (a dark→light gradient in the outline), the
// highlight-filled part up to the handle, the handle.
func (e fusionEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	hs := l.metrics.Thumb
	if hs <= 0 {
		hs = l.S(15)
	}
	if hs > b.Dy() {
		hs = b.Dy()
	}
	hs = snap(hs)
	if b.Dx() < hs+2*u || hs < 5*u {
		return
	}
	cy := b.Min.Y + b.Dy()*0.5
	gt := snap(l.S(7))
	g := fuSnap(paintengine2d.XYWH(b.Min.X, cy-gt*0.5, b.Dx(), gt))
	gi := paintengine2d.XYWH(g.Min.X+u, g.Min.Y+u, g.Dx()-3*u, g.Dy()-3*u)
	if gi.Dx() > 2*u && gi.Dy() > u {
		rr := fuR(l, 1)
		ctx.DrawRoundRect(gi.Inset(u*0.5), rr, rr, VGradient(g, c.groove...))
		ctx.DrawRoundRect(gi.Inset(u*0.5), rr, rr, paintengine2d.StrokePaint(c.outline, u))
		hx := snap(b.Min.X + (b.Dx()-hs)*t)
		if !st.Disabled() && hx > gi.Min.X {
			ctx.Save()
			ctx.ClipRect(paintengine2d.XYWH(gi.Min.X, g.Min.Y, hx-gi.Min.X+u, g.Dy()))
			ctx.DrawRoundRect(gi.Inset(u*0.5), rr, rr, VGradient(g, c.grooveHl...))
			ctx.DrawRoundRect(gi.Inset(u*0.5), rr, rr, paintengine2d.StrokePaint(c.grooveHlLine, u))
			ctx.DrawRoundRect(gi.Inset(u*1.5), rr, rr, paintengine2d.StrokePaint(c.contrast, u))
			ctx.Restore()
		}
		c.handle(l, ctx, paintengine2d.XYWH(hx, snap(cy-hs*0.5), hs, hs), st)
	}
}

// DrawProgressBar is a base groove in the outline and the highlight
// gradient bar with a darker leading edge; the busy bar fills and scrolls
// 9px diagonal stripes every 22px.
func (e fusionEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := fuR(l, 2)
	base := c.base
	if st.Disabled() {
		base = c.win
	}
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.Fill(base))
	fuStroke(ctx, b, r, u, c.outline)
	fuHLine(ctx, b.Min.X+u, b.Max.X-u, b.Min.Y+u, u, c.topShadow)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	bar := b
	if !indeterminate {
		w := snap(b.Dx() * t)
		if w < 2*u {
			return
		}
		bar = paintengine2d.XYWH(b.Min.X, b.Min.Y, w, b.Dy())
	}
	stops, line := c.progStops, c.progLine
	if st.Disabled() {
		stops, line = c.progDisStops, c.disOutline
	}
	ctx.DrawRoundRect(bar.Inset(u*0.5), r, r, VGradient(b, stops...))
	fuStroke(ctx, bar, r, u, line)
	fuStroke(ctx, bar.Inset(u), fuR(l, 1), u, paintengine2d.RGBA(1, 1, 1, 50.0/255))
	if indeterminate && !st.Disabled() {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		step := l.S(22)
		h := b.Dy()
		ctx.Save()
		ctx.ClipRect(bar.Inset(u))
		sp := paintengine2d.Paint{Color: c.stripe, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: l.S(9), Cap: paintengine2d.CapButt, MiterLimit: 4}}
		for x := b.Min.X - h - step + phase*step; x < b.Max.X+h; x += step {
			ctx.DrawLine(paintengine2d.Pt(x, b.Max.Y+u), paintengine2d.Pt(x+h, b.Min.Y-2*u), sp)
		}
		ctx.Restore()
	} else if t < 1 && !st.Disabled() {
		// The not-yet-complete bar's leading edge: its darker line and an
		// inner shadow one pixel beyond.
		fuVLine(ctx, bar.Max.X-u, bar.Min.Y+u, bar.Max.Y-u, u, darkerPct(c.hl, 140))
		if bar.Max.X+u < b.Max.X-u {
			fuVLine(ctx, bar.Max.X, bar.Min.Y+u, bar.Max.Y-u, u, paintengine2d.RGBA(0, 0, 0, 35.0/255))
		}
	}
}

// DrawComboBox is a non-editable combo: a push button with the current
// text and the small down arrow in the button text at 160/255.
func (e fusionEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := fusionColors(l)
	b = fuSnap(b)
	bst := st
	if open {
		bst = (bst | StatePressed) &^ StateHovered
	}
	c.button(l, ctx, b, bst, false)
	aw := l.S(18)
	ab := paintengine2d.XYWH(b.Max.X-aw-l.S(1), b.Min.Y, aw, b.Dy())
	col := c.arrow
	if st.Disabled() {
		col = c.dis
	}
	fuArrow(l, ctx, ab, DirDown, col)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	tb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, ab.Min.X-b.Min.X-l.S(8), b.Dy())
	if open {
		tb = tb.Translate(paintengine2d.Pt(fuU(l), fuU(l)))
	}
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
}

// DrawSpinner is the spin box button column: the button gradient in the
// frame's outline with a separator on the left (the field's own frame
// sits next to it), arrows at 160/255, a contrast wash on the hovered
// half and the down fill on the held one.
func (e fusionEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if b.Dx() < 4*u || b.Dy() < 6*u {
		return
	}
	r := fuR(l, 2)
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), b.Dy()-2*u)
	fill := VGradient(body, c.btnStops...)
	if !st.Disabled() && (st.Hovered() || upHover || downHover) {
		fill = VGradient(body, c.btnHotStops...)
	}
	path := RoundRectPath(body.Inset(u*0.5), 0, r, r, 0)
	ctx.DrawPath(path, fill)
	mid := snap((body.Min.Y + body.Max.Y) * 0.5)
	up := paintengine2d.XYWH(body.Min.X+u, body.Min.Y+u, body.Dx()-2*u, mid-body.Min.Y-u)
	dn := paintengine2d.XYWH(body.Min.X+u, mid, body.Dx()-2*u, body.Max.Y-mid-u)
	half := func(r paintengine2d.Rect, hover, press bool) {
		switch {
		case st.Disabled():
		case press:
			ctx.DrawRect(r, paintengine2d.Fill(darkerPct(c.btnColor, 110)))
		case hover:
			ctx.DrawRect(r, paintengine2d.Fill(c.contrast))
		}
	}
	half(up, upHover, upPress)
	half(dn, downHover, downPress)
	line := c.outline
	if st.Focused() && !st.Disabled() {
		line = c.hlOutline
	}
	ctx.DrawRoundRectCorners(body.Inset(u*0.5), 0, r, r, 0, paintengine2d.StrokePaint(line, u))
	col, dcol := c.arrow, c.arrow
	if st.Disabled() {
		col, dcol = c.dis, c.dis
	}
	fuArrow(l, ctx, up.Translate(paintengine2d.Pt(0, u)), DirUp, col)
	fuArrow(l, ctx, dn, DirDown, dcol)
}

func (e fusionEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	// The pane's top edge the tabs stand on (the bar itself is
	// transparent: the window shows through).
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	y := b.Max.Y - 2*u
	if b.Dy() < 4*u {
		return
	}
	fuHLine(ctx, b.Min.X, b.Max.X, y, u, c.outline)
	fuHLine(ctx, b.Min.X, b.Max.X, y+u, u, c.tabFrame)
}

// DrawTab (tabs on top): rounded top corners; unselected tabs sit 2px
// lower in a darker gradient and stop on the pane edge, the selected tab
// is lighter, full height, and opens into the pane.
func (e fusionEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	r := fuR(l, 2)
	paneY := b.Max.Y - 2*u
	var body paintengine2d.Rect
	if selected {
		body = paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), paneY-b.Min.Y)
	} else {
		// No right border: the next tab's left border is shared (the last
		// tab keeps its own).
		w := b.Dx()
		if !st.Last() {
			w += u
		}
		body = paintengine2d.XYWH(b.Min.X, b.Min.Y+2*u, w, paneY-b.Min.Y-2*u)
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), paneY-b.Min.Y))
	shape := paintengine2d.XYWH(body.Min.X, body.Min.Y, body.Dx(), body.Dy()+4*u)
	if selected {
		ctx.DrawRoundRectCorners(shape.Inset(u*0.5), r, r, 0, 0, VGradient(b, c.tabSel...))
		ctx.DrawRoundRectCorners(shape.Inset(u*0.5), r, r, 0, 0, paintengine2d.StrokePaint(c.outline, u))
	} else {
		ctx.DrawRoundRectCorners(shape.Inset(u*0.5), r, r, 0, 0, VGradient(b, c.tabStops...))
		ctx.DrawRoundRectCorners(shape.Inset(u*0.5), r, r, 0, 0, paintengine2d.StrokePaint(c.outlineLt, u))
	}
	ctx.DrawRoundRectCorners(paintengine2d.XYWH(shape.Min.X+u, shape.Min.Y+u, shape.Dx()-2*u, shape.Dy()-u).Inset(u*0.5), r, r, 0, 0, paintengine2d.StrokePaint(c.contrast, u))
	ctx.Restore()
	if selected {
		// Open the pane edge under the tab.
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, paneY, b.Dx()-2*u, b.Max.Y-paneY), paintengine2d.Fill(c.tabFrame))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, paneY, u, b.Max.Y-paneY), paintengine2d.Fill(c.outline))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, paneY, u, b.Max.Y-paneY), paintengine2d.Fill(c.outline))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	lb := paintengine2d.XYWH(body.Min.X, body.Min.Y, b.Dx(), body.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		f := l.body
		w := f.Advance(label) + l.S(8)
		if max := lb.Dx() - l.S(6); w > max {
			w = max
		}
		h := f.Height() + l.S(2)
		if max := lb.Dy() - 2*u; h > max {
			h = max
		}
		c.focusRect(l, ctx, fuSnap(paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h)))
	}
}

// DrawMenuBar is the window colour with a soft line below.
func (e fusionEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.barLine)
}

// DrawMenuTitle: only an open title is marked, with the highlight in its
// darker outline and the highlighted text.
func (e fusionEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if (open || st.Pressed()) && !st.Disabled() {
		hl := b
		if !open {
			hl.Max.Y -= u
		}
		e.MenuHighlight(l, ctx, hl, open)
		fg = c.hlText
	} else {
		fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.barLine)
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		c.focusRect(l, ctx, b.Inset(u))
	}
}

// DrawMenuFrame is the base lighter 108% in the outline with a light /
// shadow bevel inside.
func (e fusionEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.menuBg))
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.outline, u))
	in := b.Inset(u)
	fuHLine(ctx, in.Min.X, in.Max.X, in.Min.Y, u, c.menuLight)
	fuVLine(ctx, in.Min.X, in.Min.Y, in.Max.Y, u, c.menuLight)
	fuHLine(ctx, in.Min.X, in.Max.X, in.Max.Y-u, u, c.menuShadow)
	fuVLine(ctx, in.Max.X-u, in.Min.Y, in.Max.Y, u, c.menuShadow)
}

// DrawMenuItem: the selected row is the highlight in its outline; check
// marks are real check boxes, radio marks a dot, shortcuts share the label
// colour, submenus end in a small arrow.
func (e fusionEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := fusionColors(l)
	u := fuU(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := b.Min.X - ch.PadL + l.S(5)
		x1 := b.Max.X + ch.PadR - l.S(5)
		fuHLine(ctx, x0, x1, y, u, c.darkShade)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.dis
	case hot:
		fg = c.hlText
	}
	// Check column.
	gw := ch.CheckCol()
	side := snap(l.S(14))
	if side > b.Dy()-2*u {
		side = snap(b.Dy() - 2*u)
	}
	ib := fuSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	switch {
	case row.Radio:
		if row.Checked {
			d := side * 0.4
			ctx.DrawCircle(ib.Center(), d*0.5, paintengine2d.Fill(fg))
		}
	case row.Checked:
		cst := StateNone
		if st.Disabled() {
			cst = StateDisabled
		}
		e.CheckIndicator(l, ctx, ib, cst, true)
	case row.Icon != IconNone:
		l.drawToolIcon(ctx, ib.Inset(-l.S(1)), row.Icon, fg)
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		fuArrow(l, ctx, ab, DirRight, fg)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := l.body.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		l.body.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg)
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
// label colour: the highlight for a selection (Fusion keeps it when
// the view or the window loses focus — the standard palette's inactive
// highlight is the active one), the disabled highlight (145,145,145) in a
// disabled view. Fusion item views show no hover.
func (c *fusion) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if st.Checked() {
		fill := c.hl
		if st.Disabled() {
			fill = c.disHl
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		return c.hlText
	}
	if st.Disabled() {
		return c.dis
	}
	return l.fieldText()
}

func (e fusionEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := fusionColors(l)
	fg := c.row(l, ctx, b, st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e fusionEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := fusionColors(l)
	// The selection spans the whole row, branch included.
	fg := c.row(l, ctx, b, st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	aw := l.S(14)
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, aw, b.Dy()), expanded, c.arrow)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + aw + l.S(4)
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e fusionEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := fusionColors(l)
	u := fuU(l)
	fg := c.row(l, ctx, b, st)
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
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
	// Grid lines: the window darker 120%.
	grid := darkerPct(c.win, 120)
	fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y, u, grid)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, grid)
}

// DrawTableHeader is the header gradient with a hard break at half height,
// the inner contrast line on top, the outline below, a translucent
// separator on the right and the sort arrow.
func (e fusionEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	if b.Empty() {
		return
	}
	if st.Pressed() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.headerDown))
	} else {
		ctx.DrawRect(b, VGradient(b, c.headerStops...))
	}
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y, u, c.contrast)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.outline)
	if !st.Last() {
		fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y-u, u, paintengine2d.RGBA(0, 0, 0, 40.0/255))
		fuVLine(ctx, b.Max.X-2*u, b.Min.Y, b.Max.Y-u, u, c.contrast)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			// Qt on Linux points the arrow up for an ascending sort.
			dir = DirUp
		}
		ab := paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()-l.S(4))
		fuArrow(l, ctx, ab, dir, c.text.WithAlpha(180.0/255))
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar is a window lighter 104% → window gradient with a light top
// line and the shadow / light pair at the bottom.
func (e fusionEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, VGradient(b, c.toolStops...))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Min.Y, u, c.lightShade)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-2*u, u, c.darkShade)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.lightShade)
}

// DrawStatusBar: the window colour, plain texts and the size grip's dots.
func (e fusionEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	grip := l.S(16)
	if n := len(parts); n > 0 {
		slot := (b.Dx() - grip) / float32(n)
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y, slot-l.S(12), b.Dy()), c.text, AlignStart, 0)
		}
	}
	// Size grip: a triangle of raised dots in the bottom-right corner.
	if b.Dx() < grip || b.Dy() < grip*0.75 {
		return
	}
	gap := 3 * u
	ox, oy := b.Max.X-l.S(4), b.Max.Y-l.S(4)
	for i := 0; i < 4; i++ {
		for j := 0; j <= i; j++ {
			x := snap(ox - float32(i-j+1)*gap)
			y := snap(oy - float32(j+1)*gap)
			ctx.DrawRect(paintengine2d.XYWH(x, y, 2*u, 2*u), paintengine2d.Fill(c.lightShade))
			ctx.DrawRect(paintengine2d.XYWH(x, y, u, u), paintengine2d.Fill(c.darkShade))
		}
	}
}

// DrawTitleBar (panel headings): the window colour, a bold title and the
// menu bar's soft rule below.
func (e fusionEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.barLine)
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), l.palette.TextMuted, AlignStart, 0)
	}
}

// DrawAccordionHeader is a tool box tab: a button face with the branch
// arrow on the left.
func (e fusionEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := fusionColors(l)
	b = fuSnap(b)
	c.button(l, ctx, b, st&^StateFocused, false)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, l.S(12), b.Dy()), expanded, fg)
	lb := paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy())
	l.drawFittedText(ctx, l.body, title, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, fuSnap(labelFocusRect(l.body, title, lb, b)))
	}
}

// DrawSeparator is a window darker 110% / lighter 110% line pair with 6px
// margins.
func (e fusionEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := fusionColors(l)
	u := fuU(l)
	m := l.S(6)
	if vertical {
		if b.Dy() <= 2*m {
			m = 0
		}
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		fuVLine(ctx, x, b.Min.Y+m, b.Max.Y-m, u, c.sepDark)
		fuVLine(ctx, x+u, b.Min.Y+m, b.Max.Y-m, u, c.sepLight)
		return
	}
	if b.Dx() <= 2*m {
		m = 0
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	fuHLine(ctx, b.Min.X+m, b.Max.X-m, y, u, c.sepDark)
	fuHLine(ctx, b.Min.X+m, b.Max.X-m, y+u, u, c.sepLight)
}

// DrawSplitter is a column (or row) of raised 2px dots; the hot handle
// gets a faint highlight wash.
func (e fusionEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := fusionColors(l)
	u := fuU(l)
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.hl.WithAlpha(0.15)))
	}
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	gap := 3 * u
	for i := -2; i <= 3; i++ {
		x, y := cx, cy+float32(i)*gap-u
		if !vertical {
			x, y = cx+float32(i)*gap-u, cy
		}
		d := paintengine2d.XYWH(x-u, y, 2*u, 2*u)
		if !d.Intersect(b).Empty() && d.Intersect(b) == d {
			ctx.DrawRect(d, paintengine2d.Fill(c.lightShade))
			ctx.DrawRect(paintengine2d.XYWH(x-u, y, u, u), paintengine2d.Fill(c.darkShade))
		}
	}
}

// DrawTooltip is the tooltip base in a 1px frame of the tooltip text
// colour (Qt's pale yellow and black).
func (e fusionEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := fusionColors(l)
	u := fuU(l)
	b = fuSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.tip))
	if b.Dx() > 2*u && b.Dy() > 2*u {
		ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.tipBorder, u))
	}
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs -----------------------------------------------------------------------------------------------------

func fusionPack(name, label, summary string, fam ThemeName, pal Palette, extra map[string]string) ThemePack {
	tok := ThemeTokens{
		Engine:  "fusion",
		Bevel:   BevelClassic3D,
		Family:  fam,
		Palette: pal,
		Era:     EraFusion,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	fusionChrome(&tok)
	return ThemePack{
		Name: name, Label: label, Year: 2012, Lineage: "Qt", Summary: summary,
		Era: EraFusion, Palette: fam, Tokens: tok,
	}
}

// fusionChrome sets the chrome states Fusion derives from its palette
// (pressed comes from the resolver).
func fusionChrome(tok *ThemeTokens) {
	pal := tok.Palette
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
}

// fusionHighlight sets the palette entries Fusion takes from QPalette's
// Highlight: the accent, its hover and pressed shades, focus, selection
// and the menu highlight in its darker outline.
func fusionHighlight(p *Palette, hl paintengine2d.Color) {
	p.Accent, p.Focus, p.Selection, p.MenuHover = hl, hl, hl, hl
	p.AccentHover, p.AccentPress = lighterPct(hl, 110), darkerPct(hl, 115)
	p.MenuHoverBorder = darkerPct(hl, 125)
}

// Accented is Qt's QPalette::Highlight, which Plasma (5.25 and later) sets
// from the accent colour for every Qt application: Fusion draws its
// selections, the menu highlight, focus outlines, the default button's
// tint, slider and progress fills and its in-app title bars from it. The
// highlighted text stays the palette's while it reads at 3:1 and turns to
// the other text colour on a pale accent.
func (fusionEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	p := tok.Palette
	text := ReadableOn(accent, 3, accentX(tok, "highlightText", p.TextOnAccent), p.Text)
	tok = CloneTokenMaps(tok)
	tok.Extra["highlight"], tok.Extra["highlightText"] = accent, text
	fusionHighlight(&tok.Palette, accent)
	tok.Palette.TextOnAccent = text
	fusionChrome(&tok)
	tok.Pressed = ChromeState{} // the resolver's, from the new palette
	return tok.Resolve()
}

// fusionPalette maps Qt-style palette roles onto the shared palette
// (disabled text is the "disabledText" extra).
func fusionPalette(window, windowText, base, highlight, highlightText, placeholder string) Palette {
	w := hexColor(window)
	outline := darkerPct(w, 140)
	p := Palette{
		Background: w, Surface: w, SurfaceAlt: w,
		Border: outline, Divider: darkerPct(w, 120),
		Text: hexColor(windowText), TextMuted: hexColor(placeholder), TextOnAccent: hexColor(highlightText),
		Field: hexColor(base), FieldBorder: outline,
		Track: darkerPct(w, 106), Thumb: lighterPct(w, 104),
		Highlight: paintengine2d.RGBA(1, 1, 1, 30.0/255), Shadow: paintengine2d.RGBA(0, 0, 0, 0.2),
		MenuGutter: lighterPct(hexColor(base), 108),
		Danger:     hexColor("#c0392b"), Success: hexColor("#2e8b57"), Warning: hexColor("#c07000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.32),
		BevelLight: lighterPct(w, 150), BevelDark: darkerPct(w, 150),
	}
	fusionHighlight(&p, hexColor(highlight))
	return p
}

func fusionPacks() []ThemePack {
	// Fusion's standard light palette (Qt 5): window #efefef, base white,
	// highlight (48,140,198), disabled text (190,190,190), placeholder
	// text at 50%.
	light := fusionPalette("#efefef", "#000000", "#ffffff", "#308cc6", "#ffffff", "#7f7f7f")
	// The dark palette the Qt community has used with Fusion since
	// QuantumCD's 2013 gist: window (53,53,53), base (25,25,25), highlight
	// (42,130,218) with black highlighted text, a blue tooltip.
	dark := fusionPalette("#353535", "#ffffff", "#191919", "#2a82da", "#000000", "#8c8c8c")
	dark.Danger, dark.Success, dark.Warning = hexColor("#ff6b5b"), hexColor("#5fcf80"), hexColor("#f0b84a")
	dark.Overlay = paintengine2d.RGBA(0, 0, 0, 0.5)
	dark.Shadow = paintengine2d.RGBA(0, 0, 0, 0.45)
	return []ThemePack{
		fusionPack("fusion", "Fusion", "Qt's cross-platform style: 2px gradient buttons, highlight focus outlines, square scroll sliders.", ThemeLight, light,
			map[string]string{"disabledText": "#bebebe", "tip": "#ffffdc", "tipText": "#000000"}),
		fusionPack("fusion-night", "Fusion Dark", "Fusion in the widely used community dark palette. Not historical: Qt's 2012 Fusion shipped a light palette only.", ThemeDark, dark,
			map[string]string{"disabledText": "#7f7f7f", "tip": "#2a82da", "tipText": "#ffffff", "tipBorder": "#ffffff"}),
	}
}
