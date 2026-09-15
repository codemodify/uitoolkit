package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// macosEngine paints macOS after Aqua: the flat look of OS X Yosemite to
// macOS Mojave (2014) and the rounder one of macOS Big Sur (2020), light
// and dark. Colours and sizes are taken as numbers from Apple's Human
// Interface Guidelines, AppKit's resolved system colours and controls
// measured on screenshots; the drawing is this file's own.
//
// Yosemite to Mojave (param era 0):
//
//   - push buttons are white rounded rects (4pt corners at AppKit's 20pt
//     face) in a hairline that darkens toward the bottom, a soft line under
//     them; the default button is the blue gradient (#69B1FA to #0F80FF)
//     with white text and turns plain in an inactive window; any push
//     button turns the accent while pressed, as AppKit's did until macOS 12;
//   - check boxes and radios are the flat #3B99FC when on, a white tick or
//     dot on it;
//   - scroll bars are overlay scrollers (ScrollBarStyle.Transient): a 7pt
//     knob, black at 50% in a faint white outline, that floats over the
//     content while it scrolls and fades, widening to 11pt on a translucent
//     track under the pointer;
//   - tabs are a segmented control centred on the top edge of a rounded
//     box, the selected segment in the accent gradient;
//   - menus are vibrant: the content behind them blurred under a light
//     translucent tint, 4pt corners, the hot row a full-width band of the
//     selection blue lightened by the vibrancy;
//   - keyboard focus is a 3pt halo of the accent at 50% around the control,
//     following its shape;
//   - text fields are square; the stepper stands beside its field
//     (SpinBoxStyle{}); list selections are the accent while the list has
//     focus and grey when it does not; a latched textured tool button is the
//     dark grey segment with a white glyph.
//
// Big Sur and later (param era 1): larger radii (5.5pt at the 20pt face,
// 6pt menus, 10pt windows), faces lit by a soft shadow instead of a hard
// hairline, the default button's flatter accent, the white raised knob of
// the segmented tabs on a grey track, the unified toolbar with borderless
// tool buttons that show a rounded wash under the pointer, and list, tree
// and table selections drawn as a rounded row inset 10pt from the view's
// sides (a table row's cells join into one box through CellSpan); the
// menu's hot row is a rounded box inset 5pt.
//
// Pack data: every colour key of macYosemite can be overridden through
// "extra"; params "era" (0 Yosemite, 1 Big Sur) and "vibrancy" (0 turns the
// menus' blurred backdrop off).
type macosEngine struct{ BaseEngine }

func init() {
	RegisterEngine(macosEngine{})
	for _, p := range macosPacks() {
		RegisterPack(p)
	}
}

func (macosEngine) ID() string { return "macos" }

// DefaultMetrics are AppKit's regular control sizes at the toolkit's 16px
// UI font (AppKit uses 13pt): the 20pt push button face becomes a 24px face
// in a 30px cell whose margin holds the 3pt focus halo; 14pt check boxes
// in 20px cells; 22px menu rows; a 16pt overlay scroller region.
func (macosEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 3,
		Radius:    6, RadiusSmall: 4,
		ControlH: 30, FieldH: 30, ComboH: 30,
		Checkbox: 20, Radio: 22,
		MenuItemH: 22, MenuBarH: 24, TabH: 30, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 16, SliderH: 24, Thumb: 20,
		Scroll: 16, Pad: 14, FieldPad: 8, FocusWidth: 3, Border: 1,
		ToolBarH: 40, StatusBarH: 24, SpinnerW: 22, SwitchW: 46, SwitchH: 28,
	}
}

// ---- colour tables ------------------------------------------------------------------------------

// macScheme maps a colour key to "#rrggbb" or "#rrggbbaa".
type macScheme map[string]string

// macYosemite is OS X Yosemite's light appearance with the blue accent
// (measured from 1x screenshots of 10.10–10.13; the semantic colours are
// AppKit's).
var macYosemite = macScheme{
	// Window, content (text and view backgrounds), the unified title and
	// tool bar gradient and its separator.
	"win": "#ececec", "content": "#ffffff", "bar": "#e4e2e4", "bar2": "#cfcdcf", "barEdge": "#b1b0b1",
	// Labels: primary, secondary, tertiary (black at 85%, 50%, 26% over
	// the content), text on the accent.
	"text": "#262626", "text2": "#808080", "text3": "#bdbdbd", "onAccent": "#ffffff",
	// The flat accent (check boxes, radios, sliders, progress) and the
	// accent faces' gradient (the default button, the pop-up cap, the
	// selected segment): face, rim, pressed face.
	"accent": "#3b99fc", "accentTop": "#69b1fa", "accentBot": "#0f80ff", "accentEdge": "#5baaf9", "accentEdge2": "#046cfe",
	"accentPress": "#4e94fe", "accentPress2": "#0b5de4",
	// A checked box or selected radio: face (top, bottom) and rim.
	"check": "#3b99fc", "check2": "#3b99fc", "checkEdge": "#2e81fc",
	// Plain push buttons: face gradient, hairline top / bottom, the soft
	// line under the face.
	"btnTop": "#ffffff", "btnBot": "#fbfbfb", "btnEdge": "#c7c7c7", "btnEdge2": "#adadad", "btnShade": "#00000014", "btnText": "#262626",
	// Unchecked box / radio rim (top, bottom).
	"chkEdge": "#a4a4a4", "chkEdge2": "#b8b8b8",
	// Text field edge (top, sides and bottom).
	"fieldEdgeTop": "#cfcfcf", "fieldEdge": "#cfcfcf",
	// Selections: list (key, not key), text.
	"sel": "#0069d9", "selOff": "#dcdcdc", "selOffText": "#262626", "textSel": "#b3d7ff",
	// Focus halo, separators, the alternating row.
	"focus": "#3b99fc80", "sep": "#d5d5d5", "alt": "#f4f5f5",
	// Group boxes and the tab pane.
	"box": "#e3e3e3", "boxEdge": "#cbcbcb",
	// Menus: the tint over the blurred backdrop, edge, separator, the hot
	// row (the selection lightened by the vibrancy).
	"menu": "#f9f9f9e6", "menuEdge": "#00000024", "menuSep": "#e8e8e8", "menuHi": "#3382e7",
	// Help tags.
	"tip": "#f0f0f0", "tipEdge": "#cdcdcd", "tipText": "#262626",
	// The overlay scroller: knob, its outline, the expanded track and the
	// line on its content side.
	"knob": "#00000080", "knobHot": "#00000099", "knobRim": "#ffffff30", "track": "#fafafabf", "trackEdge": "#e7e7e7",
	// Slider and progress tracks, the knob and its hairline.
	"slider": "#b8b8b8", "progress": "#dbdbdb", "progress2": "#dbdbdb", "knobFace": "#ffffff", "knobEdge": "#00000038",
	// The switch track when off.
	"switchOff": "#dedede",
	// Title bar gradient, separator, the flat bar of an inactive window,
	// title text.
	"title": "#e4e2e4", "title2": "#cfcdcf", "titleEdge": "#b1b0b1", "titleOffBar": "#f6f6f6", "titleOffEdge": "#d1d1d1",
	"titleText": "#4d4d4d", "titleOff": "#a8a8a8",
	// Traffic lights and rims; the lights of an inactive window; the glyph.
	"close": "#ff5f57", "closeEdge": "#e2463f", "mini": "#ffbd2e", "miniEdge": "#e1a116",
	"zoom": "#28c940", "zoomEdge": "#12ac28", "lightOff": "#d0d0d0", "lightOffEdge": "#afafaf", "glyph": "#4d0000",
	// Window outline, table header.
	"winEdge": "#00000040", "header": "#ffffff", "headerEdge": "#cdcdcd",
	// Tool buttons: the wash under the pointer (Big Sur), pressed; the
	// latched textured segment and its glyph.
	"wash": "#0000000f", "washPress": "#0000001f", "toolOn": "#6d6c6d", "toolOnText": "#ffffff",
	// Status colours.
	"red": "#ff3b30", "green": "#28cd41", "orange": "#ff9500",
}

