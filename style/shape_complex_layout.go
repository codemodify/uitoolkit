//go:build font_complex || font_emoji

package style

import (
	"sync"
	"unicode"

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/di"
	gofont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/codemodify/paintengine2d"
)

// Shaping a whole string: the face's own tables choose the glyphs, the
// bidirectional algorithm orders them, and a fallback lends what the
// face has not got — one run at a time rather than one rune at a time.
//
// The shape of it:
//
//	paragraph direction (UAX #9 P2/P3)
//	  → Segmenter.Split: bidi runs, then script, then face
//	    → harfbuzz per run: GSUB, GPOS, the glyphs and where they go
//	      → reorder the runs for display (UAX #9 L2)
//	        → one atlas group per face, and a caret table per rune
//
// What the rune path did and this does not: look each rune up in the
// sheet by its own code point, and add pair kerning from the `kern`
// table. GPOS does the kerning now, and it does it for pairs the old
// table never had.

// shaperPoolC and segPool keep the harfbuzz buffer and the segmenter's
// scratch between calls. Both hold state across one call only.
var segPool = sync.Pool{New: func() any { return &shaping.Segmenter{} }}

// rtlDirection is di.DirectionRTL under a name the rest of the package
// can use without importing the shaper's own packages.
const rtlDirection = di.DirectionRTL

// arabicIndicDigits are the one place where "the script is right-to-left"
// is the wrong answer to "is this character strong": U+0660.. are Arabic
// by script and bidi class AN, which P2 steps over.
func strongDirection(r rune) (di.Direction, bool) {
	switch {
	case r >= 0x0660 && r <= 0x0669, r >= 0x066B && r <= 0x066C, r >= 0x06F0 && r <= 0x06F9:
		return 0, false
	}
	s := language.LookupScript(r)
	switch s {
	case language.Common, language.Inherited, language.Unknown:
		return 0, false
	}
	return scriptDirection(s), true
}

// paragraphDirection is UAX #9 P2 and P3: the direction of the first
// strong character, and left-to-right when there is none.
//
// Script stands in for the bidi class here, which is exact for every
// strong character except the Arabic-Indic digits [strongDirection]
// excludes by hand. The toolkit has no API for an application to state
// the direction of a label, and this is what decides it; a paragraph
// that wants the other one would need one.
func paragraphDirection(runes []rune) di.Direction {
	for _, r := range runes {
		if d, ok := strongDirection(r); ok {
			return d
		}
	}
	return di.DirectionLTR
}

// glyphSheet is a sheet the shaped glyphs of one face are baked into.
//
// Two kinds: [otAtlas], an A8 coverage mask of outlines, which is every
// face that draws letters; and [emojiAtlas], a colour sheet of pictures,
// which is an emoji face in a font_emoji build. The layout treats them
// alike — ask for the glyphs, get a sheet — and [Font.Draw] tells them
// apart by the only thing that differs: a colour sheet is not tinted.
type glyphSheet interface {
	ensureGIDs(gids []uint32) *paintengine2d.FontAtlas
	atlas() *paintengine2d.FontAtlas
}

// uitkFontmap answers the segmenter's "which face draws this rune" with
// the toolkit's own answer: this font's face, or the first family in
// [FallbackFamilies] that has the rune. It remembers which sheet each
// face it handed out belongs to, since the glyphs come back named by
// face and have to be baked into that face's own sheet.
type uitkFontmap struct {
	f    *Font
	prim *gofont.Face
	of   map[*gofont.Face]glyphSheet
	// runes is the whole string, and colourAt says which of its
	// positions are to be drawn as emoji ([emojiPresentation]).
	runes    []rune
	colourAt []bool
}

// ResolveFace satisfies [shaping.Fontmap] and chooses nothing: the
// segmenter is asked for direction and script, and the face is picked
// per position afterwards ([uitkFontmap.faceRuns]). It has to be, for
// two reasons the segmenter cannot see — a presentation selector makes
// one ❤ a picture and leaves the next one a dingbat, and the
// segmenter's own rule that whitespace never forces a face change puts
// the space after an emoji *in the emoji font*, whose space is a
// quarter wider than its em.
func (m *uitkFontmap) ResolveFace(rune) *gofont.Face { return m.prim }

