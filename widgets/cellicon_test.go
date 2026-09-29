package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// pixelsDiffering is how many pixels two renders of the same widget
// disagree about. Counting "ink" against the corner pixel does not work
// on a table — nearly every pixel differs from the corner, so the count
// saturates and answers nothing. Two renders that differ only in the
// mark do answer.
func pixelsDiffering(a, b *paintengine2d.Image) int {
	a.Touch()
	b.Touch()
	if a.Width != b.Width || a.Height != b.Height {
		return -1
	}
	n := 0
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			if a.At(x, y) != b.At(x, y) {
				n++
			}
		}
	}
	return n
}

func paintTable(t *testing.T, tb *TableView, w, h int) *paintengine2d.Image {
	t.Helper()
	tb.SetLook(style.DarkLook())
	tb.SetHost(&host{})
	tb.Measure(layout.Loose(float32(w), float32(h)))
	tb.Arrange(paintengine2d.XYWH(0, 0, float32(w), float32(h)))
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	tb.Paint(ctx)
	return img
}

// A cell with an icon paints more than the same cell without one, and a
// table with no CellIcon paints exactly what it painted before — the
// path is only taken for a cell that asks for a mark.
func TestTableCellIconPaints(t *testing.T) {
	cols := []TableColumn{{Title: "", Width: 30}, {Title: "Subject"}}
	cell := func(row, col int) string {
		if col == 1 {
			return "A message"
		}
		return ""
	}
	plain := NewTableView(cols, 3, cell, nil)
	before := paintTable(t, plain, 300, 120)

	marked := NewTableView(cols, 3, cell, nil)
	marked.CellIcon = func(row, col int) (style.ToolIcon, paintengine2d.Color) {
		if col == 0 && row == 1 {
			return style.IconAttach, paintengine2d.Color{}
		}
		return style.IconNone, paintengine2d.Color{}
	}
	after := paintTable(t, marked, 300, 120)
	if n := pixelsDiffering(before, after); n < 8 {
		t.Fatalf("the paperclip changed %d pixels", n)
	}

	// A tint changes what is painted, so the colour reaches the glyph.
	tinted := NewTableView(cols, 3, cell, nil)
	tinted.CellIcon = func(row, col int) (style.ToolIcon, paintengine2d.Color) {
		if col == 0 && row == 1 {
			return style.IconAttach, paintengine2d.RGB(1, 0.2, 0.2)
		}
		return style.IconNone, paintengine2d.Color{}
	}
	if n := pixelsDiffering(paintTable(t, marked, 300, 120), paintTable(t, tinted, 300, 120)); n < 8 {
		t.Fatalf("the tint changed %d pixels", n)
	}
}

// A column with an icon in its header paints one; one without does not.
func TestTableHeaderIconPaints(t *testing.T) {
	cell := func(int, int) string { return "" }
	plain := NewTableView([]TableColumn{{Title: "", Width: 30}, {Title: "Subject"}}, 2, cell, nil)
	before := paintTable(t, plain, 300, 120)

	marked := NewTableView([]TableColumn{
		{Title: "", Width: 30, Icon: style.IconAttach},
		{Title: "Subject"},
	}, 2, cell, nil)
	if n := pixelsDiffering(before, paintTable(t, marked, 300, 120)); n < 8 {
		t.Fatalf("the header icon changed %d pixels", n)
	}
}

// A tree node with an icon paints one, and a node without is untouched.
func TestTreeNodeIconPaints(t *testing.T) {
	build := func(icon style.ToolIcon) *TreeView {
		root := NewTreeNode("Inbox", NewTreeNode("Drafts"), NewTreeNode("Sent"))
		root.Children[0].Icon = icon
		tv := NewTreeView(root)
		tv.SetLook(style.DarkLook())
		tv.SetHost(&host{})
		tv.Measure(layout.Loose(240, 160))
		tv.Arrange(paintengine2d.XYWH(0, 0, 240, 160))
		return tv
	}
	paint := func(tv *TreeView) *paintengine2d.Image {
		img := paintengine2d.NewImage(240, 160)
		tv.Paint(paintengine2d.NewContext(img))
		return img
	}
	if n := pixelsDiffering(paint(build(style.IconNone)), paint(build(style.IconMute))); n < 8 {
		t.Fatalf("the node icon changed %d pixels", n)
	}
}
