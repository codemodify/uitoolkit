package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A window's silhouette, and the glass behind it.
//
// A shaped window is an ordinary window in every other way: the same widget
// tree, the same keyboard handling, the same accessibility tree. The shape
// says which of its pixels are really there — everything else shows the
// desktop and clicks through to whatever is behind, a hole in the middle as
// much as the space around a disc. docs/shapes.md has the whole picture.
//
// The shape lives with the rest of the frame state (platform.Frame), so it
// survives a resize, a scale change and a trip through the compositor, and
// it is dropped in the states where it makes no sense — maximized,
// full-screen and tiled — exactly as the frame's corners and shadow are
// dropped there today.

// SetShape gives the window a silhouette, or nil for the plain rectangle
// every window has by default. The shape is stated in the visible window's
// device pixels, so a 500-logical-pixel window at 1.75 is shaped at 875.
//
// A shape that has to follow the window's size — nearly all of them do —
// wants [Window.SetShapeFunc] instead: this one is rasterised at whatever
// size the window happens to be, and a path built for another size is
// stretched to fit rather than redrawn.
func (w *Window) SetShape(s *platform.Shape) {
	w.SetShapeFunc(nil)
	if w == nil || w.shape == s {
		return
	}
	w.shape = s
	w.shapeChanged()
}

// SetShapeFunc gives the window a silhouette that follows its size: fn is
// called with the visible window's size and scale, both in device pixels,
// whenever either changes, and returns the shape for that size (nil for
// none). This is what a shape that is "a disc filling the window" or "a
// ring with a hole a third of the way across" wants — it is redrawn at
// every size rather than stretched, so its edge is exact at 1.75 as at 1.
//
// nil drops the callback and leaves whatever [Window.SetShape] last set.
func (w *Window) SetShapeFunc(fn func(size paintengine2d.Point, scale float32) *platform.Shape) {
	if w == nil {
		return
	}
	w.shapeFn = fn
	w.shapeCur, w.shapeCurW, w.shapeCurH = nil, 0, 0
	if fn != nil {
		w.shape = nil
	}
	w.shapeChanged()
}

// Shape is the silhouette in effect now: the one last set, or the one the
// callback built for the window's current size. nil means the window is the
// plain rectangle it has always been.
func (w *Window) Shape() *platform.Shape {
	if w == nil {
		return nil
	}
	return w.currentShape()
}

// ShapeActive reports whether the window's silhouette is in effect right
// now. It is false for a window with no shape, and false while the window
// is maximized, full-screen or tiled, where the shape is dropped — so an
// app that draws its own silhouette asks this before drawing it, and draws
// a plain rectangle when the answer is no. Painting a hole into a window
// that no longer has one is how a shaped app looks broken when it is
// maximized.
func (w *Window) ShapeActive() bool { return w.shapeRaster() != nil }

// SetShapeOpaque says the window paints every pixel inside its shape
// solid, so the compositor may skip whatever is behind them. It is off by
// default, because it is the one setting here that is visible corruption
// when it is wrong: a window that says this and then paints a translucent
// tint leaves the desktop showing through in stripes on some compositors
// and not others.
//
// Only fully covered pixels are ever claimed — never the antialiased edge —
// so turning it on for a shape whose interior really is solid is safe.
func (w *Window) SetShapeOpaque(on bool) {
	if w == nil || w.shapeOpaque == on {
		return
	}
	w.shapeOpaque = on
	w.shapeChanged()
}

// ShapeOpaque reports whether the window claims its shape's interior solid.
func (w *Window) ShapeOpaque() bool { return w != nil && w.shapeOpaque }

// SetGlass asks the desktop to blur what is *behind* the window, so a
// translucent window becomes real glass over the desktop rather than over
// its own pixels — Wayland's ext_background_effect_v1, KWin's
// _KDE_NET_WM_BLUR_BEHIND_REGION on X11.
//
// The window has to be translucent for it to show: glass blurs the
// desktop, it does not make an opaque window see-through. A desktop that
// cannot blur ignores the request, which is why a look that asks for glass
// must still look right without it — [Window.GlassAvailable] says which of
// the two the window is getting.
func (w *Window) SetGlass(on bool) {
	if w == nil || w.glass == on {
		return
	}
	w.glass = on
	w.shapeChanged()
}

// SetGlassTint is the colour the window's background takes over the blurred
// desktop. It has to be translucent — an opaque tint shows none of the blur
// — and a fully transparent one (the default) means "the look's own", which
// is [style.GlassTint].
//
// This is the app's say over how much of the desktop shows: a panel that is
// mostly glass wants a low alpha, a document window a high one.
func (w *Window) SetGlassTint(c paintengine2d.Color) {
	if w == nil || w.glassTint == c {
		return
	}
	w.glassTint = c
	w.shapeChanged()
}

// GlassTint is the colour set by [Window.SetGlassTint] (fully transparent
// when the window takes the look's own).
func (w *Window) GlassTint() paintengine2d.Color { return w.glassTint }

// Glass reports whether the window asked for glass (not whether it got it).
func (w *Window) Glass() bool { return w != nil && w.glass }

