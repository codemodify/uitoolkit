package widgets

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/richtext"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// lookHost is the fake window with one look, made once, as a real window
// keeps it (host builds a fresh look per call, which would be most of what
// a timing measured).
type lookHost struct {
	host
	lk style.LookAndFeel
}

func (h *lookHost) Look() style.LookAndFeel { return h.lk }

// richHarness is an editor laid out in a fake window, with a canvas to
// paint it on.
func richHarness(t testing.TB, html string, w, h float32) (*RichText, *lookHost, *paintengine2d.Context) {
	t.Helper()
	ed := NewRichText("Write here")
	hs := &lookHost{lk: style.DarkLook()}
	ed.SetHost(hs)
	if html != "" {
		ed.SetHTML(html)
	}
	ed.Arrange(paintengine2d.XYWH(0, 0, w, h))
	hs.RequestFocus(ed)
	ctx := paintengine2d.NewContext(paintengine2d.NewImage(int(w), int(h)))
	ed.Paint(ctx)
	return ed, hs, ctx
}

func key(k platform.Key, r rune, mods platform.Modifiers) widget.KeyEvent {
	return widget.KeyEvent{Key: k, Rune: r, Mods: mods}
}

func typeInto(ed *RichText, s string) {
	for _, r := range s {
		if r == '\n' {
			ed.KeyPress(key(platform.KeyReturn, 0, 0))
			continue
		}
		ed.TextInput(r)
	}
}

// pointOf is a local point just inside character off, which a click
// there puts the caret before.
func pointOf(ed *RichText, off int) paintengine2d.Point {
	r := ed.PositionRect(off)
	return paintengine2d.Pt(r.Min.X+1, (r.Min.Y+r.Max.Y)*0.5)
}

func TestRichTextKeyboard(t *testing.T) {
	ed, _, _ := richHarness(t, "", 400, 240)
	changes := 0
	ed.OnChange = func() { changes++ }
	typeInto(ed, "Hello world\nsecond line")
	d := ed.Document()
	if d.Text() != "Hello world\nsecond line" || changes == 0 {
		t.Fatalf("typed %q (%d changes)", d.Text(), changes)
	}
	// Home, Shift+End selects the line; Ctrl+B bolds it.
	ed.KeyPress(key(platform.KeyHome, 0, 0))
	ed.KeyPress(key(platform.KeyEnd, 0, platform.ModShift))
	if ed.SelectedText() != "second line" {
		t.Fatalf("selected %q", ed.SelectedText())
	}
	ed.KeyPress(key(platform.KeyB, 'b', platform.ModCtrl))
	if !d.Block(1).StyleAt(3).Bold {
		t.Fatal("Ctrl+B")
	}
	// Up goes to the line above at the same column; Ctrl+Left by words.
	ed.KeyPress(key(platform.KeyEnd, 0, 0))
	ed.KeyPress(key(platform.KeyUp, 0, 0))
	if c := d.Selection().Caret; c.Block != 0 {
		t.Fatalf("up: %v", c)
	}
	ed.KeyPress(key(platform.KeyEnd, 0, 0))
	ed.KeyPress(key(platform.KeyLeft, 0, platform.ModCtrl))
	if c := d.Selection().Caret; c != (richtext.Pos{Block: 0, Off: 6}) {
		t.Fatalf("ctrl+left: %v", c)
	}
	// Right at a block's end crosses into the next block.
	ed.KeyPress(key(platform.KeyEnd, 0, 0))
	ed.KeyPress(key(platform.KeyRight, 0, 0))
	if c := d.Selection().Caret; c != (richtext.Pos{Block: 1}) {
		t.Fatalf("right across blocks: %v", c)
	}
	// Ctrl+Z takes the bold back, then Ctrl+Shift+Z does it again.
	ed.KeyPress(key(platform.KeyZ, 'z', platform.ModCtrl))
	if d.Block(1).StyleAt(3).Bold {
		t.Fatal("undo")
	}
	ed.KeyPress(key(platform.KeyZ, 'z', platform.ModCtrl|platform.ModShift))
	if !d.Block(1).StyleAt(3).Bold {
		t.Fatal("redo")
	}
	// Block formats from the keyboard.
	ed.KeyPress(key(platform.KeyUnknown, '2', platform.ModCtrl|platform.ModAlt))
	if k, lv := d.BlockKind(); k != richtext.Heading || lv != 2 {
		t.Fatalf("Ctrl+Alt+2: %v %d", k, lv)
	}
	ed.KeyPress(key(platform.KeyUnknown, '8', platform.ModCtrl|platform.ModShift))
	if k, _ := d.BlockKind(); k != richtext.Bullet {
		t.Fatalf("Ctrl+Shift+8: %v", k)
	}
	ed.KeyPress(key(platform.KeyUnknown, ']', platform.ModCtrl))
	if d.Block(1).Level != 1 {
		t.Fatal("Ctrl+]")
	}
	ed.KeyPress(key(platform.KeyUnknown, 'e', platform.ModCtrl|platform.ModShift))
	if d.Block(1).Align != richtext.AlignCenter {
		t.Fatal("Ctrl+Shift+E")
	}
	// Ctrl+A, then typing replaces everything as one step.
	ed.KeyPress(key(platform.KeyA, 'a', platform.ModCtrl))
	ed.TextInput('x')
	if d.Text() != "x" {
		t.Fatalf("replace all: %q", d.Text())
	}
	ed.KeyPress(key(platform.KeyZ, 'z', platform.ModCtrl))
	if d.Len() != 2 {
		t.Fatalf("undo replace all: %q", d.Text())
	}
	// Read-only: keys move and copy, but never edit.
	ed.ReadOnly = true
	before := d.Text()
	ed.TextInput('q')
	ed.KeyPress(key(platform.KeyBackspace, 0, 0))
	ed.KeyPress(key(platform.KeyB, 'b', platform.ModCtrl))
	if d.Text() != before {
		t.Fatal("read-only editor was edited")
	}
}

