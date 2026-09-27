//go:build windows

package platform

// The Win32 backend: a real top-level window, a message pump, and a DIB
// blit to put the toolkit's pixels on screen.
//
// It is pure Go. Every call goes through syscall.NewLazyDLL, the way
// status_windows.go already talks to shell32 and user32, so this backend
// cross-compiles from Linux with CGO_ENABLED=0 and needs no toolchain in
// the guest. That is worth protecting: the Linux backends need cgo, and
// a Windows backend that needed a compiler on Windows would be far
// harder to keep honest from here.
//
// Two Win32 facts shape the code more than anything else:
//
//   - **The window procedure is re-entrant.** SetWindowPos, MoveWindow
//     and friends dispatch WM_SIZE synchronously, and WM_ENTERSIZEMOVE
//     runs a modal loop inside Windows for the whole of a drag. Anything
//     the procedure does must therefore be safe to do *while the toolkit
//     is somewhere in the middle of its own call*. So the procedure only
//     ever appends to a queue, and Poll drains it — never the reverse.
//   - **Messages belong to the thread that created the window.** They
//     are only delivered to that thread's queue, which is why the
//     toolkit's run loop is pinned to the main OS thread (app's init).

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// user32 and procPostMessage are bound in status_windows.go, which got
// here first; these are the rest.
var (
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClass  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx = user32.NewProc("CreateWindowExW")
	procDefWindowProc  = user32.NewProc("DefWindowProcW")
	procDestroyWindow  = user32.NewProc("DestroyWindow")
	procShowWindow     = user32.NewProc("ShowWindow")
	procPeekMessage    = user32.NewProc("PeekMessageW")
	procTranslateMsg   = user32.NewProc("TranslateMessage")
	procDispatchMsg    = user32.NewProc("DispatchMessageW")
	procSetWindowText  = user32.NewProc("SetWindowTextW")
	procGetClientRect  = user32.NewProc("GetClientRect")
	procGetDC          = user32.NewProc("GetDC")
	procReleaseDC      = user32.NewProc("ReleaseDC")
	procInvalidateRect = user32.NewProc("InvalidateRect")
	procGetDpiForWin   = user32.NewProc("GetDpiForWindow")
	procMsgWaitForObjs = user32.NewProc("MsgWaitForMultipleObjectsEx")
	procAdjustWindow   = user32.NewProc("AdjustWindowRectEx")
	procSetWindowPos   = user32.NewProc("SetWindowPos")

	procSetDIBitsToDev = gdi32.NewProc("SetDIBitsToDevice")

	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
)

const (
	wsOverlappedWindow = 0x00CF0000
	swShow             = 5
	pmRemove           = 0x0001
	cwUseDefault       = ^uintptr(0x7FFFFFFF) // 0x80000000 as a signed int

	wmDestroy     = 0x0002
	wmSize        = 0x0005
	wmClose       = 0x0010
	wmPaint       = 0x000F
	wmEraseBkgnd  = 0x0014
	wmDpiChanged  = 0x02E0
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp2  = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp2  = 0x0205
	wmMButtonDown = 0x0207
	wmMButtonUp   = 0x0208
	wmMouseWheel  = 0x020A
	wmKeyDown     = 0x0100
	wmKeyUp       = 0x0101
	wmChar        = 0x0102
	wmSetFocus    = 0x0007
	wmKillFocus   = 0x0008

	biRGB          = 0
	dibRGBColors   = 0
	qsAllInput     = 0x04FF
	mwmoInputAvail = 0x0004
)

// Win32Backend is the Windows window backend.
type Win32Backend struct{}

func (Win32Backend) Name() string { return "win32" }

// Caps: a real desktop, and Windows places windows where they ask
// (SetWindowPos), unlike Wayland.
func (Win32Backend) Caps() BackendCaps { return BackendDesktop | BackendScreenPlace }

func (b Win32Backend) NewSurface(opts WindowOptions) (Surface, error) { return newWinSurface(opts) }

// win32Registry maps a window handle to its surface. The window
// procedure is shared by every window of the class and finds its surface
// here — the same shape status_windows.go uses, and for the same reason:
// a procedure that closes over one window is wrong the moment there are
// two.
var (
	win32Mu       sync.Mutex
	win32ByHWND   = map[uintptr]*winSurface{}
	win32Class    sync.Once
	win32ClassA   *uint16
	win32ClassErr error
)

type winSurface struct {
	hwnd  uintptr
	title string

	mu    sync.Mutex
	queue []Event

	img                *paintengine2d.Image
	bgra               []byte // the same pixels in the order a DIB wants them
	bufW, bufH         int
	logicalW, logicalH int
	scale              float32

	closed bool
	torn   bool

	sizing  Sizing
	limits  SizeLimits
	opts    WindowOptions
	frame   Frame
	state   WindowState
	deco    Decorations
	visible bool
}

