package style

import "github.com/codemodify/paintengine2d"

// The GTK 3 Adwaita of GNOME 3.14 (2014), painted by adwaitaEngine with
// scheme 2: a #ededed window, 3px-rounded buttons in #a1a1a1 borders with a
// white inset highlight over a soft three-stop gradient (#fafafa, #ededed at
// 40%, #e0e0e0), a white edge under buttons and entries, 16px check boxes
// with a dark tick that overshoots the box, a 13px gutter with a pill
// slider, notebook headers with a 3px blue indicator under the current tab,
// menus that highlight in the #4a90d9 selection blue, and the 1px dashed
// focus outline 3px inside a control. Values are the 3.14 stylesheet's.

// adw314 is GTK 3.14's Adwaita palette (the adwScheme keys the shared code
// reads, plus the GTK 3 materials).
var adw314 = adwScheme{
	"window": "#ededed", "view": "#ffffff", "fg": "#2e3436",
	"headerbar": "#ededed", "headerbarShade": "#a1a1a1",
	"card": "#ffffff", "cardShade": "#a1a1a1",
	"dialog": "#ededed", "popover": "#ffffff", "popoverEdge": "#00000021",
	"accentBg": "#4a90d9", "accentFg": "#ffffff", "accent": "#2a76c6",
	"destructive": "#ef2929", "success": "#73d216", "warning": "#f57900", "error": "#cc0000",
	"shade": "#00000026", "outline": "#ffffff", "outlineHot": "#ffffff",
	"knob": "#ffffff", "tooltip": "#000000cc", "tooltipFg": "#ffffff", "dimmer": "#00000059",

	"borders": "#a1a1a1", "selBorders": "#184472", "hilight": "#ffffff",
	"disBg": "#f4f4f4", "disFg": "#8d9091",
	"btn0": "#fafafa", "btn1": "#ededed", "btn2": "#e0e0e0",
	"btnHot0": "#ffffff", "btnHot1": "#f7f7f7", "btnHot2": "#ededed",
	"btnDown0": "#d6d6d6", "btnDown1": "#e0e0e0",
	"sug0": "#5f9ddd", "sug1": "#4a90d9", "sug2": "#3583d5", "sugEdge": "#1c5187",
	"sugHot0": "#85b4e5", "sugHot1": "#5b9add", "sugDown0": "#2b79cb", "sugDown1": "#3583d5",
	"entry0": "#f7f7f7", "entry1": "#ffffff",
	"chk0": "#fbfbfb", "chk1": "#dadada", "chkLow": "#c7c7c7",
	"chkHot0": "#fefefe", "chkHot1": "#f6f6f5", "chkDown0": "#b2b4ae", "chkDown1": "#e7e9e5",
	"trough": "#e0e0e0", "slider": "#b3b5b6", "sliderHot": "#8d9091", "sliderDown": "#4a90d9",
	"tabHeader": "#d6d6d6", "tabText": "#8d9091", "tabTextHot": "#5d6263",
	"switchTrough": "#cecece", "blue0": "#4a90d9", "blue1": "#63a0de",
	"knobEdge": "#999999", "knobEdgeDown": "#153d65", "progTrough": "#d2d2d2",
	"rowHover": "#f2f2f2", "headerLabel": "#96999a", "headerLabelHot": "#626668",
	"menuSep": "#e6e6e6", "hb0": "#f7f7f7", "hb1": "#ededed", "hbLow": "#d9d9d9",
	"focus": "#2e34364d",
}

// adw314Set is the resolved GTK 3.14 material set.
type adw314Set struct {
	win, view, fg, text, disBg, disFg                paintengine2d.Color
	border, selBorder, hilight, sel, selFg           paintengine2d.Color
	btn, btnHot, btnDown, sug, sugHot, sugDown       []paintengine2d.GradientStop
	sugEdge, entry0, entry1, focus                   paintengine2d.Color
	chk, chkHot, chkDown                             []paintengine2d.GradientStop
	chkLow                                           paintengine2d.Color
	trough, slide, slideHot, slideDown               paintengine2d.Color
	tabHeader, tabText, tabTextHot                   paintengine2d.Color
	switchTrough, knobEdge, knobDown, progTrough     paintengine2d.Color
	blue                                             []paintengine2d.GradientStop
	rowHover, headerLabel, headerHot, menuSep, hbLow paintengine2d.Color
	hb                                               []paintengine2d.GradientStop
	textShadow, tipText                              paintengine2d.Color
	entry, tabShade                                  []paintengine2d.GradientStop
	dash                                             []float32 // the focus outline: a line on, a line off
}

