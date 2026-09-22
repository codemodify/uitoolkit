package widgets

import (
	"strings"
	"unicode/utf16"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/widget"
)

// ---- drag and drop -----------------------------------------------------

// DragAt drags the selection: as HTML and plain text to other
// applications, and as the document fragment itself inside this one. A
// drop in the same editor moves it (Ctrl copies, as the desktop decides);
// a move elsewhere takes it out of the document once the target says it
// performed one.
func (t *RichText) DragAt(paintengine2d.Point) *widget.Drag {
	if !t.dragSel || !t.doc.HasSelection() {
		return nil
	}
	frag := t.doc.SelectionDoc()
	plain := frag.PlainText()
	html := frag.HTML()
	d := widget.DragText(plain)
	if d == nil {
		// An image alone has no text: it still drags as HTML.
		d = widget.NewDrag(nil, nil)
		d.Actions, d.Preferred = platform.DragCopy|platform.DragMove, platform.DragCopy
	}
	text := d.Data
	d.Types = append([]string{"text/html"}, d.Types...)
	d.Data = func(mime string) ([]byte, bool) {
		if mime == "text/html" {
			return []byte(html), true
		}
		if text == nil {
			return nil, false
		}
		return text(mime)
	}
	editable := t.editable()
	if editable {
		// Text dragged in an editor moves unless Ctrl asks for a copy, as
		// in Qt's and GTK's text views.
		d.Actions, d.Preferred = platform.DragCopy|platform.DragMove, platform.DragMove
	} else {
		d.Actions, d.Preferred = platform.DragCopy, platform.DragCopy
	}
	d.Payload = frag
	d.Source = t
	label := plain
	if label == "" {
		label = "Image"
	}
	d.Image, d.Hotspot = widget.DragLabel(t.Look(), dragLabelFor(label), dragScale(t))
	a, c := t.doc.Selection().Range()
	rev := t.doc.Rev()
	d.Done = func(action platform.DragAction) {
		if action == platform.DragMove && editable && !t.selfDrop && t.doc.Rev() == rev {
			t.doc.SetSelection(a, c)
			t.doc.DeleteSelection()
			t.edited()
		}
		t.dragSel, t.selfDrop = false, false
	}
	return d
}

// DropTypes: HTML first, then text.
func (t *RichText) DropTypes() []string { return []string{"text/html", "text/plain"} }

// DropActionFor: a drag of this editor's own selection moves it — the
// compositor settles a drag's action from what the target prefers, so
// the target says it, rather than hoping the source's preference counts;
// a copy within the document is Ctrl+C and Ctrl+V. Text from elsewhere
// is copied or moved as the desktop's modifiers say.
func (t *RichText) DropActionFor(offered platform.DragAction) platform.DragAction {
	if t.dragSel && offered.Has(platform.DragMove) {
		return platform.DragMove
	}
	return dropActions(platform.DragCopy|platform.DragMove, offered)
}

// Drop inserts what was dropped where it was dropped, and selects it.
func (t *RichText) Drop(e widget.DropEvent) bool {
	t.DragLeave()
	if !t.editable() {
		return false
	}
	var frag *richtext.Doc
	switch {
	case e.Payload != nil:
		if f, ok := e.Payload.(*richtext.Doc); ok {
			frag = f
		}
	}
	if frag == nil {
		switch {
		case e.Mime == "text/html" && len(e.Data) > 0:
			frag = richtext.FragmentHTML(decodeHTMLDrop(e.Data), t.doc.ResolveImage)
		case e.Text != "":
			frag = richtext.NewPlain(e.Text)
		default:
			return false
		}
	}
	at, _ := t.posAt(e.Pos)
	if e.Source == widget.Component(t) {
		t.selfDrop = true
		a, c := t.doc.Selection().Range()
		if e.Action == platform.DragMove {
			if !t.doc.Move(a, c, at) {
				// Dropped on itself: nothing moves.
				t.doc.SetCaret(at)
			}
			t.RequestFocus()
			t.edited()
			return true
		}
	}
	t.doc.Transact(func() {
		t.doc.SetCaret(at)
		t.doc.InsertDoc(frag)
		t.doc.SetSelection(at, t.doc.Selection().Caret)
	})
	t.RequestFocus()
	t.edited()
	return true
}

