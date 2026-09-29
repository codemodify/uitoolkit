package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func testTokenField(t *testing.T, w float32) *TokenField {
	t.Helper()
	f := NewTokenField("To", nil)
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})
	f.Measure(layout.Loose(w, 400))
	f.Arrange(paintengine2d.XYWH(0, 0, w, 120))
	return f
}

func typeToken(f *TextField, s string) {
	for _, r := range s {
		f.TextInput(r)
	}
}

// A separator ends a token. Typing three addresses with commas between
// them leaves three chips and an empty editor.
func TestTokenFieldCommitsOnSeparator(t *testing.T) {
	f := testTokenField(t, 400)
	var last []string
	f.OnChange = func(v []string) { last = append([]string(nil), v...) }
	typeToken(f.Editor(), "ada@example.com, grace@example.com;alan@example.com,")
	got := strings.Join(f.Tokens(), "|")
	want := "ada@example.com|grace@example.com|alan@example.com"
	if got != want {
		t.Fatalf("tokens %q, want %q", got, want)
	}
	if f.Pending() != "" {
		t.Fatalf("pending %q", f.Pending())
	}
	if strings.Join(last, "|") != want {
		t.Fatalf("OnChange last saw %q", last)
	}
}

// What comes after the last separator stays in the editor, being typed.
func TestTokenFieldKeepsTheTail(t *testing.T) {
	f := testTokenField(t, 400)
	typeToken(f.Editor(), "ada@example.com, gra")
	if len(f.Tokens()) != 1 || f.Pending() != "gra" {
		t.Fatalf("tokens %v pending %q", f.Tokens(), f.Pending())
	}
}

// Return commits what is pending; with nothing pending it is the field's
// own submit, which is how a form moves on.
func TestTokenFieldReturn(t *testing.T) {
	f := testTokenField(t, 400)
	submits := 0
	f.OnSubmit = func() { submits++ }
	typeToken(f.Editor(), "ada@example.com")
	f.Editor().KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if len(f.Tokens()) != 1 || f.Pending() != "" || submits != 0 {
		t.Fatalf("tokens %v pending %q submits %d", f.Tokens(), f.Pending(), submits)
	}
	f.Editor().KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if submits != 1 {
		t.Fatalf("submits %d", submits)
	}
}

// Backspace with an empty editor takes the last chip *back into* the
// editor. A mistyped address is then one keystroke from being fixed,
// where deleting it outright would mean typing it again.
func TestTokenFieldBackspaceTakesTheLastOneBack(t *testing.T) {
	f := testTokenField(t, 400)
	typeToken(f.Editor(), "ada@example.com,grace@exmaple.com,")
	if len(f.Tokens()) != 2 {
		t.Fatalf("tokens %v", f.Tokens())
	}
	// The editor lets the key go because it has nothing to delete.
	if f.Editor().KeyPress(widget.KeyEvent{Key: platform.KeyBackspace}) {
		t.Fatal("the empty editor swallowed Backspace")
	}
	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace}) {
		t.Fatal("the field ignored Backspace")
	}
	if len(f.Tokens()) != 1 || f.Pending() != "grace@exmaple.com" {
		t.Fatalf("tokens %v pending %q", f.Tokens(), f.Pending())
	}
	// With text in the editor it is an ordinary Backspace again.
	if f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace}) {
		t.Fatal("the field took a chip back while the editor had text")
	}
}

// Leaving the field commits what was typed. A typed address that
// disappeared because nobody pressed comma is the bug this stops.
func TestTokenFieldCommitsOnLeaving(t *testing.T) {
	f := testTokenField(t, 400)
	typeToken(f.Editor(), "ada@example.com")
	if len(f.Tokens()) != 0 {
		t.Fatalf("committed early: %v", f.Tokens())
	}
	f.Editor().FocusLost()
	if len(f.Tokens()) != 1 || f.Tokens()[0] != "ada@example.com" || f.Pending() != "" {
		t.Fatalf("tokens %v pending %q", f.Tokens(), f.Pending())
	}
}

// Accept vets a token and leaves a rejected one in the editor, where the
// user can see what was wrong with it. Unique drops a repeat.
func TestTokenFieldAcceptAndUnique(t *testing.T) {
	f := testTokenField(t, 400)
	f.Accept = func(s string) bool { return strings.Contains(s, "@") }
	f.Unique = true
	typeToken(f.Editor(), "not-an-address,")
	if len(f.Tokens()) != 0 {
		t.Fatalf("Accept let %v through", f.Tokens())
	}
	// The refused text stays, without the separator that tried to commit
	// it — the user can see what was wrong, and the next keystroke is
	// not a second failing commit.
	if f.Pending() != "not-an-address" {
		t.Fatalf("pending after a refused token: %q", f.Pending())
	}
	// And fixing it works.
	typeToken(f.Editor(), "@example.com,")
	if got := strings.Join(f.Tokens(), "|"); got != "not-an-address@example.com" {
		t.Fatalf("after the fix: %q", got)
	}
	f.SetTokens(nil)
	typeToken(f.Editor(), "ada@example.com,ada@example.com,")
	if len(f.Tokens()) != 1 {
		t.Fatalf("Unique let a repeat in: %v", f.Tokens())
	}
}

