package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// nextEngine paints NeXTSTEP and OPENSTEP (1989–1996), the grey world of
// the 2-bit MegaPixel Display — black, #555555, #aaaaaa and white — and,
// over the same widget shapes, the window chrome of Window Maker (1997–),
// the NeXT-inspired window manager whose themes texture title bars, menus
// and dock tiles.
//
// NeXT shapes follow NeXTSTEP 3.3 and OPENSTEP 4.2 screenshots read pixel by
// pixel, the NeXTSTEP User Interface Guidelines (Release 3) and the
// NeXT-clone artwork of GNUstep and of Window Maker's own WINGs toolkit:
//
//   - Buttons: a flat #aaaaaa face, white top/left edge, #555555 inner and
//     black outer bottom/right edge. A pressed or "on" button lights up
//     white with the edge reversed (black top/left, white bottom/right); the
//     default button carries the engraved return-arrow glyph.
//   - Switches are raised 15px squares with an engraved check; radio
//     buttons are sunken round wells that show a white bead when selected
//     (NeXT's radios were never raised).
//   - Text fields: white in a two-pixel dark bezel. Pop-up buttons are raised
//     buttons with the "nibble", a shrunken raised box and its shadow.
//   - Scrollers: a 50% stippled trough, a raised knob with a dimple, both
//     arrow buttons together at the end ([ArrowsTogetherEnd]).
//   - Menus are columns of raised cells under a black title bar; the hot cell
//     lights up white and submenu cells end in the engraved ▷.
//   - Windows: the key window's black title bar with its bold centred title,
//     the miniaturize button at the left and the close button at the right,
//     a 1px black frame and the notched resize bar along the bottom.
//
// Window Maker packs keep those widgets and take the WM theme's textures for
// the chrome: FTitleBack / UTitleBack (focused / unfocused title bars),
// MenuTitleBack, MenuTextBack with HighlightColor, ResizebarBack (status
// bars) and IconBack (tool bars, like dock tiles). Textures use wrlib's
// geometry — a dgradient runs corner to corner with its isolines parallel to
// the other diagonal — and wmaker's bevel arithmetic: +80 highlight, -40
// shadow and a black edge on gradients; value×1.6 and value÷2 on solids.
// Title bars follow NewStyle = new: square button tiles cut from the title
// texture at both ends, each bevelled on its own.
//
// Pack data:
//
//	params "wm"       1 = Window Maker chrome (title tiles, menu entry
//	                  bevels); 0 = NeXT (inset title buttons)
//	       "justify"  title alignment: 0 left, 1 centre, 2 right
//	       "dither"   1 = 50% stippled scroller and slider troughs, 0 = flat
//	       "knob"     slider knob: 0 dimple (NeXTSTEP), 1 split (OPENSTEP)
//	       "cross"    close-glyph stroke in design px (1 = NeXTSTEP's hairline)
//	       "ftitle", "utitle", "mtitle", "mtext", "resize", "icon"
//	                  texture type: 0 solid, 1 hgradient, 2 vgradient,
//	                  3 dgradient (three or more colours = the m* variants)
//	extra  "<texture>0" … "<texture>N"  the texture colours in order
//	       "<texture>Light", "<texture>Dim"  a solid texture's bevel colours
//	       "ftitleText", "utitleText", "mtitleText", "mtextText",
//	       "mdisabled", "highlight", "highlightText"  WM theme text colours
//	       "hi", "lo", "black"  widget bevel: highlight, inner shadow, edge
//	       "lit"      the pressed / on face (NeXT's white)
//	       "trough"   the stipple colour laid over the face in troughs
//	       "rowSel"   selected list / table / tree rows (default: selection)
//	       "info"     tooltip fill
type nextEngine struct{ BaseEngine }

func init() {
	RegisterEngine(nextEngine{})
	for _, p := range nextPacks() {
		RegisterPack(p)
	}
}

func (nextEngine) ID() string { return "next" }

// DefaultMetrics are NeXT's proportions at the toolkit's 16px UI font (NeXT
// used 12pt Helvetica): 24px buttons become 28px, switches and radios stay
// 15px, scrollers 20px with their black outline, menu cells 20 → 24px.
func (nextEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Square:    true, BevelDepth: 1,
		ControlH: 28, FieldH: 26, ComboH: 26,
		Checkbox: 15, Radio: 15,
		MenuItemH: 24, MenuBarH: 26, TabH: 26, RowH: 22,
		TitleBar: 26, HeaderH: 22, ProgressH: 16, SliderH: 22, Thumb: 16,
		Scroll: 20, Pad: 10, FieldPad: 5, FocusWidth: 1, Border: 1,
		ToolBarH: 40, StatusBarH: 24, SpinnerW: 17, SwitchW: 40, SwitchH: 20,
	}
}

// ---- textures -------------------------------------------------------------------

// Window Maker texture types (params "<texture>").
const (
	nxSolid = iota
	nxHGrad
	nxVGrad
	nxDGrad
)

// nxTex is a resolved Window Maker texture: its colours as evenly spaced
// stops (wrlib's m*gradients), the stops shifted +80 and -40 for the bevel
// edges of gradients, the bevel colours of a solid and the average colour
// (for readable text on it).
type nxTex struct {
	kind            int
	stops, hi, lo   []paintengine2d.GradientStop
	light, dim, mid paintengine2d.Color
	// aux is wmaker's "normal" colour of the texture: a solid's colour, a
	// two-colour gradient's average, a multi-colour gradient's first stop.
	aux paintengine2d.Color
}

func (t *nxTex) solid() bool { return t.kind == nxSolid || len(t.stops) < 2 }

// shader is the texture laid over r, painted with stops (the texture's own
// or its bevel-shifted copy, so edges line up with the ramp under them).
func (t *nxTex) shader(r paintengine2d.Rect, stops []paintengine2d.GradientStop) paintengine2d.Paint {
	switch {
	case t.solid():
		return paintengine2d.Fill(stops[0].Color)
	case t.kind == nxHGrad:
		return HGradient(r, stops...)
	case t.kind == nxVGrad:
		return VGradient(r, stops...)
	}
	return nxDiag(r, stops)
}

func (t *nxTex) paint(ctx *paintengine2d.Context, r paintengine2d.Rect) {
	if !r.Empty() {
		ctx.DrawRect(r, t.shader(r, t.stops))
	}
}

// edgePaints are the light and dim edge paints of the texture over r.
func (t *nxTex) edgePaints(r paintengine2d.Rect) (paintengine2d.Paint, paintengine2d.Paint) {
	if t.solid() {
		return paintengine2d.Fill(t.light), paintengine2d.Fill(t.dim)
	}
	return t.shader(r, t.hi), t.shader(r, t.lo)
}

// raised is wmaker's raised bevel (RBEV_RAISED2 / WREL_RAISED) inside r.
func (t *nxTex) raised(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	lp, dp := t.edgePaints(r)
	nxBevel(ctx, r, u, lp, dp, paintengine2d.Fill(paintengine2d.RGB(0, 0, 0)))
}

// entry is wmaker's menu-entry bevel (WREL_MENUENTRY): light top and left,
// dim right and inner bottom, black bottom.
func (t *nxTex) entry(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	if r.Dx() < 3*u || r.Dy() < 3*u {
		return
	}
	lp, dp := t.edgePaints(r)
	x0, y0, w, h := r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, w, u), lp)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, u, h), lp)
	ctx.DrawRect(paintengine2d.XYWH(x0+w-u, y0, u, h), dp)
	ctx.DrawRect(paintengine2d.XYWH(x0+u, y0+h-2*u, w-2*u, u), dp)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0+h-u, w, u), paintengine2d.Fill(paintengine2d.RGB(0, 0, 0)))
}

// nxDiag is wrlib's dgradient: t = (x/w + y/h) / 2, so the ramp runs corner
// to corner with its isolines parallel to the other diagonal — a wide title
// bar shades mostly top to bottom, leaning with its length.
func nxDiag(r paintengine2d.Rect, stops []paintengine2d.GradientStop) paintengine2d.Paint {
	w, h := r.Dx(), r.Dy()
	d := w*w + h*h
	if w <= 0 || h <= 0 || d <= 0 {
		return paintengine2d.Fill(stops[0].Color)
	}
	return paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: r.Min,
		End:   paintengine2d.Pt(r.Min.X+2*w*h*h/d, r.Min.Y+2*h*w*w/d),
		Stops: stops,
	})
}

// nxTexDef is a texture's fallback when a pack does not describe it.
type nxTexDef struct {
	kind int
	cols []paintengine2d.Color
}

// nxTexture reads texture key from the pack ("<key>" type, "<key>0"… colours).
func nxTexture(l *Classic, key string, def nxTexDef) nxTex {
	var cols []paintengine2d.Color
	for i := 0; i < 16; i++ {
		c, ok := l.tokens.Extra[key+itoa(i)]
		if !ok {
			break
		}
		cols = append(cols, c)
	}
	kind := int(l.P(key, float32(def.kind)))
	if len(cols) == 0 {
		cols, kind = def.cols, def.kind
	}
	if len(cols) == 1 {
		kind = nxSolid
	}
	t := nxTex{kind: kind}
	n := len(cols)
	for i, c := range cols {
		at := float32(0)
		if n > 1 {
			at = float32(i) / float32(n-1)
		}
		t.stops = append(t.stops, Stop(at, c))
		t.hi = append(t.hi, Stop(at, nxAdd(c, 80)))
		t.lo = append(t.lo, Stop(at, nxAdd(c, -40)))
		t.mid.R += c.R / float32(n)
		t.mid.G += c.G / float32(n)
		t.mid.B += c.B / float32(n)
	}
	t.mid.A = 1
	t.aux = cols[0]
	if n == 2 {
		t.aux = Mix(cols[0], cols[1], 0.5)
	}
	light, dim := nxSolidBevel(cols[0])
	t.light = l.X(key+"Light", light)
	t.dim = l.X(key+"Dim", dim)
	return t
}

// nxAdd is wrlib's RAddOperation / RSubtractOperation on one colour.
func nxAdd(c paintengine2d.Color, d float32) paintengine2d.Color {
	return paintengine2d.RGB(clamp1(c.R+d/255), clamp1(c.G+d/255), clamp1(c.B+d/255))
}

// nxSolidBevel is wTextureMakeSolid's bevel pair: the HSV value ×1.6
// (capped) and ÷2; black gets #b6b6b6 / #616161.
func nxSolidBevel(c paintengine2d.Color) (light, dim paintengine2d.Color) {
	v := c.R
	if c.G > v {
		v = c.G
	}
	if c.B > v {
		v = c.B
	}
	if v <= 0 {
		return Hex("#b6b6b6"), Hex("#616161")
	}
	k := float32(1.6)
	if v*k > 1 {
		k = 1 / v
	}
	light = paintengine2d.RGB(c.R*k, c.G*k, c.B*k)
	dim = paintengine2d.RGB(c.R*0.5, c.G*0.5, c.B*0.5)
	return light, dim
}

