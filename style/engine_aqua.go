package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// aquaEngine paints Mac OS X Aqua (10.0 Cheetah – 10.4 Tiger): lickable
// gel pills (white for plain buttons, blue — or graphite — for the default
// one), rounded gel check boxes whose black tick overshoots the box, gel
// radio beads, 15px scroll gutters with a gel thumb and both arrows
// together at the end, a segmented tab strip straddling a rounded pane,
// popup buttons with a blue double-arrow cap, horizontal pinstripes (or
// brushed metal) behind everything, and traffic lights on the left of the
// title bar.
//
// Recipes were sampled from 1:1 screenshots of 10.1, 10.2 and 10.4: the
// white gel runs #f8 → #ee under a gloss that ends near half height, dips
// to #e5 and glows back to white at the bottom; the blue gel runs navy
// outline, #c5d3ec gloss, #4d96d7 body, #9bd8f7 bottom glow, with the ends
// darkened to #0a50c0 (the "cylinder" rim). Window pinstripes repeat every
// 4px (#e6 / #ee / #ff / #ee on 10.1).
//
// Pack data (theme.json "extra" colours, derived from the palette when a
// pack omits them):
//
//	stripeLo, stripeHi        pinstripe lines over Palette.Background
//	metalHi, metalLo          brushed-metal centre / edge (param metal=1)
//	gel, gelTop, gelGlow      accent gel body middle / under the gloss / glow
//	gelEdge, gelRim           accent gel outline / end darkening
//	clear, clearTop, clearGlow, clearEdge   plain (white) gel
//	listSel                   list, table and tree selection
//	menuBg, menuHi, menuHi2   menu background, highlight band top / bottom
//	box, boxEdge              group box / tab pane wash and outline
//	title, title2             title bar gradient top / bottom
//	info, infoEdge            help tag (tooltip)
//	glow                      keyboard focus glow
//
// Params: stripe (pinstripe period in px, 0 = flat), metal (1 = brushed).
type aquaEngine struct{ BaseEngine }

func init() {
	RegisterEngine(aquaEngine{})
	for _, p := range aquaPacks() {
		RegisterPack(p)
	}
}

func (aquaEngine) ID() string { return "aqua" }

// DefaultMetrics are Aqua's proportions at the toolkit's 16px UI font
// (Aqua used 13pt Lucida Grande): 20px gel pills become ~24px pills in a
// 30px control, 14px check boxes, 15px scroll gutters.
func (aquaEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    6, RadiusSmall: 4,
		ControlH: 30, FieldH: 26, ComboH: 28,
		Checkbox: 14, Radio: 15,
		MenuItemH: 24, MenuBarH: 24, TabH: 30, RowH: 22,
		TitleBar: 28, HeaderH: 22, ProgressH: 16, SliderH: 24, Thumb: 17,
		Scroll: 15, Pad: 14, FieldPad: 6, FocusWidth: 3, Border: 1,
		ToolBarH: 40, StatusBarH: 24, SpinnerW: 17, SwitchW: 40, SwitchH: 22,
	}
}

// StyleHint: Mac dialog order (default button last) and centred tabs.
func (aquaEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDefaultPulseMs {
		return 1600 // the default button throbs
	}
	if h == HintTabsCentered || h == HintFormLabelsRight {
		return 1
	}
	return 0
}

// ---- colours ------------------------------------------------------------------

// aquaGel is one gel material: outline, body ramp (dark under the gloss,
// bright bottom glow), the rim tint that darkens its ends and the gloss.
// The stop slices are built once per look (see aquaBuild).
type aquaGel struct {
	edgeTop, edgeBot    paintengine2d.Color
	top, mid, glow, rim paintengine2d.Color
	fg                  paintengine2d.Color
	glossTop, glossBot  float32
	edge, body, gloss   []paintengine2d.GradientStop
	caps, ring          []paintengine2d.GradientStop
	shadowA             float32
	glyph               paintengine2d.Color // arrows / ticks drawn on the gel
	readOn              paintengine2d.Color // what the label mostly sits on
}

// done builds the stop slices from the gel's colours.
func (g aquaGel) done() aquaGel {
	white := paintengine2d.RGB(1, 1, 1)
	g.edge = []paintengine2d.GradientStop{Stop(0, g.edgeTop), Stop(1, g.edgeBot)}
	g.body = []paintengine2d.GradientStop{Stop(0, g.top), Stop(0.5, g.mid), Stop(1, g.glow)}
	g.gloss = []paintengine2d.GradientStop{
		Stop(0, white.WithAlpha(g.glossTop)), Stop(0.5, white.WithAlpha(g.glossBot)),
		Stop(0.5, white.WithAlpha(0)), Stop(1, white.WithAlpha(0)),
	}
	g.caps = []paintengine2d.GradientStop{Stop(0, g.rim), Stop(1, g.rim.WithAlpha(0))}
	g.ring = []paintengine2d.GradientStop{Stop(0, g.rim.WithAlpha(0)), Stop(0.6, g.rim.WithAlpha(0)), Stop(1, g.rim)}
	g.readOn = Mix(g.mid, g.glow, 0.25)
	return g
}

// toward mixes every colour of g towards c (hover, disabled variants).
func (g aquaGel) toward(c paintengine2d.Color, t, gloss float32) aquaGel {
	mix := func(a paintengine2d.Color) paintengine2d.Color {
		m := Mix(a, c, t)
		m.A = a.A
		return m
	}
	g.edgeTop, g.edgeBot = mix(g.edgeTop), mix(g.edgeBot)
	g.top, g.mid, g.glow = mix(g.top), mix(g.mid), mix(g.glow)
	g.rim = mix(g.rim)
	g.rim.A *= 1 - t
	g.glossTop *= gloss
	g.glossBot *= gloss
	return g.done()
}

// aquaNewGel makes a gel from its outline, body and rim colours.
func aquaNewGel(edgeTop, edgeBot, top, mid, glow, rim paintengine2d.Color, glossTop, glossBot float32) aquaGel {
	return aquaGel{
		edgeTop: edgeTop, edgeBot: edgeBot, top: top, mid: mid, glow: glow, rim: rim,
		glossTop: glossTop, glossBot: glossBot, shadowA: 0.3,
	}.done()
}

// aquaStreak is one brushed-metal pixel row: a faint white or dark line
// over part of the width, fading in and out along x.
type aquaStreak struct {
	from, to float32 // span, as fractions of the width
	stops    []paintengine2d.GradientStop
}

// aqua is the resolved paint set of a look.
type aqua struct {
	win, stripeLo, stripeHi paintengine2d.Color
	period                  float32
	metal, dark             bool
	metalStops              []paintengine2d.GradientStop
	streaks                 [61]aquaStreak

	text, dim, glyph paintengine2d.Color

	clear, clearHot, accent, accentHot, press, off aquaGel
	accentOff                                      aquaGel // selected but disabled
	lights                                         [3]aquaGel
	lightOff, lightPress                           aquaGel

	field, fieldTop, fieldSide, fieldBot paintengine2d.Color
	sel, selTxt, hover                   paintengine2d.Color
	selOff, selOffTxt                    paintengine2d.Color // selection of an unfocused list
	stripe                               paintengine2d.Color // alternate row wash (0 alpha: none)
	box, boxEdge, boxEdgeLo              paintengine2d.Color
	menuBg, menuEdge, menuTxt, menuHiTxt paintengine2d.Color
	menuHiStops                          []paintengine2d.GradientStop
	menuHiLo, menuHiHi                   paintengine2d.Color
	menuLo, menuLine                     paintengine2d.Color
	barLo, barHi, barEdge, barLine       paintengine2d.Color
	titleLo, titleHi                     paintengine2d.Color
	titleStops                           []paintengine2d.GradientStop
	titleEdge, titleTxt, titleOff        paintengine2d.Color
	headStops, headSelStops              []paintengine2d.GradientStop
	headEdge                             paintengine2d.Color
	trackStops                           []paintengine2d.GradientStop
	trackEdge, gutter                    paintengine2d.Color
	grooveStops                          []paintengine2d.GradientStop
	grooveEdge                           paintengine2d.Color
	info, infoEdge, infoTxt              paintengine2d.Color
	focus, menuSep                       paintengine2d.Color
	paneEdge, progTrack, pole, poleOff   []paintengine2d.GradientStop
	sep, sepHi                           paintengine2d.Color
	disclose                             paintengine2d.Color
}

type aquaKey struct{}

// aquaColors is the look's resolved paint set (built once per look).
func aquaColors(l *Classic) *aqua {
	return l.Memo(aquaKey{}, func() any { return aquaBuild(l) }).(*aqua)
}

