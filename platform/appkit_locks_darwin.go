//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

// uitk_ak_lock_state reads the keyboard's current modifier flags, which
// AppKit will answer at any time — there is no event to wait for.
// Bit 0 is Caps Lock, bit 1 Num Lock.
static unsigned uitk_ak_lock_state(void) {
	NSEventModifierFlags f = [NSEvent modifierFlags];
	unsigned out = 0;
	if (f & NSEventModifierFlagCapsLock) out |= 1;
	if (f & NSEventModifierFlagNumericPad) out |= 2;
	return out;
}
*/
import "C"

// The lock keys on macOS.
//
// NSEvent.modifierFlags is a property of the keyboard, not of an event,
// so it can be read before anything has been typed. Without this a
// passphrase prompt opened with Caps Lock on said nothing of it until the
// first key — which is the one case the warning exists for.

// LockKeys is the keyboard's current lock state ([LockKeysSurface]).
func (s *akSurface) LockKeys() (caps, num bool) {
	if s == nil {
		return false, false
	}
	st := uint32(C.uitk_ak_lock_state())
	return st&1 != 0, st&2 != 0
}
