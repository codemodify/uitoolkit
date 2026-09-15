# Theme engines

A uitoolkit theme is two things:

- a **pack** — colours, metrics and a few engine parameters (`ThemePack`,
  built in Go or loaded from `themes/<name>/theme.json`), and
- an **engine** — the Go code that decides what a button, a scrollbar or a
  tab *is* for that family of looks (`style.Engine`).

One engine paints many packs: the `win95` engine paints Windows 95, 98, 2000,
Hot Dog Stand and High Contrast; a `luna` engine paints Luna Blue, Olive and
Silver. Packs change colours and metrics; engines change shapes.

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
  `DrawScrollBarParts`.
- **Frames** — `GroupBoxInsets/DrawGroupBox` (titled frames),
  `WindowFrameInsets/DrawWindowFrame` (in-app dialogs: caption, close),
  `DrawWindowBackground` (pinstripes, brushed metal) and `DrawTabPane`.
- **View frames** — `ViewFrameInsets` / `DrawViewFrame` frame lists,
  trees and tables (Qt's `PE_Frame`). By default the frame is the engine's
  own field face, as thick as the `viewFrame` metric (Win95 2px sunken,
  Luna and Aqua 1px, flat looks 0). Views work inside it; `Frameless`
  opts a view out when its pane is already framed.
- **Tabs** — `TabOutset` grows the selected tab, which the tab bar paints
  last, so it overlaps its neighbours (Win95 and XP: 2px each side).
- **Shadows** — `PopupShadow(kind)` is how far a floating layer's drop
  shadow reaches past its bounds, and `DrawPopupShadow` paints it before the
  layer (`PopupMenu` for menus and lists, `PopupTooltip`, `PopupDialog`).
  The window paints them for its popup and tooltip layers and repaints the
  reach when a layer goes away; overlays paint their dialog's. The base
  engine gives modern soft shadows (the pack param `shadow` scales them, `0`
  turns them off); eras without shadows return zero (Win95, Motif, NeXT).
  `DropShadow` is an exact nine-piece gradient shadow; `ShadowReach` is its
  extent.
- **Controls** — the 34 `Draw*` methods, same signatures as `LookAndFeel`
  plus the look. Override when the era's layout differs (Win95 combos
  highlight their text; Aqua centres tabs).

`Role` is the uxtheme "part": `RoleButton`, `RoleTool`, `RoleField`,
`RoleCheck`, `RoleRow`, `RoleTab`, `RoleThumb`, `RoleTrack`, `RoleMenu`,
`RoleCombo`, `RoleSplitter`, `RoleBar`, `RolePanel`. `ControlState` is the
state bitset (`Hovered`, `Pressed`, `Disabled`, `Focused`, `Checked`,
`Primary` = default button, `Toggle`).

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

4. Look at it: `go run ./cmd/uitk-themesheet -theme luna-blue -o /tmp/s`
   (and `-scale 2`). Read the PNG. Iterate until it matches the era. Then
   see real apps in it: `go run ./examples/gallery -screenshot /tmp/g
   -theme luna` takes every scripted gallery shot (menus, combo lists,
   tooltips, message boxes, tables) in your pack, and `UITK_THEME=luna`
   runs any uitoolkit app in it, like `GTK_THEME` or `QT_STYLE_OVERRIDE`.

5. Test it: at minimum a registration test; the contract test covers
   painting every control in every state.

## Rules

- **Paint inside the rect you are given.** Widgets paint clipped to their
  bounds; anything outside is lost (this is why focus rings used to vanish).
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

## Pack schema (`theme.json`)

```json
{
  "label": "Luna Blue", "family": "light", "engine": "luna",
  "year": 2001, "lineage": "Windows", "summary": "Windows XP default.",
  "metrics": { "controlH": 28, "scroll": 17, "radius": 3, "square": false, "viewFrame": 1 },
  "colors":  { "background": "#ece9d8", "accent": "#316ac5", "selection": "#316ac5" },
  "extra":   { "btnTop": "#ffffff", "btnBottom": "#d6d0c5", "hot": "#f8b330" },
  "params":  { "glossy": 1 }
}
```

`colors` are the shared palette (every engine understands them); `extra` and
`params` are engine-specific and documented at the top of each engine file.
