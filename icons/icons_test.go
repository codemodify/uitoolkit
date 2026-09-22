package icons

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every stem makes an icon at every size: square, clear at its corners,
// the tile's colour at its edge and white glyph ink inside it.
func TestAppIcon(t *testing.T) {
	bg := paintengine2d.RGB(0.1, 0.4, 0.8)
	for _, stem := range Stems() {
		imgs := AppIcon(stem, bg)
		if len(imgs) != len(AppIconSizes) {
			t.Fatalf("%s: %d images, want %d", stem, len(imgs), len(AppIconSizes))
		}
		for i, im := range imgs {
			side := AppIconSizes[i]
			if im.Width != side || im.Height != side {
				t.Fatalf("%s: image %d is %dx%d, want %d square", stem, i, im.Width, im.Height, side)
			}
			if _, _, _, a := im.PremulAt(0, 0); a != 0 {
				t.Errorf("%s @%d: the corner is not clear (alpha %d)", stem, side, a)
			}
			edge := im.NRGBAAt(side/2, int(math.Max(1, math.Round(float64(side)/32))))
			if edge.A < 250 || edge.B < edge.R+60 {
				t.Errorf("%s @%d: the tile's top %v is not its blue", stem, side, edge)
			}
			white := 0
			for y := side / 4; y < side*3/4; y++ {
				for x := side / 4; x < side*3/4; x++ {
					if p := im.NRGBAAt(x, y); p.R > 220 && p.G > 220 && p.B > 220 {
						white++
					}
				}
			}
			if white < side/4 {
				t.Errorf("%s @%d: %d white glyph pixels", stem, side, white)
			}
		}
	}
	if AppIcon("no-such-stem", bg) != nil {
		t.Fatal("an icon for a stem that is not embedded")
	}
}
