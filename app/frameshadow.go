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
	// Inside the window the patch must add nothing: the window paints
	// there itself, and a shadow under an opaque surface is wasted work.
	punchRoundRect(ctx, win, radius)
	return img
}

// paintFrameShadow blits the cached patch into the margin around win: four
// corner tiles at their natural size and four one-pixel strips stretched
// along the edges between them.
func (w *Window) paintFrameShadow(ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
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
