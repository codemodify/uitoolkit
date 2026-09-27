package style

// The shared Windows-chrome kit: the bevels, glyphs, wells and dotted
// grips that every Windows-descended engine draws with — 95, 2000, XP's
// Luna, Vista and 7's Aero, 8's Metro, 10 and Fluent — and that several
// unrelated engines borrow a part or two of.
//
// It lived in engine_aero.go, which meant a build that wanted Luna, or
// Metro, or just a dotted focus rectangle, had to compile Aero too. None
// of it is about Aero: winSnap rounds to the pixel grid, winDots draws a
// dotted rectangle, winWell sinks a field. They are here because more
// than one engine needs them, which is the only rule this file has.
//
// No build tag, so everything here is in every build.

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// chkIndex picks the check box / radio / glass state row: normal, hot,
// pressed, disabled.
func aeroChkIndex(st ControlState) int {
	switch {
	case st.Disabled():
		return 3
	case st.Pressed():
		return 2
	case st.Hovered():
		return 1
	}
	return 0
}

// winPx is one device line at the look's scale: 1 at 1x, 2 at 2x.
func winPx(l *Classic) float32 {
	v := float32(int(l.S(1) + 0.5))
	if v < 1 {
		v = 1
	}
	return v
}

// winSnap puts b on the pixel grid. Edges round half down, so a snapped
// rect never covers a pixel whose centre lies outside b, and rects that
// tile (table cells, rows) still meet without a gap.
func winSnap(b paintengine2d.Rect) paintengine2d.Rect {
	f := func(v float32) float32 { return float32(math.Ceil(float64(v) - 0.5 - 1e-3)) }
	x0, y0, x1, y1 := f(b.Min.X), f(b.Min.Y), f(b.Max.X), f(b.Max.Y)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(x1, y1)}
}

// winRing fills the lw-wide band just inside the round rect b (radius r)
// with paint, leaving the inside untouched: one even-odd path.
func winRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, paint paintengine2d.Paint) {
	if b.Empty() {
		return
	}
	if b.Dx() <= 2*lw || b.Dy() <= 2*lw {
		ctx.DrawRoundRect(b, r, r, paint)
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(b, r, r)
	ri := max(r-lw, 0)
	p.AddRoundRect(b.Inset(lw), ri, ri)
	paint.Style = paintengine2d.StyleFill
	paint.FillRule = paintengine2d.FillEvenOdd
	ctx.DrawPath(p, paint)
}

// winAddRoundRect appends a closed rect with per-corner radii (tl, tr, br,
// bl) to p, so several shapes can share one even-odd path.
func winAddRoundRect(p *paintengine2d.Path, b paintengine2d.Rect, tl, tr, br, bl float32) {
	if b.Empty() {
		return
	}
	lim := min(b.Dx(), b.Dy()) * 0.5
	tl, tr, br, bl = min(max(tl, 0), lim), min(max(tr, 0), lim), min(max(br, 0), lim), min(max(bl, 0), lim)
	const k = 0.5522847 // cubic circle constant
	p.MoveTo(b.Min.X+tl, b.Min.Y)
	p.LineTo(b.Max.X-tr, b.Min.Y)
	if tr > 0 {
		p.CubicTo(b.Max.X-tr+tr*k, b.Min.Y, b.Max.X, b.Min.Y+tr-tr*k, b.Max.X, b.Min.Y+tr)
	}
	p.LineTo(b.Max.X, b.Max.Y-br)
	if br > 0 {
		p.CubicTo(b.Max.X, b.Max.Y-br+br*k, b.Max.X-br+br*k, b.Max.Y, b.Max.X-br, b.Max.Y)
	}
	p.LineTo(b.Min.X+bl, b.Max.Y)
	if bl > 0 {
		p.CubicTo(b.Min.X+bl-bl*k, b.Max.Y, b.Min.X, b.Max.Y-bl+bl*k, b.Min.X, b.Max.Y-bl)
	}
	p.LineTo(b.Min.X, b.Min.Y+tl)
	if tl > 0 {
		p.CubicTo(b.Min.X, b.Min.Y+tl-tl*k, b.Min.X+tl-tl*k, b.Min.Y, b.Min.X+tl, b.Min.Y)
	}
	p.Close()
}

// winBorder fills a square lw-wide border just inside b (one path).
func winBorder(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, col paintengine2d.Color) {
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-lw, b.Dx(), lw))
	p.AddRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+lw, lw, b.Dy()-2*lw))
	p.AddRect(paintengine2d.XYWH(b.Max.X-lw, b.Min.Y+lw, lw, b.Dy()-2*lw))
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winWell fills b with fill inside an lw-wide square border.
func winWell(ctx *paintengine2d.Context, b paintengine2d.Rect, lw float32, border, fill paintengine2d.Color) {
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(fill))
	winBorder(ctx, b, lw, border)
}

