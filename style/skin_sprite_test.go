package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An app that lays out a panel of its own paints the skin's loose sprites by
// name, by the same rules a part is painted by: the ink lands in the box it
// was given, at every scale, and nowhere else.
func TestAnAppPaintsASkinSpriteByName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme("deck")
	if !ok {
		t.Skipf("the %q pack is not in this build", "deck")
	}
	w, h, ok := SkinSpriteSize(p.Look(), "button.normal")
	if !ok || w != 72 || h != 36 {
		t.Fatalf("button.normal is %gx%g (%v), want 72x36", w, h, ok)
	}
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		InvalidateSkinCache()
		lk := WithScale(p.Look(), scale)
		img := paintengine2d.NewImage(80, 80)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		b := paintengine2d.XYWH(20, 20, w*scale, h*scale)
		if !DrawSkinSprite(lk, ctx, b, "button.normal", paintengine2d.Color{}) {
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
	if DrawSkinSprite(plain.Look(), ctx, paintengine2d.XYWH(0, 0, 40, 40), "button.normal", paintengine2d.Color{}) {
		t.Error("a look that is not a skin painted a sprite")
	}
	if _, _, ok := SkinSpriteSize(plain.Look(), "button.normal"); ok {
		t.Error("a look that is not a skin has a sprite size")
	}
}

// A sprite an app paints has a silhouette as a part's face does: Deck's
// round keys are discs in square boxes, so their corners are not the key,
// at every scale; a sprite that fills its box, a sprite the skin does not
// have and a look that is not a skin all answer nil — the box.
func TestAnAppPaintedSpriteHasItsShape(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme("deck")
	if !ok {
		t.Skipf("the %q pack is not in this build", "deck")
	}
	w, h, ok := SkinSpriteSize(p.Look(), "capbtn.normal")
	if !ok {
		t.Fatal("no capbtn.normal")
	}
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := WithScale(p.Look(), scale)
		size := paintengine2d.Pt(w*scale, h*scale)
		s := SkinSpriteShape(lk, size, paintengine2d.XYWH(0, 0, size.X, size.Y), "capbtn.normal")
		if s == nil || s.Mask == nil {
			t.Fatalf("%gx: a round key has no silhouette", scale)
		}
		at := func(x, y int) bool {
			m := s.Mask
			return m.Pix[y*m.RowStride()+x] > 0
		}
		cx, cy := int(size.X/2), int(size.Y/2)
		if !at(cx, cy) {
			t.Errorf("%gx: the middle of the key is not the key", scale)
		}
		if at(0, 0) || at(s.Mask.Width-1, s.Mask.Height-1) {
			t.Errorf("%gx: a corner of a round key's box is the key", scale)
		}
	}
	lk := p.Look()
	full := paintengine2d.Pt(40, 40)
	if SkinSpriteShape(lk, full, paintengine2d.XYWH(0, 0, 40, 40), "no.such.sprite") != nil {
		t.Error("a sprite the skin does not have has a silhouette")
	}
	plain, _ := LoadTheme("breeze-night")
	if SkinSpriteShape(plain.Look(), full, paintengine2d.XYWH(0, 0, 40, 40), "capbtn.normal") != nil {
		t.Error("a look that is not a skin shaped a sprite")
	}
}

// shortCapDoc is a skin whose caption is shorter than the 24 design pixels
// a frame would otherwise insist on for a caption button, and which binds a
// plate behind its title. The skins that used to exercise both — the music
// player's panel faces — went with the player, so the rules are pinned here
// on a skin built for the purpose rather than on anybody's art.
const shortCapDoc = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
    "mark": { "sheet": "chrome", "at": [16, 0, 8, 8] }
  },
  "text": { "caption": { "color": "#ff00ff", "size": 9 } },
  "parts": {
    "caption":        { "states": { "normal": "face" }, "text": "caption" },
    "caption.button": { "states": { "normal": "mark" } },
    "caption.title":  { "states": { "normal": "mark" } }
  },
  "window": { "border": { "caption": [0, 1, 0, 1], "content": [0, 1, 1, 1] }, "caption": %CAP% }
}`

// A skin states a caption shorter than a caption button's usual minimum and
// gets exactly that: the band is what it asked for in whole device pixels,
// and the buttons shrink to stand in it rather than growing it back.
func TestASkinsShortCaptionIsTheOneItStates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, caption := range []float32{14, 15, 18} {
		name := fmt.Sprintf("shortcap%g", caption)
		installSkin(t, name, strings.Replace(shortCapDoc, "%CAP%", fmt.Sprintf("%g", caption), 1))
		p, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("%s does not load", name)
		}
		for _, scale := range []float32{1, 1.75, 2} {
			lk := WithScale(p.Look(), scale)
			d := DecorationOf(lk, DecorationState{Active: true})
			want := float32(int(caption*scale + 0.5))
			if d.Caption != want {
				t.Errorf("caption %g@%gx: band %g, want %g", caption, scale, d.Caption, want)
			}
			if d.Button.Y <= 0 || d.ButtonPad.Top+d.Button.Y > d.Caption {
				t.Errorf("caption %g@%gx: a %gx%g button %g down does not fit a %g band",
					caption, scale, d.Button.X, d.Button.Y, d.ButtonPad.Top, d.Caption)
			}
		}
	}
}

// A skin that binds caption.title gets a plate behind the words, as wide as
// the words and no wider: the band's own art runs to either side of it and
// stops where the plate starts. That is the gap in a ribbed caption.
func TestTheCaptionTitleStandsOnAPlate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	installSkin(t, "platecap", strings.Replace(shortCapDoc, "%CAP%", "14", 1))
	p, ok := LoadTheme("platecap")
	if !ok {
		t.Fatal("platecap does not load")
	}
	lk := p.Look()
	sk := skinFor(lk)
	if sk == nil || !sk.has("caption.title") {
		t.Fatal("platecap binds no caption.title")
	}

	const W, H = 275, 14
	img := paintengine2d.NewImage(W, H)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	band := paintengine2d.XYWH(0, 0, W, H)
	st := DecorationState{Active: true}
	skinEngine{}.DrawDecoration(lk, ctx, DecorationFrame{Window: paintengine2d.XYWH(0, 0, W, 40), Caption: band}, st)
	skinEngine{}.DrawCaptionTitle(lk, ctx, band, "PLATE", st)

	// The plate is the sheet's solid-white cell; the band is the sliced one,
	// which has no white in it, and the title is set in magenta so the words
	// are not mistaken for their plate. So "is there white in this column"
	// reads the plate's extent straight off the drawing.
	plateAt := func(x int) bool {
		for y := 0; y < H; y++ {
			if r, g, b, _ := img.PremulAt(x, y); r > 240 && g > 240 && b > 240 {
				return true
			}
		}
		return false
	}
	if plateAt(2) || plateAt(W-3) {
		t.Error("the plate runs to the band's ends; it should be as wide as the title")
	}
	if !plateAt(W / 2) {
		t.Error("no plate under the title")
	}
}
