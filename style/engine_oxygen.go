package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// oxygenEngine paints in the manner of KDE's Oxygen widget style (KDE SC
// 4.0, 2008). The engine is written from Oxygen's visible behaviour and
// published colour values, measured against screenshots: every element is
// a vector recipe over the rect it is given, and the soft shadows and glows
// are stacks of translucent rings (paintengine2d has no blur):
//
//   - the window background is a vertical gradient over the top 300px
//     (a lighter top tone → window → a darker bottom tone) with an
//     elliptical radial light, 600×64, at the top centre;
//   - push buttons, check boxes and tabs are "slabs": a body filled with a
//     light-to-base gradient, a thin light rim, a soft shadow falling half
//     a pixel low, and — on hover or keyboard focus — the blue Oxygen glow
//     hugging the rim (hover #6ed6ff, focus #3aa7dd);
//     a pressed slab sinks: an inner shadow and a light lower contrast edge;
//   - text fields and lists are "holes": the view colour inside an inner
//     shadow that turns into the glow on focus;
//   - scroll bars (no arrows here), progress grooves and slider grooves are
//     dark rounded slots; the scroll handle is a small glossy slab, the
//     slider handle a round slab, the progress bar a glossy highlight bar;
//   - menus are gradient panels with a light top rim; the hot item and the
//     open menu-bar title are flat darker wells;
//     list selections are rounded highlight gradients.
//
// Shades come from the scheme colours through oxTones (see "tones"
// below). Buttons, check boxes and slider handles take the window
// gradient's tint at their height, as Oxygen's do.
//
// Pack data (theme.json "extra"; defaults are the Oxygen colour scheme):
//
//	button, buttonText, view, viewText, tip, tipText, focus, hover
//
// Params: "contrast" (KDE contrast 0–10, default 7).
type oxygenEngine struct{ BaseEngine }

func init() {
	RegisterEngine(oxygenEngine{})
	for _, p := range oxygenPacks() {
		RegisterPack(p)
	}
}

func (oxygenEngine) ID() string { return "oxygen" }

// DefaultMetrics are Oxygen's proportions at the toolkit's 16px UI font:
// 23px check boxes and radios and 21px slider handles (slab and shadow
// included), 7px slider grooves, 16px scroll bars, 4px corners.
func (oxygenEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Radius: 4, RadiusSmall: 3,
		ControlH: 32, FieldH: 30, ComboH: 30,
		Checkbox: 23, Radio: 23,
		MenuItemH: 28, MenuBarH: 30, TabH: 32, RowH: 24,
		TitleBar: 28, HeaderH: 26, ProgressH: 16, SliderH: 24, Thumb: 21,
		Scroll: 16, Pad: 12, FieldPad: 8, FocusWidth: 1, Border: 1,
		ToolBarH: 40, StatusBarH: 26, SpinnerW: 20, SwitchW: 40, SwitchH: 21,
	}
}

// StyleHint: KDE's dialog order ("OK  Cancel": the accepting button first),
// form labels right-aligned against their fields (as KDE's guidelines and
// Oxygen have them) and left-aligned tabs.
func (oxygenEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintDialogPrimaryFirst, HintFormLabelsRight:
		return 1
	}
	return 0
}

// ---- tones ---------------------------------------------------------------------------------
//
// Oxygen paints with a handful of shades of each scheme colour: a light rim
// colour, dark and mid shades for grooves and lines, a deep shadow, and the
// window gradient's top, bottom and radial-light colours. This engine models
// each shade as a move in CIE lightness (L*, 0–1), the moves fitted to the
// shades Oxygen shows for its own colour scheme at its default contrast (7):
//
//	light +0.13    radial light +0.145    gradient top +0.046
//	mid ×0.915     gradient bottom ×0.90  dark ×0.84    shadow ×0.29
//
// A lighter shade blends towards white in linear light (so light rims go
// neutral, as Oxygen's do); a darker one lowers all three channels by the
// same amount, which keeps a colour's hue and channel spread (a warm grey
// stays warm in its grooves and shadows). The "contrast" param stretches
// every move (7 is the default).

// oxTones derives shades at one contrast (k = 1 is Oxygen's default).
type oxTones struct{ k float64 }

// oxLstar is c's CIE lightness on 0..1.
func oxLstar(c paintengine2d.Color) float64 {
	y := RelLuminance(c)
	if y <= 216.0/24389 {
		return y * 24389 / 27 / 100
	}
	return (116*math.Cbrt(y) - 16) / 100
}

// oxLumOf is the relative luminance at lightness l (oxLstar's inverse).
func oxLumOf(l float64) float64 {
	l *= 100
	if l <= 8 {
		return l * 27 / 24389
	}
	f := (l + 16) / 116
	return f * f * f
}

// oxEncode is the sRGB transfer curve: linear light to a channel value.
func oxEncode(v float64) float32 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 1
	case v <= 0.0031308:
		return float32(v * 12.92)
	}
	return float32(1.055*math.Pow(v, 1/2.4) - 0.055)
}

// oxShift adds d to each channel of c (clamped), keeping alpha.
func oxShift(c paintengine2d.Color, d float32) paintengine2d.Color {
	return paintengine2d.RGBA(clamp1(c.R+d), clamp1(c.G+d), clamp1(c.B+d), c.A)
}

// oxToL moves c to lightness l, keeping its alpha.
func oxToL(c paintengine2d.Color, l float64) paintengine2d.Color {
	l = math.Max(0, math.Min(1, l))
	if l >= oxLstar(c) {
		r, g, b := linear(c.R), linear(c.G), linear(c.B)
		y := 0.2126*r + 0.7152*g + 0.0722*b
		if y >= 1 {
			return c
		}
		t := (oxLumOf(l) - y) / (1 - y)
		return paintengine2d.RGBA(oxEncode(r+(1-r)*t), oxEncode(g+(1-g)*t), oxEncode(b+(1-b)*t), c.A)
	}
	// Darker: bisect the (negative) channel shift; lightness grows with it.
	lo, hi := float32(-1), float32(0)
	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		if oxLstar(oxShift(c, mid)) < l {
			lo = mid
		} else {
			hi = mid
		}
	}
	return oxShift(c, (lo+hi)/2)
}

// lift raises c's lightness by d (scaled by the contrast).
func (t oxTones) lift(c paintengine2d.Color, d float64) paintengine2d.Color {
	return oxToL(c, oxLstar(c)+d*t.k)
}

// drop scales c's lightness to the fraction f (stretched by the contrast).
func (t oxTones) drop(c paintengine2d.Color, f float64) paintengine2d.Color {
	return oxToL(c, oxLstar(c)*math.Max(0, 1-(1-f)*t.k))
}

func (t oxTones) light(c paintengine2d.Color) paintengine2d.Color  { return t.lift(c, 0.13) }
func (t oxTones) radial(c paintengine2d.Color) paintengine2d.Color { return t.lift(c, 0.145) }
func (t oxTones) top(c paintengine2d.Color) paintengine2d.Color    { return t.lift(c, 0.046) }
func (t oxTones) mid(c paintengine2d.Color) paintengine2d.Color    { return t.drop(c, 0.915) }
func (t oxTones) bottom(c paintengine2d.Color) paintengine2d.Color { return t.drop(c, 0.90) }
func (t oxTones) dark(c paintengine2d.Color) paintengine2d.Color   { return t.drop(c, 0.84) }
func (t oxTones) shadow(c paintengine2d.Color) paintengine2d.Color { return t.drop(c, 0.29) }

// at is the window gradient's colour at ratio of its height: the top tone,
// the colour itself half way down, the bottom tone at the end.
func (t oxTones) at(c paintengine2d.Color, ratio float64) paintengine2d.Color {
	if ratio < 0.5 {
		return Mix(t.top(c), c, float32(2*ratio))
	}
	return Mix(c, t.bottom(c), float32(2*ratio-1))
}

// ---- resolved colours ------------------------------------------------------------------------

// oxLevels quantises the window-gradient position a slab takes its tint
// from, so every stop slice is built once.
const oxLevels = 12

// oxSlab is one button material: its base, light and dark shades, the
// shadow colour and the prebuilt stop slices.
type oxSlab struct {
	base, light, dark, shadow paintengine2d.Color
	fill, fillSunk, rim       []paintengine2d.GradientStop // body; pressed body; rim
	slabFill                  []paintengine2d.GradientStop // check box body
	fillDef                   []paintengine2d.GradientStop // default-button tint
	knob, knobLine, knobSunk  []paintengine2d.GradientStop // slider handle
	round, roundInner         []paintengine2d.GradientStop // radio bead
	handle, handleGloss       []paintengine2d.GradientStop // scroll handle
	mid                       paintengine2d.Color
}

type oxygen struct {
	h oxTones

	win, text, base, viewText, btn, btnText, hl, hlText paintengine2d.Color
	tip, tipText, focus, hover, dis                     paintengine2d.Color

	top, bottom, radial                  paintengine2d.Color
	winLight, winDark, winShadow, winMid paintengine2d.Color
	bgStops, radialStops                 []paintengine2d.GradientStop
	menuStops                            []paintengine2d.GradientStop

	slabs    [oxLevels]oxSlab // button colour at each gradient level
	winLevel [oxLevels]paintengine2d.Color

	holeShadow, holeTop            []paintengine2d.GradientStop // hole inner shadow rings
	sunkShadow, sunkShadow2        []paintengine2d.GradientStop // pressed slab
	holeLight                      []paintengine2d.GradientStop // HoleContrast / slab lower edge
	flatTop, flatBottom            []paintengine2d.GradientStop // flat well edges
	slotDark, slotLight            paintengine2d.Color          // scroll hole
	slotEdge                       []paintengine2d.GradientStop // scroll hole bottom light
	progFill, progGloss, progBevel []paintengine2d.GradientStop
	progBase, progTop              paintengine2d.Color
	selStops, selHotStops          []paintengine2d.GradientStop
	selOffStops, selDisStops       []paintengine2d.GradientStop
	selHot, hlOff, hlDis           paintengine2d.Color
	focusLine, focusLineSel        []paintengine2d.GradientStop
	tipStops, tipEdge              []paintengine2d.GradientStop
	tabFill                        []paintengine2d.GradientStop
	floatSide                      []paintengine2d.GradientStop
	sepDark, sepLight              []paintengine2d.GradientStop
	tabDark, tabMid                paintengine2d.Color
	menuHot                        paintengine2d.Color
	disField                       paintengine2d.Color
}

