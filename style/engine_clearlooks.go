package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// clearlooksEngine paints GTK 2's Clearlooks as GNOME 2.12 shipped it
// (2005), and the two looks that grew from it: Ubuntu's Human (ubuntulooks)
// and Qt 4's Cleanlooks.
//
// Clearlooks shades everything from two colours. The background gives the
// shade table (lightness and saturation × 1.065, 0.93, 0.896, 0.85, 0.768,
// 0.665, 0.4, 0.205 in HLS, stretched by the "contrast" option) and the
// button borders (× 0.5 at the top, × 0.62 at the bottom); the selection
// gives the three spot colours (× 1.42, 1.05, 0.65). Buttons are rounded
// 3px, bordered in a vertical border gradient, filled with three bands
// (× 1.055 → 1.005 over the top quarter, → 0.98, → 0.91 over the bottom
// quarter), lit by a 1px white inner edge top and left, and sit in a 1px
// engraved ring (× 0.96 above, × 1.055 below). The progress bar is the
// "candy bar": the selection easing darker to the bottom under 45° stripes,
// in a spot-3 border. Scroll bar sliders carry three grip lines; the
// selected tab wears a 3px stripe of spot colour along its top; check boxes
// are 13px wells with a bold dark tick; menu items and selected rows are
// the selection easing to × 0.8; focus is a dotted rectangle.
//
// Pack data:
//
//	params  "flavour"   0 Clearlooks, 1 Human (ubuntulooks), 2 Cleanlooks
//	        "contrast"  stretches the shade table (default 1)
//	extra   "prelight", "active" bg[PRELIGHT] / bg[ACTIVE]; "listActive"
//	        the backdrop selection (base[ACTIVE]); "menu", "notebook" those
//	        backgrounds; "check", "checkHot" the tick colours; "spot" the
//	        widget spot colour (Human's orange); "tooltip", "tooltipText";
//	        "caption", "captionText" the in-app title bar.
type clearlooksEngine struct{ BaseEngine }

func init() {
	RegisterEngine(clearlooksEngine{})
	for _, p := range clearlooksPacks() {
		RegisterPack(p)
	}
}

func (clearlooksEngine) ID() string { return "clearlooks" }

// DefaultMetrics are GTK 2's Clearlooks proportions at the toolkit's 16px
// UI font (GNOME 2 used 10pt, about 13px): 15px scroll bars and steppers,
// 13px check boxes, 3px corners.
func (clearlooksEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    3, RadiusSmall: 3,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 13, Radio: 13,
		MenuItemH: 26, MenuBarH: 28, TabH: 30, RowH: 24,
		TitleBar: 28, HeaderH: 26, ProgressH: 18, SliderH: 26, Thumb: 13,
		Scroll: 15, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 17, SwitchW: 44, SwitchH: 22,
	}
}

// Flavours of the engine (the "flavour" param).
const (
	clClassic    = 0 // Clearlooks, GNOME 2.12
	clHuman      = 1 // Ubuntu Human (ubuntulooks)
	clCleanlooks = 2 // Qt 4 Cleanlooks
)

// clShadeFactors is the Clearlooks shade table.
var clShadeFactors = [8]float32{1.065, 0.93, 0.896, 0.85, 0.768, 0.665, 0.4, 0.205}

// ---- the shading maths ---------------------------------------------------------------

// clShade is Clearlooks' shade: in HLS, lightness and saturation are both
// multiplied by k and clamped to [0, 1]; hue and alpha are kept.
func clShade(c paintengine2d.Color, k float32) paintengine2d.Color {
	h, lt, s := clRGBToHLS(c.R, c.G, c.B)
	lt = clamp1(lt * k)
	s = clamp1(s * k)
	r, g, b := clHLSToRGB(h, lt, s)
	return paintengine2d.RGBA(r, g, b, c.A)
}

// clRGBToHLS is the textbook RGB → hue (degrees), lightness, saturation.
func clRGBToHLS(r, g, b float32) (h, l, s float32) {
	mx := max(r, g, b)
	mn := min(r, g, b)
	l = (mx + mn) * 0.5
	if mx == mn {
		return 0, l, 0
	}
	d := mx - mn
	if l <= 0.5 {
		s = d / (mx + mn)
	} else {
		s = d / (2 - mx - mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
	case g:
		h = 2 + (b-r)/d
	default:
		h = 4 + (r-g)/d
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, l, s
}

// clHLSToRGB inverts clRGBToHLS.
func clHLSToRGB(h, l, s float32) (r, g, b float32) {
	if s == 0 {
		return l, l, l
	}
	var m2 float32
	if l <= 0.5 {
		m2 = l * (1 + s)
	} else {
		m2 = l + s - l*s
	}
	m1 := 2*l - m2
	ch := func(hue float32) float32 {
		for hue > 360 {
			hue -= 360
		}
		for hue < 0 {
			hue += 360
		}
		switch {
		case hue < 60:
			return m1 + (m2-m1)*hue/60
		case hue < 180:
			return m2
		case hue < 240:
			return m1 + (m2-m1)*(240-hue)/60
		}
		return m1
	}
	return ch(h + 120), ch(h), ch(h - 120)
}

// ---- resolved colour set ----------------------------------------------------------

// clSet is a look's resolved Clearlooks colours, built once per look.
type clSet struct {
	flavour int

	bg, fg, base, text, sel, selFg                 paintengine2d.Color
	prelight, active, disFg, listActive            paintengine2d.Color
	menu, notebook, check, checkHot, spotBase      paintengine2d.Color
	tip, tipText, caption, captionText, captionOff paintengine2d.Color

	s                  [8]paintengine2d.Color // shades of bg
	spot               [3]paintengine2d.Color // spot1..3
	upper, lower       paintengine2d.Color    // button border gradient
	aUpper, aLower     paintengine2d.Color    // pressed border gradient
	mid, white         paintengine2d.Color
	insetDk, insetLt   paintengine2d.Color // the engraved ring
	nbUpper, nbLower   paintengine2d.Color // notebook (tab) borders
	menuBorder, menuS1 paintengine2d.Color
	focus              paintengine2d.Color

	bands, bandsHot, bandsDis, press []paintengine2d.GradientStop
	sliderFace, sliderHot, sliderDn  []paintengine2d.GradientStop
	candy, menuItem, rowSel, rowOff  []paintengine2d.GradientStop
	tabSel, tabOff, insetRing        []paintengine2d.GradientStop
	scaleFill, scaleKnob, menubar    []paintengine2d.GradientStop
	capGrad, capOff, capBtn          []paintengine2d.GradientStop
	capHot, capDown, capOffBtn       []paintengine2d.GradientStop
	border, aBorder, nbBorder        []paintengine2d.GradientStop // border gradients
	tabHot, spin, dot, dotOn         []paintengine2d.GradientStop
	hdr                              [3][]paintengine2d.GradientStop // header: normal, hot, pressed
	radioShade                       [3][]paintengine2d.GradientStop // radio inner shadow: base, bg, active
	candyHi, rowSelFg                paintengine2d.Color
	dotBody, dotHi, dotLo            paintengine2d.Color

	// Human: the orange widget spot, glow and glass.
	glow, glowEdge                   paintengine2d.Color
	cellGrad, chkOn, glass, glassHot []paintengine2d.GradientStop

	// Cleanlooks: the Qt materials.
	qOutline, qTopEdge, qGroove, qGrip, qDisOutline paintengine2d.Color
	qBtn, qBtnHot, qScroll, qScrollHot, qArrow      []paintengine2d.GradientStop
	qProg                                           []paintengine2d.GradientStop
	qProgEdge, qProgHi, qTabTop, qTabMid            paintengine2d.Color
	qCheckEdge, qFocus, qDefRing, qHdrLow           paintengine2d.Color
	qEdge, qCap, qTrough, qSpin                     []paintengine2d.GradientStop
}

type clKey struct{}

// clColors is the look's resolved colour set (built once per look).
func clColors(l *Classic) *clSet {
	return l.Memo(clKey{}, func() any { return clBuild(l) }).(*clSet)
}

func clBuild(l *Classic) *clSet {
	p := l.palette
	c := &clSet{flavour: int(l.P("flavour", clClassic))}
	if c.flavour < clClassic || c.flavour > clCleanlooks {
		c.flavour = clClassic
	}
	// (Not l.fieldText: Memo does not nest.)
	c.bg, c.fg, c.base = p.Background, p.Text, p.Field
	c.text = ReadableOn(c.base, 4.5, p.Text)
	c.sel = p.Selection
	if c.sel.A < 0.9 {
		c.sel = p.Accent
	}
	c.sel.A = 1
	// Selected text is text[SELECTED] unless it would not read (3:1, as
	// Luna rules for highlight text): Clearlooks' white on #628cb2 is 3.6.
	c.selFg = ReadableOn(c.sel, 3, p.TextOnAccent, p.Text)
	c.prelight = l.X("prelight", clShade(c.bg, 1.02))
	c.active = l.X("active", clShade(c.bg, 0.9))
	c.disFg = p.TextMuted
	c.listActive = l.X("listActive", Mix(c.sel, c.bg, 0.55))
	c.menu = l.X("menu", clShade(c.bg, 1.03))
	c.notebook = l.X("notebook", c.bg)
	c.check = l.X("check", c.text)
	c.checkHot = l.X("checkHot", c.check)
	c.spotBase = l.X("spot", c.sel)
	c.tip = l.X("tooltip", Hex("#ffffbf"))
	c.tipText = l.X("tooltipText", Hex("#000000"))
	c.white = Hex("#ffffff")

	contrast := l.P("contrast", 1)
	for i, f := range clShadeFactors {
		c.s[i] = clShade(c.bg, (f-0.7)*contrast+0.7)
	}
	c.spot = [3]paintengine2d.Color{clShade(c.spotBase, 1.42), clShade(c.spotBase, 1.05), clShade(c.spotBase, 0.65)}
	c.upper, c.lower = clShade(c.bg, 0.5), clShade(c.bg, 0.62)
	c.aUpper, c.aLower = clShade(c.active, 0.5), clShade(c.active, 0.55)
	c.nbUpper, c.nbLower = clShade(c.notebook, 0.5), clShade(c.notebook, 0.62)
	c.mid = Mix(clShade(c.bg, 1.3), clShade(c.bg, 0.7), 0.5)
	c.insetDk, c.insetLt = clShade(c.bg, 0.96), clShade(c.bg, 1.055)
	c.menuBorder, c.menuS1 = clShade(c.menu, 0.5), clShade(c.menu, 0.93)
	c.focus = c.s[6]

	band := func(bg paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{
			Stop(0, clShade(bg, 1.055)), Stop(0.25, clShade(bg, 1.005)),
			Stop(0.75, clShade(bg, 0.98)), Stop(1, clShade(bg, 0.91)),
		}
	}
	two := func(a, b paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, a), Stop(1, b)}
	}
	c.bands, c.bandsHot, c.bandsDis = band(c.bg), band(c.prelight), band(c.bg)
	c.press = two(c.active, clShade(c.active, 0.94))
	c.insetRing = two(c.insetDk, c.insetLt)
	c.sliderFace = two(clShade(c.bg, 1.055), clShade(c.bg, 0.96))
	c.sliderHot = two(clShade(c.prelight, 1.055), clShade(c.prelight, 0.96))
	c.sliderDn = two(clShade(c.active, 1.055), clShade(c.active, 0.96))
	c.candy = two(c.spot[1], clShade(c.spot[1], 0.9))
	c.candyHi = clShade(c.spot[1], 1.2)
	c.menuItem = two(c.sel, clShade(c.sel, 0.8))
	c.rowSel = two(c.sel, clShade(c.sel, 0.8))
	c.rowOff = two(c.listActive, clShade(c.listActive, 0.8))
	c.rowSelFg = c.selFg
	c.tabSel = two(clShade(c.notebook, 1.04), c.notebook)
	c.tabOff = two(clShade(c.active, 1.07), c.active)
	c.scaleFill = two(c.spotBase, clShade(c.spotBase, 1.3))
	c.scaleKnob = two(c.bg, c.s[2])
	c.menubar = two(c.bg, clShade(c.bg, 0.95))
	// The dot of a radio: the tick colour, sphere-shaded.
	c.dotBody, c.dotHi, c.dotLo = clShade(c.check, 0.86), clShade(c.check, 1.9), clShade(c.check, 0.6)

	// The in-app title bar is Metacity's Clearlooks: three bands of the
	// selection (× 1.02 → 0.92, → 0.88, → 0.8) under white text.
	cap := l.X("caption", c.sel)
	c.caption = cap
	c.captionText = l.X("captionText", Hex("#ffffff"))
	c.captionOff = c.fg
	c.capGrad = []paintengine2d.GradientStop{
		Stop(0, clShade(cap, 1.02)), Stop(0.25, clShade(cap, 0.92)), Stop(0.75, clShade(cap, 0.88)), Stop(1, clShade(cap, 0.8)),
	}
	c.capOff = []paintengine2d.GradientStop{
		Stop(0, clShade(c.bg, 1.01)), Stop(0.25, clShade(c.bg, 0.96)), Stop(0.75, clShade(c.bg, 0.94)), Stop(1, clShade(c.bg, 0.89)),
	}
	c.capBtn = []paintengine2d.GradientStop{
		Stop(0, clShade(cap, 1.1)), Stop(0.5, clShade(cap, 1.02)), Stop(0.5, clShade(cap, 1.0)), Stop(1, clShade(cap, 0.92)),
	}
	c.capHot = two(clShade(cap, 1.25), clShade(cap, 1.05))
	c.capDown = two(clShade(cap, 0.8), clShade(cap, 0.95))
	c.capOffBtn = two(clShade(c.bg, 1.1), clShade(c.bg, 0.92))
	c.border, c.aBorder, c.nbBorder = two(c.upper, c.lower), two(c.aUpper, c.aLower), two(c.nbUpper, c.nbLower)
	c.tabHot = two(clShade(c.prelight, 1.02), c.bg)
	c.spin = two(c.bg, clShade(c.bg, 0.93))
	for i, base := range [...]paintengine2d.Color{c.bg, c.prelight, c.active} {
		c.hdr[i] = []paintengine2d.GradientStop{Stop(0, base), Stop(0.66, base), Stop(1, clShade(base, 0.96))}
	}

	switch c.flavour {
	case clHuman:
		// ubuntulooks: glass buttons, the orange glow, orange cells and
		// checks, a gradient menu highlight with black text.
		o := c.spotBase
		c.glow = clShade(o, 1.5)
		c.glowEdge = clShade(o, 0.5)
		glass := func(bg paintengine2d.Color) []paintengine2d.GradientStop {
			return []paintengine2d.GradientStop{
				Stop(0, clShade(bg, 1.05)), Stop(0.5, clShade(bg, 1.02)), Stop(0.5, clShade(bg, 0.98)),
				Stop(0.85, bg), Stop(1, clShade(bg, 1.02)),
			}
		}
		c.glass, c.glassHot = glass(c.bg), glass(c.prelight)
		c.cellGrad = glass(clShade(o, 1.05))
		c.chkOn = two(clShade(o, 1.25), clShade(o, 0.95))
		c.menuItem = two(l.X("menuHi", Hex("#ffdfad")), l.X("menuHi2", Hex("#f4c378")))
		c.rowSel = two(c.sel, clShade(c.sel, 0.93))
		c.scaleFill = two(clShade(o, 1.1), clShade(o, 0.95))
	case clCleanlooks:
		// Qt's QCleanlooksStyle: its own gradients and outlines, sampled
		// from the Button colour with QColor lighter / darker.
		c.qOutline = l.X("outline", Hex("#8f8882"))
		c.qTopEdge = l.X("topEdge", Hex("#827c76"))
		c.qDisOutline = Hex("#c0bbb5")
		c.qGroove = l.X("groove", Hex("#d7cfc6"))
		c.qGrip = Hex("#c4bcb4")
		c.qBtn = []paintengine2d.GradientStop{Stop(0, Hex("#fffdfa")), Stop(0.14, Hex("#f9f5f1")), Stop(0.86, Hex("#efebe7")), Stop(1, Hex("#e5dfd8"))}
		c.qBtnHot = []paintengine2d.GradientStop{Stop(0, Hex("#fffcf8")), Stop(0.14, Hex("#fffefd")), Stop(0.86, Hex("#fbf7f3")), Stop(1, Hex("#e9e5e1"))}
		c.qScroll = two(Hex("#fffdfa"), Hex("#e6e1db"))
		c.qScrollHot = two(Hex("#ffffff"), Hex("#fdf8f1"))
		c.qArrow = two(Hex("#ffffff"), Hex("#e6e1db"))
		// The bar's gradient spreads over twice its height: the bottom
		// shows the half-way colour.
		c.qProg = two(c.sel, Mix(c.sel, Hex("#7fb6e8"), 0.5))
		c.qProgEdge, c.qProgHi = Hex("#46647f"), Hex("#76a8d6")
		c.qTabTop, c.qTabMid = Hex("#415d77"), c.sel
		c.qCheckEdge = Hex("#b8b2ad")
		c.qFocus = Hex("#8b8783")
		c.qDefRing = paintengine2d.RGBA(0xb5/255.0, 0xb2/255.0, 0xaf/255.0, 0.5)
		c.qHdrLow = Hex("#e5ded7")
		c.menuItem = two(c.sel, Hex("#4e708e"))
		c.rowSel = two(c.sel, c.sel)
		c.rowOff = two(c.listActive, c.listActive)
		c.tabSel = two(Hex("#f9f5f1"), Hex("#e9e4df"))
		c.tabOff = two(c.bg, Hex("#c9c3bd"))
		c.menubar = two(c.bg, Mix(c.bg, Hex("#dad6d2"), 0.5))
		c.scaleFill = two(Hex("#517594"), Hex("#aad7ff"))
		c.scaleKnob = two(Hex("#fefcf9"), Hex("#eceae7"))
		c.focus = c.qFocus
		c.dotBody, c.dotHi, c.dotLo = Hex("#3b4349"), Hex("#596066"), Hex("#1f262b")
		c.qEdge = two(c.qOutline, c.qOutline)
		c.qCap = two(Hex("#cee8ff"), Hex("#6c9ac4"))
		c.qTrough = two(Hex("#c4bcb4"), Hex("#ede4da"))
		c.qSpin = two(Hex("#f1eeec"), Hex("#e6e1db"))
		for i := range c.hdr {
			c.hdr[i][2] = Stop(1, c.qHdrLow)
		}
	}
	// The radio's inner shadow and sphere-shaded dot, per well colour.
	edge := c.s[5]
	if c.flavour == clCleanlooks {
		edge = clRadioEdge
	}
	for i, fill := range [...]paintengine2d.Color{c.base, c.bg, c.active} {
		c.radioShade[i] = two(fill.WithAlpha(0), edge.WithAlpha(0.35))
	}
	c.dot = []paintengine2d.GradientStop{Stop(0, c.dotHi), Stop(0.45, c.dotBody), Stop(1, c.dotLo)}
	c.dotOn = []paintengine2d.GradientStop{Stop(0, Hex("#505050")), Stop(0.45, Hex("#1a1a1a")), Stop(1, Hex("#000000"))}
	return c
}

