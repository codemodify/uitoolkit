# Shaped and transparent windows

A uitoolkit window does not have to be a rectangle. It can be any
silhouette — a disc, a ring, a BeOS tab, a skin's outline — with the desktop
showing through everywhere it is not, and a hole in the middle you can see
through *and click through* to whatever is behind it. It can also be glass:
the desktop blurred by the compositor behind a translucent window, which no
amount of painting inside a window can do.

The silhouette is a normal part of a window's frame state. A shaped window
is still a themed window with the same widget tree, the same keyboard
handling and the same accessibility tree; `SetShape` is a request a look or
an app makes, not a separate kind of window.

Run the demos:

```bash
go run ./examples/uitoolkit-sample-shapes                 # a clock: a disc with a hole at the hub
go run ./examples/uitoolkit-sample-shapes -mode ring      # a rounded panel with a big hole
go run ./examples/uitoolkit-sample-shapes -mode panel -glass
go run ./examples/uitoolkit-sample-shapes -mode backdrop -size 1100   # a test card to put behind

# The same silhouette asked for by a *look* rather than by the app.
go run ./examples/uitoolkit-sample-skinshape -mode backdrop           # the test card again
go run ./examples/uitoolkit-sample-skinshape                          # the Deck skin's outline
go run ./examples/uitoolkit-sample-skinshape -theme beos              # BeOS's tab
go run ./examples/uitoolkit-sample-skinshape -theme breeze-night      # a look that frames a rectangle

```