type oxygenKey struct{}

func oxygenColors(l *Classic) *oxygen {
	return l.Memo(oxygenKey{}, func() any { return oxygenBuild(l) }).(*oxygen)
}

func oxygenBuild(l *Classic) *oxygen {
	p := l.palette
	h := oxTones{k: float64(l.P("contrast", 7)) / 7}
	c := &oxygen{
		h:    h,
		win:  p.Background,
		text: p.Text,
		base: p.Field,
		hl:   p.Selection,
	}
	if c.hl.A < 0.9 {
		c.hl = p.Accent
	}
	c.hlText = p.TextOnAccent
	c.viewText = l.X("viewText", ReadableOn(c.base, 4.5, c.text))
	c.btn = l.X("button", p.SurfaceAlt)
	c.btnText = l.X("buttonText", c.text)
	c.tip = l.X("tip", Hex("#181513"))
	c.tipText = l.X("tipText", Hex("#e7fdff"))
	c.focus = l.X("focus", Hex("#3aa7dd"))
	c.hover = l.X("hover", Hex("#6ed6ff"))
	c.dis = Mix(c.text, c.win, 0.55)

	c.top, c.bottom, c.radial = h.top(c.win), h.bottom(c.win), h.radial(c.win)
	c.winLight, c.winDark, c.winShadow, c.winMid = h.light(c.win), h.dark(c.win), h.shadow(c.win), h.mid(c.win)
	c.bgStops = []paintengine2d.GradientStop{Stop(0, c.top), Stop(0.5, c.win), Stop(1, c.bottom)}
	rc := c.radial
	c.radialStops = []paintengine2d.GradientStop{
		Stop(0, rc.WithAlpha(1)), Stop(0.5, rc.WithAlpha(101.0/255)), Stop(0.75, rc.WithAlpha(37.0/255)), Stop(1, rc.WithAlpha(0)),
	}
	c.menuStops = c.bgStops
	for i := 0; i < oxLevels; i++ {
		ratio := float64(i) / float64(oxLevels-1)
		c.winLevel[i] = h.at(c.win, ratio)
		c.slabs[i] = oxBuildSlab(h, h.at(c.btn, ratio))
	}

	// A hole's inner shadow: darkest along the top edge, medium at the
	// sides, faint at the bottom; a second ring one pixel in.
	sh := c.winShadow
	c.holeShadow = []paintengine2d.GradientStop{Stop(0, sh.WithAlpha(0.52)), Stop(0.12, sh.WithAlpha(0.4)), Stop(0.5, sh.WithAlpha(0.36)), Stop(1, sh.WithAlpha(0.25))}
	c.holeTop = []paintengine2d.GradientStop{Stop(0, sh.WithAlpha(0.14)), Stop(0.15, sh.WithAlpha(0.05)), Stop(0.5, sh.WithAlpha(0.03)), Stop(1, sh.WithAlpha(0))}
	bs := oxBuildSlab(h, c.btn).shadow
	c.sunkShadow = []paintengine2d.GradientStop{Stop(0, bs.WithAlpha(0.46)), Stop(0.15, bs.WithAlpha(0.3)), Stop(0.5, bs.WithAlpha(0.25)), Stop(1, bs.WithAlpha(0.1))}
	c.sunkShadow2 = []paintengine2d.GradientStop{Stop(0, bs.WithAlpha(0.17)), Stop(0.15, bs.WithAlpha(0.06)), Stop(0.5, bs.WithAlpha(0.03)), Stop(1, bs.WithAlpha(0))}
	c.holeLight = []paintengine2d.GradientStop{Stop(0, c.winLight.WithAlpha(0)), Stop(0.6, c.winLight.WithAlpha(0)), Stop(1, c.winLight.WithAlpha(0.45))}
	// Flat wells: a dark upper edge, a light lower one.
	c.flatTop = []paintengine2d.GradientStop{Stop(0, c.winDark), Stop(0.5, c.winDark.WithAlpha(0))}
	c.flatBottom = []paintengine2d.GradientStop{Stop(0.5, c.winLight.WithAlpha(0)), Stop(1, c.winLight)}
	// Slots: the dark tone in an inner shadow, a light lower rim.
	c.slotDark, c.slotLight = c.winDark, c.winLight
	c.slotEdge = []paintengine2d.GradientStop{Stop(0.5, c.winLight.WithAlpha(0)), Stop(1, c.winLight.WithAlpha(0.6))}

	// The progress bar.
	lh := h.light(c.hl)
	c.progBase = Mix(c.hl, c.winDark, 0.2)
	c.progGloss = []paintengine2d.GradientStop{
		Stop(0, Mix(lh, c.winLight, 0.3)), Stop(0.5, Mix(lh, c.winLight, 0.3).WithAlpha(0)),
		Stop(0.6, Mix(lh, c.winLight, 0.3).WithAlpha(0)), Stop(1, Mix(lh, c.winLight, 0.3)),
	}
	c.progBevel = []paintengine2d.GradientStop{Stop(0, lh), Stop(0.5, c.hl), Stop(1, h.dark(c.hl))}
	c.progTop = Mix(c.hl, c.winLight, 0.8)
	c.progFill = []paintengine2d.GradientStop{Stop(0, c.progBase), Stop(1, c.progBase)}

	// Item selections: the highlight, lighter at the top.
	c.selStops = []paintengine2d.GradientStop{Stop(0, lighterPct(c.hl, 130)), Stop(1, c.hl)}
	c.hlOff = l.X("selectionInactive", Hex("#3e8acc"))
	c.selOffStops = []paintengine2d.GradientStop{Stop(0, lighterPct(c.hlOff, 130)), Stop(1, c.hlOff)}
	c.hlDis = Mix(c.hl, c.win, 0.55)
	c.selDisStops = []paintengine2d.GradientStop{Stop(0, lighterPct(c.hlDis, 115)), Stop(1, c.hlDis)}
	// The current item's focus mark: an underline fading in and out at the
	// ends, the highlight (the highlighted text on a selection).
	fl := func(col paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, col.WithAlpha(0)), Stop(0.2, col), Stop(0.8, col), Stop(1, col.WithAlpha(0))}
	}
	c.focusLine, c.focusLineSel = fl(c.hl), fl(c.hlText)
	hot := c.hl.WithAlpha(0.2)
	c.selHot = hot
	c.selHotStops = []paintengine2d.GradientStop{Stop(0, lighterPct(Mix(c.base, c.hl, 0.2), 130).WithAlpha(0.2)), Stop(1, hot)}

	// Tool tips: the tooltip colour's top → bottom tones with a light rim.
	tt, tb := h.top(c.tip), h.bottom(c.tip)
	c.tipStops = []paintengine2d.GradientStop{Stop(0, tt), Stop(1, tb)}
	c.tipEdge = []paintengine2d.GradientStop{Stop(0.5, h.light(tb)), Stop(0.9, tb)}

	// The selected tab's light wash.
	wl := c.winLight
	c.tabFill = []paintengine2d.GradientStop{
		Stop(0, wl.WithAlpha(0.5)), Stop(0.1, wl.WithAlpha(0.5)), Stop(0.25, wl.WithAlpha(0.3)),
		Stop(0.5, wl.WithAlpha(0.2)), Stop(0.75, wl.WithAlpha(0.1)), Stop(0.9, wl.WithAlpha(0)),
	}
	lt := h.light(c.top)
	c.floatSide = []paintengine2d.GradientStop{Stop(0, lt), Stop(0.6, lt.WithAlpha(0.5)), Stop(1, lt.WithAlpha(0))}
	c.sepDark = []paintengine2d.GradientStop{Stop(0, c.winDark.WithAlpha(0)), Stop(0.3, c.winDark), Stop(0.7, c.winDark), Stop(1, c.winDark.WithAlpha(0))}
	c.sepLight = []paintengine2d.GradientStop{Stop(0, c.winLight.WithAlpha(0)), Stop(0.3, c.winLight), Stop(0.7, c.winLight), Stop(1, c.winLight.WithAlpha(0))}
	c.tabDark = c.winDark.WithAlpha(0.6)
	c.tabMid = c.winDark.WithAlpha(0.4)
	c.menuHot = c.winMid
	c.disField = Mix(c.base, c.win, 0.5)
	return c
}

// oxBuildSlab derives one slab material from its base colour.
func oxBuildSlab(h oxTones, base paintengine2d.Color) oxSlab {
	light, dark := h.light(base), h.dark(base)
	s := oxSlab{base: base, light: light, dark: dark, shadow: h.shadow(base), mid: h.mid(base)}
	// Button body: light → base at 60% over (top − 0.2h … bottom + 0.4h).
	s.fill = []paintengine2d.GradientStop{Stop(0, light), Stop(0.6, base), Stop(1, base)}
	tint := Mix(base, light, 0.5)
	s.fillDef = []paintengine2d.GradientStop{Stop(0, h.light(tint)), Stop(0.6, tint), Stop(1, tint)}
	// Sunken: light → base over (top − h … bottom); dark themes invert.
	if oxLstar(s.shadow) > oxLstar(base) {
		s.fillSunk = []paintengine2d.GradientStop{Stop(0, base), Stop(1, light)}
	} else {
		s.fillSunk = []paintengine2d.GradientStop{Stop(0, Mix(light, base, 0.5)), Stop(1, base)}
	}
	// Check box body: light → base over (top − h … bottom); dark themes
	// run base → light.
	if oxLstar(s.shadow) > oxLstar(base) {
		s.slabFill = []paintengine2d.GradientStop{Stop(0, base), Stop(1, light)}
	} else {
		s.slabFill = []paintengine2d.GradientStop{Stop(0, light), Stop(1, base)}
	}
	// Rim: light, easing to light at 85% over the lower corners.
	s.rim = []paintengine2d.GradientStop{Stop(0, light), Stop(1, light.WithAlpha(0.85))}
	// Slider handle: light → dark, its outline light → a darker mix.
	s.knob = []paintengine2d.GradientStop{Stop(0, light), Stop(1, dark)}
	s.knobSunk = []paintengine2d.GradientStop{Stop(0, dark), Stop(1, light)}
	s.knobLine = []paintengine2d.GradientStop{Stop(0, light), Stop(1, Mix(light, dark, 0.6))}
	// Radio bead.
	s.round = []paintengine2d.GradientStop{Stop(0, light), Stop(0.9, light.WithAlpha(0.85))}
	s.roundInner = []paintengine2d.GradientStop{Stop(0, light), Stop(1, base)}
	// Scroll handle: colour → mid, a faint light sheen at the top.
	s.handle = []paintengine2d.GradientStop{Stop(0, base), Stop(1, s.mid)}
	s.handleGloss = []paintengine2d.GradientStop{Stop(0, light.WithAlpha(0.25)), Stop(0.45, light.WithAlpha(0))}
	return s
}

