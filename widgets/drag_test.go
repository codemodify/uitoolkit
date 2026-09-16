package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// laidOutField is a text field sized and hosted, with its text selected
// from a to b.
func laidOutField(t *testing.T, text string, a, b int) *TextField {
	t.Helper()
	f := NewTextField(text, "", nil)
	f.SetHost(&host{})
	sz := f.Measure(layout.Loose(300, 40))
	f.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	f.SetSelection(a, b)
	return f
}

// caretX is the x of caret index i, a point a press can land on.
func fieldCaretX(f *TextField, i int) float32 {
	b := f.LocalBounds()
	return f.Look().Font().CaretX(f.Text, i) + b.Min.X + 6
}

// A press inside the selection keeps it: it may be the start of a drag of
// that text, and clearing it first would leave nothing to drag.
func TestTextFieldPressInSelectionKeepsIt(t *testing.T) {
	f := laidOutField(t, "Hello world", 0, 5)
	f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(fieldCaretX(f, 2), 10), Button: platform.ButtonLeft})
	if got := f.SelectedText(); got != "Hello" {
		t.Fatalf("selection after the press: %q", got)
	}
	d := f.DragAt(paintengine2d.Pt(fieldCaretX(f, 2), 10))
	if d == nil {
		t.Fatal("the press should drag the selected text")
	}
	if b, ok := d.Read("text/plain"); !ok || string(b) != "Hello" {
		t.Fatalf("drag carries %q ok %v", b, ok)
	}
	if d.Image == nil {
		t.Fatal("a text drag draws a picture for the pointer")
	}
}

// A press outside the selection is an ordinary click: it collapses the
// selection where it landed and drags nothing.
func TestTextFieldPressOutsideSelectionCollapsesIt(t *testing.T) {
	f := laidOutField(t, "Hello world", 0, 5)
	f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(fieldCaretX(f, 9), 10), Button: platform.ButtonLeft})
	if got := f.SelectedText(); got != "" {
		t.Fatalf("selection %q, want it collapsed", got)
	}
	if d := f.DragAt(paintengine2d.Pt(fieldCaretX(f, 9), 10)); d != nil {
		t.Fatal("nothing selected, nothing to drag")
	}
}

// A press inside the selection that never becomes a drag is a click after
// all: the release puts the caret where the press landed.
func TestTextFieldClickInSelectionMovesTheCaret(t *testing.T) {
	f := laidOutField(t, "Hello world", 0, 5)
	at := paintengine2d.Pt(fieldCaretX(f, 3), 10)
	f.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	f.MouseMove(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	f.MouseRelease(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if got := f.SelectedText(); got != "" {
		t.Fatalf("the click should have collapsed the selection, got %q", got)
	}
	// Exactly where a click there would have put it had nothing been
	// selected — the deferral must change nothing but the timing.
	plain := laidOutField(t, "Hello world", 0, 0)
	plain.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if f.Caret() != plain.Caret() {
		t.Fatalf("caret %d, want %d", f.Caret(), plain.Caret())
	}
}

// A move while the press waits to become a drag must not extend the
// selection under it.
func TestTextFieldDragPressDoesNotExtendTheSelection(t *testing.T) {
	f := laidOutField(t, "Hello world", 0, 5)
	f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(fieldCaretX(f, 2), 10), Button: platform.ButtonLeft})
	f.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(fieldCaretX(f, 9), 10), Button: platform.ButtonLeft})
	if got := f.SelectedText(); got != "Hello" {
		t.Fatalf("selection %q, want it untouched", got)
	}
}

// A text area does the same, and a move takes the text away once the
// target says it performed one.
func TestTextAreaDragOfSelectionMoves(t *testing.T) {
	a := NewTextArea("one two three", "", nil)
	a.SetHost(&host{})
	sz := a.Measure(layout.Loose(300, 120))
	a.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	a.SetSelection(0, 3)
	at := paintengine2d.Pt(a.Look().Font().CaretX("one two three", 1)+6, 10)
	a.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	d := a.DragAt(at)
	if d == nil {
		t.Fatal("the press should drag the selection")
	}
	if !d.Actions.Has(platform.DragMove) {
		t.Fatalf("an editable area allows a move: %v", d.Actions)
	}
	d.Done(platform.DragMove)
	if a.Text != " two three" {
		t.Fatalf("after the move: %q", a.Text)
	}
}

// A read-only area still drags its text, but only as a copy: nothing may
// take text out of something that cannot be edited.
func TestReadOnlyTextAreaDragsACopyOnly(t *testing.T) {
	a := NewTextArea("one two three", "", nil)
	a.ReadOnly = true
	a.SetHost(&host{})
	sz := a.Measure(layout.Loose(300, 120))
	a.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	a.SetSelection(0, 3)
	at := paintengine2d.Pt(a.Look().Font().CaretX("one two three", 1)+6, 10)
	a.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	d := a.DragAt(at)
	if d == nil {
		t.Fatal("read-only text still drags")
	}
	if d.Actions.Has(platform.DragMove) {
		t.Fatalf("a read-only area must not allow a move: %v", d.Actions)
	}
	d.Done(platform.DragMove)
	if a.Text != "one two three" {
		t.Fatalf("the text must be untouched: %q", a.Text)
	}
}

