//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>

static void ui_set_nscursor(int kind) {
	@autoreleasepool {
		NSCursor *c = [NSCursor arrowCursor];
		if (kind == 1) {
			c = [NSCursor resizeLeftRightCursor];
		} else if (kind == 2) {
			c = [NSCursor resizeUpDownCursor];
		} else if (kind == 3) {
			c = [NSCursor IBeamCursor];
		}
		[c set];
	}
}
*/
import "C"

// applySystemCursor sets the pointer via NSCursor (arrow, resize
// left/right, resize up/down, I-beam). An AppKit window backend should
// call this from Surface.SetCursor.
func applySystemCursor(c Cursor) {
	C.ui_set_nscursor(C.int(darwinCursorKind(c)))
}

type darwinCursorHost struct {
	cursor Cursor
}

func (d *darwinCursorHost) SetCursor(c Cursor) {
	d.cursor = c
	applySystemCursor(c)
}

func (d *darwinCursorHost) Cursor() Cursor { return d.cursor }