func adw314Build(col func(string) paintengine2d.Color) *adw314Set {
	two := func(a, b string) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col(a)), Stop(1, col(b))}
	}
	three := func(a, b, c string) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col(a)), Stop(0.4, col(b)), Stop(1, col(c))}
	}
	g := &adw314Set{
		win: col("window"), view: col("view"), fg: col("fg"), text: col("fg"),
		disBg: col("disBg"), disFg: col("disFg"),
		border: col("borders"), selBorder: col("selBorders"), hilight: col("hilight"),
		sel: col("accentBg"), selFg: col("accentFg"),
		btn: three("btn0", "btn1", "btn2"), btnHot: three("btnHot0", "btnHot1", "btnHot2"), btnDown: two("btnDown0", "btnDown1"),
		sug: three("sug0", "sug1", "sug2"), sugHot: three("sugHot0", "sugHot1", "sug1"), sugDown: two("sugDown0", "sugDown1"),
		sugEdge: col("sugEdge"), entry0: col("entry0"), entry1: col("entry1"), focus: col("focus"),
		chk: two("chk0", "chk1"), chkHot: two("chkHot0", "chkHot1"), chkDown: two("chkDown0", "chkDown1"),
		chkLow: col("chkLow"),
		trough: col("trough"), slide: col("slider"), slideHot: col("sliderHot"), slideDown: col("sliderDown"),
		tabHeader: col("tabHeader"), tabText: col("tabText"), tabTextHot: col("tabTextHot"),
		switchTrough: col("switchTrough"), knobEdge: col("knobEdge"), knobDown: col("knobEdgeDown"), progTrough: col("progTrough"),
		rowHover: col("rowHover"), headerLabel: col("headerLabel"), headerHot: col("headerLabelHot"),
		menuSep: col("menuSep"), hbLow: col("hbLow"),
		textShadow: paintengine2d.RGBA(1, 1, 1, 0.77), tipText: col("tooltipFg"),
	}
	// The blue fill of switches, scales and progress bars: #4a90d9 for
	// its first 2px, easing to #63a0de (the 2px stop sits at 1/6 of a
	// 12px fill, close enough for every height the engine paints).
	g.blue = []paintengine2d.GradientStop{Stop(0, col("blue0")), Stop(0.17, col("blue0")), Stop(1, col("blue1"))}
	g.hb = two("hb0", "hb1")
	g.entry = []paintengine2d.GradientStop{Stop(0, g.entry0), Stop(0.9, g.entry1), Stop(1, g.entry1)}
	g.tabShade = []paintengine2d.GradientStop{Stop(0, paintengine2d.RGBA(0, 0, 0, 0.12)), Stop(1, paintengine2d.RGBA(0, 0, 0, 0))}
	return g
}

// adw314Palette is the shared palette of the 3.14 pack.
func adw314Palette() Palette {
	h := func(k string) paintengine2d.Color { return Hex(adw314[k]) }
	win, border, sel := h("window"), h("borders"), h("accentBg")
	return Palette{
		Background: win, Surface: win, SurfaceAlt: h("hb1"),
		Overlay: h("dimmer"),
		Border:  border, Divider: border,
		Text: h("fg"), TextMuted: h("disFg"), TextOnAccent: h("accentFg"),
		Accent: sel, AccentHover: h("sugHot1"), AccentPress: h("sugDown0"),
		Danger: h("error"), Success: h("success"), Warning: h("warning"),
		Track: h("trough"), Thumb: h("slider"),
		Field: h("view"), FieldBorder: border,
		Focus: h("focus"), Selection: sel,
		Shadow: h("shade"), Highlight: h("hilight"),
		MenuHover: sel, MenuHoverBorder: sel, MenuGutter: h("view"),
		BevelLight: h("hilight"), BevelDark: border,
	}
}

