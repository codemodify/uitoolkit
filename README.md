# uitoolkit

**Pure-Go desktop UI toolkit.** Retained widget tree, layout, themes, and
X11 and Wayland window backends. Every pixel is painted with
[`github.com/codemodify/paintengine2d`](https://github.com/codemodify/paintengine2d)
(v0.8.0+). Default UI is **Titillium Web**; mono / code is **JetBrains Mono**
(OFL, embedded). Outlines are rasterized through paintengine2d into a white
atlas and tinted with `Paint.Color`. There is no second rasterizer, no Skia,
no Gio renderer, and no Electron.

```go
app := uitoolkit.New(uitoolkit.Options{Look: uitoolkit.DarkLook()})
win, _ := app.NewWindow(uitoolkit.WindowOptions{Title: "Hello", Width: 800, Height: 560})
win.SetContent(uitoolkit.NewColumn(
    uitoolkit.NewTitle("Hello"),
    uitoolkit.NewButton("OK", func() { app.Quit() }),
))
_ = app.Run()
```

```bash
go get github.com/codemodify/uitoolkit@dev
go get github.com/codemodify/paintengine2d@v0.8.0
```

| | |
| --- | --- |
| Language | Go 1.22+ |
| Paint | paintengine2d **v0.8.0** (`b220e31`; `GPUDevice` EGL/GLES2, `UITK_PAINT`, `OpenSurface`) |
| Fonts | Titillium Web (UI) + JetBrains Mono (code), OpenType → atlas |
| Windowing | Linux X11 + Wayland (`wl_egl_window` / eglSwapBuffers, else `wl_shm` / `XPutImage`); offscreen always |
| CGO | optional — tests and screenshots are `CGO_ENABLED=0` |
| License | MIT |

## Screenshots

Real frames from the gallery, Notes, Inspector, and Files, painted through
paintengine2d with Titillium Web (and JetBrains Mono in code views).

### Widget gallery — dark

![Dark gallery](docs/screenshots/gallery-dark.png)

### Widget gallery — light

![Light gallery](docs/screenshots/gallery-light.png)

### Themed controls

![Themed controls](docs/screenshots/widgets.png)

### Scrollable content

![ScrollView](docs/screenshots/gallery-scroll.png)

### Dialog overlay

![About dialog](docs/screenshots/gallery-dialog.png)

### MenuBar drop-down

![File menu](docs/screenshots/gallery-menu.png)

### TreeView

![TreeView](docs/screenshots/gallery-tree.png)

### ToolBar

![ToolBar](docs/screenshots/gallery-toolbar.png)

### ComboBox drop-down

![ComboBox](docs/screenshots/gallery-combo.png)

### MessageBox

![MessageBox](docs/screenshots/gallery-message.png)

### TableView

![TableView](docs/screenshots/gallery-table.png)

### File picker stub

![File dialog](docs/screenshots/gallery-file.png)

### Tooltip

![Tooltip](docs/screenshots/gallery-tooltip.png)

### TextArea

![TextArea](docs/screenshots/gallery-textarea.png)

### Accordion / Switch

![Accordion](docs/screenshots/gallery-accordion.png)

### Notes — a small desktop app

![Notes](docs/screenshots/notes.png)

### Inspector — preferences sample

![Inspector](docs/screenshots/inspector.png)

### Files — projects dogfood

![Files](docs/screenshots/files.png)

### Font roles — Titillium Web + JetBrains Mono

![Font roles](docs/screenshots/fonts.png)

LookAndFeel locks **UI → Titillium Web** and **Mono → JetBrains Mono**
(OFL, embedded). mononoki is not the default mono face.

Regenerate:

```bash
go run ./examples/gallery -screenshot docs/screenshots
```

## Quickstart

```bash
git clone https://github.com/codemodify/uitoolkit.git
cd uitoolkit
CGO_ENABLED=0 go test ./...
go run ./examples/gallery            # Wayland if WAYLAND_DISPLAY, else X11
UITK_BACKEND=x11 go run ./examples/gallery
UITK_BACKEND=wayland go run ./examples/gallery
UITK_PAINT=auto go run ./examples/gallery   # default: GPU if EGL works
UITK_PAINT=cpu go run ./examples/gallery    # v0.4.1 CPU present
go run ./examples/gallery -headless  # writes gallery.png
go run ./examples/notes
go run ./examples/inspector
go run ./examples/files
go run ./examples/files -headless   # writes files.png
```

Headless / CI paints into `paintengine2d.NewImage` and can `Window.WritePNG`.
`UITK_PAINT=auto` (default) tries paintengine2d **GPUDevice** (Linux EGL/GLES2)
and falls back to CPU. On Linux with CGO a GPU window presents with
**eglSwapBuffers** (`wl_egl_window` on Wayland, EGL window on X11). If EGL
fails, present is the v0.4.1 CPU path: **XPutImage** on X11, **wl_shm
XRGB8888** on Wayland (`UITK_WAYLAND_PRESENT=auto|shm`).
`UITK_WAYLAND_PRESENT=dmabuf` opts into linux-dmabuf on the CPU path.
`UITK_PAINT=cpu` forces the CPU painter. If a Wayland window is fully
transparent, set `UITK_PAINT=cpu` and/or `UITK_WAYLAND_PRESENT=shm`.
Auto-select is `WAYLAND_DISPLAY` → `DISPLAY` → offscreen. Scale comes from
`UITK_SCALE` / `GDK_SCALE` / `QT_SCALE_FACTOR` / `GDK_DPI_SCALE`, else
Xft.dpi / RandR on X11 or `wl_output` / fractional-scale on Wayland.
Ctrl+C/X/V and middle-click use the **OS clipboard** on both X11
(CLIPBOARD + PRIMARY, including INCR) and Wayland (`wl_data_device`,
plus primary when the compositor supports it). CJK IME preedit is wired
through XIM callbacks and `text-input-v3` into TextField / TextArea.

## How it uses paintengine2d

uitoolkit does not rasterize. A window paints through paintengine2d
`Context` → `Device`. `UITK_PAINT=auto` binds `GPUDevice` to the native
window when EGL works; otherwise the window owns a premul RGBA pixmap.

Each frame:

1. Widgets call `Invalidate` → dirty boxes land in `paintengine2d.Damage`.
2. `Context` is created on the Device; `QuickReject` / clip skip clean regions.
3. Each component `Paint`s with `DrawRoundRect`, `Fill`, `Stroke`, gradients,
   and `DrawGlyphs` (shared white atlas, themed with `Paint.Color` tint).
4. Present is **eglSwapBuffers** on a GPU window, else X11 `XPutImage` /
   MIT-SHM or Wayland `wl_shm` (opt-in dmabuf). Offscreen present is a no-op.

```
Desktop app
    → uitoolkit (widgets, layout, focus, X11 / Wayland / offscreen)
        → paintengine2d.Context / Device / Damage / FontAtlas
            → GPUDevice (EGL/GLES2) or CPU scanline AA pixmap
```

Default chrome uses Titillium Web outlines (not the old 5×7 bitmap atlas).
Inspector tables and code previews use JetBrains Mono. **v0.7.2+ blit RGB
tint** is required. **v0.8.0** adds the GPU Device. One white atlas per family+weight+size is shared;
`DrawGlyphs` receives the theme Color. `style.GlyphTint` asserts the engine
actually multiplies RGB. `TestDefaultFontsRender` draws both families and
fails if the TTFs are missing or the ink is chunky 1-bit.

## Architecture

Inspired by JUCE `Component` + `LookAndFeel`, Evas damage, and Avalonia’s
retained tree (ideas only — no copied code).

```
platform   window + event pump + present          Linux X11 (EGL or XPutImage,
           (thin OS glue)                         CLIPBOARD+PRIMARY, XIM) and
                                                  Wayland (wl_egl_window or
                                                  wl_shm, xdg-shell, seat);
                                                  Win / macOS stubs
app        Application run loop, windows,         DPI/scale, backend select,
                                                  input routing
           capture / WritePNG
widget     retained Component: bounds, children,  HitTest, focus, Invalidate
           Paint(ctx *paintengine2d.Context)
layout     Measure / Arrange                      row, column, stack, flex
widgets    Button, Label, TextField, TextArea     ScrollView, ListView, TableView
           NumberField, Checkbox, Switch, Slider  MenuBar, TabView, TreeView
           Panel, Splitter, Overlay, Separator    StatusBar, ToolBar, ComboBox
           Accordion, Expander, Spacer            ProgressBar, RadioGroup
           MessageBox, FileDialog stub, Tooltip   TitleBar, context menus
style      LookAndFeel + Palette + Metrics        Dark / Light Classic
```

Swap the skin with `Application.SetLook(uitoolkit.LightLook())`. Controls
never hard-code colors.

## Examples

| Command | What it proves |
| --- | --- |
| `go run ./examples/gallery` | Stock controls, table, textarea, switch, accordion, spinner, tooltip, file stub, toolbar, combo, radio, progress, menus, tabs, tree, themes, scroll, list, message box |
| `go run ./examples/notes` | A small real app: sortable table, textarea body, priority spinner, file stub, tooltips |
| `go run ./examples/inspector` | Preferences inspector: table (JetBrains Mono), toolbar, tabs, message box |
| `go run ./examples/files` | Files / Projects dogfood: tree, table, toolbar, menus, TextArea preview, dialogs |

```bash
go run ./examples/gallery -screenshot docs/screenshots
```

## Tests

```bash
CGO_ENABLED=0 go test ./...
```

Coverage includes flex Measure/Arrange (parent-local coords), hit-test
z-order, focus tab order (including MenuBar / TabBar / TreeView),
checkbox/slider/text/button/switch interaction, virtual list range, scroll-wheel
bubbling, scrollbar track hits, text selection and copy/paste (in-process, plus OS clipboard on X11), menu
and tab swap, tree expand/select, context-menu dispatch, toolbar and
combo, radio groups, progress clamp, message-box results, table sort,
number-field step/filter, file-dialog stub, delayed tooltips, Esc
dismiss order, textarea newline/wrap/nav, switch toggle (including
disabled), accordion exclusive expand and focus yield, expander
relayout, separator and spacer measure, IME preedit/commit on text
widgets, and an offscreen paint that produces real pixels.
`go test ./examples/gallery` regenerates the nineteen PNGs and fails if
any two share a blob.

Keyboard map: [docs/keyboard.md](docs/keyboard.md). **Esc** closes
tooltip → popup → overlay, everywhere.

## Positioning

**uitoolkit** is a desktop widget kit on **your own Go paint engine**:
pure Go, retained tree, paintengine2d pixels, Linux X11 and Wayland. It is not
Fyne (GL + batteries), not Gio (ops + GPU), not Wails (Go + webview).

| | Paint | Windowing | Model | CGO |
| --- | --- | --- | --- | --- |
| **uitoolkit** | paintengine2d (CPU AA + Linux EGL/GLES2) | X11 + Wayland + offscreen | Retained, themed | Optional (X11/Wayland/EGL) |
| [Fyne](https://fyne.io) | Own + OpenGL | Cross-platform | Retained | Yes (GL) |
| [Gio](https://gioui.org) | Own ops renderer | Cross-platform | Immediate | Optional |
| [Wails](https://wails.io) | Browser / WebView | Cross-platform | HTML/CSS + Go | Yes (webview) |

Choose uitoolkit when you want **pure Go pixels you own**, a retained tree
with damage, and no browser runtime. Linux windowing is X11 or Wayland;
Win32 and AppKit are still stubs. Choose Fyne or Gio for mature
cross-platform backends today; choose Wails when the UI should be a webview.

## Out of scope

Documented on purpose — do not expect these yet:

- Full accessibility (AT-SPI / VoiceOver)
- IME candidate-window theming (ibus / fcitx / compositor draw their own)
- Mobile and webview
- Win32 and AppKit backends (interfaces + stubs only)
- HarfBuzz / complex shaping (NullShaper + OpenType outlines)

See [docs/platform.md](docs/platform.md) for X11 vs Wayland vs offscreen.
Linux desktop clipboard, IME preedit, and HiDPI are implemented on both
X11 and Wayland as of **v0.3.0**. GPU present (`UITK_PAINT=auto`) is **v0.5.0**.

## Version

**0.5.0** — Consumes paintengine2d **v0.8.0**. Default `UITK_PAINT=auto`
tries `GPUDevice` (Linux EGL/GLES2): Wayland `wl_egl_window` +
`eglSwapBuffers`, X11 EGL window + `eglSwapBuffers`. If EGL init fails,
present is the v0.4.1 CPU path (opaque `wl_shm` / `XPutImage`).
`UITK_PAINT=cpu` forces CPU; `gpu` prefers EGL.

**0.4.1** — Wayland present is opaque again: default `auto` is `wl_shm`
`XRGB8888` + `set_opaque_region` (damage-only uint32 RGBA→BGRA, forced
alpha, four present slots, no explicit-sync wait). `UITK_WAYLAND_PRESENT=dmabuf`
is opt-in and falls back to shm on a blank upload. X11 LE TrueColor
uses the same fast upload (no per-pixel `NRGBAAt`). Workaround for a
transparent window: `UITK_WAYLAND_PRESENT=shm`. Still paintengine2d
**v0.7.2**.

**0.4.0** — Default UI is Titillium Web; mono is JetBrains Mono (OFL,
embedded, rasterized at runtime). Files / Projects sample app. Wayland
dmabuf **explicit sync** (`zwp_linux_explicit_synchronization_v1` and
`wp_linux_drm_syncobj_v1` timeline when a DRM fd is available), still
falling back to implicit + `wl_shm`. Still paintengine2d **v0.7.2**.

**0.3.1** — Wayland present prefers `zwp_linux_dmabuf_v1` (feedback +
ARGB8888/XRGB8888, GBM or memfd/dma-heap/udmabuf) and falls back to
`wl_shm`. `UITK_WAYLAND_PRESENT=shm|dmabuf|auto`. Still paintengine2d
**v0.7.2** (`Image.Pix` / `RowStride` export).

**0.3.0** — Production Linux windowing: X11 INCR clipboard, XIM preedit,
RandR/Xft HiDPI, EWMH fullscreen/maximize, MIT-SHM present; Wayland
`wl_data_device` + primary, `text-input-v3` IME, output / fractional
scale, xdg-shell states and SSD. Still paintengine2d **v0.7.2**.

**0.2.0** — Wayland `wl_shm` + xdg-shell toplevel, seat pointer/keyboard
(xkbcommon), auto-select `WAYLAND_DISPLAY` then `DISPLAY` then offscreen.
X11 harden from 0.1.8 kept. Still paintengine2d **v0.7.2**.

**0.1.8** — Harden X11: shared display and multi-window destroy, OS
CLIPBOARD + PRIMARY (TextField / TextArea Ctrl+C/X/V and middle-click),
Xft.dpi / env scale into LookAndFeel metrics, XIM compose/dead keys,
stride- and mask-correct `XPutImage`. Still paintengine2d **v0.7.2**.

**0.1.7** — Harden 0.1.6: Accordion exclusive + layout/focus, TextArea
newline and Switch toggle regressions, gallery metrics polish, comparison
vs Fyne / Gio / Wails. Still paintengine2d **v0.7.2** (`02b2939`).

**0.1.6** — TextArea (wrap or scroll, multi-line caret), Switch, Accordion /
Expander, first-class Separator and Spacer, Inspector sample (table +
toolbar + tabs + message box). Keyboard map covers the new controls.

## License

MIT — see [LICENSE](LICENSE).
