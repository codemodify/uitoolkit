package platform

import (
	"github.com/codemodify/paintengine2d"
)

// A window's silhouette: the part of its surface that is really there.
// Everything outside it is not the window at all — the desktop shows
// through, and a press lands on whatever is behind. A hole in the middle is
// nothing special: it is a second contour of the same shape, filled
// even-odd, exactly as Win32's CombineRgn(RGN_XOR) made one for
// SetWindowRgn.
//
// A shape is given either as a path or as a coverage mask, and the two are
// not interchangeable:
//
//   - A path (NewShape, NewShapeEvenOdd) is resolution-independent. It is
//     rasterised at the surface's device size, so it is exact at 1.75 as
//     well as at 1, and it costs nothing to keep. This is what a look, a
//     decoration or an app that draws its own silhouette wants.
//   - A mask (NewShapeMask) is 8-bit coverage, one byte a pixel — what a
//     skin has, because a skin's silhouette arrives as the alpha channel of
//     a bitmap. It is resampled to the surface's device size, so scaling it
//     far past its natural size softens its edge, as scaling any bitmap
//     does.
//
// Either way the window system is told the same thing: a list of
// rectangles. Wayland's wl_region and X11's XShape take rectangles and
// nothing else, so an arbitrary silhouette is rasterised to scanline runs
// before either can hear about it, and the region is hard-edged however
// smooth the painted shape is. That is the one thing about a shaped window
// that cannot be antialiased — on every toolkit, not just this one.
//
// All of a shape's geometry is device pixels in surface coordinates, like
// the rest of [Frame]. The app that builds the shape is handed the
// surface's device size and its scale and draws in those units, which is
// how a shape is exact at a fractional scale.
//
// A Shape is immutable once built and caches its own rasterisations, so
// handing the same shape back every frame allocates nothing. Like the rest
// of the toolkit it belongs to one goroutine.
type Shape struct {
	// A shape is a path or a mask, never both.
	path *paintengine2d.Path
	rule paintengine2d.FillRule
	// mask is maskH rows of maskStride bytes, the first maskW of each row
	// being coverage.
	mask                     []uint8
	maskW, maskH, maskStride int
	cache                    []*ShapeRaster
}

// shapeCacheSize is how many rasterisations a shape keeps. One covers the
// common case — the same window asking for the same shape at the same size
// every frame — and the rest cover a shape shared between windows of
// different sizes, which would otherwise rasterise afresh on every frame.
const shapeCacheSize = 4

// ShapeRectLimit is the most rectangles a rasterised shape may produce
// before it gives up and reports its bounding box instead. A silhouette
// rasterises to roughly one rectangle per scanline it is not straight on —
// a 875 px circle needs about 630 — so this leaves room for a very
// complicated one and still refuses to hand a compositor a region built
// from a dithered or noisy mask, which is a mistake in the mask rather
// than a shape anyone meant.
const ShapeRectLimit = 1 << 16

// ShapeRaster is a shape worked out for one surface size, in device pixels.
// It belongs to the shape and must not be modified.
type ShapeRaster struct {
	// W and H are the surface size it was rasterised for.
	W, H int
	// Mask is W*H bytes of coverage, one a pixel, row-major with stride W:
	// 0 outside the shape, 255 well inside it, and the antialiased values
	// between along its edge. Capture crops with it and the shadow
	// generator blurs it.
	Mask []uint8
	// Rects is the silhouette as rectangles: every pixel at least half
	// covered. This is what the input region, the bounding shape and
	// hit-testing all use, so the app and the compositor agree to the
	// pixel about where the window is.
	Rects []FrameRect
	// Opaque is the fully covered pixels only — what a compositor may take
	// as solid, and only if the window paints them solid.
	Opaque []FrameRect
	// Bounds is the smallest box holding Rects (empty when the shape is).
	Bounds FrameRect
	// Clamped: the shape needed more than [ShapeRectLimit] rectangles, so
	// Rects and Opaque are Bounds instead. The mask is still exact.
	Clamped bool
}

// Empty reports whether the shape covers nothing at this size: a window
// that is not there at all.
func (r *ShapeRaster) Empty() bool { return r == nil || len(r.Rects) == 0 }

// Contains reports whether device pixel x, y is inside the shape — the same
// answer the compositor gives, because it is asked of the same rectangles.
func (r *ShapeRaster) Contains(x, y int) bool {
	if r == nil {
		return false
	}
	// The rectangles come out of the rasteriser in scanline order, so a
	// linear walk that stops once the rows are past y is short even for a
	// shape with hundreds of them.
	for _, b := range r.Rects {
		if b.Y > y {
			break
		}
		if y < b.Y+b.H && x >= b.X && x < b.X+b.W {
			return true
		}
	}
	return false
}