// faceAt is the face that draws the character at i.
func (m *uitkFontmap) faceAt(i int) *gofont.Face {
	r := m.runes[i]
	// A character to be drawn as emoji goes to a colour face before
	// anything else is asked, including the face this font is. An
	// outline face that happens to carry a black-and-white 👍 is not
	// the answer when the machine has the picture, and taking it split
	// 👍🏽 in two: the base from one face and the skin tone from
	// another, which no shaper can then join.
	if m.colourAt[i] {
		if g := m.colour(r); g != nil {
			return g
		}
	}
	if m.f.ot != nil && m.f.ot.face.hasGlyph(r) {
		return m.prim
	}
	// No `borrowable` gate here. That rule refuses to lend the scripts
	// a glyph at a time cannot honestly draw — the ones that join or
	// reorder — and it is right in the build that has no shaper. This
	// is the shaper, so Arabic, Hebrew and Devanagari are exactly what
	// it should be borrowing.
	if fb := fallbackFor(m.f, r); fb != nil && fb.ot != nil {
		if g := fb.ot.face.gotext(); g != nil {
			m.of[g] = fb.ot
			return g
		}
	}
	// Nothing with an outline for it. A colour face may still have a
	// picture — 👀 is a PNG in Noto Color Emoji and an outline in
	// nothing at all.
	if g := m.colour(r); g != nil {
		return g
	}
	// Never nil: the sheet's .notdef box is the honest answer, and the
	// caller still needs a face to measure it with.
	return m.prim
}

// isColour reports that a face's glyphs are pictures of their own.
func (m *uitkFontmap) isColour(g *gofont.Face) bool {
	if g == nil {
		return false
	}
	_, outlines := m.of[g].(*otAtlas)
	return !outlines
}

// keepsItsRun reports a character that does not change face on its own
// account: control codes, separators, spaces and the default-ignorable
// ones, which pango and go-text both let continue whatever run they
// are in rather than cut it in two.
func keepsItsRun(r rune) bool {
	switch {
	case unicode.Is(unicode.Cc, r), unicode.Is(unicode.Cs, r),
		unicode.Is(unicode.Zl, r), unicode.Is(unicode.Zp, r):
		return true
	case unicode.Is(unicode.Zs, r) && r != '\u1680':
		return true
	case unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r):
		return true
	}
	return false
}

// faceRuns splits one piece of the segmenter's work into runs of a
// single face, by position.
//
// A character that keeps its run does so — *unless* the run is a
// colour face. An emoji font is not a text font: its space is 1.245 em
// in Noto Color Emoji, so the space after 👀 came out five times the
// width of the space before it (a mail client measured it). Whitespace
// at the edge of a picture belongs to the words, not to the picture.
func (m *uitkFontmap) faceRuns(in shaping.Input) []shaping.Input {
	var out []shaping.Input
	start := in.RunStart
	var face *gofont.Face
	flush := func(end int) {
		if end <= start || face == nil {
			return
		}
		piece := in
		piece.RunStart, piece.RunEnd, piece.Face = start, end, face
		out = append(out, piece)
	}
	for i := in.RunStart; i < in.RunEnd; i++ {
		// Part of the picture stays with the picture, whether or not
		// the colour face has a glyph of its own for it. A joiner, a
		// presentation selector and an enclosing keycap are all
		// characters the *font* composes away (UTS #51) — asking
		// whether the face "has" them and sending them elsewhere when
		// it has not is what broke every family and profession emoji
		// into its separate pictures, and every keycap into a digit
		// and a box. Three applications reported it within half an
		// hour of the release that did it.
		if face != nil && m.isColour(face) && m.colourAt[i] {
			continue
		}
		want := m.faceAt(i)
		if face != nil && keepsItsRun(m.runes[i]) {
			// Whitespace keeps the run it is in — unless that run is
			// a colour face, because an emoji font's space is a
			// quarter wider than its em.
			if !m.isColour(face) {
				continue
			}
			want = m.prim
		}
		switch {
		case face == nil:
			face = want
		case want != face:
			flush(i)
			start, face = i, want
		}
	}
	flush(in.RunEnd)
	return out
}

