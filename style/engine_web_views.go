package style

import (
	"github.com/codemodify/paintengine2d"
)

// Tabs, menus, item views, bars and the accent of the web engine (see
// engine_web.go).

// ---- tabs ---------------------------------------------------------------------------------------

// DrawTabBar is the strip under the tabs: the tab page's colour with a
// hairline under underlined tabs, the window under segmented ones, the
// title bar (its bottom edge the tool bar's top) under browser tabs.
func (webEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	b = winSnap(b)
	if b.Empty() || c.ideTabBar(l, ctx, b) {
		return
	}
	px := c.px(l)
	switch c.tabStyle {
	case webTabBrowser:
		ctx.DrawRect(b, paintengine2d.Fill(c.titleBar))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border0))
	case webTabSegmented:
		ctx.DrawRect(b, paintengine2d.Fill(c.window))
	default:
		ctx.DrawRect(b, paintengine2d.Fill(c.tabPane))
		if c.tabBar {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border2))
		}
	}
}

func (e webEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := webColors(l)
	switch c.tabStyle {
	case webTabBrowser:
		e.DrawBrowserTab(l, ctx, b, st, label, selected)
	case webTabSegmented:
		c.segmentTab(l, ctx, winSnap(b), st, label, selected)
	case webTabEditor:
		c.editorTab(l, ctx, winSnap(b), st, label, selected)
	case webTabPill:
		c.pillTab(l, ctx, winSnap(b), st, label, selected)
	default:
		c.underlineTab(l, ctx, winSnap(b), st, label, selected)
	}
}

// underlineTab is a label over the tab line: the selected label in the
// accent (SourceGit) or the text colour, the others muted, brightening under
// the pointer; the line under the selected tab's label or across the tab.
func (c *webSet) underlineTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	px := c.px(l)
	f := c.tabFace
	if selected {
		f = c.tabSelFace
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	fg := c.tabText2
	switch {
	case st.Disabled():
		fg = c.textDis
	case selected:
		fg = c.tabText
	case hot:
		fg = c.text
	}
	tw := min(f.Advance(label), max(b.Dx()-l.S(12), 0))
	lb := paintengine2d.XYWH(snap((b.Min.X+b.Max.X-tw)*0.5), b.Min.Y, snap(tw), b.Dy())
	if c.tabHover && hot {
		hb := paintengine2d.XYWH(lb.Min.X-l.S(8), b.Min.Y+l.S(5), lb.Dx()+l.S(16), b.Dy()-l.S(10)).Intersect(b)
		r := min(c.rad(l, c.radius), hb.Dy()*0.5)
		if !hb.Empty() {
			ctx.DrawRoundRect(hb, r, r, paintengine2d.Fill(c.wash))
		}
	}
	l.drawFittedText(ctx, f, label, b, fg, AlignCenter, l.S(8))
	if selected {
		h := max(snap(l.S(c.tabLineW)), px)
		y := b.Max.Y - h - snap(l.S(c.tabLineGap))
		x0, x1 := b.Min.X+snap(l.S(2)), b.Max.X-snap(l.S(2))
		if c.tabFit {
			x0, x1 = lb.Min.X, lb.Max.X
		}
		if x1 > x0 && y >= b.Min.Y {
			r := min(h*0.5, c.rad(l, c.radius))
			ctx.DrawRoundRect(paintengine2d.XYWH(x0, y, x1-x0, h), r, r, paintengine2d.Fill(c.disabled(c.tabLine, st)))
		}
	}
	if st.Focused() && !st.Disabled() {
		if c.focusStyle == webFocusDotted {
			webDots(ctx, b.Inset(px), c.text, winPx(l))
			return
		}
		fb := paintengine2d.XYWH(lb.Min.X-l.S(6), b.Min.Y+l.S(4), lb.Dx()+l.S(12), b.Dy()-l.S(8)).Intersect(b)
		c.focusRing(l, ctx, fb, fb, min(c.rad(l, c.radius), fb.Dy()*0.5))
	}
}

