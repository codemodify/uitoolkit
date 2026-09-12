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
func (h *host) RequestLayout()                                  {}

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
	c.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
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

func TestPasswordFieldMasksDisplay(t *testing.T) {
	tf := NewPasswordField("password", nil)
	if !tf.Password {
		t.Fatal("Password")
	}
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 160, 32))
	tf.TextInput('s')
	tf.TextInput('e')
	tf.TextInput('c')
	if tf.Text != "sec" {
		t.Fatalf("stored %q", tf.Text)
	}
	vis, caret, _, _ := tf.visual()
	if vis != "•••" || caret != 3 {
		t.Fatalf("masked vis=%q caret=%d", vis, caret)
	}
}

func TestMonoFieldConstructors(t *testing.T) {
	tf := NewMonoTextField("/tmp/log", "path", nil)
	if !tf.Mono {
		t.Fatal("NewMonoTextField must set Mono (JetBrains Mono role)")
	}
	ta := NewMonoTextArea("fn()", "code", nil)
	if !ta.Mono {
		t.Fatal("NewMonoTextArea must set Mono (JetBrains Mono role)")
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

func TestTextFieldSelectionAndNav(t *testing.T) {
	tf := NewTextField("Ada Lovelace", "", nil)
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 220, 32))
	tf.SetSelection(0, 3)
	a, b := tf.Selection()
	if a != 0 || b != 3 {
		t.Fatalf("sel %d %d", a, b)
	}
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyDelete})
	if tf.Text != " Lovelace" {
		t.Fatalf("delete sel %q", tf.Text)
	}
	tf.SetText("one two three")
	tf.SetSelection(13, 13)
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyLeft, Mods: platform.ModCtrl})
	if tf.Caret() != 8 {
		t.Fatalf("ctrl-left caret %d", tf.Caret())
	}
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyLeft, Mods: platform.ModShift})
	a, b = tf.Selection()
	if a == b {
		t.Fatal("shift-left should extend selection")
	}
	tf.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 16)})
	tf.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(80, 16), Button: platform.ButtonLeft})
	tf.MouseRelease(widget.MouseEvent{})
	a, b = tf.Selection()
	if a == b {
		t.Fatalf("drag should select, got %d %d caret=%d", a, b, tf.Caret())
	}
}

func TestSliderKeys(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	s.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if s.Value <= 50 {
		t.Fatalf("right %v", s.Value)
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if s.Value != 0 {
		t.Fatalf("home %v", s.Value)
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if s.Value != 100 {
		t.Fatalf("end %v", s.Value)
	}
}

func TestListViewKeys(t *testing.T) {
	n := -1
	lv := NewListView(20, func(i int) string { return "x" }, func(i int) { n = i })
	lv.RowHeight = 20
	lv.SetHost(&host{})
	lv.Arrange(paintengine2d.XYWH(0, 0, 120, 80))
	lv.Selected = 0
	lv.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if lv.Selected != 1 || n != 1 {
		t.Fatalf("down sel=%d n=%d", lv.Selected, n)
	}
	lv.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if lv.Selected != 19 {
		t.Fatalf("end %d", lv.Selected)
	}
}

func TestScrollTrackHitAndPage(t *testing.T) {
	col := NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(NewLabel("row"))
	}
	sv := NewScrollView(col)
	sv.SetHost(&host{})
	sv.Arrange(paintengine2d.XYWH(0, 0, 160, 120))
	track, thumb := sv.thumb()
	if thumb.Empty() {
		t.Fatal("expected thumb")
	}
	hit := sv.HitTest(paintengine2d.Pt(track.Min.X+1, track.Min.Y+4))
	if hit != sv {
		t.Fatalf("track should hit ScrollView, got %T", hit)
	}
	below := paintengine2d.Pt(track.Min.X+1, thumb.Max.Y+8)
	if !track.Contains(below) {
		below = paintengine2d.Pt(track.Min.X+1, track.Max.Y-4)
	}
	sv.MousePress(widget.MouseEvent{Pos: below})
	if sv.OffsetY <= 0 {
		t.Fatalf("track page should scroll, offset=%v thumb=%+v track=%+v", sv.OffsetY, thumb, track)
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
