package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// flatlafEngine paints FlatLaf (2019), the flat Swing look and feel, from
// its published UI defaults taken as numbers — the resolved colours of
// FlatLightLaf, FlatDarkLaf and FlatDarculaLaf, their arcs and focus widths
// — and its documented anatomy:
//
//   - buttons are flat white (or grey) faces in a 1px border with an arc of
//     6 (a 3px radius; FlatLaf's arcs are diameters); the pointer lifts the
//     border to the focused-border blue, a press greys the face; keyboard
//     focus is a 2px accent border (the focused border and an inner focus
//     line in Light and Dark; Darcula keeps a 2px ring outside the border
//     in a transparent focus margin every control reserves);
//   - the default button is ringed by a 2px accent border in Light and
//     filled with the dark blue, in bold, in Dark and Darcula;
//   - check boxes are 15px icons with an arc of 4 (a 2px radius): Light
//     marks a selected box with an accent border and an accent tick,
//     Dark and Darcula with a grey tick; radios carry an 8px (Darcula 5px)
//     dot;
//   - scroll bars are thin, arrow-less gutters: a pill thumb 2px inside a
//     10px track that darkens under the pointer;
//   - tabs are underlined: a 3px bar under the selected tab, bright while
//     the tab bar has focus and pale otherwise, a grey wash under the
//     pointer and a 1px separator under the strip;
//   - combos are the text field's face (an arc of 5) with a square arrow
//     button: a chevron (Light, Dark) or a triangle (Darcula);
//   - menus are square white or charcoal popups in a 1px border with a soft
//     drop shadow to the right and below, full-width accent selections;
//   - text fields are square; lists, trees and tables select full rows,
//     grey once the view loses focus.
//
// Pack data: every colour key of flatLight can be overridden through
// "extra"; params "focusWidth" (the outer focus ring, 0 or 2),
// "innerFocus" (the inner focus line), "triangles" (1: triangle arrows),
// "radioDot" (the radio dot's diameter) and "boldDefault" (1: the default
// button is bold).
type flatlafEngine struct{ BaseEngine }

func init() {
	RegisterEngine(flatlafEngine{})
	for _, p := range flatlafPacks() {
		RegisterPack(p)
	}
}

func (flatlafEngine) ID() string { return "flatlaf" }

// DefaultMetrics are FlatLaf's sizes at the toolkit's 16px UI font
// (FlatLaf's reference is 12px Segoe UI): 30px buttons and fields, 36px
// tabs, 10px scroll bars, 15px check boxes in 18px cells.
func (flatlafEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    3, RadiusSmall: 2,
		ControlH: 30, FieldH: 30, ComboH: 30,
		Checkbox: 18, Radio: 18,
		MenuItemH: 26, MenuBarH: 28, TabH: 36, RowH: 24,
		TitleBar: 30, HeaderH: 28, ProgressH: 12, SliderH: 24, Thumb: 12,
		Scroll: 10, Pad: 12, FieldPad: 7, FocusWidth: 2, Border: 1,
		ToolBarH: 36, StatusBarH: 26, SpinnerW: 22, SwitchW: 36, SwitchH: 20,
	}
}

// ---- colour tables ------------------------------------------------------------------------------

// flatScheme maps a colour key to "#rrggbb" or "#rrggbbaa".
type flatScheme map[string]string

// flatLight is FlatLightLaf.
var flatLight = flatScheme{
	// Window, text, selections, accent, separators.
	"bg": "#f2f2f2", "text": "#000000", "textDis": "#808080",
	"sel": "#2675bf", "selText": "#ffffff", "selOff": "#d3d3d3", "selOffText": "#000000",
	"accent": "#2675bf", "sep": "#cecece",
	// Component borders and focus.
	"border": "#c2c2c2", "borderDis": "#cecece", "borderFocus": "#89b0d4", "focus": "#98c3eb",
	// Fields.
	"field": "#ffffff", "fieldDis": "#f2f2f2",
	// Buttons: face, border, hover, pressed, focused, disabled.
	"btn": "#ffffff", "btnBorder": "#c2c2c2", "btnHover": "#f7f7f7", "btnHoverBorder": "#89b0d4",
	"btnPress": "#e6e6e6", "btnFocus": "#eaf3fb", "btnFocusBorder": "#89b0d4", "btnDis": "#f2f2f2", "btnDisBorder": "#cecece",
	// The default button.
	"def": "#ffffff", "defText": "#000000", "defBorder": "#4e9de7", "defHover": "#f7f7f7", "defHoverBorder": "#89b0d4",
	"defPress": "#e6e6e6", "defFocus": "#eaf3fb", "defFocusBorder": "#89b0d4", "defRing": "#98c3eb",
	// Tool bar buttons and latched toggles.
	"toolHover": "#e0e0e0", "toolPress": "#d9d9d9", "toolSel": "#cccccc", "icon": "#6e6e6e",
	// Check boxes and radios.
	"chk": "#ffffff", "chkBorder": "#afafaf", "chkHover": "#f7f7f7", "chkHoverBorder": "#7b9ebf",
	"chkPress": "#e6e6e6", "chkFocus": "#eaf3fb", "chkFocusBorder": "#7b9ebf",
	"chkSel": "#ffffff", "chkSelBorder": "#4e9de7", "chkMark": "#4e9de7",
	"chkDis": "#f2f2f2", "chkDisBorder": "#bfbfbf", "chkDisMark": "#a8a8a8",
	// Scroll bars.
	"thumb": "#cfcfcf", "thumbHover": "#b6b6b6", "thumbPress": "#9c9c9c", "track": "#f5f5f5", "trackHover": "#ededed",
	// Tabs.
	"tabLine": "#3c83c5", "tabLineOff": "#97bbdc", "tabLineDis": "#ababab", "tabHover": "#e0e0e0", "tabFocus": "#dee6ed", "tabSep": "#c2c2c2",
	// Arrows (combo, spinner, submenu).
	"arrow": "#666666", "arrowHover": "#999999", "arrowPress": "#b3b3b3", "arrowDis": "#a6a6a6", "editBtn": "#fafafa",
	// Menus.
	"menuBar": "#ffffff", "menuBarHover": "#e6e6e6", "menuBarLine": "#cecece", "menu": "#ffffff", "menuBorder": "#aeaeae",
	"accel": "#4d4d4d", "menuCheck": "#4e9de7",
	// Tables and trees.
	"hdrSep": "#e6e6e6", "hdrHover": "#f2f2f2", "hdrPress": "#e6e6e6", "cellFocus": "#15416a", "treeIcon": "#b1b1b1",
	// Progress bars and sliders.
	"progress": "#2285e1", "progressTrack": "#d1d1d1",
	"slider": "#2285e1", "sliderHover": "#1c78ce", "sliderPress": "#1a70c0", "sliderTrack": "#c4c4c4", "sliderDis": "#d1d1d1", "sliderHalo": "#94c1ea80",
	// Tool tips.
	"tip": "#fafafa", "tipText": "#000000", "tipBorder": "#919191",
	// Window decorations.
	"close": "#c42b1c", "titleText": "#000000", "titleOff": "#808080",
	// Latched toggle buttons, the split pane grip.
	"toggleSel": "#cccccc", "grip": "#afafaf",
	// Action colours.
	"red": "#db5860", "green": "#59a869", "yellow": "#eda200",
}

