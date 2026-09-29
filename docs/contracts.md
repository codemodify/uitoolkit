# The text contract

What this toolkit promises about text, what it does not, and the two
rules an application has to follow to stay on the right side of the
difference.

It is one page because the rules are one subject: **uitoolkit draws every
glyph itself**. There is no Pango, no HarfBuzz, no CoreText and no
browser underneath, which is what makes a window look the same on four
platforms and a screenshot test possible at all — and it is also why the
list below is as long as it is. Everything here follows from that one
decision.

[toolkits.md](toolkits.md) puts the same facts next to Qt, GTK, Avalonia
and the rest, for someone deciding whether to adopt this at all. This
page is for someone who already has.

---

## The two rules

### 1. A mark is an icon, not a character

`✓ ✗ → ★ 📎 ⌘ ●` are **icons**. Do not type them into a label, a button,
a menu item, a cell or a tooltip.

The interface face is Latin only and there is no font fallback, so a rune
it lacks draws as a box — on any machine, and on *every* machine under
`UITK_SYSTEM_FONTS=0`, which is what containers, CI and every screenshot
in this repository run under. An application that types its marks has a
UI that looks right for its author and wrong for everybody else.

Use `Button.Icon`, `ToolItem.Icon`, `MenuItem.Icon`,
`TableView.CellIcon`, `TableColumn.Icon`, `TreeNode.Icon`, or
`style.DrawToolIcon` where you are painting yourself. The full list of
ids, the wider stem vocabulary and the four runes that *are* safe are in
[Icons, and when not to use text](widgets.md#icons-and-when-not-to-use-text).

### 2. Text is text, not an icon

The converse, and the one that is easier to get wrong because nothing
draws a box to tell you.

**A word is not a picture.** An icon alone cannot be read by a screen
reader, cannot be translated, and cannot be searched. Every icon-only
control needs a name — `SetAccessibleName` — and the toolkit will not
invent one for you beyond the icon's own generic label ("Save", "Open"):

```go
more := widgets.ToolIconBtn(style.IconMore, "", onMore)
more.SetAccessibleName("More options")   // not "More"
```

`a11ytest.Audit` fails a tree with an unnamed interactive node, so a test
catches this rather than a user (see [accessibility.md](accessibility.md)).

**And a picture is not a word.** Do not render prose as an image to get
around the script limits below. It cannot be selected, copied, found,
scaled with the display or read aloud, and it will be wrong at a display
scale you did not test. If the toolkit cannot draw a script, the answer
is a different toolkit for that product — not a bitmap of the sentence.

---

## What is not supported

### No complex text shaping

The text stack shapes runs **rune by rune**, with pair kerning from
`GPOS` and nothing else. There is no `GSUB`. Concretely, none of this
happens:

| | |
| --- | --- |
| Arabic and Syriac **joining** | letters keep their isolated forms |
| Indic **reordering and conjuncts** | Devanagari, Bengali, Tamil, Thai |
| **Ligatures** of any kind | including `fi`, and a code font's `!=` |
| Contextual alternates, small caps, old-style figures | any `GSUB` feature |
| Combining marks **positioned** by the face | `GPOS` mark attachment |

### No bidirectional layout

There is no Unicode bidi algorithm. A string is laid out left to right in
**logical order**, so Arabic and Hebrew come out reversed on screen even
where the face can draw them.

### No right-to-left interface

Separately from the text: there is **no mirroring of the interface**. A
window is not flipped for an RTL locale — the menu bar starts on the
left, a check box's box is on the left of its label, a tree's disclosure
triangles are on the left, scroll bars are on the right, and a dialog's
buttons are in the order the look's own convention puts them. Nothing
reads a locale's direction, and there is no API to ask for the flip.

### No font fallback

**One face draws everything.** A rune the selected face lacks is a box —
the toolkit does not go looking for another font that has it, the way
every browser and every other toolkit does.

This is the rule that surprises people, because installed fonts *do*
matter: they decide **which single face is selected** for a pack, not
which glyph is used for a rune. Installing Noto does not rescue a `✓` in
a label set in Titillium Web.

A face that cannot draw Latin at all — Noto Sans Arabic, Noto Sans CJK,
Noto Color Emoji, and about thirty families the chooser offers on an
ordinary desktop — cannot draw the interface, so asking for one falls
back to the bundled face rather than failing.

### The bundled faces, exactly

Two faces ship with the toolkit, and they do **not** cover the same
ground. Measured, not assumed:

| | Titillium Web (`FamilyUI`) | JetBrains Mono (`FamilyMono`) |
| --- | --- | --- |
| Latin, accented Latin | yes | yes |
| Greek, Cyrillic | **no** | yes |
| CJK | **no** | **no** |
| `✓ ✗ → ← └` and the box drawing | **no** | yes |
| `★ 📎 ● 🔇` | drawn as paths (below) | drawn as paths |

Titillium Web has **456 glyphs**. It is the face every pack falls back
to, and it is the one that draws your labels, buttons and menus.

**Do not reach for the mono face to get a glyph.** It is a real
temptation — setting `Mono` on a label does make a `✓` appear — and it is
the rule in rule 2 broken from the other side: it changes the typeface of
a piece of text for the sake of one mark, so the label no longer matches
the interface around it, and it still fails for anything neither face
has. The icon is the answer; `Mono` is for code, paths and figures that
need to line up.

---

## What you can rely on

- **Latin and Western European text is right**, including the accented
  letters, and it is kerned.
- **A rune the face has is drawn correctly**, at any size, on every
  platform, identically. The same string is the same pixels on Wayland,
  X11, Windows and macOS — which is what makes the pixel tests in this
  repository possible.
- **Nothing crashes or refuses.** Arabic, Hebrew, CJK and emoji all
  *render*; they render as boxes, or unjoined and in logical order where
  the face has them, but a document containing them lays out, scrolls,
  selects, copies and saves without incident. Offsets, the caret and the
  selection are in runes throughout and stay correct.
- **A pack may ask for an installed face** that carries more than the
  bundled one, and where the desktop has it, it is used.

---

## If you need more than this

You need a toolkit with a shaping engine. Qt has HarfBuzz, GTK has Pango,
and both do bidi and RTL mirroring properly. That is a real answer and
this page would be dishonest without it:

- **an Arabic, Hebrew, Persian or Urdu interface** — Qt or GTK;
- **an Indic or Thai interface** — Qt or GTK;
- **a CJK interface** — possible here if the pack resolves an installed
  CJK face, but with no vertical text and no ruby;
- **anything that must mirror for a locale** — Qt or GTK.

Latin, Greek and Cyrillic *content* — as opposed to the interface — works
wherever the selected face carries it.

---

## Where this is enforced

The rules are not only prose. `style.ShapeCacheKeysForTest` and the
tests in `style/` pin what the bundled faces can do;
`style/symbolfallback_test.go` pins that the four drawn runes are exactly
four, so a fifth cannot be added without the documentation following;
`style/latinface_test.go` pins that a script-only face falls back rather
than panicking; and `a11ytest.Audit` fails an icon-only control with no
name.
