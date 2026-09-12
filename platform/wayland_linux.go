//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client xkbcommon
#cgo linux CFLAGS: -I${SRCDIR} -I/usr/include/drm
#define _GNU_SOURCE
#include <wayland-client.h>
#include <xkbcommon/xkbcommon.h>
#include <xkbcommon/xkbcommon-compose.h>
#include <fcntl.h>
#include <poll.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>
#include "xdg-shell-client-protocol.h"
#include "text-input-unstable-v3-client-protocol.h"
#include "primary-selection-unstable-v1-client-protocol.h"
#include "xdg-decoration-unstable-v1-client-protocol.h"
#include "fractional-scale-v1-client-protocol.h"
#include "viewporter-client-protocol.h"
#include "wayland_dmabuf.h"
#include "wayland_sync.h"

extern void uitkWlRegistryGlobal(uintptr_t id, struct wl_registry *reg, uint32_t name, char *iface, uint32_t ver);
extern void uitkWlPing(uintptr_t id, struct xdg_wm_base *wm, uint32_t serial);
extern void uitkWlXdgConfigure(uintptr_t sid, struct xdg_surface *surf, uint32_t serial);
extern void uitkWlTopConfigure(uintptr_t sid, struct xdg_toplevel *top, int32_t w, int32_t h, uint32_t flags);
extern void uitkWlTopClose(uintptr_t sid);
extern void uitkWlSeatCaps(uintptr_t id, struct wl_seat *seat, uint32_t caps);
extern void uitkWlPtrEnter(uintptr_t id, uint32_t serial, struct wl_surface *surf, wl_fixed_t x, wl_fixed_t y);
extern void uitkWlPtrLeave(uintptr_t id);
extern void uitkWlPtrMotion(uintptr_t id, wl_fixed_t x, wl_fixed_t y);
extern void uitkWlPtrButton(uintptr_t id, uint32_t button, uint32_t state, uint32_t serial);
extern void uitkWlPtrAxis(uintptr_t id, uint32_t axis, wl_fixed_t value);
extern void uitkWlKeymap(uintptr_t id, uint32_t format, int32_t fd, uint32_t size);
extern void uitkWlKeyEnter(uintptr_t id, struct wl_surface *surf, uint32_t serial);
extern void uitkWlKeyLeave(uintptr_t id);
extern void uitkWlKey(uintptr_t id, uint32_t key, uint32_t state, uint32_t serial);
extern void uitkWlKeyMods(uintptr_t id, uint32_t depressed, uint32_t latched, uint32_t locked, uint32_t group);
extern void uitkWlKeyRepeat(uintptr_t id, int32_t rate, int32_t delay);
extern void uitkWlBufRelease(uintptr_t sid, int slot);
extern void uitkWlExplicitRelease(uintptr_t sid, int slot);
extern void uitkWlOutputScale(uintptr_t id, int32_t factor);
extern void uitkWlDataOffer(uintptr_t id, struct wl_data_offer *offer);
extern void uitkWlDataOfferMime(uintptr_t id, struct wl_data_offer *offer, char *mime);
extern void uitkWlSelection(uintptr_t id, struct wl_data_offer *offer);
extern void uitkWlDataSend(uintptr_t id, int fd);
extern void uitkWlDataCancelled(uintptr_t id);
extern void uitkWlPrimOffer(uintptr_t id, struct zwp_primary_selection_offer_v1 *offer);
extern void uitkWlPrimOfferMime(uintptr_t id, struct zwp_primary_selection_offer_v1 *offer, char *mime);
extern void uitkWlPrimSelection(uintptr_t id, struct zwp_primary_selection_offer_v1 *offer);
extern void uitkWlPrimSend(uintptr_t id, int fd);
extern void uitkWlTIEnter(uintptr_t id, struct wl_surface *surf);
extern void uitkWlTILeave(uintptr_t id);
extern void uitkWlTIPreedit(uintptr_t id, char *text, int32_t begin, int32_t end);
extern void uitkWlTICommit(uintptr_t id, char *text);
extern void uitkWlTIDelete(uintptr_t id, uint32_t before, uint32_t after);
extern void uitkWlTIDone(uintptr_t id);
extern void uitkWlFracScale(uintptr_t sid, uint32_t scale_120);

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
	uint32_t flags = 0;
	if (states) {
		uint32_t *p;
		wl_array_for_each(p, states) {
			if (*p == XDG_TOPLEVEL_STATE_MAXIMIZED) flags |= 1;
			if (*p == XDG_TOPLEVEL_STATE_FULLSCREEN) flags |= 2;
			if (*p == XDG_TOPLEVEL_STATE_RESIZING) flags |= 4;
			if (*p == XDG_TOPLEVEL_STATE_ACTIVATED) flags |= 8;
		}
	}
	uitkWlTopConfigure((uintptr_t)data, top, w, h, flags);
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
	(void)p;
	uitkWlPtrEnter((uintptr_t)data, serial, surf, x, y);
}
static void ui_wl_set_cursor(struct wl_pointer *p, uint32_t serial, struct wl_surface *surf, int32_t hx, int32_t hy) {
	if (p) wl_pointer_set_cursor(p, serial, surf, hx, hy);
}
static struct wl_buffer *ui_wl_argb_buffer(struct wl_shm *shm, int fd, int w, int h, int stride, size_t size) {
	if (!shm) return NULL;
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, (int32_t)size);
	if (!pool) return NULL;
	struct wl_buffer *buf = wl_shm_pool_create_buffer(pool, 0, w, h, stride, WL_SHM_FORMAT_ARGB8888);
	wl_shm_pool_destroy(pool);
	return buf;
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
	(void)p; (void)time;
	uitkWlPtrButton((uintptr_t)data, button, state, serial);
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
	(void)k; (void)keys;
	uitkWlKeyEnter((uintptr_t)data, surf, serial);
}
static void uitk_kb_leave(void *data, struct wl_keyboard *k, uint32_t serial, struct wl_surface *surf) {
	(void)k; (void)serial; (void)surf;
	uitkWlKeyLeave((uintptr_t)data);
}
static void uitk_kb_key(void *data, struct wl_keyboard *k, uint32_t serial, uint32_t time, uint32_t key, uint32_t state) {
	(void)k; (void)time;
	uitkWlKey((uintptr_t)data, key, state, serial);
}
static void uitk_kb_mods(void *data, struct wl_keyboard *k, uint32_t serial, uint32_t dep, uint32_t lat, uint32_t lock, uint32_t group) {
	(void)k; (void)serial;
	uitkWlKeyMods((uintptr_t)data, dep, lat, lock, group);
}
static void uitk_kb_repeat(void *data, struct wl_keyboard *k, int32_t rate, int32_t delay) {
	(void)k;
	uitkWlKeyRepeat((uintptr_t)data, rate, delay);
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
	// XRGB8888: compositor ignores alpha. ARGB8888 + empty/wrong A
	// composites as a fully transparent window on Mutter/Weston.
	struct wl_buffer *buf = wl_shm_pool_create_buffer(pool, 0, w, h, stride, WL_SHM_FORMAT_XRGB8888);
	wl_shm_pool_destroy(pool);
	return buf;
}

