package style

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/codemodify/paintengine2d"
)

// ---- fixtures -------------------------------------------------------------

// skinTestPNG draws a sheet whose cells are solid, known colours, so a test
// can read a pixel back and say which piece of which sprite painted it.
//
// The one sprite it holds is a 12×12 cell at (0,0) with a 4px slice: the
// corners are red, the edges green and the middle blue. Stretch it and the
// corners must stay their own size and keep their colour.
//
// scale draws the same picture bigger, which is what a real skin's @2x sheet
// is — and is why the fixture cannot just reuse its 1× bytes for the 2×
// asset: the loader scales every rect by the asset's scale, so a 2× file
// that is really a 1× one has its sprites read twice their size.
func skinTestPNG(t *testing.T, scale float32) []byte {
	t.Helper()
	n := func(v float32) float32 { return v * scale }
	img := paintengine2d.NewImage(int(n(32)), int(n(32)))
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	red := paintengine2d.RGB(1, 0, 0)
	green := paintengine2d.RGB(0, 1, 0)
	blue := paintengine2d.RGB(0, 0, 1)
	// The 12×12 cell at (0,0): a 4px frame of red corners and green edges
	// round a blue middle.
	ctx.DrawRect(paintengine2d.XYWH(0, 0, n(12), n(12)), paintengine2d.Fill(green))
	ctx.DrawRect(paintengine2d.XYWH(n(4), n(4), n(4), n(4)), paintengine2d.Fill(blue))
	for _, c := range [][2]float32{{0, 0}, {8, 0}, {0, 8}, {8, 8}} {
		ctx.DrawRect(paintengine2d.XYWH(n(c[0]), n(c[1]), n(4), n(4)), paintengine2d.Fill(red))
	}
	// A second, unsliced cell at (16,0): solid white, for the glyph path.
	ctx.DrawRect(paintengine2d.XYWH(n(16), 0, n(8), n(8)), paintengine2d.Fill(paintengine2d.RGB(1, 1, 1)))
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return buf.Bytes()
}

// skinTestFS is a one-sheet skin whose manifest is doc, with a real 1× and
// 2× pair.
func skinTestFS(t *testing.T, doc string) fstest.MapFS {
	t.Helper()
	return fstest.MapFS{
		SkinFile:         &fstest.MapFile{Data: []byte(doc)},
		"art/sheet.png":  &fstest.MapFile{Data: skinTestPNG(t, 1)},
		"art/sheet2.png": &fstest.MapFile{Data: skinTestPNG(t, 2)},
	}
}

// skinTestDoc is a valid manifest exercising sheets, sprites, a strip, a
// text role and a window.
const skinTestDoc = `{
  "skin": 1,
  "label": "Probe", "year": 2026, "lineage": "test",
  "family": "dark", "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
    "mark": { "sheet": "chrome", "at": [16, 0, 8, 8], "tint": true }
  },
  "text": { "control": { "color": "#ffffff", "disabled": "#808080" } },
  "parts": {
    "button": { "states": { "normal": "face" }, "text": "control" },
    "check.mark": { "states": { "normal": "mark" } }
  },
  "window": { "border": [1, 1, 1, 1], "caption": 24 }
}`

func loadTestSkin(t *testing.T, doc string) *Skin {
	t.Helper()
	sk, err := LoadSkinFS(skinTestFS(t, doc), "probe")
	if err != nil {
		t.Fatalf("LoadSkinFS: %v", err)
	}
	return sk
}

// ---- the loader -----------------------------------------------------------

func TestLoadSkinReadsAManifest(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	if sk.Name != "probe" || sk.Label != "Probe" || sk.Year != 2026 || sk.Lineage != "test" {
		t.Fatalf("identity: %+v", sk)
	}
	if sk.Base != "breeze-night" || sk.Family != ThemeDark {
		t.Fatalf("base %q family %q", sk.Base, sk.Family)
	}
	if len(sk.Sheets) != 1 || len(sk.Sprites) != 2 || len(sk.Parts) != 2 {
		t.Fatalf("sheets %d sprites %d parts %d", len(sk.Sheets), len(sk.Sprites), len(sk.Parts))
	}
	sh := sk.Sheets["chrome"]
	if got := sh.Scales(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("scale set %v", got)
	}
	if !sh.Covers(2) || sh.Covers(3) {
		t.Fatalf("Covers: 2=%v 3=%v", sh.Covers(2), sh.Covers(3))
	}
	sp := sk.Sprites["face"]
	if !sp.Sliced() || sp.Slice != (Insets{Top: 4, Right: 4, Bottom: 4, Left: 4}) {
		t.Fatalf("slice %+v", sp.Slice)
	}
	if !sk.Sprites["mark"].Tint {
		t.Fatal("mark should be a tinted glyph")
	}
	if sk.Window == nil || sk.Window.Caption != 24 {
		t.Fatalf("window %+v", sk.Window)
	}
	// The pack it presents to Settings is a pack like any other.
	p := sk.Pack()
	if p.Tokens.Engine != skinEngineID || p.Name != "probe" || p.Era != EraSkin {
		t.Fatalf("pack %+v", p)
	}
	if p.Tokens.Bevel == "" {
		t.Fatal("a pack with no bevel fails TestLoadThemeEveryEmbeddedPack")
	}
}

