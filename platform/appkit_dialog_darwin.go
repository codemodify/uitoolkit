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
	C.uitk_ak_set_role(s.win, cbool(r == RoleDialog))
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
