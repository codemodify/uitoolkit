package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The fallback paperclip must have a hole: with both contours wound the
// same way it filled solid and read as a tofu box in Mail.
func TestFallbackPaperclipHasHole(t *testing.T) {
	const size = 64
	path, _ := fallbackPaperclip(size)
	img := paintengine2d.NewImage(size, size)
	ctx := paintengine2d.NewContext(img)
	ctx.Translate(0, size)
	ctx.DrawPath(path, paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	b := path.Bounds()
	// A point inside the loop, between the outer wire and the inner bar.
	x := int(b.Min.X + b.Dx()*0.25)
	y := int(size + b.Min.Y + b.Dy()*0.5)
	if _, _, _, a := img.PremulAt(x, y); a > 40 {
		t.Fatalf("paperclip interior at (%d,%d) is filled (alpha %d): no hole", x, y, a)
	}
	// And the wire itself is there.
	if _, _, _, a := img.PremulAt(int(b.Min.X+1), y); a < 100 {
		t.Fatalf("paperclip wire missing at the left edge (alpha %d)", a)
	}
}
