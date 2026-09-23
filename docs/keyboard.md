# Keyboard map

Focus and shortcuts for uitoolkit **v0.10.13**. Esc always tears down
floating chrome in one order, everywhere: **tooltip → popup → overlay**.

## Global

| Key | Action |
| --- | --- |
| **Tab** / **Shift+Tab** | Next / previous focusable (overlay and popup replace the tab cycle) |
| **Esc** | Hide tooltip; if none, dismiss popup (menus, combo, context menu); if none, dismiss overlay (dialog, message box, file picker) |
| **Alt+letter** | Open the MenuBar title whose mnemonic matches (underlined letter) |

Unhandled keys bubble from the focused widget to its ancestors (so a
NumberField still steps while its inner TextField has focus).

### Where the keyboard starts

A window focuses something when it first opens, so keys that bubble from
the focus reach somebody before the first click or Tab: the first control
in the tab order that a *click* would also focus. Chrome reached only with
Tab, F10 or a mnemonic — a menu bar, a tool bar, a tab strip — is skipped,
so a window holding nothing else opens with no focus at all; its
accelerators and mnemonics work anyway, since neither goes through the
focus. A modal overlay that is already up when the window opens is
searched instead of the content behind it.

Focus arrives the way a click's does rather than a Tab's, so a look that
shows its ring only after keyboard navigation draws none until the user
uses the keyboard, while a field that always shows its caret shows it.

An app overrules this in either of two ways: focus something itself before
the first frame (`Window.RequestFocus`), or name the component with
`Window.SetInitialFocus`. The window only chooses when nothing else has,
and only once — a later Escape that clears the focus leaves it cleared.

### Which field a shortcut reads

`widget.KeyEvent` carries both the key and the character it stands for:

- **`e.Key`** for anything the keyboard *does* — Escape, Tab, Return, the
  arrows, Home and End, the function keys. Those carry no `Rune`, so a
  table written over characters cannot fire on one by accident.
- **`e.Rune`** for a shortcut written as a letter, which is most of them:
  `e.Rune == 's' && e.Mods.Ctrl()` is Ctrl+S.

`Rune` is the key's identity as a character, not what typing it produces:
no modifier has been applied, so Shift+A and A are both `'a'` and the
shift is in `e.Mods`. Which layout the key belongs to is already settled —
the key labelled A on AZERTY is `KeyA` and its character is `'a'`
(`platform.KeyChar`).

Text being **typed** never arrives as a KeyEvent. It comes as characters
through `Component.TextInput`, one per character the layout, Shift, the
dead keys and the compose key actually produced — which is what a text
field reads and what an input method drives. A shortcut table must not be
written over `TextInput`, and a text field must not be written over
`KeyPress`.

## TextField

| Key | Action |
| --- | --- |
| Printable | Insert (rejected when `Accept` returns false) |
| Backspace / Delete | Delete backward / forward, or the selection |
| Left / Right | Move caret; **Ctrl** jumps words; **Shift** extends selection |
| Home / End | Line start / end |
| Ctrl+A / C / X / V | Select all, copy, cut, paste (OS CLIPBOARD on X11 and Wayland) |
| Middle-click | Paste PRIMARY (X11 / Wayland primary-selection) or the in-process buffer |
| IME | Preedit is underlined in the field; commit inserts the phrase. Esc cancels composition. On Wayland, Latin keys still emit `EventText` when text-input-v3 is idle (no preedit). |
| Password | Same keys; the field paints bullets. `NewPasswordField` / `TextField.Password`. |
| Return | `OnSubmit` |

## TextArea

Same as TextField, except **Return** inserts a newline (no `OnSubmit`).

| Key | Action |
| --- | --- |
| Up / Down | Previous / next visual line (wrap-aware); **Shift** extends |
| Home / End | Visual line start / end; **Ctrl** is document start / end |
| PageUp / PageDown | Jump by a viewport of lines |
| Wheel | Scroll; unwrapped areas also take horizontal wheel |

## RichText

The TextArea keys, plus:

| Key | Action |
| --- | --- |
| Ctrl+Up / Ctrl+Down | Previous / next paragraph |
| Ctrl+Backspace / Ctrl+Delete | Delete a word |
| Return | New paragraph (a heading is followed by body text; on an empty list item, up a level, then out of the list) |
| Ctrl+Z / Ctrl+Shift+Z, Ctrl+Y | Undo / redo |
| Ctrl+C / X / V, Ctrl+Shift+V | Copy, cut, paste with formatting; paste as plain text |
| Ctrl+B / I / U, Alt+Shift+5 | Bold, italic, underline, strikethrough |
| Ctrl+\\ | Clear formatting |
| Ctrl+Alt+1–6, Ctrl+Alt+0 | Heading 1–6, body text |
| Ctrl+Shift+8 / Ctrl+Shift+7 | Bulleted / numbered list |
| Ctrl+] / Ctrl+[ | List item a level in / out |
| Ctrl+Shift+L / E / R | Align left / centre / right |
| Ctrl+K | Link |
| Ctrl+Return, Ctrl+click | Follow the link at the caret / under the pointer |
| Menu, Shift+F10 | Context menu |

Tab leaves the editor, as in every field (Ctrl+] indents a list). The
format bar's buttons are walked with Left / Right and pressed with Space
or Return.

## MDIArea

| Key | Action |
| --- | --- |
| Ctrl+Tab / Ctrl+F6 | Next window (**Shift**: previous) |
| Ctrl+F4 / Ctrl+W | Close the active window (it may ask first) |
| Alt+- | The active window's menu (Restore, Move, Size, Minimize, Maximize, Close, Next) |
| Arrows, then Return / Escape | Move or size from the window menu (**Ctrl**: a pixel at a time); keep / put back |
| Return | On a minimised window that has the keyboard: restore it |

## Wizard

| Key | Action |
| --- | --- |
| Return | Next, or Finish on the last page (a text area keeps its Return) |
| Escape | Cancel |
| Alt+B / N / F / S / H | Back, Next (Apply), Finish, Skip, Help |

## NumberField / Spinner

Same as TextField, plus:

| Key | Action |
| --- | --- |
| Up / Down | Step by `Step` (**Shift** ×10) |
| PageUp / PageDown | Step ×10 |
| Home / End | Min / max |
| Wheel | Step while the spinner or field is targeted |

Non-numeric input is rejected. `Decimals == 0` rejects a decimal point.

## ListView / TableView / TreeView

| Key | Action |
| --- | --- |
| Up / Down | Move the selection |
| PageUp / PageDown | Jump by a viewport |
| Home / End | First / last row |
| Return / Space | Activate the selected row (`OnSelect`) |
| Left / Right (tree) | Collapse / expand, or move to the parent |

TableView: click a **sortable** header to sort (click again to reverse).
The widget reports `OnSort`; the app reorders `CellText`.

## Buttons, checkbox, switch, radio, slider, accordion

| Key | Action |
| --- | --- |
| Return / Space | Activate button, toggle checkbox or switch, select radio, toggle expander |
| Left / Right / Up / Down (slider) | Nudge; **Shift** is finer |
| Home / End (slider) | Min / max |
| Up / Down (radio group) | Previous / next exclusive choice |
| Left / Right (expander) | Collapse / expand |

## MenuBar, PopupMenu, ComboBox, ToolBar, TabBar

| Key | Action |
| --- | --- |
| Left / Right | Next menu title, tool button, or tab |
| Up / Down | Next popup row; Down/Space opens a ComboBox |
| Home / End | First / last enabled item |
| Return / Space | Activate |
| Esc | Close the open menu / combo (also handled globally) |
| Letter | Mnemonic or first-letter match inside a popup |

## Links

| Key | Action |
| --- | --- |
| Return / Space | Open the link (`LinkButton`) |

## Files

| Key | Action |
| --- | --- |
| **Alt+Left** / **Alt+Right** | Back / forward through the tab's folders (the mouse's thumb buttons and a sideways three-finger swipe do the same) |
| Return (in the listing) | Open the selected folder |
| **Ctrl+T** / **Ctrl+W** / **Ctrl+Tab** | New tab / close tab / next tab |

## Single-letter shortcuts in an application

An application may bind bare letters — the mail client
([comms-mail](https://github.com/codemodify/comms-mail)) binds the
Thunderbird set, the music player binds the transport — and the toolkit is
what makes that safe: a key event carries both `Key` (what the keyboard
does) and `Rune` (the character the key stands for), and a shortcut table
can ask the focused component what role it gives the accessibility tree
before dispatching a letter, so typing into a text field never fires one.

## FileDialog (stub)

Path field uses TextField keys. The listing is a TableView.
**Esc** or Cancel dismisses the overlay without picking.
Open/Save confirms the selected row or the path field.

## Out of scope (v0.1)

CJK candidate windows, INCR clipboard, and platform accelerators
beyond the in-window map above. X11 XIM compose / dead keys and
Wayland xkb compose plus text-input-v3 preedit are supported; see
[platform.md](platform.md).
