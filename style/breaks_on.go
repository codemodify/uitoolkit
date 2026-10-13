//go:build font_complex

package style

import (
	"sync"

	"github.com/go-text/typesetting/segmenter"
)

var breakPool = sync.Pool{New: func() any { return &segmenter.Segmenter{} }}

// lineBreaks marks every rune a line may legally break *before*, by the
// Unicode line breaking algorithm (UAX #14).
//
// A space is one of them, so this subsumes the space rule it stands
// beside; the ones that are not are what the space rule cannot see. Han
// and Kana break between almost any two characters and have no spaces
// to break at, so a Japanese paragraph wrapped by spaces is one long
// line. The rule also *forbids* breaks the space rule would take — it
// keeps a closing bracket, a full stop or a 」 with the text it belongs
// to, and will not leave an opening quote at the end of a line.
//
// It answers nil for a string with no opportunity in it but the ones a
// space already gives, so the caller keeps its own cheap path.
func lineBreaks(runes []rune) []bool {
	if len(runes) < 2 {
		return nil
	}
	seg := breakPool.Get().(*segmenter.Segmenter)
	defer breakPool.Put(seg)
	seg.Init(runes)
	out := make([]bool, len(runes)+1)
	interesting := false
	for it := seg.LineIterator(); it.Next(); {
		at := it.Line().Offset
		if at <= 0 || at > len(runes) {
			continue
		}
		out[at] = true
		// A break a space would have given anyway is no reason to walk
		// this table at all.
		if runes[at-1] != ' ' {
			interesting = true
		}
	}
	if !interesting {
		return nil
	}
	return out
}
