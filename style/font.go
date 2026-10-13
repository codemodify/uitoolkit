package style

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Font is a baked glyph atlas painted through paintengine2d.
// Default UI/mono faces are OpenType outlines (Titillium Web / JetBrains Mono)
// rasterized with the engine's scanline AA into a white sheet; Paint.Color tints.
//
// A Font value is a cheap handle: BakeFamily hands out copies that share one
// copy-on-write glyph sheet (see [Font.GlyphAtlas]), so copies are safe to use
// from different goroutines.
type Font struct {
	Size    float32
	Ascent  float32
	Descent float32
	Color   paintengine2d.Color
	Family  string
	Weight  Weight
	Outline bool // true when built from TTF outlines (not the 5×7 bitmap)
	// static is the immutable sheet of a bitmap face (ot == nil).
	static *paintengine2d.FontAtlas
	ot     *otAtlas
	shaped *shapeCache
}

// GlyphAtlas is the face's current glyph sheet.
//
// The OpenType path publishes a new immutable sheet whenever a rune is baked,
// so the returned pointer is a snapshot: safe to read from any goroutine and
// never mutated after publication. Re-read it (do not cache it) after drawing
// text that may introduce new runes.
func (f *Font) GlyphAtlas() *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot != nil {
		if a := f.ot.atlas(); a != nil {
			return a
		}
	}
	return f.static
}

// atlasBytes is the live sheet size (font cache budget accounting).
func (f *Font) atlasBytes() int64 {
	if f == nil {
		return 0
	}
	if f.ot != nil {
		return f.ot.atlasBytes()
	}
	if f.static != nil && f.static.Image != nil {
		return int64(f.static.Image.Width) * int64(f.static.Image.Height) * 4
	}
	return 0
}

// shapeCacheCap drops shaped runs once the map grows past this. Labels
// and menu titles reuse the same strings; list rows with unique text miss.
const shapeCacheCap = 512

type shapeCache struct {
	mu sync.Mutex
	m  map[string]shapedRun
}

type shapedRun struct {
	// runs is the glyphs, grouped by the sheet they come from: one for
	// text a single face draws, which is nearly all text, and one more
	// for each face a fallback borrowed a glyph from. Positions are
	// absolute within the run, so the groups may be drawn in any order.
	runs []paintengine2d.GlyphRun
	// epoch is the fallback list this was laid out against. A change to
	// that list makes the layout stale — a rune that was a box may now
	// be borrowed, and one borrowed from a face no longer named must go
	// back to being a box — and a cache that did not notice kept both.
	epoch uint64
	adv   float32
	ink   float32
	// xs is where each rune starts, kerning included, then where the run
	// ends: carets, hit-testing and eliding read the layout Draw paints.
	//
	// "Starts" is the rune's *leading* edge, which is its left in text
	// that reads left to right and its right in text that does not, so
	// xs does not have to climb.
	xs []float32
	// ends is the other edge of each rune, and is nil for a layout that
	// reads one way: then a rune ends where the next one starts, and
	// xs[i+1] is that. A shaped layout fills it, because a right-to-left
	// run's runes end to the left of where they start and a line can
	// turn round in the middle.
	ends []float32
}

// span is the x range rune i covers, lowest first.
func (s shapedRun) span(i int) (lo, hi float32) {
	if i < 0 || i+1 >= len(s.xs) {
		return 0, 0
	}
	a, b := s.xs[i], s.xs[i+1]
	if s.ends != nil {
		b = s.ends[i]
	}
	if a > b {
		return b, a
	}
	return a, b
}

// oneWay reports that the layout reads in one direction throughout, so
// that x climbs with the index and a binary search is meaningful.
func (s shapedRun) oneWay() bool { return s.ends == nil }

func newShapeCache() *shapeCache {
	return &shapeCache{m: make(map[string]shapedRun, 32)}
}

// wrapTab is what [Font.Wrap] gives a tab: four spaces. A tab stop is a
// column, and a column is a text editor's business; a paragraph painter
// needs the tab to take room and to be somewhere a line may break, and the
// lines Wrap returns have to be drawable as they stand. The face's own '\t'
// is whatever its notdef is — 8.8px and a box in Titillium Web — so leaving
// it in the line would measure one thing and paint another.
const wrapTab = "    "

// Prefix is the longest prefix of text that fits maxW, with nothing added.
//
// It is [Font.Fit] without the ellipsis: Fit answers "what should I show
// instead of this", which is the wrong question for a wrapper, a marquee or
// a measured column, all of which need "how much of this fits". Everything
// that wanted the second had to reimplement the rune scan Fit already does.
//
// The measure is the shaped layout Draw paints, pair kerning included, so
// the prefix that comes back is the one that fits to the pixel.
func (f *Font) Prefix(text string, maxW float32) string {
	if f == nil || text == "" || maxW <= 0 {
		return ""
	}
	if f.Advance(text) <= maxW {
		return text
	}
	xs := f.shapeOf(text).xs
	n := 0
	for n+1 < len(xs) && xs[n+1] <= maxW {
		n++
	}
	if n <= 0 {
		return ""
	}
	return string([]rune(text)[:n])
}

