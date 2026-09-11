# Keyboard map

Focus and shortcuts for uitoolkit **v0.5.0**. Esc always tears down
floating chrome in one order, everywhere: **tooltip → popup → overlay**.

## Global

| Key | Action |
| --- | --- |
| **Tab** / **Shift+Tab** | Next / previous focusable (overlay and popup replace the tab cycle) |
| **Esc** | Hide tooltip; if none, dismiss popup (menus, combo, context menu); if none, dismiss overlay (dialog, message box, file picker) |
| **Alt+letter** | Open the MenuBar title whose mnemonic matches (underlined letter) |

Unhandled keys bubble from the focused widget to its ancestors (so a
NumberField still steps while its inner TextField has focus).

## TextField

| Key | Action |
| --- | --- |
| Printable | Insert (rejected when `Accept` returns false) |
| Backspace / Delete | Delete backward / forward, or the selection |
| Left / Right | Move caret; **Ctrl** jumps words; **Shift** extends selection |
| Home / End | Line start / end |
| Ctrl+A / C / X / V | Select all, copy, cut, paste (OS CLIPBOARD on X11 and Wayland) |
| Middle-click | Paste PRIMARY (X11 / Wayland primary-selection) or the in-process buffer |
| IME | Preedit is underlined in the field; commit inserts the phrase. Esc cancels composition. |
| Return | `OnSubmit` |

## TextArea

Same as TextField, except **Return** inserts a newline (no `OnSubmit`).

| Key | Action |
| --- | --- |
| Up / Down | Previous / next visual line (wrap-aware); **Shift** extends |
| Home / End | Visual line start / end; **Ctrl** is document start / end |
| PageUp / PageDown | Jump by a viewport of lines |
| Wheel | Scroll; unwrapped areas also take horizontal wheel |

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

## FileDialog (stub)

Path field uses TextField keys. The listing is a TableView.
**Esc** or Cancel dismisses the overlay without picking.
Open/Save confirms the selected row or the path field.

## Out of scope (v0.1)

CJK preedit / candidate windows, INCR clipboard, and platform
accelerators beyond the in-window map above. X11 XIM compose and
dead keys are supported; see [platform.md](platform.md).
