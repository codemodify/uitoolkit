package widgets

import (
	"github.com/codemodify/uitoolkit/style"
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

// **A press on the scroll bar scrolls; it does not drag a row.**
//
// Reported from using the toolkit: grabbing a view's scroll bar dragged
// an item out of the view instead of scrolling it. The press itself is
// taken by the bar in MousePress — that part was right — but `DragAt`
// is asked on its own once the pointer has moved past the threshold,
// and it asked only "is there a row at this *height*". A bar runs down
// the rows, so every point on it has one.
func TestAPressOnTheScrollBarDoesNotDragARow(t *testing.T) {
	lk := style.DarkLook()

	t.Run("list", func(t *testing.T) {
		l := NewListView(40, func(i int) string { return "row" }, nil)
		l.SetLook(lk)
		l.OnDrag = func([]int) *widget.Drag { return &widget.Drag{Types: []string{"text/plain"}} }
		l.Selected = 0
		l.Arrange(paintengine2d.XYWH(0, 0, 220, 140))

		sp := l.vparts()
		if sp.Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		// The middle of the bar, in the list's own coordinates.
		on := fromView(sp.Bar, l.pad())
		at := paintengine2d.Pt(on.Min.X+on.Dx()/2, on.Min.Y+on.Dy()/2)
		if d := l.DragAt(at); d != nil {
			t.Errorf("a drag started from the scroll bar at %v", at)
		}
		// And a point on the rows still drags, or the test proves nothing.
		if d := l.DragAt(paintengine2d.Pt(20, 20)); d == nil {
			t.Error("a drag from the rows was refused")
		}
	})

	t.Run("tree", func(t *testing.T) {
		var roots []*TreeNode
		for i := 0; i < 40; i++ {
			roots = append(roots, &TreeNode{Label: "node"})
		}
		tv := NewTreeView(roots...)
		tv.SetLook(lk)
		tv.OnDrag = func(*TreeNode) *widget.Drag { return &widget.Drag{Types: []string{"text/plain"}} }
		tv.Arrange(paintengine2d.XYWH(0, 0, 220, 140))

		sp := tv.vparts()
		if sp.Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		on := fromView(sp.Bar, tv.frame())
		at := paintengine2d.Pt(on.Min.X+on.Dx()/2, on.Min.Y+on.Dy()/2)
		if d := tv.DragAt(at); d != nil {
			t.Errorf("a drag started from the tree's scroll bar at %v", at)
		}
		if d := tv.DragAt(paintengine2d.Pt(20, 20)); d == nil {
			t.Error("a drag from the tree's nodes was refused")
		}
	})

	t.Run("table", func(t *testing.T) {
		tb := NewTableView([]TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 80}}, 40,
			func(r, c int) string { return "cell" }, nil)
		tb.SetLook(lk)
		tb.OnDrag = func([]int) *widget.Drag { return &widget.Drag{Types: []string{"text/plain"}} }
		tb.Selected = 0
		tb.Arrange(paintengine2d.XYWH(0, 0, 220, 140))

		sp := tb.vparts()
		if sp.Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		on := fromView(sp.Bar, tb.frame())
		at := paintengine2d.Pt(on.Min.X+on.Dx()/2, on.Min.Y+on.Dy()/2)
		if d := tb.DragAt(at); d != nil {
			t.Errorf("a drag started from the table's scroll bar at %v", at)
		}
		if d := tb.DragAt(paintengine2d.Pt(20, tb.headerH()+10)); d == nil {
			t.Error("a drag from the table's rows was refused")
		}
	})
}

// A press anywhere from the row width to the view's own edge belongs to
// the scroll bar — the bar itself and the gutter either side of it — and
// not to the row it happens to be level with.
//
// The list asked `rowAtView`, which tests both axes, and refused all of
// it. The tree and the table asked for a height alone, so the two-pixel
// margins either side of the bar selected the row beside them: a press
// that visibly missed every row still moved the selection, and the drag
// it started carried that row away.
func TestAPressBesideTheScrollBarSelectsNothing(t *testing.T) {
	lk := style.DarkLook()

	t.Run("list", func(t *testing.T) {
		l := NewListView(40, func(i int) string { return "row" }, nil)
		l.SetLook(lk)
		l.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if l.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		press := func(x float32) {
			l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(x, 70), Button: platform.ButtonLeft})
		}
		press(20)
		if l.Selected < 0 {
			t.Fatal("a press on the rows selected nothing; the test proves nothing")
		}
		for x := l.rowsW(); x < l.inner().Dx(); x++ {
			l.Selected = -1
			press(x)
			if l.Selected >= 0 {
				t.Errorf("a press at x=%v, beside the bar, selected row %d", x, l.Selected)
			}
		}
	})

	t.Run("tree", func(t *testing.T) {
		var roots []*TreeNode
		for i := 0; i < 40; i++ {
			roots = append(roots, &TreeNode{Label: "node"})
		}
		tv := NewTreeView(roots...)
		tv.SetLook(lk)
		tv.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if tv.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		press := func(x float32) {
			tv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(x, 70), Button: platform.ButtonLeft})
		}
		press(100)
		if tv.Selected == nil {
			t.Fatal("a press on the nodes selected nothing; the test proves nothing")
		}
		for x := tv.rowsW(); x < tv.inner().Dx(); x++ {
			tv.Selected = nil
			press(x)
			if tv.Selected != nil {
				t.Errorf("a press at x=%v, beside the bar, selected a node", x)
			}
		}
	})

	t.Run("table", func(t *testing.T) {
		tb := NewTableView([]TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 80}}, 40,
			func(r, c int) string { return "cell" }, nil)
		tb.SetLook(lk)
		tb.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if tb.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		y := tb.headerH() + tb.rowH()*1.5
		press := func(x float32) {
			tb.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(x, y), Button: platform.ButtonLeft})
		}
		press(20)
		if tb.Selected < 0 {
			t.Fatal("a press on the rows selected nothing; the test proves nothing")
		}
		for x := tb.viewW(); x < tb.inner().Dx(); x++ {
			tb.Selected = -1
			press(x)
			if tb.Selected >= 0 {
				t.Errorf("a press at x=%v, beside the bar, selected row %d", x, tb.Selected)
			}
		}
	})
}