// ---- resolved colour set -----------------------------------------------------------

// nx is a look's resolved NeXT / Window Maker colours, built once per look.
type nx struct {
	wm, dither, split bool
	justify           int
	cross             float32

	// The widget ramp: edge, inner shadow, face, highlight; the window,
	// field and trough.
	black, lo, face, hi, win, field, trough paintengine2d.Color
	text, dim, fieldTxt, sel, selTxt        paintengine2d.Color
	rowSel, rowSelTxt, ink                  paintengine2d.Color
	lit, litTxt, hover, info, infoTxt       paintengine2d.Color

	stipple                                     *paintengine2d.Image
	ftitle, utitle, mtitle, mtext, resize, icon nxTex
	ftitleTxt, utitleTxt, mtitleTxt, mtextTxt   paintengine2d.Color
	mdis, hl, hlTxt, fsub, resizeTxt, iconGlyph paintengine2d.Color
	arrowDark, arrowDim, arrowLight             paintengine2d.Color
}

type nxKey struct{}

// nxColors is the look's resolved colour set.
func nxColors(l *Classic) *nx {
	return l.Memo(nxKey{}, func() any { return nxBuild(l) }).(*nx)
}

func nxBuild(l *Classic) *nx {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c := &nx{
		wm:      l.P("wm", 0) != 0,
		dither:  l.P("dither", 1) != 0,
		split:   l.P("knob", 0) == 1,
		justify: int(l.P("justify", 1)),
		cross:   l.P("cross", 1),
		face:    p.SurfaceAlt,
		win:     p.Background,
		field:   p.Field,
		text:    p.Text,
		dim:     p.TextMuted,
	}
	c.black = l.X("black", black)
	c.hi = l.X("hi", white)
	c.lo = l.X("lo", Shade(c.face, -0.5))
	c.trough = l.X("trough", c.lo)
	if c.dither {
		c.stipple = nxStipple(c.trough)
	}
	// l.fieldText() would re-enter Memo (its lock is held while building).
	c.fieldTxt = ReadableOn(c.field, 4.5, c.text)
	c.sel = p.Selection
	if c.sel.A < 0.9 {
		c.sel = Mix(c.field, c.sel, c.sel.A)
	}
	c.selTxt = ReadableOn(c.sel, 4.5, c.fieldTxt, c.text, white)
	// Rows on the white list background: NeXT's light-grey text selection
	// would vanish into the window around the list, so selected rows take
	// the dark grey with white text (OpenStep's selected-row inversion).
	c.rowSel = l.X("rowSel", c.sel)
	c.rowSelTxt = ReadableOn(c.rowSel, 4.5, c.fieldTxt, white, black)
	// Glyphs on the face: black on NeXT grey, light on a dark ramp.
	c.ink = ReadableOn(c.face, 4.5, c.black, c.text, white)
	c.lit = l.X("lit", white)
	c.litTxt = ReadableOn(c.lit, 4.5, c.text, black, white)
	c.hover = Mix(c.face, c.hi, 0.35)
	c.info = l.X("info", white)
	c.infoTxt = ReadableOn(c.info, 4.5, black, white)

	cols := func(s ...paintengine2d.Color) []paintengine2d.Color { return s }
	c.ftitle = nxTexture(l, "ftitle", nxTexDef{nxSolid, cols(black)})
	c.utitle = nxTexture(l, "utitle", nxTexDef{nxSolid, cols(c.face)})
	c.mtitle = nxTexture(l, "mtitle", nxTexDef{nxSolid, cols(black)})
	c.mtext = nxTexture(l, "mtext", nxTexDef{nxSolid, cols(c.face)})
	c.resize = nxTexture(l, "resize", nxTexDef{nxSolid, cols(c.face)})
	c.icon = nxTexture(l, "icon", nxTexDef{nxSolid, cols(c.face)})
	c.ftitleTxt = l.X("ftitleText", white)
	c.utitleTxt = l.X("utitleText", black)
	c.mtitleTxt = l.X("mtitleText", white)
	c.mtextTxt = l.X("mtextText", c.text)
	c.mdis = l.X("mdisabled", c.dim)
	c.hl = l.X("highlight", white)
	c.hlTxt = l.X("highlightText", black)
	c.fsub = Mix(c.ftitleTxt, c.ftitle.mid, 0.35)
	c.resizeTxt = ReadableOn(c.resize.mid, 4.5, c.text, black, white)
	c.iconGlyph = ReadableOn(c.icon.mid, 4.5, black, white)
	// The cascade indicator is drawn with the bevel colours of a solid made
	// from the menu texture's normal colour (menu_item_auxtexture).
	c.arrowDark = black
	c.arrowLight, c.arrowDim = nxSolidBevel(c.mtext.aux)
	return c
}

// ---- drawing helpers ----------------------------------------------------------------

// nxU is one design pixel of the look in device pixels (1 at 1x, 2 at 2x).
func nxU(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// nxSnap puts a rect on the pixel grid so 1px lines stay crisp.
func nxSnap(b paintengine2d.Rect) paintengine2d.Rect {
	x0, y0 := snap(b.Min.X), snap(b.Min.Y)
	return paintengine2d.XYWH(x0, y0, snap(b.Max.X)-x0, snap(b.Max.Y)-y0)
}

func nxFill(ctx *paintengine2d.Context, r paintengine2d.Rect, c paintengine2d.Color) {
	if !r.Empty() {
		ctx.DrawRect(r, paintengine2d.Fill(c))
	}
}

// nxBevel is the NeXT raised edge inside r (NSDrawButton): light top and
// left, dim inner bottom and right, dark outer bottom and right.
func nxBevel(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32, light, dim, dark paintengine2d.Paint) {
	if r.Dx() < 3*u || r.Dy() < 3*u {
		return
	}
	x0, y0, w, h := r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, w-u, u), light)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, u, h-u), light)
	ctx.DrawRect(paintengine2d.XYWH(x0+u, y0+h-2*u, w-2*u, u), dim)
	ctx.DrawRect(paintengine2d.XYWH(x0+w-2*u, y0+u, u, h-2*u), dim)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0+h-u, w, u), dark)
	ctx.DrawRect(paintengine2d.XYWH(x0+w-u, y0, u, h), dark)
}

// raised is a NeXT button edge in the widget ramp.
func (c *nx) raised(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	nxBevel(ctx, r, u, paintengine2d.Fill(c.hi), paintengine2d.Fill(c.lo), paintengine2d.Fill(c.black))
}

// pushed is the pressed / "on" edge: black top and left, white bottom and
// right (WINGs WRPushed).
func (c *nx) pushed(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	if r.Dx() < 2*u || r.Dy() < 2*u {
		return
	}
	nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Max.Y-u, r.Dx(), u), c.hi)
	nxFill(ctx, paintengine2d.XYWH(r.Max.X-u, r.Min.Y, u, r.Dy()), c.hi)
	nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx()-u, u), c.black)
	nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, u, r.Dy()-u), c.black)
}

// ring2 paints two nested 1px rings: the outer (top-left, bottom-right) pair
// and the inner pair.
func ring2(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32, otl, obr, itl, ibr paintengine2d.Color) {
	if r.Dx() < 4*u || r.Dy() < 4*u {
		return
	}
	edge2 := func(r paintengine2d.Rect, tl, br paintengine2d.Color) {
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Max.Y-u, r.Dx(), u), br)
		nxFill(ctx, paintengine2d.XYWH(r.Max.X-u, r.Min.Y, u, r.Dy()), br)
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx()-u, u), tl)
		nxFill(ctx, paintengine2d.XYWH(r.Min.X, r.Min.Y, u, r.Dy()-u), tl)
	}
	edge2(r, otl, obr)
	edge2(r.Inset(u), itl, ibr)
}

// sunken is a scroll view / slider groove: dark then black top-left, white
// then face bottom-right (WINGs WRSunken, NSDrawGrayBezel).
func (c *nx) sunken(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	ring2(ctx, r, u, c.lo, c.hi, c.black, c.face)
}

// bezel is the text-field frame (NSDrawWhiteBezel): two dark lines top-left,
// face then white bottom-right.
func (c *nx) bezel(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	ring2(ctx, r, u, c.lo, c.hi, c.lo, c.face)
}

// groove is the etched box frame (WINGs WRGroove).
func (c *nx) groove(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32) {
	ring2(ctx, r, u, c.lo, c.hi, c.hi, c.lo)
}

// etched is a separator groove: a dark line over a light one.
func (c *nx) etched(ctx *paintengine2d.Context, x, y, n, u float32, vertical bool) {
	if n <= 0 {
		return
	}
	if vertical {
		nxFill(ctx, paintengine2d.XYWH(x, y, u, n), c.lo)
		nxFill(ctx, paintengine2d.XYWH(x+u, y, u, n), c.hi)
		return
	}
	nxFill(ctx, paintengine2d.XYWH(x, y, n, u), c.lo)
	nxFill(ctx, paintengine2d.XYWH(x, y+u, n, u), c.hi)
}

// troughFill fills r with the scroller trough: the 50% stipple (the trough
// colour on every other pixel over the face, WINGs' stippleGC), or flat.
func (c *nx) troughFill(ctx *paintengine2d.Context, r paintengine2d.Rect) {
	if r.Empty() {
		return
	}
	nxFill(ctx, r, c.face)
	if c.stipple != nil {
		nxDither(ctx, nxSnap(r), c.stipple)
	}
}

// nxStippleSide is the stipple tile: even, so tiles keep the pattern's
// phase wherever they are anchored.
const nxStippleSide = 64

// nxStipple is a checkerboard tile of col over transparency, built once per
// look. It is blitted 1:1 with nearest sampling — exact on the CPU and on
// GPUs whose mediump floats cannot address single pixels of a large screen.
func nxStipple(col paintengine2d.Color) *paintengine2d.Image {
	img := paintengine2d.NewImage(nxStippleSide, nxStippleSide)
	for y := 0; y < nxStippleSide; y++ {
		for x := y & 1; x < nxStippleSide; x += 2 {
			img.SetColor(x, y, col)
		}
	}
	return img
}

// nxDither lays the stipple tile over r in tiles anchored to multiples of
// its side, so every piece of a trough shares one phase.
func nxDither(ctx *paintengine2d.Context, r paintengine2d.Rect, tile *paintengine2d.Image) {
	const n = float32(nxStippleSide)
	nearest := paintengine2d.Paint{Filter: paintengine2d.FilterNearest}
	for ty := float32(math.Floor(float64(r.Min.Y/n))) * n; ty < r.Max.Y; ty += n {
		for tx := float32(math.Floor(float64(r.Min.X/n))) * n; tx < r.Max.X; tx += n {
			d := paintengine2d.XYWH(tx, ty, n, n).Intersect(r)
			if d.Empty() {
				continue
			}
			ctx.DrawImageRectPaint(tile, paintengine2d.XYWH(d.Min.X-tx, d.Min.Y-ty, d.Dx(), d.Dy()), d, nearest)
		}
	}
}

