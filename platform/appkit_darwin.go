//go:build darwin && cgo

package platform

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa -framework QuartzCore -framework CoreGraphics
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// The AppKit backend.
//
// The shape of it follows the Win32 one, because the two window systems
// ask the same thing of a toolkit and the boundary was reshaped for
// both: the platform's callbacks only ever append to a queue, and Poll
// drains it. AppKit calls a window delegate from inside its own
// machinery, exactly as a window procedure is called from inside
// SetWindowPos, so doing anything there but queueing would run the
// toolkit inside itself.
//
// What differs from Windows, and it is the pleasant direction: the
// buffer goes up as it is. paintengine2d keeps premultiplied RGBA and
// CGBitmapContext takes premultiplied RGBA, so there is no swizzle —
// none of the BGRX business the DIB needed.

var (
	akMu sync.Mutex
	// akBySID maps a surface id back to its surface: the delegate is
	// handed an integer, because handing C a Go pointer is not allowed.
	akBySID  = map[uintptr]*akSurface{}
	akNextID uintptr
)

// AppKitBackend is the macOS backend.
type AppKitBackend struct{}

func (AppKitBackend) Name() string { return "appkit" }

// Caps: macOS places windows where it likes unless asked, and an
// application can ask — so both bits are here, as on Windows and unlike
// Wayland.
func (AppKitBackend) Caps() BackendCaps { return BackendDesktop | BackendScreenPlace }

func (AppKitBackend) NewSurface(opts WindowOptions) (Surface, error) { return newAkSurface(opts) }

type akSurface struct {
	id  uintptr
	win unsafe.Pointer // the NSWindow, retained on the C side

	title string

	// secureOn is this window's half of the process-wide secure input
	// count, so closing it can give back exactly what it took.
	secureOn bool

	mu    sync.Mutex
	queue []Event

	img                *paintengine2d.Image
	bufW, bufH         int
	logicalW, logicalH int
	scale              float32

	closed bool
	torn   bool
	// role is what kind of window this is and owner the window it belongs to
	// (-addChildWindow:). There is no task-bar field: the Mac's Dock lists
	// applications and not windows, so there is no per-window entry to be
	// left out of, and FrameSkipTaskbar is absent here because of it.
	role  WindowRole
	owner *akSurface

	// The frame and geometry seams' state (appkit_frame_darwin.go,
	// appkit_geometry_darwin.go).
	decor  Decorations
	frame  Frame
	sizing Sizing
	limits SizeLimits

	// mods is the modifier set the last event carried, kept so that
	// flagsChanged — which says what is held now, not what changed —
	// can be turned into the key going down or coming up.
	mods Modifiers
	// pointerIn tracks whether the pointer is over the content area, so
	// a leave is reported once and not on every move outside it.
	pointerIn bool
	// cursor is the shape this window asked for (cursor_darwin.go).
	cursor Cursor

	// pop is set on a popup and says what it hangs from; kids are the
	// popups open from this surface, innermost last
	// (appkit_popup_darwin.go).
	pop  *akPopup
	kids []*akSurface

	// Drag and drop (appkit_dnd_darwin.go): what the drag over this
	// window offers, what was copied out of the pasteboard at the drop,
	// what the toolkit last agreed to, and whether a drag started here
	// is running.
	dragMimes  []string
	dropData   map[string][]byte
	dragAccept bool
	dragAction DragAction
	dragging   bool

	opts WindowOptions
}

