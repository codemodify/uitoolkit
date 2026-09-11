//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client xkbcommon
#cgo linux CFLAGS: -I${SRCDIR}
#define _GNU_SOURCE
#include <wayland-client.h>
#include <xkbcommon/xkbcommon.h>
#include <fcntl.h>
#include <poll.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>
#include "xdg-shell-client-protocol.h"

extern void uitkWlRegistryGlobal(uintptr_t id, struct wl_registry *reg, uint32_t name, char *iface, uint32_t ver);
extern void uitkWlPing(uintptr_t id, struct xdg_wm_base *wm, uint32_t serial);
extern void uitkWlXdgConfigure(uintptr_t sid, struct xdg_surface *surf, uint32_t serial);
extern void uitkWlTopConfigure(uintptr_t sid, struct xdg_toplevel *top, int32_t w, int32_t h);
extern void uitkWlTopClose(uintptr_t sid);
extern void uitkWlSeatCaps(uintptr_t id, struct wl_seat *seat, uint32_t caps);
extern void uitkWlPtrEnter(uintptr_t id, struct wl_surface *surf, wl_fixed_t x, wl_fixed_t y);
extern void uitkWlPtrLeave(uintptr_t id);
extern void uitkWlPtrMotion(uintptr_t id, wl_fixed_t x, wl_fixed_t y);
extern void uitkWlPtrButton(uintptr_t id, uint32_t button, uint32_t state);
extern void uitkWlPtrAxis(uintptr_t id, uint32_t axis, wl_fixed_t value);
extern void uitkWlKeymap(uintptr_t id, uint32_t format, int32_t fd, uint32_t size);
extern void uitkWlKeyEnter(uintptr_t id, struct wl_surface *surf);
extern void uitkWlKeyLeave(uintptr_t id);
extern void uitkWlKey(uintptr_t id, uint32_t key, uint32_t state);
extern void uitkWlKeyMods(uintptr_t id, uint32_t depressed, uint32_t latched, uint32_t locked, uint32_t group);
extern void uitkWlBufRelease(uintptr_t sid, int slot);

static int ui_wl_probe(void) {
	struct wl_display *d = wl_display_connect(NULL);
	if (!d) return 0;
	wl_display_disconnect(d);
	return 1;
}

static struct wl_display *ui_wl_connect(void) { return wl_display_connect(NULL); }
static void ui_wl_disconnect(struct wl_display *d) { if (d) wl_display_disconnect(d); }
static struct wl_registry *ui_wl_registry(struct wl_display *d) { return wl_display_get_registry(d); }
static int ui_wl_roundtrip(struct wl_display *d) { return wl_display_roundtrip(d); }
static int ui_wl_flush(struct wl_display *d) { return wl_display_flush(d); }

static void uitk_reg_global(void *data, struct wl_registry *reg, uint32_t name, const char *iface, uint32_t ver) {
	uitkWlRegistryGlobal((uintptr_t)data, reg, name, (char *)iface, ver);
}
static void uitk_reg_remove(void *data, struct wl_registry *reg, uint32_t name) {
	(void)data; (void)reg; (void)name;
}
static const struct wl_registry_listener uitk_reg_listener = {
	.global = uitk_reg_global,
	.global_remove = uitk_reg_remove,
};
static void ui_wl_reg_listen(struct wl_registry *r, uintptr_t id) {
	wl_registry_add_listener(r, &uitk_reg_listener, (void*)id);
}

static void *ui_wl_bind(struct wl_registry *r, uint32_t name, const struct wl_interface *iface, uint32_t ver) {
	return wl_registry_bind(r, name, iface, ver);
}
static const struct wl_interface *ui_wl_compositor_iface(void) { return &wl_compositor_interface; }
static const struct wl_interface *ui_wl_shm_iface(void) { return &wl_shm_interface; }
static const struct wl_interface *ui_wl_seat_iface(void) { return &wl_seat_interface; }
static const struct wl_interface *ui_wl_xdg_iface(void) { return &xdg_wm_base_interface; }

static void uitk_ping(void *data, struct xdg_wm_base *wm, uint32_t serial) {
	uitkWlPing((uintptr_t)data, wm, serial);
}
static const struct xdg_wm_base_listener uitk_wm_listener = { .ping = uitk_ping };
static void ui_wl_wm_listen(struct xdg_wm_base *wm, uintptr_t id) {
	xdg_wm_base_add_listener(wm, &uitk_wm_listener, (void*)id);
}
static void ui_wl_pong(struct xdg_wm_base *wm, uint32_t serial) { xdg_wm_base_pong(wm, serial); }

