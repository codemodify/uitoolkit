package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// adwaitaEngine paints GNOME's Adwaita as libadwaita draws it (GNOME 42 and
// later): flat 6px-rounded buttons washed with the text colour at 10% (15%
// hovered, 30% pressed), the accent-filled suggested action, a 2px focus
// ring of the accent at half alpha drawn inside the control, check boxes
// and radios that fill with the accent behind a white glyph, thin overlay
// scroll bars that widen under the pointer, the pill switch, a 4px scale
// trough filled with the accent up to a round white knob, flat notebook
// tabs underlined in the accent, popover menus with rounded rows, entries
// washed like buttons, and list selections tinted with the accent at 25%.
//
// Every wash is currentColor at a fixed alpha, as the stylesheet writes it,
// so a button reads the same on the window, a view or a card; light text is
// 80% black, so a 10% wash is 8% of near-black. Colours are libadwaita's
// named colours (window, view, headerbar, card, popover, accent…), alphas
// and sizes the stylesheet's; radii follow GNOME 42–47 (6px controls, 12px
// cards and popovers).
//
// The GTK 3 Adwaita of 2014 (GTK 3.14) is the same engine with scheme 2:
// see engine_adwaita_gtk3.go.
//
// Pack data:
//
//	params  "scheme"  0 libadwaita light, 1 libadwaita dark, 2 GTK 3.14
//	        "shadow"  scales the popover / dialog shadows (0 = none)
//	extra   any key of the scheme tables (adwLight, adwDark, adw314)
//	        overrides that colour, e.g. "accentBg" for another accent.
type adwaitaEngine struct{ BaseEngine }

func init() {
	RegisterEngine(adwaitaEngine{})
	for _, p := range adwaitaPacks() {
		RegisterPack(p)
	}
}

func (adwaitaEngine) ID() string { return "adwaita" }

// DefaultMetrics are the stylesheet's sizes at the toolkit's 16px UI font
// (GNOME's 11pt is 14.7px, close enough to keep CSS pixels): 34px buttons
// and entries, 20px check boxes and radios, 46×26 switches, 20px scale
// knobs, 8px progress troughs, 32px menu rows, overlay scroll bars.
func (adwaitaEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    12, RadiusSmall: 6,
		ControlH: 34, FieldH: 34, ComboH: 34,
		Checkbox: 20, Radio: 20,
		MenuItemH: 32, MenuBarH: 34, TabH: 36, RowH: 32,
		TitleBar: 46, HeaderH: 28, ProgressH: 8, SliderH: 30, Thumb: 20,
		Scroll: 14, Pad: 12, FieldPad: 9, FocusWidth: 2, Border: 1,
		ToolBarH: 46, StatusBarH: 28, SpinnerW: 26, SwitchW: 46, SwitchH: 26,
	}
}

// ---- scheme tables --------------------------------------------------------------

// adwScheme maps a named colour to "#rrggbb" or "#rrggbbaa".
type adwScheme map[string]string

// adwLight is libadwaita's light palette. "fg" is the window, view and
// popover text colour, which the stylesheet also uses (as currentColor) for
// every wash.
var adwLight = adwScheme{
	"window": "#fafafb", "view": "#ffffff", "fg": "#000006cc",
	"headerbar": "#ffffff", "headerbarShade": "#0000061f",
	"card": "#ffffff", "cardShade": "#00000612",
	"dialog": "#fafafb", "popover": "#ffffff", "popoverEdge": "#00000024",
	"accentBg": "#3584e4", "accentFg": "#ffffff", "accent": "#0461be",
	"destructive": "#c30000", "success": "#007c3d", "warning": "#905400", "error": "#c30000",
	"shade": "#00000612", "outline": "#ffffff59", "outlineHot": "#ffffff99",
	"knob": "#ffffff", "tooltip": "#000006cc", "tooltipFg": "#ffffff", "dimmer": "#00000640",
}

// adwDark is libadwaita's dark palette.
var adwDark = adwScheme{
	"window": "#222226", "view": "#1d1d20", "fg": "#ffffff",
	"headerbar": "#2e2e32", "headerbarShade": "#0000065c",
	"card": "#ffffff14", "cardShade": "#0000065c",
	"dialog": "#36363a", "popover": "#36363a", "popoverEdge": "#00000024",
	"accentBg": "#3584e4", "accentFg": "#ffffff", "accent": "#81d0ff",
	"destructive": "#ff938c", "success": "#78e9ab", "warning": "#ffc252", "error": "#ff938c",
	"shade": "#00000640", "outline": "#00000c53", "outlineHot": "#00000c8f",
	"knob": "#d2d2d2", "tooltip": "#000006cc", "tooltipFg": "#ffffff", "dimmer": "#00000680",
}

// adwSchemes indexes the tables by the "scheme" param (adw314 is GTK 3.14).
var adwSchemes = [...]adwScheme{adwLight, adwDark, adw314}

// The stylesheet's washes: currentColor at these alphas.
const (
	adwWashBtn          = 0.10 // button, entry
	adwWashBtnHover     = 0.15
	adwWashBtnActive    = 0.30
	adwWashChecked      = 0.30 // toggled button
	adwWashCheckedHover = 0.35
	adwWashCheckedPress = 0.40
	adwWashFlatHover    = 0.07 // flat button
	adwWashFlatActive   = 0.16
	adwWashFlatChecked  = 0.10
	adwWashMenuHover    = 0.10 // popover menu row
	adwWashMenuActive   = 0.19
	adwWashTrough       = 0.15 // borders, rings, empty troughs
	adwWashTroughHover  = 0.20
	adwWashRowHover     = 0.04 // activatable list row
	adwWashRowActive    = 0.08
	adwSelAlpha         = 0.25 // selected row: the accent at a quarter
	adwSelHoverAlpha    = 0.32
	adwSelActiveAlpha   = 0.39
	adwDimOpacity       = 0.55 // dim-label, accelerators
	adwDisabledOpacity  = 0.50 // insensitive widgets
	adwFocusAlpha       = 0.50 // focus ring: the accent at half alpha
	adwTextSelAlpha     = 0.30 // text selection in a focused entry
	adwHeaderAlpha      = 0.40 // column view titles (70% hovered)
)

// ---- resolved colour set ----------------------------------------------------------

// adwSet is a look's resolved Adwaita colours, built once per look.
type adwSet struct {
	gtk3, dark bool

	win, view, headerbar, card, popover, dialog paintengine2d.Color
	// fg is currentColor with its alpha (libadwaita's text is 80% black).
	fg paintengine2d.Color
	// Opaque label colours over the window, a view and a popover.
	text, viewText, dim, disText, popText paintengine2d.Color

	accentBg, accentFg, accent paintengine2d.Color
	accentHover, accentPress   paintengine2d.Color // suggested-action overlays
	accentDis, accentFgDis     paintengine2d.Color // disabled suggested action

	border, borderOnView                  paintengine2d.Color // opaque cC 15%
	headerShade, cardEdge, popoverEdge    paintengine2d.Color
	focus                                 paintengine2d.Color
	sel, selHover, selActive, selBackdrop paintengine2d.Color
	outline, outlineHot, knob, knobOn     paintengine2d.Color
	tooltip, tooltipFg, tooltipEdge       paintengine2d.Color
	headerText, headerHot                 paintengine2d.Color

	// Popup shadows (the CSS box-shadows, see adwShadowSpecs).
	shadows [3][]adwShadow

	g3 *adw314Set // GTK 3.14 materials (scheme 2)
}

type adwKey struct{}

// adwColors is the look's resolved colour set (built once per look).
func adwColors(l *Classic) *adwSet {
	return l.Memo(adwKey{}, func() any { return adwBuild(l) }).(*adwSet)
}

