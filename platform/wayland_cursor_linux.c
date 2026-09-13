#include "wayland_cursor.h"

#include <stdlib.h>

const struct wl_interface *ui_wl_cursor_shape_iface(void)
{
	return &wp_cursor_shape_manager_v1_interface;
}

struct wp_cursor_shape_device_v1 *ui_wl_cursor_shape_pointer(struct wp_cursor_shape_manager_v1 *m, struct wl_pointer *p)
{
	if (!m || !p) {
		return NULL;
	}
	return wp_cursor_shape_manager_v1_get_pointer(m, p);
}

void ui_wl_cursor_shape_set(struct wp_cursor_shape_device_v1 *d, uint32_t serial, uint32_t shape)
{
	if (d) {
		wp_cursor_shape_device_v1_set_shape(d, serial, shape);
	}
}

void ui_wl_cursor_shape_dev_destroy(struct wp_cursor_shape_device_v1 *d)
{
	if (d) {
		wp_cursor_shape_device_v1_destroy(d);
	}
}

void ui_wl_cursor_shape_man_destroy(struct wp_cursor_shape_manager_v1 *m)
{
	if (m) {
		wp_cursor_shape_manager_v1_destroy(m);
	}
}

struct wl_cursor_theme *ui_wl_cursor_theme_load(struct wl_shm *shm)
{
	const char *name;
	const char *sz;
	int size = 24;

	if (!shm) {
		return NULL;
	}
	name = getenv("XCURSOR_THEME");
	if (name && !name[0]) {
		name = NULL;
	}
	sz = getenv("XCURSOR_SIZE");
	if (sz && sz[0]) {
		int n = atoi(sz);
		if (n >= 12 && n <= 512) {
			size = n;
		}
	}
	return wl_cursor_theme_load(name, size, shm);
}

void ui_wl_cursor_theme_destroy(struct wl_cursor_theme *t)
{
	if (t) {
		wl_cursor_theme_destroy(t);
	}
}

struct wl_cursor *ui_wl_cursor_get(struct wl_cursor_theme *t, const char *name)
{
	if (!t || !name || !name[0]) {
		return NULL;
	}
	return wl_cursor_theme_get_cursor(t, name);
}

struct wl_buffer *ui_wl_cursor_buffer(struct wl_cursor *c, int *w, int *h, int *hx, int *hy)
{
	struct wl_cursor_image *img;
	struct wl_buffer *buf;

	if (!c || c->image_count < 1 || !c->images || !c->images[0]) {
		return NULL;
	}
	img = c->images[0];
	buf = wl_cursor_image_get_buffer(img);
	if (!buf) {
		return NULL;
	}
	if (w) {
		*w = (int)img->width;
	}
	if (h) {
		*h = (int)img->height;
	}
	if (hx) {
		*hx = (int)img->hotspot_x;
	}
	if (hy) {
		*hy = (int)img->hotspot_y;
	}
	return buf;
}
