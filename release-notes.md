# Release notes

What changed in each release, and why. The reasons are here because a
list of names is not much use six months later: what a reader usually
wants to know is whether a change affects them, and that is a question
about the problem it solved.

---

## 0.23.0

Twenty-five items from the same two applications — a mail client and a
password vault — and this time nearly half of them were the toolkit's own
breakage: a fix to something 0.22.x shipped, or a hole left in it. One of
the twenty-five turned out not to be a gap at all.

### If you are upgrading

Four changes alter behaviour rather than adding to it.

- **`Pad`, `Spacer` and `ProgressBar` lengths follow the display scale.**
  0.22.0 made `FlexBox`'s gap and padding and `Grid`'s row and column
  gaps 1x design lengths and left these three in device pixels, so a form
  at 2x had scaled gaps between unscaled pads and a progress bar half as
  long as everything beside it. **If your application multiplied them by
  the scale itself, stop.** `RawSpacing` on `Pad` and `Spacer` is the way
  back, as it already was on `FlexBox` and `Grid`. At scale 1 nothing
  moved.
- **A flexible grid column may now be narrower than its content.** A
  wrapping label in a `Form` gets the width the form has and folds,
  instead of taking the width of its whole text on one line and running
  off the edge. Nothing that cannot fold is squeezed.
- **Overlays stack.** `ShowOverlay` pushes and `DismissOverlay` pops, so
  a dialog shown from inside another no longer closes it. If you relied
  on the second dialog replacing the first, call `Window.SetOverlay`,
  which still means "there is one dialog".
- **A square-cornered pack draws square chips.** 41 of the 135 packs were
  getting a capsule because a corner radius of zero read as "no opinion".

### Layout

- **A column that can wrap is allowed to.** A flexible track was frozen
  at its children's unbounded width when there was not room for it: right
  for a button, wrong for a label. Which one a child is cannot be told by
  width — narrowed, both come back narrow — so it is told by height: what
  folds gets taller, what cannot keeps its height.
- **`Grid` measures its rows at the widths it will use**, and `Overlay`
  settles its card's width before asking for its height. Both measured at
  one width and laid out at another, which left a form taller than what
  it drew and a dialog's buttons outside the dialog, where clicks on them
  missed. Every other container is swept by a test.

  The same rule applies to an application and cannot be enforced from
  inside the toolkit: a column measured at 600 and handed a 300-wide box
  of the height that measurement asked for still puts its last child
  outside the box. A container cannot grow a box it was given.
- **`Wrap` and `TokenField` flow at whole pixels**, because that is what
  `Arrange` has. A measure at 300.4 and a layout at 300 disagreed about
  which child fits on the line, and it showed as a line's height of empty
  space under the last row.

### Icons

- **Seventeen more**, and a way to name the rest. Fifteen typed ids for
  actions the five packs already shipped with no id to reach them by —
  trash, archive, junk, tag, folder, reply-all, settings, external-link,
  eye, user, bell, send, close, quit — plus print, which no pack carried
  and is now rendered from the same pinned upstreams as the rest.
  `IconByStem` reaches the whole shipped vocabulary; `IconStarFilled` and
  `IconDot` are drawn by the toolkit in every set, because neither is
  shipped by an outline pack and a filled star has no house style to
  match.
- **No icon draws nothing.** The drawn sets had no default arm at all, so
  anything they had no vector for simply did not appear — a button with
  an invisible mark on it. This mattered more than it looked: every
  shipped pack uses the drawn Classic set unless the user picks
  otherwise.
- **`Button.Icon`.** Only `ToolButton` and `MenuItem` had one, so a
  dialog's buttons could not carry a mark. The engine centres and draws
  the label, so the button reserves a strip at each end rather than
  pushing the text along, and every era's label treatment survives
  untouched.
- **Art**: heroicons' forward was the media fast-forward; the drawn
  paperclip read as a rounded box at the size a message list uses; the
  new cog read as a sun and the new bell as a lampshade.

### Colour

- **`Palette.Ink`, `DangerInk`, `WarningInk`, `SuccessInk` and
  `ReadableInk`.** The declared status colours are era-faithful on
  purpose — Clearlooks really did use `#c4a000` — and most of their uses
  are fills, where the chrome around them carries the contrast. Text is
  the other case, and 225 of the 405 status pairs across the shipped
  packs did not reach 4.5:1, the worst near 1.6:1. Both are kept now: the
  palette holds what the pack declared, and `Ink` is the same colour
  moved along its own lightness until it reads, hue intact. All 405 pairs
  pass as ink; the 180 that already read are returned untouched.
