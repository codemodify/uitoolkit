package style

import (
	"math"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// keramikEngine paints KDE's Keramik widget style, the default look of KDE
// 3.1 to 3.3 (designed in 2002). Keramik was assembled from pixmap tiles;
// this engine paints the same shapes as vector gradients, written from the
// look's visible behaviour and from colours measured on screenshots of KDE
// 3.1 (no Keramik code, tile or path data is used):
//
//   - every raised part is a gel. Over the selection colour — the menu
//     highlight, the scroll bar handle, the progress bar — the ramp starts
//     almost white, dips to its darkest tone a little below the middle and
//     lifts again along the bottom edge (kmGelStops); over the button colour
//     — push buttons, tabs, combos, step buttons — it keeps its brightest
//     band near the top, falls through the middle to a flat lower half and
//     carries a lighter line above the bottom edge (kmButtonStops);
//   - push buttons are 5px-rounded gels in a grey outline with a light top
//     edge and an almost black bottom edge; their ends shade darker like a
//     cylinder, and a soft two-pixel shadow falls right and below. The
//     default button sits in a sunken ring, dark above and white below;
//   - check boxes are small square gels with a bold dark tick, radio buttons
//     glossy beads with a dark dot;
//   - scroll bars are sunken rounded grooves holding a gel handle in the
//     selection colour with a three-line grip; both step buttons sit
//     together at the bottom (right) end, and their arrows are block arrows;
//   - combo boxes are push buttons with a three-line divider before a block
//     arrow under a bar; tabs have rounded tops, the selected one taller
//     and lighter, standing in the page's raised frame;
//   - menus are a lighter panel in a dark frame; the hot row is the
//     selection gel between two darker lines, with dark text;
//   - in-app windows wear the Keramik window decoration: a blue gel caption
//     with rounded top corners and the title in a raised "bubble".
//
// Menus cast no shadow (KDE 3.1 had none; the "shadow" param turns one on).
// Lists keep their selection when they lose focus and stripe odd rows in
// the scheme's alternate background, as KDE 3 list views did.
//
// Pack data (theme.json "extra"; defaults come from the palette):
//
//	window, button, buttonText, base, alternate, disabledText
//	caption, caption2, captionText        the active title bar
//	captionOff, captionOff2, captionOffText the inactive one
//	tip, tipText                            tooltips
//
// Params: "shadow" (0 = KDE 3.1's flat menus), "stripes" (1 = alternate
// rows), "sbHighlight" (1 = scroll handles in the selection colour, as the
// Keramik option did by default), "formLabelsRight" (0: Qt 3 dialogs),
// "toolFont" (the tool bar font's size against the general font, 0.9).
type keramikEngine struct{ BaseEngine }

func init() {
	RegisterEngine(keramikEngine{})
	for _, p := range keramikPacks() {
		RegisterPack(p)
	}
}

func (keramikEngine) ID() string { return "keramik" }

// DefaultMetrics are Keramik's proportions scaled from KDE 3's 10pt font
// (about 13px) to the toolkit's 16px: chunky 32px buttons (shadow
// included), 17px scroll bars and check boxes, 5px corners.
func (keramikEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Radius:    5, RadiusSmall: 3,
		ControlH: 32, FieldH: 28, ComboH: 30,
		Checkbox: 17, Radio: 17,
		MenuItemH: 26, MenuBarH: 28, TabH: 30, RowH: 22,
		TitleBar: 28, HeaderH: 26, ProgressH: 20, SliderH: 26, Thumb: 18,
		Scroll: 17, Pad: 10, FieldPad: 6, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 18, SwitchW: 44, SwitchH: 22,
	}
}

// StyleHint: KDE's dialogs put the accepting button first ("OK  Cancel");
// Qt 3 labels, and so KDE 3 form labels, were left-aligned unless a dialog
// said otherwise.
func (keramikEngine) StyleHint(l *Classic, h StyleHint) int {
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

// ---- colours ------------------------------------------------------------------------------

type keramik struct {
	bg, btn, base, text, btnText, dis, sel, selText, alt paintengine2d.Color
	light, dark, mid, menuBg, focus                      paintengine2d.Color
	tip, tipText                                         paintengine2d.Color

	// Gel bodies over the button colour: idle, hot, held, disabled.
	gel, gelHot, gelDown, gelDis    []paintengine2d.GradientStop
	ring, ringTop, ringBot, ringDis paintengine2d.Color
	caps                            []paintengine2d.GradientStop // the darker rounded ends
	shadow1, shadow2                paintengine2d.Color
	defRing                         []paintengine2d.GradientStop // the default button's sunken ring

	// The selection gel: menu rows, scroll handles, progress bars.
	selGel, selHot, selDown []paintengine2d.GradientStop
	selRing, grip, gripLt   paintengine2d.Color

	// Grooves (scroll bars, sliders, progress) and fields.
	groove     []paintengine2d.GradientStop
	grooveRing paintengine2d.Color
	fieldRing  []paintengine2d.GradientStop
	fieldShade paintengine2d.Color

	// Tabs and header sections.
	tabGel, tabHot, tabSel []paintengine2d.GradientStop
	tabRing, tabTop        paintengine2d.Color

	// The window decoration: caption gels, their frame and text.
	cap, capOff, bubble, bubbleOff []paintengine2d.GradientStop
	capBase, capOffBase            paintengine2d.Color
	capLine, capOffLine            paintengine2d.Color
	capText, capOffText            paintengine2d.Color
	capShadow, capOffShadow        paintengine2d.Color
	btnGlyph                       paintengine2d.Color

	// Small fixed ramps: the bead's gloss, the close button's faces, the
	// progress groove, a menu bar title under the pointer.
	gloss, closeIdle, closeHot, closePress []paintengine2d.GradientStop
	progGroove, titleHot                   []paintengine2d.GradientStop
}

type keramikKey struct{}

func keramikColors(l *Classic) *keramik {
	return l.Memo(keramikKey{}, func() any { return keramikBuild(l) }).(*keramik)
}

// kmGelStops is the gel ramp over c, top to bottom. Fitted to the KDE 3.1
// push button and menu highlight: it lands within a few levels of both.
func kmGelStops(c paintengine2d.Color) []paintengine2d.GradientStop {
	white := paintengine2d.RGB(1, 1, 1)
	lo := Mix(c, white, 0.2)
	return []paintengine2d.GradientStop{
		Stop(0, Mix(c, white, 0.55)),
		Stop(0.1, Mix(c, white, 0.48)),
		Stop(0.3, Mix(c, white, 0.32)),
		Stop(0.5, darkerPct(lo, 104)),
		Stop(0.68, darkerPct(lo, 112)),
		Stop(0.86, darkerPct(lo, 108)),
		Stop(1, darkerPct(lo, 105)),
	}
}

// kmButtonStops is the push button's gel over c, top to bottom: a dark
// hairline under the light top edge, the brightest band a sixth of the way
// down, a quick fall through the middle to a flat lower half and a lighter
// line just above the bottom edge. Fitted to the KDE 3.1 push button's
// measured column; the tabs share it.
func kmButtonStops(c paintengine2d.Color) []paintengine2d.GradientStop {
	white := paintengine2d.RGB(1, 1, 1)
	return []paintengine2d.GradientStop{
		Stop(0, darkerPct(c, 103.5)),
		Stop(0.05, Mix(c, white, 0.52)),
		Stop(0.14, Mix(c, white, 0.84)),
		Stop(0.24, Mix(c, white, 0.76)),
		Stop(0.33, Mix(c, white, 0.36)),
		Stop(0.43, Mix(c, white, 0.08)),
		Stop(0.52, darkerPct(c, 103.5)),
		Stop(0.62, darkerPct(c, 106)),
		Stop(0.72, darkerPct(c, 108)),
		Stop(0.88, darkerPct(c, 108)),
		Stop(0.95, darkerPct(c, 105.5)),
		Stop(1, darkerPct(c, 109)),
	}
}

// kmSunkStops is a held button: its ramp over a darker colour, turned over.
func kmSunkStops(c paintengine2d.Color) []paintengine2d.GradientStop {
	g := kmButtonStops(darkerPct(c, 110))
	out := make([]paintengine2d.GradientStop, len(g))
	for i, s := range g {
		out[len(g)-1-i] = Stop(1-s.Offset, s.Color)
	}
	return out
}

func keramikBuild(l *Classic) *keramik {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	c := &keramik{
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
	c.alt = l.X("alternate", Mix(c.base, c.sel, 0.08))
	c.light = lighterPct(c.bg, 150)
	c.dark = Mix(c.bg, black, 0.645) // the 3D frame's dark line, (83,83,82) on #eae9e8
	c.mid = darkerPct(c.bg, 135)
	c.menuBg = lighterPct(c.bg, 105)
	c.focus = c.text

	// Buttons: the outline is grey along the sides (the button colour at
	// 1/1.7 of its value), light along the top, almost black at the bottom.
	c.gel = kmButtonStops(c.btn)
	c.gelHot = kmButtonStops(Mix(c.btn, white, 0.45))
	c.gelDown = kmSunkStops(c.btn)
	flat := Mix(c.btn, c.bg, 0.6)
	c.gelDis = []paintengine2d.GradientStop{Stop(0, Mix(flat, white, 0.4)), Stop(1, flat)}
	c.ring = darkerPct(c.btn, 170)
	c.ringTop = Mix(c.btn, white, 0.7)
	c.ringBot = Mix(c.btn, black, 0.82)
	c.ringDis = Mix(c.ring, c.bg, 0.55)
	// The ends darken over six pixels, deepest one and a half pixels in.
	c.caps = []paintengine2d.GradientStop{
		Stop(0, black.WithAlpha(0.11)), Stop(0.25, black.WithAlpha(0.21)), Stop(0.42, black.WithAlpha(0.17)),
		Stop(0.58, black.WithAlpha(0.09)), Stop(0.75, black.WithAlpha(0.05)), Stop(1, black.WithAlpha(0)),
	}
	c.shadow1 = black.WithAlpha(0.16)
	c.shadow2 = black.WithAlpha(0.08)
	c.defRing = []paintengine2d.GradientStop{
		Stop(0, Mix(c.bg, black, 0.75)), Stop(0.18, Mix(c.bg, black, 0.62)),
		Stop(0.5, darkerPct(c.btn, 172)), Stop(0.82, Mix(c.bg, white, 0.55)), Stop(1, white),
	}

	c.selGel = kmGelStops(c.sel)
	c.selHot = kmGelStops(Mix(c.sel, white, 0.3))
	c.selDown = kmGelStops(darkerPct(c.sel, 112))
	c.selRing = darkerPct(c.sel, 130)
	c.grip = darkerPct(c.sel, 150)
	c.gripLt = Mix(c.sel, white, 0.6)
	if l.P("sbHighlight", 1) == 0 {
		c.grip, c.gripLt = darkerPct(c.btn, 150), Mix(c.btn, white, 0.6)
	}

	// Grooves shade across: the shadow side darker, the far side light.
	c.groove = []paintengine2d.GradientStop{
		Stop(0, darkerPct(c.bg, 118)), Stop(0.3, darkerPct(c.bg, 106)), Stop(1, lighterPct(c.bg, 106)),
	}
	c.grooveRing = darkerPct(c.bg, 145)
	c.fieldRing = []paintengine2d.GradientStop{
		Stop(0, darkerPct(c.bg, 190)), Stop(0.5, darkerPct(c.bg, 150)), Stop(1, lighterPct(c.bg, 110)),
	}
	c.fieldShade = Mix(c.base, c.dark, 0.3)

	c.tabGel = kmButtonStops(darkerPct(c.btn, 104))
	c.tabHot = kmButtonStops(c.btn)
	c.tabSel = kmButtonStops(Mix(c.btn, white, 0.3))
	c.tabRing = darkerPct(c.btn, 165)
	c.tabTop = Mix(c.btn, black, 0.66)

	// The decoration: a gel over the caption colour, the frame in the
	// caption colour itself, a darker outline; the bubble a touch lighter.
	capA := l.X("caption", Hex("#3e91eb"))
	capB := l.X("caption2", capA)
	offA := l.X("captionOff", Hex("#afd6ff"))
	offB := l.X("captionOff2", offA)
	c.capBase, c.capOffBase = Mix(capA, capB, 0.5), Mix(offA, offB, 0.5)
	c.cap, c.capOff = kmGelStops(c.capBase), kmGelStops(c.capOffBase)
	c.bubble, c.bubbleOff = kmGelStops(Mix(capA, white, 0.12)), kmGelStops(Mix(offA, white, 0.12))
	c.capLine, c.capOffLine = darkerPct(c.capBase, 170), darkerPct(c.capOffBase, 150)
	c.capText = l.X("captionText", white)
	c.capOffText = l.X("captionOffText", white)
	c.capShadow = kde3TitleShadow(c.capText, c.capBase)
	c.capOffShadow = kde3TitleShadow(c.capOffText, c.capOffBase)
	c.btnGlyph = Mix(c.text, white, 0.1)

	wa := func(top, bot float32) []paintengine2d.GradientStop {
		return []paintengine2d.GradientStop{Stop(0, white.WithAlpha(top)), Stop(1, white.WithAlpha(bot))}
	}
	c.gloss = wa(0.85, 0.1)
	c.closeIdle, c.closeHot, c.closePress = wa(0.85, 0.45), wa(1, 0.7), wa(0.35, 0.6)
	c.progGroove = []paintengine2d.GradientStop{
		Stop(0, darkerPct(c.bg, 115)), Stop(0.4, darkerPct(c.bg, 104)), Stop(1, lighterPct(c.bg, 105)),
	}
	c.titleHot = kmGelStops(Mix(c.sel, c.bg, 0.55))
	return c
}

// ---- painting helpers -------------------------------------------------------------------------

// kmLinear is a linear gradient paint from a to b.
func kmLinear(ax, ay, bx, by float32, stops []paintengine2d.GradientStop) paintengine2d.Paint {
	return paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(ax, ay), End: paintengine2d.Pt(bx, by), Stops: stops,
	})
}

