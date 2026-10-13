//go:build font_complex

package style

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-text/typesetting/bidi"

	"github.com/codemodify/paintengine2d"
)

// shapeInk draws a shaped run and counts the pixels it leaves.
func shapeInk(t *testing.T, run paintengine2d.GlyphRun, at paintengine2d.Point) int {
	t.Helper()
	img := paintengine2d.NewImage(96, 48)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawGlyphs(run, at, paintengine2d.Fill(paintengine2d.White))
	img.Touch()
	n := 0
	for y := 0; y < 48; y++ {
		for x := 0; x < 96; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 0 {
				n++
			}
		}
	}
	return n
}

// The spike's claim, in one test: a shaper answers in glyph indices, the
// glyphs it picks are ones no rune maps to, and the sheet and the
// renderer draw them anyway once the key says which numbering it is in.
//
// JetBrains Mono is bundled, so this needs nothing installed. Its `calt`
// table turns "->" into an arrow: two glyphs that are not the glyphs for
// '-' and '>', and that no rune of Unicode asks for.
func TestAShaperPicksGlyphsNoRuneMapsTo(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "0")
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	face, err := faceFor(FamilyMono, WeightRegular)
	if err != nil {
		t.Fatal(err)
	}

	const text = "->"
	shaped := face.shapeComplex(text, 16)
	if len(shaped) == 0 {
		t.Fatal("the shaper returned nothing")
	}

	// What the rune path would have drawn.
	naive := make(map[uint32]bool)
	face.mu.Lock()
	for _, r := range text {
		g, err := face.font.GlyphIndex(&face.buf, r)
		if err != nil {
			face.mu.Unlock()
			t.Fatal(err)
		}
		naive[uint32(g)] = true
	}
	face.mu.Unlock()

	picked := false
	for _, g := range shaped {
		if !naive[g.GID] {
			picked = true
		}
	}
	if !picked {
		t.Fatalf("shaping %q picked only the glyphs the runes map to; the ligature did not fire", text)
	}

	// And they draw. The atlas is the same sheet the rest of the toolkit
	// uses, with the top bit of the key saying "glyph index".
	at := newOTAtlas(face, 16)
	at.ascent, _ = face.metrics(16)
	if _, err := at.bake(otPreload); err != nil {
		t.Fatal(err)
	}
	run := at.glyphRun(shaped)
	if len(run.Glyphs) != len(shaped) {
		t.Fatalf("the run has %d glyphs for %d shaped", len(run.Glyphs), len(shaped))
	}
	for _, g := range run.Glyphs {
		if g.ID&shapedGIDBit == 0 {
			t.Errorf("glyph %d is keyed as a rune", g.ID)
		}
		if _, ok := run.Atlas.Cell(g.ID); !ok {
			t.Errorf("the sheet has no cell for glyph %d", g.ID&^shapedGIDBit)
		}
	}
	if ink := shapeInk(t, run, paintengine2d.Pt(8, 30)); ink < 20 {
		t.Errorf("the shaped run drew %d pixels; it is not on the sheet", ink)
	}
}

// Arabic is the case the rune path cannot represent at all: four runes
// become four *joining forms*, none of which is the glyph its rune maps
// to, laid out right to left.
//
// It needs a face with Arabic in it, which is the machine's business and
// not the toolkit's, so it skips when there is none — and it reaches the
// machine's file through fakeFonts, one line of it, rather than by
// turning the installed-font index on: the index is built once per
// process, and a test that warms it with this machine's fonts changes
// what every test after it measures.
func TestShapingJoinsArabic(t *testing.T) {
	path := arabicFontFile(t)
	if path == "" {
		t.Skip("no face on this machine covers Arabic")
	}
	fakeFonts(t, "Noto Arabic Test\t80\t0\t0\t"+path)
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	face, err := faceFor("Noto Arabic Test", WeightRegular)
	if err != nil || face == nil {
		t.Skipf("the Arabic face at %s did not load: %v", path, err)
	}

	const word = "سلام"
	shaped := face.shapeComplex(word, 16)
	if len(shaped) == 0 {
		t.Fatal("the shaper returned nothing")
	}
	var naive []uint32
	face.mu.Lock()
	for _, r := range word {
		g, _ := face.font.GlyphIndex(&face.buf, r)
		naive = append(naive, uint32(g))
	}
	face.mu.Unlock()
	if len(naive) == 0 || naive[0] == 0 {
		t.Skip("this face has no Arabic")
	}

	same := 0
	for i, g := range shaped {
		if i < len(naive) && g.GID == naive[i] {
			same++
		}
	}
	if same == len(shaped) {
		t.Fatalf("shaping %q gave the isolated forms %v; nothing joined", word, naive)
	}
}

// arabicFontFile is any file on this machine with Arabic in it, or "".
func arabicFontFile(t *testing.T) string { return scriptFontFile(t, "arabic") }