func win32RegisterClass() (*uint16, error) {
	win32Class.Do(func() {
		name, err := syscall.UTF16PtrFromString("uitoolkit.Window")
		if err != nil {
			win32ClassErr = err
			return
		}
		inst, _, _ := procGetModuleHandle.Call(0)
		var wc struct {
			Size       uint32
			Style      uint32
			WndProc    uintptr
			ClsExtra   int32
			WndExtra   int32
			Instance   uintptr
			Icon       uintptr
			Cursor     uintptr
			Background uintptr
			MenuName   *uint16
			ClassName  *uint16
			IconSm     uintptr
		}
		wc.Size = uint32(unsafe.Sizeof(wc))
		wc.Style = 0x0003 // CS_HREDRAW | CS_VREDRAW: a resize repaints
		wc.WndProc = syscall.NewCallback(win32Proc)
		wc.Instance = inst
		wc.ClassName = name
		if r, _, err := procRegisterClass.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			win32ClassErr = fmt.Errorf("win32: RegisterClassExW: %w", err)
			return
		}
		win32ClassA = name
	})
	return win32ClassA, win32ClassErr
}

// win32Proc is the shared window procedure. It **only queues**: the
// procedure runs re-entrantly, inside SetWindowPos and inside the modal
// loop Windows runs for a resize drag, so doing anything that could call
// back into the toolkit here would run it inside its own call.
func win32Proc(hwnd, msg, wparam, lparam uintptr) uintptr {
	win32Mu.Lock()
	s := win32ByHWND[hwnd]
	win32Mu.Unlock()
	if s == nil {
		r, _, _ := procDefWindowProc.Call(hwnd, msg, wparam, lparam)
		return r
	}
	switch msg {
	case wmClose:
		// A request, not an order: the application may keep the window
		// (close-to-tray), so the window is not destroyed here.
		s.push(Event{Kind: EventClose})
		return 0
	case wmDestroy:
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		return 0
	case wmEraseBkgnd:
		return 1 // the toolkit paints every pixel; erasing first flickers
	case wmPaint:
		s.push(Event{Kind: EventExpose, Width: s.bufW, Height: s.bufH})
		// Leave the paint to DefWindowProc so the update region is
		// validated; Present puts the pixels up.
	case wmSize:
		w, h := int(lparam&0xFFFF), int((lparam>>16)&0xFFFF)
		if w > 0 && h > 0 {
			sc := s.deviceScale()
			s.push(Event{Kind: EventResize,
				Width: LogicalPixels(w, sc), Height: LogicalPixels(h, sc)})
		}
		return 0
	case wmDpiChanged:
		// The new DPI is the low word of wParam. Windows also passes a
		// suggested rectangle in lParam, and this deliberately does not
		// read it: turning an LPARAM back into a pointer is the one
		// thing `go vet` will not have, and keeping the Windows build
		// vet-clean is worth more than the suggestion. The window is
		// re-sized from the logical size it already has and the scale
		// that just changed, which lands in the same place for the
		// ordinary case of a drag between two monitors.
		s.mu.Lock()
		s.scale = float32(uint32(wparam)&0xFFFF) / 96
		lw, lh, sc := s.logicalW, s.logicalH, s.scale
		s.mu.Unlock()
		if sc > 0 && lw > 0 && lh > 0 {
			r := winRect{0, 0, int32(DevicePixels(lw, sc)), int32(DevicePixels(lh, sc))}
			procAdjustWindow.Call(uintptr(unsafe.Pointer(&r)), wsOverlappedWindow, 0, 0)
			procSetWindowPos.Call(hwnd, 0, 0, 0,
				uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top),
				swpNoMove|0x0004|0x0010) // NOMOVE|NOZORDER|NOACTIVATE
		}
		s.push(Event{Kind: EventScale})
		return 0
	case wmSetFocus:
		s.push(Event{Kind: EventFocusIn})
		return 0
	case wmKillFocus:
		s.push(Event{Kind: EventFocusOut})
		return 0
	case wmMouseMove:
		s.push(Event{Kind: EventMouseMove, Pos: lparamPoint(lparam)})
		return 0
	case wmLButtonDown, wmRButtonDown, wmMButtonDown:
		s.push(Event{Kind: EventMouseDown, Pos: lparamPoint(lparam), Button: winButton(msg)})
		return 0
	case wmLButtonUp2, wmRButtonUp2, wmMButtonUp:
		s.push(Event{Kind: EventMouseUp, Pos: lparamPoint(lparam), Button: winButton(msg)})
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, msg, wparam, lparam)
	return r
}

