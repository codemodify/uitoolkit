//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#include <stdint.h>
#include <wayland-client.h>
#include "xdg-shell-client-protocol.h"

extern void uitkWlPopConfigure(uintptr_t sid, int32_t x, int32_t y, int32_t w, int32_t h);
extern void uitkWlPopDone(uintptr_t sid);

static struct xdg_positioner *ui_wl_positioner(struct xdg_wm_base *wm) {
	return wm ? xdg_wm_base_create_positioner(wm) : NULL;
}

// ui_wl_pos_set states a whole placement at once. reactive asks the
// compositor to place the popup again by itself when its parent moves
// (xdg_positioner v3), with the parent's size as it is now.
static void ui_wl_pos_set(struct xdg_positioner *p, int w, int h, int ax, int ay, int aw, int ah,
		uint32_t anchor, uint32_t gravity, uint32_t adjust, int pw, int ph) {
	if (!p) return;
	xdg_positioner_set_size(p, w, h);
	xdg_positioner_set_anchor_rect(p, ax, ay, aw, ah);
	xdg_positioner_set_anchor(p, anchor);
	xdg_positioner_set_gravity(p, gravity);
	xdg_positioner_set_constraint_adjustment(p, adjust);
	if (xdg_positioner_get_version(p) >= 3) {
		xdg_positioner_set_reactive(p);
		if (pw > 0 && ph > 0) xdg_positioner_set_parent_size(p, pw, ph);
	}
}

static void ui_wl_pos_destroy(struct xdg_positioner *p) { if (p) xdg_positioner_destroy(p); }

static void uitk_pop_cfg(void *data, struct xdg_popup *pp, int32_t x, int32_t y, int32_t w, int32_t h) {
	(void)pp;
	uitkWlPopConfigure((uintptr_t)data, x, y, w, h);
}
static void uitk_pop_done(void *data, struct xdg_popup *pp) {
	(void)pp;
	uitkWlPopDone((uintptr_t)data);
}
static void uitk_pop_repositioned(void *data, struct xdg_popup *pp, uint32_t token) {
	(void)data; (void)pp; (void)token;
}
static const struct xdg_popup_listener uitk_pop_listener = {
	.configure = uitk_pop_cfg,
	.popup_done = uitk_pop_done,
	.repositioned = uitk_pop_repositioned,
};

static struct xdg_popup *ui_wl_get_popup(struct xdg_surface *xs, struct xdg_surface *parent,
		struct xdg_positioner *p, uintptr_t sid) {
	if (!xs || !parent || !p) return NULL;
	struct xdg_popup *pp = xdg_surface_get_popup(xs, parent, p);
	if (pp) xdg_popup_add_listener(pp, &uitk_pop_listener, (void*)sid);
	return pp;
}

static void ui_wl_popup_grab(struct xdg_popup *pp, struct wl_seat *seat, uint32_t serial) {
	if (pp && seat) xdg_popup_grab(pp, seat, serial);
}

// ui_wl_popup_reposition moves an open popup (xdg_popup v3); 0 where the
// compositor's xdg_wm_base is older, and the caller reopens it instead.
static int ui_wl_popup_reposition(struct xdg_popup *pp, struct xdg_positioner *p, uint32_t token) {
	if (!pp || !p || xdg_popup_get_version(pp) < 3) return 0;
	xdg_popup_reposition(pp, p, token);
	return 1;
}

static void ui_wl_popup_destroy(struct xdg_popup *pp) { if (pp) xdg_popup_destroy(pp); }
*/
import "C"

import (
	"errors"

	"github.com/codemodify/paintengine2d"
)

// Popups on Wayland: an xdg_popup for every menu, submenu, list and tooltip
// (platform/popup.go has the whole story). The compositor places each one
// from an xdg_positioner — it alone knows where the window is on the
// screen — and flips, slides or shrinks it against the output's edges; a
// menu takes an explicit grab with the serial of the press or the key that
// opened it, so the compositor dismisses it (popup_done) when the user
// clicks outside the application.
//
// Everything that arrives on a popup's wl_surface is handed to its root
// window, translated into the root's device pixels (wlSurface.push), and
// the pointer and the keyboard moving between a window and its own popups
// are not the window losing them: a leave followed by an enter within the
// same family is no event at all (wlConn.enterPointer and its neighbours).