// Every refusal names the key it came from. That is the whole point of the
// error type: a skin is a stranger's file and "invalid character" is not a
// bug report anybody can act on.
func TestLoadSkinRefusalsNameTheirKey(t *testing.T) {
	sheets := `"sheets": { "chrome": { "1x": "art/sheet.png" } }`
	cases := []struct {
		name, doc, wantKey, wantMsg string
	}{
		{"no version", `{` + sheets + `}`, "skin", "format version is required"},
		{"future version", `{"skin": 99, ` + sheets + `}`, "skin", "newer than this build"},
		{"zero version", `{"skin": -1, ` + sheets + `}`, "skin", "not a version"},
		{"unknown top-level key", `{"skin": 1, "sheetz": {}}`, "", `unknown key "sheetz"`},
		{"unknown part key", `{"skin": 1, ` + sheets + `,
			"parts": {"button": {"stats": {}}}}`, "parts.button", `unknown key "stats"`},
		{"unknown part name", `{"skin": 1, ` + sheets + `,
			"parts": {"widget": {"states": {}}}}`, "parts.widget", "not a part of this toolkit"},
		{"unknown state", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,8,8]}},
			"parts": {"button": {"states": {"hovered": "f"}}}}`, "parts.button.states.hovered", "not a control state"},
		{"missing sprite", `{"skin": 1, ` + sheets + `,
			"parts": {"button": {"states": {"normal": "nope"}}}}`, "parts.button.states.normal", `no sprite "nope"`},
		{"no normal state", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,8,8]}},
			"parts": {"button": {"states": {"hover": "f"}}}}`, "parts.button.states", `needs "normal"`},
		{"missing sheet", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "other", "at": [0,0,8,8]}}}`, "sprites.f.sheet", `no sheet "other"`},
		{"missing art file", `{"skin": 1, "sheets": {"chrome": {"1x": "art/gone.png"}}}`,
			"sheets.chrome.1x", "missing art file"},
		{"art outside the skin", `{"skin": 1, "sheets": {"chrome": {"1x": "../../etc/passwd"}}}`,
			"sheets.chrome.1x", "leaves the skin's directory"},
		{"sheet with no art", `{"skin": 1, "sheets": {"chrome": {}}}`, "sheets.chrome", "names no art"},
		{"bad scale key", `{"skin": 1, "sheets": {"chrome": {"huge": "art/sheet.png"}}}`,
			"sheets.chrome.huge", "must be a positive scale"},
		{"short at", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,8]}}}`, "sprites.f.at", "needs 4 numbers"},
		{"empty sprite", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,0,8]}}}`, "sprites.f.at", "is empty"},
		{"slice with no middle", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,8,8], "slice": [4,4,4,4]}}}`,
			"sprites.f.slice", "leaves no middle"},
		{"bad middle", `{"skin": 1, ` + sheets + `,
			"sprites": {"f": {"sheet": "chrome", "at": [0,0,8,8], "middle": "smear"}}}`,
			"sprites.f.middle", "must be stretch, tile or none"},
		{"bad colour", `{"skin": 1, ` + sheets + `, "colors": {"text": "not a colour"}}`,
			"colors.text", "is not a colour"},
		{"bad text colour", `{"skin": 1, ` + sheets + `, "text": {"control": {"color": "zz"}}}`,
			"text.control.color", "is not a colour"},
		{"empty strip", `{"skin": 1, ` + sheets + `,
			"parts": {"button": {"strip": {"sheet": "chrome", "at": [0,0,8,8], "states": []}}}}`,
			"parts.button.strip.states", "names no states"},
		{"empty shape rect", `{"skin": 1, ` + sheets + `,
			"window": {"shape": [{"at": [0,0,0,8]}]}}`, "window.shape[0].at", "is empty"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadSkinFS(skinTestFS(t, c.doc), "probe")
			if err == nil {
				t.Fatal("loaded a skin that should have been refused")
			}
			se, ok := err.(*SkinError)
			if !ok {
				t.Fatalf("error %v is not a *SkinError", err)
			}
			if se.Key != c.wantKey {
				t.Errorf("key %q, wanted %q (%v)", se.Key, c.wantKey, err)
			}
			if !strings.Contains(se.Msg, c.wantMsg) {
				t.Errorf("message %q does not mention %q", se.Msg, c.wantMsg)
			}
			if se.Skin != "probe" {
				t.Errorf("error does not name the skin: %v", err)
			}
		})
	}
}

