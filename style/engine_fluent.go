package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// fluentEngine paints Windows 11 (2021): Fluent 2 as WinUI 3 draws it,
// from the published WinUI design tokens taken as numbers — the Control,
// Stroke, Fill and Text colour tokens (many are translucent, so controls
// take on the surface under them), CornerRadius 4 for controls and 8 for
// overlays, and the documented control anatomy:
//
//   - buttons are 4px-rounded translucent white (ControlFillColorDefault)
//     inside the elevation border — a hairline that is darker along the
//     bottom edge (lighter along the top in the dark theme); the accent
//     (default) button is the accent colour with white text;
//   - check boxes and radios are 20px, outlined in ControlStrongStroke,
//     filled with the accent when on (a white tick, a white centre dot that
//     grows under the pointer and shrinks when pressed);
//   - the toggle switch is 40×20 with a 12px knob that grows on hover and
//     stretches while pressed;
//   - the slider is a 4px track, accent up to its 20px round thumb with the
//     accent inner dot;
//   - text boxes carry the strong bottom line that becomes a 2px accent
//     line when focused;
//   - scroll bars float: a thin line until the pointer is over them, then
//     a rounded track with a 6px thumb and small arrow carets;
//   - tabs are TabView's: the selected tab has 8px top corners and curves
//     into the page; menus are 8px-rounded flyouts with rounded hot rows;
//   - list and tree items are rounded rows with the 3×16 accent pill at the
//     left of the selected one (a selection stays as it is when the view
//     loses focus, as in WinUI, and turns grey in an inactive window);
//   - keyboard focus is the two-colour focus visual: a 2px outer line
//     (black in the light theme) and a 1px inner one (white);
//   - progress bars are a 3px accent bar over a 1px track.
//
// Pack data: params "scheme" 0 = light, 1 = dark; every colour key of
// fluentLight can be overridden through "extra" ("#rrggbbaa" for the
// translucent tokens).
type fluentEngine struct{ BaseEngine }

func init() {
	RegisterEngine(fluentEngine{})
	for _, p := range fluentPacks() {
		RegisterPack(p)
	}
}

func (fluentEngine) ID() string { return "fluent" }

// DefaultMetrics are WinUI 3's sizes at the toolkit's 16px UI font (WinUI
// uses 14px): 32px buttons and fields grow slightly, check boxes and
// radios stay 20px, the toggle switch 40×20, corners 4px.
func (fluentEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Radius:    4, RadiusSmall: 4,
		ControlH: 34, FieldH: 32, ComboH: 32,
		Checkbox: 20, Radio: 20,
		MenuItemH: 32, MenuBarH: 36, TabH: 38, RowH: 32,
		TitleBar: 32, HeaderH: 32, ProgressH: 14, SliderH: 32, Thumb: 20,
		Scroll: 12, Pad: 12, FieldPad: 10, FocusWidth: 2, Border: 1,
		ToolBarH: 40, StatusBarH: 28, SpinnerW: 24, SwitchW: 40, SwitchH: 20,
	}
}

// ---- colour tables ------------------------------------------------------------------------------

// fluentScheme maps a token to "#rrggbb" or "#rrggbbaa" (the WinUI tokens'
// ARGB values with the alpha moved to the end).
type fluentScheme map[string]string

// fluentLight is WinUI 3's light theme with the default blue accent.
var fluentLight = fluentScheme{
	// Backgrounds: Mica's base, the solid tertiary (tabs, pages), the
	// flyout (menus, tool tips) and card layers.
	"bg": "#f3f3f3", "page": "#f9f9f9", "field": "#ffffff", "flyout": "#f9f9f9",
	"card": "#ffffffb3", "cardStroke": "#0000000f", "flyoutStroke": "#0000000f", "windowStroke": "#0000001f",

	// Text.
	"text": "#1b1b1b", "text2": "#616161", "text3": "#8d8d8d", "textDis": "#a3a3a3",

	// Accent fills (default, pointer over, pressed, disabled) and text on them.
	"accent": "#0067c0", "accent2": "#0067c0e6", "accent3": "#0067c0cc", "accentDis": "#00000037",
	"onAccent": "#ffffff", "onAccent2": "#ffffffb3", "onAccentDis": "#ffffff",
	"selText": "#0067c0",

	// Control fills (rest, pointer over, pressed, disabled, focused input).
	"ctl": "#ffffffb3", "ctl2": "#f9f9f980", "ctl3": "#f9f9f94d", "ctlDis": "#f9f9f94d", "ctlInput": "#ffffff",
	// Strokes: control, elevation edge, strong (check boxes, text box
	// bottom), strong disabled; strong fills (knobs, tracks).
	"stroke": "#0000000f", "stroke2": "#00000029", "strong": "#00000072", "strongDis": "#00000037",
	"strongFill": "#00000072", "strongFillDis": "#00000051",
	"onAccentStroke": "#ffffff14", "onAccentStroke2": "#00000066",
	// Alt fills (check box / switch wells) and subtle fills (list rows,
	// menu items, tool buttons).
	"alt": "#00000006", "alt2": "#0000000f", "alt3": "#00000018", "altDis": "#00000000",
	"subtle": "#00000009", "subtle2": "#00000006", "divider": "#0000000f",
	"solid": "#ffffff",

	// Focus visual.
	"focusOuter": "#000000e4", "focusInner": "#ffffffb3",

	// Status colours and the caption close button.
	"critical": "#c42b1c", "success": "#0f7b0f", "caution": "#9d5d00",
	"closeHot": "#c42b1c", "closePress": "#c42b1ce6", "onClose": "#ffffff",
}

// fluentDark is WinUI 3's dark theme.
var fluentDark = fluentScheme{
	"bg": "#202020", "page": "#282828", "field": "#1c1c1c", "flyout": "#2c2c2c",
	"card": "#ffffff0d", "cardStroke": "#00000019", "flyoutStroke": "#00000033", "windowStroke": "#ffffff26",
	"text": "#ffffff", "text2": "#cccccc", "text3": "#969696", "textDis": "#717171",
	"accent": "#4cc2ff", "accent2": "#4cc2ffe6", "accent3": "#4cc2ffcc", "accentDis": "#ffffff28",
	"onAccent": "#000000", "onAccent2": "#00000080", "onAccentDis": "#ffffff87",
	"selText": "#0067c0",
	"ctl":     "#ffffff0f", "ctl2": "#ffffff15", "ctl3": "#ffffff08", "ctlDis": "#ffffff0b", "ctlInput": "#1e1e1eb3",
	"stroke": "#ffffff12", "stroke2": "#ffffff18", "strong": "#ffffff8b", "strongDis": "#ffffff28",
	"strongFill": "#ffffff8b", "strongFillDis": "#ffffff3f",
	"onAccentStroke": "#ffffff14", "onAccentStroke2": "#00000023",
	"alt": "#00000019", "alt2": "#ffffff0b", "alt3": "#ffffff12", "altDis": "#00000000",
	"subtle": "#ffffff0f", "subtle2": "#ffffff0a", "divider": "#ffffff15",
	"solid":      "#454545",
	"focusOuter": "#ffffff", "focusInner": "#000000b3",
	"critical": "#ff99a4", "success": "#6ccb5f", "caution": "#fce100",
	"closeHot": "#c42b1c", "closePress": "#c42b1ce6", "onClose": "#ffffff",
}