func adwSchemeIndex(l *Classic) int {
	i := int(l.P("scheme", 0))
	if i < 0 || i >= len(adwSchemes) {
		i = 0
	}
	return i
}

// adwOver composites a straight-alpha colour on an opaque one.
func adwOver(base, c paintengine2d.Color) paintengine2d.Color {
	return Mix(base, paintengine2d.RGB(c.R, c.G, c.B), c.A)
}

// adwInk is the stylesheet's rgb(0 0 6 / a): its shadows and dark overlays.
func adwInk(a float32) paintengine2d.Color {
	return paintengine2d.RGBA(0, 0, 6.0/255, a)
}

func adwBuild(l *Classic) *adwSet {
	i := adwSchemeIndex(l)
	sc := adwSchemes[i]
	col := func(k string) paintengine2d.Color {
		v, ok := sc[k]
		if !ok {
			v = adwLight[k]
		}
		return l.X(k, Hex(v))
	}
	c := &adwSet{gtk3: i == 2, dark: i == 1}
	c.win, c.view, c.headerbar = col("window"), col("view"), col("headerbar")
	c.dialog, c.popover = col("dialog"), col("popover")
	c.card = adwOver(c.win, col("card"))
	c.fg = col("fg")
	c.text = adwOver(c.win, c.fg)
	c.viewText = adwOver(c.view, c.fg)
	c.popText = adwOver(c.popover, c.fg)
	c.dim = adwOver(c.win, c.wash(adwDimOpacity))
	c.disText = adwOver(c.win, c.wash(adwDisabledOpacity))
	c.accentBg, c.accentFg, c.accent = col("accentBg"), col("accentFg"), col("accent")
	c.accentHover = adwOver(c.accentBg, c.accentFg.WithAlpha(0.1))
	c.accentPress = adwOver(c.accentBg, adwInk(0.2))
	c.accentDis = Mix(c.win, c.accentBg, adwDisabledOpacity)
	c.accentFgDis = Mix(c.accentDis, c.accentFg, adwDisabledOpacity)
	c.border = adwOver(c.win, c.wash(adwWashTrough))
	c.borderOnView = adwOver(c.view, c.wash(adwWashTrough))
	c.headerShade = adwOver(c.headerbar, col("headerbarShade"))
	c.cardEdge = adwOver(c.win, col("cardShade"))
	c.popoverEdge = adwOver(c.popover, col("popoverEdge"))
	c.focus = c.accent.WithAlpha(adwFocusAlpha)
	c.sel = c.accentBg.WithAlpha(adwSelAlpha)
	c.selHover = c.accentBg.WithAlpha(adwSelHoverAlpha)
	c.selActive = c.accentBg.WithAlpha(adwSelActiveAlpha)
	c.selBackdrop = c.wash(adwWashBtn)
	c.outline, c.outlineHot = col("outline"), col("outlineHot")
	c.knob, c.knobOn = col("knob"), Hex("#ffffff")
	c.tooltip, c.tooltipFg = col("tooltip"), col("tooltipFg")
	c.tooltipEdge = paintengine2d.RGBA(1, 1, 1, 0.1)
	c.headerText = adwOver(c.view, c.wash(adwHeaderAlpha))
	c.headerHot = adwOver(c.view, c.wash(0.7))
	c.shadows = adwShadowSpecs(l, i)
	if c.gtk3 {
		c.g3 = adw314Build(col)
		c.g3.dash = []float32{adwPx(l), adwPx(l)}
		c.focus = c.g3.focus
	}
	return c
}

// wash is currentColor at alpha a (the stylesheet's alpha(currentColor, a)).
func (c *adwSet) wash(a float32) paintengine2d.Color {
	return c.fg.WithAlpha(c.fg.A * a)
}

// ---- drawing helpers ----------------------------------------------------------------

// adwPx is one line of the look: 1 device pixel at 1x, 2 at 2x.
func adwPx(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// adwSnap puts a rect on the pixel grid so hairlines stay crisp.
func adwSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

// adwPill is the radius that rounds b into a stadium (0 for square looks).
func adwPill(l *Classic, b paintengine2d.Rect) float32 {
	if l.square() {
		return 0
	}
	return max(min(b.Dx(), b.Dy())*0.5, 0)
}

// adwR is design radius r (1x px) for a shape of size b, never more than
// half its short side.
func adwR(l *Classic, r float32, b paintengine2d.Rect) float32 {
	return min(l.rx(r), adwPill(l, b))
}

// adwRing strokes a w-wide ring just inside b, following radius r.
func adwRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, w float32, col paintengine2d.Color) {
	if b.Dx() < 2*w || b.Dy() < 2*w || col.A <= 0 {
		return
	}
	ri := max(r-w*0.5, 0)
	ctx.DrawRoundRect(b.Inset(w*0.5), ri, ri, paintengine2d.StrokePaint(col, w))
}

// adwFrame fills b with edge and its lw-inset with fill, both rounded r.
func adwFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, edge, fill paintengine2d.Color) {
	if b.Empty() {
		return
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(edge))
	if b.Dx() > 2*lw && b.Dy() > 2*lw {
		ri := max(r-lw, 0)
		ctx.DrawRoundRect(b.Inset(lw), ri, ri, paintengine2d.Fill(fill))
	}
}

// adwCentered is a w×h rect centred in b.
func adwCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return paintengine2d.XYWH(b.Min.X+(b.Dx()-w)*0.5, b.Min.Y+(b.Dy()-h)*0.5, w, h)
}

// adwStroke is a round-capped stroke for symbolic glyphs.
func adwStroke(col paintengine2d.Color, w float32) paintengine2d.Paint {
	return paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
}

// adwChevron strokes a pan-*-symbolic chevron in an s-sized icon box
// centred in b: 2px strokes on a 16px icon, scaled with s.
func adwChevron(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color, s float32) {
	if b.Empty() || col.A <= 0 || s < 4 {
		return
	}
	g := adwCentered(b, s, s)
	cx, cy := (g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5
	a := s * 0.28   // half the span
	h := s * 0.14   // half the depth
	w := s / 16 * 2 // stroke
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-a, cy+h)
		p.LineTo(cx, cy-h)
		p.LineTo(cx+a, cy+h)
	case DirDown:
		p.MoveTo(cx-a, cy-h)
		p.LineTo(cx, cy+h)
		p.LineTo(cx+a, cy-h)
	case DirLeft:
		p.MoveTo(cx+h, cy-a)
		p.LineTo(cx-h, cy)
		p.LineTo(cx+h, cy+a)
	default:
		p.MoveTo(cx-h, cy-a)
		p.LineTo(cx+h, cy)
		p.LineTo(cx-h, cy+a)
	}
	ctx.DrawPath(p, adwStroke(col, w))
}

// adwTick strokes the check glyph into the icon area g.
func adwTick(ctx *paintengine2d.Context, g paintengine2d.Rect, col paintengine2d.Color, w float32) {
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X+g.Dx()*0.15, g.Min.Y+g.Dy()*0.53)
	p.LineTo(g.Min.X+g.Dx()*0.41, g.Min.Y+g.Dy()*0.78)
	p.LineTo(g.Min.X+g.Dx()*0.87, g.Min.Y+g.Dy()*0.27)
	ctx.DrawPath(p, adwStroke(col, w))
}

// ---- parts: faces ---------------------------------------------------------------------