// macBigSur is macOS Big Sur's light appearance.
var macBigSur = macScheme{
	"win": "#ececec", "content": "#ffffff", "bar": "#f8f8f8", "bar2": "#f8f8f8", "barEdge": "#dadada",
	"text": "#262626", "text2": "#808080", "text3": "#bdbdbd", "onAccent": "#ffffff",
	"accent": "#007aff", "accentTop": "#1c93ff", "accentBot": "#007aff", "accentEdge": "#0000000f", "accentEdge2": "#0000001f",
	"accentPress": "#0080ea", "accentPress2": "#006aeb",
	"check": "#1d94fe", "check2": "#007aff", "checkEdge": "#0000001a",
	"btnTop": "#ffffff", "btnBot": "#ffffff", "btnEdge": "#00000021", "btnEdge2": "#0000003b", "btnShade": "#0000000f", "btnText": "#262626",
	"chkEdge": "#bababa", "chkEdge2": "#cecece",
	"fieldEdgeTop": "#e4e4e4", "fieldEdge": "#bcbcbc",
	"sel": "#0063e1", "selOff": "#dcdcdc", "selOffText": "#262626", "textSel": "#b3d7ff",
	"focus": "#0067f480", "sep": "#e5e5e5", "alt": "#f4f5f5",
	"box": "#e5e5e5", "boxEdge": "#d3d3d3",
	"menu": "#ebebebe0", "menuEdge": "#0000001f", "menuSep": "#00000019", "menuHi": "#3382e7",
	"tip": "#f5f5f5f5", "tipEdge": "#00000021", "tipText": "#262626",
	"knob": "#00000080", "knobHot": "#00000099", "knobRim": "#ffffff30", "track": "#fafafabf", "trackEdge": "#e7e7e7",
	"slider": "#dfdfdf", "progress": "#e0e0e0", "progress2": "#eaeaea", "knobFace": "#ffffff", "knobEdge": "#0000002e",
	"switchOff": "#e4e4e4",
	"title":     "#f8f8f8", "title2": "#f8f8f8", "titleEdge": "#dadada", "titleOffBar": "#f6f6f6", "titleOffEdge": "#dadada",
	"titleText": "#4d4d4d", "titleOff": "#b0b0b0",
	"close": "#ff5f57", "closeEdge": "#e0443e", "mini": "#febc2e", "miniEdge": "#dea123",
	"zoom": "#28c840", "zoomEdge": "#1aab29", "lightOff": "#d0d0d0", "lightOffEdge": "#bcbcbc", "glyph": "#4d0000",
	"winEdge": "#00000033", "header": "#ffffff", "headerEdge": "#e5e5e5",
	"wash": "#0000000f", "washPress": "#0000001f", "toolOn": "#0000001f", "toolOnText": "#007aff",
	"red": "#ff3b30", "green": "#28cd41", "orange": "#ff9500",
}

// macBigSurDark is macOS Big Sur's dark appearance.
var macBigSurDark = macScheme{
	"win": "#323232", "content": "#1e1e1e", "bar": "#2d2d2d", "bar2": "#2d2d2d", "barEdge": "#000000",
	"text": "#e0e0e0", "text2": "#a3a3a3", "text3": "#656565", "onAccent": "#ffffff",
	"accent": "#007aff", "accentTop": "#1c93ff", "accentBot": "#007aff", "accentEdge": "#ffffff1a", "accentEdge2": "#0000001f",
	"accentPress": "#0080ea", "accentPress2": "#006aeb",
	"check": "#1d94fe", "check2": "#007aff", "checkEdge": "#00000033",
	"btnTop": "#6a6a6a", "btnBot": "#636363", "btnEdge": "#ffffff26", "btnEdge2": "#00000059", "btnShade": "#00000040", "btnText": "#e8e8e8",
	"chkEdge": "#ffffff26", "chkEdge2": "#00000040",
	"fieldEdgeTop": "#ffffff1f", "fieldEdge": "#ffffff2e",
	"sel": "#0058d0", "selOff": "#464646", "selOffText": "#e0e0e0", "textSel": "#3f638b",
	"focus": "#1aa9ff80", "sep": "#464646", "alt": "#292929",
	"box": "#2b2b2b", "boxEdge": "#262626",
	"menu": "#303033e6", "menuEdge": "#000000b3", "menuSep": "#ffffff26", "menuHi": "#1a69d5",
	"tip": "#363636f5", "tipEdge": "#000000b3", "tipText": "#e0e0e0",
	"knob": "#ffffff80", "knobHot": "#ffffff99", "knobRim": "#00000030", "track": "#c9c9c926", "trackEdge": "#ffffff1f",
	"slider": "#5a5a5a", "progress": "#4a4a4a", "progress2": "#555555", "knobFace": "#cfcfcf", "knobEdge": "#00000066",
	"switchOff": "#4a4a4a",
	"title":     "#2d2d2d", "title2": "#2d2d2d", "titleEdge": "#000000", "titleOffBar": "#2a2a2a", "titleOffEdge": "#000000",
	"titleText": "#e0e0e0", "titleOff": "#7a7a7a",
	"close": "#ff5f57", "closeEdge": "#e0443e", "mini": "#febc2e", "miniEdge": "#dea123",
	"zoom": "#28c840", "zoomEdge": "#1aab29", "lightOff": "#4d4d4d", "lightOffEdge": "#444444", "glyph": "#4d0000",
	"winEdge": "#000000cc", "header": "#1e1e1e", "headerEdge": "#464646",
	"wash": "#ffffff1a", "washPress": "#ffffff2e", "toolOn": "#ffffff2e", "toolOnText": "#3d9bff",
	"red": "#ff453a", "green": "#32d74b", "orange": "#ff9f0a",
}

// ---- resolved colour set ------------------------------------------------------------------------

// macSet is a look's resolved macOS colours (built once per look).
type macSet struct {
	bigSur, dark bool

	win, content, bar, barEdge            paintengine2d.Color
	text, text2, text3, onAccent          paintengine2d.Color
	accent, checkEdge, btnShade, btnText  paintengine2d.Color
	btnPress, fieldEdge, fieldEdgeTop     paintengine2d.Color
	sel, selOff, selOffText               paintengine2d.Color
	focus, sep, alt, box, boxEdge         paintengine2d.Color
	menu, menuEdge, menuSep, menuHi       paintengine2d.Color
	tip, tipEdge, tipText                 paintengine2d.Color
	knob, knobHot, knobRim, track         paintengine2d.Color
	trackEdge, slider, knobFace, knobEdge paintengine2d.Color
	switchOff, title, titleEdge           paintengine2d.Color
	titleOffBar, titleOffEdge             paintengine2d.Color
	titleText, titleOff                   paintengine2d.Color
	lights, lightEdges                    [3]paintengine2d.Color
	lightOff, lightOffEdge, glyph         paintengine2d.Color
	winEdge, header, headerEdge           paintengine2d.Color
	wash, washPress, toolOn, toolOnText   paintengine2d.Color
	// Text field faces (enabled, disabled); Big Sur's segment track and
	// its knob.
	fieldFill, fieldDis, segTrack, segKnob paintengine2d.Color

	// Gradient stops, built once: the accent face and its rim, pressed;
	// the checked face; the plain face and its rim; the unchecked rim; the
	// field edge; the bars; the title bar; the progress track; the busy
	// bar (accent, dim).
	accentStops, accentEdgeStops, accentPressStops, checkStops []paintengine2d.GradientStop
	btnStops, btnEdgeStops, chkEdgeStops, fieldStops           []paintengine2d.GradientStop
	barStops, titleStops, progressStops                        []paintengine2d.GradientStop
	busyStops, busyDimStops                                    []paintengine2d.GradientStop
	// The opacity of a Big Sur face's soft shadow.
	faceShadow float32
}

type macKey struct{}

func macColors(l *Classic) *macSet {
	return l.Memo(macKey{}, func() any { return macBuild(l) }).(*macSet)
}

// macSchemeOf picks the table of a look: Big Sur (dark or light) or
// Yosemite.
func macSchemeOf(l *Classic) (macScheme, bool, bool) {
	big := l.P("era", 0) >= 1
	dark := Luma(l.palette.Background) < 0.5
	switch {
	case dark:
		return macBigSurDark, true, true
	case big:
		return macBigSur, true, false
	}
	return macYosemite, false, false
}

