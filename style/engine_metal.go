package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// metalEngine paints Swing's cross-platform "Java look and feel", Metal, in
// its two stock themes: Steel (JDK 1.2, 1998) and Ocean (Java 5, 2004).
//
// Everything is drawn from the theme's eight colours the way Sun's design
// guidelines use them: three primaries for selection, focus and large
// coloured areas, three secondaries for borders, pressed faces and the
// canvas, black for text and white for highlights. The shapes follow
// observation of the shipped looks:
//
//   - the flush 3D border: a dark ring with a white line inside its top and
//     left edges and outside its right and bottom edges; the innermost dark
//     ring stays open at the two corners where the white lines would touch
//     it (buttons, fields, check boxes, arrow buttons). Default buttons wear
//     a two-pixel ring.
//   - the "bumps" drag texture: a lattice of light dots, each with a dark
//     dot one pixel down and right, every four pixels with staggered rows —
//     on scroll thumbs, slider knobs, tool bar grips, split pane dividers
//     and window title bars.
//   - tabs with a slanted top-left corner that share their border line with
//     the neighbour.
//   - focus as a thin primary-colour rectangle around a control's label
//     (the current list row, table row and tree label too); controls and
//     menus labelled in the bold control font; scroll bars with an arrow
//     button at each end.
//
// Ocean keeps the shapes and softens the faces: buttons, check boxes,
// radios, scroll thumbs, slider knobs, the active title bar and the
// selected tab turn into light blue gradients, the menu bar and tool bars
// into a white-to-grey wash, and scroll thumbs trade their bumps for a
// three-line grip.
//
// Pack data:
//
//	extra   "primary1" … "primary3", "secondary1" … "secondary3", "black",
//	        "white": the theme colours. "disabled" (dimmed text and
//	        borders), "tabSelected", "tabIdle", "tabIdleHi", "grid",
//	        "menuHotTop", "frameOff", "grooveOff", "grooveOffHi",
//	        "groupText", "accel", "divider"; Ocean's washes: "grad1" …
//	        "grad3" (button face), "barTop"/"barBottom"/"barLine" (menu
//	        bar), "toolTop"/"toolBottom", "knob0" … "knob2" (slider knob),
//	        "fill0" … "fill3" (slider fill).
//	params  "ocean" 1 = Ocean faces; "bold" 0 = plain control labels
//	        (Swing's swing.boldMetal=false); "gradPeak", "gradBack": where
//	        the button wash peaks white and returns to grad1 (0.3, 0.6).
type metalEngine struct{ BaseEngine }

func init() {
	RegisterEngine(metalEngine{})
	for _, p := range metalPacks() {
		RegisterPack(p)
	}
}

func (metalEngine) ID() string { return "metal" }

// DefaultMetrics are Metal's proportions: Swing's 12pt Dialog font is the
// toolkit's 16px UI font, so Metal's sizes carry over one to one — 17px
// scroll bars, 13px check boxes, square corners, a two-pixel view frame.
func (metalEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		Square: true, ViewFrame: 2, BevelDepth: 1,
		ControlH: 26, FieldH: 24, ComboH: 26,
		Checkbox: 13, Radio: 13,
		MenuItemH: 23, MenuBarH: 25, TabH: 26, RowH: 22,
		TitleBar: 26, HeaderH: 22, ProgressH: 14, SliderH: 24, Thumb: 15,
		Scroll: 17, Pad: 10, FieldPad: 5, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24, SpinnerW: 17,
		SwitchW: 38, SwitchH: 20,
	}
}

// ---- resolved colours -------------------------------------------------------

// mtl is a look's resolved Metal colours, gradients, bump tiles and faces,
// built once per look.
type mtl struct {
	ocean, bold bool
	u           float32 // one Metal pixel in device pixels

	p1, p2, p3, s1, s2, s3, black, white paintengine2d.Color

	text, dis, field, fieldText, sel, selText paintengine2d.Color
	focus, itemFocus, tabFocus, pressed       paintengine2d.Color
	tabSel, tabIdle, tabIdleHi                paintengine2d.Color
	tabEdgeSel, tabEdgeIdle, tabBand          paintengine2d.Color
	menuHot, menuHotTop, hotText, accel       paintengine2d.Color
	accelHot, info, infoBorder, infoText      paintengine2d.Color
	treeLine, grid, groupLine, groupText      paintengine2d.Color
	frameOn, frameOff, grooveOn, grooveOnHi   paintengine2d.Color
	grooveOff, grooveOffHi, titleOn, titleOff paintengine2d.Color
	barLine, trackShadow, knobEdge, knobHi    paintengine2d.Color
	tabDis, disFill, knobPress                paintengine2d.Color
	iconFill, iconDark, iconHi                [4]paintengine2d.Color
	btnGrad, knobGrad, fillGrad               []paintengine2d.GradientStop
	barGrad, toolGrad                         []paintengine2d.GradientStop
	thumbBumps, gripBumps, gripHotBumps       *paintengine2d.Image
	titleBumps, titleOffBumps, knobBumps      *paintengine2d.Image
	body, bold2, small                        *Font
}

type mtlKey struct{}

// mtlColors is the look's resolved Metal set (built once per look).
func mtlColors(l *Classic) *mtl {
	return l.Memo(mtlKey{}, func() any { return mtlBuild(l) }).(*mtl)
}

// mtlSteel is the Steel theme (DefaultMetalTheme, as the Java Look and
// Feel Design Guidelines tabulate it); every key the engine reads has a
// Steel value, so a theme.json pack only lists what differs.
var mtlSteel = map[string]string{
	"primary1": "#666699", "primary2": "#9999cc", "primary3": "#ccccff",
	"secondary1": "#666666", "secondary2": "#999999", "secondary3": "#cccccc",
	"black": "#000000", "white": "#ffffff",
	"disabled": "#999999", "tabSelected": "#cccccc", "tabIdle": "#999999", "tabIdleHi": "#cccccc",
	"grid": "#999999", "menuHotTop": "#666666", "frameOff": "#999999",
	"grooveOff": "#666666", "grooveOffHi": "#cccccc", "groupText": "#666699",
	"accel": "#666699", "divider": "#999999",
	"grad1": "#dde8f3", "grad2": "#ffffff", "grad3": "#b8cfe5",
	"barTop": "#ffffff", "barBottom": "#dddddd", "barLine": "#999999",
	"toolTop": "#f2f2f2", "toolBottom": "#dcdcdc",
	"knob0": "#d8e6f6", "knob1": "#f8f8f8", "knob2": "#c2d6f0",
	"fill0": "#f8f8f8", "fill1": "#d1e1ef", "fill2": "#b7cee5", "fill3": "#a3b8cc",
}

// mtlOcean is what Ocean changes: its primaries and secondaries, #333333
// text, the light blue selected tab and the washes measured off Java 5
// screenshots.
var mtlOcean = map[string]string{
	"primary1": "#6382bf", "primary2": "#a3b8cc", "primary3": "#b8cfe5",
	"secondary1": "#7a8a99", "secondary2": "#b8cfe5", "secondary3": "#eeeeee",
	"black":       "#333333",
	"tabSelected": "#c8ddf2", "tabIdle": "#eeeeee", "tabIdleHi": "#eeeeee",
	"grid": "#7a8a99", "menuHotTop": "#6382bf", "frameOff": "#7a8a99",
	"grooveOff": "#333333", "grooveOffHi": "#b8cfe5", "groupText": "#333333",
	"accel": "#6382bf", "divider": "#7a8a99", "barLine": "#cccccc",
}

