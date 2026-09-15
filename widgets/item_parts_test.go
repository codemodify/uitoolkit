package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Aero and Metro tabs share one border line with their neighbour: the bar
// lays them over each other by the look's overlap, and the shared column
// belongs to the tab that paints over it (the later one).
func TestTabsOverlapByTheLooksOverlap(t *testing.T) {
	for _, c := range []struct {
		pack string
		ov   float32
	}{{"aero", 1}, {"win8", 1}, {"win95", 0}} {
		pack, ok := style.LoadTheme(c.pack)
		if !ok {
			t.Fatalf("no %s pack", c.pack)
		}
		tb := NewTabBar("Scroll", "List", "Tree")
		tb.SetLook(pack.Look())
		tb.SetHost(&host{})
		tb.Arrange(paintengine2d.XYWH(0, 0, 400, 30))
		if got := style.TabOverlapOf(tb.Look()); got != c.ov {
			t.Fatalf("%s: overlap %v, want %v", c.pack, got, c.ov)
		}
		r := tb.tabRects()
		for i := 1; i < len(r); i++ {
			if got := r[i-1].Max.X - r[i].Min.X; got != c.ov {
				t.Errorf("%s: tabs %d/%d overlap %v, want %v", c.pack, i-1, i, got, c.ov)
			}
			if c.ov > 0 {
				if got := tb.indexAt(r[i].Min.X); got != i {
					t.Errorf("%s: the shared column hit tab %d, want %d", c.pack, got, i)
				}
			}
		}
	}
}

// The row under the pointer knows when the pointer is on its expander, so
// a look can light the arrow (Vista's blue triangle, GTK's prelight).
func TestTreeExpanderHover(t *testing.T) {
	root := &TreeNode{Label: "root", Expanded: true, Children: []*TreeNode{{Label: "leaf"}}}
	tv := NewTreeView(root)
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	rh := tv.rowH()
	f := tv.frame()
	at := func(x float32) {
		tv.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x+f.Left, f.Top+rh*0.5)})
	}
	// Find the expander along the first row.
	x := float32(-1)
	for px := float32(0); px < 60; px++ {
		if tv.expanderHit(px, root, 0) {
			x = px + 2
			break
		}
	}
	if x < 0 {
		t.Fatal("no expander on the root row")
	}
	at(x)
	if !tv.rowState(root).ExpanderHot() {
		t.Fatal("pointer on the expander: the row should say so")
	}
	at(150)
	if tv.rowState(root).ExpanderHot() {
		t.Fatal("pointer on the label: the expander is not hot")
	}
	if tv.hover != root {
		t.Fatal("the row itself stays hovered")
	}
}

// Each tree row knows which branch lines run on past it: the last child
// ends its parent's line at its elbow, and a finished subtree draws none.
//
//	root
//	├ a
//	│ ├ a1
//	│ └ a2
//	└ b
//	  └ b1
func TestTreeRowsCarryTheirBranchChain(t *testing.T) {
	a1, a2 := &TreeNode{Label: "a1"}, &TreeNode{Label: "a2"}
	b1 := &TreeNode{Label: "b1"}
	a := &TreeNode{Label: "a", Expanded: true, Children: []*TreeNode{a1, a2}}
	b := &TreeNode{Label: "b", Expanded: true, Children: []*TreeNode{b1}}
	root := &TreeNode{Label: "root", Expanded: true, Children: []*TreeNode{a, b}}
	tv := NewTreeView(root)
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	for _, c := range []struct {
		n     *TreeNode
		depth int
		next  []bool // HasNextSibling for depths 0..depth
	}{
		{root, 0, []bool{false}},
		{a, 1, []bool{false, true}},
		{a1, 2, []bool{false, true, true}},
		{a2, 2, []bool{false, true, false}},
		{b, 1, []bool{false, false}},
		{b1, 2, []bool{false, false, false}},
	} {
		st := tv.rowState(c.n)
		for d, want := range c.next {
			if got := st.HasNextSibling(d); got != want {
				t.Errorf("%s: depth %d has a next sibling = %v, want %v", c.n.Label, d, got, want)
			}
		}
	}
	// Without the chain, painters keep every line running.
	if !style.StateNone.HasNextSibling(3) {
		t.Error("a row without chain bits should keep its lines")
	}
}

// KDE 3 scroll bars had three arrows: back at the start, back and forward
// together at the end; either back button steps back.
func TestTripleArrowScrollBar(t *testing.T) {
	pack, ok := style.LoadTheme("plastik")
	if !ok {
		t.Fatal("no plastik pack")
	}
	lk := pack.Look()
	sp := style.ScrollGeometry(lk, paintengine2d.XYWH(0, 0, 200, 300), true, 3000, 300, 1000, false)
	if sp.Dec.Empty() || sp.DecEnd.Empty() || sp.Inc.Empty() {
		t.Fatalf("parts %+v", sp)
	}
	if !(sp.Dec.Max.Y <= sp.Track.Min.Y && sp.Track.Max.Y <= sp.DecEnd.Min.Y && sp.DecEnd.Max.Y <= sp.Inc.Min.Y) {
		t.Fatalf("order: dec %v, track %v, dec end %v, inc %v", sp.Dec, sp.Track, sp.DecEnd, sp.Inc)
	}
	if got := sp.HitTest(sp.DecEnd.Center(), true); got != style.ScrollDecEnd {
		t.Fatalf("the second back button hit %v", got)
	}
	lv := NewListView(200, func(i int) string { return "row" }, nil)
	lv.SetLook(lk)
	lv.SetHost(&host{})
	lv.Arrange(paintengine2d.XYWH(0, 0, 160, 200))
	lv.ScrollTo(100)
	before := lv.OffsetY
	end := fromView(lv.vparts().DecEnd, lv.frame()).Center()
	lv.MousePress(widget.MouseEvent{Pos: end, Button: platform.ButtonLeft})
	lv.MouseRelease(widget.MouseEvent{Pos: end, Button: platform.ButtonLeft})
	if lv.OffsetY >= before {
		t.Fatalf("the second back button should scroll back: %v → %v", before, lv.OffsetY)
	}
}

// The Mac's scroll box keeps its size whatever the content; the parts still
// carry the visible proportion for looks that show it beside a fixed thumb.
func TestFixedScrollThumb(t *testing.T) {
	pack, ok := style.LoadTheme("system7")
	if !ok {
		t.Fatal("no system7 pack")
	}
	lk := pack.Look()
	view := paintengine2d.XYWH(0, 0, 200, 300)
	short := style.ScrollGeometry(lk, view, true, 400, 300, 0, false)
	long := style.ScrollGeometry(lk, view, true, 6000, 300, 0, false)
	if short.Thumb.Dy() != long.Thumb.Dy() {
		t.Fatalf("the scroll box changed size: %v vs %v", short.Thumb.Dy(), long.Thumb.Dy())
	}
	if short.Proportion.Dy() <= long.Proportion.Dy() {
		t.Fatalf("the proportion should shrink with more content: %v vs %v", short.Proportion.Dy(), long.Proportion.Dy())
	}
	// Dragged to the end, the box lands at the end of the track.
	end := style.ScrollGeometry(lk, view, true, 6000, 300, 5700, false)
	if d := end.Track.Max.Y - end.Thumb.Max.Y; d > 0.5 || d < -0.5 {
		t.Fatalf("at the end the box should meet the track's end: %v", d)
	}
}
