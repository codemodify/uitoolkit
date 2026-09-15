package style

import (
	"strconv"

	"github.com/codemodify/paintengine2d"
)

// motifEngine paints OSF/Motif 1.2 chrome — the look of CDE, IRIX, HP VUE
// and every Unix workstation of the 1990s: square controls under 2px
// shadows, pushed buttons that invert their shadows and fill with the arm
// colour, a sunken default-button ring, a solid highlight rectangle for
// keyboard focus, sunken check squares and radio diamonds, scroll bars with
// bevelled triangle arrows, menus that arm items by raising them, etched
// XmFrame group boxes and the mwm window decoration.
//
// Colour model. A look is a handful of colour sets, each ONE background.
// The foreground, top shadow, bottom shadow and select (arm / trough)
// colours are derived from it with Motif's own XmGetColors algorithm
// (lib/Xm/Color.c, see motifDerive), so any colour a pack gives gets the
// shadows Motif — and the CDE colour server — would have computed:
//
//	window    buttons, toggles, scroll bars, frames (CDE colour set 5)
//	menu      menu bars and menus (CDE colour set 6)
//	text      text fields, text areas, lists (CDE colour set 4)
//	active    active window frame, CDE highlight (colour set 1)
//	inactive  inactive window frame (colour set 2)
//
// Pack data ("extra" colours; the colour-set backgrounds default to
// Palette.Background, .Field and .Accent). Set backgrounds may be given at
// the 16-bit precision of CDE palette files, so the derivation is exact:
//
//	window, menu, text, active, inactive   colour-set backgrounds (above)
//	windowFg, menuFg, textFg, activeFg, inactiveFg
//	              fix a set's foreground (default: black or white by the
//	              background's brightness — CDE's "Dynamic" foreground —
//	              flipped where body text would read below 3:1)
//	ts, bs, sel   override the derived window top shadow, bottom shadow and
//	              select colour (IRIX shading)
//	highlight     keyboard-focus highlight (XmNhighlightColor; default the
//	              window foreground, CDE uses the active colour)
//	radio         fill of a set radio diamond (default the select colour;
//	              CDE's XmNenableToggleColor fills with the highlight)
//	check         fill of a set check box (default the select colour)
//	checkMark     colour of the check glyph (default the foreground; IRIX
//	              draws it red)
//	radioPip      a pip in the centre of a set radio diamond (IRIX, blue)
//	trough        scroll bar / scale trough (XmNtroughColor; default the
//	              select colour)
//	thumb         scroll bar slider and arrow-box face (default window)
//	info, infoText  tooltip fill and text (pale yellow, black)
//
// Params:
//
//	shadow        shadow and highlight thickness in design pixels (2; 1 is
//	              CDE's XmNenableThinThickness)
//	checkGlyph    1 = CDE check mark in set check boxes (Motif 1.2 fills)
//	optionMenu    1 = drop-downs are XmOptionMenu buttons (raised, bar
//	              glyph); 0 = XmComboBox (sunken field + arrow button)
//	reverseSelect 1 = list, tree and table selection in reversed ground
//	              colours (XmList's default); 0 = the select colour
//	graded        1 = IRIX graded bevels (two shades per side)
//	outline       1 = IRIX black outline around menus and window frames
//	arrowBox      1 = scroll bar arrows in raised boxes with flat glyphs
//	grip          1 = grip lines on scroll thumbs and scale sliders
//	titleLeft     1 = window titles start at the left (IRIX 4Dwm); mwm
//	              centres them
type motifEngine struct{ BaseEngine }

func init() {
	RegisterEngine(motifEngine{})
	for _, p := range motifPacks() {
		RegisterPack(p)
	}
}

func (motifEngine) ID() string { return "motif" }

// DefaultMetrics are Motif's proportions at the toolkit's 16px UI font
// (Motif used 12–14px Helvetica): tall push buttons that carry a 2px
// highlight band and a 2px shadow, 16px scroll bars, XmScale troughs.
func (motifEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Square: true, BevelDepth: 2,
		ControlH: 32, FieldH: 30, ComboH: 30,
		Checkbox: 14, Radio: 17,
		MenuItemH: 26, MenuBarH: 30, TabH: 30, RowH: 22,
		TitleBar: 26, HeaderH: 26, ProgressH: 20, SliderH: 24, Thumb: 30,
		Scroll: 16, Pad: 10, FieldPad: 6, FocusWidth: 2, Border: 2,
		ToolBarH: 40, StatusBarH: 28, SpinnerW: 18, SwitchW: 46, SwitchH: 20,
	}
}

// ---- colour model ---------------------------------------------------------------

// motifDerive is XmGetColors (lib/Xm/Color.c, CalculateColorsRGB): the
// foreground, top shadow, bottom shadow and select colours Motif derives
// from a background. Brightness — 75% intensity plus 25% luminosity —
// picks one of three models: below 20% everything lightens towards white
// (DARK), above 93% everything darkens (LITE), in between the top shadow
// lightens and the bottom shadow darkens by factors that slide with the
// brightness (STD). The foreground is black above 70% brightness, else
// white. The arithmetic is Motif's own, in 16-bit channels; results keep
// the high byte, as a 24-bit visual does.
func motifDerive(bg paintengine2d.Color) (fg, ts, bs, sel paintengine2d.Color) {
	const max = 65535
	const pct = max / 100
	r, g, b := motif16(bg.R), motif16(bg.G), motif16(bg.B)
	intensity := (r + g + b) / 3
	lum := int(0.30*float64(r) + 0.59*float64(g) + 0.11*float64(b))
	br := (intensity*75 + lum*25) / 100
	out := func(r, g, b int) paintengine2d.Color {
		return paintengine2d.RGB(float32(r>>8)/255, float32(g>>8)/255, float32(b>>8)/255)
	}
	darken := func(f int) paintengine2d.Color {
		return out(r-r*f/100, g-g*f/100, b-b*f/100)
	}
	lighten := func(f int) paintengine2d.Color {
		return out(r+f*(max-r)/100, g+f*(max-g)/100, b+f*(max-b)/100)
	}
	switch {
	case br < 20*pct: // DARK
		sel, bs, ts = lighten(15), lighten(30), lighten(50)
	case br > 93*pct: // LITE
		sel, bs, ts = darken(15), darken(40), darken(20)
	default: // STD: factors interpolate between the LO and HI tables
		sel = darken(15 + br*(15-15)/max)
		bs = darken(60 + br*(40-60)/max)
		ts = lighten(50 + br*(60-50)/max)
	}
	fg = paintengine2d.RGB(1, 1, 1)
	if br > 70*pct {
		fg = paintengine2d.RGB(0, 0, 0)
	}
	return
}

// motif16 is a colour channel as an X 16-bit value.
func motif16(v float32) int {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 65535
	}
	return int(v*65535 + 0.5)
}

// motifShow is the colour a 24-bit visual displays for c (the high byte of
// each 16-bit channel).
func motifShow(c paintengine2d.Color) paintengine2d.Color {
	return paintengine2d.RGB(float32(motif16(c.R)>>8)/255, float32(motif16(c.G)>>8)/255, float32(motif16(c.B)>>8)/255)
}

// motifSpec parses an X colour spec: #rgb, #rrggbb, or the 16-bit
// #rrrrggggbbbb of CDE palette files, kept at full precision.
func motifSpec(s string) paintengine2d.Color {
	if len(s) == 13 && s[0] == '#' {
		ch := func(i int) float32 {
			n, err := strconv.ParseUint(s[1+i*4:5+i*4], 16, 16)
			if err != nil {
				return 0
			}
			return float32(n) / 65535
		}
		return paintengine2d.RGB(ch(0), ch(1), ch(2))
	}
	return hexColor(s)
}

// mset is one Motif colour set: a background and what Motif derives from
// it, plus the four graded shades IRIX bevels use.
type mset struct {
	bg, fg, ts, bs, sel, dim paintengine2d.Color
	hi1, hi2, lo2, lo1       paintengine2d.Color // graded: outer / inner light, inner / outer dark
}

// motifSet derives the colour set of bg. For sets that carry body text
// (legible) the black-or-white foreground Motif picks by brightness is
// flipped when it would read below 3:1 — as if the pack had tuned
// XmNforegroundThreshold; window frame titles keep Motif's choice (CDE's
// white on orange).
func motifSet(bg paintengine2d.Color, legible bool) mset {
	fg, ts, bs, sel := motifDerive(bg)
	q := motifShow(bg)
	if legible && ContrastRatio(fg, q) < 3 {
		fg = paintengine2d.RGB(1-fg.R, 1-fg.G, 1-fg.B)
	}
	return mset{
		bg: q, fg: fg, ts: ts, bs: bs, sel: sel, dim: Mix(fg, q, 0.5),
		hi1: Shade(q, 0.7), hi2: Shade(q, 0.5), lo2: Shade(q, -0.25), lo1: Shade(q, -0.5),
	}
}

// motifLook is the resolved colour set of a look.
type motifLook struct {
	win, menu, text, act, inact mset
	trough, thumb, hl           paintengine2d.Color
	radio, check                paintengine2d.Color // unset: the set's select colour
	mark, pip                   paintengine2d.Color // check glyph (unset: foreground), radio pip
	info, infoFg                paintengine2d.Color
	selBg, selFg                paintengine2d.Color // list, tree and table selection
	shadow                      float32             // design pixels
	glyph, option, reverse      bool
	graded, outline, arrowBox   bool
	grip, titleLeft             bool
}

type motifKey struct{}

// motifColors is the look's resolved colour set (built once per look).
func motifColors(l *Classic) *motifLook {
	return l.Memo(motifKey{}, func() any { return motifBuild(l) }).(*motifLook)
}