// segmentTab is one segment of a segmented tab strip (shadcn's TabsList): a
// muted track, rounded at the strip's ends, the selected tab a raised knob
// in the window colour.
func (c *webSet) segmentTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	if b.Dx() < 6 || b.Dy() < 8 {
		return
	}
	r := min(c.rad(l, c.radius+2), b.Dy()*0.5)
	rl, rr := float32(0), float32(0)
	if st.First() {
		rl = r
	}
	if st.Last() {
		rr = r
	}
	ctx.DrawPath(RoundRectPath(b, rl, rr, rr, rl), paintengine2d.Fill(c.segTrack))
	fg := c.tabText2
	knob := b.Inset(snap(l.S(3)))
	kr := min(c.rad(l, c.radius), knob.Dy()*0.5)
	switch {
	case st.Disabled():
		fg = c.textDis
	case selected:
		ctx.Save()
		ctx.ClipRect(b)
		DropShadow(ctx, knob, kr, paintengine2d.RGBA(0, 0, 0, 0.1), 0, l.S(1), l.S(3), 0)
		ctx.Restore()
		c.box(l, ctx, knob, kr, c.segOn, l.X("segOnBorder", paintengine2d.Color{}))
		fg = c.segOnText
	case st.Hovered() || st.Pressed():
		fg = c.text
	}
	if selected && st.Disabled() {
		c.box(l, ctx, knob, kr, webA(c.segOn, c.disA), paintengine2d.Color{})
	}
	l.drawFittedText(ctx, c.tabFace, label, knob, fg, AlignCenter, l.S(8))
	if st.Focused() && selected && !st.Disabled() {
		if c.focusStyle == webFocusDotted {
			webDots(ctx, knob, c.text, winPx(l))
			return
		}
		_, w := c.ring(l)
		if knob.Dx() > 4*w && knob.Dy() > 4*w {
			winRing(ctx, knob, kr, w, paintengine2d.Fill(webA(c.focus, c.focusA)))
		}
	}
}

// BrowserTabOutset: the selected tab's concave feet, as wide as its top
// corners are round (5px), reach past its slot.
func (webEngine) BrowserTabOutset(l *Classic) Insets {
	e := snap(l.S(5))
	return Insets{Left: e, Right: e}
}

// DrawBrowserTab is a title bar's document tab: the selected one a single
// outline in the tool bar's colour with 5px convex top corners and concave
// feet that run into the tool bar below it; the others bare, divided by thin
// separators that stop beside the selected tab; labels a size smaller, at
// half strength, brighter under the pointer, full on the selected tab.
func (e webEngine) DrawBrowserTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	if b.Dx() < 8*px || b.Dy() < 8*px {
		return
	}
	ear := float32(0)
	if selected {
		ear = min(e.BrowserTabOutset(l).Left, b.Dx()*0.2)
	}
	top := b.Min.Y + max(snap(b.Dy()-l.S(30)), 0)
	body := paintengine2d.XYWH(b.Min.X+ear, top, b.Dx()-2*ear, b.Max.Y-top)
	r := min(c.rad(l, 5), body.Dx()*0.5, body.Dy()*0.5)
	fg := webA(c.text, 0.5)
	switch {
	case selected:
		if ear > 0 {
			// Clear the neighbours' separators under the feet.
			clr := paintengine2d.NewPath()
			clr.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, ear, b.Dy()-ear-px))
			clr.AddRect(paintengine2d.XYWH(b.Max.X-ear, b.Min.Y, ear, b.Dy()-ear-px))
			ctx.DrawPath(clr, paintengine2d.Fill(c.titleBar))
		}
		ctx.DrawPath(fluentTabPath(b, top, r, ear, true), paintengine2d.Fill(c.toolBar))
		ctx.DrawPath(fluentTabPath(b.Inset(px*0.5), top+px*0.5, r, ear, false), paintengine2d.Paint{Color: c.border0, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: px, Cap: paintengine2d.CapButt, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		fg = c.text
	case st.Disabled():
		fg = webA(c.text, 0.3)
	default:
		if st.Hovered() || st.Pressed() {
			fg = webA(c.text, 0.85)
		}
		if !st.Last() {
			h := min(snap(l.S(18)), body.Dy()-4*px)
			sep := l.X("tabSep", webA(c.text, 0.2))
			ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, snap((body.Min.Y+body.Max.Y-h)*0.5), px, h), paintengine2d.Fill(sep))
		}
	}
	l.drawFittedText(ctx, c.smallFace, label, body, fg, AlignCenter, l.S(12))
	if st.Focused() && !st.Disabled() {
		fb := body.Inset(snap(l.S(2)))
		if c.focusStyle == webFocusDotted {
			webDots(ctx, fb, c.text, winPx(l))
		} else if fb.Dx() > 8*px && fb.Dy() > 8*px {
			_, w := c.ring(l)
			winRing(ctx, fb, max(r-l.S(2), 0), w, paintengine2d.Fill(webA(c.focus, c.focusA)))
		}
	}
}

