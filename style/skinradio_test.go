package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// radioSkinDoc is a skin that binds the two wells to art that cannot be
// confused: the check box's is the sliced, coloured cell, the radio's is
// the plain white one. Bind only `check` by leaving the second part out.
func radioSkinDoc(radio bool) string {
	parts := `"check": { "states": { "normal": "face" } }`
	if radio {
		parts += `, "radio": { "states": { "normal": "plain" } }`
	}
	return `{
  "skin": 1, "label": "Radio probe", "family": "dark", "base": "dark",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "face":  { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
    "plain": { "sheet": "chrome", "at": [16, 0, 8, 8] }
  },
  "parts": { ` + parts + ` }
}`
}

func wellShots(t *testing.T, name, doc string) (check, radio *paintengine2d.Image) {
	t.Helper()
	installSkin(t, name, doc)
	pack, ok := LoadTheme(name)
	if !ok {
		t.Skipf("the %q pack is not in this build", name)
	}
	lk := pack.Look()
	if id := lk.Engine().ID(); id != skinEngineID {
		t.Fatalf("the look paints with %q, not the skin engine", id)
	}
	shot := func(draw func(*paintengine2d.Context)) *paintengine2d.Image {
		img := paintengine2d.NewImage(24, 24)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		draw(ctx)
		img.Touch()
		return img
	}
	// The two wells are reached the way a widget reaches them, through
	// the look's own draw calls; a label would put text in the picture,
	// so there is none.
	if m := lk.Metrics(); m.Radio != m.Checkbox {
		t.Skipf("the base pack's radio is %g and its check box %g; this compares the art, not the size", m.Radio, m.Checkbox)
	}
	box := paintengine2d.XYWH(2, 2, 20, 20)
	check = shot(func(ctx *paintengine2d.Context) {
		lk.DrawCheckbox(ctx, box, StateNone, false, "")
	})
	radio = shot(func(ctx *paintengine2d.Context) {
		lk.DrawRadio(ctx, box, StateNone, false, "")
	})
	return check, radio
}

// **A skin can say what its radios' well looks like.** There was one part
// for both wells, so a skin that bound it painted its radios in its check
// boxes' well: square, where a radio is round in nearly every era, and no
// skin could say otherwise.
func TestASkinsRadioHasAWellOfItsOwn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	check, radio := wellShots(t, "radioprobe", radioSkinDoc(true))
	if n := diffPixels(check, radio); n < 40 {
		t.Errorf("the radio differs from the check box in %d pixels: it is still painting the check well", n)
	}
}

// And a skin that binds only `check` keeps painting as it did: every skin
// written before the part existed is one of these.
func TestASkinWithoutARadioPartPaintsTheCheckWell(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	check, radio := wellShots(t, "radiofallback", radioSkinDoc(false))
	if n := diffPixels(check, radio); n != 0 {
		t.Errorf("with no radio part the two wells differ in %d pixels, want the check well for both", n)
	}
}

// The part is in the vocabulary a skin writer reads: SkinPartNames lists
// it and SkinPartDoc says what it paints.
func TestTheRadioPartIsInTheVocabulary(t *testing.T) {
	found := false
	for _, n := range SkinPartNames() {
		if n == "radio" {
			found = true
		}
	}
	if !found {
		t.Error("SkinPartNames does not list radio")
	}
	if SkinPartDoc("radio") == "" {
		t.Error("SkinPartDoc(radio) says nothing")
	}
}