// ---- resolved colour set -----------------------------------------------------------------------

// fluentSet is a look's resolved Fluent tokens, built once per look.
type fluentSet struct {
	dark bool

	bg, page, field, flyout, card, cardStroke, flyoutStroke, windowStroke paintengine2d.Color
	text, text2, text3, textDis                                           paintengine2d.Color
	accent                                                                [4]paintengine2d.Color // rest, hover, pressed, disabled
	onAccent                                                              [3]paintengine2d.Color // rest, pressed, disabled
	ctl                                                                   [4]paintengine2d.Color // rest, hover, pressed, disabled
	ctlInput                                                              paintengine2d.Color
	stroke, stroke2, strong, strongDis, strongFill, strongFillDis         paintengine2d.Color
	alt                                                                   [4]paintengine2d.Color // rest, hover, pressed, disabled
	subtle, subtle2, divider, solid                                       paintengine2d.Color
	focusOuter, focusInner                                                paintengine2d.Color
	critical, caution, closeHot, closePress, onClose                      paintengine2d.Color

	// Border gradients (absolute, over the edge's last 3px): the control
	// elevation border, the accent one and the text box's strong edge.
	elev, accentElev, textElev []paintengine2d.GradientStop

	// The accent tint of a selected table row.
	tint paintengine2d.Color
}

type fluentKey struct{}

// fluentColors is the look's resolved token set (built once per look).
func fluentColors(l *Classic) *fluentSet {
	return l.Memo(fluentKey{}, func() any { return fluentBuild(l) }).(*fluentSet)
}

// fluentDarkScheme reports the dark token set: params "scheme" 1, or a dark
// pack that names no scheme.
func fluentDarkScheme(l *Classic) bool {
	def := float32(0)
	if Luma(l.palette.Background) < 0.5 {
		def = 1
	}
	return l.P("scheme", def) == 1
}

func fluentBuild(l *Classic) *fluentSet {
	dark := fluentDarkScheme(l)
	sc := fluentLight
	if dark {
		sc = fluentDark
	}
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = "#ff00ff" // a missing key shows (and the table test fails)
		}
		return l.X(k, Hex(v))
	}
	s := &fluentSet{dark: dark}
	s.bg, s.page, s.field, s.flyout = col("bg"), col("page"), col("field"), col("flyout")
	s.card, s.cardStroke, s.flyoutStroke, s.windowStroke = col("card"), col("cardStroke"), col("flyoutStroke"), col("windowStroke")
	s.text, s.text2, s.text3, s.textDis = col("text"), col("text2"), col("text3"), col("textDis")
	s.accent = [4]paintengine2d.Color{col("accent"), col("accent2"), col("accent3"), col("accentDis")}
	s.onAccent = [3]paintengine2d.Color{col("onAccent"), col("onAccent2"), col("onAccentDis")}
	s.ctl = [4]paintengine2d.Color{col("ctl"), col("ctl2"), col("ctl3"), col("ctlDis")}
	s.ctlInput = col("ctlInput")
	s.stroke, s.stroke2, s.strong, s.strongDis = col("stroke"), col("stroke2"), col("strong"), col("strongDis")
	s.strongFill, s.strongFillDis = col("strongFill"), col("strongFillDis")
	s.alt = [4]paintengine2d.Color{col("alt"), col("alt2"), col("alt3"), col("altDis")}
	s.subtle, s.subtle2, s.divider, s.solid = col("subtle"), col("subtle2"), col("divider"), col("solid")
	s.focusOuter, s.focusInner = col("focusOuter"), col("focusInner")
	s.critical, s.caution = col("critical"), col("caution")
	s.closeHot, s.closePress, s.onClose = col("closeHot"), col("closePress"), col("onClose")
	onA, onA2 := col("onAccentStroke"), col("onAccentStroke2")
	if dark {
		// The dark theme lights the top edge instead of shading the bottom.
		s.elev = []paintengine2d.GradientStop{Stop(0, s.stroke2), Stop(0.33, s.stroke2), Stop(1, s.stroke)}
		s.accentElev = []paintengine2d.GradientStop{Stop(0, onA), Stop(1, onA)}
	} else {
		s.elev = []paintengine2d.GradientStop{Stop(0, s.stroke), Stop(0.67, s.stroke2), Stop(1, s.stroke2)}
		s.accentElev = []paintengine2d.GradientStop{Stop(0, onA), Stop(0.67, onA2), Stop(1, onA2)}
	}
	s.textElev = []paintengine2d.GradientStop{Stop(0, s.stroke), Stop(0.5, s.stroke), Stop(0.5, s.strong), Stop(1, s.strong)}
	s.tint = s.accent[0].WithAlpha(0.14)
	if dark {
		s.tint = s.accent[0].WithAlpha(0.2)
	}
	return s
}

// over is c flattened over the look's background (for contrast maths).
func (c *fluentSet) over(col paintengine2d.Color) paintengine2d.Color {
	return Mix(c.bg, col, col.A)
}

// ---- painting helpers ------------------------------------------------------------------------------

// fill paints the inside of a 1px-bordered round rect (WinUI keeps the
// background inside the border, so translucent layers never double up).
func (c *fluentSet) fill(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color) {
	if col.A <= 0 || b.Empty() {
		return
	}
	lw := winPx(l)
	in := b.Inset(lw)
	if in.Empty() {
		return
	}
	ctx.DrawRoundRect(in, max(r-lw, 0), max(r-lw, 0), paintengine2d.Fill(col))
}

// edge strokes the 1px border of b with a vertical gradient over its last
// three pixels: at the bottom (light theme elevation, the accent border) or,
// with top, at the top (dark theme elevation).
func (c *fluentSet) edge(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, stops []paintengine2d.GradientStop, top bool) {
	lw := winPx(l)
	g := 3 * lw
	lg := paintengine2d.LinearGradient{Start: paintengine2d.Pt(b.Min.X, b.Max.Y-g), End: paintengine2d.Pt(b.Min.X, b.Max.Y), Stops: stops}
	if top {
		lg = paintengine2d.LinearGradient{Start: paintengine2d.Pt(b.Min.X, b.Min.Y), End: paintengine2d.Pt(b.Min.X, b.Min.Y+g), Stops: stops}
	}
	winRing(ctx, b, r, lw, paintengine2d.Linear(lg))
}

// ctlIndex picks rest / hover / pressed / disabled.
func fluentIndex(st ControlState) int {
	switch {
	case st.Disabled():
		return 3
	case st.Pressed():
		return 2
	case st.Hovered():
		return 1
	}
	return 0
}