static struct wl_surface *ui_wl_surface(struct wl_compositor *c) { return wl_compositor_create_surface(c); }
static struct xdg_surface *ui_wl_xdg_surface(struct xdg_wm_base *wm, struct wl_surface *s) {
	return xdg_wm_base_get_xdg_surface(wm, s);
}
static struct xdg_toplevel *ui_wl_toplevel(struct xdg_surface *s) { return xdg_surface_get_toplevel(s); }
static void ui_wl_ack(struct xdg_surface *s, uint32_t serial) { xdg_surface_ack_configure(s, serial); }
static void ui_wl_set_title(struct xdg_toplevel *t, const char *title) { xdg_toplevel_set_title(t, title); }
static void ui_wl_set_app_id(struct xdg_toplevel *t, const char *id) { xdg_toplevel_set_app_id(t, id); }
static void ui_wl_set_min(struct xdg_toplevel *t, int w, int h) { xdg_toplevel_set_min_size(t, w, h); }
static void ui_wl_commit(struct wl_surface *s) { wl_surface_commit(s); }
static void ui_wl_attach(struct wl_surface *s, struct wl_buffer *b) { wl_surface_attach(s, b, 0, 0); }
static void ui_wl_damage(struct wl_surface *s, int x, int y, int w, int h) {
	if (wl_surface_get_version(s) >= 4) {
		wl_surface_damage_buffer(s, x, y, w, h);
	} else {
		wl_surface_damage(s, x, y, w, h);
	}
}

static void uitk_xdg_cfg(void *data, struct xdg_surface *surf, uint32_t serial) {
	uitkWlXdgConfigure((uintptr_t)data, surf, serial);
}
static const struct xdg_surface_listener uitk_xdg_listener = { .configure = uitk_xdg_cfg };
static void ui_wl_xdg_listen(struct xdg_surface *s, uintptr_t sid) {
	xdg_surface_add_listener(s, &uitk_xdg_listener, (void*)sid);
}

static void uitk_top_cfg(void *data, struct xdg_toplevel *top, int32_t w, int32_t h, struct wl_array *states) {
	(void)states;
	uitkWlTopConfigure((uintptr_t)data, top, w, h);
}
static void uitk_top_close(void *data, struct xdg_toplevel *top) {
	(void)top;
	uitkWlTopClose((uintptr_t)data);
}
static void uitk_top_bounds(void *data, struct xdg_toplevel *top, int32_t w, int32_t h) {
	(void)data; (void)top; (void)w; (void)h;
}
static void uitk_top_caps(void *data, struct xdg_toplevel *top, struct wl_array *caps) {
	(void)data; (void)top; (void)caps;
}
static const struct xdg_toplevel_listener uitk_top_listener = {
	.configure = uitk_top_cfg,
	.close = uitk_top_close,
	.configure_bounds = uitk_top_bounds,
	.wm_capabilities = uitk_top_caps,
};
static void ui_wl_top_listen(struct xdg_toplevel *t, uintptr_t sid) {
	xdg_toplevel_add_listener(t, &uitk_top_listener, (void*)sid);
}

static void uitk_seat_caps(void *data, struct wl_seat *seat, uint32_t caps) {
	uitkWlSeatCaps((uintptr_t)data, seat, caps);
}
static void uitk_seat_name(void *data, struct wl_seat *seat, const char *name) {
	(void)data; (void)seat; (void)name;
}
static const struct wl_seat_listener uitk_seat_listener = {
	.capabilities = uitk_seat_caps,
	.name = uitk_seat_name,
};
static void ui_wl_seat_listen(struct wl_seat *s, uintptr_t id) {
	wl_seat_add_listener(s, &uitk_seat_listener, (void*)id);
}
static struct wl_pointer *ui_wl_pointer(struct wl_seat *s) { return wl_seat_get_pointer(s); }
static struct wl_keyboard *ui_wl_keyboard(struct wl_seat *s) { return wl_seat_get_keyboard(s); }

static void uitk_ptr_enter(void *data, struct wl_pointer *p, uint32_t serial, struct wl_surface *surf, wl_fixed_t x, wl_fixed_t y) {
	(void)p; (void)serial;
	uitkWlPtrEnter((uintptr_t)data, surf, x, y);
}
static void uitk_ptr_leave(void *data, struct wl_pointer *p, uint32_t serial, struct wl_surface *surf) {
	(void)p; (void)serial; (void)surf;
	uitkWlPtrLeave((uintptr_t)data);
}
static void uitk_ptr_motion(void *data, struct wl_pointer *p, uint32_t time, wl_fixed_t x, wl_fixed_t y) {
	(void)p; (void)time;
	uitkWlPtrMotion((uintptr_t)data, x, y);
}
static void uitk_ptr_button(void *data, struct wl_pointer *p, uint32_t serial, uint32_t time, uint32_t button, uint32_t state) {
	(void)p; (void)serial; (void)time;
	uitkWlPtrButton((uintptr_t)data, button, state);
}
static void uitk_ptr_axis(void *data, struct wl_pointer *p, uint32_t time, uint32_t axis, wl_fixed_t value) {
	(void)p; (void)time;
	uitkWlPtrAxis((uintptr_t)data, axis, value);
}
static void uitk_ptr_frame(void *data, struct wl_pointer *p) { (void)data; (void)p; }
static void uitk_ptr_axis_src(void *data, struct wl_pointer *p, uint32_t src) { (void)data; (void)p; (void)src; }
static void uitk_ptr_axis_stop(void *data, struct wl_pointer *p, uint32_t time, uint32_t axis) {
	(void)data; (void)p; (void)time; (void)axis;
}
static void uitk_ptr_axis_disc(void *data, struct wl_pointer *p, uint32_t axis, int32_t disc) {
	(void)data; (void)p; (void)axis; (void)disc;
}
static const struct wl_pointer_listener uitk_ptr_listener = {
	.enter = uitk_ptr_enter,
	.leave = uitk_ptr_leave,
	.motion = uitk_ptr_motion,
	.button = uitk_ptr_button,
	.axis = uitk_ptr_axis,
	.frame = uitk_ptr_frame,
	.axis_source = uitk_ptr_axis_src,
	.axis_stop = uitk_ptr_axis_stop,
	.axis_discrete = uitk_ptr_axis_disc,
};
static void ui_wl_ptr_listen(struct wl_pointer *p, uintptr_t id) {
	wl_pointer_add_listener(p, &uitk_ptr_listener, (void*)id);
}