static void ui_wl_opaque(struct wl_compositor *c, struct wl_surface *s, int w, int h) {
	if (!c || !s || w < 1 || h < 1) return;
	struct wl_region *r = wl_compositor_create_region(c);
	if (!r) return;
	wl_region_add(r, 0, 0, w, h);
	wl_surface_set_opaque_region(s, r);
	wl_region_destroy(r);
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

static int ui_wl_wait(struct wl_display *d, int ms) {
	if (!d) return -1;
	if (wl_display_prepare_read(d) != 0) {
		return wl_display_dispatch_pending(d);
	}
	wl_display_flush(d);
	struct pollfd pfd = { .fd = wl_display_get_fd(d), .events = POLLIN };
	int n = poll(&pfd, 1, ms);
	if (n > 0) {
		wl_display_read_events(d);
	} else {
		wl_display_cancel_read(d);
	}
	return wl_display_dispatch_pending(d);
}

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

static struct xkb_compose_table *ui_xkb_compose_table(struct xkb_context *ctx) {
	const char *loc = getenv("LC_ALL");
	if (!loc || !loc[0]) loc = getenv("LC_CTYPE");
	if (!loc || !loc[0]) loc = getenv("LANG");
	if (!loc || !loc[0]) loc = "C";
	return xkb_compose_table_new_from_locale(ctx, loc, XKB_COMPOSE_COMPILE_NO_FLAGS);
}
static void ui_xkb_compose_table_unref(struct xkb_compose_table *t) { if (t) xkb_compose_table_unref(t); }
static struct xkb_compose_state *ui_xkb_compose_state(struct xkb_compose_table *t) {
	return t ? xkb_compose_state_new(t, XKB_COMPOSE_STATE_NO_FLAGS) : NULL;
}
static void ui_xkb_compose_state_unref(struct xkb_compose_state *s) { if (s) xkb_compose_state_unref(s); }
static int ui_xkb_compose_feed(struct xkb_compose_state *s, uint32_t sym) {
	if (!s) return XKB_COMPOSE_NOTHING;
	return xkb_compose_state_feed(s, (xkb_keysym_t)sym);
}
static int ui_xkb_compose_status(struct xkb_compose_state *s) {
	return s ? xkb_compose_state_get_status(s) : XKB_COMPOSE_NOTHING;
}
static int ui_xkb_compose_utf8(struct xkb_compose_state *s, char *buf, int n) {
	if (!s) return 0;
	return xkb_compose_state_get_utf8(s, buf, (size_t)n);
}
static void ui_xkb_compose_reset(struct xkb_compose_state *s) { if (s) xkb_compose_state_reset(s); }

static const struct wl_interface *ui_wl_data_man_iface(void) { return &wl_data_device_manager_interface; }
static const struct wl_interface *ui_wl_output_iface(void) { return &wl_output_interface; }
static const struct wl_interface *ui_wl_ti_man_iface(void) { return &zwp_text_input_manager_v3_interface; }
static const struct wl_interface *ui_wl_prim_man_iface(void) { return &zwp_primary_selection_device_manager_v1_interface; }
static const struct wl_interface *ui_wl_deco_man_iface(void) { return &zxdg_decoration_manager_v1_interface; }
static const struct wl_interface *ui_wl_frac_man_iface(void) { return &wp_fractional_scale_manager_v1_interface; }
static const struct wl_interface *ui_wl_viewporter_iface(void) { return &wp_viewporter_interface; }

static struct wl_data_device *ui_wl_data_device(struct wl_data_device_manager *m, struct wl_seat *s) {
	return wl_data_device_manager_get_data_device(m, s);
}
static struct wl_data_source *ui_wl_data_source(struct wl_data_device_manager *m) {
	return wl_data_device_manager_create_data_source(m);
}
static void ui_wl_data_offer_mime(struct wl_data_source *s, const char *m) { wl_data_source_offer(s, m); }
static void ui_wl_set_selection(struct wl_data_device *d, struct wl_data_source *s, uint32_t serial) {
	wl_data_device_set_selection(d, s, serial);
}
static void ui_wl_data_receive(struct wl_data_offer *o, const char *mime, int fd) { wl_data_offer_receive(o, mime, fd); }
static void ui_wl_data_offer_destroy(struct wl_data_offer *o) { if (o) wl_data_offer_destroy(o); }
static void ui_wl_data_source_destroy(struct wl_data_source *s) { if (s) wl_data_source_destroy(s); }
static void ui_wl_data_device_destroy(struct wl_data_device *d) { if (d) wl_data_device_destroy(d); }
static void ui_wl_data_man_destroy(struct wl_data_device_manager *m) { if (m) wl_data_device_manager_destroy(m); }

static void uitk_doffer_offer(void *data, struct wl_data_offer *o, const char *mime) {
	uitkWlDataOfferMime((uintptr_t)data, o, (char*)mime);
}
static void uitk_doffer_src(void *data, struct wl_data_offer *o, uint32_t src) { (void)data; (void)o; (void)src; }
static void uitk_doffer_action(void *data, struct wl_data_offer *o, uint32_t dnd) {
	(void)data; (void)o; (void)dnd;
}
static const struct wl_data_offer_listener uitk_doffer_listener = {
	.offer = uitk_doffer_offer,
	.source_actions = uitk_doffer_src,
	.action = uitk_doffer_action,
};
static void ui_wl_doffer_listen(struct wl_data_offer *o, uintptr_t id) {
	wl_data_offer_add_listener(o, &uitk_doffer_listener, (void*)id);
}

static void uitk_ddev_offer(void *data, struct wl_data_device *d, struct wl_data_offer *o) {
	(void)d;
	uitkWlDataOffer((uintptr_t)data, o);
}
static void uitk_ddev_enter(void *data, struct wl_data_device *d, uint32_t serial, struct wl_surface *s, wl_fixed_t x, wl_fixed_t y, struct wl_data_offer *o) {
	(void)data; (void)d; (void)serial; (void)s; (void)x; (void)y; (void)o;
}
static void uitk_ddev_leave(void *data, struct wl_data_device *d) { (void)data; (void)d; }
static void uitk_ddev_motion(void *data, struct wl_data_device *d, uint32_t time, wl_fixed_t x, wl_fixed_t y) {
	(void)data; (void)d; (void)time; (void)x; (void)y;
}
static void uitk_ddev_drop(void *data, struct wl_data_device *d) { (void)data; (void)d; }
static void uitk_ddev_sel(void *data, struct wl_data_device *d, struct wl_data_offer *o) {
	(void)d;
	uitkWlSelection((uintptr_t)data, o);
}
static const struct wl_data_device_listener uitk_ddev_listener = {
	.data_offer = uitk_ddev_offer,
	.enter = uitk_ddev_enter,
	.leave = uitk_ddev_leave,
	.motion = uitk_ddev_motion,
	.drop = uitk_ddev_drop,
	.selection = uitk_ddev_sel,
};
static void ui_wl_ddev_listen(struct wl_data_device *d, uintptr_t id) {
	wl_data_device_add_listener(d, &uitk_ddev_listener, (void*)id);
}

static void uitk_dsrc_target(void *data, struct wl_data_source *s, const char *mime) { (void)data; (void)s; (void)mime; }
static void uitk_dsrc_send(void *data, struct wl_data_source *s, const char *mime, int32_t fd) {
	(void)s; (void)mime;
	uitkWlDataSend((uintptr_t)data, fd);
}
static void uitk_dsrc_cancelled(void *data, struct wl_data_source *s) {
	(void)s;
	uitkWlDataCancelled((uintptr_t)data);
}
static void uitk_dsrc_dnd_drop(void *data, struct wl_data_source *s) { (void)data; (void)s; }
static void uitk_dsrc_dnd_finish(void *data, struct wl_data_source *s) { (void)data; (void)s; }
static void uitk_dsrc_action(void *data, struct wl_data_source *s, uint32_t a) { (void)data; (void)s; (void)a; }
static const struct wl_data_source_listener uitk_dsrc_listener = {
	.target = uitk_dsrc_target,
	.send = uitk_dsrc_send,
	.cancelled = uitk_dsrc_cancelled,
	.dnd_drop_performed = uitk_dsrc_dnd_drop,
	.dnd_finished = uitk_dsrc_dnd_finish,
	.action = uitk_dsrc_action,
};
static void ui_wl_dsrc_listen(struct wl_data_source *s, uintptr_t id) {
	wl_data_source_add_listener(s, &uitk_dsrc_listener, (void*)id);
}

static struct zwp_primary_selection_device_v1 *ui_wl_prim_device(struct zwp_primary_selection_device_manager_v1 *m, struct wl_seat *s) {
	return zwp_primary_selection_device_manager_v1_get_device(m, s);
}
static struct zwp_primary_selection_source_v1 *ui_wl_prim_source(struct zwp_primary_selection_device_manager_v1 *m) {
	return zwp_primary_selection_device_manager_v1_create_source(m);
}
static void ui_wl_prim_offer_mime(struct zwp_primary_selection_source_v1 *s, const char *m) {
	zwp_primary_selection_source_v1_offer(s, m);
}
static void ui_wl_prim_set(struct zwp_primary_selection_device_v1 *d, struct zwp_primary_selection_source_v1 *s, uint32_t serial) {
	zwp_primary_selection_device_v1_set_selection(d, s, serial);
}
static void ui_wl_prim_receive(struct zwp_primary_selection_offer_v1 *o, const char *mime, int fd) {
	zwp_primary_selection_offer_v1_receive(o, mime, fd);
}
static void ui_wl_prim_offer_destroy(struct zwp_primary_selection_offer_v1 *o) {
	if (o) zwp_primary_selection_offer_v1_destroy(o);
}
static void ui_wl_prim_source_destroy(struct zwp_primary_selection_source_v1 *s) {
	if (s) zwp_primary_selection_source_v1_destroy(s);
}
static void ui_wl_prim_dev_destroy(struct zwp_primary_selection_device_v1 *d) {
	if (d) zwp_primary_selection_device_v1_destroy(d);
}
static void ui_wl_prim_man_destroy(struct zwp_primary_selection_device_manager_v1 *m) {
	if (m) zwp_primary_selection_device_manager_v1_destroy(m);
}

static void uitk_poffer_offer(void *data, struct zwp_primary_selection_offer_v1 *o, const char *mime) {
	uitkWlPrimOfferMime((uintptr_t)data, o, (char*)mime);
}
static const struct zwp_primary_selection_offer_v1_listener uitk_poffer_listener = { .offer = uitk_poffer_offer };
static void ui_wl_poffer_listen(struct zwp_primary_selection_offer_v1 *o, uintptr_t id) {
	zwp_primary_selection_offer_v1_add_listener(o, &uitk_poffer_listener, (void*)id);
}
static void uitk_pdev_offer(void *data, struct zwp_primary_selection_device_v1 *d, struct zwp_primary_selection_offer_v1 *o) {
	(void)d;
	uitkWlPrimOffer((uintptr_t)data, o);
}
static void uitk_pdev_sel(void *data, struct zwp_primary_selection_device_v1 *d, struct zwp_primary_selection_offer_v1 *o) {
	(void)d;
	uitkWlPrimSelection((uintptr_t)data, o);
}
static const struct zwp_primary_selection_device_v1_listener uitk_pdev_listener = {
	.data_offer = uitk_pdev_offer,
	.selection = uitk_pdev_sel,
};
static void ui_wl_pdev_listen(struct zwp_primary_selection_device_v1 *d, uintptr_t id) {
	zwp_primary_selection_device_v1_add_listener(d, &uitk_pdev_listener, (void*)id);
}
static void uitk_psrc_send(void *data, struct zwp_primary_selection_source_v1 *s, const char *mime, int32_t fd) {
	(void)s; (void)mime;
	uitkWlPrimSend((uintptr_t)data, fd);
}
static void uitk_psrc_cancelled(void *data, struct zwp_primary_selection_source_v1 *s) { (void)data; (void)s; }
static const struct zwp_primary_selection_source_v1_listener uitk_psrc_listener = {
	.send = uitk_psrc_send,
	.cancelled = uitk_psrc_cancelled,
};
static void ui_wl_psrc_listen(struct zwp_primary_selection_source_v1 *s, uintptr_t id) {
	zwp_primary_selection_source_v1_add_listener(s, &uitk_psrc_listener, (void*)id);
}

static struct zwp_text_input_v3 *ui_wl_text_input(struct zwp_text_input_manager_v3 *m, struct wl_seat *s) {
	return zwp_text_input_manager_v3_get_text_input(m, s);
}
static void ui_wl_ti_enable(struct zwp_text_input_v3 *t) { if (t) zwp_text_input_v3_enable(t); }
static void ui_wl_ti_disable(struct zwp_text_input_v3 *t) { if (t) zwp_text_input_v3_disable(t); }
static void ui_wl_ti_commit(struct zwp_text_input_v3 *t) { if (t) zwp_text_input_v3_commit(t); }
static void ui_wl_ti_cursor(struct zwp_text_input_v3 *t, int x, int y, int w, int h) {
	if (t) zwp_text_input_v3_set_cursor_rectangle(t, x, y, w, h);
}
static void ui_wl_ti_surround(struct zwp_text_input_v3 *t, const char *text, int32_t cursor, int32_t anchor) {
	if (t) zwp_text_input_v3_set_surrounding_text(t, text, cursor, anchor);
}
static void ui_wl_ti_destroy(struct zwp_text_input_v3 *t) { if (t) zwp_text_input_v3_destroy(t); }
static void ui_wl_ti_man_destroy(struct zwp_text_input_manager_v3 *m) { if (m) zwp_text_input_manager_v3_destroy(m); }

static void uitk_ti_enter(void *data, struct zwp_text_input_v3 *t, struct wl_surface *s) {
	(void)t;
	uitkWlTIEnter((uintptr_t)data, s);
}
static void uitk_ti_leave(void *data, struct zwp_text_input_v3 *t, struct wl_surface *s) {
	(void)t; (void)s;
	uitkWlTILeave((uintptr_t)data);
}
static void uitk_ti_preedit(void *data, struct zwp_text_input_v3 *t, const char *text, int32_t begin, int32_t end) {
	(void)t;
	uitkWlTIPreedit((uintptr_t)data, (char*)text, begin, end);
}
static void uitk_ti_commit(void *data, struct zwp_text_input_v3 *t, const char *text) {
	(void)t;
	uitkWlTICommit((uintptr_t)data, (char*)text);
}
static void uitk_ti_delete(void *data, struct zwp_text_input_v3 *t, uint32_t before, uint32_t after) {
	(void)t;
	uitkWlTIDelete((uintptr_t)data, before, after);
}
static void uitk_ti_done(void *data, struct zwp_text_input_v3 *t, uint32_t serial) {
	(void)t; (void)serial;
	uitkWlTIDone((uintptr_t)data);
}
static const struct zwp_text_input_v3_listener uitk_ti_listener = {
	.enter = uitk_ti_enter,
	.leave = uitk_ti_leave,
	.preedit_string = uitk_ti_preedit,
	.commit_string = uitk_ti_commit,
	.delete_surrounding_text = uitk_ti_delete,
	.done = uitk_ti_done,
};
static void ui_wl_ti_listen(struct zwp_text_input_v3 *t, uintptr_t id) {
	zwp_text_input_v3_add_listener(t, &uitk_ti_listener, (void*)id);
}

static void uitk_out_geom(void *data, struct wl_output *o, int32_t x, int32_t y, int32_t pw, int32_t ph, int32_t sub, const char *make, const char *model, int32_t transform) {
	(void)data; (void)o; (void)x; (void)y; (void)pw; (void)ph; (void)sub; (void)make; (void)model; (void)transform;
}
static void uitk_out_mode(void *data, struct wl_output *o, uint32_t flags, int32_t w, int32_t h, int32_t refresh) {
	(void)data; (void)o; (void)flags; (void)w; (void)h; (void)refresh;
}
static void uitk_out_done(void *data, struct wl_output *o) { (void)data; (void)o; }
static void uitk_out_scale(void *data, struct wl_output *o, int32_t factor) {
	(void)o;
	uitkWlOutputScale((uintptr_t)data, factor);
}
static void uitk_out_name(void *data, struct wl_output *o, const char *n) { (void)data; (void)o; (void)n; }
static void uitk_out_desc(void *data, struct wl_output *o, const char *n) { (void)data; (void)o; (void)n; }
static const struct wl_output_listener uitk_out_listener = {
	.geometry = uitk_out_geom,
	.mode = uitk_out_mode,
	.done = uitk_out_done,
	.scale = uitk_out_scale,
	.name = uitk_out_name,
	.description = uitk_out_desc,
};
static void ui_wl_out_listen(struct wl_output *o, uintptr_t id) {
	wl_output_add_listener(o, &uitk_out_listener, (void*)id);
}
static void ui_wl_out_destroy(struct wl_output *o) { if (o) wl_output_destroy(o); }

static void ui_wl_set_buf_scale(struct wl_surface *s, int32_t scale) {
	if (s && wl_surface_get_version(s) >= 3) wl_surface_set_buffer_scale(s, scale);
}
static void ui_wl_set_max(struct xdg_toplevel *t) { if (t) xdg_toplevel_set_maximized(t); }
static void ui_wl_unset_max(struct xdg_toplevel *t) { if (t) xdg_toplevel_unset_maximized(t); }
static void ui_wl_set_full(struct xdg_toplevel *t) { if (t) xdg_toplevel_set_fullscreen(t, NULL); }
static void ui_wl_unset_full(struct xdg_toplevel *t) { if (t) xdg_toplevel_unset_fullscreen(t); }

static struct zxdg_toplevel_decoration_v1 *ui_wl_deco(struct zxdg_decoration_manager_v1 *m, struct xdg_toplevel *t) {
	return zxdg_decoration_manager_v1_get_toplevel_decoration(m, t);
}
static void ui_wl_deco_ssd(struct zxdg_toplevel_decoration_v1 *d) {
	if (d) zxdg_toplevel_decoration_v1_set_mode(d, ZXDG_TOPLEVEL_DECORATION_V1_MODE_SERVER_SIDE);
}
static void ui_wl_deco_destroy(struct zxdg_toplevel_decoration_v1 *d) {
	if (d) zxdg_toplevel_decoration_v1_destroy(d);
}
static void ui_wl_deco_man_destroy(struct zxdg_decoration_manager_v1 *m) {
	if (m) zxdg_decoration_manager_v1_destroy(m);
}

static struct wp_fractional_scale_v1 *ui_wl_frac(struct wp_fractional_scale_manager_v1 *m, struct wl_surface *s) {
	return wp_fractional_scale_manager_v1_get_fractional_scale(m, s);
}
static void uitk_frac_pref(void *data, struct wp_fractional_scale_v1 *f, uint32_t scale) {
	(void)f;
	uitkWlFracScale((uintptr_t)data, scale);
}
static const struct wp_fractional_scale_v1_listener uitk_frac_listener = { .preferred_scale = uitk_frac_pref };
static void ui_wl_frac_listen(struct wp_fractional_scale_v1 *f, uintptr_t sid) {
	wp_fractional_scale_v1_add_listener(f, &uitk_frac_listener, (void*)sid);
}
static void ui_wl_frac_destroy(struct wp_fractional_scale_v1 *f) { if (f) wp_fractional_scale_v1_destroy(f); }
static void ui_wl_frac_man_destroy(struct wp_fractional_scale_manager_v1 *m) {
	if (m) wp_fractional_scale_manager_v1_destroy(m);
}

static struct wp_viewport *ui_wl_viewport(struct wp_viewporter *v, struct wl_surface *s) {
	return wp_viewporter_get_viewport(v, s);
}
static void ui_wl_viewport_dest(struct wp_viewport *v, int w, int h) {
	if (v) wp_viewport_set_destination(v, w, h);
}
static void ui_wl_viewport_destroy(struct wp_viewport *v) { if (v) wp_viewport_destroy(v); }
static void ui_wl_viewporter_destroy(struct wp_viewporter *v) { if (v) wp_viewporter_destroy(v); }

static int ui_wl_pipe(int fds[2]) { return pipe(fds); }
static ssize_t ui_wl_write(int fd, const char *p, size_t n) { return write(fd, p, n); }
static ssize_t ui_wl_read(int fd, char *p, size_t n) { return read(fd, p, n); }
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// WaylandBackend presents through paintengine2d. When UITK_PAINT=auto|gpu
// and EGL init works, each window is a wl_egl_window + eglSwapBuffers.
// Otherwise it keeps the v0.4.1 opaque wl_shm XRGB8888 path. linux-dmabuf
// is opt-in (UITK_WAYLAND_PRESENT=dmabuf) on the CPU present path.
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
		conn:     c,
		title:    title,
		logicalW: w,
		logicalH: h,
		wantW:    w,
		wantH:    h,
		bufScale: 1,
	}
	bw, bh := w, h
	if sc := int(c.outScale + 0.1); sc > 1 {
		s.bufScale = sc
		bw, bh = w*sc, h*sc
	}
	s.img = paintengine2d.NewImage(bw, bh)
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
	if c.decoMan != nil && s.top != nil {
		s.deco = C.ui_wl_deco(c.decoMan, s.top)
		if s.deco != nil {
			C.ui_wl_deco_ssd(s.deco)
		}
	}
	if c.fracMan != nil && s.surf != nil {
		s.fracObj = C.ui_wl_frac(c.fracMan, s.surf)
		if s.fracObj != nil {
			C.ui_wl_frac_listen(s.fracObj, C.uintptr_t(s.id))
		}
	}
	if c.viewporter != nil && s.surf != nil {
		s.viewport = C.ui_wl_viewport(c.viewporter, s.surf)
	}
	// Do not bind drm-syncobj / explicit-sync on the surface here.
	// Those protocols require acquire fences on every later commit; an
	// unsignaled fence (CPU write not in the implicit reservation)
	// leaves the compositor waiting and the window fully transparent.
	C.ui_wl_commit(s.surf)
	wlMu.Unlock()
	C.ui_wl_roundtrip(c.dpy)
	if !s.configured {
		C.ui_wl_roundtrip(c.dpy)
	}
	s.tryBindGPU()
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
	composeTbl *C.struct_xkb_compose_table
	compose    *C.struct_xkb_compose_state
	refs       int
	ptrSurf    int
	keySurf    int
	px, py     float32
	mods       Modifiers
	serial     uint32
	ptrSerial  uint32
	cursor     Cursor
	curSurf    *C.struct_wl_surface
	curSlot    [4]wlCursorBuf
	outScale   float32

	dataMan     *C.struct_wl_data_device_manager
	dataDev     *C.struct_wl_data_device
	dataSrc     *C.struct_wl_data_source
	clipOffer   *C.struct_wl_data_offer
	clipMime    string
	clipText    string
	pendingOff  *C.struct_wl_data_offer
	pendingMime string

	primMan      *C.struct_zwp_primary_selection_device_manager_v1
	primDev      *C.struct_zwp_primary_selection_device_v1
	primSrc      *C.struct_zwp_primary_selection_source_v1
	primOffer    *C.struct_zwp_primary_selection_offer_v1
	primMime     string
	primText     string
	pendPrim     *C.struct_zwp_primary_selection_offer_v1
	pendPrimMime string

	textMan     *C.struct_zwp_text_input_manager_v3
	textIn      *C.struct_zwp_text_input_v3
	textActive  bool
	tiWanted    bool // app focused an IMETarget; enable only then
	tiSurf      int
	imePre      string
	imeCommit   string
	imeDelB     int
	imeDelA     int
	imeBegin    int
	imeEnd      int
	lastXkbText string
	lastIMEText string

	decoMan    *C.struct_zxdg_decoration_manager_v1
	fracMan    *C.struct_wp_fractional_scale_manager_v1
	viewporter *C.struct_wp_viewporter
	output     *C.struct_wl_output

	dmabuf     unsafe.Pointer // *zwp_linux_dmabuf_v1
	dmabufFB   unsafe.Pointer // *zwp_linux_dmabuf_feedback_v1
	dmabufVer  int
	dmaPairs   []dmaFmtMod
	dmaTable   []dmaFmtMod
	useDmabuf  bool
	dmaFmt     uint32
	dmaMod     uint64
	dmaSwizzle bool
	dmaAlloc   string

	explicitSync unsafe.Pointer // *zwp_linux_explicit_synchronization_v1
	drmSyncobj   unsafe.Pointer // *wp_linux_drm_syncobj_manager_v1

	repeatRate  int
	repeatDelay int
	repeatKey   uint32
	repeatNext  time.Time
	heldKey     uint32
	heldDown    bool
	clipKeep    bool
}

