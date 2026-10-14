//go:build font_emoji

package style

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	// *With an outline face installed that has some of these
	// characters.* Noto Sans Symbols carries U+20E3, the enclosing
	// keycap, and is named in the built-in fallback list — so on an
	// ordinary machine it is offered the keycap before the colour
	// face is, and takes it. The first version of this test
	// installed nothing but the emoji face, which is why it passed
	// while a keycap drew as a digit beside a box for everyone else.
	lines := []string{"Noto Color Emoji\t80\t0\t0\t" + path}
	if sym := scriptFontFile(t, "symbols"); sym != "" {
		lines = append(lines, "Noto Sans Symbols\t80\t0\t0\t"+sym)
	}
	fakeFonts(t, lines...)
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

// glyphsDrawn is how many glyphs a string actually puts on screen,
// across every sheet it borrows from.
func glyphsDrawn(f *Font, s string) int {
	n := 0
	for _, run := range f.shapeOf(s).runs {
		n += len(run.Glyphs)
	}
	return n
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

// A space is a space, whoever drew the emoji beside it.
//
// The segmenter's rule is that whitespace never forces a change of
// face, so it keeps whatever run it is in — which is right between two
// Latin faces and wrong beside a picture: Noto Color Emoji's space is
// 1.245 em, so the space after 👀 came out five times the one before
// it. A mail client measured it. Whitespace at the edge of a picture
// belongs to the words.
func TestASpaceBesideAnEmojiIsATextSpace(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	space := f.Advance(" ")
	if space <= 0 {
		t.Fatal("a space measured nothing")
	}
	after := f.Advance("👀 x") - f.Advance("👀x")
	if d := after - space; d > 0.01 || d < -0.01 {
		t.Errorf("the space after an emoji is %.3f wide; on its own it is %.3f", after, space)
	}
	// And the line is the same width whichever side the picture is on.
	if a, b := f.Advance("👀 x"), f.Advance("x 👀"); a != b {
		t.Errorf("%q is %.3f and %q is %.3f", "👀 x", a, "x 👀", b)
	}
}

// One selector does not speak for another occurrence.
//
// ❤️ and ❤ are the same character, the first with the selector that
// asks for its emoji form. A table keyed by the character made both of
// them pictures — and ignored U+FE0E, the selector that asks for the
// text form. A music player found both.
func TestAPresentationSelectorSpeaksOnlyForItsOwnCharacter(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	outline := outlineHeartFile(t)
	if outline == "" {
		t.Skip("no outline face with a heart in it")
	}
	fakeFonts(t,
		"Noto Color Emoji\t80\t0\t0\t"+path,
		"Outline Hearts\t80\t0\t0\t"+outline)
	SetFallbackFamilies("Outline Hearts")
	t.Cleanup(func() { SetFallbackFamilies() })
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	emoji, text, space := f.Advance("❤️"), f.Advance("❤"), f.Advance(" ")
	if emoji == text {
		t.Skip("this machine draws both forms the same width; the test cannot tell them apart")
	}
	// A picture, a space, and a dingbat.
	want := emoji + space + text
	if got := f.Advance("❤️ ❤"); got < want-0.02 || got > want+0.02 {
		t.Errorf("%q is %.3f wide; a picture, a space and a dingbat are %.3f", "❤️ ❤", got, want)
	}
	// And the other selector is obeyed, not ignored.
	if got := f.Advance("❤︎"); got != text {
		t.Errorf("❤ asked for as text is %.3f wide; the text form is %.3f", got, text)
	}
}

// Two sizes of one face, drawn at once.
//
// A parsed face is not read-only: shaping fills caches in it and
// asking for a bitmap sets the pixel size its strike is chosen by. One
// face serves every Font that borrowed from it, so two goroutines
// drawing at two sizes wrote over each other. Found with the race
// detector by a music player; run this one with -race.
func TestOneEmojiFaceServesTwoSizesAtOnce(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	fonts := []*Font{
		BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White),
		BakeFamily(FamilyUI, WeightRegular, 31, paintengine2d.White),
		BakeFamily(FamilyUI, WeightRegular, 48, paintengine2d.White),
	}
	texts := []string{"👀", "😀", "🎉", "👀 x", "🥳🎉"}

	var wg sync.WaitGroup
	for _, f := range fonts {
		for _, s := range texts {
			wg.Add(1)
			go func(f *Font, s string) {
				defer wg.Done()
				img := paintengine2d.NewImage(80, 80)
				ctx := paintengine2d.NewContext(img)
				for i := 0; i < 4; i++ {
					f.Draw(ctx, s, paintengine2d.Pt(4, 4), paintengine2d.White)
					_ = f.Advance(s)
				}
			}(f, s)
		}
	}
	wg.Wait()
}

