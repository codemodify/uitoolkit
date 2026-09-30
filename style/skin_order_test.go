package style

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// A strip generates sprite names — "button.normal" and the rest — that
// another part may refer to by name. Parts were read whole, one at a
// time, in Go's map order, so whether the reference resolved depended on
// which part happened to be read first: the same manifest loaded a
// thousand times failed 135 of them with `no sprite "button.normal"`.
//
// Loading one manifest many times has to give the same answer every
// time, and the answer has to be "it loaded".
func TestSkinStripReferencesDoNotDependOnMapOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := filepath.Join(UserSkinsDir(), "striporder")
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 64, 16))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(0, 0, color.NRGBA{1, 2, 3, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "art", "sheet.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	// button defines a strip; tool names one of the sprites it generates.
	doc := `{"skin": 1, "base": "light",
		"sheets": {"chrome": {"1x": "art/sheet.png"}},
		"parts": {
			"button": {"strip": {"sheet": "chrome", "at": [0, 0, 16, 16],
				"states": ["normal", "hover", "pressed", "disabled"]}},
			"tool":   {"states": {"normal": "button.normal"}}
		}}`
	if err := os.WriteFile(filepath.Join(dir, SkinFile), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 300; i++ {
		InvalidateSkinCache()
		sk, ok := LoadSkin("striporder")
		if !ok {
			t.Fatalf("load %d of 300 failed: the same manifest has to "+
				"load the same way every time, and tool's reference to "+
				"button.normal resolves only if button was read first", i+1)
		}
		if sk.Parts["tool"] == nil || sk.Parts["tool"].States["normal"] == nil {
			t.Fatalf("load %d: tool.normal did not resolve to the generated sprite", i+1)
		}
	}
}

// A skin that declares no family inherits its base pack's, so a skin
// over a light base is light. ParseTheme's default branch answers dark,
// so reading the field through it turned "omitted" into "declared dark"
// and beat the inheritance: the palette came out light and the metadata
// dark, which disagree about what the skin is.
func TestSkinWithNoFamilyInheritsItsBase(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, ok := LoadTheme("light"); !ok {
		t.Skip("no light pack in this build")
	}
	dir := filepath.Join(UserSkinsDir(), "nofamily")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SkinFile),
		[]byte(`{"skin": 1, "base": "light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()

	sk, ok := LoadSkin("nofamily")
	if !ok {
		t.Fatal("the skin did not load")
	}
	if sk.Family != "" {
		t.Errorf("an omitted family was read as %q; it has to stay unset so the base can supply it", sk.Family)
	}
	pack := sk.Pack()
	if pack.Palette != ThemeLight {
		t.Errorf("a skin over a light base came out %q", pack.Palette)
	}
}