type wlSlot struct {
	buf    *C.struct_wl_buffer
	mem    unsafe.Pointer
	size   int
	fd     int
	w, h   int
	stride int
	busy   bool
	dma    unsafe.Pointer // *ui_dmabuf_bo
}

type wlSurface struct {
	id         int
	conn       *wlConn
	surf       *C.struct_wl_surface
	xdg        *C.struct_xdg_surface
	top        *C.struct_xdg_toplevel
	title      string
	img        *paintengine2d.Image
	slots      [4]wlSlot
	wantW      int
	wantH      int
	logicalW   int
	logicalH   int
	bufScale   int
	frac       float32
	opaqueW    int
	opaqueH    int
	scaleSet   int
	configured bool
	closed     bool
	queue      []Event
	viewport   *C.struct_wp_viewport
	fracObj    *C.struct_wp_fractional_scale_v1
	deco       *C.struct_zxdg_toplevel_decoration_v1
	surfSync   unsafe.Pointer // *zwp_linux_surface_synchronization_v1
	timeline   unsafe.Pointer // *ui_drm_timeline
	gpu        *paintengine2d.GPUDevice
	eglWin     unsafe.Pointer // *wl_egl_window
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
	for i := range c.curSlot {
		c.curSlot[i].fd = -1
	}
	wlNext++
	c.id = wlNext
	wlConns[c.id] = c
	wlc = c
	c.reg = C.ui_wl_registry(dpy)
	C.ui_wl_reg_listen(c.reg, C.uintptr_t(c.id))
	wlMu.Unlock()
	C.ui_wl_roundtrip(dpy)
	C.ui_wl_roundtrip(dpy)
	if c.dmabuf != nil && c.dmabufVer >= 4 && waylandWantDmabuf() {
		wlMu.Lock()
		c.requestDmabufFeedback()
		wlMu.Unlock()
		C.ui_wl_roundtrip(dpy)
	}
	wlMu.Lock()
	c.chooseWaylandPresent()
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
	if c.refs <= 0 && !c.clipKeep {
		c.closeLocked()
	}
}