// Wrap breaks text into lines no wider than maxW.
//
// It is the greedy word wrap every application that paints a paragraph was
// writing for itself: breaks at spaces, keeps the newlines the text already
// has (a "\r\n" counts as one), and splits a word too long for the line
// rather than letting it run out of the box. Tabs become [wrapTab] spaces
// first, so the lines that come back are ready to draw as they are. Empty
// text gives no lines; an empty line between two paragraphs is kept.
//
// The scan costs one shaped rune per distinct rune, not one shaped prefix
// per rune: measuring a wrap with Advance over growing prefixes is
// quadratic and flushes the shared shape cache for any paragraph longer
// than it. The trade is that the scan adds rune advances and so does not
// see pair kerning ("AV", "To"), which makes a line measure a hair wider
// than it paints — the safe direction, and the same measure the caret uses.
func (f *Font) Wrap(text string, maxW float32) []string {
	if f == nil || text == "" {
		return nil
	}
	if maxW < 1 {
		maxW = 1
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", wrapTab)
	adv := f.runeAdvances()
	exact := func(runes []rune) float32 { return f.Advance(string(runes)) }
	var out []string
	for _, para := range strings.Split(text, "\n") {
		out = append(out, wrapLine([]rune(para), maxW, adv, exact)...)
	}
	return out
}

// CanBreakAfter reports whether a word too long for its line may be broken
// after r.
//
// It is the second choice, for a "word" that is really a path, a URL or a
// key: a file name broken after its last slash still reads as a file name,
// where one broken in the middle of it reads as two words that are not
// there. Text that has a space to break at breaks there instead, and a word
// with none of these in it is still broken wherever the line ends, because
// something has to give.
//
// Exported because two wrappers share the rule — this one, for labels, and
// the one inside [TextArea] — and a policy written twice is a policy that
// drifts.
func CanBreakAfter(r rune) bool {
	switch r {
	case '/', '\\', '-', '.':
		return true
	}
	return false
}

// wrapLine is the greedy pass over one paragraph (no newlines left in it).
//
// It adds rune advances, which is cheap and *over*-counts: a sum of
// advances does not see the pair kerning the text is painted with, so it
// says a line is wider than it will be. That is the safe direction for
// deciding a line is full, and the wrong one for deciding it is too
// full — a label measured at exactly its own [Font.Advance] width wrapped
// anyway, and every wrapping label in a form was therefore a line taller
// than the line it drew.
//
// So the cheap sum decides when to *ask*, and exact decides whether to
// break. When the exact measure says the line still fits, the running
// sum is resynced to it: the kerning slack has to accumulate again
// before the next question, which keeps the exact measures to a handful
// per line rather than one per rune.
func wrapLine(runes []rune, maxW float32, adv func(rune) float32, exact func([]rune) float32) []string {
	if len(runes) == 0 {
		return []string{""}
	}
	var out []string
	start, brk := 0, -1 // brk is the space the line would break back to
	// sep is where the line would break *within* a word too long for it:
	// just after the last path or URL separator it has passed. Used only
	// when there is no space to break at.
	sep := -1
	var w float32
	for i := 0; i < len(runes); i++ {
		// The break point is the *first* space of a run, so a line that
		// breaks after two spaces does not keep the second one hanging
		// off its end.
		if runes[i] == ' ' && i > start && runes[i-1] != ' ' {
			brk = i
		}
		// Recorded for the rune *before* this one, which is known to be on
		// the line: a separator that is itself the rune that overflows is no
		// break point, and taking it as one would throw away the last good
		// one — a path then broke in the middle of a name with a slash two
		// characters back, and a hyphenated word left a line holding nothing
		// but "-".
		if i > start && CanBreakAfter(runes[i-1]) {
			sep = i
		}
		w += adv(runes[i])
		// The first rune of a line always stays on it: a single glyph
		// wider than the whole line would otherwise never be placed.
		if w <= maxW || i == start {
			continue
		}
		if exact != nil {
			if ex := exact(runes[start : i+1]); ex <= maxW {
				w = ex
				continue
			}
		}
		cut, next := i, i // a word longer than the line breaks mid-word
		switch {
		case brk > start:
			cut, next = brk, brk+1
			for next < len(runes) && runes[next] == ' ' {
				next++
			}
		case sep > start:
			// No space on the line: break after the separator rather than
			// in the middle of a name.
			cut, next = sep, sep
		}
		out = append(out, string(runes[start:cut]))
		start, brk, sep, w = next, -1, -1, 0
		i = next - 1
	}
	if start < len(runes) || len(out) == 0 {
		out = append(out, string(runes[start:]))
	}
	return out
}

// runeAdvances is a per-call rune → advance memo over one face. Callers
// wrap a paragraph with it and throw it away; nothing shared is touched.
func (f *Font) runeAdvances() func(rune) float32 {
	table := make(map[rune]float32, 64)
	return func(r rune) float32 {
		if w, ok := table[r]; ok {
			return w
		}
		w := f.CaretX(string(r), 1)
		table[r] = w
		return w
	}
}

// Fit returns text, or a prefix plus an ellipsis, that fits in maxW.
// fitSlack is how much a text may overrun the width it is being fitted
// to and still count as fitting: a thirty-second of a pixel.
//
// Measuring and then fitting is the common pattern — a widget measures
// its words, is given a box that holds them, and fits them into what is
// left of it — and float32 does not survive the round trip. A step of a
// breadcrumb row some hundreds of pixels along was given 57.046844 for
// words whose advance is 57.046875, so the words it was measured to
// hold came out with an ellipsis on them (secretvault item 82, and the
// same arithmetic as the whole-pixel rounding layout already does).
//
// A thirty-second of a pixel is far below anything that can be seen and
// far above the error, which is parts in a million.
const fitSlack = 1.0 / 32

func (f *Font) Fit(text string, maxW float32) string {
	if f == nil || text == "" {
		return ""
	}
	if maxW <= 0 {
		return ""
	}
	if f.Advance(text) <= maxW+fitSlack {
		return text
	}
	const ell = "…"
	ew := f.Advance(ell)
	if maxW <= ew {
		return ell
	}
	head := f.Prefix(text, maxW-ew)
	if head == "" {
		return ell
	}
	return head + ell
}

func (f *Font) Measure(text string) paintengine2d.Point {
	if f == nil {
		return paintengine2d.Point{}
	}
	return paintengine2d.Pt(f.Advance(text), f.Height())
}

func (f *Font) Height() float32 {
	if f == nil {
		return 0
	}
	if f.Descent > 0 {
		return f.Ascent + f.Descent
	}
	return f.Size + 2
}

func (f *Font) Advance(text string) float32 {
	if f == nil {
		return 0
	}
	return f.shapeOf(text).adv
}

// InkWidth is the painted AABB width of text (advance plus last-glyph
// bearing / atlas pad). Menus use this so DrawGlyphs cannot clip a stem.
func (f *Font) InkWidth(text string) float32 {
	if f == nil {
		return 0
	}
	if text == "" {
		return 0
	}
	return f.shapeOf(text).ink
}

// IndexAt is the rune index a click at x asks for: the caret position
// nearest the point, which is before or after a character depending on
// which half of it was hit.
func (f *Font) IndexAt(text string, x float32) int {
	if f == nil || text == "" {
		return 0
	}
	s := f.shapeOf(text)
	if s.oneWay() {
		if x <= 0 {
			return 0
		}
		xs := s.xs
		for i := 0; i+1 < len(xs); i++ {
			if x < (xs[i]+xs[i+1])*0.5 {
				return i
			}
		}
		return max(len(xs)-1, 0)
	}
	return s.indexAtVisual(x)
}

// indexAtVisual is IndexAt for a line that turns round: the index cannot
// be found by walking until x is passed, because x does not only grow.
// The character the point is *on* decides, and its own direction says
// whether the caret goes before or after it.
func (s shapedRun) indexAtVisual(x float32) int {
	n := len(s.xs) - 1
	best, bestD := 0, float32(math.MaxFloat32)
	note := func(at float32, i int) {
		d := at - x
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, best = d, i
		}
	}
	for i := 0; i < n; i++ {
		lo, hi := s.span(i)
		if x >= lo && x <= hi {
			lead := s.xs[i]
			if abs32(x-lead) <= abs32(x-s.ends[i]) {
				return i
			}
			return i + 1
		}
		note(s.xs[i], i)
		note(s.ends[i], i+1)
	}
	note(s.xs[n], n)
	return best
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// SelectionSpans is the x ranges a selection of the runes [i, j) covers,
// for a painter to fill behind the text.
//
// One range for text that reads one way, which is nearly all text. More
// where the selection crosses a change of direction: a Hebrew phrase
// selected together with the English on either side of it is drawn as
// one stretch on screen, but the English before it and the English after
// it are not next to each other, and a single box from the first
// character to the last would cover the wrong half of the line.
func (f *Font) SelectionSpans(text string, i, j int) [][2]float32 {
	if f == nil || text == "" || i == j {
		return nil
	}
	if i > j {
		i, j = j, i
	}
	s := f.shapeOf(text)
	n := len(s.xs) - 1
	if n <= 0 {
		return nil
	}
	i, j = max(i, 0), min(j, n)
	if i >= j {
		return nil
	}
	if s.oneWay() {
		return [][2]float32{{s.xs[i], s.xs[j]}}
	}
	// Collected in logical order, which is not the order they are drawn
	// in: the first character of a right-to-left run is the rightmost of
	// it, so the pieces arrive scattered and have to be sorted before
	// the ones that touch can be joined.
	out := make([][2]float32, 0, j-i)
	for k := i; k < j; k++ {
		lo, hi := s.span(k)
		out = append(out, [2]float32{lo, hi})
	}
	slices.SortFunc(out, func(a, b [2]float32) int {
		switch {
		case a[0] < b[0]:
			return -1
		case a[0] > b[0]:
			return 1
		}
		return 0
	})
	merged := out[:0]
	for _, sp := range out {
		if m := len(merged); m > 0 && sp[0] <= merged[m-1][1]+0.01 {
			merged[m-1][1] = max(merged[m-1][1], sp[1])
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

// cellFor is the atlas cell for r, baking it on first use.
func (f *Font) cellFor(r rune) (paintengine2d.AtlasCell, bool) {
	if f == nil {
		return paintengine2d.AtlasCell{}, false
	}
	atlas := f.GlyphAtlas()
	if atlas == nil {
		return paintengine2d.AtlasCell{}, false
	}
	if c, ok := atlas.Cell(paintengine2d.GlyphID(r)); ok {
		return c, true
	}
	if f.ot == nil {
		return paintengine2d.AtlasCell{}, false
	}
	return f.ot.ensureRune(r).Cell(paintengine2d.GlyphID(r))
}

func (f *Font) runeAdvance(r rune) float32 {
	cell, ok := f.cellFor(r)
	if !ok {
		return 0
	}
	if cell.Advance > 0 {
		return cell.Advance
	}
	return cell.Src.Dx()
}

func (f *Font) CaretX(text string, i int) float32 {
	if f == nil || i <= 0 || text == "" {
		return 0
	}
	xs := f.shapeOf(text).xs
	if i >= len(xs) {
		i = len(xs) - 1
	}
	return xs[i]
}

// ensure bakes every rune of text and returns the sheet to shape against.
func (f *Font) ensure(text string) *paintengine2d.FontAtlas {
	if f == nil {
		return nil
	}
	if f.ot == nil {
		return f.static
	}
	return f.ot.ensure(text)
}

func (f *Font) shapeOf(text string) shapedRun {
	if f == nil || text == "" {
		return shapedRun{}
	}
	// The warm path, before anything walks the string. ensure asks the
	// published sheet whether it holds every rune of text, which is a map
	// lookup a rune even when the run was shaped a thousand frames ago —
	// and Advance is what the whole layout pass is made of. A run cached
	// against the sheet that is still published needs none of it: cells
	// are only ever added, so every glyph that run names is still there.
	if f.shaped != nil {
		if cur := f.GlyphAtlas(); cur != nil {
			f.shaped.mu.Lock()
			hit, ok := f.shaped.m[text]
			f.shaped.mu.Unlock()
			if ok && hit.primary() == cur && hit.epoch == fallbackEpoch.Load() {
				return hit
			}
		}
	}
	// Shaped first, where the build has a shaper: it chooses the glyphs
	// from the face's own tables and bakes them by index, so the sheet
	// never holds the rune-keyed cells the rune path would have baked.
	shape := func() shapedRun {
		if s, ok := complexLayout(f, text); ok {
			return s
		}
		atlas := f.ensure(text)
		if atlas == nil {
			return shapedRun{}
		}
		return f.layout(text, atlas)
	}
	if f.shaped == nil {
		return shape()
	}
	f.shaped.mu.Lock()
	defer f.shaped.mu.Unlock()
	if hit, ok := f.shaped.m[text]; ok && hit.epoch == fallbackEpoch.Load() {
		// Re-point a run shaped against an earlier snapshot: cells are only
		// ever added, so the positions still hold, and the superseded sheet
		// stops being pinned by the cache. Only this font's own sheet —
		// a borrowed one belongs to the face it came from, which
		// republishes on its own account.
		if cur := f.GlyphAtlas(); cur != nil && hit.primary() != cur && len(hit.runs) > 0 {
			hit.runs[0].Atlas = cur
			f.shaped.m[text] = hit
		}
		return hit
	}
	hit := shape()
	if f.shaped.m == nil {
		f.shaped.m = make(map[string]shapedRun, 32)
	}
	if len(f.shaped.m) >= shapeCacheCap {
		clear(f.shaped.m)
	}
	f.shaped.m[text] = hit
	return hit
}

// layout places each rune of text at the sum of the advances before it,
// with the face's pair kerning ("AV", "To") between neighbours, then snaps
// the glyphs onto the pixel grid.
func (f *Font) layout(text string, atlas *paintengine2d.FontAtlas) shapedRun {
	lay := runList{atlas: atlas}
	xs := make([]float32, 0, len(text)+1)
	var x float32
	prev := rune(-1)
	for _, r := range text {
		id := paintengine2d.GlyphID(r)
		cell, ok := atlas.Cell(id)
		if ok && prev >= 0 && f.ot != nil {
			x += f.ot.kern(prev, r)
		}
		// A rune this face has not got, in a build that may borrow one.
		// The box it would otherwise draw is what makes two different
		// CJK titles the same row of pixels.
		if !ok || f.isNotdef(cell) {
			if fb := fallbackFor(f, r); fb != nil {
				if fa := fb.atlasForRune(r); fa != nil {
					if c2, ok2 := fa.Cell(id); ok2 {
						xs = append(xs, x)
						lay.add(fa, paintengine2d.Glyph{ID: id, X: x})
						x += advanceOf(c2)
						// No kerning across a borrowed glyph: the
						// pair is not in either face's table.
						prev = -1
						continue
					}
				}
			}
		}
		xs = append(xs, x)
		if !ok {
			prev = -1
			continue
		}
		lay.add(atlas, paintengine2d.Glyph{ID: id, X: x})
		x += advanceOf(cell)
		prev = r
	}
	xs = append(xs, x)
	s := lay.measure()
	s.epoch = fallbackEpoch.Load()
	// The advance is where the layout ends; snapping each glyph onto the
	// pixel grid moves ink, not the layout.
	s.adv, s.xs = x, xs
	if s.ink < x {
		s.ink = x
	}
	return s
}

// advanceOf is how far a cell moves the pen.
func advanceOf(cell paintengine2d.AtlasCell) float32 {
	if cell.Advance > 0 {
		return cell.Advance
	}
	return cell.Src.Dx() + 1
}

// isNotdef reports that a cell is the sheet's shared "no such glyph"
// box rather than a glyph of its own.
func (f *Font) isNotdef(cell paintengine2d.AtlasCell) bool {
	return f != nil && f.ot != nil && f.ot.hasNotdef && cell.Src == f.ot.notdef.Src
}

// runList collects the glyphs of one string, grouped by the sheet each
// came from. The first group is always the font's own, so the shape
// cache has something stable to check itself against; a second appears
// only where a glyph was borrowed.
type runList struct {
	atlas *paintengine2d.FontAtlas
	runs  []paintengine2d.GlyphRun
}

func (l *runList) add(atlas *paintengine2d.FontAtlas, g paintengine2d.Glyph) {
	for i := range l.runs {
		if l.runs[i].Atlas == atlas {
			l.runs[i].Glyphs = append(l.runs[i].Glyphs, g)
			return
		}
	}
	if len(l.runs) == 0 && atlas != l.atlas {
		// Keep the font's own sheet first even when the string opens
		// with a borrowed glyph.
		l.runs = append(l.runs, paintengine2d.GlyphRun{Atlas: l.atlas})
	}
	l.runs = append(l.runs, paintengine2d.GlyphRun{Atlas: atlas, Glyphs: []paintengine2d.Glyph{g}})
}

// measure snaps every group onto the pixel grid and takes the widest ink.
func (l *runList) measure() shapedRun {
	if len(l.runs) == 0 {
		l.runs = append(l.runs, paintengine2d.GlyphRun{Atlas: l.atlas})
	}
	out := shapedRun{runs: make([]paintengine2d.GlyphRun, 0, len(l.runs))}
	for _, run := range l.runs {
		one := measureShaped(snapRun(run, run.Atlas), run.Atlas)
		out.runs = append(out.runs, one.runs...)
		if one.adv > out.adv {
			out.adv = one.adv
		}
		if one.ink > out.ink {
			out.ink = one.ink
		}
	}
	return out
}

// snapRun rounds every glyph onto the pixel grid.
//
// Cells are rasterized with a fractional bearing and NullShaper accumulates
// fractional advances, so an unsnapped run lands the atlas blit between
// pixels: the engine then resamples the sheet (a 16px stem measured 255/2px
// crisp drops to ~191/3px) and loses its fast 1:1 path. Rounding the sum of
// position and bearing keeps every glyph within half a pixel of its ideal
// spot and makes the blit integer-aligned.
func snapRun(run paintengine2d.GlyphRun, atlas *paintengine2d.FontAtlas) paintengine2d.GlyphRun {
	if len(run.Glyphs) == 0 || atlas == nil {
		return run
	}
	out := run
	out.Glyphs = make([]paintengine2d.Glyph, len(run.Glyphs))
	for i, g := range run.Glyphs {
		if cell, ok := atlas.Cell(g.ID); ok {
			g.X = snapCoord(g.X+cell.Bearing.X) - cell.Bearing.X
			g.Y = snapCoord(g.Y+cell.Bearing.Y) - cell.Bearing.Y
		}
		out.Glyphs[i] = g
	}
	return out
}

func measureShaped(run paintengine2d.GlyphRun, atlas *paintengine2d.FontAtlas) shapedRun {
	var w float32
	for _, g := range run.Glyphs {
		if atlas == nil {
			break
		}
		cell, ok := atlas.Cell(g.ID)
		if !ok {
			continue
		}
		adv := cell.Advance
		if adv <= 0 {
			adv = cell.Src.Dx()
		}
		if g.X+adv > w {
			w = g.X + adv
		}
	}
	inkW := w
	ink := run.Bounds(paintengine2d.Pt(0, 0))
	if !ink.Empty() && ink.Max.X > w {
		inkW = ink.Max.X
	}
	return shapedRun{runs: []paintengine2d.GlyphRun{run}, adv: w, ink: inkW}
}

// Draw paints text at origin (baseline-top of the em box) tinted with col.
//
// Under a pure translation the origin is snapped to whole device pixels and
// the sheet is blitted 1:1 with FilterNearest — crisp stems and the engine's
// fast blit. A scaled or rotated context resamples, so it keeps bilinear.
func (f *Font) Draw(ctx *paintengine2d.Context, text string, origin paintengine2d.Point, tint paintengine2d.Color) {
	if f == nil || ctx == nil || text == "" {
		return
	}
	shaped := f.shapeOf(text)
	if shaped.empty() {
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
	paint := paintengine2d.Paint{Color: col, Filter: filter}
	for _, run := range shaped.runs {
		if len(run.Glyphs) == 0 {
			continue
		}
		ctx.DrawGlyphs(run, origin, paint)
	}
}

// primary is the sheet this font's own glyphs came from, which is the
// one the shape cache is checked against.
func (s shapedRun) primary() *paintengine2d.FontAtlas {
	if len(s.runs) == 0 {
		return nil
	}
	return s.runs[0].Atlas
}

// empty reports that there is nothing to draw.
func (s shapedRun) empty() bool {
	for _, r := range s.runs {
		if len(r.Glyphs) > 0 {
			return false
		}
	}
	return true
}

type fontKey struct {
	family string
	weight Weight
	size   int
	// ascent is set only for a face baked to sit on another face's
	// baseline (a fallback, style/fontfallback_on.go); zero is the
	// face's own, which is every ordinary bake.
	ascent int
}

// fontCacheBudget caps the glyph sheets the size cache keeps alive. Sheets
// grow as new runes are baked, so the budget is re-checked on every bake and
// the least recently used faces are dropped (a Font already handed out stays
// valid — only the cache entry goes).
//
// A var, not a const, so tests can shrink it.
var fontCacheBudget int64 = 48 << 20

type fontCacheEntry struct {
	font *Font
	used uint64
}

var fontCache = struct {
	mu   sync.Mutex
	m    map[fontKey]*fontCacheEntry
	tick uint64
}{m: map[fontKey]*fontCacheEntry{}}

func cachedFont(key fontKey) *Font {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	e, ok := fontCache.m[key]
	if !ok {
		return nil
	}
	fontCache.tick++
	e.used = fontCache.tick
	return e.font
}

// storeFont publishes f and returns the face callers should use: a
// concurrent bake of the same key wins so every caller shares one sheet.
func storeFont(key fontKey, f *Font) *Font {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	fontCache.tick++
	if e, ok := fontCache.m[key]; ok {
		e.used = fontCache.tick
		return e.font
	}
	fontCache.m[key] = &fontCacheEntry{font: f, used: fontCache.tick}
	evictFontsLocked(key)
	return f
}

// evictFontsLocked drops least-recently-used faces until the live sheets fit
// the budget. keep is never evicted (it was just requested).
func evictFontsLocked(keep fontKey) {
	total := int64(0)
	for _, e := range fontCache.m {
		total += e.font.atlasBytes()
	}
	for total > fontCacheBudget && len(fontCache.m) > 1 {
		var (
			victim fontKey
			oldest uint64
			found  bool
		)
		for k, e := range fontCache.m {
			if k == keep {
				continue
			}
			if !found || e.used < oldest {
				oldest, victim, found = e.used, k, true
			}
		}
		if !found {
			break
		}
		total -= fontCache.m[victim].font.atlasBytes()
		delete(fontCache.m, victim)
	}
}

// fontCacheStats is the live cache size (entries, glyph sheet bytes).
func fontCacheStats() (entries int, bytes int64) {
	fontCache.mu.Lock()
	defer fontCache.mu.Unlock()
	for _, e := range fontCache.m {
		bytes += e.font.atlasBytes()
	}
	return len(fontCache.m), bytes
}

// BakeFont builds the default UI face (Titillium Web Regular) at size.
// Glyphs are OpenType outlines rasterized through paintengine2d — not the
// 5×7 bitmap atlas. Theme color is applied at draw time via Paint.Color.
func BakeFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyUI, WeightRegular, size, col)
}

// BakeTitleFont is Titillium Web Bold at size.
func BakeTitleFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyUI, WeightBold, size, col)
}

// BakeMonoFont is JetBrains Mono Regular at size (LookAndFeel mono role).
func BakeMonoFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyMono, WeightRegular, size, col)
}