// ---- painting helpers ---------------------------------------------------------------------------

// oxK is one Oxygen tile pixel at the look's scale.
func oxK(l *Classic) float32 { return l.S(1) }

// oxU is one device-crisp pixel at the look's scale.
func oxU(l *Classic) float32 {
	u := snap(l.S(1))
	if u < 1 {
		u = 1
	}
	return u
}

// level is the window-gradient level of a rect's centre (Oxygen tints a
// control by where it sits in the window's 300px gradient).
func (c *oxygen) level(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) int {
	m := ctx.Matrix()
	y := m.F + (b.Min.Y+b.Max.Y)*0.5*m.D
	ratio := y / l.S(300)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return int(ratio*float32(oxLevels-1) + 0.5)
}

func (c *oxygen) slabAt(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) *oxSlab {
	return &c.slabs[c.level(l, ctx, b)]
}

// winAt is the window colour under a rect (for opaque patches over the
// window gradient).
func (c *oxygen) winAt(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) paintengine2d.Color {
	return c.winLevel[c.level(l, ctx, b)]
}

// oxRing strokes a ring of width w whose centre line lies d outside the
// rounded rect e of radius r (a negative d is inside).
func oxRing(ctx *paintengine2d.Context, e paintengine2d.Rect, r, d, w float32, col paintengine2d.Color) {
	if col.A <= 0 || w <= 0 {
		return
	}
	rr := e.Inset(-d)
	if rr.Dx() <= 0 || rr.Dy() <= 0 {
		return
	}
	rad := r + d
	if rad < 0 {
		rad = 0
	}
	ctx.DrawRoundRect(rr, rad, rad, paintengine2d.StrokePaint(col, w))
}

// oxE is a slab's outer edge inside its rect b — three pixels in, leaving
// room for the shadow and glow — and its corner radius.
func oxE(l *Classic, b paintengine2d.Rect) (paintengine2d.Rect, float32) {
	k := oxK(l)
	e := b.Inset(3 * k)
	r := l.rx(3.5)
	if lim := e.Dy() * 0.5; r > lim {
		r = lim
	}
	if lim := e.Dx() * 0.5; r > lim {
		r = lim
	}
	return e, r
}

// shadow is a slab's soft shadow as measured on Oxygen screenshots: the
// body's shape one pixel lower at 53% (only its lower edge shows below the
// body), then rings at 28% and 7% — so three pixels of shadow below, two
// at the sides and one faint pixel above. filled = the body will cover the
// inside; otherwise only the strip below it is laid.
func (c *oxygen) shadow(ctx *paintengine2d.Context, e paintengine2d.Rect, r, k float32, col paintengine2d.Color, filled bool) {
	e1 := e.Translate(paintengine2d.Pt(0, k))
	if filled {
		ctx.DrawRoundRect(e1, r, r, paintengine2d.Fill(col.WithAlpha(col.A*0.53)))
	} else {
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(e1.Min.X, e.Max.Y, e1.Dx(), k))
		ctx.DrawRoundRect(e1, r, r, paintengine2d.Fill(col.WithAlpha(col.A*0.53)))
		ctx.Restore()
	}
	oxRing(ctx, e1, r, 0.5*k, k, col.WithAlpha(col.A*0.28))
	oxRing(ctx, e1, r, 1.5*k, k, col.WithAlpha(col.A*0.07))
}

// glow is the Oxygen hover / focus glow: a strong pixel against the rim
// and two fading ones.
func (c *oxygen) glow(ctx *paintengine2d.Context, e paintengine2d.Rect, r, k float32, col paintengine2d.Color) {
	if col.A <= 0 {
		return
	}
	oxRing(ctx, e, r, 0.5*k, k, col.WithAlpha(col.A*0.85))
	oxRing(ctx, e, r, 1.5*k, k, col.WithAlpha(col.A*0.32))
	oxRing(ctx, e, r, 2.5*k, k, col.WithAlpha(col.A*0.07))
}

// oxBody is where a slab's fill sits (its outer edge; the rim covers the
// outermost pixel).
func oxBody(b paintengine2d.Rect, k float32) paintengine2d.Rect { return b.Inset(3 * k) }

// slab paints a raised slab into b: shadow, glow, the body gradient
// (unless noFill) and the light rim.
func (c *oxygen) slab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, s *oxSlab, glow paintengine2d.Color, noFill, def bool) {
	k := oxK(l)
	e, r := oxE(l, b)
	if e.Dx() < 3*k || e.Dy() < 3*k {
		return
	}
	c.shadow(ctx, e, r, k, s.shadow, !noFill)
	c.glow(ctx, e, r, k, glow)
	if !noFill {
		stops := s.fill
		if def {
			stops = s.fillDef
		}
		h := e.Dy()
		ctx.DrawRoundRect(e, r, r, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(0, e.Min.Y-0.2*h), End: paintengine2d.Pt(0, e.Max.Y+0.4*h), Stops: stops}))
	}
	c.rim(l, ctx, e, r, s)
}

// rim is a slab's bevel: the body's outermost pixel in the light colour,
// easing to 85% along the bottom.
func (c *oxygen) rim(l *Classic, ctx *paintengine2d.Context, e paintengine2d.Rect, r float32, s *oxSlab) {
	k := oxK(l)
	if e.Dx() < 2*k || e.Dy() < 2*k {
		return
	}
	rr := r - 0.5*k
	if rr < 0 {
		rr = 0
	}
	ctx.DrawRoundRect(e.Inset(0.5*k), rr, rr, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, e.Max.Y-4*k), End: paintengine2d.Pt(0, e.Max.Y), Stops: s.rim},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4},
	})
}

// sunken is a pressed slab: the body lighter at the top, an inner shadow
// heaviest along the top edge, a light contrast line just below it.
func (c *oxygen) sunken(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, s *oxSlab) {
	k := oxK(l)
	e, r := oxE(l, b)
	if e.Dx() < 3*k || e.Dy() < 3*k {
		return
	}
	h := e.Dy()
	ctx.DrawRoundRect(e, r, r, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, e.Min.Y-h), End: paintengine2d.Pt(0, e.Max.Y), Stops: s.fillSunk}))
	c.inner(ctx, e, r, k, c.sunkShadow, c.sunkShadow2)
	ctx.DrawRoundRect(e.Inset(-0.5*k), r+0.5*k, r+0.5*k, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, e.Min.Y), End: paintengine2d.Pt(0, e.Max.Y+k), Stops: c.holeLight},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
}

// inner lays two inner-shadow rings inside e: vertical gradients (top →
// middle → bottom) of the shadow colour.
func (c *oxygen) inner(ctx *paintengine2d.Context, e paintengine2d.Rect, r, k float32, ring1, ring2 []paintengine2d.GradientStop) {
	g := func(stops []paintengine2d.GradientStop, d float32) {
		rr := r - d
		if rr < 0 {
			rr = 0
		}
		ctx.DrawRoundRect(e.Inset(d), rr, rr, paintengine2d.Paint{
			Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, e.Min.Y), End: paintengine2d.Pt(0, e.Max.Y), Stops: stops},
			Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
	}
	g(ring1, 0.5*k)
	g(ring2, 1.5*k)
}

// hole is the inner shadow of a sunken frame one pixel inside b —
// half-dark along the top, lighter down the sides, a quarter at the
// bottom — or the glow when focused / hovered, with a light lower edge
// when asked.
func (c *oxygen) hole(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, glow paintengine2d.Color, contrast bool) {
	k := oxK(l)
	in := b.Inset(k)
	if in.Dx() < 4*k || in.Dy() < 4*k {
		return
	}
	r := l.rx(3)
	if glow.A > 0 {
		oxRing(ctx, in, r, -0.5*k, k, glow)
		oxRing(ctx, in, r, -1.5*k, k, glow.WithAlpha(glow.A*0.45))
		oxRing(ctx, in, r, -2.5*k, k, glow.WithAlpha(glow.A*0.15))
	} else {
		c.inner(ctx, in, r, k, c.holeShadow, c.holeTop)
	}
	if contrast {
		ctx.DrawRoundRect(b.Inset(0.5*k), r+0.5*k, r+0.5*k, paintengine2d.Paint{
			Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y+2*k), Stops: c.holeLight},
			Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
	}
}

// field paints a text field: the view colour one pixel inside b and the
// hole's shadow (the glow when focused or hovered).
func (c *oxygen) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	k := oxK(l)
	in := b.Inset(k)
	if in.Dx() < 4*k || in.Dy() < 4*k {
		return
	}
	fill := c.base
	if st.Disabled() {
		fill = c.disField
	}
	rad := l.rx(3)
	ctx.DrawRoundRect(in, rad, rad, paintengine2d.Fill(fill))
	var glow paintengine2d.Color
	switch {
	case st.Disabled():
	case st.Focused():
		glow = c.focus
	case st.Hovered():
		glow = c.hover
	}
	c.hole(l, ctx, b, glow, false)
}

// flatWell is a flat patch with a dark upper edge and a light lower one —
// the hot menu item and menu title.
func (c *oxygen) flatWell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	k := oxK(l)
	if b.Dx() < 4*k || b.Dy() < 4*k {
		return
	}
	r := l.rx(3)
	ctx.DrawRoundRect(b.Inset(0.5*k), r, r, paintengine2d.Fill(col))
	ctx.DrawRoundRect(b.Inset(k), l.rx(2.5), l.rx(2.5), paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y-2*k), End: paintengine2d.Pt(0, b.Max.Y), Stops: c.flatTop},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
	ctx.DrawRoundRect(b.Inset(0.5*k), l.rx(3.5), l.rx(3.5), paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y+4*k), Stops: c.flatBottom},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
}