func mtlBuild(l *Classic) *mtl {
	col := func(k string) paintengine2d.Color { return l.X(k, Hex(mtlSteel[k])) }
	c := &mtl{
		ocean: l.P("ocean", 0) != 0,
		bold:  l.P("bold", 1) != 0,
		u:     mtlPx(l),
		p1:    col("primary1"), p2: col("primary2"), p3: col("primary3"),
		s1: col("secondary1"), s2: col("secondary2"), s3: col("secondary3"),
		black: col("black"), white: col("white"),
	}
	p := l.palette
	c.text = c.black
	c.dis = col("disabled")
	c.field = p.Field
	if colorUnset(c.field) {
		c.field = c.white
	}
	c.fieldText = l.fieldText()
	c.sel = c.p3
	c.selText = ReadableOn(c.sel, 4.5, c.text, c.white)
	c.focus = c.p2
	c.itemFocus = c.p2
	if c.ocean {
		c.itemFocus = c.p1
	}
	c.tabFocus = c.p1
	c.pressed = c.s2
	c.tabSel, c.tabIdle, c.tabIdleHi = col("tabSelected"), col("tabIdle"), col("tabIdleHi")
	c.tabEdgeSel, c.tabEdgeIdle = c.s1, c.s1
	if c.ocean {
		c.tabEdgeSel = c.p1
	}
	c.tabBand = c.tabSel
	// Steel's unselected tabs are the dimmed grey themselves: a disabled
	// label there goes darker instead.
	c.tabDis = c.dis
	if ContrastRatio(c.tabDis, c.tabIdle) < 1.6 {
		c.tabDis = Mix(c.tabIdle, c.text, 0.45)
	}
	c.disFill = Mix(c.s3, c.dis, 0.6)
	c.menuHot, c.menuHotTop = c.p2, col("menuHotTop")
	c.hotText = ReadableOn(c.menuHot, 4.5, c.text, c.black, c.white)
	c.accel = ReadableOn(c.s3, 3, col("accel"), c.text)
	c.accelHot = ReadableOn(c.menuHot, 3, col("accel"), c.hotText)
	c.info, c.infoBorder = c.p3, c.p1
	c.infoText = ReadableOn(c.info, 4.5, c.text, c.black, c.white)
	c.treeLine, c.grid = c.p3, col("grid")
	c.groupLine = c.s2
	// Titles are bold, so Steel's primary 1 on the canvas (3.2:1) keeps
	// the large-text threshold the guidelines accepted.
	c.groupText = ReadableOn(c.s3, 3, col("groupText"), c.text)
	c.frameOn, c.frameOff = c.p1, col("frameOff")
	c.grooveOn, c.grooveOnHi = c.black, c.p2
	c.grooveOff, c.grooveOffHi = col("grooveOff"), col("grooveOffHi")
	c.titleOn, c.titleOff = c.p3, c.s3
	c.barLine = col("barLine")
	c.trackShadow = c.s2
	if c.ocean {
		c.trackShadow = Mix(c.s1, c.s3, 0.55)
	}
	c.knobEdge, c.knobHi = c.black, Mix(c.p3, c.p2, 0.2)
	c.knobPress = Mix(c.p2, c.p1, 0.25)
	if c.ocean {
		c.knobEdge, c.knobHi = c.p1, c.p3
	}
	// Message icons: info on the theme's primaries, then the stock
	// warning yellow, error pink and question green.
	c.iconFill = [4]paintengine2d.Color{c.p3, Hex("#e6cf5d"), Hex("#ee9897"), Hex("#9ac899")}
	c.iconDark = [4]paintengine2d.Color{c.p1, Hex("#5f3334"), Hex("#990000"), Hex("#054b04")}
	c.iconHi = [4]paintengine2d.Color{Mix(c.p3, c.white, 0.8), Hex("#f6eda4"), Hex("#fcc2c1"), Hex("#c2f4c2")}

	peak, back := l.P("gradPeak", 0.3), l.P("gradBack", 0.6)
	g1, g2, g3 := col("grad1"), col("grad2"), col("grad3")
	c.btnGrad = []paintengine2d.GradientStop{Stop(0, g1), Stop(peak, g2), Stop(back, g1), Stop(1, g3)}
	c.knobGrad = []paintengine2d.GradientStop{Stop(0, col("knob0")), Stop(0.3, col("knob1")), Stop(0.5, col("knob1")), Stop(0.8, col("knob0")), Stop(1, col("knob2"))}
	c.fillGrad = []paintengine2d.GradientStop{Stop(0, col("fill0")), Stop(0.33, col("fill1")), Stop(0.66, col("fill2")), Stop(1, col("fill3"))}
	c.barGrad = []paintengine2d.GradientStop{Stop(0, col("barTop")), Stop(1, col("barBottom"))}
	c.toolGrad = []paintengine2d.GradientStop{Stop(0, col("toolTop")), Stop(1, col("toolBottom"))}

	c.thumbBumps = mtlBumpTile(c.p3, c.p1)
	c.gripBumps = mtlBumpTile(c.white, c.s1)
	c.gripHotBumps = mtlBumpTile(c.p3, c.p1)
	c.titleBumps = mtlBumpTile(c.white, c.p1)
	c.titleOffBumps = mtlBumpTile(c.white, c.s1)
	c.knobBumps = mtlBumpTile(c.p3, c.black)

	c.body = l.body
	c.bold2 = l.bold
	if c.bold2 == nil {
		c.bold2 = l.body
	}
	// The small font: menu accelerators and tool tip hints (10pt Dialog to
	// the 12pt control font).
	c.small = BakeFamily(l.UIFamily(), WeightRegular, l.metrics.FontSize*10/12, c.text)
	return c
}

// mtlBumpTile is one 4×4 period of the bumps: light dots at (0,0) and
// (2,2), each with its dark dot one pixel down and right.
func mtlBumpTile(light, dark paintengine2d.Color) *paintengine2d.Image {
	img := paintengine2d.NewImage(4, 4)
	img.SetColor(0, 0, light)
	img.SetColor(2, 2, light)
	img.SetColor(1, 1, dark)
	img.SetColor(3, 3, dark)
	return img
}

// ctl is the face a control of role is labelled in: bold unless the pack
// turns Metal's bold fonts off.
func (c *mtl) ctl() *Font {
	if c.bold {
		return c.bold2
	}
	return c.body
}

// ---- drawing helpers -----------------------------------------------------------

// mtlPx is one Metal pixel: a device pixel at 1x, two at 2x.
func mtlPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
}

// mtlSnap puts b on whole device pixels.
func mtlSnap(b paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.Rect{
		Min: paintengine2d.Pt(snap(b.Min.X), snap(b.Min.Y)),
		Max: paintengine2d.Pt(snap(b.Max.X), snap(b.Max.Y)),
	}
}

// mtlRects batches rects of one colour into a single path, filled in one
// operation.
type mtlRects struct {
	p *paintengine2d.Path
}

func (r *mtlRects) add(x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if r.p == nil {
		r.p = paintengine2d.NewPath()
	}
	r.p.AddRect(paintengine2d.XYWH(x, y, w, h))
}

func (r *mtlRects) fill(ctx *paintengine2d.Context, col paintengine2d.Color) {
	if r.p != nil && col.A > 0 {
		ctx.DrawPath(r.p, paintengine2d.Fill(col))
	}
}

// ring strokes a complete one-pixel ring just inside b.
func (c *mtl) ring(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	u := c.u
	if b.Dx() < 2*u || b.Dy() < 2*u {
		return
	}
	var r mtlRects
	r.add(b.Min.X, b.Min.Y, b.Dx(), u)
	r.add(b.Min.X, b.Max.Y-u, b.Dx(), u)
	r.add(b.Min.X, b.Min.Y+u, u, b.Dy()-2*u)
	r.add(b.Max.X-u, b.Min.Y+u, u, b.Dy()-2*u)
	r.fill(ctx, col)
}

// flush paints Metal's flush 3D border inside b: depth dark rings (two for
// a default button) with a white line inside their top and left edges
// (unless !inner — a pressed face) and outside their right and bottom
// edges. The innermost dark ring stays open at the top-right and
// bottom-left corners where the white lines would meet it.
func (c *mtl) flush(ctx *paintengine2d.Context, b paintengine2d.Rect, depth int, dark, light paintengine2d.Color, inner bool) {
	u := c.u
	x0, y0, x1, y1 := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
	if x1-x0 < float32(2*depth+3)*u || y1-y0 < float32(2*depth+3)*u {
		c.ring(ctx, b, dark)
		return
	}
	var d, w mtlRects
	for k := 0; k < depth; k++ {
		kf := float32(k) * u
		end := float32(1+k) * u // the ring's far edges sit end from x1 / y1
		o := float32(0)
		if k == depth-1 {
			o = 2 * u
		}
		d.add(x0+kf, y0+kf, x1-end-(x0+kf), u)
		d.add(x0+kf, y0+kf, u, y1-end-(y0+kf))
		d.add(x1-end-u, y0+kf+o, u, y1-end-(y0+kf+o))
		d.add(x0+kf+o, y1-end-u, x1-end-(x0+kf+o), u)
	}
	df := float32(depth) * u
	if inner {
		w.add(x0+df, y0+df, x1-df-u-(x0+df), u)
		w.add(x0+df, y0+df, u, y1-df-u-(y0+df))
	}
	w.add(x1-u, y0+df, u, y1-(y0+df))
	w.add(x0+df, y1-u, x1-(x0+df), u)
	w.fill(ctx, light)
	d.fill(ctx, dark)
}

// dimRing is a disabled control's border: one ring in the dimmed colour
// (Steel leaves the white highlight off and sits the ring where the flush
// dark ring would be; Ocean rings the whole face).
func (c *mtl) dimRing(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	if c.ocean {
		c.ring(ctx, b, c.dis)
		return
	}
	c.ring(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-c.u, b.Dy()-c.u), c.dis)
}

// wash fills b with Ocean's glossy wash, top to bottom (across = false) or
// left to right.
func (c *mtl) wash(ctx *paintengine2d.Context, b paintengine2d.Rect, across bool) {
	if across {
		ctx.DrawRect(b, HGradient(b, c.btnGrad...))
		return
	}
	ctx.DrawRect(b, VGradient(b, c.btnGrad...))
}

// bumps lays the drag texture over r, anchored at r's top-left so the dots
// travel with the part they sit on.
func (c *mtl) bumps(ctx *paintengine2d.Context, r paintengine2d.Rect, tile *paintengine2d.Image) {
	r = mtlSnap(r)
	if r.Dx() < 2*c.u || r.Dy() < 2*c.u {
		return
	}
	ctx.DrawRect(r, paintengine2d.Paint{
		Shader: paintengine2d.ImagePattern{Image: tile, Origin: r.Min, Scale: c.u},
		Style:  paintengine2d.StyleFill,
	})
}

// focusRect strokes the one-pixel focus rectangle inside r.
func (c *mtl) focusRect(ctx *paintengine2d.Context, r paintengine2d.Rect, col paintengine2d.Color) {
	c.ring(ctx, mtlSnap(r), col)
}

// triangle fills Metal's stepped arrow: n rows of 2, 4, … 2n Metal pixels
// pointing dir, centred on ctr.
func (c *mtl) triangle(ctx *paintengine2d.Context, ctr paintengine2d.Point, n int, dir Direction, col paintengine2d.Color) {
	if n < 1 || col.A <= 0 {
		return
	}
	u := c.u
	nf := float32(n)
	var r mtlRects
	switch dir {
	case DirUp, DirDown:
		x0 := snap(ctr.X - nf*u)
		y0 := snap(ctr.Y - nf*u*0.5)
		for i := 0; i < n; i++ {
			w := float32(i + 1)
			if dir == DirDown {
				w = float32(n - i)
			}
			r.add(x0+(nf-w)*u, y0+float32(i)*u, 2*w*u, u)
		}
	default:
		x0 := snap(ctr.X - nf*u*0.5)
		y0 := snap(ctr.Y - nf*u)
		for i := 0; i < n; i++ {
			h := float32(i + 1)
			if dir == DirRight {
				h = float32(n - i)
			}
			r.add(x0+float32(i)*u, y0+(nf-h)*u, u, 2*h*u)
		}
	}
	r.fill(ctx, col)
}

// arrowRows is the arrow size for a glyph box of side s: Metal's 8×4
// scroll arrow in a 16px button.
func (c *mtl) arrowRows(s float32) int {
	n := int(s/(4*c.u) + 0.25)
	if n < 2 {
		n = 2
	}
	if n > 6 {
		n = 6
	}
	return n
}