// adw314Metrics: GTK 3.14's smaller controls — 16px indicators, a 13px
// scroll gutter, 3px corners, the wide two-position switch.
func adw314Metrics() ChromeMetrics {
	return ChromeMetrics{
		Radius: 7, RadiusSmall: 3,
		ControlH: 32, FieldH: 32, ComboH: 32,
		Checkbox: 16, Radio: 16,
		MenuItemH: 28, MenuBarH: 30, TabH: 36, RowH: 28,
		TitleBar: 44, HeaderH: 28, ProgressH: 8, SliderH: 28, Thumb: 20,
		Scroll: 13, FieldPad: 8, FocusWidth: 1,
		ToolBarH: 44, SwitchW: 80, SwitchH: 26,
	}
}

// frame is a bordered area (a GtkFrame, the notebook page): 1px border,
// the given fill.
func (g *adw314Set) frame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill paintengine2d.Color) {
	adwFrame(ctx, adwSnap(b), adwR(l, 3, b), adwPx(l), g.border, fill)
}

// button is the 3.14 button: a white edge under it, the #a1a1a1 border,
// the three-stop gradient under a white inset highlight; pressed, a flat
// darker face with an inset shadow; flat buttons show nothing until
// hovered.
func (g *adw314Set) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, flat bool) paintengine2d.Color {
	b = adwSnap(b)
	lw := adwPx(l)
	dis := st.Disabled()
	hot := st.Hovered() && !dis
	down := (st.Pressed() || (st.Toggle() && st.Checked())) && !dis
	fg := g.text
	if dis {
		fg = g.disFg
	}
	if flat && !hot && !down {
		return fg
	}
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return fg
	}
	sug := st.Primary() && !flat
	// The white edge under the button (box-shadow 0 1px white).
	ctx.DrawRoundRect(b, adwR(l, 3, b), adwR(l, 3, b), paintengine2d.Fill(g.hilight.WithAlpha(0.9)))
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-lw)
	r := adwR(l, 3, body)
	edge := g.border
	switch {
	case sug && !dis:
		edge = g.sugEdge
	case dis:
		edge = Mix(g.border, g.disBg, 0.35)
	}
	in := body.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRect(body, r, r, paintengine2d.Fill(edge))
	switch {
	case dis:
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(g.disBg))
		return fg
	case down:
		stops := g.btnDown
		if sug {
			stops = g.sugDown
		}
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, stops...))
		// inset 0 1px rgba(0,0,0,.07) and the 2px shadow under the top.
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.25)))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, in.Dx(), lw), paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.07)))
		ctx.Restore()
	default:
		stops := g.btn
		switch {
		case sug && hot:
			stops = g.sugHot
		case sug:
			stops = g.sug
		case hot:
			stops = g.btnHot
		}
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, stops...))
		hl := g.hilight
		if sug {
			hl = hl.WithAlpha(0.5)
		}
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(hl))
		ctx.Restore()
	}
	if sug {
		return g.selFg
	}
	return fg
}

// label draws a button caption with the stylesheet's 1px text-shadow
// (white under dark text, dark above white text).
func (g *adw314Set) label(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, fg paintengine2d.Color, st ControlState) {
	if text == "" {
		return
	}
	lb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-adwPx(l))
	if !st.Disabled() {
		sh, dy := g.textShadow, adwPx(l)
		if st.Primary() {
			sh, dy = paintengine2d.RGBA(0, 0, 0, 0.54), -adwPx(l)
		}
		l.drawFittedText(ctx, l.body, text, lb.Translate(paintengine2d.Pt(0, dy)), adwOver(g.btn[1].Color, sh), AlignCenter, l.S(16))
	}
	l.drawFittedText(ctx, l.body, text, lb, fg, AlignCenter, l.S(16))
}

