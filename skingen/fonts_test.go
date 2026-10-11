package skingen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/skingen"
	"github.com/codemodify/uitoolkit/style"
)

// **A generated skin can ask for the typeface its art was drawn around.**
// A skin is a picture of an era and the era had a face; the manifest has
// had a "fonts" block since skins existed and Plan had no field for it, so
// an application that generated its packs had to open the manifest the
// generator had just written and put the block in itself.
func TestAGeneratedSkinAsksForItsTypefaces(t *testing.T) {
	dir := t.TempDir()
	p := &skingen.Plan{
		Name:  "facedskin",
		Label: "Faced",
		Base:  "breeze-night",
		Sheets: []*skingen.Sheet{{
			Name: "chrome", W: 32, H: 32,
			Cells: []skingen.Cell{{Name: "face", X: 0, Y: 0, W: 32, H: 32,
				Draw: func(ctx *paintengine2d.Context, w, h float32) {
					ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(skingen.Hex("#202428")))
				}}},
		}},
		Parts: []skingen.PartBinding{{
			Part: "button", States: [][2]string{{"normal", "face"}},
		}},
		Fonts: &style.FontPrefs{
			UI:   []string{"Nimbus Sans", "Liberation Sans"},
			Mono: []string{"Nimbus Mono PS"},
		},
	}
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
	if err != nil {
		t.Fatalf("a plan with typefaces does not load: %v", err)
	}
	got := sk.Tokens.Fonts
	if got.Empty() {
		t.Fatal("the typefaces did not reach the skin")
	}
	if len(got.UI) != 2 || got.UI[0] != "Nimbus Sans" || got.UI[1] != "Liberation Sans" {
		t.Errorf("the interface faces are %v", got.UI)
	}
	if len(got.Mono) != 1 || got.Mono[0] != "Nimbus Mono PS" {
		t.Errorf("the monospaced faces are %v", got.Mono)
	}
}

// A plan that names no typeface writes no block, so every skin generated
// before this is byte for byte what it was.
func TestAPlanWithNoTypefacesWritesNoBlock(t *testing.T) {
	dir := t.TempDir()
	p := responsivePlan()
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	doc, err := os.ReadFile(filepath.Join(dir, p.Name, style.SkinFile))
	if err != nil {
		t.Fatal(err)
	}
	if want := `"fonts"`; contains(string(doc), want) {
		t.Errorf("the manifest has a %s block for a plan that named none:\n%s", want, doc)
	}
	// And an empty one is the same as none.
	p.Fonts = &style.FontPrefs{}
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	doc, _ = os.ReadFile(filepath.Join(dir, p.Name, style.SkinFile))
	if contains(string(doc), `"fonts"`) {
		t.Errorf("an empty FontPrefs wrote a block:\n%s", doc)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
