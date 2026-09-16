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
plus `internal/apptest` (scripted gallery + Mail).

## Mail safety (non-negotiable)

**Do not delete, junk, archive, move, expunge, empty trash, or send
real mail.** The automated driver and `go test` Mail paths must never
touch a live IMAP/POP3 account.

1. Never connect the driver to a live daemon from the user’s
   `~/.config/uitoolkit/mail.json` / production credentials / QQ / real
   accounts.
2. Mail UI tests use an **in-memory** `StartDemo` / `MemoryStore`
   fixture (or an isolated temp config that cannot be the user’s
   mailbox). Default is the **fake backend**.
3. Do not call destructive RPCs (`Delete`, `Junk`, `Archive`,
   `Expunge`, Empty Trash, `Move`, `Send`) against a real daemon.
4. Prefer driving **gallery** and headless widget trees. For Mail,
   inject synthetic lists via `StartDemo` or `mail.Open` on that
   fixture.
5. `mail.IsolateTestEnv` / `IsolateTestEnvTB` point XDG +
   `UITK_MAIL_CONFIG` at a temp dir, force `UITK_MAIL=memory`, and
   unset `UITK_MAIL_HOST` / `USER` / `PASS` / `SOCK`. The driver
   refuses any socket that is not a `StartDemo` temp path
   (`mailclientd-*` directory) and refuses any backend other than
   `memory`.

`cmd/mailclientui` against a user-started daemon is **not** a test
entry point.

## How to run

Headless widget + driver suite (no display, no CGO):

```bash
CGO_ENABLED=0 go test ./...
```

Full-paint benches (Mail / gallery / list) live in `internal/apptest`.
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
go run ./cmd/uitest-driver -app=gallery
go run ./cmd/uitest-driver -app=mail
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
CGO_ENABLED=1 go build ./cmd/mailclientui
CGO_ENABLED=1 go build ./cmd/mailclientd
CGO_ENABLED=0 go build ./cmd/uitksettings
```

Settings (`cmd/uitksettings`) **Apply** writes theme, corners, and icons
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

## What the driver covers

After each scripted step it runs `uitest.TreeInvariants` (exclusive
splitter panes, scroll clamp, thumb-in-track, non-empty visible-row
window when content remains, table first-row flush under the header).

**Gallery** (`internal/demo.Gallery`): construct, open a ComboBox and
assert the popup clears the field and fits labels, resize, drag every
splitter to several ratios, scroll lists/tables/trees/cards/ScrollViews
to top / mid / end and back, select rows.

**Mail** (in-process `StartDemo` only): same geometry/scroll/splitter
passes, select a thread row, open the message context menu (full labels
+ all items), focus the message `TextView` and type (must fail), and
(unless `-short`) open compose and type into the editable body. No
Delete / Junk / Send / Move.

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
  plus the gallery driver `combo-popup` step.
- Drag and drop: the protocol in `platform/xdnd_test.go` and
  `platform/drag_test.go`, the source's state machine in
  `app/drag_test.go`, what a widget drags in `widgets/drag_test.go`.

Do not land a “looks fine on my machine” fix without a test.
