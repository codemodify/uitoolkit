//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <stdint.h>
#include <stdlib.h>
#include <wayland-client.h>
#include "wlr-layer-shell-unstable-v1-client-protocol.h"

extern void uitkWlLayerConfigure(uintptr_t sid, uint32_t serial, uint32_t w, uint32_t h);
extern void uitkWlLayerClosed(uintptr_t sid);

static void *ui_wl_bind_layer_shell(struct wl_registry *r, uint32_t name, uint32_t ver) {
	return wl_registry_bind(r, name, &zwlr_layer_shell_v1_interface, ver);
}
static void ui_wl_layer_shell_destroy(void *shell) {
	struct zwlr_layer_shell_v1 *s = shell;
	if (!s) return;
	if (zwlr_layer_shell_v1_get_version(s) >= ZWLR_LAYER_SHELL_V1_DESTROY_SINCE_VERSION)
		zwlr_layer_shell_v1_destroy(s);
}

static void uitk_layer_configure(void *d, struct zwlr_layer_surface_v1 *ls, uint32_t serial, uint32_t w, uint32_t h) {
	(void)ls;
	uitkWlLayerConfigure((uintptr_t)d, serial, w, h);
}
static void uitk_layer_closed(void *d, struct zwlr_layer_surface_v1 *ls) {
	(void)ls;
	uitkWlLayerClosed((uintptr_t)d);
}
static const struct zwlr_layer_surface_v1_listener uitk_layer_listener = {
	.configure = uitk_layer_configure, .closed = uitk_layer_closed,
};

// ui_wl_layer_get gives surf the layer-surface role on the overlay layer,
// anchored to the top-left corner, and states everything that does not
// change while it lives. out may be NULL: the compositor then picks the
// output.
static void *ui_wl_layer_get(void *shell, struct wl_surface *surf, struct wl_output *out, const char *ns, uintptr_t sid) {
	struct zwlr_layer_shell_v1 *sh = shell;
	if (!sh || !surf) return NULL;
	struct zwlr_layer_surface_v1 *ls = zwlr_layer_shell_v1_get_layer_surface(
		sh, surf, out, ZWLR_LAYER_SHELL_V1_LAYER_OVERLAY, (char *)ns);
	if (!ls) return NULL;
	zwlr_layer_surface_v1_add_listener(ls, &uitk_layer_listener, (void *)sid);
	zwlr_layer_surface_v1_set_anchor(ls,
		ZWLR_LAYER_SURFACE_V1_ANCHOR_TOP | ZWLR_LAYER_SURFACE_V1_ANCHOR_LEFT);
	zwlr_layer_surface_v1_set_exclusive_zone(ls, -1);
	return ls;
}
static void ui_wl_layer_size(void *ls, int w, int h) {
	if (ls && w > 0 && h > 0) zwlr_layer_surface_v1_set_size(ls, (uint32_t)w, (uint32_t)h);
}
static void ui_wl_layer_margin(void *ls, int top, int left) {
	if (ls) zwlr_layer_surface_v1_set_margin(ls, top, 0, 0, left);
}
// ui_wl_layer_keyboard asks for the keyboard. on_demand — the menu takes
// it while the pointer is in it and gives it back — arrived in version 4;
// before that the request was a boolean, and 1 meant "always", which is
// the closest an old compositor can come.
static void ui_wl_layer_keyboard(void *ls) {
	struct zwlr_layer_surface_v1 *l = ls;
	if (!l) return;
	uint32_t mode = 1;
	if (zwlr_layer_surface_v1_get_version(l) >= ZWLR_LAYER_SURFACE_V1_KEYBOARD_INTERACTIVITY_ON_DEMAND_SINCE_VERSION)
		mode = ZWLR_LAYER_SURFACE_V1_KEYBOARD_INTERACTIVITY_ON_DEMAND;
	zwlr_layer_surface_v1_set_keyboard_interactivity(l, mode);
}
static void ui_wl_layer_ack(void *ls, uint32_t serial) {
	if (ls) zwlr_layer_surface_v1_ack_configure(ls, serial);
}
static void ui_wl_layer_destroy(void *ls) {
	if (ls) zwlr_layer_surface_v1_destroy(ls);
}
// The cgo preamble is per-file, so commit and flush are spelled again here.
static void ui_wl_layer_commit(struct wl_surface *s) { if (s) wl_surface_commit(s); }
static void ui_wl_layer_flush(struct wl_display *d) { if (d) wl_display_flush(d); }
*/
import "C"

