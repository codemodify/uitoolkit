//go:build font_fallback || font_complex || font_emoji

package style

import "sync"

// Borrowing a glyph, in the build that does it.
//
// Two things make this cheap enough to sit in the layout pass. A face is
// consulted only for a rune the chosen face has not got, which in Latin
// text is never; and the answer for a rune is remembered, so a playlist
// of a thousand CJK titles walks the family list once per distinct
// character, not once per title.
//
// The borrowed face is baked **to the baseline of the face it is lending
// to**. A glyph's vertical bearing is measured from the top of the line
// box using its own sheet's ascent, so a face with a taller ascent would
// otherwise hang its glyphs below the Latin beside them.

// fallbackMemo remembers which face lends which rune, per (size, weight,
// baseline). A miss is remembered too — as nil — because a rune no
// installed face has is the expensive case and the common one on a
// machine with no CJK font.
var fallbackMemo = struct {
	mu sync.RWMutex
	m  map[fallbackKey]*Font
}{m: map[fallbackKey]*Font{}}

type fallbackKey struct {
	r rune
	// family, because the answer depends on the face asking: a rune
	// one face draws itself is borrowed by another, and without this
	// the two would share an entry.
	family string
	size   int
	weight Weight
	ascent int
}

// fallbackFor is the face that lends f the glyph for r, or nil for none
// — which is every rune f draws itself, and every rune nobody has.
func fallbackFor(f *Font, r rune) *Font {
	if f == nil || r == ' ' || r == '\t' || r == '\n' || !borrowable(r) {
		return nil
	}
	key := fallbackKey{r: r, family: f.Family, size: quantFont(f.Size),
		weight: f.Weight, ascent: quantFont(f.Ascent)}
	fallbackMemo.mu.RLock()
	hit, ok := fallbackMemo.m[key]
	fallbackMemo.mu.RUnlock()
	if ok {
		return hit
	}
	found := searchFallback(f, r)
	fallbackMemo.mu.Lock()
	if len(fallbackMemo.m) >= 4096 {
		// A cap rather than an LRU: the set of distinct runes an
		// application draws is bounded by its content, and a clear
		// costs one walk of the family list per rune after it.
		clear(fallbackMemo.m)
	}
	fallbackMemo.m[key] = found
	fallbackMemo.mu.Unlock()
	return found
}

// searchFallback walks the families in order and bakes the first that
// has r, at f's size, weight and baseline.
func searchFallback(f *Font, r rune) *Font {
	// Nothing to borrow for a rune the face draws itself. The layout
	// pass has usually asked already, but this is what makes the
	// answer true on its own.
	if f.covers(r) {
		return nil
	}
	for _, name := range FallbackFamilies() {
		fam := ResolveFont([]string{name})
		// Not installed, or the face we are already using: either way
		// there is nothing to borrow. ResolveFont answers "" with
		// UITK_SYSTEM_FONTS=0, which is how the pin takes the fallback
		// with it.
		if fam == "" || fam == f.Family {
			continue
		}
		fb := bakeFallbackFace(fam, f.Weight, f.Size, f.Ascent)
		if fb == nil {
			continue
		}
		if fb.covers(r) {
			return fb
		}
	}
	return nil
}

// bakeFallbackFace is [BakeFamily] for a face that must sit on somebody
// else's baseline, cached by that baseline as well as the usual three.
func bakeFallbackFace(family string, weight Weight, size, ascent float32) *Font {
	if size < 8 {
		size = 8
	}
	key := fontKey{family: family, weight: weight, size: quantFont(size), ascent: quantFont(ascent)}
	if cached := cachedFont(key); cached != nil {
		return cached
	}
	f, err := bakeFaceAscent(family, weight, size, ascent)
	if err != nil {
		return nil
	}
	return storeFont(key, f)
}

func quantFont(v float32) int { return int(v*4 + 0.5) }

// fallbackChanged throws away every remembered decision, because the
// list they were made against has been replaced.
func fallbackChanged() {
	fallbackMemo.mu.Lock()
	clear(fallbackMemo.m)
	fallbackMemo.mu.Unlock()
}