func TestRichTextMouse(t *testing.T) {
	ed, _, _ := richHarness(t, "<p>alpha beta gamma</p><p>second paragraph here</p>", 400, 240)
	d := ed.Document()
	press := func(p paintengine2d.Point, mods platform.Modifiers) {
		ed.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft, Mods: mods})
		ed.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	}
	// A click puts the caret there.
	press(pointOf(ed, 7), 0)
	if c := d.Selection().Caret; c.Block != 0 || c.Off < 6 || c.Off > 8 {
		t.Fatalf("click: %v", c)
	}
	// A double click selects the word, a third the paragraph.
	p := pointOf(ed, 8)
	ed.clickAt = time.Time{}
	press(p, 0)
	press(p, 0)
	if ed.SelectedText() != "beta" {
		t.Fatalf("double click: %q", ed.SelectedText())
	}
	press(p, 0)
	if ed.SelectedText() != "alpha beta gamma" {
		t.Fatalf("triple click: %q", ed.SelectedText())
	}
	// A slow second click is a new single click.
	ed.clickAt = time.Now().Add(-time.Second)
	press(pointOf(ed, 2), 0)
	if d.HasSelection() {
		t.Fatal("a late click selected")
	}
	// Dragging selects, into the next block.
	ed.clickAt = time.Time{}
	from, to := pointOf(ed, 6), pointOf(ed, d.Offset(richtext.Pos{Block: 1, Off: 6}))
	ed.MousePress(widget.MouseEvent{Pos: from, Button: platform.ButtonLeft})
	ed.MouseMove(widget.MouseEvent{Pos: to, Button: platform.ButtonLeft})
	ed.MouseRelease(widget.MouseEvent{Pos: to, Button: platform.ButtonLeft})
	if got := ed.SelectedText(); got != "beta gamma\nsecond" {
		t.Fatalf("drag selected %q", got)
	}
	// Shift+click extends.
	ed.clickAt = time.Time{}
	press(pointOf(ed, 2), 0)
	ed.clickAt = time.Time{}
	press(pointOf(ed, 10), platform.ModShift)
	if got := ed.SelectedText(); !strings.HasPrefix(got, "pha beta") {
		t.Fatalf("shift+click %q", got)
	}
	// Ctrl+click follows a link; a plain click does not.
	ed2, _, _ := richHarness(t, `<p>go <a href="https://example.com">there</a> now</p>`, 400, 120)
	var followed string
	ed2.OnLink = func(h string) { followed = h }
	lp := pointOf(ed2, 5)
	ed2.MousePress(widget.MouseEvent{Pos: lp, Button: platform.ButtonLeft})
	ed2.MouseRelease(widget.MouseEvent{Pos: lp, Button: platform.ButtonLeft})
	if followed != "" {
		t.Fatal("a plain click followed the link")
	}
	ed2.clickAt = time.Time{}
	ed2.MousePress(widget.MouseEvent{Pos: lp, Button: platform.ButtonLeft, Mods: platform.ModCtrl})
	if followed != "https://example.com" {
		t.Fatalf("ctrl+click followed %q", followed)
	}
	ed2.MouseMove(widget.MouseEvent{Pos: lp})
	if ed2.Tooltip() != "https://example.com" {
		t.Errorf("tooltip %q", ed2.Tooltip())
	}
}