// button paints a standard or accent (st.Primary) button and returns the
// label colour.
func (c *fluentSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return c.text
	}
	r := l.rx(4)
	i := fluentIndex(st)
	if st.Primary() {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.accent[i]))
		if i < 2 {
			c.edge(l, ctx, b, r, c.accentElev, false)
		}
		switch i {
		case 2:
			return c.onAccent[1]
		case 3:
			return c.onAccent[2]
		}
		return c.onAccent[0]
	}
	c.fill(l, ctx, b, r, c.ctl[i])
	if i >= 2 {
		winRing(ctx, b, r, lw, paintengine2d.Fill(c.stroke))
	} else {
		c.edge(l, ctx, b, r, c.elev, c.dark)
	}
	switch i {
	case 2:
		return c.text2
	case 3:
		return c.textDis
	}
	return c.text
}

// focusVisual is WinUI's focus visual inside b: a 2px outer line and a 1px
// inner one.
func (c *fluentSet) focusVisual(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32) {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 6*lw || b.Dy() < 6*lw {
		return
	}
	winRing(ctx, b, r, 2*lw, paintengine2d.Fill(c.focusOuter))
	winRing(ctx, b.Inset(2*lw), max(r-2*lw, 0), lw, paintengine2d.Fill(c.focusInner))
}

// subtleBox fills a rounded 4px row / button highlight.
func (c *fluentSet) subtleBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	if col.A <= 0 {
		return
	}
	b = winSnap(b)
	if b.Empty() {
		return
	}
	r := l.rx(4)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
}

// pill paints the 3px accent selection indicator at the left of an item.
func (c *fluentSet) pill(l *Classic, ctx *paintengine2d.Context, item paintengine2d.Rect, st ControlState) {
	w := snap(l.S(3))
	h := min(snap(l.S(16)), item.Dy()-snap(l.S(6)))
	if h < w || item.Dx() < 4*w {
		return
	}
	col := c.accent[0]
	switch {
	case st.Disabled():
		col = c.accent[3]
	case st.Backdrop():
		col = c.strongFill
	}
	pb := paintengine2d.XYWH(item.Min.X, snap((item.Min.Y+item.Max.Y-h)*0.5), w, h)
	ctx.DrawRoundRect(pb, w*0.5, w*0.5, paintengine2d.Fill(col))
}

// rowFill is the item background of a list / tree row state.
func (c *fluentSet) rowFill(st ControlState) paintengine2d.Color {
	switch {
	case st.Disabled():
		if st.Checked() {
			return c.subtle2
		}
		return paintengine2d.Color{}
	case st.Checked() && st.Backdrop():
		return c.subtle2
	case st.Checked() && st.Pressed():
		return c.subtle
	case st.Checked() && st.Hovered():
		return c.subtle2
	case st.Checked():
		return c.subtle
	case st.Pressed():
		return c.subtle2
	case st.Hovered():
		return c.subtle
	}
	return paintengine2d.Color{}
}

// itemRect is a list / tree item inside its row: 4px in from the sides,
// 2px from the top and bottom.
func fluentItemRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return winSnap(paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y+l.S(2), b.Dx()-l.S(8), b.Dy()-l.S(4)))
}

// caret fills a small solid triangle (the scroll bar arrows).
func fluentCaret(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	u := min(l.S(1), min(b.Dx(), b.Dy())/9)
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	a, d := 3.5*u, 2.2*u
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-a, cy+d)
		p.LineTo(cx+a, cy+d)
		p.LineTo(cx, cy-d)
	case DirDown:
		p.MoveTo(cx-a, cy-d)
		p.LineTo(cx+a, cy-d)
		p.LineTo(cx, cy+d)
	case DirLeft:
		p.MoveTo(cx+d, cy-a)
		p.LineTo(cx+d, cy+a)
		p.LineTo(cx-d, cy)
	default:
		p.MoveTo(cx-d, cy-a)
		p.LineTo(cx-d, cy+a)
		p.LineTo(cx+d, cy)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStrokeAndFill,
		Stroke: paintengine2d.Stroke{Width: u, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// ---- parts ------------------------------------------------------------------------------------------

func (e fluentEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := fluentColors(l)
	switch role {
	case RoleButton, RoleCombo:
		return c.button(l, ctx, b, st)
	case RoleTool:
		return c.toolFace(l, ctx, b, st)
	case RoleField:
		c.textBox(l, ctx, b, st)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		c.subtleBox(l, ctx, b, c.rowFill(st))
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
	case RoleThumb:
		col := c.strongFill
		if st.Hovered() || st.Pressed() {
			col = c.text2
		}
		r := min(b.Dx(), b.Dy()) * 0.5
		ctx.DrawRoundRect(winSnap(b), r, r, paintengine2d.Fill(col))
	case RoleTrack:
		r := min(b.Dx(), b.Dy()) * 0.5
		ctx.DrawRoundRect(winSnap(b), r, r, paintengine2d.Fill(c.flyout))
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	}
	if st.Disabled() {
		return c.textDis
	}
	return c.text
}

// textBox paints a WinUI text box: the control fill (solid white when
// focused) inside a hairline whose bottom edge is the strong stroke, or a
// 2px accent line when focused.
func (c *fluentSet) textBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(4)
	switch {
	case st.Disabled():
		c.fill(l, ctx, b, r, c.ctl[3])
		winRing(ctx, b, r, lw, paintengine2d.Fill(c.stroke))
	case st.Focused():
		c.fill(l, ctx, b, r, c.ctlInput)
		winRing(ctx, b, r, lw, paintengine2d.Fill(c.stroke))
		// The 2px accent underline, following the corners.
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*lw, b.Dx(), 2*lw))
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.accent[0]))
		ctx.Restore()
	default:
		i := 0
		if st.Hovered() {
			i = 1
		}
		c.fill(l, ctx, b, r, c.ctl[i])
		g := 2 * lw
		winRing(ctx, b, r, lw, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(b.Min.X, b.Max.Y-2*g), End: paintengine2d.Pt(b.Min.X, b.Max.Y), Stops: c.textElev,
		}))
	}
}

// toolFace is an app-bar button: subtle until hot; a latched toggle is the
// accent.
func (c *fluentSet) toolFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	i := fluentIndex(st)
	if st.Checked() && st.Toggle() {
		c.subtleBox(l, ctx, b, c.accent[i])
		if i == 3 {
			return c.onAccent[2]
		}
		return c.onAccent[0]
	}
	switch i {
	case 3:
		return c.textDis
	case 2:
		c.subtleBox(l, ctx, b, c.subtle2)
		return c.text2
	case 1:
		c.subtleBox(l, ctx, b, c.subtle)
	}
	return c.text
}

func (e fluentEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := fluentColors(l)
	box = winSnap(box)
	lw := winPx(l)
	if box.Dx() < 6*lw || box.Dy() < 6*lw {
		return
	}
	r := l.rx(4)
	i := fluentIndex(st)
	if checked || st.Checked() {
		ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(c.accent[i]))
		mark := c.onAccent[0]
		switch i {
		case 2:
			mark = c.onAccent[1]
		case 3:
			mark = c.onAccent[2]
		}
		winTick(ctx, box.Inset(box.Dx()*0.1), mark, max(box.Dx()*0.075, lw*1.2))
		return
	}
	c.fill(l, ctx, box, r, c.alt[i])
	edge := c.strong
	if i >= 2 {
		edge = c.strongDis
	}
	winRing(ctx, box, r, lw, paintengine2d.Fill(edge))
}

