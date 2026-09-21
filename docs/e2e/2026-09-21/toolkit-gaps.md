# the five toolkit gaps on real hardware (instance 21)

Nested KWin on the real GPU, `tools/e2e` instance 21, KWin ScreenShot2
captures, Wayland and X11. Nothing was run against the user's session. The
instance was stopped and `/run/user/1000/uitk-e2e-21*` removed afterwards.

Every check was run twice from the same rig: once against a binary built
from `dev` (82d2e92) and once from `fix/toolkit-gaps`, so each screenshot
is a pair and the difference is the fix and nothing else. Shots are in
`tools/e2e/21/shots` (the rig directory is gitignored).

Two of the five cannot be seen in a player, because the players work
around them: a shortcut written over `e.Rune`, and a window that opens
focused. Those were driven with a throwaway probe app — a text field, a
button, a shortcut table over the character, and a panel with one inert
coloured child on it — built twice against the two trees.

## A window whose size is its design refuses to be resized

`examples/minim -only strip`, the 275 × 116 compact player.

KWin's own window oracle, which is the whole point: a desktop that has
been told the window is fixed is the thing that refuses.

| | dev | fix/toolkit-gaps |
| --- | --- | --- |
| Wayland `resizeable` / `maximizable` | true / true | **false / false** |
| X11 `WM_NORMAL_HINTS` | minimum 275 × 116 | **minimum 275 × 116, maximum 275 × 116** |
| `w.setMaximize(true, true)` from a KWin script | grew to **1280 × 860** | stayed **275 × 116** |

`cmp-fixed.png` stacks the two: dev's strip stretched over the whole
screen, the fix's strip at the size it was drawn.

A pointer drag on the strip's edge moves it either way, because a shaped
window's input region is its silhouette and the resize band in the shadow
margin never received a press there. That is why the maximize path is the
one that shows it.

## A press reaches the container the child ignored

The probe's panel counts presses; the red box on it is an ordinary
component that ignores them.

| | dev | fix |
| --- | --- | --- |
| click the red box | `PRESS: (click the grey panel)` | `PRESS: container got 1` |

`cmp-probe-wl.png`, `cmp-probe-x11.png` — same on both backends.

## A shortcut written over the character fires

The probe's content root has `if e.Rune == 'k'`. Focus is moved off the
text field with Tab first, so the key is not text being typed.

| | dev | fix |
| --- | --- | --- |
| press K | `RUNE: (press K)` | `RUNE: 'k' fired 1` |

## A window opens with the keyboard somewhere

Two characters are typed into the probe with no click and no Tab.

| | dev | fix |
| --- | --- | --- |
| startup log | `focus=<nil>` | `focus=*widgets.TextField` |
| after typing `h` `i` | `FOCUS: typed k` — only the key *after* Tab landed | `FOCUS: typed hi` |

## The same size gives the same window at 1 and 1.75

A probe that opens one `WindowOptions{Width: 400, Height: 240}` window,
measured as KWin's `clientGeometry` (logical on Wayland, device on X11).

| backend | scale | dev | fix |
| --- | --- | --- | --- |
| Wayland (output scale 1) | 1 | 400 × 240 | 400 × 240 |
| Wayland (output scale 1.75, `kscreen-doctor`) | 1.75 | 400 × 240 logical = 700 × 420 device | same |
| X11 | 1 | 400 × 240 | 400 × 240 |
| X11 (`UITK_SCALE=1.75`) | 1.75 | **400 × 240** — a 1× window holding 1.75× content | **700 × 420** |

`cmp-scale-x11.png` stacks the X11 pair at 1.75: dev's window is the size
it would be at 1× with the text drawn at 1.75×, the fix's is the design at
1.75×.

## The demos

`examples/{minim,marquee,lantern,gallery}` were opened on both backends
and come up at the same geometry on each: Minim's three windows all
`resizeable=false`, Marquee, Lantern and the gallery resizable. Marquee's
Ctrl+M folds 780 × 540 → 468 × 104 and back on both backends; Lantern's
Ctrl+K drops the skin to a rectangular themed app with its playlist window
intact (`wayland-lantern-unskinned.png`).

## Reproducing it

```bash
cd tools/e2e && ./build.sh && ./start.sh 21
go build -o /tmp/bin/minim ./examples/minim
./run.sh 21 /tmp/bin/minim -only strip
./kwin.py 21 'var w=(workspace.windowList())[0]; OUT(w.frameGeometry.width + "x" +
  w.frameGeometry.height + " resizeable=" + w.resizeable); w.setMaximize(true,true);'
UITK_BACKEND=x11 ./run.sh 21 /tmp/bin/minim -only strip
DISPLAY=$(cat 21/display) xprop -id $(DISPLAY=$(cat 21/display) xprop -root \
  _NET_CLIENT_LIST | grep -oE '0x[0-9a-f]+' | head -1) WM_NORMAL_HINTS
./stop.sh 21
```
