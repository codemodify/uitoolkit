# the six release bugs on real hardware (instance 24)

Nested KWin on the real GPU, `tools/e2e` instance 24: 1280×860 at scale 1,
then 2240×1505 with the output at 1.75 (`kscreen-doctor`, the instance's
own config dir) for Wayland and `UITK_SCALE=1.75` for X11. Every check ran
on a binary built from `dev` (257b470) and one from `fix/release-bugs`.
Nothing touched the user's session; the instance was stopped and
`/run/user/1000/uitk-e2e-24*` removed afterwards. Shots are in
`tools/e2e/24/shots` (gitignored).

| # | what | dev | fix/release-bugs |
| --- | --- | --- | --- |
| 1 | Tour: Frames → "The desktop's frame", then Skins → Windows 3.1 → "Use this look" | kept (the tour's own record worked round it) | kept, from `Application.Appearance`; Wayland 1 (`wl-skins-sys-after`), X11 1.75 (`x175-skins2-after`: asked for the desktop's frame, in effect server, pack win31) |
| 1 | Settings with `decorations: system` in look.json → Windows 98 → Apply | — | look.json still says `"decorations": "system"`, KWin's frame stays (`wl-settings-after`) |
| 2 | Outline floated (drag out, Float button), docked back by its ⧉ | at its size here; the stale arrangement shows only in a host arranged once (the unit test) | at its size on both backends and scales (`x-dockbtn`, `wl175-redock`) |
| 3 | A five-row list at its natural height in a scroll view, win95 | row 5 under the frame, a scroll bar for 4 px | whole, no scroll bar; Wayland 1 and 1.75, X11 1.75 (`wl-list-cmp`, `wl175-list-cmp`, `x175-list-cmp`) |
| 4 | Floating Outline dragged back by **its window's** title bar | desktop's frame: the window moves, nothing docks (`dev-kwincaption`) | the toolkit's frame; the caption carries the window and the drop docks it — Wayland 1 (`wl-capdrag`), Wayland 1.75 (`wl175-capdrag-log`, tabbed with Log), X11 1 (`x-capdrag`) |
| 5 | Tour tab torn off on X11 to 700,500 | window at 220,120 — pushed back on screen by KWin (`x-dev-tabtear`) | under the pointer at the grab offset, hanging off the screen's edge — scale 1 (`x-tabtear`) and 1.75 (`x175-tabtear`) |
| 6 | Positions logical | — | floating windows, the torn-off tab and the drop points line up at 1.75 on both backends (the shots above) |

## Found on the way

- **A drop on the dock host's centre is refused**, by design
  (`Host.targetAt`: only an edge band, a panel or its tab strip takes a
  panel). A first 1.75 attempt dropped there and looked like #4 again.
- **Switching an X11 window live to the desktop's frame shows no KWin
  title bar** under this nested Xwayland at 1.75 (the window reports
  `server` and loses its toolkit caption); the same on `dev`. Not
  investigated further.
- **The dock page's status note keeps saying "is in a window of its own"**
  after a panel is docked by dragging (only the buttons update it).