func (e fluentEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := fluentColors(l)
	box = winSnap(box)
	lw := winPx(l)
	r := min(box.Dx(), box.Dy()) * 0.5
	if r < 3*lw {
		return
	}
	ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
	i := fluentIndex(st)
	k := r / 10 // the WinUI glyph is 20px
	if selected || st.Checked() {
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.accent[i]))
		dot := [4]float32{6, 7, 5, 6}[i] * k
		mark := c.onAccent[0]
		if i == 3 {
			mark = c.onAccent[2]
		}
		ctx.DrawCircle(ctr, dot, paintengine2d.Fill(mark))
		return
	}
	ctx.DrawCircle(ctr, r-lw, paintengine2d.Fill(c.alt[i]))
	edge := c.strong
	if i >= 2 {
		edge = c.strongDis
	}
	p := paintengine2d.NewPath()
	p.AddCircle(ctr, r)
	p.AddCircle(ctr, r-lw)
	ctx.DrawPath(p, paintengine2d.Paint{Color: edge, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
	if i == 2 {
		// Pressing an empty radio shows the small centre it will get.
		ctx.DrawCircle(ctr, 5*k, paintengine2d.Fill(c.strongFill))
	}
}

// Arrow is the thin Fluent chevron.
func (fluentEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	winChevron(l, ctx, b, dir, col)
}

// Expander is TreeView's chevron: right when closed, down when open.
func (fluentEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	winChevron(l, ctx, b, dir, fluentColors(l).text2)
}

// MenuHighlight is the rounded subtle row of a menu flyout (and of an open
// menu-bar title).
func (fluentEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := fluentColors(l)
	c.subtleBox(l, ctx, b, c.subtle)
}

func (fluentEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return fluentColors(l).text
}

// Fields show focus with their accent underline (painted by Face).
func (fluentEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the focus visual.
func (fluentEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	fluentColors(l).focusVisual(l, ctx, b, l.rx(4))
}

// ---- scroll bars ------------------------------------------------------------------------------------

// ScrollBarStyle: WinUI's floating bar — a thin line at rest, a 12px
// rounded bar with arrow carets under the pointer.
func (fluentEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 12, Overlay: true, Inset: 1, Arrows: ArrowsEnds, ArrowLen: 12, MinThumb: 24, EndPad: 1}
}

func (e fluentEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := fluentColors(l)
	bar := winSnap(p.Bar)
	if bar.Empty() {
		return
	}
	hover := !st.Disabled && (st.Hovered || st.Hot != ScrollNone || st.Pressed != ScrollNone)
	across := func(r paintengine2d.Rect, w float32) paintengine2d.Rect {
		if vertical {
			return paintengine2d.XYWH(snap((bar.Min.X+bar.Max.X-w)*0.5), r.Min.Y, w, r.Dy())
		}
		return paintengine2d.XYWH(r.Min.X, snap((bar.Min.Y+bar.Max.Y-w)*0.5), r.Dx(), w)
	}
	if !hover {
		if p.Thumb.Empty() || st.Disabled {
			return
		}
		w := snap(l.S(2))
		t := across(winSnap(p.Thumb), w)
		if vertical {
			t = t.Translate(paintengine2d.Pt(snap(l.S(2)), 0))
		} else {
			t = t.Translate(paintengine2d.Pt(0, snap(l.S(2))))
		}
		ctx.DrawRoundRect(t, w*0.5, w*0.5, paintengine2d.Fill(c.strongFill))
		return
	}
	r := min(bar.Dx(), bar.Dy()) * 0.5
	ctx.DrawRoundRect(bar, r, r, paintengine2d.Fill(c.flyout))
	winRing(ctx, bar, r, winPx(l), paintengine2d.Fill(c.flyoutStroke))
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		col := c.strongFill
		switch {
		case st.Disabled:
			col = c.strongFillDis
		case st.Pressed == part:
			col = c.text
		case st.Hot == part:
			col = c.text2
		}
		fluentCaret(l, ctx, b, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	col := c.strongFill
	if st.Pressed == ScrollThumbPart || st.Hot == ScrollThumbPart {
		col = c.text2
	}
	w := snap(l.S(6))
	t := across(winSnap(p.Thumb), w)
	ctx.DrawRoundRect(t, w*0.5, w*0.5, paintengine2d.Fill(col))
}

func (e fluentEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), winScrollState(st))
}

// ---- frames ----------------------------------------------------------------------------------------------

// GroupBoxInsets: a section heading above an 8px-rounded card.
func (fluentEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top = fluentHeadH(l) + pad
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// fluentHeadH is the band a section heading takes above its card.
func fluentHeadH(l *Classic) float32 {
	f := l.bold
	if f == nil {
		f = l.body
	}
	return snap(f.Height() + l.S(4))
}

// DrawGroupBox is a settings section: the heading in the strong body
// style, then the card (the card fill when raised, its stroke alone
// otherwise).
func (fluentEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := fluentColors(l)
	b = winSnap(b)
	card := b
	if title != "" {
		hh := fluentHeadH(l)
		f := l.bold
		if f == nil {
			f = l.body
		}
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+l.S(2), b.Min.Y, b.Dx()-l.S(4), hh), c.text, AlignStart, 0)
		card = paintengine2d.XYWH(b.Min.X, b.Min.Y+hh, b.Dx(), b.Dy()-hh)
	}
	c.card8(l, ctx, card, raised)
}

// card8 is a card: 8px corners, the card fill (when filled) and stroke.
func (c *fluentSet) card8(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, filled bool) {
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(8)
	if filled {
		c.fill(l, ctx, b, r, c.card)
	}
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.cardStroke))
}

// fluentCaptionH is the title bar: 32px at the toolkit's font.
func fluentCaptionH(l *Classic) float32 {
	h := l.body.Height() + l.S(12)
	if m := l.S(32); h < m {
		h = m
	}
	return snap(h)
}

func (fluentEngine) WindowFrameInsets(l *Classic) Insets {
	lw := winPx(l)
	return Insets{Top: fluentCaptionH(l), Right: lw, Bottom: lw, Left: lw}
}

// WindowCloseRect is the 46px caption button filling the title bar at the
// right.
func (fluentEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = winSnap(b)
	lw := winPx(l)
	h := fluentCaptionH(l) - lw
	w := min(snap(h*46/32), snap(b.Dx()*0.3))
	return paintengine2d.XYWH(b.Max.X-lw-w, b.Min.Y+lw, w, h)
}

