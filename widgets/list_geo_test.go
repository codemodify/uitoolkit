package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// placedList is a list the size of a panel's playlist well, with its rows
// inset the way a skin's layout would place them and a groove down the
// right-hand edge of the art — not of the rows.
func placedList(t *testing.T, geo *RowGeometry) *ListView {
	t.Helper()
	l := NewListView(40, func(i int) string { return fmt.Sprintf("Track %d", i+1) }, nil)
	l.RowGeo = func(style.LookAndFeel) *RowGeometry { return geo }
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	return l
}

var panelGeo = &RowGeometry{
	Rows:      paintengine2d.XYWH(12, 20, 160, 80),
	Bar:       paintengine2d.XYWH(180, 20, 8, 80),
	RowHeight: 10,
}

// The skin's row height is the row height, and the rows start where the
// skin's rows slot starts: a press lands on the row the art shows.
func TestListTakesItsRowsFromTheSkin(t *testing.T) {
	l := placedList(t, panelGeo)
	if got := l.rowH(); got != 10 {
		t.Fatalf("row height %v, want the skin's 10", got)
	}
	if got := l.inner(); got.Dx() != 160 || got.Dy() != 80 {
		t.Fatalf("the rows are %vx%v, want the skin's 160x80", got.Dx(), got.Dy())
	}
	// Eight rows fit; the fourth starts 30 below the top of the rows slot.
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(60, 20+35)})
	if l.Selected != 3 {
		t.Errorf("a press on the fourth row selected %d", l.Selected)
	}
	// Above the rows slot is the panel's art, not row 0.
	l.Selected = -1
	l.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(60, 4)})
	if l.Selected != -1 {
		t.Errorf("a press on the art above the rows selected %d", l.Selected)
	}
}

// The thumb rides the groove the skin named, not a gutter cut out of the
// rows, and it is where the offset says.
func TestListScrollThumbRidesTheSkinsGroove(t *testing.T) {
	l := placedList(t, panelGeo)
	track, thumb := l.ScrollTrack()
	if track.Min.X != 180 || track.Dx() != 8 {
		t.Fatalf("the groove is %v, want the skin's", track)
	}
	if thumb.Empty() {
		t.Fatal("40 rows of 10 in 80 pixels and no thumb")
	}
	if thumb.Min.Y < track.Min.Y || thumb.Max.Y > track.Max.Y+0.01 {
		t.Errorf("the thumb %v left the groove %v", thumb, track)
	}
	top := thumb.Min.Y
	l.ScrollTo(l.MaxOffset())
	_, thumb = l.ScrollTrack()
	if thumb.Min.Y <= top {
		t.Errorf("scrolling to the end left the thumb at %v (was %v)", thumb.Min.Y, top)
	}
	if thumb.Max.Y > track.Max.Y+0.01 {
		t.Errorf("the thumb %v ran past the end of the groove %v", thumb, track)
	}
}

// A picture of a thumb is one size whatever the list holds.
func TestListThumbLengthIsFixedWhenTheSkinSaysSo(t *testing.T) {
	geo := *panelGeo
	geo.ThumbLength = 18
	l := placedList(t, &geo)
	if _, thumb := l.ScrollTrack(); thumb.Dy() != 18 {
		t.Errorf("the thumb is %v tall, want the sprite's 18", thumb.Dy())
	}
}

// The rows keep their whole width: the groove is beside them in the art,
// so nothing is taken out of the rows for a bar.
func TestPlacedListGivesUpNoGutter(t *testing.T) {
	l := placedList(t, panelGeo)
	if got := l.rowsW(); got != 160 {
		t.Errorf("the rows are %v wide, want the whole 160 of the slot", got)
	}
	plain := NewListView(40, func(i int) string { return "x" }, nil)
	plain.SetHost(&host{})
	plain.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	if plain.rowsW() >= plain.inner().Dx() {
		t.Error("an ordinary overflowing list stopped giving up its gutter")
	}
}

// A layout that names rows and no groove is a list with no bar: it still
// scrolls, it just paints nothing there.
func TestPlacedListWithNoGrooveHasNoBar(t *testing.T) {
	l := placedList(t, &RowGeometry{Rows: paintengine2d.XYWH(0, 0, 200, 80), RowHeight: 10})
	if _, thumb := l.ScrollTrack(); !thumb.Empty() {
		t.Errorf("a layout with no groove painted a thumb at %v", thumb)
	}
	before := l.OffsetY
	l.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 3)})
	if l.OffsetY == before {
		t.Error("a list with no bar stopped scrolling")
	}
}

// Nothing about a list without the hook changes, and a hook that answers
// nil for this look is a list without the hook.
func TestListWithoutTheHookIsUnchanged(t *testing.T) {
	plain := NewListView(40, func(i int) string { return fmt.Sprintf("Track %d", i+1) }, nil)
	plain.SetHost(&host{})
	plain.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	unskinned := NewListView(40, func(i int) string { return fmt.Sprintf("Track %d", i+1) }, nil)
	unskinned.RowGeo = func(style.LookAndFeel) *RowGeometry { return nil }
	unskinned.SetHost(&host{})
	unskinned.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	if plain.rowH() != unskinned.rowH() || plain.inner() != unskinned.inner() {
		t.Fatal("a nil answer changed the list's geometry")
	}
	if !sameImage(immediatePaint(plain, 200, 120), immediatePaint(unskinned, 200, 120)) {
		t.Error("a nil answer changed what the list looks like")
	}
}

// The rows are painted where the skin put them, and nowhere else.
func TestPlacedListPaintsOnItsSlot(t *testing.T) {
	l := placedList(t, panelGeo)
	l.Selected = 0
	img := immediatePaint(l, 200, 120)
	if bandInk(img, 20, 100, 12, 172) == 0 {
		t.Error("nothing was painted on the rows slot")
	}
	if n := bandInk(img, 0, 18, 0, 200); n != 0 {
		t.Errorf("%d pixels were painted above the rows slot, over the panel's art", n)
	}
}
