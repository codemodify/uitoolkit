//go:build windows

package platform

import "unsafe"

// A dialog on Win32.
//
// There is no window-type property to set: what makes a window read as a
// dialog is its style. WS_MINIMIZEBOX and WS_MAXIMIZEBOX come off, so
// the caption carries a close button alone and the window cannot be
// stuffed into a corner of the screen or the taskbar; WS_EX_DLGMODALFRAME
// gives the dialog border; and WS_EX_APPWINDOW comes off so a prompt does
// not get a taskbar button of its own, which is what every other Windows
// dialog does.
const (
	wsMinimizeBox    = 0x00020000
	wsMaximizeBox    = 0x00010000
	wsExDlgModalFrme = 0x00000001
	wsExAppWindow    = 0x00040000
	gwlExStyle       = ^uintptr(19) // GWL_EXSTYLE, -20

	swpFrameChanged = 0x0020
)

var (
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
)

// SetWindowRole restyles the window as a dialog, or back
// ([RoleSurface]).
func (s *winSurface) SetWindowRole(r WindowRole) bool {
	if s == nil || s.hwnd == 0 || s.closed {
		return false
	}
	style, _, _ := procGetWindowLongW.Call(s.hwnd, gwlStyle)
	ex, _, _ := procGetWindowLongW.Call(s.hwnd, gwlExStyle)
	if r == RoleDialog {
		style &^= wsMinimizeBox | wsMaximizeBox
		ex = ex&^wsExAppWindow | wsExDlgModalFrme
	} else {
		style |= wsMinimizeBox | wsMaximizeBox
		ex &^= wsExDlgModalFrme
	}
	procSetWindowLong.Call(s.hwnd, gwlStyle, style)
	procSetWindowLong.Call(s.hwnd, gwlExStyle, ex)
	// Windows caches the frame it computed; without this the style is
	// set and the window goes on wearing the old one until something
	// else makes it recalculate.
	procSetWindowPos.Call(s.hwnd, 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpNoActive|swpFrameChanged)
	return true
}

// Activate brings the window to the front and gives it the keyboard
// ([ActivateSurface]).
//
// SetForegroundWindow, not BringWindowToTop: raising is z-order and
// activating is focus, and a prompt needs the second. Windows refuses it
// from a process that is not already in the foreground — which is the
// rule that stops background windows stealing focus — and then flashes
// the taskbar button instead, which is the desktop's answer rather than
// a failure to report.
func (s *winSurface) Activate() bool {
	if s == nil || s.hwnd == 0 || s.closed {
		return false
	}
	procShowWindow.Call(s.hwnd, swShow)
	r, _, _ := procSetForegroundWindow.Call(s.hwnd)
	return r != 0
}

// Center puts the window in the middle of the work area of the monitor
// it is on ([CenterSurface]) — the work area, so it is centred in the
// space the taskbar leaves rather than behind it.
func (s *winSurface) Center() bool {
	if s == nil || s.hwnd == 0 || s.closed {
		return false
	}
	var wr winRect
	if r, _, _ := procGetWindowRect.Call(s.hwnd, uintptr(unsafe.Pointer(&wr))); r == 0 {
		return false
	}
	work, ok := s.monitorWorkArea()
	if !ok {
		return false
	}
	w := wr.Right - wr.Left
	h := wr.Bottom - wr.Top
	x := work.Left + (work.Right-work.Left-w)/2
	y := work.Top + (work.Bottom-work.Top-h)/2
	if x < work.Left {
		x = work.Left
	}
	if y < work.Top {
		y = work.Top
	}
	procSetWindowPos.Call(s.hwnd, 0, uintptr(x), uintptr(y), 0, 0,
		swpNoSize|swpNoZOrder|swpNoActive)
	return true
}

// monitorInfo is MONITORINFO: the monitor's rectangle and its work area.
type monitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}

func (s *winSurface) monitorWorkArea() (winRect, bool) {
	const monitorDefaultToNearest = 2
	mon, _, _ := procMonitorFromWindow.Call(s.hwnd, monitorDefaultToNearest)
	if mon == 0 {
		return winRect{}, false
	}
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if r, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return winRect{}, false
	}
	return mi.Work, true
}

var (
	_ RoleSurface           = (*winSurface)(nil)
	_ ActivateSurface       = (*winSurface)(nil)
	_ CenterSurface         = (*winSurface)(nil)
	_ CaptureExcludeSurface = (*winSurface)(nil)
)
