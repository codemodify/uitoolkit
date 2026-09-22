package style

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A pixel skin at 1.25, 1.5 and 1.75, and the seam a stretched nine-slice
// used to have.

// checkerSkin installs a pixelated skin whose sheet is a 1-texel
// checkerboard of two colours, 24×8 at 1× (and exactly doubled at 2×), with
// three sprites on it: two 8×8 squares side by side and the 16×8 they make
// together. Drawn at their own size, the two squares must paint exactly
// what the 16×8 paints — a panel's pieces meeting as if they were one
// picture.
func checkerSkin(t *testing.T) LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sheet := func(n int) []byte {
		img := paintengine2d.NewImage(24*n, 8*n)
		for y := 0; y < 8*n; y++ {
			for x := 0; x < 24*n; x++ {
				c := paintengine2d.RGB(0.1, 0.1, 0.1)
				if (x/n+y/n)%2 == 0 {
					c = paintengine2d.RGB(0.9, 0.8, 0.2)
				}
				img.SetColor(x, y, c)
			}
		}
		var buf bytes.Buffer
		if err := img.WritePNG(&buf); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	dir := filepath.Join(SkinsDir(), "checker")
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, n := range map[string]int{"sheet.png": 1, "sheet2.png": 2} {
		if err := os.WriteFile(filepath.Join(dir, "art", name), sheet(n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc := `{"skin": 1, "base": "win95", "design": {"pixelated": true},
		"sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/sheet2.png"}},
		"sprites": {
			"left":  {"at": [0, 0, 8, 8]},
			"right": {"at": [8, 0, 8, 8]},
			"both":  {"at": [0, 0, 16, 8]},
			"framed": {"at": [0, 0, 8, 8], "slice": [1, 1, 1, 1]}
		}}`
	if err := os.WriteFile(filepath.Join(dir, SkinFile), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()
	p, ok := LoadTheme("checker")
	if !ok {
		t.Fatal("the checker skin does not load")
	}
	return p.Look()
}

func TestAPixelPanelsPiecesShareOneGrid(t *testing.T) {
	base := checkerSkin(t)
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := WithScale(base, scale)
		d := func(v float32) float32 { return Dip(lk, v) }
		// Under a translation that is not whole pixels, as a component at a
		// design coordinate times 1.75 is.
		for _, off := range []float32{0, 0.25, 0.5, 0.75} {
			paint := func(draw func(ctx *paintengine2d.Context)) *paintengine2d.Image {
				img := paintengine2d.NewImage(48, 24)
				ctx := paintengine2d.NewContext(img)
				ctx.Clear(paintengine2d.Color{})
				ctx.Translate(2+off, 3+off)
				draw(ctx)
				return img
			}
			pieces := paint(func(ctx *paintengine2d.Context) {
				DrawSkinSprite(lk, ctx, paintengine2d.XYWH(0, 0, d(8), d(8)), "left", paintengine2d.Color{})
				DrawSkinSprite(lk, ctx, paintengine2d.XYWH(d(8), 0, d(8), d(8)), "right", paintengine2d.Color{})
			})
			whole := paint(func(ctx *paintengine2d.Context) {
				DrawSkinSprite(lk, ctx, paintengine2d.XYWH(0, 0, d(16), d(8)), "both", paintengine2d.Color{})
			})
			if n := diffPixels(pieces, whole); n != 0 {
				t.Errorf("%gx +%g: two pieces differ from the picture they make in %d pixels", scale, off, n)
			}
			// Every pixel is one of the two colours or empty — nothing
			// blended — and a texel is never three pixels or none.
			row := 3 + int(d(4))
			run, last, longest := 0, -1, 0
			for x := 0; x < whole.Width; x++ {
				r, g, b, a := whole.PremulAt(x, row)
				if a != 0 && a != 255 {
					t.Fatalf("%gx +%g: pixel %d is %d%% covered: the edge is not whole pixels", scale, off, x, int(a)*100/255)
				}
				if a == 255 && !(r < 40 && g < 40 && b < 40) && !(r > 200 && g > 180 && b < 80) {
					t.Fatalf("%gx +%g: pixel %d is %d,%d,%d, neither colour of the art", scale, off, x, r, g, b)
				}
				c := int(r)
				if a == 0 {
					c = -1
				}
				if c == last {
					run++
				} else {
					if last >= 0 {
						longest = max(longest, run)
					}
					run, last = 1, c
				}
			}
			ceil := int(scale)
			if float32(ceil) < scale {
				ceil++
			}
			if longest > ceil {
				t.Errorf("%gx +%g: a texel is %d pixels wide, more than ⌈%g⌉", scale, off, longest, scale)
			}
		}
	}
}

// A control face on a pixel sheet keeps the whole-multiple rule it was drawn
// for: a sliced face in a box its own size at 1.75 is corners at 1× and a
// stretched middle, not the panel's magnified picture.
func TestAPixelFaceKeepsWholeMultiples(t *testing.T) {
	lk := WithScale(checkerSkin(t), 1.75)
	l, sk := lookSkin(lk)
	b := paintengine2d.XYWH(0, 0, 14, 14)
	panel := paintengine2d.NewImage(14, 14)
	sk.panelDraw(paintengine2d.NewContext(panel), b, sk.Sprites["framed"], l.Scale(), paintengine2d.Color{})
	face := paintengine2d.NewImage(14, 14)
	sk.skinDraw(paintengine2d.NewContext(face), b, sk.Sprites["framed"], l.Scale(), paintengine2d.Color{})
	if diffPixels(panel, face) == 0 {
		t.Fatal("a face and a panel piece paint the same at 1.75; the face lost its whole multiples")
	}
}

// A nine-slice stretched wide and tall is one picture: every pixel of it is
// opaque and one of the art's own colours, with no pale line where the
// stretched middle meets the fixed edges. The skins used to guarantee this
// with a replicated border round every piece; the renderer's sampler clamps
// at an image's edge now, and this is what would catch it if it stopped.
func TestSkinNineSliceHasNoSeam(t *testing.T) {
	sk := loadTestSkin(t, skinTestDoc)
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		InvalidateSkinCache()
		img := paintengine2d.NewImage(200, 90)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		b := paintengine2d.XYWH(10, 10, 170, 60)
		if !sk.skinDraw(ctx, b, sk.Sprites["face"], scale, paintengine2d.Color{}) {
			t.Fatal("nothing painted")
		}
		for y := 11; y < 69; y++ {
			for x := 11; x < 179; x++ {
				r, g, bl, a := img.PremulAt(x, y)
				if a != 255 {
					t.Fatalf("%gx: pixel (%d, %d) has alpha %d: a seam", scale, x, y, a)
				}
				// Red corners, green edges, blue middle — and where two
				// meet under a bilinear downscale, a mix of two of them,
				// never anything paler than the art.
				if int(r)+int(g)+int(bl) < 200 {
					t.Fatalf("%gx: pixel (%d, %d) is %d,%d,%d: darker than any of the art's colours", scale, x, y, r, g, bl)
				}
			}
		}
	}
}
