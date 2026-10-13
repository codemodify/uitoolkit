package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// **The before-close callback runs while the context is still there.**
//
// OnPaintDeviceChange says what happened, after it has. An application
// holding a renderer on the device — libmpv's, which requires the
// context it was created against to be current when its renderer is
// destroyed — had nowhere to let go: every backend closed the device
// and only then queued the change. Reported against media-player-video
// as a lifecycle gap.
//
// The ordering is what matters, so that is what this checks: the
// callback runs before the device the window reports goes away, not
// after.
func TestTheClosingCallbackRunsBeforeTheDeviceGoes(t *testing.T) {
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	var atClosing paintengine2d.Device
	win.OnPaintDeviceClosing(func() {
		order = append(order, "closing")
		atClosing = win.PaintDevice()
	})
	win.OnPaintDeviceChange(func() { order = append(order, "changed") })

	// The hook the backends call, routed by the application exactly as
	// a real teardown routes it.
	a.surfaceClosingGPU(win.surf)
	if win.onPaintDevice != nil {
		win.onPaintDevice()
	}

	if len(order) != 2 || order[0] != "closing" || order[1] != "changed" {
		t.Fatalf("the callbacks ran %v, want closing then changed", order)
	}
	if atClosing == nil {
		t.Error("the window reported no paint device while it was still closing one")
	}
}

// The hook reaches the window that owns the surface and no other.
func TestTheClosingCallbackReachesOnlyItsOwnWindow(t *testing.T) {
	a := New(Options{Headless: true, Scale: 1})
	one, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	two, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	var hits []string
	one.OnPaintDeviceClosing(func() { hits = append(hits, "one") })
	two.OnPaintDeviceClosing(func() { hits = append(hits, "two") })

	a.surfaceClosingGPU(two.surf)
	if len(hits) != 1 || hits[0] != "two" {
		t.Errorf("closing the second window's device ran %v", hits)
	}
}

// A window that registered nothing, and a surface belonging to no
// window, are both quiet rather than a panic.
func TestTheClosingHookIsQuietWithNothingRegistered(t *testing.T) {
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 100, Height: 60, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	a.surfaceClosingGPU(win.surf) // no callback registered
	a.surfaceClosingGPU(nil)
}

// **The callback outlives the run loop.**
//
// `Run` installed the hook the backends call and cleared it on the way
// out, so a window closed after the loop returned had its GPU device
// closed with nobody told — and an application that pumps instead of
// running was never told at all. media-player-video reproduced both:
// its own `main` defers `Player.Close`, which runs after `a.Run()`
// returns, and that is exactly where a GPU renderer would be released.
//
// A window's device can close whenever the window can, which is for as
// long as the application exists; so the hook is installed when the
// application is made.
func TestTheClosingCallbackOutlivesTheRunLoop(t *testing.T) {
	platform.SetGPUClosing(nil)
	a := New(Options{Headless: true, Scale: 1})
	t.Cleanup(func() { platform.SetGPUClosing(nil) })

	hook := platform.GPUClosing()
	if hook == nil {
		t.Fatal("making an application installed no closing hook; a window closed outside Run tells nobody")
	}
	win, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	told := 0
	win.OnPaintDeviceClosing(func() { told++ })

	// A run loop that starts and stops, as an application's does.
	a.Post(a.Quit)
	a.Run()

	// The backend closes the window's device now, after the loop: the
	// hook is still there and still finds the window.
	hook2 := platform.GPUClosing()
	if hook2 == nil {
		t.Fatal("the hook was cleared when Run returned; this is the gap that was reported")
	}
	hook2(win.surf)
	if told != 1 {
		t.Errorf("the window was told %d times after Run returned, want once", told)
	}
}