func newAkSurface(opts WindowOptions) (Surface, error) {
	C.uitk_ak_init()

	w, h := max(opts.Width, 1), max(opts.Height, 1)
	s := &akSurface{title: opts.Title, opts: opts, scale: 1, logicalW: w, logicalH: h}

	akMu.Lock()
	akNextID++
	s.id = akNextID
	akBySID[s.id] = s
	akMu.Unlock()

	ctitle := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(ctitle))
	// AppKit's content rectangle is in *points*, which is the toolkit's
	// logical pixel: the scale is applied to the backing store, not to
	// the geometry, so unlike Win32 nothing has to be multiplied here.
	s.win = C.uitk_ak_window_new(C.uintptr_t(s.id), ctitle, C.int(w), C.int(h))
	if s.win == nil {
		akMu.Lock()
		delete(akBySID, s.id)
		akMu.Unlock()
		return nil, fmt.Errorf("appkit: the window could not be made")
	}
	s.scale = float32(C.uitk_ak_backing_scale(s.win))
	if s.scale <= 0 {
		s.scale = 1
	}
	s.resizeBuffer(DevicePixels(w, s.scale), DevicePixels(h, s.scale))
	s.decor = opts.Decorations
	if s.decor == DecorationsAuto {
		s.decor = DecorationsServer
	}
	if s.decor != DecorationsServer {
		C.uitk_ak_set_decorations(s.win, 1)
	}
	s.sizing = opts.Sizing
	s.limits = limitsFor(s.sizing, opts, w, h)
	s.applyLimits()
	// Every window takes drops: the toolkit decides what to do with one
	// by answering AcceptDrag, and a window that had not registered
	// would never be asked.
	C.uitk_ak_register_drops(s.win)
	// Before the window is shown: a dialog that appears with a minimize
	// button and loses it a frame later is a flicker the user sees, and
	// -center on a window already on screen is a jump.
	if opts.Role != RoleNormal {
		s.SetWindowRole(opts.Role)
	}
	if opts.Center {
		s.Center()
	}
	if opts.Owner != nil {
		s.SetOwner(opts.Owner)
	}
	if opts.KeepAbove {
		FrameOf(s).SetKeepAbove(true)
	}
	if !opts.Headless {
		C.uitk_ak_window_show(s.win)
	}
	return s, nil
}

func (s *akSurface) resizeBuffer(w, h int) {
	w, h = max(w, 1), max(h, 1)
	if s.img != nil && s.bufW == w && s.bufH == h {
		return
	}
	s.img = paintengine2d.NewImage(w, h)
	s.bufW, s.bufH = w, h
}

// push queues an event for this surface — or, when the surface is a
// popup and the event is input, for the popup's root window with every
// position moved by the popup's origin.
//
// That is the contract ([PopupSurface]): a popup is placed by the
// window system and paints its own buffer, but as far as the
// application is concerned its pointer and its keys are the window's.
// Everything else — the popup's own resize, its close — stays on the
// popup, because it is the backend's business and would read as the
// root window's if it were passed on.
func (s *akSurface) push(ev Event) {
	if s.pop != nil && s.pop.root != nil && s.pop.root != s && popupInput(ev.Kind) {
		s.pop.root.push(popupEvent(ev, s.Origin()))
		return
	}
	s.mu.Lock()
	s.queue = append(s.queue, ev)
	s.mu.Unlock()
}

func (s *akSurface) Title() string { return s.title }

func (s *akSurface) SetTitle(t string) {
	if s.win == nil || t == s.title {
		return
	}
	s.title = t
	ct := C.CString(t)
	defer C.free(unsafe.Pointer(ct))
	C.uitk_ak_set_title(s.win, ct)
}

func (s *akSurface) Size() (int, int) { return s.bufW, s.bufH }
func (s *akSurface) Scale() float32   { return s.scale }

func (s *akSurface) Buffer() *paintengine2d.Image { return s.img }

// Resize asks for a window w by h logical pixels across.
func (s *akSurface) Resize(w, h int) error {
	if s.win == nil || s.closed {
		return nil
	}
	s.logicalW, s.logicalH = max(w, 1), max(h, 1)
	s.resizeBuffer(DevicePixels(s.logicalW, s.scale), DevicePixels(s.logicalH, s.scale))
	return nil
}

