//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#define _GNU_SOURCE
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>
#include <wayland-client.h>
#include "xdg-shell-client-protocol.h"
#include "xdg-toplevel-icon-v1-client-protocol.h"
#include "kde_palette.h"

static void *ui_wl_bind_dress(struct wl_registry *r, uint32_t name, const struct wl_interface *iface, uint32_t ver) {
	return wl_registry_bind(r, name, iface, ver);
}
static const struct wl_interface *ui_wl_palette_man_iface(void) { return &uitk_kde_palette_manager_interface; }
static const struct wl_interface *ui_wl_icon_man_iface(void) { return &xdg_toplevel_icon_manager_v1_interface; }

static void ui_wl_proxy_destroy(void *p) { if (p) wl_proxy_destroy((struct wl_proxy *)p); }
static void ui_wl_icon_man_destroy(void *m) {
	if (m) xdg_toplevel_icon_manager_v1_destroy((struct xdg_toplevel_icon_manager_v1 *)m);
}
static void ui_wl_surface_commit_dress(struct wl_surface *s) { if (s) wl_surface_commit(s); }
static void ui_wl_flush_dress(struct wl_display *d) { if (d) wl_display_flush(d); }

// ui_wl_icon_buffers puts n square images, their ARGB8888 bytes packed one
// after the other in px (sides[i] × sides[i] × 4 each), into one wl_shm
// pool and makes a wl_buffer of each into bufs. The pool and the mapping
// are dropped at once: the buffers keep the memory alive on both sides,
// and the protocol forbids touching the bytes after add_buffer anyway.
static int ui_wl_icon_buffers(struct wl_shm *shm, const uint8_t *px, const int *sides, int n, struct wl_buffer **bufs) {
	size_t total = 0;
	for (int i = 0; i < n; i++) total += (size_t)sides[i] * (size_t)sides[i] * 4;
	if (!shm || n <= 0 || total == 0) return 0;
	int fd = memfd_create("uitk-icon", MFD_CLOEXEC);
	if (fd < 0) return 0;
	if (ftruncate(fd, (off_t)total) != 0) { close(fd); return 0; }
	void *map = mmap(NULL, total, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	if (map == MAP_FAILED) { close(fd); return 0; }
	memcpy(map, px, total);
	munmap(map, total);
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, (int32_t)total);
	close(fd);
	if (!pool) return 0;
	size_t off = 0;
	for (int i = 0; i < n; i++) {
		bufs[i] = wl_shm_pool_create_buffer(pool, (int32_t)off, sides[i], sides[i], sides[i] * 4, WL_SHM_FORMAT_ARGB8888);
		off += (size_t)sides[i] * (size_t)sides[i] * 4;
	}
	wl_shm_pool_destroy(pool);
	return 1;
}

static void *ui_wl_icon_create(void *man) {
	if (!man) return NULL;
	return xdg_toplevel_icon_manager_v1_create_icon((struct xdg_toplevel_icon_manager_v1 *)man);
}
static void ui_wl_icon_add(void *icon, struct wl_buffer *b, int32_t scale) {
	if (icon && b) xdg_toplevel_icon_v1_add_buffer((struct xdg_toplevel_icon_v1 *)icon, b, scale);
}
static void ui_wl_icon_set(void *man, struct xdg_toplevel *top, void *icon) {
	if (man && top) xdg_toplevel_icon_manager_v1_set_icon((struct xdg_toplevel_icon_manager_v1 *)man, top, (struct xdg_toplevel_icon_v1 *)icon);
}
static void ui_wl_icon_destroy(void *icon) {
	if (icon) xdg_toplevel_icon_v1_destroy((struct xdg_toplevel_icon_v1 *)icon);
}
static void ui_wl_buffer_destroy_dress(struct wl_buffer *b) { if (b) wl_buffer_destroy(b); }

static void *ui_wl_palette_create(void *man, struct wl_surface *s) {
	return uitk_kde_palette_create((struct wl_proxy *)man, s);
}
static void ui_wl_palette_set(void *p, const char *path) { uitk_kde_palette_set((struct wl_proxy *)p, path); }
static void ui_wl_palette_release(void *p) { uitk_kde_palette_release((struct wl_proxy *)p); }
*/
import "C"

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// The window's dress on Wayland (windowdress.go): KWin's server-decoration
// palette and xdg-toplevel-icon-v1.

// wlDress is a surface's share of it: the palette object and the palette
// last sent, the icon the app gave and the icon object on the wire with
// the buffers it holds.
type wlDress struct {
	palette     unsafe.Pointer // *org_kde_kwin_server_decoration_palette
	paletteSent string
	icon        []*paintengine2d.Image
	iconObj     unsafe.Pointer // *xdg_toplevel_icon_v1
	iconBufs    []*C.struct_wl_buffer
}

// bindPaletteManager binds KWin's org_kde_kwin_server_decoration_palette_manager.
func (c *wlConn) bindPaletteManager(reg *C.struct_wl_registry, name C.uint32_t) {
	c.paletteMan = C.ui_wl_bind_dress(reg, name, C.ui_wl_palette_man_iface(), 1)
}

