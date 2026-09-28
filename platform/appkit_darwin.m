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
extern void uitkAkInput(uintptr_t sid, int kind, double x, double y,
                        double dx, double dy, int button, int key,
                        uint64_t mods, const char *text, int flags);

// flipY turns a point in AppKit's screen coordinates — y up from the
// bottom-left of the *main* screen — into the toolkit's, y down from its
// top-left. It is its own inverse, which is why one function does both
// directions.
static double flipY(double y) {
	NSArray<NSScreen *> *screens = [NSScreen screens];
	if (screens.count == 0) return y;
	return NSMaxY([screens objectAtIndex:0].frame) - y;
}

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
	uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0);
}
- (void)windowDidResignKey:(NSNotification *)n {
	uitkAkEvent(self.sid, UITK_AK_FOCUS_OUT, 0, 0);
	uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0);
}
// Every way the window's state can change, so the toolkit's frame is
// repainted active or inactive, maximized or not, with the window.
- (void)windowDidMiniaturize:(NSNotification *)n   { uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0); }
- (void)windowDidDeminiaturize:(NSNotification *)n { uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0); }
- (void)windowDidEnterFullScreen:(NSNotification *)n { uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0); }
- (void)windowDidExitFullScreen:(NSNotification *)n  { uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0); }
- (void)windowDidEndLiveResize:(NSNotification *)n   { uitkAkEvent(self.sid, UITK_AK_STATE, 0, 0); }
@end

// UitkView is the window's content. It exists for input: a window
// delegate is told about the window, never about a key or a press, and
// those arrive at the view that is under the pointer or holds the
// responder chain.
//
// Everything it does is turn an NSEvent into a call to Go that appends
// to a queue — the same rule the rest of this file keeps. In
// particular it does not draw: the layer's contents is the frame (see
// the note at the top), so there is no drawRect: here at all.
// Defined in cursor_darwin.go's preamble: the NSCursor for one of the
// numbers darwinCursorKind produces.
extern NSCursor *uitk_nscursor_for(int kind);

@interface UitkView : NSView
@property(assign) uintptr_t sid;
@property(assign) int cursorKind;
// strong, not assign: ARC would release the tracking area the moment it
// was stored and the view would be left pointing at freed memory.
@property(strong) NSTrackingArea *tracking;
@end

@implementation UitkView

// Without this a window's first click only brings it forward, and a
// press on a button in a background window needs two.
- (BOOL)acceptsFirstResponder { return YES; }
- (BOOL)acceptsFirstMouse:(NSEvent *)e { return YES; }

// **Not** isFlipped, however much a y-down toolkit would like it to be.
// A flipped NSView's backing layer gets geometryFlipped, which makes
// contentsAreFlipped, which draws the layer's contents image upside
// down — and the contents image is the whole window. The y flip for
// input costs one subtraction in -where: below; the y flip for the
// buffer would cost every frame being wrong.

// Mouse moves only arrive while the window is told to want them, and
// enter and leave need a tracking area the size of the view. Both are
// re-made whenever the view is resized, which is what this override is
// for.
- (void)updateTrackingAreas {
	[super updateTrackingAreas];
	if (self.tracking) [self removeTrackingArea:self.tracking];
	NSTrackingAreaOptions opts = NSTrackingMouseEnteredAndExited |
	                             NSTrackingMouseMoved | NSTrackingCursorUpdate |
	                             NSTrackingActiveInKeyWindow | NSTrackingInVisibleRect;
	self.tracking = [[NSTrackingArea alloc] initWithRect:self.bounds
	                                            options:opts owner:self userInfo:nil];
	[self addTrackingArea:self.tracking];
}

