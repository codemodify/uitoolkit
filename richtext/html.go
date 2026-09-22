package richtext

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"image"
	_ "image/gif"  // data: URIs in GIF
	_ "image/jpeg" // and JPEG
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// The HTML subset.
//
// A document saves as, and loads from, a small subset of HTML — what a
// word processor's clipboard carries and a mail composer writes, without
// the rest of the web. What is read:
//
//   - blocks: <p>, <div>, <h1>–<h6>, <ul> and <ol> with <li> (nested lists
//     nest), <blockquote> and <pre> (as paragraphs; <pre> keeps its lines
//     and sets them in the monospace face), <br> (ends the block and starts
//     another of the same kind), and the text-align of a block's style;
//   - character styles: <b>/<strong>, <i>/<em>, <u>/<ins>, <s>/<strike>/
//     <del>, <code>/<tt>/<kbd>/<samp>, <mark> (a yellow highlight), <a
//     href>, <font color size>, and a style attribute on any element with
//     font-weight, font-style, text-decoration, font-size (px, pt, em, %),
//     color and background-color (#rgb, #rrggbb, rgb(), rgba() and the
//     basic colour names);
//   - <img src alt width height>, its pixels from a data: URI (PNG, JPEG,
//     GIF) or from Doc.ResolveImage.
//
// Everything else is read for its text alone: unknown tags are dropped but
// their content kept, <head>, <style> and <script> are skipped whole, and
// runs of white space collapse as a browser collapses them. What is
// written is the same subset: the blocks above, <b> <i> <u> <s> <code>
// <a>, and one <span style> for size and colours, so a document survives
// being saved and loaded again unchanged.

// ParseHTML reads the subset of HTML above into a new document.
func ParseHTML(s string) (*Doc, error) {
	d := New()
	if err := d.SetHTML(s); err != nil {
		return nil, err
	}
	return d, nil
}

// SetHTML replaces the document with the subset of HTML read from s and
// forgets the history, as loading a file does.
func (d *Doc) SetHTML(s string) error {
	p := htmlParser{resolve: d.ResolveImage}
	blocks := p.parse(s)
	d.Reset(blocks)
	return nil
}

// FragmentHTML reads s as a fragment to paste (the clipboard's HTML).
func FragmentHTML(s string, resolve func(string) *paintengine2d.Image) *Doc {
	p := htmlParser{resolve: resolve}
	return &Doc{blocks: p.parse(s)}
}

// ---- reading -----------------------------------------------------------

type htmlToken struct {
	kind  byte // 't' text, 's' start, 'e' end
	name  string
	attrs map[string]string
	text  string
	self  bool
}

// tokenize splits s into text, start and end tags, dropping comments,
// doctypes and processing instructions. It never fails: a stray "<" is
// text.
func tokenize(s string) []htmlToken {
	var out []htmlToken
	i := 0
	text := func(t string) {
		if t != "" {
			out = append(out, htmlToken{kind: 't', text: t})
		}
	}
	for i < len(s) {
		lt := strings.IndexByte(s[i:], '<')
		if lt < 0 {
			text(s[i:])
			break
		}
		text(s[i : i+lt])
		i += lt
		switch {
		case strings.HasPrefix(s[i:], "<!--"):
			end := strings.Index(s[i+4:], "-->")
			if end < 0 {
				return out
			}
			i += 4 + end + 3
			continue
		case strings.HasPrefix(s[i:], "<!") || strings.HasPrefix(s[i:], "<?"):
			end := strings.IndexByte(s[i:], '>')
			if end < 0 {
				return out
			}
			i += end + 1
			continue
		}
		tok, n, ok := readTag(s[i:])
		if !ok {
			text("<")
			i++
			continue
		}
		i += n
		out = append(out, tok)
		if tok.kind == 's' && (tok.name == "script" || tok.name == "style" || tok.name == "title") {
			// Raw text elements: skip to their end tag.
			end := strings.Index(strings.ToLower(s[i:]), "</"+tok.name)
			if end < 0 {
				return out
			}
			i += end
		}
	}
	return out
}