// winDots is the Windows dotted focus rectangle just inside b: dots of one
// device line on every other one, all in one path.
func winDots(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, lw float32) {
	b = winSnap(b)
	if b.Dx() < 2*lw || b.Dy() < 2*lw || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	step := 2 * lw
	for x := b.Min.X; x+lw <= b.Max.X+0.01; x += step {
		p.AddRect(paintengine2d.XYWH(x, b.Min.Y, lw, lw))
		p.AddRect(paintengine2d.XYWH(x, b.Max.Y-lw, lw, lw))
	}
	for y := b.Min.Y + step; y+lw <= b.Max.Y-lw+0.01; y += step {
		p.AddRect(paintengine2d.XYWH(b.Min.X, y, lw, lw))
		p.AddRect(paintengine2d.XYWH(b.Max.X-lw, y, lw, lw))
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winGlyph fills the solid Windows arrow glyph — a stepped triangle 8
// units across and 4 deep, one device line per step — pointing dir,
// centred in b, u pixels per unit.
func winGlyph(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, u float32, col paintengine2d.Color) {
	if col.A <= 0 || b.Empty() || u <= 0 {
		return
	}
	n := int(4*u + 0.5) // steps, one device line each
	if n < 2 {
		n = 2
	}
	w := float32(2 * n) // across
	h := float32(n)     // deep
	vertical := dir == DirUp || dir == DirDown
	var x0, y0 float32
	if vertical {
		x0, y0 = snap((b.Min.X+b.Max.X-w)*0.5), snap((b.Min.Y+b.Max.Y-h)*0.5)
	} else {
		x0, y0 = snap((b.Min.X+b.Max.X-h)*0.5), snap((b.Min.Y+b.Max.Y-w)*0.5)
	}
	p := paintengine2d.NewPath()
	for k := 0; k < n; k++ {
		f := float32(k)
		switch dir {
		case DirDown:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+f, w-2*f, 1))
		case DirUp:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+h-1-f, w-2*f, 1))
		case DirRight:
			p.AddRect(paintengine2d.XYWH(x0+f, y0+f, 1, w-2*f))
		default:
			p.AddRect(paintengine2d.XYWH(x0+h-1-f, y0+f, 1, w-2*f))
		}
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winTick strokes a check mark filling the box b (round caps: square caps
// on an open polyline draw a stray band in the engine).
func winTick(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, width float32) {
	if b.Empty() || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+b.Dx()*0.24, b.Min.Y+b.Dy()*0.52)
	p.LineTo(b.Min.X+b.Dx()*0.42, b.Min.Y+b.Dy()*0.70)
	p.LineTo(b.Min.X+b.Dx()*0.77, b.Min.Y+b.Dy()*0.30)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winCross fills an × from corner to corner of b, two bars w thick with
// flat ends.
func winCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, w float32) {
	if b.Empty() || w <= 0 || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	for _, d := range [2][4]float32{{b.Min.X, b.Min.Y, b.Max.X, b.Max.Y}, {b.Max.X, b.Min.Y, b.Min.X, b.Max.Y}} {
		x0, y0, x1, y1 := d[0], d[1], d[2], d[3]
		dx, dy := x1-x0, y1-y0
		n := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		nx, ny := -dy/n*w*0.5, dx/n*w*0.5
		p.MoveTo(x0+nx, y0+ny)
		p.LineTo(x1+nx, y1+ny)
		p.LineTo(x1-nx, y1-ny)
		p.LineTo(x0-nx, y0-ny)
		p.Close()
	}
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// winVistaTriangle is the Vista / 7 tree glyph centred in b: a hollow
// triangle pointing right (closed), a filled one pointing down-right (open).
func winVistaTriangle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, open, openEdge, closed, closedFill paintengine2d.Color) {
	u := l.S(1)
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	if expanded {
		s := snap(6 * u)
		x0, y0 := cx-snap(s*0.5), cy-snap(s*0.5)
		p := paintengine2d.NewPath()
		p.MoveTo(x0+s, y0)
		p.LineTo(x0+s, y0+s)
		p.LineTo(x0, y0+s)
		p.Close()
		ctx.DrawPath(p, paintengine2d.Fill(open))
		ctx.DrawPath(p, paintengine2d.Paint{Color: openEdge.WithAlpha(0.6), Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: u * 0.8, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
		return
	}
	h := snap(8 * u)
	w := h * 0.5
	x0, y0 := cx-snap(w*0.5), cy-h*0.5
	p := paintengine2d.NewPath()
	p.MoveTo(x0+u*0.5, y0+u*0.5)
	p.LineTo(x0+w, y0+h*0.5)
	p.LineTo(x0+u*0.5, y0+h-u*0.5)
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(closedFill))
	ctx.DrawPath(p, paintengine2d.Paint{Color: closed, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: u, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winPointer is a trackbar thumb pointing down: a block with rounded top
// corners (r) ending in a point p deep.
func winPointer(b paintengine2d.Rect, pt, r float32) *paintengine2d.Path {
	r = min(r, b.Dx()*0.3)
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+r, b.Min.Y)
	p.LineTo(b.Max.X-r, b.Min.Y)
	if r > 0 {
		p.QuadTo(b.Max.X, b.Min.Y, b.Max.X, b.Min.Y+r)
	}
	p.LineTo(b.Max.X, b.Max.Y-pt)
	p.LineTo((b.Min.X+b.Max.X)*0.5, b.Max.Y)
	p.LineTo(b.Min.X, b.Max.Y-pt)
	p.LineTo(b.Min.X, b.Min.Y+r)
	if r > 0 {
		p.QuadTo(b.Min.X, b.Min.Y, b.Min.X+r, b.Min.Y)
	}
	p.Close()
	return p
}

// winPagedPart is the part of the track a held page press darkens.
func winPagedPart(p ScrollParts, vertical bool, st ScrollState) paintengine2d.Rect {
	if (st.Pressed != ScrollPageDec && st.Pressed != ScrollPageInc) || p.Thumb.Empty() {
		return paintengine2d.Rect{}
	}
	pg := p.Track
	if vertical {
		if st.Pressed == ScrollPageDec {
			pg.Max.Y = p.Thumb.Min.Y
		} else {
			pg.Min.Y = p.Thumb.Max.Y
		}
		pg.Min.X, pg.Max.X = p.Bar.Min.X, p.Bar.Max.X
	} else {
		if st.Pressed == ScrollPageDec {
			pg.Max.X = p.Thumb.Min.X
		} else {
			pg.Min.X = p.Thumb.Max.X
		}
		pg.Min.Y, pg.Max.Y = p.Bar.Min.Y, p.Bar.Max.Y
	}
	return pg
}

// winScrollState turns a thumb's ControlState into a ScrollState for the
// DrawScrollBar fallback.
func winScrollState(st ControlState) ScrollState {
	ss := ScrollState{Disabled: st.Disabled(), Hovered: st.Hovered()}
	switch {
	case st.Pressed():
		ss.Pressed, ss.Hot = ScrollThumbPart, ScrollThumbPart
	case st.Hovered():
		ss.Hot = ScrollThumbPart
	}
	return ss
}

// winShadow is a DropShadow recipe.
type winShadow struct {
	col                     paintengine2d.Color
	r, dx, dy, blur, spread float32
}

// winShadowSpec scales a shadow's alpha by the pack's "shadow" param (0
// turns shadows off).
func winShadowSpec(l *Classic, alpha, r, dx, dy, blur, spread float32) (winShadow, bool) {
	k := l.P("shadow", 1)
	if k <= 0 {
		return winShadow{}, false
	}
	return winShadow{col: paintengine2d.RGBA(0, 0, 0, min(alpha*k, 1)), r: r, dx: dx, dy: dy, blur: blur, spread: spread}, true
}

// winLabelBox is the focus rectangle around a label drawn in lb.
func winLabelBox(l *Classic, f *Font, label string, lb paintengine2d.Rect, align Align) paintengine2d.Rect {
	w := min(f.Advance(label)+l.S(6), lb.Dx()-l.S(2))
	h := min(f.Height()+l.S(2), lb.Dy())
	x := lb.Min.X - l.S(3)
	if align == AlignCenter {
		x = lb.Min.X + (lb.Dx()-w)*0.5
	}
	return paintengine2d.XYWH(x, lb.Min.Y+(lb.Dy()-h)*0.5, w, h).Intersect(lb)
}

// winDisabledText draws a disabled field's text (or placeholder) in col.
func winDisabledText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text, placeholder string, scrollX float32, face *Font, col paintengine2d.Color) {
	pad := l.metrics.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	if inner.Empty() {
		return
	}
	show := text
	if show == "" {
		show = placeholder
	}
	f := l.faceOrBody(face)
	ctx.Save()
	ctx.ClipRect(inner)
	f.Draw(ctx, show, paintengine2d.Pt(inner.Min.X-scrollX, inner.Min.Y+(inner.Dy()-f.Height())*0.5), col)
	ctx.Restore()
}

// winMenuText lays out a menu row's label, shortcut and submenu arrow.
func winMenuText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, ch MenuChrome, row MenuRow, fg, shortcut paintengine2d.Color, arrow func(ab paintengine2d.Rect)) {
	f := l.body
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		arrow(ab)
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
		f.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), shortcut)
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