// flatDark is FlatDarkLaf (and, with a 2px focus ring and triangles,
// FlatDarculaLaf).
var flatDark = flatScheme{
	"bg": "#3c3f41", "text": "#dddddd", "textDis": "#a6a6a6",
	"sel": "#4b6eaf", "selText": "#eeeeee", "selOff": "#0f2a3d", "selOffText": "#dddddd",
	"accent": "#4b6eaf", "sep": "#595c5e",
	"border": "#616365", "borderDis": "#616365", "borderFocus": "#446e9e", "focus": "#3c628c",
	"field": "#46494b", "fieldDis": "#3c3f41",
	"btn": "#4e5052", "btnBorder": "#606263", "btnHover": "#55585a", "btnHoverBorder": "#446e9e",
	"btnPress": "#5d5f62", "btnFocus": "#4e5052", "btnFocusBorder": "#446e9e", "btnDis": "#3c3f41", "btnDisBorder": "#606263",
	"def": "#375a81", "defText": "#dddddd", "defBorder": "#557394", "defHover": "#3c618c", "defHoverBorder": "#5b7898",
	"defPress": "#406996", "defFocus": "#375a81", "defFocusBorder": "#5b7898", "defRing": "#416997",
	"toolHover": "#505355", "toolPress": "#585a5c", "toolSel": "#5f6264", "icon": "#afb1b3",
	"chk": "#46494b", "chkBorder": "#696b6d", "chkHover": "#4d5153", "chkHoverBorder": "#446e9e",
	"chkPress": "#55585b", "chkFocus": "#455569", "chkFocusBorder": "#446e9e",
	"chkSel": "#46494b", "chkSelBorder": "#87898a", "chkMark": "#c7c7c7",
	"chkDis": "#3c3f41", "chkDisBorder": "#545657", "chkDisMark": "#878787",
	"thumb": "#62696c", "thumbHover": "#7a8387", "thumbPress": "#888f93", "track": "#3e4244", "trackHover": "#484c4f",
	"tabLine": "#4c87c8", "tabLineOff": "#466a92", "tabLineDis": "#747a7e", "tabHover": "#303234", "tabFocus": "#404b5d", "tabSep": "#616365",
	"arrow": "#b7b7b7", "arrowHover": "#d1d1d1", "arrowPress": "#eaeaea", "arrowDis": "#777777", "editBtn": "#414446",
	"menuBar": "#303234", "menuBarHover": "#484c4f", "menuBarLine": "#595c5e", "menu": "#303234", "menuBorder": "#5d6061",
	"accel": "#b7b7b7", "menuCheck": "#b7b7b7",
	"hdrSep": "#5f6365", "hdrHover": "#525658", "hdrPress": "#5f6365", "cellFocus": "#6d8ac0", "treeIcon": "#cecece",
	"progress": "#4c87c8", "progressTrack": "#505456",
	"slider": "#4c87c8", "sliderHover": "#6094ce", "sliderPress": "#6b9cd2", "sliderTrack": "#616669", "sliderDis": "#54595c", "sliderHalo": "#7097c24d",
	"tip": "#1e2021", "tipText": "#dddddd", "tipBorder": "#1e2021",
	"close": "#c42b1c", "titleText": "#dddddd", "titleOff": "#a6a6a6",
	"toggleSel": "#676a6c", "grip": "#adadad",
	"red": "#c75450", "green": "#499c54", "yellow": "#f0a732",
}

// ---- resolved colour set ------------------------------------------------------------------------

// flatSet is a look's resolved FlatLaf colours and geometry (built once per
// look).
type flatSet struct {
	dark bool

	bg, text, textDis, sel, selText, selOff, selOffText, accent, sep paintengine2d.Color
	border, borderDis, borderFocus, focus, field, fieldDis           paintengine2d.Color
	btn, btnBorder, btnHover, btnHoverBorder, btnPress               paintengine2d.Color
	btnFocus, btnFocusBorder, btnDis, btnDisBorder                   paintengine2d.Color
	def, defText, defBorder, defHover, defHoverBorder                paintengine2d.Color
	defPress, defFocus, defFocusBorder, defRing                      paintengine2d.Color
	toolHover, toolPress, toolSel, icon                              paintengine2d.Color
	chk, chkBorder, chkHover, chkHoverBorder, chkPress               paintengine2d.Color
	chkFocus, chkFocusBorder, chkSel, chkSelBorder, chkMark          paintengine2d.Color
	chkDis, chkDisBorder, chkDisMark                                 paintengine2d.Color
	thumb, thumbHover, thumbPress, track, trackHover                 paintengine2d.Color
	tabLine, tabLineOff, tabLineDis, tabHover, tabFocus, tabSep      paintengine2d.Color
	arrow, arrowHover, arrowPress, arrowDis, editBtn                 paintengine2d.Color
	menuBar, menuBarHover, menuBarLine, menu, menuBorder             paintengine2d.Color
	accel, menuCheck, hdrSep, hdrHover, hdrPress, cellFocus          paintengine2d.Color
	treeIcon, progress, progressTrack                                paintengine2d.Color
	slider, sliderHover, sliderPress, sliderTrack, sliderDis         paintengine2d.Color
	sliderHalo, tip, tipText, tipBorder                              paintengine2d.Color
	close, titleText, titleOff, toggleSel, grip                      paintengine2d.Color

	// Geometry: the outer focus ring (Darcula's reserved margin), the inner
	// focus line, triangle arrows, the radio dot, a bold default button.
	ring, inner     float32
	triangles, bold bool
	radioDot        float32
	defBorderW      float32
	boldFace        *Font
}

type flatKey struct{}

func flatColors(l *Classic) *flatSet {
	return l.Memo(flatKey{}, func() any { return flatBuild(l) }).(*flatSet)
}

func flatBuild(l *Classic) *flatSet {
	dark := Luma(l.palette.Background) < 0.5
	sc := flatLight
	if dark {
		sc = flatDark
	}
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	c := &flatSet{dark: dark}
	c.bg, c.text, c.textDis = col("bg"), col("text"), col("textDis")
	c.sel, c.selText, c.selOff, c.selOffText = col("sel"), col("selText"), col("selOff"), col("selOffText")
	c.accent, c.sep = col("accent"), col("sep")
	c.border, c.borderDis, c.borderFocus, c.focus = col("border"), col("borderDis"), col("borderFocus"), col("focus")
	c.field, c.fieldDis = col("field"), col("fieldDis")
	c.btn, c.btnBorder, c.btnHover, c.btnHoverBorder = col("btn"), col("btnBorder"), col("btnHover"), col("btnHoverBorder")
	c.btnPress, c.btnFocus, c.btnFocusBorder = col("btnPress"), col("btnFocus"), col("btnFocusBorder")
	c.btnDis, c.btnDisBorder = col("btnDis"), col("btnDisBorder")
	c.def, c.defText, c.defBorder, c.defHover, c.defHoverBorder = col("def"), col("defText"), col("defBorder"), col("defHover"), col("defHoverBorder")
	c.defPress, c.defFocus, c.defFocusBorder, c.defRing = col("defPress"), col("defFocus"), col("defFocusBorder"), col("defRing")
	c.toolHover, c.toolPress, c.toolSel, c.icon = col("toolHover"), col("toolPress"), col("toolSel"), col("icon")
	c.chk, c.chkBorder, c.chkHover, c.chkHoverBorder = col("chk"), col("chkBorder"), col("chkHover"), col("chkHoverBorder")
	c.chkPress, c.chkFocus, c.chkFocusBorder = col("chkPress"), col("chkFocus"), col("chkFocusBorder")
	c.chkSel, c.chkSelBorder, c.chkMark = col("chkSel"), col("chkSelBorder"), col("chkMark")
	c.chkDis, c.chkDisBorder, c.chkDisMark = col("chkDis"), col("chkDisBorder"), col("chkDisMark")
	c.thumb, c.thumbHover, c.thumbPress, c.track, c.trackHover = col("thumb"), col("thumbHover"), col("thumbPress"), col("track"), col("trackHover")
	c.tabLine, c.tabLineOff, c.tabLineDis = col("tabLine"), col("tabLineOff"), col("tabLineDis")
	c.tabHover, c.tabFocus, c.tabSep = col("tabHover"), col("tabFocus"), col("tabSep")
	c.arrow, c.arrowHover, c.arrowPress, c.arrowDis, c.editBtn = col("arrow"), col("arrowHover"), col("arrowPress"), col("arrowDis"), col("editBtn")
	c.menuBar, c.menuBarHover, c.menuBarLine = col("menuBar"), col("menuBarHover"), col("menuBarLine")
	c.menu, c.menuBorder, c.accel, c.menuCheck = col("menu"), col("menuBorder"), col("accel"), col("menuCheck")
	c.hdrSep, c.hdrHover, c.hdrPress, c.cellFocus, c.treeIcon = col("hdrSep"), col("hdrHover"), col("hdrPress"), col("cellFocus"), col("treeIcon")
	c.progress, c.progressTrack = col("progress"), col("progressTrack")
	c.slider, c.sliderHover, c.sliderPress = col("slider"), col("sliderHover"), col("sliderPress")
	c.sliderTrack, c.sliderDis, c.sliderHalo = col("sliderTrack"), col("sliderDis"), col("sliderHalo")
	c.tip, c.tipText, c.tipBorder = col("tip"), col("tipText"), col("tipBorder")
	c.close, c.titleText, c.titleOff = col("close"), col("titleText"), col("titleOff")
	c.toggleSel, c.grip = col("toggleSel"), col("grip")
	c.ring = l.P("focusWidth", 0)
	c.inner = l.P("innerFocus", 1)
	c.triangles = l.P("triangles", 0) != 0
	c.bold = l.P("boldDefault", 0) != 0
	c.radioDot = l.P("radioDot", 8)
	c.defBorderW = l.P("defBorder", 1)
	c.boldFace = l.BoldFont()
	return c
}

