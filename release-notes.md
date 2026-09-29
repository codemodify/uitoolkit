# Release notes

What changed in each release, and why. The reasons are here because a
list of names is not much use six months later: what a reader usually
wants to know is whether a change affects them, and that is a question
about the problem it solved.

---

## 0.22.1

The contracts 0.22.0 should have carried, and a crash found while
writing them down.

### The text contract

[docs/contracts.md](docs/contracts.md) is new: one page for what the
toolkit promises about text and what it does not. The rules existed —
scattered across a widgets subsection, a comparison page's gap list and
a README table row — and one of them was not written down anywhere.

Two rules, stated as rules:

- **A mark is an icon, not a character.** `✓ ✗ →` typed into a label are
  boxes on any machine whose selected face lacks them, and on every
  machine under `UITK_SYSTEM_FONTS=0`.
- **Text is text, not an icon.** The converse, and the one nothing draws
  a box to warn you about: an icon-only control needs
  `SetAccessibleName`, because a glyph cannot be read aloud, translated
  or searched; and prose must not be rendered as a picture to dodge the
  script limits, because it cannot be selected, copied, found or scaled.

What is not supported, in one place: **no `GSUB`** (no Arabic joining,
no Indic reordering, no ligatures of any kind), **no bidi** (right-to-
left text comes out reversed), **no right-to-left interface** — nothing
reads a locale's direction, nothing mirrors, and there is no API to ask
— and **no font fallback**, which is the one that surprises people:
installed fonts decide which single face is *selected*, not which glyph
answers for a rune.

The two bundled faces are now stated exactly, because they differ and
the difference is a trap. Titillium Web has Latin and nothing else;
JetBrains Mono carries `✓ ✗ → ← └`, Greek and Cyrillic. Both are pinned
by a test, so the table cannot drift from the fonts — and the page says
plainly not to set `Mono` on a label to win a glyph, which is rule 2
broken from the other side.

### Fixed

- **A font that cannot draw Latin no longer kills the process.**
  `BakeFamily` panicked on a script-only face — Noto Sans Arabic, Noto
  Color Emoji and about thirty others the font chooser offers on an
  ordinary desktop. The choice is saved in `look.json`, so picking one
  crashed every uitoolkit application on the machine at start, and kept
  crashing until the file was edited by hand. It falls back to the
  bundled face now, which is what an uninstalled family has always done;
  the two cases had no business differing. A test walks every installed
  family and asserts each gives back a face that can draw the interface
  — 265 on the machine this was written on, of which 30 used to be
  fatal.

---

## 0.22.0

Everything in this release came from two applications built on the
toolkit — a mail client and a password vault — writing down what they
could not do and why. Each item below closed one of those, and the two
lists are now down to a single open entry between them.

### If you are upgrading

Five changes alter behaviour rather than adding to it. None needs a code
change, and three of them will silently improve an application that did
nothing.

- **Gaps and padding follow the display scale.** `FlexBox`'s `Gap` and
  padding and `Grid`'s (and so `Form`'s) `ColGap` and `RowGap` were
  device pixels; they are 1x design lengths now, scaled by the look like
  every other length in the toolkit. **If your application multiplied
  them by the scale itself, stop** — doing both makes them square. Set
  `RawSpacing` on either to keep device pixels. At scale 1 nothing moved
  at all.
- **A program with no `look.json` follows the desktop's dark mode.**
  `DefaultAppearance` has `FollowDesktop` on. A saved `look.json` is
  still a choice and is not overridden.
- **A `TextField` with nothing to delete lets Backspace bubble**, as it
  already did for Return. A parent that handles Backspace will now see
  it.
- **`Modifiers` carries the lock keys.** Compare shortcuts with
  `Modifiers.Chord()`, which masks them out, if you compare the whole
  set rather than asking `Ctrl()` and friends.
- **`richtext` has three new block kinds.** Code that switches on
  `Block.Kind` should expect `Quote`, `Rule` and `TableRow`; the last
  keeps its text in `Cells` as well as flattened into `Spans`.

### Handling secrets

A Go string cannot be wiped: every copy lives until the collector
reaches it and then lingers in freed memory. A widget that holds a
passphrase must not be built out of one.

- **`SecretField`** keeps its contents in a `[]byte` edited in place.
  The array is zeroed when the buffer is outgrown and when a delete
  shortens it; `Bytes()` hands out a copy the caller owns and wipes
  (`WipeBytes`); `Len()` is a rune count and no content, for a strength
  meter. Copy, cut, dragging the selection out, the X11 PRIMARY
  selection and middle-click paste are all refused. A test reads the
  source to keep any `string(` conversion out.
- **`SecretLabel`** is the read-only half — an item view's "show
  password", a TOTP code. Its value comes from a function, so a locked
  vault empties it; `Hide()` zeroes what it last drew from.
- **The input method is off** while a secret widget has the focus
  (`widget.SecretTarget`), which now covers `TextField` with `Password`
  too. An input method is another process that sees every keystroke and
  learns from it, so masking the preedit was never enough. X11's
  `SetIMEEnabled` was a no-op; it now unsets the input-context focus and
  resets it.
- **`ClipboardSetSecret`** writes CLIPBOARD only — never PRIMARY —
  hinted so clipboard managers do not record it, and clears itself after
  a timeout (45 s by default) and on quit. `ClipboardGetSecret` reads
  back as bytes.
- **`Window.SetSecureInput`** and **`SetExcludeFromCapture`**, each
  reporting what actually happened and each with an `Available` so an
  application can decide what to promise. They are not the same promise
  on every platform: macOS takes the keyboard from every other process,
  X11 stops the server delivering keys to other clients, Wayland
  inhibits the compositor's own shortcuts, and Windows has nothing an
  ordinary application may use.