// slot is the dark tone in a rounded slot with an inner shadow and a light
// lower rim (scroll grooves, progress and slider grooves).
func (c *oxygen) slot(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, small bool) {
	k := oxK(l)
	if b.Dx() < 3*k || b.Dy() < 3*k {
		return
	}
	in := b.Inset(k)
	r := l.rx(3)
	if small {
		r = l.rx(2.5)
	}
	if lim := in.Dx() * 0.5; r > lim {
		r = lim
	}
	if lim := in.Dy() * 0.5; r > lim {
		r = lim
	}
	ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.slotDark))
	sh := c.winShadow
	a := float32(1)
	if small {
		a = 0.6
	}
	ctx.Save()
	ctx.ClipRoundRect(in, r, r)
	ctx.DrawRoundRect(in.Inset(0.5*k), r, r, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, in.Min.Y), End: paintengine2d.Pt(0, in.Max.Y), Stops: c.holeShadow},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}, Opacity: a})
	fuHLine(ctx, in.Min.X, in.Max.X, in.Min.Y+k, k, sh.WithAlpha(0.12*a))
	ctx.Restore()
	ctx.DrawRoundRect(b.Inset(0.5*k), r+0.5*k, r+0.5*k, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y), Stops: c.slotEdge},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
}

// oxArrow is Oxygen's arrow: a 1.6px round-capped chevron over a light
// copy one pixel lower (the contrast "etch").
func oxArrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col, contrast paintengine2d.Color, small bool) {
	if b.Dx() < 3 || b.Dy() < 3 {
		return
	}
	k := oxK(l)
	ctr := b.Center()
	// A right-angled chevron about 7px across and 4px deep (measured).
	a, d := 3.6*k, 1.9*k
	if small {
		a, d = 2.9*k, 1.5*k
	}
	var pts [3]paintengine2d.Point
	switch dir {
	case DirUp:
		pts = [3]paintengine2d.Point{{X: -a, Y: d}, {X: 0, Y: -d}, {X: a, Y: d}}
	case DirDown:
		pts = [3]paintengine2d.Point{{X: -a, Y: -d}, {X: 0, Y: d}, {X: a, Y: -d}}
	case DirLeft:
		pts = [3]paintengine2d.Point{{X: d, Y: -a}, {X: -d, Y: 0}, {X: d, Y: a}}
	default:
		pts = [3]paintengine2d.Point{{X: -d, Y: -a}, {X: d, Y: 0}, {X: -d, Y: a}}
	}
	stroke := func(dy float32, cc paintengine2d.Color) {
		if cc.A <= 0 {
			return
		}
		p := paintengine2d.NewPath()
		p.MoveTo(ctr.X+pts[0].X, ctr.Y+pts[0].Y+dy)
		p.LineTo(ctr.X+pts[1].X, ctr.Y+pts[1].Y+dy)
		p.LineTo(ctr.X+pts[2].X, ctr.Y+pts[2].Y+dy)
		ctx.DrawPath(p, paintengine2d.Paint{Color: cc, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 1.6 * k, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	}
	stroke(k, contrast)
	stroke(0, col)
}

// buttonGlow is a control's glow colour: hover wins over focus.
func (c *oxygen) buttonGlow(st ControlState) paintengine2d.Color {
	switch {
	case st.Disabled():
		return paintengine2d.Color{}
	case st.Hovered():
		return c.hover
	case st.Focused():
		return c.focus
	}
	return paintengine2d.Color{}
}

// button paints a push-button slab (sunken when held) into b.
func (c *oxygen) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, def bool) {
	s := c.slabAt(l, ctx, b)
	if st.Pressed() && !st.Disabled() || (st.Toggle() && st.Checked()) {
		c.sunken(l, ctx, b, s)
		return
	}
	c.slab(l, ctx, b, s, c.buttonGlow(st), false, def && !st.Disabled())
}

// ---- parts -------------------------------------------------------------------------------------------

func (e oxygenEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := oxygenColors(l)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	if b.Empty() {
		return fg
	}
	switch role {
	case RoleButton, RoleCombo:
		c.button(l, ctx, b, st, st.Primary())
	case RoleTool:
		e.tool(l, ctx, b, st)
		fg = c.text
		if st.Disabled() {
			fg = c.dis
		}
	case RoleField:
		c.field(l, ctx, b, st)
		fg = c.viewText
		if st.Disabled() {
			fg = c.dis
		}
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, st.Checked())
	case RoleRow:
		return c.row(l, ctx, b, st)
	case RoleMenu:
		if st.Hovered() || st.Pressed() || st.Checked() {
			e.MenuHighlight(l, ctx, b, false)
		}
		return c.text
	case RoleThumb:
		c.handle(l, ctx, b, b.Dy() >= b.Dx(), st.Hovered() || st.Pressed())
	case RoleTrack:
		c.slot(l, ctx, b, false)
	case RoleBar, RolePanel, RoleSplitter:
		// Transparent: the window gradient shows through.
		return c.text
	}
	return fg
}

// tool paints an auto-raise tool button: a thin ring of the glow when hot
// or focused, a hole with a darkened centre when held or latched.
func (e oxygenEngine) tool(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 6*k || b.Dy() < 6*k {
		return
	}
	latched := st.Toggle() && st.Checked()
	if (st.Pressed() && !st.Disabled()) || latched {
		a := float32(1)
		if st.Hovered() && !st.Pressed() {
			a = 0.25
		}
		r := l.rx(3.5)
		ctx.DrawRoundRect(b.Inset(k), r, r, paintengine2d.Fill(c.winMid.WithAlpha(a)))
		c.hole(l, ctx, b, paintengine2d.Color{}, true)
		return
	}
	if glow := c.buttonGlow(st); glow.A > 0 {
		r := l.rx(2.5)
		ctx.DrawRoundRect(b.Inset(1.5*k), r, r, paintengine2d.StrokePaint(glow, k))
	}
}

// CheckIndicator is a square slab (a flat well while held) with a
// two-stroke tick over its light contrast copy.
func (e oxygenEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := oxygenColors(l)
	k := oxK(l)
	side := box.Dx()
	if box.Dy() < side {
		side = box.Dy()
	}
	b := paintengine2d.XYWH(box.Min.X+(box.Dx()-side)*0.5, box.Min.Y+(box.Dy()-side)*0.5, side, side)
	if side < 9*k {
		return
	}
	s := c.slabAt(l, ctx, b)
	pressed := st.Pressed() && !st.Disabled()
	if pressed {
		c.flatWell(l, ctx, b.Inset(2.5*k), c.winAt(l, ctx, b))
	} else {
		// The body runs light → base from one body height above it.
		sb, r := oxE(l, b)
		c.shadow(ctx, sb, r, k, s.shadow, true)
		c.glow(ctx, sb, r, k, c.buttonGlow(st))
		h := sb.Dy()
		ctx.DrawRoundRect(sb, r, r, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(0, sb.Min.Y-h), End: paintengine2d.Pt(0, sb.Max.Y), Stops: s.slabFill}))
		c.rim(l, ctx, sb, r, s)
	}
	if !checked {
		return
	}
	mark := c.btnText
	if st.Disabled() {
		mark = c.dis
	}
	if pressed {
		mark = mark.WithAlpha(0.3)
	}
	body := 17 * oxToggleScale(side, k) * k
	oxTick(ctx, b.Center(), body, k, s.light.WithAlpha(mark.A))
	oxTick(ctx, b.Center(), body, 0, mark)
}

// oxTick strokes the check mark over a check box body of side body centred
// on ctr, dy lower (its light copy is laid one pixel down first). The
// proportions are measured from Oxygen screenshots: a short arm from the
// left of the centre down to a vertex just left of and below it, the long
// arm up to the upper right, a stroke about an eighth of the body wide.
func oxTick(ctx *paintengine2d.Context, ctr paintengine2d.Point, body, dy float32, col paintengine2d.Color) {
	if col.A <= 0 || body <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(ctr.X-0.25*body, ctr.Y+0.05*body+dy)
	p.LineTo(ctr.X-0.06*body, ctr.Y+0.22*body+dy)
	p.LineTo(ctr.X+0.30*body, ctr.Y-0.19*body+dy)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: 0.12 * body, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// oxToggleScale sizes a check box or radio drawn into a side-px box: the
// 23px metric includes a pixel of focus margin each side, so boxes down to
// 21px keep the full-size mark, and smaller or larger ones scale.
func oxToggleScale(side, k float32) float32 {
	switch {
	case side >= 23*k:
		return side / (23 * k)
	case side >= 21*k:
		return 1
	}
	return side / (21 * k)
}

// bead lays a round slab's shadow (one pixel lower) and glow for a disc
// of radius r at ctr.
func (c *oxygen) bead(ctx *paintengine2d.Context, ctr paintengine2d.Point, r, k float32, s *oxSlab, glow paintengine2d.Color) {
	e := paintengine2d.XYWH(ctr.X-r, ctr.Y-r, 2*r, 2*r)
	c.shadow(ctx, e, r, k, s.shadow, true)
	c.glow(ctx, e, r, k, glow)
}

// RadioIndicator is a 15px round slab with the dot in the button text
// over a light copy.
func (e oxygenEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := oxygenColors(l)
	k := oxK(l)
	side := box.Dx()
	if box.Dy() < side {
		side = box.Dy()
	}
	if side < 9*k {
		return
	}
	ctr := box.Center()
	s := c.slabAt(l, ctx, box)
	sc := oxToggleScale(side, k)
	r := 7.5 * k * sc
	c.bead(ctx, ctr, r, k, s, c.buttonGlow(st))
	// The body runs light → base, the light well above the bead.
	ctx.DrawCircle(ctr, r, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, ctr.Y-3.8*r), End: paintengine2d.Pt(0, ctr.Y+1.25*r), Stops: s.roundInner}))
	ctx.DrawCircle(ctr, r-0.5*k, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, ctr.Y+r-4*k), End: paintengine2d.Pt(0, ctr.Y+r), Stops: s.rim},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
	if !selected {
		return
	}
	dot := c.btnText
	if st.Disabled() {
		dot = c.dis
	}
	rr := 2.6 * k * sc
	ctx.DrawCircle(paintengine2d.Pt(ctr.X, ctr.Y+rr*0.5), rr, paintengine2d.Fill(s.light.WithAlpha(dot.A)))
	ctx.DrawCircle(ctr, rr, paintengine2d.Fill(dot))
}

