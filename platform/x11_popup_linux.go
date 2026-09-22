//go:build linux && cgo

package platform

/*
#cgo linux LDFLAGS: -lX11 -lXrandr
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <X11/extensions/Xrandr.h>
#include <string.h>

// ui_pop_create makes a popup's X window: override-redirect, so no window
// manager frames, places or focuses it; transient for the window it belongs
// to; typed as a menu or a tooltip for a compositor's effects; and, on a
// 32-bit visual, fully transparent until the first frame is put.
static Window ui_pop_create(Display* d, Window parent, int x, int y, int w, int h,
		Visual* vis, Colormap cmap, int depth, int tooltip) {
	int s = DefaultScreen(d);
	XSetWindowAttributes swa;
	memset(&swa, 0, sizeof(swa));
	swa.background_pixel = 0;
	swa.border_pixel = 0;
	swa.colormap = vis ? cmap : DefaultColormap(d, s);
	swa.override_redirect = True;
	swa.save_under = True;
	swa.event_mask = ExposureMask|ButtonPressMask|ButtonReleaseMask|PointerMotionMask|
		LeaveWindowMask|StructureNotifyMask;
	unsigned long mask = CWBackPixel|CWBorderPixel|CWColormap|CWOverrideRedirect|CWSaveUnder|CWEventMask;
	Window win = XCreateWindow(d, RootWindow(d, s), x, y, (unsigned)(w > 0 ? w : 1), (unsigned)(h > 0 ? h : 1), 0,
		vis ? depth : DefaultDepth(d, s), InputOutput, vis ? vis : DefaultVisual(d, s), mask, &swa);
	Atom wtype = XInternAtom(d, "_NET_WM_WINDOW_TYPE", False);
	Atom kind = XInternAtom(d, tooltip ? "_NET_WM_WINDOW_TYPE_TOOLTIP" : "_NET_WM_WINDOW_TYPE_POPUP_MENU", False);
	XChangeProperty(d, win, wtype, XA_ATOM, 32, PropModeReplace, (unsigned char*)&kind, 1);
	XSetTransientForHint(d, win, parent);
	XClassHint ch;
	ch.res_name = "uitoolkit";
	ch.res_class = "Uitoolkit";
	XSetClassHint(d, win, &ch);
	return win;
}

static GC ui_pop_gc(Display* d, Window w) { return XCreateGC(d, w, 0, NULL); }

// ui_pop_grab takes the pointer and the keyboard for a menu, owner-events,
// so the application's own windows still hear their own events and every
// press anywhere else comes to the menu — which is how a click outside
// takes it down. It reports 1 only when both were granted.
static int ui_pop_grab(Display* d, Window w) {
	int p = XGrabPointer(d, w, True,
		ButtonPressMask|ButtonReleaseMask|PointerMotionMask|LeaveWindowMask,
		GrabModeAsync, GrabModeAsync, None, None, CurrentTime);
	int k = XGrabKeyboard(d, w, True, GrabModeAsync, GrabModeAsync, CurrentTime);
	if (p != GrabSuccess || k != GrabSuccess) {
		if (p == GrabSuccess) XUngrabPointer(d, CurrentTime);
		if (k == GrabSuccess) XUngrabKeyboard(d, CurrentTime);
		XFlush(d);
		return 0;
	}
	XFlush(d);
	return 1;
}

static void ui_pop_ungrab(Display* d) {
	XUngrabPointer(d, CurrentTime);
	XUngrabKeyboard(d, CurrentTime);
	XFlush(d);
}

static void ui_pop_map(Display* d, Window w) {
	XMapRaised(d, w);
	XFlush(d);
}

static void ui_pop_move_resize(Display* d, Window w, int x, int y, int width, int height) {
	XMoveResizeWindow(d, w, x, y, (unsigned)(width > 0 ? width : 1), (unsigned)(height > 0 ? height : 1));
	XFlush(d);
}

static int ui_pop_origin(Display* d, Window w, int* x, int* y) {
	Window child = 0;
	return XTranslateCoordinates(d, w, DefaultRootWindow(d), 0, 0, x, y, &child);
}

// ui_pop_workarea is the work area of the monitor holding root point px,
// py: the monitor's box (RandR), cut to _NET_WORKAREA for the current
// desktop (the screen less its panels), which is what EWMH offers. It
// reports 0 when there is no screen to speak of.
static int ui_pop_workarea(Display* d, int px, int py, int* ox, int* oy, int* ow, int* oh) {
	Window root = DefaultRootWindow(d);
	int s = DefaultScreen(d);
	int mx = 0, my = 0, mw = DisplayWidth(d, s), mh = DisplayHeight(d, s);
	int n = 0;
	XRRMonitorInfo* mons = XRRGetMonitors(d, root, True, &n);
	if (mons && n > 0) {
		int pick = -1;
		for (int i = 0; i < n && pick < 0; i++) {
			if (px >= mons[i].x && px < mons[i].x + mons[i].width &&
				py >= mons[i].y && py < mons[i].y + mons[i].height) pick = i;
		}
		for (int i = 0; i < n && pick < 0; i++) {
			if (mons[i].primary) pick = i;
		}
		if (pick < 0) pick = 0;
		mx = mons[pick].x; my = mons[pick].y; mw = mons[pick].width; mh = mons[pick].height;
	}
	if (mons) XRRFreeMonitors(mons);
	long desk = 0;
	Atom type = 0;
	int fmt = 0;
	unsigned long nitems = 0, after = 0;
	unsigned char* data = NULL;
	Atom cur = XInternAtom(d, "_NET_CURRENT_DESKTOP", True);
	if (cur && XGetWindowProperty(d, root, cur, 0, 1, False, XA_CARDINAL, &type, &fmt, &nitems, &after, &data) == Success &&
		data && nitems >= 1 && fmt == 32) {
		desk = ((long*)data)[0];
	}
	if (data) { XFree(data); data = NULL; }
	Atom wa = XInternAtom(d, "_NET_WORKAREA", True);
	if (wa && XGetWindowProperty(d, root, wa, 0, 1024, False, XA_CARDINAL, &type, &fmt, &nitems, &after, &data) == Success &&
		data && fmt == 32 && desk >= 0 && nitems >= (unsigned long)(4 * (desk + 1))) {
		long* v = (long*)data + 4 * desk;
		int ax = (int)v[0], ay = (int)v[1], aw = (int)v[2], ah = (int)v[3];
		int x0 = mx > ax ? mx : ax, y0 = my > ay ? my : ay;
		int x1 = (mx + mw) < (ax + aw) ? (mx + mw) : (ax + aw);
		int y1 = (my + mh) < (ay + ah) ? (my + mh) : (ay + ah);
		if (x1 > x0 && y1 > y0) { mx = x0; my = y0; mw = x1 - x0; mh = y1 - y0; }
	}
	if (data) XFree(data);
	*ox = mx; *oy = my; *ow = mw; *oh = mh;
	return mw > 0 && mh > 0;
}
*/
import "C"

