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
	procMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	procCreateIconIndirect = user32.NewProc("CreateIconIndirect")
	procDestroyIcon        = user32.NewProc("DestroyIcon")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procCreateBitmap       = gdi32.NewProc("CreateBitmap")
	procGetMonitorInfo     = user32.NewProc("GetMonitorInfoW")
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
	swMinimize     = 6
	swRestore      = 9
	swMaximize     = 3
	swHide         = 0
	wmSysCommand   = 0x0112
	scMove         = 0xF010
	scSize         = 0xF000
	hwndTopmost    = ^uintptr(0) // (HWND)-1
	hwndNoTopmst   = ^uintptr(1) // (HWND)-2
	hwndBottom     = 1
	swpNoMove      = 0x0002
	swpNoSize      = 0x0001
	swpNoActive    = 0x0010
	swpNoZOrder    = 0x0004
	swpFrameChgd   = 0x0020
	monitorNearest = 0x0002
	wmSetIcon      = 0x0080
	iconSmall      = 0
	iconBig        = 1
	smCXIcon       = 11
	smCXSmIcon     = 49
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
		// FrameShade: a window can be rolled up to its title bar here.
		// SetWindowPos is not clamped by the minimum tracking size a
		// window states — that governs a resize the user drags, not one
		// the program asks for — so the roll-up holds instead of
		// springing back open.
		FrameShade |
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
// SetFullscreen covers the monitor the window is on, or gives back the
// place it had.
//
// Windows has no fullscreen state of its own — no _NET_WM_STATE_FULLSCREEN,
// nothing to ask for — so this is the three steps an application does for
// itself: remember the placement, take the frame off the style, and set
// the window to the monitor's whole rectangle. Coming back is the same in
// reverse, and the placement is what makes a window that was maximized
// before going fullscreen maximized again after.
//
// The monitor is the one the window is *on*, not the primary: dragging a
// window to the second screen and pressing fullscreen should fill that
// screen. MONITOR_DEFAULTTONEAREST, so a window somehow off every screen
// still gets an answer.
func (s *winSurface) SetFullscreen(on bool) bool {
	if s == nil || s.hwnd == 0 || !s.FrameCaps().Has(FrameFullscreen) {
		return false
	}
	if on == s.state.Fullscreen {
		return true
	}
	if on {
		s.prePlacement = winPlacement{Length: uint32(unsafe.Sizeof(winPlacement{}))}
		if r, _, _ := procGetWindowPlacement.Call(s.hwnd, uintptr(unsafe.Pointer(&s.prePlacement))); r == 0 {
			return false
		}
		s.preStyle, _, _ = procGetWindowLongW.Call(s.hwnd, gwlStyle)
		mi := winMonitorInfo{Size: uint32(unsafe.Sizeof(winMonitorInfo{}))}
		mon, _, _ := procMonitorFromWindow.Call(s.hwnd, monitorNearest)
		if mon == 0 {
			return false
		}
		if r, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
			return false
		}
		// WS_OVERLAPPEDWINDOW off: no caption, no sizing border, and
		// nothing of the frame left to overlap the screen it is covering.
		procSetWindowLong.Call(s.hwnd, gwlStyle, s.preStyle&^wsOverlappedWindow)
		procSetWindowPos.Call(s.hwnd, 0,
			uintptr(mi.Monitor.Left), uintptr(mi.Monitor.Top),
			uintptr(mi.Monitor.Right-mi.Monitor.Left),
			uintptr(mi.Monitor.Bottom-mi.Monitor.Top),
			swpNoZOrder|swpNoActive|swpFrameChgd)
	} else {
		procSetWindowLong.Call(s.hwnd, gwlStyle, s.preStyle)
		procSetWindowPlacement.Call(s.hwnd, uintptr(unsafe.Pointer(&s.prePlacement)))
		procSetWindowPos.Call(s.hwnd, 0, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpNoZOrder|swpNoActive|swpFrameChgd)
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
// SetShadedHeight records the height a rolled-up window is held at.
//
// Nothing has to be said to Windows for the roll-up itself: the height
// this backend is about to be asked for goes through SetWindowPos, which
// the minimum tracking size does not clamp. The height is kept so that
// WM_GETMINMAXINFO could hold a *dragged* resize to it as well, which is
// the half that is not built — see docs/windows.md, and note that
// reading MINMAXINFO means turning an LPARAM back into a pointer, which
// is the one thing `go vet` will not have.
func (s *winSurface) SetShadedHeight(h int) bool {
	s.shadedH = max(h, 0)
	return true
}

// SetPalette takes a path to a KDE colour-scheme file, which means
// nothing here: Windows 11 takes three COLORREFs through
// DwmSetWindowAttribute. The signature is the open question recorded in
// RESUME.md, and until it is settled this says no rather than pretending.
func (s *winSurface) SetPalette(string) bool { return false }

// SetIcon gives the window its icon, for the title bar, the task bar and
// Alt+Tab.
//
// Windows asks for two sizes and uses them in different places — the
// small one in the caption and the task bar, the big one in Alt+Tab and
// the task manager — so the nearest image at or above each is chosen
// rather than one being stretched for both. SM_CXSMICON and SM_CXICON
// ask what those sizes are *for this window's DPI*, not for 96.
//
// The old icons are destroyed after the new ones are in place, never
// before: WM_SETICON answers with the handle it is replacing and that
// handle is still on screen until it does.
func (s *winSurface) SetIcon(imgs []*paintengine2d.Image) bool {
	if s == nil || s.hwnd == 0 || s.closed || !s.FrameCaps().Has(FrameIcon) {
		return false
	}
	if len(imgs) == 0 {
		s.setOneIcon(iconSmall, 0)
		s.setOneIcon(iconBig, 0)
		return true
	}
	sc := s.deviceScale()
	want := func(metric uintptr, fallback int) int {
		n, _, _ := procGetSystemMetrics.Call(metric)
		if n == 0 {
			return DevicePixels(fallback, sc)
		}
		return int(n)
	}
	ok := false
	for _, c := range []struct {
		which uintptr
		side  int
	}{
		{iconSmall, want(smCXSmIcon, 16)},
		{iconBig, want(smCXIcon, 32)},
	} {
		h := winIconFrom(pickIcon(imgs, c.side))
		if h == 0 {
			continue
		}
		s.setOneIcon(c.which, h)
		ok = true
	}
	return ok
}

// setOneIcon installs one of the window's two icons and destroys the one
// it replaced.
func (s *winSurface) setOneIcon(which, icon uintptr) {
	old, _, _ := procSendMessage.Call(s.hwnd, wmSetIcon, which, icon)
	if old != 0 && old != icon {
		procDestroyIcon.Call(old)
	}
}

// pickIcon is the image to use at side pixels: the smallest that is at
// least that big, or the biggest there is when none of them reach it.
// Scaling down a larger icon beats scaling up a smaller one.
func pickIcon(imgs []*paintengine2d.Image, side int) *paintengine2d.Image {
	var best *paintengine2d.Image
	for _, im := range imgs {
		if im == nil || im.Width <= 0 || im.Height <= 0 {
			continue
		}
		switch {
		case best == nil:
			best = im
		case best.Width < side:
			if im.Width > best.Width {
				best = im
			}
		case im.Width >= side && im.Width < best.Width:
			best = im
		}
	}
	return best
}

// winIconFrom turns one image into an HICON, or 0.
//
// The colour bitmap is a top-down 32-bit DIB section, which is the one
// shape that carries an alpha channel: an icon built the old way, from a
// colour bitmap and a 1-bit mask, has hard edges. The mask is supplied
// all the same because ICONINFO requires one, and it is all zeroes —
// "every pixel opaque" — so the alpha is what decides.
//
// Windows wants the pixels premultiplied, which is how paintengine2d
// already keeps them, so this is the same BGRA swizzle Present does.
func winIconFrom(im *paintengine2d.Image) uintptr {
	if im == nil || im.Width <= 0 || im.Height <= 0 {
		return 0
	}
	w, h := im.Width, im.Height
	var bi winBitmapInfoHeader
	bi.Size = uint32(unsafe.Sizeof(bi))
	bi.Width, bi.Height = int32(w), int32(-h)
	bi.Planes, bi.BitCount = 1, 32
	bi.Compression = biRGB

	var bits *byte
	colour, _, _ := procCreateDIBSect.Call(0, uintptr(unsafe.Pointer(&bi)),
		dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if colour == 0 || bits == nil {
		return 0
	}
	dst := unsafe.Slice(bits, w*h*4)
	stride := im.Stride
	if stride == 0 {
		stride = w * 4
	}
	for y := 0; y < h; y++ {
		si, di := y*stride, y*w*4
		for x := 0; x < w; x++ {
			dst[di+0] = im.Pix[si+2]
			dst[di+1] = im.Pix[si+1]
			dst[di+2] = im.Pix[si+0]
			dst[di+3] = im.Pix[si+3]
			si, di = si+4, di+4
		}
	}
	// A 1-bit mask of zeroes: ICONINFO will not take nil for it.
	mask, _, _ := procCreateBitmap.Call(uintptr(w), uintptr(h), 1, 1, 0)
	if mask == 0 {
		procDeleteObject.Call(colour)
		return 0
	}
	info := struct {
		Icon         int32
		HotX, HotY   uint32
		Mask, Colour uintptr
	}{Icon: 1, Mask: mask, Colour: colour}
	icon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	// CreateIconIndirect copies both bitmaps; they are ours to free.
	procDeleteObject.Call(colour)
	procDeleteObject.Call(mask)
	return icon
}

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

// winPlacement is WINDOWPLACEMENT and winMonitorInfo is MONITORINFO:
// where a window was before it went fullscreen, and how big the screen
// it is going to cover is.
type winPlacement struct {
	Length, Flags  uint32
	ShowCmd        uint32
	MinPosition    winPoint
	MaxPosition    winPoint
	NormalPosition winRect
}

type winMonitorInfo struct {
	Size          uint32
	Monitor, Work winRect
	Flags         uint32
}
