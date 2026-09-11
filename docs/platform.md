# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
CPU pixmap onto a window and translates input.

| Backend | When | Present | Clipboard | Scale | IME |
| --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a |
| **Wayland** | Linux + CGO + `WAYLAND_DISPLAY` | **linux-dmabuf** (`zwp_linux_dmabuf_v1`) when advertised + GBM/dma-heap/udmabuf works, else `wl_shm` ARGB8888; damage; integer / fractional buffer scale | `wl_data_device` + primary when the compositor offers it | `wl_output` scale, `wp_fractional_scale_v1` + viewporter, env | `zwp_text_input_v3` preedit / commit |
| **X11** | Linux + CGO + `DISPLAY` | dirty-rect `XPutImage`, MIT-SHM when the server allows it | CLIPBOARD + PRIMARY, ICCCM **INCR** | Xft.dpi, RandR mm, screen mm, env | XIM preedit callbacks + compose / dead keys |
| Win32 / AppKit | stub | — | — | — | — |

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
```

CGO Linux links `libX11`, `libXext`, `libXrandr`, `libwayland-client`,
and `libxkbcommon`. `CGO_ENABLED=0` never needs those libraries.

## X11 notes (0.3.0)

- One shared `Display` for all windows. Destroying a window frees its
  `XImage` / MIT-SHM segment, GC, XIC, and `XID`; the connection closes
  when the last surface is gone and no selection is owned.
- Present copies premul RGBA into a 32-bit ZPixmap using the visual
  masks and `XImage` byte order / `bytes_per_line`. MIT-SHM
  (`XShmPutImage`) is used when `XShmAttach` succeeds; Xvfb, SSH, and
  locked-down servers fall back to `XPutImage` of each damage rect.
- Resize uses `NorthWestGravity`, `XResizeWindow`, and a rebuilt pixmap;
  the toolkit full-repaints so expose after configure is not garbage.
- EWMH: `Window.SetFullscreen` / `SetMaximized` send `_NET_WM_STATE`.
  Close is `WM_DELETE_WINDOW`.
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

## Wayland notes (0.3.1)

- `wl_display_connect` → registry bind of `wl_compositor`, `wl_shm`,
  `xdg_wm_base`, `wl_seat`, `wl_data_device_manager`, `wl_output`, and
  optional `zwp_linux_dmabuf_v1`, `zwp_text_input_manager_v3`,
  `zwp_primary_selection_device_manager_v1`,
  `zxdg_decoration_manager_v1`, `wp_fractional_scale_manager_v1`,
  `wp_viewporter`.
- Each window is an `xdg_toplevel`. Configure width/height are
  surface-local (logical); the present buffer is `ceil(logical * scale)`.
  States maximized / fullscreen / resizing / activated are parsed.
  `xdg_toplevel.close` is `EventClose`. Server-side decorations are
  requested when `xdg-decoration` is present.
- Present prefers **linux-dmabuf** when all of the following hold:
  1. `UITK_WAYLAND_PRESENT` is `auto` (default) or `dmabuf` (not `shm`)
  2. The compositor advertises `zwp_linux_dmabuf_v1` (v2+) and a
     CPU-linear format: `DRM_FORMAT_ARGB8888` / `XRGB8888` (also
     `ABGR8888` / `XBGR8888` if offered), via `format`/`modifier`
     events or `get_default_feedback` (protocol v4+)
  3. A local dmabuf can be created: **GBM/DRM** linear BO
     (`libgbm.so` + `/dev/dri/renderD*` or `card*`), else
     `/dev/dma_heap/system`, else memfd + `/dev/udmabuf`
- Otherwise present uses the existing two `wl_shm` pools (memfd-style
  temp files). Xvfb, Weston headless without GBM, missing protocols,
  and `UITK_WAYLAND_PRESENT=shm` always land on shm. A failed dmabuf
  import on the first buffer falls back to shm for that connection.
- Both paths attach premul 8-bit pixels from `paintengine2d.Image.Pix`
  (`RowStride`). ARGB8888/XRGB8888 swizzle RGBA→BGRA; ABGR8888 is a
  row copy. Then `wl_surface_damage_buffer` (or `wl_surface_damage`),
  attach, commit. Integer `wl_surface.set_buffer_scale` or
  fractional-scale + viewport destination. Resize rebuilds both
  pixmap and the free present slots so attach/damage stay aligned.
- Seat: pointer (motion, buttons, axis) with coordinates multiplied by
  buffer scale; keyboard via **xkbcommon** (keymap, mods, UTF-8,
  compose / dead keys) plus compositor `repeat_info`.
- Clipboard: `wl_data_device` copy/paste (`text/plain;charset=utf-8`).
  Primary selection when the compositor binds
  `zwp_primary_selection_v1` (Weston does). Middle-click paste uses
  `ClipboardPrimaryGet`.
- IME: `zwp_text_input_v3` enable on keyboard enter; `preedit_string`,
  `commit_string`, and `delete_surrounding_text` become `EventIME*`.
  Cursor rectangle is updated from the focused TextField / TextArea.
  When text-input has entered the surface, printable UTF-8 from xkb is
  suppressed so commit is not doubled.

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
# linux-dmabuf-unstable-v1
```

```bash
# Force present path (Wayland + CGO only)
UITK_WAYLAND_PRESENT=auto go run ./examples/gallery    # dmabuf if possible
UITK_WAYLAND_PRESENT=shm go run ./examples/gallery     # always wl_shm
UITK_WAYLAND_PRESENT=dmabuf go run ./examples/gallery  # prefer dmabuf, shm fallback
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
metrics and buffer scale agree. dmabuf and shm share that scale /
damage / attach / commit path.

## Deferred

- AT-SPI / accessibility
- IME candidate-window theming (the IM draws its own window)
- Wayland explicit sync (implicit dma-buf reservation only)
- Client-side decoration chrome beyond the existing TitleBar widget
- Win32 and AppKit
