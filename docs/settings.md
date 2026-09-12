# Settings

`cmd/uitksettings` is the toolkit appearance editor. It does not invent a
second theme system: it selects a **named theme pack** and an **icon set**
on top of that pack. Other apps apply both with `Application.SetLook` /
`PreferredLook`.

Theme packs are GTK/KDE-like look files (palette + corners). Chrome
`ToolIcon`s come from a **PNG** file set under `~/.config/uitoolkit/icons/`
when one is selected. Packs may still list a default `icons` field
(`classic` / `sharp`); `look.json` `"icons"` overrides it.

## Run

```bash
go run ./cmd/uitksettings
go run ./cmd/uitksettings -headless                 # settings.png in cwd
go run ./cmd/uitksettings -screenshot docs/screenshots
```

Linux X11 or Wayland (same backends as gallery / Mail). Headless uses the
offscreen surface.

There is no menu bar. **Apply** is the only persist action for
`look.json`. **Export current look…** writes a user pack.

## What it sets

| Control | LookAndFeel | Values |
| --- | --- | --- |
| Theme picker | named pack → `Classic` | embedded starters + exported user packs |
| Icon picker | PNG set or drawn fallback | classic / sharp + `icons/<set>/` |
| Export | `themes/<name>/theme.json` | current (staged) look, asked for a name |

Corners stay inside the selected pack. Icons are chosen **on top of**
the pack.

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
  "theme": "dark-round-classic",
  "icons": "lucide"
}
```

`theme` is the pack name. `icons` is the chrome set (`classic`, `sharp`,
or an installed directory such as `lucide`). Corners are not stored
here.

Mode `0600`. Missing or invalid files yield `dark-round-classic`.

A legacy v0.11.0 / v0.11.1 file with `theme` / `corners` / `icons` is
migrated on load to the matching starter (`light` + `square` + `sharp`
→ `light-square-sharp`). A bare `"theme": "dark"` or `"light"` (no
corners/icons) becomes `dark-round-classic` or `light-round-classic`,
unless a user pack of that exact name exists.

## Theme packs

Eight **embedded starters** cover dark/light × round/square ×
classic/sharp. They ship in the binary (`style/themes/*/theme.json`) and
are **not** auto-written to disk.

| Name | Palette | Corners | Icons |
| --- | --- | --- | --- |
| `dark-round-classic` | dark | round | classic |
| `dark-round-sharp` | dark | round | sharp |
| `dark-square-classic` | dark | square | classic |
| `dark-square-sharp` | dark | square | sharp |
| `light-round-classic` | light | round | classic |
| `light-round-sharp` | light | round | sharp |
| `light-square-classic` | light | square | classic |
| `light-square-sharp` | light | square | sharp |

User packs:

```
$XDG_CONFIG_HOME/uitoolkit/themes/<name>/theme.json
# fallback: ~/.config/uitoolkit/themes/<name>/theme.json
```

```json
{
  "label": "ocean",
  "palette": "dark",
  "corners": "round",
  "icons": "classic"
}
```

JSON only. Edit the file to tweak palette or corners. The pack `icons`
field (`classic` / `sharp`) is a **fallback default** used only when
`look.json` has no `"icons"`. When Settings Apply writes `"icons"`, that
value owns chrome ToolIcons.

**Export current look…** asks for a name and writes this file from the
staged look.

`ListThemes` lists **user packs first** (sorted by name), then builtins
that are not shadowed. `LoadTheme(name)` **prefers the user pack** when
both exist — a user `dark-round-classic` overrides the embedded starter
of the same name and is listed once (as exported).

## Icon sets (PNG files)

Chrome icons are **not** embedded. Ship-in-repo sets live at
`icons/lucide`, `icons/phosphor`, `icons/tabler`, `icons/heroicons`,
and `icons/material-symbols` (24×24 + `name@2x.png` 48×48, one stem
per action). Copy them yourself:

```bash
mkdir -p ~/.config/uitoolkit/icons
cp -R icons/lucide icons/phosphor icons/tabler icons/heroicons icons/material-symbols ~/.config/uitoolkit/icons/
```

See [icons/README.md](../icons/README.md) for licenses, attribution,
and the `@2x` convention. Settings lists `classic` / `sharp` (drawn)
plus every `icons/<set>/` directory that contains at least one
ToolIcon PNG. A missing file falls back to the drawn classic glyph.
The toolkit tints monochrome/alpha PNGs with the Look foreground /
icon color.

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

so the selected pack updates without a restart. Display scale and
density on the current look are kept.

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
pack. View → Dark / Light is the only Mail write: `SaveAppearance` of
`WithPalette` (same corners/icons, opposite palette). Gallery /
screenshot fixtures keep explicit `DarkLook` / `LightLook`.

Helpers for a live look:

```go
uitoolkit.LoadTheme("light-square-sharp")
uitoolkit.ListThemes()
uitoolkit.WithTheme(look, uitoolkit.ThemeLight)
uitoolkit.WithCorners(look, uitoolkit.CornersSquare)
uitoolkit.WithIcons(look, uitoolkit.IconSetSharp)
uitoolkit.WithAppearance(look, uitoolkit.Appearance{Name: "ocean"})
```

`PreferredLook` is `LoadAppearance().Look()`. `LoadAppearance` reads the
pack name and optional `"icons"` from `look.json`, resolves the pack
with `LoadTheme` (user pack, then builtin), and overlays the icon set.

## API

| Symbol | Package |
| --- | --- |
| `ThemePack`, `ListThemes`, `LoadTheme`, `ExportTheme` | `style` / `uitoolkit` |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName` | `style` / `uitoolkit` |
| `IconSetInfo`, `ListIconSets`, `IconsDir` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `ThemesDir`, `StarterName`, `DefaultThemeName` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance` | `style` / `uitoolkit` |
| `Options.WatchLook`, `Options.DisableLookWatch` | `app` / `uitoolkit` |
| `Application.WatchingLook`, `Application.ReloadPreferredLook` | `app` |
| `DrawToolIcon`, `DrawFileToolIcon` | `style` |