static void uitk_kb_map(void *data, struct wl_keyboard *k, uint32_t format, int32_t fd, uint32_t size) {
	(void)k;
	uitkWlKeymap((uintptr_t)data, format, fd, size);
}
static void uitk_kb_enter(void *data, struct wl_keyboard *k, uint32_t serial, struct wl_surface *surf, struct wl_array *keys) {
	(void)k; (void)serial; (void)keys;
	uitkWlKeyEnter((uintptr_t)data, surf);
}
static void uitk_kb_leave(void *data, struct wl_keyboard *k, uint32_t serial, struct wl_surface *surf) {
	(void)k; (void)serial; (void)surf;
	uitkWlKeyLeave((uintptr_t)data);
}
static void uitk_kb_key(void *data, struct wl_keyboard *k, uint32_t serial, uint32_t time, uint32_t key, uint32_t state) {
	(void)k; (void)serial; (void)time;
	uitkWlKey((uintptr_t)data, key, state);
}
static void uitk_kb_mods(void *data, struct wl_keyboard *k, uint32_t serial, uint32_t dep, uint32_t lat, uint32_t lock, uint32_t group) {
	(void)k; (void)serial;
	uitkWlKeyMods((uintptr_t)data, dep, lat, lock, group);
}
static void uitk_kb_repeat(void *data, struct wl_keyboard *k, int32_t rate, int32_t delay) {
	(void)data; (void)k; (void)rate; (void)delay;
}
static const struct wl_keyboard_listener uitk_kb_listener = {
	.keymap = uitk_kb_map,
	.enter = uitk_kb_enter,
	.leave = uitk_kb_leave,
	.key = uitk_kb_key,
	.modifiers = uitk_kb_mods,
	.repeat_info = uitk_kb_repeat,
};
static void ui_wl_kb_listen(struct wl_keyboard *k, uintptr_t id) {
	wl_keyboard_add_listener(k, &uitk_kb_listener, (void*)id);
}

static int ui_wl_memfd(size_t size, void **map) {
	char tmpl[64];
	snprintf(tmpl, sizeof(tmpl), "/dev/shm/uitk-wl-XXXXXX");
	int fd = mkstemp(tmpl);
	if (fd < 0) {
		snprintf(tmpl, sizeof(tmpl), "/tmp/uitk-wl-XXXXXX");
		fd = mkstemp(tmpl);
	}
	if (fd >= 0) {
		int fl = fcntl(fd, F_GETFD);
		if (fl >= 0) fcntl(fd, F_SETFD, fl | FD_CLOEXEC);
	}
	if (fd < 0) return -1;
	unlink(tmpl);
	if (ftruncate(fd, (off_t)size) != 0) { close(fd); return -1; }
	void *p = mmap(NULL, size, PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
	if (p == MAP_FAILED) { close(fd); return -1; }
	*map = p;
	return fd;
}

static struct wl_buffer *ui_wl_buffer(struct wl_shm *shm, int fd, int w, int h, int stride, size_t size) {
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, (int32_t)size);
	if (!pool) return NULL;
	struct wl_buffer *buf = wl_shm_pool_create_buffer(pool, 0, w, h, stride, WL_SHM_FORMAT_ARGB8888);
	wl_shm_pool_destroy(pool);
	return buf;
}

static void uitk_buf_rel(void *data, struct wl_buffer *buf) {
	(void)buf;
	uintptr_t packed = (uintptr_t)data;
	uitkWlBufRelease(packed >> 8, (int)(packed & 0xff));
}
static const struct wl_buffer_listener uitk_buf_listener = { .release = uitk_buf_rel };
static void ui_wl_buf_listen(struct wl_buffer *b, uintptr_t sid, int slot) {
	wl_buffer_add_listener(b, &uitk_buf_listener, (void*)((sid << 8) | (uintptr_t)(slot & 0xff)));
}

