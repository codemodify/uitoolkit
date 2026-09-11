//go:build linux && cgo

package platform

/*
#cgo linux LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/keysym.h>
#include <stdlib.h>
#include <string.h>

static Display* ui_open(void) {
	return XOpenDisplay(NULL);
}

static int ui_pending(Display* d) { return XPending(d); }

static void ui_next(Display* d, XEvent* e) { XNextEvent(d, e); }

static unsigned long ui_black(Display* d) {
	return BlackPixel(d, DefaultScreen(d));
}

static Window ui_create(Display* d, int w, int h, const char* title) {
	int s = DefaultScreen(d);
	Window win = XCreateSimpleWindow(d, RootWindow(d, s), 40, 40, (unsigned)w, (unsigned)h, 0,
		BlackPixel(d, s), BlackPixel(d, s));
	XStoreName(d, win, title);
	XSelectInput(d, win, ExposureMask|KeyPressMask|KeyReleaseMask|ButtonPressMask|ButtonReleaseMask|
		PointerMotionMask|StructureNotifyMask|FocusChangeMask);
	Atom proto = XInternAtom(d, "WM_DELETE_WINDOW", False);
	XSetWMProtocols(d, win, &proto, 1);
	XSizeHints hints;
	memset(&hints, 0, sizeof(hints));
	hints.flags = PMinSize;
	hints.min_width = 200;
	hints.min_height = 120;
	XSetWMNormalHints(d, win, &hints);
	XMapWindow(d, win);
	XFlush(d);
	return win;
}

static void ui_resize_hints(Display* d, Window w, int minw, int minh) {
	XSizeHints hints;
	memset(&hints, 0, sizeof(hints));
	hints.flags = PMinSize;
	hints.min_width = minw;
	hints.min_height = minh;
	XSetWMNormalHints(d, w, &hints);
}

static GC ui_gc(Display* d, Window w) {
	return XCreateGC(d, w, 0, NULL);
}

static void ui_put(Display* d, Window w, GC gc, XImage* img, int sx, int sy, int dx, int dy, unsigned pw, unsigned ph) {
	XPutImage(d, w, gc, img, sx, sy, dx, dy, pw, ph);
}

static void ui_flush(Display* d) { XFlush(d); }

static void ui_destroy_image(XImage* img) {
	if (img) {
		img->data = NULL;
		XDestroyImage(img);
	}
}

static XImage* ui_image(Display* d, int w, int h, char* data) {
	Visual* v = DefaultVisual(d, DefaultScreen(d));
	int depth = DefaultDepth(d, DefaultScreen(d));
	return XCreateImage(d, v, (unsigned)depth, ZPixmap, 0, data, (unsigned)w, (unsigned)h, 32, 0);
}

static int ui_event_type(XEvent* e) { return e->type; }

static int ui_expose_x(XEvent* e) { return e->xexpose.x; }
static int ui_expose_y(XEvent* e) { return e->xexpose.y; }
static int ui_expose_w(XEvent* e) { return e->xexpose.width; }
static int ui_expose_h(XEvent* e) { return e->xexpose.height; }

static int ui_cfg_w(XEvent* e) { return e->xconfigure.width; }
static int ui_cfg_h(XEvent* e) { return e->xconfigure.height; }

static int ui_btn(XEvent* e) { return e->xbutton.button; }
static int ui_btn_x(XEvent* e) { return e->xbutton.x; }
static int ui_btn_y(XEvent* e) { return e->xbutton.y; }
static unsigned ui_btn_state(XEvent* e) { return e->xbutton.state; }

static int ui_mot_x(XEvent* e) { return e->xmotion.x; }
static int ui_mot_y(XEvent* e) { return e->xmotion.y; }
static unsigned ui_mot_state(XEvent* e) { return e->xmotion.state; }

static unsigned ui_key_state(XEvent* e) { return e->xkey.state; }

static KeySym ui_lookup(Display* d, XEvent* e, char* buf, int n, int* len) {
	KeySym ks = 0;
	*len = XLookupString(&e->xkey, buf, n, &ks, NULL);
	return ks;
}

static Atom ui_delete_atom(Display* d) {
	return XInternAtom(d, "WM_DELETE_WINDOW", False);
}

static int ui_is_delete(Display* d, XEvent* e) {
	if (e->type != ClientMessage) return 0;
	Atom del = XInternAtom(d, "WM_DELETE_WINDOW", False);
	return (Atom)e->xclient.data.l[0] == del;
}

static void ui_close(Display* d, Window w, GC gc) {
	if (gc) XFreeGC(d, gc);
	if (w) XDestroyWindow(d, w);
	if (d) XCloseDisplay(d);
}
*/
import "C"

