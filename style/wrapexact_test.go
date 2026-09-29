package style

import (
	"math"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Text measured at its own width does not wrap.
//
// Wrap adds rune advances, which does not see pair kerning and so says a
// line is wider than it paints. That is safe for deciding a line is
// full and wrong for deciding it is too full: a label measured at
// exactly its own Advance width came back as two lines with a word
// broken, and every wrapping label in a form was a line taller than the
// line it drew.
func TestWrapDoesNotBreakTextThatFits(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	for _, s := range []string{
		"/usr/bin/mail",
		"Wave AV To",
		"Unlock the vault",
		"AVATAR",
		"correct horse battery staple",
		"a",
		"WWWWWWWWWW",
	} {
		w := f.Advance(s)
		for _, at := range []float32{w, ceilUp(w), ceilUp(w) + 1} {
			lines := f.Wrap(s, at)
			if len(lines) != 1 {
				t.Fatalf("%q is %v wide and wrapped to %d lines at %v: %q", s, w, len(lines), at, lines)
			}
			if lines[0] != s {
				t.Fatalf("%q came back as %q", s, lines[0])
			}
		}
	}
}

// It still wraps what genuinely does not fit, and still breaks a word
// too long for the line rather than letting it run out of the box.
func TestWrapStillWraps(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	const s = "correct horse battery staple"
	half := f.Advance(s) / 2
	lines := f.Wrap(s, half)
	if len(lines) < 2 {
		t.Fatalf("%q at half its width is %d lines", s, len(lines))
	}
	for i, l := range lines {
		if f.Advance(l) > half+0.01 && !strings.Contains(l, " ") {
			continue // one word wider than the line: broken, not overflowing
		}
		if f.Advance(l) > half+0.01 {
			t.Fatalf("line %d %q is %v wide, over %v", i, l, f.Advance(l), half)
		}
	}
	if strings.Join(lines, " ") != s {
		t.Fatalf("the text changed: %q", lines)
	}
	// A single word longer than the line is broken rather than dropped.
	long := f.Wrap("supercalifragilistic", f.Advance("super"))
	if len(long) < 2 {
		t.Fatalf("a long word was not broken: %q", long)
	}
	if strings.Join(long, "") != "supercalifragilistic" {
		t.Fatalf("the word changed: %q", long)
	}
}

// And the whole point, at the level the bug was reported: a label
// measured at its own unbounded width is the height it was unbounded.
func TestLabelMeasuredAtItsOwnWidthIsOneLine(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	for _, s := range []string{"/usr/bin/mail", "Wave AV To", "Personal vault"} {
		// What Label.Measure does: wrap at MaxW less its 2px padding.
		unbounded := f.Advance(s) + 2
		at := ceilUp(unbounded) - 2
		if lines := f.Wrap(s, at); len(lines) != 1 {
			t.Fatalf("%q unbounded is %v; at %v it is %d lines", s, unbounded, at, len(lines))
		}
	}
}

func ceilUp(v float32) float32 { return float32(math.Ceil(float64(v))) }