// A strip is the shorthand real sheets are drawn for, and its cells have to
// land where the art is.
func TestSkinStripLaysOutCells(t *testing.T) {
	doc := `{"skin": 1,
		"sheets": {"chrome": {"1x": "art/sheet.png"}},
		"parts": {"button": {"strip": {"sheet": "chrome", "at": [0, 0, 8, 8], "gap": 2,
			"states": ["normal", "-", "pressed"]}}}}`
	sk := loadTestSkin(t, doc)
	p := sk.Parts["button"]
	if p.States["normal"].X != 0 {
		t.Errorf("first cell at x=%g", p.States["normal"].X)
	}
	if p.States["pressed"].X != 20 { // two cells of 8 plus two gaps of 2
		t.Errorf("third cell at x=%g, wanted 20", p.States["pressed"].X)
	}
	if _, skipped := p.States["-"]; skipped {
		t.Error(`"-" should skip a cell, not bind one`)
	}
	if p.States["hover"] != nil {
		t.Error("a skipped cell must not bind a state")
	}
}

// ---- state maps -----------------------------------------------------------

// Art for "normal" alone gives a control every state. This is what lets a
// skin be written a sprite at a time instead of all at once.
func TestSkinStateFallback(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	p := sk.Parts["button"]
	for _, state := range SkinStateNames() {
		if p.art(state) == nil {
			t.Errorf("state %q resolves to nothing with only normal art", state)
		}
	}

	// With a fuller set, each state takes the nearest thing it was given.
	doc := `{"skin": 1,
		"sheets": {"chrome": {"1x": "art/sheet.png"}},
		"sprites": {
			"n": {"sheet": "chrome", "at": [0, 0, 8, 8]},
			"h": {"sheet": "chrome", "at": [8, 0, 8, 8]},
			"c": {"sheet": "chrome", "at": [16, 0, 8, 8]}},
		"parts": {"button": {"states": {"normal": "n", "hover": "h", "checked": "c"}}}}`
	sk2 := loadTestSkin(t, doc)
	p2 := sk2.Parts["button"]
	for _, c := range []struct{ state, want string }{
		{"normal", "n"}, {"hover", "h"}, {"checked", "c"},
		{"pressed", "h"},        // pressed → hover
		{"focus", "h"},          // focus → hover
		{"default", "h"},        // default → hover
		{"disabled", "n"},       // disabled → normal
		{"inactive", "n"},       // inactive → disabled → normal
		{"checkedHover", "c"},   // checkedHover → checked
		{"checkedPressed", "c"}, // checkedPressed → checked
	} {
		got := p2.art(c.state)
		if got == nil || got.Name != c.want {
			t.Errorf("state %q resolved to %v, wanted %q", c.state, spriteName(got), c.want)
		}
	}
}

func spriteName(sp *SkinSprite) string {
	if sp == nil {
		return "<nothing>"
	}
	return sp.Name
}

// The state a ControlState maps onto, in the order a person reads a control.
func TestSkinStateNaming(t *testing.T) {
	for _, c := range []struct {
		st   ControlState
		want string
	}{
		{StateNone, "normal"},
		{StateHovered, "hover"},
		{StatePressed, "pressed"},
		{StateDisabled, "disabled"},
		{StateDisabled | StateHovered | StateChecked, "disabled"},
		{StateChecked, "checked"},
		{StateChecked | StateHovered, "checkedHover"},
		{StateChecked | StatePressed, "checkedPressed"},
		{StatePrimary, "default"},
		{StateFocused, "focus"},
		{StateBackdrop, "inactive"},
	} {
		if got := skinStateName(c.st); got != c.want {
			t.Errorf("state %d → %q, wanted %q", c.st, got, c.want)
		}
	}
}

// ---- nine-slice geometry --------------------------------------------------

