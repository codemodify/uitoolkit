# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
CPU pixmap onto a window and translates input.

| Backend | When | Present | Clipboard | Scale | IME |
| --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a |
| **Wayland** | Linux + CGO + `WAYLAND_DISPLAY` | `wl_shm` ARGB8888 | in-process only | env only | none (no text-input-v3) |
| **X11** | Linux + CGO + `DISPLAY` | dirty-rect `XPutImage` | CLIPBOARD + PRIMARY | Xft.dpi, screen mm, env | XIM compose / dead keys |
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

CGO Linux links `libX11`, `libwayland-client`, and `libxkbcommon`.
`CGO_ENABLED=0` never needs those libraries.

## X11 notes (0.1.8+)

- One shared `Display` for all windows. Destroying a window frees its
  `XImage`, GC, XIC, and `XID`; the connection closes when the last
  surface is gone and no selection is owned.
- Present copies premul RGBA into a 32-bit ZPixmap using the visual
  masks and `XImage` byte order / `bytes_per_line`, then `XPutImage`
  of each damage rect. MIT-SHM is not used.
- Resize uses `NorthWestGravity` and rebuilds the pixmap; the toolkit
  full-repaints so expose after configure is not garbage.
- `ClipboardSet` owns **CLIPBOARD** and **PRIMARY**. `ClipboardGet`
  converts UTF8_STRING (then XA_STRING). INCR (very large pastes) is
  not implemented.
- Input: `XFilterEvent` + `Xutf8LookupString` on an XIC created with
  `XIMPreeditNothing | XIMStatusNothing`. Dead keys and compose work.
  **Not implemented:** on-the-spot / over-the-spot preedit, candidate
  windows, or CJK composition UI.

## Wayland notes (0.2.0)

- `wl_display_connect` → registry bind of `wl_compositor`, `wl_shm`,
  `xdg_wm_base`, `wl_seat`. Each window is an `xdg_toplevel`.
- Present: two `wl_shm` pools (memfd-style temp files), ARGB8888
  (little-endian B,G,R,A), `wl_surface_damage_buffer` (or
  `wl_surface_damage` on compositor v3), attach, commit. dmabuf / GPU
  compositors are deferred.
- Seat: pointer (motion, buttons, axis) and keyboard via **xkbcommon**
  (keymap fd, keysyms, UTF-8). Mapped onto the existing `Event` types.
- Configure / close: `xdg_toplevel.configure` resizes the pixmap;
  `xdg_toplevel.close` is `EventClose`. First commit waits for
  configure before attaching a buffer.
- **Gaps:** no `zwp_text_input_v3` IME, no `wl_data_device` clipboard,
  no `wp_fractional_scale` / buffer scale (use `UITK_SCALE`), no
  decorations protocol (the compositor draws CSD/SSD).

xdg-shell C is generated from wayland-protocols and committed:

```bash
wayland-scanner client-header \
  /usr/share/wayland-protocols/stable/xdg-shell/xdg-shell.xml \
  platform/xdg-shell-client-protocol.h
wayland-scanner private-code \
  /usr/share/wayland-protocols/stable/xdg-shell/xdg-shell.xml \
  platform/xdg-shell-protocol.c
```

## HiDPI

`Options.Scale <= 0` detects scale (`UITK_SCALE`, `GDK_SCALE`,
`QT_SCALE_FACTOR`, then Xft.dpi / 96 on X11). The Classic look is
rebuilt with `style.ScaleMetrics` so control heights and baked glyph
atlases grow. Logical-pixel windows are not used; the buffer stays
device pixels. Wayland does not yet read compositor scale.