// button paints a push button face and returns its label colour: the wash,
// the toggled wash, or the accent fill of a suggested (default) action.
// flat buttons show nothing until hovered.
func (c *adwSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, flat bool) paintengine2d.Color {
	if b.Empty() {
		return c.text
	}
	if c.gtk3 {
		return c.g3.button(l, ctx, b, st, flat)
	}
	r := adwR(l, 6, b)
	dis := st.Disabled()
	if st.Primary() && !flat {
		fill := c.accentBg
		switch {
		case dis:
			fill = c.accentDis
		case st.Pressed():
			fill = c.accentPress
		case st.Hovered():
			fill = c.accentHover
		}
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		if dis {
			return c.accentFgDis
		}
		return c.accentFg
	}
	var a float32
	switch {
	case st.Toggle() && st.Checked():
		switch {
		case flat && st.Pressed():
			a = 0.19
		case flat && st.Hovered():
			a = 0.13
		case flat:
			a = adwWashFlatChecked
		case st.Pressed():
			a = adwWashCheckedPress
		case st.Hovered():
			a = adwWashCheckedHover
		default:
			a = adwWashChecked
		}
	case st.Pressed():
		a = adwWashBtnActive
		if flat {
			a = adwWashFlatActive
		}
	case st.Hovered():
		a = adwWashBtnHover
		if flat {
			a = adwWashFlatHover
		}
	case !flat:
		a = adwWashBtn
	}
	if dis {
		a *= adwDisabledOpacity
	}
	if a > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(a)))
	}
	if dis {
		return c.disText
	}
	return c.text
}

// field is an entry: washed like a button.
func (c *adwSet) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if b.Empty() {
		return
	}
	if c.gtk3 {
		c.g3.field(l, ctx, b, st)
		return
	}
	a := float32(adwWashBtn)
	if st.Disabled() {
		a *= adwDisabledOpacity
	}
	r := adwR(l, 6, b)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(a)))
}

// row paints a list / tree / table row's hover and selection and returns
// the label colour. Selections stay when the view loses focus and wash out
// to a neutral tint only while the window is in the backdrop.
func (c *adwSet) row(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if c.gtk3 {
		return c.g3.row(ctx, b, st)
	}
	var fill paintengine2d.Color
	switch {
	case st.Checked() && st.Backdrop():
		fill = c.selBackdrop
	case st.Checked() && st.Pressed():
		fill = c.selActive
	case st.Checked() && st.Hovered():
		fill = c.selHover
	case st.Checked():
		fill = c.sel
	case st.Pressed():
		fill = c.wash(adwWashRowActive)
	case st.Hovered():
		fill = c.wash(adwWashRowHover)
	}
	if fill.A > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(fill))
	}
	return c.viewText
}

// cardFace is a card or boxed list: the card colour in 12px corners with
// its hairline shade as the edge.
func (c *adwSet) cardFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if c.gtk3 {
		c.g3.frame(l, ctx, b, c.g3.view)
		return
	}
	adwFrame(ctx, adwSnap(b), adwR(l, 12, b), adwPx(l), c.cardEdge, c.card)
}

func (e adwaitaEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := adwColors(l)
	switch role {
	case RoleButton, RoleCombo:
		return c.button(l, ctx, b, st, false)
	case RoleTool:
		return c.button(l, ctx, b, st, true)
	case RoleField:
		c.field(l, ctx, b, st)
		if st.Disabled() {
			return c.disText
		}
		return c.viewText
	case RoleCheck:
		l.Engine().CheckIndicator(l, ctx, b, st, st.Checked())
		return c.text
	case RoleRow:
		return c.row(ctx, b, st)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			l.Engine().MenuHighlight(l, ctx, b, false)
			return l.Engine().MenuTextColor(l, true)
		}
		return c.popText
	case RoleTab:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			ctx.DrawRect(b, paintengine2d.Fill(c.wash(adwWashFlatHover)))
		}
		return c.text
	case RoleThumb:
		a := float32(0.2)
		switch {
		case st.Pressed():
			a = 0.6
		case st.Hovered():
			a = 0.4
		}
		r := adwPill(l, b)
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(a)))
		return c.text
	case RoleTrack:
		r := adwPill(l, b)
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(adwWashTrough)))
		return c.text
	case RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.border))
		return c.text
	case RoleBar:
		ctx.DrawRect(b, paintengine2d.Fill(c.headerbar))
		return c.text
	case RolePanel:
		c.cardFace(l, ctx, b)
		return c.text
	}
	return c.text
}

// CheckIndicator is a check button's 20px box: a 2px inset ring of
// currentColor at 15% (20% hovered; a 25% wash while pressed), or the
// accent fill behind the white check once checked.
func (e adwaitaEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := adwColors(l)
	s := min(box.Dx(), box.Dy())
	if s < 4 {
		return
	}
	b := adwCentered(box, s, s)
	if c.gtk3 {
		c.g3.check(l, ctx, b, st, checked, false)
		return
	}
	c.indicator(l, ctx, b, min(adwR(l, 5, b)*s/max(l.S(20), 1), adwPill(l, b)), st, checked)
	if checked {
		col := c.accentFg
		if st.Disabled() {
			col = c.accentFgDis
		}
		adwTick(ctx, b.Inset(s*0.15), col, s/20*2.5)
	}
}

// RadioIndicator is the round twin: the accent disc with an 8px white dot.
func (e adwaitaEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := adwColors(l)
	s := min(box.Dx(), box.Dy())
	if s < 4 {
		return
	}
	b := adwCentered(box, s, s)
	if c.gtk3 {
		c.g3.check(l, ctx, b, st, selected, true)
		return
	}
	c.indicator(l, ctx, b, adwPill(l, b), st, selected)
	if selected {
		col := c.accentFg
		if st.Disabled() {
			col = c.accentFgDis
		}
		dot := adwCentered(b, s*0.4, s*0.4)
		r := adwPill(l, dot)
		ctx.DrawRoundRect(dot, r, r, paintengine2d.Fill(col))
	}
}

// indicator is a check / radio well of radius r.
func (c *adwSet) indicator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, st ControlState, on bool) {
	dis := st.Disabled()
	if on {
		fill := c.accentBg
		switch {
		case dis:
			fill = c.accentDis
		case st.Pressed():
			fill = c.accentPress
		case st.Hovered():
			fill = c.accentHover
		}
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		return
	}
	if st.Pressed() && !dis {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(0.25)))
		return
	}
	a := float32(adwWashTrough)
	switch {
	case dis:
		a *= adwDisabledOpacity
	case st.Hovered():
		a = adwWashTroughHover
	}
	adwRing(ctx, b, r, max(b.Dx()/10, 1), c.wash(a))
}

func (adwaitaEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if adwColors(l).gtk3 {
		s := min(b.Dx(), b.Dy(), l.S(10))
		FillArrow(ctx, adwCentered(b, s, s), dir, col)
		return
	}
	adwChevron(ctx, b, dir, col, min(b.Dx(), b.Dy(), l.S(16)))
}

func (adwaitaEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	l.Engine().Arrow(l, ctx, b, dir, col)
}

// MenuHighlight is a popover row under the pointer: a 6px-rounded wash of
// currentColor at 10% (the menubar's open title too). GTK 3 fills it with
// the selection blue.
func (adwaitaEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := adwColors(l)
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	if c.gtk3 {
		c.g3.menuHighlight(ctx, b)
		return
	}
	r := adwR(l, 6, b)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(adwWashMenuHover)))
}

func (adwaitaEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := adwColors(l)
	if c.gtk3 {
		return c.g3.menuText(hot)
	}
	return c.popText
}

// Entries, text views and drop-downs ring their focus.
func (adwaitaEngine) FieldFocusRing(*Classic) bool { return true }

// DrawFocusRing is libadwaita's focus ring: 2px of the accent at half
// alpha, just inside the control, following its 6px corners.
func (adwaitaEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	if b.Dx() < 6 || b.Dy() < 6 {
		return
	}
	if c.gtk3 {
		c.g3.focusRing(l, ctx, b, c.g3.focus)
		return
	}
	adwRing(ctx, b, adwR(l, 6, b), max(l.S(2), 1), c.focus)
}

