# Doing it right

Situations where the obvious way is wrong, and what to do instead.

Every entry here was written because somebody building a real program hit
it, took the obvious route, and found out later. Each says what the
obvious route is, why it does not work, and what does. None of it is
clever; it is all the kind of thing you can only know by having been told.

If you are looking for what a widget *is*, that is
[widgets.md](widgets.md). This page is about what to *do*.

---

## A mark on a button

**Obvious:** put `✓` or `🗑` in the label.

**Wrong** — the interface face is Latin only and there is no font
fallback, so on most machines that is a box. [contracts.md](contracts.md)
is the whole rule.

**Instead**, pick by what the control is:

| the control is | use | face |
| --- | --- | --- |
| a button with a name — OK, Save, Rename | `NewButton` + `Button.Icon` | push button, wide (see below) |
| a button that is **only** a mark, and must look like a button | `NewIconButton(icon, name, on)` | push button, square |
| a button that drops a menu | `NewMenuButton(icon, name, items...)` | push button, square |
| a mark on a **tool bar** | `ToolIconBtn(icon, "", on)` in `NewToolBar` | tool face — flat until hovered |
| a mark beside a name, outside a dialog | `NewToolButton(text, icon, on)` | tool face, tight |
| a mark in a menu row, a cell, a tree node | `MenuItem.Icon`, `TableView.CellIcon`, `TreeNode.Icon` | — |

**The face is the choice, not the size.** `ToolIconBtn` is a *tool* item:
in most eras it is flat with no frame until the pointer is over it, which
is right on a tool bar and wrong anywhere somebody is meant to see that
the thing is a button. If the answer to "why doesn't that look like a
button?" is "because it isn't one", you wanted `NewIconButton`.

A latched button — a search that is on, a filter applied — sets
`Checked` (and `Toggle`, so a screen reader calls it a toggle). 53 of the
135 packs draw a checked button exactly as an ordinary one, so on those
it is drawn pressed instead, which is how Windows 3.1, Motif, CDE and
OPEN LOOK drew a toggle anyway. You do not have to know which pack you
are on.

**A button that opens a menu is `NewMenuButton`, not a `Button` whose
`OnClick` calls `ShowContextMenu`.** The difference is not cosmetic: the
menu opens on press so a drag can run into it, the button stays down
while it is open, pressing again closes it — and its items' **shortcuts
work**. An application menu moved from a `MenuBar` to a plain button
silently loses them, and the first anyone knows is that Ctrl+Q has
stopped quitting.

### What `Button.Icon` costs, and why

A button with `Icon` set is about **64 px wider at 1x** than the same
button without. That is not an oversight, and it will break a row that
only just fitted.

The reason is that a push button's label is drawn *by the engine*, and
every engine centres it in the whole button and decorates it its own way
— Clearlooks embosses it, others shadow it, grey it, underline a
mnemonic. The widget cannot move the label without taking over drawing
it, and taking over drawing it loses all of that. So it reserves a strip
for the icon at **each** end, which keeps the label centred between them
and leaves every era's treatment untouched. The mirrored strip is the
cost of not touching the engine.

**So**: use `Button.Icon` for dialog buttons and anywhere with room.
Where a row is tight, do one of these instead —

- `NewIconButton(style.IconTrash, "Delete", onDelete)` — the same push
  button face with the mark alone, square, and named for the screen
  reader and the tooltip in one go;
- `NewToolButton("Delete", style.IconTrash, onDelete)` — a tool-faced
  button, which lays icon and label out itself and is tight;
- keep `Button.Icon` and let the row fold: `NewWrap(buttons...)`.

If a row of buttons has to fit a narrow window, **measure it**. A
minimum-size test that fails when a button grows is worth more than a
rule of thumb.

---

## A row of controls that must fold

**Obvious:** write a flow layout, because surely there isn't one.

**There is.** `widgets.NewWrap(children...)` lays children out left to
right and starts a new line when the next will not fit — Qt's flow
layout, GTK's `FlowBox`, WPF's `WrapPanel`. Its height follows the width
it is given.