func motifBuild(l *Classic) *motifLook {
	p := l.palette
	field := p.Field
	if colorUnset(field) {
		field = p.Background
	}
	c := &motifLook{
		win:   motifSet(l.X("window", p.Background), true),
		text:  motifSet(l.X("text", field), true),
		act:   motifSet(l.X("active", p.Accent), false),
		inact: motifSet(l.X("inactive", p.Background), false),
	}
	c.menu = motifSet(l.X("menu", l.X("window", p.Background)), true)
	// Pack colours are painted as a 24-bit visual shows them.
	x := func(k string, def paintengine2d.Color) paintengine2d.Color {
		if v, ok := l.tokens.Extra[k]; ok {
			return motifShow(v)
		}
		return def
	}
	for _, fs := range [...]struct {
		key string
		s   *mset
	}{{"windowFg", &c.win}, {"menuFg", &c.menu}, {"textFg", &c.text}, {"activeFg", &c.act}, {"inactiveFg", &c.inact}} {
		if fg, ok := l.tokens.Extra[fs.key]; ok {
			fs.s.fg = motifShow(fg)
			fs.s.dim = Mix(fs.s.fg, fs.s.bg, 0.5)
		}
	}
	c.win.ts = x("ts", c.win.ts)
	c.win.bs = x("bs", c.win.bs)
	c.win.sel = x("sel", c.win.sel)
	c.trough = x("trough", c.win.sel)
	c.thumb = x("thumb", c.win.bg)
	c.hl = x("highlight", c.win.fg)
	c.check = x("check", paintengine2d.Color{})
	c.radio = x("radio", paintengine2d.Color{})
	c.mark = x("checkMark", paintengine2d.Color{})
	c.pip = x("radioPip", paintengine2d.Color{})
	c.info = x("info", Hex("#ffffe0"))
	c.infoFg = x("infoText", ReadableOn(c.info, 4.5, paintengine2d.RGB(0, 0, 0)))
	c.shadow = l.P("shadow", 2)
	if c.shadow < 1 {
		c.shadow = 1
	}
	if c.shadow > 4 {
		c.shadow = 4
	}
	on := func(k string) bool { return l.P(k, 0) != 0 }
	c.glyph, c.option, c.reverse = on("checkGlyph"), on("optionMenu"), on("reverseSelect")
	c.graded, c.outline, c.arrowBox, c.grip = on("graded"), on("outline"), on("arrowBox"), on("grip")
	c.titleLeft = on("titleLeft")
	if c.reverse {
		c.selBg, c.selFg = c.text.fg, c.text.bg
	} else {
		c.selBg, c.selFg = c.text.sel, c.text.fg
	}
	return c
}

// checkFill and radioFill are the fills of a set toggle drawn in set s:
// the pack's colour, else the set's select colour (XmNselectColor).
func (c *motifLook) checkFill(s *mset) paintengine2d.Color {
	if colorUnset(c.check) {
		return s.sel
	}
	return c.check
}

func (c *motifLook) radioFill(s *mset) paintengine2d.Color {
	if colorUnset(c.radio) {
		return s.sel
	}
	return c.radio
}

// markFor is the colour of a check glyph drawn on set s in state st.
func (c *motifLook) markFor(s *mset, st ControlState) paintengine2d.Color {
	if colorUnset(c.mark) {
		return s.fgFor(st)
	}
	if st.Disabled() {
		return Mix(c.mark, s.bg, 0.5)
	}
	return c.mark
}

// diamond paints a set or unset radio diamond of set s into box.
func (c *motifLook) diamond(ctx *paintengine2d.Context, box paintengine2d.Rect, s *mset, on, disabled bool, u float32) {
	if !on {
		mDiamond(ctx, box, s.ts, s.bs, s.bg, u)
		return
	}
	fill := c.radioFill(s)
	if disabled {
		fill = Mix(fill, s.bg, 0.5)
	}
	mDiamond(ctx, box, s.bs, s.ts, fill, u)
	if !colorUnset(c.pip) {
		pip := c.pip
		if disabled {
			pip = Mix(pip, fill, 0.5)
		}
		d := box.Dx()
		if box.Dy() < d {
			d = box.Dy()
		}
		d = snap(d * 0.38)
		mDiamond(ctx, paintengine2d.XYWH(box.Min.X+(box.Dx()-d)*0.5, box.Min.Y+(box.Dy()-d)*0.5, d, d), pip, pip, pip, u)
	}
}

// px is the shadow (and highlight) thickness in whole device pixels.
func (c *motifLook) px(l *Classic) float32 { return mPx(l, c.shadow) }

// mPx converts design pixels to whole device pixels (at least one).
func mPx(l *Classic, v float32) float32 {
	p := snap(l.S(v))
	if p < 1 {
		p = 1
	}
	return p
}

// mU is one design pixel in device pixels, for glyphs laid out on the
// Motif pixel grid (arrows, diamonds).
func mU(l *Classic) float32 {
	u := l.S(1)
	if u < 1 {
		u = 1
	}
	return u
}

// fgFor is the label colour of set s in state st (stippled, here dimmed,
// when insensitive).
func (s *mset) fgFor(st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return s.dim
	}
	return s.fg
}

// ---- painting primitives ------------------------------------------------------------

// mSnap rounds a rect to device pixels.
func mSnap(r paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(r.Min.X), snap(r.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(r.Max.X)-x0, snap(r.Max.Y)-y0)
}

// mShadow paints a Motif shadow t device pixels thick just inside r: top
// along the top and left, bot along the bottom and right, split on the
// diagonal at the top-right and bottom-left corners (DrawSimpleShadow,
// lib/Xm/Draw.c). The top shadow owns the diagonal pixels; with cor (the
// etched variant) the bottom shadow does. Swap the colours for XmSHADOW_IN.
func mShadow(ctx *paintengine2d.Context, r paintengine2d.Rect, top, bot paintengine2d.Color, t float32, cor bool) {
	r = mSnap(r)
	w, h := r.Dx(), r.Dy()
	if m := float32(int(w / 2)); t > m {
		t = m
	}
	if m := float32(int(h / 2)); t > m {
		t = m
	}
	if t < 1 {
		return
	}
	tf, bf := paintengine2d.Fill(top), paintengine2d.Fill(bot)
	x0, y0, x1, y1 := r.Min.X, r.Min.Y, r.Max.X, r.Max.Y
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, w, t), tf)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0+t, t, h-t), tf)
	k := float32(1)
	if cor {
		k = 0
	}
	for i := float32(0); i < t; i++ {
		ctx.DrawRect(paintengine2d.XYWH(x0+i+k, y1-1-i, w-i-k, 1), bf)
		ctx.DrawRect(paintengine2d.XYWH(x1-1-i, y0+i+k, 1, h-i-k), bf)
	}
}

// mRings paints concentric one-pixel rings inside r, outer ring first:
// hi1/lo1 for the outer half of t, hi2/lo2 for the inner half — the
// graded bevel of IRIX's IRIS IM widgets.
func mRings(ctx *paintengine2d.Context, r paintengine2d.Rect, hi1, lo1, hi2, lo2 paintengine2d.Color, t float32) {
	r = mSnap(r)
	half := float32(int(t/2 + 0.5))
	for i := float32(0); i < t; i++ {
		ri := r.Inset(i)
		if ri.Dx() < 2 || ri.Dy() < 2 {
			return
		}
		hi, lo := hi1, lo1
		if i >= half {
			hi, lo = hi2, lo2
		}
		edge(ctx, ri, hi, lo)
	}
}

// raise paints a raised (XmSHADOW_OUT) shadow of set s inside r.
func (c *motifLook) raise(ctx *paintengine2d.Context, r paintengine2d.Rect, s *mset, t float32) {
	if c.graded {
		mRings(ctx, r, s.hi1, s.lo1, s.hi2, s.lo2, t)
		return
	}
	mShadow(ctx, r, s.ts, s.bs, t, false)
}

// sink paints a sunken (XmSHADOW_IN) shadow of set s inside r.
func (c *motifLook) sink(ctx *paintengine2d.Context, r paintengine2d.Rect, s *mset, t float32) {
	if c.graded {
		mRings(ctx, r, s.lo2, s.hi1, s.lo1, s.hi2, t)
		return
	}
	mShadow(ctx, r, s.bs, s.ts, t, false)
}

// etch paints an XmSHADOW_ETCHED_IN frame (out: ETCHED_OUT) of thickness t
// inside r: half of it sunken, half raised.
func (c *motifLook) etch(ctx *paintengine2d.Context, r paintengine2d.Rect, s *mset, t float32, out bool) {
	half := float32(int(t / 2))
	if half < 1 {
		half = 1
	}
	dark, light := s.bs, s.ts
	if out {
		dark, light = light, dark
	}
	mShadow(ctx, r, dark, light, half, true)
	mShadow(ctx, r.Inset(half), light, dark, half, true)
}

// mEtchLine is an XmSHADOW_ETCHED_IN separator of thickness t: dark then
// light, centred on the line at (x, y) running length along its axis.
func mEtchLine(ctx *paintengine2d.Context, x, y, length float32, vertical bool, dark, light paintengine2d.Color, t float32) {
	if length <= 0 {
		return
	}
	half := float32(int(t / 2))
	if half < 1 {
		half = 1
	}
	if vertical {
		x = snap(x) - half
		ctx.DrawRect(paintengine2d.XYWH(x, snap(y), half, snap(length)), paintengine2d.Fill(dark))
		ctx.DrawRect(paintengine2d.XYWH(x+half, snap(y), half, snap(length)), paintengine2d.Fill(light))
		return
	}
	y = snap(y) - half
	ctx.DrawRect(paintengine2d.XYWH(snap(x), y, snap(length), half), paintengine2d.Fill(dark))
	ctx.DrawRect(paintengine2d.XYWH(snap(x), y+half, snap(length), half), paintengine2d.Fill(light))
}

// mHighlight is the Motif highlight rectangle (XmeDrawHighlight): a solid
// band t pixels wide along the inside of r.
func mHighlight(ctx *paintengine2d.Context, r paintengine2d.Rect, col paintengine2d.Color, t float32) {
	r = mSnap(r)
	if r.Dx() < 2*t+1 || r.Dy() < 2*t+1 {
		return
	}
	f := paintengine2d.Fill(col)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), t), f)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Max.Y-t, r.Dx(), t), f)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y+t, t, r.Dy()-2*t), f)
	ctx.DrawRect(paintengine2d.XYWH(r.Max.X-t, r.Min.Y+t, t, r.Dy()-2*t), f)
}

// mGrid lays glyphs out on the Motif pixel grid: design pixel (x, y) of a
// glyph whose origin is o maps to whole device pixels, u per design pixel.
type mGrid struct{ ox, oy, u float32 }

func (g mGrid) fill(ctx *paintengine2d.Context, x, y, w, h int, col paintengine2d.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, x1 := g.ox+snap(float32(x)*g.u), g.ox+snap(float32(x+w)*g.u)
	y0, y1 := g.oy+snap(float32(y)*g.u), g.oy+snap(float32(y+h)*g.u)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, x1-x0, y1-y0), paintengine2d.Fill(col))
}