func TestRichTextClipboardFlavours(t *testing.T) {
	platform.ClipboardSet("")
	src, _, _ := richHarness(t, "<p>plain <b>bold</b> <i>it</i></p><ul><li>item</li></ul>", 400, 200)
	src.Document().SelectAll()
	src.KeyPress(key(platform.KeyC, 'c', platform.ModCtrl))
	if got := platform.ClipboardGet(); got != "plain bold it\n• item" {
		t.Fatalf("text flavour %q", got)
	}
	html, ok := ClipboardHTML()
	if !ok || !strings.Contains(html, "<b>bold</b>") || !strings.Contains(html, "<ul><li>item</li></ul>") {
		t.Fatalf("html flavour %q %v", html, ok)
	}
	// Pasted into another editor, the formatting comes along.
	dst, _, _ := richHarness(t, "", 400, 200)
	dst.KeyPress(key(platform.KeyV, 'v', platform.ModCtrl))
	d := dst.Document()
	if d.Text() != "plain bold it\nitem" || !d.Block(0).StyleAt(7).Bold || d.Block(1).Kind != richtext.Bullet {
		t.Fatalf("rich paste: %q", d.Text())
	}
	// Ctrl+Shift+V pastes the text alone.
	d.SelectAll()
	dst.KeyPress(key(platform.KeyV, 'v', platform.ModCtrl|platform.ModShift))
	if d.Block(0).StyleAt(7).Bold {
		t.Fatal("plain paste kept bold")
	}
	// Something else copied since: the stale HTML is not used.
	platform.ClipboardSet("from elsewhere")
	if _, ok := ClipboardHTML(); ok {
		t.Fatal("stale HTML survived another copy")
	}
	d.SelectAll()
	dst.KeyPress(key(platform.KeyV, 'v', platform.ModCtrl))
	if d.Text() != "from elsewhere" {
		t.Fatalf("paste %q", d.Text())
	}
	// Cut takes the selection away and leaves it on the clipboard.
	d.SetSelection(richtext.Pos{Off: 0}, richtext.Pos{Off: 5})
	dst.KeyPress(key(platform.KeyX, 'x', platform.ModCtrl))
	if d.Text() != "elsewhere" || platform.ClipboardGet() != "from " {
		t.Fatalf("cut: %q / %q", d.Text(), platform.ClipboardGet())
	}
}

