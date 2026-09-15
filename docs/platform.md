# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
Device onto a window and translates input. `UITK_PAINT=auto` (default)
tries `GPUDevice` and falls back to the CPU pixmap.

| Backend | When | Present | Clipboard | Scale | IME | Cursor |
| --- | --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a | last `SetCursor` |
| **Wayland** | Linux + CGO + `WAYLAND_DISPLAY` | **`wl_egl_window` + `eglSwapBuffers`** when `UITK_PAINT=auto\|gpu` and EGL works; else v0.4.1 **`wl_shm` XRGB8888** (opaque). **linux-dmabuf** only if `UITK_WAYLAND_PRESENT=dmabuf` on the CPU path | `wl_data_device` + primary when the compositor offers it | `wl_output` scale, `wp_fractional_scale_v1` + viewporter, env | `zwp_text_input_v3` preedit / commit | host cursors: `wp_cursor_shape_v1` when advertised, else `wl_cursor_theme` (`XCURSOR_THEME` / `XCURSOR_SIZE`) + `wl_pointer_set_cursor` (enter serial); re-applied on pointer enter |
| **X11** | Linux + CGO + `DISPLAY` | **EGL window + `eglSwapBuffers`** when EGL works; else dirty-rect `XPutImage` / MIT-SHM | CLIPBOARD + PRIMARY, ICCCM **INCR** | Xft.dpi, RandR mm, screen mm, env | XIM preedit callbacks + compose / dead keys | host cursors: `XcursorLibraryLoadCursor` theme names, else `XCreateFontCursor` + `XDefineCursor` |
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

CGO Linux links `libX11`, `libXext`, `libXrandr`, `libwayland-client`,
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
  is mapped (removed again for the window manager's frame); button presses
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
  Focus-out calls `XmbResetIC` and `EventIMECancel`.
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
  `wp_viewporter`, `xdg_activation_v1`, `wp_cursor_shape_manager_v1`.
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
# linux-drm-syncobj-v1, xdg-activation-v1, cursor-shape-v1
```

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
the pixmap. Resize events stay in **logical** units — feeding buffer
pixels back through `Resize` would re-apply scale and blow up layout.
Set `UITK_SCALE` to the compositor scale (often `2`) so metrics and
buffer scale agree. List/table/tree rows and fixed column widths are
design pixels that grow with the look. EGL, dmabuf, and shm share that
scale; CPU present still uses damage / attach / commit.

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
`Minimize`, `SuitsClientFrame`), implemented by the Wayland and X11
top-level surfaces and by `Offscreen`, which records the calls for tests.

## Deferred

- IME candidate-window theming (the IM draws its own window)
- Themed window frames, shadows and rounded corners for toolkit-drawn frames
  (Phases 2–3 of [decorations.md](decorations.md#phases))
- Win32 and AppKit **windows** (tray landed in 0.16.0)
