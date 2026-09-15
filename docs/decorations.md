# Window decorations

A uitoolkit window's frame — its title bar, caption buttons and resize
borders — is drawn either by the desktop (KWin's Breeze frame, an X11 window
manager's) or by uitoolkit itself, the way Chromium, SourceGit, GTK 4 and
Electron apps draw theirs: the app's own title bar with tabs or a tool bar in
it, the caption buttons at the sides the desktop puts them, resize edges
around it. Either way the desktop keeps window management: every move,
resize, snap and the window menu is handed back to the compositor or window
manager from the user's button press, never emulated.

Phases 1 and 2 of [the design](#phases) are done: an opaque toolkit-drawn
frame, painted by the look — Windows 95's navy caption and bevelled buttons,
XP's blue one, Aqua's traffic lights, libadwaita's round buttons, SourceGit's
flat 48×30 cells — with tabs or a tool bar in the title bar.

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

A SourceGit- or Chromium-style window puts document tabs in the centre
([tabs in the title bar](#tabs-in-the-title-bar)):

```go
tabs := widgets.NewBrowserTabs("uitoolkit", "paintengine2d")
tabs.OnNew = func() { tabs.Select(tabs.AddTab(widgets.BrowserTab{Title: "New"})) }
win.SetTitleBar(widgets.NewHeaderBar([]widget.Component{menuBar}, tabs, nil))
```

How the header bar and the frame meet is the look's: a **merged** frame
makes the header bar the caption, one row with the caption buttons at its
sides (GTK's header bars, Windows 10 and 11, macOS, Chromium, SourceGit); a
**stacked** frame keeps the era's own caption — the window title and the
caption buttons in the era's strip — and puts the header bar in a row under
it, the way the tool bars of Windows 95, XP, the classic Mac and Motif sat
under their title bars. Either way the header bar's free space moves the
window, and in a merged caption a menu bar, tool bar or tab strip paints no
bar of its own: the caption band is their background.

### What moves the window

Title-bar space is caption where every component from the one under the
pointer up to the title bar says so (`widget.CaptionHitTester`), so a label
inside a button is never caption:

| Component | Caption where |
| --- | --- |
| `HeaderBar`, `FlexBox` rows / columns, `Stack`, `Pad`, `Spacer`, `Label`, `Separator`, `Picture`, `TitleBar` | everywhere (their children decide for themselves) |
| `ToolBar` | between buttons and on dividers |
| `TabBar` | after the last tab |
| `BrowserTabs` | everywhere but its tabs, their close buttons, "+" and the scroll arrows |
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
| right click | the menu of the component under the pointer (`widget.CaptionMenuer`: a tab strip's), else the header bar's own (`OnContextMenu`), else the desktop's window menu, else the toolkit's (Restore / Maximize, Minimize, Close) |
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

### Frame layout

- The look's border (`DecorationSpec.Border`, whole device pixels): a
  hairline in the plain and modern looks, Windows 95's 4 px raised frame,
  XP's five lines, Windows 7's glass, Motif's resize handles; none when
  maximized (the era's frames went off the screen).
- The caption inside it: a merged frame's header bar, at least as tall as
  the look's caption and its buttons; or a stacked frame's strip (the look's
  caption height) with the header bar's row under it when the app gave the
  header bar items of its own.
- Resize band: the border, at least 4 px, inside the left, right and bottom
  edges and at the top of the caption, with 16 px corner zones; none while
  maximized or full screen, and none on tiled edges (a quick-tiled window
  resizes only from its free sides). The band wins over the caption buttons
  of a restored window; a maximized window's buttons reach the screen's
  edge: each button's hit box runs up to the caption's top and the outermost
  one out to the window's side.
- Full screen: no frame and no caption buttons; the header bar stays the top
  row.
- A modal dialog dims the content, not the caption: the window can still be
  moved, minimized and closed, while the header bar's own controls are inert.
- Frames are opaque and square: rounded corners, shadows and translucent
  glass are Phase 3. The spec already carries the look's wishes for them
  (`Radius`, `Shadow`).

Caption buttons are the look's (see [the looks](#the-looks)): sized and
spaced by the look, hover and pressed states, the restore glyph while
maximized, the backdrop look while the window is inactive. They are hidden
for actions the desktop cannot do (`wm_capabilities`,
`_NET_WM_ALLOWED_ACTIONS`), out of the Tab order, and named Minimize,
Maximize or Restore, Close and Window Menu for assistive technology.

## Tabs in the title bar

`widgets.BrowserTabs` is a strip of document tabs, browser style —
Chromium's tab strip, SourceGit's repository tabs, Dolphin's folder tabs.
As a header bar's centre it takes the caption's whole height, so its tabs
stand on the content below and the selected one joins it (SourceGit's
concave feet in the web looks, the look's own notebook tab elsewhere):

- the tabs, their close buttons (on the selected tab and the one under the
  pointer), the "+" (shown when `OnNew` is set) and the scroll arrows are
  controls; the rest of the strip is caption: a drag moves the window, a
  double click maximizes, a right click opens `OnContextMenu(-1, at)` or the
  window menu. At least 24 px at the end are always caption.
- tabs share the width, at most `MaxTabWidth` (200 px); below `MinTabWidth`
  (80 px) the strip scrolls — the wheel, two arrow buttons at its end — and
  keeps the selected tab in view. Elided titles are the tab's tool tip.
- a press selects (browsers select on the press); a drag past 8 px moves the
  tab and the others make room (`OnReorder`); a middle click closes; closing
  the selected tab selects the next one. `OnClose` asks the app, which
  removes the tab with `RemoveTab`; `NoClose` keeps a tab (a window's last).
- keys: Left, Right, Home, End with the focus (a click does not take it, as
  in browsers; Tab reaches the strip); `Shortcut(e)` from the app's own key
  handler gives Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+PageUp / PageDown, Ctrl+W,
  Ctrl+F4 and Ctrl+T in one call (the window offers Ctrl+Tab to the app
  before moving the focus with it).
- assistive technology sees a page tab list of page tabs, each holding its
  pressable close button, and a New Tab button.
- under the desktop's frame the strip is simply the header bar's row.

Files is the pilot: its folders open in tabs next to the menu bar in the
title bar; the tree and the table show the selected tab's folder, and the
window title follows it. Tab tear-off (dragging a tab out into its own
window) is Phase 4: the drag keeps the pointer's position, where a
tear-off check will go, through `xdg-toplevel-drag-v1` where the compositor
has it.

## The looks

A look paints the frame through `style.DecorationOf` (the measurements in
a `DecorationState`: active or backdrop, maximized, tiled, a caption with
the app's own items), `DrawDecorationOf` (border and caption band, under
the title bar), `DrawCaptionTitleOf` and `DrawCaptionButtonOf`. An engine
paints its era's frame by implementing the optional `style.DecorationEngine`
(see [theme-engines.md](theme-engines.md#window-frames)); every other engine
has its in-app window frame (`DrawWindowFrame`, `WindowCloseRect`) adapted,
so every pack has a frame in its own look.

| Engines | Frame |
| --- | --- |
| `win95` | stacked: the 4 px raised border, the caption bar in the scheme's colour (98's and 2000's gradient), the bold title at its left, bevelled buttons with Marlett-shaped glyphs, close 2 px apart |
| `luna` | stacked: the scheme's caption gradient, the sizing frame's five lines, the bold shadowed title, XP's glossy 21 px buttons (minimize and maximize in the caption's shade, close red) |
| `aero` | merged: an opaque approximation of the glass all round, the black title on its glow, the joined button group hanging from the top edge (26 px frosted minimize and maximize that glow blue, the 43 px red close) |
| `metro` | Windows 8 stacked: the thick flat coloured frame, the centred title, the red close hanging from the top; Windows 10 merged: 46 px flat buttons, red close |
| `fluent` | merged: Mica, the window stroke, 46 px buttons filling the title bar, WinUI's subtle fills, `#c42b1c` close |
| `aqua` | stacked: the pinstriped title bar, the centred title, the gel traffic lights |
| `macos` | merged: 28 px title (52 px unified tool bar), the centred title, 12 px traffic lights |
| `breeze` | merged: KWin Breeze's look — glyph buttons that fill a circle under the pointer, red for close |
| `adwaita` | merged: libadwaita's header bar with 24 px round buttons (GTK 3's flat square ones) |
| `web` | merged: SourceGit's 38 px title bar, 48×30 buttons with a black-25% wash and a pure red close |
| `motif` | stacked: mwm's resize handles and raised title-bar parts |
| `flatlaf`, `material` | merged: FlatLaf's title pane; Material's surface with circular icon buttons |
| `base` | merged: a hairline, flat buttons over the tool face, close red |
| the others | adapted: their in-app caption as a stacked strip, their own close button, push buttons for the rest |

Glyphs, sizes and colours are the look's; the side and order of the
buttons are the desktop's, unless the user prefers the look's own layout:
look.json `"captionButtons": "theme"` (Settings: **Place window buttons as
the theme does**) puts the Mac's traffic lights on the left, gives GNOME's
lone close, KDE's window menu on the left. It applies live, to every app.

`go run ./cmd/uitk-themesheet -frames -theme luna -o /tmp/f` renders a
pack's frames sheet: windows active, in the backdrop, maximized, with the
close button hot and pressed, maximize hot, with tabs in the title bar
(active and backdrop) and with KDE's button layout.

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
- `style`: every pack's frame at 1, 1.25, 1.5, 1.75 and 2x — borders on
  whole device pixels, buttons inside the caption, square when maximized —
  paints inside its window, title box and button boxes; the adapter takes
  the in-app frame's caption, borders and close button; glyphs stay crisp.
  `widgets`: the tab strip (layout, caption answers, select, close, new,
  drag to reorder, overflow and scrolling, keys and shortcuts, tool tips,
  accessibility). `app`: stacked frames (strip, row, hit tests, a drag from
  either, the title for AT-SPI), tabs in the title bar (a tab is a control,
  the strip caption, the strip's menu, Ctrl+Tab to the app), the theme's
  button layout following the look, the title repainting.
- `UITK_DECORATIONS=client` gives headless screenshots with the toolkit's
  frame; `uitk-themesheet -frames` renders every pack's frames.
- End to end in the nested-KWin rig ([tools/e2e](../tools/e2e/README.md)):
  `UITK_DECORATIONS=client ./run.sh N examples-mail-binary`, drags and
  resizes with `in.sh`, window geometry and state from `./kwin.py`, protocol
  traces with `WAYLAND_DEBUG=1`; `UITK_BACKEND=x11` for the X11 path on the
  instance's Xwayland. `UITK_XDG_DECORATION=0` makes the Wayland backend
  ignore `zxdg_decoration_manager_v1`: GNOME's path, on KWin.

## Phases

1. **Phase 0** (done): window state, capabilities and decoration mode
   reported; press serials and root positions kept; resize and move cursors.
2. **Phase 1** (done): the opaque toolkit frame above — buffers stay opaque
   (XRGB, full opaque region, window geometry = the whole surface), the
   present path is unchanged.
3. **Phase 2** (done): themed frames — `style.DecorationEngine` with native
   frames for the engines above and the adapter for the rest, merged and
   stacked title bars, a restore glyph per engine, `"captionButtons":
   "theme"`, and document tabs in the title bar (`widgets.BrowserTabs`,
   Files).
4. **Phase 3**: shadows and rounded corners — ARGB buffers,
   `set_window_geometry`, input and opaque regions, `_GTK_FRAME_EXTENTS`,
   resize handles in the shadow, per-edge variants for tiled windows, a
   solid frame without a compositing manager.
5. **Phase 4**: KWin's server-decoration palette, `xdg-toplevel-icon`,
   `_NET_WM_SYNC_REQUEST`, tab tear-off, Windows and macOS mappings.

Until Phase 3 the toolkit's frame is square and shadowless, and the
window's geometry includes the look's border.