func TestRichTextDragAndDrop(t *testing.T) {
	ed, _, _ := richHarness(t, "<p>one <b>two</b> three</p>", 400, 160)
	d := ed.Document()
	d.SetSelection(richtext.Pos{Off: 4}, richtext.Pos{Off: 7}) // "two"
	// A press inside the selection waits to become a drag.
	at := pointOf(ed, 5)
	ed.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if !d.HasSelection() {
		t.Fatal("a press inside the selection dropped it")
	}
	drag := ed.DragAt(at)
	if drag == nil || !drag.Offers("text/html") || !drag.Offers("text/plain") {
		t.Fatalf("drag types %v", drag)
	}
	if b, _ := drag.Data("text/html"); !strings.Contains(string(b), "<b>two</b>") {
		t.Fatalf("html %q", b)
	}
	if b, _ := drag.Data("text/plain"); string(b) != "two" {
		t.Fatalf("text %q", b)
	}
	// Dropped at the end of the same editor: moved, as one undo step.
	end := ed.PositionRect(d.TextLen())
	ok := ed.Drop(widget.DropEvent{Pos: paintengine2d.Pt(end.Max.X+4, end.Min.Y+2), Action: platform.DragMove, Payload: drag.Payload, Source: ed})
	drag.Done(platform.DragMove)
	if !ok || d.Text() != "one  threetwo" {
		t.Fatalf("self move: %q", d.Text())
	}
	if !d.Block(0).StyleAt(11).Bold || ed.SelectedText() != "two" {
		t.Fatalf("moved text keeps its style and is selected: %q", ed.SelectedText())
	}
	ed.KeyPress(key(platform.KeyZ, 'z', platform.ModCtrl))
	if d.Text() != "one two three" {
		t.Fatalf("undo move: %q", d.Text())
	}
	// A move to another widget takes the text out once the target says so.
	d.SetSelection(richtext.Pos{Off: 0}, richtext.Pos{Off: 3})
	ed.MousePress(widget.MouseEvent{Pos: pointOf(ed, 1), Button: platform.ButtonLeft})
	drag = ed.DragAt(pointOf(ed, 1))
	other, _, _ := richHarness(t, "", 300, 100)
	other.Drop(widget.DropEvent{Pos: paintengine2d.Pt(20, 20), Mime: "text/html", Data: []byte("<b>one</b>"), Action: platform.DragMove})
	drag.Done(platform.DragMove)
	if d.Text() != " two three" || other.Document().Text() != "one" || !other.Document().Block(0).StyleAt(0).Bold {
		t.Fatalf("move out: %q / %q", d.Text(), other.Document().Text())
	}
	// Text from another application drops as text.
	other.Drop(widget.DropEvent{Pos: paintengine2d.Pt(1, 1), Text: "hi ", Action: platform.DragCopy})
	if !strings.HasPrefix(other.Document().Text(), "hi ") {
		t.Fatalf("text drop %q", other.Document().Text())
	}
	// Firefox sends its HTML as UTF-16.
	u := []byte{0xff, 0xfe}
	for _, r := range "<i>x</i>" {
		u = append(u, byte(r), 0)
	}
	if decodeHTMLDrop(u) != "<i>x</i>" {
		t.Fatal("utf-16 html")
	}
}

func TestRichTextAccessibility(t *testing.T) {
	ed, hs, _ := richHarness(t, `<p>go <a href="https://example.com">there</a></p><p>img <img src="x.png" alt="pic"></p>`, 400, 160)
	ed.SetAccessibleName("Message body")
	d := ed.Document()
	d.SetSelection(richtext.Pos{Off: 1}, richtext.Pos{Block: 1, Off: 2})
	root := &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "w", Bounds: paintengine2d.XYWH(0, 0, 400, 160)}
	col := NewColumn(ed)
	col.SetHost(hs)
	widget.AccessibleTree(root, col)
	var n *a11y.Node
	root.Walk(func(x *a11y.Node) bool {
		if x.Role == a11y.RoleTextArea {
			n = x
		}
		return true
	})
	if n == nil || n.Name != "Message body" || !n.State.Has(a11y.StateMultiLine|a11y.StateEditable) {
		t.Fatalf("node %+v", n)
	}
	if n.Value != "go there\nimg ￼" || n.SelStart != 1 || n.SelEnd != 11 || n.Caret != 11 {
		t.Fatalf("value %q sel %d-%d caret %d", n.Value, n.SelStart, n.SelEnd, n.Caret)
	}
	if len(n.Children) != 1 || n.Children[0].Role != a11y.RoleLink || n.Children[0].Name != "there" || n.Children[0].Bounds.Empty() {
		t.Fatalf("link items %+v", n.Children)
	}
	var followed string
	ed.OnLink = func(h string) { followed = h }
	if !ed.AccessibleAction(0, a11y.ActionDefault) || followed != "https://example.com" {
		t.Fatal("link action")
	}
	for _, p := range a11y.Check(root) {
		t.Error(p)
	}
	if !ed.AccessibleSetText("new text") || d.Text() != "new text" {
		t.Fatal("set text")
	}
}

