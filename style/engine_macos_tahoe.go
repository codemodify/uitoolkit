package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// tahoeEngine paints macOS 26 Tahoe's Liquid Glass (2025), light and dark:
// the macos engine's era 2. Its numbers are Apple's published ones (the
// Human Interface Guidelines, AppKit's documentation, the WWDC25 design
// sessions' labelled slides, the 2025 system colours) and a few measured
// on those sessions' frames; the drawing is this file's own and uses none of
// Apple's artwork.
//
//   - Liquid Glass is the functional layer: menus and popovers are glass over
//     a blurred backdrop, the sidebar a floating glass panel inset from the
//     window's edges, tool bar buttons share glass capsules (one per group,
//     ToolGroupEngine); glass is a translucent tint in a bright specular rim,
//     brighter at the top, over a soft shadow. Content (lists, tables, text)
//     stays solid;
//   - controls are taller (the medium size is 24pt) and rounder: push
//     buttons, pop-up buttons, fields and segmented controls are rounded
//     rectangles of about a quarter of their height, capsules from the
//     large size up; nested shapes are concentric;
//   - a push button is a flat grey face, no bezel; the default button is the
//     accent (#0088FF, #0091FF dark) with white text, plain again in an
//     inactive window; a toggled one wears the accent tint of Apple's
//     secondary prominence;
//   - check boxes and radios are white in a hairline, the accent when on;
//     the switch is iOS 26's wide capsule with a capsule knob, and switch and
//     slider knobs turn to glass while they are dragged;
//   - the slider is a 5pt track, the accent up to a white capsule knob;
//   - lists, trees and tables are rounded cards with rows inset from their
//     sides, rounded alternating stripes and a rounded accent selection;
//     the sidebar's selection is a grey row with its label in the accent;
//   - menus are 13pt-rounded glass with 25pt rows and a rounded accent
//     highlight; windows round their corners by 16pt (the title-bar window
//     radius).
//
// Pack data: every colour key of tahoeLight can be overridden through
// "extra"; params "era" 2 and "vibrancy" (0 turns the menus' blurred
// backdrop off).
type tahoeEngine struct{ macosEngine }

func init() {
	for _, p := range tahoePacks() {
		RegisterPack(p)
	}
}

// EngineFor paints packs with param "era" 2 with tahoeEngine.
func (macosEngine) EngineFor(t ThemeTokens) Engine {
	if t.Params["era"] >= 2 {
		return tahoeEngine{}
	}
	return nil
}

// DefaultMetrics are Tahoe's medium control sizes at the toolkit's 16px UI
// font (AppKit's 13pt, scaled as the macos engine scales Big Sur): the 24pt
// push button, field and pop-up face becomes 28px in a 34px cell whose
// margin holds the focus halo, 20px check boxes and radios, a 54×24 switch,
// a 32×20 slider knob, 30px menu rows and 32px list rows, a 48px tool bar
// for 36pt glass buttons.
func (tahoeEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 8,
		Radius:    8, RadiusSmall: 6,
		ControlH: 34, FieldH: 34, ComboH: 34,
		Checkbox: 26, Radio: 26,
		MenuItemH: 30, MenuBarH: 28, TabH: 34, RowH: 32,
		TitleBar: 34, HeaderH: 30, ProgressH: 16, SliderH: 30, Thumb: 32,
		Scroll: 16, Pad: 14, FieldPad: 10, FocusWidth: 3, Border: 1,
		ToolBarH: 48, StatusBarH: 26, SpinnerW: 24, SwitchW: 60, SwitchH: 30,
	}
}

// ---- colour tables ------------------------------------------------------------------------------

// tahoeLight is Tahoe's light appearance with the default Blue accent. The
// accents and status colours are Apple's 2025 system colours; the greys are
// the label colours (black at 85%, 50%, 25%) and fills as they measure on
// Apple's frames. Translucent keys ("#rrggbbaa") composite over what is
// under them, so a button reads the same on the window, a card or a view.
var tahoeLight = macScheme{
	// Window, content (views, fields), labels.
	"win": "#f2f2f4", "content": "#ffffff", "text": "#262626", "text2": "#7f7f7f", "text3": "#bfbfbf", "onAccent": "#ffffff",
	// The accent, pressed, and its tint (secondary prominence).
	"accent": "#0088ff", "accentPress": "#0070d6", "tint": "#0088ff2e",
	// Push button faces (the fill of "none" prominence), pressed.
	"btn": "#0000001a", "btnPress": "#00000033", "btnText": "#262626",
	// Text fields.
	"field": "#ffffff", "fieldEdge": "#00000024", "fieldDis": "#f7f7f8",
	// An unchecked box or radio: fill and hairline.
	"chk": "#ffffff", "chkEdge": "#0000002e",
	// Switch track (off), slider and progress tracks, the white knob and
	// its hairline.
	"switchOff": "#00000017", "track": "#0000000f", "progress": "#0000000f", "knob": "#ffffff", "knobEdge": "#0000001a",
	// Selections: a focused list's, an unfocused one's, text.
	"sel": "#0070ea", "selOff": "#0000001a", "selOffText": "#262626", "textSel": "#b3dbff",
	// Focus halo, separators, the alternating row stripe.
	"focus": "#0088ff80", "sep": "#0000001a", "alt": "#0000000a",
	// Group boxes and the tab pane.
	"box": "#0000000a", "boxEdge": "#0000000f",
	// Glass: the sidebar panel's and tool bar groups' tint, the specular
	// rim (top, bottom), the hairline that keeps it off a light ground.
	"glass": "#ffffffb3", "glassRim": "#ffffff", "glassRim2": "#ffffff66", "glassEdge": "#0000000f",
	// Menus: the tint over the blurred backdrop, edge, separator, the hot row.
	"menu": "#f6f6f8d9", "menuEdge": "#0000001f", "menuSep": "#0000001a", "menuHi": "#0088ff",
	// Help tags.
	"tip": "#f7f7f9f5", "tipEdge": "#0000001f", "tipText": "#262626",
	// The sidebar panel as it reads (the glass over the window), in an
	// inactive window; its selection and the selected label.
	"side": "#fafafb", "sideOff": "#f6f6f7", "sideSel": "#0000001a", "sideText": "#0070ea",
	// Segmented control track and knob.
	"segTrack": "#0000000f", "segKnob": "#ffffff",
	// The overlay scroller.
	"scroller": "#00000080", "scrollerHot": "#00000099", "scrollerRim": "#ffffff30", "scrollTrack": "#fafafabf", "scrollTrackEdge": "#e7e7e7",
	// Title text (active, inactive), the window outline.
	"titleText": "#262626", "titleOff": "#b0b0b0", "winEdge": "#00000026",
	// Traffic lights, their rims, the inactive lights, the glyph.
	"close": "#ff5f57", "closeEdge": "#e0443e", "mini": "#febc2e", "miniEdge": "#dea123",
	"zoom": "#28c840", "zoomEdge": "#1aab29", "lightOff": "#d0d0d0", "lightOffEdge": "#bcbcbc", "glyph": "#4d0000",
	// Table header, its label.
	"header": "#ffffff", "headerText": "#7f7f7f",
	// Tool bar buttons: the wash under the pointer, pressed; a latched
	// button's glyph.
	"wash": "#0000000f", "washPress": "#0000001f", "toolOnText": "#0088ff",
	// System colours.
	"red": "#ff383c", "green": "#34c759", "orange": "#ff8d28",
}

