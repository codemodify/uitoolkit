//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// Popups on AppKit: menus, combo lists and tooltips as windows of their
// own rather than rectangles painted inside the window.
//
// Without this the toolkit falls back to drawing a popup inside its
// window, which works until the popup is taller than the room under it
// — a combo box near the bottom of a window, a menu on a short window —
// and then it is clipped or moved somewhere it does not belong. That
// fallback is what macOS and Windows both had.
//
// **The placement is the toolkit's, not the platform's.** Wayland hands
// the whole problem to the compositor (xdg_positioner); macOS has no
// such thing, so this takes the X11 backend's path instead and calls
// [SolvePopup], which does the flipping, sliding and shrinking against
// a work area. One consequence is worth stating: the answer is computed
// here, so a popup lands in the same place on macOS as it does under a
// tested headless run, and the same function is under both.
//
// **A popup is non-activating.** A menu must not take key away from the
// window it belongs to, and does not need to: the contract is that a
// popup hands every input event to its root, so the keyboard reaching
// the root window directly is the intended destination rather than a
// gap. What the popup's own view sees is the pointer, and that is
// translated by [Origin] and pushed onto the root's queue.

// akPopup is what makes an [akSurface] a popup. A popup is a surface
// like any other — it has a buffer and presents it — so rather than a
// second surface type this is a field on the one there is, the way the
// offscreen backend does it.
type akPopup struct {
	root   *akSurface
	parent *akSurface
	opts   PopupOptions
	placed FrameRect
}

var errAkNoPopups = errors.New("appkit: this window cannot open popups")

// popRoot is the top-level window a surface belongs to: itself, or the
// root of the popup chain it hangs from.
func (s *akSurface) popRoot() *akSurface {
	for s != nil && s.pop != nil && s.pop.root != nil && s.pop.root != s {
		s = s.pop.root
	}
	return s
}

// PopupsSupported reports whether this window can open popups
// ([PopupOpener]). AppKit always can.
func (s *akSurface) PopupsSupported() bool {
	return s != nil && s.win != nil && !s.Closed()
}

// OpenPopup opens a popup from s ([PopupOpener]).
func (s *akSurface) OpenPopup(opts PopupOptions) (PopupSurface, error) {
	if !s.PopupsSupported() {
		return nil, errAkNoPopups
	}
	root := s.popRoot()
	p := &akSurface{
		title:    "",
		opts:     WindowOptions{Width: max(opts.Placement.W, 1), Height: max(opts.Placement.H, 1), Popup: true},
		scale:    root.scale,
		logicalW: max(opts.Placement.W, 1),
		logicalH: max(opts.Placement.H, 1),
		decor:    DecorationsNone,
		frame:    opts.Frame,
		pop:      &akPopup{root: root, parent: s, opts: opts},
	}
	p.resizeBuffer(DevicePixels(p.logicalW, p.scale), DevicePixels(p.logicalH, p.scale))

	akMu.Lock()
	akNextID++
	p.id = akNextID
	akBySID[p.id] = p
	akMu.Unlock()

	p.pop.placed = SolvePopup(opts.Placement, s.workArea())
	x, y := root.screenOf(p.pop.placed)
	tooltip := 0
	if opts.Role == PopupRoleTooltip {
		tooltip = 1
	}
	p.win = C.uitk_ak_popup_new(C.uintptr_t(p.id), s.win,
		C.int(x), C.int(y), C.int(p.pop.placed.W), C.int(p.pop.placed.H), C.int(tooltip))
	if p.win == nil {
		akMu.Lock()
		delete(akBySID, p.id)
		akMu.Unlock()
		return nil, errAkNoPopups
	}
	p.sizeToPlaced()

	s.kids = append(s.kids, p)
	root.push(Event{Kind: EventPopupPlaced, Popup: p})
	return p, nil
}

// Reposition places the popup afresh ([PopupSurface]).
func (s *akSurface) Reposition(pl PopupPlacement) bool {
	if s == nil || s.pop == nil || s.win == nil || s.Closed() {
		return false
	}
	s.pop.opts.Placement = pl
	s.pop.placed = SolvePopup(pl, s.pop.parent.workArea())
	x, y := s.pop.root.screenOf(s.pop.placed)
	C.uitk_ak_popup_move(s.win, C.int(x), C.int(y),
		C.int(s.pop.placed.W), C.int(s.pop.placed.H))
	s.sizeToPlaced()
	return true
}