// Coverage is the coverage of device pixel x, y: 0 outside the shape, 255
// well inside it, partial along its edge.
func (r *ShapeRaster) Coverage(x, y int) uint8 {
	if r == nil || x < 0 || y < 0 || x >= r.W || y >= r.H {
		return 0
	}
	return r.Mask[y*r.W+x]
}

// NewShape is the silhouette p encloses, filled by the nonzero winding
// rule: the shape every simple outline wants. p is copied, so the caller
// may go on using it.
func NewShape(p *paintengine2d.Path) *Shape {
	return newShapePath(p, paintengine2d.FillNonZero)
}

// NewShapeEvenOdd is the silhouette p encloses, filled even-odd: a contour
// inside another one becomes a hole you can see and click through. A ring
// is an outer rounded rectangle and an inner circle in one path.
func NewShapeEvenOdd(p *paintengine2d.Path) *Shape {
	return newShapePath(p, paintengine2d.FillEvenOdd)
}

func newShapePath(p *paintengine2d.Path, rule paintengine2d.FillRule) *Shape {
	if p == nil || p.Empty() {
		return nil
	}
	return &Shape{path: p.Clone(), rule: rule}
}

// NewShapeMask is the silhouette an 8-bit coverage mask describes: w*h
// bytes, row-major, with the given stride (at least w), 0 outside the shape
// and 255 inside it — the alpha channel of a skin's bitmap. The mask is
// copied. It is resampled to whatever size the surface is, so a mask at the
// size it was drawn for stays crisp and one stretched a long way does not.
func NewShapeMask(mask []uint8, w, h, stride int) *Shape {
	if w < 1 || h < 1 || stride < w || len(mask) < (h-1)*stride+w {
		return nil
	}
	cp := make([]uint8, h*w)
	for y := 0; y < h; y++ {
		copy(cp[y*w:(y+1)*w], mask[y*stride:y*stride+w])
	}
	return &Shape{mask: cp, maskW: w, maskH: h, maskStride: w}
}

// NewShapeImage is the silhouette an image's alpha channel describes: the
// skin case, where the silhouette and the artwork are the same file. Both
// of paintengine2d's formats work — a coverage mask as it is, a
// premultiplied RGBA image by its alpha.
func NewShapeImage(img *paintengine2d.Image) *Shape {
	if img == nil || img.Width < 1 || img.Height < 1 {
		return nil
	}
	w, h := img.Width, img.Height
	stride := img.RowStride()
	bpp := img.BytesPerPixel()
	mask := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		row := img.Pix[y*stride:]
		out := mask[y*w:]
		if bpp == 1 {
			copy(out[:w], row[:w])
			continue
		}
		for x := 0; x < w; x++ {
			out[x] = row[x*bpp+3] // premultiplied RGBA8: alpha last
		}
	}
	return &Shape{mask: mask, maskW: w, maskH: h, maskStride: w}
}

// ShapeRect is the plain rectangular silhouette — what every window has
// without asking. It exists so a caller can drop a shape back to the
// default without a nil check of its own.
func ShapeRect(w, h float32) *Shape {
	p := paintengine2d.NewPath()
	p.AddRect(paintengine2d.XYWH(0, 0, w, h))
	return NewShape(p)
}

// ShapeRoundRect is a rounded rectangle, its corners top-left clockwise.
func ShapeRoundRect(r paintengine2d.Rect, radius [4]float32) *Shape {
	p := paintengine2d.NewPath()
	p.AddRoundRectCorners(r, radius[0], radius[1], radius[2], radius[3])
	return NewShape(p)
}

// ShapeEllipse is the ellipse filling r (a circle when r is square).
func ShapeEllipse(r paintengine2d.Rect) *Shape {
	p := paintengine2d.NewPath()
	p.AddEllipse(paintengine2d.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2), r.Dx()/2, r.Dy()/2)
	return NewShape(p)
}

// Path is the shape's outline and fill rule, or nil for a mask-backed
// shape. The path belongs to the shape: clone it before changing it.
func (s *Shape) Path() (*paintengine2d.Path, paintengine2d.FillRule) {
	if s == nil {
		return nil, paintengine2d.FillNonZero
	}
	return s.path, s.rule
}

// Raster is the shape worked out for a surface w*h device pixels, from the
// cache when it has been asked for before. The result belongs to the shape
// and is read-only; a repeat call allocates nothing.
func (s *Shape) Raster(w, h int) *ShapeRaster {
	if s == nil || w < 1 || h < 1 {
		return nil
	}
	for i, r := range s.cache {
		if r.W == w && r.H == h {
			// Move to front: a shape shared between two windows keeps
			// both rasterisations warm.
			if i > 0 {
				copy(s.cache[1:i+1], s.cache[:i])
				s.cache[0] = r
			}
			return r
		}
	}
	r := s.rasterise(w, h)
	if len(s.cache) < shapeCacheSize {
		s.cache = append(s.cache, nil)
	}
	copy(s.cache[1:], s.cache[:len(s.cache)-1])
	s.cache[0] = r
	return r
}