// DrawTabPane is the page under the tabs, in the tab page's colour (an
// island where the look has them).
func (webEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	if c.islandPane(l, ctx, b) {
		return
	}
	fill := c.tabPane
	if c.tabStyle == webTabBrowser {
		fill = c.toolBar
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
}

// ---- menus --------------------------------------------------------------------------------------

func (webEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	b = winSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.titleBar))
	px := c.px(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border0))
}

// DrawMenuTitle: a rounded wash under the pointer, the menu highlight while
// its menu is open.
func (e webEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := webColors(l)
	b = winSnap(b)
	hb := b.Inset(snap(l.S(2)))
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.textDis
	case open || st.Pressed():
		e.MenuHighlight(l, ctx, hb, true)
		fg = c.menuText
	case st.Hovered():
		if !hb.Empty() {
			r := min(c.rad(l, c.menuRowR), hb.Dy()*0.5)
			ctx.DrawRoundRect(hb, r, r, paintengine2d.Fill(c.wash))
		}
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open && !st.Disabled() && !hb.Empty() {
		c.focusRing(l, ctx, hb, hb, min(c.rad(l, c.menuRowR), hb.Dy()*0.5))
	}
}

// DrawMenuFrame is a popover: the popup colour in its hairline, rounded.
func (webEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := webColors(l)
	b = winSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	c.box(l, ctx, b, min(c.rad(l, c.overlayR), b.Dy()*0.25), c.popup, c.popupBorder)
}

// DrawMenuItem is a menu row: a rounded highlight inset from the popover's
// sides, checks, bullets and icons in the leading column, shortcuts and the
// submenu chevron in the muted colour.
func (e webEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := webColors(l)
	ch := MenuChromeFor(l)
	px := c.px(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := b.Min.X-ch.PadL+px, b.Max.X+ch.PadR-px
		if c.menuSepLabel {
			x0, x1 = b.Min.X+ch.CheckCol(), b.Max.X-l.S(4)
		}
		if x1 > x0 {
			ctx.DrawRect(paintengine2d.XYWH(snap(x0), y, snap(x1-x0), px), paintengine2d.Fill(c.border2))
		}
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		k := min(snap(l.S(c.menuInset)), ch.PadL)
		e.MenuHighlight(l, ctx, paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy()), false)
	}
	fg, sc := c.text, c.text2
	switch {
	case st.Disabled():
		fg, sc = c.textDis, c.textDis
	case hot:
		fg = c.menuText
		if c.menuAccent {
			sc = c.menuText
		}
	}
	gw := ch.CheckCol()
	side := snap(min(b.Dy()-2*px, gw-2*px, l.S(16)))
	if side >= 4 {
		box := winSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
		switch {
		case row.Checked && row.Radio:
			ctx.DrawCircle(paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5), l.S(3), paintengine2d.Fill(fg))
		case row.Checked:
			webTick(ctx, box.Inset(side*0.1), fg, max(l.S(1.6), 1.2))
		case !row.Radio && row.Icon != IconNone:
			l.drawToolIcon(ctx, box, row.Icon, fg)
		}
	}
	winMenuText(l, ctx, b, ch, row, fg, sc, func(ab paintengine2d.Rect) {
		e.Arrow(l, ctx, ab, DirRight, sc)
	})
}

