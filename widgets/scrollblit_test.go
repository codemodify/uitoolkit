package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestExposedStrip(t *testing.T) {
	view := paintengine2d.XYWH(10, 20, 100, 80)
	bottom := exposedStrip(view, 0, -16)
	if bottom != paintengine2d.XYWH(10, 84, 100, 16) {
		t.Fatalf("down scroll strip %+v", bottom)
	}
	top := exposedStrip(view, 0, 16)
	if top != paintengine2d.XYWH(10, 20, 100, 16) {
		t.Fatalf("up scroll strip %+v", top)
	}
	right := exposedStrip(view, -12, 0)
	if right != paintengine2d.XYWH(98, 20, 12, 80) {
		t.Fatalf("left scroll strip %+v", right)
	}
	left := exposedStrip(view, 12, 0)
	if left != paintengine2d.XYWH(10, 20, 12, 80) {
		t.Fatalf("right scroll strip %+v", left)
	}
}

func TestPixelDeltaOK(t *testing.T) {
	if !pixelDeltaOK(40) || !pixelDeltaOK(-12) || !pixelDeltaOK(0) {
		t.Fatal("integers should blit")
	}
	if pixelDeltaOK(40.3) {
		t.Fatal("fractional delta must fall back")
	}
}

func TestPixelScrollDisabled(t *testing.T) {
	if pixelScrollEnabled {
		t.Fatal("Context.Scroll blit must stay off until Wayland/HiDPI is proven")
	}
	if tryPixelScroll(nil, paintengine2d.XYWH(0, 0, 80, 80), 0, 16) {
		t.Fatal("tryPixelScroll must refuse")
	}
}
