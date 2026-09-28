//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>

// uitk_pb_get copies the general pasteboard's string, or returns NULL
// when it holds none. The caller frees it.
static char *uitk_pb_get(void) {
	@autoreleasepool {
		NSString *s = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
		if (!s) return NULL;
		const char *u = [s UTF8String];
		return u ? strdup(u) : NULL;
	}
}

static void uitk_pb_set(const char *text) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		// clearContents is not optional: it takes ownership of the
		// pasteboard and bumps its change count, and setString: before
		// it is simply dropped.
		[pb clearContents];
		[pb setString:[NSString stringWithUTF8String:text ? text : ""]
		      forType:NSPasteboardTypeString];
	}
}
*/
import "C"

import "unsafe"

// The macOS clipboard: NSPasteboard, general, as a string.
//
// It is the easiest of the three. X11 has to own a selection and answer
// requests for it, and keep answering after the window is gone; Wayland
// has to hold a wl_data_device; NSPasteboard is a *server* the system
// runs, so setting a string hands it over and there is nothing to serve
// afterwards. Nothing here needs a background drainer, and nothing can
// hang waiting for another application to answer.
//
// There is no PRIMARY selection, exactly as on Windows: X11's
// middle-click clipboard has no counterpart here, so
// ClipboardPrimaryGet answers from the same place as ClipboardGet
// rather than pretending to a second one.
//
// A pasteboard with no string at all — an image, a file, nothing —
// answers false rather than "", so [ClipboardGet] falls back to the
// in-process buffer instead of reporting the clipboard as empty when
// it is merely holding something else.

func clipboardNativeGet() (string, bool) {
	p := C.uitk_pb_get()
	if p == nil {
		return "", false
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p), true
}

func clipboardNativePrimaryGet() (string, bool) { return clipboardNativeGet() }

func clipboardNativeSet(s string) {
	c := C.CString(s)
	defer C.free(unsafe.Pointer(c))
	C.uitk_pb_set(c)
}
