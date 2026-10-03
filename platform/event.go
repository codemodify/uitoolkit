package platform

import "github.com/codemodify/paintengine2d"

// MouseButton is a pointer button. Wheel buttons are reported as Scroll on Event.
type MouseButton int

const (
	ButtonNone MouseButton = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
	// ButtonBack and ButtonForward are a mouse's thumb buttons (X11
	// buttons 8 and 9, evdev BTN_SIDE and BTN_EXTRA): a browser's and a
	// file manager's back and forward.
	ButtonBack
	ButtonForward
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
	// KeyAlt is either Alt key on its own: windows show mnemonic
	// underlines while it is held in looks that hide them otherwise.
	KeyAlt
)

// Modifiers is a bitset of active modifier keys.
type Modifiers int

const (
	ModShift Modifiers = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
	// ModCapsLock and ModNumLock are the *lock* keys, which are latched
	// state rather than a key being held. They are on every key event
	// while the lock is on.
	//
	// Caps Lock is here because a passphrase field cannot tell the user
	// why the right passphrase was refused without it, and the only
	// other way to know would be to look at the case of what was typed
	// — which means reading the secret to do it.
	//
	// They are deliberately **not** part of a shortcut's modifiers: a
	// Ctrl+S is a Ctrl+S with Caps Lock on, and an accelerator table
	// that compared the whole set would stop matching. Everything that
	// dispatches shortcuts masks them out; [Modifiers.Chord] is that
	// mask.
	ModCapsLock
	ModNumLock
)

func (m Modifiers) Shift() bool { return m&ModShift != 0 }
func (m Modifiers) Ctrl() bool  { return m&ModCtrl != 0 }
func (m Modifiers) Alt() bool   { return m&ModAlt != 0 }

// CapsLock and NumLock report the lock keys' state at this event.
func (m Modifiers) CapsLock() bool { return m&ModCapsLock != 0 }
func (m Modifiers) NumLock() bool  { return m&ModNumLock != 0 }

// Chord is m without the lock keys: what a shortcut is compared by.
//
// Ctrl+S is Ctrl+S whether or not Caps Lock is on, so every comparison
// of a key event against an accelerator goes through this. Comparing the
// raw set would have made every shortcut in the application stop working
// the moment a user pressed Caps Lock, which is the kind of bug that
// gets reported as "the menus broke".
func (m Modifiers) Chord() Modifiers { return m &^ (ModCapsLock | ModNumLock) }

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

// KeyChar is the printable character k stands for, or 0 for a key that
// stands for none.
//
// It is the key's *identity* as a character and not what typing it
// produces: no modifier is applied, so Shift+A and A are both 'a' and
// Ctrl+S is 's' — the shift and the control are in the event's Modifiers,
// where a shortcut table wants them. The keys that navigate rather than
// type (Tab, Return, Backspace, the arrows, the function keys) have no
// character at all, so a table written over characters cannot fire on one
// of them by accident.
//
// Which layout a key belongs to has already been settled: Key comes from
// the layout's own keysym, falling back to the layout-independent one, so
// the key labelled A on AZERTY is KeyA and its character is 'a'. What the
// user actually typed — the layout, Shift, dead keys and the compose key
// all applied — arrives as text (Component.TextInput), never here.
func KeyChar(k Key) rune {
	if r, ok := KeyRune(k); ok {
		return r
	}
	switch k {
	case KeySpace:
		return ' '
	case Key3:
		return '3'
	case KeyHash:
		return '#'
	case KeyComma:
		return ','
	}
	return 0
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
	// EventResize: the window has a new size, in the logical pixels
	// WindowOptions and Resize speak (Width, Height). The buffer behind
	// Surface.Size is already that size times the display scale.
	EventResize
	// EventMove: the window is at a new position on the desktop, in the
	// logical pixels [Surface.Position] and Move speak (Width and Height
	// carry it, as EventResize carries the size).
	//
	// It is sent when the *position* changed, whether or not the size did,
	// and it is how an application that keeps other windows beside this
	// one hears about a drag. Without it a rack had to poll, which is a
	// trade between a timer fast enough to look attached and one slow
	// enough not to cost anything.
	//
	// Not every desktop can send it. A Wayland toplevel has no position:
	// a client is never told where its windows are, so the Wayland
	// backend sends this for no window, and [MoveEventSurface] is how to
	// ask in advance rather than waiting for an event that will not come.
	EventMove
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
	// EventDragMotion: something dragged from another app is over the
	// window at Pos (Mimes are what it offers, Actions what may be done
	// with it); EventDragLeave: it left; EventDrop: it was dropped at
	// Pos. The data is read with the surface's DropReceiver.
	EventDragMotion
	EventDragLeave
	EventDrop
	// EventDragEnd: a drag this window started (DragSurface.StartDrag)
	// ended. Action is what the target did with it — DragNone when
	// nothing took it, so a cancelled drag and a refused one look the
	// same to the source, as they should.
	EventDragEnd
	// EventWindowState: the desktop changed the window's state (State):
	// maximized, full screen, activated, tiled, suspended.
	EventWindowState
	// EventDecorations: the decoration mode in effect changed (Decor) — the
	// compositor answered a request, or decided on its own (KWin draws no
	// frame at all for a full-screen window).
	EventDecorations
	// EventCapabilities: what the desktop can do for the window changed
	// (Caps).
	EventCapabilities
	// EventPopupPlaced: the window system put one of the window's popups
	// (Popup) somewhere other than where it was last told — it flipped or
	// slid it against the screen's edge, or shrank it — and its new place
	// is [PopupSurface.Placed].
	EventPopupPlaced
	// EventPopupDone: the window system took one of the window's popups
	// (Popup) down on its own — a click outside the application, another
	// window activated (xdg_popup.popup_done, a broken X11 grab). The app
	// dismisses it; the surface is still open until it does.
	EventPopupDone
	// EventGesture: a touchpad gesture (Gesture) at Phase, with the
	// pointer at Pos. See [GestureKind] and [GesturePhase] for the
	// contract; Fingers, Delta, Scale and Rotation carry the gesture.
	EventGesture
	// EventScale: the display scale this window is drawn at changed —
	// it was moved to a monitor with another scale, or the scale of the
	// monitor it is on was changed under it. Read the new one from
	// [Surface.Scale]; the event carries no size, because the window's
	// size in logical pixels has not changed and its buffer is the
	// backend's to resize.
	//
	// It exists because **a scale change is not always a resize**. It
	// usually is on Wayland (a fractional-scale change arrives with a
	// configure) and on Windows (WM_DPICHANGED hands you a rect to
	// honour, and a WM_SIZE follows). On macOS it is not:
	// NSWindowDidChangeBackingProperties fires with the window the same
	// size in points, so a window dragged from a Retina display to a
	// plain one would never be told — it would go on drawing a look
	// built for the wrong scale until something else resized it. Any
	// backend that can notice must send this, resize or no resize.
	//
	// It is safe to send when nothing changed: the application compares
	// against the scale it is drawing at and ignores a repeat.
	//
	// The X11 backend does not send it, and that is not an omission:
	// X11's scale here is the display's, read from Xft.dpi at connect,
	// and one logical screen has one of them — a window moved to
	// another monitor does not change scale because X11 does not think
	// it has. Watching XSETTINGS for Xft/DPI changing under a running
	// session would make it send one; nothing has needed that yet.
	EventScale
)