// emojiPresentation says, for each *position* in the string, whether
// UTS #51 asks for the character there to be drawn as emoji.
//
// By position and not by character, because the same character can be
// both in one string: `❤️ ❤` is a picture and then a dingbat, and a
// table keyed by the rune made the second one a picture as well. A
// music player found that, and that U+FE0E — the selector that asks
// for the *text* form — was being ignored.
//
// The rules: a character from the pictographic planes is emoji on its
// own; U+FE0F after one asks for the emoji form of a character that is
// text by default; U+FE0E after one asks for the text form and wins;
// and a zero-width joiner between emoji is part of the one glyph a
// family or a flag becomes. The same joiner between Devanagari letters
// is not emoji and is left alone.
func emojiPresentation(runes []rune) []bool {
	const (
		vs15   = 0xFE0E // text presentation
		vs16   = 0xFE0F // emoji presentation
		zwj    = 0x200D
		keycap = 0x20E3 // combining enclosing keycap
	)
	out := make([]bool, len(runes))
	for i, r := range runes {
		out[i] = pictographic(r) || r == vs16
	}
	// The selectors, which decide the character before them.
	for i, r := range runes {
		switch r {
		case vs16:
			if i > 0 {
				out[i-1] = true
			}
		case vs15:
			// Asked for as text, which is the whole point of the
			// selector: neither it nor what it follows is a picture.
			out[i] = false
			if i > 0 {
				out[i-1] = false
			}
		}
	}
	// Then the joiners, which need both their neighbours *settled*: in
	// `👩‍❤️‍👩` the heart is U+2764, which is not from the pictographic
	// planes and is a picture only because the selector after it says
	// so. Asking whether the next rune was pictographic, rather than
	// whether it had been marked, left the joiner beside the heart
	// looking like a joiner between text — and the couple came apart
	// into three pictures (a video player measured it).
	for i, r := range runes {
		if r != zwj {
			continue
		}
		before := i > 0 && out[i-1]
		after := i+1 < len(runes) && out[i+1]
		out[i] = before && after
	}
	// And last the enclosing keycap, which takes the mark of what it
	// encloses: `1️⃣` is a digit, the selector and this.
	//
	// It was here, deleted as dead code when a mutation that removed
	// it changed no test, and the mutation survived because the test
	// installed nothing but the emoji face. Noto Sans Symbols has
	// U+20E3 and is in the built-in fallback list, so on a machine
	// that has it an *outline* face claimed the keycap and the digit
	// and the box drew side by side (a mail client and two players,
	// within the hour). An unmarked character is offered to the
	// outline fallback first; that is the half the earlier reasoning
	// missed.
	for i, r := range runes {
		if r != keycap || i == 0 {
			continue
		}
		// What the keycap encloses, which is the rune before it or the
		// one before an emoji selector. UTS #51 spells the sequence
		// out: ED-14c limits the base to a digit, a number sign or an
		// asterisk, and the optional U+FE0F between them is all that
		// separates the qualified form from the unqualified one.
		//
		// The *text* selector needs no case of its own, though it is
		// the other half of what was reported: U+FE0E is not a keycap
		// base either, so `1\uFE0E\u20E3` falls out at the same test
		// and stays the digit and combining mark it asked to be
		// (ED-8, and the rule the selector pass above already keeps).
		base := i - 1
		if runes[base] == vs16 {
			if base == 0 {
				continue
			}
			base--
		}
		if !keycapBase(runes[base]) {
			continue
		}
		// The keycap takes the mark of what it encloses — and gives
		// one back. `1\uFE0F\u20E3` is a digit, the selector and this,
		// and `1\u20E3` is the same sequence without the selector,
		// which UTS #51 calls unqualified and every emoji font still
		// composes. The digit is not a picture by itself, so marking
		// only from left to right left the unqualified form split in
		// two: a text digit from one face and an enclosing box from
		// another, tinted with the label (a music player, item 48).
		//
		// Marking it from *any* base is what the first version did, and
		// `A\u20E3` then went to the colour face, which has no A — a
		// tintable letter and an untinted cell side by side (the same
		// player, item 50, within the release). An enclosing mark on
		// anything else is ordinary combining text.
		for j := base; j <= i; j++ {
			out[j] = true
		}
	}
	return out
}

