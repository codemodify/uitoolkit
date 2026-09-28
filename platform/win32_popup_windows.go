//go:build windows

package platform

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// Popups on Win32: menus, combo lists and tooltips as windows of their
// own rather than rectangles painted inside the window.
//
// Without this the toolkit falls back to drawing a popup inside its
// window, which works until the popup is taller than the room under it
// — a combo box near the foot of a window, a menu on a short window —
// and then it is clipped or moved somewhere it does not belong.
//
// **The placement is the toolkit's.** Wayland hands the whole problem
// to the compositor (xdg_positioner); Win32 has no such thing for an
// application's own popup, so this takes the X11 backend's path and
// calls [SolvePopup] against the monitor's work area. The answer is
// therefore the same one a headless test computes, with the same
// function under both.
//
// **A popup never activates.** WS_EX_NOACTIVATE and SW_SHOWNOACTIVATE
// keep the focus on the window the menu belongs to, which is what the
// contract needs anyway: a popup hands every input event to its root,
// so the keyboard staying where it is means the keys arrive where they
// were going. The popup's own window procedure sees the pointer, and
// that is translated by [Origin] and queued on the root.

const (
	wsExNoActivate    = 0x08000000
	wsExToolWindow    = 0x00000080
	wsExTopmost       = 0x00000008
	csDropShadow      = 0x00020000
	swShowNoActivate  = 4
	swpNoActivateFlag = 0x0010
)

var (
	win32PopClass  sync.Once
	win32PopClassA *uint16
	win32PopErr    error
)

// win32RegisterPopupClass registers the popup window class.
//
// It is a class of its own rather than the window class with different
// styles, for CS_DROPSHADOW: that is what gives a menu the small shadow
// every Windows menu has, it is a *class* style and not a window one,
// and a top-level window must not have it — a window's shadow is DWM's
// and it would get two.
func win32RegisterPopupClass() (*uint16, error) {
	win32PopClass.Do(func() {
		name, err := syscall.UTF16PtrFromString("uitoolkit.Popup")
		if err != nil {
			win32PopErr = err
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
		wc.Style = 0x0003 | csDropShadow // CS_HREDRAW | CS_VREDRAW | CS_DROPSHADOW
		wc.WndProc = syscall.NewCallback(win32Proc)
		wc.Instance = inst
		wc.ClassName = name
		if r, _, err := procRegisterClass.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			win32PopErr = fmt.Errorf("win32: RegisterClassExW (popup): %w", err)
			return
		}
		win32PopClassA = name
	})
	return win32PopClassA, win32PopErr
}

// winPopup is what makes a [winSurface] a popup: a surface is a surface
// either way, so this is a field on the one there is rather than a
// second type — the shape the offscreen and AppKit backends use too.
type winPopup struct {
	root   *winSurface
	parent *winSurface
	opts   PopupOptions
	placed FrameRect
}

var errWinNoPopups = errors.New("win32: this window cannot open popups")

// popRoot is the top-level window a surface belongs to.
func (s *winSurface) popRoot() *winSurface {
	for s != nil && s.pop != nil && s.pop.root != nil && s.pop.root != s {
		s = s.pop.root
	}
	return s
}

// PopupsSupported reports whether this window can open popups
// ([PopupOpener]).
func (s *winSurface) PopupsSupported() bool {
	return s != nil && s.hwnd != 0 && !s.closed
}

// OpenPopup opens a popup from s ([PopupOpener]).
func (s *winSurface) OpenPopup(opts PopupOptions) (PopupSurface, error) {
	if !s.PopupsSupported() {
		return nil, errWinNoPopups
	}
	class, err := win32RegisterPopupClass()
	if err != nil {
		return nil, err
	}
	root := s.popRoot()
	p := &winSurface{
		scale:    root.scale,
		logicalW: max(opts.Placement.W, 1),
		logicalH: max(opts.Placement.H, 1),
		opts:     WindowOptions{Width: max(opts.Placement.W, 1), Height: max(opts.Placement.H, 1), Popup: true},
		deco:     DecorationsNone,
		frame:    opts.Frame,
		pop:      &winPopup{root: root, parent: s, opts: opts},
	}
	p.pop.placed = SolvePopup(opts.Placement, s.workArea())
	x, y := root.screenOf(p.pop.placed)
	w, h := DevicePixels(p.pop.placed.W, p.scale), DevicePixels(p.pop.placed.H, p.scale)
	p.resizeBuffer(w, h)

	ex := uintptr(wsExNoActivate | wsExToolWindow)
	if opts.Role == PopupRoleTooltip {
		// A tooltip sits above even another application's windows for
		// the moment it is up; a menu rides its owner's z-order.
		ex |= wsExTopmost
	}
	inst, _, _ := procGetModuleHandle.Call(0)
	hwnd, _, callErr := procCreateWindowEx.Call(ex,
		uintptr(unsafe.Pointer(class)), 0,
		wsPopup,
		uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(w)), uintptr(int32(h)),
		// Owned by the root, not a child of it: an owned window floats
		// above its owner, is hidden when the owner is minimized and is
		// destroyed with it, which is a menu's whole lifetime.
		root.hwnd, 0, inst, 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("win32: CreateWindowExW (popup): %w", callErr)
	}
	p.hwnd = hwnd

	win32Mu.Lock()
	win32ByHWND[hwnd] = p
	win32Mu.Unlock()

	procShowWindow.Call(hwnd, swShowNoActivate)
	p.visible = true
	s.kids = append(s.kids, p)
	root.push(Event{Kind: EventPopupPlaced, Popup: p})
	return p, nil
}

