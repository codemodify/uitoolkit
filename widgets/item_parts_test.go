package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
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
