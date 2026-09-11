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
go get github.com/codemodify/paintengine2d@v0.7.2
```

| | |
| --- | --- |
| Language | Go 1.22+ |
| Paint | paintengine2d **v0.7.2** (`02b2939`; `Context`, `Damage`, `DrawGlyphs` Color tint) |
| Windowing | Linux X11 (CGO + libX11); offscreen always; Wayland stub |
| CGO | optional — tests and screenshots are `CGO_ENABLED=0` |
| License | MIT |

## Screenshots

Real frames from the gallery, Notes, and Inspector, painted through
paintengine2d and written as PNG (no placeholders).

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
UITK_BACKEND=x11 go run ./examples/gallery
go run ./examples/gallery -headless  # writes gallery.png
go run ./examples/notes
go run ./examples/inspector
```

Headless / CI paints into `paintengine2d.NewImage` and can `Window.WritePNG`.
On Linux with `DISPLAY` and CGO, the same buffer is presented with XPutImage
and dirty rects from `paintengine2d.Damage`. Scale comes from `UITK_SCALE` /
`GDK_SCALE` / `QT_SCALE_FACTOR`, else Xft.dpi (or screen mm) so LookAndFeel
metrics grow on HiDPI. Ctrl+C/X/V and middle-click use the OS CLIPBOARD and
PRIMARY selections.

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
    → uitoolkit (widgets, layout, focus, X11 / offscreen)
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
platform   window + event pump + present          Linux X11 (shared display,
           (thin OS glue)                         CLIPBOARD+PRIMARY, XIM);
                                                  Wayland / Win / macOS stubs
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
| `go run ./examples/inspector` | Preferences inspector: table, toolbar, tabs, message box, switch, accordion |

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
relayout, separator and spacer measure, and an offscreen paint that
produces real pixels.
`go test ./examples/gallery` regenerates the seventeen PNGs and fails if
any two share a blob.

Keyboard map: [docs/keyboard.md](docs/keyboard.md). **Esc** closes
tooltip → popup → overlay, everywhere.

## Positioning

**uitoolkit** is a desktop widget kit on **your own Go paint engine**:
pure Go, retained tree, paintengine2d pixels, X11-first. It is not
Fyne (GL + batteries), not Gio (ops + GPU), not Wails (Go + webview).

| | Paint | Windowing | Model | CGO |
| --- | --- | --- | --- | --- |
| **uitoolkit** | paintengine2d (own CPU AA) | X11 + offscreen | Retained, themed | Optional (X11) |
| [Fyne](https://fyne.io) | Own + OpenGL | Cross-platform | Retained | Yes (GL) |
| [Gio](https://gioui.org) | Own ops renderer | Cross-platform | Immediate | Optional |
| [Wails](https://wails.io) | Browser / WebView | Cross-platform | HTML/CSS + Go | Yes (webview) |

Choose uitoolkit when you want **pure Go pixels you own**, a retained tree
with damage, and no browser runtime. Choose Fyne or Gio for mature
cross-platform backends today; choose Wails when the UI should be a webview.

## v0.1 out of scope

Documented on purpose — do not expect these yet:

- Full accessibility (AT-SPI / VoiceOver)
- Production CJK IME (preedit / candidate UI). X11 uses XIM
  (`XIMPreeditNothing`) so compose and dead keys work; filtered keys
  and UTF-8 `Xutf8LookupString` feed `EventText`.
- Incremental (INCR) X11 clipboard for huge pastes
- Mobile and webview
- Wayland, Win32, and AppKit backends (interfaces + stubs only)
- OpenType / HarfBuzz (engine text hook only)

See [docs/platform.md](docs/platform.md) for X11 vs offscreen and IME gaps.

## Version

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