import (
	"os"
	"strings"
	"unsafe"
)

// zwlr_layer_shell_v1 — placing a window where it has to be.
//
// A Wayland client cannot place an xdg_toplevel: the protocol has no
// request for it and the compositor never tells the client where the
// window went. That is fine for a document window and wrong for a tray
// menu, which the host asks for at a point in root coordinates
// (org.kde.StatusNotifierItem.ContextMenu) and which belongs next to the
// icon that was clicked. Before this the toolkit's tray menu opened as a
// toplevel and KWin dropped it in the middle of the screen.
//
// wlr-layer-shell is how every Wayland client that must be at an absolute
// position does it — panels, launchers, notification daemons, on-screen
// keyboards. A layer surface is anchored to an edge or corner of one
// output and offset from it by margins, so "top-left, margin (x, y)" is
// an absolute position. KDE (KWin), sway, Hyprland and wayfire all offer
// it; GNOME/Mutter has declined it, and there [ScreenPlacementAvailable]
// answers false and the layer above takes a different road.
//
// The toolkit uses it for exactly one thing today, the tray menu of
// [WindowOptions].Place == [PlaceAtScreen]:
//
//	layer          overlay — a menu goes above panels, and a tray menu is
//	               opened *from* a panel
//	anchor         top|left, so the margins are a position rather than a
//	               distance from some other edge
//	margins        the requested point (left, top), measured from the
//	               top-left corner of the output it lands on
//	set_size       the surface's size, margin included: a layer surface
//	               has no xdg window geometry to state a visible box with
//	exclusive zone -1 — the menu must not push panels or maximized
//	               windows out of its way, and must not be pushed itself
//	keyboard       on_demand, so the menu can take the keyboard for its
//	               own grab (arrow keys, Escape) and give it back
//
// None of this is exercised by the test suite: it needs a compositor.
// What is testable — which output a point lands on, the margins that come
// out of it, the decision to use a layer surface at all — lives in
// wlstate.go and placement.go and is.

// EnvLayerShell turns the layer-shell path off: UITK_LAYER_SHELL=0 makes
// the toolkit behave as if the compositor had no zwlr_layer_shell_v1, so
// the GNOME road can be walked on a KDE machine.
const EnvLayerShell = "UITK_LAYER_SHELL"

// layerShellWanted reports whether UITK_LAYER_SHELL leaves the protocol in
// play.
func layerShellWanted() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvLayerShell))) {
	case "0", "off", "false", "no":
		return false
	}
	return true
}

// layerNamespace is what the compositor calls these surfaces in its logs
// and its rules.
const layerNamespace = "uitoolkit-menu"

// bindLayerShell binds zwlr_layer_shell_v1, up to version 5 (v2 added
// set_layer, v3 the layer_shell.destroy request, v4 on-demand keyboard
// interactivity, v5 set_exclusive_edge). The version is kept because the
// requests we send are gated on it.
func (c *wlConn) bindLayerShell(reg *C.struct_wl_registry, name C.uint32_t, ver C.uint32_t) {
	if c == nil || c.layerShell != nil || !layerShellWanted() {
		return
	}
	v := ver
	if v > 5 {
		v = 5
	}
	if v < 1 {
		return
	}
	c.layerShell = unsafe.Pointer(C.ui_wl_bind_layer_shell(reg, name, v))
	if c.layerShell != nil {
		c.layerVer = int(v)
	}
}