A whole application built on both halves — a skin's outline, and, when it
folds down, the app's own — is the music player in
[media-player-music](https://github.com/codemodify/media-player-music); what
the toolkit does for it is in [players.md](players.md).

Put the backdrop up first, the shaped window over it, and click the hole:
the app counts every press it receives and a click in the hole adds none.
Drag the window and the test card slides past the hole, which is how you
know the hole is the window's own and not a picture of what was there.

## What a shape is

`platform.Shape` is a silhouette. It is given as **a path** or as **a
coverage mask**, and the two are for different things:

| | Path (`NewShape`, `NewShapeEvenOdd`) | Mask (`NewShapeMask`, `NewShapeImage`) |
| --- | --- | --- |
| Resolution | Rasterised at the surface's device size: exact at 1.75 as at 1 | Resampled from its own pixels: crisp at its natural size, soft when stretched |
| Cost to keep | The path's points | One byte a pixel |
| Who wants it | A look, a decoration, an app that draws its own outline | A skin, whose silhouette arrives as a bitmap's alpha channel |

Use a path wherever the shape can be described. Use a mask where the
silhouette *is* artwork — that is what a skin has, and converting it to a
path would only lose its edge.

A hole is not a special case. It is a second contour of the same path,
filled even-odd:

```go
p := paintengine2d.NewPath()
p.AddRoundRect(paintengine2d.XYWH(0, 0, w, h), 28, 28) // the body
p.AddCircle(paintengine2d.Pt(w/2, h/2), w*0.29)        // the hole
shape := platform.NewShapeEvenOdd(p)
```

That is exactly what `CombineRgn(RGN_XOR)` did for Win32's `SetWindowRgn`,
and it is why nothing in the API mentions holes: a hole is a shape that
encloses a region twice.

**Everything is device pixels.** A shape is stated in the visible window's
device pixels, like `platform.Frame`, `frameGeom` and `DecorationSpec`. The
callback is handed the size and the scale, so it draws at 875 px for a
500-logical-pixel window at 1.75 rather than drawing at 500 and letting
something stretch it.

## For apps

```go
win.SetShapeFunc(func(size paintengine2d.Point, scale float32) *platform.Shape {
	p := paintengine2d.NewPath()
	p.AddCircle(paintengine2d.Pt(size.X/2, size.Y/2), min(size.X, size.Y)/2-1)
	p.AddCircle(paintengine2d.Pt(size.X/2, size.Y/2), min(size.X, size.Y)*0.11)
	return platform.NewShapeEvenOdd(p)
})
```

- `SetShapeFunc(fn)` — the silhouette follows the window. `fn` runs again
  whenever the size or the scale changes, and its result is remembered until
  then. **Prefer this.** A shape built once and stretched is a bitmap's
  behaviour, not a path's.
- `SetShape(s)` — one fixed shape, rasterised at whatever size the window
  is. For a window that cannot be resized.
- `Shape()` — the shape asked for. `nil` is the plain rectangle every window
  has by default.
- `ShapeActive()` — whether the silhouette is **in effect right now**. It is
  false while the window is maximized, full-screen or tiled (below). An app
  that paints its own outline must ask this and paint a plain rectangle when
  the answer is no; painting a hole into a window that no longer has one is
  how a shaped app looks broken when it is maximized.
- `SetShapeOpaque(true)` — the window paints every pixel inside its
  silhouette solid, so the compositor may skip what is behind them. Off by
  default: it is the one setting here that is visible corruption when it is
  wrong. Only fully covered pixels are ever claimed, never the antialiased
  edge, so it is safe whenever the interior really is solid.

A shaped window normally wants `Decorations: platform.DecorationsNone` — it
draws its own outline, and a title bar around a disc would be a rectangle
around it. A shaped window with the toolkit's frame works too: the
silhouette is stated in the visible window's coordinates and moved by the
shadow margin.

## For looks

A silhouette does not have to be the app's. A **look** can declare one, and
then every window in that look is cut to it without the app knowing: a
skin's outline, BeOS's tab. The app's own shape wins where it set one — a
window drawing its own picture knows what it is and a look does not — and a
window that set none takes its look's, exactly as it takes the frame's
corners and shadow.

Two optional engine hooks, in `style/silhouette.go`:

```go
// The window's outline, in the visible window's own device pixels.
type WindowShapeEngine interface {
	WindowShape(l *Classic, f DecorationFrame, st DecorationState) *Silhouette
}

// The outline of one face: where a round button really is.
type ControlShapeEngine interface {
	ControlShape(l *Classic, b paintengine2d.Rect, role Role) *Silhouette
}
```

A `style.Silhouette` is a *description* — a `paintengine2d.Path` (with a
fill rule) or an image whose alpha is the coverage — rather than a
`platform.Shape`, because `platform` imports `style` and not the other way
round, and because rasterising is the expensive half:
`platform.NewShapeSilhouette` is the one place the two meet.
`style.SilhouetteOfRects` builds the common case, a union of rounded
rectangles filled nonzero so two that overlap are one outline.

- **`WindowShapeOf(lk, f, st)`** applies the per-state rules above for every
  look alike, exactly as `DecorationOf` applies them to the frame: dropped
  while maximized or tiled, kept on an uncomposited screen (XShape's
  bounding shape is the only thing that works there at all). A full-screen
  window never asks, because it has no toolkit frame.
- `f` is the frame's geometry **with the visible window at the origin**, so
  a look states its outline in the coordinates it is handed and the window
  moves it by the shadow's margin. `f.Caption` is in it because the caption's
  extent is the one part of a frame a look cannot work out for itself: BeOS's
  tab is as wide as its title, and the title is the window's. A look that
  wants a band narrower than its window asks for
  `DecorationSpec.CaptionFits` ([decorations.md](decorations.md)), which is
  dropped wherever the silhouette is.
- **`ControlShapeOf(lk, b, role)`** is the same idea one layer down, and
  `widget` hit-testing falls back to it for a component that names the face
  it paints (`widget.ShapeRole`) and has no silhouette of its own. There is
  deliberately **no `ControlState`**: a hit area that changed with the
  pointer would decide its own input — a hover face one pixel smaller than
  the resting one leaves a ring where the control hovers, stops covering the
  pointer, unhovers, and hovers again.

Bounding box and plain rectangle stay the default. A look that implements
neither hook is what it always was, which is all 123 engine packs; the cost
to a component that names no face is two nil checks and a type assertion
that fails.

## The resize band

A silhouette is the window's input region, so the band a frame keeps in its
shadow's margin — where a press resizes the window — would go with it. A
shaped window the user may resize keeps one along its silhouette's **outer**
edge instead (`app/shapeband.go`): up to 10 DIP outside the silhouette
wherever the surface has room (the margin, or the space the silhouette
leaves in its box), and 4 DIP just inside it, as a frame without a shadow
keeps inside its border. The band is part of the input region, and a press
in it starts the desktop's resize from the edge it is on.

- A row's outermost covered pixels are the left and right edges there, a
  column's the top and bottom; a diagonal is both, which is a corner, and a
  press near the ends of a straight edge takes the corner too, as a
  rectangular frame's does.
- Only an end that faces outwards counts — within a quarter of the box of
  the box's own side — so the step beside BeOS's tab, or a concave skin's
  notch, is not an edge. A hole never is: a ring is resized from its rim.
- A fixed-size window (`platform.SizingFixed`) keeps none, and neither does
  a maximized, full-screen or tiled one (which has no silhouette then).

It is worked out once per rasterisation from the silhouette's rectangles,
not per frame: 1.3 ms for a 1600x1200 window.

## Popups

Menus, combo lists, submenus and tooltips are surfaces of their own
([platform.md](platform.md#popups)), which is what lets them take a
silhouette like a window does. Each popup's surface carries its own shadow
in its own margin, and its frame is one of three things, in order:

1. **the popup component's own hit shape** (`widget.Base.SetHitShapeFunc`):
   the surface is cut to it with the same wipe and eraser as a shaped
   window, its shadow is the silhouette blurred in the colour of the look's
   own popup shadow, and its input region is the silhouette —
   `examples/uitoolkit-sample-popups`' Round menu is a disc;
2. **the look's**, `style.PopupShapeOf` — an optional engine hook,
   `PopupShapeEngine.PopupShape(l, b, kind)`, beside the window and control
   hooks, so a skin's or a look's menu can be the shape its art is;
3. **the plain box** — a look's rounded menu frame is transparent in its
   corners on a surface of its own, with nothing claimed opaque.

Where the popup is glass (below), the blur behind it must stop at its
rounded corners: the outline is then read from what the popup paints, once
per size and look, and the blur region is that.

A silhouette needs a surface. A popup drawn inside its window (headless,
`UITK_POPUPS=layer`) is the rectangle it always was.

## Glass

Glass is the *desktop* blurred behind the window. It is not
`Context.BackdropBlur`, which blurs the window's own pixels — right for a
menu, which really does sit over the window's content, and useless for a
window background, where there is nothing behind but a desktop the window
cannot see.

```go
win.SetGlass(true)
win.SetGlassTint(look.Palette().Background.WithAlpha(0.35)) // optional
if !win.GlassAvailable() {
	// this desktop cannot blur; the look's painted approximation stands
}
```

- `SetGlass(on)` asks. `GlassAvailable()` says whether this desktop can do
  it — and it **changes while the app runs**: KWin drops the capability when
  desktop effects are switched off. Ask afresh rather than once.
- The window has to be translucent for it to show. When glass is in effect
  the window's background becomes the glass tint and the look's own opaque
  background is not painted, because painting it would hide every bit of the
  blur.
- `SetGlassTint` is the app's say over how much shows through. A fully
  transparent tint (the default) means the look's own, `style.GlassTint`.

### The one switch

Three looks have faked glass since they were written: Fluent's acrylic, Big
Sur's vibrancy and Tahoe's Liquid Glass. They now ask for the real thing —
Mica and vibrancy tint a window with the desktop behind it, which needs the
compositor — and keep what they painted wherever they cannot have it.

The switch is in one place, `style/glass.go`:

- a look states `DecorationSpec.Glass` (Fluent and macOS from Big Sur on;
  their pack params `"acrylic"` and `"vibrancy"` still turn it off, and
  Yosemite's opaque windows never ask);
- `style.DecorationOf` drops it on an uncomposited screen, in the same
  clause that squares the corners and drops the shadow — there is nothing to
  blur there and alpha counts for nothing;
- `app` sets `style.SetGlassAvailable` from the platform capability every
  time a window lays itself out;
- everything else asks `style.GlassBehind(look)`, so the painted
  approximation and the real thing are never both on and never both off.

The four `ctx.BackdropBlur` calls in the engines blur what is under a *menu*
or a flyout, which is the window's own content — while the menu is drawn
inside its window. On a surface of its own there is nothing of the window's
under it, so a menu's material asks `style.LayerMaterial`, which reads what
the app says is behind the layer being painted (`style.LayerBackdrop`):

| Behind the layer | Blur the window's pixels | Tint |
| --- | --- | --- |
| the window (`BackdropWindow`) | yes | as the look states it |
| the compositor's glass (`BackdropGlass`) | no | its real alpha |
| a desktop nobody blurs (`BackdropNone`) | no | flattened solid over the look's background |

The popup's surface asks for blur behind itself whenever its look wants
glass and the desktop can blur, so Fluent's acrylic flyouts, Big Sur's
vibrant menus and Tahoe's glass menus are real glass over whatever is
behind the window.

The same switch keeps **true alpha** in the materials the looks used to
pre-flatten: over real glass Big Sur's sidebar pane keeps its material's
alpha, Tahoe's sidebar panel is laid over the blurred desktop rather than
the window colour (its rows have no opaque ground of their own), and the
macOS menu bar keeps its tint's alpha. Without glass they are what they
were.

**Aero** asks for glass too, but only for its frame
(`DecorationSpec.GlassFrame`): over a compositor that blurs, the caption
and borders take the window's background out from under themselves and lay
the glass gradient down translucent, and the client area keeps its opaque
background — Windows 7's Aero Glass, as against Mica or vibrancy, which
tint the whole window. An inactive window's frame stays opaque, as Windows
7's did; Aero Basic has no glass at all.

## Per-state rules

A silhouette is dropped where it makes no sense, exactly as the frame's
corners and shadow already are, and comes back on restore:

| State | Shape | Glass |
| --- | --- | --- |
| Restored | kept | kept |
| **Maximized** | **dropped** | kept (Mica does not stop at the screen edge) |
| **Full screen** | **dropped** | kept |
| **Tiled** on any edge | **dropped** | kept |
| Uncomposited X11 screen | kept, hard-edged | **dropped** |

A maximized, full-screen or tiled window fills a box the desktop chose and
shares its edges with the screen or a neighbour. A silhouette there would
leave gaps the user cannot click and the desktop did not expect.

## What the backends do

| | Wayland | X11 |
| --- | --- | --- |
| Where a press lands | `wl_surface.set_input_region` = the shape | `XShapeCombineRectangles` on `ShapeInput` |
| Which pixels exist | transparent pixels in the ARGB buffer | the same rectangles on `ShapeBounding` |
| What is solid | `set_opaque_region` | `_NET_WM_OPAQUE_REGION` |
| Glass | `ext_background_effect_v1` | `_KDE_NET_WM_BLUR_BEHIND_REGION` |

### Cutting the silhouette out

The window paints rectangular and the silhouette is then cut out of what it
painted — the same dest-out punch the rounded corners have always used. What
matters is *how* each pixel is cut, and there are two kinds:

- **The pixels the window does not cover at all** — the hole in the middle,
  the space outside the silhouette — are *wiped*: one path of whole pixels
  drawn with the dest-out operator, which takes every sample of every pixel
  it covers down to nothing.
- **The antialiased thread along the edges** is blended away through the
  coverage mask, because only the mask knows how much of each of those
  pixels is the window.

The distinction is not academic. Blending the whole cut away through the
mask is exact on the CPU and looks exact on a GPU too — until the GPU is
multisampling, where a blended draw does not promise to touch every sample
of a pixel and a thread of half-erased pixels survives just inside the
boundary. Against a saturated background that reads as a periodic stipple
around the hole, at 1× and without magnification. A fill of whole pixels
has no such freedom, so the pixels that must end up *entirely* transparent
are taken out by one, and `ShapeRaster.Clear` is exactly that region.

`TestUncoveredPixelsAreWipedNotBlended` pins it where a CPU render cannot:
it rasterises the wipe the paint path actually issues and checks it covers
every uncovered pixel and no covered one.

Both window systems take rectangles and nothing else, so the silhouette is
rasterised to scanline runs first. **The region is hard-edged however smooth the painted
shape is** — that is the one thing about a shaped window that cannot be
antialiased, on every toolkit and not just this one. The painted edge is
antialiased; the input edge lands on the nearest whole pixel.

A `wl_region` speaks logical pixels and a `Frame` device ones, so the
conversion rounds **outwards for input** (which must never lose a pixel the
window covers) and **inwards for opaque and blur** (which must never gain
one it does not paint solid). At a fractional scale those are different
rectangles, and converting the wrong way leaves a window deaf along its edge
or a hard seam around it.

X11 sets the bounding shape as well as the input shape. With a compositing
manager that is belt and braces — KWin honours both. Without one it is the
only thing that works: per-pixel alpha is simply ignored on an uncomposited
screen, so cutting the pixels away is what makes the hole visible. The
result is a hard-edged version of the same silhouette, which is the honest
fallback rather than a broken window.

## For widgets

A component takes input across its whole box, and always has. That is the
default, every widget in the toolkit and all 123 packs rely on it, and
nothing about it has changed — a component that never asks pays two nil
checks and a failed type assertion in `HitTest`.

A component that wants otherwise says so:

```go
c.SetHitShape(platform.ShapeEllipse(paintengine2d.XYWH(0, 0, 100, 100)))
c.SetHitShapeFunc(func(size paintengine2d.Point) *platform.Shape { … })
c.SetTransparent(true) // paints nothing solid over its box
```

A component that says nothing of its own asks its **look** instead, and only
when it names the face it paints:

```go
func (b *Button) ShapeRole() style.Role { return style.RoleButton }
```

A look with a silhouette for that face decides where the control is
(`ControlShapeOf` above) — which is how a skin's round button stops taking
clicks in the corners of its box. The answer is remembered beside the
component, keyed by the look, the face and the size, because building one
rasterises art; a look without the hook, which is every engine pack, leaves
the box alone.

`ShapeRole` is only for a component whose box *is* one face. A check box is
its indicator and its label, and shaping the pair by the indicator's art
would make most of the control deaf; `widgets.Button` is the one widget that
qualifies today.

A press outside the component's silhouette is not the component's — nor any
of its children's — and falls through to whatever is behind it in the tree,
which is the same rule a shaped window follows one layer down.

`SetTransparent` is a declaration, not a request: a window that is shaped or
glassy and claims its interior solid subtracts those boxes from the opaque
region it gives the compositor. An ordinary opaque window never looks at
them.

Note that a widget hit shape is for a component that is a shape *inside* a
window. Mirroring a shaped window's own silhouette at the widget level is
redundant — the window's input region already keeps those presses away from
the app entirely — and it is one more thing to keep in step when the window
is maximized and gives its silhouette up.

## What it costs

Measured on the author's machine (Arch, KDE Plasma 6, KWin 6.7.5, real GPU)
through the nested-KWin rig, at 875×875 device pixels — a 500-logical-pixel
window at scale 1.75.

**Memory**, settled `Private_Dirty`, two passes:

| Window | Private_Dirty |
| --- | --- |
| opaque, unshaped (baseline) | 41.5 – 41.9 MB |
| ring with a hole, static | 42.2 – 43.5 MB |

So a silhouette costs about **1–2 MB**: the coverage mask, the inverse mask
the punch uses, and the rectangle list. A clock that repaints twice a second
sits about 9 MB higher again, and that is its animation, not its shape — its
memory is flat over a minute and it rasterises its silhouette **once**,
however many frames it paints.

**Time**, per frame at 875×875. The machine was busy when these were taken,
so read them as upper bounds and as a comparison within each row rather
than as absolutes:

| | Opaque | Shaped |
| --- | --- | --- |
| Full repaint | 0.37 – 0.46 ms | 6.0 ms |
| Partial repaint (a 120×40 box) | 11 – 20 µs | 246 µs |
| Asking a window for its silhouette | — | 22 – 39 ns |
| Rasterising it from scratch (a resize) | — | 1.2 ms, 1.08 MB (was 7.5 ms) |

A large window's cold path, 1600x1200 device pixels (a rounded panel with a
round hole), measured with `go test -bench 'Shape(Mask|Raster)Large'
./platform`:

| | before | after |
| --- | --- | --- |
| The coverage mask | 14.8 ms | 1.5 ms |
| The whole rasterisation (mask and its four rectangle lists) | 19.3 ms | 2.6 ms |
| The resize band beside it (`BenchmarkShapeBandLarge`, app) | 6.8 ms | 1.3 ms |

Three things did it. The renderer (paintengine2d) now adds a span's whole
pixels once a row, as a difference array, rather than once for each of its
eight sub-scanlines, and fills a one-byte image without the per-pixel blend
dispatch: 13.8 ms to 3.8 ms on one core. The mask is drawn straight into a
one-byte coverage image over its own bytes (`NewImageA8`) — no RGBA scratch,
no copy — in up to four horizontal bands side by side. And the four
rectangle lists are worked out side by side too, one unsigned compare a
pixel. The UI thread waits on the bands; the work itself is spread over up
to four cores and done in a sixth of a frame.

The rasterisation is the one expensive thing, and it happens **once per
resize**, never per frame: a shape caches its own rasterisations and a
repeat lookup allocates nothing. The demo bears that out on real hardware —
a shaped clock repainting twice a second rasterised its silhouette once
across a minute.

The per-frame cost is the cut. It is charged only for the boxes the
silhouette actually cuts — the space outside it, its holes and its
antialiased edges — so a shaped window that happens to be a rectangle costs
what an unshaped one costs. Wiping the uncovered part rather than blending
it adds about 4% to a shaped frame, measured against the same code with the
wipe removed in the same run; it is not where the cost is.

Everyday repainting is the row that matters: a hover or a caret in a shaped
window costs a quarter of a millisecond, against a budget of sixteen.

**Rectangle counts** are the number a compositor sees: 51 for a rounded
panel, 666 for a ring, 728 for a disc with a hub hole, all at 875 px. KWin
and the X server took them without complaint — clicks and moves stayed
immediate. A silhouette needing more than `platform.ShapeRectLimit` (65 536)
rectangles is clamped to its bounding box and says so (`ShapeRaster.Clamped`):
that is a dithered or noisy mask, not a shape anyone meant.

## No regression for ordinary windows

A window that asks for no shape and no glass states exactly what it stated
before any of this existed: no shape, no opaque override, no blur, and for
an undecorated window the zero frame — which is the "say nothing at all"
path in both backends. It never rasterises a shape, never starts a frame
see-through, and its screenshot is the plain buffer.
`TestAnOrdinaryWindowIsUntouchedByShapes` pins it, and the benchmarks above
are the other half of the answer.

## Known gaps

- **Damage is clipped to the silhouette's bounding box**, not to its
  rectangles — measured, and kept. Clipping to the rows made every case
  slower on the window's own side before the compositor sees a single extra
  rectangle: a 375 px box over a ring's hole went from 1.66 ms (one box) to
  2.5 ms (64 boxes), a hover on its rim from 0.09 ms to 0.14 ms (two), at
  875 px. Each box repaints the tree clipped to it; the pixels saved are
  punched transparent anyway.
- **dmabuf presents stay opaque**; a shaped window uses shm or EGL with
  alpha. Worth revisiting only if it ever matters for performance.
- **The renderer's multisampled dest-out through an image is not
  sample-exact** on Mesa's iris (Intel, 4x MSAA): whole 2x2 blocks next to a
  change in the eraser are left unwritten, whatever the filter, the texture
  format or how the texel is chosen, and never without multisampling.
  Per-sample shading hides most of it but not all, which is not a clean fix,
  so the wipe of whole pixels above stays. paintengine2d keeps a probe of it
  (`PE_PROBE_MSAA_DESTOUT=1 go test -run TestGPUDestOutImageIsSampleExact`).
- **A skin cannot yet state a popup silhouette in `skin.json`**; the look
  hook (`PopupShapeEngine`) is there for the skin engine to implement.
- **On X11 under Xwayland a click on the bare desktop does not dismiss a
  menu** (see [platform.md](platform.md#popups)).

## See also

- [docs/players.md](players.md) — an application that wears all of this:
  a skin's silhouette, an app's own, windows that snap to one another, and
  what a desktop that will not place a window does to that.
- [docs/decorations.md](decorations.md) — the window frame the silhouette
  sits inside, and the phase 3 work (ARGB buffers, `set_input_region`, the
  X11 32-bit visual, the compositing-manager fallback) this is built on.
- [docs/platform.md](platform.md) — the backends and the protocols.
- [docs/theme-engines.md](theme-engines.md) — materials, and the four
  in-window `BackdropBlur` sites.