// mArrow paints XmeDrawArrow (lib/Xm/DrArrow.c) into r: a triangle
// pointing dir, as tall as it is wide, bevelled with a two-pixel shadow —
// top on the edges facing up and left, bot on the others — and filled with
// fill (skipped when unset). The rectangles are Motif's own, laid out in
// design pixels of u device pixels each.
func mArrow(ctx *paintengine2d.Context, r paintengine2d.Rect, dir Direction, top, bot, fill paintengine2d.Color, u float32) {
	W, H := int(r.Dx()/u+0.01), int(r.Dy()/u+0.01)
	g := mGrid{snap(r.Min.X + (r.Dx()-float32(W)*u)*0.5), snap(r.Min.Y + (r.Dy()-float32(H)*u)*0.5), u}
	var size, xo, yo int
	if W > H {
		size, xo = H-2, (W-H)/2
	} else {
		size, yo = W-2, (H-W)/2
	}
	if size < 1 {
		return
	}
	horiz := dir == DirLeft || dir == DirRight
	if horiz {
		xo, yo = yo, xo
	}
	upLeft := dir == DirUp || dir == DirLeft
	put := func(kind, x, y, w, h int) {
		if !upLeft && kind < 2 {
			kind = 1 - kind // down / right arrows swap the shadows ...
		}
		if horiz {
			x, y, w, h = y, x, h, w
		}
		if !upLeft {
			x, y = W-x-w, H-y-h // ... and mirror
		}
		col := top
		switch kind {
		case 1:
			col = bot
		case 2:
			if colorUnset(fill) {
				return
			}
			col = fill
		}
		g.fill(ctx, x, y, w, h, col)
	}
	ww, yy, start := size, size-1+yo, 1+xo
	for ww > 0 {
		switch {
		case ww == 1:
			put(0, start, yy+1, 1, 1)
		case ww == 2:
			if size == 2 || upLeft {
				put(0, start, yy, 2, 1)
				put(0, start, yy+1, 1, 1)
				put(1, start+1, yy+1, 1, 1)
			}
		case start == 1+xo: // the base
			if upLeft {
				put(0, start, yy, 2, 1)
				put(0, start, yy+1, 1, 1)
				put(1, start+1, yy+1, 1, 1)
			} else {
				put(0, start, yy, 2, 1)
				put(1, start, yy+1, 2, 1)
			}
			put(1, start+2, yy, ww-2, 2)
		default:
			put(0, start, yy, 2, 2)
			if ww == 3 {
				put(1, start+2, yy, 1, 2)
			} else {
				put(1, start+ww-2, yy, 2, 2)
			}
			if ww > 4 {
				put(2, start+2, yy, ww-4, 2)
			}
		}
		start++
		ww -= 2
		yy -= 2
	}
}

// mDiamond paints XmeDrawDiamond (lib/Xm/DrTog.c) centred in r: the
// XmONE_OF_MANY indicator, a diamond with three-pixel edges — top on the
// upper two, bot on the lower two, the right vertex bot — filled with fill.
func mDiamond(ctx *paintengine2d.Context, r paintengine2d.Rect, top, bot, fill paintengine2d.Color, u float32) {
	s := r.Dx()
	if r.Dy() < s {
		s = r.Dy()
	}
	W := int(s/u + 0.01)
	if W%2 == 0 {
		W--
	}
	if W < 3 {
		return
	}
	g := mGrid{snap(r.Min.X + (r.Dx()-float32(W)*u)*0.5), snap(r.Min.Y + (r.Dy()-float32(W)*u)*0.5), u}
	m := (W - 1) / 2
	for y := 0; y < W; y++ {
		k := y
		if y > m {
			k = W - 1 - y
		}
		a, b := m-k, m+k
		lc, rc := top, top
		if y == m {
			rc = bot
		} else if y > m {
			lc, rc = bot, bot
		}
		if k <= 2 {
			if y == m {
				g.fill(ctx, a, y, m-a+1, 1, lc)
				g.fill(ctx, m+1, y, b-m, 1, rc)
			} else {
				g.fill(ctx, a, y, b-a+1, 1, lc)
			}
			continue
		}
		g.fill(ctx, a, y, 3, 1, lc)
		if !colorUnset(fill) {
			g.fill(ctx, a+3, y, b-a-5, 1, fill)
		}
		g.fill(ctx, b-2, y, 3, 1, rc)
	}
}

// motifCheck is the XmINDICATOR_CHECK glyph of lib/Xm/DrTog.c on its
// 32×32 template.
var motifCheck = [...][2]float32{{0, 15}, {6, 9}, {14, 17}, {31, 0}, {31, 3}, {21, 17}, {16, 31}}

// mCheck fills the Motif / CDE check mark into box less margin on each side.
func mCheck(ctx *paintengine2d.Context, box paintengine2d.Rect, margin, u float32, col paintengine2d.Color) {
	sx := (box.Dx() - 2*margin - u) / 32
	sy := (box.Dy() - 2*margin - u) / 32
	if sx <= 0 || sy <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	for i, pt := range motifCheck {
		x, y := box.Min.X+margin+pt[0]*sx+u*0.5, box.Min.Y+margin+pt[1]*sy+u*0.5
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// mFlatArrow fills a flat triangle pointing dir into b (IRIS IM arrow
// boxes, XmArrowButton glyphs at small sizes).
func mFlatArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	h := s * 0.5
	FillArrow(ctx, paintengine2d.XYWH(cx-h, cy-h, s, s), dir, col)
}

// ---- parts -------------------------------------------------------------------------

func (e motifEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	pressed := st.Pressed() && !st.Disabled()
	switch role {
	case RoleButton, RoleCombo, RoleThumb, RoleTab:
		fill := c.win.bg
		if role == RoleThumb {
			fill = c.thumb
		}
		if pressed {
			fill = c.win.sel
		}
		ctx.DrawRect(r, paintengine2d.Fill(fill))
		if pressed {
			c.sink(ctx, r, &c.win, t)
		} else {
			c.raise(ctx, r, &c.win, t)
		}
		return c.win.fgFor(st)
	case RoleTool:
		on := st.Toggle() && st.Checked()
		if pressed || on {
			ctx.DrawRect(r, paintengine2d.Fill(c.win.sel))
			c.sink(ctx, r, &c.win, t)
		} else {
			ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
			c.raise(ctx, r, &c.win, t)
		}
		return c.win.fgFor(st)
	case RoleField:
		ctx.DrawRect(r, paintengine2d.Fill(c.text.bg))
		c.sink(ctx, r, &c.win, t)
		return c.text.fgFor(st)
	case RoleCheck:
		if st.Checked() {
			ctx.DrawRect(r, paintengine2d.Fill(c.checkFill(&c.win)))
			c.sink(ctx, r, &c.win, t)
		} else {
			ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
			c.raise(ctx, r, &c.win, t)
		}
		return c.win.fgFor(st)
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(r, paintengine2d.Fill(c.selBg))
			return c.selFg
		}
		return c.text.fgFor(st)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, r, false)
		}
		return c.menu.fgFor(st)
	case RoleTrack:
		ctx.DrawRect(r, paintengine2d.Fill(c.trough))
		c.sink(ctx, r, &c.win, t)
		return c.win.fgFor(st)
	case RoleBar:
		ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
		c.raise(ctx, r, &c.win, t)
		return c.win.fgFor(st)
	}
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	return c.win.fgFor(st)
}

// CheckIndicator is the XmN_OF_MANY square: raised in the background when
// off, sunken and filled with the select colour when on (with the CDE check
// mark when the pack asks). A held toggle shows the state it will take.
func (e motifEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := motifColors(l)
	t := c.px(l)
	s := box.Dx()
	if box.Dy() < s {
		s = box.Dy()
	}
	s = snap(s)
	r := paintengine2d.XYWH(snap(box.Min.X+(box.Dx()-s)*0.5), snap(box.Min.Y+(box.Dy()-s)*0.5), s, s)
	on := checked != (st.Pressed() && !st.Disabled())
	if !on {
		ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
		c.raise(ctx, r, &c.win, t)
		return
	}
	fill := c.checkFill(&c.win)
	if st.Disabled() {
		fill = Mix(fill, c.win.bg, 0.5)
	}
	ctx.DrawRect(r, paintengine2d.Fill(fill))
	c.sink(ctx, r, &c.win, t)
	if c.glyph {
		mCheck(ctx, r, t, mU(l), c.markFor(&c.win, st))
	}
}

// RadioIndicator is the XmONE_OF_MANY diamond: raised when off, sunken and
// filled with the radio (select) colour when on.
func (e motifEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := motifColors(l)
	on := selected != (st.Pressed() && !st.Disabled())
	c.diamond(ctx, box, &c.win, on, st.Disabled(), mU(l))
}

// Arrow is the Motif arrow: a bevelled triangle filled with col (the
// XmArrowButton glyph).
func (motifEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	c := motifColors(l)
	mArrow(ctx, b, dir, c.win.ts, c.win.bs, col, mU(l))
}

// Expander is the XmContainer outline button: a small bevelled arrow,
// pointing right when collapsed and down when expanded.
func (e motifEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := motifColors(l)
	e.expander(l, ctx, b, expanded, &c.win)
}

func (motifEngine) expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, s *mset) {
	u := mU(l)
	side := snap(l.S(11))
	if m := snap(b.Dy() - 2*u); side > m {
		side = m
	}
	box := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-side)*0.5), snap(b.Min.Y+(b.Dy()-side)*0.5), side, side)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	mArrow(ctx, box, dir, s.ts, s.bs, s.bg, u)
}

// MenuHighlight arms a menu item or title the Motif way: a raised shadow
// box, never a fill.
func (motifEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := motifColors(l)
	c.raise(ctx, b, &c.menu, c.px(l))
}

func (motifEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return motifColors(l).menu.fg
}

// Fields show the highlight rectangle when they own the keyboard focus.
func (motifEngine) FieldFocusRing(l *Classic) bool { return true }

// ---- scroll bars ------------------------------------------------------------------------

// ScrollBarStyle is XmScrollBar: a 16px gutter, one arrow at each end; the
// arrow cells are square inside the sunken frame.
func (motifEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	c := motifColors(l)
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsEnds, ArrowLen: 16 - c.shadow, MinThumb: 10}
}

