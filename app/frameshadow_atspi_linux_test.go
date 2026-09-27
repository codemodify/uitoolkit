//go:build linux

package app

import "testing"

// The accessible half of TestIMECursorAndAccessibleExtentsFollowTheMargin.
//
// It lives in a Linux-only file because atspiObj is the AT-SPI adapter,
// which only exists here — and a test file that does not cross-compile
// makes `GOOS=windows go vet ./...` fail, which is how a Windows port
// finds out too late that something Linux-only leaked into shared code.
func TestAccessibleExtentsFollowTheMargin(t *testing.T) {
	r := shadowRig(t, bigSurLook(t))
	// Accessible boxes are relative to the window, so the margin is out.
	tree := r.w.AccessibleTree()
	if tree == nil || len(tree.Children) == 0 {
		t.Fatal("no accessible tree")
	}
	node := tree.Children[0]
	obj := &atspiObj{win: r.w, node: node}
	e := obj.extents()
	if e.X < 0 || e.Y < 0 {
		t.Fatalf("accessible extents %v are outside the window", e)
	}
	if want := int32(node.Bounds.Min.X - r.w.WindowRect().Min.X); e.X != want {
		t.Fatalf("accessible x %d, want %d (bounds %v, window %v)", e.X, want, node.Bounds, r.w.WindowRect())
	}
}