// paintGel paints a gel body into b with corner radius r: the outline ring, its
// light leading edge and dark trailing edge, the ramp inside and, with caps,
// the darker rounded ends. across turns the body sideways (the ramp runs
// left to right, for vertical scroll bars).
func (c *keramik) paintGel(ctx *paintengine2d.Context, b paintengine2d.Rect, r, u float32, fill []paintengine2d.GradientStop,
	ring, top, bot paintengine2d.Color, across, caps bool) {
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	r = max(min(r, b.Dx()*0.5, b.Dy()*0.5), 0)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(ring))
	if across {
		if h := b.Dy() - 2*r; h > 0 {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+r, u, h), paintengine2d.Fill(top))
			ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+r, u, h), paintengine2d.Fill(bot))
		}
	} else if w := b.Dx() - 2*r; w > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+r, b.Min.Y, w, u), paintengine2d.Fill(top))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+r, b.Max.Y-u, w, u), paintengine2d.Fill(bot))
	}
	in := b.Inset(u)
	ri := max(r-u, 0)
	paint := VGradient(in, fill...)
	if across {
		paint = HGradient(in, fill...)
	}
	ctx.DrawRoundRect(in, ri, ri, paint)
	if !caps {
		return
	}
	// Each end is the inner shape again, clipped to a strip, in the
	// shading ramp: it follows the rounded corners.
	cap := func(strip paintengine2d.Rect, paint paintengine2d.Paint) {
		ctx.Save()
		ctx.ClipRect(strip)
		ctx.DrawRoundRect(in, ri, ri, paint)
		ctx.Restore()
	}
	if across {
		w := min(6*u, in.Dy()*0.25)
		cap(paintengine2d.XYWH(in.Min.X, in.Min.Y, in.Dx(), w), kmLinear(in.Min.X, in.Min.Y, in.Min.X, in.Min.Y+w, c.caps))
		cap(paintengine2d.XYWH(in.Min.X, in.Max.Y-w, in.Dx(), w), kmLinear(in.Min.X, in.Max.Y, in.Min.X, in.Max.Y-w, c.caps))
	} else {
		w := min(6*u, in.Dx()*0.25)
		cap(paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy()), kmLinear(in.Min.X, in.Min.Y, in.Min.X+w, in.Min.Y, c.caps))
		cap(paintengine2d.XYWH(in.Max.X-w, in.Min.Y, w, in.Dy()), kmLinear(in.Max.X, in.Min.Y, in.Max.X-w, in.Min.Y, c.caps))
	}
}

// gelFor picks the button gel for a state.
func (c *keramik) gelFor(st ControlState) []paintengine2d.GradientStop {
	switch {
	case st.Disabled():
		return c.gelDis
	case st.Pressed() || (st.Toggle() && st.Checked()):
		return c.gelDown
	case st.Hovered():
		return c.gelHot
	}
	return c.gel
}

// button paints a push button into b and returns its face: b less the
// drop shadow, or inside the default button's sunken ring.
func (c *keramik) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Rect {
	u := kde3U(l)
	if b.Dx() < 8*u || b.Dy() < 8*u {
		return b
	}
	dis := st.Disabled()
	r := l.rx(5)
	body := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-2*u, b.Dy()-2*u)
	switch {
	case st.Primary() && !dis:
		rr := min(r+2*u, b.Dy()*0.5)
		if r == 0 {
			rr = 0
		}
		ctx.DrawRoundRect(b, rr, rr, VGradient(b, c.defRing...))
		body = b.Inset(3 * u)
	case !dis:
		ctx.DrawRoundRect(body.Translate(paintengine2d.Pt(2*u, 2*u)), r, r, paintengine2d.Fill(c.shadow2))
		ctx.DrawRoundRect(body.Translate(paintengine2d.Pt(u, u)), r, r, paintengine2d.Fill(c.shadow1))
	}
	ring, top, bot := c.ring, c.ringTop, c.ringBot
	if dis {
		ring, top, bot = c.ringDis, c.ringDis, c.ringDis
	}
	c.paintGel(ctx, body, r, u, c.gelFor(st), ring, top, bot, false, !dis)
	return body
}

// field is a text field and a view well: the base colour in a sunken,
// slightly rounded rim (dark along the top, light along the bottom) with a
// shadow line under the top and left edges.
func (c *keramik) field(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	u := kde3U(l)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	r := max(min(l.rx(3), b.Dy()*0.5), 0)
	ctx.DrawRoundRect(b, r, r, VGradient(b, c.fieldRing...))
	in := b.Inset(u)
	ri := max(r-u, 0)
	fill := c.base
	if st.Disabled() {
		fill = c.bg
	}
	ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(fill))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X+ri, in.Min.Y, in.Dx()-2*ri, u), paintengine2d.Fill(c.fieldShade))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+max(ri, u), u, in.Dy()-2*max(ri, u)), paintengine2d.Fill(c.fieldShade))
}

// grooveBody is a sunken groove: rounded at the ends that do not meet a
// step button (open), shaded across.
func (c *keramik) grooveBody(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical, openStart, openEnd bool) {
	u := kde3U(l)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return
	}
	across := b.Dx()
	if !vertical {
		across = b.Dy()
	}
	r := across * 0.5
	if l.square() {
		r = 0
	}
	in := b.Inset(u)
	paint := VGradient(in, c.groove...)
	if vertical {
		paint = HGradient(in, c.groove...)
	}
	// Round the ends that are open: top / left is the start.
	top, right, bottom, left := true, true, true, true
	if vertical {
		top, bottom = openStart, openEnd
	} else {
		left, right = openStart, openEnd
	}
	if !openStart && !openEnd {
		r = 0
	}
	kde3Rounded(ctx, b, r, top, right, bottom, left, paintengine2d.Fill(c.grooveRing))
	kde3Rounded(ctx, in, max(r-u, 0), top, right, bottom, left, paint)
}

