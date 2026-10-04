//go:build darwin && cgo

package platform

import (
	"strings"
	"testing"
)

// A MIME type is not a valid UTI, and macOS will not store anything under
// one: `setData:forType:` returns NO, logs "not a valid UTI string", and the
// payload is simply not on the pasteboard.
//
// The toolkit's own two private drag types are exactly that shape —
// `application/x-uitoolkit-tab` and `application/x-uitoolkit-panel` — so
// tearing a tab or a dock panel out to another window could not work on
// macOS at all, while the same drag worked on X11 and Wayland, which take
// any MIME string. Nothing reported it because the write returned nothing.
func TestACustomMimeReachesThePasteboard(t *testing.T) {
	for _, mime := range []string{
		"application/x-uitoolkit-tab",
		"application/x-uitoolkit-panel",
		"application/x-someone-elses-type",
		"text/plain",
		"text/plain;charset=utf-8",
		"text/uri-list",
		"text/html",
	} {
		item := newDragItem()
		defer freeDragItem(item)
		if !addDragPayload(item, mime, []byte("payload")) {
			t.Errorf("%s: the payload was refused by the pasteboard", mime)
		}
	}
}

// And it comes back as the name the toolkit speaks, not as the name macOS
// stored it under — otherwise a target would be offered a type it has never
// heard of and would refuse its own application's drag.
func TestACustomMimeRoundTripsThroughItsUTI(t *testing.T) {
	for _, mime := range []string{
		"application/x-uitoolkit-tab",
		"application/x-uitoolkit-panel",
		"application/vnd.a+b;v=2",
		"x-weird/TYPE-with.dots_and_underscores",
	} {
		uti := dragTypeOfMime(mime)
		if strings.ContainsAny(uti, "/+;=_") {
			t.Errorf("%s encoded to %q, which is still not a valid UTI", mime, uti)
		}
		if got := mimeOfDragType(uti); got != mime {
			t.Errorf("%s encoded to %q and came back as %q", mime, uti, got)
		}
	}
	// A type that is not ours is not claimed as a MIME name.
	for _, uti := range []string{"public.utf8-plain-text", "public.file-url", "com.apple.pasteboard.promised-file-url"} {
		if got := mimeOfDragType(uti); got != "" {
			t.Errorf("%s was decoded as the MIME %q", uti, got)
		}
	}
}

// The two macOS has names of its own for keep them, so a file dragged to the
// Finder is a file and text dragged to TextEdit is text.
func TestTheTypesMacOSKnowsKeepTheirOwnNames(t *testing.T) {
	for mime, want := range map[string]string{
		"text/plain":               "public.utf8-plain-text",
		"text/plain;charset=utf-8": "public.utf8-plain-text",
		"text/uri-list":            "public.file-url",
	} {
		if got := dragTypeOfMime(mime); got != want {
			t.Errorf("%s travels as %q, want %q", mime, got, want)
		}
	}
}