// readTag reads the tag at the start of s ("<b>", "</p>", "<img …/>").
func readTag(s string) (htmlToken, int, bool) {
	i := 1
	end := false
	if i < len(s) && s[i] == '/' {
		end = true
		i++
	}
	start := i
	for i < len(s) && (isAlnum(s[i]) || s[i] == '-' || s[i] == ':') {
		i++
	}
	if i == start {
		return htmlToken{}, 0, false
	}
	tok := htmlToken{kind: 's', name: strings.ToLower(s[start:i])}
	if end {
		tok.kind = 'e'
	}
	attrs := map[string]string{}
	for i < len(s) {
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}
		if i >= len(s) {
			return htmlToken{}, 0, false
		}
		if s[i] == '>' {
			i++
			break
		}
		if s[i] == '/' {
			tok.self = true
			i++
			continue
		}
		ns := i
		for i < len(s) && !isSpaceByte(s[i]) && s[i] != '=' && s[i] != '>' && s[i] != '/' {
			i++
		}
		name := strings.ToLower(s[ns:i])
		for i < len(s) && isSpaceByte(s[i]) {
			i++
		}
		val := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && isSpaceByte(s[i]) {
				i++
			}
			if i < len(s) && (s[i] == '"' || s[i] == '\'') {
				q := s[i]
				j := strings.IndexByte(s[i+1:], q)
				if j < 0 {
					return htmlToken{}, 0, false
				}
				val = s[i+1 : i+1+j]
				i += j + 2
			} else {
				vs := i
				for i < len(s) && !isSpaceByte(s[i]) && s[i] != '>' {
					i++
				}
				val = s[vs:i]
			}
		}
		if name != "" {
			attrs[name] = html.UnescapeString(val)
		}
	}
	tok.attrs = attrs
	return tok, i, true
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// htmlParser builds blocks from tokens.
type htmlParser struct {
	resolve func(string) *paintengine2d.Image
	blocks  []*Block
	// cur is the block being filled; open reports that a block element
	// asked for it, so it is kept even when it stays empty (<p></p>).
	cur   *Block
	spans []Span
	open  bool
	// styles is the character style of each open inline element, lists
	// the kind of each open list, pre counts open <pre>s.
	styles []styleFrame
	lists  []Kind
	pre    int
	// head counts open <head>s, whose content is skipped.
	head int
	// kind and align are what the next block becomes.
	kind  Kind
	level int
	align Align
	// space is a pending collapsed space, written before the next text.
	space bool
}

type styleFrame struct {
	name string
	st   Style
}

var blockTags = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"li": true, "blockquote": true, "pre": true, "ul": true, "ol": true, "body": true, "html": true,
	"table": true, "tr": true, "section": true, "article": true, "header": true, "footer": true,
}

func (p *htmlParser) style() Style {
	if n := len(p.styles); n > 0 {
		return p.styles[n-1].st
	}
	return Style{}
}

func (p *htmlParser) parse(s string) []*Block {
	for _, t := range tokenize(s) {
		switch t.kind {
		case 't':
			if p.head == 0 {
				p.text(html.UnescapeString(t.text))
			}
		case 's':
			p.start(t)
		case 'e':
			p.end(t.name)
		}
	}
	p.flush(false)
	if len(p.blocks) == 0 {
		return []*Block{NewBlock(Paragraph, 0)}
	}
	return p.blocks
}

// flush ends the block being filled; force keeps an empty one.
func (p *htmlParser) flush(force bool) {
	if p.cur == nil {
		return
	}
	if len(p.spans) > 0 {
		// Trailing white space at a block's end is not content.
		last := &p.spans[len(p.spans)-1]
		if last.Image == nil && p.pre == 0 {
			last.Text = strings.TrimRight(last.Text, " ")
		}
	}
	if len(p.spans) > 0 || force || p.open {
		b := NewBlock(p.cur.Kind, p.cur.Level, p.spans...)
		b.Align = p.cur.Align
		if len(b.Spans) > 0 || force || p.open {
			p.blocks = append(p.blocks, b)
		}
	}
	p.cur, p.spans, p.open, p.space = nil, nil, false, false
}

// begin starts a block of the pending kind.
func (p *htmlParser) begin() {
	if p.cur != nil {
		return
	}
	p.cur = &Block{Kind: p.kind, Level: p.level, Align: p.align}
	p.space = false
}