// ---- rows ---------------------------------------------------------------------------------------

// rowFill is the highlight of a row of kind k in item state st: the
// selection colour at the pack's alpha for a focused view, under the pointer
// and without focus (or the neutral wash, SourceGit's sidebars), the wash
// under the pointer; zero for none.
func (c *webSet) rowFill(st ControlState, k int) paintengine2d.Color {
	rs := &c.rows[k]
	switch {
	case st.Checked():
		if st.Inactive() || st.Backdrop() || st.Disabled() {
			if rs.offWash {
				return webA(c.wash, rs.selOff)
			}
			return webA(c.sel, rs.selOff)
		}
		if st.Hovered() || st.Pressed() {
			return webA(c.sel, rs.selHover)
		}
		return webA(c.sel, rs.sel)
	case st.Disabled():
		return paintengine2d.Color{}
	case st.Pressed():
		return webA(c.wash, rs.hover*2)
	case st.Hovered():
		return webA(c.wash, rs.hover)
	}
	return paintengine2d.Color{}
}

// rowBox is the box a row's highlight fills inside the row b for kind k and
// its corner radii (tl, tr, br, bl): the whole row, or a box inset from the
// sides with rounded corners, square and flush where the row above or below
// is selected too, so a run of selected rows is one block.
func (c *webSet) rowBox(l *Classic, b paintengine2d.Rect, st ControlState, k int) (paintengine2d.Rect, [4]float32) {
	rs := &c.rows[k]
	in := snap(l.S(rs.inset))
	if in <= 0 && rs.gap <= 0 {
		return b, [4]float32{}
	}
	g := snap(l.S(rs.gap))
	r := float32(0)
	if in > 0 {
		r = c.rad(l, rs.radius)
	}
	top, bot := g, g
	tl, tr, br, bl := r, r, r, r
	if st.Checked() && st.SelectedAbove() {
		top, tl, tr = 0, 0, 0
	}
	if st.Checked() && st.SelectedBelow() {
		bot, br, bl = 0, 0, 0
	}
	box := paintengine2d.XYWH(b.Min.X+in, b.Min.Y+top, b.Dx()-2*in, b.Dy()-top-bot)
	if box.Dx() < 2 || box.Dy() < 2 {
		return b, [4]float32{}
	}
	return box, [4]float32{tl, tr, br, bl}
}

// viewBg is what a row of kind k sits on.
func (c *webSet) viewBg(k int) paintengine2d.Color {
	if k == webSide {
		return c.sidebar
	}
	return c.field
}

// row paints row b's highlight for kind k (and the accent bar of a
// selected row) and returns the label colour that reads on it.
func (c *webSet) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, k int) paintengine2d.Color {
	fill := c.rowFill(st, k)
	box, rad := c.rowBox(l, b, st, k)
	if fill.A > 0 && !box.Empty() {
		if rad == [4]float32{} {
			ctx.DrawRect(box, paintengine2d.Fill(fill))
		} else {
			ctx.DrawPath(RoundRectPath(box, rad[0], rad[1], rad[2], rad[3]), paintengine2d.Fill(fill))
		}
	}
	if st.Checked() && c.rowBarW > 0 && k != webTable {
		w := max(snap(l.S(c.rowBarW)), 1)
		h := box.Dy() - 2*snap(l.S(6))
		if h > w && box.Dx() > 4*w {
			ctx.DrawRoundRect(paintengine2d.XYWH(box.Min.X, snap((box.Min.Y+box.Max.Y-h)*0.5), w, h), w*0.5, w*0.5, paintengine2d.Fill(c.disabled(c.rowBar, st)))
		}
	}
	return c.rowLabel(st, k)
}

// rowLabel is the colour of a row's label: the text where it reads on the
// row's highlight, else the text on the accent.
func (c *webSet) rowLabel(st ControlState, k int) paintengine2d.Color {
	if st.Disabled() {
		return c.textDis
	}
	if fill := c.rowFill(st, k); fill.A > 0 {
		return c.label(webOver(c.viewBg(k), fill))
	}
	return c.text
}