// The hover highlight stops where the rows do. The pointer crossing the
// scroll bar is not over the row level with it, and highlighting that
// row promised it was about to be acted on — when a press there would
// scroll instead.
//
// The same y-only hit test as the press and the drag, and the same
// three views; the list was right already.
func TestHoverStopsWhereTheRowsDo(t *testing.T) {
	lk := style.DarkLook()

	t.Run("list", func(t *testing.T) {
		l := NewListView(40, func(i int) string { return "row" }, nil)
		l.SetLook(lk)
		l.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if l.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		move := func(x float32) {
			l.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, 70)})
		}
		move(20)
		if l.hovered < 0 {
			t.Fatal("the pointer on the rows hovered nothing; the test proves nothing")
		}
		for x := l.rowsW(); x < l.inner().Dx(); x++ {
			move(x)
			if l.hovered >= 0 {
				t.Errorf("x=%v, beside the bar, hovered row %d", x, l.hovered)
			}
			move(20)
		}
	})

	t.Run("tree", func(t *testing.T) {
		var roots []*TreeNode
		for i := 0; i < 40; i++ {
			roots = append(roots, &TreeNode{Label: "node"})
		}
		tv := NewTreeView(roots...)
		tv.SetLook(lk)
		tv.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if tv.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		move := func(x float32) {
			tv.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, 70)})
		}
		move(100)
		if tv.hover == nil {
			t.Fatal("the pointer on the nodes hovered nothing; the test proves nothing")
		}
		for x := tv.rowsW(); x < tv.inner().Dx(); x++ {
			move(x)
			if tv.hover != nil {
				t.Errorf("x=%v, beside the bar, hovered a node", x)
			}
			move(100)
		}
	})

	t.Run("table", func(t *testing.T) {
		tb := NewTableView([]TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 80}}, 40,
			func(r, c int) string { return "cell" }, nil)
		tb.SetLook(lk)
		tb.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		if tb.vparts().Bar.Empty() {
			t.Skip("this look draws no scroll bar")
		}
		y := tb.headerH() + tb.rowH()*1.5
		move := func(x float32) {
			tb.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, y)})
		}
		move(20)
		if tb.hovered < 0 {
			t.Fatal("the pointer on the rows hovered nothing; the test proves nothing")
		}
		for x := tb.viewW(); x < tb.inner().Dx(); x++ {
			move(x)
			if tb.hovered >= 0 {
				t.Errorf("x=%v, beside the bar, hovered row %d", x, tb.hovered)
			}
			move(20)
		}
	})
}

// A skin can place the rows anywhere inside the list's box, and then the
// list's own frame is zero: `pad` is the inset to the rows and `frame`
// is not. Every pointer question has to be asked from `pad`.
//
// Asked from `frame`, a placed list read a drop as being as many rows
// lower as the skin's chrome above the rows is tall — a file dropped on
// the first row landed on the third — and a drag could be started from
// the chrome, where there is no row at all.
func TestAPlacedListAsksFromTheRowsItWasGiven(t *testing.T) {
	const top, left, rowH = 24, 8, 20

	newPlaced := func() *ListView {
		l := NewListView(40, func(i int) string { return "row" }, nil)
		l.SetLook(style.DarkLook())
		l.RowGeo = func(style.LookAndFeel) *RowGeometry {
			return &RowGeometry{
				Rows:      paintengine2d.XYWH(left, top, 200, 100),
				RowHeight: rowH,
			}
		}
		l.Arrange(paintengine2d.XYWH(0, 0, 220, 140))
		return l
	}

	t.Run("a drop lands on the row it was dropped on", func(t *testing.T) {
		l := newPlaced()
		got := -1
		l.OnDropRow = func(row int, e widget.DropEvent) bool { got = row; return true }
		// The middle of the first row, in the list's own coordinates.
		at := paintengine2d.Pt(left+20, top+rowH/2)
		if !l.Drop(widget.DropEvent{Pos: at}) {
			t.Fatalf("the drop at %v was refused", at)
		}
		if got != 0 {
			t.Errorf("a drop on the first row landed on row %d", got)
		}
	})

	t.Run("the indicator marks that row too", func(t *testing.T) {
		l := newPlaced()
		l.OnDropRow = func(int, widget.DropEvent) bool { return true }
		l.DragOver(paintengine2d.Pt(left+20, top+rowH/2))
		if l.dropRow != 0 {
			t.Errorf("the drag over the first row marked row %d", l.dropRow)
		}
	})

	t.Run("no drag starts above the rows", func(t *testing.T) {
		l := newPlaced()
		l.Selected = 0
		l.OnDrag = func([]int) *widget.Drag { return &widget.Drag{Types: []string{"text/plain"}} }
		if d := l.DragAt(paintengine2d.Pt(left+20, top/2)); d != nil {
			t.Error("a drag started from the skin's own chrome, above the first row")
		}
		if d := l.DragAt(paintengine2d.Pt(left+20, top+rowH/2)); d == nil {
			t.Error("a drag from the first row was refused")
		}
	})
}
