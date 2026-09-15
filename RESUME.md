# Resume here — end-to-end fixes and theme engines, 2026-09-15

Everything is on feature branches, pushed to GitHub. `dev` is untouched in both
repos.

| repo | branch | worktree |
| --- | --- | --- |
| uitoolkit | `feat/theme-engines` (includes `feat/e2e-era-themes`) | `~/go/src/github.com/codemodify/uitoolkit-core` |
| paintengine2d | `feat/e2e-era-themes` | `~/go/src/github.com/codemodify/paintengine2d` |

The two branches go together: uitoolkit uses paintengine2d's new `PathCache`
(through the relative `replace ../paintengine2d` in `go.mod`), so check out both
before building. `~/go/src/github.com/codemodify/uitoolkit` itself sits on
`feat/e2e-era-themes`; the theme work is in the `uitoolkit-core` worktree.

Before merging to `dev`: commit `6194388` accidentally added a 9 MB
`uitk-themesheet` binary (removed again in `306e12c`). Squash-merge, or filter
it out of the history.

## Decisions waiting for you

1. **The default theme.** The stock looks are "Classic 95 Dark / Light" (a
   Win95 homage). Breeze, Adwaita, Fusion and Fluent all have real engines
   now, so any of them could be the default instead. Say which.
2. **Follow the desktop by default?** Apps can follow the desktop's light or
   dark mode and accent colour (Settings, look.json `followDesktop`). It is
   off unless switched on. GTK 4, libadwaita and Qt 6 apps follow by default.
3. **The Java looks are settled.** Metal (Steel, Ocean) and Nimbus were
   rewritten clean-room from published facts and are merged. The old
   OpenJDK-derived branch `engine/metal` (local only, worktree
   `uitoolkit-eng-metal`, never pushed) can be deleted:
   `git worktree remove ../uitoolkit-eng-metal && git branch -D engine/metal`.

## Real-hardware checklist (needs your laptop)

Everything below passed headless. Each item needs a real compositor and
GPU, through the nested-KWin rig (`tools/e2e/theme-tour.sh`), never your own
session:
1. **The original glitches.** Click around the gallery and Mail in several
   themes: dialogs, menus, theme switches, hover.
2. **GPU paths.** Fluent's acrylic menus and Aero's glass (backdrop blur),
   the fades (layers, cross-fades), and Plastik's dithered groove
   (patterns).
3. **Fractional scale 1.75** (yours). Check the HiDPI fixes in the base
   look and widgets. Two live-compositor tests failed on your session at
   this scale:
   - `TestWaylandSurfacePresent` expects a 160px buffer and got 280;
   - `TestWaylandPresentOpaqueColor` reports "no present slot".
4. **Desktop following.** Switch Plasma between light and dark and change
   its accent with a following app open: it should restyle live. Also
   Plasma's animation speed at Instant, which is reduced motion.
5. **The tray.** Mail's tray menu should respond at once. The loop now
   sleeps up to 1s between safety wakes instead of 100ms; events wake it.
6. **Mnemonic underlines.** They should show only while Alt is held in the
   XP, Plasma and Windows 10 looks.
7. **Touchpad scrolling.** Slow two-finger scrolls should follow the
   fingers; they used to jump by whole lines (a bug found and fixed
   today). A quick swipe should glide on and slow down (new kinetic
   scrolling). A mouse wheel should scroll three lines a notch.
8. **Drag and drop.** Drag files from Dolphin into Mail's compose window,
   which should attach them, and text into a field. This is Wayland only for
   now.
9. **Native file dialogs.** Settings, "Use the desktop's file dialogs",
   then Attach in Mail's compose window: Plasma's own dialog should open.
10. **Orca**, if you install it: `UITK_A11Y=1` is not needed, since the
   bridge starts when Orca does.

## What the glitching was

Found on real hardware with the nested-KWin rig in `tools/e2e/`; the findings
are in `docs/e2e/2026-09-14/`.

- **The main cause:** paintengine2d's EGL buffer-age ring recorded full-frame
  presents as "no damage", so partial presents showed two-frame-old buffers.
  Closed dialogs came back, menus flickered, theme switches reverted.
- **Wayland:**
  - frames painted while a frame callback was pending were dropped;
  - a focused text field spun an IME repaint loop (about 3,500 repaints per
    second);
  - there were no pointer-leave events.
