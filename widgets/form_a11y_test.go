package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/style"
)

// A form row of a caption and a value is read as both.
//
// Form.AddRow names the field beside a caption after the caption, Qt's
// buddy, which is right for something that takes input. A Label's
// accessible name is read *instead of* its text, so a row set out as
// "Program:" / "/usr/bin/mail" was read "Program:", "Program" — and the
// value, which is the whole reason a consent prompt is on the screen, was
// never read at all.
func TestAFormValueIsReadAsItself(t *testing.T) {
	f := NewForm()
	value := NewLabel("/usr/bin/mail")
	f.AddRow("Program:", value)
	field := NewTextField("", "", nil)
	f.AddRow("Name:", field)
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})

	if got := value.AccessibleName(); got != "" {
		t.Errorf("the value label was named %q after its caption; it should be read as its text", got)
	}
	var n a11y.Node
	value.Describe(&n)
	if !strings.Contains(n.Name, "/usr/bin/mail") {
		t.Errorf("a screen reader hears %q, which does not include the value", n.Name)
	}
	// A field that takes input is still named after its caption.
	if got := field.AccessibleName(); got != "Name" {
		t.Errorf("the input is named %q, want it named after its caption", got)
	}
}
