package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// plastikEngine paints KDE's Plastik widget style (shipped with KDE 3.2 in
// 2004, the default of KDE 3.4 and 3.5) and Qt 4's Plastique, its port into
// Qt itself (2006). It is written from the looks' visible behaviour and from
// colours measured on screenshots of stock KDE 3.5 and of Qt 4.8's
// Plastique widget gallery (no Plastik or Plastique code, pixmap or path
// data is used):
//
//   - every raised part is a "surface": a one-pixel contour with softly cut
//     corners around a gentle vertical gradient, lit by a one-pixel inner
//     highlight — light along the top and left, darker along the bottom and
//     right. KDE 3.5's contour is the button colour at 1/1.58 of its value
//     (#8c8d90 on #dddfe4); Plastique's is a neutral grey from the window;
//   - the mouse-over highlight lines the inside of a hot surface with the
//     selection colour, strongest along the top and bottom;
//   - the default button is inset a pixel inside a soft grey ring, with a
//     darker contour; keyboard focus is a thin rounded rectangle in the
//     selection colour; focused text fields take a selection-coloured
//     contour;
//   - check boxes are small white surfaces with a bold X, radio buttons
//     circles with a dark dot;
//   - scroll bars are a light dithered groove, white and the window colour
//     with surface step buttons at both ends and a surface slider carrying a
//     grid of raised dots;
//   - tabs have softly rounded tops and share their borders; the selected
//     tab is the window colour and opens into its page, a hot tab shows a
//     selection-coloured bar along its top — and Plastik also marks the
//     selected tab with it;
//   - progress bars fill with a selection-coloured surface: Plastik lays
//     lighter diagonal stripes over it, Plastique vertical chunks;
//   - slider handles are pointed: a small surface ending in a point below;
//   - menus are a contoured panel; the hot row and an open menu bar title
//     are selection surfaces with light text; splitters and tool bar
//     handles are rows of raised dots;
//   - in-app windows wear KWin's Plastik decoration: a blue title bar lit
//     along its top edge, a four-pixel border, square title buttons.
//
// Plastique differs from Plastik where Qt 4 did: a neutral contour, item
// views and group boxes framed in a line tinted from the selection colour,
// no bar on the selected tab, vertical progress chunks, no list stripes or
// tree branch lines, table grid lines, and right-aligned form labels (Qt's
// QFormLayout convention for Plastique).
//
// Pack data (theme.json "extra"; defaults come from the palette):
//
//	window, button, buttonText, base, alternate, disabledText
//	caption, captionText, captionOff, captionOffText   the title bar
//	tip, tipText                                        tooltips
//
// Params (0 / 1): "neutralContour", "tabBar" (bar on the selected tab),
// "chunks" (vertical progress chunks instead of stripes), "tintedFrames",
// "stripes", "branchLines", "grid", "formLabelsRight", "shadow"; and
// "toolFont", the tool bar font's size against the general font (KDE 3's
// 0.9, Qt's 1).
type plastikEngine struct{ BaseEngine }

func init() {
	RegisterEngine(plastikEngine{})
	for _, p := range plastikPacks() {
		RegisterPack(p)
	}
}

func (plastikEngine) ID() string { return "plastik" }

// DefaultMetrics are Plastik's proportions scaled from KDE 3's 10pt font
// (about 13px) to the toolkit's 16px: 16px scroll bars and indicators,
// 2px corners.
func (plastikEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Radius:    2, RadiusSmall: 2,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 16, Radio: 16,
		MenuItemH: 26, MenuBarH: 26, TabH: 28, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 18, SliderH: 24, Thumb: 13,
		Scroll: 16, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 34, StatusBarH: 24, SpinnerW: 18, SwitchW: 40, SwitchH: 20,
	}
}

// StyleHint: KDE's dialogs put the accepting button first; KDE 3 dialogs
// left-aligned their form labels as Qt 3 did, while Qt 4 gave Plastique
// right-aligned labels (the "formLabelsRight" param).
func (plastikEngine) StyleHint(l *Classic, h StyleHint) int {
	switch h {
	case HintDialogPrimaryFirst:
		return 1
	case HintFormLabelsRight:
		if l.P("formLabelsRight", 0) != 0 {
			return 1
		}
	}
	return 0
}

// ---- colours ----------------------------------------------------------------------------------

type plastik struct {
	bg, btn, base, text, btnText, dis, sel, selText, alt paintengine2d.Color
	frameDk, focus, focusOnSel, menuBg                   paintengine2d.Color
	tip, tipText                                         paintengine2d.Color

	// Contours: plain, the default button's, disabled; the default ring.
	contour, contourDef, contourDis, defRing paintengine2d.Color

	// Button surfaces: face ramps and the inner highlight lines.
	face, faceHot, faceDown, faceDis []paintengine2d.GradientStop
	hiTop, hiLeft, loBot, loRight    paintengine2d.Color
	hover, hoverSide                 paintengine2d.Color

	// Scroll bar parts shade harder, across the bar.
	sbFace, sbFaceHot, sbFaceDown []paintengine2d.GradientStop
	sbHiLeft, sbHiTop, sbLo       paintengine2d.Color
	groove                        paintengine2d.Color
	grooveTile                    *paintengine2d.Image // the white / window dither
	dotDk, dotLt                  paintengine2d.Color

	// Check boxes and radios: white surfaces with a dark mark.
	chk             []paintengine2d.GradientStop
	chkHi, chkLo    paintengine2d.Color
	chkDown, mark   paintengine2d.Color
	markDis         paintengine2d.Color
	sliderGroove    paintengine2d.Color
	sliderGrooveSel paintengine2d.Color

	// Selection surfaces: menus, progress.
	selFace          []paintengine2d.GradientStop
	selContour       paintengine2d.Color
	selHi            paintengine2d.Color
	stripe, chunk    paintengine2d.Color
	progressFrameLit paintengine2d.Color

	// Tabs and frames.
	tabFace         []paintengine2d.GradientStop
	tabHi           paintengine2d.Color
	groupLine, line paintengine2d.Color
	viewOut, viewIn paintengine2d.Color
	fieldShade      paintengine2d.Color
	fieldLit        paintengine2d.Color
	grid            paintengine2d.Color

	// KWin's Plastik decoration.
	capBase, capOffBase     paintengine2d.Color
	cap, capOff             []paintengine2d.GradientStop
	capOut, capOffOut       paintengine2d.Color
	capLit, capOffLit       paintengine2d.Color
	capLine, capOffLine     paintengine2d.Color
	capText, capOffText     paintengine2d.Color
	capShadow, capOffShadow paintengine2d.Color
}

type plastikKey struct{}

func plastikColors(l *Classic) *plastik {
	return l.Memo(plastikKey{}, func() any { return plastikBuild(l) }).(*plastik)
}

