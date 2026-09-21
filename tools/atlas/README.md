# Theme Atlas

The tools that draw the Theme Atlas — a page with every built-in pack's
preview, its full widget gallery and a timeline of when each look shipped —
and the theme sheets and timeline in the README.

    tools/atlas/render.sh   # every pack, headless: out/previews, out/gallery, out/packs.tsv
    tools/atlas/build.py    # out/index.html from out/packs.tsv and template.html
    tools/atlas/sheets.sh   # docs/screenshots/themes: the sheets and timeline.png

`out/` is scratch and gitignored. Everything renders headless on the CPU at
scale 1; nothing opens on the desktop. Needs ImageMagick (`magick`) and, for
the timeline, Chromium.

- **Previews** are the Settings window's preview panel, cropped at the
  geometry `docs/settings.md` records under "Screenshot geometry". If Settings
  moves the panel, change `CROP` in `render.sh` and `W_IMG`/`H_IMG` in
  `build.py`.
- **A new lineage** (a pack's platform) needs a lane in `build.py`'s `LANES`
  or `WEB_LANES`; `build.py` names any lineage it could not place.
- **Publishing** the page means uploading `out/index.html` with
  `out/previews/` and `out/gallery/` beside it.