// kmBlockArrow fills Keramik's scroll arrow: a triangular head on a short
// stem, w across, pointing dir, centred in b.
func kmBlockArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, w float32, col paintengine2d.Color) {
	if b.Empty() || w < 3 || col.A <= 0 {
		return
	}
	w = min(w, b.Dx(), b.Dy())
	head := w * 0.5
	stemW := w * 0.5
	stem := w * 0.42
	total := head + stem
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	// Build pointing up about the origin, then turn.
	pts := [7][2]float32{
		{0, -total * 0.5},
		{w * 0.5, -total*0.5 + head},
		{stemW * 0.5, -total*0.5 + head},
		{stemW * 0.5, total * 0.5},
		{-stemW * 0.5, total * 0.5},
		{-stemW * 0.5, -total*0.5 + head},
		{-w * 0.5, -total*0.5 + head},
	}
	p := kde3Path()
	defer kde3Done(p)
	for i, q := range pts {
		x, y := q[0], q[1]
		switch dir {
		case DirDown:
			x, y = -x, -y
		case DirLeft:
			x, y = y, -x
		case DirRight:
			x, y = -y, x
		}
		if i == 0 {
			p.MoveTo(snapHalf(cx+x), snapHalf(cy+y))
		} else {
			p.LineTo(snapHalf(cx+x), snapHalf(cy+y))
		}
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// snapHalf rounds to the nearest half pixel, so small glyph edges land on
// pixel boundaries or centres and stay crisp.
func snapHalf(v float32) float32 { return float32(math.Round(float64(v)*2)) * 0.5 }

// kde3GripLines paints n short engraved lines (dark with a light line beside
// them) across b's centre: vertical lines side by side when vertical.
func kde3GripLines(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, n int, gap, u, length float32, dark, light paintengine2d.Color) {
	if n <= 0 || b.Empty() {
		return
	}
	d, lt := kde3Path(), kde3Path()
	defer kde3Done(d)
	defer kde3Done(lt)
	span := float32(n-1) * gap
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5 - span*0.5 - u*0.5)
		y := snap((b.Min.Y+b.Max.Y)*0.5 - length*0.5)
		for i := 0; i < n; i++ {
			xx := x + float32(i)*gap
			d.AddRect(paintengine2d.XYWH(xx, y, u, length))
			lt.AddRect(paintengine2d.XYWH(xx+u, y, u, length))
		}
	} else {
		y := snap((b.Min.Y+b.Max.Y)*0.5 - span*0.5 - u*0.5)
		x := snap((b.Min.X+b.Max.X)*0.5 - length*0.5)
		for i := 0; i < n; i++ {
			yy := y + float32(i)*gap
			d.AddRect(paintengine2d.XYWH(x, yy, length, u))
			lt.AddRect(paintengine2d.XYWH(x, yy+u, length, u))
		}
	}
	ctx.DrawPath(lt, paintengine2d.Fill(light))
	ctx.DrawPath(d, paintengine2d.Fill(dark))
}

// ---- parts --------------------------------------------------------------------------------------

func (e keramikEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := keramikColors(l)
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
			c.toolFace(l, ctx, b, st)
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
		c.handle(l, ctx, b, b.Dy() >= b.Dx(), st)
	case RoleTrack:
		v := b.Dy() >= b.Dx()
		c.grooveBody(l, ctx, b, v, true, true)
	case RoleBar, RolePanel, RoleSplitter:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	}
	return fg
}

// toolFace is a tool button that is hot, held or latched: a gel in the
// button's own rect, without the push button's shadow.
func (c *keramik) toolFace(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	u := kde3U(l)
	c.paintGel(ctx, b, l.rx(4), u, c.gelFor(st&^StateDisabled), c.ring, c.ringTop, c.ringBot, false, true)
}

// row paints an item view row's background for st and returns its text
// colour: the selection (kept when the view loses focus, as KDE 3 did) or
// the scheme's alternate background on odd rows.
func (c *keramik) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
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

// CheckIndicator is a small square gel with Keramik's bold tick.
func (keramikEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b := kde3Snap(box)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	ring, top, bot := c.ring, c.ringTop, c.ringBot
	if st.Disabled() {
		ring, top, bot = c.ringDis, c.ringDis, c.ringDis
	}
	c.paintGel(ctx, b, min(l.rx(3), b.Dx()*0.25), u, c.gelFor(st&^StateToggle), ring, top, bot, false, false)
	if !checked {
		return
	}
	col := c.text
	if st.Disabled() {
		col = c.dis
	}
	w := max(1.6*u, b.Dx()*0.15)
	ctx.Save()
	ctx.ClipRect(b.Inset(u))
	if !st.Disabled() {
		kmTick(ctx, b, c.light.WithAlpha(0.7), w, u) // the embossed light edge under the tick
	}
	kmTick(ctx, b, col, w, 0)
	ctx.Restore()
}

// kmTick strokes Keramik's bold tick into the square b, dy lower: a short
// arm down to the vertex, a long arm up to the right, round ends.
func kmTick(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w, dy float32) {
	s := min(b.Dx(), b.Dy())
	at := func(fx, fy float32) (float32, float32) { return b.Min.X + fx*s, b.Min.Y + fy*s + dy }
	p := kde3Path()
	defer kde3Done(p)
	p.MoveTo(at(0.24, 0.5))
	p.LineTo(at(0.43, 0.72))
	p.LineTo(at(0.78, 0.25))
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// RadioIndicator is a glossy bead with a dark dot.
func (keramikEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := keramikColors(l)
	u := kde3U(l)
	side := min(box.Dx(), box.Dy())
	if side < 6*u {
		return
	}
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5
	r := side * 0.5
	ring := c.ring
	if st.Disabled() {
		ring = c.ringDis
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(ring))
	// The dark lower rim, as the push button's bottom edge.
	if !st.Disabled() {
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(cx-r, cy+r*0.35, 2*r, r))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(Mix(ring, c.ringBot, 0.55)))
		ctx.Restore()
	}
	in := r - u
	bb := paintengine2d.XYWH(cx-in, cy-in, 2*in, 2*in)
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), in, VGradient(bb, c.gelFor(st&^StateToggle)...))
	if !st.Disabled() {
		// The bead's gloss: a white ellipse fading down over the top half.
		gl := paintengine2d.XYWH(cx-in*0.62, cy-in*0.86, in*1.24, in*0.8)
		ctx.DrawOval(gl, VGradient(gl, c.gloss...))
	}
	if !selected {
		return
	}
	col := c.text
	if st.Disabled() {
		col = c.dis
	}
	dr := max(r*0.34, 2*u)
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), dr, paintengine2d.Fill(col))
	if !st.Disabled() {
		ctx.DrawCircle(paintengine2d.Pt(cx-dr*0.35, cy-dr*0.35), dr*0.32, paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.55)))
	}
}

// Arrow is a small solid triangle (menu submenus, sort marks); scroll bars,
// spin boxes and combos use the block arrow.
func (keramikEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	g := min(s*0.9, l.S(9))
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	FillArrow(ctx, paintengine2d.XYWH(cx-g*0.5, cy-g*0.5, g, g), dir, col)
}

// Expander is KDE 3's list view branch box: a small square, a grey frame,
// a dark plus or minus.
func (keramikEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := keramikColors(l)
	kde3PlusBox(l, ctx, b, expanded, c.base, c.mid, c.text)
}

// MenuHighlight is the hot menu row: the selection gel between two darker
// lines, square, across the whole row.
func (keramikEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 2*u || b.Dy() < 3*u {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.selRing))
	in := paintengine2d.XYWH(b.Min.X, b.Min.Y+u, b.Dx(), b.Dy()-2*u)
	if attachBottom {
		in.Max.Y = b.Max.Y
	}
	ctx.DrawRect(in, VGradient(in, c.selGel...))
}

func (keramikEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := keramikColors(l)
	if hot {
		return c.selText
	}
	return c.text
}

// Fields show only the caret when focused.
func (keramikEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is Qt 3's dotted focus rectangle, inside b.
func (keramikEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	kde3Dotted(ctx, b, kde3U(l), keramikColors(l).focus)
}

// ItemFocus is the dotted rectangle around the current item, in the colour
// that reads on the row.
func (keramikEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := keramikColors(l)
	col := c.text
	if st.Checked() && !st.Disabled() {
		col = c.selText
	}
	kde3Dotted(ctx, b, kde3U(l), col)
}

// PopupShadow: KDE 3.1's menus cast no shadow; a pack's "shadow" param
// turns on the later menu drop shadow.
func (keramikEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	if l.P("shadow", 0) <= 0 {
		return Insets{}
	}
	return BaseEngine{}.PopupShadow(l, kind)
}

func (keramikEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	if l.P("shadow", 0) > 0 {
		BaseEngine{}.DrawPopupShadow(l, ctx, b, kind)
	}
}

// ---- scroll bars ---------------------------------------------------------------------------------

// ScrollBarStyle: 17px bars, both step buttons together at the far end.
func (keramikEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 17, Arrows: ArrowsTogetherEnd, MinThumb: 26}
}

// handle is the scroll handle: a rounded gel in the selection colour (the
// button colour when the pack turns the highlight off) with three grip
// lines across its middle.
func (c *keramik) handle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 5*u || b.Dy() < 5*u {
		return
	}
	across, along := b.Dx(), b.Dy()
	if !vertical {
		across, along = b.Dy(), b.Dx()
	}
	fill, ring := c.selGel, c.selRing
	switch {
	case st.Pressed():
		fill = c.selDown
	case st.Hovered():
		fill = c.selHot
	}
	if l.P("sbHighlight", 1) == 0 {
		fill, ring = c.gelFor(st), c.ring
	}
	top, bot := Mix(ring, paintengine2d.RGB(1, 1, 1), 0.55), darkerPct(ring, 160)
	c.paintGel(ctx, b, min(l.rx(6), across*0.5), u, fill, ring, top, bot, vertical, false)
	if along >= 18*u {
		kde3GripLines(ctx, b, !vertical, 3, 3*u, u, snap(across*0.42), c.grip, c.gripLt)
	}
}

// stepButton paints one scroll step button: a gel rounded on the side that
// ends the bar, with the block arrow.
func (c *keramik) stepButton(l *Classic, ctx *paintengine2d.Context, b, bar paintengine2d.Rect, vertical, inc bool, st ControlState) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 5*u || b.Dy() < 5*u {
		return
	}
	across := b.Dx()
	if !vertical {
		across = b.Dy()
	}
	r := min(l.rx(6), across*0.5)
	// Rounded on the side that ends the bar, square where it meets the
	// groove or its twin.
	atStart, atEnd := b.Min.Y <= bar.Min.Y+0.5, b.Max.Y >= bar.Max.Y-0.5
	top, right, bottom, left := atStart, true, atEnd, true
	if !vertical {
		atStart, atEnd = b.Min.X <= bar.Min.X+0.5, b.Max.X >= bar.Max.X-0.5
		top, right, bottom, left = true, atEnd, true, atStart
	}
	if !atStart && !atEnd {
		r = 0
	}
	ring := c.ring
	if st.Disabled() {
		ring = c.ringDis
	}
	in := b.Inset(u)
	fill := c.gelFor(st)
	paint := VGradient(in, fill...)
	if vertical {
		paint = HGradient(in, fill...)
	}
	kde3Rounded(ctx, b, r, top, right, bottom, left, paintengine2d.Fill(ring))
	kde3Rounded(ctx, in, max(r-u, 0), top, right, bottom, left, paint)
	dir := DirUp
	switch {
	case vertical && inc:
		dir = DirDown
	case !vertical && !inc:
		dir = DirLeft
	case !vertical && inc:
		dir = DirRight
	}
	col := c.text
	if st.Disabled() {
		col = c.dis
	}
	g := in
	if st.Pressed() {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	kmBlockArrow(ctx, g, dir, snap(across*0.52), col)
}