// ---- helpers ------------------------------------------------------------------------------------

// flatPx is one device pixel: 1 at 1x, 2 at 2x.
func flatPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
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

// flatRing fills the lw-wide band just inside the round rect b.
func flatRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 || lw <= 0 {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	if b.Dx() <= 2*lw || b.Dy() <= 2*lw {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(b, r, r)
	ri := max(r-lw, 0)
	p.AddRoundRect(b.Inset(lw), ri, ri)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
}

// flatCentered is a w×h rect centred in b, on whole pixels.
func flatCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return flatSnap(paintengine2d.XYWH((b.Min.X+b.Max.X-w)*0.5, (b.Min.Y+b.Max.Y-h)*0.5, w, h))
}

// face is a bordered component's visible body inside its rect: Darcula
// keeps a transparent margin for its outer focus ring.
func (c *flatSet) face(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = flatSnap(b)
	m := snap(l.S(c.ring))
	if m <= 0 || b.Dx() < 4*m || b.Dy() < 4*m {
		return b
	}
	return b.Inset(m)
}

// arcR turns a FlatLaf arc (a corner diameter) into a radius.
func flatArcR(l *Classic, arc float32) float32 { return l.rx(arc * 0.5) }

// frame paints a bordered component: fill, border and — when focused —
// the focus border (the focused border line plus the inner focus line) or
// Darcula's outer ring.
func (c *flatSet) frame(l *Classic, ctx *paintengine2d.Context, b, f paintengine2d.Rect, r float32, fill, border paintengine2d.Color, bw float32, focused bool, ring paintengine2d.Color) {
	if f.Dx() < 2 || f.Dy() < 2 {
		return
	}
	px := flatPx(l)
	ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(fill))
	flatRing(ctx, f, r, bw*px, border)
	if !focused {
		return
	}
	if c.ring > 0 {
		m := f.Min.X - flatSnap(b).Min.X
		if m > 0 {
			o := f.Inset(-m)
			flatRing(ctx, o, r+m, m, ring)
		}
		return
	}
	if c.inner > 0 && f.Dx() > 4*px && f.Dy() > 4*px {
		flatRing(ctx, f.Inset(bw*px), max(r-bw*px, 0), c.inner*px, ring)
	}
}

// flatTick strokes FlatLaf's two-segment check mark into g.
func flatTick(ctx *paintengine2d.Context, g paintengine2d.Rect, col paintengine2d.Color, w float32) {
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X+g.Dx()*0.2, g.Min.Y+g.Dy()*0.52)
	p.LineTo(g.Min.X+g.Dx()*0.42, g.Min.Y+g.Dy()*0.74)
	p.LineTo(g.Min.X+g.Dx()*0.82, g.Min.Y+g.Dy()*0.3)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// arrow paints FlatLaf's arrow into b: an 8×4 chevron (Light, Dark) or a
// filled 9×5 triangle (Darcula).
func (c *flatSet) arrowGlyph(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	w, h := l.S(8), l.S(4)
	if c.triangles {
		w, h = l.S(9), l.S(5)
	}
	if dir == DirLeft || dir == DirRight {
		w, h = h, w
	}
	if b.Dx() < w || b.Dy() < h {
		k := min(b.Dx()/w, b.Dy()/h)
		if k <= 0.2 {
			return
		}
		w, h = w*k, h*k
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-w*0.5, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy+h*0.5)
	case DirDown:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
		p.LineTo(cx+w*0.5, cy-h*0.5)
	case DirLeft:
		p.MoveTo(cx+w*0.5, cy-h*0.5)
		p.LineTo(cx-w*0.5, cy)
		p.LineTo(cx+w*0.5, cy+h*0.5)
	default:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy)
		p.LineTo(cx-w*0.5, cy+h*0.5)
	}
	if c.triangles {
		p.Close()
		ctx.DrawPath(p, paintengine2d.Fill(col))
		return
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: max(l.S(1), 1), Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// ---- parts --------------------------------------------------------------------------------------

// button paints a push button face and returns its label colour.
func (c *flatSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	f := c.face(l, b)
	r := flatArcR(l, 6)
	focused := st.Focused() && !st.Disabled()
	if st.Primary() && !st.Disabled() {
		fill, border, bw := c.def, c.defBorder, c.defBorderW
		switch {
		case st.Pressed():
			fill, border = c.defPress, c.defHoverBorder
		case st.Hovered():
			fill, border = c.defHover, c.defHoverBorder
		case focused:
			fill, border = c.defFocus, c.defFocusBorder
		}
		c.frame(l, ctx, b, f, r, fill, border, bw, focused, c.defRing)
		return c.defText
	}
	fill, border := c.btn, c.btnBorder
	switch {
	case st.Disabled():
		fill, border = c.btnDis, c.btnDisBorder
	case st.Pressed():
		fill, border = c.btnPress, c.btnHoverBorder
	case st.Hovered():
		fill, border = c.btnHover, c.btnHoverBorder
	case focused:
		fill, border = c.btnFocus, c.btnFocusBorder
	case st.Toggle() && st.Checked():
		fill = c.toggleSel
	}
	if st.Toggle() && st.Checked() && !st.Disabled() && (st.Hovered() || st.Pressed()) {
		fill = mdOver(c.toggleSel, c.text, 0.06)
	}
	c.frame(l, ctx, b, f, r, fill, border, 1, focused, c.focus)
	if st.Disabled() {
		return c.textDis
	}
	return c.text
}

// fieldFrame paints a text field or combo frame (square fields, arc 5
// combos) and returns the face.
func (c *flatSet) fieldFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, arc float32) paintengine2d.Rect {
	f := c.face(l, b)
	fill, border := c.field, c.border
	focused := (st.Focused() || st.Pressed()) && !st.Disabled()
	switch {
	case st.Disabled():
		fill, border = c.fieldDis, c.borderDis
	case focused:
		border = c.borderFocus
	}
	c.frame(l, ctx, b, f, flatArcR(l, arc), fill, border, 1, focused, c.focus)
	return f
}

func (e flatlafEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := flatColors(l)
	switch role {
	case RoleButton:
		return c.button(l, ctx, b, st)
	case RoleTool:
		return c.tool(l, ctx, b, st)
	case RoleField:
		c.fieldFrame(l, ctx, b, st, 0)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleCombo:
		c.fieldFrame(l, ctx, b, st, 5)
		return c.text
	case RoleCheck:
		c.frame(l, ctx, b, flatSnap(b), flatArcR(l, 4), c.chk, c.chkBorder, 1, false, c.focus)
		return c.text
	case RoleRow:
		return c.row(ctx, flatSnap(b), st)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			ctx.DrawRect(flatSnap(b), paintengine2d.Fill(c.sel))
			return c.selText
		}
		return c.text
	case RoleTab:
		if st.Hovered() && !st.Disabled() {
			ctx.DrawRect(b, paintengine2d.Fill(c.tabHover))
		}
		return c.text
	case RoleThumb:
		ctx.DrawRoundRect(b, min(b.Dx(), b.Dy())*0.5, min(b.Dx(), b.Dy())*0.5, paintengine2d.Fill(c.thumb))
		return c.text
	case RoleTrack:
		ctx.DrawRect(b, paintengine2d.Fill(c.track))
		return c.text
	case RoleBar:
		ctx.DrawRect(b, paintengine2d.Fill(c.menuBar))
		return c.text
	case RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		return c.text
	}
	return c.text
}

