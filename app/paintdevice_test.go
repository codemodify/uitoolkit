package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A window's paint device is closed and made again when the window changes
// visual, when a resize of it fails, and when the GPU is lost mid-run and
// painting falls back to the CPU. Every one of those paths dropped it in
// silence, so an application holding a texture on it — a video frame it
// renders into — was left with a GL name that means nothing.

func deviceWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("something"))
	a.PumpOnce()
	return a, w
}

func TestOnPaintDeviceChangeReachesTheApplication(t *testing.T) {
	_, w := deviceWindow(t)
	n := 0
	w.OnPaintDeviceChange(func() { n++ })
	if n != 0 {
		t.Fatalf("installing the callback called it %d times", n)
	}
	w.dispatch(platform.Event{Kind: platform.EventPaintDevice})
	if n != 1 {
		t.Errorf("the event called back %d times, want 1", n)
	}
	w.dispatch(platform.Event{Kind: platform.EventPaintDevice})
	if n != 2 {
		t.Errorf("a second event called back %d times in total, want 2", n)
	}
}

// The window repaints whatever the application does with the news: the widgets
// have to draw against the device that is there now.
func TestAChangedPaintDeviceRepaintsTheWindow(t *testing.T) {
	_, w := deviceWindow(t)
	w.dirty.Reset()
	w.full = false
	if w.needsPaint() {
		t.Fatal("the window still wants painting before the event")
	}
	w.dispatch(platform.Event{Kind: platform.EventPaintDevice})
	if !w.needsPaint() {
		t.Error("a replaced paint device did not invalidate the window")
	}
}

// PaintDevice and UsesGPU are how the application asks what it is painting
// through — before it builds a renderer, and again after the news.
func TestAWindowSaysWhatItPaintsThrough(t *testing.T) {
	_, w := deviceWindow(t)
	if w.PaintDevice() == nil {
		t.Error("a window reports no paint device at all")
	}
	// A headless window is on the CPU, which is the answer an application
	// needs to hear before it builds a GPU renderer it cannot feed.
	if w.UsesGPU() {
		t.Error("a headless window claims a GPU")
	}
	var nilWin *Window
	if nilWin.PaintDevice() != nil || nilWin.UsesGPU() {
		t.Error("a nil window answered")
	}
}