// tick strokes the check mark into box (the check box's interior).
func (c *mtl) tick(ctx *paintengine2d.Context, box paintengine2d.Rect, col paintengine2d.Color) {
	s := box.Dx()
	if box.Dy() < s {
		s = box.Dy()
	}
	if s < 4*c.u {
		return
	}
	x, y := box.Min.X+(box.Dx()-s)*0.5, box.Min.Y+(box.Dy()-s)*0.5
	// A short, steep left arm and a long right one, two pixels thick.
	p := paintengine2d.NewPath()
	p.MoveTo(x+s*0.16, y+s*0.4)
	p.LineTo(x+s*0.34, y+s*0.8)
	p.LineTo(x+s*0.86, y+s*0.14)
	// Two pixels in Metal's 13px box, in proportion at other scales.
	w := s * 0.25
	if w < 1.5 {
		w = 1.5
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// ---- faces -----------------------------------------------------------------------

// face paints a push button, tool button or arrow button face and returns
// its label colour. tool buttons are flat until the pointer is over them
// (Metal's rollover tool bars).
func (c *mtl) face(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, tool bool) paintengine2d.Color {
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return c.text
	}
	if st.Disabled() {
		if !tool || st.Checked() {
			ctx.DrawRect(b, paintengine2d.Fill(c.s3))
			c.dimRing(ctx, b)
		}
		return c.dis
	}
	down := st.Pressed() || (st.Checked() && (tool || st.Toggle()))
	if tool && !down && !st.Hovered() {
		return c.text
	}
	depth := 1
	if st.Primary() && !tool {
		depth = 2
	}
	if c.ocean {
		if down {
			ctx.DrawRect(b, paintengine2d.Fill(c.pressed))
		} else {
			c.wash(ctx, b, false)
		}
		edge := c.s1
		if st.Hovered() && !down {
			edge = c.p1
		}
		c.ring(ctx, b, edge)
		if depth == 2 {
			c.ring(ctx, b.Inset(u), edge)
		}
		return c.text
	}
	fill := c.s3
	if down {
		fill = c.pressed
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	c.flush(ctx, b, depth, c.s1, c.white, !down)
	return c.text
}

// field paints a text field or list well: the field colour inside the
// flush border (the canvas and a dimmed ring when disabled).
func (c *mtl) fieldFace(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	b = mtlSnap(b)
	if st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
		c.dimRing(ctx, b)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.field))
	c.flush(ctx, b, 1, c.s1, c.white, true)
}

func (e metalEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := mtlColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	switch role {
	case RoleButton:
		return c.face(ctx, b, st, false)
	case RoleTool:
		return c.face(ctx, b, st, true)
	case RoleField, RoleCombo:
		c.fieldFace(ctx, b, st)
		if !st.Disabled() {
			fg = c.fieldText
		}
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, false)
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(mtlSnap(b), paintengine2d.Fill(c.sel))
			return c.selText
		}
		return c.fieldText
	case RoleMenu:
		e.MenuHighlight(l, ctx, b, false)
		return c.hotText
	case RoleThumb:
		c.thumb(ctx, b, b.Dy() >= b.Dx(), st)
	case RoleTrack:
		c.track(ctx, b, b.Dy() >= b.Dx())
	case RoleTab:
	case RoleSplitter, RoleBar, RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	}
	return fg
}

func (metalEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := mtlColors(l)
	box = mtlSnap(box)
	u := c.u
	if box.Dx() < 6*u || box.Dy() < 6*u {
		return
	}
	mark := c.text
	switch {
	case st.Disabled():
		mark = c.dis
		ctx.DrawRect(box, paintengine2d.Fill(c.s3))
		c.dimRing(ctx, box)
	case c.ocean:
		if st.Pressed() {
			ctx.DrawRect(box, paintengine2d.Fill(c.pressed))
		} else {
			c.wash(ctx, box, false)
		}
		c.ring(ctx, box, c.s1)
	default:
		fill := c.s3
		if st.Pressed() {
			fill = c.pressed
		}
		ctx.DrawRect(box, paintengine2d.Fill(fill))
		c.flush(ctx, box, 1, c.s1, c.white, !st.Pressed())
	}
	if checked || st.Checked() {
		in := box.Inset(2 * u)
		if !c.ocean {
			in = paintengine2d.XYWH(box.Min.X+2*u, box.Min.Y+2*u, box.Dx()-5*u, box.Dy()-5*u)
		}
		c.tick(ctx, in, mark)
	}
}

func (metalEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := mtlColors(l)
	box = mtlSnap(box)
	u := c.u
	d := box.Dx()
	if box.Dy() < d {
		d = box.Dy()
	}
	if d < 6*u {
		return
	}
	// Steel's radio is 12px plus the white highlight below and right.
	ring := d
	if !c.ocean && !st.Disabled() {
		ring = d - u
	}
	cx, cy := box.Min.X+ring*0.5, box.Min.Y+ring*0.5
	ctr := paintengine2d.Pt(cx, cy)
	r := ring * 0.5
	dot := c.text
	switch {
	case st.Disabled():
		dot = c.dis
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.s3))
		ctx.DrawCircle(ctr, r-u*0.5, paintengine2d.StrokePaint(c.dis, u))
	case c.ocean:
		disc := paintengine2d.XYWH(cx-r, cy-r, ring, ring)
		if st.Pressed() {
			ctx.DrawCircle(ctr, r, paintengine2d.Fill(c.pressed))
		} else {
			ctx.DrawCircle(ctr, r, VGradient(disc, c.btnGrad...))
		}
		ctx.DrawCircle(ctr, r-u*0.5, paintengine2d.StrokePaint(c.s1, u))
	default:
		fill := c.s3
		if st.Pressed() {
			fill = c.pressed
		}
		// The flush highlight on a circle: white just outside the lower
		// right of the ring, and just inside its upper left unless pressed.
		ctx.DrawArc(paintengine2d.Pt(cx+u*0.5, cy+u*0.5), r, r, -math.Pi/4, math.Pi, paintengine2d.StrokePaint(c.white, u))
		ctx.DrawCircle(ctr, r, paintengine2d.Fill(fill))
		if !st.Pressed() {
			ctx.DrawArc(ctr, r-1.5*u, r-1.5*u, 3*math.Pi/4, math.Pi, paintengine2d.StrokePaint(c.white, u))
		}
		ctx.DrawCircle(ctr, r-u*0.5, paintengine2d.StrokePaint(c.s1, u))
	}
	if selected || st.Checked() {
		ctx.DrawCircle(ctr, r*0.5, paintengine2d.Fill(dot))
	}
}

// Arrow is Metal's small stepped triangle.
func (metalEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	c := mtlColors(l)
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	c.triangle(ctx, b.Center(), c.arrowRows(s), dir, col)
}

// Expander is Metal's tree "key": a small round handle with a stem that
// points right when collapsed and down when expanded — a shaded ball in a
// black ring on Steel, a hollow blue ring on Ocean.
func (metalEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := mtlColors(l)
	u := c.u
	d := snap(8 * u)
	ctr := b.Center()
	cx, cy := snap(ctr.X-d*0.5)+d*0.5, snap(ctr.Y-d*0.5)+d*0.5
	r := d * 0.5
	stem, edge := c.black, c.black
	if c.ocean {
		stem, edge = c.p1, c.p1
	}
	var s mtlRects
	if expanded {
		s.add(cx-u, cy+r-u, 2*u, 5*u)
	} else {
		s.add(cx+r-u, cy-u, 5*u, 2*u)
	}
	s.fill(ctx, stem)
	pc := paintengine2d.Pt(cx, cy)
	if c.ocean {
		ctx.DrawCircle(pc, r, paintengine2d.Fill(c.field))
		ctx.DrawCircle(pc, r-u*0.8, paintengine2d.StrokePaint(edge, 1.6*u))
		return
	}
	disc := paintengine2d.XYWH(cx-r, cy-r, d, d)
	ctx.DrawCircle(pc, r, DGradient(disc, Stop(0, c.p3), Stop(0.5, c.p2), Stop(1, c.p1)))
	ctx.DrawCircle(pc, r-u*0.5, paintengine2d.StrokePaint(edge, u))
	ctx.DrawRect(paintengine2d.XYWH(cx-u, cy-u, 2*u, 2*u), paintengine2d.Fill(edge))
}

// MenuHighlight is the armed menu item: primary 2 with a dark line along
// its top and a white one along its bottom; an open menu bar title is
// outlined on three sides instead, so it joins its popup.
func (metalEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.menuHot))
	var d mtlRects
	if attachBottom {
		d.add(b.Min.X, b.Min.Y, b.Dx(), u)
		d.add(b.Min.X, b.Min.Y+u, u, b.Dy()-u)
		d.add(b.Max.X-u, b.Min.Y+u, u, b.Dy()-u)
		d.fill(ctx, c.menuHotTop)
		return
	}
	d.add(b.Min.X, b.Min.Y, b.Dx(), u)
	d.fill(ctx, c.menuHotTop)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.white))
}

func (metalEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := mtlColors(l)
	if hot {
		return c.hotText
	}
	return c.text
}

// Text fields show focus with the caret alone.
func (metalEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the focus rectangle in primary 2, just inside b.
func (metalEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	c.focusRect(ctx, b.Inset(c.u), c.focus)
}

// ---- scroll bars ------------------------------------------------------------------

func (metalEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsEnds, ArrowLen: 16, MinThumb: 16}
}

// track paints a scroll bar's channel: the canvas between dark edges with a
// shadow line inside its leading edges and a white line outside the far
// one.
func (c *mtl) track(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	var d, s, w mtlRects
	if vertical {
		d.add(b.Min.X, b.Min.Y, u, b.Dy())
		d.add(b.Max.X-2*u, b.Min.Y, u, b.Dy())
		s.add(b.Min.X+u, b.Min.Y, u, b.Dy())
		s.add(b.Min.X+2*u, b.Min.Y, b.Dx()-4*u, u)
		w.add(b.Max.X-u, b.Min.Y, u, b.Dy())
	} else {
		d.add(b.Min.X, b.Min.Y, b.Dx(), u)
		d.add(b.Min.X, b.Max.Y-2*u, b.Dx(), u)
		s.add(b.Min.X, b.Min.Y+u, b.Dx(), u)
		s.add(b.Min.X, b.Min.Y+2*u, u, b.Dy()-4*u)
		w.add(b.Min.X, b.Max.Y-u, b.Dx(), u)
	}
	w.fill(ctx, c.white)
	s.fill(ctx, c.trackShadow)
	d.fill(ctx, c.s1)
}

