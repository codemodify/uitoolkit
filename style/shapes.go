package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// Shape helpers shared by the theme engines. They are deliberately small
// and literal — each one is a drawing idiom that several eras used — so an
// engine reads like a description of its era.
//
// Conventions: b is the box in device-independent pixels of the current
// context (already scaled by the look when the caller used l.S), colours
// are straight alpha, strokes are centred on pixel centres where it
// matters (Inset(0.5)).

// Stop is a gradient stop shorthand.
func Stop(at float32, c paintengine2d.Color) paintengine2d.GradientStop {
	return paintengine2d.GradientStop{Offset: at, Color: c}
}

// VGradient is a top→bottom linear gradient paint across b.
func VGradient(b paintengine2d.Rect, stops ...paintengine2d.GradientStop) paintengine2d.Paint {
	return paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(b.Min.X, b.Min.Y),
		End:   paintengine2d.Pt(b.Min.X, b.Max.Y),
		Stops: stops,
	})
}

// HGradient is a left→right linear gradient paint across b.
func HGradient(b paintengine2d.Rect, stops ...paintengine2d.GradientStop) paintengine2d.Paint {
	return paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(b.Min.X, b.Min.Y),
		End:   paintengine2d.Pt(b.Max.X, b.Min.Y),
		Stops: stops,
	})
}

// DGradient is a top-left→bottom-right gradient (Window Maker dgradient).
func DGradient(b paintengine2d.Rect, stops ...paintengine2d.GradientStop) paintengine2d.Paint {
	return paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: b.Min,
		End:   b.Max,
		Stops: stops,
	})
}

// FillV fills a (round) rect with a vertical gradient.
func FillV(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, stops ...paintengine2d.GradientStop) {
	if ctx == nil || b.Empty() {
		return
	}
	ctx.DrawRoundRect(b, r, r, VGradient(b, stops...))
}

// FillH fills a (round) rect with a horizontal gradient.
func FillH(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, stops ...paintengine2d.GradientStop) {
	if ctx == nil || b.Empty() {
		return
	}
	ctx.DrawRoundRect(b, r, r, HGradient(b, stops...))
}

// Bevel4 paints the Windows 95 four-colour bevel: an outer ring
// (outerHi top/left, outerLo bottom/right) and an inner ring (innerHi,
// innerLo), 1px each. Pass the colours swapped for a sunken edge.
func Bevel4(ctx *paintengine2d.Context, b paintengine2d.Rect, outerHi, outerLo, innerHi, innerLo paintengine2d.Color) {
	if ctx == nil || b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	edge(ctx, b, outerHi, outerLo)
	if b.Dx() >= 4 && b.Dy() >= 4 {
		edge(ctx, b.Inset(1), innerHi, innerLo)
	}
}

// edge paints one 1px ring: hi on top/left, lo on bottom/right (lo owns the
// corner pixels, as Windows does).
func edge(ctx *paintengine2d.Context, b paintengine2d.Rect, hi, lo paintengine2d.Color) {
	w, h := b.Dx(), b.Dy()
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, w-1, 1), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, 1, h-1), paintengine2d.Fill(hi))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, w, 1), paintengine2d.Fill(lo))
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-1, b.Min.Y, 1, h), paintengine2d.Fill(lo))
}

// Edge paints a single 1px bevel ring (hi top/left, lo bottom/right).
func Edge(ctx *paintengine2d.Context, b paintengine2d.Rect, hi, lo paintengine2d.Color) {
	if ctx == nil || b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	edge(ctx, b, hi, lo)
}

// Shadow2 paints a Motif-style shadow of n pixels (hi top/left, lo
// bottom/right) with the diagonal corner split Motif uses.
func Shadow2(ctx *paintengine2d.Context, b paintengine2d.Rect, hi, lo paintengine2d.Color, n int) {
	if ctx == nil || n <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		r := b.Inset(float32(i))
		if r.Dx() < 2 || r.Dy() < 2 {
			return
		}
		edge(ctx, r, hi, lo)
	}
}

