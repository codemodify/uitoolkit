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
	akBySID = map[uintptr]*akSurface{}
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

	mu    sync.Mutex
	queue []Event

	img                *paintengine2d.Image
	bufW, bufH         int
	logicalW, logicalH int
	scale              float32

	closed bool
	torn   bool

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

func (s *akSurface) push(ev Event) {
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
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if s.win != nil {
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
	}
}

var _ Surface = (*akSurface)(nil)
var _ Backend = AppKitBackend{}
