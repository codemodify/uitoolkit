package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// scaledSurface is an offscreen surface that reports a display scale, the
// way a Wayland surface on a HiDPI output does.
type scaledSurface struct {
	*platform.Offscreen
	scale float32
}

func (s *scaledSurface) Scale() float32 { return s.scale }

func newScaledWindow(a *Application, scale float32, w, h int) *Window {
	surf := &scaledSurface{
		Offscreen: platform.NewOffscreen(platform.WindowOptions{Width: w, Height: h}),
		scale:     scale,
	}
	win := newWindow(a, surf, platform.WindowOptions{Width: w, Height: h})
	a.mu.Lock()
	a.windows = append(a.windows, win)
	a.mu.Unlock()
	return win
}

// With no explicit scale, each window takes its metrics from its own
// surface, so a window on a HiDPI output is not stuck at the scale the
// process happened to detect at startup.
func TestWindowTakesScaleFromSurface(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	if a.Scale() != 1 {
		t.Fatalf("app scale %v", a.Scale())
	}
	hidpi := newScaledWindow(a, 2, 200, 120)
	lodpi := newScaledWindow(a, 1, 200, 120)
	if hidpi.Scale() != 2 {
		t.Fatalf("hidpi window scale %v", hidpi.Scale())
	}
	if lodpi.Scale() != 1 {
		t.Fatalf("lodpi window scale %v", lodpi.Scale())
	}
	want := style.DefaultMetrics().ControlH
	if got := hidpi.Look().Metrics().ControlH; got != want*2 {
		t.Fatalf("hidpi metrics %v, want %v", got, want*2)
	}
	if got := lodpi.Look().Metrics().ControlH; got != want {
		t.Fatalf("lodpi metrics %v, want %v", got, want)
	}
}

// An explicit Options.Scale (or a UITK_SCALE-style override) pins every
// window regardless of what the surface reports.
func TestExplicitScalePinsWindows(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	w := newScaledWindow(a, 1, 200, 120)
	if w.Scale() != 2 {
		t.Fatalf("explicit scale not honoured: %v", w.Scale())
	}
	if got := w.Look().Metrics().ControlH; got != style.DefaultMetrics().ControlH*2 {
		t.Fatalf("metrics %v", got)
	}
}

// Moving a window to another monitor arrives as a resize: the window must
// re-read its scale, rebuild its look and re-layout.
func TestResizeRechecksDisplayScale(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w := newScaledWindow(a, 1, 200, 120)
	w.SetContent(widgets.NewButton("OK", nil))
	a.PumpOnce()
	base := w.Look().Metrics().ControlH

	w.surf.(*scaledSurface).scale = 2
	w.dispatch(platform.Event{Kind: platform.EventResize, Width: 200, Height: 120})
	if w.Scale() != 2 {
		t.Fatalf("scale after monitor change %v", w.Scale())
	}
	if got := w.Look().Metrics().ControlH; got != base*2 {
		t.Fatalf("look not rebuilt at the new scale: %v want %v", got, base*2)
	}
	if w.laid {
		t.Fatal("a scale change must request a new layout")
	}
	a.PumpOnce()
	if !w.laid {
		t.Fatal("layout should have run")
	}
}

// A theme swap must keep each window's own display scale.
func TestSetLookKeepsPerWindowScale(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	hidpi := newScaledWindow(a, 2, 200, 120)
	lodpi := newScaledWindow(a, 1, 200, 120)
	a.SetLook(style.LightLook())
	if hidpi.Look().Name() != "light" || lodpi.Look().Name() != "light" {
		t.Fatalf("theme not applied: %q %q", hidpi.Look().Name(), lodpi.Look().Name())
	}
	want := style.DefaultMetrics().ControlH
	if got := hidpi.Look().Metrics().ControlH; got != want*2 {
		t.Fatalf("hidpi window lost its scale on theme swap: %v", got)
	}
	if got := lodpi.Look().Metrics().ControlH; got != want {
		t.Fatalf("lodpi window gained scale on theme swap: %v", got)
	}
}

// Repeated theme swaps must not compound the display scale.
func TestSetLookDoesNotCompoundScale(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	want := a.Look().Metrics().ControlH
	for i := 0; i < 4; i++ {
		a.SetLook(style.LightLook())
		a.SetLook(style.DarkLook())
	}
	if got := a.Look().Metrics().ControlH; got != want {
		t.Fatalf("scale compounded across theme swaps: %v want %v", got, want)
	}
	// An already-scaled look must not be scaled a second time either.
	a.SetLook(style.WithScale(style.DarkLook(), 2))
	if got := a.Look().Metrics().ControlH; got != want {
		t.Fatalf("pre-scaled look re-scaled: %v want %v", got, want)
	}
}
