package widget

import (
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
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

// DropHoverMime is a DropHover whose highlight depends on what is being
// dragged: mime is the type the drop would be read in ([PickDropMime] of
// the target's types and the drag's), so a tab strip can mark where a
// torn-off tab would go and where a file would land differently. A target
// that implements it is asked this way and never through DragOver.
type DropHoverMime interface {
	DropHover
	DragOverMime(pos paintengine2d.Point, mime string)
}

// DropEvent is a drop: from another app, or from this one.
type DropEvent struct {
	Pos  paintengine2d.Point // in the target's local space
	Mime string
	Data []byte
	// Paths are the local files of a text/uri-list drop.
	Paths []string
	// Remote are the entries of a text/uri-list that did not become a
	// local path: a file: URI belonging to another host, and any other
	// scheme. They are kept whole, as they were written, because there is
	// nothing to open and the only honest thing to hand an application is
	// the URI itself.
	//
	// A drop can be both: a selection of three files and one network
	// share fills Paths with three entries and Remote with one. A target
	// that only opens local files uses Paths and tells the user the rest
	// could not be taken; one that speaks a protocol can look here.
	Remote []string
	// Text is a text drop's text.
	Text string
	// Action is what the source and target settled on: a copy unless
	// both agreed to move or link. A target that moved the data must
	// say so — the source removes its original on the strength of it.
	Action platform.DragAction
	// Payload and Source are set only when the drag started in this
	// application ([Drag.Payload], [Drag.Source]): the payload never
	// went through a type at all, and the source is the component it
	// came from, so a target can tell a real move from a drop on
	// itself.
	Payload any
	Source  Component
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

// NewDropEvent decodes a drop's data: a uri-list's local paths, the URIs
// of it that are not local paths, or the text.
func NewDropEvent(pos paintengine2d.Point, mime string, data []byte) DropEvent {
	e := DropEvent{Pos: pos, Mime: mime, Data: data}
	if mime == "text/uri-list" {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			u, err := url.Parse(line)
			// Not a URI at all: a bare path, which this format does not
			// carry, or — the reason for the length test — a Windows one,
			// whose drive letter parses as a one-letter scheme.
			if err != nil || len(u.Scheme) < 2 {
				continue
			}
			if u.Scheme == "file" && u.Opaque == "" && localHost(u.Host) {
				if u.Path != "" {
					e.Paths = append(e.Paths, u.Path)
				}
				continue
			}
			e.Remote = append(e.Remote, line)
		}
		return e
	}
	e.Text = strings.ReplaceAll(string(data), "\r\n", "\n")
	return e
}

// localHost reports whether a file: URI's authority names this machine,
// which is what makes its path a path here.
//
// RFC 8089 gives three ways to say "this machine": no authority at all
// ("file:///etc/hosts"), "localhost", or the machine's own name. Anything
// else is another computer, and its path means nothing locally —
// "file://build-server/etc/hosts" is not /etc/hosts. Appending it to
// Paths let an application open a *different* file of the same name
// without ever learning the drop was remote, which is a wrong answer
// rather than a failure, and the dangerous shape of this bug.
func localHost(h string) bool {
	if h == "" || strings.EqualFold(h, "localhost") {
		return true
	}
	// A hostname may carry a port in the authority. One on a file: URI is
	// meaningless, and it is not ours either way.
	if strings.Contains(h, ":") {
		return false
	}
	for _, n := range thisHost() {
		if strings.EqualFold(h, n) {
			return true
		}
	}
	return false
}

// thisHost is what this machine answers to: its hostname and, when that
// is qualified, the short name in front of the first dot. Read once —
// a drop is not the moment to ask the resolver, and a machine that is
// renamed under a running application is not a case worth a syscall per
// dropped file.
var thisHost = func() func() []string {
	var once sync.Once
	var names []string
	return func() []string {
		once.Do(func() {
			h, err := os.Hostname()
			if err != nil || h == "" {
				return
			}
			names = []string{h}
			if i := strings.IndexByte(h, '.'); i > 0 {
				names = append(names, h[:i])
			}
		})
		return names
	}
}()
