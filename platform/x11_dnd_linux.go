//go:build linux && cgo

package platform

// XDND on X11: both halves. A uitoolkit window takes drops from any XDND
// client (a file manager, a browser), and a drag started in one is
// carried out to any other.
//
// The protocol itself — the client messages, the version both sides
// settle on, the action atoms and the search for the window under the
// pointer — is in xdnd.go, where it can be tested without a display.
// What is here is the Xlib that carries it: the properties, the
// selection transfer, the pointer and keyboard grab a source holds, and
// the override-redirect window the drag's picture lives in.
//
// cgo compiles every Go file's preamble on its own and a static function
// belongs to the file that declares it, so the helpers below are this
// file's own even where x11_linux.go has one of the same shape. They are
// named uix_* to say so.

/*
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>
#include <X11/extensions/shape.h>
#include <X11/extensions/Xrender.h>
#include <poll.h>
#include <stdlib.h>
#include <string.h>

// ---- events -------------------------------------------------------------------

static int ui_x_type(XEvent* e) { return e->type; }
static Window ui_x_window(XEvent* e) { return e->xany.window; }
static int ui_x_root_x(XEvent* e) { return e->xbutton.x_root; }
static int ui_x_root_y(XEvent* e) { return e->xbutton.y_root; }
static KeySym ui_x_keysym(Display* d, XEvent* e) {
	return XkbKeycodeToKeysym(d, (KeyCode)e->xkey.keycode, 0, 0);
}

static Atom ui_x_cm_type(XEvent* e) { return e->xclient.message_type; }
static int ui_x_cm_format(XEvent* e) { return e->xclient.format; }
static long ui_x_cm_data(XEvent* e, int i) { return e->xclient.data.l[i]; }

static Window ui_x_sr_requestor(XEvent* e) { return e->xselectionrequest.requestor; }
static Atom ui_x_sr_selection(XEvent* e) { return e->xselectionrequest.selection; }
static Atom ui_x_sr_target(XEvent* e) { return e->xselectionrequest.target; }
static Atom ui_x_sr_property(XEvent* e) { return e->xselectionrequest.property; }
static Atom ui_x_sn_property(XEvent* e) { return e->xselection.property; }
static Atom ui_x_sn_selection(XEvent* e) { return e->xselection.selection; }
static Window ui_x_sn_requestor(XEvent* e) { return e->xselection.requestor; }

// ui_x_sel_notify answers a SelectionRequest: prop is the property the
// data was put in, or None to refuse.
static void ui_x_sel_notify(Display* d, XEvent* req, Atom prop) {
	XEvent e;
	memset(&e, 0, sizeof(e));
	e.xselection.type = SelectionNotify;
	e.xselection.display = d;
	e.xselection.requestor = req->xselectionrequest.requestor;
	e.xselection.selection = req->xselectionrequest.selection;
	e.xselection.target = req->xselectionrequest.target;
	e.xselection.property = prop;
	e.xselection.time = req->xselectionrequest.time;
	XSendEvent(d, e.xselection.requestor, False, NoEventMask, &e);
	XFlush(d);
}

// ui_x_send_client sends one XDND client message. dest is the window it
// goes to — the proxy, when the target set XdndProxy — and win the window
// it names, which is always the one under the pointer.
static void ui_x_send_client(Display* d, Window dest, Window win, Atom type,
		long l0, long l1, long l2, long l3, long l4) {
	XEvent e;
	memset(&e, 0, sizeof(e));
	e.xclient.type = ClientMessage;
	e.xclient.display = d;
	e.xclient.window = win;
	e.xclient.message_type = type;
	e.xclient.format = 32;
	e.xclient.data.l[0] = l0;
	e.xclient.data.l[1] = l1;
	e.xclient.data.l[2] = l2;
	e.xclient.data.l[3] = l3;
	e.xclient.data.l[4] = l4;
	XSendEvent(d, dest, False, NoEventMask, &e);
	XFlush(d);
}

static void ui_x_flush(Display* d) { if (d) XFlush(d); }
static void ui_x_free(void* p) { if (p) XFree(p); }

// ui_x_wait waits up to ms for something to read, as the main loop does.
static int ui_x_wait(Display* d, int ms) {
	struct pollfd pfd;
	if (!d) return -1;
	if (XPending(d)) return 1;
	pfd.fd = ConnectionNumber(d);
	pfd.events = POLLIN;
	pfd.revents = 0;
	return poll(&pfd, 1, ms);
}

// ui_x_wake pokes the connection so a loop blocked in poll comes back:
// a drag that ended between polls must not wait for the next input.
static void ui_x_wake(Display* d, Window helper) {
	if (!d || !helper) return;
	XChangeProperty(d, helper, XA_WM_NAME, XA_STRING, 8, PropModeReplace,
		(const unsigned char*)"d", 1);
	XFlush(d);
}

// ---- properties and selections -------------------------------------------------

static void ui_x_prop32(Display* d, Window w, Atom prop, Atom type, unsigned long* data, int n) {
	XChangeProperty(d, w, prop, type, 32, PropModeReplace, (unsigned char*)data, n);
}

static void ui_x_prop8(Display* d, Window w, Atom prop, Atom type, const char* data, int n) {
	XChangeProperty(d, w, prop, type, 8, PropModeReplace, (const unsigned char*)data, n);
}

static void ui_x_del_prop(Display* d, Window w, Atom prop) {
	if (d && w) XDeleteProperty(d, w, prop);
}

// ui_x_watch_props asks for PropertyNotify on w, which is how the
// incremental (INCR) handshake advances.
static void ui_x_watch_props(Display* d, Window w) {
	XWindowAttributes a;
	if (!d || !w) return;
	memset(&a, 0, sizeof(a));
	if (!XGetWindowAttributes(d, w, &a)) return;
	XSelectInput(d, w, a.your_event_mask | PropertyChangeMask);
}

// ui_x_prop_bytes reads a whole property as bytes and deletes it, which is
// what acknowledges one INCR piece. The type is reported so an INCR
// handshake can be told from real data.
static int ui_x_prop_bytes(Display* d, Window w, Atom prop, unsigned char** data,
		unsigned long* nitems, Atom* type) {
	Atom actual = None;
	int fmt = 0;
	unsigned long n = 0, rem = 0;
	unsigned char* p = NULL;
	*data = NULL;
	*nitems = 0;
	*type = None;
	if (!d || !w) return 0;
	if (XGetWindowProperty(d, w, prop, 0L, 0L, False, AnyPropertyType,
			&actual, &fmt, &n, &rem, &p) != Success) {
		return 0;
	}
	if (p) { XFree(p); p = NULL; }
	if (fmt == 0) {
		XDeleteProperty(d, w, prop);
		return 0;
	}
	*type = actual;
	if (actual == XInternAtom(d, "INCR", False)) {
		XDeleteProperty(d, w, prop);
		return fmt;
	}
	long words = (long)((rem + 3) / 4) + 1;
	if (XGetWindowProperty(d, w, prop, 0L, words, True, AnyPropertyType,
			&actual, &fmt, &n, &rem, &p) != Success || p == NULL) {
		if (p) XFree(p);
		XDeleteProperty(d, w, prop);
		return 0;
	}
	*type = actual;
	*nitems = n;
	*data = p;
	return fmt;
}

// ui_x_prop_atoms reads an ATOM[] property (XdndTypeList, XdndActionList).
static int ui_x_prop_atoms(Display* d, Window w, Atom prop, Atom* out, int max) {
	Atom type = None;
	int fmt = 0;
	unsigned long n = 0, rem = 0;
	unsigned char* data = NULL;
	int k = 0;
	if (!d || !w) return 0;
	if (XGetWindowProperty(d, w, prop, 0, max, False, XA_ATOM,
			&type, &fmt, &n, &rem, &data) != Success || !data) {
		return 0;
	}
	if (fmt == 32) {
		unsigned long i;
		for (i = 0; i < n && k < max; i++) out[k++] = ((Atom*)data)[i];
	}
	XFree(data);
	return k;
}

// ui_x_prop_window reads a property holding one window id (XdndProxy).
static Window ui_x_prop_window(Display* d, Window w, Atom prop) {
	Atom type = None;
	int fmt = 0;
	unsigned long n = 0, after = 0;
	unsigned char* data = NULL;
	Window out = None;
	if (!d || !w) return None;
	if (XGetWindowProperty(d, w, prop, 0, 1, False, AnyPropertyType,
			&type, &fmt, &n, &after, &data) != Success) return None;
	if (data) {
		if (fmt == 32 && n >= 1) out = (Window)(*(unsigned long*)data);
		XFree(data);
	}
	return out;
}

// ui_x_prop_card reads a property holding one 32-bit value (XdndAware's
// version); -1 when the property is not there.
static long ui_x_prop_card(Display* d, Window w, Atom prop) {
	Atom type = None;
	int fmt = 0;
	unsigned long n = 0, after = 0;
	unsigned char* data = NULL;
	long out = -1;
	if (!d || !w) return -1;
	if (XGetWindowProperty(d, w, prop, 0, 1, False, AnyPropertyType,
			&type, &fmt, &n, &after, &data) != Success) return -1;
	if (data) {
		if (fmt == 32 && n >= 1) out = (long)(*(unsigned long*)data);
		XFree(data);
	}
	return out;
}

static char* ui_x_atom_name(Display* d, Atom a) { return a ? XGetAtomName(d, a) : NULL; }
static Atom ui_x_atom(Display* d, const char* n) { return XInternAtom(d, n, False); }

static void ui_x_set_owner(Display* d, Window w, Atom sel, Time t) {
	XSetSelectionOwner(d, sel, w, t);
	XFlush(d);
}
static Window ui_x_owner(Display* d, Atom sel) { return XGetSelectionOwner(d, sel); }
static void ui_x_convert(Display* d, Window w, Atom sel, Atom target, Atom prop, Time t) {
	XConvertSelection(d, sel, target, prop, w, t);
}

// ---- the window tree, the grab and the pointer ---------------------------------

static Window ui_x_root(Display* d) { return DefaultRootWindow(d); }

// ui_x_query_tree is one level of the tree, children bottom-most first.
// The caller frees with ui_x_free_kids.
static int ui_x_query_tree(Display* d, Window w, Window** kids, unsigned int* n) {
	Window root, parent;
	*kids = NULL;
	*n = 0;
	if (!d || !XQueryTree(d, w, &root, &parent, kids, n)) return 0;
	return 1;
}
static Window ui_x_kid_at(Window* kids, int i) { return kids[i]; }
static void ui_x_free_kids(Window* kids) { if (kids) XFree(kids); }

// ui_x_win_geom is a window's box in its parent's coordinates. mapped is
// false only for a window that is not on screen: an InputOnly window is a
// perfectly good drop target, and the proxy a compositor puts up to
// bridge an X11 drag to a Wayland client is exactly that.
static int ui_x_win_geom(Display* d, Window w, int* x, int* y, int* ww, int* hh, int* mapped) {
	XWindowAttributes a;
	memset(&a, 0, sizeof(a));
	if (!d || !XGetWindowAttributes(d, w, &a)) return 0;
	*x = a.x;
	*y = a.y;
	*ww = a.width;
	*hh = a.height;
	*mapped = (a.map_state == IsViewable);
	return 1;
}

// ui_x_from_root translates a root-coordinate point into w's: XDND speaks
// in screen coordinates, the toolkit in the window's.
static int ui_x_from_root(Display* d, Window w, int rx, int ry, int* x, int* y) {
	Window child = 0;
	return XTranslateCoordinates(d, DefaultRootWindow(d), w, rx, ry, x, y, &child);
}

// ui_x_pointer is where the pointer is and which modifiers are held: on
// X11 it is the drag's source that turns Shift into a move, so the source
// has to know what the user is holding.
static int ui_x_pointer(Display* d, int* rx, int* ry, unsigned int* mask) {
	Window root = None, child = None;
	int wx = 0, wy = 0;
	*mask = 0;
	if (!d) return 0;
	return XQueryPointer(d, DefaultRootWindow(d), &root, &child, rx, ry, &wx, &wy, mask);
}
static unsigned int ui_x_mot_state(XEvent* e) { return e->xmotion.state; }

static int ui_x_grab_ptr(Display* d, Window w, Cursor cur, Time t) {
	return XGrabPointer(d, w, False,
		ButtonPressMask|ButtonReleaseMask|PointerMotionMask,
		GrabModeAsync, GrabModeAsync, None, cur, t);
}
static void ui_x_regrab_cursor(Display* d, Cursor cur, Time t) {
	XChangeActivePointerGrab(d,
		ButtonPressMask|ButtonReleaseMask|PointerMotionMask, cur, t);
	XFlush(d);
}
static void ui_x_ungrab_ptr(Display* d) { if (d) XUngrabPointer(d, CurrentTime); }
static int ui_x_grab_kbd(Display* d, Window w, Time t) {
	return XGrabKeyboard(d, w, False, GrabModeAsync, GrabModeAsync, t);
}
static void ui_x_ungrab_kbd(Display* d) { if (d) XUngrabKeyboard(d, CurrentTime); }

// ui_x_escape_held asks the server which keys are down, rather than
// waiting for a key event. A drag holds a keyboard grab, and there are
// setups where a grabbed key event never reaches the client at all
// (Xwayland hands the grab to the compositor); the key's state is still
// there to be read.
static int ui_x_escape_held(Display* d) {
	char keys[32];
	KeyCode kc;
	if (!d) return 0;
	kc = XKeysymToKeycode(d, XK_Escape);
	if (kc == 0) return 0;
	XQueryKeymap(d, keys);
	return (keys[kc >> 3] & (1 << (kc & 7))) != 0;
}

// ---- the drag's picture --------------------------------------------------------

// ui_x_argb finds a 32-bit visual with an alpha channel, and a colormap
// for it: what a translucent drag icon must be created with. Without one
// the drag runs with the cursor alone.
static int ui_x_argb(Display* d, Visual** vis, Colormap* cmap, int* depth) {
	XVisualInfo tmpl, *found;
	Visual* pick = NULL;
	int n = 0, i;
	if (!d) return 0;
	memset(&tmpl, 0, sizeof(tmpl));
	tmpl.screen = DefaultScreen(d);
	tmpl.depth = 32;
	tmpl.class = TrueColor;
	found = XGetVisualInfo(d, VisualScreenMask|VisualDepthMask|VisualClassMask, &tmpl, &n);
	if (!found || n < 1) {
		if (found) XFree(found);
		return 0;
	}
	for (i = 0; i < n; i++) {
		XRenderPictFormat* f = XRenderFindVisualFormat(d, found[i].visual);
		if (f && f->type == PictTypeDirect && f->direct.alphaMask) {
			pick = found[i].visual;
			*depth = found[i].depth;
			break;
		}
	}
	XFree(found);
	if (!pick) return 0;
	*vis = pick;
	*cmap = XCreateColormap(d, DefaultRootWindow(d), pick, AllocNone);
	return 1;
}

// ui_x_icon_win is the window a drag's picture lives in: override-redirect
// so no window manager touches it, on the ARGB visual so it is
// translucent, and with an empty input shape so it never takes the
// pointer from whatever is under it.
static Window ui_x_icon_win(Display* d, Visual* vis, Colormap cmap, int depth, int w, int h) {
	XSetWindowAttributes swa;
	XRectangle empty;
	Atom wtype, dnd;
	Window win;
	memset(&swa, 0, sizeof(swa));
	swa.override_redirect = True;
	swa.background_pixel = 0;
	swa.border_pixel = 0;
	swa.colormap = cmap;
	swa.event_mask = ExposureMask;
	win = XCreateWindow(d, DefaultRootWindow(d), -4096, -4096,
		(unsigned)w, (unsigned)h, 0, depth, InputOutput, vis,
		CWOverrideRedirect|CWBackPixel|CWBorderPixel|CWColormap|CWEventMask, &swa);
	if (!win) return 0;
	empty.x = 0; empty.y = 0; empty.width = 0; empty.height = 0;
	XShapeCombineRectangles(d, win, ShapeInput, 0, 0, &empty, 1, ShapeSet, Unsorted);
	wtype = XInternAtom(d, "_NET_WM_WINDOW_TYPE", False);
	dnd = XInternAtom(d, "_NET_WM_WINDOW_TYPE_DND", False);
	XChangeProperty(d, win, wtype, XA_ATOM, 32, PropModeReplace, (unsigned char*)&dnd, 1);
	return win;
}

static XImage* ui_x_icon_image(Display* d, Visual* vis, int depth, int w, int h, char* data) {
	return XCreateImage(d, vis, (unsigned)depth, ZPixmap, 0, data,
		(unsigned)w, (unsigned)h, 32, 0);
}

static GC ui_x_gc(Display* d, Window w) { return XCreateGC(d, w, 0, NULL); }

static void ui_x_put(Display* d, Window w, GC gc, XImage* img, int width, int height) {
	if (d && w && gc && img) XPutImage(d, w, gc, img, 0, 0, 0, 0, (unsigned)width, (unsigned)height);
}

static void ui_x_map(Display* d, Window w) { if (d && w) { XMapRaised(d, w); XFlush(d); } }
static void ui_x_move(Display* d, Window w, int x, int y) { if (d && w) XMoveWindow(d, w, x, y); }

static void ui_x_destroy_win(Display* d, Window w, GC gc) {
	if (!d) return;
	if (gc) XFreeGC(d, gc);
	if (w) XDestroyWindow(d, w);
	XFlush(d);
}

// ui_x_destroy_image drops the pixel pointer first: the bytes belong to
// the caller, not to XDestroyImage.
static void ui_x_destroy_image(XImage* img) {
	if (img) {
		img->data = NULL;
		XDestroyImage(img);
	}
}

static void ui_x_free_colormap(Display* d, Colormap c) { if (d && c) XFreeColormap(d, c); }
*/
import "C"

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// xdndDebug traces a drag's progress to stderr (UITK_XDND_DEBUG=1). A
// drag is a conversation with another application, and when one goes
// wrong there is nothing on screen to say which half stopped talking.
var xdndDebug = os.Getenv("UITK_XDND_DEBUG") == "1"

