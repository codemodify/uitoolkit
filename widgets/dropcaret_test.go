package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Which gap a pointer is in, row by row: the top and bottom quarters of a
// row are the gaps either side of it, the middle half is the row itself,
// and past the last row is the gap at the end.
func TestRowDropSpotPicksTheGap(t *testing.T) {
	const rowH, count = 20, 3
	for _, tc := range []struct {
		name    string
		y       float32
		onto    bool
		row, at int
	}{
		{"above the first row", -5, true, -1, 0},
		{"the top of row 0", 1, true, -1, 0},
		{"the middle of row 0", 10, true, 0, -1},
		{"the bottom of row 0", 19, true, -1, 1},
		{"the top of row 1", 21, true, -1, 1},
		{"the middle of row 2", 50, true, 2, -1},
		{"the bottom of row 2", 59, true, -1, 3},
		{"past the last row", 200, true, -1, 3},
		// A view that takes no drop onto a row has no dead band in the
		// middle of one: the caret jumps at the row's midpoint.
		{"no onto: above the midpoint of row 1", 29, false, -1, 1},
		{"no onto: below the midpoint of row 1", 31, false, -1, 2},
	} {
		row, at := rowDropSpot(tc.y, 0, rowH, count, tc.onto)
		if row != tc.row || at != tc.at {
			t.Errorf("%s: row=%d at=%d, want row=%d at=%d", tc.name, row, at, tc.row, tc.at)
		}
	}
	// An empty view is one gap, so a drop into it is the first row.
	if row, at := rowDropSpot(4, 0, rowH, 0, true); row != -1 || at != 0 {
		t.Fatalf("empty view: row=%d at=%d, want row=-1 at=0", row, at)
	}
	// The gap follows the rows when the view is scrolled.
	if row, at := rowDropSpot(1, 40, rowH, count, true); row != -1 || at != 2 {
		t.Fatalf("scrolled: row=%d at=%d, want row=-1 at=2", row, at)
	}
}

// A list with OnDropAt reorders: the caret marks the gap under the
// pointer, and the drop names the place the row would take — the end of
// the list included.
func TestListDropsBetweenRows(t *testing.T) {
	l := NewListView(4, func(i int) string { return "row" }, nil)
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 4*l.rowH()+l.frame().Top+l.frame().Bottom))
	if len(l.DropTypes()) != 0 {
		t.Fatal("a list with no drop hook takes no drops")
	}
	got := -1
	l.OnDropAt = func(i int, e widget.DropEvent) bool { got = i; return true }
	if types := l.DropTypes(); len(types) != 1 || types[0] != "text/uri-list" {
		t.Fatalf("drop types %q", types)
	}
	rh, top := l.rowH(), l.frame().Top
	// With no OnDropRow every position is a gap: the midpoint of row 1
	// is the boundary between the gap before it and the one after.
	l.DragOver(paintengine2d.Pt(20, top+rh*1.4))
	if l.dropAt != 1 || l.dropRow != -1 {
		t.Fatalf("at=%d row=%d, want at=1 row=-1", l.dropAt, l.dropRow)
	}
	l.DragOver(paintengine2d.Pt(20, top+rh*1.6))
	if l.dropAt != 2 {
		t.Fatalf("past the midpoint of row 1: at=%d, want 2", l.dropAt)
	}
	// Below the last row is the end of the list.
	l.DragOver(paintengine2d.Pt(20, top+rh*3.9))
	if l.dropAt != 4 {
		t.Fatalf("below the last row: at=%d, want 4", l.dropAt)
	}
	if !l.Drop(widget.DropEvent{Pos: paintengine2d.Pt(20, top+rh*3.9), Paths: []string{"/tmp/a"}}) || got != 4 {
		t.Fatalf("dropped at %d, want the end (4)", got)
	}
	if l.dropAt != -1 {
		t.Fatal("the caret goes with the drop")
	}
	l.DragOver(paintengine2d.Pt(20, top+rh*0.5))
	l.DragLeave()
	if l.dropAt != -1 {
		t.Fatal("the caret goes when the drag leaves")
	}
}

