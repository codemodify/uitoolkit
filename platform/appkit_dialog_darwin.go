//go:build darwin && cgo

package platform

/*
#include "appkit_darwin.h"
*/
import "C"

// SetWindowRole restyles the window as a dialog, or back
// ([RoleSurface]): the floating level and no minimize button, which is
// what an NSPanel would have given it.
func (s *akSurface) SetWindowRole(r WindowRole) bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	// 0 ordinary, 1 dialog, 2 a satellite panel: the C side reads the second
	// as "not an ordinary window" and the third as a panel's extra behaviour.
	kind := C.int(0)
	switch r {
	case RoleDialog:
		kind = 1
	case RoleUtility:
		kind = 2
	}
	s.role = r
	C.uitk_ak_set_role(s.win, kind)
	return true
}

// SetOwner says which window this one belongs to ([OwnedSurface]).
//
// -addChildWindow:ordered:, which is AppKit's owned window: the child stays
// above its parent, moves with it and is ordered out with it. It is not a
// view-hierarchy parent and does not clip the child, whatever the name
// suggests.
func (s *akSurface) SetOwner(owner Surface) bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	if owner == nil {
		C.uitk_ak_set_owner(s.win, nil)
		s.owner = nil
		return true
	}
	o, ok := owner.(*akSurface)
	if !ok || o == s || o.win == nil || o.Closed() {
		return false
	}
	s.owner = o
	C.uitk_ak_set_owner(s.win, o.win)
	return true
}

// Center puts the window in the middle of the screen it is on
// ([CenterSurface]). NSWindow's own -center, which is a little above the
// middle — where the Mac has always put a dialog, because a box centred
// exactly looks low.
func (s *akSurface) Center() bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	C.uitk_ak_center(s.win)
	return true
}

// Activate brings the window to the front and gives it the keyboard
// ([ActivateSurface]).
func (s *akSurface) Activate() bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	return C.uitk_ak_activate(s.win) != 0
}
