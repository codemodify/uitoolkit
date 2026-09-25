# Settings

`cmd/uitksettings` is the toolkit appearance editor and theme browser. The
command opens the window; the application is `cmd/uitksettings/settingsapp`
beside it, written against the published API like every other sample. It
is the toolkit's shop window: picking a pack draws it twice at once — a
small live application, and under it the whole widget gallery, every
control the toolkit has in every state — so a theme can be judged without
launching anything.

## Run

```bash
go run ./cmd/uitksettings
go run ./cmd/uitksettings -stage win98              # open with Windows 98 staged
go run ./cmd/uitksettings -page appearance          # open on the options page
go run ./cmd/uitksettings -headless                 # settings.png in cwd
go run ./cmd/uitksettings -stage aqua -screenshot out/
tools/shots/demos.sh                               # docs/screenshots/settings.webp and the other demos
```

`-page` takes `themes` (the default), `appearance`, `packs` or `about`.

## Pages

### Themes — browse and judge

The **browser** on the left: a **search field** (a pack is found by its
id, name, year, family, engine or what its summary says — every word has
to match; Return stages the first hit, Escape empties the field), the
**decade filter** (*All decades*, one decade, or *My themes*), the packs
that pass both listed by year (`1995 · Windows 95`), a count of what is
showing, and under the list the **details** the rows have no room for:
the pack's name, its year, family and engine, its one-line summary, and
what following the desktop does to it. The details box keeps its height,
so the list does not resize as you arrow down the packs; a long summary
scrolls inside it.

On the right, the staged pack drawn twice in a splitter you can size:

- the **preview** — a small but fully interactive application window
  (menu bar, tool bar, tabs with every control, tree, table, dialogs,
  status bar) painted entirely in the staged theme, frame, caption and
  all;
- the **gallery** — the `showcase` package, the same one the tour shows
  over its Controls, Views and Documents pages, here as one
  scrolling column: tool bar, the Buttons and Fields panels, then the
  ScrollView, ListView, TreeView, TableView and Form panels, and a status
  bar. Its Window, Theme and Quit controls are disabled here: the gallery
  is previewing a pack, not running as an app of its own.

Both are `widgets.ThemeScope`s, so Settings itself keeps the applied look.
Staging a pack switches them where they stand — the caret stays in the
search field, the focus on the list, the gallery where you scrolled it.

The browser is about 240 logical pixels wide whatever the window and
display scale are, so its rows stay readable on a small window, and it
keeps that share while the window is resized (`Window.OnResize`).
Dragging either sash replaces the split: a dragged split stays through
every resize after, and both survive Apply, Revert and Defaults.

The preview takes 62% of the right-hand splitter, enough for its Controls
tab — four rows a side — to show every row in every pack with
desktop-sized controls. Material's, shadcn's and Geist's 40- to 48-pixel
touch targets are the exception: those ten packs clip the tab's last row
until the sash is dragged down.

### Appearance — what every app does with the theme

Everything `look.json` carries besides the pack, each with the line that
says what it does, and a strip of the staged theme beside them so corners,
icons and their size are seen changing:

- **Shape and icons** — **Corners** (Theme shape / Round / Square),
  **Icon size** (16 / 24 / 32), **Icons** (the chrome set), **Animations**
  (hover fades, the default button's pulse, busy bars).
- **The desktop** — **Follow the desktop's colours**, its light or dark
  mode and its accent (see below), and **Use the desktop's file dialogs**.
- **Windows** — **Use system title bar and borders** and **Place window
  buttons as the theme does**. A window that is shaped, transparent or
  glass behind belongs in this section.

### Packs & icons

User theme packs (export the staged theme, delete user packs) and icon
sets (built-in and user, delete user sets).

### About

Versions, the engine list, and the files Settings reads and writes.

## Staged, applied, reverted

The theme, the gallery and the preview show what is **staged**. Nothing is
written until **Apply**, which saves `look.json` and switches Settings and
every app that watches the file. **Revert** drops the staged change;
**Defaults** stages what a fresh install has (it does not write either).
The three sit pinned under the pages with the line that says which of the
two states the screen is in — *Applied — every uitoolkit app is using this
look* or *Staged, not applied* — and the status bar leads with
`applied` or `unapplied`. Closing without Apply discards the staged change.

## Screenshot geometry

The Theme Atlas crops the preview panel out of `uitksettings -stage ID
-screenshot out.png`. In the default 1024×860 window at scale 1, with the
default look applied, the panel is the same rectangle in every pack:

```
x 445, y 58, 569 × 429        crop box (445, 58) – (1014, 487)
```

`TestSettingsPreviewPanelKeepsItsPlace` pins those numbers; a layout
change that moves them fails it, and the new ones belong here. (They were
`x 504, y 175, 510 × 387` before the gallery went under the preview, and
`569 × 381` until the preview took 62% of the split instead of 55%, so
its Controls tab shows every row.)
The applied look sets Settings' own metrics, so take atlas shots with a
clean `XDG_CONFIG_HOME`.

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
  "nativeDialogs": false,
  "decorations": "system",
  "captionButtons": "theme"
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
ones. `decorations` is who draws the frame of a window with its own title
bar: `system` the desktop (**Use system title bar and borders**),
`toolkit` uitoolkit for every window, left out for the default.
`captionButtons` `theme` puts the caption buttons of a frame uitoolkit
draws where the theme's era put them (the Mac's traffic lights on the
left; **Place window buttons as the theme does**); left out, they follow
the desktop's button layout. See [decorations.md](decorations.md).

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

Mode `0600`. Missing or invalid files yield the default theme,
`metal-ocean` (Metal, Ocean theme), in its own corners (`theme`), with
`classic` icons at `medium` size. A file without `iconSize` migrates to
medium.

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
so existing `look.json` files keep working. Until the user picks one,
apps show the default theme, Metal (Ocean) (`style.DefaultThemeName`).

Settings lists them in the theme browser, a row each, by year (`1995 ·
Windows 95`); user exports show as `User · <name>` and again under
**User** on Packs & icons.

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

The Appearance live preview includes a toolbar strip of chrome plus the
mail stems (Fetch / Write) for the selected set.

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
it: theme, corners, icons, motion, the file dialogs, who draws the frame,
where the caption buttons go and following the desktop.
`Application.Appearance()` is its other half — the whole appearance the
app runs in — so an app that switches packs starts from it:

```go
ap := app.Appearance()
ap.Name = "win95"
app.ApplyAppearance(ap) // the frame, the caption buttons and motion stay
```

`style.LookAppearance(look)` reads back only what a look carries (pack,
palette, corners, icons); the preferences that are the application's come
back at their defaults, so an appearance rebuilt from it and applied would
reset them. `Application.Decorations()` and `CaptionButtons()` read the
two frame preferences on their own.

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

The gallery, Files, Notes and Inspector start from `PreferredLook` and
watch the file, and so should any application: an app's own chrome
preferences (a density, a layout) belong in an app file of its own and must
**not** rewrite `look.json`. The rule the mail client established is that
the only write back to `look.json` is a deliberate View → Dark / Light —
`SaveAppearance` of `WithPalette`, same corners and icons, opposite palette
starter. Gallery / screenshot fixtures keep explicit `DarkLook` /
`LightLook`.

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
| `Application.ApplyAppearance`, `Application.Appearance`, `Application.Decorations`, `Application.OnLookChange`, `Application.DesktopColorScheme`, `ColorSchemeEnv` | `app` |
| `ColorScheme`, `SchemeVariant`, `Appearance.Effective`, `SetDesktopColorScheme`, `DesktopReducesMotion` | `style` / `uitoolkit` |
| `AccentEngine`, `SetDesktopAccent`, `DesktopAccent`, `TakesAccent`, `CloneTokenMaps` | `style` |
| `DesktopPrefs`, `ReadDesktopPrefs`, `WatchDesktopPrefs` (the portal) | `platform` |
| `DrawToolIcon`, `DrawFileToolIcon` | `style` |