// Etched paints an etched (grooved) frame: dark line with a light line
// offset by one pixel — Win95 group boxes, Metal borders, separators.
func Etched(ctx *paintengine2d.Context, b paintengine2d.Rect, dark, light paintengine2d.Color) {
	if ctx == nil || b.Dx() < 3 || b.Dy() < 3 {
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+0.5, b.Min.Y+0.5, b.Dx()-2, b.Dy()-2), paintengine2d.StrokePaint(dark, 1))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1.5, b.Min.Y+1.5, b.Dx()-2, b.Dy()-2), paintengine2d.StrokePaint(light, 1))
}

// EtchedLine is a horizontal or vertical groove (dark + light).
func EtchedLine(ctx *paintengine2d.Context, x, y, length float32, vertical bool, dark, light paintengine2d.Color) {
	if ctx == nil || length <= 0 {
		return
	}
	if vertical {
		ctx.DrawRect(paintengine2d.XYWH(x, y, 1, length), paintengine2d.Fill(dark))
		ctx.DrawRect(paintengine2d.XYWH(x+1, y, 1, length), paintengine2d.Fill(light))
		return
	}
	ctx.DrawRect(paintengine2d.XYWH(x, y, length, 1), paintengine2d.Fill(dark))
	ctx.DrawRect(paintengine2d.XYWH(x, y+1, length, 1), paintengine2d.Fill(light))
}

// Pill fills b as a stadium (radius = half the short side).
func Pill(ctx *paintengine2d.Context, b paintengine2d.Rect, paint paintengine2d.Paint) {
	if ctx == nil || b.Empty() {
		return
	}
	r := b.Dy() * 0.5
	if b.Dx() < b.Dy() {
		r = b.Dx() * 0.5
	}
	ctx.DrawRoundRect(b, r, r, paint)
}

// RoundRectPath builds a rect path with per-corner radii (tl, tr, br, bl).
func RoundRectPath(b paintengine2d.Rect, tl, tr, br, bl float32) *paintengine2d.Path {
	clampR := func(r float32) float32 {
		m := b.Dx() * 0.5
		if b.Dy()*0.5 < m {
			m = b.Dy() * 0.5
		}
		if r > m {
			r = m
		}
		if r < 0 {
			r = 0
		}
		return r
	}
	tl, tr, br, bl = clampR(tl), clampR(tr), clampR(br), clampR(bl)
	const k = 0.5522847 // cubic circle constant
	p := paintengine2d.NewPath()
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
	return p
}

// Gloss paints the classic top highlight of a gel/glass face: a white band
// over the upper part of b fading from topAlpha to midAlpha at split
// (0..1 of the height). Aqua, Luna, Aero, Nimbus, Keramik all use a variant.
func Gloss(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, split, topAlpha, midAlpha float32) {
	if ctx == nil || b.Empty() {
		return
	}
	if split <= 0 || split > 1 {
		split = 0.5
	}
	hb := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()*split)
	white := paintengine2d.RGB(1, 1, 1)
	path := RoundRectPath(hb, r, r, 0, 0)
	ctx.DrawPath(path, VGradient(hb, Stop(0, white.WithAlpha(topAlpha)), Stop(1, white.WithAlpha(midAlpha))))
}

// DottedRect paints the Windows / Motif dotted focus rectangle: 1px dots on
// alternate pixels, inside b.
func DottedRect(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	if ctx == nil || b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	x0, y0 := float32(math.Floor(float64(b.Min.X))), float32(math.Floor(float64(b.Min.Y)))
	x1, y1 := float32(math.Floor(float64(b.Max.X)))-1, float32(math.Floor(float64(b.Max.Y)))-1
	fill := paintengine2d.Fill(col)
	for x := x0; x <= x1; x += 2 {
		ctx.DrawRect(paintengine2d.XYWH(x, y0, 1, 1), fill)
		ctx.DrawRect(paintengine2d.XYWH(x, y1, 1, 1), fill)
	}
	for y := y0 + 2; y < y1; y += 2 {
		ctx.DrawRect(paintengine2d.XYWH(x0, y, 1, 1), fill)
		ctx.DrawRect(paintengine2d.XYWH(x1, y, 1, 1), fill)
	}
}

