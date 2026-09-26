package style

import "github.com/codemodify/paintengine2d"

// kde2Engine paints KDE 2's widget style — "HighColor", the style KDE 2
// chose for itself on any display deeper than 8 bits and the look of the
// KDE 2 series (2.0, October 2000, on Qt 2). It is the first KDE style
// that is KDE's own rather than a Motif or a Windows look-alike, and it
// has one idea, repeated everywhere: a raised part is a *bevelled slab
// with its four corner pixels cut away*, lit by a two-pixel highlight
// along its top and left and closed by a one-pixel shade along its bottom
// and right, over a gentle vertical gradient.
//
//   - push buttons, combo boxes, tab labels, scroll step buttons, scroll
//     handles, slider handles and header sections are that slab. Its face
//     runs from the button colour lightened a tenth at the top to the same
//     darkened a tenth at the bottom; held, it fills flat with the mid
//     shade and the bevel turns over; hovered, it fills flat and pale;
//   - check boxes are 13px sunken wells filled light, and the mark is a
//     bold **X**, not a tick; radio buttons are sunken beads with a small
//     round dot;
//   - scroll bars are KDE's three-button bars: one step-back button at the
//     top (left), then the groove, then step-back and step-forward
//     *together* at the bottom (right). The handle carries three raised
//     ridges across its middle;
//   - every arrow is a triangle with its tip cut flat — eight units across
//     the base, two across the point, five deep — which is the quickest
//     way to tell HighColor from anything else at a glance;
//   - tabs are chamfered at the top left only and square at the top right;
//     the selected one is the window colour and stands a pixel taller than
//     the rest, which sit in the mid shade;
//   - menu bars and tool bars are shallow raised panels over the same
//     gradient, a tool bar carrying three ridges as a drag grip at its
//     left, and its buttons stay flat until the pointer is on them;
//   - the armed menu row is a grey ramp with a dark line under it and the
//     label in white, while a list row selects in the scheme's blue;
//   - progress bars are plain solid blocks in the selection colour, with
//     no bevel and no gradient at all.
//
// KDE 2 had no compositor: nothing casts a shadow, and nothing is rounded
// beyond that one-pixel cut corner.
//
// Sources. No KDE or Qt code, pixmap or data is used; these are documented
// values and pixels measured on published screenshots:
//
//   - kde.org/announcements/1-2-3/2.0/ — KDE 2.0, 23 October 2000, on Qt 2
//     with "over 14" widget styles; en.wikipedia.org/wiki/KWin — KWin 2.0;
//   - KDE 2's Control Center wrote a default colour scheme of background
//     #dcdcdc, buttonBackground #e4e4e4, foreground black, windowBackground
//     white, alternateBackground #f0f0f0, selectBackground #0a5f89 on
//     white, activeBackground #0a5f89 with activeBlend the same colour —
//     so the title bar is flat — inactiveBackground #dcdcdc, link #0000c0,
//     visited #800080, contrast 7 (which sets the light and dark shades at
//     about 128% and a 2.8th of the button colour);
//   - the style's own geometry: 13px check and radio indicators, an 18px
//     slider handle with three ridges, 8px arrows drawn flat-topped, tab
//     frames of 24 and 10 with no overlap, popup rows at least 18px tall,
//     and a bevel whose interior begins four pixels in at the top and left
//     and two at the bottom and right;
//   - Wikimedia Commons "KDE-2.0-es-es.png". Measured over a #dcdcdc
//     window: a push button is a #4e4e4e outline with the corner pixel left
//     as background, a #b7b7b7 ring inside it, two rows of #ffffff, then a
//     face falling from #f0f0f0 to #cfcfcf — and #cfcfcf is exactly the
//     button colour #e4e4e4 darkened a tenth. The armed menu row measures a
//     ramp from #b7b7b7 to #7f7f7f under a #4e4e4e line, and its label is
//     white where an idle row's is black;
//   - Wikimedia Commons "KDE 2.2.2.png" and "Kde2.2.2-1.png": the
//     three-button scroll bar with ridges on its handle, the #0a5f89 title
//     bar flat across its whole width with a lighter line along its top,
//     the white view background, the flat tool bar with its grip, and the
//     tab tops.
//
// Judgement call: by KDE 2.2 the style armed a menu row with a pale sunken
// fill and a *black* label. This pack is KDE 2.0, so it keeps the grey ramp
// and the white label that the 2.0 screenshot measures.
//
// Pack data (theme.json "extra"; defaults come from the palette):
//
//	window, button, buttonText, base, baseText, alternate, disabledText
//	outline, ring                          the slab's two frames
//	selection, selectionText               list and tree rows
//	caption, captionText                   the active title bar
//	captionOff, captionOffText             the inactive one
//	tip, tipText                           tooltips
//
// Params: "titleStipple" (1 = the woven dots on the active title bar),
// "formLabelsRight" (0: Qt 2 dialogs), "toolFont" (the tool bar font's size
// against the general font; KDE 2 read tool bars at 10 against 12, so 0.83).
type kde2Engine struct{ BaseEngine }

func init() {
	RegisterEngine(kde2Engine{})
	for _, p := range kde2Packs() {
		RegisterPack(p)
	}
}

func (kde2Engine) ID() string { return "kde2" }