// thumb paints the scroll box. Steel: primary 2 edged in primary 1 on
// three sides (the channel's dark edge closes the fourth) with a primary 3
// highlight and the bumps; Ocean: a primary 1 ring round a primary 3
// highlight and the blue wash, with a three-line grip.
func (c *mtl) thumb(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	if c.ocean {
		t := b
		if vertical {
			t.Max.X -= u
		} else {
			t.Max.Y -= u
		}
		c.wash(ctx, t, vertical)
		c.ring(ctx, t.Inset(u), c.p3)
		c.ring(ctx, t, c.p1)
		// The grip: three short lines, each over a white one.
		along, across := t.Dy(), t.Dx()
		if !vertical {
			along, across = t.Dx(), t.Dy()
		}
		gl := snap(across - 8*u)
		if along < 12*u || gl < 3*u {
			return
		}
		ctr := t.Center()
		var dk, lt mtlRects
		for k := 0; k < 3; k++ {
			o := float32(2*k-3) * u
			if vertical {
				x0, y := snap(ctr.X-gl*0.5), snap(ctr.Y)+o
				dk.add(x0, y, gl, u)
				lt.add(x0, y+u, gl, u)
			} else {
				x, y0 := snap(ctr.X)+o, snap(ctr.Y-gl*0.5)
				dk.add(x, y0, u, gl)
				lt.add(x+u, y0, u, gl)
			}
		}
		lt.fill(ctx, c.white)
		dk.fill(ctx, c.p1)
		return
	}
	t := b
	if vertical {
		t.Max.X -= 2 * u
	} else {
		t.Max.Y -= 2 * u
	}
	ctx.DrawRect(t, paintengine2d.Fill(c.p2))
	var e, h mtlRects
	if vertical {
		e.add(t.Min.X, t.Min.Y, t.Dx(), u)
		e.add(t.Min.X, t.Max.Y-u, t.Dx(), u)
		e.add(t.Min.X, t.Min.Y+u, u, t.Dy()-2*u)
		h.add(t.Min.X+u, t.Min.Y+u, t.Dx()-u, u)
		h.add(t.Min.X+u, t.Min.Y+2*u, u, t.Dy()-3*u)
	} else {
		e.add(t.Min.X, t.Min.Y, u, t.Dy())
		e.add(t.Max.X-u, t.Min.Y, u, t.Dy())
		e.add(t.Min.X+u, t.Min.Y, t.Dx()-2*u, u)
		h.add(t.Min.X+u, t.Min.Y+u, u, t.Dy()-u)
		h.add(t.Min.X+2*u, t.Min.Y+u, t.Dx()-3*u, u)
	}
	h.fill(ctx, c.p3)
	e.fill(ctx, c.p1)
	in := paintengine2d.XYWH(t.Min.X+3*u, t.Min.Y+3*u, t.Dx()-5*u, t.Dy()-6*u)
	if !vertical {
		in = paintengine2d.XYWH(t.Min.X+3*u, t.Min.Y+3*u, t.Dx()-6*u, t.Dy()-5*u)
	}
	c.bumps(ctx, in, c.thumbBumps)
}

func (e metalEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := mtlColors(l)
	if p.Bar.Empty() {
		return
	}
	c.track(ctx, p.Bar, vertical)
	arrow := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		b = mtlSnap(b)
		down := st.Pressed == part && !st.Disabled
		fill := c.s3
		if down {
			fill = c.pressed
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		c.flush(ctx, b, 1, c.s1, c.white, !down)
		col := c.text
		if st.Disabled {
			col = c.dis
		}
		s := b.Dx()
		if b.Dy() < s {
			s = b.Dy()
		}
		g := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-c.u, b.Dy()-c.u)
		c.triangle(ctx, g.Center(), c.arrowRows(s), dir, col)
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
	ts := StateNone
	if st.Pressed == ScrollThumbPart {
		ts |= StatePressed
	}
	c.thumb(ctx, p.Thumb, vertical, ts)
}

// DrawScrollBar is the thumb-in-channel fallback for callers without
// scroll bar geometry.
func (e metalEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	c := mtlColors(l)
	vertical := track.Dy() >= track.Dx()
	c.track(ctx, track, vertical)
	if !thumb.Empty() {
		c.thumb(ctx, thumb, vertical, st)
	}
}

// ---- frames ------------------------------------------------------------------------

func (metalEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = mtlColors(l).ctl().Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is a titled border: a one-pixel secondary 2 line (Steel's
// grey, Ocean's pale blue) with the title in the bold control font; a
// raised box is an etched card instead.
func (metalEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	f := c.ctl()
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if raised {
		ctx.DrawRect(frame, paintengine2d.Fill(c.s3))
		c.flush(ctx, frame, 1, c.s1, c.white, true)
	} else {
		c.ring(ctx, frame, c.groupLine)
	}
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(8)
	tw := f.Advance(title) + l.S(6)
	if tw > b.Dx()-l.S(16) {
		tw = b.Dx() - l.S(16)
	}
	if tw <= 0 {
		return
	}
	// Open the line behind the title.
	ctx.DrawRect(paintengine2d.XYWH(snap(tx), top, snap(tw), 2*u), paintengine2d.Fill(c.s3))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.groupText, AlignStart, 0)
}

// mtlCaptionH is the title bar of an in-app frame: the bold title plus
// Metal's padding, never shorter than its 16px buttons need.
func mtlCaptionH(l *Classic) float32 {
	c := mtlColors(l)
	h := c.ctl().Height() + l.S(6)
	if m := l.S(22); h < m {
		h = m
	}
	return snap(h)
}

// mtlFrameW is the frame border: five lines.
func mtlFrameW(l *Classic) float32 { return 5 * mtlColors(l).u }

func (metalEngine) WindowFrameInsets(l *Classic) Insets {
	fw := mtlFrameW(l)
	return Insets{Top: fw + mtlCaptionH(l) + mtlColors(l).u, Right: fw, Bottom: fw, Left: fw}
}

// WindowCloseRect is the close button at the right end of the title bar.
func (metalEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = mtlSnap(b)
	fw := mtlFrameW(l)
	h := mtlCaptionH(l)
	if b.Dx() < 2*fw+h*2 || b.Dy() < 2*fw+h {
		return paintengine2d.Rect{}
	}
	side := snap(h - l.S(6))
	return paintengine2d.XYWH(b.Max.X-fw-l.S(3)-side, b.Min.Y+fw+snap((h-side)*0.5), side, side)
}

// No drop shadows: Metal's menus, tool tips and frames float flat.
func (metalEngine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (metalEngine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// Swing's option panes put the default button first: "OK  Cancel".
func (metalEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintDialogPrimaryFirst {
		return 1
	}
	return 0
}

// DrawWindowFrame is a Metal internal frame: a five-line border in primary
// 1 (secondary when inactive) grooved along its sides but solid for 16
// pixels at each corner, a title bar in primary 3 (Ocean's blue wash) with
// the bold title, the bumps and a flush close button.
func (e metalEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	fw := mtlFrameW(l)
	capH := mtlCaptionH(l)
	if b.Dx() < 2*fw+capH || b.Dy() < 2*fw+capH {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
		return
	}
	frame, groove, grooveHi, bar, tile := c.frameOn, c.grooveOn, c.grooveOnHi, c.titleOn, c.titleBumps
	if !st.Active {
		frame, groove, grooveHi, bar, tile = c.frameOff, c.grooveOff, c.grooveOffHi, c.titleOff, c.titleOffBumps
	}
	ctx.DrawRect(b, paintengine2d.Fill(frame))
	// The groove: a dark line two pixels in and a light one inside it,
	// broken for the solid corners.
	corner := snap(l.S(16))
	var gd, gl mtlRects
	x0, y0, x1, y1 := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y
	if b.Dx() > 2*corner && b.Dy() > 2*corner {
		gd.add(x0+corner, y0+2*u, b.Dx()-2*corner, u)
		gl.add(x0+corner, y0+3*u, b.Dx()-2*corner, u)
		gd.add(x0+corner, y1-3*u, b.Dx()-2*corner, u)
		gl.add(x0+corner, y1-2*u, b.Dx()-2*corner, u)
		gd.add(x0+2*u, y0+corner, u, b.Dy()-2*corner)
		gl.add(x0+3*u, y0+corner, u, b.Dy()-2*corner)
		gd.add(x1-3*u, y0+corner, u, b.Dy()-2*corner)
		gl.add(x1-2*u, y0+corner, u, b.Dy()-2*corner)
	}
	gl.fill(ctx, grooveHi)
	gd.fill(ctx, groove)
	tb := paintengine2d.XYWH(x0+fw, y0+fw, b.Dx()-2*fw, capH)
	if c.ocean && st.Active {
		c.wash(ctx, tb, false)
	} else {
		ctx.DrawRect(tb, paintengine2d.Fill(bar))
	}
	// The line under the title bar, then the frame's canvas.
	ctx.DrawRect(paintengine2d.XYWH(tb.Min.X, tb.Max.Y, tb.Dx(), u), paintengine2d.Fill(frame))
	ctx.DrawRect(paintengine2d.XYWH(tb.Min.X, tb.Max.Y+u, tb.Dx(), y1-fw-(tb.Max.Y+u)), paintengine2d.Fill(c.s3))
	right := tb.Max.X - l.S(3)
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			c.closeButton(ctx, cb, st)
			right = cb.Min.X - l.S(4)
		}
	}
	f := c.bold2
	tx := tb.Min.X + l.S(6)
	tw := float32(0)
	if title != "" {
		tw = f.Advance(title)
		if tw > right-tx {
			tw = right - tx
		}
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx, tb.Min.Y, right-tx, tb.Dy()), c.text, AlignStart, 0)
	}
	// The bumps fill the bar between the title and the buttons.
	bx := snap(tx + tw + l.S(6))
	if right-bx > 4*u {
		c.bumps(ctx, paintengine2d.XYWH(bx, tb.Min.Y+3*u, right-bx, tb.Dy()-6*u), tile)
	}
}