func aquaBuild(l *Classic) *aqua {
	p := l.palette
	dark := RelLuminance(p.Background) < 0.2
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c := &aqua{
		win:      p.Background,
		stripeLo: l.X("stripeLo", paintengine2d.Color{}),
		stripeHi: l.X("stripeHi", paintengine2d.Color{}),
		period:   l.P("stripe", 4),
		metal:    l.P("metal", 0) != 0,
		dark:     dark,
		text:     p.Text,
		dim:      p.TextMuted,
		field:    p.Field,
	}
	c.glyph = l.X("glyph", Mix(p.Text, p.Background, 0.25))

	// Gels. The accent (default button, checked boxes, thumbs) follows the
	// pack's "gel" family; the plain gel follows "clear".
	mid := l.X("gel", Hex("#4d96d7"))
	gedge := l.X("gelEdge", Shade(mid, -0.72))
	gglow := l.X("gelGlow", Mix(mid, Hex("#dcf6ff"), 0.6))
	c.accent = aquaNewGel(gedge, Mix(gedge, gglow, 0.45),
		l.X("gelTop", Shade(mid, -0.38)), mid, gglow,
		l.X("gelRim", Shade(mid, -0.55)).WithAlpha(0.8), 0.8, 0.3)
	cmid := l.X("clear", Mix(p.Field, p.SurfaceAlt, 0.6))
	cedge := l.X("clearEdge", Shade(cmid, -0.45))
	c.clear = aquaNewGel(cedge, Mix(cedge, cmid, 0.35),
		l.X("clearTop", Shade(cmid, 0.1)), cmid, l.X("clearGlow", Shade(cmid, 0.9)),
		black.WithAlpha(0.15), 0.62, 0.3)
	if dark {
		c.clear.glossTop, c.clear.glossBot = 0.28, 0.1
		c.clear = c.clear.done()
		c.accent.glossTop, c.accent.glossBot = 0.5, 0.18
		c.accent = c.accent.done()
	}
	pmid := l.X("press", Shade(mid, -0.2))
	pedge := l.X("pressEdge", Shade(c.accent.edgeTop, -0.3))
	c.press = aquaNewGel(pedge, Mix(pedge, l.X("pressGlow", Shade(c.accent.glow, -0.2)), 0.4),
		l.X("pressTop", Shade(c.accent.top, -0.25)), pmid, l.X("pressGlow", Shade(c.accent.glow, -0.2)),
		Shade(c.accent.rim, -0.2).WithAlpha(0.85), 0.66, 0.24)
	c.clearHot = c.clear.toward(white, 0.3, 1.1)
	if dark {
		c.clearHot = c.clear.toward(white, 0.1, 1.1)
	}
	c.accentHot = c.accent.toward(white, 0.14, 1.05)
	c.off = c.clear.toward(p.Background, 0.55, 0.45)
	c.accentOff = c.accent.toward(p.Background, 0.55, 0.5)
	c.clear.fg = ReadableOn(c.clear.readOn, 4.5, p.Text)
	c.clearHot.fg = ReadableOn(c.clearHot.readOn, 4.5, p.Text)
	c.accent.fg = ReadableOn(c.accent.readOn, 4.5, black, p.Text)
	c.accentHot.fg = ReadableOn(c.accentHot.readOn, 4.5, black, p.Text)
	c.press.fg = ReadableOn(c.press.readOn, 4.5, black, white)
	c.off.fg, c.accentOff.fg = p.TextMuted, p.TextMuted
	for _, g := range []*aquaGel{&c.clear, &c.clearHot, &c.accent, &c.accentHot, &c.press, &c.off, &c.accentOff} {
		g.glyph = ReadableOn(g.readOn, 3, c.glyph, p.Text)
	}
	c.off.glyph = Mix(c.off.glyph, p.Background, 0.45)
	c.off.shadowA = 0.08

	// Traffic lights (sampled from Tiger): red, yellow, green, and the clear
	// bead of an inactive window.
	light := func(edge, top, mid, glow, rim string) aquaGel {
		g := aquaNewGel(Hex(edge), Mix(Hex(edge), Hex(glow), 0.4), Hex(top), Hex(mid), Hex(glow), Hex(rim).WithAlpha(0.75), 0.85, 0.2)
		g.fg = Shade(Hex(rim), -0.55)
		return g
	}
	c.lights = [3]aquaGel{
		light("#4a0c08", "#c0443c", "#dc6258", "#ffc1b4", "#8a1812"),
		light("#5a3a0e", "#dc9422", "#f2b53c", "#fff47e", "#9a5e10"),
		light("#2c4a12", "#5a9e1a", "#78b832", "#c6f784", "#3a6a14"),
	}
	c.lightOff = light("#8a8a8a", "#d4d4d4", "#e2e2e2", "#fbfbfb", "#9a9a9a")
	if dark {
		c.lightOff = light("#1c1c1e", "#4c4c50", "#5a5a5e", "#86868c", "#2a2a2c")
	}
	c.lightPress = c.lights[0].toward(black, 0.25, 0.9)
	c.lightPress.fg = c.lights[0].fg

	// Fields: a dark top edge with an inner shadow, light sides and bottom.
	c.fieldTop = l.X("fieldTop", Mix(p.FieldBorder, black, 0.1))
	c.fieldSide = l.X("fieldSide", Mix(p.FieldBorder, p.Background, 0.5))
	c.fieldBot = l.X("fieldBot", Mix(p.FieldBorder, p.Background, 0.7))

	c.sel = l.X("listSel", p.Accent)
	c.selTxt = ReadableOn(c.sel, 4.5, p.TextOnAccent, white, black)
	c.hover = c.sel.WithAlpha(0.1)
	// Mac OS X greys the selection of a list without focus.
	c.selOff = l.X("listSelInactive", Mix(p.Field, Hex("#8e8e8e"), 0.4))
	if c.dark {
		c.selOff = Mix(p.Field, white, 0.18)
	}
	c.selOffTxt = ReadableOn(c.selOff, 4.5, p.Text, black, white)
	// Panther-era lists (iTunes, Mail, Finder's list view) stripe their
	// rows white and pale blue; "stripes" 0 turns it off.
	if l.P("stripes", 1) > 0 {
		c.stripe = l.X("listStripe", Hex("#edf3fe"))
		if c.dark {
			c.stripe = Mix(p.Field, white, 0.05)
		}
	}

	c.box = l.X("box", black.WithAlpha(0.045))
	c.boxEdge = l.X("boxEdge", Mix(p.Border, p.Background, 0.15))
	c.boxEdgeLo = Mix(c.boxEdge, p.Background, 0.55)
	c.paneEdge = []paintengine2d.GradientStop{Stop(0, c.boxEdge), Stop(0.25, c.boxEdgeLo), Stop(1, c.boxEdgeLo)}

	c.menuBg = l.X("menuBg", Mix(p.Field, p.Background, 0.2).WithAlpha(0.97))
	c.menuEdge = Mix(p.Border, p.Background, 0.35).WithAlpha(0.8)
	c.menuTxt = ReadableOn(Mix(p.Background, c.menuBg, c.menuBg.A), 4.5, p.Text)
	hi1 := l.X("menuHi", Hex("#3f74c6"))
	hi2 := l.X("menuHi2", Shade(hi1, -0.12))
	c.menuHiStops = []paintengine2d.GradientStop{Stop(0, hi1), Stop(1, hi2)}
	c.menuHiLo, c.menuHiHi = Shade(hi1, -0.14).WithAlpha(0.8), Shade(hi1, 0.08).WithAlpha(0.8)
	c.menuHiTxt = ReadableOn(Mix(hi1, hi2, 0.5), 4.5, white, black)
	c.menuSep = Mix(Mix(p.Border, p.Background, 0.25), c.menuBg, 0.45)
	c.menuLo = Mix(c.menuBg, black, 0.035).WithAlpha(0.5)
	c.menuLine = white.WithAlpha(0.5)
	if dark {
		c.menuLine = white.WithAlpha(0.03)
	}
	c.barLo = l.X("barLo", Mix(p.Background, black, 0.02))
	c.barHi = l.X("barHi", Mix(p.Field, p.Background, 0.3))
	c.barEdge = Mix(p.Border, p.Background, 0.3)
	c.barLine = white
	c.titleLo, c.titleHi = black.WithAlpha(0.035), white.WithAlpha(0.55)
	if dark {
		c.barLine = Mix(c.barHi, white, 0.05)
		c.titleLo, c.titleHi = black.WithAlpha(0.12), white.WithAlpha(0.035)
	}

	t1 := l.X("title", Shade(p.Background, 0.7))
	t2 := l.X("title2", Shade(p.Background, -0.07))
	c.titleStops = []paintengine2d.GradientStop{Stop(0, t1), Stop(1, t2)}
	c.titleEdge = Mix(p.Border, black, 0.1)
	c.titleTxt = ReadableOn(Mix(t1, t2, 0.5), 4.5, p.Text)
	c.titleOff = Mix(c.titleTxt, Mix(t1, t2, 0.5), 0.5)

	h1, h2 := Mix(p.Field, p.SurfaceAlt, 0.1), Mix(p.SurfaceAlt, black, 0.06)
	c.headStops = []paintengine2d.GradientStop{Stop(0, h1), Stop(0.5, Mix(h1, h2, 0.45)), Stop(0.5, Mix(h1, h2, 0.6)), Stop(1, h2)}
	c.headSelStops = []paintengine2d.GradientStop{Stop(0, c.accent.edgeTop.WithAlpha(0)), Stop(0.02, Mix(c.accent.top, white, 0.55)),
		Stop(0.5, Mix(c.accent.mid, white, 0.35)), Stop(0.5, c.accent.mid), Stop(1, c.accent.glow)}
	c.headEdge = Mix(p.Border, p.Field, 0.25)

	// Scroll gutter: an inset channel lit from the top left.
	c.gutter = Mix(p.Background, p.Field, 0.4)
	tk := Mix(p.Field, p.Background, 0.5)
	c.trackStops = []paintengine2d.GradientStop{Stop(0, Shade(tk, -0.2)), Stop(0.35, Shade(tk, -0.08)), Stop(0.7, Shade(tk, 0.06)), Stop(1, Shade(tk, -0.02))}
	c.trackEdge = Mix(p.Border, p.Background, 0.45)
	g1 := Mix(p.Border, black, 0.25)
	c.grooveStops = []paintengine2d.GradientStop{Stop(0, g1), Stop(0.45, Mix(g1, p.Background, 0.45)), Stop(1, Mix(p.Background, white, 0.4))}
	c.grooveEdge = Mix(g1, black, 0.2)

	c.info = l.X("info", Hex("#ffffc7"))
	c.infoEdge = l.X("infoEdge", Hex("#a4a4a4"))
	c.infoTxt = ReadableOn(c.info, 4.5, p.Text, black)
	c.focus = l.X("glow", Hex("#5b8fe8"))
	c.sep = Mix(p.Border, p.Background, 0.25)
	c.sepHi = Mix(p.Background, white, 0.7)
	if dark {
		c.sepHi = Mix(p.Background, white, 0.06)
	}
	c.disclose = Mix(p.Text, p.Background, 0.45)
	lift := float32(0.6)
	if dark {
		lift = 0.08
	}
	c.progTrack = []paintengine2d.GradientStop{Stop(0, Mix(c.gutter, black, 0.12)), Stop(0.3, c.gutter), Stop(1, Mix(c.gutter, white, lift))}
	c.pole = []paintengine2d.GradientStop{Stop(0, c.accent.mid), Stop(0.5, c.accent.mid), Stop(0.5, c.clear.mid), Stop(1, c.clear.mid)}
	c.poleOff = []paintengine2d.GradientStop{Stop(0, c.off.mid), Stop(0.5, c.off.mid), Stop(0.5, c.off.glow), Stop(1, c.off.glow)}

	if c.metal {
		hi, lo := l.X("metalHi", Shade(p.Background, 0.12)), l.X("metalLo", Shade(p.Background, -0.08))
		c.metalStops = []paintengine2d.GradientStop{Stop(0, lo), Stop(0.5, hi), Stop(1, lo)}
		// Brushed streaks: a deterministic pseudo-random walk per pixel row
		// (weakly correlated grain, the odd stronger scratch), each streak
		// covering part of the width and fading at its ends.
		s := uint32(0x9e3779b9)
		next := func() float32 {
			s ^= s << 13
			s ^= s >> 17
			s ^= s << 5
			return float32(s%1000) / 1000
		}
		walk := float32(0)
		for i := range c.streaks {
			r := next() - 0.5
			walk = walk*0.35 + r*0.07
			if s%7 == 0 {
				walk += r * 0.06
			}
			col := white.WithAlpha(walk * 2.2)
			if walk < 0 {
				col = black.WithAlpha(-walk)
			}
			from := next() * 0.45
			to := from + 0.45 + next()*0.55
			if walk > -0.008 && walk < 0.008 {
				col.A = 0
			}
			c.streaks[i] = aquaStreak{from: from, to: to, stops: []paintengine2d.GradientStop{
				Stop(0, col.WithAlpha(0)), Stop(0.25, col), Stop(0.75, col), Stop(1, col.WithAlpha(0))}}
		}
	}
	return c
}

