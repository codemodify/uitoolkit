package widget

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

type painted struct {
	Base
	col paintengine2d.Color
}

func newPainted(c paintengine2d.Color) *painted {
	p := &painted{col: c}
	p.Init(p)
	return p
}

func (p *painted) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(p.LocalBounds(), paintengine2d.Fill(p.col))
}

func record(root Component, cache *SceneCache, w, h int) *paintengine2d.Scene {
	rec := paintengine2d.NewRecorder(w, h)
	ctx := paintengine2d.NewContextDevice(rec)
	RecordTree(root, rec, ctx, nil, cache, false)
	cache.EndFrame()
	return rec.Finish()
}

// replay rasterizes a scene so recordings can be compared as pixels.
func replay(s *paintengine2d.Scene, w, h int) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	paintengine2d.DrawScene(s, paintengine2d.NewCPUDevice(img))
	return img
}

func samePixels(t *testing.T, a, b *paintengine2d.Image) bool {
	t.Helper()
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			if a.NRGBAAt(x, y) != b.NRGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// A cached group must not be replayed at the position it was recorded at
// once the widget has moved.
func TestReuseFollowsMove(t *testing.T) {
	root := newPainted(paintengine2d.RGB(10, 10, 10))
	child := newPainted(paintengine2d.RGB(200, 60, 60))
	root.Add(child)
	root.Arrange(paintengine2d.XYWH(0, 0, 100, 100))
	child.Arrange(paintengine2d.XYWH(0, 0, 20, 20))

	cache := NewSceneCache()
	record(root, cache, 100, 100)

	child.Arrange(paintengine2d.XYWH(50, 50, 20, 20))
	cache.Invalidate(root.ID()) // only the parent is dirty
	got := replay(record(root, cache, 100, 100), 100, 100)

	fresh := NewSceneCache()
	want := replay(record(root, fresh, 100, 100), 100, 100)
	if !samePixels(t, got, want) {
		t.Fatal("reused group did not follow the widget move")
	}
}

// A group whose clip did not move with it cannot be reused by translation.
func TestReuseRejectedWhenClipStaysPut(t *testing.T) {
	root := newPainted(paintengine2d.RGB(10, 10, 10))
	clipper := newPainted(paintengine2d.RGB(20, 20, 20))
	child := newPainted(paintengine2d.RGB(200, 60, 60))
	root.Add(clipper)
	clipper.Add(child)
	root.Arrange(paintengine2d.XYWH(0, 0, 100, 100))
	clipper.Arrange(paintengine2d.XYWH(0, 0, 100, 40))
	child.Arrange(paintengine2d.XYWH(0, 0, 100, 30))

	cache := NewSceneCache()
	record(root, cache, 100, 100)
	// Slide the child under a clip that stays where it is.
	child.Arrange(paintengine2d.XYWH(0, 20, 100, 30))
	cache.Invalidate(clipper.ID())
	got := replay(record(root, cache, 100, 100), 100, 100)

	fresh := NewSceneCache()
	want := replay(record(root, fresh, 100, 100), 100, 100)
	if !samePixels(t, got, want) {
		t.Fatal("a slide under a fixed clip must re-record, not translate")
	}
}

// Groups for widgets that were not visited must not linger.
func TestEndFrameDropsUnvisited(t *testing.T) {
	root := newPainted(paintengine2d.RGB(10, 10, 10))
	a := newPainted(paintengine2d.RGB(1, 2, 3))
	b := newPainted(paintengine2d.RGB(4, 5, 6))
	root.Add(a)
	root.Add(b)
	root.Arrange(paintengine2d.XYWH(0, 0, 100, 100))
	a.Arrange(paintengine2d.XYWH(0, 0, 10, 10))
	b.Arrange(paintengine2d.XYWH(0, 20, 10, 10))

	cache := NewSceneCache()
	record(root, cache, 100, 100)
	if cache.Len() != 3 {
		t.Fatalf("expected 3 groups, got %d", cache.Len())
	}
	root.Remove(b)
	record(root, cache, 100, 100)
	if cache.Len() != 2 {
		t.Fatalf("removed widget kept a group: %d", cache.Len())
	}
}

// A hidden child's stale dirty flag must not make its parent re-record on
// every frame forever.
func TestInvisibleChildIsNotDirty(t *testing.T) {
	root := newPainted(paintengine2d.RGB(10, 10, 10))
	child := newPainted(paintengine2d.RGB(1, 2, 3))
	root.Add(child)
	root.Arrange(paintengine2d.XYWH(0, 0, 100, 100))
	child.Arrange(paintengine2d.XYWH(0, 0, 10, 10))

	cache := NewSceneCache()
	record(root, cache, 100, 100)
	child.SetVisible(false)
	cache.Invalidate(child.ID())
	record(root, cache, 100, 100)
	if subtreeDirty(root, cache) {
		t.Fatal("an invisible child kept its parent dirty")
	}
	s := record(root, cache, 100, 100)
	if s.Reused < 1 {
		t.Fatalf("parent should be reusable, reused=%d", s.Reused)
	}
}
