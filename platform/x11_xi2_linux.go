//go:build linux && cgo

package platform

/*
#cgo linux LDFLAGS: -lXi
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/extensions/XInput2.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern void uitkXIDevice(int id, char *name, char *props, int pxdist);
extern void uitkXIScrollClass(int id, int number, int vertical, double increment);

// XInput 2.4's gesture events; a libXi older than 1.8 builds without them.
#ifdef XI_GesturePinchBegin
#define UITK_XI_GESTURES 1
#else
#define UITK_XI_GESTURES 0
#endif

static int ui_xi2_gestures_built(void) { return UITK_XI_GESTURES; }

// ui_xi2_version asks for XInput 2.4 and says what the server speaks
// (minor, or -1 for no XInput 2 at all); opcode is the extension's.
static int ui_xi2_version(Display *d, int *opcode) {
	int ev = 0, err = 0;
	if (!d || !XQueryExtension(d, "XInputExtension", opcode, &ev, &err)) return -1;
	int major = 2, minor = UITK_XI_GESTURES ? 4 : 2;
	if (XIQueryVersion(d, &major, &minor) != Success || major < 2) return -1;
	return minor;
}

// ui_xi2_select_window takes the pointer's XInput 2 events on w: motion
// (the scroll valuators ride on it), buttons (to tell a wheel's emulated
// press from a real one), enter (to restart the valuators) and, from 2.4,
// the touchpad's pinch and swipe. What is selected here stops coming as
// core events.
static void ui_xi2_select_window(Display *d, Window w, int gestures) {
	unsigned char mask[XIMaskLen(XI_LASTEVENT)];
	memset(mask, 0, sizeof mask);
	XISetMask(mask, XI_Motion);
	XISetMask(mask, XI_ButtonPress);
	XISetMask(mask, XI_ButtonRelease);
	XISetMask(mask, XI_Enter);
#if UITK_XI_GESTURES
	if (gestures) {
		XISetMask(mask, XI_GesturePinchBegin);
		XISetMask(mask, XI_GesturePinchUpdate);
		XISetMask(mask, XI_GesturePinchEnd);
		XISetMask(mask, XI_GestureSwipeBegin);
		XISetMask(mask, XI_GestureSwipeUpdate);
		XISetMask(mask, XI_GestureSwipeEnd);
	}
#endif
	XIEventMask em = { XIAllMasterDevices, sizeof mask, mask };
	XISelectEvents(d, w, &em, 1);
}

// ui_xi2_select_root hears devices come, go and change.
static void ui_xi2_select_root(Display *d) {
	unsigned char mask[XIMaskLen(XI_LASTEVENT)];
	memset(mask, 0, sizeof mask);
	XISetMask(mask, XI_HierarchyChanged);
	XISetMask(mask, XI_DeviceChanged);
	XIEventMask em = { XIAllDevices, sizeof mask, mask };
	XISelectEvents(d, DefaultRootWindow(d), &em, 1);
}

// ui_xi2_scan reports every pointer device's name, property names and
// scroll valuators (uitkXIDevice, then uitkXIScrollClass for each).
static void ui_xi2_scan(Display *d) {
	int n = 0;
	XIDeviceInfo *info = XIQueryDevice(d, XIAllDevices, &n);
	if (!info) return;
	Atom pxAtom = XInternAtom(d, "libinput Scrolling Pixel Distance", True);
	for (int i = 0; i < n; i++) {
		XIDeviceInfo *dev = &info[i];
		if (dev->use != XISlavePointer && dev->use != XIMasterPointer) continue;
		int scrolls = 0;
		for (int c = 0; c < dev->num_classes; c++)
			if (dev->classes[c]->type == XIScrollClass) scrolls++;
		if (!scrolls) continue;
		// Its property names, one per line.
		int np = 0;
		Atom *props = XIListProperties(d, dev->deviceid, &np);
		size_t len = 1;
		char **names = np > 0 ? calloc((size_t)np, sizeof(char *)) : NULL;
		int pxdist = 0;
		// One round trip for all the names.
		if (names && !XGetAtomNames(d, props, np, names)) {
			free(names);
			names = NULL;
		}
		for (int p = 0; names && p < np; p++) {
			if (names[p]) len += strlen(names[p]) + 1;
			if (pxAtom != None && props[p] == pxAtom) {
				Atom type; int format; unsigned long items, after; unsigned char *data = NULL;
				if (XIGetProperty(d, dev->deviceid, pxAtom, 0, 1, False, AnyPropertyType, &type, &format, &items, &after, &data) == Success && data) {
					if (items >= 1 && format == 32) pxdist = (int)*(uint32_t *)data;
					XFree(data);
				}
			}
		}
		char *joined = calloc(len, 1);
		for (int p = 0; names && joined && p < np; p++) {
			if (!names[p]) continue;
			strcat(joined, names[p]);
			strcat(joined, "\n");
			XFree(names[p]);
		}
		free(names);
		if (props) XFree(props);
		uitkXIDevice(dev->deviceid, dev->name, joined ? joined : "", pxdist);
		free(joined);
		for (int c = 0; c < dev->num_classes; c++) {
			if (dev->classes[c]->type != XIScrollClass) continue;
			XIScrollClassInfo *s = (XIScrollClassInfo *)dev->classes[c];
			uitkXIScrollClass(dev->deviceid, s->number, s->scroll_type == XIScrollTypeVertical, s->increment);
		}
	}
	XIFreeDeviceInfo(info);
}

// uitk_xi is one XInput 2 event, flattened for Go.
struct uitk_xi {
	int evtype, deviceid, sourceid, detail, flags, emulated, cancelled;
	Window event;
	Time time;
	double x, y, rootx, rooty, dx, dy, scale, angle;
	unsigned int mods;
	int nval;
	int valnum[16];
	double val[16];
};

// ui_xi2_decode reads a GenericEvent of the XInput extension into out; it
// reports 0 for anything else. The cookie's data is freed before it returns.
static int ui_xi2_decode(Display *d, XEvent *xe, int opcode, struct uitk_xi *out) {
	memset(out, 0, sizeof *out);
	XGenericEventCookie *ck = &xe->xcookie;
	if (xe->type != GenericEvent || ck->extension != opcode) return 0;
	if (!XGetEventData(d, ck)) return 0;
	out->evtype = ck->evtype;
	switch (ck->evtype) {
	case XI_Motion: case XI_ButtonPress: case XI_ButtonRelease: {
		XIDeviceEvent *e = ck->data;
		out->deviceid = e->deviceid; out->sourceid = e->sourceid; out->detail = e->detail;
		out->flags = e->flags; out->emulated = (e->flags & XIPointerEmulated) != 0;
		out->event = e->event; out->time = e->time;
		out->x = e->event_x; out->y = e->event_y; out->rootx = e->root_x; out->rooty = e->root_y;
		out->mods = (unsigned int)e->mods.effective;
		const double *v = e->valuators.values;
		for (int i = 0; i < e->valuators.mask_len * 8 && out->nval < 16; i++) {
			if (!XIMaskIsSet(e->valuators.mask, i)) continue;
			out->valnum[out->nval] = i;
			out->val[out->nval] = *v++;
			out->nval++;
		}
		break;
	}
	case XI_DeviceChanged: {
		XIDeviceChangedEvent *e = ck->data;
		out->deviceid = e->deviceid; out->sourceid = e->sourceid; out->detail = e->reason;
		break;
	}
	case XI_Enter: {
		XIEnterEvent *e = ck->data;
		out->deviceid = e->deviceid; out->sourceid = e->sourceid; out->event = e->event;
		break;
	}
#if UITK_XI_GESTURES
	case XI_GesturePinchBegin: case XI_GesturePinchUpdate: case XI_GesturePinchEnd: {
		XIGesturePinchEvent *e = ck->data;
		out->deviceid = e->deviceid; out->sourceid = e->sourceid; out->detail = e->detail;
		out->event = e->event; out->time = e->time;
		out->x = e->event_x; out->y = e->event_y; out->dx = e->delta_x; out->dy = e->delta_y;
		out->scale = e->scale; out->angle = e->delta_angle;
		out->cancelled = (e->flags & XIGesturePinchEventCancelled) != 0;
		out->mods = (unsigned int)e->mods.effective;
		break;
	}
	case XI_GestureSwipeBegin: case XI_GestureSwipeUpdate: case XI_GestureSwipeEnd: {
		XIGestureSwipeEvent *e = ck->data;
		out->deviceid = e->deviceid; out->sourceid = e->sourceid; out->detail = e->detail;
		out->event = e->event; out->time = e->time;
		out->x = e->event_x; out->y = e->event_y; out->dx = e->delta_x; out->dy = e->delta_y;
		out->cancelled = (e->flags & XIGestureSwipeEventCancelled) != 0;
		out->mods = (unsigned int)e->mods.effective;
		break;
	}
#endif
	}
	XFreeEventData(d, ck);
	return 1;
}

// The XI 2.4 event types, or -1 where this libXi has none.
static int ui_xi_pinch_begin(void) {
#if UITK_XI_GESTURES
	return XI_GesturePinchBegin;
#else
	return -1;
#endif
}

static void ui_xi2_ungrab(Display *d, int deviceid) {
	if (d && deviceid > 0) XIUngrabDevice(d, deviceid, CurrentTime);
}
*/
import "C"