// StyleHint: ContentDialog puts the primary button first; form labels are
// left-aligned.
func (fluentEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// DrawWindowFrame is a Windows 11 window: 8px corners, a hairline border,
// Mica for the title bar and body, the caption text at the left and the
// close button that turns red under the pointer.
func (e fluentEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	capH := fluentCaptionH(l)
	if b.Dx() < 3*capH || b.Dy() < capH+2*lw {
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		return
	}
	r := l.rx(8)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.bg))
	right := b.Max.X - lw
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		fill, glyph := paintengine2d.Color{}, c.text
		switch {
		case !st.Active:
			glyph = c.text3
		case st.ClosePress:
			fill, glyph = c.closePress, c.onClose.WithAlpha(0.8)
		case st.CloseHot:
			fill, glyph = c.closeHot, c.onClose
		}
		if fill.A > 0 {
			ctx.Save()
			ctx.ClipRoundRect(b.Inset(lw), max(r-lw, 0), max(r-lw, 0))
			ctx.DrawRect(cb, paintengine2d.Fill(fill))
			ctx.Restore()
		}
		side := snap(min(cb.Dy()*0.32, l.S(10)))
		gb := paintengine2d.XYWH(snap((cb.Min.X+cb.Max.X-side)*0.5), snap((cb.Min.Y+cb.Max.Y-side)*0.5), side, side)
		winCross(ctx, gb, glyph, lw)
		right = cb.Min.X - l.S(4)
	}
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.windowStroke))
	if title == "" {
		return
	}
	tc := c.text
	if !st.Active {
		tc = c.text3
	}
	x := b.Min.X + l.S(14)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(x, b.Min.Y+lw, right-x, capH-lw), tc, AlignStart, 0)
}

// PopupShadow: the soft, wide shadows of Windows 11 flyouts (elevation
// 32), tool tips (16) and dialogs (128).
func (fluentEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp, ok := fluentShadow(l, kind)
	if !ok {
		return Insets{}
	}
	return ShadowReach(sp.dx, sp.dy, sp.blur, sp.spread)
}

func (fluentEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if sp, ok := fluentShadow(l, kind); ok {
		DropShadow(ctx, b, sp.r, sp.col, sp.dx, sp.dy, sp.blur, sp.spread)
	}
}

func fluentShadow(l *Classic, kind PopupKind) (winShadow, bool) {
	k := float32(1)
	if fluentDarkScheme(l) {
		k = 2.4
	}
	switch kind {
	case PopupDialog:
		return winShadowSpec(l, 0.2*k, l.rx(8), 0, l.S(8), l.S(28), 0)
	case PopupTooltip:
		return winShadowSpec(l, 0.13*k, l.rx(4), 0, l.S(2), l.S(6), 0)
	}
	return winShadowSpec(l, 0.15*k, l.rx(8), 0, l.S(6), l.S(16), -l.S(1))
}

// TabOutset: the selected tab's bottom corners curve out into the page, so
// it takes room for them on both sides.
func (fluentEngine) TabOutset(l *Classic) Insets {
	e := snap(l.S(6))
	return Insets{Left: e, Right: e}
}

// DrawTabPane is the page under a TabView: the solid tertiary fill in the
// card stroke.
func (fluentEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fluentColors(l)
	b = winSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.page))
	winBorder(ctx, b, winPx(l), c.cardStroke)
}

// ViewFrameInsets: views sit in a 4px-rounded card, two pixels in.
func (fluentEngine) ViewFrameInsets(l *Classic) Insets {
	v := 2 * winPx(l)
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is the card a list, tree or table sits in.
func (fluentEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(4)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.field))
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.cardStroke))
}

// ItemFocus is the focus visual round the current item.
func (fluentEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	fluentColors(l).focusVisual(l, ctx, b, l.rx(4))
}

// ---- controls ------------------------------------------------------------------------------------------

func (e fluentEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := fluentColors(l)
	fg := c.button(l, ctx, b, st)
	l.drawFittedText(ctx, l.body, label, b, fg, AlignCenter, 8)
	if st.Focused() && !st.Disabled() {
		c.focusVisual(l, ctx, b, l.rx(4))
	}
}

func (e fluentEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := min(l.metrics.Checkbox, b.Dy())
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	fluentToggleLabel(l, ctx, b, box, st, label)
}

func (e fluentEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	side = min(side, b.Dy())
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	fluentToggleLabel(l, ctx, b, box, st, label)
}

// fluentToggleLabel draws the label of a check box / radio and, when
// focused, the focus visual.
func fluentToggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := fluentColors(l)
	var lb paintengine2d.Rect
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.textDis
		}
		gap := l.S(8)
		lb = paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		c.controlFocus(l, ctx, b, box, lb, label)
	}
}

// controlFocus rings a check box, radio or switch with the focus visual:
// round the whole control when its rect has room above and below the
// glyph (WinUI draws it outside the control), else round the label.
func (c *fluentSet) controlFocus(l *Classic, ctx *paintengine2d.Context, b, glyph, lb paintengine2d.Rect, label string) {
	right := glyph.Max.X + l.S(4)
	if label != "" {
		right = min(lb.Min.X+l.body.Advance(label)+l.S(6), b.Max.X)
	}
	r := l.rx(4)
	if b.Dy() >= glyph.Dy()+l.S(8) || label == "" {
		ring := paintengine2d.XYWH(b.Min.X, b.Min.Y, right-b.Min.X, b.Dy())
		if label == "" && b.Dy() >= glyph.Dy()+l.S(8) {
			ring = glyph.Inset(-l.S(4))
		}
		c.focusVisual(l, ctx, ring.Intersect(b), r)
		return
	}
	c.focusVisual(l, ctx, paintengine2d.XYWH(lb.Min.X-l.S(4), b.Min.Y, right-lb.Min.X+l.S(4), b.Dy()).Intersect(b), r)
}

func (e fluentEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if !st.Disabled() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	c := fluentColors(l)
	c.textBox(l, ctx, b, st)
	winDisabledText(l, ctx, b, text, placeholder, scrollX, face, c.textDis)
}

// DrawComboBox is a WinUI combo box: a button face with the chevron at the
// right (it drops two pixels while pressed).
func (e fluentEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := fluentColors(l)
	b = winSnap(b)
	fst := st &^ StatePrimary
	if open {
		fst |= StatePressed
	}
	fg := c.button(l, ctx, b, fst)
	aw := min(snap(l.S(30)), snap(b.Dx()*0.5))
	ab := paintengine2d.XYWH(b.Max.X-aw, b.Min.Y, aw-l.S(4), b.Dy())
	if st.Pressed() || open {
		ab = ab.Translate(paintengine2d.Pt(0, snap(l.S(2))))
	}
	glyph := c.text2
	if st.Disabled() {
		glyph = c.textDis
	}
	winChevron(l, ctx, ab, DirDown, glyph)
	if st.Disabled() {
		fg = c.textDis
	} else {
		fg = c.text
	}
	pad := l.metrics.FieldPad
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, ab.Min.X-b.Min.X-pad, b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.focusVisual(l, ctx, b, l.rx(4))
	}
}