// where converts an event's location to device pixels of the content
// area, y down. The toolkit works in device pixels everywhere and the
// view is flipped, so this is a point-to-pixel scale and nothing else.
- (NSPoint)where:(NSEvent *)e {
	NSPoint p = [self convertPoint:e.locationInWindow fromView:nil];
	double s = self.window ? self.window.backingScaleFactor : 1;
	// Points with y up from the bottom-left, which is the view's, to
	// device pixels with y down from the top-left, which is the
	// toolkit's. uitk_ak_post_mouse does *not* invert this — it takes
	// AppKit's own coordinates — so a test of where a press lands is a
	// test of this line rather than a round trip through it.
	return NSMakePoint(p.x * s, (self.bounds.size.height - p.y) * s);
}

- (void)send:(int)kind event:(NSEvent *)e button:(int)b {
	NSPoint p = [self where:e];
	uitkAkInput(self.sid, kind, p.x, p.y, 0, 0, b, 0,
	            (uint64_t)e.modifierFlags, NULL, 0);
}

- (void)mouseDown:(NSEvent *)e        { [self send:UITK_AK_MOUSE_DOWN event:e button:1]; }
- (void)mouseUp:(NSEvent *)e          { [self send:UITK_AK_MOUSE_UP   event:e button:1]; }
- (void)rightMouseDown:(NSEvent *)e   { [self send:UITK_AK_MOUSE_DOWN event:e button:3]; }
- (void)rightMouseUp:(NSEvent *)e     { [self send:UITK_AK_MOUSE_UP   event:e button:3]; }
- (void)otherMouseDown:(NSEvent *)e   { [self send:UITK_AK_MOUSE_DOWN event:e button:(int)e.buttonNumber + 1]; }
- (void)otherMouseUp:(NSEvent *)e     { [self send:UITK_AK_MOUSE_UP   event:e button:(int)e.buttonNumber + 1]; }
- (void)mouseMoved:(NSEvent *)e       { [self send:UITK_AK_MOUSE_MOVE event:e button:0]; }
// A drag is a move with a button held, and the toolkit wants the move:
// it tracks the buttons itself from the presses it was already sent.
- (void)mouseDragged:(NSEvent *)e      { [self send:UITK_AK_MOUSE_MOVE event:e button:0]; }
- (void)rightMouseDragged:(NSEvent *)e { [self send:UITK_AK_MOUSE_MOVE event:e button:0]; }
- (void)otherMouseDragged:(NSEvent *)e { [self send:UITK_AK_MOUSE_MOVE event:e button:0]; }
- (void)mouseEntered:(NSEvent *)e      { [self send:UITK_AK_MOUSE_MOVE event:e button:0]; }

// AppKit asks every time the pointer moves over the view, and the
// default answer would put the arrow back. Answering here is the half
// that makes a shape stick; see -[akSurface SetCursor].
- (void)cursorUpdate:(NSEvent *)e {
	[uitk_nscursor_for(self.cursorKind) set];
}

- (void)mouseExited:(NSEvent *)e {
	NSPoint p = [self where:e];
	uitkAkInput(self.sid, UITK_AK_POINTER_LEAVE, p.x, p.y, 0, 0, 0, 0,
	            (uint64_t)e.modifierFlags, NULL, 0);
}

// hasPreciseScrollingDeltas tells a trackpad from a wheel. A trackpad
// gives device pixels and the toolkit scrolls by them; a wheel gives
// notches. Getting this wrong is the difference between a list that
// glides and one that jumps a page per flick.
- (void)scrollWheel:(NSEvent *)e {
	NSPoint p = [self where:e];
	BOOL precise = e.hasPreciseScrollingDeltas;
	double dx = e.scrollingDeltaX, dy = e.scrollingDeltaY;
	if (precise) {
		double s = self.window ? self.window.backingScaleFactor : 1;
		dx *= s; dy *= s;
	}
	uitkAkInput(self.sid, UITK_AK_SCROLL, p.x, p.y, dx, dy, 0, 0,
	            (uint64_t)e.modifierFlags, NULL, precise ? 1 : 0);
}

