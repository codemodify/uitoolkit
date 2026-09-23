# Skins

A **theme** in uitoolkit is a pack plus an engine: colours and metrics as
data, shapes as Go code ([Theme engines](theme-engines.md)). A **skin** is
the other answer to the same question — the one WinAmp, VLC's skins2 and
Windows Media Player gave. It is a theme made of pictures: sprite sheets,
named sub-rects of them, and a table saying which sub-rect paints which part
of which control.

A skin is a pack of the `skin` engine. It lists in Settings beside the other
121, it is selected by the same `look.json` preference, `UITK_THEME=nocturne`
runs any app in one, and editing it applies live. There is no skin mode and
no second picker.

```
style/skins/<name>/skin.json        skins the toolkit ships (embedded)
~/.config/uitoolkit/skins/<name>/   a skin you install
    skin.json                       the manifest
    art/chrome.png                  the sheets, one file per scale
    art/chrome@2x.png
~/.config/uitoolkit/skins/<name>.uskin   or the same thing as one zip
```

Eight skins ship with the toolkit. Three are the format's worked examples:
**Nocturne**, amber on charcoal, drawn from paths so it is exact at every
scale; **Cassette**, a six-colour pixel skin with two-pixel bevels that
exercises the `pixelated` path; and **Deck**, whose window is not a
rectangle and whose buttons take the pointer only on their ink. Five more
are worn by the player demos ([docs/players.md](players.md)): **Minim**,
and the two panels it switches to, **Minim Classic** and **Minim Silver**;
**Marquee**; and **Lantern**. All eight are generated — see
[The demo skins](#the-demo-skins).

## The three rules

Everything below follows from these, and they are what the formats this one
takes its ideas from did not have.

**1. Every number is in the art's own design pixels.** A pack states its
metrics at 1× and the toolkit scales them; a skin states its sprite rects,
slices and paddings the same way. The toolkit picks the asset at or above the
display scale and draws it *down* — art is never enlarged — so 1.25, 1.5 and
1.75 are exact rather than soft. WinAmp's 275×116 was frozen at one size
forever because its format had no such unit.

**2. A skin replaces painting, never behaviour.** There is no scripting
language, no bytecode and no expression of any kind in the format, and the
parts a skin may bind are a fixed table. A skinned button is still a
`widgets.Button` — it has a name, a role, a keyboard route, an accessibility
node and a focus ring — that happens to be painted from a sprite. Tab reaches
it, Orca reads it, a magnifier follows it. A skin whose window is not a
rectangle and whose button is a disc ([Shape](#shape)) is no exception: the
art says where those are, and everything that is not a pointer — the tab
order, the mnemonics, the accessibility tree — never looks at it.

**3. A skin is always a partial override.** Anything it does not describe is
painted by its **base pack's own engine**, so a skin over `win95` keeps
Win95's bevels on the controls it never drew and a skin over `breeze-night`
keeps Breeze's. A half-finished skin is a coherent app in a real look; drop
the skin at run time and the app keeps working.

## The format

`skin.json`, beside the art. The only required key is `skin`, the format
version; everything else has a default.

```jsonc
{
  "skin": 1,                    // format version (required)

  // Identity — exactly what Settings lists for any other pack.
  "label": "Nocturne",
  "year": 2026,
  "lineage": "uitoolkit",
  "summary": "The toolkit's own skin: amber on charcoal.",
  "family": "dark",             // dark | light
  "base": "breeze-night",       // the pack painted under the art

  // The grid the art was drawn on. Every number below is in these pixels.
  "design": { "scale": 1, "pixelated": false },

  // ---- the art --------------------------------------------------------
  "sheets": {
    "chrome": {
      "1x": "art/chrome.png",
      "2x": "art/chrome@2x.png",
      "pixelated": false        // overrides design.pixelated for this sheet
    }
  },

  // Named sub-rects: WinAmp's fixed offsets, VLC's <SubBitmap>.
  "sprites": {
    "button.normal": {
      "sheet": "chrome",
      "at": [0, 0, 56, 30],     // x, y, w, h
      "slice": [9, 12, 9, 12],  // top, right, bottom, left — nine-slice
      "middle": "stretch",      // stretch | tile | none
      "tint": false             // true: a white glyph painted in the label's colour
    }
  },

  // ---- text roles -----------------------------------------------------
  // One place to change every label of a family of controls.
  "text": {
    "control": {
      "color": "#e6e9f2", "hover": "#f2f5fb", "pressed": "#9aa2b6",
      "disabled": "#5b6273", "checked": "#e6e9f2",
      "default": "#1a1305",   // the ink on the default button's accent face
      "size": 14, "bold": false,
      "case": "upper"         // set in capitals: painting only (see below)
    }
  },

  // ---- the bindings ---------------------------------------------------
  "parts": {
    "button": {
      "states": {
        "normal": "button.normal", "hover": "button.hover",
        "pressed": "button.pressed", "disabled": "button.disabled",
        "focus": "button.focus", "checked": "button.checked",
        "default": "button.default"
      },
      "text": "control",        // a role name, or an inline object
      "pad": [0, 0, 0, 0]       // extra room the art needs, design pixels
    },

    // The shorthand every real sheet is drawn for: a run of equal cells.
    "tool": {
      "strip": {
        "sheet": "chrome", "at": [0, 96, 56, 30], "gap": 2,
        "slice": [9, 12, 9, 12],
        "states": ["normal", "hover", "pressed", "-", "checked"]
      }
    }
  },

  // ---- the window -----------------------------------------------------
  "window": {
    "border": [1, 1, 1, 1],     // top, right, bottom, left — or
                                // { "caption": […], "content": […] }
    "caption": 34,              // any height, down to 8: below 24 the
                                // caption buttons are square and centred
    "captionGap": 0,            // room between the caption and the content
    "radius": [7, 7, 0, 0],     // top-left clockwise
    "layout": ":minimize,maximize,close",
    "buttons": "right",         // the side the art puts them on (else the desktop's)
    "title": { "align": "center" },   // or "start", with an "inset"
    // The silhouette the window is cut to: a union of rounded rects that
    // stretch with it or pin to an edge, or { "art": "<sprite>" } for a
    // skin whose window is a picture. See "Shape" below.
    "shape": [
      { "at": [0, 0, 0, 46], "radius": [24, 24, 0, 0], "stretchX": true },
      { "at": [18, 40, 18, 0], "radius": [0, 0, 22, 22],
        "stretchX": true, "stretchY": true }
    ],
    // Windows the app gives a role, dressed differently. See "Windows
    // that are not alike" below.
    "variants": {
      "minim.equaliser": { "caption": 13, "parts": { "caption": { "…": "…" } } }
    }
  },

  // ---- fixed layouts --------------------------------------------------
  // Named slots at design coordinates that an app binds its own widgets
  // to. See "Fixed layouts" below.
  "layouts": {
    "minim.strip": {
      "size": [273, 101], "art": "main.face",
      "slots": {
        "play": { "at": [38, 70, 21, 21],
                  "art": { "normal": "key.play", "pressed": "key.play.down" } },
        "display": { "at": [4, 3, 265, 54] }
      }
    }
  },

  // ---- what art does not cover ----------------------------------------
  // theme.json's own keys, read by theme.json's own code.
  "colors":  { "text": "#e6e9f2", "accent": "#e8a33d", "…": "…" },
  "metrics": { "controlH": 30, "checkbox": 20, "…": 0 },
  "params":  { "scrollArrows": 0 },
  "fonts":   { "ui": ["Inter"], "mono": ["JetBrains Mono"] },
  "bevel":   "none"
}
```

A sprite may also be written inline wherever a sprite name is expected, and a
sheet's sole sprite may omit `"sheet"`. `style/skins/nocturne/skin.json` is
the full worked example; it is 621 lines and every one of them was generated
from `skingen/nocturne.go`.

### Validation

A skin is a stranger's zip file, so the loader is strict and every refusal
names the key it came from:

```
nocturne: skin.json: parts.button.states.hover: no sprite "btn.h"
nocturne: skin.json: sprites.face.slice: leaves no middle: 4+4 wide and 4+4 tall in a 8×8 sprite
nocturne: skin.json: parts.widget: "widget" is not a part of this toolkit
nocturne: skin.json: sheets.chrome.1x: "../../etc/passwd" leaves the skin's directory
nocturne: skin.json: skin: format version 99 is newer than this build reads (1)
```

An unknown key is an error, not a silent drop: a manifest that half-applies
is worse than one that refuses and says why. A key a skin may only name from
inside its own package is checked too — a manifest cannot reach outside its
own directory.

A broken skin never takes an app down. It is skipped by the scanner, so it
does not list and does not load, and everything else still works.

## Parts

A skin binds *parts*, not controls. The first thirteen are the engine's
`Role` faces, so binding one changes every control that paints it — `button`
is also the scrollbar's step buttons, `bar` is the menu, tool and status bars.
That is why a skin needs a dozen sprites rather than two hundred.

| part | what it paints |
| --- | --- |
| `button` | push buttons, dialog buttons, scrollbar step buttons |
| `tool` | tool bar buttons and free toggles |
| `field` | text fields and text areas |
| `check` | the check box and radio well |
| `row` | list, tree and table row highlights |
| `tab` | tabs |
| `thumb` | scrollbar thumbs |
| `track` | scrollbar and slider tracks |
| `menu` | the hot menu item and open menu title |
| `combo` | combo boxes |
| `splitter` | splitter handles |
| `bar` | menu, tool, status bars and tab strips |
| `panel` | panels, cards and group boxes |
| `check.mark`, `radio.mark` | the tick and the dot inside them |
| `arrow.up/down/left/right` | arrows (scroll, spin, sort, combo) |
| `expander.open`, `expander.shut` | tree and accordion disclosures |
| `slider.track`, `slider.fill`, `slider.thumb` | a slider |
| `progress.back`, `progress.fill` | a progress bar |
| `switch.track`, `switch.knob` | a switch |
| `focus` | the keyboard focus ring |
| `window` | the window background |
| `caption` | the top-level window's caption band |
| `caption.button` | its close, maximize and minimize buttons (else `tool`, else `button`) |
| `caption.title` | a plate behind the caption's title, as wide as the title: the gap in a ribbed band |
| `menu.frame`, `tooltip` | the frames those float on |

`style.SkinPartNames()` is the same list at run time; a skin that names
anything else fails to load. New controls come from the toolkit, not from a
skin — otherwise a skin is a program, and running a stranger's program from a
zip file is a different project with a different threat model.

## States, and the fallback that makes a terse skin work

A part carries art per state. The state names are the toolkit's own
`ControlState` bits:

```
normal  hover  pressed  disabled  focus
checked  checkedHover  checkedPressed
default  defaultHover  defaultPressed
inactive
```

A state with no art of its own resolves along a fixed chain, so **art for
`normal` alone gives a control every state** and each extra sprite sharpens
one more:

| state | falls back to |
| --- | --- |
| `hover` | `normal` |
| `pressed` | `hover`, `normal` |
| `disabled` | `normal` |
| `focus` | `hover`, `normal` |
| `checked` | `pressed`, `hover`, `normal` |
| `checkedHover`, `checkedPressed` | `checked`, `pressed`, `hover`, `normal` |
| `default` | `hover`, `normal` |
| `defaultHover` | `default`, `hover`, `normal` |
| `defaultPressed` | `default`, `pressed`, `hover`, `normal` |
| `inactive` | `disabled`, `normal` |

The state a control is in is read in the order a person would read it:
disabled first (it outranks everything), then the on/off axis, then the
default button, then the pointer, then focus, then a backdrop window.

The default button keeps its own face under the pointer rather than taking
the ordinary hover one — its art is the accent, and the one button a dialog
is steering you to should not look like the others while you are reaching for
it. That is why it has its own hover and pressed states, exactly as `checked`
does. Its ink is the `default` colour of its text role, and that colour
applies only where the part actually drew a default face: `StatePrimary`
reaches more parts than the button, and dark on-accent text on an ordinary
field would be unreadable.

`normal` is required as soon as a part names any art at all.

### Bevels

A skin that binds any of the thirteen `Role` faces gets `"bevel": "none"`
unless it states otherwise, whatever its base pack says. A bevel language is
an instruction to the stock painter — shift a pressed button down a pixel,
draw arrow wells at the ends of a scrollbar — and a skin that paints its own
faces already said all of that in its art. Leaving the base's bevel in place
makes the stock painter add it *on top of* the pictures. A skin that binds no
faces at all keeps its base's bevel, so a skin describing nothing is still
exactly its base pack.

## The fallback rules

There are two, and the second is the subtle one.

**A part with no art falls to the base pack's engine.** `Face`,
`CheckIndicator`, `Arrow`, `MenuHighlight` and the rest ask the skin first and
call the base engine's own method when it has nothing. A skin over `win95`
gets Win95's bevels, over `aqua` gets Aqua's gel.

**A whole control whose parts the skin *does* bind is painted by the stock
engine instead.** An era engine is free to paint a control directly rather
than assembling it from parts — Breeze 6 draws its own menu row, its own
button and its own check box, because that is how those differ from the stock
ones. Handing such a control to the base engine would paint it in the base
look even though the skin has art for the part it is made of, and you would
get a skinned app with Breeze's blue menu highlight in it. So when the skin
binds any part a control is made of, the control goes to `BaseEngine`, which
re-dispatches every part through the look's engine — the skin's art is used
and the base's layout, text, mnemonics and focus handling are kept.

Two consequences worth knowing:

- Binding `menu` changes menu items *and* menu-bar titles, because both are
  assembled from the same part.
- Binding `thumb` switches scrollbars to a classic gutter bar in the skin's
  own `scroll` metric, because a modern base pack's transient overlay bar
  would hide the art it was drawn for. `"params": {"scrollArrows": 1}` adds
  step buttons.

## Focus

**A skin may re-draw the focus ring. It may never remove it.**

`DrawFocusRing` paints the skin's `focus` sprite if it has one and the base
engine's ring if it does not — one or the other, always. A part may also give
its face a `focus` state, which changes how the control looks when the
keyboard is on it; the ring is still drawn over the top. There is no way to
express "no focus marking", because a control nobody can see the focus on is
a control nobody can drive from a keyboard.

The existing `TestFocusIsVisibleInEveryLook` runs over every registered pack,
skins included, and fails a look whose focused and unfocused controls differ
by fewer than eight pixels.

## HiDPI

This is where bitmap skins historically went soft. Four mechanisms, in
priority order:

**1. Draw the art from paths.** Nocturne's sheets are gradients and rounded
rects rasterised at each scale, so 1.25 / 1.5 / 1.75 / 2 are exact. This is
the only fully correct answer and it is why the reference skin is drawn this
way.

**2. Scale sets, chosen upward.** `1x`, `1.5x`, `2x`, `3x` — the toolkit picks
the nearest asset **at or above** the target and downscales. At 1.75 the `2x`
asset is drawn at 0.875. Art is never enlarged, which is what keeps edges from
haloing. `style/iconset.go` has picked `@2x` icons by the same rule for years.
A shipped skin must cover 1× and 2×.

**3. Nine-slice with whole-pixel edges.** A slice's fixed corners are the part
that shows softness and they are small: they are drawn at the asset's own
scale snapped to whole device pixels, and the stretchable middles absorb the
fraction. A 1px highlight in a corner tile survives; the same highlight
stretched across a whole button does not. A box too small for its own corners
shrinks them proportionally rather than letting them overlap.

**4. `"pixelated": true`.** A deliberately pixel-art skin is sampled
nearest-neighbour at **integer** multiples only: at 1.75 the 2× art is drawn
at 2× and fitted, while text, layout and hit regions stay at the true 1.75. A
crisp 2× sprite in a 1.75 window reads as pixel art; a bilinear 1.75
enlargement of a 1× one reads as a mistake. This is a skin author's choice,
not a toolkit guess — and a pixelated sheet's 2× asset should be its 1× asset
with every pixel doubled, which is what Cassette's is.

What the toolkit refuses to do is silently bilinear-upscale a 1× sheet to 1.75
and call it supported.

**5. A panel's pixel pieces share one device grid.** A pixel sprite an app
draws at its own size — a panel's key, a digit, a whole face
([Sprites an app paints itself](#sprites-an-app-paints-itself),
[Fixed layouts](#fixed-layouts)) — is magnified by the display scale
itself, nearest, onto the device grid: a texel whose left edge is at
device x covers the pixels whose centres fall in [x, x + scale). At 1.75 a
texel is two pixels three times in four and one the fourth, because 1.75
pixels do not exist; what the grid decides is that every piece agrees which
— so a key continues the face it is a hole in, with no double-width or
missing column where two sprites meet. The piece is magnified on the CPU
and blitted one to one at a whole pixel, so the GPU paints the same hard
edge. A control face keeps rule 4.

That is as exact as the toolkit can make it, and on X11 and offscreen it is
what reaches the screen. **On Wayland at a fractional scale the compositor
has the last word**: a surface's size is stated in whole logical pixels, and
a window whose logical size times the scale is not a whole number — Minim's
275 × 1.75 = 481¼ — is stretched by that quarter pixel when it is composited.
KWin's screenshots of Minim Classic at 1.75 show it as a one-pixel softening
along some key edges that the same build on X11 at 1.75 does not have. A
window sized to a multiple of four logical pixels would avoid it; the
players keep their design sizes, and the docs say so rather than hiding it.

### One implementation note that matters

paintengine2d's bilinear sampler reads the two texels around each sample.
Past the edge of an image it **clamps** — reads the edge texel — on the CPU as
the GPU (GL_CLAMP_TO_EDGE) always did. It used to return transparent there on
the CPU, and a nine-slice's middle, which is almost always stretched wider
than its source, faded over its outermost half texel: a pale vertical line
down every button, exactly where the middle meets the fixed edges. The skins
worked round it with a one-texel border replicated round every cut piece.
The sampler is fixed and the border is gone; `TestSkinNineSliceHasNoSeam`
fails against a renderer that does not clamp.

Every sprite, and every one of a nine-slice's nine pieces, is still cut into
an image of its own. The clamp is at the edge of the *image*, not of a source
rect, so that is what keeps a blit from reaching a neighbouring sprite on the
sheet — the other artefact an atlas would otherwise have.

## Shape

A skin's window does not have to be a rectangle, and its controls do not have
to take the pointer across their whole box. Both come from the same place a
skin's everything else comes from — the manifest and the art — and both are
cut by the shaped-window machinery in [docs/shapes.md](shapes.md).

### The window

`window.shape` is the silhouette. It takes two forms, because skins come in
two kinds:

```jsonc
// A union of rounded rects in design pixels. Each says whether it stretches
// with the window: under stretchX the third number is a margin from the
// right edge rather than a width, and under stretchY the fourth is a margin
// from the bottom. Under fromRight / fromBottom a fixed-size rect is pinned
// to the far edge instead, its first (second) number the margin from it.
"shape": [
  { "at": [0, 0, 0, 46], "radius": [24, 24, 0, 0], "stretchX": true },
  { "at": [18, 40, 18, 0], "radius": [0, 0, 22, 22], "stretchX": true, "stretchY": true },
  { "at": [12, 0, 60, 20], "radius": [6, 6, 0, 0], "fromRight": true }  // a tab at the top right
]

// Or: the alpha channel of a sprite *is* the outline.
"shape": { "art": "window.normal" }
```

Reach for the rectangles. They are resolved against the window at the display
scale, so the outline is redrawn at every size and exact at 1.25, 1.5 and
1.75 rather than stretched, and the window can still be dragged to any size.
Two rects that overlap are one silhouette — the union is filled nonzero — so
a tab meets the body under it with no seam.

The form was chosen over an SVG path string deliberately: nothing in the
toolkit parses path strings, and a rect union is already what both consumers
want — a compositor input or opaque region *is* a rect list, and hit-testing
one is a loop rather than a rasterisation.

The `art` form is for the skin the rectangles cannot describe: the fixed
panel whose edge is in the picture and nowhere else, which is the shape a
WinAmp region file was. It is a bitmap, so it resamples at any size but its
own — the trade the art made when it was drawn.

Two things follow from the silhouette being a *frame's*:

- It is dropped while the window is maximized or tiled, and comes back on
  restore, exactly as the frame's corners and shadow are. An uncomposited X11
  screen keeps it, hard-edged, because XShape is the only thing that works
  there at all.
- **Keep the content inside it.** `window.border` is what does that: the
  content box is the window less the border, so a skin whose body is drawn in
  from the window's edge states a border at least that wide. Deck's body is
  18 design pixels in and its border is 26, which leaves eight pixels of
  shell between the content and the cut. A silhouette wide at the top and
  narrow below states the border as `{"caption": [top, right, 0, left],
  "content": [0, right, bottom, left]}` — the band keeps the first, the
  content under it the second — and `captionGap` leaves room between the
  two, which a silhouette can cut away: a window in two pieces with a slot
  of desktop between them. Maximized, both are dropped with the border.

A silhouette that is the plain window box, rounded no more than
`window.radius` already rounds it, is not a silhouette and is not used. It
would describe nothing the frame does not do already, and it is not free: a
shaped window's input region *replaces* the resize band in its shadow margin,
so it would be resized from inside its own edges instead of from the band
outside them. Nocturne's `shape` is exactly that, and Nocturne is still a
rectangle.

### The controls

A skinned control is where its art is. Nothing is declared for this: the
engine reads the alpha of the sprite the control's face is painted from, and
a press outside the ink falls through to whatever is behind — so a round
button stops taking clicks in its corners.

The mask is taken by **painting** the sprite into a scratch the size of the
control, not by reading the sheet. A face is a nine-slice, and where its ink
lands depends on how its corners and middles were fitted to that particular
box; only the painter knows that. It is cached per sprite, size and scale
beside the cut sheets, and dropped with them when a PNG is re-saved.

The coverage threshold is near the bottom rather than at half, on purpose.
The question it answers is "did the art mark this pixel at all", because what
it is for is refusing the *empty* corners of a round face; half would make a
face drawn as a pale wash — a ghost button, a glassy toggle — entirely
unclickable, and rounding this far out rounds an antialiased edge outwards,
which is the right way to round for input. Art that fills its box solidly
says so by saying nothing and keeps the box it always had, which is the
common case and the cheap one; art that could not be drawn at all keeps it
too, because a control nobody can click is a worse failure than a square one.

Only a component whose box *is* one face asks — `widgets.Button` today. A
check box is its indicator and its label, and shaping the pair by the
indicator's art would make most of the control deaf.

A control an app paints from art by name rather than from a face — a key a
panel paints with `DrawSkinSprite`, or with a slot's art — says so with
`widget.ArtShape`, and its hit shape is that art's, taken the same way:
`style.SkinSpriteShape` for a sprite drawn where the app draws it,
`style.SkinSlotShape` for a slot's resting art. Minim Silver's round keys
take the pointer on their ink by it. An art shape comes before a face's, and
a nil one falls back to the face and then to the box.

## Sprites an app paints itself

A part is the only thing the engine paints from a skin, and the parts are a
fixed table. A sprite that no part binds was always allowed in a manifest;
it is simply one that only an app knowing its name will ever draw. Two calls
are that app's way in:

```go
style.DrawSkinSprite(lk, ctx, box, "led.8", paintengine2d.Color{}) // false: not a skin, or no such sprite
w, h, ok := style.SkinSpriteSize(lk, "font.41")                     // design pixels
```

They paint by exactly the rules a part is painted by — the asset chosen
upward, nine-slice on whole pixels, nearest at whole multiples for a
pixelated sheet, the tint for a glyph marked `tint` — and they add nothing
to the format: a look that is not a skin, or a skin without that sprite,
answers false and the app paints its own. That is what keeps a panel drawn
this way a partial override like everything else.

Minim's two panel skins are built on them (docs/players.md). Their faces
are pictures of whole windows with the wells and printed labels in them,
and their digits, lamps, thumbs and bitmap capitals are loose sprites the
player paints itself, where the skin's [fixed layout](#fixed-layouts) says.

A pixel sprite drawn at its own size is magnified onto the device grid
([HiDPI](#hidpi), rule 5), so a panel drawn from these scales evenly with
nothing special in the art. (The panels used to carry an empty one-pixel
margin round every sprite, sliced there, to get the same stretch out of the
nine-slice painter; the margin could not put neighbouring pieces on one grid,
and it is gone.)

A sprite an app paints has a hit shape too (`style.SkinSpriteShape`,
[The controls](#the-controls)).

And it has somewhere to be painted. `widgets.Button` and `widgets.ToolButton`
take a `Painter` and a `Shaper` — and a `Content`, for the key that wants
the look's own face with the app's mark on it rather than a picture instead
of it — and `widgets.Slider` a `Painter` and a `Travel`, so the art above
goes on an ordinary widget rather than on a component the app has to write
itself
([docs/widgets.md](widgets.md#art-on-a-control)). The widget is unchanged
underneath: the same focus, keys, tooltip and accessibility tree, with the
skin's picture in place of the look's face — and the focus ring over it,
because a skin never takes that away.

A list is the other half of that: `widgets.ListView.RowGeo` takes the rows
rect, the groove and the row height from a layout's slots, so a playlist on
a panel is a list view on the skin's grid rather than a component the app
has to write. `RowPaint` and `ScrollPaint` are the paint that goes with that
geometry — the skin's own type in the skin's own ink, and a thumb that is a
loose sprite riding a printed groove — because a skin engine hands a list
row and a scroll bar to the pack underneath it, which is right for a form
and wrong for a panel. `ItemDetail` puts a length at the row's right-hand
end
([docs/widgets.md](widgets.md#a-list-on-a-skins-grid)).

## Fixed layouts

A skin re-skins ordinary widgets at ordinary layout, and for a form or an
editor that is the whole story. A player is not a form: WinAmp's 275×116 and
VLC's `<Layout>` were panels — a picture with holes in it, a control in each
hole — and a panel's proportions are its design. `"layouts"` is that half of
the format, with the rule that makes it safe kept:

**A skin places. It never adds.**

```jsonc
"layouts": {
  "minim.strip": {
    "size": [273, 101],            // the box it was drawn for, design pixels
    "art": "main.face",            // painted behind the slots (optional)
    "slots": {
      "play":    { "at": [38, 70, 21, 21],
                   "art": { "normal": "key.play", "hover": "key.play.hover",
                            "pressed": "key.play.down" } },
      "digit.0": { "at": [44, 11, 10, 14] },
      "badge":   { "at": [4, 2, 16, 10], "fromRight": true },
      "foot":    { "at": [0, 6, 0, 8], "stretchX": true, "fromBottom": true }
    }
  }
}
```

A slot is a rect in the silhouette's vocabulary — pinned to the top left,
pinned to the right or the bottom, or stretching — and may carry art per
state, resolved along the same fallback chain a part's states are, so a key
drawn at rest and held down still has a face when it is hovered or switched
on. A slot art that is one sprite name is that sprite in every state.

The *app* binds its own components to slot names and the toolkit places
them:

```go
slots := widget.NewSlots("minim.strip").
	Bind("display", readout).
	Bind("play", play).
	Bind("stop", stop)
if slots.Available(lk) {
	for _, c := range slots.Arrange(lk, box) {
		// the skin has no slot for c: the app's to hide or to place itself
	}
}
```

What that keeps, and the tests pin:

- **No behaviour.** A slot has a rect and art. There is no action, route or
  expression in the format; a key it does not know is refused by name.
- **No reordering.** Placing moves components and nothing else, so the tab
  order and the accessibility tree are still the order the app added its
  components in.
- **No hiding.** A component whose slot the skin leaves out is handed back
  to the app. Whether a control exists is the app's decision; a skin that
  could take one out of the tab order would be deciding what the app can do.
- **A partial override.** A look with no such layout — any pack that is not
  a skin drawn for the app — places nothing, and the app lays itself out its
  own way.

`style.SkinSlotRect` is a slot's rect in a box (for what the app prints into
— a clock's digits, a lamp — as well as what it places), `SkinLayoutSize`
what a panel laid out by it measures, `DrawSkinLayout` and `DrawSkinSlot` its
art, and `widget.SlotArtRect` the slot's own unrounded rect in a component's
coordinates: a component's bounds are whole pixels and a slot at 1.75 is not,
and a key painted into its rounded box would sit half a pixel off its hole.

Minim's two panel skins state three layouts each, `minim.strip`,
`minim.equaliser` and `minim.playlist`, generated from the same rects the
faces are drawn round (`skingen/panel`); the player reads
every rect it uses from the skin and carries none of its own.

## Windows that are not alike

A skin states one frame for the windows it dresses, and an app can say two
things about one of its windows that only the app knows:

```go
eq.SetFrameRole("minim.equaliser") // what this window is
compact.SetCaptionHeight(22)       // a caption of its own height, logical pixels
```

A skin gives a role a frame of its own under `window.variants`:

```jsonc
"variants": {
  "minim.equaliser": {
    "caption": 13,
    "title": { "align": "start", "inset": 18 },
    "parts": {
      "caption":       { "states": { "normal": "eq.caption" }, "text": "tab" },
      "caption.title": { "states": { "normal": "eq.tab" } }
    }
  }
}
```

A variant restates any of the window's keys — its border, caption, gap,
radius, shape, layout, button side, title placement — and may rebind the
frame's own parts, and only those: `caption`, `caption.button`,
`caption.title` and `window`. A variant that could rebind a button would be
a second skin. What it does not state is the window's. A role the skin has
no variant for, and a window with no role, wear the window's own frame, so a
skin can never dress a window the app did not name.

Minim Silver's equaliser is the worked example: its header is a tab with its
name on it and no title band, which is a 13-pixel caption of pale chrome whose
title plate is the tab, set from the start. A plate's slices are its ends
and the words are set in its middle, so a tab with a long curve at its right
end stops its title short of the curve.

`SetCaptionHeight` works in every look, not only skins: `DecorationOf`
applies it and the caption's buttons shrink to stand in the band — square,
centred — rather than growing it back. A merged caption that holds the app's
row is not held to its usual minimum when the app asked.

Two more window keys belong here. **`buttons`**, `"left"` or `"right"`, is the
side a skin's art puts the caption buttons on: they are still the desktop's
buttons, in the desktop's choice of which, moved to that side and turned
round so the outermost stays outermost. A silhouette that bites into one top
corner can say so and keep its close button. **`"case": "upper"`** on a text
role sets it in capitals — Minim's panels print their titles that way, as the
era did — and it is painting only: the window keeps the title the app gave
it, and that is what a screen reader and the desktop's window list read.

## What a skin cannot do

**Carry behaviour.** Not a gap: a decision. No scripting, no bytecode, no
action vocabulary. A layout places an app's controls and a variant dresses
an app's window; anything an app needs beyond painting and placing it writes
in Go.

**Lay out a panel it was not drawn for.** A fixed layout is a promise
between one skin and one app, by the app's names; a skin cannot place the
controls of an app that does not bind them, and an app that binds them in a
skin with no such layout lays itself out its own way.

**Leave its window.** Menus and tooltips are drawn inside the window they
open from, so a shaped skin's menu is a rectangle, and a strip 116 pixels
tall holds a four-row menu (docs/players.md).

## Writing one

1. Draw a sheet. One PNG with your control faces laid out in a grid, and a
   second at twice the size. Leave a couple of pixels between cells.
2. Write `skin.json`. Start with `skin`, `label`, `base`, one `sheets` entry
   and one `parts` entry for `button` with a `normal` state. That is a
   working skin: every other control is your base pack's.
3. Put it in `~/.config/uitoolkit/skins/<name>/` and run any app with
   `UITK_THEME=<name>`, or pick it in Settings.
4. Add parts one at a time. Each one you bind is one more control that is
   yours; each one you leave is one more that is your base pack's, and
   correct either way.
5. Lint it: `go run ./cmd/uitk-skin lint ~/.config/uitoolkit/skins/<name>`
   (or a `.uskin`, or a skin's name). It prints the loader's refusal keyed by
   the manifest key it came from, or — for a skin that loads — what you would
   want to hear before shipping: states a control falls back from, sprites
   nothing binds, art too small for 2×. `-strict` fails on warnings too.

Edit the manifest while an app is running and it re-applies: the look watcher
already stamps the pack's file, and a skin's manifest is that file. A re-saved
PNG shows up about a second later, on the same cache TTL as icon files.

To ship it as one file, zip the directory and name it `<name>.uskin`. An
archive whose manifest sits in a single top-level folder loads too, since that
is what most archivers produce.

## Writing one in Go

Hand-painting is the way in, not the way up. Every pack the toolkit ships is
*drawn from code* by the `skingen` package, and that package is public API —
`github.com/codemodify/uitoolkit/skingen` — for the plain reason that a skin
somebody else writes should be able to be as good as the ones in here. What
it buys, in order of how much it matters:

- **Both scales are the same drawing.** You write a cell once; the 1× and the
  2× sheet come out of it, and a 3× set is one entry in `skingen.Scales`.
- **The manifest cannot drift from the art.** The cell layout is stated once,
  in Go, and both the PNGs and `skin.json` are emitted from it. No rect in
  the manifest is ever off by the two pixels you moved a sprite by.
- **The pack is regenerable, so it is reviewable.** Your art is a diff of Go
  lines rather than of binary blobs, and anyone can rebuild the PNGs from it
  and check.
- **Provenance is clean.** Every pixel came from a path you wrote. For this
  repository that is a licensing requirement (facts about a look — metrics,
  colour values, described behaviour — never anybody else's code or bitmaps);
  for you it is the difference between art you can answer for and art you
  found.

### The smallest one that works

A *plan* is a skin: sheets of cells that know how to draw themselves, plus
the bindings that say which sprite paints which part in which state. This is
a whole program, and it writes a whole skin.

```go
// mkskin writes the smallest skin there is into ~/.config/uitoolkit/skins.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/skingen"
)

func main() {
	face := func(fill, edge string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			r := paintengine2d.XYWH(0.5, 0.5, w-1, h-1)
			ctx.DrawRoundRect(r, 6, 6, paintengine2d.Fill(skingen.Hex(fill)))
			ctx.DrawRoundRect(r, 6, 6, paintengine2d.StrokePaint(skingen.Hex(edge), 1))
		}
	}

	p := &skingen.Plan{
		Name:  "mine",
		Label: "Mine",
		Base:  "breeze-night", // everything this skin does not bind
		Sheets: []*skingen.Sheet{{
			Name: "chrome", W: 96, H: 32,
			Cells: []skingen.Cell{
				{Name: "button.normal", X: 0, Y: 0, W: 32, H: 32,
					Slice: [4]int{6, 6, 6, 6}, Draw: face("#40444c", "#0b0e12")},
				{Name: "button.hover", X: 32, Y: 0, W: 32, H: 32,
					Slice: [4]int{6, 6, 6, 6}, Draw: face("#4c515b", "#0b0e12")},
				{Name: "button.pressed", X: 64, Y: 0, W: 32, H: 32,
					Slice: [4]int{6, 6, 6, 6}, Draw: face("#2b2f36", "#0b0e12")},
			},
		}},
		Parts: []skingen.PartBinding{{
			Part: "button",
			States: [][2]string{
				{"normal", "button.normal"},
				{"hover", "button.hover"},
				{"pressed", "button.pressed"},
			},
			Text: "button",
		}},
		Text: []skingen.TextRole{{Name: "button", Color: "#e8eef2"}},
	}

	home, _ := os.UserHomeDir()
	if err := skingen.Write(filepath.Join(home, ".config", "uitoolkit", "skins"), p); err != nil {
		log.Fatal(err)
	}
}
```

`go run ./mkskin` writes `mine/art/chrome.png`, `mine/art/chrome@2x.png` and
`mine/skin.json` — the manifest of the first half of this document, generated:

```json
{
  "skin": 1,
  "label": "Mine",
  "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/chrome.png", "2x": "art/chrome@2x.png" } },
  "sprites": {
    "button.normal": { "sheet": "chrome", "at": [0, 0, 32, 32], "slice": [6, 6, 6, 6] },
    …
  },
  "parts": { "button": { "states": { "normal": "button.normal", … }, "text": "button" } },
  "text": { "button": { "color": "#e8eef2" } }
}
```

(The real file is sorted by key, which is why a change to a plan diffs as one
line rather than as a reshuffle.) Then run anything with `UITK_THEME=mine`,
and lint it the same way you would lint a painted one:

```bash
go run ./cmd/uitk-skin lint ~/.config/uitoolkit/skins/mine
# mine: skin.json: parts.button.states: warning: no "disabled" art: it shows the "normal" face
```

Which is the whole loop: **fork or write a plan → `Write` → lint → run.**
Change a colour, re-run, and the look watcher re-applies it about a second
later in the app you left running.

### Starting from one of ours

Starting from a blank sheet is not the only way in, and usually not the best
one. `skingen.Plans()` returns all eight shipped plans — `Nocturne()`,
`Cassette()`, `Deck()`, `Minim()`, `Marquee()`, `Lantern()`,
`MinimClassic()` and `MinimSilver()` — as data, freshly built on every call,
so taking one and changing it cannot disturb the pack it came from:

```go
p := skingen.Nocturne()
p.Name, p.Label = "nocturne-teal", "Nocturne Teal"
p.Colors["accent"] = "#3fd0c9"
skingen.Write(dir, p)
```

That is a pack of 65 sprites at two scales, yours, in four lines. From there:
rebind a part (`p.Parts`), redraw one cell (find it in `p.Sheets` by name and
replace its `Draw`), or copy the plan's source file out of `skingen/` into
your own package and edit it as a drawing — it is permissively licensed and
that is what it is there for. The drawing *idioms* inside those files
(bevels, gradient fills, the sheet packer) are deliberately unexported: they
are how these eight packs happen to be drawn, not an API anyone should have
to live with. Copy the ones you want.

`skingen/panel` is the same idea one level up: the rects of Minim's two panel
faces — every well, key, slider and label — published as numbers, so a panel
skin for a player-shaped app starts from a measured layout instead of a ruler
(see [Fixed layouts](#fixed-layouts)).

### How this repository uses it

`go run ./cmd/uitk-skingen` rewrites `style/skins/` from `skingen.Plans()`,
and `-list` prints what it would write without writing it.
`TestSkinArtIsReproducible` regenerates every pack into a temporary directory
and compares byte for byte, so art that was changed and not committed — or a
PNG somebody edited by hand — fails the build. That test is what lets the
next section say "generated, not painted" as a fact.

## The demo skins

Both are **generated**, not painted, by the `skingen` package and written by
`go run ./cmd/uitk-skingen`. Three reasons, all about being able to say
something true about the art:

- It is demonstrably the toolkit's own. Every pixel comes from paths and
  gradients in that package. The licensing rule for engines — facts, never
  transcribed code or artwork — extends to skins, and this is how it is kept.
  No `.wsz`, `.wal`, `.vlt` or `.wmz` file, and no bitmap out of one, is in
  this repository, as a sample or as a test fixture.
- It is regenerable at any scale. A 3× set is one entry in `Scales`.
- The manifest cannot drift from the art: the cell layout is stated once, in
  Go, and both the PNGs and `skin.json` come out of it.

`TestSkinArtIsReproducible` regenerates into a temporary directory and
compares byte for byte, so art changed and not committed fails the build.

| | Nocturne | Cassette | Deck |
| --- | --- | --- | --- |
| look | amber on charcoal, lit from above | six colours, two-pixel bevels | graphite and one cold accent |
| drawn as | paths and gradients | whole pixels on a grid | paths and gradients |
| `pixelated` | no | yes | no |
| base pack | `breeze-night` | `win95` | `breeze-night` |
| sheet | 520×338 at 1×, 65 sprites | 270×182 at 1×, 55 sprites | 516×212 at 1×, 19 sprites |
| exercises | nine-slice, tint, scale sets, `middle: none` | nearest sampling at whole multiples, `middle: tile`, exact doubling | `window.shape`, a control cut from its own alpha, a deliberately partial skin |

And the ones the players wear. They are skins for a *particular app* rather
than worked examples of the format, which is a different job: each is as
complete as the app it dresses needs and no more, and each of these three
declares a silhouette the app never mentions.

| | Minim | Marquee | Lantern |
| --- | --- | --- | --- |
| look | a pixel front panel, phosphor green on graphite | brushed steel and glass over a deep blue display | matte charcoal and one indigo light, no gloss anywhere |
| drawn as | whole pixels on a grid | paths, gradients and a gloss | paths and flat gradients |
| `pixelated` | yes | no | no |
| base pack | `win95` | `breeze-night` | `breeze-night` |
| sheet | 222×190 at 1×, 57 sprites | 600×410 at 1×, 58 sprites | 516×286 at 1×, 48 sprites |
| `window.shape` | three stretching rects: the window steps in twice and stands on a chin | two: a shallow brow across the top, and a body on a 48-pixel dome | one: the window's own box, swept round by 52 at the bottom left and 10 at the bottom right |
| exercises | a pixel sheet in a window whose size *is* its design | an app's own shape taking precedence over the look's (the compact mode) | a single rect that is a silhouette because it is rounder than the frame, and a deliberately partial binding |

Minim switches between three skins, and the second two are *panels* rather
than dressings: pictures of the whole of each window with holes where the
keys go, which is how a player of that shape was always built. Each is after
a well-known look of its era, and each is drawn from scratch — the
proportions and positions are the published facts of the 275×116 format
and colours sampled as numbers; no bitmap, alphabet or mark of anybody's is
in them.

| | Minim Classic | Minim Silver |
| --- | --- | --- |
| look | the base-skin look of the era: slate-blue bevelled chrome, gold grooves either side of the title, a black display with green segments, grey keys, orange volume, green balance, yellow equaliser faders | the rounded look of the later era: silver chrome, a navy title band, a blue dot-matrix display, glossy round keys, capsule toggles with blue lamps, a tabbed equaliser |
| drawn as | whole pixels on a grid | paths, gradients and gloss |
| `pixelated` | yes — the 2× sheet is the 1× doubled | no |
| base pack | `win95` | `luna` |
| sheet | 340×879 at 1×, 161 sprites | 340×920 at 1×, 189 sprites |
| `window.shape` | none: a rectangle, as the original was | one rect the size of the window with a 7-pixel radius on every corner, rounder than the (square) frame, so the desktop shows beyond all four |
| exercises | `caption.title` (the gap in the groove), a 14-pixel caption, fixed layouts, a pixel panel on one device grid at fractional scales | fixed layouts with hover faces, a window variant (the equaliser's tab), round keys shaped by their art, a path-drawn panel at fractional scales |

Both bind only the frame — the window, the caption, its plate, its keys —
and the focus ring, and state the player's three layouts. Everything else a
player opens over them, a menu or a tooltip, is their base pack's, which is
what the desktop under a player of that era looked like anyway.

Minim, Marquee and Lantern keep out of both top corners, which is the
constraint a shaped skin has and it is worth stating: a framed caption centres its buttons about
a fifth of its height down, and which *side* they sit on is the desktop's
choice unless the skin says otherwise — `style.CaptionButtonsDesktop` is the
default, a skin's `layout` is consulted only when the user has asked for the
look's own, and a skin's `buttons` moves them to the side its art has room
on. None of the demo skins states a side, so each keeps out of both corners:
a silhouette that bit into a top corner would eat a close button on half the
desktops it ran on. Minim Silver does round its top corners, by
seven design pixels, which is less than the room its fifteen-pixel caption
leaves above and beside a key: the close button is whole on either side.

Deck is the shaped one. Its outline is a full-width shoulder with the title
plate inlaid in it over a body drawn in on both sides, so the desktop steps
in under both shoulders and every corner is round; its push button is a
stadium whose caps are half the control height, so a box square at that
height is a disc and the corners of the box are not the button. It binds
nine parts and leaves the rest to `breeze-night`, because a shaped skin is a
partial override like any other.

## Looking at one

```bash
# The controls sheet, every control in every state.
go run ./cmd/uitk-themesheet -theme nocturne -o /tmp/sheets
go run ./cmd/uitk-themesheet -theme nocturne -scale 1.75 -o /tmp/sheets
go run ./cmd/uitk-themesheet -frames -theme nocturne -o /tmp/sheets

# A whole app.
env -u WAYLAND_DISPLAY -u DISPLAY UITK_THEME=nocturne go run ./examples/gallery -headless

# What an author would want to know about one.
go run ./cmd/uitk-skin lint nocturne deck ~/.config/uitoolkit/skins/mine

# Settings, with the skin staged in the live preview — cut to the skin's
# silhouette, for a shaped one.
go run ./cmd/uitksettings -stage deck -screenshot /tmp/shots

# What the generator would write.
go run ./cmd/uitk-skingen -list

# A shaped skin over a test card: the desktop beside the silhouette, and a
# round button that takes the pointer only on its ink.
go run ./examples/skinshape -mode backdrop &
go run ./examples/skinshape -theme deck

```

A whole application wearing one — switching between `minim`,
`minim-classic` and `minim-silver` live, all its windows at once — is the
music player in
[media-player-music](https://github.com/codemodify/media-player-music); what
the toolkit does for it is in [players.md](players.md).

## What does not change, whatever the skin

This is the part to be loud about, because no skinned system of the era could
say it.

- **Keyboard.** Tab order is component order, not art order. Mnemonics,
  accelerators, Escape and Alt+F4 are unchanged. The focus ring cannot be
  removed.
- **The accessibility tree.** `win.AccessibleTree()` is the tree the themed
  app would produce — a window, a title bar, buttons with names, sliders with
  values, a list with rows. The art is invisible to all of it, which is the
  point. A skin is an engine, and engines paint; they do not build trees.
- **HiDPI.** Layout and text are exact at every scale because they are not
  pictures. The art chooses upward, or declares itself pixelated and stays
  crisp at whole multiples.
- **The other 121 packs.** Every addition is optional and nothing about the
  existing engines changed. Remove the skin, pick Breeze or Luna or System 7,
  and the app still works.

A uitoolkit skin is a costume, not a prosthesis.