// scriptFontFile is any face file whose name says it covers want.
func scriptFontFile(t *testing.T, want string) string {
	t.Helper()
	for _, dir := range []string{
		"/usr/share/fonts", "/usr/local/share/fonts", "/usr/share/fonts/noto",
	} {
		var found string
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || found != "" {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.HasSuffix(name, ".ttf") && !strings.HasSuffix(name, ".otf") {
				return nil
			}
			if strings.Contains(name, want) && !strings.Contains(name, "ui") &&
				strings.Contains(name, "regular") && !strings.Contains(name, "rashi") {
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

// UAX #9 L2, the rule that orders the runs of a line for display: from
// the highest level down to the lowest odd one, reverse every stretch at
// that level or above. It is the whole of bidirectional *layout* — the
// shaper orders the glyphs inside a run, and this orders the runs — so
// it is worth a test that owes nothing to a font.
func TestReorderVisualFollowsL2(t *testing.T) {
	for _, c := range []struct {
		name   string
		levels []bidi.Level
		want   []int // the original indices, in the order they are drawn
	}{
		{"all one way", []bidi.Level{0, 0, 0}, []int{0, 1, 2}},
		{"Hebrew in an English sentence", []bidi.Level{0, 1, 0}, []int{0, 1, 2}},
		{"two Hebrew runs in English", []bidi.Level{0, 1, 1, 0}, []int{0, 2, 1, 3}},
		{"English in a Hebrew sentence", []bidi.Level{1, 2, 1}, []int{2, 1, 0}},
		{"a number inside English inside Hebrew", []bidi.Level{1, 2, 3, 2, 1}, []int{4, 1, 2, 3, 0}},
	} {
		t.Run(c.name, func(t *testing.T) {
			parts := make([]part, len(c.levels))
			for i, l := range c.levels {
				parts[i] = part{level: l, start: i, end: i + 1}
			}
			reorderVisual(parts)
			got := make([]int, len(parts))
			for i, p := range parts {
				got[i] = p.start
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("levels %v drew as %v, want %v", c.levels, got, c.want)
			}
		})
	}
}

// Shaping must not move the Latin text the toolkit is full of. GPOS
// kerning and the face's own `kern` table are two readings of the same
// design, and a layout that disagreed with the rune path would move
// every label in every window of a font_complex build.
func TestShapedAndRuneLayoutAgree(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "0")
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{FamilyUI, FamilyMono} {
		f := BakeFamily(family, WeightRegular, 16, paintengine2d.White)
		for _, text := range []string{
			"A", "AV", "AVATAR", "Taverns", "Open file…", "The quick brown fox",
			"1,234.56", "a b  c", "Wavy Yo To LT",
		} {
			shaped, ok := complexLayout(f, text)
			if !ok {
				t.Fatalf("%s: the shaper refused %q", family, text)
			}
			runes := f.layout(text, f.ensure(text))
			if d := shaped.adv - runes.adv; d > 0.5 || d < -0.5 {
				t.Errorf("%s %q: shaped advance %.3f, rune advance %.3f",
					family, text, shaped.adv, runes.adv)
			}
			if len(shaped.xs) != len(runes.xs) {
				t.Errorf("%s %q: %d carets shaped, %d by rune",
					family, text, len(shaped.xs), len(runes.xs))
			}
		}
	}
}

// A right-to-left run reads right to left: the caret before its first
// character is to the *right* of the caret before its second, and the
// run as a whole sits where the sentence around it puts it.
func TestARightToLeftRunCaretsRunBackwards(t *testing.T) {
	path := scriptFontFile(t, "hebrew")
	if path == "" {
		t.Skip("no face on this machine covers Hebrew")
	}
	fakeFonts(t, "Noto Hebrew Test\t80\t0\t0\t"+path)
	// With a baseline, the way a lending face is baked: a Hebrew face
	// has no Latin, and a face asked for by name has to draw 'A' or it
	// would make an unreadable window.
	f, err := bakeFaceAscent("Noto Hebrew Test", WeightRegular, 16, 13)
	if err != nil || f == nil {
		t.Skipf("the Hebrew face at %s did not load: %v", path, err)
	}
	const word = "שלום" // four letters, right to left
	s, ok := complexLayout(f, word)
	if !ok {
		t.Fatal("the shaper refused the word")
	}
	if len(s.xs) != len([]rune(word))+1 {
		t.Fatalf("%d carets for %d runes", len(s.xs), len([]rune(word)))
	}
	for i := 1; i < len([]rune(word)); i++ {
		if s.xs[i] >= s.xs[i-1] {
			t.Errorf("caret %d at %.2f is not left of caret %d at %.2f; the run reads left to right",
				i, s.xs[i], i-1, s.xs[i-1])
		}
	}
	// The first character's caret is at the *right* edge of the line —
	// the line's whole width — and not a character short of it.
	if d := s.xs[0] - s.adv; d > 0.01 || d < -0.01 {
		t.Errorf("the first character's caret is at %.2f; the right edge is %.2f", s.xs[0], s.adv)
	}
	if s.xs[len(s.xs)-1] != 0 {
		t.Errorf("the caret past the last character is at %.2f, not at the left edge", s.xs[len(s.xs)-1])
	}
}
