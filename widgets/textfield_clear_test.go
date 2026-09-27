package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func clearableField(h widget.Host, text string) *TextField {
	f := NewTextField(text, "Search themes", nil)
	f.Clearable = true
	f.SetHost(h)
	f.Arrange(paintengine2d.XYWH(0, 0, 240, 30))
	return f
}

// The cross is opt-in, it is there only while there is something to
// clear, and four kinds of field never get one whatever they ask for.
func TestFieldClearShowsOnlyWhereItShould(t *testing.T) {
	h := &host{}

	plain := NewTextField("something", "", nil)
	plain.SetHost(h)
	plain.Arrange(paintengine2d.XYWH(0, 0, 240, 30))
	if plain.clearShows() {
		t.Error("a field that did not ask for a clear button grew one")
	}

	empty := clearableField(h, "")
	if empty.clearShows() {
		t.Error("an empty field offers to clear itself")
	}

	full := clearableField(h, "breeze")
	if !full.clearShows() {
		t.Error("a clearable field with text has no cross")
	}

	secret := clearableField(h, "hunter2")
	secret.Password = true
	if secret.clearShows() {
		t.Error("a password field grew a clear button: one stray click and the secret is gone")
	}

	inner := clearableField(h, "12")
	inner.Frameless = true
	if inner.clearShows() {
		t.Error("a frameless field drew a cross in a frame its parent owns")
	}

	off := clearableField(h, "breeze")
	off.SetEnabled(false)
	if off.clearShows() {
		t.Error("a disabled field offers a live button")
	}

	// A box with no room for a cross and text beside it keeps its whole
	// width: one character is quicker to erase than to aim at.
	narrow := clearableField(h, "7")
	narrow.Arrange(paintengine2d.XYWH(0, 0, 40, 30))
	if narrow.clearShows() {
		t.Error("a 40-pixel field spent its width on a cross")
	}
}

// The promise that makes the cross tolerable: it never sits on the
// text, and it never sits on the caret at the end of it. The widget
// keeps that promise by handing the engines a string that stops before
// the cross, because each of the thirty-odd engines clips its text to a
// box of its own making and none of them knows the cross is there.
func TestFieldClearNeverSitsOnTheText(t *testing.T) {
	h := &host{}
	f := clearableField(h, strings.Repeat("Breeze Dark ", 12))

	room := f.clearRoom()
	if room <= 0 {
		t.Fatal("the cross reserves no room at all")
	}
	inner := f.LocalBounds().Dx() - f.fieldPad()*2 - room
	shown := f.fitClear(f.displayText(), inner)
	if shown == f.displayText() {
		t.Fatal("a string far longer than the field was handed to the engine whole")
	}
	if adv := f.font().Advance(shown) - f.scrollX; adv > inner+0.51 {
		t.Errorf("the string handed to the engine runs %v into a %v box", adv, inner)
	}
	// And what is drawn reaches the cross but does not cross it.
	if end := f.fieldPad() + f.font().Advance(shown) - f.scrollX; end > f.clearRect().Min.X {
		t.Errorf("the text ends at %v and the cross begins at %v", end, f.clearRect().Min.X)
	}

	// The caret at the end of a long string stops in front of the cross
	// too: ensureCaretVisible measures against the narrowed box.
	f.SetSelection(runeCount(f.Text), runeCount(f.Text))
	cx := f.fieldPad() + f.font().CaretX(f.displayText(), f.caret) - f.scrollX
	if cx > f.clearRect().Min.X {
		t.Errorf("the caret is at %v, under a cross that begins at %v", cx, f.clearRect().Min.X)
	}
}

// It scales: the cross is bigger at 1.75x, still inside the field, and
// still clear of the text.
func TestFieldClearScales(t *testing.T) {
	one := clearableField(&host{}, "breeze")
	big := clearableField(&scaledHost{look: style.WithScale(style.DarkLook(), 1.75)}, "breeze")
	big.Arrange(paintengine2d.XYWH(0, 0, 240*1.75, 30*1.75))
	if big.clearSide() <= one.clearSide() {
		t.Errorf("the cross is %v at 1.75x and %v at 1x", big.clearSide(), one.clearSide())
	}
	for _, f := range []*TextField{one, big} {
		r := f.clearRect()
		if !f.LocalBounds().Contains(r.Min) || r.Max.X > f.LocalBounds().Max.X {
			t.Errorf("the cross %v is not inside the field %v", r, f.LocalBounds())
		}
	}
}