// DrawScrollBarParts is the XmScrollBar: a sunken trough in the trough
// colour, bevelled triangle arrows at both ends (sunken while held) and a
// raised slider. With nothing to scroll the slider fills the trough.
func (e motifEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := motifColors(l)
	t := c.px(l)
	u := mU(l)
	bar := mSnap(p.Bar)
	if bar.Dx() < 2*t+2 || bar.Dy() < 2*t+2 {
		return
	}
	ctx.DrawRect(bar, paintengine2d.Fill(c.trough))
	c.sink(ctx, bar, &c.win, t)
	in := bar.Inset(t)
	cross := func(r paintengine2d.Rect) paintengine2d.Rect {
		r = mSnap(r)
		if vertical {
			return paintengine2d.XYWH(in.Min.X, r.Min.Y, in.Dx(), r.Dy()).Intersect(in)
		}
		return paintengine2d.XYWH(r.Min.X, in.Min.Y, r.Dx(), in.Dy()).Intersect(in)
	}
	arrow := func(cell paintengine2d.Rect, dir Direction, part ScrollPart) {
		if cell.Empty() {
			return
		}
		a := cross(cell)
		if a.Empty() {
			return
		}
		held := st.Pressed == part && !st.Disabled
		if c.arrowBox {
			// IRIS IM: the arrow sits in a raised box of its own.
			ctx.DrawRect(a, paintengine2d.Fill(c.thumb))
			if held {
				c.sink(ctx, a, &c.win, t)
			} else {
				c.raise(ctx, a, &c.win, t)
			}
			g := a.Inset(t + snap(a.Dx()*0.08))
			col := c.win.lo1
			if st.Disabled {
				col = Mix(col, c.thumb, 0.5)
			}
			mFlatArrow(ctx, g, dir, col)
			return
		}
		top, bot := c.win.ts, c.win.bs
		if held {
			top, bot = bot, top
		}
		mArrow(ctx, a, dir, top, bot, c.thumb, u)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	th := p.Thumb
	if th.Empty() {
		th = p.Track // sliderSize == maximum: the slider fills the trough
	}
	if th.Empty() {
		return
	}
	s := cross(th)
	if s.Dx() < 2*t+1 || s.Dy() < 2*t+1 {
		return
	}
	face := c.thumb
	if st.Pressed == ScrollThumbPart && !st.Disabled {
		face = Mix(c.thumb, c.win.sel, 0.7) // dragged: the arm colour
	}
	ctx.DrawRect(s, paintengine2d.Fill(face))
	c.raise(ctx, s, &c.win, t)
	if c.grip {
		e.grip(l, ctx, s.Inset(t), vertical, c.win.fg)
	}
	if st.Disabled {
		// The insensitive stipple, as an even wash.
		ctx.DrawRect(in, paintengine2d.Fill(c.trough.WithAlpha(0.45)))
	}
}

// grip paints three grip lines across the middle of r (IRIS IM thumbs).
func (motifEngine) grip(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, vertical bool, col paintengine2d.Color) {
	u := mU(l)
	gap := snap(l.S(3))
	if vertical {
		if r.Dy() < gap*4 {
			return
		}
		y := snap((r.Min.Y+r.Max.Y)*0.5) - gap
		for i := float32(0); i < 3; i++ {
			ctx.DrawRect(paintengine2d.XYWH(r.Min.X+u, y+i*gap, r.Dx()-2*u, u), paintengine2d.Fill(col))
		}
		return
	}
	if r.Dx() < gap*4 {
		return
	}
	x := snap((r.Min.X+r.Max.X)*0.5) - gap
	for i := float32(0); i < 3; i++ {
		ctx.DrawRect(paintengine2d.XYWH(x+i*gap, r.Min.Y+u, u, r.Dy()-2*u), paintengine2d.Fill(col))
	}
}

// DrawScrollBar paints a bare track + thumb (widgets that lay out their own).
func (e motifEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered(), Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -------------------------------------------------------------------------------

func (motifEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.5
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is an XmFrame: an etched-in 2px frame whose title child
// interrupts the top line, centred on it. A raised box is XmSHADOW_OUT.
func (motifEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	f := l.body
	top := r.Min.Y
	if title != "" {
		top = snap(r.Min.Y + f.Height()*0.5 - t*0.5)
	}
	frame := paintengine2d.XYWH(r.Min.X, top, r.Dx(), r.Max.Y-top)
	if raised {
		c.raise(ctx, frame, &c.win, t)
	} else {
		c.etch(ctx, frame, &c.win, t, false)
	}
	if title == "" {
		return
	}
	tx := r.Min.X + l.S(10)
	tw := f.Advance(title) + l.S(6)
	if max := r.Dx() - l.S(20); tw > max {
		tw = max
	}
	if tw <= 0 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(snap(tx), r.Min.Y, snap(tw), f.Height()), paintengine2d.Fill(c.win.bg))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), r.Min.Y, tw-l.S(3), f.Height()), c.win.fg, AlignStart, 0)
}

// mwm is the geometry of the mwm decoration of a frame of bounds b.
type mwm struct {
	r                          paintengine2d.Rect
	border, bar                float32 // resize border width, title bar height
	menu, title, min, max, cli paintengine2d.Rect
}

// motifMwm lays out the mwm frame: a resize border of 6 design pixels, a
// title bar one line tall with the window menu button on the left and the
// minimize and maximize buttons on the right.
func motifMwm(l *Classic, b paintengine2d.Rect) mwm {
	r := mSnap(b)
	g := mwm{r: r, border: snap(l.S(6)), bar: snap(l.body.Height())}
	if g.bar < snap(l.S(18)) {
		g.bar = snap(l.S(18))
	}
	B, T := g.border, g.bar
	x0, x1, y0 := r.Min.X+B, r.Max.X-B, r.Min.Y+B
	g.menu = paintengine2d.XYWH(x0, y0, T, T)
	g.max = paintengine2d.XYWH(x1-T, y0, T, T)
	g.min = paintengine2d.XYWH(x1-2*T, y0, T, T)
	if g.min.Min.X-g.menu.Max.X < T {
		// Too narrow for the minimize and maximize buttons.
		g.min, g.max = paintengine2d.Rect{}, paintengine2d.Rect{}
		g.title = paintengine2d.XYWH(g.menu.Max.X, y0, x1-g.menu.Max.X, T)
	} else {
		g.title = paintengine2d.XYWH(g.menu.Max.X, y0, g.min.Min.X-g.menu.Max.X, T)
	}
	g.cli = paintengine2d.XYWH(x0, y0+T, x1-x0, r.Max.Y-B-(y0+T))
	return g
}

func (motifEngine) WindowFrameInsets(l *Classic) Insets {
	g := motifMwm(l, paintengine2d.XYWH(0, 0, 400, 300))
	return Insets{Top: g.border + g.bar, Right: g.border, Bottom: g.border, Left: g.border}
}

