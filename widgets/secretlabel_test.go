package widgets

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

func testSecretLabel(t *testing.T, value func() []byte) *SecretLabel {
	t.Helper()
	l := NewSecretLabel(value)
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})
	l.Measure(layout.Loose(240, 40))
	l.Arrange(paintengine2d.XYWH(0, 0, 240, 24))
	return l
}

func paintLabel(l *SecretLabel) *paintengine2d.Image {
	img := paintengine2d.NewImage(240, 24)
	l.Paint(paintengine2d.NewContext(img))
	return img
}

// Hidden and revealed paint differently, and neither leaves the value in
// the face's shaped-run cache — which is the reason the revealed path
// does not go through Font.Draw.
func TestSecretLabelPaintsWithoutCachingTheSecret(t *testing.T) {
	const secret = "zzl-secret-only-in-this-test"
	l := testSecretLabel(t, func() []byte { return []byte(secret) })
	hidden := paintLabel(l)
	l.Reveal = true
	revealed := paintLabel(l)
	if pixelsDiffering(hidden, revealed) < 8 {
		t.Fatal("revealing changed nothing")
	}
	for _, f := range []*style.Font{l.Look().Font(), l.Look().MonoFont()} {
		for _, k := range style.ShapeCacheKeysForTest(f) {
			if strings.Contains(k, "zzl-secret") {
				t.Fatalf("the shape cache holds %q", k)
			}
		}
	}
}

// Len is a rune count and nothing else, and it follows the value.
func TestSecretLabelLen(t *testing.T) {
	val := []byte("hunter2")
	l := testSecretLabel(t, func() []byte { return val })
	if l.Len() != 7 {
		t.Fatalf("len %d", l.Len())
	}
	val = []byte("wörld")
	if l.Len() != 5 {
		t.Fatalf("len %d runes", l.Len())
	}
	val = nil
	if l.Len() != 0 {
		t.Fatalf("len %d for no value", l.Len())
	}
}

// Hide zeroes what the label was holding, so a locked vault leaves
// nothing behind in the widget.
func TestSecretLabelHideWipes(t *testing.T) {
	l := testSecretLabel(t, func() []byte { return []byte("correct horse battery staple") })
	l.Len()
	arr := l.buf[:cap(l.buf)]
	l.Hide()
	if !bytes.Equal(arr, make([]byte, len(arr))) {
		t.Fatalf("Hide left %q", arr)
	}
	if l.Len() != 0 {
		// Value still answers, so the label fills again — which is the
		// contract: Hide drops what was drawn, the source decides what
		// comes next.
		t.Logf("the value came back, as it should: %d runes", l.Len())
	}
}

// A shorter value must not leave the tail of the longer one behind in
// the buffer the label goes on using.
func TestSecretLabelShrinkWipesTheTail(t *testing.T) {
	val := []byte("correct horse battery staple")
	l := testSecretLabel(t, func() []byte { return val })
	l.Len()
	arr := l.buf[:cap(l.buf)]
	val = []byte("short")
	l.Len()
	if bytes.Contains(arr, []byte("battery")) {
		t.Fatal("the old value is still in the buffer")
	}
}

// The label takes a copy: the caller may wipe its own the moment Value
// returns, which is what a vault that hands out a fresh decryption does.
func TestSecretLabelCopiesTheValue(t *testing.T) {
	src := []byte("hunter2")
	l := testSecretLabel(t, func() []byte {
		out := make([]byte, len(src))
		copy(out, src)
		defer WipeBytes(out)
		return out
	})
	if l.Len() != 7 {
		t.Fatalf("len %d — the label read a wiped buffer", l.Len())
	}
}

// A screen reader gets neither the value nor its length, revealed or
// not. Reporting the bullets told it the length as surely as showing
// them tells the screen — and a length read aloud in an open-plan office
// is the least worth leaking of all.
func TestSecretLabelAccessibility(t *testing.T) {
	l := testSecretLabel(t, func() []byte { return []byte("hunter2") })
	l.Len()
	for _, reveal := range []bool{false, true} {
		l.Reveal = reveal
		var n a11y.Node
		l.Describe(&n)
		if strings.Contains(n.Value, "hunter") {
			t.Errorf("reveal=%v: the node says %q", reveal, n.Value)
		}
		if strings.Contains(n.Value, "•") {
			t.Errorf("reveal=%v: the node says %q, which counts out the length", reveal, n.Value)
		}
		if n.Value == "" {
			t.Errorf("reveal=%v: the node says nothing at all", reveal)
		}
	}

	// A secret of a different length reads the same.
	short := testSecretLabel(t, func() []byte { return []byte("a") })
	short.Len()
	var a, b a11y.Node
	short.Describe(&a)
	l.Reveal = false
	l.Describe(&b)
	if a.Value != b.Value {
		t.Errorf("a one-character secret reads %q and a seven-character one %q", a.Value, b.Value)
	}
}

// A fixed mask says only that there is something there: neither the
// screen nor the box gives the length away.
func TestSecretLabelFixedMask(t *testing.T) {
	l := testSecretLabel(t, func() []byte { return []byte("a-very-long-passphrase-indeed") })
	l.MaskLen = 8
	l.Len()

	if got := l.mask(); got != strings.Repeat("•", 8) {
		t.Errorf("mask %q, want eight bullets", got)
	}
	// And measured for the mask alone, not the wider of the two.
	hidden := l.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	l.Reveal = true
	shown := l.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if hidden.X >= shown.X {
		t.Errorf("hidden it measures %v and revealed %v — the box gives the length away",
			hidden.X, shown.X)
	}

	// An empty value is still empty, not eight bullets of nothing.
	e := testSecretLabel(t, func() []byte { return nil })
	e.MaskLen = 8
	e.Len()
	if got := e.mask(); got != "" {
		t.Errorf("an empty secret masks as %q", got)
	}
}

// A key is several lines, and a one-line label cannot show one.
func TestSecretLabelLines(t *testing.T) {
	const key = "-----BEGIN-----\nbody\n-----END-----"
	l := testSecretLabel(t, func() []byte { return []byte(key) })
	l.Lines = true
	l.Reveal = true
	l.Len()

	if got := len(l.valueLines()); got != 3 {
		t.Errorf("%d lines, want 3", got)
	}
	one := testSecretLabel(t, func() []byte { return []byte(key) })
	one.Reveal = true
	one.Len()
	tall := l.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	flat := one.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if tall.Y <= flat.Y {
		t.Errorf("three lines measure %v tall and one line %v", tall.Y, flat.Y)
	}
	if tall.X >= flat.X {
		t.Errorf("split into lines it should be narrower: %v vs %v", tall.X, flat.X)
	}
}

// The same source rule as the field: nothing in the widget turns the
// buffer into a string.
func TestSecretLabelSourceMakesNoString(t *testing.T) {
	src, err := os.ReadFile("secretlabel.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"string(l.buf", "string(b)", "fmt.Sprint", "%s"} {
		if strings.Contains(string(src), bad) {
			t.Fatalf("secretlabel.go contains %q", bad)
		}
	}
}
