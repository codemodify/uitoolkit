# Settings

`cmd/uitksettings` is the toolkit appearance editor and theme browser.

## Run

```bash
go run ./cmd/uitksettings
go run ./cmd/uitksettings -stage win98              # open with Windows 98 staged in the preview
go run ./cmd/uitksettings -headless                 # settings.png in cwd
go run ./cmd/uitksettings -stage aqua -screenshot docs/screenshots
```

## Pages

- **Themes** — the theme browser. Every built-in pack listed by year
  (`1995 · Windows 95`), filterable by decade or *My themes*; the selected
  pack's name, year, lineage, engine and summary; a **live preview**: a small
  but fully interactive application window (menu bar, tool bar, tabs with
  every control, tree, table, dialogs, status bar) painted entirely in the
  staged theme — frame, caption and all — while Settings itself keeps the
  applied look (`widgets.ThemeScope`). Under the preview: **Corners**
  (Theme shape / Round / Square), **Icon size** and **Icons**, all shown
  live; **Animations** (hover fades, the default button's pulse, busy
  bars); and **Match the desktop's light or dark mode** (see below).
- **Packs & icons** — user theme packs (export the staged theme, delete
  user packs) and icon sets (built-in and user, delete user sets).
- **About** — versions, the engine list, and the files Settings reads and
  writes.

**Apply** writes `look.json` and switches Settings and every app that
watches the file; **Revert** drops the staged change. Both stay pinned under
the pages. Closing without Apply discards the staged change.

## Prefs file

```
$XDG_CONFIG_HOME/uitoolkit/look.json
# fallback: ~/.config/uitoolkit/look.json
```

```json
{
  "version": 2,
  "theme": "breeze",
  "corners": "theme",
  "icons": "lucide",
  "iconSize": "medium",
  "reduceMotion": false,
  "followDesktop": true,
  "nativeDialogs": false
}
```

`theme` is the theme pack. `corners` is `theme` (the pack's own shape,
the default), `round` or `square`. `icons` is the chrome set (`classic`,
`sharp`, or an installed directory such as `lucide`). `iconSize` is
`small` (16px), `medium` (24px, default), or `large` (32px). Numeric
aliases `16` / `24` / `32` are accepted on load. `reduceMotion` turns
animations off. `followDesktop` shows the pack's light or dark sibling to
match the desktop. `nativeDialogs` shows the desktop's own file dialogs
(KDE's, GNOME's, through the XDG portal) instead of the toolkit's themed
ones.

### Following the desktop's light or dark mode

With `followDesktop` on, apps ask the desktop for its light / dark
preference, the setting GTK 4, libadwaita, Qt 6 and the browsers follow
(on Linux the XDG desktop portal's `org.freedesktop.appearance
color-scheme`, which GNOME's *Style* and Plasma's colour schemes set), and
show the saved pack's sibling for it. They switch live when the desktop
does. `theme` stays the pack you chose, so a desktop that turns light
again gets it back.

Siblings pair by name: `breeze` and `breeze-night`, `win95` and
`win95-dark`; a variant takes its family's sibling (`luna-olive` shows as
`luna-night`, Royale Noir; `aqua-graphite` as `aqua-night`); every CDE
palette darkens to Charcoal and every Window Maker scheme to Night Sky.
Packs with no sibling (Amiga, OS/2 Warp, BeOS, the high-contrast scheme)
stay as they are. A user pack `mine` pairs with a user pack `mine-night`.

Following the desktop also takes its **accent colour** (Plasma's and
GNOME 47's accent, the portal's `accent-color`) in themes whose engine
recolours around one, as Windows 10 and 11, macOS, Plasma, GNOME and
Material You do: Breeze's selection, focus, hover and default button, and
everything it mixes from them. Historical looks keep their own colours.

`UITK_COLOR_SCHEME=dark` (or `light`) stands in for the desktop, to try a
theme's other side without switching the desktop; `UITK_ACCENT=#e95420`
stands in for its accent. `UITK_THEME=<pack>`
shows exactly that pack and does not follow. Headless and offscreen apps
(tests, screenshots) never ask the desktop.

The desktop's **reduced-motion** setting (GNOME's *Animations* switch,
Plasma's animation speed at *Instant*, published as the portal's
`reduced-motion`) turns every uitoolkit animation off in every app,
whatever `reduceMotion` says; Settings shows a note under its Animations
switch while it does. Apps read the portal once at start (in the
background, alongside the font index) and follow it live.

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

**Embedded era packs** (Classic 95 through FlatLaf, each with a light
and a night twin where that era had one) ship in the binary and are
**not** auto-written to disk. `dark` / `light` are the Classic 95 twins
so existing `look.json` files keep working.

Settings lists them in one Built-in list (each row is the pack display
name, e.g. Classic 95 Dark, Luna Night). User exports live under **User**.

```
$XDG_CONFIG_HOME/uitoolkit/themes/<name>/theme.json
# fallback: ~/.config/uitoolkit/themes/<name>/theme.json
```

```json
{
  "label": "ocean",
  "palette": "dark",
  "family": "dark",
  "bevel": "classic-3d",
  "colors": {
    "background": "#3c3c3c",
    "hotFill": "#000080",
    "hotBorder": "#000040"
  }
}
```

JSON only. Pack-level `corners` / `icons` fields are ignored.
**Export current theme…** asks for a name and writes the staged
**tokens**. Corners, icons, and icon size stay in `look.json`.
A legacy `{ "palette": "dark" }` file still loads.

When a **User** theme is selected, **Delete** under that list confirms
(Yes/No) then removes `themes/<name>/` from disk. Built-in `dark` /
`light` never show Delete. If the deleted pack was selected, Settings
falls back to the matching builtin palette (or the other one if that
name was a shadow). If `look.json` named the deleted pack, it is
rewritten to the fallback so the selection is not left dangling.
The same Delete control appears under **User** icon sets (not premiere
or drawn classic/sharp).

`ListThemes` lists Built-in era packs (not shadowed) then user packs
(sorted by name). `LoadTheme(name)` **prefers the user pack** when both
exist — a user `dark` overrides the embedded starter and is listed once
under User. Legacy compound ids map to `dark` / `light`. Era aliases
(`classic95`, `luna-dark`, …) resolve to the shipped id.

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
scale and density on the current look are kept. A desktop that turns light
or dark reloads the same way while the appearance follows it, and
`Application.OnLookChange` callbacks run after every change (Settings
redraws its preview there).
`Application.ApplyAppearance(ap)` applies an `Appearance` without saving
it: theme, corners, icons, motion and following the desktop.

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
| `Application.ApplyAppearance`, `Application.OnLookChange`, `Application.DesktopColorScheme`, `ColorSchemeEnv` | `app` |
| `ColorScheme`, `SchemeVariant`, `Appearance.Effective`, `SetDesktopColorScheme`, `DesktopReducesMotion` | `style` / `uitoolkit` |
| `AccentEngine`, `SetDesktopAccent`, `DesktopAccent`, `TakesAccent`, `CloneTokenMaps` | `style` |
| `DesktopPrefs`, `ReadDesktopPrefs`, `WatchDesktopPrefs` (the portal) | `platform` |
| `DrawToolIcon`, `DrawFileToolIcon` | `style` |