// bareKey is the first code point of charactersIgnoringModifiers: the
// key the layout says this position is, before Shift or Option. 0 for an
// event that carries no characters at all.
static int bareKey(NSEvent *e) {
	NSString *b = [e charactersIgnoringModifiers];
	return b.length > 0 ? (int)[b characterAtIndex:0] : 0;
}

- (void)keyDown:(NSEvent *)e {
	// charactersIgnoringModifiers is the key, characters is the text;
	// the Go side decides what each one means (appkit_keys_darwin.go).
	// ARC keeps the string alive for the duration of the call, which is
	// all the Go side needs: it copies before returning.
	uitkAkInput(self.sid, UITK_AK_KEY_DOWN, 0, 0, 0, 0, e.isARepeat ? 1 : 0,
	            bareKey(e), (uint64_t)e.modifierFlags,
	            [[e characters] UTF8String], 0);
}

- (void)keyUp:(NSEvent *)e {
	uitkAkInput(self.sid, UITK_AK_KEY_UP, 0, 0, 0, 0, 0, bareKey(e),
	            (uint64_t)e.modifierFlags, NULL, 0);
}

// A modifier has no keyDown: of its own. The toolkit needs Alt, because
// a window underlines its mnemonics while it is held, so the flags go
// over as a key event with no key — flags 1 says so — and the Go side
// works out which modifier changed and whether it went down or up.
- (void)flagsChanged:(NSEvent *)e {
	uitkAkInput(self.sid, UITK_AK_KEY_DOWN, 0, 0, 0, 0, 0, 0,
	            (uint64_t)e.modifierFlags, NULL, 1);
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

		UitkView *view = [[UitkView alloc] initWithFrame:rect];
		view.sid = sid;
		view.wantsLayer = YES;
		// The buffer is the whole content and every pixel of it is
		// painted, so the layer neither scales nor blends what is under
		// it. kCAGravityTopLeft keeps the image still while a resize is
		// in flight instead of stretching the old frame.
		view.layer.contentsGravity = kCAGravityTopLeft;
		view.layer.magnificationFilter = kCAFilterNearest;
		win.contentView = view;
		win.acceptsMouseMovedEvents = YES;
		[win makeFirstResponder:view];

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

// --- geometry -------------------------------------------------------

void uitk_ak_move(void *w, int x, int y) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		// setFrameTopLeftPoint takes the *frame's* top left, which is
		// what the toolkit means by a window's position: the title bar
		// is part of the window as far as placing it goes.
		[win setFrameTopLeftPoint:NSMakePoint(x, flipY(y))];
	}
}

int uitk_ak_position(void *w, int *x, int *y) {
	if (!w) return 0;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSRect f = win.frame;
		if (x) *x = (int)lround(f.origin.x);
		if (y) *y = (int)lround(flipY(NSMaxY(f)));
		return 1;
	}
}

void uitk_ak_order_front(void *w, int key) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if (key) [win makeKeyAndOrderFront:nil];
		else [win orderFront:nil];
	}
}

void uitk_ak_order_out(void *w) {
	if (!w) return;
	@autoreleasepool { [(__bridge NSWindow *)w orderOut:nil]; }
}

void uitk_ak_order_back(void *w) {
	if (!w) return;
	@autoreleasepool { [(__bridge NSWindow *)w orderBack:nil]; }
}

int uitk_ak_visible(void *w) {
	if (!w) return 0;
	@autoreleasepool { return ((__bridge NSWindow *)w).isVisible ? 1 : 0; }
}

void uitk_ak_set_limits(void *w, int minW, int minH, int maxW, int maxH, int resizable) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		win.contentMinSize = NSMakeSize(minW > 0 ? minW : 0, minH > 0 ? minH : 0);
		// AppKit says "no maximum" with FLT_MAX rather than with zero.
		win.contentMaxSize = NSMakeSize(maxW > 0 ? maxW : FLT_MAX,
		                                maxH > 0 ? maxH : FLT_MAX);
		// The style mask is the other half: without the resizable bit
		// the limits are moot, because the user has no handle to pull.
		NSWindowStyleMask m = win.styleMask;
		if (resizable) m |= NSWindowStyleMaskResizable;
		else m &= ~NSWindowStyleMaskResizable;
		if (m != win.styleMask) win.styleMask = m;
	}
}