func (e keramikEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := keramikColors(l)
	if p.Bar.Empty() {
		return
	}
	bar := kde3Snap(p.Bar)
	ctx.DrawRect(bar, paintengine2d.Fill(c.bg))
	if !p.Track.Empty() {
		tr := kde3Snap(p.Track)
		touches := func(btn paintengine2d.Rect, atStart bool) bool {
			if btn.Empty() {
				return false
			}
			if vertical {
				if atStart {
					return math.Abs(float64(btn.Max.Y-tr.Min.Y)) < 1.5
				}
				return math.Abs(float64(btn.Min.Y-tr.Max.Y)) < 1.5
			}
			if atStart {
				return math.Abs(float64(btn.Max.X-tr.Min.X)) < 1.5
			}
			return math.Abs(float64(btn.Min.X-tr.Max.X)) < 1.5
		}
		openStart := !touches(p.Dec, true) && !touches(p.Inc, true)
		openEnd := !touches(p.Dec, false) && !touches(p.Inc, false)
		c.grooveBody(l, ctx, tr, vertical, openStart, openEnd)
	}
	if !p.Dec.Empty() {
		c.stepButton(l, ctx, p.Dec, bar, vertical, false, st.Part(ScrollDec))
	}
	if !p.Inc.Empty() {
		c.stepButton(l, ctx, p.Inc, bar, vertical, true, st.Part(ScrollInc))
	}
	if p.Thumb.Empty() || st.Disabled {
		return
	}
	hs := StateNone
	if st.Hot == ScrollThumbPart {
		hs |= StateHovered
	}
	if st.Pressed == ScrollThumbPart {
		hs |= StatePressed
	}
	c.handle(l, ctx, p.Thumb, vertical, hs)
}

// DrawScrollBar paints a bare groove and handle (widgets that lay out their
// own bar).
func (e keramikEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
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

func (keramikEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad + l.S(2)
	if hasTitle {
		top = l.body.Height() + pad*0.6
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox: Qt 3's group box — an etched frame, here with Keramik's
// rounded corners, the title breaking its top edge at the left.
func (keramikEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	// The window colour behind the title too: a titled panel may float
	// over something else (an overlay's dialog).
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	fill := c.bg
	if raised {
		fill = lighterPct(c.bg, 103)
	}
	top := b.Min.Y
	f := l.body
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	fr := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if fr.Dx() < 6*u || fr.Dy() < 6*u {
		return
	}
	r := l.rx(4)
	ctx.DrawRoundRect(fr, r, r, paintengine2d.Fill(c.light))
	dk := paintengine2d.XYWH(fr.Min.X, fr.Min.Y, fr.Dx()-u, fr.Dy()-u)
	ctx.DrawRoundRect(dk, r, r, paintengine2d.Fill(c.mid))
	in := paintengine2d.XYWH(fr.Min.X+u, fr.Min.Y+u, fr.Dx()-3*u, fr.Dy()-3*u)
	ri := max(r-u, 0)
	ctx.DrawRoundRect(in, ri, ri, paintengine2d.Fill(c.light))
	ctx.DrawRoundRect(paintengine2d.XYWH(in.Min.X+u, in.Min.Y+u, in.Dx()-u, in.Dy()-u), ri, ri, paintengine2d.Fill(fill))
	if title == "" {
		return
	}
	tx := b.Min.X + l.S(10)
	tw := min(f.Advance(title)+l.S(6), b.Max.X-l.S(4)-tx)
	if tw <= 0 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(tx, b.Min.Y, tw, f.Height()), paintengine2d.Fill(c.bg))
	l.drawFittedText(ctx, f, title, paintengine2d.XYWH(tx+l.S(3), b.Min.Y, tw-l.S(3), f.Height()), c.text, AlignStart, 0)
}

// kmCaptionH is the decoration's title bar height at the UI font; kmRise is
// how far the active window's title bubble rises above it.
func kmCaptionH(l *Classic) float32 {
	return max(snap(l.body.Height()+l.S(8)), snap(l.S(22)))
}

func kmRise(l *Classic) float32 { return snap(l.S(4)) }

func (keramikEngine) WindowFrameInsets(l *Classic) Insets {
	fr := snap(l.S(3))
	return Insets{Top: kmRise(l) + kmCaptionH(l), Right: fr, Bottom: fr + kde3U(l), Left: fr}
}

// kmBar is the title bar of a decorated frame b.
func kmBar(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.XYWH(b.Min.X, b.Min.Y+kmRise(l), b.Dx(), kmCaptionH(l))
}

// WindowCloseRect is the close button at the right of the title bar.
func (keramikEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = kde3Snap(b)
	bar := kmBar(l, b)
	if b.Dy() < bar.Max.Y-b.Min.Y || b.Dx() < bar.Dy()*3 {
		return paintengine2d.Rect{}
	}
	side := snap(bar.Dy() - l.S(10))
	if side < snap(l.S(12)) {
		side = snap(bar.Dy() * 0.6)
	}
	return paintengine2d.XYWH(snap(bar.Max.X-side-l.S(7)), snap(bar.Min.Y+(bar.Dy()-side)*0.5), side, side)
}

// DrawWindowFrame is the Keramik window decoration: a frame in the caption
// colour around the body, the gel title bar with rounded top corners and,
// on the active window, the title in a bubble rising above the bar.
func (e keramikEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	bar := kmBar(l, b)
	in := e.WindowFrameInsets(l)
	if b.Dx() < in.Left+in.Right+bar.Dy()*2 || b.Dy() < in.Top+in.Bottom+u {
		return
	}
	base, stops, line, bubble := c.capBase, c.cap, c.capLine, c.bubble
	fg, shadow := c.capText, c.capShadow
	if !st.Active {
		base, stops, line, bubble = c.capOffBase, c.capOff, c.capOffLine, c.bubbleOff
		fg, shadow = c.capOffText, c.capOffShadow
	}
	r := l.rx(7)
	frame := paintengine2d.XYWH(b.Min.X, bar.Min.Y, b.Dx(), b.Max.Y-bar.Min.Y)
	kde3Rounded(ctx, frame, r, true, true, false, true, paintengine2d.Fill(line))
	ri := max(r-u, 0)
	kde3Rounded(ctx, frame.Inset(u), ri, true, true, false, true, paintengine2d.Fill(base))
	kde3Rounded(ctx, paintengine2d.XYWH(bar.Min.X+u, bar.Min.Y+u, bar.Dx()-2*u, bar.Dy()-u), ri, true, true, false, true,
		VGradient(bar, stops...))
	body := in.Apply(b)
	ctx.DrawRect(body, paintengine2d.Fill(c.bg))
	// The client edge: a dark line above and left, light below and right.
	ctx.DrawRect(paintengine2d.XYWH(body.Min.X-u, body.Min.Y-u, body.Dx()+u, u), paintengine2d.Fill(line))

	// The title bubble: a pill from the left edge of the bar, as wide as
	// the title, rising above the bar on the active window.
	f := l.bold
	if f == nil {
		f = l.body
	}
	cb := e.WindowCloseRect(l, b)
	maxX := bar.Max.X - bar.Dy()*1.6
	bx := bar.Min.X + l.S(8)
	bw := min(f.Advance(title)+l.S(28), maxX-bx)
	if bw > l.S(20) {
		top := b.Min.Y
		if !st.Active {
			top = bar.Min.Y + u
		}
		bb := paintengine2d.XYWH(snap(bx), top, snap(bw), bar.Max.Y-l.S(3)-top)
		br := bb.Dy() * 0.5
		if l.square() {
			br = 0
		}
		ctx.DrawRoundRect(bb, br, br, paintengine2d.Fill(Mix(base, paintengine2d.RGB(1, 1, 1), 0.7)))
		bi := bb.Inset(u)
		ctx.DrawRoundRect(bi, max(br-u, 0), max(br-u, 0), VGradient(bi, bubble...))
		if title != "" {
			tb := paintengine2d.XYWH(bi.Min.X+l.S(12), bi.Min.Y, bi.Dx()-l.S(16), bi.Dy())
			if shadow.A > 0 {
				l.drawFittedText(ctx, f, title, tb.Translate(paintengine2d.Pt(u, u)), shadow, AlignStart, 0)
			}
			l.drawFittedText(ctx, f, title, tb, fg, AlignStart, 0)
		}
	}
	if st.CanClose && !cb.Empty() {
		c.closeButton(l, ctx, cb, st)
	}
}

// closeButton is the decoration's close button: a small rounded light gel
// with a dark cross.
func (c *keramik) closeButton(l *Classic, ctx *paintengine2d.Context, cb paintengine2d.Rect, st WindowState) {
	u := kde3U(l)
	r := min(l.rx(3), cb.Dx()*0.3)
	face := c.closeIdle
	switch {
	case st.ClosePress:
		face = c.closePress
	case st.CloseHot:
		face = c.closeHot
	}
	ctx.DrawRoundRect(cb, r, r, paintengine2d.Fill(c.capLine.WithAlpha(0.9)))
	in := cb.Inset(u)
	ctx.DrawRoundRect(in, max(r-u, 0), max(r-u, 0), VGradient(in, face...))
	g := in.Inset(in.Dx() * 0.27)
	if st.ClosePress {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	DrawCross(ctx, g, c.btnGlyph, max(l.S(1.8), u))
}

// TabOutset: Keramik tabs stand apart; the selected one grows upwards only.
func (keramikEngine) TabOutset(l *Classic) Insets { return Insets{} }

// kmPaneTop is where a tab pane's raised frame begins: the tab bar's last
// row, which the selected tab opens.
func kmPaneTop(l *Classic, b paintengine2d.Rect) float32 {
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		return b.Min.Y
	}
	return snap(b.Min.Y + th - kde3U(l))
}

// DrawTabPane is the page under the tabs: the window colour in a raised
// one-pixel frame (light above and left, dark below and right).
func (keramikEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	top := kmPaneTop(l, b)
	pane := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if pane.Dx() < 3*u || pane.Dy() < 3*u {
		return
	}
	kde3Frame(ctx, pane, u, c.bg, c.light, c.dark)
}

// ToolBarInsets keeps room for the tool bar's handle, three engraved lines.
func (keramikEngine) ToolBarInsets(l *Classic) Insets {
	return Insets{Left: snap(l.S(14)), Right: snap(l.S(6))}
}

// ControlFont: tool buttons are labelled in KDE 3's smaller tool bar font.
func (keramikEngine) ControlFont(l *Classic, role Role) *Font { return kde3ControlFont(l, role) }

func (keramikEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	kde3ToolButton(l, ctx, b, st, label, icon, keramikColors(l).dis)
}

// ---- controls -----------------------------------------------------------------------------------

func (e keramikEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	body := c.button(l, ctx, b, st)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	lb := body
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, body.Inset(snap(l.S(4))), u, c.focus)
	}
}