// Arrow is Oxygen's etched chevron in col.
func (oxygenEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	c := oxygenColors(l)
	oxArrow(l, ctx, b, dir, col, c.winLight, false)
}

// Expander is the tree view's small chevron.
func (oxygenEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := oxygenColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	oxArrow(l, ctx, b, dir, c.viewText, paintengine2d.Color{}, true)
}

// MenuHighlight is Oxygen's dark menu selection: a flat well in the
// menu's mid colour.
func (oxygenEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := oxygenColors(l)
	k := oxK(l)
	r := b
	if !attachBottom {
		r = paintengine2d.XYWH(b.Min.X+k, b.Min.Y+k, b.Dx()-2*k, b.Dy()-k)
	}
	c.flatWell(l, ctx, r, c.menuHot)
}

func (oxygenEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return oxygenColors(l).text
}

// Fields paint their own focus (the glow inside the hole).
func (oxygenEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is a thin ring of the focus glow.
func (oxygenEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 5*k || b.Dy() < 5*k {
		return
	}
	r := l.rx(2.5)
	ctx.DrawRoundRect(b.Inset(1.5*k), r, r, paintengine2d.StrokePaint(c.focus, k))
}

// ItemFocus marks the current item with a 1px underline along its bottom
// that fades in and out at the ends, in the highlight (the highlighted
// text on a selection).
func (oxygenEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := oxygenColors(l)
	u := oxU(l)
	if st.Disabled() || b.Dx() < 10*u || b.Dy() < 3*u {
		return
	}
	stops := c.focusLine
	if st.Checked() {
		stops = c.focusLineSel
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), HGradient(b, stops...))
}

// ViewFrameInsets: views sit in a hole (its inner shadow is three pixels
// deep).
func (oxygenEngine) ViewFrameInsets(l *Classic) Insets {
	u := snap(l.S(3))
	return Insets{Top: u, Right: u, Bottom: u, Left: u}
}

// DrawViewFrame is the hole Oxygen frames item views with: the view colour
// in an inner shadow that becomes the focus glow when the view has keyboard
// focus, the hover glow under the pointer.
func (oxygenEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	oxygenColors(l).field(l, ctx, b, st)
}

// ---- scroll bars ---------------------------------------------------------------------------------

// ScrollBarStyle: Oxygen's 16px bar, a slot the full width of it. KDE 4 put
// one arrow at the top and two at the bottom by default; the spec'd look
// here keeps the bar arrow-free (ArrowPlacement cannot express "1 + 2").
func (oxygenEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsNone, MinThumb: 21}
}

// handle is the scroll bar handle: a small slab ringed in the shadow
// colour (the hover glow when hot) with its glossy gradient body.
func (c *oxygen) handle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical, hot bool) {
	k := oxK(l)
	if b.Dx() < 6*k || b.Dy() < 6*k {
		return
	}
	s := &c.slabs[oxLevels/2]
	glow := s.shadow.WithAlpha(0.4)
	if hot {
		glow = c.hover
	}
	body := b.Inset(3 * k)
	// The outer glow / shadow ring hugging the body.
	r := l.rx(2.5)
	ctx.DrawRoundRect(body.Inset(-0.7*k), r+0.7*k, r+0.7*k, paintengine2d.StrokePaint(glow, 1.3*k))
	ctx.DrawRoundRect(body.Inset(-1.8*k), r+1.8*k, r+1.8*k, paintengine2d.StrokePaint(glow.WithAlpha(glow.A*0.45), 1.0*k))
	ctx.DrawRoundRect(body.Inset(-2.6*k), r+2.6*k, r+2.6*k, paintengine2d.StrokePaint(glow.WithAlpha(glow.A*0.15), 0.7*k))
	g := VGradient(body, s.handle...)
	if !vertical {
		g = VGradient(body, s.handle...)
	}
	ctx.DrawRoundRect(body, r, r, g)
	ctx.DrawRoundRect(body, r, r, VGradient(body, s.handleGloss...))
	// A light patch down the middle of the handle, as the reflect pattern
	// leaves it at rest.
	if vertical && body.Dx() > 4*k {
		ctx.DrawRect(paintengine2d.XYWH(body.Min.X+2*k, body.Min.Y+2*k, body.Dx()-4*k, body.Dy()-4*k), paintengine2d.Fill(s.light.WithAlpha(0.08)))
	} else if !vertical && body.Dy() > 4*k {
		ctx.DrawRect(paintengine2d.XYWH(body.Min.X+2*k, body.Min.Y+2*k, body.Dx()-4*k, body.Dy()-4*k), paintengine2d.Fill(s.light.WithAlpha(0.08)))
	}
}

func (e oxygenEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := oxygenColors(l)
	k := oxK(l)
	if p.Bar.Empty() {
		return
	}
	// The slot spans the track, 1px in across the bar.
	tr := p.Track
	if tr.Empty() {
		tr = p.Bar
	}
	if vertical {
		tr = paintengine2d.XYWH(p.Bar.Min.X+k, tr.Min.Y, p.Bar.Dx()-2*k, tr.Dy())
	} else {
		tr = paintengine2d.XYWH(tr.Min.X, p.Bar.Min.Y+k, tr.Dx(), p.Bar.Dy()-2*k)
	}
	c.slot(l, ctx, tr, (vertical && tr.Dx() < 10*k) || (!vertical && tr.Dy() < 10*k))
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	th := p.Thumb
	if vertical {
		th = paintengine2d.XYWH(tr.Min.X, th.Min.Y, tr.Dx(), th.Dy())
	} else {
		th = paintengine2d.XYWH(th.Min.X, tr.Min.Y, th.Dx(), tr.Dy())
	}
	c.handle(l, ctx, th, vertical, st.Hot == ScrollThumbPart || st.Pressed == ScrollThumbPart)
}

func (e oxygenEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	vertical := track.Dy() >= track.Dx()
	ss := ScrollState{Disabled: st.Disabled(), Hovered: st.Hovered()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	if st.Hovered() {
		ss.Hot = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, vertical, ss)
}

// ---- frames ----------------------------------------------------------------------------------------------

func (oxygenEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top += l.body.Height() + l.S(6)
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a raised slab frame whose sides fade out towards the
// bottom over a light wash, with the title centred inside its top.
func (e oxygenEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 16*k || b.Dy() < 16*k {
		return
	}
	s := c.slabAt(l, ctx, b)
	// The light wash (light at 40% at the top, clear by the bottom).
	body := oxBody(b, k)
	ctx.DrawRoundRect(body, l.rx(2), l.rx(2), paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, body.Min.Y-10*k), End: paintengine2d.Pt(0, body.Max.Y), Stops: c.tabFill}))
	// slope: the slab frame, its sides fading out over the lower part.
	sb, r := oxE(l, b)
	if !raised {
		fade := b.Dy() * 0.45
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-fade))
		c.shadow(ctx, sb, r, k, s.shadow, false)
		c.rim(l, ctx, sb, r, s)
		ctx.Restore()
		// The fading part: the sides again, under a mask that clears them
		// towards the bottom (a few steps of decreasing opacity).
		y0 := b.Max.Y - fade
		for i := 0; i < 4; i++ {
			seg := paintengine2d.XYWH(b.Min.X, y0+fade*float32(i)/4, b.Dx(), fade/4)
			ctx.Save()
			ctx.ClipRect(seg)
			a := 1 - (float32(i)+0.5)/4
			ctx.DrawRoundRect(sb.Inset(0.5*k), r-0.5*k, r-0.5*k, paintengine2d.Paint{
				Color: s.light, Style: paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}, Opacity: a * 0.9})
			oxRing(ctx, sb, r, 0.5*k, k, s.shadow.WithAlpha(0.28*a))
			ctx.Restore()
		}
	} else {
		c.shadow(ctx, sb, r, k, s.shadow, false)
		c.rim(l, ctx, sb, r, s)
	}
	if title != "" {
		f := l.bold
		if f == nil {
			f = l.body
		}
		tb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y+l.S(5), b.Dx()-l.S(16), f.Height()+l.S(2))
		l.drawFittedText(ctx, f, title, tb, c.text, AlignCenter, 0)
	}
}

// oxTitleH is the Oxygen decoration's title bar height at the UI font.
func oxTitleH(l *Classic) float32 {
	h := snap(l.body.Height() + l.S(10))
	if h < snap(l.S(24)) {
		h = snap(l.S(24))
	}
	return h
}

func (oxygenEngine) WindowFrameInsets(l *Classic) Insets {
	k := oxK(l)
	return Insets{Top: oxTitleH(l) + 2*k, Right: 4 * k, Bottom: 4 * k, Left: 4 * k}
}

// WindowCloseRect is the decoration's round close button at the right of
// the title bar.
func (oxygenEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	h := oxTitleH(l)
	if b.Dy() < h || b.Dx() < h*2 {
		return paintengine2d.Rect{}
	}
	side := snap(l.S(21))
	if side > h {
		side = h
	}
	return paintengine2d.XYWH(snap(b.Max.X-side-l.S(6)), snap(b.Min.Y+l.S(2)+(h-side)*0.5), side, side)
}

