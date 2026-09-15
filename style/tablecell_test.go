package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestDrawTableCellPaintsNarrowStar(t *testing.T) {
	look := DarkLook()
	img := paintengine2d.NewImage(28, 28)
	ctx := paintengine2d.NewContext(img)
	look.DrawTableCell(ctx, paintengine2d.XYWH(0, 0, 28, 28), StateNone, "★", AlignStart, look.Font())
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < 26; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a > 20 {
				n++
			}
		}
	}
	if n < 8 {
		t.Fatalf("28px star cell has no glyph ink (%d); divider-only would be ~28 at x=27", n)
	}
}
