package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func win95Look(t *testing.T) *style.Classic {
	t.Helper()
	p, ok := style.LoadTheme("win95")
	if !ok {
		t.Fatal("win95 pack missing")
	}
	return p.Look()
}

// Views sit inside the look's frame: Win95 lists are sunken 2px wells, so a
// click on the frame selects nothing, rows and the scrollbar start inside
// it, and public geometry stays in local coordinates.
func TestListViewSitsInsideViewFrame(t *testing.T) {
	lk := win95Look(t)
	in := style.ViewFrameInsetsOf(lk)
	if in.Left < 2 || in.Top < 2 {
		t.Fatalf("win95 view frame %+v, want the 2px sunken well", in)
	}
	l := NewListView(50, func(i int) string { return fmt.Sprint(i) }, nil)
	l.SetHost(&fakeWindow{look: lk})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(50, 0.5)})
	if l.Selected != -1 {
		t.Fatalf("a click on the frame selected row %d", l.Selected)
	}
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(50, in.Top+1)})
	if l.Selected != 0 {
		t.Fatalf("a click just inside the frame selected %d, want row 0", l.Selected)
	}
	track, thumb := l.ScrollTrack()
	if track.Max.X > 200-in.Right+0.01 || track.Min.Y < in.Top-0.01 || track.Max.Y > 100-in.Bottom+0.01 {
		t.Fatalf("scroll track %v leaves the frame's inside (frame %+v)", track, in)
	}
	// Dragging the thumb (found through the public, local-space geometry)
	// scrolls.
	c := paintengine2d.Pt((thumb.Min.X+thumb.Max.X)/2, (thumb.Min.Y+thumb.Max.Y)/2)
	l.MousePress(widget.MouseEvent{Pos: c})
	l.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(c.X, c.Y+30)})
	l.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(c.X, c.Y+30)})
	if l.OffsetY <= 0 {
		t.Fatal("dragging the thumb at its reported position did not scroll")
	}

	flat := NewListView(50, func(i int) string { return fmt.Sprint(i) }, nil)
	flat.Frameless = true
	flat.SetHost(&fakeWindow{look: lk})
	flat.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	flat.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(50, 0.5)})
	if flat.Selected != 0 {
		t.Fatalf("frameless list: a click at the top selected %d, want 0", flat.Selected)
	}
}

func TestTableRowBoundsAreLocalInsideFrame(t *testing.T) {
	lk := win95Look(t)
	in := style.ViewFrameInsetsOf(lk)
	tv := NewTableView([]TableColumn{{Title: "A"}, {Title: "B"}}, 20, func(r, c int) string { return "x" }, nil)
	tv.SetHost(&fakeWindow{look: lk})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	r := tv.RowBounds(0)
	if r.Min.X != in.Left || r.Min.Y != in.Top+tv.HeaderHeight() {
		t.Fatalf("row 0 at %v, want inside the frame under the header (frame %+v, header %v)", r, in, tv.HeaderHeight())
	}
	c := paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	tv.MousePress(widget.MouseEvent{Pos: c})
	if tv.Selected != 0 {
		t.Fatalf("click at RowBounds(0) selected %d", tv.Selected)
	}
	w := tv.ColumnWidths()
	if got := tv.ColumnDividerAt(in.Left + w[0]); got != 0 {
		t.Fatalf("divider at local x %v = %d, want column 0", in.Left+w[0], got)
	}
}

// EnsureVisible before layout waits for the first Arrange, then scrolls
// just enough (the Settings theme list opens on the staged pack).
func TestListEnsureVisibleBeforeLayout(t *testing.T) {
	l := NewListView(100, func(i int) string { return fmt.Sprint(i) }, nil)
	l.SetHost(&fakeWindow{look: style.DarkLook()})
	l.EnsureVisible(60)
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	lo, hi := l.VisibleRange()
	if 60 < lo || 60 >= hi {
		t.Fatalf("row 60 not in view after the first Arrange: rows [%d,%d)", lo, hi)
	}
	off := l.OffsetY
	l.EnsureVisible(lo + 1)
	if l.OffsetY != off {
		t.Fatalf("revealing a visible row scrolled from %v to %v", off, l.OffsetY)
	}
}

// A view measures its view frame as well as its rows: given the height it
// asks for — which is what a scroll view or a column gives it — every row
// fits inside the frame and nothing is left to scroll. A list that
// measured its rows alone lost its last row under the frame's bottom edge.
func TestViewsMeasureTheirFrame(t *testing.T) {
	lk := win95Look(t)
	in := style.ViewFrameInsetsOf(lk)
	if in.Top+in.Bottom <= 0 {
		t.Fatal("win95 has no view frame to measure")
	}
	text := func(i int) string { return fmt.Sprint(i) }
	list := NewListView(7, text, nil)
	tree := NewTreeView(NewTreeNode("root",
		NewTreeNode("a"), NewTreeNode("b"), NewTreeNode("c"), NewTreeNode("d")))
	table := NewTableView([]TableColumn{{Title: "A", Width: 80}}, 9, func(r, c int) string { return text(r) }, nil)

	views := []struct {
		name string
		c    widget.Component
		// room is the height the rows (and a table's header) need inside
		// the frame; max is how far the view can scroll once arranged.
		room func() float32
		max  func() float32
	}{
		{"list", list, func() float32 { return list.contentH() }, list.MaxOffset},
		{"tree", tree, func() float32 { return tree.contentH() }, tree.MaxOffset},
		{"table", table, func() float32 { return table.headerH() + table.contentH() }, table.MaxOffset},
	}
	for _, v := range views {
		v.c.SetHost(&fakeWindow{look: lk})
		m := v.c.Measure(layout.Constraints{MaxW: 300, MaxH: -1})
		if want := v.room() + in.Top + in.Bottom; m.Y < want {
			t.Errorf("%s measures %g tall, want at least %g (its rows and its frame)", v.name, m.Y, want)
		}
		v.c.Arrange(paintengine2d.XYWH(0, 0, 300, m.Y))
		if got := v.max(); got > 0 {
			t.Errorf("%s at the height it asked for still scrolls %g px: its last row is clipped", v.name, got)
		}
	}
}
