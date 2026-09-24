package platform

import (
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// StatusItem is a cross-platform tray / menu-bar / notification-area icon
// (QSystemTrayIcon / KStatusNotifierItem / Electron Tray).
//
//	Linux:   StatusNotifierItem (KDE / AppIndicator / Waybar) + freedesktop
//	         desktop notifications. HostMenu (default) exports dbusmenu at
//	         /MenuBar; ToolkitMenu uses Menu=/NO_DBUSMENU + ContextMenu.
//	         X11, Xlibre, and Wayland share this path.
//	Windows: Shell_NotifyIcon (notification area + balloon).
//	macOS:   NSStatusItem (menu bar) when built with CGO.
//
// Headless, CGO-less macOS, or a missing session bus fall back to a no-op
// stub. Set UITK_TRAY=fake for tests, UITK_TRAY=stub to force the no-op.
type StatusItem interface {
	SetIcon(StatusIcon) error
	SetTooltip(string) error
	SetTitle(string) error
	SetMenu([]StatusMenuItem) error
	Notify(Notification) error
	Close() error
	// Backend is "sni", "win32", "appkit", "fdo-notify", "fake", or "stub".
	Backend() string
	// Alive reports a live host (icon and/or notification daemon). Stubs are not.
	Alive() bool
}

// StatusIcon is a tray pictogram. First non-empty field wins: Image, Path,
// then Name (freedesktop icon theme, Windows resource, or macOS template).
type StatusIcon struct {
	Name  string
	Path  string
	Image *paintengine2d.Image
}

// StatusMenuItem is one tray context-menu row. Apps typically map
// widgets.MenuItem into this.
type StatusMenuItem struct {
	Text      string
	Disabled  bool
	Separator bool
	Checked   bool
	Icon      style.ToolIcon
	// Submenu makes this row a cascade parent: the host (Plasma,
	// AppIndicator, Waybar) or the toolkit PopupMenu opens a child menu
	// from it. Nesting has no depth limit here; hosts have their own.
	//
	// A parent is not a command. When Submenu is non-empty the row's own
	// OnClick is never called — a host that opens a child menu sends no
	// "clicked" for the parent row, and an app that relied on both would
	// do different things on different desktops. Every backend enforces
	// that (see menuItemClickable), so setting both is not a per-host
	// accident but a row whose OnClick is dead on all of them.
	//
	// Separator wins over Submenu: a rule is not a menu, so children on a
	// Separator row are dropped rather than exported as an unreachable
	// branch.
	Submenu []StatusMenuItem
	OnClick func()
}

// StatusMenuChrome selects who draws the tray context menu.
//
// HostMenu (default) matches QSystemTrayIcon, KStatusNotifierItem, and
// Electron Tray: Linux exports a real com.canonical.dbusmenu at the SNI
// Menu path and the host draws the menu. ToolkitMenu is opt-in Office XP
// chrome via a reused toolkit PopupMenu and SNI Menu=/NO_DBUSMENU.
type StatusMenuChrome int

const (
	// HostMenu lets the desktop draw a native tray menu (Linux dbusmenu).
	// Windows still uses the toolkit popup (no HMENU); macOS uses the OS
	// menu-bar item. This is the default and the robust Plasma path.
	HostMenu StatusMenuChrome = iota
	// ToolkitMenu shows a uitoolkit PopupMenu (Office XP chrome). Linux
	// sets Menu=/NO_DBUSMENU so Plasma calls ContextMenu instead of
	// importing dbusmenu.
	ToolkitMenu
)

// StatusItemOptions configure [NewStatusItem].
type StatusItemOptions struct {
	// ID is the D-Bus / app id (default "uitoolkit").
	ID string
	// Title is the status item label (macOS extra text; SNI Title).
	Title   string
	Tooltip string
	Icon    StatusIcon
	Menu    []StatusMenuItem
	// MenuChrome is HostMenu (default, native host chrome) or ToolkitMenu
	// (opt-in toolkit PopupMenu). See [StatusMenuChrome].
	MenuChrome StatusMenuChrome
	// ItemIsMenu, when true, advertises SNI ItemIsMenu so left-click is
	// treated as “menu only” by hosts (they draw dbusmenu). Activate then
	// does not fire OnClick. ToolkitMenu still routes Activate to OnMenu.
	// Default false: left-click is Activate → OnClick.
	ItemIsMenu bool
	// OnClick is the primary (left) click. Apps typically Show/Raise a window.
	OnClick func()
	// OnNotifyClick is invoked when the user activates a desktop notification.
	OnNotifyClick func()
	// OnMenu is the context / right-click handler used by ToolkitMenu
	// (and by Windows, which has no dbusmenu). Application.NewStatusItem
	// sets this to ShowStatusMenu when toolkit chrome is selected.
	// x, y are SNI root coordinates; 0,0 means "near the window".
	OnMenu func(x, y int32)
	// Dispatch runs tray callbacks (Activate, dbusmenu Event, notify
	// click) on the UI thread. Application.NewStatusItem sets this to
	// Application.Post.
	//
	// Leaving it nil is only safe for callbacks that touch nothing the
	// UI goroutine touches: they then run on the D-Bus (or Win32 pump)
	// thread, serialized against other tray callbacks but not against
	// the toolkit.
	Dispatch func(func())
	// Stub forces the no-op backend (headless apps).
	Stub bool
}

// statusCallbackMu serializes tray callbacks that have no Dispatch hook.
//
// D-Bus method calls (Activate, ContextMenu, dbusmenu Event) and the
// Win32 pump both deliver on their own threads, so without Dispatch two
// callbacks could touch application state concurrently. Serializing them
// does not make the callback run on the UI goroutine — only Dispatch does
// that — but it removes the data race between two tray callbacks.
var statusCallbackMu sync.Mutex

func invokeStatus(dispatch func(func()), fn func()) {
	if fn == nil {
		return
	}
	if dispatch != nil {
		dispatch(fn)
		return
	}
	statusCallbackMu.Lock()
	defer statusCallbackMu.Unlock()
	fn()
}

func trayDebug(format string, args ...any) {
	if os.Getenv("UITK_TRAY_DEBUG") == "" {
		return
	}
	log.Printf("uitk tray: "+format, args...)
}

// Notification is a tray balloon / desktop toast.
type Notification struct {
	Title string
	Body  string
	Icon  StatusIcon
	// OnClick overrides StatusItemOptions.OnNotifyClick for this toast.
	OnClick func()
}

// NewStatusItem opens a tray icon. It never panics: unsupported hosts
// and tray-register / export failures return a stub whose methods
// succeed as no-ops so the app window still opens.
func NewStatusItem(opts StatusItemOptions) (item StatusItem, err error) {
	defer func() {
		if recover() != nil {
			item = newStubStatusItem(opts)
			err = nil
		}
	}()
	if opts.ID == "" {
		opts.ID = "uitoolkit"
	}
	if opts.Title == "" {
		opts.Title = "uitoolkit"
	}
	if opts.Stub {
		return newStubStatusItem(opts), nil
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("UITK_TRAY"))) {
	case "fake":
		return newFakeStatusItem(opts), nil
	case "stub", "off", "none":
		return newStubStatusItem(opts), nil
	}
	item, err = newNativeStatusItem(opts)
	if err != nil || item == nil {
		return newStubStatusItem(opts), nil
	}
	return item, nil
}