// clRadioEdge is Cleanlooks' warm grey radio ring.
var clRadioEdge = Hex("#b0a69b")

// ---- drawing helpers ----------------------------------------------------------------

// clPx is one line of the look: 1 device pixel at 1x, 2 at 2x.
func clPx(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// clSnap puts a rect on the pixel grid so hairlines stay crisp.
func clSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

// clR is design radius r (1x px) for b, at most half its short side.
func clR(l *Classic, r float32, b paintengine2d.Rect) float32 {
	return max(min(l.rx(r), b.Dx()*0.5, b.Dy()*0.5), 0)
}

// clFrame fills b with edge, then its lw inset with fill (both paints).
func clFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, edge, fill paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	ctx.DrawRoundRect(b, r, r, edge)
	if b.Dx() > 2*lw && b.Dy() > 2*lw {
		ri := max(r-lw, 0)
		ctx.DrawRoundRect(b.Inset(lw), ri, ri, fill)
	}
}

// clLit draws the 1px inner edge of a face: hi along the top and left, lo
// along the bottom and right, clipped to the face's rounded corners.
func clLit(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, hi, lo paintengine2d.Color) {
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	ctx.Save()
	ctx.ClipRoundRect(b, r, r)
	if lo.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(lo))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(lo))
	}
	if hi.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-lw, lw), paintengine2d.Fill(hi))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, lw, b.Dy()-lw), paintengine2d.Fill(hi))
	}
	ctx.Restore()
}

// clDotted is GTK's focus rectangle: dots of one line, one line apart,
// just inside b, batched into one path.
func clDotted(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, col paintengine2d.Color) {
	b = clSnap(b)
	if b.Dx() < 3*lw || b.Dy() < 3*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	for x := b.Min.X; x+lw <= b.Max.X; x += 2 * lw {
		p.AddRect(paintengine2d.XYWH(x, b.Min.Y, lw, lw))
		p.AddRect(paintengine2d.XYWH(x, b.Max.Y-lw, lw, lw))
	}
	for y := b.Min.Y + 2*lw; y+lw <= b.Max.Y-lw; y += 2 * lw {
		p.AddRect(paintengine2d.XYWH(b.Min.X, y, lw, lw))
		p.AddRect(paintengine2d.XYWH(b.Max.X-lw, y, lw, lw))
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// clTriangle fills GTK 2's arrow: a solid triangle whose rows narrow by two
// pixels, fitted into b (w odd, h = w/2 + 1).
func clTriangle(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color, w float32) {
	if b.Empty() || col.A <= 0 || w < 3 {
		return
	}
	h := snap(w*0.5 + 0.5)
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	var r paintengine2d.Rect
	if dir == DirLeft || dir == DirRight {
		r = paintengine2d.XYWH(snap(cx-h*0.5), snap(cy-w*0.5), h, w)
	} else {
		r = paintengine2d.XYWH(snap(cx-w*0.5), snap(cy-h*0.5), w, h)
	}
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(r.Min.X, r.Max.Y)
		p.LineTo(r.Max.X, r.Max.Y)
		p.LineTo((r.Min.X+r.Max.X)*0.5, r.Min.Y)
	case DirDown:
		p.MoveTo(r.Min.X, r.Min.Y)
		p.LineTo(r.Max.X, r.Min.Y)
		p.LineTo((r.Min.X+r.Max.X)*0.5, r.Max.Y)
	case DirLeft:
		p.MoveTo(r.Max.X, r.Min.Y)
		p.LineTo(r.Max.X, r.Max.Y)
		p.LineTo(r.Min.X, (r.Min.Y+r.Max.Y)*0.5)
	default:
		p.MoveTo(r.Min.X, r.Min.Y)
		p.LineTo(r.Min.X, r.Max.Y)
		p.LineTo(r.Max.X, (r.Min.Y+r.Max.Y)*0.5)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// clTick strokes the bold Clearlooks tick into box (the 13px bitmap's
// shape: a short arm down to the vertex, a long arm up to the right).
func clTick(ctx *paintengine2d.Context, box paintengine2d.Rect, col paintengine2d.Color, w float32) {
	s := min(box.Dx(), box.Dy())
	p := paintengine2d.NewPath()
	p.MoveTo(box.Min.X+s*0.2, box.Min.Y+s*0.5)
	p.LineTo(box.Min.X+s*0.43, box.Min.Y+s*0.74)
	p.LineTo(box.Min.X+s*0.8, box.Min.Y+s*0.26)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 3}})
}

// clEmboss draws insensitive text GTK 2's way: a white copy one pixel down
// and right under the grey text.
func clEmboss(l *Classic, ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect, col paintengine2d.Color, align Align, pad float32, disabled bool) {
	if disabled {
		lw := clPx(l)
		l.drawFittedText(ctx, f, text, b.Translate(paintengine2d.Pt(lw, lw)), Hex("#ffffff"), align, pad)
	}
	l.drawFittedText(ctx, f, text, b, col, align, pad)
}

// clPulse is GTK 2's activity block: a fifth of the trough bouncing between
// its ends.
func clPulse(tr paintengine2d.Rect, phase float32) paintengine2d.Rect {
	if phase < 0 {
		phase = 0
	}
	phase -= float32(int(phase))
	w := tr.Dx() / 5
	u := phase * 2
	if u > 1 {
		u = 2 - u
	}
	return paintengine2d.XYWH(tr.Min.X+(tr.Dx()-w)*u, tr.Min.Y, w, tr.Dy())
}

// ---- parts: the button ---------------------------------------------------------------