static void ui_wl_munmap(void *p, size_t n) { if (p && p != MAP_FAILED) munmap(p, n); }
static void ui_wl_close_fd(int fd) { if (fd >= 0) close(fd); }
static void ui_wl_buf_destroy(struct wl_buffer *b) { if (b) wl_buffer_destroy(b); }
static void ui_wl_surface_destroy(struct wl_surface *s) { if (s) wl_surface_destroy(s); }
static void ui_wl_xdg_destroy(struct xdg_surface *s) { if (s) xdg_surface_destroy(s); }
static void ui_wl_top_destroy(struct xdg_toplevel *t) { if (t) xdg_toplevel_destroy(t); }
static void ui_wl_ptr_destroy(struct wl_pointer *p) { if (p) wl_pointer_destroy(p); }
static void ui_wl_kb_destroy(struct wl_keyboard *k) { if (k) wl_keyboard_destroy(k); }
static void ui_wl_seat_destroy(struct wl_seat *s) { if (s) wl_seat_destroy(s); }
static void ui_wl_shm_destroy(struct wl_shm *s) { if (s) wl_shm_destroy(s); }
static void ui_wl_comp_destroy(struct wl_compositor *c) { if (c) wl_compositor_destroy(c); }
static void ui_wl_wm_destroy(struct xdg_wm_base *w) { if (w) xdg_wm_base_destroy(w); }
static void ui_wl_reg_destroy(struct wl_registry *r) { if (r) wl_registry_destroy(r); }

static int ui_wl_pump(struct wl_display *d) {
	if (!d) return -1;
	if (wl_display_prepare_read(d) != 0) {
		return wl_display_dispatch_pending(d);
	}
	wl_display_flush(d);
	struct pollfd pfd = { .fd = wl_display_get_fd(d), .events = POLLIN };
	int n = poll(&pfd, 1, 0);
	if (n > 0) {
		wl_display_read_events(d);
	} else {
		wl_display_cancel_read(d);
	}
	return wl_display_dispatch_pending(d);
}

static double ui_wl_fixed(wl_fixed_t v) { return wl_fixed_to_double(v); }

static struct xkb_context *ui_xkb_ctx(void) { return xkb_context_new(XKB_CONTEXT_NO_FLAGS); }
static void ui_xkb_ctx_unref(struct xkb_context *c) { if (c) xkb_context_unref(c); }
static struct xkb_keymap *ui_xkb_map(struct xkb_context *ctx, int fd, uint32_t size) {
	if (!ctx || fd < 0 || size == 0) { if (fd >= 0) close(fd); return NULL; }
	char *map = mmap(NULL, size, PROT_READ, MAP_PRIVATE, fd, 0);
	close(fd);
	if (map == MAP_FAILED) return NULL;
	struct xkb_keymap *km = xkb_keymap_new_from_string(ctx, map, XKB_KEYMAP_FORMAT_TEXT_V1, XKB_KEYMAP_COMPILE_NO_FLAGS);
	munmap(map, size);
	return km;
}
static void ui_xkb_map_unref(struct xkb_keymap *m) { if (m) xkb_keymap_unref(m); }
static struct xkb_state *ui_xkb_state(struct xkb_keymap *m) { return m ? xkb_state_new(m) : NULL; }
static void ui_xkb_state_unref(struct xkb_state *s) { if (s) xkb_state_unref(s); }
static void ui_xkb_update_mask(struct xkb_state *s, uint32_t dep, uint32_t lat, uint32_t lock, uint32_t group) {
	if (s) xkb_state_update_mask(s, dep, lat, lock, 0, 0, group);
}
static void ui_xkb_update_key(struct xkb_state *s, uint32_t key, int pressed) {
	if (!s) return;
	xkb_state_update_key(s, (xkb_keycode_t)(key + 8), pressed ? XKB_KEY_DOWN : XKB_KEY_UP);
}
static uint32_t ui_xkb_sym(struct xkb_state *s, uint32_t key) {
	if (!s) return 0;
	return (uint32_t)xkb_state_key_get_one_sym(s, (xkb_keycode_t)(key + 8));
}
static int ui_xkb_utf8(struct xkb_state *s, uint32_t key, char *buf, int n) {
	if (!s) return 0;
	return xkb_state_key_get_utf8(s, (xkb_keycode_t)(key + 8), buf, (size_t)n);
}
static int ui_xkb_mod(struct xkb_state *s, const char *name) {
	if (!s) return 0;
	return xkb_state_mod_name_is_active(s, name, XKB_STATE_MODS_EFFECTIVE) > 0;
}
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// WaylandBackend presents paintengine2d pixmaps through wl_shm and
// xdg-shell toplevels.
type WaylandBackend struct{}

func (WaylandBackend) Name() string { return "wayland" }

func waylandProbe() bool {
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("XDG_RUNTIME_DIR") == "" {
		return false
	}
	return C.ui_wl_probe() != 0
}