func (c *wlConn) destroyLayerShellLocked() {
	if c == nil || c.layerShell == nil {
		return
	}
	C.ui_wl_layer_shell_destroy(c.layerShell)
	c.layerShell = nil
	c.layerVer = 0
}

// LayerSurfacesAvailable reports whether the compositor offers
// zwlr_layer_shell_v1, which is whether this process can put a window at
// an absolute point of the screen (see [ScreenPlacementAvailable], which
// is what callers normally want).
//
// It answers from the live connection where there is one. With none it
// opens one to ask and drops it again, which is a round trip to the
// compositor; call it once and remember the answer.
func LayerSurfacesAvailable() bool {
	if !layerShellWanted() {
		return false
	}
	wlMu.Lock()
	c := wlc
	wlMu.Unlock()
	if c != nil {
		return c.layerShell != nil
	}
	c, err := wlRetain()
	if err != nil || c == nil {
		return false
	}
	ok := c.layerShell != nil
	c.release()
	return ok
}

// LayerShellVersion is the version of zwlr_layer_shell_v1 bound, or 0
// where there is none. Version 4 is where keyboard interactivity became
// on-demand rather than all-or-nothing.
func LayerShellVersion() int {
	wlMu.Lock()
	c := wlc
	wlMu.Unlock()
	if c == nil || c.layerShell == nil {
		return 0
	}
	return c.layerVer
}

// wlLayer is a surface's layer-shell role: the zwlr_layer_surface_v1, the
// output it is anchored to (0 when the compositor chose), and the desktop
// point it was last asked for.
type wlLayer struct {
	obj  unsafe.Pointer
	out  uint32
	x, y int
	// gone is set by zwlr_layer_surface_v1.closed: the compositor has
	// taken the surface away and the role object may only be destroyed.
	gone bool
}

// wantsLayer reports whether this surface should take the layer-shell
// road: it asked to be at an absolute point and the compositor can do it.
func (s *wlSurface) wantsLayer() bool {
	return s != nil && s.placeAtScreen && s.conn != nil && s.conn.layerShell != nil
}

// bindLayerLocked gives the surface its layer-shell role, in place of the
// xdg_toplevel it would otherwise have had. It is the layer-shell half of
// bindToplevelLocked and is called from it.
func (s *wlSurface) bindLayerLocked() {
	if s == nil || s.surf == nil || s.layer.obj != nil || !s.wantsLayer() {
		return
	}
	name, left, top := s.conn.outs.layerMargins(s.layer.x, s.layer.y)
	var out *C.struct_wl_output
	if name != 0 {
		out = s.conn.outObjs[name]
	}
	ns := C.CString(layerNamespace)
	defer C.free(unsafe.Pointer(ns))
	obj := C.ui_wl_layer_get(s.conn.layerShell, s.surf, out, ns, C.uintptr_t(s.id))
	if obj == nil {
		return
	}
	s.layer.obj = unsafe.Pointer(obj)
	s.layer.out = name
	s.layer.gone = false
	lw, lh := s.surfaceLogical()
	C.ui_wl_layer_size(s.layer.obj, C.int(lw), C.int(lh))
	C.ui_wl_layer_margin(s.layer.obj, C.int(top), C.int(left))
	C.ui_wl_layer_keyboard(s.layer.obj)
	s.configured = false
	C.ui_wl_layer_commit(s.surf)
}

// destroyLayerLocked drops the role. The wl_surface survives, so Show can
// make a new one — the same dance Hide/Show does with xdg_toplevel.
func (s *wlSurface) destroyLayerLocked() {
	if s == nil || s.layer.obj == nil {
		return
	}
	C.ui_wl_layer_destroy(s.layer.obj)
	s.layer.obj = nil
	s.layer.out = 0
	s.layer.gone = false
}

