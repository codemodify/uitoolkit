//go:build linux && cgo

package platform

import (
	"os"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
)

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

// A window a drag carries goes where the drag puts it, even when it is
// re-made before it first maps and even when that is half off the screen.
//
// A torn-off tab's window takes the toolkit's frame, so before its first
// frame it is re-made on an ARGB visual — a new X window — and the drag
// used to go on moving the old, destroyed one: the new window mapped
// where the desktop places new windows. And once mapped, a plain
// XMoveWindow is a request from the application, which KWin keeps inside
// the screen, so a window as big as the one it was torn from was pushed
// back on screen, nowhere near the pointer. The drag moves it as the user
// moving it (_NET_MOVERESIZE_WINDOW, source 2), which is left where it is
// put.
//
// It needs a compositing window manager (the e2e rig's nested KWin with its
// Xwayland: DISPLAY=$(cat tools/e2e/N/display)).
func TestX11CarriedWindowFollowsTheDrag(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	src, err := b.NewSurface(WindowOptions{Title: "source", Width: 300, Height: 200, X: 60, Y: 60, Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	NewPaintContext(src).Clear(paintengine2d.RGB(0.3, 0.3, 0.6))
	_ = src.Present(nil)
	c := src.(*x11Surface).conn

	// carried opens a window and hands it to a drag running from src, as
	// StartDrag leaves one (without the pointer grab a test cannot count
	// on), 20, 10 into it under the pointer.
	carried := func(t *testing.T) Surface {
		torn, err := b.NewSurface(WindowOptions{Title: "torn", Width: 400, Height: 260, Scale: 1})
		if err != nil {
			t.Fatal(err)
		}
		x11Mu.Lock()
		c.drag.active = true
		x11Mu.Unlock()
		t.Cleanup(func() {
			x11Mu.Lock()
			c.drag = x11Drag{}
			x11Mu.Unlock()
			torn.Close()
		})
		if !AttachToplevel(src, torn, 20, 10) {
			t.Fatal("AttachToplevel")
		}
		return torn
	}
	carry := func(rx, ry int) {
		x11Mu.Lock()
		c.dragMoveAttached(rx, ry)
		x11Mu.Unlock()
	}
	show := func(s Surface) {
		NewPaintContext(s).Clear(paintengine2d.RGB(0.6, 0.3, 0.3))
		_ = s.Present(nil)
	}
	settle := func(s Surface, wantX, wantY int) (int, int) {
		x, y := 0, 0
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
			s.Poll()
			src.Poll()
			var ok bool
			if x, y, ok = SurfacePosition(s); ok && x == wantX && y == wantY {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		return x, y
	}

	t.Run("re-made before it maps", func(t *testing.T) {
		if !c.composited {
			t.Skip("no compositing manager: the window is never re-made on an ARGB visual")
		}
		torn := carried(t)
		// The toolkit's frame asks for alpha before the first frame.
		torn.(*x11Surface).recreateOnVisual(true)
		carry(520, 330)
		show(torn)
		if x, y := settle(torn, 500, 320); x != 500 || y != 320 {
			t.Errorf("the re-made window mapped at %d,%d, the drag put it at 500,320", x, y)
		}
	})
	t.Run("dragged off the edge", func(t *testing.T) {
		torn := carried(t)
		carry(320, 330)
		show(torn)
		settle(torn, 300, 320)
		carry(-60+20, 300+10)
		if x, y := settle(torn, -60, 300); x != -60 || y != 300 {
			t.Errorf("dragged off the edge the window is at %d,%d, the drag put it at -60,300", x, y)
		}
	})
}
