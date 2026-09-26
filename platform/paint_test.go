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

func TestWantSceneDefaultOn(t *testing.T) {
	t.Setenv(EnvScene, "")
	if !WantScene() {
		t.Fatal("default scene on")
	}
	t.Setenv(EnvScene, "off")
	if WantScene() {
		t.Fatal("off")
	}
	t.Setenv(EnvScene, "immediate")
	if WantScene() {
		t.Fatal("immediate")
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

// A saved preference decides the paint device when the environment says
// nothing, and UITK_PAINT beats it whenever it does — the way UITK_THEME
// beats the saved theme. Settings writes the file; a person debugging a
// driver writes the variable, and the variable wins for that one run
// without the file being rewritten behind their back.
func TestUITKPaintBeatsTheSavedPreference(t *testing.T) {
	t.Cleanup(func() { SetPaintPref("") })
	t.Setenv(EnvPaint, "")

	SetPaintPref("")
	if PaintPref() != paintengine2d.PaintAuto {
		t.Fatalf("nobody has said anything: want auto, got %q", PaintPref())
	}
	SetPaintPref("cpu")
	if PaintPref() != paintengine2d.PaintCPU || WantGPU() {
		t.Fatalf("the saved cpu preference should stand: %q", PaintPref())
	}
	SetPaintPref("gpu")
	if PaintPref() != paintengine2d.PaintGPU || !WantGPU() {
		t.Fatalf("the saved gpu preference should stand: %q", PaintPref())
	}
	// And the environment takes it back.
	t.Setenv(EnvPaint, "cpu")
	if PaintPref() != paintengine2d.PaintCPU || WantGPU() {
		t.Fatalf("UITK_PAINT=cpu must beat a saved gpu: %q", PaintPref())
	}
	if env, ok := PaintPrefEnv(); !ok || env != paintengine2d.PaintCPU {
		t.Fatalf("PaintPrefEnv %q %v", env, ok)
	}
	// "auto" in the environment is a choice too: it beats a saved cpu.
	SetPaintPref("cpu")
	t.Setenv(EnvPaint, "auto")
	if PaintPref() != paintengine2d.PaintAuto {
		t.Fatalf("UITK_PAINT=auto must beat a saved cpu: %q", PaintPref())
	}
	// An empty variable is not a choice: the saved one comes back.
	t.Setenv(EnvPaint, "  ")
	if PaintPref() != paintengine2d.PaintCPU {
		t.Fatalf("an empty UITK_PAINT leaves the saved cpu in force: %q", PaintPref())
	}
	if _, ok := PaintPrefEnv(); ok {
		t.Fatal("blank UITK_PAINT should not count as set")
	}
	// Forgetting the saved preference is not the same as saving "cpu":
	// paintengine2d reads anything it does not know as cpu, and this
	// wrapper must not turn "nobody has said" into a demand.
	SetPaintPref("")
	if PaintPref() != paintengine2d.PaintAuto {
		t.Fatalf("a forgotten preference is auto again, got %q", PaintPref())
	}
}

// What a surface is really painting through is not what was asked for:
// an offscreen window is on the CPU whatever the preference says.
func TestSurfaceBackendSaysWhatItReallyIs(t *testing.T) {
	t.Cleanup(func() { SetPaintPref("") })
	t.Setenv(EnvPaint, "gpu")
	s := NewOffscreen(WindowOptions{Width: 8, Height: 8})
	if got := SurfaceBackend(s); got != paintengine2d.PaintCPU {
		t.Fatalf("an offscreen surface paints on the cpu, not %q", got)
	}
	if SurfaceBackend(nil) != paintengine2d.PaintCPU {
		t.Fatal("no surface, nothing presented: cpu")
	}
}