func xdndTrace(format string, args ...any) {
	if xdndDebug {
		fmt.Fprintf(os.Stderr, "uitk-xdnd: "+format+"\n", args...)
	}
}

// xdndAtoms are the interned XDND atoms, plus the two properties this
// implementation keeps for itself: one a drop's data arrives in, one a
// drag's data is served from.
type xdndAtoms struct {
	a [xdndAtomCount]C.Atom
	// dropProp is where a converted XdndSelection lands on our window.
	dropProp C.Atom
	actions  XDNDActions
	ready    bool
}

func (x *xdndAtoms) at(i XDNDAtom) C.Atom { return x.a[i] }

// internXdndLocked interns every XDND atom once per connection.
func (c *x11Conn) internXdndLocked() {
	if c.xdnd.ready || c.dpy == nil {
		return
	}
	for i, name := range XDNDAtomNames {
		cs := C.CString(name)
		c.xdnd.a[i] = C.XInternAtom(c.dpy, cs, C.False)
		C.free(unsafe.Pointer(cs))
	}
	c.xdnd.dropProp = internAtom(c.dpy, "UITK_XDND_DROP")
	c.xdnd.actions = XDNDActions{
		Copy:    uint32(c.xdnd.at(XAActionCopy)),
		Move:    uint32(c.xdnd.at(XAActionMove)),
		Link:    uint32(c.xdnd.at(XAActionLink)),
		Ask:     uint32(c.xdnd.at(XAActionAsk)),
		Private: uint32(c.xdnd.at(XAActionPrivate)),
	}
	c.xdnd.ready = true
}

