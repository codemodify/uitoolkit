//go:build darwin && cgo

// AppKit, behind the C face in appkit_darwin.h.
//
// Two decisions shape this file.
//
// **The window's content is a layer, not a drawRect.** The obvious way
// to put a buffer on an NSWindow is an NSView subclass that draws a
// CGImage in drawRect:, and it is the wrong one here: it draws when
// AppKit says so, and the toolkit wants to draw when *it* says so. A
// plain NSView with a layer takes a CGImage assigned straight to
// layer.contents, so a present is one assignment and no drawing cycle.
//
// **Nothing here calls into the toolkit.** The delegate turns AppKit's
// callbacks into uitkAkEvent, which only appends to a queue on the Go
// side — the same rule the Win32 window procedure keeps, and for the
// same reason: AppKit calls these from inside its own machinery.

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <objc/runtime.h>
#include "appkit_darwin.h"

// Implemented in Go (appkit_darwin.go).
extern void uitkAkEvent(uintptr_t sid, int kind, double a, double b);

@interface UitkWindowDelegate : NSObject <NSWindowDelegate>
@property(assign) uintptr_t sid;
@end

@implementation UitkWindowDelegate
// The red button is a *request*: the application may keep the window, so
// this never closes it — it says so and lets the toolkit decide.
- (BOOL)windowShouldClose:(NSWindow *)sender {
	uitkAkEvent(self.sid, UITK_AK_CLOSE, 0, 0);
	return NO;
}
- (void)windowDidResize:(NSNotification *)n {
	NSWindow *w = n.object;
	NSRect r = [w.contentView bounds];
	uitkAkEvent(self.sid, UITK_AK_RESIZE, r.size.width, r.size.height);
}
- (void)windowDidChangeBackingProperties:(NSNotification *)n {
	NSWindow *w = n.object;
	uitkAkEvent(self.sid, UITK_AK_SCALE, w.backingScaleFactor, 0);
}
- (void)windowDidBecomeKey:(NSNotification *)n {
	uitkAkEvent(self.sid, UITK_AK_FOCUS_IN, 0, 0);
}
- (void)windowDidResignKey:(NSNotification *)n {
	uitkAkEvent(self.sid, UITK_AK_FOCUS_OUT, 0, 0);
}
@end

void uitk_ak_init(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		[NSApplication sharedApplication];
		// Regular, so the process gets a Dock tile, a menu bar and the
		// ability to come to the front. An application that never says
		// so is Prohibited and its windows cannot take focus.
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		[NSApp finishLaunching];
		[NSApp activateIgnoringOtherApps:YES];
	});
}

void *uitk_ak_window_new(uintptr_t sid, const char *title, int w, int h) {
	@autoreleasepool {
		NSRect rect = NSMakeRect(0, 0, w, h);
		NSWindowStyleMask mask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
		                         NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable;
		NSWindow *win = [[NSWindow alloc] initWithContentRect:rect
		                                            styleMask:mask
		                                              backing:NSBackingStoreBuffered
		                                                defer:NO];
		if (!win) return NULL;
		win.releasedWhenClosed = NO;
		[win setTitle:[NSString stringWithUTF8String:title ? title : ""]];
		[win center];

		NSView *view = [[NSView alloc] initWithFrame:rect];
		view.wantsLayer = YES;
		// The buffer is the whole content and every pixel of it is
		// painted, so the layer neither scales nor blends what is under
		// it. kCAGravityTopLeft keeps the image still while a resize is
		// in flight instead of stretching the old frame.
		view.layer.contentsGravity = kCAGravityTopLeft;
		view.layer.magnificationFilter = kCAFilterNearest;
		win.contentView = view;

		UitkWindowDelegate *d = [[UitkWindowDelegate alloc] init];
		d.sid = sid;
		win.delegate = d;
		// The delegate is not retained by the window, so it is parked on
		// the window itself to live exactly as long.
		objc_setAssociatedObject(win, "uitkDelegate", d, OBJC_ASSOCIATION_RETAIN);
		return (void *)CFBridgingRetain(win);
	}
}

void uitk_ak_window_show(void *w) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		[win makeKeyAndOrderFront:nil];
	}
}

void uitk_ak_window_close(void *w) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		win.delegate = nil;
		[win close];
		CFBridgingRelease(w);
	}
}

void uitk_ak_set_title(void *w, const char *title) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		[win setTitle:[NSString stringWithUTF8String:title ? title : ""]];
	}
}

void uitk_ak_present(void *w, const unsigned char *pix, int width, int height) {
	if (!w || !pix || width <= 0 || height <= 0) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
		// The toolkit keeps premultiplied RGBA, which is exactly what
		// this says: no swizzle, unlike the DIB on Windows.
		CGContextRef ctx = CGBitmapContextCreate((void *)pix, width, height, 8, (size_t)width * 4, cs,
		                                         kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
		CGColorSpaceRelease(cs);
		if (!ctx) return;
		CGImageRef img = CGBitmapContextCreateImage(ctx);
		CGContextRelease(ctx);
		if (!img) return;
		// Assigning the image is the present. Done without an animation
		// because the default implicit one would cross-fade every frame.
		[CATransaction begin];
		[CATransaction setDisableActions:YES];
		win.contentView.layer.contents = (__bridge id)img;
		[CATransaction commit];
		CGImageRelease(img);
	}
}

void uitk_ak_pump(void) {
	@autoreleasepool {
		for (;;) {
			NSEvent *e = [NSApp nextEventMatchingMask:NSEventMaskAny
			                                untilDate:[NSDate distantPast]
			                                   inMode:NSDefaultRunLoopMode
			                                  dequeue:YES];
			if (!e) break;
			[NSApp sendEvent:e];
		}
	}
}

double uitk_ak_backing_scale(void *w) {
	if (!w) return 1;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		double s = win.backingScaleFactor;
		return s > 0 ? s : 1;
	}
}

void uitk_ak_content_size(void *w, int *outW, int *outH) {
	if (outW) *outW = 0;
	if (outH) *outH = 0;
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSRect b = [win.contentView bounds];
		NSRect px = [win.contentView convertRectToBacking:b];
		if (outW) *outW = (int)px.size.width;
		if (outH) *outH = (int)px.size.height;
	}
}