// closeButton is a title bar button: a flush square in the bar's colour
// with a bold ×.
func (c *mtl) closeButton(ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	u := c.u
	fill := c.titleOn
	if !st.Active {
		fill = c.titleOff
	}
	if st.ClosePress {
		fill = c.pressed
	}
	if c.ocean && st.Active && !st.ClosePress {
		c.wash(ctx, cb, false)
	} else {
		ctx.DrawRect(cb, paintengine2d.Fill(fill))
	}
	edge := c.black
	if st.CloseHot && !st.ClosePress {
		edge = c.p1
	}
	c.flush(ctx, cb, 1, edge, c.white, !st.ClosePress)
	g := paintengine2d.XYWH(cb.Min.X, cb.Min.Y, cb.Dx()-u, cb.Dy()-u).Inset(snap(cb.Dx() * 0.28))
	DrawCross(ctx, g, c.black, 2*u)
}

// TabOutset: Metal's selected tab keeps its slot.
func (metalEngine) TabOutset(l *Classic) Insets { return Insets{} }

// TabOverlap: neighbouring tabs share one border line.
func (metalEngine) TabOverlap(l *Classic) float32 { return mtlColors(l).u }

// SpinBoxStyle: the stepper sits inside the field's border, stacked.
func (metalEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{Inside: true} }

// ToolBarInsets keeps the bumps grip clear of the first button.
func (metalEngine) ToolBarInsets(l *Classic) Insets { return Insets{Left: l.S(14), Right: l.S(4)} }

// ControlFont: buttons, check boxes, radios, combo boxes, tabs and menus
// are labelled in the bold control font; text, lists and tables read in the
// plain user font.
func (metalEngine) ControlFont(l *Classic, role Role) *Font {
	c := mtlColors(l)
	switch role {
	case RoleButton, RoleTool, RoleCheck, RoleCombo, RoleTab, RoleMenu:
		return c.ctl()
	}
	return l.body
}

// ItemFocus is the focus rectangle round the current row (primary 2 on
// Steel, primary 1 on Ocean).
func (metalEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := mtlColors(l)
	c.focusRect(ctx, b, c.itemFocus)
}

// ViewFrameInsets: the flush border round lists, trees and tables — one
// dark line at the top and left, the dark line and its white highlight at
// the right and bottom.
func (metalEngine) ViewFrameInsets(l *Classic) Insets {
	u := mtlColors(l).u
	return Insets{Top: u, Left: u, Right: 2 * u, Bottom: 2 * u}
}

func (metalEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	mtlColors(l).fieldFace(ctx, b, st&^(StateHovered|StatePressed))
}

// DrawTabPane is the page under the tabs: the canvas with a white edge at
// the left and dark ones at the right and bottom (the white top edge is
// the tab strip's); Ocean frames it with the selected tab's pale blue.
func (metalEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 6*u || b.Dy() < 6*u {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
		return
	}
	var w, d, lt mtlRects
	if c.ocean {
		ctx.DrawRect(b, paintengine2d.Fill(c.tabBand))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2*u, b.Min.Y+2*u, b.Dx()-4*u, b.Dy()-4*u), paintengine2d.Fill(c.s3))
		w.add(b.Min.X+u, b.Min.Y, b.Dx()-2*u, u)
		lt.add(b.Min.X, b.Min.Y, u, b.Dy()-u)
		d.add(b.Max.X-u, b.Min.Y, u, b.Dy())
		d.add(b.Min.X, b.Max.Y-u, b.Dx()-u, u)
		w.fill(ctx, c.white)
		lt.fill(ctx, c.p1)
		d.fill(ctx, c.s1)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	w.add(b.Min.X, b.Min.Y, u, b.Dy()-u)
	d.add(b.Max.X-u, b.Min.Y, u, b.Dy())
	d.add(b.Min.X, b.Max.Y-u, b.Dx()-u, u)
	w.fill(ctx, c.white)
	d.fill(ctx, c.s1)
}

// ---- controls ------------------------------------------------------------------

// labelFocus rings a label drawn centred in lb: the text's width plus
// three pixels a side, a line's height plus one, kept inside clip.
func (c *mtl) labelFocus(ctx *paintengine2d.Context, f *Font, label string, lb, clip paintengine2d.Rect) {
	if label == "" {
		c.focusRect(ctx, clip.Inset(3*c.u), c.focus)
		return
	}
	tw := f.Advance(label)
	if lim := lb.Dx() - 2*c.u; tw > lim {
		tw = lim
	}
	w := tw + 6*c.u
	h := f.Height() + 2*c.u
	r := paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h)
	c.focusRect(ctx, r.Intersect(clip), c.focus)
}

func (e metalEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := mtlColors(l)
	fg := c.face(ctx, b, st, false)
	f := c.ctl()
	lb := b
	if !c.ocean {
		lb = paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-c.u, b.Dy()-c.u)
	}
	l.drawFittedText(ctx, f, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && !st.Disabled() {
		in := mtlSnap(b).Inset(3 * c.u)
		c.labelFocus(ctx, f, label, lb, in)
	}
}

// toggleLabel draws a check box, radio or switch caption and its focus
// rectangle.
func (c *mtl) toggleLabel(l *Classic, ctx *paintengine2d.Context, b, lb paintengine2d.Rect, st ControlState, label string) {
	f := c.ctl()
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, f, label, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		tw := f.Advance(label)
		if tw > lb.Dx()-3*c.u {
			tw = lb.Dx() - 3*c.u
		}
		h := f.Height() + 2*c.u
		r := paintengine2d.XYWH(lb.Min.X-2*c.u, lb.Min.Y+(lb.Dy()-h)*0.5, tw+4*c.u, h)
		c.focusRect(ctx, r.Intersect(b), c.focus)
	}
}

func (e metalEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := mtlColors(l)
	side := l.metrics.Checkbox
	box := mtlSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(ctx, box.Inset(-c.u).Intersect(b), c.focus)
		}
		return
	}
	gap := l.S(6)
	c.toggleLabel(l, ctx, b, paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy()), st, label)
}

func (e metalEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := mtlColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	box := mtlSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(ctx, box.Inset(-c.u).Intersect(b), c.focus)
		}
		return
	}
	gap := l.S(6)
	c.toggleLabel(l, ctx, b, paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy()), st, label)
}

// DrawSwitch has no Metal original: a flush field slot with a knob, filled
// with primary 2 when on.
func (e metalEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := mtlColors(l)
	u := c.u
	tw, th := l.metrics.SwitchW, l.metrics.SwitchH
	if th > b.Dy() {
		th = b.Dy()
	}
	if tw > b.Dx() {
		tw = b.Dx()
	}
	track := mtlSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*u || track.Dy() < 6*u {
		return
	}
	switch {
	case st.Disabled():
		ctx.DrawRect(track, paintengine2d.Fill(c.s3))
		c.dimRing(ctx, track)
	default:
		fill := c.field
		if on {
			fill = c.p2
		}
		ctx.DrawRect(track, paintengine2d.Fill(fill))
		c.flush(ctx, track, 1, c.s1, c.white, !on)
	}
	kw := snap(track.Dy())
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	c.face(ctx, paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), st&^(StateFocused|StatePrimary|StateChecked|StateToggle), false)
	if !c.ocean && !st.Disabled() {
		c.bumps(ctx, paintengine2d.XYWH(kx+3*u, track.Min.Y+3*u, kw-6*u, track.Dy()-6*u), c.gripBumps)
	}
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(ctx, track.Inset(-c.u).Intersect(b), c.focus)
		}
		return
	}
	gap := l.S(8)
	c.toggleLabel(l, ctx, b, paintengine2d.XYWH(track.Max.X+gap, b.Min.Y, b.Max.X-track.Max.X-gap, b.Dy()), st, label)
}

// DrawComboBox is a non-editable combo. Steel draws it as one flush button
// with the arrow at its right end; Ocean as a canvas field with a pale
// inner line (primary 2 when focused) and a separate washed arrow button.
func (e metalEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 8*u || b.Dy() < 8*u {
		return
	}
	f := c.ctl()
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	down := (open || st.Pressed()) && !st.Disabled()
	aw := snap(b.Dy() - 2*u)
	if m := l.S(18); aw < m {
		aw = m
	}
	if c.ocean {
		ab := paintengine2d.XYWH(b.Max.X-aw, b.Min.Y, aw, b.Dy())
		fb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-aw, b.Dy())
		switch {
		case st.Disabled():
			ctx.DrawRect(b, paintengine2d.Fill(c.s3))
		case st.Focused() && !open && !st.Editable():
			ctx.DrawRect(fb, paintengine2d.Fill(c.p2))
			c.ring(ctx, paintengine2d.XYWH(fb.Min.X+u, fb.Min.Y+u, fb.Dx()-u, fb.Dy()-2*u), c.p3)
		default:
			ctx.DrawRect(fb, paintengine2d.Fill(c.s3))
			c.ring(ctx, paintengine2d.XYWH(fb.Min.X+u, fb.Min.Y+u, fb.Dx()-u, fb.Dy()-2*u), c.p3)
		}
		if !st.Disabled() {
			if down {
				ctx.DrawRect(ab, paintengine2d.Fill(c.pressed))
			} else {
				c.wash(ctx, ab, false)
			}
		}
		edge := c.s1
		if st.Disabled() {
			edge = c.dis
		}
		c.ring(ctx, b, edge)
		ctx.DrawRect(paintengine2d.XYWH(ab.Min.X, b.Min.Y, u, b.Dy()), paintengine2d.Fill(edge))
		c.triangle(ctx, paintengine2d.Pt(ab.Center().X+u*0.5, ab.Center().Y), 4, DirDown, fg)
		tb := paintengine2d.XYWH(fb.Min.X+l.metrics.FieldPad, fb.Min.Y, fb.Dx()-l.metrics.FieldPad-2*u, fb.Dy())
		l.drawFittedText(ctx, f, text, tb, fg, AlignStart, 0)
		return
	}
	c.face(ctx, b, st&^StateFocused|mtlIf(down, StatePressed), false)
	in := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-u, b.Dy()-u)
	ab := paintengine2d.XYWH(in.Max.X-aw, in.Min.Y, aw, in.Dy())
	c.triangle(ctx, ab.Center(), 4, DirDown, fg)
	tb := paintengine2d.XYWH(in.Min.X+l.metrics.FieldPad+u, in.Min.Y, ab.Min.X-in.Min.X-l.metrics.FieldPad-u, in.Dy())
	l.drawFittedText(ctx, f, text, tb, fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.focusRect(ctx, paintengine2d.XYWH(in.Min.X+3*u, in.Min.Y+3*u, ab.Min.X-in.Min.X-3*u, in.Dy()-6*u), c.focus)
	}
}

