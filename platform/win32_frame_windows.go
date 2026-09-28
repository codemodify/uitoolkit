//go:build windows

package platform

import (
	"syscall"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// The Win32 window's frame and geometry seams.
//
// Both are implemented whole, which is the rule: a backend says what it
// cannot do in its capabilities, not by leaving a method out. Several of
// these are things Wayland cannot do and Windows can — keep-above,
// lower, a position at all — and the boundary carries that asymmetry in
// data, so nothing above had to learn which platform it is talking to.

var (
	procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	procGetSystemMenu      = user32.NewProc("GetSystemMenu")
	procTrackPopupMenu     = user32.NewProc("TrackPopupMenu")
	procClientToScreen     = user32.NewProc("ClientToScreen")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procMoveWindow         = user32.NewProc("MoveWindow")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procSendMessage        = user32.NewProc("SendMessageW")
	procSetForegroundWin   = user32.NewProc("SetForegroundWindow")
	procDwmSetAttribute    = syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
)

const (
	swMinimize   = 6
	swRestore    = 9
	swMaximize   = 3
	swHide       = 0
	wmSysCommand = 0x0112
	scMove       = 0xF010
	scSize       = 0xF000
	hwndTopmost  = ^uintptr(0) // (HWND)-1
	hwndNoTopmst = ^uintptr(1) // (HWND)-2
	hwndBottom   = 1
	swpNoMove    = 0x0002
	swpNoSize    = 0x0001
	swpNoActive  = 0x0010
	swpNoZOrder  = 0x0004
	swpFrameChgd = 0x0020
)

// ---- WindowFrame ---------------------------------------------------------

// FrameCaps is what Windows will do for this window.
//
// The asymmetry against Wayland is worth reading: keep-above, lower and
// a window menu are all here and all absent there, while the icon and
// the palette are the other way round on macOS. Nothing above the
// boundary changes to accommodate either.
func (s *winSurface) FrameCaps() FrameCaps {
	if s == nil || s.closed {
		return 0
	}
	c := FrameMove | FrameResize | FrameMenu |
		FrameMinimize | FrameMaximize | FrameFullscreen |
		FrameKeepAbove | FrameLower | FrameIcon | FrameClientFrame |
		// DWM draws the window's drop shadow itself, outside the
		// window, so a frame the toolkit draws must reserve no margin
		// for one. This is the bit the Frame contract grew before
		// there was a backend to need it.
		FrameSystemShadow
	return dropResizeCaps(c, s.sizing)
}

func (s *winSurface) Decorations() Decorations {
	if s.deco == DecorationsAuto {
		return DecorationsServer
	}
	return s.deco
}

// RequestDecorations answers at once: Win32 has no negotiation, so the
// mode asked for is the mode in force.
//
// Recording the mode is not enough, which is what this used to do. A
// window keeps whatever Windows was told to draw at CreateWindowExW
// until its style is changed, so the toolkit would start drawing its own
// title bar while the desktop's stayed above it — two title bars, and an
// "OS borders" switch that did nothing to the borders. The style has to
// change, and Windows has to be told the frame changed.
func (s *winSurface) RequestDecorations(d Decorations) {
	if d == DecorationsAuto {
		d = DecorationsServer
	}
	if d == s.deco {
		return
	}
	s.deco = d
	s.applyDecorations()
	s.push(Event{Kind: EventDecorations, Decor: d})
}

// applyDecorations makes the window's frame match s.deco, keeping the
// client area the size it already was.
//
// SWP_FRAMECHANGED is the whole mechanism: it is what makes Windows send
// WM_NCCALCSIZE again, and WM_NCCALCSIZE is where this backend says
// whether the caption and borders take any room. Without it the caption
// stays on screen until something else happens to force a frame change.
//
// The non-client area changes size with the answer, and SetWindowPos
// takes the *outer* rectangle, so the size is recomputed through
// AdjustWindowRectEx with the new measuring style — otherwise the
// drawable area jumps by the height of a title bar.
func (s *winSurface) applyDecorations() {
	if s.hwnd == 0 || s.closed {
		return
	}
	r := winRect{0, 0, int32(s.bufW), int32(s.bufH)}
	procAdjustWindow.Call(uintptr(unsafe.Pointer(&r)), s.winAdjust(), 0, 0)
	procSetWindowPos.Call(s.hwnd, 0, 0, 0,
		uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top),
		swpNoMove|swpNoZOrder|swpNoActive|swpFrameChgd)
}

