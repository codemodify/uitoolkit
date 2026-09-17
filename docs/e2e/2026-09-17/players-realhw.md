# the three players on real hardware (instance 20)

Nested KWin on the real GPU, `tools/e2e` instance 20, KWin ScreenShot2
captures, Wayland and X11. Nothing was run against the user's session. The
instance was stopped and `/run/user/1000/uitk-e2e-20*` removed afterwards.

Builds: `examples/{minim,marquee,lantern}` and `examples/skinshape -mode
backdrop` from this branch. Shots: `tools/e2e/20/shots/*.png` and the
counting script `tools/e2e/20/stray.py` (the rig directory is gitignored).

The backdrop is the same saturated orange test card with a white grid that
the shapes work used, and it counts the presses it receives — which is half
the evidence here: a press that lands on it is a press the shaped window
above it did not take.

The virtual screen was 1400×900 for the first half and 1600×1000 for the
second, because the cabinet at 1.75 is 1365 by 945 device pixels and does
not fit on the first.

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Minim's three windows, Wayland | `launch.sh minim` | shaped; KWin cascaded them and the app said so (`places-windows=false`), `minim-wayland.png` |
| 2 | The same three stacked | `kwin.py` set each `frameGeometry` | **0 / 0** stray, `minim-wayland-stacked.png` |
| 3 | A press in the chin's cut | `in.sh 20 move 428 342 click` | the backdrop counted it (0 → 1), `minim-click-through.png` |
| 4 | A press on a control inside | `move 456 316 click` (the play button) | the backdrop did **not** move (2 → 2) and the strip played, `minim-inside-2.png` |
| 5 | A press in the cut again | `move 432 342 click` | the backdrop counted it (2 → 3), `minim-through-2.png` |
| 6 | Minim, X11 | `UITK_BACKEND=x11` | `places-windows=true`; the stack built itself flush at 562,392 / 562,508 / 562,624 |
| 7 | The stack travels with the strip | drag the strip's display from 700,432 to 558,240 | all three moved together to 420,200 / 420,316 / 420,432, **0 / 0** stray, `minim-x11-stack.png` |
| 8 | Minim at 1.75 | `kscreen-doctor output.Virtual-0.scale.1.75` | 481×203 windows, chin in proportion, **0 / 0** stray, `minim-175.png` |
| 9 | Marquee's cabinet, Wayland | `launch.sh marquee` | brow and dome both cut, **0 / 0** stray, `marquee-wayland.png` |
| 10 | It folds | `key+ 29 key 50 key- 29` (Ctrl+M) | 780×540 → 468×104 and the outline became the app's own stadium, **0 / 0** stray, `marquee-compact.png` |
| 11 | A press in the stadium's corner | `move 316 186 click` | the backdrop counted it (0 → 1), `marquee-through.png` |
| 12 | It unfolds | Ctrl+M again | 780×540 and the *skin's* brow and dome back, **0 / 0** stray, `marquee-unfolded.png` |
| 13 | The cabinet at 1.75 | as #8 | 1365×945, **0 / 0** stray, `marquee-175.png` |
| 14 | Marquee on X11, both modes | `UITK_BACKEND=x11` | **0 / 0** stray each, `marquee-x11.png`, `marquee-x11-compact.png` |
| 15 | Lantern and its anchored playlist, X11 | `UITK_BACKEND=x11` | player 120,200,880,560 and playlist 1000,200,380,560 — flush against its right edge — **0 / 0** stray on both, `lantern-x11.png` |
| 16 | The skin dropped | `key+ 29 key 37 key- 29` (Ctrl+K) | both windows rectangles in breeze-night, the whole app intact, the status line reading `theme breeze-night · rectangle · playlist right`, **0 / 0** against a rectangle, `lantern-themed.png` |
| 17 | A press in the swept corner | `move 128 752 click` | the backdrop counted it (0 → 1), `lantern-through.png` |
| 18 | Lantern at 1.75 | as #8 | 1540×980, **0 / 0** stray, `lantern-175.png` |

No panics, no fatals and nothing from any of the four applications in their
logs across the session. KWin's own log has the usual xkbcomp and
xdg-desktop-portal warnings and nothing else.

## Counting stray pixels

Two directions, as the shapes work counted them:

- **inside** — pixels two or more pixels *inside* the silhouette that are the
  test card's: the desktop showing through where the window is solid.
- **outside** — pixels two or more pixels *outside* it, inside the window's
  own bounding box, that are the window's: paint surviving where the window
  is not.

The silhouette is rebuilt in the script from **the skin's own manifest** — a
union of rounded rectangles in design pixels, scaled to the window's real
size — so it is an independent model of the outline rather than a reading of
the toolkit's rasteriser. Growing and shrinking it by the margin is what
keeps the one antialiased pixel along the edge out of both counts.

