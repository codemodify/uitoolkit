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
	ctx := NewPaintContext(s)
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

func TestWaylandAutoPresentIsSHM(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	t.Setenv(EnvPaint, paintengine2d.PaintCPU)
	t.Setenv(EnvWaylandPresent, WaylandPresentAuto)
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-auto", Width: 120, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := NewPaintContext(s)
	ctx.Clear(paintengine2d.RGB(0.3, 0.2, 0.1))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if SurfaceUsesGPU(s) {
		t.Fatal("UITK_PAINT=cpu must not bind EGL")
	}
	if waylandUsingDmabuf() {
		t.Fatal("auto must present via wl_shm (dmabuf is opt-in)")
	}
}

func TestWaylandPresentOpaqueColor(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	t.Setenv(EnvPaint, paintengine2d.PaintCPU)
	t.Setenv(EnvWaylandPresent, WaylandPresentSHM)
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-opaque", Width: 64, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ws, ok := s.(*wlSurface)
	if !ok {
		t.Fatal("wlSurface")
	}
	ctx := NewPaintContext(s)
	ctx.Clear(paintengine2d.RGB(0.25, 0.50, 1.0))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	_ = s.Poll()
	var pix []byte
	for i := 0; i < 2; i++ {
		if p := ws.slotBGRA(i); len(p) >= 4 && p[3] != 0 {
			pix = p
			break
		}
	}
	if len(pix) < 4 {
		// Slot may still be the one just attached; read whichever is mapped.
		pix = ws.slotBGRA(0)
		if len(pix) < 4 {
			pix = ws.slotBGRA(1)
		}
	}
	if len(pix) < 4 {
		t.Fatal("no present slot")
	}
	if destAlphaAllZero(pix, ws.slots[0].stride) && destAlphaAllZero(ws.slotBGRA(1), 64) {
		t.Fatal("presented buffer has zero alpha — compositor would draw a transparent window")
	}
	found := false
	limit := len(pix)
	if limit > 256 {
		limit = 256
	}
	for i := 3; i < limit; i += 4 {
		if pix[i] == 0xff && (pix[i-3] != 0 || pix[i-2] != 0 || pix[i-1] != 0) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected opaque XRGB pixels, first=%v", pix[:4])
	}
}

func TestWaylandPresentSHMOverride(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	t.Setenv(EnvPaint, paintengine2d.PaintCPU)
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
	ctx := NewPaintContext(s)
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
	t.Setenv(EnvPaint, paintengine2d.PaintCPU)
	t.Setenv(EnvWaylandPresent, WaylandPresentDmabuf)
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-dmabuf", Width: 140, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := NewPaintContext(s)
	ctx.Clear(paintengine2d.RGB(0.5, 0.1, 0.2))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Resize(160, 100); err != nil {
		t.Fatal(err)
	}
	ctx = NewPaintContext(s)
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

func TestWaylandEGLOrCPUFallback(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no Wayland compositor")
	}
	t.Setenv(EnvPaint, paintengine2d.PaintAuto)
	b := WaylandBackend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-wl-egl", Width: 140, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := NewPaintContext(s)
	if ctx == nil {
		t.Fatal("paint context")
	}
	ctx.Clear(paintengine2d.RGB(0.15, 0.35, 0.55))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if SurfaceUsesGPU(s) {
		t.Log("present: wl_egl_window + eglSwapBuffers")
	} else {
		t.Log("present: v0.4.1 opaque wl_shm (EGL unavailable)")
	}
	if err := s.Resize(160, 100); err != nil {
		t.Fatal(err)
	}
	ctx = NewPaintContext(s)
	ctx.Clear(paintengine2d.RGB(0.4, 0.2, 0.1))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
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