// Present puts the buffer on screen.
//
// The damage list is not used yet: a layer's contents is the whole
// image, so there is nothing to give a rectangle to. Taking it and
// ignoring it is deliberate — the signature is the boundary's, and a
// partial present here would mean keeping a CGImage between frames.
func (s *akSurface) Present(dirty []paintengine2d.Rect) error {
	if s.win == nil || s.closed || s.img == nil {
		return nil
	}
	if len(s.img.Pix) == 0 {
		return fmt.Errorf("appkit: the surface has no pixels to present")
	}
	C.uitk_ak_present(s.win, (*C.uchar)(unsafe.Pointer(&s.img.Pix[0])),
		C.int(s.bufW), C.int(s.bufH))
	return nil
}

func (s *akSurface) Poll() []Event {
	C.uitk_ak_pump()
	s.mu.Lock()
	ev := s.queue
	s.queue = nil
	s.mu.Unlock()
	for i := range ev {
		if ev[i].Kind == EventResize {
			s.logicalW, s.logicalH = ev[i].Width, ev[i].Height
			s.resizeBuffer(DevicePixels(ev[i].Width, s.scale), DevicePixels(ev[i].Height, s.scale))
		}
	}
	return ev
}

func (s *akSurface) Close() error {
	if s == nil || s.torn {
		return nil
	}
	s.torn = true
	// Give back whatever this window took of the process-wide secure
	// input count. An unbalanced enable would leave the desktop in
	// secure input until the process exits, and no other application
	// could undo it.
	if s.secureOn {
		s.SetSecureInput(false)
	}
	// Popups first, and deepest first: a submenu belongs to its menu,
	// and a child window outliving its parent is a window nobody can
	// reach.
	s.closeKids()
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if s.pop != nil {
		s.closePopup()
	} else if s.win != nil {
		C.uitk_ak_window_close(s.win)
		s.win = nil
	}
	akMu.Lock()
	delete(akBySID, s.id)
	akMu.Unlock()
	return nil
}

func (s *akSurface) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// akSurfaceOf is the surface an event belongs to.
func akSurfaceOf(sid C.uintptr_t) *akSurface {
	akMu.Lock()
	defer akMu.Unlock()
	return akBySID[uintptr(sid)]
}

//export uitkAkEvent
func uitkAkEvent(sid C.uintptr_t, kind C.int, a, b C.double) {
	s := akSurfaceOf(sid)
	if s == nil {
		return
	}
	switch kind {
	case C.UITK_AK_CLOSE:
		// A request, not an order: the application may keep the window,
		// which is why windowShouldClose answers NO and says so here.
		s.push(Event{Kind: EventClose})
	case C.UITK_AK_RESIZE:
		s.push(Event{Kind: EventResize, Width: int(a), Height: int(b)})
	case C.UITK_AK_MOVE:
		s.push(Event{Kind: EventMove, Width: int(a), Height: int(b)})
	case C.UITK_AK_EXPOSE:
		s.push(Event{Kind: EventExpose, Width: s.bufW, Height: s.bufH})
	case C.UITK_AK_SCALE:
		if sc := float32(a); sc > 0 && sc != s.scale {
			s.scale = sc
			s.resizeBuffer(DevicePixels(s.logicalW, sc), DevicePixels(s.logicalH, sc))
			s.push(Event{Kind: EventScale})
		}
	case C.UITK_AK_FOCUS_IN:
		s.push(Event{Kind: EventFocusIn})
	case C.UITK_AK_FOCUS_OUT:
		s.push(Event{Kind: EventFocusOut})
	case C.UITK_AK_STATE:
		s.push(Event{Kind: EventWindowState, State: s.WindowState()})
	}
}