// ---- scrollbars ------------------------------------------------------------------------

// ScrollBarStyle: libadwaita's overlay bars have no arrows; the bar is as
// wide as the hovered 8px slider plus its margins, and idle only a 3px
// indicator shows. GTK 3.14 keeps a 13px gutter with a pill slider.
func (adwaitaEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	if adwColors(l).gtk3 {
		return ScrollBarStyle{Thickness: 13, MinThumb: 42}
	}
	return ScrollBarStyle{Thickness: 14, Overlay: true, MinThumb: 40, EndPad: 3}
}

func (e adwaitaEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.scrollbar(l, ctx, p, vertical, st)
		return
	}
	if p.Thumb.Empty() {
		return
	}
	wide := st.Hovered || st.Pressed != ScrollNone
	a := float32(0.2)
	switch {
	case st.Disabled:
		a = 0.1
	case st.Pressed == ScrollThumbPart:
		a = 0.6
	case st.Hot == ScrollThumbPart:
		a = 0.4
	}
	pad := snap(l.S(3))
	th := snap(l.S(8))
	if !wide {
		th = snap(max(l.S(3), 2))
	}
	var slider, trough paintengine2d.Rect
	if vertical {
		x := p.Bar.Max.X - pad - th
		slider = paintengine2d.XYWH(x, p.Thumb.Min.Y, th, p.Thumb.Dy())
		trough = paintengine2d.XYWH(x, p.Track.Min.Y, th, p.Track.Dy())
	} else {
		y := p.Bar.Max.Y - pad - th
		slider = paintengine2d.XYWH(p.Thumb.Min.X, y, p.Thumb.Dx(), th)
		trough = paintengine2d.XYWH(p.Track.Min.X, y, p.Track.Dx(), th)
	}
	slider, trough = slider.Intersect(p.Bar), trough.Intersect(p.Bar)
	if slider.Empty() {
		return
	}
	r := adwPill(l, slider)
	o := adwPx(l)
	outline := c.outline
	if wide {
		// Hovered, the trough shows as a faint pill under the slider.
		tr := adwPill(l, trough)
		ctx.DrawRoundRect(trough, tr, tr, paintengine2d.Fill(c.wash(adwWashBtn)))
		outline = c.outlineHot
	}
	// The outline keeps the slider readable over any content.
	ol := slider.Inset(-o).Intersect(p.Bar)
	ctx.DrawRoundRect(ol, r+o, r+o, paintengine2d.Fill(outline))
	ctx.DrawRoundRect(slider, r, r, paintengine2d.Fill(c.wash(a)))
}

// DrawScrollBar is a bare slider over its track.
func (adwaitaEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	if thumb.Empty() {
		return
	}
	c := adwColors(l)
	a := float32(0.2)
	switch {
	case st.Pressed():
		a = 0.6
	case st.Hovered():
		a = 0.4
	}
	th := min(thumb.Dx(), thumb.Dy(), snap(l.S(8)))
	var s paintengine2d.Rect
	if thumb.Dy() >= thumb.Dx() {
		s = paintengine2d.XYWH(thumb.Min.X+(thumb.Dx()-th)*0.5, thumb.Min.Y, th, thumb.Dy())
	} else {
		s = paintengine2d.XYWH(thumb.Min.X, thumb.Min.Y+(thumb.Dy()-th)*0.5, thumb.Dx(), th)
	}
	col := c.wash(a)
	if c.gtk3 {
		col = c.g3.sliderColor(st.Hovered(), st.Pressed())
	}
	r := adwPill(l, s)
	ctx.DrawRoundRect(s, r, r, paintengine2d.Fill(col))
}

// ---- frames ------------------------------------------------------------------------------

// GroupBoxInsets: a titled group is a heading above a card (an
// AdwPreferencesGroup); the body sits inside the card's padding.
func (adwaitaEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.S(12)
	top := pad
	if hasTitle {
		top += adwHeadingH(l)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// adwHeadingH is the band a group heading takes above its card.
func adwHeadingH(l *Classic) float32 {
	return snap(l.BoldFont().Height() + l.S(10))
}

func (e adwaitaEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := adwColors(l)
	card := b
	if title != "" {
		hh := adwHeadingH(l)
		l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(2), b.Min.Y, b.Dx()-l.S(4), hh-l.S(6)), c.text, AlignStart, 0)
		card = paintengine2d.XYWH(b.Min.X, b.Min.Y+hh, b.Dx(), b.Dy()-hh)
	}
	c.cardFace(l, ctx, card)
}

// adwCaptionH is an in-app window's header bar.
func adwCaptionH(l *Classic) float32 {
	return snap(max(l.metrics.TitleBar, l.body.Height()+l.S(12)))
}

func (adwaitaEngine) WindowFrameInsets(l *Classic) Insets {
	lw := adwPx(l)
	return Insets{Top: adwCaptionH(l), Right: lw, Bottom: lw, Left: lw}
}

// DrawWindowFrame is a libadwaita dialog: 12px corners, a header bar in the
// dialog's own colour with the title centred in bold and the circular
// close button at the right, a hairline edge (lighter in the dark style).
func (e adwaitaEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.window(e, l, ctx, b, title, st)
		return
	}
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 8*lw || b.Dy() < 8*lw {
		return
	}
	r := adwR(l, 12, b)
	edge := adwOver(c.dialog, adwInk(0.14))
	if c.dark {
		edge = adwOver(c.dialog, paintengine2d.RGBA(1, 1, 1, 0.1))
	}
	adwFrame(ctx, b, r, lw, edge, c.dialog)
	bar := paintengine2d.XYWH(b.Min.X+lw, b.Min.Y+lw, b.Dx()-2*lw, adwCaptionH(l)-lw)
	fg := c.text
	if !st.Active {
		fg = c.dim
	}
	right := bar.Max.X
	if st.CanClose {
		if cb := e.WindowCloseRect(l, b); !cb.Empty() {
			e.closeButton(l, ctx, cb, st, fg)
			right = cb.Min.X - l.S(6)
		}
	}
	if title != "" {
		// Centred on the bar, kept clear of the close button.
		side := bar.Max.X - right
		tb := paintengine2d.XYWH(bar.Min.X+side, bar.Min.Y, bar.Dx()-2*side, bar.Dy())
		l.drawFittedText(ctx, l.BoldFont(), title, tb, fg, AlignCenter, 8)
	}
}

// closeButton is window-close-symbolic on a 24px circular button washed
// like any other.
func (adwaitaEngine) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState, fg paintengine2d.Color) {
	c := adwColors(l)
	a := float32(adwWashBtn)
	switch {
	case st.ClosePress:
		a = adwWashBtnActive
	case st.CloseHot:
		a = adwWashBtnHover
	}
	r := adwPill(l, cb)
	ctx.DrawRoundRect(cb, r, r, paintengine2d.Fill(c.wash(a)))
	g := adwCentered(cb, cb.Dx()/24*8, cb.Dx()/24*8)
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X, g.Min.Y)
	p.LineTo(g.Max.X, g.Max.Y)
	p.MoveTo(g.Max.X, g.Min.Y)
	p.LineTo(g.Min.X, g.Max.Y)
	ctx.DrawPath(p, adwStroke(fg, cb.Dx()/24*2))
}

// WindowCloseRect is the 24px circular close button at the header bar's
// right end (GTK 3's square flat close button sits in the same place).
func (adwaitaEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = adwSnap(b)
	lw := adwPx(l)
	h := adwCaptionH(l) - lw
	side := snap(l.S(24))
	if side > h-l.S(4) {
		side = snap(h - l.S(4))
	}
	if side < 8 || b.Dx() < side+l.S(24) || b.Dy() < h+2*lw {
		return paintengine2d.Rect{}
	}
	x := snap(b.Max.X - lw - l.S(10) - side)
	y := b.Min.Y + lw + snap((h-side)*0.5)
	return paintengine2d.XYWH(x, y, side, side)
}