// dimple is the scroller knob's dimple (WINGs SCROLLER_DIMPLE): a sunken
// 6px hollow, black at the top left, white at the bottom right.
func (c *nx) dimple(ctx *paintengine2d.Context, ctr paintengine2d.Point, u float32) {
	r := 3 * u
	ctx.DrawCircle(ctr, r, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(ctr.X-r, ctr.Y-r), End: paintengine2d.Pt(ctr.X+r, ctr.Y+r),
		Stops: []paintengine2d.GradientStop{Stop(0.2, c.black), Stop(0.45, c.lo), Stop(0.72, c.hi), Stop(1, c.hi)},
	}))
}

// nxTri fills the NeXT scroll-arrow triangle (base as wide as it is tall)
// pointing dir, centred in b with base side s.
func nxTri(ctx *paintengine2d.Context, b paintengine2d.Rect, s float32, dir Direction, col paintengine2d.Color) {
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	h := s * 0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-h, cy+h)
		p.LineTo(cx+h, cy+h)
		p.LineTo(cx, cy-h)
	case DirDown:
		p.MoveTo(cx-h, cy-h)
		p.LineTo(cx+h, cy-h)
		p.LineTo(cx, cy+h)
	case DirLeft:
		p.MoveTo(cx+h, cy-h)
		p.LineTo(cx+h, cy+h)
		p.LineTo(cx-h, cy)
	default:
		p.MoveTo(cx-h, cy-h)
		p.LineTo(cx-h, cy+h)
		p.LineTo(cx+h, cy)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// nxArrow3D is the engraved NeXT arrow of submenu cells, pull-downs and
// browser branches: a triangle with a dark back edge, a dim upper slope and
// a light lower slope (wmaker's cascade indicator), filled with fill when it
// is set (a raised grey arrow for rows on the white field).
func nxArrow3D(ctx *paintengine2d.Context, b paintengine2d.Rect, s, u float32, dir Direction, dark, dim, light, fill paintengine2d.Color) {
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	h := snap(s * 0.5)
	o := u * 0.5
	line := func(ax, ay, bx, by float32, col paintengine2d.Color) {
		ctx.DrawLine(paintengine2d.Pt(ax, ay), paintengine2d.Pt(bx, by), paintengine2d.StrokePaint(col, u))
	}
	tri := func(ax, ay, bx, by, ex, ey float32) {
		if fill.A <= 0 {
			return
		}
		p := paintengine2d.NewPath()
		p.MoveTo(ax, ay)
		p.LineTo(bx, by)
		p.LineTo(ex, ey)
		p.Close()
		ctx.DrawPath(p, paintengine2d.Fill(fill))
	}
	if dir == DirDown {
		x0, x1, y0 := cx-h, cx+h, cy-h*0.6
		y1 := y0 + h*1.6
		tri(x0, y0, x1, y0, cx, y1)
		line(x0+o, y0+o, cx, y1, dim)
		line(x1-o, y0+o, cx, y1, light)
		line(x0, y0+o, x1, y0+o, dark)
		return
	}
	x0, y0, y1 := cx-h*0.6, cy-h, cy+h
	x1 := x0 + h*1.6
	tri(x0, y0, x0, y1, x1, cy)
	line(x0+o, y0+o, x1, cy, dim)
	line(x0+o, y1-o, x1, cy, light)
	line(x0+o, y0, x0+o, y1, dark)
}

// nxCross fills an × of stroke w from corner to corner of b (as quads, so
// the ends stay square like the bitmap glyph).
func nxCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if b.Empty() || w <= 0 {
		return
	}
	fill := paintengine2d.Fill(col)
	for _, d := range [2][4]float32{{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}, {b.Max.X, b.Min.Y, b.Min.X, b.Max.Y}} {
		x0, y0, x1, y1 := d[0], d[1], d[2], d[3]
		dx, dy := x1-x0, y1-y0
		n := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		ox, oy := -dy/n*w*0.5, dx/n*w*0.5
		p := paintengine2d.NewPath()
		p.MoveTo(x0+ox, y0+oy)
		p.LineTo(x1+ox, y1+oy)
		p.LineTo(x1-ox, y1-oy)
		p.LineTo(x0-ox, y0-oy)
		p.Close()
		ctx.DrawPath(p, fill)
	}
}

// nxMini is the miniaturize glyph: a small window, its frame one pixel and
// its title bar three pixels thick.
func nxMini(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, col paintengine2d.Color) {
	b = nxSnap(b)
	if b.Dx() < 4*u || b.Dy() < 5*u {
		return
	}
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), 3*u), col)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), col)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, u, b.Dy()), col)
	nxFill(ctx, paintengine2d.XYWH(b.Max.X-u, b.Min.Y, u, b.Dy()), col)
}

// check is the switch's engraved check mark (WINGs CHECK_BUTTON_ON): an ink
// stroke with a white line on its upper left and a dark one on its lower
// right, laid out on the box's 16-unit grid.
func (c *nx) check(ctx *paintengine2d.Context, box paintengine2d.Rect, u float32, ink paintengine2d.Color) {
	k := box.Dx() / 16
	pt := func(x, y float32) paintengine2d.Point {
		return paintengine2d.Pt(box.Min.X+x*k, box.Min.Y+y*k)
	}
	path := func(dx float32) *paintengine2d.Path {
		p := paintengine2d.NewPath()
		a, b, e := pt(5.5, 6), pt(5.5, 10.7), pt(12.7, 3.3)
		p.MoveTo(a.X+dx, a.Y)
		p.LineTo(b.X+dx, b.Y)
		p.LineTo(e.X+dx, e.Y)
		return p
	}
	w := 1.25 * u
	stroke := func(col paintengine2d.Color) paintengine2d.Paint {
		return paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}}
	}
	ctx.DrawPath(path(-u), stroke(c.hi))
	ctx.DrawPath(path(u), stroke(c.lo))
	ctx.DrawPath(path(0), stroke(ink))
}

