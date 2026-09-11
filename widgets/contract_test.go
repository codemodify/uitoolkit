package widgets_test

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Regression map — basic UI bugs that must fail CI before a tag.
//
//	blank-after-first-page  TestListViewScenePaintsPastFirstPage, TestTableViewHeaderFlushAndClip,
//	                        TestCardListScenePaintsPastFirstPage, TestTreeViewPaintsAfterFirstPage
//	header overlap scroll   TestTableViewHeaderFlushAndClip, TestTableViewHeaderClipsBodyWhenScrolled
//	header gap at top       TestTableViewHeaderFlushAndClip, TestTableViewFirstRowFlushUnderHeader
//	splitter overlap        TestSplitterArrangeAfterDragExclusive, TestSplitterPanesExclusiveAtRatios
//	stuck resize cursor     TestSplitterRestoresPointerCursor, TestSplitterCursorReturnsAfterDrag
//	unclamped scroll        TestScrollClampWheelStopsAtEnd, TestScrollersClampOffset
//	type into read-only     TestTextViewReadOnlyAndScrollbar, TestTextAreaReadOnlyRejectsInput

func TestSplitterPanesExclusiveAtRatios(t *testing.T) {
	left := widgets.NewLabel("AAAA pane A chrome")
	right := widgets.NewLabel("BBBB pane B chrome that must not overlap A")
	split := widgets.NewSplitter(true, left, right)
	s := uitest.Mount(split, paintengine2d.XYWH(0, 0, 400, 180))
	for _, ratio := range []float32{0.08, 0.25, 0.5, 0.75, 0.92} {
		split.Ratio = ratio
		s.Layout()
		a, b := split.PaneA(), split.PaneB()
		if err := uitest.CheckExclusive(a, b); err != nil {
			t.Fatalf("ratio %v: %v", ratio, err)
		}
		if a.Max.X > b.Min.X+0.01 {
			t.Fatalf("ratio %v A not exclusive of B A.max=%v B.min=%v", ratio, a.Max.X, b.Min.X)
		}
	}
}

func TestSplitterCursorReturnsAfterDrag(t *testing.T) {
	split := widgets.NewSplitter(true, widgets.NewLabel("left"), widgets.NewLabel("right"))
	split.Ratio = 0.4
	s := uitest.Mount(split, paintengine2d.XYWH(0, 0, 400, 160))
	a, b := split.PaneA(), split.PaneB()
	sash := paintengine2d.Pt((a.Max.X+b.Min.X)*0.5, 40)
	s.MouseMove(sash)
	if s.Cursor() != platform.CursorColResize {
		t.Fatalf("hover sash cursor=%v want col-resize", s.Cursor())
	}
	s.MousePress(sash, platform.ButtonLeft)
	s.MouseMove(paintengine2d.Pt(300, 40))
	if s.Cursor() != platform.CursorColResize {
		t.Fatalf("during drag cursor=%v", s.Cursor())
	}
	// Release in pane A, then leave the sash — stuck-cursor bug stays col-resize.
	s.MouseRelease(paintengine2d.Pt(40, 40))
	s.MouseMove(paintengine2d.Pt(24, 40))
	if s.Cursor() != platform.CursorDefault {
		t.Fatalf("after release off sash cursor=%v want default", s.Cursor())
	}
}

