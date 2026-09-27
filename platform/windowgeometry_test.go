package platform

import "testing"

// The five interfaces this seam replaced were the clearest case of silent
// failure in the package: RaiseSurface, HideSurface and MoveSurface were
// `if ok { do }` with no else and no return value, so a window that could
// not be moved was not moved and nothing found out. Every request answers
// now, and the capability decides.
func TestGeometryCapsGateEveryRequest(t *testing.T) {
	cases := []struct {
		name string
		cap  GeometryCaps
		do   func(WindowGeometry) bool
	}{
		{"move", GeometryMove, func(g WindowGeometry) bool { return g.Move(10, 20) }},
		{"position", GeometryPosition, func(g WindowGeometry) bool {
			_, _, ok := g.Position()
			return ok
		}},
		{"screen-place", GeometryScreenPlace, func(g WindowGeometry) bool { return g.PlaceAtScreen(3, 4) }},
		{"visibility", GeometryVisibility, func(g WindowGeometry) bool { return g.Hide() }},
		{"size-limits", GeometrySizeLimits, func(g WindowGeometry) bool { return g.SetSizing(SizingFixed) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Present: the request is made.
			o := NewOffscreen(WindowOptions{Width: 200, Height: 140})
			if g := GeometryOf(o); !g.GeometryCaps().Has(c.cap) {
				t.Fatalf("%v missing from %v", c.cap, g.GeometryCaps())
			} else if !c.do(g) {
				t.Fatalf("%v is there and the request was refused", c.cap)
			}
			// Absent: it is refused, not silently dropped.
			o = NewOffscreen(WindowOptions{Width: 200, Height: 140})
			o.SimulateGeometry(c.cap)
			g := GeometryOf(o)
			if g.GeometryCaps().Has(c.cap) {
				t.Fatalf("%v is still in %v", c.cap, g.GeometryCaps())
			}
			if c.do(g) {
				t.Fatalf("%v is gone and the request was granted anyway", c.cap)
			}
		})
	}
}

// A surface with no geometry seam answers rather than being absent: the
// caller asks one question and gets one answer.
func TestGeometryOfSurfaceWithoutOne(t *testing.T) {
	g := GeometryOf(nil)
	if g == nil {
		t.Fatal("GeometryOf must never return nil")
	}
	if g.GeometryCaps() != 0 {
		t.Errorf("caps %v", g.GeometryCaps())
	}
	if g.Move(1, 2) || g.PlaceAtScreen(1, 2) || g.Show() || g.Hide() ||
		g.Raise() || g.Visible() || g.SetSizing(SizingFixed) {
		t.Error("a surface with no geometry granted a request")
	}
	if _, _, ok := g.Position(); ok {
		t.Error("a surface with no geometry knows where it is")
	}
	if g.Sizing() != SizingResizable {
		t.Errorf("sizing %v", g.Sizing())
	}
}

// Wayland's two absences are the reason the seam has capabilities at all,
// and the offscreen backend can be made to stand in for it: a window that
// cannot be moved and is never told where it is.
func TestGeometryWithoutMoveOrPositionIsTheWaylandCase(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 200, Height: 140})
	o.SimulateGeometry(GeometryMove | GeometryPosition)

	if SurfaceMoves(o) {
		t.Error("a window that cannot be moved said it could")
	}
	if MoveSurface(o, 5, 5) {
		t.Error("the move was granted")
	}
	if _, _, ok := SurfacePosition(o); ok {
		t.Error("the window said where it was")
	}
	// The rest of the seam still works — the absences are specific.
	if !SurfaceVisible(o) {
		t.Error("the window is not visible")
	}
	if !HideSurface(o) {
		t.Error("hiding was refused")
	}
	if SurfaceVisible(o) {
		t.Error("the window is still visible after Hide")
	}
	if !RaiseSurface(o) {
		t.Error("raising was refused")
	}
}

// Capability names print, so a diagnostic can say what a desktop will do.
func TestGeometryCapsString(t *testing.T) {
	if got := GeometryCaps(0).String(); got != "none" {
		t.Errorf("none prints as %q", got)
	}
	if got := (GeometryMove | GeometrySizeLimits).String(); got != "move size-limits" {
		t.Errorf("prints as %q", got)
	}
}