// DrawWindowFrame is the Oxygen window decoration: the window gradient
// through a seamless title bar with a centred title, round slab buttons
// (the close glyph glows red when hot) and the float frame's light rim.
func (e oxygenEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := oxygenColors(l)
	k := oxK(l)
	b = fuSnap(b)
	h := oxTitleH(l)
	if b.Dx() < 12*k || b.Dy() < h {
		return
	}
	r := l.rx(5)
	ctx.Save()
	ctx.ClipRoundRect(b, r, r)
	e.fillGradient(l, ctx, b, b.Dy())
	ctx.Restore()
	c.popupFrame(l, ctx, b)
	right := b.Max.X - l.S(8)
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			bst := StateNone
			if st.CloseHot {
				bst = StateHovered
			}
			if st.ClosePress {
				bst |= StatePressed
			}
			e.RadioIndicator(l, ctx, cb, bst&^StateHovered, false)
			glyph := c.btnText
			switch {
			case st.ClosePress:
				glyph = Mix(c.btnText, Hex("#bf0303"), 0.8)
			case st.CloseHot:
				glyph = Hex("#bf0303")
			}
			if !st.Active {
				glyph = glyph.WithAlpha(0.6)
			}
			g := cb.Inset(cb.Dx() * 0.34)
			DrawCross(ctx, g.Translate(paintengine2d.Pt(0, k)), c.winLight, 1.6*k)
			DrawCross(ctx, g, glyph, 1.6*k)
			right = cb.Min.X - l.S(6)
		}
	}
	if title != "" {
		col := c.text
		if !st.Active {
			col = Mix(c.text, c.win, 0.45)
		}
		left := b.Min.X + (b.Max.X - right)
		f := l.bold
		if f == nil {
			f = l.body
		}
		tr := paintengine2d.XYWH(left, b.Min.Y+l.S(2), right-left, h)
		l.drawFittedText(ctx, f, title, tr.Translate(paintengine2d.Pt(0, k)), c.winLight.WithAlpha(0.7), AlignCenter, 0)
		l.drawFittedText(ctx, f, title, tr, col, AlignCenter, 0)
	}
}

// popupFrame is a floating panel's edge: a light rim along the top,
// fading down its sides, and a soft outline.
func (c *oxygen) popupFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	k := oxK(l)
	if b.Dx() < 8*k || b.Dy() < 8*k {
		return
	}
	r := l.rx(5)
	ctx.DrawRoundRect(b.Inset(0.5*k), r, r, paintengine2d.StrokePaint(c.winShadow.WithAlpha(0.35), k))
	in := b.Inset(1.3 * k)
	ctx.DrawRoundRect(in, r-k, r-k, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, in.Min.Y), End: paintengine2d.Pt(0, in.Max.Y), Stops: c.floatSide},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: 0.8 * k, MiterLimit: 4}})
}

// fillGradient paints the Oxygen window gradient into b as if b were a
// window of height wh (the menu / dialog variant of the background).
func (e oxygenEngine) fillGradient(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, wh float32) {
	c := oxygenColors(l)
	split := l.S(300)
	if v := wh * 0.75; v < split {
		split = v
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.bottom))
	top := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), split)
	ctx.DrawRect(top.Intersect(b), paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Min.Y+split), Stops: c.bgStops}))
}

// DrawWindowBackground is Oxygen's window background: the vertical
// gradient over the top 300px (or ¾ of the window) and the elliptical
// radial light — 600×64 at most — centred on the top edge. It depends only
// on the window rect b, so a partial repaint under a clip matches.
func (e oxygenEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := oxygenColors(l)
	if b.Empty() {
		return
	}
	e.fillGradient(l, ctx, b, b.Dy())
	rw := l.S(600)
	if b.Dx() < rw {
		rw = b.Dx()
	}
	rh := l.S(64)
	if rw <= 0 || rh <= 0 {
		return
	}
	cx := (b.Min.X + b.Max.X) * 0.5
	// A circle of radius rh at the top centre, stretched to rw/2 × rh.
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(cx-rw*0.5, b.Min.Y, rw, rh))
	ctx.Translate(cx, b.Min.Y)
	ctx.Scale(rw*0.5/rh, 1)
	ctx.DrawRect(paintengine2d.XYWH(-rh, 0, rh*2, rh), paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(0, 0), Radius: rh, Stops: c.radialStops}))
	ctx.Restore()
}

// TabOutset: the selected tab's slab reaches a pixel over its neighbours.
func (oxygenEngine) TabOutset(l *Classic) Insets {
	k := oxU(l)
	return Insets{Left: k, Right: k}
}

// oxPaneTop is the top of the tab widget's slab frame: TabBar_BaseOverlap
// (7px) into the tab bar.
func oxPaneTop(l *Classic, bottomOfBar float32) float32 { return bottomOfBar - 7*oxK(l) }

// DrawTabPane is a slab frame without fill (the window gradient shows
// through) whose top edge the tab bar draws.
func (e oxygenEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := oxygenColors(l)
	k := oxK(l)
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		c.paneSlab(l, ctx, b, b, b.Min.Y)
		return
	}
	barBottom := b.Min.Y + th
	pane := paintengine2d.XYWH(b.Min.X, oxPaneTop(l, barBottom), b.Dx(), b.Max.Y-oxPaneTop(l, barBottom))
	if pane.Dy() < 16*k {
		return
	}
	c.paneSlab(l, ctx, pane, paintengine2d.XYWH(b.Min.X, barBottom, b.Dx(), b.Max.Y-barBottom), 0)
}

// paneSlab renders the slab frame of pane clipped to clip.
func (c *oxygen) paneSlab(l *Classic, ctx *paintengine2d.Context, pane, clip paintengine2d.Rect, _ float32) {
	s := c.slabAt(l, ctx, pane)
	k := oxK(l)
	sb, r := oxE(l, pane)
	ctx.Save()
	ctx.ClipRect(clip)
	c.shadow(ctx, sb, r, k, s.shadow, false)
	c.rim(l, ctx, sb, r, s)
	ctx.Restore()
}

// ---- controls ----------------------------------------------------------------------------------------------

// DrawPanel: a raised panel is a slab frame (no fill), a flat one a hole.
func (e oxygenEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := oxygenColors(l)
	if raised {
		s := c.slabAt(l, ctx, b)
		k := oxK(l)
		if b.Dx() > 12*k && b.Dy() > 12*k {
			sb, r := oxE(l, b)
			c.shadow(ctx, sb, r, k, s.shadow, false)
			c.rim(l, ctx, sb, r, s)
		}
		return
	}
	c.hole(l, ctx, b, paintengine2d.Color{}, true)
}

func (e oxygenEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := oxygenColors(l)
	c.button(l, ctx, b, st, st.Primary())
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	lb := b
	lb.Max.Y -= oxK(l) * 0.8
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(12))
}

func (e oxygenEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := oxygenColors(l)
	if label == "" {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

func (e oxygenEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	if side > b.Dy() {
		side = b.Dy()
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e oxygenEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	if side > b.Dy() {
		side = b.Dy()
	}
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// DrawSwitch: KDE 4 had no switch; Oxygen's is a slot with the round
// slider slab at the checked end, the slot glowing the highlight when on.
func (e oxygenEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := oxygenColors(l)
	k := oxK(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if tw <= 0 {
		tw = l.S(40)
	}
	if th <= 0 {
		th = l.S(21)
	}
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th)
	if track.Dx() < 10*k || track.Dy() < 10*k {
		return
	}
	g := paintengine2d.XYWH(track.Min.X+2*k, track.Min.Y+track.Dy()*0.25, track.Dx()-4*k, track.Dy()*0.5)
	c.slot(l, ctx, g, true)
	if on && !st.Disabled() {
		r := g.Dy()*0.5 - 2*k
		in := g.Inset(2 * k)
		if r > 0 && in.Dx() > 0 {
			ctx.DrawRoundRect(in, r, r, VGradient(in, c.progBevel...))
		}
	}
	hs := track.Dy()
	hx := track.Min.X
	if on {
		hx = track.Max.X - hs
	}
	c.knob(l, ctx, paintengine2d.XYWH(hx, track.Min.Y, hs, hs), st)
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.dis
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	}
}

// knob is the slider handle: a 21px round slab (light → dark), its
// outline, a hollow while held, the shadow and the hover / focus glow.
func (c *oxygen) knob(l *Classic, ctx *paintengine2d.Context, hb paintengine2d.Rect, st ControlState) {
	k := oxK(l)
	side := hb.Dx()
	if hb.Dy() < side {
		side = hb.Dy()
	}
	if side < 8*k {
		return
	}
	sc := side / (21 * k)
	ctr := hb.Center()
	s := c.slabAt(l, ctx, hb)
	r := 7.5 * k * sc
	c.bead(ctx, ctr, r, k, s, c.buttonGlow(st))
	top := ctr.Y - 7.5*k*sc
	ctx.DrawCircle(ctr, r, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, top), End: paintengine2d.Pt(0, top+18*k*sc), Stops: s.knob}))
	if st.Pressed() && !st.Disabled() {
		ctx.DrawCircle(ctr, 5.5*k*sc, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(0, top), End: paintengine2d.Pt(0, top+18*k*sc), Stops: s.knobSunk}))
	}
	ctx.DrawCircle(ctr, r-0.5*k*sc, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, top), End: paintengine2d.Pt(0, top+27*k*sc), Stops: s.knobLine},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k * sc, MiterLimit: 4}})
}

// DrawSlider is a 7px slot (no filled part — Oxygen sliders are
// uncoloured) and the round slider handle.
func (e oxygenEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := oxygenColors(l)
	k := oxK(l)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	hs := l.metrics.Thumb
	if hs <= 0 {
		hs = l.S(21)
	}
	if hs > b.Dy() {
		hs = b.Dy()
	}
	if b.Dx() < hs || hs < 8*k {
		return
	}
	cy := b.Min.Y + b.Dy()*0.5
	gt := 7 * k
	c.slot(l, ctx, paintengine2d.XYWH(b.Min.X+3*k, cy-gt*0.5, b.Dx()-6*k, gt), true)
	hx := b.Min.X + (b.Dx()-hs)*t
	c.knob(l, ctx, paintengine2d.XYWH(hx, cy-hs*0.5, hs, hs), st)
}

// DrawProgressBar is a slot groove and the glossy highlight bar; the busy
// bar is a 14%-wide bar sliding along the groove.
func (e oxygenEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := oxygenColors(l)
	k := oxK(l)
	g := paintengine2d.XYWH(b.Min.X+k, b.Min.Y, b.Dx()-2*k, b.Dy())
	if g.Dx() < 6*k || g.Dy() < 6*k {
		return
	}
	c.slot(l, ctx, g, g.Dy() < 10*k)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	in := g.Inset(k)
	var bar paintengine2d.Rect
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		// Bounce: left → right → left over one phase.
		p := phase * 2
		if p > 1 {
			p = 2 - p
		}
		w := in.Dx() * 0.14
		if w < in.Dy() {
			w = in.Dy()
		}
		bar = paintengine2d.XYWH(in.Min.X+(in.Dx()-w)*p, in.Min.Y, w, in.Dy())
	} else {
		w := in.Dx() * t
		if w < 1 {
			return
		}
		bar = paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy())
	}
	c.indicator(l, ctx, bar, st.Disabled())
}