func (p *htmlParser) text(t string) {
	if p.pre > 0 {
		lines := strings.Split(strings.ReplaceAll(t, "\r\n", "\n"), "\n")
		for i, ln := range lines {
			if i > 0 {
				p.flush(true)
				p.open = true
			}
			if ln == "" {
				p.begin()
				p.open = true
				continue
			}
			p.begin()
			st := p.style()
			st.Mono = true
			p.spans = append(p.spans, Span{Text: strings.ReplaceAll(ln, "\t", "    "), Style: st})
		}
		return
	}
	// Collapse white space as a browser does: a run of it is one space,
	// kept with the text before it, and none at a block's start.
	var b strings.Builder
	for _, r := range t {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			if !p.space && (b.Len() > 0 || p.cur != nil && len(p.spans) > 0) {
				b.WriteByte(' ')
			}
			p.space = true
			continue
		}
		p.space = false
		if r == '\u00a0' {
			r = ' '
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return
	}
	p.begin()
	p.spans = append(p.spans, Span{Text: b.String(), Style: p.style()})
}

func (p *htmlParser) start(t htmlToken) {
	if t.name == "head" {
		p.head++
		return
	}
	if p.head > 0 {
		return
	}
	align, hasAlign := AlignLeft, false
	if a, ok := cssProps(t.attrs["style"])["text-align"]; ok {
		hasAlign = true
		switch strings.TrimSpace(a) {
		case "center":
			align = AlignCenter
		case "right", "end":
			align = AlignRight
		}
	}
	if v, ok := t.attrs["align"]; ok {
		hasAlign = true
		switch strings.ToLower(v) {
		case "center":
			align = AlignCenter
		case "right":
			align = AlignRight
		}
	}
	if blockTags[t.name] {
		p.flush(false)
		switch t.name {
		case "ul", "ol":
			k := Bullet
			if t.name == "ol" {
				k = Numbered
			}
			p.lists = append(p.lists, k)
			return
		case "li":
			k := Bullet
			if n := len(p.lists); n > 0 {
				k = p.lists[n-1]
			}
			p.kind, p.level = k, min(max(len(p.lists)-1, 0), MaxLevel)
		case "h1", "h2", "h3", "h4", "h5", "h6":
			p.kind, p.level = Heading, int(t.name[1]-'0')
		case "pre":
			p.pre++
			p.kind, p.level = Paragraph, 0
		case "body", "html", "table", "tr", "section", "article", "header", "footer":
			return
		default:
			if len(p.lists) > 0 {
				// A paragraph inside a list item is the item's text.
				return
			}
			p.kind, p.level = Paragraph, 0
		}
		p.align = AlignLeft
		if hasAlign {
			p.align = align
		}
		p.begin()
		p.open = true
		p.push(t)
		return
	}
	switch t.name {
	case "br":
		k, lv, al := p.kind, p.level, p.align
		if p.cur != nil {
			k, lv, al = p.cur.Kind, p.cur.Level, p.cur.Align
		}
		p.flush(true)
		p.kind, p.level, p.align = k, lv, al
		p.begin()
		p.open = true
		return
	case "img":
		p.image(t)
		return
	case "hr", "meta", "link", "input", "col", "wbr", "source":
		return
	}
	if !t.self {
		p.push(t)
	}
}

// push opens an element's character style.
func (p *htmlParser) push(t htmlToken) {
	st := p.style()
	switch t.name {
	case "b", "strong":
		st.Bold = true
	case "i", "em", "cite", "var", "dfn":
		st.Italic = true
	case "u", "ins":
		st.Underline = true
	case "s", "strike", "del":
		st.Strike = true
	case "code", "tt", "kbd", "samp", "pre":
		st.Mono = true
	case "mark":
		st.Highlight = paintengine2d.RGB(1, 0.92, 0.23)
	case "a":
		if h, ok := t.attrs["href"]; ok {
			st.Link = h
		}
	case "font":
		if c, ok := ParseColor(t.attrs["color"]); ok {
			st.Color = c
		}
		if v, err := strconv.Atoi(strings.TrimSpace(t.attrs["size"])); err == nil {
			// <font size> is 1 to 7 on the browser's own scale.
			sizes := []float32{10, 13, 16, 18, 24, 32, 48}
			st.Size = sizes[min(max(v, 1), 7)-1]
		}
	}
	applyCSS(&st, t.attrs["style"])
	p.styles = append(p.styles, styleFrame{t.name, st})
}

