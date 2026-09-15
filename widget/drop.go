package widget

import (
	"net/url"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// DropTarget is implemented by components that take things dragged from
// other apps: files ("text/uri-list") or text ("text/plain"). DropTypes are
// the types it takes, best first; Drop gets the data read in one of them
// and reports whether it took it.
type DropTarget interface {
	DropTypes() []string
	Drop(e DropEvent) bool
}

// DropHover is implemented by drop targets that show a drag over them
// (a highlight): DragOver while one is, DragLeave when it goes.
type DropHover interface {
	DragOver(pos paintengine2d.Point)
	DragLeave()
}

// DropEvent is a drop from another app.
type DropEvent struct {
	Pos  paintengine2d.Point // in the target's local space
	Mime string
	Data []byte
	// Paths are the local files of a text/uri-list drop.
	Paths []string
	// Text is a text drop's text.
	Text string
}

// textMimes are the names text goes by, UTF-8 first.
var textMimes = []string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain", "STRING", "TEXT"}

// PickDropMime is the type to read a drop in: the first of the target's
// types the drag offers ("text/plain" matches any name text goes by).
func PickDropMime(want, offered []string) string {
	has := func(m string) bool {
		for _, o := range offered {
			if o == m {
				return true
			}
		}
		return false
	}
	for _, w := range want {
		if w == "text/plain" {
			for _, t := range textMimes {
				if has(t) {
					return t
				}
			}
			continue
		}
		if has(w) {
			return w
		}
	}
	return ""
}

// NewDropEvent decodes a drop's data: the local paths of a uri-list, or
// the text.
func NewDropEvent(pos paintengine2d.Point, mime string, data []byte) DropEvent {
	e := DropEvent{Pos: pos, Mime: mime, Data: data}
	if mime == "text/uri-list" {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if u, err := url.Parse(line); err == nil && u.Scheme == "file" {
				e.Paths = append(e.Paths, u.Path)
			}
		}
		return e
	}
	e.Text = strings.ReplaceAll(string(data), "\r\n", "\n")
	return e
}
