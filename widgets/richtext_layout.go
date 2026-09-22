package widgets

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// RichText's layout. Each block is laid out on its own, into lines of
// fragments (a run of one span's text on one line, or an image), and the
// result is kept by the block's pointer: blocks are values, so an edit
// that swaps one block leaves every other block's layout valid. Blocks not
// yet laid out get an estimated height from their length, which is what
// keeps loading and scrolling a ten-thousand-paragraph document cheap —
// only what is on screen is ever laid out before it is needed, and the
// rest is laid out a slice at a time while the editor is idle.

// rtFrag is a run of one span's text on one line, or one image.
type rtFrag struct {
	start, end int // block characters
	text       string
	font       *style.Font
	st         richtext.Style
	x, w       float32
	img        *richtext.Image
	imgW, imgH float32
}

// rtLine is one visual line of a block, in block coordinates.
type rtLine struct {
	start, end int
	// soft: the line ends at a wrap, so end is also the next line's start.
	soft    bool
	y, h    float32
	base    float32 // baseline, from the line's top
	x, w    float32 // where the ink starts and how wide it is
	frags   []rtFrag
	ascent  float32
	descent float32
}

// rtLayout is one block laid out at one width in one look.
type rtLayout struct {
	width  float32
	sig    rtSig
	lines  []rtLine
	h      float32 // with the spacing before and after
	before float32
	indent float32
	base   *style.Font
}

// rtSig is what a layout depends on besides the block and the width: the
// look's face and scale.
type rtSig struct {
	face  faceKey
	mono  faceKey
	scale float32
}

// rtFaceKey names a baked face.
type rtFaceKey struct {
	family string
	weight style.Weight
	size   int
}

// headingScale is a heading's size against body text, h1 to h6.
var headingScale = [7]float32{1, 1.85, 1.5, 1.25, 1.1, 1, 0.9}

func (t *RichText) sig() rtSig {
	lk := t.Look()
	return rtSig{face: faceKeyOf(lk.Font()), mono: faceKeyOf(lk.MonoFont()), scale: style.LookScale(lk)}
}

// dip is v logical pixels in the look's device pixels.
func (t *RichText) dip(v float32) float32 { return style.Dip(t.Look(), v) }

// bodySize is the size of body text in device pixels.
func (t *RichText) bodySize() float32 {
	if f := t.Look().Font(); f != nil && f.Size > 0 {
		return f.Size
	}
	return t.dip(13)
}

// face is the font span style st is drawn in, in a block of kind k.
func (t *RichText) face(st richtext.Style, k richtext.Kind, level int) *style.Font {
	lk := t.Look()
	base := lk.Font()
	family := base.Family
	if st.Mono {
		family = lk.MonoFont().Family
	}
	size := t.bodySize()
	weight := style.WeightRegular
	if k == richtext.Heading {
		size *= headingScale[min(max(level, 1), 6)]
		weight = style.WeightBold
	}
	if st.Size > 0 {
		size = t.dip(st.Size)
	}
	if st.Mono && st.Size <= 0 {
		// Monospace reads larger than the UI face at the same size.
		size *= 0.92
	}
	if st.Bold {
		weight = style.WeightBold
	}
	key := rtFaceKey{family, weight, int(size*4 + 0.5)}
	if f, ok := t.faces[key]; ok {
		return f
	}
	if t.faces == nil {
		t.faces = map[rtFaceKey]*style.Font{}
	}
	f := style.BakeFamily(family, weight, size, lk.Palette().Text)
	t.faces[key] = f
	return f
}

// listIndent is how far each list level indents.
func (t *RichText) listIndent() float32 { return float32(math.Round(float64(t.dip(24)))) }

// spacing is the room before and after a block of kind k.
func (t *RichText) spacing(b *richtext.Block) (before, after float32) {
	switch b.Kind {
	case richtext.Heading:
		return t.dip(8), t.dip(4)
	case richtext.Bullet, richtext.Numbered:
		return 0, t.dip(2)
	}
	return 0, t.dip(6)
}