func (p *htmlParser) end(name string) {
	if name == "head" {
		p.head = max(p.head-1, 0)
		return
	}
	if p.head > 0 {
		return
	}
	// Close the innermost open element of that name and everything opened
	// inside it (tolerating tags closed out of order).
	for i := len(p.styles) - 1; i >= 0; i-- {
		if p.styles[i].name == name {
			p.styles = p.styles[:i]
			break
		}
	}
	if !blockTags[name] {
		return
	}
	switch name {
	case "ul", "ol":
		p.flush(false)
		if n := len(p.lists); n > 0 {
			p.lists = p.lists[:n-1]
		}
		if len(p.lists) == 0 {
			p.kind, p.level = Paragraph, 0
		} else {
			p.kind, p.level = p.lists[len(p.lists)-1], len(p.lists)-1
		}
	case "pre":
		p.flush(false)
		p.pre = max(p.pre-1, 0)
	case "p", "div", "blockquote":
		if len(p.lists) > 0 {
			return
		}
		p.flush(false)
	default:
		p.flush(false)
	}
	if len(p.lists) == 0 {
		p.kind, p.level = Paragraph, 0
	}
	p.align = AlignLeft
}

func (p *htmlParser) image(t htmlToken) {
	src := t.attrs["src"]
	im := &Image{Src: src, Alt: t.attrs["alt"]}
	im.W = cssLength(t.attrs["width"], 0)
	im.H = cssLength(t.attrs["height"], 0)
	css := cssProps(t.attrs["style"])
	if v, ok := css["width"]; ok {
		im.W = cssLength(v, 0)
	}
	if v, ok := css["height"]; ok {
		im.H = cssLength(v, 0)
	}
	im.Pixels = DecodeDataURI(src)
	if im.Pixels == nil && p.resolve != nil && src != "" {
		im.Pixels = p.resolve(src)
	}
	p.space = false
	p.begin()
	p.spans = append(p.spans, Span{Text: string(ObjectChar), Style: p.style(), Image: im})
}

// DecodeDataURI decodes a data: URI holding a PNG, JPEG or GIF (nil for
// anything else).
func DecodeDataURI(src string) *paintengine2d.Image {
	if !strings.HasPrefix(src, "data:") {
		return nil
	}
	comma := strings.IndexByte(src, ',')
	if comma < 0 || !strings.Contains(src[:comma], ";base64") {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(src[comma+1:]))
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	return paintengine2d.NewImageFromNRGBA(img)
}

// ImageDataURI encodes pixels as a PNG data: URI.
func ImageDataURI(img *paintengine2d.Image) string {
	if img == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// cssProps splits a style attribute into lower-case properties.
func cssProps(s string) map[string]string {
	out := map[string]string{}
	for _, decl := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important"))
	}
	return out
}

// applyCSS applies the character properties of a style attribute.
func applyCSS(st *Style, s string) {
	if s == "" {
		return
	}
	for k, v := range cssProps(s) {
		lv := strings.ToLower(v)
		switch k {
		case "font-weight":
			if n, err := strconv.Atoi(lv); err == nil {
				st.Bold = n >= 600
			} else {
				st.Bold = lv == "bold" || lv == "bolder"
			}
		case "font-style":
			st.Italic = lv == "italic" || lv == "oblique"
		case "text-decoration", "text-decoration-line":
			if lv == "none" {
				st.Underline, st.Strike = false, false
			}
			if strings.Contains(lv, "underline") {
				st.Underline = true
			}
			if strings.Contains(lv, "line-through") {
				st.Strike = true
			}
		case "font-family":
			if strings.Contains(lv, "mono") || strings.Contains(lv, "courier") {
				st.Mono = true
			}
		case "font-size":
			base := st.Size
			if base <= 0 {
				base = 16
			}
			if px := cssLength(lv, base); px > 0 {
				st.Size = px
			}
		case "color":
			if c, ok := ParseColor(lv); ok {
				st.Color = c
			}
		case "background-color", "background":
			if c, ok := ParseColor(lv); ok {
				st.Highlight = c
			}
		}
	}
}

// cssLength reads a CSS length in pixels (px, pt, em relative to base, %
// of base, or a bare number); 0 when it cannot.
func cssLength(s string, base float32) float32 {
	s = strings.TrimSpace(strings.ToLower(s))
	num := func(t string) float32 {
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 32)
		if err != nil || f < 0 {
			return 0
		}
		return float32(f)
	}
	switch {
	case strings.HasSuffix(s, "px"):
		return num(s[:len(s)-2])
	case strings.HasSuffix(s, "pt"):
		return num(s[:len(s)-2]) * 4 / 3
	case strings.HasSuffix(s, "em"):
		if base <= 0 {
			return 0
		}
		return num(s[:len(s)-2]) * base
	case strings.HasSuffix(s, "%"):
		if base <= 0 {
			return 0
		}
		return num(s[:len(s)-1]) * base / 100
	}
	named := map[string]float32{"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18, "x-large": 24, "xx-large": 32}
	if v, ok := named[s]; ok {
		return v
	}
	return num(s)
}