// A list that takes drops onto rows as well keeps the middle half of each
// row for them: a drop onto a row is never mistaken for one between two.
func TestListTellsOntoFromBetween(t *testing.T) {
	l := NewListView(3, func(i int) string { return "row" }, nil)
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 3*l.rowH()+l.frame().Top+l.frame().Bottom))
	onRow, atGap := -1, -1
	l.OnDropRow = func(i int, e widget.DropEvent) bool { onRow = i; return true }
	l.OnDropAt = func(i int, e widget.DropEvent) bool { atGap = i; return true }
	rh, top := l.rowH(), l.frame().Top
	mid := paintengine2d.Pt(20, top+rh*1.5)
	l.DragOver(mid)
	if l.dropRow != 1 || l.dropAt != -1 {
		t.Fatalf("the middle of a row: row=%d at=%d, want row=1 at=-1", l.dropRow, l.dropAt)
	}
	if !l.Drop(widget.DropEvent{Pos: mid, Paths: []string{"/tmp/a"}}) || onRow != 1 || atGap != -1 {
		t.Fatalf("onto row %d, gap %d", onRow, atGap)
	}
	edge := paintengine2d.Pt(20, top+rh*1.05)
	l.DragOver(edge)
	if l.dropAt != 1 || l.dropRow != -1 {
		t.Fatalf("the top edge of a row: row=%d at=%d, want row=-1 at=1", l.dropRow, l.dropAt)
	}
	if !l.Drop(widget.DropEvent{Pos: edge, Paths: []string{"/tmp/a"}}) || atGap != 1 {
		t.Fatalf("into gap %d, want 1", atGap)
	}
}

// The table's rows start under its sticky header, so the gaps do too: the
// header itself is the gap above the first row, never a row's own band.
func TestTableCaretSitsUnderTheHeader(t *testing.T) {
	cols := []TableColumn{{Title: "A", Width: 80}, {Title: "B", Width: 80}}
	tv := NewTableView(cols, 3, func(r, c int) string { return "x" }, nil)
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 240))
	got := -1
	tv.OnDropAt = func(i int, e widget.DropEvent) bool { got = i; return true }
	hh, rh, top := tv.headerH(), tv.rowH(), tv.frame().Top
	tv.DragOver(paintengine2d.Pt(20, top+hh*0.5))
	if tv.dropAt != 0 {
		t.Fatalf("over the header: at=%d, want 0", tv.dropAt)
	}
	tv.DragOver(paintengine2d.Pt(20, top+hh+rh*1.6))
	if tv.dropAt != 2 {
		t.Fatalf("past the midpoint of row 1: at=%d, want 2", tv.dropAt)
	}
	at := paintengine2d.Pt(20, top+hh+rh*2.9)
	if !tv.Drop(widget.DropEvent{Pos: at, Paths: []string{"/tmp/a"}}) || got != 3 {
		t.Fatalf("dropped at %d, want the end (3)", got)
	}
}

// The tree's gaps name a place in the tree: the parent a row would hang
// under and the place it would take among that parent's children. The
// pointer's own x picks the level where a gap could mean more than one —
// the end of a folder's children, or the gap before the folder's next
// sibling.
func TestTreeGapNamesAPlaceInTheTree(t *testing.T) {
	a, b := NewTreeNode("a"), NewTreeNode("b")
	folder := NewTreeNode("folder", a, b)
	folder.Expanded = true
	next := NewTreeNode("next")
	tv := NewTreeView(folder, next)
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	tv.OnDropAt = func(*TreeNode, int, widget.DropEvent) bool { return true }
	// Rows: 0 folder, 1 a, 2 b, 3 next.
	deep, shallow := tv.indentX(1)+2, tv.indentX(0)
	for _, tc := range []struct {
		name   string
		at     int
		px     float32
		parent *TreeNode
		index  int
		depth  int
	}{
		{"above the first row", 0, shallow, nil, 0, 0},
		{"into the open folder, before a", 1, deep, folder, 0, 1},
		{"between the folder's children", 2, deep, folder, 1, 1},
		{"after the folder's last child", 3, deep, folder, 2, 1},
		{"before the folder's next sibling", 3, shallow, nil, 1, 0},
		{"past the last row", 4, shallow, nil, 2, 0},
	} {
		parent, index, depth := tv.gapTarget(tc.at, tc.px)
		if parent != tc.parent || index != tc.index || depth != tc.depth {
			t.Errorf("%s: parent=%v index=%d depth=%d, want parent=%v index=%d depth=%d",
				tc.name, nodeLabel(parent), index, depth, nodeLabel(tc.parent), tc.index, tc.depth)
		}
	}
}