// field is the 3.14 entry: a white edge under it, the border (blue when
// focused, with a faint blue inner ring), a white fill easing down from
// #f7f7f7 under three rows of inset shadow.
func (g *adw314Set) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	ctx.DrawRoundRect(b, adwR(l, 3, b), adwR(l, 3, b), paintengine2d.Fill(g.hilight.WithAlpha(0.9)))
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-lw)
	r := adwR(l, 3, body)
	focused := st.Focused() && !st.Disabled()
	edge := g.border
	if focused {
		edge = g.sel
	}
	in := body.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRect(body, r, r, paintengine2d.Fill(edge))
	if st.Disabled() {
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(g.disBg))
		return
	}
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, g.entry...))
	ctx.Save()
	ctx.ClipRoundRect(in, ri, ri)
	for i, a := range [...]float32{0.13, 0.05, 0.02} {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+float32(i)*lw, in.Dx(), lw), paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, a)))
	}
	ctx.Restore()
	if focused {
		adwRing(ctx, in, ri, lw, g.sel.WithAlpha(0.15))
	}
}

// row is a 3.14 list row: #f2f2f2 under the pointer, the selection blue
// with white text (kept when the view loses focus, paled in the backdrop).
func (g *adw314Set) row(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	switch {
	case st.Checked() && st.Backdrop():
		ctx.DrawRect(b, paintengine2d.Fill(Mix(g.view, g.sel, 0.55)))
		return g.selFg
	case st.Checked():
		ctx.DrawRect(b, paintengine2d.Fill(g.sel))
		return g.selFg
	case st.Hovered() || st.Pressed():
		ctx.DrawRect(b, paintengine2d.Fill(g.rowHover))
	}
	return g.text
}

// check is a 3.14 check box or radio: 15px (a 16px indicator with its white
// edge), #a1a1a1 border, a white top row over a #fbfbfb → #dadada fill; the
// dark tick overshoots the box's top right, the radio dot is 5px.
func (g *adw314Set) check(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on, radio bool) {
	b = adwSnap(b)
	lw := adwPx(l)
	s := b.Dx()
	if s < 6*lw {
		return
	}
	dis := st.Disabled()
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y, s-lw, s-lw)
	r := adwR(l, 3, box)
	if radio {
		r = adwPill(l, box)
	}
	// The white edge (icon-shadow 0 1px white).
	ctx.DrawRoundRect(box.Translate(paintengine2d.Pt(0, lw)), r, r, paintengine2d.Fill(g.hilight))
	edge := g.border
	if dis {
		edge = Mix(g.border, g.disBg, 0.35)
	}
	ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(edge))
	in := box.Inset(lw)
	ri := max(r-lw, 0)
	switch {
	case dis:
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(g.disBg))
	case st.Pressed():
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, g.chkDown...))
	default:
		stops := g.chk
		if st.Hovered() {
			stops = g.chkHot
		}
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, stops...))
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(g.hilight))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-lw, in.Dx(), lw), paintengine2d.Fill(g.chkLow))
		ctx.Restore()
	}
	if !on {
		return
	}
	mark := g.text
	if dis {
		mark = g.disFg
	}
	if radio {
		d := snap(max(s*5/16, 2*lw))
		dot := adwCentered(box, d, d)
		ctx.DrawRoundRect(dot, adwPill(l, dot), adwPill(l, dot), paintengine2d.Fill(mark))
		return
	}
	// The tick runs from the lower left past the box's top-right corner.
	p := paintengine2d.NewPath()
	p.MoveTo(box.Min.X+s*0.22, box.Min.Y+s*0.50)
	p.LineTo(box.Min.X+s*0.42, box.Min.Y+s*0.72)
	p.LineTo(box.Max.X-s*0.02, box.Min.Y+s*0.08)
	ctx.DrawPath(p, paintengine2d.Paint{Color: mark, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: max(s*0.13, lw), Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

func (g *adw314Set) menuHighlight(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(g.sel))
}

func (g *adw314Set) menuText(hot bool) paintengine2d.Color {
	if hot {
		return g.selFg
	}
	return g.text
}

// focusRing is the 3.14 outline: 1px dashed, 3px inside b, 2px corners.
func (g *adw314Set) focusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	w := adwPx(l)
	in := adwSnap(b).Inset(snap(l.S(3)))
	if in.Dx() < 4*w || in.Dy() < 4*w {
		return
	}
	r := adwR(l, 2, in)
	ctx.DrawRoundRect(in.Inset(w*0.5), r, r, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4, Dash: g.dash}})
}

func (g *adw314Set) sliderColor(hot, down bool) paintengine2d.Color {
	switch {
	case down:
		return g.slideDown
	case hot:
		return g.slideHot
	}
	return g.slide
}

