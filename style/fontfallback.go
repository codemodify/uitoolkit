package style

import (
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

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
	// Marks and emoji, where a face draws them as outlines. A colour
	// emoji face is not named here and does not need to be: a build
	// with `font_emoji` finds one for itself, and no other build can
	// draw a colour glyph at all (see style/emoji_on.go).
	"Noto Emoji", "Symbola", "Noto Sans Symbols 2", "Noto Sans Symbols",
	// The scripts that join or reorder, which only a build that shapes
	// borrows ([shapes]): until v0.23.9 the fallback refused them, so
	// naming a face for them would have been a lie. A mail client
	// named these itself in the meantime, which is what told us the
	// list was short.
	//
	// The script's own face first, because that is what a Linux
	// machine has, then the one face a platform covers many scripts
	// with — Nirmala UI is every Indic script on Windows, Segoe UI
	// carries Arabic and Hebrew there, and macOS has its own.
	"Noto Sans Arabic", "Noto Naskh Arabic", "Geeza Pro",
	"Noto Sans Hebrew", "Arial Hebrew",
	"Noto Sans Devanagari", "Kohinoor Devanagari", "Lohit Devanagari",
	"Noto Sans Bengali", "Noto Sans Gujarati", "Noto Sans Gurmukhi",
	"Noto Sans Kannada", "Noto Sans Malayalam", "Noto Sans Oriya",
	"Noto Sans Tamil", "Noto Sans Telugu", "Noto Sans Sinhala",
	"Noto Sans Thai", "Leelawadee UI", "Thonburi",
	"Noto Sans Lao", "Noto Sans Khmer", "Noto Sans Myanmar",
	"Noto Sans Ethiopic", "Noto Sans Armenian", "Noto Sans Georgian",
	"Nirmala UI", "Segoe UI",
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
	// Everything decided under the old list is now wrong in both
	// directions: a rune remembered as "nobody has it" must be looked
	// for again, and one already borrowed must be given back if the
	// face that lent it is no longer named. media-player-video and
	// media-player-music both found this by changing the list at
	// runtime and watching nothing change.
	fallbackEpoch.Add(1)
	fallbackChanged()
}

// fallbackEpoch counts changes to the list. A shaped run remembers the
// epoch it was laid out under, so a cached one is discarded rather than
// drawn with glyphs from a face that is no longer lending.
var fallbackEpoch atomic.Uint64

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

// borrowable reports whether a glyph for r may honestly be borrowed at
// all.
//
// One glyph per rune is the truth for CJK, Cyrillic, Greek, symbols and
// emoji, and a lie for the scripts that need shaping or reordering: a
// face lending Arabic letterforms draws them unjoined and in logical
// order, which is legibly wrong where a box is honestly illegible. The
// toolkit has no shaping and no bidi ([docs/contracts.md]), so it does
// not pretend here either — these scripts stay boxes until it does.
//
// Checked against Go's own script tables, which are the Unicode ones.
//
// The rule is about *this build*, not about the scripts: a build that
// shapes (`font_complex`) draws them properly and borrows them like
// anything else, so it asks [shapes] first and never gets here.
func borrowable(r rune) bool {
	for _, t := range unshapedScripts {
		if unicode.Is(t, r) {
			return false
		}
	}
	return true
}

// unshapedScripts are the writing systems a glyph at a time cannot
// honestly draw: the cursive ones that join, the Indic ones that
// reorder, and the right-to-left ones that would come out in the wrong
// order without [UAX #9].
var unshapedScripts = []*unicode.RangeTable{
	unicode.Arabic, unicode.Syriac, unicode.Thaana, unicode.Nko,
	unicode.Hebrew, unicode.Samaritan, unicode.Mandaic, unicode.Adlam,
	unicode.Devanagari, unicode.Bengali, unicode.Gurmukhi, unicode.Gujarati,
	unicode.Oriya, unicode.Tamil, unicode.Telugu, unicode.Kannada,
	unicode.Malayalam, unicode.Sinhala, unicode.Khmer, unicode.Myanmar,
	unicode.Thai, unicode.Lao, unicode.Tibetan, unicode.Mongolian,
	unicode.Javanese, unicode.Balinese, unicode.Tifinagh,
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