func plastikBuild(l *Classic) *plastik {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c := &plastik{
		bg:      l.X("window", p.Background),
		btn:     l.X("button", p.SurfaceAlt),
		base:    l.X("base", p.Field),
		text:    p.Text,
		btnText: l.X("buttonText", p.Text),
		dis:     l.X("disabledText", p.TextMuted),
		sel:     p.Selection,
		selText: p.TextOnAccent,
		tip:     l.X("tip", Hex("#ffffdc")),
		tipText: l.X("tipText", black),
	}
	if c.sel.A < 0.9 {
		c.sel = p.Accent
	}
	c.alt = l.X("alternate", Mix(c.base, c.sel, 0.1))
	c.frameDk = darkerPct(c.bg, 112)
	c.focus = Mix(c.sel, c.bg, 0.15)
	c.focusOnSel = Mix(c.selText, c.sel, 0.35)
	c.menuBg = c.bg

	// The contour: KDE 3.5's is the button colour at 1/1.58 of its value;
	// Plastique's a neutral grey, the window at 1/1.78.
	c.contour = darkerPct(c.btn, 158)
	c.contourDef = darkerPct(c.btn, 166)
	if l.P("neutralContour", 0) != 0 {
		g := grayLevel(c.bg) / 255
		c.contour = darkerPct(paintengine2d.RGB(g, g, g), 178)
		c.contourDef = darkerPct(paintengine2d.RGB(g, g, g), 188)
	}
	c.contourDis = Mix(c.contour, c.bg, 0.55)
	c.defRing = Mix(c.bg, c.contour, 0.35)

	// Buttons: measured on KDE 3.5's #dddfe4 button, top #e5e7ec easing to
	// #d2d4d9, a #f1f1f6 top line, #caccd1 bottom line.
	c.face = []paintengine2d.GradientStop{Stop(0, Mix(c.btn, white, 0.24)), Stop(1, darkerPct(c.btn, 105))}
	c.faceHot = []paintengine2d.GradientStop{Stop(0, Mix(c.btn, white, 0.42)), Stop(1, Mix(darkerPct(c.btn, 105), white, 0.25))}
	c.faceDown = []paintengine2d.GradientStop{Stop(0, darkerPct(c.btn, 112)), Stop(1, darkerPct(c.btn, 103))}
	c.faceDis = []paintengine2d.GradientStop{Stop(0, Mix(c.btn, c.bg, 0.6)), Stop(1, Mix(c.btn, c.bg, 0.4))}
	c.hiTop = Mix(c.btn, white, 0.55)
	c.hiLeft = Mix(c.btn, white, 0.3)
	c.loBot = darkerPct(c.btn, 110)
	c.loRight = darkerPct(c.btn, 106)
	c.hover = Mix(c.sel, c.btn, 0.15)
	c.hoverSide = Mix(c.sel, c.btn, 0.55)

	// Scroll parts: #e8e9ef easing to #cecfd5 across, #f4f4f6 left line,
	// #ebecf2 top line, #c3c4c9 right and bottom lines.
	c.sbFace = []paintengine2d.GradientStop{Stop(0, Mix(c.btn, white, 0.33)), Stop(1, darkerPct(c.btn, 107))}
	c.sbFaceHot = []paintengine2d.GradientStop{Stop(0, Mix(c.btn, white, 0.5)), Stop(1, Mix(darkerPct(c.btn, 107), white, 0.25))}
	c.sbFaceDown = []paintengine2d.GradientStop{Stop(0, darkerPct(c.btn, 110)), Stop(1, darkerPct(c.btn, 104))}
	c.sbHiLeft = Mix(c.btn, white, 0.72)
	c.sbHiTop = Mix(c.btn, white, 0.4)
	c.sbLo = darkerPct(c.btn, 113)
	c.groove = Mix(c.bg, white, 0.5) // the white / window dither, averaged
	c.grooveTile = DitherTile(c.bg, white)
	c.dotDk = darkerPct(c.btn, 150)
	c.dotLt = Mix(c.btn, white, 0.8)

	// Check boxes: #fcfcfc easing to #e1e1e1 on white, #fefefe top and left
	// lines, #dfdfdf bottom line; the X in the text colour softened.
	c.chk = []paintengine2d.GradientStop{Stop(0, darkerPct(c.base, 101)), Stop(1, darkerPct(c.base, 113))}
	c.chkHi = c.base
	c.chkLo = darkerPct(c.base, 114)
	c.chkDown = darkerPct(c.base, 112)
	c.mark = Mix(c.text, c.base, 0.2)
	c.markDis = c.dis
	c.sliderGroove = darkerPct(c.bg, 104)
	c.sliderGrooveSel = Mix(c.sel, c.bg, 0.35)

	c.selFace = []paintengine2d.GradientStop{Stop(0, Mix(c.sel, white, 0.18)), Stop(1, darkerPct(c.sel, 106))}
	c.selContour = darkerPct(c.sel, 135)
	c.selHi = Mix(c.sel, white, 0.35)
	c.stripe = Mix(c.sel, white, 0.28).WithAlpha(0.6)
	c.chunk = darkerPct(c.sel, 108)
	c.progressFrameLit = Mix(c.sel, white, 0.45)

	c.tabFace = c.face
	c.tabHi = Mix(c.bg, white, 0.6)
	c.line = Mix(c.contour, c.bg, 0.45)
	c.groupLine = c.line
	c.viewOut, c.viewIn = c.contour, c.contour
	if l.P("tintedFrames", 0) != 0 {
		// Plastique frames views and group boxes in the selection colour
		// eased towards the window (#8fa9c2 on #678db2), views twice.
		c.groupLine = Mix(c.sel, c.bg, 0.3)
		c.viewOut, c.viewIn = c.groupLine, darkerPct(c.sel, 118)
	}
	c.fieldShade = Mix(c.base, c.contour, 0.55)
	c.fieldLit = darkerPct(c.base, 110)
	c.grid = darkerPct(c.base, 115)

	// KWin Plastik: the title colour lit along the top by a line a fifth of
	// the way to white, dipping to 1/1.13 of its value below it and easing
	// back down the bar; the border's outer line at half the value.
	capA := l.X("caption", Hex("#418edc"))
	offA := l.X("captionOff", Hex("#9daaba"))
	c.capBase, c.capOffBase = capA, offA
	ramp := func(t paintengine2d.Color) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, Mix(t, white, 0.08)), Stop(0.16, darkerPct(t, 113)), Stop(1, t)}
	}
	c.cap, c.capOff = ramp(capA), ramp(offA)
	c.capOut, c.capOffOut = darkerPct(capA, 200), darkerPct(offA, 200)
	c.capLit, c.capOffLit = Mix(capA, white, 0.2), Mix(offA, white, 0.16)
	c.capLine, c.capOffLine = darkerPct(capA, 110), darkerPct(offA, 110)
	c.capText = l.X("captionText", white)
	c.capOffText = l.X("captionOffText", Hex("#dddddd"))
	c.capShadow = kde3TitleShadow(c.capText, capA)
	c.capOffShadow = kde3TitleShadow(c.capOffText, offA)
	return c
}

// ---- painting helpers -------------------------------------------------------------------------

// plR is Plastik's corner: a pixel and a half, so the corner pixel is half
// covered and its neighbours whole (0 when the Corners pref is square).
func plR(l *Classic) float32 { return l.rx(1.5) }

// surfaceKind picks a surface's palette.
type plSurface uint8

const (
	plButton plSurface = iota // push buttons, combos, tool buttons, tabs
	plScroll                  // scroll bar step buttons and sliders
)