This is here because a mail client wrote its own and found out
afterwards. If you are about to write a layout, search
[widgets.md](widgets.md) first: `Wrap`, `FlexBox`, `Grid`, `Form`,
`ScrollView`, `Pad` and `Spacer` cover most of it.

---

## Shipping your own icons, themes or skins

**Obvious:** copy them into `~/.config/uitoolkit/icons/`, which is where
the toolkit reads them from.

**Wrong, and it breaks other programs.** That directory belongs to the
*person using the machine* — it is this toolkit's `~/.icons`, where they
install a set once for every uitoolkit program. Two applications copying
into it overwrite each other, last one wins, and neither can tell: a
program built against a newer toolkit installs stems that an older
program's copy then removes.

**Instead**, keep your art wherever you like and register it:

```go
style.AddSearchPath("/opt/my-app/share")   // …/share/icons/<set>/*.png
```

The directory has the same shape as the user's — `icons/`, `themes/`,
`skins/`, any of them optional — and is private to your process. **The
user's copy comes first, file by file**: where they have installed a set
their version of an icon wins, and where they have nothing yours
answers. They keep control of how their desktop looks; you can rely on
art you ship.

Only Settings writes to the user's directory, and only when asked.

---

## A passphrase, or anything that must not linger

**Obvious:** a `TextField` with `Password` set.

**Wrong.** That paints bullets and nothing more: the value is the
exported `string` field `Text`, rebuilt on every keystroke, and a Go
string cannot be wiped — every copy lives until the collector reaches it
and then lingers in freed memory. Ctrl+C, Ctrl+X and a drag of the
selection all carry the plaintext out.

**Instead:**

| | |
| --- | --- |
| one line | `NewSecretField(placeholder)` |
| several lines — a PEM block, a private key | `NewSecretArea(placeholder)` |
| read-only — "show password", a TOTP code | `SecretLabel` |
| putting one on the clipboard | `platform.ClipboardSetSecret(b, timeout)` |

The contents live in a `[]byte` edited in place, the array is zeroed when
it is outgrown and when a delete shortens it, and nothing in the widget
turns it into a string. `Bytes()` hands out a copy **you** wipe
(`WipeBytes`); `Len()` is a rune count with no content, for a strength
meter. Copy, cut, drag-out, the X11 PRIMARY selection and middle-click
paste are refused, and the input method is off.

Caps Lock is shown by the field itself — see below.

---

## Sizing something that wraps

**Obvious:** measure the container, then arrange it in the box you got.

**Wrong if the width changed in between.** What a widget that folds
reports is only true of the width it was asked about. Measure a column at
600, hand it a 300-wide box of the height that measurement gave, and its
last child lands outside the box — where clicks miss it.

**Instead**, measure at the width you are going to give it:

```go
w := available.Dx()
sz := content.Measure(layout.Constraints{MaxW: w, MaxH: -1})
content.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))
```

Every container *in the toolkit* does this, and a test keeps it true. A
container cannot fix it from the inside: it cannot grow a box it was
handed.

---

## A form that has to fit a narrow pane

**Obvious:** make the window smaller and expect the layout to cope.

**It will stop at a floor, and then run over the edge.** A flexible grid
column shares out the width it is given, but it never squeezes a child
below what that child can actually be: a label folds, so its column can
shrink to its longest word; a text field, a button and a check box have
no narrower form, so their columns stop at what they ask for. When the
floors together are wider than the pane, the row runs over. That is
deliberate — the alternative is a button squeezed to nothing — and Qt and
GTK behave the same way.

What you can do:

- **Lower what the fixed children ask for.** `SecretField.PreferredWidth`,
  `SecretArea.PreferredWidth` and `TokenField.PreferredWidth` are that
  lever. A field that asks for 120 instead of 180 buys 60 px.
- **Fold the row** with `NewWrap`, so the button drops under the field
  instead of off the edge.
- **Use fewer columns.** A label above its field rather than beside it
  halves the floor.
- **Let a splitter take its panes' minimums.** `Splitter` divides by
  `Ratio`, but it will not put a pane below what that pane says it needs
  — so a sidebar cannot be dragged until its buttons run off the edge.
  `MinA` / `MinB` override that, and `AllowCollapse` lets a pane close
  entirely.
