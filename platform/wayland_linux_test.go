//go:build linux && cgo

package platform

import (
	"math"
	"os"
	"testing"
	"time"

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
	// The buffer is in device pixels: 160 at the output's scale (280 at
	// 1.75), once the connection knows it.
	if want := int(math.Ceil(float64(float32(160) * s.Scale()))); s.Buffer() == nil || s.Buffer().Width != want {
		t.Fatalf("buffer %+v, want %d wide at scale %v", s.Buffer(), want, s.Scale())
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
	SetCursor(s, CursorColResize)
	SetCursor(s, CursorDefault)
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
	// A real compositor can hold the first frame back until its configure
	// or frame callback: poll until a slot is mapped, for up to a second.
	var pix []byte
	for deadline := time.Now().Add(time.Second); ; {
		_ = s.Poll()
		for i := 0; i < 2 && len(pix) < 4; i++ {
			if p := ws.slotBGRA(i); len(p) >= 4 && p[3] != 0 {
				pix = p
			}
		}
		if len(pix) >= 4 || time.Now().After(deadline) {
			break
		}
		ws.Wait(10 * time.Millisecond)
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

// A frame the compositor could not take yet (frame callback outstanding,
// every buffer busy) must not lose its damage: it is folded into the next
// present. Dropping it left pressed buttons and popups off screen.
func TestWaylandDeferredDamageIsKept(t *testing.T) {
	s := &wlSurface{}
	a := paintengine2d.XYWH(0, 0, 10, 10)
	b := paintengine2d.XYWH(50, 0, 10, 10)
	s.deferDamage([]paintengine2d.Rect{a})
	s.deferDamage([]paintengine2d.Rect{b})
	c := paintengine2d.XYWH(90, 0, 5, 5)
	rects, full := s.takeDeferred([]paintengine2d.Rect{c}, false)
	if full || len(rects) != 3 {
		t.Fatalf("deferred a+b plus current c: got %v full=%v", rects, full)
	}
	if s.pendAny || len(s.pendRects) != 0 {
		t.Fatal("takeDeferred must clear the pending damage")
	}
	// A deferred full frame wins over everything after it.
	s.deferDamage(nil)
	s.deferDamage([]paintengine2d.Rect{a})
	if _, full := s.takeDeferred([]paintengine2d.Rect{c}, false); !full {
		t.Fatal("a deferred full frame must present the whole surface")
	}
	// Flushing presents only what was deferred.
	s.deferDamage([]paintengine2d.Rect{b})
	rects, full = s.takeDeferred(nil, true)
	if full || len(rects) != 1 || rects[0] != b {
		t.Fatalf("flush-only: got %v full=%v", rects, full)
	}
}

// Each shm buffer remembers what was presented through the others since it
// last held a frame; a buffer reused frames later must get all of it.
func TestWaylandSlotDamageHistory(t *testing.T) {
	s := &wlSurface{}
	r1 := paintengine2d.XYWH(0, 0, 10, 10)
	r2 := paintengine2d.XYWH(20, 0, 10, 10)
	s.noteSlotPresented(0, []paintengine2d.Rect{r1}) // slot 0 valid
	s.noteSlotPresented(1, []paintengine2d.Rect{r2}) // slot 1 valid; slot 0 misses r2
	if !s.slots[0].valid || len(s.slots[0].stale) != 1 || s.slots[0].stale[0] != r2 {
		t.Fatalf("slot 0 history %+v", s.slots[0])
	}
	if s.slots[2].valid || len(s.slots[2].stale) != 0 {
		t.Fatal("a never-used buffer stays invalid (full copy) and keeps no history")
	}
	s.noteSlotPresented(0, []paintengine2d.Rect{r1})
	if len(s.slots[0].stale) != 0 || len(s.slots[1].stale) != 1 {
		t.Fatalf("presenting into slot 0 clears its history and extends slot 1's: %+v / %+v", s.slots[0], s.slots[1])
	}
	for i := 0; i < maxSlotStale+1; i++ {
		s.noteSlotPresented(0, []paintengine2d.Rect{r1})
	}
	if s.slots[1].valid {
		t.Fatal("a buffer whose history overflowed must be rewritten in full")
	}
}
