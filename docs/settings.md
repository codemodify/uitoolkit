# Settings

`cmd/uitksettings` is the toolkit appearance editor. Theme (palette),
**corners**, **icons**, and **icon size** are independent prefs. Other
apps apply them with `Application.SetLook` / `PreferredLook`.

A theme is a color scheme only (`dark`, `light`, or a user-exported
palette). Control shape is the Corners radio (Round / Square →
`Metrics.Radius`). Chrome `ToolIcon`s come from `look.json` `"icons"`
(a PNG set under `~/.config/uitoolkit/icons/`, or drawn classic/sharp).

## Run

```bash
go run ./cmd/uitksettings
go run ./cmd/uitksettings -headless                 # settings.png in cwd
go run ./cmd/uitksettings -screenshot docs/screenshots
```

Linux X11 or Wayland (same backends as gallery / Mail). Headless uses the
offscreen surface.

There is no menu bar. **Apply** is the only persist action for
`look.json`. **Export current theme…** writes a user color theme
(palette only).

## What it sets

| Control | LookAndFeel | Values |
| --- | --- | --- |
| Theme | palette pack → `Classic` | Built-in `dark` / `light` + exported user palettes |
| Corners | `Metrics.Radius` | Round / Square |
| Icons | PNG set or drawn fallback | Built-in (classic/sharp + premiere names when copied) + User folders |
| Icon size | ToolIcon destination side | Small (16) / Medium (24) / Large (32); HiDPI still uses `@2x` |
| Export | `themes/<name>/theme.json` | current palette only; corners/icons/size stay prefs |

Theme and Icons lists use the same **Built-in** / **User** grouping.

The pickers update a **staged** appearance and preview it in the
Settings window (`Application.SetLook`) without touching disk. **Apply**
writes `$XDG_CONFIG_HOME/uitoolkit/look.json` (atomic rename) so other
running apps can reload. Closing the window without Apply **discards**
staged changes; the last applied file stays as-is.

## Prefs file

```
$XDG_CONFIG_HOME/uitoolkit/look.json
# fallback: ~/.config/uitoolkit/look.json
```

```json
{
  "theme": "dark",
  "corners": "square",
  "icons": "lucide",
  "iconSize": "medium"
}
```

`theme` is the color theme name. `corners` is `round` or `square`.
`icons` is the chrome set (`classic`, `sharp`, or an installed
directory such as `lucide`). `iconSize` is `small` (16px), `medium`
(24px, default), or `large` (32px). Numeric aliases `16` / `24` / `32`
are accepted on load.

Mode `0600`. Missing or invalid files yield `dark` + `round` +
`classic` + `medium`. A file without `iconSize` migrates to medium.

Compound v0.11–v0.12.1 theme ids migrate on load:

| Old `theme` | Becomes |
| --- | --- |
| `dark-round-classic` / `dark-round-sharp` / `dark-round` | `theme: dark`, `corners: round` |
| `light-square-sharp` / `light-square-classic` / `light-square` | `theme: light`, `corners: square` |
| `dark` / `light` (already palette-only) | same name; corners from the `corners` field or `round` |

An explicit `"corners"` field wins over corners parsed from a compound
name. `"icons"` is always the chrome set (never inferred from
`-classic` / `-sharp` suffixes).

## Theme packs

Two **embedded starters** (`dark`, `light`) ship in the binary
(`style/themes/*/theme.json`) and are **not** auto-written to disk.

Settings lists them under **Built-in**. User exports live under
**User**.

```
$XDG_CONFIG_HOME/uitoolkit/themes/<name>/theme.json
# fallback: ~/.config/uitoolkit/themes/<name>/theme.json
```

```json
{
  "label": "ocean",
  "palette": "dark"
}
```

JSON only. Pack-level `corners` / `icons` fields are ignored.
**Export current theme…** asks for a name and writes the staged
**palette**. Corners, icons, and icon size stay in `look.json`.

When a **User** theme is selected, **Delete** under that list confirms
(Yes/No) then removes `themes/<name>/` from disk. Built-in `dark` /
`light` never show Delete. If the deleted pack was selected, Settings
falls back to the matching builtin palette (or the other one if that
name was a shadow). If `look.json` named the deleted pack, it is
rewritten to the fallback so the selection is not left dangling.
The same Delete control appears under **User** icon sets (not premiere
or drawn classic/sharp).

`ListThemes` lists Built-in starters (not shadowed) then user packs
(sorted by name). `LoadTheme(name)` **prefers the user pack** when both
exist — a user `dark` overrides the embedded starter and is listed once
under User. Legacy compound ids map to `dark` / `light`.

## Icon sets (PNG files)