func TestScrollersClampOffset(t *testing.T) {
	lv := widgets.NewListView(40, func(i int) string { return fmt.Sprintf("row-%02d", i) }, nil)
	lv.RowHeight = 16
	_ = uitest.Mount(lv, paintengine2d.XYWH(0, 0, 160, 64))
	lv.ScrollTo(-80)
	if lv.OffsetY != 0 {
		t.Fatalf("list neg offset %v", lv.OffsetY)
	}
	lv.ScrollTo(1e6)
	if lv.OffsetY != lv.MaxOffset() || lv.MaxOffset() <= 0 {
		t.Fatalf("list past-end %v max=%v", lv.OffsetY, lv.MaxOffset())
	}
	if _, thumb := lv.ScrollTrack(); thumb.Empty() {
		t.Fatal("overflow list thumb")
	}
	if err := uitest.CheckThumbInTrack(lv.ScrollTrack()); err != nil {
		t.Fatal(err)
	}

	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "Col"}}, 30, func(row, col int) string {
		return fmt.Sprintf("R%02d", row)
	}, nil)
	tv.RowHeight = 18
	_ = uitest.Mount(tv, paintengine2d.XYWH(0, 0, 200, 90))
	tv.ScrollTo(-40)
	if tv.OffsetY != 0 {
		t.Fatalf("table neg %v", tv.OffsetY)
	}
	tv.ScrollTo(1e6)
	if tv.OffsetY != tv.MaxOffset() || tv.MaxOffset() <= 0 {
		t.Fatalf("table past-end %v max=%v", tv.OffsetY, tv.MaxOffset())
	}

	cards := widgets.NewCardList(20, func(i int) widgets.CardContent {
		return widgets.CardContent{Title: fmt.Sprintf("Card %d", i), Subtitle: "sub", Meta: "now", Snippet: "snip"}
	}, nil)
	cards.CardHeight = 48
	_ = uitest.Mount(cards, paintengine2d.XYWH(0, 0, 240, 120))
	cards.ScrollTo(1e6)
	if cards.OffsetY != cards.MaxOffset() || cards.MaxOffset() <= 0 {
		t.Fatalf("cards %v max=%v", cards.OffsetY, cards.MaxOffset())
	}

	root := widgets.NewTreeNode("root")
	for i := 0; i < 40; i++ {
		root.Children = append(root.Children, widgets.NewTreeNode(fmt.Sprintf("n%d", i)))
	}
	root.Expanded = true
	tree := widgets.NewTreeView(root)
	_ = uitest.Mount(tree, paintengine2d.XYWH(0, 0, 180, 80))
	tree.ScrollTo(1e6)
	if tree.OffsetY != tree.MaxOffset() || tree.MaxOffset() <= 0 {
		t.Fatalf("tree %v max=%v", tree.OffsetY, tree.MaxOffset())
	}

	col := widgets.NewColumn()
	for i := 0; i < 40; i++ {
		col.Add(widgets.NewLabel("row"))
	}
	sv := widgets.NewScrollView(col)
	_ = uitest.Mount(sv, paintengine2d.XYWH(0, 0, 160, 80))
	sv.ScrollTo(-10)
	if sv.OffsetY != 0 {
		t.Fatalf("sv neg %v", sv.OffsetY)
	}
	sv.ScrollTo(1e6)
	if sv.OffsetY != sv.MaxOffset() || sv.MaxOffset() <= 0 {
		t.Fatalf("sv %v max=%v", sv.OffsetY, sv.MaxOffset())
	}
}

func TestTreeViewPaintsAfterFirstPage(t *testing.T) {
	root := widgets.NewTreeNode("root")
	root.Expanded = true
	for i := 0; i < 50; i++ {
		root.Children = append(root.Children, widgets.NewTreeNode(fmt.Sprintf("====NODE %02d====", i)))
	}
	tree := widgets.NewTreeView(root)
	s := uitest.Mount(tree, paintengine2d.XYWH(0, 0, 220, 90))
	if tree.MaxOffset() <= 0 {
		t.Fatal("expected overflow")
	}
	tree.ScrollTo(tree.LocalBounds().Dy() + 24)
	lo, hi := tree.VisibleRange()
	if lo < 1 || hi <= lo {
		t.Fatalf("window lo=%d hi=%d", lo, hi)
	}
	img := s.Paint()
	field := uitest.FieldColor(tree.Look())
	painted := uitest.CountNonColor(img, tree.LocalBounds().Inset(4), field, 18)
	if painted < 20 {
		t.Fatalf("tree blank after first page: %d lo=%d hi=%d", painted, lo, hi)
	}
}