// ret is the default button's return glyph (common_ret): an outlined hooked
// arrow, black on its upper-left edges and white on its lower-right ones,
// in a 15×10 box.
func (c *nx) ret(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, ink paintengine2d.Color) {
	k := b.Dx() / 15
	pt := func(x, y float32) paintengine2d.Point { return paintengine2d.Pt(b.Min.X+x*k, b.Min.Y+y*k) }
	seg := func(a, e paintengine2d.Point, col paintengine2d.Color) {
		ctx.DrawLine(a, e, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: u, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
	}
	tip, headTop, headIn := pt(0.5, 5), pt(5.5, 0.5), pt(5.5, 3.5)
	barL, barTL, barTR := pt(10.5, 3.5), pt(10.5, 0.5), pt(14.5, 0.5)
	barBR, shaftBL, headBot := pt(14.5, 7.5), pt(5.5, 7.5), pt(5.5, 9.5)
	seg(barBR, shaftBL, c.hi)
	seg(barTR, barBR, c.hi)
	seg(shaftBL, headBot, c.hi)
	seg(headTop, headIn, c.lo)
	seg(tip, headTop, ink)
	seg(headIn, barL, ink)
	seg(barL, barTL, ink)
	seg(barTL, barTR, ink)
	seg(headBot, tip, ink)
}

// nibble is the pop-up button indicator (WINGs POPUP_INDICATOR): a raised
// 9×6 box over a two-pixel dark shadow, at the right of b.
func (c *nx) nibble(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, disabled bool) {
	w, h := 9*u, 6*u
	x := snap(b.Max.X - w - 2*u)
	y := snap(b.Min.Y + (b.Dy()-h-2*u)*0.5)
	shadow := c.lo
	if disabled {
		shadow = Mix(c.lo, c.face, 0.5)
	}
	nxFill(ctx, paintengine2d.XYWH(x+2*u, y+2*u, w, h), shadow)
	box := paintengine2d.XYWH(x, y, w, h)
	nxFill(ctx, box, c.face)
	c.raised(ctx, box, u)
}

// ---- parts ------------------------------------------------------------------------

func (e nextEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	switch role {
	case RoleButton, RoleThumb, RoleCombo, RoleTool:
		return c.button(ctx, b, u, st)
	case RoleField, RoleCheck:
		fill := c.field
		if st.Disabled() {
			fill = c.face
		}
		nxFill(ctx, b, fill)
		c.bezel(ctx, b, u)
		return c.fieldTxt
	case RoleRow:
		if st.Checked() {
			nxFill(ctx, b, c.rowSel)
			return c.rowSelTxt
		}
		return c.fieldTxt
	case RoleMenu:
		if st.Hovered() || st.Pressed() || st.Checked() {
			nxFill(ctx, b, c.hl)
			return c.hlTxt
		}
		return c.mtextTxt
	case RoleTrack:
		c.troughFill(ctx, b)
		return fg
	case RoleTab:
		return fg
	}
	nxFill(ctx, b, c.face)
	return fg
}

// button paints a NeXT button face in state st and returns its label colour:
// raised on the face, lighter under the pointer, lit white and pushed in
// while pressed or latched on.
func (c *nx) button(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, st ControlState) paintengine2d.Color {
	on := !st.Disabled() && (st.Pressed() || (st.Toggle() && st.Checked()))
	switch {
	case on:
		nxFill(ctx, b, c.lit)
		c.pushed(ctx, b, u)
		return c.litTxt
	case st.Disabled():
		nxFill(ctx, b, c.face)
		c.raised(ctx, b, u)
		return c.dim
	case st.Hovered():
		nxFill(ctx, b, c.hover)
	default:
		nxFill(ctx, b, c.face)
	}
	c.raised(ctx, b, u)
	return c.text
}

// CheckIndicator is the NeXT switch: a raised square with the engraved
// check; pressed lights it up.
func (e nextEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := nxColors(l)
	u := nxU(l)
	box = nxSnap(box)
	face, ink := c.face, c.ink
	switch {
	case st.Disabled():
		ink = c.dim
	case st.Pressed():
		face, ink = c.lit, c.litTxt
	case st.Hovered():
		face = c.hover
	}
	nxFill(ctx, box, face)
	c.raised(ctx, box, u)
	if checked {
		c.check(ctx, box, u, ink)
	}
}

// RadioIndicator is the NeXT radio: a sunken round well — dark to white
// around its rim — with a white bead when selected.
func (e nextEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := nxColors(l)
	u := nxU(l)
	box = nxSnap(box)
	ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
	r := box.Dx() * 0.5
	rim := func(rad float32, a, b paintengine2d.Color) {
		ctx.DrawCircle(ctr, rad, paintengine2d.Linear(paintengine2d.LinearGradient{
			Start: paintengine2d.Pt(ctr.X-rad, ctr.Y-rad), End: paintengine2d.Pt(ctr.X+rad, ctr.Y+rad),
			Stops: []paintengine2d.GradientStop{Stop(0.38, a), Stop(0.62, b)},
		}))
	}
	outer, inner := c.lo, c.black
	if st.Disabled() {
		outer, inner = Mix(c.lo, c.face, 0.4), c.lo
	}
	rim(r, outer, c.hi)
	rim(r-u, inner, c.face)
	well := c.face
	if st.Hovered() && !st.Disabled() {
		well = c.hover
	}
	ctx.DrawCircle(ctr, r-2*u, paintengine2d.Fill(well))
	bead := paintengine2d.Color{}
	switch {
	case selected && st.Disabled():
		bead = Mix(c.face, c.hi, 0.5)
	case selected:
		bead = c.lit
	case st.Pressed():
		bead = Mix(c.face, c.lit, 0.5)
	}
	if bead.A > 0 {
		br := r - 3*u
		if c.split {
			// OPENSTEP's bead is shaded like a pearl.
			ctx.DrawCircle(paintengine2d.Pt(ctr.X+u*0.3, ctr.Y+u*0.3), br, paintengine2d.Radial(paintengine2d.RadialGradient{
				Center: paintengine2d.Pt(ctr.X-br*0.3, ctr.Y-br*0.3), Radius: br * 1.6,
				Stops: []paintengine2d.GradientStop{Stop(0, bead), Stop(0.55, bead), Stop(1, Mix(bead, c.face, 0.55))},
			}))
			return
		}
		ctx.DrawCircle(paintengine2d.Pt(ctr.X+u*0.3, ctr.Y+u*0.3), br, paintengine2d.Fill(bead))
	}
}

// Arrow is the NeXT scroll arrow: a solid black triangle as tall as wide.
func (nextEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	s *= 0.62
	if m := l.S(9); s > m {
		s = m
	}
	nxTri(ctx, b, s, dir, col)
}

// Expander is the browser branch arrow: the engraved ▷, ▽ when open.
func (nextEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := nxColors(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	nxArrow3D(ctx, b, l.S(8), nxU(l), dir, c.black, c.lo, c.hi, c.face)
}

// MenuHighlight is the NeXT hot cell: the highlight colour (white).
func (nextEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	nxFill(ctx, nxSnap(b), nxColors(l).hl)
}

func (nextEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := nxColors(l)
	if hot {
		return c.hlTxt
	}
	return c.mtextTxt
}

// Fields show only the caret when focused, as NeXT did.
func (nextEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is OpenStep's NSDottedFrameRect just inside b.
func (nextEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	DottedRect(ctx, nxSnap(b).Inset(nxU(l)*2), nxColors(l).text)
}

// ---- scrollbars ---------------------------------------------------------------------

// ScrollBarStyle: 20px scrollers (a black outline, a 1px margin, 16px knob
// and buttons) with both buttons together at the end. NeXT put them at the
// min end — the bottom of a vertical scroller, the left of a horizontal one;
// the toolkit's placement is shared by both axes, so both go to the end.
func (nextEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	// Grouped arrows: at the bottom of vertical scrollers, at the left of
	// horizontal ones (NeXTSTEP UI Guidelines).
	return ScrollBarStyle{Thickness: 20, Arrows: ArrowsTogetherEnd, HArrows: ArrowsTogetherStart, HArrowsSet: true, ArrowLen: 18, MinThumb: 18}
}

func (e nextEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := nxColors(l)
	u := nxU(l)
	bar := nxSnap(p.Bar)
	if bar.Dx() < 6*u || bar.Dy() < 6*u {
		return
	}
	// The black outline and the face margin inside it.
	nxFill(ctx, bar, c.black)
	in := bar.Inset(u)
	nxFill(ctx, in, c.face)
	lane := in.Inset(u)
	// The trough, less the button block.
	tr := lane
	if !p.Dec.Empty() {
		d := nxSnap(p.Dec)
		if vertical {
			tr.Max.Y = d.Min.Y
		} else {
			tr.Max.X = d.Min.X
		}
	}
	thumb := p.Thumb
	if st.Disabled || thumb.Empty() {
		// Nothing to scroll: NeXT shows a plain stippled strip.
		c.troughFill(ctx, lane)
		return
	}
	c.troughFill(ctx, tr)
	btn := func(seg paintengine2d.Rect, first bool, dir Direction, part ScrollPart) {
		if seg.Empty() {
			return
		}
		seg = nxSnap(seg)
		var r paintengine2d.Rect
		side := 16 * u
		if vertical {
			y := seg.Min.Y + u
			if !first {
				y = seg.Min.Y
			}
			r = paintengine2d.XYWH(lane.Min.X, y, lane.Dx(), side)
		} else {
			x := seg.Min.X + u
			if !first {
				x = seg.Min.X
			}
			r = paintengine2d.XYWH(x, lane.Min.Y, side, lane.Dy())
		}
		r = r.Intersect(lane)
		if r.Empty() {
			return
		}
		s := StateNone
		if st.Pressed == part {
			s = StatePressed
		}
		ink := c.button(ctx, r, u, s)
		g := r
		if st.Pressed == part {
			g = g.Translate(paintengine2d.Pt(u, u))
		}
		nxTri(ctx, g, snap(l.S(9)), dir, ink)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	btn(p.Dec, true, dec, ScrollDec)
	btn(p.Inc, false, inc, ScrollInc)
	// The knob: a raised button with the dimple at its centre.
	k := nxSnap(thumb)
	if vertical {
		k = paintengine2d.XYWH(lane.Min.X, k.Min.Y, lane.Dx(), k.Dy())
	} else {
		k = paintengine2d.XYWH(k.Min.X, lane.Min.Y, k.Dx(), lane.Dy())
	}
	k = k.Intersect(tr)
	if k.Empty() {
		return
	}
	nxFill(ctx, k, c.face)
	c.raised(ctx, k, u)
	if k.Dx() >= 10*u && k.Dy() >= 10*u {
		c.dimple(ctx, paintengine2d.Pt(snap((k.Min.X+k.Max.X-u)*0.5), snap((k.Min.Y+k.Max.Y-u)*0.5)), u)
	}
}

// DrawScrollBar is the bare fallback: a stippled track and a dimpled knob.
func (e nextEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := nxColors(l)
	u := nxU(l)
	c.troughFill(ctx, nxSnap(track))
	if thumb.Empty() {
		return
	}
	k := nxSnap(thumb).Intersect(nxSnap(track))
	nxFill(ctx, k, c.face)
	c.raised(ctx, k, u)
	if k.Dx() >= 10*u && k.Dy() >= 10*u {
		c.dimple(ctx, paintengine2d.Pt(snap((k.Min.X+k.Max.X-u)*0.5), snap((k.Min.Y+k.Max.Y-u)*0.5)), u)
	}
}

// ---- frames -------------------------------------------------------------------------

func (nextEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is an NXBox: an etched groove with its title centred on the
// top edge (a raised box takes the button edge instead).
func (e nextEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	f := l.body
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5 - u)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if raised {
		nxFill(ctx, frame, c.face)
		c.raised(ctx, frame, u)
	} else {
		c.groove(ctx, frame, u)
	}
	if title == "" {
		return
	}
	tw := f.Advance(title) + l.S(10)
	if tw > b.Dx()-l.S(16) {
		tw = b.Dx() - l.S(16)
	}
	if tw <= 0 {
		return
	}
	tb := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-tw)*0.5), b.Min.Y, snap(tw), f.Height())
	nxFill(ctx, tb, c.win)
	fg := c.text
	l.drawFittedText(ctx, f, title, tb, fg, AlignCenter, 0)
}

// Window geometry. NeXT title bars are 23px at 12pt (a black line, the 1px
// bevel ring round the bar and the black line under it); the title bar
// grows with the toolkit's bold font. The resize bar is 9px.
func nxTitleH(l *Classic) float32 {
	f := l.bold
	if f == nil {
		f = l.body
	}
	h := f.Height() + l.S(6)
	if m := l.S(22); h < m {
		h = m
	}
	return snap(h)
}

func nxResizeH(l *Classic) float32 { return snap(l.S(9)) }

func (nextEngine) WindowFrameInsets(l *Classic) Insets {
	u := nxU(l)
	return Insets{Top: u + nxTitleH(l), Right: u, Bottom: nxResizeH(l), Left: u}
}

// nxTitleRect is the title bar of a frame b: inside the frame's top and left
// line, running out over its right line (the bar's own black edge is it).
func nxTitleRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	u := nxU(l)
	b = nxSnap(b)
	return paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-u, nxTitleH(l))
}

// nxTitleButton is the close (right) or miniaturize (left) button of a frame:
// NeXT's 15px buttons inset in the bar, or Window Maker's square tiles cut
// from the bar's ends.
func nxTitleButton(l *Classic, b paintengine2d.Rect, right bool) paintengine2d.Rect {
	c := nxColors(l)
	u := nxU(l)
	t := nxTitleRect(l, b)
	if c.wm {
		side := t.Dy()
		if right {
			return paintengine2d.XYWH(t.Max.X-side, t.Min.Y, side, side)
		}
		return paintengine2d.XYWH(t.Min.X, t.Min.Y, side, side)
	}
	side := snap(t.Dy() - 7*u)
	top := snap(t.Min.Y + (t.Dy()-u-side)*0.5)
	if right {
		return paintengine2d.XYWH(snap(b.Max.X)-3*u-side-u, top, side, side)
	}
	return paintengine2d.XYWH(t.Min.X+3*u, top, side, side)
}

// nxFrameFits reports whether a frame b has room for its chrome (smaller
// frames paint as a plain face, without buttons).
func nxFrameFits(l *Classic, b paintengine2d.Rect) bool {
	b = nxSnap(b)
	t := nxTitleH(l)
	return b.Dx() >= t*3 && b.Dy() >= t+nxResizeH(l)+2*nxU(l)
}

