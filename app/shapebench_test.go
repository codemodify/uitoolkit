package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// What a shape costs a window, against the same window without one. The
// numbers that matter are the second and third: a shaped window must not
// pay for its silhouette on every frame, only when it changes.

// benchWindow is a 500x500 window, shaped or not.
func benchWindow(b *testing.B, shaped bool) *shapeRig {
	b.Helper()
	r := &shapeRig{a: New(Options{Headless: true})}
	w, err := r.a.NewWindow(platform.WindowOptions{
		Title: "Bench", Width: 875, Height: 875, Headless: true,
		Decorations: platform.DecorationsNone,
	})
	if err != nil {
		b.Fatal(err)
	}
	r.w, r.o = w, w.Surface().(*platform.Offscreen)
	if shaped {
		w.SetShapeFunc(ring)
	}
	r.a.PumpOnce()
	w.frame()
	return r
}

// BenchmarkFrameOpaque is the baseline: a window with no shape at all.
func BenchmarkFrameOpaque(b *testing.B) { benchFrame(b, false) }

// BenchmarkFrameShaped is the same window with a ring silhouette, hole and
// all, repainted in full every frame.
func BenchmarkFrameShaped(b *testing.B) { benchFrame(b, true) }

func benchFrame(b *testing.B, shaped bool) {
	r := benchWindow(b, shaped)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.w.fullInvalidate()
		r.w.frame()
	}
}

// BenchmarkFramePartialOpaque and ...Shaped are the everyday case: one
// small box of the window changed (a hover, a caret), not the whole thing.
// A shaped window's silhouette is cut only where the damage reaches it.
func BenchmarkFramePartialOpaque(b *testing.B) { benchPartial(b, false) }
func BenchmarkFramePartialShaped(b *testing.B) { benchPartial(b, true) }

func benchPartial(b *testing.B, shaped bool) {
	r := benchWindow(b, shaped)
	box := paintengine2d.XYWH(40, 40, 120, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.w.dirty.Reset()
		r.w.full = false
		r.w.dirty.Add(box)
		r.w.frame()
	}
}

// BenchmarkShapeRasterCached is what asking a window for its silhouette
// costs once it has one: a cache lookup, not a rasterisation.
func BenchmarkShapeRasterCached(b *testing.B) {
	r := benchWindow(b, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if r.w.shapeRaster() == nil {
			b.Fatal("no raster")
		}
	}
}

// BenchmarkShapeRasterCold is the cost a resize pays: the silhouette drawn
// and turned into rectangles from scratch.
func BenchmarkShapeRasterCold(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s := ring(paintengine2d.Pt(875, 875), 1.75)
		if s.Raster(875, 875) == nil {
			b.Fatal("no raster")
		}
	}
}

// BenchmarkShapeBandLarge is the resize band a large resizable shaped
// window works out on a resize, beside its silhouette.
func BenchmarkShapeBandLarge(b *testing.B) {
	r := ring(paintengine2d.Pt(1600, 1200), 1).Raster(1600, 1200)
	m := platform.FrameInsets{Top: 20, Right: 20, Bottom: 20, Left: 20}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if buildShapeBand(r, m, 10, 4, 16) == nil {
			b.Fatal("no band")
		}
	}
}