func nodeLabel(n *TreeNode) string {
	if n == nil {
		return "<root>"
	}
	return n.Label
}

// A tree with both hooks keeps them apart: the middle of a folder's row
// drops into the folder, its edges drop beside it.
func TestTreeTellsIntoAFolderFromBetweenRows(t *testing.T) {
	kid := NewTreeNode("kid")
	folder := NewTreeNode("folder", kid)
	folder.Expanded = true
	tv := NewTreeView(folder, NewTreeNode("other"))
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	var into *TreeNode
	gap := -2
	tv.OnDropNode = func(n *TreeNode, e widget.DropEvent) bool { into = n; return true }
	tv.OnDropAt = func(p *TreeNode, i int, e widget.DropEvent) bool { gap = i; return true }
	rh, top := tv.rowH(), tv.frame().Top
	mid := paintengine2d.Pt(tv.indentX(0)+20, top+rh*0.5)
	tv.DragOver(mid)
	if tv.dropRow != 0 || tv.dropAt != -1 {
		t.Fatalf("the middle of a folder: row=%d at=%d, want row=0 at=-1", tv.dropRow, tv.dropAt)
	}
	if !tv.Drop(widget.DropEvent{Pos: mid, Paths: []string{"/tmp/a"}}) || into != folder {
		t.Fatalf("dropped into %v", nodeLabel(into))
	}
	edge := paintengine2d.Pt(tv.indentX(0)+20, top+rh*0.05)
	tv.DragOver(edge)
	if tv.dropAt != 0 || tv.dropRow != -1 {
		t.Fatalf("the top edge of a folder: row=%d at=%d, want row=-1 at=0", tv.dropRow, tv.dropAt)
	}
	if !tv.Drop(widget.DropEvent{Pos: edge, Paths: []string{"/tmp/a"}}) || gap != 0 {
		t.Fatalf("into gap %d, want 0", gap)
	}
	// A tree with no OnDropAt has no gaps at all: it behaves as it did
	// before there were any.
	tv.OnDropAt = nil
	tv.DragOver(edge)
	if tv.dropAt != -1 || tv.dropRow != 0 {
		t.Fatalf("without OnDropAt: row=%d at=%d, want row=0 at=-1", tv.dropRow, tv.dropAt)
	}
}

// The caret paints in the gap, across the rows, in the pack's accent —
// and a caret for the gap at the very top is drawn whole rather than
// sliced in half by the viewport's edge.
func TestDropCaretPaintsBetweenRows(t *testing.T) {
	l := NewListView(3, func(i int) string { return "row" }, nil)
	l.SetHost(&host{})
	l.OnDropAt = func(int, widget.DropEvent) bool { return true }
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 3*l.rowH()+l.frame().Top+l.frame().Bottom))
	rh, top := l.rowH(), l.frame().Top
	l.DragOver(paintengine2d.Pt(20, top+rh*1.4))
	img := paintengine2d.NewImage(200, int(l.LocalBounds().Dy())+2)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	widget.PaintTree(l, ctx, nil)
	want := style.DarkLook().Palette().Accent.NRGBA()
	wr, wg, wb, _ := want.RGBA()
	// The caret sits on the boundary between rows 0 and 1, well inside
	// the rows' width.
	x, y := 100, int(top+rh)
	gr, gg, gb, ga := img.At(x, y).RGBA()
	if ga == 0 || gr != wr || gg != wg || gb != wb {
		t.Fatalf("the caret at %d,%d is %v,%v,%v,%v, want the accent %v,%v,%v", x, y, gr, gg, gb, ga, wr, wg, wb)
	}
	// Nothing is painted a row away from it: the caret is a line, not a
	// wash over the row.
	if _, _, _, a := img.At(x, int(top+rh*1.5)).RGBA(); a != 0 {
		mr, mg, mb, _ := img.At(x, int(top+rh*1.5)).RGBA()
		if mr == wr && mg == wg && mb == wb {
			t.Fatal("the middle of a row must not be the accent: that is the mark for a drop onto it")
		}
	}
}

