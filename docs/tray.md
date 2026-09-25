# Status item (system tray)

`StatusItem` is a first-class toolkit API in the same role as Qt
`QSystemTrayIcon`, KDE `KStatusNotifierItem`, and Electron `Tray`: a
cross-platform notification-area / menu-bar icon, a context menu, and a
desktop toast. An application only ever calls toolkit APIs
(`app.NewStatusItem` / `platform.StatusItem`).

**v0.20.0** adds submenus. `StatusMenuItem.Submenu` nests a tray menu
to any depth on both paths — `com.canonical.dbusmenu` for HostMenu, a
`widgets.PopupMenu` cascade for ToolkitMenu and for Windows. See
[Submenus](#submenus).

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

`GetLayout` nests, but the **Go types do not**: the layout is
`(ia{sv}av)` and children are `[]dbus.Variant` wrapping a distinct leaf
type, whose own children are variants again. A variant's signature is
fixed at `v`, so the tree is as deep as the menu while the Go structs
stay finite. Never nest the same Go struct — a self-referential layout
type has no finite D-Bus signature and is the v0.16.1 godbus
`container nesting too deep` panic. `assertFiniteMenuSignature` runs
before export and fails the export (stub fallback) rather than letting
godbus panic when a host calls `GetLayout`.

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
| `StatusMenuItem` | text, separator, check, icon, `Submenu`, `OnClick` — map from `widgets.MenuItem` |
| `Notification` | title + body (+ optional icon / click) |
| `Application.NewStatusItem` | owns the item; a live item keeps `Run` after the last window closes |
| `Window.SetCloseHides` | WM close hides (close-to-tray) instead of destroy |
| `Window.Show` / `Hide` / `Raise` / `Visible` | map / unmap / activate |

`Backend()` is `sni`, `win32`, `appkit`, `fdo-notify`, `fake`, or `stub`.

Headless apps and missing hosts get a **stub**. `UITK_TRAY=fake` records
calls for tests. `UITK_TRAY=stub` forces the no-op.
`UITK_TRAY_DEBUG=1` logs Activate / ContextMenu / dbusmenu / Post.

## Submenus

A row with children opens a child menu. `Submenu` nests to any depth:

```go
Menu: []uitoolkit.StatusMenuItem{
    {Text: "Show", OnClick: win.Raise},
    {Separator: true},
    {Text: "Folders", Submenu: []uitoolkit.StatusMenuItem{
        {Text: "Inbox", OnClick: openInbox},
        {Separator: true},
        {Text: "Archive", Submenu: []uitoolkit.StatusMenuItem{
            {Text: "2025", OnClick: open2025},
        }},
    }},
    {Text: "Quit", OnClick: app.Quit},
}
```

`uitoolkit.StatusMenuFromItems` maps a `widgets.Submenu(...)` row to the
same thing, so a menu written once as `widgets.MenuItem` can be a window
menu and a tray menu.

**A parent is not a command.** A row with children never fires its own
`OnClick`, on any backend. Hosts that open a child menu send no
`clicked` for the row they opened it from — and the ones that send it
anyway would make the app behave differently per desktop — so the rule
is enforced in one place (`menuItemClickable`) instead of being left to
the host. Setting both `Submenu` and `OnClick` is not a per-host
accident: the `OnClick` is dead everywhere. A `Separator` row with
children stays a rule; the children are dropped.

### What the host is told

| dbusmenu | What this exporter does |
| --- | --- |
| ids | Pre-order over the whole tree: `1, 2, 3, …`, a parent before the rows under it, unique at every depth. Stable because they are a function of the menu alone, so `GetLayout`, `GetProperty`, `GetGroupProperties` and `Event` agree. `SetMenu` bumps the revision and emits `LayoutUpdated` when they change |
| `children-display` | `submenu` on every row that has children — in the layout and in `GetProperty` / `GetGroupProperties` — and on the root |
| `GetLayout(parent, depth, …)` | The subtree rooted at `parent` (`0` is the menu). `depth` is honoured: `-1` every level, `0` the node alone with an empty child array, `n` that many levels. A truncated parent still says `children-display: submenu` |
| `Event("clicked", id)` | Fires that row's `OnClick` at any depth. A parent, a separator and a disabled row fire nothing |
| `EventGroup` / `AboutToShowGroup` | Ids that are in no tree come back in `idErrors` |
| `AboutToShow(id)` | Always `false`, parents included. The bool means "your layout is stale, fetch it again", not "may I open?", and the whole tree was published by `GetLayout` already. Answering `true` would buy a pointless `GetLayout` round on every submenu hover. A menu built lazily per open would need the opposite answer |

### What each host does with one

Nested menus are what the dbusmenu layout type is for — `GetLayout`
takes a `recursionDepth` and rows carry `children-display` — and each
host below imports that layout with an importer it already had, not one
written for this toolkit:

| Host | Submenu |
| --- | --- |
| KDE Plasma (HostMenu) | Imports the layout and draws the cascade itself |
| GNOME 45+ AppIndicator extension | Same: the extension imports dbusmenu |
| Waybar tray module | Same |
| ToolkitMenu (any Linux desktop) | Toolkit `PopupMenu` cascade, hover or Right to open |
| Windows | Toolkit `PopupMenu` cascade — there is no dbusmenu and no `HMENU` |
| macOS | The menu is not bridged to `NSMenu` at all yet, with or without children |

What this repository verifies is the layout a host receives (ids,
`children-display`, `recursionDepth`, `Event` routing) over a private
session bus, and the toolkit cascade end to end. It does **not** drive a
real Plasma, GNOME or Waybar tray in the test suite: how deep a menu each
of them will draw, and what it does with an icon or a checkmark on a
parent row, is the host's business and is not asserted here.

### ToolkitMenu and Windows

A cascade is a **surface of its own**, as every other menu in the toolkit
is: an `xdg_popup` on Wayland — parented to the status menu's layer
surface through `zwlr_layer_surface_v1.get_popup`, since a layer surface
has no `xdg_surface` to be a parent and the popup is therefore created
with `xdg_surface.get_popup(NULL)` first — and an override-redirect
window on X11. The compositor (or `SolvePopup`) then places it against
the **screen's** edges, so it is free to run past the status-menu
window. The status-menu window is measured for the parent menu alone.

Only where a popup cannot be a surface — `UITK_POPUPS=layer`, a
compositor that refused one, headless and offscreen windows — does the
cascade open inside the status-menu window, where `PlacePopupBeside`
clamps it to that surface. The window is then measured for the parent
menu **plus** the widest chain of children and the tallest menu in it
(`measureStatusMenu`'s `cascadeInWindow`), because a window measured for
the parent alone would fold every child menu on top of its parent.

Measuring that way unconditionally is what made the tray menu on KDE
Wayland a window wide enough for two menus, with an empty band under the
last row where the tallest child menu's height had been reserved, and a
cascade squeezed into the sliver the parent left it — scrolling, with
arrows, on an otherwise empty screen.

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
| KDE Plasma | yes (SNI) | native dbusmenu | `ContextMenu` popup, placed by layer shell | Prefer HostMenu |
| GNOME 45+ | AppIndicator extension | native dbusmenu | **demoted to HostMenu** — Mutter has no layer shell | Extension expects dbusmenu |
| Sway / Hyprland + Waybar | tray module | dbusmenu | if Waybar sends `ContextMenu`; layer shell places it | |
| Xfce / Cinnamon / MATE | usually SNI | dbusmenu | if host calls `ContextMenu` | |
| Xlibre | same as X11 | same | same | |
| No session bus / SSH | stub | stub | stub | |

**Wayland:** a client cannot place an `xdg_toplevel` at SNI `(x,y)` —
the protocol has no request for it. HostMenu sidesteps the whole problem
(the panel draws the menu). ToolkitMenu needs the position, and gets it
from **`zwlr_layer_shell_v1`**: the menu window opens as a layer surface
on the `overlay` layer, anchored top|left with the requested point as its
margins, exclusive zone −1, on-demand keyboard interactivity for the
menu's grab. KDE (KWin), sway, Hyprland and wayfire offer the protocol;
**GNOME/Mutter has declined it**.

Where it is missing, `Application.NewStatusItem` **falls back to
HostMenu** rather than opening a menu the compositor drops somewhere else
— before this fix, in the middle of the screen on KWin. The demotion is
logged once (`uitk tray: ToolkitMenu asked for, HostMenu used: …`), the
item is registered with the chrome it really has (so it exports a real
dbusmenu, not `Menu=/NO_DBUSMENU`), and `uitoolkit.StatusMenuChromeFor`
answers what an item asking for a chrome will actually get, with a reason
— which is what the tour's Desktop page prints under **menu chrome**.
`platform.ScreenPlacementAvailable` is the same question one layer down;
`UITK_LAYER_SHELL=0` makes a KDE machine behave like GNOME.

Close-to-tray Hide drops the `xdg_toplevel` role; Show remaps and
requests `xdg_activation_v1` when available. The layer-surface path is
**not exercised by the test suite** — it needs a real compositor. What is
tested headlessly is the fallback decision, the placement arithmetic and
which output a point lands on.

### Where the menu goes

SNI gives a **point** and never the icon's rectangle, so both halves of
"beside the icon, on the screen" are the toolkit's to work out. One
function does it for every backend — `app.statusMenuScreenRect` over
`platform.SolveScreenMenu`, which is `platform.SolvePopup` with an anchor
invented for the point — so X11 and Wayland cannot drift apart:

- **Clear of the point.** The menu is placed a gap away from it
  (`platform.ScreenMenuPointerGap`, 24 device pixels — roughly a
  cursor's width — scaled to logical pixels), not with its corner on it.
  With the corner on the point the menu covered the tray icon it had
  been opened from. The gap is an approximation of an icon rectangle the
  protocol does not send.
- **On the screen.** The rectangle constrained is the whole surface,
  which is the parent menu — and, only where the cascades are drawn
  inside the window rather than on surfaces of their own, the chain they
  need as well (`measureStatusMenu`). It flips to the other side of the
  point where it does not fit, slides back on where flipping does not
  help, and is shrunk only when it is larger than the screen. Where the
  cascades are inside the window, constraining the parent alone is what
  left the submenus off the right edge of the screen.
- **Against which screen.** `platform.ScreenRectAt` — on Wayland the
  output holding the point, in the compositor's own logical coordinates
  (`zxdg_output_v1`, which is the only source that is right under
  fractional scaling); on X11 the RandR monitor cut to `_NET_WORKAREA`.
  A backend that cannot say constrains nothing, which is what the
  toolkit did before it could ask.

The headless tests carry the numbers from the KDE session that reported
both faults: a 1645x1029 logical desktop (2880x1800 at 175%), the icon at
device 2305,28, a 988x221 device menu.

**X11:** ToolkitMenu places the popup at root `(x,y)` (typically above
a bottom panel), through the same `SolveScreenMenu`. `Window.Raise` sends
`_NET_ACTIVE_WINDOW`.

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
toolkit `PopupMenu` when a toolkit window exists — which is also how
Windows gets submenus, for free, through `StatusMenuToItems`.
Left-click is `OnClick`; balloon click is `OnNotifyClick`. HWND app windows are still
a stub — only the tray landed.

```bat
go build ./examples/uitoolkit-sample-tour
uitoolkit-sample-tour.exe
```

### macOS

`GOOS=darwin` **and** `CGO_ENABLED=1`: `NSStatusItem` on the menu bar
and `NSUserNotification` toasts. The OS owns the menu-bar click;
toolkit `PopupMenu` is not used. Without CGO the item is a stub. AppKit
windows are still a stub.

`SetMenu` on macOS stores the rows and hands **none** of them to AppKit
— there is no `NSMenu` bridge yet, so this is not new with submenus: the
whole menu is inert there, children or not. macOS is a pinned platform
and nothing was ported to it.

```bash
CGO_ENABLED=1 go build ./examples/uitoolkit-sample-tour
```

## History

**v0.20.0** — Submenus. `StatusMenuItem.Submenu`, dbusmenu ids
flattened in pre-order over the whole tree, `recursionDepth` honoured,
`children-display: submenu`, `Event` / `GetProperty` /
`GetGroupProperties` resolving any id, and a `PopupMenu` cascade on the
toolkit path. A parent never fires `OnClick`.

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
go run ./examples/uitoolkit-sample-tour -page desktop   # the app owns the tray (HostMenu)

UITK_TRAY=stub go run ./examples/uitoolkit-sample-tour -page desktop
UITK_TRAY=fake go run ./examples/uitoolkit-sample-tour -page desktop
```

The application this was built for is the mail client
([comms-mail](https://github.com/codemodify/comms-mail)): it shows a tray
item while it runs, a click raises its window, and new mail becomes a
notification whose click raises it too.

Dependency: [`github.com/godbus/dbus/v5`](https://github.com/godbus/dbus) on
Linux only (pure Go). Windows and macOS use OS APIs.