func TestRichTextBarFollowsCaret(t *testing.T) {
	ed, hs, _ := richHarness(t, "<h2>Title</h2><p>plain <b>bold</b></p><ol><li>one</li></ol>", 500, 200)
	bar := NewRichTextBar(ed)
	bar.SetHost(hs)
	d := ed.Document()
	d.SetCaret(richtext.Pos{Block: 1, Off: 8})
	ed.moved()
	chars := bar.groups[0]
	if !chars.tools[0].isDown() || chars.tools[1].isDown() {
		t.Fatal("bold button should be down in bold text")
	}
	if bar.block.Selected != 0 {
		t.Fatalf("block list %d", bar.block.Selected)
	}
	d.SetCaret(richtext.Pos{Block: 0, Off: 2})
	ed.moved()
	if bar.block.Text() != "Heading 2" {
		t.Fatalf("block list %q", bar.block.Text())
	}
	// Choosing in the bar formats the document.
	d.SetSelection(richtext.Pos{Block: 1}, richtext.Pos{Block: 1, Off: 5})
	ed.moved()
	chars.activate(1) // italic
	if !d.Block(1).StyleAt(2).Italic {
		t.Fatal("italic from the bar")
	}
	bar.block.Select(1)
	if k, lv := d.BlockKind(); k != richtext.Heading || lv != 1 {
		t.Fatalf("heading from the bar: %v %d", k, lv)
	}
	bar.size.Select(9) // 24
	if d.Block(1).StyleAt(1).Size != 24 {
		t.Fatalf("size from the bar: %v", d.Block(1).StyleAt(1).Size)
	}
	lists := bar.groups[1]
	d.SetCaret(richtext.Pos{Block: 2})
	ed.moved()
	if !lists.tools[1].isDown() || !lists.tools[3].isEnabled() {
		t.Fatal("numbered list state")
	}
	lists.activate(3)
	if d.Block(2).Level != 1 {
		t.Fatal("indent from the bar")
	}
	// Every button is named for assistive technology.
	root := &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "w", Bounds: paintengine2d.XYWH(0, 0, 500, 200)}
	bar.Arrange(paintengine2d.XYWH(0, 0, 500, 80))
	widget.AccessibleTree(root, bar)
	for _, p := range a11y.Check(root) {
		t.Error(p)
	}
	// The bar's parts are all in the Tab order.
	focus := widget.Focusables(bar)
	if len(focus) < 2+len(bar.groups)+2 {
		t.Fatalf("%d focusable parts in the bar", len(focus))
	}
}

// longDoc is n paragraphs of mixed text, a heading every 50.
func longDoc(n int) string {
	var b strings.Builder
	for i := range n {
		if i%50 == 0 {
			fmt.Fprintf(&b, "<h2>Section %d</h2>", i/50+1)
		}
		fmt.Fprintf(&b, "<p>Paragraph %d has <b>some bold</b>, <i>some italic</i> and enough plain words to wrap once or twice at the width of the editor it is shown in.</p>", i)
	}
	return b.String()
}

