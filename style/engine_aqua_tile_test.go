package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The brushed-metal tile paints the same pixels as drawing every streak,
// on whole pixels at 1x and 2x, and under a translation (a widget painting
// the window background at its own origin).
func TestBrushedMetalTileMatchesStreaks(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := WithScale(mustLook(t, "brushed-metal"), scale).(*Classic)
		c := aquaColors(lk)
		const w, h = 300, 200
		draw := func(tile bool) *paintengine2d.Image {
			img := paintengine2d.NewImage(w, h)
			ctx := paintengine2d.NewContext(img)
			ctx.Translate(0, 7) // the anchor moves the rows
			b := paintengine2d.XYWH(0, -7, w, h)
			if tile {
				c.texture(lk, ctx, b)
			} else {
				// The path route, forced by a fractional left edge that
				// covers the same pixels.
				ctx.Save()
				ctx.ClipRect(paintengine2d.XYWH(0, -7, w, h))
				c.texture(lk, ctx, paintengine2d.XYWH(0.0001, -7, w, h))
				ctx.Restore()
			}
			return img
		}
		a, b := draw(true), draw(false)
		diff := 0
		for y := 0; y < h; y++ {
			for x := 2; x < w-2; x++ {
				ar, ag, ab, _ := a.PremulAt(x, y)
				br, bg, bb, _ := b.PremulAt(x, y)
				if d := absI(int(ar)-int(br)) + absI(int(ag)-int(bg)) + absI(int(ab)-int(bb)); d > 6 {
					diff++
				}
			}
		}
		if diff > w*h/200 {
			t.Errorf("%gx: the tile differs from the streaks at %d pixels", scale, diff)
		}
	}
}

func absI(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
