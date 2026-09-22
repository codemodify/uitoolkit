package richtext

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// typeText types s a character at a time, as the keyboard does.
func typeText(d *Doc, s string) {
	for _, r := range s {
		if r == '\n' {
			d.Split()
			continue
		}
		d.InsertText(string(r))
	}
}

func TestTypingAndBlocks(t *testing.T) {
	d := New()
	typeText(d, "Hello world\nSecond")
	if got := d.Text(); got != "Hello world\nSecond" {
		t.Fatalf("text %q", got)
	}
	if d.Len() != 2 || d.Selection().Caret != (Pos{1, 6}) {
		t.Fatalf("blocks %d caret %v", d.Len(), d.Selection().Caret)
	}
	if off := d.Offset(Pos{1, 2}); off != 14 {
		t.Errorf("offset %d, want 14", off)
	}
	if p := d.PosAt(14); p != (Pos{1, 2}) {
		t.Errorf("pos %v", p)
	}
	// Backspace at a block's start joins it to the one before.
	d.SetCaret(Pos{1, 0})
	d.DeleteBackward(false)
	if d.Text() != "Hello worldSecond" || d.Selection().Caret != (Pos{0, 11}) {
		t.Fatalf("join: %q %v", d.Text(), d.Selection().Caret)
	}
	// Ctrl+Backspace takes a word.
	d.SetCaret(Pos{0, 5})
	d.DeleteBackward(true)
	if d.Text() != " worldSecond" {
		t.Errorf("word delete: %q", d.Text())
	}
	d.DeleteForward(true) // to the next word's start, as Qt's Ctrl+Delete
	if d.Text() != "worldSecond" {
		t.Errorf("word delete forward: %q", d.Text())
	}
}

func TestUndoGroupsWords(t *testing.T) {
	d := New()
	typeText(d, "one two three")
	d.Undo()
	if got := d.Text(); got != "one two " {
		t.Fatalf("undo took back %q, want the last word only", got)
	}
	d.Undo()
	if got := d.Text(); got != "one " {
		t.Fatalf("second undo: %q", got)
	}
	d.Redo()
	d.Redo()
	if d.Text() != "one two three" || d.Selection().Caret != (Pos{0, 13}) {
		t.Fatalf("redo: %q %v", d.Text(), d.Selection().Caret)
	}
	// Moving the caret closes the group: two runs of typing, two steps.
	d.SetCaret(Pos{0, 3})
	typeText(d, "X")
	d.SetCaret(Pos{0, 0})
	typeText(d, "Y")
	d.Undo()
	if d.Text() != "oneX two three" {
		t.Fatalf("caret move should split the group: %q", d.Text())
	}
	// A run of Backspaces is one step.
	d.SetCaret(d.End())
	for range 5 {
		d.DeleteBackward(false)
	}
	if d.Text() != "oneX two " {
		t.Fatalf("backspace: %q", d.Text())
	}
	d.Undo()
	if d.Text() != "oneX two three" {
		t.Fatalf("undo backspaces: %q", d.Text())
	}
	// A new edit forgets what could be redone.
	d.Undo()
	typeText(d, "Z")
	if d.CanRedo() {
		t.Error("redo survived a new edit")
	}
}

func TestFormatting(t *testing.T) {
	d := New()
	typeText(d, "bold and plain")
	d.SetSelection(Pos{0, 0}, Pos{0, 4})
	d.ToggleBold()
	if !d.Block(0).StyleAt(1).Bold || d.Block(0).StyleAt(6).Bold {
		t.Fatal("bold did not apply to the selection only")
	}
	d.ToggleBold()
	if d.Block(0).StyleAt(1).Bold {
		t.Fatal("toggling a bold selection turns bold off")
	}
	d.Undo()
	if !d.Block(0).StyleAt(1).Bold || len(d.Block(0).Spans) != 2 {
		t.Fatalf("undo restores the bold span: %+v", d.Block(0).Spans)
	}
	// With nothing selected a format command sets the typing style.
	d.SetCaret(d.End())
	d.ToggleItalic()
	typeText(d, "!")
	if st := d.Block(0).StyleAt(14); !st.Italic {
		t.Errorf("typed text is not italic: %+v", st)
	}
	// Colour, highlight, size and links over a mixed selection.
	d.SelectAll()
	d.SetColor(paintengine2d.RGB(1, 0, 0))
	d.SetHighlight(paintengine2d.RGB(1, 1, 0))
	d.SetSize(20)
	for _, s := range d.Block(0).Spans {
		if s.Style.Color.R != 1 || s.Style.Highlight.G != 1 || s.Style.Size != 20 {
			t.Fatalf("span %+v", s.Style)
		}
	}
	d.SetSelection(Pos{0, 5}, Pos{0, 8})
	d.SetLink("https://example.com")
	if l := d.LinkAt(Pos{0, 6}); l != "https://example.com" {
		t.Fatalf("link %q", l)
	}
	// Typing just past a link's end is not part of the link.
	d.SetCaret(Pos{0, 8})
	typeText(d, "x")
	if d.Block(0).StyleAt(8).Link != "" {
		t.Error("the link grew past its end")
	}
	// SetLink with only a caret on the link changes the whole link.
	d.SetCaret(Pos{0, 6})
	d.SetLink("")
	if d.LinkAt(Pos{0, 6}) != "" {
		t.Error("the link was not removed")
	}
}

