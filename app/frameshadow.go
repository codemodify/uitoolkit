package app

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// The translucent part of a toolkit-drawn frame: the drop shadow in the
// margin around the visible window, and the rounded corners cut out of it.
//
// The shadow is rasterised once into a nine-patch — four corner tiles and
// the one-pixel strips between them — and blitted from there afterwards, so
// a window that repaints for any other reason never rasterises a gradient
// again, and the patch survives a resize (it depends on the look, the
// state, the scale and the margin, not on the window's size). The corners
// are punched out of the painted surface with the dest-out operator, which
// is why everything else is painted rectangular and fast, exactly as an
// opaque window is.

// frameShadowKey is what a cached shadow patch is good for.
type frameShadowKey struct {
	look   style.LookAndFeel
	margin [4]int
	radius [4]float32
	active bool
	custom bool
}

// frameShadow is a window's cached shadow nine-patch.
type frameShadow struct {
	key frameShadowKey
	img *paintengine2d.Image
	// m is the margin the patch was built for and ext how far the shadow's
	// corner tiles reach into the window, both in device pixels.
	m       [4]int // top, right, bottom, left
	extX    float32
	extY    float32
	invalid bool
}

// drop forgets the patch (the look, the state or the margin changed).
func (s *frameShadow) drop() {
	s.img = nil
	s.key = frameShadowKey{}
}

// shadowPatch is the cached nine-patch for the window's current frame, or
// nil when the frame drops no shadow.
func (w *Window) shadowPatch() *frameShadow {
	g := w.geom
	if !g.framed || !g.shadow || w.caption == nil {
		return nil
	}
	st := w.caption.DecorationState()
	key := frameShadowKey{
		look:   w.look,
		margin: [4]int{g.margin.Top, g.margin.Right, g.margin.Bottom, g.margin.Left},
		radius: g.radius,
		active: st.Active,
		custom: st.Custom,
	}
	if w.shadow.img != nil && w.shadow.key == key {
		return &w.shadow
	}
	w.shadow.key = key
	w.shadow.m = key.margin
	w.shadow.img = buildShadowPatch(w.look, st, g.margin, g.radius, &w.shadow.extX, &w.shadow.extY)
	if w.shadow.img == nil {
		return nil
	}
	return &w.shadow
}

// buildShadowPatch rasterises the look's window shadow around a template
// window just large enough that its corners do not meet, and cuts the
// window's own rounded shape out of it, so the patch adds nothing where the
// window itself paints.
func buildShadowPatch(lk style.LookAndFeel, st style.DecorationState, m platform.FrameInsets, radius [4]float32, extX, extY *float32) *paintengine2d.Image {
	mt, mr, mb, ml := float32(m.Top), float32(m.Right), float32(m.Bottom), float32(m.Left)
	rmax := float32(0)
	for _, r := range radius {
		rmax = max(rmax, r)
	}
	// The shadow's profile varies within about its own reach plus the
	// corner's radius of each corner; past that it is the same all along
	// the edge, which is what the one-pixel strips carry.
	ex := float32(math.Ceil(float64(max(ml, mr) + rmax + 1)))
	ey := float32(math.Ceil(float64(max(mt, mb) + rmax + 1)))
	*extX, *extY = ex, ey
	pw := int(ml + ex + 1 + ex + mr)
	ph := int(mt + ey + 1 + ey + mb)
	if pw < 3 || ph < 3 || pw > 4096 || ph > 4096 {
		return nil
	}
	img := paintengine2d.NewImage(pw, ph)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return nil
	}
	ctx.Clear(paintengine2d.Transparent)
	win := paintengine2d.XYWH(ml, mt, float32(pw)-ml-mr, float32(ph)-mt-mb)
	style.DrawDecorationShadowOf(lk, ctx, win, st)
	// Inside the window the patch must hold nothing: the window paints
	// there itself, and the patch goes on last, over it.
	eraseRoundRect(ctx, win, radius)
	return img
}