// DrawSpinner: NumberBox's spin buttons — a button face split in two with
// subtle hot halves and chevrons.
func (e fluentEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	c.button(l, ctx, b, st&^(StateHovered|StatePressed|StatePrimary))
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(h paintengine2d.Rect, dir Direction, hover, press bool) {
		in := h.Inset(2 * lw)
		switch {
		case st.Disabled():
		case press:
			c.subtleBox(l, ctx, in, c.subtle2)
		case hover:
			c.subtleBox(l, ctx, in, c.subtle)
		}
		col := c.text2
		if st.Disabled() {
			col = c.textDis
		}
		winChevron(l, ctx, h, dir, col)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

// DrawTabBar is TabView's strip: Mica with the page's top line along its
// bottom (the selected tab opens it).
func (fluentEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.cardStroke))
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

// DrawTab is a TabView item: the selected tab is the page colour with 8px
// top corners, a hairline and ears curving into the page; the others are
// bare text, a subtle rounded fill when hot, divided by short lines.
func (fluentEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 8*lw || b.Dy() < 8*lw {
		return
	}
	top := b.Min.Y + snap(l.S(4))
	e := float32(0)
	if selected {
		e = min(snap(l.S(6)), b.Dx()*0.2)
	}
	body := paintengine2d.XYWH(b.Min.X+e, top, b.Dx()-2*e, b.Max.Y-top)
	r := l.rx(8)
	fg := c.text2
	switch {
	case selected:
		// The neighbours' dividers stop at the selected tab: clear the
		// room its ears take, above the ears.
		if e > 0 {
			clear := paintengine2d.NewPath()
			clear.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, e, b.Dy()-e-lw))
			clear.AddRect(paintengine2d.XYWH(b.Max.X-e, b.Min.Y, e, b.Dy()-e-lw))
			ctx.DrawPath(clear, paintengine2d.Fill(c.bg))
		}
		ctx.DrawPath(fluentTabPath(b, top, r, e, true), paintengine2d.Fill(c.page))
		ctx.DrawPath(fluentTabPath(b.Inset(lw*0.5), top+lw*0.5, r, e, false), paintengine2d.Paint{Color: c.cardStroke, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: lw, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		fg = c.text
	case st.Disabled():
		fg = c.textDis
	default:
		hb := paintengine2d.XYWH(body.Min.X+lw, body.Min.Y, body.Dx()-2*lw, body.Dy()-2*lw)
		switch {
		case st.Pressed():
			ctx.DrawPath(RoundRectPath(hb, r, r, lw*2, lw*2), paintengine2d.Fill(c.subtle2))
			fg = c.text
		case st.Hovered():
			ctx.DrawPath(RoundRectPath(hb, r, r, lw*2, lw*2), paintengine2d.Fill(c.subtle))
			fg = c.text
		case !st.Last():
			h := min(snap(l.S(16)), body.Dy()-4*lw)
			ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, snap((body.Min.Y+body.Max.Y-h)*0.5), lw, h), paintengine2d.Fill(c.divider))
		}
	}
	l.drawFittedText(ctx, l.body, label, body, fg, AlignCenter, 8)
	if st.Focused() && selected {
		c.focusVisual(l, ctx, paintengine2d.XYWH(body.Min.X, body.Min.Y, body.Dx(), body.Dy()-lw), r)
	}
}

// DrawPanel: a raised panel is a card; a flat one is the window.
func (fluentEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := fluentColors(l)
	if !raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		return
	}
	c.card8(l, ctx, b, true)
}

func (fluentEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(fluentColors(l).bg))
}

// DrawMenuTitle is a MenuBar item: text on Mica, a rounded subtle fill when
// hot, a deeper one while its menu is open; keyboard focus shows the focus
// visual.
func (e fluentEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := fluentColors(l)
	hb := winSnap(b).Inset(snap(l.S(2)))
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.textDis
	case open:
		e.MenuHighlight(l, ctx, hb, true)
	case st.Pressed() || st.Hovered():
		e.MenuHighlight(l, ctx, hb, false)
	case st.Focused():
		c.focusVisual(l, ctx, hb, l.rx(4))
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame is a menu flyout: 8px corners, the flyout fill and its
// hairline.
func (fluentEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(8)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.flyout))
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.flyoutStroke))
}

// DrawMenuItem is a MenuFlyoutItem: a rounded subtle row 4px in from the
// flyout's sides, checks and bullets in the icon column, shortcuts and the
// submenu chevron in the secondary text colour.
func (e fluentEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := fluentColors(l)
	lw := winPx(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := snap(b.Min.X-ch.PadL+lw), snap(b.Max.X+ch.PadR-lw)
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(c.divider))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		k := l.S(4)
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y+lw, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy()-2*lw)
		col := c.subtle
		if st.Pressed() {
			col = c.subtle2
		}
		c.subtleBox(l, ctx, hb, col)
	}
	fg, sc := c.text, c.text2
	if st.Disabled() {
		fg, sc = c.textDis, c.textDis
	}
	gw := ch.CheckCol()
	side := min(b.Dy()-4*lw, gw-2*lw, snap(l.S(16)))
	box := winSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	switch {
	case row.Checked && row.Radio:
		ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
		ctx.DrawCircle(ctr, l.S(3), paintengine2d.Fill(fg))
	case row.Checked:
		winTick(ctx, box, fg, max(lw*1.1, box.Dx()*0.08))
	case !row.Radio && row.Icon != IconNone:
		l.drawToolIcon(ctx, box, row.Icon, fg)
	}
	winMenuText(l, ctx, b, ch, row, fg, sc, func(ab paintengine2d.Rect) {
		winChevron(l, ctx, ab, DirRight, sc)
	})
}

// DrawProgressBar is WinUI's progress bar: a 3px rounded accent bar over a
// 1px track, centred; indeterminate slides a segment.
func (fluentEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	h := min(snap(l.S(3)), b.Dy())
	if b.Dx() < 4*lw || h < lw {
		return
	}
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	bar := paintengine2d.XYWH(b.Min.X, cy-snap(h*0.5), b.Dx(), h)
	col := c.accent[0]
	if st.Disabled() {
		col = c.strongFillDis
	}
	r := h * 0.5
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.4
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, bar.Min.Y, span, h).Intersect(bar)
		if seg.Dx() >= lw {
			ctx.DrawRoundRect(seg, r, r, paintengine2d.Fill(col))
		}
		return
	}
	w := snap(bar.Dx() * clamp1(t))
	track := paintengine2d.XYWH(bar.Min.X+w, cy-snap(lw*0.5), bar.Dx()-w, lw)
	if !track.Empty() {
		ctx.DrawRect(track, paintengine2d.Fill(c.strong))
	}
	if w >= lw {
		ctx.DrawRoundRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, r, paintengine2d.Fill(col))
	}
}