// ---- painting helpers -----------------------------------------------------------

// aquaU is one design pixel at the look's scale (never under a device pixel).
func aquaU(l *Classic) float32 {
	u := l.S(1)
	if u < 1 {
		u = 1
	}
	return u
}

// aquaGelPaint paints gel g as a lozenge of corner radius r in b: outline,
// body ramp, darkened ends, gloss over the upper half. A vertical gel
// (scroll thumb, stepper) is the same material turned 90°: lit from the
// left, glowing on the right.
func aquaGelPaint(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32, g *aquaGel, vertical bool) {
	if ctx == nil || g == nil || b.Dx() < 3 || b.Dy() < 3 {
		return
	}
	across := func(rr paintengine2d.Rect, stops []paintengine2d.GradientStop) paintengine2d.Paint {
		if vertical {
			return HGradient(rr, stops...)
		}
		return VGradient(rr, stops...)
	}
	ctx.DrawRoundRect(b, r, r, across(b, g.edge))
	in := b.Inset(u)
	if in.Empty() {
		return
	}
	ri := r - u
	if ri < 0 {
		ri = 0
	}
	ctx.DrawRoundRect(in, ri, ri, across(in, g.body))
	long, short := in.Dx(), in.Dy()
	if vertical {
		long, short = short, long
	}
	fade := short * 0.62
	capW := fade
	if capW < ri*2 {
		capW = ri * 2
	}
	if g.rim.A > 0 {
		if capW*2 <= long {
			// Each end darkens over about a radius: a clamped gradient on a
			// lozenge that shares the body's end corners.
			if vertical {
				t := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), capW)
				ctx.DrawRoundRect(t, ri, ri, paintengine2d.Linear(paintengine2d.LinearGradient{
					Start: paintengine2d.Pt(0, in.Min.Y), End: paintengine2d.Pt(0, in.Min.Y+fade), Stops: g.caps}))
				bt := paintengine2d.XYWH(in.Min.X, in.Max.Y-capW, in.Dx(), capW)
				ctx.DrawRoundRect(bt, ri, ri, paintengine2d.Linear(paintengine2d.LinearGradient{
					Start: paintengine2d.Pt(0, in.Max.Y), End: paintengine2d.Pt(0, in.Max.Y-fade), Stops: g.caps}))
			} else {
				lt := paintengine2d.XYWH(in.Min.X, in.Min.Y, capW, in.Dy())
				ctx.DrawRoundRect(lt, ri, ri, paintengine2d.Linear(paintengine2d.LinearGradient{
					Start: paintengine2d.Pt(in.Min.X, 0), End: paintengine2d.Pt(in.Min.X+fade, 0), Stops: g.caps}))
				rt := paintengine2d.XYWH(in.Max.X-capW, in.Min.Y, capW, in.Dy())
				ctx.DrawRoundRect(rt, ri, ri, paintengine2d.Linear(paintengine2d.LinearGradient{
					Start: paintengine2d.Pt(in.Max.X, 0), End: paintengine2d.Pt(in.Max.X-fade, 0), Stops: g.caps}))
			}
		} else {
			// Beads and small squares darken all round.
			rad := in.Dx()
			if in.Dy() > rad {
				rad = in.Dy()
			}
			ctx.DrawRoundRect(in, ri, ri, paintengine2d.Radial(paintengine2d.RadialGradient{
				Center: in.Center(), Radius: rad * 0.5, Stops: g.ring}))
		}
	}
	if g.glossTop > 0 {
		gl := in.Inset(u)
		rg := ri - u
		if rg < 0 {
			rg = 0
		}
		if !gl.Empty() {
			ctx.DrawRoundRect(gl, rg, rg, across(gl, g.gloss))
		}
	}
}

// aquaDrop is the soft shadow under a gel (the caller leaves 2u below b).
func aquaDrop(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u, a float32) {
	if a <= 0 {
		return
	}
	black := paintengine2d.RGB(0, 0, 0)
	ctx.DrawRoundRect(b.Translate(paintengine2d.Pt(0, u*1.6)), r, r, paintengine2d.Fill(black.WithAlpha(a*0.35)))
	ctx.DrawRoundRect(b.Translate(paintengine2d.Pt(0, u*0.8)), r, r, paintengine2d.Fill(black.WithAlpha(a*0.6)))
}

// aquaGlow is the Aqua keyboard-focus glow: three fading rings hugging the
// outside of shape b (or its inside when inside is set), clipped to clip.
func aquaGlow(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32, col paintengine2d.Color, inside bool, clip paintengine2d.Rect) {
	ctx.Save()
	ctx.ClipRect(clip)
	alphas := [3]float32{0.95, 0.55, 0.22}
	for i, a := range alphas {
		o := u * (float32(i) + 0.5)
		rr, rad := b.Inset(-o), r+o
		if inside {
			rr, rad = b.Inset(o), r-o
			if rad < 0 {
				rad = 0
			}
		}
		if rr.Empty() {
			break
		}
		ctx.DrawRoundRect(rr, rad, rad, paintengine2d.StrokePaint(col.WithAlpha(col.A*a), u))
	}
	ctx.Restore()
}

// aquaAnchor is the user-space offset that puts a texture on the device
// grid, so pinstripes continue across neighbouring panels.
func aquaAnchor(ctx *paintengine2d.Context) float32 {
	m := ctx.Matrix()
	if m.D == 0 {
		return 0
	}
	return m.F / m.D
}

// aquaStripes fills b with base plus the two pinstripe lines of a period
// in design px (lo on the first row of each period, hi half a period
// later), through the shared Pinstripes helper.
func aquaStripes(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, base, lo, hi paintengine2d.Color, period float32) {
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(base))
	if period < 2 || (colorUnset(lo) && colorUnset(hi)) {
		return
	}
	u := aquaU(l)
	per := l.S(period)
	off := aquaAnchor(ctx)
	y0 := float32(math.Floor(float64((b.Min.Y+off)/per)))*per - off
	ctx.Save()
	ctx.ClipRect(b)
	if !colorUnset(lo) {
		Pinstripes(ctx, paintengine2d.XYWH(b.Min.X, y0, b.Dx(), b.Max.Y-y0), lo, per, u)
	}
	if !colorUnset(hi) {
		h := per * 0.5
		Pinstripes(ctx, paintengine2d.XYWH(b.Min.X, y0+h, b.Dx(), b.Max.Y-y0-h), hi, per, u)
	}
	ctx.Restore()
}

// texture paints the window background: pinstripes, brushed metal or flat.
func (c *aqua) texture(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if b.Empty() {
		return
	}
	if !c.metal {
		aquaStripes(l, ctx, b, c.win, c.stripeLo, c.stripeHi, c.period)
		return
	}
	// Brushed metal: a light band down the middle and fine horizontal
	// streaks, one design pixel tall, anchored to the device grid.
	ctx.DrawRect(b, HGradient(b, c.metalStops...))
	u := aquaU(l)
	off := aquaAnchor(ctx)
	first := int(math.Floor(float64((b.Min.Y + off) / u)))
	n := len(c.streaks)
	w := b.Dx()
	// Rows that share a streak share its span and ramp: gather each
	// streak's rows into one path and fill it once (61 draws per texture,
	// not one per pixel row).
	for k := 0; k < n; k++ {
		st := &c.streaks[k]
		if st.stops[1].Color.A == 0 {
			continue
		}
		x0, x1 := b.Min.X+st.from*w, b.Min.X+st.to*w
		if x1 > b.Max.X {
			x1 = b.Max.X
		}
		if x1 <= x0 {
			continue
		}
		// The first row at or after first whose streak is k.
		row := first + ((k-first)%n+n)%n
		var path *paintengine2d.Path
		for y := float32(row)*u - off; y < b.Max.Y; y += float32(n) * u {
			y0, y1 := y, y+u
			if y0 < b.Min.Y {
				y0 = b.Min.Y
			}
			if y1 > b.Max.Y {
				y1 = b.Max.Y
			}
			if y1 <= y0 {
				continue
			}
			if path == nil {
				path = paintengine2d.NewPath()
			}
			path.AddRect(paintengine2d.XYWH(x0, y0, x1-x0, y1-y0))
		}
		if path != nil {
			ctx.DrawPath(path, paintengine2d.Linear(paintengine2d.LinearGradient{
				Start: paintengine2d.Pt(b.Min.X+st.from*w, b.Min.Y), End: paintengine2d.Pt(b.Min.X+st.to*w, b.Min.Y), Stops: st.stops}))
		}
	}
}

