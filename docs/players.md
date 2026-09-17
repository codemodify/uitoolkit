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
the three skins are generated from paths in `internal/skinart` like the
other three, and the transport glyphs are drawn in `internal/players/ui.go`.

## What each one is for

| | Minim | Marquee | Lantern |
| --- | --- | --- | --- |
| shape of it | a strip, 275 × 116 design pixels | a cabinet, 780 × 540, that folds to 468 × 104 | a window with a menu bar, 880 × 560 |
| windows | three, stacked and travelling together | one, in two shapes | two, one anchored to the other |
| skin | `minim`, a pixel sheet over `win95` | `marquee`, paths and glass over `breeze-night` | `lantern`, matte, a *partial* skin over `breeze-night` |
| silhouette | the skin's: steps in twice at the bottom | the skin's brow and dome, **and the app's own stadium** when folded | the skin's asymmetric skirt |
| what it shows | proportions as design pixels, pixel art at fractional scales, windows that snap | one control changing the size, the layout *and* the outline at once | a skin is a pack: one key drops it |

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

Marquee adds Ctrl+M (fold), Lantern adds Ctrl+K (skin) and Ctrl+L
(playlist).

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
same, `players.Rack` is tested either way — and the windows simply cannot
follow. Each player says which of the two it got rather than pretending:
Minim in its strip, Lantern in its status line ("playlist not placeable
here"), and both `examples/*` print `places-windows=` on startup.

## How it is put together

```
internal/players/            the model, and the pieces of interface all three share
  players.go  spectrum.go    the transport, the playlist, the analyser, the equaliser
  rack.go                    the arithmetic of windows that stick together
  desk.go                    the same rack with real windows in it
  ui.go                      the glyphs, the transport button, the fader, the clock
internal/players/minim/      one package per player: its look and its layout, and no more
internal/players/marquee/
internal/players/lantern/
internal/players/playertest/ what the three tests do to a window
```

The line between the shared package and each player is worth stating:
**anything a player of any era would have drawn the same way** is shared —
the transport marks, the analyser, the clock that drives them, the keyboard
— and everything about *how a particular player looked* is its own, because
that is the subject of the demo and sharing it would be sharing the thing
being demonstrated.

Three pieces there are worth knowing about.

**`players.Fader`** is a slider stood on end. It is a component rather than
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
themed fallback that is a whole app.

The silhouette is checked as **fractions of the window** rather than as
pixels or as a golden image. That is exactly what a shape stated in design
pixels and resolved against the window buys — the same numbers at every
scale, which a bitmap stretched to fit would not give — and a failure names
the row that is wrong instead of handing back two pictures.

Real hardware, through the nested-KWin rig, is in
[docs/e2e/2026-09-17/players-realhw.md](e2e/2026-09-17/players-realhw.md).

## What they ran into

Things a skinned, shaped, multi-window app wanted and the toolkit could not
do, kept here because they are the useful output of writing one:

- **A look states the caption's height and an app cannot ask for a shorter
  band.** A player with a compact mode has to choose a caption that suits
  its *smallest* window; Marquee's came down from 56 design pixels to 40 for
  that reason alone.
- **`window.shape` rects anchor to the top and the left only.** A rect may
  stretch to a margin from the right or the bottom edge, but there is no
  fixed-size box anchored to either — so a tab at the top right, or a band a
  fixed distance up from the bottom, cannot be stated. All three silhouettes
  here are shaped by what *is* expressible.
- **`window.border` is one inset for the whole window.** A silhouette that
  varies down the window — wide at the top, narrow at the bottom — still has
  to keep its content inside a single rectangle, so the border has to clear
  the deepest intrusion on each side. Deck sidestepped it with a symmetric
  waist; Marquee's dome and Lantern's skirt are sized so a modest border
  clears them.
- **A skin cannot ask for a gap between its caption band and the content.**
  The sketch that fell out of this was a window in two pieces with a slot of
  desktop between them, which is a striking outline and not expressible: the
  content starts where the caption ends.
- **Which side the caption buttons sit on is the desktop's**
  (`style.CaptionButtonsDesktop` is the default and a skin's `layout` is
  consulted only when the user asked for the look's own), so a silhouette
  must stay out of *both* top corners or it eats a close button on half the
  desktops it runs on. Every shape here does.
- **`widget.KeyEvent` carries the key and not the character.** Text arrives
  as a separate event, so a shortcut table that reads `e.Rune` never fires —
  which is a fine design, and worth saying in the doc comment on KeyEvent.
- **A window opens with nothing focused**, so keys that bubble from the
  focus reach nobody until the first Tab. All three focus a control on their
  first frame.
- **A mouse press does not bubble.** `Window.hit` finds the deepest
  component under the pointer and tells that one alone, so a press on a
  player's display did nothing until every part of the face that is *not* a
  control said so itself (`players.DragsWindow`). It is the right design —
  bubbling presses is how one click ends up doing two things — but "the
  container will get it" is the natural assumption and it is not true.
- **`platform.WindowOptions.Resizable` is declared and never read**, and
  there is no maximum size, so a player whose size *is* its design cannot
  say it is fixed. A press eight pixels below the strip's top edge lands in
  the frame's resize band and grew a 275×116 window to 275×308.
- **Window sizes are device pixels on X11 and logical pixels on Wayland**,
  and no scale factor reconciles them: an X11 server has no notion of a
  display scale, so a 275-design-pixel strip at 1.75 asks for 481, while a
  Wayland toplevel's geometry is logical and the compositor multiplies, so
  the same strip asks for 275. `players.WindowSize` is the work-around here;
  the fix belongs in `platform`.

Two more are about *when* rather than *what*, and only real hardware showed
them (see the e2e report):

- **A window manager places a window when it maps it**, and the application
  is told afterwards — so a rack built when its windows are made is built
  around windows that are all still at the origin. `Desk.Adopt` takes the
  real boxes in first and each player attaches its panes once that answers
  yes.
- **A move is a request.** X11 answers one a frame or two later, so reading
  the position straight back says the window is where it was and looks
  exactly like a drag. `Desk` waits for a move it asked for to arrive, and
  gives up after twenty-five looks so a window manager that refuses one —
  KWin clamps a window that would hang off the screen — is not argued with
  forever.

Three small things were added rather than worked around: `app.Window.Move`
and `CanMove` (Position's other half, over `platform.SurfaceMoves`),
`app.Window.SetSize` (Size's other half), and `platform.SurfaceMoves`
itself.
