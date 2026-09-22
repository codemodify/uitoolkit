package widgets

import (
	"math"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// RichText is a rich-text editor (Qt's QTextEdit, GTK's GtkTextView with
// tags): bold, italic, underline, strikethrough, monospace, sizes, text
// and highlight colours, headings, bulleted and numbered lists that nest,
// alignment, links and inline images, over a richtext.Doc the application
// reads and writes (HTML in and out, plain text out).
//
// Editing follows TextArea — the same keys, mouse, wrap and scroll bar —
// and adds what a document needs: double-click selects a word and
// triple-click a paragraph (drag on to extend by words or paragraphs), the
// selection drags and drops (a move inside the editor, text and HTML to
// other applications), the clipboard carries HTML beside the plain text,
// and Ctrl+Z / Ctrl+Shift+Z undo and redo in steps a person would expect.
// Formatting keys are the word processors': Ctrl+B, I and U, Alt+Shift+5
// strikethrough, Ctrl+Alt+1–6 headings and Ctrl+Alt+0 body text,
// Ctrl+Shift+7 and 8 numbered and bulleted lists, Ctrl+] and Ctrl+[ list
// levels, Ctrl+Shift+L, E and R alignment, Ctrl+K a link, Ctrl+\ clears
// formatting. Ctrl+click follows a link (a click, read-only).
//
// RichTextBar is the optional formatting tool bar; place it where the app
// wants it.
type RichText struct {
	widget.Base
	// Placeholder shows while the document is empty and unfocused.
	Placeholder string
	// ReadOnly shows the document without letting it be edited; links
	// follow on a plain click.
	ReadOnly bool
	// MinRows is the height the editor asks for, in body lines (0: 6).
	MinRows int
	// OnChange runs after every change to the document.
	OnChange func()
	// OnSelectionChange runs when the caret or the selection moves.
	OnSelectionChange func()
	// OnLink is asked to follow a link (Ctrl+click, a click when read-only,
	// Return on a link from assistive technology).
	OnLink func(href string)
	// OnLinkRequest, when set, is asked for a link's target when the user
	// asks to make one (Ctrl+K, the tool bar), with the current one; it
	// calls done with the new target, or not at all to cancel. Without it
	// the editor asks in a small dialog of its own.
	OnLinkRequest func(current string, done func(href string))
	// ResolveImage turns an image's Src into pixels when HTML is loaded
	// (the document's own data: URIs need nothing).
	ResolveImage func(src string) *paintengine2d.Image

	doc       *richtext.Doc
	stopWatch func()
	listeners []func()

	// layout (richtext_layout.go)
	cache    map[*richtext.Block]*rtLayout
	faces    map[rtFaceKey]*style.Font
	heights  []float32
	tops     []float32
	topsOK   int
	laidW    float32
	laidSig  rtSig
	avgW     float32
	laidOut  int
	idleNext int
	idleStop func()
	pinTop   bool

	scrollY  float32
	vbar     scrollDrag
	blinkOn  bool
	caretUp  bool
	preferX  float32
	havePref bool
	lastSel  richtext.Selection
	lastRev  uint64

	// pointer
	dragging  bool
	unit      int // what a drag extends by: 0 characters, 1 words, 2 blocks
	unitA     richtext.Pos
	unitB     richtext.Pos
	clicks    int
	clickAt   time.Time
	clickPos  paintengine2d.Point
	dragSel   bool
	dragAt    richtext.Pos
	dragAtUp  bool
	selfDrop  bool
	dropAt    richtext.Pos
	dropShown bool
	hoverLink string
}

// NewRichText is an editor over an empty document.
func NewRichText(placeholder string) *RichText {
	t := &RichText{Placeholder: placeholder, blinkOn: true}
	t.Init(t)
	t.SetWantsFocus(true)
	t.SetDocument(richtext.New())
	return t
}

// NewRichTextHTML is an editor over the document html describes.
func NewRichTextHTML(html string) *RichText {
	t := NewRichText("")
	t.SetHTML(html)
	return t
}

// Document is the document being edited. Edits made to it directly show
// in the editor.
func (t *RichText) Document() *richtext.Doc { return t.doc }

// SetDocument edits d instead.
func (t *RichText) SetDocument(d *richtext.Doc) {
	if d == nil {
		d = richtext.New()
	}
	if t.stopWatch != nil {
		t.stopWatch()
	}
	t.doc = d
	if d.ResolveImage == nil && t.ResolveImage != nil {
		d.ResolveImage = t.ResolveImage
	}
	t.stopWatch = d.Watch(t.blocksChanged)
	t.heights, t.tops, t.topsOK = nil, nil, 0
	clear(t.cache)
	t.scrollY = 0
	t.havePref, t.caretUp = false, false
	t.lastRev, t.lastSel = d.Rev(), d.Selection()
	t.Invalidate()
	t.notify(true)
}

// SetHTML loads the document from HTML (the subset package richtext
// documents) and forgets the undo history.
func (t *RichText) SetHTML(html string) {
	if t.ResolveImage != nil {
		t.doc.ResolveImage = t.ResolveImage
	}
	_ = t.doc.SetHTML(html)
	t.scrollY = 0
	t.edited()
}

// HTML saves the document as HTML.
func (t *RichText) HTML() string { return t.doc.HTML() }

// SetPlainText replaces the document with text, a paragraph per line.
func (t *RichText) SetPlainText(s string) {
	t.doc.Reset(richtext.NewPlain(s).Blocks())
	t.scrollY = 0
	t.edited()
}

// PlainText is the document as plain text (list markers, images' alt).
func (t *RichText) PlainText() string { return t.doc.PlainText() }

// SelectedText is the selection as plain text.
func (t *RichText) SelectedText() string { return t.doc.SelectedText() }

// Selection is the selection as character offsets into Document().Text().
func (t *RichText) Selection() (anchor, caret int) {
	s := t.doc.Selection()
	return t.doc.Offset(s.Anchor), t.doc.Offset(s.Caret)
}

// SetSelection selects between two character offsets.
func (t *RichText) SetSelection(anchor, caret int) {
	t.doc.SetSelection(t.doc.PosAt(anchor), t.doc.PosAt(caret))
	t.caretUp, t.havePref = false, false
	t.ensureCaretVisible()
	t.Invalidate()
	t.notify(false)
}

// SetCaretBlink turns the caret's blink phase on or off (tests, stills).
func (t *RichText) SetCaretBlink(on bool) { t.blinkOn = on }

// Listen adds fn to what runs on every change and every caret move (the
// format bar keeps its buttons in step this way). It returns a func that
// stops it.
func (t *RichText) Listen(fn func()) (stop func()) {
	t.listeners = append(t.listeners, fn)
	i := len(t.listeners) - 1
	return func() {
		if i < len(t.listeners) {
			t.listeners[i] = nil
		}
	}
}

// notify tells the application and the listeners that the document or
// the selection moved.
func (t *RichText) notify(content bool) {
	if content && t.OnChange != nil {
		t.OnChange()
	}
	if s := t.doc.Selection(); s != t.lastSel || content {
		t.lastSel = s
		if t.OnSelectionChange != nil {
			t.OnSelectionChange()
		}
	}
	for _, fn := range t.listeners {
		if fn != nil {
			fn()
		}
	}
}

func (t *RichText) editable() bool { return t.Enabled() && !t.ReadOnly }

// edited follows a change to the document: the caret into view, a
// repaint, and the callbacks.
func (t *RichText) edited() {
	t.havePref, t.caretUp = false, false
	t.lastRev = t.doc.Rev()
	t.ensureCaretVisible()
	t.Invalidate()
	t.notify(true)
}

// moved follows a caret or selection move.
func (t *RichText) moved() {
	t.ensureCaretVisible()
	t.Invalidate()
	t.notify(false)
}

// ---- commands ----------------------------------------------------------

// run applies an edit to the document when the editor is editable.
func (t *RichText) run(fn func(d *richtext.Doc)) {
	if !t.editable() {
		return
	}
	fn(t.doc)
	t.edited()
}

func (t *RichText) ToggleBold()      { t.run((*richtext.Doc).ToggleBold) }
func (t *RichText) ToggleItalic()    { t.run((*richtext.Doc).ToggleItalic) }
func (t *RichText) ToggleUnderline() { t.run((*richtext.Doc).ToggleUnderline) }
func (t *RichText) ToggleStrike()    { t.run((*richtext.Doc).ToggleStrike) }
func (t *RichText) ToggleMono()      { t.run((*richtext.Doc).ToggleMono) }
func (t *RichText) ClearFormat()     { t.run((*richtext.Doc).ClearFormat) }

// SetSize sets the selection's text size in logical pixels (0: default).
func (t *RichText) SetSize(px float32) { t.run(func(d *richtext.Doc) { d.SetSize(px) }) }

// SetColor sets the selection's text colour (zero: the look's).
func (t *RichText) SetColor(c paintengine2d.Color) { t.run(func(d *richtext.Doc) { d.SetColor(c) }) }

// SetHighlight sets the colour behind the selection (zero: none).
func (t *RichText) SetHighlight(c paintengine2d.Color) {
	t.run(func(d *richtext.Doc) { d.SetHighlight(c) })
}

// SetLink links the selection to href ("" removes the link).
func (t *RichText) SetLink(href string) { t.run(func(d *richtext.Doc) { d.SetLink(href) }) }

// SetKind makes the touched blocks paragraphs, headings (of level) or list
// items.
func (t *RichText) SetKind(k richtext.Kind, level int) {
	t.run(func(d *richtext.Doc) { d.SetKind(k, level) })
}

// ToggleList makes the touched blocks list items of kind k, or back into
// paragraphs.
func (t *RichText) ToggleList(k richtext.Kind) { t.run(func(d *richtext.Doc) { d.ToggleList(k) }) }

// Indent moves list items a level in (delta 1) or out (-1).
func (t *RichText) Indent(delta int) { t.run(func(d *richtext.Doc) { d.Indent(delta) }) }

// SetAlign aligns the touched blocks.
func (t *RichText) SetAlign(a richtext.Align) { t.run(func(d *richtext.Doc) { d.SetAlign(a) }) }

// InsertImage puts img at the caret.
func (t *RichText) InsertImage(img *richtext.Image) {
	t.run(func(d *richtext.Doc) { d.InsertImage(img) })
}

// InsertText types s at the caret, over the selection.
func (t *RichText) InsertText(s string) { t.run(func(d *richtext.Doc) { d.InsertText(s) }) }

// Undo and Redo walk the history.
func (t *RichText) Undo() { t.run(func(d *richtext.Doc) { d.Undo() }) }
func (t *RichText) Redo() { t.run(func(d *richtext.Doc) { d.Redo() }) }

// Copy puts the selection on the clipboard, as HTML and as plain text.
func (t *RichText) Copy() {
	if !t.doc.HasSelection() {
		return
	}
	f := t.doc.SelectionDoc()
	SetClipboardHTML(f.HTML(), f.PlainText())
}

// Cut copies the selection and takes it away.
func (t *RichText) Cut() {
	if !t.doc.HasSelection() || !t.editable() {
		return
	}
	t.Copy()
	t.run(func(d *richtext.Doc) { d.DeleteSelection() })
}

// Paste inserts the clipboard: its HTML when it has some, its text
// otherwise. plain pastes the text alone (Ctrl+Shift+V).
func (t *RichText) Paste(plain bool) {
	if !t.editable() {
		return
	}
	if html, ok := ClipboardHTML(); ok && !plain {
		f := richtext.FragmentHTML(html, t.doc.ResolveImage)
		t.run(func(d *richtext.Doc) { d.InsertDoc(f) })
		return
	}
	if s := platform.ClipboardGet(); s != "" {
		t.run(func(d *richtext.Doc) { d.InsertText(s) })
	}
}

// RequestLink asks for the target of a link over the selection (Ctrl+K):
// OnLinkRequest when set, else a small dialog of the editor's own.
func (t *RichText) RequestLink() {
	if !t.editable() {
		return
	}
	cur := t.doc.LinkAt(t.doc.Selection().Caret)
	if !t.doc.HasSelection() && cur == "" {
		// Nothing to link: select the word under the caret first.
		a, c := t.doc.WordAt(t.doc.Selection().Caret)
		if a == c {
			return
		}
		t.doc.SetSelection(a, c)
		t.moved()
	}
	sel := t.doc.Selection()
	apply := func(href string) {
		t.doc.SetSelection(sel.Anchor, sel.Caret)
		t.SetLink(strings.TrimSpace(href))
		t.RequestFocus()
	}
	if t.OnLinkRequest != nil {
		t.OnLinkRequest(cur, apply)
		return
	}
	showLinkDialog(t, cur, apply)
}

// showLinkDialog asks for a link's target in a small modal card.
func showLinkDialog(from widget.Component, current string, done func(string)) {
	field := NewTextField(current, "https://", nil)
	var ov *Overlay
	ok := NewButton("OK", func() {
		widget.DismissOverlay(ov)
		done(field.Text)
	})
	ok.Primary = true
	field.OnSubmit = func(string) { ok.OnClick() }
	cancel := NewButton("Cancel", func() { widget.DismissOverlay(ov) })
	remove := NewButton("Remove link", func() {
		widget.DismissOverlay(ov)
		done("")
	})
	remove.SetEnabled(current != "")
	bb := NewButtonBox().AddButton(ok, RoleAccept).AddButton(cancel, RoleReject).AddButton(remove, RoleDestructive)
	card := NewPanel("Link", NewLabel("Address").For(field), field, bb)
	card.Window = true
	card.Raised = true
	card.OnClose = func() { widget.DismissOverlay(ov) }
	ov = NewOverlay(card)
	ov.Modal = true
	ov.MinCardW = 360
	ov.InitialFocus = field
	widget.ShowOverlay(from, ov)
}

// ---- geometry ----------------------------------------------------------

func (t *RichText) fieldPad() float32 {
	p := t.Look().Metrics().FieldPad
	if p <= 0 {
		p = 8
	}
	return p
}

func (t *RichText) inner() paintengine2d.Rect {
	pad := t.fieldPad()
	b := t.LocalBounds()
	return paintengine2d.XYWH(pad, pad, max(b.Dx()-pad*2, 0), max(b.Dy()-pad*2, 0))
}

func (t *RichText) lineH() float32 {
	f := t.Look().Font()
	return f.Height() + t.dip(2)
}

func (t *RichText) Measure(c layout.Constraints) paintengine2d.Point {
	rows := t.MinRows
	if rows <= 0 {
		rows = 6
	}
	h := float32(rows)*t.lineH() + t.fieldPad()*2
	w := t.dip(320)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *RichText) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	t.syncGeometry()
	t.clampScroll()
}