// Reposition places the popup afresh ([PopupSurface]).
func (s *winSurface) Reposition(pl PopupPlacement) bool {
	if s == nil || s.pop == nil || s.hwnd == 0 || s.closed {
		return false
	}
	s.pop.opts.Placement = pl
	s.pop.placed = SolvePopup(pl, s.pop.parent.workArea())
	x, y := s.pop.root.screenOf(s.pop.placed)
	w, h := DevicePixels(s.pop.placed.W, s.scale), DevicePixels(s.pop.placed.H, s.scale)
	procSetWindowPos.Call(s.hwnd, 0,
		uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(w)), uintptr(int32(h)),
		swpNoZOrder|swpNoActivateFlag)
	if s.bufW != w || s.bufH != h {
		s.logicalW, s.logicalH = s.pop.placed.W, s.pop.placed.H
		s.resizeBuffer(w, h)
		s.push(Event{Kind: EventResize, Width: s.logicalW, Height: s.logicalH})
	}
	return true
}

// Placed is where the popup's visible box ended up ([PopupSurface]).
func (s *winSurface) Placed() FrameRect {
	if s == nil || s.pop == nil {
		return FrameRect{}
	}
	return s.pop.placed
}

// Origin is where the popup's buffer (0,0) lands in the root window's
// buffer ([PopupSurface]).
//
// The root's margin is zero here — Windows draws a top-level window's
// shadow itself (FrameSystemShadow) and the toolkit reserves nothing
// for one — and so is the popup's, because CS_DROPSHADOW puts the
// menu's shadow outside the window too. It goes through [PopupOrigin]
// rather than being written out because the rounding is the part that
// matters, and it is the same rounding on every backend.
func (s *winSurface) Origin() paintengine2d.Point {
	if s == nil || s.pop == nil {
		return paintengine2d.Point{}
	}
	root := s.pop.root
	m := root.frame.Margin
	win := paintengine2d.Pt(float32(m.Left), float32(m.Top))
	return PopupOrigin(win, s.pop.placed, s.frame.Margin, root.scale)
}

// Root is the top-level window the popup belongs to ([PopupSurface]).
func (s *winSurface) Root() Surface {
	if s == nil {
		return nil
	}
	return s.popRoot()
}

// PopupWorkArea is where this window's popups may go
// ([PopupWorkArea]): the monitor it is on, less the task bar, in
// logical pixels relative to the window's client area.
//
// Windows can answer this and a Wayland client cannot, which is why the
// capability is optional at all.
func (s *winSurface) PopupWorkArea() (FrameRect, bool) {
	if s == nil || s.hwnd == 0 || s.closed {
		return FrameRect{}, false
	}
	return s.workArea(), true
}

// workArea is the monitor's usable area in this window's own
// coordinates and logical pixels.
func (s *winSurface) workArea() FrameRect {
	mon, _, _ := procMonitorFromWindow.Call(s.hwnd, monitorNearest)
	if mon == 0 {
		return FrameRect{}
	}
	var mi winMonitorInfo
	mi.Size = uint32(unsafe.Sizeof(mi))
	if r, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return FrameRect{}
	}
	ox, oy := s.clientOrigin()
	sc := s.deviceScale()
	// Device pixels on screen to logical pixels relative to the client
	// area: the placement PopupPlacement speaks.
	x0 := ceilDiv(int(mi.Work.Left)-ox, sc)
	y0 := ceilDiv(int(mi.Work.Top)-oy, sc)
	x1 := floorDiv(int(mi.Work.Right)-ox, sc)
	y1 := floorDiv(int(mi.Work.Bottom)-oy, sc)
	return FrameRect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

// clientOrigin is where this window's client area begins on screen, in
// device pixels — the origin a popup's placement is relative to.
func (s *winSurface) clientOrigin() (int, int) {
	var p winPoint
	procClientToScreen.Call(s.hwnd, uintptr(unsafe.Pointer(&p)))
	return int(p.X), int(p.Y)
}

// screenOf turns a placement — logical pixels relative to this window's
// client area — into device pixels on screen.
func (s *winSurface) screenOf(r FrameRect) (int, int) {
	ox, oy := s.clientOrigin()
	sc := s.deviceScale()
	return ox + DevicePosition(r.X, sc), oy + DevicePosition(r.Y, sc)
}

// closePopup destroys the popup's window and unhooks it from its
// parent.
func (s *winSurface) closePopup() {
	if s == nil || s.pop == nil {
		return
	}
	if s.hwnd != 0 {
		win32Mu.Lock()
		delete(win32ByHWND, s.hwnd)
		win32Mu.Unlock()
		procDestroyWindow.Call(s.hwnd)
		s.hwnd = 0
	}
	if p := s.pop.parent; p != nil {
		for i, k := range p.kids {
			if k == s {
				p.kids = append(p.kids[:i], p.kids[i+1:]...)
				break
			}
		}
	}
	if root := s.pop.root; root != nil {
		root.push(Event{Kind: EventPopupDone, Popup: s})
	}
}

// closeKids takes down every popup open from s, deepest first.
func (s *winSurface) closeKids() {
	for len(s.kids) > 0 {
		s.kids[len(s.kids)-1].Close()
	}
}

var (
	_ PopupSurface  = (*winSurface)(nil)
	_ PopupOpener   = (*winSurface)(nil)
	_ PopupWorkArea = (*winSurface)(nil)
)