// DropReceiver is implemented by surfaces that take drops from other
// apps: ReceiveDrop reads the dropped data as mime, and FinishDrop
// completes (or, with ok false, refuses) the drop.
type DropReceiver interface {
	ReceiveDrop(mime string) ([]byte, bool)
	FinishDrop(ok bool)
}

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
	// Mimes are the types a drag offers (EventDragMotion, EventDrop).
	Mimes []string
	// Actions are what a drag's source allows (EventDragMotion,
	// EventDrop); Action is the one it prefers there, and on
	// EventDragEnd the one the target performed.
	Actions DragAction
	Action  DragAction
	// Dropped (EventDragEnd) says the user let the pointer go rather than
	// calling the drag off: the drop happened, whether or not anything
	// took it. A drag that carries a window needs the difference — a
	// window dropped on the desktop stays where it was let go, a
	// cancelled one goes away again (wl_data_source.dnd_drop_performed
	// against cancelled, XdndDrop against a lost grab or Escape).
	Dropped bool
	// ScrollPrecise: Scroll is in device pixels from a touchpad or other
	// continuous source; otherwise it counts wheel notches.
	ScrollPrecise bool
	// State is the window's new state (EventWindowState), Decor the new
	// decoration mode (EventDecorations), Caps what the window system will
	// now do for the window's frame (EventCapabilities).
	State WindowState
	Decor Decorations
	Caps  FrameCaps
	// Gesture and Phase say which touchpad gesture an EventGesture is
	// and where in it; Fingers is how many fingers make it. Delta is how
	// far the fingers (a pinch: the point between them) moved since the
	// last event, in device pixels; Scale is a pinch's spread relative to
	// its begin (1 at the begin, 2 when the fingers are twice as far
	// apart); Rotation is the degrees a pinch turned since the last
	// event, clockwise.
	Gesture  GestureKind
	Phase    GesturePhase
	Fingers  int
	Delta    paintengine2d.Point
	Scale    float32
	Rotation float32
	// Popup is the popup an EventPopupPlaced or EventPopupDone is about.
	// Every other event from a popup arrives on its root window with Pos
	// already in that window's coordinates, as if the popup were part of
	// it (platform.PopupSurface).
	Popup Surface
}