// rowStripe washes an unselected odd row with the list stripe.
func (c *aqua) rowStripe(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if st.Alternate() && !st.Checked() && c.stripe.A > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.stripe))
	}
}

// pane is the Aqua group box / tab pane: a rounded, slightly darker wash
// over the window texture with a darker top edge and an inner top shadow.
func (c *aqua) pane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	u := aquaU(l)
	r := l.S(5)
	c.texture(l, ctx, b)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.box))
	black := paintengine2d.RGB(0, 0, 0)
	// Inner top shadow: two fading rows under the top edge, clear of the
	// rounded corners.
	k := r * 0.6
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+k, b.Min.Y+u, b.Dx()-k*2, u), paintengine2d.Fill(black.WithAlpha(0.07)))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u*2, b.Dx()-u*2, u), paintengine2d.Fill(black.WithAlpha(0.03)))
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: b.Min, End: paintengine2d.Pt(b.Min.X, b.Max.Y), Stops: c.paneEdge},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: u, MiterLimit: 4}})
}

// well paints an Aqua text-field frame: white field, dark top edge with an
// inner shadow, light sides and bottom.
func (c *aqua) well(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, disabled bool) {
	u := aquaU(l)
	fill := c.field
	if disabled {
		fill = Mix(c.field, c.win, 0.5)
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, u, b.Dy()), paintengine2d.Fill(c.fieldSide))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y, u, b.Dy()), paintengine2d.Fill(c.fieldSide))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.fieldBot))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.fieldTop))
	black := paintengine2d.RGB(0, 0, 0)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-u*2, u), paintengine2d.Fill(black.WithAlpha(0.16)))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u*2, b.Dx()-u*2, u), paintengine2d.Fill(black.WithAlpha(0.05)))
}

// aquaPillRect is where a push button / popup draws its gel inside a
// control rect: room for the focus glow at the sides and the shadow below.
func aquaPillRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	mx, top, bot := l.S(3), l.S(2), l.S(4)
	h := b.Dy() - top - bot
	if max := l.S(26); h > max {
		// Tall rects keep the pill at its natural height, centred.
		extra := h - max
		top += extra * 0.5
		h = max
	}
	if h < l.S(12) {
		top, h = 0, b.Dy()-l.S(2)
	}
	return paintengine2d.XYWH(b.Min.X+mx, snap(b.Min.Y+top), b.Dx()-mx*2, snap(h))
}

// checkGel is the material of a check box or radio bead in state st.
func (c *aqua) checkGel(st ControlState, on bool) *aquaGel {
	switch {
	case st.Disabled():
		return &c.off
	case st.Pressed():
		return &c.press
	case on:
		return &c.accent
	case st.Hovered():
		return &c.clearHot
	}
	return &c.clear
}

// gelFor picks the push-button material for a state.
func (c *aqua) gelFor(st ControlState) *aquaGel {
	switch {
	case st.Disabled():
		return &c.off
	case st.Pressed():
		return &c.press
	case st.Primary() && st.Backdrop():
		// In an inactive window the default button is plain clear gel.
		return &c.clear
	case st.Primary() && st.Hovered():
		return &c.accentHot
	case st.Primary():
		return &c.accent
	case st.Hovered():
		return &c.clearHot
	}
	return &c.clear
}

// ---- parts --------------------------------------------------------------------

func (e aquaEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := aquaColors(l)
	u := aquaU(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	switch role {
	case RoleButton, RoleCombo:
		g := c.gelFor(st)
		aquaGelPaint(ctx, b, b.Dy()*0.5, u, g, false)
		return g.fg
	case RoleTool:
		black := paintengine2d.RGB(0, 0, 0)
		r := l.S(5)
		switch {
		case st.Disabled():
		case st.Pressed():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(black.WithAlpha(0.2)))
		case st.Checked():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(black.WithAlpha(0.13)))
			ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.StrokePaint(black.WithAlpha(0.22), u))
		case st.Hovered():
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(black.WithAlpha(0.07)))
		}
		return fg
	case RoleField:
		c.well(l, ctx, b, st.Disabled())
		return fg
	case RoleCheck:
		g := c.checkGel(st, st.Checked())
		aquaGelPaint(ctx, b, l.S(3), u, g, false)
		return g.fg
	case RoleRow:
		if st.Checked() && st.Inactive() {
			ctx.DrawRect(b, paintengine2d.Fill(c.selOff))
			return c.selOffTxt
		}
		if st.Checked() {
			ctx.DrawRect(b, paintengine2d.Fill(c.sel))
			return c.selTxt
		}
		if st.Hovered() && !st.Disabled() {
			ctx.DrawRect(b, paintengine2d.Fill(c.hover))
		}
		return fg
	case RoleMenu:
		if !st.Disabled() && (st.Hovered() || st.Pressed()) {
			e.MenuHighlight(l, ctx, b, false)
			return c.menuHiTxt
		}
		return c.menuTxt
	case RoleThumb:
		g := &c.accent
		if st.Pressed() {
			g = &c.press
		}
		v := b.Dy() > b.Dx()
		r := b.Dx() * 0.5
		if !v {
			r = b.Dy() * 0.5
		}
		aquaGelPaint(ctx, b, r, u, g, v)
		return g.fg
	case RoleTrack:
		v := b.Dy() > b.Dx()
		r := b.Dx() * 0.5
		if !v {
			r = b.Dy() * 0.5
		}
		if v {
			ctx.DrawRoundRect(b, r, r, HGradient(b, c.trackStops...))
		} else {
			ctx.DrawRoundRect(b, r, r, VGradient(b, c.trackStops...))
		}
		return fg
	case RoleTab:
		return fg // DrawTab paints the segment
	case RoleSplitter:
		c.texture(l, ctx, b)
		return fg
	case RoleBar:
		aquaStripes(l, ctx, b, c.barHi, c.barLo, paintengine2d.Color{}, c.period)
		return fg
	case RolePanel:
		c.texture(l, ctx, b)
		return fg
	}
	return fg
}

func (e aquaEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := aquaColors(l)
	u := aquaU(l)
	if checked {
		st |= StateChecked
	}
	box = paintengine2d.XYWH(snap(box.Min.X), snap(box.Min.Y), snap(box.Dx()), snap(box.Dx()))
	g := c.checkGel(st, checked)
	aquaDrop(ctx, box, l.S(3), u, g.shadowA*0.8)
	aquaGelPaint(ctx, box, l.S(3), u, g, false)
	if !checked {
		return
	}
	// The Aqua tick: black, bold, and it overshoots the top right corner.
	s := box.Dx()
	tick := paintengine2d.NewPath()
	tick.MoveTo(box.Min.X+s*0.2, box.Min.Y+s*0.5)
	tick.LineTo(box.Min.X+s*0.44, box.Min.Y+s*0.78)
	tick.LineTo(box.Min.X+s*1.02, box.Min.Y-s*0.12)
	col := paintengine2d.RGB(0, 0, 0)
	if c.dark {
		col = c.text
	}
	if st.Disabled() {
		col = c.off.glyph
	}
	ctx.DrawPath(tick, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: s * 0.17, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 6}})
}

func (e aquaEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := aquaColors(l)
	u := aquaU(l)
	d := box.Dx()
	if box.Dy() < d {
		d = box.Dy()
	}
	d = snap(d)
	bead := paintengine2d.XYWH(snap(box.Min.X+(box.Dx()-d)*0.5), snap(box.Min.Y+(box.Dy()-d)*0.5), d, d)
	g := c.checkGel(st, selected)
	aquaDrop(ctx, bead, d*0.5, u, g.shadowA*0.8)
	aquaGelPaint(ctx, bead, d*0.5, u, g, false)
	if selected {
		dot := paintengine2d.RGB(0, 0, 0)
		if c.dark {
			dot = c.text
		}
		if st.Disabled() {
			dot = c.off.glyph
		}
		ctx.DrawCircle(bead.Center(), d*0.17, paintengine2d.Fill(dot))
	}
}

