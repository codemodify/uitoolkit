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

// winCursorHost is a CursorSurface that applies system IDC_* shapes.
// Wire a Win32 HWND surface's SetCursor to this (or call applySystemCursor).
type winCursorHost struct {
	cursor Cursor
}

func (w *winCursorHost) SetCursor(c Cursor) {
	w.cursor = c
	applySystemCursor(c)
}

func (w *winCursorHost) Cursor() Cursor { return w.cursor }