// WindowOptions configure a native or offscreen surface.
//
// Every size here is in **logical pixels** — see scale.go: a window asked
// for as 275 by 116 is that at any display scale, and the backend converts
// where the window system wants device pixels instead.
type WindowOptions struct {
	Title     string
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
	// MaxWidth, MaxHeight cap the window; zero is no cap.
	MaxWidth  int
	MaxHeight int
	// Sizing says whether the desktop may resize the window at all (see
	// [Sizing]). The zero value is resizable; SizingFixed pins the window
	// to the size it opens at, and Min / Max are then that size.
	Sizing Sizing
	// Scale is the display scale the toolkit draws at, where the
	// application has one to state: an explicit app scale, or UITK_SCALE
	// and friends. A backend whose window geometry is device pixels
	// converts the sizes above with it, so the window and the metrics
	// drawn in it agree. Zero — the usual case — means the display's own
	// scale, which the backend detects for itself.
	Scale           float32
	Headless        bool
	BackgroundPixel uint32
	// X, Y are where on the desktop the window goes, in logical pixels
	// like the size (X11 converts to root device pixels); zero for both
	// leaves it to the desktop. A popup (X11 override-redirect /
	// _NET_WM_WINDOW_TYPE_POPUP_MENU) is always put there. Wayland
	// toplevels cannot be placed by the client.
	X, Y int
	// Place says what X, Y mean (see [Placement]). The zero value is a
	// hint. PlaceAtScreen means the window belongs at that point and
	// nowhere else — a tray menu at the pointer — and the Wayland backend
	// then gives it a zwlr_layer_shell_v1 surface instead of an
	// xdg_toplevel, which is the only Wayland role a client can place.
	// Ask [ScreenPlacementAvailable] before relying on it.
	Place Placement
	// Role is what kind of window this is, for the desktop: ordinary (the
	// zero value) or a dialog. See [WindowRole]; it can be changed
	// afterwards through [RoleSurface].
	Role WindowRole
	// Center opens the window in the middle of the monitor it lands on,
	// rather than wherever the desktop would have cascaded it. It is for
	// a prompt: a passphrase dialog that opens in a corner is one the
	// user has to look for.
	//
	// Wayland ignores it, and correctly: a client cannot place a
	// toplevel, and a compositor told the window is a dialog
	// ([RoleDialog]) centres it itself. Ask [CenterAvailable] to know
	// which you have.
	Center bool
	// KeepAbove asks for the window to open above the others, rather
	// than being raised afterwards. The difference matters for a prompt:
	// a window that is raised a frame after it is mapped can be covered
	// in that frame, and on X11 it can be covered by the thing that
	// launched it.
	//
	// It is the same request as [WindowFrame.SetKeepAbove] and has the
	// same answer — no Wayland compositor allows it.
	KeepAbove bool
	// FitContent makes the window as tall as its content the first time
	// it is laid out, keeping Width. A dialog whose height depends on
	// its text — a description of one line or of five — does not know
	// its height when WindowOptions is written, and measuring the
	// content by hand first means guessing at two things the window
	// knows and the caller does not: the display scale this window will
	// open at, and the height of a caption the toolkit may draw.
	//
	// [app.Window.FitToContent] is the same thing afterwards.
	FitContent bool
	// Popup requests a short-lived menu surface: no taskbar, no
	// decorations when the backend can, positioned at X,Y on X11.
	Popup bool
	// Decorations says who draws the window's frame (see [Decorations]).
	// The app package resolves Auto before the surface is made; a backend
	// handed Auto uses the desktop's frame.
	Decorations Decorations
}

// Surface is the OS (or offscreen) window seam. The toolkit paints with
// NewPaintContext (GPUDevice when EGL is bound, else Buffer) and calls Present.
//
// Threading: every Surface method, and the package-level clipboard and
// cursor helpers, must be called from **the main OS thread**, the one
// that runs the event loop, with [runtime.LockOSThread] held — which the
// app package does on the application's behalf.
//
// One goroutine is not enough, and the difference is not pedantic. Go may
// move a goroutine between OS threads at any function call, and:
//
//   - Xlib and libwayland connections are not goroutine-safe in the way
//     this package uses them, which is what "one goroutine" was for;
//   - AppKit requires that NSApplication and every NSWindow be touched
//     from the process's *first* thread, and nothing else will do;
//   - Win32 delivers messages to the thread that created the HWND, and
//     OLE drag-and-drop needs that thread to be an STA.
//
// Two of those three are platforms this package does not have yet, which
// is why the weaker rule held so far. It will not survive the first one
// of them, so it is written down now, while there is nothing to break.
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
	// Size is the buffer, in device pixels: what Buffer() is that many
	// pixels across, and what Present's rectangles are measured in.
	Size() (w, h int)
	// Resize asks for a window this many **logical** pixels across (the
	// unit WindowOptions states, and the one EventResize reports back),
	// whatever the display scale. The buffer that follows is larger by
	// the scale, and by the margin a frame the toolkit draws keeps for
	// its shadow.
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
	// Name says which backend this is: "wayland", "x11", "offscreen".
	// It is for diagnostics and for what a page prints — nothing
	// branches on it, because a name is not a capability (see
	// [BackendCaps]).
	Name() string
	// Caps is what this backend will do, asked afresh: one of them
	// depends on the compositor and can change under a running
	// application.
	Caps() BackendCaps
	NewSurface(opts WindowOptions) (Surface, error)
}