// bindIconManager binds xdg_toplevel_icon_manager_v1.
func (c *wlConn) bindIconManager(reg *C.struct_wl_registry, name C.uint32_t) {
	c.iconMan = C.ui_wl_bind_dress(reg, name, C.ui_wl_icon_man_iface(), 1)
}

// destroyDressLocked lets go of the managers as the connection closes.
func (c *wlConn) destroyDressLocked() {
	// The palette manager has no destructor request: the proxy only.
	C.ui_wl_proxy_destroy(c.paletteMan)
	c.paletteMan = nil
	C.ui_wl_icon_man_destroy(c.iconMan)
	c.iconMan = nil
}

// DecorationPaletteSupported reports whether KWin advertised its palette
// manager (DecorationPaletteSurface).
func (s *wlSurface) DecorationPaletteSupported() bool {
	return s != nil && !s.popup && s.conn != nil && s.conn.paletteMan != nil
}

// SetDecorationPalette names the colour scheme KWin paints the window's
// frame with (DecorationPaletteSurface). The palette belongs to the
// wl_surface, so it outlives a hidden window's role.
func (s *wlSurface) SetDecorationPalette(path string) {
	if s == nil || s.closed || s.surf == nil || s.popup || s.conn == nil {
		return
	}
	plan := planPalette(s.conn.paletteMan != nil, s.dress.palette != nil, s.dress.paletteSent, path)
	if plan.create {
		s.dress.palette = C.ui_wl_palette_create(s.conn.paletteMan, s.surf)
		if s.dress.palette == nil {
			return
		}
	}
	if !plan.set {
		return
	}
	cp := C.CString(path)
	C.ui_wl_palette_set(s.dress.palette, cp)
	C.free(unsafe.Pointer(cp))
	s.dress.paletteSent = path
	C.ui_wl_flush_dress(s.conn.dpy)
}

// SetIcon gives the toplevel its icon (IconSurface). The icon is sent now
// if the window has a role and again whenever it gets a new one (Show
// after Hide).
func (s *wlSurface) SetIcon(images []*paintengine2d.Image) {
	if s == nil || s.closed || s.popup {
		return
	}
	s.dress.icon = iconImages(images)
	s.applyIconLocked()
	if s.conn != nil && s.conn.dpy != nil {
		if s.mapped {
			// set_icon is double-buffered: it takes on the next commit.
			C.ui_wl_surface_commit_dress(s.surf)
		}
		C.ui_wl_flush_dress(s.conn.dpy)
	}
}

// applyIconLocked puts the icon on the toplevel: a new icon object with a
// buffer per size, set_icon, and then the icon it replaces goes — its
// buffers after it, since a buffer must outlive every icon that holds it.
// No images sets a null icon, the desktop's default.
func (s *wlSurface) applyIconLocked() {
	c := s.conn
	if c == nil || c.iconMan == nil || c.shm == nil || s.top == nil {
		return
	}
	oldObj, oldBufs := s.dress.iconObj, s.dress.iconBufs
	s.dress.iconObj, s.dress.iconBufs = nil, nil
	if imgs := s.dress.icon; len(imgs) > 0 {
		var px []byte
		sides := make([]C.int, len(imgs))
		for i, im := range imgs {
			px = append(px, wlIconPixels(im)...)
			sides[i] = C.int(im.Width)
		}
		bufs := make([]*C.struct_wl_buffer, len(imgs))
		cpx := C.CBytes(px)
		ok := C.ui_wl_icon_buffers(c.shm, (*C.uint8_t)(cpx), &sides[0], C.int(len(imgs)), &bufs[0]) != 0
		C.free(cpx)
		if ok {
			obj := C.ui_wl_icon_create(c.iconMan)
			for _, b := range bufs {
				C.ui_wl_icon_add(obj, b, 1)
			}
			s.dress.iconObj, s.dress.iconBufs = obj, bufs
		}
	}
	if s.dress.iconObj != nil || oldObj != nil {
		// A null icon puts the desktop's default back.
		C.ui_wl_icon_set(c.iconMan, s.top, s.dress.iconObj)
	}
	dropIcon(oldObj, oldBufs)
}

// dropIcon destroys an icon object and then its buffers.
func dropIcon(obj unsafe.Pointer, bufs []*C.struct_wl_buffer) {
	C.ui_wl_icon_destroy(obj)
	for _, b := range bufs {
		C.ui_wl_buffer_destroy_dress(b)
	}
}

// closeDress lets go of the surface's palette and icon as it closes.
func (s *wlSurface) closeDress() {
	if s.dress.palette != nil {
		C.ui_wl_palette_release(s.dress.palette)
		s.dress.palette = nil
	}
	dropIcon(s.dress.iconObj, s.dress.iconBufs)
	s.dress.iconObj, s.dress.iconBufs = nil, nil
}

var (
	_ DecorationPaletteSurface = (*wlSurface)(nil)
	_ IconSurface              = (*wlSurface)(nil)
)