// GlassAvailable reports whether this desktop blurs behind a window at all.
// It changes while the window is up — KWin drops it when desktop effects
// are switched off — so a look asks afresh rather than once.
func (w *Window) GlassAvailable() bool {
	return w != nil && platform.SurfaceBlurBehind(w.surf)
}

// shapeChanged re-lays the window out: the silhouette is part of the frame
// the window system is told about, and the frame is decided in layout.
func (w *Window) shapeChanged() {
	if w == nil || w.Closed() {
		return
	}
	w.shadow.drop()
	w.eraser, w.eraserFor = nil, nil
	w.shapeShade = shapeShadow{}
	w.laid = false
	w.dropScene()
	w.fullInvalidate()
}

// currentShape is the shape the app asked for, built by the callback where
// there is one and remembered until the size or the scale it was built for
// changes. The memo matters: a callback returns a fresh Shape every time,
// and a fresh Shape has an empty rasterisation cache, so calling it once a
// frame would re-rasterise the silhouette on every frame — tens of
// milliseconds for a window of any size.
//
// It takes no notice of the window's state; shapeRaster does that.
func (w *Window) currentShape() *platform.Shape {
	if w.shapeFn == nil {
		return w.shape
	}
	ww, hh := w.shapeSize()
	sc := max(w.scale, 1)
	if w.shapeCur != nil && w.shapeCurW == ww && w.shapeCurH == hh && w.shapeCurScale == sc {
		return w.shapeCur
	}
	w.shapeCur = w.shapeFn(paintengine2d.Pt(float32(ww), float32(hh)), sc)
	w.shapeCurW, w.shapeCurH, w.shapeCurScale = ww, hh, sc
	return w.shapeCur
}

// shapeSize is the visible window's size in device pixels: the surface less
// the margin it is currently keeping for its shadow. The margin grows the
// surface around the window rather than eating into it, so this stays the
// size the app and the compositor agreed on even while the margin changes.
func (w *Window) shapeSize() (int, int) {
	ww, hh := w.surf.Size()
	m := w.sysFrame.Margin
	return max(ww-m.Width(), 1), max(hh-m.Height(), 1)
}

// shapeDropped reports whether the window's state leaves no room for a
// shape. A maximized, full-screen or tiled window fills a box the desktop
// chose and shares its edges with the screen or a neighbour: a silhouette
// there would leave gaps the user cannot click and the desktop did not
// expect, exactly as a corner radius or a drop shadow would, and those are
// already dropped in these states. An uncomposited screen keeps its shape —
// X11's bounding shape still cuts the window, hard-edged.
func (w *Window) shapeDropped() bool {
	st := w.state
	return st.Maximized || st.Fullscreen || st.Tiled != 0
}

// shapeRaster is the window's silhouette worked out for its current size,
// or nil when it has none or its state has dropped it. The rasterisation
// is the shape's own and cached there, so asking every frame costs a
// lookup.
func (w *Window) shapeRaster() *platform.ShapeRaster {
	if w == nil || w.Closed() || w.shapeDropped() {
		return nil
	}
	s := w.currentShape()
	if s == nil {
		return nil
	}
	ww, hh := w.shapeSize()
	return s.Raster(ww, hh)
}

// applyShapeToFrame adds the silhouette and the glass to the frame the
// window system is about to be told about. f already holds the margin, the
// resize band, the corners and the alpha the look asked for.
//
// A shape's rectangles are stated in the visible window's coordinates and
// the frame's are the surface's, so they move by the margin on the way in.
func (w *Window) applyShapeToFrame(f *platform.Frame, margin platform.FrameInsets) {
	r := w.shapeRaster()
	if r != nil {
		f.Shape = platform.OffsetRects(r.Rects, margin.Left, margin.Top)
		// The shape takes the whole job over: the resize band the frame
		// would have kept in the margin is not in the silhouette, so a
		// shaped window is resized from its own edges or not at all.
		f.Input = platform.FrameInsets{}
		// Transparent pixels are the whole point, so the buffer needs an
		// alpha channel whatever the look asked for.
		f.Alpha = true
		if w.shapeOpaque {
			f.Opaque = platform.OffsetRects(w.opaqueRegion(r), margin.Left, margin.Top)
		} else {
			// Non-nil and empty: "nothing here is certainly solid",
			// which is different from "keep the default".
			f.Opaque = []platform.FrameRect{}
		}
	}
	if !w.wantsGlass() {
		return
	}
	// Blur exactly where the window is. A compositor clips it to the
	// surface anyway, but saying so keeps the blur out of a shaped
	// window's hole, where it would smear the desktop for no reason.
	if r != nil {
		f.Blur = platform.OffsetRects(r.Rects, margin.Left, margin.Top)
	} else {
		ww, hh := w.shapeSize()
		box := platform.FrameRect{X: margin.Left, Y: margin.Top, W: ww, H: hh}
		f.Blur = platform.OpaqueRects(box, f.Radius, 1)
		if len(f.Blur) == 0 {
			f.Blur = []platform.FrameRect{box}
		}
	}
	f.Alpha = true
}

