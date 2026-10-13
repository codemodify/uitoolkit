package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A rich-text line is drawn one fragment at a time, so the line itself
// has to be put in the order it is read — and the pieces of a
// right-to-left stretch go in the opposite order to the one they were
// written in.
//
// It needs no font: the order follows from the text (UAX #9), and the
// fragments are measured afterwards.
func TestARichTextLineIsOrderedForReading(t *testing.T) {
	frag := func(text string, start int) rtFrag {
		return rtFrag{text: text, start: start, end: start + len([]rune(text))}
	}
	build := func(texts ...string) *rtLine {
		ln := &rtLine{}
		at := 0
		for _, s := range texts {
			ln.frags = append(ln.frags, frag(s, at))
			at += len([]rune(s))
		}
		ln.start, ln.end = 0, at
		return ln
	}
	order := func(ln *rtLine) []string {
		var out []string
		for _, f := range ln.frags {
			out = append(out, f.text)
		}
		return out
	}

	t.Run("a line that reads one way is left alone", func(t *testing.T) {
		ln := build("The ", "quick", " brown fox")
		was := order(ln)
		orderBidi(ln)
		if got := order(ln); strings.Join(got, "|") != strings.Join(was, "|") {
			t.Errorf("the line was reordered to %v", got)
		}
	})

	t.Run("two styled Hebrew runs swap", func(t *testing.T) {
		// "hello " then bold "שלום" then plain " עולם" then " ok".
		ln := build("hello ", "שלום", " עולם", " ok")
		if style.DirectionRuns("hello שלום עולם ok") == nil {
			t.Skip("this build has no bidirectional algorithm (font_complex)")
		}
		orderBidi(ln)
		got := order(ln)
		at := func(want string) int {
			for i, s := range got {
				if strings.Contains(s, want) {
					return i
				}
			}
			return -1
		}
		shalom, olam := at("שלום"), at("עולם")
		if shalom < 0 || olam < 0 {
			t.Fatalf("a piece went missing: %v", got)
		}
		if olam > shalom {
			t.Errorf("the second Hebrew word is drawn after the first: %v", got)
		}
		if h := at("hello"); h != 0 {
			t.Errorf("the line does not open with the English: %v", got)
		}
		if k := at("ok"); k != len(got)-1 {
			t.Errorf("the line does not close with the English: %v", got)
		}
		// Nothing lost on the way.
		var n int
		for _, s := range got {
			n += len([]rune(s))
		}
		if n != ln.end {
			t.Errorf("the line is %d runes after ordering, was %d", n, ln.end)
		}
	})
}

// A selection across two fragments of a line that reads one way is one
// box, not two abutting ones: the pieces that touch are joined, so the
// highlight has no seam down the middle of a word.
func TestARichTextSelectionJoinsTouchingPieces(t *testing.T) {
	f := style.BakeFamily(style.FamilyUI, style.WeightRegular, 16, paintengine2d.White)
	ln := &rtLine{}
	at := float32(0)
	for _, s := range []string{"hello ", "world"} {
		fr := rtFrag{text: s, font: f, x: at}
		if n := len(ln.frags); n > 0 {
			fr.start = ln.frags[n-1].end
		}
		fr.end = fr.start + len([]rune(s))
		fr.w = f.Advance(s)
		ln.frags = append(ln.frags, fr)
		at += fr.w
	}
	ln.start, ln.end = 0, ln.frags[1].end

	spans := ln.selectionSpans(0, ln.end)
	if len(spans) != 1 {
		t.Fatalf("selecting the whole line gave %d boxes, want 1: %v", len(spans), spans)
	}
	if spans[0][0] != 0 {
		t.Errorf("the box starts at %.2f, not at the line's start", spans[0][0])
	}
	if want := at; spans[0][1] < want-0.5 || spans[0][1] > want+0.5 {
		t.Errorf("the box ends at %.2f, want %.2f", spans[0][1], want)
	}
	if sp := ln.selectionSpans(3, 3); sp != nil {
		t.Errorf("an empty selection gave %v", sp)
	}
}