// imageBox is an image's size on screen, no wider than avail.
func (t *RichText) imageBox(im *richtext.Image, avail float32) (w, h float32) {
	lw, lh := im.Size()
	w, h = t.dip(lw), t.dip(lh)
	if w > avail && w > 0 {
		h *= avail / w
		w = avail
	}
	return float32(math.Round(float64(w))), float32(math.Round(float64(h)))
}

// drawText is text as it is drawn: a tab shows as a space (the character
// count is the layout's, so carets still line up).
func drawText(s string) string {
	if strings.IndexByte(s, '\t') < 0 {
		return s
	}
	return strings.ReplaceAll(s, "\t", " ")
}

// rtSeg is a word and the spaces after it, or one image: what line
// breaking places.
type rtSeg struct {
	span       int
	start, end int
	text       string
	w, ink     float32 // with and without the trailing spaces
	font       *style.Font
	st         richtext.Style
	img        *richtext.Image
	imgW, imgH float32
}

// layoutBlock lays b out at width.
func (t *RichText) layoutBlock(b *richtext.Block, width float32) *rtLayout {
	lay := &rtLayout{width: width, sig: t.sig()}
	lay.base = t.face(richtext.Style{}, b.Kind, b.Level)
	if b.Kind.IsList() {
		lay.indent = t.listIndent() * float32(min(max(b.Level, 0), richtext.MaxLevel)+1)
	}
	avail := max(width-lay.indent, t.dip(24))
	var segs []rtSeg
	pos := 0
	for si, s := range b.Spans {
		n := utf8.RuneCountInString(s.Text)
		if s.Image != nil {
			w, h := t.imageBox(s.Image, avail)
			segs = append(segs, rtSeg{span: si, start: pos, end: pos + 1, w: w, ink: w, st: s.Style, img: s.Image, imgW: w, imgH: h})
			pos++
			continue
		}
		f := t.face(s.Style, b.Kind, b.Level)
		rs := []rune(s.Text)
		for i := 0; i < len(rs); {
			j := i
			for j < len(rs) && rs[j] != ' ' && rs[j] != '\t' {
				j++
			}
			k := j
			for k < len(rs) && (rs[k] == ' ' || rs[k] == '\t') {
				k++
			}
			word := drawText(string(rs[i:k]))
			segs = append(segs, rtSeg{span: si, start: pos + i, end: pos + k, text: word,
				w: f.Advance(word), ink: f.Advance(drawText(string(rs[i:j]))), font: f, st: s.Style})
			i = k
		}
		pos += n
	}
	var lines []rtLine
	cur := rtLine{}
	var x float32
	add := func(sg rtSeg) {
		if n := len(cur.frags); n > 0 && sg.img == nil && cur.frags[n-1].img == nil &&
			cur.frags[n-1].font == sg.font && cur.frags[n-1].st == sg.st && cur.frags[n-1].end == sg.start {
			cur.frags[n-1].text += sg.text
			cur.frags[n-1].end = sg.end
		} else {
			cur.frags = append(cur.frags, rtFrag{start: sg.start, end: sg.end, text: sg.text, font: sg.font, st: sg.st, img: sg.img, imgW: sg.imgW, imgH: sg.imgH})
		}
		x += sg.w
	}
	wrap := func(at int) {
		cur.end, cur.soft = at, true
		lines = append(lines, cur)
		cur = rtLine{start: at}
		x = 0
	}
	for _, sg := range segs {
		if x > 0 && x+sg.ink > avail {
			wrap(sg.start)
		}
		if sg.img == nil && x == 0 && sg.ink > avail {
			// A word longer than the line breaks between characters.
			rs := []rune(sg.text)
			for len(rs) > 0 {
				piece := string(rs)
				if sg.font.Advance(piece) <= avail {
					add(rtSeg{span: sg.span, start: sg.start, end: sg.end, text: piece, w: sg.font.Advance(piece), font: sg.font, st: sg.st})
					break
				}
				k := max(sg.font.IndexAt(piece, avail), 1)
				for k > 1 && sg.font.CaretX(piece, k) > avail {
					k--
				}
				head := string(rs[:k])
				add(rtSeg{span: sg.span, start: sg.start, end: sg.start + k, text: head, w: sg.font.Advance(head), font: sg.font, st: sg.st})
				sg.start += k
				rs = rs[k:]
				wrap(sg.start)
			}
			continue
		}
		add(sg)
	}
	cur.end = b.Len()
	lines = append(lines, cur)

	lineGap := t.dip(2)
	before, after := t.spacing(b)
	lay.before = before
	y := before
	for i := range lines {
		ln := &lines[i]
		var fx float32
		asc, desc := lay.base.Ascent, fontDescent(lay.base)
		if len(ln.frags) > 0 {
			asc, desc = 0, 0
		}
		for j := range ln.frags {
			f := &ln.frags[j]
			f.x = fx
			if f.img != nil {
				f.w = f.imgW
				asc = max(asc, f.imgH)
			} else {
				f.w = f.font.Advance(f.text)
				asc = max(asc, f.font.Ascent)
				desc = max(desc, fontDescent(f.font))
			}
			fx += f.w
		}
		// The ink width leaves out the spaces a wrapped line ends with.
		ink := fx
		if n := len(ln.frags); n > 0 && ln.frags[n-1].img == nil {
			last := ln.frags[n-1]
			ink = last.x + last.font.Advance(strings.TrimRight(last.text, " "))
		}
		ln.w = ink
		ln.x = lay.indent
		switch b.Align {
		case richtext.AlignCenter:
			ln.x += max(avail-ink, 0) * 0.5
		case richtext.AlignRight:
			ln.x += max(avail-ink, 0)
		}
		ln.x = float32(math.Round(float64(ln.x)))
		ln.ascent, ln.descent = asc, desc
		ln.base = float32(math.Round(float64(lineGap*0.5 + asc)))
		ln.h = float32(math.Ceil(float64(asc + desc + lineGap)))
		ln.y = y
		y += ln.h
	}
	lay.lines = lines
	lay.h = y + after
	return lay
}

