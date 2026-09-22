# Rich text, MDI and wizard on real hardware — 2026-09-22

The three widgets of `feat/widgets-breadth`, driven in the nested-KWin rig
(`tools/e2e`, instance 28) on the real GPU, on the Wayland backend and on
X11 (the instance's Xwayland, `UITK_BACKEND=x11`), at scale 1 and at a real
output scale of 1.75 (`kscreen-doctor output.Virtual-0.scale.1.75` on a
2240 × 1505 virtual screen). The apps were `examples/richtext`,
`examples/mdi`, `examples/wizard` and `examples/gallery` in Adwaita 48.
Stills are in [widgets-breadth/](widgets-breadth/).

## Rich text

| what | Wayland | X11 |
| --- | --- | --- |
| Type a line, Shift+Home, Ctrl+B, Ctrl+I | [rt-typed](widgets-breadth/rt-typed.webp) | — |
| Click into a heading and type | ✓ | ✓ |
| Ctrl+Shift+Left selects a word, Ctrl+U | — | [x11-rt-edit](widgets-breadth/x11-rt-edit-crop.webp) |
| Double-click a word, drag it to the end: moved | [rt-move](widgets-breadth/rt-move-crop.webp) | [x11-rt-edit](widgets-breadth/x11-rt-edit-crop.webp) |
| Ctrl+Z takes the move back; the bar's bullet button | [rt-undo-list](widgets-breadth/rt-undo-list-crop.webp) | — |
| Gallery tab at 1.75: click, End, Return, type | [gal-rt-typed](widgets-breadth/gal-rt-typed.webp) | — |

Found and fixed here: a drag of the editor's own selection **copied** on
Wayland — the compositor settles the action from the target's preference,
and the window prefers what the compositor last said (a copy), so the
source's preferred move never counted. The editor now says move for its
own selection (`RichText.DropActionFor`). A press 300 ms after a double
click is a third click, so the drag after it extended by paragraphs — as
intended; the script waits longer now.

## MDI area

| what | Wayland | X11 |
| --- | --- | --- |
| Drag a window by its caption; it stops with its caption in the area | [mdi-moved2](widgets-breadth/mdi-moved2.webp) | [x11-mdi](widgets-breadth/x11-mdi.webp) |
| Ctrl+Tab to the next document, the live Window menu | [mdi-ctrltab](widgets-breadth/mdi-ctrltab.webp) | — |
| Window ▸ Tile | [mdi-tiled](widgets-breadth/mdi-tiled.webp) | — |
| Ctrl+F6, then Ctrl+F4 on an edited document asks first; Escape keeps it | [mdi-close-ask](widgets-breadth/mdi-close-ask.webp) | — |
| Resize by an edge | ✓ (top edge) | — |
| Gallery tab at 1.75: drag a window | [gal-mdiwiz](widgets-breadth/gal-mdiwiz.webp) | — |

Found and fixed here: the example opened with the **bottom** window
active — the window's first-focus rule focuses the first control in Tab
order, and focusing it raised its window. The area now puts the keyboard in
its active window when it arrives in a window, and `SetInitialFocus` names
the control (the document, not the format bar). At 1.75 the gallery's
windows were **tiny**: they were sized before the area was in a window, in
the fallback look at scale 1; they are placed on the area's first layout
now.

## Wizard

| what | Wayland | X11 |
| --- | --- | --- |
| Return on the welcome page moves on; Next greyed until the name is typed | ✓ | ✓ |
| Return on an invalid address shows the page's message | [wiz-3-invalid](widgets-breadth/wiz-3-invalid.webp) | [x11-wiz](widgets-breadth/x11-wiz-crop.webp) |
| Alt+S skips the optional page, Alt+B back, Space ticks, Alt+N on | [wiz-7](widgets-breadth/wiz-7-crop.webp) | — |
| Return, Return finishes (the example prints the account) | ✓ | — |
| Escape cancels (the example exits) | — | ✓ |

Found and fixed here: a page's content **filled the page**, so the proxy
page's one field stood as tall as the wizard; pages get their natural
height. **Alt+N typed an "n"** into the next page's field: the key's
character arrives as text after the access key has moved the focus there.
After an access key the wizard moves the focus once it has gone by. The
same can happen to any access key that moves the focus into a text field
(polish.md).

## Rig notes

- The **first pointer press after a window maps was lost twice** on
  Wayland (a click in the rich-text editor, a caption drag in the MDI
  example); the same press a second later worked, and so did every later
  one. Not seen on X11. A warm-up click avoids it in scripts; the cause is
  not established (polish.md).
- `type.py` in the instance directory turns a string into `in.sh` key
  codes (letters, digits, space, `,.-/`).