// A drop of a text widget's own drag back into itself is not a move: the
// text was inserted here, and taking the original away too would lose it.
func TestTextAreaSelfDropIsNotAMove(t *testing.T) {
	a := NewTextArea("one two three", "", nil)
	a.SetHost(&host{})
	sz := a.Measure(layout.Loose(300, 120))
	a.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	a.SetSelection(0, 3)
	at := paintengine2d.Pt(a.Look().Font().CaretX("one two three", 1)+6, 10)
	a.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	d := a.DragAt(at)
	if d == nil {
		t.Fatal("no drag")
	}
	a.Drop(widget.DropEvent{Pos: paintengine2d.Pt(200, 8), Text: "one", Source: a})
	before := a.Text
	d.Done(platform.DragMove)
	if a.Text != before {
		t.Fatalf("a self-drop must take nothing away: %q then %q", before, a.Text)
	}
}

// A table drags the rows that are selected, through the hook that knows
// what they are worth dragging as.
func TestTableViewDragsTheSelectedRows(t *testing.T) {
	cells := func(row, col int) string { return "cell" }
	tv := NewTableView([]TableColumn{{Title: "Name"}}, 5, cells, nil)
	tv.SetHost(&host{})
	tv.Mode = SelectExtended
	sz := tv.Measure(layout.Loose(300, 200))
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	at := tv.RowBounds(2).Center()
	if d := tv.DragAt(at); d != nil {
		t.Fatal("a table with no OnDrag drags nothing")
	}
	var got []int
	tv.OnDrag = func(rows []int) *widget.Drag {
		got = rows
		return widget.DragFiles("/tmp/a.txt")
	}
	tv.SetSelectedRows([]int{2, 4})
	tv.Selected = 2
	d := tv.DragAt(at)
	if d == nil || len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("drag %v rows %v", d != nil, got)
	}
	// A press on the header sorts a column; it never drags rows.
	if d := tv.DragAt(paintengine2d.Pt(10, tv.frame().Top+1)); d != nil {
		t.Fatal("the header drags no rows")
	}
}

// A tree takes a drop on the node under it, and says so while the drag is
// over that row.
func TestTreeViewDropsOnTheNodeUnderIt(t *testing.T) {
	kids := []*TreeNode{NewTreeNode("one"), NewTreeNode("two")}
	root := NewTreeNode("root", kids...)
	root.Expanded = true
	tv := NewTreeView(root)
	tv.SetHost(&host{})
	sz := tv.Measure(layout.Loose(300, 200))
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	if len(tv.DropTypes()) != 0 {
		t.Fatal("a tree with no OnDropNode takes no drops")
	}
	var on *TreeNode
	tv.OnDropNode = func(n *TreeNode, e widget.DropEvent) bool {
		on = n
		return true
	}
	if got := tv.DropTypes(); len(got) != 1 || got[0] != "text/uri-list" {
		t.Fatalf("drop types %q", got)
	}
	at := paintengine2d.Pt(40, tv.rowH()*1.5+tv.frame().Top)
	tv.DragOver(at)
	if tv.dropRow != 1 {
		t.Fatalf("highlighted row %d, want 1", tv.dropRow)
	}
	tv.DragLeave()
	if tv.dropRow != -1 {
		t.Fatal("the highlight must go when the drag does")
	}
	if !tv.Drop(widget.DropEvent{Pos: at, Paths: []string{"/tmp/a.txt"}}) {
		t.Fatal("the drop should have been taken")
	}
	if on == nil || on.Label != "one" {
		t.Fatalf("dropped on %v", on)
	}
}

// A tab strip takes a drop on the tab under it.
func TestBrowserTabsDropOnATab(t *testing.T) {
	tabs := NewBrowserTabs("first", "second", "third")
	tabs.SetHost(&host{})
	sz := tabs.Measure(layout.Loose(600, 60))
	tabs.Arrange(paintengine2d.XYWH(0, 0, 600, sz.Y))
	if len(tabs.DropTypes()) != 0 {
		t.Fatal("a strip with no OnDropTab takes no drops")
	}
	got := -1
	tabs.OnDropTab = func(i int, e widget.DropEvent) bool {
		got = i
		return true
	}
	g := tabs.geom()
	at := tabs.slotOf(1, g).Center()
	tabs.DragOver(at)
	if tabs.dropTab != 1 {
		t.Fatalf("highlighted tab %d, want 1", tabs.dropTab)
	}
	if !tabs.Drop(widget.DropEvent{Pos: at, Paths: []string{"/tmp/a.txt"}}) || got != 1 {
		t.Fatalf("dropped on tab %d", got)
	}
	if tabs.dropTab != -1 {
		t.Fatal("the highlight goes with the drop")
	}
}