func macBuild(l *Classic) *macSet {
	sc, big, dark := macSchemeOf(l)
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	c := &macSet{bigSur: big, dark: dark}
	c.win, c.content, c.bar, c.barEdge = col("win"), col("content"), col("bar"), col("barEdge")
	c.text, c.text2, c.text3, c.onAccent = col("text"), col("text2"), col("text3"), col("onAccent")
	c.accent, c.checkEdge, c.btnShade, c.btnText = col("accent"), col("checkEdge"), col("btnShade"), col("btnText")
	c.fieldEdge, c.fieldEdgeTop = col("fieldEdge"), col("fieldEdgeTop")
	c.sel, c.selOff, c.selOffText = col("sel"), col("selOff"), col("selOffText")
	c.focus, c.sep, c.alt = col("focus"), col("sep"), col("alt")
	c.box, c.boxEdge = col("box"), col("boxEdge")
	c.menu, c.menuEdge, c.menuSep, c.menuHi = col("menu"), col("menuEdge"), col("menuSep"), col("menuHi")
	c.tip, c.tipEdge, c.tipText = col("tip"), col("tipEdge"), col("tipText")
	c.knob, c.knobHot, c.knobRim, c.track, c.trackEdge = col("knob"), col("knobHot"), col("knobRim"), col("track"), col("trackEdge")
	c.slider, c.knobFace, c.knobEdge, c.switchOff = col("slider"), col("knobFace"), col("knobEdge"), col("switchOff")
	c.title, c.titleEdge = col("title"), col("titleEdge")
	c.titleOffBar, c.titleOffEdge = col("titleOffBar"), col("titleOffEdge")
	c.titleText, c.titleOff = col("titleText"), col("titleOff")
	c.lights = [3]paintengine2d.Color{col("close"), col("mini"), col("zoom")}
	c.lightEdges = [3]paintengine2d.Color{col("closeEdge"), col("miniEdge"), col("zoomEdge")}
	c.lightOff, c.lightOffEdge, c.glyph, c.winEdge = col("lightOff"), col("lightOffEdge"), col("glyph"), col("winEdge")
	c.header, c.headerEdge, c.wash, c.washPress = col("header"), col("headerEdge"), col("wash"), col("washPress")
	c.toolOn, c.toolOnText = col("toolOn"), col("toolOnText")
	two := func(a, b string) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col(a)), Stop(1, col(b))}
	}
	c.accentStops, c.accentEdgeStops, c.accentPressStops = two("accentTop", "accentBot"), two("accentEdge", "accentEdge2"), two("accentPress", "accentPress2")
	c.checkStops = two("check", "check2")
	c.btnStops, c.btnEdgeStops, c.chkEdgeStops = two("btnTop", "btnBot"), two("btnEdge", "btnEdge2"), two("chkEdge", "chkEdge2")
	c.btnPress = mdOver(col("btnBot"), c.text, 0.1)
	c.fieldStops = []paintengine2d.GradientStop{Stop(0, c.fieldEdgeTop), Stop(0.15, c.fieldEdge), Stop(1, c.fieldEdge)}
	c.barStops, c.titleStops, c.progressStops = two("bar", "bar2"), two("title", "title2"), two("progress", "progress2")
	busy := func(col paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col.WithAlpha(0)), Stop(0.5, col), Stop(1, col.WithAlpha(0))}
	}
	c.busyStops, c.busyDimStops = busy(c.accent), busy(c.text3)
	c.faceShadow = 0.2
	c.fieldFill = c.content
	c.segTrack = mdOver(c.win, c.text, 0.06)
	c.segKnob = paintengine2d.RGB(1, 1, 1)
	if dark {
		c.faceShadow = 0.5
		// Dark fields are the window lifted by 5% white.
		c.fieldFill = mdOver(c.win, paintengine2d.RGB(1, 1, 1), 0.05)
		c.segTrack = mdOver(c.win, paintengine2d.RGB(1, 1, 1), 0.1)
		c.segKnob = Hex("#636366")
	}
	c.fieldDis = mdOver(c.fieldFill, c.win, 0.5)
	return c
}

// ---- helpers ------------------------------------------------------------------------------------

// macPx is one device pixel: 1 at 1x, 2 at 2x.
func macPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
}

// macSnap puts b on whole pixels; edges round half down, so a snapped rect
// never covers a pixel whose centre lies outside b.
func macSnap(b paintengine2d.Rect) paintengine2d.Rect {
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

// macRing fills the lw-wide band just inside the round rect b with paint.
func macRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, paint paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	if b.Dx() <= 2*lw || b.Dy() <= 2*lw {
		ctx.DrawRoundRect(b, r, r, paint)
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(b, r, r)
	ri := max(r-lw, 0)
	p.AddRoundRect(b.Inset(lw), ri, ri)
	paint.Style = paintengine2d.StyleFill
	paint.FillRule = paintengine2d.FillEvenOdd
	ctx.DrawPath(p, paint)
}

// macFill fills a round rect with the radius clamped to the shape.
func macFill(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, paint paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	ctx.DrawRoundRect(b, r, r, paint)
}

// macCentered is a w×h rect centred in b, on whole pixels.
func macCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return macSnap(paintengine2d.XYWH((b.Min.X+b.Max.X-w)*0.5, (b.Min.Y+b.Max.Y-h)*0.5, w, h))
}

// macHaloW is the width of the focus ring: 3pt.
func macHaloW(l *Classic) float32 { return snap(l.S(3)) }

// macFace is the visible body of a control inside its cell: the cell keeps
// a margin for the focus halo around it.
func macFace(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = macSnap(b)
	m := macHaloW(l)
	if b.Dx() < 4*m || b.Dy() < 3*m {
		return b
	}
	return b.Inset(m)
}

// halo paints the focus ring: a band of the focus colour hugging the
// outside of face (radius r), kept inside clip.
func (c *macSet) halo(l *Classic, ctx *paintengine2d.Context, face, clip paintengine2d.Rect, r float32) {
	if face.Empty() {
		return
	}
	w := macHaloW(l)
	ring := face.Inset(-w)
	ctx.Save()
	ctx.ClipRect(macSnap(clip))
	macRing(ctx, ring, r+w, w+macPx(l)*0.5, paintengine2d.Fill(c.focus))
	ctx.Restore()
}

// radius is the corner of a push button face: 4pt on Yosemite, 5pt on Big
// Sur (at AppKit's 20pt face).
func (c *macSet) radius(l *Classic, f paintengine2d.Rect) float32 {
	if l.square() {
		return 0
	}
	r := f.Dy() * 4 / 20
	if c.bigSur {
		r = f.Dy() * 5.5 / 20
	}
	return min(r, f.Dy()*0.5)
}

// Kinds of bezel.
const (
	macPlain  = iota // a white (or grey) face
	macAccent        // the accent face: default button, pop-up cap, segment
	macCheck         // a checked box or selected radio
	macBox           // an unchecked box or radio
)

// bezel paints a face of kind into f with corner r.
func (c *macSet) bezel(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, r float32, kind int, pressed, disabled bool) {
	if f.Dx() < 2 || f.Dy() < 2 {
		return
	}
	lw := macPx(l)
	if disabled && (kind == macAccent || kind == macCheck) {
		kind = macPlain
	}
	// The soft shadow (Big Sur) or the hairline shade (Yosemite) under the
	// face.
	if !disabled {
		if c.bigSur {
			DropShadow(ctx, f, r, paintengine2d.RGBA(0, 0, 0, c.faceShadow), 0, lw*0.5, l.S(1.5), 0)
		} else if kind == macPlain || kind == macAccent {
			ctx.DrawRoundRect(f.Translate(paintengine2d.Pt(0, lw)), r, r, paintengine2d.Fill(c.btnShade))
		}
	}
	var face, rim paintengine2d.Paint
	switch kind {
	case macAccent:
		face, rim = VGradient(f, c.accentStops...), VGradient(f, c.accentEdgeStops...)
		if pressed {
			face = VGradient(f, c.accentPressStops...)
		}
	case macCheck:
		face, rim = VGradient(f, c.checkStops...), paintengine2d.Fill(c.checkEdge)
		if pressed {
			face = VGradient(f, c.accentPressStops...)
		}
	case macBox:
		face, rim = VGradient(f, c.btnStops...), VGradient(f, c.chkEdgeStops...)
		if pressed {
			face = paintengine2d.Fill(c.btnPress)
		}
	default:
		face, rim = VGradient(f, c.btnStops...), VGradient(f, c.btnEdgeStops...)
		if pressed {
			face = paintengine2d.Fill(c.btnPress)
		}
	}
	if disabled {
		// A disabled face fades half way toward the window.
		ctx.DrawRoundRect(f, r, r, face)
		macRing(ctx, f, r, lw, rim)
		ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.win.WithAlpha(0.5)))
		return
	}
	ctx.DrawRoundRect(f, r, r, face)
	if c.dark && (kind == macPlain || kind == macBox) {
		// Dark faces are lit along their top edge.
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(f.Min.X, f.Min.Y, f.Dx(), lw))
		macRing(ctx, f, r, lw, paintengine2d.Fill(c.fieldEdge))
		ctx.Restore()
		return
	}
	macRing(ctx, f, r, lw, rim)
}

// macAddRoundRect appends a closed rect with per-corner radii (tl, tr, br,
// bl) to p, so a shape and its inset can make one even-odd ring.
func macAddRoundRect(p *paintengine2d.Path, b paintengine2d.Rect, tl, tr, br, bl float32) {
	if b.Empty() {
		return
	}
	lim := min(b.Dx(), b.Dy()) * 0.5
	tl, tr, br, bl = min(max(tl, 0), lim), min(max(tr, 0), lim), min(max(br, 0), lim), min(max(bl, 0), lim)
	const k = 0.5522847 // cubic circle constant
	p.MoveTo(b.Min.X+tl, b.Min.Y)
	p.LineTo(b.Max.X-tr, b.Min.Y)
	if tr > 0 {
		p.CubicTo(b.Max.X-tr+tr*k, b.Min.Y, b.Max.X, b.Min.Y+tr-tr*k, b.Max.X, b.Min.Y+tr)
	}
	p.LineTo(b.Max.X, b.Max.Y-br)
	if br > 0 {
		p.CubicTo(b.Max.X, b.Max.Y-br+br*k, b.Max.X-br+br*k, b.Max.Y, b.Max.X-br, b.Max.Y)
	}
	p.LineTo(b.Min.X+bl, b.Max.Y)
	if bl > 0 {
		p.CubicTo(b.Min.X+bl-bl*k, b.Max.Y, b.Min.X, b.Max.Y-bl+bl*k, b.Min.X, b.Max.Y-bl)
	}
	p.LineTo(b.Min.X, b.Min.Y+tl)
	if tl > 0 {
		p.CubicTo(b.Min.X, b.Min.Y+tl-tl*k, b.Min.X+tl-tl*k, b.Min.Y, b.Min.X+tl, b.Min.Y)
	}
	p.Close()
}