// WindowCloseRect is the window menu button at the left of the title bar:
// in mwm that is how a window is closed (its menu, or a double click).
// No drop shadows: Motif and CDE menus and tooltips float flat.
func (motifEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (motifEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

func (motifEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return motifMwm(l, b).menu
}

// DrawWindowFrame is the mwm decoration: a raised border with the resize
// handles cut at the corners, a sunken inner edge, and a title bar of
// separately raised parts — window menu button (a bar), the centred title,
// minimize (a small square) and maximize (a large square) — all in the
// active or inactive colour set.
func (e motifEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := motifColors(l)
	g := motifMwm(l, b)
	r := g.r
	if r.Dx() < 4*g.border || r.Dy() < 2*g.border+g.bar {
		return
	}
	s := &c.inact
	if st.Active {
		s = &c.act
	}
	ext := c.px(l)  // FRAME_EXTERNAL_SHADOW_WIDTH
	ib := mPx(l, 1) // FRAME_INTERNAL_SHADOW_WIDTH
	B, T := g.border, g.bar
	ctx.DrawRect(r, paintengine2d.Fill(s.bg))
	if c.graded {
		mRings(ctx, r, s.hi1, s.lo1, s.hi2, s.lo2, ext)
	} else {
		mShadow(ctx, r, s.ts, s.bs, ext, false)
	}
	// Resize handle cuts: each handle is bevelled on its own, so a cut
	// shows the dark edge of one handle beside the light edge of the next.
	cw := B + T
	lo, hi := paintengine2d.Fill(s.bs), paintengine2d.Fill(s.ts)
	if r.Dx() > 3*cw {
		for _, x := range [2]float32{r.Min.X + cw, r.Max.X - cw} {
			ctx.DrawRect(paintengine2d.XYWH(x-ib, r.Min.Y, ib, B), lo)
			ctx.DrawRect(paintengine2d.XYWH(x, r.Min.Y, ib, B), hi)
			ctx.DrawRect(paintengine2d.XYWH(x-ib, r.Max.Y-B, ib, B), lo)
			ctx.DrawRect(paintengine2d.XYWH(x, r.Max.Y-B, ib, B), hi)
		}
	}
	if r.Dy() > 3*cw {
		for _, y := range [2]float32{r.Min.Y + cw, r.Max.Y - cw} {
			ctx.DrawRect(paintengine2d.XYWH(r.Min.X, y-ib, B, ib), lo)
			ctx.DrawRect(paintengine2d.XYWH(r.Min.X, y, B, ib), hi)
			ctx.DrawRect(paintengine2d.XYWH(r.Max.X-B, y-ib, B, ib), lo)
			ctx.DrawRect(paintengine2d.XYWH(r.Max.X-B, y, B, ib), hi)
		}
	}
	// The sunken inner edge around title bar and client area.
	mShadow(ctx, r.Inset(B-ib), s.bs, s.ts, ib, false)
	// The client area in the window background.
	if !g.cli.Empty() {
		ctx.DrawRect(g.cli, paintengine2d.Fill(c.win.bg))
	}
	part := func(p paintengine2d.Rect, sunk bool) {
		if p.Empty() {
			return
		}
		if sunk {
			mShadow(ctx, p, s.bs, s.ts, ib, false)
		} else {
			mShadow(ctx, p, s.ts, s.bs, ib, false)
		}
	}
	glyph := func(p paintengine2d.Rect, w, h float32) {
		if p.Empty() {
			return
		}
		w, h = snap(w), snap(h)
		gr := paintengine2d.XYWH(snap(p.Min.X+(p.Dx()-w)*0.5), snap(p.Min.Y+(p.Dy()-h)*0.5), w, h)
		mShadow(ctx, gr, s.ts, s.bs, ib, false)
	}
	press := st.ClosePress && st.CanClose
	part(g.menu, press)
	glyph(g.menu, T*0.5, maxf(T*0.2, 3*ib))
	part(g.title, false)
	part(g.min, false)
	glyph(g.min, maxf(T*0.18, 3*ib), maxf(T*0.18, 3*ib))
	part(g.max, false)
	glyph(g.max, T*0.52, T*0.52)
	if c.outline {
		ctx.DrawRect(r.Inset(0.5), paintengine2d.StrokePaint(paintengine2d.RGB(0, 0, 0), 1))
	}
	if title != "" && !g.title.Empty() {
		tb := g.title.Inset(ib + l.S(4))
		tb.Min.Y, tb.Max.Y = g.title.Min.Y, g.title.Max.Y
		align := AlignCenter
		if c.titleLeft {
			align = AlignStart
		}
		l.drawFittedText(ctx, l.body, title, tb, s.fg, align, 0)
	}
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// DrawTabPane is the XmNotebook page: raised, joined to the selected tab.
func (motifEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	c.raise(ctx, r, &c.win, c.px(l))
}

// Motif dialogs put the default action first: OK, Cancel, Help.
func (motifEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// ---- push buttons ------------------------------------------------------------------------

// DrawButton is an XmPushButton: a highlight band (the focus rectangle,
// drawn solid when focused and thin while the pointer is over the button,
// XmNhighlightOnEnter), then the 2px shadow. Armed, the shadow inverts and
// the face fills with the arm colour; the label stays put. The default
// button carries the sunken default-button shadow ring
// (XmNdefaultButtonShadowThickness) between highlight and button.
func (e motifEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	face := r.Inset(t)
	if st.Primary() {
		d := float32(int(t / 2))
		if d < 1 {
			d = 1
		}
		mShadow(ctx, face, c.win.bs, c.win.ts, d, false)
		face = face.Inset(d + mPx(l, 1))
	}
	if face.Dx() < 2*t+2 || face.Dy() < 2*t+2 {
		face = r
	}
	pressed := st.Pressed() && !st.Disabled()
	if pressed {
		ctx.DrawRect(face, paintengine2d.Fill(c.win.sel))
		c.sink(ctx, face, &c.win, t)
	} else {
		c.raise(ctx, face, &c.win, t)
	}
	if label != "" {
		lb := paintengine2d.XYWH(face.Min.X+t, r.Min.Y, face.Dx()-2*t, r.Dy())
		l.drawFittedText(ctx, l.body, label, lb, c.win.fgFor(st), AlignCenter, l.S(6))
	}
	e.highlight(l, ctx, r, st)
}

// highlight paints the highlight band of a primitive: solid when it has
// the keyboard focus, one pixel while the pointer is over it.
func (motifEngine) highlight(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
	if st.Disabled() {
		return
	}
	c := motifColors(l)
	switch {
	case st.Focused():
		mHighlight(ctx, r, c.hl, c.px(l))
	case st.Hovered():
		mHighlight(ctx, r, c.hover(), mPx(l, 1))
	}
}

// hover is the thin highlight drawn while the pointer is over a primitive
// (XmNhighlightOnEnter): the highlight colour, halfway to the background.
func (c *motifLook) hover() paintengine2d.Color { return Mix(c.hl, c.win.bg, 0.5) }

// DrawToolButton is a tool-bar push button: raised, sunken and filled with
// the arm colour while held or latched, with the same highlight band.
func (e motifEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	face := r.Inset(t)
	on := st.Toggle() && st.Checked()
	if (st.Pressed() && !st.Disabled()) || on {
		ctx.DrawRect(face, paintengine2d.Fill(c.win.sel))
		c.sink(ctx, face, &c.win, t)
	} else {
		c.raise(ctx, face, &c.win, t)
	}
	fg := c.win.fgFor(st)
	pad, iconSide, iconGap := ToolButtonChromeFor(l, r.Dy())
	x := r.Min.X + pad
	if icon != IconNone {
		if max := face.Dy() - 2*t; iconSide > max {
			iconSide = max
		}
		ib := paintengine2d.XYWH(x, r.Min.Y+(r.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(r.Min.X+(r.Dx()-iconSide)*0.5, r.Min.Y+(r.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		right := r.Max.X - pad
		if icon == IconNone {
			x = r.Min.X + t
			right = r.Max.X - t
			l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, r.Min.Y, right-x, r.Dy()), fg, AlignCenter, l.S(4))
		} else if right > x {
			l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, r.Min.Y, right-x, r.Dy()), fg, AlignStart, 0)
		}
	}
	e.highlight(l, ctx, r, st)
}

// ---- toggles ------------------------------------------------------------------------------

// toggleLayout places a toggle's indicator (side px) and label inside b:
// [highlight][margin][indicator][spacing][label].
func (motifEngine) toggleLayout(l *Classic, b paintengine2d.Rect, side float32) (box, lb paintengine2d.Rect) {
	t := motifColors(l).px(l)
	side = snap(side)
	x := snap(b.Min.X + t + l.S(1))
	box = paintengine2d.XYWH(x, snap(b.Min.Y+(b.Dy()-side)*0.5), side, side)
	lx := box.Max.X + l.S(6)
	lb = paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-t-lx, b.Dy())
	return box, lb
}

// toggleFocus is the highlight rectangle around a whole toggle.
func (motifEngine) toggleFocus(l *Classic, ctx *paintengine2d.Context, b, lb paintengine2d.Rect, label string, st ControlState) {
	if !st.Focused() || st.Disabled() {
		return
	}
	c := motifColors(l)
	t := c.px(l)
	right := b.Max.X
	if label != "" {
		if x := lb.Min.X + l.body.Advance(label) + l.S(4) + t; x < right {
			right = x
		}
	}
	mHighlight(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, right-b.Min.X, b.Dy()), c.hl, t)
}

func (e motifEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := motifColors(l)
	box, lb := e.toggleLayout(l, b, l.metrics.Checkbox)
	if checked {
		st |= StateChecked
	}
	e.CheckIndicator(l, ctx, box, st, checked)
	if label != "" {
		l.drawFittedText(ctx, l.body, label, lb, c.win.fgFor(st), AlignStart, 0)
	}
	e.toggleFocus(l, ctx, b, lb, label, st)
}

func (e motifEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := motifColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box, lb := e.toggleLayout(l, b, side)
	if selected {
		st |= StateChecked
	}
	e.RadioIndicator(l, ctx, box, st, selected)
	if label != "" {
		l.drawFittedText(ctx, l.body, label, lb, c.win.fgFor(st), AlignStart, 0)
	}
	e.toggleFocus(l, ctx, b, lb, label, st)
}

// DrawSwitch: Motif had no switch; it is a sunken slot (the radio colour
// when on) with a raised slider, like a two-position XmScale.
func (e motifEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := motifColors(l)
	t := c.px(l)
	m := l.metrics
	tw, th := snap(m.SwitchW), snap(m.SwitchH)
	if th > b.Dy()-2*t {
		th = snap(b.Dy() - 2*t)
	}
	outer := paintengine2d.XYWH(snap(b.Min.X), snap(b.Min.Y+(b.Dy()-th)*0.5)-t, tw+2*t, th+2*t)
	track := outer.Inset(t)
	fill := c.trough
	if on {
		fill = c.radio
		if colorUnset(fill) {
			fill = c.act.bg
		}
	}
	if st.Disabled() {
		fill = Mix(fill, c.win.bg, 0.5)
	}
	ctx.DrawRect(track, paintengine2d.Fill(fill))
	c.sink(ctx, track, &c.win, t)
	in := track.Inset(t)
	kw := snap(in.Dx() * 0.5)
	kx := in.Min.X
	if on {
		kx = in.Max.X - kw
	}
	knob := paintengine2d.XYWH(kx, in.Min.Y, kw, in.Dy())
	ctx.DrawRect(knob, paintengine2d.Fill(c.win.bg))
	c.raise(ctx, knob, &c.win, t)
	mEtchLine(ctx, knob.Min.X+knob.Dx()*0.5, knob.Min.Y+t, knob.Dy()-2*t, true, c.win.bs, c.win.ts, 2*mPx(l, 1))
	if label != "" {
		lb := paintengine2d.XYWH(outer.Max.X+l.S(6), b.Min.Y, b.Max.X-outer.Max.X-l.S(6), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, c.win.fgFor(st), AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		mHighlight(ctx, outer, c.hl, t)
	}
}

// ---- text ------------------------------------------------------------------------------------

// ibeam is the XmText insertion cursor: an I-beam in the text colour.
func (motifEngine) ibeam(l *Classic, ctx *paintengine2d.Context, x, y, h float32, col paintengine2d.Color) {
	u := mU(l)
	x = snap(x)
	top, bot := snap(y+l.S(3)), snap(y+h-l.S(3))
	if bot-top < 3*u {
		return
	}
	f := paintengine2d.Fill(col)
	ctx.DrawRect(paintengine2d.XYWH(x, top, u, bot-top), f)
	ctx.DrawRect(paintengine2d.XYWH(x-2*u, top, 5*u, u), f)
	ctx.DrawRect(paintengine2d.XYWH(x-2*u, bot-u, 5*u, u), f)
}

// field paints the frame of a text widget: highlight band, sunken shadow,
// the text colour set's background. It returns the text area.
func (e motifEngine) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Rect {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	well := r.Inset(t)
	ctx.DrawRect(well, paintengine2d.Fill(c.text.bg))
	c.sink(ctx, well, &c.win, t)
	if st.Focused() && !st.Disabled() {
		mHighlight(ctx, r, c.hl, t)
	}
	return well.Inset(t)
}

// DrawTextField is an XmTextField: sunken in the text colour set, selection
// in reversed ground colours, the I-beam caret, and the highlight band.
func (e motifEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := motifColors(l)
	area := e.field(l, ctx, b, st)
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = l.S(6)
	}
	inner := paintengine2d.XYWH(area.Min.X+pad-l.S(2), area.Min.Y, area.Dx()-2*pad+l.S(4), area.Dy())
	if inner.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.faceOrBody(face)
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	ox := inner.Min.X - scrollX
	if text == "" {
		if placeholder != "" && !st.Focused() {
			f.Draw(ctx, placeholder, paintengine2d.Pt(ox, ty), c.text.dim)
		}
	} else {
		fg := c.text.fgFor(st)
		f.Draw(ctx, text, paintengine2d.Pt(ox, ty), fg)
		if selA != selB {
			if selA > selB {
				selA, selB = selB, selA
			}
			x0, x1 := ox+f.CaretX(text, selA), ox+f.CaretX(text, selB)
			sb := paintengine2d.XYWH(snap(x0), snap(ty+l.S(2)), snap(x1-x0), snap(f.Height()-l.S(4)))
			ctx.DrawRect(sb, paintengine2d.Fill(fg))
			ctx.Save()
			ctx.ClipRect(sb)
			f.Draw(ctx, text, paintengine2d.Pt(ox, ty), c.text.bg)
			ctx.Restore()
		}
	}
	if st.Focused() && blink && !st.Disabled() {
		e.ibeam(l, ctx, ox+f.CaretX(text, caret), ty, f.Height(), c.text.fg)
	}
	ctx.Restore()
}

// DrawTextArea is a multi-line XmText in a sunken frame.
func (e motifEngine) DrawTextArea(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	c := motifColors(l)
	area := e.field(l, ctx, b, st)
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = l.S(6)
	}
	inner := paintengine2d.XYWH(area.Min.X+pad-l.S(2), area.Min.Y+l.S(3), area.Dx()-2*pad+l.S(4), area.Dy()-l.S(6))
	if inner.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(inner)
	defer ctx.Restore()
	f := l.faceOrBody(face)
	lh := f.Height() + 2
	fg := c.text.fgFor(st)
	empty := len(lines) == 0 || (len(lines) == 1 && lines[0].Text == "" && lines[0].End <= lines[0].Start)
	if empty && placeholder != "" && !st.Focused() {
		f.Draw(ctx, placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), c.text.dim)
		return
	}
	if selA > selB {
		selA, selB = selB, selA
	}
	for i, line := range lines {
		y := inner.Min.Y + float32(i)*lh - scrollY
		if y+lh < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		ox := inner.Min.X - scrollX
		if line.Text != "" {
			f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), fg)
		}
		if selA != selB && selB > line.Start && selA < line.End {
			a, z := selA, selB
			if a < line.Start {
				a = line.Start
			}
			if z > line.End {
				z = line.End
			}
			n := len([]rune(line.Text))
			x0 := ox + f.CaretX(line.Text, a-line.Start)
			x1 := ox + f.CaretX(line.Text, z-line.Start)
			if z-line.Start > n || selB > line.Start+n {
				x1 = inner.Max.X
			}
			if x1 < x0 {
				x1 = x0
			}
			sb := paintengine2d.XYWH(snap(x0), snap(y+1), snap(x1-x0), snap(f.Height()-2))
			ctx.DrawRect(sb, paintengine2d.Fill(fg))
			if line.Text != "" {
				ctx.Save()
				ctx.ClipRect(sb)
				f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), c.text.bg)
				ctx.Restore()
			}
		}
		if st.Focused() && blink && caret >= line.Start && caret <= line.End {
			here := caret < line.End || i == len(lines)-1
			if caret == line.End && i < len(lines)-1 && lines[i+1].Start == line.End {
				here = false
			}
			if here {
				e.ibeam(l, ctx, ox+f.CaretX(line.Text, caret-line.Start), y, f.Height(), c.text.fg)
			}
		}
	}
}

