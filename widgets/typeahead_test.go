package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Typing in a list jumps to the matching row, as Explorer, Finder, GTK and
// Qt views do: a prefix narrows, one letter repeated cycles, a pause of a
// second starts over.
func TestListTypeAheadFind(t *testing.T) {
	now := time.Unix(1000, 0)
	typeAheadNow = func() time.Time { return now }
	defer func() { typeAheadNow = time.Now }()
	items := []string{"Apple", "Banana", "Blueberry", "Cherry", "blackberry", "Date"}
	l := NewListView(len(items), func(i int) string { return items[i] }, nil)
	l.SetHost(&fakeWindow{look: style.DarkLook()})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	typ := func(s string) {
		for _, r := range s {
			l.TextInput(r)
			now = now.Add(100 * time.Millisecond)
		}
	}
	typ("b")
	if l.Selected != 1 {
		t.Fatalf("b -> %d, want Banana (1)", l.Selected)
	}
	typ("l")
	if l.Selected != 2 {
		t.Fatalf("bl -> %d, want Blueberry (2)", l.Selected)
	}
	typ("a")
	if l.Selected != 4 {
		t.Fatalf("bla -> %d, want blackberry (4), case-insensitive", l.Selected)
	}
	now = now.Add(2 * time.Second)
	typ("b")
	if l.Selected != 1 {
		t.Fatalf("after a pause, b -> %d, want the next b-row after blackberry, wrapping: Banana (1)", l.Selected)
	}
	typ("b")
	if l.Selected != 2 {
		t.Fatalf("b again -> %d, want to cycle on to Blueberry (2)", l.Selected)
	}
	now = now.Add(2 * time.Second)
	if l.TextInput(' ') {
		t.Fatal("space outside a search should be left to the Space key")
	}
	lo, hi := l.VisibleRange()
	typ("d")
	if l.Selected != 5 {
		t.Fatalf("d -> %d, want Date (5)", l.Selected)
	}
	if lo2, hi2 := l.VisibleRange(); 5 < lo2 || 5 >= hi2 {
		t.Fatalf("the found row is not scrolled into view: [%d,%d) (was [%d,%d))", lo2, hi2, lo, hi)
	}
}

// The table searches its main text column, not the star column before it.
func TestTableTypeAheadUsesMainColumn(t *testing.T) {
	typeAheadNow = func() time.Time { return time.Unix(2000, 0) }
	defer func() { typeAheadNow = time.Now }()
	rows := [][]string{{"★", "Invoice"}, {"", "Meeting"}, {"★", "Minutes"}}
	tv := NewTableView([]TableColumn{{Title: "★", Width: 28}, {Title: "Topic"}}, len(rows),
		func(r, c int) string { return rows[r][c] }, nil)
	tv.SetHost(&fakeWindow{look: style.DarkLook()})
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	tv.TextInput('m')
	if tv.Selected != 1 {
		t.Fatalf("m -> %d, want Meeting (1)", tv.Selected)
	}
}

// The Menu key and Shift+F10 open the context menu of the current row.
func TestContextMenuFromKeyboard(t *testing.T) {
	items := []string{"a", "b", "c"}
	l := NewListView(len(items), func(i int) string { return items[i] }, nil)
	host := &fakeWindow{look: style.DarkLook()}
	l.SetHost(host)
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	l.Selected = 1
	got, at := -2, paintengine2d.Point{}
	l.OnContext = func(i int, p paintengine2d.Point) { got, at = i, p }
	l.KeyPress(widget.KeyEvent{Key: platform.KeyMenu})
	if got != 1 {
		t.Fatalf("menu key opened the context menu for %d, want 1", got)
	}
	if r := l.rowRect(1); at.Y < r.Max.Y-0.5 || at.Y > r.Max.Y+l.frame().Top+0.5 {
		t.Fatalf("menu at %v, want under row 1 %v", at, r)
	}
	got = -2
	l.KeyPress(widget.KeyEvent{Key: platform.KeyF10, Mods: platform.ModShift})
	if got != 1 {
		t.Fatal("Shift+F10 did not open the context menu")
	}
}
