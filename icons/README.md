# uitoolkit icon sets

Three **24×24** SVG families for chrome `ToolIcon` actions (Mail, Settings,
gallery, Files, Notes). They are **not** embedded and are **not** copied
on install. Copy a set into your config directory, then pick it in
Settings.

## Install (manual)

```bash
mkdir -p ~/.config/uitoolkit/icons
cp -R filled outline duotone ~/.config/uitoolkit/icons/
```

If `XDG_CONFIG_HOME` is set:

```bash
mkdir -p "$XDG_CONFIG_HOME/uitoolkit/icons"
cp -R filled outline duotone "$XDG_CONFIG_HOME/uitoolkit/icons/"
```

Result:

```
~/.config/uitoolkit/icons/filled/open.svg
~/.config/uitoolkit/icons/outline/open.svg
~/.config/uitoolkit/icons/duotone/open.svg
```

Settings lists every subdirectory of `icons/` that contains at least one
action SVG. **Apply** writes `look.json`:

```json
{
  "theme": "dark-round-classic",
  "icons": "outline"
}
```

Theme packs may still have an `icons` field (`classic` / `sharp`). That
value is a fallback default only. When `look.json` has `"icons"`, it
owns chrome icons.

## Sets

| Directory | Treatment |
| --- | --- |
| `filled` | Solid glyphs, even-odd cutouts for marks |
| `outline` | 1.75px round-cap strokes |
| `duotone` | Body at 40% opacity, marks at 100% — both `currentColor` |

Each file is a single tintable SVG (`currentColor` / replaceable fill).
The toolkit paints with the Look icon / foreground color.

## File names

One SVG per toolkit action / `style.Icon*` id:

`new` `open` `save` `cut` `copy` `paste` `undo` `redo` `search`
`info` `warning` `error` `question`

A missing file falls back to the drawn **classic** set. `classic` and
`sharp` remain built-in vector fallbacks and do not need files.

## Custom sets

Create `~/.config/uitoolkit/icons/<name>/` with the same filenames
(24×24 `viewBox`, `currentColor`). Name must be
`[a-z][a-z0-9_-]{0,63}`.