import (
	"fmt"
	"unicode/utf8"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// X11Backend opens Xlib windows and presents premul RGBA via XPutImage.
type X11Backend struct{}

func (X11Backend) Name() string { return "x11" }

func (X11Backend) NewSurface(opts WindowOptions) (Surface, error) {
	if opts.Headless {
		return NewOffscreen(opts), nil
	}
	d := C.ui_open()
	if d == nil {
		return nil, fmt.Errorf("platform: XOpenDisplay failed (set DISPLAY or use headless)")
	}
	w, h := opts.Width, opts.Height
	if w < 1 {
		w = 640
	}
	if h < 1 {
		h = 480
	}
	title := opts.Title
	if title == "" {
		title = "uitoolkit"
	}
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	win := C.ui_create(d, C.int(w), C.int(h), ctitle)
	if opts.MinWidth > 0 || opts.MinHeight > 0 {
		mw, mh := opts.MinWidth, opts.MinHeight
		if mw < 1 {
			mw = 1
		}
		if mh < 1 {
			mh = 1
		}
		C.ui_resize_hints(d, win, C.int(mw), C.int(mh))
	}
	gc := C.ui_gc(d, win)
	s := &x11Surface{
		dpy:   d,
		win:   win,
		gc:    gc,
		title: title,
		img:   paintengine2d.NewImage(w, h),
		xbuf:  make([]byte, w*h*4),
	}
	s.rebuildImage()
	return s, nil
}

type x11Surface struct {
	dpy    *C.Display
	win    C.Window
	gc     C.GC
	ximg   *C.XImage
	title  string
	img    *paintengine2d.Image
	xbuf   []byte
	closed bool
}

func (s *x11Surface) Title() string                { return s.title }
func (s *x11Surface) Size() (w, h int)             { return s.img.Width, s.img.Height }
func (s *x11Surface) Buffer() *paintengine2d.Image { return s.img }
func (s *x11Surface) Closed() bool                 { return s.closed }

func (s *x11Surface) SetTitle(title string) {
	s.title = title
	if s.dpy == nil {
		return
	}
	ct := C.CString(title)
	C.XStoreName(s.dpy, s.win, ct)
	C.free(unsafe.Pointer(ct))
}

func (s *x11Surface) Resize(w, h int) error {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if s.img.Width == w && s.img.Height == h {
		return nil
	}
	s.img = paintengine2d.NewImage(w, h)
	s.xbuf = make([]byte, w*h*4)
	s.rebuildImage()
	return nil
}

func (s *x11Surface) rebuildImage() {
	if s.dpy == nil {
		return
	}
	if s.ximg != nil {
		C.ui_destroy_image(s.ximg)
		s.ximg = nil
	}
	s.ximg = C.ui_image(s.dpy, C.int(s.img.Width), C.int(s.img.Height), (*C.char)(unsafe.Pointer(&s.xbuf[0])))
}

func (s *x11Surface) copyRect(r paintengine2d.Rect) {
	x0, y0, x1, y1 := r.IntBounds()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > s.img.Width {
		x1 = s.img.Width
	}
	if y1 > s.img.Height {
		y1 = s.img.Height
	}
	if x0 >= x1 || y0 >= y1 {
		return
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			n := s.img.NRGBAAt(x, y)
			i := (y*s.img.Width + x) * 4
			s.xbuf[i+0] = n.B
			s.xbuf[i+1] = n.G
			s.xbuf[i+2] = n.R
			s.xbuf[i+3] = n.A
		}
	}
}

