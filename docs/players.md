# The players — skins, shapes and racks at work

The skinned media player this toolkit was built against lives in
**[codemodify/media-player-music](https://github.com/codemodify/media-player-music)**
and builds on uitoolkit's published module. Its three faces — Minim the
compact strip, Marquee the cabinet that folds, Lantern the shaped window
with a menu bar — grew up here as `examples/minim`, `examples/marquee` and
`examples/lantern`; that code moved out, and the running instructions moved
with it.

This document is the toolkit's half: what a skinned, shaped, multi-window
application asks of uitoolkit, what uitoolkit gives it, and what writing one
changed here. The eight skins are still the toolkit's
([skins.md](skins.md)), so is the snapping (`rack`), so are the shaped
windows ([shapes.md](shapes.md)) — the player is what proves they work
together.

![Minim in its three skins — Minim, Minim Classic and Minim Silver — each as the strip, the equaliser and the playlist snapped together](screenshots/minim.webp)

| | |
|---|---|
| ![Marquee's cabinet, and folded into its stadium](screenshots/marquee.webp) | ![Lantern and its playlist in its skin, and with the skin dropped](screenshots/lantern.webp) |
| **Marquee** — the cabinet, and folded down | **Lantern** — in its skin, and with it dropped |

## It plays nothing, and the art is ours

**Nothing is decoded and nothing is played.** There is no audio stack, no
video stack and no media dependency of any kind in this toolkit, and there
is not going to be one — adding PipeWire, ALSA or FFmpeg to a UI toolkit to
make a demo honest would make the toolkit dishonest instead. The player says
so where it cannot be missed, in its title bar, on its display and in its
About box.

No skin, bitmap, icon, font, name or mark belonging to any real player is in
this repository, as art or as a fixture. The five skins the player wears —
`minim`, `minim-classic`, `minim-silver`, `marquee` and `lantern` — are
generated from paths and whole pixels in `skingen` like the other three, and
that generator is public API, so a skin for a player of somebody else's is
made the same way ([skins.md](skins.md#writing-one-in-go)).

## What the toolkit gives it

| | |
| --- | --- |
| a silhouette | a window shaped by its skin rather than by a rectangle, stated in **design pixels** and resolved against the window, so it is the same shape at 1, 1.25, 1.5, 1.75 and 2 ([shapes.md](shapes.md)) |
| a skin that lays out | **fixed layouts**: the skin states named slots and the app binds its controls to them with `widget.Slots`, carrying no rect of its own; the sprites it prints into — digits, lamps, a scrolling title, playlist rows — it paints by name with `style.DrawSkinSprite` ([skins.md](skins.md#fixed-layouts)) |
| a caption per window | a skin states a caption per **window role** (`app.Window.SetFrameRole`), so an equaliser can wear a tab for a header while the windows beside it keep the title band ([skins.md](skins.md#windows-that-are-not-alike)) |
| art on ordinary controls | `Painter` and `Shaper` on `widgets.Button` and `ToolButton`, `Painter`/`Travel` on `Slider`, `ListView.RowGeo` — a skinned control is a toolkit control wearing the slot's art, not a bespoke component ([widgets.md](widgets.md#art-on-a-control)) |
| a skin is a pack | one key drops it: the same widget tree, tab order and accessibility tree run in an ordinary theme, which is how a skinned app stays usable for someone who cannot use the skin |
| windows that snap | `rack`, below |
| a window sized by its design | `platform.WindowOptions` is logical pixels everywhere, with `Sizing`/`MaxWidth`/`MaxHeight` for a window whose size *is* its design |

## The one thing a desktop may refuse

The snapping is the toolkit's, not the player's: `rack` is a public
package any application can use for satellite windows of its own — see
[widgets.md](widgets.md#windows-that-snap-together).

Windows that stick to one another need two things of a desktop: being told
where a window is, and being able to put one somewhere. **X11 has both** —
clients place their own windows — and so does the offscreen backend, which
is why every snapping test runs without a compositor.

**A Wayland toplevel has no position.** A client is never told where its
windows are and cannot ask for one; that is the protocol, not an omission in
this toolkit. (Wayland's answer for the one case it cares about is
`xdg-toplevel-drag-v1`, which carries a window under the pointer *during a
drag*; the toolkit uses it for tear-off. It cannot place a window that is
not being dragged, which is what a rack is.)

So on Wayland the arithmetic still runs — the rack snaps, the model is the
same, `rack.Rack` is tested either way — and the windows simply cannot
follow. An application should say which of the two it got rather than pretend: the
player prints `places-windows=` on startup and says "playlist not placeable
here" in its status line.

## What is tested here

The toolkit's own suite covers the parts that stayed: `rack.Rack`'s
arithmetic and where a snapped window lands, the skins at every scale, the
slots and sprites, and the silhouettes — checked as **fractions of the
window** rather than as pixels or as a golden image, which is exactly what a
shape stated in design pixels buys, and which names the row that is wrong
instead of handing back two pictures. The player's own headless tests —
every control on the keyboard, `a11y.Check` in the skin and with it dropped,
the switch at every stop — moved with the player.

Real hardware, through the nested-KWin rig, is in
[docs/e2e/2026-09-17/players-realhw.md](e2e/2026-09-17/players-realhw.md),
and the five gaps below that were closed afterwards are checked on the same
rig in [docs/e2e/2026-09-21/toolkit-gaps.md](e2e/2026-09-21/toolkit-gaps.md).
Minim's three skins, the live switch and Minim Silver's round corners are
checked there too, on Wayland and X11 at 1 and 1.75, in
[docs/e2e/2026-09-21/minim-skins-realhw.md](e2e/2026-09-21/minim-skins-realhw.md),
and the skin gaps closed after that — the silver equaliser's own caption,
the round keys' corners, Classic at 1.75 and Settings' preview of Deck — in
[docs/e2e/2026-09-21/skins-2-realhw.md](e2e/2026-09-21/skins-2-realhw.md).
Those reports were written when the player was in this tree and name paths
such as `examples/minim`; they are dated records, kept as they were run.

## What they ran into

Things a skinned, shaped, multi-window app wanted and the toolkit could not
do, kept here because they are the useful output of writing one — the
reason the player earned its place in this repository, and the reason its
leaving costs the toolkit nothing. Most have
since been fixed in the toolkit and are struck through, here and under
[Fixed since](#fixed-since); what is not struck through still stands.

- ~~**A look states the caption's height and an app cannot ask for a
  shorter band.**~~ **Fixed:** `app.Window.SetCaptionHeight` asks for a
  caption of the app's own height in any look, and the caption's buttons
  shrink to stand in it. A caption stays at the design pixels its art was
  redrawn for.
- ~~**`window.shape` rects anchor to the top and the left only.**~~
  **Fixed:** `fromRight` and `fromBottom` pin a fixed-size rect to the far
  edges — a tab at the top right, a foot a fixed height up from the bottom.
- ~~**`window.border` is one inset for the whole window.**~~ **Fixed:**
  `"border": {"caption": […], "content": […]}` insets the caption band and
  the content under it differently, for a silhouette wide at the top and
  narrow below.
- ~~**A skin cannot ask for a gap between its caption band and the
  content.**~~ **Fixed:** `captionGap`. The window in two pieces with a
  slot of desktop between them is a `captionGap` and a silhouette that cuts
  it away.
- ~~**Which side the caption buttons sit on is the desktop's.**~~
  **Fixed:** a skin's `"buttons": "left" | "right"` moves the desktop's
  buttons to the side its art has room on. None of the shipped skins states one; their
  silhouettes keep out of both corners.

The panel skins ran into five more:

- ~~**A skin states one caption for every window it dresses.**~~ **Fixed:**
  window variants, chosen by the role the app gives a window — Minim
  Silver's equaliser has its tab for a header and no title band.
- ~~**A control's pointer shape is read from the part it paints, not from a
  picture an app paints for it.**~~ **Fixed:** `widget.ArtShape`, with
  `style.SkinSpriteShape` and `style.SkinSlotShape`: Silver's round keys
  refuse the corners of their boxes.
- ~~**Pixel art is only ever magnified by whole multiples**, so a pixelated
  sprite drawn unsliced into a box at 1.75 is drawn at 1× in the middle of
  it.~~ **Fixed:** a pixel sprite an app draws at its own size is magnified
  onto one device grid every piece shares, and the one-pixel margin the
  classic panel carried to get round it is gone. On Wayland at 1.75 the
  compositor still stretches a 275-pixel window by a quarter of a pixel
  ([docs/skins.md](skins.md#hidpi), rule 5).
- **A menu is drawn inside the window it opens from.** The strip is 116
  design pixels tall, so the skin menu is four rows and nothing else — no
  separator, no shortcut column — and the panel skins set 20-pixel menu
  rows so four fit without a scroll bar. A menu that could leave its
  window (a Wayland xdg_popup, an X11 override-redirect window) would lift
  that limit for every compact app.
- **A skin's caption shorter than 24 pixels was grown to 24**: its buttons
  had the caption's height, and a frame gives such a button at least that.
  Fixed in the toolkit — a short caption now gets square buttons stood in
  it — which also gives the `minim` skin the 18-pixel caption it
  always stated.

And three that writing the panels on the skins' own layouts closed:

- ~~**A skin cannot lay a panel out**, so an app had to carry one layout per
  face on rects it shared with the generator.~~ **Fixed:** fixed layouts
  ([docs/skins.md](skins.md#fixed-layouts)); the player binds its controls
  to slot names and carries no rects.
- ~~**Panel keys have no hover state.**~~ **Fixed:** a slot's art has states,
  Silver's keys have hover faces, Classic's — whose era had none — fall back
  to rest.
- ~~**The panels turned window titles upper-case**, which is what a screen
  reader then read.~~ **Fixed:** the titles are the player's in every face;
  capitals are each panel skin's `"case": "upper"`.

Two more are about *when* rather than *what*, and only real hardware showed
them (see the e2e report):

- **A window manager places a window when it maps it**, and the application
  is told afterwards — so a rack built when its windows are made is built
  around windows that are all still at the origin. `rack.Desk.Adopt` takes the
  real boxes in first, and an app attaches its panes once that answers yes.
- **A move is a request.** X11 answers one a frame or two later, so reading
  the position straight back says the window is where it was and looks
  exactly like a drag. `rack.Desk` waits for a move it asked for to arrive, and
  gives up after twenty-five looks so a window manager that refuses one —
  KWin clamps a window that would hang off the screen — is not argued with
  forever.

Three small things were added rather than worked around: `app.Window.Move`
and `CanMove` (Position's other half, over `platform.SurfaceMoves`),
`app.Window.SetSize` (Size's other half), and `platform.SurfaceMoves`
itself.

## Fixed since

What each of the five became, with the player work-around it retires:

- ~~**Nothing public to hang skin art on.** The skin runtime is all exported
  — `style.SkinLayout`, `DrawSkinSlot`, `SkinSlotShape`, `widget.Slots`,
  `SlotArtRect`, `ArtShape` — but the two fields that let `paintAsKey` put a
  panel's art on a key, `Painter` and `Shaper`, existed only on the player's
  own glyph button, so an app outside this repo could read every line of the
  panels and not reproduce them.~~ **Fixed:** `widgets.Button` and the
  new `widgets.ToolButton` carry both, `widgets.Slider` carries `Painter` and
  `Travel`, and `widgets.ListView.RowGeo` takes its rows, groove and row
  height from a layout's slots — the one thing that made
  the player's track list a bespoke list rather than a list view.
  [docs/widgets.md](widgets.md#art-on-a-control).
- ~~**A control tells the app about the app's own change.** `OnChange` fires
  whoever set the value, so a seek bar driven from the playback position
  re-enters the handler that answers the user, and a player needs a suppress
  flag.~~ **Fixed:**
  `OnChange` still fires on every change, and `OnInput` fires only for the
  user's: pointer, keyboard, or an accessibility action.
  [docs/widgets.md](widgets.md#onchange-and-oninput).
- ~~**`widget.KeyEvent` carries the key and not the character.** Text
  arrives as a separate event, so a shortcut table that reads `e.Rune`
  never fires.~~ **Fixed:** a key event carries both — `Key` for what the
  keyboard *does*, `Rune` for the character the key stands for, with no
  modifier folded in. Typing still arrives only as `TextInput`.
  [docs/keyboard.md](keyboard.md) says which of the two a shortcut reads.
- ~~**A window opens with nothing focused**, so keys that bubble from the
  focus reach nobody until the first Tab.~~ **Fixed:** a window focuses the
  first control a click would focus, at its first laid-out frame, unless
  the app placed focus itself or named one with `Window.SetInitialFocus`.
  Chrome reached only with Tab or a mnemonic is skipped.
- ~~**A mouse press does not bubble.** `Window.hit` finds the deepest
  component under the pointer and tells that one alone, so a press on a
  player's display did nothing until every part of the face that is *not*
  a control said so itself.~~ **Fixed:** a press
  walks up from the component under the pointer until one takes it, as a
  wheel notch and a key already did, and the contract for what "took it"
  means is written down in [docs/widgets.md](widgets.md). The taker becomes
  the pointer capture; focus still goes to what the press landed on.
  An app's own "this part of the face is a handle" flag is still the clearer
  way to say it, but it is no longer the only way.
- ~~**`platform.WindowOptions.Resizable` is declared and never read**, and
  there is no maximum size, so a player whose size *is* its design cannot
  say it is fixed. A press eight pixels below the strip's top edge lands in
  the frame's resize band and grew a 275×116 window to 275×308.~~
  **Fixed:** `Resizable` became `Sizing` (an enum whose zero value is
  resizable, beside `Decorations`) with `MaxWidth` and `MaxHeight`.
  `SizingFixed` states the same minimum and maximum to the desktop, refuses
  an interactive resize, drops maximize from the window's capabilities and
  leaves no resize band in the toolkit's own frame. A player whose size is its design can
  now say so.
- ~~**Window sizes are device pixels on X11 and logical pixels on
  Wayland**, and no scale factor reconciles them.~~ **Fixed:** window
  geometry is logical pixels everywhere and the X11 backend converts at its
  own boundary; `app.Window.PixelSize` is the device-pixel box for code
  that pairs a size with a widget rectangle. A design pixel *is* a logical
  pixel now.
- ~~**Window positions were still device pixels**~~ while sizes were
  logical. **Fixed:** `Position`, `Move` and `WindowOptions.X`/`Y` are
  logical too, so a rack's box is a position and a `Size` in one unit, and
  its reach (`rack.DefaultReach`) grows with the display without the apps
  scaling it. At a fractional scale on X11 two windows snapped flush can
  still meet a device pixel apart: each edge is rounded to the root's
  pixels on its own.
