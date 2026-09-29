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

// pbType is the pasteboard type a MIME string travels as. The two
// macOS has names of its own for get them; everything else goes on the
// pasteboard under the MIME string itself, which NSPasteboard allows
// and which lets a toolkit-specific type cross between two uitoolkit
// windows without anyone registering anything.
static NSString *pbType(NSString *mime) {
	if ([mime hasPrefix:@"text/plain"]) return NSPasteboardTypeString;
	if ([mime isEqualToString:@"text/uri-list"]) return NSPasteboardTypeFileURL;
	return mime;
}

// The toolkit's DragAction bitset (copy 1, move 2, link 4) and
// NSDragOperation, which agree on nothing.
static NSDragOperation opsFromActions(int a) {
	NSDragOperation op = NSDragOperationNone;
	if (a & 1) op |= NSDragOperationCopy;
	if (a & 2) op |= NSDragOperationMove;
	if (a & 4) op |= NSDragOperationLink;
	return op;
}

static int actionsFromOps(NSDragOperation op) {
	int a = 0;
	if (op & NSDragOperationCopy) a |= 1;
	if (op & NSDragOperationMove) a |= 2;
	if (op & NSDragOperationLink) a |= 4;
	return a;
}

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

@interface UitkView : NSView <NSDraggingSource, NSTextInputClient>
@property(assign) uintptr_t sid;
@property(assign) int cursorKind;
// strong, not assign: ARC would release the tracking area the moment it
// was stored and the view would be left pointing at freed memory.
@property(strong) NSTrackingArea *tracking;
// The drag now over this view, what the toolkit last said it would do
// with it, and the drag this view started. currentDrag is weak: the
// dragging info belongs to AppKit and is gone when the drag ends.
@property(weak) id<NSDraggingInfo> currentDrag;
@property(assign) int dropAnswer;
@property(assign) int dragActions;
@property(assign) BOOL dragging;
// The text being composed, and where the caret is so the candidate
// window can be put beside it.
@property(strong) NSString *marked;
@property(assign) NSRange markedSel;
@property(assign) NSRect imeCaret;
@property(assign) BOOL imeOn;
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

// --- drop target ---
//
// Every one of these reports to Go and answers from what the toolkit
// last said; none of them decides anything itself. The application is
// asked through EventDragMotion and replies with AcceptDrag, which
// lands in dropAnswer, which is what -draggingUpdated: returns.

- (NSPoint)dragPoint:(id<NSDraggingInfo>)info {
	NSPoint p = [self convertPoint:[info draggingLocation] fromView:nil];
	double s = self.window ? self.window.backingScaleFactor : 1;
	return NSMakePoint(p.x * s, (self.bounds.size.height - p.y) * s);
}

- (void)reportDrag:(int)kind info:(id<NSDraggingInfo>)info {
	NSPoint p = [self dragPoint:info];
	int allowed = actionsFromOps([info draggingSourceOperationMask]);
	uitkAkInput(self.sid, kind, p.x, p.y, 0, 0, allowed, allowed, 0, NULL, 0);
}

- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)info {
	self.currentDrag = info;
	self.dropAnswer = 0;
	[self reportDrag:UITK_AK_DRAG_MOTION info:info];
	return opsFromActions(self.dropAnswer);
}

- (NSDragOperation)draggingUpdated:(id<NSDraggingInfo>)info {
	self.currentDrag = info;
	[self reportDrag:UITK_AK_DRAG_MOTION info:info];
	return opsFromActions(self.dropAnswer);
}

- (void)draggingExited:(id<NSDraggingInfo>)info {
	self.currentDrag = nil;
	self.dropAnswer = 0;
	uitkAkInput(self.sid, UITK_AK_DRAG_LEAVE, 0, 0, 0, 0, 0, 0, 0, NULL, 0);
}

