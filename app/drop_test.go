package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// Files dragged from another app reach the drop zone under them as local
// paths; text dropped on a field goes into it where it was dropped; a drop
// nothing takes is refused.
func TestDropsFromOtherApps(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	field := widgets.NewTextField("Hello world", "", nil)
	var got []string
	zone := widgets.NewDropZone(widgets.NewColumn(field, widgets.NewLabel("drop files here")), func(p []string) { got = p })
	w.SetContent(zone)
	a.PumpOnce()
	off, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Skip("not an offscreen surface")
	}
	// Files, over the label part of the zone.
	off.SimulateDrop(paintengine2d.Pt(100, 200), map[string][]byte{
		"text/uri-list": []byte("file:///home/ada/a%20b.txt\r\n# comment\r\nfile:///tmp/c.png\r\n"),
	})
	a.PumpOnce()
	if len(got) != 2 || got[0] != "/home/ada/a b.txt" || got[1] != "/tmp/c.png" {
		t.Fatalf("paths %q", got)
	}
	if taken := off.DropTaken(); taken == nil || !*taken {
		t.Fatal("the drop should be finished as taken")
	}
	// Text, onto the field (the zone takes no text).
	fb := field.Bounds()
	off.SimulateDrop(paintengine2d.Pt(fb.Max.X-4, fb.Min.Y+fb.Dy()/2), map[string][]byte{
		"text/plain;charset=utf-8": []byte("big\nnew"),
		"text/html":                []byte("<b>big</b>"),
	})
	a.PumpOnce()
	if field.Text != "Hello worldbig new" {
		t.Fatalf("field %q", field.Text)
	}
	// Something nobody takes.
	off.SimulateDrop(paintengine2d.Pt(100, 200), map[string][]byte{"image/png": {1, 2, 3}})
	a.PumpOnce()
	if taken := off.DropTaken(); taken == nil || *taken {
		t.Fatal("an unwanted drop is refused")
	}
}