// itemFocus marks the current row: the dotted adorner, or a ring inside
// the row's own box (not the joined block).
func (c *webSet) itemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, k int) {
	box, rad := c.rowBox(l, b, st&^(StateSelectedAbove|StateSelectedBelow), k)
	px := winPx(l)
	if box.Dx() < 4*px || box.Dy() < 4*px {
		return
	}
	if c.focusStyle == webFocusDotted {
		webDots(ctx, box, c.text, px)
		return
	}
	_, w := c.ring(l)
	w = min(w, 2*px)
	col := webA(c.focus, c.focusA)
	if fill := c.rowFill(st, k); st.Checked() && fill.A > 0 {
		// A ring lost on the selection (the accent on the accent) takes
		// the label's colour.
		bg := webOver(c.viewBg(k), fill)
		if ContrastRatio(webOver(bg, col), bg) < 1.5 {
			col = c.rowLabel(st&^StateFocused, k)
		}
	}
	winRing(ctx, box, rad[0], w, paintengine2d.Fill(col))
}

func (e webEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := webColors(l)
	k := webList
	if st.Sidebar() {
		k = webSide
	}
	b = winSnap(b)
	fg := c.row(l, ctx, b, st, k)
	box, _ := c.rowBox(l, b, st, k)
	pad := box.Min.X - b.Min.X + l.S(8)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-l.S(6), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		c.itemFocus(l, ctx, b, st, k)
	}
}

func (e webEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := webColors(l)
	k := webList
	if st.Sidebar() {
		k = webSide
	}
	b = winSnap(b)
	fg := c.row(l, ctx, b, st, k)
	box, _ := c.rowBox(l, b, st, k)
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := box.Min.X + l.S(4) + float32(depth)*indent
	if !leaf {
		col := c.text2
		if c.triangles {
			col = c.text
		}
		if st.ExpanderHot() || (st.Checked() && fg != c.text) {
			col = fg
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, c.disabled(col, st))
	}
	f := l.body
	if bold {
		f = c.bold
	}
	lx := x + indent + l.S(2)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, b.Max.X-lx-l.S(4), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		c.itemFocus(l, ctx, b, st, k)
	}
}

// DrawTableHeader: the header's colour, a hairline under it and (where the
// pack draws them) at each column's end, the label in the header's weight,
// a wash under the pointer.
func (e webEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := webColors(l)
	b = winSnap(b)
	if b.Empty() {
		return
	}
	px := c.px(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.header))
	if !st.Disabled() && (st.Hovered() || st.Pressed()) {
		a := float32(1)
		if st.Pressed() {
			a = 1.8
		}
		ctx.DrawRect(b, paintengine2d.Fill(webA(c.wash, a)))
	}
	line := l.X("headerLine", c.border0)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(line))
	if l.P("headerSep", 1) != 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-px, b.Min.Y, px, b.Dy()-px), paintengine2d.Fill(line))
	}
	fg := c.headerText
	if st.Disabled() {
		fg = c.textDis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(14)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		e.Arrow(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, fg)
	}
	align := AlignStart
	if c.headerCenter {
		align = AlignCenter
	}
	l.drawFittedText(ctx, c.headFace, label, paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, b.Dx()-l.S(12)-aw, b.Dy()-px), fg, align, 0)
}

// DrawTableCell: a table selects whole rows, the cells of a row one band.
func (e webEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := webColors(l)
	cell := winSnap(b)
	fill := c.rowFill(st, webTable)
	if fill.A > 0 && !cell.Empty() {
		ctx.DrawRect(cell, paintengine2d.Fill(fill))
	}
	fg := c.text
	switch {
	case st.Disabled():
		fg = c.textDis
	case fill.A > 0:
		fg = c.label(webOver(c.field, fill))
	}
	winCellText(l, ctx, b, label, align, face, fg)
}

// ---- bars ---------------------------------------------------------------------------------------

func (webEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(webColors(l).toolBar))
}

func (webEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.window))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), px), paintengine2d.Fill(c.border0))
	winStatusParts(l, ctx, b, parts, c.text2, c.border2, paintengine2d.Color{})
}

