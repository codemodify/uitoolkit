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

Run the demo:

```bash
go run ./examples/shapes                 # a clock: a disc with a hole at the hub
go run ./examples/shapes -mode ring      # a rounded panel with a big hole
go run ./examples/shapes -mode panel -glass
go run ./examples/shapes -mode backdrop -size 1100   # a test card to put behind
```

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

The four `ctx.BackdropBlur` calls in the engines are unchanged. They blur
what is under a *menu* or a flyout, which is the window's own content, and
they are still right.

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

Both take rectangles and nothing else, so the silhouette is rasterised to
scanline runs first. **The region is hard-edged however smooth the painted
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
default, every widget in the toolkit and all 121 packs rely on it, and
nothing about it has changed — a component that never asks pays one nil
check in `HitTest`.

A component that wants otherwise says so:

```go
c.SetHitShape(platform.ShapeEllipse(paintengine2d.XYWH(0, 0, 100, 100)))
c.SetHitShapeFunc(func(size paintengine2d.Point) *platform.Shape { … })
c.SetTransparent(true) // paints nothing solid over its box
```

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

**Time**, per frame at 875×875:

| | Opaque | Shaped |
| --- | --- | --- |
| Full repaint | 0.61 ms | 2.98 ms |
| Partial repaint (a 120×40 box) | 19 µs | 153 µs |
| Asking a window for its silhouette | — | 41 ns |
| Rasterising it from scratch (a resize) | — | 16 ms, 1.97 MB |

The rasterisation is the one expensive thing, and it happens **once per
resize**, never per frame: a shape caches its own rasterisations and a
repeat lookup allocates nothing. The per-frame cost is the dest-out punch
that cuts the silhouette out of the painted window, and it is charged only
for the boxes the silhouette actually cuts — the space outside it, its holes
and its antialiased edges — so a shaped window that happens to be a
rectangle costs what an unshaped one costs.

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

- **The rasteriser is CPU and single-threaded.** 16 ms for an 875 px ring is
  a visible hitch during an interactive resize of a large shaped window. It
  is one pass over the mask and an obvious candidate for the GPU, or for
  rasterising only the rows a resize actually changed.
- **A shape's coverage is rasterised through an RGBA scratch image**, a strip
  at a time, because paintengine2d's CPU device takes its 8-bit `FormatA8`
  as a source but not as a render target — every blend writes four bytes at
  a one-byte-per-pixel index, so an A8 target runs past the end of its own
  buffer (it panics; see `platform.rasterPathMask`). Fixing that in the
  renderer would remove a copy and a quarter of the scratch.
- **Shaped popups, menus and tooltips** are not done. They are in-window
  overlay layers here, not separate surfaces, so they need the same work one
  level down.
- **Damage is clipped to the silhouette's bounding box**, not to its
  rectangles. Clipping each damage box against a silhouette's own rows would
  turn one box into as many as the shape has rows, and presenting hundreds
  of slivers costs the compositor more than repainting a corner the window
  does not own — whose pixels are punched transparent anyway.
- **The looks' pre-flattened materials.** Big Sur's and Tahoe's sidebars and
  macOS's menu bar still flatten their translucent tints over the window
  colour, which was the right thing when no real blur existed. With glass in
  effect they could keep their real alpha.
- **dmabuf presents stay opaque**; a shaped window uses shm or EGL with
  alpha. Worth revisiting only if it ever matters for performance.

## See also

- [docs/decorations.md](decorations.md) — the window frame the silhouette
  sits inside, and the phase 3 work (ARGB buffers, `set_input_region`, the
  X11 32-bit visual, the compositing-manager fallback) this is built on.
- [docs/platform.md](platform.md) — the backends and the protocols.
- [docs/theme-engines.md](theme-engines.md) — materials, and the four
  in-window `BackdropBlur` sites.