// BakeMonoBoldFont is JetBrains Mono Bold at size.
func BakeMonoBoldFont(size float32, col paintengine2d.Color) *Font {
	return BakeFamily(FamilyMono, WeightBold, size, col)
}

// BakeBitmapFont is the legacy 5×7 nearest-neighbor atlas. Default UI and
// mono faces never use this when the bundled OFL TTFs are present.
func BakeBitmapFont(scale int, col paintengine2d.Color) *Font {
	if scale < 1 {
		scale = 1
	}
	f := bakeScaled(scale, col)
	f.Family = "bitmap"
	f.Outline = false
	return f
}

// BakeFamily rasterizes a bundled OFL face. It panics only if the embedded
// TTF is missing or unloadable — that is a packaging bug, not a runtime
// fallback to the bitmap atlas.
func BakeFamily(family string, weight Weight, size float32, col paintengine2d.Color) *Font {
	if size < 8 {
		size = 8
	}
	key := fontKey{family: family, weight: weight, size: int(size*4 + 0.5)}
	if cached := cachedFont(key); cached != nil {
		f := *cached
		f.Color = col
		return &f
	}
	f, err := bakeOutline(family, weight, size)
	if err != nil {
		panic(err)
	}
	out := *storeFont(key, f)
	out.Color = col
	return &out
}