// sizeToPlaced grows the buffer to whatever the placement came out as:
// SolvePopup may have shrunk the popup to fit the work area, and the
// buffer is the answer's size rather than the request's.
func (s *akSurface) sizeToPlaced() {
	s.logicalW, s.logicalH = max(s.pop.placed.W, 1), max(s.pop.placed.H, 1)
	w, h := DevicePixels(s.logicalW, s.scale), DevicePixels(s.logicalH, s.scale)
	if s.bufW != w || s.bufH != h {
		s.resizeBuffer(w, h)
		s.push(Event{Kind: EventResize, Width: s.logicalW, Height: s.logicalH})
	}
}

// Placed is where the popup's visible box ended up ([PopupSurface]),
// in logical pixels relative to the root window's visible box.
func (s *akSurface) Placed() FrameRect {
	if s == nil || s.pop == nil {
		return FrameRect{}
	}
	return s.pop.placed
}

// Origin is where the popup's buffer (0,0) sits in the root window's
// buffer ([PopupSurface]).
//
// The root's visible box starts at its own frame margin — which on
// macOS is always zero, because AppKit draws the shadow itself
// (FrameSystemShadow) and the toolkit reserves nothing for one — so
// this is the placement scaled, less the popup's own margin, which is
// zero here for the same reason. It goes through [PopupOrigin] anyway
// rather than being written out: the rounding is the part that matters,
// and it is the same rounding on every backend.
func (s *akSurface) Origin() paintengine2d.Point {
	if s == nil || s.pop == nil {
		return paintengine2d.Point{}
	}
	root := s.pop.root
	m := root.frame.Margin
	win := paintengine2d.Pt(float32(m.Left), float32(m.Top))
	return PopupOrigin(win, s.pop.placed, s.frame.Margin, root.scale)
}

// Root is the top-level window the popup belongs to ([PopupSurface]).
func (s *akSurface) Root() Surface {
	if s == nil {
		return nil
	}
	return s.popRoot()
}

// PopupWorkArea is where this window's popups may go
// ([PopupWorkArea]): the screen less the menu bar and the Dock, in
// logical pixels relative to the window's visible box.
//
// macOS can answer this and Wayland cannot, which is the whole reason
// the capability is optional: a Wayland client never learns where its
// own window is, so there the compositor places popups itself.
func (s *akSurface) PopupWorkArea() (FrameRect, bool) {
	if s == nil || s.win == nil || s.Closed() {
		return FrameRect{}, false
	}
	return s.workArea(), true
}

// workArea is the usable screen in the window's own coordinates.
func (s *akSurface) workArea() FrameRect {
	var x, y, w, h C.int
	C.uitk_ak_work_area(s.win, &x, &y, &w, &h)
	if w <= 0 || h <= 0 {
		return FrameRect{}
	}
	ox, oy := s.contentOrigin()
	return FrameRect{X: int(x) - ox, Y: int(y) - oy, W: int(w), H: int(h)}
}

// contentOrigin is where this window's content area begins on screen,
// in points, y down — the origin a popup's placement is relative to.
func (s *akSurface) contentOrigin() (int, int) {
	var x, y C.int
	C.uitk_ak_content_origin(s.win, &x, &y)
	return int(x), int(y)
}

// screenOf turns a placement — logical pixels relative to this window's
// visible box — into the toolkit's screen coordinates.
func (s *akSurface) screenOf(r FrameRect) (int, int) {
	ox, oy := s.contentOrigin()
	return ox + r.X, oy + r.Y
}

// closePopup takes the popup down and unhooks it from its parent.
func (s *akSurface) closePopup() {
	if s == nil || s.pop == nil {
		return
	}
	var parent unsafe.Pointer
	if s.pop.parent != nil {
		parent = s.pop.parent.win
	}
	if s.win != nil {
		C.uitk_ak_popup_close(parent, s.win)
		s.win = nil
	}
	if p := s.pop.parent; p != nil {
		for i, k := range p.kids {
			if k == s {
				p.kids = append(p.kids[:i], p.kids[i+1:]...)
				break
			}
		}
	}
	if root := s.pop.root; root != nil {
		root.push(Event{Kind: EventPopupDone, Popup: s})
	}
}

// closeKids takes down every popup open from s, deepest first: a
// submenu belongs to its menu and must not outlive it.
func (s *akSurface) closeKids() {
	for len(s.kids) > 0 {
		k := s.kids[len(s.kids)-1]
		k.Close()
	}
}

var (
	_ PopupSurface  = (*akSurface)(nil)
	_ PopupOpener   = (*akSurface)(nil)
	_ PopupWorkArea = (*akSurface)(nil)
)
