//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <stdint.h>
#include <stdlib.h>
#include <wayland-client.h>
#include "xdg-output-unstable-v1-client-protocol.h"

extern void uitkWlXdgOutPos(uintptr_t id, struct zxdg_output_v1 *o, int32_t x, int32_t y);
extern void uitkWlXdgOutSize(uintptr_t id, struct zxdg_output_v1 *o, int32_t w, int32_t h);
extern void uitkWlXdgOutDone(uintptr_t id, struct zxdg_output_v1 *o);

static void uitk_xdgout_pos(void *d, struct zxdg_output_v1 *o, int32_t x, int32_t y) {
	uitkWlXdgOutPos((uintptr_t)d, o, x, y);
}
static void uitk_xdgout_size(void *d, struct zxdg_output_v1 *o, int32_t w, int32_t h) {
	uitkWlXdgOutSize((uintptr_t)d, o, w, h);
}
static void uitk_xdgout_done(void *d, struct zxdg_output_v1 *o) {
	uitkWlXdgOutDone((uintptr_t)d, o);
}
static void uitk_xdgout_name(void *d, struct zxdg_output_v1 *o, const char *n) {
	(void)d; (void)o; (void)n;
}
static void uitk_xdgout_desc(void *d, struct zxdg_output_v1 *o, const char *n) {
	(void)d; (void)o; (void)n;
}
static const struct zxdg_output_v1_listener uitk_xdgout_listener = {
	.logical_position = uitk_xdgout_pos,
	.logical_size = uitk_xdgout_size,
	.done = uitk_xdgout_done,
	.name = uitk_xdgout_name,
	.description = uitk_xdgout_desc,
};

static void *ui_wl_bind_xdg_output_man(struct wl_registry *r, uint32_t name, uint32_t ver) {
	return wl_registry_bind(r, name, &zxdg_output_manager_v1_interface, ver);
}
static struct zxdg_output_v1 *ui_wl_xdg_output_get(void *man, struct wl_output *out, uintptr_t id) {
	struct zxdg_output_manager_v1 *m = man;
	if (!m || !out) return NULL;
	struct zxdg_output_v1 *o = zxdg_output_manager_v1_get_xdg_output(m, out);
	if (!o) return NULL;
	zxdg_output_v1_add_listener(o, &uitk_xdgout_listener, (void *)id);
	return o;
}
static void ui_wl_xdg_output_destroy(struct zxdg_output_v1 *o) {
	if (o) zxdg_output_v1_destroy(o);
}
static void ui_wl_xdg_output_man_destroy(void *man) {
	struct zxdg_output_manager_v1 *m = man;
	if (m) zxdg_output_manager_v1_destroy(m);
}
*/
import "C"

import "unsafe"

// xdg-output-unstable-v1 — what the desktop's logical geometry really is.
//
// wl_output describes a monitor in its own pixels: a mode, and an integer
// scale factor. Divide one by the other and you have the logical size the
// compositor works in — as long as the scale really is an integer. Under
// fractional scaling it is not: KWin at 175% still reports
// wl_output.scale 2 (clients are meant to render at 2x and let the
// compositor downscale, or use wp_fractional_scale_v1), so 2880x1800/2
// says the screen is 1440x900 logical when the compositor's own space is
// 1645x1029. Anything placed against that number — a tray menu, say — is
// constrained to a screen 200 pixels narrower than the real one and hangs
// off the edge.
//
// zxdg_output_v1 reports the compositor's logical_position and
// logical_size for each wl_output and is the only protocol that does.
// KWin advertises zxdg_output_manager_v1 version 3; sway, Hyprland,
// wayfire and Mutter all offer it too. Where it is missing (a compositor
// older than 2017, or one built without it) [outputSet.logicalBox] keeps
// the wl_output arithmetic, which is exactly right whenever the scale is
// an integer and is the best guess otherwise.
//
// The events are asynchronous like the rest of the registry: an
// xdg_output is requested as soon as its wl_output is bound (or as soon
// as the manager arrives, whichever is later) and the logical rectangle
// appears some round trips afterwards. Nothing here blocks for it —
// logicalBox simply answers from wl_output until it lands.

