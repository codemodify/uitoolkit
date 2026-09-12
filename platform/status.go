package platform

import (
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// StatusItem is a cross-platform tray / menu-bar / notification-area icon.
//
//	Linux:   StatusNotifierItem (KDE / AppIndicator / Waybar) + freedesktop
//	         desktop notifications. X11, Xlibre, and Wayland share this path.
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
	OnClick   func()
}

// StatusItemOptions configure [NewStatusItem].
type StatusItemOptions struct {
	// ID is the D-Bus / app id (default "uitoolkit").
	ID string
	// Title is the status item label (macOS extra text; SNI Title).
	Title   string
	Tooltip string
	Icon    StatusIcon
	Menu    []StatusMenuItem
	// OnClick is the primary (left) click. Apps typically Show/Raise a window.
	OnClick func()
	// OnNotifyClick is invoked when the user activates a desktop notification.
	OnNotifyClick func()
	// Dispatch runs tray callbacks (Activate, dbusmenu Event, notify
	// click) on the UI thread. Application.NewStatusItem sets this to
	// Application.Post. If nil, callbacks run on the D-Bus goroutine.
	Dispatch func(func())
	// Stub forces the no-op backend (headless apps).
	Stub bool
}

func invokeStatus(dispatch func(func()), fn func()) {
	if fn == nil {
		return
	}
	if dispatch != nil {
		dispatch(fn)
		return
	}
	fn()
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

func copyMenu(in []StatusMenuItem) []StatusMenuItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]StatusMenuItem, len(in))
	copy(out, in)
	return out
}