func (t *RichText) maxScroll() float32 { return layout.MaxScroll(t.docH(), t.inner().Dy()) }

func (t *RichText) clampScroll() {
	t.scrollY = layout.ClampScroll(t.scrollY, t.docH(), t.inner().Dy())
}

// ScrollOffset is the vertical scroll in device pixels; MaxScroll its
// limit.
func (t *RichText) ScrollOffset() float32 { return t.scrollY }
func (t *RichText) MaxScroll() float32    { return t.maxScroll() }

// ScrollTo scrolls to y (clamped).
func (t *RichText) ScrollTo(y float32) {
	t.scrollY = y
	t.clampScroll()
	t.Invalidate()
}

func (t *RichText) vparts() style.ScrollParts {
	in := t.inner()
	box := paintengine2d.XYWH(0, in.Min.Y, t.LocalBounds().Dx(), in.Dy())
	return style.ScrollGeometry(t.Look(), box, true, t.docH(), in.Dy(), t.scrollY, false)
}

func (t *RichText) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    t.vparts,
		get:      func() (float32, float32) { return t.scrollY, t.maxScroll() },
		set:      func(y float32) { t.scrollY = y; t.clampScroll(); t.Invalidate() },
		steps:    func() (float32, float32) { return t.lineH(), t.inner().Dy() * 0.9 },
	}
}

