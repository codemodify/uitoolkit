package widget

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

type stubHost struct {
	focus Component
	n     int
	look  style.LookAndFeel
}

func (h *stubHost) Invalidate(Component, paintengine2d.Rect) { h.n++ }
func (h *stubHost) RequestFocus(c Component)                 { h.focus = c }
func (h *stubHost) Focus() Component                         { return h.focus }
func (h *stubHost) Scale() float32                           { return 1 }
func (h *stubHost) RequestLayout()                           {}
func (h *stubHost) Look() style.LookAndFeel {
	if h.look != nil {
		return h.look
	}
	return style.DarkLook()
}

func TestHitTestFrontChildWins(t *testing.T) {
	root := &Base{}
	root.Init(root)
	a := &Base{}
	a.Init(a)
	b := &Base{}
	b.Init(b)
	root.Add(a)
	root.Add(b)
	root.Arrange(paintengine2d.XYWH(0, 0, 100, 40))
	a.Arrange(paintengine2d.XYWH(0, 0, 50, 40))
	b.Arrange(paintengine2d.XYWH(30, 0, 50, 40))
	hit := root.HitTest(paintengine2d.Pt(40, 10))
	if hit != b {
		t.Fatalf("overlap should hit later child, got %v", hit)
	}
	hit = root.HitTest(paintengine2d.Pt(10, 10))
	if hit != a {
		t.Fatalf("left hit %v", hit)
	}
	if root.HitTest(paintengine2d.Pt(200, 10)) != nil {
		t.Fatal("outside")
	}
}

func TestFocusablesOrder(t *testing.T) {
	root := &Base{}
	root.Init(root)
	a := &Base{}
	a.Init(a)
	a.SetWantsFocus(true)
	b := &Base{}
	b.Init(b)
	c := &Base{}
	c.Init(c)
	c.SetWantsFocus(true)
	root.Add(a)
	root.Add(b)
	root.Add(c)
	got := Focusables(root)
	if len(got) != 2 || got[0] != a || got[1] != c {
		t.Fatalf("%v", got)
	}
}

func TestMeasurePreferred(t *testing.T) {
	b := &Base{}
	b.Init(b)
	b.SetPreferred(40, 18)
	sz := b.Measure(layout.Loose(100, 100))
	if sz.X != 40 || sz.Y != 18 {
		t.Fatalf("%v", sz)
	}
}

func TestHiddenSkipsHit(t *testing.T) {
	root := &Base{}
	root.Init(root)
	a := &Base{}
	a.Init(a)
	root.Add(a)
	root.Arrange(paintengine2d.XYWH(0, 0, 40, 40))
	a.Arrange(paintengine2d.XYWH(0, 0, 40, 40))
	a.SetVisible(false)
	if root.HitTest(paintengine2d.Pt(10, 10)) != root {
		t.Fatal("hidden child should be ignored")
	}
}

func TestWalkSkipsInvisible(t *testing.T) {
	root := &Base{}
	root.Init(root)
	a := &Base{}
	a.Init(a)
	a.SetWantsFocus(true)
	b := &Base{}
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetVisible(false)
	root.Add(a)
	root.Add(b)
	got := Focusables(root)
	if len(got) != 1 || got[0] != a {
		t.Fatalf("%v", got)
	}
	if !Contains(root, b) || !Contains(root, a) || Contains(a, b) {
		t.Fatal("Contains should ignore visibility")
	}
}
