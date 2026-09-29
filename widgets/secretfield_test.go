package widgets

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func testSecret(t *testing.T) *SecretField {
	t.Helper()
	f := NewSecretField("Passphrase")
	f.SetLook(style.DarkLook())
	f.SetHost(&host{})
	f.Measure(layout.Loose(240, 40))
	f.Arrange(paintengine2d.XYWH(0, 0, 240, 28))
	f.RequestFocus()
	return f
}

func typeSecret(f *SecretField, s string) {
	for _, r := range s {
		f.TextInput(r)
	}
}

// Typing, editing and reading back.
func TestSecretFieldEdits(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "correct horse")
	if got := string(f.Bytes()); got != "correct horse" {
		t.Fatalf("%q", got)
	}
	if f.Len() != len("correct horse") {
		t.Fatalf("len %d", f.Len())
	}
	f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if got := string(f.Bytes()); got != "correct hors" {
		t.Fatalf("after backspace: %q", got)
	}
	f.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	f.KeyPress(widget.KeyEvent{Key: platform.KeyDelete})
	if got := string(f.Bytes()); got != "orrect hors" {
		t.Fatalf("after delete: %q", got)
	}
	// Select all, then type over.
	f.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl})
	typeSecret(f, "x")
	if got := string(f.Bytes()); got != "x" {
		t.Fatalf("after select-all and type: %q", got)
	}
	// Multi-byte runes move whole.
	f.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl})
	typeSecret(f, "wörld")
	f.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	f.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	f.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if f.Caret() != 3 {
		t.Fatalf("caret %d, want the byte after ö", f.Caret())
	}
	if f.Len() != 5 {
		t.Fatalf("len %d runes", f.Len())
	}
}

// Bytes is a copy: wiping it must not empty the field, and wiping the
// field must not depend on the caller having kept the copy.
func TestSecretFieldBytesIsACopy(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "hunter2")
	b := f.Bytes()
	WipeBytes(b)
	if got := string(f.Bytes()); got != "hunter2" {
		t.Fatalf("wiping the copy changed the field: %q", got)
	}
	if !bytes.Equal(b, make([]byte, len(b))) {
		t.Fatal("WipeBytes left something")
	}
}

// After Wipe the backing array is all zero — not just the slice, the
// whole array, because a shrunk slice still owns what is past its end.
func TestSecretFieldWipeZeroesTheArray(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "correct horse battery staple")
	arr := f.buf[:cap(f.buf)]
	f.Wipe()
	if !f.Empty() || f.Len() != 0 {
		t.Fatal("the field is not empty")
	}
	for i, c := range arr {
		if c != 0 {
			t.Fatalf("byte %d of the old array is %#x", i, c)
		}
	}
}

// Deleting must not leave the removed bytes past the end of the slice,
// where they would sit in the array until it is reused.
func TestSecretFieldShrinkWipesTheTail(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "correct horse battery staple")
	arr := f.buf[:cap(f.buf)]
	f.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl})
	f.KeyPress(widget.KeyEvent{Key: platform.KeyBackspace})
	if bytes.Contains(arr, []byte("battery")) {
		t.Fatal("the deleted text is still in the array")
	}
}

// Growing past the capacity must zero the array it leaves behind.
func TestSecretFieldGrowthWipesTheOldArray(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "short")
	old := f.buf[:cap(f.buf)]
	for len(f.buf) <= cap(old) {
		typeSecret(f, "0123456789")
	}
	if &old[0] == &f.buf[0] {
		t.Skip("the buffer did not move")
	}
	for i, c := range old {
		if c != 0 {
			t.Fatalf("byte %d of the abandoned array is %#x", i, c)
		}
	}
}

// Ctrl+C and Ctrl+X put nothing on the clipboard, and consume the key so
// it cannot reach anything above the field that would.
func TestSecretFieldRefusesCopyAndCut(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "hunter2")
	f.KeyPress(widget.KeyEvent{Key: platform.KeyA, Mods: platform.ModCtrl})
	platform.ClipboardSet("untouched")
	for _, k := range []platform.Key{platform.KeyC, platform.KeyX} {
		if !f.KeyPress(widget.KeyEvent{Key: k, Mods: platform.ModCtrl}) {
			t.Fatalf("%v bubbled out of the field", k)
		}
		if got := platform.ClipboardGet(); got != "untouched" {
			t.Fatalf("%v put %q on the clipboard", k, got)
		}
	}
	// And cut did not delete either — it is refused, not half-done.
	if got := string(f.Bytes()); got != "hunter2" {
		t.Fatalf("cut changed the field: %q", got)
	}
}

// Paste is allowed, and stops at the first newline: the rest belongs to
// whatever came after it, and taking it silently would make a wrong
// passphrase that looks right.
func TestSecretFieldPaste(t *testing.T) {
	f := testSecret(t)
	platform.ClipboardSet("from the clipboard\nand the next line")
	f.KeyPress(widget.KeyEvent{Key: platform.KeyV, Mods: platform.ModCtrl})
	if got := string(f.Bytes()); got != "from the clipboard" {
		t.Fatalf("%q", got)
	}
}