func (c *wlConn) closeLocked() {
	if c.keyboard != nil {
		C.ui_wl_kb_destroy(c.keyboard)
		c.keyboard = nil
	}
	c.destroyCursorsLocked()
	if c.pointer != nil {
		C.ui_wl_ptr_destroy(c.pointer)
		c.pointer = nil
	}
	if c.textIn != nil {
		C.ui_wl_ti_destroy(c.textIn)
		c.textIn = nil
	}
	if c.textMan != nil {
		C.ui_wl_ti_man_destroy(c.textMan)
		c.textMan = nil
	}
	if c.dataSrc != nil {
		C.ui_wl_data_source_destroy(c.dataSrc)
		c.dataSrc = nil
	}
	if c.clipOffer != nil {
		C.ui_wl_data_offer_destroy(c.clipOffer)
		c.clipOffer = nil
	}
	if c.dataDev != nil {
		C.ui_wl_data_device_destroy(c.dataDev)
		c.dataDev = nil
	}
	if c.dataMan != nil {
		C.ui_wl_data_man_destroy(c.dataMan)
		c.dataMan = nil
	}
	if c.primSrc != nil {
		C.ui_wl_prim_source_destroy(c.primSrc)
		c.primSrc = nil
	}
	if c.primOffer != nil {
		C.ui_wl_prim_offer_destroy(c.primOffer)
		c.primOffer = nil
	}
	if c.primDev != nil {
		C.ui_wl_prim_dev_destroy(c.primDev)
		c.primDev = nil
	}
	if c.primMan != nil {
		C.ui_wl_prim_man_destroy(c.primMan)
		c.primMan = nil
	}
	c.destroyDmabufLocked()
	if c.explicitSync != nil {
		C.ui_wl_explicit_destroy((*C.struct_zwp_linux_explicit_synchronization_v1)(c.explicitSync))
		c.explicitSync = nil
	}
	if c.drmSyncobj != nil {
		C.ui_wl_drm_syncobj_destroy((*C.struct_wp_linux_drm_syncobj_manager_v1)(c.drmSyncobj))
		c.drmSyncobj = nil
	}
	if c.decoMan != nil {
		C.ui_wl_deco_man_destroy(c.decoMan)
		c.decoMan = nil
	}
	if c.fracMan != nil {
		C.ui_wl_frac_man_destroy(c.fracMan)
		c.fracMan = nil
	}
	if c.viewporter != nil {
		C.ui_wl_viewporter_destroy(c.viewporter)
		c.viewporter = nil
	}
	if c.output != nil {
		C.ui_wl_out_destroy(c.output)
		c.output = nil
	}
	if c.seat != nil {
		C.ui_wl_seat_destroy(c.seat)
		c.seat = nil
	}
	C.ui_xkb_compose_state_unref(c.compose)
	C.ui_xkb_compose_table_unref(c.composeTbl)
	C.ui_xkb_state_unref(c.xkbState)
	C.ui_xkb_map_unref(c.xkbMap)
	C.ui_xkb_ctx_unref(c.xkbCtx)
	c.xkbState, c.xkbMap, c.xkbCtx = nil, nil, nil
	c.compose, c.composeTbl = nil, nil
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

func (s *wlSurface) Title() string { return s.title }
func (s *wlSurface) Size() (w, h int) {
	if s.gpu != nil {
		return s.gpu.Size()
	}
	if s.img != nil {
		return s.img.Width, s.img.Height
	}
	return s.bufferWH()
}
func (s *wlSurface) Buffer() *paintengine2d.Image {
	if s.gpu != nil {
		if img := s.gpu.Image(); img != nil {
			return img
		}
	}
	return s.img
}
func (s *wlSurface) Closed() bool   { return s.closed }
func (s *wlSurface) Scale() float32 { return s.deviceScale() }

func (s *wlSurface) deviceScale() float32 {
	if s.frac > 1 {
		return s.frac
	}
	if s.bufScale > 1 {
		return float32(s.bufScale)
	}
	if s.conn != nil && s.conn.outScale > 1 {
		return s.conn.outScale
	}
	return 1
}

func (s *wlSurface) toDevice(x, y float32) (float32, float32) {
	sc := s.deviceScale()
	return x * sc, y * sc
}

func (s *wlSurface) SetFullscreen(on bool) {
	if s.top == nil {
		return
	}
	if on {
		C.ui_wl_set_full(s.top)
	} else {
		C.ui_wl_unset_full(s.top)
	}
}

func (s *wlSurface) SetMaximized(on bool) {
	if s.top == nil {
		return
	}
	if on {
		C.ui_wl_set_max(s.top)
	} else {
		C.ui_wl_unset_max(s.top)
	}
}

func (s *wlSurface) SetCursor(cur Cursor) {
	if s == nil || s.conn == nil {
		return
	}
	wlMu.Lock()
	s.conn.cursor = cur
	s.conn.applyCursorLocked()
	wlMu.Unlock()
}

type wlCursorBuf struct {
	buf    *C.struct_wl_buffer
	mem    unsafe.Pointer
	fd     int
	size   int
	hx, hy int
	ready  bool
}

const wlCursorSize = 24

func (c *wlConn) destroyCursorsLocked() {
	if c.curSurf != nil {
		C.ui_wl_surface_destroy(c.curSurf)
		c.curSurf = nil
	}
	for i := range c.curSlot {
		s := &c.curSlot[i]
		if s.buf != nil {
			C.ui_wl_buf_destroy(s.buf)
			s.buf = nil
		}
		if s.mem != nil {
			C.ui_wl_munmap(s.mem, C.size_t(s.size))
			s.mem = nil
		}
		if s.ready && s.fd >= 0 {
			C.ui_wl_close_fd(C.int(s.fd))
		}
		s.fd = -1
		s.ready = false
	}
}

func (c *wlConn) ensureCursorBuf(cur Cursor) *wlCursorBuf {
	idx := int(cur)
	if idx < 0 || idx >= len(c.curSlot) {
		idx = 0
	}
	slot := &c.curSlot[idx]
	if slot.ready && slot.buf != nil {
		return slot
	}
	const n = wlCursorSize
	stride := n * 4
	size := stride * n
	var mem unsafe.Pointer
	fd := int(C.ui_wl_memfd(C.size_t(size), &mem))
	if fd < 0 || mem == nil {
		return nil
	}
	pix := unsafe.Slice((*byte)(mem), size)
	hx, hy := drawCursorARGB(pix, n, stride, cur)
	buf := C.ui_wl_argb_buffer(c.shm, C.int(fd), C.int(n), C.int(n), C.int(stride), C.size_t(size))
	if buf == nil {
		C.ui_wl_munmap(mem, C.size_t(size))
		C.ui_wl_close_fd(C.int(fd))
		return nil
	}
	slot.buf, slot.mem, slot.fd, slot.size = buf, mem, fd, size
	slot.hx, slot.hy, slot.ready = hx, hy, true
	return slot
}

func putARGB(pix []byte, stride, x, y int, r, g, b, a byte) {
	if x < 0 || y < 0 {
		return
	}
	off := y*stride + x*4
	if off < 0 || off+3 >= len(pix) {
		return
	}
	// WL_SHM_FORMAT_ARGB8888 little-endian: B, G, R, A
	pix[off+0] = b
	pix[off+1] = g
	pix[off+2] = r
	pix[off+3] = a
}

func drawCursorARGB(pix []byte, n, stride int, cur Cursor) (hx, hy int) {
	for i := range pix {
		pix[i] = 0
	}
	dot := func(x, y int, on bool) {
		if on {
			putARGB(pix, stride, x, y, 0, 0, 0, 255)
		} else {
			putARGB(pix, stride, x, y, 255, 255, 255, 255)
		}
	}
	switch cur {
	case CursorColResize:
		cy := n / 2
		for x := 3; x < n-3; x++ {
			dot(x, cy, true)
			dot(x, cy-1, false)
			dot(x, cy+1, false)
		}
		for i := 0; i < 5; i++ {
			dot(3+i, cy-i, true)
			dot(3+i, cy+i, true)
			dot(n-4-i, cy-i, true)
			dot(n-4-i, cy+i, true)
		}
		return n / 2, cy
	case CursorRowResize:
		cx := n / 2
		for y := 3; y < n-3; y++ {
			dot(cx, y, true)
			dot(cx-1, y, false)
			dot(cx+1, y, false)
		}
		for i := 0; i < 5; i++ {
			dot(cx-i, 3+i, true)
			dot(cx+i, 3+i, true)
			dot(cx-i, n-4-i, true)
			dot(cx+i, n-4-i, true)
		}
		return cx, n / 2
	case CursorText:
		cx := n / 2
		for y := 4; y < n-4; y++ {
			dot(cx, y, true)
			dot(cx-1, y, false)
			dot(cx+1, y, false)
		}
		for x := cx - 3; x <= cx+3; x++ {
			dot(x, 4, true)
			dot(x, n-5, true)
		}
		return cx, n / 2
	default:
		// left-pointing arrow
		for y := 1; y < 16; y++ {
			w := y
			if w > 10 {
				w = 10
			}
			if y > 12 {
				w = 16 - y
			}
			for x := 1; x <= w; x++ {
				dot(x, y, x == 1 || x == w || y == 1)
			}
		}
		return 1, 1
	}
}

func (c *wlConn) applyCursorLocked() {
	if c == nil || c.pointer == nil || c.compositor == nil || c.shm == nil || c.dpy == nil {
		return
	}
	if c.curSurf == nil {
		c.curSurf = C.ui_wl_surface(c.compositor)
	}
	slot := c.ensureCursorBuf(c.cursor)
	if slot == nil && c.cursor != CursorDefault {
		slot = c.ensureCursorBuf(CursorDefault)
	}
	if slot == nil || slot.buf == nil || c.curSurf == nil {
		return
	}
	C.ui_wl_attach(c.curSurf, slot.buf)
	C.ui_wl_damage(c.curSurf, C.int(0), C.int(0), C.int(wlCursorSize), C.int(wlCursorSize))
	C.ui_wl_commit(c.curSurf)
	C.ui_wl_set_cursor(c.pointer, C.uint32_t(c.ptrSerial), c.curSurf, C.int32_t(slot.hx), C.int32_t(slot.hy))
	C.ui_wl_flush(c.dpy)
}

func (s *wlSurface) SetIMEEnabled(on bool) {
	if s.conn == nil || s.conn.textIn == nil {
		return
	}
	if s.conn.tiWanted == on {
		if on && !s.conn.textActive {
			wlEnableTextInput(s.conn, s)
		}
		return
	}
	s.conn.tiWanted = on
	if on {
		wlEnableTextInput(s.conn, s)
		return
	}
	wlDisableTextInput(s.conn)
}

func (s *wlSurface) SetIMECursor(x, y, w, h int) {
	if s.conn == nil || s.conn.textIn == nil || !s.conn.textActive {
		return
	}
	sc := s.deviceScale()
	if sc < 1 {
		sc = 1
	}
	// text-input cursor rect is in surface-local (logical) units
	C.ui_wl_ti_cursor(s.conn.textIn, C.int(float32(x)/sc), C.int(float32(y)/sc), C.int(float32(w)/sc+0.5), C.int(float32(h)/sc+0.5))
	C.ui_wl_ti_commit(s.conn.textIn)
}

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
	w, h = fitLogicalSize(w, h, s.logicalW, s.logicalH, s.deviceScale())
	if s.img != nil && s.img.Width == w && s.img.Height == h {
		// Caller passed current buffer pixels; keep logical, just sync.
		if bw, bh := s.bufferWH(); bw == w && bh == h {
			return nil
		}
	}
	s.logicalW, s.logicalH = w, h
	s.wantW, s.wantH = w, h
	bw, bh := s.bufferWH()
	if s.img != nil && s.img.Width == bw && s.img.Height == bh {
		return nil
	}
	s.img = paintengine2d.NewImage(bw, bh)
	if s.gpu != nil {
		s.resizeGPU(bw, bh)
	}
	return nil
}

