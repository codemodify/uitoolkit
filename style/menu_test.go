package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestMenuChromeGutterFitsIconSize(t *testing.T) {
	for _, sz := range []IconSize{IconSizeSmall, IconSizeMedium, IconSizeLarge} {
		lk := WithIconSize(DarkLook(), sz)
		ch := MenuChromeFor(lk)
		need := IconSizePixels(sz)
		if ch.CheckCol()+0.5 < need {
			t.Fatalf("%s gutter %v < icon %v", sz, ch.CheckCol(), need)
		}
	}
}

func TestMenuChromeForScalesWithLook(t *testing.T) {
	lo := MenuChromeFor(DarkLook())
	hi := MenuChromeFor(WithScale(DarkLook(), 2))
	if lo.CheckCol() < 19 || lo.ItemPad < 9 || lo.LabelGap < 7 {
		t.Fatalf("1x chrome %+v", lo)
	}
	if hi.CheckCol() < lo.CheckCol()*1.9 || hi.ItemPad < lo.ItemPad*1.9 {
		t.Fatalf("2x chrome %+v vs 1x %+v", hi, lo)
	}
	w1 := lo.FrameWidth(100, 0)
	w2 := hi.FrameWidth(200, 0)
	if w2+1 < w1*1.8 {
		t.Fatalf("scaled frame width 1x=%v 2x=%v", w1, w2)
	}
}

func TestMenuPaletteHasHoverChrome(t *testing.T) {
	for _, p := range []Palette{Dark(), Light()} {
		p := ResolveMenuChrome(p)
		if sameRGB(p.MenuHover, p.SurfaceAlt) {
			t.Fatalf("MenuHover matches SurfaceAlt %+v", p.MenuHover)
		}
		if sameRGB(p.MenuHover, p.MenuHoverBorder) {
			t.Fatalf("MenuHover matches border %+v", p.MenuHover)
		}
		if sameRGB(p.MenuGutter, p.SurfaceAlt) {
			t.Fatalf("MenuGutter matches SurfaceAlt %+v", p.MenuGutter)
		}
	}
}

func TestResolveMenuChromeTintsFromAccent(t *testing.T) {
	p := Palette{
		SurfaceAlt: paintengine2d.RGB(0.9, 0.9, 0.9),
		Background: paintengine2d.RGB(0.8, 0.8, 0.8),
		Accent:     paintengine2d.RGB(0.1, 0.4, 0.9),
		Border:     paintengine2d.RGB(0.5, 0.5, 0.5),
	}
	got := ResolveMenuChrome(p)
	if colorUnset(got.MenuHover) || colorUnset(got.MenuHoverBorder) || colorUnset(got.MenuGutter) {
		t.Fatalf("unresolved %+v", got)
	}
	if sameRGB(got.MenuHover, p.SurfaceAlt) {
		t.Fatalf("derived hover should tint toward Accent")
	}
}

func sameRGB(a, b paintengine2d.Color) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B
}

func TestMenuChromeLabelGapAfterGutter(t *testing.T) {
	ch := MenuChromeFor(DarkLook())
	if ch.LabelGap < 8 {
		t.Fatalf("LabelGap %v", ch.LabelGap)
	}
	gutter := ch.GutterW()
	label := ch.PadL + ch.CheckCol() + ch.LabelGap
	if gap := label - gutter; gap < 8 {
		t.Fatalf("gutter→label %v (gutter %v label %v)", gap, gutter, label)
	}
	row := float32(8)
	if ch.LabelMinX(row)-row < ch.CheckCol()+7 {
		t.Fatalf("LabelMinX %v CheckCol %v", ch.LabelMinX(row), ch.CheckCol())
	}
}

func TestMenuChromeFrameWidthIncludesItemPad(t *testing.T) {
	ch := MenuChromeFor(nil)
	label := float32(80)
	w := ch.FrameWidth(label, 0)
	// Label starts after PadL+CheckCol+LabelGap; reserved advance ends ItemPad
	// before item right, and item right is PadR before the frame.
	need := ch.PadL + ch.CheckCol() + ch.LabelGap + label + ch.ItemPad + ch.PadR
	if w+0.5 < need {
		t.Fatalf("frame %v < pad+label %v chrome=%+v", w, need, ch)
	}
}
