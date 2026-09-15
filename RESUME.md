# Resume here — end-to-end fixes and theme engines, 2026-09-14

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

1. **Java looks and licensing.** `engine/metal` (local only, worktree
   `uitoolkit-eng-metal`, not pushed) holds Metal (Steel, Ocean) and Nimbus. They
   are pixel-exact because they were ported and transcribed from OpenJDK, which
   is GPL-2.0 with the Classpath exception, and uitoolkit is MIT. Your options:
   - keep them behind a build tag, under their GPL terms;
   - have them rewritten clean-room from visual facts (queued as a later batch);
   - drop them.

   Every later engine brief forbids transcribing.
2. **The default theme.** The stock looks are "Classic 95 Dark / Light" (a
   Win95 homage). A modern engine such as Adwaita, Breeze or Fluent could be the
   default instead. Say which.

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

In progress, or queued with briefs written:
- KDE (Fusion, Oxygen, Breeze);
- GNOME (Clearlooks, Bluecurve, Human, Cleanlooks, Adwaita);
- modern Windows (Aero, Windows 8 and 10, Fluent);
- KDE 3 (Keramik, Plastik);
- the first desktops (Mac System 1 and 7, Windows 3.1, OPEN LOOK, Amiga, BeOS,
  OS/2);
- a clean-room Metal and Nimbus.

Core features the engines drive, added along the way:
- selected tabs overlap their neighbours;
- menu, tooltip and dialog drop shadows;
- list, tree and table view frames;
- per-axis scroll arrows;
- window backgrounds and tab panes;
- group boxes and in-app dialog windows;
- item states: selected, current, unfocused, and backdrop when the window is
  inactive, each engine following its platform's rules.

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
- **Settings:** restructured as a theme browser with a live, themed preview
  application.
- **`UITK_THEME=<pack>`:** runs any app in any theme, like `GTK_THEME`.
- **`gallery -screenshot DIR -theme <pack>`:** takes every scripted gallery
  shot in one pack.

## Performance

`BenchmarkGalleryRepaint` and `BenchmarkGallerySmallRepaint` in
`examples/gallery` repaint the gallery in each engine.

- paintengine2d keeps recorded paths across frames and allocates draw ops in
  slabs.
- Its CPU fast path now takes gradient rects, and rect lists whose pixels don't
  touch.
- Textured looks batch their stripes.

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
the real-hardware pass.