// macTick strokes the check mark into g.
func macTick(ctx *paintengine2d.Context, g paintengine2d.Rect, col paintengine2d.Color, w float32) {
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X+g.Dx()*0.18, g.Min.Y+g.Dy()*0.54)
	p.LineTo(g.Min.X+g.Dx()*0.41, g.Min.Y+g.Dy()*0.78)
	p.LineTo(g.Min.X+g.Dx()*0.84, g.Min.Y+g.Dy()*0.22)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// macTriangle fills a small solid triangle (Yosemite's disclosure and
// submenu arrows).
func macTriangle(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, w, h float32, col paintengine2d.Color) {
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
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
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// macChevron strokes a small chevron (Big Sur's arrows).
func macChevron(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color, s float32) {
	s = min(s, b.Dx(), b.Dy())
	if s < 3 {
		return
	}
	Chevron(ctx, macCentered(b, s, s), dir, col, max(l.S(1.5), 1))
}

// updown strokes the double chevron of a pop-up button's cap.
func macUpDown(l *Classic, ctx *paintengine2d.Context, cap paintengine2d.Rect, col paintengine2d.Color) {
	cx, cy := (cap.Min.X+cap.Max.X)*0.5, (cap.Min.Y+cap.Max.Y)*0.5
	a := min(l.S(3), cap.Dx()*0.25)
	g := min(l.S(2), cap.Dy()*0.12)
	st := paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: max(l.S(1.5), 1), Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
	up := paintengine2d.NewPath()
	up.MoveTo(cx-a, cy-g)
	up.LineTo(cx, cy-g-a)
	up.LineTo(cx+a, cy-g)
	ctx.DrawPath(up, st)
	dn := paintengine2d.NewPath()
	dn.MoveTo(cx-a, cy+g)
	dn.LineTo(cx, cy+g+a)
	dn.LineTo(cx+a, cy+g)
	ctx.DrawPath(dn, st)
}

// vibrant paints a material: what lies under b, blurred, under a
// translucent tint (the menu and menu bar vibrancy).
func (c *macSet) vibrant(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, tint paintengine2d.Color) {
	if l.P("vibrancy", 1) != 0 && tint.A < 1 {
		ctx.Save()
		ctx.ClipRoundRect(b, r, r)
		ctx.BackdropBlur(b, l.S(16))
		ctx.Restore()
	}
	macFill(ctx, b, r, paintengine2d.Fill(tint))
}

// ---- parts --------------------------------------------------------------------------------------

func (e macosEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := macColors(l)
	switch role {
	case RoleButton, RoleCombo:
		f := macFace(l, b)
		// The default button is the accent (plain in an inactive window);
		// until macOS 12 any push button turned the accent while pressed.
		pressed := st.Pressed() && !st.Disabled()
		accent := (st.Primary() && !st.Backdrop() || pressed) && !st.Disabled() && role == RoleButton
		kind := macPlain
		if accent {
			kind = macAccent
		}
		c.bezel(l, ctx, f, c.radius(l, f), kind, pressed && st.Primary(), st.Disabled())
		switch {
		case st.Disabled():
			return c.text3
		case accent:
			return c.onAccent
		}
		return c.btnText
	case RoleTool:
		return c.tool(l, ctx, b, st)
	case RoleField:
		c.field(l, ctx, macFace(l, b), st)
		if st.Disabled() {
			return c.text2
		}
		return c.text
	case RoleCheck:
		f := macFace(l, b)
		c.bezel(l, ctx, f, l.rx(3), macBox, false, st.Disabled())
		return c.text
	case RoleRow:
		return c.row(l, ctx, macSnap(b), st, 0)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
			return c.onAccent
		}
		return c.text
	case RoleTab:
		return c.text
	case RoleThumb:
		macFill(ctx, b, min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.knob))
		return c.text
	case RoleTrack:
		macFill(ctx, b, min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.slider))
		return c.text
	case RoleBar:
		ctx.DrawRect(b, VGradient(b, c.barStops...))
		return c.text
	case RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.win))
		return c.text
	}
	return c.text
}

// field paints a text field face into f.
func (c *macSet) field(l *Classic, ctx *paintengine2d.Context, f paintengine2d.Rect, st ControlState) {
	if f.Dx() < 2 || f.Dy() < 2 {
		return
	}
	lw := macPx(l)
	r := float32(0) // text fields stay square in both eras
	if c.bigSur && !st.Disabled() {
		ctx.DrawRect(paintengine2d.XYWH(f.Min.X, f.Max.Y, f.Dx(), lw), paintengine2d.Fill(c.btnShade))
	}
	fill := c.fieldFill
	if st.Disabled() {
		fill = c.fieldDis
	}
	ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(fill))
	macRing(ctx, f, r, lw, VGradient(f, c.fieldStops...))
}

// tool paints a tool button: Yosemite's bezelled (textured) button, Big
// Sur's borderless one with a rounded wash under the pointer. It returns
// the glyph colour.
func (c *macSet) tool(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return c.text2
	}
	fg := c.text2
	if c.dark {
		fg = c.text
	}
	if st.Disabled() {
		fg = c.text3
	}
	if c.bigSur {
		r := min(l.rx(6), b.Dy()*0.5)
		switch {
		case st.Disabled():
		case st.Pressed():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.washPress))
		case st.Checked():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.toolOn))
			fg = c.toolOnText
		case st.Hovered():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash))
		}
		return fg
	}
	// Yosemite: a textured rounded bezel, darker while pressed; a latched
	// one is the dark grey segment with a white glyph.
	f := macCentered(b, b.Dx()-l.S(4), min(b.Dy()-l.S(6), l.S(26)))
	r := min(l.rx(4), f.Dy()*0.5)
	if st.Checked() && !st.Disabled() {
		ctx.DrawRoundRect(f.Translate(paintengine2d.Pt(0, macPx(l))), r, r, paintengine2d.Fill(c.btnShade))
		ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.toolOn))
		if st.Pressed() {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.15)))
		}
		return c.toolOnText
	}
	c.bezel(l, ctx, f, r, macPlain, st.Pressed(), st.Disabled())
	return fg
}

// row paints an item row's selection (list, tree, or a table cell's part
// of it) into box with corner r and returns the text colour. Lists grey
// their selection when they lose focus, as AppKit does.
func (c *macSet) row(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, r float32) paintengine2d.Color {
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	if !st.Checked() {
		return fg
	}
	fill, text := c.sel, c.onAccent
	if st.Inactive() || st.Backdrop() {
		fill, text = c.selOff, c.selOffText
	}
	if st.Disabled() {
		fill, text = c.selOff, c.text3
	}
	macFill(ctx, box, r, paintengine2d.Fill(fill))
	return text
}

// rowBox is where a row's selection paints: Yosemite fills the row, Big
// Sur a rounded box inset from the view's sides.
func (c *macSet) rowBox(l *Classic, b paintengine2d.Rect) (paintengine2d.Rect, float32) {
	b = macSnap(b)
	if !c.bigSur {
		return b, 0
	}
	in := snap(l.S(10))
	v := snap(l.S(1))
	if b.Dx() < 4*in || b.Dy() < 4*v {
		return b, 0
	}
	box := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+in, b.Min.Y+v), Max: paintengine2d.Pt(b.Max.X-in, b.Max.Y-v)}
	return box, min(l.rx(5), box.Dy()*0.5)
}

func (e macosEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := macColors(l)
	s := snap(min(l.S(16), box.Dx(), box.Dy()))
	if s < 4 {
		return
	}
	g := macCentered(box, s, s)
	on := checked && !st.Disabled()
	r := l.rx(3.5)
	if c.bigSur {
		r = l.rx(4)
	}
	kind := macBox
	if on {
		kind = macCheck
	}
	c.bezel(l, ctx, g, min(r, s*0.5), kind, st.Pressed() && !st.Disabled(), st.Disabled())
	if checked {
		col := c.onAccent
		if !on {
			col = c.text3
		}
		macTick(ctx, g.Inset(s*0.12), col, max(l.S(1.8), 1.2))
	}
}

func (e macosEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := macColors(l)
	d := snap(min(macRadioD(l), box.Dx(), box.Dy()))
	if d < 4 {
		return
	}
	g := macCentered(box, d, d)
	on := selected && !st.Disabled()
	kind := macBox
	if on {
		kind = macCheck
	}
	c.bezel(l, ctx, g, d*0.5, kind, st.Pressed() && !st.Disabled(), st.Disabled())
	if selected {
		col := c.onAccent
		if !on {
			col = c.text3
		}
		ctx.DrawCircle(paintengine2d.Pt((g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5), d*0.19, paintengine2d.Fill(col))
	}
}

// macRadioD is a radio's diameter: 16pt on Yosemite, 14pt (the check
// box's size) on Big Sur, at the toolkit's scale.
func macRadioD(l *Classic) float32 {
	if macColors(l).bigSur {
		return l.S(16)
	}
	return l.S(18)
}

// Arrow: Yosemite's small solid triangles, Big Sur's chevrons.
func (macosEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	c := macColors(l)
	if c.bigSur {
		macChevron(l, ctx, b, dir, col, l.S(9))
		return
	}
	w := min(l.S(8), b.Dx(), b.Dy())
	macTriangle(ctx, b, dir, w, w*0.6, col)
}

// Expander is the disclosure triangle (Yosemite) or chevron (Big Sur).
func (macosEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := macColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	if c.bigSur {
		macChevron(l, ctx, b, dir, col, l.S(9))
		return
	}
	w := min(l.S(9), b.Dx(), b.Dy())
	macTriangle(ctx, b, dir, w, w*0.78, col)
}

// MenuHighlight is the hot menu row: a full-width accent band on
// Yosemite, a rounded box inset from the menu's sides on Big Sur; an open
// menu-bar title is the accent (Yosemite) or a translucent rounded wash
// (Big Sur).
func (macosEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := macColors(l)
	b = macSnap(b)
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	if c.bigSur {
		r := min(l.rx(4), b.Dy()*0.5)
		col := c.menuHi
		if attachBottom {
			col = c.washPress
		}
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.menuHi))
}

