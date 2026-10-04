package style

import (
	"strings"
	"testing"
)

// A word longer than the line breaks after a separator rather than in the
// middle of a name.
//
// A path, a URL or a key has no spaces in it, so it used to be cut wherever
// the line happened to end — through the middle of a file name, which reads
// as two words that are not there. Broken after its last slash it still reads
// as a path.
func TestALongWordBreaksAfterASeparator(t *testing.T) {
	f := DarkLook().ControlFont(RoleField)
	if f == nil {
		t.Skip("the look has no field font")
	}
	const path = "/home/user/go/src/github.com/codemodify/secretvault/plan.txt"
	lines := f.Wrap(path, 150)
	if len(lines) < 2 {
		t.Fatalf("the path did not wrap at all: %q", lines)
	}
	// Nothing was lost or gained.
	if got := strings.Join(lines, ""); got != path {
		t.Fatalf("the wrap changed the text: %q", got)
	}
	// Every line but the last ends at a separator, and no line starts with
	// one — which is what a break *before* the separator looks like.
	for i, l := range lines[:len(lines)-1] {
		if l == "" {
			t.Errorf("line %d is empty", i)
			continue
		}
		if !CanBreakAfter(rune(l[len(l)-1])) {
			t.Errorf("line %d is %q: it does not end at a separator", i, l)
		}
	}
	for i, l := range lines {
		if l != "" && i > 0 && CanBreakAfter(rune(l[0])) && len(l) > 1 {
			t.Errorf("line %d is %q: it begins with the separator the break should have kept", i, l)
		}
	}
}

// A hyphenated word breaks after a hyphen, and never leaves a line holding
// nothing but the hyphen. The first attempt recorded the break point for a
// separator that was itself the rune that overflowed, which threw away the
// last good one and produced exactly that.
func TestAHyphenatedWordDoesNotLeaveALineOfOneHyphen(t *testing.T) {
	f := DarkLook().ControlFont(RoleField)
	if f == nil {
		t.Skip("the look has no field font")
	}
	lines := f.Wrap("a-very-long-hyphenated-identifier-name", 80)
	for i, l := range lines {
		if strings.TrimSpace(l) == "-" {
			t.Errorf("line %d holds nothing but a hyphen: %q", i, lines)
		}
	}
}

// Words still break at spaces, which come first. A separator inside a word
// that fits is not a break point at all: a sentence must not be broken at a
// full stop because some other text is a path.
func TestWordsStillBreakAtSpacesNotAtSeparators(t *testing.T) {
	f := DarkLook().ControlFont(RoleField)
	if f == nil {
		t.Skip("the look has no field font")
	}
	const text = "aaa.bbb ccc ddd"
	// Room for the first word and a little more, so the line is full in the
	// middle of "ccc" and the break has a choice: the space, or the stop
	// inside a word that fits whole.
	lines := f.Wrap(text, f.Advance("aaa.bbb cc"))
	if len(lines) < 2 {
		t.Fatalf("the text did not wrap: %q", lines)
	}
	if lines[0] != "aaa.bbb" {
		t.Errorf("the first line is %q, want %q: it broke at the stop rather than the space", lines[0], "aaa.bbb")
	}
}

// A word with nothing to break at is still broken, because something has to
// give: a glyph run wider than the line cannot be placed any other way.
func TestAWordWithNoSeparatorIsStillBroken(t *testing.T) {
	f := DarkLook().ControlFont(RoleField)
	if f == nil {
		t.Skip("the look has no field font")
	}
	lines := f.Wrap(strings.Repeat("a", 60), 60)
	if len(lines) < 2 {
		t.Errorf("a run of 60 a's in a 60-pixel line came out as %d line(s)", len(lines))
	}
}
