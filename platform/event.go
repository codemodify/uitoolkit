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
	KeyA
	KeyC
	KeyV
	KeyX
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
)

// Event is a platform-translated input or window event.
type Event struct {
	Kind   EventKind
	Pos    paintengine2d.Point
	Button MouseButton
	Scroll paintengine2d.Point
	Key    Key
	Rune   rune
	Mods   Modifiers
	Width  int
	Height int
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
}

// Surface is the OS (or offscreen) window seam. The toolkit paints into
// Buffer via paintengine2d.WrapImage / NewImage and calls Present.
type Surface interface {
	Title() string
	SetTitle(title string)
	Size() (w, h int)
	Resize(w, h int) error
	Buffer() *paintengine2d.Image
	Present(dirty []paintengine2d.Rect) error
	Poll() []Event
	Close() error
	Closed() bool
}

// Backend opens surfaces. Linux ships X11 (CGO) plus a always-on offscreen
// backend for tests, screenshots, and headless CI.
type Backend interface {
	Name() string
	NewSurface(opts WindowOptions) (Surface, error)
}