// DefaultMetrics are KDE 2's proportions scaled from Qt 2's 12px Helvetica
// to the toolkit's 16px: 13px indicators become 16, the 16px scroll bar 20,
// and the one-pixel cut corner stays one pixel.
func (kde2Engine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 2,
		Radius:    1, RadiusSmall: 1, BevelDepth: 2,
		ControlH: 30, FieldH: 28, ComboH: 28,
		Checkbox: 17, Radio: 17,
		MenuItemH: 24, MenuBarH: 26, TabH: 28, RowH: 22,
		TitleBar: 26, HeaderH: 24, ProgressH: 20, SliderH: 26, Thumb: 24,
		Scroll: 18, Pad: 9, FieldPad: 5, FocusWidth: 1, Border: 1,
		ToolBarH: 36, StatusBarH: 24, SpinnerW: 18, SwitchW: 42, SwitchH: 20,
	}
}

// StyleHint: KDE's dialogs put the accepting button first, and Qt 2 — and
// so KDE 2 — left-aligned the labels of a form.
func (kde2Engine) StyleHint(l *Classic, h StyleHint) int {
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

type kde2Look struct {
	bg, btn, base, baseText, text, btnText, dis paintengine2d.Color
	sel, selText, alt, groove, focus            paintengine2d.Color
	outline, ring, light, midlight, mid         paintengine2d.Color
	tip, tipText                                paintengine2d.Color
	face                                        []paintengine2d.GradientStop
	barFace                                     []paintengine2d.GradientStop
	menuHot                                     []paintengine2d.GradientStop
	menuHotText, menuHotLine                    paintengine2d.Color
	grip, gripLt                                paintengine2d.Color
	cap, capOff                                 paintengine2d.Color
	capDot, capDotDark, capOffDot, capOffDark   paintengine2d.Color
	capText, capOffText, capTop, capOffTop      paintengine2d.Color
	stipple                                     bool
}

type kde2Key struct{}

func kde2Colors(l *Classic) *kde2Look {
	return l.Memo(kde2Key{}, func() any { return kde2Build(l) }).(*kde2Look)
}

// kde2Ramp is the face of a slab over c: the colour lightened a tenth at
// the top, darkened a tenth at the bottom.
func kde2Ramp(c paintengine2d.Color) []paintengine2d.GradientStop {
	return []paintengine2d.GradientStop{Stop(0, lighterPct(c, 110)), Stop(1, darkerPct(c, 110))}
}

func kde2Build(l *Classic) *kde2Look {
	p := l.palette
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	bg := l.X("window", p.Background)
	btn := l.X("button", firstSet(p.SurfaceAlt, bg))
	base := l.X("base", firstSet(p.Field, white))
	sel := l.X("selection", firstSet(p.Selection, Hex("#0a5f89")))
	c := &kde2Look{
		bg: bg, btn: btn, base: base,
		text:     l.X("windowText", p.Text),
		baseText: l.X("baseText", p.Text),
		btnText:  l.X("buttonText", p.Text),
		dis:      l.X("disabledText", firstSet(p.TextMuted, Hex("#808080"))),
		sel:      sel,
		selText:  l.X("selectionText", ReadableOn(sel, 4.5, white, black)),
		alt:      l.X("alternate", base),
		// contrast 7: the light shade is the button at about 128% of its
		// value, the dark one a 2.8th of it.
		light:   l.X("light", lighterPct(btn, 128)),
		outline: l.X("outline", darkerPct(btn, 280)),
		ring:    l.X("ring", darkerPct(btn, 125)),
		tip:     l.X("tip", Hex("#ffffdc")),
	}
	c.mid = c.ring
	c.midlight = Mix(btn, c.light, 0.5)
	c.tipText = l.X("tipText", ReadableOn(c.tip, 4.5, black, white))
	c.groove = l.X("groove", bg)
	c.focus = l.X("focus", c.text)
	c.face = kde2Ramp(btn)
	c.barFace = kde2Ramp(bg)
	// The armed menu row: the ring shade falling to half the button's
	// value, under the outline, with a white label.
	c.menuHot = []paintengine2d.GradientStop{Stop(0, c.ring), Stop(1, darkerPct(btn, 175))}
	c.menuHotText = l.X("menuHotText", white)
	c.menuHotLine = c.outline
	c.grip, c.gripLt = c.ring, c.light

	cap := l.X("caption", Hex("#0a5f89"))
	off := l.X("captionOff", bg)
	c.cap, c.capOff = cap, off
	c.capTop, c.capOffTop = lighterPct(cap, 145), lighterPct(off, 112)
	c.capDot, c.capDotDark = lighterPct(cap, 150), darkerPct(cap, 150)
	c.capOffDot, c.capOffDark = lighterPct(off, 150), darkerPct(off, 150)
	c.capText = l.X("captionText", ReadableOn(cap, 4.5, white, black))
	c.capOffText = l.X("captionOffText", ReadableOn(off, 4.5, black, white))
	c.stipple = l.P("titleStipple", 1) != 0
	return c
}

// firstSet is a, or b when a is unset.
func firstSet(a, b paintengine2d.Color) paintengine2d.Color {
	if colorUnset(a) {
		return b
	}
	return a
}

func (c *kde2Look) fgFor(st ControlState) paintengine2d.Color {
	if st.Disabled() {
		return c.dis
	}
	return c.btnText
}

// ---- the slab -----------------------------------------------------------------------------

// kde2CutFrame is the one-unit frame HighColor drew round every raised
// and sunken part: four edges with the corner pixels left alone, which is
// what gives the style its chamfered look. A look forced square keeps its
// corners.
func kde2CutFrame(ctx *paintengine2d.Context, b paintengine2d.Rect, u float32, col paintengine2d.Color, square bool) {
	if square {
		ctx.DrawRect(b, paintengine2d.Fill(col))
		return
	}
	paint := paintengine2d.Fill(col)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Min.Y, b.Dx()-2*u, u), paint)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+u, b.Max.Y-u, b.Dx()-2*u, u), paint)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+u, u, b.Dy()-2*u), paint)
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-u, b.Min.Y+u, u, b.Dy()-2*u), paint)
}