// indicator is the progress bar: the highlight mixed with the dark window
// tone, glossed at its top and bottom edges, a bevel in light → highlight
// → dark and a light line along the top.
func (c *oxygen) indicator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, disabled bool) {
	k := oxK(l)
	if b.Dx() < 2*k || b.Dy() < 3*k {
		return
	}
	r := l.rx(2.5)
	if lim := b.Dx() * 0.5; r > lim {
		r = lim
	}
	op := float32(1)
	if disabled {
		op = 0.45
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Paint{Color: c.progBase, Style: paintengine2d.StyleFill, Opacity: op})
	ctx.DrawRoundRect(b.Inset(k), r, r, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y), Stops: c.progGloss},
		Style:  paintengine2d.StyleFill, Opacity: op})
	ctx.DrawRoundRect(b.Inset(0.5*k), r, r, paintengine2d.Paint{
		Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y), Stops: c.progBevel},
		Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}, Opacity: op})
	if b.Dx() > 4*k {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+r, b.Min.Y+k, b.Dx()-2*r, k), paintengine2d.Paint{Color: c.progTop.WithAlpha(0.8), Style: paintengine2d.StyleFill, Opacity: op})
	}
}

// DrawComboBox is a button slab, the text and Oxygen's etched down arrow.
func (e oxygenEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := oxygenColors(l)
	bst := st
	if open {
		bst |= StatePressed
	}
	c.button(l, ctx, b, bst, false)
	aw := l.S(20)
	ab := paintengine2d.XYWH(b.Max.X-aw-l.S(6), b.Min.Y, aw, b.Dy()-oxK(l))
	col := c.btnText
	if st.Disabled() {
		col = c.dis
	}
	oxArrow(l, ctx, ab, DirDown, col, c.slabAt(l, ctx, b).light, false)
	tb := paintengine2d.XYWH(b.Min.X+l.S(12), b.Min.Y, ab.Min.X-b.Min.X-l.S(14), b.Dy()-oxK(l))
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
}

// DrawSpinner is the spin box arrow column: a hole in the view colour with
// the etched up / down arrows (the hover colour when hot).
func (e oxygenEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 8*k || b.Dy() < 10*k {
		return
	}
	c.field(l, ctx, b, st&^StateFocused)
	mid := (b.Min.Y + b.Max.Y) * 0.5
	up := paintengine2d.XYWH(b.Min.X, b.Min.Y+2*k, b.Dx(), mid-b.Min.Y-2*k)
	dn := paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid-2*k)
	arrow := func(r paintengine2d.Rect, dir Direction, hot, press bool) {
		col := c.viewText
		switch {
		case st.Disabled():
			col = c.dis
		case press:
			col = c.focus
		case hot:
			col = c.hover
		}
		oxArrow(l, ctx, r, dir, col, paintengine2d.Color{}, true)
	}
	arrow(up, DirUp, upHover, upPress)
	arrow(dn, DirDown, downHover, downPress)
}

// DrawTabBar draws the top edge of the tab widget's slab frame, TabBar_
// BaseOverlap (7px) up into the bar; the bar is otherwise transparent.
func (e oxygenEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dy() < 8*k || b.Dx() < 16*k {
		return
	}
	pane := paintengine2d.XYWH(b.Min.X, oxPaneTop(l, b.Max.Y), b.Dx(), 40*k)
	c.paneSlab(l, ctx, pane, b, 0)
}

// DrawTab: the selected tab is a raised slab open at the bottom, washed
// light at the top and merging into the frame; unselected tabs are
// translucent dark tabs set 3px lower, rounded at the ends of the strip, and
// a hot one lights the frame edge beneath it with the hover glow.
func (e oxygenEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 12*k || b.Dy() < 12*k {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if selected {
		// Cover the frame's top edge under the tab with the window behind,
		// then the slab (its sides run down into the frame) and the wash.
		s := c.slabAt(l, ctx, b)
		under := paintengine2d.XYWH(b.Min.X+3*k, b.Max.Y-7*k, b.Dx()-6*k, 7*k)
		ctx.DrawRect(under, paintengine2d.Fill(c.winAt(l, ctx, b)))
		slab := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()+10*k)
		sb, r := oxE(l, slab)
		ctx.Save()
		ctx.ClipRect(b)
		c.shadow(ctx, sb, r, k, s.shadow, false)
		c.rim(l, ctx, sb, r, s)
		ctx.Restore()
		fr := b.Inset(4 * k)
		fr.Max.Y = b.Max.Y
		ctx.DrawRoundRect(fr, l.rx(2), l.rx(2), paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(0, fr.Min.Y), End: paintengine2d.Pt(0, fr.Max.Y), Stops: c.tabFill}))
		lb := paintengine2d.XYWH(b.Min.X, b.Min.Y+2*k, b.Dx(), b.Dy()-7*k)
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(10))
		if st.Focused() && !st.Disabled() {
			e.DrawFocusRing(l, ctx, lb.Inset(4*k))
		}
		return
	}
	tr := paintengine2d.XYWH(b.Min.X, b.Min.Y+3*k, b.Dx(), b.Dy()-7*k)
	rad := l.rx(4)
	tl, trr := float32(0), float32(0)
	if st.First() {
		tl = rad
	}
	if st.Last() {
		trr = rad
	}
	path := RoundRectPath(tr.Inset(0.5*k), tl, trr, 0, 0)
	ctx.DrawPath(path, paintengine2d.Fill(c.tabMid))
	ctx.DrawPath(path, paintengine2d.StrokePaint(c.tabDark, k))
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		// The hover slab's glow along the frame edge under the tab.
		y := b.Max.Y - 3.5*k
		ctx.DrawRect(paintengine2d.XYWH(tr.Min.X+2*k, y, tr.Dx()-4*k, 1.3*k), paintengine2d.Fill(c.hover))
		ctx.DrawRect(paintengine2d.XYWH(tr.Min.X+2*k, y-1.2*k, tr.Dx()-4*k, 1.2*k), paintengine2d.Fill(c.hover.WithAlpha(0.35)))
	}
	l.drawFittedText(ctx, l.body, label, tr, fg, AlignCenter, l.S(10))
}

// DrawMenuBar: transparent over the window gradient.
func (e oxygenEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {}

// DrawMenuTitle: the hot or open title is a flat well in the window's mid
// colour.
func (e oxygenEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := oxygenColors(l)
	k := oxK(l)
	if (open || st.Pressed() || st.Hovered()) && !st.Disabled() {
		c.flatWell(l, ctx, b.Inset(k), c.winMid)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		e.DrawFocusRing(l, ctx, b)
	}
}

// DrawMenuFrame is the menu gradient (200px split) inside the rounded
// popup frame.
func (e oxygenEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() < 6*k || b.Dy() < 6*k {
		return
	}
	r := l.rx(5)
	split := l.S(200)
	if v := b.Dy() * 0.75; v < split {
		split = v
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.bottom))
	ctx.Save()
	ctx.ClipRoundRect(b, r, r)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), split), paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Min.Y+split), Stops: c.menuStops}))
	ctx.Restore()
	c.popupFrame(l, ctx, b)
}

// DrawMenuItem is the flat dark well under the hot row, Oxygen checks (a
// tick in a flat well) and radio beads, the etched separator and submenu
// arrow; labels keep the window text.
func (e oxygenEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := oxygenColors(l)
	k := oxK(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		x0 := b.Min.X - ch.PadL + l.S(4)
		x1 := b.Max.X + ch.PadR - l.S(4)
		c.separator(l, ctx, paintengine2d.XYWH(x0, b.Min.Y, x1-x0, b.Dy()), false)
		return
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		hb := l.menuItemHighlightBounds(b)
		c.flatWell(l, ctx, paintengine2d.XYWH(hb.Min.X, hb.Min.Y+k, hb.Dx()-k, hb.Dy()-k), c.menuHot)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gw := ch.CheckCol()
	side := l.S(21)
	if side > b.Dy() {
		side = b.Dy()
	}
	ib := paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	ist := StateNone
	if st.Disabled() {
		ist = StateDisabled
	}
	switch {
	case row.Radio:
		e.RadioIndicator(l, ctx, ib, ist, row.Checked)
	case row.Checked:
		// A flat well with the tick.
		c.flatWell(l, ctx, ib.Inset(3*k), c.winAt(l, ctx, ib))
		e.menuTick(l, ctx, ib, fg)
	case row.Icon != IconNone:
		l.drawToolIcon(ctx, ib.Inset(l.S(2)), row.Icon, fg)
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow, b.Dy())
		oxArrow(l, ctx, ab, DirRight, fg, c.winLight, false)
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

// menuTick is the check box tick without its slab.
func (e oxygenEngine) menuTick(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	c := oxygenColors(l)
	k := oxK(l)
	body := b.Dx() - 6*k
	if body < 8*k {
		body = b.Dx()
	}
	oxTick(ctx, b.Center(), body, k, c.winLight.WithAlpha(col.A))
	oxTick(ctx, b.Center(), body, 0, col)
}

// row paints a list / tree / table selection (a rounded highlight
// gradient with a 1px border; hover is the highlight at 20%) and returns
// the label colour. KDE keeps the selection when only the
// view loses focus; an inactive window (Backdrop) shows the scheme's
// inactive selection colour.
func (c *oxygen) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	k := oxK(l)
	selected, hovered := st.Checked(), st.Hovered() && !st.Disabled()
	fg := c.viewText
	if st.Disabled() {
		fg = c.dis
	}
	if !selected && !hovered || b.Dx() < 4*k || b.Dy() < 4*k {
		return fg
	}
	r := l.rx(3)
	rb := b.Inset(0.5 * k)
	if !selected {
		ctx.DrawRoundRect(rb, r, r, VGradient(rb, c.selHotStops...))
		ctx.DrawRoundRect(rb.Inset(0.5*k), r-0.5*k, r-0.5*k, paintengine2d.StrokePaint(c.selHot, k))
		return fg
	}
	stops, line := c.selStops, c.hl
	switch {
	case st.Disabled():
		stops, line = c.selDisStops, c.hlDis
	case st.Backdrop():
		stops, line = c.selOffStops, c.hlOff
	}
	ctx.DrawRoundRect(rb, r, r, VGradient(rb, stops...))
	if hovered {
		ctx.DrawRoundRect(rb, r, r, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.12)))
	}
	ctx.DrawRoundRect(rb.Inset(0.5*k), r-0.5*k, r-0.5*k, paintengine2d.StrokePaint(line, k))
	return c.hlText
}