func (s *wlSurface) bufferWH() (int, int) {
	lw, lh := s.logicalW, s.logicalH
	if lw < 1 {
		lw = s.wantW
	}
	if lh < 1 {
		lh = s.wantH
	}
	if lw < 1 {
		lw = 1
	}
	if lh < 1 {
		lh = 1
	}
	sc := s.deviceScale()
	w := int(math.Ceil(float64(float32(lw) * sc)))
	h := int(math.Ceil(float64(float32(lh) * sc)))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func (s *wlSurface) Present(dirty []paintengine2d.Rect) error {
	if s.closed || s.surf == nil || s.conn == nil || s.conn.dpy == nil {
		return nil
	}
	if !s.configured {
		return nil
	}
	if s.wantW > 0 {
		s.logicalW = s.wantW
	}
	if s.wantH > 0 {
		s.logicalH = s.wantH
	}
	bw, bh := s.bufferWH()
	if s.img.Width != bw || s.img.Height != bh {
		s.img = paintengine2d.NewImage(bw, bh)
		s.queue = append(s.queue, Event{Kind: EventResize, Width: s.logicalW, Height: s.logicalH})
		if s.gpu != nil {
			s.resizeGPU(bw, bh)
		}
	}
	if s.frac > 1 && s.viewport != nil {
		if s.scaleSet != 1 {
			C.ui_wl_set_buf_scale(s.surf, 1)
			s.scaleSet = 1
		}
		C.ui_wl_viewport_dest(s.viewport, C.int(s.logicalW), C.int(s.logicalH))
	} else if sc := int(s.deviceScale() + 0.1); sc > 1 {
		if s.scaleSet != sc {
			C.ui_wl_set_buf_scale(s.surf, C.int32_t(sc))
			s.scaleSet = sc
		}
	}
	if s.conn.compositor != nil && s.logicalW > 0 && s.logicalH > 0 &&
		(s.opaqueW != s.logicalW || s.opaqueH != s.logicalH) {
		C.ui_wl_opaque(s.conn.compositor, s.surf, C.int(s.logicalW), C.int(s.logicalH))
		s.opaqueW, s.opaqueH = s.logicalW, s.logicalH
	}
	if s.gpu != nil {
		if err := s.presentGPU(dirty); err == nil {
			C.ui_wl_flush(s.conn.dpy)
			return nil
		}
		// EGL failed this frame — keep the last GPU frame on the CPU
		// pixmap and fall back to v0.4.1 opaque shm.
		s.abandonGPU()
	}
	if len(dirty) == 0 {
		dirty = []paintengine2d.Rect{paintengine2d.XYWH(0, 0, float32(s.img.Width), float32(s.img.Height))}
	}
	slot := s.pickSlot()
	if err := s.ensureSlot(slot, s.img.Width, s.img.Height); err != nil {
		if s.conn.useDmabuf {
			s.conn.useDmabuf = false
			s.destroySlot(slot)
			if err2 := s.ensureSlot(slot, s.img.Width, s.img.Height); err2 != nil {
				return err
			}
		} else {
			return err
		}
	}
	s.blitDirty(slot, dirty)
	if s.dmabufUploadBlank(slot) {
		s.finishDMAWrite(slot)
		s.conn.useDmabuf = false
		s.destroySlot(slot)
		if err := s.ensureSlot(slot, s.img.Width, s.img.Height); err != nil {
			return err
		}
		s.blitDirty(slot, dirty)
	}
	s.finishDMAWrite(slot)
	C.ui_wl_attach(s.surf, s.slots[slot].buf)
	C.ui_wl_commit(s.surf)
	s.slots[slot].busy = true
	C.ui_wl_flush(s.conn.dpy)
	return nil
}

func (s *wlSurface) blitDirty(slot int, dirty []paintengine2d.Rect) {
	s.beginDMAWrite(slot)
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
}

func (s *wlSurface) beginDMAWrite(slot int) {
	if slot < 0 || slot >= len(s.slots) {
		return
	}
	fd := s.slots[slot].fd
	if s.slots[slot].dma != nil && fd >= 0 {
		C.ui_dmabuf_cpu_begin(C.int(fd))
	}
}

func (s *wlSurface) finishDMAWrite(slot int) {
	if slot < 0 || slot >= len(s.slots) {
		return
	}
	fd := s.slots[slot].fd
	if s.slots[slot].dma != nil && fd >= 0 {
		C.ui_dmabuf_cpu_end(C.int(fd))
	}
}

func (s *wlSurface) dmabufUploadBlank(slot int) bool {
	if s.conn == nil || !s.conn.useDmabuf || slot < 0 || slot >= len(s.slots) {
		return false
	}
	sl := s.slots[slot]
	if sl.dma == nil || sl.mem == nil || !pixmapHasOpaque(s.img) {
		return false
	}
	dst := unsafe.Slice((*byte)(sl.mem), sl.size)
	n := sl.stride
	if n < 64 {
		n = sl.size
	}
	return destAlphaAllZero(dst, n)
}

func (s *wlSurface) bindExplicitSync() {
	if s == nil || s.conn == nil || s.surf == nil {
		return
	}
	if s.conn.drmSyncobj != nil {
		fd := int(C.ui_dmabuf_drm_fd())
		if fd >= 0 {
			tl := C.ui_drm_timeline_setup(
				(*C.struct_wp_linux_drm_syncobj_manager_v1)(s.conn.drmSyncobj),
				s.surf, C.int(fd))
			if tl != nil {
				s.timeline = unsafe.Pointer(tl)
				return
			}
		}
	}
	if s.conn.explicitSync != nil {
		sync := C.ui_wl_explicit_get_sync(
			(*C.struct_zwp_linux_explicit_synchronization_v1)(s.conn.explicitSync),
			s.surf)
		if sync != nil {
			s.surfSync = unsafe.Pointer(sync)
		}
	}
}

func (s *wlSurface) markAcquire(slot, dmaFD int) {
	if s.timeline != nil && s.slots[slot].dma != nil {
		C.ui_drm_timeline_set_points((*C.struct_ui_drm_timeline)(s.timeline), C.int(slot))
		return
	}
	if s.surfSync == nil || s.slots[slot].dma == nil || dmaFD < 0 {
		return
	}
	fence := int(C.ui_dmabuf_export_sync_file(C.int(dmaFD)))
	if fence >= 0 {
		C.ui_wl_explicit_set_acquire((*C.struct_zwp_linux_surface_synchronization_v1)(s.surfSync), C.int(fence))
		C.ui_wl_close_fd(C.int(fence))
	}
	C.ui_wl_explicit_get_release((*C.struct_zwp_linux_surface_synchronization_v1)(s.surfSync), C.uintptr_t(s.id), C.int(slot))
}

func (s *wlSurface) pickSlot() int {
	if i, ok := s.freeSlot(); ok {
		return i
	}
	// All four slots are in flight. Wait briefly for a release —
	// not a 24× wl_display_roundtrip storm (~55ms/frame at 1000×760).
	if s.conn != nil && s.conn.dpy != nil {
		C.ui_wl_wait(s.conn.dpy, 8)
	}
	if i, ok := s.freeSlot(); ok {
		return i
	}
	if s.conn != nil && s.conn.dpy != nil {
		C.ui_wl_wait(s.conn.dpy, 8)
	}
	if i, ok := s.freeSlot(); ok {
		return i
	}
	return 0
}

func (s *wlSurface) freeSlot() (int, bool) {
	for i := range s.slots {
		if !s.slots[i].busy {
			return i, true
		}
	}
	return 0, false
}

func (s *wlSurface) ensureSlot(i, w, h int) error {
	sl := &s.slots[i]
	if sl.buf != nil && sl.w == w && sl.h == h {
		return nil
	}
	s.destroySlot(i)
	if s.conn != nil && s.conn.useDmabuf {
		if err := s.ensureDmabufSlot(i, w, h); err == nil {
			if s.slots[i].buf != nil {
				C.ui_wl_buf_listen(s.slots[i].buf, C.uintptr_t(s.id), C.int(i))
			}
			return nil
		}
		s.conn.useDmabuf = false
	}
	return s.ensureShmSlot(i, w, h)
}

func (s *wlSurface) ensureShmSlot(i, w, h int) error {
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
	s.slots[i] = wlSlot{buf: buf, mem: mem, size: size, fd: -1, w: w, h: h, stride: stride}
	return nil
}

func (s *wlSurface) destroySlot(i int) {
	sl := &s.slots[i]
	if sl.buf != nil {
		C.ui_wl_buf_destroy(sl.buf)
		sl.buf = nil
	}
	if sl.dma != nil {
		s.freeDmabufSlot(i)
		*sl = wlSlot{}
		return
	}
	if sl.mem != nil {
		C.ui_wl_munmap(sl.mem, C.size_t(sl.size))
		sl.mem = nil
	}
	*sl = wlSlot{}
}

func (s *wlSurface) slotBGRA(slot int) []byte {
	if slot < 0 || slot >= len(s.slots) {
		return nil
	}
	sl := s.slots[slot]
	if sl.mem == nil || sl.size < 4 {
		return nil
	}
	return unsafe.Slice((*byte)(sl.mem), sl.size)
}

func (s *wlSurface) copyRect(slot int, r paintengine2d.Rect) {
	sl := s.slots[slot]
	if sl.mem == nil {
		return
	}
	dst := unsafe.Slice((*byte)(sl.mem), sl.size)
	stride := sl.stride
	if stride < sl.w*4 {
		stride = sl.w * 4
	}
	swizzle := true
	if sl.dma != nil && s.conn != nil {
		swizzle = s.conn.dmaSwizzle
	}
	copyImageRect(dst, stride, s.img, r, swizzle, true)
}

func (s *wlSurface) Poll() []Event {
	if s.closed || s.conn == nil || s.conn.dpy == nil {
		return nil
	}
	C.ui_wl_pump(s.conn.dpy)
	wlMu.Lock()
	s.conn.flushRepeatLocked()
	ev := s.queue
	s.queue = nil
	wlMu.Unlock()
	return ev
}

func (s *wlSurface) Wait(timeout time.Duration) bool {
	if s == nil || s.closed || s.conn == nil || s.conn.dpy == nil {
		return false
	}
	n := C.ui_wl_wait(s.conn.dpy, C.int(waitMillis(timeout)))
	return n != 0
}

func (s *wlSurface) WakeAt() time.Time {
	if s == nil || s.conn == nil {
		return time.Time{}
	}
	wlMu.Lock()
	defer wlMu.Unlock()
	if !s.conn.heldDown || s.conn.repeatKey == 0 || s.conn.repeatRate <= 0 {
		return time.Time{}
	}
	return s.conn.repeatNext
}

func (c *wlConn) flushRepeatLocked() {
	if !c.heldDown || c.repeatKey == 0 || c.repeatRate <= 0 {
		return
	}
	s := wlSurfaces[c.keySurf]
	if s == nil {
		return
	}
	now := time.Now()
	if now.Before(c.repeatNext) {
		return
	}
	interval := time.Second / time.Duration(c.repeatRate)
	if interval < time.Millisecond {
		interval = 25 * time.Millisecond
	}
	for !c.repeatNext.After(now) {
		ks := uint64(C.ui_xkb_sym(c.xkbState, C.uint32_t(c.repeatKey)))
		s.push(Event{Kind: EventKeyDown, Key: KeyFromKeysym(ks), Mods: c.mods})
		c.pushXKBText(s, C.uint32_t(c.repeatKey), ks)
		c.repeatNext = c.repeatNext.Add(interval)
		if c.repeatNext.Before(now.Add(-250 * time.Millisecond)) {
			c.repeatNext = now.Add(interval)
			break
		}
	}
}

func (s *wlSurface) Close() error {
	if s.closed && s.surf == nil {
		return nil
	}
	s.closed = true
	s.closeGPU()
	wlMu.Lock()
	delete(wlSurfaces, s.id)
	if s.surf != nil {
		delete(wlByNative, uintptr(unsafe.Pointer(s.surf)))
	}
	wlMu.Unlock()
	for i := range s.slots {
		s.destroySlot(i)
	}
	if s.timeline != nil {
		C.ui_drm_timeline_destroy((*C.struct_ui_drm_timeline)(s.timeline))
		s.timeline = nil
	}
	if s.surfSync != nil {
		C.ui_wl_explicit_sync_destroy((*C.struct_zwp_linux_surface_synchronization_v1)(s.surfSync))
		s.surfSync = nil
	}
	if s.fracObj != nil {
		C.ui_wl_frac_destroy(s.fracObj)
		s.fracObj = nil
	}
	if s.viewport != nil {
		C.ui_wl_viewport_destroy(s.viewport)
		s.viewport = nil
	}
	if s.deco != nil {
		C.ui_wl_deco_destroy(s.deco)
		s.deco = nil
	}
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
			c.bindSeatExtras()
		}
	case "wl_data_device_manager":
		v := ver
		if v > 3 {
			v = 3
		}
		c.dataMan = (*C.struct_wl_data_device_manager)(C.ui_wl_bind(reg, name, C.ui_wl_data_man_iface(), v))
		c.bindSeatExtras()
	case "wl_output":
		v := ver
		if v > 4 {
			v = 4
		}
		if v < 2 {
			v = 2
		}
		c.output = (*C.struct_wl_output)(C.ui_wl_bind(reg, name, C.ui_wl_output_iface(), v))
		if c.output != nil {
			C.ui_wl_out_listen(c.output, id)
		}
	case "zwp_text_input_manager_v3":
		c.textMan = (*C.struct_zwp_text_input_manager_v3)(C.ui_wl_bind(reg, name, C.ui_wl_ti_man_iface(), 1))
		c.bindSeatExtras()
	case "zwp_primary_selection_device_manager_v1":
		c.primMan = (*C.struct_zwp_primary_selection_device_manager_v1)(C.ui_wl_bind(reg, name, C.ui_wl_prim_man_iface(), 1))
		c.bindSeatExtras()
	case "zxdg_decoration_manager_v1":
		c.decoMan = (*C.struct_zxdg_decoration_manager_v1)(C.ui_wl_bind(reg, name, C.ui_wl_deco_man_iface(), 1))
	case "wp_fractional_scale_manager_v1":
		c.fracMan = (*C.struct_wp_fractional_scale_manager_v1)(C.ui_wl_bind(reg, name, C.ui_wl_frac_man_iface(), 1))
	case "wp_viewporter":
		c.viewporter = (*C.struct_wp_viewporter)(C.ui_wl_bind(reg, name, C.ui_wl_viewporter_iface(), 1))
	case "zwp_linux_explicit_synchronization_v1":
		if !waylandWantDmabuf() {
			break
		}
		v := ver
		if v > 2 {
			v = 2
		}
		if v >= 1 {
			c.explicitSync = unsafe.Pointer(C.ui_wl_bind(reg, name, C.ui_wl_explicit_sync_iface(), v))
		}
	case "wp_linux_drm_syncobj_v1":
		if !waylandWantDmabuf() {
			break
		}
		v := ver
		if v > 1 {
			v = 1
		}
		if v >= 1 {
			c.drmSyncobj = unsafe.Pointer(C.ui_wl_bind(reg, name, C.ui_wl_drm_syncobj_iface(), v))
		}
	case "zwp_linux_dmabuf_v1":
		if !waylandWantDmabuf() {
			break
		}
		v := ver
		if v > 5 {
			v = 5
		}
		if v >= 1 {
			d := C.ui_wl_bind(reg, name, C.ui_wl_dmabuf_iface(), v)
			if d != nil {
				c.dmabuf = d
				c.dmabufVer = int(v)
				C.ui_wl_dmabuf_listen((*C.struct_zwp_linux_dmabuf_v1)(d), C.uintptr_t(c.id))
			}
		}
	}
}

