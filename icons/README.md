# uitoolkit icon sets

Five **premiere open-license PNG** families for chrome `ToolIcon` actions
and a **wide** extra stem vocabulary (Mail, Settings, gallery, Files,
Notes, future apps). They are **not** embedded and are **not** copied on
install. Copy a set into your config directory, then pick it in Settings.

The old hand-drawn `filled` / `outline` / `duotone` SVG folders are
**gone**. Use these packs. Glyphs are official upstream SVGs rendered by
`icons/render.sh` — not hand-drawn.

## Install (manual)

After clone or pull, copy again. New stems (and refreshed PNGs) only
appear in Settings once the folders under `~/.config/uitoolkit/icons/`
are updated:

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
~/.config/uitoolkit/icons/lucide/pen.png
~/.config/uitoolkit/icons/lucide/download.png
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
`icons/<name>/` folder). The Appearance page shows a live preview strip
of chrome + Mail Fetch / Write stems for the selected set.

## Sets

| Directory | Upstream | License | Style | Pin |
| --- | --- | --- | --- | --- |
| `lucide` | [Lucide](https://lucide.dev) | ISC | 24px outline | 1.45.0 |
| `phosphor` | [Phosphor](https://phosphoricons.com) | MIT | regular weight | v2.0.8 |
| `tabler` | [Tabler Icons](https://tabler.io/icons) | MIT | outline | v3.46.0 |
| `heroicons` | [Heroicons](https://heroicons.com) | MIT | 24px outline | v2.2.0 |
| `material-symbols` | [Material Symbols](https://fonts.google.com/icons) | Apache-2.0 | outlined 400 | marella v0.47.2 |

Each folder keeps the upstream `LICENSE`, a `SOURCES.txt` glyph map, and
crisp PNGs rendered from official SVGs with `rsvg-convert`
(`icons/render.sh`). All five may be redistributed with this tree.

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

`style.ToolIcon` ids (always present):

`new` `open` `save` `cut` `copy` `paste` `undo` `redo` `search`
`info` `warning` `error` `question` `mail` `download` `pen`

Wide coverage (every shipped set, see `STEMS.txt`):

- chrome: `filter` `settings` `preferences` `quit` `close` `more`
  `chevron-up` `chevron-down` `chevron-left` `chevron-right`
  `arrow-up` `arrow-down` `arrow-left` `arrow-right`
- mail: `mail-open` `inbox` `send` `reply` `reply-all` `forward`
  `attach` `upload` `trash` `archive` `junk` `tag` `star` `flag`
  `pencil` `compose` `fetch` `sync` `folder` `folder-plus` `user` `users`
  `bell` `bell-off` `eye` `eye-off` `lock` `unlock`
- UI: `check` `x` `plus` `minus` `edit` `trash-2` `help` `home`
  `calendar` `clock` `link` `external-link` `list` `layout` `columns`
  `rows` `table` `cards` `sun` `moon`

`pen.png` and `download.png` are required in every set (Mail **Write** /
**Fetch**). Aliases (`pencil` `compose` for pen; `fetch` for download)
ship as their own files. Every set also ships `no-icon` / `no-icon@2x`
(dashed box + X) for a missing stem.

When a premiere or user set is selected and one stem is missing, the
loader logs once and paints **`no-icon`** (dashed box + X) from that
folder — or the copy embedded in the toolkit if the directory is
absent. It does **not** fall through to a drawn classic scribble.
`classic` and `sharp` remain explicit vector sets only when those names
are chosen in Settings.

Glyph → upstream name for each pack is in that folder’s `SOURCES.txt`.

## Rebuild

```bash
# librsvg2-bin + python3 + curl + tar + unzip
./icons/render.sh
```

Archives are pinned in the script and cached under
`$TMPDIR/uitoolkit-icon-src`. Commit the generated PNGs.

## Custom sets

Create `~/.config/uitoolkit/icons/<name>/` with the same filenames
(24×24 and optional `@2x` 48×48 PNG, black- or white-on-transparent).
Name must be `[a-z][a-z0-9_-]{0,63}`.

## Attribution

These icons remain under their upstream licenses. Copying a pack into
config keeps `LICENSE` + `SOURCES.txt` with it. uitoolkit itself is under
The Free License (see the repository's `LICENSE`).