// tahoeDark is Tahoe's dark appearance.
var tahoeDark = macScheme{
	"win": "#2a2a2c", "content": "#1c1c1e", "text": "#dfdfe1", "text2": "#98989d", "text3": "#5a5a5e", "onAccent": "#ffffff",
	"accent": "#0091ff", "accentPress": "#0078d4", "tint": "#0091ff38",
	"btn": "#ffffff1f", "btnPress": "#ffffff3d", "btnText": "#e5e5e7",
	"field": "#ffffff0d", "fieldEdge": "#ffffff26", "fieldDis": "#ffffff08",
	"chk": "#ffffff1a", "chkEdge": "#ffffff33",
	"switchOff": "#ffffff29", "track": "#ffffff1f", "progress": "#ffffff1f", "knob": "#f2f2f7", "knobEdge": "#00000033",
	"sel": "#0066dc", "selOff": "#ffffff26", "selOffText": "#dfdfe1", "textSel": "#1f4f80",
	"focus": "#0091ff80", "sep": "#ffffff1a", "alt": "#ffffff08",
	"box": "#ffffff0a", "boxEdge": "#ffffff12",
	"glass": "#ffffff14", "glassRim": "#ffffff47", "glassRim2": "#ffffff0f", "glassEdge": "#00000066",
	"menu": "#2c2c2ed9", "menuEdge": "#000000b3", "menuSep": "#ffffff1f", "menuHi": "#0091ff",
	"tip": "#2c2c2ef5", "tipEdge": "#000000b3", "tipText": "#dfdfe1",
	"side": "#373739", "sideOff": "#2f2f31", "sideSel": "#ffffff1f", "sideText": "#48a8ff",
	"segTrack": "#ffffff14", "segKnob": "#5b5b5f",
	"scroller": "#ffffff80", "scrollerHot": "#ffffff99", "scrollerRim": "#00000030", "scrollTrack": "#c9c9c926", "scrollTrackEdge": "#ffffff1f",
	"titleText": "#dfdfe1", "titleOff": "#7a7a7e", "winEdge": "#000000cc",
	"close": "#ff5f57", "closeEdge": "#e0443e", "mini": "#febc2e", "miniEdge": "#dea123",
	"zoom": "#28c840", "zoomEdge": "#1aab29", "lightOff": "#4d4d4f", "lightOffEdge": "#444446", "glyph": "#4d0000",
	"header": "#1c1c1e", "headerText": "#98989d",
	"wash": "#ffffff1a", "washPress": "#ffffff2e", "toolOnText": "#3aa0ff",
	"red": "#ff4245", "green": "#30d158", "orange": "#ff9230",
}

// ---- resolved colour set ------------------------------------------------------------------------

// tahoeSet is a look's resolved Tahoe colours (built once per look).
type tahoeSet struct {
	dark bool

	win, content, text, text2, text3, onAccent paintengine2d.Color
	accent, accentPress, tint                  paintengine2d.Color
	btn, btnPress, btnText                     paintengine2d.Color
	field, fieldEdge, fieldDis, chk, chkEdge   paintengine2d.Color
	switchOff, track, progress, knob, knobEdge paintengine2d.Color
	sel, selOff, selOffText, textSel           paintengine2d.Color
	focus, sep, alt, box, boxEdge              paintengine2d.Color
	glass, glassRim, glassRim2, glassEdge      paintengine2d.Color
	menu, menuEdge, menuSep, menuHi            paintengine2d.Color
	tip, tipEdge, tipText                      paintengine2d.Color
	side, sideOff, sideSel, sideText           paintengine2d.Color
	segTrack, segKnob                          paintengine2d.Color
	scroller, scrollerHot, scrollerRim         paintengine2d.Color
	scrollTrack, scrollTrackEdge               paintengine2d.Color
	titleText, titleOff, winEdge               paintengine2d.Color
	lights, lightEdges                         [3]paintengine2d.Color
	lightOff, lightOffEdge, glyph              paintengine2d.Color
	header, headerText                         paintengine2d.Color
	wash, washPress, toolOnText                paintengine2d.Color

	// busy is the progress bar's sweeping segment (accent, dim).
	busy, busyDim []paintengine2d.GradientStop
}

type tahoeKey struct{}

func tahoeColors(l *Classic) *tahoeSet {
	return l.Memo(tahoeKey{}, func() any { return tahoeBuild(l) }).(*tahoeSet)
}

func tahoeBuild(l *Classic) *tahoeSet {
	dark := Luma(l.palette.Background) < 0.5
	sc := tahoeLight
	if dark {
		sc = tahoeDark
	}
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	c := &tahoeSet{dark: dark}
	c.win, c.content, c.text, c.text2, c.text3, c.onAccent = col("win"), col("content"), col("text"), col("text2"), col("text3"), col("onAccent")
	c.accent, c.accentPress, c.tint = col("accent"), col("accentPress"), col("tint")
	c.btn, c.btnPress, c.btnText = col("btn"), col("btnPress"), col("btnText")
	c.field, c.fieldEdge, c.fieldDis, c.chk, c.chkEdge = col("field"), col("fieldEdge"), col("fieldDis"), col("chk"), col("chkEdge")
	c.switchOff, c.track, c.progress, c.knob, c.knobEdge = col("switchOff"), col("track"), col("progress"), col("knob"), col("knobEdge")
	c.sel, c.selOff, c.selOffText, c.textSel = col("sel"), col("selOff"), col("selOffText"), col("textSel")
	c.focus, c.sep, c.alt, c.box, c.boxEdge = col("focus"), col("sep"), col("alt"), col("box"), col("boxEdge")
	c.glass, c.glassRim, c.glassRim2, c.glassEdge = col("glass"), col("glassRim"), col("glassRim2"), col("glassEdge")
	c.menu, c.menuEdge, c.menuSep, c.menuHi = col("menu"), col("menuEdge"), col("menuSep"), col("menuHi")
	c.tip, c.tipEdge, c.tipText = col("tip"), col("tipEdge"), col("tipText")
	c.side, c.sideOff, c.sideSel, c.sideText = col("side"), col("sideOff"), col("sideSel"), col("sideText")
	c.segTrack, c.segKnob = col("segTrack"), col("segKnob")
	c.scroller, c.scrollerHot, c.scrollerRim = col("scroller"), col("scrollerHot"), col("scrollerRim")
	c.scrollTrack, c.scrollTrackEdge = col("scrollTrack"), col("scrollTrackEdge")
	c.titleText, c.titleOff, c.winEdge = col("titleText"), col("titleOff"), col("winEdge")
	c.lights = [3]paintengine2d.Color{col("close"), col("mini"), col("zoom")}
	c.lightEdges = [3]paintengine2d.Color{col("closeEdge"), col("miniEdge"), col("zoomEdge")}
	c.lightOff, c.lightOffEdge, c.glyph = col("lightOff"), col("lightOffEdge"), col("glyph")
	c.header, c.headerText = col("header"), col("headerText")
	c.wash, c.washPress, c.toolOnText = col("wash"), col("washPress"), col("toolOnText")
	sweep := func(col paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col.WithAlpha(0)), Stop(0.5, col), Stop(1, col.WithAlpha(0))}
	}
	c.busy, c.busyDim = sweep(c.accent), sweep(c.text3)
	return c
}

// flat is col composited over the window colour.
func (c *tahoeSet) flat(col paintengine2d.Color) paintengine2d.Color { return webOver(c.win, col) }

// ---- shapes -------------------------------------------------------------------------------------

// ctlR is the corner of a control face f: about a quarter of its height for
// the mini to medium sizes, a capsule from the large size (28pt) up.
func (c *tahoeSet) ctlR(l *Classic, f paintengine2d.Rect) float32 {
	if l.square() {
		return 0
	}
	h := f.Dy()
	if h >= l.S(33) {
		return h * 0.5
	}
	return min(h*0.26, f.Dx()*0.5)
}

// rad is a design radius at the look's scale, no more than half of b's
// short side.
func tahoeRad(l *Classic, v float32, b paintengine2d.Rect) float32 {
	return min(l.rx(v), min(b.Dx(), b.Dy())*0.5)
}

