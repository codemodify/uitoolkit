package platform

import "testing"

// A window that asks for an absolute position gets it on a desktop that
// places windows — the offscreen one does, like X11.
func TestPlaceSurfaceAtScreen(t *testing.T) {
	s, err := OffscreenBackend{}.NewSurface(WindowOptions{
		Width: 120, Height: 80, Scale: 1,
		X: 40, Y: 50, Place: PlaceAtScreen,
	})
	if err != nil {
		t.Fatalf("NewSurface: %v", err)
	}
	defer s.Close()
	x, y, ok := SurfacePosition(s)
	if !ok || x != 40 || y != 50 {
		t.Fatalf("opened at %d,%d ok=%v, want 40,50", x, y, ok)
	}
	if !PlaceSurfaceAtScreen(s, 300, 210) {
		t.Fatal("PlaceSurfaceAtScreen said it could not place an offscreen window")
	}
	if x, y, _ = SurfacePosition(s); x != 300 || y != 210 {
		t.Fatalf("moved to %d,%d, want 300,210", x, y)
	}
}

// Place is carried into the surface's options untouched, so a backend that
// cares can see it and one that does not is unaffected.
func TestPlacementOptionPlumbed(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 10, Height: 10, Place: PlaceAtScreen})
	if o.opts.Place != PlaceAtScreen {
		t.Fatalf("Place = %v, want PlaceAtScreen", o.opts.Place)
	}
	d := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	if d.opts.Place != PlaceDesktop {
		t.Fatalf("default Place = %v, want PlaceDesktop", d.opts.Place)
	}
}

// PlaceSurfaceAtScreen on nothing is a no-op, not a panic.
func TestPlaceSurfaceAtScreenNil(t *testing.T) {
	if PlaceSurfaceAtScreen(nil, 1, 2) {
		t.Fatal("placed a nil surface")
	}
}

// The simulation hook is what lets the headless tests walk both roads:
// a desktop that places windows and one that does not.
func TestSimulateScreenPlacement(t *testing.T) {
	restore := SimulateScreenPlacement(false)
	if ScreenPlacementAvailable() {
		t.Fatal("simulated absence of layer shell still reports placement")
	}
	restore()
	restore = SimulateScreenPlacement(true)
	if !ScreenPlacementAvailable() {
		t.Fatal("simulated layer shell does not report placement")
	}
	restore()
	// Back to the backend's own answer: offscreen and X11 place windows,
	// and this test suite runs on neither a compositor nor a display.
	if !ScreenPlacementAvailable() {
		t.Fatal("headless desktop should place windows")
	}
}
