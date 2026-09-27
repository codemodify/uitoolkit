# Bundled OFL fonts

uitoolkit embeds these typefaces and **rasterizes them at runtime**
(OpenType outlines → paintengine2d glyph atlas). They are not decoration.

| File | Family | Role |
| --- | --- | --- |
| `TitilliumWeb-Regular.ttf` | Titillium Web | Default UI (labels, buttons, menus, fields) |
| `TitilliumWeb-Bold.ttf` | Titillium Web | Titles |
| `JetBrainsMono-Regular.ttf` | JetBrains Mono | Mono role: code, Inspector, logs, mono fields |
| `JetBrainsMono-Bold.ttf` | JetBrains Mono | Mono emphasis |

mononoki is **not** the default mono face (optional later theme only).

These two are what a look **falls back to**, not what it is stuck with.
Each theme pack names the typefaces of its era first and reads in the
first one installed (`style.FontPrefs`, found through fontconfig's
`fc-list`), and Settings has a chooser for each role — the list leads
with these two, then the families the machine has — whose choice goes in
front of the pack's list and persists in `look.json` as `fontUI` /
`fontMono`. Nothing else is bundled: a family chosen there is read from
where it is already installed. `UITK_SYSTEM_FONTS=0` switches installed
fonts off entirely and leaves these two, which is what makes a
screenshot reproducible. See
[docs/settings.md](../docs/settings.md#two-typefaces-not-one).

Licenses: [OFL-TitilliumWeb.txt](OFL-TitilliumWeb.txt),
[OFL-JetBrainsMono.txt](OFL-JetBrainsMono.txt).

Titillium Web © Accademia di Belle Arti di Urbino (OFL).
JetBrains Mono © The JetBrains Mono Project Authors (OFL).