// DrawTitleBar is a caption strip: the title bar colour over its hairline,
// the title in bold (centred where the pack centres captions), the
// subtitle muted.
func (webEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := webColors(l)
	b = winSnap(b)
	px := c.px(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.titleBar))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border0))
	if c.captionCenter && subtitle == "" && title != "" {
		l.drawFittedText(ctx, c.bold, title, b, c.text, AlignCenter, l.S(16))
		return
	}
	winHeading(l, ctx, b, title, subtitle, c.text, c.text2, true)
}

// DrawAccordionHeader: SourceGit's group header (a chevron at the left and
// a bold muted label) or a web accordion (the label, a chevron at the right,
// a divider under it).
func (e webEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := webColors(l)
	b = winSnap(b)
	if b.Dx() < 8 || b.Dy() < 6 {
		return
	}
	px := c.px(l)
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	if c.accordion == 0 {
		fg := c.text2
		switch {
		case st.Disabled():
			fg = c.textDis
		case st.Pressed() || st.Hovered():
			fg = c.text
		}
		x := b.Min.X + l.S(6)
		webChevron(ctx, paintengine2d.XYWH(x, b.Min.Y, l.S(10), b.Dy()), dir, l.S(9), l.S(4.5), max(l.S(1.5), 1), fg)
		x += l.S(16)
		l.drawFittedText(ctx, c.bold, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x-l.S(4), b.Dy()), fg, AlignStart, 0)
		if st.Focused() && !st.Disabled() {
			c.focusRing(l, ctx, b, b, 0)
		}
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		a := float32(1)
		if st.Pressed() {
			a = 1.8
		}
		hb := b.Inset(snap(l.S(1)))
		r := min(c.rad(l, c.radius), hb.Dy()*0.5)
		ctx.DrawRoundRect(hb, r, r, paintengine2d.Fill(webA(c.wash, a)))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.border2))
	aw := l.S(16)
	webChevron(ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(8), b.Min.Y, aw, b.Dy()), dir, l.S(9), l.S(4.5), max(l.S(1.4), 1), c.text2)
	x := b.Min.X + l.S(10)
	l.drawFittedText(ctx, c.btnFace, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x-aw-l.S(12), b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !st.Disabled() {
		c.focusRing(l, ctx, b, b, min(c.rad(l, c.radius), b.Dy()*0.5))
	}
}

func (webEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := webColors(l)
	px := c.px(l)
	if vertical {
		h := min(b.Dy(), max(b.Dy()-2*snap(l.S(4)), 0))
		ctx.DrawRect(paintengine2d.XYWH(snap((b.Min.X+b.Max.X-px)*0.5), snap((b.Min.Y+b.Max.Y-h)*0.5), px, snap(h)), paintengine2d.Fill(c.border2))
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-px)*0.5), b.Dx(), px), paintengine2d.Fill(c.border2))
}

// DrawSplitter is a hairline in the structure colour across its track,
// darker under the pointer; no grip.
func (webEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := webColors(l)
	b = winSnap(b)
	if b.Empty() {
		return
	}
	px := c.px(l)
	col := c.border0
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		col = webOver(c.window, Mix(c.flat(c.border0), c.text, 0.35))
	}
	if vertical {
		ctx.DrawRect(paintengine2d.XYWH(snap((b.Min.X+b.Max.X-px)*0.5), b.Min.Y, px, b.Dy()), paintengine2d.Fill(col))
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-px)*0.5), b.Dx(), px), paintengine2d.Fill(col))
}

// TooltipStyle: A web tip is set in the small type the stylesheet gives it.
func (webEngine) TooltipStyle(l *Classic) TooltipStyle {
	return l.tipStyle(webColors(l).tipFace, l.tipPad(l.S(8)), AlignStart)
}

// DrawTooltip is a small popover (inverted where the design system inverts
// its tips: Primer, shadcn, Geist).
func (webEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := webColors(l)
	tb := b
	b = winSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	c.box(l, ctx, b, min(c.rad(l, c.tipR), b.Dy()*0.5), c.tip, c.tipBorder)
	pad := l.TooltipStyle().Pad
	l.drawTipText(ctx, paintengine2d.XYWH(tb.Min.X+pad, tb.Min.Y, tb.Dx()-pad-c.px(l), tb.Dy()), text, c.tipText)
}