// winCellText draws a table cell's label with the stock padding rules.
func winCellText(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, align Align, face *Font, fg paintengine2d.Color) {
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
	ctx.ClipRect(b.Inset(1))
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

// winToolContent draws a tool button's icon and label in fg.
func winToolContent(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, label string, icon ToolIcon, fg paintengine2d.Color) {
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
		right := max(b.Max.X-pad, x)
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		l.body.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-l.body.Height())*0.5), fg)
		ctx.Restore()
	}
}

// winStatusParts lays status bar parts out in equal slots (leaving room
// for the size grip), divided by an etched pair of lines.
func winStatusParts(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string, fg, sepDk, sepLt paintengine2d.Color) {
	if len(parts) == 0 {
		return
	}
	lw := winPx(l)
	slot := (b.Dx() - l.S(16)) / float32(len(parts))
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			sx := snap(x)
			y0 := b.Min.Y + snap(l.S(4))
			h := b.Max.Y - y0 - snap(l.S(4))
			dk.AddRect(paintengine2d.XYWH(sx-lw, y0, lw, h))
			lt.AddRect(paintengine2d.XYWH(sx, y0, lw, h))
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(6), b.Min.Y+lw, slot-l.S(10), b.Dy()-lw), fg, AlignStart, 0)
	}
	if sepLt.A > 0 {
		ctx.DrawPath(lt, paintengine2d.Fill(sepLt))
	}
	ctx.DrawPath(dk, paintengine2d.Fill(sepDk))
}

