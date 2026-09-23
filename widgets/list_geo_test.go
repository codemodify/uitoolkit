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

// ---- the rows' own paint -------------------------------------------------------

// A skin's rows are its own type in its own ink: a placed list paints them
// through the app's painter rather than through the pack underneath, which
// is what DrawListRow would have given it.
func TestSkinPaintedRowUsesTheSkinsInk(t *testing.T) {
	green := paintengine2d.RGB(0, 1, 0)
	l := placedList(t, panelGeo)
	pack := immediatePaint(l, 200, 120)
	rows := 0
	l.RowPaint = func(ctx *paintengine2d.Context, b paintengine2d.Rect, i int, _ style.ControlState) bool {
		rows++
		ctx.DrawRect(b, paintengine2d.Fill(green))
		return true
	}
	skin := immediatePaint(l, 200, 120)
	if rows < 8 {
		t.Fatalf("the painter was asked for %d of the eight rows in view", rows)
	}
	if sameImage(pack, skin) {
		t.Fatal("the painter drew and the list still looks like the pack's rows")
	}
	// The middle of the third row, in the list's own coordinates.
	r, g, b, _ := skin.PremulAt(60, 20+25)
	if r > 8 || g < 240 || b > 8 {
		t.Errorf("a skin-painted row is rgb(%d,%d,%d), want the skin's green", r, g, b)
	}
	// And the art above the rows slot is still the art: the painter is
	// given the rows and nothing else.
	if _, g, _, _ := skin.PremulAt(60, 4); g > 240 {
		t.Error("the row painter drew outside the skin's rows slot")
	}
}

// A painter with nothing for this look answers false and the look's own row
// is drawn, so a half-skinned app is a coherent app in a real look.
func TestRowPainterThatRefusesFallsBackToTheLook(t *testing.T) {
	l := placedList(t, panelGeo)
	plain := immediatePaint(l, 200, 120)
	l.RowPaint = func(*paintengine2d.Context, paintengine2d.Rect, int, style.ControlState) bool { return false }
	if !sameImage(plain, immediatePaint(l, 200, 120)) {
		t.Error("a painter that took nothing over changed the rows")
	}
}

// The detail column sits at the row's right-hand end, which is where a
// track's length goes.
func TestListDetailSitsAtTheRowsEnd(t *testing.T) {
	l := NewListView(6, func(i int) string { return "Track" }, nil)
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 200, 120))
	bare := immediatePaint(l, 200, 120)
	l.ItemDetail = func(i int) string { return "3:07" }
	with := immediatePaint(l, 200, 120)
	if changedPixels(bare, with, 4, 28, 140, 196) == 0 {
		t.Error("the detail column painted nothing at the end of the row")
	}
	// The title is still at the start of the row: the detail is beside
	// it, not instead of it.
	if bandInk(with, 4, 28, 4, 60) == 0 {
		t.Error("the row lost its title to the detail column")
	}
	// And the detail is the row's second cell, not a longer title: the
	// rows the app never asked about are untouched.
	l.ItemDetail = func(i int) string { return "" }
	blank := immediatePaint(l, 200, 120)
	if changedPixels(blank, with, 4, 28, 140, 196) == 0 {
		t.Error("an empty detail painted the same as a full one")
	}
}

// A panel's thumb is a sprite of the skin's own, riding the groove the
// layout named; the look's bar is not painted under it.
func TestListScrollPainterTakesTheGroove(t *testing.T) {
	l := placedList(t, panelGeo)
	look := immediatePaint(l, 200, 120)
	var track, thumb paintengine2d.Rect
	l.ScrollPaint = func(ctx *paintengine2d.Context, tr, th paintengine2d.Rect, _ style.ControlState) bool {
		track, thumb = tr, th
		ctx.DrawRect(th, paintengine2d.Fill(paintengine2d.RGB(1, 0, 0)))
		return true
	}
	own := immediatePaint(l, 200, 120)
	// The painter is given the groove the skin named, in the box the bar
	// would have been drawn in.
	if track.Dx() != 8 || track.Dy() != 80 {
		t.Fatalf("the painter was given the groove %v", track)
	}
	if thumb.Dy() >= track.Dy() || thumb.Dx() != track.Dx() {
		t.Fatalf("the painter was given the thumb %v in the groove %v", thumb, track)
	}
	if sameImage(look, own) {
		t.Fatal("the scroll painter drew and the bar did not change")
	}
	if r, _, _, _ := own.PremulAt(183, 20+2); r < 240 {
		t.Errorf("the skin's thumb is not on the skin's groove (r=%d)", r)
	}
	// A painter that has no sprite for this look leaves the look's bar.
	l.ScrollPaint = func(*paintengine2d.Context, paintengine2d.Rect, paintengine2d.Rect, style.ControlState) bool {
		return false
	}
	if !sameImage(look, immediatePaint(l, 200, 120)) {
		t.Error("a scroll painter that took nothing over changed the bar")
	}
}

// changedPixels is how many pixels of a band two paints disagree about.
func changedPixels(a, b *paintengine2d.Image, y0, y1, x0, x1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r1, g1, b1, a1 := a.PremulAt(x, y)
			r2, g2, b2, a2 := b.PremulAt(x, y)
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				n++
			}
		}
	}
	return n
}
