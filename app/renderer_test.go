package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// The saved renderer reaches the surfaces this application makes — but
// only the ones it has not made yet, which is the whole point of the
// control Settings puts on the page. An application reads look.json at
// start-up, pushes the preference into platform, and every NewWindow
// after that binds its device by it.
func TestApplicationTakesTheSavedRenderer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(RendererEnv, "")
	t.Cleanup(func() { platform.SetPaintPref("") })

	ap := style.DefaultAppearance()
	ap.Renderer = style.RendererCPU
	if err := style.SaveAppearance(ap); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := a.Renderer(); got != style.RendererCPU {
		t.Fatalf("the application asks for %q, look.json says cpu", got)
	}
	if got := platform.PaintPref(); got != "cpu" {
		t.Fatalf("platform was told %q", got)
	}
	// Appearance reports it back, the way it reports the frame and the
	// caption buttons: a preference a look cannot carry.
	if got := a.Appearance().Renderer; got != style.RendererCPU {
		t.Fatalf("Appearance says %q", got)
	}

	// Applying is what Settings' Apply does in the running app, and what
	// every other app's look watcher does when the file changes.
	next := ap
	next.Renderer = style.RendererGPU
	a.ApplyAppearance(next)
	if got := a.Renderer(); got != style.RendererGPU {
		t.Fatalf("after ApplyAppearance the request is %q", got)
	}
	if got := platform.PaintPref(); got != "gpu" {
		t.Fatalf("platform was told %q", got)
	}

	// And what it is actually painting with is a different question with
	// a different answer: a headless window is a pixmap, whatever was
	// asked for. This is what the chooser reports rather than echoing
	// the preference back at the user.
	w, err := a.NewWindow(platform.WindowOptions{Title: "r", Width: 40, Height: 30, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if got := w.PaintBackend(); got != style.RendererCPU {
		t.Fatalf("an offscreen window paints on %q", got)
	}
	if got := a.PaintBackend(); got != style.RendererCPU {
		t.Fatalf("the application is painting on %q while asking for gpu", got)
	}
}

// UITK_PAINT beats the file, as UITK_THEME beats the saved theme, and it
// beats it where it matters: at the surface. The application still
// reports (and Settings still writes) the user's own saved choice, so a
// debugging session under the variable cannot bake itself into
// everybody's look.json.
func TestUITKPaintBeatsTheSavedRenderer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(RendererEnv, "cpu")
	t.Cleanup(func() { platform.SetPaintPref("") })

	ap := style.DefaultAppearance()
	ap.Renderer = style.RendererGPU
	if err := style.SaveAppearance(ap); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := a.Renderer(); got != style.RendererGPU {
		t.Fatalf("the saved request is still gpu, not %q", got)
	}
	if got := platform.PaintPref(); got != "cpu" {
		t.Fatalf("UITK_PAINT=cpu must decide the device, got %q", got)
	}
	if platform.WantGPU() {
		t.Fatal("UITK_PAINT=cpu must not try EGL")
	}
	env, ok := RendererOverride()
	if !ok || env != style.RendererCPU {
		t.Fatalf("RendererOverride %q %v", env, ok)
	}
	// Applying another preference does not take the override away.
	next := ap
	next.Renderer = style.RendererAuto
	a.ApplyAppearance(next)
	if got := platform.PaintPref(); got != "cpu" {
		t.Fatalf("the environment still wins after Apply, got %q", got)
	}
	if got := a.Renderer(); got != style.RendererAuto {
		t.Fatalf("the applied request is %q", got)
	}
}

// With nothing saved and nothing in the environment the toolkit is where
// it always was: auto, which is GPU where EGL starts.
func TestRendererDefaultIsAuto(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(RendererEnv, "")
	t.Cleanup(func() { platform.SetPaintPref("") })
	a := New(Options{Headless: true, Scale: 1})
	if got := a.Renderer(); got != style.RendererAuto {
		t.Fatalf("default request %q", got)
	}
	if got := platform.PaintPref(); got != "auto" {
		t.Fatalf("platform default %q", got)
	}
	if _, ok := RendererOverride(); ok {
		t.Fatal("no UITK_PAINT, no override")
	}
}