- **Widgets, menus and focus:** about 50 more findings, including hover
  leaking into toolbars, tab bars and table headers; menu keyboard
  navigation and accelerators; modality; focus rings clipped away; contrast;
  the paperclip glyph; and Mail multi-selection.

All of these are fixed, with tests. **Not yet re-verified on real hardware**:
that needs your laptop and the rig, and is the first thing to do next.

## Theme engines

**See them all:** the Theme Atlas, https://claude.ai/artifact/FDfGxoNiUnPuyU9TMEy9aZ
(private until you share it), shows every pack's live preview and full gallery.

A theme is now an **engine** (Go code that decides the shapes, like Qt's
`QStyle`) plus **packs** (colours, metrics and parameters). The guide is
`docs/theme-engines.md`; `go run ./cmd/uitk-themesheet -list` lists every
pack.

Merged engines:

| engine | packs |
| --- | --- |
| win95 | Windows 95, 98, 2000, Hot Dog Stand, High Contrast, Windows 95 Dark |
| luna | XP Luna Blue, Olive, Silver, Royale, Royale Noir |
| aqua | Aqua, Graphite, Graphite Night, Brushed Metal |
| platinum | Mac OS 8 Platinum, Platinum Lime |
| motif | Motif, CDE (six schemes), IRIX Indigo Magic, HP VUE |
| next | NeXTSTEP, OPENSTEP, NeXTSTEP Night, Window Maker (four themes) |
| fusion | Qt Fusion, Fusion Dark |
| oxygen | KDE 4 Oxygen |
| breeze | KDE Plasma Breeze, Breeze Dark |
| clearlooks, bluecurve | GNOME 2 Clearlooks, Human, Qt Cleanlooks; Red Hat Bluecurve |
| adwaita | GNOME Adwaita, Adwaita Dark, Adwaita (GTK 3) |
| aero, metro, fluent | Windows Vista/7 Aero and 7 Basic; Windows 8, 10, 10 Dark; Windows 11 Fluent and Fluent Dark |
| keramik, plastik | KDE 3 Keramik, Plastik, Qt 4 Plastique |
| system7, win31, openlook, amiga, beos, os2 | System 1 and 7, Windows 3.1 and Hot Dog Stand, OPEN LOOK, Workbench 1.3 and 3.1, BeOS, OS/2 Warp 4 |
| macos, material, flatlaf | OS X Yosemite, macOS Big Sur and Dark; Material 2 and 3, light and dark; FlatLaf Light, Dark, Darcula |
| metal, nimbus | Swing's Metal (Steel 1998, Ocean 2004) and Nimbus (2008), clean-room |

Also merged: `macos` (OS X Yosemite, macOS Big Sur and Big Sur Dark),
`material` (Material and Material Dark, Material 3 and Material 3 Dark,
whose tonal palettes are computed from a seed colour) and `flatlaf` (FlatLaf
Light, FlatLaf Dark, Darcula).

78 packs from 28 engines in all. Running now in its own worktree: sidebar
styles and toolbar tool buttons for macOS, Adwaita, Fluent, Material and
Metal (`engine/sidebar`).

Core features the engines drive, added along the way:
- selected tabs overlap their neighbours;
- menu, tooltip and dialog drop shadows;
- list, tree and table view frames;
- per-axis scroll arrows;
- window backgrounds and tab panes;
- group boxes and in-app dialog windows;
- item states: selected, current, unfocused, and backdrop when the window is
  inactive, each engine following its platform's rules;
- **whole-pixel layout:** component bounds and table column edges are
  rounded to device pixels (WPF / Avalonia layout rounding). This removed the
  seam every engine showed at column edges in a selected table row, and makes
  every engine's 1px lines land on real pixels;
- **every theme reads in its era's typeface** when installed (fontconfig):
  Tahoma for XP, Segoe UI for Vista to 11, Lucida Grande for Aqua, Helvetica
  (or Nimbus Sans) for NeXT and Motif, Cantarell for GNOME, Noto Sans for
  Plasma, Roboto for Material, then open look-alikes; the bundled Titillium
  Web is the last resort. `theme.json` can list its own `fonts`;
  `UITK_SYSTEM_FONTS=0` keeps the bundled faces;
- **transient overlay scroll bars** (libadwaita, Fluent): the content keeps
  its full width and the bar shows while scrolling or hovered, then fades;
