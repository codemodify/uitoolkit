//go:build linux && cgo

package platform

import (
	"os"
	"testing"
)

// A resizable shaped window's silhouette has its resize bands appended after
// it, so the list goes back to the top of the window and overlaps itself.
// X11's SHAPE extension was told that list was YXBanded regardless, and
// answers a false claim with BadMatch and no shape at all: a rounded window
// came out square, with the default full-window input region, and nothing in
// the application could tell.
//
// Under a real server, because the rejection is the server's. tools/e2e/… is
// where this runs; everywhere else it skips.
func TestAShapeWithResizeBandsIsAccepted(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := x11Backend{}
	raw, err := b.NewSurface(WindowOptions{Title: "uitoolkit-shape", Width: 240, Height: 160})
	if err != nil {
		t.Skip("no X11 surface: ", err)
	}
	s, ok := raw.(*x11Surface)
	if !ok {
		t.Fatalf("not an X11 surface: %T", raw)
	}
	defer s.Close()

	// A rounded silhouette: scanlines down the window, which *is* YXBanded —
	// and then the resize bands, which are not.
	shape := []FrameRect{
		{X: 2, Y: 0, W: 236, H: 1},
		{X: 1, Y: 1, W: 238, H: 1},
		{X: 0, Y: 2, W: 240, H: 156},
		{X: 1, Y: 158, W: 238, H: 1},
		{X: 2, Y: 159, W: 236, H: 1},
		// The bands, back at the top and over everything above.
		{X: 0, Y: 0, W: 240, H: 4},
		{X: 0, Y: 156, W: 240, H: 4},
		{X: 0, Y: 0, W: 4, H: 160},
		{X: 236, Y: 0, W: 4, H: 160},
	}
	if yxBanded(shape) {
		t.Fatal("this shape is YXBanded, so it does not test what it is here to test")
	}

	before := x11ErrorCount()
	s.SetFrame(Frame{Shape: shape})
	// The error comes back from the server, so it has to be waited for: a
	// round trip is what makes a reply or an error arrive.
	s.conn.syncRoundTrip()
	if got := x11ErrorCount(); got != before {
		t.Errorf("the server rejected the shape: %d protocol error(s)", got-before)
	}
	if !s.shaped {
		t.Error("the surface does not believe it is shaped")
	}
}

// And a silhouette that really is banded is still declared so, because that is
// what lets a shape of hundreds of rectangles go in without being sorted.
func TestABandedShapeIsStillAccepted(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	b := x11Backend{}
	raw, err := b.NewSurface(WindowOptions{Title: "uitoolkit-shape-banded", Width: 240, Height: 160})
	if err != nil {
		t.Skip("no X11 surface: ", err)
	}
	s := raw.(*x11Surface)
	defer s.Close()

	var shape []FrameRect
	for y := 0; y < 160; y++ {
		shape = append(shape, FrameRect{X: y % 8, Y: y, W: 240 - 2*(y%8), H: 1})
	}
	if !yxBanded(shape) {
		t.Fatal("these scanlines are not YXBanded, so this tests nothing")
	}
	before := x11ErrorCount()
	s.SetFrame(Frame{Shape: shape})
	s.conn.syncRoundTrip()
	if got := x11ErrorCount(); got != before {
		t.Errorf("the server rejected a banded shape: %d protocol error(s)", got-before)
	}
}
