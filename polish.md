# Polish

Rough edges found while building the demos and the features under them —
none blocking, all worth doing. Each names where it lives and why it matters.
Struck through once fixed; newest findings at the end of their section.

## Paused

- **Mail, full fledged.** The last app on the sample list (folders, threads,
  search, compose with attachments, tray, accessibility and the keyboard
  throughout). Paused by the author on 2026-09-21.

## Appearance and themes

- **Changing the theme resets the app-level preferences.**
  `style.LookAppearance` does not carry decorations, caption-button placement,
  reduced motion or native dialogs back out, and `Application` has no getter
  for the decorations preference. An app that rebuilds an `Appearance` from its
  look to switch packs silently undoes the rest; the tour keeps its own
  `style.Appearance` to work round it (`tourState.apply`).
- ~~**Settings' preview draws a shaped skin as a rectangle.** The preview is a
  panel inside the Settings window, so a compositor-level silhouette never
  applies to it. Needs the panel clipped by `style.WindowShapeOf`.~~ A
  window panel is cut to its look's silhouette (`widget.PaintClipper`) and
  keeps its body inside the look's border.
- **Settings' theme browser width is not recomputed on a live resize** —
  only on a rebuild (page change, Apply, Revert). There is no window-resize
  hook in `app.Window` to hang it on.
- **The preview app's Controls tab clips its last radio row** at Settings'
  default split; dragging the sash fixes it.
- ~~**Skins have no lint command.** `style.LoadSkin` already produces the keyed
  errors; `uitk-skin lint` would be a ~40-line `main`.~~ `uitk-skin lint`,
  with `style.LintSkin`'s warnings: fallback states, unbound sprites, art too
  small for 2×.

## Window frames and shapes

- **Shaped popups, menus and tooltips are still rectangles** (they are
  in-window layers), so a skin's silhouette stops at its window.
- **Caption buttons are not hit-shaped** — their boxes belong to the window
  system, which takes rectangles.
- **A shaped window loses its outer resize band**: the input region replaces
  it. Deck's 26 px bezel has never been dragged by hand.
- **Rasterising a large silhouette takes ~16 ms on the CPU, single-threaded**,
  a visible hitch while resizing a big shaped window. Cache is fine; the cold
  path wants the GPU or a second thread.
- **Damage is clipped to the silhouette's bounding box**, not its rectangles —
  deliberate, but a ring repaints its hole's pixels.
- **Big Sur's and Tahoe's sidebars and macOS's menu bar pre-flatten their
  tints**; with real glass behind them they could keep true alpha.
- **Aero's glass is still an opaque approximation** — no blur behind.
- **BeOS's tab uses the header bar's title padding**, a few pixels wider than
  R5's 19/17-cell gaps (the in-app frame keeps R5's exact metrics).
- **Platinum's title centres in the caption's free space**, so it sits a few
  pixels right of where Mac OS 8 put it; **Window Maker's** tile seam lacks the
  middle section's 1 px highlight.
