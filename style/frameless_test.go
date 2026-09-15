package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A frameless field (a spin box's, an editable combo box's) types in the
// same ink as the look's plain text fields, in every pack: engines that
// paint their own field text paint these too.
func TestFramelessFieldTypesLikeTheFields(t *testing.T) {
	const text = "MMMMWWWW"
	for _, name := range AllBuiltinThemeNames() {
		lk := retroLook(t, name, 1)
		b := paintengine2d.XYWH(0, 0, 240, 32)
		framed := paintengine2d.NewImage(240, 32)
		lk.DrawTextField(paintengine2d.NewContext(framed), b, StateNone, text, "", 0, 0, 0, false, 0, nil)
		ground, want, ok := fieldInk(framed, 12, 8, 120, 24)
		if !ok {
			t.Errorf("%s: no text in the field", name)
			continue
		}
		bare := paintengine2d.NewImage(240, 32)
		ctx := paintengine2d.NewContext(bare)
		// The frameless field on the ground its parent painted.
		if ground[3] == 255 {
			ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(float32(ground[0])/255, float32(ground[1])/255, float32(ground[2])/255)))
		}
		lk.DrawTextField(ctx, b, StateFrameless, text, "", 0, 0, 0, false, 0, nil)
		if _, got, ok := fieldInk(bare, 12, 8, 120, 24); !ok || !inkClose(got, want) {
			t.Errorf("%s: frameless text in %v, the fields' in %v", name, got, want)
		}
	}
}

// fieldInk is the commonest premultiplied pixel in the region (the ground)
// and the commonest well apart from it (the text's ink).
func fieldInk(img *paintengine2d.Image, x0, y0, x1, y1 int) (ground, ink [4]uint8, ok bool) {
	counts := map[[4]uint8]int{}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, a := img.PremulAt(x, y)
			counts[[4]uint8{r, g, b, a}]++
		}
	}
	commonest := func(keep func([4]uint8) bool) (best [4]uint8, found bool) {
		n := 0
		for k, c := range counts {
			if !keep(k) {
				continue
			}
			// Ties go to the lower key, so the answer is stable.
			if c > n || (c == n && inkLess(k, best)) {
				best, n, found = k, c, true
			}
		}
		return best, found
	}
	ground, _ = commonest(func([4]uint8) bool { return true })
	ink, ok = commonest(func(k [4]uint8) bool { return inkDist(k, ground) > 90 })
	return ground, ink, ok
}

func inkDist(a, b [4]uint8) int {
	d := 0
	for i := range a {
		if a[i] > b[i] {
			d += int(a[i] - b[i])
		} else {
			d += int(b[i] - a[i])
		}
	}
	return d
}

func inkLess(a, b [4]uint8) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func inkClose(a, b [4]uint8) bool {
	for i := range a {
		if inkDist([4]uint8{a[i]}, [4]uint8{b[i]}) > 40 {
			return false
		}
	}
	return true
}
