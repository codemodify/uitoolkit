//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <stdint.h>
#include <wayland-client.h>
#include "pointer-gestures-unstable-v1-client-protocol.h"

extern void uitkWlGestBegin(uintptr_t id, uint32_t kind, struct wl_surface *surf, uint32_t fingers);
extern void uitkWlGestUpdate(uintptr_t id, uint32_t kind, wl_fixed_t dx, wl_fixed_t dy, wl_fixed_t scale, wl_fixed_t rotation);
extern void uitkWlGestEnd(uintptr_t id, uint32_t kind, int32_t cancelled);

static void *ui_wl_bind_gestures(struct wl_registry *r, uint32_t name, uint32_t ver) {
	return wl_registry_bind(r, name, &zwp_pointer_gestures_v1_interface, ver);
}

// The kinds match GestureKind: 1 pinch, 2 swipe, 3 hold.
static void uitk_swipe_begin(void *d, struct zwp_pointer_gesture_swipe_v1 *g, uint32_t serial, uint32_t time, struct wl_surface *s, uint32_t fingers) {
	(void)g; (void)serial; (void)time;
	uitkWlGestBegin((uintptr_t)d, 2, s, fingers);
}
static void uitk_swipe_update(void *d, struct zwp_pointer_gesture_swipe_v1 *g, uint32_t time, wl_fixed_t dx, wl_fixed_t dy) {
	(void)g; (void)time;
	uitkWlGestUpdate((uintptr_t)d, 2, dx, dy, 0, 0);
}
static void uitk_swipe_end(void *d, struct zwp_pointer_gesture_swipe_v1 *g, uint32_t serial, uint32_t time, int32_t cancelled) {
	(void)g; (void)serial; (void)time;
	uitkWlGestEnd((uintptr_t)d, 2, cancelled);
}
static const struct zwp_pointer_gesture_swipe_v1_listener uitk_swipe_listener = {
	.begin = uitk_swipe_begin, .update = uitk_swipe_update, .end = uitk_swipe_end,
};

static void uitk_pinch_begin(void *d, struct zwp_pointer_gesture_pinch_v1 *g, uint32_t serial, uint32_t time, struct wl_surface *s, uint32_t fingers) {
	(void)g; (void)serial; (void)time;
	uitkWlGestBegin((uintptr_t)d, 1, s, fingers);
}
static void uitk_pinch_update(void *d, struct zwp_pointer_gesture_pinch_v1 *g, uint32_t time, wl_fixed_t dx, wl_fixed_t dy, wl_fixed_t scale, wl_fixed_t rotation) {
	(void)g; (void)time;
	uitkWlGestUpdate((uintptr_t)d, 1, dx, dy, scale, rotation);
}
static void uitk_pinch_end(void *d, struct zwp_pointer_gesture_pinch_v1 *g, uint32_t serial, uint32_t time, int32_t cancelled) {
	(void)g; (void)serial; (void)time;
	uitkWlGestEnd((uintptr_t)d, 1, cancelled);
}
static const struct zwp_pointer_gesture_pinch_v1_listener uitk_pinch_listener = {
	.begin = uitk_pinch_begin, .update = uitk_pinch_update, .end = uitk_pinch_end,
};

static void uitk_hold_begin(void *d, struct zwp_pointer_gesture_hold_v1 *g, uint32_t serial, uint32_t time, struct wl_surface *s, uint32_t fingers) {
	(void)g; (void)serial; (void)time;
	uitkWlGestBegin((uintptr_t)d, 3, s, fingers);
}
static void uitk_hold_end(void *d, struct zwp_pointer_gesture_hold_v1 *g, uint32_t serial, uint32_t time, int32_t cancelled) {
	(void)g; (void)serial; (void)time;
	uitkWlGestEnd((uintptr_t)d, 3, cancelled);
}
static const struct zwp_pointer_gesture_hold_v1_listener uitk_hold_listener = {
	.begin = uitk_hold_begin, .end = uitk_hold_end,
};

// ui_wl_gesture_objects makes the pointer's gesture objects: swipe and pinch
// from version 1, hold from 3.
static void ui_wl_gesture_objects(void *man, struct wl_pointer *p, uintptr_t id, void **out) {
	struct zwp_pointer_gestures_v1 *m = man;
	out[0] = zwp_pointer_gestures_v1_get_swipe_gesture(m, p);
	zwp_pointer_gesture_swipe_v1_add_listener(out[0], &uitk_swipe_listener, (void *)id);
	out[1] = zwp_pointer_gestures_v1_get_pinch_gesture(m, p);
	zwp_pointer_gesture_pinch_v1_add_listener(out[1], &uitk_pinch_listener, (void *)id);
	out[2] = NULL;
	if (zwp_pointer_gestures_v1_get_version(m) >= ZWP_POINTER_GESTURES_V1_GET_HOLD_GESTURE_SINCE_VERSION) {
		out[2] = zwp_pointer_gestures_v1_get_hold_gesture(m, p);
		zwp_pointer_gesture_hold_v1_add_listener(out[2], &uitk_hold_listener, (void *)id);
	}
}