// WindowCloseRect is the close button at the right of the title bar.
// No drop shadows: NeXTSTEP and Window Maker menus and tooltips float flat.
func (nextEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (nextEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// StyleHint: NeXT dialogs put the default button last, at the bottom right,
// and NSForm right-aligns its titles against the fields.
func (nextEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintFormLabelsRight {
		return 1
	}
	return 0
}

func (nextEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	if !nxFrameFits(l, b) {
		return paintengine2d.Rect{}
	}
	return nxTitleButton(l, b, true)
}

// DrawWindowFrame is a NeXT window: 1px black frame, the title bar (black
// when key, light grey otherwise — or the WM theme's FTitleBack / UTitleBack)
// with the miniaturize button at the left, the close button at the right
// and the bold title, and the notched resize bar at the bottom.
func (e nextEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	t := nxTitleRect(l, b)
	rh := nxResizeH(l)
	if !nxFrameFits(l, b) {
		nxFill(ctx, b, c.face)
		return
	}
	nxFill(ctx, b, c.black)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X+u, t.Max.Y, b.Dx()-2*u, b.Dy()-t.Dy()-rh-u), c.win)
	tex, txt := &c.ftitle, c.ftitleTxt
	if !st.Active {
		tex, txt = &c.utitle, c.utitleTxt
	}
	left, right := t.Min.X, t.Max.X
	miniB := nxTitleButton(l, b, false)
	closeB := nxTitleButton(l, b, true)
	if c.wm {
		// NewStyle = new: the texture over the whole bar, cut into three
		// separately bevelled pieces.
		tex.paint(ctx, t)
		mid := t
		if st.CanClose {
			mid.Min.X, mid.Max.X = miniB.Max.X, closeB.Min.X
			left, right = mid.Min.X, mid.Max.X
		}
		tex.raised(ctx, mid, u)
		if st.CanClose {
			e.wmTitleButton(l, ctx, t, miniB, tex, txt, false, false, false)
			e.wmTitleButton(l, ctx, t, closeB, tex, txt, true, st.CloseHot, st.ClosePress)
		}
	} else {
		tex.paint(ctx, t)
		tex.raised(ctx, t, u)
		if st.CanClose {
			e.nextTitleButton(l, ctx, miniB, false, false, false)
			e.nextTitleButton(l, ctx, closeB, true, st.CloseHot, st.ClosePress)
			left, right = miniB.Max.X+u*2, closeB.Min.X-u*2
		}
	}
	if title != "" {
		f := l.bold
		if f == nil {
			f = l.body
		}
		tb := paintengine2d.XYWH(left+l.S(6), t.Min.Y+u, right-left-l.S(12), t.Dy()-3*u)
		l.drawFittedText(ctx, f, title, tb, txt, nxAlign(c.justify), 0)
	}
	e.resizeBar(l, ctx, paintengine2d.XYWH(b.Min.X+u, b.Max.Y-rh, b.Dx()-2*u, rh-u))
}

func nxAlign(j int) Align {
	switch j {
	case 0:
		return AlignStart
	case 2:
		return AlignEnd
	}
	return AlignCenter
}

// nextTitleButton is a NeXT title-bar button: a raised light-grey square
// with a black glyph; pressed lights it up.
func (e nextEngine) nextTitleButton(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, close, hot, pressed bool) {
	c := nxColors(l)
	u := nxU(l)
	st := StateNone
	if hot {
		st |= StateHovered
	}
	if pressed {
		st |= StatePressed
	}
	ink := c.button(ctx, r, u, st)
	g := r.Inset(snap(r.Dx() * 0.2))
	if pressed {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	if close {
		w := c.cross * u
		nxCross(ctx, g.Inset(w*0.35), ink, w)
		return
	}
	nxMini(ctx, g.Inset(u*0.5), u, ink)
}

// wmTitleButton is a Window Maker new-style button: a square tile of the
// title texture, bevelled on its own, the glyph in the title colour; pushed
// it turns white with a black outline and a black glyph.
func (e nextEngine) wmTitleButton(l *Classic, ctx *paintengine2d.Context, bar, r paintengine2d.Rect, tex *nxTex, glyph paintengine2d.Color, close, hot, pressed bool) {
	c := nxColors(l)
	u := nxU(l)
	if pressed {
		nxFill(ctx, r, paintengine2d.RGB(1, 1, 1))
		ctx.DrawRect(r.Inset(u*0.5), paintengine2d.StrokePaint(c.black, u))
		glyph = c.black
	} else {
		// The tile continues the bar's ramp.
		ctx.Save()
		ctx.ClipRect(r)
		tex.paint(ctx, bar)
		ctx.Restore()
		if hot {
			nxFill(ctx, r, c.hi.WithAlpha(0.18))
		}
		tex.raised(ctx, r, u)
	}
	side := snap(r.Dy() * 0.46)
	g := paintengine2d.XYWH(snap(r.Min.X+(r.Dx()-side)*0.5), snap(r.Min.Y+(r.Dy()-side)*0.5), side, side)
	if pressed {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	if close {
		nxCross(ctx, g.Inset(u*0.5), glyph, 1.6*u)
		return
	}
	nxMini(ctx, g, u, glyph)
}

// resizeBar is the bottom strip of a window (and the status bar): the resize
// texture under a dark and a light line, split near both ends by notches
// that mark the corner handles.
func (e nextEngine) resizeBar(l *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect) {
	c := nxColors(l)
	u := nxU(l)
	r = nxSnap(r)
	if r.Dx() < 4*u || r.Dy() < 3*u {
		return
	}
	c.resize.paint(ctx, r)
	lp, dp := c.resize.edgePaints(r)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), u), dp)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y+u, r.Dx(), u), lp)
	cw := snap(l.S(28))
	if r.Dx() < cw*3 {
		return
	}
	for _, x := range [2]float32{r.Min.X + cw, r.Max.X - cw - 2*u} {
		ctx.DrawRect(paintengine2d.XYWH(x, r.Min.Y+2*u, u, r.Dy()-2*u), dp)
		ctx.DrawRect(paintengine2d.XYWH(x+u, r.Min.Y+2*u, u, r.Dy()-2*u), lp)
	}
}

// DrawWindowBackground is NeXT's flat light grey.
func (nextEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	nxFill(ctx, b, nxColors(l).win)
}

// DrawTabPane is the raised page under the tabs (the tab bar paints over its
// top edge and opens it under the selected tab).
func (nextEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nxColors(l)
	b = nxSnap(b)
	nxFill(ctx, b, c.face)
	c.raised(ctx, b, nxU(l))
}

// ---- controls -----------------------------------------------------------------------

// DrawButton is the NeXT push button; the default button shows the return
// glyph at its right.
func (e nextEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	fg := c.button(ctx, b, u, st)
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	pad := float32(6)
	if st.Primary() {
		gw, gh := snap(l.S(15)), snap(l.S(10))
		if b.Dx() > gw*3 && b.Dy() > gh+4*u {
			g := paintengine2d.XYWH(lb.Max.X-gw-l.S(5), snap(lb.Min.Y+(lb.Dy()-gh)*0.5), gw, gh)
			c.ret(ctx, g, u, fg)
			lb.Max.X = g.Min.X
			pad = 2
		}
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, pad)
	if st.Focused() && !st.Disabled() {
		DottedRect(ctx, b.Inset(3*u), c.text)
	}
}

func (e nextEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := l.metrics.Checkbox
	box := nxSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggleLabel(l, ctx, b, box, st, label)
}

func (e nextEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := nxSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggleLabel(l, ctx, b, box, st, label)
}

// toggleLabel draws a switch / radio caption with the dotted focus frame.
func (nextEngine) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := nxColors(l)
	if label == "" {
		if st.Focused() {
			DottedRect(ctx, box.Inset(-nxU(l)).Intersect(b), c.text)
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		DottedRect(ctx, labelFocusRect(l.body, label, lb, b), c.text)
	}
}

func (e nextEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	if !st.Disabled() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	// A disabled field greys to the face with dark-grey text.
	c := nxColors(l)
	e.Face(l, ctx, b, RoleField, st)
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = l.S(5)
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	show := text
	if show == "" {
		show = placeholder
	}
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.faceOrBody(face)
	f.Draw(ctx, show, paintengine2d.Pt(inner.Min.X-scrollX, inner.Min.Y+(inner.Dy()-f.Height())*0.5), c.dim)
	ctx.Restore()
}

// DrawComboBox is the NXPopUpButton: a raised button with the current item
// and the nibble at its right; open, it lights up like a pressed button.
func (e nextEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	bs := st
	if open {
		bs |= StatePressed
	}
	fg := c.button(ctx, b, u, bs)
	in := b
	if open && !st.Disabled() {
		in = in.Translate(paintengine2d.Pt(u, u))
	}
	c.nibble(ctx, paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx()-l.S(4), in.Dy()), u, st.Disabled())
	tb := paintengine2d.XYWH(in.Min.X+l.S(7), in.Min.Y, in.Dx()-l.S(7)-l.S(20), in.Dy())
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		DottedRect(ctx, b.Inset(3*u), c.text)
	}
}

// DrawSpinner is a pair of small NeXT buttons with scroll arrows.
func (e nextEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, dir Direction, hover, press bool) {
		s := StateNone
		switch {
		case st.Disabled():
			s = StateDisabled
		case press:
			s = StatePressed
		case hover:
			s = StateHovered
		}
		ink := c.button(ctx, r, u, s)
		if press && !st.Disabled() {
			r = r.Translate(paintengine2d.Pt(u, u))
		}
		side := r.Dx() * 0.5
		if m := r.Dy() * 0.55; side > m {
			side = m
		}
		nxTri(ctx, r, snap(side), dir, ink)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

// DrawTabBar is the strip the tabs stand on; its bottom line is the raised
// pane's top edge.
func (e nextEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	nxFill(ctx, b, c.win)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), c.hi)
}

// DrawTab is an OPENSTEP NSTabView tab: a trapezoid with slanted sides, a
// white left slope and top, a black right slope over a dark one; the
// selected tab opens into the pane.
func (e nextEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	top := b.Min.Y + l.S(3)
	if !selected {
		top += l.S(2)
	}
	top = snap(top)
	h := b.Max.Y - top
	// NSTabView's slopes run about half a tab height; they stay inside the
	// 14px each side the tab strip reserves around a label.
	slant := snap(h * 0.5)
	if m := snap(l.S(12)); slant > m {
		slant = m
	}
	if slant*2 > b.Dx()-l.S(8) {
		slant = snap((b.Dx() - l.S(8)) * 0.5)
	}
	if h < 4*u || slant < 0 {
		return
	}
	x0, x1, y1 := b.Min.X, b.Max.X, b.Max.Y
	face := c.face
	switch {
	case selected:
	case !st.Disabled() && (st.Hovered() || st.Pressed()):
		face = c.hover
	default:
		// Tabs behind the pane are a shade darker than the one in front.
		face = Mix(c.face, c.lo, 0.12)
	}
	shape := paintengine2d.NewPath()
	shape.MoveTo(x0, y1)
	shape.LineTo(x0+slant, top)
	shape.LineTo(x1-slant, top)
	shape.LineTo(x1, y1)
	shape.Close()
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawPath(shape, paintengine2d.Fill(face))
	o := u * 0.5
	line := func(ax, ay, bx, by float32, col paintengine2d.Color) {
		ctx.DrawLine(paintengine2d.Pt(ax, ay), paintengine2d.Pt(bx, by), paintengine2d.StrokePaint(col, u))
	}
	line(x1-slant-u, top+o, x1-o, y1, c.lo)
	line(x1-slant, top+o, x1+o, y1, c.black)
	line(x0+o, y1, x0+slant+o, top+o, c.hi)
	line(x0+slant, top+o, x1-slant-u, top+o, c.hi)
	ctx.Restore()
	if selected {
		// The pane's white top edge stops at the tab: it opens into the page.
		nxFill(ctx, paintengine2d.XYWH(x0+u, y1-u, x1-x0-2*u, u), face)
	} else {
		nxFill(ctx, paintengine2d.XYWH(x0, y1-u, x1-x0, u), c.hi)
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dim
	}
	// The label sits at mid height, where the tab is half a slope wider.
	lb := paintengine2d.XYWH(x0+slant*0.5, top+u, x1-x0-slant, y1-top-u)
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, 0)
	if st.Focused() && selected {
		f := l.body
		w := f.Advance(label) + l.S(6)
		if w > lb.Dx() {
			w = lb.Dx()
		}
		fh := f.Height() + l.S(2)
		if fh > lb.Dy() {
			fh = lb.Dy()
		}
		DottedRect(ctx, nxSnap(paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-fh)*0.5, w, fh)), c.text)
	}
}

