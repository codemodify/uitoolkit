//go:build linux && cgo

package platform

/*
#cgo linux LDFLAGS: -lX11 -lXext -lXrandr
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <X11/Xresource.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>
#include <X11/extensions/XShm.h>
#include <X11/extensions/Xrandr.h>
#include <locale.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ipc.h>
#include <sys/select.h>
#include <sys/shm.h>
#include <sys/time.h>
#include <unistd.h>

static void ui_threads(void) {
	XInitThreads();
}

static void ui_detectable_repeat(Display* d) {
	int supported = 0;
	if (d) XkbSetDetectableAutoRepeat(d, True, &supported);
}

static void ui_init_locale(void) {
	setlocale(LC_CTYPE, "");
	if (XSupportsLocale()) {
		XSetLocaleModifiers("");
	}
}

static Display* ui_open(void) { return XOpenDisplay(NULL); }

static int ui_pending(Display* d) { return XPending(d); }

static void ui_next(Display* d, XEvent* e) { XNextEvent(d, e); }

static int ui_filter(Display* d, XEvent* e) { return XFilterEvent(e, None); }

static int ui_fd(Display* d) { return ConnectionNumber(d); }

static int ui_wait(Display* d, int ms) {
	int fd = ConnectionNumber(d);
	if (XPending(d)) return 1;
	fd_set fds;
	FD_ZERO(&fds);
	FD_SET(fd, &fds);
	struct timeval tv;
	tv.tv_sec = ms / 1000;
	tv.tv_usec = (ms % 1000) * 1000;
	return select(fd + 1, &fds, NULL, NULL, &tv);
}

static Window ui_create(Display* d, int w, int h, const char* title) {
	int s = DefaultScreen(d);
	XSetWindowAttributes swa;
	memset(&swa, 0, sizeof(swa));
	swa.background_pixel = BlackPixel(d, s);
	swa.border_pixel = BlackPixel(d, s);
	swa.bit_gravity = NorthWestGravity;
	swa.colormap = DefaultColormap(d, s);
	swa.event_mask = ExposureMask|KeyPressMask|KeyReleaseMask|ButtonPressMask|ButtonReleaseMask|
		PointerMotionMask|StructureNotifyMask|FocusChangeMask|PropertyChangeMask;
	Window win = XCreateWindow(d, RootWindow(d, s), 40, 40, (unsigned)w, (unsigned)h, 0,
		DefaultDepth(d, s), InputOutput, DefaultVisual(d, s),
		CWBackPixel|CWBorderPixel|CWBitGravity|CWColormap|CWEventMask, &swa);
	XStoreName(d, win, title);
	Atom net = XInternAtom(d, "_NET_WM_NAME", False);
	Atom utf8 = XInternAtom(d, "UTF8_STRING", False);
	XChangeProperty(d, win, net, utf8, 8, PropModeReplace, (const unsigned char*)title, (int)strlen(title));
	Atom proto = XInternAtom(d, "WM_DELETE_WINDOW", False);
	XSetWMProtocols(d, win, &proto, 1);
	XClassHint ch;
	ch.res_name = "uitoolkit";
	ch.res_class = "Uitoolkit";
	XSetClassHint(d, win, &ch);
	XSizeHints hints;
	memset(&hints, 0, sizeof(hints));
	hints.flags = PMinSize;
	hints.min_width = 200;
	hints.min_height = 120;
	XSetWMNormalHints(d, win, &hints);
	XFlush(d);
	return win;
}

static void ui_map(Display* d, Window w) {
	XMapWindow(d, w);
	XFlush(d);
}

static void ui_resize_hints(Display* d, Window w, int minw, int minh) {
	XSizeHints hints;
	memset(&hints, 0, sizeof(hints));
	hints.flags = PMinSize;
	hints.min_width = minw;
	hints.min_height = minh;
	XSetWMNormalHints(d, w, &hints);
}

static GC ui_gc(Display* d, Window w) { return XCreateGC(d, w, 0, NULL); }

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

static int ui_img_bpl(XImage* img) { return img ? img->bytes_per_line : 0; }
static int ui_img_msb(XImage* img) { return img && img->byte_order == MSBFirst; }
static unsigned long ui_red_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->red_mask; }
static unsigned long ui_green_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->green_mask; }
static unsigned long ui_blue_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->blue_mask; }

static int ui_event_type(XEvent* e) { return e->type; }
static Window ui_event_window(XEvent* e) { return e->xany.window; }

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

#ifdef X_HAVE_UTF8_STRING
static int ui_lookup_im(XIC ic, Display* d, XEvent* e, char* buf, int n, KeySym* ks) {
	*ks = 0;
	if (ic) {
		Status st = 0;
		int len = Xutf8LookupString(ic, &e->xkey, buf, n, ks, &st);
		if (st == XBufferOverflow) {
			return -len;
		}
		return len;
	}
	return XLookupString(&e->xkey, buf, n, ks, NULL);
}
#else
static int ui_lookup_im(XIC ic, Display* d, XEvent* e, char* buf, int n, KeySym* ks) {
	(void)ic;
	(void)d;
	*ks = 0;
	return XLookupString(&e->xkey, buf, n, ks, NULL);
}
#endif

static Atom ui_delete_atom(Display* d) { return XInternAtom(d, "WM_DELETE_WINDOW", False); }

static int ui_is_delete(Display* d, XEvent* e) {
	if (e->type != ClientMessage) return 0;
	Atom del = XInternAtom(d, "WM_DELETE_WINDOW", False);
	return (Atom)e->xclient.data.l[0] == del;
}

static void ui_destroy_win(Display* d, Window w, GC gc) {
	if (gc) XFreeGC(d, gc);
	if (w) XDestroyWindow(d, w);
	if (d) XFlush(d);
}

static XIM ui_open_im(Display* d) {
	if (!d) return NULL;
	return XOpenIM(d, NULL, NULL, NULL);
}

static XIC ui_create_ic(XIM im, Window w) {
	if (!im || !w) return NULL;
	return XCreateIC(im, XNInputStyle, XIMPreeditNothing | XIMStatusNothing,
		XNClientWindow, w, XNFocusWindow, w, NULL);
}

static void ui_set_ic_focus(XIC ic) { if (ic) XSetICFocus(ic); }
static void ui_unset_ic_focus(XIC ic) { if (ic) XUnsetICFocus(ic); }
static void ui_destroy_ic(XIC ic) { if (ic) XDestroyIC(ic); }

static Window ui_helper(Display* d) {
	int s = DefaultScreen(d);
	Window w = XCreateSimpleWindow(d, RootWindow(d, s), -64, -64, 1, 1, 0,
		BlackPixel(d, s), BlackPixel(d, s));
	XSelectInput(d, w, PropertyChangeMask);
	return w;
}

static Atom ui_atom(Display* d, const char* n) { return XInternAtom(d, n, False); }

static void ui_set_owner(Display* d, Window w, Atom sel) {
	XSetSelectionOwner(d, sel, w, CurrentTime);
}

static Window ui_owner(Display* d, Atom sel) { return XGetSelectionOwner(d, sel); }

static void ui_convert(Display* d, Window w, Atom sel, Atom target, Atom prop) {
	XConvertSelection(d, sel, target, prop, w, CurrentTime);
}

static Window ui_sr_requestor(XEvent* e) { return e->xselectionrequest.requestor; }
static Atom ui_sr_selection(XEvent* e) { return e->xselectionrequest.selection; }
static Atom ui_sr_target(XEvent* e) { return e->xselectionrequest.target; }
static Atom ui_sr_property(XEvent* e) { return e->xselectionrequest.property; }

static Atom ui_sc_selection(XEvent* e) { return e->xselectionclear.selection; }

static Atom ui_sn_property(XEvent* e) { return e->xselection.property; }
static Atom ui_sn_target(XEvent* e) { return e->xselection.target; }
static Window ui_sn_requestor(XEvent* e) { return e->xselection.requestor; }

static void ui_send_sel_notify(Display* d, XEvent* req, Atom prop) {
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.xselection.type = SelectionNotify;
	ev.xselection.display = d;
	ev.xselection.requestor = req->xselectionrequest.requestor;
	ev.xselection.selection = req->xselectionrequest.selection;
	ev.xselection.target = req->xselectionrequest.target;
	ev.xselection.property = prop;
	ev.xselection.time = req->xselectionrequest.time;
	XSendEvent(d, req->xselectionrequest.requestor, False, NoEventMask, &ev);
}

static void ui_change_prop8(Display* d, Window w, Atom prop, Atom type, const char* data, int n) {
	XChangeProperty(d, w, prop, type, 8, PropModeReplace, (const unsigned char*)data, n);
}

static void ui_change_prop32(Display* d, Window w, Atom prop, Atom type, unsigned long* data, int n) {
	XChangeProperty(d, w, prop, type, 32, PropModeReplace, (const unsigned char*)data, n);
}

static int ui_get_prop(Display* d, Window w, Atom prop, unsigned char** data, unsigned long* nitems, Atom* type) {
	Atom actual = 0;
	int fmt = 0;
	unsigned long n = 0, rem = 0;
	unsigned char* p = NULL;
	int st = XGetWindowProperty(d, w, prop, 0L, 256 * 1024, True, AnyPropertyType,
		&actual, &fmt, &n, &rem, &p);
	if (st != Success || p == NULL) {
		*data = NULL;
		*nitems = 0;
		*type = None;
		return 0;
	}
	*type = actual;
	*nitems = n;
	*data = p;
	return fmt;
}

static void ui_xfree(void* p) { if (p) XFree(p); }

static void ui_refresh_mapping(XEvent* e) {
	if (e->type == MappingNotify) {
		XRefreshKeyboardMapping(&e->xmapping);
	}
}

static double ui_xft_dpi(Display* d) {
	char* rms = XResourceManagerString(d);
	if (!rms) return 0;
	XrmInitialize();
	XrmDatabase db = XrmGetStringDatabase(rms);
	if (!db) return 0;
	char* type = NULL;
	XrmValue val;
	double dpi = 0;
	if (XrmGetResource(db, "Xft.dpi", "Xft.Dpi", &type, &val) && val.addr) {
		dpi = atof((char*)val.addr);
	}
	XrmDestroyDatabase(db);
	return dpi;
}

static double ui_screen_dpi(Display* d) {
	int s = DefaultScreen(d);
	int hmm = DisplayHeightMM(d, s);
	int hpx = DisplayHeight(d, s);
	if (hmm <= 0 || hpx <= 0) return 0;
	return (double)hpx * 25.4 / (double)hmm;
}

static double ui_randr_dpi(Display* d) {
	int evb = 0, erb = 0;
	if (!d || !XRRQueryExtension(d, &evb, &erb)) return 0;
	XRRScreenResources* res = XRRGetScreenResourcesCurrent(d, DefaultRootWindow(d));
	if (!res) return 0;
	double best = 0;
	for (int i = 0; i < res->noutput; i++) {
		XRROutputInfo* oi = XRRGetOutputInfo(d, res, res->outputs[i]);
		if (!oi) continue;
		if (oi->connection == RR_Connected && oi->crtc && oi->mm_height > 0) {
			XRRCrtcInfo* ci = XRRGetCrtcInfo(d, res, oi->crtc);
			if (ci && ci->height > 0) {
				double dpi = (double)ci->height * 25.4 / (double)oi->mm_height;
				if (dpi > best) best = dpi;
			}
			if (ci) XRRFreeCrtcInfo(ci);
		}
		XRRFreeOutputInfo(oi);
	}
	XRRFreeScreenResources(res);
	return best;
}

static void ui_select_randr(Display* d) {
	int evb = 0, erb = 0;
	if (!d || !XRRQueryExtension(d, &evb, &erb)) return;
	XRRSelectInput(d, DefaultRootWindow(d), RRScreenChangeNotifyMask);
}

static long ui_max_req(Display* d) {
	if (!d) return 0;
	return (long)XMaxRequestSize(d) * 4;
}

static int ui_prop_state(XEvent* e) { return e->xproperty.state; }
static Atom ui_prop_atom(XEvent* e) { return e->xproperty.atom; }
static Window ui_prop_window(XEvent* e) { return e->xproperty.window; }

static void ui_select_prop(Display* d, Window w) {
	XSelectInput(d, w, PropertyChangeMask);
}

static void ui_delete_prop(Display* d, Window w, Atom prop) {
	XDeleteProperty(d, w, prop);
}

static void ui_resize_win(Display* d, Window w, int width, int height) {
	XResizeWindow(d, w, (unsigned)width, (unsigned)height);
	XFlush(d);
}

static void ui_ewmh_state(Display* d, Window w, long action, Atom a, Atom b) {
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.xclient.type = ClientMessage;
	ev.xclient.window = w;
	ev.xclient.message_type = XInternAtom(d, "_NET_WM_STATE", False);
	ev.xclient.format = 32;
	ev.xclient.data.l[0] = action;
	ev.xclient.data.l[1] = (long)a;
	ev.xclient.data.l[2] = (long)b;
	ev.xclient.data.l[3] = 1;
	XSendEvent(d, DefaultRootWindow(d), False, SubstructureRedirectMask|SubstructureNotifyMask, &ev);
	XFlush(d);
}

static int uitk_xerr = 0;
static int uitk_on_xerr(Display* d, XErrorEvent* e) {
	(void)d;
	uitk_xerr = e->error_code;
	return 0;
}

static int ui_shm_query(Display* d) {
	int ev = 0, err = 0, major = 0, minor = 0, pix = 0;
	if (!d || !XShmQueryExtension(d)) return 0;
	if (!XShmQueryVersion(d, &major, &minor, &pix)) return 0;
	return 1;
}

static XImage* ui_shm_image(Display* d, int w, int h, void** info_out, void** addr) {
	XShmSegmentInfo* info = (XShmSegmentInfo*)calloc(1, sizeof(XShmSegmentInfo));
	if (!info) return NULL;
	Visual* v = DefaultVisual(d, DefaultScreen(d));
	int depth = DefaultDepth(d, DefaultScreen(d));
	XImage* img = XShmCreateImage(d, v, (unsigned)depth, ZPixmap, NULL, info, (unsigned)w, (unsigned)h);
	if (!img) {
		free(info);
		return NULL;
	}
	info->shmid = shmget(IPC_PRIVATE, (size_t)img->bytes_per_line * (size_t)h, IPC_CREAT | 0600);
	if (info->shmid < 0) {
		img->data = NULL;
		XDestroyImage(img);
		free(info);
		return NULL;
	}
	info->shmaddr = (char*)shmat(info->shmid, NULL, 0);
	shmctl(info->shmid, IPC_RMID, NULL);
	if (info->shmaddr == (char*)(-1) || info->shmaddr == NULL) {
		img->data = NULL;
		XDestroyImage(img);
		free(info);
		return NULL;
	}
	info->readOnly = False;
	img->data = info->shmaddr;
	{
		XErrorHandler prev = XSetErrorHandler(uitk_on_xerr);
		uitk_xerr = 0;
		if (!XShmAttach(d, info)) {
			uitk_xerr = 1;
		}
		XSync(d, False);
		XSetErrorHandler(prev);
	}
	if (uitk_xerr) {
		shmdt(info->shmaddr);
		img->data = NULL;
		XDestroyImage(img);
		free(info);
		return NULL;
	}
	*info_out = info;
	*addr = info->shmaddr;
	return img;
}

static void ui_shm_put(Display* d, Window w, GC gc, XImage* img, int sx, int sy, int dx, int dy, unsigned pw, unsigned ph) {
	XShmPutImage(d, w, gc, img, sx, sy, dx, dy, pw, ph, False);
}

static void ui_shm_destroy(Display* d, XImage* img, void* info_ptr) {
	XShmSegmentInfo* info = (XShmSegmentInfo*)info_ptr;
	if (d && info) {
		XShmDetach(d, info);
	}
	if (info && info->shmaddr && info->shmaddr != (char*)(-1)) {
		shmdt(info->shmaddr);
	}
	if (img) {
		img->data = NULL;
		XDestroyImage(img);
	}
	free(info);
}

static char* ui_reset_ic(XIC ic) {
	if (!ic) return NULL;
	return XmbResetIC(ic);
}

static void ui_set_spot(XIC ic, int x, int y) {
	XPoint pt;
	if (!ic) return;
	pt.x = (short)x;
	pt.y = (short)y;
	XVaNestedList a = XVaCreateNestedList(0, XNSpotLocation, &pt, NULL);
	if (a) {
		XSetICValues(ic, XNPreeditAttributes, a, NULL);
		XFree(a);
	}
}
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"time"
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
	c, err := x11Retain()
	if err != nil {
		return nil, err
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
	x11Mu.Lock()
	win := C.ui_create(c.dpy, C.int(w), C.int(h), ctitle)
	if opts.MinWidth > 0 || opts.MinHeight > 0 {
		mw, mh := opts.MinWidth, opts.MinHeight
		if mw < 1 {
			mw = 1
		}
		if mh < 1 {
			mh = 1
		}
		C.ui_resize_hints(c.dpy, win, C.int(mw), C.int(mh))
	}
	gc := C.ui_gc(c.dpy, win)
	ic, cbs := x11CreateIC(c.im, win)
	s := &x11Surface{
		conn:  c,
		win:   win,
		gc:    gc,
		ic:     ic,
		ximCbs: cbs,
		title:  title,
		img:   paintengine2d.NewImage(w, h),
		rmask: uint32(C.ui_red_mask(c.dpy)),
		gmask: uint32(C.ui_green_mask(c.dpy)),
		bmask: uint32(C.ui_blue_mask(c.dpy)),
	}
	s.rebuildImageLocked()
	c.surfaces[win] = s
	x11Mu.Unlock()
	return s, nil
}

type x11Conn struct {
	dpy      *C.Display
	im       C.XIM
	helper   C.Window
	refs     int
	keep     bool
	scale    float32
	surfaces map[C.Window]*x11Surface
	queues   map[C.Window][]Event

	atomClipboard C.Atom
	atomPrimary   C.Atom
	atomUTF8      C.Atom
	atomTargets   C.Atom
	atomText      C.Atom
	atomINCR      C.Atom
	atomProp      C.Atom
	atomString    C.Atom
	atomNetState  C.Atom
	atomMaxVert   C.Atom
	atomMaxHorz   C.Atom
	atomFullscr   C.Atom

	clipText  string
	ownClip   bool
	ownPrim   bool
	pasteText string
	pasteDone bool
	pasteWant C.Atom
	incrRecv  incrRecvState
	incrSends []incrSendState
	maxReq    int
}

type x11Surface struct {
	conn   *x11Conn
	win    C.Window
	gc     C.GC
	ic     C.XIC
	ximg   *C.XImage
	title  string
	img    *paintengine2d.Image
	xbuf   []byte
	stride int
	msb    bool
	rmask  uint32
	gmask  uint32
	bmask  uint32
	mapped   bool
	closed   bool
	shm      bool
	shmInfo  unsafe.Pointer
	imeSpotX   int
	imeSpotY   int
	preeditBuf string
	ximCbs     unsafe.Pointer
}

var (
	x11Mu   sync.Mutex
	x11Once sync.Once
	x11c    *x11Conn
)

func x11InitOnce() {
	x11Once.Do(func() {
		C.ui_threads()
		C.ui_init_locale()
	})
}

func x11OpenLocked() (*x11Conn, error) {
	x11InitOnce()
	d := C.ui_open()
	if d == nil {
		return nil, fmt.Errorf("platform: XOpenDisplay failed (set DISPLAY or use headless)")
	}
	c := &x11Conn{
		dpy:      d,
		surfaces: make(map[C.Window]*x11Surface),
		queues:   make(map[C.Window][]Event),
	}
	c.im = C.ui_open_im(d)
	c.helper = C.ui_helper(d)
	c.atomClipboard = internAtom(d, "CLIPBOARD")
	c.atomPrimary = C.XA_PRIMARY
	c.atomUTF8 = internAtom(d, "UTF8_STRING")
	c.atomTargets = internAtom(d, "TARGETS")
	c.atomText = internAtom(d, "TEXT")
	c.atomINCR = internAtom(d, "INCR")
	c.atomProp = internAtom(d, "UITK_CLIP")
	c.atomString = C.XA_STRING
	c.atomNetState = internAtom(d, "_NET_WM_STATE")
	c.atomMaxVert = internAtom(d, "_NET_WM_STATE_MAXIMIZED_VERT")
	c.atomMaxHorz = internAtom(d, "_NET_WM_STATE_MAXIMIZED_HORZ")
	c.atomFullscr = internAtom(d, "_NET_WM_STATE_FULLSCREEN")
	c.maxReq = int(C.ui_max_req(d))
	C.ui_detectable_repeat(d)
	C.ui_select_randr(d)
	dpi := float32(C.ui_xft_dpi(d))
	if dpi <= 0 {
		dpi = float32(C.ui_randr_dpi(d))
	}
	if dpi <= 0 {
		dpi = float32(C.ui_screen_dpi(d))
	}
	c.scale = scaleFromDPI(dpi)
	if c.scale <= 0 {
		c.scale = 1
	}
	x11c = c
	return c, nil
}

func x11Retain() (*x11Conn, error) {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if x11c == nil {
		if _, err := x11OpenLocked(); err != nil {
			return nil, err
		}
	}
	x11c.refs++
	return x11c, nil
}

func x11Get() (*x11Conn, error) {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if x11c != nil {
		return x11c, nil
	}
	if os.Getenv("DISPLAY") == "" {
		return nil, fmt.Errorf("platform: no DISPLAY")
	}
	return x11OpenLocked()
}

func (c *x11Conn) release() {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	c.refs--
	if c.refs < 0 {
		c.refs = 0
	}
	if c.refs == 0 && !c.keep {
		c.closeLocked()
	}
}

func (c *x11Conn) closeLocked() {
	if c.im != nil {
		C.XCloseIM(c.im)
		c.im = nil
	}
	if c.helper != 0 && c.dpy != nil {
		C.XDestroyWindow(c.dpy, c.helper)
		c.helper = 0
	}
	if c.dpy != nil {
		C.XCloseDisplay(c.dpy)
		c.dpy = nil
	}
	if x11c == c {
		x11c = nil
	}
}

func (c *x11Conn) unregister(win C.Window) {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	delete(c.surfaces, win)
	delete(c.queues, win)
}

func nativeDetectScale() float32 {
	if s := waylandDetectScale(); s > 0 {
		return s
	}
	if os.Getenv("DISPLAY") == "" {
		return 0
	}
	x11Mu.Lock()
	if x11c != nil && x11c.scale > 0 {
		s := x11c.scale
		x11Mu.Unlock()
		return s
	}
	x11Mu.Unlock()
	x11InitOnce()
	d := C.ui_open()
	if d == nil {
		return 0
	}
	dpi := float32(C.ui_xft_dpi(d))
	if dpi <= 0 {
		dpi = float32(C.ui_randr_dpi(d))
	}
	if dpi <= 0 {
		dpi = float32(C.ui_screen_dpi(d))
	}
	C.XCloseDisplay(d)
	return scaleFromDPI(dpi)
}

func (s *x11Surface) Title() string                { return s.title }
func (s *x11Surface) Size() (w, h int)             { return s.img.Width, s.img.Height }
func (s *x11Surface) Buffer() *paintengine2d.Image { return s.img }
func (s *x11Surface) Closed() bool                 { return s.closed }
func (s *x11Surface) Scale() float32 {
	if s.conn != nil && s.conn.scale > 0 {
		return s.conn.scale
	}
	return 1
}

func (s *x11Surface) SetTitle(title string) {
	s.title = title
	if s.conn == nil || s.conn.dpy == nil {
		return
	}
	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	x11Mu.Lock()
	C.XStoreName(s.conn.dpy, s.win, ct)
	net := internAtom(s.conn.dpy, "_NET_WM_NAME")
	C.ui_change_prop8(s.conn.dpy, s.win, net, s.conn.atomUTF8, ct, C.int(len(title)))
	x11Mu.Unlock()
}

func internAtom(d *C.Display, name string) C.Atom {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	return C.ui_atom(d, cn)
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
	x11Mu.Lock()
	s.rebuildImageLocked()
	if s.conn != nil && s.conn.dpy != nil && s.win != 0 {
		C.ui_resize_win(s.conn.dpy, s.win, C.int(w), C.int(h))
	}
	x11Mu.Unlock()
	return nil
}

func (s *x11Surface) rebuildImageLocked() {
	if s.conn == nil || s.conn.dpy == nil {
		return
	}
	s.destroyImageLocked()
	w, h := s.img.Width, s.img.Height
	if C.ui_shm_query(s.conn.dpy) != 0 {
		var info, addr unsafe.Pointer
		img := C.ui_shm_image(s.conn.dpy, C.int(w), C.int(h), &info, &addr)
		if img != nil && addr != nil {
			s.ximg = img
			s.shm = true
			s.shmInfo = info
			s.stride = int(C.ui_img_bpl(img))
			s.msb = C.ui_img_msb(img) != 0
			need := s.stride * h
			if need < w*h*4 {
				need = w * h * 4
			}
			s.xbuf = unsafe.Slice((*byte)(addr), need)
			if s.stride < w*4 {
				s.stride = w * 4
			}
			return
		}
	}
	s.shm = false
	s.xbuf = make([]byte, w*h*4+64)
	s.ximg = C.ui_image(s.conn.dpy, C.int(w), C.int(h), (*C.char)(unsafe.Pointer(&s.xbuf[0])))
	if s.ximg != nil {
		s.stride = int(C.ui_img_bpl(s.ximg))
		s.msb = C.ui_img_msb(s.ximg) != 0
		need := s.stride * h
		if need > len(s.xbuf) {
			s.xbuf = make([]byte, need)
			C.ui_destroy_image(s.ximg)
			s.ximg = C.ui_image(s.conn.dpy, C.int(w), C.int(h), (*C.char)(unsafe.Pointer(&s.xbuf[0])))
			if s.ximg != nil {
				s.stride = int(C.ui_img_bpl(s.ximg))
			}
		}
	}
	if s.stride < w*4 {
		s.stride = w * 4
	}
}

func (s *x11Surface) destroyImageLocked() {
	if s.ximg != nil && s.shm {
		var dpy *C.Display
		if s.conn != nil {
			dpy = s.conn.dpy
		}
		C.ui_shm_destroy(dpy, s.ximg, s.shmInfo)
		s.ximg = nil
		s.shmInfo = nil
		s.shm = false
		s.xbuf = nil
		return
	}
	if s.ximg != nil {
		C.ui_destroy_image(s.ximg)
		s.ximg = nil
	}
	s.xbuf = nil
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
	stride := s.stride
	if stride < s.img.Width*4 {
		stride = s.img.Width * 4
	}
	// Typical LE TrueColor (red 0xff0000) is the same BGRA layout as
	// Wayland XRGB8888 — one damage-rect swizzle, not NRGBAAt+pack
	// per pixel (that path was ~70ms/frame at 1000×760).
	if !s.msb && s.rmask == 0x00ff0000 && s.gmask == 0x0000ff00 && s.bmask == 0x000000ff {
		copyImageRect(s.xbuf, stride, s.img, r, true, true)
		return
	}
	for y := y0; y < y1; y++ {
		row := y * stride
		for x := x0; x < x1; x++ {
			n := s.img.NRGBAAt(x, y)
			i := row + x*4
			if i+4 > len(s.xbuf) {
				return
			}
			packXPixel(s.xbuf[i:], n.R, n.G, n.B, n.A, s.msb, s.rmask, s.gmask, s.bmask)
		}
	}
}

func (s *x11Surface) Present(dirty []paintengine2d.Rect) error {
	if s.closed || s.conn == nil || s.conn.dpy == nil || s.ximg == nil {
		return nil
	}
	if len(dirty) == 0 {
		dirty = []paintengine2d.Rect{paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height))}
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if s.closed || s.conn.dpy == nil || s.ximg == nil {
		return nil
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
		if s.shm {
			C.ui_shm_put(s.conn.dpy, s.win, s.gc, s.ximg, C.int(x0), C.int(y0), C.int(x0), C.int(y0), C.uint(x1-x0), C.uint(y1-y0))
		} else {
			C.ui_put(s.conn.dpy, s.win, s.gc, s.ximg, C.int(x0), C.int(y0), C.int(x0), C.int(y0), C.uint(x1-x0), C.uint(y1-y0))
		}
	}
	if !s.mapped {
		C.ui_map(s.conn.dpy, s.win)
		s.mapped = true
	}
	C.ui_flush(s.conn.dpy)
	return nil
}

func (s *x11Surface) Poll() []Event {
	if s.closed || s.conn == nil {
		return nil
	}
	s.conn.drain()
	x11Mu.Lock()
	ev := s.conn.queues[s.win]
	s.conn.queues[s.win] = nil
	x11Mu.Unlock()
	return ev
}

func (c *x11Conn) drain() {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	c.drainLocked()
}

func (c *x11Conn) drainLocked() {
	if c.dpy == nil {
		return
	}
	for C.ui_pending(c.dpy) != 0 {
		var xe C.XEvent
		C.ui_next(c.dpy, &xe)
		if C.ui_filter(c.dpy, &xe) != 0 {
			continue
		}
		switch C.ui_event_type(&xe) {
		case C.SelectionRequest:
			c.handleSelReq(&xe)
			continue
		case C.SelectionClear:
			c.handleSelClear(&xe)
			continue
		case C.SelectionNotify:
			c.handleSelNotify(&xe)
			continue
		case C.PropertyNotify:
			c.handleProperty(&xe)
			continue
		case C.MappingNotify:
			C.ui_refresh_mapping(&xe)
			continue
		}
		win := C.ui_event_window(&xe)
		s := c.surfaces[win]
		if s == nil {
			continue
		}
		c.queues[win] = append(c.queues[win], s.translate(&xe)...)
	}
}

func (s *x11Surface) translate(xe *C.XEvent) []Event {
	switch C.ui_event_type(xe) {
	case C.Expose:
		r := paintengine2d.XYWH(float32(C.ui_expose_x(xe)), float32(C.ui_expose_y(xe)), float32(C.ui_expose_w(xe)), float32(C.ui_expose_h(xe)))
		return []Event{{Kind: EventExpose, Pos: r.Min, Width: int(r.Dx()), Height: int(r.Dy())}}
	case C.ConfigureNotify:
		w, h := int(C.ui_cfg_w(xe)), int(C.ui_cfg_h(xe))
		if w != s.img.Width || h != s.img.Height {
			if w < 1 {
				w = 1
			}
			if h < 1 {
				h = 1
			}
			s.img = paintengine2d.NewImage(w, h)
			s.rebuildImageLocked()
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
		kind := EventKeyDown
		if C.ui_event_type(xe) == C.KeyRelease {
			kind = EventKeyUp
		}
		var buf [128]C.char
		var ks C.KeySym
		n := int(C.ui_lookup_im(s.ic, s.conn.dpy, xe, &buf[0], 128, &ks))
		if n < 0 {
			need := -n
			if need > 8 && need < 4096 {
				big := make([]C.char, need+1)
				n = int(C.ui_lookup_im(s.ic, s.conn.dpy, xe, &big[0], C.int(need+1), &ks))
				ev := Event{Kind: kind, Key: xkey(ks), Mods: xmods(uint(C.ui_key_state(xe)))}
				out := []Event{ev}
				if kind == EventKeyDown && n > 0 && !ev.Mods.Ctrl() {
					out = append(out, textEvents(C.GoBytes(unsafe.Pointer(&big[0]), C.int(n)), ev.Mods)...)
				}
				return out
			}
			n = 0
		}
		ev := Event{Kind: kind, Key: xkey(ks), Mods: xmods(uint(C.ui_key_state(xe)))}
		out := []Event{ev}
		if kind == EventKeyDown && n > 0 && !ev.Mods.Ctrl() {
			out = append(out, textEvents(C.GoBytes(unsafe.Pointer(&buf[0]), C.int(n)), ev.Mods)...)
		}
		return out
	case C.FocusIn:
		C.ui_set_ic_focus(s.ic)
		return []Event{{Kind: EventFocusIn}}
	case C.FocusOut:
		C.ui_unset_ic_focus(s.ic)
		if leftover := C.ui_reset_ic(s.ic); leftover != nil {
			txt := C.GoString(leftover)
			C.ui_xfree(unsafe.Pointer(leftover))
			out := []Event{{Kind: EventFocusOut}, {Kind: EventIMECancel}}
			if txt != "" {
				out = append(out, Event{Kind: EventIMECommit, Text: txt})
			}
			return out
		}
		return []Event{{Kind: EventFocusOut}, {Kind: EventIMECancel}}
	case C.DestroyNotify:
		s.closed = true
		return []Event{{Kind: EventClose}}
	case C.ClientMessage:
		if C.ui_is_delete(s.conn.dpy, xe) != 0 {
			s.closed = true
			return []Event{{Kind: EventClose}}
		}
	}
	return nil
}

func textEvents(b []byte, mods Modifiers) []Event {
	var out []Event
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r >= 32 && r != 127 {
			out = append(out, Event{Kind: EventText, Rune: r, Mods: mods})
		}
		if size < 1 {
			break
		}
		b = b[size:]
	}
	return out
}

func (s *x11Surface) Close() error {
	if s.closed && (s.conn == nil || s.conn.dpy == nil) {
		return nil
	}
	s.closed = true
	if s.conn != nil {
		s.conn.unregister(s.win)
	}
	x11Mu.Lock()
	if s.ic != nil {
		C.ui_destroy_ic(s.ic)
		s.ic = nil
	}
	if s.ximCbs != nil {
		x11FreeCbs(s.ximCbs)
		s.ximCbs = nil
	}
	s.destroyImageLocked()
	if s.conn != nil && s.conn.dpy != nil {
		C.ui_destroy_win(s.conn.dpy, s.win, s.gc)
		s.win = 0
		s.gc = nil
	}
	x11Mu.Unlock()
	if s.conn != nil {
		s.conn.release()
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

// MapXKeySym is exported for X11-tagged tests.
func MapXKeySym(ks uint64) Key { return xkey(C.KeySym(ks)) }

type incrRecvState struct {
	active bool
	win    C.Window
	prop   C.Atom
	buf    []byte
}

type incrSendState struct {
	requestor C.Window
	prop      C.Atom
	typ       C.Atom
	data      []byte
	off       int
	chunk     int
}

func (c *x11Conn) handleSelReq(xe *C.XEvent) {
	req := C.ui_sr_requestor(xe)
	target := C.ui_sr_target(xe)
	prop := C.ui_sr_property(xe)
	if prop == 0 {
		prop = target
	}
	sel := C.ui_sr_selection(xe)
	if (sel == c.atomClipboard && !c.ownClip) || (sel == c.atomPrimary && !c.ownPrim) {
		C.ui_send_sel_notify(c.dpy, xe, 0)
		return
	}
	data := c.clipText
	switch target {
	case c.atomTargets:
		atoms := []C.ulong{
			C.ulong(c.atomTargets), C.ulong(c.atomUTF8), C.ulong(c.atomString),
			C.ulong(c.atomText), C.ulong(c.atomINCR),
		}
		C.ui_change_prop32(c.dpy, req, prop, C.XA_ATOM, &atoms[0], C.int(len(atoms)))
		C.ui_send_sel_notify(c.dpy, xe, prop)
	case c.atomUTF8, c.atomString, c.atomText:
		raw := []byte(data)
		thr := INCRThreshold(c.maxReq)
		if len(raw) > thr {
			C.ui_select_prop(c.dpy, req)
			sz := []C.ulong{C.ulong(len(raw))}
			C.ui_change_prop32(c.dpy, req, prop, c.atomINCR, &sz[0], 1)
			C.ui_send_sel_notify(c.dpy, xe, prop)
			c.incrSends = append(c.incrSends, incrSendState{
				requestor: req,
				prop:      prop,
				typ:       target,
				data:      raw,
				chunk:     INCRChunkSize(thr),
			})
			c.keep = true
			return
		}
		ct := C.CString(data)
		C.ui_change_prop8(c.dpy, req, prop, target, ct, C.int(len(data)))
		C.free(unsafe.Pointer(ct))
		C.ui_send_sel_notify(c.dpy, xe, prop)
	default:
		C.ui_send_sel_notify(c.dpy, xe, 0)
	}
}

func (c *x11Conn) handleSelClear(xe *C.XEvent) {
	sel := C.ui_sc_selection(xe)
	if sel == c.atomClipboard {
		c.ownClip = false
	}
	if sel == c.atomPrimary {
		c.ownPrim = false
	}
	if !c.ownClip && !c.ownPrim && len(c.incrSends) == 0 {
		c.keep = false
	}
}

func (c *x11Conn) handleSelNotify(xe *C.XEvent) {
	prop := C.ui_sn_property(xe)
	if prop == 0 {
		c.pasteDone = true
		c.pasteText = ""
		return
	}
	var data *C.uchar
	var n C.ulong
	var typ C.Atom
	fmtb := C.ui_get_prop(c.dpy, C.ui_sn_requestor(xe), prop, &data, &n, &typ)
	if typ == c.atomINCR {
		if data != nil {
			C.ui_xfree(unsafe.Pointer(data))
		}
		c.incrRecv = incrRecvState{active: true, win: C.ui_sn_requestor(xe), prop: prop}
		return
	}
	if data == nil || n == 0 {
		c.pasteDone = true
		c.pasteText = ""
		return
	}
	defer C.ui_xfree(unsafe.Pointer(data))
	if fmtb == 8 {
		c.pasteText = C.GoStringN((*C.char)(unsafe.Pointer(data)), C.int(n))
	}
	c.pasteDone = true
}

func (c *x11Conn) handleProperty(xe *C.XEvent) {
	win := C.ui_prop_window(xe)
	atom := C.ui_prop_atom(xe)
	state := C.ui_prop_state(xe)
	const propertyNewValue = 0
	const propertyDelete = 1
	if c.incrRecv.active && win == c.incrRecv.win && atom == c.incrRecv.prop && state == propertyNewValue {
		var data *C.uchar
		var n C.ulong
		var typ C.Atom
		fmtb := C.ui_get_prop(c.dpy, win, atom, &data, &n, &typ)
		_ = typ
		var piece []byte
		if data != nil && n > 0 && fmtb == 8 {
			piece = C.GoBytes(unsafe.Pointer(data), C.int(n))
			C.ui_xfree(unsafe.Pointer(data))
		} else if data != nil {
			C.ui_xfree(unsafe.Pointer(data))
		}
		var done bool
		c.incrRecv.buf, done = AppendINCRPiece(c.incrRecv.buf, piece)
		if done {
			c.pasteText = string(c.incrRecv.buf)
			c.pasteDone = true
			c.incrRecv = incrRecvState{}
		}
		return
	}
	if state != propertyDelete {
		return
	}
	for i := 0; i < len(c.incrSends); i++ {
		s := &c.incrSends[i]
		if s.requestor != win || s.prop != atom {
			continue
		}
		if s.off >= len(s.data) {
			C.ui_change_prop8(c.dpy, s.requestor, s.prop, s.typ, (*C.char)(nil), 0)
			c.incrSends = append(c.incrSends[:i], c.incrSends[i+1:]...)
			if !c.ownClip && !c.ownPrim && len(c.incrSends) == 0 {
				c.keep = false
			}
			return
		}
		end := s.off + s.chunk
		if end > len(s.data) {
			end = len(s.data)
		}
		chunk := s.data[s.off:end]
		s.off = end
		var ptr *C.char
		if len(chunk) > 0 {
			ptr = (*C.char)(unsafe.Pointer(&chunk[0]))
		}
		C.ui_change_prop8(c.dpy, s.requestor, s.prop, s.typ, ptr, C.int(len(chunk)))
		return
	}
}

func (c *x11Conn) setClipboard(s string) {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if c.dpy == nil || c.helper == 0 {
		return
	}
	c.clipText = s
	C.ui_set_owner(c.dpy, c.helper, c.atomClipboard)
	C.ui_set_owner(c.dpy, c.helper, c.atomPrimary)
	c.ownClip = C.ui_owner(c.dpy, c.atomClipboard) == c.helper
	c.ownPrim = C.ui_owner(c.dpy, c.atomPrimary) == c.helper
	c.keep = c.ownClip || c.ownPrim
	C.ui_flush(c.dpy)
}

func (c *x11Conn) readSelection(sel C.Atom, timeout time.Duration) (string, bool) {
	x11Mu.Lock()
	if c.dpy == nil || c.helper == 0 {
		x11Mu.Unlock()
		return "", false
	}
	if sel == c.atomClipboard && c.ownClip {
		s := c.clipText
		x11Mu.Unlock()
		return s, true
	}
	if sel == c.atomPrimary && c.ownPrim {
		s := c.clipText
		x11Mu.Unlock()
		return s, true
	}
	c.pasteDone = false
	c.pasteText = ""
	c.pasteWant = sel
	c.incrRecv = incrRecvState{}
	C.ui_convert(c.dpy, c.helper, sel, c.atomUTF8, c.atomProp)
	C.ui_flush(c.dpy)
	deadline := time.Now().Add(timeout)
	for !c.pasteDone && time.Now().Before(deadline) {
		c.drainLocked()
		if c.pasteDone {
			break
		}
		x11Mu.Unlock()
		C.ui_wait(c.dpy, 20)
		x11Mu.Lock()
		if c.dpy == nil {
			x11Mu.Unlock()
			return "", false
		}
	}
	if !c.pasteDone && c.dpy != nil && !c.incrRecv.active {
		c.pasteDone = false
		C.ui_convert(c.dpy, c.helper, sel, c.atomString, c.atomProp)
		C.ui_flush(c.dpy)
		end := time.Now().Add(timeout / 2)
		for !c.pasteDone && time.Now().Before(end) {
			c.drainLocked()
			if c.pasteDone {
				break
			}
			x11Mu.Unlock()
			C.ui_wait(c.dpy, 20)
			x11Mu.Lock()
			if c.dpy == nil {
				x11Mu.Unlock()
				return "", false
			}
		}
	}
	s := c.pasteText
	ok := c.pasteDone
	x11Mu.Unlock()
	return s, ok
}

func x11ClipSet(s string) {
	c, err := x11Get()
	if err != nil {
		return
	}
	c.setClipboard(s)
}

func x11ClipGet(primary bool) (string, bool) {
	c, err := x11Get()
	if err != nil {
		return "", false
	}
	sel := c.atomClipboard
	if primary {
		sel = c.atomPrimary
	}
	timeout := 250 * time.Millisecond
	if INCRThreshold(c.maxReq) < 1024 {
		timeout = 2 * time.Second
	}
	return c.readSelection(sel, timeout)
}

func x11Live() bool {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return x11c != nil && x11c.dpy != nil
}

func (s *x11Surface) SetFullscreen(on bool) {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	action := C.long(0)
	if on {
		action = 1
	}
	x11Mu.Lock()
	C.ui_ewmh_state(s.conn.dpy, s.win, action, s.conn.atomFullscr, 0)
	x11Mu.Unlock()
}

func (s *x11Surface) SetMaximized(on bool) {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	action := C.long(0)
	if on {
		action = 1
	}
	x11Mu.Lock()
	C.ui_ewmh_state(s.conn.dpy, s.win, action, s.conn.atomMaxVert, s.conn.atomMaxHorz)
	x11Mu.Unlock()
}

func (s *x11Surface) SetIMECursor(x, y, w, h int) {
	_ = w
	if s.ic == nil {
		return
	}
	if x == s.imeSpotX && y == s.imeSpotY {
		return
	}
	s.imeSpotX, s.imeSpotY = x, y
	spotY := y + h
	if spotY < y {
		spotY = y
	}
	x11Mu.Lock()
	C.ui_set_spot(s.ic, C.int(x), C.int(spotY))
	x11Mu.Unlock()
}