func (aquaEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	// Aqua arrows are small solid triangles: 7×4 in a 15px button.
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	w := s * 0.52
	if w > l.S(8) {
		w = l.S(8)
	}
	if w < l.S(4) {
		w = l.S(4)
	}
	aquaTri(ctx, (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5, w, w*0.6, dir, col)
}

// aquaTri fills an isosceles triangle of base w and height h centred on
// (cx, cy), pointing dir.
func aquaTri(ctx *paintengine2d.Context, cx, cy, w, h float32, dir Direction, col paintengine2d.Color) {
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

// Expander is the Aqua disclosure triangle: grey, pointing right or down.
func (aquaEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	s := l.S(9)
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	p := paintengine2d.NewPath()
	if expanded {
		h := s * 0.8
		p.MoveTo(cx-s*0.5, cy-h*0.5)
		p.LineTo(cx+s*0.5, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
	} else {
		w := s * 0.8
		p.MoveTo(cx-w*0.5, cy-s*0.5)
		p.LineTo(cx+w*0.5, cy)
		p.LineTo(cx-w*0.5, cy+s*0.5)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// MenuHighlight is the Aqua menu selection: a blue band with the same
// pinstripes as the menu under it.
func (aquaEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := aquaColors(l)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, VGradient(b, c.menuHiStops...))
	if c.period >= 2 {
		u := aquaU(l)
		per := l.S(c.period)
		off := aquaAnchor(ctx)
		y0 := float32(math.Floor(float64((b.Min.Y+off)/per)))*per - off
		ctx.Save()
		ctx.ClipRect(b)
		Pinstripes(ctx, paintengine2d.XYWH(b.Min.X, y0, b.Dx(), b.Max.Y-y0), c.menuHiLo, per, u)
		Pinstripes(ctx, paintengine2d.XYWH(b.Min.X, y0+per*0.5, b.Dx(), b.Max.Y-y0-per*0.5), c.menuHiHi, per, u)
		ctx.Restore()
	}
}

func (aquaEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := aquaColors(l)
	if hot {
		return c.menuHiTxt
	}
	return c.menuTxt
}

// Fields show the blue focus glow.
func (aquaEngine) FieldFocusRing(l *Classic) bool { return true }

// ---- scrollbars -------------------------------------------------------------

func (aquaEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 15, Arrows: ArrowsTogetherEnd, MinThumb: 26}
}

func (e aquaEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := aquaColors(l)
	u := aquaU(l)
	if p.Bar.Empty() {
		return
	}
	// Gutter with a hairline against the content.
	ctx.DrawRect(p.Bar, paintengine2d.Fill(c.gutter))
	if vertical {
		ctx.DrawRect(paintengine2d.XYWH(p.Bar.Min.X, p.Bar.Min.Y, u, p.Bar.Dy()), paintengine2d.Fill(c.trackEdge))
	} else {
		ctx.DrawRect(paintengine2d.XYWH(p.Bar.Min.X, p.Bar.Min.Y, p.Bar.Dx(), u), paintengine2d.Fill(c.trackEdge))
	}
	body := p.Bar
	if vertical {
		body.Min.X += u
	} else {
		body.Min.Y += u
	}
	// The groove: a channel with round ends, shaded across the bar.
	if !p.Track.Empty() {
		g := p.Track
		if vertical {
			g = paintengine2d.XYWH(body.Min.X, g.Min.Y, body.Dx(), g.Dy())
		} else {
			g = paintengine2d.XYWH(g.Min.X, body.Min.Y, g.Dx(), body.Dy())
		}
		e.Face(l, ctx, g, RoleTrack, StateNone)
		if st.Pressed == ScrollPageDec || st.Pressed == ScrollPageInc {
			r := g.Dx() * 0.5
			if !vertical {
				r = g.Dy() * 0.5
			}
			ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.06)))
		}
	}
	// Both arrows together at the end, divided by a hairline.
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		ab := b
		if vertical {
			ab.Min.X += u
		} else {
			ab.Min.Y += u
		}
		if vertical {
			ctx.DrawRect(ab, HGradient(ab, c.clear.body...))
		} else {
			ctx.DrawRect(ab, VGradient(ab, c.clear.body...))
		}
		if st.Pressed == part && !st.Disabled {
			ctx.DrawRect(ab, paintengine2d.Fill(c.press.mid.WithAlpha(0.45)))
		}
		col := c.glyph
		if st.Disabled {
			col = c.off.glyph
		}
		e.Arrow(l, ctx, ab, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec)
	arrow(p.Inc, inc, ScrollInc)
	if !p.Dec.Empty() && !p.Inc.Empty() {
		if vertical {
			ctx.DrawRect(paintengine2d.XYWH(body.Min.X, snap(p.Inc.Min.Y), body.Dx(), u), paintengine2d.Fill(c.trackEdge))
			ctx.DrawRect(paintengine2d.XYWH(body.Min.X, snap(p.Dec.Min.Y), body.Dx(), u), paintengine2d.Fill(c.trackEdge.WithAlpha(0.5)))
		} else {
			ctx.DrawRect(paintengine2d.XYWH(snap(p.Inc.Min.X), body.Min.Y, u, body.Dy()), paintengine2d.Fill(c.trackEdge))
			ctx.DrawRect(paintengine2d.XYWH(snap(p.Dec.Min.X), body.Min.Y, u, body.Dy()), paintengine2d.Fill(c.trackEdge.WithAlpha(0.5)))
		}
	}
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	th := p.Thumb
	if vertical {
		th = paintengine2d.XYWH(body.Min.X+u, th.Min.Y+u, body.Dx()-u*2, th.Dy()-u*2)
	} else {
		th = paintengine2d.XYWH(th.Min.X+u, body.Min.Y+u, th.Dx()-u*2, body.Dy()-u*2)
	}
	ts := StateNone
	if st.Pressed == ScrollThumbPart {
		ts |= StatePressed
	}
	e.Face(l, ctx, th, RoleThumb, ts)
}

// DrawScrollBar paints a bare track + thumb (widgets that lay out their own).
func (e aquaEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	vertical := track.Dy() >= track.Dx()
	ss := ScrollState{}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	if st.Hovered() {
		ss.Hovered = true
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, vertical, ss)
}

// ---- frames -----------------------------------------------------------------

func (aquaEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top += l.body.Height() + l.S(4)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is the Aqua box: the title sits above a rounded, slightly
// darker translucent pane. A raised box is a sheet: window texture, a soft
// outline and the title as a heading.
func (e aquaEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := aquaColors(l)
	u := aquaU(l)
	f := l.body
	if raised {
		e.DrawPanel(l, ctx, b, true)
		if title != "" {
			bf := l.BoldFont()
			l.drawFittedText(ctx, bf, title, paintengine2d.XYWH(b.Min.X+l.metrics.Pad, b.Min.Y+u, b.Dx()-l.metrics.Pad*2, f.Height()+l.S(6)), c.text, AlignStart, 0)
		}
		return
	}
	top := b.Min.Y
	if title != "" {
		top += f.Height() + l.S(2)
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8), f.Height()), c.text, AlignStart, 0)
	}
	c.pane(l, ctx, paintengine2d.XYWH(b.Min.X, snap(top), b.Dx(), b.Max.Y-snap(top)))
}

// aquaTitleH is the in-app window's title bar height.
func aquaTitleH(l *Classic) float32 {
	h := l.body.Height() + l.S(6)
	if h < l.S(22) {
		h = l.S(22)
	}
	return snap(h)
}

func (aquaEngine) WindowFrameInsets(l *Classic) Insets {
	u := aquaU(l)
	return Insets{Top: u + aquaTitleH(l), Right: u, Bottom: u, Left: u}
}

// aquaLight is the rect of traffic light i (0 close, 1 minimise, 2 zoom).
func aquaLight(l *Classic, b paintengine2d.Rect, i int) paintengine2d.Rect {
	u := aquaU(l)
	bar := aquaTitleH(l)
	d := snap(l.S(13))
	if d > bar-u*4 {
		d = bar - u*4
	}
	x := b.Min.X + l.S(8) + float32(i)*l.S(20)
	y := b.Min.Y + u + (bar-d)*0.5
	return paintengine2d.XYWH(snap(x), snap(y), d, d)
}

// WindowCloseRect is the red traffic light, first on the left.
// DrawWindowBackground puts the pinstripes (or brushed metal) behind the
// whole window, anchored to the device grid so partial redraws line up.
func (aquaEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	aquaColors(l).texture(l, ctx, b)
}

// PopupShadow: Aqua floats everything on big soft shadows — menus, help
// tags and, deepest of all, windows and sheets.
func (aquaEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	sp := aquaShadow(l, kind)
	return ShadowReach(0, sp.dy, sp.blur, 0)
}

func (aquaEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	sp := aquaShadow(l, kind)
	DropShadow(ctx, b, sp.r, sp.col, 0, sp.dy, sp.blur, 0)
}

func aquaShadow(l *Classic, kind PopupKind) baseShadowSpec {
	switch kind {
	case PopupTooltip:
		return baseShadowSpec{col: paintengine2d.RGBA(0, 0, 0, 0.3), dy: l.S(2), blur: l.S(6)}
	case PopupDialog:
		return baseShadowSpec{col: paintengine2d.RGBA(0, 0, 0, 0.5), r: l.S(5), dy: l.S(10), blur: l.S(30)}
	}
	return baseShadowSpec{col: paintengine2d.RGBA(0, 0, 0, 0.4), r: l.S(5), dy: l.S(5), blur: l.S(16)}
}

// ItemFocus: none — a focused list is ringed as a whole (its view frame
// draws the look's focus ring), the Mac way.
func (aquaEngine) ItemFocus(*Classic, *paintengine2d.Context, paintengine2d.Rect, ControlState) {}

func (aquaEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return aquaLight(l, b, 0)
}

// DrawWindowFrame is an Aqua window: rounded top corners, a light metal
// title bar with pinstripes, the traffic lights on the left and a centred
// title over the window's pinstripes (or brushed metal, seamless).
func (e aquaEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := aquaColors(l)
	u := aquaU(l)
	r := l.S(5)
	barH := aquaTitleH(l)
	outline := Mix(c.titleEdge, c.win, 0.2)
	path := RoundRectPath(b, r, r, 0, 0)
	ctx.DrawPath(path, paintengine2d.Fill(outline))
	in := paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-u*2, b.Dy()-u*2)
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), barH)
	body := paintengine2d.XYWH(in.Min.X, bar.Max.Y, in.Dx(), in.Max.Y-bar.Max.Y)
	// Only the title bar needs the rounded clip; the body is a plain rect.
	ctx.Save()
	ctx.ClipPath(RoundRectPath(bar, r-u, r-u, 0, 0))
	if c.metal {
		c.texture(l, ctx, bar)
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), u), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.55)))
	} else {
		ctx.DrawRect(bar, VGradient(bar, c.titleStops...))
		aquaStripes(l, ctx, bar, paintengine2d.RGBA(1, 1, 1, 0), c.titleLo, c.titleHi, c.period)
		ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y-u, bar.Dx(), u), paintengine2d.Fill(c.titleEdge))
	}
	ctx.Restore()
	c.texture(l, ctx, body)
	// Traffic lights: close, minimise, zoom.
	for i := 0; i < 3; i++ {
		lb := aquaLight(l, b, i)
		g := &c.lights[i]
		if !st.Active || (i == 0 && !st.CanClose) {
			g = &c.lightOff
		}
		if i == 0 && st.ClosePress && st.CanClose && st.Active {
			g = &c.lightPress
		}
		aquaDrop(ctx, lb, lb.Dx()*0.5, u, 0.18)
		aquaGelPaint(ctx, lb, lb.Dx()*0.5, u, g, false)
		if i == 0 && st.CanClose && (st.CloseHot || st.ClosePress) {
			DrawCross(ctx, lb.Inset(lb.Dx()*0.31), g.fg, l.S(1.5))
		}
	}
	if title == "" {
		return
	}
	col := c.titleTxt
	if !st.Active {
		col = c.titleOff
	}
	left := aquaLight(l, b, 2).Max.X + l.S(8)
	f := l.body
	tw := f.Advance(title)
	// Centred over the whole bar, pushed right only when it would hit the lights.
	x := bar.Min.X + (bar.Dx()-tw)*0.5
	if x < left {
		x = left
	}
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, bar.Min.Y, bar.Max.X-l.S(8)-x, bar.Dy()-u), col, AlignStart, 0)
}