// wantsGlass reports whether the window should ask for blur behind it: the
// app asked, or the look asks for every window of its era, and the desktop
// can actually do it. A desktop that cannot leaves the look painting
// whatever approximation it has.
func (w *Window) wantsGlass() bool {
	if w == nil || w.Closed() {
		return false
	}
	if !w.glass && !style.GlassBehind(w.look) {
		return false
	}
	return platform.SurfaceBlurBehind(w.surf)
}

// ---- painting -------------------------------------------------------------

// paintWindowShape cuts everything outside the window's silhouette out of
// what has been painted, so the desktop shows through it — the same
// dest-out punch the rounded corners use, with the shape's own coverage
// mask as the eraser. It is what makes the hole a hole to look at; the
// input region is what makes it a hole to click through.
//
// The eraser is the rasterised mask rather than the path it came from, so
// the edge that is painted is the same edge the compositor was given: a
// window can never look like it covers a pixel it does not take clicks on.
func (w *Window) paintWindowShape(ctx *paintengine2d.Context, dirty *paintengine2d.Damage) {
	r := w.shapeRaster()
	if r == nil || ctx == nil {
		return
	}
	if len(r.Cut) == 0 {
		// The silhouette covers its box solidly: there is nothing to
		// erase, and a shaped window that happens to be a rectangle costs
		// exactly what an unshaped one costs.
		return
	}
	img := w.shapeEraser(r)
	if img == nil {
		return
	}
	win := w.windowBox()
	paint := paintengine2d.Paint{
		Color:  paintengine2d.White,
		Blend:  paintengine2d.BlendDestOut,
		Filter: paintengine2d.FilterNearest,
	}
	// Only the boxes the silhouette actually cuts: the space outside it,
	// its holes and its antialiased edges. Blitting the whole window would
	// spend most of its time erasing nothing — a few milliseconds a frame
	// on a large window — and the compositor was told about these very
	// boxes, so nothing else can need erasing.
	for _, c := range r.Cut {
		src := paintengine2d.XYWH(float32(c.X), float32(c.Y), float32(c.W), float32(c.H))
		dst := src.Translate(paintengine2d.Pt(win.Min.X, win.Min.Y))
		if dirty != nil && !dirty.Empty() && !dirty.Overlaps(dst) {
			continue
		}
		ctx.DrawImageRectPaint(img, src, dst, paint)
	}
}

// shapeEraser is the inverse of the shape's coverage as an 8-bit image,
// cached beside the shadow: 255 where the window is not, 0 where it is,
// which dest-out turns into "erase exactly the coverage this pixel is
// missing". It is rebuilt only when the rasterisation changes, so on a
// resize and not on a frame.
func (w *Window) shapeEraser(r *platform.ShapeRaster) *paintengine2d.Image {
	if w.eraser != nil && w.eraserFor == r {
		return w.eraser
	}
	img := paintengine2d.NewImageA8(r.W, r.H)
	stride := img.RowStride()
	for y := 0; y < r.H; y++ {
		row := img.Pix[y*stride:]
		src := r.Mask[y*r.W:]
		for x := 0; x < r.W; x++ {
			row[x] = 255 - src[x]
		}
	}
	w.eraser, w.eraserFor = img, r
	return img
}

// opaqueRegion is the fully covered part of the silhouette less every box a
// widget declared see-through: what the compositor may really skip drawing
// behind. With nothing declared it is the shape's own opaque rectangles,
// untouched.
//
// The subtraction goes through the coverage mask rather than through region
// arithmetic on the rectangle lists: the mask is already there, one pass
// over it is exact, and it happens on a layout rather than on a frame.
func (w *Window) opaqueRegion(r *platform.ShapeRaster) []platform.FrameRect {
	holes := w.transparentBoxes()
	if len(holes) == 0 {
		return r.Opaque
	}
	win := w.windowBox()
	solid := make([]uint8, r.W*r.H)
	for i, c := range r.Mask {
		if c == 255 {
			solid[i] = 255
		}
	}
	for _, h := range holes {
		// Widget boxes are surface coordinates; the mask is the visible
		// window's. Round outwards, so a box that covers half a pixel
		// never leaves that pixel claimed solid.
		x0, y0, x1, y1 := h.Translate(paintengine2d.Pt(-win.Min.X, -win.Min.Y)).IntBounds()
		for y := max(y0, 0); y < min(y1+1, r.H); y++ {
			row := solid[y*r.W:]
			for x := max(x0, 0); x < min(x1+1, r.W); x++ {
				row[x] = 0
			}
		}
	}
	return platform.MaskRects(solid, r.W, r.H, r.W, 255)
}

// transparentBoxes is every box a widget declared see-through, in surface
// device pixels. The tree is walked only for a window that is shaped or
// glassy and claims its interior solid — an ordinary window never asks.
func (w *Window) transparentBoxes() []paintengine2d.Rect {
	var out []paintengine2d.Rect
	if w.root != nil {
		out = widget.TransparentRects(w.root, out)
	}
	if w.caption != nil {
		out = widget.TransparentRects(w.caption, out)
	}
	return out
}