// tool paints a tool bar button: borderless, a wash under the pointer, a
// darker one pressed or latched. It returns the glyph colour.
func (c *flatSet) tool(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	b = flatSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return c.icon
	}
	r := flatArcR(l, 6)
	var fill paintengine2d.Color
	switch {
	case st.Disabled():
		if st.Checked() {
			fill = mdOver(c.toolSel, c.bg, 0.5)
		}
	case st.Pressed():
		fill = c.toolPress
	case st.Checked():
		fill = c.toolSel
	case st.Hovered():
		fill = c.toolHover
	}
	if fill.A > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
	}
	if st.Disabled() {
		return c.textDis
	}
	return c.icon
}

// row paints a selected row into box and returns its text colour: the
// accent while the view has focus, grey otherwise.
func (c *flatSet) row(ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState) paintengine2d.Color {
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	if !st.Checked() {
		return fg
	}
	fill, text := c.sel, c.selText
	if st.Inactive() || st.Backdrop() || st.Disabled() {
		fill, text = c.selOff, c.selOffText
	}
	if st.Disabled() {
		text = c.textDis
	}
	ctx.DrawRect(box, paintengine2d.Fill(fill))
	return text
}

func (e flatlafEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := flatColors(l)
	s := snap(min(l.S(15), box.Dx(), box.Dy()))
	if s < 4 {
		return
	}
	g := flatCentered(box, s, s)
	fill, border, mark := c.indicator(st, checked)
	r := flatArcR(l, 4)
	ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(fill))
	flatRing(ctx, g, r, flatPx(l), border)
	if checked {
		flatTick(ctx, g.Inset(s*0.12), mark, max(l.S(1.9), 1.2))
	}
}

// indicator is a check box or radio's fill, border and mark colour.
func (c *flatSet) indicator(st ControlState, on bool) (fill, border, mark paintengine2d.Color) {
	fill, border, mark = c.chk, c.chkBorder, c.chkMark
	if on {
		fill, border = c.chkSel, c.chkSelBorder
	}
	switch {
	case st.Disabled():
		return c.chkDis, c.chkDisBorder, c.chkDisMark
	case st.Pressed():
		fill, border = c.chkPress, c.chkHoverBorder
	case st.Hovered():
		fill, border = c.chkHover, c.chkHoverBorder
	case st.Focused():
		fill, border = c.chkFocus, c.chkFocusBorder
	}
	return fill, border, mark
}

func (e flatlafEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := flatColors(l)
	d := snap(min(l.S(15), box.Dx(), box.Dy()))
	if d < 4 {
		return
	}
	g := flatCentered(box, d, d)
	fill, border, mark := c.indicator(st, selected)
	ctr := paintengine2d.Pt((g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5)
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(border))
	ctx.DrawCircle(ctr, d*0.5-flatPx(l), paintengine2d.Fill(fill))
	if selected {
		ctx.DrawCircle(ctr, min(l.S(c.radioDot), d-2*flatPx(l))*0.5, paintengine2d.Fill(mark))
	}
}

// Arrow is FlatLaf's chevron (or Darcula's triangle).
func (flatlafEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	flatColors(l).arrowGlyph(l, ctx, b, dir, col)
}

// Expander is the tree's 11px chevron (or triangle).
func (flatlafEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := flatColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	c.arrowGlyph(l, ctx, b, dir, col)
}

// MenuHighlight is the full-width accent selection of a menu item and of
// an open menu-bar title.
func (flatlafEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	ctx.DrawRect(flatSnap(b), paintengine2d.Fill(flatColors(l).sel))
}

func (flatlafEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := flatColors(l)
	if hot {
		return c.selText
	}
	return c.text
}

// Fields and views show focus with the focus border.
func (flatlafEngine) FieldFocusRing(l *Classic) bool { return true }

// DrawFocusRing is the focus border inside b: the focused border line and
// the inner focus line (a 2px accent border), or Darcula's 2px ring.
func (flatlafEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := flatColors(l)
	b = flatSnap(b)
	px := flatPx(l)
	if b.Dx() < 4*px || b.Dy() < 4*px {
		return
	}
	r := flatArcR(l, 6)
	if c.ring > 0 {
		flatRing(ctx, b, r, snap(l.S(c.ring)), c.focus)
		return
	}
	flatRing(ctx, b, r, px, c.borderFocus)
	flatRing(ctx, b.Inset(px), max(r-px, 0), px*c.inner, c.focus)
}

// ---- scroll bars --------------------------------------------------------------------------------

// ScrollBarStyle: a 10px gutter, no arrows, the pill thumb 2px inside it.
func (flatlafEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 10, MinThumb: 18}
}

func (flatlafEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := flatColors(l)
	bar := flatSnap(p.Bar)
	if bar.Empty() {
		return
	}
	track := c.track
	if st.Hovered && !st.Disabled {
		track = c.trackHover
	}
	ctx.DrawRect(bar, paintengine2d.Fill(track))
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	col := c.thumb
	switch {
	case st.Pressed == ScrollThumbPart:
		col = c.thumbPress
	case st.Hot == ScrollThumbPart:
		col = c.thumbHover
	}
	in := snap(l.S(2))
	t := flatSnap(p.Thumb)
	if vertical {
		t = paintengine2d.XYWH(bar.Min.X+in, t.Min.Y+in, bar.Dx()-2*in, t.Dy()-2*in)
	} else {
		t = paintengine2d.XYWH(t.Min.X+in, bar.Min.Y+in, t.Dx()-2*in, bar.Dy()-2*in)
	}
	if t.Empty() {
		return
	}
	r := min(t.Dx(), t.Dy()) * 0.5
	ctx.DrawRoundRect(t, r, r, paintengine2d.Fill(col))
}

func (e flatlafEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered() || st.Pressed()}
	if st.Hovered() {
		ss.Hot = ScrollThumbPart
	}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------------------------

// GroupBoxInsets: a titled border — a 1px line broken by its title.
func (flatlafEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top = snap(l.body.Height()) + pad*0.5
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

func (flatlafEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := flatColors(l)
	b = flatSnap(b)
	px := flatPx(l)
	if b.Dx() < 4*px || b.Dy() < 4*px {
		return
	}
	if raised {
		ctx.DrawRect(b, paintengine2d.Fill(mdOver(c.bg, c.field, 0.5)))
	}
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + l.body.Height()*0.5)
	}
	frame := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, top), Max: b.Max}
	if title == "" {
		flatRing(ctx, frame, 0, px, c.sep)
		return
	}
	tx := b.Min.X + l.S(8)
	tw := min(l.body.Advance(title)+l.S(8), b.Dx()-l.S(16))
	ctx.Save()
	p := paintengine2d.NewPath()
	p.AddRect(b)
	p.AddRect(paintengine2d.XYWH(tx, b.Min.Y, tw, l.body.Height()))
	ctx.ClipPathRule(p, paintengine2d.FillEvenOdd)
	flatRing(ctx, frame, 0, px, c.sep)
	ctx.Restore()
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(tx+l.S(4), b.Min.Y, tw-l.S(4), l.body.Height()), c.text, AlignStart, 0)
}

// flatTitleH is FlatLaf's window title bar (its 30px caption buttons).
func flatTitleH(l *Classic) float32 {
	return snap(max(l.S(30), l.body.Height()+l.S(10)))
}

func (flatlafEngine) WindowFrameInsets(l *Classic) Insets {
	px := flatPx(l)
	return Insets{Top: flatTitleH(l), Right: px, Bottom: px, Left: px}
}

// WindowCloseRect is the square-cornered 44×30 close button filling the
// title bar at the right.
func (flatlafEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = flatSnap(b)
	px := flatPx(l)
	h := flatTitleH(l) - px
	w := min(snap(h*44/30), snap(b.Dx()*0.3))
	if b.Dy() < h+px || w < 4 {
		return paintengine2d.Rect{}
	}
	return paintengine2d.XYWH(b.Max.X-px-w, b.Min.Y+px, w, h)
}