// button paints a push-button face and returns the label colour. flat
// buttons (tool buttons, relief none) show nothing until hovered.
func (c *clSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, flat bool) paintengine2d.Color {
	b = clSnap(b)
	lw := clPx(l)
	dis := st.Disabled()
	down := !dis && (st.Pressed() || (st.Toggle() && st.Checked()))
	hot := !dis && st.Hovered()
	fg := c.fg
	if dis {
		fg = c.disFg
	}
	if flat && !hot && !down {
		return fg
	}
	if b.Dx() < 6*lw || b.Dy() < 6*lw {
		return fg
	}
	switch c.flavour {
	case clCleanlooks:
		c.qButton(l, ctx, b, st, down, hot, flat)
		return fg
	}
	r := clR(l, 3, b)
	frame := b
	if !flat {
		// The engraved ring the button sits in (solid mid for the
		// default button).
		ring := paintengine2d.Paint(VGradient(b, c.insetRing...))
		if st.Primary() && !dis {
			ring = paintengine2d.Fill(c.mid)
		}
		ctx.DrawRoundRect(b, r+lw, r+lw, ring)
		frame = b.Inset(lw)
	}
	r = clR(l, 3, frame)
	var edge paintengine2d.Paint
	switch {
	case dis:
		edge = paintengine2d.Fill(c.s[4])
	case st.Primary() && !flat:
		edge = paintengine2d.Fill(Hex("#000000"))
	case down:
		edge = VGradient(frame, c.aBorder...)
	case c.flavour == clHuman && hot:
		edge = paintengine2d.Fill(c.glowEdge)
	case c.flavour == clHuman:
		edge = paintengine2d.Fill(Hex("#736a60"))
	default:
		edge = VGradient(frame, c.border...)
	}
	face := frame.Inset(lw)
	rf := max(r-lw, 0)
	var fill paintengine2d.Paint
	switch {
	case down && st.Toggle() && st.Checked() && hot:
		fill = paintengine2d.Fill(c.s[1])
	case down:
		fill = VGradient(face, c.press...)
	case c.flavour == clHuman && hot:
		fill = VGradient(face, c.glassHot...)
	case c.flavour == clHuman:
		fill = VGradient(face, c.glass...)
	case hot:
		fill = VGradient(face, c.bandsHot...)
	case dis:
		fill = VGradient(face, c.bandsDis...)
	default:
		fill = VGradient(face, c.bands...)
	}
	clFrame(ctx, frame, r, lw, edge, fill)
	switch {
	case down && st.Toggle() && st.Checked():
		clLit(ctx, face, rf, lw, c.s[3], c.s[1])
	case down:
		clLit(ctx, face, rf, lw, c.s[4], paintengine2d.Color{})
	case c.flavour == clHuman && hot:
		// ubuntulooks' hover: a 1px orange glow inside the border.
		adwRing(ctx, face, rf, lw, c.glow)
	default:
		clLit(ctx, face, rf, lw, c.white, c.s[1])
	}
	return fg
}

// qButton is Cleanlooks' push button: a 2px-rounded #8f8882 outline with a
// darker top edge, the four-stop gradient, a white inner left edge, a
// faint shadow line above and a white line below (the frame is inset 1px
// top and bottom); the default button adds a black outline in a grey ring.
func (c *clSet) qButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, down, hot, flat bool) {
	lw := clPx(l)
	dis := st.Disabled()
	def := st.Primary() && !flat && !dis
	frame := paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), b.Dy()-2*lw)
	if !flat {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2*lw, b.Min.Y, b.Dx()-4*lw, lw), paintengine2d.Fill(c.qOutline.WithAlpha(0.12)))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2*lw, b.Max.Y-lw, b.Dx()-4*lw, lw), paintengine2d.Fill(c.white.WithAlpha(0.43)))
	}
	if def {
		ctx.DrawRoundRect(frame, clR(l, 3, frame), clR(l, 3, frame), paintengine2d.Fill(c.qDefRing))
		frame = frame.Inset(lw)
	}
	r := clR(l, 2, frame)
	edge := c.qOutline
	switch {
	case dis:
		edge = c.qDisOutline
	case def:
		edge = Hex("#000000")
	}
	face := frame.Inset(lw)
	rf := max(r-lw, 0)
	var fill paintengine2d.Paint
	switch {
	case down:
		fill = paintengine2d.Fill(Hex("#d0cac4"))
	case hot:
		fill = VGradient(face, c.qBtnHot...)
	case dis:
		fill = paintengine2d.Fill(c.bg)
	default:
		fill = VGradient(face, c.qBtn...)
	}
	clFrame(ctx, frame, r, lw, paintengine2d.Fill(edge), fill)
	if !dis && !def {
		// The top edge is a shade darker than the outline.
		ctx.Save()
		ctx.ClipRoundRect(frame, r, r)
		ctx.DrawRect(paintengine2d.XYWH(frame.Min.X, frame.Min.Y, frame.Dx(), lw), paintengine2d.Fill(c.qTopEdge))
		ctx.Restore()
	}
	if down {
		clLit(ctx, face, rf, lw, Hex("#b7b2ad"), paintengine2d.Color{})
	} else if !dis {
		ctx.Save()
		ctx.ClipRoundRect(face, rf, rf)
		ctx.DrawRect(paintengine2d.XYWH(face.Min.X, face.Min.Y, lw, face.Dy()), paintengine2d.Fill(c.white.WithAlpha(0.43)))
		ctx.Restore()
	}
}

// well is a sunken field: the engraved ring, the border (darker above) and
// the base colour, with the inner top/left shadow; focused, a spot-3
// border and a spot-1 inner ring.
func (c *clSet) well(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 5*lw || b.Dy() < 5*lw {
		return
	}
	dis := st.Disabled()
	focused := st.Focused() && !dis
	if c.flavour == clCleanlooks {
		// Qt's line edit: #a59d95 in 2px corners, an inner #d5d5d5 shadow
		// top and left, a white line under the box; focus: #415d77 with an
		// inner #b0c5d8 line.
		frame := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-lw)
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2*lw, b.Max.Y-lw, b.Dx()-4*lw, lw), paintengine2d.Fill(c.white.WithAlpha(0.5)))
		edge, fill := Hex("#a59d95"), c.base
		if focused {
			edge = c.qTabTop
		}
		if dis {
			fill = c.bg
		}
		r := clR(l, 2, frame)
		clFrame(ctx, frame, r, lw, paintengine2d.Fill(edge), paintengine2d.Fill(fill))
		in := frame.Inset(lw)
		if focused {
			adwRing(ctx, in, max(r-lw, 0), lw, Hex("#b0c5d8"))
		} else if !dis {
			clLit(ctx, in, max(r-lw, 0), lw, Hex("#d5d5d5"), paintengine2d.Color{})
		}
		return
	}
	r := clR(l, 3, b)
	ctx.DrawRoundRect(b, r+lw, r+lw, VGradient(b, c.insetRing...))
	frame := b.Inset(lw)
	r = clR(l, 3, frame)
	edge := paintengine2d.Paint(VGradient(frame, c.border...))
	fill := c.base
	switch {
	case dis:
		edge, fill = paintengine2d.Fill(c.s[3]), c.bg
	case focused && c.flavour == clHuman:
		edge = paintengine2d.Fill(c.glowEdge)
	case focused:
		edge = paintengine2d.Fill(c.spot[2])
	}
	clFrame(ctx, frame, r, lw, edge, paintengine2d.Fill(fill))
	in := frame.Inset(lw)
	ri := max(r-lw, 0)
	switch {
	case focused && c.flavour == clHuman:
		adwRing(ctx, in, ri, lw, c.glow)
	case focused:
		adwRing(ctx, in, ri, lw, c.spot[0])
	case !dis:
		clLit(ctx, in, ri, lw, c.bg, paintengine2d.Color{})
	}
}

// ---- parts ---------------------------------------------------------------------------

func (e clearlooksEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := clColors(l)
	switch role {
	case RoleButton, RoleCombo:
		return c.button(l, ctx, b, st, false)
	case RoleTool:
		return c.button(l, ctx, b, st, true)
	case RoleField:
		c.well(l, ctx, b, st)
		if st.Disabled() {
			return c.disFg
		}
		return c.text
	case RoleCheck:
		l.Engine().CheckIndicator(l, ctx, b, st, st.Checked())
		return c.fg
	case RoleRow:
		return c.row(ctx, b, st)
	case RoleMenu:
		if (st.Hovered() || st.Pressed() || st.Checked()) && !st.Disabled() {
			l.Engine().MenuHighlight(l, ctx, b, false)
			return l.Engine().MenuTextColor(l, true)
		}
		return c.fg
	case RoleTab:
		return c.fg
	case RoleThumb:
		c.slider(l, ctx, b, b.Dy() >= b.Dx(), st)
		return c.fg
	case RoleTrack:
		c.trough(l, ctx, b)
		return c.fg
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		return c.fg
	}
	return c.fg
}

// row paints a selected or hovered row and returns its label colour.
// Clearlooks runs the selection down to × 0.8 (listviewitemstyle 1); the
// selection stays when the view loses focus and turns to base[ACTIVE] only
// while the window is in the backdrop.
func (c *clSet) row(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	if !st.Checked() {
		return c.text
	}
	if st.Backdrop() {
		ctx.DrawRect(b, VGradient(b, c.rowOff...))
		return ReadableOn(c.rowOff[0].Color, 3, c.selFg, c.text)
	}
	ctx.DrawRect(b, VGradient(b, c.rowSel...))
	return c.rowSelFg
}

// CheckIndicator is the 13px Clearlooks check box: a square well framed in
// shade 5, its first row and column shaded for depth, and the bold dark
// tick. Human fills a checked box orange; Cleanlooks frames it lighter.
func (e clearlooksEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := clColors(l)
	lw := clPx(l)
	s := snap(min(box.Dx(), box.Dy()))
	if s < 5*lw {
		return
	}
	b := clSnap(adwCentered(box, s, s))
	dis := st.Disabled()
	edge := c.s[5]
	fill := c.base
	switch {
	case dis:
		fill = c.bg
	case st.Pressed():
		fill = c.active
	}
	if c.flavour == clCleanlooks {
		edge = c.qCheckEdge
		if st.Pressed() && !dis {
			fill = Hex("#d9d3cd")
		}
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	in := b.Inset(lw)
	on := checked && c.flavour == clHuman && !dis
	if on {
		ctx.DrawRect(b, paintengine2d.Fill(c.glowEdge))
		ctx.DrawRect(in, VGradient(in, c.chkOn...))
	} else {
		ctx.DrawRect(in, paintengine2d.Fill(fill))
		if !dis && c.flavour != clCleanlooks {
			// The first row and column let the frame show through.
			sh := Mix(fill, edge, 0.2)
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(sh))
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, lw, in.Dy()-lw), paintengine2d.Fill(sh))
		}
		if c.flavour == clHuman && st.Hovered() && !dis {
			adwRing(ctx, in, 0, lw, c.glow)
		}
	}
	if !checked {
		return
	}
	col := c.check
	switch {
	case dis:
		col = c.disFg
	case on:
		col = Hex("#000000")
	case st.Hovered():
		col = c.checkHot
	}
	clTick(ctx, b, col, max(s*0.17, 1.5*lw))
}

// RadioIndicator is the 13px Clearlooks radio: a shade-5 disc around an
// 11px white disc shaded at its upper left, and a 7px sphere-shaded dot.
func (e clearlooksEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := clColors(l)
	lw := clPx(l)
	s := min(box.Dx(), box.Dy())
	if s < 5*lw {
		return
	}
	ctr := box.Center()
	R := s * 0.5
	dis := st.Disabled()
	edge := c.s[5]
	fill, shade := c.base, c.radioShade[0]
	if c.flavour == clCleanlooks {
		edge = clRadioEdge
	}
	switch {
	case dis:
		fill, shade = c.bg, c.radioShade[1]
	case st.Pressed():
		fill, shade = c.active, c.radioShade[2]
	}
	on := selected && c.flavour == clHuman && !dis
	if on {
		edge = c.glowEdge
	}
	ctx.DrawCircle(ctr, R, paintengine2d.Fill(edge))
	inner := R - lw
	if on {
		ctx.DrawCircle(ctr, inner, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: paintengine2d.Pt(ctr.X-R*0.3, ctr.Y-R*0.3), Radius: R * 1.4, Stops: c.chkOn}))
	} else {
		ctx.DrawCircle(ctr, inner, paintengine2d.Fill(fill))
		if !dis {
			// A soft shadow inside the upper left of the white disc.
			ctx.DrawCircle(ctr, inner, paintengine2d.Radial(paintengine2d.RadialGradient{
				Center: paintengine2d.Pt(ctr.X+inner*0.35, ctr.Y+inner*0.35), Inner: inner * 0.9, Radius: inner * 1.4,
				Stops: shade}))
		}
		if c.flavour == clHuman && st.Hovered() && !dis {
			ctx.DrawCircle(ctr, inner-lw*0.5, paintengine2d.StrokePaint(c.glow, lw))
		}
	}
	if !selected {
		return
	}
	d := R * 7 / 13
	if dis {
		ctx.DrawCircle(ctr, d, paintengine2d.Fill(c.disFg))
		return
	}
	dot := c.dot
	if on {
		dot = c.dotOn
	}
	ctx.DrawCircle(ctr, d, paintengine2d.Radial(paintengine2d.RadialGradient{
		Center: paintengine2d.Pt(ctr.X-d*0.35, ctr.Y-d*0.35), Radius: d * 1.35, Stops: dot}))
}

