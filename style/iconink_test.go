package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// paintedIcon draws one icon into an image, for tests that ask whether
// anything was drawn at all.
func paintedIcon(t *testing.T, icon ToolIcon, set IconSetName, side int) *paintengine2d.Image {
	t.Helper()
	img := paintengine2d.NewImage(side, side)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(side), float32(side)),
		paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	DrawToolIcon(ctx, paintengine2d.XYWH(0, 0, float32(side), float32(side)),
		icon, paintengine2d.RGB(0, 0, 0), set)
	return img
}

// anyInk reports whether anything darker than the white ground was drawn.
func anyInk(img *paintengine2d.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 0xd000 || g < 0xd000 || bl < 0xd000 {
				return true
			}
		}
	}
	return false
}

// No typed icon may draw nothing. A button with an invisible mark on it
// is worse than one with a placeholder, and the drawn sets had no
// default arm at all — anything they had no vector for simply did not
// appear.
func TestEveryTypedIconDrawsSomething(t *testing.T) {
	for _, set := range []IconSetName{IconSetClassic, IconSetSharp} {
		for _, icon := range AllToolIcons() {
			if !anyInk(paintedIcon(t, icon, set, 32)) {
				t.Errorf("%v (%s) drew nothing in set %q", icon, ToolIconName(icon), set)
			}
		}
	}
}

// And a stem-only icon draws the missing-icon mark rather than nothing.
func TestStemOnlyIconDrawsThePlaceholder(t *testing.T) {
	icon, ok := IconByStem("calendar")
	if !ok {
		t.Skip("calendar is not shipped in this build")
	}
	if _, typed := ToolIconByName("calendar"); typed {
		t.Skip("calendar has a typed id now, so this proves nothing")
	}
	if !anyInk(paintedIcon(t, icon, IconSetClassic, 32)) {
		t.Error("a stem-only icon drew nothing in a drawn set")
	}
}
