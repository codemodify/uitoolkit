package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

func TestParseMnemonic(t *testing.T) {
	label, key, idx := ParseMnemonic("&File")
	if label != "File" || key != platform.KeyF || idx != 0 {
		t.Fatalf("File: %q %v %d", label, key, idx)
	}
	label, key, idx = ParseMnemonic("E&xit")
	if label != "Exit" || key != platform.KeyX || idx != 1 {
		t.Fatalf("Exit: %q %v %d", label, key, idx)
	}
	label, key, idx = ParseMnemonic("Help")
	if label != "Help" || key != platform.KeyUnknown || idx != -1 {
		t.Fatalf("Help: %q %v %d", label, key, idx)
	}
}

func TestTextFieldClipboardStubs(t *testing.T) {
	platform.ClipboardSet("")
	tf := NewTextField("hello world", "", nil)
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 200, 32))
	tf.SetSelection(0, 5)
	if tf.SelectedText() != "hello" {
		t.Fatalf("sel %q", tf.SelectedText())
	}
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyC, Mods: platform.ModCtrl})
	if platform.ClipboardGet() != "hello" {
		t.Fatalf("copy %q", platform.ClipboardGet())
	}
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyX, Mods: platform.ModCtrl})
	if tf.Text != " world" {
		t.Fatalf("cut %q", tf.Text)
	}
	tf.SetSelection(0, 0)
	tf.KeyPress(widget.KeyEvent{Key: platform.KeyV, Mods: platform.ModCtrl})
	if tf.Text != "hello world" {
		t.Fatalf("paste %q", tf.Text)
	}
}

func TestTabViewSwapsContent(t *testing.T) {
	a := NewLabel("Alpha")
	b := NewLabel("Beta")
	tv := NewTabView(Tab{Title: "A", Content: a}, Tab{Title: "B", Content: b})
	tv.SetHost(&host{})
	tv.Measure(layout.Loose(240, 160))
	tv.Arrange(paintengine2d.XYWH(0, 0, 240, 160))
	if !a.Visible() || b.Visible() {
		t.Fatalf("initial vis a=%v b=%v", a.Visible(), b.Visible())
	}
	focus := widget.Focusables(tv)
	for _, c := range focus {
		if c == b {
			t.Fatal("hidden tab page should not be in tab order")
		}
	}
	tv.Select(1)
	if a.Visible() || !b.Visible() {
		t.Fatalf("after select vis a=%v b=%v", a.Visible(), b.Visible())
	}
	if tv.Bar().Selected != 1 {
		t.Fatalf("bar %d", tv.Bar().Selected)
	}
}

func TestTabBarKeysAndPress(t *testing.T) {
	bar := NewTabBar("One", "Two", "Three")
	bar.SetHost(&host{})
	bar.Arrange(paintengine2d.XYWH(0, 0, 300, 30))
	n := -1
	bar.OnSelect = func(i int) { n = i }
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if bar.Selected != 1 || n != 1 {
		t.Fatalf("right sel=%d n=%d", bar.Selected, n)
	}
	bar.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(250, 15), Button: platform.ButtonLeft})
	bar.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(250, 15), Button: platform.ButtonLeft})
	if bar.Selected != 2 {
		t.Fatalf("click %d", bar.Selected)
	}
}

func TestTreeViewExpandSelect(t *testing.T) {
	child := NewTreeNode("child.go")
	src := NewTreeNode("src", child)
	src.Expanded = false
	tree := NewTreeView(src)
	tree.SetHost(&host{})
	tree.Arrange(paintengine2d.XYWH(0, 0, 200, 160))
	if n := len(tree.flatten()); n != 1 {
		t.Fatalf("collapsed rows %d", n)
	}
	tree.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(12, 13), Button: platform.ButtonLeft})
	if !src.Expanded {
		t.Fatal("expander should open")
	}
	if n := len(tree.flatten()); n != 2 {
		t.Fatalf("expanded rows %d", n)
	}
	tree.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if tree.Selected != child {
		t.Fatalf("down selected %+v", tree.Selected)
	}
	tree.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	if tree.Selected != src {
		t.Fatalf("left to parent %+v", tree.Selected)
	}
}

func TestStatusBarSet(t *testing.T) {
	sb := NewStatusBar("Ready", "Ln 1")
	sb.Set(0, "Saved")
	if sb.Parts()[0] != "Saved" {
		t.Fatalf("%v", sb.Parts())
	}
	sz := sb.Measure(layout.Loose(400, 100))
	if sz.Y < 20 {
		t.Fatalf("height %v", sz)
	}
}

func TestListViewContext(t *testing.T) {
	got := -2
	var pos paintengine2d.Point
	lv := NewListView(5, func(i int) string { return "x" }, nil)
	lv.OnContext = func(i int, p paintengine2d.Point) {
		got = i
		pos = p
	}
	lv.SetHost(&host{})
	lv.Arrange(paintengine2d.XYWH(10, 20, 120, 140))
	lv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 40), Button: platform.ButtonRight})
	if got < 0 {
		t.Fatalf("context i=%d", got)
	}
	if pos.X < 10 {
		t.Fatalf("window pos %+v", pos)
	}
	if lv.Selected < 0 {
		t.Fatal("right-click should select")
	}
}

func TestPopupMenuActivate(t *testing.T) {
	n := 0
	pop := NewPopupMenu(Item("New", func() { n++ }), Sep(), Item("Quit", nil))
	pop.SetHost(&host{})
	sz := pop.Measure(layout.Loose(240, 200))
	pop.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	if sz.Y < 40 {
		t.Fatalf("menu size %v", sz)
	}
	pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if n != 1 {
		t.Fatalf("activate n=%d", n)
	}
}

func TestMenuBarMnemonic(t *testing.T) {
	mb := NewMenuBar(
		NewMenu("&File", Item("About", nil)),
		NewMenu("&Edit", Item("Copy", nil)),
	)
	mb.SetHost(&host{})
	mb.Arrange(paintengine2d.XYWH(0, 0, 400, 28))
	if !mb.HandleAlt(platform.KeyF) {
		t.Fatal("alt+F")
	}
	// without a PopupHost the drop-down cannot mount, but index is tracked
	if mb.focus != 0 {
		t.Fatalf("focus %d", mb.focus)
	}
}
