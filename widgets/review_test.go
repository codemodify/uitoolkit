package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"
)

// A target that refuses the text must not tell the source it took it.
// The source removes its original on the strength of that answer, so
// "accepted" and "inserted nothing" cannot be the same reply.
func TestRefusedTextDropIsNotReportedAsAccepted(t *testing.T) {
	t.Run("TextField", func(t *testing.T) {
		f := NewTextField("", "", nil)
		f.Accept = func(string) bool { return false }
		if f.Drop(widget.DropEvent{Text: "nope"}) {
			t.Error("a refused drop reported success")
		}
		if f.Text != "" {
			t.Errorf("the field took %q anyway", f.Text)
		}
		f.Accept = func(string) bool { return true }
		if !f.Drop(widget.DropEvent{Text: "yes"}) {
			t.Error("an accepted drop reported failure")
		}
	})
	t.Run("TextArea", func(t *testing.T) {
		a := NewTextArea("", "", nil)
		a.Accept = func(string) bool { return false }
		if a.Drop(widget.DropEvent{Text: "nope"}) {
			t.Error("a refused drop reported success")
		}
		if a.Text != "" {
			t.Errorf("the area took %q anyway", a.Text)
		}
	})
}

// Disabling an editable combo has to disable the field it holds. The
// field is a child, but focus enumeration and key dispatch do not
// consult an ancestor's enabled state, so a disabled combo used to leave
// a perfectly editable text field behind.
func TestDisabledEditableComboDisablesItsField(t *testing.T) {
	c := NewComboBox([]string{"One", "Two"}, 0, nil)
	c.SetEditable(true)
	c.SetEnabled(false)
	if c.field == nil {
		t.Skip("no field")
	}
	if c.field.Enabled() {
		t.Fatal("the combo is disabled and its field is still enabled")
	}
	c.SetEnabled(true)
	if !c.field.Enabled() {
		t.Fatal("re-enabling the combo left its field disabled")
	}
}

// Grid cells are the child's, not the position's. Removing a child used
// to shift every later child onto its neighbour's cell, so a grid
// silently rearranged itself.
func TestRemovingAGridChildKeepsTheOthersInPlace(t *testing.T) {
	g := NewGrid()
	a := NewButton("A", nil)
	b := NewButton("B", nil)
	g.Place(a, 0, 0)
	g.Place(b, 1, 1)

	if cell := g.CellOf(b); cell == nil || cell.Row != 1 || cell.Col != 1 {
		t.Fatalf("B started at %+v", cell)
	}
	g.Remove(a)
	cell := g.CellOf(b)
	if cell == nil {
		t.Fatal("B lost its cell when A was removed")
	}
	if cell.Row != 1 || cell.Col != 1 {
		t.Fatalf("removing A moved B to (%d,%d); a cell belongs to its child", cell.Row, cell.Col)
	}
}

// Flexible tracks share what is left, but a track whose content needs
// more than its share keeps its content — and the others must then share
// what *remains*, not what would have remained. Taking max(content,
// share) for each track independently overflowed the row.
func TestGridFlexTracksDoNotOverflow(t *testing.T) {
	g := NewGrid()
	g.ColGap = 0
	g.Cols = []Track{{Mode: TrackFlex}, {Mode: TrackFlex}}
	wide := newSized(90, 10)
	narrow := newSized(10, 10)
	g.Place(wide, 0, 0)
	g.Place(narrow, 0, 1)

	const avail = 100
	cols := g.columns(avail)
	var total float32
	for _, w := range cols {
		total += w
	}
	if total > avail+0.5 {
		t.Fatalf("two flex columns came to %v in %v: %v", total, avail, cols)
	}
	g.Arrange(paintengine2d.XYWH(0, 0, avail, 20))
	if r := narrow.Bounds(); r.Max.X > avail+0.5 {
		t.Fatalf("the second child ends at %v, past the %v edge", r.Max.X, avail)
	}
}