// Arrow is GTK 2's solid triangle in the given colour (steppers, combo
// boxes, menus, headers).
func (clearlooksEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	w := min(b.Dx(), b.Dy())
	w = min(snap(w*0.55), snap(l.S(7)))
	if int(w)%2 == 0 {
		w--
	}
	clTriangle(ctx, b, dir, col, w)
}

// Expander is GTK 2's tree expander: a small triangle, outlined in the text
// colour and filled with the base.
func (clearlooksEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := clColors(l)
	if c.flavour == clCleanlooks {
		// Qt falls back to the Windows branch indicator: a 9px box in the
		// Dark colour with a plus or minus in the text colour.
		lw := clPx(l)
		s := snap(l.S(9))
		box := clSnap(adwCentered(b, s, s))
		ctx.DrawRect(box, paintengine2d.Fill(Hex("#9f9d9a")))
		ctx.DrawRect(box.Inset(lw), paintengine2d.Fill(c.base))
		mid := snap(box.Center().Y - lw*0.5)
		p := paintengine2d.NewPath()
		p.AddRect(paintengine2d.XYWH(box.Min.X+2*lw, mid, box.Dx()-4*lw, lw))
		if !expanded {
			p.AddRect(paintengine2d.XYWH(snap(box.Center().X-lw*0.5), box.Min.Y+2*lw, lw, box.Dy()-4*lw))
		}
		ctx.DrawPath(p, paintengine2d.Fill(c.text))
		return
	}
	s := min(b.Dx(), b.Dy(), l.S(10))
	g := adwCentered(b, s, s)
	cx, cy := (g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5
	p := paintengine2d.NewPath()
	if expanded {
		p.MoveTo(g.Min.X+s*0.1, cy-s*0.25)
		p.LineTo(g.Max.X-s*0.1, cy-s*0.25)
		p.LineTo(cx, cy+s*0.35)
	} else {
		p.MoveTo(cx-s*0.25, g.Min.Y+s*0.1)
		p.LineTo(cx+s*0.35, cy)
		p.LineTo(cx-s*0.25, g.Max.Y-s*0.1)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(c.base))
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: clPx(l), Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// MenuHighlight is Clearlooks' menu item (menuitemstyle 1): the selection
// easing to × 0.8 in a border of that darker colour, corners cut by a
// pixel. Human's is its light orange gradient; the menubar's open title
// joins the bar below.
func (clearlooksEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 3*lw || b.Dy() < 3*lw {
		return
	}
	r := clR(l, 1, b)
	edge := c.menuItem[1].Color
	if c.flavour == clHuman {
		edge = clShade(edge, 0.82)
	}
	if attachBottom {
		path := RoundRectPath(b, r, r, 0, 0)
		ctx.DrawPath(path, paintengine2d.Fill(edge))
		in := paintengine2d.XYWH(b.Min.X+lw, b.Min.Y+lw, b.Dx()-2*lw, b.Dy()-lw)
		ctx.DrawRoundRectCorners(in, max(r-lw, 0), max(r-lw, 0), 0, 0, VGradient(in, c.menuItem...))
		return
	}
	clFrame(ctx, b, r, lw, paintengine2d.Fill(edge), VGradient(b, c.menuItem...))
}

func (clearlooksEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := clColors(l)
	if hot {
		return ReadableOn(c.menuItem[0].Color, 4.5, c.selFg, c.fg)
	}
	return c.fg
}

// Fields show focus with their spot border; nothing rings them.
func (clearlooksEngine) FieldFocusRing(*Classic) bool { return false }

// DrawFocusRing is GTK 2's focus rectangle: 1px dots, one on one off, in
// shade 6 (Cleanlooks: Qt's 50% dotted #8b8783), square, just inside b.
func (clearlooksEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	clDotted(ctx, b, clPx(l), c.focus)
}

// ---- scrollbars ------------------------------------------------------------------------

// ScrollBarStyle: GTK 2's 15px bar with a stepper at each end.
func (clearlooksEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	ms := float32(30)
	switch clColors(l).flavour {
	case clHuman:
		ms = 35
	case clCleanlooks:
		ms = 26
	}
	return ScrollBarStyle{Thickness: 15, Arrows: ArrowsEnds, ArrowLen: 15, MinThumb: ms}
}

// trough is the scroll bar trough: shade 3 in a shade-5 border (Human's is
// a shade darker; Cleanlooks' groove is lined on its long edges).
func (c *clSet) trough(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = clSnap(b)
	lw := clPx(l)
	if b.Empty() {
		return
	}
	switch c.flavour {
	case clCleanlooks:
		ctx.DrawRect(b, paintengine2d.Fill(c.qGroove))
		return
	case clHuman:
		clFrame(ctx, b, 0, lw, paintengine2d.Fill(c.s[5]), paintengine2d.Fill(clShade(c.bg, 0.85)))
		return
	}
	clFrame(ctx, b, 0, lw, paintengine2d.Fill(c.s[5]), paintengine2d.Fill(c.s[3]))
}

// slider paints a scroll bar slider: the light-to-dark gradient across its
// thickness in the button border and inner ring, with three grip lines.
// Human's is rounded with a 6×3 grid of dots and orange ends when hovered;
// Cleanlooks' is Qt's.
func (c *clSet) slider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c.scrollFace(l, ctx, b, vertical, st, true, 0, 0)
}

// scrollFace paints a slider (grip true) or a stepper (grip false). A
// stepper rounds only its outer corners: rStart at the start of the bar,
// rEnd at its end.
func (c *clSet) scrollFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState, grip bool, rStart, rEnd float32) {
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	dis := st.Disabled()
	down := st.Pressed() && !dis
	hot := st.Hovered() && !dis
	grad := func(stops []paintengine2d.GradientStop) paintengine2d.Paint {
		if vertical {
			return HGradient(b, stops...)
		}
		return VGradient(b, stops...)
	}
	path := func(r paintengine2d.Rect, off float32) *paintengine2d.Path {
		rs, re := max(rStart-off, 0), max(rEnd-off, 0)
		if vertical {
			return RoundRectPath(r, rs, rs, re, re)
		}
		return RoundRectPath(r, rs, re, re, rs)
	}
	switch c.flavour {
	case clCleanlooks:
		edge := c.qOutline
		if dis {
			edge = c.qDisOutline
		}
		stops := c.qScroll
		if !grip {
			stops = c.qArrow
		}
		if hot || down {
			stops = c.qScrollHot
		}
		ctx.DrawPath(path(b, 0), paintengine2d.Fill(edge))
		in := b.Inset(lw)
		if down && !grip {
			ctx.DrawPath(path(in, lw), paintengine2d.Fill(Hex("#c0bcb7")))
		} else {
			ctx.DrawPath(path(in, lw), grad(stops))
		}
		if grip && !dis {
			c.qGripLines(l, ctx, in, vertical)
		}
		return
	case clHuman:
		if grip {
			r := clR(l, 3, b)
			edge := clShade(c.bg, 0.5)
			fill := clShade(c.bg, 0.896)
			if down {
				fill = clShade(c.bg, 0.82)
			}
			clFrame(ctx, b, r, lw, paintengine2d.Fill(edge), paintengine2d.Fill(fill))
			in := b.Inset(lw)
			ri := max(r-lw, 0)
			// The white gloss over the near half.
			var gl paintengine2d.Rect
			if vertical {
				gl = paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx()*0.5, in.Dy())
			} else {
				gl = paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), in.Dy()*0.5)
			}
			ctx.Save()
			ctx.ClipRoundRect(in, ri, ri)
			ctx.DrawRect(gl, paintengine2d.Fill(c.white.WithAlpha(0.35)))
			if hot || down {
				// The ends light up orange under the pointer.
				cap := snap(l.S(6))
				if vertical {
					ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), cap), VGradient(in, c.chkOn...))
					ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-cap, in.Dx(), cap), VGradient(in, c.chkOn...))
				} else {
					ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, cap, in.Dy()), VGradient(in, c.chkOn...))
					ctx.DrawRect(paintengine2d.XYWH(in.Max.X-cap, in.Min.Y, cap, in.Dy()), VGradient(in, c.chkOn...))
				}
			}
			ctx.Restore()
			c.dotGrip(l, ctx, in, vertical)
			return
		}
	}
	edge := c.upper
	stops := c.sliderFace
	switch {
	case dis:
		edge = c.s[4]
	case down:
		edge, stops = c.aUpper, c.sliderDn
	case hot:
		stops = c.sliderHot
	}
	ctx.DrawPath(path(b, 0), paintengine2d.Fill(edge))
	in := b.Inset(lw)
	ctx.DrawPath(path(in, lw), grad(stops))
	hi := c.white
	if down {
		hi = c.s[4]
	}
	ri := max(max(rStart, rEnd)-lw, 0)
	clLit(ctx, in, ri, lw, hi, c.s[1])
	if grip && !dis {
		c.gripLines(l, ctx, in, vertical)
	}
}

// gripLines are the three Clearlooks grip lines across a slider: 7px long
// pairs of shade 4 over shade 0, 3px apart, batched into two paths.
func (c *clSet) gripLines(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	lw := clPx(l)
	long := b.Dy()
	if !vertical {
		long = b.Dx()
	}
	if long < l.S(20) {
		return
	}
	n := snap(l.S(7))
	pitch := 3 * lw
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	ctr := b.Center()
	for i := -1; i <= 1; i++ {
		o := float32(i) * pitch
		if vertical {
			x0 := snap(ctr.X - n*0.5)
			y := snap(ctr.Y+o) - lw
			dk.AddRect(paintengine2d.XYWH(x0, y, n, lw))
			lt.AddRect(paintengine2d.XYWH(x0, y+lw, n, lw))
		} else {
			y0 := snap(ctr.Y - n*0.5)
			x := snap(ctr.X+o) - lw
			dk.AddRect(paintengine2d.XYWH(x, y0, lw, n))
			lt.AddRect(paintengine2d.XYWH(x+lw, y0, lw, n))
		}
	}
	ctx.DrawPath(dk, paintengine2d.Fill(c.s[4]))
	ctx.DrawPath(lt, paintengine2d.Fill(c.s[0]))
}

// qGripLines are Cleanlooks' three grip line pairs, 3px apart, 4px in from
// the slider's sides.
func (c *clSet) qGripLines(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	lw := clPx(l)
	in := snap(l.S(3))
	ctr := b.Center()
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for i := -1; i <= 1; i++ {
		o := float32(i) * 3 * lw
		if vertical {
			y := snap(ctr.Y + o)
			dk.AddRect(paintengine2d.XYWH(b.Min.X+in, y, b.Dx()-2*in, lw))
			lt.AddRect(paintengine2d.XYWH(b.Min.X+in, y+lw, b.Dx()-2*in, lw))
		} else {
			x := snap(ctr.X + o)
			dk.AddRect(paintengine2d.XYWH(x, b.Min.Y+in, lw, b.Dy()-2*in))
			lt.AddRect(paintengine2d.XYWH(x+lw, b.Min.Y+in, lw, b.Dy()-2*in))
		}
	}
	ctx.DrawPath(dk, paintengine2d.Fill(c.qGrip))
	ctx.DrawPath(lt, paintengine2d.Fill(c.white))
}

// dotGrip is ubuntulooks' 6×3 grid of dots in the middle of a slider.
func (c *clSet) dotGrip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	lw := clPx(l)
	cols, rows := 3, 6
	if !vertical {
		cols, rows = 6, 3
	}
	pitch := 3 * lw
	w, h := float32(cols-1)*pitch+lw, float32(rows-1)*pitch+lw
	if w > b.Dx()-2*lw || h > b.Dy()-2*lw {
		return
	}
	x0, y0 := snap(b.Center().X-w*0.5), snap(b.Center().Y-h*0.5)
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for r := 0; r < rows; r++ {
		for k := 0; k < cols; k++ {
			x, y := x0+float32(k)*pitch, y0+float32(r)*pitch
			dk.AddRect(paintengine2d.XYWH(x, y, lw, lw))
			lt.AddRect(paintengine2d.XYWH(x+lw, y+lw, lw, lw))
		}
	}
	ctx.DrawPath(lt, paintengine2d.Fill(c.white.WithAlpha(0.8)))
	ctx.DrawPath(dk, paintengine2d.Fill(c.s[5]))
}

