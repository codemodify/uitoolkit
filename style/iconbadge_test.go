package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func badgeInk(t *testing.T, badge IconBadge) (corner, total int) {
	t.Helper()
	const n = 48
	img := paintengine2d.NewImage(n, n)
	ctx := paintengine2d.NewContext(img)
	lk := DarkLook()
	box := paintengine2d.XYWH(8, 8, 32, 32)
	DrawToolIcon(ctx, box, IconSafe, lk.Palette().Text, "")
	DrawIconBadge(ctx, box, badge, lk)
	img.Touch()
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				total++
				if x > 28 && y > 28 {
					corner++
				}
			}
		}
	}
	return corner, total
}

// A badge is a mark on the corner of an icon, and nothing at all when
// there is nothing to say.
//
// A password manager drew a padlock with a plus, a vault mark with a
// plus and a count, all three by hand, because an icon could carry no
// mark of its own.
func TestABadgeMarksTheCorner(t *testing.T) {
	plain, plainTotal := badgeInk(t, IconBadge{})
	if plainTotal == 0 {
		t.Fatal("the icon drew nothing")
	}
	plus, plusTotal := badgeInk(t, IconBadge{Mark: BadgePlus})
	if plus <= plain {
		t.Errorf("a plus badge drew %d pixels in the corner and a plain icon %d", plus, plain)
	}
	if plusTotal <= plainTotal {
		t.Error("the badge added no ink")
	}

	t.Run("a dot says only that there is something", func(t *testing.T) {
		dot, _ := badgeInk(t, IconBadge{Mark: BadgeDot})
		if dot <= plain {
			t.Errorf("a dot badge drew %d pixels in the corner and a plain icon %d", dot, plain)
		}
	})

	t.Run("a count of none is no badge", func(t *testing.T) {
		if b := Badged(0); !b.Empty() {
			t.Error("Badged(0) is a badge")
		}
		got, _ := badgeInk(t, Badged(0))
		if got != plain {
			t.Errorf("a zero count drew %d corner pixels, a plain icon %d", got, plain)
		}
	})

	t.Run("a count is drawn where counts are", func(t *testing.T) {
		const n = 48
		img := paintengine2d.NewImage(n, n)
		ctx := paintengine2d.NewContext(img)
		lk := DarkLook()
		box := paintengine2d.XYWH(8, 8, 32, 32)
		DrawIconBadge(ctx, box, Badged(7), lk)
		img.Touch()
		top, bottom := 0, 0
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if _, _, _, a := img.PremulAt(x, y); a > 0 {
					if y < 24 {
						top++
					} else {
						bottom++
					}
				}
			}
		}
		if top <= bottom {
			t.Errorf("a count drew %d pixels up top and %d below; a count belongs at the top right", top, bottom)
		}
	})
}

// The cloud is an icon of its own in every set, drawn rather than
// missing. A password manager needed one for a vault that comes from a
// sync server and drew it itself.
func TestTheCloudIsDrawnEverywhere(t *testing.T) {
	lk := DarkLook()
	box := paintengine2d.XYWH(2, 2, 28, 28)
	ink := func(set IconSetName, icon ToolIcon) int {
		img := paintengine2d.NewImage(32, 32)
		ctx := paintengine2d.NewContext(img)
		DrawToolIcon(ctx, box, icon, lk.Palette().Text, set)
		img.Touch()
		n := 0
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				if _, _, _, a := img.PremulAt(x, y); a > 0 {
					n++
				}
			}
		}
		return n
	}
	for _, set := range []IconSetName{"", "sharp"} {
		cloud := ink(set, IconCloud)
		if cloud == 0 {
			t.Errorf("the %q set drew no cloud", set)
		}
		// Not the same ink as a shape nobody drew: that would mean
		// the cloud is falling through to the missing-icon mark.
		if cloud == ink(set, ToolIcon(9999)) {
			t.Errorf("the %q set drew the no-icon mark for the cloud", set)
		}
	}
	// The two drawn sets do not draw it the same way, or the square
	// era is borrowing the round one's art.
	if ink("", IconCloud) == ink("sharp", IconCloud) {
		t.Error("the classic and sharp clouds are the same drawing")
	}
}
