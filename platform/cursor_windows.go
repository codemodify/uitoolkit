//go:build windows

package platform

import "syscall"

var (
	user32Cursor    = syscall.NewLazyDLL("user32.dll")
	procLoadCursorW = user32Cursor.NewProc("LoadCursorW")
	procSetCursor   = user32Cursor.NewProc("SetCursor")
)

// applySystemCursor sets the process pointer via LoadCursorW + SetCursor.
// A Win32 window backend should also apply this from WM_SETCURSOR so the
// shape sticks while the pointer is over the client area.
func applySystemCursor(c Cursor) {
	h, _, _ := procLoadCursorW.Call(0, win32CursorID(c))
	if h != 0 {
		procSetCursor.Call(h)
	}
}

// SetCursor gives this window's pointer a shape ([CursorSurface]).
//
// Two halves, and only having both makes it stick. SetCursor changes the
// shape now, which is what a widget wants when the pointer crosses into
// it; but Windows asks the window again with WM_SETCURSOR every time the
// pointer moves within it, and the default answer loads the class cursor
// back. So the shape is remembered and answered there too, and
// WM_SETCURSOR is where the arrow-over-text flicker comes from when a
// backend does the first half alone.
//
// Only over the client area. Over the frame — the resize edges, the
// caption, the buttons of a window the desktop draws — the low word of
// WM_SETCURSOR's lParam is not HTCLIENT, and Windows' own shape is the
// right one: that is what makes the resize arrows appear.
func (s *winSurface) SetCursor(c Cursor) {
	if s == nil {
		return
	}
	s.cursor = c
	if s.pointerIn {
		applySystemCursor(c)
	}
}

// Cursor is the shape this window last asked for.
func (s *winSurface) Cursor() Cursor { return s.cursor }

var _ CursorSurface = (*winSurface)(nil)
