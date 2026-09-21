# Minim's three skins on real hardware (instance 23)

Nested KWin on the real GPU, `tools/e2e` instance 23, KWin ScreenShot2
captures, Wayland and X11. Nothing was run against the user's session; the
player and the backdrop each ran with their own `XDG_CONFIG_HOME` under the
instance directory, and `kscreen-doctor` was pointed at the instance's own
config. The instance was stopped and `/run/user/1000/uitk-e2e-23*` removed
afterwards.

Build: `examples/minim` and `examples/skinshape` from `feat/minim-skins`.
Shots are in `tools/e2e/23/shots`, and the stray-pixel counter is
`tools/e2e/23/stray.py` (the rig directory is gitignored).

The three windows are shown over `skinshape -mode backdrop`, the orange test
card with a white grid, so the desktop beyond a cut corner is unmistakable.
On Wayland the player cannot place its windows, so they were laid side by
side over the card with KWin's scripting API; on X11 the player stacked
them itself and they were moved by moving the strip.

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Minim Silver's round corners, Wayland, 1× | `run.sh 23 minim -theme minim-silver` | ok — `silver-wl-row.png`, **0 / 0** stray on all three windows |
| 2 | The same at 1.75 | `kscreen-doctor output.Virtual-0.scale.1.75` | ok — `silver-wl-175.png`, 481 × 203 windows, corners redrawn at 12¼ px, **0 / 0** |
| 3 | Minim Silver on X11, 1× | `UITK_BACKEND=x11` | ok — `silver-x11.png`, the three stacked flush by the player, **0 / 0** |
| 4 | Minim Silver on X11, 1.75 | `-scale 1.75` | ok — `silver-x11-175.png`, **0 / 0** |
| 5 | The skin key switches all three windows live, Wayland | a click on the SKIN column | ok — silver → minim in all three at once, `wl-after-skinkey.png` |
| 6 | Ctrl+K, Wayland | `key+ 29 key 37 key- 29` | ok — minim → minim-classic in all three, `wl-after-ctrlk.png`; stacked in `switch-wl.png` |
| 7 | The skin menu, Wayland | a right-click on the strip | ok after a fix (below) — four rows, the worn skin ticked, `wl-menu-z.png`; picking Minim Silver switched all three windows and they were still **0 / 0** (`wl-menu-picked.png`) |
| 8 | The choice is remembered | relaunch with no `-theme` | ok — `23/cfg/uitoolkit/minim.json` said `minim-classic` and the player opened in it |
| 9 | The skin key, Ctrl+K and Ctrl+Shift+K, X11 | as #5 and #6, then Ctrl+Shift+K | ok — silver → minim → classic → breeze-night, the stack flush throughout, `switch-x11.png` |

No panics and nothing from either app in `23/app.log` or `23/backdrop.log`
across the session.

## Counting stray pixels

The same two counts as [the shapes run](../2026-09-17/skin-shapes-realhw.md):
**inside** is the card showing two or more pixels inside the silhouette, and
**outside** is the window's own paint two or more pixels outside it but
inside its box. The silhouette is rebuilt from the skin's numbers — the
window's box with a 7-design-pixel radius on every corner — and grown and
shrunk by the margin, so the antialiased edge is never counted either way.

Silver's chrome, navy and blues are all cool or neutral and the card is
warm, so a pixel is the card's when red leads blue by forty and the
window's when blue is at least red and it is not the black desktop. One
thing on the window is warm on purpose — the orange lozenge the player's
mark sits on, at the strip's lower right — and its box is left out of the
count; without that the strip showed 136 "strays" inside, every one of them
the mark.

**Zero in both directions, on every window, in every configuration above**,
including straight after a live switch into the skin: a silhouette the look
declares is applied when the look changes, not only when a window opens.

## Found and fixed on the rig

- **A right-click on a key pressed it.** The first right-click, aimed at the
  strip, landed on the stop key and stopped the transport. GlyphButton and
  Fader took any mouse button; they take the primary one now, and a
  secondary click bubbles to the face and opens the skin menu wherever it
  lands.
- **The menu did not fit the strip.** A menu is drawn inside the window it
  opens from, and with a separator, a shortcut column and five rows it
  scrolled in the 116-pixel strip and cut "No skin (breeze-night)" to
  "No skin (br". It is four rows now, and the panel skins set 20-pixel menu
  rows so the four fit.

## Seen, and left

- **Wayland's caption carries the desktop's buttons**, here KDE's window-menu
  key at the top left beside minimize and close at the right. On Minim
  Silver the window-menu key sits inside the rounded top-left corner with a
  pixel or two to spare at both scales; it is whole.
- **KWin keeps a window on the screen.** At 1.75 on X11 the playlist would
  have hung off the bottom of the 1280 × 860 output, and KWin moved it up
  and across rather than place it flush; the stack is loose until it is
  dragged back, which is the behaviour docs/players.md already describes.
