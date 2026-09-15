package widgets

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// typeAheadNow is the clock of type-ahead find (tests replace it).
var typeAheadNow = time.Now

// typeAheadReset is the pause after which typing starts a new search.
const typeAheadReset = time.Second

// typeAhead is the incremental find of item views: typing jumps to the next
// row whose text starts with what was typed, a pause starts over, and one
// letter repeated cycles through the rows starting with it (Explorer,
// Finder, GTK and Qt views all do this).
type typeAhead struct {
	prefix string
	at     time.Time
}

// next feeds r and returns the row to go to among n rows, the current one
// being cur (-1 for none), or -1 when r is not a search or nothing matches.
func (ta *typeAhead) next(r rune, cur, n int, text func(int) string) (int, bool) {
	if n <= 0 || text == nil || !unicode.IsPrint(r) {
		return -1, false
	}
	now := typeAheadNow()
	if now.Sub(ta.at) > typeAheadReset {
		ta.prefix = ""
	}
	if ta.prefix == "" && unicode.IsSpace(r) {
		// Space selects or toggles; it only searches inside a prefix.
		return -1, false
	}
	ta.at = now
	ta.prefix += string(unicode.ToLower(r))
	p := ta.prefix
	start := cur
	if utf8.RuneCountInString(p) == 1 {
		start = cur + 1
	}
	if i := findPrefix(p, start, n, text); i >= 0 {
		return i, true
	}
	if first, size := utf8.DecodeRuneInString(p); strings.Count(p, string(first))*size == len(p) {
		return findPrefix(string(first), cur+1, n, text), true
	}
	return -1, true
}

// findPrefix is the first row at or after start (wrapping) whose text starts
// with p, ignoring case, or -1.
func findPrefix(p string, start, n int, text func(int) string) int {
	start = ((start % n) + n) % n
	for k := 0; k < n; k++ {
		i := (start + k) % n
		if strings.HasPrefix(strings.ToLower(text(i)), p) {
			return i
		}
	}
	return -1
}

// contextKey reports whether e asks for the context menu from the keyboard:
// the Menu key or Shift+F10.
func contextKey(e widget.KeyEvent) bool {
	return e.Key == platform.KeyMenu || (e.Key == platform.KeyF10 && e.Mods.Shift())
}

// contextPoint is where a keyboard-opened context menu appears: under the
// start of row (local rect) of view, or at the view's top-left when there
// is no row, in window coordinates.
func contextPoint(view widget.Component, row paintengine2d.Rect) paintengine2d.Point {
	o := widget.DeviceOrigin(view)
	if row.Empty() {
		return paintengine2d.Pt(o.X+8, o.Y+8)
	}
	return paintengine2d.Pt(o.X+row.Min.X+16, o.Y+row.Max.Y)
}
