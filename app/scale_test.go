package app

import (
	"testing"

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