// ---- the shadow around a silhouette ---------------------------------------
//
// A nine-patch cannot carry the shadow of an arbitrary shape: it works
// because a rectangle's shadow is the same all along each edge, and a
// silhouette's is not. A shaped window's shadow is a real blur of its
// coverage mask instead — one image the size of the window and its margin,
// built once per (shape, size, look, state) and blitted whole, so it costs
// a resize rather than a frame.
//
// The look's own numbers decide what the blur looks like: the margin it
// asked for is the reach, the difference between the top and bottom margins
// is how far its shadow falls, and its colour comes from its own rectangular
// shadow, probed once. A shaped window in any of the 121 packs therefore
// casts the shadow of that pack rather than a generic one.

// shapeShadow is a shaped window's cached shadow image.
type shapeShadow struct {
	key shapeShadowKey
	// img is the shadow's coverage (FormatA8), drawn tinted col.
	img *paintengine2d.Image
	col paintengine2d.Color
	// ox, oy are where the image goes relative to the visible window's
	// top-left corner, in device pixels (negative: up and to the left).
	ox, oy int
}

type shapeShadowKey struct {
	look   style.LookAndFeel
	raster *platform.ShapeRaster
	margin [4]int
	active bool
	custom bool
}

// shapeShadowPatch is the blurred silhouette for the window's current
// shape, or nil when it casts no shadow.
func (w *Window) shapeShadowPatch(r *platform.ShapeRaster) *shapeShadow {
	g := w.geom
	if r == nil || !g.framed || !g.shadow || w.caption == nil {
		return nil
	}
	st := w.caption.DecorationState()
	key := shapeShadowKey{
		look:   w.look,
		raster: r,
		margin: [4]int{g.margin.Top, g.margin.Right, g.margin.Bottom, g.margin.Left},
		active: st.Active,
		custom: st.Custom,
	}
	if w.shapeShade.img != nil && w.shapeShade.key == key {
		return &w.shapeShade
	}
	w.shapeShade = shapeShadow{key: key, col: probeShadowColor(w.look, st, g.margin)}
	w.shapeShade.img = buildShadeImage(r, g.margin, w.shapeShade.col)
	w.shapeShade.ox, w.shapeShade.oy = -g.margin.Left, -g.margin.Top
	if w.shapeShade.img == nil {
		return nil
	}
	return &w.shapeShade
}

// buildShadeImage is silhouette r blurred into margin m around it, with
// r's own shape erased from it: the image goes at r's top-left corner less
// the margin. A shaped window's shadow and a shaped popup's are both this.
//
// It is coverage only (FormatA8), drawn tinted in the shadow's colour: a
// shadow is one colour at varying strength, and as RGBA it was four times
// the size — a whole window's worth of pixels, 9.9 MB of Lantern's heap
// for its two windows at 1.75x, and as much again in the GPU's copy.
func buildShadeImage(r *platform.ShapeRaster, m platform.FrameInsets, col paintengine2d.Color) *paintengine2d.Image {
	ml, mr := m.Left, m.Right
	mt, mb := m.Top, m.Bottom
	pw, ph := r.W+ml+mr, r.H+mt+mb
	if pw < 3 || ph < 3 || pw > 8192 || ph > 8192 || col.A <= 0 {
		return nil
	}
	// The reach sideways is the blur; the difference between the top and
	// bottom margins is how far the look drops its shadow, since a shadow
	// offset down needs that much more room below than above.
	radius := max((ml+mr)/2, 1)
	dy := (mb - mt) / 2
	// Coverage of the silhouette, laid into the larger box and blurred.
	img := paintengine2d.NewImageA8(pw, ph)
	cov := img.Pix
	stride := img.RowStride()
	for y := 0; y < r.H; y++ {
		if yy := y + mt + dy; yy >= 0 && yy < ph {
			copy(cov[yy*stride+ml:yy*stride+ml+r.W], r.Mask[y*r.W:(y+1)*r.W])
		}
	}
	blurMask(cov, pw, ph, radius)
	// Inside the window the image must hold nothing: the window paints
	// there itself, and this goes on over the top of it.
	for y := 0; y < r.H; y++ {
		row := cov[(y+mt)*stride+ml:]
		for x, in := range r.Mask[y*r.W : (y+1)*r.W] {
			if in != 0 {
				row[x] = uint8((int(row[x])*(255-int(in)) + 127) / 255)
			}
		}
	}
	return img
}

