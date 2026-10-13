package style

import "testing"

// **A mark's strip is the same width in logical pixels at every scale.**
//
// The icon was multiplied by the display scale and the pad and the gap
// round it were not — they were device pixels at any scale — so a
// button with a mark was 32 logical pixels wider than a plain one at 1x
// and 28 at 2x. secretvault measured a consent question, the same size
// at every scale, coming out 17 pixels narrower at 2x with four marked
// buttons in a row.
func TestAMarkedButtonsStripIsTheSameAtEveryScale(t *testing.T) {
	logical := func(scale float32) float32 {
		lk := WithScale(DarkLook(), scale)
		pad, side, gap := ToolButtonChromeFor(lk, 100*scale)
		// What a mark costs a button: the strip beside the words.
		return (pad + side + gap) / scale
	}
	one := logical(1)
	for _, scale := range []float32{1.25, 1.5, 1.75, 2} {
		got := logical(scale)
		if d := got - one; d > 0.75 || d < -0.75 {
			t.Errorf("a mark's strip is %.2f logical pixels at %gx and %.2f at 1x: %.2f out", got, scale, one, d)
		}
	}
}
