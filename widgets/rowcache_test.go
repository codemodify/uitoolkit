package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestRecordScrollingRowsReuses(t *testing.T) {
	cache := &rowSceneCache{}
	cache.ready(0, 0, 80, 20, 0, 1)

	paint := func(rec *paintengine2d.Recorder, off float32) *paintengine2d.Scene {
		ctx := paintengine2d.NewContextDevice(rec)
		_ = ctx
		lo := int(off / 20)
		hi := lo + 4
		recordScrollingRows(rec, cache, 1, off, 0, lo, hi,
			func(i int) uint64 { return uint64(i + 1) },
			func(i int) uint64 { return visualSig(false, false, 0, "row") },
			func(i int) {
				ctx.DrawRect(paintengine2d.XYWH(0, float32(i)*20-off, 80, 20), paintengine2d.Fill(paintengine2d.White))
			},
		)
		return rec.Finish()
	}

	s0 := paint(paintengine2d.NewRecorder(80, 80), 0)
	if s0.Reused != 0 {
		t.Fatalf("first record reused=%d", s0.Reused)
	}
	s1 := paint(paintengine2d.NewRecorder(80, 80), 0)
	if s1.Reused < 4 {
		t.Fatalf("same offset should reuse rows, reused=%d", s1.Reused)
	}
	cache.ready(0, 0, 80, 20, 40, 1)
	s2 := paint(paintengine2d.NewRecorder(80, 80), 40)
	if s2.Reused != 0 {
		t.Fatalf("new offset must not attach stale viewport rows, reused=%d", s2.Reused)
	}
}

func TestVisualSigDiffers(t *testing.T) {
	a := visualSig(false, false, 0, "hello")
	b := visualSig(true, false, 0, "hello")
	c := visualSig(false, false, 0, "hallo")
	if a == b || a == c {
		t.Fatalf("signatures collided a=%d b=%d c=%d", a, b, c)
	}
}