// toggle draws a check box or radio caption and, with focus, the dotted
// rectangle around it.
func (keramikEngine) toggle(l *Classic, ctx *paintengine2d.Context, b, box paintengine2d.Rect, st ControlState, label string) {
	c := keramikColors(l)
	u := kde3U(l)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			kde3Dotted(ctx, box.Inset(-u).Intersect(b), u, c.focus)
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
		kde3Dotted(ctx, labelFocusRect(l.body, label, lb, b), u, c.focus)
	}
}

func (e keramikEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	side := min(l.metrics.Checkbox, b.Dy())
	box := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.CheckIndicator(l, ctx, box, st, checked)
	e.toggle(l, ctx, b, box, st, label)
}

func (e keramikEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	side = min(side, b.Dy())
	box := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	e.RadioIndicator(l, ctx, box, st, selected)
	e.toggle(l, ctx, b, box, st, label)
}

// DrawSwitch: KDE 3 had no switch; this one is a groove with a gel knob
// that turns the selection colour when on.
func (e keramikEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := keramikColors(l)
	u := kde3U(l)
	m := l.metrics
	tw, th := min(m.SwitchW, b.Dx()), min(m.SwitchH, b.Dy())
	track := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 8*u || track.Dy() < 6*u {
		return
	}
	g := kde3Snap(paintengine2d.XYWH(track.Min.X, track.Min.Y+track.Dy()*0.2, track.Dx(), track.Dy()*0.6))
	c.grooveBody(l, ctx, g, false, true, true)
	if on && !st.Disabled() {
		in := g.Inset(u)
		ctx.DrawRoundRect(in, in.Dy()*0.5, in.Dy()*0.5, VGradient(in, c.selGel...))
	}
	kw := track.Dy()
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	knob := paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy())
	ring := c.ring
	if st.Disabled() {
		ring = c.ringDis
	}
	c.paintGel(ctx, knob, kw*0.5, u, c.gelFor(st&^StateToggle), ring, c.ringTop, c.ringBot, false, false)
	if label == "" {
		if st.Focused() && !st.Disabled() {
			kde3Dotted(ctx, track, u, c.focus)
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
		kde3Dotted(ctx, labelFocusRect(l.body, label, lb, b), u, c.focus)
	}
}

// DrawSlider: a thin sunken groove and a gel handle in the selection
// colour, like a short scroll handle standing across the groove.
func (e keramikEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	t = max(0, min(t, 1))
	hw := snap(l.S(13))
	hh := min(snap(l.S(20)), b.Dy())
	if b.Dx() < hw+4*u || hh < 8*u {
		return
	}
	cy := (b.Min.Y + b.Max.Y) * 0.5
	gt := snap(l.S(6))
	groove := kde3Snap(paintengine2d.XYWH(b.Min.X+u, cy-gt*0.5, b.Dx()-2*u, gt))
	c.grooveBody(l, ctx, groove, false, true, true)
	hx := snap(b.Min.X + (b.Dx()-hw)*t)
	hb := paintengine2d.XYWH(hx, snap(cy-hh*0.5), hw, hh)
	if st.Disabled() {
		c.paintGel(ctx, hb, min(l.rx(5), hw*0.5), u, c.gelDis, c.ringDis, c.ringDis, c.ringDis, false, false)
	} else {
		c.handle(l, ctx, hb, false, st)
	}
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, b, u, c.focus)
	}
}

// DrawProgressBar: a sunken rounded groove filled with the selection gel;
// the busy bar is a gel block bouncing between the ends.
func (e keramikEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 6*u || b.Dy() < 6*u {
		return
	}
	r := max(min(l.rx(4), b.Dy()*0.5), 0)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.grooveRing))
	in := b.Inset(u)
	ri := max(r-u, 0)
	ctx.DrawRoundRect(in, ri, ri, VGradient(in, c.progGroove...))
	t = max(0, min(t, 1))
	bar := in
	if indeterminate {
		bar = kde3Bounce(in, phase, 0.25)
	} else {
		w := snap(in.Dx() * t)
		if w < 3*u {
			return
		}
		bar.Max.X = bar.Min.X + w
	}
	fill, ring := c.selGel, c.selRing
	if st.Disabled() {
		fill, ring = c.gelDis, c.ringDis
	}
	ctx.DrawRoundRect(bar, ri, ri, paintengine2d.Fill(ring))
	bi := bar.Inset(u)
	if bi.Dx() > 0 && bi.Dy() > 0 {
		ctx.DrawRoundRect(bi, max(ri-u, 0), max(ri-u, 0), VGradient(bi, fill...))
	}
}

// DrawComboBox is a push button with a three-line divider before the
// combo arrow: a block arrow under a bar.
func (e keramikEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	bst := st &^ StatePrimary
	if open {
		bst = (bst | StatePressed) &^ StateHovered
	}
	body := c.button(l, ctx, b, bst)
	aw := min(snap(l.S(20)), body.Dx()*0.4)
	ab := paintengine2d.XYWH(body.Max.X-aw-u, body.Min.Y, aw, body.Dy())
	col := c.btnText
	if st.Disabled() {
		col = c.dis
	}
	if !st.Disabled() {
		div := paintengine2d.XYWH(ab.Min.X-l.S(6), body.Min.Y, l.S(6), body.Dy())
		kde3GripLines(ctx, div, true, 3, 2*u, u, snap(body.Dy()*0.55), c.ring.WithAlpha(0.55), c.light.WithAlpha(0.8))
	}
	g := ab
	if open {
		g = g.Translate(paintengine2d.Pt(u, u))
	}
	aws := snap(min(ab.Dx(), ab.Dy()) * 0.5)
	kmComboArrow(ctx, g, aws, u, col)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	tb := paintengine2d.XYWH(body.Min.X+l.S(8), body.Min.Y, ab.Min.X-body.Min.X-l.S(16), body.Dy())
	if open {
		tb = tb.Translate(paintengine2d.Pt(u, u))
	}
	l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
	if st.Focused() && !open && !st.Disabled() {
		kde3Dotted(ctx, paintengine2d.XYWH(body.Min.X+l.S(4), body.Min.Y+l.S(4), ab.Min.X-body.Min.X-l.S(12), body.Dy()-l.S(8)), u, c.focus)
	}
}

// kmComboArrow is the combo box glyph: a bar over a block arrow pointing
// down, w across, centred in b.
func kmComboArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, w, u float32, col paintengine2d.Color) {
	if w < 4 {
		return
	}
	h := w * 0.92
	y0 := snap((b.Min.Y+b.Max.Y)*0.5 - (h+2*u)*0.5)
	x0 := snap((b.Min.X+b.Max.X)*0.5 - w*0.5)
	ctx.DrawRect(paintengine2d.XYWH(x0, y0, w, u), paintengine2d.Fill(col))
	kmBlockArrow(ctx, paintengine2d.XYWH(x0, y0+2*u, w, h), DirDown, w, col)
}

// DrawSpinner is the spin box's two stacked gel buttons, rounded at the
// outer corners, with block arrows.
func (e keramikEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 6*u || b.Dy() < 8*u {
		return
	}
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, hot, down bool, dir Direction, rad float32, up bool) {
		s := st &^ (StateHovered | StatePressed | StateToggle)
		if hot {
			s |= StateHovered
		}
		if down {
			s |= StatePressed
		}
		ring := c.ring
		if st.Disabled() {
			ring = c.ringDis
		}
		// Rounded only at the outer right corner: the top one above, the
		// bottom one below.
		kde3Rounded(ctx, r, rad, up, true, !up, false, paintengine2d.Fill(ring))
		in := r.Inset(u)
		kde3Rounded(ctx, in, max(rad-u, 0), up, true, !up, false, VGradient(in, c.gelFor(s)...))
		col := c.btnText
		if st.Disabled() {
			col = c.dis
		}
		g := in
		if down && !st.Disabled() {
			g = g.Translate(paintengine2d.Pt(0, u))
		}
		kmBlockArrow(ctx, g, dir, snap(min(in.Dx()*0.6, in.Dy()*0.95)), col)
	}
	r := min(l.rx(4), b.Dx()*0.4)
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y+u), upHover, upPress, DirUp, r, true)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), downHover, downPress, DirDown, r, false)
}

func (e keramikEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	// The page's raised top edge, which the tabs stand on.
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.light))
}