func (c *wlConn) bindSeatExtras() {
	if c.seat == nil {
		return
	}
	if c.dataMan != nil && c.dataDev == nil {
		c.dataDev = C.ui_wl_data_device(c.dataMan, c.seat)
		if c.dataDev != nil {
			C.ui_wl_ddev_listen(c.dataDev, C.uintptr_t(c.id))
		}
	}
	if c.primMan != nil && c.primDev == nil {
		c.primDev = C.ui_wl_prim_device(c.primMan, c.seat)
		if c.primDev != nil {
			C.ui_wl_pdev_listen(c.primDev, C.uintptr_t(c.id))
		}
	}
	if c.textMan != nil && c.textIn == nil {
		c.textIn = C.ui_wl_text_input(c.textMan, c.seat)
		if c.textIn != nil {
			C.ui_wl_ti_listen(c.textIn, C.uintptr_t(c.id))
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
func uitkWlTopConfigure(sid C.uintptr_t, top *C.struct_xdg_toplevel, w, h C.int32_t, flags C.uint32_t) {
	_ = top
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	if w > 0 && h > 0 {
		s.wantW, s.wantH = int(w), int(h)
		s.logicalW, s.logicalH = int(w), int(h)
		bw, bh := s.bufferWH()
		if s.img != nil && (s.img.Width != bw || s.img.Height != bh) {
			s.push(Event{Kind: EventResize, Width: int(w), Height: int(h)})
		}
	}
	_ = flags
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
func uitkWlPtrEnter(id C.uintptr_t, serial C.uint32_t, surf *C.struct_wl_surface, x, y C.wl_fixed_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.serial = uint32(serial)
	c.ptrSerial = uint32(serial)
	c.px = float32(C.ui_wl_fixed(x))
	c.py = float32(C.ui_wl_fixed(y))
	if s := wlSurfNative(surf); s != nil {
		c.ptrSurf = s.id
		dx, dy := s.toDevice(c.px, c.py)
		s.push(Event{Kind: EventMouseMove, Pos: paintengine2d.Pt(dx, dy), Mods: c.mods})
	}
	wlMu.Lock()
	c.applyCursorLocked()
	wlMu.Unlock()
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
		dx, dy := s.toDevice(c.px, c.py)
		s.push(Event{Kind: EventMouseMove, Pos: paintengine2d.Pt(dx, dy), Mods: c.mods})
	}
}

//export uitkWlPtrButton
func uitkWlPtrButton(id C.uintptr_t, button, state, serial C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.serial = uint32(serial)
	s := wlSurfaces[c.ptrSurf]
	if s == nil {
		return
	}
	kind := EventMouseUp
	if state == 1 {
		kind = EventMouseDown
	}
	dx, dy := s.toDevice(c.px, c.py)
	s.push(Event{Kind: kind, Pos: paintengine2d.Pt(dx, dy), Button: wlButton(uint32(button)), Mods: c.mods})
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
	dx, dy := float32(0), float32(0)
	if s != nil {
		dx, dy = s.toDevice(c.px, c.py)
	}
	ev := Event{Kind: EventScroll, Pos: paintengine2d.Pt(dx, dy), Mods: c.mods}
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
	C.ui_xkb_compose_state_unref(c.compose)
	C.ui_xkb_compose_table_unref(c.composeTbl)
	c.composeTbl = C.ui_xkb_compose_table(c.xkbCtx)
	c.compose = C.ui_xkb_compose_state(c.composeTbl)
}

//export uitkWlKeyEnter
func uitkWlKeyEnter(id C.uintptr_t, surf *C.struct_wl_surface, serial C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.serial = uint32(serial)
	if s := wlSurfNative(surf); s != nil {
		c.keySurf = s.id
		s.push(Event{Kind: EventFocusIn})
		if c.tiWanted {
			wlEnableTextInput(c, s)
		}
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
		s.push(Event{Kind: EventIMECancel})
	}
	c.keySurf = 0
	c.heldDown = false
	wlDisableTextInput(c)
}

//export uitkWlKey
func uitkWlKey(id C.uintptr_t, key, state, serial C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.serial = uint32(serial)
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
		c.heldKey = uint32(key)
		c.heldDown = true
		if c.repeatDelay > 0 && c.repeatRate > 0 {
			c.repeatKey = uint32(key)
			c.repeatNext = time.Now().Add(time.Duration(c.repeatDelay) * time.Millisecond)
		}
	} else if c.heldKey == uint32(key) {
		c.heldDown = false
		c.repeatKey = 0
	}
	s.push(ev)
	if !pressed {
		return
	}
	c.pushXKBText(s, key, ks)
}

// pushXKBText emits EventText from xkb/compose unless IME preedit owns the key.
// textActive alone must not suppress text: compositors often enter
// zwp_text_input_v3 without ever sending commit or preedit.
func (c *wlConn) pushXKBText(s *wlSurface, key C.uint32_t, ks uint64) {
	if s == nil || !ShouldEmitXKBText(true, c.mods.Ctrl(), c.imePre != "") {
		return
	}
	var raw []byte
	if c.compose != nil {
		_ = C.ui_xkb_compose_feed(c.compose, C.uint32_t(ks))
		switch C.ui_xkb_compose_status(c.compose) {
		case C.XKB_COMPOSE_COMPOSING:
			return
		case C.XKB_COMPOSE_COMPOSED:
			var buf [64]C.char
			n := int(C.ui_xkb_compose_utf8(c.compose, &buf[0], 64))
			C.ui_xkb_compose_reset(c.compose)
			if n > 0 {
				raw = C.GoBytes(unsafe.Pointer(&buf[0]), C.int(n))
			}
		case C.XKB_COMPOSE_CANCELLED:
			C.ui_xkb_compose_reset(c.compose)
			return
		}
	}
	if raw == nil {
		var buf [64]C.char
		n := int(C.ui_xkb_utf8(c.xkbState, key, &buf[0], 64))
		if n > 0 {
			raw = C.GoBytes(unsafe.Pointer(&buf[0]), C.int(n))
		}
	}
	if len(raw) == 0 {
		return
	}
	emit, lastX, lastI := PairXKBText(string(raw), c.lastIMEText)
	c.lastXkbText, c.lastIMEText = lastX, lastI
	if emit == "" {
		return
	}
	for _, te := range textEvents([]byte(emit), c.mods) {
		s.push(te)
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

//export uitkWlExplicitRelease
func uitkWlExplicitRelease(sid C.uintptr_t, slot C.int) {
	uitkWlBufRelease(sid, slot)
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

func waylandLive() bool {
	wlMu.Lock()
	defer wlMu.Unlock()
	return wlc != nil && wlc.dpy != nil
}

func waylandDetectScale() float32 {
	wlMu.Lock()
	if wlc != nil && wlc.outScale > 0 {
		s := wlc.outScale
		wlMu.Unlock()
		return s
	}
	wlMu.Unlock()
	return 0
}

//export uitkWlKeyRepeat
func uitkWlKeyRepeat(id C.uintptr_t, rate, delay C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.repeatRate = int(rate)
	c.repeatDelay = int(delay)
}

//export uitkWlOutputScale
func uitkWlOutputScale(id C.uintptr_t, factor C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if factor > 0 {
		c.outScale = float32(factor)
	}
}

//export uitkWlFracScale
func uitkWlFracScale(sid C.uintptr_t, scale120 C.uint32_t) {
	s := wlSurfBy(sid)
	if s == nil || scale120 == 0 {
		return
	}
	s.frac = float32(scale120) / 120
	if s.frac < 1 {
		s.frac = 1
	}
}

//export uitkWlDataOffer
func uitkWlDataOffer(id C.uintptr_t, offer *C.struct_wl_data_offer) {
	c := wlConnBy(id)
	if c == nil || offer == nil {
		return
	}
	c.pendingOff = offer
	c.pendingMime = ""
	C.ui_wl_doffer_listen(offer, id)
}

//export uitkWlDataOfferMime
func uitkWlDataOfferMime(id C.uintptr_t, offer *C.struct_wl_data_offer, mime *C.char) {
	c := wlConnBy(id)
	if c == nil || mime == nil {
		return
	}
	m := C.GoString(mime)
	if preferMime(c.pendingMime, m) {
		c.pendingMime = m
	}
	_ = offer
}

//export uitkWlSelection
func uitkWlSelection(id C.uintptr_t, offer *C.struct_wl_data_offer) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if c.clipOffer != nil && c.clipOffer != offer {
		C.ui_wl_data_offer_destroy(c.clipOffer)
	}
	c.clipOffer = offer
	c.clipMime = c.pendingMime
	if offer == c.pendingOff {
		c.pendingOff = nil
	}
}

//export uitkWlDataSend
func uitkWlDataSend(id C.uintptr_t, fd C.int) {
	c := wlConnBy(id)
	if c == nil {
		C.ui_wl_close_fd(fd)
		return
	}
	b := []byte(c.clipText)
	if len(b) > 0 {
		C.ui_wl_write(fd, (*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b)))
	}
	C.ui_wl_close_fd(fd)
}

//export uitkWlDataCancelled
func uitkWlDataCancelled(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if c.dataSrc != nil {
		C.ui_wl_data_source_destroy(c.dataSrc)
		c.dataSrc = nil
	}
}

//export uitkWlPrimOffer
func uitkWlPrimOffer(id C.uintptr_t, offer *C.struct_zwp_primary_selection_offer_v1) {
	c := wlConnBy(id)
	if c == nil || offer == nil {
		return
	}
	c.pendPrim = offer
	c.pendPrimMime = ""
	C.ui_wl_poffer_listen(offer, id)
}

//export uitkWlPrimOfferMime
func uitkWlPrimOfferMime(id C.uintptr_t, offer *C.struct_zwp_primary_selection_offer_v1, mime *C.char) {
	c := wlConnBy(id)
	if c == nil || mime == nil {
		return
	}
	m := C.GoString(mime)
	if preferMime(c.pendPrimMime, m) {
		c.pendPrimMime = m
	}
	_ = offer
}

//export uitkWlPrimSelection
func uitkWlPrimSelection(id C.uintptr_t, offer *C.struct_zwp_primary_selection_offer_v1) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if c.primOffer != nil && c.primOffer != offer {
		C.ui_wl_prim_offer_destroy(c.primOffer)
	}
	c.primOffer = offer
	c.primMime = c.pendPrimMime
}

//export uitkWlPrimSend
func uitkWlPrimSend(id C.uintptr_t, fd C.int) {
	c := wlConnBy(id)
	if c == nil {
		C.ui_wl_close_fd(fd)
		return
	}
	b := []byte(c.primText)
	if len(b) == 0 {
		b = []byte(c.clipText)
	}
	if len(b) > 0 {
		C.ui_wl_write(fd, (*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b)))
	}
	C.ui_wl_close_fd(fd)
}

//export uitkWlTIEnter
func uitkWlTIEnter(id C.uintptr_t, surf *C.struct_wl_surface) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.textActive = true
	if s := wlSurfNative(surf); s != nil {
		c.tiSurf = s.id
	}
}

