# Theme engines

A uitoolkit theme is two things:

- a **pack** — colours, metrics and a few engine parameters (`ThemePack`,
  built in Go or loaded from `themes/<name>/theme.json`), and
- an **engine** — the Go code that decides what a button, a scrollbar or a
  tab *is* for that family of looks (`style.Engine`).

One engine paints many packs: the `win95` engine paints Windows 95, 98, 2000,
Hot Dog Stand and High Contrast; a `luna` engine paints Luna Blue, Olive and
Silver. Packs change colours and metrics; engines change shapes.

Engines so far, each a good model for its family:

| engine | looks | a good example of |
| --- | --- | --- |
| `win95` | Windows 95 / 98 / 2000 and schemes | the reference: bevels, dotted focus, overlapping tabs |
| `luna` | Windows XP Luna, Royale | gradients, memoised colours, rebar grips |
| `aqua`, `platinum` | Mac OS X, Mac OS 8 | gel materials, textures, Mac rules (inactive grey, hollow selections) |
| `motif` | Motif, CDE, IRIX, HP VUE | colour derivation, shadow thickness per pack |
| `next` | NeXTSTEP, OPENSTEP, Window Maker | textures as pack data, dithers |
| `fusion`, `oxygen`, `breeze` | Qt Fusion, KDE 4, Plasma | modern flat and glow looks, tone models |
| `base` | the stock looks | defaults every engine inherits |

## Where this comes from

Every serious toolkit separates *what a control does* from *how it looks*,
and the good ones let the look replace shapes, not just colours:

| Toolkit | Look is… | Takeaway for uitoolkit |
| --- | --- | --- |
| Qt | `QStyle` subclasses (Windows, Fusion, Breeze, Oxygen, Plastique…) that draw *primitives*, *controls* and *complex controls*, answer *pixel metrics*, *sub-control rects* and *style hints*, over a `QPalette` | Our `Engine`: parts + controls + metrics + scrollbar geometry; `QProxyStyle` ≈ embedding `BaseEngine` |
| WPF | Control templates in theme dictionaries — Microsoft shipped `PresentationFramework.Classic/Luna/Royale/Aero` as separate theme assemblies | One engine per era, packs per colour scheme |
| WinForms / uxtheme | `VisualStyleRenderer(part, state)` — `BP_PUSHBUTTON` × `PBS_HOT`… | `Engine.Face(role, ControlState)` |
| GTK 3/4 | CSS over a node tree (`button:hover`), rendered by GSK | Data belongs in packs (`theme.json` colours, `extra`, `params`); shapes in engines |
| Avalonia | `ControlTheme`s (Fluent, Simple) with selectors | Same split; Fluent is one engine among many |
| Swing | `LookAndFeel` + `UIDefaults`; Metal themes are palettes, Synth is XML skins | Metal/Ocean = one engine, several packs |

## Anatomy

```
style/engine.go        Engine interface, registry, scrollbar geometry, Memo
style/engine_base.go   BaseEngine: the stock painter every engine embeds
style/shapes.go        shared era idioms (Bevel4, Etched, Gloss, Bumps, …)
style/engine_<id>.go   one file per engine: engine + its packs + init()
internal/themesheet    renders every control × state for a look
cmd/uitk-themesheet    writes those sheets as PNG
```

A `Classic` look owns palette, metrics, fonts and tokens; every `Draw*` call
on it goes to its engine: `l.eng().DrawButton(l, ctx, b, st, label)`.

### Dispatch rule

`BaseEngine` never calls its own methods: every part and nested control is
re-dispatched through `l.Engine()`. So an engine that overrides only
`CheckIndicator` changes the check box *and* anything else that paints one,
while inheriting all layout and text code. Override the smallest thing that
gets the era right:

- **Parts** — `Face(role, state)` (fill + border + bevel, returns the label
  colour), `CheckIndicator`, `RadioIndicator`, `Arrow`, `Expander`,
  `MenuHighlight`, `MenuTextColor`, `FieldFocusRing`.
- **Scrollbars** — `ScrollBarStyle` (thickness, overlay vs gutter, arrow
  placement: none / both ends / grouped end / grouped start) and
  `DrawScrollBarParts`. `Transient` bars come and go over the content,
  which keeps its full width (libadwaita, WinUI, Mac OS X since Lion): the
  view shows them while it scrolls or the pointer moves over it and fades
  them out after a second of rest; the engine draws the thin idle form unless
  `ScrollState.Hovered` or a part is pressed.