// atomName is an atom's name, which is how a type reaches the toolkit:
// XDND names types by atom, and everything above the backend speaks MIME.
func (c *x11Conn) atomName(a C.Atom) string {
	if a == 0 || c.dpy == nil {
		return ""
	}
	p := C.ui_x_atom_name(c.dpy, a)
	if p == nil {
		return ""
	}
	s := C.GoString(p)
	C.ui_x_free(unsafe.Pointer(p))
	return s
}

func (c *x11Conn) atomFor(name string) C.Atom {
	if name == "" || c.dpy == nil {
		return 0
	}
	return internAtom(c.dpy, name)
}

// ---- the target half: taking a drop -------------------------------------------

// x11Drop is the drag another client has over one of our windows.
type x11Drop struct {
	active bool
	// source is the window that started the drag, send the window its
	// messages go to (the source itself), version the protocol both
	// sides settled on.
	source  C.Window
	version int
	// win is our window the drag is over; types what it offers.
	win   C.Window
	types []string
	// posPending is an XdndPosition waiting for the answer the app owes
	// it; pos is where it was, in our window's coordinates.
	posPending bool
	pos        paintengine2d.Point
	// accept is the type the app said it would read the drop as ("": it
	// takes nothing here) and action what it would do.
	accept string
	action DragAction
	// offered is what the source allows, prefer the one it asked for.
	offered DragAction
	prefer  DragAction
	// dropped is set between XdndDrop and XdndFinished; when is the
	// timestamp the data must be converted at.
	dropped bool
	when    C.Time
	// recv is the selection transfer of the dropped data.
	recv xdndRecv
}

// xdndRecv is one conversion of XdndSelection, INCR included.
type xdndRecv struct {
	waiting bool
	done    bool
	incr    bool
	data    []byte
}