// winGripDots paints the size grip: a triangle of dots, each with a light
// shadow, at the bottom right of b.
func winGripDots(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dot, lite paintengine2d.Color) {
	lw := winPx(l)
	d := 2 * lw
	step := snap(l.S(4))
	if b.Dy() < 3*step+d || b.Dx() < 3*step+d {
		return
	}
	dk, lt := paintengine2d.NewPath(), paintengine2d.NewPath()
	for row := 0; row < 3; row++ {
		for k := 0; k <= row; k++ {
			x := b.Max.X - float32(k+1)*step
			y := b.Max.Y - float32(3-row)*step
			lt.AddRect(paintengine2d.XYWH(x+lw, y+lw, d, d))
			dk.AddRect(paintengine2d.XYWH(x, y, d, d))
		}
	}
	if lite.A > 0 {
		ctx.DrawPath(lt, paintengine2d.Fill(lite))
	}
	ctx.DrawPath(dk, paintengine2d.Fill(dot))
}

// winHeading draws a title (in the title font when it fits and big) and a
// muted subtitle after it.
func winHeading(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string, col, muted paintengine2d.Color, bold bool) {
	x := b.Min.X + l.metrics.Pad
	f := l.body
	if bold && l.bold != nil {
		f = l.bold
	} else if subtitle == "" && l.title != nil && b.Dy() >= l.title.Height()+l.S(6) {
		f = l.title
	}
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), col, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), muted, AlignStart, 0)
	}
}

