//go:build font_emoji

package style

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// emojiFontFile is a colour emoji face on this machine, or "".
func emojiFontFile(t *testing.T) string {
	t.Helper()
	for _, dir := range []string{"/usr/share/fonts", "/usr/local/share/fonts", "/Library/Fonts"} {
		var found string
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || found != "" {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.HasSuffix(name, ".ttf") && !strings.HasSuffix(name, ".ttc") {
				return nil
			}
			if strings.Contains(name, "emoji") && strings.Contains(name, "color") {
				found = p
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

// drawnPixels is every visible pixel of one string, in order.
func drawnPixels(t *testing.T, f *Font, s string, tint paintengine2d.Color) []uint32 {
	t.Helper()
	img := paintengine2d.NewImage(40, 40)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, s, paintengine2d.Pt(4, 4), tint)
	img.Touch()
	var out []uint32
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a == 0 {
				continue
			}
			out = append(out, uint32(r)<<24|uint32(g)<<16|uint32(b)<<8|uint32(a))
		}
	}
	return out
}

// An emoji is a picture, and a picture is not tinted.
//
// Every other glyph in this toolkit is a coverage mask: the sheet says
// how much of each pixel is covered and the draw says what colour to
// make it, which is how one baked face serves every colour of text. A
// colour emoji is the other thing — 👀 is a PNG in Noto Color Emoji and
// an outline in no face at all — so its sheet carries the colours and
// the draw must leave them alone. Multiplying it by the text colour
// would make a red label's emoji red.
func TestAnEmojiKeepsItsOwnColours(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	box := drawnPixels(t, f, "", paintengine2d.White)
	eyes := drawnPixels(t, f, "👀", paintengine2d.White)
	if len(eyes) == 0 {
		t.Fatal("the emoji drew nothing at all")
	}
	if len(eyes) == len(box) {
		t.Fatalf("the emoji drew %d pixels, the same as the box; it was not borrowed", len(eyes))
	}

	// In colour: a mask tinted white is grey, and this is not.
	coloured := false
	for _, p := range eyes {
		r, g, b := byte(p>>24), byte(p>>16), byte(p>>8)
		if r != g || g != b {
			coloured = true
			break
		}
	}
	if !coloured {
		t.Error("every pixel of the emoji is grey; it was drawn as a coverage mask")
	}

	// And the same whatever colour the text is.
	red := drawnPixels(t, f, "👀", paintengine2d.Color{R: 1, A: 1})
	if len(red) != len(eyes) {
		t.Fatalf("red text drew %d pixels of emoji, white text %d", len(red), len(eyes))
	}
	for i := range red {
		if red[i] != eyes[i] {
			t.Fatalf("the emoji changed colour with the text: pixel %d is %08x in red text and %08x in white",
				i, red[i], eyes[i])
		}
	}
}

// The letters beside it are still tinted, or the fix went too far.
func TestTextBesideAnEmojiIsStillTinted(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	white := drawnPixels(t, f, "A", paintengine2d.White)
	red := drawnPixels(t, f, "A", paintengine2d.Color{R: 1, A: 1})
	if len(white) == 0 || len(red) == 0 {
		t.Fatal("the letter drew nothing")
	}
	same := len(white) == len(red)
	if same {
		for i := range white {
			if white[i] != red[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("a letter drew the same pixels in red as in white; the tint is not being applied")
	}
}
