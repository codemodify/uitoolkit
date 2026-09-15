package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func TestSegmentedChooses(t *testing.T) {
	got := -1
	s := NewSegmented([]string{"List", "Cards", "Columns"}, 0, func(i int) { got = i })
	s.SetHost(&fakeWindow{look: style.DarkLook()})
	sz := s.Measure(layout.Unbounded())
	s.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	r := s.segRect(2)
	p := paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	s.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	s.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	if got != 2 || s.Selected != 2 {
		t.Fatalf("click on the third segment chose %d", got)
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	if s.Selected != 1 {
		t.Fatalf("left -> %d, want 1", s.Selected)
	}
	// Every look paints it inside its bounds.
	for _, p := range style.ListBuiltinThemes() {
		s.SetHost(&fakeWindow{look: p.Look()})
		sz := s.Measure(layout.Unbounded())
		s.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
		img := paintengine2d.NewImage(int(sz.X)+2, int(sz.Y)+2)
		s.Paint(paintengine2d.NewContext(img))
	}
}