// winEtched is a two-line groove (dark over light) across the middle of b.
func winEtched(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, dk, lt paintengine2d.Color) {
	lw := winPx(l)
	if vertical {
		x := snap((b.Min.X+b.Max.X)*0.5) - lw
		y0, h := snap(b.Min.Y+l.S(2)), snap(b.Dy()-l.S(4))
		ctx.DrawRect(paintengine2d.XYWH(x, y0, lw, h), paintengine2d.Fill(dk))
		if lt.A > 0 {
			ctx.DrawRect(paintengine2d.XYWH(x+lw, y0, lw, h), paintengine2d.Fill(lt))
		}
		return
	}
	y := snap((b.Min.Y+b.Max.Y)*0.5) - lw
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), lw), paintengine2d.Fill(dk))
	if lt.A > 0 {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y+lw, b.Dx(), lw), paintengine2d.Fill(lt))
	}
}

// winGroupFrame is a group box frame of radius r in a line (with an inner
// light line when inner is set), left open behind the title; raised fills
// it with card.
func winGroupFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool, r float32, line, inner, card, text paintengine2d.Color) {
	b = winSnap(b)
	lw := winPx(l)
	f := l.body
	top := b.Min.Y
	if title != "" {
		top = snap(b.Min.Y + f.Height()*0.5)
	}
	frame := paintengine2d.XYWH(b.Min.X, top, b.Dx(), b.Max.Y-top)
	if frame.Dx() < 4*lw || frame.Dy() < 4*lw {
		return
	}
	if raised {
		ctx.DrawRoundRect(frame, r, r, paintengine2d.Fill(card))
	}
	var gap paintengine2d.Rect
	if title != "" {
		tw := min(f.Advance(title)+l.S(6), b.Dx()-l.S(16))
		gap = paintengine2d.XYWH(b.Min.X+l.S(8), b.Min.Y, max(tw, 0), f.Height())
	}
	if !gap.Empty() {
		clip := paintengine2d.NewPath()
		clip.AddRect(b)
		clip.AddRect(gap)
		ctx.Save()
		ctx.ClipPathRule(clip, paintengine2d.FillEvenOdd)
	}
	if inner.A > 0 {
		winRing(ctx, frame.Inset(lw), max(r-lw, 0), lw, paintengine2d.Fill(inner))
	}
	winRing(ctx, frame, r, lw, paintengine2d.Fill(line))
	if !gap.Empty() {
		ctx.Restore()
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(gap.Min.X+l.S(3), gap.Min.Y, gap.Dx()-l.S(3), gap.Dy()), text, AlignStart, 0)
	}
}

// winChevron strokes a thin chevron (Segoe MDL2's) centred in b: 8 units
// across and 4 deep, one line thick.
func winChevron(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	u := min(l.S(1), min(b.Dx(), b.Dy())/10)
	lw := max(winPx(l)*0.9, u)
	cx, cy := snap((b.Min.X+b.Max.X)*0.5), snap((b.Min.Y+b.Max.Y)*0.5)
	a, d := 4*u, 2*u
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-a, cy+d)
		p.LineTo(cx, cy-d)
		p.LineTo(cx+a, cy+d)
	case DirDown:
		p.MoveTo(cx-a, cy-d)
		p.LineTo(cx, cy+d)
		p.LineTo(cx+a, cy-d)
	case DirLeft:
		p.MoveTo(cx+d, cy-a)
		p.LineTo(cx-d, cy)
		p.LineTo(cx+d, cy+a)
	default:
		p.MoveTo(cx-d, cy-a)
		p.LineTo(cx+d, cy)
		p.LineTo(cx-d, cy+a)
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: lw, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// winFlatDisc paints a flat message-box disc with a glyph.
func winFlatDisc(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, fill paintengine2d.Color, glyph string, gc paintengine2d.Color) {
	s := min(b.Dx(), b.Dy())
	if s < 4 {
		return
	}
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	r := s * 0.46
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(fill))
	if glyph == "×" {
		k := r * 0.36
		winCross(ctx, paintengine2d.XYWH(cx-k, cy-k, 2*k, 2*k), gc, r*0.16)
		return
	}
	f := l.bold
	if f == nil {
		f = l.body
	}
	w := f.Advance(glyph)
	f.Draw(ctx, glyph, paintengine2d.Pt(cx-w*0.5, cy-f.Height()*0.5), gc)
}