// ParseColor reads a CSS colour: #rgb, #rrggbb, #rrggbbaa, rgb(), rgba()
// or one of the basic names.
func ParseColor(s string) (paintengine2d.Color, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "transparent" || s == "inherit" || s == "initial" {
		return paintengine2d.Color{}, false
	}
	if strings.HasPrefix(s, "#") {
		h := s[1:]
		if len(h) == 3 || len(h) == 4 {
			var e strings.Builder
			for _, c := range h {
				e.WriteRune(c)
				e.WriteRune(c)
			}
			h = e.String()
		}
		if len(h) != 6 && len(h) != 8 {
			return paintengine2d.Color{}, false
		}
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return paintengine2d.Color{}, false
		}
		a := float32(1)
		if len(h) == 8 {
			a = float32(v&0xff) / 255
			v >>= 8
		}
		return paintengine2d.RGBA(float32(v>>16&0xff)/255, float32(v>>8&0xff)/255, float32(v&0xff)/255, a), true
	}
	if strings.HasPrefix(s, "rgb") {
		open, close := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
		if open < 0 || close < open {
			return paintengine2d.Color{}, false
		}
		parts := strings.FieldsFunc(s[open+1:close], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(parts) < 3 {
			return paintengine2d.Color{}, false
		}
		ch := func(t string, scale float64) float32 {
			if strings.HasSuffix(t, "%") {
				f, _ := strconv.ParseFloat(strings.TrimSuffix(t, "%"), 64)
				return float32(math.Min(math.Max(f/100, 0), 1))
			}
			f, _ := strconv.ParseFloat(t, 64)
			return float32(math.Min(math.Max(f/scale, 0), 1))
		}
		a := float32(1)
		if len(parts) > 3 {
			a = ch(parts[3], 1)
		}
		return paintengine2d.RGBA(ch(parts[0], 255), ch(parts[1], 255), ch(parts[2], 255), a), true
	}
	if hex, ok := colorNames[s]; ok {
		return ParseColor(hex)
	}
	return paintengine2d.Color{}, false
}

var colorNames = map[string]string{
	"black": "#000000", "white": "#ffffff", "red": "#ff0000", "green": "#008000", "blue": "#0000ff",
	"yellow": "#ffff00", "orange": "#ffa500", "purple": "#800080", "gray": "#808080", "grey": "#808080",
	"silver": "#c0c0c0", "maroon": "#800000", "olive": "#808000", "lime": "#00ff00", "aqua": "#00ffff",
	"teal": "#008080", "navy": "#000080", "fuchsia": "#ff00ff", "magenta": "#ff00ff", "cyan": "#00ffff",
	"pink": "#ffc0cb", "brown": "#a52a2a",
}

// ColorHex writes c as #rrggbb (#rrggbbaa when translucent).
func ColorHex(c paintengine2d.Color) string {
	b := func(v float32) int { return int(math.Round(float64(min(max(v, 0), 1)) * 255)) }
	if c.A < 1 {
		return fmt.Sprintf("#%02x%02x%02x%02x", b(c.R), b(c.G), b(c.B), b(c.A))
	}
	return fmt.Sprintf("#%02x%02x%02x", b(c.R), b(c.G), b(c.B))
}

// ---- writing -----------------------------------------------------------

