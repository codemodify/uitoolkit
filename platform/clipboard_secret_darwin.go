//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>

// uitk_pb_set_secret puts bytes on the general pasteboard as a string,
// and declares the two types the Mac's clipboard managers read:
//
//   org.nspasteboard.ConcealedType  a password; do not record it
//   org.nspasteboard.TransientType  do not keep it in history
//
// Both are the nspasteboard.org convention rather than an Apple API,
// which is what every Mac password manager writes and what every
// clipboard manager that behaves reads. They are declared as types of
// this pasteboard item, which is what "declare, then set" means here:
// setString: alone would not name them.
//
// It takes a pointer and a length rather than a C string, so the caller
// never has to build one, and copies nothing it does not have to.
static void uitk_pb_set_secret(const void *bytes, int len) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSString *s = [[NSString alloc] initWithBytes:bytes
		                                       length:(NSUInteger)len
		                                     encoding:NSUTF8StringEncoding];
		if (!s) s = @"";
		[pb clearContents];
		[pb declareTypes:@[NSPasteboardTypeString,
		                   @"org.nspasteboard.ConcealedType",
		                   @"org.nspasteboard.TransientType"]
		           owner:nil];
		[pb setString:s forType:NSPasteboardTypeString];
		[pb setString:@"" forType:@"org.nspasteboard.ConcealedType"];
		[pb setString:@"" forType:@"org.nspasteboard.TransientType"];
	}
}

// uitk_pb_get_n is uitk_pb_get with the length, so the caller can copy
// the bytes without walking for a NUL and can zero what it was given.
// The caller frees it with uitk_pb_zfree.
static char *uitk_pb_get_n(int *n) {
	@autoreleasepool {
		*n = 0;
		NSString *s = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
		if (!s) return NULL;
		const char *u = [s UTF8String];
		if (!u) return NULL;
		size_t len = strlen(u);
		char *out = malloc(len ? len : 1);
		if (!out) return NULL;
		memcpy(out, u, len);
		*n = (int)len;
		return out;
	}
}

// uitk_pb_zfree clears a buffer before releasing it: free() does not, and
// a passphrase left in a freed allocation is readable by whatever gets
// that memory next. Volatile so the clearing is not optimised away as a
// write to memory about to be released.
static void uitk_pb_zfree(char *p, int n) {
	if (!p) return;
	volatile unsigned char *q = (volatile unsigned char *)p;
	for (int i = 0; i < n; i++) q[i] = 0;
	free(p);
}

static void uitk_pb_clear(void) {
	@autoreleasepool {
		[[NSPasteboard generalPasteboard] clearContents];
	}
}
*/
import "C"

import "unsafe"

func clipboardNativeSetSecret(b []byte) {
	if len(b) == 0 {
		C.uitk_pb_set_secret(nil, 0)
		return
	}
	C.uitk_pb_set_secret(unsafe.Pointer(&b[0]), C.int(len(b)))
}

func clipboardNativeClear() { C.uitk_pb_clear() }

// clipboardNativeGetSecret reads the general pasteboard as bytes, without
// making a Go string of it.
//
// The ordinary read is C.GoString of a strdup'd buffer, which is a Go
// string — unwipeable — and then free()s the C copy without clearing it,
// which leaves a second copy in whatever gets that allocation next. This
// copies the bytes out and zeroes the C buffer before releasing it.
func clipboardNativeGetSecret() ([]byte, bool) {
	var n C.int
	p := C.uitk_pb_get_n(&n)
	if p == nil || n <= 0 {
		if p != nil {
			C.uitk_pb_zfree(p, n)
		}
		return nil, false
	}
	out := C.GoBytes(unsafe.Pointer(p), n)
	C.uitk_pb_zfree(p, n)
	return out, true
}

// clipboardSecretBytesPath reports whether this platform can read a secret
// from the clipboard without making a string of it. Where it can, a failed
// read is a failed read: falling back to the string reader would undo the
// whole point of the bytes path on the try that happens to succeed.
func clipboardSecretBytesPath() bool { return true }