// Clicking it clears the field and takes neither the focus nor the
// caret, and it acts on the release where the press landed — a press
// that turned out to be a mistake can be taken back.
func TestFieldClearClickDoesNotStealTheCaret(t *testing.T) {
	h := &host{}
	f := clearableField(h, "breeze")
	var changed, input []string
	f.OnChange = func(s string) { changed = append(changed, s) }
	f.OnInput = func(s string) { input = append(input, s) }
	at := f.clearRect().Center()

	if !f.MousePress(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft}) {
		t.Fatal("the cross ignored a press")
	}
	if h.Focus() == widget.Component(f) {
		t.Error("pressing the cross took the focus into the field")
	}
	if f.Text != "breeze" {
		t.Error("the field cleared on the press instead of the release")
	}
	f.MouseRelease(widget.MouseEvent{Pos: at, Button: platform.ButtonLeft})
	if f.Text != "" {
		t.Errorf("the field still reads %q after a click on the cross", f.Text)
	}
	if h.Focus() == widget.Component(f) {
		t.Error("clearing the field took the focus into it")
	}
	if f.caret != 0 || f.selA != 0 || f.selB != 0 {
		t.Errorf("the caret is at %d and the selection is [%d,%d] after clearing", f.caret, f.selA, f.selB)
	}
	// A click on the cross is the user's own edit, so both callbacks
	// fire and in the documented order (widgets/oninput.go).
	if len(changed) != 1 || len(input) != 1 || changed[0] != "" || input[0] != "" {
		t.Errorf("OnChange %v, OnInput %v; want one empty string each", changed, input)
	}

	// Released somewhere else: nothing happens.
	g := clearableField(h, "breeze")
	g.MousePress(widget.MouseEvent{Pos: g.clearRect().Center(), Button: platform.ButtonLeft})
	g.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(10, 15), Button: platform.ButtonLeft})
	if g.Text != "breeze" {
		t.Error("a press on the cross that was released off it still cleared the field")
	}

	// And a press in the text is still a press in the text.
	k := clearableField(h, "breeze")
	k.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 15), Button: platform.ButtonLeft})
	if h.Focus() != widget.Component(k) {
		t.Error("a press in the text did not focus the field")
	}
}

// Escape is the keyboard's way to the cross, and it is the convention
// of every field that has one: the first Escape empties the field, and
// only once it is empty does Escape mean what the page says.
func TestFieldClearEscapeClearsFirst(t *testing.T) {
	h := &host{}
	f := clearableField(h, "breeze")
	escaped := 0
	f.OnEscape = func() { escaped++ }

	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyEscape}) {
		t.Fatal("Escape did nothing in a field with something to clear")
	}
	if f.Text != "" {
		t.Errorf("Escape left %q in the field", f.Text)
	}
	if escaped != 0 {
		t.Error("Escape ran the page's handler as well as clearing the field")
	}
	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyEscape}) || escaped != 1 {
		t.Errorf("Escape in the empty field ran the page's handler %d times", escaped)
	}

	// A field that did not ask for a clear button is untouched by any
	// of this.
	p := NewTextField("breeze", "", nil)
	p.SetHost(h)
	p.Arrange(paintengine2d.XYWH(0, 0, 240, 30))
	n := 0
	p.OnEscape = func() { n++ }
	p.KeyPress(widget.KeyEvent{Key: platform.KeyEscape})
	if p.Text != "breeze" || n != 1 {
		t.Errorf("Escape in a plain field left %q and ran the handler %d times", p.Text, n)
	}
}

// A screen reader has to be able to clear the field too, so the cross
// is in the tree as a named button under the field and answers the
// default action.
func TestFieldClearIsNamedAndReachable(t *testing.T) {
	h := &host{}
	f := clearableField(h, "breeze")

	items := f.AccessibleItems()
	if len(items) != 1 {
		t.Fatalf("the tree has %d items under the field, want the clear button", len(items))
	}
	n := items[0]
	if n.Role != a11y.RoleButton {
		t.Errorf("the clear button is a %v in the tree", n.Role)
	}
	// Named for the field it clears: "Clear" alone, on a page with four
	// fields, names four different buttons the same.
	if n.Name != "Clear Search themes" {
		t.Errorf("the clear button is called %q", n.Name)
	}
	if n.Bounds.Empty() {
		t.Error("the clear button has no box in the tree")
	}
	if !n.Actions.Has(a11y.ActionDefault) {
		t.Error("the clear button offers no default action")
	}
	if !f.AccessibleAction(0, a11y.ActionDefault) || f.Text != "" {
		t.Errorf("the default action left %q in the field", f.Text)
	}
	// And it leaves the tree with the text.
	if len(f.AccessibleItems()) != 0 {
		t.Error("an empty field still advertises a clear button")
	}

	// An accessible name beats the placeholder.
	g := clearableField(h, "breeze")
	g.SetAccessibleName("Filter packs")
	if got := g.AccessibleItems()[0].Name; got != "Clear Filter packs" {
		t.Errorf("the clear button is called %q, want the field's own name", got)
	}
}