// slab is HighColor's raised part: the outline with its corner pixels cut
// away, a ring inside it, a two-pixel light along the top and left, a
// one-pixel ring along the bottom and right, and the face inside that. A
// held part turns the light and the ring over; fill, when set, replaces
// the gradient with a flat colour.
func (c *kde2Look) slab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 4*u || b.Dy() < 4*u {
		return
	}
	down := st.Pressed() || (st.Toggle() && st.Checked())
	outline, ring, light := c.outline, c.ring, c.light
	if st.Disabled() {
		outline, ring, light = Mix(outline, c.bg, 0.5), Mix(ring, c.bg, 0.5), Mix(light, c.bg, 0.4)
	}
	if down {
		light, ring = ring, light
	}
	kde2CutFrame(ctx, b, u, outline, l.square())
	ctx.DrawRect(b.Inset(u), paintengine2d.Fill(ring))
	ctx.DrawRect(b.Inset(2*u), paintengine2d.Fill(light))
	face := paintengine2d.Rect{
		Min: paintengine2d.Pt(b.Min.X+4*u, b.Min.Y+4*u),
		Max: paintengine2d.Pt(b.Max.X-2*u, b.Max.Y-2*u),
	}
	if face.Dx() <= 0 || face.Dy() <= 0 {
		return
	}
	switch {
	case st.Disabled():
		ctx.DrawRect(face, paintengine2d.Fill(Mix(c.btn, c.bg, 0.4)))
	case down:
		// Held: flat, in the mid shade.
		ctx.DrawRect(face, paintengine2d.Fill(c.mid))
	case st.Hovered():
		// Entering a button trades the gradient for a flat pale fill.
		ctx.DrawRect(face, paintengine2d.Fill(c.midlight))
	default:
		ctx.DrawRect(face, VGradient(face, c.face...))
	}
}

// well is a sunken hole — a field, a groove, a check box: the outline with
// its corners cut, the light along the bottom and right, the ring along the
// top and left, and fill inside. It returns the filled box.
func (c *kde2Look) well(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill paintengine2d.Color, st ControlState) paintengine2d.Rect {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 3*u || b.Dy() < 3*u {
		return b
	}
	outline, ring, light := c.outline, c.ring, c.light
	if st.Disabled() {
		outline, ring, light = Mix(outline, c.bg, 0.5), Mix(ring, c.bg, 0.5), Mix(light, c.bg, 0.4)
	}
	kde2CutFrame(ctx, b, u, light, l.square())
	ctx.DrawRect(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+u, b.Min.Y), Max: paintengine2d.Pt(b.Max.X-u, b.Max.Y-u)}, paintengine2d.Fill(outline))
	ctx.DrawRect(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+u), Max: paintengine2d.Pt(b.Max.X-u, b.Max.Y-u)}, paintengine2d.Fill(outline))
	in := b.Inset(u)
	if in.Dx() <= 0 || in.Dy() <= 0 {
		return in
	}
	ctx.DrawRect(paintengine2d.Rect{Min: in.Min, Max: paintengine2d.Pt(in.Max.X, in.Max.Y)}, paintengine2d.Fill(ring))
	inner := paintengine2d.Rect{Min: paintengine2d.Pt(in.Min.X+u, in.Min.Y+u), Max: in.Max}
	if inner.Dx() > 0 && inner.Dy() > 0 {
		ctx.DrawRect(inner, paintengine2d.Fill(fill))
	}
	return inner
}