func (macosEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := macColors(l)
	if hot {
		return c.onAccent
	}
	return c.text
}

// Fields, combos and views ring focus with the halo.
func (macosEngine) FieldFocusRing(l *Classic) bool { return true }

// DrawFocusRing is the focus halo just inside b.
func (macosEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	w := macHaloW(l)
	if b.Dx() < 3*w || b.Dy() < 3*w {
		return
	}
	macRing(ctx, b, min(l.rx(4)+w, b.Dy()*0.5), w, paintengine2d.Fill(c.focus))
}

// ---- scroll bars --------------------------------------------------------------------------------

// ScrollBarStyle: overlay scrollers — the knob floats over the content
// while it scrolls and fades away; the pointer widens it onto a track.
func (macosEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Overlay: true, Transient: true, Inset: 0, MinThumb: 26, EndPad: 3}
}

func (macosEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := macColors(l)
	bar := macSnap(p.Bar)
	if bar.Empty() || st.Disabled {
		return
	}
	hover := st.Hovered || st.Pressed != ScrollNone
	px := macPx(l)
	if hover {
		// The expanded scroller: a translucent light track with a line on
		// its content side.
		ctx.DrawRect(bar, paintengine2d.Fill(c.track))
		if vertical {
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, px, bar.Dy()), paintengine2d.Fill(c.trackEdge))
		} else {
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), px), paintengine2d.Fill(c.trackEdge))
		}
	}
	if p.Thumb.Empty() {
		return
	}
	// The knob: 7pt at rest, 11pt under the pointer, its outer edge 2pt
	// from the view's, fully round, in a faint outline of the other ink.
	w := snap(l.S(7))
	if hover {
		w = snap(l.S(11))
	}
	edge := snap(l.S(2))
	col := c.knob
	if hover && (st.Pressed == ScrollThumbPart || st.Hot == ScrollThumbPart) {
		col = c.knobHot
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
	macRing(ctx, t, r, max(px*0.8, 0.8), paintengine2d.Fill(c.knobRim))
}

func (e macosEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered() || st.Pressed()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------------------------

// boxR is the corner of group boxes and the tab pane.
func (c *macSet) boxR(l *Classic) float32 {
	if c.bigSur {
		return l.rx(7)
	}
	return l.rx(5)
}

// GroupBoxInsets: an NSBox — the title sits above the box, left-aligned.
func (macosEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top = snap(l.body.Height()+l.S(4)) + pad
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

func (macosEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := macColors(l)
	b = macSnap(b)
	box := b
	if title != "" {
		th := snap(l.body.Height() + l.S(4))
		l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8), th), c.text, AlignStart, 0)
		box = paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+th), Max: b.Max}
	}
	c.boxPane(l, ctx, box, raised)
}

// boxPane is the rounded box of group boxes and tab panes.
func (c *macSet) boxPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := min(c.boxR(l), b.Dy()*0.5)
	fill := c.box
	if raised {
		fill = mdOver(c.box, c.content, 0.5)
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	macRing(ctx, b, r, macPx(l), paintengine2d.Fill(c.boxEdge))
}

// macTitleH is the in-app window's title bar.
func macTitleH(l *Classic) float32 {
	c := macColors(l)
	h := l.S(26)
	if c.bigSur {
		h = l.S(32)
	}
	return snap(max(h, l.body.Height()+l.S(6)))
}

func (macosEngine) WindowFrameInsets(l *Classic) Insets {
	px := macPx(l)
	return Insets{Top: macTitleH(l), Right: px, Bottom: px, Left: px}
}

// macLight is the rect of traffic light i (0 close, 1 minimise, 2 zoom).
func macLight(l *Classic, b paintengine2d.Rect, i int) paintengine2d.Rect {
	c := macColors(l)
	h := macTitleH(l)
	d := snap(min(l.S(12), h-l.S(6)))
	x0 := l.S(8)
	if c.bigSur {
		x0 = l.S(12)
	}
	x := b.Min.X + x0 + float32(i)*l.S(20)
	return paintengine2d.XYWH(snap(x), snap(b.Min.Y+(h-d)*0.5), d, d)
}

// WindowCloseRect is the red traffic light, first on the left.
func (macosEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = macSnap(b)
	if b.Dx() < l.S(80) || b.Dy() < macTitleH(l) {
		return paintengine2d.Rect{}
	}
	return macLight(l, b, 0)
}

// DrawWindowFrame is a macOS window: Yosemite's light gradient title bar
// with rounded top corners, Big Sur's flat one with 10px corners all round;
// traffic lights on the left, the title centred.
func (e macosEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := macColors(l)
	b = macSnap(b)
	if b.Empty() {
		return
	}
	px := macPx(l)
	h := macTitleH(l)
	// Every corner is round since Lion: 4pt until Catalina, 10pt on Big Sur.
	r := l.rx(4)
	if c.bigSur {
		r = l.rx(10)
	}
	r = min(r, b.Dy()*0.3)
	rb := r
	shape := RoundRectPath(b, r, r, rb, rb)
	ctx.Save()
	ctx.ClipPath(shape)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	bar := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), min(h, b.Dy()))
	if st.Active {
		ctx.DrawRect(bar, VGradient(bar, c.titleStops...))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-px, bar.Dx(), px), paintengine2d.Fill(c.titleEdge))
		if !c.bigSur && !c.dark {
			// A lit top edge.
			ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y+px, bar.Dx(), px), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.6)))
		}
	} else {
		ctx.DrawRect(bar, paintengine2d.Fill(c.titleOffBar))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-px, bar.Dx(), px), paintengine2d.Fill(c.titleOffEdge))
	}
	ctx.Restore()
	// The outline.
	edge := paintengine2d.NewPath()
	macAddRoundRect(edge, b, r, r, rb, rb)
	macAddRoundRect(edge, b.Inset(px), max(r-px, 0), max(r-px, 0), max(rb-px, 0), max(rb-px, 0))
	ctx.DrawPath(edge, paintengine2d.Paint{Color: c.winEdge, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
	if b.Dx() < l.S(80) || b.Dy() < h {
		return
	}
	for i := 0; i < 3; i++ {
		lb := macLight(l, b, i)
		fill, rim := c.lights[i], c.lightEdges[i]
		if !st.Active || (i == 0 && !st.CanClose) {
			fill, rim = c.lightOff, c.lightOffEdge
		}
		if i == 0 && st.ClosePress && st.CanClose {
			fill = mdOver(fill, paintengine2d.RGB(0, 0, 0), 0.2)
		}
		ctr := paintengine2d.Pt((lb.Min.X+lb.Max.X)*0.5, (lb.Min.Y+lb.Max.Y)*0.5)
		ctx.DrawCircle(ctr, lb.Dx()*0.5, paintengine2d.Fill(rim))
		ctx.DrawCircle(ctr, lb.Dx()*0.5-px*0.75, paintengine2d.Fill(fill))
		if i == 0 && st.CanClose && (st.CloseHot || st.ClosePress) {
			DrawCross(ctx, lb.Inset(lb.Dx()*0.3), c.glyph, max(l.S(1.2), 1))
		}
	}
	if title == "" {
		return
	}
	col := c.titleText
	if !st.Active {
		col = c.titleOff
	}
	left := macLight(l, b, 2).Max.X + l.S(8)
	f := l.body
	tw := f.Advance(title)
	x := b.Min.X + (b.Dx()-tw)*0.5
	if x < left {
		x = left
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-l.S(8)-x, h-px), col, AlignStart, 0)
}

func (macosEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(macColors(l).win))
}

// PopupShadow: menus float on a soft shadow, windows and sheets on a deep
// one, help tags on a small one.
func (macosEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp := macShadow(l, kind)
	return ShadowReach(0, sp.dy, sp.blur, 0)
}

func (macosEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	sp := macShadow(l, kind)
	DropShadow(ctx, b, sp.r, sp.col, 0, sp.dy, sp.blur, 0)
}

func macShadow(l *Classic, kind PopupKind) baseShadowSpec {
	c := macColors(l)
	k := l.P("shadow", 1)
	if c.dark {
		k *= 1.6
	}
	a := func(v float32) paintengine2d.Color { return paintengine2d.RGBA(0, 0, 0, min(v*k, 0.9)) }
	switch kind {
	case PopupTooltip:
		return baseShadowSpec{col: a(0.2), r: l.rx(1), dy: l.S(1), blur: l.S(4)}
	case PopupDialog:
		// Windows and sheets sit on a deep shadow, darkest below.
		r := l.rx(4)
		if c.bigSur {
			r = l.rx(10)
		}
		return baseShadowSpec{col: a(0.42), r: r, dy: l.S(12), blur: l.S(40)}
	}
	return baseShadowSpec{col: a(0.26), r: c.menuR(l), dy: l.S(4), blur: l.S(16)}
}