func (s *x11Surface) Present(dirty []paintengine2d.Rect) error {
	if s.closed || s.dpy == nil || s.ximg == nil {
		return nil
	}
	if len(dirty) == 0 {
		dirty = []paintengine2d.Rect{paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height))}
	}
	for _, r := range dirty {
		s.copyRect(r)
		x0, y0, x1, y1 := r.IntBounds()
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > s.img.Width {
			x1 = s.img.Width
		}
		if y1 > s.img.Height {
			y1 = s.img.Height
		}
		if x0 >= x1 || y0 >= y1 {
			continue
		}
		C.ui_put(s.dpy, s.win, s.gc, s.ximg, C.int(x0), C.int(y0), C.int(x0), C.int(y0), C.uint(x1-x0), C.uint(y1-y0))
	}
	C.ui_flush(s.dpy)
	return nil
}

func (s *x11Surface) Poll() []Event {
	if s.closed || s.dpy == nil {
		return nil
	}
	var out []Event
	for C.ui_pending(s.dpy) != 0 {
		var xe C.XEvent
		C.ui_next(s.dpy, &xe)
		out = append(out, s.translate(&xe)...)
	}
	return out
}

func (s *x11Surface) translate(xe *C.XEvent) []Event {
	switch C.ui_event_type(xe) {
	case C.Expose:
		r := paintengine2d.XYWH(float32(C.ui_expose_x(xe)), float32(C.ui_expose_y(xe)), float32(C.ui_expose_w(xe)), float32(C.ui_expose_h(xe)))
		return []Event{{Kind: EventExpose, Pos: r.Min, Width: int(r.Dx()), Height: int(r.Dy())}}
	case C.ConfigureNotify:
		w, h := int(C.ui_cfg_w(xe)), int(C.ui_cfg_h(xe))
		if w != s.img.Width || h != s.img.Height {
			_ = s.Resize(w, h)
			return []Event{{Kind: EventResize, Width: w, Height: h}}
		}
	case C.ButtonPress, C.ButtonRelease:
		btn := int(C.ui_btn(xe))
		x, y := float32(C.ui_btn_x(xe)), float32(C.ui_btn_y(xe))
		mods := xmods(uint(C.ui_btn_state(xe)))
		kind := EventMouseDown
		if C.ui_event_type(xe) == C.ButtonRelease {
			kind = EventMouseUp
		}
		if btn == 4 || btn == 5 || btn == 6 || btn == 7 {
			ev := Event{Kind: EventScroll, Pos: paintengine2d.Pt(x, y), Mods: mods}
			switch btn {
			case 4:
				ev.Scroll = paintengine2d.Pt(0, -48)
			case 5:
				ev.Scroll = paintengine2d.Pt(0, 48)
			case 6:
				ev.Scroll = paintengine2d.Pt(-48, 0)
			case 7:
				ev.Scroll = paintengine2d.Pt(48, 0)
			}
			return []Event{ev}
		}
		return []Event{{Kind: kind, Pos: paintengine2d.Pt(x, y), Button: xbutton(btn), Mods: mods}}
	case C.MotionNotify:
		return []Event{{
			Kind: EventMouseMove,
			Pos:  paintengine2d.Pt(float32(C.ui_mot_x(xe)), float32(C.ui_mot_y(xe))),
			Mods: xmods(uint(C.ui_mot_state(xe))),
		}}
	case C.KeyPress, C.KeyRelease:
		var buf [16]C.char
		var n C.int
		ks := C.ui_lookup(s.dpy, xe, &buf[0], 16, &n)
		kind := EventKeyDown
		if C.ui_event_type(xe) == C.KeyRelease {
			kind = EventKeyUp
		}
		ev := Event{Kind: kind, Key: xkey(ks), Mods: xmods(uint(C.ui_key_state(xe)))}
		out := []Event{ev}
		if kind == EventKeyDown && n > 0 {
			b := C.GoBytes(unsafe.Pointer(&buf[0]), n)
			r, _ := utf8.DecodeRune(b)
			if r >= 32 && r != 127 {
				out = append(out, Event{Kind: EventText, Rune: r, Mods: ev.Mods})
			}
		}
		return out
	case C.FocusIn:
		return []Event{{Kind: EventFocusIn}}
	case C.FocusOut:
		return []Event{{Kind: EventFocusOut}}
	case C.ClientMessage:
		if C.ui_is_delete(s.dpy, xe) != 0 {
			s.closed = true
			return []Event{{Kind: EventClose}}
		}
	}
	return nil
}