// kde2Arrow is HighColor's arrow: a triangle with its tip cut flat — eight
// units across the base, two across the point, five deep.
func kde2Arrow(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	base, deep := b.Dx(), b.Dy()
	if dir == DirLeft || dir == DirRight {
		base, deep = deep, base
	}
	u := max(snap(min(base/8, deep/5)), 1)
	w, d := 8*u, 5*u
	along, across := w, d
	if dir == DirLeft || dir == DirRight {
		along, across = d, w
	}
	if along > b.Dx() || across > b.Dy() {
		return
	}
	x0 := snap(b.Min.X + (b.Dx()-along)*0.5)
	y0 := snap(b.Min.Y + (b.Dy()-across)*0.5)
	p := kde3Path()
	defer kde3Done(p)
	switch dir {
	case DirUp:
		p.MoveTo(x0, y0+d)
		p.LineTo(x0+w, y0+d)
		p.LineTo(x0+5*u, y0)
		p.LineTo(x0+3*u, y0)
	case DirDown:
		p.MoveTo(x0, y0)
		p.LineTo(x0+w, y0)
		p.LineTo(x0+5*u, y0+d)
		p.LineTo(x0+3*u, y0+d)
	case DirLeft:
		p.MoveTo(x0+d, y0)
		p.LineTo(x0+d, y0+w)
		p.LineTo(x0, y0+5*u)
		p.LineTo(x0, y0+3*u)
	default:
		p.MoveTo(x0, y0)
		p.LineTo(x0, y0+w)
		p.LineTo(x0+d, y0+5*u)
		p.LineTo(x0+d, y0+3*u)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// ---- parts --------------------------------------------------------------------------------

func (e kde2Engine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := kde2Colors(l)
	switch role {
	case RoleButton, RoleCombo, RoleTab, RoleThumb, RoleTool:
		c.slab(l, ctx, b, st)
		return c.fgFor(st)
	case RoleField:
		fill := c.base
		if st.Disabled() {
			fill = Mix(c.base, c.bg, 0.5)
		}
		c.well(l, ctx, b, fill, st)
		if st.Disabled() {
			return c.dis
		}
		return c.baseText
	case RoleCheck:
		fill := c.light
		if st.Disabled() {
			fill = Mix(c.light, c.bg, 0.5)
		}
		c.well(l, ctx, b, fill, st)
		return c.fgFor(st)
	case RoleRow:
		if st.Checked() {
			ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.sel))
			return c.selText
		}
		if st.Disabled() {
			return c.dis
		}
		return c.baseText
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
			return c.menuHotText
		}
		if st.Disabled() {
			return c.dis
		}
		return c.text
	case RoleTrack:
		c.well(l, ctx, b, c.groove, st)
		return c.fgFor(st)
	case RoleBar:
		c.barPanel(l, ctx, b)
		return c.text
	case RolePanel:
		ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.bg))
		return c.text
	}
	ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.bg))
	return c.fgFor(st)
}

// CheckIndicator is HighColor's check box: a sunken well filled light,
// with a bold X in it.
func (kde2Engine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	side := snap(min(box.Dx(), box.Dy()))
	r := paintengine2d.XYWH(snap(box.Min.X+(box.Dx()-side)*0.5), snap(box.Min.Y+(box.Dy()-side)*0.5), side, side)
	fill := c.light
	if st.Disabled() {
		fill = Mix(c.light, c.bg, 0.5)
	}
	in := c.well(l, ctx, r, fill, st)
	if !checked {
		return
	}
	DrawCross(ctx, in.Inset(max(u, side*0.14)), c.fgFor(st), max(l.S(2), u))
}

// RadioIndicator is HighColor's radio button: a sunken bead with a small
// round dot.
func (kde2Engine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	side := snap(min(box.Dx(), box.Dy()))
	if side < 6*u {
		return
	}
	cx, cy := snap(box.Min.X+box.Dx()*0.5), snap(box.Min.Y+box.Dy()*0.5)
	rad := side * 0.5
	outline, ring, light, fill := c.outline, c.ring, c.light, c.light
	if st.Disabled() {
		outline, ring, light = Mix(outline, c.bg, 0.5), Mix(ring, c.bg, 0.5), Mix(light, c.bg, 0.4)
		fill = light
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad, paintengine2d.Fill(light))
	// The sunken rim: the outline over the top-left half, the light left
	// showing along the bottom-right.
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(cx-rad, cy-rad, side, side*0.62))
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad, paintengine2d.Fill(outline))
	ctx.Restore()
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad-u, paintengine2d.Fill(ring))
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad-2*u, paintengine2d.Fill(fill))
	if !selected {
		return
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), max(rad*0.3, 2*u), paintengine2d.Fill(c.fgFor(st)))
}

func (kde2Engine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	kde2Arrow(ctx, b, dir, col)
}

// Expander is Qt 2's list view plus box.
func (kde2Engine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	c := kde2Colors(l)
	kde3PlusBox(l, ctx, b, expanded, c.base, c.outline, col)
}

// MenuHighlight is the armed row: the grey ramp under a dark line.
func (kde2Engine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	if r.Dy() < 2*u {
		return
	}
	body := paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), r.Dy()-u)
	ctx.DrawRect(body, VGradient(body, c.menuHot...))
	if !attachBottom {
		ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Max.Y-u, r.Dx(), u), paintengine2d.Fill(c.menuHotLine))
	}
}

func (kde2Engine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	c := kde2Colors(l)
	if hot {
		return c.menuHotText
	}
	return c.text
}

// Qt 2 marked keyboard focus with a dotted rectangle inside the control,
// and a text field with its caret alone.
func (kde2Engine) FieldFocusRing(l *Classic) bool { return false }

func (kde2Engine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := kde2Colors(l)
	kde3Dotted(ctx, kde3Snap(b).Inset(kde3U(l)*2), kde3U(l), c.focus)
}

func (kde2Engine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := kde2Colors(l)
	col := c.focus
	if st.Checked() {
		col = c.selText
	}
	kde3Dotted(ctx, kde3Snap(b), kde3U(l), col)
}

// KDE 2 ran on plain X: nothing floated over a shadow.
func (kde2Engine) PopupShadow(*Classic, PopupKind) Insets { return Insets{} }

func (kde2Engine) DrawPopupShadow(*Classic, *paintengine2d.Context, paintengine2d.Rect, PopupKind) {}

// ---- scroll bars --------------------------------------------------------------------------

// ScrollBarStyle is KDE's own layout: a step-back button at the start and
// both buttons together at the end.
func (kde2Engine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 18, Arrows: ArrowsTripleEnd, MinThumb: 22}
}