Classifying a pixel needs care and it is where the shapes work went wrong
twice. Here it is straightforward, because the test card and all three
players are on opposite sides of one axis: the card is saturated orange and
its grid lines and label plate are washes of the same hue, so red leads in
every one of them, while all three players are cool — green leads, or the
pixel is near-neutral graphite. So a pixel is the card's when red leads green
by a clear margin, the window's when green leads red at all, and neither
otherwise, which is what the black desktop and the card's own grey label are.

**The counter is checked against itself.** Every shape has a `rect-` twin
that claims the same window is a plain rectangle, and running that reports
the pixels the real outline cuts away: 448 for Minim's strip, 1635 for
Marquee's cabinet at 1×, 4565 at 1.75, 1331 for Lantern's. A counter that
answered zero for both would be a counter looking at nothing.

One thing had to be added to it, and it is a fact about unions rather than
about these windows. Where one rect of a union ends and the next begins — a
brow meeting a body, a chin stepping in — a device row can be half covered by
each, and which side of half it lands on is the rasteriser's business. The
first run of Marquee at 1.75 reported 168 strays, all of them on one row
(device row 10, where the brow's rect ends at design row 6 × 1.75 = 10.5).
The two-pixel margin now applies down the window as well as across it, and
that row is not counted either way.

With that: **zero in every direction, in every configuration above.**

## Findings

**Wayland places windows, and the players say so.** Three windows that snap
to one another need two things of a desktop — being told where a window is,
and being able to put one somewhere — and a Wayland toplevel has neither.
The arithmetic still runs (the rack is tested headless either way); the
windows simply cannot follow, `places-windows=false` in the startup line, and
Minim's own status says it. On X11 the same binary built its stack and
carried it.

**A window manager places a window when it maps it, and the app is told
afterwards.** The rack was originally built when the windows were made, which
put every box at the origin, and the stack it arranged was arranged around a
strip that was not there yet. `players.Desk.Adopt` takes the real boxes into
the rack first, and each player attaches its panes once that answers yes.
This is the first thing the rig found that headless testing could not.

**A move is a request, and X11 answers it a frame or two later.** Without
hysteresis, `Follow` read the old position back on the very next look,
decided the user must have dragged the window there, and took it straight out
of the stack it had just been put into. `Desk` now waits for a move it asked
for to arrive (and gives up after twenty-five looks, so a window manager that
refuses is not argued with forever). A backend that places a window as the
call is made — the offscreen one — is given no patience at all, because
patience there would make a real drag half a second later look like a move
nobody asked for.

**KWin keeps a window on the screen.** Lantern's playlist anchored to the
right of a player that KWin had placed at x=360 would have hung 20 pixels off
a 1600-pixel screen, and the move was clamped. The anchor is now asked for on
each tick until the desktop agrees or twenty ticks have passed; moving the
player left made it take. A playlist that ends up loose is not broken —
dragging it back to the player's edge snaps it, which is the path a person
takes anyway.

**Window sizes are device pixels on X11 and logical pixels on Wayland**, and
no scale factor reconciles the two: an X11 server has no notion of a display
scale, so a 275-design-pixel strip at 1.75 asks for 481, while a Wayland
toplevel's geometry is logical and the compositor multiplies, so the same
strip asks for 275. Asking the wrong way round opened the cabinet at 1365
*logical* pixels — nearly twice the screen. `players.WindowSize` is the
work-around; the fix belongs in `platform`.

**A press eight pixels below the strip's top edge resized it** rather than
moving it: the toolkit's frame keeps a resize band at the window's own edge,
and `platform.WindowOptions.Resizable` is declared and never read, so a
player whose size *is* its design cannot say it is fixed. The strip grew from
275×116 to 275×308 on the first attempt at dragging it.

**The toolkit does not bubble a mouse press.** `Window.hit` finds the deepest
component under the pointer and that one alone is told, so a press on a
player's display did nothing at all until every part of the face that is not
a control said so itself (`players.DragsWindow`). It is the right design —
bubbling presses is how a click ends up doing two things — but it is worth a
line in the widget docs, because "the container will get it" is the natural
assumption.

## Reproducing it

```bash
cd tools/e2e && ./build.sh && ./start.sh 20 1600 1000
go build -o /tmp/bin/minim ./examples/minim          # and marquee, lantern, skinshape
./run.sh 20 /tmp/bin/skinshape -mode backdrop
# the players need a second process, which run.sh does not track; the
# session used its own pid-tracked launcher for that.
./shot.sh 20 name
python3 20/stray.py 20/shots/name.png X,Y,W,H:minim-strip
./stop.sh 20 && rm -f /run/user/1000/uitk-e2e-20*
```
