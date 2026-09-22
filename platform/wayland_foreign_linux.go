//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <stdint.h>
#include <wayland-client.h>
#include "xdg-foreign-unstable-v2-client-protocol.h"

extern void uitkWlExportHandle(uintptr_t sid, char *handle);

static void *ui_wl_bind_exporter(struct wl_registry *r, uint32_t name) {
	return wl_registry_bind(r, name, &zxdg_exporter_v2_interface, 1);
}
static void uitk_exported_handle(void *d, struct zxdg_exported_v2 *e, const char *handle) {
	(void)e;
	uitkWlExportHandle((uintptr_t)d, (char *)handle);
}
static const struct zxdg_exported_v2_listener uitk_exported_listener = { .handle = uitk_exported_handle };
static void *ui_wl_export(void *exporter, struct wl_surface *s, uintptr_t sid) {
	if (!exporter || !s) return NULL;
	struct zxdg_exported_v2 *e = zxdg_exporter_v2_export_toplevel(exporter, s);
	if (e) zxdg_exported_v2_add_listener(e, &uitk_exported_listener, (void *)sid);
	return e;
}
static void ui_wl_unexport(void *e) { if (e) zxdg_exported_v2_destroy(e); }
*/
import "C"

import "unsafe"

// wlExport is a toplevel exported with xdg-foreign (zxdg_exporter_v2) so
// another client — the desktop portal's file dialog — can make its own
// window a child of it: obj is the zxdg_exported_v2, handle what the
// compositor named it.
type wlExport struct {
	obj    unsafe.Pointer
	handle string
}

// wlExportRoundtrips is how many round trips PortalParent waits for the
// compositor to name an exported window (it names it in the first).
const wlExportRoundtrips = 3

func (c *wlConn) bindExporter(reg *C.struct_wl_registry, name C.uint32_t) {
	if c.exporter == nil {
		c.exporter = C.ui_wl_bind_exporter(reg, name)
	}
}

// PortalParent is the window's parent id for a desktop portal dialog,
// "wayland:<handle>": the toplevel is exported through xdg-foreign the
// first time it is asked for, and the compositor's handle is kept for as
// long as the window has its role. The portal hands the handle to its
// dialog, which the compositor then keeps above the window, modal to it and
// centred on it — KWin and Mutter both do. Empty where the compositor has
// no zxdg_exporter_v2, or for a hidden window: the dialog then opens on its
// own, as it did before.
//
// It waits for the compositor's answer, so it is called from the UI
// goroutine like the rest of the surface.
func (s *wlSurface) PortalParent() string {
	if s == nil || s.conn == nil || s.conn.exporter == nil || s.surf == nil || s.top == nil || s.popup {
		return ""
	}
	if s.export.obj == nil {
		s.export.obj = C.ui_wl_export(s.conn.exporter, s.surf, C.uintptr_t(s.id))
		if s.export.obj == nil {
			return ""
		}
	}
	for i := 0; i < wlExportRoundtrips && s.export.handle == ""; i++ {
		s.conn.roundtrip()
	}
	return wlPortalParent(s.export.handle)
}

// wlPortalParent is the portal's name for an exported toplevel.
func wlPortalParent(handle string) string {
	if handle == "" {
		return ""
	}
	return "wayland:" + handle
}

// unexportLocked revokes the export: the handle names the role, and a
// window that loses it (Hide, Close) is exported afresh when asked again.
func (s *wlSurface) unexportLocked() {
	if s.export.obj != nil {
		C.ui_wl_unexport(s.export.obj)
	}
	s.export = wlExport{}
}

//export uitkWlExportHandle
func uitkWlExportHandle(sid C.uintptr_t, handle *C.char) {
	if s := wlSurfBy(sid); s != nil && s.export.obj != nil {
		s.export.handle = C.GoString(handle)
	}
}