// A caret for the gap above the first row is nudged inside the viewport,
// so the whole bar shows instead of half of it.
func TestRowCaretStaysInsideTheView(t *testing.T) {
	lk := style.DarkLook()
	view := paintengine2d.XYWH(0, 0, 100, 60)
	w := style.DropCaretThickness(lk)
	top := rowCaretRect(lk, 0, 0, 100, view)
	if top.Empty() || top.Dy() < w || top.Min.Y < view.Min.Y {
		t.Fatalf("the caret at the top is %v, want a whole %g-tall bar inside %v", top, w, view)
	}
	bottom := rowCaretRect(lk, 60, 0, 100, view)
	if bottom.Empty() || bottom.Dy() < w || bottom.Max.Y > view.Max.Y {
		t.Fatalf("the caret at the bottom is %v, want a whole bar inside %v", bottom, view)
	}
	// In the middle it is centred on the gap, snapped to whole pixels.
	mid := rowCaretRect(lk, 30, 0, 100, view)
	if mid.Min.Y != float32(int(mid.Min.Y)) || mid.Min.Y > 30 || mid.Max.Y < 30 {
		t.Fatalf("the caret for the gap at y=30 is %v", mid)
	}
}

// A list lays its caret across the rows; a tree indents it to the level
// the drop would land at, which is what tells the end of a folder's
// children from the gap before the folder's next sibling.
func TestTreeCaretIsIndentedToItsLevel(t *testing.T) {
	kid := NewTreeNode("kid")
	folder := NewTreeNode("folder", kid)
	folder.Expanded = true
	tv := NewTreeView(folder, NewTreeNode("other"))
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	tv.OnDropAt = func(*TreeNode, int, widget.DropEvent) bool { return true }
	rh, top := tv.rowH(), tv.frame().Top
	// The gap after the folder's only child, aimed at the child's level.
	tv.DragOver(paintengine2d.Pt(tv.indentX(1)+2, top+rh*1.9))
	if tv.dropAt != 2 || tv.dropDepth != 1 {
		t.Fatalf("at=%d depth=%d, want at=2 depth=1", tv.dropAt, tv.dropDepth)
	}
	deep := tv.indentX(tv.dropDepth)
	// The same gap, aimed at the root level.
	tv.DragOver(paintengine2d.Pt(tv.indentX(0), top+rh*1.9))
	if tv.dropAt != 2 || tv.dropDepth != 0 {
		t.Fatalf("at=%d depth=%d, want at=2 depth=0", tv.dropAt, tv.dropDepth)
	}
	if shallow := tv.indentX(tv.dropDepth); shallow >= deep {
		t.Fatalf("a caret at the root level starts at %g, one a level in at %g: the two must differ", shallow, deep)
	}
}

// A drag over a list that is measured but has no rows still marks the one
// gap it has, so a drop into an empty view lands at the top.
func TestEmptyListTakesADropAtItsHead(t *testing.T) {
	l := NewListView(0, func(i int) string { return "" }, nil)
	l.SetHost(&host{})
	l.OnDropAt = func(int, widget.DropEvent) bool { return true }
	sz := l.Measure(layout.Loose(200, 120))
	l.Arrange(paintengine2d.XYWH(0, 0, 200, sz.Y))
	l.DragOver(paintengine2d.Pt(20, l.frame().Top+4))
	if l.dropAt != 0 {
		t.Fatalf("an empty list marks gap %d, want 0", l.dropAt)
	}
}
