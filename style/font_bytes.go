package style

import (
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// The byte-slice text path: measuring, hit-testing and drawing that
// **caches nothing and allocates no string**.
//
// Every other path through this file goes through [Font.shapeOf], which
// keeps the shaped run in a cache keyed by the string it shaped. That is
// the right trade for a label, which is drawn the same way sixty times a
// second. It is the wrong one for a passphrase: `string(buf)` cannot be
// wiped, and the cache would then hold the secret for as long as the
// face lives.
//
// So a secret is measured and drawn from the `[]byte` it is stored in,
// rune by rune, with nothing written down afterwards. Glyph cells are
// baked into the face's sheet as they always are — a passphrase of Latin
// letters bakes no cell a window has not already baked — but the *run*,
// which is what says which glyphs in which order, is built per call and
// dropped.
//
// The price is pair kerning, which is a property of a shaped run and so
// is not applied here. Revealed text is set a hair wider than the same
// string in a label. That is the cost of not remembering it.

// ensureRunes bakes every rune of b, without building a string.
func (f *Font) ensureRunes(b []byte) *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot == nil {
		return f.static
	}
	atlas := f.GlyphAtlas()
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if n == 0 {
			break
		}
		atlas = f.ot.ensureRune(r)
		i += n
	}
	return atlas
}

// layoutRunes places each rune of b at the sum of the advances before it
// and snaps the glyphs onto the pixel grid, exactly as [Font.layout]
// does, but over bytes and without the kerning a shaped run would carry.
// xs is where each rune starts, then where the run ends.
func (f *Font) layoutRunes(b []byte, atlas *paintengine2d.FontAtlas) (paintengine2d.GlyphRun, []float32, float32) {
	run := paintengine2d.GlyphRun{Atlas: atlas}
	xs := make([]float32, 0, len(b)+1)
	var x float32
	if atlas == nil {
		return run, append(xs, 0), 0
	}
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if n == 0 {
			break
		}
		// One entry per byte, so a caret offset into the buffer indexes
		// straight into xs: a rune's continuation bytes repeat where it
		// starts, which is where a caret in the middle of one belongs.
		for k := 0; k < n; k++ {
			xs = append(xs, x)
		}
		i += n
		id := paintengine2d.GlyphID(r)
		cell, ok := atlas.Cell(id)
		if !ok {
			continue
		}
		run.Glyphs = append(run.Glyphs, paintengine2d.Glyph{ID: id, X: x})
		adv := cell.Advance
		if adv <= 0 {
			adv = cell.Src.Dx() + 1
		}
		x += adv
	}
	xs = append(xs, x)
	return snapRun(run, atlas), xs, x
}

// AdvanceBytes is [Font.Advance] for a byte slice: the width the runes of
// b occupy, measured without shaping them into a cache.
//
// It is for text that must not be remembered — see the note at the top of
// this file. For ordinary text use [Font.Advance], which is faster on the
// second call and kerns.
func (f *Font) AdvanceBytes(b []byte) float32 {
	if f == nil || len(b) == 0 {
		return 0
	}
	_, _, adv := f.layoutRunes(b, f.ensureRunes(b))
	return adv
}

// CaretXBytes is where a caret at byte offset i in b sits, measured the
// same way. An offset inside a rune reads as the start of that rune.
func (f *Font) CaretXBytes(b []byte, i int) float32 {
	if f == nil || i <= 0 || len(b) == 0 {
		return 0
	}
	_, xs, _ := f.layoutRunes(b, f.ensureRunes(b))
	if i >= len(xs) {
		i = len(xs) - 1
	}
	return xs[i]
}

// IndexAtBytes is the byte offset of the rune boundary nearest x, the
// hit-test a click in a secret field needs.
func (f *Font) IndexAtBytes(b []byte, x float32) int {
	if f == nil || len(b) == 0 || x <= 0 {
		return 0
	}
	_, xs, _ := f.layoutRunes(b, f.ensureRunes(b))
	best, bestD := 0, float32(-1)
	for i := 0; i <= len(b); i++ {
		if i > 0 && i < len(b) && !utf8.RuneStart(b[i]) {
			continue
		}
		at := xs[len(xs)-1]
		if i < len(xs) {
			at = xs[i]
		}
		d := at - x
		if d < 0 {
			d = -d
		}
		if bestD < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// DrawBytes is [Font.Draw] for a byte slice: it paints the runes of b at
// origin and leaves nothing behind — no shaped run in the cache, no
// string on the heap.
func (f *Font) DrawBytes(ctx *paintengine2d.Context, b []byte, origin paintengine2d.Point, tint paintengine2d.Color) {
	if f == nil || ctx == nil || len(b) == 0 {
		return
	}
	run, _, _ := f.layoutRunes(b, f.ensureRunes(b))
	if len(run.Glyphs) == 0 {
		return
	}
	col := tint
	if col == (paintengine2d.Color{}) {
		col = f.Color
	}
	if col == (paintengine2d.Color{}) {
		col = paintengine2d.White
	}
	filter := paintengine2d.FilterNearest
	if m := ctx.Matrix(); m.IsTranslation() {
		origin = paintengine2d.Pt(snapCoord(origin.X+m.E)-m.E, snapCoord(origin.Y+m.F)-m.F)
	} else {
		filter = paintengine2d.FilterBilinear
	}
	ctx.DrawGlyphs(run, origin, paintengine2d.Paint{Color: col, Filter: filter})
}

// ShapeCacheKeysForTest is what this face has shaped and kept. It is
// exported for the tests in other packages that assert a secret never
// reaches the cache, and it is the only way to ask from outside.
func ShapeCacheKeysForTest(f *Font) []string {
	if f == nil || f.shaped == nil {
		return nil
	}
	f.shaped.mu.Lock()
	defer f.shaped.mu.Unlock()
	out := make([]string, 0, len(f.shaped.m))
	for k := range f.shaped.m {
		out = append(out, k)
	}
	return out
}
