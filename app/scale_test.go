package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

func TestApplicationScaleRebuildsLook(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	if a.Scale() != 2 {
		t.Fatalf("scale %v", a.Scale())
	}
	if a.Look().Metrics().ControlH != style.DefaultMetrics().ControlH*2 {
		t.Fatalf("metrics %v", a.Look().Metrics().ControlH)
	}
	if a.BackendName() != "offscreen" {
		t.Fatalf("backend %q", a.BackendName())
	}
}

func TestApplicationExplicitOffscreenBackend(t *testing.T) {
	a := New(Options{Headless: false, Backend: "offscreen"})
	if a.BackendName() != "offscreen" {
		t.Fatalf("backend %q", a.BackendName())
	}
}

func TestWindowScaleAtOpenMatchesSurface(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	if w.Scale() != w.Surface().Scale() {
		t.Fatalf("window scale %v != surface %v", w.Scale(), w.Surface().Scale())
	}
	if w.Scale() != a.Scale() {
		t.Fatalf("window %v app %v", w.Scale(), a.Scale())
	}
}

func TestExplicitScaleIsNotShrunkBySurface(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	if w.Scale() != 2 {
		t.Fatalf("explicit 2× dropped to %v (surface is %v)", w.Scale(), w.Surface().Scale())
	}
}

func TestSetLookKeepsDisplayScale(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	a.SetLook(style.LightLook())
	if a.Look().Metrics().FontSize != style.DefaultMetrics().FontSize*2 {
		t.Fatalf("theme swap dropped scale, font %v", a.Look().Metrics().FontSize)
	}
	if a.Look().Name() != "light" {
		t.Fatalf("look %q", a.Look().Name())
	}
}