- **Ask where the floor is, and tell the window.**
  `widget.MinWidthOf(content)` is the width below which something starts
  leaving the box; `WindowOptions.MinWidth` is where that answer goes, and
  every backend passes it to the window system. A window then cannot be
  dragged below what it holds. Assert it in a test too — that is what
  catches a button growing an icon before a user does.

```go
min := widget.MinWidthOf(form)          // the floor, in device pixels
sz := form.Measure(layout.Constraints{MaxW: max(paneW, min), MaxH: -1})
```

Do **not** try to find it by measuring at a small width. A container
asked to fit in one pixel reports the one pixel it was constrained to,
and a button does the same — measuring a button at 1 px says 65, and a
layout that believed it would draw the button with its label hanging out
of both ends. `MinWidthOf` asks containers, which know, and works the
rest out from the **height**: what folds gets taller when narrowed, what
cannot keeps its height.

## Content that is genuinely wider than its pane

**Obvious:** a `ScrollView`, which is what scrolls things.

**It scrolls up and down only**, unless you say otherwise. A child wider
than the view is clipped at the edge and the rest is unreachable.

That is the default on purpose: a scroll view's usual job is to give its
child the width it has and let it fold. **A form should not scroll
sideways** — drop its labels above its fields, or fold it with `Wrap`.
Reading anything laid out in columns by sliding it back and forth is a
poor interface, and reaching for it is usually a sign the layout is
wrong.

For content that truly has no narrower form — a table with more columns
than fit, a diagram, an image at its own size:

```go
sv := widgets.NewScrollView(content)
sv.Horizontal = true      // and the child keeps its natural width
```

```go
tv.Horizontal = true      // a TableView scrolls its own columns
```

Both give you a bar, the wheel (with Shift for a one-axis wheel), and
the arrow keys. A `ScrollView` that scrolls sideways also stops imposing
its child's width on whatever is above it — `MinWidthOf` drops to
nothing, because it no longer needs the room.

Note that widgets owning their own text already scroll sideways
themselves: `TextArea` with wrap off has a real bar, and `TextField`,
`SecretField` and `SecretArea` slide their contents under the caret. This
is only about *containers*.

## Chrome that lines up with a pane

**Obvious:** measure the splitter after a layout and tell the header bar.

**That is always a frame late.** The caption is laid out *before* the
content, so a divider position discovered during content layout reaches
the chrome one frame after it changed — and on a drag that is a frame the
user watches.

**Instead**, let the drag tell you, and give the bar a number:

```go
split.OnRatioChanged = func(paneA, ratio float32) { head.StartWidth = paneA }
head.StartWidth = /* whatever it starts at */
```

`OnRatioChanged` fires on the **drag**, not the layout, so the next pass
already has the number. Nothing is bound to anything: a footer, a second
tool bar or a status bar can read the same value, and neither widget
learns about the other.

A sidebar with a header of its own does not need any of this — put a
`HeaderBar` at the top of the pane. This is only for lining the *window's*
chrome up with a pane below it.

Note that a full-height sidebar running beside the title bar needs
client-side decorations: under a desktop-drawn frame the title bar is not
the toolkit's to lay out. The usual pattern — and what Safari, Chrome, VS
Code and every web dashboard do — is a full-width title bar with the
sidebar starting underneath it, which needs none of that.

## Buttons in the title bar

`Window.SetCaptionButtonVisible` turns one of the window's own buttons
off for this window — a tool window with no maximize, a dialog with only
a close. It cannot add one the desktop cannot do: the look and the
compositor decide what is possible, and this picks from that.

`Window.SetCaptionActions` puts the application's own buttons up there,
each saying which side it belongs on:

```go
w.SetCaptionActions(
    widgets.CaptionAction{Icon: style.IconUser, Name: "Profile", OnClick: showProfile},
    widgets.CaptionAction{Icon: style.IconMore, Name: "More", Lead: true, OnClick: showMenu},
)
```