// ---- controls ---------------------------------------------------------------

// DrawPanel: a plain panel is the Aqua box; a raised one is a sheet of
// window texture with a soft outline.
func (e aquaEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := aquaColors(l)
	if !raised {
		c.pane(l, ctx, b)
		return
	}
	u := aquaU(l)
	c.texture(l, ctx, b)
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(Mix(c.titleEdge, c.win, 0.25), u))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-u*2, u), paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.45)))
}

func (e aquaEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := aquaColors(l)
	u := aquaU(l)
	pill := aquaPillRect(l, b)
	g := c.gelFor(st)
	r := pill.Dy() * 0.5
	aquaDrop(ctx, pill, r, u, g.shadowA)
	aquaGelPaint(ctx, pill, r, u, g, false)
	if st.Focused() && !st.Disabled() {
		aquaGlow(ctx, pill, r, u, c.focus, false, b)
	}
	l.drawFittedText(ctx, l.body, label, pill.Translate(paintengine2d.Pt(0, -u*0.5)), g.fg, AlignCenter, r)
}

// DrawComboBox is the Aqua popup button: a white gel with a blue gel cap
// on the right holding the double arrows.
func (e aquaEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := aquaColors(l)
	u := aquaU(l)
	pill := aquaPillRect(l, b)
	r := pill.Dy() * 0.3
	g := &c.clear
	cap := &c.accent
	switch {
	case st.Disabled():
		g, cap = &c.off, &c.off
	case open || st.Pressed():
		g, cap = &c.clearHot, &c.press
	case st.Hovered():
		g, cap = &c.clearHot, &c.accentHot
	}
	aquaDrop(ctx, pill, r, u, g.shadowA)
	aquaGelPaint(ctx, pill, r, u, g, false)
	capW := snap(pill.Dy() * 0.85)
	cb := paintengine2d.XYWH(pill.Max.X-capW, pill.Min.Y, capW, pill.Dy())
	ctx.Save()
	ctx.ClipRect(cb)
	aquaGelPaint(ctx, pill, r, u, cap, false)
	ctx.Restore()
	ctx.DrawRect(paintengine2d.XYWH(cb.Min.X, pill.Min.Y+u, u, pill.Dy()-u*2), paintengine2d.Fill(cap.edgeTop.WithAlpha(0.7)))
	// Double arrows: a small up and a small down triangle.
	aw, ah := l.S(7), l.S(4)
	cx, cy := cb.Min.X+capW*0.5, pill.Min.Y+pill.Dy()*0.5-u*0.5
	gap := l.S(1.5)
	aquaTri(ctx, cx, cy-gap-ah*0.5, aw, ah, DirUp, cap.glyph)
	aquaTri(ctx, cx, cy+gap+ah*0.5, aw, ah, DirDown, cap.glyph)
	if st.Focused() && !st.Disabled() && !open {
		aquaGlow(ctx, pill, r, u, c.focus, false, b)
	}
	fg := g.fg
	f := l.body
	if text == "" || st.Disabled() {
		fg = c.dim
	}
	l.drawFittedText(ctx, f, text, paintengine2d.XYWH(pill.Min.X+l.S(9), pill.Min.Y, cb.Min.X-pill.Min.X-l.S(12), pill.Dy()-u), fg, AlignStart, 0)
}

// DrawSpinner is the Aqua stepper: a small white gel capsule split into an
// up and a down half; the held half turns blue.
func (e aquaEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := aquaColors(l)
	u := aquaU(l)
	w := b.Dx() - l.S(2)
	if w > l.S(15) {
		w = l.S(15)
	}
	h := b.Dy() - l.S(3)
	if h > l.S(26) {
		h = l.S(26)
	}
	cap := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-w)*0.5), snap(b.Min.Y+(b.Dy()-h-u)*0.5), snap(w), snap(h))
	r := cap.Dx() * 0.5
	g := &c.clear
	if st.Disabled() {
		g = &c.off
	}
	aquaDrop(ctx, cap, r, u, g.shadowA)
	aquaGelPaint(ctx, cap, r, u, g, false)
	mid := snap(cap.Min.Y + cap.Dy()*0.5)
	half := func(top bool, press, hover bool) {
		hb := paintengine2d.XYWH(cap.Min.X, cap.Min.Y, cap.Dx(), mid-cap.Min.Y)
		dir := DirUp
		if !top {
			hb = paintengine2d.XYWH(cap.Min.X, mid, cap.Dx(), cap.Max.Y-mid)
			dir = DirDown
		}
		glyph := g.glyph
		if !st.Disabled() && (press || hover) {
			hg := &c.clearHot
			if press {
				hg = &c.press
			}
			ctx.Save()
			ctx.ClipRect(hb)
			aquaGelPaint(ctx, cap, r, u, hg, false)
			ctx.Restore()
			glyph = hg.glyph
		}
		a := cap.Dx() * 0.5
		cy := hb.Min.Y + hb.Dy()*0.5
		if top {
			cy += u * 0.5
		} else {
			cy -= u * 0.5
		}
		aquaTri(ctx, hb.Min.X+hb.Dx()*0.5, cy, a, a*0.6, dir, glyph)
	}
	half(true, upPress, upHover)
	half(false, downPress, downHover)
	ctx.DrawRect(paintengine2d.XYWH(cap.Min.X+u, mid, cap.Dx()-u*2, u), paintengine2d.Fill(g.edgeBot.WithAlpha(0.8)))
}

// DrawTabBar is the pane's top edge the segmented tabs straddle: window
// texture above the middle, the rounded box from the middle down.
func (e aquaEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aquaColors(l)
	mid := snap(b.Min.Y + b.Dy()*0.5)
	c.texture(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y))
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid))
	c.pane(l, ctx, paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid+l.S(12)))
	ctx.Restore()
}

// DrawTab is one segment of the Aqua tab control (10.3+): a gel segment
// with softly rounded ends; the selected one is the blue gel.
func (e aquaEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := aquaColors(l)
	u := aquaU(l)
	h := b.Dy() - l.S(6)
	if h > l.S(24) {
		h = l.S(24)
	}
	seg := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-h)*0.5-u*0.5), b.Dx(), snap(h))
	r := l.S(5)
	g := &c.clear
	switch {
	case st.Disabled():
		g = &c.off
		if selected {
			g = &c.accentOff
		}
	case st.Pressed():
		g = &c.press
	case selected:
		g = &c.accent
	case st.Hovered():
		g = &c.clearHot
	}
	aquaDrop(ctx, seg, r, u, g.shadowA*0.7)
	aquaGelPaint(ctx, seg, r, u, g, false)
	fg := g.fg
	if st.Disabled() {
		fg = c.dim
	}
	l.drawFittedText(ctx, l.body, label, seg.Translate(paintengine2d.Pt(0, -u*0.5)), fg, AlignCenter, r*2)
	if st.Focused() && selected {
		aquaGlow(ctx, seg, r, u, c.focus, true, b)
	}
}

func (e aquaEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aquaColors(l)
	u := aquaU(l)
	aquaStripes(l, ctx, b, c.barHi, c.barLo, c.barLine, c.period)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.barEdge))
}

func (e aquaEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := aquaColors(l)
	u := aquaU(l)
	hl := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u)
	fg := c.menuTxt
	switch {
	case st.Disabled():
		fg = c.dim
	case open || st.Pressed():
		e.MenuHighlight(l, ctx, hl, true)
		fg = c.menuHiTxt
	case st.Hovered():
		// Aqua had no hover here; a faint wash keeps the pointer feedback.
		ctx.DrawRect(hl, paintengine2d.Fill(c.sel.WithAlpha(0.12)))
	}
	// Mac menus never underline mnemonics; the keys still work.
	l.drawLabeled(ctx, l.body, label, -1, b, fg)
	if st.Focused() && !open {
		aquaGlow(ctx, hl, 0, u, c.focus, true, b)
	}
}

// DrawMenuFrame is the Aqua menu: slightly translucent white with the
// pinstripes showing, rounded bottom corners and a soft outline.
func (e aquaEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aquaColors(l)
	u := aquaU(l)
	r := l.S(5)
	shape := RoundRectPath(b, 0, 0, r, r)
	ctx.Save()
	ctx.ClipPath(shape)
	ctx.DrawRect(b, paintengine2d.Fill(c.menuBg))
	if c.period >= 2 {
		aquaStripes(l, ctx, b, paintengine2d.RGBA(0, 0, 0, 0), c.menuLo, c.menuLine, c.period)
	}
	ctx.Restore()
	ctx.DrawPath(RoundRectPath(b.Inset(u*0.5), 0, 0, r, r), paintengine2d.StrokePaint(c.menuEdge, u))
}