func (WaylandBackend) NewSurface(opts WindowOptions) (Surface, error) {
	if opts.Headless {
		return NewOffscreen(opts), nil
	}
	c, err := wlRetain()
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
	s := &wlSurface{
		conn:  c,
		title: title,
		img:   paintengine2d.NewImage(w, h),
		wantW: w,
		wantH: h,
	}
	wlMu.Lock()
	wlNextSurf++
	s.id = wlNextSurf
	wlSurfaces[s.id] = s
	if c.compositor == nil || c.shm == nil || c.wm == nil {
		delete(wlSurfaces, s.id)
		wlMu.Unlock()
		c.release()
		return nil, fmt.Errorf("platform: Wayland compositor missing wl_compositor, wl_shm, or xdg_wm_base")
	}
	s.surf = C.ui_wl_surface(c.compositor)
	if s.surf != nil {
		wlByNative[uintptr(unsafe.Pointer(s.surf))] = s.id
	}
	s.xdg = C.ui_wl_xdg_surface(c.wm, s.surf)
	C.ui_wl_xdg_listen(s.xdg, C.uintptr_t(s.id))
	s.top = C.ui_wl_toplevel(s.xdg)
	C.ui_wl_top_listen(s.top, C.uintptr_t(s.id))
	ct := C.CString(title)
	C.ui_wl_set_title(s.top, ct)
	C.free(unsafe.Pointer(ct))
	app := C.CString("uitoolkit")
	C.ui_wl_set_app_id(s.top, app)
	C.free(unsafe.Pointer(app))
	mw, mh := opts.MinWidth, opts.MinHeight
	if mw < 1 {
		mw = 200
	}
	if mh < 1 {
		mh = 120
	}
	C.ui_wl_set_min(s.top, C.int(mw), C.int(mh))
	C.ui_wl_commit(s.surf)
	wlMu.Unlock()
	C.ui_wl_roundtrip(c.dpy)
	if !s.configured {
		C.ui_wl_roundtrip(c.dpy)
	}
	return s, nil
}

type wlConn struct {
	id         int
	dpy        *C.struct_wl_display
	reg        *C.struct_wl_registry
	compositor *C.struct_wl_compositor
	shm        *C.struct_wl_shm
	wm         *C.struct_xdg_wm_base
	seat       *C.struct_wl_seat
	pointer    *C.struct_wl_pointer
	keyboard   *C.struct_wl_keyboard
	xkbCtx     *C.struct_xkb_context
	xkbMap     *C.struct_xkb_keymap
	xkbState   *C.struct_xkb_state
	refs       int
	ptrSurf    int
	keySurf    int
	px, py     float32
	mods       Modifiers
}

type wlSlot struct {
	buf  *C.struct_wl_buffer
	mem  unsafe.Pointer
	size int
	fd   int
	w, h int
	busy bool
}

type wlSurface struct {
	id         int
	conn       *wlConn
	surf       *C.struct_wl_surface
	xdg        *C.struct_xdg_surface
	top        *C.struct_xdg_toplevel
	title      string
	img        *paintengine2d.Image
	slots      [2]wlSlot
	wantW      int
	wantH      int
	configured bool
	closed     bool
	queue      []Event
}

var (
	wlMu       sync.Mutex
	wlConns    = map[int]*wlConn{}
	wlSurfaces = map[int]*wlSurface{}
	wlByNative = map[uintptr]int{}
	wlNext     int
	wlNextSurf int
	wlc        *wlConn
)

func wlRetain() (*wlConn, error) {
	wlMu.Lock()
	defer wlMu.Unlock()
	if wlc != nil {
		wlc.refs++
		return wlc, nil
	}
	dpy := C.ui_wl_connect()
	if dpy == nil {
		return nil, fmt.Errorf("platform: wl_display_connect failed (set WAYLAND_DISPLAY or use headless)")
	}
	c := &wlConn{dpy: dpy, refs: 1, xkbCtx: C.ui_xkb_ctx()}
	wlNext++
	c.id = wlNext
	wlConns[c.id] = c
	wlc = c
	c.reg = C.ui_wl_registry(dpy)
	C.ui_wl_reg_listen(c.reg, C.uintptr_t(c.id))
	wlMu.Unlock()
	C.ui_wl_roundtrip(dpy)
	C.ui_wl_roundtrip(dpy)
	wlMu.Lock()
	if c.compositor == nil || c.shm == nil || c.wm == nil {
		c.closeLocked()
		return nil, fmt.Errorf("platform: Wayland registry missing compositor/shm/xdg-shell")
	}
	return c, nil
}

func (c *wlConn) release() {
	wlMu.Lock()
	defer wlMu.Unlock()
	c.refs--
	if c.refs <= 0 {
		c.closeLocked()
	}
}

func (c *wlConn) closeLocked() {
	if c.keyboard != nil {
		C.ui_wl_kb_destroy(c.keyboard)
		c.keyboard = nil
	}
	if c.pointer != nil {
		C.ui_wl_ptr_destroy(c.pointer)
		c.pointer = nil
	}
	if c.seat != nil {
		C.ui_wl_seat_destroy(c.seat)
		c.seat = nil
	}
	C.ui_xkb_state_unref(c.xkbState)
	C.ui_xkb_map_unref(c.xkbMap)
	C.ui_xkb_ctx_unref(c.xkbCtx)
	c.xkbState, c.xkbMap, c.xkbCtx = nil, nil, nil
	if c.wm != nil {
		C.ui_wl_wm_destroy(c.wm)
		c.wm = nil
	}
	if c.shm != nil {
		C.ui_wl_shm_destroy(c.shm)
		c.shm = nil
	}
	if c.compositor != nil {
		C.ui_wl_comp_destroy(c.compositor)
		c.compositor = nil
	}
	if c.reg != nil {
		C.ui_wl_reg_destroy(c.reg)
		c.reg = nil
	}
	if c.dpy != nil {
		C.ui_wl_disconnect(c.dpy)
		c.dpy = nil
	}
	delete(wlConns, c.id)
	if wlc == c {
		wlc = nil
	}
}