// DrawComboBox is the drop-down of the pack's era: an XmOptionMenu (a
// raised button with the small raised bar glyph) or an XmComboBox (a
// sunken field beside an arrow button holding the combo glyph: a raised
// down arrow over a raised bar).
func (e motifEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := motifColors(l)
	t := c.px(l)
	u := mU(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	body := r.Inset(t)
	down := (open || st.Pressed()) && !st.Disabled()
	f := l.body
	if c.option {
		if down {
			ctx.DrawRect(body, paintengine2d.Fill(c.win.sel))
			c.sink(ctx, body, &c.win, t)
		} else {
			c.raise(ctx, body, &c.win, t)
		}
		// The option menu glyph: a raised bar as wide as ~3/4 of a line.
		bw, bh := snap(f.Height()*0.62), snap(f.Height()*0.32)
		if bh < 2*t+u {
			bh = 2*t + u
		}
		bar := paintengine2d.XYWH(body.Max.X-t-l.S(6)-bw, snap(body.Min.Y+(body.Dy()-bh)*0.5), bw, bh)
		c.raise(ctx, bar, &c.win, t)
		lb := paintengine2d.XYWH(body.Min.X+t+l.S(6), r.Min.Y, bar.Min.X-body.Min.X-t-l.S(10), r.Dy())
		l.drawFittedText(ctx, f, text, lb, c.win.fgFor(st), AlignStart, 0)
		e.highlight(l, ctx, r, st)
		return
	}
	// XmComboBox: the text field, then the arrow button.
	bw := body.Dy()
	btn := paintengine2d.XYWH(body.Max.X-bw, body.Min.Y, bw, body.Dy())
	fld := paintengine2d.XYWH(body.Min.X, body.Min.Y, btn.Min.X-body.Min.X-mPx(l, 2), body.Dy())
	ctx.DrawRect(fld, paintengine2d.Fill(c.text.bg))
	c.sink(ctx, fld, &c.win, t)
	lb := paintengine2d.XYWH(fld.Min.X+t+l.S(4), r.Min.Y, fld.Dx()-2*t-l.S(6), r.Dy())
	l.drawFittedText(ctx, f, text, lb, c.text.fgFor(st), AlignStart, 0)
	if down {
		ctx.DrawRect(btn, paintengine2d.Fill(c.win.sel))
		c.sink(ctx, btn, &c.win, t)
	} else {
		c.raise(ctx, btn, &c.win, t)
	}
	in := btn.Inset(t + u)
	top, bot := c.win.ts, c.win.bs
	if down {
		top, bot = bot, top
	}
	// XmComboBox's glyph: a down arrow in a box sqrt(3)/2 of the glyph,
	// over a raised bar filling the rest.
	side := in.Dx()
	if in.Dy() < side {
		side = in.Dy()
	}
	side = snap(side)
	nb := snap(side * 0.866)
	bh := side - nb
	if bh < 3*u {
		bh = 3 * u
		nb = side - bh
	}
	gx := snap(in.Min.X + (in.Dx()-nb)*0.5)
	gy := snap(in.Min.Y + (in.Dy()-side)*0.5)
	fill := c.win.bg
	if down {
		fill = c.win.sel
	}
	mArrow(ctx, paintengine2d.XYWH(gx, gy, nb, nb), DirDown, top, bot, fill, u)
	mShadow(ctx, paintengine2d.XYWH(gx+u, gy+nb, nb-2*u, bh), top, bot, u, false)
	if st.Focused() && !open && !st.Disabled() {
		mHighlight(ctx, r, c.hl, t)
	} else if st.Hovered() && !st.Disabled() && !open {
		mHighlight(ctx, r, c.hover(), mPx(l, 1))
	}
}

// DrawSpinner is the XmSpinBox arrow pair: bevelled arrows filled with the
// foreground (the XmArrowButton glyph), stacked; a held arrow sinks.
func (e motifEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := motifColors(l)
	u := mU(l)
	r := mSnap(b)
	mid := snap((r.Min.Y + r.Max.Y) * 0.5)
	half := func(hb paintengine2d.Rect, dir Direction, held bool) {
		top, bot := c.win.ts, c.win.bs
		if held && !st.Disabled() {
			top, bot = bot, top
		}
		mArrow(ctx, hb.Inset(u), dir, top, bot, c.win.fgFor(st), u)
	}
	half(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), mid-r.Min.Y), DirUp, upPress)
	half(paintengine2d.XYWH(r.Min.X, mid, r.Dx(), r.Max.Y-mid), DirDown, downPress)
}

// ---- ranges ----------------------------------------------------------------------------------

// DrawSlider is an XmScale: a sunken trough in the trough colour with a
// raised rectangular slider marked by an etched centre line; the highlight
// band surrounds the trough.
func (e motifEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t0 float32) {
	c := motifColors(l)
	t := c.px(l)
	if t0 < 0 {
		t0 = 0
	}
	if t0 > 1 {
		t0 = 1
	}
	r := mSnap(b)
	h := snap(l.S(16)) + 2*t
	if h > r.Dy() {
		h = r.Dy()
	}
	outer := paintengine2d.XYWH(r.Min.X, snap(r.Min.Y+(r.Dy()-h)*0.5), r.Dx(), h)
	trough := outer.Inset(t)
	if trough.Dx() < 4*t || trough.Dy() < 3*t {
		return
	}
	ctx.DrawRect(trough, paintengine2d.Fill(c.trough))
	c.sink(ctx, trough, &c.win, t)
	in := trough.Inset(t)
	sl := snap(l.metrics.Thumb)
	if sl <= 0 {
		sl = snap(l.S(30))
	}
	if max := snap(in.Dx() * 0.45); sl > max {
		sl = max
	}
	sx := snap(in.Min.X + (in.Dx()-sl)*t0)
	s := paintengine2d.XYWH(sx, in.Min.Y, sl, in.Dy())
	face := c.thumb
	if st.Pressed() && !st.Disabled() {
		face = Mix(c.thumb, c.win.sel, 0.7) // dragged: the arm colour
	}
	ctx.DrawRect(s, paintengine2d.Fill(face))
	c.raise(ctx, s, &c.win, t)
	if c.grip {
		e.grip(l, ctx, s.Inset(t), false, c.win.fg)
	} else {
		u := mPx(l, 1)
		mEtchLine(ctx, s.Min.X+snap(sl*0.5), s.Min.Y+u, s.Dy()-2*u, true, c.win.bs, c.win.ts, 2*u)
	}
	if st.Disabled() {
		ctx.DrawRect(in, paintengine2d.Fill(c.win.bg.WithAlpha(0.5)))
	}
	if st.Focused() && !st.Disabled() {
		mHighlight(ctx, outer, c.hl, t)
	}
}

// DrawProgressBar: Motif had no progress widget; CDE applications used a
// thermometer scale — a sunken trough filling with the select colour.
func (e motifEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t0 float32, indeterminate bool, phase float32) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	h := r.Dy()
	if m := snap(l.metrics.ProgressH); m > 0 && h > m {
		h = m
	}
	bar := paintengine2d.XYWH(r.Min.X, snap(r.Min.Y+(r.Dy()-h)*0.5), r.Dx(), h)
	ctx.DrawRect(bar, paintengine2d.Fill(c.win.bg))
	c.sink(ctx, bar, &c.win, t)
	in := bar.Inset(t)
	if in.Empty() {
		return
	}
	fill := c.win.sel
	if !c.graded {
		fill = Mix(c.win.sel, c.win.bs, 0.35)
	}
	if st.Disabled() {
		fill = Mix(fill, c.win.bg, 0.6)
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		w := snap(in.Dx() * 0.3)
		x := in.Min.X + (in.Dx()+w)*phase - w
		seg := paintengine2d.XYWH(snap(x), in.Min.Y, w, in.Dy()).Intersect(in)
		if !seg.Empty() {
			ctx.DrawRect(seg, paintengine2d.Fill(fill))
		}
		return
	}
	if t0 < 0 {
		t0 = 0
	}
	if t0 > 1 {
		t0 = 1
	}
	if w := snap(in.Dx() * t0); w > 0 {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy()), paintengine2d.Fill(fill))
	}
}

// ---- tabs ---------------------------------------------------------------------------------------

// DrawTabBar is the strip above an XmNotebook page: the window background
// with the page's raised top edge along its bottom (tabs stand on it, the
// selected tab opens it).
func (e motifEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	for i := float32(0); i < t; i++ {
		y := r.Max.Y - t + i
		ctx.DrawRect(paintengine2d.XYWH(r.Min.X, y, r.Dx()-i, 1), paintengine2d.Fill(c.win.ts))
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(r.Max.X-i, y, i, 1), paintengine2d.Fill(c.win.bs))
		}
	}
}

