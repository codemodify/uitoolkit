//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifInfo    = 0x00000010
	nisHidden  = 0x00000001
	wmApp      = 0x8000
	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205
	ninBalloonUserClick = wmApp + 5
	hwndMessage = ^uintptr(2) // HWND_MESSAGE = -3
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	procNotifyIcon   = shell32.NewProc("Shell_NotifyIconW")
	procCreateWindow = user32.NewProc("CreateWindowExW")
	procDefWindow    = user32.NewProc("DefWindowProcW")
	procRegister     = user32.NewProc("RegisterClassExW")
	procGetMessage   = user32.NewProc("GetMessageW")
	procTranslate    = user32.NewProc("TranslateMessage")
	procDispatch     = user32.NewProc("DispatchMessageW")
	procPostQuit     = user32.NewProc("PostQuitMessage")
	procLoadIcon     = user32.NewProc("LoadIconW")
	procDestroyWin   = user32.NewProc("DestroyWindow")
)

type winStatusItem struct {
	mu      sync.Mutex
	opts    StatusItemOptions
	icon    StatusIcon
	tooltip string
	title   string
	menu    []StatusMenuItem
	hwnd    uintptr
	nid     notifyIconData
	closed  bool
}

type notifyIconData struct {
	Size     uint32
	Wnd      uintptr
	ID       uint32
	Flags    uint32
	Callback uint32
	Icon     uintptr
	Tip      [128]uint16
	State    uint32
	StateMask uint32
	Info     [256]uint16
	Timeout  uint32
	InfoTitle [64]uint16
	InfoFlags uint32
}

func newNativeStatusItem(opts StatusItemOptions) (StatusItem, error) {
	item := &winStatusItem{
		opts:    opts,
		icon:    opts.Icon,
		tooltip: opts.Tooltip,
		title:   opts.Title,
		menu:    copyMenu(opts.Menu),
	}
	if err := item.start(); err != nil {
		return newStubStatusItem(opts), nil
	}
	return item, nil
}

func nativeStatusItemAvailable() bool { return true }

func (w *winStatusItem) start() error {
	class, _ := syscall.UTF16PtrFromString("uitoolkit.StatusItem")
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
	wc.WndProc = syscall.NewCallback(w.wndProc)
	wc.ClassName = class
	procRegister.Call(uintptr(unsafe.Pointer(&wc)))
	title, _ := syscall.UTF16PtrFromString(w.title)
	hwnd, _, err := procCreateWindow.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
	if hwnd == 0 {
		return err
	}
	w.hwnd = hwnd
	icon, _, _ := procLoadIcon.Call(0, 32512) // IDI_APPLICATION
	w.nid.Size = uint32(unsafe.Sizeof(w.nid))
	w.nid.Wnd = hwnd
	w.nid.ID = 1
	w.nid.Flags = nifMessage | nifIcon | nifTip
	w.nid.Callback = wmApp + 1
	w.nid.Icon = icon
	utf16Copy(w.nid.Tip[:], w.tooltip)
	r, _, callErr := procNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&w.nid)))
	if r == 0 {
		return callErr
	}
	go w.loop()
	return nil
}

func (w *winStatusItem) loop() {
	var msg struct {
		HWND    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), w.hwnd, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslate.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (w *winStatusItem) wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if msg == wmApp+1 {
		switch lparam {
		case wmLButtonUp:
			w.mu.Lock()
			fn := w.opts.OnClick
			w.mu.Unlock()
			if fn != nil {
				fn()
			}
		case wmRButtonUp:
			w.mu.Lock()
			items := copyMenu(w.menu)
			w.mu.Unlock()
			if len(items) > 0 && items[0].OnClick != nil && !items[0].Separator {
				items[0].OnClick()
			}
		case ninBalloonUserClick:
			w.mu.Lock()
			fn := w.opts.OnNotifyClick
			if fn == nil {
				fn = w.opts.OnClick
			}
			w.mu.Unlock()
			if fn != nil {
				fn()
			}
		}
		return 0
	}
	r, _, _ := procDefWindow.Call(hwnd, msg, wparam, lparam)
	return r
}

func (w *winStatusItem) SetIcon(icon StatusIcon) error {
	w.mu.Lock()
	w.icon = icon
	w.mu.Unlock()
	return w.modify(nifIcon)
}

func (w *winStatusItem) SetTooltip(s string) error {
	w.mu.Lock()
	w.tooltip = s
	utf16Copy(w.nid.Tip[:], s)
	w.mu.Unlock()
	return w.modify(nifTip)
}

func (w *winStatusItem) SetTitle(s string) error {
	w.mu.Lock()
	w.title = s
	w.mu.Unlock()
	return nil
}

func (w *winStatusItem) SetMenu(items []StatusMenuItem) error {
	w.mu.Lock()
	w.menu = copyMenu(items)
	w.mu.Unlock()
	return nil
}

func (w *winStatusItem) Notify(n Notification) error {
	w.mu.Lock()
	utf16Copy(w.nid.InfoTitle[:], n.Title)
	utf16Copy(w.nid.Info[:], n.Body)
	w.nid.Flags |= nifInfo
	w.nid.Timeout = 8000
	if n.OnClick != nil {
		w.opts.OnNotifyClick = n.OnClick
	}
	w.mu.Unlock()
	return w.modify(nifInfo)
}

func (w *winStatusItem) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	hwnd := w.hwnd
	w.mu.Unlock()
	procNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&w.nid)))
	if hwnd != 0 {
		procDestroyWin.Call(hwnd)
		procPostQuit.Call(0)
	}
	return nil
}

func (w *winStatusItem) Backend() string { return "win32" }

func (w *winStatusItem) Alive() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return !w.closed && w.hwnd != 0
}

func (w *winStatusItem) modify(flags uint32) error {
	w.nid.Flags = nifMessage | nifIcon | nifTip | flags
	procNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&w.nid)))
	return nil
}

func utf16Copy(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	n := len(u)
	if n > len(dst) {
		n = len(dst)
		u[n-1] = 0
	}
	copy(dst, u[:n])
}