// mtlIf returns s when cond holds.
func mtlIf(cond bool, s ControlState) ControlState {
	if cond {
		return s
	}
	return 0
}

// DrawSpinner is the stepper: two small flush arrow buttons stacked, inside
// the field's border (a dark line against the text) or in a border of
// their own.
func (e metalEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	in := b
	if st.Frameless() {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, u, b.Dy()), paintengine2d.Fill(c.s1))
		in = paintengine2d.XYWH(b.Min.X+u, b.Min.Y, b.Dx()-u, b.Dy())
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
		c.ring(ctx, b, c.s1)
		in = b.Inset(u)
	}
	mid := snap((in.Min.Y + in.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, dir Direction, press bool) {
		down := press && !st.Disabled()
		fill := c.s3
		if down {
			fill = c.pressed
		}
		ctx.DrawRect(r, paintengine2d.Fill(fill))
		if !c.ocean && !down {
			var w mtlRects
			w.add(r.Min.X, r.Min.Y, r.Dx(), u)
			w.add(r.Min.X, r.Min.Y+u, u, r.Dy()-u)
			w.fill(ctx, c.white)
		}
		col := c.text
		if st.Disabled() {
			col = c.dis
		}
		c.triangle(ctx, r.Center(), 3, dir, col)
	}
	half(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), mid-in.Min.Y), DirUp, upPress)
	half(paintengine2d.XYWH(in.Min.X, mid+u, in.Dx(), in.Max.Y-mid-u), DirDown, downPress)
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, mid, in.Dx(), u), paintengine2d.Fill(c.s1))
}

// DrawTabBar is the strip: the canvas, with the page's top edge along its
// bottom — white on Steel, primary 1 on Ocean — for the selected tab to
// open.
func (metalEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	b = mtlSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	edge := c.white
	if c.ocean {
		edge = c.p1
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-c.u, b.Dx(), c.u), paintengine2d.Fill(edge))
}

// tabShape is a tab's outline: up the left side to the slanted corner,
// across the top and down the right side, left open at the bottom.
func mtlTabShape(t paintengine2d.Rect, slant float32) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	p.MoveTo(t.Min.X, t.Max.Y)
	p.LineTo(t.Min.X, t.Min.Y+slant)
	p.LineTo(t.Min.X+slant, t.Min.Y)
	p.LineTo(t.Max.X, t.Min.Y)
	p.LineTo(t.Max.X, t.Max.Y)
	p.Close()
	return p
}

// DrawTab is Metal's tab: a slanted top-left corner, the dark edge shared
// with the neighbour, bold label. Steel's selected tab is the canvas with
// a white highlight, the others secondary 2; Ocean's selected tab is pale
// blue edged in primary 1, the others the canvas. Focus rings the inside of
// the selected tab in primary 1.
func (e metalEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	t := paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), b.Dy()-2*u)
	if selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), b.Dy()-u)
	}
	if t.Dx() < 8*u || t.Dy() < 8*u {
		return
	}
	slant := snap(l.S(6))
	if m := snap(t.Dy() * 0.35); slant > m {
		slant = m
	}
	fill, edge, hi := c.tabIdle, c.tabEdgeIdle, c.tabIdleHi
	if selected {
		fill, edge, hi = c.tabSel, c.tabEdgeSel, c.white
	}
	ctx.DrawPath(mtlTabShape(t, slant), paintengine2d.Fill(fill))
	// Edges as pixel-centred strokes: the dark outline, then the highlight
	// one pixel inside its left, slanted and top edges. An inset shape
	// keeps its diagonal one pixel to the right of the outline's when its
	// slant is a pixel shorter per pixel of inset.
	in := func(k float32) *paintengine2d.Path {
		return mtlTabShape(paintengine2d.XYWH(t.Min.X+k*u, t.Min.Y+k*u, t.Dx()-2*k*u, t.Dy()+u), slant-(k-0.5)*u)
	}
	if !nearColor(hi, fill) || selected {
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx()-u, t.Dy()))
		ctx.DrawPath(in(1.5), paintengine2d.StrokePaint(hi, u))
		ctx.Restore()
	}
	ctx.Save()
	ctx.ClipRect(t)
	ctx.DrawPath(in(0.5), paintengine2d.StrokePaint(edge, u))
	ctx.Restore()
	if selected {
		// Open the page's top edge under the selected tab.
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X+u, b.Max.Y-u, t.Dx()-2*u, u), paintengine2d.Fill(fill))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
		if !selected {
			fg = c.tabDis
		}
	}
	f := c.ctl()
	lb := paintengine2d.XYWH(t.Min.X+slant*0.5, t.Min.Y, t.Dx()-slant*0.5, t.Dy())
	l.drawFittedText(ctx, f, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(t.Min.X, t.Min.Y, t.Dx(), t.Dy()-u))
		fr := paintengine2d.XYWH(t.Min.X+2.5*u, t.Min.Y+2.5*u, t.Dx()-5*u, t.Dy()-5*u)
		ctx.DrawPath(mtlTabShape(fr, slant-2*u), paintengine2d.StrokePaint(c.tabFocus, u))
		ctx.Restore()
	}
}

// DrawPanel: the canvas; raised adds the flush border.
func (metalEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	if raised {
		c.flush(ctx, b, 1, c.s1, c.white, true)
	}
}

// DrawMenuBar: the canvas (Ocean's white-to-grey wash) over a bottom line.
func (metalEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	b = mtlSnap(b)
	if c.ocean {
		ctx.DrawRect(b, VGradient(b, c.barGrad...))
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-c.u, b.Dx(), c.u), paintengine2d.Fill(c.barLine))
}

// DrawMenuTitle: an open (or pointed-at, or keyboard-focused) title turns
// primary 2, outlined on three sides when its menu is open.
func (e metalEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.dis
	case open || st.Pressed():
		e.MenuHighlight(l, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+c.u, b.Dx(), b.Dy()-c.u), true)
		fg = c.hotText
	case st.Hovered() || st.Focused():
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+c.u, b.Dx(), b.Dy()-2*c.u), paintengine2d.Fill(c.menuHot))
		fg = c.hotText
	}
	l.drawLabeled(ctx, c.ctl(), label, underline, b, fg)
}

// DrawMenuFrame is a popup: the canvas in a primary 1 line with a white
// highlight inside its top and left.
func (metalEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	var w mtlRects
	w.add(b.Min.X+u, b.Min.Y+u, b.Dx()-2*u, u)
	w.add(b.Min.X+u, b.Min.Y+2*u, u, b.Dy()-3*u)
	w.fill(ctx, c.white)
	c.ring(ctx, b, c.p1)
}

// DrawMenuItem is a Metal menu row: the armed row across the popup, check
// and radio items with small flush indicators, the bold label, the small
// accelerator in primary 1, separators in primary 1 over white.
func (e metalEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := mtlColors(l)
	u := c.u
	ch := MenuChromeFor(l)
	left := snap(b.Min.X - ch.PadL + 2*u)
	right := snap(b.Max.X + ch.PadR - u)
	if row.Separator {
		y := snap((b.Min.Y+b.Max.Y)*0.5) - u
		if right > left {
			ctx.DrawRect(paintengine2d.XYWH(left-u, y, right-left, u), paintengine2d.Fill(c.p1))
			ctx.DrawRect(paintengine2d.XYWH(left-u, y+u, right-left, u), paintengine2d.Fill(c.white))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot && right > left {
		e.MenuHighlight(l, ctx, paintengine2d.XYWH(left, snap(b.Min.Y), right-left, snap(b.Max.Y)-snap(b.Min.Y)), false)
	}
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.dis
	case hot:
		fg = c.hotText
	}
	gw := ch.CheckCol()
	switch {
	case row.Radio || row.Checked:
		side := snap(l.S(10))
		box := mtlSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		ist := st &^ (StateHovered | StatePressed | StateFocused)
		if row.Radio {
			e.RadioIndicator(l, ctx, box, ist, row.Checked)
		} else {
			e.CheckIndicator(l, ctx, box, ist, true)
		}
	case row.Icon != IconNone:
		l.drawMenuGutter(ctx, b, ch, row, fg)
	}
	f := c.ctl()
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		c.triangle(ctx, ab.Center(), 4, DirRight, fg)
		labelRight = ab.Min.X
	}
	if row.Shortcut != "" {
		sf := c.small
		tw := sf.InkWidth(row.Shortcut)
		if tw <= 0 {
			tw = sf.Advance(row.Shortcut)
		}
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		acc := c.accel
		switch {
		case st.Disabled():
			acc = c.dis
		case hot:
			acc = c.accelHot
		}
		sf.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, b.Min.Y+(b.Dy()-sf.Height())*0.5), acc)
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

// DrawProgressBar is a dark-edged channel filled with primary 2, the fill
// edged in primary 1 along its top and left; indeterminate slides a box
// of the fill.
func (metalEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if b.Dx() < 6*u || b.Dy() < 5*u {
		return
	}
	edge := c.s1
	if st.Disabled() {
		edge = c.dis
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	c.ring(ctx, b, edge)
	in := b.Inset(u)
	if !c.ocean && !st.Disabled() {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.trackShadow))
	}
	bar := func(x0, x1 float32) {
		x0, x1 = snap(x0), snap(x1)
		if x1-x0 < u {
			return
		}
		r := paintengine2d.XYWH(x0, in.Min.Y, x1-x0, in.Dy())
		if st.Disabled() {
			ctx.DrawRect(r, paintengine2d.Fill(c.disFill))
			return
		}
		ctx.DrawRect(r, paintengine2d.Fill(c.p2))
		var e mtlRects
		e.add(r.Min.X, r.Min.Y, r.Dx(), u)
		e.add(r.Min.X, r.Min.Y+u, u, r.Dy()-u)
		e.fill(ctx, c.p1)
	}
	if indeterminate {
		if phase < 0 {
			phase = 0
		}
		phase -= float32(int(phase))
		box := in.Dx() / 6
		if m := 12 * u; box < m {
			box = m
		}
		x := in.Min.X + (in.Dx()+box)*phase - box
		bar(max(x, in.Min.X), min(x+box, in.Max.X))
		return
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	bar(in.Min.X, in.Min.X+in.Dx()*t)
}