func (e clearlooksEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := clColors(l)
	lw := clPx(l)
	if p.Bar.Empty() {
		return
	}
	// The trough runs between the steppers, a pixel short at each end.
	tr := p.Track
	if vertical {
		tr = paintengine2d.XYWH(p.Bar.Min.X, p.Track.Min.Y-lw, p.Bar.Dx(), p.Track.Dy()+2*lw)
	} else {
		tr = paintengine2d.XYWH(p.Track.Min.X-lw, p.Bar.Min.Y, p.Track.Dx()+2*lw, p.Bar.Dy())
	}
	ctx.DrawRect(p.Bar, paintengine2d.Fill(c.bg))
	c.trough(l, ctx, tr.Intersect(p.Bar))
	if c.flavour == clCleanlooks {
		// Qt lines the groove's long edges with the dark outline.
		g := tr.Intersect(p.Bar)
		if vertical {
			ctx.DrawRect(paintengine2d.XYWH(g.Min.X, g.Min.Y, lw, g.Dy()), paintengine2d.Fill(c.qOutline))
			ctx.DrawRect(paintengine2d.XYWH(g.Max.X-lw, g.Min.Y, lw, g.Dy()), paintengine2d.Fill(c.qOutline))
		} else {
			ctx.DrawRect(paintengine2d.XYWH(g.Min.X, g.Min.Y, g.Dx(), lw), paintengine2d.Fill(c.qOutline))
			ctx.DrawRect(paintengine2d.XYWH(g.Min.X, g.Max.Y-lw, g.Dx(), lw), paintengine2d.Fill(c.qOutline))
		}
	}
	r := clR(l, 3, p.Bar)
	if c.flavour == clCleanlooks {
		r = clR(l, 1, p.Bar)
	}
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart, first bool) {
		if b.Empty() {
			return
		}
		ps := st.Part(part)
		rs, re := r, float32(0)
		if !first {
			rs, re = 0, r
		}
		c.scrollFace(l, ctx, b, vertical, ps, false, rs, re)
		col := c.fg
		if st.Disabled {
			col = c.disFg
			l.Engine().Arrow(l, ctx, b.Translate(paintengine2d.Pt(lw, lw)), dir, c.white)
		}
		l.Engine().Arrow(l, ctx, b, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	arrow(p.Dec, dec, ScrollDec, true)
	arrow(p.Inc, inc, ScrollInc, false)
	if p.Thumb.Empty() {
		return
	}
	// At either end of its travel the slider takes a pixel of the stepper
	// so their borders merge.
	th := p.Thumb
	if vertical {
		if th.Min.Y <= p.Track.Min.Y+0.5 && !p.Dec.Empty() {
			th.Min.Y -= lw
		}
		if th.Max.Y >= p.Track.Max.Y-0.5 && !p.Inc.Empty() {
			th.Max.Y += lw
		}
	} else {
		if th.Min.X <= p.Track.Min.X+0.5 && !p.Dec.Empty() {
			th.Min.X -= lw
		}
		if th.Max.X >= p.Track.Max.X-0.5 && !p.Inc.Empty() {
			th.Max.X += lw
		}
	}
	ts := StateNone
	switch {
	case st.Disabled:
		ts = StateDisabled
	case st.Pressed == ScrollThumbPart:
		ts = StatePressed | StateHovered
	case st.Hot == ScrollThumbPart:
		ts = StateHovered
	}
	c.slider(l, ctx, th.Intersect(p.Bar), vertical, ts)
}

// DrawScrollBar is a bare slider over its trough.
func (e clearlooksEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := clColors(l)
	c.trough(l, ctx, track)
	if !thumb.Empty() {
		c.slider(l, ctx, thumb, thumb.Dy() >= thumb.Dx(), st)
	}
}

// ---- frames ------------------------------------------------------------------------------

// GroupBoxInsets: a GtkFrame's etched border with its label set into the
// top edge.
func (clearlooksEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.BoldFont().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a GtkFrame: an etched-in rectangle (shade 3 over a
// white copy one pixel down-right) with its bold label in the top edge.
func (e clearlooksEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	f := l.BoldFont()
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if frame.Dx() < 4*lw || frame.Dy() < 4*lw {
		return
	}
	dark := c.s[3]
	if c.flavour == clCleanlooks {
		dark = Hex("#cbc7c4")
	}
	r := clR(l, 3, frame)
	inner := paintengine2d.XYWH(frame.Min.X+lw, frame.Min.Y+lw, frame.Dx()-lw, frame.Dy()-lw)
	outer := paintengine2d.XYWH(frame.Min.X, frame.Min.Y, frame.Dx()-lw, frame.Dy()-lw)
	adwRing(ctx, inner, r, lw, c.white)
	adwRing(ctx, outer, r, lw, dark)
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(10)
	tw := f.Advance(title) + l.S(6)
	if tw > b.Dx()-l.S(20) {
		tw = b.Dx() - l.S(20)
	}
	ctx.DrawRect(paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), paintengine2d.Fill(c.bg))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(6), f.Height()), c.fg, AlignStart, 0)
}

// clCaptionH is the in-app title bar height.
func clCaptionH(l *Classic) float32 {
	return snap(max(l.metrics.TitleBar, l.BoldFont().Height()+l.S(8)))
}

func (clearlooksEngine) WindowFrameInsets(l *Classic) Insets {
	fr := snap(l.S(4))
	return Insets{Top: clCaptionH(l) + fr, Right: fr, Bottom: fr, Left: fr}
}

// DrawWindowFrame is Metacity's Clearlooks frame for an in-app window: a
// dark outline with a light inner edge, the title bar in three bands of the
// selection with rounded top corners, the title in white over a darker
// shadow, and the rounded close button at the right. Unfocused windows
// turn the bar to the background colour.
func (e clearlooksEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 12*lw || b.Dy() < 12*lw {
		return
	}
	r := clR(l, 5, b)
	base := c.caption
	if !st.Active {
		base = c.bg
	}
	ctx.DrawRoundRectCorners(b, r, r, 0, 0, paintengine2d.Fill(clShade(base, 0.45)))
	in := b.Inset(lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRectCorners(in, ri, ri, 0, 0, paintengine2d.Fill(c.bg))
	bar := paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), clCaptionH(l)+snap(l.S(4))-2*lw)
	stops := c.capGrad
	if !st.Active {
		stops = c.capOff
	}
	ctx.DrawRoundRectCorners(bar, ri, ri, 0, 0, VGradient(bar, stops...))
	ctx.Save()
	ctx.ClipPath(RoundRectPath(bar, ri, ri, 0, 0))
	ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Min.Y, bar.Dx(), lw), paintengine2d.Fill(clShade(base, 1.2)))
	ctx.Restore()
	ctx.DrawRect(paintengine2d.XYWH(bar.Min.X, bar.Max.Y, bar.Dx(), lw), paintengine2d.Fill(clShade(base, 0.5)))
	right := bar.Max.X
	if st.CanClose {
		if cb := e.WindowCloseRect(l, b); !cb.Empty() {
			c.closeButton(l, ctx, cb, st)
			right = cb.Min.X - l.S(4)
		}
	}
	if title == "" {
		return
	}
	f := l.BoldFont()
	tb := paintengine2d.XYWH(bar.Min.X+l.S(6), bar.Min.Y, right-bar.Min.X-l.S(6), bar.Dy())
	if st.Active {
		l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(lw, lw)), clShade(base, 0.5), AlignCenter, 8)
		l.drawFittedText(ctx, f, title, tb, c.captionText, AlignCenter, 8)
		return
	}
	l.drawFittedText(ctx, f, title, tb, c.captionOff, AlignCenter, 8)
}

// closeButton is Metacity Clearlooks' close button: a 3px-rounded button of
// the title colour (lighter on hover, darker pressed) with a white ×.
func (c *clSet) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	lw := clPx(l)
	base := c.caption
	glyph := c.captionText
	if !st.Active {
		base, glyph = c.bg, Mix(c.bg, c.fg, 0.55)
	}
	r := clR(l, 3, cb)
	fill := paintengine2d.Paint(VGradient(cb, c.capBtn...))
	switch {
	case !st.Active:
		fill = VGradient(cb, c.capOffBtn...)
	case st.ClosePress:
		fill = VGradient(cb, c.capDown...)
	case st.CloseHot:
		fill = VGradient(cb, c.capHot...)
	}
	clFrame(ctx, cb, r, lw, paintengine2d.Fill(clShade(base, 0.6)), fill)
	if st.Active {
		clLit(ctx, cb.Inset(lw), max(r-lw, 0), lw, clShade(base, 1.18), paintengine2d.Color{})
	}
	g := cb.Inset(snap(cb.Dx() * 0.3))
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X, g.Min.Y)
	p.LineTo(g.Max.X, g.Max.Y)
	p.MoveTo(g.Max.X, g.Min.Y)
	p.LineTo(g.Min.X, g.Max.Y)
	ctx.DrawPath(p, paintengine2d.Paint{Color: glyph, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: 2 * lw, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// WindowCloseRect is the close button at the title bar's right end.
func (clearlooksEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = clSnap(b)
	lw := clPx(l)
	h := clCaptionH(l) + snap(l.S(4)) - 2*lw
	side := snap(h - l.S(8))
	if side < 8 || b.Dx() < side+l.S(24) || b.Dy() < h+2*lw {
		return paintengine2d.Rect{}
	}
	return paintengine2d.XYWH(b.Max.X-lw-snap(l.S(4))-side, b.Min.Y+lw+snap((h-side)*0.5), side, side)
}

// No drop shadows: GNOME 2's menus and tooltips floated flat.
func (clearlooksEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (clearlooksEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {
}

// GNOME (and Qt under GNOME) put the affirmative button last: "Cancel OK".
func (clearlooksEngine) StyleHint(l *Classic, h StyleHint) int { return 0 }

// TabOutset: Qt's tabs overlap; GTK 2's sit side by side.
func (clearlooksEngine) TabOutset(l *Classic) Insets {
	if clColors(l).flavour == clCleanlooks {
		return Insets{Left: snap(l.S(2)), Right: snap(l.S(2))}
	}
	return Insets{}
}

// DrawTabPane is a GtkNotebook's frame: the notebook colour in a shade-5
// border lit white inside; its top edge is the tab bar's.
func (clearlooksEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	edge, fill := c.s[5], c.notebook
	if c.flavour == clCleanlooks {
		edge, fill = Hex("#9d968f"), Hex("#e9e4df")
	}
	r := clR(l, 3, b)
	ctx.DrawRoundRectCorners(b, 0, 0, r, r, paintengine2d.Fill(edge))
	in := b.Inset(lw)
	ctx.DrawRoundRectCorners(in, 0, 0, max(r-lw, 0), max(r-lw, 0), paintengine2d.Fill(fill))
	clLit(ctx, in, max(r-lw, 0), lw, c.white, clShade(fill, 0.93))
}

// ItemFocus: GTK 2 marks the cursor row with the dotted focus rectangle,
// drawn in the selected text colour on a selection.
func (clearlooksEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := clColors(l)
	col := c.focus
	if st.Checked() {
		col = c.rowSelFg.WithAlpha(0.75)
	}
	clDotted(ctx, b, clPx(l), col)
}

// ViewFrameInsets: a scrolled window's 1px shadow-in frame.
func (clearlooksEngine) ViewFrameInsets(l *Classic) Insets {
	v := float32(math.Ceil(float64(l.metrics.ViewFrame)))
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is that frame (shade 4) around the base colour.
func (e clearlooksEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := clColors(l)
	in := e.ViewFrameInsets(l)
	if in.Zero() || b.Empty() {
		return
	}
	edge := c.s[4]
	if c.flavour == clCleanlooks {
		edge = c.qOutline
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	ctx.DrawRect(in.Apply(b), paintengine2d.Fill(c.base))
}

// ---- controls ------------------------------------------------------------------------------

func (e clearlooksEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := clColors(l)
	fg := l.Engine().Face(l, ctx, b, RoleButton, st)
	lb := b
	if st.Pressed() && !st.Disabled() {
		lw := clPx(l)
		lb = lb.Translate(paintengine2d.Pt(lw, lw)) // child-displacement 1, 1
	}
	clEmboss(l, ctx, l.body, label, lb, fg, AlignCenter, l.S(12), st.Disabled())
	if st.Focused() && !st.Disabled() {
		in := snap(l.S(4))
		if c.flavour == clCleanlooks {
			in = snap(l.S(3))
		}
		l.Engine().DrawFocusRing(l, ctx, b.Inset(in))
	}
}

func (e clearlooksEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	fg := l.Engine().Face(l, ctx, b, RoleTool, st)
	lw := clPx(l)
	if (st.Pressed() || (st.Toggle() && st.Checked())) && !st.Disabled() {
		b = b.Translate(paintengine2d.Pt(lw, lw))
	}
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
		right := max(b.Max.X-pad, x)
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		l.body.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-l.body.Height())*0.5), fg)
		ctx.Restore()
	}
	if st.Focused() && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b.Inset(snap(l.S(3))))
	}
}