// paintGlass paints a piece of Liquid Glass into b (corner r): the backdrop
// blurred when blur (menus over content; a sidebar or a tool bar group over
// the flat window has nothing to blur), the tint, the hairline edge that
// keeps it off a light ground and the specular rim inside it, bright along
// the top and faint at the bottom.
func (c *tahoeSet) paintGlass(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, tint, edge paintengine2d.Color, blur bool) {
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r = min(r, b.Dx()*0.5, b.Dy()*0.5)
	if blur {
		// A menu on a surface of its own has none of the window's content
		// under it: the compositor's glass, or nothing to blur at all.
		tint, blur = LayerMaterial(tint, l.Palette().Background)
	}
	if blur && tint.A < 1 && l.P("vibrancy", 1) != 0 {
		ctx.Save()
		ctx.ClipRoundRect(b, r, r)
		ctx.BackdropBlur(b, l.S(14))
		ctx.Restore()
	}
	macFill(ctx, b, r, paintengine2d.Fill(tint))
	px := macPx(l)
	macRing(ctx, b, r, px, paintengine2d.Fill(edge))
	in := b.Inset(px)
	macRing(ctx, in, max(r-px, 0), px, VGradient(in, Stop(0, c.glassRim), Stop(1, c.glassRim2)))
}

// tahoeShadowAround paints DropShadow of the round rect b outside b only: a
// translucent layer over b then shows what lies under it, not its own
// shadow's core.
func tahoeShadowAround(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color, dy, blur float32) {
	if b.Empty() || col.A <= 0 {
		return
	}
	r = min(r, b.Dx()*0.5, b.Dy()*0.5)
	p := paintengine2d.NewPath()
	p.AddRect(ShadowReach(0, dy, blur, 0).Grow(b))
	p.AddRoundRect(b, r, r)
	ctx.Save()
	ctx.ClipPathRule(p, paintengine2d.FillEvenOdd)
	DropShadow(ctx, b, r, col, 0, dy, blur, 0)
	ctx.Restore()
}

// glassShadow is the soft shadow round a piece of glass that sits in the
// window (a tool bar group, the sidebar), within reach px of b.
func (c *tahoeSet) glassShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r, reach float32) {
	a := float32(0.1)
	if c.dark {
		a = 0.35
	}
	blur := max(reach*2-l.S(2), 1)
	tahoeShadowAround(ctx, b, r, paintengine2d.RGBA(0, 0, 0, a), l.S(1), blur)
}

// halo paints the focus ring: a band of the focus colour hugging the
// outside of face (radius r), kept inside clip.
func (c *tahoeSet) halo(l *Classic, ctx *paintengine2d.Context, face, clip paintengine2d.Rect, r float32) {
	if face.Empty() {
		return
	}
	w := macHaloW(l)
	ctx.Save()
	ctx.ClipRect(macSnap(clip))
	macRing(ctx, face.Inset(-w), r+w, w+macPx(l)*0.5, paintengine2d.Fill(c.focus))
	ctx.Restore()
}

// knob paints a white capsule knob k (a switch's, a slider's) over its soft
// shadow; held (dragged), it grows and turns to glass.
func (c *tahoeSet) knobAt(l *Classic, ctx *paintengine2d.Context, k paintengine2d.Rect, st ControlState) {
	if k.Dx() < 2 || k.Dy() < 2 {
		return
	}
	px := macPx(l)
	dis := st.Disabled()
	if st.Pressed() && !dis {
		g := k.Inset(-snap(l.S(2)))
		r := min(g.Dx(), g.Dy()) * 0.5
		tahoeShadowAround(ctx, g, r, paintengine2d.RGBA(0, 0, 0, 0.22), px, l.S(4))
		c.paintGlass(l, ctx, g, r, c.knob.WithAlpha(0.55), c.glassEdge, false)
		return
	}
	r := min(k.Dx(), k.Dy()) * 0.5
	if !dis {
		DropShadow(ctx, k, r, paintengine2d.RGBA(0, 0, 0, 0.2), 0, px*0.5, l.S(2.5), 0)
	}
	fill := c.knob
	if dis {
		fill = Mix(c.flat(c.btn), c.knob, 0.5)
	}
	macFill(ctx, k, r, paintengine2d.Fill(fill))
	macRing(ctx, k, r, px*0.5, paintengine2d.Fill(c.knobEdge))
}

// ---- parts --------------------------------------------------------------------------------------

func (e tahoeEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := tahoeColors(l)
	switch role {
	case RoleButton, RoleCombo:
		return c.button(l, ctx, macFace(l, b), st, role == RoleButton)
	case RoleTool:
		fg, _ := c.tool(l, ctx, b, st)
		return fg
	case RoleField:
		c.fieldFace(l, ctx, macFace(l, b), st)
		if st.Disabled() {
			return c.text3
		}
		return c.text
	case RoleCheck:
		f := macFace(l, b)
		c.toggleGlyph(l, ctx, f, tahoeRad(l, 5.5, f), st, false)
		return c.text
	case RoleRow:
		box, r := c.rowBox(l, macSnap(b), false)
		return c.row(l, ctx, box, st, r)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
			return c.onAccent
		}
		return c.text
	case RoleTab:
		return c.text
	case RoleThumb:
		macFill(ctx, b, min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.scroller))
		return c.text
	case RoleTrack:
		macFill(ctx, b, min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.track))
		return c.text
	case RoleBar, RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.win))
		return c.text
	}
	return c.text
}

// button paints a push button (or pop-up button, push false) face f and
// returns its label colour: the flat grey face, darker while pressed; the
// default button in the accent (plain in an inactive window); a toggled
// button in the accent's tint.
func (c *tahoeSet) button(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState, push bool) paintengine2d.Color {
	if f.Dx() < 2 || f.Dy() < 2 {
		return c.btnText
	}
	r := c.ctlR(l, f)
	dis := st.Disabled()
	pressed := st.Pressed() && !dis
	switch {
	case push && st.Primary() && !st.Backdrop() && !dis:
		fill := c.accent
		if pressed {
			fill = c.accentPress
		}
		macFill(ctx, f, r, paintengine2d.Fill(fill))
		return c.onAccent
	case push && st.Toggle() && st.Checked() && !dis:
		macFill(ctx, f, r, paintengine2d.Fill(c.tint))
		if pressed {
			macFill(ctx, f, r, paintengine2d.Fill(c.wash))
		}
		return c.sel
	}
	fill := c.btn
	switch {
	case dis:
		fill = webA(c.btn, 0.5)
	case pressed:
		fill = c.btnPress
	}
	macFill(ctx, f, r, paintengine2d.Fill(fill))
	if dis {
		return c.text3
	}
	return c.btnText
}

// fieldR is a text field's corner: a medium push button's, whatever the
// field's height (a text view does not grow rounder).
func (c *tahoeSet) fieldR(l *Classic, f paintengine2d.Rect) float32 {
	return min(c.ctlR(l, f), l.rx(7.5))
}

// fieldFace paints a text field face f: the content colour in a hairline,
// rounded as a medium push button is.
func (c *tahoeSet) fieldFace(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState) {
	if f.Dx() < 2 || f.Dy() < 2 {
		return
	}
	r := c.fieldR(l, f)
	fill := c.field
	if st.Disabled() {
		fill = c.fieldDis
	}
	macFill(ctx, f, r, paintengine2d.Fill(fill))
	macRing(ctx, f, r, macPx(l), paintengine2d.Fill(c.fieldEdge))
}

// toggleGlyph paints a check box or radio face g (corner r): white in a
// hairline over a faint shadow, the accent when on (its pressed shade
// held); a disabled one fades.
func (c *tahoeSet) toggleGlyph(l *Classic, ctx *paintengine2d.Context, g paintengine2d.Rect, r float32, st ControlState, on bool) {
	if g.Dx() < 2 || g.Dy() < 2 {
		return
	}
	px := macPx(l)
	dis := st.Disabled()
	if on {
		fill := c.accent
		switch {
		case dis:
			fill = c.btn
		case st.Pressed():
			fill = c.accentPress
		}
		macFill(ctx, g, r, paintengine2d.Fill(fill))
		return
	}
	if !dis {
		DropShadow(ctx, g, r, paintengine2d.RGBA(0, 0, 0, 0.06), 0, px*0.5, l.S(1.5), 0)
	}
	fill := c.chk
	if st.Pressed() && !dis {
		fill = webOver(c.chk, c.wash)
	}
	macFill(ctx, g, r, paintengine2d.Fill(fill))
	macRing(ctx, g, r, px, paintengine2d.Fill(c.chkEdge))
	if dis {
		macFill(ctx, g, r, paintengine2d.Fill(c.win.WithAlpha(0.5)))
	}
}

