package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The click-gesture rule is one rule, and the other two row widgets had the
// same hole in it as ListView: a table activated a row, and a tree expanded a
// node, from one click and an interruption. Neither was in the report that
// found it on the list; both are the same defect in the same shipped release
// line, and a rule that holds in one of three widgets is not a rule.

func tableRig(t *testing.T, rows int) (*TableView, *[]int) {
	t.Helper()
	var act []int
	cols := []TableColumn{{Title: "Name", Width: 120}}
	tv := NewTableView(cols, rows, func(row, col int) string { return "cell" }, nil)
	tv.OnActivate = func(i int) { act = append(act, i) }
	tv.SetLook(style.DarkLook())
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	return tv, &act
}

// tableRowPoint is a point in the middle of row i of a table.
func tableRowPoint(tv *TableView, i int) paintengine2d.Point {
	r := tv.RowBounds(i)
	return paintengine2d.Pt(r.Min.X+4, r.Min.Y+r.Dy()/2)
}

func TestATablesInterruptedGestureIsNotADoubleClick(t *testing.T) {
	for _, tc := range []struct {
		what      string
		interrupt func(tv *TableView)
	}{
		{"the pointer left it", func(tv *TableView) { tv.MouseExit() }},
		{"it lost the focus", func(tv *TableView) { tv.FocusLost() }},
		{"the wheel scrolled it", func(tv *TableView) {
			tv.MouseWheel(widget.MouseEvent{Pos: tableRowPoint(tv, 2), Scroll: paintengine2d.Pt(0, 4), Precise: true})
		}},
		{"a secondary press opened a menu", func(tv *TableView) {
			tv.MousePress(widget.MouseEvent{Pos: tableRowPoint(tv, 2), Button: platform.ButtonRight})
		}},
		{"the keyboard moved the selection", func(tv *TableView) {
			tv.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
		}},
		{"the content was replaced", func(tv *TableView) { tv.RowCount = 12 }},
		{"it was scrolled programmatically", func(tv *TableView) { tv.ScrollTo(4) }},
	} {
		tv, act := tableRig(t, 40)
		tv.MousePress(widget.MouseEvent{Pos: tableRowPoint(tv, 2), Button: platform.ButtonLeft})
		tc.interrupt(tv)
		second := tableRowPoint(tv, 2)
		tv.MousePress(widget.MouseEvent{Pos: second, Button: platform.ButtonLeft})
		if len(*act) != 0 {
			t.Errorf("a click, %s, and a click activated %v, want nothing", tc.what, *act)
		}
	}
}

// The control: a table's ordinary double click still commits.
func TestATablesUninterruptedDoubleClickStillActivates(t *testing.T) {
	tv, act := tableRig(t, 40)
	at := tableRowPoint(tv, 2)
	tv.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	tv.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 1 || (*act)[0] != 2 {
		t.Errorf("a double click activated %v, want row 2 once", *act)
	}
}

// A tree's double click expands or collapses the node, so a stale one is a
// folder that opens itself.
func treeRig(t *testing.T) (*TreeView, *TreeNode) {
	t.Helper()
	leaf := NewTreeNode("child")
	folder := NewTreeNode("folder", leaf)
	other := NewTreeNode("other")
	root := NewTreeNode("root", folder, other)
	root.Expanded = true
	// Closed to start with, so an unwanted toggle is an unwanted *opening*.
	folder.Expanded = false
	tv := NewTreeView(root)
	tv.SetLook(style.DarkLook())
	tv.SetHost(&host{})
	sz := tv.Measure(layout.Loose(300, 400))
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	return tv, folder
}

// treeLabelPoint is a point on a node's label, clear of its expander.
func treeLabelPoint(tv *TreeView, i int) paintengine2d.Point {
	return paintengine2d.Pt(200, tv.rowH()*(float32(i)+0.5)+tv.frame().Top)
}

func TestATreesInterruptedGestureDoesNotToggleANode(t *testing.T) {
	for _, tc := range []struct {
		what      string
		interrupt func(tv *TreeView)
	}{
		{"the pointer left it", func(tv *TreeView) { tv.MouseExit() }},
		{"it lost the focus", func(tv *TreeView) { tv.FocusLost() }},
		{"a secondary press opened a menu", func(tv *TreeView) {
			tv.MousePress(widget.MouseEvent{Pos: treeLabelPoint(tv, 1), Button: platform.ButtonRight})
		}},
		{"the keyboard moved the selection", func(tv *TreeView) {
			tv.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
		}},
	} {
		tv, folder := treeRig(t)
		at := treeLabelPoint(tv, 1) // row 0 is the root, row 1 the folder
		tv.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
		if folder.Expanded {
			t.Fatalf("%s: one click already expanded the folder", tc.what)
		}
		tc.interrupt(tv)
		tv.MousePress(widget.MouseEvent{Pos: treeLabelPoint(tv, 1), Button: platform.ButtonLeft})
		if folder.Expanded {
			t.Errorf("a click, %s, and a click expanded the folder", tc.what)
		}
	}
}

// The control: a tree's ordinary double click still toggles.
func TestATreesUninterruptedDoubleClickToggles(t *testing.T) {
	tv, folder := treeRig(t)
	at := treeLabelPoint(tv, 1)
	tv.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	tv.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if !folder.Expanded {
		t.Error("a double click on a folder did not expand it")
	}
}
