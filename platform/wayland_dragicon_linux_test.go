//go:build linux && cgo

package platform

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The drag icon's geometry at every scale a display can have.
//
// Wayland sizes a surface in logical pixels, so a picture drawn at 1.75
// covers 1/1.75 of its own pixels' worth of them. Submitting it at the
// next whole scale — the only number wl_surface.set_buffer_scale can
// carry — put it on screen at the wrong size; the viewport's destination
// says the real one, and the hotspot has to be worth the same scale less
// or the picture sits off the pointer by the difference.
func TestDragIconGeometryAtFractionalScales(t *testing.T) {
	for _, tc := range []struct {
		scale  float32
		whole  bool
		lw, lh int // a 140x48 picture, in logical pixels
		hot    int // a hotspot 21 pixels in
	}{
		{1, true, 140, 48, 21},
		{1.25, false, 112, 38, 17},
		{1.5, false, 93, 32, 14},
		{1.75, false, 80, 27, 12},
		{2, true, 70, 24, 11},
	} {
		if got := wholeScale(tc.scale); got != tc.whole {
			t.Errorf("scale %g: wholeScale = %v, want %v", tc.scale, got, tc.whole)
		}
		lw, lh := iconLogical(140, 48, tc.scale)
		if lw != tc.lw || lh != tc.lh {
			t.Errorf("scale %g: 140x48 is %dx%d logical, want %dx%d", tc.scale, lw, lh, tc.lw, tc.lh)
		}
		if got := logicalLen(21, tc.scale); got != tc.hot {
			t.Errorf("scale %g: a hotspot at 21 px is %d logical, want %d", tc.scale, got, tc.hot)
		}
		// The logical size is what the picture really covers, to within
		// the rounding a whole pixel forces.
		if d := float64(lw)*float64(tc.scale) - 140; math.Abs(d) > float64(tc.scale) {
			t.Errorf("scale %g: %d logical px come to %.1f device px, want about 140", tc.scale, lw, float64(lw)*float64(tc.scale))
		}
	}
	// A picture smaller than one logical pixel still covers one.
	if lw, lh := iconLogical(1, 1, 4); lw != 1 || lh != 1 {
		t.Errorf("a 1x1 picture at 4x is %dx%d logical, want 1x1", lw, lh)
	}
	// A nonsense scale is one.
	if got := logicalLen(10, 0); got != 10 {
		t.Errorf("with no scale at all, 10 px is %d logical, want 10", got)
	}
}

// The icon is drawn at the scale the window that started the drag says,
// and at the surface's own when the drag says nothing.
func TestDragIconScaleComesFromTheDrag(t *testing.T) {
	img := paintengine2d.NewImage(4, 4)
	if got := dragIconScale(nil, DragPayload{Icon: img, Scale: 1.75}); got != 1.75 {
		t.Errorf("the drag said 1.75, got %g", got)
	}
	s := &wlSurface{frac: 1.5}
	if got := dragIconScale(s, DragPayload{Icon: img}); got != 1.5 {
		t.Errorf("the surface is at 1.5, got %g", got)
	}
	if got := dragIconScale(nil, DragPayload{Icon: img}); got != 1 {
		t.Errorf("with nothing to go on the scale is 1, got %g", got)
	}
}
