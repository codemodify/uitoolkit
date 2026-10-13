//go:build font_complex

package style

import (
	"sync"

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

// uitkFontmap answers the segmenter's "which face draws this rune" with
// the toolkit's own answer: this font's face, or the first family in
// [FallbackFamilies] that has the rune. It remembers which sheet each
// face it handed out belongs to, since the glyphs come back named by
// face and have to be baked into that face's own sheet.
type uitkFontmap struct {
	f    *Font
	prim *gofont.Face
	of   map[*gofont.Face]*otAtlas
}

func (m *uitkFontmap) ResolveFace(r rune) *gofont.Face {
	if m.f.ot != nil && m.f.ot.face.hasGlyph(r) {
		return m.prim
	}
	if borrowable(r) {
		if fb := fallbackFor(m.f, r); fb != nil && fb.ot != nil {
			if g := fb.ot.face.gotext(); g != nil {
				m.of[g] = fb.ot
				return g
			}
		}
	}
	// Never nil: the sheet's .notdef box is the honest answer, and the
	// caller still needs a face to measure it with.
	return m.prim
}

// part is one shaped run: glyphs in visual order, from one face, at one
// bidi level.
type part struct {
	at     *otAtlas
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

	fm := &uitkFontmap{f: f, prim: prim, of: map[*gofont.Face]*otAtlas{prim: f.ot}}
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
	for _, piece := range seg.Split(in, fm) {
		at := fm.of[piece.Face]
		if at == nil {
			at = f.ot
		}
		out := sh.Shape(piece)
		p := part{at: at, start: piece.RunStart, end: piece.RunEnd}
		if piece.RunStart < len(levels) {
			p.level = levels[piece.RunStart]
		}
		// A face drawn heavier than it was cut spreads by that much,
		// and the shaper knows nothing about it: the outline is the
		// face's own and so is every advance in its tables.
		var emb float32
		if at.face != nil {
			emb = at.face.embolden * at.size
		}
		for _, g := range out.Glyphs {
			sg := shapedGlyph{
				GID:     uint32(g.GlyphID),
				Advance: fx32(g.Advance) + emb,
				X:       fx32(g.XOffset),
				Y:       -fx32(g.YOffset),
				Cluster: g.TextIndex(),
			}
			if g.GlyphID == 0 && sg.Cluster >= 0 && sg.Cluster < len(runes) {
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
	byAtlas := map[*otAtlas][]uint32{}
	byRune := map[*otAtlas][]rune{}
	for _, p := range parts {
		for _, g := range p.glyphs {
			if g.byRune {
				byRune[p.at] = append(byRune[p.at], rune(g.GID))
				continue
			}
			byAtlas[p.at] = append(byAtlas[p.at], g.GID)
		}
	}
	sheets := make(map[*otAtlas]*paintengine2d.FontAtlas, len(byAtlas)+len(byRune))
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

// reorderVisual is UAX #9 L2: from the highest level down to the lowest
// odd one, reverse every run of parts at that level or above. It is what
// puts a Hebrew phrase inside an English sentence the right way round,
// and the quoted English inside *that* the right way round again.
func reorderVisual(parts []part) {
	if len(parts) < 2 {
		return
	}
	var highest bidi.Level
	lowestOdd := bidi.Level(127)
	for _, p := range parts {
		if p.level > highest {
			highest = p.level
		}
		if p.level%2 != 0 && p.level < lowestOdd {
			lowestOdd = p.level
		}
	}
	for lvl := highest; lvl >= lowestOdd; lvl-- {
		for i := 0; i < len(parts); i++ {
			if parts[i].level < lvl {
				continue
			}
			j := i
			for j < len(parts) && parts[j].level >= lvl {
				j++
			}
			for a, b := i, j-1; a < b; a, b = a+1, b-1 {
				parts[a], parts[b] = parts[b], parts[a]
			}
			i = j
		}
	}
}
