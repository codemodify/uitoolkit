package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// buttonLabels lists the message box's buttons left to right.
func buttonLabels(mb *MessageBox) string {
	var got []string
	widget.Walk(mb.Overlay(), func(c widget.Component) {
		if b, ok := c.(*Button); ok {
			got = append(got, b.Text)
		}
	})
	return strings.Join(got, " ")
}

// Windows and KDE put the default button first and read the row "Yes No
// Cancel"; Mac and GNOME put it last: "Cancel No Yes".
func TestMessageBoxButtonOrderFollowsLook(t *testing.T) {
	cases := []struct {
		pack    string
		buttons MessageButtons
		want    string
	}{
		{"win95", ButtonsYesNoCancel, "Yes No Cancel"},
		{"win95", ButtonsOKCancel, "OK Cancel"},
		{"win95", ButtonsYesNo, "Yes No"},
		{"aqua", ButtonsYesNoCancel, "Cancel No Yes"},
		{"aqua", ButtonsOKCancel, "Cancel OK"},
	}
	for _, c := range cases {
		pack, ok := style.LoadTheme(c.pack)
		if !ok {
			t.Fatalf("no %s pack", c.pack)
		}
		h := &fakeWindow{look: pack.Look()}
		tf := NewTextField("", "", nil)
		root := NewColumn(tf)
		root.SetHost(h)
		root.Arrange(paintengine2d.XYWH(0, 0, 400, 200))
		mb := NewMessageBox(MessageBoxOptions{Message: "Save changes?", Buttons: c.buttons})
		mb.Show(tf)
		if got := buttonLabels(mb); got != c.want {
			t.Errorf("%s: buttons %q, want %q", c.pack, got, c.want)
		}
		// Shown twice, the order must not flip back.
		mb.finish(ResultCancel)
		mb.Show(tf)
		if got := buttonLabels(mb); got != c.want {
			t.Errorf("%s shown again: buttons %q, want %q", c.pack, got, c.want)
		}
		mb.finish(ResultCancel)
	}
}