// HTML writes the document in the subset of HTML above: blocks one per
// line, lists nested in their items.
func (d *Doc) HTML() string {
	var b strings.Builder
	type openList struct{ kind Kind }
	var stack []openList // one per open list, deepest last
	liOpen := false
	closeTo := func(depth int) {
		for len(stack) > depth {
			if liOpen {
				b.WriteString("</li>")
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top.kind == Numbered {
				b.WriteString("</ol>")
			} else {
				b.WriteString("</ul>")
			}
			liOpen = len(stack) > 0
			if len(stack) == 0 {
				b.WriteByte('\n')
			}
		}
	}
	for _, bl := range d.blocks {
		if !bl.Kind.IsList() {
			closeTo(0)
			tag := "p"
			if bl.Kind == Heading {
				tag = "h" + itoa(min(max(bl.Level, 1), 6))
			}
			b.WriteString("<" + tag + alignAttr(bl.Align) + ">")
			writeSpans(&b, bl.Spans)
			b.WriteString("</" + tag + ">\n")
			continue
		}
		depth := min(max(bl.Level, 0), MaxLevel) + 1
		if len(stack) > depth {
			closeTo(depth)
		}
		if len(stack) == depth && stack[depth-1].kind != bl.Kind {
			closeTo(depth - 1)
		}
		if len(stack) == depth && liOpen {
			b.WriteString("</li>")
			liOpen = false
		}
		for len(stack) < depth {
			tag := "<ul>"
			if bl.Kind == Numbered {
				tag = "<ol>"
			}
			if len(stack) < depth-1 {
				// A level skipped: open its list in a bare item.
				if bl.Kind == Numbered {
					tag = "<ol>"
				}
				b.WriteString(tag + "<li>")
				stack = append(stack, openList{bl.Kind})
				liOpen = true
				continue
			}
			b.WriteString(tag)
			stack = append(stack, openList{bl.Kind})
			liOpen = false
		}
		b.WriteString("<li" + alignAttr(bl.Align) + ">")
		writeSpans(&b, bl.Spans)
		liOpen = true
	}
	closeTo(0)
	return b.String()
}

func alignAttr(a Align) string {
	switch a {
	case AlignCenter:
		return ` style="text-align:center"`
	case AlignRight:
		return ` style="text-align:right"`
	}
	return ""
}

// writeSpans writes a block's spans: each span's tags open and close
// around it, in a fixed order.
func writeSpans(b *strings.Builder, spans []Span) {
	// A space HTML would drop — at the block's start or end, or after
	// another — is written as a non-breaking one, which reads back as a
	// space.
	prevSpace := true
	for si, s := range spans {
		st := s.Style
		var open, close []string
		tag := func(o, c string) {
			open = append(open, o)
			close = append([]string{c}, close...)
		}
		if st.Link != "" {
			tag(`<a href="`+html.EscapeString(st.Link)+`">`, "</a>")
		}
		var css []string
		if st.Size > 0 {
			css = append(css, "font-size:"+strconv.FormatFloat(float64(st.Size), 'f', -1, 32)+"px")
		}
		if st.Color.A > 0 {
			css = append(css, "color:"+ColorHex(st.Color))
		}
		if st.Highlight.A > 0 {
			css = append(css, "background-color:"+ColorHex(st.Highlight))
		}
		if len(css) > 0 {
			tag(`<span style="`+strings.Join(css, ";")+`">`, "</span>")
		}
		if st.Bold {
			tag("<b>", "</b>")
		}
		if st.Italic {
			tag("<i>", "</i>")
		}
		if st.Underline {
			tag("<u>", "</u>")
		}
		if st.Strike {
			tag("<s>", "</s>")
		}
		if st.Mono {
			tag("<code>", "</code>")
		}
		for _, o := range open {
			b.WriteString(o)
		}
		if s.Image != nil {
			writeImage(b, s.Image)
			prevSpace = false
		} else {
			prevSpace = writeText(b, s.Text, prevSpace, si == len(spans)-1)
		}
		for _, c := range close {
			b.WriteString(c)
		}
	}
}

// writeText escapes text, keeping the spaces HTML would collapse as
// non-breaking ones; it reports whether t ended in a space.
func writeText(b *strings.Builder, t string, prevSpace, last bool) bool {
	for i, r := range t {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case ' ':
			if prevSpace || last && i == len(t)-1 {
				b.WriteString("&nbsp;")
			} else {
				b.WriteByte(' ')
			}
		case '\t':
			b.WriteString("&nbsp;&nbsp;&nbsp;&nbsp;")
		default:
			b.WriteRune(r)
		}
		prevSpace = r == ' ' || r == '\t'
	}
	return prevSpace
}

func writeImage(b *strings.Builder, im *Image) {
	src := im.Src
	if src == "" && im.Pixels != nil {
		src = ImageDataURI(im.Pixels)
	}
	b.WriteString(`<img src="` + html.EscapeString(src) + `"`)
	if im.Alt != "" {
		b.WriteString(` alt="` + html.EscapeString(im.Alt) + `"`)
	}
	if im.W > 0 {
		b.WriteString(` width="` + strconv.FormatFloat(float64(im.W), 'f', -1, 32) + `"`)
	}
	if im.H > 0 {
		b.WriteString(` height="` + strconv.FormatFloat(float64(im.H), 'f', -1, 32) + `"`)
	}
	b.WriteString(">")
}