// A nine-slice's corners keep their size and their colour however far the
// middle is stretched, at every scale — which is the property that lets one
// 12×12 sprite be a button of any width in any language.
func TestSkinNineSliceKeepsItsCorners(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	sp := sk.Sprites["face"]
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		t.Run(fmt.Sprintf("%gx", scale), func(t *testing.T) {
			InvalidateSkinCache()
			img := paintengine2d.NewImage(200, 60)
			ctx := paintengine2d.NewContext(img)
			ctx.Clear(paintengine2d.Color{})
			box := paintengine2d.XYWH(10, 10, 180, 40)
			if !sk.skinDraw(ctx, box, sp, scale, paintengine2d.Color{}) {
				t.Fatal("nothing painted")
			}
			// The fixed corner is 4 design pixels; at the display scale it is
			// a whole number of device pixels, at least one.
			corner := int(4 * scale)
			at := func(x, y int) (r, g, b uint8) {
				pr, pg, pb, _ := img.PremulAt(x, y)
				return pr, pg, pb
			}
			isRed := func(x, y int) bool { r, g, b := at(x, y); return r > 180 && g < 70 && b < 70 }
			isBlue := func(x, y int) bool { r, g, b := at(x, y); return b > 180 && r < 70 && g < 70 }
			isGreen := func(x, y int) bool { r, g, b := at(x, y); return g > 180 && r < 70 && b < 70 }

			for _, c := range [][2]int{{11, 11}, {188, 11}, {11, 48}, {188, 48}} {
				if !isRed(c[0], c[1]) {
					r, g, b := at(c[0], c[1])
					t.Errorf("corner (%d,%d) is %d,%d,%d, wanted red", c[0], c[1], r, g, b)
				}
			}
			// Just inside the corner, along the top, is the green edge.
			if x := 10 + corner + 2; !isGreen(x, 11) {
				r, g, b := at(x, 11)
				t.Errorf("top edge at x=%d is %d,%d,%d, wanted green", x, r, g, b)
			}
			// The centre is the blue middle, stretched right across.
			for _, x := range []int{60, 100, 140} {
				if !isBlue(x, 30) {
					r, g, b := at(x, 30)
					t.Errorf("middle at x=%d is %d,%d,%d, wanted blue", x, r, g, b)
				}
			}
			// And the seam the padding exists to prevent: no transparent or
			// dimmed column anywhere across the middle band.
			for x := 10 + corner + 2; x < 190-corner-2; x++ {
				if _, _, _, a := img.PremulAt(x, 30); a < 250 {
					t.Fatalf("a gap at x=%d (alpha %d): the stretched middle is fading at a seam", x, a)
				}
			}
		})
	}
}

// A box too small for its own corners shrinks them rather than letting them
// overlap: a 12×12 sprite drawn into a 6px-tall track still paints.
func TestSkinNineSliceSurvivesASmallBox(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	InvalidateSkinCache()
	img := paintengine2d.NewImage(40, 10)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	if !sk.skinDraw(ctx, paintengine2d.XYWH(0, 2, 40, 6), sk.Sprites["face"], 1, paintengine2d.Color{}) {
		t.Fatal("nothing painted into a short box")
	}
	painted := 0
	for y := 2; y < 8; y++ {
		for x := 0; x < 40; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 200 {
				painted++
			}
		}
	}
	if painted < 40*6/2 {
		t.Fatalf("only %d of %d pixels painted in a short box", painted, 40*6)
	}
}

// Nothing a sprite paints may land outside the box it was given.
func TestSkinDrawsInsideItsBox(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	for _, name := range []string{"face", "mark"} {
		for _, scale := range []float32{1, 1.75, 2} {
			InvalidateSkinCache()
			img := paintengine2d.NewImage(120, 60)
			ctx := paintengine2d.NewContext(img)
			ctx.Clear(paintengine2d.Color{})
			box := paintengine2d.XYWH(20, 15, 80, 30)
			sk.skinDraw(ctx, box, sk.Sprites[name], scale, paintengine2d.RGB(1, 1, 1))
			for y := 0; y < 60; y++ {
				for x := 0; x < 120; x++ {
					inside := float32(x) >= box.Min.X-1 && float32(x) < box.Max.X+1 &&
						float32(y) >= box.Min.Y-1 && float32(y) < box.Max.Y+1
					if inside {
						continue
					}
					if _, _, _, a := img.PremulAt(x, y); a > 8 {
						t.Fatalf("%s @%gx painted (%d,%d) outside %v", name, scale, x, y, box)
					}
				}
			}
		}
	}
}

// ---- assets ---------------------------------------------------------------

// Art is chosen upward and drawn down: never enlarged.
func TestSkinChoosesTheAssetAtOrAboveTheScale(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	sh := sk.Sheets["chrome"]
	for _, c := range []struct {
		target float32
		want   float32
	}{
		{0.5, 1}, {1, 1}, {1.25, 2}, {1.75, 2}, {2, 2},
		{3, 2}, // nothing bigger exists: the largest, rather than nothing
	} {
		got, _, ok := sh.pickScale(c.target)
		if !ok || got != c.want {
			t.Errorf("target %g chose %g, wanted %g", c.target, got, c.want)
		}
	}
}

// A pixelated sheet is sampled at whole multiples, so its pixels stay square
// at a fractional display scale.
func TestSkinPixelatedUsesWholeMultiples(t *testing.T) {
	doc := `{"skin": 1, "design": {"pixelated": true},
		"sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/sheet2.png"}},
		"sprites": {"f": {"sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4,4,4,4]}},
		"parts": {"button": {"states": {"normal": "f"}}}}`
	sk := loadTestSkin(t, doc)
	sp := sk.Sprites["f"]
	if !sp.Sheet.Pixelated {
		t.Fatal("design.pixelated should reach the sheet")
	}
	for _, c := range []struct{ scale, want float32 }{
		{1, 1}, {1.25, 1}, {1.75, 1}, {2, 2}, {2.5, 2},
	} {
		if got := sk.assetTarget(sp, c.scale); got != c.want {
			t.Errorf("at %gx a pixelated sheet asks for %g, wanted %g", c.scale, got, c.want)
		}
	}
	// And it draws hard pixels: every painted sample is one of the fixture's
	// three flat colours, never a blend of two.
	InvalidateSkinCache()
	img := paintengine2d.NewImage(80, 40)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	sk.skinDraw(ctx, paintengine2d.XYWH(0, 0, 80, 40), sp, 1.75, paintengine2d.Color{})
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a == 0 {
				continue
			}
			flat := (r > 200 && g < 40 && b < 40) || (g > 200 && r < 40 && b < 40) || (b > 200 && r < 40 && g < 40)
			if !flat {
				t.Fatalf("pixel (%d,%d) = %d,%d,%d is a blend: a pixelated sheet was filtered", x, y, r, g, b)
			}
		}
	}
}