// wlPopup is a wlSurface's popup half.
type wlPopup struct {
	xpop *C.struct_xdg_popup
	// parent is the surface it hangs from, root the top level whose queue
	// its events go to.
	parent, root *wlSurface
	role         PopupRole
	// place is the placement it was last given; placed where the
	// compositor put its visible box, logical pixels relative to the
	// root's, and pend a configure not yet made current by
	// xdg_surface.configure. placedOnce is set by the first.
	place      PopupPlacement
	placed     FrameRect
	pend       FrameRect
	pendSet    bool
	placedOnce bool
	token      uint32
}

var errWlPopup = errors.New("platform: the compositor did not map the popup")

// PopupsSupported reports whether a popup can be opened from s now: it has
// its xdg role and has been configured (PopupOpener).
func (s *wlSurface) PopupsSupported() bool {
	return s != nil && !s.closed && s.conn != nil && s.conn.wm != nil && s.xdg != nil &&
		s.configured && s.surf != nil
}

// OpenPopup opens an xdg_popup from s (PopupOpener). It returns once the
// compositor has placed it, so Placed is its real place.
func (s *wlSurface) OpenPopup(opts PopupOptions) (PopupSurface, error) {
	if !s.PopupsSupported() {
		return nil, errWlPopup
	}
	c, err := wlRetain()
	if err != nil {
		return nil, err
	}
	root := s.popRoot()
	pl := opts.Placement
	w, h := max(pl.W, 1), max(pl.H, 1)
	p := &wlSurface{
		conn: c, title: s.title, appID: s.appID,
		logicalW: w, logicalH: h, wantW: w, wantH: h,
		bufScale: root.bufScale, frac: root.frac,
		outs:     newOutputSet(),
		wantDeco: DecorationsNone, decoMode: DecorationsNone,
		frame: opts.Frame,
		pop:   &wlPopup{parent: s, root: root, role: opts.Role, place: pl},
	}
	bw, bh := p.bufferWH()
	p.img = paintengine2d.NewImage(bw, bh)
	p.bufW, p.bufH = bw, bh
	wlMu.Lock()
	wlNextSurf++
	p.id = wlNextSurf
	wlSurfaces[p.id] = p
	if !c.makePopupSurfaceLocked(p) {
		wlMu.Unlock()
		_ = p.Close()
		return nil, errWlPopup
	}
	pos := p.positioner(pl)
	p.pop.xpop = C.ui_wl_get_popup(p.xdg, s.xdg, pos, C.uintptr_t(p.id))
	C.ui_wl_pos_destroy(pos)
	if p.pop.xpop == nil {
		wlMu.Unlock()
		_ = p.Close()
		return nil, errWlPopup
	}
	if opts.Role == PopupRoleMenu && c.seat != nil {
		// A grab needs the serial of the press or the key that opened the
		// menu; the compositor refuses one without (popup_done at once).
		C.ui_wl_popup_grab(p.pop.xpop, c.seat, C.uint32_t(c.grabSerial))
	}
	// The window geometry goes with the first commit: the positioner
	// places the visible box, not the shadow's margin around it.
	sw, sh := p.surfaceLogical()
	p.applyFrameLocked(sw, sh)
	p.commitSurface()
	wlMu.Unlock()
	for i := 0; i < 3 && !p.configured && !p.closed; i++ {
		c.roundtrip()
	}
	if !p.configured || p.closed {
		_ = p.Close()
		return nil, errWlPopup
	}
	s.kids = append(s.kids, p)
	return p, nil
}

// popRoot is the top level s belongs to (s itself for a top level).
func (s *wlSurface) popRoot() *wlSurface {
	if s.pop != nil && s.pop.root != nil {
		return s.pop.root
	}
	return s
}

// family is the id of the top level s belongs to: input moving between a
// window and its own popups stays within one family.
func (s *wlSurface) family() int { return s.popRoot().id }

// placedAt is where s's visible box is relative to its root's (zero for a
// top level).
func (s *wlSurface) placedAt() FrameRect {
	if s.pop == nil {
		return FrameRect{}
	}
	return s.pop.placed
}

