# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
CPU pixmap onto a window and translates input.

| Backend | When | Present | Clipboard | Scale | IME |
| --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a |
| **Wayland** | Linux + CGO + `WAYLAND_DISPLAY` | **`wl_shm` XRGB8888** (opaque) by default; **linux-dmabuf** only if `UITK_WAYLAND_PRESENT=dmabuf` and a linear XRGB/XBGR (else ARGB) allocator works; damage-only upload; `wl_surface.set_opaque_region` | `wl_data_device` + primary when the compositor offers it | `wl_output` scale, `wp_fractional_scale_v1` + viewporter, env | `zwp_text_input_v3` preedit / commit |
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
- Present copies premul RGBA into a 32-bit ZPixmap. Typical LE
  TrueColor (`red 0xff0000`) uses the same damage-rect uint32
  RGBA→BGRA upload as Wayland; unusual visuals still pack via masks
  and `XImage` byte order / `bytes_per_line`. MIT-SHM
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

## Wayland notes (0.4.1)

- `wl_display_connect` → registry bind of `wl_compositor`, `wl_shm`,
  `xdg_wm_base`, `wl_seat`, `wl_data_device_manager`, `wl_output`, and
  optional `zwp_linux_dmabuf_v1`, `zwp_linux_explicit_synchronization_v1`,
  `wp_linux_drm_syncobj_v1`, `zwp_text_input_manager_v3`,
  `zwp_primary_selection_device_manager_v1`,
  `zxdg_decoration_manager_v1`, `wp_fractional_scale_manager_v1`,
  `wp_viewporter`.
- Each window is an `xdg_toplevel`. Configure width/height are
  surface-local (logical); the present buffer is `ceil(logical * scale)`.
  States maximized / fullscreen / resizing / activated are parsed.
  `xdg_toplevel.close` is `EventClose`. Server-side decorations are
  requested when `xdg-decoration` is present.
- Present defaults to **`wl_shm` `XRGB8888`** (opaque). `auto` and `shm`
  are the same path. **Do not use ARGB8888 for opaque UI**: paintengine2d
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
# linux-dmabuf-unstable-v1, linux-explicit-synchronization-unstable-v1,
# linux-drm-syncobj-v1
```

```bash
# Force present path (Wayland + CGO only)
UITK_WAYLAND_PRESENT=auto go run ./examples/gallery    # wl_shm XRGB8888 (default)
UITK_WAYLAND_PRESENT=shm go run ./examples/gallery     # same as auto; workaround if a window is transparent
UITK_WAYLAND_PRESENT=dmabuf go run ./examples/gallery  # experimental; falls back to shm if the upload is blank
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
- Client-side decoration chrome beyond the existing TitleBar widget
- Win32 and AppKit