// --- frame ----------------------------------------------------------

void uitk_ak_miniaturize(void *w) {
	if (!w) return;
	@autoreleasepool { [(__bridge NSWindow *)w miniaturize:nil]; }
}

// Zoom is the Mac's maximize, and it is a toggle rather than a state to
// be set, so this asks only when the answer would change. It is not
// quite "fill the screen" — AppKit zooms to what the window says it
// wants, and a window with no maximum gets the visible frame — but it
// is the gesture the green button makes and the one a user means.
int uitk_ak_set_zoomed(void *w, int on) {
	if (!w) return 0;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if ((win.isZoomed != 0) != (on != 0)) [win zoom:nil];
		return 1;
	}
}

int uitk_ak_zoomed(void *w) {
	if (!w) return 0;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		// isZoomed is true of a window that merely happens to be the
		// size it would zoom to, which a full-screen window always is.
		if (win.styleMask & NSWindowStyleMaskFullScreen) return 0;
		return win.isZoomed ? 1 : 0;
	}
}

int uitk_ak_miniaturized(void *w) {
	if (!w) return 0;
	@autoreleasepool { return ((__bridge NSWindow *)w).isMiniaturized ? 1 : 0; }
}

int uitk_ak_fullscreen(void *w) {
	if (!w) return 0;
	@autoreleasepool {
		return (((__bridge NSWindow *)w).styleMask & NSWindowStyleMaskFullScreen) ? 1 : 0;
	}
}

void uitk_ak_set_fullscreen(void *w, int on) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		int now = (win.styleMask & NSWindowStyleMaskFullScreen) ? 1 : 0;
		if (now != (on != 0)) [win toggleFullScreen:nil];
	}
}

void uitk_ak_set_level(void *w, int above) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		win.level = above ? NSFloatingWindowLevel : NSNormalWindowLevel;
	}
}

int uitk_ak_above(void *w) {
	if (!w) return 0;
	@autoreleasepool {
		return ((__bridge NSWindow *)w).level > NSNormalWindowLevel ? 1 : 0;
	}
}

int uitk_ak_active(void *w) {
	if (!w) return 0;
	@autoreleasepool { return ((__bridge NSWindow *)w).isKeyWindow ? 1 : 0; }
}

void uitk_ak_set_decorations(void *w, int borderless) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSWindowStyleMask titled = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
		                           NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable;
		if (borderless) {
			// Borderless loses the resize handle with everything else, so
			// the bit is put back: a frame the toolkit draws still has
			// edges to pull, and it asks for the drag itself.
			win.styleMask = NSWindowStyleMaskBorderless | NSWindowStyleMaskResizable;
			// The toolkit's frame brings its own shadow and its own
			// corners, so AppKit's would be a second shadow round a
			// rectangle that is no longer the window.
			win.hasShadow = NO;
			win.opaque = NO;
			win.backgroundColor = [NSColor clearColor];
		} else {
			win.styleMask = titled;
			win.hasShadow = YES;
			win.opaque = YES;
			win.backgroundColor = [NSColor windowBackgroundColor];
		}
		// A style-mask change drops the first responder, and with it
		// every key the window would have received.
		if (win.contentView) [win makeFirstResponder:win.contentView];
	}
}

int uitk_ak_start_move(void *w) {
	if (!w) return 0;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSEvent *e = [NSApp currentEvent];
		// Only a press will do: performWindowDragWithEvent: tracks until
		// the button comes up, so handing it anything else hangs the
		// drag on a button that is already released.
		if (!e) return 0;
		switch (e.type) {
		case NSEventTypeLeftMouseDown:
		case NSEventTypeRightMouseDown:
		case NSEventTypeOtherMouseDown:
		case NSEventTypeLeftMouseDragged:
			[win performWindowDragWithEvent:e];
			return 1;
		default:
			return 0;
		}
	}
}

