#ifndef UITK_WAYLAND_CURSOR_H
#define UITK_WAYLAND_CURSOR_H

#include <stdint.h>
#include <wayland-client.h>
#include <wayland-cursor.h>
#include "cursor-shape-v1-client-protocol.h"

#ifdef __cplusplus
extern "C" {
#endif

const struct wl_interface *ui_wl_cursor_shape_iface(void);
struct wp_cursor_shape_device_v1 *ui_wl_cursor_shape_pointer(struct wp_cursor_shape_manager_v1 *m, struct wl_pointer *p);
void ui_wl_cursor_shape_set(struct wp_cursor_shape_device_v1 *d, uint32_t serial, uint32_t shape);
void ui_wl_cursor_shape_dev_destroy(struct wp_cursor_shape_device_v1 *d);
void ui_wl_cursor_shape_man_destroy(struct wp_cursor_shape_manager_v1 *m);

struct wl_cursor_theme *ui_wl_cursor_theme_load(struct wl_shm *shm);
void ui_wl_cursor_theme_destroy(struct wl_cursor_theme *t);
struct wl_cursor *ui_wl_cursor_get(struct wl_cursor_theme *t, const char *name);
struct wl_buffer *ui_wl_cursor_buffer(struct wl_cursor *c, int *w, int *h, int *hx, int *hy);

#ifdef __cplusplus
}
#endif

#endif