- **Still open from the decorations plan**: KWin's server-decoration palette
  (colouring the desktop's own frame to match the look), `xdg-toplevel-icon`,
  and `_NET_WM_SYNC_REQUEST` for tear-free X11 resizes.

## Skins, from building the players

- ~~**A look owns the caption's height**; an app cannot ask for a shorter one
  (Marquee's compact mode had to live with 40 px).~~ `Window.SetCaptionHeight`.
- ~~**`window.shape` rects anchor to the top and left only**; a rect cannot pin
  to the right or bottom edge.~~ `fromRight` / `fromBottom`.
- ~~**`window.border` is one inset for the whole window**, so a silhouette whose
  bezel varies down its height cannot inset content to match.~~
  `"border": {"caption": …, "content": …}`.
- ~~**A skin cannot ask for a gap between its caption band and the content.**~~
  `captionGap`.
- ~~**Which side the caption buttons sit on is the desktop's** unless the user
  switches to the theme's layout — a skin whose art puts them somewhere must
  force its own layout.~~ `"buttons": "left" | "right"`.
- ~~**A skin cannot give each window its own caption**~~ (Minim Silver's
  equaliser kept a navy band under its tab). Window variants, chosen by
  `Window.SetFrameRole`.
- ~~**A skin cannot lay out a panel itself**; apps hand-place every
  control.~~ Fixed layouts: named slots an app binds its widgets to
  (`widget.Slots`); Minim's panels are on them.
- ~~**Round keys an app paints take clicks across their whole box.**~~
  `widget.ArtShape`, `style.SkinSpriteShape`, `style.SkinSlotShape`.
- ~~**Panel keys have no hover state.**~~ Slot art has states; Silver's keys
  brighten, Classic's era had none.
- ~~**Minim Classic's pixels are uneven at fractional scales.**~~ A panel's
  pixel pieces are magnified onto one device grid, identical on CPU and GPU
  (0 blended pixels in the transport row on X11 at 1.75, real GPU).
- ~~**Panel skins turn window titles upper-case.**~~ The titles are the
  app's; capitals are a text role's `"case": "upper"`.
- **Wayland stretches a window whose logical size × a fractional scale is not
  whole** — Minim's 275 × 1.75 = 481¼ — by that quarter pixel when it
  composites it, which softens a pixel skin's edges there (978 blended pixels
  in Minim Classic's transport row on KWin at 1.75, 0 for the same build on
  X11). Sizing such a window to a multiple of four logical pixels at 1.75
  would avoid it; the players keep their design sizes.
- **An in-app window panel does not know a shaped skin's caption or border
  art**: Settings' preview is cut to Deck's outline and keeps its body inside
  Deck's border, but its caption is the base engine's in-app caption.
- **Marquee could now ask for its compact caption** (`SetCaptionHeight`) and
  give its full cabinet the 56-pixel band it was first drawn with; its art is
  still the 40-pixel compromise.

## Docking, tabs and tear-off

- **A floating dock panel's title bar cannot start a re-dock drag on
  Wayland**: `dock.Host.dragFloatingPanel` falls back to the desktop's move, so
  dropping it on the host only moves the window. The dock button works.
- **`Host.DockPanel` can restore a panel at zero width** when its area's split
  weight collapsed while it floated: right in the layout JSON, invisible on
  screen.
- **On X11 a torn-off window is placed by the window manager**, not under the
  pointer, even though `DragsWindows()` reports true; KWin also clamps a
  client-placed window to the screen.
- **A tab merged into another application carries its title only**; richer
  data needs the app's own type through `OnTabDrag`.
- **Escape during an X11 drag is unproven on real hardware**: under Xwayland
  KWin takes the keyboard for the drag it mirrors. Both cancel paths are
  written; check on a real X server.

## Drag and drop

- **Wayland's "ask" action is tested headlessly only** — no source available
  in the rig (Qt, KIO) ever sets it.

## Widgets

- **`ListView.Measure` returns exactly rows × row height**, so inside a scroll
  view the view frame's border clips the last row.

## Platform

- **`Position` and `Move` are still device pixels** while sizes are logical;
  converting them is the natural follow-up to the size fix.
- **X11's scale is connection-wide**: the last window's `WindowOptions.Scale`
  sets it for the display (X11's own model, one `Xft.dpi`).
- **`UITK_SCALE` on a Wayland output at scale 1** draws at the asked scale into
  a 1× buffer. Use a real output scale instead.
- **Platform facts, not bugs**, that apps must live with: Wayland windows
  cannot place themselves (Minim's snapping windows cannot follow there); a
  window manager places a window when it maps it; an X11 move is answered a
  frame or two later.

## Renderer (paintengine2d)

- ~~**The bilinear sampler returns transparent outside an image** instead of
  clamping, which seams a nine-slice; skins work round it with a replicated
  1-texel border on every piece.~~ Clamps to the edge on the CPU as the GPU
  always did (paintengine2d `fix/sampler-clamp`); the skins' border is gone.
- **Multisampled dest-out is not sample-exact** on the GPU; the shapes work
  wipes whole pixels first for that reason, and anything else that erases
  through an image will meet it.

## Tooling

- **`tools/e2e/crop.py` assumes filter-0 PNG rows**, so it garbles the
  toolkit's own `WritePNG` output (KWin's screenshots are fine).
- **The tour's `-shot` renders one page per window**; there is no combined
  contact sheet.