func (e kde2Engine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := kde2Colors(l)
	if p.Bar.Empty() {
		return
	}
	ctx.DrawRect(kde3Snap(p.Bar), paintengine2d.Fill(c.bg))
	if !p.Track.Empty() {
		// The groove is the window colour between two dark edge lines.
		u := kde3U(l)
		tr := kde3Snap(p.Track)
		ctx.DrawRect(tr, paintengine2d.Fill(c.outline))
		in := tr
		if vertical {
			in = paintengine2d.XYWH(tr.Min.X+u, tr.Min.Y, tr.Dx()-2*u, tr.Dy())
		} else {
			in = paintengine2d.XYWH(tr.Min.X, tr.Min.Y+u, tr.Dx(), tr.Dy()-2*u)
		}
		if in.Dx() > 0 && in.Dy() > 0 {
			ctx.DrawRect(in, VGradient(in, kde2Ramp(c.groove)...))
		}
	}
	step := func(b paintengine2d.Rect, inc bool, part ControlState) {
		if b.Empty() {
			return
		}
		c.slab(l, ctx, b, part)
		dir := DirUp
		switch {
		case vertical && inc:
			dir = DirDown
		case !vertical && !inc:
			dir = DirLeft
		case !vertical && inc:
			dir = DirRight
		}
		col := c.btnText
		if st.Disabled {
			col = c.dis
		}
		kde2Arrow(ctx, kde3Snap(b), dir, col)
	}
	step(p.Dec, false, st.Part(ScrollDec))
	step(p.DecEnd, false, st.Part(ScrollDecEnd))
	step(p.Inc, true, st.Part(ScrollInc))
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

// handle is the scroll handle: a slab with three ridges across it.
func (c *kde2Look) handle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	u := kde3U(l)
	b = kde3Snap(b)
	if b.Dx() < 5*u || b.Dy() < 5*u {
		return
	}
	c.slab(l, ctx, b, st)
	across, along := b.Dx(), b.Dy()
	if !vertical {
		across, along = b.Dy(), b.Dx()
	}
	if along >= 16*u {
		kde3GripLines(ctx, b, !vertical, 3, 3*u, u, snap(across*0.45), c.grip, c.gripLt)
	}
}

func (e kde2Engine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
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

// ---- frames -------------------------------------------------------------------------------

func (kde2Engine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top = snap(l.body.Height()) + pad*0.5
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

// DrawGroupBox is Qt 2's: a sunken one-pixel frame with the legend sitting
// in its top line.
func (kde2Engine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	top := r.Min.Y
	if title != "" {
		top = snap(r.Min.Y + l.body.Height()*0.5)
	}
	frame := paintengine2d.Rect{Min: paintengine2d.Pt(r.Min.X, top), Max: r.Max}
	hi, lo := c.light, c.ring
	if raised {
		hi, lo = lo, hi
	}
	kde3Frame(ctx, frame, u, paintengine2d.Color{}, lo, hi)
	if title == "" {
		return
	}
	tw := min(l.body.Advance(title)+l.S(6), max(r.Dx()-l.S(10), 0))
	tx := r.Min.X + l.S(8)
	th := snap(l.body.Height())
	ctx.DrawRect(paintengine2d.XYWH(tx, r.Min.Y, tw, th), paintengine2d.Fill(c.bg))
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(tx+l.S(3), r.Min.Y, max(tw-l.S(6), 0), th), c.text, AlignStart, 0)
}

func (kde2Engine) ViewFrameInsets(l *Classic) Insets {
	u := kde3U(l)
	return Insets{Top: 2 * u, Right: 2 * u, Bottom: 2 * u, Left: 2 * u}
}

// ---- tabs ---------------------------------------------------------------------------------

func (kde2Engine) TabOutset(l *Classic) Insets { return Insets{} }

// kde2PaneTop is where the page frame begins: the tab bar's last row.
func kde2PaneTop(l *Classic, b paintengine2d.Rect) float32 {
	th := l.metrics.TabH
	if th <= 0 || b.Dy() < th*2 {
		return b.Min.Y
	}
	return snap(b.Min.Y + th - kde3U(l))
}

// DrawTabPane is the page under the tabs: the window colour in a raised
// one-pixel frame.
func (kde2Engine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, kde2PaneTop(l, b)), Max: b.Max})
	kde3Frame(ctx, r, u, c.bg, c.light, c.ring)
}

func (kde2Engine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := kde2Colors(l)
	ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.bg))
}

// kde2TabPath is the "rounded above" shape: the top-left corner cut at 45°
// over four units, the top-right square.
func kde2TabPath(p *paintengine2d.Path, r paintengine2d.Rect, cut float32) {
	p.MoveTo(r.Min.X, r.Max.Y)
	p.LineTo(r.Min.X, r.Min.Y+cut)
	p.LineTo(r.Min.X+cut, r.Min.Y)
	p.LineTo(r.Max.X, r.Min.Y)
	p.LineTo(r.Max.X, r.Max.Y)
	p.Close()
}

