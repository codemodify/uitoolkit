# Platform backends

uitoolkit paints only through paintengine2d. The platform layer maps that
CPU pixmap onto a window and translates input.

| Backend | When | Present | Clipboard | Scale | IME |
| --- | --- | --- | --- | --- | --- |
| **offscreen** | `Headless`, `UITK_BACKEND=offscreen`, or no display | no-op | in-process | env or 1 | n/a |
| **X11** | Linux + CGO + `DISPLAY` | dirty-rect `XPutImage` | CLIPBOARD + PRIMARY | Xft.dpi, screen mm, env | XIM compose / dead keys |
| **Wayland** | stub in 0.1.8 | — | — | — | — |
| Win32 / AppKit | stub | — | — | — | — |

Auto-select: `WAYLAND_DISPLAY` (when a Wayland backend exists) else
`DISPLAY` (X11) else offscreen. Override with `UITK_BACKEND=x11|wayland|offscreen`
or `Application` `Options.Backend`.

```bash
go run ./examples/gallery                 # X11 when DISPLAY is set
UITK_BACKEND=x11 go run ./examples/gallery
UITK_SCALE=2 go run ./examples/gallery    # force 2× metrics
go run ./examples/gallery -headless       # offscreen PNG
CGO_ENABLED=0 go test ./...               # no native windowing
```

## X11 notes (0.1.8)

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

## HiDPI

`Options.Scale <= 0` detects scale (`UITK_SCALE`, `GDK_SCALE`,
`QT_SCALE_FACTOR`, then Xft.dpi / 96). The Classic look is rebuilt with
`style.ScaleMetrics` so control heights and baked glyph atlases grow.
Logical-pixel windows are not used; the buffer stays device pixels.
