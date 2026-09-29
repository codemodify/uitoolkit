package widgets

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

const pemKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
c2gtZWQyNTUxOQAAACBQ0Zt7Zp8PTnNRnvQnJvTqZ0Yx3RmVXvT0K0m0Wd5zLQ
-----END OPENSSH PRIVATE KEY-----`

func newArea(t *testing.T) *SecretArea {
	t.Helper()
	a := NewSecretArea("Private key")
	atScale(t, a, 1)
	sz := a.Measure(layout.Constraints{MaxW: 400, MaxH: -1})
	a.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	return a
}

// Some secrets are several lines: PEM and OpenSSH private keys. A
// SecretField drops a newline typed or pasted into it.
func TestSecretAreaKeepsNewlines(t *testing.T) {
	a := newArea(t)
	a.SetBytes([]byte(pemKey))

	if got := a.Bytes(); string(got) != pemKey {
		t.Errorf("the key came back changed:\n%q", got)
	}
	if got, want := a.Lines(), strings.Count(pemKey, "\n")+1; got != want {
		t.Errorf("Lines() = %d, want %d", got, want)
	}
}

// Return inserts a line rather than submitting.
func TestSecretAreaReturnInsertsALine(t *testing.T) {
	a := newArea(t)
	submitted := 0
	a.OnSubmit = func(*SecretField) { submitted++ }

	for _, r := range "ab" {
		a.TextInput(r)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	for _, r := range "cd" {
		a.TextInput(r)
	}
	if got := a.Bytes(); string(got) != "ab\ncd" {
		t.Errorf("value %q, want %q", got, "ab\ncd")
	}
	if submitted != 0 {
		t.Error("Return submitted instead of inserting a line")
	}
	// Ctrl+Return is still the way out.
	a.KeyPress(widget.KeyEvent{Key: platform.KeyReturn, Mods: platform.ModCtrl})
	if submitted != 1 {
		t.Errorf("Ctrl+Return submitted %d times", submitted)
	}
}

// A pasted key keeps its lines — taking only the first would make a
// value that looks right and is not — and comes out with Unix newlines
// whatever the clipboard had.
func TestSecretAreaPasteKeepsAndNormalizesLines(t *testing.T) {
	a := newArea(t)
	c := platform.ClipboardSetSecret([]byte("-----BEGIN-----\r\nbody\rmore\n-----END-----"), time.Minute)
	defer c.Clear()

	a.paste()
	want := "-----BEGIN-----\nbody\nmore\n-----END-----"
	if got := a.Bytes(); string(got) != want {
		t.Errorf("pasted %q, want %q", got, want)
	}
}

// The caret moves by line, and keeps the x it was aiming for so a run
// through a short line does not drag it in.
func TestSecretAreaCaretMovesByLine(t *testing.T) {
	a := newArea(t)
	a.SetBytes([]byte("aaaaaaaa\nbb\ncccccccc"))
	a.caret = 0
	a.selA, a.selB = 0, 0

	a.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	atEnd := a.Caret()
	if atEnd != 8 {
		t.Fatalf("End put the caret at %d, want 8", atEnd)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if got := a.Caret(); got != 11 {
		t.Errorf("down from the end of a long line went to %d, want the end of the short one (11)", got)
	}
	// And down again returns to the x it was aiming for.
	a.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if got := a.Caret(); got != 20 {
		t.Errorf("down again went to %d, want 20 — the goal x was dropped", got)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	a.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if got := a.Caret(); got != 8 {
		t.Errorf("back up went to %d, want 8", got)
	}
}

// Home and End are the line's, not the value's; Ctrl makes them the
// value's.
func TestSecretAreaHomeAndEnd(t *testing.T) {
	a := newArea(t)
	a.SetBytes([]byte("one\ntwo\nthree"))
	a.caret, a.selA, a.selB = 5, 5, 5

	a.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if got := a.Caret(); got != 4 {
		t.Errorf("Home went to %d, want the line's start (4)", got)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if got := a.Caret(); got != 7 {
		t.Errorf("End went to %d, want the line's end (7)", got)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyHome, Mods: platform.ModCtrl})
	if got := a.Caret(); got != 0 {
		t.Errorf("Ctrl+Home went to %d, want 0", got)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyEnd, Mods: platform.ModCtrl})
	if got := a.Caret(); got != 13 {
		t.Errorf("Ctrl+End went to %d, want 13", got)
	}
}

// Every rule the one-line field has, the area has, because it is the
// same widget: copy, cut and the PRIMARY selection are refused, and the
// input method stays off.
func TestSecretAreaKeepsTheFieldsRules(t *testing.T) {
	a := newArea(t)
	a.SetBytes([]byte(pemKey))

	if _, ok := any(a).(widget.IMETarget); ok {
		t.Error("a secret area must not be an input-method target")
	}
	if _, ok := any(a).(widget.SecretTarget); !ok {
		t.Error("a secret area must declare itself a secret target")
	}
	for _, k := range []platform.Key{platform.KeyC, platform.KeyX} {
		if !a.KeyPress(widget.KeyEvent{Key: k, Mods: platform.ModCtrl}) {
			t.Errorf("Ctrl+%v was not swallowed, so it would bubble", k)
		}
	}
	if got := a.Bytes(); string(got) != pemKey {
		t.Error("a refused copy changed the value")
	}
}

// Wipe zeroes a multi-line secret as thoroughly as a one-line one.
func TestSecretAreaWipe(t *testing.T) {
	a := newArea(t)
	key := []byte(pemKey)
	a.SetBytes(key)
	a.Wipe()
	if !a.Empty() || a.Lines() != 1 {
		t.Errorf("not empty after Wipe: %d lines", a.Lines())
	}
	if !bytes.Equal(key, make([]byte, len(key))) {
		t.Error("the handed-over key was not wiped")
	}
}

// And the source carries no string conversion of the buffer, the same
// rule the one-line field is held to.
func TestSecretAreaSourceMakesNoString(t *testing.T) {
	src, err := os.ReadFile("secretarea.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"string(a.buf", "string(f.buf"} {
		if bytes.Contains(src, []byte(bad)) {
			t.Errorf("secretarea.go contains %q: a secret must never become a Go string", bad)
		}
	}
}