func TestListsAndReturn(t *testing.T) {
	d := New()
	typeText(d, "item")
	d.ToggleList(Bullet)
	if k, _ := d.BlockKind(); k != Bullet {
		t.Fatal("not a bullet")
	}
	typeText(d, "\nnext")
	if d.Block(1).Kind != Bullet {
		t.Fatal("Return in a list makes another item")
	}
	d.Indent(1)
	if d.Block(1).Level != 1 {
		t.Fatal("indent")
	}
	// Return on an empty item goes up a level, then out of the list.
	typeText(d, "\n")
	d.Split()
	if b := d.Block(2); b.Kind != Bullet || b.Level != 0 {
		t.Fatalf("empty item: %v level %d", b.Kind, b.Level)
	}
	d.Split()
	if d.Block(2).Kind != Paragraph {
		t.Fatal("empty top item should leave the list")
	}
	// Backspace at a list item's start takes the bullet away first.
	d.SetCaret(Pos{1, 0})
	d.DeleteBackward(false)
	if b := d.Block(1); b.Kind != Bullet || b.Level != 0 {
		t.Fatalf("backspace outdents first: level %d", b.Level)
	}
	d.DeleteBackward(false)
	if d.Block(1).Kind != Paragraph || d.Len() != 3 {
		t.Fatalf("then the bullet goes: %v", d.Block(1).Kind)
	}
	// Numbering restarts after a paragraph and nests.
	d2, _ := ParseHTML("<ol><li>a</li><li>b<ol><li>c</li><li>d</li></ol></li><li>e</li></ol><p>x</p><ol><li>f</li></ol>")
	got := d2.Numbers()
	want := []int{1, 2, 1, 2, 3, 0, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("numbers %v, want %v", got, want)
		}
	}
	if pt := d2.PlainText(); !strings.Contains(pt, "  b. d") || !strings.HasPrefix(pt, "1. a") {
		t.Errorf("plain text:\n%s", pt)
	}
	// Return after a heading makes a paragraph.
	d3 := New()
	d3.SetKind(Heading, 1)
	typeText(d3, "Title\nbody")
	if d3.Block(0).Kind != Heading || d3.Block(1).Kind != Paragraph {
		t.Fatal("heading then paragraph")
	}
}

func TestPasteFragment(t *testing.T) {
	d := New()
	typeText(d, "start end")
	frag, _ := ParseHTML("<p>one <b>two</b></p><h2>three</h2><p>four</p>")
	d.SetCaret(Pos{0, 6})
	d.InsertDoc(frag)
	if got := d.Text(); got != "start one two\nthree\nfourend" {
		t.Fatalf("paste: %q", got)
	}
	if d.Block(1).Kind != Heading || !d.Block(0).StyleAt(11).Bold {
		t.Fatal("pasted blocks and styles kept")
	}
	if c := d.Selection().Caret; c != (Pos{2, 4}) {
		t.Fatalf("caret after paste %v", c)
	}
	d.Undo()
	if d.Text() != "start end" {
		t.Fatalf("undo paste: %q", d.Text())
	}
	// Slice and paste back round trip.
	d.SetSelection(Pos{0, 0}, Pos{0, 5})
	f := d.SelectionDoc()
	if f.Text() != "start" {
		t.Fatal(f.Text())
	}
	d.SetCaret(d.End())
	d.InsertDoc(f)
	if d.Text() != "start endstart" {
		t.Fatal(d.Text())
	}
}

