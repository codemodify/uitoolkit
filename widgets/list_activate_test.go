package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Selecting a row and committing to one are different things, and a list that
// has only one of them makes an application guess.
//
// A player wired its play action to OnSelect, because there was nowhere else
// to put it, and a single click started a track — arrowing down the playlist
// played every row on the way past. TableView has had OnActivate all along;
// this is the same contract on ListView.

func listRig(t *testing.T, rows int) (*ListView, *[]int, *[]int) {
	t.Helper()
	var sel, act []int
	l := NewListView(rows, func(i int) string { return "row" }, nil)
	l.OnSelect = func(i int) { sel = append(sel, i) }
	l.OnActivate = func(i int) { act = append(act, i) }
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	return l, &sel, &act
}

// Arrowing down selects and commits to nothing.
func TestArrowingDownAListSelectsWithoutActivating(t *testing.T) {
	l, sel, act := listRig(t, 5)
	for i := 0; i < 3; i++ {
		l.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	}
	if len(*sel) == 0 {
		t.Error("arrowing down selected nothing")
	}
	if len(*act) != 0 {
		t.Errorf("arrowing down activated %v", *act)
	}
}

// Return and Space commit to the current row.
func TestReturnAndSpaceActivateTheCurrentRow(t *testing.T) {
	for _, k := range []platform.Key{platform.KeyReturn, platform.KeySpace} {
		l, _, act := listRig(t, 5)
		l.Selected = 2
		l.KeyPress(widget.KeyEvent{Key: k})
		if len(*act) != 1 || (*act)[0] != 2 {
			t.Errorf("%v activated %v, want row 2 once", k, *act)
		}
	}
}

// One click selects; a second on the same row commits.
func TestASecondClickOnARowActivatesIt(t *testing.T) {
	l, sel, act := listRig(t, 5)
	at := rowPoint(l, 2)
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 0 {
		t.Fatalf("one click activated %v", *act)
	}
	l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act) != 1 {
		t.Errorf("a double click activated %v, want one row", *act)
	}
	if len(*sel) != 2 {
		t.Errorf("two clicks selected %v, want both of them", *sel)
	}
	// And two clicks far enough apart are two clicks.
	l2, _, act2 := listRig(t, 5)
	l2.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	l2.lastAt = time.Now().Add(-2 * doubleClickInterval)
	l2.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if len(*act2) != 0 {
		t.Errorf("two slow clicks activated %v", *act2)
	}
}

// A screen reader's default action commits, including on the row that is
// already current — which is the case it used to answer true and do nothing
// for, because navigate to where you already are is no change at all.
func TestTheDefaultAccessibilityActionActivates(t *testing.T) {
	l, _, act := listRig(t, 5)
	l.Selected = 1
	if !l.AccessibleAction(1, a11y.ActionDefault) {
		t.Fatal("the default action was refused")
	}
	if len(*act) != 1 || (*act)[0] != 1 {
		t.Errorf("the default action on the current row activated %v, want row 1", *act)
	}
}

// With no OnActivate the list behaves as it did: an application written
// against the old contract still hears about Return through OnSelect.
func TestWithoutOnActivateCommitmentStillReachesOnSelect(t *testing.T) {
	var sel []int
	l := NewListView(5, func(i int) string { return "row" }, func(i int) { sel = append(sel, i) })
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 300))
	l.Selected = 3
	l.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if len(sel) != 1 || sel[0] != 3 {
		t.Errorf("Return with no OnActivate reported %v, want row 3 through OnSelect", sel)
	}
}

// A secondary press selects the row under it by default, and leaves the
// selection alone when the application says its menu is about the selection.
func TestContextKeepsSelectionIsThePolicyForASecondaryPress(t *testing.T) {
	at := paintengine2d.Pt(20, 28*2+5)
	for _, tc := range []struct{ keeps, want bool }{{false, true}, {true, false}} {
		l, sel, _ := listRig(t, 5)
		l.ContextKeepsSelection = tc.keeps
		var ctx []int
		l.OnContext = func(i int, _ paintengine2d.Point) { ctx = append(ctx, i) }
		l.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonRight})
		if got := len(*sel) > 0; got != tc.want {
			t.Errorf("ContextKeepsSelection=%v selected %v", tc.keeps, *sel)
		}
		// Either way the menu is told which row it was over.
		if len(ctx) != 1 {
			t.Errorf("ContextKeepsSelection=%v: OnContext was told %v", tc.keeps, ctx)
		}
	}
}

// RowAt is the row under a point, and nothing under the bar or past the end.
func TestRowAtFindsTheRowUnderAPoint(t *testing.T) {
	l, _, _ := listRig(t, 3)
	if got := l.RowAt(rowPoint(l, 1)); got != 1 {
		t.Errorf("RowAt in the second row = %d, want 1", got)
	}
	if got := l.RowAt(paintengine2d.Pt(20, l.rowH()*20)); got != -1 {
		t.Errorf("RowAt past the last row = %d, want -1", got)
	}
	if got := l.RowAt(paintengine2d.Pt(20, -5)); got != -1 {
		t.Errorf("RowAt above the view = %d, want -1", got)
	}
}

// rowPoint is a point in the middle of row i, in the list's own coordinates.
func rowPoint(l *ListView, i int) paintengine2d.Point {
	r := fromView(l.rowRect(i), l.pad())
	return paintengine2d.Pt(r.Min.X+4, r.Min.Y+r.Dy()/2)
}