// tahoeCheckS is a check box's and radio's side: 16.5pt.
func tahoeCheckS(l *Classic, box paintengine2d.Rect) float32 {
	return snap(min(l.S(20), box.Dx(), box.Dy()))
}

func (tahoeEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := tahoeColors(l)
	s := tahoeCheckS(l, box)
	if s < 4 {
		return
	}
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(macSnap(box))
	g := macCentered(box, s, s)
	c.toggleGlyph(l, ctx, g, tahoeRad(l, 5.5, g), st, checked)
	if checked {
		col := c.onAccent
		if st.Disabled() {
			col = c.text3
		}
		macTick(ctx, g.Inset(s*0.17), col, max(l.S(2), 1.2))
	}
}

func (tahoeEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := tahoeColors(l)
	d := tahoeCheckS(l, box)
	if d < 4 {
		return
	}
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(macSnap(box))
	g := macCentered(box, d, d)
	c.toggleGlyph(l, ctx, g, d*0.5, st, selected)
	if selected {
		col := c.onAccent
		if st.Disabled() {
			col = c.text3
		}
		ctx.DrawCircle(g.Center(), d*0.2, paintengine2d.Fill(col))
	}
}

// Arrow is Tahoe's chevron.
func (tahoeEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	macChevron(l, ctx, b, dir, col, l.S(9))
}

// Expander is the disclosure chevron.
func (tahoeEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	macChevron(l, ctx, b, dir, col, l.S(9))
}

// MenuHighlight is a menu's hot row: a rounded box of the accent inset from
// the menu's sides; an open menu-bar title is a rounded wash.
func (tahoeEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	col := c.menuHi
	if attachBottom {
		col = c.washPress
	}
	macFill(ctx, b, tahoeRad(l, 8, b), paintengine2d.Fill(col))
}

func (tahoeEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := tahoeColors(l)
	if hot {
		return c.onAccent
	}
	return c.text
}

// DrawFocusRing is the focus halo just inside b.
func (tahoeEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := tahoeColors(l)
	b = macSnap(b)
	w := macHaloW(l)
	if b.Dx() < 3*w || b.Dy() < 3*w {
		return
	}
	macRing(ctx, b, min(l.rx(8)+w, b.Dy()*0.5), w, paintengine2d.Fill(c.focus))
}

// ---- scroll bars --------------------------------------------------------------------------------

// DrawScrollBarParts is the overlay scroller: a 7pt knob that floats over
// the content and widens to 11pt on a translucent track under the pointer.
func (tahoeEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := tahoeColors(l)
	bar := macSnap(p.Bar)
	if bar.Empty() || st.Disabled {
		return
	}
	hover := st.Hovered || st.Pressed != ScrollNone
	px := macPx(l)
	if hover {
		ctx.DrawRect(bar, paintengine2d.Fill(c.scrollTrack))
		if vertical {
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, px, bar.Dy()), paintengine2d.Fill(c.scrollTrackEdge))
		} else {
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), px), paintengine2d.Fill(c.scrollTrackEdge))
		}
	}
	if p.Thumb.Empty() {
		return
	}
	w := snap(l.S(7))
	if hover {
		w = snap(l.S(11))
	}
	edge := snap(l.S(2))
	col := c.scroller
	if hover && (st.Pressed == ScrollThumbPart || st.Hot == ScrollThumbPart) {
		col = c.scrollerHot
	}
	t := macSnap(p.Thumb)
	if vertical {
		t = paintengine2d.XYWH(bar.Max.X-edge-w, t.Min.Y, w, t.Dy())
	} else {
		t = paintengine2d.XYWH(t.Min.X, bar.Max.Y-edge-w, t.Dx(), w)
	}
	t = t.Intersect(bar)
	if t.Empty() {
		return
	}
	r := min(t.Dx(), t.Dy()) * 0.5
	macFill(ctx, t, r, paintengine2d.Fill(col))
	macRing(ctx, t, r, max(px*0.8, 0.8), paintengine2d.Fill(c.scrollerRim))
}

func (e tahoeEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered() || st.Pressed()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------------------------

// boxPane is the rounded box of group boxes and tab panes: 12px corners, a
// faint fill in a hairline (a raised one lifted towards the content).
func (c *tahoeSet) boxPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := tahoeRad(l, 12, b)
	fill := c.box
	if raised {
		fill = webA(c.content, 0.6)
	}
	macFill(ctx, b, r, paintengine2d.Fill(fill))
	macRing(ctx, b, r, macPx(l), paintengine2d.Fill(c.boxEdge))
}

func (tahoeEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	box := b
	if title != "" {
		th := snap(l.body.Height() + l.S(4))
		l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8), th), c.text, AlignStart, 0)
		box = paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+th), Max: b.Max}
	}
	c.boxPane(l, ctx, box, raised)
}

// tahoeTitleH is an in-app window's title bar: 34px at the 16px font.
func tahoeTitleH(l *Classic) float32 {
	return snap(max(l.S(34), l.body.Height()+l.S(8)))
}

func (tahoeEngine) WindowFrameInsets(l *Classic) Insets {
	px := macPx(l)
	return Insets{Top: tahoeTitleH(l), Right: px, Bottom: px, Left: px}
}

// tahoeLightRect is the rect of traffic light i (0 close, 1 minimise, 2 zoom):
// 14px lights 22px apart, the first 16px from the edge, inside the 16pt
// corner.
func tahoeLightRect(l *Classic, b paintengine2d.Rect, i int) paintengine2d.Rect {
	h := tahoeTitleH(l)
	d := snap(min(l.S(14), h-l.S(8)))
	x := b.Min.X + l.S(16) + float32(i)*l.S(22)
	return paintengine2d.XYWH(snap(x), snap(b.Min.Y+(h-d)*0.5), d, d)
}

// WindowCloseRect is the red traffic light, first on the left.
func (tahoeEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = macSnap(b)
	if b.Dx() < l.S(90) || b.Dy() < tahoeTitleH(l) {
		return paintengine2d.Rect{}
	}
	return tahoeLightRect(l, b, 0)
}

// DrawWindowFrame is a Tahoe title-bar window: 16pt corners all round, the
// title bar one with the window (no band, no line), the traffic lights on
// the left and the title centred.
func (e tahoeEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Empty() {
		return
	}
	px := macPx(l)
	h := tahoeTitleH(l)
	r := min(l.rx(18), b.Dx()*0.3, b.Dy()*0.3)
	macFill(ctx, b, r, paintengine2d.Fill(c.win))
	macRing(ctx, b, r, px, paintengine2d.Fill(c.winEdge))
	if b.Dx() < l.S(90) || b.Dy() < h {
		return
	}
	for i := 0; i < 3; i++ {
		lb := tahoeLightRect(l, b, i)
		fill, rim := c.lights[i], c.lightEdges[i]
		if !st.Active || (i == 0 && !st.CanClose) {
			fill, rim = c.lightOff, c.lightOffEdge
		}
		if i == 0 && st.ClosePress && st.CanClose {
			fill = mdOver(fill, paintengine2d.RGB(0, 0, 0), 0.2)
		}
		ctr := lb.Center()
		ctx.DrawCircle(ctr, lb.Dx()*0.5, paintengine2d.Fill(rim))
		ctx.DrawCircle(ctr, lb.Dx()*0.5-px*0.75, paintengine2d.Fill(fill))
		if i == 0 && st.CanClose && (st.CloseHot || st.ClosePress) {
			DrawCross(ctx, lb.Inset(lb.Dx()*0.3), c.glyph, max(l.S(1.3), 1))
		}
	}
	if title == "" {
		return
	}
	col := c.titleText
	if !st.Active {
		col = c.titleOff
	}
	left := tahoeLightRect(l, b, 2).Max.X + l.S(10)
	f := l.body
	tw := f.Advance(title)
	x := max(b.Min.X+(b.Dx()-tw)*0.5, left)
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-l.S(10)-x, h), col, AlignStart, 0)
}

func (tahoeEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(tahoeColors(l).win))
}