func bakeOutline(family string, weight Weight, size float32) (*Font, error) {
	f, err := bakeFace(family, weight, size)
	if err == nil {
		return f, nil
	}
	// An installed face that cannot draw the interface is not one the
	// interface can be drawn in, and the answer is the same as for a
	// family that is not installed at all: the bundled face stands in.
	//
	// The face this catches is a script-only one — Noto Sans Arabic,
	// Noto Sans CJK, Noto Color Emoji — which has no Latin 'A' and so
	// cannot draw a menu, a button or its own name. Thirty of them are
	// in the chooser's list on an ordinary desktop, the choice is saved
	// in look.json, and this used to *panic*: picking one made every
	// uitoolkit application on the machine crash on start, and keep
	// crashing, because the setting outlived the process.
	//
	// It is not the toolkit's place to hide those families from the
	// chooser — a face is a face, and the list is deliberately
	// unfiltered (ListFontFamilies) — but it is very much its place not
	// to die of one.
	if family == "" || bundledFamily(family) == FamilyUI || bundledFamily(family) == FamilyMono {
		return nil, err
	}
	return bakeFace("", weight, size)
}

// bakeFace bakes exactly the family asked for, with no fallback.
func bakeFace(family string, weight Weight, size float32) (*Font, error) {
	return bakeFaceAscent(family, weight, size, 0)
}

