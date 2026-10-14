//go:build linux && cgo

package app

import (
	"os"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Closing a real GPU window through Window.Close calls back.
//
// The callback exists for a renderer that must let go of its textures
// while the context they live in is still current — libmpv's, which is
// why a video player asked for it. Everything about it was tested by
// firing the notification by hand at a window that was still open, and
// that test went on passing when v0.23.8 moved the registration: Close
// dropped it *before* closing the surface, so the teardown that follows
// had nothing to call and a real window called back zero times. Two
// applications reported it within the hour, both from real EGL windows.
//
// So this one closes a window the way an application does, and needs a
// display and a GPU to mean anything (tools/test-display.sh).
func TestARealGPUWindowCallsBackWhenItIsClosed(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display: this one is about a real GPU device closing")
	}
	t.Setenv(platform.EnvPaint, "gpu")
	a := New(Options{Look: style.DarkLook()})
	win, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 160})
	if err != nil {
		t.Skipf("no window here: %v", err)
	}
	if !win.UsesGPU() {
		win.Close()
		t.Skip("this display gave a CPU device; there is no GPU teardown to watch")
	}

	called := 0
	var had paintengine2d.Device
	win.OnPaintDeviceClosing(func() {
		called++
		had = win.PaintDevice()
	})

	win.Close()

	if called != 1 {
		t.Fatalf("closing a GPU window called the closing callback %d times, want 1", called)
	}
	if had == nil {
		t.Error("the callback ran with no device: it is for letting go of one")
	}
}
