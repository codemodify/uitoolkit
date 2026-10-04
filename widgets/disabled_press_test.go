package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A disabled list takes a press and does nothing with it.
//
// It used to select the row, call OnSelect, and open a context menu for a
// secondary press — a disabled playlist was still a playlist to the pointer.
// Taking the press matters as much as ignoring it: the window offers a press
// to the component under the pointer whether or not it is enabled, and then
// to its ancestors until one takes it, so a press *refused* by a disabled
// control reaches an enabled container and is acted on as that container's
// own.
func TestADisabledListTakesAPressAndDoesNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(onSelect func(int), onContext func(int, paintengine2d.Point)) widget.Component
	}{
		{"list", func(sel func(int), ctx func(int, paintengine2d.Point)) widget.Component {
			l := NewListView(4, func(i int) string { return "row" }, sel)
			l.OnContext = ctx
			l.Selected = 0
			return l
		}},
		{"table", func(sel func(int), ctx func(int, paintengine2d.Point)) widget.Component {
			tv := NewTableView([]TableColumn{{Title: "Name"}}, 4,
				func(row, col int) string { return "row" }, sel)
			tv.OnContext = ctx
			tv.Selected = 0
			return tv
		}},
		{"tree", func(sel func(int), ctx func(int, paintengine2d.Point)) widget.Component {
			tr := NewTreeView(
				&TreeNode{Label: "a"}, &TreeNode{Label: "b"}, &TreeNode{Label: "c"},
			)
			tr.OnSelect = func(n *TreeNode) { sel(0) }
			tr.OnContext = func(n *TreeNode, p paintengine2d.Point) { ctx(0, p) }
			return tr
		}},
		{"cards", func(sel func(int), ctx func(int, paintengine2d.Point)) widget.Component {
			c := NewCardList(4, func(i int) CardContent {
				return CardContent{Title: "card", Subtitle: "sub"}
			}, sel)
			c.OnContext = ctx
			c.Selected = 0
			return c
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selects, contexts := 0, 0
			c := tc.build(func(int) { selects++ }, func(int, paintengine2d.Point) { contexts++ })
			c.SetLook(style.DarkLook())
			c.SetHost(&host{})
			c.Arrange(paintengine2d.XYWH(0, 0, 240, 180))
			c.SetEnabled(false)

			at := paintengine2d.Pt(40, 60)
			for _, b := range []platform.MouseButton{platform.ButtonLeft, platform.ButtonRight} {
				if !c.MousePress(widget.MouseEvent{Pos: at, Button: b}) {
					t.Errorf("%v: the press was refused, so an enabled container above would act on it", b)
				}
			}
			if selects != 0 || contexts != 0 {
				t.Errorf("a disabled control reported %d selections and %d context actions", selects, contexts)
			}
		})
	}
}

// And an enabled one still does its job, which is what makes the test above
// about the disabled state rather than about presses in general.
func TestAnEnabledListStillTakesPresses(t *testing.T) {
	selects := 0
	l := NewListView(4, func(i int) string { return "row" }, func(int) { selects++ })
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 240, 180))

	if !l.MousePress(widget.MouseEvent{Pos: rowPoint(l, 1), Button: platform.ButtonLeft}) {
		t.Fatal("an enabled list refused a press")
	}
	if selects != 1 {
		t.Errorf("an enabled list reported %d selections, want 1", selects)
	}
}