type winRect struct{ Left, Top, Right, Bottom int32 }

func lparamPoint(lparam uintptr) paintengine2d.Point {
	x := int16(lparam & 0xFFFF)
	y := int16((lparam >> 16) & 0xFFFF)
	return paintengine2d.Pt(float32(x), float32(y))
}

func winButton(msg uintptr) MouseButton {
	switch msg {
	case wmRButtonDown, wmRButtonUp2:
		return ButtonRight
	case wmMButtonDown, wmMButtonUp:
		return ButtonMiddle
	}
	return ButtonLeft
}

func (s *winSurface) push(ev Event) {
	s.mu.Lock()
	s.queue = append(s.queue, ev)
	s.mu.Unlock()
}

// ---- the surface ---------------------------------------------------------

func newWinSurface(opts WindowOptions) (Surface, error) {
	class, err := win32RegisterClass()
	if err != nil {
		return nil, err
	}
	title, err := syscall.UTF16PtrFromString(opts.Title)
	if err != nil {
		return nil, err
	}
	s := &winSurface{
		title: opts.Title, opts: opts, scale: 1,
		sizing: opts.Sizing, deco: opts.Decorations,
	}
	w, h := max(opts.Width, 1), max(opts.Height, 1)
	s.logicalW, s.logicalH = w, h

	// The size asked for is the *client* area, so the frame has to be
	// added on top of it: CreateWindowExW takes the outer rectangle.
	r := winRect{0, 0, int32(w), int32(h)}
	procAdjustWindow.Call(uintptr(unsafe.Pointer(&r)), wsOverlappedWindow, 0, 0)
	inst, _, _ := procGetModuleHandle.Call(0)
	hwnd, _, callErr := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow, cwUseDefault, cwUseDefault,
		uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top),
		0, 0, inst, 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("win32: CreateWindowExW: %w", callErr)
	}
	s.hwnd = hwnd

	win32Mu.Lock()
	win32ByHWND[hwnd] = s
	win32Mu.Unlock()

	s.scale = s.readDPI()
	s.resizeBuffer(DevicePixels(w, s.scale), DevicePixels(h, s.scale))
	s.limits = limitsFor(s.sizing, opts, w, h)
	if !opts.Headless {
		procShowWindow.Call(hwnd, swShow)
		s.visible = true
	}
	return s, nil
}

// readDPI is the window's scale. GetDpiForWindow is per-monitor and
// Windows 10 or later; anything older answers 0 and gets 1.
func (s *winSurface) readDPI() float32 {
	if s.hwnd == 0 {
		return 1
	}
	dpi, _, _ := procGetDpiForWin.Call(s.hwnd)
	if dpi < 48 {
		return 1
	}
	return float32(dpi) / 96
}

func (s *winSurface) deviceScale() float32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scale <= 0 {
		return 1
	}
	return s.scale
}

func (s *winSurface) resizeBuffer(w, h int) {
	w, h = max(w, 1), max(h, 1)
	if s.img != nil && s.bufW == w && s.bufH == h {
		return
	}
	s.img = paintengine2d.NewImage(w, h)
	s.bgra = make([]byte, w*h*4)
	s.bufW, s.bufH = w, h
}

func (s *winSurface) Title() string { return s.title }

func (s *winSurface) SetTitle(t string) {
	if s.hwnd == 0 || t == s.title {
		return
	}
	s.title = t
	if p, err := syscall.UTF16PtrFromString(t); err == nil {
		procSetWindowText.Call(s.hwnd, uintptr(unsafe.Pointer(p)))
	}
}

func (s *winSurface) Size() (int, int) { return s.bufW, s.bufH }

func (s *winSurface) Scale() float32 { return s.deviceScale() }

func (s *winSurface) Buffer() *paintengine2d.Image { return s.img }

// Resize asks for a window w by h **logical** pixels across.
func (s *winSurface) Resize(w, h int) error {
	if s.hwnd == 0 || s.closed {
		return nil
	}
	w, h = max(w, 1), max(h, 1)
	sc := s.deviceScale()
	if s.logicalW == w && s.logicalH == h {
		// Still make sure the buffer matches: the scale may have moved
		// under a window whose logical size did not.
		s.resizeBuffer(DevicePixels(w, sc), DevicePixels(h, sc))
		return nil
	}
	s.logicalW, s.logicalH = w, h
	s.limits = limitsFor(s.sizing, s.opts, w, h)
	s.resizeBuffer(DevicePixels(w, sc), DevicePixels(h, sc))

	r := winRect{0, 0, int32(DevicePixels(w, sc)), int32(DevicePixels(h, sc))}
	procAdjustWindow.Call(uintptr(unsafe.Pointer(&r)), wsOverlappedWindow, 0, 0)
	// SWP_NOMOVE | SWP_NOZORDER | SWP_NOACTIVATE: only the size changes.
	procSetWindowPos.Call(s.hwnd, 0, 0, 0,
		uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x0002|0x0004|0x0010)
	return nil
}