- (BOOL)prepareForDragOperation:(id<NSDraggingInfo>)info {
	self.currentDrag = info;
	return self.dropAnswer != 0;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)info {
	self.currentDrag = info;
	[self reportDrag:UITK_AK_DROP info:info];
	// The pasteboard belongs to the drag and the drag is over once this
	// returns, so the Go side reads what it wants inside this call —
	// uitk_ak_drop_data answers from self.currentDrag, which is still
	// set. The toolkit's ReceiveDrop is served from a copy it took here.
	self.currentDrag = nil;
	return YES;
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
	// The key first, always: a shortcut, an arrow or Escape is about
	// the key and must arrive whatever the input method does with the
	// event afterwards.
	uitkAkInput(self.sid, UITK_AK_KEY_DOWN, 0, 0, 0, 0, e.isARepeat ? 1 : 0,
	            bareKey(e), (uint64_t)e.modifierFlags, NULL, 0);
	// Then the text, through the input context rather than off the
	// event. That is what makes an input method work at all: with a
	// Japanese or Pinyin method active, -interpretKeyEvents: calls
	// -setMarkedText: while the user composes and -insertText: when
	// they choose, and taking [e characters] instead would insert the
	// raw Latin keystrokes as the user typed them.
	//
	// It is also why the dead keys work: é on a US-International
	// layout is two events and one insertText:.
	[self interpretKeyEvents:@[e]];
}

- (void)keyUp:(NSEvent *)e {
	uitkAkInput(self.sid, UITK_AK_KEY_UP, 0, 0, 0, 0, 0, bareKey(e),
	            (uint64_t)e.modifierFlags, NULL, 0);
}

// --- input method ---
//
// Nothing here decides anything either: each callback reports and the
// toolkit does the editing, which is the same rule the rest of this
// file keeps. The toolkit is the one that knows where the caret is and
// what is selected, so the ranges below are answered from what it last
// said rather than from a document this view does not have.

- (BOOL)hasMarkedText { return self.marked.length > 0; }
- (NSRange)markedRange {
	return self.marked.length ? NSMakeRange(0, self.marked.length) : NSMakeRange(NSNotFound, 0);
}
- (NSRange)selectedRange { return self.markedSel; }
- (NSArray<NSAttributedStringKey> *)validAttributesForMarkedText { return @[]; }
- (NSAttributedString *)attributedSubstringForProposedRange:(NSRange)r actualRange:(NSRangePointer)a {
	return nil;
}
- (NSUInteger)characterIndexForPoint:(NSPoint)p { return NSNotFound; }

// utf8Caret is a UTF-16 offset into s as a byte offset into its UTF-8,
// because that is the unit EventIMEPreedit's caret is in and the two
// part company on the first character outside the basic plane.
static int utf8Caret(NSString *s, NSUInteger utf16Off) {
	if (!s || utf16Off == 0 || utf16Off > s.length) return 0;
	NSString *head = [s substringToIndex:utf16Off];
	return (int)[head lengthOfBytesUsingEncoding:NSUTF8StringEncoding];
}

- (void)setMarkedText:(id)string selectedRange:(NSRange)sel replacementRange:(NSRange)repl {
	NSString *t = [string isKindOfClass:[NSAttributedString class]] ? [string string] : string;
	self.marked = t.length ? [t copy] : nil;
	self.markedSel = sel;
	uitkAkInput(self.sid, UITK_AK_IME_PREEDIT, 0, 0, 0, 0,
	            repl.location == NSNotFound ? 0 : (int)repl.length,
	            utf8Caret(t, sel.location), 0,
	            t.length ? [t UTF8String] : "", 0);
}

- (void)unmarkText {
	if (!self.marked) return;
	self.marked = nil;
	self.markedSel = NSMakeRange(0, 0);
	uitkAkInput(self.sid, UITK_AK_IME_CANCEL, 0, 0, 0, 0, 0, 0, 0, NULL, 0);
}

