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

**Or run `tools/test.sh`**, which is that plus `-tags theme_engine_all`,
so every theme engine the toolkit ships is compiled in and tested — a
bare `go test ./...` tests the *default product build*, one engine, and
a test written about what Aqua paints has nothing to assert when Aqua
was not built.

```bash
tools/test.sh ./...
tools/test.sh -run TestFoo ./style/
```

It also pins `UITK_SYSTEM_FONTS=0`. A look reads in the era's typeface
when the machine has it, so what a look *measures* — where a panel
ends, how a sentence wraps — depends on what is installed, and a test
that asserts a pixel would pass here and fail on a machine with a
different font set. Three did: on macOS, which has no fontconfig at
all, and they would have in any clean container too. Pinning the
bundled faces makes those numbers a property of the toolkit rather than
of the developer. Set `UITK_SYSTEM_FONTS=1` to run against this
machine's fonts.

**The other two platforms** are tested on real ones, from here:

```bash
UITK_WIN_VM_MON=/path/to/monitor.sock tools/test-windows.sh   # docs/windows.md
UITK_MAC_HOST=user@the-mac            tools/test-darwin.sh ./...  # docs/macos.md
```

Both run the **whole suite**, not just `./platform/`. The Windows one
used to run that single package, and the asymmetry cost real bugs: the
widget, style and app suites had never executed on Windows, so a layout
that depended on the developer's installed fonts, or a path assumption,
surfaced on macOS first and only because macOS ran everything. Widening
it turned up five failures the same afternoon — four tests written in
Unix terms, and one real one (see below).

The Windows tests cross-compile with `CGO_ENABLED=0` and are handed to a
VM over QEMU's user networking; `go test -c` builds one package at a
time, so a full run is two dozen binaries, fetched and deleted one at a
time so the guest never holds more than one. The source tree goes over
once as well, because some tests read files next to themselves
(`internal/uitest` opens `testdata/`, `skingen` reads `../style/skins`)
and each binary is run from its own package's directory, as `go test`
would. `UITK_WIN_PKGS=./platform/` narrows it.

The macOS ones cannot cross-compile at all, because AppKit is cgo, so
the tree is mirrored to a Mac with rsync and built there. Both bring
their output back.

**A test that only makes sense on one platform skips on the others,
with a reason** — the XDG icon theme search path, KDE and GTK
configuration files, Unix permission bits, making a directory
unwritable with `chmod`. A build tag would take the whole file out;
these files are mostly portable and worth running everywhere, so the
skip is per test and says what it is about.

Headless widget + driver suite (no display, no CGO):

```bash
CGO_ENABLED=0 go test ./...
```

### The screenshots in the documentation are scripted

Every picture under `docs/screenshots` is rebuilt by a command, and that
is a rule rather than a convenience: a screenshot no script can rebuild
is one that will be wrong and stay wrong, because by the time the thing
it shows has changed nobody can reconstruct how it was taken.

```bash
tools/atlas/render.sh          # every pack, twice: the preview and the gallery
python3 tools/atlas/build.py   # the Theme Atlas page the timeline is cut from
tools/atlas/sheets.sh          # the contact sheet per decade, and the timeline
tools/atlas/frames.sh          # one window frame per era
tools/atlas/breadth.sh         # rich text, MDI and the wizard in nine looks
go run -tags theme_engine_all ./cmd/uitk-shots docs/screenshots
```

The last one writes the gallery and the per-widget comparison thumbnails.
All of them are headless; nothing opens on the desktop.

### Driving a headless window from an application's own tests

`Headless: true` makes a window that renders to a buffer and takes
events; `Window.Inject` is what the backend would have sent and
`Application.PumpOnce` runs a frame. Those two have always been enough,
and on top of them are the four everybody writes anyway:

```go
a := uitoolkit.New(uitoolkit.Options{Headless: true})
w, _ := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
w.SetContent(form)
a.PumpOnce()

w.FocusOn(field)                 // the Tab a test would have to count
w.Type("correct horse")          // one text event per rune
w.Press(platform.KeyA, platform.ModCtrl)
w.ClickComponent(okButton)       // through the window's hit testing
```

Each runs a frame before it returns, so the effect has happened by the
time the call does. `Type` is a keyboard, not `SetText`: the widget's own
input handling runs, so `OnInput` fires, an `Accept` rejects what it
would reject, and a `SecretField`'s buffer takes the path a real key
takes. `ClickComponent` goes to the middle of the widget's arranged box
through the window's hit testing, so a widget under a popup is not
clicked and a disabled one hears the press and does nothing; it answers
false for a widget that was never laid out rather than clicking the
corner of the window. `Window.WritePNG` is the frame.

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
`cmd/uitoolkit-settings/settingsapp`), with the widget showcase in the
top-level `showcase` package because the tour spreads it over three
pages. That is also what lets the tests and `uitest-driver` drive them as
libraries.

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