// caretBox is the caret's box in document coordinates (x from the
// content's left, y from the document's top).
func (t *RichText) caretBox(p richtext.Pos, up bool) paintengine2d.Rect {
	lay := t.lay(p.Block)
	li := lay.lineFor(p.Off, up)
	ln := &lay.lines[li]
	top := t.top(p.Block) + ln.y
	x := ln.caretX(p.Off)
	asc, desc := ln.ascent, ln.descent
	if f := ln.fragAt(p.Off); f != nil && f.img == nil {
		asc, desc = f.font.Ascent, fontDescent(f.font)
	} else if len(ln.frags) == 0 {
		asc, desc = lay.base.Ascent, fontDescent(lay.base)
	}
	y := top + ln.base - asc
	return paintengine2d.XYWH(x, y, max(1.6, t.dip(1.6)), asc+desc)
}

// posAt is the document position under local point p, and its affinity.
func (t *RichText) posAt(p paintengine2d.Point) (richtext.Pos, bool) {
	in := t.inner()
	y := p.Y - in.Min.Y + t.scrollY
	x := p.X - in.Min.X
	if y < 0 {
		return richtext.Pos{}, false
	}
	if y >= t.docH() {
		return t.doc.End(), false
	}
	i := t.blockAt(y)
	lay := t.lay(i)
	by := y - t.top(i)
	li := 0
	for li < len(lay.lines)-1 && by >= lay.lines[li].y+lay.lines[li].h {
		li++
	}
	off, up := lay.lines[li].offsetAt(x)
	return richtext.Pos{Block: i, Off: off}, up
}