func (adwaitaEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(adwColors(l).win))
}

// DrawTabPane is a notebook page: the view colour in a hairline frame
// whose top edge is the tab bar's.
func (adwaitaEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 2*lw || b.Dy() < 2*lw {
		return
	}
	edge, fill := c.borderOnView, c.view
	if c.gtk3 {
		edge, fill = c.g3.border, c.g3.view
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+lw, b.Min.Y, b.Dx()-2*lw, b.Dy()-lw), paintengine2d.Fill(fill))
}

// ItemFocus: the focused row wears the focus ring inside its bounds (GTK 3:
// the dashed outline).
func (e adwaitaEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := adwColors(l)
	if b.Dx() < 6 || b.Dy() < 6 {
		return
	}
	if c.gtk3 {
		col := c.g3.focus
		if st.Checked() {
			col = paintengine2d.RGBA(1, 1, 1, 0.5)
		}
		c.g3.focusRing(l, ctx, b.Inset(-l.S(2)), col)
		return
	}
	adwRing(ctx, b, adwR(l, 6, b), max(l.S(2), 1), c.focus)
}

// ViewFrameInsets: lists, trees and tables sit in a 1px frame.
func (adwaitaEngine) ViewFrameInsets(l *Classic) Insets {
	v := float32(math.Ceil(float64(l.metrics.ViewFrame)))
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is a framed view: the view colour in a hairline border.
// GTK rings the focused row, not the view.
func (e adwaitaEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := adwColors(l)
	in := e.ViewFrameInsets(l)
	if in.Zero() || b.Empty() {
		return
	}
	edge, fill := c.borderOnView, c.view
	if c.gtk3 {
		edge, fill = c.g3.border, c.g3.view
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	ctx.DrawRect(in.Apply(b), paintengine2d.Fill(fill))
}

// adwShadow is one CSS box-shadow layer.
type adwShadow struct {
	col              paintengine2d.Color
	dy, blur, spread float32
}

// adwShadowSpecs are the popover, tooltip and dialog box-shadows of scheme
// i. A CSS blur radius fades over twice its length, which is DropShadow's
// blur.
func adwShadowSpecs(l *Classic, i int) [3][]adwShadow {
	var out [3][]adwShadow
	k := l.P("shadow", 1)
	if k <= 0 {
		return out
	}
	black := func(a float32) paintengine2d.Color {
		if i == 1 {
			a *= 2 // the dark style needs the depth to show on charcoal
		}
		return paintengine2d.RGBA(0, 0, 0, min(a*k, 1))
	}
	if i == 2 {
		out[PopupMenu] = []adwShadow{{black(0.35), l.S(2), l.S(6), 0}}
		out[PopupTooltip] = []adwShadow{{black(0.2), l.S(1), l.S(4), 0}}
		out[PopupDialog] = []adwShadow{{black(0.45), l.S(3), l.S(18), l.S(1)}, {black(0.23), 0, 0, l.S(1)}}
		return out
	}
	out[PopupMenu] = []adwShadow{{black(0.09), l.S(1), l.S(10), l.S(1)}, {black(0.05), l.S(2), l.S(28), l.S(3)}}
	out[PopupTooltip] = []adwShadow{{black(0.12), l.S(1), l.S(6), 0}}
	out[PopupDialog] = []adwShadow{{black(0.15), 0, l.S(28), l.S(5)}, {black(0.08), 0, 0, l.S(1)}}
	return out
}

// PopupShadow: popovers float on libadwaita's two-layer soft shadow,
// dialogs on the window's.
func (e adwaitaEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	var out Insets
	if int(kind) >= 3 {
		return out
	}
	for _, s := range adwColors(l).shadows[kind] {
		out = out.Max(ShadowReach(0, s.dy, s.blur, s.spread))
	}
	return out
}

func (e adwaitaEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	c := adwColors(l)
	if int(kind) >= 3 {
		return
	}
	r := l.rx(12)
	switch {
	case c.gtk3:
		r = l.rx(5)
	case kind == PopupTooltip:
		r = l.rx(6)
	}
	for _, s := range c.shadows[kind] {
		DropShadow(ctx, b, r, s.col, 0, s.dy, s.blur, s.spread)
	}
}

// GNOME dialogs put the affirmative button last: "Cancel  OK".
func (adwaitaEngine) StyleHint(l *Classic, h StyleHint) int { return 0 }

// ---- controls ------------------------------------------------------------------------------

// adwButtonFont is the label face: libadwaita buttons are bold.
func adwButtonFont(l *Classic, c *adwSet) *Font {
	if c.gtk3 {
		return l.body
	}
	return l.BoldFont()
}

func (e adwaitaEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := adwColors(l)
	fg := c.button(l, ctx, b, st, false)
	if c.gtk3 {
		c.g3.label(l, ctx, b, label, fg, st)
	} else {
		l.drawFittedText(ctx, adwButtonFont(l, c), label, b, fg, AlignCenter, l.S(16))
	}
	if st.Focused() && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

func (e adwaitaEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := adwColors(l)
	fg := c.button(l, ctx, b, st, true)
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = adwCentered(b, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		f := adwButtonFont(l, c)
		right := max(b.Max.X-pad, x)
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
		ctx.Restore()
	}
	if st.Focused() && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

// adwToggleLayout places a check / radio indicator of side s at the left of
// b, 3px in (the check button's padding), and the label 6px after it.
func adwToggleLayout(l *Classic, b paintengine2d.Rect, s float32) (box, label paintengine2d.Rect) {
	s = min(s, b.Dy(), b.Dx())
	pad := snap(l.S(3))
	if s+2*pad > b.Dy() {
		pad = snap(max((b.Dy()-s)*0.5, 0))
	}
	box = adwSnap(paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+(b.Dy()-s)*0.5, s, s))
	x := box.Max.X + l.S(6)
	return box, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy())
}

// toggleLabel draws a check / radio caption and, focused, rings the whole
// check button (indicator and label, 3px out) as libadwaita does.
func (e adwaitaEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box, lb paintengine2d.Rect, st ControlState, label string) {
	c := adwColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	if label != "" {
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
	if !st.Focused() || st.Disabled() {
		return
	}
	right := box.Max.X + l.S(3)
	if label != "" {
		right = min(lb.Min.X+l.body.Advance(label)+l.S(4), b.Max.X)
	}
	h := min(box.Dy()+l.S(6), b.Dy())
	ring := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-h)*0.5, right-b.Min.X, h).Intersect(b)
	if c.gtk3 {
		c.g3.focusRing(l, ctx, ring.Inset(-l.S(3)).Intersect(b), c.g3.focus)
		return
	}
	adwRing(ctx, ring, adwR(l, 6, ring), max(l.S(2), 1), c.focus)
}

func (e adwaitaEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	box, lb := adwToggleLayout(l, b, l.metrics.Checkbox)
	if box.Dx() < 4 {
		return
	}
	l.Engine().CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, lb, st, label)
}

func (e adwaitaEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box, lb := adwToggleLayout(l, b, side)
	if box.Dx() < 4 {
		return
	}
	l.Engine().RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, lb, st, label)
}

