package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every look's popup shadow stays inside the reach it declares: the window
// only repaints that band when a menu or tooltip goes away, so paint beyond
// it would be left behind on screen.
func TestPopupShadowStaysInsideReach(t *testing.T) {
	kinds := []PopupKind{PopupMenu, PopupTooltip, PopupDialog}
	for _, p := range ListBuiltinThemes() {
		for _, scale := range []float32{1, 2} {
			lk := p.Look().setScale(scale)
			for _, kind := range kinds {
				img := paintengine2d.NewImage(400, 300)
				ctx := paintengine2d.NewContext(img)
				b := paintengine2d.XYWH(100.25, 80.5, 160, 120)
				lk.DrawPopupShadow(ctx, b, kind)
				reach := lk.PopupShadow(kind).Grow(b)
				for y := 0; y < 300; y++ {
					for x := 0; x < 400; x++ {
						if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
							continue
						}
						px := paintengine2d.XYWH(float32(x), float32(y), 1, 1)
						if !reach.Inset(-1).Contains(px.Min) || !reach.Inset(-1).Contains(px.Max) {
							t.Fatalf("%s @%vx kind %d: shadow pixel (%d,%d) outside reach %v", p.Name, scale, kind, x, y, reach)
						}
					}
				}
			}
		}
	}
}
