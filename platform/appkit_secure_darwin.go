//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Carbon
#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>

// Secure event input is a process-wide mode, not a window's: macOS has
// one keyboard and one front process, and turning it on takes the keys
// away from every other process on the machine — including the ones with
// accessibility permission, which is the hole every other platform
// leaves open. The menu bar shows the indicator while it is on.
//
// It is reference counted by the system, so an unbalanced enable leaves
// the whole desktop in secure input until the process exits, and no
// other application can undo it. uitk_ak_secure_input keeps the count
// here so the toolkit can never hand out an unbalanced pair, whatever
// the application does with SetSecureInput.
static int uitk_secure_depth = 0;

static int uitk_ak_secure_input(int on) {
	if (on) {
		if (uitk_secure_depth == 0) {
			OSStatus e = EnableSecureEventInput();
			if (e != noErr) return 0;
		}
		uitk_secure_depth++;
		return 1;
	}
	if (uitk_secure_depth == 0) return 1;
	uitk_secure_depth--;
	if (uitk_secure_depth == 0) DisableSecureEventInput();
	return 1;
}

// NSWindowSharingNone keeps the window out of screenshots, screen
// recordings and screen sharing: the window server composites it to the
// display and leaves it out of every capture of that display.
static void uitk_ak_set_sharing(void *w, int exclude) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if (!win) return;
		[win setSharingType:exclude ? NSWindowSharingNone : NSWindowSharingReadOnly];
	}
}
*/
import "C"

// SetSecureInput turns macOS secure event input on or off
// ([SecureInputSurface]).
//
// It is the real thing, and it is the only one of the four platforms
// where it is: the keyboard is taken from every other process while it
// is on, and the menu bar says so. It is process-wide rather than this
// window's, because macOS has one keyboard and one front process; the
// count is kept on the C side so the pair is always balanced, since an
// unbalanced enable would leave the whole desktop in secure input until
// this process exits and no other application could undo it.
func (s *akSurface) SetSecureInput(on bool) bool {
	if s == nil {
		return false
	}
	if on == s.secureOn {
		return true
	}
	if C.uitk_ak_secure_input(cbool(on)) == 0 {
		return false
	}
	s.secureOn = on
	return true
}

// SetExcludeFromCapture keeps this window out of screenshots and screen
// shares ([CaptureExcludeSurface]).
func (s *akSurface) SetExcludeFromCapture(on bool) bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	C.uitk_ak_set_sharing(s.win, cbool(on))
	return true
}
