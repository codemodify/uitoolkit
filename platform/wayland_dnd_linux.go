//go:build linux && cgo

package platform

// Dragging out of a Wayland window: wl_data_source, the types it offers,
// the actions it allows, wl_data_device.start_drag from the serial of the
// press that began it, and a drag-icon surface that the compositor moves
// with the pointer.
//
// The target half — taking a drop — is in wayland_linux.go, where the
// data device's own listener lives. Only the source is here.
//
// cgo compiles every Go file's preamble on its own and a static function
// belongs to the file that declares it, so the helpers below are this
// file's own even where wayland_linux.go has one of the same shape. They
// are named ui_wd_* to say so.

/*
#cgo linux pkg-config: wayland-client
#define _GNU_SOURCE
#include <wayland-client.h>
#include <errno.h>
#include <fcntl.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <time.h>
#include <unistd.h>

extern void uitkWlDragSend(uintptr_t id, char *mime, int fd);
extern void uitkWlDragTarget(uintptr_t id, char *mime);
extern void uitkWlDragCancelled(uintptr_t id);
extern void uitkWlDragDropped(uintptr_t id);
extern void uitkWlDragFinished(uintptr_t id);
extern void uitkWlDragAction(uintptr_t id, uint32_t action);

// A drag source needs its own listener: the clipboard's tells the
// clipboard what to send, and a drag has three events the clipboard
// never sees (dnd_drop_performed, dnd_finished, action).
static void uitk_drag_target(void *data, struct wl_data_source *s, const char *mime) {
	(void)s;
	uitkWlDragTarget((uintptr_t)data, (char*)mime);
}
static void uitk_drag_send(void *data, struct wl_data_source *s, const char *mime, int32_t fd) {
	(void)s;
	uitkWlDragSend((uintptr_t)data, (char*)mime, fd);
}
static void uitk_drag_cancelled(void *data, struct wl_data_source *s) {
	(void)s;
	uitkWlDragCancelled((uintptr_t)data);
}
static void uitk_drag_dropped(void *data, struct wl_data_source *s) {
	(void)s;
	uitkWlDragDropped((uintptr_t)data);
}
static void uitk_drag_finished(void *data, struct wl_data_source *s) {
	(void)s;
	uitkWlDragFinished((uintptr_t)data);
}
static void uitk_drag_action(void *data, struct wl_data_source *s, uint32_t a) {
	(void)s;
	uitkWlDragAction((uintptr_t)data, a);
}
static const struct wl_data_source_listener uitk_drag_listener = {
	.target = uitk_drag_target,
	.send = uitk_drag_send,
	.cancelled = uitk_drag_cancelled,
	.dnd_drop_performed = uitk_drag_dropped,
	.dnd_finished = uitk_drag_finished,
	.action = uitk_drag_action,
};

static struct wl_data_source *ui_wd_source(struct wl_data_device_manager *m, uintptr_t id) {
	struct wl_data_source *s = wl_data_device_manager_create_data_source(m);
	if (s) wl_data_source_add_listener(s, &uitk_drag_listener, (void*)id);
	return s;
}

static void ui_wd_offer(struct wl_data_source *s, const char *mime) {
	if (s) wl_data_source_offer(s, mime);
}

// set_actions arrived in version 3. Against an older compositor the drag
// still works; it is just always a copy, which is the protocol's own
// fallback.
static void ui_wd_set_actions(struct wl_data_source *s, uint32_t actions) {
	if (s && wl_data_source_get_version(s) >= 3) wl_data_source_set_actions(s, actions);
}

static void ui_wd_start(struct wl_data_device *d, struct wl_data_source *src,
		struct wl_surface *origin, struct wl_surface *icon, uint32_t serial) {
	if (d) wl_data_device_start_drag(d, src, origin, icon, serial);
}

static void ui_wd_source_destroy(struct wl_data_source *s) { if (s) wl_data_source_destroy(s); }

// The target half's two answers to a drag over the window: the type it
// would read the drop as, and what it would do with it.
static void ui_wd_accept(struct wl_data_offer *o, uint32_t serial, const char *mime) {
	if (o) wl_data_offer_accept(o, serial, mime);
}
static void ui_wd_offer_actions(struct wl_data_offer *o, uint32_t actions, uint32_t preferred) {
	if (o && wl_data_offer_get_version(o) >= 3) wl_data_offer_set_actions(o, actions, preferred);
}

static struct wl_surface *ui_wd_surface(struct wl_compositor *c) {
	return c ? wl_compositor_create_surface(c) : NULL;
}
static void ui_wd_surface_destroy(struct wl_surface *s) { if (s) wl_surface_destroy(s); }
// ui_wd_attach puts the buffer on the surface, moved by dx,dy — which is
// how the icon's hotspot lands under the pointer. Version 5 made a
// non-zero offset in attach a protocol error and gave it its own request,
// so the two have to be told apart.
static void ui_wd_attach(struct wl_surface *s, struct wl_buffer *b, int dx, int dy) {
	if (!s) return;
	if (wl_surface_get_version(s) >= 5) {
		wl_surface_attach(s, b, 0, 0);
		if (dx || dy) wl_surface_offset(s, dx, dy);
		return;
	}
	wl_surface_attach(s, b, dx, dy);
}
static void ui_wd_damage(struct wl_surface *s, int w, int h) {
	if (s) wl_surface_damage_buffer(s, 0, 0, w, h);
}
static void ui_wd_scale(struct wl_surface *s, int scale) {
	if (s && scale > 0) wl_surface_set_buffer_scale(s, scale);
}
static void ui_wd_commit(struct wl_surface *s) { if (s) wl_surface_commit(s); }
static void ui_wd_flush(struct wl_display *d) { if (d) wl_display_flush(d); }

static int ui_wd_memfd(size_t size, void **map) {
	char tmpl[64];
	int fd;
	void *p;
	snprintf(tmpl, sizeof(tmpl), "/dev/shm/uitk-dnd-XXXXXX");
	fd = mkstemp(tmpl);
	if (fd < 0) {
		snprintf(tmpl, sizeof(tmpl), "/tmp/uitk-dnd-XXXXXX");
		fd = mkstemp(tmpl);
	}
	if (fd < 0) return -1;
	{
		int fl = fcntl(fd, F_GETFD);
		if (fl >= 0) fcntl(fd, F_SETFD, fl | FD_CLOEXEC);
	}
	unlink(tmpl);
	if (ftruncate(fd, (off_t)size) != 0) { close(fd); return -1; }
	p = mmap(NULL, size, PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
	if (p == MAP_FAILED) { close(fd); return -1; }
	*map = p;
	return fd;
}

// The icon is always ARGB8888 and premultiplied: it is drawn over the
// desktop, so its transparency is the whole point.
static struct wl_buffer *ui_wd_buffer(struct wl_shm *shm, int fd, int w, int h, int stride, size_t size) {
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, (int32_t)size);
	struct wl_buffer *buf;
	if (!pool) return NULL;
	buf = wl_shm_pool_create_buffer(pool, 0, w, h, stride, WL_SHM_FORMAT_ARGB8888);
	wl_shm_pool_destroy(pool);
	return buf;
}

static void ui_wd_buf_destroy(struct wl_buffer *b) { if (b) wl_buffer_destroy(b); }
static void ui_wd_munmap(void *p, size_t n) { if (p && p != MAP_FAILED) munmap(p, n); }
static void ui_wd_close(int fd) { if (fd >= 0) close(fd); }

static void ui_wd_nonblock(int fd) {
	int fl;
	if (fd < 0) return;
	fl = fcntl(fd, F_GETFL);
	if (fl >= 0) fcntl(fd, F_SETFL, fl | O_NONBLOCK);
}

// ui_wd_write_all writes the whole payload with a deadline. This runs on
// the UI thread: a receiver that stops reading must not be able to hang
// the application, nor to truncate a large drop silently.
static ssize_t ui_wd_write_all(int fd, const char *p, size_t n, int ms) {
	size_t off = 0;
	int waited = 0;
	while (off < n) {
		ssize_t w = write(fd, p + off, n - off);
		if (w > 0) {
			off += (size_t)w;
			continue;
		}
		if (w < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
			struct timespec ts;
			if (waited >= ms) break;
			ts.tv_sec = 0;
			ts.tv_nsec = 1000000;
			nanosleep(&ts, NULL);
			waited++;
			continue;
		}
		break;
	}
	return (ssize_t)off;
}
*/
import "C"

