# Status item (system tray)

`StatusItem` is a first-class toolkit API in the same role as Qt
`QSystemTrayIcon`, KDE `KStatusNotifierItem`, and Electron `Tray`: a
cross-platform notification-area / menu-bar icon, a context menu, and a
desktop toast. An application only ever calls toolkit APIs
(`app.NewStatusItem` / `platform.StatusItem`).

**v0.18.1** hardens the v0.17 HostMenu path: empty `GetLayout` stays a
valid root, `SetMenu` still dedupes `LayoutUpdated`, SNI `IconName`
falls back to `application-default-icon` (toasts no longer hardcode
`mail-unread`), `ItemIsMenu` is menu-only (no extra `OnClick`),
`SecondaryActivate` does not raise on HostMenu, dbusmenu exports
`Version=3`, and `Event` ignores separator / disabled rows.

**v0.17.0** makes the Linux path robust. The default is **HostMenu**:
advertise `Menu=/MenuBar` with a real `com.canonical.dbusmenu` layout
and let Plasma / AppIndicator / Waybar draw the menu. Opt-in
**ToolkitMenu** uses the official KDE sentinel `Menu=/NO_DBUSMENU` and a
reused toolkit `PopupMenu` (Office XP chrome).

## Why HostMenu is the default

Hosts expect a real dbusmenu at the SNI `Menu` path and **draw the menu
themselves**. Electron’s docs (`tray.setContextMenu`) and field fixes
are explicit: `right-click` + a custom popup is unreliable on Linux
SNI.

Plasma (`statusnotifieritemsource.cpp`):

| SNI `Menu` | Plasma |
| --- | --- |
| `/MenuBar` (or any real object path except the sentinel) | Import dbusmenu; host draws the menu |
| `/NO_DBUSMENU` | Official KDE / `KStatusNotifierItem` hack: **disable** dbusmenu and call SNI `ContextMenu(x,y)` |
| `/` | Spec “empty” path — **not** the KDE sentinel. Not a reliable ContextMenu fallback |

v0.16.4 advertised `Menu=/` and opened a new top-level toolkit window
on every right-click. First click was delayed (`Application.Post` often
waited out the 100ms tray poll on Wayland); the second click often did
nothing (FocusOut / unmap / create races left an invisible surface over
the panel).

## HostMenu vs ToolkitMenu

```go
item, _ := app.NewStatusItem(uitoolkit.StatusItemOptions{
    ID:         "myapp",
    Title:      "My App",
    Tooltip:    "My App",
    MenuChrome: uitoolkit.HostMenu, // default; omit for the same effect
    Icon:       uitoolkit.StatusIconFromTool(uitoolkit.IconMail, app.Look(), 22),
    Menu: uitoolkit.StatusMenuFromItems([]*uitoolkit.MenuItem{
        uitoolkit.NewMenuItem("Show", win.Raise),
        uitoolkit.MenuSep(),
        uitoolkit.NewMenuItem("Quit", app.Quit),
    }),
    OnClick: win.Raise,
})
_ = item.Notify(uitoolkit.Notification{Title: "Hello", Body: "Ready."})
```

| | **HostMenu** (default) | **ToolkitMenu** |
| --- | --- | --- |
| Linux `Menu` | `/MenuBar` — real dbusmenu rows | `/NO_DBUSMENU` exactly |
| Who draws | Plasma / GNOME AppIndicator / Waybar | Toolkit `PopupMenu` |
| `ItemIsMenu` | Advertised; left-click is host menu only (`Activate` does not `OnClick`) | Advertised; `Activate` → `OnMenu` |
| Left-click | SNI `Activate` → `OnClick` (show window) unless `ItemIsMenu` | Same unless `ItemIsMenu` |
| Middle-click | Ignored (some hosts also fire this on right-click) | `SecondaryActivate` → reused popup |
| Right-click | Host menu every time | `ContextMenu` / `SecondaryActivate` → reused popup |
| `SetMenu` | `LayoutUpdated` (deduped; no spam) | Updates next popup |
| UI thread | dbusmenu `Event("clicked")` via `Dispatch` / `Post` | `OnMenu` via `Dispatch` / `Post` |

