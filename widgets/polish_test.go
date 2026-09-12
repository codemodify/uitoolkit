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
	if ta.Caret() != runeCount("hello\n") {
		t.Fatalf("caret after return %d", ta.Caret())
	}
	if n := len(ta.Lines()); n != 2 {
		t.Fatalf("lines after return %d", n)
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

func TestTextAreaNewlineMidAndReplaceSel(t *testing.T) {
	ta := NewTextArea("hello", "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 240, 90))
	ta.SetSelection(2, 2)
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if ta.Text != "he\nllo" {
		t.Fatalf("mid return %q", ta.Text)
	}
	if ta.Caret() != 3 {
		t.Fatalf("caret after mid return %d", ta.Caret())
	}
	ta.SetSelection(0, runeCount(ta.Text))
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if ta.Text != "\n" {
		t.Fatalf("replace sel %q", ta.Text)
	}
	if ta.Caret() != 1 {
		t.Fatalf("caret after replace %d", ta.Caret())
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
	var got []bool
	s := NewSwitch("Dark", false, func(v bool) { got = append(got, v) })
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 140, 32))
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	s.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !s.On {
		t.Fatal("click should turn on")
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if s.On {
		t.Fatal("space should toggle off")
	}
	s.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if !s.On {
		t.Fatal("return should toggle on")
	}
	if len(got) != 3 || !got[0] || got[1] || !got[2] {
		t.Fatalf("onchange %v", got)
	}
	s.SetEnabled(false)
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	s.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	s.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if !s.On || len(got) != 3 {
		t.Fatalf("disabled toggled on=%v got=%v", s.On, got)
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
	if a.Content() != nil && a.Content().Visible() {
		t.Fatal("closed section body still visible")
	}
}

func TestAccordionExclusiveConstruct(t *testing.T) {
	a := NewExpander("A", true, NewLabel("aa"))
	b := NewExpander("B", true, NewLabel("bb"))
	acc := NewAccordion(true, a, b)
	if len(acc.Sections()) != 2 {
		t.Fatalf("sections %d", len(acc.Sections()))
	}
	if a.Expanded {
		t.Fatal("exclusive construct should close earlier sections")
	}
	if !b.Expanded {
		t.Fatal("last open section should stay open")
	}
	c := NewExpander("C", false, NewLabel("cc"))
	d := NewExpander("D", false, NewLabel("dd"))
	open := NewAccordion(false, c, d)
	c.SetExpanded(true)
	d.SetExpanded(true)
	if !c.Expanded || !d.Expanded {
		t.Fatal("non-exclusive should allow both")
	}
	_ = open
}

func TestAccordionExclusiveYieldsFocus(t *testing.T) {
	h := &host{}
	inner := NewSwitch("Hidden", true, nil)
	a := NewExpander("A", true, inner)
	b := NewExpander("B", false, NewLabel("bb"))
	acc := NewAccordion(true, a, b)
	acc.SetHost(h)
	sz := acc.Measure(layout.Loose(220, 300))
	acc.Arrange(paintengine2d.XYWH(0, 0, 220, sz.Y))
	inner.RequestFocus()
	if h.Focus() != inner {
		t.Fatal("inner should have focus")
	}
	b.SetExpanded(true)
	if h.Focus() == inner {
		t.Fatal("focus stuck in collapsed body")
	}
	if h.Focus() != a.head {
		t.Fatalf("focus %T", h.Focus())
	}
	if inner.Visible() && a.Content() != nil && a.Content().Visible() {
		t.Fatal("A should be collapsed")
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