They are drawn on the era's **tool** face, not its window-control face.
A look draws a window control's shape and its glyph as one piece, keyed
on which control it is, so there is no way to borrow a close button's
shape and put a different mark in it — 67 of the packs draw a glyph of
their own for anything they do not recognise. The tool face is the era's
own button for a mark, and every pack has one.

## An application whose chrome is its own — a browser, an editor

This is the second of the three ways to build with this toolkit, and it is
opted into a call at a time: a themed application that writes none of this
keeps the look's own chrome, and a skin answers the question a third way
([building.md](building.md#which-kind-of-application-are-you-building)).

The shape a browser or a code editor wears is not a special case the
toolkit knows about. It is five ordinary pieces, and every one of them is
a thing any application can ask for. `examples/uitoolkit-sample-look-like-chromium`
is the whole of it in one file.

**1. Your title bar *is* the caption.** A classic look would put its own
title strip above yours; say that your bar is the caption and it will not,
under any pack:

```go
win.SetTitleBar(tabStrip())           // a HeaderBar holding BrowserTabs
win.SetCaptionStyle(style.CaptionMerged)
win.SetBorderless(true)               // your chrome runs to the window's edge
```

**2. Paint your own chrome.** A browser tints its tab strip and its tool
bar; it does not wear the pack's neutral grey. State the two surfaces by
name — the same names a pack's `theme.json` uses, so the engine was
already reading them:

```go
uitoolkit.New(uitoolkit.Options{
    Chrome: map[string]paintengine2d.Color{
        "titleBar": accent,
        "toolBar":  style.Mix(accent, paintengine2d.RGB(1, 1, 1), 0.55),
    },
})
```

A tint carries no ink, so keep a tinted surface inside the contrast its
palette was built for.

**3. The row under the tabs is a `ToolBar`, not a `Row`.** This is the one
that is easy to get wrong and hard to see: a look paints its tool bar as
its own surface, and the engine fills the *selected tab* with that same
colour so the two meet and read as one. On a plain `Row` there is no tool
bar surface, the tab has nothing to merge into, and the window reads as
a page with tabs floating above it.

**4. One control takes the spare width.** `ToolStretch` is blank space;
`ToolGrow` is the control that grows:

```go
widgets.NewToolBar(
    widgets.ToolIconBtn(style.IconArrowLeft, "", onBack),
    widgets.ToolIconBtn(style.IconArrowRight, "", onForward),
    widgets.ToolIconBtn(style.IconSync, "", onReload),
    widgets.ToolGrow(omnibox),
    widgets.ToolIconBtn(style.IconArchive, "", onExtensions),
    widgets.ToolWidget(menu),
)
```

**5. Marks inside the address bar, and marks with no frame.** A browser's
site-information lock and its bookmark star are *inside* the field, not
beside it, and none of its marks has a frame until you point at it:

```go
addr := widgets.NewTextField(url, "Search or enter address", nil)
lock := widgets.NewIconButton(style.IconLock, "View site information", onInfo)
star := widgets.NewIconButton(style.IconStar, "Bookmark this tab", onStar)
lock.Flat, star.Flat = true, true     // no frame until pointed at
omnibox := widgets.NewFieldBox(
    []widget.Component{lock}, addr, []widget.Component{star})
```

`FieldBox` draws the well once, around everything, and takes the focus
ring from the field inside it — one control to look at, not three.
`IconButton.Flat` is the same control as an ordinary one, with the tool
face instead of the button face; `MenuButton` inherits it, which is what
a three-dot menu wants.

**What the pack still decides.** The tab's shape, the caption buttons'
size and side, the radius: those are the look's, and they differ by era on
purpose. `style.ThemePackAvailable` tells you whether the pack you pinned
is even in your build ([engines.md](engines.md)), and anything the toolkit
does other than what you asked is on the log and in
`Application.Diagnostics()` ([diagnostics.md](diagnostics.md)).

## A status colour as text

**Obvious:** draw the word in `Palette().Danger`.

**Wrong on most packs.** The declared status colours are era-faithful on
purpose — Clearlooks really did use `#c4a000` — and most of their uses
are fills, where the chrome around them carries the contrast. As *text*
on the window's own background, 225 of the 405 pairs across the shipped
packs do not reach 4.5:1, the worst near 1.6:1.

**Instead:**

```go
lbl := NewLabel("could not reach the server")
lbl.Tone = widgets.ToneDanger        // resolved at paint time, follows the theme
```

or, painting yourself, `lk.Palette().DangerInk()` — the same colour moved
along its own lightness until it reads, hue intact. `Palette.Ink(c)` does
it for any colour; `ReadableInk(c, bg)` for any background.

Keep the declared colour for fills. Use the ink for text and icons.

---

## A check that has to ask a server

**Obvious:** do the round trip inside `MessageBoxInput.Validate`.

**Wrong** — `Validate` runs on the UI goroutine, so the interface stops
until the server answers.

**Instead** use `ValidateAsync`:

```go
Input: &widgets.MessageBoxInput{
    Text:        current,
    AcceptLabel: "Rename",
    ValidateAsync: func(name string, done func(error)) {
        go func() {
            err := server.Rename(name)      // whatever takes time
            app.Post(func() { done(err) })  // back on the UI goroutine
        }()
    },
},
```

The dialog stays up with the accepting button busy, ignoring further
presses, until `done`. `done(nil)` closes it with its accepting result so
`OnResult` runs; `done(err)` puts the reason under the field and
re-enables the button, with what the user typed still there.

`Validate` is still the right thing for a check you can make at once —
empty, malformed, already in a list you hold.

---

## Closing a dialog from your own code

**Obvious:** `widget.DismissOverlay(mb.Overlay())`.

**That takes it off the screen without finishing it** — no result is
recorded and `OnResult` never runs, so whatever was waiting on the answer
waits forever.

**Instead:** `mb.Close(widgets.ResultCancel)`. It does exactly what
pressing that button would.

---

## A dialog raised from inside a dialog

Overlays stack. `ShowOverlay` pushes, `DismissOverlay` pops the one your
component is in, and the dialog underneath stays up with its fields as
they were. Only the top one takes the keyboard and the pointer.

- from **inside** a dialog, closing yourself: `DismissOverlay(from)`, or
  `MessageBox.Close`;
- from the window's **content**, taking down whatever is up:
  `DismissOverlay(w.Content())` closes the top one;
- closing **everything**: `Window.SetOverlay(nil)`.

---

## Caps Lock, and other window-wide state

**Obvious:** `Window.OnLockKeys`.

**It is one callback a window.** A widget that takes it has taken it from
everyone, and two widgets cannot both follow it.

**Instead** implement `widget.LockKeysWatcher` and every component under
the window's roots hears every change — the same shape as
`widget.FocusWatcher`. A focused `SecretField` already draws its own Caps
Lock mark with no application code at all.

`OnLockKeys` remains for the window's own code.

---

## Following the window's focus

**Obvious:** `Window.Active()` on every paint.

**That is a check per frame for an event that happens twice a minute**, and
it is only right for as long as the toolkit happens to repaint on the change.

**Instead** `Window.OnActiveChange(func(active bool))`, which is called when
the window takes or loses the keyboard focus and only when that changes — so
an application does not have to remember what it was last told. It is what a
tray icon that marks unread mail until the window is looked at is built from
(Qt's `QEvent::WindowActivate`, GTK's `notify::is-active`, Win32's
`WM_ACTIVATE`).

It is one callback a window, like `OnMove` and `OnLockKeys`: the window's own
code. Installing it reports nothing by itself — the first call is the first
*change* after it — so ask `Window.Active()` for where things stand.

---

## Anything at all in a list, a cell or a menu

Reach for the icon fields rather than characters — `TableColumn.Icon`,
`TableView.CellIcon`, `TreeNode.Icon`, `MenuItem.Icon`, `ToolItem.Icon`
— and `style.IconByStem` for the wider shipped vocabulary when no typed
id fits. Four runes (`★ 📎 ● 🔇`) are drawn from paths and are safe as
characters; nothing else is. [widgets.md](widgets.md#icons-and-when-not-to-use-text)
has the list and [contracts.md](contracts.md) has the rule.
