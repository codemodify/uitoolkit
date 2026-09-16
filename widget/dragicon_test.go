package widget

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// A drag's picture is drawn once at the display's scale, not twice.
//
// A window's look is already baked at that window's scale — its font and
// every Dip come out in device pixels — so multiplying by the scale again
// drew the chip scale times too large, which at 1.75 is three times the
// size it should be. What the look does not already carry is all that is
// left to apply here.
func TestDragLabelDrawsAtTheScaleOnce(t *testing.T) {
	base := style.DarkLook()
	at1, _ := DragLabel(base, "3 files", 1)
	if at1 == nil {
		t.Fatal("no picture at 1x")
	}
	for _, scale := range []float32{1.25, 1.5, 1.75, 2} {
		// A window's look: baked at the scale, asked for the same scale.
		lk := style.WithScale(base, scale)
		got, hot := DragLabel(lk, "3 files", scale)
		if got == nil {
			t.Fatalf("scale %g: no picture", scale)
		}
		want := float32(at1.Width) * scale
		if d := float32(got.Width) - want; d > 2 || d < -2 {
			t.Errorf("scale %g: the picture is %d px wide, want about %.0f", scale, got.Width, want)
		}
		if hot.X <= 0 || hot.X >= float32(got.Width) {
			t.Errorf("scale %g: the hotspot %v is not inside the picture", scale, hot)
		}
		// An unscaled look asked for the same scale draws the same size:
		// the scale is applied wherever it is not already baked in.
		plain, _ := DragLabel(base, "3 files", scale)
		if plain == nil || abs(plain.Width-got.Width) > 2 {
			t.Errorf("scale %g: an unscaled look gave %v, a baked one %d", scale, plain, got.Width)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