// setXdndAwareLocked advertises that a window takes XDND drops. It goes on
// the toplevel window, which is where version 3 moved it.
func (s *x11Surface) setXdndAwareLocked() {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return
	}
	c.internXdndLocked()
	v := []C.ulong{C.ulong(XDNDVersion)}
	C.ui_x_prop32(c.dpy, s.win, c.xdnd.at(XAAware), C.XA_ATOM, &v[0], 1)
}

// handleXdndMessage takes the XDND client messages, both the ones a drag
// over us sends and the ones a drag of ours gets back. It reports whether
// the message was one of ours.
func (c *x11Conn) handleXdndMessage(xe *C.XEvent) bool {
	if c.dpy == nil || C.ui_x_cm_format(xe) != 32 {
		return false
	}
	c.internXdndLocked()
	typ := C.ui_x_cm_type(xe)
	var m XDNDMessage
	for i := 0; i < 5; i++ {
		m[i] = uint32(C.ui_x_cm_data(xe, C.int(i)))
	}
	switch typ {
	case c.xdnd.at(XAEnter):
		c.xdndEnter(C.ui_x_window(xe), m)
	case c.xdnd.at(XAPosition):
		c.xdndPosition(C.ui_x_window(xe), m)
	case c.xdnd.at(XALeave):
		c.xdndLeave(m)
	case c.xdnd.at(XADrop):
		c.xdndDrop(m)
	case c.xdnd.at(XAStatus):
		c.dragStatus(m)
	case c.xdnd.at(XAFinished):
		c.dragFinished(m)
	default:
		return false
	}
	return true
}