// DrawTab: rounded tops in a grey outline under a dark top edge; the
// unselected tabs stand apart and four pixels lower in a darker gel, the
// selected tab is full height, lighter, and opens into its page.
func (e keramikEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 8*u || b.Dy() < 10*u {
		return
	}
	t := b
	if !selected {
		t = paintengine2d.XYWH(b.Min.X+u, b.Min.Y+4*u, b.Dx()-2*u, b.Dy()-5*u)
	}
	r := min(l.rx(5), t.Dx()*0.3)
	kde3Rounded(ctx, t, r, true, true, false, true, paintengine2d.Fill(c.tabRing))
	if w := t.Dx() - 2*r; w > 0 {
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X+r, t.Min.Y, w, u), paintengine2d.Fill(c.tabTop))
	}
	in := paintengine2d.XYWH(t.Min.X+u, t.Min.Y+u, t.Dx()-2*u, t.Dy()-u)
	if !selected {
		in.Max.Y -= u
	}
	stops := c.tabGel
	switch {
	case selected:
		stops = c.tabSel
	case st.Hovered() && !st.Disabled():
		stops = c.tabHot
	}
	ri := max(r-u, 0)
	kde3Rounded(ctx, in, ri, true, true, false, true, VGradient(in, stops...))
	if !selected {
		// Stand on the page's light edge.
		ctx.DrawRect(paintengine2d.XYWH(t.Min.X, b.Max.Y-u, t.Dx(), u), paintengine2d.Fill(c.light))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	lb := paintengine2d.XYWH(t.Min.X, t.Min.Y+u, t.Dx(), t.Dy()-u)
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		f := l.body
		w := min(f.Advance(label)+l.S(6), lb.Dx()-l.S(8))
		h := min(f.Height()+l.S(2), lb.Dy()-2*u)
		kde3Dotted(ctx, kde3Snap(paintengine2d.XYWH(lb.Min.X+(lb.Dx()-w)*0.5, lb.Min.Y+(lb.Dy()-h)*0.5, w, h)), u, c.focus)
	}
}

// DrawPanel: flat panels are sunken one-pixel frames, raised panels a
// lighter face in a raised frame.
func (e keramikEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if raised {
		kde3Frame(ctx, b, u, lighterPct(c.bg, 103), c.light, c.mid)
		return
	}
	kde3Frame(ctx, b, u, c.bg, c.mid, c.light)
}

func (e keramikEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(keramikColors(l).bg))
}

// DrawMenuTitle: an open title is the selection gel with dark text; a hot
// one lights a faint gel so the pointer is seen.
func (e keramikEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	hot := (open || st.Pressed()) && !st.Disabled()
	switch {
	case hot:
		e.MenuHighlight(l, ctx, b.Inset(u), false)
	case st.Hovered() && !st.Disabled():
		r := l.rx(3)
		in := b.Inset(u)
		ctx.DrawRoundRect(in, r, r, paintengine2d.Fill(c.selRing.WithAlpha(0.5)))
		ctx.DrawRoundRect(in.Inset(u), max(r-u, 0), max(r-u, 0), VGradient(in, c.titleHot...))
	}
	fg := e.MenuTextColor(l, hot)
	if st.Disabled() {
		fg = c.dis
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() {
		kde3Dotted(ctx, b.Inset(2*u), u, c.focus)
	}
}

// DrawMenuFrame is a popup menu: a lighter panel in a dark frame with a
// light inner edge.
func (e keramikEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.dark))
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	in := b.Inset(u)
	ctx.DrawRect(in, paintengine2d.Fill(c.light))
	ctx.DrawRect(paintengine2d.XYWH(in.Min.X+u, in.Min.Y+u, in.Dx()-u, in.Dy()-u), paintengine2d.Fill(c.menuBg))
}

// DrawMenuItem: separators are etched across the menu; checked rows carry
// the check box's tick.
func (e keramikEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := keramikColors(l)
	kde3MenuItem(l, ctx, b, st, row, c.mid, c.light, c.dis, func(ctx *paintengine2d.Context, box paintengine2d.Rect, col paintengine2d.Color) {
		kmTick(ctx, box, col, max(1.6*kde3U(l), box.Dx()*0.15), 0)
	})
}

// DrawMessageIcon paints the message box icons in the manner of KDE 3's
// Crystal set.
func (keramikEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	kde3MessageIcon(l, ctx, b, icon)
}

func (e keramikEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := keramikColors(l)
	fg := c.row(l, ctx, b, st)
	lb := paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTreeRow is a KDE 3 tree: dotted branch lines, the plus/minus box and
// the selection from the item's text to the row's end.
func (e keramikEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := keramikColors(l)
	kde3TreeRow(l, ctx, b, st, expanded, leaf, depth, label, bold, kde3TreeColors{
		mid: c.mid, text: c.text, dis: c.dis, sel: c.sel, selText: c.selText, alt: c.alt, base: c.base,
	}, e.Expander, e.ItemFocus)
}

// DrawTableCell: a KDE 3 detail view has no grid; the selection spans the
// row and odd rows are striped.
func (e keramikEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := keramikColors(l)
	fg := c.row(l, ctx, b, st)
	kde3CellText(l, ctx, b, label, align, face, fg)
}

// DrawTableHeader is a square gel section with a divider and the sort
// triangle.
func (e keramikEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, VGradient(b, c.gelFor(st&^(StateToggle|StateFocused))...))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), u), paintengine2d.Fill(c.ringTop))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(c.ring))
	if !st.Last() {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+2*u, u, b.Dy()-4*u), paintengine2d.Fill(c.ring))
	}
	lb := b
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
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
		e.Arrow(l, ctx, paintengine2d.XYWH(lb.Max.X-aw-l.S(4), lb.Min.Y, aw, lb.Dy()), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(lb.Min.X+l.S(6), lb.Min.Y, lb.Dx()-l.S(10)-aw, lb.Dy()), fg, AlignStart, 0)
}

// DrawToolBar: the window colour with the handle at the left, three
// engraved lines.
func (e keramikEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if b.Dy() < 10*u {
		return
	}
	h := paintengine2d.XYWH(b.Min.X+l.S(2), b.Min.Y, l.S(10), b.Dy())
	kde3GripLines(ctx, h, true, 3, 2*u, u, b.Dy()-l.S(10), c.mid, c.light)
}

func (e keramikEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := keramikColors(l)
	kde3StatusBar(l, ctx, b, parts, c.bg, c.mid, c.light, c.text)
}

// DrawTitleBar (panel headings): the window colour, a bold title and an
// etched rule below.
func (e keramikEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := keramikColors(l)
	kde3Heading(l, ctx, b, title, subtitle, c.bg, c.mid, c.light, c.text, c.dis)
}

// DrawAccordionHeader is a Qt tool box tab: a gel with the branch arrow.
func (e keramikEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := keramikColors(l)
	u := kde3U(l)
	b = kde3Snap(b)
	ring, top, bot := c.ring, c.ringTop, c.ringBot
	if st.Disabled() {
		ring, top, bot = c.ringDis, c.ringDis, c.ringDis
	}
	c.paintGel(ctx, b, l.rx(3), u, c.gelFor(st&^(StateFocused|StateToggle)), ring, top, bot, false, false)
	fg := c.btnText
	if st.Disabled() {
		fg = c.dis
	}
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	e.Arrow(l, ctx, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, l.S(12), b.Dy()), dir, fg)
	lb := paintengine2d.XYWH(b.Min.X+l.S(24), b.Min.Y, b.Dx()-l.S(28), b.Dy())
	l.drawFittedText(ctx, l.body, title, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, labelFocusRect(l.body, title, lb, b), u, c.focus)
	}
}

func (e keramikEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := keramikColors(l)
	kde3Separator(l, ctx, b, vertical, c.mid, c.light)
}

// DrawSplitter: the window colour and three short engraved lines, lit in
// the selection colour under the pointer.
func (e keramikEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := keramikColors(l)
	u := kde3U(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.sel.WithAlpha(0.3)))
	}
	length := snap(min(l.S(24), max(b.Dx(), b.Dy())*0.5))
	if vertical && b.Dx() >= 4*u {
		kde3GripLines(ctx, b, true, 2, 3*u, u, length, c.mid, c.light)
	} else if !vertical && b.Dy() >= 4*u {
		kde3GripLines(ctx, b, false, 2, 3*u, u, length, c.mid, c.light)
	}
}

// DrawTooltip: Qt 3's pale yellow tip in a one-pixel black frame.
func (e keramikEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := keramikColors(l)
	kde3Tooltip(l, ctx, b, text, c.tip, c.tipText)
}

// ---- packs --------------------------------------------------------------------------------------

func keramikPacks() []ThemePack {
	// KDE 3.1–3.3's default colour scheme (preserved in KDE 3.4 as the
	// "Keramik" scheme): a warm light grey window, blue-white buttons, a
	// pale blue selection with dark text, blue captions.
	s := kde3Scheme{
		background: "#eae9e8", foreground: "#000000",
		button: "#e6f0f9", buttonText: "#000000",
		selection: "#a9d1ff", selectionText: "#030303",
		base: "#ffffff", text: "#000000",
		alternate: "#eef6ff", link: "#0000c0",
		caption: "#3e91eb", captionBlend: "#3e91eb", captionText: "#ffffff",
		captionOff: "#afd6ff", captionOffBlend: "#afd6ff", captionOffText: "#ffffff",
	}
	return []ThemePack{
		kde3Pack("keramik", "Keramik", 2002, "KDE", "Keramik",
			"KDE 3.1's default: gel buttons with a white sheen, blue gel scroll handles, rounded tabs and the bubble title bar.",
			"keramik", BevelSoftShadow, s, map[string]float32{"shadow": 0, "stripes": 1, "sbHighlight": 1}),
	}
}

// ---- KDE 3 helpers, shared with the Plastik engine ------------------------------------------------

// kde3Scheme is a KDE 3 colour scheme: the colours KDE's Colors module
// wrote to kdeglobals for every application (the .kcsrc keys).
type kde3Scheme struct {
	background, foreground, button, buttonText  string
	selection, selectionText, base, text        string
	alternate, link                             string
	caption, captionBlend, captionText          string
	captionOff, captionOffBlend, captionOffText string
}

// kde3Disabled is KDE 3's disabled text: black text greys to Qt's darkGray,
// other colours move halfway to the background.
func kde3Disabled(fg, bg paintengine2d.Color) paintengine2d.Color {
	if fg.R+fg.G+fg.B < 0.05 {
		return Hex("#808080")
	}
	return Mix(fg, bg, 0.5)
}