- (void)insertText:(id)string replacementRange:(NSRange)repl {
	NSString *t = [string isKindOfClass:[NSAttributedString class]] ? [string string] : string;
	if (t.length == 0) { [self unmarkText]; return; }
	BOOL wasComposing = self.marked.length > 0;
	self.marked = nil;
	self.markedSel = NSMakeRange(0, 0);
	// A commit while composing is the input method's answer and goes to
	// the IME seam; text typed with no composition is ordinary typing
	// and goes to the text one. The toolkit tells them apart, so this
	// has to as well.
	uitkAkInput(self.sid, wasComposing ? UITK_AK_IME_COMMIT : UITK_AK_KEY_DOWN,
	            0, 0, 0, 0, 0, 0, 0, [t UTF8String], wasComposing ? 0 : 2);
}

// The candidate window goes beside the caret, in screen coordinates.
- (NSRect)firstRectForCharacterRange:(NSRange)r actualRange:(NSRangePointer)a {
	NSRect in = self.imeCaret;
	if (NSIsEmptyRect(in)) in = NSMakeRect(0, 0, 1, 16);
	NSRect w = [self convertRect:in toView:nil];
	return [self.window convertRectToScreen:w];
}

// interpretKeyEvents turns an arrow into moveLeft: and Return into
// insertNewline:. The toolkit has already had those as EventKeyDown,
// so they are swallowed here rather than becoming text.
- (void)doCommandBySelector:(SEL)sel {}

// --- drag source ---

- (NSDragOperation)draggingSession:(NSDraggingSession *)session
    sourceOperationMaskForDraggingContext:(NSDraggingContext)context {
	return opsFromActions(self.dragActions);
}

- (void)draggingSession:(NSDraggingSession *)session
           endedAtPoint:(NSPoint)p
              operation:(NSDragOperation)op {
	self.dragging = NO;
	// The action that was performed, and whether anything took it: a
	// drag let go over nothing ends with NSDragOperationNone, which the
	// toolkit reads as a drop that happened and nobody wanted.
	uitkAkInput(self.sid, UITK_AK_DRAG_END, p.x, p.y, 0, 0,
	            actionsFromOps(op), actionsFromOps(op), 0, NULL, 1);
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

// --- popups ----------------------------------------------------------

void *uitk_ak_popup_new(uintptr_t sid, void *parentWin, int x, int y, int w, int h, int tooltip) {
	@autoreleasepool {
		NSWindow *parent = (__bridge NSWindow *)parentWin;
		NSRect content = NSMakeRect(0, 0, w > 0 ? w : 1, h > 0 ? h : 1);
		// A panel, and non-activating: clicking a menu must not take key
		// away from the window the menu belongs to. NSWindow has no such
		// style bit; NSPanel does.
		NSPanel *p = [[NSPanel alloc] initWithContentRect:content
		                                        styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		                                          backing:NSBackingStoreBuffered
		                                            defer:NO];
		if (!p) return NULL;
		p.releasedWhenClosed = NO;
		p.opaque = NO;
		p.backgroundColor = [NSColor clearColor];
		// macOS draws a menu's shadow itself (FrameSystemShadow), and a
		// tooltip's too; the toolkit reserves no margin for one.
		p.hasShadow = YES;
		p.level = tooltip ? NSStatusWindowLevel : NSPopUpMenuWindowLevel;
		// It rides with its window across spaces and full screen, and
		// never gets a window-cycling entry of its own.
		p.collectionBehavior = NSWindowCollectionBehaviorTransient |
		                       NSWindowCollectionBehaviorIgnoresCycle |
		                       NSWindowCollectionBehaviorFullScreenAuxiliary;
		p.hidesOnDeactivate = YES;

		UitkView *view = [[UitkView alloc] initWithFrame:content];
		view.sid = sid;
		view.wantsLayer = YES;
		view.layer.contentsGravity = kCAGravityTopLeft;
		view.layer.magnificationFilter = kCAFilterNearest;
		p.contentView = view;
		p.acceptsMouseMovedEvents = YES;

		[p setFrameTopLeftPoint:NSMakePoint(x, flipY(y))];
		// A child window follows its parent when the parent moves and
		// goes away with it, which is what a menu should do.
		if (parent) [parent addChildWindow:p ordered:NSWindowAbove];
		[p orderFront:nil];
		return (void *)CFBridgingRetain(p);
	}
}

void uitk_ak_popup_move(void *w, int x, int y, int width, int height) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *p = (__bridge NSWindow *)w;
		[p setContentSize:NSMakeSize(width > 0 ? width : 1, height > 0 ? height : 1)];
		[p setFrameTopLeftPoint:NSMakePoint(x, flipY(y))];
	}
}