// surface paints a Plastik surface into b: the contour, the gradient face
// and the one-pixel inner highlight, with the mouse-over highlight when
// hot. across turns the gradient sideways (vertical scroll bar parts).
func (c *plastik) surface(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, contour paintengine2d.Color, kind plSurface, across bool) {
	u := kde3U(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	r := plR(l)
	dis := st.Disabled()
	down := !dis && (st.Pressed() || (st.Toggle() && st.Checked()))
	hot := !dis && st.Hovered() && !down
	face, hiA, hiB, lo, lo2 := c.face, c.hiTop, c.hiLeft, c.loBot, c.loRight
	if kind == plScroll {
		face, hiA, hiB, lo, lo2 = c.sbFace, c.sbHiTop, c.sbHiLeft, c.sbLo, c.sbLo
		switch {
		case down:
			face = c.sbFaceDown
		case hot:
			face = c.sbFaceHot
		}
	} else {
		switch {
		case down:
			face = c.faceDown
		case hot:
			face = c.faceHot
		}
	}
	if dis {
		face = c.faceDis
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(contour))
	in := b.Inset(u)
	ri := max(r-u, 0)
	paint := VGradient(in, face...)
	if across {
		paint = HGradient(in, face...)
	}
	ctx.DrawRoundRect(in, ri, ri, paint)
	if in.Dx() < 3*u || in.Dy() < 3*u {
		return
	}
	top, left, bot, right := hiA, hiB, lo, lo2
	switch {
	case dis:
		lit := Mix(c.bg, paintengine2d.RGB(1, 1, 1), 0.3)
		top, left, bot, right = lit, lit, c.bg, c.bg
	case down:
		// A sunken surface turns its lighting over.
		top, left, bot, right = lo, lo2, hiB, hiB
	case kind == plScroll && !across:
		// Along a horizontal bar the lightest line runs on top.
		top, left = hiB, hiA
	}
	lines := [4]paintengine2d.Rect{
		paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u),
		paintengine2d.XYWH(in.Min.X, in.Min.Y+u, u, in.Dy()-2*u),
		paintengine2d.XYWH(in.Min.X, in.Max.Y-u, in.Dx(), u),
		paintengine2d.XYWH(in.Max.X-u, in.Min.Y+u, u, in.Dy()-2*u),
	}
	for i, col := range [4]paintengine2d.Color{top, left, bot, right} {
		ctx.DrawRect(lines[i], paintengine2d.Fill(col))
	}
	if hot {
		// The mouse-over highlight: the selection colour along the inside
		// of the contour, two lines deep at the top and bottom.
		hv, soft := paintengine2d.Fill(c.hover), paintengine2d.Fill(c.hoverSide)
		ctx.DrawRect(lines[0], hv)
		ctx.DrawRect(lines[2], hv)
		ctx.DrawRect(lines[1], soft)
		ctx.DrawRect(lines[3], soft)
		if in.Dy() > 6*u {
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u, in.Min.Y+u, in.Dx()-2*u, u), soft)
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u, in.Max.Y-2*u, in.Dx()-2*u, u), soft)
		}
	}
}

// plArrow fills Plastik's small solid arrow: a triangle whose base is an
// odd number of pixels (seven at 1x) and whose height is half of it plus
// one, centred in b.
func plArrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	u := kde3U(l)
	w := snap(min(l.S(7), min(b.Dx(), b.Dy())*0.7))
	if int(w/u)%2 == 0 {
		w -= u
	}
	if w < 3*u || col.A <= 0 {
		return
	}
	h := snap(w*0.5 + u*0.5)
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	var r paintengine2d.Rect
	if dir == DirLeft || dir == DirRight {
		r = paintengine2d.XYWH(cx-snap(h*0.5), cy-snap(w*0.5), h, w)
	} else {
		r = paintengine2d.XYWH(cx-snap(w*0.5), cy-snap(h*0.5), w, h)
	}
	p := kde3Path()
	defer kde3Done(p)
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

// plX fills Plastik's check mark, a bold X, into b: two bars corner to
// corner, w thick, as one path.
func plX(ctx *paintengine2d.Context, b paintengine2d.Rect, w float32, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	p := kde3Path()
	defer kde3Done(p)
	bar := func(ax, ay, bx, by float32) {
		// A rectangle around the segment a–b, w/2 out on either side.
		dx, dy := bx-ax, by-ay
		n := float32(math.Hypot(float64(dx), float64(dy)))
		if n == 0 {
			return
		}
		nx, ny := -dy/n*w*0.5, dx/n*w*0.5
		p.MoveTo(ax+nx, ay+ny)
		p.LineTo(bx+nx, by+ny)
		p.LineTo(bx-nx, by-ny)
		p.LineTo(ax-nx, ay-ny)
		p.Close()
	}
	bar(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y)
	bar(b.Max.X, b.Min.Y, b.Min.X, b.Max.Y)
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawPath(p, paintengine2d.Fill(col))
	ctx.Restore()
}

// plDots paints a grid of raised dots (a dark pixel with a light one below
// and to the right) centred in b: cols × rows, gap apart, as two paths.
func plDots(ctx *paintengine2d.Context, b paintengine2d.Rect, cols, rows int, gap, u float32, dark, light paintengine2d.Color) {
	if cols <= 0 || rows <= 0 {
		return
	}
	w := float32(cols-1)*gap + 2*u
	h := float32(rows-1)*gap + 2*u
	if w > b.Dx() || h > b.Dy() {
		return
	}
	x0 := snap((b.Min.X+b.Max.X)*0.5 - w*0.5)
	y0 := snap((b.Min.Y+b.Max.Y)*0.5 - h*0.5)
	dk, lt := kde3Path(), kde3Path()
	defer kde3Done(dk)
	defer kde3Done(lt)
	for r := 0; r < rows; r++ {
		for k := 0; k < cols; k++ {
			x, y := x0+float32(k)*gap, y0+float32(r)*gap
			lt.AddRect(paintengine2d.XYWH(x+u, y+u, u, u))
			dk.AddRect(paintengine2d.XYWH(x, y, u, u))
		}
	}
	ctx.DrawPath(lt, paintengine2d.Fill(light))
	ctx.DrawPath(dk, paintengine2d.Fill(dark))
}

// focusRect is Plastik's keyboard focus: a thin rounded rectangle in the
// selection colour, inside b.
func (c *plastik) focusRect(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := l.rx(2)
	ctx.DrawRoundRect(b.Inset(u*0.5), r, r, paintengine2d.StrokePaint(col, u))
}

// ring fills the one-unit frame of b in col (square corners), as one path.
func plRing(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, col paintengine2d.Color) {
	if b.Dx() < 2*u || b.Dy() < 2*u {
		return
	}
	ctx.DrawRect(b.Inset(u*0.5), paintengine2d.StrokePaint(col, u))
}

// ---- parts --------------------------------------------------------------------------------------

func (e plastikEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := plastikColors(l)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if b.Empty() {
		return fg
	}
	b = kde3Snap(b)
	switch role {
	case RoleButton, RoleCombo:
		c.button(l, ctx, b, st)
		if !st.Disabled() {
			fg = c.btnText
		}
	case RoleTool:
		if st.Disabled() && !(st.Toggle() && st.Checked()) {
			return fg
		}
		if st.Hovered() || st.Pressed() || (st.Toggle() && st.Checked()) {
			c.surface(l, ctx, b, st&^StateDisabled, c.contour, plButton, false)
		}
	case RoleField:
		c.field(l, ctx, b, st)
	case RoleCheck:
		e.CheckIndicator(l, ctx, b, st, st.Checked())
	case RoleRow:
		return c.row(l, ctx, b, st)
	case RoleMenu:
		if (st.Hovered() || st.Pressed() || st.Checked()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
			return c.selText
		}
	case RoleThumb:
		c.slider(l, ctx, b, b.Dy() >= b.Dx(), st)
	case RoleTrack:
		ctx.DrawRect(b, DevicePattern(ctx, c.grooveTile, kde3U(l)))
	case RoleBar, RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	}
	return fg
}

// button paints a push button into b and returns its surface: b itself, or
// a pixel inside the default button's soft ring.
func (c *plastik) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Rect {
	u := kde3U(l)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return b
	}
	contour := c.contour
	switch {
	case st.Disabled():
		contour = c.contourDis
	case st.Primary():
		r := plR(l) + u
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.defRing))
		b = b.Inset(u)
		contour = c.contourDef
	}
	c.surface(l, ctx, b, st, contour, plButton, false)
	return b
}