static double ui_wl_gfixed(wl_fixed_t v) { return wl_fixed_to_double(v); }
*/
import "C"

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// wlGestures is a connection's zwp_pointer_gestures_v1: the global, the
// three gesture objects made for its pointer, and the gesture in progress
// over the surface surf.
type wlGestures struct {
	man   unsafe.Pointer
	objs  [3]unsafe.Pointer
	track gestureTracker
	surf  int
}

// bindGestures binds zwp_pointer_gestures_v1 (up to version 3, which adds
// hold) and makes the gesture objects if the pointer is already there.
func (c *wlConn) bindGestures(reg *C.struct_wl_registry, name C.uint32_t, ver C.uint32_t) {
	v := ver
	if v > 3 {
		v = 3
	}
	if v < 1 || c.gest.man != nil {
		return
	}
	c.gest.man = C.ui_wl_bind_gestures(reg, name, v)
	c.pointerGestures()
}

// pointerGestures makes the pointer's gesture objects once there are both
// a pointer and the global, whichever comes first.
func (c *wlConn) pointerGestures() {
	if c.gest.man == nil || c.pointer == nil || c.gest.objs[0] != nil {
		return
	}
	C.ui_wl_gesture_objects(c.gest.man, c.pointer, C.uintptr_t(c.id), (*unsafe.Pointer)(unsafe.Pointer(&c.gest.objs[0])))
}

// gesturePos is the pointer in s's device pixels.
func (c *wlConn) gesturePos(s *wlSurface) paintengine2d.Point {
	x, y := s.toDevice(c.px, c.py)
	return paintengine2d.Pt(x, y)
}

func (c *wlConn) pushGesture(s *wlSurface, evs []Event) {
	for _, ev := range evs {
		s.push(ev)
	}
}

// cancelGesture calls off a gesture in progress (the pointer left).
func (c *wlConn) cancelGesture() {
	if s := wlSurfaces[c.gest.surf]; s != nil && c.gest.track.open() {
		c.pushGesture(s, c.gest.track.cancel(c.gesturePos(s), c.mods))
	}
	c.gest.track = gestureTracker{}
}

//export uitkWlGestBegin
func uitkWlGestBegin(id C.uintptr_t, kind C.uint32_t, surf *C.struct_wl_surface, fingers C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfNative(surf)
	if s == nil {
		s = wlSurfaces[c.ptrSurf]
	}
	if s == nil {
		return
	}
	if k := GestureKind(kind); k == GestureHold {
		// Fingers put down on the pad stop a glide, as a finger put on
		// a phone's screen does (GTK 4 does the same with hold).
		c.kin.stop()
	}
	if old := wlSurfaces[c.gest.surf]; old != nil && old != s && c.gest.track.open() {
		c.pushGesture(old, c.gest.track.cancel(c.gesturePos(old), c.mods))
	}
	c.gest.surf = s.id
	c.pushGesture(s, c.gest.track.begin(GestureKind(kind), int(fingers), c.gesturePos(s), c.mods))
}

//export uitkWlGestUpdate
func uitkWlGestUpdate(id C.uintptr_t, kind C.uint32_t, dx, dy, scale, rotation C.wl_fixed_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.gest.surf]
	if s == nil {
		return
	}
	ddx, ddy := s.toDevice(float32(C.ui_wl_gfixed(dx)), float32(C.ui_wl_gfixed(dy)))
	c.pushGesture(s, c.gest.track.update(GestureKind(kind), c.gesturePos(s), paintengine2d.Pt(ddx, ddy),
		float32(C.ui_wl_gfixed(scale)), float32(C.ui_wl_gfixed(rotation)), c.mods))
}

//export uitkWlGestEnd
func uitkWlGestEnd(id C.uintptr_t, kind C.uint32_t, cancelled C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.gest.surf]
	if s == nil {
		return
	}
	c.pushGesture(s, c.gest.track.finish(GestureKind(kind), cancelled != 0, c.gesturePos(s), c.mods))
	if !c.gest.track.open() {
		c.gest.surf = 0
	}
}
