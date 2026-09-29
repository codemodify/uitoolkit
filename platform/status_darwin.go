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
// The tray menu.
//
// SetMenu used to store the rows and show nothing, so the macOS tray was
// an icon that did nothing when clicked. NSStatusItem takes an NSMenu
// and draws it itself — there is no protocol to export and no host to
// negotiate with, unlike the Linux dbusmenu — so what is needed is to
// build one and to get the click back into Go.
//
// The whole menu is built in **one** hop to the main thread, from a flat
// array Go hands over. Building it row by row would mean a
// dispatch_sync per row, and a dispatch_sync to the main queue from a
// goroutine deadlocks whenever nothing is draining that queue — which is
// every moment before NSApp's run loop starts, and every test. One
// async block waits for nothing and cannot deadlock at all.
//
// The array is pre-order: each row says how many children follow it, so
// a submenu is a run of rows rather than a second allocation. C owns the
// array and frees it, because the block outlives the Go call.
//
// Each clickable row carries its id in the NSMenuItem's tag, and one
// shared target turns the action into a call to uitkStatusMenuClick. The
// tag is an NSInteger, 64 bits on every Mac this runs on, so the id is
// the tag and nothing has to be looked up on the C side.

typedef struct {
	char *title;
	int separator;
	int disabled;
	int checked;
	int nkids;
	unsigned long long id;
} UitkMenuRow;

@interface UitkMenuTarget : NSObject
- (void)fire:(id)sender;
@end

extern void uitkStatusMenuClick(unsigned long long id);

@implementation UitkMenuTarget
- (void)fire:(id)sender {
	NSMenuItem *it = (NSMenuItem *)sender;
	uitkStatusMenuClick((unsigned long long)it.tag);
}
@end

static UitkMenuTarget *uitk_menu_target(void) {
	static UitkMenuTarget *t = nil;
	static dispatch_once_t once;
	dispatch_once(&once, ^{ t = [[UitkMenuTarget alloc] init]; });
	return t;
}

// uitk_menu_fill adds rows[i..] to m until n rows have been added at
// this level, and returns the index after the last one it consumed.
static int uitk_menu_fill(NSMenu *m, UitkMenuRow *rows, int i, int n, int total) {
	for (int k = 0; k < n && i < total; k++) {
		UitkMenuRow r = rows[i++];
		if (r.separator) {
			[m addItem:[NSMenuItem separatorItem]];
			continue;
		}
		NSString *t = r.title ? [NSString stringWithUTF8String:r.title] : @"";
		NSMenuItem *it = [[NSMenuItem alloc] initWithTitle:t action:nil keyEquivalent:@""];
		it.enabled = r.disabled ? NO : YES;
		it.state = r.checked ? NSControlStateValueOn : NSControlStateValueOff;
		if (r.nkids > 0) {
			NSMenu *sub = [[NSMenu alloc] init];
			sub.autoenablesItems = NO;
			sub.title = t;
			i = uitk_menu_fill(sub, rows, i, r.nkids, total);
			it.submenu = sub;
		} else if (!r.disabled && r.id != 0) {
			it.target = uitk_menu_target();
			it.action = @selector(fire:);
			it.tag = (NSInteger)r.id;
		}
		[m addItem:it];
	}
	return i;
}