// rasterise draws the shape at w*h and turns its coverage into rectangles.
func (s *Shape) rasterise(w, h int) *ShapeRaster {
	r := &ShapeRaster{W: w, H: h}
	if s.path != nil {
		r.Mask = rasterPathMask(s.path, s.rule, w, h)
	} else {
		r.Mask = resampleMask(s.mask, s.maskW, s.maskH, s.maskStride, w, h)
	}
	// Half covered is in: a pixel the shape owns more of than it does not.
	// The same threshold decides the input region and the hit test, so the
	// app never believes it owns a pixel the compositor gave away.
	r.Rects = MaskRects(r.Mask, w, h, w, 128)
	r.Bounds = BoundsOfRects(r.Rects)
	if len(r.Rects) > ShapeRectLimit {
		r.Clamped = true
		r.Rects = []FrameRect{r.Bounds}
		r.Opaque = r.Rects
		return r
	}
	// Only a fully covered pixel may be called solid: the antialiased edge
	// is translucent, and claiming it opaque is visible corruption.
	r.Opaque = MaskRects(r.Mask, w, h, w, 255)
	if len(r.Opaque) > ShapeRectLimit {
		r.Opaque = nil
	}
	return r
}

// rasterPathMask draws the path white on nothing and hands back the alpha
// channel as a coverage mask.
//
// It rasterises into an RGBA image and picks the alpha out, rather than
// drawing straight into paintengine2d's own 8-bit mask format: the CPU
// rasteriser takes FormatA8 as a source (a glyph sheet, an icon) but not as
// a render target — every blend writes four bytes at a one-byte-per-pixel
// index — so an A8 target runs past the end of its own buffer. The scratch
// image is transient and dropped as soon as the mask is taken from it.
func rasterPathMask(path *paintengine2d.Path, rule paintengine2d.FillRule, w, h int) []uint8 {
	mask := make([]uint8, w*h)
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return mask
	}
	ctx.Clear(paintengine2d.Transparent)
	paint := paintengine2d.Fill(paintengine2d.White)
	paint.FillRule = rule
	paint.AntiAlias = true
	ctx.DrawPath(path, paint)
	stride := img.RowStride()
	for y := 0; y < h; y++ {
		row := img.Pix[y*stride:]
		out := mask[y*w:]
		for x := 0; x < w; x++ {
			out[x] = row[x*4+3] // premultiplied RGBA8: alpha last
		}
	}
	return mask
}

// resampleMask stretches a coverage mask to w*h, bilinearly: a skin's
// silhouette at a scale its artwork was not drawn for. At its natural size
// it is a straight copy.
func resampleMask(mask []uint8, mw, mh, stride, w, h int) []uint8 {
	out := make([]uint8, w*h)
	if mw == w && mh == h {
		for y := 0; y < h; y++ {
			copy(out[y*w:(y+1)*w], mask[y*stride:y*stride+w])
		}
		return out
	}
	// Sample at pixel centres, so the edges of the mask are not half a
	// pixel adrift of the edges of the surface.
	sx := float32(mw) / float32(w)
	sy := float32(mh) / float32(h)
	for y := 0; y < h; y++ {
		fy := (float32(y)+0.5)*sy - 0.5
		y0, ty := split(fy, mh)
		y1 := min(y0+1, mh-1)
		row0 := mask[y0*stride:]
		row1 := mask[y1*stride:]
		dst := out[y*w:]
		for x := 0; x < w; x++ {
			fx := (float32(x)+0.5)*sx - 0.5
			x0, tx := split(fx, mw)
			x1 := min(x0+1, mw-1)
			a := lerp8(row0[x0], row0[x1], tx)
			b := lerp8(row1[x0], row1[x1], tx)
			dst[x] = lerp8(a, b, ty)
		}
	}
	return out
}

// split is the sample's whole part, clamped into [0, n), and the fraction
// past it.
func split(v float32, n int) (int, float32) {
	if v <= 0 {
		return 0, 0
	}
	i := int(v)
	if i >= n-1 {
		return n - 1, 0
	}
	return i, v - float32(i)
}

func lerp8(a, b uint8, t float32) uint8 {
	return uint8(float32(a) + (float32(b)-float32(a))*t + 0.5)
}