func fontDescent(f *style.Font) float32 {
	if f.Descent > 0 {
		return f.Descent
	}
	return max(f.Height()-f.Ascent, 0)
}

// ---- the document's geometry -------------------------------------------

// contentW is the width blocks are laid out at: the view less its padding
// and, where the look's scroll bar takes room of its own, the bar.
func (t *RichText) contentW() float32 {
	w := t.inner().Dx()
	if sb := style.ScrollBarStyleOf(t.Look()); !sb.Overlay {
		th := sb.Thickness
		if th <= 0 {
			th = t.Look().Metrics().Scroll
		}
		w -= th
	}
	return max(w, t.dip(40))
}

// syncGeometry drops every layout when the look or the width changed, and
// keeps the heights as estimates until the blocks are laid out again.
func (t *RichText) syncGeometry() {
	sig, w := t.sig(), t.contentW()
	n := t.doc.Len()
	if len(t.heights) != n {
		t.heights = make([]float32, n)
		for i := range t.heights {
			t.heights[i] = t.estimate(t.doc.Block(i), w)
		}
		t.topsOK = 0
	}
	if sig == t.laidSig && w == t.laidW {
		return
	}
	resized := sig == t.laidSig
	t.laidSig, t.laidW = sig, w
	clear(t.cache)
	t.avgW = 0
	if !resized {
		t.faces = nil
	}
	for i := range t.heights {
		t.heights[i] = t.estimate(t.doc.Block(i), w)
	}
	t.topsOK = 0
	t.scheduleIdle()
}

