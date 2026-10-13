package widgets

import (
	"slices"
	"strings"

	"github.com/codemodify/uitoolkit/style"
)

// Ordering a line of styled runs for display.
//
// A font draws one string in the order the bidirectional algorithm says
// (style.Font.Draw), but a rich-text line is many strings — one per
// style, one per link, one per image — each drawn by a call of its own
// at an x this file decides. Nothing below the line knows that the bold
// Hebrew and the plain Latin after it belong to one sentence, so the
// line has to be ordered here, by the same rule: UAX #9 L2.
//
// In a build without font_complex, [style.DirectionRuns] answers nil for
// every string and none of this runs.

// orderBidi splits ln's fragments where the line changes direction and
// puts the pieces in the order they are drawn. It leaves the line alone
// when it reads one way, which is nearly always.
func orderBidi(ln *rtLine) {
	if len(ln.frags) < 1 {
		return
	}
	var sb strings.Builder
	for _, f := range ln.frags {
		sb.WriteString(f.text)
	}
	runs := style.DirectionRuns(sb.String())
	if len(runs) < 2 {
		return
	}

	// Where each fragment sits in the line's own text. An image is a
	// fragment of no length, and belongs to the run it falls inside.
	type placedFrag struct {
		f        rtFrag
		from, to int // rune indices into the line's text
	}
	placed := make([]placedFrag, 0, len(ln.frags))
	at := 0
	for _, f := range ln.frags {
		n := len([]rune(f.text))
		placed = append(placed, placedFrag{f: f, from: at, to: at + n})
		at += n
	}

	out := make([]rtFrag, 0, len(ln.frags))
	for _, r := range runs {
		var pieces []rtFrag
		for _, p := range placed {
			switch {
			case p.from == p.to: // an image, or an empty fragment
				if p.from >= r.Start && (p.from < r.End || r.End == at && p.from == at) {
					pieces = append(pieces, p.f)
				}
			default:
				lo, hi := max(p.from, r.Start), min(p.to, r.End)
				if lo >= hi {
					continue
				}
				pieces = append(pieces, cutFrag(p.f, lo-p.from, hi-p.from))
			}
		}
		if r.RTL {
			for a, b := 0, len(pieces)-1; a < b; a, b = a+1, b-1 {
				pieces[a], pieces[b] = pieces[b], pieces[a]
			}
		}
		out = append(out, pieces...)
	}
	// Every fragment accounted for, or the line is left as it was: a
	// line drawn with a piece of itself missing is worse than one drawn
	// in the wrong order.
	var before, after int
	for _, f := range ln.frags {
		before += len([]rune(f.text))
	}
	for _, f := range out {
		after += len([]rune(f.text))
	}
	if before != after || len(out) == 0 {
		return
	}
	ln.frags = out
}

// cutFrag is the piece of f from rune from to rune to, keeping its font,
// its style and the block offsets its characters have.
func cutFrag(f rtFrag, from, to int) rtFrag {
	if from <= 0 && to >= len([]rune(f.text)) {
		return f
	}
	rs := []rune(f.text)
	from, to = max(from, 0), min(to, len(rs))
	cut := f
	cut.text = string(rs[from:to])
	cut.start = f.start + from
	cut.end = f.start + to
	return cut
}

// selectionSpans is the x ranges, in line coordinates, that a selection
// of the characters [from, to) covers on this line.
//
// One range for a line that reads one way. A line that changes
// direction is asked fragment by fragment, since each fragment is drawn
// by its own call at its own x, and the pieces that touch are joined.
func (ln *rtLine) selectionSpans(from, to int) [][2]float32 {
	if from >= to || len(ln.frags) == 0 {
		return nil
	}
	var out [][2]float32
	for _, f := range ln.frags {
		lo, hi := max(from, f.start), min(to, f.end)
		if lo >= hi {
			continue
		}
		if f.img != nil {
			out = append(out, [2]float32{f.x, f.x + f.w})
			continue
		}
		for _, sp := range f.font.SelectionSpans(f.text, lo-f.start, hi-f.start) {
			out = append(out, [2]float32{f.x + sp[0], f.x + sp[1]})
		}
	}
	if len(out) < 2 {
		return out
	}
	slices.SortFunc(out, func(a, b [2]float32) int {
		switch {
		case a[0] < b[0]:
			return -1
		case a[0] > b[0]:
			return 1
		}
		return 0
	})
	merged := out[:0]
	for _, sp := range out {
		if m := len(merged); m > 0 && sp[0] <= merged[m-1][1]+0.01 {
			merged[m-1][1] = max(merged[m-1][1], sp[1])
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}
