# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
Device onto a window and translates input. `UITK_PAINT=auto` (default)
tries `GPUDevice` and falls back to the CPU pixmap.

| Backend | When | Present | Clipboard | Scale | IME | Cursor |
| --- | --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a | last `SetCursor` |
| **Wayland** | Linux + CGO + `WAYLAND_DISPLAY` | **`wl_egl_window` + `eglSwapBuffers`** when `UITK_PAINT=auto\|gpu` and EGL works; else v0.4.1 **`wl_shm` XRGB8888** (opaque). **linux-dmabuf** only if `UITK_WAYLAND_PRESENT=dmabuf` on the CPU path | `wl_data_device` + primary when the compositor offers it (drag and drop rides the same device) | `wl_output` scale, `wp_fractional_scale_v1` + viewporter, env | `zwp_text_input_v3` preedit / commit | host cursors: `wp_cursor_shape_v1` when advertised, else `wl_cursor_theme` (`XCURSOR_THEME` / `XCURSOR_SIZE`) + `wl_pointer_set_cursor` (enter serial); re-applied on pointer enter |
| **X11** | Linux + CGO + `DISPLAY` | **EGL window + `eglSwapBuffers`** when EGL works; else dirty-rect `XPutImage` / MIT-SHM | CLIPBOARD + PRIMARY, ICCCM **INCR**; drag and drop is **XDND 5** | Xft.dpi, RandR mm, screen mm, env | XIM preedit callbacks + compose / dead keys | host cursors: `XcursorLibraryLoadCursor` theme names, else `XCreateFontCursor` + `XDefineCursor` |
| Win32 / AppKit | stub windows; **tray** is `Shell_NotifyIcon` / `NSStatusItem` | — | — | — | — | host cursors: `LoadCursorW` (`IDC_ARROW` / `SIZEWE` / `SIZENS` / `IBEAM`); AppKit `NSCursor` (arrow / resizeLeftRight / resizeUpDown / IBeam) |

Auto-select: Wayland if `WAYLAND_DISPLAY` is set **and** a compositor
accepts the connection, else X11 if `DISPLAY` is set, else offscreen.
Override with `UITK_BACKEND=x11|wayland|offscreen` or `Application`
`Options.Backend`. A missing Wayland compositor falls through so
offscreen and X11 keep working.

```bash
# Wayland (when a compositor is running)
WAYLAND_DISPLAY=wayland-0 go run ./examples/gallery
WAYLAND_DISPLAY=wayland-0 go run ./examples/notes

# X11
DISPLAY=:0 UITK_BACKEND=x11 go run ./examples/gallery
UITK_BACKEND=x11 go run ./examples/notes

# Offscreen / CI
go run ./examples/gallery -headless
CGO_ENABLED=0 go test ./...

# Paint backend (honors paintengine2d UITK_PAINT)
UITK_PAINT=auto go run ./examples/gallery   # default: GPU if EGL works
UITK_PAINT=cpu  go run ./examples/gallery   # force CPU + shm / XPutImage
UITK_PAINT=gpu  go run ./examples/gallery   # prefer EGL; CPU if init fails
```

CGO Linux links `libX11`, `libXext`, `libXrandr`, `libXi`, `libwayland-client`,
`libwayland-cursor`, `libwayland-egl`, and `libxkbcommon`. Theme
cursors on X11 load `libXcursor.so.1` at runtime when present.
paintengine2d's GPUDevice also needs EGL / GLES2. `CGO_ENABLED=0`
never needs those libraries.

## Cursors (0.18.2)

Pointer shapes are **host-provided**. `platform.Cursor` stays a small
logical enum (`default`, `col-resize`, `row-resize`, `text`, the eight
window-edge resize shapes `n-resize` … `sw-resize`, `move`, `grab`,
`grabbing`). Widgets (`TableView` column dividers, `Splitter` sash, text
fields) and the window frame's resize edges (`Edges.ResizeCursor`) call
`Window.SetCursor`; each backend maps that to the compositor or OS theme.
uitoolkit does not draw 24×24 ARGB cursor glyphs.

| Backend | How |
| --- | --- |
| Wayland | Prefer `wp_cursor_shape_manager_v1` (`default`, `col_resize`, `row_resize`, `text`, `n_resize` … `sw_resize`, `move`, `grab`, `grabbing`). Else `wl_cursor_theme_load` from `XCURSOR_THEME` / `XCURSOR_SIZE` and `wl_pointer_set_cursor` with the theme buffer (CSS names such as `n-resize`, then the legacy `top_side` …). Both need a pointer-enter serial. |
| X11 | `XcursorLibraryLoadCursor` (`left_ptr`, `sb_h_double_arrow`, `sb_v_double_arrow`, `xterm`, `top_side` … `bottom_left_corner`, `fleur`) when libXcursor is available, else `XCreateFontCursor` (`XC_left_ptr`, `XC_sb_h_double_arrow`, `XC_sb_v_double_arrow`, `XC_xterm`, `XC_top_side` … `XC_bottom_left_corner`, `XC_fleur`). |
| Win32 | `LoadCursorW` + `SetCursor` with `IDC_ARROW`, `IDC_SIZEWE`, `IDC_SIZENS`, `IDC_SIZENWSE`, `IDC_SIZENESW`, `IDC_SIZEALL`, `IDC_HAND`, `IDC_IBEAM`. |
| AppKit | `NSCursor` `arrowCursor`, `resizeLeftRightCursor`, `resizeUpDownCursor`, `IBeamCursor` (no public diagonal resize cursors: corners keep the arrow). |
| offscreen | Records the last logical `SetCursor` (tests / screenshots). |

## X11 notes (0.3.0 / 0.5.0)

- When `UITK_PAINT=auto|gpu` and EGL init succeeds, present is an EGL
  window surface + `eglSwapBuffers` (`NewGPUDeviceEGL` with
  `EGLPlatformX11`). A failed swap rebuilds the CPU `XImage` and falls
  back to `XPutImage`.
- One shared `Display` for all windows. Destroying a window frees its
  `XImage` / MIT-SHM segment, GC, XIC, and `XID`; the connection closes
  when the last surface is gone and no selection is owned.
- Present copies premul RGBA into a 32-bit ZPixmap. Typical LE
  TrueColor (`red 0xff0000`) uses the same damage-rect uint32
  RGBA→BGRA upload as Wayland; unusual visuals still pack via masks
  and `XImage` byte order / `bytes_per_line`. MIT-SHM
  (`XShmPutImage`) is used when `XShmAttach` succeeds; Xvfb, SSH, and
  locked-down servers fall back to `XPutImage` of each damage rect.
