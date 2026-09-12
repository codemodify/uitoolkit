# Status item (system tray)

**v0.16.0** adds `StatusItem`: a cross-platform tray / menu-bar /
notification-area icon plus a desktop toast. Mail dogfoods it; any
uitoolkit app can open one.

```go
item, _ := app.NewStatusItem(uitoolkit.StatusItemOptions{
    ID:      "myapp",
    Title:   "My App",
    Tooltip: "My App",
    Icon:    uitoolkit.StatusIconFromTool(uitoolkit.IconInfo, app.Look(), 22),
    Menu: uitoolkit.StatusMenuFromItems([]*uitoolkit.MenuItem{
        uitoolkit.NewMenuItem("Show", win.Raise),
        uitoolkit.MenuSep(),
        uitoolkit.NewMenuItem("Quit", app.Quit),
    }),
    OnClick: win.Raise,
})
_ = item.Notify(uitoolkit.Notification{Title: "Hello", Body: "Ready."})
```

Headless apps and missing hosts get a **stub** (methods succeed, no
window chrome). `UITK_TRAY=fake` records calls for tests.
`UITK_TRAY=stub` forces the no-op.

## API

| Type / func | Role |
| --- | --- |
| `StatusItem` | `SetIcon` / `SetTooltip` / `SetTitle` / `SetMenu` / `Notify` / `Close` / `Backend` / `Alive` |
| `StatusItemOptions` | ID, title, tooltip, icon, menu, `OnClick`, `OnNotifyClick`, `Stub` |
| `StatusIcon` | `Image` (paintengine2d), `Path` (PNG), or `Name` (freedesktop / theme) |
| `StatusMenuItem` | text, separator, check, `OnClick` — map from `widgets.MenuItem` |
| `Notification` | title + body (+ optional icon / click) |
| `Application.NewStatusItem` | owns the item; a live item keeps `Run` after the last window closes |
| `Window.SetCloseHides` | WM close hides (close-to-tray) instead of destroy |
| `Window.Show` / `Hide` / `Raise` / `Visible` | map / unmap / activate |

`Backend()` is `sni`, `win32`, `appkit`, `fdo-notify`, `fake`, or `stub`.

## Backends

### Linux (X11, Xlibre, Wayland)

One D-Bus path for every Linux display server. No extra CGO.

| Piece | Bus name | Needs |
| --- | --- | --- |
| Tray icon | `org.kde.StatusNotifierItem` + `RegisterStatusNotifierItem` on `org.kde.StatusNotifierWatcher` | A StatusNotifier host |
| Context menu | `com.canonical.dbusmenu` | Same host |
| Toast | `org.freedesktop.Notifications` | notification daemon (almost always present) |

**Compositor / desktop:**

| Environment | Tray icon | Toast |
| --- | --- | --- |
| KDE Plasma | yes (native SNI) | yes |
| GNOME 45+ | install [AppIndicator](https://extensions.gnome.org/extension/615/appindicator-support/) (or `gnome-shell-extension-appindicator`) | yes without the extension |
| Sway / Hyprland | Waybar / fuzzel / similar `tray` module | Mako / Dunst / fnott |
| Xfce / Cinnamon / MATE | usually yes (SNI or Ayatana) | yes |
| Xlibre | same as X11 (SNI over the session bus) | yes |
| No session bus / SSH | stub | stub |

```bash
# Debian/Ubuntu
sudo apt install gnome-shell-extension-appindicator   # GNOME tray
# or
sudo apt install xfce4-indicator-plugin               # Xfce

# Session bus is required (normal desktop login).
echo "$DBUS_SESSION_BUS_ADDRESS"
```

XEmbed `_NET_SYSTEM_TRAY` is **not** implemented. Portal
`org.freedesktop.portal.Notification` is not used (SNI + fdo cover
the same toasts outside a sandbox).

`Window.Raise` on X11 maps the window and sends `_NET_ACTIVE_WINDOW`.
On Wayland, hide is `xdg_toplevel.set_minimized`; restore is
compositor-dependent (clicking the tray still `Show`s / invalidates).

### Windows

`GOOS=windows`: `Shell_NotifyIconW` (notification area) plus `NIF_INFO`
balloons. Left-click fires `OnClick`; balloon click fires
`OnNotifyClick`. Windowing (HWND app windows) is still a stub — only
the tray landed.

```bat
go build ./cmd/mailclientui
mailclientui.exe
```

### macOS

`GOOS=darwin` **and** `CGO_ENABLED=1`: `NSStatusItem` on the menu bar
and `NSUserNotification` toasts. Without CGO the item is a stub.
AppKit windows are still a stub.

```bash
CGO_ENABLED=1 go build ./cmd/mailclientui
```

## Build / run

```bash
# Tests (always stub/fake; no bus required)
CGO_ENABLED=0 go test ./...
UITK_TRAY=fake go test ./platform ./app ./internal/mail

# Linux desktop
go run ./cmd/mailclientui          # UI owns the tray
go run ./examples/mail             # in-process daemon + UI

# Force backends
UITK_TRAY=stub go run ./cmd/mailclientui
UITK_TRAY=fake go run ./cmd/mailclientui   # no icon; recorded API
```

Dependency: [`github.com/godbus/dbus/v5`](https://github.com/godbus/dbus) on
Linux only (pure Go). Windows and macOS use OS APIs.