- **`Label.Tone`** says what a label's text *means* — danger, warning,
  success, muted, accent — and takes the ink form at paint time, so it
  follows a theme change. An application that set `Color` from the
  palette got the unreadable version and had to set it again on every
  look change.

### Secrets

- **`SecretArea`**, for a PEM block or an OpenSSH private key:
  `SecretField` over several lines, and the same widget rather than a
  second one, so the buffer, the wipes and the refusals are not written
  twice. A pasted key keeps its lines and CRLF is normalised in place.
- **`SecretField.SetBytes`** takes a generated passphrase over rather
  than copying it, and wipes what it replaces.
- **`SecretClip.OnCleared`**, for the "copied — clears in 45s"
  indicator. The alternative was polling `Cleared`, which is a timer of
  its own and a window in which the interface is wrong.
- **Caps Lock in the field itself.** `widget.LockKeysWatcher` gives every
  widget in the window the lock state, where `Window.OnLockKeys` was one
  callback that a widget had to take and hand back. A focused
  `SecretField` draws the mark with no application code.
- **`SecretField` declares itself a `widget.SecretTarget`.** Not being an
  `IMETarget` already kept the input method away; declaring it lets
  anything that asks get a straight answer rather than inferring one.

### Dialogs and windows

- **Overlays stack**, so a confirmation raised from inside a dialog comes
  back to that dialog with its fields as they were — and a dialog that
  wipes its secret fields on close no longer wipes them because something
  else opened.
- **`MessageBoxInput.Validate` and `AcceptLabel`.** A prompt can refuse a
  value without closing, showing why under the field with what was typed
  still there; and the accepting button can name its action, which is
  what every desktop's guidelines say. `SetInputError` is the same for a
  check that is a round trip; `PromptFor` is the short way to both.
- **A fitted dialog is centred for the height it ends up at.** A resize
  keeps the top-left corner, and the caller cannot sequence the two
  itself because the fit is what changes the size.
- **`StatusItem.Shown` and `SetOnShownChange`.** `Alive` says there is
  somewhere to send the item; `Shown` says someone is showing it. They
  differ on exactly the desktop where getting it wrong costs the user
  their window: a GNOME with no AppIndicator extension has a session bus
  and no tray.

### Rich text

- **A table cell keeps its bold, its code and its links, and wraps.** A
  cell was drawn as its plain text in one face fitted to one line, so a
  newsletter's table of links lost its links. Cells are laid out the way
  paragraphs are now, through the same two halves of the code, because a
  second text layout is how a widget ends up with two sets of rules.
- **The block after a table gets its space.** Rows of one table stay
  tight — they are one grid — but a quote's rule used to butt straight
  onto the bottom row.
- **A protocol-relative image src is remote.** `//host/pixel.gif` has no
  scheme and was read as a file beside the document, so a mail client
  told an image was local fetched a tracking pixel without asking.

### Fixed

- **A chip no longer cuts its own text short.** `Token.Measure` asked for
  a fractional width and `SetBounds` rounds each edge on its own, so the
  box could come back a pixel under and `Paint` elided the label it had
  just measured for.
- **A quoted comma does not split a token.** `"Lovelace, Ada" <ada@x>`
  was two recipients, both nonsense. An unclosed quote is not an error:
  the user is still typing.
- **A chip field measured without a width asks for a field's width**, not
  for every chip on one line. That measurement was what pushed a Write
  window's form past the window's edge.
- **`UITK_FONT` no longer stops a program following the desktop's dark
  mode.** The override was folded into the file read from disk, so a
  machine with no `look.json` resolved an empty file instead of the
  defaults — and an empty file says `FollowDesktop` is off.
- **A widget taken out of the tree lets go of its window.** `Add` hands
  the host down a subtree and `Remove` cleared only the parent, so
  anything holding a removed component kept the window reachable.
- **`ClickComponent` and `FocusOn` scroll to the target first**, as a
  person does. A button below a `ScrollView`'s fold was clicked where its
  box said it was, which is outside the view.

### Not done

**Font fallback for message content** was asked for again and is still
declined. One face draws everything; a rune it lacks is a box. A product
that must show other people's arbitrary scripts needs a toolkit with a
shaping engine, which [contracts.md](contracts.md) says plainly and at
length.

**`widgets.Wrap` was reported missing and is not.** It has laid controls
out in a row that folds since before 0.22, and is documented. That one
was a discoverability failure rather than a gap, and the report has been
withdrawn.

### Verified

The suite with every engine and with one; Windows and Darwin
cross-compiled. The 64 failures in a one-engine build are unchanged and
all of the same kind — a test naming a pack that build does not have.

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
