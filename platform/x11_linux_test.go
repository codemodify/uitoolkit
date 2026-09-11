//go:build linux && cgo

package platform

import (
	"bytes"
	"os"
	"testing"

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
	ctx := paintengine2d.NewContext(img)
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