// tahoeShadow is a floating layer's shadow: Liquid Glass's soft, adaptive
// shadow under menus, a small one under help tags, a deep one round
// windows.
func tahoeShadow(l *Classic, kind PopupKind) baseShadowSpec {
	c := tahoeColors(l)
	k := l.P("shadow", 1)
	if c.dark {
		k *= 2
	}
	a := func(v float32) paintengine2d.Color { return paintengine2d.RGBA(0, 0, 0, min(v*k, 0.9)) }
	switch kind {
	case PopupTooltip:
		return baseShadowSpec{col: a(0.14), r: l.rx(8), dy: l.S(2), blur: l.S(8)}
	case PopupDialog:
		return baseShadowSpec{col: a(0.3), r: l.rx(18), dy: l.S(16), blur: l.S(48)}
	}
	return baseShadowSpec{col: a(0.18), r: l.rx(16), dy: l.S(6), blur: l.S(28)}
}

func (tahoeEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	if l.P("shadow", 1) <= 0 {
		return Insets{}
	}
	sp := tahoeShadow(l, kind)
	return ShadowReach(0, sp.dy, sp.blur, 0)
}

// DrawPopupShadow paints the shadow round b only: the glass over b blurs
// the content under it, not its own shadow.
func (tahoeEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if l.P("shadow", 1) <= 0 {
		return
	}
	sp := tahoeShadow(l, kind)
	tahoeShadowAround(ctx, b, sp.r, sp.col, sp.dy, sp.blur)
}

// tahoeViewPad is the room inside a view's frame: the focus halo's margin,
// then the card's edge and the padding that keeps rows off its rounded
// corners (and, in a sidebar, inside the floating panel).
func tahoeViewPad(l *Classic) float32 { return macHaloW(l) + snap(l.S(5)) }

// ViewFrameInsets: rows start inside the rounded card (or the sidebar's
// glass panel).
func (tahoeEngine) ViewFrameInsets(l *Classic) Insets {
	v := tahoeViewPad(l)
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// tahoeSideInset is how far the floating sidebar panel keeps from the
// view's edges.
func tahoeSideInset(l *Classic) float32 { return snap(l.S(6)) }

// DrawViewFrame is a list, tree or table: a rounded card in the content
// colour inside the focus halo's margin, ringed by the halo while the view
// has focus. A sidebar is the window with the floating glass panel on it.
func (tahoeEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Empty() {
		return
	}
	if st.Sidebar() {
		if !GlassBehind(l) {
			// Without real glass the panel is laid over the window colour
			// (pre-flattened); with it, over the blurred desktop itself.
			ctx.DrawRect(b, paintengine2d.Fill(c.win))
		}
		p := b.Inset(tahoeSideInset(l))
		if p.Dx() < 8 || p.Dy() < 8 {
			return
		}
		r := tahoeRad(l, 18, p)
		if !st.Backdrop() {
			c.glassShadow(l, ctx, p, r, tahoeSideInset(l))
		}
		tint := c.glass
		if st.Backdrop() {
			tint = c.sideOff
		}
		c.paintGlass(l, ctx, p, r, tint, c.glassEdge, false)
		return
	}
	w := macHaloW(l)
	if b.Dx() < 3*w || b.Dy() < 3*w {
		return
	}
	card := b.Inset(w)
	r := tahoeRad(l, 10, card)
	macFill(ctx, card, r, paintengine2d.Fill(c.content))
	macRing(ctx, card, r, macPx(l), paintengine2d.Fill(c.sep))
	if st.Focused() && !st.Disabled() {
		macRing(ctx, b, r+w, w, paintengine2d.Fill(c.focus))
	}
}

// ViewBackground: a sidebar's rows sit on its glass panel, other views on
// the content colour.
func (e tahoeEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	c := tahoeColors(l)
	switch {
	case st.Sidebar() && st.Backdrop():
		return c.flat(c.sideOff)
	case st.Sidebar() && GlassBehind(l):
		// The rows sit on the glass panel itself: nothing opaque under
		// them, or the blur would stop at the first row.
		return paintengine2d.Transparent
	case st.Sidebar():
		return c.side
	}
	return c.content
}

// DrawTabPane is the rounded box under the segmented tabs, from half way
// down the tab strip, so the segments straddle its top edge.
func (tahoeEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := tahoeColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	top := snap(b.Min.Y + l.metrics.TabH*0.5)
	if top >= b.Max.Y-l.S(4) {
		return
	}
	c.boxPane(l, ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, false)
}

// ---- controls -----------------------------------------------------------------------------------

func (e tahoeEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := tahoeColors(l)
	f := macFace(l, b)
	fg := c.button(l, ctx, f, st, true)
	l.drawFittedText(ctx, l.body, label, f, fg, AlignCenter, l.S(14))
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, f, b, c.ctlR(l, f))
	}
}

// tool paints a tool button and returns its glyph and label colours. In a
// tool bar (StateAutoRaise) the button sits on its group's glass: a capsule
// wash under the pointer, darker pressed, a latched one washed with its
// glyph in the accent. A free tool button is a push button, the accent while
// latched.
func (c *tahoeSet) tool(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) (glyph, label paintengine2d.Color) {
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return c.text, c.text
	}
	dis := st.Disabled()
	if !st.AutoRaise() {
		f := macFace(l, b)
		r := c.ctlR(l, f)
		on := st.Checked() && !dis
		if on {
			fill := c.accent
			if st.Pressed() {
				fill = c.accentPress
			}
			macFill(ctx, f, r, paintengine2d.Fill(fill))
			return c.onAccent, c.onAccent
		}
		fg := c.button(l, ctx, f, st&^StatePrimary, true)
		return fg, fg
	}
	if dis {
		return c.text3, c.text3
	}
	h := b.Inset(snap(l.S(3)))
	r := min(h.Dx(), h.Dy()) * 0.5
	fg := c.text
	switch {
	case st.Pressed():
		macFill(ctx, h, r, paintengine2d.Fill(c.washPress))
	case st.Checked():
		macFill(ctx, h, r, paintengine2d.Fill(c.wash))
		fg = c.toolOnText
	case st.Hovered():
		macFill(ctx, h, r, paintengine2d.Fill(c.wash))
	}
	if st.Checked() {
		return fg, fg
	}
	return fg, c.text
}

func (e tahoeEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := tahoeColors(l)
	fg, tc := c.tool(l, ctx, b, st)
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib.Intersect(b), icon, fg)
		x = ib.Max.X + iconGap
	}
	switch {
	case label == "":
	case icon == IconNone && !st.AutoRaise():
		l.drawFittedText(ctx, l.body, label, macFace(l, b), tc, AlignCenter, l.S(4))
	default:
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x-pad*0.5, b.Dy()), tc, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		f := macSnap(b).Inset(macHaloW(l))
		r := min(f.Dx(), f.Dy()) * 0.5
		if !st.AutoRaise() {
			f = macFace(l, b)
			r = c.ctlR(l, f)
		}
		c.halo(l, ctx, f, b, r)
	}
}

// DrawToolGroup is a group of tool bar buttons on one glass capsule
// (ToolGroupEngine), over its soft shadow.
func (tahoeEngine) DrawToolGroup(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, n int) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Dx() < 8 || b.Dy() < 8 {
		return
	}
	r := b.Dy() * 0.5
	if l.square() {
		r = 0
	}
	c.glassShadow(l, ctx, b, r, snap(l.S(3)))
	c.paintGlass(l, ctx, b, r, c.glass, c.glassEdge, false)
}

// tahoeToggleLayout is macToggleLayout with the cell no taller than b.
func tahoeToggleLayout(l *Classic, b paintengine2d.Rect, side float32) (cell, lb paintengine2d.Rect) {
	cell, lb = macToggleLayout(l, b, min(side, b.Dy()))
	return cell.Intersect(macSnap(b)), lb
}

func (e tahoeEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := tahoeColors(l)
	cell, lb := tahoeToggleLayout(l, b, l.metrics.Checkbox)
	e.CheckIndicator(l, ctx, cell, st, checked)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		s := tahoeCheckS(l, cell)
		g := macCentered(cell, s, s)
		c.halo(l, ctx, g, b, tahoeRad(l, 5.5, g))
	}
}