// DrawTab: the selected tab is the window colour and stands a pixel
// taller; the rest sit in the mid shade under the same chamfered top.
func (e kde2Engine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	if !selected {
		r = paintengine2d.XYWH(r.Min.X+u, r.Min.Y+u, r.Dx()-2*u, r.Dy()-u)
	}
	cut := snap(min(l.S(4), r.Dy()*0.4))
	outline, light := c.outline, c.light
	if st.Disabled() {
		outline, light = Mix(outline, c.bg, 0.5), Mix(light, c.bg, 0.4)
	}
	p := kde3Path()
	kde2TabPath(p, r, cut)
	ctx.DrawPath(p, paintengine2d.Fill(outline))
	kde3Done(p)
	in := paintengine2d.Rect{Min: paintengine2d.Pt(r.Min.X+u, r.Min.Y+u), Max: paintengine2d.Pt(r.Max.X-u, r.Max.Y)}
	if in.Dx() > 0 && in.Dy() > 0 {
		q := kde3Path()
		kde2TabPath(q, in, max(cut-u, 0))
		fill := paintengine2d.Fill(c.bg)
		switch {
		case selected:
			fill = paintengine2d.Fill(c.bg)
		case st.Hovered() && !st.Disabled():
			fill = paintengine2d.Fill(c.midlight)
		default:
			fill = paintengine2d.Fill(c.mid)
		}
		ctx.DrawPath(q, fill)
		kde3Done(q)
		// The lit top and left edge.
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X+cut, in.Min.Y, in.Dx()-cut, u), paintengine2d.Fill(light))
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y+cut, u, in.Dy()-cut), paintengine2d.Fill(light))
	}
	fg := c.fgFor(st)
	tb := paintengine2d.XYWH(r.Min.X+l.S(6), r.Min.Y, r.Dx()-l.S(12), r.Dy())
	l.drawFittedText(ctx, l.body, label, tb, fg, AlignCenter, 0)
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, r.Inset(l.S(3)), u, c.focus)
	}
}

// ---- bars ---------------------------------------------------------------------------------

// ControlFont: KDE 2 read tool bars at 10 points against the general 12.
func (kde2Engine) ControlFont(l *Classic, role Role) *Font {
	if role == RoleTool && l.P("toolFont", 0.83) > 0 {
		return kde3ControlFont(l, role)
	}
	return l.body
}

func (kde2Engine) ToolBarInsets(l *Classic) Insets {
	return Insets{Top: l.S(3), Right: l.S(4), Bottom: l.S(3), Left: l.S(14)}
}

// barPanel is a menu bar or tool bar: a shallow raised panel over the
// window colour's own gradient.
func (c *kde2Look) barPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	u := kde3U(l)
	r := kde3Snap(b)
	if r.Dx() < 2*u || r.Dy() < 2*u {
		return
	}
	ctx.DrawRect(r, paintengine2d.Fill(c.ring))
	in := paintengine2d.Rect{Min: r.Min, Max: paintengine2d.Pt(r.Max.X-u, r.Max.Y-u)}
	ctx.DrawRect(in, paintengine2d.Fill(c.light))
	face := r.Inset(u)
	if face.Dx() > 0 && face.Dy() > 0 {
		ctx.DrawRect(face, VGradient(face, c.barFace...))
	}
}

func (kde2Engine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	kde2Colors(l).barPanel(l, ctx, b)
}

func (kde2Engine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	c.barPanel(l, ctx, r)
	// The drag grip: three ridges, each a light line beside a mid one.
	if r.Dy() >= 10*u {
		h := paintengine2d.XYWH(r.Min.X+l.S(3), r.Min.Y, l.S(10), r.Dy())
		kde3GripLines(ctx, h, true, 3, 2*u, u, r.Dy()-l.S(8), c.ring, c.light)
	}
}

// DrawToolButton: flat until the pointer is on it, then the slab.
func (e kde2Engine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := kde2Colors(l)
	kde3ToolButton(l, ctx, b, st, label, icon, c.dis)
}

func (e kde2Engine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := kde2Colors(l)
	r := kde3Snap(b)
	fg := c.text
	if armed := !st.Disabled() && (open || st.Pressed() || st.Hovered()); armed {
		e.MenuHighlight(l, ctx, r, true)
		fg = c.menuHotText
	} else if st.Disabled() {
		fg = c.dis
	}
	l.drawLabeled(ctx, l.body, label, underline, r, fg)
}

// DrawMenuFrame is Qt 2's pull-down: the window colour in a raised
// one-pixel frame.
func (kde2Engine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := kde2Colors(l)
	u := kde3U(l)
	kde3Frame(ctx, kde3Snap(b), u, c.bg, c.light, c.outline)
}

func (e kde2Engine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	hi, lo := c.light, c.ring
	if !raised {
		hi, lo = lo, hi
	}
	kde3Frame(ctx, kde3Snap(b), u, c.bg, hi, lo)
}

func (e kde2Engine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := kde2Colors(l)
	kde3StatusBar(l, ctx, b, parts, c.bg, c.ring, c.light, c.text)
}

func (e kde2Engine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := kde2Colors(l)
	kde3Heading(l, ctx, b, title, subtitle, c.bg, c.ring, c.light, c.text, c.dis)
}

func (e kde2Engine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := kde2Colors(l)
	kde3Separator(l, ctx, b, vertical, c.ring, c.light)
}

func (e kde2Engine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := kde2Colors(l)
	u := kde3U(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
	if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(c.sel.WithAlpha(0.18)))
	}
	length := snap(min(l.S(20), max(b.Dx(), b.Dy())*0.5))
	if vertical && b.Dx() >= 4*u {
		kde3GripLines(ctx, b, true, 3, 2*u, u, length, c.ring, c.light)
	} else if !vertical && b.Dy() >= 4*u {
		kde3GripLines(ctx, b, false, 3, 2*u, u, length, c.ring, c.light)
	}
}

