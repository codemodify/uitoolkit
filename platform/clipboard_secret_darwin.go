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

// clipboardNativeGetSecret: no bytes path here yet, so the caller falls
// back to the ordinary read. See the Linux file for what this is for.
func clipboardNativeGetSecret() ([]byte, bool) { return nil, false }

// clipboardSecretBytesPath reports whether this platform can read a secret
// from the clipboard without making a string of it. Where it can, a failed
// read is a failed read: falling back to the string reader would undo the
// whole point of the bytes path on the try that happens to succeed.
func clipboardSecretBytesPath() bool { return false }