// DrawMenuItem lays out an Aqua menu row: shortcuts in the label colour
// (white on the highlight), no mnemonic underlines, a hairline separator.
func (e aquaEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := aquaColors(l)
	ch := MenuChromeFor(l)
	u := aquaU(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0 := b.Min.X - ch.PadL + u
		ctx.DrawRect(paintengine2d.XYWH(x0, y, b.Max.X+ch.PadR-u-x0, u), paintengine2d.Fill(c.menuSep))
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := c.menuTxt
	switch {
	case st.Disabled():
		fg = c.dim
	case hot:
		fg = c.menuHiTxt
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	l.drawMenuGutter(ctx, b, ch, row, fg)
	right := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = l.S(10)
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		aquaTri(ctx, ab.Min.X+ab.Dx()*0.5, ab.Min.Y+ab.Dy()*0.5, l.S(8), l.S(5), DirRight, fg)
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
	f.Draw(ctx, row.Label, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// DrawProgressBar: a blue gel bar in a light inset track; the busy bar is
// the barber pole — blue and white gel stripes sliding with phase.
func (e aquaEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := aquaColors(l)
	u := aquaU(l)
	// Room for the drop shadow under the bar, inside b.
	h := b.Dy() - u*3
	if h > l.S(16) {
		h = l.S(16)
	}
	h = float32(int(h))
	bar := paintengine2d.XYWH(b.Min.X, float32(int(b.Min.Y+(b.Dy()-h-u*2)*0.5)), b.Dx(), h)
	r := l.S(4)
	if r > bar.Dy()*0.5 {
		r = bar.Dy() * 0.5
	}
	aquaDrop(ctx, bar, r, u, 0.12)
	// Track: light inset channel.
	ctx.DrawRoundRect(bar, r, r, paintengine2d.Fill(c.fieldSide))
	in := bar.Inset(u)
	ri := r - u
	if ri < 0 {
		ri = 0
	}
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.progTrack...))
	g, pole := &c.accent, c.pole
	if st.Disabled() {
		g, pole = &c.off, c.poleOff
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		// Stripes at 45°: a repeating gradient with hard stops, one draw.
		per := l.S(16)
		sx := in.Min.X + phase*per
		ctx.Save()
		ctx.ClipRoundRect(in, ri, ri)
		ctx.DrawRect(in, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(sx, in.Min.Y), End: paintengine2d.Pt(sx+per*0.5, in.Min.Y+per*0.5),
			Stops: pole,
			Tile:  paintengine2d.TileRepeat,
		}))
		aquaGelShade(ctx, in)
		ctx.Restore()
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	w := in.Dx() * t
	if w < ri*2+u {
		if w <= 0 {
			return
		}
	}
	fill := paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy())
	ctx.Save()
	ctx.ClipRoundRect(in, ri, ri)
	ctx.DrawRect(fill, VGradient(fill, g.body...))
	aquaGelShade(ctx, fill)
	if fill.Max.X < in.Max.X-u {
		ctx.DrawRect(paintengine2d.XYWH(fill.Max.X-u, fill.Min.Y, u, fill.Dy()), paintengine2d.Fill(g.edgeTop.WithAlpha(0.35)))
	}
	ctx.Restore()
}

// aquaGelShade lays the gel's gloss and bottom glow over flat colour
// (progress stripes, fills): white gloss on the top half, a brighter base.
func aquaGelShade(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, VGradient(b, aquaShadeStops...))
}

// Colour-free overlays, allocated once.
var (
	aquaShadeStops = []paintengine2d.GradientStop{
		Stop(0, paintengine2d.RGBA(1, 1, 1, 0.72)), Stop(0.48, paintengine2d.RGBA(1, 1, 1, 0.22)),
		Stop(0.48, paintengine2d.RGBA(0, 0, 0, 0.06)), Stop(0.62, paintengine2d.RGBA(0, 0, 0, 0)),
		Stop(1, paintengine2d.RGBA(1, 1, 1, 0.42)),
	}
	aquaChannelStops = []paintengine2d.GradientStop{
		Stop(0, paintengine2d.RGBA(0, 0, 0, 0.18)), Stop(0.4, paintengine2d.RGBA(0, 0, 0, 0)), Stop(1, paintengine2d.RGBA(0, 0, 0, 0)),
	}
)

// DrawSlider is the Aqua slider: a thin inset groove and a round gel knob.
func (e aquaEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := aquaColors(l)
	u := aquaU(l)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	d := l.metrics.Thumb
	if d <= 0 {
		d = l.S(17)
	}
	if d > b.Dy()-u*3 {
		d = b.Dy() - u*3
	}
	d = snap(d)
	cy := snap(b.Min.Y + b.Dy()*0.5 - u*0.5)
	gh := snap(l.S(5))
	x0, x1 := b.Min.X+d*0.5+l.S(2), b.Max.X-d*0.5-l.S(2)
	groove := paintengine2d.XYWH(b.Min.X+l.S(3), cy-gh*0.5, b.Dx()-l.S(6), gh)
	ctx.DrawRoundRect(groove, gh*0.5, gh*0.5, VGradient(groove, c.grooveStops...))
	ctx.DrawRoundRect(groove.Inset(u*0.5), gh*0.5, gh*0.5, paintengine2d.StrokePaint(c.grooveEdge.WithAlpha(0.55), u))
	kx := snap(x0 + (x1-x0)*t - d*0.5)
	knob := paintengine2d.XYWH(kx, snap(cy-d*0.5), d, d)
	g := &c.accent
	switch {
	case st.Disabled():
		g = &c.off
	case st.Pressed():
		g = &c.press
	case st.Hovered():
		g = &c.accentHot
	}
	aquaDrop(ctx, knob, d*0.5, u, g.shadowA)
	aquaGelPaint(ctx, knob, d*0.5, u, g, false)
	if st.Focused() && !st.Disabled() {
		aquaGlow(ctx, knob, d*0.5, u, c.focus, false, b)
	}
}

// DrawSwitch: Aqua had no switch; it is drawn as a gel capsule groove (blue
// when on) with a white gel bead.
func (e aquaEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := aquaColors(l)
	u := aquaU(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if th > b.Dy()-u*2 {
		th = b.Dy() - u*2
	}
	track := paintengine2d.XYWH(b.Min.X+u, snap(b.Min.Y+(b.Dy()-th)*0.5-u*0.5), snap(tw), snap(th))
	r := track.Dy() * 0.5
	g := &c.clear
	switch {
	case st.Disabled():
		g = &c.off
	case on:
		g = &c.accent
	}
	aquaDrop(ctx, track, r, u, 0.1)
	aquaGelPaint(ctx, track, r, u, g, false)
	// Groove shadow: the capsule reads as a channel.
	ctx.DrawRoundRect(track.Inset(u), r-u, r-u, VGradient(track, aquaChannelStops...))
	kd := track.Dy() - u*2
	kx := track.Min.X + u
	if on {
		kx = track.Max.X - u - kd
	}
	knob := paintengine2d.XYWH(kx, track.Min.Y+u, kd, kd)
	kg := &c.clear
	if st.Disabled() {
		kg = &c.off
	} else if st.Hovered() {
		kg = &c.clearHot
	}
	aquaDrop(ctx, knob, kd*0.5, u, 0.25)
	aquaGelPaint(ctx, knob, kd*0.5, u, kg, false)
	if st.Focused() && !st.Disabled() {
		aquaGlow(ctx, track, r, u, c.focus, false, b)
	}
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.dim
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
}

func (e aquaEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	aquaColors(l).rowStripe(ctx, b, st)
	selected, hovered := st.Checked(), st.Hovered()
	fg := l.fieldText()
	if selected || hovered {
		fg = e.Face(l, ctx, b, RoleRow, st&^StateFocused)
		if !selected {
			fg = l.fieldText()
		}
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy()), fg, AlignStart, 0)
}

func (e aquaEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	aquaColors(l).rowStripe(ctx, b, st)
	selected, hovered := st.Checked(), st.Hovered()
	c := aquaColors(l)
	fg := l.fieldText()
	if selected || hovered {
		fg = e.Face(l, ctx, b, RoleRow, st&^StateFocused)
		if !selected {
			fg = l.fieldText()
		}
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		col := c.disclose
		if selected && !st.Inactive() {
			col = c.selTxt
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, col)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(3)
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

func (e aquaEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	aquaColors(l).rowStripe(ctx, b, st)
	selected, hovered := st.Checked(), st.Hovered()
	fg := l.fieldText()
	if selected || hovered {
		fg = e.Face(l, ctx, b, RoleRow, st&^StateFocused)
		if !selected {
			fg = l.fieldText()
		}
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
	ctx.ClipRect(b.Inset(1))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawTableHeader is the Aqua list header: a light gel strip with hairline
// dividers; the sorted column is the blue gel.
func (e aquaEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := aquaColors(l)
	u := aquaU(l)
	fg := c.text
	switch {
	case st.Pressed() && !st.Disabled():
		ctx.DrawRect(b, VGradient(b, c.press.body...))
		fg = c.press.fg
	case sorted && !st.Disabled():
		ctx.DrawRect(b, VGradient(b, c.headSelStops...))
		fg = c.accent.fg
	default:
		ctx.DrawRect(b, VGradient(b, c.headStops...))
		if st.Hovered() && !st.Disabled() {
			ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.35)))
		}
	}
	if st.Disabled() {
		fg = c.dim
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.headEdge))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+u*2, u, b.Dy()-u*4), paintengine2d.Fill(c.headEdge))
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()-u), fg, AlignStart, 0)
}

// DrawToolBar: toolbars sit on the window's pinstripes over a hairline.
func (e aquaEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := aquaColors(l)
	u := aquaU(l)
	c.texture(l, ctx, b)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.sep))
}

// DrawStatusBar is the Aqua window bottom: texture, a hairline above, the
// parts in small slots, and the three-line grow box at the corner.
func (e aquaEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := aquaColors(l)
	u := aquaU(l)
	c.texture(l, ctx, b)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.sep))
	grip := l.S(15)
	if len(parts) > 0 {
		slot := (b.Dx() - grip) / float32(len(parts))
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			if i > 0 {
				ctx.DrawRect(paintengine2d.XYWH(snap(x), b.Min.Y+l.S(5), u, b.Dy()-l.S(10)), paintengine2d.Fill(c.sep.WithAlpha(0.6)))
			}
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y+u, slot-l.S(14), b.Dy()-u), c.dim, AlignStart, 0)
		}
	}
	// Grow box: three diagonal ridges.
	gx, gy := b.Max.X-l.S(3), b.Max.Y-l.S(3)
	for i := 1; i <= 3; i++ {
		o := l.S(3.5) * float32(i)
		ctx.DrawLine(paintengine2d.Pt(gx-o, gy), paintengine2d.Pt(gx, gy-o), paintengine2d.StrokePaint(c.sep, u))
		ctx.DrawLine(paintengine2d.Pt(gx-o+u, gy), paintengine2d.Pt(gx, gy-o+u), paintengine2d.StrokePaint(c.sepHi, u))
	}
}

