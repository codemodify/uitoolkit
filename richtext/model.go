// Package richtext is the document model behind widgets.RichText: styled
// paragraphs, headings, bulleted and numbered lists that nest, links and
// inline images, with a selection, an undo history grouped the way people
// type, and a small documented subset of HTML to load and save (html.go).
//
// A document is a list of blocks, each a run of spans. Blocks are values:
// an edit never changes a block in place but builds a new one and swaps it
// in, so the undo history holds the blocks an edit replaced as they were,
// and a view that caches a block's layout by its pointer keeps every cache
// entry the edit did not touch — typing in one paragraph of ten thousand
// re-lays one paragraph.
package richtext

import (
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// ObjectChar stands for an inline image in a block's text: the character
// AT-SPI and Qt use for an embedded object, so offsets count it as one.
const ObjectChar = '￼'

// Style is how a span of text is drawn. The zero Style is the block's own
// text: the look's face at the block's size, in the look's text colour.
type Style struct {
	Bold, Italic, Underline, Strike bool
	// Mono sets the span in the look's monospace face (HTML's <code>).
	Mono bool
	// Size is the text size in logical pixels; 0 is the block's own (the
	// look's body size, or a heading's).
	Size float32
	// Color is the text colour and Highlight the colour behind it; a zero
	// colour (alpha 0) is the look's text and no highlight.
	Color     paintengine2d.Color
	Highlight paintengine2d.Color
	// Link is a hyperlink's target; empty for none.
	Link string
}

// Image is an inline picture. Src is where it came from — a data: URI
// the document carries itself, or a name the application resolves
// (Doc.ResolveImage) — and survives a save unchanged. W and H are its size
// in logical pixels; zero takes the picture's own.
type Image struct {
	Src    string
	Alt    string
	W, H   float32
	Pixels *paintengine2d.Image
}

// Size is the image's box in logical pixels: W and H when given, the
// picture's own size otherwise (keeping its aspect when only one is set),
// and a small placeholder when there is no picture yet.
func (im *Image) Size() (w, h float32) {
	if im == nil {
		return 0, 0
	}
	w, h = im.W, im.H
	var pw, ph float32 = 32, 32
	if im.Pixels != nil && im.Pixels.Width > 0 && im.Pixels.Height > 0 {
		pw, ph = float32(im.Pixels.Width), float32(im.Pixels.Height)
	}
	switch {
	case w > 0 && h > 0:
	case w > 0:
		h = w * ph / pw
	case h > 0:
		w = h * pw / ph
	default:
		w, h = pw, ph
	}
	return w, h
}

// Span is a run of text in one style, or one inline image (Text is then
// ObjectChar).
type Span struct {
	Text  string
	Style Style
	Image *Image
}

// Kind is what a block is.
type Kind uint8

const (
	Paragraph Kind = iota
	// Heading is a section heading of Level 1 to 6.
	Heading
	// Bullet and Numbered are list items; Level is the nesting, 0 the
	// outermost.
	Bullet
	Numbered
)

// MaxLevel is the deepest list nesting.
const MaxLevel = 8

// IsList reports whether k is a list item.
func (k Kind) IsList() bool { return k == Bullet || k == Numbered }

// Align is a block's horizontal alignment.
type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Block is a paragraph, a heading or a list item: a run of spans.
//
// Blocks are values. Never change one that is in a document; build a new
// one (NewBlock) and put it in with Doc.ReplaceBlocks, so the history and
// any view's layout cache stay right.
type Block struct {
	Kind  Kind
	Level int
	Align Align
	Spans []Span
	n     int // runes, counted once
}

// NewBlock is a block of kind k at level holding spans (merged where two
// neighbours share a style).
func NewBlock(k Kind, level int, spans ...Span) *Block {
	b := &Block{Kind: k, Level: level}
	b.Spans = normalize(spans)
	b.count()
	return b
}

func (b *Block) count() {
	n := 0
	for _, s := range b.Spans {
		n += utf8.RuneCountInString(s.Text)
	}
	b.n = n
}

// Len is the block's length in characters (an image counts one).
func (b *Block) Len() int { return b.n }

// Text is the block's characters, an image as ObjectChar.
func (b *Block) Text() string {
	if len(b.Spans) == 1 {
		return b.Spans[0].Text
	}
	n := 0
	for _, s := range b.Spans {
		n += len(s.Text)
	}
	out := make([]byte, 0, n)
	for _, s := range b.Spans {
		out = append(out, s.Text...)
	}
	return string(out)
}

// with is a copy of b's kind, level and alignment around new spans.
func (b *Block) with(spans []Span) *Block {
	nb := &Block{Kind: b.Kind, Level: b.Level, Align: b.Align, Spans: normalize(spans)}
	nb.count()
	return nb
}

// clone is a copy of b that may be given a new kind, level or alignment.
func (b *Block) clone() *Block {
	nb := *b
	return &nb
}

// Equal reports whether two blocks hold the same content and form.
func (b *Block) Equal(o *Block) bool {
	if b == o {
		return true
	}
	if b == nil || o == nil || b.Kind != o.Kind || b.Level != o.Level || b.Align != o.Align || len(b.Spans) != len(o.Spans) {
		return false
	}
	for i := range b.Spans {
		x, y := b.Spans[i], o.Spans[i]
		if x.Text != y.Text || x.Style != y.Style || (x.Image == nil) != (y.Image == nil) {
			return false
		}
		if x.Image != nil && (x.Image.Src != y.Image.Src || x.Image.Alt != y.Image.Alt || x.Image.W != y.Image.W || x.Image.H != y.Image.H) {
			return false
		}
	}
	return true
}

// normalize drops empty spans and merges neighbours that share a style.
// Images never merge: each is one span of its own.
func normalize(spans []Span) []Span {
	out := make([]Span, 0, len(spans))
	for _, s := range spans {
		if s.Text == "" {
			continue
		}
		if s.Image != nil {
			s.Text = string(ObjectChar)
		}
		if n := len(out); n > 0 && s.Image == nil && out[n-1].Image == nil && out[n-1].Style == s.Style {
			out[n-1].Text += s.Text
			continue
		}
		out = append(out, s)
	}
	return out
}

// runeOffset is the byte offset of rune i in s (len(s) past the end).
func runeOffset(s string, i int) int {
	if i <= 0 {
		return 0
	}
	n := 0
	for bi := range s {
		if n == i {
			return bi
		}
		n++
	}
	return len(s)
}

// sliceSpans is a copy of b's spans covering characters [a, c).
func sliceSpans(b *Block, a, c int) []Span {
	if a < 0 {
		a = 0
	}
	if c > b.n {
		c = b.n
	}
	if a >= c {
		return nil
	}
	var out []Span
	pos := 0
	for _, s := range b.Spans {
		n := utf8.RuneCountInString(s.Text)
		lo, hi := max(a, pos), min(c, pos+n)
		if lo < hi {
			t := s
			t.Text = s.Text[runeOffset(s.Text, lo-pos):runeOffset(s.Text, hi-pos)]
			out = append(out, t)
		}
		pos += n
		if pos >= c {
			break
		}
	}
	return out
}

// spliced is b with characters [a, c) replaced by ins.
func spliced(b *Block, a, c int, ins []Span) *Block {
	spans := make([]Span, 0, len(b.Spans)+len(ins)+1)
	spans = append(spans, sliceSpans(b, 0, a)...)
	spans = append(spans, ins...)
	spans = append(spans, sliceSpans(b, c, b.n)...)
	return b.with(spans)
}

// restyled is b with fn applied to the style of characters [a, c).
func restyled(b *Block, a, c int, fn func(*Style)) *Block {
	mid := sliceSpans(b, a, c)
	for i := range mid {
		fn(&mid[i].Style)
	}
	return spliced(b, a, c, mid)
}

// StyleAt is the style of the character at off (the last one for off past
// the end); the zero Style for an empty block.
func (b *Block) StyleAt(off int) Style {
	if s, ok := b.spanAt(off); ok {
		return s.Style
	}
	return Style{}
}

// spanAt is the span holding character off.
func (b *Block) spanAt(off int) (Span, bool) {
	if len(b.Spans) == 0 {
		return Span{}, false
	}
	if off >= b.n {
		return b.Spans[len(b.Spans)-1], true
	}
	pos := 0
	for _, s := range b.Spans {
		n := utf8.RuneCountInString(s.Text)
		if off < pos+n {
			return s, true
		}
		pos += n
	}
	return b.Spans[len(b.Spans)-1], true
}

// ImageAt is the image at character off, or nil.
func (b *Block) ImageAt(off int) *Image {
	if off < 0 || off >= b.n {
		return nil
	}
	if s, ok := b.spanAt(off); ok {
		return s.Image
	}
	return nil
}

// LinkRange is the run of characters around off that share its link, and
// the link; ok is false when off is not on a link.
func (b *Block) LinkRange(off int) (a, c int, href string, ok bool) {
	if off < 0 || off >= b.n {
		return 0, 0, "", false
	}
	pos := 0
	var spans []struct{ a, c int }
	var hit = -1
	for _, s := range b.Spans {
		n := utf8.RuneCountInString(s.Text)
		spans = append(spans, struct{ a, c int }{pos, pos + n})
		if off >= pos && off < pos+n {
			hit = len(spans) - 1
			href = s.Style.Link
		}
		pos += n
	}
	if hit < 0 || href == "" {
		return 0, 0, "", false
	}
	lo, hi := hit, hit
	for lo > 0 && b.Spans[lo-1].Style.Link == href {
		lo--
	}
	for hi+1 < len(b.Spans) && b.Spans[hi+1].Style.Link == href {
		hi++
	}
	return spans[lo].a, spans[hi].c, href, true
}
