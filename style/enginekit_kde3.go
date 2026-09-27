package style

// The KDE 3 kit: the frames, tool buttons, tree rows, status bars and
// tooltips that Keramik, Plastik and KDE 2's engine all draw with. KDE 3
// shipped them as one style vocabulary and the engines are variations on
// it, so they belong to all three rather than to whichever was written
// first.
//
// No build tag, so everything here is in every build.

import (
	"math"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// kde3CellText draws a table cell's label (padded, fitted, aligned).
func kde3CellText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, align Align, face *Font, fg paintengine2d.Color) {
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
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
		return BakeFamily(l.UIFamily(), WeightRegular, l.metrics.FontSize*k, l.palette.Text)
	}).(*Font)
}

func kde3Done(p *paintengine2d.Path) { kde3Paths.Put(p) }

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

// kde3Path is a cleared path from the pool; hand it back with kde3Done.
func kde3Path() *paintengine2d.Path {
	p := kde3Paths.Get().(*paintengine2d.Path)
	p.Reset()
	return p
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

// kde3Snap rounds b inwards to whole pixels, so hairlines stay crisp and
// nothing paints outside the rect it was given.
func kde3Snap(b paintengine2d.Rect) paintengine2d.Rect {
	x0 := float32(math.Ceil(float64(b.Min.X) - 0.01))
	y0 := float32(math.Ceil(float64(b.Min.Y) - 0.01))
	x1 := float32(math.Floor(float64(b.Max.X) + 0.01))
	y1 := float32(math.Floor(float64(b.Max.Y) + 0.01))
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(max(x1, x0), max(y1, y0))}
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

// kde3Tooltip is Qt 3's tooltip: a pale fill in a one-unit frame of the
// text colour.
func kde3Tooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string, fill, fg paintengine2d.Color) {
	u := kde3U(l)
	b = kde3Snap(b)
	ctx.DrawRect(b, paintengine2d.Fill(fg))
	if b.Dx() > 2*u && b.Dy() > 2*u {
		ctx.DrawRect(b.Inset(u), paintengine2d.Fill(fill))
	}
	pad := l.TooltipStyle().Pad
	l.drawTipText(ctx, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), text, fg)
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
			if d < depth && !st.HasNextSibling(d) {
				continue // that ancestor's branch has ended
			}
			gx := snap(b.Min.X + pad + float32(d)*indent + indent*0.5)
			y1 := b.Max.Y
			if d == depth && !st.HasNextSibling(depth) {
				y1 = cy + u // the last child's elbow
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

// kde3U is one design pixel at the look's scale, never under a device
// pixel.
func kde3U(l *Classic) float32 {
	return max(snap(l.S(1)), 1)
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

// kde3Scheme is a KDE 3 colour scheme: the colours KDE's Colors module
// wrote to kdeglobals for every application (the .kcsrc keys).
type kde3Scheme struct {
	background, foreground, button, buttonText  string
	selection, selectionText, base, text        string
	alternate, link                             string
	caption, captionBlend, captionText          string
	captionOff, captionOffBlend, captionOffText string
}

// kde3TitleShadow is the shadow under a title bar's text: a darker tone of
// the bar under light text, none under dark text.
func kde3TitleShadow(text, bar paintengine2d.Color) paintengine2d.Color {
	if Luma(text) < 0.5 {
		return paintengine2d.Color{}
	}
	return darkerPct(bar, 200).WithAlpha(0.75)
}

// kde3Paths recycles the paths the KDE 3 engines build for glyphs, grips
// and dotted lines. A device consumes a path during the draw call (the
// context reuses its own scratch path the same way, and recorders keep a
// snapshot), so a drawn path can go straight back to the pool.
var kde3Paths = sync.Pool{New: func() any { return paintengine2d.NewPath() }}

type kde3ToolFontKey struct{}

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

// kde3Disabled is KDE 3's disabled text: black text greys to Qt's darkGray,
// other colours move halfway to the background.
func kde3Disabled(fg, bg paintengine2d.Color) paintengine2d.Color {
	if fg.R+fg.G+fg.B < 0.05 {
		return Hex("#808080")
	}
	return Mix(fg, bg, 0.5)
}
