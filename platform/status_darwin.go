//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
#import <AppKit/AppKit.h>

static void *ui_status_create(const char *title, const char *tip) {
	@autoreleasepool {
		NSStatusItem *item = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
		item.button.title = title ? [NSString stringWithUTF8String:title] : @"";
		if (tip) {
			item.button.toolTip = [NSString stringWithUTF8String:tip];
		}
		item.button.image = [NSImage imageNamed:NSImageNameApplicationIcon];
		item.button.image.template = YES;
		return (void *)CFBridgingRetain(item);
	}
}

static void ui_status_set_title(void *p, const char *title) {
	@autoreleasepool {
		NSStatusItem *item = (__bridge NSStatusItem *)p;
		item.button.title = title ? [NSString stringWithUTF8String:title] : @"";
	}
}

static void ui_status_set_tip(void *p, const char *tip) {
	@autoreleasepool {
		NSStatusItem *item = (__bridge NSStatusItem *)p;
		item.button.toolTip = tip ? [NSString stringWithUTF8String:tip] : @"";
	}
}

static void ui_status_remove(void *p) {
	@autoreleasepool {
		NSStatusItem *item = CFBridgingRelease(p);
		[[NSStatusBar systemStatusBar] removeStatusItem:item];
	}
}

static void ui_status_notify(const char *title, const char *body) {
	@autoreleasepool {
		NSUserNotification *n = [NSUserNotification new];
		n.title = title ? [NSString stringWithUTF8String:title] : @"uitoolkit";
		n.informativeText = body ? [NSString stringWithUTF8String:body] : @"";
		[[NSUserNotificationCenter defaultUserNotificationCenter] deliverNotification:n];
	}
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