// estimate is a block's height before it is laid out: its characters at
// the body face's average width, wrapped at w.
func (t *RichText) estimate(b *richtext.Block, w float32) float32 {
	f := t.face(richtext.Style{}, b.Kind, b.Level)
	if t.avgW <= 0 {
		const sample = "the quick brown fox jumps over a lazy dog, THEN 12 more."
		t.avgW = max(t.face(richtext.Style{}, richtext.Paragraph, 0).Advance(sample)/float32(len(sample)), 1)
	}
	avg := t.avgW * f.Size / max(t.bodySize(), 1)
	lines := 1
	if b.Len() > 0 {
		lines = int(math.Ceil(float64(float32(b.Len()) * avg / max(w, 1))))
	}
	before, after := t.spacing(b)
	lh := float32(math.Ceil(float64(f.Ascent + fontDescent(f) + t.dip(2))))
	for _, s := range b.Spans {
		if s.Image != nil {
			_, ih := t.imageBox(s.Image, w)
			lh = max(lh, ih+t.dip(4))
		}
	}
	return before + float32(max(lines, 1))*lh + after
}

// lay is block i's layout, laid out now if it is not already. A height
// that changes moves every block below it; one that changes above the
// view moves the view with it, so what the user is reading stays put.
func (t *RichText) lay(i int) *rtLayout {
	t.syncGeometry()
	b := t.doc.Block(i)
	if l, ok := t.cache[b]; ok && l.width == t.laidW && l.sig == t.laidSig {
		if t.heights[i] != l.h {
			t.setHeight(i, l.h)
		}
		return l
	}
	l := t.layoutBlock(b, t.laidW)
	if t.cache == nil {
		t.cache = map[*richtext.Block]*rtLayout{}
	}
	if len(t.cache) > 2*t.doc.Len()+256 {
		t.pruneCache()
	}
	t.cache[b] = l
	t.laidOut++
	if t.heights[i] != l.h {
		t.setHeight(i, l.h)
	}
	return l
}

func (t *RichText) setHeight(i int, h float32) {
	top := t.top(i)
	old := t.heights[i]
	t.heights[i] = h
	t.topsOK = min(t.topsOK, i+1)
	if top+old <= t.scrollY && !t.pinTop {
		t.scrollY += h - old
	}
}

// pruneCache drops the layouts of blocks that are no longer in the
// document (undo restores the old blocks, and re-lays them if it must).
func (t *RichText) pruneCache() {
	live := make(map[*richtext.Block]bool, t.doc.Len())
	for _, b := range t.doc.Blocks() {
		live[b] = true
	}
	for b := range t.cache {
		if !live[b] {
			delete(t.cache, b)
		}
	}
}

// blocksChanged follows an edit: removed blocks at at became added new
// ones, whose heights are estimated until they are laid out.
func (t *RichText) blocksChanged(at, removed, added int) {
	if len(t.heights) != t.doc.Len()-added+removed {
		t.heights = nil // the next sync rebuilds them all
		return
	}
	nh := make([]float32, added)
	for k := range nh {
		nh[k] = t.estimate(t.doc.Block(at+k), t.contentW())
	}
	tail := append([]float32(nil), t.heights[at+removed:]...)
	t.heights = append(append(t.heights[:at], nh...), tail...)
	t.topsOK = min(t.topsOK, at)
}

// top is block i's top in the document (i == Len is the document's end).
func (t *RichText) top(i int) float32 {
	n := len(t.heights)
	if len(t.tops) != n+1 {
		t.tops = make([]float32, n+1)
		t.topsOK = 0
	}
	if i > t.topsOK {
		for k := t.topsOK; k < i; k++ {
			t.tops[k+1] = t.tops[k] + t.heights[k]
		}
		t.topsOK = i
	}
	return t.tops[i]
}

// docH is the whole document's height.
func (t *RichText) docH() float32 {
	t.syncGeometry()
	return t.top(len(t.heights))
}

// blockAt is the block at document y.
func (t *RichText) blockAt(y float32) int {
	n := len(t.heights)
	t.top(n)
	i := sort.Search(n, func(i int) bool { return t.tops[i+1] > y })
	return min(max(i, 0), n-1)
}

// lineFor is the line of lay that holds block character off; up picks the
// end of a wrapped line over the start of the next.
func (l *rtLayout) lineFor(off int, up bool) int {
	for i, ln := range l.lines {
		if off < ln.end && off >= ln.start {
			return i
		}
		if off == ln.end && (i == len(l.lines)-1 || up && ln.soft) {
			return i
		}
	}
	return len(l.lines) - 1
}

