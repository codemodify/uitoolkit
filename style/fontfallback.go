package style

import (
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Which faces lend a glyph the chosen one has not got.
//
// **The rule the toolkit keeps by default is one face draws everything**,
// and a rune that face lacks is a box ([docs/contracts.md]). It is a rule
// with a cost: two different CJK song titles are then the same row of
// boxes, pixel for pixel, so a playlist shows two identical rows for two
// different songs — which is what media-player-music reported as its item
// 34, with the accessible names correctly distinct and nothing to see.
//
// Built with `-tags font_fallback`, a rune the chosen face has not got is
// looked for in these families instead, in order, and drawn from the
// first that has it. Without the tag the names here are inert: nothing
// is searched and the box stays.
//
// It costs no bytes in the binary either way. The faces are the
// machine's own — the toolkit embeds two Latin faces and nothing else,
// and a CJK family is 18 MB, which is why it never will.

// defaultFallbackFamilies are the families a desktop is likely to have,
// widest coverage first. Names only: [ResolveFont] decides which of them
// is actually installed, and one that is not is skipped.
//
// Four Noto Sans CJK families are listed because their Han glyphs
// genuinely differ — 直 is drawn one way in a Japanese face and another
// in a Chinese one — and a Go string carries no language to choose by.
// An application that knows better says so with [SetFallbackFamilies].
var defaultFallbackFamilies = []string{
	// CJK, in the order a machine usually has them.
	"Noto Sans CJK SC", "Noto Sans CJK TC", "Noto Sans CJK JP", "Noto Sans CJK KR",
	"Source Han Sans SC", "Source Han Sans",
	"PingFang SC", "Hiragino Sans", "Microsoft YaHei", "Malgun Gothic", "Meiryo",
	"WenQuanYi Zen Hei", "Droid Sans Fallback",
	// Everything else a Latin face is likely to be short of.
	"Noto Sans", "DejaVu Sans", "FreeSans", "Arial Unicode MS", "Segoe UI Symbol",
}

var fallback = struct {
	mu    sync.RWMutex
	names []string
	set   bool
}{}

// SetFallbackFamilies names the faces a glyph is borrowed from, in order,
// replacing the built-in list; no names at all puts the built-in list
// back.
//
// It is the one place an application can say what the toolkit cannot
// work out: which face a mixed-script string should borrow from. A
// Japanese application names a Japanese face first, a Chinese one a
// Chinese face, and both get the Han glyphs their readers expect.
//
// The names are families, as installed — "Noto Sans CJK JP", not a file
// — and one that is not installed is skipped. This has no effect in a
// build without `-tags font_fallback`.
func SetFallbackFamilies(names ...string) {
	fallback.mu.Lock()
	defer fallback.mu.Unlock()
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	fallback.names, fallback.set = out, len(out) > 0
}

// FallbackFamilies is the list in force: the application's, or the
// built-in one.
func FallbackFamilies() []string {
	fallback.mu.RLock()
	defer fallback.mu.RUnlock()
	if !fallback.set {
		return append([]string(nil), defaultFallbackFamilies...)
	}
	return append([]string(nil), fallback.names...)
}

// covers reports whether this face draws r itself, rather than pointing
// it at the sheet's .notdef box.
//
// It is the same question [bakeFace] asks of 'A' to decide whether a
// family can draw the interface at all, and the answer is a map lookup
// once the glyph has been baked.
func (f *Font) covers(r rune) bool {
	if f == nil {
		return false
	}
	at := f.atlasForRune(r)
	if at == nil {
		return false
	}
	cell, ok := at.Cell(paintengine2d.GlyphID(r))
	if !ok {
		return false
	}
	if f.ot != nil && f.ot.hasNotdef && cell.Src == f.ot.notdef.Src {
		return false
	}
	return true
}

// atlasForRune is the sheet with r baked into it, or the static sheet
// for a bitmap font.
func (f *Font) atlasForRune(r rune) *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot == nil {
		return f.static
	}
	return f.ot.ensureRune(r)
}