// DrawTab is an XmNotebook major tab: a raised tab with 2px shadows on its
// top and sides; unselected tabs stand lower on the page edge, the
// selected one is taller and joins the raised page. A held tab sinks.
func (e motifEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := motifColors(l)
	t := c.px(l)
	u := mPx(l, 1)
	r := mSnap(b)
	var tab paintengine2d.Rect
	if selected {
		tab = paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), r.Dy())
	} else {
		drop := snap(l.S(4))
		tab = paintengine2d.XYWH(r.Min.X+u, r.Min.Y+drop, r.Dx()-2*u, r.Dy()-drop-t)
	}
	if tab.Dx() < 3*t || tab.Dy() < 2*t {
		return
	}
	held := st.Pressed() && !st.Disabled() && !selected
	fill, top, bot := c.win.bg, c.win.ts, c.win.bs
	if held {
		fill, top, bot = c.win.sel, c.win.bs, c.win.ts
	}
	ctx.DrawRect(tab, paintengine2d.Fill(fill))
	// Top and left in the top shadow, right in the bottom shadow split on
	// the diagonal; no bottom edge (the tab stands on / opens the page).
	ctx.DrawRect(paintengine2d.XYWH(tab.Min.X, tab.Min.Y, tab.Dx(), t), paintengine2d.Fill(top))
	ctx.DrawRect(paintengine2d.XYWH(tab.Min.X, tab.Min.Y+t, t, tab.Dy()-t), paintengine2d.Fill(top))
	for i := float32(0); i < t; i++ {
		ctx.DrawRect(paintengine2d.XYWH(tab.Max.X-1-i, tab.Min.Y+i+1, 1, tab.Dy()-i-1), paintengine2d.Fill(bot))
	}
	lb := paintengine2d.XYWH(tab.Min.X+t, tab.Min.Y+t, tab.Dx()-2*t, tab.Dy()-t)
	if !selected {
		lb.Max.Y = tab.Max.Y
	}
	l.drawFittedText(ctx, l.body, label, lb, c.win.fgFor(st), AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		f := l.body
		w := f.Advance(label) + l.S(10)
		if w > lb.Dx()-l.S(2) {
			w = lb.Dx() - l.S(2)
		}
		h := f.Height() + l.S(2)
		if h > lb.Dy() {
			h = lb.Dy()
		}
		mHighlight(ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h), c.hl, t)
	}
}

// ---- menus ----------------------------------------------------------------------------------------

// DrawMenuBar is a Motif menu bar: a raised strip in the menu colour set.
func (motifEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.menu.bg))
	c.raise(ctx, r, &c.menu, c.px(l))
}

// DrawMenuTitle is a menu-bar cascade button: armed (open, pressed, under
// the pointer or reached with the keyboard) it raises a 2px shadow box
// inside the bar — Motif menus highlight by raising, not by filling.
func (e motifEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	armed := !st.Disabled() && (open || st.Pressed() || st.Hovered() || st.Focused())
	box := paintengine2d.XYWH(r.Min.X, r.Min.Y+t+mPx(l, 1), r.Dx(), r.Dy()-2*t-2*mPx(l, 1))
	if armed && box.Dy() > 2*t {
		c.raise(ctx, box, &c.menu, t)
	}
	l.drawLabeled(ctx, l.body, label, underline, r, c.menu.fgFor(st))
}

// DrawMenuFrame is a Motif pull-down: the menu colour set in a raised
// 2px frame.
func (motifEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.menu.bg))
	if c.outline {
		ctx.DrawRect(r.Inset(0.5), paintengine2d.StrokePaint(paintengine2d.RGB(0, 0, 0), 1))
		r = r.Inset(1)
	}
	c.raise(ctx, r, &c.menu, c.px(l))
}

// DrawMenuItem is a menu entry: armed, a raised shadow box across the
// menu; separators etched in; toggles show their sunken indicators only
// when set (XmNvisibleWhenOff); cascades a bevelled right arrow that sinks
// while armed; mnemonics underlined, accelerators right-aligned.
func (e motifEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := motifColors(l)
	ch := MenuChromeFor(l)
	t := c.px(l)
	u := mU(l)
	gap := mPx(l, 1)
	x0, x1 := snap(b.Min.X-ch.PadL+t+gap), snap(b.Max.X+ch.PadR-t-gap)
	if row.Separator {
		mEtchLine(ctx, x0-gap, (b.Min.Y+b.Max.Y)*0.5, x1-x0+2*gap, false, c.menu.bs, c.menu.ts, t)
		return
	}
	armed := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if armed {
		c.raise(ctx, paintengine2d.XYWH(x0, snap(b.Min.Y), x1-x0, snap(b.Max.Y)-snap(b.Min.Y)), &c.menu, t)
	}
	fg := c.menu.fgFor(st)
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	// Indicator / icon column.
	gw := ch.CheckCol()
	switch {
	case row.Radio && row.Checked:
		side := l.metrics.Radio
		if side > gw-2*u {
			side = gw - 2*u
		}
		box := paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
		c.diamond(ctx, box, &c.menu, true, st.Disabled(), u)
	case row.Checked && !row.Radio:
		side := snap(l.metrics.Checkbox)
		if side > gw-2*u {
			side = snap(gw - 2*u)
		}
		box := paintengine2d.XYWH(snap(b.Min.X+(gw-side)*0.5), snap(b.Min.Y+(b.Dy()-side)*0.5), side, side)
		fill := c.checkFill(&c.menu)
		if st.Disabled() {
			fill = Mix(fill, c.menu.bg, 0.5)
		}
		ctx.DrawRect(box, paintengine2d.Fill(fill))
		c.sink(ctx, box, &c.menu, t)
		if c.glyph {
			mCheck(ctx, box, t, u, c.markFor(&c.menu, st))
		}
	case row.Icon != IconNone:
		side := IconSizePixels(l.IconSize())
		if s := LookScale(l); s > 1.01 {
			side *= s
		}
		if side > gw-2 {
			side = gw - 2
		}
		l.drawToolIcon(ctx, paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side), row.Icon, fg)
	}
	right := b.Max.X
	if row.Submenu {
		s := snap(f.Height() * 0.55)
		if s > b.Dy()-2*t {
			s = snap(b.Dy() - 2*t)
		}
		ax := ch.ArrowMinX(b.Max.X) + (ch.SubmenuArrow-s)*0.5
		ab := paintengine2d.XYWH(snap(ax), snap(b.Min.Y+(b.Dy()-s)*0.5), s, s)
		top, bot := c.menu.ts, c.menu.bs
		if armed {
			top, bot = bot, top // the armed cascade arrow sinks
		}
		mArrow(ctx, ab, DirRight, top, bot, c.menu.bg, u)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := f.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg)
		right = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if right < lx {
		right = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// ---- lists, trees, tables ------------------------------------------------------------------------

// DrawListRow is an XmList item: selected items in the select colour (or
// reversed ground colours); Motif lists did not track the pointer.
func (motifEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string) {
	c := motifColors(l)
	fg := c.text.fg
	if selected {
		ctx.DrawRect(mSnap(b), paintengine2d.Fill(c.selBg))
		fg = c.selFg
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy()), fg, AlignStart, 0)
}

// DrawTreeRow is an XmContainer outline row: solid outline lines, the
// bevelled outline button, the selected label highlighted.
func (e motifEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	c := motifColors(l)
	u := mPx(l, 1)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	line := paintengine2d.Fill(Mix(c.text.fg, c.text.bg, 0.55))
	ctx.Save()
	ctx.ClipRect(b)
	for d := 0; d < depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
		ctx.DrawRect(paintengine2d.XYWH(gx, b.Min.Y, u, b.Dy()), line)
	}
	if depth > 0 {
		gx := snap(b.Min.X + pad + float32(depth-1)*indent + indent*0.5)
		ctx.DrawRect(paintengine2d.XYWH(gx, cy, x-gx+l.S(2), u), line)
	}
	ctx.Restore()
	if !leaf {
		e.expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, &c.text)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(4)
	fg := c.text.fg
	if selected {
		hb := paintengine2d.XYWH(lx-l.S(3), b.Min.Y+u, f.Advance(label)+l.S(6), b.Dy()-2*u).Intersect(b)
		ctx.DrawRect(mSnap(hb), paintengine2d.Fill(c.selBg))
		fg = c.selFg
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawTableHeader is a column heading: a raised push button, sunken while
// held, with a bevelled sort arrow.
func (e motifEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	held := st.Pressed() && !st.Disabled()
	fill := c.win.bg
	if held {
		fill = c.win.sel
	}
	ctx.DrawRect(r, paintengine2d.Fill(fill))
	if held {
		c.sink(ctx, r, &c.win, t)
	} else {
		c.raise(ctx, r, &c.win, t)
	}
	aw := float32(0)
	if sorted {
		s := snap(l.S(10))
		aw = s + l.S(6)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		ab := paintengine2d.XYWH(r.Max.X-t-l.S(4)-s, snap(r.Min.Y+(r.Dy()-s)*0.5), s, s)
		mArrow(ctx, ab, dir, c.win.ts, c.win.bs, c.win.bg, mU(l))
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(r.Min.X+t+l.S(4), r.Min.Y, r.Dx()-2*t-l.S(6)-aw, r.Dy()), c.win.fgFor(st), AlignStart, 0)
}

func (motifEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font) {
	c := motifColors(l)
	fg := c.text.fg
	if selected {
		ctx.DrawRect(mSnap(b), paintengine2d.Fill(c.selBg))
		fg = c.selFg
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
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// ---- bars and panels -----------------------------------------------------------------------------------

// DrawToolBar is a raised strip holding raised tool buttons.
func (motifEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	c.raise(ctx, r, &c.win, c.px(l))
}

// DrawStatusBar is a footer of sunken one-pixel panels (XmFrame
// XmSHADOW_IN).
func (motifEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := motifColors(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	if len(parts) == 0 {
		return
	}
	u := mPx(l, 1)
	gap := snap(l.S(2))
	slot := (r.Dx() - gap) / float32(len(parts))
	for i, s := range parts {
		x0 := snap(r.Min.X + gap + slot*float32(i))
		x1 := snap(r.Min.X + slot*float32(i+1))
		p := paintengine2d.XYWH(x0, r.Min.Y+gap, x1-x0, r.Dy()-2*gap)
		mShadow(ctx, p, c.win.bs, c.win.ts, u, false)
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(p.Min.X+l.S(6), r.Min.Y, p.Dx()-l.S(10), r.Dy()), c.win.fg, AlignStart, 0)
	}
}

// DrawTitleBar is a heading strip: bold title and dim subtitle over an
// etched XmSeparator.
func (motifEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	mEtchLine(ctx, r.Min.X, r.Max.Y-t*0.5, r.Dx(), false, c.win.bs, c.win.ts, t)
	x := r.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, r.Min.Y, r.Max.X-x, r.Dy()-t), c.win.fg, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < r.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, r.Min.Y, r.Max.X-x, r.Dy()-t), c.win.dim, AlignStart, 0)
	}
}

// DrawPanel: a raised panel is an XmFrame with XmSHADOW_OUT, a flat one
// the XmFrame default, etched in.
func (motifEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	if raised {
		c.raise(ctx, r, &c.win, t)
		return
	}
	c.etch(ctx, r, &c.win, t, false)
}

// DrawAccordionHeader: Motif had none; it is a raised push button carrying
// an outline arrow.
func (e motifEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	face := r.Inset(t)
	held := st.Pressed() && !st.Disabled()
	if held {
		ctx.DrawRect(face, paintengine2d.Fill(c.win.sel))
		c.sink(ctx, face, &c.win, t)
	} else {
		c.raise(ctx, face, &c.win, t)
	}
	e.expander(l, ctx, paintengine2d.XYWH(face.Min.X+t+l.S(2), face.Min.Y, l.S(16), face.Dy()), expanded, &c.win)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(face.Min.X+t+l.S(22), r.Min.Y, face.Dx()-2*t-l.S(24), r.Dy()), c.win.fgFor(st), AlignStart, 0)
	e.highlight(l, ctx, r, st)
}

