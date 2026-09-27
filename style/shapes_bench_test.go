package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The focus ring is the most-drawn dotted shape in the toolkit: every
// engine's DrawFocusRing ends in DottedRect, and a hovered list row
// records one per frame. Both halves of the cost are measured — the
// recording (what a scene frame pays, in draw ops and the bytes behind
// them) and the rasterizing (what the CPU device pays).
func benchDottedRing() paintengine2d.Rect { return paintengine2d.XYWH(2, 2, 260, 24) }

func BenchmarkDottedRectRecord(b *testing.B) {
	box := benchDottedRing()
	b.ReportAllocs()
	b.ResetTimer()
	ops := 0
	for i := 0; i < b.N; i++ {
		rec := paintengine2d.NewRecorder(300, 40)
		DottedRect(paintengine2d.NewContextDevice(rec), box, paintengine2d.Black)
		ops += rec.Finish().Nodes
	}
	b.ReportMetric(float64(ops)/float64(b.N), "ops/op")
}

func BenchmarkDottedRectRaster(b *testing.B) {
	box := benchDottedRing()
	img := paintengine2d.NewImage(300, 40)
	ctx := paintengine2d.NewContext(img)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DottedRect(ctx, box, paintengine2d.Black)
	}
}

func BenchmarkBumpsRecord(b *testing.B) {
	box := paintengine2d.XYWH(0, 0, 320, 200)
	b.ReportAllocs()
	b.ResetTimer()
	ops := 0
	for i := 0; i < b.N; i++ {
		rec := paintengine2d.NewRecorder(320, 200)
		Bumps(paintengine2d.NewContextDevice(rec), box, paintengine2d.White, paintengine2d.Black, 4)
		ops += rec.Finish().Nodes
	}
	b.ReportMetric(float64(ops)/float64(b.N), "ops/op")
}

func BenchmarkBumpsRaster(b *testing.B) {
	box := paintengine2d.XYWH(0, 0, 320, 200)
	img := paintengine2d.NewImage(320, 200)
	ctx := paintengine2d.NewContext(img)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Bumps(ctx, box, paintengine2d.White, paintengine2d.Black, 4)
	}
}
