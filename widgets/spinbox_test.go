package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The look decides where a spin box puts its buttons (Qt's CC_SpinBox):
// inside the field's frame for Windows and GNOME (GNOME side by side), a
// stepper beside the field for Mac OS. Inside, the editor is frameless and
// ends where the buttons start.
func TestSpinBoxFollowsTheLook(t *testing.T) {
	for _, c := range []struct {
		pack           string
		inside, across bool
	}{{"win95", true, false}, {"adwaita", true, true}, {"aqua", false, false}, {"dark", true, false}} {
		pack, ok := style.LoadTheme(c.pack)
		if !ok {
			t.Fatalf("no %s pack", c.pack)
		}
		got := 0.0
		n := NewNumberField(0, 10, 5, 1, func(v float64) { got = v })
		n.SetLook(pack.Look())
		n.SetHost(&host{})
		n.Arrange(paintengine2d.XYWH(0, 0, 160, 30))
		field, box, up, down, inside := n.spinParts()
		if inside != c.inside || n.Field().Frameless != c.inside {
			t.Fatalf("%s: inside %v (field frameless %v), want %v", c.pack, inside, n.Field().Frameless, c.inside)
		}
		if field.Max.X > box.Min.X {
			t.Errorf("%s: the editor %v runs under the buttons %v", c.pack, field, box)
		}
		if c.inside && (box.Max.X >= 160 || box.Min.Y <= 0) {
			t.Errorf("%s: buttons %v should sit inside the frame", c.pack, box)
		}
		if across := up.Min.X > down.Min.X; across != c.across {
			t.Errorf("%s: buttons across %v, want %v (up %v, down %v)", c.pack, across, c.across, up, down)
		}
		n.MousePress(widget.MouseEvent{Pos: up.Center(), Button: platform.ButtonLeft})
		n.MouseRelease(widget.MouseEvent{Pos: up.Center(), Button: platform.ButtonLeft})
		if got != 6 {
			t.Errorf("%s: up gave %v, want 6", c.pack, got)
		}
		n.MousePress(widget.MouseEvent{Pos: down.Center(), Button: platform.ButtonLeft})
		if got != 5 {
			t.Errorf("%s: down gave %v, want 5", c.pack, got)
		}
	}
}