//export uitkWlTILeave
func uitkWlTILeave(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.textActive = false
	c.tiSurf = 0
	if s := wlSurfaces[c.keySurf]; s != nil {
		s.push(Event{Kind: EventIMECancel})
	}
}

//export uitkWlTIPreedit
func uitkWlTIPreedit(id C.uintptr_t, text *C.char, begin, end C.int32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if text != nil {
		c.imePre = C.GoString(text)
	} else {
		c.imePre = ""
	}
	c.imeBegin, c.imeEnd = int(begin), int(end)
}

//export uitkWlTICommit
func uitkWlTICommit(id C.uintptr_t, text *C.char) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	if text != nil {
		c.imeCommit = C.GoString(text)
	}
}

//export uitkWlTIDelete
func uitkWlTIDelete(id C.uintptr_t, before, after C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.imeDelB, c.imeDelA = int(before), int(after)
}

//export uitkWlTIDone
func uitkWlTIDone(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	s := wlSurfaces[c.keySurf]
	if s == nil {
		s = wlSurfaces[c.tiSurf]
	}
	if s == nil {
		return
	}
	if c.imePre != "" || (c.imeCommit == "" && c.imeDelB == 0 && c.imeDelA == 0) {
		caret := runeCountStr(c.imePre)
		if c.imeEnd >= 0 && c.imeEnd < caret {
			caret = c.imeEnd
		}
		s.push(Event{Kind: EventIMEPreedit, Text: c.imePre, IMECaret: caret})
	}
	emit, lastX, lastI := PairIMECommit(c.imeCommit, c.lastXkbText)
	c.lastXkbText, c.lastIMEText = lastX, lastI
	c.imeCommit = emit
	if c.imeDelB > 0 || c.imeDelA > 0 || c.imeCommit != "" {
		s.push(Event{Kind: EventIMECommit, Text: c.imeCommit, IMEDelBefore: c.imeDelB, IMEDelAfter: c.imeDelA})
	}
	c.imePre, c.imeCommit = "", ""
	c.imeDelB, c.imeDelA = 0, 0
}

