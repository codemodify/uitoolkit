//go:build font_complex

package style

import (
	"github.com/go-text/typesetting/bidi"
)

// DirectionRuns divides text into the stretches that read one way and
// returns them in the order they are drawn (UAX #9, rules P2 to L2).
//
// It answers nil for a string that reads one way throughout — which is
// nearly every string, and every string at all in a build without
// font_complex — so that a caller can keep its simple path and reach
// for this only when there is something to reorder.
//
// It is here for a layout that places *several* pieces of text on one
// line, each drawn by its own call: a rich-text line of styled runs, a
// label beside a value. Text drawn in one call is already ordered by
// [Font.Draw] itself.
func DirectionRuns(text string) []DirectionRun {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	def := bidi.LeftToRight
	if paragraphDirection(runes) == rtlDirection {
		def = bidi.RightToLeft
	}
	var para bidi.Paragraph
	brs := para.Segment(runes, def)
	n := brs.NumRuns()
	if n < 2 {
		// One run: it reads one way. A right-to-left string still
		// reads one way, and Draw has already laid it out that way.
		return nil
	}
	runs := make([]DirectionRun, n)
	levels := make([]bidi.Level, n)
	for i := 0; i < n; i++ {
		r := brs.Run(i)
		runs[i] = DirectionRun{Start: r.Start, End: r.End, RTL: !r.IsLeftToRight()}
		levels[i] = r.Level
	}
	out := make([]DirectionRun, n)
	for i, from := range visualOrder(levels) {
		out[i] = runs[from]
	}
	return out
}