// pictographic says whether r is one of the characters that are emoji on
// their own, with no selector asked for.
//
// The planes, and not everything above them: the first version of this
// said `r >= 0x1F000`, which is true of U+20BB7 — a Han ideograph from
// plane 2, and of every other character in the two planes past the
// pictographs. A video player found it in a file name, `🎬𠮷野家.mp4`:
// the Han was marked as colour, the emoji run swallowed it, and the one
// face that has it was never asked (item: emoji run swallows a following
// Han fallback). U+1FAFF ends Symbols and Pictographs Extended-A; what
// follows it is Symbols for Legacy Computing, which is text.
func pictographic(r rune) bool {
	return r >= 0x1F000 && r <= 0x1FAFF
}

// keycapBase says whether r can be enclosed by U+20E3 to make an emoji
// keycap: UTS #51 ED-14c limits it to these twelve.
func keycapBase(r rune) bool {
	return r >= '0' && r <= '9' || r == '#' || r == '*'
}

// colour is the emoji face for r, remembered with its sheet.
func (m *uitkFontmap) colour(r rune) *gofont.Face {
	g, sheet := emojiSheetFor(m.f, r)
	if g == nil || sheet == nil {
		return nil
	}
	m.of[g] = sheet
	return g
}

// part is one shaped run: glyphs in visual order, from one face, at one
// bidi level.
type part struct {
	at     glyphSheet
	glyphs []shapedGlyph
	start  int // first rune of the run, in the whole string
	end    int
	level  bidi.Level
	width  float32
}