// ItemFocus: none — a focused view is ringed as a whole (its view frame
// draws the halo), the Mac way.
func (macosEngine) ItemFocus(*Classic, *paintengine2d.Context, paintengine2d.Rect, ControlState) {}

// ViewFrameInsets: the scroll view's hairline bezel, with room outside it
// for the focus halo.
func (macosEngine) ViewFrameInsets(l *Classic) Insets {
	v := macHaloW(l)
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

func (macosEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := macColors(l)
	b = macSnap(b)
	w := macHaloW(l)
	if b.Dx() < 3*w || b.Dy() < 3*w {
		return
	}
	px := macPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	inner := b.Inset(w - px)
	ctx.DrawRect(inner, paintengine2d.Fill(c.content))
	edge := c.fieldEdge
	if c.bigSur || c.dark {
		edge = c.sep
	}
	macRing(ctx, inner, 0, px, paintengine2d.Fill(edge))
	if st.Focused() && !st.Disabled() {
		macRing(ctx, b, min(l.rx(3)+w, b.Dy()*0.5), w, paintengine2d.Fill(c.focus))
	}
}

// StyleHint: the Mac order (default button last), centred segmented tabs,
// right-aligned form labels; no hover fades and no pulsing default button
// since Yosemite.
func (macosEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintTabsCentered, HintFormLabelsRight:
		return 1
	}
	return 0
}

// SpinBoxStyle: the stepper stands beside its field.
func (macosEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{} }

// TabOverlap: neighbouring segments share their divider.
func (macosEngine) TabOverlap(l *Classic) float32 { return macPx(l) }

// DrawTabPane is the rounded box under the segmented tabs: it starts
// half way down the tab strip, so the segments straddle its top edge.
func (macosEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	top := snap(b.Min.Y + l.metrics.TabH*0.5)
	if top >= b.Max.Y-l.S(4) {
		return
	}
	c.boxPane(l, ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}, false)
}

// ---- controls -----------------------------------------------------------------------------------

func (e macosEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := macColors(l)
	fg := e.Face(l, ctx, b, RoleButton, st)
	f := macFace(l, b)
	l.drawFittedText(ctx, l.body, label, f, fg, AlignCenter, l.S(12))
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, f, b, c.radius(l, f))
	}
}

func (e macosEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := macColors(l)
	fg := c.tool(l, ctx, b, st)
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
	if label != "" {
		tc := c.text
		switch {
		case st.Disabled():
			tc = c.text3
		case st.Checked():
			tc = fg // on the latched segment
		}
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x-pad*0.5, b.Dy()), tc, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		f := macSnap(b).Inset(macHaloW(l))
		c.halo(l, ctx, f, b, min(l.rx(4), f.Dy()*0.5))
	}
}

// macToggleLayout puts a check box / radio cell at the left of b and the
// label after the indicator.
func macToggleLayout(l *Classic, b paintengine2d.Rect, side float32) (cell, lb paintengine2d.Rect) {
	side = min(side, b.Dx())
	cell = macSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	x := cell.Max.X - macHaloW(l) + l.S(6)
	lb = paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy())
	return cell, lb
}

func (e macosEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := macColors(l)
	cell, lb := macToggleLayout(l, b, l.metrics.Checkbox)
	e.CheckIndicator(l, ctx, cell, st, checked)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		s := snap(min(l.S(16), cell.Dx(), cell.Dy()))
		g := macCentered(cell, s, s)
		r := l.rx(3)
		if c.bigSur {
			r = l.rx(4)
		}
		c.halo(l, ctx, g, b, min(r, s*0.5))
	}
}

func (e macosEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := macColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	cell, lb := macToggleLayout(l, b, side)
	e.RadioIndicator(l, ctx, cell, st, selected)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		d := snap(min(macRadioD(l), cell.Dx(), cell.Dy()))
		g := macCentered(cell, d, d)
		c.halo(l, ctx, g, b, d*0.5)
	}
}

func (c *macSet) toggleLabel(l *Classic, ctx *paintengine2d.Context, lb paintengine2d.Rect, st ControlState, label string) {
	if label == "" || lb.Dx() <= 0 {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

// DrawSwitch is NSSwitch (it arrived in 10.15; the Yosemite pack draws it
// in its own colours): a capsule track, the accent when on, a white knob
// with a soft shadow.
func (e macosEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := macColors(l)
	sw, sh := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	cell := macSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-sh)*0.5, min(sw, b.Dx()), sh))
	w := macHaloW(l)
	if cell.Dx() < 4*w || cell.Dy() < 3*w {
		return
	}
	track := cell.Inset(w)
	r := track.Dy() * 0.5
	px := macPx(l)
	switch {
	case on && !st.Disabled():
		fill := c.accent
		if st.Backdrop() {
			fill = c.selOff
		}
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(fill))
	default:
		fill := c.switchOff
		if st.Disabled() {
			fill = mdOver(c.switchOff, c.win, 0.5)
		}
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(fill))
		macRing(ctx, track, r, px, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.1)))
	}
	d := track.Dy() - 2*px
	cx := track.Min.X + px + d*0.5
	if on {
		cx = track.Max.X - px - d*0.5
	}
	cy := (track.Min.Y + track.Max.Y) * 0.5
	knob := paintengine2d.XYWH(cx-d*0.5, cy-d*0.5, d, d)
	ctx.Save()
	ctx.ClipRect(cell)
	DropShadow(ctx, knob, d*0.5, paintengine2d.RGBA(0, 0, 0, 0.3), 0, px*0.5, l.S(2), 0)
	ctx.Restore()
	kc := c.knobFace
	if st.Pressed() && !st.Disabled() {
		kc = mdOver(kc, paintengine2d.RGB(0, 0, 0), 0.08)
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), d*0.5, paintengine2d.Fill(kc))
	lb := paintengine2d.XYWH(cell.Max.X-w+l.S(6), b.Min.Y, b.Max.X-(cell.Max.X-w+l.S(6)), b.Dy())
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, track, b, r)
	}
}

// DrawSlider is NSSlider: a thin rounded track, the accent up to a round
// white knob.
func (e macosEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := macColors(l)
	b = macSnap(b)
	t = clamp1(t)
	w := macHaloW(l)
	d := min(snap(l.S(17)), b.Dy()-2*w)
	th := max(snap(l.S(3)), 1)
	if c.bigSur {
		d, th = min(snap(l.S(21)), b.Dy()-2*w), snap(l.S(4))
	}
	if d < 6 || b.Dx() < d+2*w {
		return
	}
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	x0, x1 := b.Min.X+w+d*0.5, b.Max.X-w-d*0.5
	tx := x0 + (x1-x0)*t
	track := paintengine2d.XYWH(b.Min.X+w, cy-th*0.5, b.Dx()-2*w, th)
	macFill(ctx, track, th*0.5, paintengine2d.Fill(c.slider))
	fill := c.accent
	if st.Disabled() || st.Backdrop() {
		fill = c.selOff
		if c.dark {
			fill = c.text3
		}
	}
	if tx > track.Min.X {
		macFill(ctx, paintengine2d.XYWH(track.Min.X, track.Min.Y, tx-track.Min.X, th), th*0.5, paintengine2d.Fill(fill))
	}
	knob := paintengine2d.XYWH(tx-d*0.5, cy-d*0.5, d, d)
	px := macPx(l)
	ctx.Save()
	ctx.ClipRect(b)
	DropShadow(ctx, knob, d*0.5, paintengine2d.RGBA(0, 0, 0, 0.25), 0, px*0.5, l.S(2), 0)
	ctx.Restore()
	kc := c.knobFace
	if st.Pressed() && !st.Disabled() {
		kc = mdOver(kc, paintengine2d.RGB(0, 0, 0), 0.08)
	}
	ctr := paintengine2d.Pt(tx, cy)
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(c.knobEdge))
	ctx.DrawCircle(ctr, d*0.5-px*0.5, paintengine2d.Fill(kc))
	if st.Focused() && !st.Disabled() {
		c.halo(l, ctx, knob, b, d*0.5)
	}
}

// DrawProgressBar is NSProgressIndicator's bar: a thin rounded track and
// the accent fill; the busy bar sweeps a soft accent segment across.
func (e macosEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := macColors(l)
	b = macSnap(b)
	h := min(snap(l.S(6)), b.Dy())
	if b.Dx() < 4 || h < 2 {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := h * 0.5
	ctx.DrawRoundRect(bar, r, r, VGradient(bar, c.progressStops...))
	fill, busy := c.accent, c.busyStops
	if st.Disabled() {
		fill, busy = c.text3, c.busyDimStops
	}
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.35
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, bar.Min.Y, span, h)
		ctx.Save()
		ctx.ClipRoundRect(bar, r, r)
		ctx.DrawRect(seg, HGradient(seg, busy...))
		ctx.Restore()
		return
	}
	if w := snap(bar.Dx() * clamp1(t)); w >= 1 {
		ctx.Save()
		ctx.ClipRoundRect(bar, r, r)
		ctx.DrawRoundRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, r, paintengine2d.Fill(fill))
		ctx.Restore()
	}
}