// A sequence the font composes is one picture.
//
// A family, a profession, a flag, a keycap and a skin tone are each
// several characters that an emoji font's own GSUB turns into one
// glyph (UTS #51). The characters that do the joining — the zero-width
// joiner, the presentation selector, the enclosing keycap — are ones
// the font composes *away*, and asking whether the colour face "has" a
// glyph for them, then sending them to the text face when it has not,
// split every one of these into its parts: a family drew as three
// people, a keycap as a digit beside a box. Three applications
// reported it within half an hour of the release that did it.
func TestASequenceTheFontComposesIsOnePicture(t *testing.T) {
	path := emojiFontFile(t)
	if path == "" {
		t.Skip("no colour emoji face on this machine")
	}
	fakeFonts(t, "Noto Color Emoji\t80\t0\t0\t"+path)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)

	one := f.Advance("👩")
	if one <= 0 {
		t.Fatal("one emoji measured nothing")
	}
	for _, seq := range []struct {
		text, what string
	}{
		{"👨‍👩‍👧", "a family"},
		{"👨‍💻", "a profession"},
		{"🏳️‍🌈", "a flag"},
		{"\U0001F469\u200d\u2764\ufe0f\u200d\U0001F469", "a couple with a heart"},
		{"1️⃣", "a keycap"},
		{"#\ufe0f\u20e3", "a hash keycap"},
		{"👍🏽", "a skin tone"},
	} {
		// A quarter over, which is loose enough for another emoji
		// font's advances and tight enough to catch the sequence
		// drawn as a picture *plus a leftover box* — which is what a
		// keycap did while the enclosing character was treated as
		// text, and what a 1.5 threshold let through.
		// Counted, not only measured. An enclosing keycap has *no
		// advance* — it is a combining mark — so a stray one taken
		// from an outline face draws over the picture and changes the
		// width by nothing at all. Width alone said the keycap was
		// fine while it was being drawn twice, from two faces.
		if n := glyphsDrawn(f, seq.text); n != 1 {
			t.Errorf("%s drew %d glyphs; one picture is one glyph", seq.what, n)
		}
		got := f.Advance(seq.text)
		if got > one*1.25 {
			t.Errorf("%s is %.2f wide and one picture is %.2f: it was drawn as %.1f pictures",
				seq.what, got, one, got/one)
		}
	}
}

// The same sequences against the machine's own font set.
//
// The test above stubs the index with two families, which is
// reproducible and is not what anybody runs. On a real machine the
// built-in fallback list finds several outline faces, and one of them
// — Noto Sans Symbols — carries U+20E3: offered the enclosing keycap
// before the colour face is, it takes it, and `1️⃣` becomes a picture
// with somebody else's box drawn over it. That is what a mail client
// and two players saw while this file's stubbed test passed.
func TestSequencesComposeAgainstTheMachinesOwnFonts(t *testing.T) {
	realFonts(t)
	f := BakeFamily(FamilyUI, WeightRegular, 16, paintengine2d.White)
	if glyphsDrawn(f, "👩") != 1 {
		t.Skip("this machine has no colour emoji face")
	}
	for _, seq := range []struct {
		text, what string
	}{
		{"\U0001F468‍\U0001F469‍\U0001F467", "a family"},
		{"\U0001F469‍❤️‍\U0001F469", "a couple with a heart"},
		{"1️⃣", "a keycap"},
		{"#️⃣", "a hash keycap"},
		{"\U0001F3F3️‍\U0001F308", "a flag"},
	} {
		if n := glyphsDrawn(f, seq.text); n != 1 {
			t.Errorf("%s drew %d glyphs; one picture is one glyph", seq.what, n)
		}
	}
}

