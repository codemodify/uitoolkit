# Settings

`cmd/uitksettings` is the toolkit appearance editor. It does not invent a
second theme system: it writes the same LookAndFeel knobs other apps apply
with `Application.SetLook`.

## Run

```bash
go run ./cmd/uitksettings
go run ./cmd/uitksettings -headless                 # settings.png in cwd
go run ./cmd/uitksettings -screenshot docs/screenshots
```

Linux X11 or Wayland (same backends as gallery / Mail). Headless uses the
offscreen surface.

There is no menu bar. **Apply** is the only persist action.

## What it sets

| Control | LookAndFeel | Values |
| --- | --- | --- |
| Theme | `Classic` palette / `Name()` | `dark` (Dark graphite), `light` (Light paper) |
| Corners | `Metrics.Radius` / `RadiusSmall` | `round` (8 / 5), `square` (0 / 0) |
| Icon set | `DrawToolIcon` glyph set | `classic` (rounded stroke), `sharp` (geometric) |

Radios update a **staged** appearance and preview it in the Settings
window (`Application.SetLook`) without touching disk. **Apply** writes
`$XDG_CONFIG_HOME/uitoolkit/look.json` (atomic rename) so other running
apps can reload. Closing the window without Apply **discards** staged
changes; the last applied file stays as-is.

## Prefs file

```
$XDG_CONFIG_HOME/uitoolkit/look.json
# fallback: ~/.config/uitoolkit/look.json
```

```json
{
  "theme": "dark",
  "corners": "round",
  "icons": "classic"
}
```

Mode `0600`. Missing or invalid files yield Dark + round + classic.

## Live reload in other apps

Source of truth is `look.json`. After Apply, every `Application` that
watches the file stats it (poll, ~300 ms while idle; every `PumpOnce`)
and, on change, reloads prefs and calls

```go
app.SetLook(style.WithAppearance(app.Look(), style.LoadAppearance()))
```

so theme, corners, and icons update without a restart. Display scale and
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

Opt out (Settings does this so radios stay staged until Apply):

```go
app := uitoolkit.New(uitoolkit.Options{DisableLookWatch: true})
```

`DarkLook` / `LightLook` fixtures stay static unless `WatchLook` is set,
so gallery screenshot pixels stay deterministic.

Mail, gallery, Files, Notes, and Inspector start from `PreferredLook`
and watch the file. Mail’s View → Dark / Light still toggles the theme
and writes the same `look.json` (its own `mailui.json` light flag stays
in sync). Gallery / screenshot fixtures keep explicit `DarkLook` /
`LightLook`.

Helpers for a live look:

```go
uitoolkit.WithTheme(look, uitoolkit.ThemeLight)
uitoolkit.WithCorners(look, uitoolkit.CornersSquare)
uitoolkit.WithIcons(look, uitoolkit.IconSetSharp)
uitoolkit.WithAppearance(look, uitoolkit.Appearance{...})
```

## API

| Symbol | Package |
| --- | --- |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance` | `style` / `uitoolkit` |
| `Options.WatchLook`, `Options.DisableLookWatch` | `app` / `uitoolkit` |
| `Application.WatchingLook`, `Application.ReloadPreferredLook` | `app` |
| `DrawToolIcon` | `style` |
