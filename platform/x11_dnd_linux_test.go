//go:build linux && cgo

package platform

import "testing"

// A drag owns the pointer until it drops, and not one event longer.
//
// XdndDrop is the end of the pointer half: the grab goes back, the
// picture comes down, and the spec has the source say nothing more until
// XdndFinished. A drag that kept taking the pointer's events after that
// swallowed motion the window under the pointer should have had, and
// forwarded it as another XdndPosition — which the target reads as a
// fresh drag and answers by putting its drop mark back up over a drop it
// has already taken. That is what left a band of the drop indicator
// across a dock host after a floating panel was dragged back into it, and
// what made moving the window the drag carried — closed by then — an
// error on a dead window.
func TestADroppedDragCarriesNothingMore(t *testing.T) {
	for _, tc := range []struct {
		name string
		drag x11Drag
		want bool
	}{
		{"no drag at all", x11Drag{}, false},
		{"running", x11Drag{active: true}, true},
		{"dropped, waiting for XdndFinished", x11Drag{active: true, dropped: true, released: true}, false},
		{"released with no target, before it ends", x11Drag{active: true, released: true}, true},
		{"finished", x11Drag{dropped: true}, false},
	} {
		if got := tc.drag.carries(); got != tc.want {
			t.Errorf("%s: carries() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
