package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// A window dragged from a HiDPI monitor to a plain one changes scale
// without changing size: it is the same number of logical pixels across,
// and only the buffer behind it is different. Until EventScale existed,
// syncScale was reached only from EventResize, so this window went on
// drawing a look built for the old scale — every metric, every font, every
// border a fraction wrong, until something else happened to resize it.
//
// This is macOS's ordinary case (NSWindowDidChangeBackingProperties fires
// with the window the same size in points), so it had to be answerable
// before a macOS backend, not after.
func TestScaleChangeWithNoResizeRebuildsTheLook(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	win := newScaledWindow(a, 2, 200, 120)
	if got := win.Scale(); got != 2 {
		t.Fatalf("the window opened at scale %v", got)
	}
	before := win.Look()

	// The monitor under it changes. No resize: the window is still 200
	// by 120 logical pixels.
	win.surf.(*scaledSurface).scale = 1
	win.dispatch(platform.Event{Kind: platform.EventScale})

	if got := win.Scale(); got != 1 {
		t.Fatalf("the window is still drawing at scale %v", got)
	}
	if win.Look() == before {
		t.Fatal("the look was not rebuilt for the new scale")
	}
}

// The event is safe to send when nothing changed — a backend that cannot
// tell whether the scale really moved may send it anyway, and some will.
func TestScaleChangeToTheSameScaleIsIgnored(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	win := newScaledWindow(a, 2, 200, 120)
	before := win.Look()
	win.dispatch(platform.Event{Kind: platform.EventScale})
	if win.Look() != before {
		t.Fatal("a scale change to the same scale rebuilt the look")
	}
}

// The offscreen backend queues the event, so the case above can be
// reached through a backend rather than only by hand — and it queues no
// resize with it, because that is the case that was being missed.
func TestOffscreenSimulateScaleQueuesTheEvent(t *testing.T) {
	o := platform.NewOffscreen(platform.WindowOptions{Width: 200, Height: 120})
	drain(o)
	o.SimulateScale(2)

	var scales, resizes int
	for _, ev := range o.Poll() {
		switch ev.Kind {
		case platform.EventScale:
			scales++
		case platform.EventResize:
			resizes++
		}
	}
	if scales != 1 {
		t.Errorf("SimulateScale queued %d scale events, want 1", scales)
	}
	if resizes != 0 {
		t.Errorf("SimulateScale queued %d resizes; the window did not change size", resizes)
	}
}

func drain(o *platform.Offscreen) { _ = o.Poll() }