// DrawWindowFrame is FlatLaf's decorated window: the title pane in the
// window colour, the title at the left, the close button that turns red
// under the pointer, a 1px border.
func (e flatlafEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := flatColors(l)
	b = flatSnap(b)
	if b.Empty() {
		return
	}
	px := flatPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	right := b.Max.X
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			glyph := c.titleText
			if !st.Active {
				glyph = c.titleOff
			}
			switch {
			case st.ClosePress:
				ctx.DrawRect(cb, paintengine2d.Fill(c.close.WithAlpha(0.9)))
				glyph = paintengine2d.RGB(1, 1, 1)
			case st.CloseHot:
				ctx.DrawRect(cb, paintengine2d.Fill(c.close))
				glyph = paintengine2d.RGB(1, 1, 1)
			}
			s := snap(min(l.S(10), cb.Dy()*0.4))
			DrawCross(ctx, flatCentered(cb, s, s), glyph, max(l.S(1), 1))
			right = cb.Min.X
		}
	}
	flatRing(ctx, b, 0, px, c.border)
	if title == "" {
		return
	}
	col := c.titleText
	if !st.Active {
		col = c.titleOff
	}
	x := b.Min.X + l.S(10)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(x, b.Min.Y, right-x-l.S(6), flatTitleH(l)), col, AlignStart, 0)
}

func (flatlafEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(flatColors(l).bg))
}

// PopupShadow: FlatLaf's drop shadow falls to the right and below (4px,
// 15% black on light themes, 25% on dark ones).
func (flatlafEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp, ok := flatShadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(sp.dx, sp.dy, sp.blur, 0)
}

func (flatlafEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if sp, ok := flatShadow(l, kind); ok {
		DropShadow(ctx, b, 0, sp.col, sp.dx, sp.dy, sp.blur, 0)
	}
}

type flatShadowSpec struct {
	col          paintengine2d.Color
	dx, dy, blur float32
}

func flatShadow(l *Classic, kind PopupKind) (flatShadowSpec, bool) {
	k := l.P("shadow", 1)
	if k <= 0 {
		return flatShadowSpec{}, false
	}
	a := float32(0.15)
	if flatColors(l).dark {
		a = 0.25
	}
	s := flatShadowSpec{col: paintengine2d.RGBA(0, 0, 0, min(a*k*1.6, 0.8)), dx: l.S(2), dy: l.S(2), blur: l.S(6)}
	if kind == PopupDialog {
		s.dx, s.dy, s.blur = l.S(3), l.S(4), l.S(14)
		s.col = paintengine2d.RGBA(0, 0, 0, min(a*k*2, 0.8))
	}
	return s, true
}

// ItemFocus is the cell focus rectangle: a 1px line round the current row.
func (flatlafEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := flatColors(l)
	b = flatSnap(b)
	px := flatPx(l)
	if b.Dx() < 4*px || b.Dy() < 4*px {
		return
	}
	col := c.cellFocus
	if st.Checked() && !st.Inactive() {
		col = c.selText.WithAlpha(0.7)
	}
	flatRing(ctx, b, 0, px, col)
}

// ViewFrameInsets: a 1px scroll pane border (plus Darcula's focus margin).
func (flatlafEngine) ViewFrameInsets(l *Classic) Insets {
	c := flatColors(l)
	v := flatPx(l) + snap(l.S(c.ring))
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is the scroll pane's border round a list, tree or table;
// it turns the focused-border blue while the view has focus.
func (flatlafEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := flatColors(l)
	b = flatSnap(b)
	f := c.face(l, b)
	if f.Dx() < 4 || f.Dy() < 4 {
		return
	}
	focused := st.Focused() && !st.Disabled()
	border := c.border
	if focused {
		border = c.borderFocus
	}
	if c.ring > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	}
	ctx.DrawRect(f, paintengine2d.Fill(c.field))
	flatRing(ctx, f, 0, flatPx(l), border)
	if focused && c.ring > 0 {
		flatRing(ctx, b, 0, f.Min.X-b.Min.X, c.focus)
	}
}

// StyleHint: Swing's option panes put OK first (Windows and Linux); tabs
// and form labels start at the left; FlatLaf switches states at once.
func (flatlafEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	if h == HintMnemonics {
		return MnemonicsOnAlt // FlatLaf hides them until Alt
	}
	return 0
}

// SpinBoxStyle: the arrow buttons sit inside the spinner's border.
func (flatlafEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{Inside: true} }

func (flatlafEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(flatColors(l).bg))
}

// ---- controls -----------------------------------------------------------------------------------

func (e flatlafEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := flatColors(l)
	fg := c.button(l, ctx, b, st)
	f := l.body
	if st.Primary() && c.bold && !st.Disabled() {
		f = c.boldFace
	}
	l.drawFittedText(ctx, f, label, c.face(l, b), fg, AlignCenter, l.S(12))
}

func (e flatlafEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := flatColors(l)
	in := flatSnap(b).Inset(flatPx(l))
	fg := c.tool(l, ctx, in, st)
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
		if st.Disabled() {
			tc = c.textDis
		}
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x-pad*0.5, b.Dy()), tc, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		// A 1.5px focus outline.
		flatRing(ctx, in, flatArcR(l, 6), max(l.S(1.5), 1), c.focus)
	}
}

// flatToggleLayout puts the 15px icon at the left of b and the label four
// pixels after it.
func flatToggleLayout(l *Classic, b paintengine2d.Rect, side float32) (cell, lb paintengine2d.Rect) {
	side = min(side, b.Dx())
	cell = flatSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	x := cell.Max.X + l.S(4)
	lb = paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy())
	return cell, lb
}

// iconFocus rings a focused check box or radio icon (1px, Darcula 2px).
func (c *flatSet) iconFocus(l *Classic, ctx *paintengine2d.Context, cell, clip paintengine2d.Rect, round bool) {
	s := snap(min(l.S(15), cell.Dx(), cell.Dy()))
	g := flatCentered(cell, s, s)
	w := snap(l.S(max(c.ring, 1)))
	o := g.Inset(-w)
	r := flatArcR(l, 4) + w
	if round {
		r = o.Dx() * 0.5
	}
	ctx.Save()
	ctx.ClipRect(clip)
	flatRing(ctx, o, r, w, c.focus)
	ctx.Restore()
}

func (e flatlafEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := flatColors(l)
	cell, lb := flatToggleLayout(l, b, l.metrics.Checkbox)
	e.CheckIndicator(l, ctx, cell, st, checked)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		c.iconFocus(l, ctx, cell, b, false)
	}
}

func (e flatlafEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := flatColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	cell, lb := flatToggleLayout(l, b, side)
	e.RadioIndicator(l, ctx, cell, st, selected)
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		c.iconFocus(l, ctx, cell, b, true)
	}
}

func (c *flatSet) toggleLabel(l *Classic, ctx *paintengine2d.Context, lb paintengine2d.Rect, st ControlState, label string) {
	if label == "" || lb.Dx() <= 0 {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

// DrawSwitch: Swing has no switch; this one is drawn in FlatLaf's parts —
// a bordered pill that fills with the accent when on, a round knob.
func (e flatlafEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := flatColors(l)
	sw, sh := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	track := flatSnap(paintengine2d.XYWH(b.Min.X+snap(l.S(c.ring)), b.Min.Y+(b.Dy()-sh)*0.5, min(sw, b.Dx()), sh))
	px := flatPx(l)
	if track.Dx() < 6*px || track.Dy() < 6*px {
		return
	}
	r := track.Dy() * 0.5
	fill, border, knob := c.field, c.border, c.chkBorder
	switch {
	case st.Disabled():
		fill, border, knob = c.fieldDis, c.borderDis, c.chkDisBorder
		if on {
			fill, knob = c.sliderDis, c.fieldDis
		}
	case on:
		fill, border, knob = c.slider, c.slider, paintengine2d.RGB(1, 1, 1)
		if st.Pressed() {
			fill, border = c.sliderPress, c.sliderPress
		} else if st.Hovered() {
			fill, border = c.sliderHover, c.sliderHover
		}
	case st.Hovered() || st.Pressed():
		border = c.chkHoverBorder
	}
	ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(fill))
	flatRing(ctx, track, r, px, border)
	d := track.Dy() - 6*px
	cx := track.Min.X + 3*px + d*0.5
	if on {
		cx = track.Max.X - 3*px - d*0.5
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, (track.Min.Y+track.Max.Y)*0.5), d*0.5, paintengine2d.Fill(knob))
	lb := paintengine2d.XYWH(track.Max.X+l.S(6), b.Min.Y, b.Max.X-track.Max.X-l.S(6), b.Dy())
	c.toggleLabel(l, ctx, lb, st, label)
	if st.Focused() && !st.Disabled() {
		w := snap(l.S(max(c.ring, 1)))
		ctx.Save()
		ctx.ClipRect(b)
		flatRing(ctx, track.Inset(-w), r+w, w, c.focus)
		ctx.Restore()
	}
}