import (
	"errors"

	"github.com/codemodify/paintengine2d"
)

// Popups on X11: an override-redirect window for every menu, submenu, list
// and tooltip (platform/popup.go has the whole story). No window manager
// touches one, so the toolkit places it itself — SolvePopup, within the
// work area of the monitor the window is on — and a menu takes the pointer
// and the keyboard with owner-events grabs, so a press anywhere outside the
// application comes to it and takes it down, as in Qt and GTK.
//
// Every event on a popup's window goes to its root window's queue with its
// position moved into the root's coordinates (x11Conn.deliverLocked), and a
// pointer that leaves a window for one of its own popups never left it.

// x11Popup is an x11Surface's popup half.
type x11Popup struct {
	parent, root *x11Surface
	role         PopupRole
	place        PopupPlacement
	placed       FrameRect
	// wantGrab: a menu, which grabs once its window is viewable (a grab on
	// an unmapped window fails); grabbed once it has, and tries how many
	// times it has been refused (another client holding a grab).
	wantGrab bool
	grabbed  bool
	tries    int
}

var errX11Popup = errors.New("platform: could not open a popup window")

// PopupsSupported reports whether a popup can be opened from s now
// (PopupOpener).
func (s *x11Surface) PopupsSupported() bool {
	return s != nil && !s.closed && s.conn != nil && s.conn.dpy != nil && s.win != 0
}

// popRoot is the top level s belongs to (s itself for a top level).
func (s *x11Surface) popRoot() *x11Surface {
	if s.pop != nil && s.pop.root != nil {
		return s.pop.root
	}
	return s
}

// rootOriginLocked is where the root window's X window is on the screen,
// device pixels.
func (s *x11Surface) rootOriginLocked() (int, int) {
	var x, y C.int
	if C.ui_pop_origin(s.conn.dpy, s.win, &x, &y) == 0 {
		return s.posX, s.posY
	}
	return int(x), int(y)
}

// workAreaLocked is the work area of the monitor holding the popup's
// anchor, in logical pixels relative to the root's visible box — the unit
// PopupPlacement speaks.
func (s *x11Surface) workAreaLocked(anchor FrameRect) (FrameRect, bool) {
	root := s.popRoot()
	sc := s.conn.displayScale()
	rx, ry := root.rootOriginLocked()
	m := root.frame.Margin
	vx, vy := rx+m.Left, ry+m.Top
	px := vx + DevicePosition(anchor.X, sc)
	py := vy + DevicePosition(anchor.Y, sc)
	var ax, ay, aw, ah C.int
	if C.ui_pop_workarea(s.conn.dpy, C.int(px), C.int(py), &ax, &ay, &aw, &ah) == 0 {
		return FrameRect{}, false
	}
	x0 := ceilDiv(int(ax)-vx, sc)
	y0 := ceilDiv(int(ay)-vy, sc)
	x1 := floorDiv(int(ax+aw)-vx, sc)
	y1 := floorDiv(int(ay+ah)-vy, sc)
	return FrameRect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}, true
}

