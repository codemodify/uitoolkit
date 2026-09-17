# a look's silhouette on real hardware (instance 19)

Nested KWin on the real GPU, `tools/e2e` instance 19, KWin ScreenShot2
captures, `UITK_DECORATIONS=client` throughout. Nothing was run against the
user's session. Instance stopped and `/run/user/1000/uitk-e2e-19*` removed
afterwards.

Build: `examples/skinshape` from this branch. Shots: `tools/e2e/19/shots/*.png`
and the two counting scripts `tools/e2e/19/stray*.py` (the rig directory is
gitignored).

The window under test is a 560×340 `skinshape` window over a 900×620 backdrop
of the same binary (`-mode backdrop`): a saturated orange test card with a
white grid, which counts the presses it receives.

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Deck's outline, Wayland | `run.sh 19 skinshape` | ok — `deck-wayland.png`, **0 / 0** stray |
| 2 | Deck's outline, X11 | `UITK_BACKEND=x11` | ok — `deck-x11.png`, **0 / 0** stray |
| 3 | Deck at 1.75 | `kscreen-doctor output.Virtual-0.scale.1.75` | ok — `deck-175.png`, **0 / 0** stray |
| 4 | Deck resized to 900×520 | `kwin.py` `frameGeometry` | ok — `deck-resized.png`, **0 / 0** stray |
| 5 | A press beside the body | `in.sh 19 move 368 450 click` | ok — the backdrop counted it and KWin raised the backdrop, `deck-click-through.png` |
| 6 | The disc takes its ink | click (420, 408) | ok — `disc 1`, `deck-clicks-in.png` |
| 7 | The disc refuses its corners | click (404, 392) and (436, 424) | ok — `panel 2`, same shot |
| 8 | The same two on X11 | `UITK_BACKEND=x11` | ok — `disc 1 · panel 1`, `deck-x11-clicks.png`; backdrop press counted, `deck-x11-through.png` |
| 9 | Maximize drops the shape | `d.setMaximize(true, true)` | ok — full screen, square, "This window is a rectangle", `deck-maximized2.png` |
| 10 | BeOS's tab, Wayland | `-theme beos` | ok — `beos-wayland.png`, tab 146×19, **0 / 0** stray |
| 11 | BeOS's tab, X11 | `UITK_BACKEND=x11 -theme beos` | ok — `beos-x11.png`, tab 146×19, **0 / 0** stray |
| 12 | BeOS's tab at 1.75 | as #3 | ok — `beos-175.png`, tab 274×38, **0 / 0** stray |
| 13 | Maximize drops BeOS's tab *and* its fitted caption | `setMaximize` | ok — full-width yellow band, buttons at the far right, `beos-maximized.png` |

No panics, no fatals and nothing from either app in `19/app.log`,
`19/backdrop.log` or `19/kwin.log` across the session.

## Counting stray pixels

The shapes work counted the pixels a silhouette leaves behind; the same count
here, in two directions, against a backdrop chosen so the two sides are
separable by colour alone:

- **inside** — pixels two or more pixels *inside* the silhouette that are the
  test card's: the backdrop showing through where the window is solid, which
  is what the multisampled dest-out stipple looked like.
- **outside** — pixels two or more pixels *outside* it, inside the window's
  own bounding box, that are the window's: paint surviving where the window
  is not.

The silhouette is rebuilt in the script from the skin's own numbers (a
full-width shoulder 46 tall with 24 px top corners over a body inset 18 with
22 px bottom corners) and grown and shrunk by the margin, so the antialiased
thread along the edge is never counted either way. For BeOS the tab's extent
is measured out of the screenshot instead, since it depends on the title.

Classifying a pixel needs care and the first two attempts got it wrong — the
grid crossings of the test card, and then the antialiased edges of the black
title on BeOS's yellow tab, both landed on the wrong side of a naive
red-versus-blue test and showed up as dozens of "strays" that were nothing of
the sort. Deck's shell, plate and ink are all cool (blue leads red in every
one of them) and the card is warm, so a pixel is the window's when blue leads
and the card's when red does, with neutrals — the black desktop, the card's
own grey label — counted as neither. BeOS is neutral grey and saturated
yellow against orange, 35 degrees of hue apart, so that one goes by hue.

With that: **zero in every direction, in every configuration above**. The
edge is one antialiased pixel wide and nothing survives on either side of it.

## Findings

**The silhouette is the compositor's, not a picture of one.** Clicking the
strip beside Deck's body did not merely miss the window: KWin treated it as a
press on the backdrop, counted it there, and *raised the backdrop over the
shaped window*. That is the whole claim in one screenshot pair
(`deck-wayland.png` → `deck-click-through.png`).

**X11 is the same as Wayland here.** Both backends produced identical stray
counts and an identical picture, through completely different mechanisms —
`wl_surface.set_input_region` against `XShapeCombineRectangles` on
`ShapeInput` and `ShapeBounding`. The X11 run went through the nested
Xwayland, so KWin was honouring an XShape region on a window it was also
compositing.

**1.75 is exact.** Deck's outline at 1.75 is the same outline, not a stretched
one: the shoulder's 24 px corner and the body's 18 px waist are redrawn at
42 and 31.5 device pixels because the manifest states them in design pixels
and the rect union is resolved against the window at the display scale. The
disc button is still a circle, because its caps are half the control height
at every scale. BeOS's tab goes 146×19 → 274×38 for the same reason one step
removed: its cell unit rounds to 2 device pixels at 1.75, so the tab is
nineteen cells tall either way and as wide as the bold title needs.

**Resizing keeps the shape.** Dragged from 560×340 to 900×520 through KWin's
scripting API, the window's outline followed exactly — still zero strays —
which is the property the rect-union form exists for. A mask-backed
silhouette (`"shape": {"art": …}`) would have resampled.

**Maximize is the clean case.** Both looks dropped their silhouette and
squared off, and BeOS dropped its *fitted caption* with it: the yellow tab
became a full-width band with the buttons at the far right, which is what a
window filling a box the desktop chose has to be. The app's own label flips
to "This window is a rectangle" because it asks `Window.ShapeActive()`, which
is the API an app that paints its own outline is supposed to ask.

## Gaps this run did not close

- **No interactive resize of a shaped window by dragging its edge.** The
  resize above went through KWin's scripting API. A shaped window's input
  region replaces the resize band in its shadow margin, so it is resized from
  inside its own edges; BeOS's five-pixel border has always worked that way
  (it casts no shadow), but Deck's 26 px bezel was not dragged by hand.
- **A silhouette under a server-side frame** was not exercised, because there
  is nothing to exercise: a look's outline is asked for only when the toolkit
  draws the frame.
- **Popups, menus and tooltips** are still in-window overlay layers, so a
  shaped skin's menu is a rectangle — the same gap docs/shapes.md already
  lists.