// probeShadowColor asks the look for its own rectangular shadow and reads
// the pixel just outside the window's edge: the densest part of the shadow,
// which is the colour a blurred silhouette should be tinted with. It is one
// small rasterisation per cache key, not per frame.
func probeShadowColor(lk style.LookAndFeel, st style.DecorationState, m platform.FrameInsets) paintengine2d.Color {
	ml, mt := max(m.Left, 1), max(m.Top, 1)
	pw, ph := ml*2+8, mt*2+8
	img := paintengine2d.NewImage(pw, ph)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return paintengine2d.Transparent
	}
	ctx.Clear(paintengine2d.Transparent)
	win := paintengine2d.XYWH(float32(ml), float32(mt), float32(pw-2*ml), float32(ph-2*mt))
	style.DrawDecorationShadowOf(lk, ctx, win, st)
	// Just outside the left edge, half way down: past the window itself
	// and in the thickest part of the shadow.
	c := img.NRGBAAt(ml-1, ph/2)
	return paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255)
}

// blurMask box-blurs an 8-bit coverage mask in place, three passes each
// way, which is close enough to a Gaussian that no one can tell and far
// cheaper. radius is in pixels.
func blurMask(mask []uint8, w, h, radius int) {
	if radius < 1 || w < 1 || h < 1 {
		return
	}
	// Three box passes of a third of the reach each add up to the reach.
	r := max(radius/3, 1)
	tmp := make([]uint8, len(mask))
	for pass := 0; pass < 3; pass++ {
		boxBlurRows(mask, tmp, w, h, r)
		boxBlurCols(tmp, mask, w, h, r)
	}
}

// boxBlurRows averages each row of src into dst over a window of 2r+1.
func boxBlurRows(src, dst []uint8, w, h, r int) {
	win := 2*r + 1
	for y := 0; y < h; y++ {
		row := src[y*w : (y+1)*w]
		out := dst[y*w : (y+1)*w]
		var sum int
		for i := -r; i <= r; i++ {
			sum += int(row[clampIdx(i, w)])
		}
		for x := 0; x < w; x++ {
			out[x] = uint8(sum / win)
			sum += int(row[clampIdx(x+r+1, w)]) - int(row[clampIdx(x-r, w)])
		}
	}
}

// boxBlurCols is boxBlurRows down the columns.
func boxBlurCols(src, dst []uint8, w, h, r int) {
	win := 2*r + 1
	for x := 0; x < w; x++ {
		var sum int
		for i := -r; i <= r; i++ {
			sum += int(src[clampIdx(i, h)*w+x])
		}
		for y := 0; y < h; y++ {
			dst[y*w+x] = uint8(sum / win)
			sum += int(src[clampIdx(y+r+1, h)*w+x]) - int(src[clampIdx(y-r, h)*w+x])
		}
	}
}

// clampIdx holds an index inside [0, n): the edges of the mask repeat, so
// a shape that runs to the edge of its box does not fade there.
func clampIdx(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// paintShapeShadow blits the blurred silhouette into place.
func (w *Window) paintShapeShadow(ctx *paintengine2d.Context, r *platform.ShapeRaster, dirty *paintengine2d.Damage) {
	s := w.shapeShadowPatch(r)
	if s == nil || s.img == nil {
		return
	}
	win := w.windowBox()
	dst := paintengine2d.XYWH(win.Min.X+float32(s.ox), win.Min.Y+float32(s.oy),
		float32(s.img.Width), float32(s.img.Height))
	if dirty != nil && !dirty.Empty() && !dirty.Overlaps(dst) {
		return
	}
	ctx.DrawImageRectPaint(s.img, paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height)), dst,
		paintengine2d.Paint{Color: s.col, Filter: paintengine2d.FilterNearest})
}

