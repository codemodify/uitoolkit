package widget

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A child is exposed while any of it is inside every ancestor's box — the
// way a scroll view's viewport clips its content — and while all of them
// are visible.
func TestExposed(t *testing.T) {
	view, content, child := &namedBox{}, &namedBox{}, &namedBox{}
	for _, c := range []*namedBox{view, content, child} {
		c.Init(c)
	}
	view.Add(content)
	content.Add(child)
	view.SetBounds(paintengine2d.XYWH(0, 0, 200, 100))
	content.SetBounds(paintengine2d.XYWH(0, 0, 200, 1000))
	child.SetBounds(paintengine2d.XYWH(10, 500, 100, 20))
	if Exposed(child) {
		t.Fatal("a child below the viewport is exposed")
	}
	content.SetBounds(paintengine2d.XYWH(0, -450, 200, 1000)) // scrolled
	if !Exposed(child) {
		t.Fatal("a child scrolled into the viewport is not exposed")
	}
	view.SetVisible(false)
	if Exposed(child) {
		t.Fatal("a child of a hidden view is exposed")
	}
}