// DrawSlider is FlatLaf's slider: a 2px round-ended track, the accent up
// to a 12px round thumb that darkens under the pointer; focus is a 4px
// halo round the thumb.
func (e flatlafEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := flatColors(l)
	b = flatSnap(b)
	t = clamp1(t)
	halo := snap(l.S(4))
	d := min(snap(l.S(12)), b.Dy()-2*halo)
	if d < 4 || b.Dx() < d+2*halo {
		return
	}
	th := max(snap(l.S(2)), 1)
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	x0, x1 := b.Min.X+halo+d*0.5, b.Max.X-halo-d*0.5
	tx := x0 + (x1-x0)*t
	track := paintengine2d.XYWH(x0, cy-th*0.5, x1-x0, th)
	ctx.DrawRoundRect(track, th*0.5, th*0.5, paintengine2d.Fill(c.sliderTrack))
	thumb := c.slider
	switch {
	case st.Disabled():
		thumb = c.sliderDis
	case st.Pressed():
		thumb = c.sliderPress
	case st.Hovered():
		thumb = c.sliderHover
	}
	if tx > x0 && !st.Disabled() {
		ctx.DrawRoundRect(paintengine2d.XYWH(x0, cy-th*0.5, tx-x0, th), th*0.5, th*0.5, paintengine2d.Fill(c.slider))
	}
	ctr := paintengine2d.Pt(tx, cy)
	if st.Focused() && !st.Disabled() {
		ctx.DrawCircle(ctr, d*0.5+halo, paintengine2d.Fill(c.sliderHalo))
	}
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(thumb))
}

// DrawProgressBar is a 4px fully rounded bar (arc 4) over its track; the
// busy bar slides a segment.
func (e flatlafEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := flatColors(l)
	b = flatSnap(b)
	h := min(snap(l.S(4)), b.Dy())
	if b.Dx() < 4 || h < 1 {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	r := h * 0.5
	ctx.DrawRoundRect(bar, r, r, paintengine2d.Fill(c.progressTrack))
	fill := c.progress
	if st.Disabled() {
		fill = c.sliderDis
	}
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.3
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, bar.Min.Y, span, h).Intersect(bar)
		if seg.Dx() >= 1 {
			ctx.DrawRoundRect(seg, r, r, paintengine2d.Fill(fill))
		}
		return
	}
	if w := snap(bar.Dx() * clamp1(t)); w >= 1 {
		ctx.DrawRoundRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, r, paintengine2d.Fill(fill))
	}
}

func (e flatlafEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := flatColors(l)
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	f := c.fieldFrame(l, ctx, b, st, 0)
	l.baseDrawTextField(ctx, f, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

// DrawComboBox is a JComboBox: the field face with an arc of 5 and the
// square arrow button at its right (no separator when not editable).
func (e flatlafEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := flatColors(l)
	if open {
		st |= StatePressed
	}
	f := c.fieldFrame(l, ctx, b, st, 5)
	if f.Dx() < 8 || f.Dy() < 6 {
		return
	}
	px := flatPx(l)
	bw := snap(min(f.Dy()-2*px, f.Dx()*0.4))
	btn := paintengine2d.XYWH(f.Max.X-px-bw, f.Min.Y+px, bw, f.Dy()-2*px)
	col := c.arrow
	switch {
	case st.Disabled():
		col = c.arrowDis
	case open || st.Pressed():
		col = c.arrowPress
	case st.Hovered():
		col = c.arrowHover
	}
	c.arrowGlyph(l, ctx, btn, DirDown, col)
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	pad := l.S(6) + px
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(f.Min.X+pad, f.Min.Y, btn.Min.X-f.Min.X-pad, f.Dy()), fg, AlignStart, 0)
}

// DrawSpinner: the spinner's arrow buttons, stacked at the right inside its
// border, a separator against the text.
func (e flatlafEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := flatColors(l)
	b = flatSnap(b)
	px := flatPx(l)
	if b.Dx() < 4*px || b.Dy() < 6*px {
		return
	}
	if !st.Frameless() {
		c.fieldFrame(l, ctx, b, st&^StateFocused, 5)
		b = c.face(l, b).Inset(px)
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.editBtn))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, px, b.Dy()), paintengine2d.Fill(c.border))
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, dir Direction, hot, press bool) {
		col := c.arrow
		switch {
		case st.Disabled():
			col = c.arrowDis
		case press:
			col = c.arrowPress
		case hot:
			col = c.arrowHover
		}
		c.arrowGlyph(l, ctx, r, dir, col)
	}
	half(paintengine2d.XYWH(b.Min.X+px, b.Min.Y, b.Dx()-px, mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X+px, mid, b.Dx()-px, b.Max.Y-mid), DirDown, downHover, downPress)
}

func (flatlafEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := flatColors(l)
	b = flatSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	px := flatPx(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.tabSep))
}

// DrawTab is an underlined tab: a wash under the pointer, the focus wash
// on the selected tab while the bar has focus, and the 3px underline —
// bright with focus, pale without.
func (flatlafEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := flatColors(l)
	b = flatSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	px := flatPx(l)
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-px)
	switch {
	case st.Disabled():
	case st.Hovered() || st.Pressed():
		ctx.DrawRect(body, paintengine2d.Fill(c.tabHover))
	case selected && st.Focused():
		ctx.DrawRect(body, paintengine2d.Fill(c.tabFocus))
	}
	if selected {
		col := c.tabLineOff
		switch {
		case st.Disabled():
			col = c.tabLineDis
		case st.Focused():
			col = c.tabLine
		}
		h := snap(l.S(3))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-h, b.Dx(), h), paintengine2d.Fill(col))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, label, body, fg, AlignCenter, l.S(12))
}

func (flatlafEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := flatColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if raised {
		flatRing(ctx, flatSnap(b), 0, flatPx(l), c.sep)
	}
}

func (flatlafEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := flatColors(l)
	b = flatSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.menuBar))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-flatPx(l), b.Dx(), flatPx(l)), paintengine2d.Fill(c.menuBarLine))
}

// DrawMenuTitle: a grey wash under the pointer, the accent selection when
// its menu is open.
func (e flatlafEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := flatColors(l)
	b = flatSnap(b)
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-flatPx(l))
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.textDis
	case open || st.Pressed():
		ctx.DrawRect(body, paintengine2d.Fill(c.sel))
		fg = c.selText
	case st.Hovered():
		ctx.DrawRect(body, paintengine2d.Fill(c.menuBarHover))
	}
	l.drawLabeled(ctx, l.body, label, underline, body, fg)
	if st.Focused() && !open {
		flatRing(ctx, body, 0, flatPx(l), c.focus)
	}
}

// DrawMenuFrame is a popup menu: square, the menu colour in a 1px border.
func (flatlafEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := flatColors(l)
	b = flatSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.menu))
	flatRing(ctx, b, 0, flatPx(l), c.menuBorder)
}