`GetLayout` stays **flat** (`(ia{sv}av)` children as `[]dbus.Variant` of
a distinct leaf type). Do not nest the same Go struct — that is the
v0.16.1 godbus `container nesting too deep` panic.

If `org.kde.StatusNotifierWatcher` restarts, the item watches
`NameOwnerChanged` and calls `RegisterStatusNotifierItem` again.

### ToolkitMenu robustness

Set `MenuChrome: uitoolkit.ToolkitMenu` to dogfood Office XP chrome:

- **One** status-menu window, reused (`Hide` / `Show` + rebind). No
  create/destroy per click.
- FocusOut-dismiss is armed only after ~180ms so map/focus churn does
  not kill the first frame.
- Dismiss **Hides and unmaps** fully. Never leave an invisible surface
  over the panel.
- Prefer screen `(x,y)` on **X11**. On **Wayland** the compositor owns
  toplevel placement; the menu is still mapped and visible every time.
- `Application.Post` wakes the display promptly (Linux `eventfd` polled
  with `wl_display`; X11 helper property). The 100ms tray wait cap is a
  fallback, not the wake path.

## API

| Type / func | Role |
| --- | --- |
| `StatusItem` | `SetIcon` / `SetTooltip` / `SetTitle` / `SetMenu` / `Notify` / `Close` / `Backend` / `Alive` |
| `StatusItemOptions` | ID, title, tooltip, icon, menu, `MenuChrome`, `ItemIsMenu`, `OnClick`, `OnNotifyClick`, `OnMenu`, `Stub` |
| `StatusMenuChrome` | `HostMenu` (default) or `ToolkitMenu` |
| `StatusIcon` | `Image` (paintengine2d), `Path` (PNG), or `Name` (freedesktop / theme) |
| `StatusMenuItem` | text, separator, check, icon, `OnClick` — map from `widgets.MenuItem` |
| `Notification` | title + body (+ optional icon / click) |
| `Application.NewStatusItem` | owns the item; a live item keeps `Run` after the last window closes |
| `Window.SetCloseHides` | WM close hides (close-to-tray) instead of destroy |
| `Window.Show` / `Hide` / `Raise` / `Visible` | map / unmap / activate |

`Backend()` is `sni`, `win32`, `appkit`, `fdo-notify`, `fake`, or `stub`.

Headless apps and missing hosts get a **stub**. `UITK_TRAY=fake` records
calls for tests. `UITK_TRAY=stub` forces the no-op.
`UITK_TRAY_DEBUG=1` logs Activate / ContextMenu / dbusmenu / Post.

## Per-OS behavior

### Linux (X11, Xlibre, Wayland)

One D-Bus path for every Linux display server. No extra CGO.

| Piece | Bus name | HostMenu | ToolkitMenu |
| --- | --- | --- | --- |
| Tray icon | `org.kde.StatusNotifierItem` + `RegisterStatusNotifierItem` | same | same |
| Context menu | `com.canonical.dbusmenu` at `/MenuBar` | host draws | unused |
| SNI `Menu` | object path | `/MenuBar` | `/NO_DBUSMENU` |
| Fallback | SNI `ContextMenu(x,y)` | not used | toolkit popup |
| Toast | `org.freedesktop.Notifications` | same | same |

| Environment | Icon | HostMenu | ToolkitMenu | Notes |
| --- | --- | --- | --- | --- |
| KDE Plasma | yes (SNI) | native dbusmenu | `ContextMenu` popup | Prefer HostMenu |
| GNOME 45+ | AppIndicator extension | native dbusmenu | only if host calls `ContextMenu` | Extension expects dbusmenu |
| Sway / Hyprland + Waybar | tray module | dbusmenu | if Waybar sends `ContextMenu` | |
| Xfce / Cinnamon / MATE | usually SNI | dbusmenu | if host calls `ContextMenu` | |
| Xlibre | same as X11 | same | same | |
| No session bus / SSH | stub | stub | stub | |