// toggle lays out a check / radio: the indicator at the left, the label
// 6px after it, the focus rectangle around the label (GTK 2 rings the
// check button's child).
func (e clearlooksEngine) toggle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, side float32, radio, on bool) {
	c := clColors(l)
	side = min(side, b.Dy(), b.Dx())
	box := clSnap(paintengine2d.XYWH(b.Min.X+l.S(1), b.Min.Y+(b.Dy()-side)*0.5, side, side))
	if radio {
		l.Engine().RadioIndicator(l, ctx, box, st, on)
	} else {
		l.Engine().CheckIndicator(l, ctx, box, st, on)
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	if label != "" {
		fg := c.fg
		if st.Disabled() {
			fg = c.disFg
		}
		clEmboss(l, ctx, l.body, label, lb, fg, AlignStart, 0, st.Disabled())
	}
	if !st.Focused() || st.Disabled() {
		return
	}
	if label == "" {
		l.Engine().DrawFocusRing(l, ctx, box.Inset(-l.S(2)).Intersect(b))
		return
	}
	l.Engine().DrawFocusRing(l, ctx, labelFocusRect(l.body, label, lb, b))
}

func (e clearlooksEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	e.toggle(l, ctx, b, st, label, l.metrics.Checkbox, false, checked)
}

func (e clearlooksEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	e.toggle(l, ctx, b, st, label, side, true, selected)
}

// DrawSwitch: GTK 2 had no switch. It is drawn as the toolkit's two-state
// slider in Clearlooks materials: a sunken well (filled like a scale's
// lower trough when on) with a button knob.
func (e clearlooksEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := clColors(l)
	lw := clPx(l)
	m := l.metrics
	tw, th := min(m.SwitchW, b.Dx()), min(m.SwitchH, b.Dy())
	track := clSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 10*lw || track.Dy() < 8*lw {
		return
	}
	dis := st.Disabled()
	c.well(l, ctx, track, st&^StateFocused)
	in := track.Inset(2 * lw)
	if on && !dis {
		r := clR(l, 2, in)
		ctx.DrawRoundRect(in, r, r, VGradient(in, c.scaleFill...))
	}
	kw := snap(in.Dx() * 0.5)
	kx := in.Min.X
	if on {
		kx = in.Max.X - kw
	}
	knob := paintengine2d.XYWH(kx, in.Min.Y, kw, in.Dy())
	ks := StateNone
	switch {
	case dis:
		ks = StateDisabled
	case st.Pressed():
		ks = StateHovered
	case st.Hovered():
		ks = StateHovered
	}
	if dis {
		l.Engine().Face(l, ctx, knob, RoleButton, StateDisabled)
	} else {
		// A flat button forced to show its face: the knob.
		if ks == StateNone {
			ks = StateHovered
		}
		l.Engine().Face(l, ctx, knob, RoleTool, ks)
	}
	if label != "" {
		fg := c.fg
		if dis {
			fg = c.disFg
		}
		gap := l.S(8)
		lb := paintengine2d.XYWH(track.Max.X+gap, b.Min.Y, b.Max.X-track.Max.X-gap, b.Dy())
		clEmboss(l, ctx, l.body, label, lb, fg, AlignStart, 0, dis)
		if st.Focused() && !dis {
			l.Engine().DrawFocusRing(l, ctx, labelFocusRect(l.body, label, lb, b))
		}
	} else if st.Focused() && !dis {
		l.Engine().DrawFocusRing(l, ctx, track)
	}
}

// DrawSlider is a GtkHScale: a 5px trough (shade 3 in shade 5, lit shade 4
// inside) filled with the selection up to the knob, and the rounded knob
// with its grip lines.
func (e clearlooksEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := clColors(l)
	lw := clPx(l)
	t = clamp1(t)
	kl := snap(l.S(27)) // slider-length along the trough
	kw := snap(min(l.S(15), b.Dy()-2*lw))
	if c.flavour == clHuman {
		kl = snap(l.S(31))
	}
	kl = min(kl, snap(b.Dx()*0.4))
	if kw < 6*lw || kl < 6*lw {
		return
	}
	dis := st.Disabled()
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	k0 := snap(b.Min.X + (b.Dx()-kl)*t) // the knob's left edge: kept inside b
	kx := k0 + kl*0.5
	th := snap(l.S(5))
	if c.flavour == clCleanlooks {
		th = snap(l.S(7))
	}
	trough := clSnap(paintengine2d.XYWH(b.Min.X+lw, cy-th*0.5, b.Dx()-2*lw, th))
	edge, fill, hi := c.s[5], c.s[3], c.s[4]
	if dis {
		edge, fill, hi = c.s[3], c.s[2], c.s[4]
	}
	if c.flavour == clCleanlooks {
		edge = c.qTopEdge
	}
	r := clR(l, 2, trough)
	if c.flavour == clCleanlooks {
		clFrame(ctx, trough, r, lw, paintengine2d.Fill(edge), VGradient(trough, c.qTrough...))
	} else {
		clFrame(ctx, trough, r, lw, paintengine2d.Fill(edge), paintengine2d.Fill(fill))
		clLit(ctx, trough.Inset(lw), max(r-lw, 0), lw, hi, paintengine2d.Color{})
	}
	if lo := paintengine2d.XYWH(trough.Min.X, trough.Min.Y, kx-trough.Min.X, trough.Dy()); lo.Dx() > 2*lw && !dis {
		fe := c.spot[2]
		if c.flavour == clCleanlooks {
			fe = c.qTabTop
		}
		clFrame(ctx, lo, r, lw, paintengine2d.Fill(fe), VGradient(lo, c.scaleFill...))
	}
	knob := paintengine2d.XYWH(k0, snap(cy-kw*0.5), kl, kw)
	ks := StateNone
	switch {
	case dis:
		ks = StateDisabled
	case st.Pressed():
		ks = StatePressed
	case st.Hovered():
		ks = StateHovered
	}
	c.knob(l, ctx, knob, ks)
	if st.Focused() && !dis {
		l.Engine().DrawFocusRing(l, ctx, b)
	}
}

// knob is a scale's slider: rounded, the background easing to shade 2 in
// the button border with a white / shade-2 inner ring and three grip lines
// (Cleanlooks: chamfered with end dividers, blue caps hovered).
func (c *clSet) knob(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	lw := clPx(l)
	dis := st.Disabled()
	r := clR(l, 3, b)
	edge := paintengine2d.Paint(VGradient(b, c.border...))
	stops := c.scaleKnob
	switch {
	case dis:
		edge = paintengine2d.Fill(c.s[4])
	case st.Pressed():
		stops = c.press
	case st.Hovered() && c.flavour == clCleanlooks:
		edge = paintengine2d.Fill(Hex("#4b6c89"))
	case st.Hovered():
		stops = c.bandsHot
	}
	if c.flavour == clCleanlooks && !dis && !st.Hovered() {
		edge = paintengine2d.Fill(Hex("#807d79"))
	}
	in := b.Inset(lw)
	clFrame(ctx, b, r, lw, edge, VGradient(in, stops...))
	clLit(ctx, in, max(r-lw, 0), lw, c.white, c.s[2])
	if dis {
		return
	}
	if c.flavour == clCleanlooks {
		if st.Hovered() {
			cap := snap(l.S(6))
			ctx.Save()
			ctx.ClipRoundRect(in, max(r-lw, 0), max(r-lw, 0))
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, cap, in.Dy()), VGradient(in, c.qCap...))
			ctx.DrawRect(paintengine2d.XYWH(in.Max.X-cap, in.Min.Y, cap, in.Dy()), VGradient(in, c.qCap...))
			ctx.Restore()
		}
		c.qGripLines(l, ctx, in, false)
		return
	}
	c.gripLines(l, ctx, in, false)
}

// DrawProgressBar is the Clearlooks candy bar: a shade-2 trough in shade 5
// with 1px cut corners; the fill is the selection easing a tenth darker
// down, striped at 45° in the selection every bar-height, in a spot-3
// border lit along its top. Human fills orange square cells; Cleanlooks
// draws Qt's blue bar with its 7px stripes.
func (e clearlooksEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 6*lw || b.Dy() < 5*lw {
		return
	}
	dis := st.Disabled()
	rt := clR(l, 1, b)
	switch c.flavour {
	case clCleanlooks:
		clFrame(ctx, b, clR(l, 2, b), lw, paintengine2d.Fill(c.qCheckEdge), paintengine2d.Fill(c.base))
		clLit(ctx, b.Inset(lw), 0, lw, Hex("#dbd8d5"), paintengine2d.Color{})
	default:
		clFrame(ctx, b, rt, lw, paintengine2d.Fill(c.s[5]), paintengine2d.Fill(c.s[2]))
	}
	tr := b.Inset(lw)
	var fill paintengine2d.Rect
	if indeterminate {
		fill = clPulse(tr, phase)
	} else {
		fill = paintengine2d.XYWH(tr.Min.X, tr.Min.Y, snap(tr.Dx()*clamp1(t)), tr.Dy())
	}
	if fill.Dx() < 2*lw {
		return
	}
	fill = clSnap(fill)
	if dis {
		ctx.DrawRect(fill, paintengine2d.Fill(Mix(c.s[2], c.spot[1], 0.35)))
		return
	}
	T := fill.Dy()
	switch c.flavour {
	case clHuman:
		// A row of square cells, each a glass gradient of the orange with
		// a light left edge and a dark right edge.
		ctx.Save()
		ctx.ClipRect(fill)
		cell := max(snap(T), 4*lw)
		lit, shade := paintengine2d.NewPath(), paintengine2d.NewPath()
		for x := fill.Min.X; x < fill.Max.X; x += cell {
			lit.AddRect(paintengine2d.XYWH(x, fill.Min.Y, lw, T))
			shade.AddRect(paintengine2d.XYWH(x+cell-lw, fill.Min.Y, lw, T))
		}
		ctx.DrawRect(fill, VGradient(fill, c.cellGrad...))
		ctx.DrawPath(lit, paintengine2d.Fill(c.white.WithAlpha(0.4)))
		ctx.DrawPath(shade, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.1)))
		ctx.Restore()
		ctx.DrawRect(paintengine2d.XYWH(fill.Min.X, fill.Min.Y, fill.Dx(), lw), paintengine2d.Fill(clShade(c.spotBase, 0.8)))
		return
	case clCleanlooks:
		r := clR(l, 1, fill)
		clFrame(ctx, fill, r, lw, paintengine2d.Fill(c.qProgEdge), VGradient(fill, c.qProg...))
		in := fill.Inset(lw)
		// 7px stripes every 18px, leaning over 23px across the bar.
		c.stripes(ctx, in, snap(l.S(7)), snap(l.S(18)), in.Dy()*23/max(l.S(14), 1), c.sel.WithAlpha(0.6))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.qProgHi))
		return
	}
	ctx.DrawRect(fill, VGradient(fill, c.candy...))
	in := fill.Inset(lw)
	c.stripes(ctx, in, snap(T*0.5), snap(T*0.5)*2, in.Dy(), c.spot[1])
	clBorder(ctx, fill, lw, c.spot[2])
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.candyHi))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+lw, lw, in.Dy()-lw), paintengine2d.Fill(c.candyHi))
}

// stripes fills "/" parallelograms of width w every period px across b,
// each leaning lean px over b's height, batched into one path.
func (c *clSet) stripes(ctx *paintengine2d.Context, b paintengine2d.Rect, w, period, lean float32, col paintengine2d.Color) {
	if b.Empty() || w < 1 || period < w+1 {
		return
	}
	p := paintengine2d.NewPath()
	for x := b.Min.X - lean; x < b.Max.X; x += period {
		p.MoveTo(x+lean, b.Min.Y)
		p.LineTo(x+lean+w, b.Min.Y)
		p.LineTo(x+w, b.Max.Y)
		p.LineTo(x, b.Max.Y)
		p.Close()
	}
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawPath(p, paintengine2d.Fill(col))
	ctx.Restore()
}