import (
	"time"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// wlDrag is a drag started out of one of this connection's windows.
type wlDrag struct {
	src     *C.struct_wl_data_source
	payload DragPayload
	// surf is the window the drag started in: it is told how the drag
	// ended.
	surf int
	// icon is the surface the compositor moves with the pointer.
	icon     *C.struct_wl_surface
	iconBuf  *C.struct_wl_buffer
	iconMem  unsafe.Pointer
	iconSize int
	// action is the compositor's latest word on what the drop will do.
	action DragAction
	// dropped is set by dnd_drop_performed: the pointer is up and the
	// target is reading, and the source must stay alive until it says it
	// is done.
	dropped bool
}

// wlDragActions maps the toolkit's actions to the protocol's bit field
// (wl_data_device_manager.dnd_action). Wayland has copy, move and ask;
// it has no link, so a link-only drag is offered as a copy — a drag the
// compositor thinks does nothing shows the user a refusal everywhere.
func wlDragActions(a DragAction) C.uint32_t {
	var out C.uint32_t
	if a.Has(DragCopy) {
		out |= 1
	}
	if a.Has(DragMove) {
		out |= 2
	}
	if out == 0 {
		out = 1
	}
	return out
}

// wlDragAction maps one protocol action back.
func wlDragAction(a C.uint32_t) DragAction {
	switch a {
	case 1:
		return DragCopy
	case 2:
		return DragMove
	}
	return DragNone
}

// wlDragWriteTimeout is how long a target has to take the data before the
// transfer is given up on.
const wlDragWriteTimeout = 2 * time.Second

// StartDrag implements [DragSurface]. The compositor carries the drag
// from here: it moves the icon, tells the target, and reports back
// through the source's own listener.
func (s *wlSurface) StartDrag(p DragPayload) bool {
	c := s.conn
	if c == nil || c.dpy == nil || c.dataMan == nil || c.dataDev == nil || len(p.Types) == 0 {
		return false
	}
	if c.drag.src != nil {
		return false
	}
	// start_drag needs the serial of the press it came from; without one
	// the compositor refuses the drag as not user-driven.
	if c.pressSerial == 0 {
		return false
	}
	if p.Actions == DragNone {
		p.Actions = DragCopy
	}
	src := C.ui_wd_source(c.dataMan, C.uintptr_t(c.id))
	if src == nil {
		return false
	}
	for _, t := range p.Types {
		ct := C.CString(t)
		C.ui_wd_offer(src, ct)
		C.free(unsafe.Pointer(ct))
	}
	C.ui_wd_set_actions(src, wlDragActions(p.Actions))
	c.drag = wlDrag{src: src, payload: p, surf: s.id}
	c.dragMakeIcon(s, p)
	C.ui_wd_start(c.dataDev, src, s.surf, c.drag.icon, C.uint32_t(c.pressSerial))
	C.ui_wd_flush(c.dpy)
	// Hold the connection open the way the clipboard does: the data is
	// sent from a callback, and a loop that stopped dispatching would
	// leave the target waiting on a pipe that never fills.
	c.clipKeep = true
	return true
}

// CancelDrag implements [DragSurface]. Dropping the source is how a
// client withdraws a drag: the compositor cancels it and tells every
// target.
func (s *wlSurface) CancelDrag() {
	if s.conn != nil {
		s.conn.dragEnd(DragNone)
	}
}

// Dragging implements [DragSurface].
func (s *wlSurface) Dragging() bool { return s.conn != nil && s.conn.drag.src != nil }

// dragMakeIcon builds the surface the compositor moves with the pointer.
// start_drag gives it the drag-icon role; a nil one is legal and means
// the drag shows only a cursor.
func (c *wlConn) dragMakeIcon(s *wlSurface, p DragPayload) {
	img := p.Icon
	if img == nil || img.Width < 1 || img.Height < 1 || c.compositor == nil || c.shm == nil {
		return
	}
	w, h := img.Width, img.Height
	stride := w * 4
	size := stride * h
	var mem unsafe.Pointer
	fd := int(C.ui_wd_memfd(C.size_t(size), &mem))
	if fd < 0 || mem == nil {
		return
	}
	buf := C.ui_wd_buffer(c.shm, C.int(fd), C.int(w), C.int(h), C.int(stride), C.size_t(size))
	C.ui_wd_close(C.int(fd))
	if buf == nil {
		C.ui_wd_munmap(mem, C.size_t(size))
		return
	}
	// The picture is premultiplied RGBA and the buffer premultiplied
	// ARGB8888, which is the same bytes with red and blue swapped.
	copyImageRect(unsafe.Slice((*byte)(mem), size), stride, img,
		paintengine2d.XYWH(0, 0, float32(w), float32(h)), true, false)
	surf := C.ui_wd_surface(c.compositor)
	if surf == nil {
		C.ui_wd_buf_destroy(buf)
		C.ui_wd_munmap(mem, C.size_t(size))
		return
	}
	// The picture was drawn at the window's scale, so the buffer covers
	// that many logical pixels fewer. A fractional scale has no integer
	// to say here; the icon then draws a little large, which is far
	// better than a scaled-down blur.
	if s.bufScale > 1 {
		C.ui_wd_scale(surf, C.int(s.bufScale))
	}
	// The hotspot is where in the picture the pointer sits, so the icon
	// hangs back from the pointer by exactly that much.
	C.ui_wd_attach(surf, buf, C.int(-int(p.Hotspot.X)), C.int(-int(p.Hotspot.Y)))
	C.ui_wd_damage(surf, C.int(w), C.int(h))
	C.ui_wd_commit(surf)
	c.drag.icon, c.drag.iconBuf, c.drag.iconMem, c.drag.iconSize = surf, buf, mem, size
}

// dragFreeIcon lets the icon surface and its buffer go.
func (c *wlConn) dragFreeIcon() {
	if c.drag.icon != nil {
		C.ui_wd_surface_destroy(c.drag.icon)
		c.drag.icon = nil
	}
	if c.drag.iconBuf != nil {
		C.ui_wd_buf_destroy(c.drag.iconBuf)
		c.drag.iconBuf = nil
	}
	if c.drag.iconMem != nil {
		C.ui_wd_munmap(c.drag.iconMem, C.size_t(c.drag.iconSize))
		c.drag.iconMem = nil
	}
}

// dragEnd stops the drag and tells the window it started in what became
// of it. Every way a drag can finish comes through here exactly once.
func (c *wlConn) dragEnd(action DragAction) {
	if c.drag.src == nil {
		return
	}
	surf := c.drag.surf
	c.dragFreeIcon()
	C.ui_wd_source_destroy(c.drag.src)
	c.drag = wlDrag{}
	c.clipKeepLocked()
	if s := wlSurfaces[surf]; s != nil {
		s.push(Event{Kind: EventDragEnd, Action: action})
	}
}

//export uitkWlDragTarget
func uitkWlDragTarget(id C.uintptr_t, mime *C.char) {
	// The target accepting (or dropping) a type. Nothing here needs it:
	// the action event is what the source reports, and the data is asked
	// for by type when it is wanted.
	_ = id
	_ = mime
}

//export uitkWlDragSend
func uitkWlDragSend(id C.uintptr_t, mime *C.char, fd C.int) {
	c := wlConnBy(id)
	if c == nil || c.drag.src == nil || c.drag.payload.Data == nil {
		C.ui_wd_close(fd)
		return
	}
	m := C.GoString(mime)
	offered := false
	for _, t := range c.drag.payload.Types {
		if t == m {
			offered = true
			break
		}
	}
	if !offered {
		C.ui_wd_close(fd)
		return
	}
	data, ok := c.drag.payload.Data(m)
	wlSendDrag(fd, data, ok)
}

// wlSendDrag writes the drag's data down the target's pipe and always
// closes it: a pipe left open is a target that waits for ever.
func wlSendDrag(fd C.int, data []byte, ok bool) {
	defer C.ui_wd_close(fd)
	if !ok || len(data) == 0 {
		return
	}
	C.ui_wd_nonblock(fd)
	C.ui_wd_write_all(fd, (*C.char)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		C.int(wlDragWriteTimeout/time.Millisecond))
}

//export uitkWlDragAction
func uitkWlDragAction(id C.uintptr_t, action C.uint32_t) {
	c := wlConnBy(id)
	if c == nil || c.drag.src == nil {
		return
	}
	c.drag.action = wlDragAction(action)
}

//export uitkWlDragDropped
func uitkWlDragDropped(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil || c.drag.src == nil {
		return
	}
	// The button is up and the target has the offer. The icon's job is
	// over; the source stays alive until dnd_finished, because the
	// target may only now ask for the data.
	c.drag.dropped = true
	c.dragFreeIcon()
	if c.dpy != nil {
		C.ui_wd_flush(c.dpy)
	}
}

//export uitkWlDragFinished
func uitkWlDragFinished(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil || c.drag.src == nil {
		return
	}
	c.dragEnd(c.drag.action)
}

//export uitkWlDragCancelled
func uitkWlDragCancelled(id C.uintptr_t) {
	c := wlConnBy(id)
	if c == nil || c.drag.src == nil {
		return
	}
	// cancelled after a drop is the protocol saying the source may go:
	// the compositor sends it once the data has been taken. Before a
	// drop it is the real thing — Escape, or no target would have it.
	if c.drag.dropped {
		c.dragEnd(c.drag.action)
		return
	}
	c.dragEnd(DragNone)
}

// ---- the target half's action negotiation --------------------------------------

// waylandDndActions maps the toolkit's actions to
// wl_data_device_manager.dnd_action, which is what both halves of the
// Wayland protocol speak.
func waylandDndActions(a DragAction) C.uint32_t { return wlDragActions(a) }

// dndEvent is one event about the drag over a window, carrying what its
// source allows and the action the compositor has settled on. A source
// too old for actions (version 2) offers none, which the window reads as
// a copy.
func (c *wlConn) dndEvent(kind EventKind, mimes []string) Event {
	acts := c.offerActs[c.dndOffer]
	return Event{
		Kind:    kind,
		Pos:     c.dndPos,
		Mimes:   mimes,
		Actions: acts.source,
		Action:  acts.chosen,
	}
}

//export uitkWlOfferActions
func uitkWlOfferActions(id C.uintptr_t, offer *C.struct_wl_data_offer, actions, chosen C.uint32_t) {
	c := wlConnBy(id)
	if c == nil || offer == nil {
		return
	}
	// Kept per offer, not per connection: source_actions arrives before
	// the enter that makes an offer the current drag, so a value on the
	// connection would be wiped by the drag it belongs to.
	if c.offerActs == nil {
		c.offerActs = map[*C.struct_wl_data_offer]wlOfferAction{}
	}
	got := c.offerActs[offer]
	// 0xffffffff means "this event did not carry that half": the two
	// arrive separately and each says only its own.
	if actions != 0xffffffff {
		got.source = wlOfferActions(actions)
	}
	if chosen != 0xffffffff {
		got.chosen = wlDragAction(chosen)
	}
	c.offerActs[offer] = got
}

// wlOfferAction is what one offer's source allows and what the
// compositor has settled on for it.
type wlOfferAction struct{ source, chosen DragAction }

// wlOfferActions maps a dnd_action bit field back to a set of actions.
func wlOfferActions(a C.uint32_t) DragAction {
	var out DragAction
	if a&1 != 0 {
		out |= DragCopy
	}
	if a&2 != 0 {
		out |= DragMove
	}
	return out
}

// AcceptDrag implements [DropNegotiator]: the window says what it would
// do with the drag over it, and the compositor tells the source. An
// offer accepted for no type is what makes the source show a refusal.
func (s *wlSurface) AcceptDrag(mime string, allowed, a DragAction) {
	c := s.conn
	if c == nil || c.dndOffer == nil {
		return
	}
	var cm *C.char
	if mime != "" && a != DragNone {
		cm = C.CString(mime)
		defer C.free(unsafe.Pointer(cm))
	}
	C.ui_wd_accept(c.dndOffer, C.uint32_t(c.dndSerial), cm)
	if a == DragNone {
		// No action at all: wl_data_offer.set_actions with an empty mask
		// is how a target says "not here".
		C.ui_wd_offer_actions(c.dndOffer, 0, 0)
	} else {
		// Everything this target would take, and the one it would take
		// now. Naming only the latter would pin the compositor to it,
		// and the user's Shift for a move could never change anything.
		C.ui_wd_offer_actions(c.dndOffer, wlDragActions(allowed|a), wlDragActions(a))
	}
	if c.dpy != nil {
		C.ui_wd_flush(c.dpy)
	}
}
