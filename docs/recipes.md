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

| the control is | use | width |
| --- | --- | --- |
| a button with a name — OK, Save, Rename | `NewButton` + `Button.Icon` | wide (see below) |
| a button that is *only* a mark | `ToolIconBtn(icon, "", on)` | the mark |
| a row of marks — a tool bar | `NewToolBar(items...)` | the marks |
| a mark beside a name, outside a dialog | `NewToolButton(text, icon, on)` | tight |
| a mark in a menu row, a cell, a tree node | `MenuItem.Icon`, `TableView.CellIcon`, `TreeNode.Icon` | — |

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

- `ToolIconBtn(style.IconTrash, "", onDelete)` — an icon-only button,
  with `SetAccessibleName("Delete")` so it can still be read aloud;
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

There is **no minimum-size API and no horizontal scrolling**, so the
toolkit will not stop you reaching that point or let you scroll out of
it. What you can do:

- **Lower what the fixed children ask for.** `SecretField.PreferredWidth`,
  `SecretArea.PreferredWidth` and `TokenField.PreferredWidth` are that
  lever. A field that asks for 120 instead of 180 buys 60 px.
- **Fold the row** with `NewWrap`, so the button drops under the field
  instead of off the edge.
- **Use fewer columns.** A label above its field rather than beside it
  halves the floor.
- **Find the floor and design above it.** Lay the form out at decreasing
  widths in a test and assert nothing leaves the box — a minimum-size
  test is the only thing that catches this before a user does, and it
  catches it again when somebody adds an icon to a button.

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

## Anything at all in a list, a cell or a menu

Reach for the icon fields rather than characters — `TableColumn.Icon`,
`TableView.CellIcon`, `TreeNode.Icon`, `MenuItem.Icon`, `ToolItem.Icon`
— and `style.IconByStem` for the wider shipped vocabulary when no typed
id fits. Four runes (`★ 📎 ● 🔇`) are drawn from paths and are safe as
characters; nothing else is. [widgets.md](widgets.md#icons-and-when-not-to-use-text)
has the list and [contracts.md](contracts.md) has the rule.