// caretX is where character off sits on line ln, in block coordinates.
func (ln *rtLine) caretX(off int) float32 {
	if off <= ln.start || len(ln.frags) == 0 {
		return ln.x
	}
	for _, f := range ln.frags {
		if off < f.end || off == f.end && f.end == ln.frags[len(ln.frags)-1].end {
			if f.img != nil {
				if off <= f.start {
					return ln.x + f.x
				}
				return ln.x + f.x + f.w
			}
			return ln.x + f.x + f.font.CaretX(f.text, off-f.start)
		}
	}
	last := ln.frags[len(ln.frags)-1]
	return ln.x + last.x + last.w
}

// offsetAt is the character of line ln nearest x (block coordinates), and
// whether it sits at the end of a wrapped line rather than the next one's
// start.
func (ln *rtLine) offsetAt(x float32) (int, bool) {
	if x <= ln.x || len(ln.frags) == 0 {
		return ln.start, false
	}
	for _, f := range ln.frags {
		if x < ln.x+f.x+f.w {
			if f.img != nil {
				if x < ln.x+f.x+f.w*0.5 {
					return f.start, false
				}
				return f.end, false
			}
			k := f.font.IndexAt(f.text, x-ln.x-f.x)
			off := f.start + min(k, f.end-f.start)
			if ln.soft && off >= ln.end {
				// Past the spaces a wrapped line ends with: stay on it.
				return ln.end, true
			}
			return off, false
		}
	}
	return ln.end, ln.soft
}

// frag is the fragment of ln that holds off, for the caret's height.
func (ln *rtLine) fragAt(off int) *rtFrag {
	for i := range ln.frags {
		f := &ln.frags[i]
		if off >= f.start && off < f.end {
			return f
		}
	}
	if n := len(ln.frags); n > 0 && off > ln.frags[0].start {
		return &ln.frags[n-1]
	}
	if len(ln.frags) > 0 {
		return &ln.frags[0]
	}
	return nil
}

// ---- idle layout -------------------------------------------------------

// scheduleIdle lays the blocks not laid out yet a slice at a time while
// the editor is idle, so the scroll bar's size settles on the truth.
func (t *RichText) scheduleIdle() {
	if t.idleStop != nil || t.doc == nil {
		return
	}
	if _, ok := t.Host().(widget.Timers); !ok {
		return // no clock (a detached editor, a test): lay out on demand
	}
	t.idleStop = afterIdle(t, func() {
		t.idleStop = nil
		t.idleStep()
	})
}

// idleSlice is how many blocks one idle step lays out.
const idleSlice = 150

func (t *RichText) idleStep() {
	if t.doc == nil || t.Host() == nil {
		return
	}
	t.syncGeometry()
	done := 0
	for done < idleSlice && t.idleNext < t.doc.Len() {
		b := t.doc.Block(t.idleNext)
		if l, ok := t.cache[b]; !ok || l.width != t.laidW || l.sig != t.laidSig {
			t.lay(t.idleNext)
			done++
		}
		t.idleNext++
	}
	if t.idleNext < t.doc.Len() {
		t.scheduleIdle()
	} else {
		t.idleNext = 0
	}
	if done > 0 {
		// The bar's thumb follows the document's new height.
		t.Invalidate()
	}
}

// visibleBlocks is the range of blocks the view shows.
func (t *RichText) visibleBlocks() (lo, hi int) {
	t.syncGeometry()
	in := t.inner()
	lo = t.blockAt(t.scrollY)
	hi = lo
	for hi < t.doc.Len() && t.top(hi) < t.scrollY+in.Dy() {
		hi++
	}
	return lo, hi
}

// fragColor is the colour a fragment's text is drawn in.
func (t *RichText) fragColor(st richtext.Style, text, link paintengine2d.Color) paintengine2d.Color {
	switch {
	case st.Color.A > 0:
		return st.Color
	case st.Link != "":
		return link
	}
	return text
}
