package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// An editable combo box takes typed text, completes it inline from its
// items (the next key replaces the completion), and a pick from the list
// fills the field.
func TestEditableComboBox(t *testing.T) {
	var edits []string
	c := NewComboBox([]string{"apple", "apricot", "banana"}, -1, nil)
	c.OnEdit = func(s string) { edits = append(edits, s) }
	c.SetEditable(true)
	h := &host{}
	c.SetLook(style.DarkLook())
	c.SetHost(h)
	c.Arrange(paintengine2d.XYWH(0, 0, 200, 30))
	f := c.field
	if f == nil || c.WantsFocus() || !f.Frameless {
		t.Fatal("the field takes the focus for the box")
	}
	fr := f.Bounds()
	if fr.Empty() || fr.Max.X >= 200 {
		t.Fatalf("field box %v leaves no room for the arrow", fr)
	}
	f.RequestFocus()
	type_ := func(s string) {
		for _, r := range s {
			f.TextInput(r)
		}
	}
	type_("a")
	if c.Text() != "apple" || c.Selected != 0 {
		t.Fatalf("after 'a': %q, selected %d", c.Text(), c.Selected)
	}
	type_("pr") // replaces the completion each time
	if c.Text() != "apricot" || c.Selected != 1 {
		t.Fatalf("after 'apr': %q, selected %d", c.Text(), c.Selected)
	}
	f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if c.Text() != "apr" || c.Selected != -1 {
		t.Fatalf("backspace drops the completion: %q, selected %d", c.Text(), c.Selected)
	}
	type_("x")
	if c.Text() != "aprx" {
		t.Fatalf("no item matches: %q", c.Text())
	}
	c.Select(2)
	if c.Text() != "banana" || c.Selected != 2 {
		t.Fatalf("picked: %q", c.Text())
	}
	if len(edits) == 0 || edits[len(edits)-1] != "aprx" {
		t.Fatalf("OnEdit %v", edits)
	}
	// Down opens the list from the field.
	if !c.KeyPress(widget.KeyEvent{Key: platform.KeyDown}) {
		t.Fatal("Down in an editable box")
	}
	c.NoCompletion = true
	c.SetText("")
	type_("b")
	if c.Text() != "b" {
		t.Fatalf("NoCompletion: %q", c.Text())
	}
	img := paintengine2d.NewImage(210, 40)
	c.Paint(paintengine2d.NewContext(img))
}