// positioner is pl as an xdg_positioner for s: the anchor moved from the
// root's coordinates into the parent's, which is what a nested popup's
// positioner speaks.
func (s *wlSurface) positioner(pl PopupPlacement) *C.struct_xdg_positioner {
	pos := C.ui_wl_positioner(s.conn.wm)
	if pos == nil {
		return nil
	}
	par := s.pop.parent
	off := par.placedAt()
	a := pl.Anchor
	pw, ph := par.logicalW, par.logicalH
	C.ui_wl_pos_set(pos, C.int(max(pl.W, 1)), C.int(max(pl.H, 1)),
		C.int(a.X-off.X), C.int(a.Y-off.Y), C.int(max(a.W, 1)), C.int(max(a.H, 1)),
		C.uint32_t(xdgAnchor(pl.AnchorEdge)), C.uint32_t(xdgAnchor(pl.Gravity)),
		C.uint32_t(pl.Adjust), C.int(pw), C.int(ph))
	return pos
}

// xdgAnchor is e as xdg_positioner's anchor and gravity enum: none 0, top
// 1, bottom 2, left 3, right 4, then the corners top_left 5, bottom_left 6,
// top_right 7, bottom_right 8.
func xdgAnchor(e Edges) uint32 {
	switch e & (EdgeTop | EdgeBottom | EdgeLeft | EdgeRight) {
	case EdgeTop:
		return 1
	case EdgeBottom:
		return 2
	case EdgeLeft:
		return 3
	case EdgeRight:
		return 4
	case EdgeTop | EdgeLeft:
		return 5
	case EdgeBottom | EdgeLeft:
		return 6
	case EdgeTop | EdgeRight:
		return 7
	case EdgeBottom | EdgeRight:
		return 8
	}
	return 0
}

//export uitkWlPopConfigure
func uitkWlPopConfigure(sid C.uintptr_t, x, y, w, h C.int32_t) {
	s := wlSurfBy(sid)
	if s == nil || s.pop == nil {
		return
	}
	off := s.pop.parent.placedAt()
	s.pop.pend = FrameRect{X: off.X + int(x), Y: off.Y + int(y), W: int(w), H: int(h)}
	s.pop.pendSet = true
}

// applyPopupConfigure makes the popup's last configure current: its place,
// and its size where the compositor shrank it. The buffer follows at once,
// so the app's next paint is at the new size; the root hears about any
// place after the first as EventPopupPlaced.
func (s *wlSurface) applyPopupConfigure() {
	pp := s.pop
	if pp == nil || !pp.pendSet {
		return
	}
	pp.pendSet = false
	moved := pp.pend != pp.placed
	pp.placed = pp.pend
	if pp.placed.W > 0 && pp.placed.H > 0 {
		s.logicalW, s.logicalH = pp.placed.W, pp.placed.H
		s.wantW, s.wantH = pp.placed.W, pp.placed.H
		if bw, bh := s.bufferWH(); bw != s.bufW || bh != s.bufH {
			s.setBuffer(bw, bh)
			s.frameSent = false
		}
	}
	if moved && pp.placedOnce {
		r := pp.root
		r.queue = append(r.queue, Event{Kind: EventPopupPlaced, Popup: s})
	}
	pp.placedOnce = true
}

//export uitkWlPopDone
func uitkWlPopDone(sid C.uintptr_t) {
	s := wlSurfBy(sid)
	if s == nil || s.pop == nil {
		return
	}
	if !s.configured {
		// Refused before it was ever shown: OpenPopup reports it.
		s.closed = true
		return
	}
	r := s.pop.root
	r.queue = append(r.queue, Event{Kind: EventPopupDone, Popup: s})
}

// Reposition moves the popup (xdg_popup.reposition) and waits for the
// compositor's answer, so Placed is the new place when it returns
// (PopupSurface). false on a compositor too old to move a popup.
func (s *wlSurface) Reposition(pl PopupPlacement) bool {
	if s == nil || s.pop == nil || s.closed || s.pop.xpop == nil {
		return false
	}
	pos := s.positioner(pl)
	if pos == nil {
		return false
	}
	s.pop.token++
	ok := C.ui_wl_popup_reposition(s.pop.xpop, pos, C.uint32_t(s.pop.token)) != 0
	C.ui_wl_pos_destroy(pos)
	if !ok {
		return false
	}
	s.pop.place = pl
	if pl.W != s.logicalW || pl.H != s.logicalH {
		// Until the answer: the size asked for, so a compositor that
		// only moves it leaves it the new size.
		s.wantW, s.wantH = pl.W, pl.H
	}
	s.conn.roundtrip()
	return true
}

