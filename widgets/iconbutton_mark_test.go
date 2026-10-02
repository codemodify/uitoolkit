package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// An IconButton's mark is the size the user's icon size asks for, as a
// tool button's and a tab's are — not whatever the control height has
// left after a fixed inset.
//
// The face is the look's control height, so on a compact pack the mark
// came out at whatever that left: 6 device pixels on metal-ocean, 8 on
// Window Maker, beside tab marks drawn at 16 in the same strip. The
// iconSize preference was not consulted at all.
func TestAnIconButtonsMarkFollowsTheIconSize(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, pack := range []string{"metal-ocean", "wmaker-default", "light", "breeze-night"} {
		p, ok := style.LoadTheme(pack)
		if !ok {
			continue
		}
		lk := p.Look()
		b := NewIconButton(style.IconTrash, "Delete", nil)
		b.SetLook(lk)
		b.SetHost(&host{})
		face := lk.Metrics().ControlH
		b.Arrange(paintengine2d.XYWH(0, 0, face, face))

		got := b.markSide(paintengine2d.XYWH(0, 0, face, face))
		// It must be readable, and it must not spill off its own face.
		if got > face {
			t.Errorf("%s: a %g mark on a %g face", pack, got, face)
		}
		if floor := min(style.Dip(lk, 10), face); got < floor {
			t.Errorf("%s: the mark is %g, below the %g it can be seen at (face %g)", pack, got, floor, face)
		}
	}
}