Chrome icons are **not** embedded. Ship-in-repo sets live at
`icons/lucide`, `icons/phosphor`, `icons/tabler`, `icons/heroicons`,
and `icons/material-symbols` (24×24 + `name@2x.png` 48×48). Each pack
covers the typed `ToolIcon` stems plus a wide chrome / Mail / UI
vocabulary (`pen` and `download` are always present). Copy them
yourself after every pull that refreshes `icons/`:

```bash
mkdir -p ~/.config/uitoolkit/icons
cp -R icons/lucide icons/phosphor icons/tabler icons/heroicons icons/material-symbols ~/.config/uitoolkit/icons/
```

See [icons/README.md](../icons/README.md) for licenses, the full stem
list, attribution, and the `@2x` convention.

Settings groups icon sets the same way as themes:

- **Built-in** — drawn `classic` / `sharp`, plus the five premiere
  names when those folders are present under `icons/`
- **User** — any other `icons/<name>/` folder that contains at least
  one ToolIcon PNG

The Appearance live preview includes a toolbar strip of chrome + Mail
Fetch / Write glyphs for the selected set.

When a premiere or user set is selected, a missing stem logs once and
paints **`no-icon`** (pack file, or the embedded placeholder if the
folder is not copied yet). Drawn classic is used only when the Icons
pref is explicitly `classic` or `sharp`. The toolkit tints
monochrome/alpha PNGs with the Look foreground / icon color.

The v0.12.0 `filled` / `outline` / `duotone` SVG folders are removed.

`ListIconSets` / `IconsDir` / `WithIcons` are the API. Changing icons
and Apply reloads other apps the same way as a theme change.

## Live reload in other apps

Source of truth is `look.json`. After Apply, every `Application` that
watches the file stats it (poll, ~300 ms while idle; every `PumpOnce`)
and, on change, reloads prefs and calls

```go
app.SetLook(style.WithAppearance(app.Look(), style.LoadAppearance()))
```

so theme, corners, icons, and icon size update without a restart. Display
scale and density on the current look are kept.

`Application.New` enables the watcher **by default when `Options.Look` is
nil** (that path already uses `PreferredLook()`). This is the least
friction for new apps:

```go
app := uitoolkit.New(uitoolkit.Options{}) // PreferredLook + look.json watch
```

Apps that pass an explicit look must opt in:

```go
look := uitoolkit.PreferredLook()
app := uitoolkit.New(uitoolkit.Options{Look: look, WatchLook: true})
```

Opt out (Settings does this so the picker stays staged until Apply):

```go
app := uitoolkit.New(uitoolkit.Options{DisableLookWatch: true})
```

`DarkLook` / `LightLook` fixtures stay static unless `WatchLook` is set,
so gallery screenshot pixels stay deterministic.

Mail, gallery, Files, Notes, and Inspector start from `PreferredLook`
and watch the file. Mail chrome persist (`mailui.json` density / layout)
does **not** rewrite `look.json`. `applyLook` applies `PreferredLook`
plus Mail density and syncs the View menu checkmark from the loaded
palette. View → Dark / Light is the only Mail write: `SaveAppearance` of
`WithPalette` (same corners/icons, opposite palette starter). Gallery /
screenshot fixtures keep explicit `DarkLook` / `LightLook`.

Helpers for a live look:

```go
uitoolkit.LoadTheme("light")
uitoolkit.ListThemes()
uitoolkit.WithTheme(look, uitoolkit.ThemeLight)
uitoolkit.WithCorners(look, uitoolkit.CornersSquare)
uitoolkit.WithIcons(look, uitoolkit.IconSetLucide)
uitoolkit.WithIconSize(look, uitoolkit.IconSizeLarge)
uitoolkit.WithAppearance(look, uitoolkit.Appearance{Name: "ocean", Corners: uitoolkit.CornersSquare, IconSize: uitoolkit.IconSizeLarge})
```

`PreferredLook` is `LoadAppearance().Look()`. `LoadAppearance` reads
`theme`, `corners`, `icons`, and `iconSize` from `look.json`, resolves
the color theme with `LoadTheme` (user pack, then builtin), and applies
corners, the icon set, and icon size on top.

## API

| Symbol | Package |
| --- | --- |
| `ThemePack`, `ListThemes`, `ListBuiltinThemes`, `ListUserThemes` | `style` / `uitoolkit` |
| `LoadTheme`, `ExportTheme`, `SplitLookThemeName` | `style` / `uitoolkit` |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName`, `IconSize` | `style` / `uitoolkit` |
| `IconSetInfo`, `ListIconSets`, `ListBuiltinIconSets`, `ListUserIconSets` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `ThemesDir`, `StarterName`, `DefaultThemeName` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance`, `WithIconSize`, `IconSizePixels` | `style` / `uitoolkit` |
| `Options.WatchLook`, `Options.DisableLookWatch` | `app` / `uitoolkit` |
| `Application.WatchingLook`, `Application.ReloadPreferredLook` | `app` |
| `DrawToolIcon`, `DrawFileToolIcon` | `style` |
