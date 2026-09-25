package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every line a wrap returns has to fit the width it was asked for.
func wrapFits(t *testing.T, f *Font, lines []string, maxW float32) {
	t.Helper()
	for _, ln := range lines {
		if w := f.Advance(ln); w > maxW {
			t.Errorf("line %q is %.1f wide, asked for %.1f", ln, w, maxW)
		}
	}
}

func TestFontWrapBreaksAtSpaces(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	const text = "The quick brown fox jumps over the lazy dog"
	maxW := f.Advance("The quick brown fox ")
	lines := f.Wrap(text, maxW)
	if len(lines) < 3 {
		t.Fatalf("expected several lines, got %q", lines)
	}
	wrapFits(t, f, lines, maxW)
	if got := strings.Join(lines, " "); got != text {
		t.Fatalf("the wrap changed the text:\n got %q\nwant %q", got, text)
	}
	for _, ln := range lines {
		if strings.HasPrefix(ln, " ") || strings.HasSuffix(ln, " ") {
			t.Errorf("the break left a space on %q", ln)
		}
	}
	// Wide enough: one line, untouched.
	if one := f.Wrap(text, f.Advance(text)+1); len(one) != 1 || one[0] != text {
		t.Fatalf("a text that fits must come back whole, got %q", one)
	}
}

func TestFontWrapKeepsNewlines(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	lines := f.Wrap("one\ntwo\r\n\nfour", 10_000)
	want := []string{"one", "two", "", "four"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q want %q", lines, want)
	}
}

func TestFontWrapBreaksAnOverlongWord(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	const word = "Supercalifragilisticexpialidocious"
	maxW := f.Advance("Supercali")
	lines := f.Wrap(word, maxW)
	if len(lines) < 3 {
		t.Fatalf("a word four times the line should break, got %q", lines)
	}
	wrapFits(t, f, lines, maxW)
	if got := strings.Join(lines, ""); got != word {
		t.Fatalf("the break lost letters: %q", got)
	}
	// A word after a space breaks too, and the space still breaks first.
	lines = f.Wrap("ab "+word, maxW)
	if lines[0] != "ab" {
		t.Fatalf("the space should break before the long word: %q", lines)
	}
	wrapFits(t, f, lines, maxW)
}

// A width narrower than a single glyph must still terminate, one rune a line.
func TestFontWrapNarrowerThanAGlyph(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	lines := f.Wrap("abcd", 1)
	if len(lines) != 4 {
		t.Fatalf("want a rune a line, got %q", lines)
	}
}

func TestFontWrapExpandsTabs(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	lines := f.Wrap("a\tb", 10_000)
	if len(lines) != 1 || lines[0] != "a"+wrapTab+"b" {
		t.Fatalf("a tab should become spaces, got %q", lines)
	}
	// And it is a place a line may break, and takes room while it does.
	wide := f.Wrap("aaa\tbbb", f.Advance("aaa"+wrapTab))
	if len(wide) != 2 || wide[0] != "aaa" || wide[1] != "bbb" {
		t.Fatalf("a tab should break like a space, got %q", wide)
	}
	if f.Wrap("", 100) != nil {
		t.Fatal("empty text has no lines")
	}
}

func TestFontPrefixAddsNothing(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	const full = "Welcome to Mail on uitoolkit"
	if got := f.Prefix(full, f.Advance(full)); got != full {
		t.Fatalf("a text that fits comes back whole, got %q", got)
	}
	head := "Welcome to Mail"
	got := f.Prefix(full, f.Advance(head))
	if !strings.HasPrefix(full, got) {
		t.Fatalf("%q is not a prefix of %q", got, full)
	}
	if got == "" || len(got) < len(head) {
		t.Fatalf("prefix %q is shorter than the %q it was measured for", got, head)
	}
	if f.Advance(got) > f.Advance(head) {
		t.Fatalf("prefix %q does not fit", got)
	}
	if containsEllipsis(got) {
		t.Fatalf("Prefix must not add an ellipsis: %q", got)
	}
	// This is the one thing Fit cannot answer, and Fit is now built on it.
	if f.Prefix(full, 0) != "" || f.Prefix("", 100) != "" {
		t.Fatal("nothing fits in nothing")
	}
}

// A wrap must not depend on the shared shape cache, and must not flush it:
// the same paragraph laid out twice gives the same lines.
func TestFontWrapIsStable(t *testing.T) {
	f := BakeFont(16, paintengine2d.White)
	text := strings.Repeat("alpha beta gamma delta ", 40)
	a := f.Wrap(text, 200)
	b := f.Wrap(text, 200)
	if strings.Join(a, "|") != strings.Join(b, "|") {
		t.Fatal("two wraps of one paragraph disagree")
	}
	wrapFits(t, f, a, 200)
}