// DrawPanel: a flat panel is the window face; a raised one a bevelled box.
func (e nextEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := nxColors(l)
	b = nxSnap(b)
	nxFill(ctx, b, c.face)
	if raised {
		c.raised(ctx, b, nxU(l))
	}
}

// nxMenuCell is the full-width cell of a menu row (the popup clips two
// pixels inside its frame; cells start there).
func nxMenuCell(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	ch := MenuChromeFor(l)
	in := nxU(l)
	if in < 2 {
		in = 2
	}
	return nxSnap(paintengine2d.XYWH(b.Min.X-ch.PadL+in, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*in, b.Dy()))
}

// cell paints one menu cell: the MenuTextBack texture with its bevel (a
// NeXT button edge, or Window Maker's menu-entry edge); hot, the highlight
// fills it inside the light top and left edge.
func (c *nx) cell(ctx *paintengine2d.Context, r paintengine2d.Rect, u float32, hot bool) {
	if r.Dx() < 3*u || r.Dy() < 3*u {
		return
	}
	c.mtext.paint(ctx, r)
	if c.wm {
		c.mtext.entry(ctx, r, u)
	} else {
		c.mtext.raised(ctx, r, u)
	}
	if hot {
		nxFill(ctx, paintengine2d.XYWH(r.Min.X+u, r.Min.Y+u, r.Dx()-2*u, r.Dy()-3*u), c.hl)
	}
}

// DrawMenuBar is a horizontal run of NeXT menu cells.
func (e nextEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nxColors(l)
	c.cell(ctx, nxSnap(b), nxU(l), false)
}

// DrawMenuTitle: an open (or pressed) title lights up with the highlight
// colour like a NeXT cell; under the pointer it rises as a cell of its own.
func (e nextEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	fg := c.mtextTxt
	switch {
	case st.Disabled():
		fg = c.mdis
	case open || st.Pressed():
		c.cell(ctx, b, u, true)
		fg = c.hlTxt
	case st.Hovered():
		c.cell(ctx, b, u, false)
	}
	// NeXT menus never underlined mnemonics; the keys still work.
	l.drawLabeled(ctx, l.body, label, -1, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()-u), fg)
	if st.Focused() && !open {
		DottedRect(ctx, b.Inset(3*u), fg)
	}
}

// DrawMenuFrame is the NeXT menu panel: a dark / black outline and, in the
// padding above the rows, a strip of the menu title bar (MenuTitleBack).
func (e nextEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	nxFill(ctx, b, c.black)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-u, u), c.lo)
	nxFill(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, u, b.Dy()-u), c.lo)
	in := paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-2*u, b.Dy()-2*u)
	c.mtext.paint(ctx, in)
	ch := MenuChromeFor(l)
	// Too thin for its bevel and title: just the texture, as a band.
	if th := snap(ch.PadT) - u; th >= 2*u {
		c.mtitle.paint(ctx, paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), th))
	}
}

// DrawMenuItem is a NeXT / Window Maker menu cell: key equivalents at the
// right in the label colour, the engraved ▷ for submenus, a check or WM's
// diamond (radio) in the gutter; separators are etched grooves.
func (e nextEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := nxColors(l)
	u := nxU(l)
	ch := MenuChromeFor(l)
	cell := nxMenuCell(l, b)
	if row.Separator {
		y := snap((b.Min.Y+b.Max.Y)*0.5) - u
		c.etched(ctx, cell.Min.X+l.S(3), y, cell.Dx()-l.S(6), u, false)
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	c.cell(ctx, cell, u, hot)
	fg := c.mtextTxt
	switch {
	case st.Disabled():
		fg = c.mdis
	case hot:
		fg = c.hlTxt
	}
	ink := b
	ink.Max.Y -= u
	// The gutter: WM's check and diamond indicators, or the item's icon.
	gw := ch.CheckCol()
	gb := paintengine2d.XYWH(b.Min.X, ink.Min.Y, gw, ink.Dy())
	switch {
	case row.Radio && row.Checked:
		// wmaker's MENU_RADIO_INDICATOR: a thick diamond ring.
		s := snap(l.S(9))
		d := paintengine2d.XYWH(snap(gb.Min.X+(gb.Dx()-s)*0.5), snap(gb.Min.Y+(gb.Dy()-s)*0.5), s, s)
		w := 1.8 * u
		ctx.DrawPath(DiamondPath(d.Inset(w*0.5)), paintengine2d.Paint{Color: fg, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: w, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
	case row.Checked:
		s := snap(l.S(10))
		d := paintengine2d.XYWH(snap(gb.Min.X+(gb.Dx()-s)*0.5), snap(gb.Min.Y+(gb.Dy()-s)*0.5), s, s)
		PixelTick(ctx, d, fg)
	case row.Icon != IconNone:
		// The gutter icon, kept inside its cell at every scale.
		side := IconSizePixels(l.IconSize()) * l.Scale()
		if m := gw - 2*u; side > m {
			side = m
		}
		if m := ink.Dy() - 4*u; side > m {
			side = m
		}
		if side > 2*u {
			l.drawToolIcon(ctx, paintengine2d.XYWH(gb.Min.X+(gb.Dx()-side)*0.5, gb.Min.Y+(gb.Dy()-side)*0.5, side, side), row.Icon, fg)
		}
	}
	f := l.body
	ty := ink.Min.Y + (ink.Dy()-f.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = l.S(10)
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), ink.Min.Y, aw, ink.Dy())
		light := c.arrowLight
		if hot {
			light = Mix(c.hl, c.arrowDim, 0.35)
		}
		nxArrow3D(ctx, ab, snap(l.S(9)), u, DirRight, c.arrowDark, c.arrowDim, light, paintengine2d.Color{})
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := f.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw - l.S(4)
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

// DrawProgressBar: NeXT had no progress view; apps drew a sunken dark track
// with a light raised bar. Busy slides a light block.
func (e nextEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	track := c.lo
	if st.Disabled() {
		track = Mix(c.lo, c.face, 0.5)
	}
	nxFill(ctx, b, track)
	c.sunken(ctx, b, u)
	in := b.Inset(2 * u)
	var bar paintengine2d.Rect
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		w := snap(in.Dx() * 0.3)
		x := in.Min.X + (in.Dx()+w)*phase - w
		bar = paintengine2d.XYWH(snap(x), in.Min.Y, w, in.Dy()).Intersect(in)
	} else {
		t = clamp1(t)
		bar = paintengine2d.XYWH(in.Min.X, in.Min.Y, snap(in.Dx()*t), in.Dy())
	}
	if bar.Dx() < 3*u {
		return
	}
	fill := c.face
	if st.Disabled() {
		fill = Mix(c.face, c.lo, 0.3)
	}
	nxFill(ctx, bar, fill)
	c.raised(ctx, bar, u)
}

// DrawSlider is NXSlider: a sunken stippled bar with a raised knob sliding
// inside it, dimpled (NeXTSTEP) or split by a groove (OPENSTEP).
func (e nextEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	t = clamp1(t)
	bh := snap(l.S(18))
	if bh > b.Dy() {
		bh = b.Dy()
	}
	bar := paintengine2d.XYWH(b.Min.X, snap(b.Min.Y+(b.Dy()-bh)*0.5), b.Dx(), bh)
	if bar.Dx() < 8*u || bar.Dy() < 6*u {
		return
	}
	c.sunken(ctx, bar, u)
	lane := bar.Inset(2 * u)
	if st.Disabled() {
		nxFill(ctx, lane, c.face)
	} else {
		c.troughFill(ctx, lane)
	}
	kw := snap(l.S(18))
	if kw > lane.Dx()/2 {
		kw = snap(lane.Dx() / 2)
	}
	kx := snap(lane.Min.X + (lane.Dx()-kw)*t)
	k := paintengine2d.XYWH(kx, lane.Min.Y, kw, lane.Dy())
	face := c.face
	if st.Hovered() && !st.Disabled() {
		face = c.hover
	}
	nxFill(ctx, k, face)
	c.raised(ctx, k, u)
	if !st.Disabled() {
		if c.split {
			x := snap((k.Min.X + k.Max.X - u) * 0.5)
			c.etched(ctx, x-u, k.Min.Y+u, k.Dy()-3*u, u, true)
		} else if k.Dx() >= 10*u && k.Dy() >= 10*u {
			c.dimple(ctx, paintengine2d.Pt(snap((k.Min.X+k.Max.X-u)*0.5), snap((k.Min.Y+k.Max.Y-u)*0.5)), u)
		}
	}
	if st.Focused() {
		DottedRect(ctx, k.Inset(2*u), c.text)
	}
}

// DrawSwitch: NeXT had no toggle switch; it is drawn as a two-position
// slider — a sunken slot, stippled when off and lit when on, and a dimpled
// knob.
func (e nextEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := nxColors(l)
	u := nxU(l)
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := nxSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*u || track.Dy() < 8*u {
		return
	}
	c.sunken(ctx, track, u)
	lane := track.Inset(2 * u)
	switch {
	case st.Disabled():
		nxFill(ctx, lane, c.face)
	case on:
		nxFill(ctx, lane, c.lit)
	default:
		c.troughFill(ctx, lane)
	}
	kw := snap(lane.Dy())
	kx := lane.Min.X
	if on {
		kx = lane.Max.X - kw
	}
	k := paintengine2d.XYWH(kx, lane.Min.Y, kw, lane.Dy())
	face := c.face
	if st.Hovered() && !st.Disabled() {
		face = c.hover
	}
	nxFill(ctx, k, face)
	c.raised(ctx, k, u)
	if !st.Disabled() && k.Dx() >= 10*u {
		c.dimple(ctx, paintengine2d.Pt(snap((k.Min.X+k.Max.X-u)*0.5), snap((k.Min.Y+k.Max.Y-u)*0.5)), u)
	}
	if label != "" {
		fg := c.text
		if st.Disabled() {
			fg = c.dim
		}
		lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
		l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
		if st.Focused() {
			DottedRect(ctx, labelFocusRect(l.body, label, lb, b), c.text)
		}
	} else if st.Focused() {
		DottedRect(ctx, track, c.text)
	}
}

// DrawListRow: lists sit on the white field, where NeXT's light-grey
// selection would read as a hole; selected rows take the pack's row colour
// (dark grey with white text). Lists do not hot-track.
func (e nextEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	selected := st.Checked()
	c := nxColors(l)
	fg := c.fieldTxt
	if selected {
		nxFill(ctx, nxSnap(b), c.rowSel)
		fg = c.rowSelTxt
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a browser-style row: the engraved branch arrow and the
// selection across the row.
func (e nextEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	selected := st.Checked()
	c := nxColors(l)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	fg := c.fieldTxt
	if selected {
		nxFill(ctx, nxSnap(b), c.rowSel)
		fg = c.rowSelTxt
	}
	x := b.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, fg)
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
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableHeader: a row of NeXT buttons (the NSTableView header); pressed
// lights up, the sorted column shows a small black arrow.
func (e nextEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	fg := c.button(ctx, b, u, st&^StateFocused)
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	aw := float32(0)
	if sorted {
		aw = l.S(12)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		nxTri(ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(4), lb.Min.Y, aw, lb.Dy()), snap(l.S(7)), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()-u), fg, AlignStart, 0)
}

func (e nextEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	selected := st.Checked()
	c := nxColors(l)
	fg := c.fieldTxt
	if selected {
		nxFill(ctx, nxSnap(b), c.rowSel)
		fg = c.rowSelTxt
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

// DrawToolBar: the face with an etched foot on NeXT; Window Maker packs lay
// the IconBack texture under the buttons, like a row of dock tiles.
func (e nextEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	if c.wm {
		c.icon.paint(ctx, b)
		c.icon.raised(ctx, b, u)
		return
	}
	nxFill(ctx, b, c.win)
	c.etched(ctx, b.Min.X, b.Max.Y-2*u, b.Dx(), u, false)
}

// DrawToolButton: NeXT's icon buttons are always-raised buttons (Mail's
// Delete, Compose, Find). Window Maker packs make them dock tiles of the
// IconBack texture, pushed in when pressed or latched on.
func (e nextEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	on := !st.Disabled() && (st.Pressed() || (st.Toggle() && st.Checked()))
	var fg paintengine2d.Color
	if c.wm {
		// A dock tile: IconBack, raised; lighter under the pointer, pushed
		// in with a dark top-left edge when pressed or latched on.
		fg = c.iconGlyph
		c.icon.paint(ctx, b)
		switch {
		case on:
			nxFill(ctx, b, c.black.WithAlpha(0.2))
			ring2(ctx, b, u, c.black, c.icon.light, c.icon.dim, c.icon.mid)
		case st.Hovered() && !st.Disabled():
			nxFill(ctx, b, c.hi.WithAlpha(0.2))
			c.icon.raised(ctx, b, u)
		default:
			c.icon.raised(ctx, b, u)
		}
		if st.Disabled() {
			fg = Mix(fg, c.icon.mid, 0.55)
		}
	} else {
		fg = c.button(ctx, b, u, st)
	}
	in := b
	if on {
		in = in.Translate(paintengine2d.Pt(u, u))
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, in.Dy())
	x := in.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, in.Min.Y+(in.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(in.Min.X+(in.Dx()-iconSide)*0.5, in.Min.Y+(in.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		right := in.Max.X - pad
		if right < x {
			right = x
		}
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, in.Min.Y, right-x, in.Dy()))
		l.body.Draw(ctx, label, paintengine2d.Pt(x, in.Min.Y+(in.Dy()-l.body.Height())*0.5), fg)
		ctx.Restore()
	}
	if st.Focused() && !st.Disabled() {
		DottedRect(ctx, b.Inset(3*u), fg)
	}
}

// DrawStatusBar is the window's resize bar grown tall enough for text: the
// ResizebarBack texture, its corner notches and grooves between the parts.
func (e nextEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	e.resizeBar(l, ctx, b)
	if len(parts) == 0 {
		return
	}
	cw := snap(l.S(28)) + 2*u
	if b.Dx() < cw*3 {
		cw = 0
	}
	x0, x1 := b.Min.X+cw, b.Max.X-cw
	slot := (x1 - x0) / float32(len(parts))
	lp, dp := c.resize.edgePaints(b)
	for i, s := range parts {
		x := x0 + slot*float32(i)
		if i > 0 {
			sx := snap(x)
			ctx.DrawRect(paintengine2d.XYWH(sx, b.Min.Y+2*u, u, b.Dy()-2*u), dp)
			ctx.DrawRect(paintengine2d.XYWH(sx+u, b.Min.Y+2*u, u, b.Dy()-2*u), lp)
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(8), b.Min.Y+2*u, slot-l.S(12), b.Dy()-2*u), c.resizeTxt, AlignStart, 0)
	}
}

// DrawTitleBar is client-side window chrome: the focused title texture
// (NeXT's black key-window bar) with the bold title in the title colour.
func (e nextEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	c.ftitle.paint(ctx, b)
	c.ftitle.raised(ctx, b, u)
	f := l.bold
	if f == nil {
		f = l.body
	}
	tb := paintengine2d.XYWH(b.Min.X+l.metrics.Pad, b.Min.Y+u, b.Dx()-2*l.metrics.Pad, b.Dy()-3*u)
	if tb.Empty() {
		return
	}
	tw := f.Advance(title)
	sw := float32(0)
	if subtitle != "" {
		sw = l.S(12) + l.body.Advance(subtitle)
	}
	x := tb.Min.X
	switch c.justify {
	case 1:
		x = tb.Min.X + (tb.Dx()-tw-sw)*0.5
	case 2:
		x = tb.Max.X - tw - sw
	}
	if x < tb.Min.X {
		x = tb.Min.X
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, tb.Min.Y, tb.Max.X-x, tb.Dy()), c.ftitleTxt, AlignStart, 0)
		x += tw + l.S(12)
	}
	if subtitle != "" && x < tb.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, tb.Min.Y, tb.Max.X-x, tb.Dy()), c.fsub, AlignStart, 0)
	}
}