// DrawSwitch is GtkSwitch: a 46×26 pill (currentColor at 15%, the accent
// when on) with a 20px knob 3px in that casts a soft shadow. The focus ring
// sits inside the track: GTK draws it outside, beyond the widget's rect.
func (e adwaitaEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := adwColors(l)
	m := l.metrics
	tw, th := min(m.SwitchW, b.Dx()), min(m.SwitchH, b.Dy())
	track := adwSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8 || track.Dy() < 8 {
		return
	}
	dis := st.Disabled()
	if c.gtk3 {
		c.g3.switchTrack(l, ctx, track, st, on)
	} else {
		r := adwPill(l, track)
		fill := c.wash(adwWashTrough)
		switch {
		case on && dis:
			fill = c.accentDis
		case on && st.Pressed():
			fill = c.accentPress
		case on && st.Hovered():
			fill = c.accentHover
		case on:
			fill = c.accentBg
		case dis:
			fill = c.wash(adwWashTrough * adwDisabledOpacity)
		case st.Pressed():
			fill = c.wash(0.25)
		case st.Hovered():
			fill = c.wash(adwWashTroughHover)
		}
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(fill))
		pad := snap(l.S(3))
		kd := track.Dy() - 2*pad
		kx := track.Min.X + pad
		if on {
			kx = track.Max.X - pad - kd
		}
		knob := paintengine2d.XYWH(kx, track.Min.Y+pad, kd, kd)
		kr := adwPill(l, knob)
		if !dis {
			// box-shadow 0 2px 4px, kept inside the track.
			ctx.Save()
			ctx.ClipRoundRect(track, r, r)
			ctx.DrawRoundRect(knob.Translate(paintengine2d.Pt(0, l.S(1))).Inset(-l.S(0.5)), kr+l.S(0.5), kr+l.S(0.5), paintengine2d.Fill(adwInk(0.2)))
			ctx.Restore()
		}
		kc := c.knob
		if on || st.Hovered() || st.Pressed() {
			kc = c.knobOn
		}
		if dis {
			kc = Mix(c.win, kc, adwDisabledOpacity)
		}
		ctx.DrawRoundRect(knob, kr, kr, paintengine2d.Fill(kc))
		if st.Focused() && !dis {
			adwRing(ctx, track, r, max(l.S(2), 1), c.focus)
		}
	}
	if label != "" {
		fg := c.text
		if dis {
			fg = c.disText
		}
		gap := l.S(12)
		lb := paintengine2d.XYWH(track.Max.X+gap, b.Min.Y, b.Max.X-track.Max.X-gap, b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
}

// DrawSlider is GtkScale: a 4px trough of currentColor at 15%, filled with
// the accent up to a 20px knob ringed in a 10% hairline over a soft shadow.
// Focus rings the knob.
func (e adwaitaEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := adwColors(l)
	t = clamp1(t)
	kd := l.metrics.Thumb
	if kd <= 0 {
		kd = l.S(20)
	}
	ring := l.S(3) // room for the focus ring around the knob
	kd = min(kd, b.Dy()-2*ring, b.Dx()*0.5)
	if kd < 4 {
		return
	}
	dis := st.Disabled()
	x0 := b.Min.X + ring + kd*0.5
	x1 := b.Max.X - ring - kd*0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	kx := x0 + (x1-x0)*t
	if c.gtk3 {
		c.g3.slider(l, ctx, b, st, x0, x1, kx, cy, kd)
		return
	}
	th := snap(max(l.S(4), 2))
	trough := paintengine2d.XYWH(x0-kd*0.5+l.S(2), snap(cy-th*0.5), x1-x0+kd-l.S(4), th)
	tr := adwPill(l, trough)
	a := float32(adwWashTrough)
	switch {
	case dis:
		a *= adwDisabledOpacity
	case st.Hovered() || st.Pressed():
		a = adwWashTroughHover
	}
	ctx.DrawRoundRect(trough, tr, tr, paintengine2d.Fill(c.wash(a)))
	fill := paintengine2d.XYWH(trough.Min.X, trough.Min.Y, kx-trough.Min.X, th)
	if fill.Dx() > 0.5 {
		col := c.accentBg
		switch {
		case dis:
			col = c.accentDis
		case st.Hovered():
			col = c.accentHover
		}
		ctx.DrawRoundRect(fill, tr, tr, paintengine2d.Fill(col))
	}
	knob := paintengine2d.XYWH(kx-kd*0.5, cy-kd*0.5, kd, kd)
	kr := adwPill(l, knob)
	o := adwPx(l)
	if !dis {
		// 0 2px 4px rgb(0 0 6 / 20%), inside the rect.
		ctx.DrawRoundRect(knob.Translate(paintengine2d.Pt(0, l.S(1))).Inset(-l.S(0.5)), kr+l.S(0.5), kr+l.S(0.5), paintengine2d.Fill(adwInk(0.16)))
	}
	// 0 0 0 1px rgb(0 0 6 / 10%): the hairline ring, then the knob.
	ctx.DrawRoundRect(knob.Inset(-o*0.5), kr+o*0.5, kr+o*0.5, paintengine2d.Fill(adwInk(0.1)))
	kc := c.knob
	if st.Hovered() || st.Pressed() {
		kc = c.knobOn
	}
	if dis {
		kc = Mix(c.win, kc, adwDisabledOpacity)
	}
	ctx.DrawRoundRect(knob.Inset(o*0.5), max(kr-o*0.5, 0), max(kr-o*0.5, 0), paintengine2d.Fill(kc))
	if st.Focused() && !dis {
		fr := knob.Inset(-ring + l.S(1)).Intersect(b)
		adwRing(ctx, fr, adwPill(l, fr), max(l.S(2), 1), c.focus)
	}
}

// DrawProgressBar is GtkProgressBar: an 8px pill trough of currentColor at
// 15% filled with the accent; busy, a fifth-length block bounces between
// the ends.
func (e adwaitaEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.progress(l, ctx, b, st, t, indeterminate, phase)
		return
	}
	h := min(b.Dy(), snap(l.S(8)))
	if h < 2 || b.Dx() < 4 {
		return
	}
	trough := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-h)*0.5), b.Dx(), h)
	r := adwPill(l, trough)
	dis := st.Disabled()
	a := float32(adwWashTrough)
	if dis {
		a *= adwDisabledOpacity
	}
	ctx.DrawRoundRect(trough, r, r, paintengine2d.Fill(c.wash(a)))
	col := c.accentBg
	if dis {
		col = c.accentDis
	}
	fill := adwProgressFill(trough, t, indeterminate, phase)
	if fill.Dx() >= 1 {
		fr := min(r, fill.Dx()*0.5)
		ctx.DrawRoundRect(fill, fr, fr, paintengine2d.Fill(col))
	}
}

// adwProgressFill is the filled part of a trough: the fraction t, or the
// pulse block (a fifth of the trough) bouncing with phase.
func adwProgressFill(trough paintengine2d.Rect, t float32, indeterminate bool, phase float32) paintengine2d.Rect {
	if !indeterminate {
		return paintengine2d.XYWH(trough.Min.X, trough.Min.Y, trough.Dx()*clamp1(t), trough.Dy())
	}
	if phase < 0 {
		phase = 0
	}
	phase -= float32(int(phase))
	w := trough.Dx() / 5
	u := phase * 2
	if u > 1 {
		u = 2 - u
	}
	return paintengine2d.XYWH(trough.Min.X+(trough.Dx()-w)*u, trough.Min.Y, w, trough.Dy())
}

func (e adwaitaEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

// DrawComboBox is a GtkDropDown: a button with the label at the left and
// pan-down-symbolic at the right; open, it stays pressed.
func (e adwaitaEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := adwColors(l)
	bs := st &^ StatePrimary
	if open {
		bs |= StatePressed
	}
	fg := c.button(l, ctx, b, bs, false)
	aw := l.S(16)
	pad := l.S(10)
	ab := paintengine2d.XYWH(b.Max.X-pad-aw, b.Min.Y, aw, b.Dy())
	l.Engine().Arrow(l, ctx, ab, DirDown, fg)
	if text != "" {
		l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, ab.Min.X-b.Min.X-pad-l.S(4), b.Dy()), fg, AlignStart, 0)
	}
	if st.Focused() && !open && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