import (
	"os"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
)

// EnvX11XI2 set to 0 leaves XInput 2 alone: core events only, wheel
// notches, no smooth scrolling and no gestures — the path X11 took before.
const EnvX11XI2 = "UITK_X11_XI2"

// x11XI2 is what a connection does with XInput 2: smooth scrolling from
// 2.1 (x11scroll.go), touchpad gestures from 2.4.
type x11XI2 struct {
	opcode int
	// minor is the server's XInput 2 minor version; -1 for none (or
	// UITK_X11_XI2=0), when everything stays on core events.
	minor    int
	gestures bool
	scroll   *x11Scroller
	// scrollWin and scrollPos are where the last finger scroll was, which
	// its glide goes on scrolling; scrollMods the modifiers then.
	scrollWin  C.Window
	scrollPos  paintengine2d.Point
	scrollMods Modifiers
	// gest is the gesture in progress, over gestWin.
	gest    gestureTracker
	gestWin C.Window
	// pointer is the master pointer the last XInput 2 press came from,
	// for ungrabbing it when the window manager takes the pointer over.
	pointer int
}

// x11ScanInto collects a device scan (the C side calls back into Go).
var x11ScanInto *x11Scroller

var x11ScanDevs map[int]*x11ScrollDevice

//export uitkXIDevice
func uitkXIDevice(id C.int, name, props *C.char, pxdist C.int) {
	if x11ScanDevs == nil {
		return
	}
	kind := x11ClassifyDevice(C.GoString(name), strings.Split(strings.TrimSpace(C.GoString(props)), "\n"))
	x11ScanDevs[int(id)] = &x11ScrollDevice{kind: kind, pxPerNotch: float32(pxdist)}
}