// DrawAccordionHeader is a raised NeXT cell with the engraved branch arrow.
func (e nextEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	fg := c.button(ctx, b, u, st&^(StateFocused|StateHovered))
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	nxArrow3D(ctx, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, l.S(12), lb.Dy()-u), l.S(8), u, dir, c.black, c.lo, c.hi, paintengine2d.Color{})
	tb := paintengine2d.XYWH(lb.Min.X+l.S(24), lb.Min.Y, lb.Dx()-l.S(28), lb.Dy()-u)
	l.drawFittedText(ctx, l.body, title, tb, fg, AlignStart, 0)
	if st.Focused() {
		DottedRect(ctx, labelFocusRect(l.body, title, tb, b), fg)
	}
}

func (nextEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := nxColors(l)
	u := nxU(l)
	if vertical {
		c.etched(ctx, snap((b.Min.X+b.Max.X)*0.5)-u, snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4)), u, true)
		return
	}
	c.etched(ctx, snap(b.Min.X), snap((b.Min.Y+b.Max.Y)*0.5)-u, snap(b.Dx()), u, false)
}

// DrawSplitter is the NXSplitView divider: the face with a dimple at its
// centre.
func (nextEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	face := c.win
	if st.Hovered() || st.Pressed() {
		face = c.hover
	}
	nxFill(ctx, b, face)
	if b.Dx() >= 7*u && b.Dy() >= 7*u {
		c.dimple(ctx, paintengine2d.Pt(snap((b.Min.X+b.Max.X-u)*0.5), snap((b.Min.Y+b.Max.Y-u)*0.5)), u)
	}
}

// DrawTooltip: a white label in a black frame.
func (nextEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := nxColors(l)
	u := nxU(l)
	b = nxSnap(b)
	nxFill(ctx, b, c.black)
	nxFill(ctx, b.Inset(u), c.info)
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(5)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.infoTxt, AlignStart, 0)
}

// DrawMessageIcon: NeXT alert panels showed the application's icon on its
// tile; here a bold alert glyph sits on such a tile (IconBack on WM packs).
func (e nextEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	if icon == IconNone {
		return
	}
	c := nxColors(l)
	u := nxU(l)
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	tile := nxSnap(paintengine2d.XYWH(b.Min.X+(b.Dx()-s)*0.5, b.Min.Y+(b.Dy()-s)*0.5, s, s))
	c.icon.paint(ctx, tile)
	c.icon.raised(ctx, tile, u)
	ink := c.iconGlyph
	glyph := ""
	switch icon {
	case IconInfo:
		glyph = "i"
	case IconWarning:
		glyph = "!"
	case IconQuestion:
		glyph = "?"
	case IconError:
		g := tile.Inset(snap(s * 0.3))
		nxCross(ctx, paintengine2d.XYWH(g.Min.X, g.Min.Y-u*0.5, g.Dx(), g.Dy()), ink, snap(s*0.11))
		return
	default:
		l.drawToolIcon(ctx, tile.Inset(snap(s*0.2)), icon, ink)
		return
	}
	f := l.title
	if f == nil || f.Height() > s*1.25 {
		f = l.BoldFont()
	}
	l.drawFittedText(ctx, f, glyph, paintengine2d.XYWH(tile.Min.X, tile.Min.Y, tile.Dx()-u, tile.Dy()-u), ink, AlignCenter, 0)
}

// ---- packs ----------------------------------------------------------------------------

// nextPack builds a pack for the NeXT engine. Textures are given as
// "type:#c0,#c1,…" strings for readability (type s, h, v or d).
func nextPack(name, label, lineage string, year int, summary string, fam ThemeName, pal Palette, extra map[string]string, params map[string]float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "next",
		Bevel:   BevelClassic3D,
		Family:  fam,
		Palette: pal,
		Params:  map[string]float32{},
	}
	for k, v := range params {
		tok.Params[k] = v
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		if kind, cols, ok := nxParseTexture(v); ok {
			tok.Params[k] = float32(kind)
			for i, cv := range cols {
				tok.Extra[k+itoa(i)] = hexColor(cv)
			}
			continue
		}
		tok.Extra[k] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.12), Border: pal.Focus}
	era := EraNext
	if lineage != "NeXT" {
		era = lineage
	}
	tok.Era = era
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: lineage, Summary: summary,
		Era: era, Palette: fam, Tokens: tok,
	}
}