// DrawSlider is WinUI's slider: a 4px track, accent up to the thumb — a
// 20px white disc in the elevation border with an accent dot that grows
// under the pointer and shrinks while dragged.
func (fluentEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	t = clamp1(t)
	d := min(snap(l.S(20)), b.Dy())
	if b.Dx() < d+2*lw || d < 8*lw {
		return
	}
	cy := (b.Min.Y + b.Max.Y) * 0.5
	x0, x1 := b.Min.X+d*0.5, b.Max.X-d*0.5
	tx := x0 + (x1-x0)*t
	th := snap(l.S(4))
	i := fluentIndex(st)
	rest, fill := c.strongFill, c.accent[i]
	if i == 3 {
		rest, fill = c.strongFillDis, c.accent[3]
	}
	tr := paintengine2d.XYWH(b.Min.X+lw, snap(cy-th*0.5), b.Dx()-2*lw, th)
	ctx.DrawRoundRect(tr, th*0.5, th*0.5, paintengine2d.Fill(rest))
	if w := tx - tr.Min.X; w > th*0.5 {
		ctx.DrawRoundRect(paintengine2d.XYWH(tr.Min.X, tr.Min.Y, w, th), th*0.5, th*0.5, paintengine2d.Fill(fill))
	}
	thumb := paintengine2d.XYWH(snap(tx-d*0.5), snap(cy-d*0.5), d, d)
	ctr := paintengine2d.Pt(thumb.Min.X+d*0.5, thumb.Min.Y+d*0.5)
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(c.solid))
	c.edge(l, ctx, thumb, d*0.5, c.elev, c.dark)
	dot := [4]float32{0.43, 0.58, 0.36, 0.43}[i] * d * 0.5 * 1.2
	ctx.DrawCircle(ctr, dot, paintengine2d.Fill(fill))
	if st.Focused() && !st.Disabled() {
		c.focusVisual(l, ctx, b, l.rx(4))
	}
}

// DrawSwitch is the ToggleSwitch: a 40×20 pill, outlined with a strong
// knob when off, the accent with a white knob when on; the knob grows
// under the pointer and stretches while pressed.
func (e fluentEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := fluentColors(l)
	lw := winPx(l)
	tw, th := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	track := winSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 3*track.Dy()*0.5 || track.Dy() < 8*lw {
		return
	}
	r := track.Dy() * 0.5
	i := fluentIndex(st)
	k := track.Dy() / 20
	var knob paintengine2d.Color
	if on {
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.accent[i]))
		knob = c.onAccent[0]
		if i == 3 {
			knob = c.onAccent[2]
		}
	} else {
		c.fill(l, ctx, track, r, c.alt[i])
		edge := c.strong
		if i == 3 {
			edge = c.strongDis
		}
		winRing(ctx, track, r, lw, paintengine2d.Fill(edge))
		knob = c.strongFill
		if i == 3 {
			knob = c.strongFillDis
		}
	}
	kw, kh := 12*k, 12*k
	switch i {
	case 1:
		kw, kh = 14*k, 14*k
	case 2:
		kw, kh = 17*k, 14*k
	}
	cy := (track.Min.Y + track.Max.Y) * 0.5
	kx := track.Min.X + r - 6*k
	if i == 1 {
		kx = track.Min.X + r - 7*k
	}
	if on {
		kx = track.Max.X - r + 6*k - kw
		if i == 1 {
			kx = track.Max.X - r + 7*k - kw
		}
	}
	ctx.DrawRoundRect(paintengine2d.XYWH(kx, cy-kh*0.5, kw, kh), kh*0.5, kh*0.5, paintengine2d.Fill(knob))
	var lb paintengine2d.Rect
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.textDis
		}
		lb = paintengine2d.XYWH(track.Max.X+l.S(12), b.Min.Y, b.Max.X-track.Max.X-l.S(12), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		c.controlFocus(l, ctx, b, track, lb, label)
	}
}

// DrawListRow is a ListView item: a rounded subtle row with the accent pill
// at the left when selected.
func (e fluentEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := fluentColors(l)
	item := fluentItemRect(l, b)
	c.subtleBox(l, ctx, item, c.rowFill(st))
	if st.Checked() {
		c.pill(l, ctx, item, st)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(item.Min.X+l.S(12), b.Min.Y, item.Dx()-l.S(16), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		// Round the row, clear of the item and its pill.
		e.ItemFocus(l, ctx, winSnap(b), st)
	}
}

// DrawTreeRow is a TreeView item: chevrons, indentation, and the list
// item's rounded row and accent pill.
func (e fluentEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := fluentColors(l)
	item := fluentItemRect(l, b)
	c.subtleBox(l, ctx, item, c.rowFill(st))
	if st.Checked() {
		c.pill(l, ctx, item, st)
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := item.Min.X + l.S(6) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text2)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	lx := x + indent + l.S(4)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, item.Max.X-lx-l.S(4), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, winSnap(b), st)
	}
}

// DrawTableHeader is a details-view column header: plain text on the view,
// hairline dividers, a rounded subtle fill when hot and the sort chevron
// centred at the top.
func (fluentEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.field))
	switch {
	case st.Disabled():
	case st.Pressed():
		c.subtleBox(l, ctx, b.Inset(snap(l.S(2))), c.subtle2)
	case st.Hovered():
		c.subtleBox(l, ctx, b.Inset(snap(l.S(2))), c.subtle)
	}
	h := min(snap(l.S(16)), b.Dy()-4*lw)
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, snap((b.Min.Y+b.Max.Y-h)*0.5), lw, h), paintengine2d.Fill(c.divider))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.divider))
	if sorted {
		dir := DirDown
		if asc {
			dir = DirUp
		}
		winChevron(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), snap(l.S(9))), dir, c.text2)
	}
	fg := c.text2
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(10), b.Min.Y, b.Dx()-l.S(14), b.Dy()), fg, AlignStart, 0)
}

// DrawTableCell: a selected row is tinted with the accent (a table has no
// pill to carry the selection), a hot one takes the subtle fill; the focused
// row's focus visual spans the row.
func (fluentEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := fluentColors(l)
	r := winSnap(b)
	switch {
	case st.Checked() && (st.Backdrop() || st.Disabled()):
		ctx.DrawRect(r, paintengine2d.Fill(c.subtle))
	case st.Checked():
		ctx.DrawRect(r, paintengine2d.Fill(c.tint))
	case st.Hovered() && !st.Disabled():
		ctx.DrawRect(r, paintengine2d.Fill(c.subtle))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	winCellText(l, ctx, b, label, align, face, fg)
}

func (fluentEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(fluentColors(l).bg))
}

func (fluentEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := fluentColors(l)
	in := winSnap(b).Inset(winPx(l))
	fg := c.toolFace(l, ctx, in, st)
	winToolContent(l, ctx, b, label, icon, fg)
	if st.Focused() && !st.Disabled() {
		c.focusVisual(l, ctx, in, l.rx(4))
	}
}

// DrawStatusBar: Mica with a hairline on top and divided parts.
func (fluentEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.divider))
	winStatusParts(l, ctx, b, parts, c.text, c.divider, paintengine2d.Color{})
}

// DrawTitleBar is a page heading in the strong body style.
func (fluentEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := fluentColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	winHeading(l, ctx, b, title, subtitle, c.text, c.text2, true)
}