// ensureCaretVisible scrolls the caret's line into view (with the room
// above a block's first line, so a heading comes with its spacing).
func (t *RichText) ensureCaretVisible() {
	in := t.inner()
	if in.Dy() <= 0 || t.doc == nil {
		return
	}
	p := t.doc.Selection().Caret
	lay := t.lay(p.Block)
	li := lay.lineFor(p.Off, t.caretUp)
	ln := lay.lines[li]
	top := t.top(p.Block) + ln.y
	if li == 0 {
		top = t.top(p.Block)
	}
	bottom := t.top(p.Block) + ln.y + ln.h
	if top < t.scrollY {
		t.scrollY = top
	}
	if bottom > t.scrollY+in.Dy() {
		t.scrollY = bottom - in.Dy()
	}
	t.clampScroll()
}

// CaretRect is the caret's box in local coordinates (an input method's
// anchor, tests).
func (t *RichText) CaretRect() paintengine2d.Rect {
	in := t.inner()
	return t.caretBox(t.doc.Selection().Caret, t.caretUp).Translate(paintengine2d.Pt(in.Min.X, in.Min.Y-t.scrollY))
}

// PositionRect is the local box of the character at document offset off.
func (t *RichText) PositionRect(off int) paintengine2d.Rect {
	in := t.inner()
	return t.caretBox(t.doc.PosAt(off), false).Translate(paintengine2d.Pt(in.Min.X, in.Min.Y-t.scrollY))
}

// ---- painting ----------------------------------------------------------

func (t *RichText) blink() bool {
	if !t.Focused() || !t.editable() {
		return false
	}
	if h := t.Host(); h != nil {
		if b, ok := h.(interface{ CaretBlink() bool }); ok {
			return b.CaretBlink()
		}
	}
	return t.blinkOn
}

// textColors are the look's field text, link, selection, selected text
// and caret colours — TextArea's, pinned extras included.
func (t *RichText) textColors() (text, link, sel, selText, caret paintengine2d.Color) {
	lk := t.Look()
	p := lk.Palette()
	text = p.Text
	field := p.Field
	if field.A > 0 {
		text = style.ReadableOn(field, 4.5, p.Text)
	} else {
		field = p.Background
	}
	link = style.ReadableOn(field, 3, p.Accent, text)
	sel = p.Selection
	bg := sel
	if bg.A < 1 {
		bg = style.Mix(field, sel, sel.A)
	}
	selText = style.ReadableOn(bg, 4.5, text, p.TextOnAccent)
	caret = text
	if c, ok := lk.(*style.Classic); ok {
		selText = c.X("selectionText", selText)
		caret = c.X("caret", caret)
	}
	return
}

func (t *RichText) Paint(ctx *paintengine2d.Context) {
	lk := t.Look()
	b := t.LocalBounds()
	if t.doc.Rev() != t.lastRev {
		// The application edited the document directly.
		t.lastRev = t.doc.Rev()
		t.notify(true)
	} else if t.doc.Selection() != t.lastSel {
		t.notify(false)
	}
	t.syncGeometry()
	t.clampScroll()
	lk.DrawTextArea(ctx, b, t.State(), []style.TextLine{{}}, -1, 0, 0, false, 0, 0, "", lk.Font())
	in := t.inner()
	text, link, selCol, selText, caretCol := t.textColors()
	ctx.Save()
	ctx.ClipRect(in)
	if t.doc.Len() == 1 && t.doc.Block(0).Len() == 0 && t.Placeholder != "" && !t.Focused() {
		lk.MutedFont().Draw(ctx, t.Placeholder, paintengine2d.Pt(in.Min.X, in.Min.Y+t.dip(1)), lk.Palette().TextMuted)
	}
	lo, hi := t.visibleBlocks()
	sel := t.doc.Selection()
	a, c := sel.Range()
	var nums []int
	for i := lo; i < hi; i++ {
		if t.doc.Block(i).Kind == richtext.Numbered {
			nums = t.doc.Numbers()
			break
		}
	}
	for i := lo; i < hi && i < t.doc.Len(); i++ {
		lay := t.lay(i)
		oy := in.Min.Y + t.top(i) - t.scrollY
		if oy > in.Max.Y {
			break
		}
		blk := t.doc.Block(i)
		// The selection's range on this block; past its end when the
		// selection carries on into the next one.
		sa, sc := -1, -1
		if !sel.Empty() && a.Block <= i && c.Block >= i {
			sa, sc = 0, blk.Len()+1
			if a.Block == i {
				sa = a.Off
			}
			if c.Block == i {
				sc = c.Off
			}
		}
		if blk.Kind.IsList() {
			t.paintMarker(ctx, blk, lay, nums, i, in.Min.X, oy, text)
		}
		for li := range lay.lines {
			ln := &lay.lines[li]
			ly := oy + ln.y
			if ly > in.Max.Y || ly+ln.h < in.Min.Y {
				continue
			}
			t.paintLine(ctx, ln, in.Min.X, ly, sa, sc, blk.Len(), in, text, link, selCol, selText)
		}
	}
	// The caret, and where a drag would drop.
	if t.blink() && sel.Empty() {
		cb := t.caretBox(sel.Caret, t.caretUp).Translate(paintengine2d.Pt(in.Min.X, in.Min.Y-t.scrollY))
		ctx.DrawRect(cb, paintengine2d.Fill(caretCol))
	}
	if t.dropShown {
		cb := t.caretBox(t.dropAt, false).Translate(paintengine2d.Pt(in.Min.X, in.Min.Y-t.scrollY))
		style.DrawDropCaretOf(lk, ctx, paintengine2d.XYWH(cb.Min.X-t.dip(1), cb.Min.Y, max(cb.Dx(), t.dip(2)), cb.Dy()), true)
	}
	ctx.Restore()
	t.vbar.paint(t, ctx, lk, t.vparts(), true, t.scrollY)
	if t.laidOut < t.doc.Len() {
		t.scheduleIdle()
	}
}