// DrawTitleBar is a pane header: the light gel strip of Aqua list headers
// with a bold title and a muted subtitle.
func (e aquaEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := aquaColors(l)
	u := aquaU(l)
	ctx.DrawRect(b, VGradient(b, c.headStops...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.headEdge))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-u), c.dim, AlignStart, 0)
	}
}

// DrawAccordionHeader is an Aqua disclosure row: triangle, label and a
// hairline, on the window.
func (e aquaEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := aquaColors(l)
	u := aquaU(l)
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.04)))
	}
	col := c.disclose
	if st.Pressed() {
		col = c.text
	}
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(14), b.Dy()), expanded, col)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy()), fg, AlignStart, 0)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.sep.WithAlpha(0.7)))
	if st.Focused() {
		aquaGlow(ctx, b, l.S(3), u, c.focus, true, b)
	}
}

func (aquaEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := aquaColors(l)
	u := aquaU(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x-u, b.Min.Y+l.S(2), u, b.Dy()-l.S(4)), paintengine2d.Fill(c.sep))
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+l.S(2), u, b.Dy()-l.S(4)), paintengine2d.Fill(c.sepHi))
		return
	}
	y := snap((b.Min.Y + b.Max.Y) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y-u, b.Dx(), u), paintengine2d.Fill(c.sep))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), paintengine2d.Fill(c.sepHi))
}

// DrawSplitter is the Aqua split-view divider: window texture with the
// round gel dimple in the middle.
func (aquaEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := aquaColors(l)
	u := aquaU(l)
	c.texture(l, ctx, b)
	d := b.Dx()
	if !vertical {
		d = b.Dy()
	}
	d -= u * 2
	if d > l.S(7) {
		d = l.S(7)
	}
	if d < u*3 {
		return
	}
	g := &c.clear
	if st.Hovered() || st.Pressed() {
		g = &c.accent
	}
	dim := paintengine2d.XYWH(snap((b.Min.X+b.Max.X)*0.5-d*0.5), snap((b.Min.Y+b.Max.Y)*0.5-d*0.5), snap(d), snap(d))
	aquaGelPaint(ctx, dim, dim.Dx()*0.5, u, g, false)
}

// DrawTooltip is the Aqua help tag: pale yellow with a thin grey border.
func (aquaEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := aquaColors(l)
	u := aquaU(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.info))
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(c.infoEdge, u))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.infoTxt, AlignStart, 0)
}

// DrawFocusRing is the Aqua focus glow, painted just inside b.
func (aquaEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if b.Dx() < 6 || b.Dy() < 6 {
		return
	}
	c := aquaColors(l)
	aquaGlow(ctx, b, l.S(2), aquaU(l), c.focus, true, b)
}

// ---- packs --------------------------------------------------------------------

func aquaPack(name, label string, year int, summary string, fam ThemeName, pal Palette, extra map[string]string, params map[string]float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "aqua",
		Bevel:   BevelSoftShadow,
		Family:  fam,
		Palette: pal,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Params = params
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Accent, Border: pal.Accent}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.2), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Mac OS", Summary: summary,
		Era: EraAqua, Palette: fam, Tokens: tok,
	}
}

// aquaPalette builds an Aqua palette: window, field, text, list selection
// and text-highlight colours.
func aquaPalette(win, field, text, muted, border, sel, selText, hilite, focus string) Palette {
	w := hexColor(win)
	return Palette{
		Background: w, Surface: w, SurfaceAlt: Shade(w, -0.03),
		Border: hexColor(border), Divider: Mix(hexColor(border), w, 0.45),
		Text: hexColor(text), TextMuted: hexColor(muted), TextOnAccent: hexColor(selText),
		Accent: hexColor(sel), AccentHover: Shade(hexColor(sel), 0.15), AccentPress: Shade(hexColor(sel), -0.2),
		Field: hexColor(field), FieldBorder: hexColor(border),
		Focus: hexColor(focus), Selection: hexColor(hilite),
		Track: Mix(w, hexColor(field), 0.5), Thumb: hexColor(sel),
		Highlight: paintengine2d.RGBA(1, 1, 1, 0.8), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: hexColor(sel), MenuHoverBorder: hexColor(sel), MenuGutter: hexColor(field),
		Danger: hexColor("#d8302a"), Success: hexColor("#2f9a3a"), Warning: hexColor("#d88a12"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: hexColor("#ffffff"), BevelDark: hexColor(border),
	}
}

func aquaPacks() []ThemePack {
	blue := aquaPalette("#eeeeee", "#ffffff", "#000000", "#808080", "#8c8c8c", "#3875d7", "#ffffff", "#b5d5ff", "#5b8fe8")
	graphite := aquaPalette("#eeeeee", "#ffffff", "#000000", "#808080", "#8c8c8c", "#6f7f94", "#ffffff", "#c8d0da", "#7b8aa0")
	metal := aquaPalette("#c9c9c9", "#ffffff", "#000000", "#5e5e5e", "#6e6e6e", "#1b6fd6", "#ffffff", "#b5d5ff", "#4a82e0")
	night := aquaPalette("#2a2a2d", "#1b1b1d", "#e6e6e8", "#8e8e94", "#4c4c52", "#56698a", "#ffffff", "#3f5373", "#6f8fc8")
	night.Danger, night.Success, night.Warning = hexColor("#ff6a60"), hexColor("#62c86c"), hexColor("#f0b040")
	return []ThemePack{
		aquaPack("aqua", "Aqua", 2001, "Mac OS X 10.0–10.4: blue gel pills, pinstripes, traffic lights.", ThemeLight, blue,
			map[string]string{
				"stripeLo": "#e6e6e6", "stripeHi": "#ffffff",
				"gel": "#4d96d7", "gelTop": "#2f63bd", "gelGlow": "#9bd8f7", "gelEdge": "#12207f", "gelRim": "#0a4cc0",
				"press": "#3276d2", "pressTop": "#1d4ba6", "pressGlow": "#7fc2f2", "pressEdge": "#0a1566",
				"clear": "#e5e5e5", "clearTop": "#e8e8e8", "clearGlow": "#ffffff", "clearEdge": "#7c7c7c",
				"listSel": "#3875d7", "menuHi": "#3e74c8", "menuHi2": "#3068bb", "menuBg": "#fafafa",
				"title": "#fdfdfd", "title2": "#d8d8d8", "barLo": "#eaeaea", "barHi": "#fbfbfb",
			}, map[string]float32{"stripe": 4}),
		aquaPack("aqua-graphite", "Aqua Graphite", 2001, "The Graphite appearance: the same gel in blue-grey.", ThemeLight, graphite,
			map[string]string{
				"stripeLo": "#e6e6e6", "stripeHi": "#ffffff",
				"gel": "#8d99a8", "gelTop": "#65717f", "gelGlow": "#d4dbe3", "gelEdge": "#343c47", "gelRim": "#525e6c",
				"press": "#72808f", "pressTop": "#4f5a67", "pressGlow": "#b6c1cd", "pressEdge": "#262c34",
				"clear": "#e5e5e5", "clearTop": "#e8e8e8", "clearGlow": "#ffffff", "clearEdge": "#7c7c7c",
				"listSel": "#6f7f94", "menuHi": "#76869b", "menuHi2": "#66768b", "menuBg": "#fafafa",
				"title": "#fdfdfd", "title2": "#d8d8d8", "barLo": "#eaeaea", "barHi": "#fbfbfb", "glow": "#7b8aa0",
			}, map[string]float32{"stripe": 4}),
		aquaPack("brushed-metal", "Brushed Metal", 2003, "Panther's textured windows: brushed aluminium and darker gels.", ThemeLight, metal,
			map[string]string{
				"metalHi": "#d6d6d6", "metalLo": "#b9b9b9",
				"gel": "#3f86d0", "gelTop": "#2254a8", "gelGlow": "#8cc9f2", "gelEdge": "#0e1a66", "gelRim": "#083d9e",
				"press": "#2c6cc4", "pressTop": "#17428f", "pressGlow": "#72b4ea", "pressEdge": "#08104f",
				"clear": "#cdcdcd", "clearTop": "#d6d6d6", "clearGlow": "#f4f4f4", "clearEdge": "#5a5a5a",
				"listSel": "#1b6fd6", "menuHi": "#3e74c8", "menuHi2": "#3068bb", "menuBg": "#fafafa",
				"barLo": "#eaeaea", "barHi": "#fbfbfb", "box": "#0000001a", "boxEdge": "#6e6e6e",
				"stripeLo": "#00000000",
			}, map[string]float32{"metal": 1, "stripe": 4}),
		aquaPack("aqua-night", "Aqua Graphite Night", 2001, "Not historical: Aqua never had a dark mode — graphite gel on charcoal pinstripes.", ThemeDark, night,
			map[string]string{
				"stripeLo": "#26262a", "stripeHi": "#2f2f33",
				"gel": "#5f7090", "gelTop": "#46546e", "gelGlow": "#90a4c6", "gelEdge": "#11151c", "gelRim": "#26303f",
				"clear": "#56565c", "clearTop": "#626268", "clearGlow": "#7c7c84", "clearEdge": "#131315",
				"press": "#4c5d7c", "pressTop": "#374660", "pressGlow": "#7d93b8",
				"listSel": "#56698a", "menuHi": "#5a6d8e", "menuHi2": "#4a5c7c", "menuBg": "#303034f5",
				"title": "#48484d", "title2": "#2e2e32", "barLo": "#2c2c30", "barHi": "#36363a",
				"box": "#00000033", "boxEdge": "#18181a", "info": "#4a4630", "infoEdge": "#18181a",
				"fieldTop": "#0c0c0e", "fieldSide": "#3a3a3f", "fieldBot": "#434348", "glyph": "#d0d0d4",
			}, map[string]float32{"stripe": 4}),
	}
}

// SpinBoxStyle: Mac OS X: the little arrows stand beside the field.
func (aquaEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{} }