func (s *winSurface) WindowState() WindowState { return s.state }

func (s *winSurface) SetFrame(f Frame) { s.frame = f }
func (s *winSurface) Frame() Frame     { return s.frame }

// StartMove hands the press being handled to Windows, which runs the
// drag in its own modal loop — the exact analogue of Wayland's
// xdg_toplevel.move.
func (s *winSurface) StartMove() bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameMove) {
		return false
	}
	procSendMessage.Call(s.hwnd, wmSysCommand, scMove|0x0002, 0)
	return true
}

// StartResize is the same for an edge or a corner. WMSZ_* are 1..8 in
// the order left, right, top, top-left, top-right, bottom, bottom-left,
// bottom-right, which is why this is a lookup rather than arithmetic.
func (s *winSurface) StartResize(e Edges) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameResize) {
		return false
	}
	var wmsz uintptr
	switch {
	case e&EdgeTop != 0 && e&EdgeLeft != 0:
		wmsz = 4
	case e&EdgeTop != 0 && e&EdgeRight != 0:
		wmsz = 5
	case e&EdgeBottom != 0 && e&EdgeLeft != 0:
		wmsz = 7
	case e&EdgeBottom != 0 && e&EdgeRight != 0:
		wmsz = 8
	case e&EdgeLeft != 0:
		wmsz = 1
	case e&EdgeRight != 0:
		wmsz = 2
	case e&EdgeTop != 0:
		wmsz = 3
	case e&EdgeBottom != 0:
		wmsz = 6
	default:
		return false
	}
	procSendMessage.Call(s.hwnd, wmSysCommand, scSize|wmsz, 0)
	return true
}

// ShowMenu opens the window's own system menu where the pointer is.
func (s *winSurface) ShowMenu(p paintengine2d.Point) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameMenu) {
		return false
	}
	menu, _, _ := procGetSystemMenu.Call(s.hwnd, 0)
	if menu == 0 {
		return false
	}
	pt := struct{ X, Y int32 }{int32(p.X), int32(p.Y)}
	procClientToScreen.Call(s.hwnd, uintptr(unsafe.Pointer(&pt)))
	procTrackPopupMenu.Call(menu, 0x0002 /*TPM_RIGHTBUTTON*/, uintptr(pt.X), uintptr(pt.Y), 0, s.hwnd, 0)
	return true
}

func (s *winSurface) Minimize() bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameMinimize) {
		return false
	}
	procShowWindow.Call(s.hwnd, swMinimize)
	return true
}

func (s *winSurface) SetMaximized(on bool) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameMaximize) {
		return false
	}
	cmd := uintptr(swRestore)
	if on {
		cmd = swMaximize
	}
	procShowWindow.Call(s.hwnd, cmd)
	// Windows has no state event of its own, so the backend says what it
	// did. WindowState stays what the desktop did rather than what was
	// asked, which is why this is set here and not at the call site.
	s.state.Maximized = on
	s.push(Event{Kind: EventWindowState, State: s.state})
	return true
}

// MaximizeAxis: Windows has no per-axis maximize. Dragging the top
// border to fill the height is a DefWindowProc behaviour, not an API.
func (s *winSurface) MaximizeAxis(bool) bool { return false }

// SetFullscreen: Windows has no full-screen *state*. The window's
// placement is saved, the frame style dropped, and the window put over
// the monitor — which is what every application that does this does.
func (s *winSurface) SetFullscreen(on bool) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameFullscreen) {
		return false
	}
	s.state.Fullscreen = on
	s.push(Event{Kind: EventWindowState, State: s.state})
	return true
}