func (e macosEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := macColors(l)
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	f := macFace(l, b)
	c.field(l, ctx, f, st)
	l.baseDrawTextField(ctx, f, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
	if st.Focused() && !st.Disabled() {
		r := float32(0)
		if c.bigSur {
			r = l.rx(3)
		}
		c.halo(l, ctx, f, b, r)
	}
}

// DrawComboBox is NSPopUpButton: a push button face with the value and,
// at the right, the accent cap with its double chevron (Yosemite: the
// whole right end; Big Sur: a small rounded square inside the face).
func (e macosEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := macColors(l)
	f := macFace(l, b)
	if f.Dx() < 8 || f.Dy() < 6 {
		return
	}
	r := c.radius(l, f)
	pressed := (open || st.Pressed()) && !st.Disabled()
	c.bezel(l, ctx, f, r, macPlain, pressed, st.Disabled())
	on := !st.Disabled() && !st.Backdrop()
	var cap paintengine2d.Rect
	if c.bigSur {
		s := snap(min(f.Dy()-l.S(6), l.S(16)))
		cap = paintengine2d.XYWH(f.Max.X-s-snap(l.S(3)), snap((f.Min.Y+f.Max.Y-s)*0.5), s, s)
		if on {
			macFill(ctx, cap, l.rx(4), paintengine2d.Fill(c.accent))
		}
	} else {
		cw := snap(min(l.S(18), f.Dx()*0.4))
		cap = paintengine2d.XYWH(f.Max.X-cw, f.Min.Y, cw, f.Dy())
		if on {
			ctx.Save()
			ctx.ClipRect(cap)
			ctx.DrawRoundRect(f, r, r, VGradient(f, c.accentStops...))
			macRing(ctx, f, r, macPx(l), VGradient(f, c.accentEdgeStops...))
			ctx.Restore()
		}
	}
	glyph := c.onAccent
	if !on {
		glyph = c.text2
	}
	macUpDown(l, ctx, cap, glyph)
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	pad := l.S(9)
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(f.Min.X+pad, f.Min.Y, cap.Min.X-f.Min.X-pad-l.S(2), f.Dy()), fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.halo(l, ctx, f, b, r)
	}
}

// DrawSpinner is NSStepper, standing beside its field: a narrow rounded
// face split in two, a small arrow in each half.
func (e macosEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := macColors(l)
	b = macSnap(b)
	w := snap(min(l.S(15), b.Dx()-l.S(4)))
	h := snap(min(l.S(24), b.Dy()-2*macHaloW(l)))
	if w < 6 || h < 8 {
		return
	}
	f := paintengine2d.XYWH(b.Max.X-w-snap(max(l.S(1), (b.Dx()-w-l.S(4))*0.5)), snap((b.Min.Y+b.Max.Y-h)*0.5), w, h)
	r := min(l.rx(4), w*0.5)
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(b)
	c.bezel(l, ctx, f, r, macPlain, false, st.Disabled())
	mid := snap((f.Min.Y + f.Max.Y) * 0.5)
	px := macPx(l)
	up := paintengine2d.XYWH(f.Min.X, f.Min.Y, w, mid-f.Min.Y)
	dn := paintengine2d.XYWH(f.Min.X, mid, w, f.Max.Y-mid)
	press := func(half paintengine2d.Rect, tl, tr, br, bl float32) {
		p := RoundRectPath(half.Inset(px), tl, tr, br, bl)
		ctx.DrawPath(p, paintengine2d.Fill(c.btnPress))
	}
	if upPress && !st.Disabled() {
		press(up, r, r, 0, 0)
	}
	if downPress && !st.Disabled() {
		press(dn, 0, 0, r, r)
	}
	ctx.DrawRect(paintengine2d.XYWH(f.Min.X+px*2, mid, w-px*4, px), paintengine2d.Fill(c.sep))
	col := c.text
	if st.Disabled() {
		col = c.text3
	}
	e.Arrow(l, ctx, up, DirUp, col)
	e.Arrow(l, ctx, dn, DirDown, col)
}

// DrawTabBar paints what the segmented tabs straddle: the window above
// their middle, the pane's box from the middle down.
func (e macosEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	mid := snap(b.Min.Y + b.Dy()*0.5)
	if mid >= b.Max.Y {
		return
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, mid), Max: b.Max})
	c.boxPane(l, ctx, paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, mid), Max: paintengine2d.Pt(b.Max.X, b.Max.Y+l.S(16))}, false)
	ctx.Restore()
}

// DrawTab is one segment of the tab control: the first and last round
// their outer ends, neighbours share a divider; the selected segment is
// the accent (Yosemite) or a white raised knob on the grey track (Big Sur).
func (e macosEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := macColors(l)
	b = macSnap(b)
	h := snap(min(l.S(24), b.Dy()-l.S(4)))
	if b.Dx() < 6 || h < 8 {
		return
	}
	seg := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := min(l.rx(4), h*0.5)
	if c.bigSur {
		r = min(l.rx(6), h*0.5)
	}
	rl, rr := float32(0), float32(0)
	if st.First() {
		rl = r
	}
	if st.Last() {
		rr = r
	}
	shape := RoundRectPath(seg, rl, rr, rr, rl)
	px := macPx(l)
	fg := c.text
	on := selected && !st.Disabled() && !st.Backdrop()
	if c.bigSur {
		// A grey track; the selected segment a raised knob inside it.
		track := c.segTrack
		ctx.DrawPath(shape, paintengine2d.Fill(track))
		if !selected {
			if st.Pressed() && !st.Disabled() {
				ctx.DrawPath(shape, paintengine2d.Fill(c.wash))
			}
			// Dividers stand between unselected segments.
			if !st.Last() {
				ctx.DrawRect(paintengine2d.XYWH(seg.Max.X-px, seg.Min.Y+snap(h*0.25), px, snap(h*0.5)), paintengine2d.Fill(c.sep))
			}
		} else {
			knob := seg.Inset(snap(l.S(2)))
			kr := max(min(r-l.S(2), knob.Dy()*0.5), 0)
			fill := c.segKnob
			if st.Disabled() {
				fill = mdOver(fill, c.win, 0.5)
			}
			DropShadow(ctx, knob, kr, paintengine2d.RGBA(0, 0, 0, 0.18), 0, px*0.5, l.S(1.5), 0)
			ctx.DrawRoundRect(knob, kr, kr, paintengine2d.Fill(fill))
		}
	} else {
		switch {
		case on:
			ctx.DrawPath(shape, VGradient(seg, c.accentStops...))
			fg = c.onAccent
		case st.Pressed() && !st.Disabled():
			ctx.DrawPath(shape, paintengine2d.Fill(c.btnPress))
		default:
			ctx.DrawPath(shape, VGradient(seg, c.btnStops...))
		}
		// The segment's hairline; the selected one is ringed in its accent.
		edge := VGradient(seg, c.btnEdgeStops...)
		if on {
			edge = VGradient(seg, c.accentEdgeStops...)
		}
		ring := paintengine2d.NewPath()
		macAddRoundRect(ring, seg, rl, rr, rr, rl)
		macAddRoundRect(ring, seg.Inset(px), max(rl-px, 0), max(rr-px, 0), max(rr-px, 0), max(rl-px, 0))
		edge.FillRule = paintengine2d.FillEvenOdd
		ctx.DrawPath(ring, edge)
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

// DrawPanel: a raised panel is the light box; a flat one the window.
func (macosEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := macColors(l)
	if raised {
		c.boxPane(l, ctx, macSnap(b), true)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
}

// DrawMenuBar is the menu bar's material: its tint, a hairline under it.
// An in-window bar lies over the flat window, where a blur would change
// nothing, so it takes the tint flattened over the window.
func (macosEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(Mix(c.win, c.menu, c.menu.A)))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-macPx(l), b.Dx(), macPx(l)), paintengine2d.Fill(c.sep))
}

func (e macosEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := macColors(l)
	b = macSnap(b)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.text3
	case open || st.Pressed():
		hl := b
		if c.bigSur {
			hl = b.Inset(snap(l.S(2)))
			e.MenuHighlight(l, ctx, hl, true)
		} else {
			hl.Max.Y -= macPx(l)
			e.MenuHighlight(l, ctx, hl, true)
			fg = c.onAccent
		}
	case st.Hovered():
		// AppKit has no hover here; a faint wash keeps the feedback.
		macFill(ctx, b.Inset(snap(l.S(2))), l.rx(4), paintengine2d.Fill(c.wash))
	}
	// Mac menus never underline mnemonics; the keys still work.
	l.drawLabeled(ctx, l.body, label, -1, b, fg)
	if st.Focused() && !open {
		c.halo(l, ctx, b.Inset(macHaloW(l)), b, l.rx(4))
	}
}