// DrawSpinner is a vertical GtkSpinButton's pair of flat buttons, "+" above
// "−", divided by a hairline of currentColor at 10%.
func (e adwaitaEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := adwColors(l)
	b = adwSnap(b)
	if b.Dx() < 4 || b.Dy() < 6 {
		return
	}
	c.field(l, ctx, b, st&^StateFocused)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	lw := adwPx(l)
	half := func(r paintengine2d.Rect, plus, hover, press bool) {
		s := StateNone
		switch {
		case st.Disabled():
			s = StateDisabled
		case press:
			s = StatePressed
		case hover:
			s = StateHovered
		}
		fg := c.button(l, ctx, r, s, true)
		g := min(r.Dx(), r.Dy(), l.S(16)) * 0.6
		gb := adwCentered(r, g, g)
		cx, cy := (gb.Min.X+gb.Max.X)*0.5, (gb.Min.Y+gb.Max.Y)*0.5
		p := paintengine2d.NewPath()
		p.MoveTo(gb.Min.X, cy)
		p.LineTo(gb.Max.X, cy)
		if plus {
			p.MoveTo(cx, gb.Min.Y)
			p.LineTo(cx, gb.Max.Y)
		}
		ctx.DrawPath(p, adwStroke(fg, max(l.S(1.5), 1)))
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), true, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), false, downHover, downPress)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+l.S(3), mid, b.Dx()-l.S(6), lw), paintengine2d.Fill(c.wash(adwWashBtn)))
}

// DrawTabBar is a notebook header: the window colour over the page-side
// hairline.
func (e adwaitaEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.tabBar(l, ctx, b)
		return
	}
	b = adwSnap(b)
	lw := adwPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.border))
}

// DrawTab is a flat notebook tab: its label, a 7% wash with a 4px underline
// in the border colour under the pointer, the 4px accent underline when
// selected.
func (e adwaitaEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.tab(l, ctx, b, st, label, selected)
		return
	}
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 4*lw || b.Dy() < 6*lw {
		return
	}
	dis := st.Disabled()
	hot := !dis && (st.Hovered() || st.Pressed())
	u := snap(max(l.S(4), 2))
	if hot {
		a := float32(adwWashFlatHover)
		if st.Pressed() {
			a = adwWashFlatActive
		}
		ctx.DrawRect(b, paintengine2d.Fill(c.wash(a)))
		if !selected {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.wash(adwWashTrough)))
		}
	}
	if selected {
		col := c.accentBg
		if dis {
			col = c.accentDis
		}
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(col))
	}
	fg := c.text
	if dis {
		fg = c.disText
	}
	lb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u*0.5)
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(16))
	if st.Focused() && !dis {
		adwRing(ctx, b.Inset(l.S(1)), adwR(l, 6, b), max(l.S(2), 1), c.focus)
	}
}

// DrawPanel: raised is a card; flat is a frame (a 12px hairline border on
// the window colour).
func (e adwaitaEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := adwColors(l)
	if raised {
		c.cardFace(l, ctx, b)
		return
	}
	if c.gtk3 {
		c.g3.frame(l, ctx, b, c.win)
		return
	}
	adwFrame(ctx, adwSnap(b), adwR(l, 12, b), adwPx(l), c.border, c.win)
}

func (e adwaitaEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.menuBar(l, ctx, b)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
}

// DrawMenuTitle is a menubar item: 6px-rounded, washed under the pointer
// and while its menu is open.
func (e adwaitaEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := adwColors(l)
	hot := !st.Disabled() && (open || st.Pressed() || st.Hovered())
	fg := c.text
	if hot {
		if c.gtk3 {
			fg = c.g3.menuTitle(l, ctx, b)
		} else {
			hb := b.Inset(l.S(2))
			a := float32(adwWashFlatHover)
			if open || st.Pressed() {
				a = adwWashMenuHover
			}
			r := adwR(l, 6, hb)
			ctx.DrawRoundRect(hb, r, r, paintengine2d.Fill(c.wash(a)))
		}
	}
	if st.Disabled() {
		fg = c.disText
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

// DrawMenuFrame is a popover: the popover colour in 12px corners with its
// hairline edge; the drop shadow is PopupShadow's.
func (e adwaitaEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.menuFrame(l, ctx, b)
		return
	}
	adwFrame(ctx, adwSnap(b), adwR(l, 12, b), adwPx(l), c.popoverEdge, c.popover)
}

// adwMenuRow is the rounded hover area of a menu row: the row across the
// popover, 6px in from its edges (the menu box's padding).
func adwMenuRow(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	ch := MenuChromeFor(l)
	in := l.S(6)
	return paintengine2d.XYWH(b.Min.X-ch.PadL+in, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*in, b.Dy())
}

// DrawMenuItem is a popover menu row: the rounded 10% wash under the
// pointer (19% pressed), bare check marks and ringed radios in the gutter,
// dim accelerators, the 30% go-next arrow for submenus, hairline
// separators.
func (e adwaitaEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := adwColors(l)
	ch := MenuChromeFor(l)
	lw := adwPx(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := snap(b.Min.X-ch.PadL+lw), snap(b.Max.X+ch.PadR-lw)
		col := adwOver(c.popover, c.wash(adwWashTrough))
		if c.gtk3 {
			col = c.g3.menuSep
		}
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(col))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		switch {
		case c.gtk3:
			l.Engine().MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
		case st.Pressed():
			hb := adwMenuRow(l, b)
			r := adwR(l, 6, hb)
			ctx.DrawRoundRect(hb, r, r, paintengine2d.Fill(c.wash(adwWashMenuActive)))
		default:
			l.Engine().MenuHighlight(l, ctx, adwMenuRow(l, b), false)
		}
	}
	fg := l.Engine().MenuTextColor(l, hot)
	if st.Disabled() {
		fg = adwOver(c.popover, c.wash(adwDisabledOpacity))
	}
	dimFg := Mix(c.popover, fg, adwDimOpacity)
	if c.gtk3 && hot {
		dimFg = fg
	}
	gw := ch.CheckCol()
	s := snap(min(l.S(14), gw-l.S(4)))
	ib := adwSnap(paintengine2d.XYWH(b.Min.X+(gw-s)*0.5, b.Min.Y+(b.Dy()-s)*0.5, s, s))
	switch {
	case row.Radio:
		ctx.DrawCircle(ib.Center(), s*0.5-lw*0.5, paintengine2d.StrokePaint(Mix(c.popover, fg, 0.3), lw))
		if row.Checked {
			ctx.DrawCircle(ib.Center(), s*0.22, paintengine2d.Fill(fg))
		}
	case row.Checked && row.Icon == IconNone:
		adwTick(ctx, ib, fg, s/14*2)
	default:
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := max(ch.SubmenuArrow, l.S(10))
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		ac := Mix(c.popover, fg, 0.3)
		if c.gtk3 {
			ac = fg
		}
		l.Engine().Arrow(l, ctx, ab, DirRight, ac)
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
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), dimFg)
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

func (e adwaitaEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := adwColors(l)
	fg := c.row(ctx, b, st)
	pad := l.S(12)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-2*pad, b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		l.Engine().ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a GtkTreeExpander row: indented 16px per level, the pan
// chevron as the expander, no guide lines.
func (e adwaitaEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := adwColors(l)
	fg := c.row(ctx, b, st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(6) + float32(depth)*indent
	es := l.S(16)
	if !leaf {
		l.Engine().Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, es, b.Dy()), expanded, Mix(c.view, fg, 0.7))
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + es + l.S(6)
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		l.Engine().ItemFocus(l, ctx, b, st)
	}
}

