//go:build linux && cgo

package app

import (
	"os"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestWaylandApplicationPaints(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no WAYLAND_DISPLAY")
	}
	a := New(Options{Look: style.DarkLook(), Backend: "wayland"})
	if a.BackendName() != "wayland" {
		t.Skip(a.BackendName())
	}
	w, err := a.NewWindow(platform.WindowOptions{Title: "wl-app", Width: 240, Height: 160})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewPanel("Wayland", widgets.NewLabel("paintengine2d"), widgets.NewButton("OK", nil)))
	img := w.Capture()
	if img == nil || img.Width < 1 {
		t.Fatal("capture")
	}
	if countOpaque(img, 20) < 100 {
		t.Fatalf("expected painted pixels, n=%d", countOpaque(img, 20))
	}
}