// field is a text field: the base colour in the contour — the selection
// colour when focused, Plastik's input focus highlight — with a shadow line
// under the top and a lighter line along the bottom.
func (c *plastik) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c.well(l, ctx, b, st, st.Focused() && !st.Disabled())
}

func (c *plastik) well(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, focus bool) {
	u := kde3U(l)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := plR(l)
	contour := c.contour
	switch {
	case st.Disabled():
		contour = c.contourDis
	case focus:
		contour = c.sel
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(contour))
	in := b.Inset(u)
	fill := c.base
	if st.Disabled() {
		fill = c.bg
	}
	ctx.DrawRoundRect(in, max(r-u, 0), max(r-u, 0), paintengine2d.Fill(fill))
	if in.Dx() > 2*u && in.Dy() > 2*u {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.fieldShade))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-u, in.Dx(), u), paintengine2d.Fill(Mix(fill, c.fieldLit, 0.6)))
	}
}

// row paints an item view row's background for st and returns its text
// colour: the selection (kept when the view loses focus, as KDE 3 did),
// the alternate background on odd rows where the pack stripes them.
func (c *plastik) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	switch {
	case st.Checked():
		fill, fg := c.sel, c.selText
		if st.Disabled() {
			fill, fg = Mix(c.sel, c.bg, 0.6), c.dis
		}
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		return fg
	case st.Alternate() && l.P("stripes", 1) != 0:
		ctx.DrawRect(b, paintengine2d.Fill(c.alt))
	}
	if st.Disabled() {
		return c.dis
	}
	return l.fieldText()
}

// CheckIndicator is a small white surface with a bold X.
func (plastikEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b := kde3Snap(box)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	r := plR(l)
	contour := c.contour
	if st.Disabled() {
		contour = c.contourDis
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(contour))
	in := b.Inset(u)
	switch {
	case st.Disabled():
		ctx.DrawRect(in, paintengine2d.Fill(c.bg))
	case st.Pressed():
		ctx.DrawRect(in, paintengine2d.Fill(c.chkDown))
	default:
		ctx.DrawRect(in, VGradient(in, c.chk...))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.chkHi))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+u, u, in.Dy()-u), paintengine2d.Fill(c.chkHi))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u, in.Max.Y-u, in.Dx()-u, u), paintengine2d.Fill(c.chkLo))
		if st.Hovered() {
			plRing(ctx, in, u, c.hoverSide)
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.hover))
			ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Max.Y-u, in.Dx(), u), paintengine2d.Fill(c.hover))
		}
	}
	if !checked {
		return
	}
	col := c.mark
	if st.Disabled() {
		col = c.markDis
	}
	s := b.Dx()
	plX(ctx, in.Inset(max(u, snap(s*0.08))), max(1.5*u, s*0.16), col)
}

// RadioIndicator is a circle in the contour, a white surface and a dark dot.
func (plastikEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := plastikColors(l)
	u := kde3U(l)
	side := min(box.Dx(), box.Dy())
	if side < 6*u {
		return
	}
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	r := side * 0.5
	contour := c.contour
	if st.Disabled() {
		contour = c.contourDis
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(contour))
	in := r - u
	bb := paintengine2d.XYWH(cx-in, cy-in, 2*in, 2*in)
	switch {
	case st.Disabled():
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), in, paintengine2d.Fill(c.bg))
	case st.Pressed():
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), in, paintengine2d.Fill(c.chkDown))
	default:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), in, VGradient(bb, c.chk...))
		// The inner highlight: light around the top, as the box's lines.
		ctx.DrawArc(paintengine2d.Pt(cx, cy), in-u*0.5, in-u*0.5, math.Pi*0.9, math.Pi*1.2, paintengine2d.StrokePaint(c.chkHi, u))
		if st.Hovered() {
			ctx.DrawCircle(paintengine2d.Pt(cx, cy), in-u*0.5, paintengine2d.StrokePaint(c.hoverSide, u))
		}
	}
	if !selected {
		return
	}
	col := c.mark
	if st.Disabled() {
		col = c.markDis
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), max(side*0.2, 1.5*u), paintengine2d.Fill(col))
}

// Arrow is Plastik's small solid triangle.
func (plastikEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	plArrow(l, ctx, b, dir, col)
}

// Expander is the KDE 3 list view's plus / minus box.
func (plastikEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := plastikColors(l)
	kde3PlusBox(l, ctx, b, expanded, c.base, c.line, c.text)
}

// MenuHighlight is the hot menu row and the open menu bar title: a
// selection surface in its darker contour.
func (plastikEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	kde3Rounded(ctx, b, plR(l), true, true, !attachBottom, true, paintengine2d.Fill(c.selContour))
	in := b.Inset(u)
	if attachBottom {
		in.Max.Y = b.Max.Y
	}
	ctx.DrawRect(in, VGradient(in, c.selFace...))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), u), paintengine2d.Fill(c.selHi))
}

func (plastikEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := plastikColors(l)
	if hot {
		return c.selText
	}
	return c.text
}