// scrollbar is the 3.14 gutter: #e0e0e0 trough, a pill slider 3px in from
// each side (#b3b5b6, darker hovered, blue while dragged), no steppers.
func (g *adw314Set) scrollbar(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	if p.Bar.Empty() {
		return
	}
	ctx.DrawRect(p.Bar, paintengine2d.Fill(g.trough))
	if p.Thumb.Empty() {
		return
	}
	in := snap(l.S(3))
	s := p.Thumb
	if vertical {
		s = paintengine2d.XYWH(s.Min.X+in, s.Min.Y+in*0.5, s.Dx()-2*in, s.Dy()-in)
	} else {
		s = paintengine2d.XYWH(s.Min.X+in*0.5, s.Min.Y+in, s.Dx()-in, s.Dy()-2*in)
	}
	if s.Empty() {
		return
	}
	col := g.sliderColor(st.Hot == ScrollThumbPart && !st.Disabled, st.Pressed == ScrollThumbPart)
	if st.Disabled {
		col = Mix(g.trough, g.slide, 0.5)
	}
	r := adwPill(l, s)
	ctx.DrawRoundRect(s, r, r, paintengine2d.Fill(col))
}

// window is a 3.14 client-side-decorated dialog: 7px top corners, the
// header bar's gradient and hairlines, a bold title and the close button
// (a flat button showing its face under the pointer).
func (g *adw314Set) window(e adwaitaEngine, l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 8*lw || b.Dy() < 8*lw {
		return
	}
	r := adwR(l, 7, b)
	ctx.DrawPath(RoundRectPath(b, r, r, 0, 0), paintengine2d.Fill(g.border))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawPath(RoundRectPath(in, ri, ri, 0, 0), paintengine2d.Fill(g.win))
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), adwCaptionH(l)-2*lw)
	if st.Active {
		ctx.DrawPath(RoundRectPath(bar, ri, ri, 0, 0), VGradient(bar, g.hb...))
		ctx.Save()
		ctx.ClipPath(RoundRectPath(bar, ri, ri, 0, 0))
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), lw), paintengine2d.Fill(g.hilight))
		ctx.Restore()
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-lw, bar.Dx(), lw), paintengine2d.Fill(g.hbLow))
	}
	ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y, bar.Dx(), lw), paintengine2d.Fill(g.border))
	fg := g.text
	if !st.Active {
		fg = g.disFg
	}
	right := bar.Max.X
	if st.CanClose {
		if cb := e.WindowCloseRect(l, b); !cb.Empty() {
			bs := StateNone
			switch {
			case st.ClosePress:
				bs = StatePressed | StateHovered
			case st.CloseHot:
				bs = StateHovered
			}
			cfg := g.button(l, ctx, cb, bs, true)
			if !st.Active {
				cfg = fg
			}
			gb := adwCentered(cb, cb.Dx()*0.34, cb.Dx()*0.34)
			p := paintengine2d.NewPath()
			p.MoveTo(gb.Min.X, gb.Min.Y)
			p.LineTo(gb.Max.X, gb.Max.Y)
			p.MoveTo(gb.Max.X, gb.Min.Y)
			p.LineTo(gb.Min.X, gb.Max.Y)
			ctx.DrawPath(p, paintengine2d.Paint{Color: cfg, Style: paintengine2d.StyleStroke,
				Stroke: paintengine2d.Stroke{Width: max(l.S(2), 1), Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
			right = cb.Min.X - l.S(6)
		}
	}
	if title != "" {
		side := bar.Max.X - right
		tb := paintengine2d.XYWH(bar.Min.X+side, bar.Min.Y, bar.Dx()-2*side, bar.Dy())
		if st.Active {
			l.drawFittedText(ctx, l.BoldFont(), title, tb.Translate(paintengine2d.Pt(0, lw)), adwOver(g.win, g.textShadow), AlignCenter, 8)
		}
		l.drawFittedText(ctx, l.BoldFont(), title, tb, fg, AlignCenter, 8)
	}
}

// switchTrack is the 3.14 switch: a 3px-rounded #cecece trough in the
// border (blue with a dark blue edge when on, reading ON / OFF) and a
// button-faced slider covering half of it.
func (g *adw314Set) switchTrack(l *Classic, ctx *paintengine2d.Context, track paintengine2d.Rect, st ControlState, on bool) {
	lw := adwPx(l)
	r := adwR(l, 3, track)
	dis := st.Disabled()
	in := track.Inset(lw)
	ri := max(r-lw, 0)
	switch {
	case on && !dis:
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(g.selBorder))
		ctx.DrawRoundRect(in, ri, ri, VGradient(in, g.blue...))
	case dis:
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(Mix(g.border, g.disBg, 0.35)))
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(g.disBg))
	default:
		ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(g.border))
		ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(g.switchTrough))
	}
	half := snap(track.Dx() * 0.5)
	slot := paintengine2d.XYWH(track.Min.X, track.Min.Y, half, track.Dy())
	text, tcol := "ON", g.selFg
	if on {
		slot = paintengine2d.XYWH(track.Max.X-half, track.Min.Y, half, track.Dy())
	} else {
		text, tcol = "OFF", g.disFg
	}
	if dis {
		tcol = g.disFg
	}
	other := paintengine2d.XYWH(track.Min.X, track.Min.Y, track.Dx()-half, track.Dy())
	if !on {
		other = paintengine2d.XYWH(track.Min.X+half, track.Min.Y, track.Dx()-half, track.Dy())
	}
	l.drawFittedText(ctx, adwHeaderFont(l), text, other, tcol, AlignCenter, 0)
	bs := StateNone
	switch {
	case dis:
		bs = StateDisabled
	case st.Pressed():
		bs = StateHovered
	case st.Hovered():
		bs = StateHovered
	}
	g.button(l, ctx, slot, bs, false)
	if st.Focused() && !dis {
		g.focusRing(l, ctx, track, g.focus)
	}
}