// PopupWorkArea is the work area of the monitor the window is on, logical
// pixels relative to its visible box (PopupWorkArea).
func (s *x11Surface) PopupWorkArea() (FrameRect, bool) {
	if s == nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return FrameRect{}, false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	root := s.popRoot()
	sc := s.conn.displayScale()
	centre := FrameRect{X: LogicalPixels(root.geomW, sc) / 2, Y: LogicalPixels(root.geomH, sc) / 2}
	return s.workAreaLocked(centre)
}

// OpenPopup opens an override-redirect window from s, placed by SolvePopup
// in the monitor's work area (PopupOpener).
func (s *x11Surface) OpenPopup(opts PopupOptions) (PopupSurface, error) {
	if !s.PopupsSupported() {
		return nil, errX11Popup
	}
	c, err := x11Retain()
	if err != nil {
		return nil, err
	}
	root := s.popRoot()
	pl := opts.Placement
	x11Mu.Lock()
	area, _ := s.workAreaLocked(pl.Anchor)
	placed := SolvePopup(pl, area)
	sc := c.displayScale()
	f := opts.Frame
	if !c.composited {
		// Alpha counts for nothing on an uncomposited screen: no shadow,
		// and the bounding shape is what cuts a silhouette.
		f.Margin, f.Alpha, f.Blur = FrameInsets{}, false, nil
	}
	p := &x11Surface{
		conn: c, title: root.title, popup: true, wantDeco: DecorationsNone, focused: true,
		opts:   WindowOptions{Popup: true, Width: placed.W, Height: placed.H},
		sizing: SizingFixed,
		frame:  f,
		geomLW: placed.W, geomLH: placed.H,
		geomW: DevicePixels(placed.W, sc), geomH: DevicePixels(placed.H, sc),
		pop: &x11Popup{parent: s, root: root, role: opts.Role, place: pl, placed: placed,
			wantGrab: opts.Role == PopupRoleMenu},
	}
	w, h := p.geomW+f.Margin.Width(), p.geomH+f.Margin.Height()
	p.img = paintengine2d.NewImage(w, h)
	vis, cmap, depth := c.visualLocked(f.Alpha)
	rx, ry := root.rootOriginLocked()
	o := p.Origin()
	tooltip := C.int(0)
	if opts.Role == PopupRoleTooltip {
		tooltip = 1
	}
	p.win = C.ui_pop_create(c.dpy, root.win, C.int(rx+int(o.X)), C.int(ry+int(o.Y)), C.int(w), C.int(h),
		vis, cmap, depth, tooltip)
	if p.win == 0 {
		x11Mu.Unlock()
		c.release()
		return nil, errX11Popup
	}
	p.vis, p.cmap, p.visDepth = vis, cmap, int(depth)
	p.rmask, p.gmask, p.bmask = c.visualMasksLocked(vis)
	p.gc = C.ui_pop_gc(c.dpy, p.win)
	p.rebuildImageLocked()
	c.surfaces[p.win] = p
	p.applyFrameLocked()
	x11Mu.Unlock()
	s.kids = append(s.kids, p)
	return p, nil
}

// mappedPopupLocked runs once a popup's window is viewable: a menu takes
// its grabs now, since a grab on an unmapped window is refused.
func (s *x11Surface) mappedPopupLocked() {
	pp := s.pop
	if pp == nil || !pp.wantGrab || pp.grabbed || s.conn == nil || s.conn.dpy == nil {
		return
	}
	if C.ui_pop_grab(s.conn.dpy, s.win) != 0 {
		pp.grabbed = true
		return
	}
	// Someone else holds a grab — a window manager's key binding, a drag
	// just ending. Try again on the next few polls; a menu without its
	// grab still works, it only misses a click outside the application.
	pp.tries++
	if pp.tries > 30 {
		pp.wantGrab = false
	}
}

// retryGrabsLocked gives every popup still waiting for its grab another try.
func (c *x11Conn) retryGrabsLocked() {
	for _, s := range c.surfaces {
		if s != nil && s.pop != nil && s.mapped && s.pop.wantGrab && !s.pop.grabbed {
			s.mappedPopupLocked()
		}
	}
}