func TestHTMLRoundTrip(t *testing.T) {
	src := `<h1>Release notes</h1>
<p style="text-align:center">Plain <b>bold</b> <i>italic</i> <u>under</u> <s>gone</s> <code>mono</code></p>
<p><span style="font-size:20px;color:#ff0000;background-color:#ffff00">loud</span> and <a href="https://example.com/a?b=1&amp;c=2">a link</a>&nbsp;&nbsp;two spaces</p>
<ul><li>one<ul><li>nested</li></ul></li><li>two</li></ul>
<ol><li>first</li><li>second</li></ol>
<p></p>
<p>image <img src="DOT" alt="dot" width="16" height="16"> here</p>
`
	dot := paintengine2d.NewImage(2, 2)
	dot.Clear(paintengine2d.RGB(0.2, 0.4, 0.8))
	src = strings.Replace(src, "DOT", ImageDataURI(dot), 1)
	d, err := ParseHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	kinds := []Kind{Heading, Paragraph, Paragraph, Bullet, Bullet, Bullet, Numbered, Numbered, Paragraph, Paragraph}
	if d.Len() != len(kinds) {
		t.Fatalf("%d blocks: %q", d.Len(), d.Text())
	}
	for i, k := range kinds {
		if d.Block(i).Kind != k {
			t.Errorf("block %d is %v, want %v", i, d.Block(i).Kind, k)
		}
	}
	if d.Block(1).Align != AlignCenter || d.Block(4).Level != 1 {
		t.Error("align / nesting")
	}
	st := d.Block(2).StyleAt(0)
	if st.Size != 20 || st.Color.R != 1 || st.Highlight.G != 1 {
		t.Errorf("span style %+v", st)
	}
	if l := d.Block(2).StyleAt(10).Link; l != "https://example.com/a?b=1&c=2" {
		t.Errorf("link %q", l)
	}
	if !strings.Contains(d.Block(2).Text(), "link  two") {
		t.Errorf("non-breaking spaces: %q", d.Block(2).Text())
	}
	im := d.Block(9).ImageAt(6)
	if im == nil || im.Pixels == nil || im.Alt != "dot" || im.W != 16 {
		t.Fatalf("image %+v", im)
	}
	if d.PlainText() == "" || !strings.Contains(d.PlainText(), "image dot here") {
		t.Errorf("plain text keeps the alt text:\n%s", d.PlainText())
	}
	// Saved and read back, the document is the same.
	out := d.HTML()
	d2, _ := ParseHTML(out)
	if d2.Len() != d.Len() {
		t.Fatalf("round trip: %d blocks, want %d\n%s", d2.Len(), d.Len(), out)
	}
	for i := range d.Len() {
		if !d.Block(i).Equal(d2.Block(i)) {
			t.Errorf("block %d differs after a round trip:\n%+v\n%+v\n%s", i, d.Block(i), d2.Block(i), out)
		}
	}
	if out2 := d2.HTML(); out2 != out {
		t.Errorf("second save differs:\n%s\n---\n%s", out, out2)
	}
}

func TestHTMLTolerance(t *testing.T) {
	// Whatever a clipboard hands over: a head, a style block, comments,
	// unknown tags and entities.
	d, _ := ParseHTML(`<html><head><style>p{color:red}</style><title>x</title></head><body>
<!--StartFragment--><div>Hello&amp;<unknown>world</unknown> &lt;3</div><br>after<!--EndFragment--></body></html>`)
	if got := d.Text(); got != "Hello&world <3\n\nafter" && got != "Hello&world <3\nafter" {
		t.Fatalf("text %q", got)
	}
	// Broken markup never panics and keeps its text.
	for _, s := range []string{"<b>unclosed", "a < b", "<p", "<a href='x'>y", "</p></p>text", "<ul><li>a<li>b</ul>"} {
		d, err := ParseHTML(s)
		if err != nil || d.Len() == 0 {
			t.Errorf("%q: %v", s, err)
		}
	}
	d, _ = ParseHTML("<ul><li>a<li>b</ul>")
	if d.Len() != 2 || d.Block(1).Text() != "b" {
		t.Errorf("unclosed items: %q", d.Text())
	}
	if c, ok := ParseColor("rgb(255, 128, 0)"); !ok || c.R != 1 || c.B != 0 {
		t.Error("rgb()")
	}
	if c, ok := ParseColor("#0f0"); !ok || c.G != 1 {
		t.Error("#rgb")
	}
}

func TestWordAt(t *testing.T) {
	d := NewPlain("hello, world")
	a, c := d.WordAt(Pos{0, 2})
	if a.Off != 0 || c.Off != 5 {
		t.Errorf("word %v %v", a, c)
	}
	a, c = d.WordAt(Pos{0, 12})
	if a.Off != 7 || c.Off != 12 {
		t.Errorf("word at end %v %v", a, c)
	}
	if p := d.WordRight(Pos{0, 0}); p.Off != 5 {
		t.Errorf("word right %v", p)
	}
	if p := d.WordLeft(Pos{0, 12}); p.Off != 7 {
		t.Errorf("word left %v", p)
	}
}