// paintMarker draws a list item's bullet or number in the gutter before
// its first line.
func (t *RichText) paintMarker(ctx *paintengine2d.Context, blk *richtext.Block, lay *rtLayout, nums []int, i int, x0, oy float32, col paintengine2d.Color) {
	if len(lay.lines) == 0 {
		return
	}
	ln := lay.lines[0]
	f := lay.base
	if len(ln.frags) > 0 && ln.frags[0].img == nil {
		f = ln.frags[0].font
	}
	gap := t.dip(6)
	right := x0 + lay.indent - gap
	baseY := oy + ln.y + ln.base
	if blk.Kind == richtext.Numbered && nums != nil {
		label := richtext.NumberLabel(nums[i], blk.Level)
		w := f.Advance(label)
		f.Draw(ctx, label, paintengine2d.Pt(right-w, baseY-f.Ascent), col)
		return
	}
	// Bullets: a disc, a ring, a square, and round again with depth.
	r := max(f.Size*0.14, t.dip(2))
	cx := right - r - t.dip(2)
	cy := baseY - f.Ascent*0.36
	switch blk.Level % 3 {
	case 0:
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r, paintengine2d.Fill(col))
	case 1:
		lw := max(1, t.dip(1.1))
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), r-lw*0.5, paintengine2d.StrokePaint(col, lw))
	default:
		s := r * 1.7
		ctx.DrawRect(paintengine2d.XYWH(cx-s*0.5, cy-s*0.5, s, s), paintengine2d.Fill(col))
	}
}

// paintLine draws one line: highlights, the selection, the text with its
// underlines and strikes, images, and the selected text over the
// selection in its own colour.
func (t *RichText) paintLine(ctx *paintengine2d.Context, ln *rtLine, x0, ly float32, sa, sc, blen int, in paintengine2d.Rect,
	text, link, selCol, selText paintengine2d.Color) {
	lx := x0 + ln.x
	for _, f := range ln.frags {
		if f.st.Highlight.A > 0 && f.img == nil {
			w := f.w
			if f.end == ln.end && ln.soft {
				w = f.font.Advance(strings.TrimRight(f.text, " "))
			}
			ctx.DrawRect(paintengine2d.XYWH(lx+f.x, ly, w, ln.h), paintengine2d.Fill(f.st.Highlight))
		}
	}
	var selBox paintengine2d.Rect
	if sa >= 0 {
		s0, s1 := max(sa, ln.start), min(sc, ln.end)
		past := sc > blen && ln.end == blen && !ln.soft
		if s0 < s1 || past && s0 <= ln.end {
			xa := x0 + ln.caretX(s0)
			xb := x0 + ln.caretX(min(s1, ln.end))
			if past {
				// The selection runs on past the block's end: show its
				// newline as a sliver, as TextArea does.
				xb = max(xb+t.dip(6), xb)
			}
			selBox = paintengine2d.Rect{Min: paintengine2d.Pt(xa, ly), Max: paintengine2d.Pt(xb, ly+ln.h)}
			ctx.DrawRect(selBox, paintengine2d.Fill(selCol))
		}
	}
	draw := func(override bool) {
		for i := range ln.frags {
			f := &ln.frags[i]
			fx := lx + f.x
			if fx > in.Max.X || fx+f.w < in.Min.X {
				continue
			}
			if f.img != nil {
				if override {
					continue
				}
				top := ly + ln.base - f.imgH
				dst := paintengine2d.XYWH(fx, top, f.imgW, f.imgH)
				if px := f.img.Pixels; px != nil && px.Width > 0 && px.Height > 0 {
					ctx.DrawImageRectPaint(px, paintengine2d.XYWH(0, 0, float32(px.Width), float32(px.Height)), dst, paintengine2d.Paint{})
				} else {
					// A picture not loaded: its box, as a browser shows it.
					ctx.DrawRect(dst.Inset(0.5), paintengine2d.StrokePaint(text.WithAlpha(0.4), 1))
				}
				continue
			}
			col := t.fragColor(f.st, text, link)
			if f.st.Highlight.A > 0 && f.st.Color.A == 0 {
				// Text on a highlight is read against the highlight.
				col = style.ReadableOn(f.st.Highlight, 4.5, col, paintengine2d.RGB(0.1, 0.1, 0.1), paintengine2d.RGB(1, 1, 1))
			}
			if override {
				col = selText
			}
			top := ly + ln.base - f.font.Ascent
			if f.st.Italic {
				// No italic face is bundled: slant the upright one about
				// its baseline, as Pango and Qt synthesise an oblique.
				ctx.Save()
				k := float32(0.2)
				base := ly + ln.base
				ctx.Transform(paintengine2d.Matrix{A: 1, C: -k, D: 1, E: k * base})
				f.font.Draw(ctx, f.text, paintengine2d.Pt(fx, top), col)
				ctx.Restore()
			} else {
				f.font.Draw(ctx, f.text, paintengine2d.Pt(fx, top), col)
			}
			lw := max(1, float32(math.Round(float64(f.font.Size/14))))
			w := f.w
			if f.end == ln.end && ln.soft {
				w = f.font.Advance(strings.TrimRight(f.text, " "))
			}
			if f.st.Underline || f.st.Link != "" {
				uy := float32(math.Round(float64(ly + ln.base + max(fontDescent(f.font)*0.35, lw))))
				ctx.DrawRect(paintengine2d.XYWH(fx, uy, w, lw), paintengine2d.Fill(col))
			}
			if f.st.Strike {
				sy := float32(math.Round(float64(ly + ln.base - f.font.Ascent*0.32)))
				ctx.DrawRect(paintengine2d.XYWH(fx, sy, w, lw), paintengine2d.Fill(col))
			}
		}
	}
	draw(false)
	if !selBox.Empty() {
		ctx.Save()
		ctx.ClipRect(selBox)
		draw(true)
		ctx.Restore()
	}
}