// Reposition places the popup afresh and moves its window (PopupSurface).
func (s *x11Surface) Reposition(pl PopupPlacement) bool {
	if s == nil || s.pop == nil || s.closed || s.conn == nil || s.conn.dpy == nil {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	area, _ := s.workAreaLocked(pl.Anchor)
	placed := SolvePopup(pl, area)
	s.pop.place, s.pop.placed = pl, placed
	sc := s.conn.displayScale()
	s.geomLW, s.geomLH = placed.W, placed.H
	s.geomW, s.geomH = DevicePixels(placed.W, sc), DevicePixels(placed.H, sc)
	m := s.frame.Margin
	w, h := s.geomW+m.Width(), s.geomH+m.Height()
	if s.img == nil || s.img.Width != w || s.img.Height != h {
		s.img = paintengine2d.NewImage(w, h)
		s.rebuildImageLocked()
	}
	rx, ry := s.pop.root.rootOriginLocked()
	o := s.Origin()
	C.ui_pop_move_resize(s.conn.dpy, s.win, C.int(rx+int(o.X)), C.int(ry+int(o.Y)), C.int(w), C.int(h))
	s.applyFrameLocked()
	return true
}

// Placed is where the popup's visible box is (PopupSurface).
func (s *x11Surface) Placed() FrameRect {
	if s == nil || s.pop == nil {
		return FrameRect{}
	}
	return s.pop.placed
}

// Origin is the popup's buffer origin in its root's buffer, device pixels
// (PopupSurface).
func (s *x11Surface) Origin() paintengine2d.Point {
	if s == nil || s.pop == nil {
		return paintengine2d.Point{}
	}
	r := s.pop.root
	m := r.frame.Margin
	return PopupOrigin(paintengine2d.Pt(float32(m.Left), float32(m.Top)), s.pop.placed, s.frame.Margin, s.conn.displayScale())
}

// Root is the top level the popup belongs to (PopupSurface).
func (s *x11Surface) Root() Surface {
	if s == nil {
		return nil
	}
	return s.popRoot()
}

// closePopupsLocked closes the popups open from s, the newest first, and a
// popup's own grab and place in its parent's list.
func (s *x11Surface) closePopupsLocked() {
	kids := s.kids
	s.kids = nil
	for i := len(kids) - 1; i >= 0; i-- {
		x11Mu.Unlock()
		_ = kids[i].Close()
		x11Mu.Lock()
	}
	pp := s.pop
	if pp == nil {
		return
	}
	if pp.grabbed && s.conn != nil && s.conn.dpy != nil {
		C.ui_pop_ungrab(s.conn.dpy)
		pp.grabbed = false
	}
	if par := pp.parent; par != nil {
		for i, k := range par.kids {
			if k == s {
				par.kids = append(par.kids[:i:i], par.kids[i+1:]...)
				break
			}
		}
	}
}

// deliverLocked queues evs from surface s for the app. A popup's go to its
// root with positions moved into the root's coordinates. A pointer leaving
// any window of a family is held back until the end of the burst: if it
// only went to another window of the same family — the window's own menu,
// a submenu — the family never lost it (flushLeavesLocked).
func (c *x11Conn) deliverLocked(s *x11Surface, evs []Event) {
	root := s.popRoot()
	var o paintengine2d.Point
	if s.pop != nil {
		o = s.Origin()
	}
	for _, ev := range evs {
		switch ev.Kind {
		case EventPointerLeave:
			if c.leaves == nil {
				c.leaves = map[*x11Surface]bool{}
			}
			c.leaves[root] = true
			continue
		case EventMouseMove, EventMouseDown, EventMouseUp, EventScroll:
			delete(c.leaves, root)
		}
		if s.pop != nil {
			if !popupInput(ev.Kind) {
				continue
			}
			ev = popupEvent(ev, o)
		}
		c.queues[root.win] = append(c.queues[root.win], ev)
	}
}

// flushLeavesLocked tells every family the pointer left in the burst and
// did not come back to.
func (c *x11Conn) flushLeavesLocked() {
	for r := range c.leaves {
		if !r.closed && r.win != 0 {
			c.queues[r.win] = append(c.queues[r.win], Event{Kind: EventPointerLeave})
		}
		delete(c.leaves, r)
	}
}

// popupsLostLocked is a root window really losing the keyboard (another
// window activated) while it has menus up: they go, as a Wayland
// compositor's popup_done takes them.
func (s *x11Surface) popupsLostLocked() []Event {
	for _, k := range s.kids {
		if k.pop != nil && k.pop.role == PopupRoleMenu {
			return []Event{{Kind: EventPopupDone, Popup: k}}
		}
	}
	return nil
}

var (
	_ PopupOpener   = (*x11Surface)(nil)
	_ PopupSurface  = (*x11Surface)(nil)
	_ PopupWorkArea = (*x11Surface)(nil)
)