// adwHeaderFontKey memoises the column view title face (bold, 82%).
type adwHeaderFontKey struct{}

func adwHeaderFont(l *Classic) *Font {
	return l.Memo(adwHeaderFontKey{}, func() any {
		return BakeFamily(FamilyUI, WeightBold, l.metrics.FontSize*0.82, l.palette.Text)
	}).(*Font)
}

// DrawTableHeader is a column view header: a bold title at 82% in
// currentColor at 40% (70% hovered, full when sorted), the sort chevron,
// hairline dividers.
func (e adwaitaEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.header(l, ctx, b, st, label, sorted, asc)
		return
	}
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.view))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.borderOnView))
	if b.Dy() > 12*lw {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+l.S(6), lw, b.Dy()-l.S(12)), paintengine2d.Fill(c.borderOnView))
	}
	fg := c.headerText
	switch {
	case st.Disabled():
		fg = adwOver(c.view, c.wash(adwHeaderAlpha*adwDisabledOpacity))
	case st.Pressed() || sorted:
		fg = c.viewText
	case st.Hovered():
		fg = c.headerHot
	}
	aw := float32(0)
	if sorted {
		aw = l.S(16)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		adwChevron(ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy()), dir, fg, l.S(12))
	}
	l.drawFittedText(ctx, adwHeaderFont(l), label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(12)-aw, b.Dy()), fg, AlignStart, 0)
}

func (e adwaitaEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := adwColors(l)
	fg := c.row(ctx, b, st&^StateFocused)
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
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
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawToolBar is a header bar: its own colour over a hairline shade.
func (e adwaitaEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := adwColors(l)
	if c.gtk3 {
		c.g3.headerBar(l, ctx, b)
		return
	}
	b = adwSnap(b)
	lw := adwPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.headerbar))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.headerShade))
}

func (e adwaitaEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := adwColors(l)
	b = adwSnap(b)
	lw := adwPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.win))
	border := c.border
	if c.gtk3 {
		border = c.g3.border
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(border))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(12), b.Min.Y+lw, slot-l.S(18), b.Dy()-lw), c.dim, AlignStart, 0)
	}
}

// DrawTitleBar is a header bar heading: bold title, dim subtitle.
func (e adwaitaEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := adwColors(l)
	e.DrawToolBar(l, ctx, b)
	lw := adwPx(l)
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-lw), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-lw), c.dim, AlignStart, 0)
	}
}

// DrawAccordionHeader is an AdwExpanderRow: title at the left, the arrow at
// the right pointing down, turned up once expanded; rows wash 4% hovered.
func (e adwaitaEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := adwColors(l)
	if !st.Disabled() && (st.Hovered() || st.Pressed()) {
		a := float32(adwWashRowHover)
		if st.Pressed() {
			a = adwWashRowActive
		}
		r := adwR(l, 6, b)
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.wash(a)))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.disText
	}
	aw := l.S(16)
	dir := DirDown
	if expanded {
		dir = DirUp
	}
	l.Engine().Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(12), b.Min.Y, aw, b.Dy()), dir, fg)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(12), b.Min.Y, b.Dx()-aw-l.S(30), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

func (e adwaitaEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := adwColors(l)
	lw := adwPx(l)
	col := c.border
	if c.gtk3 {
		col = c.g3.border
	}
	if vertical {
		x := snap((b.Min.X + b.Max.X - lw) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(col))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - lw) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(col))
}

// DrawSplitter is a GtkPaned separator: a hairline, no handle.
func (e adwaitaEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	e.DrawSeparator(l, ctx, b, vertical)
}

// DrawTooltip is libadwaita's tooltip: 80% near-black in 6px corners with a
// 10% white hairline and white text, in both styles (GTK 3: 5px corners).
func (e adwaitaEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := adwColors(l)
	if b.Empty() {
		return
	}
	r := adwR(l, 6, b)
	if c.gtk3 {
		r = adwR(l, 5, b)
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.tooltip))
	adwRing(ctx, b, r, adwPx(l), c.tooltipEdge)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(10)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tooltipFg, AlignStart, 0)
}

// DrawOverlay dims the window under a dialog.
func (e adwaitaEngine) DrawOverlay(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(l.palette.Overlay))
}

// ---- packs ------------------------------------------------------------------------------------

// adwaitaPalette is the shared palette of a libadwaita scheme: what widgets
// paint themselves (backgrounds, labels, text selection, dividers).
func adwaitaPalette(scheme int) Palette {
	sc := adwSchemes[scheme]
	h := func(k string) paintengine2d.Color {
		if v, ok := sc[k]; ok {
			return Hex(v)
		}
		return Hex(adwLight[k])
	}
	win, view, fg := h("window"), h("view"), h("fg")
	wash := func(a float32) paintengine2d.Color { return adwOver(win, fg.WithAlpha(fg.A*a)) }
	accentBg, accentFg, accent := h("accentBg"), h("accentFg"), h("accent")
	pop := h("popover")
	hover := adwOver(pop, fg.WithAlpha(fg.A*adwWashMenuHover))
	return Palette{
		Background: win, Surface: win, SurfaceAlt: h("headerbar"),
		Overlay: h("dimmer"),
		Border:  wash(adwWashTrough), Divider: wash(adwWashTrough),
		Text: adwOver(win, fg), TextMuted: wash(adwDimOpacity), TextOnAccent: accentFg,
		Accent: accent, AccentHover: adwOver(accentBg, accentFg.WithAlpha(0.1)), AccentPress: adwOver(accentBg, adwInk(0.2)),
		Danger: h("error"), Success: h("success"), Warning: h("warning"),
		Track: wash(adwWashTrough), Thumb: wash(0.4),
		Field: view, FieldBorder: wash(adwWashTrough),
		Focus: accent.WithAlpha(adwFocusAlpha), Selection: accentBg.WithAlpha(adwTextSelAlpha),
		Shadow: h("shade"), Highlight: paintengine2d.RGBA(1, 1, 1, 0.1),
		MenuHover: hover, MenuHoverBorder: hover, MenuGutter: pop,
		BevelLight: win, BevelDark: wash(adwWashTrough),
	}
}

func adwaitaPack(name, label string, year int, summary string, fam ThemeName, scheme int, pal Palette, m ChromeMetrics) ThemePack {
	tok := ThemeTokens{
		Engine:  "adwaita",
		Bevel:   BevelNone,
		Family:  fam,
		Palette: pal,
		Metrics: m,
		Params:  map[string]float32{"scheme": float32(scheme)},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Accent}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.1), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "GNOME", Summary: summary,
		Era: "Adwaita", Palette: fam, Tokens: tok,
	}
}

func adwaitaPacks() []ThemePack {
	return []ThemePack{
		adwaitaPack("adwaita-gtk3", "Adwaita (GTK 3)", 2014,
			"GNOME 3.14's Adwaita: soft gradient buttons in grey borders, blue selections, pill scroll bars.",
			ThemeLight, 2, adw314Palette(), adw314Metrics()),
		adwaitaPack("adwaita", "Adwaita", 2020,
			"libadwaita (GNOME 42+): flat washed buttons, accent checks, pill switches, overlay scroll bars.",
			ThemeLight, 0, adwaitaPalette(0), ChromeMetrics{}),
		adwaitaPack("adwaita-night", "Adwaita Dark", 2020,
			"libadwaita's dark style: the same flat shapes on charcoal, sky-blue accent text.",
			ThemeDark, 1, adwaitaPalette(1), ChromeMetrics{}),
	}
}