// slider is the 3.14 scale: a 4px bordered trough (#cecece), the blue fill
// in its dark blue border, a 20px round knob with the button gradient.
func (g *adw314Set) slider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, x0, x1, kx, cy, kd float32) {
	lw := adwPx(l)
	dis := st.Disabled()
	th := snap(max(l.S(5), 3*lw))
	trough := adwSnap(paintengine2d.XYWH(x0-kd*0.25, cy-th*0.5, x1-x0+kd*0.5, th))
	r := adwR(l, 3, trough)
	adwFrame(ctx, trough, r, lw, g.border, g.switchTrough)
	if fw := kx - trough.Min.X; fw > 2*lw && !dis {
		fill := paintengine2d.XYWH(trough.Min.X, trough.Min.Y, fw, trough.Dy())
		ctx.DrawRoundRect(fill, r, r, paintengine2d.Fill(g.selBorder))
		fin := fill.Inset(lw)
		if !fin.Empty() {
			ctx.DrawRoundRect(fin, max(r-lw, 0), max(r-lw, 0), VGradient(fin, g.blue...))
		}
	}
	knob := adwSnap(paintengine2d.XYWH(kx-kd*0.5, cy-kd*0.5, kd, kd))
	kr := adwPill(l, knob)
	edge := g.knobEdge
	if st.Pressed() && !dis {
		edge = g.knobDown
	}
	ctx.DrawRoundRect(knob, kr, kr, paintengine2d.Fill(edge))
	stops := g.btn
	if st.Hovered() && !dis {
		stops = g.btnHot
	}
	kin := knob.Inset(lw)
	if dis {
		ctx.DrawRoundRect(kin, max(kr-lw, 0), max(kr-lw, 0), paintengine2d.Fill(g.disBg))
	} else {
		ctx.DrawRoundRect(kin, max(kr-lw, 0), max(kr-lw, 0), VGradient(kin, stops...))
	}
	if st.Focused() && !dis {
		g.focusRing(l, ctx, knob.Inset(-l.S(3)).Intersect(b), g.focus)
	}
}

