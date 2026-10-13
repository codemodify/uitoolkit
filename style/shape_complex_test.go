//go:build font_complex

package style

import (
	"testing"

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
// to. It needs an Arabic face, which is the machine's business and not
// the toolkit's, so it skips rather than fails when there is none.
func TestShapingJoinsArabic(t *testing.T) {
	t.Setenv("UITK_SYSTEM_FONTS", "1")
	if err := LoadEmbeddedFonts(); err != nil {
		t.Fatal(err)
	}
	const word = "سلام"
	var face *otFace
	for _, family := range []string{"Noto Naskh Arabic", "Noto Sans Arabic", "Noto Kufi Arabic", "DejaVu Sans", "Noto Sans"} {
		if f, err := faceFor(family, WeightRegular); err == nil && f != nil {
			face = f
			break
		}
	}
	if face == nil {
		t.Skip("no face on this machine covers Arabic")
	}

	shaped := face.shapeComplex(word, 16)
	if len(shaped) == 0 {
		t.Fatal("the shaper returned nothing")
	}
	face.mu.Lock()
	var naive []uint32
	for _, r := range word {
		g, _ := face.font.GlyphIndex(&face.buf, r)
		naive = append(naive, uint32(g))
	}
	face.mu.Unlock()
	if len(naive) > 0 && naive[0] == 0 {
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
