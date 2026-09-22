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
	// runs are the band's pixels a row at a time, each run one set of
	// edges (platform.Edges bits); row y's runs are runs[rows[y]:rows[y+1]]
	// and a pixel in none is outside the band. A byte a surface pixel was
	// 1.4 MB for each of Lantern's windows at 1.75x, for a band a few
	// pixels wide.
	runs []bandRun
	rows []int32
	// rects is the band as rectangles, for the input region.
	rects []platform.FrameRect
}

// bandRun is pixels x0 to x1 (exclusive) of a row with the same edges.
type bandRun struct {
	x0, x1 int32
	e      uint8
}

// buildShapeBand works out the band around r inside a surface with margin
// m: out pixels outside the silhouette, in inside it, corner along each
// straight edge's ends.
func buildShapeBand(r *platform.ShapeRaster, m platform.FrameInsets, out, in, corner int) *shapeBand {
	gw, gh := r.W+m.Width(), r.H+m.Height()
	if gw < 1 || gh < 1 || gw*gh > 8192*8192 {
		return nil
	}
	b := &shapeBand{raster: r, margin: m, out: out, w: gw, h: gh}
	edges := make([]uint8, gw*gh)
	bb := r.Bounds
	// Near the ends of a straight edge a press takes the corner, as a
	// rectangular frame's does: a side's pixels within corner of the box's
	// ends take the neighbouring side too.
	cx0, cx1 := bb.X+m.Left+corner, bb.X+bb.W+m.Left-corner
	cy0, cy1 := bb.Y+m.Top+corner, bb.Y+bb.H+m.Top-corner
	mark := func(x0, x1, y0, y1 int, e platform.Edges) {
		x0, x1 = max(x0, 0), min(x1, gw)
		y0, y1 = max(y0, 0), min(y1, gh)
		for y := y0; y < y1; y++ {
			row := edges[y*gw:]
			ey := e
			if e&(platform.EdgeLeft|platform.EdgeRight) != 0 {
				if y < cy0 {
					ey |= platform.EdgeTop
				} else if y >= cy1 {
					ey |= platform.EdgeBottom
				}
			}
			for x := x0; x < x1; x++ {
				ex := ey
				if e&(platform.EdgeTop|platform.EdgeBottom) != 0 {
					if x < cx0 {
						ex |= platform.EdgeLeft
					} else if x >= cx1 {
						ex |= platform.EdgeRight
					}
				}
				row[x] |= uint8(ex)
			}
		}
	}
	// The silhouette's rectangles come in scanline order, a row group at a
	// time (every rectangle of a group starts on the same row and is as
	// tall), so a group's first and last rectangles are its rows' ends.
	//
	// An end counts only where it faces outwards — within a quarter of the
	// box of the box's own side — so the step beside BeOS's tab, or the
	// notch of a concave skin, is not an edge of the window.
	reachX := max(out+in, bb.W/4)
	reachY := max(out+in, bb.H/4)
	rects := r.Rects
	for i := 0; i < len(rects); {
		g := rects[i]
		lo, hi := g.X, g.X+g.W
		j := i + 1
		for ; j < len(rects) && rects[j].Y == g.Y; j++ {
			lo, hi = min(lo, rects[j].X), max(hi, rects[j].X+rects[j].W)
		}
		y0, y1 := g.Y+m.Top, g.Y+g.H+m.Top
		if lo < bb.X+reachX {
			mark(lo+m.Left-out, lo+m.Left+in, y0, y1, platform.EdgeLeft)
		}
		if hi > bb.X+bb.W-reachX {
			mark(hi+m.Left-in, hi+m.Left+out, y0, y1, platform.EdgeRight)
		}
		i = j
	}
	// Each column's top: the first rectangle over it, top down, which for
	// anything but a deep notch is found within a few groups; its bottom
	// likewise, bottom up.
	edgeCols := func(from, to, step int, top bool) {
		seen := make([]bool, r.W)
		left := r.W
		for i := from; i != to && left > 0; i += step {
			g := rects[i]
			for x := g.X; x < g.X+g.W; x++ {
				if seen[x] {
					continue
				}
				seen[x] = true
				left--
				gx := x + m.Left
				if top {
					if g.Y < bb.Y+reachY {
						mark(gx, gx+1, g.Y+m.Top-out, g.Y+m.Top+in, platform.EdgeTop)
					}
				} else if end := g.Y + g.H; end > bb.Y+bb.H-reachY {
					mark(gx, gx+1, end+m.Top-in, end+m.Top+out, platform.EdgeBottom)
				}
			}
		}
	}
	if len(rects) > 0 {
		edgeCols(0, len(rects), 1, true)
		edgeCols(len(rects)-1, -1, -1, false)
	}
	b.rects = platform.MaskRectsRange(edges, gw, gh, gw, 1, 255)
	b.keepRuns(edges)
	return b
}

// keepRuns keeps edges, one byte a surface pixel, as the runs of each row.
func (b *shapeBand) keepRuns(edges []uint8) {
	b.rows = make([]int32, b.h+1)
	for y := 0; y < b.h; y++ {
		b.rows[y] = int32(len(b.runs))
		row := edges[y*b.w : (y+1)*b.w]
		for x := 0; x < b.w; {
			e := row[x]
			if e == 0 {
				x++
				continue
			}
			x0 := x
			for x < b.w && row[x] == e {
				x++
			}
			b.runs = append(b.runs, bandRun{x0: int32(x0), x1: int32(x), e: e})
		}
	}
	b.rows[b.h] = int32(len(b.runs))
	// A copy the size it needs: append grew the array up to twice that.
	b.runs = append([]bandRun(nil), b.runs...)
}

// at is the edges a press at surface point x, y resizes, zero outside the
// band.
func (b *shapeBand) at(x, y int) platform.Edges {
	if b == nil || x < 0 || y < 0 || x >= b.w || y >= b.h {
		return 0
	}
	var e platform.Edges
	for _, r := range b.runs[b.rows[y]:b.rows[y+1]] {
		if int(r.x0) <= x && x < int(r.x1) {
			e = platform.Edges(r.e)
			break
		}
	}
	// Opposite sides at once is a sliver thinner than the band itself:
	// the left and the top win.
	if e&platform.EdgeLeft != 0 {
		e &^= platform.EdgeRight
	}
	if e&platform.EdgeTop != 0 {
		e &^= platform.EdgeBottom
	}
	return e
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