func (e tahoeEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := tahoeColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	cell, lb := tahoeToggleLayout(l, b, side)
	e.RadioIndicator(l, ctx, cell, st, selected)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		d := tahoeCheckS(l, cell)
		g := macCentered(cell, d, d)
		c.halo(l, ctx, g, b, d*0.5)
	}
}

func (c *tahoeSet) toggleLabel(l *Classic, ctx *paintengine2d.Context, lb paintengine2d.Rect, st ControlState, label string) {
	if label == "" || lb.Dx() <= 0 {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

// DrawSwitch is the Tahoe switch: iOS 26's wide capsule (the accent when
// on) with a white capsule knob, which turns to glass while held.
func (e tahoeEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := tahoeColors(l)
	sw, sh := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	cell := macSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-sh)*0.5, min(sw, b.Dx()), sh))
	w := macHaloW(l)
	if cell.Dx() < 4*w || cell.Dy() < 3*w {
		return
	}
	track := cell.Inset(w)
	r := track.Dy() * 0.5
	if l.square() {
		r = 0
	}
	dis := st.Disabled()
	fill := c.switchOff
	switch {
	case on && dis:
		fill = webA(c.accent, 0.4)
	case on && st.Backdrop():
		fill = c.selOff
	case on:
		fill = c.accent
	case dis:
		fill = webA(c.switchOff, 0.5)
	}
	macFill(ctx, track, r, paintengine2d.Fill(fill))
	in := snap(l.S(2))
	kh := track.Dy() - 2*in
	kw := min(snap(kh*1.6), track.Dx()*0.62)
	kx := track.Min.X + in
	if on {
		kx = track.Max.X - in - kw
	}
	ctx.Save()
	ctx.ClipRect(cell)
	c.knobAt(l, ctx, paintengine2d.XYWH(kx, track.Min.Y+in, kw, kh), st)
	ctx.Restore()
	lb := paintengine2d.XYWH(cell.Max.X-w+l.S(8), b.Min.Y, b.Max.X-(cell.Max.X-w+l.S(8)), b.Dy())
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !dis {
		c.halo(l, ctx, track, b, r)
	}
}

// tahoeKnob is the slider knob's size: a 26×16.5pt capsule.
func tahoeKnob(l *Classic, b paintengine2d.Rect) (w, h float32) {
	hw := macHaloW(l)
	h = min(snap(l.S(20)), b.Dy()-2*hw)
	w = min(snap(l.S(32)), b.Dx()*0.4)
	return w, h
}

// SliderTravel is where the capsule knob's centre runs.
func (tahoeEngine) SliderTravel(l *Classic, b paintengine2d.Rect) (x0, x1 float32) {
	b = macSnap(b)
	kw, _ := tahoeKnob(l, b)
	hw := macHaloW(l)
	return b.Min.X + hw + kw*0.5, b.Max.X - hw - kw*0.5
}

// DrawSlider is NSSlider in Tahoe: a 5pt track, filled with the accent up
// to the white capsule knob (glass while dragged).
func (e tahoeEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := tahoeColors(l)
	b = macSnap(b)
	t = clamp1(t)
	kw, kh := tahoeKnob(l, b)
	hw := macHaloW(l)
	if kh < 6 || b.Dx() < kw+2*hw {
		return
	}
	th := max(snap(l.S(6)), 2)
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	x0, x1 := e.SliderTravel(l, b)
	tx := x0 + (x1-x0)*t
	track := paintengine2d.XYWH(b.Min.X+hw, cy-th*0.5, b.Dx()-2*hw, th)
	macFill(ctx, track, th*0.5, paintengine2d.Fill(c.track))
	fill := c.accent
	switch {
	case st.Disabled():
		fill = c.text3
	case st.Backdrop():
		fill = c.flat(c.selOff)
	}
	if tx > track.Min.X {
		macFill(ctx, paintengine2d.XYWH(track.Min.X, track.Min.Y, tx-track.Min.X, th), th*0.5, paintengine2d.Fill(fill))
	}
	knob := paintengine2d.XYWH(tx-kw*0.5, cy-kh*0.5, kw, kh)
	ctx.Save()
	ctx.ClipRect(b)
	c.knobAt(l, ctx, knob, st)
	ctx.Restore()
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, knob, b, kh*0.5)
	}
}

// DrawProgressBar is a 5pt capsule track and the accent's fill; the busy bar
// sweeps a soft accent segment across.
func (e tahoeEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := tahoeColors(l)
	b = macSnap(b)
	h := min(snap(l.S(6)), b.Dy())
	if b.Dx() < 4 || h < 2 {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := h * 0.5
	macFill(ctx, bar, r, paintengine2d.Fill(c.progress))
	fill, busy := c.accent, c.busy
	if st.Disabled() {
		fill, busy = c.text3, c.busyDim
	}
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRoundRect(bar, r, r)
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.35
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, bar.Min.Y, span, h)
		ctx.DrawRect(seg, HGradient(seg, busy...))
		return
	}
	if w := snap(bar.Dx() * clamp1(t)); w >= 1 {
		macFill(ctx, paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, paintengine2d.Fill(fill))
	}
}

func (e tahoeEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := tahoeColors(l)
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	f := macFace(l, b)
	c.fieldFace(l, ctx, f, st)
	l.baseDrawTextField(ctx, f, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, f, b, c.fieldR(l, f))
	}
}

// DrawComboBox is NSPopUpButton in Tahoe: the grey push button face with
// the value and, at the right, the up-and-down chevron in the label colour
// (the blue cap is gone).
func (e tahoeEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := tahoeColors(l)
	f := macFace(l, b)
	if f.Dx() < 8 || f.Dy() < 6 {
		return
	}
	bst := st &^ StatePrimary
	if open {
		bst |= StatePressed
	}
	fg := c.button(l, ctx, f, bst, false)
	s := snap(min(f.Dy()-l.S(8), l.S(16)))
	cap := paintengine2d.XYWH(f.Max.X-s-snap(l.S(6)), snap((f.Min.Y+f.Max.Y-s)*0.5), s, s)
	macUpDown(l, ctx, cap, fg)
	pad := l.S(11)
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(f.Min.X+pad, f.Min.Y, cap.Min.X-f.Min.X-pad-l.S(2), f.Dy()), fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.halo(l, ctx, f, b, c.ctlR(l, f))
	}
}

// DrawSpinner is NSStepper standing beside its field: a narrow grey capsule
// split in two, a chevron in each half.
func (e tahoeEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	w := snap(min(l.S(17), b.Dx()-l.S(4)))
	h := snap(min(l.S(28), b.Dy()-2*macHaloW(l)))
	if w < 6 || h < 8 {
		return
	}
	f := paintengine2d.XYWH(b.Max.X-w-snap(max(l.S(1), (b.Dx()-w-l.S(4))*0.5)), snap((b.Min.Y+b.Max.Y-h)*0.5), w, h)
	r := w * 0.5
	if l.square() {
		r = 0
	}
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(b)
	fill := c.btn
	if st.Disabled() {
		fill = webA(c.btn, 0.5)
	}
	macFill(ctx, f, r, paintengine2d.Fill(fill))
	mid := snap((f.Min.Y + f.Max.Y) * 0.5)
	px := macPx(l)
	up := paintengine2d.XYWH(f.Min.X, f.Min.Y, w, mid-f.Min.Y)
	dn := paintengine2d.XYWH(f.Min.X, mid, w, f.Max.Y-mid)
	if upPress && !st.Disabled() {
		ctx.DrawPath(RoundRectPath(up, r, r, 0, 0), paintengine2d.Fill(c.wash))
	}
	if downPress && !st.Disabled() {
		ctx.DrawPath(RoundRectPath(dn, 0, 0, r, r), paintengine2d.Fill(c.wash))
	}
	ctx.DrawRect(paintengine2d.XYWH(f.Min.X+px*3, mid, w-px*6, px), paintengine2d.Fill(c.sep))
	col := c.text
	if st.Disabled() {
		col = c.text3
	}
	macChevron(l, ctx, up.Translate(paintengine2d.Pt(0, px)), DirUp, col, l.S(8))
	macChevron(l, ctx, dn.Translate(paintengine2d.Pt(0, -px)), DirDown, col, l.S(8))
}

