package richtext

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
)

// Pos is a place in a document: before character Off of block Block.
type Pos struct{ Block, Off int }

// Less reports whether p comes before q.
func (p Pos) Less(q Pos) bool { return p.Block < q.Block || p.Block == q.Block && p.Off < q.Off }

// Selection is the selected run: Anchor where it started, Caret where it
// ends and the caret blinks. They are equal when nothing is selected.
type Selection struct{ Anchor, Caret Pos }

// Range is the selection in document order.
func (s Selection) Range() (a, c Pos) {
	if s.Caret.Less(s.Anchor) {
		return s.Caret, s.Anchor
	}
	return s.Anchor, s.Caret
}

// Empty reports a collapsed selection: a caret only.
func (s Selection) Empty() bool { return s.Anchor == s.Caret }

// editKind is what an undo step did, which decides whether the next edit
// joins it.
type editKind uint8

const (
	editOther editKind = iota
	editTyping
	editBackspace
	editDelete
)

// step is one entry in the history: the blocks at at that an edit replaced
// (old) and what replaced them (now), and the selection either side.
type step struct {
	kind          editKind
	at            int
	old, now      []*Block
	before, after Selection
	lastRune      rune
	sealed        bool
	// group holds the steps of a transaction (Transact), undone and
	// redone together.
	group []step
}

// Doc is a rich-text document with a selection and an edit history.
//
// Every edit goes through the selection, as a user's does: InsertText
// replaces it, ToggleBold formats it, DeleteBackward takes it away. The
// history groups the way people type — a word and the spaces after it are
// one step, a run of Backspaces another — and moving the caret closes the
// group, so Undo takes back what the user thinks of as one action.
type Doc struct {
	blocks []*Block
	sel    Selection
	// typing is the style the next character typed at a collapsed caret
	// takes: set by a format command with nothing selected (Ctrl+B, then
	// type), forgotten when the caret moves.
	typing *Style
	undo   []step
	redo   []step
	// MaxUndo caps the history (0: 1000 steps).
	MaxUndo int
	// ResolveImage turns an image's Src that is not a data: URI into
	// pixels when HTML is loaded (SetHTML); nil leaves such images as
	// placeholders that still save their Src.
	ResolveImage func(src string) *paintengine2d.Image
	rev          uint64
	watchers     []watcher
	nextWatch    int
	// starts[i] is block i's first character as a document offset (each
	// block ends in one newline); valid below startsOK.
	starts   []int
	startsOK int
	inTx     int
}

type watcher struct {
	id int
	fn func(at, removed, added int)
}

// New is an empty document: one empty paragraph.
func New() *Doc {
	return &Doc{blocks: []*Block{NewBlock(Paragraph, 0)}}
}

// NewFromBlocks is a document holding blocks (an empty paragraph when
// there are none).
func NewFromBlocks(blocks ...*Block) *Doc {
	d := New()
	if len(blocks) > 0 {
		d.blocks = append([]*Block(nil), blocks...)
	}
	return d
}

// NewPlain is a document of plain text, a paragraph per line.
func NewPlain(text string) *Doc {
	d := New()
	d.blocks = plainBlocks(text, Style{}, Paragraph, 0)
	return d
}

func plainBlocks(text string, st Style, k Kind, level int) []*Block {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), "\n")
	out := make([]*Block, len(lines))
	for i, ln := range lines {
		out[i] = NewBlock(k, level, Span{Text: ln, Style: st})
	}
	return out
}

// Len is the number of blocks.
func (d *Doc) Len() int { return len(d.blocks) }

// Block is block i.
func (d *Doc) Block(i int) *Block { return d.blocks[i] }

// Blocks are the document's blocks. Read them; change the document through
// its methods or ReplaceBlocks.
func (d *Doc) Blocks() []*Block { return d.blocks }

// Rev counts the document's changes: it moves with every edit, undo and
// redo, and a view compares it to know whether to look again.
func (d *Doc) Rev() uint64 { return d.rev }

// Watch calls fn after every change with where it happened: removed blocks
// at at were replaced by added new ones. The returned func stops it.
func (d *Doc) Watch(fn func(at, removed, added int)) (stop func()) {
	d.nextWatch++
	id := d.nextWatch
	d.watchers = append(d.watchers, watcher{id, fn})
	return func() {
		for i, w := range d.watchers {
			if w.id == id {
				d.watchers = append(d.watchers[:i], d.watchers[i+1:]...)
				return
			}
		}
	}
}

// ---- positions ---------------------------------------------------------