// paintFrameShadow blits the cached patch into the margin around win: four
// corner tiles at their natural size and four one-pixel strips stretched
// along the edges between them. A shaped window takes the blurred
// silhouette instead — a nine-patch cannot follow a silhouette's edge.
func (w *Window) paintFrameShadow(ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	if r := w.shapeRaster(); r != nil {
		w.paintShapeShadow(ctx, r, dirty)
		return
	}
	s := w.shadowPatch()
	if s == nil || s.img == nil {
		return
	}
	g := w.geom
	win := g.window
	mt, mr := float32(g.margin.Top), float32(g.margin.Right)
	mb, ml := float32(g.margin.Bottom), float32(g.margin.Left)
	reach := paintengine2d.Rect{
		Min: paintengine2d.Pt(win.Min.X-ml, win.Min.Y-mt),
		Max: paintengine2d.Pt(win.Max.X+mr, win.Max.Y+mb),
	}
	if dirty != nil && !dirty.Empty() && !dirty.Overlaps(reach) {
		return
	}
	// Corner tiles shrink when the window is too small to hold both.
	ex := min(s.extX, float32(math.Floor(float64(win.Dx()/2))))
	ey := min(s.extY, float32(math.Floor(float64(win.Dy()/2))))
	if ex < 1 || ey < 1 {
		return
	}
	iw, ih := float32(s.img.Width), float32(s.img.Height)
	paint := paintengine2d.Paint{Color: paintengine2d.White, Filter: paintengine2d.FilterNearest}
	blit := func(src, dst paintengine2d.Rect) {
		if src.Empty() || dst.Empty() {
			return
		}
		if dirty != nil && !dirty.Empty() && !dirty.Overlaps(dst) {
			return
		}
		ctx.DrawImageRectPaint(s.img, src, dst, paint)
	}
	// Source bands: margin + corner extent, the single stretched pixel,
	// then the far corner extent + margin.
	sx0, sy0 := ml+s.extX, mt+s.extY
	dx0, dx1 := win.Min.X+ex, win.Max.X-ex
	dy0, dy1 := win.Min.Y+ey, win.Max.Y-ey
	// Corners.
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(0, 0), Max: paintengine2d.Pt(ml+ex, mt+ey)},
		paintengine2d.Rect{Min: paintengine2d.Pt(win.Min.X-ml, win.Min.Y-mt), Max: paintengine2d.Pt(dx0, dy0)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(iw-mr-ex, 0), Max: paintengine2d.Pt(iw, mt+ey)},
		paintengine2d.Rect{Min: paintengine2d.Pt(dx1, win.Min.Y-mt), Max: paintengine2d.Pt(win.Max.X+mr, dy0)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(0, ih-mb-ey), Max: paintengine2d.Pt(ml+ex, ih)},
		paintengine2d.Rect{Min: paintengine2d.Pt(win.Min.X-ml, dy1), Max: paintengine2d.Pt(dx0, win.Max.Y+mb)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(iw-mr-ex, ih-mb-ey), Max: paintengine2d.Pt(iw, ih)},
		paintengine2d.Rect{Min: paintengine2d.Pt(dx1, dy1), Max: paintengine2d.Pt(win.Max.X+mr, win.Max.Y+mb)})
	// Edges: one pixel of the patch, stretched.
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(sx0, 0), Max: paintengine2d.Pt(sx0+1, mt+ey)},
		paintengine2d.Rect{Min: paintengine2d.Pt(dx0, win.Min.Y-mt), Max: paintengine2d.Pt(dx1, dy0)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(sx0, ih-mb-ey), Max: paintengine2d.Pt(sx0+1, ih)},
		paintengine2d.Rect{Min: paintengine2d.Pt(dx0, dy1), Max: paintengine2d.Pt(dx1, win.Max.Y+mb)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(0, sy0), Max: paintengine2d.Pt(ml+ex, sy0+1)},
		paintengine2d.Rect{Min: paintengine2d.Pt(win.Min.X-ml, dy0), Max: paintengine2d.Pt(dx0, dy1)})
	blit(paintengine2d.Rect{Min: paintengine2d.Pt(iw-mr-ex, sy0), Max: paintengine2d.Pt(iw, sy0+1)},
		paintengine2d.Rect{Min: paintengine2d.Pt(dx1, dy0), Max: paintengine2d.Pt(win.Max.X+mr, dy1)})
}

