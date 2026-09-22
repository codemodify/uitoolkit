package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The cold cost of a silhouette — what a resize of a shaped window pays —
// at the size of a large window: 1600x1200 device pixels, a rounded panel
// with a round hole.
func benchRing(w, h float32) *Shape {
	p := paintengine2d.NewPath()
	p.AddRoundRect(paintengine2d.XYWH(0, 0, w, h), 48, 48)
	p.AddCircle(paintengine2d.Pt(w/2, h/2), min(w, h)*0.3)
	return NewShapeEvenOdd(p)
}

// BenchmarkShapeMaskLarge is the rasterisation alone: the path to coverage.
func BenchmarkShapeMaskLarge(b *testing.B) {
	s := benchRing(1600, 1200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rasterPathMask(s.path, s.rule, 1600, 1200)
	}
}

// BenchmarkShapeRasterLarge is the whole cold path: coverage, then the
// silhouette's, the opaque, the cut and the clear rectangles.
func BenchmarkShapeRasterLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if benchRing(1600, 1200).Raster(1600, 1200) == nil {
			b.Fatal("no raster")
		}
	}
}