// DrawAccordionHeader is an Expander header: a 4px-rounded card (square
// bottom corners when open) with the title at the left and the chevron in
// a subtle button at the right.
func (e fluentEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := fluentColors(l)
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 8*lw || b.Dy() < 6*lw {
		return
	}
	r := l.rx(4)
	rb := r
	if expanded {
		rb = 0
	}
	i := fluentIndex(st)
	fill := c.card
	if i > 0 {
		fill = c.ctl[i]
	}
	in := b.Inset(lw)
	fp := paintengine2d.NewPath()
	winAddRoundRect(fp, in, max(r-lw, 0), max(r-lw, 0), max(rb-lw, 0), max(rb-lw, 0))
	ctx.DrawPath(fp, paintengine2d.Fill(fill))
	p := paintengine2d.NewPath()
	winAddRoundRect(p, b, r, r, rb, rb)
	winAddRoundRect(p, in, max(r-lw, 0), max(r-lw, 0), max(rb-lw, 0), max(rb-lw, 0))
	ctx.DrawPath(p, paintengine2d.Paint{Color: c.cardStroke, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
	bs := min(snap(l.S(28)), b.Dy()-4*lw)
	btn := paintengine2d.XYWH(b.Max.X-bs-snap(l.S(6)), snap((b.Min.Y+b.Max.Y-bs)*0.5), bs, bs)
	if i == 1 || i == 2 {
		c.subtleBox(l, ctx, btn, c.subtle)
	}
	dir := DirDown
	if expanded {
		dir = DirUp
	}
	fg := c.text
	if i == 3 {
		fg = c.textDis
	}
	winChevron(l, ctx, btn, dir, fg)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(14), b.Min.Y, btn.Min.X-b.Min.X-l.S(18), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && i != 3 {
		c.focusVisual(l, ctx, b, r)
	}
}

// DrawSeparator is the divider stroke.
func (fluentEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := fluentColors(l)
	lw := winPx(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, snap(b.Min.Y+l.S(4)), lw, snap(b.Dy()-l.S(8))), paintengine2d.Fill(c.divider))
		return
	}
	y := snap((b.Min.Y + b.Max.Y) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(c.divider))
}

// DrawSplitter is invisible at rest; under the pointer a short rounded grip
// shows where to drag.
func (fluentEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := fluentColors(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if !st.Hovered() && !st.Pressed() {
		return
	}
	w, h := snap(l.S(4)), snap(l.S(24))
	if !vertical {
		w, h = h, w
	}
	w, h = min(w, b.Dx()), min(h, b.Dy())
	g := paintengine2d.XYWH(snap((b.Min.X+b.Max.X-w)*0.5), snap((b.Min.Y+b.Max.Y-h)*0.5), w, h)
	r := min(w, h) * 0.5
	ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(c.strongFill))
}

// DrawTooltip is a Windows 11 tool tip: 4px corners, the flyout fill and
// hairline.
func (fluentEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := fluentColors(l)
	tb := b
	b = winSnap(b)
	lw := winPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	r := l.rx(4)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.flyout))
	winRing(ctx, b, r, lw, paintengine2d.Fill(c.flyoutStroke))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(8)
	}
	// The bubble is sized to the text plus the padding: let the text use
	// the right padding rather than lose its last letters to rounding.
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(tb.Min.X+pad, tb.Min.Y, tb.Dx()-pad-winPx(l), tb.Dy()), c.text, AlignStart, 0)
}

// DrawMessageIcon: InfoBar's glyphs — filled discs in the status colours
// with the glyph in the text-on-accent colour.
func (fluentEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := fluentColors(l)
	on := c.onAccent[0]
	switch icon {
	case IconNone:
	case IconError:
		winFlatDisc(l, ctx, b, c.critical, "×", ReadableOn(c.critical, 3, Hex("#ffffff"), Hex("#000000")))
	case IconInfo:
		winFlatDisc(l, ctx, b, c.accent[0], "i", on)
	case IconQuestion:
		winFlatDisc(l, ctx, b, c.accent[0], "?", on)
	case IconWarning:
		winFlatDisc(l, ctx, b, c.caution, "!", ReadableOn(c.caution, 3, Hex("#ffffff"), Hex("#000000")))
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// ---- packs -----------------------------------------------------------------------------------------------

func fluentPack(name, label string, fam ThemeName, scheme int, summary string) ThemePack {
	sc := fluentLight
	if scheme == 1 {
		sc = fluentDark
	}
	h := func(k string) paintengine2d.Color { return Hex(sc[k]) }
	bg := h("bg")
	flat := func(k string) paintengine2d.Color { c := h(k); return Mix(bg, c, c.A) }
	pal := Palette{
		Background: bg, Surface: bg, SurfaceAlt: h("page"),
		Border: flat("stroke2"), Divider: flat("divider"),
		Text: h("text"), TextMuted: h("text2"), TextOnAccent: h("onAccent"),
		Accent: h("accent"), AccentHover: flat("accent2"), AccentPress: flat("accent3"),
		Field: h("field"), FieldBorder: flat("strong"),
		Focus: flat("focusOuter"), Selection: h("selText"),
		Track: flat("strongFill"), Thumb: flat("strongFill"),
		Highlight: flat("subtle"), Shadow: paintengine2d.RGBA(0, 0, 0, 0.14),
		MenuHover: Mix(h("flyout"), h("subtle"), h("subtle").A), MenuHoverBorder: h("accent"), MenuGutter: h("flyout"),
		Danger: h("critical"), Success: h("success"), Warning: h("caution"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: h("field"), BevelDark: flat("stroke2"),
	}
	if scheme == 1 {
		pal.Shadow, pal.Overlay = paintengine2d.RGBA(0, 0, 0, 0.4), paintengine2d.RGBA(0, 0, 0, 0.5)
	}
	tok := ThemeTokens{
		Engine:  "fluent",
		Bevel:   BevelFluentAccent,
		Family:  fam,
		Palette: pal,
		Params:  map[string]float32{"scheme": float32(scheme)},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Pressed = ChromeState{Fill: Mix(pal.MenuHover, pal.Text, 0.04), Border: pal.Accent}
	tok.Selected = ChromeState{Fill: pal.Highlight, Border: pal.Accent}
	tok.Focus = ChromeState{Fill: pal.Field, Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: 2021, Lineage: "Windows", Summary: summary,
		Era: EraFluent, Palette: fam, Tokens: tok,
	}
}

func fluentPacks() []ThemePack {
	return []ThemePack{
		fluentPack("fluent", "Fluent", ThemeLight, 0, "Windows 11's Fluent (WinUI 3): 4px and 8px corners, translucent control fills over Mica, accent check boxes, pill selection indicators, the two-colour focus visual."),
		fluentPack("fluent-night", "Fluent Dark", ThemeDark, 1, "Windows 11's dark theme: Fluent's shapes on #202020 Mica with the light-blue accent and black text on it."),
	}
}