// Bumps paints the Java Metal "bumps" texture: a lattice of light dots with
// dark dots one pixel down-right, every spacing pixels, clipped to b.
func Bumps(ctx *paintengine2d.Context, b paintengine2d.Rect, light, dark paintengine2d.Color, spacing float32) {
	if ctx == nil || b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	if spacing < 2 {
		spacing = 4
	}
	ctx.Save()
	ctx.ClipRect(b)
	lf, df := paintengine2d.Fill(light), paintengine2d.Fill(dark)
	row := 0
	for y := b.Min.Y; y < b.Max.Y; y += spacing / 2 {
		off := float32(0)
		if row%2 == 1 {
			off = spacing / 2
		}
		for x := b.Min.X + off; x < b.Max.X; x += spacing {
			ctx.DrawRect(paintengine2d.XYWH(x, y, 1, 1), lf)
			ctx.DrawRect(paintengine2d.XYWH(x+1, y+1, 1, 1), df)
		}
		row++
	}
	ctx.Restore()
}

// Pinstripes paints horizontal stripes (Aqua window backgrounds, Mac OS
// title bars): a line of col every period pixels.
func Pinstripes(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, period, thickness float32) {
	if ctx == nil || b.Empty() {
		return
	}
	if period < 2 {
		period = 4
	}
	if thickness <= 0 {
		thickness = 1
	}
	ctx.Save()
	ctx.ClipRect(b)
	for y := b.Min.Y; y < b.Max.Y; y += period {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), thickness), paintengine2d.Fill(col))
	}
	ctx.Restore()
}

// GripLines paints n raised grip lines across b (toolbar handles, thumb
// grips, splitters). vertical = lines run vertically (side by side).
func GripLines(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, n int, gap float32, hi, lo paintengine2d.Color) {
	if ctx == nil || n <= 0 {
		return
	}
	if gap < 2 {
		gap = 3
	}
	span := float32(n-1) * gap
	if vertical {
		x := (b.Min.X+b.Max.X)*0.5 - span*0.5
		for i := 0; i < n; i++ {
			xx := float32(math.Floor(float64(x + float32(i)*gap)))
			ctx.DrawRect(paintengine2d.XYWH(xx, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(hi))
			ctx.DrawRect(paintengine2d.XYWH(xx+1, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(lo))
		}
		return
	}
	y := (b.Min.Y+b.Max.Y)*0.5 - span*0.5
	for i := 0; i < n; i++ {
		yy := float32(math.Floor(float64(y + float32(i)*gap)))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, yy, b.Dx(), 1), paintengine2d.Fill(hi))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, yy+1, b.Dx(), 1), paintengine2d.Fill(lo))
	}
}

// GripDots paints a grid of raised dots (Plastik, Fusion, Office 2003).
func GripDots(ctx *paintengine2d.Context, b paintengine2d.Rect, cols, rows int, gap float32, hi, lo paintengine2d.Color) {
	if ctx == nil || cols <= 0 || rows <= 0 {
		return
	}
	if gap < 2 {
		gap = 3
	}
	w := float32(cols-1) * gap
	h := float32(rows-1) * gap
	x0 := float32(math.Floor(float64((b.Min.X+b.Max.X)*0.5 - w*0.5)))
	y0 := float32(math.Floor(float64((b.Min.Y+b.Max.Y)*0.5 - h*0.5)))
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x, y := x0+float32(c)*gap, y0+float32(r)*gap
			ctx.DrawRect(paintengine2d.XYWH(x+1, y+1, 1, 1), paintengine2d.Fill(lo))
			ctx.DrawRect(paintengine2d.XYWH(x, y, 1, 1), paintengine2d.Fill(hi))
		}
	}
}