// mtlKnob is the slider knob's outline: a flat-topped block ending in a
// point below.
func mtlKnob(r paintengine2d.Rect, point float32) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	p.MoveTo(r.Min.X, r.Min.Y)
	p.LineTo(r.Max.X, r.Min.Y)
	p.LineTo(r.Max.X, r.Max.Y-point)
	p.LineTo((r.Min.X+r.Max.X)*0.5, r.Max.Y)
	p.LineTo(r.Min.X, r.Max.Y-point)
	p.Close()
	return p
}

// DrawSlider is Metal's slider: a flush channel filled up to the knob
// (secondary 2 on Steel, a blue wash edged in primary 1 on Ocean) and the
// pointed knob — bumps on primary 2 in black on Steel, the pale wash in
// primary 1 on Ocean. Focus rings the slider in primary 2.
func (e metalEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	tw := snap(15 * u)
	th := snap(16 * u)
	if th > b.Dy()-2*u {
		th = snap(b.Dy() - 2*u)
	}
	if tw > b.Dx() {
		tw = snap(b.Dx())
	}
	if th < 8*u || tw < 7*u {
		return
	}
	ty := snap(b.Min.Y + (b.Dy()-th)*0.5)
	tx := snap(b.Min.X + (b.Dx()-tw)*t)
	trackH := 7 * u
	if c.ocean {
		trackH = 6 * u
	}
	tr := paintengine2d.XYWH(b.Min.X+u, snap(ty+(th-trackH)*0.5), b.Dx()-2*u, trackH)
	fillTo := tx + tw*0.5
	switch {
	case c.ocean:
		edge := c.s1
		if st.Disabled() {
			edge = c.dis
		}
		ctx.DrawRect(tr, paintengine2d.Fill(c.s3))
		ctx.DrawRect(paintengine2d.XYWH(tr.Min.X+u, tr.Min.Y+u, tr.Dx()-2*u, u), paintengine2d.Fill(c.p2))
		c.ring(ctx, tr, edge)
		if !st.Disabled() && fillTo > tr.Min.X+2*u {
			fr := paintengine2d.XYWH(tr.Min.X, tr.Min.Y, snap(fillTo)-tr.Min.X, tr.Dy())
			ctx.DrawRect(fr.Inset(u), VGradient(fr.Inset(u), c.fillGrad...))
			c.ring(ctx, fr, c.p1)
		}
	default:
		ctx.DrawRect(tr, paintengine2d.Fill(c.s3))
		if st.Disabled() {
			c.dimRing(ctx, tr)
		} else {
			if fillTo > tr.Min.X+2*u {
				ctx.DrawRect(paintengine2d.XYWH(tr.Min.X+u, tr.Min.Y+2*u, snap(fillTo)-tr.Min.X-u, tr.Dy()-4*u), paintengine2d.Fill(c.s2))
			}
			c.flush(ctx, tr, 1, c.s1, c.white, false)
		}
	}
	knob := paintengine2d.XYWH(tx, ty, tw, th)
	point := snap((tw - u) * 0.5)
	if point > th*0.5 {
		point = snap(th * 0.5)
	}
	edge := c.knobEdge
	if st.Disabled() {
		edge = c.dis
	}
	ctx.DrawPath(mtlKnob(knob, point), paintengine2d.Fill(edge))
	inner := paintengine2d.XYWH(knob.Min.X+u, knob.Min.Y+u, knob.Dx()-2*u, knob.Dy()-2*u)
	ip := point - u*0.4
	switch {
	case st.Disabled():
		ctx.DrawPath(mtlKnob(inner, ip), paintengine2d.Fill(c.s3))
	case c.ocean:
		ctx.DrawPath(mtlKnob(inner, ip), paintengine2d.Fill(c.knobHi))
		face := paintengine2d.XYWH(inner.Min.X+u, inner.Min.Y+u, inner.Dx()-2*u, inner.Dy()-2*u)
		ctx.DrawPath(mtlKnob(face, ip-u*0.4), VGradient(face, c.knobGrad...))
	default:
		fill := c.p2
		if st.Pressed() {
			fill = c.knobPress
		}
		ctx.DrawPath(mtlKnob(inner, ip), paintengine2d.Fill(c.knobHi))
		face := paintengine2d.XYWH(inner.Min.X+u, inner.Min.Y+u, inner.Dx()-u, inner.Dy()-u)
		ctx.DrawPath(mtlKnob(face, ip-u*0.5), paintengine2d.Fill(fill))
		tile := c.knobBumps
		if st.Focused() {
			tile = c.thumbBumps
		}
		c.bumps(ctx, paintengine2d.XYWH(face.Min.X+u, face.Min.Y+u, face.Dx()-2*u, face.Dy()-point-u), tile)
	}
	if st.Focused() && !st.Disabled() {
		c.focusRect(ctx, b, c.focus)
	}
}

// ItemFocus aside, rows follow Swing: the selection in primary 3 with the
// text colour on it, kept whether or not the view has focus.

// DrawListRow is a JList cell: the selection across the row and the focus
// rectangle round the current one.
func (e metalEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := mtlColors(l)
	fg := c.fieldText
	if st.Checked() {
		ctx.DrawRect(mtlSnap(b), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.dis
	}
	l.drawFittedText(ctx, c.body, label, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a JTree row with Metal's angled lines in primary 3, the
// key handles and the selection behind the label only.
func (e metalEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := mtlColors(l)
	u := c.u
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	var lines mtlRects
	for d := 0; d <= depth; d++ {
		if d < depth && !st.HasNextSibling(d) {
			continue
		}
		gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
		y1 := b.Max.Y
		if d == depth && !st.HasNextSibling(depth) {
			y1 = cy + u
		}
		lines.add(gx, snap(b.Min.Y), u, y1-snap(b.Min.Y))
		if d == depth {
			lines.add(gx+u, cy, snap(x+indent+l.S(2))-gx-u, u)
		}
	}
	ctx.Save()
	ctx.ClipRect(b)
	lines.fill(ctx, c.treeLine)
	ctx.Restore()
	if !leaf {
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + indent + l.S(4)
	lb := mtlSnap(paintengine2d.XYWH(lx-l.S(2), b.Min.Y+u, f.Advance(label)+l.S(4), b.Dy()-2*u)).Intersect(b)
	fg := c.fieldText
	if st.Checked() {
		ctx.DrawRect(lb, paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.dis
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() {
		e.ItemFocus(l, ctx, lb, st)
	}
}

// DrawTableHeader is a raised header cell: white at its top and left, dark
// at its right and bottom, the label centred in the plain font; pressed
// sinks to secondary 2.
func (e metalEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	fill := c.s3
	pressed := st.Pressed() && !st.Disabled()
	if pressed {
		fill = c.pressed
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	if b.Dx() > 3*u && b.Dy() > 3*u {
		var w, d mtlRects
		if !pressed {
			w.add(b.Min.X, b.Min.Y, b.Dx()-u, u)
			w.add(b.Min.X, b.Min.Y+u, u, b.Dy()-2*u)
		}
		d.add(b.Max.X-u, b.Min.Y, u, b.Dy())
		d.add(b.Min.X, b.Max.Y-u, b.Dx()-u, u)
		w.fill(ctx, c.white)
		d.fill(ctx, c.s1)
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
		c.triangle(ctx, paintengine2d.Pt(b.Max.X-l.S(4)-aw*0.5, (b.Min.Y+b.Max.Y)*0.5), 3, dir, fg)
	}
	l.drawFittedText(ctx, c.body, label, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, b.Dx()-l.S(8)-aw, b.Dy()-u), fg, AlignCenter, 0)
}

// DrawTableCell is a JTable cell: the selection in primary 3 and the grid
// lines at its right and bottom.
func (e metalEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := mtlColors(l)
	sb := mtlSnap(b)
	u := c.u
	fg := c.fieldText
	if st.Checked() {
		ctx.DrawRect(sb, paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	if st.Disabled() {
		fg = c.dis
	}
	var g mtlRects
	g.add(sb.Max.X-u, sb.Min.Y, u, sb.Dy())
	g.add(sb.Min.X, sb.Max.Y-u, sb.Dx()-u, u)
	g.fill(ctx, c.grid)
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
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
	ctx.ClipRect(paintengine2d.XYWH(sb.Min.X, sb.Min.Y, sb.Dx()-u, sb.Dy()-u))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-u-f.Height())*0.5), fg)
	ctx.Restore()
}

// DrawToolBar: the canvas (Ocean's wash) with the bumps grip at the left.
func (metalEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	if c.ocean {
		ctx.DrawRect(b, VGradient(b, c.toolGrad...))
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	}
	if b.Dx() < 16*u || b.Dy() < 8*u {
		return
	}
	grip := paintengine2d.XYWH(b.Min.X+2*u, b.Min.Y+2*u, 10*u, b.Dy()-4*u)
	if c.ocean {
		ctx.DrawRect(grip, paintengine2d.Fill(c.s3))
	}
	c.bumps(ctx, grip.Inset(u), c.gripBumps)
}

func (e metalEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := mtlColors(l)
	fg := c.face(ctx, b, st, true)
	if st.Disabled() {
		fg = c.dis
	}
	font := c.ctl()
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		right := b.Max.X - pad
		if right < x {
			right = x
		}
		if icon == IconNone {
			l.drawFittedText(ctx, font, label, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()), fg, AlignCenter, l.S(4))
		} else {
			ctx.Save()
			ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
			font.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-font.Height())*0.5), fg)
			ctx.Restore()
		}
	}
	if st.Focused() && !st.Disabled() {
		c.focusRect(ctx, mtlSnap(b).Inset(3*c.u), c.focus)
	}
}

// DrawStatusBar has no Metal original: the canvas under an etched line,
// panes split by etched rules.
func (metalEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.s2))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), u), paintengine2d.Fill(c.white))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			sx := snap(x)
			y0 := b.Min.Y + 2*u + l.S(3)
			h := b.Max.Y - y0 - l.S(3)
			ctx.DrawRect(paintengine2d.XYWH(sx-u, y0, u, h), paintengine2d.Fill(c.s2))
			ctx.DrawRect(paintengine2d.XYWH(sx, y0, u, h), paintengine2d.Fill(c.white))
		}
		l.drawFittedText(ctx, c.body, s, paintengine2d.XYWH(x+l.S(6), b.Min.Y+2*u, slot-l.S(10), b.Dy()-2*u), c.text, AlignStart, 0)
	}
}

