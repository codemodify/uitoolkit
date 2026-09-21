package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An app that lays out a panel of its own paints the skin's loose sprites by
// name, by the same rules a part is painted by: the ink lands in the box it
// was given, at every scale, and nowhere else.
func TestAnAppPaintsASkinSpriteByName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme("minim-classic")
	if !ok {
		t.Fatal("minim-classic does not load")
	}
	w, h, ok := SkinSpriteSize(p.Look(), "led.8")
	if !ok || w != 11 || h != 15 {
		t.Fatalf("led.8 is %gx%g (%v), want the 9x13 digit and its margin", w, h, ok)
	}
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		InvalidateSkinCache()
		lk := WithScale(p.Look(), scale)
		img := paintengine2d.NewImage(80, 80)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		b := paintengine2d.XYWH(20, 20, w*scale, h*scale)
		if !DrawSkinSprite(lk, ctx, b, "led.8", paintengine2d.Color{}) {
			t.Fatalf("%gx: led.8 did not paint", scale)
		}
		ink, outside := 0, 0
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				if _, _, _, a := img.PremulAt(x, y); a > 0 {
					if b.Contains(paintengine2d.Pt(float32(x)+0.5, float32(y)+0.5)) {
						ink++
					} else {
						outside++
					}
				}
			}
		}
		if ink == 0 || outside > 0 {
			t.Errorf("%gx: %d pixels of ink in the box and %d outside it", scale, ink, outside)
		}
	}

	// A name the skin does not have, and a look that is not a skin, both
	// say so and paint nothing, so the app can paint its own.
	img := paintengine2d.NewImage(40, 40)
	ctx := paintengine2d.NewContext(img)
	if DrawSkinSprite(p.Look(), ctx, paintengine2d.XYWH(0, 0, 40, 40), "no.such.sprite", paintengine2d.Color{}) {
		t.Error("an unknown sprite reported that it painted")
	}
	plain, _ := LoadTheme("breeze-night")
	if DrawSkinSprite(plain.Look(), ctx, paintengine2d.XYWH(0, 0, 40, 40), "led.8", paintengine2d.Color{}) {
		t.Error("a look that is not a skin painted a sprite")
	}
	if _, _, ok := SkinSpriteSize(plain.Look(), "led.8"); ok {
		t.Error("a look that is not a skin has a sprite size")
	}
}

// A skin that states a caption shorter than a frame's usual 24 pixels gets
// the caption it stated, with square buttons stood inside it, rather than a
// band grown to fit buttons as tall as the caption would have been.
func TestASkinsShortCaptionIsTheOneItStates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct {
		pack    string
		caption float32
	}{{"minim-classic", 14}, {"minim-silver", 15}, {"minim", 18}} {
		p, ok := LoadTheme(c.pack)
		if !ok {
			t.Fatalf("%s does not load", c.pack)
		}
		for _, scale := range []float32{1, 1.75, 2} {
			lk := WithScale(p.Look(), scale)
			d := DecorationOf(lk, DecorationState{Active: true})
			want := float32(int(c.caption*scale + 0.5))
			if d.Caption != want {
				t.Errorf("%s@%gx: caption %g, want %g", c.pack, scale, d.Caption, want)
			}
			if d.Button.Y <= 0 || d.ButtonPad.Top+d.Button.Y > d.Caption {
				t.Errorf("%s@%gx: a %gx%g button %g down does not fit a %g caption",
					c.pack, scale, d.Button.X, d.Button.Y, d.ButtonPad.Top, d.Caption)
			}
		}
	}
}

// The caption.title part is a plate under the title, as wide as the title:
// the band's groove shows either side of the words and stops short of them.
func TestTheCaptionTitleStandsOnAPlate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, _ := LoadTheme("minim-classic")
	lk := p.Look()
	sk := skinFor(lk)
	if sk == nil || !sk.has("caption.title") {
		t.Fatal("minim-classic binds no caption.title")
	}
	const W, H = 275, 14
	img := paintengine2d.NewImage(W, H)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	band := paintengine2d.XYWH(0, 0, W, H)
	st := DecorationState{Active: true}
	skinEngine{}.DrawDecoration(lk, ctx, DecorationFrame{Window: paintengine2d.XYWH(0, 0, W, 40), Caption: band}, st)
	skinEngine{}.DrawCaptionTitle(lk, ctx, band, "MINIM", st)

	// The groove's cream row is the brightest thing on the band. It runs
	// across the band well to the left of the title and is gone in the
	// middle, where the plate is.
	creamAt := func(x int) bool {
		for y := 0; y < H; y++ {
			if r, g, b, _ := img.PremulAt(x, y); r > 240 && g > 220 && b > 140 && b < 200 {
				return true
			}
		}
		return false
	}
	if !creamAt(40) {
		t.Error("no groove to the left of the title")
	}
	if !creamAt(W - 50) {
		t.Error("no groove to the right of the title")
	}
	// The words are set at the caption role's own nine design pixels.
	tw := BakeFamily(lk.uiFamily, WeightBold, 9, paintengine2d.Color{}).Advance("MINIM")
	for x := int(W/2 - tw/2); x < int(W/2+tw/2); x++ {
		if creamAt(x) {
			t.Fatalf("the groove runs under the title at x=%d", x)
		}
	}
}