//export uitkXIScrollClass
func uitkXIScrollClass(id, number, vertical C.int, increment C.double) {
	if d := x11ScanDevs[int(id)]; d != nil {
		d.vals = append(d.vals, x11Valuator{number: int(number), vertical: vertical != 0, increment: float64(increment)})
	}
}

// initXI2Locked asks the server for XInput 2 and, where it speaks it,
// takes the devices' scroll valuators.
func (c *x11Conn) initXI2Locked() {
	c.xi.minor = -1
	if os.Getenv(EnvX11XI2) == "0" {
		return
	}
	var opcode C.int
	minor := int(C.ui_xi2_version(c.dpy, &opcode))
	if minor < 1 {
		// 2.0 has no scroll valuators: the core wheel buttons do.
		return
	}
	c.xi.minor, c.xi.opcode = minor, int(opcode)
	c.xi.gestures = minor >= 4 && C.ui_xi2_gestures_built() != 0
	c.xi.scroll = newX11Scroller()
	C.ui_xi2_select_root(c.dpy)
	c.scanXI2Locked()
}

// scanXI2Locked (re)reads every pointer device's scroll valuators.
func (c *x11Conn) scanXI2Locked() {
	x11ScanDevs = map[int]*x11ScrollDevice{}
	C.ui_xi2_scan(c.dpy)
	devs := x11ScanDevs
	x11ScanDevs = nil
	sc := c.xi.scroll
	sc.devices = map[int]*x11ScrollDevice{}
	for id, d := range devs {
		sc.setDevice(id, d.kind, d.pxPerNotch, d.vals)
	}
}