// Two sizes of one *outline* face, drawn at once.
//
// sfnt hands back the segments of a glyph as a view into the Buffer
// they were loaded through — valid only until the next call that uses
// it. Every one of the four places that load a glyph unlocked the face
// and *then* walked those segments into a path, so two goroutines
// drawing the same face at two sizes read each other's glyph. The race
// detector says so; a wrong letter is what it looks like when it does
// not. A music player found it with mixed text at two sizes, and
// distinguished it from the colour-parser race it had already
// reported.
func TestOneOutlineFaceServesTwoSizesAtOnce(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "0")
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	// Cold sheets and a wide vocabulary, because a glyph is loaded
	// *once*: a warm atlas never calls LoadGlyph again and the window
	// in which two goroutines can cross is gone. Each goroutine takes
	// its own size — a Font per size, one face under all of them —
	// and draws enough different characters to keep baking.
	const letters = "Handgloves AVAST quick brown fox 0123456789 ÀÉÎÕÜ àéîõü ‹›«»†‡"
	var wg sync.WaitGroup
	for size := 11; size < 11+12; size++ {
		wg.Add(1)
		go func(size float32) {
			defer wg.Done()
			f := BakeFamily(FamilyUI, WeightRegular, size, paintengine2d.White)
			img := paintengine2d.NewImage(400, 80)
			ctx := paintengine2d.NewContext(img)
			rs := []rune(letters)
			for i := 0; i+8 < len(rs); i += 3 {
				s := string(rs[i : i+8])
				f.Draw(ctx, s, paintengine2d.Pt(4, 4), paintengine2d.White)
				_ = f.Advance(s)
			}
		}(float32(size))
	}
	wg.Wait()
}

// A keycap without its selector is still a keycap.
//
// `1⃣` is `1️⃣` without U+FE0F — unqualified, in UTS #51's words, and
// composed by every emoji font all the same. The digit is not a
// picture on its own, so marking the sequence only from left to right
// left this form split: a text digit from one face and an enclosing
// box from another, which the label's colour then tinted.
func TestAnUnqualifiedKeycapIsOnePicture(t *testing.T) {
	realFonts(t)
	f := BakeFamily(FamilyUI, WeightRegular, 24, paintengine2d.White)
	if glyphsDrawn(f, "\U0001F469") != 1 {
		t.Skip("this machine has no colour emoji face")
	}
	qualified := f.Advance("1️⃣")
	for _, seq := range []struct{ text, what string }{
		{"1⃣", "1 without the selector"},
		{"#⃣", "# without the selector"},
		{"*⃣", "* without the selector"},
	} {
		if n := glyphsDrawn(f, seq.text); n != 1 {
			t.Errorf("%s drew %d glyphs; one keycap is one glyph", seq.what, n)
		}
		if got := f.Advance(seq.text); got != qualified {
			t.Errorf("%s is %.3f wide and the qualified form is %.3f", seq.what, got, qualified)
		}
	}
}

// colourGlyphs is how many of the glyphs s draws come from a sheet
// that carries its own colours.
func colourGlyphs(f *Font, s string) int {
	n := 0
	for _, run := range f.shapeOf(s).runs {
		if isColourSheet(run.Atlas) {
			n += len(run.Glyphs)
		}
	}
	return n
}

// untintedPixels is how many of the pixels s draws are the same colour
// whatever the label's colour is — which is the definition of a
// picture, and a defect anywhere else.
func untintedPixels(t *testing.T, f *Font, s string) int {
	t.Helper()
	white := drawnPixels(t, f, s, paintengine2d.White)
	red := drawnPixels(t, f, s, paintengine2d.Color{R: 1, A: 1})
	n := 0
	for i, p := range white {
		if i < len(red) && p == red[i] {
			n++
		}
	}
	return n
}

