//go:build linux && cgo

package platform

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
)

func TestMapXKeySym(t *testing.T) {
	if MapXKeySym(0xff1b) != KeyEscape {
		t.Fatal("Escape")
	}
	if MapXKeySym(0x0061) != KeyA || MapXKeySym(0x0041) != KeyA {
		t.Fatal("A")
	}
	if MapXKeySym(0xff0d) != KeyReturn {
		t.Fatal("Return")
	}
}

func TestX11SurfacePresentAndClose(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-test", Width: 160, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	img := s.Buffer()
	if img == nil || img.Width != 160 {
		t.Fatalf("buffer %+v", img)
	}
	ctx := NewPaintContext(s)
	ctx.Clear(paintengine2d.RGB(0.2, 0.4, 0.8))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	s.SetTitle("uitoolkit-test-2")
	if s.Title() != "uitoolkit-test-2" {
		t.Fatalf("title %q", s.Title())
	}
	if err := s.Resize(200, 120); err != nil {
		t.Fatal(err)
	}
	w, h := s.Size()
	if w != 200 || h != 120 {
		t.Fatalf("size %d×%d", w, h)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if !s.Closed() {
		t.Fatal("expected closed")
	}
}

func TestX11MultiWindowClose(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	a, err := b.NewSurface(WindowOptions{Title: "a", Width: 120, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.NewSurface(WindowOptions{Title: "b", Width: 140, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Present(nil)
	_ = c.Present(nil)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if c.Closed() {
		t.Fatal("second window should stay open")
	}
	_ = c.Present(nil)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestX11ClipboardRoundtrip(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	want := "uitoolkit-clip-0.1.8"
	ClipboardSet(want)
	got := ClipboardGet()
	if got != want {
		t.Fatalf("CLIPBOARD %q want %q", got, want)
	}
	if prim := ClipboardPrimaryGet(); prim != want {
		t.Fatalf("PRIMARY %q want %q", prim, want)
	}
}

func TestX11DetectScalePositive(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	s := DetectScale()
	if s < 0.75 || s > 4 {
		t.Fatalf("scale %v", s)
	}
}

func TestX11INCRClipboardRoundtrip(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Setenv("UITK_X11_INCR_THRESHOLD", "128")
	want := string(bytes.Repeat([]byte("incr-"), 80))
	ClipboardSet(want)
	got := ClipboardGet()
	if got != want {
		t.Fatalf("INCR CLIPBOARD len=%d want %d", len(got), len(want))
	}
	if prim := ClipboardPrimaryGet(); prim != want {
		t.Fatalf("INCR PRIMARY len=%d", len(prim))
	}
}

func TestX11EWMHAndIMECursor(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	s, err := b.NewSurface(WindowOptions{Title: "ewmh", Width: 180, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Present(nil)
	SetFullscreen(s, true)
	SetMaximized(s, true)
	SetFullscreen(s, false)
	SetMaximized(s, false)
	if ime, ok := s.(IMESurface); ok {
		ime.SetIMECursor(10, 20, 2, 16)
	}
	SetCursor(s, CursorColResize)
	SetCursor(s, CursorDefault)
}

func TestX11EGLOrCPUFallback(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Setenv(EnvPaint, paintengine2d.PaintAuto)
	b := X11Backend{}
	s, err := b.NewSurface(WindowOptions{Title: "uitoolkit-x11-egl", Width: 160, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := NewPaintContext(s)
	if ctx == nil {
		t.Fatal("paint context")
	}
	ctx.Clear(paintengine2d.RGB(0.2, 0.3, 0.5))
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	if SurfaceUsesGPU(s) {
		t.Log("present: X11 EGL + eglSwapBuffers")
	} else {
		t.Log("present: XPutImage (EGL unavailable)")
	}
}

func TestX11IMEEventsOnFocusOut(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	s, err := b.NewSurface(WindowOptions{Title: "ime", Width: 160, Height: 90})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_ = s.Present(nil)
	_ = s.Poll()
}

func TestX11SupportedBPP(t *testing.T) {
	for _, bpp := range []int{32, 16} {
		if !x11SupportedBPP(bpp) {
			t.Fatalf("%d bpp should be supported", bpp)
		}
	}
	// 8-bit PseudoColor and packed 24 present nothing rather than
	// writing past the image's real bytes_per_line.
	for _, bpp := range []int{0, 1, 8, 24, 64} {
		if x11SupportedBPP(bpp) {
			t.Fatalf("%d bpp should be refused", bpp)
		}
	}
}

// X sends a press and a release for every wheel notch; only the press may
// scroll. The release used to scroll a second time (two steps per notch).
func TestX11WheelScrollsOncePerNotch(t *testing.T) {
	for btn := 4; btn <= 7; btn++ {
		if ev := x11ButtonEvents(btn, false, paintengine2d.Pt(1, 2), 0); len(ev) != 1 || ev[0].Kind != EventScroll {
			t.Fatalf("button %d press: %+v", btn, ev)
		}
		if ev := x11ButtonEvents(btn, true, paintengine2d.Pt(1, 2), 0); len(ev) != 0 {
			t.Fatalf("button %d release must not scroll: %+v", btn, ev)
		}
	}
	if ev := x11ButtonEvents(1, true, paintengine2d.Pt(1, 2), 0); len(ev) != 1 || ev[0].Kind != EventMouseUp {
		t.Fatalf("left release: %+v", ev)
	}
}

// A window the client places maps where it was put, not where the window
// manager's own policy would have put it — whether the position came with
// the window or from a Move before its first frame. WM_NORMAL_HINTS is all
// a window manager reads at map time, and the size hints written after
// the window was made used to replace the position flags with nothing, so
// KWin centred or cascaded the window instead: a torn-off tab appeared at
// the desktop's spot for new windows rather than under the pointer.
//
// It needs a window manager to mean anything (the e2e rig's nested KWin
// with its Xwayland: DISPLAY=$(cat tools/e2e/N/display)).
func TestX11PlacedWindowMapsWhereItWasPut(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := X11Backend{}
	for _, tc := range []struct {
		name  string
		opts  WindowOptions
		moveX int
		moveY int
	}{
		{"opened at a position", WindowOptions{Title: "placed", Width: 220, Height: 140, X: 310, Y: 230, Scale: 1}, 0, 0},
		{"moved before it maps", WindowOptions{Title: "moved", Width: 220, Height: 140, Scale: 1}, 370, 190},
	} {
		s, err := b.NewSurface(tc.opts)
		if err != nil {
			t.Fatal(err)
		}
		wantX, wantY := tc.opts.X, tc.opts.Y
		if tc.moveX != 0 || tc.moveY != 0 {
			MoveSurface(s, tc.moveX, tc.moveY)
			wantX, wantY = tc.moveX, tc.moveY
		}
		NewPaintContext(s).Clear(paintengine2d.RGB(0.2, 0.5, 0.3))
		if err := s.Present(nil); err != nil {
			t.Fatal(err)
		}
		x, y, ok := 0, 0, false
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
			s.Poll()
			if x, y, ok = SurfacePosition(s); ok && x == wantX && y == wantY {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !ok || x != wantX || y != wantY {
			t.Errorf("%s: the window mapped at %d,%d (known=%v), put at %d,%d", tc.name, x, y, ok, wantX, wantY)
		}
		s.Close()
	}
}
