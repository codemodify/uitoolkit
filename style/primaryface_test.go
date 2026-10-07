package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every pack shows which button Enter presses.
//
// Button.Primary is "this is the default button", and an engine that
// paints its own buttons is free to ignore it — Plastik, the toolkit's
// own default pack, painted every button the same, so a consent dialog
// that makes Deny the default showed nothing about which that was.
func TestEveryPackMarksTheDefaultButton(t *testing.T) {
	for _, pack := range ListBuiltinThemes() {
		look, ok := LoadTheme(pack.Name)
		if !ok {
			continue
		}
		lk := Appearance{Name: look.Name, Theme: look.Palette}.Look()
		const w, h = 72, 26
		draw := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(w, h)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(lk.Palette().Background))
			lk.DrawButton(ctx, paintengine2d.XYWH(2, 2, w-4, h-4), st, ButtonDraw{Label: "OK"})
			img.Touch()
			return img
		}
		// Visibly different, not merely different: a bevel shifted by a
		// fraction and rounded away marks nothing.
		if diff := visiblyDifferent(draw(StateNone), draw(StatePrimary)); diff < primaryFaceMinPixels {
			t.Errorf("%s: the default button differs by %d visible pixels", pack.Name, diff)
		}
	}
}

// A pack that paints its own default face gets no ring on top of it: two
// marks is worse than one, and the ring is a fallback rather than a
// house style.
func TestPacksWithTheirOwnDefaultFaceAreNotRinged(t *testing.T) {
	var own, fallback int
	for _, pack := range ListBuiltinThemes() {
		lk := Appearance{Name: pack.Name, Theme: pack.Palette}.Look()
		if PrimaryFaceOf(lk) {
			own++
		} else {
			fallback++
		}
	}
	if own == 0 {
		t.Fatal("no pack paints its own default button, which cannot be right")
	}
	t.Logf("%d packs paint their own default face, %d take the ring", own, fallback)
	// The measurement is cached per pack and must be stable.
	for _, pack := range ListBuiltinThemes() {
		lk := Appearance{Name: pack.Name, Theme: pack.Palette}.Look()
		if PrimaryFaceOf(lk) != PrimaryFaceOf(lk) {
			t.Fatalf("%s: PrimaryFaceOf changes its mind", pack.Name)
		}
	}
}

// A disabled default button is not ringed: it is not the button Enter
// presses while it cannot be pressed at all.
func TestDisabledDefaultIsNotMarked(t *testing.T) {
	lk := Appearance{Name: DefaultTheme()}.Normalize().Look()
	blank := paintengine2d.NewImage(72, 26)
	blank.Touch()
	img := paintengine2d.NewImage(72, 26)
	ctx := paintengine2d.NewContext(img)
	DrawDefaultMark(lk, ctx, paintengine2d.XYWH(2, 2, 68, 22), StatePrimary|StateDisabled)
	img.Touch()
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if img.At(x, y) != blank.At(x, y) {
				t.Fatal("a disabled default button was ringed")
			}
		}
	}
}