// Present puts the buffer on screen.
//
// The pixels are swizzled into a scratch buffer first: paintengine2d
// keeps premultiplied RGBA and a 32-bit BI_RGB DIB is BGRX, so red and
// blue trade places. Only the damaged rows are converted — a full-window
// swizzle on every frame would cost more than the drawing usually does.
func (s *winSurface) Present(dirty []paintengine2d.Rect) error {
	if s.hwnd == 0 || s.closed || s.img == nil {
		return nil
	}
	s.swizzle(dirty)
	dc, _, _ := procGetDC.Call(s.hwnd)
	if dc == 0 {
		return fmt.Errorf("win32: GetDC returned nothing")
	}
	defer procReleaseDC.Call(s.hwnd, dc)

	var bi struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPelsPerMeter, YPels   int32
		ClrUsed, ClrImportant  uint32
	}
	bi.Size = uint32(unsafe.Sizeof(bi))
	bi.Width = int32(s.bufW)
	bi.Height = int32(-s.bufH) // negative: top-down, the order Pix is in
	bi.Planes, bi.BitCount = 1, 32
	bi.Compression = biRGB

	procSetDIBitsToDev.Call(dc,
		0, 0, uintptr(s.bufW), uintptr(s.bufH),
		0, 0, 0, uintptr(s.bufH),
		uintptr(unsafe.Pointer(&s.bgra[0])),
		uintptr(unsafe.Pointer(&bi)), dibRGBColors)
	return nil
}

// swizzle copies the damaged pixels into s.bgra with red and blue
// exchanged. A nil or empty damage list means the whole window.
func (s *winSurface) swizzle(dirty []paintengine2d.Rect) {
	rows := func(y0, y1, x0, x1 int) {
		stride := s.img.Stride
		if stride == 0 {
			stride = s.bufW * 4
		}
		for y := y0; y < y1; y++ {
			si := y*stride + x0*4
			di := y*s.bufW*4 + x0*4
			for x := x0; x < x1; x++ {
				s.bgra[di+0] = s.img.Pix[si+2]
				s.bgra[di+1] = s.img.Pix[si+1]
				s.bgra[di+2] = s.img.Pix[si+0]
				s.bgra[di+3] = 0xff
				si += 4
				di += 4
			}
		}
	}
	if len(dirty) == 0 {
		rows(0, s.bufH, 0, s.bufW)
		return
	}
	for _, r := range dirty {
		x0, y0 := clampInt(int(r.Min.X), 0, s.bufW), clampInt(int(r.Min.Y), 0, s.bufH)
		x1, y1 := clampInt(int(r.Max.X+0.999), 0, s.bufW), clampInt(int(r.Max.Y+0.999), 0, s.bufH)
		if x1 > x0 && y1 > y0 {
			rows(y0, y1, x0, x1)
		}
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Poll drains what the window procedure queued, after pumping whatever
// Windows has for this thread.
func (s *winSurface) Poll() []Event {
	s.pump()
	s.mu.Lock()
	ev := s.queue
	s.queue = nil
	s.mu.Unlock()
	for i := range ev {
		if ev[i].Kind == EventResize {
			sc := s.deviceScale()
			s.logicalW, s.logicalH = ev[i].Width, ev[i].Height
			s.resizeBuffer(DevicePixels(ev[i].Width, sc), DevicePixels(ev[i].Height, sc))
		}
	}
	return ev
}

// pump runs the thread's message queue dry. Dispatch calls the window
// procedure, which queues; nothing here touches the toolkit.
func (s *winSurface) pump() {
	var msg struct {
		HWnd           uintptr
		Message        uint32
		WParam, LParam uintptr
		Time           uint32
		X, Y           int32
		Private        uint32
	}
	for {
		r, _, _ := procPeekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmRemove)
		if r == 0 {
			return
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (s *winSurface) Close() error {
	if s == nil || s.torn {
		return nil
	}
	s.torn = true
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if s.hwnd != 0 {
		win32Mu.Lock()
		delete(win32ByHWND, s.hwnd)
		win32Mu.Unlock()
		procDestroyWindow.Call(s.hwnd)
		s.hwnd = 0
	}
	return nil
}

func (s *winSurface) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
