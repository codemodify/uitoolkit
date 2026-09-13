package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// recordRows records rows lo..hi at scroll offset off into a fresh scene.
func recordRows(cache *rowSceneCache, off float32, count int) *paintengine2d.Scene {
	rec := paintengine2d.NewRecorder(80, 80)
	ctx := paintengine2d.NewContextDevice(rec)
	o := rowOrigin(ctx)
	cache.ready(o.X, o.Y, 80, 20, 1)
	lo := int(off / 20)
	hi := lo + count
	recordScrollingRows(rec, ctx, cache, 1, paintengine2d.XYWH(0, 0, 80, 80), 80, 20, off, 0, lo, hi,
		func(i int) uint64 { return uint64(i + 1) },
		func(i int) uint64 {
			sig := newRowSig(false, false, 0)
			sig.str("row")
			return sig.sum()
		},
		func(i int) {
			ctx.DrawRect(paintengine2d.XYWH(0, 0, 80, 20), paintengine2d.Fill(paintengine2d.White))
		},
	)
	return rec.Finish()
}

func TestRecordScrollingRowsReuses(t *testing.T) {
	cache := &rowSceneCache{}
	if s := recordRows(cache, 0, 4); s.Reused != 0 {
		t.Fatalf("first record reused=%d", s.Reused)
	}
	if s := recordRows(cache, 0, 4); s.Reused < 4 {
		t.Fatalf("same offset should reuse rows, reused=%d", s.Reused)
	}
}

// The point of recording rows in row-local space: scrolling keeps every row
// that is still on screen instead of re-recording the whole viewport.
func TestRecordScrollingRowsSurvivesScroll(t *testing.T) {
	cache := &rowSceneCache{}
	recordRows(cache, 0, 4) // rows 0..3
	s := recordRows(cache, 40, 4)
	// rows 2..5: two carried over from the previous frame.
	if s.Reused < 2 {
		t.Fatalf("scrolling should reuse the rows that stayed visible, reused=%d", s.Reused)
	}
}

// A widget that moved must re-record: its rows bake the clip they were
// recorded under.
func TestRowCacheDropsOnMove(t *testing.T) {
	cache := &rowSceneCache{}
	recordRows(cache, 0, 4)
	cache.ready(17, 23, 80, 20, 1)
	if len(cache.nodes) != 0 {
		t.Fatalf("a moved widget kept %d rows", len(cache.nodes))
	}
}

// A theme change must re-record every row.
func TestRowCacheDropsOnLookChange(t *testing.T) {
	cache := &rowSceneCache{}
	recordRows(cache, 0, 4)
	cache.ready(0, 0, 80, 20, 2)
	if len(cache.nodes) != 0 {
		t.Fatalf("a theme change kept %d rows", len(cache.nodes))
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

// rowSig must agree with visualSig so the two can be mixed during a
// transition, and must still separate cell boundaries.
func TestRowSigMatchesVisualSig(t *testing.T) {
	s := newRowSig(true, false, 7)
	s.str("a")
	s.str("bc")
	if got, want := s.sum(), visualSig(true, false, 7, "a", "bc"); got != want {
		t.Fatalf("rowSig %d != visualSig %d", got, want)
	}
	x := newRowSig(false, false, 0)
	x.str("ab")
	x.str("c")
	y := newRowSig(false, false, 0)
	y.str("a")
	y.str("bc")
	if x.sum() == y.sum() {
		t.Fatal("cell boundaries must change the signature")
	}
}
