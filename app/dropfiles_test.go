package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// A drag carrying files goes to the thing that wants files, even when
// something that only wants text is nearer the pointer.
//
// A file manager offers text/uri-list and text/plain on the same drag —
// the paths as text, for anything that can only take text. Taking the
// innermost widget that accepts anything at all therefore handed every
// file drop to a text widget: dropping files on the body of a Write
// window typed their paths into it rather than attaching them, which is
// exactly where people drop them.
//
// The shape here is the one every "drop files here" window has: a zone
// that wants files, with an editable field inside it.
func TestFileDropPrefersTheZoneOverATextWidgetInsideIt(t *testing.T) {
	a := New(Options{Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	field := widgets.NewTextField("", "", nil)
	var gotFiles []string
	zone := widgets.NewDropZone(field, func(p []string) { gotFiles = append(gotFiles, p...) })
	w.SetContent(widgets.NewColumn(zone))
	a.PumpOnce()

	at := widget.DeviceBounds(field).Center()
	if at.X == 0 && at.Y == 0 {
		t.Fatal("the field was not laid out")
	}

	// What a file manager really offers, in the order it offers it.
	tgt, c := w.dropTarget(at, []string{"text/uri-list", "text/plain"})
	if tgt == nil {
		t.Fatal("nothing took a file drop over the field")
	}
	if _, isField := c.(*widgets.TextField); isField {
		t.Error("the text field took a drop carrying files: the paths would be typed into it")
	}
	if c != widget.Component(zone) {
		t.Errorf("a file drop went to %T, want the zone that asked for files", c)
	}

	// And a drag carrying only text still goes to the field: nothing
	// above it wants text, and the field does.
	_, c = w.dropTarget(at, []string{"text/plain"})
	if _, isField := c.(*widgets.TextField); !isField {
		t.Errorf("a text-only drag went to %T, want the field under the pointer", c)
	}
}
