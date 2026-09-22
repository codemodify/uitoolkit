# Popups as surfaces, the shaped band and real glass, on real hardware (instance 26)

Nested KWin 6 on the real GPU (Intel Arrow Lake, Mesa iris), `tools/e2e`
instance 26, KWin ScreenShot2 captures, Wayland and X11 (the instance's own
Xwayland). Nothing ran against the user's session; every app had its own
`XDG_CONFIG_HOME` under the instance, `kscreen-doctor` was pointed at the
instance's config, and the instance was stopped and
`/run/user/1000/uitk-e2e-26*` removed afterwards.

Build: `examples/popups`, `examples/minim`, `examples/shapes` from
`feat/shapes-2`, against paintengine2d `fix/shapes-2`. Shots are in
`tools/e2e/26/shots` (the rig directory is gitignored); `cN.png` are crops
stacked with `crop.py`.

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Minim's skin menu runs past its 275x116 strip, Wayland 1x | right-click on the face | ok — `m3.png`: the four rows hang below the strip; hover, Down, Down, Return picked "No skin" (`c1.png`) |
| 2 | The same at 1.75 | `kscreen-doctor ... scale.1.75` | ok — `c14.png`, `c15.png` (in the Minim skin) |
| 3 | The same on X11 at 1.75 | `UITK_BACKEND=x11 UITK_SCALE=1.75` | ok — `c17.png` |
| 4 | A menu bar menu with nested submenus past its 360x150 window, Wayland | click File, hover Recent, hover Older | ok — `p1.png`, `c2.png`, `c3.png` (three levels, each beside its row) |
| 5 | Switching menus by hovering the bar, and a click outside | hover Edit with File open; click the desktop | ok — `c3.png` (Edit opened from a hover, the old menus fading out); `WAYLAND_DEBUG` shows `xdg_popup.popup_done` and the menu went (`c4.png`) |
| 6 | A combo list past the window, keyboard | click the combo, Down, Down, Return | ok — `c4.png` (list to the screen's bottom, scrolling), `c5.png` (Option 03 picked) |
| 7 | Tooltip and a disc-shaped menu past the window | hover Tip; click Round | ok — `c5.png`: the tooltip runs past the window's edge; the Round menu is a disc, cut surface and all |
| 8 | xdg_popup in use | `WAYLAND_DEBUG=1` | ok — `create_positioner`, `set_anchor_rect`, `set_anchor(6)`, `set_gravity(8)`, `set_constraint_adjustment(41)`, `set_reactive`, `get_popup`, `grab(wl_seat, serial)`, `configure(...)`; the nested one `get_popup` from the menu's own `xdg_surface` |
| 9 | Wayland at 1.75: menu, submenu, combo | as #4, #6 | ok — `f1.png`, `c12.png`, `c13.png`; crisp, placed flush |
| 10 | X11 1x: menu, submenu, switch, combo, keys | `UITK_BACKEND=x11` | ok — `c6.png`, `c7.png` |
| 11 | X11 at 1.75 | as #10 | **found a bug, fixed**: every popup flipped above its window — KWin's Xwayland states `_NET_WORKAREA` in logical pixels (0, 0, 731, 491) while windows are placed in device ones. A work area under 60% of its monitor is now ignored for the monitor; after the fix `c16.png` |
| 12 | A shaped window resized from its band, Wayland | `shapes -mode ring -size 400`, drag from 1 px inside the left rim | ok — KWin geometry 400 → 450 wide (`s2.png`); before this work that press was the app's |
| 13 | Aero with real glass, Wayland | `UITK_DECORATIONS=client popups -theme aero` over `shapes -mode backdrop` | ok — `WAYLAND_DEBUG` shows `get_background_effect` and a blur region; the frame is translucent over the blurred card, Aero Basic beside it is opaque (`c10.png`) |
| 14 | Aero with real glass, X11 at 1.75 | as #13 with `UITK_BACKEND=x11` | ok — `_KDE_NET_WM_BLUR_BEHIND_REGION` set on the window; `c19.png` (active), `c18.png` (inactive, opaque as Windows 7's was) |
| 15 | A glass look's menu over the card | `popups -theme tahoe` | ok — `c11.png`: the menu is real blurred glass with rounded corners, past the window |

## Known on this hardware

- **X11 under Xwayland: a click on the bare desktop does not dismiss a
  menu.** An X client's pointer grab cannot see presses on surfaces that are
  not X windows; Qt and GTK have the same limit there. A click on any other
  X window, Escape, or a click in the app's own window does dismiss; on a
  real X server the grab sees every press.
- **The menu's shadow is hard to see on the rig's black desktop**; it is
  there (the headless tests measure it).