// Fields paint their own focus: the selection-coloured contour.
func (plastikEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the thin rounded selection-coloured rectangle, inside b.
func (plastikEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := plastikColors(l)
	c.focusRect(l, ctx, b, c.focus)
}

// ItemFocus rings the current item with the focus rectangle, in a colour
// that reads on a selection.
func (plastikEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := plastikColors(l)
	col := c.focus
	if st.Checked() && !st.Disabled() {
		col = c.focusOnSel
	}
	c.focusRect(l, ctx, b, col)
}

// ViewFrameInsets: item views sit in a two-pixel frame.
func (plastikEngine) ViewFrameInsets(l *Classic) Insets {
	v := 2 * kde3U(l)
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame: KDE 3.5 frames item views like its text fields (without
// the input focus highlight); Plastique in its selection-tinted pair of
// lines.
func (plastikEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if l.P("tintedFrames", 0) == 0 {
		c.well(l, ctx, b, st, false)
		return
	}
	fill := c.base
	if st.Disabled() {
		fill = c.bg
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	plRing(ctx, b, u, c.viewOut)
	plRing(ctx, b.Inset(u), u, c.viewIn)
}

// PopupShadow: KDE 3 menus cast no shadow unless the pack asks.
func (plastikEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	if l.P("shadow", 0) <= 0 {
		return Insets{}
	}
	return BaseEngine{}.PopupShadow(l, kind)
}

func (plastikEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if l.P("shadow", 0) > 0 {
		BaseEngine{}.DrawPopupShadow(l, ctx, b, kind)
	}
}

// ---- scroll bars ---------------------------------------------------------------------------------

// ScrollBarStyle: 16px bars with a step button at each end.
func (plastikEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 16, Arrows: ArrowsEnds, MinThumb: 20}
}

// slider is the scroll bar slider: a scroll surface with its grid of
// raised dots.
func (c *plastik) slider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 5*u || b.Dy() < 5*u {
		return
	}
	c.surface(l, ctx, b, st, c.contour, plScroll, vertical)
	along, across := b.Dy(), b.Dx()
	if !vertical {
		along, across = b.Dx(), b.Dy()
	}
	gap := 4 * u
	n := 5
	for n > 2 && float32(n-1)*gap+8*u > along {
		n--
	}
	if float32(n-1)*gap+8*u > along || across < 10*u {
		return
	}
	cols, rows := 3, n
	if !vertical {
		cols, rows = n, 3
	}
	plDots(ctx, b, cols, rows, gap, u, c.dotDk, c.dotLt)
}

func (e plastikEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := plastikColors(l)
	if p.Bar.Empty() {
		return
	}
	ctx.DrawRect(kde3Snap(p.Bar), DevicePattern(ctx, c.grooveTile, kde3U(l)))
	step := func(b paintengine2d.Rect, dir Direction, part ScrollPart) {
		if b.Empty() {
			return
		}
		b = kde3Snap(b)
		ps := st.Part(part)
		contour := c.contour
		if ps.Disabled() {
			contour = c.contourDis
		}
		c.surface(l, ctx, b, ps, contour, plScroll, vertical)
		col := c.btnText
		if ps.Disabled() {
			col = c.dis
		}
		plArrow(l, ctx, b, dir, col)
	}
	dec, inc := DirUp, DirDown
	if !vertical {
		dec, inc = DirLeft, DirRight
	}
	step(p.Dec, dec, ScrollDec)
	step(p.Inc, inc, ScrollInc)
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	ts := StateNone
	if st.Hot == ScrollThumbPart {
		ts |= StateHovered
	}
	if st.Pressed == ScrollThumbPart {
		ts |= StatePressed
	}
	c.slider(l, ctx, p.Thumb, vertical, ts)
}

// DrawScrollBar paints a bare groove and slider (widgets that lay out their
// own bar).
func (e plastikEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	vertical := track.Dy() >= track.Dx()
	ss := ScrollState{Disabled: st.Disabled()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	if st.Hovered() {
		ss.Hot, ss.Hovered = ScrollThumbPart, true
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, vertical, ss)
}

// ---- frames -------------------------------------------------------------------------------------

func (plastikEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox: a softly rounded one-pixel frame (grey on Plastik, tinted
// on Plastique), the title breaking its top edge at the left.
func (plastikEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	// The window colour behind the title too: a titled panel may float
	// over something else (an overlay's dialog).
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	top := b.Min.Y
	f := l.body
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	fr := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if fr.Dx() < 6*u || fr.Dy() < 6*u {
		return
	}
	r := l.rx(3)
	fill := c.bg
	if raised {
		fill = Mix(c.bg, paintengine2d.RGB(1, 1, 1), 0.25)
	}
	ctx.DrawRoundRect(fr, r, r, paintengine2d.Fill(c.groupLine))
	ctx.DrawRoundRect(fr.Inset(u), max(r-u, 0), max(r-u, 0), paintengine2d.Fill(fill))
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(8)
	tw := min(f.Advance(title)+l.S(6), b.Max.X-l.S(4)-tx)
	if tw <= 0 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), paintengine2d.Fill(c.bg))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// plCaptionH is the decoration's title bar height at the UI font.
func plCaptionH(l *Classic) float32 {
	return max(snap(l.body.Height()+l.S(9)), snap(l.S(24)))
}

func (plastikEngine) WindowFrameInsets(l *Classic) Insets {
	fr := 4 * kde3U(l)
	return Insets{Top: plCaptionH(l), Right: fr, Bottom: fr, Left: fr}
}

// WindowCloseRect is the square close button at the right of the title bar.
func (plastikEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = kde3Snap(b)
	h := plCaptionH(l)
	if b.Dy() < h || b.Dx() < h*3 {
		return paintengine2d.Rect{}
	}
	side := snap(h - l.S(8))
	return paintengine2d.XYWH(snap(b.Max.X-side-l.S(6)), snap(b.Min.Y+(h-side)*0.5+kde3U(l)*0.5), side, side)
}

// DrawWindowFrame is KWin's Plastik decoration: the title colour around the
// body in a four-pixel border (a dark outer line, a light line, the colour,
// a darker line), the title bar lit along its top, the bold title at the
// left and the square close button.
func (e plastikEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	h := plCaptionH(l)
	in := e.WindowFrameInsets(l)
	if b.Dx() < in.Left+in.Right+h*2 || b.Dy() < h+in.Bottom+u {
		return
	}
	base, stops, out, lit, line := c.capBase, c.cap, c.capOut, c.capLit, c.capLine
	fg, shadow := c.capText, c.capShadow
	if !st.Active {
		base, stops, out, lit, line = c.capOffBase, c.capOff, c.capOffOut, c.capOffLit, c.capOffLine
		fg, shadow = c.capOffText, c.capOffShadow
	}
	r := l.rx(3)
	kde3Rounded(ctx, b, r, true, true, false, true, paintengine2d.Fill(out))
	b1 := b.Inset(u)
	r1 := max(r-u, 0)
	kde3Rounded(ctx, b1, r1, true, true, false, true, paintengine2d.Fill(lit))
	b2 := b1.Inset(u)
	r2 := max(r1-u, 0)
	kde3Rounded(ctx, b2, r2, true, true, false, true, paintengine2d.Fill(base))
	bar := paintengine2d.XYWH(b2.Min.X, b2.Min.Y, b2.Dx(), b.Min.Y+h-b2.Min.Y-u)
	kde3Rounded(ctx, bar, r2, true, true, false, true, VGradient(bar, stops...))
	body := in.Apply(b)
	// The darker line around the client, then the body.
	ctx.DrawRect(body.Inset(-u), paintengine2d.Fill(line))
	ctx.DrawRect(body, paintengine2d.Fill(c.bg))
	cb := e.WindowCloseRect(l, b)
	right := bar.Max.X - l.S(6)
	if st.CanClose && !cb.Empty() {
		c.closeButton(l, ctx, cb, base, fg, st)
	}
	if !cb.Empty() {
		right = cb.Min.X - l.S(4)
	}
	if title == "" {
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	tb := paintengine2d.XYWH(bar.Min.X+l.S(8), b.Min.Y, right-bar.Min.X-l.S(8), h)
	if shadow.A > 0 {
		l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(u, u)), shadow, AlignStart, 0)
	}
	l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
}

// closeButton is the decoration's square button: a lighter face in a light
// frame with a white cross.
func (c *plastik) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, base, glyph paintengine2d.Color, st WindowState) {
	u := kde3U(l)
	white := paintengine2d.RGB(1, 1, 1)
	r := l.rx(2)
	face := Mix(base, white, 0.12)
	switch {
	case st.ClosePress:
		face = darkerPct(base, 118)
	case st.CloseHot:
		face = Mix(base, white, 0.32)
	}
	ctx.DrawRoundRect(cb, r, r, paintengine2d.Fill(Mix(base, white, 0.45)))
	ctx.DrawRoundRect(cb.Inset(u), max(r-u, 0), max(r-u, 0), paintengine2d.Fill(face))
	g := cb.Inset(cb.Dx() * 0.3)
	DrawCross(ctx, g, glyph, max(l.S(2), u))
}

// TabOverlap: neighbouring tabs share one contour line.
func (plastikEngine) TabOverlap(l *Classic) float32 { return kde3U(l) }

// plPaneTop is where a tab pane's contour runs: the tab bar's last row.
func plPaneTop(l *Classic, b paintengine2d.Rect) float32 {
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		return b.Min.Y
	}
	return snap(b.Min.Y + th - kde3U(l))
}

// DrawTabPane is the page under the tabs: the window colour in the contour
// with a light inner line along its top and left.
func (plastikEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	top := plPaneTop(l, b)
	pane := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if pane.Dx() < 4*u || pane.Dy() < 4*u {
		return
	}
	ctx.DrawRect(pane, paintengine2d.Fill(c.contour))
	kde3Frame(ctx, pane.Inset(u), u, c.bg, c.tabHi, c.frameDk)
}

// ToolBarInsets keeps room for the tool bar handle's column of dots.
func (plastikEngine) ToolBarInsets(l *Classic) Insets {
	return Insets{Left: snap(l.S(12)), Right: snap(l.S(6))}
}

// ControlFont: tool buttons are labelled in KDE 3's smaller tool bar font.
func (plastikEngine) ControlFont(l *Classic, role Role) *Font { return kde3ControlFont(l, role) }

func (plastikEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	kde3ToolButton(l, ctx, b, st, label, icon, plastikColors(l).dis)
}

// ---- controls -----------------------------------------------------------------------------------

func (e plastikEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := plastikColors(l)
	b = kde3Snap(b)
	body := c.button(l, ctx, b, st)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	// Plastik keeps the label still when the button is pressed.
	l.drawFittedText(ctx, l.body, label, body, fg, AlignCenter, l.S(8))
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, body.Inset(snap(l.S(3))), c.focus)
	}
}