// Middle click is the PRIMARY paste, and this field neither reads that
// selection nor writes it.
func TestSecretFieldIgnoresMiddleClick(t *testing.T) {
	f := testSecret(t)
	platform.ClipboardSet("not this")
	f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(10, 10), Button: platform.ButtonMiddle})
	if !f.Empty() {
		t.Fatalf("a middle click pasted %q", string(f.Bytes()))
	}
}

// The field is not a drag source and not an input-method target: both
// are interfaces, and not implementing them is what turns the feature
// off everywhere it is asked about.
func TestSecretFieldIsNotADragSourceOrIMETarget(t *testing.T) {
	var f any = testSecret(t)
	if _, ok := f.(interface {
		DragAt(paintengine2d.Point) *widget.Drag
	}); ok {
		t.Fatal("the secret field offers a drag")
	}
	if _, ok := f.(widget.IMETarget); ok {
		t.Fatal("the secret field is an IME target, so an input method would see the passphrase")
	}
	// A TextField is both, which is the difference this widget exists for.
	var tf any = NewTextField("", "", nil)
	if _, ok := tf.(widget.IMETarget); !ok {
		t.Fatal("a text field is no longer an IME target; this test is checking the wrong thing")
	}
}

// The accessibility tree gets the role, the length and nothing else.
func TestSecretFieldAccessibility(t *testing.T) {
	f := testSecret(t)
	typeSecret(f, "hunter2")
	var n a11y.Node
	f.Describe(&n)
	if n.Role != a11y.RolePasswordField {
		t.Fatalf("role %v", n.Role)
	}
	if n.Name != "Passphrase" {
		t.Fatalf("name %q", n.Name)
	}
	if strings.Contains(n.Value, "hunter") || n.Value != strings.Repeat("•", 7) {
		t.Fatalf("value %q", n.Value)
	}
	if f.AccessibleSetText("typed by a screen reader") {
		t.Fatal("AccessibleSetText was accepted")
	}
	if got := string(f.Bytes()); got != "hunter2" {
		t.Fatalf("AccessibleSetText changed the field: %q", got)
	}
}

// Clicking puts the caret on a rune boundary, both hidden and revealed.
func TestSecretFieldClickLandsOnARuneBoundary(t *testing.T) {
	for _, reveal := range []bool{false, true} {
		f := testSecret(t)
		f.Reveal = reveal
		typeSecret(f, "wörld wörld")
		for x := float32(0); x < 240; x += 3 {
			f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(x, 14)})
			i := f.Caret()
			if i < 0 || i > len(f.buf) {
				t.Fatalf("reveal=%v x=%v: caret %d", reveal, x, i)
			}
			if i > 0 && i < len(f.buf) && f.buf[i]&0xC0 == 0x80 {
				t.Fatalf("reveal=%v x=%v: caret %d is inside a rune", reveal, x, i)
			}
		}
	}
}

// Painting works both ways and neither leaves the contents in the shape
// cache — the hidden path draws bullets, the revealed one draws bytes.
func TestSecretFieldPaintsWithoutCachingTheSecret(t *testing.T) {
	f := testSecret(t)
	const secret = "zzp-secret-only-in-this-test"
	typeSecret(f, secret)
	for _, reveal := range []bool{false, true} {
		f.Reveal = reveal
		img := paintengine2d.NewImage(240, 28)
		ctx := paintengine2d.NewContext(img)
		f.Paint(ctx)
	}
	for _, fc := range []*style.Font{f.Look().Font(), f.Look().MonoFont()} {
		for _, k := range style.ShapeCacheKeysForTest(fc) {
			if strings.Contains(k, "zzp-secret") {
				t.Fatalf("the shape cache holds %q", k)
			}
		}
	}
}

// The rule this widget exists for, checked on the source: nothing in it
// turns the buffer into a string. A string cannot be wiped, so one
// conversion anywhere — a debug print, a helper reached for out of habit
// — undoes the whole widget, and it would not show up in any behavioural
// test.
func TestSecretFieldSourceMakesNoString(t *testing.T) {
	for _, f := range []string{"secretfield.go", "a11y.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		body := string(src)
		if f == "a11y.go" {
			// Only this widget's part of the shared file.
			i := strings.Index(body, "func (f *SecretField) Describe")
			if i < 0 {
				t.Fatal("the secret field's a11y node has moved")
			}
			body = body[i:]
		}
		for _, bad := range []string{"string(f.buf", "string(b)", "fmt.Sprint", "%s"} {
			if strings.Contains(body, bad) {
				t.Fatalf("%s contains %q", f, bad)
			}
		}
	}
}