// MaskRects turns an 8-bit coverage mask into the axis-aligned rectangles
// covering every pixel at or above threshold, in scanline order. Rows with
// identical runs are merged into one taller rectangle, which is what keeps
// the count down: a circle's waist barely changes width from row to row, so
// a 875 px circle needs about 630 rectangles rather than 875.
//
// mask is w*h bytes, row-major, with the given stride (at least w). The
// rectangles come out in the mask's own pixel unit; the caller scales them.
func MaskRects(mask []uint8, w, h, stride int, threshold uint8) []FrameRect {
	if w < 1 || h < 1 || stride < w || len(mask) < (h-1)*stride+w {
		return nil
	}
	var out []FrameRect
	// open holds the rectangles of the row group being extended, each one
	// started when the group began; prev is that group's runs.
	var open []FrameRect
	var prev []int // flattened start, end pairs
	cur := make([]int, 0, 16)
	flush := func() {
		out = append(out, open...)
		open = open[:0]
	}
	for y := 0; y < h; y++ {
		cur = cur[:0]
		row := mask[y*stride : y*stride+w]
		x := 0
		for x < w {
			for x < w && row[x] < threshold {
				x++
			}
			if x >= w {
				break
			}
			s := x
			for x < w && row[x] >= threshold {
				x++
			}
			cur = append(cur, s, x)
		}
		if sameRuns(cur, prev) {
			for i := range open {
				open[i].H++
			}
			continue
		}
		flush()
		for i := 0; i < len(cur); i += 2 {
			open = append(open, FrameRect{X: cur[i], Y: y, W: cur[i+1] - cur[i], H: 1})
		}
		prev = append(prev[:0], cur...)
	}
	flush()
	return out
}

// sameRuns reports whether two rows hold the same runs. Two empty rows are
// not the same: an empty row starts no rectangle to extend.
func sameRuns(a, b []int) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// BoundsOfRects is the smallest rectangle holding every one of rects (the
// zero rectangle for none).
func BoundsOfRects(rects []FrameRect) FrameRect {
	if len(rects) == 0 {
		return FrameRect{}
	}
	x0, y0 := rects[0].X, rects[0].Y
	x1, y1 := x0+rects[0].W, y0+rects[0].H
	for _, r := range rects[1:] {
		x0, y0 = min(x0, r.X), min(y0, r.Y)
		x1, y1 = max(x1, r.X+r.W), max(y1, r.Y+r.H)
	}
	return FrameRect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// ScaleRects converts rects from device pixels to the logical unit a
// wl_region speaks. grow rounds every rectangle outwards — right for an
// input region, which must never lose a pixel the shape covers — and
// inwards otherwise, which is what an opaque region needs: claiming a
// translucent pixel solid is visible corruption.
//
// At a fractional scale the two differ, and a shape converted the wrong way
// either goes deaf along its edge or leaves a hard seam around it.
func ScaleRects(rects []FrameRect, scale float32, grow bool) []FrameRect {
	if scale <= 0 || scale == 1 || len(rects) == 0 {
		return rects
	}
	out := make([]FrameRect, 0, len(rects))
	for _, r := range rects {
		var x0, y0, x1, y1 int
		if grow {
			x0 = floorDiv(r.X, scale)
			y0 = floorDiv(r.Y, scale)
			x1 = ceilDiv(r.X+r.W, scale)
			y1 = ceilDiv(r.Y+r.H, scale)
		} else {
			x0 = ceilDiv(r.X, scale)
			y0 = ceilDiv(r.Y, scale)
			x1 = floorDiv(r.X+r.W, scale)
			y1 = floorDiv(r.Y+r.H, scale)
		}
		if x1 > x0 && y1 > y0 {
			out = append(out, FrameRect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0})
		}
	}
	return out
}

// floorDiv and ceilDiv divide a device coordinate by the scale. They take
// the sign into account: a shape's rectangle may start left of or above the
// surface origin (a shadow's margin is part of those coordinates), and Go's
// conversion to int truncates towards zero, which would round the wrong way
// there.
func floorDiv(v int, scale float32) int {
	q := float32(v) / scale
	i := int(q)
	if q < 0 && float32(i) != q {
		i--
	}
	return i
}

func ceilDiv(v int, scale float32) int {
	q := float32(v) / scale
	i := int(q)
	if q > 0 && float32(i) != q {
		i++
	}
	return i
}

// OffsetRects moves every rectangle by dx, dy — a shape stated in the
// visible window's coordinates becoming one in the surface's, which is the
// same box moved by the shadow margin.
func OffsetRects(rects []FrameRect, dx, dy int) []FrameRect {
	if len(rects) == 0 || (dx == 0 && dy == 0) {
		return rects
	}
	out := make([]FrameRect, len(rects))
	for i, r := range rects {
		out[i] = FrameRect{X: r.X + dx, Y: r.Y + dy, W: r.W, H: r.H}
	}
	return out
}

// RectsContain reports whether device pixel x, y falls in any of rects.
// Hit-testing a window's shape goes through here, so it agrees with the
// region the compositor was given rather than with the path it came from.
func RectsContain(rects []FrameRect, x, y int) bool {
	for _, r := range rects {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return true
		}
	}
	return false
}