// ---- keyboard ----------------------------------------------------------

func (t *RichText) FocusGained() {
	t.blinkOn = true
	t.Invalidate()
}

func (t *RichText) FocusLost() {
	t.dragging = false
	t.Invalidate()
}

func (t *RichText) TextInput(r rune) bool {
	if !t.editable() || r < 32 || r == 127 {
		return false
	}
	t.doc.InsertText(string(r))
	t.edited()
	return true
}

// setCaret moves the caret to p, extending the selection when extend.
func (t *RichText) setCaret(p richtext.Pos, extend bool) {
	s := t.doc.Selection()
	if extend {
		t.doc.SetSelection(s.Anchor, p)
	} else {
		t.doc.SetCaret(p)
	}
}

func (t *RichText) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	ctrl, shift, alt := e.Mods.Ctrl(), e.Mods.Shift(), e.Mods.Alt()
	if e.Rune == 0 {
		// A synthesised event may name only the key.
		e.Rune, _ = platform.KeyRune(e.Key)
	}
	d := t.doc
	sel := d.Selection()
	a, c := sel.Range()
	nav := func(p richtext.Pos, keepPref bool) bool {
		t.setCaret(p, shift)
		if !keepPref {
			t.havePref = false
		}
		t.moved()
		return true
	}
	switch e.Key {
	case platform.KeyLeft:
		t.caretUp = false
		switch {
		case ctrl:
			return nav(d.WordLeft(sel.Caret), false)
		case !sel.Empty() && !shift:
			return nav(a, false)
		}
		return nav(t.stepChar(sel.Caret, -1), false)
	case platform.KeyRight:
		t.caretUp = false
		switch {
		case ctrl:
			return nav(d.WordRight(sel.Caret), false)
		case !sel.Empty() && !shift:
			return nav(c, false)
		}
		return nav(t.stepChar(sel.Caret, 1), false)
	case platform.KeyUp:
		if ctrl {
			return nav(richtext.Pos{Block: max(sel.Caret.Block-boolInt(sel.Caret.Off == 0), 0)}, false)
		}
		return nav(t.stepLine(-1), true)
	case platform.KeyDown:
		if ctrl {
			nb := min(sel.Caret.Block+1, d.Len()-1)
			if sel.Caret.Block == d.Len()-1 {
				return nav(d.End(), false)
			}
			return nav(richtext.Pos{Block: nb}, false)
		}
		return nav(t.stepLine(1), true)
	case platform.KeyHome:
		t.caretUp = false
		if ctrl {
			return nav(richtext.Pos{}, false)
		}
		lay := t.lay(sel.Caret.Block)
		ln := lay.lines[lay.lineFor(sel.Caret.Off, t.caretUp)]
		return nav(richtext.Pos{Block: sel.Caret.Block, Off: ln.start}, false)
	case platform.KeyEnd:
		if ctrl {
			t.caretUp = false
			return nav(d.End(), false)
		}
		lay := t.lay(sel.Caret.Block)
		li := lay.lineFor(sel.Caret.Off, t.caretUp)
		ln := lay.lines[li]
		t.caretUp = ln.soft
		return nav(richtext.Pos{Block: sel.Caret.Block, Off: ln.end}, false)
	case platform.KeyPageUp, platform.KeyPageDown:
		dir := float32(1)
		if e.Key == platform.KeyPageUp {
			dir = -1
		}
		cb := t.caretBox(sel.Caret, t.caretUp)
		if !t.havePref {
			t.preferX, t.havePref = cb.Min.X, true
		}
		page := t.inner().Dy() * 0.9
		t.scrollY += dir * page
		t.clampScroll()
		in := t.inner()
		p, up := t.posAt(paintengine2d.Pt(in.Min.X+t.preferX, in.Min.Y+cb.Min.Y+cb.Dy()*0.5+dir*page-t.scrollY+dir*0))
		t.caretUp = up
		t.setCaret(p, shift)
		t.moved()
		return true
	}
	if t.ReadOnly || !t.Enabled() {
		return t.readOnlyKey(e)
	}
	switch e.Key {
	case platform.KeyBackspace:
		d.DeleteBackward(ctrl)
		t.edited()
		return true
	case platform.KeyDelete:
		if shift && !ctrl {
			t.Cut()
			return true
		}
		d.DeleteForward(ctrl)
		t.edited()
		return true
	case platform.KeyReturn:
		if ctrl {
			if href := d.LinkAt(sel.Caret); href != "" && t.OnLink != nil {
				t.OnLink(href)
				return true
			}
		}
		d.Split()
		t.edited()
		return true
	case platform.KeyMenu:
		t.showMenu(t.CaretRect().Min.Add(paintengine2d.Pt(0, t.CaretRect().Dy())))
		return true
	case platform.KeyF10:
		if shift {
			t.showMenu(t.CaretRect().Min.Add(paintengine2d.Pt(0, t.CaretRect().Dy())))
			return true
		}
	}
	if alt && shift && !ctrl && e.Rune == '5' {
		t.ToggleStrike()
		return true
	}
	if !ctrl {
		return false
	}
	if alt && !shift {
		switch {
		case e.Rune >= '1' && e.Rune <= '6':
			t.SetKind(richtext.Heading, int(e.Rune-'0'))
			return true
		case e.Rune == '0':
			t.SetKind(richtext.Paragraph, 0)
			return true
		}
		return false
	}
	switch e.Rune {
	case 'a':
		d.SelectAll()
		t.moved()
		return true
	case 'c':
		t.Copy()
		return true
	case 'x':
		t.Cut()
		return true
	case 'v':
		t.Paste(shift)
		return true
	case 'z':
		if shift {
			t.Redo()
		} else {
			t.Undo()
		}
		return true
	case 'y':
		t.Redo()
		return true
	case 'b':
		t.ToggleBold()
		return true
	case 'i':
		t.ToggleItalic()
		return true
	case 'u':
		t.ToggleUnderline()
		return true
	case 'k':
		t.RequestLink()
		return true
	case '\\':
		t.ClearFormat()
		return true
	case ']':
		t.Indent(1)
		return true
	case '[':
		t.Indent(-1)
		return true
	case '7':
		if shift {
			t.ToggleList(richtext.Numbered)
			return true
		}
	case '8':
		if shift {
			t.ToggleList(richtext.Bullet)
			return true
		}
	case 'l':
		if shift {
			t.SetAlign(richtext.AlignLeft)
			return true
		}
	case 'e':
		if shift {
			t.SetAlign(richtext.AlignCenter)
			return true
		}
	case 'r':
		if shift {
			t.SetAlign(richtext.AlignRight)
			return true
		}
	}
	return false
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (t *RichText) readOnlyKey(e widget.KeyEvent) bool {
	if e.Rune == 0 {
		e.Rune, _ = platform.KeyRune(e.Key)
	}
	if e.Mods.Ctrl() {
		switch e.Rune {
		case 'a':
			t.doc.SelectAll()
			t.moved()
			return true
		case 'c':
			t.Copy()
			return true
		}
	}
	if e.Key == platform.KeyReturn {
		if href := t.doc.LinkAt(t.doc.Selection().Caret); href != "" && t.OnLink != nil {
			t.OnLink(href)
			return true
		}
	}
	return false
}