// PlaceAtScreen puts the window at x, y in logical pixels of the desktop
// ([ScreenPlacer]). It works only for a window opened with Place:
// PlaceAtScreen on a compositor with zwlr_layer_shell_v1; every other
// Wayland window answers false, because a toplevel has no position.
//
// Moving means new margins. The output cannot be changed on a live layer
// surface (zwlr_layer_shell_v1 takes it at creation and has no request to
// change it), so a point that lands on another monitor makes a new role
// object on that monitor — the surface is unmapped and remapped, which
// for a menu that is being opened anyway costs nothing.
func (s *wlSurface) PlaceAtScreen(x, y int) bool {
	if s == nil || s.closed || !s.wantsLayer() {
		return false
	}
	s.layer.x, s.layer.y = x, y
	if s.layer.obj == nil || s.layer.gone {
		s.destroyLayerLocked()
		s.bindLayerLocked()
		return s.layer.obj != nil
	}
	name, left, top := s.conn.outs.layerMargins(x, y)
	if name != s.layer.out {
		// Another monitor: the anchor output is fixed at creation.
		s.unmapToplevelLocked()
		s.bindLayerLocked()
		return s.layer.obj != nil
	}
	lw, lh := s.surfaceLogical()
	C.ui_wl_layer_size(s.layer.obj, C.int(lw), C.int(lh))
	C.ui_wl_layer_margin(s.layer.obj, C.int(top), C.int(left))
	C.ui_wl_layer_commit(s.surf)
	if s.conn.dpy != nil {
		C.ui_wl_layer_flush(s.conn.dpy)
	}
	return true
}

//export uitkWlLayerConfigure
func uitkWlLayerConfigure(sid C.uintptr_t, serial C.uint32_t, w, h C.uint32_t) {
	s := wlSurfBy(sid)
	if s == nil || s.layer.obj == nil {
		return
	}
	// A layer surface acks on its own role object; there is no
	// xdg_surface under it and so no uitkWlXdgConfigure for these.
	C.ui_wl_layer_ack(s.layer.obj, serial)
	s.configured = true
	if s.surf != nil {
		wlByNative[uintptr(unsafe.Pointer(s.surf))] = s.id
	}
	// The compositor may hand back a size of its own (it does when the
	// surface asked for 0 in a dimension, or would not fit). It is the
	// whole surface, margin included, which is the unit set_size speaks.
	if w > 0 && h > 0 {
		lw, lh := layerWindowSize(int(w), int(h), s.marginLogical())
		if lw != s.logicalW || lh != s.logicalH {
			s.wantW, s.wantH = lw, lh
			s.logicalW, s.logicalH = lw, lh
			s.push(Event{Kind: EventResize, Width: lw, Height: lh})
		}
	}
}

//export uitkWlLayerClosed
func uitkWlLayerClosed(sid C.uintptr_t) {
	s := wlSurfBy(sid)
	if s == nil {
		return
	}
	// The compositor has taken the surface away — the output went, or a
	// session lock came up. The role object is dead; the app hears the
	// same EventClose an xdg_toplevel.close would have given it, and the
	// next Show makes a fresh role.
	s.layer.gone = true
	s.push(Event{Kind: EventClose})
}

// wlScreenRectAt is the logical rectangle of the output holding the
// desktop point x, y ([ScreenRectAt]) — the rectangle a layer surface's
// margins are measured in, and so the one a window placed at that point
// must be constrained to.
//
// It answers only from a live connection: the outputs are known through
// the registry, and opening one here to ask would be a round trip in the
// middle of opening a menu. With no connection, and while the compositor
// has not yet sent an output's geometry, it says so and the caller
// constrains against nothing.
func wlScreenRectAt(x, y int) (FrameRect, bool) {
	wlMu.Lock()
	c := wlc
	wlMu.Unlock()
	if c == nil || c.outs == nil {
		return FrameRect{}, false
	}
	_, box, ok := c.outs.outputAt(x, y)
	return box, ok
}
