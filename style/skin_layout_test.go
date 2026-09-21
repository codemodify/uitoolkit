package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The fixed-layout half of the format: named slots at design coordinates,
// what a panel laid out by them measures and where each slot lands at every
// scale, and the art a slot gives the control in it.

const skinLayoutDoc = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
    "mark": { "sheet": "chrome", "at": [16, 0, 8, 8] }
  },
  "layouts": {
    "probe.panel": {
      "size": [100, 40],
      "art": "face",
      "slots": {
        "play":  { "at": [10, 5, 20, 12], "art": { "normal": "face", "pressed": "mark" } },
        "badge": { "at": [4, 2, 16, 10], "fromRight": true },
        "foot":  { "at": [0, 6, 0, 8], "stretchX": true, "fromBottom": true },
        "well":  { "at": [40, 5, 40, 20] }
      }
    }
  }
}`

func layoutTestLook(t *testing.T, scale float32) LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	installSkin(t, "slotted", skinLayoutDoc)
	p, ok := LoadTheme("slotted")
	if !ok {
		t.Fatal("the slotted skin does not load")
	}
	return WithScale(p.Look(), scale)
}

func TestSkinLayoutsLoad(t *testing.T) {
	sk := loadTestSkin(t, skinLayoutDoc)
	lay := sk.Layouts["probe.panel"]
	if lay == nil || lay.W != 100 || lay.H != 40 {
		t.Fatalf("layout %+v", lay)
	}
	if got := lay.SlotNames(); len(got) != 4 || got[0] != "badge" || got[3] != "well" {
		t.Fatalf("slots %v", got)
	}
	play := lay.Slots["play"]
	if play.Art.art("pressed") != sk.Sprites["mark"] || play.Art.art("hover") != sk.Sprites["face"] {
		t.Fatal("a slot's art does not resolve along the state fallback chain")
	}
	if lay.Slots["well"].Art != nil {
		t.Fatal("a slot with no art has some")
	}
	if !lay.Slots["badge"].Rect.FromRight || !lay.Slots["foot"].Rect.StretchX {
		t.Fatal("slot anchors are not read")
	}
}

func TestSkinLayoutRefusalsNameTheirKey(t *testing.T) {
	head := `{"skin": 1, "sheets": {"chrome": {"1x": "art/sheet.png"}},
		"sprites": {"mark": {"at": [16, 0, 8, 8]}}, "layouts": `
	cases := []struct{ name, layouts, wantKey string }{
		{"no size", `{"p": {"slots": {}}}`, "layouts.p.size"},
		{"a flat size", `{"p": {"size": [10, 0]}}`, "layouts.p.size"},
		{"a slot with no rect", `{"p": {"size": [10, 10], "slots": {"a": {}}}}`, "layouts.p.slots.a.at"},
		{"a slot art that is no sprite", `{"p": {"size": [10, 10], "slots": {"a": {"at": [0, 0, 1, 1], "art": "nope"}}}}`, "layouts.p.slots.a.art"},
		{"a slot art state that is no state", `{"p": {"size": [10, 10], "slots": {"a": {"at": [0, 0, 1, 1], "art": {"normal": "mark", "lit": "mark"}}}}}`, "layouts.p.slots.a.art.lit"},
		{"a slot art with no normal", `{"p": {"size": [10, 10], "slots": {"a": {"at": [0, 0, 1, 1], "art": {"hover": "mark"}}}}}`, "layouts.p.slots.a.art"},
		{"stretching and pinned", `{"p": {"size": [10, 10], "slots": {"a": {"at": [0, 0, 1, 1], "stretchY": true, "fromBottom": true}}}}`, "layouts.p.slots.a"},
		{"a key the format does not have", `{"p": {"size": [10, 10], "slots": {"a": {"at": [0, 0, 1, 1], "onClick": "quit"}}}}`, "layouts.p.slots.a"},
	}
	for _, c := range cases {
		_, err := LoadSkinFS(skinTestFS(t, head+c.layouts+`}`), "probe")
		if err == nil {
			t.Errorf("%s: loaded", c.name)
			continue
		}
		if keyOf(err) != c.wantKey {
			t.Errorf("%s: %v (key %q), want key %q", c.name, err, keyOf(err), c.wantKey)
		}
	}
}

// A slot is design pixels times the scale, pinned where it says, in the box
// it is asked about — whatever size that box is.
func TestSkinSlotsLandWhereTheySayAtEveryScale(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := layoutTestLook(t, scale)
		sz, ok := SkinLayoutSize(lk, "probe.panel")
		if !ok || sz != paintengine2d.Pt(100*scale, 40*scale) {
			t.Fatalf("%gx: layout size %v (%v)", scale, sz, ok)
		}
		for _, box := range []paintengine2d.Rect{
			paintengine2d.XYWH(0, 0, sz.X, sz.Y),
			paintengine2d.XYWH(7, 3, sz.X+60, sz.Y+30), // a bigger box, somewhere else
		} {
			s := scale
			for slot, want := range map[string]paintengine2d.Rect{
				"play":  paintengine2d.XYWH(box.Min.X+10*s, box.Min.Y+5*s, 20*s, 12*s),
				"badge": paintengine2d.XYWH(box.Max.X-(4+16)*s, box.Min.Y+2*s, 16*s, 10*s),
				"foot":  paintengine2d.XYWH(box.Min.X, box.Max.Y-(6+8)*s, box.Dx(), 8*s),
			} {
				got, ok := SkinSlotRect(lk, "probe.panel", slot, box)
				if !ok || got != want {
					t.Errorf("%gx in %v: %s at %v (%v), want %v", scale, box, slot, got, ok, want)
				}
			}
		}
		if _, ok := SkinSlotRect(lk, "probe.panel", "no-such-slot", paintengine2d.XYWH(0, 0, 10, 10)); ok {
			t.Errorf("%gx: a slot the layout does not have has a rect", scale)
		}
	}
	plain, _ := LoadTheme("breeze-night")
	if _, ok := SkinLayoutOf(plain.Look(), "probe.panel"); ok {
		t.Error("a look that is not a skin has a layout")
	}
}

// A slot's art is painted per state, falling back as a part's does, and a
// slot with none says so, so its control can paint itself.
func TestASlotPaintsItsArtPerState(t *testing.T) {
	lk := layoutTestLook(t, 1)
	paint := func(slot string, st ControlState) (*paintengine2d.Image, bool) {
		img := paintengine2d.NewImage(40, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		ok := DrawSkinSlot(lk, ctx, paintengine2d.XYWH(0, 0, 20, 12), "probe.panel", slot, st)
		return img, ok
	}
	rest, ok := paint("play", StateNone)
	if !ok {
		t.Fatal("the play slot painted nothing")
	}
	if r, g, b, _ := rest.PremulAt(1, 1); r < 250 || g > 5 || b > 5 {
		t.Errorf("at rest the corner is %d,%d,%d, want the face's red", r, g, b)
	}
	down, _ := paint("play", StatePressed)
	if r, g, b, _ := down.PremulAt(1, 1); r < 250 || g < 250 || b < 250 {
		t.Errorf("held down the corner is %d,%d,%d, want the white mark", r, g, b)
	}
	hover, _ := paint("play", StateHovered)
	if diffPixels(hover, rest) != 0 {
		t.Error("hover has no art of its own and did not fall back to the resting face")
	}
	if _, ok := paint("well", StateNone); ok {
		t.Error("a slot with no art reported that it painted")
	}
	img := paintengine2d.NewImage(100, 40)
	if !DrawSkinLayout(lk, paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 100, 40), "probe.panel") {
		t.Error("the layout's own art did not paint")
	}
}