void uitk_ak_set_app_icon(const unsigned char *pix, int w, int h) {
	@autoreleasepool {
		if (!pix || w <= 0 || h <= 0) {
			[NSApp setApplicationIconImage:nil];
			return;
		}
		CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
		CGContextRef ctx = CGBitmapContextCreate((void *)pix, w, h, 8, (size_t)w * 4, cs,
		                                         kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
		CGColorSpaceRelease(cs);
		if (!ctx) return;
		CGImageRef img = CGBitmapContextCreateImage(ctx);
		CGContextRelease(ctx);
		if (!img) return;
		NSImage *ns = [[NSImage alloc] initWithCGImage:img size:NSMakeSize(w, h)];
		CGImageRelease(img);
		[NSApp setApplicationIconImage:ns];
	}
}

// --- verification ---------------------------------------------------

int uitk_ak_readback(void *w, unsigned char *out, int width, int height) {
	if (!w || !out || width <= 0 || height <= 0) return 0;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		CALayer *layer = win.contentView.layer;
		if (!layer) return 0;
		CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
		CGContextRef ctx = CGBitmapContextCreate(out, width, height, 8, (size_t)width * 4, cs,
		                                         kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
		CGColorSpaceRelease(cs);
		if (!ctx) return 0;
		// The layer is in points and the readback is in device pixels,
		// and renderInContext: draws in the layer's own coordinates: so
		// the context is scaled by however many pixels a point is here.
		CGSize sz = layer.bounds.size;
		if (sz.width > 0 && sz.height > 0) {
			CGContextScaleCTM(ctx, width / sz.width, height / sz.height);
		}
		[layer renderInContext:ctx];
		CGContextRelease(ctx);
		return 1;
	}
}

void uitk_ak_post_key(void *w, const char *chars, const char *bare,
                      uint64_t mods, int down) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSEvent *e = [NSEvent keyEventWithType:(down ? NSEventTypeKeyDown : NSEventTypeKeyUp)
		                              location:NSZeroPoint
		                         modifierFlags:(NSEventModifierFlags)mods
		                             timestamp:0
		                          windowNumber:[win windowNumber]
		                               context:nil
		                            characters:[NSString stringWithUTF8String:chars ? chars : ""]
		           charactersIgnoringModifiers:[NSString stringWithUTF8String:bare ? bare : ""]
		                             isARepeat:NO
		                               keyCode:0];
		if (e) [NSApp postEvent:e atStart:NO];
	}
}

void uitk_ak_post_mouse(void *w, int kind, double x, double y,
                        int button, uint64_t mods) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSPoint loc = NSMakePoint(x, y);
		NSEventType type;
		switch (kind) {
		case UITK_AK_MOUSE_DOWN:
			type = button == 3 ? NSEventTypeRightMouseDown : NSEventTypeLeftMouseDown;
			break;
		case UITK_AK_MOUSE_UP:
			type = button == 3 ? NSEventTypeRightMouseUp : NSEventTypeLeftMouseUp;
			break;
		default:
			type = NSEventTypeMouseMoved;
			break;
		}
		NSEvent *e = [NSEvent mouseEventWithType:type
		                                location:loc
		                           modifierFlags:(NSEventModifierFlags)mods
		                               timestamp:0
		                            windowNumber:[win windowNumber]
		                                 context:nil
		                             eventNumber:0
		                              clickCount:1
		                                pressure:1.0];
		if (e) [NSApp postEvent:e atStart:NO];
	}
}

void uitk_ak_set_cursor(void *w, int kind) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if ([win.contentView isKindOfClass:[UitkView class]]) {
			((UitkView *)win.contentView).cursorKind = kind;
		}
	}
}
