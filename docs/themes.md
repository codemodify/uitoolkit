# Theme packs

`uitoolkit` skins are **token packs**, not palette-only recolors. A pack
names a bevel language, metrics, elevation, and per-state chrome colors.
`Classic` paints every control from those tokens. Corners, icon set, and
icon size stay independent `look.json` prefs.

## Built-in era packs

| Id | Label | Era | Family | Bevel |
| --- | --- | --- | --- | --- |
| `dark` | Classic 95 Dark | Classic 95 | dark | `classic-3d` (1px) |
| `light` | Classic 95 Light | Classic 95 | light | `classic-3d` (1px) |
| `motif` | Motif | Motif / CDE | light | `classic-3d` (2px) |
| `cde` | CDE Charcoal | Motif / CDE | light | `classic-3d` (2px) |
| `cde-crimson` | CDE Crimson | Motif / CDE | light | `classic-3d` (2px) |
| `next` | NeXT | NeXT | dark chrome / light fields | `classic-3d` |
| `next-night` | NeXT Night | NeXT | dark | `classic-3d` |
| `luna` | Luna | Luna | light | `luna-hottrack` |
| `luna-night` | Luna Night | Luna | dark | `luna-hottrack` |
| `aqua` | Aqua | Aqua | light | `soft-shadow` |
| `aqua-night` | Aqua Night | Aqua | dark | `soft-shadow` |
| `fusion` | Fusion | Fusion | light | `classic-3d` (dense) |
| `fusion-night` | Fusion Night | Fusion | dark | `classic-3d` |
| `breeze` | Breeze | Breeze | light | `none` + focus ring |
| `breeze-night` | Breeze Night | Breeze | dark | `none` + focus ring |
| `fluent` | Fluent | Fluent | light | `fluent-accent` |
| `fluent-night` | Fluent Night | Fluent | dark | `fluent-accent` |
| `material` | Material | Material | light | `soft-shadow` + elevation |
| `material-night` | Material Night | Material | dark | `soft-shadow` + elevation |
| `flatlaf` | FlatLaf | FlatLaf | light | `none` (dense IDE) |
| `flatlaf-night` | FlatLaf Night | FlatLaf | dark | `none` (dense IDE) |

Aliases (load-only, not listed twice): `classic95` → `light`,
`classic95-dark` → `dark`, `luna-dark` → `luna-night`, and the other
`*-dark` ids → the matching `*-night` pack.

`dark` / `light` remain the `look.json` defaults so existing files keep
working. They are the Classic 95 twins.

## Token schema

`~/.config/uitoolkit/themes/<name>/theme.json`:

```json
{
  "label": "Luna",
  "era": "Luna",
  "family": "light",
  "palette": "light",
  "bevel": "luna-hottrack",
  "elevation": 0,
  "metrics": {
    "radius": 2,
    "radiusSmall": 1,
    "bevelDepth": 1,
    "gutterWidth": 10,
    "scroll": 16,
    "controlH": 32,
    "comboH": 28
  },
  "colors": {
    "background": "#ece9d8",
    "surface": "#ece9d8",
    "hotFill": "#c1d2ee",
    "hotBorder": "#316ac5"
  }
}
```

Legacy `{ "label": "ocean", "palette": "dark" }` still loads: family is
taken from `palette` / `theme`, tokens fall back to the Classic 95 pack
of that family.

`corners` and `icons` in a pack file are ignored.

### Bevel

| Value | Language |
| --- | --- |
| `none` | Flat fill + 1px border; Breeze / FlatLaf focus ring |
| `classic-3d` | Motif / Win95 highlight + shadow edges (depth 1–2) |
| `luna-hottrack` | Pale fill + thin border on hot / press / focus |
| `soft-shadow` | Raised face + drop shadow (Aqua / Material elevation) |
| `fluent-accent` | Wash + accent outline / bottom bar on focus |

### Controls painted from tokens

Button, ToolButton / ToolToggle, MenuBar titles, Menu / Popup / Context
rows, ListView / TreeView / TableView rows, Tab / TabBar, scrollbar
track / thumb / arrow wells, ComboBox field + dropdown, CheckBox /
Radio, TextField / NumberField / TextArea focus, Splitter grip, ToolBar
strip.

## API

```go
pack, _ := uitoolkit.LoadTheme("luna")
look := pack.Look()
tok := style.LookTokens(look)
_ = tok.Bevel
```