// ArrowPath is a filled triangle pointing dir, fitted into b (keeps the
// classic 2:1 base:height proportions).
func ArrowPath(b paintengine2d.Rect, dir Direction) *paintengine2d.Path {
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	half := s * 0.5
	h := half // height along the pointing axis
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-half, cy+h*0.5)
		p.LineTo(cx+half, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
	case DirDown:
		p.MoveTo(cx-half, cy-h*0.5)
		p.LineTo(cx+half, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
	case DirLeft:
		p.MoveTo(cx+h*0.5, cy-half)
		p.LineTo(cx+h*0.5, cy+half)
		p.LineTo(cx-h*0.5, cy)
	default:
		p.MoveTo(cx-h*0.5, cy-half)
		p.LineTo(cx-h*0.5, cy+half)
		p.LineTo(cx+h*0.5, cy)
	}
	p.Close()
	return p
}

// FillArrow fills a triangle arrow into b.
func FillArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	ctx.DrawPath(ArrowPath(b, dir), paintengine2d.Fill(col))
}

// Chevron strokes an open chevron (›, ⌄) into b — the modern arrow.
func Chevron(ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color, width float32) {
	if ctx == nil || b.Empty() {
		return
	}
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	s := b.Dx()
	if b.Dy() < s {
		s = b.Dy()
	}
	a := s * 0.42
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-a, cy+a*0.5)
		p.LineTo(cx, cy-a*0.5)
		p.LineTo(cx+a, cy+a*0.5)
	case DirDown:
		p.MoveTo(cx-a, cy-a*0.5)
		p.LineTo(cx, cy+a*0.5)
		p.LineTo(cx+a, cy-a*0.5)
	case DirLeft:
		p.MoveTo(cx+a*0.5, cy-a)
		p.LineTo(cx-a*0.5, cy)
		p.LineTo(cx+a*0.5, cy+a)
	default:
		p.MoveTo(cx-a*0.5, cy-a)
		p.LineTo(cx+a*0.5, cy)
		p.LineTo(cx-a*0.5, cy+a)
	}
	if width <= 0 {
		width = 1.5
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// Tick strokes a check mark into b.
func Tick(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, width float32) {
	if ctx == nil || b.Empty() {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+b.Dx()*0.14, b.Min.Y+b.Dy()*0.52)
	p.LineTo(b.Min.X+b.Dx()*0.40, b.Min.Y+b.Dy()*0.80)
	p.LineTo(b.Min.X+b.Dx()*0.88, b.Min.Y+b.Dy()*0.18)
	if width <= 0 {
		width = 2
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// PixelTick fills the chunky 7x7 Windows 95 check mark scaled to b.
func PixelTick(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	u := b.Dx() / 7
	if v := b.Dy() / 7; v < u {
		u = v
	}
	ox := b.Min.X + (b.Dx()-7*u)*0.5
	oy := b.Min.Y + (b.Dy()-7*u)*0.5
	// Win95 tick: columns 0..6, top rows 2,3,4,3,2,1,0, each 3px tall.
	tops := [7]int{2, 3, 4, 3, 2, 1, 0}
	for c := 0; c < 7; c++ {
		ctx.DrawRect(paintengine2d.XYWH(ox+float32(c)*u, oy+float32(tops[c])*u, u, 3*u), paintengine2d.Fill(col))
	}
}

// DrawCross strokes an × into b (close buttons, Motif/OS/2 checks).
func DrawCross(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color, width float32) {
	if ctx == nil || b.Empty() {
		return
	}
	if width <= 0 {
		width = 1.5
	}
	st := paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}}
	ctx.DrawLine(b.Min, b.Max, st)
	ctx.DrawLine(paintengine2d.Pt(b.Max.X, b.Min.Y), paintengine2d.Pt(b.Min.X, b.Max.Y), st)
}

// DiamondPath is a diamond (rotated square) inscribed in b.
func DiamondPath(b paintengine2d.Rect) *paintengine2d.Path {
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	p := paintengine2d.NewPath()
	p.MoveTo(cx, b.Min.Y)
	p.LineTo(b.Max.X, cy)
	p.LineTo(cx, b.Max.Y)
	p.LineTo(b.Min.X, cy)
	p.Close()
	return p
}

// Diamond3D paints a Motif radio diamond: fill, then hi on the upper two
// edges and lo on the lower two (swap for "set"/sunken), width px.
func Diamond3D(ctx *paintengine2d.Context, b paintengine2d.Rect, fill, hi, lo paintengine2d.Color, width float32) {
	if ctx == nil || b.Empty() {
		return
	}
	ctx.DrawPath(DiamondPath(b), paintengine2d.Fill(fill))
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	if width <= 0 {
		width = 2
	}
	in := width * 0.5
	st := func(c paintengine2d.Color) paintengine2d.Paint {
		return paintengine2d.Paint{Color: c, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: width, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 4}}
	}
	up := paintengine2d.NewPath()
	up.MoveTo(b.Min.X+in, cy)
	up.LineTo(cx, b.Min.Y+in)
	up.LineTo(b.Max.X-in, cy)
	ctx.DrawPath(up, st(hi))
	dn := paintengine2d.NewPath()
	dn.MoveTo(b.Min.X+in, cy)
	dn.LineTo(cx, b.Max.Y-in)
	dn.LineTo(b.Max.X-in, cy)
	ctx.DrawPath(dn, st(lo))
}

