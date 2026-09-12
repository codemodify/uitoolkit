package style

import "testing"

func TestMenuChromeForScalesWithLook(t *testing.T) {
	lo := MenuChromeFor(DarkLook())
	hi := MenuChromeFor(WithScale(DarkLook(), 2))
	if lo.CheckCol() < 19 || lo.ItemPad < 9 {
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

func TestMenuChromeFrameWidthIncludesItemPad(t *testing.T) {
	ch := MenuChromeFor(nil)
	label := float32(80)
	w := ch.FrameWidth(label, 0)
	// Label starts at PadL+CheckCol; reserved advance ends ItemPad before item
	// right, and item right is PadR before the frame. Border is extra slop.
	need := ch.PadL + ch.CheckCol() + label + ch.ItemPad + ch.PadR
	if w+0.5 < need {
		t.Fatalf("frame %v < pad+label %v chrome=%+v", w, need, ch)
	}
}