// clBorder strokes a square lw-wide border inside b (four rects).
func clBorder(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, col paintengine2d.Color) {
	if b.Dx() < 2*lw || b.Dy() < 2*lw {
		return
	}
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, lw, b.Dy()-2*lw))
	p.AddRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw))
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// DrawTextField is a GtkEntry: the sunken well, the text inside.
func (e clearlooksEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

// DrawComboBox is a GtkComboBox: a push button with the label at the left,
// a separator (shade 3 over white) and the stacked up / down arrows of an
// option menu at the right.
func (e clearlooksEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := clColors(l)
	lw := clPx(l)
	bs := st &^ (StatePrimary | StateFocused)
	if open {
		bs |= StatePressed
	}
	fg := l.Engine().Face(l, ctx, b, RoleButton, bs)
	aw := snap(l.S(22))
	ax := snap(b.Max.X - aw)
	if b.Dy() > 10*lw {
		sep := paintengine2d.XYWH(ax, b.Min.Y+snap(l.S(6)), lw, b.Dy()-2*snap(l.S(6)))
		ctx.DrawRect(sep, paintengine2d.Fill(c.s[3]))
		ctx.DrawRect(sep.Translate(paintengine2d.Pt(lw, 0)), paintengine2d.Fill(c.white))
	}
	ab := paintengine2d.XYWH(ax+lw, b.Min.Y, aw-lw, b.Dy())
	w := snap(l.S(7))
	gap := snap(l.S(1.5))
	h := snap(w*0.5 + 0.5)
	up := paintengine2d.XYWH(ab.Min.X, ab.Center().Y-h-gap, ab.Dx(), h)
	dn := paintengine2d.XYWH(ab.Min.X, ab.Center().Y+gap, ab.Dx(), h)
	col := fg
	clTriangle(ctx, up, DirUp, col, w)
	clTriangle(ctx, dn, DirDown, col, w)
	lb := paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, ax-b.Min.X-l.S(10), b.Dy())
	if open {
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	}
	clEmboss(l, ctx, l.body, text, lb, fg, AlignStart, 0, st.Disabled())
	if st.Focused() && !open && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, ax-b.Min.X, b.Dy()).Inset(snap(l.S(4))))
	}
}

// DrawSpinner is a GtkSpinButton's two buttons: rounded on their outer
// corner only, the background easing to × 0.93, split at mid-height.
func (e clearlooksEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 5*lw || b.Dy() < 8*lw {
		return
	}
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	r := clR(l, 3, b)
	half := func(h paintengine2d.Rect, dir Direction, hover, press, top bool) {
		dis := st.Disabled()
		edge := paintengine2d.Paint(VGradient(h, c.border...))
		face := c.spin
		switch {
		case dis:
			edge = paintengine2d.Fill(c.s[4])
		case press:
			edge = VGradient(h, c.aBorder...)
			face = c.press
		case hover:
			face = c.bandsHot
		}
		if c.flavour == clCleanlooks {
			edge = paintengine2d.Fill(c.qOutline)
			if !press {
				face = c.qSpin
			}
		}
		tr, br := float32(0), float32(0)
		if top {
			tr = r
		} else {
			br = r
		}
		ctx.DrawRoundRectCorners(h, 0, tr, br, 0, edge)
		in := h.Inset(lw)
		ctx.DrawRoundRectCorners(in, 0, max(tr-lw, 0), max(br-lw, 0), 0, VGradient(in, face...))
		hi := c.white
		if press {
			hi = c.s[4]
		}
		clLit(ctx, in, 0, lw, hi, paintengine2d.Color{})
		col := c.fg
		g := in
		if press {
			g = g.Translate(paintengine2d.Pt(lw, lw))
		}
		if dis {
			col = c.disFg
			l.Engine().Arrow(l, ctx, g.Translate(paintengine2d.Pt(lw, lw)), dir, c.white)
		}
		l.Engine().Arrow(l, ctx, g, dir, col)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y+lw), DirUp, upHover, upPress, true)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress, false)
}

// DrawTabBar is the strip above a notebook page: the window colour with
// the page frame's top edge (shade 5 over its white inner line) along the
// bottom; the selected tab opens it.
func (e clearlooksEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	edge := c.s[5]
	if c.flavour == clCleanlooks {
		edge = Hex("#9d968f")
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*lw, b.Dx(), lw), paintengine2d.Fill(edge))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.white))
}

// DrawTab is a Clearlooks notebook tab: rounded top corners, open into the
// page. The selected tab is taller, fills with the page colour lightening
// towards its top, wears a 3px band along its top edge — spot 3 over two
// rows of spot 2 (Human: orange; Cleanlooks: Qt's dark line over two blue
// rows) — and is lit white inside. The others sit 2px lower, darker, in
// the pressed border.
func (e clearlooksEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	t := b
	if !selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+snap(l.S(2)), b.Dx(), b.Dy()-snap(l.S(2))-2*lw)
	}
	if t.Dx() < 8*lw || t.Dy() < 8*lw {
		return
	}
	r := clR(l, 3, t)
	edge, stops := c.nbBorder, c.tabSel
	if !selected {
		edge, stops = c.aBorder, c.tabOff
		if st.Hovered() && !st.Disabled() {
			stops = c.tabHot
		}
	}
	if c.flavour == clCleanlooks {
		edge = c.qEdge
	}
	ctx.DrawRoundRectCorners(t, r, r, 0, 0, VGradient(t, edge...))
	in := paintengine2d.XYWH(t.Min.X+lw, t.Min.Y+lw, t.Dx()-2*lw, t.Dy()-lw)
	ri := max(r-lw, 0)
	ctx.DrawRoundRectCorners(in, ri, ri, 0, 0, VGradient(in, stops...))
	ctx.Save()
	ctx.ClipPath(RoundRectPath(in, ri, ri, 0, 0))
	if selected {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.white))
		ctx.DrawRect(paintengine2d.XYWH(in.Max.X-lw, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(clShade(c.notebook, 0.93)))
	} else {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), lw), paintengine2d.Fill(c.white.WithAlpha(0.7)))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, lw, in.Dy()), paintengine2d.Fill(c.white.WithAlpha(0.7)))
	}
	ctx.Restore()
	if selected {
		// The stripe follows the rounded top: clip to the tab's outline.
		ctx.Save()
		ctx.ClipPath(RoundRectPath(t, r, r, 0, 0))
		outer, inner := c.spot[2], c.spot[1]
		switch c.flavour {
		case clCleanlooks:
			outer, inner = c.qTabTop, c.qTabMid
		case clHuman:
			outer, inner = clShade(c.spotBase, 0.65), c.spotBase
		}
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx(), lw), paintengine2d.Fill(outer))
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X, t.Min.Y+lw, t.Dx(), 2*lw), paintengine2d.Fill(inner))
		ctx.Restore()
		// Open the page edge under the selected tab.
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, b.Max.Y-2*lw, in.Dx(), 2*lw), paintengine2d.Fill(stops[len(stops)-1].Color))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, b.Max.Y-2*lw, lw, 2*lw), paintengine2d.Fill(c.white))
	}
	fg := c.fg
	if st.Disabled() {
		fg = c.disFg
	}
	lb := paintengine2d.XYWH(t.Min.X, t.Min.Y+2*lw, t.Dx(), t.Dy()-2*lw)
	clEmboss(l, ctx, l.body, label, lb, fg, AlignCenter, l.S(10), st.Disabled())
	if st.Focused() && selected {
		f := l.body
		w := min(f.Advance(label)+l.S(6), lb.Dx()-l.S(6))
		h := f.Height() + l.S(2)
		l.Engine().DrawFocusRing(l, ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h))
	}
}

// DrawPanel: a raised panel is an OUT frame (white inside a shade-4
// border); a flat one the plain background.
func (e clearlooksEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if b.Dx() < 4*lw || b.Dy() < 4*lw {
		return
	}
	if raised {
		clBorder(ctx, b, lw, c.s[4])
		clLit(ctx, b.Inset(lw), 0, lw, c.white, c.s[1])
		return
	}
	clBorder(ctx, b, lw, c.s[3])
}

// DrawMenuBar is menubarstyle 2: the background easing to × 0.95 over a
// shade-3 bottom line.
func (e clearlooksEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, VGradient(b, c.menubar...))
	line := c.s[3]
	if c.flavour == clCleanlooks {
		line = Hex("#d4cfcb")
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(line))
}

// DrawMenuTitle is a menubar item: the menu-item highlight, taller by a
// pixel so its bottom merges into the bar, while hovered or open.
func (e clearlooksEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := clColors(l)
	hot := !st.Disabled() && (open || st.Pressed() || st.Hovered())
	fg := c.fg
	if hot {
		hb := paintengine2d.XYWH(b.Min.X, b.Min.Y+l.S(2), b.Dx(), b.Dy()-l.S(2))
		l.Engine().MenuHighlight(l, ctx, hb, true)
		fg = l.Engine().MenuTextColor(l, true)
	}
	if st.Disabled() {
		fg = c.disFg
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b.Inset(snap(l.S(2))))
	}
}

// DrawMenuFrame is a GtkMenu: the menu colour in a square border of its
// × 0.5 shade, lit white top and left, shade 1 bottom and right.
func (e clearlooksEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Empty() {
		return
	}
	edge, lo := c.menuBorder, c.menuS1
	if c.flavour == clCleanlooks {
		edge, lo = c.qOutline, Hex("#dad6d2")
	}
	ctx.DrawRect(b, paintengine2d.Fill(edge))
	in := b.Inset(lw)
	ctx.DrawRect(in, paintengine2d.Fill(c.menu))
	clLit(ctx, in, 0, lw, c.white, lo)
}

// DrawMenuItem is a GtkMenuItem: the highlight across the row, the check or
// radio indicator in the gutter, the accelerator and the submenu arrow in
// the label colour; separators are shade 2 over shade 0.
func (e clearlooksEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := clColors(l)
	ch := MenuChromeFor(l)
	lw := clPx(l)
	if row.Separator {
		y := snap((b.Min.Y+b.Max.Y)*0.5) - lw
		x0, x1 := snap(b.Min.X-ch.PadL+2*lw), snap(b.Max.X+ch.PadR-2*lw)
		if c.flavour == clCleanlooks {
			x0, x1 = snap(b.Min.X-ch.PadL+l.S(5)), snap(b.Max.X+ch.PadR-l.S(5))
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(Hex("#dad5d1")))
			return
		}
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, lw), paintengine2d.Fill(clShade(c.menu, 0.896)))
			ctx.DrawRect(paintengine2d.XYWH(x0, y+lw, x1-x0, lw), paintengine2d.Fill(clShade(c.menu, 1.065)))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL+lw, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*lw, b.Dy())
		l.Engine().MenuHighlight(l, ctx, hb, false)
	}
	fg := l.Engine().MenuTextColor(l, hot)
	if st.Disabled() {
		fg = c.disFg
	}
	gw := ch.CheckCol()
	s := snap(min(l.S(13), gw-2*lw, b.Dy()-2*lw))
	ib := clSnap(paintengine2d.XYWH(b.Min.X+(gw-s)*0.5, b.Min.Y+(b.Dy()-s)*0.5, s, s))
	switch {
	case row.Radio:
		// GTK 2 menus draw their own indicators: a disc or a tick in the
		// label colour, no well.
		ctx.DrawCircle(ib.Center(), s*0.5-lw*0.5, paintengine2d.StrokePaint(fg, lw))
		if row.Checked {
			ctx.DrawCircle(ib.Center(), s*0.22, paintengine2d.Fill(fg))
		}
	case row.Checked && row.Icon == IconNone:
		clTick(ctx, ib, fg, max(s*0.17, 1.5*lw))
	default:
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := max(ch.SubmenuArrow, l.S(10))
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		l.Engine().Arrow(l, ctx, ab, DirRight, fg)
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
		if st.Disabled() {
			f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx+lw, ty+lw), c.white)
		}
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), fg)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()))
	if st.Disabled() {
		l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx+lw, ty+lw), c.white)
	}
	l.drawTextUnderline(ctx, f, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

func (e clearlooksEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := clColors(l)
	fg := l.Engine().Face(l, ctx, b, RoleRow, st)
	if !st.Checked() && st.Hovered() {
		fg = c.text
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		l.Engine().ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a GtkTreeView row: the expander triangle, the label, the
// selection across the whole row.
func (e clearlooksEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := clColors(l)
	fg := l.Engine().Face(l, ctx, b, RoleRow, st)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	es := l.S(14) // GtkTreeView expander-size 14
	if c.flavour == clCleanlooks {
		// Qt's dotted branch lines, one per ancestor level plus the elbow.
		lw := clPx(l)
		dots := paintengine2d.NewPath()
		cy := snap((b.Min.Y + b.Max.Y) * 0.5)
		for d := 0; d <= depth; d++ {
			if d < depth && !st.HasNextSibling(d) {
				continue // that ancestor's branch has ended
			}
			gx := snap(b.Min.X + l.S(4) + float32(d)*indent + es*0.5)
			y1 := b.Max.Y
			if d == depth && !st.HasNextSibling(depth) {
				y1 = cy // the last child's elbow
			}
			for y := snap(b.Min.Y); y < y1; y += 2 * lw {
				dots.AddRect(paintengine2d.XYWH(gx, y, lw, lw))
			}
			if d == depth {
				for xx := gx; xx < x+es+l.S(2); xx += 2 * lw {
					dots.AddRect(paintengine2d.XYWH(xx, cy, lw, lw))
				}
			}
		}
		ctx.Save()
		ctx.ClipRect(b)
		ctx.DrawPath(dots, paintengine2d.Fill(Hex("#9f9d9a")))
		ctx.Restore()
	}
	if !leaf {
		l.Engine().Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, es, b.Dy()), expanded, fg)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + es + l.S(4)
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		l.Engine().ItemFocus(l, ctx, b, st)
	}
}

// DrawTableHeader is a tree-view column header: the top two thirds flat,
// the last third easing to × 0.96, a shade-0 top line and a shade-5
// bottom line, and the etched divider 4px in from the ends.
func (e clearlooksEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	if b.Dx() < 4*lw || b.Dy() < 6*lw {
		return
	}
	face := c.hdr[0]
	switch {
	case st.Pressed() && !st.Disabled():
		face = c.hdr[2]
	case st.Hovered() && !st.Disabled():
		face = c.hdr[1]
	}
	ctx.DrawRect(b, VGradient(b, face...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.s[0]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.s[5]))
	in := snap(l.S(4))
	if b.Dy() > 2*in+2*lw {
		dk := c.s[4]
		if c.flavour == clCleanlooks {
			dk = Hex("#cdc9c5")
		}
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-2*lw, b.Min.Y+in, lw, b.Dy()-2*in), paintengine2d.Fill(dk))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+in, lw, b.Dy()-2*in), paintengine2d.Fill(c.s[0]))
	}
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(lw, lw))
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		l.Engine().Arrow(l, ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(4), lb.Min.Y, aw, lb.Dy()), dir, c.fg)
	}
	fg := c.fg
	if st.Disabled() {
		fg = c.disFg
	}
	clEmboss(l, ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()), fg, AlignStart, 0, st.Disabled())
}

