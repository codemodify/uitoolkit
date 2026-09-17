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

Two skins ship with the toolkit: **Nocturne**, amber on charcoal, drawn from
paths so it is exact at every scale; and **Cassette**, a six-colour pixel
skin with two-pixel bevels that exercises the `pixelated` path. Both are
generated — see [The demo skins](#the-demo-skins).

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
it, Orca reads it, a magnifier follows it.

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
      "size": 14, "bold": false
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
    "border": [1, 1, 1, 1],     // top, right, bottom, left
    "caption": 34,
    "radius": [7, 7, 0, 0],     // top-left clockwise
    "layout": ":minimize,maximize,close",
    // The silhouette, for when the toolkit can cut a window to one.
    "shape": [
      { "at": [0, 0, 0, 0], "radius": [7, 7, 0, 0],
        "stretchX": true, "stretchY": true }
    ]
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
from `internal/skinart/nocturne.go`.

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
| `menu.frame`, `tooltip` | the frames those float on |

`style.SkinPartNames()` is the same list at run time; a skin that names
anything else fails to load. New controls come from the toolkit, not from a
skin — otherwise a skin is a program, and running a stranger's program from a
zip file is a different project with a different threat model.

## States, and the fallback that makes a terse skin work

A part carries art per state. The state names are the toolkit's own
`ControlState` bits:

```
normal  hover  pressed  disabled  focus  checked  checkedHover
checkedPressed  default  inactive
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
| `inactive` | `disabled`, `normal` |

The state a control is in is read in the order a person would read it:
disabled first (it outranks everything), then the on/off axis, then the
pointer, then focus, then the default button, then a backdrop window.

`normal` is required as soon as a part names any art at all.

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

### One implementation note that matters

paintengine2d's bilinear sampler reads the two texels around each sample and
returns *transparent* outside the image; it does not clamp. A nine-slice's
middle is almost always stretched wider than its source, so its outermost
samples fall half a texel outside it and fade toward nothing — a pale vertical
line down every button, exactly where the middle meets the fixed edges.

So every sprite, and every one of a nine-slice's nine pieces, is cut into an
image of its own with a one-texel border replicated from its edge, and blitted
from its inner rect. That is clamp-to-edge, done by hand. It also means no
blit can ever reach a neighbouring sprite on the sheet, which is the other
artefact an atlas would otherwise have.

## What a skin cannot do yet

**Shape its window.** A skin paints inside an ordinary rectangular window.
The manifest already carries `window.shape` — a union of rounded rects in
design pixels, with a per-rect flag saying whether it stretches with the
window — and it is validated, resolved (`style.SkinWindowRects`) and tested,
so a skin written today is a complete document. Nothing consumes it until the
shaped-window work lands.

The form was chosen over an SVG path string deliberately: nothing in the
toolkit parses path strings, and a rect union is already what both consumers
want — a compositor input or opaque region *is* a rect list, and hit-testing
one is a loop rather than a rasterisation.

**Shape its controls.** Same reason. A skin's round button takes the pointer
in its bounding box. The art's own alpha is the obvious source for a per-
control hit shape, and every sprite is already cut into its own image, so a
coverage threshold is a read of its pixels.

**Lay out a fixed panel.** There is no absolute-layout half of this format
(WinAmp's 275×116 window, VLC's `<Layout>`). A skin re-skins ordinary widgets
at ordinary layout. That is the half where every invariant — accessibility,
keyboard, HiDPI, translation — holds for free rather than having to be
enforced.

**Carry behaviour.** Not a gap: a decision. No scripting, no bytecode, no
action vocabulary. Anything an app needs beyond painting it writes in Go.

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

Edit the manifest while an app is running and it re-applies: the look watcher
already stamps the pack's file, and a skin's manifest is that file. A re-saved
PNG shows up about a second later, on the same cache TTL as icon files.

To ship it as one file, zip the directory and name it `<name>.uskin`. An
archive whose manifest sits in a single top-level folder loads too, since that
is what most archivers produce.

## The demo skins

Both are **generated**, not painted, by `internal/skinart` and written by
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

| | Nocturne | Cassette |
| --- | --- | --- |
| look | amber on charcoal, lit from above | six colours, two-pixel bevels |
| drawn as | paths and gradients | whole pixels on a grid |
| `pixelated` | no | yes |
| base pack | `breeze-night` | `win95` |
| sheet | 404×338 at 1×, 60 sprites | 236×182 at 1×, 52 sprites |
| exercises | nine-slice, tint, scale sets, `middle: none` | nearest sampling at whole multiples, `middle: tile`, exact doubling |

## Looking at one

```bash
# The controls sheet, every control in every state.
go run ./cmd/uitk-themesheet -theme nocturne -o /tmp/sheets
go run ./cmd/uitk-themesheet -theme nocturne -scale 1.75 -o /tmp/sheets
go run ./cmd/uitk-themesheet -frames -theme nocturne -o /tmp/sheets

# A whole app.
env -u WAYLAND_DISPLAY -u DISPLAY UITK_THEME=nocturne go run ./examples/gallery -headless

# Settings, with the skin staged in the live preview.
go run ./cmd/uitksettings -stage nocturne -screenshot /tmp/shots

# What the generator would write.
go run ./cmd/uitk-skingen -list
```

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
