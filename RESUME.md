# Resume here — end-to-end fixes and theme engines, 2026-09-15

**Merged to `dev` and pushed on 2026-09-15**, in both repos, together with
the 2026-09-13 review merge that had been waiting on local `dev`:

| repo | merge on `dev` | worktree |
| --- | --- | --- |
| uitoolkit | `c3083df` (`feat/theme-engines`) | `~/go/src/github.com/codemodify/uitoolkit-core` |
| paintengine2d | `4ed8330` (`feat/e2e-era-themes`) | `~/go/src/github.com/codemodify/paintengine2d` |

uitoolkit uses paintengine2d's new APIs (`PathCache`, `BackdropBlur`,
`DrawCrossFade`, `ImagePattern`) through the relative
`replace ../paintengine2d` in `go.mod`, so keep both checked out side by side
until paintengine2d v0.11.0 is tagged.

Both repos are now under **The Free License** (the license of
simple-http-fileserver), replacing MIT.

Before the merge, the feature branch's history was cleaned of stray
binaries: a 25 MB `gallery` build that was still tracked, and the 9 MB
`uitk-themesheet` of `6194388`. Every commit and merge was kept; only those
files are gone. The branch as it was is kept locally as
`backup/theme-engines-pre-clean`.

## Later on 2026-09-15: today's themes and app-drawn title bars

Also merged to `dev` and pushed:

| merge on `dev` | branch | what |
| --- | --- | --- |
| `468d557` | `feat/csd` | windows that draw their own title bar and borders (phase 1) |
| `04d0746` | `engine/web` | the `web` engine: 25 packs of today's web and app looks |
| `ff2cc6b` | `docs/themes-web` | the Settings preview no longer jumps; README sheets and timeline |
| `6dbe277` | `feat/titlebar` | tabs in the title bar, and a title bar in every look (phase 2) |
| `476dc19` | `engine/modern` | 18 more packs of today's looks; 121 in all |

- **Today's themes.** The 29th engine, `web`, draws the looks developers
  use now: GitHub's Primer (light, dark, dark dimmed), Vercel's Geist,
  shadcn/ui, Linear, **SourceGit** (light and dark, from its own theme),
  Dracula and Alucard, Nord, Tokyo Night (night, storm, day), Catppuccin
  (four flavours) and Rosé Pine (three). On the GPU in the rig every one
  matches the CPU render within 46 pixels (0.01%).
- **Title bars like SourceGit's and Chromium's.** A window can draw its own
  title bar: `widgets.HeaderBar` with the caption buttons where the desktop
  puts them, while moves, resizes, snapping and the window menu stay the
  compositor's. Mail's tool-bar row is now its title bar. Settings has "Use
  system title bar and borders"; `UITK_DECORATIONS` and look.json's
  `"decorations"` choose too. GNOME, which has no server frames, now gets a
  frame. How it works: `docs/decorations.md`.
- **Tabs in the title bar, and a title bar in every look** (phase 2).
  `widgets.BrowserTabs` is a Chromium-style document-tab strip that sits in
  a header bar (Files has folder tabs); `style.DecorationEngine` lets an
  engine paint its era's frame, and every other pack gets its in-app window
  caption adapted, so all 121 have one: 70 native, 31 adapted, 2 plain.
  Settings: "Place window buttons as the theme does" (look.json
  `captionButtons`).