func runeCountStr(s string) int { return len([]rune(s)) }

func preferMime(cur, next string) bool {
	if cur == "" {
		return next != ""
	}
	rank := func(m string) int {
		switch m {
		case "text/plain;charset=utf-8":
			return 3
		case "text/plain":
			return 2
		case "UTF8_STRING", "TEXT":
			return 1
		}
		return 0
	}
	return rank(next) > rank(cur)
}

func wlEnableTextInput(c *wlConn, s *wlSurface) {
	if c == nil || c.textIn == nil || s == nil {
		return
	}
	C.ui_wl_ti_enable(c.textIn)
	C.ui_wl_ti_cursor(c.textIn, 0, 0, 1, 1)
	C.ui_wl_ti_commit(c.textIn)
}

func wlDisableTextInput(c *wlConn) {
	if c == nil || c.textIn == nil {
		return
	}
	C.ui_wl_ti_disable(c.textIn)
	C.ui_wl_ti_commit(c.textIn)
	c.textActive = false
}

func wlClipSet(s string) bool {
	c, err := wlRetain()
	if err != nil || c == nil || c.dataDev == nil || c.dataMan == nil {
		if c != nil {
			c.release()
		}
		return false
	}
	wlMu.Lock()
	c.clipText = s
	c.primText = s
	c.clipKeep = true
	if c.dataSrc != nil {
		C.ui_wl_data_source_destroy(c.dataSrc)
		c.dataSrc = nil
	}
	src := C.ui_wl_data_source(c.dataMan)
	if src == nil {
		wlMu.Unlock()
		c.release()
		return false
	}
	c.dataSrc = src
	C.ui_wl_dsrc_listen(src, C.uintptr_t(c.id))
	utf := C.CString("text/plain;charset=utf-8")
	plain := C.CString("text/plain")
	C.ui_wl_data_offer_mime(src, utf)
	C.ui_wl_data_offer_mime(src, plain)
	C.free(unsafe.Pointer(utf))
	C.free(unsafe.Pointer(plain))
	C.ui_wl_set_selection(c.dataDev, src, C.uint32_t(c.serial))
	if c.primMan != nil && c.primDev != nil {
		if c.primSrc != nil {
			C.ui_wl_prim_source_destroy(c.primSrc)
			c.primSrc = nil
		}
		ps := C.ui_wl_prim_source(c.primMan)
		if ps != nil {
			c.primSrc = ps
			C.ui_wl_psrc_listen(ps, C.uintptr_t(c.id))
			utf2 := C.CString("text/plain;charset=utf-8")
			C.ui_wl_prim_offer_mime(ps, utf2)
			C.free(unsafe.Pointer(utf2))
			C.ui_wl_prim_set(c.primDev, ps, C.uint32_t(c.serial))
		}
	}
	C.ui_wl_flush(c.dpy)
	wlMu.Unlock()
	c.release()
	return true
}

func wlClipGet(primary bool) (string, bool) {
	c, err := wlRetain()
	if err != nil || c == nil {
		return "", false
	}
	defer c.release()
	if primary {
		if c.primText != "" {
			return c.primText, true
		}
		if c.primOffer == nil || c.primMime == "" {
			return "", false
		}
		return wlReadFD(c, func(fd int) {
			cm := C.CString(c.primMime)
			C.ui_wl_prim_receive(c.primOffer, cm, C.int(fd))
			C.free(unsafe.Pointer(cm))
		})
	}
	if c.clipText != "" {
		return c.clipText, true
	}
	if c.clipOffer == nil || c.clipMime == "" {
		return "", false
	}
	return wlReadFD(c, func(fd int) {
		cm := C.CString(c.clipMime)
		C.ui_wl_data_receive(c.clipOffer, cm, C.int(fd))
		C.free(unsafe.Pointer(cm))
	})
}

func wlReadFD(c *wlConn, request func(fd int)) (string, bool) {
	var fds [2]C.int
	if C.ui_wl_pipe(&fds[0]) != 0 {
		return "", false
	}
	request(int(fds[1]))
	C.ui_wl_close_fd(fds[1])
	C.ui_wl_flush(c.dpy)
	C.ui_wl_roundtrip(c.dpy)
	var out []byte
	var buf [4096]byte
	for {
		n := int(C.ui_wl_read(fds[0], (*C.char)(unsafe.Pointer(&buf[0])), 4096))
		if n <= 0 {
			break
		}
		out = append(out, buf[:n]...)
	}
	C.ui_wl_close_fd(fds[0])
	return string(out), true
}