// StatusItemAvailable reports whether a native tray or notification host
// is likely present (session bus, notification area, or AppKit).
func StatusItemAvailable() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("UITK_TRAY"))) {
	case "fake":
		return true
	case "stub", "off", "none":
		return false
	}
	return nativeStatusItemAvailable()
}

// copyMenu snapshots a menu, children included. The copy is deep in the
// slices and shallow in the callbacks (that is the point: a backend keeps
// the app's OnClick), so a caller that keeps mutating the slice it passed
// to SetMenu cannot change a menu a host is already showing.
func copyMenu(in []StatusMenuItem) []StatusMenuItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]StatusMenuItem, len(in))
	copy(out, in)
	for i := range out {
		out[i].Submenu = copyMenu(out[i].Submenu)
	}
	return out
}

// statusIconName is the freedesktop / SNI IconName. Empty Name falls
// back to a generic theme icon so hosts that ignore IconPixmap still
// show something (not a Mail-specific name).
func statusIconName(icon StatusIcon) string {
	if icon.Name != "" {
		return icon.Name
	}
	return "application-default-icon"
}

// menuItemChildren is the rows a menu row actually opens: a separator is
// a rule and never a parent, whatever the app put in Submenu.
func menuItemChildren(it StatusMenuItem) []StatusMenuItem {
	if it.Separator {
		return nil
	}
	return it.Submenu
}

// menuItemClickable reports whether a row's OnClick may fire. A cascade
// parent is not a command ([StatusMenuItem.Submenu]), so it never is.
func menuItemClickable(it StatusMenuItem) bool {
	return !it.Separator && !it.Disabled && it.OnClick != nil && len(menuItemChildren(it)) == 0
}

// menuRowsEqual compares the visible shape of two menus, children
// included, so SetMenu dedupes only menus a host would draw identically.
// Callbacks are deliberately not compared: funcs are not comparable, and
// a changed closure over the same rows draws the same menu.
func menuRowsEqual(a, b []StatusMenuItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Text != b[i].Text || a[i].Disabled != b[i].Disabled ||
			a[i].Separator != b[i].Separator || a[i].Checked != b[i].Checked ||
			a[i].Icon != b[i].Icon {
			return false
		}
		if !menuRowsEqual(menuItemChildren(a[i]), menuItemChildren(b[i])) {
			return false
		}
	}
	return true
}

// HostMenuNative reports whether this OS draws a native tray menu for
// HostMenu (Linux StatusNotifier dbusmenu). Other OSes keep the existing
// toolkit or AppKit path.
func HostMenuNative() bool {
	return runtime.GOOS == "linux"
}
