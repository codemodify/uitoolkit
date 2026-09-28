//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <dispatch/dispatch.h>
#import <AppKit/AppKit.h>

// The corner resize pointer is the one shape AppKit does not publish.
// The one the window frame itself draws is private, and the public
// +[NSCursor frameResizeCursorFromPosition:inDirection:] is too new to
// rely on — it is missing from command line tools that are otherwise
// current, so calling it would make the toolkit fail to build on Macs
// that are perfectly up to date. The arrow stands in.
//
// It costs little in practice. macOS resizes windows from its own
// invisible band (FrameSystemResizeBand), so a diagonal is only ever
// asked for by something the toolkit draws inside its own window, and
// a splitter is horizontal or vertical.

// nscursorFor maps the numbers darwinCursorKind produces (cursor.go)
// to an NSCursor. Kept apart from setting one so the window backend
// can hand the same shape back from -[NSView cursorUpdate:].
NSCursor *uitk_nscursor_for(int kind) {
	switch (kind) {
	case 1:  return [NSCursor resizeLeftRightCursor];
	case 2:  return [NSCursor resizeUpDownCursor];
	case 3:  return [NSCursor IBeamCursor];
	case 4:  return [NSCursor openHandCursor];
	case 5:  return [NSCursor closedHandCursor];
	case 6:  return [NSCursor pointingHandCursor];
	case 7:  return [NSCursor dragCopyCursor];
	case 8:  return [NSCursor dragLinkCursor];
	case 9:  return [NSCursor operationNotAllowedCursor];
	case 10: return [NSCursor arrowCursor]; // NWSE: see above
	case 11: return [NSCursor arrowCursor]; // NESW: see above
	default: return [NSCursor arrowCursor];
	}
}

// NSCursor, like the rest of AppKit, is main-thread only.
static void ui_set_nscursor(int kind) {
	void (^apply)(void) = ^{
		@autoreleasepool { [uitk_nscursor_for(kind) set]; }
	};
	if ([NSThread isMainThread]) {
		apply();
		return;
	}
	dispatch_async(dispatch_get_main_queue(), apply);
}
*/
import "C"

// applySystemCursor sets the pointer via NSCursor.
func applySystemCursor(c Cursor) {
	C.ui_set_nscursor(C.int(darwinCursorKind(c)))
}
