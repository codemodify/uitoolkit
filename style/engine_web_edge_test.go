//go:build theme_engine_all || theme_engine_web

package style

import "testing"

// A pack states one neutral border grey and uses it on every edge, which is
// right while its surfaces are neutral. An application that tints its chrome
// moves the surfaces and not the border, and the selected browser tab came
// out outlined in flat grey against a tinted strip.
//
// edgeOn keeps the border's lightness — that is what makes it read as an
// edge — and gives it the surface's distance from grey. The property that
// matters is that it changes nothing for a pack that tints nothing.
func TestATabsEdgeFollowsTheSurfaceItBorders(t *testing.T) {
	border := Hex("#e2e2e2")

	// A neutral surface has no colour to give, at any lightness.
	for _, grey := range []string{"#ffffff", "#fafafa", "#808080", "#0a0a0a", "#000000"} {
		if got := edgeOn(border, Hex(grey)); got != border {
			t.Errorf("on neutral %s the edge is %v, want the pack's own %v", grey, got, border)
		}
	}

	// A tinted surface lends its hue and keeps the border's lightness: the
	// edge reads as the same edge, in the chrome's colour.
	surface := Hex("#ebf2fe") // the blue-white a browser sample tints its tool bar
	got := edgeOn(border, surface)
	if got == border {
		t.Fatal("a tinted surface left the edge neutral")
	}
	if got.B <= got.R {
		t.Errorf("the edge %v did not take the surface's blue", got)
	}
	if d := luma(got) - luma(border); d < -0.02 || d > 0.02 {
		t.Errorf("the edge shifted in lightness by %.3f; it should only have shifted in colour", d)
	}

	// It stays a colour: nothing runs past the ends of the range, and the
	// border's own alpha is kept.
	for _, s := range []string{"#ffffff", "#000000", "#ff0000", "#00ff00"} {
		e := edgeOn(border, Hex(s))
		for _, v := range []float32{e.R, e.G, e.B} {
			if v < 0 || v > 1 {
				t.Errorf("on %s the edge is out of range: %v", s, e)
			}
		}
		if e.A != border.A {
			t.Errorf("on %s the edge changed alpha: %v", s, e)
		}
	}
}