// toggle draws a check box or radio caption and, with focus, the focus
// rectangle around it.
func (plastikEngine) toggle(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := plastikColors(l)
	u := kde3U(l)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(l, ctx, box.Inset(-u).Intersect(b), c.focus)
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	gap := l.S(6)
	lb := paintengine2d.XYWH(box.Max.X+gap, b.Min.Y, b.Max.X-box.Max.X-gap, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, labelFocusRect(l.body, label, lb, b), c.focus)
	}
}

func (e plastikEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := min(l.metrics.Checkbox, b.Dy())
	box := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggle(l, ctx, b, box, st, label)
}

func (e plastikEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	side = min(side, b.Dy())
	box := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggle(l, ctx, b, box, st, label)
}

// DrawSwitch: KDE 3 had no switch; this one is a slider groove that fills
// with the selection colour, with a surface knob.
func (e plastikEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := plastikColors(l)
	u := kde3U(l)
	m := l.metrics
	tw, th := min(m.SwitchW, b.Dx()), min(m.SwitchH, b.Dy())
	track := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*u || track.Dy() < 6*u {
		return
	}
	g := kde3Snap(paintengine2d.XYWH(track.Min.X, track.Min.Y+track.Dy()*0.25, track.Dx(), track.Dy()*0.5))
	fill := c.sliderGroove
	if on && !st.Disabled() {
		fill = c.sliderGrooveSel
	}
	r := plR(l)
	ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(c.contour))
	ctx.DrawRoundRect(g.Inset(u), max(r-u, 0), max(r-u, 0), paintengine2d.Fill(fill))
	kw := track.Dy()
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	contour := c.contour
	if st.Disabled() {
		contour = c.contourDis
	}
	c.surface(l, ctx, paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), st&^StateToggle, contour, plButton, false)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			c.focusRect(l, ctx, track, c.focus)
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, labelFocusRect(l.body, label, lb, b), c.focus)
	}
}

// DrawSlider is a thin contoured groove with Plastik's pointed handle: a
// surface with softly rounded top corners that ends in a point below.
func (e plastikEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	t = max(0, min(t, 1))
	hw := snap(l.S(11))
	if int(hw/u)%2 == 0 {
		hw += u
	}
	hh := min(snap(l.S(18)), b.Dy())
	if b.Dx() < hw+2*u || hh < 10*u {
		return
	}
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	gt := 4 * u
	g := paintengine2d.XYWH(b.Min.X, cy-gt*0.5-u, b.Dx(), gt)
	r := plR(l)
	ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(c.contour))
	ctx.DrawRoundRect(g.Inset(u), max(r-u, 0), max(r-u, 0), paintengine2d.Fill(c.sliderGroove))
	hx := snap(b.Min.X + (b.Dx()-hw)*t)
	hy := snap(cy - hh*0.5)
	c.pointedHandle(l, ctx, paintengine2d.XYWH(hx, hy, hw, hh), st)
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, b, c.focus)
	}
}

// pointedHandle paints the slider handle into h: contour, face and inner
// highlight following the pentagon, two raised dots near its top.
func (c *plastik) pointedHandle(l *Classic, ctx *paintengine2d.Context, h paintengine2d.Rect, st ControlState) {
	u := kde3U(l)
	p := kde3Path()
	defer kde3Done(p)
	shape := func(b paintengine2d.Rect, r float32) *paintengine2d.Path {
		pt := b.Dx() * 0.5
		p.Reset()
		p.MoveTo(b.Min.X+r, b.Min.Y)
		p.LineTo(b.Max.X-r, b.Min.Y)
		p.QuadTo(b.Max.X, b.Min.Y, b.Max.X, b.Min.Y+r)
		p.LineTo(b.Max.X, b.Max.Y-pt)
		p.LineTo((b.Min.X+b.Max.X)*0.5, b.Max.Y)
		p.LineTo(b.Min.X, b.Max.Y-pt)
		p.LineTo(b.Min.X, b.Min.Y+r)
		p.QuadTo(b.Min.X, b.Min.Y, b.Min.X+r, b.Min.Y)
		p.Close()
		return p
	}
	contour := c.contour
	face := c.face
	dis := st.Disabled()
	switch {
	case dis:
		contour, face = c.contourDis, c.faceDis
	case st.Pressed():
		face = c.faceDown
	case st.Hovered():
		face = c.faceHot
	}
	r := plR(l)
	ctx.DrawPath(shape(h, r), paintengine2d.Fill(contour))
	// The inner outline: the same pentagon a pixel in (the point moves up
	// by the diagonal's extra length).
	in := paintengine2d.Rect{Min: paintengine2d.Pt(h.Min.X+u, h.Min.Y+u), Max: paintengine2d.Pt(h.Max.X-u, h.Max.Y-u*1.414)}
	hi := c.hiTop
	if st.Hovered() && !dis && !st.Pressed() {
		hi = c.hover
	}
	ctx.DrawPath(shape(in, max(r-u, 0)), paintengine2d.Fill(hi))
	in2 := paintengine2d.Rect{Min: paintengine2d.Pt(in.Min.X+u, in.Min.Y+u), Max: paintengine2d.Pt(in.Max.X, in.Max.Y)}
	ctx.DrawPath(shape(in2, 0), VGradient(in2, face...))
	if !dis && in2.Dx() >= 5*u {
		plDots(ctx, paintengine2d.XYWH(in2.Min.X, in2.Min.Y+u, in2.Dx(), 3*u), 2, 1, 3*u, u, c.dotDk, c.dotLt)
	}
}

