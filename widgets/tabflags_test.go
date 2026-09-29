package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

func testBar(t *testing.T, titles ...string) *TabBar {
	t.Helper()
	bar := NewTabBar(titles...)
	bar.SetHost(&host{})
	bar.Arrange(paintengine2d.XYWH(0, 0, 400, 30))
	return bar
}

// A hidden tab takes no width and no place in the strip: the tabs after
// it move left, and a click where it used to be lands on its neighbour.
func TestHiddenTabLeavesNoGap(t *testing.T) {
	bar := testBar(t, "One", "Two", "Three")
	full := bar.tabRects()
	bar.SetTabVisible(1, false)
	r := bar.tabRects()
	if r[1].Dx() != 0 {
		t.Fatalf("hidden tab still %v wide", r[1].Dx())
	}
	if r[0] != full[0] {
		t.Fatalf("first tab moved: %v then %v", full[0], r[0])
	}
	if r[2].Min.X >= full[2].Min.X {
		t.Fatalf("third tab did not move left: %v then %v", full[2], r[2])
	}
	// The pointer where "Two" was now finds "Three".
	if got := bar.indexAt(full[1].Min.X + 2); got == 1 {
		t.Fatal("a click found the hidden tab")
	}
	bar.SetTabVisible(1, true)
	if bar.tabRects()[1].Dx() == 0 {
		t.Fatal("the tab did not come back")
	}
}

// A disabled tab keeps its place — the page exists, it is just not
// available — and refuses the click, the arrow key and the wheel.
func TestDisabledTabKeepsItsPlaceAndRefusesSelection(t *testing.T) {
	bar := testBar(t, "One", "Two", "Three")
	full := bar.tabRects()
	bar.SetTabEnabled(1, false)
	if bar.tabRects()[1] != full[1] {
		t.Fatalf("the disabled tab moved: %v then %v", full[1], bar.tabRects()[1])
	}
	fired := -1
	bar.OnSelect = func(i int) { fired = i }

	at := full[1].Min.X + 4
	bar.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(at, 10)})
	bar.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(at, 10)})
	if bar.Selected != 0 || fired != -1 {
		t.Fatalf("a click selected the disabled tab: sel=%d fired=%d", bar.Selected, fired)
	}
	// Hovering it must not light it up either.
	bar.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(at, 10)})
	if bar.hover == 1 {
		t.Fatal("the disabled tab is hovered")
	}
	// Right steps over it to Three.
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if bar.Selected != 2 || fired != 2 {
		t.Fatalf("right: sel=%d fired=%d, want the tab after the disabled one", bar.Selected, fired)
	}
	// And Left steps back over it to One.
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	if bar.Selected != 0 {
		t.Fatalf("left: sel=%d", bar.Selected)
	}
	// Left again, at the leftmost tab, stays put rather than wrapping.
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	if bar.Selected != 0 {
		t.Fatalf("left at the edge wrapped to %d", bar.Selected)
	}
}

// Home and End mean the first and last tab there *is*, not index 0 and
// index n-1, which may be hidden or disabled.
func TestHomeAndEndSkipUnselectableEnds(t *testing.T) {
	bar := testBar(t, "One", "Two", "Three", "Four")
	bar.SetTabVisible(0, false)
	bar.SetTabEnabled(3, false)
	bar.Selected = 2
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if bar.Selected != 1 {
		t.Fatalf("home landed on %d, want 1", bar.Selected)
	}
	bar.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if bar.Selected != 2 {
		t.Fatalf("end landed on %d, want 2", bar.Selected)
	}
}

// Turning off the tab that is selected moves the selection, and the view
// swaps the pages to match — the caller does not have to.
func TestTabViewFollowsWhenTheSelectedTabGoesAway(t *testing.T) {
	a, b, c := NewLabel("a"), NewLabel("b"), NewLabel("c")
	tv := NewTabView(Tab{Title: "A", Content: a}, Tab{Title: "B", Content: b}, Tab{Title: "C", Content: c})
	tv.SetHost(&host{})
	tv.Measure(layout.Loose(320, 200))
	tv.Arrange(paintengine2d.XYWH(0, 0, 320, 200))
	tv.Select(1)
	if !b.Visible() {
		t.Fatal("B is not showing")
	}
	tv.SetTabVisible(1, false)
	if tv.Selected() == 1 {
		t.Fatal("still on the hidden tab")
	}
	if b.Visible() {
		t.Fatal("the hidden tab's page is still showing")
	}
	if !a.Visible() && !c.Visible() {
		t.Fatal("no page is showing")
	}
	if tv.TabVisible(1) {
		t.Fatal("TabVisible disagrees")
	}
	tv.SetTabEnabled(0, false)
	if tv.TabEnabled(0) {
		t.Fatal("TabEnabled disagrees")
	}
	if tv.Selected() == 0 {
		t.Fatal("selected a disabled tab")
	}
}

// A view built with the flags already set opens on the first tab that
// can be opened, not on a hidden index 0 with no page showing.
func TestTabViewOpensOnTheFirstUsableTab(t *testing.T) {
	a, b, c := NewLabel("a"), NewLabel("b"), NewLabel("c")
	tv := NewTabView(
		Tab{Title: "A", Content: a, Hidden: true},
		Tab{Title: "B", Content: b, Disabled: true},
		Tab{Title: "C", Content: c},
	)
	tv.SetHost(&host{})
	tv.Measure(layout.Loose(320, 200))
	tv.Arrange(paintengine2d.XYWH(0, 0, 320, 200))
	if tv.Selected() != 2 || !c.Visible() || a.Visible() || b.Visible() {
		t.Fatalf("selected %d, vis a=%v b=%v c=%v", tv.Selected(), a.Visible(), b.Visible(), c.Visible())
	}
	if tv.Bar().Selected != 2 {
		t.Fatalf("bar %d", tv.Bar().Selected)
	}
}

// HeightForRows answers without a layout pass, which is the point: a
// popover is sized by the rows it should show.
func TestHeightForRows(t *testing.T) {
	l := NewListView(0, nil, nil)
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	rh := l.RowHeightPx()
	if rh <= 0 {
		t.Fatalf("row height %v", rh)
	}
	five := l.HeightForRows(5)
	if want := 5*rh + l.HeightForRows(0); five != want {
		t.Fatalf("five rows %v, want %v", five, want)
	}
	// Count does not clamp it: an empty list asked for five rows still
	// reports five.
	if l.Count != 0 {
		t.Fatalf("count %d", l.Count)
	}
	if five <= l.HeightForRows(0) {
		t.Fatalf("an empty list reported %v for five rows", five)
	}
	if l.HeightForRows(-3) != l.HeightForRows(0) {
		t.Fatal("a negative row count is not zero rows")
	}
	// And it agrees with what Measure gives for that many real rows.
	l.Count = 5
	if got := l.Measure(layout.Loose(200, 10000)).Y; got != five {
		t.Fatalf("Measure %v, HeightForRows %v", got, five)
	}

	tb := NewTableView([]TableColumn{{Title: "Name"}}, 0, func(int, int) string { return "" }, nil)
	tb.SetHost(&host{})
	tb.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	if h := tb.HeightForRows(6); h <= tb.HeightForRows(0) || tb.HeightForRows(0) <= 0 {
		t.Fatalf("table: six rows %v, header+frame %v", h, tb.HeightForRows(0))
	}
	if got, want := tb.HeightForRows(6)-tb.HeightForRows(0), 6*tb.RowHeightPx(); got != want {
		t.Fatalf("table: six rows are %v, want %v", got, want)
	}
}