void uitk_ak_popup_close(void *parentWin, void *w) {
	if (!w) return;
	@autoreleasepool {
		NSWindow *p = (__bridge NSWindow *)w;
		NSWindow *parent = parentWin ? (__bridge NSWindow *)parentWin : nil;
		if (parent) [parent removeChildWindow:p];
		[p orderOut:nil];
		[p close];
		CFBridgingRelease(w);
	}
}

void uitk_ak_content_origin(void *w, int *x, int *y) {
	if (x) *x = 0;
	if (y) *y = 0;
	if (!w) return;
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		NSRect c = [win contentRectForFrameRect:win.frame];
		if (x) *x = (int)lround(c.origin.x);
		if (y) *y = (int)lround(flipY(NSMaxY(c)));
	}
}

void uitk_ak_work_area(void *w, int *x, int *y, int *width, int *height) {
	if (x) *x = 0;
	if (y) *y = 0;
	if (width) *width = 0;
	if (height) *height = 0;
	@autoreleasepool {
		NSScreen *sc = nil;
		if (w) sc = ((__bridge NSWindow *)w).screen;
		if (!sc) sc = [NSScreen mainScreen];
		if (!sc) return;
		// visibleFrame, not frame: the menu bar and the Dock are not
		// somewhere a menu may open.
		NSRect v = sc.visibleFrame;
		if (x) *x = (int)lround(v.origin.x);
		if (y) *y = (int)lround(flipY(NSMaxY(v)));
		if (width) *width = (int)lround(v.size.width);
		if (height) *height = (int)lround(v.size.height);
	}
}

// --- drag and drop ---------------------------------------------------

// uitkView is the window's content view as a UitkView, or nil.
static UitkView *uitkView(void *w) {
	if (!w) return nil;
	NSView *v = ((__bridge NSWindow *)w).contentView;
	return [v isKindOfClass:[UitkView class]] ? (UitkView *)v : nil;
}

void uitk_ak_register_drops(void *w) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v) return;
		// Text and file URLs by their own names, and NSPasteboardTypeURL
		// so a link dragged out of a browser arrives too. Anything else
		// a uitoolkit window offers travels under its MIME string, and
		// registering for a type nobody sends costs nothing.
		[v registerForDraggedTypes:@[
			NSPasteboardTypeString, NSPasteboardTypeFileURL, NSPasteboardTypeURL,
			@"text/plain", @"text/plain;charset=utf-8", @"text/uri-list", @"text/html",
		]];
	}
}

void uitk_ak_set_drop_answer(void *w, int action) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (v) v.dropAnswer = action;
	}
}

// mimesOf is the pasteboard's types as the MIME names the toolkit
// speaks, NUL-separated and NUL-terminated.
static char *mimesOf(NSPasteboard *pb, int *n) {
	NSMutableArray<NSString *> *out = [NSMutableArray array];
	if ([pb canReadObjectForClasses:@[[NSURL class]] options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}]) {
		[out addObject:@"text/uri-list"];
	}
	if ([pb canReadObjectForClasses:@[[NSString class]] options:@{}]) {
		[out addObject:@"text/plain;charset=utf-8"];
		[out addObject:@"text/plain"];
	}
	for (NSPasteboardType t in pb.types) {
		// A type that is already a MIME string is one of ours and goes
		// through untranslated; the rest are Apple's and are covered
		// above or are of no use to the toolkit.
		if ([t containsString:@"/"] && ![t hasPrefix:@"public."] && ![out containsObject:t]) {
			[out addObject:t];
		}
	}
	NSMutableData *buf = [NSMutableData data];
	for (NSString *m in out) {
		const char *u = [m UTF8String];
		[buf appendBytes:u length:strlen(u) + 1];
	}
	if (n) *n = (int)buf.length;
	if (buf.length == 0) return NULL;
	char *res = (char *)malloc(buf.length);
	memcpy(res, buf.bytes, buf.length);
	return res;
}

