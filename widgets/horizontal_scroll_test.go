package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

func wideChild() *Button {
	return NewButton("a button with a label long enough that it will not fit a narrow pane", nil)
}

// A ScrollView scrolls up and down only unless it is told otherwise, so
// a child with no narrower form is clipped at the edge and part of it is
// simply unreachable.
func TestScrollViewHorizontal(t *testing.T) {
	s := NewScrollView(wideChild())
	s.Horizontal = true
	atScale(t, s, 1)
	const w = 160
	sz := s.Measure(layout.Constraints{MaxW: w, MaxH: 80})
	s.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))

	if s.ContentWidth() <= w {
		t.Fatalf("the content is %v wide in a %v pane — this proves nothing", s.ContentWidth(), w)
	}
	if s.MaxOffsetX() <= 0 {
		t.Fatal("nothing to scroll sideways to")
	}

	s.ScrollToX(s.MaxOffsetX())
	if s.OffsetX != s.MaxOffsetX() {
		t.Errorf("OffsetX is %v, want %v", s.OffsetX, s.MaxOffsetX())
	}
	// The far edge of the child is now at the edge of the viewport —
	// which is the pane less the vertical bar's gutter, not the pane.
	right := s.Children()[0].Bounds().Max.X
	if right < s.viewW()-1 {
		t.Errorf("scrolled to the end, the child's right edge is at %v in a %v viewport",
			right, s.viewW())
	}
	// And it clamps.
	s.ScrollToX(1e6)
	if s.OffsetX != s.MaxOffsetX() {
		t.Errorf("OffsetX ran past the end: %v", s.OffsetX)
	}
	s.ScrollToX(-100)
	if s.OffsetX != 0 {
		t.Errorf("OffsetX ran before the start: %v", s.OffsetX)
	}
}

// Off by default, so nothing that clips today silently starts scrolling.
func TestScrollViewIsVerticalUnlessAsked(t *testing.T) {
	s := NewScrollView(wideChild())
	atScale(t, s, 1)
	sz := s.Measure(layout.Constraints{MaxW: 160, MaxH: 80})
	s.Arrange(paintengine2d.XYWH(0, 0, 160, sz.Y))

	if s.MaxOffsetX() != 0 {
		t.Error("a plain scroll view offers horizontal scrolling")
	}
	s.ScrollToX(50)
	if s.OffsetX != 0 {
		t.Errorf("a plain scroll view scrolled sideways to %v", s.OffsetX)
	}
}

// A view that scrolls sideways imposes no width on its parent — that is
// the whole point of turning it on.
func TestHorizontalScrollViewHasNoMinimumWidth(t *testing.T) {
	plain := NewScrollView(wideChild())
	atScale(t, plain, 1)
	scrolls := NewScrollView(wideChild())
	scrolls.Horizontal = true
	atScale(t, scrolls, 1)

	if widget.MinWidthOf(scrolls) >= widget.MinWidthOf(plain) {
		t.Errorf("a sideways-scrolling view's minimum (%v) should be far below a clipping one's (%v)",
			widget.MinWidthOf(scrolls), widget.MinWidthOf(plain))
	}
}

// A table with more columns than fit: the last ones used to be
// unreachable, squeezed to their floors and clipped.
func TestTableViewHorizontal(t *testing.T) {
	tv := &TableView{RowCount: 3}
	tv.Init(tv)
	for _, name := range []string{"Name", "Kind", "Size", "Modified", "Owner", "Where"} {
		tv.Columns = append(tv.Columns, TableColumn{Title: name, MinWidth: 90})
	}
	tv.CellText = func(r, c int) string { return "cell" }
	tv.Horizontal = true
	atScale(t, tv, 1)
	const w = 260
	tv.Arrange(paintengine2d.XYWH(0, 0, w, 200))

	if tv.MaxOffsetX() <= 0 {
		t.Fatalf("six 90px columns in a %v pane should scroll; rowsW=%v viewW=%v",
			w, tv.rowsW(), tv.viewW())
	}
	// The last column is reachable.
	tv.ScrollToX(tv.MaxOffsetX())
	widths := tv.ColumnWidths()
	var end float32
	for _, cw := range widths {
		end += cw
	}
	if end-tv.OffsetX > tv.viewW()+1 {
		t.Errorf("scrolled to the end, the last column still ends at %v past a %v viewport",
			end-tv.OffsetX, tv.viewW())
	}
	// A hit test in the scrolled view names a later column.
	if col := tv.colAt(10); col == 0 {
		t.Error("with the table scrolled right, the leftmost pixel is still column 0")
	}
}

// Off by default there too.
func TestTableViewIsVerticalUnlessAsked(t *testing.T) {
	tv := &TableView{RowCount: 3}
	tv.Init(tv)
	for i := 0; i < 6; i++ {
		tv.Columns = append(tv.Columns, TableColumn{Title: "c", MinWidth: 90})
	}
	atScale(t, tv, 1)
	tv.Arrange(paintengine2d.XYWH(0, 0, 260, 200))
	if tv.MaxOffsetX() != 0 {
		t.Error("a plain table offers horizontal scrolling")
	}
}