// stepChar is the position dir characters from p, across block ends.
func (t *RichText) stepChar(p richtext.Pos, dir int) richtext.Pos {
	d := t.doc
	if dir < 0 {
		if p.Off > 0 {
			return richtext.Pos{Block: p.Block, Off: p.Off - 1}
		}
		if p.Block > 0 {
			return richtext.Pos{Block: p.Block - 1, Off: d.Block(p.Block - 1).Len()}
		}
		return p
	}
	if p.Off < d.Block(p.Block).Len() {
		return richtext.Pos{Block: p.Block, Off: p.Off + 1}
	}
	if p.Block+1 < d.Len() {
		return richtext.Pos{Block: p.Block + 1}
	}
	return p
}

// stepLine is the position a visual line up (dir -1) or down from the
// caret, at the column the caret keeps while it moves up and down.
func (t *RichText) stepLine(dir int) richtext.Pos {
	d := t.doc
	p := d.Selection().Caret
	lay := t.lay(p.Block)
	li := lay.lineFor(p.Off, t.caretUp)
	if !t.havePref {
		t.preferX, t.havePref = lay.lines[li].caretX(p.Off), true
	}
	bi, ni := p.Block, li+dir
	if ni < 0 {
		if bi == 0 {
			t.caretUp = false
			return richtext.Pos{}
		}
		bi--
		ni = len(t.lay(bi).lines) - 1
	} else if ni >= len(lay.lines) {
		if bi == d.Len()-1 {
			t.caretUp = false
			return d.End()
		}
		bi++
		ni = 0
	}
	ln := &t.lay(bi).lines[ni]
	off, up := ln.offsetAt(t.preferX)
	t.caretUp = up
	return richtext.Pos{Block: bi, Off: off}
}

// ---- mouse -------------------------------------------------------------

// multiClickGap is the room a second click may land from the first and
// still count as a double click.
func (t *RichText) multiClickGap() float32 { return t.dip(5) }

func (t *RichText) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	if t.vbar.press(t, e.Pos, t.vaxis()) {
		t.Invalidate()
		return true
	}
	p, up := t.posAt(e.Pos)
	switch e.Button {
	case platform.ButtonMiddle:
		if t.editable() {
			t.doc.SetCaret(p)
			if s := platform.ClipboardPrimaryGet(); s != "" {
				t.doc.InsertText(s)
			}
			t.edited()
		}
		return true
	case platform.ButtonRight:
		if !t.inSelection(p) {
			t.doc.SetCaret(p)
			t.caretUp = up
			t.moved()
		}
		t.showMenu(e.Pos)
		return true
	}
	if e.Button != platform.ButtonLeft {
		return false
	}
	// Links follow on Ctrl+click, or on a plain click when nothing can be
	// edited.
	if href := t.linkUnder(e.Pos); href != "" && (e.Mods.Ctrl() || t.ReadOnly) && t.OnLink != nil {
		t.OnLink(href)
		return true
	}
	now := time.Now()
	if now.Sub(t.clickAt) < doubleClickInterval && absf(e.Pos.X-t.clickPos.X) <= t.multiClickGap() && absf(e.Pos.Y-t.clickPos.Y) <= t.multiClickGap() {
		t.clicks++
	} else {
		t.clicks = 1
	}
	t.clickAt, t.clickPos = now, e.Pos
	if t.clicks == 1 && !e.Mods.Shift() && t.inSelection(p) {
		// A press inside the selection may be the start of a drag of it;
		// the release puts the caret here if it was only a click.
		t.dragSel, t.selfDrop, t.dragAt, t.dragAtUp = true, false, p, up
		return true
	}
	t.dragging = true
	t.havePref = false
	t.caretUp = up
	switch {
	case t.clicks >= 3:
		t.unit = 2
		t.unitA, t.unitB = richtext.Pos{Block: p.Block}, richtext.Pos{Block: p.Block, Off: t.doc.Block(p.Block).Len()}
		t.doc.SetSelection(t.unitA, t.unitB)
	case t.clicks == 2:
		t.unit = 1
		t.unitA, t.unitB = t.doc.WordAt(p)
		t.doc.SetSelection(t.unitA, t.unitB)
	default:
		t.unit = 0
		t.setCaret(p, e.Mods.Shift())
	}
	t.moved()
	return true
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// inSelection reports whether p is inside the selection.
func (t *RichText) inSelection(p richtext.Pos) bool {
	s := t.doc.Selection()
	if s.Empty() {
		return false
	}
	a, c := s.Range()
	return !p.Less(a) && p.Less(c)
}