// selectXI2Locked takes a window's pointer events through XInput 2.
func (s *x11Surface) selectXI2Locked() {
	c := s.conn
	if c == nil || c.xi.minor < 1 || s.win == 0 {
		return
	}
	g := 0
	if c.xi.gestures {
		g = 1
	}
	C.ui_xi2_select_window(c.dpy, s.win, C.int(g))
}

// ungrabXI2Locked lets go of the pointer's XInput 2 implicit grab (a press
// selected through XInput 2 grabs the device that way), before the window
// manager is handed the pointer.
func (c *x11Conn) ungrabXI2Locked() {
	if c.xi.minor >= 1 && c.xi.pointer > 0 {
		C.ui_xi2_ungrab(c.dpy, C.int(c.xi.pointer))
	}
}

// handleXI2Locked takes an XInput 2 event; it reports false for any other.
func (c *x11Conn) handleXI2Locked(xe *C.XEvent) bool {
	if c.xi.minor < 1 {
		return false
	}
	var e C.struct_uitk_xi
	if C.ui_xi2_decode(c.dpy, xe, C.int(c.xi.opcode), &e) == 0 {
		return false
	}
	if e.time != 0 {
		c.serverTime = e.time
	}
	switch int(e.evtype) {
	case C.XI_HierarchyChanged:
		c.scanXI2Locked()
		return true
	case C.XI_DeviceChanged:
		// The master pointer switching from one device to another (the
		// touchpad, the mouse) only restarts the valuators; a device
		// whose valuators themselves changed is read again.
		if int(e.detail) == C.XIDeviceChange {
			c.scanXI2Locked()
		} else {
			c.xi.scroll.reset()
		}
		return true
	}
	s := c.surfaces[e.event]
	if s == nil {
		return true
	}
	pos := paintengine2d.Pt(float32(e.x), float32(e.y))
	mods := xmods(uint(e.mods))
	switch ev := int(e.evtype); {
	case ev == C.XI_Enter:
		c.xi.scroll.reset()
	case ev == C.XI_Motion:
		values := make(map[int]float64, int(e.nval))
		for i := 0; i < int(e.nval); i++ {
			values[int(e.valnum[i])] = float64(e.val[i])
		}
		c.xi.scroll.scale = c.displayScale()
		dx, dy, precise, ok := c.xi.scroll.motion(time.Now(), int(e.sourceid), values)
		evs := []Event{{Kind: EventMouseMove, Pos: pos, Mods: mods}}
		if ok {
			evs = append(evs, Event{Kind: EventScroll, Pos: pos, Scroll: paintengine2d.Pt(dx, dy), Mods: mods, ScrollPrecise: precise})
			if precise {
				c.xi.scrollWin, c.xi.scrollPos, c.xi.scrollMods = s.win, pos, mods
			}
		}
		c.deliverLocked(s, evs)
	case ev == C.XI_ButtonPress || ev == C.XI_ButtonRelease:
		btn := int(e.detail)
		if btn >= 4 && btn <= 7 && e.emulated != 0 && c.xi.scroll.hasScroll(int(e.sourceid)) {
			// The valuators already scrolled; this is the server's copy
			// for clients that only know buttons.
			return true
		}
		if ev == C.XI_ButtonPress {
			c.xi.pointer = int(e.deviceid)
		}
		c.deliverLocked(s, s.buttonLocked(btn, ev == C.XI_ButtonRelease, pos,
			int(e.rootx), int(e.rooty), mods, e.time))
	default:
		if c.xi.gestures {
			c.xi2GestureLocked(s, ev, &e, pos, mods)
		}
	}
	return true
}

