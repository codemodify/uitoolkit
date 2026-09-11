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
```

Linux CGO build (Wayland/X11) — required on a real desktop, not just
`CGO_ENABLED=0 go test`:

```bash
CGO_ENABLED=1 go build ./cmd/mailclientui
CGO_ENABLED=1 go build ./cmd/mailclientd
```

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

## Adding a regression test

When you fix a UI bug, add a `go test` that would have failed on the
broken code:

- Prefer `internal/uitest` (geometry, hit-test, paint-pixel counts)
  over screenshots.
- Put widget contracts in `widgets/*_test.go`. Map the bug in the
  comment at the top of `widgets/contract_test.go`.
- If the bug only shows up in a real app, add a driver step in
  `internal/apptest` with an invariant check after the action.
- Context-menu / MenuBar clip: `TestPopupMenuFitsLongLabelsAndManyItems`,
  `TestMenuBarDropdownFitsLabelsAndShortcuts`,
  `TestMenuBarHelpNearRightEdgeFitsAboutMail`, Mail driver `context-menu`.
- ComboBox overlap / clipped rows: `TestComboBoxPopupClearsFieldAndFitsLabels`
  plus the gallery driver `combo-popup` step.

Do not land a “looks fine on my machine” fix without a test.