func (e kde2Engine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := kde2Colors(l)
	r := kde3Snap(b)
	c.slab(l, ctx, r, st&^StateFocused)
	fg := c.fgFor(st)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	kde2Arrow(ctx, paintengine2d.XYWH(r.Min.X+l.S(6), r.Min.Y, l.S(12), r.Dy()), dir, fg)
	lb := paintengine2d.XYWH(r.Min.X+l.S(24), r.Min.Y, r.Dx()-l.S(28), r.Dy())
	l.drawFittedText(ctx, l.body, title, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, labelFocusRect(l.body, title, lb, r), kde3U(l), c.focus)
	}
}

// ---- lists, trees, tables -----------------------------------------------------------------

func (e kde2Engine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := kde2Colors(l)
	fg := c.baseText
	if st.Checked() {
		ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.sel))
		fg = c.selText
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(6), b.Min.Y, b.Dx()-l.S(10), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e kde2Engine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := kde2Colors(l)
	kde3TreeRow(l, ctx, b, st, expanded, leaf, depth, label, bold, kde3TreeColors{
		mid: c.ring, text: c.baseText, dis: c.dis, sel: c.sel, selText: c.selText, alt: c.alt, base: c.base,
	}, e.Expander, e.ItemFocus)
}

func (e kde2Engine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := kde2Colors(l)
	fg := c.baseText
	if st.Checked() {
		ctx.DrawRect(kde3Snap(b), paintengine2d.Fill(c.sel))
		fg = c.selText
	} else if st.Disabled() {
		fg = c.dis
	}
	kde3CellText(l, ctx, b, label, align, face, fg)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e kde2Engine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := kde2Colors(l)
	r := kde3Snap(b)
	c.slab(l, ctx, r, st&^StateFocused)
	fg := c.fgFor(st)
	right := r.Max.X - l.S(6)
	if sorted {
		s := l.S(10)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		kde2Arrow(ctx, paintengine2d.XYWH(right-s, r.Min.Y, s, r.Dy()), dir, fg)
		right -= s + l.S(3)
	}
	lb := paintengine2d.XYWH(r.Min.X+l.S(7), r.Min.Y, max(right-r.Min.X-l.S(7), 0), r.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

// ---- the rest -----------------------------------------------------------------------------

// DrawProgressBar: HighColor filled a progress bar with plain solid
// blocks — no bevel, no gradient.
func (e kde2Engine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := kde2Colors(l)
	in := c.well(l, ctx, b, c.base, st)
	if in.Dx() <= 0 || in.Dy() <= 0 {
		return
	}
	fill := c.sel
	if st.Disabled() {
		fill = Mix(fill, c.bg, 0.55)
	}
	if indeterminate {
		w := in.Dx() * 0.3
		x0 := max(in.Min.X+(in.Dx()+w)*phase-w, in.Min.X)
		x1 := min(in.Min.X+(in.Dx()+w)*phase, in.Max.X)
		if x1 > x0 {
			ctx.DrawRect(kde3Snap(paintengine2d.XYWH(x0, in.Min.Y, x1-x0, in.Dy())), paintengine2d.Fill(fill))
		}
		return
	}
	if w := snap(in.Dx() * clamp01(t)); w > 0 {
		ctx.DrawRect(paintengine2d.XYWH(in.Min.X, in.Min.Y, w, in.Dy()), paintengine2d.Fill(fill))
	}
}

// DrawSlider is Qt 2's QSlider: an etched groove across the middle and a
// slab handle carrying three ridges.
func (e kde2Engine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	gh := snap(max(l.S(6), 3*u))
	groove := paintengine2d.XYWH(r.Min.X, snap(r.Min.Y+(r.Dy()-gh)*0.5), r.Dx(), gh)
	c.well(l, ctx, groove, c.groove, st)
	hw := snap(max(l.S(13), 7*u))
	hh := snap(min(r.Dy(), max(l.S(24), 12*u)))
	x := snap(r.Min.X + (r.Dx()-hw)*clamp01(t))
	handle := paintengine2d.XYWH(x, snap(r.Min.Y+(r.Dy()-hh)*0.5), hw, hh)
	c.slab(l, ctx, handle, st)
	if hh >= 12*u {
		kde3GripLines(ctx, handle, false, 3, 2*u, u, snap(hw*0.45), c.ring, c.light)
	}
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, r, u, c.focus)
	}
}

// DrawButton is the slab with the label centred on it; the default button
// wears an extra outline, its corners cut like everything else.
func (e kde2Engine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	body := r
	if st.Primary() {
		kde2CutFrame(ctx, r, u, c.outline, l.square())
		body = r.Inset(u)
	}
	c.slab(l, ctx, body, st)
	fg := c.fgFor(st)
	lb := body
	if st.Pressed() && !st.Disabled() {
		lb = lb.Translate(paintengine2d.Pt(u, u))
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignCenter, l.S(8))
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, body.Inset(snap(l.S(4))), u, c.focus)
	}
}