// bakeFaceAscent is bakeFace with the baseline it must sit on, or 0 for
// the face's own.
//
// A glyph's vertical bearing is baked as `ascent + ymin`, measured from
// the top of the line box, so a face lending a glyph to another face has
// to be baked against *that* face's ascent or its glyphs hang below the
// line they are on. Nothing else wants this.
//
// A baseline is also what says this face is a *lender*, which is the
// one case where the Latin check below does not apply.
func bakeFaceAscent(family string, weight Weight, size, baseline float32) (*Font, error) {
	face, err := faceFor(family, weight)
	if err != nil {
		return nil, err
	}
	ascent, descent := face.metrics(size)
	if baseline > 0 {
		ascent = baseline
	}
	ot := newOTAtlas(face, size)
	ot.ascent = ascent
	atlas, err := ot.bake(otPreload)
	if err != nil {
		return nil, err
	}
	// A face that cannot draw Latin cannot draw the interface, so one
	// asked for by name falls through to the next on the list rather
	// than making an unreadable window. It is the wrong test for a face
	// that is only lending a glyph — Noto Emoji, Noto Sans Symbols, a
	// script face — which has to cover one rune and nothing else, and
	// the check kept every one of them from ever lending (comms-mail
	// item 58).
	if baseline == 0 {
		cell, ok := atlas.Cell(paintengine2d.GlyphID('A'))
		if !ok || cell.Src.Empty() || (ot.hasNotdef && cell.Src == ot.notdef.Src) {
			return nil, fmt.Errorf("style: %s cannot draw Latin", family)
		}
	}
	return &Font{
		Size:    size,
		Ascent:  ascent,
		Descent: descent,
		Family:  family,
		Weight:  weight,
		Outline: true,
		static:  atlas,
		ot:      ot,
		shaped:  newShapeCache(),
	}, nil
}

