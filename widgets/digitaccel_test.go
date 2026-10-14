package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// An accelerator written as a digit is a digit.
//
// accelKey knew the letters and three characters, so a menu item with
// Ctrl+1 on it never fired: the name resolved to KeyUnknown, which
// matches no key event. Ctrl+3 worked, which is the sort of thing that
// makes a person doubt their keyboard.
func TestAMenuAcceleratorCanBeADigit(t *testing.T) {
	ran := -1
	var items []*MenuItem
	for i := 1; i <= 9; i++ {
		i := i
		items = append(items, ItemAccel("Tab "+string(rune('0'+i)), "Ctrl+"+string(rune('0'+i)), func() { ran = i }))
	}
	m := NewMenuBar(NewMenu("&Go", items...))
	m.SetLook(style.DarkLook())
	m.SetHost(&host{})

	for i := 1; i <= 9; i++ {
		ran = -1
		key := platform.DigitKey(rune('0' + i))
		if key == platform.KeyUnknown {
			t.Fatalf("%d is no key", i)
		}
		if !m.HandleAccelerator(key, platform.ModCtrl) {
			t.Errorf("Ctrl+%d was not handled", i)
			continue
		}
		if ran != i {
			t.Errorf("Ctrl+%d ran item %d", i, ran)
		}
	}
}

// A tool button can carry a badge, and does not when it has none.
//
// Counted as *changed pixels*, not as inked ones: a tool bar paints a
// background over the whole strip, so every pixel has ink in it and a
// count of them says nothing at all. The first version of this test
// was written that way and passed whatever the badge did.
func TestAToolButtonCanCarryABadge(t *testing.T) {
	shot := func(badge style.IconBadge) *paintengine2d.Image {
		bar := NewToolBar(&ToolItem{Text: "New", Icon: style.IconSafe, Badge: badge})
		bar.SetLook(style.DarkLook())
		bar.SetHost(&host{})
		bar.Arrange(paintengine2d.XYWH(0, 0, 120, 40))
		img := paintengine2d.NewImage(120, 40)
		bar.Paint(paintengine2d.NewContext(img))
		img.Touch()
		return img
	}
	differs := func(a, b *paintengine2d.Image) int {
		n := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 120; x++ {
				r1, g1, b1, a1 := a.PremulAt(x, y)
				r2, g2, b2, a2 := b.PremulAt(x, y)
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					n++
				}
			}
		}
		return n
	}

	plain := shot(style.IconBadge{})
	if n := differs(plain, shot(style.IconBadge{})); n != 0 {
		t.Fatalf("two plain bars differ in %d pixels", n)
	}
	if n := differs(plain, shot(style.IconBadge{Mark: style.BadgePlus})); n == 0 {
		t.Error("a badged button looks exactly like a plain one")
	}
	if n := differs(plain, shot(style.Badged(3))); n == 0 {
		t.Error("a counted button looks exactly like a plain one")
	}
	if n := differs(plain, shot(style.Badged(0))); n != 0 {
		t.Errorf("a zero count changed %d pixels; it should draw nothing", n)
	}
}