// Two skins that both call their sheet art/chrome.png must not read each
// other's pixels. Both built-in skins do exactly that, and keying the asset
// cache on the path alone made Cassette paint from Nocturne's sheet.
func TestSkinsWithTheSameArtPathDoNotCollide(t *testing.T) {
	a, err := LoadSkinFS(skinTestFS(t, skinTestDoc), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	// A second skin, same file name, different pixels.
	img := paintengine2d.NewImage(32, 32)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.RGB(1, 1, 0)) // yellow everywhere
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatal(err)
	}
	b, err := LoadSkinFS(fstest.MapFS{
		SkinFile:        &fstest.MapFile{Data: []byte(skinTestDoc)},
		"art/sheet.png": &fstest.MapFile{Data: buf.Bytes()},
		// sheet2 is only referenced as the 2x asset; 1x is what we draw.
		"art/sheet2.png": &fstest.MapFile{Data: buf.Bytes()},
	}, "beta")
	if err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()
	read := func(sk *Skin) (uint8, uint8, uint8) {
		out := paintengine2d.NewImage(40, 40)
		c := paintengine2d.NewContext(out)
		c.Clear(paintengine2d.Color{})
		sk.skinDraw(c, paintengine2d.XYWH(0, 0, 40, 40), sk.Sprites["face"], 1, paintengine2d.Color{})
		r, g, bl, _ := out.PremulAt(20, 20)
		return r, g, bl
	}
	if r, g, bl := read(a); !(bl > 180 && r < 70 && g < 70) {
		t.Fatalf("alpha's middle is %d,%d,%d, wanted its own blue", r, g, bl)
	}
	if r, g, bl := read(b); !(r > 180 && g > 180 && bl < 70) {
		t.Fatalf("beta's middle is %d,%d,%d, wanted its own yellow (it read alpha's sheet)", r, g, bl)
	}
}

// ---- the shipped skins ----------------------------------------------------

// Every skin the toolkit ships loads, binds only parts that exist, and has
// every sprite inside the sheet it names. The loader cannot check the last
// one — it does not decode the art — so this is where it is checked.
func TestBuiltinSkinsAreSound(t *testing.T) {
	skins := ListSkins()
	if len(skins) < 2 {
		t.Fatalf("expected the shipped skins, got %v", skins)
	}
	for _, p := range skins {
		sk, ok := LoadSkin(p.Name)
		if !ok {
			t.Fatalf("%s lists but does not load", p.Name)
		}
		t.Run(sk.Name, func(t *testing.T) {
			if sk.Summary == "" || sk.Year == 0 || sk.Lineage == "" {
				t.Error("a shipped skin needs a year, a lineage and a summary to list properly")
			}
			for name := range sk.Parts {
				if SkinPartDoc(name) == "" {
					t.Errorf("part %q is not in the vocabulary", name)
				}
			}
			// A shipped skin must cover 1x and 2x: the format's required pair.
			for name, sh := range sk.Sheets {
				if !sh.Covers(2) {
					t.Errorf("sheet %q has no asset at or above 2x (%v)", name, sh.Scales())
				}
			}
			for name, sp := range sk.Sprites {
				scale, file, ok := sp.Sheet.pickScale(1)
				if !ok {
					t.Fatalf("sprite %q: sheet names no asset", name)
				}
				img := sk.sheetImage(file)
				if img == nil {
					t.Fatalf("sprite %q: sheet %q does not decode", name, file)
				}
				k := scale / sk.Design.Scale
				x1, y1 := int((sp.X+sp.W)*k+0.5), int((sp.Y+sp.H)*k+0.5)
				if x1 > img.Width || y1 > img.Height {
					t.Errorf("sprite %q runs to (%d,%d), past the %dx%d sheet",
						name, x1, y1, img.Width, img.Height)
				}
			}
			// The rule no skin may break.
			if sk.part("focus").art("normal") == nil {
				t.Error("a shipped skin should draw its own focus ring")
			}
		})
	}
}