// ui_status_set_menu builds the menu and hands it to the status item,
// which shows it on any click — so the button needs no action of its
// own. It takes ownership of rows and of every title in it.
//
// Assigning item.menu releases whatever was there, so replacing a menu
// needs no bookkeeping on this side; what Go still has to do is forget
// the old menu's click handlers.
static void ui_status_set_menu(void *p, UitkMenuRow *rows, int n) {
	if (!p) {
		for (int i = 0; i < n; i++) free(rows[i].title);
		free(rows);
		return;
	}
	ui_main_async(^{
		@autoreleasepool {
			NSStatusItem *item = (__bridge NSStatusItem *)p;
			if (n == 0) {
				item.menu = nil;
			} else {
				NSMenu *m = [[NSMenu alloc] init];
				// The toolkit decides what is enabled; without this
				// AppKit asks a validator that does not exist and greys
				// everything.
				m.autoenablesItems = NO;
				uitk_menu_fill(m, rows, 0, n, n);
				item.menu = m;
			}
			for (int i = 0; i < n; i++) free(rows[i].title);
			free(rows);
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
	// menuIDs are the click handlers registered for the menu the status
	// item is showing. The NSMenu itself is AppKit's: assigning a new
	// one releases the old, so there is nothing to track on this side.
	menuIDs []uint64
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
	d := &darwinStatusItem{
		opts:    opts,
		icon:    opts.Icon,
		tooltip: opts.Tooltip,
		title:   opts.Title,
		menu:    copyMenu(opts.Menu),
		item:    p,
	}
	if len(d.menu) > 0 {
		d.applyMenu(copyMenu(d.menu))
	}
	return d, nil
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

// SetMenu builds a real NSMenu and hands it to the status item, which
// is what makes the macOS tray usable at all: it used to store the rows
// and show nothing, so the icon sat there and did nothing when clicked.
func (d *darwinStatusItem) SetMenu(items []StatusMenuItem) error {
	d.mu.Lock()
	d.menu = copyMenu(items)
	rows := copyMenu(d.menu)
	d.mu.Unlock()
	d.applyMenu(rows)
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
	ids := d.menuIDs
	d.item, d.menuIDs = nil, nil
	d.mu.Unlock()
	if p != nil {
		C.ui_status_remove(p)
	}
	// The menu goes with the item — removing the status item releases it
	// — and its click handlers go with it: a closure held for a tray
	// that is gone is a leak of whatever the application captured in it.
	trayMenuForget(ids)
	return nil
}

func (d *darwinStatusItem) Backend() string { return "appkit" }

func (d *darwinStatusItem) Alive() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.closed && d.item != nil
}

// flattenTrayMenu turns the rows into the pre-order array the C side
// builds from, and registers a click handler for each row that has one.
//
// A row with a submenu is a parent and not a command — every backend
// enforces that, see [StatusMenuItem.Submenu] — so it gets no id, and a
// separator with children is a separator.
func flattenTrayMenu(items []StatusMenuItem, out *[]C.UitkMenuRow, ids *[]uint64) int {
	n := 0
	for _, it := range items {
		n++
		if it.Separator {
			*out = append(*out, C.UitkMenuRow{separator: 1})
			continue
		}
		row := C.UitkMenuRow{
			title:    C.CString(it.Text),
			disabled: cmenuBool(it.Disabled),
			checked:  cmenuBool(it.Checked),
		}
		at := len(*out)
		*out = append(*out, row)
		switch {
		case len(it.Submenu) > 0:
			kids := flattenTrayMenu(it.Submenu, out, ids)
			(*out)[at].nkids = C.int(kids)
			n += kids
		case it.OnClick != nil && !it.Disabled:
			id := trayMenuRegister(it.OnClick)
			*ids = append(*ids, id)
			(*out)[at].id = C.ulonglong(id)
		}
	}
	return n
}

func cmenuBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// applyMenu replaces the status item's menu, and drops the previous
// one's handlers with it.
//
// The C side owns the array and every string in it from the moment it is
// handed over: the block that builds the menu runs later, on the main
// thread, and freeing here would free them out from under it.
func (d *darwinStatusItem) applyMenu(items []StatusMenuItem) {
	var rows []C.UitkMenuRow
	var ids []uint64
	flattenTrayMenu(items, &rows, &ids)

	d.mu.Lock()
	oldIDs := d.menuIDs
	d.menuIDs = ids
	p := d.item
	d.mu.Unlock()

	if len(rows) == 0 {
		C.ui_status_set_menu(p, nil, 0)
	} else {
		buf := (*C.UitkMenuRow)(C.calloc(C.size_t(len(rows)), C.size_t(unsafe.Sizeof(rows[0]))))
		copy(unsafe.Slice(buf, len(rows)), rows)
		C.ui_status_set_menu(p, buf, C.int(len(rows)))
	}
	trayMenuForget(oldIDs)
}

// trayMenuRow is one flattened row as a test sees it: the fields that
// decide the menu's shape, without the C types.
type trayMenuRow struct {
	Text      string
	Separator bool
	Disabled  bool
	Checked   bool
	Kids      int
	ID        uint64
}

// flattenTrayMenuForTest is [flattenTrayMenu] with the C array turned
// back into Go, so a test can assert the shape without reaching into
// cgo memory. It frees the strings it allocated, since nothing here
// hands them to AppKit.
func flattenTrayMenuForTest(items []StatusMenuItem) ([]trayMenuRow, []uint64, int) {
	var rows []C.UitkMenuRow
	var ids []uint64
	n := flattenTrayMenu(items, &rows, &ids)
	out := make([]trayMenuRow, len(rows))
	for i, r := range rows {
		out[i] = trayMenuRow{
			Text:      C.GoString(r.title),
			Separator: r.separator != 0,
			Disabled:  r.disabled != 0,
			Checked:   r.checked != 0,
			Kids:      int(r.nkids),
			ID:        uint64(r.id),
		}
		if r.title != nil {
			C.free(unsafe.Pointer(r.title))
		}
	}
	return out, ids, n
}