func (s *x11Surface) Close() error {
	if s.closed && s.dpy == nil {
		return nil
	}
	s.closed = true
	if s.ximg != nil {
		C.ui_destroy_image(s.ximg)
		s.ximg = nil
	}
	if s.dpy != nil {
		C.ui_close(s.dpy, s.win, s.gc)
		s.dpy = nil
	}
	return nil
}

func xbutton(b int) MouseButton {
	switch b {
	case 1:
		return ButtonLeft
	case 2:
		return ButtonMiddle
	case 3:
		return ButtonRight
	}
	return ButtonNone
}

func xmods(state uint) Modifiers {
	var m Modifiers
	const (
		shift = 1 << 0
		ctrl  = 1 << 2
		mod1  = 1 << 3
		mod4  = 1 << 6
	)
	if state&shift != 0 {
		m |= ModShift
	}
	if state&ctrl != 0 {
		m |= ModCtrl
	}
	if state&mod1 != 0 {
		m |= ModAlt
	}
	if state&mod4 != 0 {
		m |= ModSuper
	}
	return m
}

func xkey(ks C.KeySym) Key {
	switch ks {
	case C.XK_Escape:
		return KeyEscape
	case C.XK_Tab:
		return KeyTab
	case C.XK_Return, C.XK_KP_Enter:
		return KeyReturn
	case C.XK_BackSpace:
		return KeyBackspace
	case C.XK_Delete, C.XK_KP_Delete:
		return KeyDelete
	case C.XK_Left, C.XK_KP_Left:
		return KeyLeft
	case C.XK_Right, C.XK_KP_Right:
		return KeyRight
	case C.XK_Up, C.XK_KP_Up:
		return KeyUp
	case C.XK_Down, C.XK_KP_Down:
		return KeyDown
	case C.XK_Home, C.XK_KP_Home:
		return KeyHome
	case C.XK_End, C.XK_KP_End:
		return KeyEnd
	case C.XK_Page_Up, C.XK_KP_Page_Up:
		return KeyPageUp
	case C.XK_Page_Down, C.XK_KP_Page_Down:
		return KeyPageDown
	case C.XK_space:
		return KeySpace
	case C.XK_a, C.XK_A:
		return KeyA
	case C.XK_b, C.XK_B:
		return KeyB
	case C.XK_c, C.XK_C:
		return KeyC
	case C.XK_d, C.XK_D:
		return KeyD
	case C.XK_e, C.XK_E:
		return KeyE
	case C.XK_f, C.XK_F:
		return KeyF
	case C.XK_g, C.XK_G:
		return KeyG
	case C.XK_h, C.XK_H:
		return KeyH
	case C.XK_i, C.XK_I:
		return KeyI
	case C.XK_j, C.XK_J:
		return KeyJ
	case C.XK_k, C.XK_K:
		return KeyK
	case C.XK_l, C.XK_L:
		return KeyL
	case C.XK_m, C.XK_M:
		return KeyM
	case C.XK_n, C.XK_N:
		return KeyN
	case C.XK_o, C.XK_O:
		return KeyO
	case C.XK_p, C.XK_P:
		return KeyP
	case C.XK_q, C.XK_Q:
		return KeyQ
	case C.XK_r, C.XK_R:
		return KeyR
	case C.XK_s, C.XK_S:
		return KeyS
	case C.XK_t, C.XK_T:
		return KeyT
	case C.XK_u, C.XK_U:
		return KeyU
	case C.XK_v, C.XK_V:
		return KeyV
	case C.XK_w, C.XK_W:
		return KeyW
	case C.XK_x, C.XK_X:
		return KeyX
	case C.XK_y, C.XK_Y:
		return KeyY
	case C.XK_z, C.XK_Z:
		return KeyZ
	case C.XK_F1:
		return KeyF1
	case C.XK_F2:
		return KeyF2
	case C.XK_F3:
		return KeyF3
	case C.XK_F4:
		return KeyF4
	case C.XK_F5:
		return KeyF5
	case C.XK_F6:
		return KeyF6
	case C.XK_F7:
		return KeyF7
	case C.XK_F8:
		return KeyF8
	case C.XK_F9:
		return KeyF9
	case C.XK_F10:
		return KeyF10
	case C.XK_F11:
		return KeyF11
	case C.XK_F12:
		return KeyF12
	}
	return KeyUnknown
}