// A skinned look paints its controls from art, and paints the ones it does
// not describe in its base pack's look — not in a hole and not in the stock
// grey.
func TestSkinFallsBackToItsBasePack(t *testing.T) {
	sk, ok := LoadSkin("nocturne")
	if !ok {
		t.Skip("nocturne is not registered")
	}
	pack, ok := LoadTheme("nocturne")
	if !ok {
		t.Fatal("nocturne does not load as a pack")
	}
	lk := pack.Look()
	if lk.Engine().ID() != skinEngineID {
		t.Fatalf("look paints with %q", lk.Engine().ID())
	}
	if e := sk.baseEngine(lk); e.ID() == skinEngineID || e.ID() == "base" {
		t.Fatalf("the base engine resolved to %q; a skin should stand on a real pack", e.ID())
	}

	// A bound part paints differently from the base pack alone.
	base, _ := LoadTheme(sk.Base)
	baseLook := base.Look()
	render := func(l *Classic, draw func(*paintengine2d.Context, *Classic)) *paintengine2d.Image {
		img := paintengine2d.NewImage(120, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		draw(ctx, l)
		return img
	}
	button := func(ctx *paintengine2d.Context, l *Classic) {
		l.DrawButton(ctx, paintengine2d.XYWH(5, 5, 110, 30), StateNone, "Ok")
	}
	if diffPixels(render(lk, button), render(baseLook, button)) < 200 {
		t.Error("a skinned button looks like its base pack's: the art is not reaching Face")
	}

}

// The promise in one test: a skin that describes nothing is its base pack,
// pixel for pixel, on every control. That is what makes a half-finished skin
// a working app instead of a grid of holes, and what makes dropping a skin
// at run time safe.
func TestSkinThatDescribesNothingIsItsBasePack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	installSkin(t, "hollow", `{"skin": 1, "label": "Hollow", "base": "win95",
		"sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/sheet2.png"}}}`)

	pack, ok := LoadTheme("hollow")
	if !ok {
		t.Fatal("the hollow skin does not load")
	}
	if pack.Tokens.Engine != skinEngineID {
		t.Fatalf("engine %q", pack.Tokens.Engine)
	}
	base, ok := LoadTheme("win95")
	if !ok {
		t.Skip("win95 is not registered")
	}
	skin, plain := pack.Look(), base.Look()

	draws := []struct {
		name string
		draw func(*paintengine2d.Context, *Classic)
	}{
		{"button", func(c *paintengine2d.Context, l *Classic) {
			l.DrawButton(c, paintengine2d.XYWH(5, 5, 110, 30), StateHovered, "Ok")
		}},
		{"checkbox", func(c *paintengine2d.Context, l *Classic) {
			l.DrawCheckbox(c, paintengine2d.XYWH(5, 5, 110, 30), StateNone, true, "On")
		}},
		{"field", func(c *paintengine2d.Context, l *Classic) {
			l.DrawTextField(c, paintengine2d.XYWH(5, 5, 110, 30), StateFocused, "text", "", 4, 0, 0, false, 0, nil)
		}},
		{"tab", func(c *paintengine2d.Context, l *Classic) {
			l.DrawTab(c, paintengine2d.XYWH(5, 5, 110, 30), StateNone, "Tab", true)
		}},
		{"menu item", func(c *paintengine2d.Context, l *Classic) {
			l.DrawMenuItem(c, paintengine2d.XYWH(5, 5, 110, 30), StateHovered, MenuRow{Label: "Open"})
		}},
		{"slider", func(c *paintengine2d.Context, l *Classic) {
			l.DrawSlider(c, paintengine2d.XYWH(5, 5, 110, 30), StateNone, 0.4)
		}},
		{"switch", func(c *paintengine2d.Context, l *Classic) {
			l.DrawSwitch(c, paintengine2d.XYWH(5, 5, 110, 30), StateNone, true, "On")
		}},
		{"scrollbar", func(c *paintengine2d.Context, l *Classic) {
			p := ScrollGeometry(l, paintengine2d.XYWH(5, 5, 110, 30), true, 300, 100, 20, false)
			DrawScrollBarParts(l, c, p, true, ScrollState{})
		}},
		{"focus ring", func(c *paintengine2d.Context, l *Classic) {
			l.DrawFocusRing(c, paintengine2d.XYWH(5, 5, 110, 30))
		}},
		{"window background", func(c *paintengine2d.Context, l *Classic) {
			l.DrawWindowBackground(c, paintengine2d.XYWH(0, 0, 120, 40))
		}},
		{"tool button", func(c *paintengine2d.Context, l *Classic) {
			l.DrawToolButton(c, paintengine2d.XYWH(5, 5, 110, 30), StateHovered, "Cut", IconNone)
		}},
		{"list row", func(c *paintengine2d.Context, l *Classic) {
			l.DrawListRow(c, paintengine2d.XYWH(5, 5, 110, 30), StateChecked, "Row")
		}},
	}
	render := func(l *Classic, draw func(*paintengine2d.Context, *Classic)) *paintengine2d.Image {
		img := paintengine2d.NewImage(120, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		draw(ctx, l)
		return img
	}
	for _, d := range draws {
		if n := diffPixels(render(skin, d.draw), render(plain, d.draw)); n != 0 {
			t.Errorf("%s: a skin that describes nothing differs from its base pack in %d pixels", d.name, n)
		}
	}
	// And its frame is the base pack's too.
	if got, want := DecorationOf(skin, DecorationState{Active: true}), DecorationOf(plain, DecorationState{Active: true}); got != want {
		t.Errorf("frame %+v, wanted the base pack's %+v", got, want)
	}
}

func diffPixels(a, b *paintengine2d.Image) int {
	n := 0
	for i := 0; i+3 < len(a.Pix) && i+3 < len(b.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
			n++
		}
	}
	return n
}

