package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A row is a box, and a point is on it or it is not. Both of those are two
// numbers, and the list was asking about one of them.
//
// RowAt and the list's own pointer handling tested Y alone, so a point level
// with a row but beside it — in the view frame, in the gutter a bar takes out
// of the content, in the gap a skin's layout leaves between its rows and its
// groove — was answered with that row. An application that recognises a
// gesture of its own acted on a row the pointer was never over.

// skinWell is the reported geometry: rows inset from the art with a groove
// beside them and a gap in between, in a 200x100 list.
var skinWell = &RowGeometry{
	Rows:      paintengine2d.XYWH(10, 20, 120, 60),
	Bar:       paintengine2d.XYWH(140, 20, 10, 60),
	RowHeight: 20,
}

func wellList(t *testing.T) *ListView {
	t.Helper()
	l := NewListView(40, func(i int) string { return "row" }, nil)
	l.RowGeo = func(style.LookAndFeel) *RowGeometry { return skinWell }
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	return l
}

// Every point outside the rows slot is outside every row, on both axes.
func TestRowAtChecksBothAxesOfTheRowViewport(t *testing.T) {
	l := wellList(t)
	// The row the art shows at that height, to have something to be wrong
	// about: row 0 spans y 20..40 and x 10..130.
	if got := l.RowAt(paintengine2d.Pt(60, 30)); got != 0 {
		t.Fatalf("RowAt inside the first row = %d, want 0", got)
	}
	for _, p := range []struct {
		at   paintengine2d.Point
		what string
	}{
		{paintengine2d.Pt(0, 30), "left of the rows slot, in the art"},
		{paintengine2d.Pt(135, 30), "in the gap between the rows and the groove"},
		{paintengine2d.Pt(145, 30), "on the groove"},
		{paintengine2d.Pt(151, 30), "right of the groove, in the art"},
	} {
		if got := l.RowAt(p.at); got != -1 {
			t.Errorf("RowAt %v (%s) = %d, want -1", p.at, p.what, got)
		}
	}
}

// And what the list acts on itself is the same answer: a press beside a row
// is not a press on it.
func TestAPressBesideARowSelectsNothing(t *testing.T) {
	for _, tc := range []struct {
		at   paintengine2d.Point
		what string
	}{
		{paintengine2d.Pt(0, 30), "left of the rows slot"},
		{paintengine2d.Pt(135, 30), "in the gap beside the rows"},
	} {
		l := wellList(t)
		var sel []int
		l.OnSelect = func(i int) { sel = append(sel, i) }
		l.Selected = -1
		l.MousePress(widget.MouseEvent{Pos: tc.at, Button: platform.ButtonLeft})
		if l.Selected != -1 || len(sel) != 0 {
			t.Errorf("a press %s selected %d (OnSelect %v)", tc.what, l.Selected, sel)
		}
		l.MouseMove(widget.MouseEvent{Pos: tc.at})
		if l.hovered != -1 {
			t.Errorf("the pointer %s hovered row %d", tc.what, l.hovered)
		}
	}
}

// Two clicks a quarter of a second apart are a double click only when nothing
// happened in between. The list kept the first click across every boundary a
// gesture cannot cross, so a click, an interruption and a single click
// activated the row — a player started a track the user clicked once.
func TestAnInterruptedGestureIsNotADoubleClick(t *testing.T) {
	ctrl := platform.Modifiers(0)
	ctrl |= platform.ModCtrl
	for _, tc := range []struct {
		what      string
		interrupt func(l *ListView)
	}{
		{"the pointer left the list", func(l *ListView) { l.MouseExit() }},
		{"the list lost the focus", func(l *ListView) { l.FocusLost() }},
		{"the wheel scrolled it", func(l *ListView) {
			l.MouseWheel(widget.MouseEvent{Pos: rowPoint(l, 2), Scroll: paintengine2d.Pt(0, 4), Precise: true})
		}},
		{"a secondary press opened a menu", func(l *ListView) {
			l.MousePress(widget.MouseEvent{Pos: rowPoint(l, 2), Button: platform.ButtonRight})
		}},
		{"the keyboard moved the selection", func(l *ListView) {
			l.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
		}},
		{"a Ctrl click edited the selection", func(l *ListView) {
			l.Mode = SelectExtended
			l.MousePress(widget.MouseEvent{Pos: rowPoint(l, 2), Button: platform.ButtonLeft, Mods: ctrl})
		}},
		{"it was disabled and enabled again", func(l *ListView) {
			l.SetEnabled(false)
			l.MousePress(widget.MouseEvent{Pos: rowPoint(l, 2), Button: platform.ButtonLeft})
			l.SetEnabled(true)
		}},
		{"the content was replaced", func(l *ListView) { l.Count = 12 }},
		{"it was scrolled programmatically", func(l *ListView) { l.ScrollTo(4) }},
		{"type-ahead moved the selection", func(l *ListView) { l.TextInput('r') }},
	} {
		l, _, act := listRig(t, 40)
		l.MousePress(widget.MouseEvent{Pos: rowPoint(l, 2), Button: platform.ButtonLeft})
		tc.interrupt(l)
		// The same row, wherever it is now: an interruption that moves the
		// rows is already answered by the row under the pointer changing,
		// and the thing under test is the one that is not.
		second := rowPoint(l, 2)
		if l.RowAt(second) != 2 {
			t.Fatalf("%s: the second click is on row %d, want row 2 — the case tests nothing", tc.what, l.RowAt(second))
		}
		l.MousePress(widget.MouseEvent{Pos: second, Button: platform.ButtonLeft})
		if len(*act) != 0 {
			t.Errorf("a click, %s, and a click activated %v, want nothing", tc.what, *act)
		}
	}
}

// The control: nothing in between, and it is a double click as it always was.
func TestAnUninterruptedDoubleClickStillActivates(t *testing.T) {
	l, _, act := listRig(t, 40)
	at := rowPoint(l, 2)
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 1 || (*act)[0] != 2 {
		t.Errorf("a double click activated %v, want row 2 once", *act)
	}
	// And a third click starts again rather than activating a second time.
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 1 {
		t.Errorf("three clicks activated %v, want one", *act)
	}
}
