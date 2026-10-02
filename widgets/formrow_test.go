package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A form whose fields depend on a choice shows only the rows that choice
// takes. Hiding the field alone left its caption behind, and there was no
// handle on the row to hide both, so such a form either showed every field
// it might ever need or was torn down and built again.
func TestAFormRowCanBeHidden(t *testing.T) {
	f := NewForm()
	a := f.Row("Passphrase:", NewTextField("", "", nil))
	b := f.Row("Keyfile:", NewTextField("", "", nil))
	c := f.Row("Shares:", NewTextField("", "", nil))
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})

	full := f.Measure(layout.Unbounded())
	f.Arrange(paintengine2d.XYWH(0, 0, full.X, full.Y))
	if !b.Visible() {
		t.Fatal("a new row is not showing")
	}

	b.SetVisible(false)
	if b.Visible() {
		t.Error("the row still reports itself visible")
	}
	if b.Label.Visible() || b.Field.Visible() {
		t.Error("hiding the row left the caption or the field showing")
	}
	// The two that are left keep their captions and the form is shorter by
	// a row: the gap closes rather than leaving a hole where the row was.
	short := f.Measure(layout.Unbounded())
	if short.Y >= full.Y {
		t.Errorf("hiding a row left the form %g tall, no shorter than %g", short.Y, full.Y)
	}
	if !a.Visible() || !c.Visible() {
		t.Error("hiding one row hid another")
	}

	b.SetVisible(true)
	if back := f.Measure(layout.Unbounded()); back.Y != full.Y {
		t.Errorf("showing the row again gave %g, want the original %g", back.Y, full.Y)
	}
}

// Hiding a row takes its gap with it, and moves the rows under it to
// exactly where they are in a form that never had it.
//
// A hidden row's height is nothing, but the grid counted it among its
// tracks and kept the RowGap before it, so each hidden row left a gap: a
// form of eight rows showing three had five gaps too many between them.
func TestAHiddenFormRowTakesItsGapWithIt(t *testing.T) {
	build := func(n int) *Form {
		f := NewForm()
		for i := 0; i < n; i++ {
			f.Row("Row:", NewTextField("", "", nil))
		}
		f.SetLook(style.DarkLook())
		f.SetHost(&host{})
		return f
	}
	// Three rows, and three rows with five hidden ones among them.
	want := build(3)
	got := build(8)
	for i := 3; i < 8; i++ {
		got.rows[i].SetVisible(false)
	}
	w := want.Measure(layout.Unbounded())
	g := got.Measure(layout.Unbounded())
	if g.Y != w.Y {
		t.Errorf("eight rows showing three measure %g tall; three rows measure %g", g.Y, w.Y)
	}

	// And the rows that are left sit where they would in the short form.
	want.Arrange(paintengine2d.XYWH(0, 0, w.X, w.Y))
	got.Arrange(paintengine2d.XYWH(0, 0, w.X, w.Y))
	for i := 0; i < 3; i++ {
		a := want.rows[i].Field.Bounds().Min.Y
		b := got.rows[i].Field.Bounds().Min.Y
		if a != b {
			t.Errorf("row %d sits at %g with rows hidden below it, want %g", i, b, a)
		}
	}
}