func bakeScaled(scale int, col paintengine2d.Color) *Font {
	base := paintengine2d.NewBitmapAtlas(col)
	extra := extraPunctAtlas(col)
	// Merge extra cells into a combined source by listing ids.
	ids := make([]paintengine2d.GlyphID, 0, 80)
	seen := map[paintengine2d.GlyphID]bool{}
	for id := range base.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for id := range extra.Cells {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	srcW, srcH := 6, 8
	dstW, dstH := srcW*scale, srcH*scale
	const cols = 16
	rows := (len(ids) + cols - 1) / cols
	img := paintengine2d.NewImage(cols*dstW, rows*dstH)
	ctx := paintengine2d.NewContext(img)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, len(ids))
	for i, id := range ids {
		cell, ok := base.Cell(id)
		srcImg := base.Image
		if !ok {
			cell, ok = extra.Cell(id)
			srcImg = extra.Image
			if !ok {
				continue
			}
		}
		colI := i % cols
		rowI := i / cols
		dx := float32(colI * dstW)
		dy := float32(rowI * dstH)
		dst := paintengine2d.XYWH(dx, dy, float32(dstW), float32(dstH))
		ctx.DrawImageRectPaint(srcImg, cell.Src, dst, paintengine2d.Paint{
			Color:  paintengine2d.White,
			Filter: paintengine2d.FilterNearest,
		})
		adv := float32(dstW) + 1
		cells[id] = paintengine2d.AtlasCell{
			Src:     dst,
			Advance: adv,
		}
	}
	return &Font{
		static:  &paintengine2d.FontAtlas{Image: img, Cells: cells},
		Size:    float32(dstH),
		Ascent:  float32(7 * scale),
		Descent: float32(scale + 2),
		Color:   col,
		shaped:  newShapeCache(),
	}
}

