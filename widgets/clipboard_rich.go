package widgets

import (
	"sync"

	"github.com/codemodify/uitoolkit/platform"
)

// The clipboard's rich flavour.
//
// The platform clipboard carries text. A rich copy puts its plain text
// there, for every other application, and keeps its HTML here beside it;
// a paste reads the HTML back only while the clipboard still holds the
// text that went with it — so copying from another application in the
// meantime is never pasted as our stale HTML. Between rich-text editors
// of this process (two documents, an editor and a mail composer) the
// formatting survives a copy and paste. A drag carries both flavours to
// other applications as well (text/html and text/plain), since a drag's
// types are offered natively.

var richClip struct {
	mu          sync.Mutex
	html, plain string
	ok          bool
}

// SetClipboardHTML puts html on the clipboard with plain as its text
// flavour. Call it from the UI goroutine, as platform.ClipboardSet.
func SetClipboardHTML(html, plain string) {
	platform.ClipboardSet(plain)
	richClip.mu.Lock()
	richClip.html, richClip.plain, richClip.ok = html, plain, true
	richClip.mu.Unlock()
}

// ClipboardHTML is the clipboard's HTML flavour, when the clipboard still
// holds the text a rich copy put there.
func ClipboardHTML() (html string, ok bool) {
	richClip.mu.Lock()
	h, p, have := richClip.html, richClip.plain, richClip.ok
	richClip.mu.Unlock()
	if !have {
		return "", false
	}
	if platform.ClipboardGet() != p {
		return "", false
	}
	return h, true
}