**Wayland:** clients cannot place a toplevel at SNI `(x,y)`. HostMenu
avoids that (the panel draws the menu). ToolkitMenu still shows a
visible popup; the compositor chooses where. Close-to-tray Hide drops
the `xdg_toplevel` role; Show remaps and requests `xdg_activation_v1`
when available.

**X11:** ToolkitMenu places the popup at root `(x,y)` (typically above
a bottom panel). `Window.Raise` sends `_NET_ACTIVE_WINDOW`.

XEmbed `_NET_SYSTEM_TRAY` is **not** implemented.

A notification does not need a tray item: `platform.Notifier`
(`Application.NewNotifier`) sends one with buttons through
`org.freedesktop.portal.Notification` or the notification server, and
brings its clicks back as the app's action names — see
[platform.md](platform.md#portals). A mail client's new-mail notification goes
through it; `StatusItem.Notify` stays for the tray's own toast, and no
longer answers other notifications' clicks.

```bash
# Debian/Ubuntu
sudo apt install gnome-shell-extension-appindicator   # GNOME tray
# or
sudo apt install xfce4-indicator-plugin               # Xfce

echo "$DBUS_SESSION_BUS_ADDRESS"
```

Verify an app's item on abox / Plasma (HostMenu):

```bash
busctl --user get-property \
  "$(busctl --user tree --list | grep -i StatusNotifierItem | head -1)" \
  /StatusNotifierItem org.kde.StatusNotifierItem Menu
# expect: o "/MenuBar"
```

### Windows

`GOOS=windows`: `Shell_NotifyIconW` plus `NIF_INFO` balloons. There is
no Win32 `HMENU` yet, so both HostMenu and ToolkitMenu use `OnMenu` →
toolkit `PopupMenu` when a toolkit window exists. Left-click is
`OnClick`; balloon click is `OnNotifyClick`. HWND app windows are still
a stub — only the tray landed.

```bat
go build ./examples/tour
tour.exe
```

### macOS

`GOOS=darwin` **and** `CGO_ENABLED=1`: `NSStatusItem` on the menu bar
and `NSUserNotification` toasts. The OS owns the menu-bar click;
toolkit `PopupMenu` is not used. Without CGO the item is a stub. AppKit
windows are still a stub.

```bash
CGO_ENABLED=1 go build ./examples/tour
```

## History

**v0.18.1** — HostMenu robustness: empty menu layout, `IconName`
fallback, `ItemIsMenu` / `SecondaryActivate` vs docs, dbusmenu
`Version=3`, `Event` skips inert rows. ToolkitMenu stays opt-in
(`/NO_DBUSMENU` + reused popup). CGO preambles stay free of nested
`*/`.

**v0.16.1** — `GetLayout` must return D-Bus `(ia{sv}av)`. v0.16.0
exported a recursive Go struct; godbus panicked (`container nesting too
deep`) when Plasma’s watcher called `GetLayout`.

**v0.16.2–0.16.4** — Toolkit-only popup experiments (`/NO_DBUSMENU`,
then `Menu=/`, new window per click). Unreliable on Plasma/Wayland;
superseded by v0.17.0 HostMenu.

## Build / run

```bash
# Tests (always stub/fake; no bus required)
CGO_ENABLED=0 go test ./...
UITK_TRAY=fake go test ./platform ./app

# Linux desktop: the tour's Desktop page opens a status item
go run ./examples/tour -page desktop   # the app owns the tray (HostMenu)

UITK_TRAY=stub go run ./examples/tour -page desktop
UITK_TRAY=fake go run ./examples/tour -page desktop
```

The application this was built for is the mail client
([comms-mail](https://github.com/codemodify/comms-mail)): it shows a tray
item while it runs, a click raises its window, and new mail becomes a
notification whose click raises it too.

Dependency: [`github.com/godbus/dbus/v5`](https://github.com/godbus/dbus) on
Linux only (pure Go). Windows and macOS use OS APIs.