func (e flatlafEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := flatColors(l)
	ch := MenuChromeFor(l)
	px := flatPx(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := b.Min.X-ch.PadL+px, b.Max.X+ch.PadR-px
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, px), paintengine2d.Fill(c.sep))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+px, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*px, b.Dy())
		ctx.DrawRect(flatSnap(hb), paintengine2d.Fill(c.sel))
	}
	fg, sc, mark := c.text, c.accel, c.menuCheck
	switch {
	case st.Disabled():
		fg, sc, mark = c.textDis, c.textDis, c.textDis
	case hot:
		fg, sc, mark = c.selText, c.selText, c.selText
	}
	font := l.body
	gw := ch.CheckCol()
	side := min(b.Dy()-2*px, gw-2*px, snap(l.S(15)))
	box := flatSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	switch {
	case row.Checked && row.Radio:
		ctx.DrawCircle(paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5), l.S(3.5), paintengine2d.Fill(mark))
	case row.Checked:
		flatTick(ctx, box, mark, max(l.S(1.9), 1.2))
	case !row.Radio && row.Icon != IconNone:
		l.drawToolIcon(ctx, box, row.Icon, fg)
	}
	ty := b.Min.Y + (b.Dy()-font.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow, b.Dy())
		arrow := c.arrow
		if hot {
			arrow = c.selText
		}
		c.arrowGlyph(l, ctx, ab, DirRight, arrow)
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
	l.drawTextUnderline(ctx, font, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

func (e flatlafEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := flatColors(l)
	fg := c.row(ctx, flatSnap(b), st)
	pad := l.S(6)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e flatlafEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := flatColors(l)
	fg := c.row(ctx, flatSnap(b), st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		col := c.treeIcon
		if st.Checked() && !st.Inactive() && !st.Backdrop() {
			col = c.selText
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
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableHeader: the table's colour, a 1px separator at each column's
// right and along the bottom, a grey wash under the pointer.
func (e flatlafEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := flatColors(l)
	b = flatSnap(b)
	if b.Empty() {
		return
	}
	px := flatPx(l)
	fill := c.field
	switch {
	case st.Disabled():
	case st.Pressed():
		fill = c.hdrPress
	case st.Hovered():
		fill = c.hdrHover
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.hdrSep))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, b.Min.Y, px, b.Dy()-px), paintengine2d.Fill(c.hdrSep))
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		c.arrowGlyph(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, c.arrow)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8)-aw, b.Dy()), fg, AlignStart, 0)
}

func (e flatlafEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := flatColors(l)
	cell := flatSnap(b)
	fg := c.row(ctx, cell, st)
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
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

func (flatlafEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(flatColors(l).bg))
}

func (flatlafEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := flatColors(l)
	b = flatSnap(b)
	px := flatPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), px), paintengine2d.Fill(c.sep))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(snap(x), b.Min.Y+l.S(5), px, b.Dy()-l.S(10)), paintengine2d.Fill(c.sep))
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y, slot-l.S(12), b.Dy()), c.text, AlignStart, 0)
	}
}

func (flatlafEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := flatColors(l)
	b = flatSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.textDis, AlignStart, 0)
	}
}

// DrawAccordionHeader: a collapsible section header drawn from FlatLaf's
// parts — the chevron, the title, a grey wash under the pointer and a
// separator below.
func (e flatlafEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := flatColors(l)
	b = flatSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	px := flatPx(l)
	switch {
	case st.Disabled():
	case st.Pressed():
		ctx.DrawRect(b, paintengine2d.Fill(c.toolPress))
	case st.Hovered():
		ctx.DrawRect(b, paintengine2d.Fill(c.tabHover))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.sep))
	col := c.arrow
	if st.Disabled() {
		col = c.arrowDis
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(16), b.Dy()), expanded, col)
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		flatRing(ctx, b, 0, flatPx(l), c.focus)
		flatRing(ctx, b.Inset(px), 0, px, c.borderFocus)
	}
}

func (flatlafEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := flatColors(l)
	px := flatPx(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(c.sep))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(c.sep))
}

// DrawSplitter is the split pane divider's grip: three small dots.
func (flatlafEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := flatColors(l)
	b = flatSnap(b)
	d := snap(l.S(3))
	gap := snap(l.S(2))
	if d < 1 || b.Dx() < d || b.Dy() < d {
		return
	}
	col := c.grip
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		col = c.arrow
	}
	span := 3*d + 2*gap
	p := paintengine2d.NewPath()
	for i := 0; i < 3; i++ {
		o := float32(i) * (d + gap)
		var dot paintengine2d.Rect
		if vertical {
			dot = paintengine2d.XYWH(snap((b.Min.X+b.Max.X-d)*0.5), snap((b.Min.Y+b.Max.Y-span)*0.5)+o, d, d)
		} else {
			dot = paintengine2d.XYWH(snap((b.Min.X+b.Max.X-span)*0.5)+o, snap((b.Min.Y+b.Max.Y-d)*0.5), d, d)
		}
		if dot.Min.X >= b.Min.X && dot.Max.X <= b.Max.X && dot.Min.Y >= b.Min.Y && dot.Max.Y <= b.Max.Y {
			p.AddCircle(paintengine2d.Pt((dot.Min.X+dot.Max.X)*0.5, (dot.Min.Y+dot.Max.Y)*0.5), d*0.5)
		}
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// DrawTooltip: Light's pale tip in a 1px border, Dark's charcoal one.
func (flatlafEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := flatColors(l)
	b = flatSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.tip))
	flatRing(ctx, b, 0, flatPx(l), c.tipBorder)
	pad := l.S(6)
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs --------------------------------------------------------------------------------------

func flatPack(name, label string, summary string, fam ThemeName, sc flatScheme, metrics ChromeMetrics, params map[string]float32) ThemePack {
	h := func(k string) paintengine2d.Color { return Hex(sc[k]) }
	bg, field := h("bg"), h("field")
	pal := Palette{
		Background: bg, Surface: bg, SurfaceAlt: h("btn"),
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.3),
		Border:  h("border"), Divider: h("sep"),
		Text: h("text"), TextMuted: h("textDis"), TextOnAccent: h("selText"),
		Accent: h("accent"), AccentHover: h("sliderHover"), AccentPress: h("sliderPress"),
		Danger: h("red"), Success: h("green"), Warning: h("yellow"),
		Track: h("track"), Thumb: h("thumb"),
		Field: field, FieldBorder: h("border"),
		Focus: h("borderFocus"), Selection: h("sel"),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.15), Highlight: h("toolHover"),
		MenuHover: h("sel"), MenuHoverBorder: h("sel"), MenuGutter: h("menu"),
		BevelLight: h("btn"), BevelDark: h("border"),
	}
	tok := ThemeTokens{
		Engine:  "flatlaf",
		Bevel:   BevelNone,
		Family:  fam,
		Palette: pal,
		Metrics: metrics,
		Params:  params,
		Extra:   map[string]paintengine2d.Color{"selectionText": h("selText")},
	}
	flatChrome(&tok, h)
	return ThemePack{
		Name: name, Label: label, Year: 2019, Lineage: "Java", Summary: summary,
		Era: EraFlatLaf, Palette: fam, Tokens: tok,
	}
}

// flatChrome sets the chrome states FlatLaf derives from its colours.
func flatChrome(tok *ThemeTokens, h func(string) paintengine2d.Color) {
	tok.Hot = ChromeState{Fill: h("toolHover"), Border: h("btnHoverBorder")}
	tok.Pressed = ChromeState{Fill: h("btnPress"), Border: h("btnHoverBorder")}
	tok.Selected = ChromeState{Fill: h("sel"), Border: h("sel")}
	tok.Focus = ChromeState{Fill: h("btnFocus"), Border: h("borderFocus")}
}

// ---- accent -------------------------------------------------------------------------------------

// FlatLaf's colour functions, as its properties files use them: LESS's
// operations in HSL (hue in degrees, saturation and lightness in percent),
// each rounding its result to 8-bit channels as java.awt.Color does.

func flat8(v float64) float32 { return float32(math.Floor(math.Min(1, math.Max(0, v))*255+0.5)) / 255 }

func flatHSL(c paintengine2d.Color) (h, s, l float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l * 100
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s * 100, l * 100
}

func flatFromHSL(h, s, l float64, a float32) paintengine2d.Color {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	s, l = math.Min(100, math.Max(0, s))/100, math.Min(100, math.Max(0, l))/100
	if s == 0 {
		v := flat8(l)
		return paintengine2d.RGBA(v, v, v, a)
	}
	q := l + s - l*s
	if l < 0.5 {
		q = l * (1 + s)
	}
	p := 2*l - q
	ch := func(t float64) float32 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return flat8(p + (q-p)*6*t)
		case t < 0.5:
			return flat8(q)
		case t < 2.0/3:
			return flat8(p + (q-p)*(2.0/3-t)*6)
		}
		return flat8(p)
	}
	return paintengine2d.RGBA(ch(h+1.0/3), ch(h), ch(h-1.0/3), a)
}