char *uitk_ak_drop_types(void *w, int *n) {
	if (n) *n = 0;
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v || !v.currentDrag) return NULL;
		return mimesOf([v.currentDrag draggingPasteboard], n);
	}
}

void *uitk_ak_drop_data(void *w, const char *mime, int *n) {
	if (n) *n = 0;
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v || !v.currentDrag || !mime) return NULL;
		NSPasteboard *pb = [v.currentDrag draggingPasteboard];
		NSString *m = [NSString stringWithUTF8String:mime];
		NSData *data = nil;
		if ([m isEqualToString:@"text/uri-list"]) {
			// The toolkit's uri-list is what every other backend hands
			// it: one URI per line, CRLF, which is what RFC 2483 says.
			NSArray *urls = [pb readObjectsForClasses:@[[NSURL class]]
			                                  options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
			NSMutableString *s = [NSMutableString string];
			for (NSURL *u in urls) [s appendFormat:@"%@\r\n", u.absoluteString];
			data = [s dataUsingEncoding:NSUTF8StringEncoding];
		} else if ([m hasPrefix:@"text/plain"]) {
			NSString *s = [pb stringForType:NSPasteboardTypeString];
			data = [s dataUsingEncoding:NSUTF8StringEncoding];
		} else {
			data = [pb dataForType:m];
		}
		if (!data || data.length == 0) return NULL;
		void *res = malloc(data.length);
		memcpy(res, data.bytes, data.length);
		if (n) *n = (int)data.length;
		return res;
	}
}

void *uitk_ak_drag_new(void) {
	@autoreleasepool { return (void *)CFBridgingRetain([[NSPasteboardItem alloc] init]); }
}

void uitk_ak_drag_add(void *item, const char *mime, const void *bytes, int n) {
	if (!item || !mime || !bytes || n <= 0) return;
	@autoreleasepool {
		NSPasteboardItem *it = (__bridge NSPasteboardItem *)item;
		NSString *m = [NSString stringWithUTF8String:mime];
		NSData *d = [NSData dataWithBytes:bytes length:(NSUInteger)n];
		// Under the type macOS knows it by *and* under the MIME name, so
		// the same item satisfies a Finder drop and a uitoolkit one.
		[it setData:d forType:pbType(m)];
		if (![pbType(m) isEqualToString:m]) [it setData:d forType:m];
	}
}

void uitk_ak_drag_free(void *item) {
	if (item) CFBridgingRelease(item);
}

int uitk_ak_drag_start(void *w, void *item, const unsigned char *icon, int iw, int ih,
                       double hotX, double hotY, int actions) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v || !item) return 0;
		// A drag has to come from a press, the way an interactive move
		// does: beginDraggingSession tracks the button that is down.
		NSEvent *e = [NSApp currentEvent];
		if (!e) return 0;
		switch (e.type) {
		case NSEventTypeLeftMouseDown:
		case NSEventTypeLeftMouseDragged:
		case NSEventTypeRightMouseDown:
		case NSEventTypeRightMouseDragged:
		case NSEventTypeOtherMouseDown:
		case NSEventTypeOtherMouseDragged:
			break;
		default:
			return 0;
		}
		NSPasteboardItem *it = (__bridge NSPasteboardItem *)item;
		NSDraggingItem *di = [[NSDraggingItem alloc] initWithPasteboardWriter:it];

		double s = v.window ? v.window.backingScaleFactor : 1;
		NSPoint at = [v convertPoint:e.locationInWindow fromView:nil];
		NSImage *img = nil;
		NSRect frame = NSMakeRect(at.x - 8, at.y - 8, 16, 16);
		if (icon && iw > 0 && ih > 0) {
			CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
			CGContextRef ctx = CGBitmapContextCreate((void *)icon, iw, ih, 8, (size_t)iw * 4, cs,
			                                         kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
			CGColorSpaceRelease(cs);
			if (ctx) {
				CGImageRef cg = CGBitmapContextCreateImage(ctx);
				CGContextRelease(ctx);
				if (cg) {
					// The picture is in device pixels and the frame is in
					// points, so it is divided by the scale: an icon drawn
					// for a Retina display is the same size on screen as
					// one drawn for a plain one, and sharper.
					double pw = iw / s, ph = ih / s;
					img = [[NSImage alloc] initWithCGImage:cg size:NSMakeSize(pw, ph)];
					CGImageRelease(cg);
					frame = NSMakeRect(at.x - hotX / s, at.y - (ph - hotY / s), pw, ph);
				}
			}
		}
		[di setDraggingFrame:frame contents:img];
		v.dragActions = actions;
		v.dragging = YES;
		[v beginDraggingSessionWithItems:@[di] event:e source:v];
		return 1;
	}
}