// SoftShadow fakes a blurred drop shadow with stacked translucent round
// rects (the engine has no blur). spread is the blur radius in px.
func SoftShadow(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color, dx, dy, spread float32) {
	if ctx == nil || b.Empty() || col.A <= 0 {
		return
	}
	steps := int(spread)
	if steps < 1 {
		steps = 1
	}
	if steps > 8 {
		steps = 8
	}
	a := col.A / float32(steps)
	for i := steps; i >= 1; i-- {
		g := float32(i) * spread / float32(steps)
		sb := paintengine2d.XYWH(b.Min.X+dx-g*0.5, b.Min.Y+dy-g*0.5, b.Dx()+g, b.Dy()+g)
		ctx.DrawRoundRect(sb, r+g*0.5, r+g*0.5, paintengine2d.Fill(col.WithAlpha(a)))
	}
}

// Glow strokes concentric fading rings outside b (Oxygen / Aero focus and
// hover glows). width is how far the glow extends.
func Glow(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color, width float32) {
	if ctx == nil || b.Empty() || width <= 0 {
		return
	}
	n := int(width + 0.5)
	if n < 1 {
		n = 1
	}
	for i := 1; i <= n; i++ {
		f := float32(i)
		a := col.A * (1 - (f-1)/float32(n)) * 0.6
		ctx.DrawRoundRect(b.Inset(-f+0.5), r+f, r+f, paintengine2d.StrokePaint(col.WithAlpha(a), 1))
	}
}

// Hex parses a colour literal for engine code; invalid literals are black.
func Hex(s string) paintengine2d.Color { return hexColor(s) }

// Mix returns a blended opaque colour: t=0 → a, t=1 → b.
func Mix(a, b paintengine2d.Color, t float32) paintengine2d.Color {
	c := a.Lerp(b, t)
	c.A = 1
	return c
}

// Shade lightens (f > 0) or darkens (f < 0) c towards white / black.
func Shade(c paintengine2d.Color, f float32) paintengine2d.Color {
	if f >= 0 {
		return Mix(c, paintengine2d.RGB(1, 1, 1), f)
	}
	return Mix(c, paintengine2d.RGB(0, 0, 0), -f)
}

// Luma is the relative luminance of c (sRGB, approximate).
func Luma(c paintengine2d.Color) float32 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}

// Contrast returns black or white, whichever reads better on bg.
func Contrast(bg paintengine2d.Color) paintengine2d.Color {
	if Luma(bg) > 0.55 {
		return paintengine2d.RGB(0, 0, 0)
	}
	return paintengine2d.RGB(1, 1, 1)
}