- **`Modifiers.ModCapsLock`**, `Window.LockKeys()` and `OnLockKeys`. A
  right passphrase refused is nearly always Caps Lock, and a field that
  holds a secret cannot work that out for itself — the only other way is
  to read the secret.

### Windows and dialogs

- **`WindowOptions.Role`, `Center` and `KeepAbove`**, applied before the
  window is shown, plus `Window.Activate`, `Center` and `SetWindowRole`.
  A prompt opened from a terminal now appears in front, focused and
  centred. `_NET_WM_WINDOW_TYPE_DIALOG` on X11, `xdg_dialog_v1` on
  Wayland, an owned dialog-framed window on Win32, a floating `NSPanel`
  on macOS.
- **`WindowOptions.FitContent`** and **`Window.FitToContent()`** make a
  window as tall as its content, measured at the window's own scale and
  through its own frame — the two things a caller measuring by hand
  could not know at the same time.

### Widgets

- **`Prompt`** — the "name this" dialog, as `MessageBoxOptions.Input`:
  the field takes the focus, Return is the default button, the initial
  value comes up selected, and `Required` greys OK out while it is
  blank.
- **`TokenField`** — chips with a cross and an editor at the end, the
  recipient field of every mail client written since Gmail. A comma, a
  semicolon or Return ends a value; leaving the field commits what is in
  the editor; a refused value stays where the user can see it.
- **`FileOpenFolder`** and **`FileDialogOptions.Name`**. A Save `Path`
  that names a file is also split into the folder to list and the name
  to suggest, which used to produce an empty listing and no error.
- **Per-tab disable and hide** on `TabView` and `TabBar`. A hidden tab
  takes no width; a disabled one is stepped over by the arrow keys; Home
  and End mean the first and last tab there *is*.
- **`HeightForRows`** on `ListView` and `TableView`, for a popover sized
  by how many rows it should show.
- **`ScrollView.ShrinkToContent`** and `MaxHeight` — a panel as tall as
  its contents up to a limit.
- **`widget.SetPopupKeysPass`** — a non-capturing popup, so a completion
  list can be open while the user goes on typing into the field under
  it.
- **Icons in views**: `TableColumn.Icon`, `TableView.CellIcon` with a
  colour, `TreeNode.Icon`, and seven stock ids — `IconAttach`,
  `IconStar`, `IconFlag`, `IconReply`, `IconForward`, `IconCheck`,
  `IconMute`. Every one resolves to a PNG the five shipped icon sets
  already carry. A message list's marks used to have to be characters,
  which meant they came out of the font.

### Rich text

- **Tables, quotes and rules.** `<table>` was read for the characters
  inside it and nothing else, `<blockquote>` as plain paragraphs, and
  `<hr>` was dropped. There are now `Quote` blocks (one accent rule per
  level of nesting, so a thread's depth reads at a glance), `Rule`, and
  `TableRow` — a table being a run of rows whose column widths are
  measured once for the run.
- **`ResolveImageKind`** tells the callback whether a src is inline,
  remote, local or a data URI. A mail client cannot treat the first two
  the same: an inline part came with the message, and a remote one tells
  the sender it was opened. `Doc.Images`, `UnresolvedImages` and
  `ImageCounts` are the enumeration a "load remote images" control is
  built from.
- An unresolved image shows its **alt text** instead of an empty box.

### Fixes

- **Text measured at its own width no longer wraps.** `Font.Wrap` added
  rune advances, which does not see pair kerning and so reports a line
  wider than it paints — safe for deciding a line is full and wrong for
  deciding it is too full. `/usr/bin/mail` is 78.05 wide and came back
  as two lines with the word broken when measured at 78.05, so every
  wrapping label in a `Form` was a line taller than the line it drew.
- **A wrapping label that flexes in a row is no longer clipped.**
  `layout.Flex` measures a child again at its final width — Qt's
  `heightForWidth`, GTK's height-for-width — and takes the taller
  answer. The pass only ever grows a child, so no existing geometry
  moved.
- **A theme this build cannot paint no longer turns the window dark.**
  The palette family was guessed from the pack's *name*, and the parser
  behind that guess knows the two words "light" and "dark" and reads
  everything else as dark — so every pack a build was missing came up
  dark. `Appearance.Missing()` and `style.MissingThemeNote()` now report
  it, and Settings prints that line.
- **Every pack marks its default button.** Which engines ignore
  `Primary` is measured rather than listed: 129 of 132 mark it
  themselves, and the two Amiga packs get a heavier border.
- **A real tray menu on macOS.** `SetMenu` stored the rows and showed
  nothing.
- **A drop carrying files** goes to the widget that takes files, not to
  a text field nested inside it that would take the paths as text.
- **`TextField.SetText`** leaves the caret at the end.

### Other

- **`plastik-night`**, a dark sibling for the default pack — 132 packs
  now. Without one, a program nobody has configured opened white on a
  dark desktop.
- **`Window.Type`, `Press`, `ClickComponent` and `FocusOn`** for driving
  a headless window from an application's own tests.
- `keyboard-shortcuts-inhibit-unstable-v1` and `xdg-dialog-v1` are
  vendored alongside the other generated Wayland protocols.

### Verified

The suite with every engine and with one; every engine built on its own;
Windows and Darwin cross-compiled; and `platform`, `app`, `widgets`,
`richtext` and `style` run on macOS 15.

---

## 0.21.0 and earlier

Before this file existed the release notes were the tag and its commit
message. `git show v0.21.0` and its neighbours have them.
