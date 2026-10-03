package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/style"
)

// What a screen reader is told about a row is where that row is drawn.
//
// The accessibility path rebuilt the geometry instead of sharing it with
// painting, and got three things wrong: it translated by inner().Min,
// which is (0,0) in view space, so a list with padding reported every row
// at the component's own corner; it used the whole viewport's width, so
// the rectangle was a scroll gutter wider than the row; and it tested
// offscreen against the whole component, so a row scrolled out of the
// rows area still counted as on screen.
func TestAccessibleRowsAreWhereTheRowsAreDrawn(t *testing.T) {
	l := NewListView(20, func(i int) string { return "row" }, nil)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.RowGeo = func(style.LookAndFeel) *RowGeometry {
		return &RowGeometry{Rows: paintengine2d.XYWH(10, 20, 120, 60), RowHeight: 20}
	}
	l.Arrange(paintengine2d.XYWH(0, 0, 160, 100))

	items := l.AccessibleItems()
	if len(items) != 20 {
		t.Fatalf("got %d rows", len(items))
	}
	// The first row is where the slot says, not at the corner.
	first := items[0].Bounds
	if first.Min.X != 10 || first.Min.Y != 20 {
		t.Errorf("the first row is at %g,%g; it is drawn at 10,20", first.Min.X, first.Min.Y)
	}
	if first.Dx() != 120 {
		t.Errorf("the first row is %g wide; the rows slot is 120", first.Dx())
	}

	// Scrolled, it moves with the rows rather than off the top.
	l.ScrollTo(20)
	items = l.AccessibleItems()
	if b := items[0].Bounds; b.Min.X != 10 || b.Min.Y != 0 {
		t.Errorf("after scrolling 20 the first row is at %g,%g, want 10,0", b.Min.X, b.Min.Y)
	}
	// A row *below the slot but still inside the component* is offscreen.
	// That is the case the whole-component test could not see: the slot
	// is y 20..80 of a 100-tall component, so a row at 80..100 is out of
	// the rows area and well inside the view.
	l.ScrollTo(0)
	items = l.AccessibleItems()
	if b := items[3].Bounds; b.Min.Y < 80 || b.Min.Y > 100 {
		t.Fatalf("row 3 is at y=%g; the case needs it between the slot's foot and the view's", b.Min.Y)
	}
	if items[3].State&a11y.StateOffscreen == 0 {
		t.Error("a row below the rows slot is not marked offscreen, though it is inside the component")
	}
	if items[1].State&a11y.StateOffscreen != 0 {
		t.Error("a row inside the slot is marked offscreen")
	}
}

// A framed list's accessible row stops where the row stops: the scroll
// bar's gutter is not part of it.
func TestAnAccessibleRowExcludesTheScrollGutter(t *testing.T) {
	l := NewListView(200, func(i int) string { return "row" }, nil)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Arrange(paintengine2d.XYWH(0, 0, 160, 100))
	if l.MaxOffset() <= 0 {
		t.Skip("no scroll bar, so no gutter to leave out")
	}
	items := l.AccessibleItems()
	if len(items) == 0 {
		t.Fatal("no rows")
	}
	w := items[0].Bounds.Dx()
	if full := l.inner().Dx(); w >= full {
		t.Errorf("the accessible row is %g wide, the whole viewport; the gutter is not part of the row", w)
	}
	if w != l.rowsW() {
		t.Errorf("the accessible row is %g wide, the painted row %g", w, l.rowsW())
	}
}