// DrawTabBar paints what the segmented tabs straddle: the window above
// their middle, the pane's box from the middle down.
func (e tahoeEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := tahoeColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	mid := snap(b.Min.Y + b.Dy()*0.5)
	if mid >= b.Max.Y {
		return
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, mid), Max: b.Max})
	c.boxPane(l, ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, mid), Max: paintengine2d.Pt(b.Max.X, b.Max.Y+l.S(24))}, false)
	ctx.Restore()
}

// DrawTab is one segment of the segmented tabs: a grey track rounded at the
// strip's ends, the selected segment a raised knob, dividers between the
// others.
func (e tahoeEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	h := snap(min(l.S(28), b.Dy()-l.S(4)))
	if b.Dx() < 6 || h < 8 {
		return
	}
	seg := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := c.ctlR(l, seg)
	rl, rr := float32(0), float32(0)
	if st.First() {
		rl = r
	}
	if st.Last() {
		rr = r
	}
	px := macPx(l)
	ctx.DrawPath(RoundRectPath(seg, rl, rr, rr, rl), paintengine2d.Fill(c.segTrack))
	fg := c.text
	if selected {
		knob := seg.Inset(snap(l.S(2)))
		kr := max(min(r-l.S(2), knob.Dy()*0.5), 0)
		fill := c.segKnob
		if st.Disabled() {
			fill = Mix(fill, c.win, 0.5)
		}
		ctx.Save()
		ctx.ClipRect(b)
		DropShadow(ctx, knob, kr, paintengine2d.RGBA(0, 0, 0, 0.16), 0, px*0.5, l.S(2), 0)
		ctx.Restore()
		macFill(ctx, knob, kr, paintengine2d.Fill(fill))
	} else {
		if st.Pressed() && !st.Disabled() {
			ctx.DrawPath(RoundRectPath(seg, rl, rr, rr, rl), paintengine2d.Fill(c.wash))
		}
		if !st.Last() {
			ctx.DrawRect(paintengine2d.XYWH(seg.Max.X-px, seg.Min.Y+snap(h*0.28), px, snap(h*0.44)), paintengine2d.Fill(c.sep))
		}
	}
	if st.Disabled() {
		fg = c.text3
	}
	l.drawFittedText(ctx, l.body, label, seg, fg, AlignCenter, l.S(10))
	if st.Focused() && selected && !st.Disabled() {
		f := seg.Inset(macHaloW(l) * 0.5)
		ctx.Save()
		ctx.ClipRect(b)
		macRing(ctx, f, max(r-l.S(1), 0), macHaloW(l), paintengine2d.Fill(c.focus))
		ctx.Restore()
	}
}

// DrawPanel: a raised panel is the rounded box; a flat one the window.
func (tahoeEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := tahoeColors(l)
	if raised {
		c.boxPane(l, ctx, macSnap(b), true)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
}

// DrawMenuBar: an in-window menu bar is the window, as Tahoe's transparent
// menu bar shows it.
func (tahoeEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(macSnap(b), paintengine2d.Fill(tahoeColors(l).win))
}

func (e tahoeEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	hb := b.Inset(snap(l.S(2)))
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.text3
	case open || st.Pressed():
		e.MenuHighlight(l, ctx, hb, true)
	case st.Hovered():
		macFill(ctx, hb, tahoeRad(l, 8, hb), paintengine2d.Fill(c.wash))
	}
	l.drawLabeled(ctx, l.body, label, -1, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		c.halo(l, ctx, b.Inset(macHaloW(l)), b, tahoeRad(l, 8, b.Inset(macHaloW(l))))
	}
}

// menuR is a menu's corner: 13pt.
func (c *tahoeSet) menuR(l *Classic, b paintengine2d.Rect) float32 {
	return min(tahoeRad(l, 16, b), b.Dy()*0.25)
}

// DrawMenuFrame is a glass menu: the content behind blurred under the
// translucent tint, 13pt corners, the rim and hairline.
func (tahoeEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	c.paintGlass(l, ctx, b, c.menuR(l, b), c.menu, c.menuEdge, true)
}

func (e tahoeEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := tahoeColors(l)
	ch := MenuChromeFor(l)
	px := macPx(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := b.Min.X-ch.PadL+l.S(14), b.Max.X+ch.PadR-l.S(14)
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, px), paintengine2d.Fill(c.menuSep))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		k := min(snap(l.S(6)), ch.PadL)
		e.MenuHighlight(l, ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy()), false)
	}
	fg, sc := c.text, c.text2
	switch {
	case st.Disabled():
		fg, sc = c.text3, c.text3
	case hot:
		fg, sc = c.onAccent, c.onAccent
	}
	macGutter(l, ctx, b, ch, row, fg)
	font := l.body
	ty := b.Min.Y + (b.Dy()-font.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow, b.Dy())
		e.Arrow(l, ctx, ab, DirRight, fg)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := font.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		font.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), sc)
		right = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if right < lx {
		right = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	font.Draw(ctx, row.Label, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// rowBox is where a row's selection paints inside the row b: a rounded box
// 4px in from the sides of a view's card (8px into a sidebar's panel) and a
// pixel from its top and bottom.
func (c *tahoeSet) rowBox(l *Classic, b paintengine2d.Rect, sidebar bool) (paintengine2d.Rect, float32) {
	b = macSnap(b)
	in := snap(l.S(4))
	if sidebar {
		in = snap(l.S(8))
	}
	v := snap(l.S(1))
	if b.Dx() < 4*in || b.Dy() < 6*v {
		return b, 0
	}
	box := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+in, b.Min.Y+v), Max: paintengine2d.Pt(b.Max.X-in, b.Max.Y-v)}
	return box, tahoeRad(l, 8, box)
}

// row paints a row's selection into box (corner r) and returns its text
// colour: the accent with white text while the view has focus, the grey of
// an unemphasized selection otherwise, as AppKit does.
func (c *tahoeSet) row(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, r float32) paintengine2d.Color {
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	if !st.Checked() {
		return fg
	}
	fill, text := c.sel, c.onAccent
	switch {
	case st.Disabled():
		fill, text = c.selOff, c.text3
	case st.Inactive() || st.Backdrop():
		fill, text = c.selOff, c.selOffText
	}
	macFill(ctx, box, r, paintengine2d.Fill(fill))
	return text
}

// sideRow paints a sidebar row's selection and returns its label colour:
// Tahoe's grey rounded row with the label in the accent (the text colour in
// an inactive window).
func (c *tahoeSet) sideRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	if !st.Checked() {
		return fg
	}
	box, r := c.rowBox(l, b, true)
	macFill(ctx, box, r, paintengine2d.Fill(c.sideSel))
	if st.Backdrop() || st.Disabled() {
		return fg
	}
	return c.sideText
}

// tahoeSidePad is where a sidebar row's label starts: 10px into its box.
func tahoeSidePad(l *Classic) float32 { return snap(l.S(8)) + l.S(10) }

func (e tahoeEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := tahoeColors(l)
	if st.Sidebar() {
		fg := c.sideRow(l, ctx, b, st)
		pad := tahoeSidePad(l)
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-2*pad, b.Dy()), fg, AlignStart, 0)
		return
	}
	box, r := c.rowBox(l, b, false)
	fg := c.row(l, ctx, box, st, r)
	pad := l.S(14)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-l.S(8), b.Dy()), fg, AlignStart, 0)
}

func (e tahoeEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := tahoeColors(l)
	side := st.Sidebar()
	var fg paintengine2d.Color
	if side {
		fg = c.sideRow(l, ctx, b, st)
	} else {
		box, r := c.rowBox(l, b, false)
		fg = c.row(l, ctx, box, st, r)
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(10) + float32(depth)*indent
	if side {
		x = b.Min.X + tahoeSidePad(l) - l.S(6) + float32(depth)*indent
	}
	if !leaf {
		col := c.text2
		switch {
		case st.Checked() && (fg == c.onAccent || fg == c.sideText):
			col = fg
		case st.ExpanderHot():
			col = c.text
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, col)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(3)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(6), b.Dy()), fg, AlignStart, 0)
}

