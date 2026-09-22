//go:build linux && cgo

package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The core wheel buttons are notches too: one press, ±1, three lines.
func TestX11CoreWheelIsANotch(t *testing.T) {
	for btn, want := range map[int][2]float32{4: {0, -1}, 5: {0, 1}, 6: {-1, 0}, 7: {1, 0}} {
		ev := x11ButtonEvents(btn, false, paintengine2d.Pt(1, 2), 0)
		if len(ev) != 1 || ev[0].ScrollPrecise || ev[0].Scroll.X != want[0] || ev[0].Scroll.Y != want[1] {
			t.Fatalf("button %d: %+v", btn, ev)
		}
	}
	for btn, want := range map[int]MouseButton{8: ButtonBack, 9: ButtonForward} {
		ev := x11ButtonEvents(btn, false, paintengine2d.Pt(1, 2), 0)
		if len(ev) != 1 || ev[0].Kind != EventMouseDown || ev[0].Button != want {
			t.Fatalf("button %d: %+v", btn, ev)
		}
	}
}

// The portal names an X window "x11:<hex id>", with no 0x and no padding.
func TestX11PortalParentHandle(t *testing.T) {
	if got := x11PortalParent(0x1a00003); got != "x11:1a00003" {
		t.Fatalf("%q", got)
	}
	if got := x11PortalParent(0); got != "" {
		t.Fatalf("no window: %q", got)
	}
}
