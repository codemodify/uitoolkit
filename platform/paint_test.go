package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestPaintPrefDefaultAuto(t *testing.T) {
	t.Setenv(EnvPaint, "")
	if PaintPref() != paintengine2d.PaintAuto {
		t.Fatalf("empty UITK_PAINT want auto, got %q", PaintPref())
	}
	if !WantGPU() {
		t.Fatal("auto should try GPU")
	}
	t.Setenv(EnvPaint, "cpu")
	if PaintPref() != paintengine2d.PaintCPU {
		t.Fatal(PaintPref())
	}
	if WantGPU() {
		t.Fatal("cpu must not request GPU")
	}
	t.Setenv(EnvPaint, "gpu")
	if PaintPref() != paintengine2d.PaintGPU || !WantGPU() {
		t.Fatal("gpu")
	}
	t.Setenv(EnvPaint, "AUTO")
	if PaintPref() != paintengine2d.PaintAuto {
		t.Fatal(PaintPref())
	}
}

func TestNewPaintContextOffscreen(t *testing.T) {
	s := NewOffscreen(WindowOptions{Width: 16, Height: 12})
	ctx := NewPaintContext(s)
	if ctx == nil {
		t.Fatal("ctx")
	}
	ctx.Clear(paintengine2d.RGB(1, 0, 0))
	img := s.Buffer()
	if img == nil || len(img.Pix) < 4 {
		t.Fatal("buffer")
	}
	if img.Pix[0] == 0 && img.Pix[3] == 0 {
		t.Fatal("expected painted pixels")
	}
	if SurfaceUsesGPU(s) {
		t.Fatal("offscreen is CPU")
	}
}

func TestOpenSurfaceCPUPref(t *testing.T) {
	s, err := paintengine2d.OpenSurfacePref(8, 8, paintengine2d.PaintCPU)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Kind() != paintengine2d.BackendCPU {
		t.Fatalf("kind %s", s.Kind())
	}
}

func TestOpenSurfaceAutoFallsBackWithoutEGL(t *testing.T) {
	t.Setenv(EnvPaint, "auto")
	s, err := paintengine2d.OpenSurface(8, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Kind() != paintengine2d.BackendCPU && s.Kind() != paintengine2d.BackendGPU {
		t.Fatalf("kind %s", s.Kind())
	}
	if !paintengine2d.GPUAvailable() && s.Kind() != paintengine2d.BackendCPU {
		t.Fatal("auto without EGL must be CPU")
	}
}
