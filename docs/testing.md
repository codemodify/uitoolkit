# Testing uitoolkit

Automated tests exist so basic UI bugs (blank lists, header bleed/gap,
splitter overlap, stuck resize cursor, unclamped scroll, typing into a
read-only message view) fail **before** a tag — not after a human
tweaks pixels.

Peers do the same thing with different names:

| Toolkit | How they test widgets |
| --- | --- |
| **Qt** | `QTest` event injection (`mousePress` / `keyClick`) plus item-view scroll tests that assert the visible row window, not screenshots. |
| **GTK** | Fixtures, wait-for-draw / tick the frame clock, then check `GtkAdjustment` bounds and allocation rectangles. |
| **Avalonia** | Headless (`HeadlessPlatform`) layout + input; assert arranged bounds and control state. |

uitoolkit’s analogue is `internal/uitest` (Measure/Arrange, inject
Mouse/Wheel/Key, optional paint / scene record, geometry assertions)
plus `internal/apptest` (scripted showcase + Settings).

## Mail safety — moved with the app

Mail lives in its own repository now
([comms-mail](https://github.com/codemodify/comms-mail)), and the rule that
never let a test touch a real mailbox went with it: the in-memory `StartDemo`
fixture, the isolated temp config, the refusal of any socket that is not a
disposable one. Nothing in this repository speaks IMAP any more.

What stays here is the shape of that rule, because every app in the family
needs it: **a test must not be able to reach the thing the user actually
cares about.** In this repository that is the desktop rather than a mailbox —
see `tools/testenv.sh` below, which is why no test can open a window on the
session it runs in.

## How to run

**Run tests through `tools/testenv.sh`** on a desktop machine. It gives the
command no display, a private D-Bus session bus that can start no services,
and runtime, config, cache and data dirs of its own. Unsetting
`DBUS_SESSION_BUS_ADDRESS` alone is not enough: godbus and the platform
layer both fall back to `$XDG_RUNTIME_DIR/bus`, the real session bus, where
a tray, notification or portal test would reach the desktop it runs on.

```bash
tools/testenv.sh go test -p 2 ./...
```

Headless widget + driver suite (no display, no CGO):

```bash
CGO_ENABLED=0 go test ./...
```

Full-paint benches (Settings / showcase / list) live in `internal/apptest`.
See [perf.md](perf.md) for the v0.15.0 command line and numbers.

Tray tests use `UITK_TRAY=fake` or the stub (`docs/tray.md`). Do not
expect a real StatusNotifier host in CI.

Skip the slower driver compose pass:

```bash
CGO_ENABLED=0 go test ./... -short
```

Scripted app driver (same checks, prints one line per step):

```bash
go run ./cmd/uitest-driver
go run ./cmd/uitest-driver -short
go run ./cmd/uitest-driver -app=showcase
go run ./cmd/uitest-driver -app=settings
go run ./cmd/uitest-driver -compare
```

Chrome vs Avalonia / Qt / GTK (focus-visible, toolbar gaps, toggle
chrome, menu dismiss, Office XP menu hover, scroll/splitter/table,
combo/field heights, switch/slider/tabs/dialog) is
`TestDesktopChromeNorms` and `-compare`. See [compare.md](compare.md).
`TestChromeStripGolden` is the 1×/2× paint-hash guard
(`UITK_UPDATE_GOLDEN=1` rewrites `internal/uitest/testdata`).

Linux CGO build (Wayland/X11) — required on a real desktop, not just
`CGO_ENABLED=0 go test`:

```bash
CGO_ENABLED=1 go build ./cmd/uitoolkit-settings
CGO_ENABLED=1 go build ./examples/uitoolkit-sample-tour
```

Settings (`cmd/uitoolkit-settings`) **Apply** writes theme, corners, and icons
to `$XDG_CONFIG_HOME/uitoolkit/look.json`. The pickers preview only; close
without Apply discards. **Export current theme…** writes a palette-only
`themes/<name>/theme.json`. **Delete** (User themes only) confirms then
removes that folder. `Application` watches `look.json` when Look
came from `PreferredLook` (`Look == nil` or `WatchLook: true`). Widget
tests that persist appearance should point `XDG_CONFIG_HOME` at a temp
dir (same isolation idea as Mail chrome prefs). Tests that pass
`DarkLook` / `LightLook` do not watch unless they set `WatchLook`.
`Application.Run` (Wayland / X11 / offscreen) calls `pollLookFile` on
each idle wake (`waitTimeout` ≤ 300 ms when `WatchLook` is on). Mail
rebuild / `persistChrome` must not change `look.json`; Settings Apply
must update a running Mail look via that watcher.

A display is **not** required for `go test` or `uitest-driver`. Native
backends stay behind `CGO` build tags; headless contracts always run.

## The samples build on the published API

`TestSamplesUseOnlyThePublicAPI` (root package) parses every Go file under
`examples/`, `cmd/`, `showcase/` and `tools/` and fails if one imports a
`github.com/codemodify/uitoolkit/internal/...` package. The samples are
the toolkit's first customer: what they can reach is exactly what a
`go get` gives anyone, so a sample that reached into `internal/` would be
demonstrating something no reader can do, and would hide the piece of
public API that is actually missing.

Two commands are exempt, named in the test: `cmd/uitest-driver` (the
end-to-end rig's other half, which drives a window with `internal/uitest`
and `internal/apptest`) and `cmd/uitk-themesheet` (the theme atlas, which
renders with `internal/themesheet`). Neither is a sample. Nothing else
goes on that list — when the test fails, either use the public API or
make the thing it needed public.

Each sample's own code lives in a package beside its `main.go`
(`examples/uitoolkit-sample-files/filesapp`, `examples/uitoolkit-sample-notes/notesapp`,
`examples/uitoolkit-sample-inspector/inspectorapp`, `examples/uitoolkit-sample-tour/tourapp`,
`cmd/uitoolkit-settings/settingsapp`), with the widget gallery in the top-level
`showcase` package because Settings shows it too. That is also what lets
the tests and `uitest-driver` drive them as libraries.

## What the driver covers

After each scripted step it runs `uitest.TreeInvariants` (exclusive
splitter panes, scroll clamp, thumb-in-track, non-empty visible-row
window when content remains, table first-row flush under the header).

**Showcase** (`showcase.App`): construct, open a ComboBox and
assert the popup clears the field and fits labels, resize, drag every
splitter to several ratios, scroll lists/tables/trees/cards/ScrollViews
to top / mid / end and back, select rows.

**Settings**: the same geometry / scroll / splitter passes, select rows in
the theme browser, open a row context menu (full labels, all items, Escape
dismisses), focus a read-only preview and type into it (must fail), and
(unless `-short`) open each page in turn in its own window — a page builds
its controls when it is first shown, which is where construction breaks.

## Drag and drop

The protocol is testable without a display, and it is meant to be: a drag
is a conversation between two applications, and almost none of it needs
pixels.

- **The wire format and the policy** are pure Go: XDND's client messages,
  the version two sides settle on, the search for the window under the
  pointer (over an interface the tests fill with a table — reparenting
  frames, stacking order, `XdndProxy`, unmapped windows), the action
  negotiation and the modifier convention. `platform/xdnd_test.go`,
  `platform/drag_test.go`.
- **The drag source's state machine** runs on the offscreen backend,
  which implements `DragSurface` and `DropNegotiator` and stands in for
  the desktop:

```go
off := w.Surface().(*platform.Offscreen)
off.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: from, Button: platform.ButtonLeft})
off.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: far, Button: platform.ButtonLeft})
a.PumpOnce()                        // the press has become a drag
p, _ := off.DragOffer()             // the types, actions and icon it offered
off.SimulateDragOver(pos); a.PumpOnce()
mime, action := off.DragAccepted()  // what the window told the source
off.SimulateDragDrop(pos); a.PumpOnce()
off.DragEnded()                     // the action the source was told ran
```

  `off.CancelDrag()` is Escape, `off.DragData(mime)` reads what a target
  would have got. `app/drag_test.go` covers the press threshold, the
  negotiation, the payload an in-process drop carries, cancelling, and a
  refused drop reporting nothing performed.
- **Widget behaviour** — what a press drags, what a row or tab takes —
  belongs in `widgets/drag_test.go`, with no window at all.
- **The backends themselves** need the nested-KWin rig (`tools/e2e`), two
  applications and a real file manager. What only that can show: the drag
  icon on screen, a drag between two processes, a drop to and from
  Dolphin or Nautilus, and the desktop's own modifiers choosing a move.
  `UITK_XDND_DEBUG=1` traces an X11 drag when one goes quiet.

## Adding a regression test

When you fix a UI bug, add a `go test` that would have failed on the
broken code:

- Prefer `internal/uitest` (geometry, hit-test, paint-pixel counts,
  `ChromeNorms`) over screenshots.
- Put widget contracts in `widgets/*_test.go`. Map the bug in the
  comment at the top of `widgets/contract_test.go`.
- If the bug only shows up in a real app, add a driver step in
  `internal/apptest` with an invariant check after the action.
- Context-menu / MenuBar clip: `TestPopupMenuFitsLongLabelsAndManyItems`,
  `TestPopupMenuVIPLabelNotClipped`,
  `TestPopupMenuAtRightEdgeKeepsIntrinsicWidth`,
  `TestMenuBarDropdownFitsLabelsAndShortcuts`,
  `TestMenuBarHelpNearRightEdgeFitsAboutMail`, Mail driver `context-menu`.
- ComboBox overlap / clipped rows: `TestComboBoxPopupClearsFieldAndFitsLabels`
  plus the showcase driver `combo-popup` step.
- Drag and drop: the protocol in `platform/xdnd_test.go` and
  `platform/drag_test.go`, the source's state machine in
  `app/drag_test.go`, what a widget drags in `widgets/drag_test.go`.

Do not land a “looks fine on my machine” fix without a test.
