package widgets

import (
	"fmt"
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
	rects := bar.tabRects()
	if len(rects) < 3 {
		t.Fatal("tab rects")
	}
	pt := paintengine2d.Pt((rects[2].Min.X+rects[2].Max.X)*0.5, 15)
	bar.MousePress(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	bar.MouseRelease(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	if bar.Selected != 2 {
		t.Fatalf("click %d", bar.Selected)
	}
	bar.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, -1)})
	if bar.Selected != 1 {
		t.Fatalf("wheel %d", bar.Selected)
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

func mailLikeMenuItems() []*MenuItem {
	return []*MenuItem{
		Item("Reply", nil),
		Item("Forward", nil),
		Sep(),
		Item("Mark as Read", nil),
		Item("Mark as Unread", nil),
		Item("Star", nil),
		Sep(),
		Item("Tag · Important", nil),
		Item("Mute Thread", nil),
		Item("Add sender to VIP", nil),
		Item("Archive", nil),
		Item("Junk", nil),
		Item("Delete", nil),
	}
}

func TestPopupMenuMeasureFitsLongestLabel(t *testing.T) {
	pop := NewPopupMenu(mailLikeMenuItems()...)
	pop.SetHost(&host{})
	sz := pop.Measure(layout.Unbounded())
	pop.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	f := pop.Look().Font()
	longest := "Add sender to VIP"
	ch := pop.chrome()
	need := f.Advance(longest) + ch.PadL + ch.CheckCol() + ch.ItemPad + ch.PadR
	if sz.X+0.5 < need {
		t.Fatalf("width %v < need %v for %q", sz.X, need, longest)
	}
	vip := -1
	for i, it := range pop.Items {
		if it != nil && it.Text == longest {
			vip = i
			break
		}
	}
	if vip < 0 {
		t.Fatal("missing VIP item")
	}
	lb := pop.LabelBounds(vip)
	item := pop.ItemBounds(vip)
	if adv := f.Advance(longest); adv > lb.Dx()+0.5 {
		t.Fatalf("label column %+v < advance %v", lb, adv)
	} else if lb.Min.X+adv > item.Max.X+0.5 {
		t.Fatalf("text maxX %v exceeds item %+v", lb.Min.X+adv, item)
	}
	last := pop.ItemBounds(len(pop.Items) - 1)
	if last.Empty() || last.Max.Y > sz.Y+1 {
		t.Fatalf("last item %+v clipped by height %v", last, sz.Y)
	}
	if pop.MaxOffset() != 0 {
		t.Fatalf("intrinsic size should not scroll, max=%v", pop.MaxOffset())
	}
}

func TestPopupMenuScrollsWhenClampedShort(t *testing.T) {
	items := make([]*MenuItem, 0, 24)
	for i := 0; i < 20; i++ {
		items = append(items, Item(fmt.Sprintf("Item %02d with a long label", i), nil))
	}
	pop := NewPopupMenu(items...)
	pop.SetHost(&host{})
	full := pop.Measure(layout.Unbounded())
	pop.Arrange(paintengine2d.XYWH(0, 0, full.X, 90))
	if pop.MaxOffset() <= 0 {
		t.Fatalf("expected scroll when height 90 < content %v", full.Y)
	}
	if pop.ItemBounds(0).Min.Y > 8 {
		t.Fatalf("first row should be visible %+v", pop.ItemBounds(0))
	}
	pop.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 400)})
	if pop.OffsetY <= 0 {
		t.Fatal("wheel should scroll")
	}
	pop.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 4000)})
	if pop.OffsetY != pop.MaxOffset() {
		t.Fatalf("wheel to end offset=%v max=%v", pop.OffsetY, pop.MaxOffset())
	}
	last := pop.ItemBounds(len(items) - 1)
	if last.Max.Y < 80 {
		t.Fatalf("scrolled last item still hidden %+v", last)
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

func TestPopupMenuMnemonicKey(t *testing.T) {
	n := 0
	pop := NewPopupMenu(
		ItemAccel("&New", "Ctrl+N", nil),
		ItemAccel("&Quit", "Ctrl+Q", func() { n++ }),
	)
	pop.SetHost(&host{})
	pop.Arrange(paintengine2d.XYWH(0, 0, 180, 70))
	if !pop.KeyPress(widget.KeyEvent{Key: platform.KeyQ}) {
		t.Fatal("q should hit Quit")
	}
	if n != 1 {
		t.Fatalf("quit n=%d", n)
	}
}

func TestTreeViewPageKeys(t *testing.T) {
	var kids []*TreeNode
	for i := 0; i < 20; i++ {
		kids = append(kids, NewTreeNode("leaf"))
	}
	root := NewTreeNode("root", kids...)
	root.Expanded = true
	tree := NewTreeView(root)
	tree.SetHost(&host{})
	tree.Arrange(paintengine2d.XYWH(0, 0, 200, 80))
	tree.Selected = root
	tree.KeyPress(widget.KeyEvent{Key: platform.KeyPageDown})
	if tree.Selected == root {
		t.Fatal("page down should move")
	}
}
