# The three players

Three skinned, shaped media-player look-alikes: the toolkit's shop window
for [skins](skins.md) and [shaped windows](shapes.md), and the answer to
"what does a skinned app actually look like once it is a whole application
rather than a test card".

```bash
go run ./examples/minim      # the compact one: a strip, an equaliser, a playlist
go run ./examples/marquee    # the big one: a cabinet that folds down
go run ./examples/lantern    # the shaped one: a menu bar, and a switch
```

![Minim in its three skins — Minim, Minim Classic and Minim Silver — each as the strip, the equaliser and the playlist snapped together](screenshots/minim.webp)

| | |
|---|---|
| ![Marquee's cabinet, and folded into its stadium](screenshots/marquee.webp) | ![Lantern and its playlist in its skin, and with the skin dropped](screenshots/lantern.webp) |
| **Marquee** — the cabinet, and folded down | **Lantern** — in its skin, and with it dropped |

Each player's `-shot DIR` poses it at 1:37 and writes every window;
`tools/shots/demos.sh` puts those together as the pictures above.

## They play nothing

**Nothing here is decoded and nothing is played.** There is no audio stack,
no video stack and no media dependency of any kind in this toolkit, and
there is not going to be one — adding PipeWire, ALSA or FFmpeg to a UI
toolkit to make a demo honest would make the toolkit dishonest instead.

So the model underneath is a clock that counts, a list of invented tracks, a
band of numbers that move the way an analyser does, and an equaliser whose
ten faders filter nothing. Every one of the three says so where it cannot be
missed: Minim in its title bar and its playlist's footer, Marquee on its
display, Lantern in its status bar, its About box, and in every menu item
that would need a media stack if you pick it.

Every title, artist and album in the invented library was made up for
`internal/players/players.go`. No skin, bitmap, icon, font, name or mark
belonging to any real player is in this repository, as art or as a fixture:
the players' five skins are generated from paths and whole pixels in
`internal/skinart` like the other three, the transport glyphs are drawn in
`internal/players/ui.go`, and the display alphabets and clock digits Minim's
panels print in were set by hand on their grids for this toolkit.

## What each one is for

| | Minim | Marquee | Lantern |
| --- | --- | --- | --- |
| shape of it | a strip, 275 × 116 design pixels | a cabinet, 780 × 540, that folds to 468 × 104 | a window with a menu bar, 880 × 560 |
| windows | three, stacked and travelling together | one, in two shapes | two, one anchored to the other |
| skin | `minim`, a pixel sheet over `win95`, and two panels it switches to live: `minim-classic` and `minim-silver` | `marquee`, paths and glass over `breeze-night` | `lantern`, matte, a *partial* skin over `breeze-night` |
| silhouette | the skin's: steps in twice at the bottom; round at every corner in silver | the skin's brow and dome, **and the app's own stadium** when folded | the skin's asymmetric skirt |
| what it shows | proportions as design pixels, pixel art at fractional scales, windows that snap, one widget tree laid out as three different panels | one control changing the size, the layout *and* the outline at once | a skin is a pack: one key drops it |

### Minim — the compact one

```bash
go run ./examples/minim
go run ./examples/minim -only strip           # the strip on its own
go run ./examples/minim -theme breeze-night   # the same app, no skin
go run ./examples/minim -scale 1.75
go run ./examples/minim -shot out/            # one frame of each window
```

A strip with a phosphor display, a seek bar and a transport row, plus an
equaliser and a playlist that **snap flush to its edges and travel with it**.
Drag any of the three within ten pixels of another's edge and it sticks;
drag the strip and the stack follows; drag one out of the middle and it
takes whatever hangs from it and leaves what it hung from.

275 by 116 is what this shape was at 1×, and here it is a *design* size
rather than a frozen one: at 1.75 the window is 481 by 203, the silhouette
is redrawn at that size rather than stretched, and the art is the 2× sheet
drawn at a whole multiple because the skin declares itself `pixelated`.

#### Three skins, one tree

```bash
go run ./examples/minim -theme minim-classic   # the base-skin look of the era
go run ./examples/minim -theme minim-silver    # the rounded silver-and-blue one
```

Minim wears three skins and switches between them live, all three windows
at once:

- **`minim`**, its own: a pixel front panel dressing ordinary widgets.
- **`minim-classic`**, **Minim Classic**: the base-skin look of the era —
  slate-blue bevelled chrome, title bands with a gold groove either side of
  the words, a black display with thin green segments and a small green
  analyser, an orange volume bar and a green balance bar, grey transport
  keys, and an equaliser of eleven yellow faders. A pixel skin, exact at 1×
  and 2×.
- **`minim-silver`**, **Minim Silver**: the rounded look of the later era —
  silver chrome under a navy title band, a blue dot-matrix display with big
  pixel digits, glossy round keys that brighten under the pointer, capsule
  toggles beside blue lamps, an equaliser whose header is a tab with its name
  on it and no title band at all, a blue playlist — in windows whose four
  corners are round, with the desktop showing beyond them. Drawn from paths,
  exact from 1 to 2.

The two panels are what their names say: pictures of each whole window with
holes where the keys go, which is how a player of this shape was built.
**Each skin says where its holes are**: it states three fixed layouts —
`minim.strip`, `minim.equaliser`, `minim.playlist` — whose named slots the
player binds its controls to with `widget.Slots`
([docs/skins.md](skins.md#fixed-layouts)). The player carries no rect of its
own for either panel; it reads every one it uses from the skin it is
wearing, including the ones it prints into — the clock's digits, the lamps,
the scrolling title, the playlist's rows. The rects are generated from
`internal/players/minim/panel`, the same numbers the faces are drawn round,
so the art and the layout cannot disagree about where a key is. A key's
look in every state is its slot's art; the digits, lamps, thumbs and bitmap
capitals are sprites the player paints by name
(`style.DrawSkinSprite`, [docs/skins.md](skins.md#sprites-an-app-paints-itself)).

Each window is given a role, the name of its layout
(`app.Window.SetFrameRole`), and that is how Minim Silver's equaliser gets
its tab: the skin states a caption for the `minim.equaliser` role — thirteen
pixels of pale chrome whose title plate is the tab — and the other two
windows keep the navy band ([docs/skins.md](skins.md#windows-that-are-not-alike)).
The windows keep the player's titles in every face; the panels print them in
capitals because their caption text roles say `"case": "upper"`.

The widget tree is the **same one in every face**. A switch moves and
repaints the controls, and hides the few one face has and another does not:
the panels have their own pause key, an eject key and a balance slider, and
no mute key. The focus, the tab order and the accessibility tree carry
across a switch, and a control that disappears hands the keyboard to the
obvious next one rather than to nothing. Under any pack that is not one of
the panels — `minim` itself, or any of the others — the player is laid out
as widgets.

The switch is reachable every way a control should be:

| | |
| --- | --- |
| the skin key | in the strip's own chrome: the column of option letters in the panels (it reads SKIN), the last toggle in Minim's own row. It steps to the next skin, and its accessible name says which one it is on |
| **Ctrl+K** | from any of the three windows, the same step: minim → minim-classic → minim-silver → minim |
| **Ctrl+Shift+K** | drops the skin for `breeze-night` and puts back the one it dropped — Lantern's switch, kept |
| a menu | a right-click anywhere on the face of any window, or the menu key (Shift+F10) from the keyboard, lists the skins by name with the one being worn ticked, and "No skin" |

The choice is **remembered**: it is written to
`$XDG_CONFIG_HOME/uitoolkit/minim.json` (the one piece of state the player
keeps), and `examples/minim` wears it again on the next run unless `-theme`
names a pack. A `-shot` run is posed, so it neither reads nor writes it.

The panels' extra keys do what is true in a player that plays nothing.
Eject stops and goes back to the first track. AUTO chooses a preset for
each track as it comes up. PRESETS lists the presets. MISC opens the skin
menu; SEL shows the playing track; ADD and REM say plainly that there is
nothing to add or remove; LIST OPTS jumps to the first track or closes the
list.

### Marquee — the one that folds

```bash
go run ./examples/marquee
go run ./examples/marquee -compact            # open folded
go run ./examples/marquee -theme breeze-night
```

A cabinet with a display, a queue and a curved transport shelf. Ctrl+M, or
the button at the right-hand end of the shelf, folds it into a flat stadium
and back.

The fold is the only place in the three where an **app** sets a silhouette
of its own. A skin declares one window shape — here the brow across the top
and the dome under the cabinet — and the stadium is the app's, set with
`SetShapeFunc`; the toolkit gives the app's shape precedence over the look's
exactly so that a window which knows it is a different shape can say so.
Passing `nil` hands the window back to its look.

It keeps **one widget tree** for both shapes rather than swapping two. That
is the behaviour and not a saving: the focus stays where it was, the
accessibility nodes keep their ids, and a control the keyboard was on is
still the control the keyboard is on after the window has changed size and
outline. Folding hides what does not fit, which takes those controls out of
the tab order and out of the accessibility tree — which is what "not in this
mode" has to mean.

### Lantern — the switch

```bash
go run ./examples/lantern
go run ./examples/lantern -themed             # start with the skin dropped
go run ./examples/lantern -only main
```

A main window cut to its skin's outline, with a playlist window anchored to
its right-hand edge. **Ctrl+K drops the skin**, and that is the argument of
the whole feature in four lines:

```go
ap := style.LookAppearance(app.Look())
ap.Name = "breeze-night"          // or "lantern"
app.SetLook(style.WithAppearance(app.Look(), ap))
```

It is the call Settings makes for any theme. What comes back is the same
widget tree in another look — the same tab order, the same accessibility
tree, the same keyboard — and a window that is a rectangle again, because
the outline belonged to the skin and never to the app.
`TestDroppingTheSkinKeepsTheApp` pins exactly that: the tab order compared
as a cycle, the tree compared by role and by every control's name, and the
one thing that is *supposed* to differ — the status line, which names the
pack — asserted separately.

Its skin is deliberately partial. It binds seventeen parts and leaves tabs,
splitters, switches and tree disclosures to `breeze-night`, because a skin
is always a partial override and a demo that only ever showed a complete one
would be hiding the common case.

## The keyboard

One table for all three, so learning the keys in one is learning them in all
(`players.CommandFor`):

```
Space  play / pause          ←  →   seek five seconds
Z  B   previous / next       ↑  ↓   volume
V      stop                  M      mute
S      shuffle               R      repeat: off → all → one
Escape quit
```

Minim adds Ctrl+K (next skin), Ctrl+Shift+K (drop the skin) and the menu
key (the skin menu); Marquee adds Ctrl+M (fold); Lantern adds Ctrl+K (skin)
and Ctrl+L (playlist).

Bare letters as shortcuts are safe here because no player hands one to a
text field. There is exactly one field in the three — Lantern's playlist
filter — and `players.Typing` asks the focused component what role it gives
the accessibility tree before any letter is dispatched. Typing "Beacon" into
the filter does not skip six tracks.

Every control is on the keyboard, including the ones drawn as glyphs:
`players.GlyphButton` is an ordinary component with a name, a focus ring, an
accessibility node and Return/Space, that happens to be painted from one of
the look's faces with a mark on it.

## The one thing a desktop may refuse

The snapping itself is not the players': it is the `rack` package, which
any application can use for satellite windows of its own — see
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
follow. Each player says which of the two it got rather than pretending:
Minim in its strip, Lantern in its status line ("playlist not placeable
here"), and both `examples/*` print `places-windows=` on startup.

## How it is put together

```
internal/players/            the model, and the pieces of interface all three share
  players.go  spectrum.go    the transport, the playlist, the analyser, the equaliser
  ui.go                      the glyphs, the transport button, the fader, the clock
internal/players/minim/      one package per player: its look and its layout, and no more
  panel/                     Minim's two panel layouts, read by the player and the skin generator
internal/players/marquee/
internal/players/lantern/
internal/players/playertest/ what the three tests do to a window
rack/                        windows that stick together: the arithmetic and the
                             desktop half, a public package the players import
                             like any other application would
```

The line between the shared package and each player is worth stating:
**anything a player of any era would have drawn the same way** is shared —
the transport marks, the analyser, the clock that drives them, the keyboard
— and everything about *how a particular player looked* is its own, because
that is the subject of the demo and sharing it would be sharing the thing
being demonstrated.

Three pieces there are worth knowing about.

**`players.Fader`** is a slider stood on end — or on its side, for a seek
bar whose thumb is a picture (`Horizontal`, `Painter`). It is a component rather than
a flag on `widgets.Slider` because the toolkit's slider is horizontal by
construction — it takes its height from the slider metric and draws along
its width — and ten faders side by side *are* an equaliser while a row of
horizontal ones is a form. It paints by asking the look for a slider inside
a rotated frame, so a skinned fader is skinned from `slider.track`,
`slider.fill` and `slider.thumb` with no art of its own.

**`players.Pulse`** is one timer for a whole application, not one per
window: three windows waking separately would advance the same transport
three times a frame and drop the analyser's peak markers three times as
fast. A wake delayed past a few periods is clamped, so a machine that was
asleep does not come back to a row of bars with no markers on them.

**`players.OpenSized`** opens a window whose size is stated in design
pixels. `platform.WindowOptions` speaks device pixels — the numbers go to
the compositor, which knows nothing about the toolkit's scale — so a strip
asking for 275 × 116 at 1.75 would otherwise open at 275 device pixels with
a silhouette resolved at 1.75, standing on a chin twice as deep as it was
drawn.

## The analyser, and why a screenshot is reproducible

The bars are a pure function of the transport's position: a few sines at
incommensurable rates per band under a falling tilt, and nothing else. Two
consequences, both deliberate.

A screenshot of a player posed at 1:37 is the *same picture* every time it
is taken, so one run can be compared with the one before it. And the same
second of playing looks the same whether the window repainted twelve times
in it or sixty, which matters because the three run at different rates and
all three run slower under a screen recorder.

Only the peak markers and the fall to silence have memory. The scrolling
title takes its phase from the same position, and slides back and forth with
a rest at each end rather than looping: a loop is what this era did and is
one line of arithmetic, but two thirds of every still frame of one is a
title cut in half with the next copy beside it.

## Tests

```bash
env -u WAYLAND_DISPLAY -u DISPLAY go test ./internal/players/...
```

The model is tested without opening a window at all — the transport's wrap
at the end of a track, the seek bar's fraction, the shuffle that is a
permutation beside the list rather than of it, where a snapped window lands.
Each player is then tested headless for the four things a skinned app has to
keep: every control on the keyboard, `a11y.Check` clean in the skin *and*
with the skin dropped, the silhouette at 1, 1.25, 1.5, 1.75 and 2, and a
themed fallback that is a whole app. Minim's switch is tested for the same
things at every stop: each of its skins paints all three windows at the
five scales with the display where the layout says, the silver windows are
round at their corners and nowhere else at every scale, the skin key and
Ctrl+K go all the way round, the menu lists the skins by name, every panel
control is on Tab, and the choice is remembered.

The silhouette is checked as **fractions of the window** rather than as
pixels or as a golden image. That is exactly what a shape stated in design
pixels and resolved against the window buys — the same numbers at every
scale, which a bitmap stretched to fit would not give — and a failure names
the row that is wrong instead of handing back two pictures.

The panels on the skins' own layouts are tested for what placing may and may
not do: every control is on its slot at every scale, the tab order is the
widget face's, less the keys one face has and the other does not; the silver
equaliser wears its own caption and the other two windows the band; a round
key refuses its corners and takes its middle; Silver's keys have a hover face
and Classic's do not; the titles are the player's in every face; and at 1.25,
1.5 and 1.75 Minim Classic's transport row is nothing but colours of its own
art.

Real hardware, through the nested-KWin rig, is in
[docs/e2e/2026-09-17/players-realhw.md](e2e/2026-09-17/players-realhw.md),
and the five gaps below that were closed afterwards are checked on the same
rig in
[docs/e2e/2026-09-21/toolkit-gaps.md](e2e/2026-09-21/toolkit-gaps.md). Minim's three skins, the live switch and Minim Silver's
round corners are checked there too, on Wayland and X11 at 1 and 1.75, in
[docs/e2e/2026-09-21/minim-skins-realhw.md](e2e/2026-09-21/minim-skins-realhw.md),
and the skin gaps closed after that — the silver equaliser's own caption,
the round keys' corners, Classic at 1.75 and Settings' preview of Deck — in
[docs/e2e/2026-09-21/skins-2-realhw.md](e2e/2026-09-21/skins-2-realhw.md).

## What they ran into

Things a skinned, shaped, multi-window app wanted and the toolkit could not
do, kept here because they are the useful output of writing one. Most have
since been fixed in the toolkit and are struck through, here and under
[Fixed since](#fixed-since); what is not struck through still stands.

- ~~**A look states the caption's height and an app cannot ask for a
  shorter band.**~~ **Fixed:** `app.Window.SetCaptionHeight` asks for a
  caption of the app's own height in any look, and the caption's buttons
  shrink to stand in it. Marquee's caption stays at the 40 design pixels its
  art was redrawn for.
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
  buttons to the side its art has room on. None of the players' skins states
  one; their silhouettes keep out of both corners.

Minim's panels ran into five more:

- ~~**A skin states one caption for every window it dresses.**~~ **Fixed:**
  window variants, chosen by the role the app gives a window. Minim Silver's
  equaliser has its tab for a header and no title band.
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
  rows so the four fit without a scroll bar. A menu that could leave its
  window (a Wayland xdg_popup, an X11 override-redirect window) would lift
  that limit for every compact app.
- **A skin's caption shorter than 24 pixels was grown to 24**: its buttons
  had the caption's height, and a frame gives such a button at least that.
  Fixed in the toolkit — a short caption now gets square buttons stood in
  it — which also gives Minim's own skin the 18-pixel caption it always
  stated.

And three that writing the panels on the skins' own layouts closed:

- ~~**A skin cannot lay a panel out**, so the player carried one layout per
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
  real boxes in first and each player attaches its panes once that answers
  yes.
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
  a control said so itself (`players.DragsWindow`).~~ **Fixed:** a press
  walks up from the component under the pointer until one takes it, as a
  wheel notch and a key already did, and the contract for what "took it"
  means is written down in [docs/widgets.md](widgets.md). The taker becomes
  the pointer capture; focus still goes to what the press landed on.
  `DragsWindow` is still the clearer way to say "this part of the face is a
  handle", but it is no longer the only way.
- ~~**`platform.WindowOptions.Resizable` is declared and never read**, and
  there is no maximum size, so a player whose size *is* its design cannot
  say it is fixed. A press eight pixels below the strip's top edge lands in
  the frame's resize band and grew a 275×116 window to 275×308.~~
  **Fixed:** `Resizable` became `Sizing` (an enum whose zero value is
  resizable, beside `Decorations`) with `MaxWidth` and `MaxHeight`.
  `SizingFixed` states the same minimum and maximum to the desktop, refuses
  an interactive resize, drops maximize from the window's capabilities and
  leaves no resize band in the toolkit's own frame. `players.OpenSized`'s
  `fixed` now means it.
- ~~**Window sizes are device pixels on X11 and logical pixels on
  Wayland**, and no scale factor reconciles them.~~ **Fixed:** window
  geometry is logical pixels everywhere and the X11 backend converts at its
  own boundary; `app.Window.PixelSize` is the device-pixel box for code
  that pairs a size with a widget rectangle. `players.WindowSize` is the
  identity now — a design pixel *is* a logical pixel — and is kept only
  because two call sites read better with it.
- ~~**Window positions were still device pixels**~~ while sizes were
  logical. **Fixed:** `Position`, `Move` and `WindowOptions.X`/`Y` are
  logical too, so a rack's box is a position and a `Size` in one unit, and
  its reach (`rack.DefaultReach`) grows with the display without the apps
  scaling it. At a fractional scale on X11 two windows snapped flush can
  still meet a device pixel apart: each edge is rounded to the root's
  pixels on its own.