// DrawProgressBar: a white groove in the contour, filled with a selection
// surface — under Plastik's lighter diagonal stripes, or Plastique's
// vertical chunks; the busy bar is a striped block bouncing between the
// ends.
func (e plastikEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	c.well(l, ctx, b, st&^StateFocused, false)
	in := b.Inset(u)
	t = max(0, min(t, 1))
	bar := in
	if indeterminate {
		bar = kde3Bounce(in, phase, 0.3)
	} else {
		w := snap(in.Dx() * t)
		if w < 3*u {
			return
		}
		bar.Max.X = bar.Min.X + w
	}
	if st.Disabled() {
		ctx.DrawRect(bar, paintengine2d.Fill(c.contourDis))
		ctx.DrawRect(bar.Inset(u), VGradient(bar, c.faceDis...))
		return
	}
	chunks := l.P("chunks", 0) != 0
	if chunks {
		// Plastique: the selection colour in a lighter frame, darker
		// vertical chunks every other ten pixels.
		ctx.DrawRect(bar, paintengine2d.Fill(c.progressFrameLit))
		bi := bar.Inset(u)
		if bi.Empty() {
			return
		}
		ctx.DrawRect(bi, paintengine2d.Fill(c.sel))
		p := kde3Path()
		defer kde3Done(p)
		step := 10 * u
		for x := bi.Min.X + step; x < bi.Max.X; x += 2 * step {
			p.AddRect(paintengine2d.XYWH(x, bi.Min.Y, min(step, bi.Max.X-x), bi.Dy()))
		}
		ctx.DrawPath(p, paintengine2d.Fill(c.chunk))
		return
	}
	ctx.DrawRect(bar, paintengine2d.Fill(c.selContour))
	bi := bar.Inset(u)
	if bi.Empty() {
		return
	}
	ctx.DrawRect(bi, VGradient(bi, c.selFace...))
	// The lighter stripes lean 45°, ten pixels wide every twenty.
	ctx.Save()
	ctx.ClipRect(bi)
	p := kde3Path()
	defer kde3Done(p)
	h := bi.Dy()
	w := 10 * u
	for x := bi.Min.X - h; x < bi.Max.X; x += 2 * w {
		p.MoveTo(x, bi.Max.Y)
		p.LineTo(x+w, bi.Max.Y)
		p.LineTo(x+w+h, bi.Min.Y)
		p.LineTo(x+h, bi.Min.Y)
		p.Close()
	}
	ctx.DrawPath(p, paintengine2d.Fill(c.stripe))
	ctx.Restore()
	ctx.DrawRect(paintengine2d.XYWH(bi.Min.X, bi.Min.Y, bi.Dx(), u), paintengine2d.Fill(c.selHi))
}

// DrawComboBox is a push button surface with a contour line before the
// arrow's part and the small arrow.
func (e plastikEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	bst := st &^ StatePrimary
	if open {
		bst = (bst | StatePressed) &^ StateHovered
	}
	body := c.button(l, ctx, b, bst)
	aw := min(snap(l.S(18)), body.Dx()*0.4)
	sx := body.Max.X - u - aw
	contour := c.contour
	if st.Disabled() {
		contour = c.contourDis
	}
	if body.Dy() > 4*u {
		ctx.DrawRect(paintengine2d.XYWH(sx, body.Min.Y+u, u, body.Dy()-2*u), paintengine2d.Fill(contour))
		ctx.DrawRect(paintengine2d.XYWH(sx+u, body.Min.Y+2*u, u, body.Dy()-4*u), paintengine2d.Fill(c.hiLeft))
	}
	col := c.btnText
	if st.Disabled() {
		col = c.dis
	}
	plArrow(l, ctx, paintengine2d.XYWH(sx+u, body.Min.Y, aw, body.Dy()), DirDown, col)
	tb := paintengine2d.XYWH(body.Min.X+l.S(7), body.Min.Y, sx-body.Min.X-l.S(10), body.Dy())
	l.drawFittedText(ctx, l.body, text, tb, col, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		c.focusRect(l, ctx, paintengine2d.XYWH(body.Min.X+l.S(3), body.Min.Y+l.S(3), sx-body.Min.X-l.S(5), body.Dy()-l.S(6)), c.focus)
	}
}

// DrawSpinner is the spin box's column of two stacked surfaces with small
// arrows.
func (e plastikEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, hot, down bool, dir Direction) {
		s := st &^ (StateHovered | StatePressed | StateToggle | StateFocused)
		if hot {
			s |= StateHovered
		}
		if down {
			s |= StatePressed
		}
		contour := c.contour
		if st.Disabled() {
			contour = c.contourDis
		}
		c.surface(l, ctx, r, s, contour, plButton, false)
		col := c.btnText
		if st.Disabled() {
			col = c.dis
		}
		plArrow(l, ctx, r, dir, col)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y+u), upHover, upPress, DirUp)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), downHover, downPress, DirDown)
}

// DrawTabBar: the window colour with the page's contour along the bottom.
func (e plastikEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.contour))
}

// DrawTab: softly rounded tops in the contour, sharing borders with their
// neighbours; unselected tabs sit three pixels lower on the button
// gradient, the selected tab is the window colour and opens into its
// page. A hot tab shows the selection colour along its top; Plastik also
// marks the selected tab so.
func (e plastikEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 8*u || b.Dy() < 10*u {
		return
	}
	paneY := b.Max.Y - u
	t := b
	if !selected {
		t = paintengine2d.XYWH(b.Min.X, b.Min.Y+3*u, b.Dx(), paneY-b.Min.Y-3*u)
	}
	r := l.rx(2.5)
	contour := c.contour
	if st.Disabled() && !selected {
		contour = c.contourDis
	}
	kde3Rounded(ctx, t, r, true, true, false, true, paintengine2d.Fill(contour))
	in := paintengine2d.XYWH(t.Min.X+u, t.Min.Y+u, t.Dx()-2*u, t.Dy()-u)
	ri := max(r-u, 0)
	switch {
	case selected:
		kde3Rounded(ctx, in, ri, true, true, false, true, paintengine2d.Fill(c.bg))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X+ri, in.Min.Y, in.Dx()-2*ri, u), paintengine2d.Fill(c.tabHi))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+ri, u, in.Dy()-ri), paintengine2d.Fill(c.tabHi))
	default:
		face := c.tabFace
		if st.Disabled() {
			face = c.faceDis
		}
		kde3Rounded(ctx, in, ri, true, true, false, true, VGradient(in, face...))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X+ri, in.Min.Y, in.Dx()-2*ri, u), paintengine2d.Fill(c.hiTop))
	}
	hot := !selected && st.Hovered() && !st.Disabled()
	if hot || (selected && l.P("tabBar", 0) != 0 && !st.Disabled()) {
		col := c.sel
		if hot {
			col = Mix(c.sel, c.btn, 0.25)
		}
		kde3Rounded(ctx, paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), 2*u), min(ri, 2*u), true, true, false, true, paintengine2d.Fill(col))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	lb := paintengine2d.XYWH(t.Min.X, t.Min.Y+u, t.Dx(), t.Dy()-u)
	if selected {
		lb.Max.Y -= u
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		f := l.body
		w := min(f.Advance(label)+l.S(8), lb.Dx()-l.S(6))
		h := min(f.Height()+l.S(2), lb.Dy()-2*u)
		c.focusRect(l, ctx, paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h), c.focus)
	}
}

// DrawPanel: flat panels are the window colour in a sunken one-pixel
// frame; raised panels a surface-lit face in the contour.
func (e plastikEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if raised {
		ctx.DrawRect(b, paintengine2d.Fill(c.line))
		kde3Frame(ctx, b.Inset(u), u, Mix(c.bg, paintengine2d.RGB(1, 1, 1), 0.25), c.tabHi, c.frameDk)
		return
	}
	kde3Frame(ctx, b, u, c.bg, c.line, c.tabHi)
}

func (e plastikEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(plastikColors(l).bg))
}

// DrawMenuTitle: an open title is a selection surface with light text; a
// hot one shows a button surface, so the pointer is seen.
func (e plastikEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	hot := (open || st.Pressed()) && !st.Disabled()
	switch {
	case hot:
		e.MenuHighlight(l, ctx, b.Inset(u), false)
	case st.Hovered() && !st.Disabled():
		c.surface(l, ctx, b.Inset(u), StateNone, c.line, plButton, false)
	}
	fg := e.MenuTextColor(l, hot)
	if st.Disabled() {
		fg = c.dis
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		c.focusRect(l, ctx, b.Inset(u), c.focus)
	}
}