// palette maps the scheme onto the shared palette: Background is the
// window background, SurfaceAlt the button colour, Field the view base,
// Selection / TextOnAccent the highlight pair and Accent the link colour.
func (s kde3Scheme) palette() Palette {
	bg, btn, sel := Hex(s.background), Hex(s.button), Hex(s.selection)
	fg, base := Hex(s.foreground), Hex(s.base)
	link := Hex(s.link)
	white := paintengine2d.RGB(1, 1, 1)
	return Palette{
		Background: bg, Surface: bg, SurfaceAlt: btn,
		Border: darkerPct(btn, 158), Divider: darkerPct(bg, 125),
		Text: fg, TextMuted: kde3Disabled(fg, bg), TextOnAccent: Hex(s.selectionText),
		Accent: link, AccentHover: lighterPct(link, 130), AccentPress: darkerPct(link, 120),
		Field: base, FieldBorder: darkerPct(btn, 158),
		Focus: sel, Selection: sel,
		Track: Mix(bg, white, 0.5), Thumb: btn,
		Highlight: white.WithAlpha(0.5), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: sel, MenuHoverBorder: darkerPct(sel, 130), MenuGutter: lighterPct(bg, 105),
		Danger: Hex("#c00000"), Success: Hex("#008000"), Warning: Hex("#c08000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: lighterPct(bg, 150), BevelDark: darkerPct(bg, 150),
	}
}

// kde3Pack builds a pack from a KDE 3 colour scheme; the scheme's own keys
// travel as extras so a theme.json can restyle any of them.
func kde3Pack(name, label string, year int, lineage, era, summary, engine string, bevel BevelStyle, s kde3Scheme, params map[string]float32) ThemePack {
	pal := s.palette()
	extra := map[string]paintengine2d.Color{
		"window": pal.Background, "button": pal.SurfaceAlt, "buttonText": Hex(s.buttonText),
		"base": pal.Field, "alternate": Hex(s.alternate), "disabledText": pal.TextMuted,
		"caption": Hex(s.caption), "caption2": Hex(s.captionBlend), "captionText": Hex(s.captionText),
		"captionOff": Hex(s.captionOff), "captionOff2": Hex(s.captionOffBlend), "captionOffText": Hex(s.captionOffText),
	}
	tok := ThemeTokens{
		Engine:  engine,
		Bevel:   bevel,
		Family:  ThemeLight,
		Palette: pal,
		Era:     era,
		Extra:   extra,
		Params:  params,
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Selection}
	tok.Focus = ChromeState{Fill: pal.Focus.WithAlpha(0.15), Border: pal.Focus}
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: lineage, Summary: summary,
		Era: era, Palette: ThemeLight, Tokens: tok,
	}
}

// kde3Rounded fills b with paint, rounded (radius r) only at the corners
// between two of the listed sides; the other corners stay square. It clips
// a fully rounded rectangle, grown past the square sides, to b — drawing
// with the context's scratch path instead of building one.
func kde3Rounded(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, top, right, bottom, left bool, paint paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	if r <= 0 || (top && right && bottom && left) {
		ctx.DrawRoundRect(b, r, r, paint)
		return
	}
	g := b
	if !top {
		g.Min.Y -= r
	}
	if !bottom {
		g.Max.Y += r
	}
	if !left {
		g.Min.X -= r
	}
	if !right {
		g.Max.X += r
	}
	ctx.Save()
	ctx.ClipRect(b)
	ctx.DrawRoundRect(g, r, r, paint)
	ctx.Restore()
}

// kde3Paths recycles the paths the KDE 3 engines build for glyphs, grips
// and dotted lines. A device consumes a path during the draw call (the
// context reuses its own scratch path the same way, and recorders keep a
// snapshot), so a drawn path can go straight back to the pool.
var kde3Paths = sync.Pool{New: func() any { return paintengine2d.NewPath() }}

// kde3Path is a cleared path from the pool; hand it back with kde3Done.
func kde3Path() *paintengine2d.Path {
	p := kde3Paths.Get().(*paintengine2d.Path)
	p.Reset()
	return p
}

func kde3Done(p *paintengine2d.Path) { kde3Paths.Put(p) }

type kde3ToolFontKey struct{}

// kde3ControlFont is the face KDE 3 labelled a control with: its tool bar
// font a point smaller than the general font (Sans 9 against 10 in KDE 3's
// defaults; the "toolFont" param is the ratio), everything else in the
// general font.
func kde3ControlFont(l *Classic, role Role) *Font {
	k := l.P("toolFont", 0.9)
	if role != RoleTool || k <= 0 || k >= 0.999 {
		return l.body
	}
	return l.Memo(kde3ToolFontKey{}, func() any {
		return BakeFamily(FamilyUI, WeightRegular, l.metrics.FontSize*k, l.palette.Text)
	}).(*Font)
}

// kde3ToolButton paints a tool button the stock way — the engine's tool
// face while hot, held or latched, the icon, the label — with the label in
// the tool bar font the tool bar measures it with.
func kde3ToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon, dis paintengine2d.Color) {
	e := l.eng()
	fg := l.palette.Text
	if st.Toggle() || ((st.Hovered() || st.Pressed()) && !st.Disabled()) {
		fg = e.Face(l, ctx, b.Inset(1), RoleTool, st)
	}
	if st.Focused() {
		e.DrawFocusRing(l, ctx, b.Inset(1))
	}
	if st.Disabled() {
		fg = dis
	}
	font := kde3ControlFont(l, RoleTool)
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
	if label == "" {
		return
	}
	right := max(b.Max.X-pad, x)
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
	font.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-font.Height())*0.5), fg)
	ctx.Restore()
}

// kde3TitleShadow is the shadow under a title bar's text: a darker tone of
// the bar under light text, none under dark text.
func kde3TitleShadow(text, bar paintengine2d.Color) paintengine2d.Color {
	if Luma(text) < 0.5 {
		return paintengine2d.Color{}
	}
	return darkerPct(bar, 200).WithAlpha(0.75)
}

// kde3U is one design pixel at the look's scale, never under a device
// pixel.
func kde3U(l *Classic) float32 {
	return max(snap(l.S(1)), 1)
}

// kde3Snap rounds b inwards to whole pixels, so hairlines stay crisp and
// nothing paints outside the rect it was given.
func kde3Snap(b paintengine2d.Rect) paintengine2d.Rect {
	x0 := float32(math.Ceil(float64(b.Min.X) - 0.01))
	y0 := float32(math.Ceil(float64(b.Min.Y) - 0.01))
	x1 := float32(math.Floor(float64(b.Max.X) + 0.01))
	y1 := float32(math.Floor(float64(b.Max.Y) + 0.01))
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(max(x1, x0), max(y1, y0))}
}

// kde3Dotted is Qt 3's focus rectangle: dots one unit apart just inside b,
// batched into one path.
func kde3Dotted(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, col paintengine2d.Color) {
	b = kde3Snap(b)
	if b.Dx() < 3*u || b.Dy() < 3*u || col.A <= 0 {
		return
	}
	p := kde3Path()
	defer kde3Done(p)
	for x := b.Min.X; x+u <= b.Max.X; x += 2 * u {
		p.AddRect(paintengine2d.XYWH(x, b.Min.Y, u, u))
		p.AddRect(paintengine2d.XYWH(x, b.Max.Y-u, u, u))
	}
	for y := b.Min.Y + 2*u; y+u <= b.Max.Y-u; y += 2 * u {
		p.AddRect(paintengine2d.XYWH(b.Min.X, y, u, u))
		p.AddRect(paintengine2d.XYWH(b.Max.X-u, y, u, u))
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// kde3Frame fills b with fill inside a one-unit 3D frame: hi along the top
// and left, lo along the bottom and right (swap them to sink it).
func kde3Frame(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, fill, hi, lo paintengine2d.Color) {
	if b.Dx() < 2*u || b.Dy() < 2*u {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(lo))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()-u, b.Dy()-u), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y+u, b.Dx()-2*u, b.Dy()-2*u), paintengine2d.Fill(fill))
}

// kde3PlusBox is the KDE 3 list view expander: an odd-sized square with a
// one-unit frame and a plus (collapsed) or minus (expanded).
func kde3PlusBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, fill, frame, sign paintengine2d.Color) {
	u := kde3U(l)
	s := snap(l.S(9))
	if int(s/u)%2 == 0 {
		s += u
	}
	if s > min(b.Dx(), b.Dy()) {
		return
	}
	box := paintengine2d.XYWH(snap(b.Min.X+(b.Dx()-s)*0.5), snap(b.Min.Y+(b.Dy()-s)*0.5), s, s)
	ctx.DrawRect(box, paintengine2d.Fill(frame))
	ctx.DrawRect(box.Inset(u), paintengine2d.Fill(fill))
	mid := box.Min.Y + snap((s-u)*0.5)
	ctx.DrawRect(paintengine2d.XYWH(box.Min.X+2*u, mid, s-4*u, u), paintengine2d.Fill(sign))
	if !expanded {
		ctx.DrawRect(paintengine2d.XYWH(box.Min.X+snap((s-u)*0.5), box.Min.Y+2*u, u, s-4*u), paintengine2d.Fill(sign))
	}
}

// kde3TreeColors are the colours a KDE 3 tree row needs.
type kde3TreeColors struct {
	mid, text, dis, sel, selText, alt, base paintengine2d.Color
}