// DrawMessageIcon: flat discs in the status colours with a white glyph.
func (webEngine) DrawMessageIcon(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	c := webColors(l)
	on := func(bg paintengine2d.Color) paintengine2d.Color {
		return ReadableOn(bg, 3, paintengine2d.RGB(1, 1, 1), paintengine2d.RGB(0, 0, 0))
	}
	switch icon {
	case IconNone:
	case IconError:
		winFlatDisc(l, ctx, b, c.danger, "×", on(c.danger))
	case IconWarning:
		winFlatDisc(l, ctx, b, c.warning, "!", on(c.warning))
	case IconInfo:
		winFlatDisc(l, ctx, b, c.accent, "i", on(c.accent))
	case IconQuestion:
		winFlatDisc(l, ctx, b, c.accent, "?", on(c.accent))
	default:
		l.baseDrawMessageIcon(ctx, b, icon)
	}
}

// ---- accent -------------------------------------------------------------------------------------

// webLight1 and webDark1 are Avalonia's accent shades (its
// SystemAccentColorLight1 and Dark1, which SourceGit's accent buttons use
// under the pointer and pressed): the accent's HSL lightness moved up 39 and
// down 28.5 of 255.
func webLight1(c paintengine2d.Color) paintengine2d.Color { return flatLighten(c, 39.0/255*100) }
func webDark1(c paintengine2d.Color) paintengine2d.Color  { return flatLighten(c, -28.5/255*100) }

// webSameRGB reports two colours with the same channels (any alpha).
func webSameRGB(a, b paintengine2d.Color) bool {
	d := func(x, y float32) bool { return x-y < 0.5/255 && y-x < 0.5/255 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// Accented takes the desktop's accent in the packs that follow it
// ("accentFollows": SourceGit, whose accent is the system's with #0078D7 as
// the fallback): the accent and Avalonia's shades of it, the text on it, and
// every colour the pack drew in its own accent at any alpha (selections,
// focus, ticks, the tab line) follow. The other packs keep their brand
// colours.
func (webEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	if accentP(tok, "accentFollows", 0) == 0 {
		return tok
	}
	own := accentX(tok, "accent", tok.Palette.Accent)
	accent = accent.WithAlpha(1)
	on := ReadableOn(accent, 3, accentX(tok, "onAccent", tok.Palette.TextOnAccent), paintengine2d.RGB(0, 0, 0))
	tok = CloneTokenMaps(tok)
	for k, v := range tok.Extra {
		if webSameRGB(v, own) {
			tok.Extra[k] = accent.WithAlpha(v.A)
		}
	}
	tok.Extra["accent"], tok.Extra["accentHover"], tok.Extra["accentPress"], tok.Extra["onAccent"] = accent, webLight1(accent), webDark1(accent), on
	p := &tok.Palette
	follow := func(c *paintengine2d.Color) {
		if webSameRGB(*c, own) {
			*c = accent.WithAlpha(c.A)
		}
	}
	for _, f := range []*paintengine2d.Color{&p.Selection, &p.Focus, &p.MenuHoverBorder, &p.MenuHover} {
		follow(f)
	}
	p.Accent, p.AccentHover, p.AccentPress, p.TextOnAccent = accent, webLight1(accent), webDark1(accent), on
	webChrome(&tok)
	return tok
}

// webChrome sets the chrome states the base painters read from the palette.
func webChrome(tok *ThemeTokens) {
	p := tok.Palette
	tok.Hot = ChromeState{Fill: p.MenuHover, Border: p.MenuHoverBorder}
	tok.Pressed = ChromeState{Fill: Mix(p.MenuHover, p.Text, 0.06), Border: p.Accent}
	tok.Selected = ChromeState{Fill: webOver(p.Field, p.Selection), Border: p.Accent}
	tok.Focus = ChromeState{Fill: p.Field, Border: p.Focus}
}