// menuR is the corner of a menu.
func (c *macSet) menuR(l *Classic) float32 {
	if c.bigSur {
		return l.rx(6)
	}
	return l.rx(4)
}

// DrawMenuFrame is a vibrant menu: the content behind blurred under a
// translucent tint, rounded corners and a hairline edge.
func (macosEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := min(c.menuR(l), b.Dy()*0.25)
	c.vibrant(l, ctx, b, r, c.menu)
	macRing(ctx, b, r, macPx(l), paintengine2d.Fill(c.menuEdge))
}

func (e macosEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := macColors(l)
	ch := MenuChromeFor(l)
	px := macPx(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := b.Min.X-ch.PadL+px, b.Max.X+ch.PadR-px
		if c.bigSur {
			// Big Sur starts its separators in line with the item text.
			x0, x1 = b.Min.X-ch.PadL+l.S(14), b.Max.X+ch.PadR-l.S(14)
		}
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, px), paintengine2d.Fill(c.menuSep))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+px, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*px, b.Dy())
		if c.bigSur {
			k := l.S(5)
			hb = paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy())
		}
		e.MenuHighlight(l, ctx, hb, false)
	}
	fg, sc := c.text, c.text2
	font := l.body
	switch {
	case st.Disabled():
		fg, sc = c.text3, c.text3
	case hot:
		fg, sc = c.onAccent, c.onAccent
	}
	macGutter(l, ctx, b, ch, row, fg)
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

// macGutter paints a menu row's leading column: a check mark for a chosen
// item (Mac menus mark radio choices the same way) or a 16pt icon.
func macGutter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, ch MenuChrome, row MenuRow, fg paintengine2d.Color) {
	gw := ch.CheckCol()
	side := snap(min(b.Dy()-l.S(4), gw-l.S(2), l.S(16)))
	if side < 4 {
		return
	}
	box := macSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	switch {
	case row.Checked:
		macTick(ctx, box.Inset(side*0.12), fg, max(l.S(1.6), 1.2))
	case !row.Radio && row.Icon != IconNone:
		l.drawToolIcon(ctx, box, row.Icon, fg)
	}
}

func (e macosEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := macColors(l)
	box, r := c.rowBox(l, b)
	fg := c.row(l, ctx, box, st, r)
	pad := l.S(8)
	if c.bigSur {
		pad = l.S(14)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-l.S(8), b.Dy()), fg, AlignStart, 0)
}

func (e macosEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := macColors(l)
	box, r := c.rowBox(l, b)
	fg := c.row(l, ctx, box, st, r)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if c.bigSur {
		x += l.S(8)
	}
	if !leaf {
		col := c.text2
		if st.Checked() && fg == c.onAccent {
			col = c.onAccent
		} else if st.ExpanderHot() {
			col = c.text
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, col)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(3)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(4), b.Dy()), fg, AlignStart, 0)
}

// DrawTableHeader is the list header: a hairline under it and between
// columns, the sorted column's small arrow at its right end.
func (e macosEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := macColors(l)
	b = macSnap(b)
	if b.Empty() {
		return
	}
	px := macPx(l)
	fill := c.header
	if st.Pressed() && !st.Disabled() {
		fill = mdOver(c.header, c.text, 0.08)
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.headerEdge))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, b.Min.Y+snap(l.S(4)), px, b.Dy()-snap(l.S(8))), paintengine2d.Fill(c.headerEdge))
	fg := c.text
	if st.Disabled() {
		fg = c.text3
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, c.text2)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawTableCell: alternating rows, the selection one box across the row
// (Big Sur: rounded and inset from the view's sides, through CellSpan).
func (e macosEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := macColors(l)
	cell := macSnap(b)
	ctx.Save()
	ctx.ClipRect(cell)
	span := CellSpan(cell, st, l.S(24))
	box, r := c.rowBox(l, span)
	if st.Alternate() && !st.Checked() {
		// Big Sur rounds its alternating rows less than its selection.
		macFill(ctx, box, min(r, l.rx(3)), paintengine2d.Fill(c.alt))
	}
	fg := c.row(l, ctx, box, st, r)
	ctx.Restore()
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
	if st.First() && c.bigSur {
		pad = max(pad, l.S(14))
	}
	avail := max(b.Dx()-pad*2, 4)
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
	ctx.ClipRect(cell)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawToolBar: Yosemite's unified title-and-toolbar gradient; Big Sur's
// flat unified toolbar. A hairline under both.
func (macosEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := macColors(l)
	b = macSnap(b)
	ctx.DrawRect(b, VGradient(b, c.barStops...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-macPx(l), b.Dx(), macPx(l)), paintengine2d.Fill(c.barEdge))
}

func (macosEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := macColors(l)
	b = macSnap(b)
	px := macPx(l)
	ctx.DrawRect(b, VGradient(b, c.barStops...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), px), paintengine2d.Fill(c.barEdge))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(10), b.Min.Y, slot-l.S(14), b.Dy()), c.text2, AlignStart, 0)
	}
}

func (macosEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := macColors(l)
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

// DrawAccordionHeader is a disclosure section (Finder's Get Info): the
// triangle and the bold title, no chrome.
func (e macosEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := macColors(l)
	b = macSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	if st.Pressed() && !st.Disabled() {
		macFill(ctx, b, l.rx(4), paintengine2d.Fill(c.wash))
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
		c.halo(l, ctx, b.Inset(macHaloW(l)), b, l.rx(4))
	}
}

func (macosEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := macColors(l)
	px := macPx(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(c.sep))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(c.sep))
}

// DrawSplitter is NSSplitView's thin divider.
func (macosEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := macColors(l)
	px := macPx(l)
	col := c.sep
	if c.dark {
		col = paintengine2d.RGB(0, 0, 0)
	}
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(col))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(col))
}

// DrawTooltip is the help tag: a light rounded tag with a hairline edge.
func (macosEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := macColors(l)
	b = macSnap(b)
	r := min(l.rx(1), b.Dy()*0.5)
	if c.bigSur {
		r = min(l.rx(4), b.Dy()*0.5)
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.tip))
	macRing(ctx, b, r, macPx(l), paintengine2d.Fill(c.tipEdge))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(8)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs --------------------------------------------------------------------------------------

func macPack(name, label string, year int, summary string, fam ThemeName, sc macScheme, metrics ChromeMetrics, params map[string]float32) ThemePack {
	h := func(k string) paintengine2d.Color { return Hex(sc[k]) }
	win, content := h("win"), h("content")
	accent := h("accent")
	pal := Palette{
		Background: win, Surface: win, SurfaceAlt: h("box"),
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.3),
		Border:  h("fieldEdge"), Divider: h("sep"),
		Text: h("text"), TextMuted: h("text2"), TextOnAccent: h("onAccent"),
		Accent: accent, AccentHover: Shade(accent, 0.1), AccentPress: h("accentPress"),
		Danger: h("red"), Success: h("green"), Warning: h("orange"),
		Track: h("slider"), Thumb: h("knob"),
		Field: content, FieldBorder: h("fieldEdge"),
		Focus: accent, Selection: h("textSel"),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.25), Highlight: h("wash"),
		MenuHover: h("menuHi"), MenuHoverBorder: h("menuHi"), MenuGutter: content,
		BevelLight: Hex("#ffffff"), BevelDark: h("fieldEdge"),
	}
	if fam == ThemeDark {
		// Views fill with the content colour (#1e1e1e); text fields paint
		// their own lighter face over the window.
		pal.BevelLight = h("btnTop")
	}
	tok := ThemeTokens{
		Engine:  "macos",
		Bevel:   BevelSoftShadow,
		Family:  fam,
		Palette: pal,
		Metrics: metrics,
		Params:  params,
		Extra:   map[string]paintengine2d.Color{"selectionText": h("text")},
	}
	if fam == ThemeDark {
		tok.Extra["selectionText"] = Hex("#ffffff")
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: h("sel"), Border: h("sel")}
	tok.Focus = ChromeState{Fill: accent.WithAlpha(0.2), Border: accent}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Mac OS", Summary: summary,
		Era: "macOS", Palette: fam, Tokens: tok,
	}
}

func macosPacks() []ThemePack {
	bigSur := ChromeMetrics{
		Radius: 8, RadiusSmall: 6, Radio: 20, SliderH: 28,
		RowH: 28, HeaderH: 28, MenuItemH: 24, TitleBar: 32, ToolBarH: 44, TabH: 32,
	}
	return []ThemePack{
		macPack("yosemite", "OS X Yosemite", 2014,
			"Yosemite to Mojave: flat white buttons, the blue accent, overlay scrollers, vibrant menus.",
			ThemeLight, macYosemite, ChromeMetrics{}, map[string]float32{"era": 0}),
		macPack("bigsur", "macOS Big Sur", 2020,
			"Big Sur: rounder controls, the unified toolbar, rounded inset selections in lists and sidebars.",
			ThemeLight, macBigSur, bigSur, map[string]float32{"era": 1}),
		macPack("bigsur-night", "macOS Big Sur Dark", 2020,
			"Big Sur's dark appearance: grey faces on charcoal, the brighter blue accent.",
			ThemeDark, macBigSurDark, bigSur, map[string]float32{"era": 1}),
	}
}
