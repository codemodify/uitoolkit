package app

import (
	"runtime"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func secureWindow(t *testing.T) (*Application, *Window, *platform.Offscreen) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewSecretField("Passphrase"))
	a.PumpOnce()
	o, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Fatalf("surface %T", w.surf)
	}
	return a, w, o
}

// The request reaches the surface, both ways, and the window reports
// what the surface said rather than what was asked.
func TestSecureInputReachesTheSurface(t *testing.T) {
	_, w, o := secureWindow(t)
	if o.SecureInput() {
		t.Fatal("secure input is on before anyone asked")
	}
	if !w.SetSecureInput(true) || !o.SecureInput() {
		t.Fatal("SetSecureInput(true) did not take")
	}
	if !w.SetSecureInput(false) || o.SecureInput() {
		t.Fatal("SetSecureInput(false) did not take")
	}
}

func TestExcludeFromCaptureReachesTheSurface(t *testing.T) {
	_, w, o := secureWindow(t)
	if !w.SetExcludeFromCapture(true) || !o.ExcludeFromCapture() {
		t.Fatal("SetExcludeFromCapture(true) did not take")
	}
	if !w.SetExcludeFromCapture(false) || o.ExcludeFromCapture() {
		t.Fatal("SetExcludeFromCapture(false) did not take")
	}
}

// A closed window asks nothing, and says so rather than reporting a
// protection it did not get.
func TestSecureInputOnAClosedWindow(t *testing.T) {
	_, w, _ := secureWindow(t)
	w.Close()
	if w.SetSecureInput(true) {
		t.Fatal("a closed window claims secure input")
	}
	if w.SetExcludeFromCapture(true) {
		t.Fatal("a closed window claims capture exclusion")
	}
}

// The Available answers are facts about the build, and an application
// reads them to decide what to promise the user.
func TestAvailabilityIsHonestAboutThisPlatform(t *testing.T) {
	// They must not panic and must be stable within a run.
	if platform.SecureInputAvailable() != platform.SecureInputAvailable() {
		t.Fatal("SecureInputAvailable changes its mind")
	}
	if platform.CaptureExclusionAvailable() != platform.CaptureExclusionAvailable() {
		t.Fatal("CaptureExclusionAvailable changes its mind")
	}
	// Linux cannot keep a window out of a capture, and says so rather
	// than letting an application promise it; Windows and macOS can.
	switch runtime.GOOS {
	case "linux":
		if platform.CaptureExclusionAvailable() {
			t.Fatal("this build claims capture exclusion on Linux")
		}
	case "windows":
		if platform.SecureInputAvailable() {
			t.Fatal("this build claims secure input on Windows")
		}
		if !platform.CaptureExclusionAvailable() {
			t.Fatal("Windows can exclude from capture and says it cannot")
		}
	case "darwin":
		if !platform.SecureInputAvailable() || !platform.CaptureExclusionAvailable() {
			t.Fatal("macOS has both and says it has not")
		}
	}
}