func (s *wlSurface) Title() string                { return s.title }
func (s *wlSurface) Size() (w, h int)             { return s.img.Width, s.img.Height }
func (s *wlSurface) Buffer() *paintengine2d.Image { return s.img }
func (s *wlSurface) Closed() bool                 { return s.closed }
func (s *wlSurface) Scale() float32               { return 1 }

func (s *wlSurface) SetTitle(title string) {
	s.title = title
	if s.top == nil {
		return
	}
	ct := C.CString(title)
	C.ui_wl_set_title(s.top, ct)
	C.free(unsafe.Pointer(ct))
}

func (s *wlSurface) Resize(w, h int) error {
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
	return nil
}

func (s *wlSurface) Present(dirty []paintengine2d.Rect) error {
	if s.closed || s.surf == nil || s.conn == nil || s.conn.dpy == nil {
		return nil
	}
	if !s.configured {
		return nil
	}
	if s.wantW > 0 && s.wantH > 0 && (s.wantW != s.img.Width || s.wantH != s.img.Height) {
		s.img = paintengine2d.NewImage(s.wantW, s.wantH)
		s.queue = append(s.queue, Event{Kind: EventResize, Width: s.wantW, Height: s.wantH})
	}
	slot := s.pickSlot()
	if err := s.ensureSlot(slot, s.img.Width, s.img.Height); err != nil {
		return err
	}
	if len(dirty) == 0 {
		dirty = []paintengine2d.Rect{paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height))}
	}
	for _, r := range dirty {
		s.copyRect(slot, r)
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
		C.ui_wl_damage(s.surf, C.int(x0), C.int(y0), C.int(x1-x0), C.int(y1-y0))
	}
	C.ui_wl_attach(s.surf, s.slots[slot].buf)
	C.ui_wl_commit(s.surf)
	s.slots[slot].busy = true
	C.ui_wl_flush(s.conn.dpy)
	return nil
}

func (s *wlSurface) pickSlot() int {
	for i := range s.slots {
		if s.slots[i].buf != nil && !s.slots[i].busy {
			return i
		}
	}
	for i := range s.slots {
		if s.slots[i].buf == nil {
			return i
		}
	}
	return 0
}

func (s *wlSurface) ensureSlot(i, w, h int) error {
	sl := &s.slots[i]
	if sl.buf != nil && sl.w == w && sl.h == h {
		return nil
	}
	s.destroySlot(i)
	stride := w * 4
	size := stride * h
	if size < 4 {
		size = 4
	}
	var mem unsafe.Pointer
	fd := int(C.ui_wl_memfd(C.size_t(size), &mem))
	if fd < 0 || mem == nil {
		return fmt.Errorf("platform: wl_shm memfd failed")
	}
	buf := C.ui_wl_buffer(s.conn.shm, C.int(fd), C.int(w), C.int(h), C.int(stride), C.size_t(size))
	C.ui_wl_close_fd(C.int(fd))
	if buf == nil {
		C.ui_wl_munmap(mem, C.size_t(size))
		return fmt.Errorf("platform: wl_shm_pool_create_buffer failed")
	}
	C.ui_wl_buf_listen(buf, C.uintptr_t(s.id), C.int(i))
	s.slots[i] = wlSlot{buf: buf, mem: mem, size: size, fd: -1, w: w, h: h}
	return nil
}

func (s *wlSurface) destroySlot(i int) {
	sl := &s.slots[i]
	if sl.buf != nil {
		C.ui_wl_buf_destroy(sl.buf)
		sl.buf = nil
	}
	if sl.mem != nil {
		C.ui_wl_munmap(sl.mem, C.size_t(sl.size))
		sl.mem = nil
	}
	*sl = wlSlot{}
}

func (s *wlSurface) copyRect(slot int, r paintengine2d.Rect) {
	sl := s.slots[slot]
	if sl.mem == nil {
		return
	}
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
	dst := unsafe.Slice((*byte)(sl.mem), sl.size)
	stride := sl.w * 4
	for y := y0; y < y1; y++ {
		row := y * stride
		for x := x0; x < x1; x++ {
			n := s.img.NRGBAAt(x, y)
			packXPixel(dst[row+x*4:], n.R, n.G, n.B, n.A, false, 0x00ff0000, 0x0000ff00, 0x000000ff)
		}
	}
}

func (s *wlSurface) Poll() []Event {
	if s.closed || s.conn == nil || s.conn.dpy == nil {
		return nil
	}
	C.ui_wl_pump(s.conn.dpy)
	wlMu.Lock()
	ev := s.queue
	s.queue = nil
	wlMu.Unlock()
	return ev
}