//export uitkAkInput
func uitkAkInput(sid C.uintptr_t, kind C.int, x, y, dx, dy C.double,
	button, key C.int, mods C.uint64_t, text *C.char, flags C.int) {
	s := akSurfaceOf(sid)
	if s == nil {
		return
	}
	m := akMods(uint64(mods))
	at := paintengine2d.Pt(float32(x), float32(y))
	switch kind {
	case C.UITK_AK_MOUSE_DOWN:
		s.pointerIn = true
		s.push(Event{Kind: EventMouseDown, Pos: at, Button: MouseButton(button), Mods: m})
	case C.UITK_AK_MOUSE_UP:
		s.push(Event{Kind: EventMouseUp, Pos: at, Button: MouseButton(button), Mods: m})
	case C.UITK_AK_MOUSE_MOVE:
		s.pointerIn = true
		s.push(Event{Kind: EventMouseMove, Pos: at, Mods: m})
	case C.UITK_AK_POINTER_LEAVE:
		if s.pointerIn {
			s.pointerIn = false
			s.push(Event{Kind: EventPointerLeave, Pos: at, Mods: m})
		}
	case C.UITK_AK_SCROLL:
		s.push(Event{
			Kind: EventScroll, Pos: at, Mods: m,
			Scroll:        paintengine2d.Pt(float32(dx), float32(dy)),
			ScrollPrecise: flags != 0,
		})
	case C.UITK_AK_KEY_DOWN:
		if flags == 2 {
			// Text from -insertText: with nothing being composed:
			// ordinary typing, after the layout, the Shift and any dead
			// key have had their say. The filter is the same one every
			// backend uses — a printable rune is >= 32 and not 127 —
			// because -interpretKeyEvents: also turns Return and Escape
			// into text nobody should insert.
			if t := akText(C.GoString(text), uint64(mods)); t != "" {
				s.push(Event{Kind: EventText, Text: t, Rune: firstRune(t), Mods: m})
			}
			return
		}
		if flags != 0 {
			// A modifier changed. AppKit says which are held now, so the
			// one that moved is whichever bit differs, and which way it
			// moved is whether the new set has it.
			s.pushModifierChange(m)
			return
		}
		s.mods = m
		s.push(Event{Kind: EventKeyDown, Key: akKey(rune(key)), Mods: m})
		// The text no longer rides with the key: it comes back from
		// -insertText: through the input context, which is what lets an
		// input method compose (appkit_ime_darwin.go).
	case C.UITK_AK_KEY_UP:
		s.mods = m
		s.push(Event{Kind: EventKeyUp, Key: akKey(rune(key)), Mods: m})
	case C.UITK_AK_IME_PREEDIT:
		s.push(Event{
			Kind: EventIMEPreedit, Text: C.GoString(text),
			IMECaret: int(key), IMEDelBefore: int(button),
		})
	case C.UITK_AK_IME_COMMIT:
		s.push(Event{Kind: EventIMECommit, Text: C.GoString(text)})
	case C.UITK_AK_IME_CANCEL:
		s.push(Event{Kind: EventIMECancel})
	case C.UITK_AK_DRAG_MOTION, C.UITK_AK_DRAG_LEAVE, C.UITK_AK_DROP, C.UITK_AK_DRAG_END:
		// button carries what the source allows and key what it prefers
		// — or, at the end, the action that was performed.
		s.akDragEvent(kind, at, DragAction(button), DragAction(key))
	}
}

// pushModifierChange turns AppKit's "these are held now" into the key
// events the toolkit expects. Only Alt has a [Key] of its own — a
// window underlines its mnemonics while it is held — so the others
// change s.mods and say nothing.
func (s *akSurface) pushModifierChange(now Modifiers) {
	was := s.mods
	s.mods = now
	if (was^now)&ModAlt == 0 {
		return
	}
	k := EventKeyUp
	if now&ModAlt != 0 {
		k = EventKeyDown
	}
	s.push(Event{Kind: k, Key: KeyAlt, Mods: now})
}

// firstRune is the first rune of t, for the Rune field an EventText
// carries beside its Text.
func firstRune(t string) rune {
	for _, r := range t {
		return r
	}
	return 0
}

var _ Surface = (*akSurface)(nil)
var _ Backend = AppKitBackend{}

// SendsMoveEvents: AppKit posts NSWindowDidMove, so this backend reports
// a move ([MoveEventSurface]).
func (s *akSurface) SendsMoveEvents() bool { return true }
