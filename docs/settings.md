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

## What it sets

| Control | LookAndFeel | Values |
| --- | --- | --- |
| Theme | `Classic` palette / `Name()` | `dark` (Dark graphite), `light` (Light paper) |
| Corners | `Metrics.Radius` / `RadiusSmall` | `round` (8 / 5), `square` (0 / 0) |
| Icon set | `DrawToolIcon` glyph set | `classic` (rounded stroke), `sharp` (geometric) |

Changing a radio or View menu item applies immediately (preview pane plus
the settings window itself) and writes the prefs file. There is no
separate Apply / OK.

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

## Other apps

```go
look := uitoolkit.PreferredLook()
app := uitoolkit.New(uitoolkit.Options{Look: look})
```

`PreferredLook` is `LoadAppearance().Look()`. Helpers for a live look:

```go
uitoolkit.WithTheme(look, uitoolkit.ThemeLight)
uitoolkit.WithCorners(look, uitoolkit.CornersSquare)
uitoolkit.WithIcons(look, uitoolkit.IconSetSharp)
uitoolkit.WithAppearance(look, uitoolkit.Appearance{...})
```

`Application.New` uses `PreferredLook()` when `Options.Look` is nil, then
applies display scale as before.

Mail, gallery, Files, Notes, and Inspector start from `PreferredLook`
so corners and icon set match Settings. Mail’s View → Dark / Light still
toggles the theme and writes the same `look.json` (its own `mailui.json`
light flag stays in sync). Gallery / screenshot fixtures keep explicit
`DarkLook` / `LightLook` so CI pixels stay deterministic.

## API

| Symbol | Package |
| --- | --- |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance` | `style` / `uitoolkit` |
| `DrawToolIcon` | `style` |
