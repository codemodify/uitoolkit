package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

type host struct {
	focus widget.Component
}

func (h *host) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *host) RequestFocus(c widget.Component)                 { h.focus = c }
func (h *host) Focus() widget.Component                         { return h.focus }
func (h *host) Scale() float32                                  { return 1 }
func (h *host) Look() style.LookAndFeel                         { return style.DarkLook() }

func TestCheckboxToggle(t *testing.T) {
	h := &host{}
	n := 0
	c := NewCheckbox("X", false, func(v bool) {
		if v {
			n++
		}
	})
	c.SetHost(h)
	c.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	c.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !c.Checked || n != 1 {
		t.Fatalf("checked=%v n=%d", c.Checked, n)
	}
}

func TestSliderDrag(t *testing.T) {
	s := NewSlider(0, 100, 0, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(100, 14)})
	if s.Value < 40 || s.Value > 60 {
		t.Fatalf("mid value %v", s.Value)
	}
}

func TestTextFieldEdit(t *testing.T) {
	tf := NewTextField("Hi", "", nil)
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 160, 32))
	tf.TextInput('!')
	if tf.Text != "Hi!" {
		t.Fatalf("%q", tf.Text)
	}
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if tf.Text != "Hi" {
		t.Fatalf("backspace %q", tf.Text)
	}
}

func TestButtonClick(t *testing.T) {
	n := 0
	b := NewButton("Go", func() { n++ })
	b.SetHost(&host{})
	b.Arrange(paintengine2d.XYWH(0, 0, 80, 32))
	b.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(10, 10)})
	b.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(10, 10)})
	if n != 1 {
		t.Fatalf("clicks %d", n)
	}
}

func TestColumnLayoutTwoKids(t *testing.T) {
	col := NewColumn(NewLabel("A"), NewLabel("B")).WithGap(6)
	sz := col.Measure(layout.Loose(200, 200))
	if sz.Y < 20 {
		t.Fatalf("column %v", sz)
	}
	col.Arrange(paintengine2d.XYWH(0, 0, 200, sz.Y))
	chs := col.Children()
	if chs[0].Bounds().Min.Y >= chs[1].Bounds().Min.Y {
		t.Fatalf("not stacked: %+v %+v", chs[0].Bounds(), chs[1].Bounds())
	}
}

func TestListViewVirtualRange(t *testing.T) {
	lv := NewListView(1000, func(i int) string { return "x" }, nil)
	lv.RowHeight = 20
	lv.Arrange(paintengine2d.XYWH(0, 0, 100, 100))
	lo, hi := lv.visibleRange()
	if lo != 0 || hi < 5 || hi > 8 {
		t.Fatalf("visible %d %d", lo, hi)
	}
	lv.OffsetY = 400
	lv.clamp()
	lo, hi = lv.visibleRange()
	if lo < 18 || lo > 22 {
		t.Fatalf("scrolled lo=%d", lo)
	}
}
