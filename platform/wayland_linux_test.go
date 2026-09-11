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

func TestWaylandClipboardRoundtrip(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "clip", Width: 120, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Present(nil)
	_ = s.Poll()
	want := "uitoolkit-wl-clip-0.3.0"
	ClipboardSet(want)
	got := ClipboardGet()
	if got != want {
		t.Fatalf("CLIPBOARD %q want %q", got, want)
	}
	if prim := ClipboardPrimaryGet(); prim != want && prim != "" {
		// Primary is optional; Weston usually supports it.
		t.Logf("PRIMARY %q", prim)
	}
}

func TestWaylandDesktopChrome(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "chrome", Width: 140, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Present(nil)
	SetFullscreen(s, true)
	SetMaximized(s, true)
	SetFullscreen(s, false)
	if ime, ok := s.(IMESurface); ok {
		ime.SetIMECursor(8, 12, 2, 16)
	}
	if s.Scale() < 0.75 || s.Scale() > 4 {
		t.Fatalf("scale %v", s.Scale())
	}
}

func TestWaylandPresentSHMOverride(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	t.Setenv(EnvWaylandPresent, WaylandPresentSHM)
	// Force a fresh connection so the env is honored.
	if waylandLive() {
		t.Log("existing Wayland connection; present path may already be chosen")
	}
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-shm", Width: 120, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := paintengine2d.NewContext(s.Buffer())
	ctx.Clear(paintengine2d.RGB(0.2, 0.4, 0.1))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if waylandUsingDmabuf() {
		t.Fatal("UITK_WAYLAND_PRESENT=shm must not use dmabuf")
	}
}

func TestWaylandDmabufPathOrSkip(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	if !dmabufAllocatorOK() {
		t.Log("dmabuf allocator unavailable (no GBM/DRM, dma-heap, or udmabuf); shm fallback")
		t.Skip("no local dmabuf allocator")
	}
	t.Logf("dmabuf allocator: %s", dmabufAllocatorName())
	t.Setenv(EnvWaylandPresent, WaylandPresentDmabuf)
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-dmabuf", Width: 140, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := paintengine2d.NewContext(s.Buffer())
	ctx.Clear(paintengine2d.RGB(0.5, 0.1, 0.2))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Resize(160, 100); err != nil {
		t.Fatal(err)
	}
	ctx = paintengine2d.NewContext(s.Buffer())
	ctx.Clear(paintengine2d.RGB(0.1, 0.5, 0.3))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if !waylandUsingDmabuf() {
		t.Log("compositor did not advertise a usable linux-dmabuf format; shm fallback")
	}
}

func TestWaylandDmabufAllocatorProbe(t *testing.T) {
	ok := dmabufAllocatorOK()
	name := dmabufAllocatorName()
	if ok {
		t.Logf("dmabuf allocator works: %s", name)
		return
	}
	t.Log("dmabuf allocator unavailable (no GBM/DRM, dma-heap, or udmabuf); present will use wl_shm")
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