// Placed is where the compositor put the popup's visible box (PopupSurface).
func (s *wlSurface) Placed() FrameRect { return s.placedAt() }

// Origin is the popup's buffer origin in its root's buffer, device pixels
// (PopupSurface).
func (s *wlSurface) Origin() paintengine2d.Point {
	if s == nil || s.pop == nil {
		return paintengine2d.Point{}
	}
	r := s.pop.root
	m := r.frame.Margin
	return PopupOrigin(paintengine2d.Pt(float32(m.Left), float32(m.Top)), s.pop.placed, s.frame.Margin, r.deviceScale())
}

// Root is the top level the popup belongs to (PopupSurface).
func (s *wlSurface) Root() Surface {
	if s == nil {
		return nil
	}
	return s.popRoot()
}

// closePopupLocked takes a popup's protocol objects down: its own popups
// first (a compositor insists the topmost goes first), then its xdg_popup,
// and hands the keyboard back to its root, which the compositor gives it
// again anyway.
func (s *wlSurface) closePopupLocked() {
	for i := len(s.kids) - 1; i >= 0; i-- {
		_ = s.kids[i].Close()
	}
	s.kids = nil
	pp := s.pop
	if pp == nil {
		return
	}
	if par := pp.parent; par != nil {
		for i, k := range par.kids {
			if k == s {
				par.kids = append(par.kids[:i:i], par.kids[i+1:]...)
				break
			}
		}
	}
	if pp.xpop != nil {
		C.ui_wl_popup_destroy(pp.xpop)
		pp.xpop = nil
	}
	if c := s.conn; c != nil {
		if c.keySurf == s.id {
			c.keySurf = pp.root.id
		}
		if c.ptrSurf == s.id {
			c.ptrSurf = 0
		}
	}
}

// ---- input within a family ------------------------------------------------

// enterPointer notes the pointer entering s: a pointer that left s's own
// family for it (a window for its menu, a menu for its submenu) never left
// at all.
func (c *wlConn) enterPointer(s *wlSurface) {
	if c.ptrLeft != 0 && c.ptrLeft != s.family() {
		if r := wlSurfaces[c.ptrLeft]; r != nil {
			r.queue = append(r.queue, Event{Kind: EventPointerLeave, Mods: c.mods})
		}
	}
	c.ptrLeft = 0
}

// leavePointer notes the pointer leaving s; the window hears of it only if
// no surface of its family is entered in the same burst (flushLeaves).
func (c *wlConn) leavePointer(s *wlSurface) {
	if s != nil {
		c.ptrLeft = s.family()
	}
}

// enterKeyboard notes keyboard focus arriving on s and reports whether the
// family is new to it (the window hears EventFocusIn only then).
func (c *wlConn) enterKeyboard(s *wlSurface) bool {
	fam := s.family()
	if c.keyLeft && c.keyRoot != fam {
		c.focusOutLocked()
	}
	c.keyLeft = false
	if c.keyRoot == fam {
		return false
	}
	c.keyRoot = fam
	return true
}

// flushLeaves tells a window the pointer or the keyboard left its family,
// once a burst of events has shown nothing of the family was entered
// instead.
func (c *wlConn) flushLeaves() {
	if c.ptrLeft != 0 {
		if r := wlSurfaces[c.ptrLeft]; r != nil {
			r.queue = append(r.queue, Event{Kind: EventPointerLeave, Mods: c.mods})
		}
		c.ptrLeft = 0
	}
	if c.keyLeft {
		c.focusOutLocked()
		c.keyLeft = false
	}
}

// focusOutLocked is the keyboard leaving the family it was in.
func (c *wlConn) focusOutLocked() {
	if r := wlSurfaces[c.keyRoot]; r != nil {
		r.queue = append(r.queue, Event{Kind: EventFocusOut}, Event{Kind: EventIMECancel})
	}
	c.keyRoot = 0
}

// PopupWorkArea: a Wayland client never learns where its window is, so it
// cannot say (PopupWorkArea); the compositor places popups on its own.
func (s *wlSurface) PopupWorkArea() (FrameRect, bool) { return FrameRect{}, false }

var (
	_ PopupOpener  = (*wlSurface)(nil)
	_ PopupSurface = (*wlSurface)(nil)
)