- **18 more packs of today's looks** (`engine/modern`): macOS Tahoe's Liquid
  Glass, Plasma 6's Breeze, GNOME 48's Adwaita, Material 3 Expressive, VS
  Code's 2026 themes, JetBrains Islands, and the Gruvbox, Solarized and Atom
  One palettes. Platform engines can now hand a later era to an engine of
  their own (`style.EraEngine`), and tool bars can share group chrome
  (`style.ToolGroupEngine`, Tahoe's glass capsules). 121 packs, 29 engines.
- **Frames with shadows and rounded corners** (phase 3). A look with a
  shadow puts its window inside a larger surface: the margin around it
  holds the shadow and the resize handles, while the desktop is told where
  the window really is (`set_window_geometry`, `_GTK_FRAME_EXTENTS`, input
  and opaque regions). Alpha buffers only where a frame needs them; on X11
  without a compositing manager the frame goes solid by itself. Each era
  gets the shadow and corners its windows had. paintengine2d gained
  `BlendDestOut` for the corners (its `dev`, `f852dbc`).
- **Next:**
  - title-bar phase 4: tab tear-off, KWin's server-decoration palette,
    `_NET_WM_SYNC_REQUEST`;
  - dockable panels;
  - the 31 adapted looks could get frames of their own (NeXT, Platinum,
    System 7, Amiga, BeOS, OS/2, Windows 3.1, KDE 3 and GNOME 2).
- **Off the list** (the user's call, 2026-09-15): releases and tags; IME and
  right-to-left text; Windows and macOS backends (pinned until Linux is
  polished); printing.

## Decisions (answered 2026-09-15)

1. **The default theme is Metal (Ocean)** (`metal-ocean`, Swing's own
   default since Java 5), replacing Classic 95 Dark. Apps show it until the
   user picks a theme; `dark` and `light` remain the Classic 95 packs.
2. **Apps do not follow the desktop by default.** Following its light or
   dark mode and accent colour stays a Settings switch (look.json
   `followDesktop`), off unless switched on.
3. **The OpenJDK-derived `engine/metal` branch is deleted.** Metal (Steel,
   Ocean) and Nimbus are the clean-room engines, from published facts.
4. **The real-hardware pass may use the laptop**, through the nested-KWin
   rig only.

## Real-hardware pass (2026-09-15, nested KWin on your GPU)

Run through the rig in `tools/e2e/` (nested KWin on the real GPU, never your
session). Verified there:
1. **The original glitches: gone.** The gallery in ten themes (Metal Ocean,
   Fluent, Aero, Luna, Breeze, Adwaita, Big Sur, Material 3, Windows 95,
   Aqua) went through hover, a menu, the About dialog, a tab, a combo popup
   and a theme switch. Every frame matched a full-frame repaint.
2. **GPU paths: match the CPU.** All 78 themes' galleries on the GPU are
   within 0.03% of the CPU render. Fluent's acrylic and Big Sur's vibrant
   menus, and Aero's glass dialogs, render.
3. **Fractional scale 1.75: correct.** At a real 1.75 output apps are asked
   for 1.75 (`preferred_scale` 210) and draw 1750×1330 buffers for a
   1000×760 window, crisp. The two tests that failed on your session were
   wrong about scale and timing, and are fixed.
6. **Alt underlines: fixed.** They never showed on a live compositor (Alt
   only re-presented the cached frame); now they show while Alt is held.

Also found and fixed there: a crash ("concurrent map writes") when a window
opened while the app was still serving a clipboard it owned after its last
window closed.

Still to try on your own desktop, since they need its services:
4. **Desktop following.** Switch Plasma between light and dark, or change
   its accent, with "Match the desktop" on in Settings.
5. **The tray.** Mail's tray menu should respond at once.
7. **Touchpad scrolling.** Slow two-finger scrolls follow the fingers; a
   quick swipe glides and slows down; a wheel notch scrolls three lines.
8. **Drag and drop.** Files from Dolphin into Mail's compose window.
9. **Native file dialogs.** Settings, "Use the desktop's file dialogs",
   then Attach in Mail's compose window.
10. **Orca**, if you install it: the bridge starts when Orca does.

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

All of these are fixed, with tests, and re-verified on your GPU in the
nested-KWin rig (see the real-hardware pass above).

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
| web | GitHub Primer, Vercel Geist, shadcn/ui, Linear, SourceGit, Dracula, Nord, Tokyo Night, Catppuccin, Rosé Pine (25 packs) |

Also merged: `macos` (OS X Yosemite, macOS Big Sur and Big Sur Dark),
`material` (Material and Material Dark, Material 3 and Material 3 Dark,
whose tonal palettes are computed from a seed colour) and `flatlaf` (FlatLaf
Light, FlatLaf Dark, Darcula).

121 packs from 29 engines in all, with `web` and the later eras (above). Every agent branch is merged, the last
being sidebar styles and tool-bar buttons for macOS, Adwaita, Fluent and
Material (`engine/sidebar`).

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
  `ViewBackground` engine hook. Styled in Aqua (Leopard's source list),
  macOS Yosemite and Big Sur, libadwaita, Fluent's navigation pane and
  Material's drawers.
- **The desktop's own file dialogs** through the XDG portal (Settings:
  "Use the desktop's file dialogs"), modal to the window on X11.
- **Drops from other apps** (Wayland): files and text, `DropZone`; Mail's
  compose window attaches dropped files.
- **Touchpad scrolling** follows the fingers (a bug made slow scrolls jump
  by lines), with kinetic flings; wheels scroll three lines a notch.
- **Medium and semibold weights** for looks that need them (Material,
  Fluent, macOS).
- **Idle cost:** look.json is watched with inotify, not polled every
  300ms, and tray apps sleep up to 1s instead of 100ms.
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
text, caret moves), and supports EditableText (for automation tools such as
dogtail), Selection, and Table with TableCell (Orca's row and column
navigation). Label.For names a control after its caption. The audit also
covers Files, Notes and Inspector, and flags controls the keyboard cannot
reach. See `docs/accessibility.md`. Still to do: relations and the Windows
and macOS adapters.

## Gaps against Qt and GTK (my proposed order)

1. **Complex text:** bidirectional text (Arabic, Hebrew) and shaping for
   scripts that need it (Arabic joining, Indic), plus right-to-left
   layout mirroring. Text is kerned but not shaped. A pure-Go HarfBuzz port
   (go-text/typesetting, BSD) could do the shaping.
2. **Drag and drop, what is left:** both halves work on both backends —
   XDND 5 on X11, `wl_data_device` / `wl_data_source` on Wayland, with a
   drag icon, action negotiation and in-process payloads. Still missing:
   a drop indicator between rows for reordering a list, and XDND's "ask"
   action (the Copy / Move / Link menu a file manager shows on a drop).
3. **Other platforms:** Windows and macOS backends, with UI Automation and
   NSAccessibility adapters for the accessibility tree that now exists.
4. **Portals:** a parent window for native dialogs on Wayland
   (xdg-foreign), OpenURI, notifications through the portal.
5. **Widgets:**
   - a rich-text editor;
   - dock widgets (QDockWidget);
   - an MDI area;
   - a wizard;
   - touchpad gestures (pinch, swipe), and kinetic scrolling on X11
     (Wayland has it).
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
allocations, and on Aqua from 4,376 to 356.

### Memory (branch `feat/memory`, both repos)

Measured on the GPU in the nested KWin at a real 1.75 scale, 10 s after
launch; "own" is private dirty memory, what the app itself costs. The rest
of resident memory is the GPU driver's shared libraries (Mesa, LLVM: about
38 MB, shared by every GL app) and the binary's clean pages.