// DrawSwitch: KDE 2 had no switch, so it is drawn in the style's own
// language — a sunken groove with a slab knob that slides to the right and
// fills the groove behind it with the selection colour.
func (e kde2Engine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := kde2Colors(l)
	u := kde3U(l)
	m := l.metrics
	tw, th := min(m.SwitchW, b.Dx()), min(m.SwitchH, b.Dy())
	track := kde3Snap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th))
	if track.Dx() < 10*u || track.Dy() < 6*u {
		return
	}
	g := kde3Snap(paintengine2d.XYWH(track.Min.X, snap(track.Min.Y+track.Dy()*0.25), track.Dx(), snap(track.Dy()*0.5)))
	fill := c.groove
	if on && !st.Disabled() {
		fill = c.sel
	}
	c.well(l, ctx, g, fill, st)
	kw := snap(track.Dy())
	kx := track.Min.X
	if on {
		kx = track.Max.X - kw
	}
	c.slab(l, ctx, paintengine2d.XYWH(kx, track.Min.Y, kw, track.Dy()), st&^StateToggle)
	fg := c.text
	if st.Disabled() {
		fg = c.dis
	}
	if label == "" {
		if st.Focused() && !st.Disabled() {
			kde3Dotted(ctx, track, u, c.focus)
		}
		return
	}
	lb := paintengine2d.XYWH(track.Max.X+l.S(8), b.Min.Y, b.Max.X-track.Max.X-l.S(8), b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, labelFocusRect(l.body, label, lb, b), u, c.focus)
	}
}

// DrawComboBox is a slab with a flat-topped arrow at its right.
func (e kde2Engine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := kde2Colors(l)
	u := kde3U(l)
	r := kde3Snap(b)
	cs := st
	if open {
		cs |= StatePressed
	}
	c.slab(l, ctx, r, cs)
	fg := c.fgFor(st)
	aw := snap(max(r.Dy()/3, l.S(11)))
	ab := paintengine2d.XYWH(r.Max.X-aw-l.S(6), r.Min.Y, aw, r.Dy())
	if ab.Min.X > r.Min.X+l.S(10) {
		kde2Arrow(ctx, ab, DirDown, fg)
	}
	tb := paintengine2d.XYWH(r.Min.X+l.S(8), r.Min.Y, max(ab.Min.X-r.Min.X-l.S(12), 0), r.Dy())
	if tb.Dx() > 0 {
		l.drawFittedText(ctx, l.body, text, tb, fg, AlignStart, 0)
	}
	if st.Focused() && !st.Disabled() {
		kde3Dotted(ctx, r.Inset(l.S(4)), u, c.focus)
	}
}

// TooltipStyle and DrawTooltip: Qt 2's pale yellow tip in a one-pixel
// black frame.
func (kde2Engine) TooltipStyle(l *Classic) TooltipStyle {
	return l.tipStyle(l.body, l.tipPad(l.S(4)), AlignStart)
}

func (kde2Engine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := kde2Colors(l)
	kde3Tooltip(l, ctx, b, text, c.tip, c.tipText)
}

// ---- packs --------------------------------------------------------------------------------

func kde2Packs() []ThemePack {
	white, black := paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0)
	// KDE 2's default colour scheme, as its Control Center wrote it.
	bg, btn, base := Hex("#dcdcdc"), Hex("#e4e4e4"), white
	sel := Hex("#0a5f89")
	pal := Palette{
		Background: bg, Surface: bg, SurfaceAlt: btn,
		Border: Hex("#4e4e4e"), Divider: Hex("#b7b7b7"),
		Text: black, TextMuted: Hex("#808080"), TextOnAccent: white,
		Accent: Hex("#0000c0"), AccentHover: lighterPct(Hex("#0000c0"), 130), AccentPress: darkerPct(Hex("#0000c0"), 120),
		Field: base, FieldBorder: Hex("#4e4e4e"),
		Focus: black, Selection: sel,
		Track: bg, Thumb: btn,
		Highlight: white.WithAlpha(0.6), Shadow: paintengine2d.RGBA(0, 0, 0, 0.25),
		MenuHover: Hex("#b7b7b7"), MenuHoverBorder: Hex("#4e4e4e"), MenuGutter: bg,
		Danger: Hex("#c00000"), Success: Hex("#008000"), Warning: Hex("#c08000"),
		Overlay:    paintengine2d.RGBA(0, 0, 0, 0.3),
		BevelLight: white, BevelDark: Hex("#4e4e4e"),
	}
	tok := ThemeTokens{
		Engine:  "kde2",
		Bevel:   BevelClassic3D,
		Family:  ThemeLight,
		Palette: pal,
		Era:     EraKDE2,
		Extra: map[string]paintengine2d.Color{
			"window": bg, "button": btn, "buttonText": black,
			"base": base, "baseText": black, "alternate": Hex("#f0f0f0"),
			"disabledText": Hex("#808080"),
			"light":        white, "outline": Hex("#4e4e4e"), "ring": Hex("#b7b7b7"),
			"selection": sel, "selectionText": white,
			"caption": sel, "captionText": white,
			"captionOff": bg, "captionOffText": black,
			"tip": Hex("#ffffdc"), "tipText": black,
		},
		Params: map[string]float32{"titleStipple": 1, "toolFont": 0.83},
	}
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHoverBorder}
	tok.Selected = ChromeState{Fill: sel, Border: sel}
	tok.Focus = ChromeState{Fill: sel.WithAlpha(0.15), Border: black}
	return []ThemePack{{
		Name: "kde2", Label: "KDE 2", Year: 2000, Lineage: "KDE",
		Summary: "KDE 2's HighColor: slabs with their corners cut and a white-lit top, flat-topped arrows, three-button scroll bars and a woven blue title bar.",
		Era:     EraKDE2, Palette: ThemeLight, Tokens: tok,
	}}
}