func (t *RichText) MouseMove(e widget.MouseEvent) bool {
	if handled, dirty := t.vbar.move(t, e.Pos, t.vaxis()); handled || dirty {
		if dirty {
			t.Invalidate()
		}
		if handled {
			return true
		}
	}
	if t.dragSel {
		return true
	}
	if !t.dragging {
		if l := t.linkUnder(e.Pos); l != t.hoverLink {
			t.hoverLink = l
		}
		return false
	}
	// Past the top or bottom edge, the view scrolls along with the drag.
	in := t.inner()
	if e.Pos.Y < in.Min.Y {
		t.scrollY -= min(in.Min.Y-e.Pos.Y, t.lineH())
	} else if e.Pos.Y > in.Max.Y {
		t.scrollY += min(e.Pos.Y-in.Max.Y, t.lineH())
	}
	t.clampScroll()
	p, up := t.posAt(e.Pos)
	switch t.unit {
	case 1, 2:
		pa, pc := t.doc.WordAt(p)
		if t.unit == 2 {
			pa, pc = richtext.Pos{Block: p.Block}, richtext.Pos{Block: p.Block, Off: t.doc.Block(p.Block).Len()}
		}
		if pa.Less(t.unitA) {
			t.doc.SetSelection(t.unitB, pa)
		} else {
			t.doc.SetSelection(t.unitA, pc)
		}
	default:
		t.caretUp = up
		t.setCaret(p, true)
	}
	t.moved()
	return true
}

func (t *RichText) MouseRelease(e widget.MouseEvent) bool {
	if t.dragSel {
		t.dragSel = false
		t.havePref = false
		t.caretUp = t.dragAtUp
		t.doc.SetCaret(t.dragAt)
		t.moved()
		return true
	}
	t.dragging = false
	if t.vbar.release() {
		t.Invalidate()
	}
	return true
}

func (t *RichText) MouseExit() {
	t.vbar.exit()
	t.hoverLink = ""
	t.Base.MouseExit()
}

func (t *RichText) MouseWheel(e widget.MouseEvent) bool {
	if e.Scroll.Y == 0 {
		return false
	}
	before := t.scrollY
	t.scrollY += wheelDelta(e.Scroll.Y, t.lineH(), e.Precise)
	t.clampScroll()
	if t.scrollY == before {
		return false
	}
	t.vbar.wake(t)
	t.Invalidate()
	return true
}

// linkUnder is the link at local point p, or "".
func (t *RichText) linkUnder(p paintengine2d.Point) string {
	in := t.inner()
	if !in.Contains(p) {
		return ""
	}
	pos, _ := t.posAt(p)
	b := t.doc.Block(pos.Block)
	// posAt rounds to the nearer side of a character; the link is the
	// character under the pointer, so look at both.
	for _, off := range []int{pos.Off, pos.Off - 1} {
		if off < 0 || off >= b.Len() {
			continue
		}
		if l := b.StyleAt(off).Link; l != "" {
			box := t.caretBox(richtext.Pos{Block: pos.Block, Off: off}, false)
			nx := t.caretBox(richtext.Pos{Block: pos.Block, Off: off + 1}, false)
			x := p.X - in.Min.X
			if x >= min(box.Min.X, nx.Min.X) && x <= max(box.Min.X, nx.Min.X) {
				return l
			}
		}
	}
	return ""
}

// Tooltip is the target of the link under the pointer.
func (t *RichText) Tooltip() string { return t.hoverLink }

// CursorAt is the I-beam over the text, the arrow over the scroll bar.
func (t *RichText) CursorAt(p paintengine2d.Point) platform.Cursor {
	if t.inner().Contains(p) && !t.vparts().Track.Contains(p) {
		return platform.CursorText
	}
	return platform.CursorDefault
}

// showMenu opens the editor's context menu at local point p.
func (t *RichText) showMenu(p paintengine2d.Point) {
	has := t.doc.HasSelection()
	ed := t.editable()
	items := []*MenuItem{
		{Text: "Undo", Shortcut: "Ctrl+Z", Disabled: !ed || !t.doc.CanUndo(), OnClick: t.Undo},
		{Text: "Redo", Shortcut: "Ctrl+Shift+Z", Disabled: !ed || !t.doc.CanRedo(), OnClick: t.Redo},
		Sep(),
		{Text: "Cut", Shortcut: "Ctrl+X", Icon: style.IconCut, Disabled: !ed || !has, OnClick: t.Cut},
		{Text: "Copy", Shortcut: "Ctrl+C", Icon: style.IconCopy, Disabled: !has, OnClick: t.Copy},
		{Text: "Paste", Shortcut: "Ctrl+V", Icon: style.IconPaste, Disabled: !ed, OnClick: func() { t.Paste(false) }},
		{Text: "Paste as plain text", Shortcut: "Ctrl+Shift+V", Disabled: !ed, OnClick: func() { t.Paste(true) }},
		Sep(),
		{Text: "Select all", Shortcut: "Ctrl+A", OnClick: func() { t.doc.SelectAll(); t.moved() }},
	}
	if ed {
		items = append(items, Sep(),
			&MenuItem{Text: "Link…", Shortcut: "Ctrl+K", OnClick: t.RequestLink},
			&MenuItem{Text: "Clear formatting", Shortcut: "Ctrl+\\", OnClick: t.ClearFormat})
	}
	ShowContextMenu(t, widget.DeviceOrigin(t).Add(p), items...)
}

// afterIdle runs fn on c's UI thread soon, when there is time.
func afterIdle(c widget.Component, fn func()) (stop func()) {
	return widget.After(c, 16*time.Millisecond, fn)
}