// nxParseTexture reads "h:#505a5e,#202a2e" (s solid, h / v / d gradient).
func nxParseTexture(s string) (int, []string, bool) {
	if len(s) < 3 || s[1] != ':' {
		return 0, nil, false
	}
	var k int
	switch s[0] {
	case 's':
		k = nxSolid
	case 'h':
		k = nxHGrad
	case 'v':
		k = nxVGrad
	case 'd':
		k = nxDGrad
	default:
		return 0, nil, false
	}
	var cols []string
	start := 2
	for i := 2; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				cols = append(cols, s[start:i])
			}
			start = i + 1
		}
	}
	return k, cols, len(cols) > 0
}

// nextPalette is the shared palette of a NeXT-engine pack: window face,
// field, text, disabled text, text selection, inner shadow (divider) and
// the lit face (menu hover).
func nextPalette(face, field, text, muted, sel, dark, lit string) Palette {
	f := hexColor(face)
	return Palette{
		Background: f, Surface: f, SurfaceAlt: f,
		Border: hexColor("#000000"), Divider: hexColor(dark),
		Text: hexColor(text), TextMuted: hexColor(muted), TextOnAccent: hexColor(face),
		Accent: hexColor(text), AccentHover: hexColor(muted), AccentPress: hexColor(text),
		Field: hexColor(field), FieldBorder: hexColor(dark),
		Focus: hexColor(text), Selection: hexColor(sel),
		Track: hexColor(dark), Thumb: f,
		Highlight: hexColor(lit).WithAlpha(0.6), Shadow: paintengine2d.RGBA(0, 0, 0, 0.35),
		MenuHover: hexColor(lit), MenuHoverBorder: hexColor("#000000"), MenuGutter: f,
		Danger: hexColor("#9a0000"), Success: hexColor("#005c00"), Warning: hexColor("#6e5000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.35),
		BevelLight: hexColor("#ffffff"), BevelDark: hexColor(dark),
	}
}

func nextPacks() []ThemePack {
	// NeXT: the four greys of the MegaPixel Display. Text selections are
	// light grey on white, the lit (pressed, hot) face is white.
	next := nextPalette("#aaaaaa", "#ffffff", "#000000", "#555555", "#aaaaaa", "#555555", "#ffffff")
	// Not historical: the same shapes over a darker ramp.
	night := nextPalette("#474747", "#1c1c1c", "#e8e8e8", "#9a9a9a", "#6a6a6a", "#262626", "#bdbdbd")
	night.BevelLight, night.TextOnAccent = hexColor("#8c8c8c"), hexColor("#000000")
	night.Danger, night.Success, night.Warning = hexColor("#ff7b6e"), hexColor("#6fd36f"), hexColor("#f0c050")
	// WINGs, Window Maker's NeXT-clone toolkit: gray #aeaaae, darkGray #515551.
	wings := nextPalette("#aeaaae", "#ffffff", "#000000", "#515551", "#aeaaae", "#515551", "#ffffff")
	// Night Sky's app colours are the theme's own: MenuTextBack for the
	// face, its deepest stop for fields, HighlightColor for selections.
	sky := nextPalette("#444661", "#202841", "#ffffff", "#9e9a9e", "#ffe0ac", "#1c1e39", "#ffe0ac")
	sky.BevelLight, sky.TextOnAccent = hexColor("#9496b1"), hexColor("#000000")
	sky.Danger, sky.Success, sky.Warning = hexColor("#ff8a7a"), hexColor("#7ad37a"), hexColor("#ffe0ac")

	nextChrome := map[string]string{
		"ftitle": "s:#000000", "ftitleLight": "#aaaaaa", "ftitleDim": "#555555",
		"utitle": "s:#aaaaaa", "mtitle": "s:#000000", "mtitleLight": "#aaaaaa", "mtitleDim": "#555555",
		"mtext": "s:#aaaaaa", "resize": "s:#aaaaaa", "icon": "s:#aaaaaa",
		"ftitleText": "#ffffff", "utitleText": "#000000", "mtitleText": "#ffffff", "mtextText": "#000000",
		"mdisabled": "#555555", "highlight": "#ffffff", "highlightText": "#000000",
		"hi": "#ffffff", "lo": "#555555", "black": "#000000", "lit": "#ffffff", "trough": "#555555",
		"rowSel": "#555555",
	}
	nightChrome := map[string]string{
		"ftitle": "s:#000000", "ftitleLight": "#6a6a6a", "ftitleDim": "#2a2a2a",
		"utitle": "s:#5e5e5e", "mtitle": "s:#000000", "mtitleLight": "#6a6a6a", "mtitleDim": "#2a2a2a",
		"mtext": "s:#474747", "mtextLight": "#8c8c8c", "mtextDim": "#262626",
		"resize": "s:#474747", "resizeLight": "#8c8c8c", "resizeDim": "#262626",
		"icon": "s:#474747", "iconLight": "#8c8c8c", "iconDim": "#262626",
		"ftitleText": "#ffffff", "utitleText": "#e8e8e8", "mtitleText": "#ffffff", "mtextText": "#e8e8e8",
		"mdisabled": "#8a8a8a", "highlight": "#bdbdbd", "highlightText": "#000000",
		"hi": "#8c8c8c", "lo": "#262626", "black": "#000000", "lit": "#bdbdbd", "trough": "#262626",
		"info": "#2a2a2a",
	}
	// Window Maker packs: the theme files' textures verbatim, rgb:rr/gg/bb
	// written as #rrggbb; ResizebarBack falls back to WindowMaker.in's
	// (solid, "rgb:aa/aa/aa") where a theme leaves it out.
	wm := func(m map[string]string) map[string]string {
		base := map[string]string{
			"hi": "#ffffff", "lo": "#515551", "black": "#000000", "lit": "#ffffff", "trough": "#515551",
			"resize": "s:#aaaaaa", "rowSel": "#515551",
		}
		for k, v := range m {
			base[k] = v
		}
		return base
	}
	wmDefault := wm(map[string]string{
		// Themes/Default.style
		"ftitle": "h:#505a5e,#202a2e", "utitle": "h:#c2c0c5,#828085",
		"mtitle": "h:#505a5e,#202a2e", "mtext": "h:#c2c0c5,#828085", "icon": "d:#a6a6b6,#515561",
		"ftitleText": "#ffffff", "utitleText": "#000000", "mtitleText": "#ffffff", "mtextText": "#000000",
		"mdisabled": "#666666", "highlight": "#ffffff", "highlightText": "#000000",
	})
	wmOpenStep := wm(map[string]string{
		// Themes/OpenStep.style
		"ftitle": "d:#000010,#202070", "utitle": "d:#909090,#d0d0d0",
		"mtitle": "d:#000020,#202070", "mtext": "h:#d0d0d0,#808080", "icon": "d:#a6a6b6,#515561",
		"ftitleText": "#ffffff", "utitleText": "#333333", "mtitleText": "#ffffff", "mtextText": "#000000",
		"mdisabled": "#666666", "highlight": "#ffffff", "highlightText": "#000000",
	})
	wmNight := wm(map[string]string{
		// Styles/NightSky.style
		"ftitle": "h:#000000,#3d637f,#315c77,#333f3e", "utitle": "h:#000000,#595969,#444661,#202831",
		"mtitle": "h:#000000,#3d637f,#315c77,#333f3e", "mtext": "h:#000000,#494c63,#444661,#202841",
		"resize": "h:#000000,#494c63,#444661,#202841", "icon": "h:#000000,#595969,#444661,#202831",
		"ftitleText": "#ffffff", "utitleText": "#bebebe", "mtitleText": "#ffffff", "mtextText": "#ffffff",
		"mdisabled": "#9e9a9e", "highlight": "#ffe0ac", "highlightText": "#000000",
		"hi": "#9496b1", "lo": "#1c1e39", "lit": "#ffe0ac", "trough": "#1c1e39", "info": "#202841",
		"rowSel": "#ffe0ac",
	})
	wmSilk := wm(map[string]string{
		// Themes/SteelBlueSilk.style
		"ftitle": "d:#18191f,#939abd,#616185,#616185,#5f5f83,#555575,#59597a,#555575,#939abd",
		"utitle": "h:#989aa6,#9fa1b5,#86879b",
		"mtitle": "v:#18191f,#474967,#413b6d", "mtext": "h:#384246,#707080,#4a4a61",
		"icon": "d:#666666,#6d6aa4,#564e8c,#41436c,#464771,#595090", "ftitleText": "#ffffff",
		"utitleText": "#333333", "mtitleText": "#ffffff", "mtextText": "#ffffff",
		"mdisabled": "#999999", "highlight": "#ffffff", "highlightText": "#000000",
	})
	return []ThemePack{
		nextPack("next", "NeXTSTEP", "NeXT", 1989,
			"NeXTSTEP's four greys: black key-window titles, white-lit buttons, stippled scrollers, menus of raised cells.",
			ThemeLight, next, nextChrome, map[string]float32{"justify": 1, "dither": 1, "cross": 1}),
		nextPack("next-night", "NeXTSTEP Night", "NeXT", 1989,
			"Not historical: NeXTSTEP never had a dark mode — its shapes and black titles on a dark grey ramp.",
			ThemeDark, night, nightChrome, map[string]float32{"justify": 1, "dither": 1, "cross": 1}),
		nextPack("openstep", "OPENSTEP", "NeXT", 1994,
			"OPENSTEP 4: NeXTSTEP's greys with tab views, split slider knobs, pearl radio beads and a bolder close box.",
			ThemeLight, next, nextChrome, map[string]float32{"justify": 1, "dither": 1, "cross": 1.6, "knob": 1}),
		nextPack("wmaker-default", "Window Maker", "Window Maker", 1997,
			"Window Maker's Default theme: slate title gradients, grey gradient menus, dock-tile tool bars.",
			ThemeLight, wings, wmDefault, map[string]float32{"wm": 1, "justify": 0, "dither": 1, "cross": 1}),
		nextPack("wmaker-openstep", "Window Maker OpenStep", "Window Maker", 1997,
			"Window Maker's OpenStep theme: midnight-blue diagonal titles over light grey menus.",
			ThemeLight, wings, wmOpenStep, map[string]float32{"wm": 1, "justify": 0, "dither": 1, "cross": 1}),
		nextPack("wmaker-night", "Window Maker Night Sky", "Window Maker", 1998,
			"Window Maker's Night Sky: black-to-teal title gradients, dark menus lit in warm cream.",
			ThemeDark, sky, wmNight, map[string]float32{"wm": 1, "justify": 1, "dither": 1, "cross": 1}),
		nextPack("wmaker-steelbluesilk", "Window Maker SteelBlueSilk", "Window Maker", 1999,
			"Window Maker's SteelBlueSilk: multi-stop steel-blue silk titles, dark silk menus, violet dock tiles.",
			ThemeLight, wings, wmSilk, map[string]float32{"wm": 1, "justify": 1, "dither": 1, "cross": 1}),
	}
}