func complexLayout(f *Font, text string) (shapedRun, bool) {
	if f == nil || f.ot == nil || f.ot.face == nil || text == "" {
		return shapedRun{}, false
	}
	prim := f.ot.face.gotext()
	if prim == nil {
		return shapedRun{}, false
	}
	runes := []rune(text)
	base := paragraphDirection(runes)

	// The levels, for L2 below. The segmenter runs the same algorithm
	// over the same runes with the same default, so its run boundaries
	// and these agree.
	def := bidi.LeftToRight
	if base == di.DirectionRTL {
		def = bidi.RightToLeft
	}
	var para bidi.Paragraph
	brs := para.Segment(runes, def)
	levels := make([]bidi.Level, len(runes))
	for i := 0; i < brs.NumRuns(); i++ {
		r := brs.Run(i)
		for j := r.Start; j < r.End && j < len(levels); j++ {
			levels[j] = r.Level
		}
	}

	fm := &uitkFontmap{
		f: f, prim: prim,
		of:       map[*gofont.Face]glyphSheet{prim: f.ot},
		runes:    runes,
		colourAt: emojiPresentation(runes),
	}
	seg := segPool.Get().(*shaping.Segmenter)
	defer segPool.Put(seg)
	sh := shaperPool.Get().(*shaping.HarfbuzzShaper)
	defer shaperPool.Put(sh)

	in := shaping.Input{
		Text: runes, RunStart: 0, RunEnd: len(runes),
		Face: prim, Size: fixed.Int26_6(f.Size * 64),
		Direction: base,
	}
	var parts []part
	pieces := seg.Split(in, fm)
	faced := make([]shaping.Input, 0, len(pieces)+2)
	for _, piece := range pieces {
		faced = append(faced, fm.faceRuns(piece)...)
	}
	for _, piece := range faced {
		at := fm.of[piece.Face]
		if at == nil {
			at = f.ot
		}
		unlock := lockFace(piece.Face)
		out := sh.Shape(piece)
		unlock()
		p := part{at: at, start: piece.RunStart, end: piece.RunEnd}
		if piece.RunStart < len(levels) {
			p.level = levels[piece.RunStart]
		}
		// A face drawn heavier than it was cut spreads by that much,
		// and the shaper knows nothing about it: the outline is the
		// face's own and so is every advance in its tables.
		var emb float32
		if ot, ok := at.(*otAtlas); ok && ot.face != nil {
			emb = ot.face.embolden * ot.size
		}
		for _, g := range out.Glyphs {
			sg := shapedGlyph{
				GID:     uint32(g.GlyphID),
				Advance: fx32(g.Advance) + emb,
				X:       fx32(g.XOffset),
				Y:       -fx32(g.YOffset),
				Cluster: g.TextIndex(),
			}
			_, outlines := at.(*otAtlas)
			if outlines && g.GlyphID == 0 && sg.Cluster >= 0 && sg.Cluster < len(runes) {
				// Nothing in this face, and nothing lent: hand the
				// cluster back to the rune path, which has a drawn
				// star for '★' and a box for everything else.
				sg.GID, sg.byRune = uint32(runes[sg.Cluster]), true
			}
			p.glyphs = append(p.glyphs, sg)
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return shapedRun{}, false
	}
	reorderVisual(parts)

	// Bake every glyph, one draft per sheet, then place them.
	byAtlas := map[glyphSheet][]uint32{}
	byRune := map[*otAtlas][]rune{}
	for _, p := range parts {
		for _, g := range p.glyphs {
			if g.byRune {
				if ot, ok := p.at.(*otAtlas); ok {
					byRune[ot] = append(byRune[ot], rune(g.GID))
				}
				continue
			}
			byAtlas[p.at] = append(byAtlas[p.at], g.GID)
		}
	}
	sheets := make(map[glyphSheet]*paintengine2d.FontAtlas, len(byAtlas)+len(byRune))
	for at, gids := range byAtlas {
		sheets[at] = at.ensureGIDs(gids)
	}
	for at, rs := range byRune {
		sheets[at] = at.ensure(string(rs))
	}
	// A cluster handed back to the rune path takes that cell's advance:
	// a drawn star is as wide as symbols.go made it, and the shaper's
	// answer was the width of a glyph the face has not got.
	for pi := range parts {
		sheet := sheets[parts[pi].at]
		for gi := range parts[pi].glyphs {
			g := &parts[pi].glyphs[gi]
			if g.byRune && sheet != nil {
				if cell, ok := sheet.Cell(g.key()); ok {
					g.Advance = advanceOf(cell)
				}
			}
			parts[pi].width += g.Advance
		}
	}

	lay := runList{atlas: sheets[f.ot]}
	if lay.atlas == nil {
		lay.atlas = f.ot.atlas()
	}
	caret := make([]float32, len(runes)+1)
	ends := make([]float32, len(runes)+1)
	seen := make([]bool, len(runes)+1)
	var pen float32
	for _, p := range parts {
		sheet := sheets[p.at]
		rtl := p.level%2 != 0
		x := pen
		for i := 0; i < len(p.glyphs); {
			// One cluster: the glyphs that share a first rune.
			j := i + 1
			w := p.glyphs[i].Advance
			for j < len(p.glyphs) && p.glyphs[j].Cluster == p.glyphs[i].Cluster {
				w += p.glyphs[j].Advance
				j++
			}
			for k := i; k < j; k++ {
				g := p.glyphs[k]
				if sheet != nil {
					lay.add(sheet, paintengine2d.Glyph{ID: g.key(), X: x + g.X, Y: g.Y})
				}
				x += g.Advance
			}
			markCluster(caret, ends, seen, p, i, j, x-w, w, rtl)
			i = j
		}
		pen += p.width
	}
	fillCarets(caret, ends, seen, pen, base)

	s := lay.measure()
	s.epoch = fallbackEpoch.Load()
	s.adv, s.xs, s.ends = pen, caret, ends
	if s.ink < pen {
		s.ink = pen
	}
	return s, true
}

// markCluster records where the caret goes for every rune of one
// cluster. The caret sits at a character's *leading* edge, which is its
// left in a left-to-right run and its right in a right-to-left one, and
// a cluster of several runes — a ligature, a syllable — divides its
// width between them so a caret can still be put inside it.
func markCluster(caret, ends []float32, seen []bool, p part, i, j int, x, w float32, rtl bool) {
	first := p.glyphs[i].Cluster
	last := p.end
	// The cluster runs to the next cluster's first rune; within one run
	// the clusters are in visual order, so the neighbour is on whichever
	// side the direction says.
	if rtl {
		if i > 0 {
			last = p.glyphs[i-1].Cluster
		}
	} else {
		if j < len(p.glyphs) {
			last = p.glyphs[j].Cluster
		}
	}
	if last <= first {
		last = first + 1
	}
	n := last - first
	step := w / float32(n)
	for k := 0; k < n; k++ {
		idx := first + k
		if idx < 0 || idx >= len(caret) {
			continue
		}
		if rtl {
			caret[idx] = x + w - float32(k)*step
			ends[idx] = caret[idx] - step
		} else {
			caret[idx] = x + float32(k)*step
			ends[idx] = caret[idx] + step
		}
		seen[idx] = true
	}
}

// fillCarets gives the runes no cluster named — and the one past the
// end — a position, so that every index into the string has a caret.
func fillCarets(caret, ends []float32, seen []bool, total float32, base di.Direction) {
	end := total
	if base == di.DirectionRTL {
		end = 0
	}
	caret[len(caret)-1], ends[len(ends)-1] = end, end
	seen[len(seen)-1] = true
	last := float32(0)
	for i := range caret {
		if seen[i] {
			last = caret[i]
			continue
		}
		// A rune no cluster named — one the shaper folded into its
		// neighbour — sits where that neighbour does, with no width of
		// its own, so a caret can still be put on either side of it.
		caret[i], ends[i] = last, last
	}
}

// visualOrder is UAX #9 L2: from the highest level down to the lowest
// odd one, reverse every stretch at that level or above. It is what puts
// a Hebrew phrase inside an English sentence the right way round, and
// the quoted English inside *that* the right way round again.
//
// It answers with the order to draw the levels in — index 0 first —
// rather than moving anything, so that a caller holding runs, glyphs or
// styled fragments can put its own in that order.
func visualOrder(levels []bidi.Level) []int {
	order := make([]int, len(levels))
	for i := range order {
		order[i] = i
	}
	if len(levels) < 2 {
		return order
	}
	var highest bidi.Level
	lowestOdd := bidi.Level(127)
	for _, l := range levels {
		if l > highest {
			highest = l
		}
		if l%2 != 0 && l < lowestOdd {
			lowestOdd = l
		}
	}
	at := func(i int) bidi.Level { return levels[order[i]] }
	for lvl := highest; lvl >= lowestOdd; lvl-- {
		for i := 0; i < len(order); i++ {
			if at(i) < lvl {
				continue
			}
			j := i
			for j < len(order) && at(j) >= lvl {
				j++
			}
			for a, b := i, j-1; a < b; a, b = a+1, b-1 {
				order[a], order[b] = order[b], order[a]
			}
			i = j
		}
	}
	return order
}

// reorderVisual puts shaped runs in the order they are drawn.
func reorderVisual(parts []part) {
	levels := make([]bidi.Level, len(parts))
	for i, p := range parts {
		levels[i] = p.level
	}
	out := make([]part, len(parts))
	for i, from := range visualOrder(levels) {
		out[i] = parts[from]
	}
	copy(parts, out)
}
