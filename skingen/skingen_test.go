package skingen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// shippedDir is where uitk-skingen writes, relative to this package.
const shippedDir = "../style/skins"

// The art in the repo is what this package draws. Regenerating into a
// temporary directory and comparing byte for byte is what makes "generated,
// not painted" a fact rather than a claim: a change to the drawing that was
// not committed fails here, and so does a hand-edited PNG.
//
// If this fails after a deliberate change, run:
//
//	go run ./cmd/uitk-skingen
func TestSkinArtIsReproducible(t *testing.T) {
	tmp := t.TempDir()
	for _, p := range Plans() {
		if err := Write(tmp, p); err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
	}
	for _, p := range Plans() {
		files := []string{SkinManifestName}
		for _, sh := range p.Sheets {
			for _, s := range Scales {
				files = append(files, filepath.ToSlash(sheetFile(sh.Name, s)))
			}
		}
		for _, f := range files {
			want, err := os.ReadFile(filepath.Join(shippedDir, p.Name, f))
			if err != nil {
				t.Errorf("%s/%s is not committed: %v", p.Name, f, err)
				continue
			}
			got, err := os.ReadFile(filepath.Join(tmp, p.Name, f))
			if err != nil {
				t.Errorf("%s/%s was not generated: %v", p.Name, f, err)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s/%s differs from the generator's output (%d bytes committed, %d generated); run go run ./cmd/uitk-skingen",
					p.Name, f, len(want), len(got))
			}
		}
	}
}

// Nothing a cell draws may land in a neighbouring cell: the sheet is a grid
// of independent sprites, and a stray pixel in the gutter is a stray pixel
// in somebody's button.
func TestSkinArtStaysInsideItsCells(t *testing.T) {
	for _, p := range Plans() {
		for _, sh := range p.Sheets {
			img := Render(sh, 1)
			inside := make([]bool, img.Width*img.Height)
			for _, c := range sh.Cells {
				for y := c.Y; y < c.Y+c.H; y++ {
					for x := c.X; x < c.X+c.W; x++ {
						if x >= 0 && y >= 0 && x < img.Width && y < img.Height {
							inside[y*img.Width+x] = true
						}
					}
				}
			}
			for y := 0; y < img.Height; y++ {
				for x := 0; x < img.Width; x++ {
					if inside[y*img.Width+x] {
						continue
					}
					if _, _, _, a := img.PremulAt(x, y); a > 0 {
						t.Fatalf("%s/%s: ink at (%d,%d) is outside every cell", p.Name, sh.Name, x, y)
					}
				}
			}
		}
	}
}

// A pixelated sheet's 2× asset is its 1× asset with every pixel doubled —
// the same picture with square pixels, not a re-rasterised approximation of
// it. That is the only honest way to publish a 2× pixel-art sheet, and it is
// what lets the engine draw it nearest-neighbour at a whole multiple.
func TestPixelArtDoublesExactly(t *testing.T) {
	for _, p := range Plans() {
		for _, sh := range p.Sheets {
			if !sh.Pixelated {
				continue
			}
			one, two := Render(sh, 1), Render(sh, 2)
			if two.Width != one.Width*2 || two.Height != one.Height*2 {
				t.Fatalf("%s/%s: 2x is %dx%d, wanted %dx%d",
					p.Name, sh.Name, two.Width, two.Height, one.Width*2, one.Height*2)
			}
			for y := 0; y < one.Height; y++ {
				for x := 0; x < one.Width; x++ {
					r, g, b, a := one.PremulAt(x, y)
					for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
						r2, g2, b2, a2 := two.PremulAt(x*2+d[0], y*2+d[1])
						if r != r2 || g != g2 || b != b2 || a != a2 {
							t.Fatalf("%s/%s: (%d,%d) doubled to %d,%d,%d,%d, wanted %d,%d,%d,%d",
								p.Name, sh.Name, x, y, r2, g2, b2, a2, r, g, b, a)
						}
					}
				}
			}
		}
	}
}

// Every sprite the manifest binds is drawn: an empty cell that a part points
// at is a control that vanishes.
func TestEveryBoundSpriteHasInk(t *testing.T) {
	// A few are deliberately empty. A tool button is flat until it is
	// touched, so its resting and disabled states draw nothing at all, and
	// Nocturne's caption buttons are flat on the band for the same reason —
	// the glyph over them is what carries the meaning.
	blank := map[string]bool{
		"tool.normal": true, "tool.disabled": true, "capbtn.normal": true,
	}
	for _, p := range Plans() {
		bound := map[string]bool{}
		for _, b := range p.Parts {
			for _, kv := range b.States {
				bound[kv[1]] = true
			}
		}
		for _, sh := range p.Sheets {
			img := Render(sh, 1)
			for _, c := range sh.Cells {
				if !bound[c.Name] || blank[c.Name] {
					continue
				}
				ink := 0
				for y := c.Y; y < c.Y+c.H; y++ {
					for x := c.X; x < c.X+c.W; x++ {
						if _, _, _, a := img.PremulAt(x, y); a > 8 {
							ink++
						}
					}
				}
				if ink == 0 {
					t.Errorf("%s: sprite %q is bound but draws nothing", p.Name, c.Name)
				}
			}
		}
	}
}

// Every part a plan binds is a part of the toolkit, and every sprite it
// names is a cell that exists. The loader checks this too, but failing here
// names the Go line that is wrong instead of the JSON it produced.
func TestPlansBindRealSpritesAndParts(t *testing.T) {
	for _, p := range Plans() {
		cells := map[string]bool{}
		for _, sh := range p.Sheets {
			for _, c := range sh.Cells {
				if cells[c.Name] {
					t.Errorf("%s: two cells named %q", p.Name, c.Name)
				}
				cells[c.Name] = true
			}
		}
		roles := map[string]bool{}
		for _, r := range p.Text {
			roles[r.Name] = true
		}
		for _, b := range p.Parts {
			if b.Text != "" && !roles[b.Text] {
				t.Errorf("%s: part %q uses text role %q, which is not declared", p.Name, b.Part, b.Text)
			}
			seenNormal := false
			for _, kv := range b.States {
				if kv[0] == "normal" {
					seenNormal = true
				}
				if !cells[kv[1]] {
					t.Errorf("%s: part %q state %q names sprite %q, which no cell draws",
						p.Name, b.Part, kv[0], kv[1])
				}
			}
			if !seenNormal {
				t.Errorf("%s: part %q has no normal state", p.Name, b.Part)
			}
		}
	}
}