- Resize uses `NorthWestGravity`, `XResizeWindow`, and a rebuilt pixmap;
  the toolkit full-repaints so expose after configure is not garbage.
- EWMH: `Window.SetFullscreen` / `SetMaximized` send `_NET_WM_STATE`.
  Close is `WM_DELETE_WINDOW`. The window's own `_NET_WM_STATE` and
  `_NET_WM_ALLOWED_ACTIONS` are read back on `PropertyNotify`
  (`EventWindowState`, `EventCapabilities`): maximized, full screen,
  hidden, one-way maximize as tiled edges, and activated from
  `_NET_WM_STATE_FOCUSED` (focus events where the window manager does not
  set it).
- Window frames (see [decorations.md](decorations.md)): a toolkit-drawn
  frame sets `_MOTIF_WM_HINTS` {flags 2, decorations 0} before the window
  is mapped; switched to the window manager's frame, a window that had
  asked for none asks for all of it back ({flags 2, decorations 1}) —
  KWin ignores the property going away; button presses
  keep their root position, button and time, and `StartSystemMove` /
  `StartSystemResize` ungrab the pointer and send `_NET_WM_MOVERESIZE`
  (move 8, edges 0–7) to the root when `_NET_SUPPORTED` lists it; the
  window menu is `_GTK_SHOW_WINDOW_MENU`; `Minimize` is `XIconifyWindow`.
  `Hide` keeps a window unmapped until `Show` (a repaint used to map it
  again).
- `ClipboardSet` owns **CLIPBOARD** and **PRIMARY**. `ClipboardGet`
  converts UTF8_STRING (then XA_STRING). Transfers larger than
  `UITK_X11_INCR_THRESHOLD` (default 16KiB, or ¼ of `XMaxRequestSize`)
  use the ICCCM **INCR** protocol in both directions.