- **Frames** — `GroupBoxInsets/DrawGroupBox` (titled frames),
  `WindowFrameInsets/DrawWindowFrame` (in-app dialogs: caption, close),
  `DrawWindowBackground` (pinstripes, brushed metal) and `DrawTabPane`.
- **View frames** — `ViewFrameInsets` / `DrawViewFrame` frame lists,
  trees and tables (Qt's `PE_Frame`). By default the frame is the engine's
  own field face, as thick as the `viewFrame` metric (Win95 2px sunken,
  Luna and Aqua 1px, flat looks 0). Views work inside it; `Frameless`
  opts a view out when its pane is already framed.
- **Text and bars** — `ControlFont(role)` is the face a control is labelled
  in (Metal's bold buttons); widgets measure labels with it, so a bold look's
  text fits. `ToolBarInsets` keeps room for a tool bar grip (XP's rebar
  handle). Odd rows carry `StateAlternate` for striped lists (Aqua).
- **Spin boxes** — `SpinBoxStyle` (Qt's `CC_SpinBox`): `Inside` puts the
  step buttons inside the field's frame, which the spin box draws across the
  whole box, then paints the editor and the stepper `StateFrameless` inside
  it (Windows, KDE, GNOME and the base look); `Across` lays them side by
  side, "− +" (GTK 3 and later); neither stands a stepper beside the field
  (Mac OS, Motif, NeXT).
- **Tabs** — `TabOutset` grows the selected tab, which the tab bar paints
  last, so it overlaps its neighbours (Win95 and XP: 2px each side).
  `TabOverlap` lays every tab over its neighbour's border, so two tabs share
  one line (Aero and Metro: 1px; Qt's `PM_TabBarTabOverlap`).
- **Shadows** — `PopupShadow(kind)` is how far a floating layer's drop
  shadow reaches past its bounds, and `DrawPopupShadow` paints it before the
  layer (`PopupMenu` for menus and lists, `PopupTooltip`, `PopupDialog`).
  The window paints them for its popup and tooltip layers and repaints the
  reach when a layer goes away; overlays paint their dialog's. The base
  engine gives modern soft shadows (the pack param `shadow` scales them, `0`
  turns them off); eras without shadows return zero (Win95, Motif, NeXT).
  `DropShadow` is an exact nine-piece gradient shadow; `ShadowReach` is its
  extent.
- **Materials** — `ctx.BackdropBlur(r, sigma)` blurs what lies under r
  (inside the clip) on every device: paint a translucent tint over it next
  for acrylic, glass or vibrancy (Fluent's flyouts, Aero's glass frames).
  `ctx.DrawLayer` fades a group as one, `ctx.DrawCrossFade` mixes two
  drawings. `DitherTile` and `DevicePattern` tile a 50% dither (or any
  image, through `paintengine2d.ImagePattern`) anchored to the device grid,
  for the eras before true colour.
- **Controls** — the 34 `Draw*` methods, same signatures as `LookAndFeel`
  plus the look. Override when the era's layout differs (Win95 combos
  highlight their text; Aqua centres tabs).

`Role` is the uxtheme "part": `RoleButton`, `RoleTool`, `RoleField`,
`RoleCheck`, `RoleRow`, `RoleTab`, `RoleThumb`, `RoleTrack`, `RoleMenu`,
`RoleCombo`, `RoleSplitter`, `RoleBar`, `RolePanel`. `ControlState` is the
state bitset (`Hovered`, `Pressed`, `Disabled`, `Focused`, `Checked`,
`Primary` = default button, `Toggle`, `First` / `Last` in a strip or a
table row, `Inactive`, `Backdrop`, `Alternate` = an odd row, `ExpanderHot`
= the pointer is on a tree row's expander, `Frameless` = a part inside a
frame its parent drew).

`StyleHint` answers behaviour questions like Qt's `styleHint`:
`HintDialogPrimaryFirst` (1: "OK Cancel", Windows and KDE; 0: "Cancel OK",
Mac and GNOME), `HintTabsCentered` (Aqua's segmented tabs) and
`HintFormLabelsRight` (Mac, NeXT, Oxygen and Breeze right-align form
labels) and `HintHoverFadeMs`: buttons, check boxes, radios, switches and
combo boxes cross-fade that long when only their hover or focus changes
(Aero 200ms, Adwaita 200ms, Windows 10, Breeze and Oxygen 150ms, Fluent
83ms; older eras switch at once, and presses never fade). The new state is
composited as one layer, so engines draw each state as usual.
`UITK_ANIMATIONS=0` turns fades off.

### Item views

`DrawListRow`, `DrawTreeRow` and `DrawTableCell` get the item state:
`Checked` = selected, `Hovered`, `Focused` = the current row of a focused
view, `Inactive` = the view does not have keyboard focus, `Backdrop` = the
window is inactive (GTK's `:backdrop`). Follow the platform you imitate:
Windows and Mac looks grey a selection as soon as the view is `Inactive`
(Win95 to the button face, Aqua pale grey, Platinum a hollow outline), GTK
and Qt-style looks keep it until `Backdrop`, Motif and NeXT never change
it. `RoleRow` faces see the same bits, and every widget's state carries
`Backdrop` while its window is inactive (Aqua's default button turns clear).

Table cells carry `First` / `Last` (a cell with neither is a middle cell).
To draw one selection box across a row, paint the row box over
`CellSpan(b, st, reach)` clipped to the cell: the span runs past the sides
where the row goes on, so the cells join into one box (Aero's Explorer
selection, Fluent's list item and pill).

Tree rows carry their branch chain (Qt's `State_Sibling`): `st.HasNextSibling(d)`
says whether the node at depth `d` on the row's chain (its ancestors, then
the row itself) has a sibling after it. Draw a branch line through the row
only where that branch continues, and end a last child's line at its elbow
("└" rather than "├"). Rows without the chain (`StateTreeChain` unset)
answer true, so lines keep running.

List and tree painters mark a `Focused` row themselves — default
`ItemFocus` over the row, Win95 and XP a dotted rectangle around the label,
Motif the solid location cursor; tables call `DrawItemFocus` over the whole
row after its cells. Looks whose fields have a focus ring (`FieldFocusRing`)
may leave `ItemFocus` empty and let the view frame ring the focused view.

## Writing an engine

1. Copy the shape of `style/engine_win95.go`:

   ```go
   type lunaEngine struct{ BaseEngine }

   func init() {
       RegisterEngine(lunaEngine{})
       for _, p := range lunaPacks() { RegisterPack(p) }
   }

   func (lunaEngine) ID() string { return "luna" }
   func (lunaEngine) DefaultMetrics() ChromeMetrics { return ChromeMetrics{…} }
   ```

2. Resolve your colours once per look with `Memo` and read pack data with
   `l.X("key", fallback)` (colours from `extra`) and `l.P("key", def)`
   (numbers from `params`):

   ```go
   type lunaKey struct{}
   func lunaColors(l *Classic) luna {
       return l.Memo(lunaKey{}, func() any { return lunaBuild(l) }).(luna)
   }
   ```

3. Register packs next to the engine — `ThemePack{Name, Label, Year, Lineage,
   Summary, Tokens: ThemeTokens{Engine: "luna", Family, Palette, Extra,
   Params}}`. Registered packs sort by `Year`; a pack with the name of a
   legacy era pack (`luna`, `aqua`, `motif`…) replaces it.

4. Look at it: `go run ./cmd/uitk-themesheet -theme luna -o /tmp/s`
   (and `-scale 2`). Read the PNG. Iterate until it matches the era. Then
   see real apps in it: `go run ./examples/gallery -screenshot /tmp/g
   -theme luna` takes every scripted gallery shot (menus, combo lists,
   tooltips, message boxes, tables) in your pack, and `UITK_THEME=luna`
   runs any uitoolkit app in it, like `GTK_THEME` or `QT_STYLE_OVERRIDE`.

5. Test it. The shared tests run over every registered pack: every control
   paints inside its bounds in every state, shadows stay within their
   reach, keyboard focus is visible, text meets the contrast checks. Add a
   test file for the engine: registration and year order, bounds at scale 1
   and 2, `WindowCloseRect` against the painting, your style hints.

## Rules

- **Paint inside the rect you are given.** Widgets paint clipped to their
  bounds; anything outside is lost (this is why focus rings used to vanish).
- **Rects arrive on whole pixels.** Layout rounds every component's bounds
  to device pixels (as WPF and Avalonia do), and tables round their column
  edges, so `snap`ped lines in your painting land on real pixels. Cells that
  meet share an edge exactly; never grow a cell to cover a seam.
- **Focus is painted inside** the control (`DrawFocusRing(ctx, b)` rings
  the inside of `b`). Check boxes and radios ring their label.
- **Scale everything** that is a design length with `l.S(v)` (display scale)
  or take it from `l.metrics` (already scaled). Snap 1px lines with `snap`.
- **Label colour comes from `Face`** — return a readable foreground for the
  fill you painted (selected rows, hot menus, default buttons).
- **Deterministic and cheap**: no randomness, no allocation-heavy work per
  paint — derive once with `Memo`; prefer rects and simple paths; gradients
  are fine (the GPU path caches their ramps).
- **Metrics**: `DefaultMetrics` is the era's geometry at the toolkit's 16px
  UI font (scale the original proportions; Windows used 11px). Set
  `Square: true` for natively square looks so the "theme" Corners pref keeps
  them square. Density shifts these values; do not fight it.
- **No hover where the era had none** only when authentic (Win95 buttons
  had no hot state) — but never lose feedback users rely on: hover on tool
  buttons and menus, pressed on everything, visible focus.
- **Facts, not copies.** uitoolkit is MIT. Take colours, sizes, radii,
  gradient stops and behaviour from other toolkits as facts, and write the
  drawing yourself; never transcribe or port code, path tables, pixmaps or
  data from GPL / LGPL toolkits (Qt and KDE styles, GTK engines, OpenJDK,
  Window Maker) or proprietary artwork.

## For widget authors

Widgets reach the look through `LookAndFeel`, plus optional interfaces a
look may implement; use the `…Of` helpers, which fall back when a look
does not: `ControlFontOf` (measure labels), `DrawArrowOf` (era arrows),
`DrawItemFocusOf` (current-row mark), `ViewFrameInsetsOf` /
`DrawViewFrameOf`, `PopupShadowOf` / `DrawPopupShadowOf`, `TabOutsetOf`,
`TabOverlapOf`, `ToolBarInsetsOf` and `LookHint`. A scrolling view gives up
`ScrollGutter` to its bar (nothing for transient bars) and lets
`scrollDrag` show, fade and hit-test it. Build row states with `widget.ItemState`
or `widget.RowItemState`, which add `Focused`, `Inactive`, `Backdrop` and
`Alternate` for you.

## Pack schema (`theme.json`)

```json
{
  "label": "Luna Blue", "family": "light", "engine": "luna",
  "year": 2001, "lineage": "Windows", "summary": "Windows XP default.",
  "metrics": { "controlH": 28, "scroll": 17, "radius": 3, "square": false, "viewFrame": 1 },
  "colors":  { "background": "#ece9d8", "accent": "#316ac5", "selection": "#316ac5" },
  "extra":   { "btnTop": "#ffffff", "btnBottom": "#d6d0c5", "hot": "#f8b330" },
  "params":  { "glossy": 1 },
  "fonts":   { "ui": ["Tahoma", "Verdana", "DejaVu Sans"], "mono": ["Courier New"] }
}
```

`colors` are the shared palette (every engine understands them); `extra` and
`params` are engine-specific and documented at the top of each engine file.

`fonts` lists typefaces, most wanted first; the first one installed wins
(fontconfig), and the bundled Titillium Web / JetBrains Mono stand in when
none is. A pack that names none reads in its era's typefaces
(`style/fonts_era.go`): XP's Tahoma, Vista's Segoe UI, Aqua's Lucida Grande,
NeXT and Motif's Helvetica, GNOME's Cantarell, Plasma's Noto Sans, each
followed by open look-alikes (Nimbus Sans for Helvetica, Liberation for Arial
and Courier New, Selawik and Noto Sans for Segoe UI). A variable font's bold
is drawn by emboldening its regular outline. `UITK_SYSTEM_FONTS=0` keeps
every look on the bundled faces (reproducible screenshots).