// clamp keeps p inside the document.
func (d *Doc) clamp(p Pos) Pos {
	if p.Block < 0 {
		return Pos{}
	}
	if p.Block >= len(d.blocks) {
		last := len(d.blocks) - 1
		return Pos{last, d.blocks[last].n}
	}
	if p.Off < 0 {
		p.Off = 0
	}
	if n := d.blocks[p.Block].n; p.Off > n {
		p.Off = n
	}
	return p
}

// End is the position after the last character.
func (d *Doc) End() Pos {
	last := len(d.blocks) - 1
	return Pos{last, d.blocks[last].n}
}

// Offset is p as a character offset into Text.
func (d *Doc) Offset(p Pos) int {
	p = d.clamp(p)
	d.ensureStarts(p.Block + 1)
	return d.starts[p.Block] + p.Off
}

// PosAt is the position of character offset off in Text.
func (d *Doc) PosAt(off int) Pos {
	if off <= 0 {
		return Pos{}
	}
	d.ensureStarts(len(d.blocks))
	i := sort.Search(len(d.blocks), func(i int) bool { return d.starts[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	return d.clamp(Pos{i, off - d.starts[i]})
}

// TextLen is the length of Text in characters.
func (d *Doc) TextLen() int { return d.Offset(d.End()) }

func (d *Doc) ensureStarts(n int) {
	if len(d.starts) != len(d.blocks) {
		d.starts = append(d.starts[:0], make([]int, len(d.blocks))...)
		d.startsOK = 0
	}
	if d.startsOK >= n {
		return
	}
	i := d.startsOK
	off := 0
	if i > 0 {
		off = d.starts[i-1] + d.blocks[i-1].n + 1
	}
	for ; i < n; i++ {
		d.starts[i] = off
		off += d.blocks[i].n + 1
	}
	d.startsOK = n
}

// ---- selection ---------------------------------------------------------

// Selection is the current selection.
func (d *Doc) Selection() Selection { return d.sel }

// HasSelection reports whether any characters are selected.
func (d *Doc) HasSelection() bool { return !d.sel.Empty() }

// SetSelection selects from anchor to caret (clamped). Moving the caret
// closes the current undo group and forgets a pending typing style.
func (d *Doc) SetSelection(anchor, caret Pos) {
	s := Selection{d.clamp(anchor), d.clamp(caret)}
	if s != d.sel {
		d.typing = nil
		d.seal()
	}
	d.sel = s
}

// SetCaret collapses the selection at p.
func (d *Doc) SetCaret(p Pos) { d.SetSelection(p, p) }

// SelectAll selects the whole document.
func (d *Doc) SelectAll() { d.SetSelection(Pos{}, d.End()) }

// seal closes the undo group, so the next edit starts a step of its own.
func (d *Doc) seal() {
	if n := len(d.undo); n > 0 {
		d.undo[n-1].sealed = true
	}
}

// SealUndo closes the current undo group.
func (d *Doc) SealUndo() { d.seal() }

// ---- reading -----------------------------------------------------------

// Text is the document's characters: a newline between blocks, an image
// as ObjectChar. Offsets into it are what Offset and PosAt count, and what
// assistive technology reads.
func (d *Doc) Text() string {
	var b strings.Builder
	for i, bl := range d.blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, s := range bl.Spans {
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

// PlainText is the document as plain text for export and the clipboard's
// text flavour: a line per block, list items marked ("• ", "1. ") and
// indented two spaces a level, images as their alternative text.
func (d *Doc) PlainText() string {
	nums := d.Numbers()
	var b strings.Builder
	for i, bl := range d.blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		if bl.Kind.IsList() {
			b.WriteString(strings.Repeat("  ", bl.Level))
			if bl.Kind == Bullet {
				b.WriteString("• ")
			} else {
				b.WriteString(NumberLabel(nums[i], bl.Level))
				b.WriteByte(' ')
			}
		}
		for _, s := range bl.Spans {
			if s.Image != nil {
				b.WriteString(s.Image.Alt)
				continue
			}
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

// Numbers is each block's number in its numbered list (1 for the first
// item), 0 for blocks that are not numbered items. A list restarts after
// any block that is not a list item, and a nested list restarts under
// each item of the level above it.
func (d *Doc) Numbers() []int {
	out := make([]int, len(d.blocks))
	var count [MaxLevel + 1]int
	for i, b := range d.blocks {
		if !b.Kind.IsList() {
			count = [MaxLevel + 1]int{}
			continue
		}
		lv := min(max(b.Level, 0), MaxLevel)
		for j := lv + 1; j <= MaxLevel; j++ {
			count[j] = 0
		}
		if b.Kind == Numbered {
			count[lv]++
			out[i] = count[lv]
		} else {
			count[lv] = 0
		}
	}
	return out
}

// NumberLabel is how item n of a numbered list at level is marked: 1. at
// the top, a. under it, i. under that, and round again.
func NumberLabel(n, level int) string {
	switch level % 3 {
	case 1:
		return alphaLabel(n) + "."
	case 2:
		return romanLabel(n) + "."
	}
	return itoa(n) + "."
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func alphaLabel(n int) string {
	if n <= 0 {
		return "a"
	}
	var out []byte
	for n > 0 {
		n--
		out = append([]byte{byte('a' + n%26)}, out...)
		n /= 26
	}
	return string(out)
}

func romanLabel(n int) string {
	if n <= 0 || n >= 4000 {
		return itoa(n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

// Slice is a copy of the document from a to c: partial blocks at either
// end keep their kind, level and alignment.
func (d *Doc) Slice(a, c Pos) *Doc {
	a, c = d.clamp(a), d.clamp(c)
	if c.Less(a) {
		a, c = c, a
	}
	out := &Doc{}
	for i := a.Block; i <= c.Block; i++ {
		b := d.blocks[i]
		lo, hi := 0, b.n
		if i == a.Block {
			lo = a.Off
		}
		if i == c.Block {
			hi = c.Off
		}
		out.blocks = append(out.blocks, b.with(sliceSpans(b, lo, hi)))
	}
	return out
}

// SelectionDoc is the selected part of the document.
func (d *Doc) SelectionDoc() *Doc {
	a, c := d.sel.Range()
	return d.Slice(a, c)
}

// SelectedText is the selection as plain text (PlainText's form).
func (d *Doc) SelectedText() string {
	if d.sel.Empty() {
		return ""
	}
	return d.SelectionDoc().PlainText()
}

// ---- history -----------------------------------------------------------

// CanUndo and CanRedo report whether there is something to take back or
// do again.
func (d *Doc) CanUndo() bool { return len(d.undo) > 0 }
func (d *Doc) CanRedo() bool { return len(d.redo) > 0 }

// ClearHistory forgets every step (a freshly loaded document).
func (d *Doc) ClearHistory() { d.undo, d.redo = nil, nil }

// Undo takes back the last step and restores the selection it started
// from; it reports whether there was one.
func (d *Doc) Undo() bool {
	n := len(d.undo)
	if n == 0 {
		return false
	}
	s := d.undo[n-1]
	d.undo = d.undo[:n-1]
	if s.group != nil {
		for i := len(s.group) - 1; i >= 0; i-- {
			g := s.group[i]
			d.replace(g.at, len(g.now), g.old)
		}
	} else {
		d.replace(s.at, len(s.now), s.old)
	}
	d.sel = Selection{d.clamp(s.before.Anchor), d.clamp(s.before.Caret)}
	d.typing = nil
	s.sealed = true
	d.redo = append(d.redo, s)
	return true
}

// Redo does the last undone step again.
func (d *Doc) Redo() bool {
	n := len(d.redo)
	if n == 0 {
		return false
	}
	s := d.redo[n-1]
	d.redo = d.redo[:n-1]
	if s.group != nil {
		for _, g := range s.group {
			d.replace(g.at, len(g.old), g.now)
		}
	} else {
		d.replace(s.at, len(s.old), s.now)
	}
	d.sel = Selection{d.clamp(s.after.Anchor), d.clamp(s.after.Caret)}
	d.typing = nil
	d.undo = append(d.undo, s)
	return true
}

// replace swaps blocks [at, at+n) for nb and tells the watchers.
func (d *Doc) replace(at, n int, nb []*Block) {
	tail := append([]*Block(nil), d.blocks[at+n:]...)
	d.blocks = append(append(d.blocks[:at], nb...), tail...)
	if len(d.blocks) == 0 {
		d.blocks = []*Block{NewBlock(Paragraph, 0)}
	}
	d.startsOK = min(d.startsOK, at)
	if len(d.starts) != len(d.blocks) {
		d.startsOK = 0
	}
	d.rev++
	for _, w := range append([]watcher(nil), d.watchers...) {
		w.fn(at, n, len(nb))
	}
}

// edit replaces blocks [at, at+n) with nb as one undoable step, leaves the
// selection at after, and joins the step to the last one when both are
// the same kind of typing in the same block.
func (d *Doc) edit(kind editKind, at, n int, nb []*Block, after Selection, r rune) {
	before := d.sel
	old := append([]*Block(nil), d.blocks[at:at+n]...)
	d.replace(at, n, nb)
	d.sel = Selection{d.clamp(after.Anchor), d.clamp(after.Caret)}
	d.redo = nil
	if k := len(d.undo); k > 0 && kind != editOther {
		last := &d.undo[k-1]
		joins := !last.sealed && last.kind == kind && last.at == at &&
			len(last.now) == 1 && n == 1 && len(nb) == 1 && old[0] == last.now[0]
		if kind == editTyping && isSpace(last.lastRune) && !isSpace(r) {
			// A word and the spaces after it are one step; the next word
			// starts another, as GTK and Word group typing.
			joins = false
		}
		if joins {
			last.now = append([]*Block(nil), nb...)
			last.after = d.sel
			last.lastRune = r
			return
		}
	}
	d.undo = append(d.undo, step{kind: kind, at: at, old: old, now: append([]*Block(nil), nb...), before: before, after: d.sel, lastRune: r})
	limit := d.MaxUndo
	if limit <= 0 {
		limit = 1000
	}
	if len(d.undo) > limit && d.inTx == 0 {
		d.undo = append(d.undo[:0], d.undo[len(d.undo)-limit:]...)
	}
}

// Transact runs fn and makes every edit it makes one undo step.
func (d *Doc) Transact(fn func()) {
	n := len(d.undo)
	before := d.sel
	d.inTx++
	fn()
	d.inTx--
	switch k := len(d.undo) - n; {
	case k == 1:
		d.undo[n].sealed = true
	case k > 1:
		group := append([]step(nil), d.undo[n:]...)
		d.undo = append(d.undo[:n], step{group: group, before: before, after: d.sel, sealed: true})
	}
}

// Move moves the text from a to c to the place to, as one step, and
// selects it there (a drag of the selection dropped in the same
// document). It reports false, and does nothing, when to is inside the
// text moved.
func (d *Doc) Move(a, c, to Pos) bool {
	a, c, to = d.clamp(a), d.clamp(c), d.clamp(to)
	if c.Less(a) {
		a, c = c, a
	}
	if a == c || !to.Less(a) && !c.Less(to) {
		return false
	}
	frag := d.Slice(a, c)
	if c.Less(to) {
		if to.Block == c.Block {
			to = Pos{a.Block, a.Off + to.Off - c.Off}
		} else {
			to.Block -= c.Block - a.Block
		}
	}
	d.Transact(func() {
		d.sel = Selection{a, c}
		d.DeleteSelection()
		d.sel = Selection{to, to}
		d.InsertDoc(frag)
		d.sel = Selection{to, d.sel.Caret}
	})
	return true
}

func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }

// ReplaceBlocks swaps blocks [at, at+n) for blocks as one undoable step
// (the way to change blocks an application built itself). The selection
// is kept where it still fits.
func (d *Doc) ReplaceBlocks(at, n int, blocks ...*Block) {
	at = min(max(at, 0), len(d.blocks))
	n = min(max(n, 0), len(d.blocks)-at)
	sel := d.sel
	d.edit(editOther, at, n, blocks, sel, 0)
	d.sel = Selection{d.clamp(sel.Anchor), d.clamp(sel.Caret)}
}

// Reset replaces the whole document with blocks and forgets the history.
func (d *Doc) Reset(blocks []*Block) {
	if len(blocks) == 0 {
		blocks = []*Block{NewBlock(Paragraph, 0)}
	}
	d.replace(0, len(d.blocks), blocks)
	d.sel = Selection{}
	d.typing = nil
	d.ClearHistory()
}

// ---- typing and pasting ------------------------------------------------

// InsertStyle is the style the next character typed would take: a pending
// typing style, else the character before the caret's (the one after it at
// a block's start). A link does not reach past its end: text typed there
// is plain.
func (d *Doc) InsertStyle() Style {
	if d.typing != nil {
		return *d.typing
	}
	a, _ := d.sel.Range()
	b := d.blocks[a.Block]
	if b.n == 0 {
		return Style{}
	}
	if a.Off == 0 {
		st := b.StyleAt(0)
		st.Link = ""
		return st
	}
	st := b.StyleAt(a.Off - 1)
	if st.Link != "" && (a.Off >= b.n || b.StyleAt(a.Off).Link != st.Link) {
		st.Link = ""
	}
	return st
}

// InsertText types s over the selection in the insert style. A newline
// starts a new block of the same kind, as Return does.
func (d *Doc) InsertText(s string) {
	if s == "" {
		d.DeleteSelection()
		return
	}
	a, _ := d.sel.Range()
	cur := d.blocks[a.Block]
	st := d.InsertStyle()
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	if !strings.Contains(s, "\n") {
		r, _ := utf8.DecodeLastRuneInString(s)
		kind := editOther
		if utf8.RuneCountInString(s) == 1 && d.sel.Empty() {
			kind = editTyping
		}
		d.insert([]*Block{cur.with([]Span{{Text: s, Style: st}})}, kind, r, true)
		d.typing = nil
		return
	}
	lines := strings.Split(s, "\n")
	frag := make([]*Block, len(lines))
	for i, ln := range lines {
		b := cur.with([]Span{{Text: ln, Style: st}})
		if i > 0 && b.Kind == Heading {
			b.Kind, b.Level = Paragraph, 0
		}
		frag[i] = b
	}
	d.insert(frag, editOther, '\n', true)
	d.typing = nil
}

// InsertDoc pastes f over the selection: its first block joins the one
// the caret is in, its last takes the rest of that block with it.
func (d *Doc) InsertDoc(f *Doc) {
	if f == nil || len(f.blocks) == 0 {
		return
	}
	frag := make([]*Block, len(f.blocks))
	for i, b := range f.blocks {
		frag[i] = b.clone()
	}
	d.insert(frag, editOther, 0, false)
}

// InsertImage puts an image at the caret, in place of the selection.
func (d *Doc) InsertImage(im *Image) {
	if im == nil {
		return
	}
	a, _ := d.sel.Range()
	st := d.InsertStyle()
	d.insert([]*Block{d.blocks[a.Block].with([]Span{{Text: string(ObjectChar), Style: st, Image: im}})}, editOther, 0, true)
}

// insert replaces the selection with the blocks of frag: the first joins
// the block the selection starts in, the last takes the text after the
// selection, and whole blocks between are put in as they are. keepKind
// keeps the first block's kind (typing).
func (d *Doc) insert(frag []*Block, kind editKind, r rune, keepKind bool) {
	a, c := d.sel.Range()
	first, last := d.blocks[a.Block], d.blocks[c.Block]
	head := sliceSpans(first, 0, a.Off)
	tail := sliceSpans(last, c.Off, last.n)
	out := make([]*Block, 0, len(frag))
	b0 := first.clone()
	if !keepKind && a.Off == 0 && (len(frag) > 1 || c.Off == last.n) && (frag[0].n > 0 || len(frag) > 1) {
		// The fragment's first block starts the block and, unless more
		// blocks follow, fills it: it brings its own form (a heading
		// pasted onto an empty line). Pasted into the middle of a block
		// it takes that block's.
		b0.Kind, b0.Level, b0.Align = frag[0].Kind, frag[0].Level, frag[0].Align
	}
	var caret Pos
	if len(frag) == 1 {
		nb := b0.with(append(append(head, frag[0].Spans...), tail...))
		out = append(out, nb)
		caret = Pos{a.Block, a.Off + frag[0].n}
	} else {
		out = append(out, b0.with(append(head, frag[0].Spans...)))
		out = append(out, frag[1:len(frag)-1]...)
		lf := frag[len(frag)-1]
		out = append(out, lf.with(append(append([]Span(nil), lf.Spans...), tail...)))
		caret = Pos{a.Block + len(frag) - 1, lf.n}
	}
	d.edit(kind, a.Block, c.Block-a.Block+1, out, Selection{caret, caret}, r)
}

// Split breaks the block at the caret, as Return does: the new block is of
// the same kind, except that a heading is followed by a paragraph, and
// Return on an empty list item leaves the list (or goes up a level)
// instead of making another.
func (d *Doc) Split() {
	a, _ := d.sel.Range()
	cur := d.blocks[a.Block]
	if d.sel.Empty() && cur.Kind.IsList() && cur.n == 0 {
		nb := cur.clone()
		if nb.Level > 0 {
			nb.Level--
		} else {
			nb.Kind, nb.Level = Paragraph, 0
		}
		d.edit(editOther, a.Block, 1, []*Block{nb}, d.sel, 0)
		return
	}
	keep := d.InsertStyle()
	next := cur.with(nil)
	if cur.Kind == Heading && a.Off >= cur.n {
		next.Kind, next.Level, next.Align = Paragraph, 0, AlignLeft
	}
	d.insert([]*Block{cur.with(nil), next}, editOther, '\n', true)
	if keep != (Style{}) {
		// Formatting carries on into the new block, as it does in every
		// word processor.
		d.typing = &keep
	}
}

// ---- deleting ----------------------------------------------------------

// DeleteSelection removes the selected text; it reports whether there was
// any.
func (d *Doc) DeleteSelection() bool {
	if d.sel.Empty() {
		return false
	}
	a, c := d.sel.Range()
	first, last := d.blocks[a.Block], d.blocks[c.Block]
	nb := first.with(append(sliceSpans(first, 0, a.Off), sliceSpans(last, c.Off, last.n)...))
	d.edit(editOther, a.Block, c.Block-a.Block+1, []*Block{nb}, Selection{a, a}, 0)
	return true
}

// DeleteBackward is Backspace: the selection, else the character (or with
// word, the word) before the caret. At a block's start it first takes the
// block's form away — a list item goes up a level and then out of the
// list, a heading becomes a paragraph — and only then joins the block to
// the one before.
func (d *Doc) DeleteBackward(word bool) {
	if d.DeleteSelection() {
		return
	}
	p := d.sel.Caret
	cur := d.blocks[p.Block]
	if p.Off > 0 {
		q := p.Off - 1
		if word {
			q = wordLeft([]rune(cur.Text()), p.Off)
		}
		r := []rune(cur.Text())[p.Off-1]
		d.edit(editBackspace, p.Block, 1, []*Block{spliced(cur, q, p.Off, nil)}, Selection{Pos{p.Block, q}, Pos{p.Block, q}}, r)
		return
	}
	switch {
	case cur.Kind.IsList() && cur.Level > 0:
		nb := cur.clone()
		nb.Level--
		d.edit(editOther, p.Block, 1, []*Block{nb}, d.sel, 0)
	case cur.Kind != Paragraph:
		nb := cur.clone()
		nb.Kind, nb.Level = Paragraph, 0
		d.edit(editOther, p.Block, 1, []*Block{nb}, d.sel, 0)
	case p.Block > 0:
		prev := d.blocks[p.Block-1]
		nb := prev.with(append(append([]Span(nil), prev.Spans...), cur.Spans...))
		at := Pos{p.Block - 1, prev.n}
		d.edit(editOther, p.Block-1, 2, []*Block{nb}, Selection{at, at}, 0)
	}
}

// DeleteForward is Delete: the selection, else the character (or word)
// after the caret, and at a block's end the next block joins this one.
func (d *Doc) DeleteForward(word bool) {
	if d.DeleteSelection() {
		return
	}
	p := d.sel.Caret
	cur := d.blocks[p.Block]
	if p.Off < cur.n {
		q := p.Off + 1
		if word {
			q = wordRight([]rune(cur.Text()), p.Off)
		}
		d.edit(editDelete, p.Block, 1, []*Block{spliced(cur, p.Off, q, nil)}, d.sel, 0)
		return
	}
	if p.Block+1 < len(d.blocks) {
		next := d.blocks[p.Block+1]
		nb := cur.with(append(append([]Span(nil), cur.Spans...), next.Spans...))
		d.edit(editOther, p.Block, 2, []*Block{nb}, d.sel, 0)
	}
}

// ---- words -------------------------------------------------------------

// wordClass sorts characters for word movement: spaces, word characters,
// and everything else (punctuation, an image).
func wordClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\'':
		return 1
	}
	return 2
}

// wordLeft is where Ctrl+Left goes from i: back over spaces, then over one
// run of word characters (or of punctuation).
func wordLeft(rs []rune, i int) int {
	for i > 0 && wordClass(rs[i-1]) == 0 {
		i--
	}
	if i == 0 {
		return 0
	}
	c := wordClass(rs[i-1])
	for i > 0 && wordClass(rs[i-1]) == c {
		i--
	}
	return i
}

// wordRight is where Ctrl+Right goes from i: over one run of word
// characters (or punctuation), then over the spaces after it.
func wordRight(rs []rune, i int) int {
	n := len(rs)
	if i < n && wordClass(rs[i]) != 0 {
		c := wordClass(rs[i])
		for i < n && wordClass(rs[i]) == c {
			i++
		}
	}
	for i < n && wordClass(rs[i]) == 0 {
		i++
	}
	return i
}

// WordLeft and WordRight are where Ctrl+Left and Ctrl+Right take the caret
// from p: across block boundaries at a block's ends.
func (d *Doc) WordLeft(p Pos) Pos {
	p = d.clamp(p)
	if p.Off == 0 {
		if p.Block == 0 {
			return p
		}
		return Pos{p.Block - 1, d.blocks[p.Block-1].n}
	}
	return Pos{p.Block, wordLeft([]rune(d.blocks[p.Block].Text()), p.Off)}
}

func (d *Doc) WordRight(p Pos) Pos {
	p = d.clamp(p)
	b := d.blocks[p.Block]
	if p.Off >= b.n {
		if p.Block+1 >= len(d.blocks) {
			return p
		}
		return Pos{p.Block + 1, 0}
	}
	return Pos{p.Block, wordRight([]rune(b.Text()), p.Off)}
}

// WordAt is the word around p (a double click's selection): the run of
// characters of the same class as the one under p.
func (d *Doc) WordAt(p Pos) (a, c Pos) {
	p = d.clamp(p)
	rs := []rune(d.blocks[p.Block].Text())
	if len(rs) == 0 {
		return p, p
	}
	i := min(p.Off, len(rs)-1)
	if p.Off == len(rs) || (p.Off > 0 && wordClass(rs[i]) == 0 && wordClass(rs[p.Off-1]) != 0) {
		i = p.Off - 1
	}
	cl := wordClass(rs[i])
	lo, hi := i, i+1
	for lo > 0 && wordClass(rs[lo-1]) == cl {
		lo--
	}
	for hi < len(rs) && wordClass(rs[hi]) == cl {
		hi++
	}
	return Pos{p.Block, lo}, Pos{p.Block, hi}
}

// ---- formatting --------------------------------------------------------

// Format applies fn to the style of the selected text. With nothing
// selected it sets the style the next typed text takes instead.
func (d *Doc) Format(fn func(*Style)) {
	if d.sel.Empty() {
		st := d.InsertStyle()
		fn(&st)
		d.typing = &st
		return
	}
	a, c := d.sel.Range()
	out := make([]*Block, 0, c.Block-a.Block+1)
	for i := a.Block; i <= c.Block; i++ {
		b := d.blocks[i]
		lo, hi := 0, b.n
		if i == a.Block {
			lo = a.Off
		}
		if i == c.Block {
			hi = c.Off
		}
		out = append(out, restyled(b, lo, hi, fn))
	}
	d.edit(editOther, a.Block, len(out), out, d.sel, 0)
}

// SelectionStyle is the style the selection starts with (the insert style
// with nothing selected): what a format bar shows.
func (d *Doc) SelectionStyle() Style {
	if d.sel.Empty() {
		return d.InsertStyle()
	}
	a, c := d.sel.Range()
	b := d.blocks[a.Block]
	if a.Off >= b.n && a.Block < c.Block {
		return d.blocks[a.Block+1].StyleAt(0)
	}
	return b.StyleAt(a.Off)
}

// all reports whether every selected character's style passes pred (the
// insert style with nothing selected).
func (d *Doc) all(pred func(Style) bool) bool {
	if d.sel.Empty() {
		return pred(d.InsertStyle())
	}
	a, c := d.sel.Range()
	any := false
	for i := a.Block; i <= c.Block; i++ {
		b := d.blocks[i]
		lo, hi := 0, b.n
		if i == a.Block {
			lo = a.Off
		}
		if i == c.Block {
			hi = c.Off
		}
		for _, s := range sliceSpans(b, lo, hi) {
			any = true
			if !pred(s.Style) {
				return false
			}
		}
	}
	return any
}

// ToggleBold, ToggleItalic, ToggleUnderline, ToggleStrike and ToggleMono
// switch the attribute on for the whole selection, or off when it is on
// all of it already.
func (d *Doc) ToggleBold() {
	on := !d.all(func(s Style) bool { return s.Bold })
	d.Format(func(s *Style) { s.Bold = on })
}

func (d *Doc) ToggleItalic() {
	on := !d.all(func(s Style) bool { return s.Italic })
	d.Format(func(s *Style) { s.Italic = on })
}

func (d *Doc) ToggleUnderline() {
	on := !d.all(func(s Style) bool { return s.Underline })
	d.Format(func(s *Style) { s.Underline = on })
}

func (d *Doc) ToggleStrike() {
	on := !d.all(func(s Style) bool { return s.Strike })
	d.Format(func(s *Style) { s.Strike = on })
}

func (d *Doc) ToggleMono() {
	on := !d.all(func(s Style) bool { return s.Mono })
	d.Format(func(s *Style) { s.Mono = on })
}

// SetSize sets the text size in logical pixels (0: the block's own).
func (d *Doc) SetSize(px float32) { d.Format(func(s *Style) { s.Size = max(px, 0) }) }

// SetColor sets the text colour (a zero colour: the look's).
func (d *Doc) SetColor(c paintengine2d.Color) { d.Format(func(s *Style) { s.Color = c }) }

// SetHighlight sets the colour behind the text (a zero colour: none).
func (d *Doc) SetHighlight(c paintengine2d.Color) { d.Format(func(s *Style) { s.Highlight = c }) }

// ClearFormat takes every character attribute off the selection.
func (d *Doc) ClearFormat() {
	d.Format(func(s *Style) { link := s.Link; *s = Style{Link: link} })
}

// SetLink makes the selection a link to href (an empty href removes the
// link). With nothing selected on a link it changes that whole link.
func (d *Doc) SetLink(href string) {
	if d.sel.Empty() {
		p := d.sel.Caret
		b := d.blocks[p.Block]
		off := p.Off
		if off >= b.n && off > 0 {
			off--
		}
		lo, hi, _, ok := b.LinkRange(off)
		if !ok {
			return
		}
		keep := d.sel
		d.sel = Selection{Pos{p.Block, lo}, Pos{p.Block, hi}}
		d.Format(func(s *Style) { s.Link = href })
		d.sel = keep
		if n := len(d.undo); n > 0 {
			d.undo[n-1].before, d.undo[n-1].after = keep, keep
		}
		return
	}
	d.Format(func(s *Style) { s.Link = href })
}

// LinkAt is the link at p (checking the character after p, then the one
// before), or "".
func (d *Doc) LinkAt(p Pos) string {
	p = d.clamp(p)
	b := d.blocks[p.Block]
	if p.Off < b.n {
		if l := b.StyleAt(p.Off).Link; l != "" {
			return l
		}
	}
	if p.Off > 0 {
		return b.StyleAt(p.Off - 1).Link
	}
	return ""
}

// ---- block form --------------------------------------------------------

// blocksTouched is the range of blocks the selection touches.
func (d *Doc) blocksTouched() (lo, hi int) {
	a, c := d.sel.Range()
	return a.Block, c.Block
}

// reform applies fn to every block the selection touches, as one step.
func (d *Doc) reform(fn func(b *Block)) {
	lo, hi := d.blocksTouched()
	out := make([]*Block, 0, hi-lo+1)
	changed := false
	for i := lo; i <= hi; i++ {
		nb := d.blocks[i].clone()
		fn(nb)
		if !nb.Equal(d.blocks[i]) {
			changed = true
		}
		out = append(out, nb)
	}
	if changed {
		d.edit(editOther, lo, len(out), out, d.sel, 0)
	}
}

// SetKind makes every block the selection touches a k (Heading at level;
// list items at their level, 0 for a paragraph that becomes one).
func (d *Doc) SetKind(k Kind, level int) {
	d.reform(func(b *Block) {
		switch {
		case k == Heading:
			b.Kind, b.Level = Heading, min(max(level, 1), 6)
		case k.IsList():
			if !b.Kind.IsList() {
				b.Level = min(max(level, 0), MaxLevel)
			}
			b.Kind = k
		default:
			b.Kind, b.Level = Paragraph, 0
		}
	})
}

// BlockKind is the kind and level of the block the caret is in.
func (d *Doc) BlockKind() (Kind, int) {
	b := d.blocks[d.sel.Caret.Block]
	return b.Kind, b.Level
}

// ToggleList makes the touched blocks list items of kind k, or paragraphs
// when they all are such items already.
func (d *Doc) ToggleList(k Kind) {
	lo, hi := d.blocksTouched()
	all := true
	for i := lo; i <= hi; i++ {
		if d.blocks[i].Kind != k {
			all = false
		}
	}
	if all {
		d.SetKind(Paragraph, 0)
		return
	}
	d.SetKind(k, 0)
}

// Indent moves the touched list items delta levels in (negative: out);
// it reports whether any was a list item.
func (d *Doc) Indent(delta int) bool {
	lo, hi := d.blocksTouched()
	any := false
	for i := lo; i <= hi; i++ {
		if d.blocks[i].Kind.IsList() {
			any = true
		}
	}
	if !any {
		return false
	}
	d.reform(func(b *Block) {
		if b.Kind.IsList() {
			b.Level = min(max(b.Level+delta, 0), MaxLevel)
		}
	})
	return true
}

// SetAlign aligns the touched blocks.
func (d *Doc) SetAlign(a Align) { d.reform(func(b *Block) { b.Align = a }) }