// kde3TreeRow paints a Qt 3 list view tree row: dotted branch lines to the
// item, the engine's expander, the selection from the label to the end of
// the row, and the current-item mark around the label.
func kde3TreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool,
	c kde3TreeColors, expander func(*Classic, *paintengine2d.Context, paintengine2d.Rect, bool, paintengine2d.Color),
	focus func(*Classic, *paintengine2d.Context, paintengine2d.Rect, ControlState)) {
	u := kde3U(l)
	if st.Alternate() && !st.Checked() && l.P("stripes", 1) != 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.alt))
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	pad := l.S(4)
	x := b.Min.X + pad + float32(depth)*indent
	cy := snap((b.Min.Y+b.Max.Y)*0.5) - u
	// The branch lines, one dotted pass for every ancestor level plus the
	// elbow to this item, batched into one path.
	if c.mid.A > 0 {
		dots := kde3Path()
		defer kde3Done(dots)
		for d := 0; d <= depth; d++ {
			gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
			y1 := b.Max.Y
			if d == depth {
				y1 = cy + u
			}
			for y := snap(b.Min.Y); y < y1; y += 2 * u {
				dots.AddRect(paintengine2d.XYWH(gx, y, u, u))
			}
			if d == depth {
				for xx := gx + 2*u; xx < x+indent+l.S(2); xx += 2 * u {
					dots.AddRect(paintengine2d.XYWH(xx, cy, u, u))
				}
			}
		}
		ctx.Save()
		ctx.ClipRect(b)
		ctx.DrawPath(dots, paintengine2d.Fill(c.mid))
		ctx.Restore()
	}
	if !leaf {
		expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.text)
	}
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	}
	lx := x + indent + l.S(4)
	sel := paintengine2d.XYWH(lx-l.S(3), b.Min.Y, b.Max.X-(lx-l.S(3)), b.Dy()).Intersect(b)
	fg := c.text
	if st.Checked() {
		fill := c.sel
		fg = c.selText
		if st.Disabled() {
			fill, fg = Mix(c.sel, c.base, 0.6), c.dis
		}
		ctx.DrawRect(sel, paintengine2d.Fill(fill))
	} else if st.Disabled() {
		fg = c.dis
	} else {
		fg = l.fieldText()
	}
	ctx.Save()
	ctx.ClipRect(b)
	f.Draw(ctx, label, paintengine2d.Pt(lx, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
	if st.Focused() && !sel.Empty() {
		focus(l, ctx, sel, st)
	}
}

// kde3CellText draws a table cell's label (padded, fitted, aligned).
func kde3CellText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, align Align, face *Font, fg paintengine2d.Color) {
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
	ctx.ClipRect(b.Inset(1))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// kde3StatusBar is KDE 3's status bar: each part in a sunken one-unit
// panel, the size grip's diagonal lines at the right.
func kde3StatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string, bg, dark, light, text paintengine2d.Color) {
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	grip := l.S(16)
	if n := len(parts); n > 0 {
		slot := (b.Dx() - grip) / float32(n)
		gap := l.S(2)
		for i, s := range parts {
			r := kde3Snap(paintengine2d.XYWH(b.Min.X+slot*float32(i)+gap, b.Min.Y+2*u, slot-gap*2, b.Dy()-4*u))
			kde3Frame(ctx, r, u, bg, dark, light)
			l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(r.Min.X+l.S(4), r.Min.Y, r.Dx()-l.S(8), r.Dy()), text, AlignStart, 0)
		}
	}
	if b.Dx() < grip || b.Dy() < grip*0.75 {
		return
	}
	d, lt := kde3Path(), kde3Path()
	defer kde3Done(d)
	defer kde3Done(lt)
	gx, gy := b.Max.X-u, b.Max.Y-u
	for i := 1; i <= 3; i++ {
		o := snap(l.S(4) * float32(i))
		for k := float32(0); k < o; k += u {
			d.AddRect(paintengine2d.XYWH(gx-o+k, gy-k-u, u, u))
			lt.AddRect(paintengine2d.XYWH(gx-o+k+u, gy-k-u, u, u))
		}
	}
	ctx.DrawPath(lt, paintengine2d.Fill(light))
	ctx.DrawPath(d, paintengine2d.Fill(dark))
}

// kde3Heading is a panel heading: the window colour, a bold title, a
// muted subtitle and an etched rule along the bottom.
func kde3Heading(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string, bg, dark, light, text, muted paintengine2d.Color) {
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(bg))
	if b.Dy() > 3*u {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-2*u, b.Dx(), u), paintengine2d.Fill(dark))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-u, b.Dx(), u), paintengine2d.Fill(light))
	}
	x := b.Min.X + l.metrics.Pad
	f := l.bold
	if f == nil {
		f = l.body
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()-2*u), muted, AlignStart, 0)
	}
}

// kde3Separator is Qt 3's sunken line (QFrame::HLine / VLine): a dark line
// and a light one beside it.
func kde3Separator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, dark, light paintengine2d.Color) {
	u := kde3U(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - u
		m := min(l.S(2), b.Dy()*0.25)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+m, u, b.Dy()-2*m), paintengine2d.Fill(dark))
		ctx.DrawRect(paintengine2d.XYWH(x+u, b.Min.Y+m, u, b.Dy()-2*m), paintengine2d.Fill(light))
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - u
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), u), paintengine2d.Fill(dark))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+u, b.Dx(), u), paintengine2d.Fill(light))
}

// kde3Tooltip is Qt 3's tooltip: a pale fill in a one-unit frame of the
// text colour.
func kde3Tooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, fill, fg paintengine2d.Color) {
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(fg))
	if b.Dx() > 2*u && b.Dy() > 2*u {
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(fill))
	}
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(4)
	}
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), fg, AlignStart, 0)
}

// kde3MenuItem paints a popup menu row as Qt 3 laid it out: an etched
// separator across the menu, or the engine's highlight when hot, the check
// column (the engine's check mark, a radio dot or the action's icon, sized
// to the row), the label with its mnemonic, the shortcut in the label's
// colour and the engine's submenu arrow.
func kde3MenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow,
	dark, light, dis paintengine2d.Color, check func(*paintengine2d.Context, paintengine2d.Rect, paintengine2d.Color)) {
	e := l.eng()
	u := kde3U(l)
	ch := MenuChromeFor(l)
	if row.Separator {
		y := snap((b.Min.Y+b.Max.Y)*0.5) - u
		x0 := b.Min.X - ch.PadL + 2*u
		x1 := b.Max.X + ch.PadR - 2*u
		ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, u), paintengine2d.Fill(dark))
		ctx.DrawRect(paintengine2d.XYWH(x0, y+u, x1-x0, u), paintengine2d.Fill(light))
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		e.MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	fg := e.MenuTextColor(l, hot)
	if st.Disabled() {
		fg = dis
	}
	gw := ch.CheckCol()
	side := min(snap(l.S(16)), snap(b.Dy()-4*u), snap(gw))
	if side >= 6*u {
		ib := kde3Snap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		switch {
		case row.Radio:
			if row.Checked {
				ctx.DrawCircle(ib.Center(), max(side*0.2, 2*u), paintengine2d.Fill(fg))
			}
		case row.Checked:
			check(ctx, ib.Inset(u), fg)
		case row.Icon != IconNone:
			l.drawToolIcon(ctx, ib, row.Icon, fg)
		}
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		aw := max(ch.SubmenuArrow, l.S(10))
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		e.Arrow(l, ctx, ab, DirRight, fg)
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
	right = max(right, lx)
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	l.drawTextUnderline(ctx, l.body, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// Message icon colours, in the manner of KDE 3's Crystal icons: a light
// and a deep tone for each glossy disc or triangle, and its rim.
var (
	kde3InfoStops  = []paintengine2d.GradientStop{Stop(0, Hex("#a9d4ff")), Stop(0.55, Hex("#3b82e0")), Stop(1, Hex("#1a4fa8"))}
	kde3ErrStops   = []paintengine2d.GradientStop{Stop(0, Hex("#ffb0a0")), Stop(0.55, Hex("#e0301e")), Stop(1, Hex("#9c140c"))}
	kde3WarnStops  = []paintengine2d.GradientStop{Stop(0, Hex("#fff6a0")), Stop(0.6, Hex("#f7c51b")), Stop(1, Hex("#d08c00"))}
	kde3GlossStops = []paintengine2d.GradientStop{Stop(0, paintengine2d.RGBA(1, 1, 1, 0.7)), Stop(1, paintengine2d.RGBA(1, 1, 1, 0.05))}
)

// kde3MessageIcon paints a message box icon: a glossy blue disc with a
// white "i" or "?", a red disc with a white cross, or a yellow triangle
// with a black "!".
func kde3MessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	s := min(b.Dx(), b.Dy())
	if icon == IconNone || s < 8 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	stroke := func(col paintengine2d.Color, w float32) paintengine2d.Paint {
		return paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
	}
	glossy := func(stops []paintengine2d.GradientStop, rim paintengine2d.Color) {
		r := s * 0.46
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Radial(paintengine2d.RadialGradient{
			Center: paintengine2d.Pt(cx-r*0.35, cy-r*0.45), Radius: r * 1.55, Stops: stops,
		}))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r-0.5, paintengine2d.StrokePaint(rim, 1))
		gl := paintengine2d.XYWH(cx-r*0.6, cy-r*0.9, r*1.2, r*0.75)
		ctx.DrawOval(gl, VGradient(gl, kde3GlossStops...))
	}
	w := max(s*0.12, 1.5)
	switch icon {
	case IconInfo:
		glossy(kde3InfoStops, Hex("#163f86"))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy-s*0.2), s*0.07, paintengine2d.Fill(white))
		ctx.DrawLine(paintengine2d.Pt(cx, cy-s*0.04), paintengine2d.Pt(cx, cy+s*0.24), stroke(white, w))
	case IconQuestion:
		glossy(kde3InfoStops, Hex("#163f86"))
		p := paintengine2d.NewPath()
		p.MoveTo(cx-s*0.12, cy-s*0.1)
		p.CubicTo(cx-s*0.12, cy-s*0.28, cx+s*0.13, cy-s*0.28, cx+s*0.13, cy-s*0.11)
		p.CubicTo(cx+s*0.13, cy-s*0.01, cx, cy, cx, cy+s*0.08)
		ctx.DrawPath(p, stroke(white, w*0.9))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy+s*0.23), s*0.065, paintengine2d.Fill(white))
	case IconError:
		glossy(kde3ErrStops, Hex("#7a0f08"))
		d := s * 0.15
		ctx.DrawLine(paintengine2d.Pt(cx-d, cy-d), paintengine2d.Pt(cx+d, cy+d), stroke(white, w))
		ctx.DrawLine(paintengine2d.Pt(cx+d, cy-d), paintengine2d.Pt(cx-d, cy+d), stroke(white, w))
	case IconWarning:
		t := paintengine2d.NewPath()
		top, bot := cy-s*0.44, cy+s*0.4
		t.MoveTo(cx, top)
		t.LineTo(cx+s*0.47, bot)
		t.LineTo(cx-s*0.47, bot)
		t.Close()
		tb := paintengine2d.XYWH(cx-s*0.47, top, s*0.94, bot-top)
		ctx.DrawPath(t, VGradient(tb, kde3WarnStops...))
		ctx.DrawPath(t, paintengine2d.Paint{Color: Hex("#8a5a00"), Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 1, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		ctx.DrawLine(paintengine2d.Pt(cx, cy-s*0.17), paintengine2d.Pt(cx, cy+s*0.1), stroke(black, w))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy+s*0.25), s*0.065, paintengine2d.Fill(black))
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// kde3Bounce is Qt 3's busy indicator: a block a fraction of the groove
// wide moving from one end to the other and back over one phase.
func kde3Bounce(tr paintengine2d.Rect, phase, frac float32) paintengine2d.Rect {
	phase -= float32(math.Floor(float64(phase)))
	w := snap(tr.Dx() * frac)
	t := phase * 2
	if t > 1 {
		t = 2 - t
	}
	return paintengine2d.XYWH(snap(tr.Min.X+(tr.Dx()-w)*t), tr.Min.Y, w, tr.Dy())
}