func (c *x11Conn) xdndEnter(win C.Window, m XDNDMessage) {
	s := c.surfaces[win]
	if s == nil {
		return
	}
	c.xdndEnd(false)
	source, version, more, types := DecodeXdndEnter(m)
	v, ok := XDNDNegotiateVersion(XDNDVersion, version)
	if !ok {
		return
	}
	names := make([]string, 0, len(types)+4)
	if more {
		// More than three types: the whole list is a property on the
		// source window.
		var buf [64]C.Atom
		n := C.ui_x_prop_atoms(c.dpy, C.Window(source), c.xdnd.at(XATypeList), &buf[0], C.int(len(buf)))
		for i := 0; i < int(n); i++ {
			if name := c.atomName(buf[i]); name != "" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		for _, t := range types {
			if name := c.atomName(C.Atom(t)); name != "" {
				names = append(names, name)
			}
		}
	}
	c.drop = x11Drop{
		active:  true,
		source:  C.Window(source),
		version: v,
		win:     win,
		types:   names,
		offered: DragCopy,
	}
}

func (c *x11Conn) xdndPosition(win C.Window, m XDNDMessage) {
	if !c.drop.active || c.drop.win != win {
		return
	}
	s := c.surfaces[win]
	if s == nil {
		return
	}
	// An earlier position the app never answered: answer it now with
	// what we last knew, so the source is never left waiting.
	if c.drop.posPending {
		c.sendXdndStatus()
	}
	source, rx, ry, when, action := DecodeXdndPosition(m)
	if C.Window(source) != c.drop.source {
		return
	}
	if when != 0 {
		c.drop.when = C.Time(when)
	}
	c.drop.prefer = c.xdnd.actions.Action(action)
	// XDND names one action per position; a source that would allow more
	// says so in XdndActionList. XdndActionAsk is not one of them — it is
	// the source asking for the user to pick out of that list — so it
	// stays on the requested action alone and never in the offered set.
	c.drop.offered = c.drop.prefer.Actions()
	var list [8]C.Atom
	if n := C.ui_x_prop_atoms(c.dpy, c.drop.source, c.xdnd.at(XAActionList), &list[0], C.int(len(list))); n > 0 {
		var all DragAction
		for i := 0; i < int(n); i++ {
			all |= c.xdnd.actions.Action(uint32(list[i]))
		}
		if all = all.Actions(); all != DragNone {
			c.drop.offered = all
		}
	}
	if c.drop.offered == DragNone {
		c.drop.offered = DragCopy
	}
	// Root coordinates to the window's: the app knows nothing of the
	// screen, and the frame's margin is part of our window.
	var wx, wy C.int
	C.ui_x_from_root(c.dpy, win, C.int(rx), C.int(ry), &wx, &wy)
	c.drop.pos = paintengine2d.Pt(float32(wx), float32(wy))
	c.drop.posPending = true
	c.queues[win] = append(c.queues[win], Event{
		Kind:    EventDragMotion,
		Pos:     c.drop.pos,
		Mimes:   c.drop.types,
		Actions: c.drop.offered,
		Action:  c.drop.prefer,
	})
}

func (c *x11Conn) xdndLeave(m XDNDMessage) {
	if !c.drop.active || C.Window(DecodeXdndLeave(m)) != c.drop.source {
		return
	}
	if s := c.surfaces[c.drop.win]; s != nil {
		c.queues[s.win] = append(c.queues[s.win], Event{Kind: EventDragLeave})
	}
	c.xdndEnd(false)
}

func (c *x11Conn) xdndDrop(m XDNDMessage) {
	if !c.drop.active {
		return
	}
	source, when := DecodeXdndDrop(m)
	if C.Window(source) != c.drop.source {
		return
	}
	if when != 0 {
		c.drop.when = C.Time(when)
	}
	c.drop.dropped = true
	c.drop.posPending = false
	if s := c.surfaces[c.drop.win]; s != nil {
		c.queues[s.win] = append(c.queues[s.win], Event{
			Kind:    EventDrop,
			Pos:     c.drop.pos,
			Mimes:   c.drop.types,
			Actions: c.drop.offered,
			Action:  c.drop.prefer,
		})
	}
}

// sendXdndStatus answers the position the app owes an answer for. Every
// XdndPosition must be answered: a source that gets no XdndStatus shows
// the user a drag that has stopped responding.
func (c *x11Conn) sendXdndStatus() {
	if !c.drop.active || !c.drop.posPending {
		return
	}
	c.drop.posPending = false
	accept := c.drop.accept != "" && c.drop.action != DragNone
	// An empty rectangle asks the source to keep sending positions: our
	// targets are widgets, far smaller than the window, so the answer
	// changes as the pointer moves inside it.
	m := EncodeXdndStatus(uint32(c.drop.win), accept, XDNDRect{},
		c.xdnd.actions.Atom(c.drop.action))
	c.sendXdnd(c.drop.source, c.drop.source, XAStatus, m)
}

// sendXdnd sends one message: dest is the window it goes to, win the one
// it names.
func (c *x11Conn) sendXdnd(dest, win C.Window, which XDNDAtom, m XDNDMessage) {
	if c.dpy == nil || dest == 0 {
		return
	}
	C.ui_x_send_client(c.dpy, dest, win, c.xdnd.at(which),
		C.long(int32(m[0])), C.long(int32(m[1])), C.long(int32(m[2])),
		C.long(int32(m[3])), C.long(int32(m[4])))
}

// xdndEnd forgets the drag over us, telling the source it is done when the
// drop was completed.
func (c *x11Conn) xdndEnd(finished bool) {
	if !c.drop.active {
		return
	}
	if finished && c.drop.dropped {
		accept := c.drop.action != DragNone
		m := EncodeXdndFinished(uint32(c.drop.win), accept, c.xdnd.actions.Atom(c.drop.action))
		c.sendXdnd(c.drop.source, c.drop.source, XAFinished, m)
	}
	c.drop = x11Drop{}
}

// AcceptDrag implements [DropNegotiator]: the app says what it would do
// with the drag over the window, and the source is told at once.
// allowed is not sent: XdndStatus carries exactly one action, and on X11
// it is the source that picks it from the user's modifiers.
func (s *x11Surface) AcceptDrag(mime string, allowed, a DragAction) {
	_ = allowed
	x11Mu.Lock()
	defer x11Mu.Unlock()
	c := s.conn
	if c == nil || !c.drop.active || c.drop.win != s.win {
		return
	}
	c.drop.accept, c.drop.action = mime, a
	if mime == "" {
		c.drop.action = DragNone
	}
	c.sendXdndStatus()
}

// ReceiveDrop implements [DropReceiver]: it converts XdndSelection to the
// type the target asked for, at the timestamp the drop named — which is
// what keeps a second drag started before this one finished from handing
// over the wrong data.
func (s *x11Surface) ReceiveDrop(mime string) ([]byte, bool) {
	x11Mu.Lock()
	c := s.conn
	if c == nil || c.dpy == nil || !c.drop.active || !c.drop.dropped || mime == "" {
		x11Mu.Unlock()
		return nil, false
	}
	target := c.atomFor(mime)
	if target == 0 {
		x11Mu.Unlock()
		return nil, false
	}
	c.drop.recv = xdndRecv{waiting: true}
	C.ui_x_watch_props(c.dpy, s.win)
	C.ui_x_del_prop(c.dpy, s.win, c.xdnd.dropProp)
	when := c.drop.when
	if when == 0 {
		when = c.selectionTimeLocked()
	}
	C.ui_x_convert(c.dpy, s.win, c.xdnd.at(XASelection), target, c.xdnd.dropProp, when)
	C.ui_x_flush(c.dpy)
	// A transfer that is being sent in INCR pieces takes as long as the
	// other side needs; a plain one is a single round trip.
	deadline := time.Now().Add(xdndTransferTimeout)
	for !c.drop.recv.done && time.Now().Before(deadline) {
		c.drainLocked()
		if c.drop.recv.done {
			break
		}
		if c.drop.recv.incr {
			deadline = time.Now().Add(xdndTransferTimeout)
		}
		dpy := c.dpy
		x11Mu.Unlock()
		C.ui_x_wait(dpy, 5)
		x11Mu.Lock()
		if c.dpy == nil || !c.drop.active {
			x11Mu.Unlock()
			return nil, false
		}
	}
	data, ok := c.drop.recv.data, c.drop.recv.done
	c.drop.recv = xdndRecv{}
	x11Mu.Unlock()
	return data, ok
}

// xdndTransferTimeout is how long a drop's data may take to arrive before
// the drop is given up on. An INCR transfer restarts it on every piece,
// so this bounds a silent peer, not a big file.
const xdndTransferTimeout = 3 * time.Second

// FinishDrop implements [DropReceiver]: the source is told the drop is
// complete and may let go of the data.
func (s *x11Surface) FinishDrop(ok bool) {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if s.conn == nil {
		return
	}
	if !ok {
		s.conn.drop.action = DragNone
	}
	s.conn.xdndEnd(true)
}

// xdndSelNotify takes the answer to our conversion of XdndSelection.
func (c *x11Conn) xdndSelNotify(xe *C.XEvent) bool {
	if !c.drop.recv.waiting || C.ui_x_sn_selection(xe) != c.xdnd.at(XASelection) {
		return false
	}
	prop := C.ui_x_sn_property(xe)
	if prop == 0 {
		// The source could not convert to that type.
		c.drop.recv.done, c.drop.recv.waiting = true, false
		c.drop.recv.data = nil
		return true
	}
	if prop != c.xdnd.dropProp {
		return false
	}
	win := C.ui_x_sn_requestor(xe)
	var data *C.uchar
	var n C.ulong
	var typ C.Atom
	fmtb := C.ui_x_prop_bytes(c.dpy, win, prop, &data, &n, &typ)
	if typ == c.atomINCR {
		if data != nil {
			C.ui_x_free(unsafe.Pointer(data))
		}
		// The data comes in pieces, each one a property the sender
		// writes after we delete the last.
		c.drop.recv.incr = true
		c.incrRecv = incrRecvState{active: true, win: win, prop: prop}
		C.ui_x_del_prop(c.dpy, win, prop)
		C.ui_x_flush(c.dpy)
		return true
	}
	if data != nil {
		if fmtb == 8 && n > 0 {
			c.drop.recv.data = C.GoBytes(unsafe.Pointer(data), C.int(n))
		}
		C.ui_x_free(unsafe.Pointer(data))
	}
	c.drop.recv.done, c.drop.recv.waiting = true, false
	C.ui_x_del_prop(c.dpy, win, prop)
	return true
}

// xdndIncrPiece takes one INCR piece of a drop's data. An empty piece
// ends the transfer.
func (c *x11Conn) xdndIncrPiece(piece []byte) bool {
	if !c.drop.recv.waiting || !c.drop.recv.incr {
		return false
	}
	var done bool
	c.drop.recv.data, done = AppendINCRPiece(c.drop.recv.data, piece)
	if done {
		c.drop.recv.done, c.drop.recv.waiting, c.drop.recv.incr = true, false, false
		c.incrRecv = incrRecvState{}
	}
	return true
}

// ---- the source half: dragging out --------------------------------------------

// x11Drag is a drag started in one of our windows. It owns the pointer
// and the keyboard until it ends, so every motion, the release and
// Escape come to us wherever the pointer goes.
type x11Drag struct {
	active  bool
	payload DragPayload
	// win is the window the drag started in — the one XDND names as the
	// source, and the one XdndSelection is owned by and XdndTypeList
	// read from.
	win C.Window
	// target is the window under the pointer now, its proxy and the
	// version we speak to it.
	target XDNDTarget
	// accepted and action are its last XdndStatus.
	accepted bool
	action   DragAction
	// icon is the picture that follows the pointer.
	icon xdndIcon
	// attach is a window this drag carries — a tab torn out of its
	// strip, a floating panel on its way back — with hx, hy where in it
	// the pointer sits. Wayland has a compositor move it
	// (xdg-toplevel-drag-v1); on X11 a client places its own windows, so
	// the drag moves it itself on every motion. It is left out of the
	// search for a target: a window carried by the drag is not something
	// the drag can be dropped on.
	attach  C.Window
	attachH struct{ x, y int }
	// sends are the INCR transfers of our data still in flight; ended
	// stops a drag that has already reported its result.
	ended bool
	// dropped is set from XdndDrop until XdndFinished; a source must
	// keep serving the data until then. released is the user letting the
	// button go, drop or no drop, which is what tells a drop on the
	// desktop from a drag called off with Escape.
	dropped  bool
	released bool
	dropAt   time.Time
	cursor   Cursor
	// mods are the modifier keys held at the last motion: on X11 the
	// source, not the compositor, turns Shift into a move.
	mods uint
}

// xdndIcon is the override-redirect window a drag's picture lives in.
type xdndIcon struct {
	win  C.Window
	gc   C.GC
	img  *C.XImage
	mem  unsafe.Pointer
	w, h int
	hx   int
	hy   int
}

// StartDrag implements [DragSurface]. It takes XdndSelection, grabs the
// pointer and the keyboard, and puts the drag's picture on screen; from
// then on the drag runs in the event loop until it is dropped or
// cancelled.
func (s *x11Surface) StartDrag(p DragPayload) bool {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 || c.drag.active || len(p.Types) == 0 {
		return false
	}
	c.internXdndLocked()
	if p.Actions == DragNone {
		p.Actions = DragCopy
	}
	when := c.selectionTimeLocked()
	C.ui_x_set_owner(c.dpy, s.win, c.xdnd.at(XASelection), when)
	if C.ui_x_owner(c.dpy, c.xdnd.at(XASelection)) != s.win {
		return false
	}
	// Every type goes in XdndTypeList: only three fit in XdndEnter, and
	// a target that wants the rest reads them from here.
	atoms := make([]C.ulong, 0, len(p.Types))
	for _, t := range p.Types {
		if a := c.atomFor(t); a != 0 {
			atoms = append(atoms, C.ulong(a))
		}
	}
	if len(atoms) == 0 {
		return false
	}
	C.ui_x_prop32(c.dpy, s.win, c.xdnd.at(XATypeList), C.XA_ATOM, &atoms[0], C.int(len(atoms)))
	acts := c.xdnd.actions.List(p.Actions, p.Preferred)
	if len(acts) > 0 {
		list := make([]C.ulong, len(acts))
		for i, a := range acts {
			list[i] = C.ulong(a)
		}
		C.ui_x_prop32(c.dpy, s.win, c.xdnd.at(XAActionList), C.XA_ATOM, &list[0], C.int(len(list)))
	}
	c.drag = x11Drag{active: true, payload: p, win: s.win, cursor: CursorNoDrop}
	xdndTrace("drag started, offering %v", p.Types)
	c.keep = true
	if r := C.ui_x_grab_ptr(c.dpy, s.win, c.xcursor(CursorNoDrop), when); r != C.GrabSuccess {
		// Without the pointer there is no drag: something else has it.
		xdndTrace("pointer grab refused: %d", int(r))
		c.dragEnd(DragNone)
		return false
	}
	// The keyboard too, so Escape reaches us wherever the pointer is.
	if r := C.ui_x_grab_kbd(c.dpy, s.win, when); r != C.GrabSuccess {
		// Not fatal: the drag runs on the pointer grab alone, and
		// dragCheckEscape still calls it off.
		xdndTrace("keyboard grab refused: %d", int(r))
	}
	c.dragMakeIcon(s, p)
	c.dragMotionTo(rootPointer(c))

	C.ui_x_flush(c.dpy)
	return true
}

// CancelDrag implements [DragSurface].
func (s *x11Surface) CancelDrag() {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if s.conn != nil {
		s.conn.dragCancel()
	}
}

// Dragging implements [DragSurface].
func (s *x11Surface) Dragging() bool {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return s.conn != nil && s.conn.drag.active
}

// DragsToplevels implements [ToplevelDragSurface]. An X11 client places
// its own windows, so a drag here can always carry one — no protocol
// needed, and no window manager has to help.
func (s *x11Surface) DragsToplevels() bool { return true }

// AttachToplevel implements [ToplevelDragSurface]: win follows the
// pointer for the rest of the drag, dx, dy inside it under the pointer.
func (s *x11Surface) AttachToplevel(win Surface, dx, dy int) bool {
	x11Mu.Lock()
	defer x11Mu.Unlock()
	c := s.conn
	if c == nil || !c.drag.active {
		return false
	}
	w, ok := win.(*x11Surface)
	if !ok || w.conn != c || w.win == 0 || w.closed {
		return false
	}
	c.drag.attach = w.win
	// The window holds the frame's margin too, and the offset is inside
	// the window the user sees: the margin puts the two apart.
	c.drag.attachH.x, c.drag.attachH.y = dx+w.frame.Margin.Left, dy+w.frame.Margin.Top
	rx, ry, _ := rootPointer(c)
	c.dragMoveAttached(rx, ry)
	C.ui_x_flush(c.dpy)
	return true
}

// dragMoveAttached puts the window the drag carries under the pointer.
func (c *x11Conn) dragMoveAttached(rx, ry int) {
	if c.drag.attach == 0 || c.dpy == nil {
		return
	}
	C.ui_x_move(c.dpy, c.drag.attach, C.int(rx-c.drag.attachH.x), C.int(ry-c.drag.attachH.y))
}

// rootPointer is where the pointer is on the screen now, and which
// modifiers are held there.
func rootPointer(c *x11Conn) (int, int, uint) {
	var rx, ry C.int
	var mask C.uint
	if c.dpy == nil {
		return 0, 0, 0
	}
	C.ui_x_pointer(c.dpy, &rx, &ry, &mask)
	return int(rx), int(ry), uint(mask)
}

// dragHandle takes the events a running drag owns: the pointer motion and
// release its grab delivers, Escape on its keyboard grab, and the icon
// window's exposures. It reports whether the event was the drag's.
func (c *x11Conn) dragHandle(xe *C.XEvent) bool {
	if !c.drag.active {
		return false
	}
	switch C.ui_x_type(xe) {
	case C.MotionNotify:
		c.dragMotionTo(int(C.ui_x_root_x(xe)), int(C.ui_x_root_y(xe)), uint(C.ui_x_mot_state(xe)))
		return true
	case C.ButtonRelease:
		c.dragRelease()
		return true
	case C.ButtonPress:
		// A second button during a drag cancels it, as it does
		// everywhere else.
		c.dragCancel()
		return true
	case C.KeyPress:
		if xkey(C.ui_x_keysym(c.dpy, xe)) == KeyEscape {
			c.dragCancel()
		}
		return true
	case C.KeyRelease:
		return true
	case C.Expose:
		if C.ui_x_window(xe) == c.drag.icon.win && c.drag.icon.win != 0 {
			c.dragPaintIcon()
			return true
		}
	}
	return false
}

// dragMotionTo follows the pointer: it finds the window under it, tells
// the one it left that the drag is gone and the one it entered that it
// has arrived, and asks for a position either way.
func (c *x11Conn) dragMotionTo(rx, ry int, mods uint) {
	if !c.drag.active || c.dpy == nil {
		return
	}
	c.drag.mods = mods
	c.dragMoveIcon(rx, ry)
	c.dragMoveAttached(rx, ry)
	tree := x11Tree{c: c}
	// The picture and the window the drag carries are both under the
	// pointer and neither is a target: the drop goes to whatever they are
	// over.
	skip := func(w uint32) bool {
		return (c.drag.icon.win != 0 && C.Window(w) == c.drag.icon.win) ||
			(c.drag.attach != 0 && C.Window(w) == c.drag.attach)
	}
	found := XDNDFindTarget(tree, uint32(C.ui_x_root(c.dpy)), rx, ry, skip)
	if found.Window != c.drag.target.Window {
		xdndTrace("target at %d,%d: window=%#x proxy=%#x version=%d",
			rx, ry, found.Window, found.Proxy, found.Version)
		c.dragLeaveTarget()
		c.drag.target = found
		if found.Valid() {
			var atoms []uint32
			for _, t := range c.drag.payload.Types {
				if a := c.atomFor(t); a != 0 {
					atoms = append(atoms, uint32(a))
				}
			}
			c.sendXdnd(C.Window(found.Send()), C.Window(found.Window), XAEnter,
				EncodeXdndEnter(uint32(c.drag.win), found.Version, atoms))
		}
	}
	if !c.drag.target.Valid() {
		c.dragCursor(CursorNoDrop)
		return
	}
	// The action the user is asking for with the modifiers they hold;
	// XDND puts exactly one in every position message.
	action := ModifierDragAction(xmods(mods), c.drag.payload.Actions, c.drag.payload.Preferred)
	c.sendXdnd(C.Window(c.drag.target.Send()), C.Window(c.drag.target.Window), XAPosition,
		EncodeXdndPosition(uint32(c.drag.win), rx, ry, uint32(c.serverTime),
			c.xdnd.actions.Atom(action)))
}

// dragLeaveTarget withdraws the drag from the window it was over.
func (c *x11Conn) dragLeaveTarget() {
	if !c.drag.target.Valid() {
		return
	}
	c.sendXdnd(C.Window(c.drag.target.Send()), C.Window(c.drag.target.Window), XALeave,
		EncodeXdndLeave(uint32(c.drag.win)))
	c.drag.target = XDNDTarget{}
	c.drag.accepted, c.drag.action = false, DragNone
}

// dragStatus is a target's answer: whether it takes a drop where the
// pointer is and what it would do, which the cursor then shows.
func (c *x11Conn) dragStatus(m XDNDMessage) {
	if !c.drag.active {
		return
	}
	target, accept, _, _, action := DecodeXdndStatus(m)
	if !c.drag.target.Valid() || C.Window(target) != C.Window(c.drag.target.Window) {
		// A status from a window the drag has already left.
		return
	}
	c.drag.accepted = accept
	c.drag.action = DragNone
	if accept {
		c.drag.action = c.xdnd.actions.Action(action)
	}
	if !accept {
		c.dragCursor(CursorNoDrop)
		return
	}
	c.dragCursor(DragCursor(c.drag.action))
}

// dragRelease is the button going up: the drop, or nothing at all when
// the window under the pointer would not take it.
func (c *x11Conn) dragRelease() {
	if !c.drag.active {
		return
	}
	c.drag.released = true
	if !c.drag.target.Valid() || !c.drag.accepted {
		c.dragLeaveTarget()
		c.dragEnd(DragNone)
		return
	}
	c.sendXdnd(C.Window(c.drag.target.Send()), C.Window(c.drag.target.Window), XADrop,
		EncodeXdndDrop(uint32(c.drag.win), uint32(c.serverTime)))
	c.drag.dropped = true
	c.drag.dropAt = time.Now()
	// The pointer and the picture go now: the data transfer and
	// XdndFinished are between the two clients, and the user's drag is
	// visibly over the moment the button comes up.
	c.dragRelease_ungrab()
}

// dragRelease_ungrab gives the pointer and the keyboard back and takes the
// picture off the screen, leaving the drag alive only for the transfer.
func (c *x11Conn) dragRelease_ungrab() {
	if c.dpy == nil {
		return
	}
	C.ui_x_ungrab_ptr(c.dpy)
	C.ui_x_ungrab_kbd(c.dpy)
	c.dragFreeIcon()
	C.ui_x_flush(c.dpy)
}

// dragFinished is the target saying it is done with the data.
func (c *x11Conn) dragFinished(m XDNDMessage) {
	if !c.drag.active || !c.drag.dropped {
		return
	}
	version := XDNDVersion
	if c.drag.target.Valid() {
		version = c.drag.target.Version
	}
	_, accepted, action := DecodeXdndFinished(m, version)
	got := DragNone
	if accepted {
		got = c.xdnd.actions.Action(action)
		if got == DragNone {
			// A version-5 target that took the drop but named no
			// action still did something: the one it last agreed to.
			got = c.drag.action
		}
	}
	c.dragEnd(got)
}

// dragCancel ends a drag with nothing taken (Escape, or a second button).
func (c *x11Conn) dragCancel() {
	if !c.drag.active {
		return
	}
	c.dragLeaveTarget()
	c.dragEnd(DragNone)
}

// dragEnd stops the drag and tells the window it started in what became
// of it.
func (c *x11Conn) dragEnd(action DragAction) {
	if !c.drag.active {
		return
	}
	xdndTrace("drag ended, the target performed %v", action)
	win, released := c.drag.win, c.drag.released
	c.dragRelease_ungrab()
	if c.dpy != nil && win != 0 {
		C.ui_x_del_prop(c.dpy, win, c.xdnd.at(XATypeList))
		C.ui_x_del_prop(c.dpy, win, c.xdnd.at(XAActionList))
		if C.ui_x_owner(c.dpy, c.xdnd.at(XASelection)) == win {
			C.ui_x_set_owner(c.dpy, 0, c.xdnd.at(XASelection), C.CurrentTime)
		}
	}
	c.drag = x11Drag{}
	if !c.ownClip && !c.ownPrim && len(c.incrSends) == 0 {
		c.keep = false
	}
	if s := c.surfaces[win]; s != nil {
		c.queues[win] = append(c.queues[win], Event{Kind: EventDragEnd, Action: action, Dropped: released})
		// The drag ended between polls; wake the loop so the source
		// hears about it without waiting for the next input.
		C.ui_x_wake(c.dpy, c.helper)
	}
}

// dragCheckEscape cancels a drag whose Escape never arrived as a key
// event. The grab is the first way Escape is seen and this the second:
// between them a drag can always be called off.
func (c *x11Conn) dragCheckEscape() {
	if !c.drag.active || c.drag.dropped || c.dpy == nil {
		return
	}
	if C.ui_x_escape_held(c.dpy) != 0 {
		xdndTrace("escape held: the drag is off")
		c.dragCancel()
	}
}

// dragTimedOut gives up on a target that took the drop and never sent
// XdndFinished. Without this the source would hold the data — and the
// user's "move" would never complete — for as long as the app runs.
func (c *x11Conn) dragTimedOut() {
	if c.drag.active && c.drag.dropped && time.Since(c.drag.dropAt) > xdndFinishTimeout {
		c.dragEnd(c.drag.action)
	}
}

// xdndFinishTimeout is how long a source waits for XdndFinished after a
// drop. The spec warns a source to time out rather than block on a target
// that is misbehaving.
const xdndFinishTimeout = 10 * time.Second

// dragSelRequest serves our drag's data: a target converting
// XdndSelection to one of the types we offered.
func (c *x11Conn) dragSelRequest(xe *C.XEvent) bool {
	if !c.drag.active {
		return false
	}
	req := C.ui_x_sr_requestor(xe)
	target := C.ui_x_sr_target(xe)
	prop := C.ui_x_sr_property(xe)
	if prop == 0 {
		prop = target
	}
	switch target {
	case c.atomTargets:
		atoms := make([]C.ulong, 0, len(c.drag.payload.Types)+2)
		atoms = append(atoms, C.ulong(c.atomTargets), C.ulong(c.atomTimestamp))
		for _, t := range c.drag.payload.Types {
			if a := c.atomFor(t); a != 0 {
				atoms = append(atoms, C.ulong(a))
			}
		}
		C.ui_x_prop32(c.dpy, req, prop, C.XA_ATOM, &atoms[0], C.int(len(atoms)))
		C.ui_x_sel_notify(c.dpy, xe, prop)
		return true
	case c.atomTimestamp:
		ts := []C.ulong{C.ulong(c.serverTime)}
		C.ui_x_prop32(c.dpy, req, prop, C.XA_INTEGER, &ts[0], 1)
		C.ui_x_sel_notify(c.dpy, xe, prop)
		return true
	}
	mime := c.atomName(target)
	data, ok := c.dragData(mime)
	if !ok {
		C.ui_x_sel_notify(c.dpy, xe, 0)
		return true
	}
	thr := INCRThreshold(c.maxReq)
	if len(data) > thr {
		// Too big for one property: the ICCCM incremental protocol
		// sends it a piece at a time as the requestor consumes them.
		C.ui_x_watch_props(c.dpy, req)
		sz := []C.ulong{C.ulong(len(data))}
		C.ui_x_prop32(c.dpy, req, prop, c.atomINCR, &sz[0], 1)
		C.ui_x_sel_notify(c.dpy, xe, prop)
		c.incrSends = append(c.incrSends, incrSendState{
			requestor: req, prop: prop, typ: target,
			data: data, chunk: INCRChunkSize(thr),
		})
		c.keep = true
		return true
	}
	var ptr *C.char
	if len(data) > 0 {
		ptr = (*C.char)(unsafe.Pointer(&data[0]))
	}
	C.ui_x_prop8(c.dpy, req, prop, target, ptr, C.int(len(data)))
	C.ui_x_sel_notify(c.dpy, xe, prop)
	return true
}

// dragData reads one of the drag's types. The callback runs on the event
// loop, which is where every other backend call runs too.
func (c *x11Conn) dragData(mime string) ([]byte, bool) {
	if mime == "" || c.drag.payload.Data == nil {
		return nil, false
	}
	offered := false
	for _, t := range c.drag.payload.Types {
		if t == mime {
			offered = true
			break
		}
	}
	if !offered {
		return nil, false
	}
	return c.drag.payload.Data(mime)
}

// dragCursor sets the shape the pointer grab shows.
func (c *x11Conn) dragCursor(cur Cursor) {
	if c.drag.cursor == cur || c.dpy == nil {
		return
	}
	c.drag.cursor = cur
	C.ui_x_regrab_cursor(c.dpy, c.xcursor(cur), C.CurrentTime)
}

// ---- the drag's picture --------------------------------------------------------

// dragMakeIcon builds the window the drag's picture lives in. Without a
// 32-bit visual there is nowhere to put a translucent one, and the drag
// runs with the cursor alone.
func (c *x11Conn) dragMakeIcon(s *x11Surface, p DragPayload) {
	img := p.Icon
	if img == nil || img.Width < 1 || img.Height < 1 || c.dpy == nil {
		return
	}
	var vis *C.Visual
	var cmap C.Colormap
	var depth C.int
	if C.ui_x_argb(c.dpy, &vis, &cmap, &depth) == 0 {
		return
	}
	w, h := img.Width, img.Height
	win := C.ui_x_icon_win(c.dpy, vis, cmap, depth, C.int(w), C.int(h))
	if win == 0 {
		C.ui_x_free_colormap(c.dpy, cmap)
		return
	}
	stride := w * 4
	mem := C.malloc(C.size_t(stride * h))
	if mem == nil {
		C.ui_x_destroy_win(c.dpy, win, nil)
		C.ui_x_free_colormap(c.dpy, cmap)
		return
	}
	buf := unsafe.Slice((*byte)(mem), stride*h)
	// The picture is premultiplied, which is what a compositor reads a
	// 32-bit window's pixels as; the alpha is kept.
	copyImageRect(buf, stride, img, paintengine2d.XYWH(0, 0, float32(w), float32(h)), true, false)
	ximg := C.ui_x_icon_image(c.dpy, vis, depth, C.int(w), C.int(h), (*C.char)(mem))
	if ximg == nil {
		C.free(mem)
		C.ui_x_destroy_win(c.dpy, win, nil)
		C.ui_x_free_colormap(c.dpy, cmap)
		return
	}
	c.drag.icon = xdndIcon{
		win: win, gc: C.ui_x_gc(c.dpy, win), img: ximg, mem: mem,
		w: w, h: h, hx: int(p.Hotspot.X), hy: int(p.Hotspot.Y),
	}
	C.ui_x_map(c.dpy, win)
	c.dragPaintIcon()
}

func (c *x11Conn) dragPaintIcon() {
	ic := &c.drag.icon
	if ic.win == 0 || ic.img == nil || c.dpy == nil {
		return
	}
	C.ui_x_put(c.dpy, ic.win, ic.gc, ic.img, C.int(ic.w), C.int(ic.h))
	C.ui_x_flush(c.dpy)
}

func (c *x11Conn) dragMoveIcon(rx, ry int) {
	ic := &c.drag.icon
	if ic.win == 0 || c.dpy == nil {
		return
	}
	C.ui_x_move(c.dpy, ic.win, C.int(rx-ic.hx), C.int(ry-ic.hy))
}

func (c *x11Conn) dragFreeIcon() {
	ic := &c.drag.icon
	if ic.win == 0 {
		c.drag.icon = xdndIcon{}
		return
	}
	if c.dpy != nil {
		if ic.img != nil {
			// ui_destroy_image drops the data pointer first: the pixels
			// are ours to free, not XDestroyImage's.
			C.ui_x_destroy_image(ic.img)
		}
		C.ui_x_destroy_win(c.dpy, ic.win, ic.gc)
	}
	if ic.mem != nil {
		C.free(ic.mem)
	}
	c.drag.icon = xdndIcon{}
}

// ---- the window tree, for the search -------------------------------------------

// x11Tree is [XDNDTree] over a live X connection.
type x11Tree struct{ c *x11Conn }

func (t x11Tree) Children(win uint32) []uint32 {
	if t.c.dpy == nil {
		return nil
	}
	var kids *C.Window
	var n C.uint
	if C.ui_x_query_tree(t.c.dpy, C.Window(win), &kids, &n) == 0 || kids == nil {
		return nil
	}
	defer C.ui_x_free_kids(kids)
	out := make([]uint32, 0, int(n))
	for i := 0; i < int(n); i++ {
		out = append(out, uint32(C.ui_x_kid_at(kids, C.int(i))))
	}
	return out
}

func (t x11Tree) Geometry(win uint32) (x, y, w, h int, mapped bool) {
	if t.c.dpy == nil {
		return 0, 0, 0, 0, false
	}
	var cx, cy, cw, ch, cm C.int
	if C.ui_x_win_geom(t.c.dpy, C.Window(win), &cx, &cy, &cw, &ch, &cm) == 0 {
		return 0, 0, 0, 0, false
	}
	return int(cx), int(cy), int(cw), int(ch), cm != 0
}

func (t x11Tree) Aware(win uint32) (int, uint32) {
	if t.c.dpy == nil {
		return 0, 0
	}
	version := 0
	if v := C.ui_x_prop_card(t.c.dpy, C.Window(win), t.c.xdnd.at(XAAware)); v >= 0 {
		version = int(v)
	}
	proxy := uint32(C.ui_x_prop_window(t.c.dpy, C.Window(win), t.c.xdnd.at(XAProxy)))
	return version, proxy
}