func (motifEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := motifColors(l)
	t := c.px(l)
	if vertical {
		mEtchLine(ctx, (b.Min.X+b.Max.X)*0.5, b.Min.Y+l.S(2), b.Dy()-l.S(4), true, c.win.bs, c.win.ts, t)
		return
	}
	mEtchLine(ctx, b.Min.X, (b.Min.Y+b.Max.Y)*0.5, b.Dx(), false, c.win.bs, c.win.ts, t)
}

// DrawSplitter is an XmPanedWindow separator: an etched line with the
// raised sash square near its far end (sinks while dragged).
func (motifEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := motifColors(l)
	t := c.px(l)
	r := mSnap(b)
	ctx.DrawRect(r, paintengine2d.Fill(c.win.bg))
	s := snap(l.S(10))
	var sash paintengine2d.Rect
	if vertical {
		mEtchLine(ctx, (r.Min.X+r.Max.X)*0.5, r.Min.Y, r.Dy(), true, c.win.bs, c.win.ts, t)
		if s > r.Dx() {
			s = r.Dx()
		}
		sash = paintengine2d.XYWH(snap(r.Min.X+(r.Dx()-s)*0.5), r.Max.Y-snap(l.S(10))-s, s, s)
	} else {
		mEtchLine(ctx, r.Min.X, (r.Min.Y+r.Max.Y)*0.5, r.Dx(), false, c.win.bs, c.win.ts, t)
		if s > r.Dy() {
			s = r.Dy()
		}
		sash = paintengine2d.XYWH(r.Max.X-snap(l.S(10))-s, snap(r.Min.Y+(r.Dy()-s)*0.5), s, s)
	}
	sash = sash.Intersect(r)
	if sash.Dx() < 2*t+1 || sash.Dy() < 2*t+1 {
		return
	}
	ctx.DrawRect(sash, paintengine2d.Fill(c.win.bg))
	if st.Pressed() {
		c.sink(ctx, sash, &c.win, t)
	} else {
		c.raise(ctx, sash, &c.win, t)
	}
}

// DrawTooltip: Motif had none; CDE's help is a pale yellow box with a thin
// black border.
func (motifEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := motifColors(l)
	r := mSnap(b)
	u := mPx(l, 1)
	ctx.DrawRect(r, paintengine2d.Fill(c.infoFg))
	ctx.DrawRect(r.Inset(u), paintengine2d.Fill(c.info))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(r.Min.X+pad, r.Min.Y, r.Dx()-pad*2, r.Dy()), c.infoFg, AlignStart, 0)
}

// DrawFocusRing is the Motif highlight rectangle: a solid band inside b.
func (motifEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := motifColors(l)
	mHighlight(ctx, b, c.hl, c.px(l))
}

// ---- packs -----------------------------------------------------------------------------------------------

// motifSets names the colour-set backgrounds of a pack as X colour specs
// (CDE palette files give 16-bit #rrrrggggbbbb).
type motifSets struct {
	window, menu, text, active, inactive string
}

func motifPack(name, label string, year int, summary string, cs motifSets, extra map[string]string, params map[string]float32) ThemePack {
	win := motifSet(motifSpec(cs.window), true)
	menu := motifSet(motifSpec(cs.menu), true)
	text := motifSet(motifSpec(cs.text), true)
	act := motifSet(motifSpec(cs.active), false)
	if s, ok := extra["activeFg"]; ok {
		act.fg = motifShow(motifSpec(s))
	}
	if s, ok := extra["textFg"]; ok {
		text.fg = motifShow(motifSpec(s))
	}
	fam := ThemeLight
	if RelLuminance(win.fg) > 0.5 {
		fam = ThemeDark
	}
	hl := win.fg
	if s, ok := extra["highlight"]; ok {
		hl = motifShow(motifSpec(s))
	}
	trough := win.sel
	if s, ok := extra["trough"]; ok {
		trough = motifShow(motifSpec(s))
	}
	danger, success, warning := Hex("#b01818"), Hex("#146c14"), Hex("#8a5a00")
	if fam == ThemeDark {
		danger, success, warning = Hex("#ffa0a0"), Hex("#a8f0a8"), Hex("#ffe08a")
	}
	sel := text.sel
	if params["reverseSelect"] != 0 {
		sel = text.fg
	}
	pal := Palette{
		Background: win.bg, Surface: win.bg, SurfaceAlt: win.bg,
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.35),
		Border:  win.bs, Divider: win.bs,
		Text: win.fg, TextMuted: Mix(win.fg, win.bg, 0.42), TextOnAccent: act.fg,
		Accent: act.bg, AccentHover: act.ts, AccentPress: act.bs,
		Danger: danger, Success: success, Warning: warning,
		Track: trough, Thumb: win.bg,
		Field: text.bg, FieldBorder: win.bs,
		Focus: hl, Selection: sel,
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.3), Highlight: win.ts,
		MenuHover: menu.bg, MenuHoverBorder: menu.bs, MenuGutter: menu.bg,
		BevelLight: win.ts, BevelDark: win.bs,
	}
	tok := ThemeTokens{
		Engine: "motif", Bevel: BevelClassic3D, Family: fam, Palette: pal,
		Extra: map[string]paintengine2d.Color{
			"window": motifSpec(cs.window), "menu": motifSpec(cs.menu), "text": motifSpec(cs.text),
			"active": motifSpec(cs.active), "inactive": motifSpec(cs.inactive),
		},
		Params:   params,
		Hot:      ChromeState{Fill: menu.bg, Border: menu.bs},
		Pressed:  ChromeState{Fill: win.sel, Border: win.bs},
		Selected: ChromeState{Fill: sel, Border: text.bs},
		Focus:    ChromeState{Fill: hl.WithAlpha(0.16), Border: hl},
	}
	for k, v := range extra {
		tok.Extra[k] = motifSpec(v)
	}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Unix", Summary: summary,
		Era: EraMotif, Palette: fam, Tokens: tok,
	}
}

// motifPacks are the Motif looks. CDE palettes are the colour sets of the
// .dp files CDE shipped (cde/programs/palettes, CDE 2.1.30), in the
// HIGH_COLOR mapping — set 5 windows, 6 menus, 4 text, 1 and 2 frames —
// unless noted; the shadows come from the same XmGetColors arithmetic the
// CDE colour server used.
func motifPacks() []ThemePack {
	cde := map[string]float32{"checkGlyph": 1}
	// CDE highlights focus in the active window colour and fills set radio
	// diamonds with it (XmNenableToggleColor).
	cdeHL := func(active string) map[string]string {
		return map[string]string{"highlight": active, "radio": active}
	}
	return []ThemePack{
		motifPack("motif", "Motif", 1990,
			"OSF/Motif and mwm as shipped: #c4c4c4 widgets, shadows derived by XmGetColors, diamond radios, CadetBlue frames.",
			motifSets{window: "#c4c4c4", menu: "#c4c4c4", text: "#d3d3d3", active: "#5f9ea0", inactive: "#d3d3d3"},
			nil, map[string]float32{"optionMenu": 1, "reverseSelect": 1}),
		motifPack("hp-vue", "HP VUE", 1990,
			"HP's Visual User Environment, CDE's parent: light-blue Motif windows, blue menus, salmon active frames.",
			motifSets{window: "#82a8d8", menu: "#3296c6", text: "#6d829c", active: "#fd8989", inactive: "#7acac5"},
			map[string]string{"highlight": "#fd8989"}, map[string]float32{"optionMenu": 1}),
		motifPack("cde", "CDE Default", 1993,
			"The Common Desktop Environment's default palette: sand windows, teal menus, slate text areas, orange active frames.",
			motifSets{window: "#c600b2d2a87e", menu: "#49009200a700", text: "#68006f008200", active: "#ed00a8007000", inactive: "#9900991b99fe"},
			cdeHL("#ed00a8007000"), cde),
		motifPack("cde-alpine", "CDE Alpine", 1993,
			"CDE's Alpine palette: tan windows, cool grey menus and text areas, periwinkle active frames.",
			motifSets{window: "#bf0092007900", menu: "#bf00c000c500", text: "#bf00c000c500", active: "#af35bfa1fb00", inactive: "#7800aa00b800"},
			cdeHL("#af35bfa1fb00"), cde),
		motifPack("cde-broica", "CDE Broica", 1993,
			"Broica, the palette CDE's Default copies, as four colour sets (MEDIUM_COLOR): blue-grey windows and menus.",
			motifSets{window: "#89559808aa00", menu: "#89559808aa00", text: "#68006f008200", active: "#ed00a8007000", inactive: "#9900991b99fe"},
			cdeHL("#ed00a8007000"), cde),
		motifPack("cde-charcoal", "CDE Charcoal", 1993,
			"CDE's dark Charcoal palette: grey windows, slate-green menus, rose text areas, white labels.",
			motifSets{window: "#820085008500", menu: "#63ae6f006dde", text: "#89006d126d12", active: "#8787b400b000", inactive: "#760072c27900"},
			cdeHL("#8787b400b000"), cde),
		motifPack("cde-crimson", "CDE Crimson", 1993,
			"CDE's Crimson palette: sea-green windows, steel-blue menus, cream text areas, crimson active frames.",
			motifSets{window: "#68cea600a0e6", menu: "#8d40ad03c700", text: "#ff00f700e900", active: "#b2004d007a00", inactive: "#ae00b200c300"},
			cdeHL("#b2004d007a00"), cde),
		motifPack("cde-desert", "CDE Desert", 1993,
			"CDE's Desert palette: dusty blue windows, teal menus, white text areas, sand active frames.",
			motifSets{window: "#9f2dae00b500", menu: "#6f519800a25a", text: "#fd00fd00fa00", active: "#e500b7ae93d8", inactive: "#c200bb00a700"},
			cdeHL("#e500b7ae93d8"), cde),
		motifPack("irix", "IRIX Indigo Magic", 1993,
			"SGI IRIX Indigo Magic: #c1c1c1 IRIS IM widgets with SGI's graded shading, lavender fields, khaki active frames.",
			motifSets{window: "#c1c1c1", menu: "#c1c1c1", text: "#8e8eb9", active: "#a59f80", inactive: "#808080"},
			map[string]string{"ts": "#e8e8e8", "bs": "#707070", "trough": "#999999", "thumb": "#999999",
				"textFg": "#000000", "activeFg": "#000000", "inactiveFg": "#000000",
				"check": "#efefef", "checkMark": "#ff0000", "radio": "#efefef", "radioPip": "#0000ff"},
			map[string]float32{"optionMenu": 1, "graded": 1, "outline": 1, "arrowBox": 1, "grip": 1, "titleLeft": 1, "checkGlyph": 1}),
	}
}