// DrawTitleBar is a panel heading: the canvas over a primary 1 rule with
// its white highlight, the title in the bold control font.
func (metalEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*u, b.Dx(), u), paintengine2d.Fill(c.p1))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.white))
	x := b.Min.X + l.metrics.Pad
	f := c.bold2
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.groupText, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, c.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), c.dis, AlignStart, 0)
	}
}

// DrawAccordionHeader has no Metal original: a flush button face with the
// tree key and a bold title.
func (e metalEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := mtlColors(l)
	fg := c.face(ctx, b, st&^(StateFocused|StatePrimary), false)
	e.Expander(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y, l.S(16), b.Dy()-c.u), expanded, fg)
	f := c.ctl()
	tb := paintengine2d.XYWH(b.Min.X+l.S(26), b.Min.Y, b.Dx()-l.S(30), b.Dy()-c.u)
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		tw := f.Advance(title)
		if tw > tb.Dx()-2*c.u {
			tw = tb.Dx() - 2*c.u
		}
		h := f.Height() + 2*c.u
		c.focusRect(ctx, paintengine2d.XYWH(tb.Min.X-2*c.u, tb.Min.Y+(tb.Dy()-h)*0.5, tw+4*c.u, h).Intersect(mtlSnap(b).Inset(2*c.u)), c.focus)
	}
}

// DrawSeparator is JSeparator: primary 1 over white.
func (metalEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := mtlColors(l)
	u := c.u
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		y0, h := snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4))
		ctx.DrawRect(paintengine2d.XYWH(x, y0, u, h), paintengine2d.Fill(c.p1))
		ctx.DrawRect(paintengine2d.XYWH(x+u, y0, u, h), paintengine2d.Fill(c.white))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), paintengine2d.Fill(c.p1))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+u, b.Dx(), u), paintengine2d.Fill(c.white))
}

// DrawSplitter is a split pane divider: the canvas with the bumps along
// it, turned primary when it is grabbed or focused.
func (metalEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := mtlColors(l)
	b = mtlSnap(b)
	u := c.u
	ctx.DrawRect(b, paintengine2d.Fill(c.s3))
	tile := c.gripBumps
	if (st.Pressed() || st.Focused() || st.Hovered()) && !st.Disabled() {
		tile = c.gripHotBumps
	}
	g := paintengine2d.XYWH(b.Min.X+l.S(4), b.Min.Y+2*u, b.Dx()-l.S(8), b.Dy()-4*u)
	if vertical {
		g = paintengine2d.XYWH(b.Min.X+2*u, b.Min.Y+l.S(4), b.Dx()-4*u, b.Dy()-l.S(8))
	}
	c.bumps(ctx, g, tile)
}

// DrawTooltip is primary 3 in a primary 1 line, the text in the plain
// system font.
func (metalEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := mtlColors(l)
	b = mtlSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.info))
	c.ring(ctx, b, c.infoBorder)
	mtlTipText(ctx, l, c.body, text, b, c.infoText)
}

// mtlTipText draws a tool tip's text at its padding, clipped to the bubble
// (the bubble is sized to the text, so fitting it would only clip on a
// rounding).
func mtlTipText(ctx *paintengine2d.Context, l *Classic, f *Font, text string, b paintengine2d.Rect, col paintengine2d.Color) {
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	if text == "" || f == nil {
		return
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, text, paintengine2d.Pt(b.Min.X+pad, b.Min.Y+(b.Dy()-f.Height())*0.5), col)
	ctx.Restore()
}

// DrawMessageIcon draws the option pane symbols in Metal's style: flat
// shapes edged in a dark tone with a light inner edge — a primary 3 disc
// with an "i", a yellow triangle with "!", a pink octagon with an × and a
// green square with "?".
func (metalEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := mtlColors(l)
	var k int
	switch icon {
	case IconInfo:
		k = 0
	case IconWarning:
		k = 1
	case IconError:
		k = 2
	case IconQuestion:
		k = 3
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
		return
	}
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	if s < 8 {
		return
	}
	ctr := b.Center()
	r := s * 0.44
	u := s / 32
	fill, dark, hi := c.iconFill[k], c.iconDark[k], c.iconHi[k]
	var shape *paintengine2d.Path
	switch k {
	case 0:
		shape = paintengine2d.NewPath()
		shape.AddCircle(ctr, r)
	case 1:
		shape = paintengine2d.NewPath()
		shape.MoveTo(ctr.X, ctr.Y-r)
		shape.LineTo(ctr.X+r*1.05, ctr.Y+r*0.85)
		shape.LineTo(ctr.X-r*1.05, ctr.Y+r*0.85)
		shape.Close()
	case 2:
		shape = paintengine2d.NewPath()
		for i := 0; i < 8; i++ {
			a := float64(i)*math.Pi/4 + math.Pi/8
			x, y := ctr.X+r*float32(math.Cos(a)), ctr.Y+r*float32(math.Sin(a))
			if i == 0 {
				shape.MoveTo(x, y)
			} else {
				shape.LineTo(x, y)
			}
		}
		shape.Close()
	default:
		shape = paintengine2d.NewPath()
		shape.AddRect(paintengine2d.XYWH(ctr.X-r*0.9, ctr.Y-r*0.9, r*1.8, r*1.8))
	}
	ctx.DrawPath(shape, paintengine2d.Fill(fill))
	ctx.DrawPath(shape, mtlRoundStroke(dark, 1.4*u))
	f := l.bold
	if f == nil {
		f = l.body
	}
	glyph := func(g string, dy float32) {
		w := f.Advance(g)
		f.Draw(ctx, g, paintengine2d.Pt(ctr.X-w*0.5+u, ctr.Y-f.Height()*0.5+dy+u), hi)
		f.Draw(ctx, g, paintengine2d.Pt(ctr.X-w*0.5, ctr.Y-f.Height()*0.5+dy), dark)
	}
	switch k {
	case 0:
		// The "i": a round dot over a stem, each with its light edge.
		sw := 3.5 * u
		ctx.DrawRect(paintengine2d.XYWH(ctr.X-sw*0.5+u, ctr.Y-r*0.3+u, sw, r*1.0), paintengine2d.Fill(hi))
		ctx.DrawRect(paintengine2d.XYWH(ctr.X-sw*0.5, ctr.Y-r*0.3, sw, r*1.0), paintengine2d.Fill(dark))
		ctx.DrawCircle(paintengine2d.Pt(ctr.X, ctr.Y-r*0.58), sw*0.6, paintengine2d.Fill(dark))
	case 1:
		glyph("!", r*0.18)
	case 2:
		g := paintengine2d.XYWH(ctr.X-r*0.42, ctr.Y-r*0.42, r*0.84, r*0.84)
		DrawCross(ctx, g.Translate(paintengine2d.Pt(u, u)), hi, 4*u)
		DrawCross(ctx, g, dark, 4*u)
	default:
		glyph("?", 0)
	}
}

// mtlRoundStroke is an outline stroke with round joins, so sharp corners
// (a warning triangle's) stay close to the shape.
func mtlRoundStroke(col paintengine2d.Color, w float32) paintengine2d.Paint {
	return paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
}

// ---- packs --------------------------------------------------------------------

// metalPack builds a Metal pack from its theme colours (Steel's for every
// key the table leaves out).
func metalPack(name, label string, year int, summary string, over map[string]string, params map[string]float32) ThemePack {
	x := map[string]string{}
	for k, v := range mtlSteel {
		x[k] = v
	}
	for k, v := range over {
		x[k] = v
	}
	pal := metalPalette(x)
	tok := ThemeTokens{
		Engine:  "metal",
		Bevel:   BevelClassic3D,
		Family:  ThemeLight,
		Era:     "Metal",
		Palette: pal,
		Params:  params,
	}
	tok.Extra = map[string]paintengine2d.Color{}
	for k, v := range over {
		tok.Extra[k] = hexColor(v)
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: hexColor(x["secondary2"]), Border: pal.Border}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Focus}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.2), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Java", Summary: summary,
		Era: "Metal", Palette: ThemeLight, Tokens: tok,
	}
}

// metalPalette is the shared palette a Metal theme implies: the canvas in
// secondary 3, text in black, primary 1 as the accent (Metal's label and
// accelerator colour), primary 2 for focus and menus, primary 3 for
// selections.
func metalPalette(x map[string]string) Palette {
	c := func(k string) paintengine2d.Color { return hexColor(x[k]) }
	face, p1 := c("secondary3"), c("primary1")
	return Palette{
		Background: face, Surface: face, SurfaceAlt: face,
		Border: c("secondary1"), Divider: c("divider"),
		Text: c("black"), TextMuted: c("disabled"), TextOnAccent: ReadableOn(p1, 4.5, c("white"), hexColor("#000000")),
		Accent: p1, AccentHover: Mix(p1, c("primary2"), 0.4), AccentPress: Shade(p1, -0.2),
		Field: c("white"), FieldBorder: c("secondary1"),
		Focus: c("primary2"), Selection: c("primary3"),
		Track: face, Thumb: c("primary2"),
		Highlight: c("white"), Shadow: paintengine2d.RGBA(0, 0, 0, 0),
		MenuHover: c("primary2"), MenuHoverBorder: c("menuHotTop"), MenuGutter: face,
		Danger: hexColor("#b00000"), Success: hexColor("#2a7a2a"), Warning: hexColor("#9a6a00"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: c("white"), BevelDark: c("secondary1"),
	}
}

func metalPacks() []ThemePack {
	return []ThemePack{
		metalPack("metal-steel", "Metal (Steel)", 1998,
			"Swing's Java look and feel: grey-blue flush 3D borders, bumps, bold labels.", nil, nil),
		metalPack("metal-ocean", "Metal (Ocean)", 2004,
			"Java 5's Metal theme: soft blue gradients on buttons, scroll bars and sliders.", mtlOcean,
			map[string]float32{"ocean": 1}),
	}
}