- Input: `XkbSetDetectableAutoRepeat`, `XFilterEvent`, and
  `Xutf8LookupString` on an XIC. The IC is created with
  `XIMPreeditCallbacks | XIMStatusNothing` when the IM accepts it
  (fallback: `XIMPreeditNothing`). Preedit draw/caret callbacks become
  `EventIMEPreedit` and land in TextField / TextArea as underlined
  composition. Candidate windows stay with the IM (ibus / fcitx / XIM).
  Focus-out calls `XmbResetIC` and `EventIMECancel`. The pointer comes
  through XInput 2 where the server has it (smooth scrolling, gestures:
  see [Touchpad gestures and scrolling](#touchpad-gestures-and-scrolling));
  a wheel notch is `Scroll` ±1 as on Wayland, three lines in a widget.
- Scale: `UITK_SCALE` / `GDK_SCALE` / `QT_SCALE_FACTOR` / `GDK_DPI_SCALE`,
  then Xft.dpi, then RandR output mm vs CRTC pixels, then screen mm.
  Buffer and event coordinates stay device pixels; metrics grow with scale.

## Scene graph (0.6.0)

**v0.15.0:** still full `DrawScene` + full `Surface.Present` (v0.13.8 /
v0.14.7). A full present keeps recorded groups; layout and look still
re-record. See [perf.md](perf.md).

`UITK_SCENE` (default on) records each dirty widget into a retained
`paintengine2d.Scene` (Qt Quick QSG / GTK GSK lite): rects, paths,
glyph/image blits, and transform groups. `DrawScene` presents the graph
(GPU batches opaque axis-aligned rects; CPU rasterizes nodes).
`ScrollView` keeps a child group and only updates a translation when
the offset changes. `ListView` / `TableView` / `TreeView` / `CardList`
keep per-row groups and paint visible rows in viewport space under a
body clip (below a sticky table header) so `DrawScene` cannot shift the
clip with the content. Hover and selection re-record rows whose visual
signature changed.

```bash
UITK_SCENE=auto go run ./examples/gallery   # default: retained scene
UITK_SCENE=off  go run ./examples/gallery   # v0.5 immediate Fill path
```

### Idle CPU on Wayland gallery

```bash
UITK_PAINT=auto UITK_SCENE=auto go run ./examples/gallery
# another terminal:
pidof gallery   # or: pgrep -f 'examples/gallery'
top -p "$(pgrep -n -f 'examples/gallery')"
# or: pidstat -p "$(pgrep -n -f 'examples/gallery')" 1
```

Idle (no mouse, no focused text field): the process should sit near **0%**
CPU — `Run` is blocked on `wl_display` / X11 fd, and `eglSwapBuffers` is
not called. Moving the mouse should stay responsive; CPU rises only while
events/damage arrive. A focused TextField wakes ~2 Hz for caret blink.

## Run loop (0.5.1)

`Application.Run` is event-driven. It blocks on the Wayland (`wl_display_get_fd`)
or X11 (`ConnectionNumber`) file descriptor with `poll` / `select`. A timeout
is used only for caret blink (530 ms, text fields only), tooltip delay,
Wayland key repeat, or `Window.RequestAnim`. Idle gallery should nearly
sleep: no 16 ms ticker, and `Present` / `eglSwapBuffers` run only when
`Damage` is non-empty.

Hover on virtualized rows (list / table / tree) and chrome strips
(toolbar / menubar / tabs) invalidates the old and new item, not the
whole window.

## Paint backend (0.5.0 / 0.5.1)

`UITK_PAINT` is defined by paintengine2d and honored here. It selects the
**paint device**, not the event loop:

| Value | Window paint | Present |
| --- | --- | --- |
| **auto** (default, unset) | `NewGPUDeviceEGL` when EGL init works, else CPU pixmap | `eglSwapBuffers`, else shm / `XPutImage` |
| **gpu** | same try-EGL | same; CPU fallback if the native window cannot bind |
| **cpu** | CPU pixmap (`NewContext` on `Buffer`) | v0.4.1 opaque `wl_shm` / `XPutImage` |

Wayland GPU present creates a `wl_egl_window` on the `wl_surface` and
passes `wl_display*` + `wl_egl_window*` to `NewGPUDeviceEGL`
(`EGLPlatformWayland`). X11 passes `Display*` + `Window`
(`EGLPlatformX11`). GPU window configs request `EGL_ALPHA_SIZE` 0 so an
opaque UI cannot present as a fully transparent ARGB surface.

`platform.NewPaintContext` is the toolkit paint seam: `GPUDevice` when
bound, otherwise a CPU device wrapping `Surface.Buffer`. Offscreen and
`CGO_ENABLED=0` stay on the CPU path.

A transparent Wayland window is a present-path bug, not an idle-loop
bug. If a window is fully transparent, set `UITK_PAINT=cpu` and/or
`UITK_WAYLAND_PRESENT=shm` (opaque `XRGB8888`). GPU window configs still
request `EGL_ALPHA_SIZE` 0.

## Wayland notes (0.4.1 / 0.5.0)

- `wl_display_connect` → registry bind of `wl_compositor`, `wl_shm`,
  `xdg_wm_base`, `wl_seat`, `wl_data_device_manager`, `wl_output`, and
  optional `zwp_linux_dmabuf_v1`, `zwp_linux_explicit_synchronization_v1`,
  `wp_linux_drm_syncobj_v1`, `zwp_text_input_manager_v3`,
  `zwp_primary_selection_device_manager_v1`,
  `zxdg_decoration_manager_v1`, `wp_fractional_scale_manager_v1`,
  `wp_viewporter`, `xdg_activation_v1`, `wp_cursor_shape_manager_v1`,
  `xdg_toplevel_drag_manager_v1`, `zwp_pointer_gestures_v1` (touchpad
  gestures) and `zxdg_exporter_v2` (a window named to a portal dialog).
- Each window is an `xdg_toplevel`; `xdg_wm_base` is bound up to v6.
  Configure width/height are surface-local (logical); the present buffer
  is `ceil(logical * scale)`. The toplevel's states (maximized,
  fullscreen, resizing, activated, the four tiled edges, suspended) and
  `wm_capabilities` are applied with the `xdg_surface.configure` that
  follows them and reported as `EventWindowState` / `EventCapabilities`
  (`Window.WindowState()`); `activated` drives the window's active /
  backdrop look. `configure_bounds` keeps a window the client sizes
  itself inside the work area. `xdg_toplevel.close` is `EventClose`.
  Decorations are negotiated with `xdg-decoration` (see
  [decorations.md](decorations.md)): a mode is always requested and the
  compositor's `configure(mode)` answer is obeyed.
- When `UITK_PAINT=auto|gpu` and EGL init succeeds, present is
  **`wl_egl_window` + `eglSwapBuffers`** (paintengine2d `GPUDevice`).
  Resize calls `wl_egl_window_resize` and `GPUDevice.Resize`. A failed
  swap drops GPU for that surface and falls back to shm.
- CPU present (no EGL, or `UITK_PAINT=cpu`) defaults to **`wl_shm`
  `XRGB8888`** (opaque). `UITK_WAYLAND_PRESENT=auto` and `shm` are the
  same path. **Do not use ARGB8888 for opaque UI**: paintengine2d
  is premul RGBA; a wrong swizzle or an empty GBM map leaves alpha=0 and
  Mutter/Weston draw a fully transparent window. Workaround if a build
  still prefers dmabuf: `UITK_WAYLAND_PRESENT=shm`.
- **linux-dmabuf** is opt-in (`UITK_WAYLAND_PRESENT=dmabuf`) when:
  1. The compositor advertises `zwp_linux_dmabuf_v1` (v2+) and a
     CPU-linear **opaque** format first: `DRM_FORMAT_XRGB8888` /
     `XBGR8888`, then `ARGB8888` / `ABGR8888`
  2. A local dmabuf can be created: **GBM/DRM** linear BO
     (`libgbm.so` + `/dev/dri/renderD*` or `card*`), else
     `/dev/dma_heap/system`, else memfd + `/dev/udmabuf`
  A failed import **or a blank first upload** (destination alpha still
  zero after an opaque source blit) falls back to shm for that
  connection. Explicit acquire fences (`zwp_linux_explicit_synchronization_v1`
  / `wp_linux_drm_syncobj_v1`) are **not** attached on present: an
  unsignaled fence leaves the compositor waiting on an invisible surface.
  Implicit reservation + `DMA_BUF_IOCTL_SYNC` + `wl_buffer.release` remain.
- Both paths attach premul 8-bit pixels from `paintengine2d.Image.Pix`
  (`RowStride`) with a **single** damage-rect uint32 swizzle: XRGB/ARGB
  RGBA→BGRA and force A/X=0xFF; XBGR/ABGR is a row copy with A=0xFF.
  Then `wl_surface.set_opaque_region` (once per size), damage, attach,
  commit. Four present slots pipeline frames so a tight capture loop
  does not `wl_display_roundtrip` every present (two slots + 24
  roundtrips was ~55ms/frame at 1000×760). If every slot is busy,
  `pickSlot` blocks on **one** `wl_display_dispatch` for a release —
  not a sync-object wait. Integer `wl_surface.set_buffer_scale` or
  fractional-scale + viewport destination (set when scale changes).
  Resize rebuilds the pixmap and free present slots.
- Seat: pointer (motion, buttons, axis) with coordinates multiplied by
  buffer scale; keyboard via **xkbcommon** (keymap, mods, UTF-8,
  compose / dead keys) plus compositor `repeat_info`. The last button
  press's serial and the buttons held are kept per seat: `StartSystemMove`
  / `StartSystemResize` send `xdg_toplevel.move` / `resize` with it while
  the button is down (KWin and Mutter want that), `ShowWindowMenu` sends
  `show_window_menu` at surface-local logical coordinates, `Minimize` is
  `set_minimized` (Hide still drops the role). `UITK_XDG_DECORATION=0`
  ignores `zxdg_decoration_manager_v1` (GNOME's path, for testing).
- Clipboard: `wl_data_device` copy/paste (`text/plain;charset=utf-8`).
  Primary selection when the compositor binds
  `zwp_primary_selection_v1` (Weston does). Middle-click paste uses
  `ClipboardPrimaryGet`.
- IME: `zwp_text_input_v3` is enabled when a TextField / TextArea has
  focus (`IMESurface.SetIMEEnabled`), not on every keyboard enter.
  `preedit_string`, `commit_string`, and `delete_surrounding_text`
  become `EventIME*`. Cursor rectangle is updated from the focused
  field. Printable UTF-8 from xkb/compose still becomes `EventText`
  unless an **active preedit** is in progress — compositors that enter
  text-input without committing must not starve Latin typing (Add
  Account, Quick Filter). Matching IME commit + xkb utf8 for the same
  key is deduped.

xdg-shell and the optional protocols are generated from
wayland-protocols and committed:

```bash
wayland-scanner client-header \
  /usr/share/wayland-protocols/stable/xdg-shell/xdg-shell.xml \
  platform/xdg-shell-client-protocol.h
wayland-scanner private-code \
  /usr/share/wayland-protocols/stable/xdg-shell/xdg-shell.xml \
  platform/xdg-shell-protocol.c
# same for text-input-unstable-v3, primary-selection-unstable-v1,
# xdg-decoration-unstable-v1, fractional-scale-v1, viewporter,
# linux-dmabuf-unstable-v1, linux-explicit-synchronization-unstable-v1,
# linux-drm-syncobj-v1, xdg-activation-v1, cursor-shape-v1,
# xdg-toplevel-icon-v1 (staging), pointer-gestures-unstable-v1,
# xdg-foreign-unstable-v2
```

`zwlr_layer_shell_v1` is not in wayland-protocols — it is wlroots'
(`wlr-layer-shell-unstable-v1.xml`, © 2017 Drew DeVault, MIT). The XML is
not vendored; the generated pair is, with the upstream copyright intact
in its header:

```bash
wayland-scanner client-header \
  wlr-layer-shell-unstable-v1.xml \
  platform/wlr-layer-shell-unstable-v1-client-protocol.h
wayland-scanner private-code \
  wlr-layer-shell-unstable-v1.xml \
  platform/wlr-layer-shell-unstable-v1-protocol.c
```

KWin's `org_kde_kwin_server_decoration_palette` is the exception: its XML
is KDE's and LGPL, so `platform/kde_palette.h` / `kde_palette_linux.c`
spell its two interfaces out by hand from the protocol's facts (the
names, `create(new_id, wl_surface)`, `set_palette(string)`, `release`)
rather than generating them.

```bash
# Paint + present (Linux + CGO)
UITK_PAINT=auto go run ./examples/gallery              # GPU if EGL works, else CPU
UITK_PAINT=cpu go run ./examples/gallery               # CPU pixmap + shm / XPutImage
UITK_WAYLAND_PRESENT=auto go run ./examples/gallery    # CPU path: wl_shm XRGB8888
UITK_WAYLAND_PRESENT=shm go run ./examples/gallery     # same as auto; workaround if a window is transparent
UITK_WAYLAND_PRESENT=dmabuf go run ./examples/gallery  # CPU path experimental; shm if the upload is blank
```

## HiDPI

`Options.Scale <= 0` detects scale (`UITK_SCALE`, `GDK_SCALE`,
`QT_SCALE_FACTOR`, `GDK_DPI_SCALE`, then native). The Classic look is
rebuilt with `style.ScaleMetrics` so control heights and baked glyph
atlases grow.

On **X11** the window buffer is the pixel size the WM gave; scale only
grows metrics. On **Wayland** configure size is logical and the shm
buffer is scaled; pointer events are multiplied so hit-testing matches
the pixmap. Set `UITK_SCALE` to the compositor scale (often `2`) so
metrics and buffer scale agree. List/table/tree rows and fixed column
widths are design pixels that grow with the look. EGL, dmabuf, and shm
share that scale; CPU present still uses damage / attach / commit.

### Which pixels a number is in

**Window geometry — size and position — is logical pixels. Everything
else is device pixels.**

| Logical | Device |
| --- | --- |
| `WindowOptions.Width`, `Height`, `MinWidth`/`MinHeight`, `MaxWidth`/`MaxHeight`, `X`, `Y` | `Surface.Size`, `Surface.Buffer`, `Present`'s rectangles |
| `Surface.Resize`, `EventResize.Width`/`Height` | every event position, every widget rectangle |
| `app.Window.Size`, `SetSize`, `Position`, `Move`, `platform.SizeLimits` | `app.Window.PixelSize`, `SurfaceSize`, `WindowRect` |
| `HostMover.Move`, `HostPositioner.Position`, `dock` float geometry and its layout JSON | `platform.Frame` (margin, input band, radii) |

A window's position and its size add up: the window at `Position()` that
is `Size()` wide ends where a window moved to `x + w` begins, at any
scale, so an app that keeps windows side by side (the players' rack) does
arithmetic in one unit. `platform.DevicePosition` / `LogicalPosition`
convert a position the way `DevicePixels` / `LogicalPixels` convert a
size, keeping zero and negative values; at a fractional scale each edge
rounds to the root's pixels on its own, so two such windows can meet a
device pixel apart. Qt's `QWindow::position()` is the same idea.

A window asked for as 275 × 116 is that at any display scale: 275 × 116
device pixels at 1, 481 × 203 at 1.75, with its content drawn at 1.75 to
match. An app states its design once and gets the same window on both
backends — which is the point, because the backends do not agree about
this and cannot be made to.

Wayland already speaks logical pixels (`xdg_surface.set_window_geometry`,
and the compositor multiplies). **X11 has no notion of a display scale at
all** — an X window's size *is* pixels — so the X11 backend converts with
`platform.DevicePixels` and `LogicalPixels` at that one boundary: on the
way in for `Resize`, `WindowOptions` and `WM_NORMAL_HINTS`, on the way out
for the `EventResize` a `ConfigureNotify` becomes — and positions the same
way, for `Move`, `WindowOptions.X`/`Y` and `Position`. A resize the app echoes
straight back (which is what `Window.dispatch` does) is recognised as the
size the window was last told and changes nothing, so a window manager
that picked a size between two logical pixels is not argued with.

`Window.OnResize(func(w, h int))` hears the window's logical size before
the first layout and before every layout at a size it has not heard: the
desktop resizing or maximizing it, `SetSize`, a new display scale that
changes it. It runs ahead of that layout, so content can change its
proportions there and be arranged at the new size in the same frame —
Settings keeps its theme browser at its share of the window that way. An
echoed resize at the same size is not news, and the function it returns
unregisters the hook.

`WindowOptions.Scale` is how an explicit app scale (`Options.Scale`,
`UITK_SCALE`) reaches that conversion: without it a look drawn at 2×
would sit in a window sized for 1×. `platform.Offscreen.SimulateScale`
does the same for tests, so the whole path is exercised headlessly.

## Status item / tray (0.16.0, 0.17.0, 0.18.1)

`StatusItem` is a tray icon + desktop toast. Linux uses StatusNotifierItem
and freedesktop Notifications over the session bus (X11, Xlibre, and
Wayland). Windows uses `Shell_NotifyIcon`; macOS (`CGO`) uses
`NSStatusItem`. See [tray.md](tray.md).

**0.18.1:** HostMenu empty-menu / `ItemIsMenu` / `SecondaryActivate` /
`IconName` fallback / dbusmenu `Version=3`. See [tray.md](tray.md).

**0.17.0:** default **HostMenu** (`Menu=/MenuBar`, real dbusmenu rows).
**ToolkitMenu** is `Menu=/NO_DBUSMENU` plus a reused toolkit popup.
`Application.Post` wakes via eventfd on Wayland. `GetLayout` stays a
finite `(ia{sv}av)` (v0.16.1). Watcher `NameOwnerChanged` re-registers
the item.

**0.16.1:** dbusmenu `GetLayout` uses a finite `(ia{sv}av)` signature so
Plasma's StatusNotifierWatcher does not panic the app. Tray setup
failures fall back to a stub.

`Window.Show` / `Hide` / `Raise` map the surface (X11 `_NET_ACTIVE_WINDOW`;
Wayland remaps `xdg_toplevel` + `xdg_activation_v1`). Offscreen tracks a
visibility flag.

**0.16.4:** SNI `Menu=/` (spec empty path). Toolkit tray menu is a
dedicated popup window at the SNI `(x,y)` when the backend can place
it. `Post` wakes the UI loop.

**0.16.3:** SNI `Menu=/NO_DBUSMENU`; toolkit popup ignored out-of-window
screen coords (bottom-right anchor). Plasma treated the path as
dbusmenu and skipped `ContextMenu`.

**0.16.2:** Mail tray uses `IconMail`. Wayland close-to-tray can restore
(the protocol has no unset_minimized). Tray context menu is a toolkit
`PopupMenu` via SNI `ContextMenu`. Callbacks post to the UI loop.

## Window frames

Who draws a window's frame — the desktop or uitoolkit — and how an app puts
its own title bar in it (`Window.SetTitleBar`, `widgets.HeaderBar`) is in
[decorations.md](decorations.md). The platform side is the optional
`FrameSurface` capability (negotiated `Decorations`, `WindowState`,
`Capabilities`, `StartSystemMove`, `StartSystemResize`, `ShowWindowMenu`,
`Minimize`, `SuitsClientFrame`, `SetFrame`), implemented by the Wayland and
X11 top-level surfaces and by `Offscreen`, which records the calls for
tests.

### The window's dress

Two more optional capabilities say how the desktop should dress a window
it lists or frames (`platform/windowdress.go`), and `Offscreen` records
both (`FrameCalls.Palettes`, `.Icons`; `SimulateDecorationPalette`):

| | Wayland | X11 |
| --- | --- | --- |
| Frame palette (`DecorationPaletteSurface`) | `org_kde_kwin_server_decoration_palette_manager.create` on the `wl_surface` the first time there is a palette, then `set_palette(path)` once per change; `set_palette("")` gives KWin's own colours back, `release` goes with the surface. Only where KWin advertises the manager | `_KDE_NET_WM_COLOR_SCHEME` (STRING, the path) while the window manager is KWin; removed for none |
| Icon (`IconSurface`) | `xdg-toplevel-icon-v1`: one `wl_shm` pool, an ARGB8888 (premultiplied, stride 4 × side) buffer per square size, `create_icon`, `add_buffer(buffer, 1)` for each, `set_icon(toplevel, icon)`; a new icon replaces it and the old icon is destroyed before its buffers. Re-sent on every new role (Show after Hide). KWin asks for 96 px (`icon_size`) | `_NET_WM_ICON`: width, height and straight (not premultiplied) ARGB CARDINALs per size, smallest first |

Both are put back on an X window made afresh on another visual.
`platform/wlfake_linux_test.go` is a compositor on a socket pair
(`WAYLAND_SOCKET`) that checks the Wayland requests on the wire and reads
the icon's pixels back through the pool's descriptor.

### `_NET_WM_SYNC_REQUEST`

An X11 toplevel lists `_NET_WM_SYNC_REQUEST` in `WM_PROTOCOLS` and names an
XSync counter in `_NET_WM_SYNC_REQUEST_COUNTER` before it is mapped. A
request's value is answered (`XSyncSetCounter`) by the first frame put on
the window after the `ConfigureNotify` that resized it, once its pixels
are on the connection; a configure that kept the size is answered at once,
and a request nothing answers within 250 ms is answered anyway (the loop
wakes for it), so the window manager never stalls on an app that did not
repaint (`platform/xsync.go` is the state machine). `UITK_X11_SYNC=0`
leaves the protocol out.

### The frame's margin, regions and buffers

`SetFrame(platform.Frame)` is how a toolkit-drawn frame tells the window
system about itself, in device pixels: `Margin` is the invisible band
around the visible window where its drop shadow lives, `Input` how far into
that band a press still reaches the window (the resize handles; the rest
clicks through), `Radius` the visible window's corner radii and `Alpha`
whether the buffer needs an alpha channel at all. The surface grows by the
margin at once — `Surface.Size()` is the buffer, `Surface.Resize(w, h)` and
`EventResize` are the *window* — and the window system hears about it with
the next present, in the same commit as the pixels it describes.
`platform.FrameMargin` picks a margin that is a whole number of logical
pixels at the surface's scale (Wayland states geometry in logical pixels,
so 10 device px at 1.75 becomes 12 logical = 21 device).

A frame with neither shadow nor rounded corners (Windows 95, Motif, a
maximized or tiled window, an X11 screen with no compositing manager) asks
for nothing: opaque buffers, the full opaque region, the infinite input
region — byte for byte the path every window took before.

| | Wayland | X11 |
| --- | --- | --- |
| Visible window | `xdg_surface.set_window_geometry` (logical px), re-stated for every frame once stated, so a maximized window is not placed by the margin it used to have | `_GTK_FRAME_EXTENTS` (device px; KWin and Mutter honour it when moving, snapping, tiling and maximizing) |
| Input | `wl_surface.set_input_region` = the window grown by the resize band; `NULL` (infinite), never an empty region, when there is no margin | XShape `ShapeInput` rectangle; the mask is reset when the margin goes |
| Opaque | `wl_surface.set_opaque_region` = the window less its rounded corners (rounded inward); the whole surface when opaque | `_NET_WM_OPAQUE_REGION`, the same rects |
| Alpha buffers | `wl_shm` **ARGB8888** premultiplied (the slots are remade when the frame starts or stops needing alpha), EGL config with `EGL_ALPHA_SIZE 8`; the GPU device is rebound on the same `wl_surface` when that changes, so the EGL display is never torn down | a 32-bit TrueColor visual with its own colormap: an X window's visual is fixed at creation, so the window is re-created on one the first time a frame asks (before it is ever mapped, in the usual flow) — `XShmPutImage` and EGL then keep the alpha instead of forcing it opaque |
| Silhouette | `set_input_region` = the shape's rectangles (device px converted **outwards**, so no pixel the window covers goes deaf); the transparent pixels of the ARGB buffer are what you see | `ShapeInput` *and* `ShapeBounding` from the same rectangles, `YXBanded` since the rasteriser emits scanline order — the bounding shape is what cuts the pixels on a screen with no compositing manager |
| Blur behind | `ext_background_effect_v1` (`get_background_effect` per surface, `set_blur_region`, double-buffered like the rest of the frame); the manager's `capabilities` event comes and goes as desktop effects are switched | `_KDE_NET_WM_BLUR_BEHIND_REGION` (CARDINAL rects; an empty property means the whole window). KWin reads it, everything else ignores it |
| Composited? | always | `_NET_WM_CM_S<screen>` at connect, watched with XFixes: without an owner every window reports `WindowState.Solid` and its frame goes square and shadowless, live |

`platform.Frame`'s `Shape`, `Opaque` and `Blur` are region lists, so `Frame`
is no longer comparable with `==`: every backend compares with `Frame.Same`.
A nil `Shape` is an ordinary rectangular window and takes byte for byte the
path it took before shapes existed. See [docs/shapes.md](shapes.md) for the
silhouette itself, the per-state rules and what it costs.

ext-background-effect is generated from wayland-protocols staging like the
rest:

```bash
wayland-scanner client-header \
  /usr/share/wayland-protocols/staging/ext-background-effect/ext-background-effect-v1.xml \
  platform/ext-background-effect-v1-client-protocol.h
wayland-scanner private-code \
  /usr/share/wayland-protocols/staging/ext-background-effect/ext-background-effect-v1.xml \
  platform/ext-background-effect-v1-protocol.c
```

linux-dmabuf stays on the opaque `XRGB` formats; a frame that needs alpha
presents through `wl_shm` instead (the dmabuf path is opt-in and CPU-only).

## Popups

Menus, submenus, combo lists, context menus, the drop-action menu and
tooltips open as surfaces of their own, so they may run past their window —
Minim's 275x116 strip opens its skin menu below itself, a combo list in a
small dialog reaches the bottom of the screen. The code is
`platform/popup.go` (the vocabulary), `wayland_popup_linux.go`,
`x11_popup_linux.go`, `offscreen_popup.go`, and `app/popupsurf.go`.

| | Wayland | X11 |
| --- | --- | --- |
| Surface | `xdg_popup` from the window's (or the parent menu's) `xdg_surface` | override-redirect window, `_NET_WM_WINDOW_TYPE_POPUP_MENU` / `_TOOLTIP`, transient for the window |
| Placement | `xdg_positioner`: anchor rect, anchor edge, gravity, flip / slide / resize, `set_reactive`; the compositor places it | `platform.SolvePopup` — the same semantics — in the work area of the monitor under the anchor (RandR box cut to `_NET_WORKAREA`) |
| A menu's input | `xdg_popup.grab` with the serial of the press or key that opened it | owner-events `XGrabPointer` + `XGrabKeyboard` once the window is mapped (retried while another client holds a grab) |
| Dismissed by the desktop | `popup_done` → `EventPopupDone` | the window losing the keyboard for real (`FocusOut`, not a grab's) → `EventPopupDone` |
| Moved | `xdg_popup.reposition` (v3), answered synchronously | `XMoveResizeWindow` |
| A tooltip | no grab, empty input region | no grab, empty input shape |

**Input stays the window's.** Every event on a popup's surface is queued on
its *root* window with the position moved into the root's device pixels
(`PopupSurface.Origin`), so the app's hit-testing, hover, capture and
click-outside dismissal are unchanged — the popup component is still the
window's popup layer and its accessibility tree is the same tree. A pointer
or keyboard moving between a window and its own popups is not the window
losing it: a leave followed by an enter in the same family, in one burst of
events, is no event at all. Only input goes to the root; a popup's own
configure, resize or close never does.

Placement is stated in **logical pixels relative to the root's visible box**
whatever the parent, and the app arranges the component where the window
system answered (`Placed`, `Origin` — rounded to the root's device grid, so a
menu is as crisp at 1.75 as at 1). Before the answer, the widgets place the
popup against `Window.PopupArea`: the work area on X11 and offscreen, and on
Wayland — where a client never learns where it is — a screen-sized guess
around the window from `configure_bounds`, which the compositor corrects.

**Fallback.** Headless and offscreen windows draw popups inside themselves as
they always did; so does a window whose compositor refused a popup (it is
remembered, and logged once). `UITK_POPUPS=layer` forces it everywhere, for
comparison; `Window.PopupSurfaces` says which a window has. Tests turn the
surfaces on with `Offscreen.SimulatePopups(workArea)` and can refuse them
(`SimulatePopupsRefused`) or take one down (`SimulatePopupDone`).
`UITK_POPUP_DEBUG=1` logs X11 placements.

Known: under Xwayland an X client's grab cannot see a press on a surface that
is not an X window, so a click on the bare desktop does not dismiss an X11
menu there (Qt and GTK share it); on a real X server it does. KWin states
`_NET_WORKAREA` in logical pixels on a scaled Xwayland while windows are
placed in device ones; a work area under 60% of its monitor is ignored for
the monitor.

### Absolute placement (layer shell)

A popup hangs from a window, and the window system places it. A **tray
menu** does not: the host names a point in root coordinates
(`org.kde.StatusNotifierItem.ContextMenu`) and the menu belongs there.
X11 and the offscreen desktop simply place the window. A Wayland client
cannot place an `xdg_toplevel` at all — so `WindowOptions.X, Y` were
silently ignored there, and KWin dropped the toolkit's tray menu in the
middle of the screen.

`WindowOptions.Place` says which of the two a window means:

| | `PlaceDesktop` (zero) | `PlaceAtScreen` |
| --- | --- | --- |
| Meaning | `X, Y` are a hint | the window belongs at `X, Y` and nowhere else |
| X11 / offscreen | placed | placed |
| Wayland, layer shell | `xdg_toplevel`, compositor places it | **`zwlr_layer_shell_v1` surface**, placed |
| Wayland, no layer shell | `xdg_toplevel` | `xdg_toplevel` — the caller was told in advance |

`platform.ScreenPlacementAvailable()` is the question to ask **before**
committing to something that needs a position; `PlaceSurfaceAtScreen(s,
x, y)` moves such a window and reports whether it went (the `ScreenPlacer`
capability, which is deliberately not `HostMover` — Wayland still does
not place windows in general). `LayerSurfacesAvailable()` and
`LayerShellVersion()` are the Wayland-specific answers.

The layer surface (`platform/wayland_layer_linux.go`) is configured:
`overlay` layer, so a menu is above the panel it was opened from; anchor
`top|left`, which turns the margins into an absolute position;
`set_margin(top, 0, 0, left)` = the requested point; `set_size` = the
whole surface, margin included, because a layer surface has no xdg window
geometry to state a visible box with; `set_exclusive_zone(-1)` so the
menu neither pushes panels around nor is pushed; keyboard interactivity
`on_demand` (v4+, else the old boolean) so the menu can hold the keyboard
for its grab. `configure` is acked on the layer-surface object — there is
no `xdg_surface` underneath — and `closed` becomes an `EventClose`, after
which `Show` makes a fresh role.

**Multi-monitor.** Layer-shell margins are measured from the edges of one
output, and the output is fixed when the role is created. The toolkit
tracks `wl_output.geometry` and `.mode`, finds the output whose logical
rectangle holds the requested point, anchors to *that* output and
subtracts its origin from the margins; a point no output claims is passed
as-is with no output, letting the compositor choose (right on a single
monitor, a guess anywhere else). Moving the window to a point on another
monitor re-creates the role, because the protocol has no `set_output`.

**Compositors.** KDE (KWin, v5 here), sway, Hyprland and wayfire offer
layer shell. GNOME/Mutter has declined it; there
`ScreenPlacementAvailable` is false and `app.NewStatusItem` falls back to
the desktop-drawn tray menu (see [tray.md](tray.md)). `UITK_LAYER_SHELL=0`
forces that road on a compositor that has the protocol.

**Not tested against a compositor.** The suite covers the fallback
decision, the output arithmetic (`outputSet.layerMargins`), the configure
size arithmetic and the `WindowOptions` plumbing;
`platform.SimulateScreenPlacement(bool)` is the hook that lets a headless
test walk both roads. Nothing here has an automated test on a real
Wayland session.

## Drag and drop

Both halves on both backends: a window takes drops from any application,
and a drag started in one is carried to any other. The toolkit-facing API
is in [widgets.md](widgets.md); what follows is what the backends do.

Shared, and testable without a display (`platform/drag.go`,
`platform/xdnd.go`): `DragAction` is a bit set of copy / move / link, so a
source can offer several and a target answer with one;
`NegotiateDragAction` settles what a drop performs; `ModifierDragAction`
is the desktop-wide convention (Shift moves, Ctrl copies, both link) for
the backend that has to apply it itself.

The seams a surface implements are `DragSurface` (`StartDrag`,
`CancelDrag`, `Dragging`), `DropReceiver` (`ReceiveDrop`, `FinishDrop`)
and `DropNegotiator` (`AcceptDrag`). `AcceptDrag` takes both the set of
actions the target allows and the one it would take now: a compositor
picks from the set with the user's modifiers, so a target that named only
its current choice would pin the drag to it. `EventDragEnd` reports the
action the target performed and `Dropped`, whether the pointer was let go
at all — a drag that ends with nothing performed is a window left on the
desktop when it was dropped and a window that should never have existed
when it was cancelled.

A drag can also carry a window: `ToplevelDragSurface` (`DragsToplevels`,
`AttachToplevel`) is what tear-off is built on (see
[decorations.md](decorations.md#tear-off)). Wayland binds
`xdg_toplevel_drag_manager_v1` and makes the drag object from the data
source before `start_drag`, which is the only moment the protocol allows;
X11 moves the window itself on every motion, since an X11 client places
its own windows, and leaves it out of the search for a target. A window's
position comes back through `HostPositioner` (`Position`), which X11
answers from the last `ConfigureNotify` and Wayland does not implement at
all: a toplevel has no position there, which is why the drag protocol
exists.

**Wayland.** Taking a drop is `wl_data_device` — the offer is accepted
for the type the window will read, `set_actions` says what it would do,
and the drop is read from a pipe and finished. Dragging out is
`wl_data_source` with the offered types and `set_actions`, and
`wl_data_device.start_drag` from the serial of the press that began it:
without that serial a compositor refuses the drag, and rightly, since
nothing else proves a user asked for it. The picture that follows the
pointer is a surface of its own with an ARGB `wl_shm` buffer, attached
with the hotspot as its offset. `dnd_drop_performed` takes the picture
away while the source stays alive to answer for the data;
`dnd_finished` (or `cancelled`) reports the action the target performed.
Escape needs nothing from the client: the compositor owns the pointer
during a drag and cancels it itself.

**X11** is XDND version 5, the whole conversation
(`platform/x11_dnd_linux.go`). Every toplevel advertises `XdndAware`;
as a target the window answers each `XdndPosition` with an `XdndStatus`
(which is what tells the source whether the drop will be taken, and with
which action), reads `XdndTypeList` when a source offers more than three
types, converts `XdndSelection` at the timestamp the drop named — so a
second drag cannot hand over the first one's data — in ICCCM **INCR**
pieces when it is too big for one property, and closes with
`XdndFinished`.

As a source it takes `XdndSelection`, grabs the pointer and the keyboard,
and walks the window tree on every move to find the target under the
pointer: down through a reparenting manager's frame, following
`XdndProxy` (which is read from, and messaged at, the proxy while the
messages still name the window under the pointer), and passing over its
own icon window. An **input-only** window counts — the proxy a compositor
puts up to bridge an X11 drag to a Wayland client is exactly that. The
cursor shows what the last `XdndStatus` agreed to. On release it sends
`XdndDrop`, gives the pointer back at once, and keeps serving the
selection until `XdndFinished` or a timeout, which the spec tells a
source to have rather than block on a misbehaving target. The drag's
picture is an override-redirect window on the ARGB visual with an empty
input shape; without such a visual the drag runs on the cursor alone.

Escape reaches an X11 drag two ways: as a key event on the grab, taken
before `XFilterEvent` so a composing input method cannot swallow it, and
by asking the server which keys are down — there are setups where a
grabbed key event never arrives at all. `UITK_XDND_DEBUG=1` traces a
drag's progress to stderr.

**Offscreen** implements the source side too, so all of this is testable
with no display: it stands in for the desktop, and a test moves the drag,
drops it, and reads back what the source was told
(`platform/offscreen_drag.go`).

## Touchpad gestures and scrolling

`EventGesture` carries a touchpad's pinch, swipe and hold (`GestureKind`,
`GesturePhase`, `Fingers`, `Delta`, `Scale`, `Rotation`; the contract is in
`platform/gesture.go` and, for widgets, [widgets.md](widgets.md#touchpad-gestures)).
A shared tracker keeps it whatever the window system sends: nothing without a
begin, a begin over an open gesture cancels that one, `Scale` from 1 at the
begin and kept through an end that carries none. A popup's gesture reaches
its root window like its other input.

| | Wayland | X11 |
| --- | --- | --- |
| Gestures | `zwp_pointer_gestures_v1` up to version 3: swipe, pinch and hold objects made for the seat's pointer; deltas converted to device pixels | XInput 2.4 (`XI_GesturePinch*`, `XI_GestureSwipe*`; Xorg 21.1+, Xwayland 22+, libXi 1.8+ to build them): pinch and swipe. **No hold**: XInput has none. A server older than 2.4 sends no gestures at all |
| Two-finger scroll | `wl_pointer.axis` with `axis_source` finger: device pixels, `ScrollPrecise` | XInput 2.1 scroll valuators (below): pixels, `ScrollPrecise` |
| Glide after the fingers lift | `axis_stop` releases it; `kinetic.go` | synthesised: the scroll going quiet for 60 ms is the lift; the same `kinetic.go` curve |
| Glide stops | a new scroll, a click, the pointer leaving, **fingers resting on the pad** (hold) | a new scroll, a click, the pointer leaving, a new pinch or swipe |
| Wheel | `axis` / 10 per notch: `Scroll` ±1, not precise; never glides | the valuator's increment is a notch: `Scroll` ±1 (a high-resolution wheel's fractions as fractions), not precise; never glides. Emulated buttons 4–7 are dropped by their `XIPointerEmulated` flag; real ones (XTest, a device without valuators) are notches |

**X11's smooth scrolling.** XInput reports a scroll as motion on a device's
scroll valuators — a running total per axis, and an increment that is one
notch — and says neither where the scroll came from nor when the fingers
lifted. So, as GTK and Qt do: a touchpad is known by its XInput properties
(`libinput Tapping Enabled`, `Synaptics Finger`) or its name; a device with
libinput or evdev properties and neither is a mouse. Xwayland has one pointer
for every device and cannot say, so there the scroll itself is read: a wheel
moves in whole 120ths of a notch (what `value120` is), fingers by accelerated
fractions of a pixel that almost never land on one; a run that showed a
finger stays one until it pauses. A finger's scroll is `pxPerNotch` pixels a
notch — the driver's `libinput Scrolling Pixel Distance` where there is one,
Xwayland's 10 where not — times the display scale. The valuators are seeded
with where they stand at a device switch (from the `XI_DeviceChanged` event)
and on entering a window (asked of the server), so the first notch after is
not lost. Motion, buttons and enter are selected through XInput 2 on the
toplevels and popups; a press so selected grabs the device the XInput 2 way,
so a system move or resize lets go of that grab too (`XIUngrabDevice`).

`UITK_X11_XI2=0` keeps the core events (wheel buttons, no gestures, no
smooth scrolling); `UITK_X11_DEBUG=1` logs the XInput version, the scroll
devices and every scroll, `=2` every pointer event.

The mouse's thumb buttons are `ButtonBack` and `ButtonForward` (X11 8 and
9, `BTN_SIDE` and `BTN_EXTRA`); the app turns them into history navigation
rather than presses.

Tests: the tracker's contract (`gesture_test.go`), the Wayland objects and
events on the wire against the fake compositor
(`wayland_gesture_linux_test.go`), the X11 classification, notches, the
velocity, the deceleration, stopping on a touch and no glide on a wheel
(`x11scroll_test.go`), and delivery in the window (`app/gesture_test.go`).
The e2e rig's fake input has no gestures and no axis source, so gestures and
the glide are proven headlessly; wheel notches, the thumb buttons and
XInput 2 clicks, menus and system moves were checked in the rig on both
backends.

## Portals

Every call to a desktop service checks first that it is running or that
D-Bus can start it (`NameHasOwner`, `ListActivatableNames`) and carries a
deadline: two seconds for a running service, five for one D-Bus has to
start. With no portal, the fallback — the toolkit's dialog, xdg-open, an
`ErrNoNotifications` — comes at once. A notification server is never
started by activation (its activatable name can be a helper that waits for
a shell).

**A parent for the desktop's dialogs (xdg-foreign).** A surface that is a
`PortalParenter` names itself for the portal's `parent_window`: `x11:<hex
id>` on X11; on Wayland the toplevel is exported with `zxdg_exporter_v2`
the first time it is asked, and the compositor's handle becomes
`wayland:<handle>`, kept for as long as the window has its role (Hide and
Close revoke it; a compositor without the exporter gives no parent). The
desktop's dialog is then the window's child: KWin keeps it above the window,
modal and centred on it. `FileDialogOptions.Native` uses it; so do
`OpenURI` and `widgets.OpenLink`.

**OpenURI.** `platform.OpenURI(uri, OpenURIOptions)` opens a link through
`org.freedesktop.portal.OpenURI` (`ParentWindow`, `Ask` for "Open with…",
`ActivationToken`); a local file goes by descriptor to `OpenFile`, a folder
to `OpenDirectory`, as the portal wants. No portal, or a portal that
refuses: `xdg-open`. A URI that looks like an option is refused.
`UITK_OPENURI=xdg-open` skips the portal, `off` opens nothing.
`Window.OpenURI` and `widgets.OpenLink` do it off the UI goroutine.

**Notifications.** `platform.Notifier` (`Application.NewNotifier`, which
dispatches clicks on the UI goroutine and sends nothing from a headless app)
sends a `DesktopNotification` — `ID` (the same ID replaces it), title,
body, a theme icon name or pixels, buttons, priority — and calls
`OnActivate` with `""` for a click on it or the button's ID. It speaks both
services:

| | `org.freedesktop.portal.Notification` | `org.freedesktop.Notifications` |
| --- | --- | --- |
| Send | `AddNotification(id, a{sv})`: `title`, `body`, `icon` a serialised GIcon (`("themed", names)` or `("bytes", PNG)`), `priority`, `default-action`, `buttons` `aa{sv}` | `Notify(app_name, replaces_id, icon, summary, body, actions, hints, -1)`: actions `default`, then a key and label per button; hints `urgency`, `desktop-entry`, `image-data` for pixels |
| Replace | the same id | `replaces_id` of the last one with that ID |
| Click | `ActionInvoked(id, action, …)` | `ActionInvoked(id, key)`; `NotificationClosed` forgets it |
| Withdraw | `RemoveNotification` | `CloseNotification` |

Which: in a Flatpak or Snap sandbox the portal (the only way out); outside
one the notification server when it is running — what GTK's and KDE's
unsandboxed apps use, since the portal names an unsandboxed app by nothing
and GNOME's cannot show a notification for it — and the portal otherwise.
`UITK_NOTIFY=portal|fdo|off` decides outright. Button action names on the bus
are the notifier's own (`button-N`), never the app's, so any ID the app
likes is valid.

Tests run every call against fakes on a private `dbus-daemon` that can
start nothing (`portal_linux_test.go`): the exact arguments, the clicks
back, replacement, the order, and the quick failures with nothing on the
bus. In the e2e rig, `tools/e2e/fakeportal` stands in for the portal and the
notification server on the instance's own bus, and its `importer` helper
makes the file dialog a real window that imports the handle (see
[tools/e2e/README.md](../tools/e2e/README.md)).

## Deferred

- IME candidate-window theming (the IM draws its own window)
- The extended `_NET_WM_SYNC_REQUEST` (a second counter and
  `_NET_WM_FRAME_DRAWN` / `_NET_WM_FRAME_TIMINGS`): the basic handshake is
  in place
- Win32 and AppKit **windows** (tray landed in 0.16.0)
