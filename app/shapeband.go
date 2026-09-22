package app

import (
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// The resize band of a shaped window.
//
// A silhouette's input region is exactly the silhouette, so the band a
// frame keeps in its shadow's margin — where a press resizes the window —
// used to go with it, and a shaped window could not be resized by hand at
// all. The band is now kept along the silhouette's *outer* edge: a few
// pixels outside it wherever the surface has room (the margin, or the space
// the silhouette leaves in its box), and a thinner strip just inside it, as
// a frame without a shadow keeps inside its border. Holes are not edges: a
// ring is resized from its rim, never from the hole in its middle.
//
// Which edge a press resizes follows from where it is: a row's left or
// right end, a column's top or bottom end — both at once on a diagonal,
// which is a corner — and a press near the ends of a straight edge takes
// the corner too, as a rectangular frame's does. A window that may not be
// resized (platform.SizingFixed) keeps no band.

// shapeBand is the band worked out for one rasterisation, in surface
// device pixels (the visible window's pixels moved by the margin).
type shapeBand struct {
	raster *platform.ShapeRaster
	margin platform.FrameInsets
	out    int
	w, h   int
	// edges holds each surface pixel's edges (platform.Edges bits), zero
	// outside the band.
	edges []uint8
	// rects is the band as rectangles, for the input region.
	rects []platform.FrameRect
}

// buildShapeBand works out the band around r inside a surface with margin
// m: out pixels outside the silhouette, in inside it, corner along each
// straight edge's ends.
func buildShapeBand(r *platform.ShapeRaster, m platform.FrameInsets, out, in, corner int) *shapeBand {
	gw, gh := r.W+m.Width(), r.H+m.Height()
	if gw < 1 || gh < 1 || gw*gh > 8192*8192 {
		return nil
	}
	b := &shapeBand{raster: r, margin: m, out: out, w: gw, h: gh, edges: make([]uint8, gw*gh)}
	mark := func(x0, x1, y0, y1 int, e platform.Edges) {
		x0, x1 = max(x0, 0), min(x1, gw)
		y0, y1 = max(y0, 0), min(y1, gh)
		for y := y0; y < y1; y++ {
			row := b.edges[y*gw:]
			for x := x0; x < x1; x++ {
				row[x] |= uint8(e)
			}
		}
	}
	// Rows: each one's outermost covered pixels are the left and right
	// edges there; columns likewise for the top and bottom. An end counts
	// only where it faces outwards — within a quarter of the box of the
	// box's own side — so the step beside BeOS's tab, or the notch of a
	// concave skin, is not an edge of the window.
	bb := r.Bounds
	reachX := max(out+in, bb.W/4)
	reachY := max(out+in, bb.H/4)
	for y := 0; y < r.H; y++ {
		row := r.Mask[y*r.W : (y+1)*r.W]
		lo, hi := -1, -1
		for x, c := range row {
			if c >= 128 {
				if lo < 0 {
					lo = x
				}
				hi = x
			}
		}
		if lo < 0 {
			continue
		}
		gy := y + m.Top
		if lo < bb.X+reachX {
			mark(lo+m.Left-out, lo+m.Left+in, gy, gy+1, platform.EdgeLeft)
		}
		if hi+1 > bb.X+bb.W-reachX {
			mark(hi+1+m.Left-in, hi+1+m.Left+out, gy, gy+1, platform.EdgeRight)
		}
	}
	for x := 0; x < r.W; x++ {
		lo, hi := -1, -1
		for y := 0; y < r.H; y++ {
			if r.Mask[y*r.W+x] >= 128 {
				if lo < 0 {
					lo = y
				}
				hi = y
			}
		}
		if lo < 0 {
			continue
		}
		gx := x + m.Left
		if lo < bb.Y+reachY {
			mark(gx, gx+1, lo+m.Top-out, lo+m.Top+in, platform.EdgeTop)
		}
		if hi+1 > bb.Y+bb.H-reachY {
			mark(gx, gx+1, hi+1+m.Top-in, hi+1+m.Top+out, platform.EdgeBottom)
		}
	}
	// Near the ends of a straight edge a press takes the corner, as a
	// rectangular frame's does.
	x0, x1 := bb.X+m.Left, bb.X+bb.W+m.Left
	y0, y1 := bb.Y+m.Top, bb.Y+bb.H+m.Top
	for y := 0; y < gh; y++ {
		row := b.edges[y*gw:]
		for x := 0; x < gw; x++ {
			e := platform.Edges(row[x])
			switch e {
			case platform.EdgeTop, platform.EdgeBottom:
				if x < x0+corner {
					e |= platform.EdgeLeft
				} else if x >= x1-corner {
					e |= platform.EdgeRight
				}
			case platform.EdgeLeft, platform.EdgeRight:
				if y < y0+corner {
					e |= platform.EdgeTop
				} else if y >= y1-corner {
					e |= platform.EdgeBottom
				}
			}
			if !e.Valid() && e != 0 {
				// Opposite sides at once: a sliver thinner than the band
				// itself. The first one found wins.
				if e&platform.EdgeLeft != 0 && e&platform.EdgeRight != 0 {
					e &^= platform.EdgeRight
				}
				if e&platform.EdgeTop != 0 && e&platform.EdgeBottom != 0 {
					e &^= platform.EdgeBottom
				}
			}
			row[x] = uint8(e)
		}
	}
	on := make([]uint8, gw*gh)
	for i, e := range b.edges {
		if e != 0 {
			on[i] = 255
		}
	}
	b.rects = platform.MaskRects(on, gw, gh, gw, 255)
	return b
}

// at is the edges a press at surface point x, y resizes, zero outside the
// band.
func (b *shapeBand) at(x, y int) platform.Edges {
	if b == nil || x < 0 || y < 0 || x >= b.w || y >= b.h {
		return 0
	}
	return platform.Edges(b.edges[y*b.w+x])
}

// shapeBandFor is the band around the window's silhouette r, or nil where
// the window keeps none: it may not be resized, or the silhouette is not in
// effect (shapeRaster is nil then anyway). It is built once per
// rasterisation and margin, not per frame.
func (w *Window) shapeBandFor(r *platform.ShapeRaster, m platform.FrameInsets) *shapeBand {
	if r == nil || r.Empty() || !w.Resizable() {
		return nil
	}
	if b := w.band; b != nil && b.raster == r && b.margin == m {
		return b
	}
	band, corner := w.resizeBand()
	in := int(math.Round(float64(style.Dip(w.look, insideBandDip))))
	w.band = buildShapeBand(r, m, int(band), in, int(corner))
	return w.band
}

// shapeBandEdges is the resize edges under surface point p of a shaped
// window, zero anywhere else or in any state that resizes nothing.
func (w *Window) shapeBandEdges(p paintengine2d.Point) platform.Edges {
	st := w.state
	if st.Maximized || st.Fullscreen {
		return 0
	}
	r := w.shapeRaster()
	if r == nil {
		return 0
	}
	e := w.shapeBandFor(r, w.sysFrame.Margin).at(int(math.Floor(float64(p.X))), int(math.Floor(float64(p.Y))))
	e &^= st.Tiled | st.Constrained
	if !e.Valid() {
		return 0
	}
	return e
}