// flatLighten is lighten(c, pct) (darken with a negative pct).
func flatLighten(c paintengine2d.Color, pct float64) paintengine2d.Color {
	h, s, l := flatHSL(c)
	return flatFromHSL(h, s, l+pct, c.A)
}

func flatSaturate(c paintengine2d.Color, pct float64) paintengine2d.Color {
	h, s, l := flatHSL(c)
	return flatFromHSL(h, s+pct, l, c.A)
}

func flatSpin(c paintengine2d.Color, deg float64) paintengine2d.Color {
	h, s, l := flatHSL(c)
	return flatFromHSL(h+deg, s, l, c.A)
}

// flatSetLightness is changeLightness(c, pct).
func flatSetLightness(c paintengine2d.Color, pct float64) paintengine2d.Color {
	h, s, _ := flatHSL(c)
	return flatFromHSL(h, s, pct, c.A)
}

// flatMix is mix(a, b, w): w of a and the rest of b.
func flatMix(a, b paintengine2d.Color, w float64) paintengine2d.Color {
	m := func(x, y float32) float32 { return flat8(float64(x)*w + float64(y)*(1-w)) }
	return paintengine2d.RGBA(m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 1)
}

func flatTint(c paintengine2d.Color, w float64) paintengine2d.Color {
	return flatMix(paintengine2d.RGB(1, 1, 1), c, w)
}

func flatShade(c paintengine2d.Color, w float64) paintengine2d.Color {
	return flatMix(paintengine2d.RGB(0, 0, 0), c, w)
}

// flatFade is fade(c, a): c at alpha a.
func flatFade(c paintengine2d.Color, a float64) paintengine2d.Color { return c.WithAlpha(flat8(a)) }

// Accented is FlatLaf's accent colour. FlatLaf derives its whole accent
// family from one colour — @accentBaseColor, the #2675BF of FlatLaf Light
// and #4B6EAF of FlatLaf Dark and Darcula: the selection, the checked box
// and its tick, the slider and progress (a brighter variation of it), the
// tab underline and its faded unfocused form, the focus colour and the
// focused and hovered borders mixed from it, the default button (its ring
// in Light, its fill in Dark), the table's focused-cell line. The desktop
// accent takes the base's place and every property follows by FlatLaf's
// own formulas (lighten, saturate, spin, tint, shade, mix). FlatLaf's
// @accentColor would instead paint every one of them in the single accent
// colour; taking the base keeps the variations, and the pack's own blue
// gives the pack back exactly. The text on the selection keeps FlatLaf's
// contrast rule: white until the accent reaches about 2.2:1 in the light
// theme, the light text until 3:1 in the dark ones.
func (flatlafEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	dark := accentDark(tok)
	sc := flatLight
	if dark {
		sc = flatDark
	}
	own := func(k string) paintengine2d.Color { return accentX(tok, k, Hex(sc[k])) }
	base, bg := accent, own("bg")
	set := map[string]paintengine2d.Color{"sel": base, "accent": base}
	put := func(c paintengine2d.Color, keys ...string) {
		for _, k := range keys {
			set[k] = c
		}
	}
	if dark {
		base2 := flatLighten(flatSaturate(flatSpin(base, -8), 13), 5)
		focus := flatShade(flatSpin(base, -8), 0.2)
		def := flatLighten(flatSpin(base, -8), -13)
		put(base2, "slider", "progress", "tabLine")
		put(focus, "focus")
		put(flatLighten(focus, 5), "borderFocus", "btnHoverBorder", "btnFocusBorder", "chkHoverBorder", "chkFocusBorder")
		put(def, "def", "defFocus")
		put(flatLighten(def, 3), "defHover")
		put(flatLighten(def, 6), "defPress")
		put(flatTint(def, 0.15), "defBorder")
		put(flatTint(def, 0.18), "defHoverBorder", "defFocusBorder")
		put(flatLighten(focus, 3), "defRing")
		put(flatSpin(flatSaturate(flatShade(base, 0.7), 20), -15), "selOff")
		put(flatLighten(base, 10), "cellFocus")
		put(flatLighten(base2, 5), "sliderHover")
		put(flatLighten(base2, 8), "sliderPress")
		put(flatFade(flatSetLightness(focus, 60), 0.3), "sliderHalo")
		put(flatMix(base2, bg, 0.6), "tabLineOff")
		put(flatMix(base, bg, 0.25), "tabFocus")
		// The focused check box's wash is no formula of FlatLaf's: it is
		// the pack's wash of its base, retinted.
		put(accentWash(accent, own("accent"), own("chkFocus")), "chkFocus")
		ink := flatShade(bg, 0.5)
		put(ReadableOn(base, 3, own("selText"), ink), "selText")
		put(ReadableOn(def, 3, own("defText"), ink), "defText")
	} else {
		base2 := flatLighten(flatSaturate(base, 10), 6)
		focus := flatLighten(base, 31)
		border := flatShade(focus, 0.1)
		under := flatTint(base, 0.1)
		put(base2, "slider", "progress")
		put(flatTint(base2, 0.2), "chkMark", "chkSelBorder", "menuCheck", "defBorder")
		put(focus, "focus", "defRing")
		put(border, "borderFocus", "btnHoverBorder", "btnFocusBorder", "defHoverBorder", "defFocusBorder")
		put(flatShade(border, 0.1), "chkHoverBorder", "chkFocusBorder")
		put(flatSetLightness(focus, 95), "btnFocus", "defFocus", "chkFocus")
		put(flatLighten(base2, -5), "sliderHover")
		put(flatLighten(base2, -8), "sliderPress")
		put(flatFade(flatSetLightness(focus, 75), 0.5), "sliderHalo")
		put(under, "tabLine")
		put(flatMix(under, bg, 0.5), "tabLineOff")
		put(flatMix(base, bg, 0.1), "tabFocus")
		put(flatLighten(base, -20), "cellFocus")
		put(ReadableOn(base, 2.2, own("selText"), own("text")), "selText")
	}
	tok = CloneTokenMaps(tok)
	for k, c := range set {
		tok.Extra[k] = c
	}
	h := func(k string) paintengine2d.Color { return accentX(tok, k, Hex(sc[k])) }
	p := &tok.Palette
	p.Accent, p.Selection, p.Focus = h("accent"), h("sel"), h("borderFocus")
	p.AccentHover, p.AccentPress, p.TextOnAccent = h("sliderHover"), h("sliderPress"), h("selText")
	p.MenuHover, p.MenuHoverBorder = h("sel"), h("sel")
	tok.Extra["selectionText"] = h("selText")
	flatChrome(&tok, h)
	return tok
}

func flatlafPacks() []ThemePack {
	darcula := ChromeMetrics{ControlH: 34, FieldH: 34, ComboH: 34, Checkbox: 20, Radio: 20}
	return []ThemePack{
		flatPack("flatlaf", "FlatLaf Light",
			"FlatLightLaf: flat white faces with an arc of 6, a 2px accent focus border, underlined tabs.",
			ThemeLight, flatLight, ChromeMetrics{}, map[string]float32{"focusWidth": 0, "innerFocus": 1, "defBorder": 2, "radioDot": 8}),
		flatPack("flatlaf-night", "FlatLaf Dark",
			"FlatDarkLaf: charcoal faces, the dark blue bold default button, grey ticks.",
			ThemeDark, flatDark, ChromeMetrics{}, map[string]float32{"focusWidth": 0, "innerFocus": 1, "defBorder": 1, "boldDefault": 1, "radioDot": 8}),
		flatPack("flatlaf-darcula", "Darcula",
			"FlatDarculaLaf: IntelliJ's Darcula — a 2px focus ring outside every border, triangle arrows.",
			ThemeDark, flatDark, darcula, map[string]float32{"focusWidth": 2, "innerFocus": 0, "defBorder": 1, "boldDefault": 1, "triangles": 1, "radioDot": 5}),
	}
}
