//go:build darwin && cgo

package platform

/*
#include "appkit_darwin.h"
*/
import "C"

// The pointer shape, on the window side. The NSCursor mapping itself
// is in cursor_darwin.go, which the tray uses too; this is the half
// that belongs to a window.

// SetCursor gives this window's pointer a shape ([CursorSurface]).
//
// Two halves, as on Windows, and only having both makes it stick.
// Setting the cursor changes it now, which is what a widget wants when
// the pointer crosses into it; but AppKit asks the view again with
// cursorUpdate: whenever the pointer moves over it, and the default
// answer puts back whatever the cursor rects say. So the shape is
// remembered here and answered there too — that second half is where
// the arrow-over-text flicker comes from when a backend does the first
// alone.
//
// Only over the content area. Over the frame — the resize edges, the
// title bar, the traffic lights — the window's own tracking wins and
// AppKit's shape is the right one: that is what makes the resize
// pointers appear at all, since [WindowFrame.StartResize] cannot.
func (s *akSurface) SetCursor(c Cursor) {
	if s == nil {
		return
	}
	s.cursor = c
	if s.win != nil {
		C.uitk_ak_set_cursor(s.win, C.int(darwinCursorKind(c)))
	}
	if s.pointerIn {
		applySystemCursor(c)
	}
}

// Cursor is the shape this window last asked for.
func (s *akSurface) Cursor() Cursor { return s.cursor }

var _ CursorSurface = (*akSurface)(nil)
