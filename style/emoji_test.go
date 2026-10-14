//go:build font_emoji

package style

import (
	"io/fs"
	"os"
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

// inkBox is the rectangle the ink of s occupies, drawn at origin.
func inkBox(t *testing.T, f *Font, s string, origin paintengine2d.Point) (minY, maxY int, n int) {
	t.Helper()
	const w, h = 64, 64
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	f.Draw(ctx, s, origin, paintengine2d.White)
	img.Touch()
	minY, maxY = h, -1
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				n++
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	return minY, maxY, n
}

// An emoji sits on the line, like the letters beside it.
//
// Every cell in this toolkit measures its bearing from the *top* of the
// line — an outline's is `ascent + ymin` — and the colour cell measured
// its own from the baseline, which put each picture a whole ascent
// above the text and let the window clip it away. A mail client found
// it the hour the tag shipped.
func TestAnEmojiSitsOnItsLine(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	const top = 12
	at := paintengine2d.Pt(4, top)
	lo, hi, n := inkBox(t, f, "👀", at)
	if n == 0 {
		t.Fatal("the emoji drew nothing")
	}
	if lo < top {
		t.Errorf("the emoji starts at y=%d, above the line's top at %d", lo, top)
	}
	if bottom := top + int(f.Height()+0.5); hi > bottom {
		t.Errorf("the emoji ends at y=%d, below the line's bottom at %d", hi, bottom)
	}
	// And in the same band as the letters beside it.
	tlo, thi, tn := inkBox(t, f, "x", at)
	if tn == 0 {
		t.Fatal("the letter drew nothing")
	}
	if lo > thi || hi < tlo {
		t.Errorf("the emoji is at y=%d..%d and the letter at y=%d..%d: they do not share a line", lo, hi, tlo, thi)
	}
}

// A character to be drawn as emoji goes to the colour face even when an
// outline face has a black-and-white one.
//
// ❤ and ❤️ are the same character; the second is followed by U+FE0F,
// the selector that asks for the emoji form. Taking the outline face
// first drew both of them flat beside colour emoji, and split 👍🏽
// between two faces — the base from one and the tone from the other,
// which no shaper can then join. A mail client and a music player
// reported it separately, the same afternoon.
func TestEmojiPresentationBeatsAnOutlineFace(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	outline := outlineHeartFile(t)
	if outline == "" {
		t.Skip("no outline face with a heart in it to compete")
	}
	fakeFonts(t,
		"Noto Color Emoji\t80\t0\t0\t"+path,
		"Outline Hearts\t80\t0\t0\t"+outline)
	// The outline face is the *only* family in the fallback list, which
	// is the arrangement that was getting it wrong.
	SetFallbackFamilies("Outline Hearts")
	t.Cleanup(func() { SetFallbackFamilies() })
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	tinted := func(what string) bool {
		w := drawnPixels(t, f, what, paintengine2d.White)
		r := drawnPixels(t, f, what, paintengine2d.Color{R: 1, A: 1})
		if len(w) == 0 {
			t.Fatalf("%q drew nothing", what)
		}
		if len(w) != len(r) {
			return true
		}
		for i := range w {
			if w[i] != r[i] {
				return true
			}
		}
		return false
	}

	// Asked for as text: the outline face, which the label's colour
	// tints, exactly as before.
	if !tinted("❤") {
		t.Error("❤ was drawn in colour; without the selector it is text, and text takes the label's colour")
	}
	// Asked for as emoji: the picture, whatever colour the label is.
	if tinted("❤\ufe0f") {
		t.Error("❤️ took the text colour: the outline face won over the colour one")
	}

	// And a hand and its tone are one picture, not two glyphs from two
	// faces.
	hand := f.Advance("👍")
	toned := f.Advance("👍🏽")
	if hand <= 0 {
		t.Fatal("the hand measured nothing")
	}
	if toned > hand*1.5 {
		t.Errorf("👍🏽 is %.2f wide and 👍 is %.2f: the tone was drawn as a glyph of its own", toned, hand)
	}
}

// outlineHeartFile is an installed face with U+2764 drawn as an
// outline, or "".
func outlineHeartFile(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		"/usr/share/fonts/Adwaita/AdwaitaMono-Regular.ttf",
		"/usr/share/fonts/gsfonts/NimbusSans-Regular.otf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
	} {
		if _, err := os.Stat(p); err == nil {
			if f, err := parseFaceBytes(p, mustRead(t, p), 0); err == nil && f.hasGlyph('\u2764') {
				return p
			}
		}
	}
	return ""
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("cannot read %s: %v", p, err)
	}
	return b
}

// Baking more emoji does not turn the ones already drawn back into
// masks.
//
// The colour sheets were kept in a set beside them, and publishing a
// bigger sheet deleted the old one from it — tidy, and wrong: a run
// shaped earlier still names the older sheet, so the first emoji
// started taking the label's colour as soon as a second one was baked.
// A music player caught it by drawing 👀, baking 😀, and drawing 👀
// again.
func TestGrowingTheEmojiSheetKeepsTheColoursOfWhatIsDrawn(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 31, paintengine2d.White)

	untinted := func(what string) bool {
		w := drawnPixels(t, f, what, paintengine2d.White)
		r := drawnPixels(t, f, what, paintengine2d.Color{R: 1, A: 1})
		if len(w) == 0 || len(w) != len(r) {
			return false
		}
		for i := range w {
			if w[i] != r[i] {
				return false
			}
		}
		return true
	}

	if !untinted("👀") {
		t.Fatal("👀 took the text colour before anything else was baked")
	}
	// Bake more of the face, which publishes a bigger sheet.
	for _, more := range []string{"😀", "🎉", "🚀", "🥳", "🇯🇵"} {
		f.Advance(more)
		f.Draw(paintengine2d.NewContext(paintengine2d.NewImage(64, 64)), more,
			paintengine2d.Pt(4, 4), paintengine2d.White)
	}
	if !untinted("👀") {
		t.Error("👀 took the text colour once the sheet had grown under it")
	}
}