func (s *wlSurface) Close() error {
	if s.closed && s.surf == nil {
		return nil
	}
	s.closed = true
	wlMu.Lock()
	delete(wlSurfaces, s.id)
	if s.surf != nil {
		delete(wlByNative, uintptr(unsafe.Pointer(s.surf)))
	}
	wlMu.Unlock()
	s.destroySlot(0)
	s.destroySlot(1)
	if s.top != nil {
		C.ui_wl_top_destroy(s.top)
		s.top = nil
	}
	if s.xdg != nil {
		C.ui_wl_xdg_destroy(s.xdg)
		s.xdg = nil
	}
	if s.surf != nil {
		C.ui_wl_surface_destroy(s.surf)
		s.surf = nil
	}
	if s.conn != nil {
		C.ui_wl_flush(s.conn.dpy)
		s.conn.release()
	}
	return nil
}

func (s *wlSurface) push(ev Event) {
	s.queue = append(s.queue, ev)
}

func wlConnBy(id C.uintptr_t) *wlConn {
	return wlConns[int(id)]
}

func wlSurfBy(id C.uintptr_t) *wlSurface {
	return wlSurfaces[int(id)]
}

func wlSurfNative(surf *C.struct_wl_surface) *wlSurface {
	if surf == nil {
		return nil
	}
	id, ok := wlByNative[uintptr(unsafe.Pointer(surf))]
	if !ok {
		return nil
	}
	return wlSurfaces[id]
}

//export uitkWlRegistryGlobal
func uitkWlRegistryGlobal(id C.uintptr_t, reg *C.struct_wl_registry, name C.uint32_t, iface *C.char, ver C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	is := C.GoString(iface)
	switch is {
	case "wl_compositor":
		v := ver
		if v > 4 {
			v = 4
		}
		c.compositor = (*C.struct_wl_compositor)(C.ui_wl_bind(reg, name, C.ui_wl_compositor_iface(), v))
	case "wl_shm":
		c.shm = (*C.struct_wl_shm)(C.ui_wl_bind(reg, name, C.ui_wl_shm_iface(), 1))
	case "xdg_wm_base":
		v := ver
		if v > 3 {
			v = 3
		}
		if v < 1 {
			v = 1
		}
		c.wm = (*C.struct_xdg_wm_base)(C.ui_wl_bind(reg, name, C.ui_wl_xdg_iface(), v))
		if c.wm != nil {
			C.ui_wl_wm_listen(c.wm, id)
		}
	case "wl_seat":
		v := ver
		if v > 5 {
			v = 5
		}
		if v < 1 {
			v = 1
		}
		c.seat = (*C.struct_wl_seat)(C.ui_wl_bind(reg, name, C.ui_wl_seat_iface(), v))
		if c.seat != nil {
			C.ui_wl_seat_listen(c.seat, id)
		}
	}
}

//export uitkWlPing
func uitkWlPing(id C.uintptr_t, wm *C.struct_xdg_wm_base, serial C.uint32_t) {
	_ = id
	C.ui_wl_pong(wm, serial)
}

//export uitkWlXdgConfigure
func uitkWlXdgConfigure(sid C.uintptr_t, surf *C.struct_xdg_surface, serial C.uint32_t) {
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	C.ui_wl_ack(surf, serial)
	s.configured = true
	if s.surf != nil {
		wlByNative[uintptr(unsafe.Pointer(s.surf))] = s.id
	}
}

//export uitkWlTopConfigure
func uitkWlTopConfigure(sid C.uintptr_t, top *C.struct_xdg_toplevel, w, h C.int32_t) {
	_ = top
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	if w > 0 && h > 0 {
		s.wantW, s.wantH = int(w), int(h)
		if s.img != nil && (s.img.Width != s.wantW || s.img.Height != s.wantH) {
			s.push(Event{Kind: EventResize, Width: s.wantW, Height: s.wantH})
		}
	}
}

//export uitkWlTopClose
func uitkWlTopClose(sid C.uintptr_t) {
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	s.closed = true
	s.push(Event{Kind: EventClose})
}

//export uitkWlSeatCaps
func uitkWlSeatCaps(id C.uintptr_t, seat *C.struct_wl_seat, caps C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	const (
		capPointer  = 1
		capKeyboard = 2
	)
	if caps&capPointer != 0 && c.pointer == nil {
		c.pointer = C.ui_wl_pointer(seat)
		if c.pointer != nil {
			C.ui_wl_ptr_listen(c.pointer, id)
		}
	}
	if caps&capKeyboard != 0 && c.keyboard == nil {
		c.keyboard = C.ui_wl_keyboard(seat)
		if c.keyboard != nil {
			C.ui_wl_kb_listen(c.keyboard, id)
		}
	}
}

//export uitkWlPtrEnter
func uitkWlPtrEnter(id C.uintptr_t, surf *C.struct_wl_surface, x, y C.wl_fixed_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.px = float32(C.ui_wl_fixed(x))
	c.py = float32(C.ui_wl_fixed(y))
	if s := wlSurfNative(surf); s != nil {
		c.ptrSurf = s.id
		s.push(Event{Kind: EventMouseMove, Pos: paintengine2d.Pt(c.px, c.py), Mods: c.mods})
	}
}

//export uitkWlPtrLeave
func uitkWlPtrLeave(id C.uintptr_t) {
	c := wlConnBy(id)
	if c != nil {
		c.ptrSurf = 0
	}
}