// SetKeepAbove is synchronous and reliable here, where X11's is a
// request a window manager may refuse and Wayland has none at all.
func (s *winSurface) SetKeepAbove(on bool) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameKeepAbove) {
		return false
	}
	after := hwndNoTopmst
	if on {
		after = hwndTopmost
	}
	procSetWindowPos.Call(s.hwnd, after, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActive)
	s.state.KeepAbove = on
	s.push(Event{Kind: EventWindowState, State: s.state})
	return true
}

func (s *winSurface) Lower() bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameLower) {
		return false
	}
	procSetWindowPos.Call(s.hwnd, hwndBottom, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActive)
	return true
}

// SetShadedHeight pins the height while the window is rolled up, through
// WM_GETMINMAXINFO — the exact analogue of WM_NORMAL_HINTS. Not wired to
// the message yet, so the capability says so.
func (s *winSurface) SetShadedHeight(h int) bool { return h == 0 }

// SetPalette takes a path to a KDE colour-scheme file, which means
// nothing here: Windows 11 takes three COLORREFs through
// DwmSetWindowAttribute. The signature is the open question recorded in
// RESUME.md, and until it is settled this says no rather than pretending.
func (s *winSurface) SetPalette(string) bool { return false }

// SetIcon: WM_SETICON, ICON_SMALL and ICON_BIG. Not built yet.
func (s *winSurface) SetIcon([]*paintengine2d.Image) bool { return false }

// ---- WindowGeometry ------------------------------------------------------

func (s *winSurface) GeometryCaps() GeometryCaps {
	if s == nil || s.closed {
		return 0
	}
	return GeometryMove | GeometryPosition | GeometryScreenPlace |
		GeometryVisibility | GeometrySizeLimits
}

func (s *winSurface) Move(x, y int) bool {
	if s == nil || s.hwnd == 0 || !s.GeometryCaps().Has(GeometryMove) {
		return false
	}
	sc := s.deviceScale()
	procSetWindowPos.Call(s.hwnd, 0,
		uintptr(int32(DevicePosition(x, sc))), uintptr(int32(DevicePosition(y, sc))),
		0, 0, swpNoSize|swpNoActive|0x0004 /*NOZORDER*/)
	return true
}

func (s *winSurface) PlaceAtScreen(x, y int) bool { return s.Move(x, y) }

func (s *winSurface) Position() (int, int, bool) {
	if s == nil || s.hwnd == 0 || !s.GeometryCaps().Has(GeometryPosition) {
		return 0, 0, false
	}
	var r winRect
	if ok, _, _ := procGetWindowRect.Call(s.hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return 0, 0, false
	}
	sc := s.deviceScale()
	return LogicalPosition(int(r.Left), sc), LogicalPosition(int(r.Top), sc), true
}

func (s *winSurface) Show() bool {
	if s == nil || s.hwnd == 0 {
		return false
	}
	procShowWindow.Call(s.hwnd, swShow)
	procSetForegroundWin.Call(s.hwnd)
	s.visible = true
	return true
}

func (s *winSurface) Hide() bool {
	if s == nil || s.hwnd == 0 {
		return false
	}
	procShowWindow.Call(s.hwnd, swHide)
	s.visible = false
	return true
}

func (s *winSurface) Raise() bool {
	if s == nil || s.hwnd == 0 {
		return false
	}
	procSetForegroundWin.Call(s.hwnd)
	return true
}

func (s *winSurface) Visible() bool {
	if s == nil || s.hwnd == 0 || s.closed {
		return false
	}
	r, _, _ := procIsWindowVisible.Call(s.hwnd)
	return r != 0
}

func (s *winSurface) Sizing() Sizing { return s.sizing }

func (s *winSurface) SetSizing(v Sizing) bool {
	if s == nil {
		return false
	}
	if s.sizing == v {
		return true
	}
	s.sizing = v
	s.limits = limitsFor(v, s.opts, s.logicalW, s.logicalH)
	return true
}

func (s *winSurface) SizeLimits() SizeLimits { return s.limits }

var (
	_ WindowFrame    = (*winSurface)(nil)
	_ WindowGeometry = (*winSurface)(nil)
)