func TestRichTextLongDocument(t *testing.T) {
	const paragraphs = 10000
	html := longDoc(paragraphs)
	start := time.Now()
	ed, _, ctx := richHarness(t, html, 640, 480)
	load := time.Since(start)
	d := ed.Document()
	if d.Len() < paragraphs {
		t.Fatalf("%d blocks", d.Len())
	}
	laid := ed.laidOut
	if laid > 100 {
		t.Errorf("opening laid out %d blocks; only what is on screen should be", laid)
	}
	// Type in the middle of the document, painting after every key, as
	// the window does.
	mid := richtext.Pos{Block: d.Len() / 2, Off: 10}
	d.SetCaret(mid)
	ed.moved()
	ed.Paint(ctx)
	before := ed.laidOut
	const keys = 200
	var worst time.Duration
	start = time.Now()
	for i := range keys {
		k := time.Now()
		ed.TextInput(rune('a' + i%26))
		ed.Paint(ctx)
		worst = max(worst, time.Since(k))
	}
	typing := time.Since(start)
	per := typing / keys
	relaid := ed.laidOut - before
	t.Logf("%d blocks: load+first paint %v, typing %v a key (worst %v), %d blocks re-laid for %d keys",
		d.Len(), load.Round(time.Millisecond), per.Round(time.Microsecond), worst.Round(time.Microsecond), relaid, keys)
	if relaid > keys+50 {
		t.Errorf("typing re-laid %d blocks for %d keys: layout is not incremental", relaid, keys)
	}
	// No wall-clock assertion here: the guarantee this test exists for is the
	// one above — typing re-lays the blocks it touched and no more. A clock
	// threshold measures the machine and the rest of the suite running beside
	// it, and failed for that reason. Speed is BenchmarkRichTextTyping's job.
	if per > 8*time.Millisecond {
		t.Logf("typing costs %v a key, more than the 8ms this used to insist on", per)
	}
	// The same typing without the pixels: the model, the history and the
	// layout of what is on screen.
	start = time.Now()
	for i := range keys {
		ed.TextInput(rune('a' + i%26))
		lo, hi := ed.visibleBlocks()
		for b := lo; b < hi; b++ {
			ed.lay(b)
		}
	}
	t.Logf("edit and layout alone: %v a key", (time.Since(start) / keys).Round(time.Microsecond))
	// Jumping to the end and back is cheap too, and the caret shows.
	start = time.Now()
	ed.KeyPress(key(platform.KeyEnd, 0, platform.ModCtrl))
	ed.Paint(ctx)
	ed.KeyPress(key(platform.KeyHome, 0, platform.ModCtrl))
	ed.Paint(ctx)
	t.Logf("Ctrl+End and Ctrl+Home with a paint each: %v", time.Since(start).Round(time.Microsecond))
	if ed.ScrollOffset() != 0 {
		t.Errorf("Ctrl+Home left the view at %v", ed.ScrollOffset())
	}
	// Undo takes the typing back in word-sized steps, not a key at a time.
	steps := 0
	for d.CanUndo() && steps < keys {
		d.Undo()
		steps++
	}
	if steps != 1 {
		t.Logf("undo steps for %d typed letters: %d", keys, steps)
	}
}

func BenchmarkRichTextTyping(b *testing.B) {
	ed, _, ctx := richHarness(b, longDoc(10000), 640, 480)
	d := ed.Document()
	d.SetCaret(richtext.Pos{Block: d.Len() / 2, Off: 10})
	ed.moved()
	b.ResetTimer()
	for i := range b.N {
		ed.TextInput(rune('a' + i%26))
		ed.Paint(ctx)
	}
}

// With no OnLink, an editor opens a link with the desktop's application for
// it, as every other link the toolkit shows does (OpenLink).
func TestRichTextOpensLinksByDefault(t *testing.T) {
	got := make(chan string, 1)
	old := openURI
	openURI = func(uri string, _ platform.OpenURIOptions) error {
		got <- uri
		return nil
	}
	defer func() { openURI = old }()
	ed, _, _ := richHarness(t, `<p>go <a href="https://example.com">there</a></p>`, 400, 120)
	if ed.OnLink != nil {
		t.Fatal("OnLink should start nil")
	}
	if !ed.AccessibleAction(0, a11y.ActionDefault) {
		t.Fatal("the link's action was refused")
	}
	select {
	case u := <-got:
		if u != "https://example.com" {
			t.Fatalf("opened %q", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was opened")
	}
}
