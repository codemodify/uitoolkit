# Window decorations

A uitoolkit window's frame — its title bar, caption buttons and resize
borders — is drawn either by the desktop (KWin's Breeze frame, an X11 window
manager's) or by uitoolkit itself, the way Chromium, SourceGit, GTK 4 and
Electron apps draw theirs: the app's own title bar with tabs or a tool bar in
it, the caption buttons at the sides the desktop puts them, resize edges
around it. Either way the desktop keeps window management: every move,
resize, snap and the window menu is handed back to the compositor or window
manager from the user's button press, never emulated.

This is Phase 1 of [the design](#phases): an opaque toolkit-drawn frame.

## For apps: a title bar

Give the window a title bar with `SetTitleBar`:

```go
hb := widgets.NewHeaderBar(
	[]widget.Component{menuButton, fetchWriteToolBar}, // leading items
	nil,                                               // centre: free space that moves the window
	[]widget.Component{searchField, filterToolBar})    // trailing items
win.SetTitleBar(hb)
win.SetContent(body)
```

- **Where uitoolkit draws the frame** the header bar is the window's caption:
  the caption buttons (minimize, maximize / restore, close, the window menu)
  are added at the sides the desktop puts them, a hairline border runs around
  the window, the content sits below.
- **Under the desktop's frame** the header bar is the window's first row,
  laid out exactly as the equivalent `widgets.NewRow(…)` — no caption
  buttons, same pixels.

`NewHeaderBar(start, center, end)` lays its items out in one row (8 px apart,
centred vertically); `center` takes the free width (tabs, a search field) or,
when nil, the free space is caption. `ShowTitle` paints the window title in
that free space. `OnContextMenu` replaces the window menu on a right-click in
caption space (Chromium's tab-strip menu). Any other component passed to
`SetTitleBar` (a `TitleBar`, a row) becomes the centre of a header bar.
`SetTitleBar(nil)` goes back to no title bar.

Mail is the pilot: its Thunderbird-style row (the M menu, Fetch / Write, free
space, the quick filter and its tool bar) is its title bar.

A SourceGit-style window puts its tabs in the centre:

```go
tabs := widgets.NewTabBar("uitoolkit", "paintengine2d")
win.SetTitleBar(widgets.NewHeaderBar(nil, tabs, []widget.Component{widgets.NewToolBar(fetch, push)}))
```

### What moves the window

Title-bar space is caption where every component from the one under the
pointer up to the title bar says so (`widget.CaptionHitTester`), so a label
inside a button is never caption:

| Component | Caption where |
| --- | --- |
| `HeaderBar`, `FlexBox` rows / columns, `Stack`, `Pad`, `Spacer`, `Label`, `Separator`, `Picture`, `TitleBar` | everywhere (their children decide for themselves) |
| `ToolBar` | between buttons and on dividers |
| `TabBar` | after the last tab |
| `MenuBar` | after the last title |
| `WindowControls` | gaps between caption buttons |
| anything else | nowhere: it is a control |

`widgets.DragArea(c)` makes all of `c` caption (its content gets no pointer
input — a logo, a decorative title), `widgets.NoDrag(c)` none of it: the
equivalents of Electron's `app-region: drag` / `no-drag`. A custom widget
implements `CaptionAt(p paintengine2d.Point) bool` to take part.

### Gestures

| Gesture on caption space | Result |
| --- | --- |
| press and drag past the desktop's drag threshold (8 px; KDE 10) | the desktop moves the window (`xdg_toplevel.move`, `_NET_WM_MOVERESIZE`) |
| click | nothing — clicks and double-clicks keep working because the move waits for the threshold |
| double-click (within the desktop's double-click time, both on caption) | the desktop's double-click action, default toggle maximize |
| middle click | the desktop's middle-click action, default none |
| right click | the header bar's own menu, else the desktop's window menu, else the toolkit's (Restore / Maximize, Minimize, Close) |
| press on a resize edge or corner | the desktop resizes from there (on the press) |
| right click on a caption button | the window menu |

After the desktop takes over a move or resize, the window drops its pointer
capture and hover: the release goes to the desktop. Apps can start the same
gestures from their own press handlers: `Window.StartMove()`,
`StartResize(edges)`, `ShowWindowMenu(p)`; and `Minimize()`,
`ToggleMaximize()`, `RequestClose()` (the close button's request: an app that
hides on close keeps its window).

`Window.NonClientHit(p)` answers what a point is — client, caption, a resize
edge, a caption button — from the last layout, the way Win32's
`WM_NCHITTEST` wants it.

### Frame layout (Phase 1)

- A 1-device-pixel border in the look's border colour; none when maximized.
- The caption inside it: the header bar, at least as tall as the caption
  buttons (30 px) and a line of title.
- Resize band: 4 px inside the left, right and bottom edges and the top
  4 px of the caption, with 16 px corner zones; none while maximized or full
  screen, and none on tiled edges (a quick-tiled window resizes only from its
  free sides). The band wins over the caption buttons of a restored window;
  a maximized window's buttons reach the screen's edge.
- Full screen: no frame and no caption buttons; the header bar stays the top
  row.
- A modal dialog dims the content, not the caption: the window can still be
  moved, minimized and closed, while the header bar's own controls are inert.

Caption buttons are generic glyphs (✕, a bar, a square, two squares for
restore, a small window for the window menu) drawn crisp at any scale over
the look's tool-button face; close turns red under the pointer; they are
dimmed while the window is in the backdrop, hidden for actions the desktop
cannot do (`wm_capabilities`, `_NET_WM_ALLOWED_ACTIONS`), out of the Tab
order, and named Minimize, Maximize or Restore, Close and Window Menu for
assistive technology.

### Keyboard and accessibility

- The title bar's controls come first in Tab order; their mnemonics and menu
  accelerators work; keys from them reach the content root (an app's
  shortcuts).
- Alt+F4 closes a window whose frame uitoolkit draws (a compositor that owns
  the shortcut never sends it). Alt+F3, Alt+Space and Meta+arrows stay the
  desktop's.
- The accessible window's first child is the title bar (`a11y.RoleTitleBar`,
  AT-SPI `title bar`, 104) named after the window, holding the header bar's
  content and the caption buttons, which AT-SPI clients can press. Under the
  desktop's frame the header bar is a plain pane (the real title bar is the
  desktop's).

## Who draws the frame

`platform.Decorations`: `DecorationsAuto`, `DecorationsServer` (the desktop),
`DecorationsClient` (uitoolkit), `DecorationsNone` (no frame: splash screens,
kiosks). First match wins:

1. `UITK_DECORATIONS=server|client|none` (also `system`, `toolkit`, `auto`),
   for testing and as an escape hatch.
2. `WindowOptions.Decorations` when not Auto.
3. Offscreen / headless windows: none of the toolkit's (screenshots and tests
   look the same everywhere) unless 1 or 2 asks.
4. The user's preference, look.json `"decorations"`: `"system"` (Settings'
   **Use system title bar and borders**) or `"toolkit"` (every window).
5. Auto: the toolkit's frame for a window with a title bar (`SetTitleBar`)
   where the desktop moves and resizes on request; the desktop's frame for
   every other window. So every app without a title bar looks exactly as
   before on KDE.

The choice is made before the window is first shown and again whenever an
input changes (a title bar set, the preference applied): the switch takes
effect at once, without a restart. The desktop's answer is obeyed.

### The Settings switch

Settings → Themes shows **Use system title bar and borders** next to **Use
the desktop's file dialogs** (Chromium's wording and meaning). On: windows
that draw their own title bar get the desktop's title bar and borders instead,
and their title bar becomes the first row. Apply writes look.json
(`"decorations": "system"`; auto is left out) and every running app switches
live.

