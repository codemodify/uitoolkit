package platform

import "github.com/codemodify/paintengine2d"

// MouseButton is a pointer button. Wheel buttons are reported as Scroll on Event.
type MouseButton int

const (
	ButtonNone MouseButton = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
)

// Key is a physical / logical key for focus and shortcuts.
type Key int

const (
	KeyUnknown Key = iota
	KeyEscape
	KeyTab
	KeyReturn
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeySpace
	Key3
	KeyHash
	KeyComma
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	// KeyMenu is the context-menu key (between AltGr and Ctrl; Shift+F10
	// does the same).
	KeyMenu
)

// Modifiers is a bitset of active modifier keys.
type Modifiers int

const (
	ModShift Modifiers = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
)

func (m Modifiers) Shift() bool { return m&ModShift != 0 }
func (m Modifiers) Ctrl() bool  { return m&ModCtrl != 0 }
func (m Modifiers) Alt() bool   { return m&ModAlt != 0 }

// LetterKey maps a/A..z/Z to KeyA..KeyZ.
func LetterKey(r rune) Key {
	if r >= 'A' && r <= 'Z' {
		r = r - 'A' + 'a'
	}
	if r >= 'a' && r <= 'z' {
		return KeyA + Key(r-'a')
	}
	return KeyUnknown
}

// KeyRune returns the lowercase letter for KeyA..KeyZ.
func KeyRune(k Key) (rune, bool) {
	if k >= KeyA && k <= KeyZ {
		return rune('a' + (k - KeyA)), true
	}
	return 0, false
}

// EventKind classifies a window event.
type EventKind int

const (
	EventNone EventKind = iota
	EventMouseDown
	EventMouseUp
	EventMouseMove
	EventScroll
	EventKeyDown
	EventKeyUp
	EventText
	EventResize
	EventClose
	EventExpose
	EventFocusIn
	EventFocusOut
	EventIMEPreedit
	EventIMECommit
	EventIMECancel
	// EventPointerLeave: the pointer left the window (wl_pointer.leave,
	// X11 LeaveNotify). Hover and pending tooltips must not survive it.
	EventPointerLeave
)

// Event is a platform-translated input or window event.
type Event struct {
	Kind         EventKind
	Pos          paintengine2d.Point
	Button       MouseButton
	Scroll       paintengine2d.Point
	Key          Key
	Rune         rune
	Mods         Modifiers
	Width        int
	Height       int
	Text         string // IME preedit or commit (UTF-8)
	IMECaret     int    // caret within EventIMEPreedit text
	IMEDelBefore int    // text-input-v3 delete_surrounding bytes before caret
	IMEDelAfter  int    // text-input-v3 delete_surrounding bytes after caret
}

// WindowOptions configure a native or offscreen surface.
type WindowOptions struct {
	Title           string
	Width           int
	Height          int
	MinWidth        int
	MinHeight       int
	Resizable       bool
	Headless        bool
	BackgroundPixel uint32
	// X, Y are root/screen coordinates. Used when Popup is set (X11
	// override-redirect / _NET_WM_WINDOW_TYPE_POPUP_MENU). Wayland
	// toplevels cannot be placed by the client.
	X, Y int
	// Popup requests a short-lived menu surface: no taskbar, no
	// decorations when the backend can, positioned at X,Y on X11.
	Popup bool
}

// Surface is the OS (or offscreen) window seam. The toolkit paints with
// NewPaintContext (GPUDevice when EGL is bound, else Buffer) and calls Present.
//
// Threading: every Surface method, and the package-level clipboard and
// cursor helpers, must be called from the one goroutine that runs the
// event loop. The backends talk to Xlib / libwayland connections that are
// not goroutine-safe in the way this package uses them.
//
// Close semantics: [EventClose] is a *request* from the window manager or
// compositor (WM_DELETE_WINDOW, xdg_toplevel.close). The application may
// ignore it (close-to-tray) and keep using the surface. Closed only
// reports true once Close has been called or the window died for real
// (X11 DestroyNotify / a lost connection), so a surface that is still
// usable is never reaped by a run loop that polls Closed.
type Surface interface {
	Title() string
	SetTitle(title string)
	Size() (w, h int)
	Resize(w, h int) error
	Buffer() *paintengine2d.Image
	Present(dirty []paintengine2d.Rect) error
	// Poll returns the events queued since the last call. It keeps
	// returning already-queued events after the surface is closed so a
	// final EventClose cannot be lost.
	Poll() []Event
	Close() error
	// Closed reports that the surface is gone for good: Close was
	// called, or the window was destroyed by the server. Receiving
	// EventClose does not by itself make this true.
	Closed() bool
	Scale() float32
}

// Backend opens surfaces. Linux ships X11 (CGO) plus a always-on offscreen
// backend for tests, screenshots, and headless CI.
type Backend interface {
	Name() string
	NewSurface(opts WindowOptions) (Surface, error)
}
