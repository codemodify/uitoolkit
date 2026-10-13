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

	// What a backend calls immediately before it closes the device,
	// routed by the surface exactly as a real teardown routes it.
	platform.NotifyGPUClosing(win.surf)
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

	platform.NotifyGPUClosing(two.surf)
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
	platform.NotifyGPUClosing(win.surf) // no callback registered
	platform.NotifyGPUClosing(nil)
}

// **The callback outlives the run loop, and belongs to its own window.**
//
// Two reports, one cause: the process-wide hook could only answer for
// one owner. `Run` installed it and cleared it on the way out, so a
// window closed after the loop — `defer player.Close()`, which is
// exactly where a renderer is released — told nobody
// (media-player-video); and making a *second* Application took the
// first one's windows off it, so their callbacks stopped firing, which
// media-player-music reproduced on real GPU windows as 1,1 under
// v0.23.6 and 1,0 under v0.23.7.
//
// A surface has one owner for as long as it lives, so that is what the
// callback is registered against.
func TestTheClosingCallbackOutlivesTheRunLoopAndOtherApplications(t *testing.T) {
	platform.SetGPUClosing(nil)
	t.Cleanup(func() { platform.SetGPUClosing(nil) })
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	told := 0
	win.OnPaintDeviceClosing(func() { told++ })

	// A run loop that starts and stops, as an application's does.
	a.Post(a.Quit)
	a.Run()
	platform.NotifyGPUClosing(win.surf)
	if told != 1 {
		t.Errorf("the window was told %d times after Run returned, want once", told)
	}

	// And an unrelated Application, made while the first window is
	// still live, does not take it off them.
	b := New(Options{Headless: true, Scale: 1})
	other, err := b.NewWindow(platform.WindowOptions{Width: 120, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(other.Close)
	platform.NotifyGPUClosing(win.surf)
	if told != 2 {
		t.Errorf("the window was told %d times in all; a second Application took its callback", told)
	}

	// A closed window lets go of its surface, so the map does not hold
	// what the window system has dropped.
	win.Close()
	platform.NotifyGPUClosing(win.surf)
	if told != 2 {
		t.Errorf("a closed window was told again (%d in all)", told)
	}
}