// extraPunctAtlas is a tiny 5×7 sheet for punctuation the engine atlas omits.
func extraPunctAtlas(fg paintengine2d.Color) *paintengine2d.FontAtlas {
	type bits [7]byte
	extra := map[rune]bits{
		',':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x18},
		';':  {0x00, 0x0C, 0x0C, 0x00, 0x0C, 0x0C, 0x18},
		'\'': {0x0C, 0x0C, 0x08, 0x00, 0x00, 0x00, 0x00},
		'"':  {0x1B, 0x1B, 0x12, 0x00, 0x00, 0x00, 0x00},
		'@':  {0x0E, 0x11, 0x17, 0x15, 0x17, 0x10, 0x0E},
		'#':  {0x0A, 0x1F, 0x0A, 0x0A, 0x1F, 0x0A, 0x00},
		'[':  {0x1C, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1C},
		']':  {0x07, 0x01, 0x01, 0x01, 0x01, 0x01, 0x07},
		'{':  {0x06, 0x08, 0x08, 0x10, 0x08, 0x08, 0x06},
		'}':  {0x0C, 0x02, 0x02, 0x01, 0x02, 0x02, 0x0C},
		'<':  {0x02, 0x04, 0x08, 0x10, 0x08, 0x04, 0x02},
		'>':  {0x08, 0x04, 0x02, 0x01, 0x02, 0x04, 0x08},
		'|':  {0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
		'\\': {0x10, 0x10, 0x08, 0x04, 0x04, 0x02, 0x01},
		'*':  {0x00, 0x15, 0x0E, 0x04, 0x0E, 0x15, 0x00},
		'&':  {0x0C, 0x12, 0x14, 0x08, 0x15, 0x12, 0x0D},
		'`':  {0x08, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		'~':  {0x00, 0x00, 0x09, 0x16, 0x00, 0x00, 0x00},
	}
	const cw, ch, cols = 6, 8, 8
	n := len(extra)
	rows := (n + cols - 1) / cols
	img := paintengine2d.NewImage(cols*cw, rows*ch)
	cells := make(map[paintengine2d.GlyphID]paintengine2d.AtlasCell, n)
	i := 0
	for r, b := range extra {
		col := i % cols
		row := i / cols
		ox, oy := col*cw, row*ch
		for y := 0; y < 7; y++ {
			rowBits := b[y]
			for x := 0; x < 5; x++ {
				if rowBits&(1<<uint(4-x)) != 0 {
					img.SetColor(ox+x, oy+y, fg)
				}
			}
		}
		cells[paintengine2d.GlyphID(r)] = paintengine2d.AtlasCell{
			Src:     paintengine2d.XYWH(float32(ox), float32(oy), cw, ch),
			Advance: cw,
		}
		i++
	}
	return &paintengine2d.FontAtlas{Image: img, Cells: cells}
}

var (
	tintOnce sync.Once
	tintOK   bool
)

// GlyphTint reports whether paintengine2d blits apply RGB Color as a tint
// (v0.7.2+). uitoolkit requires this for themed white atlases.
func GlyphTint() bool {
	tintOnce.Do(func() {
		atlas := paintengine2d.NewBitmapAtlas(paintengine2d.White)
		img := paintengine2d.NewImage(20, 16)
		ctx := paintengine2d.NewContext(img)
		run := paintengine2d.NullShaper{}.Shape("I", atlas)
		ctx.DrawGlyphs(run, paintengine2d.Pt(1, 1), paintengine2d.Paint{
			Color:  paintengine2d.RGB(1, 0, 0),
			Filter: paintengine2d.FilterNearest,
		})
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				r, g, b, a := img.PremulAt(x, y)
				if a > 20 && int(r) > int(g)+24 && int(r) > int(b)+24 {
					tintOK = true
					return
				}
			}
		}
	})
	return tintOK
}