| app | before: resident / own | after: resident / own |
| --- | --- | --- |
| Mail | 140 / 80 MB | 91 / 36 MB |
| gallery | 103 / 65 MB | 83 / 30 MB |
| Settings | 100 / 46 MB | 82 / 31 MB |
| Files | 98 / 45 MB | 82 / 31 MB |
| Notes | 97 / 44 MB | 78 / 28 MB |
| Inspector | 91 / 37 MB | 79 / 28 MB |

- **Glyph sheets start small** (128² or 256²) and double as they fill; the
  512² and 768² first sheets were mostly empty, 47 MB of Mail's 53 MB live
  heap.
- **Idle trim:** two seconds after a burst of allocation (start-up, a
  window, a theme, new content) with no events, the free heap goes back to
  the OS once. Go otherwise keeps up to twice the live heap.
- **No CPU copies beside the GPU:** the Wayland window's device-size pixmap
  and the GPU device's readback image are made only when the CPU path or a
  screenshot needs them.

Next candidates: one-byte (alpha) glyph sheets, 4× smaller, for about 3–5
MB per app and less GPU memory (a new image format in paintengine2d); the
same pixmap change for X11.

## Checks that ran

`go build`, `go vet`, gofmt and `go test ./...` pass in both repos, as does
`go test -race ./...` for uitoolkit, with no data races. Contract tests cover
every pack:
- controls paint inside their bounds;
- shadows stay within their reach;
- keyboard focus is visible;
- text meets contrast checks.

On a live compositor (nested KWin at 1.75), the platform tests pass 30
runs in a row and run race-clean; before the clipboard fix, 3 in 11 runs
crashed.

A slip on 2026-09-15: one `go test ./platform` ran without unsetting
`WAYLAND_DISPLAY`, so its live-compositor tests opened test windows on the
real session for under a second. The two that failed there
(`TestWaylandSurfacePresent` expected 160px where 1.75 gives 280;
`TestWaylandPresentOpaqueColor` read the first frame too early) are fixed.
