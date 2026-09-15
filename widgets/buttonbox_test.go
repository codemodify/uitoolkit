package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A dialog's buttons follow the look's platform: "Save Don't-Save Cancel"
// under Windows, "Don't-Save … Cancel Save" under Mac OS; Help at the far
// left; a theme switch moves them.
func TestButtonBoxFollowsThePlatform(t *testing.T) {
	bb := NewButtonBox().
		AddButton(NewButton("Cancel", nil), RoleReject).
		AddButton(NewButton("Save", nil), RoleAccept).
		AddButton(NewButton("Don't Save", nil), RoleDestructive).
		AddButton(NewButton("Help", nil), RoleHelp)
	names := func(bs []*Button) string {
		var out []string
		for _, b := range bs {
			out = append(out, b.Text)
		}
		return strings.Join(out, ",")
	}
	for _, c := range []struct{ pack, left, right string }{
		{"win95", "Help", "Save,Don't Save,Cancel"},
		{"aqua", "Help,Don't Save", "Cancel,Save"},
	} {
		pack, ok := style.LoadTheme(c.pack)
		if !ok {
			t.Fatalf("no %s", c.pack)
		}
		bb.SetLook(pack.Look())
		bb.SetHost(&host{})
		left, right := bb.Order()
		if names(left) != c.left || names(right) != c.right {
			t.Errorf("%s: %s | %s, want %s | %s", c.pack, names(left), names(right), c.left, c.right)
		}
		bb.Arrange(paintengine2d.XYWH(0, 0, 500, 30))
		last := right[len(right)-1].Bounds()
		if last.Max.X != 500 {
			t.Errorf("%s: the right group should end at the box's edge, ends at %v", c.pack, last.Max.X)
		}
		if left[0].Bounds().Min.X != 0 {
			t.Errorf("%s: help should start the left group", c.pack)
		}
	}
	for _, x := range bb.buttons {
		if x.b.Text == "Save" && !x.b.Primary {
			t.Error("the accept button should be the default")
		}
	}
}

// A label's newlines start new lines (QLabel, GtkLabel): its height grows
// with them, and no line-break glyph is drawn.
func TestLabelLines(t *testing.T) {
	l := NewLabel("Backend: memory\nHealth: ok\nConfig: ~/.config")
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	one := NewLabel("Backend: memory")
	one.SetLook(style.DarkLook())
	one.SetHost(&host{})
	h3, h1 := l.Measure(layout.Unbounded()).Y, one.Measure(layout.Unbounded()).Y
	if h3 < h1*2.5 {
		t.Fatalf("three lines measure %v, one line %v", h3, h1)
	}
	if l.Measure(layout.Unbounded()).X != one.Measure(layout.Unbounded()).X {
		t.Fatal("the widest line sets the width")
	}
}