// DrawTableHeader is the list header: the content colour, the label in the
// secondary colour (the label colour when sorted, with its chevron), short
// hairlines between columns and one under the header.
func (e tahoeEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Empty() {
		return
	}
	px := macPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.header))
	if st.Pressed() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.wash))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.sep))
	if !st.Last() {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, b.Min.Y+snap(b.Dy()*0.25), px, snap(b.Dy()*0.5)), paintengine2d.Fill(c.sep))
	}
	fg := c.headerText
	switch {
	case st.Disabled():
		fg = c.text3
	case sorted:
		fg = c.text
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy()), dir, c.text2)
	}
	pad := l.tableCellPad(b.Dx()-aw, l.body.Advance(label))
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-aw-l.S(4), b.Dy()), fg, AlignStart, 0)
}

// DrawTableCell: rounded alternating stripes and one rounded selection box
// across the row, inset from the card's sides (through CellSpan).
func (e tahoeEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := tahoeColors(l)
	cell := macSnap(b)
	ctx.Save()
	ctx.ClipRect(cell)
	box, r := c.rowBox(l, CellSpan(cell, st, l.S(24)), false)
	if st.Alternate() && !st.Checked() {
		macFill(ctx, box, r, paintengine2d.Fill(c.alt))
	}
	fg := c.row(l, ctx, box, st, r)
	ctx.Restore()
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
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

// DrawToolBar: Tahoe's tool bar has no band of its own; its buttons float
// over the window on glass (DrawToolGroup).
func (tahoeEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(macSnap(b), paintengine2d.Fill(tahoeColors(l).win))
}

func (tahoeEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := tahoeColors(l)
	b = macSnap(b)
	px := macPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), px), paintengine2d.Fill(c.sep))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(12), b.Min.Y, slot-l.S(16), b.Dy()), c.text2, AlignStart, 0)
	}
}

func (tahoeEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := tahoeColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(10)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.text2, AlignStart, 0)
	}
}

// DrawAccordionHeader is a disclosure section: the chevron and the bold
// title, a rounded wash while pressed.
func (e tahoeEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	if st.Pressed() && !st.Disabled() {
		macFill(ctx, b, tahoeRad(l, 8, b), paintengine2d.Fill(c.wash))
	}
	col := c.text2
	if st.Hovered() && !st.Disabled() {
		col = c.text
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, col)
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(22), b.Min.Y, b.Dx()-l.S(26), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		f := b.Inset(macHaloW(l))
		c.halo(l, ctx, f, b, tahoeRad(l, 8, f))
	}
}

func (tahoeEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := tahoeColors(l)
	px := macPx(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(c.sep))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(c.sep))
}

// DrawSplitter is the 1pt split-view divider.
func (e tahoeEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	e.DrawSeparator(l, ctx, b, vertical)
}

// DrawTooltip is the help tag: a light rounded tag in a hairline.
func (tahoeEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := tahoeColors(l)
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := tahoeRad(l, 8, b)
	macFill(ctx, b, r, paintengine2d.Fill(c.tip))
	macRing(ctx, b, r, macPx(l), paintengine2d.Fill(c.tipEdge))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(8)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- accent -------------------------------------------------------------------------------------

// tahoeAccentKeys are the shades of the accent Tahoe paints with: the
// accent (default buttons, checks, switches, sliders, progress, the menu
// highlight), its pressed shade, its tint, the focused list selection and
// the selected sidebar label, the focus halo and a latched tool button's
// glyph; the text highlight is the accent washed towards the content.
var tahoeAccentKeys = [...]string{"accent", "accentPress", "tint", "sel", "sideText", "focus", "menuHi", "toolOnText"}

// Accented is the accent colour of System Settings in Tahoe (Blue #0088FF,
// #0091FF dark, by default): every shade makes the step from the accent the
// pack's own makes from its blue. The unemphasized grey selection stays.
// White on the accent turns dark for an accent too pale to carry it at 2.2:1.
func (tahoeEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	sc := tahoeLight
	if accentDark(tok) {
		sc = tahoeDark
	}
	own := func(k string) paintengine2d.Color { return accentX(tok, k, Hex(sc[k])) }
	ref := own("accent")
	tok = CloneTokenMaps(tok)
	for _, k := range tahoeAccentKeys {
		tok.Extra[k] = accentShift(accent, ref, own(k))
	}
	tok.Extra["textSel"] = accentWash(accent, ref, own("textSel"))
	on := own("onAccent")
	for _, k := range [...]string{"accent", "sel", "menuHi"} {
		if ContrastRatio(on, tok.Extra[k]) < 2.2 {
			on = Hex(tahoeLight["text"])
			break
		}
	}
	tok.Extra["onAccent"] = on
	p := &tok.Palette
	a := tok.Extra["accent"]
	p.Accent, p.Focus, p.AccentHover, p.AccentPress = a, a, Shade(a, 0.1), tok.Extra["accentPress"]
	p.TextOnAccent, p.Selection = on, tok.Extra["textSel"]
	p.MenuHover, p.MenuHoverBorder = tok.Extra["menuHi"], tok.Extra["menuHi"]
	if c, ok := tok.Extra["selectionText"]; ok {
		tok.Extra["selectionText"] = ReadableOn(p.Selection, 4.5, c, p.Text, Hex("#ffffff"))
	}
	macChrome(&tok, tok.Extra["sel"])
	tok.Pressed = ChromeState{} // the resolver's, from the new palette
	return tok.Resolve()
}

// ---- packs --------------------------------------------------------------------------------------

func tahoePack(name, label, summary string, fam ThemeName, sc macScheme) ThemePack {
	h := func(k string) paintengine2d.Color { return Hex(sc[k]) }
	win, content := h("win"), h("content")
	over := func(c paintengine2d.Color) paintengine2d.Color { return webOver(win, c) }
	accent := h("accent")
	pal := Palette{
		Background: win, Surface: win, SurfaceAlt: over(h("box")),
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.3),
		Border:  over(h("fieldEdge")), Divider: over(h("sep")),
		Text: h("text"), TextMuted: h("text2"), TextOnAccent: h("onAccent"),
		Accent: accent, AccentHover: Shade(accent, 0.1), AccentPress: h("accentPress"),
		Danger: h("red"), Success: h("green"), Warning: h("orange"),
		Track: over(h("track")), Thumb: over(h("scroller")),
		Field: content, FieldBorder: webOver(content, h("fieldEdge")),
		Focus: accent, Selection: h("textSel"),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.2), Highlight: h("wash"),
		MenuHover: h("menuHi"), MenuHoverBorder: h("menuHi"), MenuGutter: content,
		BevelLight: content, BevelDark: over(h("fieldEdge")),
	}
	tok := ThemeTokens{
		Engine:  "macos",
		Bevel:   BevelSoftShadow,
		Family:  fam,
		Palette: pal,
		Params:  map[string]float32{"era": 2},
		Extra:   map[string]paintengine2d.Color{"selectionText": h("text")},
	}
	if fam == ThemeDark {
		tok.Extra["selectionText"] = Hex("#ffffff")
	}
	macChrome(&tok, h("sel"))
	return ThemePack{
		Name: name, Label: label, Year: 2025, Lineage: "Mac OS", Summary: summary,
		Era: "macOS", Palette: fam, Tokens: tok,
	}
}

func tahoePacks() []ThemePack {
	return []ThemePack{
		tahoePack("tahoe", "macOS Tahoe",
			"macOS 26's Liquid Glass: grey rounded controls, glass menus and tool bar groups, a floating glass sidebar, the #0088FF blue.",
			ThemeLight, tahoeLight),
		tahoePack("tahoe-night", "macOS Tahoe Dark",
			"Tahoe's dark appearance: the same glass over charcoal windows and #1c1c1e content, the #0091FF blue.",
			ThemeDark, tahoeDark),
	}
}
