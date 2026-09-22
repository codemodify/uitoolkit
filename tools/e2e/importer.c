// importer — the rig's stand-in for a portal's file dialog: a window made the
// child of the window a portal call named, the way xdg-desktop-portal's
// backends do it. On Wayland ("wayland:<handle>") it imports the handle with
// xdg-foreign (zxdg_importer_v2.import_toplevel) and set_parent_of's its own
// toplevel; on X11 ("x11:<hex id>") it sets WM_TRANSIENT_FOR. Then it stays
// up for SECONDS so the compositor's placement can be read and shot.
//
// usage: importer PARENT TITLE SECONDS
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <time.h>
#include <unistd.h>
#include <wayland-client.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#include "xdg-shell-client-protocol.h"
#include "xdg-foreign-unstable-v2-client-protocol.h"

enum { W = 420, H = 260 };

static struct wl_compositor *comp;
static struct wl_shm *shm;
static struct xdg_wm_base *wm;
static struct zxdg_importer_v2 *importer;
static int configured;

static void reg_global(void *d, struct wl_registry *r, uint32_t name, const char *iface, uint32_t ver) {
	if (!strcmp(iface, wl_compositor_interface.name)) comp = wl_registry_bind(r, name, &wl_compositor_interface, 4);
	else if (!strcmp(iface, wl_shm_interface.name)) shm = wl_registry_bind(r, name, &wl_shm_interface, 1);
	else if (!strcmp(iface, xdg_wm_base_interface.name)) wm = wl_registry_bind(r, name, &xdg_wm_base_interface, 1);
	else if (!strcmp(iface, zxdg_importer_v2_interface.name)) importer = wl_registry_bind(r, name, &zxdg_importer_v2_interface, 1);
}
static void reg_remove(void *d, struct wl_registry *r, uint32_t name) {}
static const struct wl_registry_listener rl = {reg_global, reg_remove};
static void wm_ping(void *d, struct xdg_wm_base *b, uint32_t serial) { xdg_wm_base_pong(b, serial); }
static const struct xdg_wm_base_listener wml = {wm_ping};
static void xs_configure(void *d, struct xdg_surface *s, uint32_t serial) { xdg_surface_ack_configure(s, serial); configured = 1; }
static const struct xdg_surface_listener xsl = {xs_configure};
static void top_configure(void *d, struct xdg_toplevel *t, int32_t w, int32_t h, struct wl_array *st) {}
static void top_close(void *d, struct xdg_toplevel *t) { exit(0); }
static const struct xdg_toplevel_listener tl = {top_configure, top_close};
static void imported_destroyed(void *d, struct zxdg_imported_v2 *i) { fprintf(stderr, "importer: the parent is gone\n"); }
static const struct zxdg_imported_v2_listener il = {imported_destroyed};

static struct wl_buffer *buffer(void) {
	int fd = memfd_create("importer", MFD_CLOEXEC);
	if (fd < 0 || ftruncate(fd, W * H * 4) != 0) return NULL;
	uint32_t *px = mmap(NULL, W * H * 4, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	for (int y = 0; y < H; y++)
		for (int x = 0; x < W; x++)
			px[y * W + x] = y < 36 ? 0xff3d6fb5 : ((x / 20 + y / 20) % 2 ? 0xffe8e8e8 : 0xffd8d8d8);
	munmap(px, W * H * 4);
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, W * H * 4);
	struct wl_buffer *b = wl_shm_pool_create_buffer(pool, 0, W, H, W * 4, WL_SHM_FORMAT_XRGB8888);
	wl_shm_pool_destroy(pool);
	close(fd);
	return b;
}

static int wayland(const char *handle, const char *title, int secs) {
	struct wl_display *dpy = wl_display_connect(NULL);
	if (!dpy) { fprintf(stderr, "importer: no wayland display\n"); return 1; }
	struct wl_registry *reg = wl_display_get_registry(dpy);
	wl_registry_add_listener(reg, &rl, NULL);
	wl_display_roundtrip(dpy);
	if (!comp || !shm || !wm) { fprintf(stderr, "importer: missing globals\n"); return 1; }
	if (!importer) { fprintf(stderr, "importer: the compositor has no zxdg_importer_v2\n"); return 1; }
	xdg_wm_base_add_listener(wm, &wml, NULL);
	struct wl_surface *s = wl_compositor_create_surface(comp);
	struct xdg_surface *xs = xdg_wm_base_get_xdg_surface(wm, s);
	xdg_surface_add_listener(xs, &xsl, NULL);
	struct xdg_toplevel *top = xdg_surface_get_toplevel(xs);
	xdg_toplevel_add_listener(top, &tl, NULL);
	xdg_toplevel_set_title(top, title);
	xdg_toplevel_set_app_id(top, "uitk-e2e-fake-portal");
	struct zxdg_imported_v2 *imp = zxdg_importer_v2_import_toplevel(importer, handle);
	zxdg_imported_v2_add_listener(imp, &il, NULL);
	zxdg_imported_v2_set_parent_of(imp, s);
	wl_surface_commit(s);
	while (!configured && wl_display_dispatch(dpy) != -1) {}
	wl_surface_attach(s, buffer(), 0, 0);
	wl_surface_damage(s, 0, 0, W, H);
	wl_surface_commit(s);
	fprintf(stderr, "importer: dialog up, the child of wayland handle %s\n", handle);
	time_t end = time(NULL) + secs;
	while (time(NULL) < end) {
		wl_display_roundtrip(dpy);
		usleep(100000);
	}
	wl_display_disconnect(dpy);
	return 0;
}

static int x11(const char *id, const char *title, int secs) {
	Display *d = XOpenDisplay(NULL);
	if (!d) { fprintf(stderr, "importer: no X display\n"); return 1; }
	Window parent = (Window)strtoul(id, NULL, 16);
	int scr = DefaultScreen(d);
	Window w = XCreateSimpleWindow(d, RootWindow(d, scr), 0, 0, W, H, 0, 0, 0xe8e8e8);
	XStoreName(d, w, title);
	XSetTransientForHint(d, w, parent);
	Atom type = XInternAtom(d, "_NET_WM_WINDOW_TYPE", False), dlg = XInternAtom(d, "_NET_WM_WINDOW_TYPE_DIALOG", False);
	XChangeProperty(d, w, type, XA_ATOM, 32, PropModeReplace, (unsigned char *)&dlg, 1);
	Atom state = XInternAtom(d, "_NET_WM_STATE", False), modal = XInternAtom(d, "_NET_WM_STATE_MODAL", False);
	XChangeProperty(d, w, state, XA_ATOM, 32, PropModeReplace, (unsigned char *)&modal, 1);
	XMapWindow(d, w);
	XFlush(d);
	fprintf(stderr, "importer: dialog 0x%lx up, transient for 0x%lx\n", w, parent);
	time_t end = time(NULL) + secs;
	while (time(NULL) < end) {
		while (XPending(d)) { XEvent e; XNextEvent(d, &e); }
		usleep(100000);
	}
	XCloseDisplay(d);
	return 0;
}

int main(int argc, char **argv) {
	if (argc < 4) { fprintf(stderr, "usage: importer PARENT TITLE SECONDS\n"); return 2; }
	const char *parent = argv[1], *title = argv[2];
	int secs = atoi(argv[3]);
	if (!strncmp(parent, "wayland:", 8)) return wayland(parent + 8, title, secs);
	if (!strncmp(parent, "x11:", 4)) return x11(parent + 4, title, secs);
	fprintf(stderr, "importer: parent \"%s\" names no window\n", parent);
	return 1;
}
