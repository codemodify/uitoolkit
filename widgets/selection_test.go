package widgets

import (
	"fmt"
	"slices"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The mail-client convention (e2e finding: Ctrl+click, Shift+click,
// Shift+arrows and Ctrl+A all selected one row).
func TestTableExtendedSelection(t *testing.T) {
	tv := NewTableView([]TableColumn{{Title: "Subject"}}, 20, func(r, c int) string { return fmt.Sprint(r) }, nil)
	tv.Mode = SelectExtended
	var reported []int
	tv.OnSelectionChange = func(rows []int) { reported = rows }
	tv.SetHost(&fakeWindow{look: style.DarkLook()})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 400))
	click := func(row int, mods platform.Modifiers, button platform.MouseButton) {
		r := tv.RowBounds(row)
		p := paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
		tv.MousePress(widget.MouseEvent{Pos: p, Mods: mods, Button: button})
		tv.MouseRelease(widget.MouseEvent{Pos: p, Button: button})
	}
	want := func(step string, rows ...int) {
		t.Helper()
		if got := tv.SelectedRows(); !slices.Equal(got, rows) {
			t.Fatalf("%s: selected %v, want %v", step, got, rows)
		}
		if !slices.Equal(reported, rows) {
			t.Fatalf("%s: OnSelectionChange reported %v, want %v", step, reported, rows)
		}
	}
	click(1, 0, platform.ButtonLeft)
	want("click", 1)
	click(3, platform.ModCtrl, platform.ButtonLeft)
	want("ctrl+click adds", 1, 3)
	click(5, platform.ModShift, platform.ButtonLeft)
	want("shift+click ranges from the anchor", 3, 4, 5)
	click(7, platform.ModShift|platform.ModCtrl, platform.ButtonLeft)
	want("ctrl+shift+click adds a range", 3, 4, 5, 6, 7)
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyDown, Mods: platform.ModShift})
	want("shift+down extends", 3, 4, 5, 6, 7, 8)
	if tv.Selected != 8 {
		t.Fatalf("current row %d, want 8", tv.Selected)
	}
	click(4, 0, platform.ButtonRight)
	want("right-click inside keeps the selection", 3, 4, 5, 6, 7, 8)
	click(12, 0, platform.ButtonRight)
	want("right-click outside selects that row", 12)
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl})
	if got := tv.SelectedRows(); len(got) != 20 {
		t.Fatalf("ctrl+a selected %d rows, want 20", len(got))
	}
	click(2, platform.ModCtrl, platform.ButtonLeft)
	if tv.IsSelected(2) || !tv.IsSelected(3) {
		t.Fatal("ctrl+click on a selected row should deselect only it")
	}
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	want("plain down selects only the next row", 3)
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyDown, Mods: platform.ModCtrl})
	tv.KeyPress(widget.KeyEvent{Key: platform.KeySpace, Mods: platform.ModCtrl})
	want("ctrl+down moves, ctrl+space toggles", 3, 4)
}

// SelectSingle (the default) is unchanged: modifiers do nothing special
// and Ctrl+A is left for someone else.
func TestListSingleSelectionIgnoresModifiers(t *testing.T) {
	l := NewListView(10, func(i int) string { return fmt.Sprint(i) }, nil)
	l.SetHost(&fakeWindow{look: style.DarkLook()})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, l.rowH()*1.5)})
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, l.rowH()*3.5), Mods: platform.ModCtrl})
	if got := l.SelectedRows(); !slices.Equal(got, []int{3}) {
		t.Fatalf("single list selected %v, want [3]", got)
	}
	if l.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl}) {
		t.Fatal("a single-selection list swallowed Ctrl+A")
	}
}

// Card lists (Mail's card view) select like tables.
func TestCardListExtendedSelection(t *testing.T) {
	cl := NewCardList(10, func(i int) CardContent { return CardContent{Title: fmt.Sprint(i)} }, nil)
	cl.Mode = SelectExtended
	cl.SetHost(&fakeWindow{look: style.DarkLook()})
	cl.Arrange(paintengine2d.XYWH(0, 0, 300, 600))
	at := func(i int) paintengine2d.Point { return paintengine2d.Pt(40, (float32(i)+0.5)*cl.rowH()-cl.OffsetY) }
	cl.MousePress(widget.MouseEvent{Pos: at(1)})
	cl.MousePress(widget.MouseEvent{Pos: at(4), Mods: platform.ModCtrl})
	cl.MousePress(widget.MouseEvent{Pos: at(6), Mods: platform.ModShift})
	if got := cl.SelectedRows(); !slices.Equal(got, []int{4, 5, 6}) {
		t.Fatalf("cards selected %v, want [4 5 6]", got)
	}
}