// progress is the 3.14 progress bar: a 3px-rounded #d2d2d2 trough in the
// border with the blue fill in its dark blue edge; busy, a block bounces.
func (g *adw314Set) progress(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	lw := adwPx(l)
	h := min(b.Dy(), snap(l.S(8)))
	if h < 3*lw || b.Dx() < 4*lw {
		return
	}
	trough := adwSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-h)*0.5, b.Dx(), h))
	r := adwR(l, 3, trough)
	fillCol := g.progTrough
	if st.Disabled() {
		fillCol = g.disBg
	}
	adwFrame(ctx, trough, r, lw, g.border, fillCol)
	fill := adwProgressFill(trough, t, indeterminate, phase)
	if fill.Dx() < 2*lw || st.Disabled() {
		return
	}
	ctx.DrawRoundRect(fill, r, r, paintengine2d.Fill(g.selBorder))
	if in := fill.Inset(lw); !in.Empty() {
		ctx.DrawRoundRect(in, max(r-lw, 0), max(r-lw, 0), VGradient(in, g.blue...))
	}
}

// tabBar is a 3.14 notebook header: #d6d6d6 with a soft shadow along its
// top and the border line on the page side.
func (g *adw314Set) tabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = adwSnap(b)
	lw := adwPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(g.tabHeader))
	sh := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), min(snap(l.S(3)), b.Dy()))
	ctx.DrawRect(sh, VGradient(sh, g.tabShade...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(g.border))
}

// tab is a 3.14 notebook tab: no face, a bold label (grey, darker hovered,
// black when current) over a 3px indicator on the page side — the border
// grey hovered, the selection blue on the current tab.
func (g *adw314Set) tab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Dx() < 4*lw || b.Dy() < 6*lw {
		return
	}
	dis := st.Disabled()
	hot := !dis && (st.Hovered() || st.Pressed())
	u := snap(max(l.S(3), 2))
	ind := paintengine2d.XYWH(b.Min.X+l.S(4), b.Max.Y-lw-u, b.Dx()-l.S(8), u)
	fg := g.tabText
	switch {
	case dis:
		fg = Mix(g.tabHeader, g.disFg, 0.8)
	case selected:
		fg = g.text
		ctx.DrawRect(ind, paintengine2d.Fill(g.sel))
	case hot:
		fg = g.tabTextHot
		ctx.DrawRect(ind, paintengine2d.Fill(g.border))
	}
	l.drawFittedText(ctx, l.BoldFont(), label, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u), fg, AlignCenter, l.S(16))
	if st.Focused() && !dis {
		g.focusRing(l, ctx, b, g.focus)
	}
}

// menuBar is the 3.14 menubar: the window colour over a 10% black line.
func (g *adw314Set) menuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = adwSnap(b)
	lw := adwPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(g.win))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(Mix(g.win, Hex("#000000"), 0.1)))
}

// menuTitle marks a hot menubar item as 3.14 did: a 3px blue underline
// and blue text.
func (g *adw314Set) menuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) paintengine2d.Color {
	u := snap(max(l.S(3), 2))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(g.sel))
	return Hex("#2a76c6")
}

// menuFrame is a 3.14 menu: white, framed by the popup's 13% black line.
func (g *adw314Set) menuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	adwFrame(ctx, adwSnap(b), 0, adwPx(l), Mix(g.view, Hex("#000000"), 0.23), g.view)
}

// header is a 3.14 tree-view column header: white, a bold grey title
// (darker hovered, black when sorted), dividers in the window colour.
func (g *adw314Set) header(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(g.view))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(g.win))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(g.win))
	fg := g.headerLabel
	switch {
	case st.Disabled():
		fg = Mix(g.view, g.disFg, 0.7)
	case st.Pressed() || sorted:
		fg = g.text
	case st.Hovered():
		fg = g.headerHot
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		s := l.S(8)
		FillArrow(ctx, adwCentered(paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy()), s, s), dir, fg)
	}
	l.drawFittedText(ctx, l.BoldFont(), label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(12)-aw, b.Dy()), fg, AlignStart, 0)
}

// headerBar is the 3.14 header bar and toolbar band: #f7f7f7 → #ededed
// under a white top line, a #d9d9d9 inner line and the border at the
// bottom.
func (g *adw314Set) headerBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = adwSnap(b)
	lw := adwPx(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, VGradient(b, g.hb...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(g.hilight))
	if b.Dy() > 3*lw {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*lw, b.Dx(), lw), paintengine2d.Fill(g.hbLow))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(g.border))
}