func (e clearlooksEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	fg := l.Engine().Face(l, ctx, b, RoleRow, st&^StateFocused)
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

// DrawToolBar is a GtkToolbar (toolbarstyle 1): the menubar's gentle
// gradient between a shade-0 top line and a shade-3 bottom line.
func (e clearlooksEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, VGradient(b, c.menubar...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.s[0]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.s[3]))
}

// DrawStatusBar is a GtkStatusbar: a shade-3 line over a shade-0 line at the
// top, panes split by etched lines and the resize grip's 45° lines.
func (e clearlooksEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw), paintengine2d.Fill(c.s[3]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, b.Dx(), lw), paintengine2d.Fill(c.s[0]))
	grip := l.S(16)
	if len(parts) > 0 {
		slot := (b.Dx() - grip) / float32(len(parts))
		for i, s := range parts {
			x := snap(b.Min.X + slot*float32(i))
			if i > 0 {
				ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+l.S(5), lw, b.Dy()-l.S(8)), paintengine2d.Fill(c.s[3]))
				ctx.DrawRect(paintengine2d.XYWH(x+lw, b.Min.Y+l.S(5), lw, b.Dy()-l.S(8)), paintengine2d.Fill(c.s[0]))
			}
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(6), b.Min.Y+2*lw, slot-l.S(10), b.Dy()-2*lw), c.fg, AlignStart, 0)
		}
	}
	// The resize grip: 45° lines, white then shade 4, every 4px.
	gx, gy := b.Max.X-lw, b.Max.Y-lw
	for i := 1; i <= 3; i++ {
		o := l.S(4) * float32(i)
		if o > b.Dy()-3*lw {
			break
		}
		ctx.DrawLine(paintengine2d.Pt(gx-o, gy), paintengine2d.Pt(gx, gy-o), paintengine2d.StrokePaint(c.s[4], lw))
		ctx.DrawLine(paintengine2d.Pt(gx-o+lw, gy), paintengine2d.Pt(gx, gy-o+lw), paintengine2d.StrokePaint(c.white, lw))
	}
}

// DrawTitleBar is a panel heading: bold title, then the subtitle, over an
// etched line.
func (e clearlooksEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*lw, b.Dx(), lw), paintengine2d.Fill(c.s[2]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw), paintengine2d.Fill(c.s[0]))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*lw), c.fg, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*lw), c.disFg, AlignStart, 0)
	}
}

// DrawAccordionHeader is a GtkExpander: the triangle, the bold title, the
// prelight box under the pointer.
func (e clearlooksEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := clColors(l)
	if !st.Disabled() && (st.Hovered() || st.Pressed()) {
		ctx.DrawRect(b, paintengine2d.Fill(c.prelight))
	}
	fg := c.fg
	if st.Disabled() {
		fg = c.disFg
	}
	es := l.S(16) // GtkExpander expander-size 16
	l.Engine().Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, es, b.Dy()), expanded, fg)
	clEmboss(l, ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+es+l.S(10), b.Min.Y, b.Dx()-es-l.S(14), b.Dy()), fg, AlignStart, 0, st.Disabled())
	if st.Focused() && !st.Disabled() {
		l.Engine().DrawFocusRing(l, ctx, b.Inset(snap(l.S(2))))
	}
}

// DrawSeparator is GTK 2's separator: shade 2 then shade 0.
func (e clearlooksEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := clColors(l)
	lw := clPx(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - lw
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(c.s[2]))
		ctx.DrawRect(paintengine2d.XYWH(x+lw, b.Min.Y, lw, b.Dy()), paintengine2d.Fill(c.s[0]))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - lw
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(c.s[2]))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+lw, b.Dx(), lw), paintengine2d.Fill(c.s[0]))
}

// DrawSplitter is a GtkPaned handle: a column of dots (shade 4 and a
// lighter pixel), prelit under the pointer.
func (e clearlooksEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := clColors(l)
	lw := clPx(l)
	fill := c.bg
	if st.Hovered() && !st.Disabled() {
		fill = c.prelight
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	n := 8
	pitch := 3 * lw
	span := float32(n-1) * pitch
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	ctr := b.Center()
	for i := 0; i < n; i++ {
		o := float32(i)*pitch - span*0.5
		var p paintengine2d.Point
		if vertical {
			p = paintengine2d.Pt(snap(ctr.X-lw), snap(ctr.Y+o))
		} else {
			p = paintengine2d.Pt(snap(ctr.X+o), snap(ctr.Y-lw))
		}
		if !b.Contains(p) || !b.Contains(paintengine2d.Pt(p.X+2*lw-0.1, p.Y+2*lw-0.1)) {
			continue
		}
		lt.AddRect(paintengine2d.XYWH(p.X, p.Y, 2*lw, 2*lw))
		dk.AddRect(paintengine2d.XYWH(p.X, p.Y, lw, lw))
	}
	ctx.DrawPath(lt, paintengine2d.Fill(clShade(c.s[4], 1.5).WithAlpha(0.8)))
	ctx.DrawPath(dk, paintengine2d.Fill(c.s[4].WithAlpha(0.9)))
}

// DrawTooltip is GTK 2's tooltip: the pale yellow window in a 1px black
// border.
func (e clearlooksEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := clColors(l)
	b = clSnap(b)
	lw := clPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.tip))
	clBorder(ctx, b, lw, c.tipText)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipText, AlignStart, 0)
}

// ---- packs ------------------------------------------------------------------------------------

// clPalette builds a Clearlooks palette from the gtkrc colours: bg, fg,
// base, text, selected bg / fg, insensitive fg and the accent.
func clPalette(bg, fg, base, text, sel, selFg, dis, accent string) Palette {
	b := Hex(bg)
	return Palette{
		Background: b, Surface: b, SurfaceAlt: b,
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.3),
		Border:  clShade(b, 0.665), Divider: clShade(b, 0.85),
		Text: Hex(fg), TextMuted: Hex(dis), TextOnAccent: Hex(selFg),
		Accent: Hex(accent), AccentHover: clShade(Hex(accent), 1.1), AccentPress: clShade(Hex(accent), 0.85),
		Danger: Hex("#cc0000"), Success: Hex("#4e9a06"), Warning: Hex("#c4a000"),
		Track: clShade(b, 0.85), Thumb: b,
		Field: Hex(base), FieldBorder: clShade(b, 0.62),
		Focus: clShade(b, 0.4), Selection: Hex(sel),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0), Highlight: Hex("#ffffff"),
		MenuHover: Hex(sel), MenuHoverBorder: clShade(Hex(sel), 0.8), MenuGutter: clShade(b, 1.03),
		BevelLight: Hex("#ffffff"), BevelDark: clShade(b, 0.665),
	}
}

func clPack(name, label string, year int, lineage, summary string, pal Palette, flavour int, extra map[string]string, m ChromeMetrics) ThemePack {
	tok := ThemeTokens{
		Engine:  "clearlooks",
		Bevel:   BevelSoftShadow,
		Family:  ThemeLight,
		Palette: pal,
		Metrics: m,
		Params:  map[string]float32{"flavour": float32(flavour)},
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = Hex(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.MenuHoverBorder}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.1), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: lineage, Summary: summary,
		Era: "Clearlooks", Palette: ThemeLight, Tokens: tok,
	}
}

func clearlooksPacks() []ThemePack {
	// GNOME 2.12's gtkrc: warm grey #efebe7, the blueish #628cb2 selection.
	cl := clPalette("#efebe7", "#101010", "#ffffff", "#000000", "#628cb2", "#ffffff", "#b5b3ac", "#628cb2")
	// Ubuntu 8.04's Human: the same grey, a pale orange #ffd799 selection
	// with black text, #ff6d0c widgets, #cc863e selected chrome.
	human := clPalette("#efebe7", "#101010", "#ffffff", "#000000", "#ffd799", "#000000", "#b1a498", "#cc863e")
	human.TextOnAccent = Hex("#000000")
	human.MenuHover, human.MenuHoverBorder = Hex("#ffdfad"), Hex("#c89b58")
	// Qt 4's Cleanlooks standard palette.
	qt := clPalette("#efebe7", "#000000", "#ffffff", "#000000", "#628cb2", "#ffffff", "#bebebe", "#628cb2")
	qt.Border, qt.Divider = Hex("#b8b5b2"), Hex("#cbc7c4")
	return []ThemePack{
		clPack("clearlooks", "Clearlooks", 2005, "GNOME",
			"GNOME 2.12's default: rounded gradient buttons, the striped candy progress bar, blue tab stripes.", cl, clClassic,
			map[string]string{
				"prelight": "#f5f3f0", "active": "#d4cfca", "listActive": "#a29e8e",
				"menu": "#f8f5f2", "notebook": "#eae4df", "check": "#2f3941", "checkHot": "#3c4a53",
				"tooltip": "#ffffbf", "tooltipText": "#000000",
			}, ChromeMetrics{}),
		clPack("human", "Human", 2006, "Ubuntu",
			"Ubuntu's Clearlooks-born look: warm greys, glassy buttons with an orange glow, orange progress and checks.", human, clHuman,
			map[string]string{
				"prelight": "#f3f0ed", "active": "#dbd3cc", "listActive": "#e3cfa9",
				"menu": "#f8f5f2", "notebook": "#efebe5", "check": "#101010",
				"spot": "#ff6d0c", "menuHi": "#ffdfad", "menuHi2": "#f4c378",
				"tooltip": "#f5f5b5", "tooltipText": "#000000",
				"caption": "#cc863e",
			}, ChromeMetrics{Checkbox: 14, Radio: 14}),
		clPack("cleanlooks", "Cleanlooks", 2007, "Qt",
			"Qt 4's clone of Clearlooks: the same greys and blue, Qt's cut corners, dotted focus and striped progress.", qt, clCleanlooks,
			map[string]string{
				"prelight": "#f9f5f1", "active": "#d0cac4", "listActive": "#918d7e",
				"menu": "#f9f5f1", "notebook": "#e9e4df", "check": "#000000",
				"tooltip": "#ffffdc", "tooltipText": "#000000",
			}, ChromeMetrics{}),
	}
}
