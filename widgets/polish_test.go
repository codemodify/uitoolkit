package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

func TestTextAreaNewlineAndNav(t *testing.T) {
	ta := NewTextArea("hello", "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 240, 90))
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if ta.Text != "hello\n" {
		t.Fatalf("return %q", ta.Text)
	}
	ta.TextInput('w')
	ta.TextInput('o')
	if ta.Text != "hello\nwo" {
		t.Fatalf("type %q", ta.Text)
	}
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if ta.Caret() > runeCount("hello") {
		t.Fatalf("up caret %d", ta.Caret())
	}
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if ta.Caret() != 0 {
		t.Fatalf("home %d", ta.Caret())
	}
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyEnd, Mods: platform.ModCtrl})
	if ta.Caret() != runeCount(ta.Text) {
		t.Fatalf("ctrl-end %d", ta.Caret())
	}
	ta.SetSelection(0, 5)
	if ta.SelectedText() != "hello" {
		t.Fatalf("sel %q", ta.SelectedText())
	}
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyC, Mods: platform.ModCtrl})
	if platform.ClipboardGet() != "hello" {
		t.Fatalf("copy %q", platform.ClipboardGet())
	}
}

func TestTextAreaWrapLines(t *testing.T) {
	ta := NewTextArea("one two three four five six", "", nil)
	ta.Wrap = true
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 90, 120))
	lines := ta.Lines()
	if len(lines) < 2 {
		t.Fatalf("expected wrap, lines=%d %+v", len(lines), lines)
	}
	joined := ""
	for _, ln := range lines {
		joined += ln.Text
	}
	if !strings.Contains(joined, "one") || !strings.Contains(joined, "six") {
		t.Fatalf("lost text %q", joined)
	}
	hard := NewTextArea("a\nb\n", "", nil)
	hard.SetHost(&host{})
	hard.Arrange(paintengine2d.XYWH(0, 0, 200, 80))
	hl := hard.Lines()
	if len(hl) != 3 {
		t.Fatalf("trailing newline lines=%d %+v", len(hl), hl)
	}
}

func TestTextAreaScroll(t *testing.T) {
	ta := NewTextArea("1\n2\n3\n4\n5\n6\n7\n8\n9\n10", "", nil)
	ta.MinRows = 2
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 160, 50))
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyEnd, Mods: platform.ModCtrl})
	if ta.scrollY <= 0 {
		t.Fatalf("caret at end should scroll, y=%v", ta.scrollY)
	}
	ta.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, -40)})
	if ta.scrollY < 0 {
		t.Fatal("scroll clamp")
	}
}

func TestSwitchToggle(t *testing.T) {
	n := 0
	s := NewSwitch("Dark", false, func(v bool) {
		if v {
			n++
		}
	})
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 140, 32))
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !s.On || n != 1 {
		t.Fatalf("on=%v n=%d", s.On, n)
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if s.On {
		t.Fatal("space should toggle off")
	}
}

func TestExpanderCollapse(t *testing.T) {
	child := NewLabel("secret")
	e := NewExpander("More", true, child)
	e.SetHost(&host{})
	sz := e.Measure(layout.Loose(200, 200))
	e.Arrange(paintengine2d.XYWH(0, 0, 200, sz.Y))
	if !child.Visible() {
		t.Fatal("expanded child hidden")
	}
	openH := sz.Y
	e.head.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(10, 10)})
	if e.Expanded || child.Visible() {
		t.Fatalf("collapsed exp=%v vis=%v", e.Expanded, child.Visible())
	}
	sz = e.Measure(layout.Loose(200, 200))
	if sz.Y >= openH {
		t.Fatalf("collapsed should shrink %v vs %v", sz.Y, openH)
	}
	e.head.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if !e.Expanded {
		t.Fatal("right expands")
	}
}

func TestAccordionExclusive(t *testing.T) {
	a := NewExpander("A", true, NewLabel("aa"))
	b := NewExpander("B", false, NewLabel("bb"))
	acc := NewAccordion(true, a, b)
	acc.SetHost(&host{})
	sz := acc.Measure(layout.Loose(220, 300))
	acc.Arrange(paintengine2d.XYWH(0, 0, 220, sz.Y))
	if !a.Expanded || b.Expanded {
		t.Fatal("initial")
	}
	b.head.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	if !b.Expanded {
		t.Fatal("B should open")
	}
	if a.Expanded {
		t.Fatal("exclusive should close A")
	}
}

func TestSeparatorAndSpacer(t *testing.T) {
	sep := NewSeparator()
	sz := sep.Measure(layout.Loose(200, 40))
	if sz.Y < 4 || sz.X < 16 {
		t.Fatalf("h-sep %v", sz)
	}
	v := NewVSeparator()
	vz := v.Measure(layout.Loose(40, 80))
	if vz.X < 4 || vz.Y < 16 {
		t.Fatalf("v-sep %v", vz)
	}
	sp := NewSpacerSize(12, 20)
	ss := sp.Measure(layout.Loose(100, 100))
	if ss.X != 12 || ss.Y != 20 {
		t.Fatalf("spacer %v", ss)
	}
	grow := NewSpacer()
	gz := grow.Measure(layout.Constraints{MinW: 0, MinH: 0, MaxW: 80, MaxH: 40})
	if gz.X != 0 || gz.Y != 0 {
		t.Fatalf("zero spacer %v", gz)
	}
}