// decodeHTMLDrop reads text/html as sent: UTF-8, or UTF-16 with a byte
// order mark (what Firefox puts on an X11 selection).
func decodeHTMLDrop(b []byte) string {
	if len(b) >= 2 && (b[0] == 0xff && b[1] == 0xfe || b[0] == 0xfe && b[1] == 0xff) {
		le := b[0] == 0xff
		u := make([]uint16, 0, len(b)/2)
		for i := 2; i+1 < len(b); i += 2 {
			if le {
				u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
			} else {
				u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
			}
		}
		return string(utf16.Decode(u))
	}
	return strings.TrimPrefix(string(b), "\ufeff")
}

// DragOver shows where a drop would land.
func (t *RichText) DragOver(pos paintengine2d.Point) {
	at, _ := t.posAt(pos)
	if !t.dropShown || at != t.dropAt {
		t.dropAt, t.dropShown = at, true
		t.Invalidate()
	}
}

// DragLeave hides the drop caret.
func (t *RichText) DragLeave() {
	if t.dropShown {
		t.dropShown = false
		t.Invalidate()
	}
}

// ---- accessibility -----------------------------------------------------

// Describe: a multi-line text whose value is the document's characters
// (an image as U+FFFC, the embedded-object character), with the caret
// and selection as offsets into it.
func (t *RichText) Describe(n *a11y.Node) {
	n.Role = a11y.RoleTextArea
	n.State |= a11y.StateMultiLine
	if t.ReadOnly {
		n.State |= a11y.StateReadOnly
	} else {
		n.State |= a11y.StateEditable
	}
	n.Value = t.doc.Text()
	if n.Name == "" {
		n.Name = t.Placeholder
	}
	s := t.doc.Selection()
	a, c := s.Range()
	n.Caret = t.doc.Offset(s.Caret)
	n.SelStart, n.SelEnd = t.doc.Offset(a), t.doc.Offset(c)
}

// rtLinkItem is a link on screen: its block, its characters and target.
type rtLinkItem struct {
	block, a, c int
	href        string
}

// visibleLinks are the links in the blocks on screen.
func (t *RichText) visibleLinks() []rtLinkItem {
	var out []rtLinkItem
	if t.doc == nil || t.LocalBounds().Empty() {
		return nil
	}
	lo, hi := t.visibleBlocks()
	for i := lo; i < hi && i < t.doc.Len(); i++ {
		b := t.doc.Block(i)
		for off := 0; off < b.Len(); {
			a, c, href, ok := b.LinkRange(off)
			if !ok {
				off++
				continue
			}
			out = append(out, rtLinkItem{i, a, c, href})
			off = c
		}
	}
	return out
}

// AccessibleItems are the links on screen, each a link node named by its
// text, so a screen reader can list and follow them.
func (t *RichText) AccessibleItems() []*a11y.Node {
	links := t.visibleLinks()
	out := make([]*a11y.Node, 0, len(links))
	o := widget.DeviceOrigin(t)
	for i, l := range links {
		txt := []rune(t.doc.Block(l.block).Text())
		name := strings.TrimSpace(string(txt[l.a:l.c]))
		if name == "" {
			name = l.href
		}
		r0 := t.caretBox(richtext.Pos{Block: l.block, Off: l.a}, false)
		r1 := t.caretBox(richtext.Pos{Block: l.block, Off: l.c}, true)
		in := t.inner()
		box := paintengine2d.Rect{Min: r0.Min, Max: paintengine2d.Pt(max(r1.Max.X, r0.Max.X), max(r1.Max.Y, r0.Max.Y))}
		box = box.Translate(paintengine2d.Pt(in.Min.X+o.X, in.Min.Y-t.scrollY+o.Y))
		out = append(out, &a11y.Node{
			ID: widget.ItemID(t, i), Role: a11y.RoleLink, Name: name, Description: l.href,
			Bounds: box, Actions: a11y.Actions(0).With(a11y.ActionDefault),
		})
	}
	return out
}

// AccessibleAction follows link item (ActionDefault).
func (t *RichText) AccessibleAction(item int, act a11y.Action) bool {
	if item < 0 || act != a11y.ActionDefault {
		return false
	}
	links := t.visibleLinks()
	if item >= len(links) {
		return false
	}
	t.followLink(links[item].href)
	return true
}

// AccessibleSetText replaces the document with plain text (assistive
// technology's EditableText, automation).
func (t *RichText) AccessibleSetText(s string) bool {
	if !t.editable() {
		return false
	}
	t.doc.Transact(func() {
		t.doc.SelectAll()
		t.doc.InsertText(s)
	})
	t.edited()
	return true
}