func (e oxygenEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := oxygenColors(l)
	fg := c.row(l, ctx, b, st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow: the rounded selection, dotted branch lines and chevrons.
func (e oxygenEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := oxygenColors(l)
	u := oxU(l)
	fg := c.row(l, ctx, b, st)
	selected := st.Checked()
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	aw := l.S(14)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	line := Mix(c.base, c.viewText, 0.25)
	for d := 0; d < depth; d++ {
		gx := snap(b.Min.X + pad + float32(d)*indent + aw*0.5)
		fuVLine(ctx, gx, b.Min.Y, b.Max.Y, u, line)
	}
	if depth > 0 {
		gx := snap(b.Min.X + pad + float32(depth-1)*indent + aw*0.5)
		fuHLine(ctx, gx, x+aw*0.25, cy, u, line)
	}
	if !leaf {
		col := c.viewText
		if selected {
			col = c.hlText
		}
		dir := DirRight
		if expanded {
			dir = DirDown
		}
		oxArrow(l, ctx, paintengine2d.XYWH(x, b.Min.Y, aw, b.Dy()), dir, col, paintengine2d.Color{}, true)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(x+aw+l.S(4), b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e oxygenEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := oxygenColors(l)
	u := oxU(l)
	fg := c.viewText
	if st.Disabled() {
		fg = c.dis
	}
	switch {
	case st.Checked():
		stops := c.selStops
		switch {
		case st.Disabled():
			stops = c.selDisStops
		case st.Backdrop():
			stops = c.selOffStops
		}
		ctx.DrawRect(b, VGradient(b, stops...))
		fg = c.hlText
	case st.Hovered() && !st.Disabled():
		ctx.DrawRect(b, paintengine2d.Fill(c.selHot))
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
	ctx.ClipRect(b.Inset(u))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	grid := Mix(c.base, c.viewText, 0.12)
	fuVLine(ctx, b.Max.X-u, b.Min.Y, b.Max.Y, u, grid)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, grid)
}

// dot is a light dot under a dark one, the header and grip mark.
func (c *oxygen) dot(ctx *paintengine2d.Context, x, y, k float32) {
	d := 1.8 * k
	ctx.DrawCircle(paintengine2d.Pt(x+0.5*k, y+0.5*k), d*0.5, paintengine2d.Fill(c.winLight))
	ctx.DrawCircle(paintengine2d.Pt(x, y), d*0.5, paintengine2d.Fill(darkerPct(c.winDark, 130)))
}

// DrawTableHeader is the window gradient (transparent here) with a dark /
// light line pair below, three dots as the section separator and the
// etched sort arrow (the hover colour when hot).
func (e oxygenEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := oxygenColors(l)
	u := oxU(l)
	k := oxK(l)
	if st.Pressed() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.winDark.WithAlpha(0.25)))
	}
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-u, u, c.winDark)
	fuHLine(ctx, b.Min.X, b.Max.X, b.Max.Y-2*u, u, c.winLight)
	if !st.Last() && b.Dy() > 10*k {
		cy := (b.Min.Y + b.Max.Y) * 0.5
		x := b.Max.X - 2*k
		for _, dy := range [3]float32{-3, 0, 3} {
			c.dot(ctx, x, cy+dy*k, k)
		}
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
		col := c.text
		if st.Hovered() && !st.Disabled() {
			col = c.hover
		}
		oxArrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(8), b.Min.Y+k, aw, b.Dy()), dir, col, c.winLight, false)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(14)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar: transparent over the window gradient.
func (e oxygenEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {}

// DrawStatusBar: the window gradient with plain texts.
func (e oxygenEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := oxygenColors(l)
	if n := len(parts); n > 0 {
		slot := b.Dx() / float32(n)
		for i, s := range parts {
			x := b.Min.X + slot*float32(i)
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(10), b.Min.Y, slot-l.S(16), b.Dy()), c.text, AlignStart, 0)
		}
	}
}

// DrawTitleBar (panel headings): bold text over the gradient with Oxygen's
// fading separator below.
func (e oxygenEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := oxygenColors(l)
	k := oxK(l)
	c.separator(l, ctx, paintengine2d.XYWH(b.Min.X, b.Max.Y-4*k, b.Dx(), 4*k), false)
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*k), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*k), l.palette.TextMuted, AlignStart, 0)
	}
}

// DrawAccordionHeader is a tool box tab: a flat hot well, the chevron and
// the label.
func (e oxygenEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := oxygenColors(l)
	k := oxK(l)
	switch {
	case st.Disabled():
	case st.Pressed():
		c.flatWell(l, ctx, b.Inset(k), c.winMid)
	case st.Hovered():
		r := l.rx(3)
		ctx.DrawRoundRect(b.Inset(1.5*k), r, r, paintengine2d.StrokePaint(c.hover, k))
	}
	if st.Focused() && !st.Disabled() && !st.Hovered() {
		e.DrawFocusRing(l, ctx, b)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	oxArrow(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, l.S(14), b.Dy()), dir, fg, c.winLight, false)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(26), b.Min.Y, b.Dx()-l.S(30), b.Dy()), fg, AlignStart, 0)
}

// separator is a dark line over a light one, both fading out at the ends.
func (c *oxygen) separator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	u := oxU(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		if b.Dy() < 2 {
			return
		}
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, u, b.Dy()), VGradient(b, c.sepLight...))
		ctx.DrawRect(paintengine2d.XYWH(x+u, b.Min.Y, u, b.Dy()), VGradient(b, c.sepDark...))
		ctx.DrawRect(paintengine2d.XYWH(x+2*u, b.Min.Y, u, b.Dy()), VGradient(b, c.sepLight...))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	if b.Dx() < 2 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), HGradient(b, c.sepDark...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+u, b.Dx(), u), HGradient(b, c.sepLight...))
}

func (e oxygenEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := oxygenColors(l)
	if vertical {
		m := l.S(2)
		c.separator(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+m, b.Dx(), b.Dy()-2*m), true)
		return
	}
	c.separator(l, ctx, b, false)
}

// DrawSplitter is three dots across the handle's middle, the hover glow
// washed over it when hot.
func (e oxygenEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := oxygenColors(l)
	k := oxK(l)
	if st.Hovered() && !st.Disabled() {
		r := l.rx(2)
		ctx.DrawRoundRect(b.Inset(k), r, r, paintengine2d.Fill(c.hover.WithAlpha(0.3)))
	}
	ctr := b.Center()
	for _, d := range [3]float32{-3, 0, 3} {
		x, y := ctr.X, ctr.Y+d*k
		if !vertical {
			x, y = ctr.X+d*k, ctr.Y
		}
		if b.Contains(paintengine2d.Pt(x+k, y+k)) && b.Contains(paintengine2d.Pt(x-k, y-k)) {
			c.dot(ctx, x, y, k)
		}
	}
}

// DrawTooltip is the tooltip colour's top → bottom tone gradient, rounded,
// with a light rim.
func (e oxygenEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := oxygenColors(l)
	k := oxK(l)
	if b.Dx() > 4*k && b.Dy() > 4*k {
		in := b.Inset(0.5 * k)
		r := l.rx(4)
		ctx.DrawRoundRect(in, r, r, VGradient(b, c.tipStops...))
		ctx.DrawRoundRect(in, r-0.5*k, r-0.5*k, paintengine2d.Paint{
			Shader: paintengine2d.LinearGradient{Start: paintengine2d.Pt(0, b.Min.Y), End: paintengine2d.Pt(0, b.Max.Y), Stops: c.tipEdge},
			Style:  paintengine2d.StyleStroke, Stroke: paintengine2d.Stroke{Width: k, MiterLimit: 4}})
	}
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(6)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs -----------------------------------------------------------------------------------------------------

func oxygenPacks() []ThemePack {
	// The Oxygen colour scheme's values.
	win, text := Hex("#d6d2d0"), Hex("#221f1e")
	view, viewText := Hex("#ffffff"), Hex("#1f1c1b")
	btn := Hex("#dfdcd9")
	sel := Hex("#43ace8")
	h := oxTones{k: 1}
	pal := Palette{
		Background: win, Surface: win, SurfaceAlt: btn,
		Border: h.dark(win), Divider: h.dark(win),
		Text: text, TextMuted: Hex("#898887"), TextOnAccent: Hex("#ffffff"),
		Accent: sel, AccentHover: Hex("#6ed6ff"), AccentPress: Hex("#3aa7dd"),
		Field: view, FieldBorder: h.dark(win),
		Focus: Hex("#3aa7dd"), Selection: sel,
		Track: h.dark(win), Thumb: btn,
		Highlight: sel.WithAlpha(0.2), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: h.mid(win), MenuHoverBorder: h.dark(win), MenuGutter: win,
		Danger: Hex("#bf0303"), Success: Hex("#006e28"), Warning: Hex("#b08000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: h.light(win), BevelDark: h.dark(win),
	}
	extra := map[string]paintengine2d.Color{
		"button": btn, "buttonText": Hex("#221f1e"),
		"view": view, "viewText": viewText,
		"tip": Hex("#181513"), "tipText": Hex("#e7fdff"),
		"focus": Hex("#3aa7dd"), "hover": Hex("#6ed6ff"),
	}
	tok := ThemeTokens{
		Engine:  "oxygen",
		Bevel:   BevelSoftShadow,
		Family:  ThemeLight,
		Palette: pal,
		Era:     "Oxygen",
		Extra:   extra,
		Params:  map[string]float32{"contrast": 7},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: sel, Border: sel}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.15), Border: pal.Focus}
	return []ThemePack{{
		Name: "oxygen", Label: "Oxygen", Year: 2008, Lineage: "KDE",
		Summary: "KDE 4's Oxygen: glossy slabs with soft shadows, the blue hover glow and a window-wide gradient.",
		Era:     "Oxygen", Palette: ThemeLight, Tokens: tok,
	}}
}