//export uitkWlPtrMotion
func uitkWlPtrMotion(id C.uintptr_t, x, y C.wl_fixed_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.px = float32(C.ui_wl_fixed(x))
	c.py = float32(C.ui_wl_fixed(y))
	if s := wlSurfaces[c.ptrSurf]; s != nil {
		s.push(Event{Kind: EventMouseMove, Pos: paintengine2d.Pt(c.px, c.py), Mods: c.mods})
	}
}

//export uitkWlPtrButton
func uitkWlPtrButton(id C.uintptr_t, button, state C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.ptrSurf]
	if s == nil {
		return
	}
	kind := EventMouseUp
	if state == 1 {
		kind = EventMouseDown
	}
	s.push(Event{Kind: kind, Pos: paintengine2d.Pt(c.px, c.py), Button: wlButton(uint32(button)), Mods: c.mods})
}

//export uitkWlPtrAxis
func uitkWlPtrAxis(id C.uintptr_t, axis C.uint32_t, value C.wl_fixed_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.ptrSurf]
	if s == nil {
		return
	}
	v := float32(C.ui_wl_fixed(value)) * 4
	ev := Event{Kind: EventScroll, Pos: paintengine2d.Pt(c.px, c.py), Mods: c.mods}
	if axis == 0 {
		ev.Scroll = paintengine2d.Pt(0, v)
	} else {
		ev.Scroll = paintengine2d.Pt(v, 0)
	}
	s.push(ev)
}

//export uitkWlKeymap
func uitkWlKeymap(id C.uintptr_t, format C.uint32_t, fd C.int32_t, size C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		C.ui_wl_close_fd(C.int(fd))
		return
	}
	if format != 1 {
		C.ui_wl_close_fd(C.int(fd))
		return
	}
	C.ui_xkb_state_unref(c.xkbState)
	C.ui_xkb_map_unref(c.xkbMap)
	c.xkbMap = C.ui_xkb_map(c.xkbCtx, fd, size)
	c.xkbState = C.ui_xkb_state(c.xkbMap)
}

//export uitkWlKeyEnter
func uitkWlKeyEnter(id C.uintptr_t, surf *C.struct_wl_surface) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if s := wlSurfNative(surf); s != nil {
		c.keySurf = s.id
		s.push(Event{Kind: EventFocusIn})
	}
}

//export uitkWlKeyLeave
func uitkWlKeyLeave(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if s := wlSurfaces[c.keySurf]; s != nil {
		s.push(Event{Kind: EventFocusOut})
	}
	c.keySurf = 0
}

//export uitkWlKey
func uitkWlKey(id C.uintptr_t, key, state C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.keySurf]
	if s == nil {
		return
	}
	pressed := state == 1
	C.ui_xkb_update_key(c.xkbState, key, C.int(btoi(pressed)))
	c.mods = wlMods(c)
	ks := uint64(C.ui_xkb_sym(c.xkbState, key))
	ev := Event{Kind: EventKeyUp, Key: KeyFromKeysym(ks), Mods: c.mods}
	if pressed {
		ev.Kind = EventKeyDown
	}
	s.push(ev)
	if pressed && !c.mods.Ctrl() {
		var buf [64]C.char
		n := int(C.ui_xkb_utf8(c.xkbState, key, &buf[0], 64))
		if n > 0 {
			for _, te := range textEvents(C.GoBytes(unsafe.Pointer(&buf[0]), C.int(n)), c.mods) {
				s.push(te)
			}
		}
	}
}

//export uitkWlKeyMods
func uitkWlKeyMods(id C.uintptr_t, dep, lat, lock, group C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	C.ui_xkb_update_mask(c.xkbState, dep, lat, lock, group)
	c.mods = wlMods(c)
}

//export uitkWlBufRelease
func uitkWlBufRelease(sid C.uintptr_t, slot C.int) {
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	i := int(slot)
	if i >= 0 && i < len(s.slots) {
		s.slots[i].busy = false
	}
}

func wlButton(code uint32) MouseButton {
	switch code {
	case 0x110:
		return ButtonLeft
	case 0x111:
		return ButtonRight
	case 0x112:
		return ButtonMiddle
	}
	return ButtonNone
}

func wlMods(c *wlConn) Modifiers {
	var m Modifiers
	if c == nil || c.xkbState == nil {
		return 0
	}
	shift := C.CString("Shift")
	ctrl := C.CString("Control")
	mod1 := C.CString("Mod1")
	logo := C.CString("Mod4")
	defer C.free(unsafe.Pointer(shift))
	defer C.free(unsafe.Pointer(ctrl))
	defer C.free(unsafe.Pointer(mod1))
	defer C.free(unsafe.Pointer(logo))
	if C.ui_xkb_mod(c.xkbState, shift) != 0 {
		m |= ModShift
	}
	if C.ui_xkb_mod(c.xkbState, ctrl) != 0 {
		m |= ModCtrl
	}
	if C.ui_xkb_mod(c.xkbState, mod1) != 0 {
		m |= ModAlt
	}
	if C.ui_xkb_mod(c.xkbState, logo) != 0 {
		m |= ModSuper
	}
	return m
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}