- **hover and focus cross-fade** where the platform animated (Aero and Adwaita
  200ms, Windows 10, Breeze and Oxygen 150ms, Fluent 83ms); presses stay
  instant; `UITK_ANIMATIONS=0` turns fades off;
- **spin boxes** put their buttons where the platform did: inside the field's
  frame (Windows, KDE, GNOME side by side "− +"), beside it (Mac OS, Motif);
- engine hooks from the Windows engines' wish list: table cells know their
  place in the row (one rounded selection box across a row), tabs overlap by a
  border so neighbours share one line, tree expanders light up under the
  pointer, packs can pin the colour of selected text (Windows' white on blue);
- message boxes read "Yes No Cancel" under Windows and KDE, and a pack's look
  keeps the pack's own corners;
- **materials:** backdrop blur on CPU and GPU (Fluent's acrylic menus, Aero's
  blurred glass frames), group opacity and true cross-fades, tiled image
  patterns (Plastik's dithered groove);
- **text is kerned** (the fonts' GPOS / kern pairs), with carets, hit-testing
  and eliding on the kerned layout;
- **tree branch lines** end where their branches do (last-child elbows);
  KDE 3's three-arrow scroll bars; fixed-size thumbs (Mac, Windows 3.1,
  OPEN LOOK); menus take each look's row height;
- **`ButtonBox`** puts dialog buttons in the platform's order (Qt's
  QDialogButtonBox), live with the theme; Mail and the file dialog use it;
- busy bars animate themselves; the default button throbs in Aqua and
  breathes in Aero; a saved "reduce motion" preference and a Settings switch
  turn every animation off;
- labels honour newlines, and paddings scale with the display.

## Toolkit features added

- **Item views:**
  - multi-selection: Ctrl, Shift, Shift+arrows, Ctrl+A and right-click rules,
    in lists, tables and card lists;
  - type-ahead find;
  - Menu key / Shift+F10 context menus;
  - `EnsureVisible`;
  - a focus mark on the current row.
- **Mail:** its message list multi-selects in both the table and card views,
  so bulk actions act on every selected message.
- **New widgets:**
  - `Grid` and `Form` layouts (QGridLayout, QFormLayout; form labels are
    right-aligned in Mac looks);
  - `Calendar` and `DateField`;
  - `ColorButton` with a drop-down picker;
  - `Segmented`, a view switch of joined toggle buttons;
  - `Picture` for PNG, JPEG and GIF images.

  The gallery's Form tab shows them.
- **Windows know when they're inactive:** captions, selections and Aqua's
  default button follow it; odd list rows can be striped (Aqua).
- **Settings:** restructured as a theme browser with a live, themed preview
  application.
- **Follows the desktop's light or dark mode:** Settings → "Match the
  desktop's light or dark mode" (look.json `followDesktop`). Apps read the
  XDG portal's `color-scheme`, as GTK 4 and Qt 6 apps do, and show the
  saved theme's sibling: Breeze and Breeze Dark, Luna Olive and Royale
  Noir, CDE palettes and Charcoal. They switch live when the desktop does.
  Themes with no sibling stay as chosen. `UITK_COLOR_SCHEME=dark` stands in
  for the desktop. See `docs/settings.md`.
- **Takes the desktop's accent colour** in themes whose engine recolours
  around one (`style.AccentEngine`): Breeze, Fluent, Windows 10, Adwaita
  (snapped to GNOME's nine accents, as libadwaita does), macOS Big Sur,
  Material 2 and 3 (Material You: the accent becomes the seed), FlatLaf,
  Fusion, Oxygen, and Aero's glass. Each derives its shades the way its
  platform did. `UITK_ACCENT=#e95420` stands in for the desktop's accent.
- **Honours the desktop's reduced-motion setting** (yours is on): fades,
  pulses and busy bars stop in every app, as in GTK 4 apps.
- **Mnemonic underlines** show when each platform showed them: always in
  Windows 95, only while Alt is held in XP and Plasma, never on the Mac
  (`HintMnemonics`).
- **Labels word-wrap** (`Label.Wrap`, QLabel's wordWrap).
- **Editable combo boxes** (`ComboBox.SetEditable`): type a value or pick
  one, with inline completion (QComboBox's editable).
- **Slider tick marks** (`Slider.Ticks`). Pressing the thumb takes hold of
  it where it was pressed; it used to jump a few pixels in most looks.
- **Progress text** (`ProgressBar.ShowText`): in the bar where it fits,
  beside thin bars.
- **HiDPI:** the base look (the default Classic 95 themes) and the widgets'
  default sizes and paddings now scale with the display. At 1.75x, which is
  your laptop's scale, labels crowded their controls and lines came out
  thin.
- **Sidebars** (`ListView.Sidebar`, `TreeView.Sidebar`) and a new
  `ViewBackground` engine hook: Aqua draws Leopard's source list; more
  engines are in progress.
- **Start-up:** the font index and the portal read run alongside display
  setup.
- **`UITK_THEME=<pack>`:** runs any app in any theme, like `GTK_THEME`.
- **`gallery -screenshot DIR -theme <pack>`:** takes every scripted gallery
  shot in one pack.

## Accessibility

uitoolkit had no accessibility layer, so screen readers could not see its
apps. Now:
- **The model** (`a11y`): every window is a tree of roles, names, states,
  values and actions. Every stock widget describes itself; views list their
  items, and actions (press, toggle, select, expand) reach the widgets.
- **The audit**: `a11y.Check` flags controls without a name, duplicate IDs
  and similar problems. Tests run it over every gallery page, Settings and
  Mail; it found unnamed controls in all three, now fixed.
- **The Linux bridge** (AT-SPI2): Orca and other assistive technology read
  and drive the apps. It stays off until a screen reader runs (your desktop
  says none does).
- **The smoke test**: `tools/a11y/smoke.sh` proves the bridge in a private
  D-Bus session with a real libatspi client. It checks roles, names, values
  and text, actions, and the focus and "checked" announcements.

It also announces changes on the focused object (a ticked check box, typed
text, caret moves) and supports EditableText, for automation tools such as
dogtail. See `docs/accessibility.md`. Still to do: relations, the Selection
and Table interfaces, and the Windows and macOS adapters.

## Gaps against Qt and GTK (my proposed order)

1. **Complex text:** bidirectional text (Arabic, Hebrew) and shaping for
   scripts that need it (Arabic joining, Indic), plus right-to-left
   layout mirroring. Text is kerned but not shaped. A pure-Go HarfBuzz port
   (go-text/typesetting, BSD) could do the shaping.
2. **Drag and drop, the rest:** X11's XDND, and dragging out of uitoolkit
   apps (drag sources). Drops onto windows work on Wayland.
3. **Other platforms:** Windows and macOS backends, with UI Automation and
   NSAccessibility adapters for the accessibility tree that now exists.
4. **Portals:** a parent window for native dialogs on Wayland
   (xdg-foreign), OpenURI, notifications through the portal.
5. **Widgets:**
   - a rich-text editor;
   - dock widgets (QDockWidget);
   - an MDI area;
   - a wizard;
   - kinetic scrolling and touchpad gestures.
6. **Printing.**

## Performance

`BenchmarkGalleryRepaint` and `BenchmarkGallerySmallRepaint` in
`examples/gallery` repaint the gallery in each engine.

- paintengine2d keeps recorded paths across frames and allocates draw ops in
  slabs.
- Its CPU fast path now takes gradient rects, and rect lists whose pixels don't
  touch.
- Textured looks batch their stripes.
- Open strokes with square caps no longer draw a stray band (their outline
  was left unclosed).
- `Context.SetAlpha` (global alpha) and `Context.DrawLayer` (group opacity)
  drive the fades; GPU gradients now honour `Paint.Opacity`, which they
  ignored before.

The result: a hover repaint on the stock look went from 321 to 152
allocations, and on Aqua from 4,376 to 356. The gallery peaks at about 35 MB
RSS.

## Checks that ran

`go build`, `go vet`, gofmt and `go test ./...` pass in both repos, as does
`go test -race ./...` for uitoolkit. Contract tests cover every pack:
- controls paint inside their bounds;
- shadows stay within their reach;
- keyboard focus is visible;
- text meets contrast checks.

Everything headless ran in this session; the GPU and Wayland paths still need
the real-hardware pass. A slip on 2026-09-15: one `go test ./platform` ran
without unsetting `WAYLAND_DISPLAY`, so its live-compositor tests opened
test windows on the real session for under a second. Two of them failed
there, and both belong in the real-hardware pass:
- `TestWaylandSurfacePresent` expects a 160px buffer and got 280 (the
  output's 1.75 scale);
- `TestWaylandPresentOpaqueColor` reported "no present slot".