// xi2GestureLocked turns XInput 2.4's pinch and swipe into EventGesture.
func (c *x11Conn) xi2GestureLocked(s *x11Surface, ev int, e *C.struct_uitk_xi, pos paintengine2d.Point, mods Modifiers) {
	base := int(C.ui_xi_pinch_begin())
	if base < 0 {
		return
	}
	// Pinch begin, update, end, then swipe begin, update, end.
	i := ev - base
	if i < 0 || i > 5 {
		return
	}
	kind := GesturePinch
	if i >= 3 {
		kind, i = GestureSwipe, i-3
	}
	var evs []Event
	delta := paintengine2d.Pt(float32(e.dx), float32(e.dy))
	switch i {
	case 0:
		// A new finger stops a glide.
		c.xi.scroll.stop()
		if old := c.surfaces[c.xi.gestWin]; old != nil && old != s && c.xi.gest.open() {
			c.deliverLocked(old, c.xi.gest.cancel(pos, mods))
		}
		c.xi.gestWin = s.win
		evs = c.xi.gest.begin(kind, int(e.detail), pos, mods)
		// XInput's begin can already carry motion.
		if delta.X != 0 || delta.Y != 0 || (kind == GesturePinch && float32(e.scale) != 1) {
			evs = append(evs, c.xi.gest.update(kind, pos, delta, float32(e.scale), float32(e.angle), mods)...)
		}
	case 1:
		evs = c.xi.gest.update(kind, pos, delta, float32(e.scale), float32(e.angle), mods)
	case 2:
		if delta.X != 0 || delta.Y != 0 {
			evs = c.xi.gest.update(kind, pos, delta, float32(e.scale), float32(e.angle), mods)
		}
		evs = append(evs, c.xi.gest.finish(kind, e.cancelled != 0, pos, mods)...)
	}
	if t := c.surfaces[c.xi.gestWin]; t != nil {
		c.deliverLocked(t, evs)
	}
}

// flushScrollLocked delivers a finger scroll's glide, once the fingers
// have lifted.
func (c *x11Conn) flushScrollLocked() {
	if c.xi.scroll == nil {
		return
	}
	dx, dy, ok := c.xi.scroll.tick(time.Now())
	if !ok {
		return
	}
	s := c.surfaces[c.xi.scrollWin]
	if s == nil {
		c.xi.scroll.stop()
		return
	}
	c.deliverLocked(s, []Event{{Kind: EventScroll, Pos: c.xi.scrollPos, Scroll: paintengine2d.Pt(dx, dy), Mods: c.xi.scrollMods, ScrollPrecise: true}})
}

// scrollWakeLocked is when a finger scroll's glide next needs the loop.
func (c *x11Conn) scrollWakeLocked() time.Time {
	if c.xi.scroll == nil {
		return time.Time{}
	}
	return c.xi.scroll.wakeAt()
}

// buttonLocked maps a press or release of core button btn at pos (root
// position rx, ry) — from a core event or an XInput 2 one. Buttons 4–7 are
// wheel notches; anything else is a press the window keeps for a system
// move or resize.
func (s *x11Surface) buttonLocked(btn int, release bool, pos paintengine2d.Point, rx, ry int, mods Modifiers, t C.Time) []Event {
	if bit := x11ButtonBit(btn); bit != 0 {
		if release {
			s.buttons &^= bit
		} else {
			s.buttons |= bit
			s.pressRootX, s.pressRootY = rx, ry
			s.pressButton = btn
			s.pressTime = t
		}
	}
	if !release && (btn < 4 || btn > 7) && s.conn != nil && s.conn.xi.scroll != nil {
		// A click stops a glide.
		s.conn.xi.scroll.stop()
	}
	return x11ButtonEvents(btn, release, pos, mods)
}