// bindXdgOutputManager binds zxdg_output_manager_v1, up to version 3 (v2
// added name and description, v3 made zxdg_output_v1.done redundant: the
// values become current on wl_output.done). It then asks for the
// xdg_output of every wl_output already bound, because the registry may
// announce the outputs before the manager.
func (c *wlConn) bindXdgOutputManager(reg *C.struct_wl_registry, name C.uint32_t, ver C.uint32_t) {
	if c == nil || c.xdgOutMan != nil {
		return
	}
	v := ver
	if v > 3 {
		v = 3
	}
	if v < 1 {
		return
	}
	c.xdgOutMan = unsafe.Pointer(C.ui_wl_bind_xdg_output_man(reg, name, v))
	if c.xdgOutMan == nil {
		return
	}
	c.xdgOutVer = int(v)
	for outName, out := range c.outObjs {
		c.requestXdgOutput(outName, out)
	}
}

// requestXdgOutput asks for one wl_output's xdg_output. It is a no-op
// without the manager (the compositor has no xdg-output) or when this
// output already has one.
func (c *wlConn) requestXdgOutput(name uint32, out *C.struct_wl_output) {
	if c == nil || c.xdgOutMan == nil || out == nil {
		return
	}
	if c.xdgOutObjs == nil {
		c.xdgOutObjs = map[uint32]*C.struct_zxdg_output_v1{}
		c.xdgOutNames = map[uintptr]uint32{}
	}
	if c.xdgOutObjs[name] != nil {
		return
	}
	o := C.ui_wl_xdg_output_get(c.xdgOutMan, out, C.uintptr_t(c.id))
	if o == nil {
		return
	}
	c.xdgOutObjs[name] = o
	c.xdgOutNames[uintptr(unsafe.Pointer(o))] = name
}

// dropXdgOutput destroys one output's xdg_output (the wl_output left the
// registry, or the connection is going away).
func (c *wlConn) dropXdgOutput(name uint32) {
	if c == nil || c.xdgOutObjs == nil {
		return
	}
	o := c.xdgOutObjs[name]
	if o == nil {
		return
	}
	delete(c.xdgOutNames, uintptr(unsafe.Pointer(o)))
	delete(c.xdgOutObjs, name)
	C.ui_wl_xdg_output_destroy(o)
}

// destroyXdgOutputsLocked drops every xdg_output and the manager.
func (c *wlConn) destroyXdgOutputsLocked() {
	if c == nil {
		return
	}
	for name := range c.xdgOutObjs {
		c.dropXdgOutput(name)
	}
	if c.xdgOutMan != nil {
		C.ui_wl_xdg_output_man_destroy(c.xdgOutMan)
		c.xdgOutMan = nil
		c.xdgOutVer = 0
	}
}

// xdgOutputName is the registry name of the wl_output an xdg_output
// belongs to.
func (c *wlConn) xdgOutputName(o *C.struct_zxdg_output_v1) (uint32, bool) {
	if c == nil || o == nil || c.xdgOutNames == nil {
		return 0, false
	}
	n, ok := c.xdgOutNames[uintptr(unsafe.Pointer(o))]
	return n, ok
}

//export uitkWlXdgOutPos
func uitkWlXdgOutPos(id C.uintptr_t, o *C.struct_zxdg_output_v1, x, y C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if name, ok := c.xdgOutputName(o); ok {
		c.outs.setLogicalPos(name, int(x), int(y))
	}
}

//export uitkWlXdgOutSize
func uitkWlXdgOutSize(id C.uintptr_t, o *C.struct_zxdg_output_v1, w, h C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if name, ok := c.xdgOutputName(o); ok {
		c.outs.setLogicalSize(name, int(w), int(h))
	}
}

//export uitkWlXdgOutDone
func uitkWlXdgOutDone(id C.uintptr_t, o *C.struct_zxdg_output_v1) {
	// Nothing to do: the position and the size are applied as they
	// arrive rather than staged until done, and from version 3 the
	// compositor does not send this event at all.
	_, _ = id, o
}
