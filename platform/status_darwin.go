//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework UserNotifications
#include <dispatch/dispatch.h>
#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <UserNotifications/UserNotifications.h>

// AppKit is main-thread only: NSStatusItem must be created, mutated and
// removed there. A Go caller can be on any thread (and NewStatusItem is
// routinely called off the main goroutine), so every call is trampolined
// onto the main queue. Creation is synchronous because the caller needs
// the handle; the rest is fire-and-forget.
static void ui_main_sync(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_sync(dispatch_get_main_queue(), block);
}

static void ui_main_async(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_async(dispatch_get_main_queue(), block);
}

static void *ui_status_create(const char *title, const char *tip) {
	__block void *out = NULL;
	NSString *t = title ? [NSString stringWithUTF8String:title] : @"";
	NSString *tp = tip ? [NSString stringWithUTF8String:tip] : nil;
	ui_main_sync(^{
		@autoreleasepool {
			NSStatusItem *item = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
			item.button.title = t;
			if (tp) {
				item.button.toolTip = tp;
			}
			item.button.image = [NSImage imageNamed:NSImageNameApplicationIcon];
			item.button.image.template = YES;
			out = (void *)CFBridgingRetain(item);
		}
	});
	return out;
}

static void ui_status_set_title(void *p, const char *title) {
	if (!p) return;
	NSString *t = title ? [NSString stringWithUTF8String:title] : @"";
	ui_main_async(^{
		@autoreleasepool {
			NSStatusItem *item = (__bridge NSStatusItem *)p;
			item.button.title = t;
		}
	});
}

static void ui_status_set_tip(void *p, const char *tip) {
	if (!p) return;
	NSString *t = tip ? [NSString stringWithUTF8String:tip] : @"";
	ui_main_async(^{
		@autoreleasepool {
			NSStatusItem *item = (__bridge NSStatusItem *)p;
			item.button.toolTip = t;
		}
	});
}

static void ui_status_remove(void *p) {
	if (!p) return;
	ui_main_sync(^{
		@autoreleasepool {
			NSStatusItem *item = CFBridgingRelease(p);
			[[NSStatusBar systemStatusBar] removeStatusItem:item];
		}
	});
}

// UNUserNotificationCenter replaces NSUserNotification, which is
// deprecated and silently does nothing for an unbundled binary on recent
// macOS. Delivery still requires a bundle identifier; when there is none
// the request is dropped by the framework rather than crashing.
static void ui_status_notify(const char *title, const char *body) {
	NSString *t = title ? [NSString stringWithUTF8String:title] : @"uitoolkit";
	NSString *b = body ? [NSString stringWithUTF8String:body] : @"";
	ui_main_async(^{
		@autoreleasepool {
			if ([NSBundle mainBundle].bundleIdentifier == nil) {
				return;
			}
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
			UNMutableNotificationContent *content = [UNMutableNotificationContent new];
			content.title = t;
			content.body = b;
			UNNotificationRequest *req = [UNNotificationRequest
				requestWithIdentifier:[[NSUUID UUID] UUIDString]
				content:content
				trigger:nil];
			[center addNotificationRequest:req withCompletionHandler:nil];
		}
	});
}
*/
import "C"

import (
	"sync"
	"unsafe"
)

type darwinStatusItem struct {
	mu      sync.Mutex
	opts    StatusItemOptions
	icon    StatusIcon
	tooltip string
	title   string
	menu    []StatusMenuItem
	item    unsafe.Pointer
	closed  bool
}

func newNativeStatusItem(opts StatusItemOptions) (StatusItem, error) {
	title := C.CString(opts.Title)
	tip := C.CString(opts.Tooltip)
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(tip))
	p := C.ui_status_create(title, tip)
	if p == nil {
		return newStubStatusItem(opts), nil
	}
	return &darwinStatusItem{
		opts:    opts,
		icon:    opts.Icon,
		tooltip: opts.Tooltip,
		title:   opts.Title,
		menu:    copyMenu(opts.Menu),
		item:    p,
	}, nil
}

func nativeStatusItemAvailable() bool { return true }

func (d *darwinStatusItem) SetIcon(icon StatusIcon) error {
	d.mu.Lock()
	d.icon = icon
	d.mu.Unlock()
	return nil
}

func (d *darwinStatusItem) SetTooltip(s string) error {
	d.mu.Lock()
	d.tooltip = s
	p := d.item
	d.mu.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.ui_status_set_tip(p, cs)
	return nil
}

func (d *darwinStatusItem) SetTitle(s string) error {
	d.mu.Lock()
	d.title = s
	p := d.item
	d.mu.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.ui_status_set_title(p, cs)
	return nil
}

func (d *darwinStatusItem) SetMenu(items []StatusMenuItem) error {
	d.mu.Lock()
	d.menu = copyMenu(items)
	d.mu.Unlock()
	return nil
}

func (d *darwinStatusItem) Notify(n Notification) error {
	title := C.CString(n.Title)
	body := C.CString(n.Body)
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(body))
	C.ui_status_notify(title, body)
	return nil
}

func (d *darwinStatusItem) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	p := d.item
	d.item = nil
	d.mu.Unlock()
	if p != nil {
		C.ui_status_remove(p)
	}
	return nil
}

func (d *darwinStatusItem) Backend() string { return "appkit" }

func (d *darwinStatusItem) Alive() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.closed && d.item != nil
}