int uitk_ak_dragging(void *w) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		return v && v.dragging ? 1 : 0;
	}
}

void uitk_ak_set_ime_enabled(void *w, int on) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v) return;
		v.imeOn = on ? YES : NO;
		if (on) [[v inputContext] activate];
		else {
			// Anything half-composed is abandoned with it, or the next
			// field inherits the last one's preedit.
			[[v inputContext] discardMarkedText];
			[[v inputContext] deactivate];
			[v unmarkText];
		}
	}
}

void uitk_ak_set_ime_cursor(void *w, int x, int y, int cw, int ch) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v) return;
		double s = v.window ? v.window.backingScaleFactor : 1;
		// Device pixels, y down, to the view's points, y up.
		double px = x / s, ph = ch / s;
		double py = v.bounds.size.height - (y / s) - ph;
		v.imeCaret = NSMakeRect(px, py, cw / s, ph);
		[[v inputContext] invalidateCharacterCoordinates];
	}
}

void uitk_ak_ime_simulate(void *w, int what, const char *text, int caret) {
	@autoreleasepool {
		UitkView *v = uitkView(w);
		if (!v) return;
		NSString *t = text ? [NSString stringWithUTF8String:text] : @"";
		switch (what) {
		case 1: [v setMarkedText:t selectedRange:NSMakeRange((NSUInteger)caret, 0)
		           replacementRange:NSMakeRange(NSNotFound, 0)]; break;
		case 2: [v insertText:t replacementRange:NSMakeRange(NSNotFound, 0)]; break;
		case 3: [v unmarkText]; break;
		}
	}
}

// uitk_ak_set_role restyles a window as a dialog, or back.
//
// Not an NSPanel: the window is made before the role is known, and
// rebuilding it as a panel would mean rebuilding its view, its layer and
// its delegate with it. What a panel gives a dialog is the floating
// level and the absence of a minimize button, and both are properties of
// an NSWindow — so this sets those, and the window behaves as a panel
// without being one.
//
// NSFloatingWindowLevel keeps it above the application's ordinary
// windows without putting it above every other application's, which is
// what a prompt wants and what NSStatusWindowLevel would get wrong.
void uitk_ak_set_role(void *w, int dialog) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if (!win) return;
		if (dialog) {
			win.styleMask &= ~NSWindowStyleMaskMiniaturizable;
			win.level = NSFloatingWindowLevel;
			win.collectionBehavior |= NSWindowCollectionBehaviorMoveToActiveSpace;
		} else {
			win.styleMask |= NSWindowStyleMaskMiniaturizable;
			win.level = NSNormalWindowLevel;
		}
	}
}

void uitk_ak_center(void *w) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if (win) [win center];
	}
}

// uitk_ak_activate brings the window to the front and gives it the
// keyboard. activateIgnoringOtherApps: as well, because a process that
// is not frontmost gets its window ordered in behind everything else
// otherwise — which is exactly the case a prompt opened from a terminal
// is in.
int uitk_ak_activate(void *w) {
	@autoreleasepool {
		NSWindow *win = (__bridge NSWindow *)w;
		if (!win) return 0;
		[NSApp activateIgnoringOtherApps:YES];
		[win makeKeyAndOrderFront:nil];
		return 1;
	}
}