// A skin may decorate focus; it may never remove it. DrawFocusRing always
// paints, whatever the skin says.
func TestSkinAlwaysDrawsAFocusRing(t *testing.T) {
	for _, name := range []string{"nocturne", "cassette"} {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Skip(name + " is not registered")
		}
		lk := pack.Look()
		img := paintengine2d.NewImage(60, 30)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		lk.DrawFocusRing(ctx, paintengine2d.XYWH(5, 5, 50, 20))
		ink := 0
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] > 40 {
				ink++
			}
		}
		if ink < 20 {
			t.Errorf("%s: a focus ring of %d pixels is not a focus ring", name, ink)
		}
	}
}

// ---- user skins, live reload and archives ---------------------------------

// installSkin writes a skin under a temporary XDG config dir and returns the
// manifest's path.
func installSkin(t *testing.T, name, doc string) string {
	t.Helper()
	dir := filepath.Join(SkinsDir(), name)
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "art", "sheet.png"), skinTestPNG(t, 1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "art", "sheet2.png"), skinTestPNG(t, 2), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, SkinFile)
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()
	return path
}

// A skin installed under the config dir is a pack: it lists, it loads, and
// the look watcher is pointed at its manifest.
func TestUserSkinIsAPack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	installSkin(t, "probe", skinTestDoc)

	pack, ok := LoadTheme("probe")
	if !ok {
		t.Fatal("an installed skin does not load as a pack")
	}
	if pack.Source != ThemeSourceUser || pack.Tokens.Engine != skinEngineID {
		t.Fatalf("pack %+v", pack)
	}
	found := false
	for _, p := range ListThemes() {
		if p.Name == "probe" {
			found = true
		}
	}
	if !found {
		t.Error("an installed skin does not appear in ListThemes")
	}
	if !IsSkin("probe") {
		t.Error("IsSkin says no")
	}
	// The watcher stamps the manifest, which is what makes live reload work
	// through the machinery that already reloads an edited theme.json.
	if got := ThemeSourceFile("probe"); filepath.Base(got) != SkinFile {
		t.Errorf("ThemeSourceFile(probe) = %q, wanted the manifest", got)
	}
	// And the look it builds paints with the skin engine.
	if id := pack.Look().Engine().ID(); id != skinEngineID {
		t.Errorf("look paints with %q", id)
	}
}

