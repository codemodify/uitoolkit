# uitoolkit icon sets

Five **premiere open-license PNG** families for chrome `ToolIcon` actions
(Mail, Settings, gallery, Files, Notes). They are **not** embedded and
are **not** copied on install. Copy a set into your config directory,
then pick it in Settings.

The old hand-drawn `filled` / `outline` / `duotone` SVG folders are
**gone**. Use these packs.

## Install (manual)

```bash
mkdir -p ~/.config/uitoolkit/icons
cp -R lucide phosphor tabler heroicons material-symbols ~/.config/uitoolkit/icons/
```

If `XDG_CONFIG_HOME` is set:

```bash
mkdir -p "$XDG_CONFIG_HOME/uitoolkit/icons"
cp -R lucide phosphor tabler heroicons material-symbols "$XDG_CONFIG_HOME/uitoolkit/icons/"
```

Result:

```
~/.config/uitoolkit/icons/lucide/open.png
~/.config/uitoolkit/icons/lucide/open@2x.png
```

Settings lists every subdirectory of `icons/` that contains at least one
action PNG. **Apply** writes `look.json`:

```json
{
  "theme": "dark",
  "corners": "round",
  "icons": "lucide"
}
```

Theme packs are palette only. `look.json` stores theme, corners, and
icons independently.

Settings lists **Built-in** (drawn classic/sharp, plus the five premiere
names when those folders are present) and **User** (any other
`icons/<name>/` folder).

## Sets

| Directory | Upstream | License | Style |
| --- | --- | --- | --- |
| `lucide` | [Lucide](https://lucide.dev) | ISC | 24px outline |
| `phosphor` | [Phosphor](https://phosphoricons.com) | MIT | regular weight |
| `tabler` | [Tabler Icons](https://tabler.io/icons) | MIT | outline |
| `heroicons` | [Heroicons](https://heroicons.com) | MIT | 24px outline |
| `material-symbols` | [Material Symbols](https://fonts.google.com/icons) | Apache-2.0 | outlined 400 |

Each folder keeps the upstream `LICENSE`, a `SOURCES.txt` glyph map, and
crisp PNGs rendered from official SVGs with `rsvg-convert` (`icons/render.sh`).

## Sizes

| File | Pixels |
| --- | --- |
| `name.png` | 24×24 (1×) |
| `name@2x.png` | 48×48 (HiDPI) |

The toolkit picks `@2x` when the destination is ≥36 px (2× toolbar,
message icons), otherwise `name.png`. If only one size is present, that
file is used.

## Tint

PNGs are monochrome/alpha. The toolkit treats them as a coverage mask
and paints with the Look icon / foreground color (same intent as SVG
`currentColor`).

## File names

One stem per toolkit action / `style.Icon*` id:

`new` `open` `save` `cut` `copy` `paste` `undo` `redo` `search`
`info` `warning` `error` `question` `mail` `download` `pen`

Extra mail-themed files (not ToolIcon ids): `inbox` `mail-open`.

A missing file falls back to the drawn **classic** set. `classic` and
`sharp` remain built-in vector fallbacks and do not need files.

Glyph → upstream name for each pack is in that folder’s `SOURCES.txt`.

## Custom sets

Create `~/.config/uitoolkit/icons/<name>/` with the same filenames
(24×24 and optional `@2x` 48×48 PNG, black- or white-on-transparent).
Name must be `[a-z][a-z0-9_-]{0,63}`.

## Attribution

These icons remain under their upstream licenses. Copying a pack into
config keeps `LICENSE` + `SOURCES.txt` with it. uitoolkit itself is MIT.
