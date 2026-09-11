package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// CursorHost is implemented by app.Window: change the OS pointer shape.
type CursorHost interface {
	SetCursor(platform.Cursor)
}

// CursorHint reports the pointer shape for a local point (splitter sash).
type CursorHint interface {
	CursorAt(local paintengine2d.Point) platform.Cursor
}

// ApplyCursor sets the host pointer when the host implements CursorHost.
func ApplyCursor(h Host, c platform.Cursor) {
	if ch, ok := h.(CursorHost); ok {
		ch.SetCursor(c)
	}
}
