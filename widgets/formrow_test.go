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
