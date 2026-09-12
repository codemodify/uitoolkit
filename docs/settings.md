# Settings

`cmd/uitksettings` is the toolkit appearance editor. It does not invent a
second theme system: it selects a **named theme pack** and other apps
apply it with `Application.SetLook` / `PreferredLook`.

Theme packs are GTK/KDE-like look files (palette + corners + icons), not
full style plugins.

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
| Export | `themes/<name>/theme.json` | current (staged) look, asked for a name |

There is no independent Theme / Corners / Icons triad. Those knobs live
**inside** the selected pack.

The picker updates a **staged** appearance and previews it in the
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
  "theme": "dark-round-classic"
}
```

`theme` is the **pack name only**. Corners and icons are not stored here.

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

JSON only. Edit the file to tweak palette, corners, or icons. **Export
current look…** asks for a name and writes this file from the staged
look.

`ListThemes` lists **user packs first** (sorted by name), then builtins
that are not shadowed. `LoadTheme(name)` **prefers the user pack** when
both exist — a user `dark-round-classic` overrides the embedded starter
of the same name and is listed once (as exported).

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
and watch the file. Mail’s View → Dark / Light switches to the matching
starter (same corners/icons, opposite palette) and writes that pack name
to `look.json` (its own `mailui.json` light flag stays in sync). Gallery
/ screenshot fixtures keep explicit `DarkLook` / `LightLook`.

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
pack name from `look.json` and resolves it with `LoadTheme` (user pack,
then builtin).

## API

| Symbol | Package |
| --- | --- |
| `ThemePack`, `ListThemes`, `LoadTheme`, `ExportTheme` | `style` / `uitoolkit` |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `ThemesDir`, `StarterName`, `DefaultThemeName` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance` | `style` / `uitoolkit` |
| `Options.WatchLook`, `Options.DisableLookWatch` | `app` / `uitoolkit` |
| `Application.WatchingLook`, `Application.ReloadPreferredLook` | `app` |
| `DrawToolIcon` | `style` |
