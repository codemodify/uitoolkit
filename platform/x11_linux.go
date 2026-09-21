//go:build linux && cgo

package platform

/*
#cgo linux LDFLAGS: -lX11 -lXext -lXrandr -lXfixes -lXrender -ldl
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <X11/Xresource.h>
#include <X11/XKBlib.h>
#include <X11/keysym.h>
#include <X11/cursorfont.h>
#include <X11/extensions/XShm.h>
#include <X11/extensions/Xrandr.h>
#include <X11/extensions/Xfixes.h>
#include <X11/extensions/shape.h>
#include <X11/extensions/Xrender.h>
#include <dlfcn.h>
#include <locale.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <poll.h>
#include <sys/ipc.h>
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

// poll, not select: a connection fd above FD_SETSIZE (1024) is undefined
// behaviour with fd_set and silently corrupts the stack.
static int ui_wait(Display* d, int ms) {
	if (!d) return -1;
	if (XPending(d)) return 1;
	struct pollfd pfd;
	pfd.fd = ConnectionNumber(d);
	pfd.events = POLLIN;
	pfd.revents = 0;
	int n = poll(&pfd, 1, ms);
	if (n > 0 && XPending(d) == 0) {
		// readable but no complete event yet: let the caller loop.
		return 1;
	}
	return n;
}

// ui_argb_visual finds a 32-bit TrueColor visual with an alpha channel and
// makes a colormap for it: what a window with a translucent shadow margin
// and rounded corners must be created with. It returns 0 when the screen
// has none (then the frame stays solid).
static int ui_argb_visual(Display* d, Visual** vis, Colormap* cmap, int* depth) {
	XVisualInfo tmpl, *found;
	int n = 0;
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
	Visual* pick = NULL;
	for (int i = 0; i < n; i++) {
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

// _NET_WM_CM_S<screen>: a compositing manager owns it while one runs.
// Without one the X server simply ignores a window's alpha, so a
// translucent frame would show black instead of the desktop.
static int ui_composited(Display* d) {
	char name[32];
	if (!d) return 0;
	snprintf(name, sizeof(name), "_NET_WM_CM_S%d", DefaultScreen(d));
	Atom sel = XInternAtom(d, name, False);
	return XGetSelectionOwner(d, sel) != None;
}

// ui_watch_composited asks for XFixesSelectionNotify on the compositing
// manager's selection, so a compositor starting or stopping is an event
// rather than a poll. It returns the extension's event base (0 when
// XFixes is missing).
static int ui_watch_composited(Display* d) {
	int ev = 0, err = 0;
	char name[32];
	if (!d || !XFixesQueryExtension(d, &ev, &err)) return 0;
	snprintf(name, sizeof(name), "_NET_WM_CM_S%d", DefaultScreen(d));
	Atom sel = XInternAtom(d, name, False);
	XFixesSelectSelectionInput(d, DefaultRootWindow(d), sel,
		XFixesSetSelectionOwnerNotifyMask | XFixesSelectionWindowDestroyNotifyMask |
		XFixesSelectionClientCloseNotifyMask);
	return ev;
}

// ui_shape_input restricts where a press reaches the window to one box
// (the visible window plus the resize band in the shadow); the rest of the
// margin clicks through to whatever is behind it.
static void ui_shape_input(Display* d, Window w, int x, int y, int width, int height) {
	XRectangle r;
	int ev = 0, err = 0;
	if (!d || !w || !XShapeQueryExtension(d, &ev, &err)) return;
	r.x = (short)x;
	r.y = (short)y;
	r.width = (unsigned short)(width > 0 ? width : 1);
	r.height = (unsigned short)(height > 0 ? height : 1);
	XShapeCombineRectangles(d, w, ShapeInput, 0, 0, &r, 1, ShapeSet, Unsorted);
}

// ui_shape_input_all gives the whole window back to input (a frame that
// stopped keeping a margin).
static void ui_shape_input_all(Display* d, Window w) {
	int ev = 0, err = 0;
	if (!d || !w || !XShapeQueryExtension(d, &ev, &err)) return;
	XShapeCombineMask(d, w, ShapeInput, 0, 0, None, ShapeSet);
}

// ui_shape_rects sets a window's silhouette from n rectangles of four ints
// each: the input shape (kind 0), which decides where a press lands, or the
// bounding shape (kind 1), which decides which pixels the window has at
// all. The bounding shape is the hard-edged fallback for a screen with no
// compositing manager, where per-pixel alpha is simply ignored and cutting
// the pixels away is the only way to make a hole. XShape rectangles are
// device pixels in window coordinates.
static void ui_shape_rects(Display* d, Window w, int kind, int* rects, int n) {
	int ev = 0, err = 0;
	if (!d || !w || !XShapeQueryExtension(d, &ev, &err)) return;
	XRectangle* r = NULL;
	int count = 0;
	if (n > 0) {
		r = (XRectangle*)calloc((size_t)n, sizeof(XRectangle));
		if (!r) return;
		for (int i = 0; i < n; i++) {
			int* q = rects + i * 4;
			if (q[2] <= 0 || q[3] <= 0) continue;
			r[count].x = (short)q[0];
			r[count].y = (short)q[1];
			r[count].width = (unsigned short)q[2];
			r[count].height = (unsigned short)q[3];
			count++;
		}
	}
	int kk = (kind == 1) ? ShapeBounding : ShapeInput;
	if (count == 0) {
		// An empty shape: no pixels, no input. ShapeSet with no
		// rectangles is the documented way to say that.
		XShapeCombineRectangles(d, w, kk, 0, 0, NULL, 0, ShapeSet, Unsorted);
	} else {
		// The rasteriser emits scanline order, which is exactly YXBanded:
		// saying so lets the server skip sorting hundreds of rectangles.
		XShapeCombineRectangles(d, w, kk, 0, 0, r, count, ShapeSet, YXBanded);
	}
	free(r);
}

// ui_shape_none drops a shape, giving every pixel of the window back.
static void ui_shape_none(Display* d, Window w, int kind) {
	int ev = 0, err = 0;
	if (!d || !w || !XShapeQueryExtension(d, &ev, &err)) return;
	XShapeCombineMask(d, w, (kind == 1) ? ShapeBounding : ShapeInput, 0, 0, None, ShapeSet);
}

static Window ui_create_visual(Display* d, int x, int y, int w, int h, const char* title, int popup, Visual* vis, Colormap cmap, int depth) {
	int s = DefaultScreen(d);
	XSetWindowAttributes swa;
	memset(&swa, 0, sizeof(swa));
	swa.background_pixel = vis ? 0 : BlackPixel(d, s);
	swa.border_pixel = vis ? 0 : BlackPixel(d, s);
	swa.bit_gravity = NorthWestGravity;
	swa.colormap = vis ? cmap : DefaultColormap(d, s);
	swa.override_redirect = popup ? True : False;
	swa.event_mask = ExposureMask|KeyPressMask|KeyReleaseMask|ButtonPressMask|ButtonReleaseMask|
		PointerMotionMask|LeaveWindowMask|StructureNotifyMask|FocusChangeMask|PropertyChangeMask;
	unsigned long mask = CWBackPixel|CWBorderPixel|CWBitGravity|CWColormap|CWEventMask;
	if (popup) {
		mask |= CWOverrideRedirect;
	}
	if (!popup && x == 0 && y == 0) {
		x = 40;
		y = 40;
	}
	Window win = XCreateWindow(d, RootWindow(d, s), x, y, (unsigned)w, (unsigned)h, 0,
		vis ? depth : DefaultDepth(d, s), InputOutput, vis ? vis : DefaultVisual(d, s),
		mask, &swa);
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
	hints.min_width = popup ? 1 : 200;
	hints.min_height = popup ? 1 : 120;
	if (popup || x != 40 || y != 40) {
		hints.flags |= USPosition | PPosition;
		hints.x = x;
		hints.y = y;
	}
	XSetWMNormalHints(d, win, &hints);
	if (popup) {
		Atom wtype = XInternAtom(d, "_NET_WM_WINDOW_TYPE", False);
		Atom menu = XInternAtom(d, "_NET_WM_WINDOW_TYPE_POPUP_MENU", False);
		XChangeProperty(d, win, wtype, XA_ATOM, 32, PropModeReplace, (unsigned char*)&menu, 1);
		Atom state = XInternAtom(d, "_NET_WM_STATE", False);
		Atom skipTask = XInternAtom(d, "_NET_WM_STATE_SKIP_TASKBAR", False);
		Atom skipPager = XInternAtom(d, "_NET_WM_STATE_SKIP_PAGER", False);
		Atom above = XInternAtom(d, "_NET_WM_STATE_ABOVE", False);
		Atom states[3] = {skipTask, skipPager, above};
		XChangeProperty(d, win, state, XA_ATOM, 32, PropModeReplace, (unsigned char*)states, 3);
	}
	XFlush(d);
	return win;
}

static Window ui_create(Display* d, int x, int y, int w, int h, const char* title, int popup) {
	return ui_create_visual(d, x, y, w, h, title, popup, NULL, 0, 0);
}

static void ui_move(Display* d, Window w, int x, int y) {
	XMoveWindow(d, w, x, y);
	XFlush(d);
}

static void ui_focus(Display* d, Window w) {
	XSetInputFocus(d, w, RevertToParent, CurrentTime);
	XFlush(d);
}

static void ui_wake(Display* d, Window helper) {
	if (!d || !helper) return;
	XChangeProperty(d, helper, XA_WM_NAME, XA_STRING, 8, PropModeReplace, (const unsigned char*)"w", 1);
	XFlush(d);
}

static void ui_map(Display* d, Window w) {
	XMapWindow(d, w);
	XFlush(d);
}

// WM_NORMAL_HINTS: the window's minimum, and its maximum where it has one.
// A window manager reads PMinSize == PMaxSize as "not resizable" — it drops
// the resize edges, greys maximize out and clamps a drag — which is how a
// fixed-size window is stated on X11.
static void ui_resize_hints(Display* d, Window w, int minw, int minh, int maxw, int maxh) {
	XSizeHints hints;
	memset(&hints, 0, sizeof(hints));
	if (minw > 0 || minh > 0) {
		hints.flags |= PMinSize;
		hints.min_width = minw > 0 ? minw : 1;
		hints.min_height = minh > 0 ? minh : 1;
	}
	if (maxw > 0 || maxh > 0) {
		hints.flags |= PMaxSize;
		hints.max_width = maxw > 0 ? maxw : 1 << 20;
		hints.max_height = maxh > 0 ? maxh : 1 << 20;
	}
	if (!hints.flags) return;
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

static XImage* ui_image(Display* d, int w, int h, char* data, Visual* vis, int depth) {
	Visual* v = vis ? vis : DefaultVisual(d, DefaultScreen(d));
	int dep = vis ? depth : DefaultDepth(d, DefaultScreen(d));
	return XCreateImage(d, v, (unsigned)dep, ZPixmap, 0, data, (unsigned)w, (unsigned)h, 32, 0);
}

static int ui_img_bpl(XImage* img) { return img ? img->bytes_per_line : 0; }
static int ui_img_bpp(XImage* img) { return img ? img->bits_per_pixel : 0; }
static int ui_img_msb(XImage* img) { return img && img->byte_order == MSBFirst; }
static unsigned long ui_red_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->red_mask; }
static unsigned long ui_green_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->green_mask; }
static unsigned long ui_blue_mask(Display* d) { return DefaultVisual(d, DefaultScreen(d))->blue_mask; }
static unsigned long ui_vis_red(Visual* v) { return v ? v->red_mask : 0; }
static unsigned long ui_vis_green(Visual* v) { return v ? v->green_mask : 0; }
static unsigned long ui_vis_blue(Visual* v) { return v ? v->blue_mask : 0; }
static void ui_free_colormap(Display* d, Colormap c) { if (d && c) XFreeColormap(d, c); }

static int ui_event_type(XEvent* e) { return e->type; }
static int ui_cross_mode(XEvent* e) { return e->xcrossing.mode; }
static Window ui_event_window(XEvent* e) { return e->xany.window; }

static int ui_expose_x(XEvent* e) { return e->xexpose.x; }
static int ui_expose_y(XEvent* e) { return e->xexpose.y; }
static int ui_expose_w(XEvent* e) { return e->xexpose.width; }
static int ui_expose_h(XEvent* e) { return e->xexpose.height; }

static int ui_cfg_w(XEvent* e) { return e->xconfigure.width; }
static int ui_cfg_h(XEvent* e) { return e->xconfigure.height; }
static int ui_cfg_x(XEvent* e) { return e->xconfigure.x; }
static int ui_cfg_y(XEvent* e) { return e->xconfigure.y; }
// A window manager sends a synthetic ConfigureNotify whose x, y are the
// window's position on the root (ICCCM 4.2.3); the real one from a
// reparenting manager carries the position inside the frame instead.
static int ui_cfg_synthetic(XEvent* e) { return e->xconfigure.send_event ? 1 : 0; }

static int ui_btn(XEvent* e) { return e->xbutton.button; }
static int ui_btn_x(XEvent* e) { return e->xbutton.x; }
static int ui_btn_y(XEvent* e) { return e->xbutton.y; }
static unsigned ui_btn_state(XEvent* e) { return e->xbutton.state; }
static int ui_btn_xroot(XEvent* e) { return e->xbutton.x_root; }
static int ui_btn_yroot(XEvent* e) { return e->xbutton.y_root; }
static Time ui_btn_time(XEvent* e) { return e->xbutton.time; }

// ui_get_atoms reads a list property (ATOM[] such as _NET_WM_STATE,
// _NET_SUPPORTED, _NET_WM_ALLOWED_ACTIONS) into out; it returns how many.
static int ui_get_atoms(Display* d, Window w, Atom prop, Atom* out, int max) {
	Atom type = None;
	int fmt = 0;
	unsigned long n = 0, rem = 0;
	unsigned char* data = NULL;
	if (!d || !w) return 0;
	if (XGetWindowProperty(d, w, prop, 0, max, False, XA_ATOM, &type, &fmt, &n, &rem, &data) != Success || !data) {
		return 0;
	}
	int k = 0;
	if (fmt == 32) {
		for (unsigned long i = 0; i < n && k < max; i++) out[k++] = ((Atom*)data)[i];
	}
	XFree(data);
	return k;
}

// ui_wm_name is the window manager's name: _NET_WM_NAME of the window
// _NET_SUPPORTING_WM_CHECK names on the root window.
static int ui_wm_name(Display* d, char* out, int max) {
	Atom check = XInternAtom(d, "_NET_SUPPORTING_WM_CHECK", False);
	Atom name = XInternAtom(d, "_NET_WM_NAME", False);
	Atom utf8 = XInternAtom(d, "UTF8_STRING", False);
	Atom type = None;
	int fmt = 0;
	unsigned long n = 0, rem = 0;
	unsigned char* data = NULL;
	Window wm = 0;
	if (XGetWindowProperty(d, DefaultRootWindow(d), check, 0, 1, False, XA_WINDOW, &type, &fmt, &n, &rem, &data) == Success && data) {
		if (fmt == 32 && n == 1) wm = *(Window*)data;
		XFree(data);
	}
	if (!wm) return 0;
	data = NULL;
	int k = 0;
	if (XGetWindowProperty(d, wm, name, 0, 64, False, utf8, &type, &fmt, &n, &rem, &data) == Success && data) {
		if (fmt == 8) {
			for (unsigned long i = 0; i < n && k < max - 1; i++) out[k++] = (char)data[i];
		}
		XFree(data);
	}
	out[k] = 0;
	return k;
}

// ui_root_message sends a client message about w to the root window, the
// way EWMH requests (_NET_WM_MOVERESIZE, _GTK_SHOW_WINDOW_MENU) travel.
static void ui_root_message(Display* d, Window w, Atom type, long l0, long l1, long l2, long l3, long l4) {
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.xclient.type = ClientMessage;
	ev.xclient.window = w;
	ev.xclient.message_type = type;
	ev.xclient.format = 32;
	ev.xclient.data.l[0] = l0;
	ev.xclient.data.l[1] = l1;
	ev.xclient.data.l[2] = l2;
	ev.xclient.data.l[3] = l3;
	ev.xclient.data.l[4] = l4;
	XSendEvent(d, DefaultRootWindow(d), False, SubstructureRedirectMask|SubstructureNotifyMask, &ev);
	XFlush(d);
}

// The client "MUST release all grabs" before _NET_WM_MOVERESIZE: the
// button press's implicit grab included.
static void ui_ungrab_pointer(Display* d) {
	XUngrabPointer(d, CurrentTime);
	XFlush(d);
}

static void ui_iconify(Display* d, Window w) {
	XIconifyWindow(d, w, DefaultScreen(d));
	XFlush(d);
}

static void ui_lower(Display* d, Window w) {
	XLowerWindow(d, w);
	XFlush(d);
}

static int ui_to_root(Display* d, Window w, int x, int y, int* rx, int* ry) {
	Window child = 0;
	return XTranslateCoordinates(d, w, DefaultRootWindow(d), x, y, rx, ry, &child);
}

static int ui_mot_x(XEvent* e) { return e->xmotion.x; }
static int ui_mot_y(XEvent* e) { return e->xmotion.y; }
static unsigned ui_mot_state(XEvent* e) { return e->xmotion.state; }

static unsigned ui_key_state(XEvent* e) { return e->xkey.state; }

static KeySym ui_lookup(Display* d, XEvent* e, char* buf, int n, int* len) {
	KeySym ks = 0;
	*len = XLookupString(&e->xkey, buf, n, &ks, NULL);
	return ks;
}

// utf8 is set when the bytes are UTF-8 (Xutf8LookupString). Without an
// input context XLookupString returns ISO-8859-1, which must be re-encoded
// before it reaches Go or every accented character becomes U+FFFD.
#ifdef X_HAVE_UTF8_STRING
static int ui_lookup_im(XIC ic, Display* d, XEvent* e, char* buf, int n, KeySym* ks, int* utf8) {
	*ks = 0;
	*utf8 = 0;
	if (ic) {
		Status st = 0;
		int len = Xutf8LookupString(ic, &e->xkey, buf, n, ks, &st);
		*utf8 = 1;
		if (st == XBufferOverflow) {
			return -len;
		}
		return len;
	}
	(void)d;
	return XLookupString(&e->xkey, buf, n, ks, NULL);
}
#else
static int ui_lookup_im(XIC ic, Display* d, XEvent* e, char* buf, int n, KeySym* ks, int* utf8) {
	(void)ic;
	(void)d;
	*ks = 0;
	*utf8 = 0;
	return XLookupString(&e->xkey, buf, n, ks, NULL);
}
#endif

// ui_base_sym is the keysym this physical key produces on group 0, level 0
// -- the layout-independent symbol. Shortcuts must resolve against it or
// Ctrl+C stops working under a Cyrillic / Greek / Hebrew layout.
static KeySym ui_base_sym(Display* d, XEvent* e) {
	if (!d || !e) return 0;
	return XkbKeycodeToKeysym(d, (KeyCode)e->xkey.keycode, 0, 0);
}

static Time ui_event_time(XEvent* e) {
	if (!e) return CurrentTime;
	switch (e->type) {
	case KeyPress:
	case KeyRelease:
		return e->xkey.time;
	case ButtonPress:
	case ButtonRelease:
		return e->xbutton.time;
	case MotionNotify:
		return e->xmotion.time;
	case PropertyNotify:
		return e->xproperty.time;
	case SelectionClear:
		return e->xselectionclear.time;
	case SelectionRequest:
		return e->xselectionrequest.time;
	case SelectionNotify:
		return e->xselection.time;
	}
	return CurrentTime;
}

// A zero-length append to a property generates a PropertyNotify carrying a
// real server timestamp. ICCCM wants selection ownership taken with such a
// timestamp, never CurrentTime.
static void ui_touch_prop(Display* d, Window w, Atom prop) {
	if (!d || !w) return;
	XChangeProperty(d, w, prop, XA_STRING, 8, PropModeAppend, (const unsigned char*)"", 0);
	XFlush(d);
}

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

static void ui_set_owner(Display* d, Window w, Atom sel, Time t) {
	XSetSelectionOwner(d, sel, w, t);
}

static Window ui_owner(Display* d, Atom sel) { return XGetSelectionOwner(d, sel); }

static void ui_convert(Display* d, Window w, Atom sel, Atom target, Atom prop, Time t) {
	XConvertSelection(d, sel, target, prop, w, t);
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

// ui_get_prop reads a whole property, however large: size it with a
// zero-length probe (bytes_after), then fetch that many 32-bit words in
// one request. The old fixed 256KiB cap silently truncated a large
// non-INCR paste. The caller frees the data with ui_xfree.
static int ui_get_prop(Display* d, Window w, Atom prop, unsigned char** data, unsigned long* nitems, Atom* type) {
	Atom actual = 0;
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
	if (p) {
		XFree(p);
		p = NULL;
	}
	if (fmt == 0) {
		// property does not exist; still delete so INCR handshakes advance
		XDeleteProperty(d, w, prop);
		return 0;
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

static Cursor ui_font_cursor(Display* d, unsigned int shape) {
	if (!d) return None;
	return XCreateFontCursor(d, shape);
}
static Cursor ui_theme_cursor(Display* d, const char* name) {
	static int tried = 0;
	static Cursor (*load)(Display*, const char*) = 0;
	if (!tried) {
		void* h;
		tried = 1;
		h = dlopen("libXcursor.so.1", RTLD_LAZY | RTLD_LOCAL);
		if (h) {
			union { void* p; Cursor (*f)(Display*, const char*); } u;
			u.p = dlsym(h, "XcursorLibraryLoadCursor");
			load = u.f;
		}
	}
	if (d && load && name && name[0]) {
		Cursor c = load(d, name);
		if (c) return c;
	}
	return None;
}
static void ui_define_cursor(Display* d, Window w, Cursor c) {
	if (!d || !w) return;
	XDefineCursor(d, w, c);
	XFlush(d);
}
static void ui_free_cursor(Display* d, Cursor c) {
	if (d && c) XFreeCursor(d, c);
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

static void ui_raise(Display* d, Window w) {
	XMapRaised(d, w);
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.xclient.type = ClientMessage;
	ev.xclient.window = w;
	ev.xclient.message_type = XInternAtom(d, "_NET_ACTIVE_WINDOW", False);
	ev.xclient.format = 32;
	ev.xclient.data.l[0] = 1;
	ev.xclient.data.l[1] = CurrentTime;
	XSendEvent(d, DefaultRootWindow(d), False, SubstructureRedirectMask|SubstructureNotifyMask, &ev);
	XFlush(d);
}

static void ui_unmap(Display* d, Window w) {
	XUnmapWindow(d, w);
	XFlush(d);
}

static int uitk_xerr = 0;
static int uitk_on_xerr(Display* d, XErrorEvent* e) {
	(void)d;
	uitk_xerr = e->error_code;
	return 0;
}

// Xlib's default error handler prints and calls exit(1). A BadWindow from
// a request racing the window manager (or our own request on a window the
// server already destroyed) must not kill the application, so install a
// handler that hands the error to Go for logging and returns.
extern void uitkXError(int code, int request, int minor);
static int uitk_report_xerr(Display* d, XErrorEvent* e) {
	(void)d;
	uitk_xerr = e->error_code;
	uitkXError((int)e->error_code, (int)e->request_code, (int)e->minor_code);
	return 0;
}
static void ui_install_error_handler(void) {
	XSetErrorHandler(uitk_report_xerr);
}

static int ui_shm_query(Display* d) {
	int ev = 0, err = 0, major = 0, minor = 0, pix = 0;
	if (!d || !XShmQueryExtension(d)) return 0;
	if (!XShmQueryVersion(d, &major, &minor, &pix)) return 0;
	return 1;
}

static XImage* ui_shm_image(Display* d, int w, int h, void** info_out, void** addr, Visual* vis, int depth) {
	XShmSegmentInfo* info = (XShmSegmentInfo*)calloc(1, sizeof(XShmSegmentInfo));
	if (!info) return NULL;
	Visual* v = vis ? vis : DefaultVisual(d, DefaultScreen(d));
	int dep = vis ? depth : DefaultDepth(d, DefaultScreen(d));
	XImage* img = XShmCreateImage(d, v, (unsigned)dep, ZPixmap, NULL, info, (unsigned)w, (unsigned)h);
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
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// X11Backend presents through paintengine2d. UITK_PAINT=auto|gpu binds
// GPUDevice to the X window (eglSwapBuffers) when EGL works; otherwise
// present is dirty-rect XPutImage / MIT-SHM.
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
	// WindowOptions are logical pixels; an X window's size is device
	// pixels, because an X server has no notion of a display scale. The
	// two meet here and nowhere else: lw, lh stay the logical size the
	// app asked for, w, h are what the server is told.
	lw, lh := opts.Width, opts.Height
	if lw < 1 {
		lw = 640
	}
	if lh < 1 {
		lh = 480
	}
	if opts.Scale > 0 {
		// The application states the scale it draws at; the connection's
		// own is only the fallback.
		c.scale = opts.Scale
	}
	scale := c.displayScale()
	w, h := DevicePixels(lw, scale), DevicePixels(lh, scale)
	title := opts.Title
	if title == "" {
		title = "uitoolkit"
	}
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	x11Mu.Lock()
	x, y := opts.X, opts.Y
	popup := 0
	if opts.Popup {
		popup = 1
	}
	win := C.ui_create(c.dpy, C.int(x), C.int(y), C.int(w), C.int(h), ctitle, C.int(popup))
	gc := C.ui_gc(c.dpy, win)
	ic, cbs := x11CreateIC(c.im, win)
	s := &x11Surface{
		conn:     c,
		win:      win,
		gc:       gc,
		ic:       ic,
		ximCbs:   cbs,
		title:    title,
		img:      paintengine2d.NewImage(w, h),
		rmask:    uint32(C.ui_red_mask(c.dpy)),
		gmask:    uint32(C.ui_green_mask(c.dpy)),
		bmask:    uint32(C.ui_blue_mask(c.dpy)),
		popup:    opts.Popup,
		wantDeco: requestedDecorations(opts.Decorations),
		focused:  true,
		geomW:    w,
		geomH:    h,
		geomLW:   lw,
		geomLH:   lh,
		opts:     opts,
		sizing:   opts.Sizing,
	}
	s.limits = limitsFor(s.sizing, opts, lw, lh)
	s.applySizeHintsLocked()
	s.state.Solid = !c.composited
	if opts.Popup {
		// Override-redirect: no window manager frame to ask about.
		s.wantDeco = DecorationsNone
	} else if s.wantDeco != DecorationsServer {
		// Before the window is mapped, so it never shows with a frame.
		s.setMotifLocked()
	}
	s.rebuildImageLocked()
	c.surfaces[win] = s
	// XdndAware goes on the toplevel window before it is mapped, so a
	// drag over it finds a target from the very first frame.
	s.setXdndAwareLocked()
	x11Mu.Unlock()
	s.tryBindGPU()
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
	atomTimestamp C.Atom
	atomProp      C.Atom
	atomString    C.Atom
	atomNetState  C.Atom
	atomMaxVert   C.Atom
	atomMaxHorz   C.Atom
	atomFullscr   C.Atom
	atomHidden    C.Atom
	atomFocused   C.Atom
	atomAllowed   C.Atom
	atomActMin    C.Atom
	atomActMaxH   C.Atom
	atomActMaxV   C.Atom
	atomActFull   C.Atom
	atomSupported C.Atom
	atomMoveRes   C.Atom
	atomShowMenu  C.Atom
	atomMotif     C.Atom
	// supported is the window manager's _NET_SUPPORTED, read once (nil
	// until then); wmName is its _NET_WM_NAME.
	supported map[C.Atom]bool
	wmName    string
	// composited is whether a compositing manager owns _NET_WM_CM_S<n>
	// (translucent frames need one); fixesEvent is the XFixes event base
	// the selection notifications arrive on (0: no XFixes).
	composited  bool
	fixesEvent  int
	atomExtents C.Atom
	atomOpaque  C.Atom
	// atomBlur is KWin's _KDE_NET_WM_BLUR_BEHIND_REGION: the X11 way to
	// ask for the desktop behind the window to be blurred.
	atomBlur C.Atom

	clipText string
	ownClip  bool
	ownPrim  bool
	// ownTime is the timestamp ownership was taken with; the ICCCM
	// TIMESTAMP target must return exactly that value.
	ownTime C.Time
	// serverTime is the most recent timestamp seen on an X event. ICCCM
	// requires selection ownership and conversions to use a real server
	// time, never CurrentTime.
	serverTime C.Time
	// servicing is set while the background selection servicer runs (no
	// window loop is left to answer SelectionRequest).
	servicing bool
	pasteText string
	pasteDone bool
	pasteWant C.Atom
	incrRecv  incrRecvState
	incrSends []incrSendState
	maxReq    int

	// cursors caches one X cursor per shape.
	cursors map[Cursor]C.Cursor

	// XDND (x11_dnd_linux.go): the interned atoms, the drag another
	// client has over one of our windows, and the drag one of them
	// started.
	xdnd xdndAtoms
	drop x11Drop
	drag x11Drag
}

type x11Surface struct {
	soft   softDevice // software devices of the buffers (no GPU)
	conn   *x11Conn
	win    C.Window
	gc     C.GC
	ic     C.XIC
	ximg   *C.XImage
	title  string
	img    *paintengine2d.Image
	xbuf   []byte
	xmem   unsafe.Pointer // C.malloc'd XImage backing store (non-SHM)
	stride int
	bpp    int
	msb    bool
	rmask  uint32
	gmask  uint32
	bmask  uint32
	mapped bool
	// shaped: a bounding shape is set, so dropping the shape has to give
	// the pixels back rather than leave the last silhouette behind.
	shaped     bool
	closed     bool
	shm        bool
	shmInfo    unsafe.Pointer
	imeSpotX   int
	imeSpotY   int
	preeditBuf string
	ximCbs     unsafe.Pointer
	gpu        *paintengine2d.GPUDevice
	popup      bool
	// hidden is set by Hide: presenting must not map the window again
	// (close-to-tray hid it, and the next repaint showed it).
	hidden bool

	// state, caps: what the window manager says about the window
	// (_NET_WM_STATE, _NET_WM_ALLOWED_ACTIONS); focused tracks focus
	// events for managers without _NET_WM_STATE_FOCUSED. wantDeco is the
	// requested decoration mode (_MOTIF_WM_HINTS).
	state    WindowState
	caps     WMCaps
	focused  bool
	wantDeco Decorations
	// frame is the client frame's margin and regions; geomW / geomH the
	// visible window in device pixels, which the X window is that plus
	// the margin, and geomLW / geomLH the same window in the logical
	// pixels the toolkit's window geometry speaks — what Resize was last
	// given, and what the last EventResize reported. vis is the 32-bit
	// visual the window was created with for a translucent frame (nil:
	// the screen's default, opaque one).
	frame          Frame
	geomW, geomH   int
	geomLW, geomLH int
	vis            *C.Visual
	visDepth       int
	cmap           C.Colormap
	// opts are what the window was made with, so it can be re-created on
	// another visual when the frame starts or stops needing alpha; sizing
	// is its resize policy and limits what WM_NORMAL_HINTS last said
	// (platform.SizingSurface).
	opts   WindowOptions
	sizing Sizing
	limits SizeLimits
	// posX, posY are where the last ConfigureNotify put the X window and
	// posKnown that one has arrived. posRoot says those coordinates were
	// already root-relative — a synthetic configure from the window
	// manager — so a reparenting manager's frame-relative ones are
	// translated when they are asked for rather than on every configure
	// of an interactive move.
	posX, posY        int
	posKnown, posRoot bool
	// The last button press: root position, button and time, which
	// _NET_WM_MOVERESIZE needs; buttons is the set held now.
	pressRootX, pressRootY int
	pressButton            int
	pressTime              C.Time
	buttons                uint8
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
		C.ui_install_error_handler()
	})
}

// x11Errors counts X protocol errors delivered to the process-wide
// handler; the first few are logged, the rest only counted so a storm
// cannot flood a terminal.
var x11Errors atomic.Int64

// X11Errors is the number of X protocol errors seen since start (tests
// and diagnostics).
func X11Errors() int { return int(x11Errors.Load()) }

//export uitkXError
func uitkXError(code, request, minor C.int) {
	n := x11Errors.Add(1)
	if n <= 8 || os.Getenv("UITK_X11_DEBUG") != "" {
		log.Printf("uitk x11: protocol error %d (request %d.%d)", int(code), int(request), int(minor))
	}
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
	c.atomTimestamp = internAtom(d, "TIMESTAMP")
	c.atomProp = internAtom(d, "UITK_CLIP")
	c.atomString = C.XA_STRING
	c.atomNetState = internAtom(d, "_NET_WM_STATE")
	c.atomMaxVert = internAtom(d, "_NET_WM_STATE_MAXIMIZED_VERT")
	c.atomMaxHorz = internAtom(d, "_NET_WM_STATE_MAXIMIZED_HORZ")
	c.atomFullscr = internAtom(d, "_NET_WM_STATE_FULLSCREEN")
	c.atomHidden = internAtom(d, "_NET_WM_STATE_HIDDEN")
	c.atomFocused = internAtom(d, "_NET_WM_STATE_FOCUSED")
	c.atomAllowed = internAtom(d, "_NET_WM_ALLOWED_ACTIONS")
	c.atomActMin = internAtom(d, "_NET_WM_ACTION_MINIMIZE")
	c.atomActMaxH = internAtom(d, "_NET_WM_ACTION_MAXIMIZE_HORZ")
	c.atomActMaxV = internAtom(d, "_NET_WM_ACTION_MAXIMIZE_VERT")
	c.atomActFull = internAtom(d, "_NET_WM_ACTION_FULLSCREEN")
	c.atomSupported = internAtom(d, "_NET_SUPPORTED")
	c.atomMoveRes = internAtom(d, "_NET_WM_MOVERESIZE")
	c.atomShowMenu = internAtom(d, "_GTK_SHOW_WINDOW_MENU")
	c.atomMotif = internAtom(d, "_MOTIF_WM_HINTS")
	c.atomExtents = internAtom(d, "_GTK_FRAME_EXTENTS")
	c.atomOpaque = internAtom(d, "_NET_WM_OPAQUE_REGION")
	c.atomBlur = internAtom(d, "_KDE_NET_WM_BLUR_BEHIND_REGION")
	c.composited = C.ui_composited(d) != 0
	c.fixesEvent = int(C.ui_watch_composited(d))
	c.cursors = map[Cursor]C.Cursor{}
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
	// An explicit UITK_SCALE (or GDK_SCALE, QT_SCALE_FACTOR) is what the
	// toolkit draws at, so it has to be what a window's geometry is
	// converted with too: a look at 2x inside a window sized for 1x is a
	// window with half its content cut off.
	if env := ScaleFromEnv(); env > 0 {
		c.scale = env
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
	if c.refs > 0 {
		return
	}
	if c.keep {
		// We still own a selection: keep serving it from a background
		// drainer now that no window loop is left to do it.
		c.startSelectionServiceLocked()
		return
	}
	if !c.servicing {
		c.closeLocked()
	}
}

func (c *x11Conn) closeLocked() {
	if c.dpy != nil {
		for k, cur := range c.cursors {
			if cur != 0 {
				C.ui_free_cursor(c.dpy, cur)
			}
			delete(c.cursors, k)
		}
	}
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

func (s *x11Surface) Title() string { return s.title }
func (s *x11Surface) Size() (w, h int) {
	if s.gpu != nil {
		return s.gpu.Size()
	}
	if s.img != nil {
		return s.img.Width, s.img.Height
	}
	return 0, 0
}
func (s *x11Surface) Buffer() *paintengine2d.Image {
	if s.gpu != nil {
		if img := s.gpu.Image(); img != nil {
			return img
		}
	}
	return s.img
}
func (s *x11Surface) Closed() bool { return s.closed }
func (s *x11Surface) Scale() float32 {
	if s == nil {
		return 1
	}
	return s.conn.displayScale()
}

// displayScale is the connection's display scale, never zero: what a
// window's logical size is multiplied by to get the pixels the X server
// is told about.
func (c *x11Conn) displayScale() float32 {
	if c != nil && c.scale > 0 {
		return c.scale
	}
	return 1
}

func (s *x11Surface) SetTitle(title string) {
	s.title = title
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		// After Close the window id is gone but the display may still be
		// shared; a request on window 0 is a BadWindow error.
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

// Resize sizes the *window*, in logical pixels: the X window is that many
// device pixels — an X server has no notion of a display scale — plus the
// frame's margin, the band the shadow lives in (_GTK_FRAME_EXTENTS tells
// the window manager as much, so it moves and snaps the window, not the
// shadow).
func (s *x11Surface) Resize(lw, lh int) error {
	if s.closed || s.win == 0 {
		return nil
	}
	if lw < 1 {
		lw = 1
	}
	if lh < 1 {
		lh = 1
	}
	if lw == s.geomLW && lh == s.geomLH {
		// The size the app was last told, handed back: an EventResize
		// echoed by the window's own handler. Asking the server for the
		// device size this rounds to would fight a window manager that
		// chose a size between two logical pixels.
		return nil
	}
	s.geomLW, s.geomLH = lw, lh
	w, h := DevicePixels(lw, s.conn.displayScale()), DevicePixels(lh, s.conn.displayScale())
	s.geomW, s.geomH = w, h
	if s.sizing == SizingFixed {
		// The pin follows the window: the app is allowed to resize its
		// own fixed window, the user is not.
		s.limits = limitsFor(s.sizing, s.opts, lw, lh)
		x11Mu.Lock()
		s.applySizeHintsLocked()
		x11Mu.Unlock()
	}
	m := s.frame.Margin
	w, h = w+m.Width(), h+m.Height()
	if s.img.Width == w && s.img.Height == h {
		return nil
	}
	s.img = paintengine2d.NewImage(w, h)
	x11Mu.Lock()
	if s.gpu != nil {
		s.resizeGPU(w, h)
	}
	if s.gpu == nil {
		s.rebuildImageLocked()
	}
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
		img := C.ui_shm_image(s.conn.dpy, C.int(w), C.int(h), &info, &addr, s.vis, C.int(s.visDepth))
		if img != nil && addr != nil {
			if !s.adoptImageLocked(img, addr, h) {
				C.ui_shm_destroy(s.conn.dpy, img, info)
				return
			}
			s.shm = true
			s.shmInfo = info
			return
		}
	}
	// XCreateImage keeps the data pointer for the life of the image, so
	// the backing store must be C memory: handing C a pointer into a Go
	// slice it retains violates the cgo pointer rules.
	probe := C.ui_image(s.conn.dpy, C.int(w), C.int(h), nil, s.vis, C.int(s.visDepth))
	if probe == nil {
		return
	}
	stride := int(C.ui_img_bpl(probe))
	C.ui_destroy_image(probe)
	if stride < 1 {
		return
	}
	size := stride * h
	if size < 4 {
		size = 4
	}
	mem := C.calloc(1, C.size_t(size))
	if mem == nil {
		return
	}
	img := C.ui_image(s.conn.dpy, C.int(w), C.int(h), (*C.char)(mem), s.vis, C.int(s.visDepth))
	if img == nil {
		C.free(mem)
		return
	}
	s.xmem = mem
	if !s.adoptImageLocked(img, mem, h) {
		C.ui_destroy_image(img)
		C.free(mem)
		s.xmem = nil
		return
	}
	s.shm = false
}

// adoptImageLocked wires an XImage and its backing store into the surface,
// refusing any geometry this code cannot pack pixels for. stride comes
// from the server (bytes_per_line) and is never widened: doing that on a
// 16-bpp visual made the slice twice the real MIT-SHM segment and the
// blit walked off the end of it.
func (s *x11Surface) adoptImageLocked(img *C.XImage, addr unsafe.Pointer, h int) bool {
	stride := int(C.ui_img_bpl(img))
	bpp := int(C.ui_img_bpp(img))
	if stride < 1 || h < 1 || !x11SupportedBPP(bpp) {
		x11WarnVisual(bpp)
		return false
	}
	s.ximg = img
	s.stride = stride
	s.bpp = bpp
	s.msb = C.ui_img_msb(img) != 0
	s.xbuf = unsafe.Slice((*byte)(addr), stride*h)
	return true
}

var x11VisualWarned atomic.Bool

// x11SupportedBPP reports whether the CPU present path can pack pixels
// for this server pixel size. 32 and 16 bits per pixel cover every
// TrueColor visual in practice; anything else (8-bit PseudoColor, packed
// 24) presents nothing rather than corrupting memory.
func x11SupportedBPP(bpp int) bool { return bpp == 32 || bpp == 16 }

func x11WarnVisual(bpp int) {
	if x11VisualWarned.CompareAndSwap(false, true) {
		log.Printf("uitk x11: unsupported visual (%d bits per pixel); "+
			"window will not present -- use a 32- or 16-bpp visual, or UITK_BACKEND=offscreen", bpp)
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
		s.stride = 0
		s.bpp = 0
		return
	}
	if s.ximg != nil {
		C.ui_destroy_image(s.ximg)
		s.ximg = nil
	}
	if s.xmem != nil {
		C.free(s.xmem)
		s.xmem = nil
	}
	s.xbuf = nil
	s.stride = 0
	s.bpp = 0
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
	if stride < 1 || len(s.xbuf) == 0 {
		return
	}
	bytesPP := s.bpp / 8
	if bytesPP < 1 {
		return
	}
	// Typical LE TrueColor (red 0xff0000) is the same BGRA layout as
	// Wayland XRGB8888 — one damage-rect swizzle, not NRGBAAt+pack
	// per pixel (that path was ~70ms/frame at 1000×760).
	if bytesPP == 4 && !s.msb && s.rmask == 0x00ff0000 && s.gmask == 0x0000ff00 && s.bmask == 0x000000ff &&
		stride >= s.img.Width*4 {
		// A 32-bit ARGB visual keeps the alpha the frame painted (the
		// compositor reads it premultiplied); every other visual has the
		// alpha byte forced opaque.
		copyImageRect(s.xbuf, stride, s.img, r, true, !s.frame.Alpha)
		return
	}
	for y := y0; y < y1; y++ {
		row := y * stride
		for x := x0; x < x1; x++ {
			n := s.img.NRGBAAt(x, y)
			i := row + x*bytesPP
			if i+bytesPP > len(s.xbuf) {
				return
			}
			packXPixelN(s.xbuf[i:], bytesPP, n.R, n.G, n.B, n.A, s.msb, s.rmask, s.gmask, s.bmask)
		}
	}
}

func (s *x11Surface) Present(dirty []paintengine2d.Rect) error {
	if s.closed || s.conn == nil || s.conn.dpy == nil {
		return nil
	}
	if s.gpu != nil {
		if err := s.presentGPU(); err == nil {
			x11Mu.Lock()
			if !s.mapped && !s.hidden && s.conn.dpy != nil && s.win != 0 {
				C.ui_map(s.conn.dpy, s.win)
				s.mapped = true
			}
			if s.conn.dpy != nil {
				C.ui_flush(s.conn.dpy)
			}
			x11Mu.Unlock()
			return nil
		}
		// EGL failed this frame — keep the last GPU frame and fall
		// back to XPutImage / MIT-SHM.
		s.abandonGPU()
		x11Mu.Lock()
		if s.ximg == nil {
			s.rebuildImageLocked()
		}
		x11Mu.Unlock()
	}
	if s.ximg == nil {
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
	if !s.mapped && !s.hidden {
		C.ui_map(s.conn.dpy, s.win)
		s.mapped = true
	}
	C.ui_flush(s.conn.dpy)
	return nil
}

func (s *x11Surface) Poll() []Event {
	if s.conn == nil {
		return nil
	}
	if !s.closed {
		s.conn.drain()
	}
	// Drain the queue even when closed: an EventClose (or the events
	// before it) must never be swallowed because the surface changed
	// state between Wait and Poll.
	x11Mu.Lock()
	ev := s.conn.queues[s.win]
	s.conn.queues[s.win] = nil
	x11Mu.Unlock()
	return ev
}

func (s *x11Surface) Wait(timeout time.Duration) bool {
	if s == nil || s.closed || s.conn == nil || s.conn.dpy == nil {
		return false
	}
	n := C.ui_wait(s.conn.dpy, C.int(waitMillis(timeout)))
	return n > 0
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
		if t := C.ui_event_time(&xe); t != 0 {
			c.serverTime = t
		}
		// A drag of ours holds the pointer and the keyboard: the motion,
		// the release and Escape are the drag's, not any widget's — and
		// not the input method's either, which is why this comes before
		// XFilterEvent. A composing XIM swallowed the Escape that was
		// meant to cancel the drag.
		if c.drag.active && c.dragHandle(&xe) {
			continue
		}
		if C.ui_filter(c.dpy, &xe) != 0 {
			continue
		}
		if c.fixesEvent > 0 && int(C.ui_event_type(&xe)) == c.fixesEvent+C.XFixesSelectionNotify {
			c.compositingChangedLocked()
			continue
		}
		switch C.ui_event_type(&xe) {
		case C.ClientMessage:
			if c.handleXdndMessage(&xe) {
				continue
			}
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
			if s := c.surfaces[C.ui_prop_window(&xe)]; s != nil {
				c.queues[s.win] = append(c.queues[s.win], s.propertyChangedLocked(C.ui_prop_atom(&xe))...)
			}
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
	// A target that took a drop and never answered must not hold the
	// drag — and the user's move — open for the rest of the session;
	// and Escape ends a drag even where the grab never delivers it.
	c.dragCheckEscape()
	c.dragTimedOut()
}

func (s *x11Surface) translate(xe *C.XEvent) []Event {
	switch C.ui_event_type(xe) {
	case C.Expose:
		r := paintengine2d.XYWH(float32(C.ui_expose_x(xe)), float32(C.ui_expose_y(xe)), float32(C.ui_expose_w(xe)), float32(C.ui_expose_h(xe)))
		return []Event{{Kind: EventExpose, Pos: r.Min, Width: int(r.Dx()), Height: int(r.Dy())}}
	case C.ConfigureNotify:
		// Where the window manager put the window, for whoever asks
		// ([HostPositioner]): a drag that carries a window needs it, and
		// a saved layout keeps a floating panel's place with it.
		s.posX, s.posY = int(C.ui_cfg_x(xe)), int(C.ui_cfg_y(xe))
		s.posRoot, s.posKnown = C.ui_cfg_synthetic(xe) != 0, true
		w, h := int(C.ui_cfg_w(xe)), int(C.ui_cfg_h(xe))
		if w != s.img.Width || h != s.img.Height {
			if w < 1 {
				w = 1
			}
			if h < 1 {
				h = 1
			}
			s.img = paintengine2d.NewImage(w, h)
			if s.gpu != nil {
				s.resizeGPU(w, h)
			}
			if s.gpu == nil {
				s.rebuildImageLocked()
			}
			// The X window holds the frame's margin too; the app hears
			// about the window inside it, in the logical pixels all
			// window geometry here is stated in.
			m := s.frame.Margin
			s.geomW, s.geomH = max(w-m.Width(), 1), max(h-m.Height(), 1)
			sc := s.conn.displayScale()
			s.geomLW = LogicalPixels(s.geomW, sc)
			s.geomLH = LogicalPixels(s.geomH, sc)
			return []Event{{Kind: EventResize, Width: s.geomLW, Height: s.geomLH}}
		}
	case C.ButtonPress, C.ButtonRelease:
		btn := int(C.ui_btn(xe))
		x, y := float32(C.ui_btn_x(xe)), float32(C.ui_btn_y(xe))
		mods := xmods(uint(C.ui_btn_state(xe)))
		release := C.ui_event_type(xe) == C.ButtonRelease
		if bit := x11ButtonBit(btn); bit != 0 {
			if release {
				s.buttons &^= bit
			} else {
				s.buttons |= bit
				s.pressRootX, s.pressRootY = int(C.ui_btn_xroot(xe)), int(C.ui_btn_yroot(xe))
				s.pressButton = btn
				s.pressTime = C.ui_btn_time(xe)
			}
		}
		return x11ButtonEvents(btn, release, paintengine2d.Pt(x, y), mods)
	case C.LeaveNotify:
		// Grab and ungrab crossings (menus, the implicit button grab
		// ending) are not the pointer leaving the window.
		if C.ui_cross_mode(xe) != C.NotifyNormal {
			return nil
		}
		return []Event{{Kind: EventPointerLeave}}
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
		var isUTF8 C.int
		n := int(C.ui_lookup_im(s.ic, s.conn.dpy, xe, &buf[0], 128, &ks, &isUTF8))
		text := C.GoBytes(unsafe.Pointer(&buf[0]), C.int(max(n, 0)))
		if n < 0 {
			need := -n
			text = nil
			if need > 8 && need < 1<<20 {
				big := make([]C.char, need+1)
				n = int(C.ui_lookup_im(s.ic, s.conn.dpy, xe, &big[0], C.int(need+1), &ks, &isUTF8))
				if n > 0 {
					text = C.GoBytes(unsafe.Pointer(&big[0]), C.int(n))
				}
			} else {
				n = 0
			}
		}
		if isUTF8 == 0 {
			// XLookupString (no input context) yields ISO-8859-1.
			text = latin1ToUTF8(text)
		}
		base := uint64(C.ui_base_sym(s.conn.dpy, xe))
		ev := Event{
			Kind: kind,
			Key:  KeyFromKeysymFallback(uint64(ks), base),
			Mods: xmods(uint(C.ui_key_state(xe))),
		}
		out := []Event{ev}
		if kind == EventKeyDown && len(text) > 0 && !ev.Mods.Ctrl() {
			out = append(out, textEvents(text, ev.Mods)...)
		}
		return out
	case C.FocusIn:
		C.ui_set_ic_focus(s.ic)
		return append([]Event{{Kind: EventFocusIn}}, s.focusChangedLocked(true)...)
	case C.FocusOut:
		C.ui_unset_ic_focus(s.ic)
		out := []Event{{Kind: EventFocusOut}, {Kind: EventIMECancel}}
		if leftover := C.ui_reset_ic(s.ic); leftover != nil {
			txt := C.GoString(leftover)
			C.ui_xfree(unsafe.Pointer(leftover))
			if txt != "" {
				out = append(out, Event{Kind: EventIMECommit, Text: txt})
			}
		}
		return append(out, s.focusChangedLocked(false)...)
	case C.DestroyNotify:
		// The window really is gone (server-side death or our own
		// XDestroyWindow): only here does the surface become Closed.
		s.closed = true
		return []Event{{Kind: EventClose}}
	case C.ClientMessage:
		if C.ui_is_delete(s.conn.dpy, xe) != 0 {
			// WM_DELETE_WINDOW is a *request*. The app may veto it
			// (close-to-tray). Surface.Closed stays false until
			// Close() runs or the window is destroyed.
			return []Event{{Kind: EventClose}}
		}
	}
	return nil
}

func (s *x11Surface) Close() error {
	if s.closed && (s.conn == nil || s.conn.dpy == nil) {
		return nil
	}
	s.closed = true
	s.closeGPU()
	if s.conn != nil {
		s.conn.unregister(s.win)
	}
	x11Mu.Lock()
	// A window that is going cannot finish a drag it started or a drop
	// it was taking; both sides have to be let go of.
	if s.conn != nil {
		if s.conn.drag.active && s.conn.drag.win == s.win {
			s.conn.dragCancel()
		}
		if s.conn.drop.active && s.conn.drop.win == s.win {
			s.conn.xdndEnd(false)
		}
		if s.conn.drop.active && s.conn.drop.source == s.win {
			// The window that started the drag is going with it: taking
			// the drop closed it, which is what a dock panel docking
			// back into its host does. There is nobody left to send
			// XdndFinished to, and sending it anyway is a BadWindow on a
			// window the server has already forgotten.
			s.conn.drop.source = 0
		}
	}
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

// x11ButtonEvents maps a core ButtonPress / ButtonRelease. Buttons 4–7 are
// wheel steps: X sends a press and a release for every notch, and only the
// press may scroll (the release used to scroll a second time).
func x11ButtonEvents(btn int, release bool, pos paintengine2d.Point, mods Modifiers) []Event {
	if btn >= 4 && btn <= 7 {
		if release {
			return nil
		}
		ev := Event{Kind: EventScroll, Pos: pos, Mods: mods}
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
	kind := EventMouseDown
	if release {
		kind = EventMouseUp
	}
	return []Event{{Kind: kind, Pos: pos, Button: xbutton(btn), Mods: mods}}
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
	case C.XK_3:
		return Key3
	case C.XK_numbersign:
		return KeyHash
	case C.XK_comma:
		return KeyComma
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
	case C.XK_Menu:
		return KeyMenu
	case C.XK_Alt_L, C.XK_Alt_R, C.XK_Meta_L, C.XK_Meta_R:
		return KeyAlt
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
	// A drag of ours owns XdndSelection while it runs and answers for
	// it itself; the clipboard knows nothing about that selection.
	if c.drag.active && C.ui_sr_selection(xe) == c.xdnd.at(XASelection) {
		if c.dragSelRequest(xe) {
			return
		}
	}
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
		// INCR is a transfer *type*, never a target: advertising it let
		// requestors ask for a target we cannot convert. TIMESTAMP is
		// required of every ICCCM selection owner.
		atoms := []C.ulong{
			C.ulong(c.atomTargets), C.ulong(c.atomTimestamp),
			C.ulong(c.atomUTF8), C.ulong(c.atomString), C.ulong(c.atomText),
		}
		C.ui_change_prop32(c.dpy, req, prop, C.XA_ATOM, &atoms[0], C.int(len(atoms)))
		C.ui_send_sel_notify(c.dpy, xe, prop)
	case c.atomTimestamp:
		ts := []C.ulong{C.ulong(c.ownTime)}
		C.ui_change_prop32(c.dpy, req, prop, C.XA_INTEGER, &ts[0], 1)
		C.ui_send_sel_notify(c.dpy, xe, prop)
	case c.atomUTF8, c.atomString, c.atomText:
		// TEXT is a polymorphic target: the reply must name a concrete
		// type, not echo TEXT back at the requestor.
		replyType := target
		if replyType == c.atomText {
			replyType = c.atomUTF8
		}
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
				typ:       replyType,
				data:      raw,
				chunk:     INCRChunkSize(thr),
			})
			c.keep = true
			return
		}
		ct := C.CString(data)
		C.ui_change_prop8(c.dpy, req, prop, replyType, ct, C.int(len(data)))
		C.free(unsafe.Pointer(ct))
		C.ui_send_sel_notify(c.dpy, xe, prop)
	default:
		C.ui_send_sel_notify(c.dpy, xe, 0)
	}
}

func (c *x11Conn) handleSelClear(xe *C.XEvent) {
	sel := C.ui_sc_selection(xe)
	if c.drag.active && sel == c.xdnd.at(XASelection) {
		// Another client took the drag selection: ours cannot complete.
		c.dragCancel()
		return
	}
	if sel == c.atomClipboard {
		c.ownClip = false
	}
	if sel == c.atomPrimary {
		c.ownPrim = false
	}
	if !c.ownClip && !c.ownPrim && len(c.incrSends) == 0 {
		c.keep = false
		c.clipText = ""
	}
}

func (c *x11Conn) handleSelNotify(xe *C.XEvent) {
	if c.xdndSelNotify(xe) {
		return
	}
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
		// A drop's data arrives in the same pieces as a paste's, into
		// its own buffer.
		if c.xdndIncrPiece(piece) {
			return
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
	t := c.selectionTimeLocked()
	c.ownTime = t
	C.ui_set_owner(c.dpy, c.helper, c.atomClipboard, t)
	C.ui_set_owner(c.dpy, c.helper, c.atomPrimary, t)
	c.ownClip = C.ui_owner(c.dpy, c.atomClipboard) == c.helper
	c.ownPrim = C.ui_owner(c.dpy, c.atomPrimary) == c.helper
	c.keep = c.ownClip || c.ownPrim
	C.ui_flush(c.dpy)
	c.startSelectionServiceLocked()
}

// selectionTimeLocked returns a real server timestamp for selection
// ownership and conversion. ICCCM forbids CurrentTime there: with
// CurrentTime an owner cannot tell a stale request from a fresh one.
// When no timestamped event has arrived yet, a zero-length property
// append on the helper window produces one.
func (c *x11Conn) selectionTimeLocked() C.Time {
	if c.serverTime != 0 {
		return c.serverTime
	}
	if c.dpy == nil || c.helper == 0 {
		return 0 // CurrentTime
	}
	C.ui_touch_prop(c.dpy, c.helper, c.atomProp)
	deadline := time.Now().Add(100 * time.Millisecond)
	for c.serverTime == 0 && time.Now().Before(deadline) {
		c.drainLocked()
		if c.serverTime != 0 {
			break
		}
		dpy := c.dpy
		x11Mu.Unlock()
		C.ui_wait(dpy, 5)
		x11Mu.Lock()
		if c.dpy == nil {
			return 0
		}
	}
	return c.serverTime
}

// startSelectionServiceLocked keeps answering SelectionRequest after the
// last window is gone. An X selection owner that never reads its
// connection makes every other application's paste hang until their own
// timeout: ownership is a promise to serve the data. The goroutine exits
// as soon as a window loop takes over (refs > 0) or ownership is lost.
func (c *x11Conn) startSelectionServiceLocked() {
	if c.servicing || c.refs > 0 || !c.keep || c.dpy == nil {
		return
	}
	c.servicing = true
	go c.serviceSelections()
}

func (c *x11Conn) serviceSelections() {
	for {
		x11Mu.Lock()
		if c.dpy == nil || c.refs > 0 || !c.keep {
			c.servicing = false
			done := c.dpy != nil && c.refs == 0 && !c.keep
			c.mayCloseLocked(done)
			x11Mu.Unlock()
			return
		}
		c.drainLocked()
		dpy := c.dpy
		x11Mu.Unlock()
		if dpy == nil {
			return
		}
		C.ui_wait(dpy, 50)
	}
}

// mayCloseLocked closes an idle connection nobody is using any more.
func (c *x11Conn) mayCloseLocked(ok bool) {
	if ok && c.refs == 0 && !c.keep && !c.servicing {
		c.closeLocked()
	}
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
	C.ui_convert(c.dpy, c.helper, sel, c.atomUTF8, c.atomProp, c.selectionTimeLocked())
	C.ui_flush(c.dpy)
	deadline := time.Now().Add(timeout)
	for !c.pasteDone && time.Now().Before(deadline) {
		c.drainLocked()
		if c.pasteDone {
			break
		}
		dpy := c.dpy
		x11Mu.Unlock()
		C.ui_wait(dpy, 5)
		x11Mu.Lock()
		if c.dpy == nil {
			x11Mu.Unlock()
			return "", false
		}
	}
	if !c.pasteDone && c.dpy != nil && !c.incrRecv.active {
		c.pasteDone = false
		C.ui_convert(c.dpy, c.helper, sel, c.atomString, c.atomProp, c.selectionTimeLocked())
		C.ui_flush(c.dpy)
		end := time.Now().Add(timeout / 2)
		for !c.pasteDone && time.Now().Before(end) {
			c.drainLocked()
			if c.pasteDone {
				break
			}
			dpy := c.dpy
			x11Mu.Unlock()
			C.ui_wait(dpy, 5)
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

func (c *x11Conn) xcursor(cur Cursor) C.Cursor {
	if c == nil || c.dpy == nil {
		return 0
	}
	if c.cursors == nil {
		c.cursors = map[Cursor]C.Cursor{}
	}
	if got := c.cursors[cur]; got != 0 {
		return got
	}
	for _, name := range x11ThemeCursorNames(cur) {
		cs := C.CString(name)
		got := C.ui_theme_cursor(c.dpy, cs)
		C.free(unsafe.Pointer(cs))
		if got != 0 {
			c.cursors[cur] = got
			return got
		}
	}
	got := C.ui_font_cursor(c.dpy, C.uint(x11FontCursorShape(cur)))
	c.cursors[cur] = got
	return got
}

func (s *x11Surface) SetCursor(cur Cursor) {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_define_cursor(s.conn.dpy, s.win, s.conn.xcursor(cur))
	x11Mu.Unlock()
}

func (s *x11Surface) Raise() {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_raise(s.conn.dpy, s.win)
	if s.popup {
		C.ui_focus(s.conn.dpy, s.win)
	}
	s.mapped = true
	s.hidden = false
	x11Mu.Unlock()
}

func (s *x11Surface) Move(x, y int) {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	// The caller places the window the user sees; the X window is that
	// plus the margin the frame's shadow lives in.
	C.ui_move(s.conn.dpy, s.win, C.int(x-s.frame.Margin.Left), C.int(y-s.frame.Margin.Top))
	x11Mu.Unlock()
}

// Position implements [HostPositioner]: where the window manager has put
// the window, in root coordinates, counting from the window the user sees
// rather than from the margin around it. It is the last ConfigureNotify's
// answer, translated to the root here when the manager's configure was
// frame-relative, so an interactive move costs no round trip.
func (s *x11Surface) Position() (int, int, bool) {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return 0, 0, false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if s.closed {
		return 0, 0, false
	}
	x, y := s.posX, s.posY
	if !s.posKnown || !s.posRoot {
		var rx, ry C.int
		if C.ui_to_root(s.conn.dpy, s.win, 0, 0, &rx, &ry) == 0 && !s.posKnown {
			return 0, 0, false
		}
		x, y = int(rx), int(ry)
	}
	return x + s.frame.Margin.Left, y + s.frame.Margin.Top, true
}

func (s *x11Surface) Wake() {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.conn.helper == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_wake(s.conn.dpy, s.conn.helper)
	x11Mu.Unlock()
}

// x11DragPoll is how often the loop wakes while a drag of ours runs.
const x11DragPoll = 40 * time.Millisecond

// WakeAt implements [WakeScheduler]: a drag of ours keeps the loop
// ticking even while the connection is idle. Both of the drag's safety
// nets need it — the key state is polled for an Escape that never
// arrived as a key event (a compositor's own XWayland drag bridge can
// hold the keyboard), and a target that took the drop and never answered
// has to time out — and a pointer held still delivers nothing to wake on.
func (s *x11Surface) WakeAt() time.Time {
	if s == nil || s.conn == nil {
		return time.Time{}
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if !s.conn.drag.active {
		return time.Time{}
	}
	return time.Now().Add(x11DragPoll)
}

func (s *x11Surface) Show() { s.Raise() }

func (s *x11Surface) Hide() {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_unmap(s.conn.dpy, s.win)
	s.mapped = false
	s.hidden = true
	x11Mu.Unlock()
}

func (s *x11Surface) Visible() bool {
	if s == nil || s.closed {
		return false
	}
	return s.mapped
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

func (s *x11Surface) SetIMEEnabled(on bool) {
	// XIC is created with the window; focus is enough for XIM.
	_ = on
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

// PortalParent is the window's id for the desktop portal ("x11:<hex id>"),
// so the desktop's dialogs open as its children.
func (s *x11Surface) PortalParent() string {
	if s == nil || s.win == 0 {
		return ""
	}
	return fmt.Sprintf("x11:%x", uint64(s.win))
}

// ---- client-side frame support (FrameSurface) ----------------------------

// x11ButtonBit is a core pointer button's bit in x11Surface.buttons (wheel
// "buttons" 4-7 are not held buttons).
func x11ButtonBit(btn int) uint8 {
	switch btn {
	case 1:
		return 1
	case 3:
		return 2
	case 2:
		return 4
	case 8, 9:
		return 8
	}
	return 0
}

// supportsLocked reports whether the window manager lists atom in
// _NET_SUPPORTED (read once per connection).
func (c *x11Conn) supportsLocked(atom C.Atom) bool {
	if c == nil || c.dpy == nil {
		return false
	}
	if c.supported == nil {
		c.supported = map[C.Atom]bool{}
		var buf [512]C.Atom
		n := int(C.ui_get_atoms(c.dpy, C.XDefaultRootWindow(c.dpy), c.atomSupported, &buf[0], C.int(len(buf))))
		for _, a := range buf[:n] {
			c.supported[a] = true
		}
		var name [128]C.char
		if C.ui_wm_name(c.dpy, &name[0], C.int(len(name))) > 0 {
			c.wmName = C.GoString(&name[0])
		}
	}
	return c.supported[atom]
}

// setMotifLocked writes (or removes) _MOTIF_WM_HINTS for the requested
// decoration mode.
func (s *x11Surface) setMotifLocked() {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return
	}
	hints, ok := motifHints(s.wantDeco)
	if !ok {
		C.ui_delete_prop(c.dpy, s.win, c.atomMotif)
		C.ui_flush(c.dpy)
		return
	}
	var v [5]C.ulong
	for i, h := range hints {
		v[i] = C.ulong(h)
	}
	// The type must be _MOTIF_WM_HINTS itself or KWin ignores it.
	C.ui_change_prop32(c.dpy, s.win, c.atomMotif, c.atomMotif, &v[0], 5)
	C.ui_flush(c.dpy)
}

// compositingChangedLocked follows a compositing manager starting or
// stopping: every window hears about it, so a frame that was translucent
// goes solid (or the other way round) at once, as GTK's .solid-csd does.
func (c *x11Conn) compositingChangedLocked() {
	now := C.ui_composited(c.dpy) != 0
	if now == c.composited {
		return
	}
	c.composited = now
	for win, s := range c.surfaces {
		if s == nil || s.closed {
			continue
		}
		c.queues[win] = append(c.queues[win], s.readStateLocked()...)
	}
}

// focusChangedLocked follows keyboard focus as the active state where the
// window manager does not maintain _NET_WM_STATE_FOCUSED.
func (s *x11Surface) focusChangedLocked(in bool) []Event {
	s.focused = in
	if s.conn == nil || s.conn.supportsLocked(s.conn.atomFocused) {
		return nil
	}
	return s.readStateLocked()
}

// propertyChangedLocked follows the window manager's properties on the
// window.
func (s *x11Surface) propertyChangedLocked(atom C.Atom) []Event {
	switch atom {
	case s.conn.atomNetState:
		return s.readStateLocked()
	case s.conn.atomAllowed:
		return s.readCapsLocked()
	}
	return nil
}

// readStateLocked re-reads _NET_WM_STATE and reports a change.
func (s *x11Surface) readStateLocked() []Event {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return nil
	}
	var buf [32]C.Atom
	n := int(C.ui_get_atoms(c.dpy, s.win, c.atomNetState, &buf[0], C.int(len(buf))))
	var bits uint32
	for _, a := range buf[:n] {
		switch a {
		case c.atomMaxVert:
			bits |= netStateMaxVert
		case c.atomMaxHorz:
			bits |= netStateMaxHorz
		case c.atomFullscr:
			bits |= netStateFullscreen
		case c.atomHidden:
			bits |= netStateHidden
		case c.atomFocused:
			bits |= netStateFocused
		}
	}
	st := netWMState(bits, c.supportsLocked(c.atomFocused), s.focused)
	// Without a compositing manager the X server ignores alpha: a frame
	// the toolkit draws has to be solid there.
	st.Solid = !c.composited
	if st == s.state {
		return nil
	}
	s.state = st
	return []Event{{Kind: EventWindowState, State: st}}
}

// readCapsLocked re-reads _NET_WM_ALLOWED_ACTIONS and reports a change.
func (s *x11Surface) readCapsLocked() []Event {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return nil
	}
	var buf [64]C.Atom
	n := int(C.ui_get_atoms(c.dpy, s.win, c.atomAllowed, &buf[0], C.int(len(buf))))
	var bits uint32
	for _, a := range buf[:n] {
		switch a {
		case c.atomActMin:
			bits |= netActionMinimize
		case c.atomActMaxH:
			bits |= netActionMaximizeHorz
		case c.atomActMaxV:
			bits |= netActionMaximizeVert
		case c.atomActFull:
			bits |= netActionFullscreen
		}
	}
	caps := netAllowedCaps(bits, c.supportsLocked(c.atomShowMenu))
	if caps == s.caps {
		return nil
	}
	s.caps = caps
	return []Event{{Kind: EventCapabilities, Caps: caps}}
}

// Decorations is the requested mode: an X11 window manager does not answer
// (FrameSurface).
func (s *x11Surface) Decorations() Decorations {
	if s.popup {
		return DecorationsNone
	}
	return s.wantDeco
}

// RequestDecorations sets or removes _MOTIF_WM_HINTS; the window manager
// adds or drops its frame (FrameSurface).
func (s *x11Surface) RequestDecorations(d Decorations) {
	if s == nil || s.popup || s.closed {
		return
	}
	d = requestedDecorations(d)
	if d == s.wantDeco {
		return
	}
	s.wantDeco = d
	x11Mu.Lock()
	s.setMotifLocked()
	x11Mu.Unlock()
	if s.conn != nil {
		x11Mu.Lock()
		s.conn.queues[s.win] = append(s.conn.queues[s.win], Event{Kind: EventDecorations, Decor: d})
		x11Mu.Unlock()
	}
}

// WindowState is the window manager's _NET_WM_STATE, decoded (FrameSurface).
func (s *x11Surface) WindowState() WindowState { return s.state }

// Capabilities are the window's _NET_WM_ALLOWED_ACTIONS (FrameSurface).
// Capabilities is what _NET_WM_ALLOWED_ACTIONS says, less what a fixed
// window cannot have (FrameSurface).
func (s *x11Surface) Capabilities() WMCaps { return dropResizeCaps(s.caps, s.sizing) }

// Sizing is the window's resize policy (SizingSurface).
func (s *x11Surface) Sizing() Sizing { return s.sizing }

// SetSizing changes the policy and re-states WM_NORMAL_HINTS
// (SizingSurface).
func (s *x11Surface) SetSizing(sz Sizing) {
	if s == nil || s.sizing == sz {
		return
	}
	s.sizing = sz
	s.syncLimits()
}

// SizeLimits is what WM_NORMAL_HINTS last said about the visible window
// (SizingSurface).
func (s *x11Surface) SizeLimits() SizeLimits { return s.limits }

// syncLimits recomputes the window's limits for its size and hands them to
// the window manager.
func (s *x11Surface) syncLimits() {
	if s == nil {
		return
	}
	s.limits = limitsFor(s.sizing, s.opts, s.geomLW, s.geomLH)
	x11Mu.Lock()
	s.applySizeHintsLocked()
	x11Mu.Unlock()
}

// applySizeHintsLocked publishes the window's limits as WM_NORMAL_HINTS.
//
// The limits are logical pixels, as all window geometry here is; an X
// server's are device pixels, so each one is scaled. The hints are about
// the X window, which holds the frame's margin too: without adding it the
// window manager would let the visible window shrink past the app's
// minimum by the shadow's width, and would hold a fixed window at the
// margin's size short of its own (GTK adds it as well).
func (s *x11Surface) applySizeHintsLocked() {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return
	}
	l := s.limits
	m := s.frame.Margin
	sc := c.displayScale()
	grow := func(v, margin int) C.int {
		if v <= 0 {
			return 0
		}
		return C.int(DevicePixels(v, sc) + margin)
	}
	C.ui_resize_hints(c.dpy, s.win,
		grow(l.MinWidth, m.Width()), grow(l.MinHeight, m.Height()),
		grow(l.MaxWidth, m.Width()), grow(l.MaxHeight, m.Height()))
}

// SuitsClientFrame: the window manager moves and resizes on request
// (_NET_WM_MOVERESIZE) and is not a tiling manager.
func (s *x11Surface) SuitsClientFrame() bool {
	if s == nil || s.conn == nil {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return s.conn.supportsLocked(s.conn.atomMoveRes) && !tilingWM(s.conn.wmName)
}

// moveResizeLocked sends _NET_WM_MOVERESIZE for the held press.
func (s *x11Surface) moveResize(dir uint32) bool {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 || s.buttons == 0 {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if !c.supportsLocked(c.atomMoveRes) {
		return false
	}
	C.ui_ungrab_pointer(c.dpy)
	// Source 1: a normal application.
	C.ui_root_message(c.dpy, s.win, c.atomMoveRes, C.long(s.pressRootX), C.long(s.pressRootY), C.long(dir), C.long(s.pressButton), 1)
	// The window manager owns the pointer now; the release goes to it.
	s.buttons = 0
	return true
}

// StartSystemMove hands the held press to the window manager for an
// interactive move (_NET_WM_MOVERESIZE, MOVE), from where it was pressed.
func (s *x11Surface) StartSystemMove() bool { return s.moveResize(netMoveResizeMove) }

// StartSystemResize hands the held press to the window manager for an
// interactive resize from edges.
func (s *x11Surface) StartSystemResize(edges Edges) bool {
	if s.sizing == SizingFixed {
		return false
	}
	dir, ok := netMoveResizeDirection(edges)
	if !ok {
		return false
	}
	return s.moveResize(dir)
}

// ShowWindowMenu asks the window manager for its window menu at p (window
// device pixels) with _GTK_SHOW_WINDOW_MENU, where it is supported.
func (s *x11Surface) ShowWindowMenu(p paintengine2d.Point) bool {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	if !c.supportsLocked(c.atomShowMenu) {
		return false
	}
	var rx, ry C.int
	if C.ui_to_root(c.dpy, s.win, C.int(p.X), C.int(p.Y), &rx, &ry) == 0 {
		return false
	}
	C.ui_ungrab_pointer(c.dpy)
	// {device id, x_root, y_root}; the core pointer's device when unknown.
	C.ui_root_message(c.dpy, s.win, c.atomShowMenu, 0, C.long(rx), C.long(ry), 0, 0)
	s.buttons = 0
	return true
}

// SetFrame takes the frame the toolkit draws (FrameSurface): the X window
// grows by the margin, _GTK_FRAME_EXTENTS tells the window manager how much
// of it is shadow (KWin and Mutter honour it when snapping, tiling and
// maximizing), an input shape keeps presses in the margin out of the
// window, and _NET_WM_OPAQUE_REGION says what of it is solid.
//
// A translucent frame needs a 32-bit visual, which an X window is born
// with: one that has to change is re-created (invisibly before the window
// is first mapped, where a frame is normally decided).
func (s *x11Surface) SetFrame(f Frame) {
	if s == nil || s.closed || s.win == 0 || f.Same(s.frame) {
		return
	}
	if f.Alpha && !s.argb() {
		s.recreateOnVisual(true)
	}
	// A window that stops needing alpha keeps its 32-bit visual: every
	// pixel is opaque then (and _NET_WM_OPAQUE_REGION says so), and
	// re-creating the window to go back would flash for nothing.
	s.frame = f
	x11Mu.Lock()
	s.applyFrameLocked()
	x11Mu.Unlock()
	w, h := s.geomW+f.Margin.Width(), s.geomH+f.Margin.Height()
	if s.img != nil && (s.img.Width != w || s.img.Height != h) {
		s.img = paintengine2d.NewImage(max(w, 1), max(h, 1))
		x11Mu.Lock()
		if s.gpu != nil {
			s.resizeGPU(w, h)
		}
		if s.gpu == nil {
			s.rebuildImageLocked()
		}
		if s.conn != nil && s.conn.dpy != nil {
			C.ui_resize_win(s.conn.dpy, s.win, C.int(w), C.int(h))
		}
		x11Mu.Unlock()
	}
}

// Frame is the frame last set (FrameSurface).
func (s *x11Surface) Frame() Frame {
	if s == nil {
		return Frame{}
	}
	return s.frame
}

// argb reports whether the window was created on a visual with an alpha
// channel.
func (s *x11Surface) argb() bool { return s != nil && s.vis != nil }

// recreateOnVisual makes the window again on a 32-bit visual (argb) or on
// the screen's default one. An X window's visual is fixed when it is
// created, and a frame with a shadow needs an alpha channel, so the window
// is destroyed and made afresh with everything on it: its properties, its
// input context, its graphics context and its GPU device.
//
// Before the first map — where a frame is normally decided, the title bar
// being set after the window is made — nothing shows. A mapped window is
// re-created at the place it stood.
func (s *x11Surface) recreateOnVisual(argb bool) {
	c := s.conn
	if s == nil || c == nil || c.dpy == nil || s.win == 0 || s.popup {
		return
	}
	if argb == s.argb() {
		return
	}
	var vis *C.Visual
	var cmap C.Colormap
	depth := C.int(0)
	if argb {
		if !c.composited {
			return
		}
		if C.ui_argb_visual(c.dpy, &vis, &cmap, &depth) == 0 {
			return
		}
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	mapped := s.mapped
	x, y := s.opts.X, s.opts.Y
	if mapped {
		var rx, ry C.int
		if C.ui_to_root(c.dpy, s.win, 0, 0, &rx, &ry) != 0 {
			x, y = int(rx), int(ry)
		}
	}
	w, h := s.img.Width, s.img.Height
	// Everything that hangs off the old window goes with it.
	s.closeGPU()
	if s.ic != nil {
		C.ui_destroy_ic(s.ic)
		s.ic = nil
	}
	if s.ximCbs != nil {
		x11FreeCbs(s.ximCbs)
		s.ximCbs = nil
	}
	s.destroyImageLocked()
	old := s.win
	delete(c.surfaces, old)
	delete(c.queues, old)
	C.ui_destroy_win(c.dpy, old, s.gc)
	s.gc = nil
	if s.cmap != 0 {
		C.ui_free_colormap(c.dpy, s.cmap)
		s.cmap = 0
	}
	ctitle := C.CString(s.title)
	popup := C.int(0)
	s.win = C.ui_create_visual(c.dpy, C.int(x), C.int(y), C.int(w), C.int(h), ctitle, popup, vis, cmap, depth)
	C.free(unsafe.Pointer(ctitle))
	s.vis, s.cmap, s.visDepth = vis, cmap, int(depth)
	if vis != nil {
		s.rmask = uint32(C.ui_vis_red(vis))
		s.gmask = uint32(C.ui_vis_green(vis))
		s.bmask = uint32(C.ui_vis_blue(vis))
	} else {
		s.rmask = uint32(C.ui_red_mask(c.dpy))
		s.gmask = uint32(C.ui_green_mask(c.dpy))
		s.bmask = uint32(C.ui_blue_mask(c.dpy))
	}
	s.applySizeHintsLocked()
	s.gc = C.ui_gc(c.dpy, s.win)
	s.ic, s.ximCbs = x11CreateIC(c.im, s.win)
	c.surfaces[s.win] = s
	s.setMotifLocked()
	// A new window on another visual is a new X window: it takes drops
	// only once it says so again.
	s.setXdndAwareLocked()
	s.mapped = false
	s.rebuildImageLocked()
	if mapped && !s.hidden {
		C.ui_map(c.dpy, s.win)
		s.mapped = true
	}
	// The GPU device is bound to the window: a new one for the new visual,
	// with an alpha channel where the frame needs it.
	x11Mu.Unlock()
	s.tryBindGPU()
	x11Mu.Lock()
}

// applyFrameLocked publishes the frame's geometry: the shadow margin, the
// input shape and the opaque region.
func (s *x11Surface) applyFrameLocked() {
	c := s.conn
	if c == nil || c.dpy == nil || s.win == 0 {
		return
	}
	m := s.frame.Margin
	s.applySizeHintsLocked()
	if s.frame.Shape != nil {
		s.applyShapeLocked()
		s.applyBlurLocked()
		C.ui_flush(c.dpy)
		return
	}
	if s.shaped {
		// The shape went away (maximized, tiled, or the app dropped it):
		// every pixel of the window is the window's again.
		s.shaped = false
		C.ui_shape_none(c.dpy, s.win, 1)
	}
	s.applyBlurLocked()
	if m.Zero() {
		C.ui_delete_prop(c.dpy, s.win, c.atomExtents)
		C.ui_shape_input_all(c.dpy, s.win)
	} else {
		ext := [4]C.ulong{C.ulong(m.Left), C.ulong(m.Right), C.ulong(m.Top), C.ulong(m.Bottom)}
		C.ui_change_prop32(c.dpy, s.win, c.atomExtents, C.XA_CARDINAL, &ext[0], 4)
		in := InputRect(FrameRect{X: m.Left, Y: m.Top, W: s.geomW, H: s.geomH}, m, s.frame.Input)
		C.ui_shape_input(c.dpy, s.win, C.int(in.X), C.int(in.Y), C.int(in.W), C.int(in.H))
	}
	// The opaque region is the window less its rounded corners, in window
	// (X) coordinates: a compositor that knows what is solid need not
	// blend it.
	rects := s.opaqueRects()
	if len(rects) == 0 {
		C.ui_delete_prop(c.dpy, s.win, c.atomOpaque)
	} else {
		C.ui_change_prop32(c.dpy, s.win, c.atomOpaque, C.XA_CARDINAL, &rects[0], C.int(len(rects)))
	}
	C.ui_flush(c.dpy)
}

// applyShapeLocked states a shaped window's silhouette. X11 has two shapes
// and a shaped window wants both: ShapeInput decides where a press lands,
// so a hole takes no clicks and the press reaches whatever is behind;
// ShapeBounding decides which pixels the window has at all, which is what
// makes the hole *visible* on a screen with no compositing manager, where
// the ARGB buffer's transparency counts for nothing. With a compositing
// manager both are set and KWin honours both, so the fallback needs no
// separate code path — only a hard edge instead of an antialiased one.
//
// XShape works in device pixels, which is what a Frame already holds, so
// unlike Wayland there is no conversion here.
func (s *x11Surface) applyShapeLocked() {
	c := s.conn
	flat := x11FlatRects(s.frame.Shape)
	var p *C.int
	if len(flat) > 0 {
		p = &flat[0]
	}
	C.ui_shape_rects(c.dpy, s.win, 0, p, C.int(len(flat)/4))
	C.ui_shape_rects(c.dpy, s.win, 1, p, C.int(len(flat)/4))
	s.shaped = true
	// The margin is still the shadow band the window manager should ignore
	// when it places the window, shape or no shape.
	if m := s.frame.Margin; m.Zero() {
		C.ui_delete_prop(c.dpy, s.win, c.atomExtents)
	} else {
		ext := [4]C.ulong{C.ulong(m.Left), C.ulong(m.Right), C.ulong(m.Top), C.ulong(m.Bottom)}
		C.ui_change_prop32(c.dpy, s.win, c.atomExtents, C.XA_CARDINAL, &ext[0], 4)
	}
	rects := x11ULongRects(s.frame.Opaque)
	if len(rects) == 0 {
		C.ui_delete_prop(c.dpy, s.win, c.atomOpaque)
	} else {
		C.ui_change_prop32(c.dpy, s.win, c.atomOpaque, C.XA_CARDINAL, &rects[0], C.int(len(rects)))
	}
}

// applyBlurLocked states the blur-behind region as
// _KDE_NET_WM_BLUR_BEHIND_REGION, a flat list of x, y, width, height in
// window coordinates. KWin is the compositor that reads it; anything else
// ignores the property, which is the fallback working as intended.
func (s *x11Surface) applyBlurLocked() {
	c := s.conn
	if c.atomBlur == 0 {
		return
	}
	if s.frame.Blur == nil {
		C.ui_delete_prop(c.dpy, s.win, c.atomBlur)
		return
	}
	rects := x11ULongRects(s.frame.Blur)
	if len(rects) == 0 {
		// An empty property is how KWin is told "the whole window".
		C.ui_change_prop32(c.dpy, s.win, c.atomBlur, C.XA_CARDINAL, nil, 0)
		return
	}
	C.ui_change_prop32(c.dpy, s.win, c.atomBlur, C.XA_CARDINAL, &rects[0], C.int(len(rects)))
}

// BlurBehindSupported reports whether the desktop blurs behind a window: on
// X11 that is a compositing manager being there at all (KWin reads the
// property; a plain X screen has no one to do it). The toolkit already
// watches the compositing manager come and go (phase 3) (GlassSurface).
func (s *x11Surface) BlurBehindSupported() bool {
	return s != nil && s.conn != nil && s.conn.composited && s.conn.atomBlur != 0
}

// x11FlatRects is rects as the flat x, y, w, h list the C helpers take.
func x11FlatRects(rects []FrameRect) []C.int {
	out := make([]C.int, 0, len(rects)*4)
	for _, r := range rects {
		out = append(out, C.int(r.X), C.int(r.Y), C.int(r.W), C.int(r.H))
	}
	return out
}

// x11ULongRects is rects as the flat CARDINAL list an X property takes.
func x11ULongRects(rects []FrameRect) []C.ulong {
	out := make([]C.ulong, 0, len(rects)*4)
	for _, r := range rects {
		out = append(out, C.ulong(r.X), C.ulong(r.Y), C.ulong(r.W), C.ulong(r.H))
	}
	return out
}

// opaqueRects is the window less the squares its rounded corners live in,
// as _NET_WM_OPAQUE_REGION's flat list of x, y, width, height (device
// pixels: X11 has no logical ones).
func (s *x11Surface) opaqueRects() []C.ulong {
	m := s.frame.Margin
	box := FrameRect{X: m.Left, Y: m.Top, W: s.geomW, H: s.geomH}
	var out []C.ulong
	for _, r := range OpaqueRects(box, s.frame.Radius, 1) {
		out = append(out, C.ulong(r.X), C.ulong(r.Y), C.ulong(r.W), C.ulong(r.H))
	}
	return out
}

// Minimize iconifies the window (XIconifyWindow).
func (s *x11Surface) Minimize() {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_iconify(s.conn.dpy, s.win)
	x11Mu.Unlock()
}

// Lower puts the window below the others (a title-bar action that only X11
// offers).
func (s *x11Surface) Lower() {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	x11Mu.Lock()
	C.ui_lower(s.conn.dpy, s.win)
	x11Mu.Unlock()
}

// ToggleMaximizeAxis maximizes (or restores) one way only: KWin's "Maximize
// (vertical only)" title-bar action, which only X11 can express.
func (s *x11Surface) ToggleMaximizeAxis(vertical bool) {
	if s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	atom := s.conn.atomMaxHorz
	if vertical {
		atom = s.conn.atomMaxVert
	}
	x11Mu.Lock()
	C.ui_ewmh_state(s.conn.dpy, s.win, 2, atom, 0) // _NET_WM_STATE_TOGGLE
	x11Mu.Unlock()
}

// x11Surface is a full FrameSurface (compile-time check).
var _ FrameSurface = (*x11Surface)(nil)
