# The skin gaps, closed, on real hardware (instance 25)

Nested KWin on the real GPU, `tools/e2e` instance 25, KWin ScreenShot2
captures, Wayland and X11. Nothing was run against the user's session: the
player, Settings and the backdrop each ran with their own `XDG_CONFIG_HOME`
under the instance directory, and `kscreen-doctor` was pointed at the
instance's own config. The instance was stopped and
`/run/user/1000/uitk-e2e-25*` removed afterwards.

Build: `examples/minim`, `examples/skinshape` and `cmd/uitksettings` from
`feat/skins-2`, against paintengine2d `fix/sampler-clamp`. Shots are in
`tools/e2e/25/shots`, and the three counters — `stray` (a rounded window over
the test card), `palette` (pixels that are not a colour of a pixel skin's
art) and `deckstray` (Settings' preview of Deck) — are small Go programs
beside them (the rig directory is gitignored).

The player's windows are shown over `skinshape -mode backdrop`, the orange
test card with a white grid, which counts the presses that reach it.

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Minim Silver's equaliser wears its own caption, Wayland 1× | `run.sh 25 minim -theme minim-silver`, windows laid side by side with `kwin.py` | ok — `silver-wl-1.png`: a tab reading MINIM EQUALISER, pale chrome, no navy band; the strip and playlist keep theirs. **0 / 0** stray on all three windows |
| 2 | The same at 1.75 | `kscreen-doctor output.Virtual-0.scale.1.75` | ok — `silver-wl-175.png`, 481 × 203 windows, **0 / 0** on all three |
| 3 | The same on X11, 1× | `UITK_BACKEND=x11` | ok — `silver-x11-1.png`, **0 / 0** on all three |
| 4 | A round key refuses its corners, Wayland | two clicks in the corners of Silver's stop key's box, then one in its middle | ok — the corners left the transport playing (`silver-stop-corner*.png`), the middle stopped it and lit the key's hover ring (`silver-stop-middle.png`) |
| 5 | The same on X11 | as #4 | ok — `silver-x11-corners.png` still playing, `silver-x11-middle.png` stopped |
| 6 | A window corner takes no press | a click on the strip's cut top-left corner, X11 | ok — the backdrop counted it (`silver-x11-windowcorner.png`, "backdrop presses: 1") |
| 7 | Minim Classic crisp at 1.75, X11 | `UITK_BACKEND=x11 minim -theme minim-classic -scale 1.75` on a 1× output | ok — `classic-x11-175.png`: **0 of 5 488** pixels of the transport row are anything but a colour of the skin's own art |
| 8 | The same on Wayland at 1.75 | output scale 1.75, GPU and `UITK_PAINT=cpu` | **not exact — the compositor's** — `classic-wl-175.png`, `classic-wl-175-cpu.png`: 978 of 5 488 blended, the same number with either painter (see below) |
| 9 | Settings previews Deck with its outline, Wayland 1.75 | `uitksettings -stage deck` | ok — `settings-deck-wl-175.png`: the shoulder over the waist, Settings' own background beside the body; **0** dark pixels three or more outside the outline (6 at two, all on the top-right corner's antialiased edge, against a box measured off the screenshot) |
| 10 | The same on X11, 1× | `UITK_BACKEND=x11` | ok — `settings-deck-x11.png`, **0** at two pixels and at three |

No panics and nothing unexpected in `25/app.log` or `25/backdrop.log`.

## Counting

Stray pixels are counted as in
[the shapes run](../2026-09-17/skin-shapes-realhw.md): **inside** is the
card showing two or more pixels inside the silhouette, **outside** is the
window's own paint two or more pixels outside it but inside its box. The
silhouette is rebuilt from the skin's numbers — the window's box with a
7-design-pixel radius on every corner — at the display scale. The strip's
mark sits on an orange lozenge on purpose and its box is left out; at 1.75
a first count put the box a pixel short and found 6 "strays" on the
lozenge's own antialiased rim, which the corrected box does not.

Crispness is counted as colours: every opaque pixel of Minim Classic's 1×
sheet is a colour of the art, and a pixel of its transport row that is not
one of them was blended rather than magnified.

## Findings

**The silver equaliser's tab is a caption, not a picture of one.** The
window's caption band is thirteen pixels tall, the tab is the title's plate
set from the start, and the press-and-drag, the double-click and the caption
buttons are the frame's as on the other two windows. KDE's window-menu key
sits at the band's left and the tab starts after it.

**The corners of a round key are the panel, not the key.** A press there
lands on the strip's face — which moves the window, as the face of a player
this size always has — and not on the key; the backdrop does not see it,
because the window is still there. The corner of the *window* is another
matter, and on X11 as on Wayland the backdrop took that press.

**Classic is whole pixels at 1.75 wherever the toolkit has the last word.**
On X11 the transport row at 1.75 is nothing but colours of the art. On
Wayland at 1.75 it is not, with the CPU painter as with the GPU one, and the
Wayland row is not the X11 row at any whole-pixel offset: KWin resamples the
surface. A Wayland surface's size is whole logical pixels, and Minim's
275-pixel window times 1.75 is 481¼ device pixels; the 481-pixel buffer is
stretched by that quarter pixel when it is composited. That softens a pixel
skin's edges by a pixel here and there and nothing more — but it is not the
toolkit's to fix inside the skin, and polish.md says so.

**Settings' preview is the skin's outline, drawn by the preview.** The
preview panel cuts itself, and its theme scope's background, to Deck's
silhouette, and keeps its body inside Deck's border: the menu bar and the
status bar stop short of the waist, and Settings' own background shows
beside it where the desktop would.