// Editing an installed skin applies without a restart, through the same
// generation counter the icon cache uses.
func TestUserSkinReloadsWhenEdited(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := installSkin(t, "probe", skinTestDoc)

	sk, ok := LoadSkin("probe")
	if !ok {
		t.Fatal("not installed")
	}
	if sk.Label != "Probe" {
		t.Fatalf("label %q", sk.Label)
	}

	edited := strings.Replace(skinTestDoc, `"label": "Probe"`, `"label": "Edited"`, 1)
	edited = strings.Replace(edited, `"caption": 24`, `"caption": 40`, 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache() // what the look watcher's reload does

	sk2, ok := LoadSkin("probe")
	if !ok {
		t.Fatal("the edited skin does not load")
	}
	if sk2.Label != "Edited" || sk2.Window.Caption != 40 {
		t.Fatalf("the edit did not apply: label %q caption %g", sk2.Label, sk2.Window.Caption)
	}
	if p, _ := LoadTheme("probe"); p.Label != "Edited" {
		t.Errorf("LoadTheme still answers %q", p.Label)
	}

	// A manifest edited into nonsense drops the skin rather than taking the
	// theme browser down with it.
	if err := os.WriteFile(path, []byte(`{"skin": 1, "parts": {"nope": {}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()
	if _, ok := LoadSkin("probe"); ok {
		t.Error("a broken skin still loads")
	}
	for _, p := range ListThemes() {
		if p.Name == "probe" {
			t.Error("a broken skin still lists")
		}
	}
}

// A skin shared as one file is a zip of the directory, which is what every
// skin format of the era turned out to be.
func TestSkinLoadsFromAnArchive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(SkinsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, nested := range []bool{false, true} {
		name := "flat"
		prefix := ""
		if nested {
			name, prefix = "nested", "nested/"
		}
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		add := func(p string, data []byte) {
			w, err := zw.Create(prefix + p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		add(SkinFile, []byte(skinTestDoc))
		add("art/sheet.png", skinTestPNG(t, 1))
		add("art/sheet2.png", skinTestPNG(t, 2))
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(SkinsDir(), name+SkinArchiveExt), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	InvalidateSkinCache()

	for _, name := range []string{"flat", "nested"} {
		sk, ok := LoadSkin(name)
		if !ok {
			t.Fatalf("%s%s does not load", name, SkinArchiveExt)
		}
		// The archive is closed after loading, so its art has to have been
		// taken into memory or nothing will paint.
		InvalidateSkinCache()
		sk, _ = LoadSkin(name)
		img := paintengine2d.NewImage(40, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		if !sk.skinDraw(ctx, paintengine2d.XYWH(0, 0, 40, 40), sk.Sprites["face"], 1, paintengine2d.Color{}) {
			t.Errorf("%s: nothing painted from the archive", name)
		}
	}
}

// ---- the window silhouette ------------------------------------------------

// The shape a skin declares resolves against a window box. Nothing consumes
// it yet; it is tested so that the day something does, the arithmetic is
// already right.
func TestSkinWindowRectsResolve(t *testing.T) {
	doc := `{"skin": 1,
		"sheets": {"chrome": {"1x": "art/sheet.png"}},
		"window": {"shape": [
			{"at": [0, 0, 4, 4], "stretchX": true, "stretchY": true},
			{"at": [10, 0, 40, 20], "radius": [4, 4, 0, 0]}]}}`
	sk := loadTestSkin(t, doc)
	win := paintengine2d.XYWH(100, 50, 400, 300)
	got := SkinWindowRects(sk, win, 2)
	if len(got) != 2 {
		t.Fatalf("%d rects, wanted 2", len(got))
	}
	// Stretching: the third and fourth numbers are margins from the far edge.
	want0 := paintengine2d.XYWH(100, 50, 400-8, 300-8)
	if got[0] != want0 {
		t.Errorf("stretched rect %v, wanted %v", got[0], want0)
	}
	want1 := paintengine2d.XYWH(120, 50, 80, 40)
	if got[1] != want1 {
		t.Errorf("fixed rect %v, wanted %v", got[1], want1)
	}
	if SkinWindowRects(loadTestSkin(t, skinTestDoc), win, 1) != nil {
		t.Error("a skin with no shape should ask for none")
	}
}

// Skins never opt a look into a shape it cannot have: every registered pack,
// skins included, states a rectangular frame today.
func TestSkinFrameIsWholePixelsAtEveryScale(t *testing.T) {
	for _, name := range []string{"nocturne", "cassette"} {
		pack, ok := LoadTheme(name)
		if !ok {
			t.Skip(name + " is not registered")
		}
		for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
			lk := pack.Look().setScale(scale)
			s := DecorationOf(lk, DecorationState{Active: true})
			for _, v := range []float32{s.Border.Top, s.Border.Right, s.Border.Bottom, s.Border.Left} {
				if v != float32(int(v)) {
					t.Errorf("%s @%gx: border %v is not whole device pixels", name, scale, s.Border)
				}
			}
			if s.Caption < 8 {
				t.Errorf("%s @%gx: caption %g is too short to hold a button", name, scale, s.Caption)
			}
		}
	}
}

// ---- the vocabulary -------------------------------------------------------

// The docs and the lint command are generated from these tables, so they
// have to agree with what the engine actually reads.
func TestSkinVocabularyIsConsistent(t *testing.T) {
	for _, r := range []Role{RoleButton, RoleTool, RoleField, RoleCheck, RoleRow, RoleTab,
		RoleThumb, RoleTrack, RoleMenu, RoleCombo, RoleSplitter, RoleBar, RolePanel} {
		name, ok := skinRolePart[r]
		if !ok {
			t.Errorf("role %d has no part name", r)
			continue
		}
		if SkinPartDoc(name) == "" {
			t.Errorf("part %q is bound to a role but not documented", name)
		}
	}
	for _, s := range SkinStateNames() {
		if _, ok := skinStateFallback[s]; !ok {
			t.Errorf("state %q has no fallback chain", s)
		}
		for _, alt := range SkinStateFallback(s) {
			if !skinStateSet[alt] {
				t.Errorf("state %q falls back to %q, which is not a state", s, alt)
			}
		}
	}
	if len(SkinPartNames()) != len(skinPartNames) {
		t.Error("SkinPartNames does not list every part")
	}
}

// A manifest round-trips: what the generator writes, the loader reads.
func TestShippedManifestsAreValidJSON(t *testing.T) {
	for _, name := range []string{"nocturne", "cassette"} {
		raw, err := os.ReadFile(filepath.Join("skins", name, SkinFile))
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v, _ := doc["skin"].(float64); int(v) != SkinVersion {
			t.Errorf("%s declares version %v, this build reads %d", name, doc["skin"], SkinVersion)
		}
	}
}