// paintFrameCorners cuts the window's rounded corners out of everything
// painted so far, so the desktop shows through them.
func (w *Window) paintFrameCorners(ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	g := w.geom
	if !g.framed || g.radius == [4]float32{} {
		return
	}
	if w.shapeRaster() != nil {
		// A silhouette replaces the corners: it is punched out of the same
		// pixels, and it already runs round them.
		return
	}
	win := g.window
	for i, r := range g.radius {
		if r <= 0 {
			continue
		}
		box := cornerBox(win, i, r)
		if dirty != nil && !dirty.Empty() && !dirty.Overlaps(box) {
			continue
		}
		ctx.Save()
		ctx.ClipRect(box)
		punchRoundRect(ctx, win, g.radius)
		ctx.Restore()
	}
}

// cornerBox is the square a corner's curve lives in: i is the corner
// (top-left clockwise), r its radius.
func cornerBox(win paintengine2d.Rect, i int, r float32) paintengine2d.Rect {
	switch i {
	case 0:
		return paintengine2d.XYWH(win.Min.X, win.Min.Y, r, r)
	case 1:
		return paintengine2d.XYWH(win.Max.X-r, win.Min.Y, r, r)
	case 2:
		return paintengine2d.XYWH(win.Max.X-r, win.Max.Y-r, r, r)
	}
	return paintengine2d.XYWH(win.Min.X, win.Max.Y-r, r, r)
}

// eraseRoundRect erases the rounded window b itself, leaving what lies
// outside it (a shadow's nine-patch keeps only what falls beyond the
// window's own shape).
func eraseRoundRect(ctx *paintengine2d.Context, b paintengine2d.Rect, radius [4]float32) {
	if ctx == nil || b.Empty() {
		return
	}
	p := paintengine2d.NewPath()
	addRoundRectRadii(p, b, radius)
	ctx.DrawPath(p, paintengine2d.Paint{
		Color:     paintengine2d.White,
		Blend:     paintengine2d.BlendDestOut,
		AntiAlias: true,
	})
}

// punchRoundRect erases everything outside the rounded window b from the
// current clip (paintengine2d's dest-out operator: the source's alpha is
// the eraser, its colour plays no part).
func punchRoundRect(ctx *paintengine2d.Context, b paintengine2d.Rect, radius [4]float32) {
	if ctx == nil || b.Empty() {
		return
	}
	outside := paintengine2d.NewPath()
	outside.AddRect(b.Inset(-1))
	addRoundRectRadii(outside, b, radius)
	ctx.DrawPath(outside, paintengine2d.Paint{
		Color:     paintengine2d.White,
		Blend:     paintengine2d.BlendDestOut,
		FillRule:  paintengine2d.FillEvenOdd,
		AntiAlias: true,
	})
}

// addRoundRectRadii appends b with its four corner radii (top-left
// clockwise) to p, each clamped to half the shorter side: a window frame
// rounds only the corners its look rounds (XP and Aqua the top ones,
// Windows 11 and macOS all four).
func addRoundRectRadii(p *paintengine2d.Path, b paintengine2d.Rect, radius [4]float32) {
	lim := min(b.Dx(), b.Dy()) * 0.5
	r := radius
	for i := range r {
		r[i] = max(min(r[i], lim), 0)
	}
	if r == [4]float32{} {
		p.AddRect(b)
		return
	}
	p.AddRoundRectCorners(b, r[0], r[1], r[2], r[3])
}