// An enclosing keycap encloses a keycap, and leaves everything else to
// the text it was combining with.
//
// The fix above this one — the unqualified `1⃣`, which has no selector
// to mark it — marked whatever stood before U+20E3 as a picture, with
// no question asked about what it was. UTS #51 asks two: ED-14c limits
// the base of a keycap to `[0-9#*]`, and ED-8 says a text selector
// requests the text form, which this file already honours everywhere
// else. So `A⃣` went to the colour face, which has no A, and painted a
// tintable letter beside an untinted cell; and `1︎⃣`, which asks for
// text in as many words, got a picture anyway. A music player measured
// both the morning after the release that did it (item 50).
func TestAnEnclosingMarkOnPlainTextStaysText(t *testing.T) {
	realFonts(t)
	f := BakeFamily(FamilyUI, WeightRegular, 24, paintengine2d.White)
	if colourGlyphs(f, "\U0001F469") != 1 {
		t.Skip("this machine has no colour emoji face")
	}
	for _, c := range []struct{ text, what string }{
		{"A⃣", "an enclosing mark on a letter"},
		{"1︎⃣", "an enclosing mark on a digit asked for as text"},
		{"❤︎⃣", "an enclosing mark on a heart asked for as text"},
	} {
		if n := colourGlyphs(f, c.text); n != 0 {
			t.Errorf("%s drew %d glyphs from a colour sheet; it is text", c.what, n)
		}
		if n := untintedPixels(t, f, c.text); n != 0 {
			t.Errorf("%s left %d pixels the label's colour does not reach", c.what, n)
		}
	}
	// And the keycaps themselves are still one picture, qualified or
	// not: the half of this the release before got right.
	for _, c := range []struct{ text, what string }{
		{"1️⃣", "a qualified keycap"},
		{"1⃣", "an unqualified keycap"},
		{"#⃣", "an unqualified # keycap"},
	} {
		if n := colourGlyphs(f, c.text); n != 1 {
			t.Errorf("%s drew %d glyphs from a colour sheet; it is one picture", c.what, n)
		}
	}
}

// A character past the emoji planes is not an emoji.
//
// The test for "a picture on its own" was `r >= 0x1F000`, which is
// true of every character in the two planes above the pictographs as
// well — including U+20BB7, a Han ideograph. A video player found it
// in a file name, `🎬𠮷野家.mp4`: the Han was marked as colour, the
// emoji run kept it (a run keeps the parts of its picture whether or
// not the colour face has them), and the one installed face that could
// draw it was never asked. The clapper board beside it is what made
// the difference — `𠮷🎬` was always right, because nothing had marked
// the Han before a run was open to swallow it.
func TestAHanIdeographAfterAnEmojiKeepsItsOwnFace(t *testing.T) {
	realFonts(t)
	f := BakeFamily(FamilyUI, WeightRegular, 24, paintengine2d.White)
	if colourGlyphs(f, "\U0001F469") != 1 {
		t.Skip("this machine has no colour emoji face")
	}
	const (
		emoji = "\U0001F3AC" // 🎬 clapper board
		han   = "\U00020BB7" // 𠮷, plane 2
	)
	if n := colourGlyphs(f, emoji+han); n != 1 {
		t.Errorf("`%s` drew %d glyphs from a colour sheet; only the clapper board is a picture", emoji+han, n)
	}
	// Laid out together, it is laid out the same as each on its own:
	// the measure the video player sent (toolkit 49.81, separate
	// 44.91) with the emoji face answering for a character it has no
	// glyph for.
	if got, want := f.Advance(emoji+han), f.Advance(emoji)+f.Advance(han); got != want {
		t.Errorf("`%s` is %.4f wide; the two apart are %.4f", emoji+han, got, want)
	}
	if n := untintedPixels(t, f, han); n != 0 {
		t.Errorf("the Han alone left %d pixels untinted; it is text", n)
	}
}