// Clicking a chip's cross removes it; clicking its body does not.
func TestTokenFieldChipCrossRemoves(t *testing.T) {
	f := testTokenField(t, 400)
	f.SetTokens([]string{"ada@example.com", "grace@example.com"})
	f.Measure(layout.Loose(400, 400))
	f.Arrange(paintengine2d.XYWH(0, 0, 400, 120))
	chip := f.Chips()[0]
	body := paintengine2d.Pt(chip.LocalBounds().Min.X+4, chip.LocalBounds().Dy()*0.5)
	chip.MousePress(widget.MouseEvent{Pos: body})
	chip.MouseRelease(widget.MouseEvent{Pos: body})
	if len(f.Tokens()) != 2 {
		t.Fatalf("a click on the body removed a chip: %v", f.Tokens())
	}
	cross := chip.crossRect().Center()
	chip.MousePress(widget.MouseEvent{Pos: cross})
	chip.MouseRelease(widget.MouseEvent{Pos: cross})
	if len(f.Tokens()) != 1 || f.Tokens()[0] != "grace@example.com" {
		t.Fatalf("tokens %v", f.Tokens())
	}
}

// The chips fold onto a second line when they do not fit, and the editor
// takes the rest of the last line — or a line of its own when what is
// left is too narrow to type in.
func TestTokenFieldFolds(t *testing.T) {
	f := testTokenField(t, 220)
	f.SetTokens([]string{"ada@example.com", "grace@example.com", "alan@example.com"})
	tall := f.Measure(layout.Loose(220, 1000))
	f.Arrange(paintengine2d.XYWH(0, 0, 220, tall.Y))

	lines := map[float32]int{}
	for _, c := range f.Chips() {
		lines[c.Bounds().Min.Y]++
	}
	if len(lines) < 2 {
		t.Fatalf("three chips in %v fit on %d line(s)", 220.0, len(lines))
	}
	// One line's worth would be shorter than this.
	one := testTokenField(t, 220)
	one.SetTokens([]string{"ada@example.com"})
	if short := one.Measure(layout.Loose(220, 1000)); tall.Y <= short.Y {
		t.Fatalf("folded height %v is not more than one line's %v", tall.Y, short.Y)
	}
	// Nothing runs past the right edge.
	for i, c := range f.Chips() {
		if c.Bounds().Max.X > 220 {
			t.Fatalf("chip %d ends at %v, past the field", i, c.Bounds().Max.X)
		}
	}
	if e := f.Editor().Bounds(); e.Max.X > 220 || e.Dx() <= 0 {
		t.Fatalf("editor %v", e)
	}
}

// An empty chip field is as tall as the text field it stands in for.
func TestTokenFieldEmptyIsFieldHeight(t *testing.T) {
	f := testTokenField(t, 300)
	got := f.Measure(layout.Loose(300, 400)).Y
	if want := style.FieldHeight(f.Look().Metrics()); got < want {
		t.Fatalf("empty field is %v tall, want at least %v", got, want)
	}
}

// The chips are in the accessibility tree as the items they are, each
// with a named remove button that answers a screen reader's press.
func TestTokenFieldAccessibility(t *testing.T) {
	f := testTokenField(t, 400)
	f.SetTokens([]string{"ada@example.com"})
	f.Measure(layout.Loose(400, 400))
	f.Arrange(paintengine2d.XYWH(0, 0, 400, 120))

	var n a11y.Node
	f.Describe(&n)
	if n.Role != a11y.RoleList || n.Name != "To" || n.Value != "ada@example.com" {
		t.Fatalf("field node %+v", n)
	}
	chip := f.Chips()[0]
	var cn a11y.Node
	chip.Describe(&cn)
	if cn.Role != a11y.RoleListItem || cn.Name != "ada@example.com" {
		t.Fatalf("chip node %+v", cn)
	}
	items := chip.AccessibleItems()
	if len(items) != 1 || items[0].Name != "Remove ada@example.com" {
		t.Fatalf("items %+v", items)
	}
	if !chip.AccessibleAction(0, a11y.ActionDefault) {
		t.Fatal("the remove button refused ActionDefault")
	}
	if len(f.Tokens()) != 0 {
		t.Fatalf("after the action: %v", f.Tokens())
	}
}