### Per platform

| Platform | Toolkit frame | Desktop frame | Notes |
| --- | --- | --- | --- |
| Wayland with `zxdg_decoration_manager_v1` (KWin, wlroots, COSMIC, Hyprland…) | `set_mode(client_side)` | `set_mode(server_side)` | A mode is always asked for (KWin reads "unset" as server) and re-sent only when it changes; the compositor's `configure(mode)` is obeyed whenever it comes. KWin answers `server_side` — meaning no frame at all — for full-screen windows and, with `BorderlessMaximizedWindows`, maximized ones: the caption buttons go, the header bar stays the top row. Hyprland always answers `server_side`. |
| Wayland without it (GNOME's Mutter, Weston) | always | — | Nobody else draws a frame: every window gets the toolkit's, a window without a title bar a default caption with its title. (Before, such windows had no title bar, buttons or resize edges at all.) |
| X11 | `_MOTIF_WM_HINTS` {flags 2, decorations 0}, set before mapping | the property removed | Auto picks the toolkit's frame only where the window manager lists `_NET_WM_MOVERESIZE` and is not a tiling manager (i3, xmonad, awesome, …). Moves and resizes are `_NET_WM_MOVERESIZE` from the press's root position after ungrabbing; the window menu is `_GTK_SHOW_WINDOW_MENU`; minimize is `XIconifyWindow`. |
| Offscreen | when asked | — | `Offscreen` records every request (`FrameCalls`) and can simulate states, capabilities and answers, for tests. |

Window state follows the desktop: `EventWindowState` (maximized, full screen,
activated, resizing, tiled edges, suspended) from `xdg_toplevel.configure` or
`_NET_WM_STATE`; `activated` (`_NET_WM_STATE_FOCUSED`) drives the active /
backdrop look; a suspended window stops blinking its caret.

## The desktop's conventions

`Application.TitleBarPrefs()` (overridable with `SetTitleBarPrefs`):

| Desktop | Button layout | Click actions | Double-click time, drag threshold |
| --- | --- | --- | --- |
| KDE Plasma | kwinrc `[org.kde.kdecoration2]` `ButtonsOnLeft` / `ButtonsOnRight` (M window menu, I minimize, A maximize, X close, `_` spacer; KWin's other buttons are skipped). Defaults `MSE` / `HIAX`: the window menu on the left, minimize, maximize, close on the right | kwinrc `[Windows] TitlebarDoubleClickCommand`, `[MouseBindings] CommandActiveTitlebar2/3` | kdeglobals `[KDE] DoubleClickInterval`, `StartDragDist` |
| GNOME and others with a GTK / GNOME portal backend | `org.gnome.desktop.wm.preferences` `button-layout` through the settings portal (GNOME's default `appmenu:close`: close only) | `action-double-click-titlebar`, `-middle-`, `-right-` | `org.gnome.desktop.peripherals.mouse` `double-click`, `drag-threshold` |
| elsewhere | GTK's `settings.ini` `gtk-decoration-layout` (GTK 4, then 3) | `gtk-titlebar-*-click` | `gtk-double-click-time`, `gtk-dnd-drag-threshold` |
| defaults | trailing minimize, maximize, close | double: toggle maximize; middle: none; right: menu | 400 ms, 8 px |

The settings portal carries none of KWin's keys on Plasma, so kwinrc is read
directly (cascading over `XDG_CONFIG_DIRS`), and re-read when a window becomes
active after a config file changed; the portal's keys arrive live. Actions
xdg-shell cannot express fall back: "maximize vertically / horizontally only"
toggle-maximizes on Wayland (X11 does it one way), "lower" works on X11 only,
"shade" and "on all desktops" do nothing.

## Testing

- Headless: `app/frame_test.go` (hit-test table, drag threshold vs click vs
  double-click, one `StartSystemMove` per drag and none per click, capture
  dropped after it, resize edges, window menus, caption buttons, the a11y
  tree, modal dialogs, the decoration policy); `platform` tests decode the
  protocol states, capabilities, resize-edge and `_NET_WM_MOVERESIZE`
  mappings, `_MOTIF_WM_HINTS`, and parse button layouts and kwinrc / GTK
  files; `widgets/caption_test.go` checks every built-in's caption answer.
- `UITK_DECORATIONS=client` gives headless screenshots with the toolkit's
  frame.
- End to end in the nested-KWin rig ([tools/e2e](../tools/e2e/README.md)):
  `UITK_DECORATIONS=client ./run.sh N examples-mail-binary`, drags and
  resizes with `in.sh`, window geometry and state from `./kwin.py`, protocol
  traces with `WAYLAND_DEBUG=1`; `UITK_BACKEND=x11` for the X11 path on the
  instance's Xwayland. `UITK_XDG_DECORATION=0` makes the Wayland backend
  ignore `zxdg_decoration_manager_v1`: GNOME's path, on KWin.

## Phases

1. **Phase 0** (done): window state, capabilities and decoration mode
   reported; press serials and root positions kept; resize and move cursors.
2. **Phase 1** (this): the opaque toolkit frame above — buffers stay opaque
   (XRGB, full opaque region, window geometry = the whole surface), the
   present path is unchanged.
3. **Phase 2**: themed frames — a `style.DecorationEngine` so each look
   paints its era's caption, border and buttons (Win95's gradient caption,
   Luna's rounded one, Aqua's traffic lights), a restore glyph per engine, and
   `"captionButtons": "theme"` to follow the look's own layout.
4. **Phase 3**: shadows and rounded corners — ARGB buffers,
   `set_window_geometry`, input and opaque regions, `_GTK_FRAME_EXTENTS`,
   resize handles in the shadow, per-edge variants for tiled windows, a
   solid frame without a compositing manager.
5. **Phase 4**: KWin's server-decoration palette, `xdg-toplevel-icon`,
   `_NET_WM_SYNC_REQUEST`, tab tear-off, Windows and macOS mappings.

Until Phases 2–3 the toolkit's frame is square, shadowless and the same in
every look apart from the tool-button face under the caption buttons; the
window's geometry includes the 1 px border.
