# Status item (system tray)

**v0.16.0** adds `StatusItem`: a cross-platform tray / menu-bar /
notification-area icon plus a desktop toast. Mail dogfoods it; any
uitoolkit app can open one.

**v0.16.2** Mail tray: `IconMail` (not `IconInfo`); Wayland Show/Raise
remaps the toplevel after close-to-tray and activates via
`xdg_activation_v1` when the compositor has it. The tray **context
menu is a toolkit `PopupMenu`** (Office XP chrome), not Plasma’s
native dbusmenu. Left-click is still SNI `Activate` → Show Mail.
Callbacks run on the UI thread.

**v0.16.1** fixes a startup panic on KDE Plasma. `com.canonical.dbusmenu.GetLayout`
must return D-Bus type `(ia{sv}av)` — children are an array of variants,
not a recursive Go struct. v0.16.0 exported `Children []dbusMenuLayout`,
so godbus `getSignature` panicked (`container nesting too deep`) when
`org.kde.StatusNotifierWatcher` called `GetLayout` after
`RegisterStatusNotifierItem`. The UI process now also recovers tray
export / register failures and keeps the window on a stub item.

```go
item, _ := app.NewStatusItem(uitoolkit.StatusItemOptions{
    ID:      "myapp",
    Title:   "My App",
    Tooltip: "My App",
    Icon:    uitoolkit.StatusIconFromTool(uitoolkit.IconMail, app.Look(), 22),
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
| `StatusItemOptions` | ID, title, tooltip, icon, menu, `OnClick`, `OnNotifyClick`, `OnMenu`, `Stub` |
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
| Context menu | SNI `ContextMenu(x,y)` → toolkit `PopupMenu` | Same host (right-click) |
| dbusmenu stub | `com.canonical.dbusmenu` at `/MenuBar` (empty `Menu=/`) | Hosts that probe the iface |
| Toast | `org.freedesktop.Notifications` | notification daemon (almost always present) |

**Tray menu chrome**

Linux SNI `Menu` is advertised as `/` with `ItemIsMenu=false`. Plasma
then calls `ContextMenu` instead of drawing a native dbusmenu. The
app opens the same `PopupMenu` / `MenuItem` path as in-window context
menus (gutter icons, checks, Look fonts). Left-click stays `Activate`.

| Environment | Icon | Toolkit popup | Notes |
| --- | --- | --- | --- |
| KDE Plasma | yes (SNI) | yes (`ContextMenu`) | Native dbusmenu is **not** used for chrome |
| GNOME 45+ | AppIndicator extension | only if the host calls `ContextMenu` | Extension often expects dbusmenu; right-click may no-op |
| Sway / Hyprland + Waybar | tray module | if Waybar sends `ContextMenu` | dbusmenu-only modules get the stub |
| Xfce / Cinnamon / MATE | usually SNI | if the host calls `ContextMenu` | |
| Xlibre | same as X11 | same | |
| Windows | `Shell_NotifyIcon` | yes (`OnMenu` on right-click) when a window exists | No Win32 `HMENU` |
| macOS (`CGO`) | `NSStatusItem` | not wired | OS menu-bar click is native; toolkit popup not used |
| No session bus / SSH | stub | stub | |

**Compositor / desktop (toasts):**

| Environment | Toast |
| --- | --- |
| KDE Plasma | yes |
| GNOME 45+ | yes (extension only needed for the icon) |
| Sway / Hyprland | Mako / Dunst / fnott |
| Xfce / Cinnamon / MATE | yes |
| Xlibre | yes |
| No session bus / SSH | stub |

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
On Wayland, Hide drops the `xdg_toplevel` role (after `set_minimized`);
Show remaps it and requests `xdg_activation_v1` when available.

### Windows

`GOOS=windows`: `Shell_NotifyIconW` (notification area) plus `NIF_INFO`
balloons. Left-click fires `OnClick`; right-click fires `OnMenu` (toolkit
`PopupMenu` when a toolkit window exists). Balloon click fires
`OnNotifyClick`. Windowing (HWND app windows) is still a stub — only
the tray landed.

```bat
go build ./cmd/mailclientui
mailclientui.exe
```

### macOS

`GOOS=darwin` **and** `CGO_ENABLED=1`: `NSStatusItem` on the menu bar
and `NSUserNotification` toasts. The OS owns the menu-bar click;
toolkit `PopupMenu` is not used on macOS. Without CGO the item is a stub.
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
