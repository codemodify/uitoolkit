# uitoolkit

**Pure-Go desktop UI toolkit.** Retained widget tree, layout, themes, and
an X11 window backend. Every pixel is painted with
[`github.com/codemodify/paintengine2d`](https://github.com/codemodify/paintengine2d)
(v0.7.2+). Labels share a white glyph atlas and theme through `Paint.Color`
tint. There is no second rasterizer, no Skia, no Gio renderer, and no Electron.

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
```

| | |
| --- | --- |
| Language | Go 1.22+ |
| Paint | paintengine2d 0.7.2 (`Context`, `Damage`, `DrawGlyphs` Color tint) |
| Windowing | Linux X11 first (CGO + libX11); offscreen always |
| CGO | optional — tests and screenshots are `CGO_ENABLED=0` |
| License | MIT |

## Screenshots

Real frames from the gallery and the Notes app, painted through paintengine2d
and written as PNG (no placeholders).

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

### Notes — a small desktop app

![Notes](docs/screenshots/notes.png)

Regenerate:

```bash
go run ./examples/gallery -screenshot docs/screenshots
```

## Quickstart

```bash
git clone https://github.com/codemodify/uitoolkit.git
cd uitoolkit
CGO_ENABLED=0 go test ./...
go run ./examples/gallery            # X11 when DISPLAY is set
go run ./examples/gallery -headless  # writes gallery.png
go run ./examples/notes
```

Headless / CI paints into `paintengine2d.NewImage` and can `Window.WritePNG`.
On Linux with `DISPLAY` and CGO, the same buffer is presented with XPutImage
and dirty rects from `paintengine2d.Damage`.

## How it uses paintengine2d

uitoolkit does not rasterize. A window owns a premul RGBA pixmap
(`NewImage` or, on X11, the same CPU buffer then copied to the X image).
Each frame:

1. Widgets call `Invalidate` → dirty boxes land in `paintengine2d.Damage`.
2. `Context` is created on the pixmap; `QuickReject` / clip skip clean regions.
3. Each component `Paint`s with `DrawRoundRect`, `Fill`, `Stroke`, gradients,
   and `DrawGlyphs` (shared white atlas, themed with `Paint.Color` tint).
4. `Damage.Rects` are presented to X11. Offscreen present is a no-op.

```
Desktop app
    → uitoolkit (widgets, layout, focus, X11/offscreen)
        → paintengine2d.Context / Device / Damage / FontAtlas
            → CPU scanline AA pixmap
```

If the engine is missing a primitive (nine-patch, SaveLayer, OpenType), the
fix belongs in paintengine2d — not a second painter here. **v0.7.2 blit RGB
tint** is required. One white atlas per size is shared; `DrawGlyphs` receives
the theme Color. `style.GlyphTint` asserts the engine actually multiplies RGB.

## Architecture

Inspired by JUCE `Component` + `LookAndFeel`, Evas damage, and Avalonia’s
retained tree (ideas only — no copied code).

```
platform   window + event pump + present          Linux X11; stubs for
           (thin OS glue)                         Wayland / Win / macOS
app        Application run loop, windows,         DPI/scale, input routing
           capture / WritePNG
widget     retained Component: bounds, children,  HitTest, focus, Invalidate
           Paint(ctx *paintengine2d.Context)
layout     Measure / Arrange                      row, column, stack, flex
widgets    Button, Label, TextField, Checkbox,    ScrollView, ListView
           Slider, Panel, Splitter, Overlay       (virtualized rows)
style      LookAndFeel + Palette + Metrics        Dark / Light Classic
```

Swap the skin with `Application.SetLook(uitoolkit.LightLook())`. Controls
never hard-code colors.

## Examples

| Command | What it proves |
| --- | --- |
| `go run ./examples/gallery` | Every stock control, themes, scroll, list, dialog overlay, second window |
| `go run ./examples/notes` | A small real app: filterable list, editor, add/delete |

```bash
go run ./examples/gallery -screenshot docs/screenshots
```

## Tests

```bash
CGO_ENABLED=0 go test ./...
```

Coverage includes flex Measure/Arrange (parent-local coords), hit-test
z-order, focus tab order, checkbox/slider/text/button interaction, virtual
list range, scroll-wheel bubbling, scrollbar track hits, text selection,
and an offscreen paint that produces real pixels. `go test ./examples/gallery`
regenerates the six PNGs and fails if any two share a blob.

## Positioning

**uitoolkit** is a desktop widget kit on **your own Go paint engine**. It is
not Fyne (GL + batteries), not Gio (ops + GPU), not a webview.

| | Paint | Windowing | Widgets | CGO |
| --- | --- | --- | --- | --- |
| **uitoolkit** | paintengine2d (own CPU AA) | X11 + offscreen | Retained, themed | Optional (X11) |
| [Fyne](https://fyne.io) | Own + OpenGL | Cross-platform | Yes | Yes (GL) |
| [Gio](https://gioui.org) | Own ops renderer | Cross-platform | Immediate + widgets | Optional |
| Electron | Browser (Skia) | Chromium | HTML/CSS | Native binary |

Choose uitoolkit when you want **pure Go pixels you own**, a retained tree
with damage, and no browser runtime. Choose Fyne/Gio when you need mature
cross-platform backends today.

## v0.1 out of scope

Documented on purpose — do not expect these yet:

- Full accessibility (AT-SPI / VoiceOver)
- Production IME / CJK composition
- Mobile and webview
- Wayland, Win32, and AppKit backends (interfaces + stubs only)
- OpenType / HarfBuzz (engine text hook only)
- System clipboard

## Version

**0.1.2** — paintengine2d **0.7.2** (`02b2939`): shared white glyph atlas +
`Paint.Color` tint. Keeps 0.1.1 screenshot and input hardening.

## License

MIT — see [LICENSE](LICENSE).
