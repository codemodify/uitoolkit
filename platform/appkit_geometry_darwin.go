//go:build darwin && cgo

package platform

/*
#include "appkit_darwin.h"
*/
import "C"

// [WindowGeometry] for AppKit.
//
// macOS is the easy platform for this. It places windows where it is
// asked, it says where they ended up, and there is no compositor policy
// in the way — so unlike Wayland, where Position can only ever answer
// "not told", every capability here is real.
//
// The units are the toolkit's: logical pixels of the desktop, y down
// from the top-left of the main screen. AppKit measures y *up* from the
// bottom-left, and the flip happens in appkit_darwin.m so that only one
// file knows which way is up.

func (s *akSurface) GeometryCaps() GeometryCaps {
	if s == nil || s.win == nil || s.Closed() {
		return 0
	}
	return GeometryMove | GeometryPosition | GeometryScreenPlace |
		GeometryVisibility | GeometrySizeLimits
}

func (s *akSurface) Move(x, y int) bool {
	if !s.GeometryCaps().Has(GeometryMove) {
		return false
	}
	C.uitk_ak_move(s.win, C.int(x), C.int(y))
	return true
}

// PlaceAtScreen is Move: macOS has no separate way to put a window at a
// spot on the desktop, and needs none — an ordinary move is honoured.
func (s *akSurface) PlaceAtScreen(x, y int) bool { return s.Move(x, y) }

func (s *akSurface) Position() (int, int, bool) {
	if !s.GeometryCaps().Has(GeometryPosition) {
		return 0, 0, false
	}
	var x, y C.int
	if C.uitk_ak_position(s.win, &x, &y) == 0 {
		return 0, 0, false
	}
	return int(x), int(y), true
}

func (s *akSurface) Show() bool {
	if !s.GeometryCaps().Has(GeometryVisibility) {
		return false
	}
	C.uitk_ak_order_front(s.win, 1)
	return true
}

func (s *akSurface) Hide() bool {
	if !s.GeometryCaps().Has(GeometryVisibility) {
		return false
	}
	C.uitk_ak_order_out(s.win)
	return true
}

func (s *akSurface) Raise() bool {
	if !s.GeometryCaps().Has(GeometryVisibility) {
		return false
	}
	C.uitk_ak_order_front(s.win, 1)
	return true
}

func (s *akSurface) Visible() bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	return C.uitk_ak_visible(s.win) != 0
}

func (s *akSurface) Sizing() Sizing { return s.sizing }

func (s *akSurface) SetSizing(v Sizing) bool {
	if s == nil {
		return false
	}
	if s.sizing != v {
		s.sizing = v
		s.limits = limitsFor(v, s.opts, s.logicalW, s.logicalH)
		s.applyLimits()
	}
	return true
}

func (s *akSurface) SizeLimits() SizeLimits { return s.limits }

// applyLimits tells AppKit the limits, in points, which is what the
// toolkit's logical pixel is here — the scale lives in the backing
// store, not in the geometry.
func (s *akSurface) applyLimits() {
	if s == nil || s.win == nil {
		return
	}
	l := s.limits
	resizable := 0
	if s.sizing != SizingFixed {
		resizable = 1
	}
	C.uitk_ak_set_limits(s.win, C.int(l.MinWidth), C.int(l.MinHeight),
		C.int(l.MaxWidth), C.int(l.MaxHeight), C.int(resizable))
}

var _ WindowGeometry = (*akSurface)(nil)