// DrawMenuFrame: the window colour in the contour, lit inside along the
// top and left.
func (e plastikEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.contour))
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	kde3Frame(ctx, b.Inset(u), u, c.menuBg, c.tabHi, c.frameDk)
}

// DrawMenuItem: etched separators; checked rows carry the check box's X.
func (e plastikEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := plastikColors(l)
	kde3MenuItem(l, ctx, b, st, row, c.line, c.tabHi, c.dis, func(ctx *paintengine2d.Context, box paintengine2d.Rect, col paintengine2d.Color) {
		plX(ctx, box.Inset(box.Dx()*0.18), max(1.5*kde3U(l), box.Dx()*0.15), col)
	})
}

// DrawMessageIcon paints the message box icons in the manner of KDE 3's
// Crystal set.
func (plastikEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	kde3MessageIcon(l, ctx, b, icon)
}

func (e plastikEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := plastikColors(l)
	fg := c.row(l, ctx, b, st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow: the KDE 3 tree (dotted branch lines unless the pack drops
// them, as Qt 4 did), the plus / minus box and the selection from the
// label to the row's end.
func (e plastikEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := plastikColors(l)
	mid := c.line
	if l.P("branchLines", 1) == 0 {
		mid = paintengine2d.Color{}
	}
	kde3TreeRow(l, ctx, b, st, expanded, leaf, depth, label, bold, kde3TreeColors{
		mid: mid, text: c.text, dis: c.dis, sel: c.sel, selText: c.selText, alt: c.alt, base: c.base,
	}, e.Expander, e.ItemFocus)
}

// DrawTableCell: KDE 3's detail views have no grid; Qt 4's table views
// (Plastique) draw one.
func (e plastikEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := plastikColors(l)
	fg := c.row(l, ctx, b, st)
	kde3CellText(l, ctx, b, label, align, face, fg)
	if l.P("grid", 0) != 0 {
		u := kde3U(l)
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y, u, b.Dy()), paintengine2d.Fill(c.grid))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx()-u, u), paintengine2d.Fill(c.grid))
	}
}

// DrawTableHeader is a square button surface section: the top light line,
// the contour below and between sections, the sort arrow.
func (e plastikEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Empty() {
		return
	}
	face := c.face
	switch {
	case st.Disabled():
		face = c.faceDis
	case st.Pressed():
		face = c.faceDown
	case st.Hovered():
		face = c.faceHot
	}
	ctx.DrawRect(b, VGradient(b, face...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.hiTop))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*u, b.Dx(), u), paintengine2d.Fill(c.loBot))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.contour))
	if !st.Last() {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y, u, b.Dy()-u), paintengine2d.Fill(c.contour))
	}
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		plArrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar: the window colour with the handle at the left, a staggered
// column of raised dots.
func (e plastikEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := plastikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if b.Dy() < 10*u {
		return
	}
	dk, lt := kde3Path(), kde3Path()
	defer kde3Done(dk)
	defer kde3Done(lt)
	x := snap(b.Min.X + l.S(3))
	i := 0
	for y := snap(b.Min.Y + l.S(5)); y+2*u <= b.Max.Y-l.S(5); y += 3 * u {
		xx := x
		if i%2 == 1 {
			xx += 3 * u
		}
		dk.AddRect(paintengine2d.XYWH(xx, y, u, u))
		lt.AddRect(paintengine2d.XYWH(xx+u, y+u, u, u))
		i++
	}
	ctx.DrawPath(lt, paintengine2d.Fill(c.dotLt))
	ctx.DrawPath(dk, paintengine2d.Fill(c.dotDk))
}

func (e plastikEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := plastikColors(l)
	kde3StatusBar(l, ctx, b, parts, c.bg, c.line, c.tabHi, c.text)
}

// DrawTitleBar (panel headings): the window colour, a bold title and an
// etched rule below.
func (e plastikEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := plastikColors(l)
	kde3Heading(l, ctx, b, title, subtitle, c.bg, c.line, c.tabHi, c.text, c.dis)
}

// DrawAccordionHeader is a Qt tool box tab: a button surface with the
// branch arrow and the title.
func (e plastikEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := plastikColors(l)
	b = kde3Snap(b)
	contour := c.contour
	if st.Disabled() {
		contour = c.contourDis
	}
	c.surface(l, ctx, b, st&^(StateFocused|StateToggle), contour, plButton, false)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	plArrow(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, l.S(12), b.Dy()), dir, fg)
	lb := paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy())
	l.drawFittedText(ctx, l.body, title, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusRect(l, ctx, labelFocusRect(l.body, title, lb, b), c.focus)
	}
}

func (e plastikEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := plastikColors(l)
	kde3Separator(l, ctx, b, vertical, c.line, c.tabHi)
}

// DrawSplitter: the window colour and a short row of raised dots across
// the handle's middle, lit in the selection colour under the pointer.
func (e plastikEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := plastikColors(l)
	u := kde3U(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.sel.WithAlpha(0.25)))
	}
	if vertical {
		plDots(ctx, b, 1, 5, 3*u, u, c.dotDk, c.dotLt)
		return
	}
	plDots(ctx, b, 5, 1, 3*u, u, c.dotDk, c.dotLt)
}

// DrawTooltip: the pale yellow tip in a one-pixel black frame.
func (e plastikEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := plastikColors(l)
	kde3Tooltip(l, ctx, b, text, c.tip, c.tipText)
}

// ---- packs --------------------------------------------------------------------------------------

func plastikPacks() []ThemePack {
	// KDE 3.4 and 3.5's default colour scheme (the "Plastik" scheme): a
	// neutral light grey window, blue-grey buttons, a steel-blue selection
	// with white text, a vivid blue active title bar.
	kde35 := kde3Scheme{
		background: "#efefef", foreground: "#000000",
		button: "#dddfe4", buttonText: "#000000",
		selection: "#678db2", selectionText: "#ffffff",
		base: "#ffffff", text: "#000000",
		alternate: "#edf4f9", link: "#0000ee",
		caption: "#418edc", captionBlend: "#6b91b8", captionText: "#ffffff",
		captionOff: "#9daaba", captionOffBlend: "#9daaba", captionOffText: "#dddddd",
	}
	// Qt 4's Plastique standard palette: the same window, button and
	// highlight, the alternate base the base at 1/1.1 of its value, and the
	// MDI title bar in the highlight.
	qt4 := kde35
	qt4.alternate = "#e8e8e8"
	qt4.caption, qt4.captionBlend = "#678db2", "#678db2"
	qt4.captionOff, qt4.captionOffBlend, qt4.captionOffText = "#c7c7c7", "#c7c7c7", "#000000"
	return []ThemePack{
		kde3Pack("plastik", "Plastik", 2004, "KDE", "Plastik",
			"KDE 3.5's default: flat gradient surfaces in a soft contour, the blue mouse-over highlight, dotted grips and striped progress.",
			"plastik", BevelClassic3D, kde35, map[string]float32{
				"shadow": 0, "stripes": 1, "branchLines": 1, "tabBar": 1,
			}),
		kde3Pack("plastique", "Plastique", 2006, "Qt", "Plastik",
			"Qt 4's port of Plastik: neutral contours, selection-tinted frames, chunked progress and right-aligned form labels.",
			"plastik", BevelClassic3D, qt4, map[string]float32{
				"shadow": 0, "stripes": 0, "branchLines": 0, "tabBar": 0,
				"neutralContour": 1, "tintedFrames": 1, "chunks": 1, "grid": 1, "formLabelsRight": 1, "toolFont": 1,
			}),
	}
}
