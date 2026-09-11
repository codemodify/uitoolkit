//go:build linux && cgo

package platform

import (
	"os"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestWaylandProbeWithoutDisplay(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		t.Skip("WAYLAND_DISPLAY is set")
	}
	// Probe may still succeed if a default socket exists; either way
	// Default(true) must stay offscreen.
	if Default(true).Name() != "offscreen" {
		t.Fatal("headless")
	}
}

func TestWaylandSurfacePresent(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-test", Width: 160, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Buffer() == nil || s.Buffer().Width != 160 {
		t.Fatalf("buffer %+v", s.Buffer())
	}
	ctx := paintengine2d.NewContext(s.Buffer())
	ctx.Clear(paintengine2d.RGB(0.1, 0.3, 0.6))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	_ = s.Poll()
	s.SetTitle("uitoolkit-wl-test-2")
	if err := s.Resize(180, 110); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultPrefersWayland(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	if Default(false).Name() != "wayland" {
		t.Fatalf("auto %q", Default(false).Name())
	}
	if Default(true).Name() != "offscreen" {
		t.Fatal("headless must stay offscreen")
	}
}

func TestSelectWaylandFallsBack(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	if Select("wayland", false).Name() == "wayland" && !waylandProbe() {
		t.Fatal("wayland without compositor should not be selected")
	}
	if Select("offscreen", false).Name() != "offscreen" {
		t.Fatal("offscreen")
	}
}