func TestTableViewFirstRowFlushUnderHeader(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH"}}, 12,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	tv.RowHeight = 22
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 260, 140))
	if tv.OffsetY != 0 {
		t.Fatalf("start offset %v", tv.OffsetY)
	}
	row0 := tv.RowBounds(0)
	hh := tv.HeaderHeight()
	if row0.Empty() {
		t.Fatal("row0 empty")
	}
	if d := row0.Min.Y - hh; d < -1 || d > 2 {
		t.Fatalf("header gap: first row y=%v headerH=%v delta=%v", row0.Min.Y, hh, d)
	}
	img := s.Paint()
	field := uitest.FieldColor(tv.Look())
	band := paintengine2d.XYWH(8, hh+2, 180, row0.Dy()-4)
	painted := uitest.CountNonColor(img, band, field, 16)
	if painted < 10 {
		t.Fatalf("large gap under header: non-field pixels in first-row band=%d band=%+v", painted, band)
	}
}

func TestTableViewHeaderClipsBodyWhenScrolled(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "HHHHHHHHHH"}}, 20,
		func(row, col int) string { return fmt.Sprintf("====ROW %02d====", row) }, nil)
	tv.RowHeight = 22
	tv.Selected = 0
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 260, 140))
	hh := tv.HeaderHeight()
	img0 := s.Paint()

	tv.ScrollTo(hh + tv.RowBounds(0).Dy())
	if tv.OffsetY <= 0 {
		t.Fatal("expected scroll")
	}
	row0 := tv.RowBounds(0)
	img1 := s.Paint()
	// Sticky header must not pick up selected-row / body pixels (bleed).
	diff := 0
	for y := 1; y < int(hh)-1 && y < img0.Height && y < img1.Height; y++ {
		for x := 4; x < 80 && x < img0.Width && x < img1.Width; x++ {
			ar, ag, ab, _ := img0.PremulAt(x, y)
			br, bg, bb, _ := img1.PremulAt(x, y)
			dr, dg, db := int(ar)-int(br), int(ag)-int(bg), int(ab)-int(bb)
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if dr+dg+db > 48 {
				diff++
			}
		}
	}
	if diff > 30 {
		t.Fatalf("header bleed on scroll-down: %d header pixels changed (row0=%+v offset=%v)", diff, row0, tv.OffsetY)
	}
	field := uitest.FieldColor(tv.Look())
	band := paintengine2d.XYWH(8, hh+2, 180, tv.RowBounds(1).Dy()-4)
	if uitest.CountNonColor(img1, band, field, 16) < 8 {
		t.Fatal("body under header empty after scroll")
	}
}

func TestTextAreaReadOnlyRejectsInput(t *testing.T) {
	view := widgets.NewTextView("plain message body\nline two", "view")
	s := uitest.Mount(view, paintengine2d.XYWH(0, 0, 240, 80))
	s.Focus(view)
	before := view.Text
	if view.TextInput('x') {
		t.Fatal("read-only TextView accepted TextInput")
	}
	s.Type("typed into view")
	view.KeyPress(widget.KeyEvent{Key: platform.KeyA})
	view.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	view.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if view.Text != before {
		t.Fatalf("read-only mutated %q -> %q", before, view.Text)
	}

	edit := widgets.NewTextArea("hello", "", nil)
	s = uitest.Mount(edit, paintengine2d.XYWH(0, 0, 240, 80))
	s.Focus(edit)
	edit.SetSelection(5, 5)
	if !edit.TextInput('!') {
		t.Fatal("editable TextArea rejected input")
	}
	s.Type("ok")
	if edit.Text != "hello!ok" {
		t.Fatalf("editable %q", edit.Text)
	}
}

func TestTreeInvariantsOnMountedTable(t *testing.T) {
	tv := widgets.NewTableView([]widgets.TableColumn{{Title: "A"}}, 25, func(row, col int) string {
		return fmt.Sprintf("%d", row)
	}, nil)
	s := uitest.Mount(tv, paintengine2d.XYWH(0, 0, 200, 96))
	s.Wheel(paintengine2d.Pt(40, 50), 80)
	if errs := uitest.TreeInvariants(tv); len(errs) > 0 {
		t.Fatal(errs)
	}
}
